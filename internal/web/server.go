// Package web 是统一本地服务的 HTTP 层。
//
// 一个进程、一个端口、一套页面：
//
//	/debug/       调试工作台 SPA（含其专属的 /debug/api/*）
//	/api/*        统一 API —— 环境/数据库配置（/api/hosts）、配置派生状态、
//	              PATH 安装，以及字典类动作（源码镜像拉取、字典同步、BDL 文档）
//
// 合并前这里是两套独立 SPA（调试工作台 /debug/ 与字典页 /dict/），各有自己的 /api/*
// 靠挂载前缀分开。字典页已并入调试工作台里的**统一设置页**，于是：
//   - 「配置」（环境、数据库、查询数据源、镜像目录、同步目标、BDL 目录）统一走 /api/hosts；
//   - 「动作」（拉源码镜像、跑字典同步）留在设置页「数据字典」分区的对应卡片里，
//     端点从 /dict/api/* 升到共享层 /api/*；
//   - 环境清单只有一份数据源（config.json 的 hosts 节），这就是"统一配置管理"落地的样子。
package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/designer"
	"tt/internal/envmove"
	"tt/internal/erpdb"
	"tt/internal/host"
	"tt/internal/sshtun"
)

// DefaultListen 是 serve 未显式配置监听地址时的默认地址。
// 合并前 TDebug 的 debug.listen 与 TDictCli 的 server.DefaultListen 就是同一个值，
// 合并后自然收敛成同一个端口。
const DefaultListen = config.DefaultListen

// maxPortTries 端口被占用时向后探测的次数（原 TDictCli 的行为：最多顺延 50 个端口）。
const maxPortTries = 50

// Options 装配一个统一服务所需的全部外部依赖。
type Options struct {
	// DebugFS 前端构建产物根目录（dist/debug）。nil 或没构建时回落引导页。
	DebugFS fs.FS
	// ConfigPath 配置文件路径。空则按统一规则解析（写路径允许缺省落点）。
	ConfigPath string
	// Version 版本号，出现在 /api/health 与引导页上。**给人看的完整串**
	// （`0.2.2 (commit 25a639d, 2026-10-07)`），不要拿它去比较版本。
	Version string
	// BareVersion 同一个版本的**裸形态**（`0.2.2`），只给更新机制用。
	//
	// 两个字段分开是必须的：检查缓存是以裸版本为键存的（`tt version` 读的也是它），
	// 把展示串喂进比较会被解析器拒掉 —— 表现是设置页的「检查更新」永远失败，
	// 而错误文案里那句 “不是 MAJOR.MINOR.PATCH” 看起来像是发布方写错了版本号。
	// 空串 = 这份构建没有版本号（`make build`），更新只允许查到“版本号不可用”为止。
	BareVersion string

	// Debug 调试子系统（工作台页面 + 其 /debug/api/*）。nil = 未接入。
	Debug http.Handler
	// Dict 字典子系统（源码镜像拉取 / 字典同步 / BDL 文档）。nil = 未接入。
	//
	// 它不再是"另一个页面"——合并前字典页有自己的一套 SPA，两套页面各占一个挂载前缀；
	// 那个页面已并入统一设置页，于是它的端点直接挂在共享层 /api/ 下。
	Dict http.Handler
	// GP TOP GP 执行面（环境/源码镜像/字典同步），挂在 /api/gp/ 下。
	GP http.Handler

	// Reloaders 配置被本服务改写后需要重新加载内存态的子系统。
	//
	// 起作用的场景：设置页写的是同一份 hosts 节。调试子系统在内存里持有一份
	// *Config 并做了热替换，若从设置页改了环境，它得跟着重读 —— 否则一次页面写
	// 下去，它还按旧环境连，表现是"改了没生效"。
	Reloaders []ConfigReloader

	// Shutdown 收到 POST /api/shutdown 时调用；nil 表示不注册该端点。
	// 让 Go 侧优雅收尾（会话/SSH 连接收口），而不是被直接 kill。
	Shutdown func()

	// SyncDefaultTarget 字典同步目标的缺省位置。
	//
	// 由挂载方注入，好让统一层与字典子系统**对同一个值**达成一致：设置页会显示
	// 「默认位置：X」，而字典子系统真正往那儿写。两边各算一次的话，显示的那个路径
	// 可能不是实际写入的那个 —— 用户按显示去核对会发现文件不在那里。
	//
	// 留空 = 用 config.Locations 的缺省（数据目录下的 erp_data.db）。
	SyncDefaultTarget string

	// ---- 三个"动作"端点的外部依赖 ----
	//
	// 它们三个都要连服务器/连库，所以默认档的测试进不去 —— 而这三条路径上的
	// **错误分支**（信封形状、HTTP 码、stage 取值）恰恰是对外契约，改了没人会响。
	// 抽成函数字段就能只测那一半，不必造一个假的 *host.SSHConn
	// （那个结构体的 cli 是私有字段，伪造要付 8 个方法，见 internal/debug 的 dial 字段）。
	//
	// 全是 nil = 用真实现，所以**生产路径的行为一字不差**。

	// ProbeDB POST /api/dbprobe。nil = host.ProbeDBConfig。
	ProbeDB func(host.DBProbeReq) (*host.DBProbeOut, error)
	// VerifyAcc POST /api/dbaccverify。nil = host.VerifyDBAcct。
	VerifyAcc func(host.DBAccVerifyReq) error
	// OpenDB POST /api/conntest。nil = erpdb.Open。
	//
	// 与前两个不同：它的返回值 `erpdb.Connector` 是**接口**，所以注入一个假 Connector
	// 能测到 hConnTest 的**快乐路径** —— 那是这三个端点里唯一一条能测全的。
	OpenDB func(context.Context, dbconfig.Connection) (erpdb.Connector, error)
}

