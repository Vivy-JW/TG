package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/host"
)

// isGPEnv 判断一个环境是不是 TOP GP（config.hosts.sshs[].product == "gp"）。
//
// 字典侧这组镜像接口是 T100 的历史遗留：GP 的镜像走 /api/gp/mirror、字典走
// /api/gp/sync，遇到 GP 环境必须拒掉，否则同一个环境会有两套镜像目录。
func isGPEnv(e *config.NamedSsh) bool {
	return e != nil && strings.EqualFold(strings.TrimSpace(e.Product), "gp")
}

// mirrorJob 一次(或最近一次)源码镜像拉取的状态;前端据它画进度条。
type mirrorJob struct {
	Running   bool   `json:"running"`
	Env       string `json:"env"`
	Full      bool   `json:"full"`
	Phase     string `json:"phase"` // connect|probe|pack|download|done|error
	Message   string `json:"message"`
	Bytes     int64  `json:"bytes"`
	Total     int64  `json:"total"` // 下载阶段=归档总字节;0=未知(pack 阶段)
	Files     int    `json:"files"`
	Pruned    int    `json:"pruned"` // 本地清理掉的残留备份文件数
	Elapsed   string `json:"elapsed"`
	Error     string `json:"error,omitempty"`
	Note      string `json:"note,omitempty"`
	Done      bool   `json:"done"`
	StartedAt string `json:"startedAt,omitempty"`
}

// mirrorEnv 单个环境的镜像现状。
type mirrorEnv struct {
	Name  string `json:"name"`
	Zone  string `json:"zone"`
	Path  string `json:"path"`
	Ready bool   `json:"ready"` // 本地已有完整基线(增量前提)
}

type mirrorResp struct {
	MirrorDir string      `json:"mirrorDir"`
	Envs      []mirrorEnv `json:"envs"`
	Job       mirrorJob   `json:"job"`
}

// hMirrorGet 返回镜像根、各环境镜像现状与当前/最近一次拉取任务(前端轮询用)。
func (s *Server) hMirrorGet(w http.ResponseWriter, r *http.Request) {
	dir := ""
	if root, err := loadConfig(s.cfgPath); err == nil {
		dir = config.AbsPath(root.Mirror.Dir)
	}
	resp := mirrorResp{MirrorDir: dir, Envs: []mirrorEnv{}}
	if hosts, err := config.LoadHosts(s.cfgPath); err == nil {
		for i := range hosts.SSHs {
			e := &hosts.SSHs[i]
			if isGPEnv(e) {
				continue // GP 走 /api/gp/mirror，不能出现在 T100 的镜像清单里
			}
			envDir := e.MirrorPath(dir)
			resp.Envs = append(resp.Envs, mirrorEnv{
				Name:  e.Name,
				Zone:  e.Zone,
				Path:  envDir,
				Ready: host.MirrorReadyDir(envDir),
			})
		}
	}
	resp.Job = s.mirrorSnapshot()
	writeJSON(w, 200, resp)
}

// hMirrorPull 启动一次拉取(单实例:已有任务在跑返回 409);返回后前端轮询 /api/mirror。
func (s *Server) hMirrorPull(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Env  string `json:"env"`
		Full bool   `json:"full"`
	}
	if !readBody(w, r, &req) {
		return
	}
	envName := strings.TrimSpace(req.Env)
	if envName == "" {
		fail(w, 400, fmt.Errorf("请选择环境"))
		return
	}
	rootDir := ""
	if root, err := loadConfig(s.cfgPath); err == nil {
		rootDir = root.Mirror.Dir
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
		fail(w, 400, fmt.Errorf("「%s」是 TOP GP 环境，请走 TOP GP 页的源码镜像，不能使用 T100 的 /api/mirror/pull", e.Name))
		return
	}
	envDir := e.MirrorPath(rootDir)
	if envDir == "" {
		writeJSON(w, 200, map[string]any{"ok": false, "error": "请先设置镜像根目录，或在该环境行单独指定镜像目录"})
		return
	}

	s.mirrorMu.Lock()
	if s.mirror.Running {
		s.mirrorMu.Unlock()
		fail(w, 409, fmt.Errorf("已有拉取任务进行中"))
		return
	}
	s.mirror = mirrorJob{
		Running:   true,
		Env:       e.Name,
		Full:      req.Full,
		Phase:     "connect",
		Message:   "准备中…",
		StartedAt: time.Now().Format(time.RFC3339),
	}
	s.mirrorStart = time.Now()
	// 可取消的 context:用户点「停止」时由 hMirrorStop 调用它。
	// 这里不用 r.Context() —— 那个会在 HTTP 响应写完之后就结束,
	// 而拉取是**有意**跑在请求之外的后台任务(见上面 go s.runMirror)。
	ctx, cancel := context.WithCancel(context.Background())
	s.mirrorCancel = cancel
	s.mirrorMu.Unlock()

	go s.runMirror(ctx, e, envDir, req.Full)
	writeJSON(w, 200, map[string]any{"ok": true})
}

