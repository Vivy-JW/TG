package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/dict/dbsync"
	"tt/internal/host"
)

// dbSyncJob 一次(或最近一次)数据同步任务的状态;前端据它画进度。
type dbSyncJob struct {
	Running    bool   `json:"running"`
	Env        string `json:"env"`
	Phase      string `json:"phase"` // open|table|index|replace|done|error
	Message    string `json:"message"`
	Table      string `json:"table"`
	TableIndex int    `json:"tableIndex"` // 当前第几张表(1-based)
	TableTotal int    `json:"tableTotal"`
	TableRows  int    `json:"tableRows"` // 当前表已写入行数
	TotalRows  int    `json:"totalRows"` // 累计行数
	Tables     int    `json:"tables"`    // 完成的表数
	Elapsed    string `json:"elapsed"`
	Target     string `json:"target"` // 目标 SQLite
	Backup     string `json:"backup,omitempty"`
	Warning    string `json:"warning,omitempty"`
	Error      string `json:"error,omitempty"`
	Done       bool   `json:"done"`
	StartedAt  string `json:"startedAt,omitempty"`
}

// dbSyncEnv 可用于同步的环境(挂了数据库的环境)。
type dbSyncEnv struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Address  string `json:"address"`
	DBPath   string `json:"dbPath"`
	Exists   bool   `json:"exists"`
	Size     int64  `json:"size"`
	Modified string `json:"modified,omitempty"`
}

type dbSyncResp struct {
	Dir  string      `json:"dir"` // 本地字典库目录（每个环境一个 .db）
	Envs []dbSyncEnv `json:"envs"`
	Job  dbSyncJob   `json:"job"`
}

// isGPEnv 判断环境是否属于 TOP GP。T100 的字典/镜像接口必须拒绝 GP：
// 两者共用 hosts.sshs，但 GP 的同步/镜像语义与落点完全不同（GP 走 /api/gp/*）。
func isGPEnv(e *config.NamedSsh) bool {
	return e != nil && strings.EqualFold(strings.TrimSpace(e.Product), "gp")
}

// dbSyncView 汇总同步页面需要的状态（库目录/各环境本地库/任务）。
func (s *Server) dbSyncView() dbSyncResp {
	dir := s.localDBDir()
	resp := dbSyncResp{Dir: dir, Envs: []dbSyncEnv{}}
	if hosts, err := config.LoadHosts(s.cfgPath); err == nil {
		for i := range hosts.SSHs {
			e := &hosts.SSHs[i]
			if isGPEnv(e) {
				continue // GP 走 /api/gp/sync，不能出现在 T100 的同步清单里
			}
			if e.DB == nil {
				continue // 未挂库的环境无法同步
			}
			p := s.dbPathFor(e.Name)
			row := dbSyncEnv{Name: e.Name, Type: e.DB.Type, Address: e.DB.Address(), DBPath: p}
			if st, err := os.Stat(p); err == nil {
				row.Exists = true
				row.Size = st.Size()
				row.Modified = st.ModTime().Format("2006-01-02 15:04")
			}
			resp.Envs = append(resp.Envs, row)
		}
	}
	resp.Job = s.syncSnapshot()
	return resp
}

// hDBSyncGet 返回同步目标、可用环境与当前/最近一次任务(前端轮询用)。
func (s *Server) hDBSyncGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, s.dbSyncView())
}