// ConfigReloader 由持有配置内存态的子系统实现；本服务在配置写入成功后调用它。
type ConfigReloader interface {
	// ReloadConfig 重新从磁盘读取配置并热替换内存态。
	// 实现须保证：并发安全、幂等、失败时保留原有内存态不变。
	ReloadConfig() error
}

// Server 是统一服务。
type Server struct {
	opt  Options
	mux  *http.ServeMux
	once bool
}

// New 装配路由。返回的 Server 可直接交给 http.Server 使用。
func New(opt Options) *Server {
	s := &Server{opt: opt, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler 返回顶层 http.Handler。
func (s *Server) Handler() http.Handler { return s.mux }

// routes 装配路由表。集中在一处，方便一眼看清整个服务的接口面。
//
// 子系统的挂载用带前缀的 API 路径 + 本包统一提供的 SPA，而不是把子系统 handler
// 整个挂在 /debug/ 下：后者的做法会让子系统的 "/" 兜底路由吃掉整个前缀下的
// 所有请求（Go 的 ServeMux 匹配到前缀后不再向外回落），SPA 就永远出不来。
// 只把 /*/api/ 转给子系统，页面由本包统一从 web/dist 提供。
func (s *Server) routes() {
	m := s.mux

	// ---- 统一 API：设置页用的环境/数据库管理 ----
	m.HandleFunc("GET /api/health", s.hHealth)
	m.HandleFunc("GET /api/hosts", s.hHostsGet)
	m.HandleFunc("PUT /api/hosts", s.hHostsPut)
	// 从 T100 设计器的站点清单里发现环境(只读那份加密文件,让用户勾选导入)。
	m.HandleFunc("GET /api/discover/envs", s.hDiscoverEnvs)
	m.HandleFunc("POST /api/dbprobe", s.hDBProbe)
	m.HandleFunc("POST /api/dbaccverify", s.hDBAccVerify)
	m.HandleFunc("POST /api/conntest", s.hConnTest)

	m.HandleFunc("GET /api/config/meta", s.hConfigMeta)
	m.HandleFunc("GET /api/config/status", s.hConfigStatus)
	m.HandleFunc("POST /api/cache/clear", s.hCacheClear)
	// 原生“选择文件夹”对话框。浏览器拿不到本地绝对路径，只有服务端能弹系统对话框；
	// 只允许本机请求（服务可能被配置成监听非回环地址）。
	m.HandleFunc("POST /api/pickdir", s.hPickDir)
	// 前端选择器用的"列目录"：浏览器拿不到本地绝对路径，只能由服务端列出子目录，
	// 前端画成自己的选择器（比服务端弹 Windows 对话框快一个数量级）。
	m.HandleFunc("GET /api/fs/list", s.hFsList)
	// 命令行安装(PATH):应用级动作,归统一层 —— 合并前它在 /dict/api/install 下,
	// 于是共用的设置页要用它就得去调字典子系统的私有 API。
	m.HandleFunc("GET /api/install", s.hInstallGet)
	m.HandleFunc("POST /api/install", s.hInstallAdd)
	m.HandleFunc("DELETE /api/install", s.hInstallRemove)

	// 技能：官方那一棵随包走，第三方靠设置页登记（config.json 的 skills.externals）。
	// 这里只有**登记**（查/增/删）；**装到哪**是命令行的事（`tt install skills`）——
	// 前端不再提供第二条安装入口，也就没有“按请求里的路径往磁盘写”这个面。
	m.HandleFunc("GET /api/skills", s.hSkillsGet)
	m.HandleFunc("POST /api/skills/add", s.hSkillsAdd)
	m.HandleFunc("POST /api/skills/remove", s.hSkillsRemove)

	// tt 自身的更新（查/装）。与 install 同一层：应用级动作，不属于任何子系统。
	m.HandleFunc("GET /api/update", s.hUpdateGet)
	m.HandleFunc("POST /api/update/check", s.hUpdateCheck)
	m.HandleFunc("POST /api/update/apply", s.hUpdateApply)
	if s.opt.Shutdown != nil {
		m.HandleFunc("POST /api/shutdown", s.hShutdown)
	}

	// ---- 调试子系统：挂在 /debug/api/（它自己的页面在 /debug/） ----
	if s.opt.Debug != nil {
		m.Handle("/debug/api/", http.StripPrefix("/debug", s.opt.Debug))
	}

	// ---- 字典子系统的端点：直接挂在共享层 /api/ 下 ----
	//
	// 合并前它们挂在 /dict/api/，因为那时还有一套独立的字典页 SPA，两套页面各有自己的
	// /api/* 需要靠前缀分开。那个页面已并入统一设置页，「/dict」这套前缀不再有存在理由：
	// 这些端点（源码镜像拉取、字典同步、BDL 文档）就是统一 API 的一部分。
	//
	// 包内注册的就是 /api/… 绝对路径，所以这里不加 StripPrefix。
	// 上面那些更具体的具名路由优先于这条前缀模式，不会被它遮住。
	if s.opt.Dict != nil {
		m.Handle("/api/", s.opt.Dict)
	}

	// ---- TOP GP：/api/gp/* ----
	// 比上面的 /api/ 兜底更具体，ServeMux 会让它优先匹配，不会被字典子系统吃掉。
	if s.opt.GP != nil {
		m.Handle("/api/gp/", s.opt.GP)
	}

	// ---- 页面（只剩调试工作台一套 SPA） ----
	m.Handle("/debug/", SPAHandler(s.opt.DebugFS, "/debug/", "界面未构建"))

	// ---- 根路径 ----
	m.HandleFunc("/", s.hRoot)
}

// ListenAndServe 在 addr 上启动服务。addr 为空时用配置里的 listen，再空用 DefaultListen。
// 端口被占用时向后顺延（最多 maxPortTries 个），返回实际使用的地址。
func (s *Server) ListenAndServe(addr string) (string, error) {
	if addr == "" {
		addr = s.listenFromConfig()
	}
	ln, used, err := listenWithFallback(addr)
	if err != nil {
		return "", err
	}
	srv := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
		// 调试工作台有长连接的 WebSocket 与长时间挂起的命令，
		// 所以不设 WriteTimeout / IdleTimeout。
	}
	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[tt] HTTP 服务退出: %v", err)
		}
	}()
	return used, nil
}