// hMirrorStop 中途取消正在跑的拉取。没有任务在跑时返回 ok=false(而不是报错)——
// 前端可能在轮询间隙多点了一次,那不算错误。
func (s *Server) hMirrorStop(w http.ResponseWriter, r *http.Request) {
	s.mirrorMu.Lock()
	cancel := s.mirrorCancel
	running := s.mirror.Running
	if running && cancel != nil {
		// 先把状态写成"正在停止",再真正取消:取消是异步的(SFTP 读到下一个分片才返回),
		// 这期间轮询会看到 running=true,靠这个文案告诉用户"已经在停了"。
		s.mirror.Message = "正在停止…"
	}
	s.mirrorMu.Unlock()

	if !running || cancel == nil {
		writeJSON(w, 200, map[string]any{"ok": false, "error": "没有正在进行的拉取任务"})
		return
	}
	cancel()
	writeJSON(w, 200, map[string]any{"ok": true})
}

// runMirror 后台执行拉取,把 host 的进度回调同步到任务状态。
func (s *Server) runMirror(ctx context.Context, e *host.NamedSsh, envDir string, full bool) {
	onProgress := func(p host.MirrorProgress) {
		s.mirrorMu.Lock()
		if p.Phase != "" {
			s.mirror.Phase = p.Phase
		}
		if p.Message != "" {
			s.mirror.Message = p.Message
		}
		if p.Bytes > 0 {
			s.mirror.Bytes = p.Bytes
		}
		if p.Total > 0 {
			s.mirror.Total = p.Total
		}
		if p.Files > 0 {
			s.mirror.Files = p.Files
		}
		s.mirror.Elapsed = time.Since(s.mirrorStart).Round(time.Second).String()
		s.mirrorMu.Unlock()
	}

	st, err := host.MirrorPullToDirProgress(ctx, e, envDir, full, onProgress)

	s.mirrorMu.Lock()
	defer s.mirrorMu.Unlock()
	s.mirror.Running = false
	s.mirror.Done = true
	s.mirrorCancel = nil // 任务收尾:清掉取消句柄,停止端点据此回"没有正在进行的任务"
	s.mirror.Elapsed = time.Since(s.mirrorStart).Round(100 * time.Millisecond).String()
	if err != nil {
		// 用户取消 ≠ 失败:标成 cancelled 并给一句人话,不要把 context.Canceled
		// 原样丢到界面上("context canceled" 对用户毫无意义)。
		// 中途取消不会留下被误判的完整基线:.tdict-mirror.ok 没写(见 host.MirrorPullProgress)。
		if errors.Is(err, context.Canceled) {
			s.mirror.Phase = "cancelled"
			s.mirror.Message = "已停止;本地镜像未写完,下次拉取会自动按全量重建"
			return
		}
		s.mirror.Phase = "error"
		s.mirror.Error = err.Error()
		return
	}
	s.mirror.Phase = "done"
	s.mirror.Full = st.Full // 可能因缺基线或白名单升级而自动转全量,以实际为准
	s.mirror.Bytes = st.Bytes
	s.mirror.Total = st.Bytes
	s.mirror.Files = st.Files
	s.mirror.Pruned = st.Pruned
	if st.Note != "" {
		s.mirror.Note = st.Note
		s.mirror.Message = st.Note
	} else {
		s.mirror.Message = fmt.Sprintf("完成:%d 个文件,共 %s", st.Files, humanBytes(st.Bytes))
		if st.Pruned > 0 {
			s.mirror.Message += fmt.Sprintf(";清理本地备份 %d 个", st.Pruned)
		}
	}
}

// mirrorSnapshot 取当前(或最近一次)拉取状态副本。
func (s *Server) mirrorSnapshot() mirrorJob {
	s.mirrorMu.Lock()
	defer s.mirrorMu.Unlock()
	return s.mirror
}

// humanBytes 字节数人性化(服务器侧消息用)。
func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
