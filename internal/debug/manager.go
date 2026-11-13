package debug

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"tt/internal/config"
	dictlib "tt/internal/dict"
	"tt/internal/host"
	sessionid "tt/internal/session"
	"tt/internal/target"

	"github.com/pkg/sftp"
)

// subscriber 是一个事件订阅者（契约 6.6 / C6.7 的"范围"维度）。
//
// 从前 `subs` 的 value 只有"标签"，筛选只能落在**客户端**（`hWait` 就是这么做的，
// 契约 C7.2-2 记为 E 类）；Q11 定稿是**发送那头筛**，所以登记时要带上订阅者
// 关心的那条会话：`sessionID == ""` = 全部（总览台显式订阅全部时用它）。
type subscriber struct {
	tag       string
	sessionID string
}

// Manager 调试会话管理器:持有会话、向订阅者(WS/MCP)广播事件、保留最近事件供查询
type Manager struct {
	cfg *Config

	mu       sync.Mutex
	sessions map[string]*Session

	subMu sync.Mutex
	subs  map[chan Event]subscriber

	// auditPath 是会话操作审计行的落点（`<DataDir>/audit/sessions.jsonl`）。
	// 空 = 不落盘（DataDir 为空：单测与未开持久化的部署，见 audit.go）。
	auditPath string
	// auditActor 取"谁"（缺省 `target.DefaultActor(os.Getenv)`，读 `TT_ACTOR`）。
	// 做成字段是为了让判据测试能注入一个确定的身份，而不必去改进程环境。
	auditActor func(string) string

	// newSession 是"开一条新会话"的入口。生产路径**恒为** `NewSession`（一字不差），
	// 做成字段只有一个理由：`NewSession` 会真拨 SSH（`host.Dial`，没有可注入的缝），
	// 于是"多会话/单会话"那几条 HTTP 判据在无网环境里只能测到拨号失败那一层，
	// 测不到"落位之后到底成不成功"。注入一个不拨号的实现，那些判据才测得到它们要说的事。
	newSession func(cfg *Config, module, prog, runProg, launchRef, extraArgs string, emit func(Event)) (*Session, error)

	// 最近事件环形缓冲(供 AI/日志 tail 查询;只存结构化事件,output/autovars 不入)
	evMu sync.Mutex
	evs  []Event

	// seq 事件单调序号。用原子而非在 evMu 里自增:output 事件也要有序号
	// (前端统一按序号去重),而 output 不入缓冲、根本不进 evMu。
	// 序号是进程级的、重启归零,所以配一个 Epoch 让前端知道该重置游标。
	seq   atomic.Uint64
	Epoch string

	// T100 动态环境缓存(登录区域 → 路径),TTL 5 分钟;探针失败不缓存
	envMu    sync.Mutex
	envCache map[string]cachedEnv

	// 作业→实体程序的解析缓存,TTL 5 分钟;与 envCache 同一条策略。
	// 为什么要有它:每次启动都要经 SSH 起 ksql 连库查 gzzz_t JOIN gzza_t（2026-10-23
	// 实测约 3s，是"点启动到弹画面"等待的大头），而调试场景里**同一个作业会反复启动**
	// （改断点再跑），映射在服务器上变化极低频 —— 5 分钟的陈旧窗口可接受。
	jobMu    sync.Mutex
	jobCache map[string]cachedJob

	// LocalJobLookup 本地字典库的作业解析（先于远程 ksql 链路，本地命中零远程往返）。
	// 注入式（与 auditActor 同风格）：debug 包不直接依赖 config 的落点计算，
	// 由 NewServer 装配；单测注入 fake 验证"本地命中时不拨 SSH"。可为 nil（回退远程）。
	LocalJobLookup func(envName, job string) (JobResolve, bool)

	// 会话外源码读取共用的连接(见 srcmirror.go):
	// 每次 tt debug source/grep 重新握手太贵,按 SSH 目标缓存一条长连接。
	//
	// 按 key 分开存（缺陷 Z / T5.3）:从前只有一条 + 一把全局锁,
	// 换目标即换连接 ⇒ 两个环境不可能同时在读源码。
	srcMu sync.Mutex
	src   map[string]*srcConn
	// srcKeyLocks 每个连接 key 一把锁:不同目标真正并行,同一目标排队（见 withSrcConn）。
	srcLocksMu  sync.Mutex
	srcKeyLocks map[string]*sync.Mutex

	// 企业目录(ENT→账号):只读 SQL / 接口日志都要先问一次 gzou_t,
	// 同一个环境(主机+账号+区域+库)下缓存 10 分钟,并落一份快照供离线使用。
	// 见 ents.go —— 取值顺序、降级规则、新鲜度口径都在那里,**只有那一份实现**。
	ents *EntCatalog
}

// maxBufferedEvents 环形缓冲上限。
const maxBufferedEvents = 4000

type cachedEnv struct {
	env *host.RuntimeEnv
	at  time.Time
}

// cachedJob 作业解析缓存的一行（resolveJobWith 的成功返回四元组）。
type cachedJob struct {
	mod, prog, launchRef, extra string
	at                          time.Time
}

func NewManager(cfg *Config) *Manager {
	return &Manager{
		cfg:      cfg,
		sessions: map[string]*Session{},
		subs:     map[chan Event]subscriber{},
		envCache: map[string]cachedEnv{},
		src:      map[string]*srcConn{},
		// cfg.DataDir 可能为空(单测里的 NewManager(&Config{})):那一档只走内存不落盘
		ents:  NewEntCatalog(cfg.DataDir),
		Epoch: strconv.FormatInt(time.Now().UnixNano(), 36),
		// 会话操作审计（WP6）：<DataDir>/audit/sessions.jsonl，谁 / 哪条 / 干了什么。
		auditPath: auditPath(cfg.DataDir),
		// "谁"的缺省来源是 `TT_ACTOR`（与 api.go 的 actorOf 同一个开关，不另立第二个）。
		// 包一层是因为 `target.DefaultActor` 要一个**注入的** getenv（那样它才是纯函数、可测）。
		auditActor: func(string) string { return target.DefaultActor(os.Getenv) },
		// 生产路径就是 NewSession：行为与从前一字不差（只有 _test.go 会替换它）。
		newSession: NewSession,
	}
}

// getRuntimeEnv 取 T100 动态环境:缓存命中直接返回;未命中(或过期)探针一次并缓存。
// 失败返回错误(不缓存),调用方须报错,无静态配置可回退。key 按 服务器+账号+区域 区分
// (用连接的真实 host,覆盖/多环境时不受 active 环境影响),换环境互不污染。
//
// **它只返回,不写任何配置**（缺陷 Y / T5.2 的修法）:动态路径是"这一次连接"算出来的事实,
// 由调用方放进**自己那一份**配置里（本轮请求克隆出来的、或会话自己持有的那一份）;
// 从前 `ensureRuntimeEnv` 把它写回 `m.cfg.Runtime` —— 而 `m.cfg` 是**共享**的,
// 于是"哪条会话先启动"决定了另一条看到的 TOP/ERP,两条会话的动态路径互相污染。
func (m *Manager) getRuntimeEnv(conn *host.SSHConn, zone string) (*host.RuntimeEnv, error) {
	cc := conn.Cfg()
	key := cc.Host + "|" + cc.User + "|" + zone
	m.envMu.Lock()
	if c, ok := m.envCache[key]; ok && time.Since(c.at) < 5*time.Minute && c.env.Valid() {
		m.envMu.Unlock()
		return c.env, nil
	}
	m.envMu.Unlock()
	env, err := host.ProbeTEnv(conn, zone)
	if err != nil {
		return nil, err
	}
	m.envMu.Lock()
	m.envCache[key] = cachedEnv{env: env, at: time.Now()}
	m.envMu.Unlock()
	return env, nil
}

// zoneOf 生效登录区域:空 = 默认 36（与既有行为一致）。
func zoneOf(cfg *Config) string {
	if cfg == nil || cfg.Zone == "" {
		return "36"
	}
	return cfg.Zone
}

// ensureRuntimeEnvOn 确保**这一份** cfg 有动态路径(探针+缓存),写进 cfg 自己而不是共享配置。
//
// 它是 `ensureRuntimeEnv` 的替代:后者写的是 `m.cfg.Runtime`（共享字段,缺陷 Y）。
// cfg 必须是调用方自己那一份（本轮的克隆 / 会话持有的快照）—— 传 `m.cfg` 进来
// 就是把这个缺陷原样搬个地方。
func (m *Manager) ensureRuntimeEnvOn(conn *host.SSHConn, cfg *Config) (*host.RuntimeEnv, error) {
	if cfg.Runtime != nil && cfg.Runtime.Valid() {
		return cfg.Runtime, nil
	}
	// TOP GP 没有 T100 的运行时环境（$TOP/$CUST/$FGLLDPATH 那套由 r.d2 自己铺垫），
	// 探测必然失败；给它一个空壳，让启动段按自己的方式走。
	if strings.EqualFold(cfg.Product, "gp") {
		cfg.Runtime = &host.RuntimeEnv{}
		return cfg.Runtime, nil
	}
	zone := zoneOf(cfg)
	env, err := m.getRuntimeEnv(conn, zone)
	if err != nil {
		return nil, fmt.Errorf("无法获取环境 %s(zone %s)的 T100 路径: %w", cfg.EnvName(), zone, err)
	}
	cfg.Runtime = env
	return env, nil
}

// Subscribe 订阅事件流(WS 用),返回取消函数
//
// **签名、语义与调用点一个字不动**（Q22 / C6.7 的"纯新增"纪律）：它等价于
// `SubscribeScoped(tag, "")` —— `sessionID == ""` 就是"全部"这个范围。
func (m *Manager) Subscribe(tag string) (<-chan Event, func()) {
	return m.SubscribeScoped(tag, "")
}

// SubscribeScoped 订阅事件流，并声明**只关心哪一条会话**（Q11 / 契约 C6.4 / C6.7）。
//
// `sessionID == ""` = 全部（总览台显式订阅全部时走它，与 Subscribe 等价）。
// 非空时**在发送那头筛**（`emit` 里按 sub 判），不是推给客户端让它自己丢 ——
// 这正是 `hWait` 从前那处 E 类写法（契约 C7.2-2）的定稿归属。
//
// 契约 C6.5：**无会话的事件**（`SessionID == ""`，如服务级 log / rec.progress）
// 只发给"全部"订阅者；单会话订阅者收不到 —— 它拿到的每一条都必须能被它自己那条会话解释。
func (m *Manager) SubscribeScoped(tag string, sessionID string) (<-chan Event, func()) {
	ch := make(chan Event, 4096)
	m.subMu.Lock()
	m.subs[ch] = subscriber{tag: tag, sessionID: sessionID}
	m.subMu.Unlock()
	return ch, func() {
		m.subMu.Lock()
		delete(m.subs, ch)
		m.subMu.Unlock()
	}
}

// SubscribeWithReplay 订阅事件流,同时取回最近 n 条历史(供 WS 建连时开场补发)。
//
// 订阅与取快照放进**同一个 evMu→subMu 临界区**(锁序与 emit 一致):
// 这样保证**不遗漏** —— emit 的 append 先于 broadcast,窗口内产生的事件
// 要么已经进了快照、要么会进 channel。但**仍可能重复**(emit 的 append 与
// broadcast 之间不持锁),所以调用方必须用 Seq 去重。
//
// **签名、语义与调用点一个字不动**（Q22 / C6.7 的"纯新增"）：它等价于
// `SubscribeWithReplayScoped(tag, "", n)`。
func (m *Manager) SubscribeWithReplay(tag string, n int) (<-chan Event, []Event, func()) {
	return m.SubscribeWithReplayScoped(tag, "", n)
}