// listenFromConfig 读取配置里的监听地址；读不到就用缺省值。
func (s *Server) listenFromConfig() string {
	path, err := s.configPath(false)
	if err != nil {
		return DefaultListen
	}
	r, err := config.Load(path)
	if err != nil || r.Listen == "" {
		return DefaultListen
	}
	return r.Listen
}

// listenWithFallback 在 addr 上监听；被占用时按端口递增顺延。
// 返回监听器与实际使用的地址。
func listenWithFallback(addr string) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		return ln, ln.Addr().String(), nil
	}
	hostPart, portPart, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		return nil, "", fmt.Errorf("监听地址格式不正确 %q: %w", addr, err)
	}
	port := 0
	if _, e := fmt.Sscanf(portPart, "%d", &port); e != nil || port <= 0 {
		return nil, "", fmt.Errorf("监听 %s 失败: %w", addr, err)
	}
	for i := 1; i <= maxPortTries; i++ {
		try := net.JoinHostPort(hostPart, fmt.Sprint(port+i))
		if ln, e := net.Listen("tcp", try); e == nil {
			log.Printf("[tt] %s 被占用，改用 %s", addr, try)
			return ln, ln.Addr().String(), nil
		}
	}
	return nil, "", fmt.Errorf("监听 %s 失败，且向后探测 %d 个端口都被占用: %w", addr, maxPortTries, err)
}

// configPath 解析配置文件路径。allowMissing=true 时返回缺省落点而不是报错。
func (s *Server) configPath(allowMissing bool) (string, error) {
	if s.opt.ConfigPath != "" {
		return s.opt.ConfigPath, nil
	}
	return config.ResolvePath("", allowMissing)
}

// ---------- 统一 API 处理器 ----------

// hHealth 健康检查。server 字段是**探活标记**：服务的单实例判断（见
// internal/cli/debug/servebg.go 的 serveUp）靠它认领"这个端口上跑的是我们自己"，
// 而不是恰好占着同一端口的别的 HTTP 服务。改这个值会让 --stop 认不出在跑的实例。
func (s *Server) hHealth(w http.ResponseWriter, r *http.Request) {
	path, _ := s.configPath(true)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"server":  "tt-unified",
		"version": s.opt.Version,
		"config":  path,
		"debug":   s.opt.Debug != nil,
		"dict":    s.opt.Dict != nil,
	})
}

// hHostsGet 返回共用环境清单。设置页的「设置 → 环境」读它。
func (s *Server) hHostsGet(w http.ResponseWriter, r *http.Request) {
	path, err := s.configPath(false)
	if err != nil {
		writeErr(w, http.StatusNotFound, err)
		return
	}
	root, err := config.Load(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"config": path,
		"sshs":   root.Hosts.SSHs,
		// 监听地址也在这一份里下发，设置页的「外观/高级」页要用
		"listen":  root.Listen,
		"debug":   root.Debug,
		"query":   root.Query,
		"mirror":  root.Mirror,
		"dict":    root.Dict,
		"sync":    root.Sync,
		"version": s.opt.Version,
	})
}

// hostsPutReq 是 PUT /api/hosts 的请求体。
//
// 只带要改的节：nil 表示"这一节不动"。这样设置页的各张卡片各改各的节时不会互相覆盖 ——
// 原来 TDebug 的 hSettingsPut 就只替换自己那一节，这里沿用同样的约定。
type hostsPutReq struct {
	Hosts  *config.Hosts         `json:"hosts,omitempty"`
	Listen *string               `json:"listen,omitempty"`
	Debug  *config.DebugSettings `json:"debug,omitempty"`
	Query  *config.QuerySettings `json:"query,omitempty"`
	Mirror *config.MirrorSettings
	Dict   *config.DictSettings
	Sync   *config.SyncSettings
}

