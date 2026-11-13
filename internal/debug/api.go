package debug

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/coder/websocket"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/host"
	"tt/internal/output"
	"tt/internal/target"
)

// Server 本地调试服务:REST + WebSocket + Web 前端
type Server struct {
	cfg     *Config
	cfgPath string // config.json 路径(重读配置用,可为空=只读)
	web     fs.FS
	webSub  fs.FS  // web/dist 子文件系统(未构建前端时为 nil)
	addr    string // 实际监听地址(端口顺延时与 cfg.Listen 不同;不写回配置)
	mgr     *Manager

	// dial 是拨 SSH 的入口。抽成**字段**是为了让默认测试不拨真机
	// （与 internal/dev/tzs 的 Dialer 接口、internal/web.Options 的函数字段同一个理由）。
	// 四个 handler 直接调它，注入一个必然失败的实现就能测它们的 400/500 分支 ——
	// （从前是五个：那个"服务测试"端点已随 `/api/wstest` 一起取消）
	// 那批分支里有一处**顺序**性质（入参先校验、再收口会话，见 hWSLogDebug 的注释），
	// 改动它没有任何东西会响。
	//
	// NewServer 里默认成 host.Dial，所以**生产路径的行为与从前一字不差**；
	// 只有 _test.go 会替换它。
	dial func(host.SSHConfig) (*host.SSHConn, error)

	// mu 同时保护两类运行态:
	//   stop —— Serve 期间指向该次运行的 cancel(POST /api/shutdown 用,优雅停止);
	//   *cfg 的替换 —— 配置热替换必须整个换掉,不能让别人看到半个新配置。
	// 合并后 hosts 节由统一服务的 /api/hosts 统一读写,调试服务要能安全地重读并热替换,
	// 所以 cfg 的替换也收进这把锁(原来是直接 *s.cfg = nc,没有任何同步)。
	mu   sync.Mutex
	stop func()

	// dbPool 常驻直连会话池：/api/dbsql 在同一进程里复用"建连 + 登录"，
	// 之后的查询只付执行时间（详见 dbpool.go）。nil 表示未初始化（测试里可能直接构造 Server）。
	dbPool *dbDirectPool
}

// NewServer 创建服务实例；cfgPath 为 config.json 路径(重读配置用,可为空=只读)。
//
// web 是**本应用 SPA 的根**（即 web/dist 的内容），不是模块根的嵌入 FS ——
// 合并前两套前端各占 dist 下的一个子目录，剥离那一层得由调用方做；合并后只剩一套 SPA、
// 直接产出在 dist 根，调用方（tt serve / tt debug serve）传 common.WebFrontend() 即可。
// 这里只确认 index.html 在；不在就当没前端，回落成纯 API + 引导页。
func NewServer(cfg *Config, web fs.FS, cfgPath string) *Server {
	// 先把默认环境(hosts.activeEnv,可被 debug.activeEnv 覆盖)的连接与启动参数
	// 合并到运行时字段,再交给 Manager。统一服务 tt serve 只调 LoadConfigAllowEmpty
	// 就直接 NewServer,不会自己补这一步 —— 少了它 s.cfg.SSH/Zone/Topent/DB 全是空的,
	// /api/status 报不出主机,从 /api/sessions 起的会话也会拿空地址去连。
	// 重复调用无副作用(CLI 的 serve 路径本来就会先调一次)。
	cfg.ApplyDefaultEnv()
	s := &Server{cfg: cfg, cfgPath: cfgPath, web: web, mgr: NewManager(cfg), dial: host.Dial, dbPool: newDBDirectPool()}
	// 本地字典库优先的作业解析（2026-10-23）：<数据目录>/dict/<环境>.db 里有同一张
	// gzzz_t/gzza_t，本地命中就省掉每次启动的 SSH+连库（实测约 3s）。CLI 的 serve
	// 路径（servebg.go）同样走 NewServer，这里装配一次两条路都生效。
	s.mgr.LocalJobLookup = func(envName, job string) (JobResolve, bool) {
		return lookupJobLocalDB(config.DictDBPathForEnv(cfgPath, envName), job)
	}
	if web != nil {
		if f, err := web.Open("index.html"); err == nil {
			f.Close()
			s.webSub = web
		}
	}
	return s
}

// Run 启动 HTTP 服务(阻塞到 ctx 取消)。
// 先按 cfg.Listen 建监听;端口被占用时自动顺延到下一个空闲端口(最多尝试 maxPortTries 个),
// 实际监听地址记在 s.addr(不改 cfg.Listen,避免顺延后的端口被设置页保存写回配置)。
func (s *Server) Run(ctx context.Context) error {
	ln, addr, err := s.Listen()
	if err != nil {
		return err
	}
	fmt.Printf("[tt debug] 服务已启动  前端+API: http://%s\n", addr)
	return s.Serve(ctx, ln)
}

// maxPortTries 端口被占用时最多顺延尝试的次数。
const maxPortTries = 50

// Listen 建监听;cfg.Listen 被占用时尝试后续端口,成功后将实际地址记入 s.addr。
// 返回 listener 与实际监听地址(host:port);cfg.Listen 保持配置原值不变。
func (s *Server) Listen() (net.Listener, string, error) {
	host, portStr, err := net.SplitHostPort(s.cfg.Listen)
	if err != nil {
		return nil, "", fmt.Errorf("非法监听地址 %q: %w", s.cfg.Listen, err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return nil, "", fmt.Errorf("非法监听端口 %q: %w", portStr, err)
	}
	var lastErr error
	for i := 0; i < maxPortTries; i++ {
		addr := net.JoinHostPort(host, strconv.Itoa(port+i))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			// 报出真实监听地址:--listen 端口写 0 时端口由系统分配,只能回读才知道;
			// 端口顺延时同理(原来的 requested 端口只用来重试,不作为结果上报)。
			// host 为空 = 监听全部网卡,那种地址浏览器/壳用不了,上报回环地址。
			real := addr
			if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
				h := host
				if h == "" {
					h = "127.0.0.1"
				}
				real = net.JoinHostPort(h, strconv.Itoa(tcp.Port))
			}
			s.addr = real
			return ln, real, nil
		}
		lastErr = err
	}
	return nil, "", fmt.Errorf("监听 %s 失败(尝试 %d 个端口均不可用): %w",
		s.cfg.Listen, maxPortTries, lastErr)
}

// Serve 用既有 listener 提供 HTTP 服务(阻塞到 ctx 取消或连接关闭)。
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	srv := &http.Server{Handler: s.Handler()}

	// 记下本次运行的 cancel,供 POST /api/shutdown 优雅停止
	runCtx, cancelRun := context.WithCancel(ctx)
	s.setStop(cancelRun)
	defer s.setStop(nil)

	go func() {
		<-runCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	err := srv.Serve(ln)
	if err == http.ErrServerClosed {
		s.mgr.CloseAll()
		return nil
	}
	return err
}

// setStop 记录/清除本次运行的停止钩子。
func (s *Server) setStop(f func()) {
	s.mu.Lock()
	s.stop = f
	s.mu.Unlock()
}

// ListenAddr 返回实际监听地址(端口顺延时与配置值不同;未启动时返回配置值)。
func (s *Server) ListenAddr() string {
	if s.addr != "" {
		return s.addr
	}
	return s.cfg.Listen
}

// Handler 返回调试工作台的全部 HTTP 面(REST + WebSocket + 内嵌 SPA 静态资源)。
//
// 面内的路径都是**相对挂载点**的,与合并前独立起服务时一模一样:
//
//	/api/status /api/shutdown /api/events /api/source-file /api/jobinfo
//	/api/sessions…(会话/断点/控制/求值/源码/接口日志)/api/wslogs…
//	/api/dbsql /api/source-preview /api/ws
//	/api/hosts(只读,见 hHostsGet)
//	/(SPA 兜底,hStatic)
//
// 统一服务(tt serve)用 http.StripPrefix("/debug", srv.Handler()) 把它挂在 /debug/ 下,
// 于是面内的 /api/sessions/… 对外是 /debug/api/sessions/…,内嵌 SPA 的 / 对外是 /debug/。
// 这样一个进程里,调试面内的 /api/status、/api/ws 等路径都落在 /debug/api/ 下,
// 与共享层 /api/* 的同名路径不会互相覆盖 —— 合并前两个工具各起各的服务,压根不存在
// 冲突;合并后靠挂载前缀区分。
//
// 注意:统一的 /api/health、/api/hosts、/api/dbprobe、/api/dbaccverify、/api/conntest、
// /api/config/meta 由 tt/internal/web 在顶层持有。本面挂在 /debug/ 前缀下,与它们不在
// 同一个 mux 上,故不会重复注册;hHostsGet 是给**独立运行**(tt debug serve)时的
// tt debug env 用的,挂到前缀下时即 /debug/api/hosts,不参与统一服务的路由。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.routes(mux)
	return mux
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/status", s.hStatus)
	mux.HandleFunc("POST /api/shutdown", s.hShutdown)
	mux.HandleFunc("GET /api/events", s.hEvents)
	mux.HandleFunc("GET /api/source-file", s.hSourceFile)
	mux.HandleFunc("GET /api/jobinfo", s.hJobInfo)
	mux.HandleFunc("POST /api/sessions", s.hLaunch)
	mux.HandleFunc("GET /api/sessions", s.hList)
	mux.HandleFunc("GET /api/sessions/{id}", s.hSnapshot)
	mux.HandleFunc("DELETE /api/sessions/{id}", s.hQuit)
	mux.HandleFunc("POST /api/sessions/{id}/restart", s.hSessionRestart)
	mux.HandleFunc("POST /api/sessions/{id}/close", s.hSessionClose)
	// 规格：在**既有会话**里起作业（会话已存在，{id} 就是宿主）
	mux.HandleFunc("POST /api/sessions/{id}/job", s.hSessionJob)
	mux.HandleFunc("POST /api/sessions/{id}/topent", s.hTopent)
	mux.HandleFunc("POST /api/sessions/{id}/ent", s.hSessionEnt)
	// 规格里的 quit 是 POST（DELETE 保留：旧调用方在用的名字）。两者同一件事。
	mux.HandleFunc("POST /api/sessions/{id}/quit", s.hQuit)
	mux.HandleFunc("POST /api/sessions/{id}/mode", s.hMode)
	mux.HandleFunc("POST /api/sessions/switch", s.hSessionSwitch)
	// 再建一条**空闲会话**（不碰池里任何一条）。与 /switch 的分工见 hSessionCreate。
	mux.HandleFunc("POST /api/sessions/new", s.hSessionCreate)
	mux.HandleFunc("GET /api/sessions/{id}/breakpoints", s.hBPList)
	mux.HandleFunc("POST /api/sessions/{id}/breakpoints", s.hBPAdd)
	mux.HandleFunc("DELETE /api/sessions/{id}/breakpoints/{num}", s.hBPDel)
	mux.HandleFunc("POST /api/sessions/{id}/control", s.hControl)
	mux.HandleFunc("POST /api/sessions/{id}/print", s.hPrint)
	mux.HandleFunc("POST /api/sessions/{id}/where", s.hWhere)
	mux.HandleFunc("POST /api/sessions/{id}/raw", s.hRaw)
	mux.HandleFunc("POST /api/sessions/{id}/why", s.hWhy)
	mux.HandleFunc("GET /api/sessions/{id}/wait", s.hWait)
	mux.HandleFunc("GET /api/sessions/{id}/locals", s.hLocals)
	mux.HandleFunc("GET /api/sessions/{id}/globals", s.hGlobals)
	mux.HandleFunc("GET /api/sessions/{id}/sources", s.hSources)
	mux.HandleFunc("GET /api/sessions/{id}/functions", s.hFunctions)
	mux.HandleFunc("GET /api/sessions/{id}/autovars", s.hAutovars)
	mux.HandleFunc("POST /api/sessions/{id}/autovars", s.hAutovarsSet)
	mux.HandleFunc("POST /api/sessions/{id}/frame", s.hFrame)
	mux.HandleFunc("POST /api/sessions/{id}/locate", s.hLocate)
	mux.HandleFunc("POST /api/sessions/{id}/calibrate", s.hCalibrate)
	mux.HandleFunc("POST /api/sessions/{id}/breakpoints/{num}/enabled", s.hBPEnabled)
	mux.HandleFunc("GET /api/sessions/{id}/source", s.hSource)
	mux.HandleFunc("GET /api/source-preview", s.hSourcePreview)
	mux.HandleFunc("GET /api/wslogs", s.hWSLogs)
	mux.HandleFunc("GET /api/ents", s.hEnts)
	mux.HandleFunc("POST /api/ents/sync", s.hEntsSync)
	mux.HandleFunc("POST /api/dbsql", s.hDBSQL)
	mux.HandleFunc("GET /api/wslogs/content", s.hWSLogContent)
	mux.HandleFunc("POST /api/wslogs/debug", s.hWSLogDebug)
	// 环境清单(只读):独立运行时 tt debug env 读它。统一服务下环境页改的是
	// 顶层 /api/hosts(tt/internal/web),这里那份只在 /debug/ 前缀下可达,互不干扰。
	mux.HandleFunc("GET /api/hosts", s.hHostsGet)
	mux.HandleFunc("GET /api/ws", s.hWS)
	mux.HandleFunc("/", s.hStatic)
}

// ---------- 工具 ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// fail 是**所有**端点的失败出口（契约 C5.15）。
//
// 信封形状是"在现行 `{"ok":false,"error":…}` 上**增加** `code`"：`ok` 与 `error`
// 一个字都不改（B8 有测试盯着它），多出来的是 `code`（**必给**）与
// `candidates`（只在那五种"要一条会话却拿不到/拿错"的码下出现，见 api_target.go 的
// `failureEnvelope` 与 C5.11）。
//
// 为什么 `code` 落在这一层而不是只给那几个端点：C5.15 的原话是"失败信封**增加** `code`（必给）" ——
// 调用方（Agent）靠 `code` 分支，而不是靠猜 `error` 的中文（Q15 / 4.1.1）。给一半端点加，
// 另一半就还得靠文案，那正是这次要根除的东西。
//
// `candidates` 不在这里：它是**会话候选清单**，只有会话寻址那几族失败才知道该列谁
// （候选的九键是 `internal/target` 的 `Candidate`，来自会话池，不是通用错误能凭空造的）。
func fail(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"ok": false, "code": codeOf(code, err), "error": err.Error()})
}

// codeOf 决定一条通用失败的 `code`（C5.15：必给）。
//
// 两条来源，按顺序：
//  1. 错误自己带的码（`*output.Error`，C4.1 的"所有失败路径都带 Code + Exit"）——它是权威；
//  2. 按 HTTP 状态码给一个**现行码**（B5 那一族），语义按 4.1.1 的表：
//     400 = 用法错、404 = 指名的东西不存在、409 = 会话状态不符、其余 = 运行失败。
//
// **为什么不在这里给 SESSION_* 那几个新码**：那几个码的语义是"会话寻址"的结论
// （歧义 / 指名不存在 / 句柄过期 / 别名撞车 / 别名指向的会话不能复用），只有真在会话池上
// 落过位的那几族才判得出来。它们走 `failureEnvelope`（api_target.go），
// 而通用失败按现行码给 —— 这样单会话下老脚本看到的 `code` 与退出码一档不变（H2 / Q15 分档）。
func codeOf(status int, err error) string {
	var oe *output.Error
	if errors.As(err, &oe) && strings.TrimSpace(oe.Code) != "" {
		return oe.Code
	}
	switch status {
	case http.StatusBadRequest:
		return output.CodeUsage
	case http.StatusNotFound:
		return output.CodeNotFound
	case http.StatusConflict:
		return output.CodeSessionState
	default:
		return output.CodeQueryFailed
	}
}

func readBody[T any](w http.ResponseWriter, r *http.Request, out *T) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		fail(w, 400, fmt.Errorf("请求体解析失败: %v", err))
		return false
	}
	return true
}