// SubscribeWithReplayScoped 同 SubscribeWithReplay，但带上**订阅范围**（Q11 / C6.6 / C6.7）。
//
// 两条与范围有关的语义：
//   - **补发也按范围过滤**（C6.6）：`sessionID` 非空时只回该会话的历史，
//     过滤在这**一次**做掉，不在客户端做；
//   - **`Seq` 空间不分叉**（C6.7 末段）：序号仍由 `emit` 的第一件事分配，
//     全订阅者共享同一个序号空间 —— 补发切片被过滤后，客户端的去重游标仍然单调。
func (m *Manager) SubscribeWithReplayScoped(tag string, sessionID string, n int) (<-chan Event, []Event, func()) {
	ch := make(chan Event, 4096)
	m.evMu.Lock()
	m.subMu.Lock()
	m.subs[ch] = subscriber{tag: tag, sessionID: sessionID}
	m.subMu.Unlock()
	replay := replayForLocked(m.evs, sessionID, n)
	m.evMu.Unlock()
	cancel := func() {
		m.subMu.Lock()
		delete(m.subs, ch)
		m.subMu.Unlock()
	}
	return ch, replay, cancel
}

// replayForLocked 从环形缓冲尾部取最多 n 条**该范围**的历史（调用方必须持有 `m.evMu`）。
//
// 过滤**在取切片之前**（而不是"先取 n 条再过滤"）：后者会让单会话订阅者拿到
// 少于 n 条的历史 —— 别的会话的事件把配额吃掉了，而那正是"补发丢历史"的表现。
// `sessionID == ""` 时不筛，逐字等于 SubscribeWithReplay 今天的行为。
func replayForLocked(evs []Event, sessionID string, n int) []Event {
	if n <= 0 || len(evs) == 0 {
		return nil
	}
	out := make([]Event, 0, n)
	for i := len(evs) - 1; i >= 0 && len(out) < n; i-- {
		if !eventInScope(sessionID, evs[i]) {
			continue
		}
		out = append(out, evs[i])
	}
	// 倒着收集的，翻回来（补发按时间升序）。
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// eventInScope 判一条事件该不该发给某个范围的订阅者（Q11 / C6.4 / C6.5 的唯一判据）。
//
//	sessionID == ""  → 全部：什么都发；
//	会话事件         → 只发给订阅了**那一条**的人；
//	无会话事件       → 只发给"全部"（C6.5：单会话订阅者拿到的每一条都必须能被它那条会话解释）。
func eventInScope(subSessionID string, ev Event) bool {
	if subSessionID == "" {
		return true
	}
	return ev.SessionID != "" && ev.SessionID == subSessionID
}

// emit 会话事件回调:入环形缓冲并广播给所有订阅者(非阻塞,满则丢弃)
func (m *Manager) emit(ev Event) {
	ev.Seq = m.seq.Add(1) // 第一件事:序号必须先于任何入缓冲/广播动作
	// 只保留对"发生了什么"有信息量的结构化事件(输出流/自动变量列表不入缓冲,
	// 会占用大量内存且 AI 用 tail 查询时噪声过大)
	// watchdog-approaching / watchdog-failed 与 watchdog 同属“防行锁”那条线：
	// 前者是“快到期了”，后者是“看门狗自己没成功”—— 后者尤其不能静默丢掉。
	switch ev.Type {
	case "state", "stopped", "log", "dead", "watchdog", "watchdog-approaching", "watchdog-failed", "ai_action":
		m.evMu.Lock()
		m.evs = append(m.evs, ev)
		if len(m.evs) > maxBufferedEvents {
			m.evs = m.evs[len(m.evs)-maxBufferedEvents:]
		}
		m.evMu.Unlock()
	}
	m.subMu.Lock()
	defer m.subMu.Unlock()
	for ch, sub := range m.subs {
		// 发送那头筛（Q11 / C6.4）：范围对不上的订阅者根本不进这条事件的投递。
		if !eventInScope(sub.sessionID, ev) {
			continue
		}
		select {
		case ch <- ev:
		default: // 订阅者消费太慢,丢弃(前端靠 state/stopped 事件对齐,丢原始行无碍)
		}
	}
}

// 池与界面会话的事件类型（W2）。字符串只在这里出现一次，别的文件引用常量。
//
// 为什么这四类要有自己的事件，而不是靠副作用（`log` / `state`）判：
//   - 前端要维持一棵**由服务端派生**的树（D9）。靠"看到一条带新 id 的 log 就当池变了"
//     在删除那一半完全失效（删的时候没有任何事件），于是树里留死条目、点进去是空的。
//   - 这两类都是"叫一声"：权威始终是 `GET /api/sessions`（见 W2）。
const (
	EvSessionAdded   = "session.added"
	EvSessionRemoved = "session.removed"
)

// emitAdded 广播"池里多了一条"。**必须在会话已登记之后调**：收到的人会立刻按 id 取快照。
func (m *Manager) emitAdded(s *Session) {
	if s == nil || s.ID == "" {
		return
	}
	b := m.briefOf(s)
	m.emit(Event{Type: EvSessionAdded, SessionID: s.ID, Time: time.Now(), Brief: &b})
}

// emitRemoved 广播"池里少了一条"。**必须在 delete 之后调**。
func (m *Manager) emitRemoved(id, reason string) {
	if id == "" {
		return
	}
	m.emit(Event{Type: EvSessionRemoved, SessionID: id, Time: time.Now(), Reason: reason})
}

// Events 返回最近 n 条事件(按时间升序;n<=0 返回全部缓冲)。
func (m *Manager) Events(n int) []Event {
	m.evMu.Lock()
	defer m.evMu.Unlock()
	if n <= 0 || n >= len(m.evs) {
		out := make([]Event, len(m.evs))
		copy(out, m.evs)
		return out
	}
	out := make([]Event, n)
	copy(out, m.evs[len(m.evs)-n:])
	return out
}

// Launch 创建并启动一个调试会话(同一时间仅允许一个会话,保证生产安全)。
// 启动前清扫已结束的残留会话(程序退出/后端死亡),重启无需先 DELETE。
func (m *Manager) Launch(module, prog string) (*Session, error) {
	return m.launchWith(m.cfg, module, prog, "", "")
}

// LaunchWith 按指定配置启动(多 SSH/多区域:调用方克隆好 cfg 再传入),并给这条会话起别名。
//
// name 是 `--name` 的取值（Q17）：空 = 未命名。别名**就是幂等键**（Q23 / C1.6）——
// 三种结局（空闲复用 / 忙碌报 SESSION_TARGET_BUSY / 名字不存在起新的）由调用方先判一次
// （`beginNamedLaunch`），这里负责把名字**登记成服务端的事实**（C5.3 的 `name` 键），
// 并在登记的瞬间兜住竞态（F18：`ErrNameTaken`）。
//
// hostID 是 `--session` **显式指名**的那条既有会话（服务端原始句柄 `{id}`，C5.4 / Bug A）：
// 空 = 没指名，逐字等于今天的行为。非空 = "**就用这一条**当宿主" —— 候选集当场收敛成它一条，
// 于是 `prepareSession` 那一步的"按环境+名字收窄候选"根本不发生，多候选歧义也就不可能再出现。
// 它**不跳过校验**：状态处置仍是 `decideHost` 那三条（空闲复用 / 忙碌报错 / 目标或别名对不上
// → 报错），见 `prepareSessionOn` 与 `namedHostConflict` 的注释。
func (m *Manager) LaunchWith(cfg *Config, module, prog, name, hostID string) (*Session, error) {
	return m.launchWith(cfg, module, prog, name, hostID)
}

func (m *Manager) launchWith(cfg *Config, module, prog, name, hostID string) (*Session, error) {
	// 对齐 T100 gendbg:启动前连库把作业编号解析成实体程序+模块+启动引用(gzzz_t JOIN gzza_t),
	// 源码与 42r 都跟实体程序走;未配置 db/查询失败/未命中 → 会话内按名称文件搜索兜底
	runProg, launchRef, extra := "", "", ""
	log.Printf("[debug] launchWith 入口 product=%q env=%q module=%q prog=%q hostID=%q",
		cfg.Product, cfg.EnvName(), module, prog, hostID)
	// TOP GP 跳过 gzzz_t 解析：那是 T100 的作业目录表，GP 上查出来的东西对不上
	// （实测一个 csf_csfi300_icd 被解析成 "select gzza001"）。GP 的程序名就是作业编号本身。
	gp := strings.EqualFold(cfg.Product, "gp")
	if !gp {
		if mod2, prog2, ref2, extra2, rerr := m.resolveJobWith(cfg, module, prog); rerr != nil {
			log.Printf("[debug] launchWith 作业解析失败: %v", rerr)
			return nil, rerr
		} else if prog2 != "" {
			runProg = prog2
			launchRef, extra = ref2, extra2
			if module == "" && mod2 != "" {
				module = mod2
			}
		}
	}
	sess, err := m.prepareSessionOn(cfg, module, prog, runProg, launchRef, extra, "", name, hostID)
	if err != nil {
		log.Printf("[debug] launchWith prepareSessionOn 失败: %v", err)
		return nil, err
	}
	log.Printf("[debug] launchWith 准备完成: session=%s module=%q 开始 Launch", sess.ID, sess.Module)
	// 新的一轮运行:清空上一轮攒下的源码副本(见 srcmirror.go)。
	// 放在 prepareSession 成功之后 —— 它可能因"已有活跃运行"而拒绝,
	// 那种失败不该顺手毁掉正在跑的那一轮的副本。
	// 两件事必须成对:清了目录还要清会话的"已镜像"记录,否则复用的宿主会话
	// 会以为文件早就镜像过,新一轮再也落不下副本。
	//
	// 清的是**这条会话自己的**那棵子树（缺陷 X / T5.1）:从前的 `clearMirror(cfg.DataDir)`
	// 删的是全局 `srccache/`,并发下 A 的启动会销毁 B 刚拉下来的镜像。
	// 复用的宿主会话要连"已镜像"那份记录一起继承（见 inheritMirrorState）。
	sess.inheritMirrorState()
	clearMirrorForSession(cfg.DataDir, sess.mirrorScope())
	sess.resetMirrored()
	if runProg != "" && runProg != prog {
		sess.emitEvent(Event{Type: "log", Text: fmt.Sprintf("作业编号 %s → 实体程序 %s(gzzz_t)", prog, runProg)})
	}
	return sess, nil
}

// prepareSession 启动一轮运行前的会话准备（**多会话**：WP6 起不再假设池里只有一条）。
//
// # 它现在干什么（判据只有一条：**按目标在池里找**）
//
// 四条造会话的路径（`Launch`/`LaunchWith`/`LaunchReplay`/`CreateSessionOn` 里的前三条
// 都经这里）共用这一个咽喉点，把"这次落在哪条会话上"判成三档：
//
//   - **候选 0 条 → 起新会话**（`NewSession` + `assignIdentityLocked` 登记身份）；
//   - **候选 1 条 → 按 `decideHost` 的判据处置**：
//     `hostReuse` 复用宿主（`SetRun` 换代）/ `hostCollect` 收口重建 / `hostBusy` 报错；
//   - **候选 ≥2 条 → 报歧义**：`SESSION_AMBIGUOUS` + **退 3** + C5.12 的候选清单，
//     **绝不挑一条**（F1 / C8.1 / Q2：只在多会话时收紧，多条就是"你没说清要哪一条"）。
//
// # "候选"的定义
//
// **与本次目标同环境（`sameTarget`）的活会话** —— 判据与 `decideHost` 的复用判据
// **同一个函数**，不另立一套：`sameTarget` 的语义（连接四元组 + 环境名）是"许不许
// 复用这条连接"的判据，判错就是上错号（见 `Session.sameTarget` 的注释，那里有实测教训）。
// 别名命中是**另一条精确通道**（Q23）：给了 `--name X` 就先按别名找那一条，找到就只针对它判。
//
// # 两条与今天**逐字一致**的行为（H2）
//
//   - 池里只有一条、同目标、空闲 → 复用（`SetRun` 刷新本轮参数）；
//   - 池里只有一条、同目标、非空闲 → 报"已存在调试会话…请先结束当前调试"（不许静默再开一条）。
//
// # 收口的时机（本批改了一处，写在这里免得被读成漏改）
//
// 从前是"**先把旧会话收掉再建新的**"；现在**先建新的（`NewSession` 只拨号 + 开 PTY），
// 建成了才 `Close` 旧的那条**。理由：`hostCollect` 原本是"一条会话的空转会毁在这一步"
// 也不心疼，多条之后它可能收掉一条**正要紧的**会话（B 的会话被 A 的启动收掉）；先建成功
// 再收，失败路径上**一条都不少**。收口仍走 `Close`（它会先中断在跑的那一轮），
// 别名与序号也在它从池里消失的那一刻当场释放（C3.7b / Q19）。
//
// **name（`--name` 的取值）参与复用判据的第二半**（Q23 / C2.5b，见 nameCompatible）：
// 指定了 X 而那条空闲会话叫 Y 时**不复用**它，而是按"换目标"那条路收掉旧的、开一条叫 X 的。
// 别名只在创建时给一次，所以"要一条叫 X 的会话"绝不能落在一条已经叫 Y 的会话上。
//
// # 显式指名（`--session`，Bug A）走的是九参的 `prepareSessionOn`
//
// 末尾那个 `hostID` 非空 = 调用方已经点名了**这一条**既有会话当宿主。于是候选**就是它一条**，
// "按环境+名字收窄候选"这一步根本不发生 —— 多候选歧义连成立的机会都没有（这不是"挑一条"：
// 是调用方点的名，而 `sessionCandidates` 那套"多候选绝不自己挑"的纪律一个字没动）。
// 但指名**不跳过校验**：状态处置仍是 `decideHost` 那三条，唯一的差别是
// `hostCollect`（目标/别名对不上）在指名这一档改报错，不把点名的会话收掉重建
// （见 `namedHostConflict`）。本函数是"没指名"那一档，逐字等于今天的行为 ——
// 既有判据测试走的就是它。
//
// 返回已登记的会话,由调用方驱动 Launch 登录/启动。
func (m *Manager) prepareSession(cfg *Config, module, prog, runProg, launchRef, extra, argsOverride, name string) (*Session, error) {
	return m.prepareSessionOn(cfg, module, prog, runProg, launchRef, extra, argsOverride, name, "")
}

// prepareSessionOn 是 prepareSession 的**九参入口**：多一个 `hostID`（`--session` 指名的宿主，
// 空 = 没指名）。判据、文案、收口时机全在 prepareSession 的注释里，这里只多那一档。
func (m *Manager) prepareSessionOn(cfg *Config, module, prog, runProg, launchRef, extra, argsOverride, name, hostID string) (*Session, error) {
	// 候选从哪来：指名了就是那一条；没指名就按环境+名字收窄（今天的行为）。
	var cands []*Session
	named := false
	if id := strings.TrimSpace(hostID); id != "" {
		host, err := m.namedHost(id)
		if err != nil {
			return nil, err
		}
		cands, named = []*Session{host}, true
	} else {
		cands = m.sessionCandidates(cfg, name)
	}

	// ---- 候选 ≥2：报歧义，绝不挑一条（F1 / C8.1 / Q2） ----
	if len(cands) >= 2 {
		f := m.failAmbiguous(cfg, cands)
		for _, s := range cands {
			m.audit(s, auditAmbiguous) // 每条候选都留一条审计：'谁被卷进这次歧义'要查得到
		}
		return nil, f
	}

	var prev *Session
	if len(cands) == 1 {
		prev = cands[0]
		decision := decideHost(prev, cfg, name)
		// 指名的会话"换了目标（环境）"或"别名对不上"时，decideHost 给的是 hostCollect ——
		// 那条路会把它 `Close` 掉再按新的重建。对**候选**那是对的（空转宿主没什么可丢的），
		// 对**点名的那一条**不行：那等于把"用这一条"读成"用别处起一条"，而用户手上那个
		// ref/别名还指着它。所以这一档改报错，绝不收口。
		if named && decision == hostCollect {
			return nil, m.namedHostConflict(prev, cfg, name)
		}
		switch decision {
		case hostReuse:
			// 企业编号必须是**显式的**（见 requireExplicitEnt）。
			if err := requireExplicitEnt(cfg, prev); err != nil {
				return nil, err
			}
			// 复用宿主跑新一轮：SetRun 会把世代换掉（F15：旧句柄当场失效），
			// 序号与别名**不动**（它们标识的是"这条会话"，跨轮不变：C2.0 / C2.5b）。
			prev.SetRun(cfg, module, prog, runProg, launchRef, extra)
			prev.ArgsOverride = argsOverride
			// 这条宿主此前没有名字，而这次点名要 X：别名登记给它（Q23 的"起新的"落在
			// 复用宿主上的形态）。它此前无名，所以不是改名（C2.5b 禁的是改名）。
			if err := m.nameSession(prev, name); err != nil {
				return nil, err
			}
			m.audit(prev, auditReuse)
			return prev, nil
		case hostBusy:
			// **运行中 / 已停站：不许悄悄收掉它** —— 这与今天那句"请先结束当前调试"
			// 是同一条取向（H2 保护的就是这句）。别名对不上也不改变这一条：
			// 要抢名字先停掉那条会话（F20 的 hint 是这么说的）。
			m.audit(prev, auditBusy)
			return nil, fmt.Errorf("已存在调试会话(%s 运行中/已停站),请先结束当前调试", prev.Prog)
		default: // hostCollect：空闲但换了目标（环境）或换了名字 —— 收口它再按新的重建
			// 收口**推迟到新会话建成之后**（见函数头"收口的时机"）。
		}
	}

	// ---- 候选 0 条：起新会话 ----
	// 新会话没有历史 override，企业编号**只能**来自本次请求。
	if err := requireExplicitEnt(cfg, nil); err != nil {
		return nil, err
	}
	// 别名先查（F18）再拨号 —— 注定要被拒的请求不该先花一次 SSH 握手。
	if err := m.checkNameFree(name); err != nil {
		return nil, err
	}
	sess, err := m.newSession(cfg, module, prog, runProg, launchRef, extra, m.emit)
	if err != nil {
		// 新会话没建成，池子一个字都不动（空的宿主照旧留着 —— 它在 idle，随时可用）。
		return nil, err
	}
	if argsOverride != "" {
		sess.ArgsOverride = argsOverride
	}
	m.mu.Lock()
	if err := m.assignIdentityLocked(sess, name); err != nil {
		// 两次判断之间被人抢了名字（F18 的竞态那一支）：这条会话不能留在池外活着。
		m.mu.Unlock()
		sess.ForceExit()
		return nil, err
	}
	m.sessions[sess.ID] = sess
	m.mu.Unlock()

	// 已登记 → 广播"多了一条"（W2）。放在 unlock 之后：briefOf 要拿会话锁，
	// 而 emit 只碰事件的锁，两者不重叠，但"登记完成"这个语义必须在锁外表达。
	m.emitAdded(sess)

	// 新会话已登记 —— 这时才收口那条换了目标/换了名字的空闲宿主。
	// 它是空闲的，没有在跑的活要保护；而且它从池里移除的那一刻，它的别名与序号
	// **当场释放**（C3.7b / Q19）。
	if prev != nil {
		log.Printf("[debug] 目标切换为 %s(旧会话 %s 在 %s),新会话 %s 建成后断开旧会话",
			cfg.EnvName(), prev.ID, prev.EnvNameNow(), sess.ID)
		m.audit(prev, auditCollect)
		_ = prev.Close()
		m.Remove(prev.ID)
	}
	return sess, nil
}

// sessionCandidates 收回"这次要落的那条会话"的候选集（WP6 / Q2 / C8.1 / Q23）。
//
// 判据两条，按顺序：
//
//  1. **与本次目标同环境**（`sameTarget`：连接四元组 + 环境名）—— 这是"候选"的定义，
//     与 `decideHost` 的复用判据同一个函数（不另立一套，见 prepareSession 的说明）；
//  2. 池里那条会话**已经有名字**、而这个名字与 `--name` 给的对不上 → 整条不算候选
//     （Q23 / C2.5b：别名只在创建时给一次，"要 X 却落在叫 Y 的会话上"就是静默改名）。
//
// **没名字的会话不受第 2 条影响** —— 它正是 Q23 的"名字不存在 → 起新的"要落的那条空闲宿主
// （契约 C1.6 的①：别名登记给它，它此前无名，所以不是改名）。这一条与 `nameCompatible`
// 是同一套判据的两半，两处必须同时读：`nameCompatible` 判"这条既有会话能不能充当这次要的那条"。
//
// 返回的切片顺序按 `id` 排（`m.sessions` 是 map，顺序随机 —— 候选块与审计行必须可比）。
// 已断开（`StateExit`）的残留顺手清掉：它不该占候选（也不该占别名/序号，Q19）。
//
// **不持有 `m.mu` 返回**：`decideHost` 与 `briefOf` 都只读会话自己的锁，
// 锁序仍是 `m.mu → s.mu` 单方向（与 `Snapshot` / `Current` 一致）。
func (m *Manager) sessionCandidates(cfg *Config, name string) []*Session {
	want := strings.TrimSpace(name)
	m.mu.Lock()
	m.sweepExitedLocked()
	out := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		if stored := s.Name(); stored != "" && !nameCompatible(stored, want) {
			continue
		}
		if !s.sameTarget(cfg) {
			continue
		}
		out = append(out, s)
	}
	m.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// requireExplicitEnt 要求"这一轮要用的企业编号"是**显式给的**，不许静默沿用登录 shell 的 $TOPENT。
//
// 为什么删掉那条静默路径：ENT 决定用哪个账号连库、以及框架的据点校验过不过。而"没说"与
// "说错了"在老实现里长得一模一样 —— 作业跑在登录 shell 自带的那个企业上，输出一切正常，
// 结论却可能来自另一家企业。环境已经"一律点名"，企业没有理由保留一个隐式默认。
//
// 两个合法来源，任一即可：
//   - 本次启动显式给的 `--ent`（调用方落在 `cfg.Topent`）；
//   - 这条会话**此前**显式设过的 `tt debug topent`（会话级 override）。
//
// `prev` 只在"空闲复用"那条路上有意义：换目标/换名字会收掉旧会话另建一条，旧会话的
// override **不继承** —— 所以那种情况调用方传 prev=nil，必须重新给 `--ent`。
// productOfEnv 从整份配置里按当前环境名查产品别。
//
// 有些调用点（如 hLaunch）传进来的是**未 applySsh 的整份配置**，cfg.Product 是空的；
// 这时只能回头去 SSHs 里按环境名找。
func (c *Config) productOfEnv() string {
	name := c.EnvName()
	for i := range c.SSHs {
		if c.SSHs[i].Name == name {
			return strings.ToLower(c.SSHs[i].Product)
		}
	}
	return ""
}

func requireExplicitEnt(cfg *Config, prev *Session) error {
	// TOP GP 没有 ENT 概念（那是 T100 的"用哪个账号连库/据点校验"）。
	// 对 GP 会话要求 ENT 等于凭空加一道谁也不知道该填什么的门 —— 直接放行。
	if cfg != nil && (strings.EqualFold(cfg.Product, "gp") || cfg.productOfEnv() == "gp") {
		return nil
	}
	if cfg != nil && strings.TrimSpace(string(cfg.Topent)) != "" {
		return nil
	}
	if prev != nil {
		if strings.TrimSpace(prev.TopentOverride()) != "" {
			return nil
		}
		// 这条会话**自己**建立时定下的企业编号也算显式：它是当初 `--ent`（或 `topent`）
		// 明确给的，随会话一起存在 `cfg.Topent` 里，不是登录 shell 那个隐式值。
		if strings.TrimSpace(prev.TopentSession()) != "" {
			return nil
		}
	}
	f := target.NewFailure(target.CodeUsage,
		"必须显式指定企业编号(ENT)：tt debug start <作业> --ent <企业编号>。"+
			"若这条环境已有空闲会话，也可先 tt debug topent <编号> 再启动。"+
			"**不再**沿用登录 shell 的 $TOPENT —— 那会让「忘了点名」与「点错了」长得一样，"+
			"而 ENT 决定用哪个账号连库、以及据点校验过不过。")
	// 老写法（不带 --ent）也要报这一条：它正是这次要禁掉的静默行为，不是"新场景"。
	f.NewScenario = false
	f.Hint = "tt debug ents 可列出这台机器上的企业编号与各自账号；确认后再点名"
	return f
}

// namedHost 按**服务端原始句柄**取"这次显式指名的那条会话"（`--session` 落位之后的 id，C5.4）。
//
// 两条纪律，都是 Bug A 的一半：
//
//   - **找不到就报 `SESSION_NOT_FOUND`，绝不回落成"那就按候选再收窄一次"**。回落会把
//     "我点的那条没了"变成"落到另一条上，而输出看起来一切正常" —— 那正是这次重构开头
//     要根除的那句话，所以它必须是一条失败；
//   - **先扫一遍已断开的残留**（`sweepExitedLocked`）：点名的会话可能刚掉线，那条残留
//     还挂在 map 上但已经不占身份（Q19 / C3.7b）。扫掉之后取不到 = 真的不存在。
//
// 为什么不在 `sessionCandidates` 里按环境/名字再筛一遍：指名是**精确通道**，
// 它给出的是"哪一条"，不是"一类"。再筛一次就又变回候选集了。
func (m *Manager) namedHost(id string) (*Session, error) {
	m.mu.Lock()
	m.sweepExitedLocked()
	s := m.sessions[id]
	m.mu.Unlock()
	if s == nil {
		// 候选块给**池里现有全部会话**（C5.11 这个码下必须出现）：照抄 <ref> 那一列重发
		// 就能落到一条真的存在的会话上，比一句"不存在"有用。
		f := target.NewFailure(target.CodeNotFound,
			"点名的 SSH 会话 %s 不存在；不会退而落到池里的其它会话（那就成了替你挑一条）", id)
		f.Hint = "先运行 tt debug status 看现有会话，照抄 <ref> 那一列重发 --session"
		f.NewScenario = true // `--session` 是本次新放开的写法：这个失败今天无从发生
		f.Candidates = sessionsToCandidates(m.Snapshot())
		return nil, f
	}
	return s, nil
}

// namedHostConflict 造"点名的宿主不能用"的失败（`--session` 与 `--env` / `--name` 对不上）。
//
// 它是 `decideHost` 给出 `hostCollect` 时**指名档**的替代处置：`hostCollect` 会把这条会话
// `Close` 掉再按新的重建，而点名的会话不许这样被悄悄收掉 —— 调用方手上那个 ref/别名
// 还指着它，收掉之后"用这一条"就变成了"用别处起了一条"，而且旧的那条当场消失。
//
// 两类冲突各给各的话（同时踩两条时报环境那条，与 `target.Resolve` 的 F12 先于 F13 同序）：
//
//   - **目标不同** = F12 那一族在服务端的兜底（正常路径上 CLI 的 `target.Resolve` 已经拦下，
//     见 `internal/target` 的 `failureEnvConflict`）；服务端这一份是防线，不是重复；
//   - **别名不同** = C2.5b 的"别名只在创建时给一次"：要把一条已经叫 Y 的会话叫成 X，
//     那叫**改名**，而改名这条路是关着的。
func (m *Manager) namedHostConflict(prev *Session, cfg *Config, wantName string) *target.Failure {
	ref := m.auditRefOf(prev)
	var f *target.Failure
	if !prev.sameTarget(cfg) {
		f = target.NewFailure(target.CodeUsage,
			"点名要用的会话 %s 属于环境 %q，与这次启动的目标环境 %q 不一致；指名一条会话时环境由它决定",
			ref, prev.EnvNameNow(), cfg.EnvName())
		f.Hint = "去掉 --env（环境由 --session 那条决定），或让 --session 与 --env 指同一个环境"
	} else {
		f = target.NewFailure(target.CodeUsage,
			"点名要用的会话 %s 已经有别名 %q，不能改成 %q（别名只在创建时给一次）",
			ref, prev.Name(), strings.TrimSpace(wantName))
		f.Hint = "去掉 --name（沿用那条会话的别名），或换一条空闲会话"
	}
	f.NewScenario = true
	return f
}

// failAmbiguous 把"候选 ≥2 条"写成 F1 的失败（`SESSION_AMBIGUOUS` + 退 3 + 候选清单）。
//
// 文案与判据都来自**目标层那一处**（`internal/target` 的 `failureAmbiguousOmitted`，
// 唯一出处）：服务端这一侧不另写一句"有 N 条会话"的中文，也不另拼一份候选清单 ——
// 契约 C5.12 的九键候选只有一处定义（`sessionView(...).Candidate()`）。
//
// **无条件退 3**（不按"有没有打 --session"分档）：这个场景今天不存在（今天只有一条会话，
// 多会话 + 不给目标是本批才放开的状态），F1 的判据是"**这个场景是不是新场景**"，
// 见 `target.ExitFor` / `contract.go` 的 Failure.NewScenario。
func (m *Manager) failAmbiguous(cfg *Config, cands []*Session) *target.Failure {
	pool := target.Pool{
		Environments: []string{cfg.EnvName()},
		Sessions:     make([]target.Session, 0, len(cands)),
	}
	for _, s := range cands {
		pool.Sessions = append(pool.Sessions, sessionView(m.briefOf(s)))
	}
	// 省略目标（Selector == nil）走 `failureAmbiguousOmitted`（F1 的文案）——
	// 它自己的 Hint 就写着"用 --session <别名> 或 <ref> 指明"，与候选块配套。
	if _, f := target.Resolve(pool, target.Request{}); f != nil {
		return f
	}
	// 走不到：候选 ≥2 时 target.Resolve 必报歧义。留着是为了"哪天它改了"时有一个响，
	// 而且**绝不回落成"挑一条"**。
	return target.NewFailure(target.CodeAmbiguous,
		"有 %d 条 SSH 会话，无法确定你要哪一条", len(cands))
}

// resolveJob 连库按 gendbg 语义解析作业编号(gzzz_t:gzzz001 → gzzz002 实体程序 + gzzz005 模块)。
// 返回 (模块, 实体程序, err):err 仅在 SSH 连接/动态路径探测失败时非 nil;
// 未配置 db/查询失败/未命中 → ("","","","", nil),由调用方回退文件搜索。
func (m *Manager) resolveJob(module, job string) (mod, prog, launchRef, extra string, err error) {
	// 传一份**副本**（缺陷 Y / T5.2）:`resolveJobWith` 会把探到的动态路径写进它拿到的
	// 那一份 cfg,而这一条路拿到的正是共享的 `m.cfg` —— 一次作业解析就会把
	// "当前生效环境"的 Runtime 改掉,另一条会话随后读到的是别人的路径。
	cc := *m.cfg
	return m.resolveJobWith(&cc, module, job)
}

// resolveJobWith 同 resolveJob,但按传入 cfg(多 SSH/多区域克隆)解析。
func (m *Manager) resolveJobWith(cfg *Config, module, job string) (mod, prog, launchRef, extra string, err error) {
	if cfg.DB == nil || !reProgName.MatchString(job) {
		return "", "", "", "", nil
	}
	// **先查缓存**：命中就完全不碰 SSH/数据库（远程 ksql 冷启动一次约 3s，见 jobCache 注释）。
	// key 带上 module —— 显式模块会改变返回值（尊重指定、仅采纳实体程序），不能共用。
	// key 带上**环境名**：缓存值来自该环境的 gzzz_t/gzza_t，同一台服务器上不同环境
	// （甚至不同库）的作业表内容可以不同，只用 host|user|zone 会把两边的结果混成一份。
	ckey := cfg.SSH.Host + "|" + cfg.SSH.User + "|" + zoneOf(cfg) + "|" + cfg.EnvName() + "|" + module + "|" + strings.ToLower(job)
	m.jobMu.Lock()
	if c, ok := m.jobCache[ckey]; ok && time.Since(c.at) < 5*time.Minute {
		m.jobMu.Unlock()
		log.Printf("[launch-timing] 作业解析 %s: 缓存命中，跳过 SSH+连库", job)
		return c.mod, c.prog, c.launchRef, c.extra, nil
	}
	m.jobMu.Unlock()
	// **本地字典库优先**：本地有这条作业就直接用（零远程往返），查不到才走远程链路。
	// 与缓存同语义：本地命中不做 42r 目录校验（那需要 SSH，见下）。
	if m.LocalJobLookup != nil {
		if jr, ok := m.LocalJobLookup(cfg.EnvName(), job); ok && jr.Prog != "" {
			p := jr.Prog
			// 与远程路径同一条规范化：剥 $FGLRUN 前缀（幂等，防两个实现漂移），
			// 非 $ 开头的启动引用清空（gendbg 语义：交由选区 shell 展开的才算数）。
			jr.LaunchRef = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(jr.LaunchRef), "$FGLRUN"))
			if !strings.HasPrefix(jr.LaunchRef, "$") {
				jr.LaunchRef, jr.Extra = "", ""
			}
			launchRef, extra = jr.LaunchRef, jr.Extra
			if module != "" {
				return module, p, launchRef, extra, nil // 用户显式指定模块:尊重指定,仅采纳实体程序
			}
			// 模块码缺失或写成 "-"（gzzz005 的口径）**不能**当模块名用：远程链路对这两种
			// 都走"搜 42r"兜底（见下面 mod == "" || mod == "-" 那一支），本地这条只做加速、
			// 不做远端校验，所以要交回远程 —— 否则会拼出 <top>/erp/- 这种启动目录，
			// 启动失败且报错点跟真因（模块码缺失）看不出关系。
			if lm := strings.ToLower(strings.TrimSpace(jr.Module)); lm != "" && lm != "-" {
				return lm, p, launchRef, extra, nil
			}
		}
	}
	// 分段计时：这条链路是"点启动到弹画面"最重的一段（2026-10-23 用户反馈启动慢）。
	// 每一步都是一次**独立的远程往返**，日志里拼出耗时分布才能对症下药。
	tAll := time.Now()
	tDial, tRuntime, tDBConn, tQuery, tCheck := time.Duration(0), time.Duration(0), time.Duration(0), time.Duration(0), time.Duration(0)
	defer func() {
		log.Printf("[launch-timing] 作业解析 %s: dial=%v runtime探针=%v 连库探测=%v gzzz查询=%v 42r校验=%v 总计=%v",
			job, tDial.Round(time.Millisecond), tRuntime.Round(time.Millisecond), tDBConn.Round(time.Millisecond), tQuery.Round(time.Millisecond), tCheck.Round(time.Millisecond), time.Since(tAll).Round(time.Millisecond))
	}()
	t0 := time.Now()
	conn, err := host.Dial(cfg.SSH)
	tDial = time.Since(t0)
	if err != nil {
		return "", "", "", "", fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer conn.Close()
	// 动态路径(登录区域 → 环境脚本):按本次启动的区域探针+缓存(覆盖/多环境时不用
	// 全局 active 区域),结果写进**调用方自己那一份** cfg（缺陷 Y / T5.2:
	// 不再落到共享的 `m.cfg.Runtime` 上）——预会话模块解析/42r 校验直接用真实路径。
	// 失败即报错:T100 路径只来自登录动态获取,无静态配置可回退
	t0 = time.Now()
	if _, err := m.ensureRuntimeEnvOn(conn, cfg); err != nil {
		return "", "", "", "", err
	}
	tRuntime = time.Since(t0)
	// 显式连接(SSH 页 dbConn 引用)连库解析作业;失败静默,由调用方回退文件搜索
	t0 = time.Now()
	d, err := resolveDBRun(conn, cfg)
	tDBConn = time.Since(t0)
	if err != nil {
		return "", "", "", "", nil
	}
	t0 = time.Now()
	jr, err := dbResolveJob(conn, d, job)
	tQuery = time.Since(t0)
	if err != nil || jr.Prog == "" {
		return "", "", "", "", nil
	}
	p := jr.Prog
	// gendbg 原版语义:gzza004 的 $变量 由选区后 shell 展开为权威 42r 路径(标准/客制都覆盖)
	if !strings.HasPrefix(jr.LaunchRef, "$") {
		jr.LaunchRef, jr.Extra = "", ""
	}
	launchRef, extra = jr.LaunchRef, jr.Extra
	if module != "" {
		m.cacheJob(ckey, module, p, launchRef, extra)
		return module, p, launchRef, extra, nil // 用户显式指定模块:尊重指定,仅采纳实体程序
	}
	mod = strings.ToLower(jr.Module)
	t0 = time.Now()
	if jr.Module == "" || jr.Module == "-" || !modHas42r(conn, cfg.TopDirActual(), mod, p) {
		// 模块码缺失或目录对不上:全模块搜索实体程序的 42r;0/多命中留给会话内兜底报错
		if cands := searchModule42r(conn, cfg.ModuleRootsActual(), p); len(cands) == 1 {
			mod = cands[0]
		} else {
			mod = ""
		}
	}
	tCheck = time.Since(t0)
	m.cacheJob(ckey, mod, p, launchRef, extra)
	return mod, p, launchRef, extra, nil
}