func (s *Server) hHostsPut(w http.ResponseWriter, r *http.Request) {
	path, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var req hostsPutReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	// 环境改名是跨文件迁移：先把要搬的落点规划清楚、并在**写配置之前**搬完。
	// 反过来（先写配置、后搬文件）一旦搬失败，配置已经是新名、文件还挂在旧名 ——
	// 那正是本 PR 要消灭的悬空状态，而且用户只会看到"保存成功"。
	oldRoot, err := config.Open(path)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	var sec map[string]any
	var renamed []envRename
	var removedEnvs []string
	// renamedMeta 记下旧名对应的 host/zone：按旧名算 ents 的目录段。
	renamedMeta := map[string]struct{ host, zone string }{}
	if req.Hosts != nil {
		if err := validateHosts(req.Hosts); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		sec, err = structToSection(*req.Hosts)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		// 保留 TOP GP 的环境：设置页那份表单只认识 T100 环境，整份覆盖会把 GP 条目抹掉
		// （实测：在设置页点一次「保存」，GP 环境就消失）。GP 归 /api/gp/env 管，
		// 这里只把它们原样贴回去。
		if oldHosts, ok := oldRoot["hosts"].(map[string]any); ok {
			if oldList, ok := oldHosts["sshs"].([]any); ok {
				cur, _ := sec["sshs"].([]any)
				names := map[string]bool{}
				for _, it := range cur {
					if m, ok := it.(map[string]any); ok {
						if n, ok := m["name"].(string); ok {
							names[n] = true
						}
					}
				}
				for _, it := range oldList {
					m, ok := it.(map[string]any)
					if !ok {
						continue
					}
					if prod, _ := m["product"].(string); !strings.EqualFold(prod, "gp") {
						continue
					}
					n, _ := m["name"].(string)
					if names[n] {
						writeErr(w, http.StatusBadRequest, fmt.Errorf(
							"环境名 %q 已被 TOP GP 环境使用；T100 与 TOP GP 不能同名，请改名", n))
						return
					}
					cur = append(cur, m)
					names[n] = true
				}
				sec["sshs"] = cur
			}
		}
		oldNames := map[string]bool{}
		if oldHosts, ok := oldRoot["hosts"].(map[string]any); ok {
			oldEntries := envEntries(oldHosts)
			renamed, removedEnvs = diffEnvs(oldEntries, envEntries(sec))
			for _, e := range oldEntries {
				oldNames[e.name] = true
			}
			for _, r := range renamed {
				for _, e := range oldEntries {
					if e.name != r.From {
						continue
					}
					host, _ := e.raw["host"].(string)
					zone, _ := e.raw["zone"].(string)
					renamedMeta[r.From] = struct{ host, zone string }{host: host, zone: zone}
				}
			}
		}
		// 新增/改名的名字要能安全当路径键与 `local:<名>`。老配置里已有的怪名字
		// 不在这里拦（否则一次与它无关的保存会被旧数据卡住）。
		for i := range req.Hosts.SSHs {
			if name := req.Hosts.SSHs[i].Name; !oldNames[name] {
				if err := config.ValidateEnvName(name); err != nil {
					writeErr(w, http.StatusBadRequest, err)
					return
				}
			}
		}
	}

	// 1) 规划：任何一个目标落点被占用都整体拒绝，配置文件一个字节都不写。
	type plannedMove struct {
		r     envRename
		moves []envmove.Move
	}
	var plans []plannedMove
	for _, r := range renamed {
		if err := config.ValidateEnvName(r.To); err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		meta := renamedMeta[r.From]
		moves, err := envmove.Plan(path, r.From, r.To, meta.host, meta.zone)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err)
			return
		}
		plans = append(plans, plannedMove{r: r, moves: moves})
	}
	// 2) 搬文件（此时配置仍是旧名）。失败就把已搬的搬回去。
	var executed []plannedMove
	for _, p := range plans {
		if err := envmove.Execute(p.moves, p.r.To, p.r.From); err != nil {
			for i := len(executed) - 1; i >= 0; i-- {
				if uerr := envmove.Undo(executed[i].moves, executed[i].r.From); uerr != nil {
					log.Printf("[tt] 回滚环境 %s 的落点失败: %v", executed[i].r.From, uerr)
				}
			}
			writeErr(w, http.StatusInternalServerError, err)
			return
		}
		executed = append(executed, p)
	}
	rollbackMoves := func() {
		for i := len(executed) - 1; i >= 0; i-- {
			if uerr := envmove.Undo(executed[i].moves, executed[i].r.From); uerr != nil {
				log.Printf("[tt] 回滚环境 %s 的落点失败: %v", executed[i].r.From, uerr)
			}
		}
	}

	// 3) 再写配置：改名 + query.source 改写 + 悬空校验在同一次原子写里完成。
	err = config.Edit(path, nil, func(root map[string]any) error {
		if req.Hosts != nil {
			// renamed / removedEnvs 只为"改名不算删"的判断保留；文件已经在上一步搬完。
			// 保留前端表单不管理的 db 字段（目前是 viaSsh）。
			// 配置页只编辑 type/host/port/service/database/readonlySql/accounts，
			// 整节替换会把手工配的 SSH 隧道悄悄抹掉 —— 那是"保存一次就丢配置"，
			// 而用户完全看不出发生过什么。按环境名对齐，缺什么补什么。
			preserveUngovernedFields(root, sec)
			root["hosts"] = sec
		}
		if req.Listen != nil {
			root["listen"] = *req.Listen
		}
		// 逐节判空。注意**不能**写成 map 循环 + `if v == nil { continue }` ——
		// 那些字段是指针类型，把 nil 指针装进 any 得到的接口值不等于 nil，
		// 于是"本次没提交的节"会被序列化成 null 再当成空对象写回去，
		// 把用户原有的 query/mirror/debug 全清掉。这里逐个类型化判断。
		patches := []struct {
			key string
			set func() error
		}{
			{"debug", func() error {
				if req.Debug == nil {
					return nil
				}
				return putSection(root, "debug", req.Debug)
			}},
			{"query", func() error {
				if req.Query == nil {
					return nil
				}
				return putSection(root, "query", req.Query)
			}},
			{"mirror", func() error {
				if req.Mirror == nil {
					return nil
				}
				return putSection(root, "mirror", req.Mirror)
			}},
			{"dict", func() error {
				if req.Dict == nil {
					return nil
				}
				return putSection(root, "dict", req.Dict)
			}},
			{"sync", func() error {
				if req.Sync == nil {
					return nil
				}
				return putSection(root, "sync", req.Sync)
			}},
		}
		for _, p := range patches {
			if err := p.set(); err != nil {
				return err
			}
		}
		// 改名与数据源在同一次写入里保持一致：source 指向旧名就跟着改。
		for _, r := range renamed {
			config.RewriteQuerySource(root, r.From, r.To)
		}
		// 保存时就拦住悬空的数据源：用户刚改过数据源时做完整校验。
		// 悬空引用不会当场报错，只会在下次查询时变成「本地库不存在」——
		// 那个症状与「数据不存在」长得一模一样，只能靠写入时拦。
		if req.Query != nil {
			if err := config.ValidateQuerySource(root); err != nil {
				return err
			}
		}
		// 改名没被识别出来时（例如同时改了 host，①② 都不成立），旧名字会从配置里
		// 消失而 query.source 还指着它。不能指望 ValidateQuerySource 全量跑：
		// 历史配置里的裸 `local` 会被它判成错，那不该阻塞一次与数据源无关的保存。
		if len(removedEnvs) > 0 {
			if env := config.QuerySourceEnv(root); env != "" {
				for _, old := range removedEnvs {
					if env == old {
						return fmt.Errorf("query.source 仍指向已删除/改名的环境 %q；若是改名，请先只改名（不改 host 等字段），再改其它内容", env)
					}
				}
			}
		}
		root["schemaVersion"] = config.SchemaVersion
		return nil
	})
	if err != nil {
		// 配置没写成（校验失败/IO）→ 把已经搬走的落点搬回旧名。
		rollbackMoves()
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	// 通知持有配置内存态的子系统（如调试服务）重新加载。
	// 失败只记日志：配置已经落盘成功，重新加载失败不该把一个成功的写报成失败，
	// 但必须让用户看见 —— 否则就成了"保存成功但没生效"。
	for _, r := range s.opt.Reloaders {
		if r == nil {
			continue
		}
		if err := r.ReloadConfig(); err != nil {
			log.Printf("[tt] 配置已保存，但子系统重新加载失败: %v", err)
		}
	}
	s.hHostsGet(w, r)
}