func (s *Server) sessOf(w http.ResponseWriter, r *http.Request) *Session {
	id := r.PathValue("id")
	sess := s.mgr.Get(id)
	if sess == nil {
		fail(w, 404, fmt.Errorf("会话不存在: %s", id))
	}
	return sess
}

// ---------- 归因与模式闸门 ----------

// actorHeader 调用方声明自己是谁。人与 AI 打的是同一批 REST 端点,服务端分辨不出来,
// 只能靠声明:CLI 统一带 X-Actor: ai,浏览器带 human(不带也按 human 算)。
const actorHeader = "X-Actor"

// ActorSystem 是**平台自己的进程**（serve 拉起的、替某条会话干活的子进程）声明的身份。
//
// 它存在的原因是这一条链：serve 起一个脱离的 `tt …` 子进程 → 那个子进程再调调试线 REST
// （收上一轮、点 ENT、起作业、**在入口停站后放行**）。子进程替会话干活，不该被
// "这条会话是谁在驾驶"的模式闸门挡下。
//
// 所以 `system` 的语义是"平台在替这条会话干活"，不是"某个用户在操作"。
// 它和 `ai` / `human` 一样由调用方声明（`X-Actor` 无鉴权），保护的是**误操作**，
// 不是攻击者 —— 与既有两道闸门同一档。
const ActorSystem = "system"

// 环境变量 TT_ACTOR（"这次调用算谁的动作"）的常量定义在 internal/target.ActorEnv，
// 调试面不再重复定义一份 —— 两边读的必须字面相同，两份定义就会漂。
// 为什么需要它、serve 起链路时怎么传，见那处的注释。

func actorOf(r *http.Request) string {
	switch strings.ToLower(strings.TrimSpace(r.Header.Get(actorHeader))) {
	case "ai":
		return "ai"
	case ActorSystem:
		// 平台自己的进程声明 system：它替这条会话干活，
		// 不受"人/AI 谁在驾驶"这道闸门约束（理由见 Session.MayDo）。
		return ActorSystem
	}
	return "human"
}

// launchModeFor 新会话的初始模式,由**发起方**决定:
// AI 从 CLI 发起的调试(start / wsdebug)默认协作(人只读),人从界面发起的保持纯人工。
// 两条启动路径(hLaunch / hWSLogDebug)必须都过这里 —— 漏一条就会出现
// "AI 起出来的会话却写着纯人工、AI 自己被自己的闸门拦下"的怪状。
func launchModeFor(r *http.Request) string {
	if actorOf(r) == "ai" {
		return ModeCollab
	}
	return ModeSolo
}

// MayWrite 该身份此刻能否对这个会话执行写操作。
// 闸门是**会话级的静态标志**,不做逐命令协商 —— 原先设想的 holder/TTL/续期那一套
// 最难的地方是服务端看不见 AI 的"一轮"(CLI 每次调用都是独立进程),降级成模式之后
// 这一整块就不存在了。
// MayWrite 该身份此刻能否对这个会话执行写操作。
// 闸门是**会话级的静态标志**,不做逐命令协商 —— 原先设想的 holder/TTL/续期那一套
// 最难的地方是服务端看不见 AI 的"一轮"(CLI 每次调用都是独立进程),降级成模式之后
// 这一整块就不存在了。
//
// **`system` 例外**：平台自己拉起的平台子进程声明这个身份（见 ActorSystem）。
// 它要替这条会话干活（收上一轮、点 ENT、起作业、入口停站后**放行**），
// 而"放行"是交互式命令 —— 按 W7 那类命令对 record 用途是拒的，拦住它等于功能不可用。
func (s *Session) MayWrite(actor string) bool {
	if actor == ActorSystem {
		return true
	}
	if s.Mode() == ModeCollab {
		return actor == "ai"
	}
	// solo:AI 只能读。否则"纯人工模式下 AI 只能读不能写"就是摆设。
	return actor != "ai"
}

// writeDeniedMsg 拒绝文案要能直接照做,而不是只说"没权限"。
func writeDeniedMsg(mode, actor string) string {
	if mode == ModeCollab {
		return "协作模式:本次调试由 AI 主导,请在对话中把操作委托给 AI(例如「在 4452 行下个断点」);" +
			"需要自己动手请点界面上的「接管」。"
	}
	if actor == "ai" {
		return "纯人工模式:AI 不可操作。可先执行「tt debug mode collab」接管(人在界面上点「接管」可随时收回)。"
	}
	return "当前模式不允许该操作"
}

// hMode 切换会话模式(纯人工 / 协作)。
// body: {"mode": "solo"|"collab"}
//
// **两个方向都放行,人与 AI 都能切。** 从前 AI 不能自行解除纯人工模式(要请人去界面上
// 点「交给 AI」),这条限制已移除:它并不真的提供保护,却让"人从界面起过一个会话"
// 永久锁住 AI 的写操作 —— AI 连自己接管一步都做不到,只能回过头来求人。
// 人始终有最终控制权:界面上的「接管」随时能把模式切回 solo。
//
// 切换**不中断在飞命令**,只改变"之后谁能写",所以本身没有竞态。
func (s *Server) hMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"`
	}
	if !readBody(w, r, &req) {
		return
	}
	m := strings.ToLower(strings.TrimSpace(req.Mode))
	if m != ModeSolo && m != ModeCollab {
		fail(w, 400, fmt.Errorf("mode 只能是 %s 或 %s", ModeSolo, ModeCollab))
		return
	}
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	actor := actorOf(r)
	text := map[string]string{ModeSolo: "已接管:纯人工模式", ModeCollab: "已交给 AI:协作模式"}[m]
	sess.SetMode(m)
	ev := Event{Type: "log", SessionID: sess.ID, Time: time.Now(), Actor: actor, Action: "session.mode", Text: text}
	if actor == "ai" {
		ev.Type = "ai_action" // 让时间线上明确标出是 AI 做的
	}
	s.mgr.emit(ev)
	writeJSON(w, 200, map[string]any{"ok": true, "mode": m})
}

// sessOfWriteGlobal 是**起会话**那条路径的模式闸门（`hLaunch`）。
//
// 它为什么还叫"Global"：起会话是唯一一个"池子里还没有目标"的写操作 ——
// 闸门要保护的是**被这一轮顶掉的那条会话**，而在 `Manager.Current()` 兜底删掉之后
// （WP4b / Q4 / C8.2），服务端不再从池子里挑一条来判。所以：
//
//   - `target == nil`（池空 / 会话正在被建）→ **放行**：没有东西可保护。
//     空会话同样放行（`cur.Bare()`）：前端一连上来就会建一个还没挂程序的宿主会话，
//     而它是 human 身份建的（→ 纯人工）。若把它也算作"要保护的一轮"，AI 就被一个
//     什么都没在跑的会话永远关在门外 —— 连自己发起调试都做不到。
//   - `target != nil` → 就按**那一条**判"谁能写"（`sessOfWrite` 这一族已经在做的事）。
//
// 多会话下这条闸门**不再替调用方挑一条**：要按会话判就往里传那一条
// （`hSessionSwitch` / `hWSLogDebug` 都先落位再传，见 api_target.go 的 resolveTarget）。
func (s *Server) sessOfWriteGlobal(w http.ResponseWriter, r *http.Request, target *Session) bool {
	if target == nil || target.Bare() {
		return true
	}
	if actor := actorOf(r); !target.MayWrite(actor) {
		fail(w, 403, fmt.Errorf("%s", writeDeniedMsg(target.Mode(), actor)))
		return false
	}
	return true
}

// emitAction 把一次 AI 写操作记进操作时间线(前端按 origin:'ai' 渲染成蓝色行)。
//
// 在**操作成功之后**调用,不在放行时调用:被拒(模式闸门/上一条命令还在跑/参数错)
// 的命令不该在时间线上留痕 —— 真机验证时,一次忙等把 60 多次失败的 next 全记了进去。
// 长命令的"正在进行"由快照里的 inflight 承担(状态栏实时显示"⟳ next(已 12s)"),
// 不需要时间线抢答。
func (s *Server) emitAction(r *http.Request, sess *Session, action, text string) {
	if sess == nil || actorOf(r) != "ai" {
		return
	}
	s.mgr.emit(Event{
		Type: "ai_action", SessionID: sess.ID, Time: time.Now(),
		Actor: "ai", Action: action, Text: text,
	})
}

// sessOfWrite 写操作的会话入口 = sessOf + 模式闸门。被拒时已写好 403 并返回 nil。
//
// 闸门是会话级的静态标志,不做逐命令协商:协作模式下只有 AI 能写,
// 纯人工模式下只有人能写。切换靠界面上的按钮,不进这条路径。
// sessOfWrite 写操作的会话入口 = sessOf + **交互式**调试那道闸门（W7 的 CapInteractive）。
// 被拒时已写好响应并返回 nil。
func (s *Server) sessOfWrite(w http.ResponseWriter, r *http.Request) *Session {
	return s.sessOfWriteCap(w, r, CapInteractive)
}

// sessOfWriteCap 是写操作的会话入口，按**能力类**判（W7）。
//
// 分类的好处：编排（起作业）与收尾任何时候都放行，交互式调试照旧按模式闸门判。
// 判据本身仍只有 `Session.MayDo` 一处。
func (s *Server) sessOfWriteCap(w http.ResponseWriter, r *http.Request, cap Capability) *Session {
	sess := s.sessOf(w, r)
	if sess == nil {
		return nil
	}
	actor := actorOf(r)
	if sess.MayDo(cap, actor) {
		return sess
	}
	fail(w, 403, fmt.Errorf("%s", writeDeniedMsg(sess.Mode(), actor)))
	return nil
}

// ---------- 会话 ----------

func (s *Server) hStatus(w http.ResponseWriter, r *http.Request) {
	m := map[string]any{
		"server": "tdebug-debug",
		"ssh":    s.cfg.SSH.Host,
		"zone":   s.cfg.Zone,

		// wsPrefix:服务区前缀（出貨區 topprd 是 `/ws`，其余区是 `/w`）—— 前端拼占位符地址用。
		// 这条规则只有后端一份实现（`wsURLPrefix`，见 wsurl.go），前端要抄就得抄对：
		// 两边不一致时界面上的占位符地址是错的（实测 zone=t 时前端给 /wtopprd、后端给 /wstopprd）。
		"wsPrefix": wsURLPrefix(s.cfg.Zone),
		"listen":   s.ListenAddr(),
		"watchdog": s.cfg.WatchdogSeconds,
	}
	// topDir 只在登录动态获取后才有意义(未登录无静态值);无会话/未探测时省略
	if top := s.cfg.TopDirActual(); top != "" {
		m["topDir"] = top
	}
	writeJSON(w, 200, m)
}

// hEvents 最近会话事件(tail):GET /api/events?tail=N,供 AI/CLI 观察"发生了什么"。
func (s *Server) hEvents(w http.ResponseWriter, r *http.Request) {
	tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
	writeJSON(w, 200, map[string]any{"ok": true, "events": s.mgr.Events(tail)})
}

// hShutdown 优雅停止服务:POST /api/shutdown {}。
// 效果等同 Ctrl+C —— ctx 取消 → http 收口 + mgr.CloseAll 收掉会话与 SSH 连接。
// 停服务的主要路径是 `tt serve --stop` / `tt debug serve --stop`(按 pid 杀进程);
// 这个端点留给"说得上话但拿不到 pid"的调用方(曾经是桌面外壳的关窗动作)。
// 要求 JSON 请求体(与全站一致):浏览器跨站发不出这个 Content-Type(会先被预检挡下),
// 所以只有本机同源调用方能用,不额外加鉴权。
func (s *Server) hShutdown(w http.ResponseWriter, r *http.Request) {
	var req struct{}
	if !readBody(w, r, &req) {
		return
	}
	s.mu.Lock()
	stop := s.stop
	s.mu.Unlock()
	if stop == nil {
		fail(w, 409, fmt.Errorf("服务尚未就绪"))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
	go stop() // 先回响应再停,避免调用方拿到连接重置
}

// hSourceFile 会话外白名单源码读取:GET /api/source-file?module=&file=&path=&from=&to=
// (AI 信息通道;独立短连接,不经会话,路径限制在登录区源码目录)
func (s *Server) hSourceFile(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, _ := strconv.Atoi(q.Get("from"))
	to, _ := strconv.Atoi(q.Get("to"))
	res, err := s.mgr.ReadSourceStandalone(q.Get("env"), q.Get("module"), q.Get("file"), q.Get("path"), from, to)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "source": res})
}

// hJobInfo 作业→实体程序/模块解析:GET /api/jobinfo?prog=&module= (不启动会话)
func (s *Server) hJobInfo(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	prog := strings.TrimSpace(q.Get("prog"))
	if prog == "" {
		fail(w, 400, fmt.Errorf("需要 prog 参数"))
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "job": s.mgr.ResolveJob(strings.TrimSpace(q.Get("module")), prog)})
}