// cacheJob 写一行解析缓存；懒初始化（Manager 构造时不必记得 new 这个 map）。
func (m *Manager) cacheJob(key, mod, prog, launchRef, extra string) {
	m.jobMu.Lock()
	if m.jobCache == nil {
		m.jobCache = make(map[string]cachedJob)
	}
	m.jobCache[key] = cachedJob{mod: mod, prog: prog, launchRef: launchRef, extra: extra, at: time.Now()}
	m.jobMu.Unlock()
}

// LaunchReplay 接口日志重放调试:等价 T100 日志内嵌的 `r.dg <作业> '<req>' '<rsp>'`。
// 作业取 wsfa012(gzja_t 服务→程序的解析结果),再走 gzzz_t 解析实体程序+模块;
// 报文文件被清理时用 CLOB 内容落到服务器临时文件再重放。
// 会话复用规则与普通启动一致:同目标空闲宿主直接复用。
// LaunchReplay 用日志报文重放调试。reqOverride 非空 = 界面里改过的入参(落临时文件后作为 reqPath)。
// fglserver 非 nil = 本轮 FGLSERVER 覆盖(语义见 Server.launchCfg 与 ScreenTarget)。
// name 是 `--name` 给的别名（Bug B：`wsdebug` 与 `debug start` 同样是**起会话**的命令，C3.5）：
// 它原样交给 `prepareSession`，于是别名在会话登记那一刻成为服务端的事实（C5.3 的 `name` 键），
// 从此参与寻址、当幂等键（Q23）。空 = 未命名，逐字等于今天的行为。
// 返回的 warn 非空表示这次重放用的是入库的截断文本,结果可能不可信(见 WriteReplayFiles)。
func (m *Manager) LaunchReplay(item *WSLogItem, content *WSLogContent, reqOverride string, fglserver *string, name string) (*Session, string, error) {
	// 本轮启动用的配置快照:只有"单轮 FGLSERVER 覆盖"会改它。
	// 直接改 m.cfg 会把一次重放的接线留在内存配置里 —— 下一轮(哪怕只是普通启动)
	// 都还带着它,而画面去哪恰恰是静默的(见 ScreenTarget)。
	cfg := m.cfg
	if fglserver != nil {
		cc := *m.cfg
		cc.FGLServer = strings.TrimSpace(*fglserver)
		cfg = &cc
	}
	job := strings.TrimSpace(item.Job)
	if job == "" {
		return nil, "", fmt.Errorf("该日志没有关联作业编号(wsfa012 为空),无法重放")
	}
	module, runProg, launchRef, extra := "", "", "", ""
	if mod2, prog2, ref2, extra2, rerr := m.resolveJobWith(cfg, "", job); rerr != nil {
		return nil, "", rerr
	} else if prog2 != "" {
		module, runProg, launchRef, extra = mod2, prog2, ref2, extra2
	}
	conn, err := host.Dial(cfg.SSH)
	if err != nil {
		return nil, "", fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer conn.Close()
	reqPath, rspPath, replayWarn, err := WriteReplayFiles(conn, item, content, reqOverride)
	if err != nil {
		return nil, "", err
	}
	// 路径进 shell 单引号,剔除单引号防注入。
	// 对齐 r.dg:报文文件追加在标准参数(BBDL 会话标记等)之后,
	// cl_wss_init 依 BBDL 标记识别调试模式并重建 g_argv;只传报文会导致 g_argv 为空 → -1
	q := func(p string) string { return "'" + strings.ReplaceAll(p, "'", "") + "'" }
	args := strings.ReplaceAll(cfg.LaunchArgs, "{prog}", job)
	args += " " + q(reqPath)
	if rspPath != "" {
		args += " " + q(rspPath)
	}
	sess, err := m.prepareSession(cfg, module, job, runProg, launchRef, extra, args, name)
	if err != nil {
		return nil, "", err
	}
	// 新的一轮运行:清空上一轮攒下的源码副本(见 srcmirror.go);
	// "已镜像"记录必须跟着一起清(复用的宿主会话会把它带进下一轮)。
	// 同 launchWith:只清**这条会话自己的**那棵子树（缺陷 X / T5.1）。
	sess.inheritMirrorState()
	clearMirrorForSession(cfg.DataDir, sess.mirrorScope())
	sess.resetMirrored()
	// 用了入库的截断文本重放:说清楚,别让人把"程序一路退出"当成 bug 去查
	if replayWarn != "" {
		sess.emitEvent(Event{Type: "log", Text: "⚠ " + replayWarn})
	}
	if runProg != "" && runProg != job {
		sess.emitEvent(Event{Type: "log", Text: fmt.Sprintf("重放调试:作业 %s → 实体程序 %s(gzzz_t)", job, runProg)})
	}
	if reqOverride != "" {
		sess.emitEvent(Event{Type: "log", Text: fmt.Sprintf("重放使用界面修改后的入参(%d 字节),已写入 %s", len(reqOverride), reqPath)})
	}
	// 记下响应会落到哪:程序退出后要把它读回来回显(见 loadReplayResponse)。
	// 响应文件是程序退出时才写的,所以只能等跑完再读 —— 停在入口/断点时它还不存在。
	sess.mu.Lock()
	sess.replayRspPath = rspPath
	sess.mu.Unlock()
	sess.emitEvent(Event{Type: "log", Text: fmt.Sprintf("重放 %s:fglrun -d %s %s", item.Service, runProgOr(job, runProg), args)})
	return sess, replayWarn, nil
}

func runProgOr(job, runProg string) string {
	if runProg != "" {
		return runProg
	}
	return job
}

// Get 取会话
func (m *Manager) Get(id string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

// Current 返回当前唯一会话(无则 nil)
func (m *Manager) Current() *Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		return s
	}
	return nil
}