// hDBSyncPost 启动一次同步(单实例:已有任务在跑返回 409);返回后前端轮询 /api/dbsync。
func (s *Server) hDBSyncPost(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Env string `json:"env"`
	}
	if !readBody(w, r, &req) {
		return
	}
	envName := strings.TrimSpace(req.Env)
	if envName == "" {
		fail(w, 400, fmt.Errorf("请选择环境"))
		return
	}
	hosts, err := config.LoadHosts(s.cfgPath)
	if err != nil {
		fail(w, 500, err)
		return
	}
	e := hosts.ByName(envName)
	if e == nil {
		fail(w, 400, fmt.Errorf("未找到环境 %q", envName))
		return
	}
	if isGPEnv(e) {
		fail(w, 400, fmt.Errorf("「%s」是 TOP GP 环境，请走 TOP GP 页的字典同步，不能使用 T100 的 /api/dbsync", e.Name))
		return
	}
	if e.DB == nil {
		fail(w, 400, fmt.Errorf("环境 %q 未配置数据库", envName))
		return
	}

	target := s.dbPathFor(e.Name)
	s.syncMu.Lock()
	if s.sync.Running {
		s.syncMu.Unlock()
		fail(w, 409, fmt.Errorf("已有同步任务进行中"))
		return
	}
	s.sync = dbSyncJob{
		Running:   true,
		Env:       e.Name,
		Phase:     "open",
		Message:   "准备中…",
		Target:    target,
		StartedAt: time.Now().Format(time.RFC3339),
	}
	s.syncStart = time.Now()
	// 可取消的 context(理由同镜像:任务有意跑在请求之外)
	ctx, cancel := context.WithCancel(context.Background())
	s.syncCancel = cancel
	s.syncMu.Unlock()

	go s.runDBSync(ctx, e, target)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// hDBSyncStop 中途取消正在跑的同步。没有任务在跑时返回 ok=false(不算错误)。
func (s *Server) hDBSyncStop(w http.ResponseWriter, r *http.Request) {
	s.syncMu.Lock()
	cancel := s.syncCancel
	running := s.sync.Running
	if running && cancel != nil {
		s.sync.Message = "正在停止…"
	}
	s.syncMu.Unlock()

	if !running || cancel == nil {
		writeJSON(w, 200, map[string]any{"ok": false, "error": "没有正在进行的同步任务"})
		return
	}
	cancel()
	writeJSON(w, 200, map[string]any{"ok": true})
}

// runDBSync 后台执行同步,把 dbsync 的进度回调同步到任务状态。
func (s *Server) runDBSync(ctx context.Context, e *host.NamedSsh, target string) {
	onProgress := func(p dbsync.Progress) {
		s.syncMu.Lock()
		if p.Phase != "" {
			s.sync.Phase = p.Phase
		}
		if p.Message != "" {
			s.sync.Message = p.Message
		}
		if p.Table != "" {
			s.sync.Table = p.Table
		}
		if p.TableIndex > 0 {
			s.sync.TableIndex = p.TableIndex
		}
		if p.TableTotal > 0 {
			s.sync.TableTotal = p.TableTotal
		}
		s.sync.TableRows = p.TableRows
		if p.TotalRows > 0 {
			s.sync.TotalRows = p.TotalRows
		}
		s.sync.Elapsed = time.Since(s.syncStart).Round(time.Second).String()
		s.syncMu.Unlock()
	}

	st, err := dbsync.Run(ctx, *e.DB, target, nil, onProgress)

	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	s.sync.Running = false
	s.sync.Done = true
	s.syncCancel = nil // 收尾:停止端点据此回"没有正在进行的任务"
	s.sync.Elapsed = time.Since(s.syncStart).Round(100 * time.Millisecond).String()
	if err != nil {
		// 用户取消 ≠ 失败(与镜像同一套口径)
		if errors.Is(err, context.Canceled) {
			s.sync.Phase = "cancelled"
			s.sync.Message = "已停止;本地库未被替换,原库保持原样"
			return
		}
		s.sync.Phase = "error"
		s.sync.Error = err.Error()
		return
	}
	s.sync.Phase = "done"
	s.sync.Tables = st.Tables
	s.sync.TotalRows = st.Rows
	s.sync.Backup = st.Backup
	if len(st.Warnings) > 0 {
		s.sync.Warning = strings.Join(st.Warnings, "\n")
	}
	s.sync.Message = fmt.Sprintf("同步完成:%d 张表,共 %d 行", st.Tables, st.Rows)
}

// syncSnapshot 取当前(或最近一次)同步状态副本。
func (s *Server) syncSnapshot() dbSyncJob {
	s.syncMu.Lock()
	defer s.syncMu.Unlock()
	return s.sync
}