// envEntry 是 hosts.sshs 里的一条（原始 JSON 形态 + 它在列表里的下标）。
type envEntry struct {
	idx  int
	name string
	raw  map[string]any
}

// envRename 是一对"改名"（旧名 → 新名）。
type envRename struct{ From, To string }

// envEntries 按顺序取 hosts 节里的环境条目。
//
// 没有名字的照留（下标要对齐 —— 改名判定第一道判据就是"同一个下标"）。
func envEntries(hostsSec map[string]any) []envEntry {
	list, _ := hostsSec["sshs"].([]any)
	out := make([]envEntry, 0, len(list))
	for i, e := range list {
		m, _ := e.(map[string]any)
		if m == nil {
			continue
		}
		name, _ := m["name"].(string)
		out = append(out, envEntry{idx: i, name: name, raw: m})
	}
	return out
}

// envIdent 是对比两条环境时**看身份、不看名字**的那部分：同一台机器上的同一个库。
var envIdentEnvKeys = []string{"host", "port", "user", "zone"}

// sameEnvBesidesName 判"两条环境除了名字之外一模一样"（设置页改名就是这个形态）。
func sameEnvBesidesName(a, b map[string]any) bool {
	return jsonEqual(withoutKey(a, "name"), withoutKey(b, "name"))
}

// sameEnvIdentity 判"两条环境指向同一台机器上的同一个库"（忽略名字，也忽略别的无关字段）。
//
// 第二道判据用它：重排 + 改名时下标对不上，但 host/zone/db 还在。
func sameEnvIdentity(a, b map[string]any) bool {
	for _, k := range envIdentEnvKeys {
		if !jsonEqual(a[k], b[k]) {
			return false
		}
	}
	ad, _ := a["db"].(map[string]any)
	bd, _ := b["db"].(map[string]any)
	for _, k := range []string{"type", "host", "port", "service", "database"} {
		if !jsonEqual(ad[k], bd[k]) {
			return false
		}
	}
	return true
}

// withoutKey 复制一份并去掉某个键（不改原 map）。
func withoutKey(m map[string]any, key string) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}

// jsonEqual 比两个 any 的 JSON 形态（map 的键序由 encoding/json 归一，再不用手写深比较）。
func jsonEqual(a, b any) bool {
	ab, aerr := json.Marshal(a)
	bb, berr := json.Marshal(b)
	if aerr != nil || berr != nil {
		return false
	}
	return string(ab) == string(bb)
}