// Snapshot 会话列表快照
//
// 键集是契约 C5.1 / C5.2 / C5.3 定死的那一份（`session` 块的形状）：
// **十一个恒定键**（`id`/`ref`/`display`/`name`/`env`/`ent`/`n`/`kind`/`prog`/`state`/`epoch`）
// 加一批按 `omitempty` 出现的快照键。恒定键**取不到就给空串/零值，不省键** ——
// 前端与 CLI 按这份形状解包，少一个键它们就得写第二套兜底。
//
// 五个新键的语义（Q22：纯新增、不改旧键）：
//   - `ref`    规范 id `环境#序号`，**生命周期内恒定不变**（C2.0 / C5.4）——身份，不是快照；
//   - `display` 人读形态 `环境/作业#序号`，作业段是快照，会随 `SetRun` 漂（C5.4）；
//   - `name`   别名，空串 = 未命名（C2.5b）；
//   - `ent`    **当前**生效的企业编号，会随 `tt debug topent` 变（C2.5c）——快照，不是地址；
//   - `n`      序号：同一环境下的第 N 条（C2.2 / Q24）——身份，建立时分配、之后不变；
//   - `epoch`  世代号（C5.3 / F15）——见 `Session.Epoch` 的语义。
type SessionBrief struct {
	ID string `json:"id"`
	// Ref 是**规范 id**：`环境#序号`（C2.1.2 / C5.4）。只由不可变部分拼 ——
	// 企业编号与作业段都不在里面（C2.0：可变字段不进地址）。
	Ref string `json:"ref"`
	// Display 是人读形态：`环境/作业#序号`（C5.4）。比 ref 多一段**当前作业快照**，
	// 所以作业中途变了它就会变 —— 要恒定用 ref，要好读用 display，两者不混用。
	Display string `json:"display"`
	// Name 是别名（Q17 / C2.5b）。空串 = 未命名（**不省键**，C5.1 明写取不到给空串）。
	Name string `json:"name"`
	// Ent 是该会话**当前**生效的企业编号（C5.3 / C2.5c）：shell 回读值优先，
	// 取不到退到本会话显式给的值；空 = 未取到。**它是快照**，`SetTopent` 会改它。
	Ent string `json:"ent"`
	// N 是序号（同一环境下的第 N 条，从 1 起；C2.2 / Q24）。建立时分配，之后不变。
	N int `json:"n"`
	// Kind 是会话类型（C5.5）：本契约只寻址 SSH 会话，所以恒为 `ssh`
	// （界面会话不是第二个目标，它是这条会话的属性，走信封的 `ui` 块）。
	Kind     string  `json:"kind"`
	Module   string  `json:"module"`
	Prog     string  `json:"prog"`
	RunProg  string  `json:"runProg,omitempty"`
	State    string  `json:"state"`
	Env      string  `json:"env,omitempty"` // 会话所属环境名(设置页 envs)
	Started  bool    `json:"started"`
	File     string  `json:"file,omitempty"`
	Line     int     `json:"line,omitempty"`
	Func     string  `json:"func,omitempty"`
	Reason   string  `json:"reason,omitempty"`
	Holding  float64 `json:"holdingSeconds"`
	Breaks   int     `json:"breakpoints"`
	Watchdog int     `json:"watchdogSeconds"`
	// WatchdogDisabled 这一轮**真的**不会自动放行（运行时的口径：`<= 0` 即不启定时器）。
	// 与 `config.WatchdogDisabled`（只把负数当“配错”）故意不同：配置里的 0 会被补成
	// 默认 1800，而运行时的 0 是“显式关”。给前端的就是“会不会触发”这个事实。
	WatchdogDisabled bool `json:"watchdogDisabled,omitempty"`
	// 停站现场就是交互语句(INPUT/MENU/DISPLAY ARRAY…):程序把控制权交给了界面。
	// 零成本判定(只看停站那一行的源码文本),供 AI 直接分支,不必再读源码做预测。
	WaitingForUser bool   `json:"waitingForUser,omitempty"`
	WaitingKind    string `json:"waitingKind,omitempty"`
	// SilentSeconds:运行态下距最近一次协议输出的秒数 —— 判断"跑了很久却毫无动静"
	// (可能就是停在界面上等用户,也可能是慢查询)。停站态为 0。
	SilentSeconds float64 `json:"silentSeconds,omitempty"`
	// Mode 谁在驾驶:solo(纯人工)| collab(协作,AI 主导)
	Mode string `json:"mode"`
	// Epoch 是世代号（C5.3 / F15）。空 = 没有代（不该出现：登记时必发一个）。
	Epoch string `json:"epoch"`

	// 它不是 `kind` —— `kind` 已经是 `ssh`（C5.5）。

	// Inflight 正在执行的那条命令(空闲为 nil)—— 界面上"AI 正在执行 continue(已 12s)"
	Inflight *InflightInfo `json:"inflight,omitempty"`
}