func (s *Server) hLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Module string `json:"module"`
		Prog   string `json:"prog"`
		SSH    string `json:"ssh"`  // 多 SSH 配置名(设置页维护);空 = 默认连接
		Zone   string `json:"zone"` // 区域覆盖(31/35/36/39/t);空 = 默认区域
		// Name 是 `--name` 给的别名（契约 C2.5 / Q17 / C1.6）。空 = 未命名。
		// 它**就是这条会话的幂等键**（Q23）：同名会话空闲 → 复用；正在跑 / 已停站 → 报
		// `SESSION_TARGET_BUSY` 退 3，**不许静默再开一条**；名字不存在 → 起新会话。
		Name string `json:"name"`
		// Session 是 `--session` **显式指名**的宿主会话（服务端原始句柄 `{id}`，C5.4 / Bug A）。
		// 空 = 没指名 —— 这一档与今天逐字相同（按环境+名字收窄候选，多条即 F1 歧义）。
		// 非空 = "就用这一条当宿主"：候选集当场收敛成它一条，于是**不可能**再报
		// `SESSION_AMBIGUOUS`。从前 CLI 只在本机校验这个标志、POST 里不带它，于是
		// 同环境两条空闲会话时 `--session 恒一#1` 照样歧义 —— 标志在 --help 里写着却帮不上忙。
		Session string `json:"session"`
		// FGLServer 本轮 FGLSERVER 覆盖(指针:给了就是权威,空串 = 本轮强制交回 T100 自动设置)。
		// 存在的理由是"画面去哪"只该由这一轮决定:以前只能改全局配置,
		// 于是一次前端实验就把调试的画面接线顶掉了,而且静默(见 ScreenTarget)。
		FGLServer *string `json:"fglserver"`
		// Ent 是**这一轮**的企业编号(ENT)。从前没有这个字段:企业只能靠"起会话后
		// tt debug topent"或者沿用登录 shell 的 $TOPENT,而后者是静默的 —— "忘了点名"
		// 与"点错了"表现完全一样(作业照跑,只是跑在另一家企业上)。现在它是必填:
		// 空且目标会话也没有显式 override 时,`prepareSession` 会拒掉(见 requireExplicitEnt)。
		Ent string `json:"ent"`
	}
	if !readBody(w, r, &req) {
		return
	}
	if req.Prog == "" {
		fail(w, 400, fmt.Errorf("prog 必填(module 可留空,自动按作业名解析)"))
		return
	}
	// `--session` 显式指名的宿主先落一次位（Bug A）：指名的那条**不在池里**就报
	// `SESSION_NOT_FOUND`，**绝不回落成"那就按候选再收窄一次"** —— 回落会把"我点的那条
	// 没了"变成"落到另一条上、而输出看起来一切正常"，那正是这次重构要根除的那句话。
	// 落上了就把**服务端原始句柄**带下去（路径与请求体只认 id，C5.4）；"它还是不是活着"
	// 由 `prepareSessionOn` 再兜一次 —— 两次判断之间它可能刚好断掉，与
	// `checkNameFree` / `assignIdentityLocked` 那条两段式同款（判据只有一份，只是兜两次）。
	if id := strings.TrimSpace(req.Session); id != "" {
		sess, reason, cands := s.resolveTarget(id)
		if reason != targetOK {
			// 非空 id 只可能落到 OK 或 Missing 两种结局：Missing → 404 + `SESSION_NOT_FOUND`。
			failTarget(w, reason, cands, "调试启动")
			return
		}
		req.Session = sess.ID
		// 指名了一条会话却没点名 `--env`：环境就是**它自己的那个**。
		//
		// 为什么必须在这里补、而且必须排在 `launchCfg` 之前：不补的话 cfg 会按"服务端当前
		// 环境"（可能是空的 `-`）组装，于是同一条请求被判成"换目标"，而 `namedHostConflict`
		// 给出的那句话恰恰写着「指名一条会话时环境由它决定」—— 说了却不做，实测就是这么
		// 自相矛盾的（CLI 靠 body["ssh"] 补了这一刀，直接调 API 的客户端没有）。
		// 只在空值时补：两边都给了的时候矛盾由 `namedHostConflict` 如实报，不在这里替谁做主。
		// 只在**配置里真有这个名字**时才补：没有点名环境时 `EnvName()` 会合成一个
		// `host-zone` 形式的占位名（空主机+空区域就是 `-`），把它当环境名送进 `launchCfg`
		// 只会换来一句"环境 "-" 不存在"。
		if env := sess.EnvNameNow(); req.SSH == "" && env != "" && s.cfg.CloneEnv(env) != nil {
			req.SSH = env
		}
	}
	// 按名换 SSH(环境)/ 覆盖区域/ 覆盖本轮 FGLSERVER:克隆配置,默认链路零影响
	cfg, err := s.launchCfg(req.SSH, req.Zone, req.FGLServer)
	if err != nil {
		fail(w, 404, err)
		return
	}
	// `--ent`：只改**本轮**配置（launchCfg 在没有任何覆盖时会直接返回 s.cfg，
	// 就地写它会把这个企业留给后面每一轮）。浅拷贝与 CloneWithZone 同一套做法。
	if e := strings.TrimSpace(req.Ent); e != "" {
		c2 := *cfg
		c2.Topent = host.EntValue(e)
		cfg = &c2
	}
	// `--name` 的判定都在**建立会话之前**（C2.8 的同一条纪律：一个字节都不该白花）。
	// 顺序是：prog（必填）→ 环境（存在性，F6）→ 指名的会话（存在性，F 那一族）→
	// 别名（F19 取值纪律 + Q23 的三种结局）→ 真正登记时兜住撞车（F18，竞态那一支，在 LaunchWith 里）。
	// 为什么别名排在环境之后：它与目标层"文法 → 环境存在 → 落位"的顺序同源 ——
	// Q23 判的是"落位"，而落位要先知道环境是哪一个。指名的会话排在别名之前同理：
	// 它也是"落位"的一部分（"落在哪一条"），而别名判的是"这条会话能不能叫这个名字"。
	name, ok := s.beginNamedLaunch(w, req.Name)
	if !ok {
		return
	}
	// 启动会结束当前这次调试,所以它同样受模式闸门约束(无会话或空会话放行)。
	// **不传目标**是刻意的:这一轮要保护的那条会话正被顶掉,而"是哪一条"由
	// `prepareSession` 的复用/收口规则决定(它才是那条规则的家)。池里有 >=2 条时
	// 这条闸门不再替谁挑一条 —— 它按传进来的目标判,没有目标就是"没有东西可保护"。
	// 显式指名（`--session`）也不改这一条：指名回答的是"落在哪条会话上"，不是
	// "这一轮要不要过它的模式闸门" —— 起会话今天就能顶掉任意一条会话，
	// 多一个标志不该改变"AI 能不能起"（那是既有行为，H2 保护的就是这一档）。
	if !s.sessOfWriteGlobal(w, r, nil) {
		return
	}
	sess, err := s.mgr.LaunchWith(cfg, req.Module, req.Prog, name, req.Session)
	if err != nil {
		if s.writeLaunchFailure(w, err) { // F18 别名撞车 / F1 多会话歧义（见 api_target.go）
			return
		}
		fail(w, 409, err)
		return
	}
	// reused: 这次是"复用已登录宿主"还是"新建连接"。CLI 据此说明"我没有真的重连",
	// 免得把"复用了旧会话"当成"新建了一个会话"——那正是上错号时最难发现的一步。
	reused := sess.Booted()
	// 模式由发起方决定:AI 从 CLI 发起 → 协作(人只读);人从界面发起 → 纯人工(与旧行为一致)
	sess.SetMode(launchModeFor(r))
	if actorOf(r) == "ai" {
		s.mgr.emit(Event{
			Type: "ai_action", SessionID: sess.ID, Time: time.Now(),
			Actor: "ai", Action: "session.launch",
			Text: fmt.Sprintf("启动调试 %s/%s", req.Module, req.Prog),
		})
	}
	go func() {
		if err := sess.Launch(r.Context()); err != nil {
			log.Printf("[debug] 会话 %s(%s) 启动失败: %v", sess.ID, sess.Prog, err)
			s.mgr.emit(Event{Type: "log", SessionID: sess.ID, Text: "启动失败: " + err.Error()})
			if sess.Booted() {
				// 复用宿主上启动失败:Launch 已把会话收回空闲(可换作业重试/重启会话);
				// 仅在异常退回 exit 时移除,避免卡在 loading
				if sess.State() == StateExit {
					s.mgr.Remove(sess.ID)
				}
			} else {
				// 首次登录失败:宿主不可用,终结并移除,避免卡在 loading 挡住下一次启动
				sess.ForceExit()
				s.mgr.Remove(sess.ID)
			}
		}
	}()
	// runProg: gzzz_t 解析出的实体程序(前端源码命名/预取要用它,而不是作业编号)
	writeJSON(w, 200, map[string]any{"ok": true, "sessionId": sess.ID,
		"env": sess.EnvNameNow(), "reused": reused,
		"module": sess.Module, "prog": sess.Prog, "runProg": sess.RunProg})
}

// hList 列会话。
//
// 过滤（规格 §2 的 `GET /api/sessions?env=…&all=1`）：
//   - `env`：只看这个环境（精确匹配环境名，不做模糊 —— 环境名是身份的一部分）；
//   - `all`：含已断开（`exit`）的残留。默认仍全部返回（与既有行为一致），
//     但给了 `all=0` 就只给活着的 —— 两个方向都要能表达，避免"想排除却没法说"。
func (s *Server) hList(w http.ResponseWriter, r *http.Request) {
	list := s.mgr.Snapshot()
	env := strings.TrimSpace(r.URL.Query().Get("env"))
	if env != "" {
		kept := list[:0]
		for _, b := range list {
			if b.Env == env {
				kept = append(kept, b)
			}
		}
		list = kept
	}
	if v := strings.TrimSpace(r.URL.Query().Get("all")); v == "0" || v == "false" {
		kept := list[:0]
		for _, b := range list {
			if b.State != "exit" {
				kept = append(kept, b)
			}
		}
		list = kept
	}
	writeJSON(w, 200, map[string]any{"ok": true, "sessions": list})
}

// identityKeysOf 把**会话身份那一组键**从 `SessionBrief` 里搬进快照（契约 C5.1 / C5.3 / C5.4 / C5.9）。
//
// 为什么是这一层而不是往里抄字段：`SessionBrief`（`briefOf` 那一份投影）是这**七个**键的
// **唯一出处**（C5.17–C5.19 正在消灭的正是"客户端/别处手抄第二份事实"）。这里只做一件事 ——
// 把已有投影里的七个键**按键搬到 map 上**，绝不在本文件重新拼一遍 `环境#序号` / `环境/作业#序号`
// / `ent` 的取法。谁改 `briefOf` 的取法，这里跟着变，不需要有人记得改两处。
//
// 为什么是**七个**（`ref` / `display` / `name` / `ent` / `n` / `kind` / `epoch`）：前六个是
// 列表快照与寻址用的身份，而 **`ent` 是 C5.9 的身份行要的那一个** ——
// `会话 <display> · <name> · <state> · ENT <ent>`。上一版这里漏了 `ent`，CLI 只好在自己那边
// 从快照的 `topentSession` 兜底补一个；那等于让 `ent` 与 `topentSession` 两个近似的东西
// 在客户端做取舍，正是这一路重构要拆掉的"第二份事实"。补上第七个键之后，
// CLI 的 `ENT` 只有**这一个**来源（`SessionBrief.Ent` = `Session.CurrentEnt()`）。
//
// **恒定键（C5.1 / C5.3）取不到就给空串/零值，不省键**：`ref` 在序号还没分配时是空串、
// `name` 未命名是空串、`ent` 取不到是空串、`epoch` 没代是空串 —— 一律照写。
// 前端与 CLI 按这份形状解包，少一个键它们就得写第二套兜底。
//
// **只写这七个键，别的一概不碰**：快照的既有键（`topent` / `topentSession` / `zone` /
// `fglserver` … 那些 `SessionBrief` 里**没有**的字段）语义一字不动（H2）——
// `topentSession` 有它自己的含义（本会话显式给的 TOPENT），不是 `ent` 的备胎。
func identityKeysOf(resp map[string]any, b SessionBrief) {
	resp["ref"] = b.Ref
	resp["display"] = b.Display
	resp["name"] = b.Name
	resp["ent"] = b.Ent
	resp["n"] = b.N
	resp["kind"] = b.Kind
	resp["epoch"] = b.Epoch
}

func (s *Server) hSnapshot(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	cur := sess.Cur()
	resp := map[string]any{
		"ok": true,
		"id": sess.ID, "module": sess.Module, "prog": sess.Prog, "runProg": sess.RunProg,
		"env":   sess.EnvNameNow(),
		"state": string(sess.State()), "stop": cur,
		"zone":            sess.ZoneNow(),
		"started":         sess.Started(),
		"breakpoints":     sess.Breakpoints(),
		"holdingSeconds":  sess.HoldingSeconds(),
		"watchdogSeconds": s.cfg.WatchdogSeconds,
		"topent":          sess.TopentOverride(),
		// 本会话显式给的 TOPENT(启动 --ent / 面板输入)。**不再是"配置级"** ——
		// 环境配置里的企业编号已移除,所以它与 topent(会话内覆盖)只在"启动给 vs 中途改"上有别。
		"topentSession": sess.TopentSession(),
		"topentShell":   sess.TopentShell(),
		// 本次重放跑完后读回来的响应原文(非重放会话为空)。
		// 以前想看它只能"停在收尾断点再 continue 到 exit"从 stdout 抓,很绕。
		"replayResponse": sess.ReplayResponse(),
		// 运行态:静默多久 / 已跑多久。配合 why 判断"是在等用户还是在空转"。
		"silentSeconds": sess.SilentSeconds(),
		// 谁在驾驶 + 正在执行哪条命令(界面上的"AI 正在执行 continue(已 12s)")
		"mode": sess.Mode(),
		// 这一轮作业界面画到哪儿:机器可读的原始值 + 一句人话结论。
		// 界面上没有它的时候,"放行后什么都没弹"只能靠人去翻配置猜(实测踩过)。
		"fglserver":    sess.FGLServer(),
		"screenTarget": ScreenTarget(sess.FGLServer()),
		// product: 这一轮的产品别。GP 没有 ENT/区域/FGLSERVER 那几层,CLI 的身份行与提示按它分档。

	}
	// 身份那一组键（C5.1 / C5.3 / C5.9）：从 `SessionBrief` 那一份投影搬过来，**不在这里手抄**
	// （C5.17–C5.19）。既有键一个不动、语义一字不改（H2）—— 这七个都是**新增**键。
	identityKeysOf(resp, s.mgr.briefOf(sess))
	if fl := sess.Inflight(); fl != nil {
		resp["inflight"] = fl
	}
	if since := sess.RunningSince(); !since.IsZero() {
		resp["runningSince"] = since
	}
	if cur.WaitingForUser {
		resp["waitingForUser"] = true
		resp["waitingKind"] = cur.WaitingKind
	}
	writeJSON(w, 200, resp)
}

func (s *Server) hQuit(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOfWriteCap(w, r, CapOwn)
	if sess == nil {
		return
	}
	// 「结束调试」只结束本轮运行,宿主会话保留(idle);再次启动免重新登录。
	// 卡死降级整体断开时(StateExit)才移除会话。
	if err := sess.EndRun(); err != nil {
		fail(w, 500, err)
		return
	}
	s.emitAction(r, sess, "session.quit", "结束本轮调试")
	kept := sess.State() != StateExit
	if !kept {
		s.mgr.RemoveAs(sess.ID, "exit")
	}
	writeJSON(w, 200, map[string]any{"ok": true, "kept": kept, "state": "idle"})
}

// ---------- 会话管理(单一常驻会话的切换/重启/结束) ----------

// launchCfg 组装"这次启动要连哪个环境"的生效配置:ssh(环境名)/ zone 覆盖 + 单轮 FGLSERVER 覆盖。
//
// 按名点名是**权威**的:点到的环境不存在就报错列出可用名,绝不静默回落默认连接 ——
// 同一台机器上有多个环境(同库不同企业)时,"静默回落"与"上错了号"在输出上完全
// 无法区分(见 manager.prepareSession 的环境判据)。
//
// fglserver 用**指针**而不是字符串:空串在 JSON 里与"这一轮没给"无法区分,而
// "这一轮把画面交回 T100 自动设置"恰恰是最需要的那个动作(见 ScreenTarget)。
// 覆盖只落在克隆出来的本轮配置上,s.cfg 一字不动 —— 否则一次前端实验的接线
// 会留在内存里,后面每一轮(哪怕只是普通启动)都还带着它。
func (s *Server) launchCfg(sshName, zone string, fglserver *string) (*Config, error) {
	if sshName == "" && zone == "" && fglserver == nil {
		return s.cfg, nil
	}
	cfg := s.cfg
	if sshName != "" {
		c2 := s.cfg.CloneEnv(sshName)
		if c2 == nil {
			names := make([]string, 0, len(s.cfg.SSHs))
			for _, e := range s.cfg.SSHs {
				names = append(names, e.Name)
			}
			if len(names) == 0 {
				return nil, fmt.Errorf("环境 %q 不存在(配置里没有任何环境,运行 tt env list 查看)", sshName)
			}
			return nil, fmt.Errorf("环境 %q 不存在;可用环境: %s", sshName, strings.Join(names, "、"))
		}
		cfg = c2
	}
	c2 := *cfg.CloneWithZone(zone)
	if fglserver != nil {
		c2.FGLServer = strings.TrimSpace(*fglserver)
	}
	return &c2, nil
}

// envCfgFor 取指定环境的生效配置(CloneEnv);若与当前生效目标同名(或未配置 envs 列表)则用当前配置快照
func (s *Server) envCfgFor(name string) *Config {
	if c := s.cfg.CloneEnv(name); c != nil {
		return c
	}
	if name != "" && s.cfg.EnvName() == name {
		cc := *s.cfg
		return &cc
	}
	return nil
}

// selectEnv 把当前环境切到 cfg 所属环境并热生效(仅内存,不写 config.json)。
// 当前环境是运行时概念:由会话切换/CLI env 命令确定,服务重启后回到 sshs 首条。
func (s *Server) selectEnv(cfg *Config) {
	if cfg != nil {
		*s.cfg = *cfg
	}
}

// bootToIdle 异步登录新会话到 idle(切换/重启用),失败终结并移除
func (s *Server) bootToIdle(ns *Session, ctx context.Context) {
	if err := ns.BootIdle(ctx); err != nil {
		log.Printf("[debug] 会话 %s(%s) 登录失败: %v", ns.ID, ns.EnvName(), err)
		s.mgr.emit(Event{Type: "log", SessionID: ns.ID, Text: "会话连接失败: " + err.Error()})
		ns.ForceExit()
		s.mgr.RemoveAs(ns.ID, "exit")
	}
}