// diffEnvs 把"旧名单 → 新名单"分成两类：**改名**（录像要跟着搬）与**真的删掉了**（才级联清）。
//
// 为什么不能只看名字（2026-10-15 那次事故的根因）：设置页保存一次 hosts，
// "改了名"与"被删了"在名字集合上长得一模一样，而级联清理只认名字 ——
// 于是**改个名字 = 把那个环境名下长期库里的录像全删掉**。实测那次：清空环境名再保存，
// 前端静默补成 `host-zone`，服务端判定旧环境被删，27 份录像 2.5 MB，`os.RemoveAll` 不可恢复。
//
// 判据三道，从准到宽：
//  1. **同一个下标、只有名字变了** —— 设置页改名的形态（位置不动）；
//  2. 剩下的一对一里**身份相同**（host/port/user/zone/db 全同）—— 重排 + 改名；
//  3. 还剩下的，gone 与 added **数量相等**时按下标配对 —— 当作改名。
//
// 第三道是**故意偏向"搬"而不是"删"**：判错的代价不对称 —— 误判成改名只是把甲的录像
// 留在乙名下（用户看得见、可手动删），误判成删除则是不可再生的录像没了。
// 只有"删得比加得多"（真的少了一个环境）才会走到级联删除。
func diffEnvs(oldList, newList []envEntry) (renames []envRename, removed []string) {
	oldName := map[string]bool{}
	newName := map[string]bool{}
	for _, e := range oldList {
		oldName[e.name] = true
	}
	for _, e := range newList {
		newName[e.name] = true
	}
	usedOld := make([]bool, len(oldList))
	usedNew := make([]bool, len(newList))

	// ① 同一个下标、只有名字变了
	for i := 0; i < len(oldList) && i < len(newList); i++ {
		o, n := oldList[i], newList[i]
		if o.name == "" || n.name == "" || o.name == n.name {
			continue
		}
		if sameEnvBesidesName(o.raw, n.raw) {
			renames = append(renames, envRename{From: o.name, To: n.name})
			usedOld[i], usedNew[i] = true, true
		}
	}
	// ② 身份相同的一对一（名字确实是一边没了、一边新来）
	for i, o := range oldList {
		if usedOld[i] || o.name == "" || newName[o.name] {
			continue
		}
		for j, n := range newList {
			if usedNew[j] || n.name == "" || oldName[n.name] {
				continue
			}
			if sameEnvIdentity(o.raw, n.raw) {
				renames = append(renames, envRename{From: o.name, To: n.name})
				usedOld[i], usedNew[j] = true, true
				break
			}
		}
	}
	// ③ 曾按“剩余数量相等”兜底配对（宁可搬，不要删）。**去掉**：
	// 删除环境 A + 新增无关环境 B 在同一次保存里也会满足"数量相等"，而改名现在会
	// 搬 A 的字典库/企业快照到 B 名下 —— 那是把两个环境的数据串在一起，且没有任何
	// 报错。改名+改 host 这种组合宁可不识别（由 ValidateQuerySource 当场拒绝，
	// 提示先只改名），也不能误搬。①（只改名字）与 ②（身份一致）已经覆盖正常改名。
	// 剩下的才是真的删了
	for i, o := range oldList {
		if !usedOld[i] && o.name != "" && !newName[o.name] {
			removed = append(removed, o.name)
		}
	}
	return renames, removed
}

// preserveUngovernedFields 把旧配置里、配置页表单不管理的字段补进新节。
//
// 有两层：
//
//	节级    hosts.activeEnv                        —— "默认环境"（`config.ActiveHost` 还在拿它兜底）
//	环境级  hosts.sshs[].launchArgs / watchdogSeconds —— 该环境的调试参数覆盖
//	db 级   hosts.sshs[].db.viaSsh                  —— SSH 端口转发隧道
//
// 表单里都没有对应的输入控件，而 PUT 是整节替换，所以不做这一步就会"打开设置页点一次
// 保存，手工配的东西静默消失"。节级按"旧节有、新节没有"整键搬；环境级按环境名对齐。
//
// 权衡：这样也就无法通过配置页删掉它们 —— 想删就手改 config.json，或 `tt config` 改。
// 宁可"删不掉"也不要"保存一次就静默丢配置"。
func preserveUngovernedFields(oldRoot, newSec map[string]any) {
	oldHosts, _ := oldRoot["hosts"].(map[string]any)
	// 节级：旧 hosts 节里有、新节里没有的键（activeEnv 就是这一类）原样搬过去。
	for k, v := range oldHosts {
		if _, has := newSec[k]; !has {
			newSec[k] = v
		}
	}
	oldList, _ := oldHosts["sshs"].([]any)
	if len(oldList) == 0 {
		return
	}
	byName := make(map[string]map[string]any, len(oldList))
	for _, e := range oldList {
		m, _ := e.(map[string]any)
		if m == nil {
			continue
		}
		if name, _ := m["name"].(string); name != "" {
			byName[name] = m
		}
	}

	newList, _ := newSec["sshs"].([]any)
	for _, e := range newList {
		m, _ := e.(map[string]any)
		if m == nil {
			continue
		}
		name, _ := m["name"].(string)
		old, ok := byName[name]
		if !ok {
			continue
		}
		carryOver(old, m, ungovernedEnvKeys)
		oldDB, _ := old["db"].(map[string]any)
		newDB, _ := m["db"].(map[string]any)
		if oldDB == nil || newDB == nil {
			continue
		}
		carryOver(oldDB, newDB, ungovernedDBKeys)
	}
}