// refOf / displayOf 拼身份的两个字符串（C5.4 的三分工）。
//
// 序号还没分配（0：只可能出现在手工造的会话上）时给**空串**，绝不编一个 `#0` ——
// `ref` 是恒定不变的地址，一个错的地址比一个空字段坏得多（C2.0 / C5.4。
// `internal/target` 的 `CanonicalRef` 在服务端没给 ref 时会退回原始句柄，那是它的兜底，
// 不是让服务端去编）。
func refOf(env string, n int) string {
	if strings.TrimSpace(env) == "" || n <= 0 {
		return ""
	}
	return target.RefOf(env, n)
}

func displayOf(env, prog string, n int) string {
	if refOf(env, n) == "" {
		return ""
	}
	return target.Session{Env: env, Prog: prog, N: n}.DisplayForm()
}

func (m *Manager) Snapshot() []SessionBrief {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]SessionBrief, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, m.briefOf(s))
	}
	return out
}

// briefOf 把一条会话投影成它的信封形状（C5.1 的 `session` 块）。
//
// `Snapshot` 与**候选清单**（C5.12 的九键）共用这一份投影 —— 候选是 `session` 的
// 真子集（C5.4b），投影写成两处必然漂，而漂的那一处恰好是"照着候选重发一枪"要用的字段。
//
// 调用方**不必**持有 `m.mu`：`briefOf` 只读会话自己的锁与不可变的 `m.cfg`。
func (m *Manager) briefOf(s *Session) SessionBrief {
	cur := s.Cur()
	env, n := s.EnvName(), s.Seq()
	return SessionBrief{
		ID: s.ID,
		// 身份（C5.1 末段：只有 id / ref / n 是身份，判定身份只许用这三个）
		Ref: refOf(env, n), Display: displayOf(env, s.Prog, n),
		N: n, Kind: target.KindSSH, Epoch: s.Epoch(),
		// 快照（会变：name 只在创建时给，ent 随 topent 改）
		Name: s.Name(), Ent: s.CurrentEnt(),
		Module: s.Module, Prog: s.Prog, RunProg: s.RunProg,
		State: string(s.State()), Env: s.EnvName(), Started: s.Started(), File: cur.File, Line: cur.Line,
		Func: cur.Func, Reason: cur.Reason,
		Holding:          s.HoldingSeconds(),
		Breaks:           len(s.Breakpoints()),
		Watchdog:         m.cfg.WatchdogSeconds,
		WatchdogDisabled: m.cfg.WatchdogSeconds <= 0,
		WaitingForUser:   cur.WaitingForUser,
		WaitingKind:      cur.WaitingKind,
		SilentSeconds:    s.SilentSeconds(),
		Mode:             s.Mode(),
		Inflight:         s.Inflight(),
	}
}