// hSessionClose 「结束会话」：按归属把这条 SSH 会话收干净（W3）。
//
// 顺序不能反（systemd 的 stop sequence，H5 幂等）：
//
//	① 先断调试（closeNow：interrupt → 等到 stopped → quit → forceExit）。
//	② 最后从池里移除（内部广播 session.removed，reason=close）。
func (s *Server) hSessionClose(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOfWriteCap(w, r, CapOwn)
	if sess == nil {
		return
	}
	if err := sess.Close(); err != nil {
		fail(w, 500, err)
		return
	}
	s.emitAction(r, sess, "session.close", "关闭会话")
	s.mgr.RemoveAs(sess.ID, "close")
	writeJSON(w, 200, map[string]any{"ok": true, "state": "exit"})
}

// hSessionRestart 「重启会话」:断开并重连同环境,回到 idle 待启动。
// ent = 本次要用的企业编号(同 hSessionSwitch;空 = 沿用登录 shell 的 $TOPENT)。
func (s *Server) hSessionRestart(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOfWriteCap(w, r, CapOwn)
	if sess == nil {
		return
	}
	// 请求体可选:不传就是"原样重启"(沿用该会话当前的企业编号)。
	var req struct {
		Ent string `json:"ent"`
	}
	_ = readBody(w, r, &req)
	env := sess.EnvName()
	id := sess.ID
	// 别名要在收口**之前**取下来：重启是"同一条逻辑会话重新登录"（F15 的前提正是
	// "环境与序号都对、只是换了代"），所以旧会话从池里消失、别名当场释放的同时，
	// 把同一个名字重新登记给新会话 —— 调用方手上的别名与 ref 因此继续有效（C3.7b / Q19）。
	name := sess.Name()

	// 先收口当前会话(运行中会先中断/quit),再重建连接
	if err := sess.Close(); err != nil {
		fail(w, 500, err)
		return
	}
	s.mgr.RemoveAs(id, "restart")
	cfg := s.envCfgFor(env)
	if cfg == nil {
		// 原环境已被删除 ⇒ **不再退回某个"默认"环境**(默认环境这个概念已移除):
		// 那会把会话悄悄连到另一台机器上。直接报错,让人点名一个存在的环境。
		fail(w, 500, fmt.Errorf("环境 %q 已被删除,无法重启;请用 /api/sessions/switch 点名一个存在的环境", env))
		return
	}
	if ent := strings.TrimSpace(req.Ent); ent != "" {
		cfg.Topent = host.EntValue(ent)
	}
	// 重启 = 同一条逻辑会话重新登录（名字跟着继承）。
	ns, err := s.mgr.CreateSessionOn(cfg, name)
	if err != nil {
		var taken *ErrNameTaken
		if errors.As(err, &taken) {
			s.writeNameTaken(w, taken)
			return
		}
		fail(w, 502, err)
		return
	}
	// 与 hLaunch / hSessionSwitch / hWSLogDebug 同一条规则：模式由**发起方**定。
	// 这里也漏过（同一类 bug，见 hSessionSwitch 那段注释）：重启出来的会话若保留
	// NewSession 的初始值，业主授权（aiTakeover）时它就是 collab，随后"人"的写操作全被拒。
	ns.SetMode(launchModeFor(r))
	s.mgr.emit(Event{Type: "log", Text: fmt.Sprintf("重启会话:重新连接 %s …", cfg.EnvName())})
	s.emitAction(r, ns, "session.restart", "重新开始调试")
	// 用会话自己的生命周期，而不是 HTTP 请求的 ctx：请求一返回 ctx 就被取消，
	// 而登录（尤其 GP 的实例菜单 + 环境铺开要好几秒）还在跑 —— 那会让会话在
	// loading 里被静默掐断，表现就是"建完几秒变 exit、且没有任何失败日志"。
	go s.bootToIdle(ns, context.Background())
	writeJSON(w, 200, map[string]any{"ok": true, "sessionId": ns.ID, "env": cfg.EnvName(), "state": "loading"})
}

// hTopent 空闲态重新设置 TOPENT(会话内,立即下发到 shell)。
// 仅允许 idle(宿主 shell 就绪、无调试运行);值不限数字/文本(导出为环境变量),
// 服务端剔除两侧空白;留空 = 清除手动覆盖并重新下发配置默认。
func (s *Server) hTopent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value string `json:"value"`
	}
	if !readBody(w, r, &req) {
		return
	}
	s.applyTopent(w, r, req.Value)
}

// hSessionEnt 是规格里的 `POST /api/sessions/{id}/ent`：与 `/topent` 同一件事、同一条规矩，
// 只是请求体按规格写成 `{"value":"99"}` / `{"clear":true}`。
//
// 为什么两个端点都留：`/topent` 是既有调用方（CLI 与前端面板）在用的名字，
// `/ent` 是规格名。两者的**行为**收在同一个 `applyTopent` 里 —— 两条路各写一份必然漂，
// 而漂的那一处恰好是"到底设上了没有"。
func (s *Server) hSessionEnt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value string `json:"value"`
		Clear bool   `json:"clear"`
	}
	if !readBody(w, r, &req) {
		return
	}
	s.applyTopent(w, r, entValueOf(req.Value, req.Clear))
}

// entValueOf 把规格里的两种写法归一：`{"value":"99"}` 与 `{"clear":true}`。
//
// 单独一个函数是为了能钉住它：两个端点（/ent 与 /topent）共用同一份实现，
// 而"清除"在这条链路上等于空值（见 SetTopent 的注释：空值 = 回到环境默认，不是 unset）。
func entValueOf(value string, clear bool) string {
	if clear {
		return ""
	}
	return value
}

// applyTopent 是设置/清除会话级企业编号的唯一实现。
func (s *Server) applyTopent(w http.ResponseWriter, r *http.Request, value string) {
	req := struct{ Value string }{Value: value}
	req.Value = trimTopent(req.Value)
	// 设 ENT 属于**编排**（CapSetup）：平台子进程启动时就要设它。拍在交互式那一类里
	// 会让 record 会话连企业编号都设不上 —— W7 分类正是为这种差别存在的。
	sess := s.sessOfWriteCap(w, r, CapSetup)
	if sess == nil {
		return
	}
	if st := sess.State(); st != StateIdle {
		fail(w, 409, fmt.Errorf("仅会话空闲时可设置 TOPENT(当前 %s),请先结束当前调试", st))
		return
	}
	if err := sess.SetTopent(req.Value); err != nil {
		fail(w, 500, err)
		return
	}
	txt := "设置 TOPENT"
	if req.Value == "" {
		txt += "(清除)"
	} else {
		txt += "=" + req.Value
	}
	s.emitAction(r, sess, "session.topent", txt)
	writeJSON(w, 200, map[string]any{"ok": true, "topent": req.Value})
}

// trimTopent TOPENT 归一:剔除两侧空白(值不限数字/文本);全空白归一为空(清除)
func trimTopent(v string) string {
	return strings.TrimSpace(v)
}

// hSessionSwitch 「连到该环境」:连上目标环境(仅内存)并按该环境重连到 idle。
// 若已是目标环境会话则幂等返回(不改动正在进行的调试)。
//
// ent = 本次要用的**企业编号**(界面面板填的那个)。环境配置里已经没有企业编号了
// (hosts.sshs[].topent 已移除),所以 ENT 只能随这次请求进来;留空 = 不覆盖,
// 沿用登录 shell 自己的 $TOPENT(见 Session.bootHost 2.5)。
//
// **定稿（Q26 / Q27 / Q28 + 契约 C7.1-2）**：这个端点**不退役**，正名为「**连到该环境**」——
// 与 `start` 同类（选环境并连上去），**不是**"把一个会话切到另一个环境"；「连到该环境」就
// **收掉已在跑的那条**（Q27：我只想连到那个环境，收掉旧的才对），并**不新增 `--session` 参数**
// （Q28：多会话时谁给目标靠调用方侧的固有落位 —— 唯一就落、多条报 409 并列候选，Q12）。
//
// 服务端不再用 `Manager.Current()` 找"当前会话"。要收掉的那一条**由请求体可选给**
// （`session`，服务端的原始句柄 `{id}`）；落位规则照现状写死（WP4b 的安全状态即最终形状）：
//   - 给了 → 只收那一条（找不到就 404 + `SESSION_NOT_FOUND`）；
//   - 没给且池里**恰好一条** → 收它（C8.1"只在多会话时收紧"：唯一是确定性的）；
//   - 没给且池里 **>=2 条** → **409 + `SESSION_AMBIGUOUS` + 候选清单**（不许替调用方挑一条 —— 这正是"随机落一条却看起来成功"的病根）。
func (s *Server) hSessionSwitch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Env string `json:"env"`
		Ent string `json:"ent"`
		// Name 是**这条新会话的别名**（可选）：创建即命名。
		//
		// 为什么需要它（多会话）：那几步要能点名"我这条"会话，而点名要有名字。
		// 从前只能在 `debug start` 那一步用 `--name` 给，于是 `env`/`topent`/`exec` 这几步
		// 只能裸奔 —— 多会话下要么撞歧义，要么落到别人的会话上。
		Name string `json:"name"`
		// Session 是**要收掉的那条会话**的服务端原始句柄（可选，见上面那段定稿）。
		// 只认 `{id}`（`internal/target/view.go` 的 Session.ID），不认 `ref` / `display`
		// —— C5.4 明写"路径与句柄只认 id"，选择器的解析住在 CLI 的目标层里。
		Session string `json:"session"`
	}
	if !readBody(w, r, &req) {
		return
	}
	cfg := s.envCfgFor(req.Env)
	if cfg == nil {
		fail(w, 404, fmt.Errorf("环境 %q 不存在(请在设置中添加)", req.Env))
		return
	}
	if ent := strings.TrimSpace(req.Ent); ent != "" {
		cfg.Topent = host.EntValue(ent)
	}
	// 目标先落位,再判"谁能写":闸门必须按**要收掉的那一条**判,不能按池子里随便一条
	// (从前那一处 `Manager.Current()` 就是这么判的 —— 多会话下按的是随机一条)。
	//
	// 走**冷启动档**（mustTargetOrEmpty）而不是 mustTarget：池子空、也没点名时，
	// 这里没有"要收掉的那一条"，但下面有"要新建的那一条" —— 那正是「创建会话」。
	// 从前这一处用的是 mustTarget，于是 `tt debug env <环境>` 在会话池为空时**必失败**：
	// 用户只能先"起个作业"来凑一条会话，顺序被彻底反过来（真机踩到过）。
	cur, ok := s.mustTargetOrEmpty(w, r, req.Session, "切换会话")
	if !ok {
		return
	}
	if s.cfg.EnvName() != req.Env {
		s.selectEnv(cfg)
	}
	// 别名：取值纪律在这里判（与 `debug start` 同一套 F19），冲突由 CreateSessionOn 兜底。
	name := strings.TrimSpace(req.Name)
	if name != "" {
		if f := target.ValidateAlias(name); f != nil {
			writeCandidateFailure(w, http.StatusBadRequest, f.Code, f.Message, nil)
			return
		}
	}
	if cur != nil {
		if cur.sameEnvAs(cfg) {
			// 同环境：已经满足"这个环境里有一条能用的会话"，但**名字**要按这次的请求对齐 ——
			// 那条链路（`debug env --name <别名>` → `topent --session <别名>` → `start`）
			// 就靠这一步把"我这条"建出来：
			//   - 没点名 / 同名  → 幂等返回；
			//   - 它此前无名      → 把名字登记给它（C2.5b 禁的是**改名**，不是首次命名）；
			//   - 它叫别的名字    → 那就是改名，不允许：空闲时收口重建，忙就拒。
			switch {
			case name == "" || cur.Name() == name:
				writeJSON(w, 200, map[string]any{"ok": true, "sessionId": cur.ID, "env": req.Env, "state": string(cur.State())})
				return
			case cur.Name() == "":
				if err := s.mgr.NameSession(cur, name); err != nil {
					var taken *ErrNameTaken
					if errors.As(err, &taken) {
						s.writeNameTaken(w, taken)
						return
					}
					fail(w, 409, err)
					return
				}
				writeJSON(w, 200, map[string]any{"ok": true, "sessionId": cur.ID, "env": req.Env, "state": string(cur.State())})
				return
			default:
				if cur.State() != StateIdle {
					fail(w, 409, fmt.Errorf("这个环境里已有一条叫 %q 的会话，且它不在空闲（%s）；"+
						"先 tt debug quit --session %s，或换一个 --name", cur.Name(), cur.State(), cur.Name()))
					return
				}
				s.mgr.emit(Event{Type: "log", SessionID: cur.ID,
					Text: fmt.Sprintf("切换会话到 %s：收掉旧的命名会话（%s），按 %s 重建", req.Env, cur.Name(), name)})
				_ = cur.Close()
				s.mgr.RemoveAs(cur.ID, "switch")
			}
		} else {
			s.mgr.emit(Event{Type: "log", SessionID: cur.ID, Text: fmt.Sprintf("切换会话到 %s,断开当前会话", req.Env)})
			_ = cur.Close()
			s.mgr.RemoveAs(cur.ID, "switch")
		}
	}
	// 「连到该环境」= 新建一条会话：别名**不继承**（C3.7b / Q19：别名随旧会话消失即释放；
	// 要名字就在起会话时再给一次 —— 这条与 hSessionRestart 的"继承"刻意不同，后者是
	// 同一条逻辑会话重新登录）。冲突由 CreateSessionOn 在"判定 → 登记"之间兜住（F18 竞态）。
	ns, err := s.mgr.CreateSessionOn(cfg, name)
	if err != nil {
		var taken *ErrNameTaken
		if errors.As(err, &taken) {
			s.writeNameTaken(w, taken)
			return
		}
		fail(w, 502, err)
		return
	}
	// 模式由**发起方**决定 —— 与 hLaunch / hWSLogDebug 同一条规则。
	//
	// 漏了这一句的后果（实测 2026-10-16，用户报"点了没反应"）：切换环境新建出来的会话
	// 保留 NewSession 的初始模式，而业主授权（config.aiTakeover=true）时那个初始值是
	// **collab** —— 于是流水线下一步（`debug topent`）以"人"的身份被"协作模式只有 AI 能写"
	// 拒掉，操作静默失败。launchModeFor 的注释早写过"两条启动路径必须都过这里"，
	// 这里是漏掉的第三条（/api/sessions/switch）。
	ns.SetMode(launchModeFor(r))
	s.mgr.emit(Event{Type: "log", Text: fmt.Sprintf("切换会话:正在连接 %s …", req.Env)})
	s.emitAction(r, ns, "session.switch", "切换环境到 "+req.Env)
	// 同 hSessionRestart：登录跑在请求之后，不能用 r.Context()；否则响应一返回
	// 登录就被取消，会话静默变 exit，且没有任何失败日志。
	go s.bootToIdle(ns, context.Background())
	writeJSON(w, 200, map[string]any{"ok": true, "sessionId": ns.ID, "env": req.Env, "state": "loading"})
}