// carryOver 把 from 里、to 中缺席的键原样搬过去。
func carryOver(from, to map[string]any, keys []string) {
	for _, k := range keys {
		if _, has := to[k]; has {
			continue
		}
		if v, has := from[k]; has {
			to[k] = v
		}
	}
}

// ungovernedEnvKeys 配置页表单没有输入控件的**环境级**子键，保存时需从旧配置补回。
var ungovernedEnvKeys = []string{"launchArgs", "watchdogSeconds"}

// ungovernedDBKeys 配置页表单没有输入控件的 **db 级**子键，保存时需从旧配置补回。
var ungovernedDBKeys = []string{"viaSsh"}

// hDiscoverEnvs 扫描本机的 T100 设计器站点清单,把里面的站点读成候选环境。
//
// **只读**:解密设计器的 sites.xml 属于跨应用读凭据(里面有明文 SSH 口令),
// 所以这里只读不写 —— 绝不回写设计器的文件(见 internal/designer 的包注释)。
//
// 响应里的 reason 是"为什么没读到/读到几条"的自然语言说明,与 envs 一同展示:
// "这台机器没装设计器"是个正常结论,不该走错误弹窗那条路。
func (s *Server) hDiscoverEnvs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, designer.Discover())
}

// hDBProbe 「从服务器获取」：登录 SSH 后探测库连接要素，结果仅作回填参考。
func (s *Server) hDBProbe(w http.ResponseWriter, r *http.Request) {
	var req host.DBProbeReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	out, err := s.probeDB(req)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// probeDB 走注入的实现；没注入就用真的。三处同款，理由见 Options 的字段注释。
func (s *Server) probeDB(req host.DBProbeReq) (*host.DBProbeOut, error) {
	if s.opt.ProbeDB != nil {
		return s.opt.ProbeDB(req)
	}
	return host.ProbeDBConfig(req)
}

func (s *Server) verifyAcc(req host.DBAccVerifyReq) error {
	if s.opt.VerifyAcc != nil {
		return s.opt.VerifyAcc(req)
	}
	return host.VerifyDBAcct(req)
}

func (s *Server) openDB(ctx context.Context, c dbconfig.Connection) (erpdb.Connector, error) {
	if s.opt.OpenDB != nil {
		return s.opt.OpenDB(ctx, c)
	}
	return erpdb.Open(ctx, c)
}

// hDBAccVerify 逐账号在服务器侧验证。
func (s *Server) hDBAccVerify(w http.ResponseWriter, r *http.Request) {
	var req host.DBAccVerifyReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	if err := s.verifyAcc(req); err != nil {
		writeErr(w, http.StatusBadGateway, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// connTestReq 客户端直连测试。
type connTestReq struct {
	Connection dbconfig.Connection `json:"connection"`
}

// hConnTest 从本机直连数据库，验证「环境 → 库」这条链路真的通。
// 服务器侧可达而客户端不可达时，连接里配的 viaSsh 隧道由调用方先建好再测。
func (s *Server) hConnTest(w http.ResponseWriter, r *http.Request) {
	var req connTestReq
	if err := decodeBody(r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	c := req.Connection
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	// viaSsh：客户端不可达 DB 时，真实查询（dict live）走 SSH 隧道再连。
	// 测试若只直连，就会出现「测试一直超时、实际查询却正常」——测的不是同一条链路。
	// 这里复用 sshtun：起隧道 → host/port 换成 127.0.0.1:本地转发端口 → 再连。
	via := "direct"
	if c.ViaSSH != nil {
		remoteHost, remotePort := c.ViaSSH.EffectiveRemote(c.Host, c.Port)
		tun, err := sshtun.Dial(sshtun.Options{
			Host:       c.ViaSSH.Host,
			Port:       c.ViaSSH.Port,
			User:       c.ViaSSH.User,
			Password:   c.ViaSSH.Password,
			RemoteHost: remoteHost,
			RemotePort: remotePort,
			LocalPort:  c.ViaSSH.BindPort,
		})
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"ok": false, "stage": "tunnel", "error": err.Error(), "via": "tunnel",
			})
			return
		}
		defer tun.Close()
		c.Host = "127.0.0.1"
		c.Port = tun.LocalPort
		via = "tunnel"
	}

	conn, err := s.openDB(ctx, c)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": false, "stage": "connect", "error": err.Error(), "via": via,
		})
		return
	}
	defer conn.Close()

	ver, err := conn.ServerVersion(ctx)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok": false, "stage": "version", "error": err.Error(), "via": via,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "type": conn.Type(), "serverVersion": ver, "address": c.Address(), "via": via,
	})
}

// hConfigMeta 给设置页展示"配置在哪、什么结构"用。
func (s *Server) hConfigMeta(w http.ResponseWriter, r *http.Request) {
	path, _ := s.configPath(true)
	writeJSON(w, http.StatusOK, map[string]any{
		"config":         path,
		"defaultConfig":  config.DefaultConfigPath(),
		"portable":       config.IsPortable(),
		"toolsHome":      config.UserConfigDir(),
		"schemaVersion":  config.SchemaVersion,
		"defaultListen":  DefaultListen,
		"supportedTypes": []string{"oracle", "kingbase"},
	})
}