// candidatesOf 把若干条会话投影成 C5.12 的候选九键（**唯一出处**是 `internal/target`）。
//
// 顺序：先按 `id` 排一道（`m.sessions` 是 map，顺序随机），**定稿的定序规矩在
// `internal/target` 的 `sortCandidates`**（"序号 → 原始句柄"，C5.14）—— 歧义那几条
// 失败的候选是经 `target.Resolve` 造出来的，所以最终顺序由它说了算；这里这一道保证
// 调用方在任何路径下拿到的都是**确定的**顺序（同一个失败两次跑出的文案必须一样）。
func (m *Manager) candidatesOf(sessions []*Session) []target.Candidate {
	out := make([]target.Candidate, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, sessionView(m.briefOf(s)).Candidate())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---------- 会话身份：序号 / 别名 / 世代（契约 C5.3 / C2.2 / Q19 / Q23） ----------

// ErrNameTaken 是 F18：要登记的别名已经被池子里另一条**活着的**会话占着。
//
// 它判在**注册那一刻**：Q23 的三种结局（空闲复用 / 忙碌报错 / 起新的）在起会话那条路上
// 先判一次（见 `internal/debug/api.go` 的 `beginNamedLaunch`），这里兜住两次判断之间的
// 竞态 —— 两个同名请求同时在飞时，晚的那一个必须报错，**不许**把名字从别人手里抢过来。
type ErrNameTaken struct {
	Name     string
	Occupant string // 占用者的服务端原始句柄（调用方用它取候选行，C5.12）
}

func (e *ErrNameTaken) Error() string {
	return fmt.Sprintf("别名 %q 已被会话 %s 占用；换一个名字，或先结束那条会话", e.Name, e.Occupant)
}

// SweepExited 把已断开（`StateExit`）的会话从池子里清掉。
//
// 为什么本批需要它（Q19 / C3.7b）：**别名与序号都挂在会话对象上** —— 会话从池子里消失，
// 别名立刻可以被别人用。而"断开"到"被清掉"之间原本有一段窗口（只有 `prepareSession` /
// `CreateSessionOn` 顺手扫），起会话时若不先扫一遍，一条早就断了的会话会继续占着别名，
// 于是"重起同名被拒"，报的还是"别名已被 <ref> 占用"而 `<ref>` 指向一条早没了的会话 ——
// C3.7b 点名的正是这个极难查的故障。
//
// 判据锚在"**池子里还有没有它**"，与"人敲没敲 quit"无关：SSH 掉线 / 进程被杀 / 断电
// 都会走 `forceExit` → `StateExit`，于是同一套判据覆盖异常结束。
func (m *Manager) SweepExited() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sweepExitedLocked()
}

func (m *Manager) sweepExitedLocked() {
	for id, s := range m.sessions {
		if s.State() == StateExit {
			delete(m.sessions, id) // 已断开的残留：资源已释放，身份（含别名）随之释放
		}
	}
}

// allocSeqLocked 在同一环境下取"**尚未被占用的最小正整数**"（C2.2 / C2.3 / Q16 / Q24），从 1 起。
//
// 企业编号**不参与**：按（环境, 企业编号）自增会让两条会话拿到同一个 `环境#序号`
// （企业 99 的 #1 与企业 88 的 #1 都叫 `恒烁正式区#1`），地址当场撞车 —— 2.1.3 写死了理由。
//
// 序号**不单独记账**：它按池子里活着的会话现算，所以"会话消失 → 序号释放"是天然的，
// 与别名同一条性质（Q19 的同源口径）。已断开（`StateExit`）的残留不算占用者。
func (m *Manager) allocSeqLocked(env string) int {
	// 规则住在 `internal/session`：这里只把池子拍平成它要的形状。
	slots := make([]sessionid.Slot, 0, len(m.sessions))
	for _, s := range m.sessions {
		slots = append(slots, sessionid.Slot{
			Env:    s.EnvName(),
			N:      s.Seq(),
			Exited: s.State() == StateExit,
		})
	}
	return sessionid.NextSeq(env, slots)
}

// identityOwnerLocked 找别名的占用者：**整个会话池、不区分大小写**（C2.5 的查重口径）。
//
// 与 `internal/target` 的 `ResolveName` 同一个范围与同一条查重规矩（`SameAlias`）：
// 起会话时的"空闲复用 / 忙碌报错"用它，登记的撞车判定也用它，两处必须同源 ——
// 否则会出现"起的时候说没撞、找的时候找到两条"。
func (m *Manager) identityOwnerLocked(name string) *Session {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	for _, s := range m.sessions {
		if s.State() == StateExit {
			continue
		}
		if s.Name() != "" && target.SameAlias(s.Name(), name) {
			return s
		}
	}
	return nil
}

// checkNameFree 在**建连接之前**问一句"这个名字能不能给"（F18）。
//
// 顺序是刻意的：注定要被拒的请求不该先拨一次 SSH 再拒。竞态由
// `assignIdentityLocked` 里的同一次判断兜住（两处都在 m.mu 下，判据只有一份）。
func (m *Manager) checkNameFree(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sweepExitedLocked()
	if owner := m.identityOwnerLocked(name); owner != nil {
		return &ErrNameTaken{Name: name, Occupant: owner.ID}
	}
	return nil
}

// assignNameLocked 把别名登记给一条会话（空名字 = 什么都不做）。
func (m *Manager) assignNameLocked(sess *Session, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if owner := m.identityOwnerLocked(name); owner != nil && owner != sess {
		return &ErrNameTaken{Name: name, Occupant: owner.ID}
	}
	sess.mu.Lock()
	sess.name = name
	sess.mu.Unlock()
	return nil
}

