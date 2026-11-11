package dict

// 字典同步的“跑一次”入口：Web 的后台任务与 CLI 的命令都落到这里。
//
// 为什么放在 internal/dict 而不是 web 面：同步的输入是一个 GP 环境、输出是一个
// 本地 SQLite，与 HTTP 无关；CLI（tt dict db sync）也要用它，
// 而 CLI 不该为了同步去 import internal/web。
//
// 这里只做三件事：校验连接配置、按固定据点（ds）跑 dbsync.Run、把进度回调透出去。
// 落点（DictPath）、据点口径（SyncCompany）、投影 SQL（dbsync.SyncGP）都在别处。

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/dict/dbsync"
)

// SyncEnv 同步一个 GP 环境的字典，阻塞到完成。
//
// onProgress 逐条收进度文案，onCount 收“已完成/总数”（两者都可为 nil）。
// 返回的 Stats 是本次写入的规模；失败时是零值 + error。
func SyncEnv(ctx context.Context, env config.NamedSsh, onProgress func(string), onCount func(done, total int)) (dbsync.Stats, error) {
	var zero dbsync.Stats
	if env.DB == nil || env.DB.Host == "" {
		return zero, fmt.Errorf("这个环境没配数据库")
	}
	var account, password string
	for _, a := range env.DB.Accounts {
		account, password = a.Account, a.Password
		if account != "" {
			break
		}
	}
	if account == "" {
		return zero, fmt.Errorf("这个环境的数据库没有账号")
	}

	// 本地字典库**固定按 ds 取**（用户 2026-11-10 定调）：ds 是系统示范据点，
	// 表档最全（2,962 张），拿来当"查得到 GP 有哪些表"的底库。这是常量，
	// 不看配置里的营运据点 —— 营运据点只决定**直查**连哪个 schema。
	lib := SyncCompany
	if onProgress != nil {
		onProgress("连接 " + env.DB.Host + ":" + strconv.Itoa(env.DB.Port) + "/" + env.DB.Service + " …")
	}

	st, err := dbsync.Run(ctx, dbsync.Options{
		Path:  DictPath(env),
		DSN:   dbsync.DSN(*env.DB, account, password),
		Owner: strings.ToUpper(lib),
		Lib:   lib,
		// SyncGP 要据点代号（拼 zta_file 的 WHERE）与账号（打进度文案），
		// 引擎只知道 DSN，所以在这里绑进去。
		Sync: func(ctx context.Context, src, dst *sql.DB, rep func(string), count func(done, total int), meta map[string]string) error {
			return dbsync.SyncGP(ctx, src, dst, lib, account, rep, count, meta)
		},
		OnCount: onCount,
	}, onProgress)
	if err != nil {
		if st != nil {
			return *st, err
		}
		return zero, err
	}
	return *st, nil
}

// SyncTimeout 是一次字典同步的时限：实测 1 分钟左右，留足余量但不至于把连接一直挂着。
const SyncTimeout = 10 * time.Minute