// hSessionCreate 再建一条**空闲会话**（`POST /api/sessions/new`）。
//
// 与 `/api/sessions/switch` 的分工必须说清，否则两条路会互相顶掉：
//
//   - switch 是"**连到这个环境**"：池里已有一条同环境的就复用；叫别的名字且空闲时
//     **收掉旧的按新名重建**（C3.7b / Q19 —— 那是"换目标"的语义）；
//   - 这条是"**再给我一条**"：池里有十条也一条都不动，新的那条自己登录、自己待命。
//     多会话（几个 Agent 同时用、同一个环境并发调试）要的就是这条 —— 用 switch 建第二条
//     会把第一条收掉，于是"我建了新的、旧的却没了"，而那正是最难发现的一类错。
//
// 幂等键是**名字**：同环境已有同名会话 → 原样返回它（不重连、不收口）。不给名字则必然新建
// —— "都没有名字"不构成"是同一条"的证据，猜错的代价是两条会话落在不同作业上。
//
// ENT 必填（requireExplicitEnt）：与启动作业同一条纪律，不许静默沿用登录 shell 的 $TOPENT。
func (s *Server) hSessionCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Env  string `json:"env"`
		Ent  string `json:"ent"`
		Name string `json:"name"`
	}
	if !readBody(w, r, &req) {
		return
	}
	env := strings.TrimSpace(req.Env)
	if env == "" {
		fail(w, 400, fmt.Errorf("需要 env：新会话落在哪个环境（tt env list 可看全部）"))
		return
	}

	cfg := s.envCfgFor(env)
	if cfg == nil {
		fail(w, 404, fmt.Errorf("环境 %q 不存在(请在设置中添加)", env))
		return
	}
	// `--ent` 只改**这一次**的配置（envCfgFor 给的是克隆，写它不会把这个企业留给别的会话）。
	if e := strings.TrimSpace(req.Ent); e != "" {
		cfg.Topent = host.EntValue(e)
	}
	if err := requireExplicitEnt(cfg, nil); err != nil {
		var f *target.Failure
		if errors.As(err, &f) {
			writeCandidateFailure(w, http.StatusBadRequest, f.Code, f.Message, nil)
			return
		}
		fail(w, 400, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name != "" {
		if f := target.ValidateAlias(name); f != nil {
			writeCandidateFailure(w, http.StatusBadRequest, f.Code, f.Message, nil)
			return
		}
		// 同名已在 → 就是那一条（幂等），**不收口任何别的会话**（与 switch 的差别）。
		for _, b := range s.mgr.Snapshot() {
			if b.Env != env || b.Name == "" || !target.SameAlias(b.Name, name) {
				continue
			}
			writeJSON(w, 200, map[string]any{
				"ok": true, "sessionId": b.ID, "ref": b.Ref, "env": env,
				"name": b.Name, "state": b.State, "reused": true,
			})
			return
		}
	}
	// 写闸门：与其它"能动池子"的端点同一道（nil 目标 = 这一步不保护某一条）。
	if !s.sessOfWriteGlobal(w, r, nil) {
		return
	}
	ns, err := s.mgr.CreateSessionOn(cfg, name)
	if err != nil {
		var taken *ErrNameTaken
		if errors.As(err, &taken) {
			s.writeNameTaken(w, taken)
			return
		}
		fail(w, 502, err)
		return
	}
	// 模式由**发起方**决定（与 hLaunch / hSessionSwitch 同一条规则，见 hSessionSwitch 里那段注释）。
	ns.SetMode(launchModeFor(r))
	ref := refOf(ns.EnvName(), ns.Seq())
	s.mgr.emit(Event{Type: "log", SessionID: ns.ID, Text: fmt.Sprintf("新建会话:正在连接 %s …", env)})
	s.emitAction(r, ns, "session.new", "新建会话到 "+env)
	// 同 hSessionRestart：GP 登录（实例菜单 + 环境铺开）慢于 HTTP 请求，
	// 必须用进程生命周期，不能用 r.Context()。
	go s.bootToIdle(ns, context.Background())
	writeJSON(w, 200, map[string]any{
		"ok": true, "sessionId": ns.ID, "ref": ref, "env": env,
		"name": name, "state": "loading", "reused": false,
	})
}

// ---------- 断点 ----------

// hBPList 列出断点 —— **带当下**的命中计数（A36）。
//
// 修前这里是 `writeJSON(… sess.Breakpoints())`，一个字节都不重读内核；而账本只在
// "改断点"那刻回灌（见 hRaw 的 `IsBreakpointCmd`）⇒ `命中 N 次` 反映的是**上次改断点时的
// 计数**。2026-10-18 真机就是这么被撞破的：`bp add` 正常、`info breakpoints` 原文有
// `breakpoint already hit 1 time`、点击后确实停站，可 `bp list` 前后都写 `命中 0 次`。
//
// 现在：停站态先回灌再返回；运行态**不发命令**（A34：只允许 interrupt），并如实带上
// `fresh:false` + 一句"可能不是最新 / 先 interrupt"的说明 —— 不许拿账本冒充最新。
//
// 回灌发的是一条**读**命令（`info breakpoints`），不改断点集合，所以不需要过写闸门。
func (s *Server) hBPList(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	bps, fresh, note := sess.BreakpointsNow()
	resp := map[string]any{
		"ok":          true,
		"breakpoints": bps,
		"fresh":       fresh, // 这份账本是否刚从内核读回来
		"state":       string(sess.State()),
	}
	if note != "" {
		// 只在不新鲜时出现：让调用方（CLI/前端/AI）能照做，而不是以为数字是当下的
		resp["note"] = note
	}
	writeJSON(w, 200, resp)
}

func (s *Server) hBPAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Location string `json:"location"`         // 行号 / 函数名 / file:line
		Cond     string `json:"cond,omitempty"`   // A20:条件表达式(空 = 无条件)
		Ignore   int    `json:"ignore,omitempty"` // A20:忽略前 N 次命中(0 = 不忽略)
		Kind     string `json:"kind,omitempty"`   // A20:"break" | "watch"(空 = "break")
	}
	if !readBody(w, r, &req) {
		return
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	opts := BreakpointOpts{Location: req.Location, Cond: req.Cond, Ignore: req.Ignore, Kind: req.Kind}
	// 校验放在**取会话之后、发命令之前**:错误要 400 说清是哪个字段,
	// 而且这条路径不发任何字节(生成命令的函数是纯函数,拒绝时返回 nil)。
	if err := opts.Validate(); err != nil {
		fail(w, 400, err)
		return
	}
	log.Printf("[bp-add] session=%s loc=%s kind=%s cond=%q ignore=%d",
		sess.ID, opts.Location, opts.kind(), opts.Cond, opts.Ignore)
	bp, err := sess.BreakWith(opts)
	if err != nil {
		fail(w, 400, err)
		return
	}
	log.Printf("[bp-add] result #%d %s:%d", bp.Num, bp.File, bp.Line)
	// 用 fgldb 落定后的实际位置(它会把非可执行行上的断点下移到下一条语句)
	s.emitAction(r, sess, "bp.add", fmt.Sprintf("下断点 %s:%d", bp.File, bp.Line))
	writeJSON(w, 200, map[string]any{"ok": true, "breakpoint": bp})
}

func (s *Server) hBPDel(w http.ResponseWriter, r *http.Request) {
	num, err := strconv.Atoi(r.PathValue("num"))
	if err != nil {
		fail(w, 400, fmt.Errorf("断点编号无效"))
		return
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	log.Println("[bp-del] session=" + sess.ID + " num=" + strconv.Itoa(num))
	if err := sess.DeleteBreakpoint(num); err != nil {
		fail(w, 400, err)
		return
	}
	s.emitAction(r, sess, "bp.del", fmt.Sprintf("删除断点 %d", num))
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---------- 控制 / 求值 ----------

func (s *Server) hControl(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // continue|run|next|step|finish|until|interrupt
		Arg    string `json:"arg,omitempty"`
		// Wait 软等待秒数(仅步进类):到点若程序仍在跑就返回 softTimeout。
		// 步进可能撞上交互语句而长时间等用户,需要这条退路;continue/run 本来就立即返回。
		Wait int `json:"wait"`
	}
	if !readBody(w, r, &req) {
		return
	}
	// 闸门放在读 body 之后:这样归因文案能带上动作细节(continue / step 12 …)
	txt := req.Action
	if req.Arg != "" {
		txt += " " + req.Arg
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	softWait := time.Duration(req.Wait) * time.Second
	if softWait < 0 {
		softWait = 0
	}
	var stop *StopInfo
	var soft bool
	var err error
	switch req.Action {
	case "continue":
		_, err = sess.Continue()
	case "run":
		_, err = sess.Run()
	case "next", "step", "finish":
		stop, soft, err = sess.StepSoft(req.Action, softWait)
	case "until":
		loc := req.Arg
		if loc == "" {
			stop, soft, err = sess.StepSoft("until", softWait)
		} else {
			stop, soft, err = sess.StepSoft("until "+loc, softWait)
		}
	case "interrupt":
		err = sess.Interrupt()
	default:
		fail(w, 400, fmt.Errorf("未知 action: %s", req.Action))
		return
	}
	if err != nil {
		fail(w, 409, err)
		return
	}
	s.emitAction(r, sess, "control."+req.Action, txt)
	resp := map[string]any{"ok": true, "state": string(sess.State())}
	if stop != nil {
		resp["stop"] = stop
	}
	if soft {
		resp["softTimeout"] = true
		resp["silentSeconds"] = sess.SilentSeconds()
		resp["hint"] = "步进到点仍未停站:程序仍在运行(大概率停在了交互界面上等用户)。" +
			"用 why 判断,或 wait 等它停下来"
	}
	writeJSON(w, 200, resp)
}

func (s *Server) hPrint(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	var req struct {
		Expr string `json:"expr"`
	}
	if !readBody(w, r, &req) {
		return
	}
	v, err := sess.Print(req.Expr)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "value": v})
}

func (s *Server) hWhere(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	frames, err := sess.Where()
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "frames": frames})
}

func (s *Server) hRaw(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
		Timeout int    `json:"timeout"` // 等待命令完成的秒数(0=默认30);continue/until 等长命令可调大
		// Wait 软等待秒数:到点若程序仍在跑就直接返回(softTimeout),**不发 SIGINT、不取消命令**。
		// 与 timeout 的区别是硬/软:timeout 到点会发 \x03 探测并报错,wait 到点只是"不再等"。
		Wait int `json:"wait"`
	}
	if !readBody(w, r, &req) {
		return
	}
	cmd := strings.TrimSpace(req.Command)
	if cmd == "" || strings.ContainsAny(cmd, "\r\n") {
		fail(w, 400, fmt.Errorf("命令不能为空或含换行"))
		return
	}
	lower := strings.ToLower(cmd)
	if lower == "quit" || strings.HasPrefix(lower, "quit ") {
		fail(w, 400, fmt.Errorf("请使用会话控制接口执行 quit"))
		return
	}
	// 闸门放在入参与合法性校验之后:被拒的命令不该在时间线上留痕
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	// run / 入口停站时的 continue 有会话级语义,不能当裸命令透传:
	// fgldb 在 run 之前不接受 continue(会回 "The program is not being run"),
	// 而会话层的 Run()/Continue() 对这个状态有兜底(Continue 等效成 Run)。
	// 不分流的话**协作模式会死锁**:人按不了「继续」(模式闸门),AI 又发不出 run,
	// 谁都启动不了程序。Web 界面走的是 /control,两边落到同一实现。
	if lower == "run" || (lower == "continue" && !sess.Started()) {
		var er error
		if lower == "run" {
			_, er = sess.Run()
		} else {
			_, er = sess.Continue()
		}
		if er != nil {
			fail(w, 400, er)
			return
		}
		writeJSON(w, 200, map[string]any{
			"ok": true, "lines": []string{cmd},
			"state": string(sess.State()), "started": sess.Started(),
			"hint": "程序已放行,用 tt debug wait --for stopped 等它停下来",
		})
		return
	}
	// 入口态(程序还没 run)的 next/step/until/finish:fgldb 在 run 之前**不受理**这些命令,
	// 只回一句 "The program is not being run."。以前这里直接把它们当裸命令透传,后果是
	// 会话被卡死:execOpt 见"放行类命令"就把状态乐观翻成 running 且无回滚,之后
	// interrupt 因"未启动"被拒、exec 因"运行中"被拒、EndRun 的中断同样失败 ——
	// 只能整体断开会话重来。真机踩过。
	//
	// 会话语义上这几条本来是有办法的:stepFromEntry(tbreak main + run)。
	// /control 一直走的就是它,只有 /raw 漏了 —— 这里补齐,两条路行为对齐。
	if !sess.Started() && lower != "run" && lower != "continue" && IsResumeCmd(cmd) {
		soft := time.Duration(req.Wait) * time.Second
		if soft < 0 {
			soft = 0
		}
		st, completed, er := sess.StepSoft(cmd, soft)
		if er != nil {
			fail(w, 400, er)
			return
		}
		resp := map[string]any{
			"ok": true, "lines": []string{cmd},
			"state": string(sess.State()), "started": sess.Started(),
			"hint": "入口停站没有调用栈,step/next 会先 tbreak main 把程序放起来;" +
				"用 tt debug wait --for stopped 等它停到 MAIN",
		}
		if !completed && st == nil {
			resp["softTimeout"] = true
			resp["hint"] = "程序已放行但还没停站;用 tt debug wait --for stopped 等它停下来"
		} else if st != nil {
			resp["stop"] = st
		}
		writeJSON(w, 200, resp)
		return
	}
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	if timeout > 600 {
		timeout = 600
	}
	soft := time.Duration(req.Wait) * time.Second
	if soft < 0 {
		soft = 0
	}
	// 软等待必须短于硬超时,否则永远轮不到它
	if soft > 0 && soft >= time.Duration(timeout)*time.Second {
		soft = time.Duration(timeout-1) * time.Second
	}
	res, err := sess.RawSoft(cmd, time.Duration(timeout)*time.Second, soft)
	if err != nil {
		fail(w, 400, err)
		return
	}
	// 断点类裸命令:命令本身已经执行完了,但会话的断点缓存还没跟上 —— 见 IsBreakpointCmd
	// 的注释。/raw 走的 kind 是 "other",onLine 里那条按 kind 分派的解析不会进,
	// 所以必须在这里用 info breakpoints 回灌一次,否则这个断点在快照/持久化里都不存在。
	act := "raw"
	if IsBreakpointCmd(cmd) {
		sess.syncBreakpoints()
		act = "bp." + strings.ToLower(strings.Fields(cmd)[0])
	} else if IsBreakpointReadCmd(cmd) && sess.State() == StateStopped {
		// A36 后半：**读**命令也回灌。`info breakpoints` 读的就是内核真相（命中计数、忽略次数
		// 都在里面），可它走的是 kind="other"，onLine 那条按 kind 分派的解析不会进 ——
		// 不回灌的话账本照旧停在"上次改断点那刻"，`bp list` 于是报一个更旧的命中数。
		//
		// 只认这一条命令（`IsBreakpointReadCmd`），**不**把它塞进 `bpCmds`：那一族的语义是
		// "会改动断点集合"，`act` 也会变成 `bp.info` —— 读命令借写命令的语义与事件名是错配，
		// 既有测试 `TestIsBreakpointCmd` 也明确钉着 `info breakpoints` 不属于它。
		// 状态仍是 stopped 才发（运行期 fgldb 对任何调试命令都 400，A34）。
		sess.syncBreakpoints()
	}
	s.emitAction(r, sess, act, cmd)
	resp := map[string]any{"ok": true, "lines": res.Lines}
	// 会话层就截断过(行数/单值超上限)要如实说,否则调用方会把残缺内容当成全部
	if res.Truncated {
		resp["truncated"] = true
		resp["truncReason"] = res.TruncReason
	}
	// 输出够大就落一份**完整**的本地副本并回报路径。
	// 理由:正文有上限(超了只给头部),但完整内容必须留得住 —— 否则想多看一点就得重跑命令,
	// 而重跑会改变现场。落本地后可以随时 grep/整读,零往返。
	// 阈值是为了别给每条小命令都留一个文件。
	if n := len(res.Lines); n > execLogMinLines || linesBytes(res.Lines) > execLogMinBytes {
		// 落在这条会话自己的作用域下（缺陷 X / T5.1）:清理是按同一棵子树做的,
		// 从前的全局 `execlog/` 会让 A 的启动顺手删掉 B 刚落下的大输出副本。
		if p := writeExecLog(s.cfg.DataDir, sess.mirrorScope(), execLogName(cmd), []byte(strings.Join(res.Lines, "\n"))); p != "" {
			resp["localPath"] = p
			resp["totalLines"] = n
		}
	}
	if res.SoftTimeout {
		// 不是错误:命令仍在飞,程序仍在跑
		resp["softTimeout"] = true
		resp["state"] = string(sess.State())
		resp["silentSeconds"] = sess.SilentSeconds()
		resp["hint"] = "程序仍在运行:命令没有被取消,也没有发 SIGINT。" +
			"用 tt debug wait 等停站(或 POST /api/sessions/{id}/wait)," +
			"用 tt debug why 判断它在等用户还是在空转"
	}
	writeJSON(w, 200, resp)
}