// nameSession 给一条**已经存在、还没名字**的会话补一个别名。
//
// 用途只有一处：Q23 的"名字不存在 → 起新的"在**复用宿主**形态下的落点 ——
// 池子里没有叫 X 的会话，而恰好有一条空闲的同目标宿主，于是这一轮就落在这条宿主上，
// X 就是"这条会话叫什么"。它此前没有名字，所以这不是改名（C2.5b 禁的是改名）。
// NameSession 把别名登记给一条**此前无名**的会话。
//
// 为什么要导出：`hSessionSwitch`（`debug env --name`）也要走这条 —— 命令要能
// "在这个环境里建一条叫 X 的空闲会话"，而池里可能已经有一条**没名字**的空闲宿主，
// 把名字登记给它比重新登录一次便宜。它此前无名，所以不是改名（C2.5b 禁的是改名）。
func (m *Manager) NameSession(sess *Session, name string) error {
	return m.nameSession(sess, name)
}

func (m *Manager) nameSession(sess *Session, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sweepExitedLocked()
	return m.assignNameLocked(sess, name)
}

// assignIdentityLocked 给一条**刚建立**的会话分配身份（C5.3 的四键：n / name / epoch，ref 由 n 拼）。
//
// 顺序是刻意的：先查别名（撞车就整条不成立，调用方按 F18 报错），再发序号与世代。
// 别名与序号都只在这里发一次，之后**不再重算** —— 序号恒定的依据（C2.0）就在这一句上。
// 调用方必须持有 `m.mu`。
func (m *Manager) assignIdentityLocked(sess *Session, name string) error {
	if err := m.assignNameLocked(sess, name); err != nil {
		return err
	}
	seq := m.allocSeqLocked(sess.EnvName()) // 按**环境**自增（Q24），企业编号不参与
	sess.mu.Lock()
	sess.seq = seq
	if sess.epoch == "" { // NewSession 已经发过第一代；手工造的会话在这里补一个
		sess.epoch = nextSessionEpoch()
	}
	sess.mu.Unlock()
	// 登记完就是一条**有身份的会话**：审计行在这里记（**所有**造会话的路径都过这一步，
	// 所以"建立"只需要一个落点）。它不必等调用方把它放进池子 —— 身份齐了，ref 就定了（C2.0）。
	m.audit(sess, auditCreate)
	return nil
}

// nameCompatible 判"这条既有会话能不能充当这次要的那条"（复用判据的第二半）。
//
// `want` 为空 = 这次没点名要别名 → 既有会话叫什么都能用（沿用今天的复用语义，H2）。
// `want` 非空 = 只许落在"还没名字"或"正好叫这个名字"的会话上：别名**只在创建时给一次**
// （C2.5b），把 X 挂到一条已经叫 Y 的会话上就是静默改名 —— 而 Y 那个句柄还在别人手上。
// 名字对不上时不复用：按"换目标"那条路收掉旧的、按新名字开一条（Q23 的"起新的"）。
func nameCompatible(stored, want string) bool {
	if strings.TrimSpace(want) == "" {
		return true
	}
	if strings.TrimSpace(stored) == "" {
		return true
	}
	return target.SameAlias(stored, want)
}

// hostDecision 是 `prepareSession` 对"池里那条既有会话"的三种处置。
type hostDecision int

const (
	// hostReuse：同目标 + 空闲 + 别名不冲突 → 复用宿主（免重新登录）。
	hostReuse hostDecision = iota
	// hostCollect：**空闲**的宿主，但换了目标（环境）或换了名字 → 收口它再按新的重建。
	// 空转的宿主没什么可丢的；它从池里消失的那一刻，别名与序号当场释放（C3.7b / Q19）。
	hostCollect
	// hostBusy：同目标而那条会话**正在跑 / 已停站** → 报错，**不许悄悄收掉它**。
	hostBusy
)

// decideHost 是"既有会话怎么处置"的**唯一判据**（单一常驻会话的复用规则 + Q23 的别名那一半）。
//
// 写成纯函数是为了让它可测：这三种处置各自的分支都要真拨 SSH 才走得到（除了复用），
// 而"什么时候敢收掉别人的会话"恰恰是最不该靠碰运气验的一条。
//
// 三条规矩，按顺序（**`wantName` 为空时逐字等于今天的行为**，这是 H2 的落点）：
//
//  1. **换了目标**（环境/连接不同）→ 收口重建。今天的原话：即使连接可复用也不复用 ——
//     否则会"换了环境却还连着旧企业/旧账号"。这一条今天就不看状态（收口会先中断）。
//  2. **同目标但不在空闲**（运行中 / 已停站）→ **报错**。今天那句"已存在调试会话,请先结束当前调试"
//     就是它；别名对不上也不改变它（要抢名字先停掉那条会话，F20 的 hint 同一条口径）。
//  3. **同目标 + 空闲** → 别名不冲突就复用；**别名对不上**（要 X 而它叫 Y）则收口重建 ——
//     它是空闲的，没有在跑的活要保护，而别名只能在创建时给一次（C2.5b）。
func decideHost(prev *Session, cfg *Config, wantName string) hostDecision {
	if prev == nil {
		return hostCollect
	}
	if !prev.sameTarget(cfg) {
		return hostCollect
	}
	if prev.State() != StateIdle {
		return hostBusy
	}
	if !nameCompatible(prev.Name(), wantName) {
		return hostCollect
	}
	return hostReuse
}

// Remove 移除已结束的会话。
//
// 它**一定广播** `session.removed`（W2）：树的另一半（删除）没有别的信号可用。
// 知道原因（close / exit / restart / switch）的调用方请用 `RemoveAs`，事件里会带上它。
func (m *Manager) Remove(id string) {
	m.RemoveAs(id, "removed")
}

// RemoveAs 移除会话并广播原因（W2 的 reason 取值：close|exit|restart|switch|removed）。
//
// 两步都要：先从池里删（`GET /api/sessions` 立刻不再列它），再广播（在删之后发 ——
// 收到的人若去 GET 它，本来就该 404，这与"它已经不在了"是同一件事）。
func (m *Manager) RemoveAs(id, reason string) {
	if id == "" {
		return
	}
	m.mu.Lock()
	_, existed := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if existed {
		m.emitRemoved(id, reason)
	}
}

// CreateSessionOn 新建并登记一个纯宿主会话(module/prog 为空,登录后回到 idle),
// 供「会话重启/切换/连接」在指定目标环境上重连;调用方随后驱动 BootIdle。
//
// purpose 是这条 SSH 会话的用途（W1：debug | record | replay，空 = debug）。
// **只在建会话时登记一次**，之后不可改 —— 要换用途就换一条会话。
//
// name 是这条新会话要不要继承的别名：
//   - **会话重启**（`/api/sessions/{id}/restart`）传旧会话的名字 —— 重启是"同一条逻辑会话
//     重新登录"（F15 的前提就是"环境与序号都对、只是换了代"）：旧会话已从池里移除、
//     别名当场释放，这里就地把同一个名字重新登记给新会话，于是调用方手上的别名/ref 继续有效；
//   - **连到该环境**（`/api/sessions/switch`）传空 —— 那是新建一条会话，别名随旧会话消失
//     即释放（C3.7b / Q19），要名字就再给一次。
func (m *Manager) CreateSessionOn(cfg *Config, name string) (*Session, error) {
	m.mu.Lock()
	m.sweepExitedLocked()
	m.mu.Unlock()
	if err := m.checkNameFree(name); err != nil {
		return nil, err
	}

	sess, err := m.newSession(cfg, "", "", "", "", "", m.emit)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	if err := m.assignIdentityLocked(sess, name); err != nil {
		m.mu.Unlock()
		sess.ForceExit()
		return nil, err
	}
	m.sessions[sess.ID] = sess
	m.mu.Unlock()
	m.emitAdded(sess) // W2：已登记 → 广播"多了一条"
	return sess, nil
}