// hShutdown 让 Go 侧优雅收尾（会话/SSH 连接收口），而不是被直接 kill 掉进程。
//
// 只在本机监听时可用 —— 服务默认绑 127.0.0.1，但用户可以把 listen 配成
// 0.0.0.0；那种情况下暴露一个能停服务的端点是不合适的，所以这里显式挡掉。
// 先回响应再触发停止：调用方要能读到结果，否则它只会看到一个连接被重置。
func (s *Server) hShutdown(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) {
		writeErr(w, http.StatusForbidden, fmt.Errorf(
			"/api/shutdown 只接受本机请求（服务当前监听在非回环地址上）"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	go s.opt.Shutdown()
}

// isLoopback 判断请求来源是否本机。
func isLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// ---------- 静态资源 ----------

// hRoot 把根路径送到工作台。合并后只剩一套 SPA，这里只做跳转。
func (s *Server) hRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if s.opt.Debug != nil {
		http.Redirect(w, r, "/debug/", http.StatusFound)
		return
	}
	writeLanding(w, fmt.Sprintf("tt %s", s.opt.Version))
}

// ---------- 辅助 ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		// 头已经发出去了，只能记日志
		log.Printf("[tt] 写响应失败: %v", err)
	}
}

func writeErr(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"ok": false, "error": err.Error()})
}

// decodeBody 读请求体并解析 JSON。体积上限 4 MiB（配置里可能有较长的账号列表）。
func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return fmt.Errorf("请求体为空")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 4<<20))
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("请求体不是合法 JSON: %w", err)
	}
	return nil
}

// putSection 把类型化结构写回某个顶层节。
//
// **缺席的键保留旧值**（不是整节替换）：请求体是前端照它的表单序列化出来的，
// 表单里没有的字段（后加进 schema 的、只给命令行用的）会整个缺席 —— 整节替换就等于
// "打开设置页点一次保存，那个字段静默消失"。实测踩过两次：`debug.aiTakeover` 与
// `query.limit`（两个都在 schema 里、都有代码读，而前端表单里没有）。
//
// 代价：这些字段也就**没法从配置页删掉**（想删就手改 config.json 或 `tt config`）——
// 与 preserveUngovernedFields 同一笔权衡：宁可"删不掉"，不要"保存一次就静默丢配置"。
func putSection(root map[string]any, key string, v any) error {
	sec, err := structToSection(v)
	if err != nil {
		return fmt.Errorf("配置节 %s 无法序列化: %w", key, err)
	}
	if old, ok := root[key].(map[string]any); ok {
		for k, val := range old {
			if _, has := sec[k]; !has {
				sec[k] = val
			}
		}
	}
	root[key] = sec
	return nil
}

// structToSection 把类型化结构转成可持久化的 map（经 JSON 标签，保留 omitempty 语义）。
func structToSection(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// validateHosts 校验环境清单。环境页保存前调用 —— 拦在这里，比让命令在几秒后
// 拿着一个错端口去连要好得多。
func validateHosts(h *config.Hosts) error {
	// 允许空清单：设置页要能删到零个环境（首次运行本来就是空的），
	// 命令侧仍由 config.LoadHosts 负责报「尚未配置环境」。
	seen := map[string]bool{}
	for i := range h.SSHs {
		e := &h.SSHs[i]
		where := fmt.Sprintf("第 %d 个环境", i+1)
		if strings.TrimSpace(e.Name) == "" {
			return fmt.Errorf("%s 缺少名称", where)
		}
		if e.Name != strings.TrimSpace(e.Name) {
			return fmt.Errorf("环境名 %q 首尾不能有空白", e.Name)
		}
		if seen[e.Name] {
			return fmt.Errorf("环境名 %q 重复", e.Name)
		}
		seen[e.Name] = true
		if e.Host == "" {
			return fmt.Errorf("环境 %q 缺少服务器地址", e.Name)
		}
		// 看门狗负数 = 关掉防行锁保护。设置页能自由写这个字段，所以在这里就拦住：
		// 一个“看起来还在、实际不生效”的保护开关，比没有这个开关更危险。
		if config.WatchdogDisabled(e.WatchdogSeconds) {
			return fmt.Errorf("环境 %q 的停站超时 %d 无效：负数等于关闭看门狗（停站后不会自动放行）；要恢复默认写 0，要缩短超时写正整数",
				e.Name, e.WatchdogSeconds)
		}
		if e.Port < 0 || e.Port > 65535 {
			return fmt.Errorf("环境 %q 的 SSH 端口 %d 超出范围", e.Name, e.Port)
		}
		if e.User == "" {
			return fmt.Errorf("环境 %q 缺少登录账号", e.Name)
		}
		if e.DB == nil {
			continue
		}
		if e.DB.Port < 0 || e.DB.Port > 65535 {
			return fmt.Errorf("环境 %q 的数据库端口 %d 超出范围", e.Name, e.DB.Port)
		}
		switch e.DB.Type {
		case "oracle", "kingbase":
		case "":
			return fmt.Errorf("环境 %q 的数据库未选择类型", e.Name)
		default:
			return fmt.Errorf("环境 %q 的数据库类型 %q 不支持（应为 oracle 或 kingbase）", e.Name, e.DB.Type)
		}
	}
	// 原先这里还校验"默认环境在不在列表里" —— activeEnv 已整个移除,
	// 环境一律由调用方点名,没有"默认环境指向不存在的环境"这种状态了。
	return nil
}