// ---------- 上下文查询 ----------

// hWhy 探测「程序此刻到底在干什么」——在等用户操作,还是在空转。
// 把 SKILL.md 里教 AI 人肉做的三步(interrupt → where → 看停在哪一行)收成一次调用。
// 已停站时只做分类,不发任何命令(零副作用);运行中才发 SIGINT。
// body: {"resume": true} —— 默认 true(探测完自动放回运行,免得把等用户的程序挂住)。
//
// 闸门上是**条件写**:只有运行中(会发 SIGINT)才受模式闸门约束;已停站时纯粹是
// 只读分类,放行给任一方 —— 否则人在协作模式下连"现在什么情况"都问不了。
func (s *Server) hWhy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Resume *bool `json:"resume"`
	}
	if !readBody(w, r, &req) {
		return
	}
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	if sess.State() == StateRunning {
		if sess = s.sessOfWrite(w, r); sess == nil {
			return
		}
	}
	resume := true
	if req.Resume != nil {
		resume = *req.Resume
	}
	res, err := sess.Why(resume)
	if err != nil {
		fail(w, 409, err)
		return
	}
	s.emitAction(r, sess, "why", "探测程序在等用户还是空转")
	writeJSON(w, 200, map[string]any{"ok": true, "why": res})
}

// hWait 长轮询:阻塞到会话出现指定事件或超时。
// GET /api/sessions/{id}/wait?for=stopped,exit,dead,watchdog&timeout=300
//
// 手法是「先订阅、后判现状」:订阅动作与读快照之间的窗口由快照本身补齐 ——
// 谓词只看权威现状(State()/Cur()),不看事件历史,所以历史丢失无关紧要,
// **hWait 因此不依赖单调序号**。(Event.Seq 是为 WS 开场补发与前端去重的
// **游标**语义引入的,与此处的**栅栏**判断互不冲突。)
// **超时不是错误**:返回 200 + timedOut:true + 当前快照。
func (s *Server) hWait(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	want := map[string]bool{}
	for _, t := range strings.Split(r.URL.Query().Get("for"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			want[t] = true
		}
	}
	if len(want) == 0 {
		for _, t := range []string{"stopped", "exit", "dead", "watchdog"} {
			want[t] = true
		}
	}
	to := 300 * time.Second
	if v := r.URL.Query().Get("timeout"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			to = time.Duration(n) * time.Second
		}
	}
	if to > time.Hour {
		to = time.Hour
	}

	// 先订阅,后判现状:反向顺序会漏掉"订阅建立前刚发生"的事件。
	//
	// **按会话订阅**（Q11 / 契约 6.6 / C7.2-2 的 E 类修正）：从前是 `Subscribe("wait")`
	// + 客户端自己 `if ev.SessionID != sess.ID { continue }` —— 那是"客户端筛"，
	// 与 Q11 的"发送那头筛"相反，而且别的会话的事件照样要挤进这个通道再被丢掉。
	// 现在范围在**登记订阅**时就声明了（无会话事件也不会发给它，C6.5）。
	ch, cancel := s.mgr.SubscribeScoped("wait", sess.ID)
	defer cancel() // 必须 defer,否则 m.subs 泄漏

	if ev, ok := waitNow(sess, want); ok {
		writeJSON(w, 200, waitResp(sess, ev, false))
		return
	}

	timer := time.NewTimer(to)
	defer timer.Stop()
	for {
		select {
		case <-r.Context().Done():
			return // 客户端断开:不写响应
		case ev := <-ch:
			// 范围已经在发送那头筛过（见上面的 SubscribeScoped）—— 这里不再自己丢事件。
			if waitMatch(ev, want) {
				writeJSON(w, 200, waitResp(sess, &ev, false))
				return
			}
		case <-timer.C:
			writeJSON(w, 200, waitResp(sess, nil, true))
			return
		}
	}
}

// waitMatch 事件是否命中等待条件。
// **注意**:stopped / exit / idle 在事件流里也可能是 `state` 事件的取值,不是独立类型 ——
// 尤其**入口停站只发 `state(stopped)`,不发 `stopped` 事件**(见 startRun),
// 不做这个映射就会漏掉它,tt debug start 会一直等到超时。
func waitMatch(ev Event, want map[string]bool) bool {
	if want[ev.Type] {
		return true
	}
	if ev.Type == "state" && ev.State != "" {
		if want[ev.State] {
			return true
		}
		if (ev.State == string(StateExit) || ev.State == string(StateIdle)) && (want["exit"] || want["dead"]) {
			return true
		}
	}
	return false
}

// waitNow 把"权威现状"映射成一条合成事件;不满足等待条件时返回 false。
func waitNow(sess *Session, want map[string]bool) (*Event, bool) {
	switch sess.State() {
	case StateStopped:
		if want["stopped"] {
			st := sess.Cur()
			return &Event{Type: "stopped", SessionID: sess.ID, Stop: &st}, true
		}
	case StateExit, StateIdle:
		if want["exit"] || want["dead"] {
			return &Event{Type: "exit", SessionID: sess.ID}, true
		}
	}
	return nil, false
}

func waitResp(sess *Session, ev *Event, timedOut bool) map[string]any {
	resp := map[string]any{
		"ok":            true,
		"state":         string(sess.State()),
		"silentSeconds": sess.SilentSeconds(),
	}
	if timedOut {
		resp["timedOut"] = true
	}
	if ev != nil {
		resp["event"] = ev
		if ev.Stop != nil {
			resp["stop"] = ev.Stop
		}
	}
	return resp
}

func (s *Server) hLocals(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	vars, err := sess.Locals()
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "vars": vars})
}

func (s *Server) hGlobals(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	vars, total, err := sess.Globals(limit)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "vars": vars, "total": total})
}

// hLocate POST /api/sessions/{id}/locate {word}:定位函数到源文件与行号。
// 用 fgldb info line [module.]function(BDL 文档语法),仅停站状态可用;Ctrl+点击跳函数用
func (s *Server) hLocate(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	var req struct {
		Word string `json:"word"`
	}
	if !readBody(w, r, &req) {
		return
	}
	word := strings.TrimSpace(req.Word)
	if !reIdent.MatchString(word) {
		fail(w, 400, fmt.Errorf("函数名非法: %q", word))
		return
	}
	file, line, err := sess.InfoLine(word)
	if err != nil {
		var miss *SymbolNotDefinedError
		if errors.As(err, &miss) {
			fail(w, http.StatusNotFound, fmt.Errorf("%s。查当前进程已加载的符号: tg debug exec \"info functions\"；跨程序用 tg outline <作业> 或 tg file grep -r \"FUNCTION <名>\" <模块>/4gl", err.Error()))
			return
		}
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "file": file, "line": line})
}

// hCalibrate 行号校准:停站后检测 fgldb(DVM)行号与磁盘源码的偏移量,
// 供前端把 Monaco 行号对齐到协议流。POST /api/sessions/{id}/calibrate
func (s *Server) hCalibrate(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	offset, err := sess.CalibrateOffset()
	if err != nil {
		fail(w, 400, err)
		return
	}
	log.Printf("[calibrate] session=%s offset=%d", sess.ID, offset)
	s.emitAction(r, sess, "session.calibrate", fmt.Sprintf("行号校准(偏移 %d)", offset))
	writeJSON(w, 200, map[string]any{"ok": true, "offset": offset})
}

func (s *Server) hSources(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	sources, err := sess.Sources()
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "sources": sources})
}

func (s *Server) hFunctions(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	fns, total, err := sess.Functions(limit)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "functions": fns, "total": total})
}

func (s *Server) hAutovars(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "vars": sess.Autovars()})
}

// hAutovarsSet POST /api/sessions/{id}/autovars {auto}:开关停站后自动求值自动变量
// (默认关;开启且已停站时立即补一次求值,结果照常以 autovars 事件推送)
func (s *Server) hAutovarsSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Auto *bool `json:"auto"`
	}
	if !readBody(w, r, &req) || req.Auto == nil {
		return
	}
	verb := "关闭"
	if *req.Auto {
		verb = "开启"
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	sess.SetAutovarsOn(*req.Auto)
	s.emitAction(r, sess, "session.autovars", verb+"自动变量求值")
	writeJSON(w, 200, map[string]any{"ok": true, "auto": *req.Auto})
}

func (s *Server) hFrame(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Num int `json:"num"`
	}
	if !readBody(w, r, &req) {
		return
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	if err := sess.Frame(req.Num); err != nil {
		fail(w, 400, err)
		return
	}
	s.emitAction(r, sess, "session.frame", fmt.Sprintf("选帧 #%d", sess.CurFrame()))
	writeJSON(w, 200, map[string]any{"ok": true, "frame": sess.CurFrame()})
}

func (s *Server) hBPEnabled(w http.ResponseWriter, r *http.Request) {
	num, err := strconv.Atoi(r.PathValue("num"))
	if err != nil {
		fail(w, 400, fmt.Errorf("断点编号无效"))
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if !readBody(w, r, &req) || req.Enabled == nil {
		return
	}
	verb := "停用"
	if *req.Enabled {
		verb = "启用"
	}
	sess := s.sessOfWrite(w, r)
	if sess == nil {
		return
	}
	if err := sess.SetBPEnabled(num, *req.Enabled); err != nil {
		fail(w, 400, err)
		return
	}
	s.emitAction(r, sess, "bp.enable", fmt.Sprintf("%s断点 %d", verb, num))
	writeJSON(w, 200, map[string]any{"ok": true})
}

// ---------- 源码 ----------

// hSourcePreview 会话建立前预取源码(消除启动期空白):GET /api/source-preview?module=&prog=
func (s *Server) hSourcePreview(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// 模块可以不填:不填/填错都按本地字典的 p_link 解析实际模块(与 tg debug source 同一套)。
	module, prog := q.Get("module"), q.Get("prog")
	if prog == "" {
		fail(w, 400, fmt.Errorf("需要 prog 参数(模块可以不填，按 p_link 解析)"))
		return
	}
	sf, err := s.mgr.SourcePreview(module, prog)
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "source": sf})
}

func (s *Server) hSource(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOf(w, r)
	if sess == nil {
		return
	}
	q := r.URL.Query()
	var sf *SourceFile
	var err error
	switch {
	case q.Get("path") != "":
		sf, err = sess.ReadPath(q.Get("path"))
	case q.Get("file") != "":
		sf, err = sess.ResolveSource(q.Get("file"), q.Get("module"))
	default:
		err = fmt.Errorf("需要 file 或 path 参数")
	}
	if err != nil {
		fail(w, 400, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "source": sf})
}

// ---------- 接口日志(wsfa_t) ----------

// 服务测试端点（`POST /api/wstest` → `hWSTest` → `WSTest`）**本批已取消**：
// 仓库主人定性它不是标准接口、是临时测试工具，所以从产品里去掉（不是给它加闸门）。
// 它从前不过模式闸门、不过 `safesql`、对**任意 URL** 发 POST，还往服务器写临时文件
// —— 是那次只读审计里"最严重的单点只靠纪律"。整个实现文件 `wstest.go` 与只测它的
// `wstest_test.go` 一并删掉；`wsURLPrefix`（`/api/status` 的 `wsPrefix` 用它）搬到了
// `wsurl.go`。契约与计划里的相关条目已标「已取消」。

// dialSSH 是那五个"要连服务器"的端点统一走的门（拨当前生效环境）。
//
// 多包一层只为了一件事：**"服务当前没有生效环境"这句要说出来**。
//
// `ApplyDefaultEnv` 在"记住的环境名已经不在配置里"（改名、删除、手改配置）时会故意清空
// 运行时字段 —— 它**不**回落到首条（回落会把会话悄悄切到另一台机器上，那正是要根除的行为）。
// 于是这里拿到的是空地址，原样报出去就是 `dial tcp :0`：接口日志与服务测试一起全红，
// 而会话面板还显示着环境在跑，谁看都不知道该干什么（2026-10-15 实测过这个现场）。
func (s *Server) dialSSH() (*host.SSHConn, error) { return s.dialSSHWith(s.cfg) }

// dialSSHWith 用**指定那份配置**拨 SSH（接口日志按请求里的 env 选环境）。
func (s *Server) dialSSHWith(cfg *Config) (*host.SSHConn, error) {
	conn, err := s.dial(cfg.SSH)
	if err != nil && strings.TrimSpace(cfg.SSH.Host) == "" {
		return nil, fmt.Errorf("%w（服务当前**没有生效环境**：配置里那个环境可能被改名或删掉了 —— "+
			"在左侧「会话」面板重新选一个环境再试）", err)
	}
	return conn, err
}

// wslogCfg 选这次查询用哪份配置：请求点名了 env 就用那个环境的克隆，
// 否则退回服务当前生效环境。
//
// 为什么要有它：统一服务启动时**可能还没有生效环境**（环境一律点名），
// 而界面上的接口日志是按"当前所在的环境"看的 —— 前端把 activeEnv 带上来即可，
// 不必先切一次会话。
func (s *Server) wslogCfg(env string) *Config {
	if name := strings.TrimSpace(env); name != "" {
		if c := s.envCfgFor(name); c != nil {
			return c
		}
	}
	return s.cfg
}

// ---------- 接口日志(wsfa_t) ----------

// wsZone 返回查询用 zone(空配置默认 36)
func (s *Server) wsZone() string {
	if s.cfg.Zone == "" {
		return "36"
	}
	return s.cfg.Zone
}

// entOfQuery 解析查询里的企业编号(空 / 非数字 / 非正数 → 0 = 未指定)。
//
// 接口日志(wsfa_t)本身**不按企业分 schema**(它在系统账号 ds 下),这个值只用来:
// ① 在结果里如实回显"这次选的是哪个企业";② 重放时作为那一轮的 TOPENT。
func entOfQuery(raw string) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// hEnts GET /api/ents?env=<环境名>&refresh=1 —— 企业目录(ENT → 数据库账号)。
//
// 这是**不依赖会话**的那条路:接口日志页与「创建会话」的 ENT 下拉都用它。
// 数据源仍是一张 gzou_t,固定用系统账号 ds 读(见 dbconfig.SystemAccount),
// 取 gzoustus='Y' 的 gzou001,并把 gzou003(该企业的账号/schema)一并返回 ——
// 下拉里能直接看到"这个企业用哪个号",不用等一次查询失败再去反推。
//
// 新鲜度(live / snapshot / stale / fetchedAt)原样带出:缓存可以旧,但"旧"不能藏着。
func (s *Server) hEnts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	cfg := s.cfg
	if name := strings.TrimSpace(q.Get("env")); name != "" {
		c := s.envCfgFor(name)
		if c == nil {
			fail(w, 404, fmt.Errorf("环境 %q 不存在(设置 → 站点管理 里可看全部)", name))
			return
		}
		cfg = c
	}
	opt := EntListOpt{Refresh: q.Get("refresh") == "1", Cached: q.Get("cached") == "1"}
	l, err := s.mgr.ListEnts(cfg, opt)
	if err != nil {
		fail(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{
		"ok": true, "env": l.Env, "zone": l.Zone, "dialect": l.Dialect, "target": l.Target,
		"account": dbconfig.SystemAccount, // 查 gzou_t 用的系统账号(只是说明)
		"ents":    l.Ents, "count": l.Count,
		"source": l.Source, "stale": l.Stale, "fetchedAt": l.FetchedAt,
		"ageSeconds": l.AgeSeconds, "ttlSeconds": l.TTLSeconds, "notes": l.Notes,
	})
}