// SourcePreview 会话建立前预取源码:读母版消除前端启动期空白,顺带落本地镜像。
//
// 模块可以不填:和会话启动一样按 p_link(t_prog_link) 解析作业实际所属模块。
//
// 它跑在会话之前,所以作用域只能是**目标**（环境段,`srcConnScopeSeg`）——
// 与从前的落点逐字相同;会话建起来之后由该会话自己的作用域接着落（见 mirrorScope）。
func (m *Manager) SourcePreview(module, prog string) (*SourceFile, error) {
	prog = strings.TrimSuffix(strings.TrimSpace(prog), ".4gl")
	if prog == "" {
		return nil, fmt.Errorf("需要 prog 参数")
	}
	cfg, cfgErr := m.sourceCfgFor("")
	if cfgErr != nil {
		return nil, cfgErr
	}
	mods := sourceModuleCandidates(cfg.sourceModuleOf, module, prog)
	dvmFile := prog + ".4gl"
	if strings.TrimSpace(module) != "" {
		dvmFile = strings.TrimSpace(module) + "_" + prog + ".4gl"
	}
	// 一次调用只认一份配置快照:下面几处都用它,免得中途被别的请求改掉的共享态混进来。
	scope := srcConnScopeSeg(cfg)
	var out *SourceFile
	err := m.withSrcConn(cfg, func(_ *host.SSHConn, cl *sftp.Client) error {
		var lastErr error
		for _, p := range sourceCandidatePaths(cfg.ModuleRootsActual(), mods, dvmFile) {
			f, err := cl.Open(p)
			if err != nil {
				lastErr = err
				continue
			}
			var mt time.Time
			if fi, err := f.Stat(); err == nil {
				mt = fi.ModTime()
			}
			data, err := io.ReadAll(f)
			f.Close()
			if err != nil {
				lastErr = err
				continue
			}
			out = &SourceFile{
				DVMFile: dvmFile, Path: p, Content: string(data), ModTime: mt,
				LocalPath: writeMirrorScoped(cfg.DataDir, scope, p, data),
			}
			return nil
		}
		return fmt.Errorf("源码未找到(%s)：p_link 解析模块 %v 都没命中: %w", dvmFile, mods, lastErr)
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// sourceCfg 取"会话外读取"要用的那份配置快照（缺陷 Y / T5.2）。
//
// 为什么是副本而不是 `m.cfg` 本身:探针会把动态路径写进拿到的那一份 cfg,
// 而 `m.cfg` 是**共享**的 —— 一条会话的启动或一次源码读取都会改掉另一条看到的路径。
func (m *Manager) sourceCfg() *Config {
	c, err := m.sourceCfgFor("")
	if err != nil {
		return &Config{}
	}
	return c
}

// sourceCfgFor 同 sourceCfg,但可以点名环境(TG 只有一个产品,不再分 T100/GP)。
//
// m.cfg 是启动时的骨架:SSH/workspace/custRoot 要 ApplyDefaultEnv 才合并进来。
// 以前 source 直接拷 m.cfg,拿到的 SSH 端口是 0,真机上就是 `dial tcp :0`。
func (m *Manager) sourceCfgFor(env string) (*Config, error) {
	cc := *m.cfg
	name := strings.TrimSpace(env)
	if name == "" {
		name = strings.TrimSpace(cc.envName)
	}
	// TG 通常只配一个环境：没点名就用它（服务启动时那份配置不一定带 activeEnv）。
	if name == "" && len(cc.SSHs) == 1 {
		name = strings.TrimSpace(cc.SSHs[0].Name)
	}
	if name != "" {
		found := ""
		for i := range cc.SSHs {
			if strings.EqualFold(strings.TrimSpace(cc.SSHs[i].Name), name) {
				found = cc.SSHs[i].Name
				break
			}
		}
		if found == "" {
			return nil, fmt.Errorf("环境 %s 不存在：tg env list 看环境名（配置里现有的：%s）", name, strings.Join(envNamesOf(cc.SSHs), "、"))
		}
		cc.envName = found
	}
	cc.ApplyDefaultEnv()
	return &cc, nil
}

// envNamesOf 现有环境名列表（错误文案里给候选，免得"环境不存在"后面还得再查一次）。
func envNamesOf(sshs []config.NamedSsh) []string {
	out := make([]string, 0, len(sshs))
	for _, e := range sshs {
		if n := strings.TrimSpace(e.Name); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// sourceModuleOf 按本地字典里的 p_link(t_prog_link) 查作业实际属于哪个模块。
func (c *Config) sourceModuleOf(prog string) (string, bool) {
	// serve 启动时没有 --env，envName 是空的；只有一个环境时按它算，
	// 否则 gp 的 t_prog 查询永远命中不了（envName 与 SSHs[i].Name 对不上）。
	if e := c.ActiveEnvOf(); e != nil {
		return dictlib.LookupModule(*e, prog)
	}
	return "", false
}

// sourceErrNoEnv 指明是哪一步没配,而不是一句"路径未知"。
func sourceErrNoEnv(cfg *Config, env string) error {
	name := strings.TrimSpace(env)
	if name == "" && cfg != nil {
		name = strings.TrimSpace(cfg.EnvName())
	}
	if name == "" {
		return fmt.Errorf("没有可用的环境：先用 tg env list 看环境名，再 tg debug source <文件> --env <环境>")
	}
	return fmt.Errorf("环境 %s 没有可用的 SSH 连接：去设置页「SSH 服务器」补地址/端口/账号（当前配置里这份是空的）", name)
}

// ---------- 会话外只读能力(AI/CLI 信息通道;不经会话,独立短连接,白名单限登录区源码目录) ----------

// SourceResult 会话外源码读取结果(带行段裁剪,避免大文件整读灌满 AI 上下文)。
type SourceResult struct {
	SourceFile
	From int `json:"from"` // 1-based 返回行段起点
	To   int `json:"to"`   // 1-based 返回行段终点(含)
	All  int `json:"all"`  // 文件总行数
}

// maxSourceFileBytes 会话外读取的文件大小上限(源码一般远小于此)。
const maxSourceFileBytes = 4 << 20

// sourceRootsUnknownErr 源码根取不到时的错误。
//
// GP 的源码根就是环境里配的 $TOP 与 $CUST（见 ModuleRootsActual），
// 不猜目录、也不回落到某个默认路径 —— 那会把"读了另一台机上的文件"伪装成成功。
func sourceRootsUnknownErr(cfg *Config) error {
	env := ""
	if cfg != nil {
		env = strings.TrimSpace(cfg.EnvName())
	}
	if env == "" {
		return fmt.Errorf("这个环境没有配 $TOP（workspace）：去设置页「SSH 服务器」填上，例如 /u1/<实例>/tiptop")
	}
	return fmt.Errorf("环境 %s 没有配 $TOP（workspace）：去设置页「SSH 服务器」填上，例如 /u1/<实例>/tiptop；"+
		"客制源码根 $CUST（custRoot）没填也不要紧，会只查 $TOP", env)
}

// ReadSourceStandalone 读取 GP 的 4GL 源码（$TOP / $CUST 两棵树,只读）。
//   - path 非空:直接按路径读(白名单校验两棵树之内);
//   - path 为空:file 按 <root>/<模块>/4gl/<程序>.4gl 解析。
//     模块可以不填、也可以填错 —— 两种都先按本地字典的 p_link(t_prog_link)
//     查作业实际属于哪个模块,查到的排在用户填的前面试(与启动会话同一套规则)。
//   - from/to(1-based):返回行段(from<=0 从头,to<=0 到底),行文本超长截断防注入超长行。
//
// env 为空时用服务当前环境;点名环境时按该环境合并 SSH/workspace 后再连。
// 走会话外复用连接(srcmirror.go),顺带把整份源码落本地镜像并在结果里带出 LocalPath。
func (m *Manager) ReadSourceStandalone(env, module, file, path string, from, to int) (*SourceResult, error) {
	if path == "" && file == "" {
		return nil, fmt.Errorf("需要 file 或 path 参数")
	}
	var (
		data []byte
		mt   time.Time
	)
	// 一次调用只认一份配置快照（缺陷 Y / T5.2）:探到的动态路径写在这一份上,
	// 共享的 `m.cfg` 一字不动。
	cfg, cfgErr := m.sourceCfgFor(env)
	if cfgErr != nil {
		return nil, cfgErr
	}
	if strings.TrimSpace(cfg.SSH.Host) == "" {
		return nil, sourceErrNoEnv(cfg, env)
	}
	scope := srcConnScopeSeg(cfg)
	err := m.withSrcConn(cfg, func(_ *host.SSHConn, cl *sftp.Client) error {
		roots := cfg.ModuleRootsActual()
		if len(roots) == 0 {
			return sourceRootsUnknownErr(cfg)
		}
		readOne := func(p string) ([]byte, time.Time, error) {
			f, err := cl.Open(p)
			if err != nil {
				return nil, time.Time{}, err
			}
			defer f.Close()
			var t time.Time
			if fi, err := f.Stat(); err == nil {
				t = fi.ModTime()
			}
			d, err := io.ReadAll(f)
			return d, t, err
		}
		// 路径白名单:必须在登录区源码目录之内
		inRoots := func(p string) bool {
			for _, root := range roots {
				if strings.HasPrefix(p, root+"/") {
					return true
				}
			}
			return false
		}
		if path != "" {
			if !inRoots(path) {
				return fmt.Errorf("路径不在登录区源码目录内: %s", path)
			}
			d, t, err := readOne(path)
			if err != nil {
				return err
			}
			data, mt = d, t
			return nil
		}
		var lastErr error
		prog := strings.TrimSuffix(sourceBaseName(file), ".4gl")
		mods := sourceModuleCandidates(cfg.sourceModuleOf, module, prog)
		for _, p := range sourceCandidatePaths(roots, mods, file) {
			if !inRoots(p) {
				continue
			}
			d, t, e := readOne(p)
			if e == nil {
				data, mt, path = d, t, p
				return nil
			}
			lastErr = e
		}
		return fmt.Errorf("源码未找到(%s)：按 p_link 解析出的模块 %v 在 $TOP/$CUST 的 <模块>/4gl 下都没找到：%v", file, mods, lastErr)
	})
	if err != nil {
		return nil, err
	}
	// 镜像落的是**整份**源码(在下面的行段裁剪与大小截断之前),本地副本要能整读
	mirrorPath := writeMirrorScoped(cfg.DataDir, scope, path, data)
	if len(data) > maxSourceFileBytes {
		data = data[:maxSourceFileBytes]
	}
	lines := strings.Split(string(data), "\n")
	all := len(lines)
	// 去尾部空行(文件常见尾换行)
	for all > 0 && strings.TrimSpace(lines[all-1]) == "" {
		all--
	}
	if from <= 0 {
		from = 1
	}
	if to <= 0 || to > all {
		to = all
	}
	if from > to {
		from, to = 1, all
	}
	slice := lines[from-1 : to]
	// 单行超长截断(记录行/极长字符串防爆)
	for i, ln := range slice {
		if len(ln) > 4096 {
			slice[i] = ln[:4096] + "…(截断)"
		}
	}
	res := &SourceResult{
		SourceFile: SourceFile{DVMFile: file, Path: path, ModTime: mt, LocalPath: mirrorPath},
		From:       from, To: to, All: all,
	}
	if from <= all {
		res.Content = strings.Join(slice, "\n")
	}
	return res, nil
}

// JobInfo 作业编号 → 实体程序/模块解析结果(信息通道,不启动会话)。
type JobInfo struct {
	Prog      string `json:"prog"`                // 用户作业编号
	Module    string `json:"module"`              // 解析出的模块目录(空=未解析)
	RunProg   string `json:"runProg"`             // 实体程序(gzzz_t 解析;空=未解析)
	LaunchRef string `json:"launchRef,omitempty"` //
	Extra     string `json:"extra,omitempty"`     //
	Note      string `json:"note,omitempty"`      // 未解析/降级时的说明
}

// ResolveJob 尽力解析作业:未配置 db 或连库失败/未命中时返回 note,不报错(启动时会再次解析/兜底)。
func (m *Manager) ResolveJob(module, prog string) *JobInfo {
	// GP 没有 gzzz_t(作业→实体程序)这层：作业就是程序，模块在本地字典的
	// t_prog.module / t_prog_link(由 gal_file 同步而来)。这一条不连库、不碰 SSH。
	if m.cfg.IsGPEnv() {
		return m.resolveJobGP(module, prog)
	}
	out := &JobInfo{Prog: prog, Module: module}
	if m.cfg.DB == nil {
		out.Note = "config.json 未配置 debug.db,无法按 gzzz_t 解析实体程序;将按作业名直接启动"
		return out
	}
	if !reProgName.MatchString(prog) {
		out.Note = "作业名非标准标识符,跳过库解析"
		return out
	}
	// resolveJobWith 内部含 连库→gzzz_t→42r 校验;动态路径/连接失败(err)或解析失败(空串)都提示。
	// 传副本（缺陷 Y / T5.2）:它会把探到的动态路径写进拿到的那一份 cfg。
	cfg := m.sourceCfg()
	mod, run, ref, extra, rerr := m.resolveJobWith(cfg, module, prog)
	if rerr != nil {
		out.Note = "无法获取该环境 T100 路径: " + rerr.Error()
		return out
	}
	if run == "" {
		out.Note = "未能按 gzzz_t 解析(连库失败或未命中);将按作业名直接启动"
		return out
	}
	if mod != "" {
		out.Module = mod
	}
	out.RunProg, out.LaunchRef, out.Extra = run, ref, extra
	return out
}

// resolveJobGP 解析 GP 的"作业 → 模块"：只读本地字典（t_prog / t_prog_link），
// 不连数据库、不发 SSH。GP 的作业就是程序本身，所以 RunProg 恒等于作业名。
func (m *Manager) resolveJobGP(module, prog string) *JobInfo {
	mod := strings.ToLower(strings.TrimSpace(module))
	out := &JobInfo{Prog: prog, Module: mod, RunProg: prog}
	if !reProgName.MatchString(prog) {
		out.Note = "作业名不是标准标识符(字母开头、字母数字下划线)，跳过字典解析"
		return out
	}
	if got, ok := m.cfg.sourceModuleOf(prog); ok && strings.TrimSpace(got) != "" {
		out.Module = strings.ToLower(strings.TrimSpace(got))
		out.Note = "模块来自本地字典(t_prog / t_prog_link)"
		return out
	}
	if mod != "" {
		out.Note = "本地字典里没有这条作业的模块记录，用你给的模块 " + mod
		return out
	}
	out.Note = "本地字典里没有 " + prog + " 的模块记录：先跑 tg dict db sync --env <环境>，或用 --module 指定"
	return out
}

// ---------- 会话生命周期 ----------

// CloseAll 断开全部会话(服务退出时调用)
func (m *Manager) CloseAll() {
	m.mu.Lock()
	ss := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		ss = append(ss, s)
	}
	m.sessions = map[string]*Session{}
	m.mu.Unlock()
	for _, s := range ss {
		_ = s.Close()
	}
}
