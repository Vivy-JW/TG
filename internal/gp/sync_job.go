package gp

// GP 字典同步的入口：把 dbsync.Run 包成一个后台 job（Web 的 /api/gp/sync 用）。
//
// 真正的同步逻辑在 internal/dict/dbsync（Run 负责临时库与原子替换，SyncGP 负责
// 八张 t_* 表的投影 SQL）；路径口径（DictPath）与据点口径（SyncCompany）在
// internal/dict。这里只剩 job 状态机与进度缓冲。

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"tt/internal/config"
	dictlib "tt/internal/dict"
	"tt/internal/dict/dbsync"
)

// Sync 同步一个环境的字典，阻塞到完成；onProgress 逐条收到进度（可为 nil）。
//
// Web 侧走 /api/gp/sync 的后台任务，CLI 侧走 tt gp sync / tt dict db sync ——
// 两条路都落到同一个 <环境>.db（路径只有 dictlib.DictPath 一份实现），
// 跑的是同一个 dbsync.Run，所以"命令行同步完的库，界面里查得到"这件事
// 不靠约定，靠同一份代码。
func (s *Service) Sync(env config.NamedSsh, onProgress func(string)) error {
	if s.job(env.Name).State == "running" {
		return fmt.Errorf("环境 %s 的字典同步已经在跑", env.Name)
	}
	s.setJob(env.Name, func(j *Job) {
		*j = Job{State: "running", Kind: "sync", Started: time.Now()}
	})

	done := make(chan struct{})
	go func() {
		s.runSync(env)
		close(done)
	}()

	seen := 0
	flush := func() {
		j := s.job(env.Name)
		for seen < len(j.Progress) {
			if onProgress != nil {
				onProgress(j.Progress[seen])
			}
			seen++
		}
	}
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-done:
			flush()
			j := s.job(env.Name)
			switch j.State {
			case "done":
				return nil
			case "error":
				return fmt.Errorf("%s", j.Error)
			default:
				return fmt.Errorf("同步没有正常结束（state=%s）", j.State)
			}
		case <-tick.C:
			flush()
		}
	}
}

// runSync 跑一次同步，把结果写进 job。整段用 10 分钟超时：字典拉取实测 1 分钟左右，
// 留足余量但不至于把连接一直挂着。
func (s *Service) runSync(env config.NamedSsh) {
	finish := func(st *dbsync.Stats, err error) {
		s.setJob(env.Name, func(j *Job) {
			j.Ended = time.Now()
			if err != nil {
				j.State = "error"
				j.Error = err.Error()
				return
			}
			j.State = "done"
			j.Done = int64(st.Tables)
			j.Total = int64(st.Tables)
			j.Result = map[string]any{
				"path": st.Path, "tables": st.Tables, "columns": st.Columns,
				"indexes": st.Indexes, "constraints": st.Constraints,
				"sizeMB": st.SizeMB, "messages": st.Messages,
			}
		})
	}
	if env.DB == nil || env.DB.Host == "" {
		finish(nil, fmt.Errorf("这个环境没配数据库"))
		return
	}
	var account, password string
	for _, a := range env.DB.Accounts {
		account, password = a.Account, a.Password
		if account != "" {
			break
		}
	}
	if account == "" {
		finish(nil, fmt.Errorf("这个环境的数据库没有账号"))
		return
	}

	// 本地字典库**固定按 ds 取**（用户 2026-11-10 定调）：ds 是系统示范据点，
	// 表档最全（2,962 张），拿来当"查得到 GP 有哪些表"的底库。这是常量，
	// 不看配置里的营运据点 —— 营运据点只决定**直查**连哪个 schema。
	lib := dictlib.SyncCompany
	s.progress(env.Name, "连接 "+env.DB.Host+":"+fmt.Sprint(env.DB.Port)+"/"+env.DB.Service+" …")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	st, err := dbsync.Run(ctx, dbsync.Options{
		Path:  dictlib.DictPath(env),
		DSN:   dbsync.DSN(*env.DB, account, password),
		Owner: strings.ToUpper(lib),
		Lib:   lib,
		// SyncGP 要据点代号（拼 zta_file 的 WHERE）与账号（打进度文案），
		// 引擎只知道 DSN，所以在这里绑进去。
		Sync: func(ctx context.Context, src, dst *sql.DB, rep func(string), onCount func(done, total int), meta map[string]string) error {
			return dbsync.SyncGP(ctx, src, dst, lib, account, rep, onCount, meta)
		},
		OnCount: func(done, total int) {
			s.setJob(env.Name, func(j *Job) {
				j.Done = int64(done)
				j.Total = int64(total)
			})
		},
	}, func(msg string) {
		s.progress(env.Name, msg)
	})
	finish(st, err)
}