// hEntsSync POST /api/ents/sync {env} —— 连库读 gzou_t，并把结果**写回 config.json**。
//
// 与 GET /api/ents?refresh=1 的分工：那个只是“预览一次现查”，不改配置；这个是
// 设置页「ENT」页签的「从服务器刷新」—— 写完即成为日常读取的权威清单，
// `--at-list` / tt env ents 从此都是纯本地读取。
func (s *Server) hEntsSync(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Env string `json:"env"`
	}
	if !readBody(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Env)
	if name == "" {
		fail(w, 400, fmt.Errorf("需要 env：要同步哪个环境的企业清单"))
		return
	}
	cfg := s.envCfgFor(name)
	if cfg == nil {
		fail(w, 404, fmt.Errorf("环境 %q 不存在(请在设置中添加)", name))
		return
	}
	if s.cfgPath == "" {
		fail(w, 500, fmt.Errorf("本服务没有绑定配置文件，无法写回 ENT 清单"))
		return
	}
	l, err := s.mgr.ListEnts(cfg, EntListOpt{Refresh: true})
	if err != nil {
		fail(w, 500, err)
		return
	}
	synced := make([]config.SyncedEnt, 0, len(l.Ents))
	for _, m := range l.Ents {
		pwd := ""
		if cfg.DB != nil {
			if p, ok := cfg.DB.PasswordFor(m.Account); ok {
				pwd = p
			}
		}
		synced = append(synced, config.SyncedEnt{Ent: m.Ent, Account: m.Account, Password: pwd})
	}
	at := time.Now()
	added, err := config.SaveEnvEnts(s.cfgPath, name, synced, at)
	if err != nil {
		fail(w, 500, err)
		return
	}
	// 内存里的那份同步更新（GET /api/ents 与会话都读它），并丢掉企业目录缓存 ——
	// 否则下一次读还会命中同步前的那份（mem 以指纹为键，与清单内容无关）。
	entList := make([]config.EntEntry, 0, len(synced))
	for _, e := range synced {
		entList = append(entList, config.EntEntry{Ent: e.Ent, Account: e.Account, Password: e.Password})
	}
	for i := range s.cfg.SSHs {
		if s.cfg.SSHs[i].Name == name {
			s.cfg.SSHs[i].Ents = entList
			s.cfg.SSHs[i].EntsSyncedAt = at.Format(time.RFC3339)
		}
	}
	s.mgr.ForgetEnts(s.cfg)

	writeJSON(w, 200, map[string]any{
		"ok": true, "env": name, "count": len(entList),
		"ents":          l.Ents,
		"addedAccounts": added,
		"syncedAt":      at.Format(time.RFC3339),
	})
}

// hWSLogs GET /api/wslogs?service=&result=&origin=&server=&onlyFail=1&page=&pageSize=&startFrom=&endTo=&env=&ent=
// 时间窗对齐原生 awsq990:startFrom 是 wsfa003(起始时间)下界,endTo 是 wsfa004(结束时间)上界
func (s *Server) hWSLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	cfg := s.wslogCfg(q.Get("env"))
	conn, err := s.dialSSHWith(cfg)
	if err != nil {
		fail(w, 500, fmt.Errorf("SSH 连接失败: %w", err))
		return
	}
	defer conn.Close()
	dbc, err := resolveDBRun(conn, cfg)
	if err != nil {
		fail(w, 500, err)
		return
	}
	items, hasMore, err := listWSLogs(conn, dbc, WSLogFilter{
		Service:   q.Get("service"),
		Job:       q.Get("job"),
		Result:    q.Get("result"),
		Origin:    q.Get("origin"),
		Server:    q.Get("server"),
		PID:       q.Get("pid"),
		OnlyFail:  q.Get("onlyFail") == "1",
		StartFrom: q.Get("startFrom"),
		EndTo:     q.Get("endTo"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		fail(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "items": items, "hasMore": hasMore,
		"acct": wslogAcctFor(s.wslogTopent(nil)), "ent": entOfQuery(q.Get("ent"))})
}

// hWSLogContent GET /api/wslogs/content?rowid=
func (s *Server) hWSLogContent(w http.ResponseWriter, r *http.Request) {
	rowid := r.URL.Query().Get("rowid")
	cfg := s.wslogCfg(r.URL.Query().Get("env"))
	conn, err := s.dialSSHWith(cfg)
	if err != nil {
		fail(w, 500, fmt.Errorf("SSH 连接失败: %w", err))
		return
	}
	defer conn.Close()
	dbc, err := resolveDBRun(conn, cfg)
	if err != nil {
		fail(w, 500, err)
		return
	}
	item, content, err := WSLogDetail(conn, dbc, rowid)
	if err != nil {
		fail(w, 500, err)
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "item": item, "content": content,
		"acct": wslogAcctFor(s.wslogTopent(nil))})
}

// hWSLogDebug POST /api/wslogs/debug {rowid, request?, session?, name?} → 用该日志的报文重放调试。
// request 非空 = 界面里改过的入参(落服务器临时文件后作为入参文件),空 = 用日志原报文。
// name = `--name` 的别名（Bug B）：wsdebug 同样是起会话的命令，别名随这条请求登记。
//
// **本批的定稿（Q4 / C8.2 硬切 + 契约 C7.1-3）**：服务端不再用 `Manager.Current()` 找
// "当前会话"来收口（从前那一处不经模式闸门，是五处里最隐蔽的一处 —— 见计划开头 A–E 表的 B）。
// 要收掉的那一条由请求体可选的 `session`（服务端原始句柄 `{id}`）给出，
// 缺目标/歧义按 `mustTargetOrEmpty` 的口径报 409 + `SESSION_AMBIGUOUS` + 候选清单
// （池空 = 冷启动：这一轮要的那条会话由重放**自己**建）。
func (s *Server) hWSLogDebug(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RowID   string `json:"rowid"`
		Request string `json:"request"`
		SSH     string `json:"ssh"`  // 多 SSH 配置名(环境名);空 = 当前环境
		Zone    string `json:"zone"` // 区域覆盖(31/35/36/39/t);空 = 默认区域
		// Session 是**要收掉的那条会话**的服务端原始句柄（可选）。只认 `{id}`（C5.4）。
		Session string `json:"session"`
		// Name 是 `--name` 给的别名（Bug B / Q17 / C2.5）。`wsdebug` 与 `debug start` 同样是
		// **起会话**的命令（C3.5），所以别名在这条路上也要登记 —— 否则它起的会话只能用 ref
		// 寻址、也不能当幂等键（C1.6 / Q23）。空 = 未命名，与今天逐字相同（键不出现）。
		Name string `json:"name"`
		// FGLServer 本轮 FGLSERVER 覆盖,语义同 hLaunch(指针:给空串 = 本轮交回 T100 自动设置)。
		FGLServer *string `json:"fglserver"`
		// Ent 是**这一轮**的企业编号(ENT),语义同 hLaunch：必填（或目标会话已有显式 override）。
		Ent string `json:"ent"`
	}
	if !readBody(w, r, &req) {
		return
	}
	// 入参先校验:非法请求不该先把当前会话收口(旧顺序是收口后才在 LaunchReplay 里失败)
	if err := checkReplayOverride(req.Request); err != nil {
		fail(w, 400, err)
		return
	}
	// 点名的环境同样权威:重放必须落在被点名的环境上,否则"重放出来的现象"与
	// "这条日志来自哪个环境"对不上,查出来的结论会整个偏掉。
	// 这里**不带** FGLSERVER 覆盖:那一个只影响"这一轮画面去哪",不是环境身份,
	// 塞进 selectEnv 会把一次实验的接线留在内存配置里。
	targetCfg, err := s.launchCfg(req.SSH, req.Zone, nil)
	if err != nil {
		fail(w, 404, err)
		return
	}
	// `--ent`：与 hLaunch 同一条规矩，只改本轮克隆。
	if e := strings.TrimSpace(req.Ent); e != "" {
		c2 := *targetCfg
		c2.Topent = host.EntValue(e)
		targetCfg = &c2
	}
	// `--name` 的判定与 hLaunch **同一处、同一序**（C2.8 的纪律：一个字节都不该白花）：
	// 取值纪律（F19 → 400 + `USAGE`）与别名的三种结局（Q23：空闲复用 / 忙碌
	// `SESSION_TARGET_BUSY` / 名字不存在起新的）都在**拨号与收口之前**判完 ——
	// 注定要被拒的请求不该先把当前会话收掉。落下来的 name 由 LaunchReplay 一路带到
	// `prepareSession`，别名就在会话登记那一刻成为服务端的事实。
	name, ok := s.beginNamedLaunch(w, req.Name)
	if !ok {
		return
	}
	// 目标先落位、闸门按那一条判 —— 这两件事收在 mustTargetOrEmpty 里（见 api_target.go）。
	//
	// **走冷启动档**（与 `hSessionSwitch` 同一条规矩、同一个理由）：重放干的是**创建**一条
	// 新会话（`LaunchReplay` → `prepareSession`），会话池空恰恰是它最正常的起点。
	// 从前这一处用 mustTarget，于是界面上点「调试此调用」（或 `tt debug wsdebug <rowid>`）
	// 在一台还没有任何会话的机器上**必失败**：409 + `NO_SESSION`「先运行 tt debug start
	// <作业> 起一条」—— 可它自己就是造会话的命令，那条提示等于让人先绕一圈去凑一条，
	// 而重放要的那一条由它自己建才对。
	//
	// 契约 2.4 的 F2（`NO_SESSION`）因此**不再由这两条端点产出**（切环境与按日志重放都是
	// 造会话的命令）。这一档仍活在 CLI 目标层（`internal/target` 的 `failureNoSession`：
	// `tt debug exec --session …` 那类"必须有既有会话"的命令），判据在那个包的 resolve_test，
	// 信封形状由 `TestContractNoSessionEnvelopeIsStillPinned` 钉住。
	// **冷启动档只放松"池空"**：点名了不存在的会话仍是 `SESSION_NOT_FOUND`、多条仍报歧义。
	cur, ok := s.mustTargetOrEmpty(w, r, req.Session, "按接口日志重放")
	if !ok {
		return
	}
	// 换环境 = 换连接目标:热生效(仅内存),后续会话与只读 SQL 都以它为准。
	// 少了这一步,本次重放仍会去连旧环境 —— 那正是"点了名却没生效"的老毛病。
	if targetCfg != s.cfg {
		s.selectEnv(targetCfg)
	}
	conn, err := s.dialSSH()
	if err != nil {
		fail(w, 500, fmt.Errorf("SSH 连接失败: %w", err))
		return
	}
	dbc, err := resolveDBRun(conn, s.cfg)
	if err != nil {
		conn.Close()
		fail(w, 500, err)
		return
	}
	item, content, err := WSLogDetail(conn, dbc, req.RowID)
	conn.Close()
	if err != nil {
		fail(w, 500, err)
		return
	}
	// 已有会话先收口:同环境(可复用空闲宿主)只结束本轮运行;不同环境直接断开。
	// 重放调试独占调试通道,旧会话不再有用。
	//
	// **收口对象来自上面落位的目标**（C7.1-3）：从前这里是 `s.mgr` 的 Current() ——
	// 多会话下那等于"随机收掉一条"（B11 的第三处），而且这一处**不经模式闸门**。
	if cur != nil {
		if cur.sameEnvAs(s.cfg) {
			s.mgr.emit(Event{Type: "log", SessionID: cur.ID, Text: "重放调试启动,结束当前调试(会话保留)"})
			_ = cur.EndRun()
			// EndRun 卡死时降级为 Close(StateExit),残留会话由 prepareSession 清扫
		} else {
			s.mgr.emit(Event{Type: "log", SessionID: cur.ID, Text: fmt.Sprintf("重放调试启动,断开当前会话(环境切换为 %s)", s.cfg.EnvName())})
			_ = cur.Close()
			s.mgr.RemoveAs(cur.ID, "switch")
		}
	}
	sess, replayWarn, err := s.mgr.LaunchReplay(item, content, req.Request, req.FGLServer, name)
	if err != nil {
		// WP6：与 hLaunch 同一条 —— 会话准备报出来的 F1（多会话 + 没点名要哪条宿主）
		// 必须带 `code` 与候选清单，不许降级成一句散文。
		if s.writeLaunchFailure(w, err) {
			return
		}
		fail(w, 409, err)
		return
	}
	// 模式与 hLaunch 同规则:谁发起谁驾驶。重放同样是 AI 从 CLI 发起的,
	// 漏掉这一句会让 AI 连自己刚启动的会话都写不了(quit 都会被 403 拦下)。
	sess.SetMode(launchModeFor(r))
	go func() {
		if err := sess.Launch(r.Context()); err != nil {
			s.mgr.emit(Event{Type: "log", SessionID: sess.ID, Text: "启动失败: " + err.Error()})
			if sess.Booted() {
				if sess.State() == StateExit {
					s.mgr.Remove(sess.ID)
				}
			} else {
				sess.ForceExit()
				s.mgr.Remove(sess.ID)
			}
		}
	}()
	s.emitAction(r, sess, "session.replay", "按接口日志重放调试")
	// warn 非空 = 这次重放用的是入库的截断文本,结果可能不可信(见 WriteReplayFiles)。
	// 随回包一起给出去,让 CLI 能当场说清,而不是等人自己去翻事件日志。
	writeJSON(w, 200, map[string]any{"ok": true, "sessionId": sess.ID,
		"module": sess.Module, "prog": sess.Prog, "runProg": sess.RunProg, "warn": replayWarn,
		"acct": wslogAcctFor(s.wslogTopent(cur))})
}

// wslogTopent 取"此刻的 TOPENT 是什么"。
//
// **只用于说明** —— 查 wsfa_t 恒用系统账号 ds(见 wslogAcctFor),这个值不影响用哪个账号。
// 优先会话里的生效值(会话级覆盖 > 环境配置),没有会话就用环境配置。
//
// **`sess` 由调用方给,不再从会话池里取"当前会话"**（C7.1-4 的那一处 `Manager.Current()`）：
// 服务端不许自己挑一条（Q4/C8.2）。会话外的那两条（`hWSLogs` / `hWSLogContent`）
// 传 nil —— 它们是服务级只读，目标本来就只有"环境"这一层（契约第 7 节的表）。
func (s *Server) wslogTopent(sess *Session) string {
	if sess != nil {
		if v := sess.TopentOverride(); v != "" {
			return v
		}
	}
	return string(s.cfg.Topent)
}

// hDBSQL 只读 SQL:POST /api/dbsql {sql, ent, timeout}
//
// **不过模式闸门** —— 与 /api/wslogs 同理:模式闸门管的是"谁能写调试会话",
// 而这条不碰会话、不移动程序位置,只是读库。人与 AI 都放行。
// 它有自己的配置闸门(db.readonlySql,默认开),那才是这个能力的开关。
//
// **本批的定稿（Q4 / C8.2 硬切 + 契约 C7.1-5）**：`ent` **必填**。
// 从前 `ent <= 0` 会回落 `Manager.Current()` 的 TOPENT（B11 的第五处），于是"查哪个企业"
// 取决于池子里随机哪条会话 —— 查错 schema 的结果是 0 行，而 0 行会被读成
// "这条业务数据不存在"（`Session.TopentIntForDB` 的注释里写着这是最危险的错误结论）。
// 现在这个值只能由调用方显式给：`tt debug sql --ent`，CLI 侧按它会落位的那条会话填
// （`internal/cli/debug` 的 dbgSQLTopentForRun），单会话行为因此一字不变。
//
// 归因用 Type=log(不是 ai_action —— 那个的语义是"AI 做了写操作",只读不该占用它),
// 且记在**执行成功之后**,与 emitAction 的时机一致。
func (s *Server) hDBSQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SQL     string `json:"sql"`
		Ent     int    `json:"ent"`     // 必填(>0):要查哪个企业。C7.1-5 删掉了"回落当前会话"那条兜底
		Timeout int    `json:"timeout"` // 秒
	}
	if !readBody(w, r, &req) {
		return
	}
	// 企业编号必须显式给（C7.1-5）。这不是"多要一个参数"：缺它的时候服务端只能去
	// 某条会话上猜一个企业出来，而猜错的后果（0 行）与大写"数据不存在"一模一样。
	if req.Ent <= 0 {
		writeJSON(w, 400, failureEnvelope{
			OK: false, Code: output.CodeUsage,
			Error: "ent 必填(要查哪个企业编号):tg db sql --at <环境> " +
				"（旧 tt debug sql 才会从会话补；新命令请显式点名）",
		})
		return
	}
	ent := req.Ent

	// 直连优先：进程内复用一条连接（建连+登录只付一次），拿不到才回退 SSH + sqlplus。
	// 这一条只在挂了库、且能按 ENT 清单解出账号时才尝试 —— 其余情况行为与从前一字不差。
	if res, derr, handled := s.directSQL(r.Context(), ent, req.Timeout, req.SQL); handled {
		if derr != nil {
			fail(w, 400, derr)
			return
		}
		s.logAISQL(r, res, req.SQL)
		writeJSON(w, 200, map[string]any{"ok": true, "result": res})
		return
	}

	conn, err := s.dialSSH()
	if err != nil {
		fail(w, 500, fmt.Errorf("SSH 连接失败: %w", err))
		return
	}
	defer conn.Close()
	res, err := s.mgr.RunReadonlyQuery(conn, ReadonlyQueryReq{SQL: req.SQL, Ent: ent, Timeout: req.Timeout})
	if err != nil {
		fail(w, 400, err)
		return
	}
	s.logAISQL(r, res, req.SQL)
	writeJSON(w, 200, map[string]any{"ok": true, "result": res})
}

// logAISQL 把 AI 发起的只读 SQL 记进时间线（只留首行线索，不留全文）。
func (s *Server) logAISQL(r *http.Request, res *DBQueryResult, sqlText string) {
	if actorOf(r) != "ai" {
		return
	}
	text := "只读 SQL(企业 " + fmt.Sprint(res.Ent) + " → " + res.Account + "): " + firstLine(sqlText, 120)
	s.mgr.emit(Event{Type: "log", Actor: "ai", Action: "sql", Text: text})
}

// firstLine 取首行并截断(时间线与日志里只留个线索,不留全文)
func firstLine(s string, max int) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// 命令输出落本地的阈值:太小会给每条命令都留一个文件,没必要
const (
	execLogMinLines = 200
	execLogMinBytes = 16 << 10
)

func linesBytes(ls []string) int {
	n := 0
	for _, s := range ls {
		n += len(s) + 1
	}
	return n
}

// execLogName 落盘文件名:时间戳 + 命令摘要,便于人肉对照这是哪个命令的输出。
// 摘要按 rune 走、用字节数封顶,别把中文劈成半个。
func execLogName(cmd string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(cmd) {
		switch {
		case r == ' ' || r == '\t':
			b.WriteRune('_')
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
		if b.Len() >= 60 {
			break
		}
	}
	return time.Now().Format("20060102-150405.000") + "-" + pathSafeSeg(b.String()) + ".txt"
}

// ---------- 设置 / 环境清单 ----------

// hHostsGet 返回环境清单。独立运行(tt debug serve)时给 tt debug env 读,
// 字段与统一服务的顶层 GET /api/hosts 一致(那边由 tt/internal/web 实现)。
//
// 合并前这条读的是 /api/settings,它吐的是整个 debug 配置节 —— 那时节既装
// 环境清单(debug.sshs)又装调试设置。拆节之后环境清单归 hosts、设置归 debug,
// 页面统一走顶层 /api/hosts;这里只为独立运行保留一份只读视图。
func (s *Server) hHostsGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"config":    s.cfgPath,
		"activeEnv": s.cfg.envName, // 当前生效环境;空 = 按 hosts.activeEnv 取首条
		"sshs":      s.cfg.SSHs,
		"listen":    s.cfg.Listen,
		"debug":     debugSettingsOf(s.cfg),
	})
}

// ReloadConfig 重新从磁盘读取配置并热替换内存态。
//
// 统一服务(tt serve)的共享配置端点 /api/hosts 在写入后会调用它:统一设置页
// 改的就是同一份 hosts 节,不重读的话本服务还按旧环境连,表现就是"改了没生效"。
//
// 三条约定:
//   - 并发安全:整份配置的替换在 s.mu 里做,不会让读者看到半个新配置;
//   - 幂等:重复调用只是又读一遍同样的文件;
//   - 失败不动现场:文件缺失/格式错误时直接返回错误,内存里的 *s.cfg 原样保留 ——
//     别处一次坏写不该把正在跑的调试会话的配置清空。
func (s *Server) ReloadConfig() error {
	if s.cfgPath == "" {
		return fmt.Errorf("服务未挂接配置文件路径,无法重新读取配置")
	}
	cfg, err := LoadConfigAllowEmpty(s.cfgPath)
	if err != nil {
		return err
	}
	// 运行时字段不随文件走:数据目录由服务注入,当前环境由会话选定 ——
	// 保留它,设置页改一次 hosts 就不会把正在调试的会话悄悄切到默认环境。
	cfg.DataDir = s.cfg.DataDir
	if cur := s.cfg.envName; cur != "" {
		cfg.envName = cur
	}
	cfg.ApplyDefaultEnv() // 当前环境的连接/参数合并到运行时字段
	cfg.fillDefaults()
	s.swapConfig(cfg)
	return nil
}

// swapConfig 整份替换内存配置。配置写入路径与 ReloadConfig 共用同一把锁。
func (s *Server) swapConfig(cfg *Config) {
	s.mu.Lock()
	*s.cfg = *cfg
	s.mu.Unlock()
	// 连接要素可能变了（换了环境/账号/库地址）：整池作废，下一次查询重连。
	s.dbPool.closeAll()
}

// ---------- WebSocket ----------

// wsReplayLimit WS 建连时补发多少条历史(对齐前端时间线的容量)
const wsReplayLimit = 500

// replayTypes 只补发"时间线类"事件:state/dead 由页面自己的快照拉取与状态同步负责,
// 混进补发批次反而可能让前端拿旧状态覆盖新状态。
var replayTypes = map[string]bool{"stopped": true, "ai_action": true, "log": true, "watchdog": true}

// wsScope 解出这次订阅的**范围**（契约 6.1 / C6.2 / C6.4 / C6.7 / Q11）。
//
//	?all=1               → 全部（总览台显式订阅全部）→ sessionID = ""
//	?session=<原始句柄>  → 只看那一条 → sessionID = 那个 id
//	两个都不给           → 等价 `all=1`：**今天的 `/api/ws` 就是"全部事件"**
//	                       （调用点只有前端的 store.ts，它今天不带任何查询参数）。
//
// 「两个都不给 = 全部」这一条是**刻意的**，理由两条：
//
//  1. **C6.2 说这是"等价唯一就落"**（0 条或 1 条时按它筛，≥2 条报 400 + SESSION_AMBIGUOUS）。
//     而它今天就是"全部"，且在**单会话**下"按唯一那条筛"与"全部"**恰好同解**
//     （池里只有一条，无会话事件本来就只发给 all —— C6.5）。所以这一档**没有**可观测的行为差，
//     而 H2 要的是"单会话下与今天逐字相同"；
//  2. 多会话下再收紧成 400 会**当场打断已经构建好的前端包**（`web/app/src/store.ts`
//     连的是不带参数的 `/api/ws`，而 `web/` 不在本批的可改范围里）——
//     那不是本批要交付的东西，也不会因为服务端先收紧就自动对齐。
//
// 于是：新语义住在**新参数**上（`?session=` 会让它只拿那一条），旧写法原样可用；
// 前端切到 `?all=1`（外加按选中会话的 `?session=`）是前端那一批的活（WP-F 的 ④/⑤）。
//
// `?session=` 收的是**服务端的原始句柄**（与 C5.4 一致：路径与句柄只认 `id`，不认
// `ref` / `display`；选择器文法的家在 CLI 的目标层，Q12 / C8.2）。指名的会话不在池里
// 就当场 404 + `SESSION_NOT_FOUND`（不许静默回落成"那就全部"）。
func (s *Server) wsScope(w http.ResponseWriter, r *http.Request) (string, bool) {
	q := r.URL.Query()
	id := strings.TrimSpace(q.Get("session"))
	if id == "" {
		return "", true // ?all=1 与"什么都没给"同解：全部
	}
	if s.mgr.Get(id) == nil {
		writeCandidateFailure(w, http.StatusNotFound, target.CodeNotFound,
			"会话不存在: "+id, nil)
		return "", false
	}
	return id, true
}

func (s *Server) hWS(w http.ResponseWriter, r *http.Request) {
	// 范围在**升级之前**定：落不上目标要能正常回一个 JSON 失败信封（C5.16），
	// 而升级成 WebSocket 之后就再也写不了普通响应了。
	sessionID, ok := s.wsScope(w, r)
	if !ok {
		return
	}
	ch, replay, cancel := s.mgr.SubscribeWithReplayScoped("ws", sessionID, wsReplayLimit)
	defer cancel()
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	// ctx 与连接同生命周期:下面那个丢弃式 reader 一旦读到错误就取消它。
	ctx, cancelConn := context.WithCancel(r.Context())
	defer cancelConn()

	// 必须有人读:否则库处理不了对端的 close/ping 帧 —— 客户端关闭时服务端要干等到
	// 超时才发现,代理上的 ping 也得不到回应(这条通道是纯推送,收到的都丢掉)。
	go func() {
		defer cancelConn()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	writeFrame := func(v any) error {
		data, err := json.Marshal(v)
		if err != nil {
			return err
		}
		wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
		defer wcancel()
		return conn.Write(wctx, websocket.MessageText, data)
	}

	// 开场补发:页面刷新后时间线不该是空的(而"看着 AI 干活时刷新一下"是很自然的动作)。
	// 前端收到 replay 帧**只追加时间线、不触发任何副作用**,页面状态另有 refreshSnapshot 负责。
	kept := replay[:0]
	for _, ev := range replay {
		if replayTypes[ev.Type] {
			kept = append(kept, ev)
		}
	}
	if err := writeFrame(map[string]any{"type": "replay", "epoch": s.mgr.Epoch, "events": kept}); err != nil {
		return
	}
	// hello 哨兵:带 epoch,前端据此判断服务端是否重启过(重启则序号归零,要重置去重游标)
	if err := writeFrame(map[string]any{"type": "hello", "epoch": s.mgr.Epoch}); err != nil {
		return
	}

	for {
		select {
		case ev := <-ch:
			if err := writeFrame(ev); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// hStatic 是 `mux.HandleFunc("/", …)` 那条兜底：先服务 web/dist 里的静态文件，
// 未命中再回 index.html(SPA 客户端路由)。
//
// **但 `/api/` 前缀不走兜底**。兜底会把任何未注册路径都吞掉，于是打错的 API 路径
// 拿到的是 **200 + HTML**（前端没构建时是"前端尚未构建"那页，构建过是 index.html）——
// 调用方按 JSON 解析时报的错与真实原因（接口不存在）完全对不上。这条规矩本仓库
// 在 internal/web（hRoot 对非 `/` 路径回 404）与 internal/dict/server
// （TestSharedEndpointsNotServedHere）两侧都已经立过，这里是最后一处。
//
// 代价是方法不匹配也归到这里（`GET /api/dbsql` 得 404 而不是 405）：Go 的 mux
// 只在**没有** catch-all 时才会自己产生 405，这里有一条，所以 405 得手写。
// 404 与 405 对调用方是同一件事——"这个路径上你没有可用的接口"——不值得为它
// 再维护一张注册表。
func (s *Server) hStatic(w http.ResponseWriter, r *http.Request) {
	if p := r.URL.Path; p == "/api" || strings.HasPrefix(p, "/api/") {
		fail(w, http.StatusNotFound, fmt.Errorf("接口不存在：%s %s", r.Method, p))
		return
	}
	if s.webSub == nil {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html lang="zh"><meta charset="utf-8">
<title>tt debug debug</title><body style="font-family:system-ui;padding:40px;line-height:1.8">
<h2>tt debug server 已运行</h2>
<p>前端尚未构建。请在 web/ 目录执行 <code>npm install &amp;&amp; npm run build</code> 后重启服务。</p>
<p>API 已可用:<code>/api/status</code>、<code>/api/ws</code>、<code>/api/sessions</code> …</p></body>`)
		return
	}
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}
	f, err := s.webSub.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		// SPA 兜底:未命中路径回 index.html
		f2, err2 := s.webSub.Open("index.html")
		if err2 != nil {
			http.NotFound(w, r)
			return
		}
		f = f2
		path = "/index.html"
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if st.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, path, st.ModTime(), f.(readSeeker))
}

type readSeeker interface {
	Read(p []byte) (int, error)
	Seek(offset int64, whence int) (int64, error)
}

// hSessionJob 在**指定的会话**里起一个作业（`POST /api/sessions/{id}/job`）。
//
// 与 `POST /api/sessions` 的分工：那条是"建会话（可顺带起作业）"；这条是"会话已经在了，
// 只在它里面起作业" —— 调用方不必再拼环境/别名的匹配条件，`{id}` 就是宿主的唯一事实。
//
// 前置：会话必须**空闲**（作业在跑时再起一个，等于同一台 shell 上叠两个客户端）。
func (s *Server) hSessionJob(w http.ResponseWriter, r *http.Request) {
	sess := s.sessOfWriteCap(w, r, CapSetup)
	if sess == nil {
		return
	}
	var req struct {
		Prog      string  `json:"prog"`
		Module    string  `json:"module"`
		Ent       string  `json:"ent"`
		FGLServer *string `json:"fglserver"`
	}
	if !readBody(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Prog) == "" {
		fail(w, 400, fmt.Errorf("需要 prog：在会话里起的是哪个作业"))
		return
	}
	if st := sess.State(); st != StateIdle {
		fail(w, 409, fmt.Errorf("会话不是空闲（%s）：先结束当前这一轮，再在它里面起新作业", st))
		return
	}
	cfg := s.envCfgFor(sess.EnvName())
	if cfg == nil {
		fail(w, 404, fmt.Errorf("环境 %q 不存在（会话登记着它，但配置里没有）", sess.EnvName()))
		return
	}
	cc := *cfg
	cfg = &cc // 只改本轮克隆：ent 覆盖不许留在内存配置里
	if e := strings.TrimSpace(req.Ent); e != "" {
		cfg.Topent = host.EntValue(e)
	}
	module := strings.TrimSpace(req.Module)
	// T100 的 -m 默认是 asf；GP 没有这个默认，空模块要留给 startRunGP 按 p_link 解析。
	if module == "" && !strings.EqualFold(cfg.Product, "gp") {
		module = "asf"
	}
	// hostID = 这条会话：`prepareSessionOn` 的候选**就是它一条**，不会再挑别人。
	ns, err := s.mgr.LaunchWith(cfg, module, strings.TrimSpace(req.Prog), "", sess.ID)
	if err != nil {
		fail(w, 502, err)
		return
	}
	// 模式由**发起方**决定（与 hLaunch / hSessionSwitch 同一条规矩）：漏了它，业主授权时
	// 新起的这一轮默认 collab，人在界面上的动作会被自己的闸门拒掉，而且是静默的。
	ns.SetMode(launchModeFor(r))
	s.emitAction(r, ns, "session.job", fmt.Sprintf("在会话里起作业 %s", req.Prog))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok": true, "sessionId": ns.ID, "session": refOf(ns.EnvName(), ns.Seq()),
		"prog": strings.TrimSpace(req.Prog), "module": module,
	})
}
