package debug

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"tt/internal/host"
	sessionid "tt/internal/session"

	"github.com/pkg/sftp"
)

// rawDebug 置 TDBG_RAW=1 时把协议原始行打到服务日志(排障用)
var rawDebug = os.Getenv("TDBG_RAW") == "1"

// State 会话状态
type State string

const (
	StateLoading State = "loading" // 正在建立(菜单/shell/启动 fglrun)或正在启动新一轮调试
	StateStopped State = "stopped" // 停在 (fgldb) 提示符,可发命令
	StateRunning State = "running" // 程序运行中,禁止发命令,仅可 \x03 中断
	StateIdle    State = "idle"    // 本轮调试已结束,宿主 shell 保留(单一常驻会话,可直接再启动)
	StateExit    State = "exit"    // 会话已断开(SSH 已释放)
)

// ErrNotStopped 运行期发命令被闸门拒绝
var ErrNotStopped = errors.New("程序运行中,禁止发送调试命令(仅允许中断)")

// Frame 调用栈帧
type Frame struct {
	Idx  int    `json:"idx"`
	Func string `json:"func"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// SourceLine 停站上下文中的源码行
type SourceLine struct {
	Num   int    `json:"num"`
	Text  string `json:"text"`
	IsCur bool   `json:"isCur"`
}

// StopInfo 停站现场
type StopInfo struct {
	Reason string       `json:"reason"` // entry|breakpoint|interrupt|step|finish|unknown
	BPNum  int          `json:"bpNum,omitempty"`
	Func   string       `json:"func,omitempty"`
	File   string       `json:"file,omitempty"`
	Line   int          `json:"line,omitempty"`
	Frames []Frame      `json:"frames,omitempty"`
	Source []SourceLine `json:"source,omitempty"`
	// WaitingForUser:当前停站行本身是交互语句(INPUT/MENU/DISPLAY ARRAY…),
	// 即程序此刻把控制权交给了前端界面。由停站源码块的当前行文本判定,零副作用。
	// 注意它描述的是「停在这条语句上」,不是「正在语句里等操作」——后者要看 Why()。
	WaitingForUser bool   `json:"waitingForUser,omitempty"`
	WaitingKind    string `json:"waitingKind,omitempty"` // input|input_by_name|input_array|display_array|menu|prompt|construct|window
}

// Breakpoint 断点
type Breakpoint struct {
	Num     int    `json:"num"`
	File    string `json:"file"`
	Line    int    `json:"line"`
	Func    string `json:"func,omitempty"`
	Enabled bool   `json:"enabled"`
	Note    string `json:"note,omitempty"` // 重复下断等场景的说明(返回既有断点时非空)
	// Module 是 `info breakpoints` 里 `in <模块> at …` 的那一段（实测形如
	// `axm_axmt500_wf.axmt500_ui_dialog`，A27）。监视点行为空。
	Module string `json:"module,omitempty"`
	// A20 三件套：条件 / 忽略次数 / 种类。
	//
	// **回显都读得回来**（A27 + A30 + A33，`info breakpoints` 的真实续行）：
	//
	//	\tstop only if <expr>              ⇒ Cond（条件断点的回声，A26 的 `break … if <expr>`）
	//	\tignore next N hits               ⇒ Ignore
	//	\tbreakpoint already hit N time(s) ⇒ Hits（命中计数，见下面的 Hits 字段）
	//
	// 所以 sync 时**以 fgldb 为准**：旧账本只在"这次没回显"（Cond 为空 / Ignore 为 0）
	// 时补空缺，见 `inheritMissingBPFields`。别写成无条件覆盖 —— 那会把刚读到的
	// `stop only if 1=1` 抹成旧值或空串（Q2 修的正是这个）。
	Cond   string `json:"cond,omitempty"`
	Ignore int    `json:"ignore,omitempty"`
	Kind   string `json:"kind,omitempty"` // "break" | "watch"(空 = "break")
	// Hits 是**命中计数**（A33，第四次真机实测）：`info breakpoints` 在命中的断点下多一行
	//
	//	\tbreakpoint already hit N time(s)
	//
	// 它才是"这条断点命中过没有"的判据 —— **`Address` 恒 `0x00000000`（命中与否都是 0），
	// 不得用于判定断点是否装上/生效**。未命中过的断点不打印那一行 ⇒ 这里是 0。
	//
	// **账本兼容**：`omitempty` ⇒ 旧存档/旧报文里没有这个键时读成 0，写出去时 0 不落键；
	// 而且它只从 fgldb 的实时回显来，**不进** `StoredBP`（`bpsstore.go` 的 JSON 契约不动），
	// 所以老账本（没有 hits 键）照旧读、照旧恢复，零迁移成本。
	Hits int `json:"hits,omitempty"`
}

// VarItem 变量求值项(局部变量/自动变量)
type VarItem struct {
	Expr  string `json:"expr"`
	Value string `json:"value,omitempty"`
}

// VarDecl 全局变量声明项(info variables 输出名字+类型)
type VarDecl struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Event 会话事件(WS/MCP 广播)
type Event struct {
	Type      string    `json:"type"` // state|stopped|output|watchdog|log|dead|autovars|ai_action
	SessionID string    `json:"sessionId"`
	Time      time.Time `json:"time"`
	State     string    `json:"state,omitempty"`
	Stop      *StopInfo `json:"stop,omitempty"`
	Vars      []VarItem `json:"vars,omitempty"`
	Text      string    `json:"text,omitempty"`
	// Seq 单调递增序号,由 Manager.emit 分配(进程级,重启归零;配 Manager.Epoch 识别)。
	// 用途是**游标语义**:WS 开场补发后靠它去重、前端靠它判重。
	// 注意 hWait 不依赖它 —— 那里是栅栏语义,谓词只看权威现状。
	Seq uint64 `json:"seq,omitempty"`
	// Actor 谁发起的:ai | human | system(空 = 未声明,按 system 处理)
	Actor string `json:"actor,omitempty"`
	// Action 机器可读的动作名(bp.add / control.step / session.restart),供前端按类型分支;
	// Text 则是给人看的文案
	Action string `json:"action,omitempty"`

	// Brief 是 `session.added` 带的那条会话的投影（W2）。用**同一份** `SessionBrief`：
	// 客户端收到就能直接入索引，不必再 GET 一次（少一次往返、也少一个"列表还没刷到"的窗口）。
	Brief *SessionBrief `json:"brief,omitempty"`
	// Reason 是 `session.removed` 的原因：close | exit | restart | switch | stop | stale | removed
	Reason string `json:"reason,omitempty"`
}

// execResult 命令响应
type execResult struct {
	Cmd        string
	Lines      []string
	Stop       *StopInfo
	Frames     []Frame
	Value      string
	BP         *Breakpoint
	BPs        []Breakpoint
	Continuing bool
	SawShell   bool
	Err        string // 调试器错误行(No stack / No symbol 等)
	// SoftTimeout:软等待到点返回(程序仍在跑,命令没有被取消、也没发 SIGINT)。
	// 区别于 exec 返回的硬超时 error —— 那条路径会发 \x03 探测。
	SoftTimeout bool `json:"softTimeout,omitempty"`

	// Truncated/TruncReason:这条命令的响应**在会话层就被截断过**
	// (行数超 maxPendingLines,或单个 print 值超 maxValueBytes)。
	// 以前这对标记只用来"只标一次"、从不对外暴露 —— 于是调用方拿到的是被砍过的内容
	// 却毫不知情,会当成全部。现在如实带出去。
	Truncated   bool   `json:"truncated,omitempty"`
	TruncReason string `json:"truncReason,omitempty"` // lines | value
}

type pendingCmd struct {
	cmd     string
	kind    string // where|print|break|breakpoints|step|continue|run|other
	mode    int    // waitPrompt|waitMarker|waitQuiet
	lines   []string
	value   strings.Builder
	valueOn bool
	frames  []Frame
	// bpRaw 是 `info breakpoints` 的**原始行**（不在这里逐行判读）。
	// 为什么留着原始行：A27 实测一条断点会跨两行（第二行是缩进的 `ignore next N hits`），
	// 逐行匹配会把那条 ignore 丢掉；统一交给纯函数 ParseInfoBreakpoints 处理，
	// 解析失败时还能把原文原样带出来（不许静默吞掉）。
	bpRaw   []string
	errText string
	res     chan *execResult

	startedAt time.Time // 命令发出时刻(快照用来回答"AI 正在执行什么、跑了多久")

	valueTrunc bool // print 值已截断
	linesTrunc bool // 行列表已截断

	// softAbandoned 软等待已放弃等待这条命令,但**命令仍在飞**。
	// 槽位不能在这里清:onStop 收口时要靠它区分「还有人等结果(sync 路径)」与
	// 「已无人等待(async 路径)」——sync 路径只回填响应、不置 stopped、不发 stopped 事件,
	// 若被放弃的命令误走 sync,会话会永久卡在 running 且停站事件永久丢失。
	softAbandoned bool
}

const (
	maxPendingLines = 20000     // 单命令响应行数上限(容纳 info variables/functions)
	maxValueBytes   = 256 << 10 // 单个 print 值字节上限
)

// writeValue 追加 print 值(带截断保护)
func (p *pendingCmd) writeValue(str string) {
	if p.value.Len() >= maxValueBytes {
		if !p.valueTrunc {
			p.valueTrunc = true
			p.value.WriteString("\n[输出已截断]")
		}
		return
	}
	p.value.WriteString(str)
}

// addLine 追加响应行(带截断保护)
func (p *pendingCmd) addLine(ln string) {
	if len(p.lines) >= maxPendingLines {
		if !p.linesTrunc {
			p.linesTrunc = true
			p.lines = append(p.lines, "[输出已截断]")
		}
		return
	}
	p.lines = append(p.lines, ln)
}

// infoBPs 用 A27 的解析器把 `info breakpoints` 的原始行翻成断点账本。
//
// 解析失败时把错误写进 errText（调用方会当命令错误返回），**不静默吞掉** ——
// 一张空断点表 + 没有任何线索，是最难查的那种现场。
func (p *pendingCmd) infoBPs() ([]Breakpoint, string) {
	if len(p.bpRaw) == 0 {
		return nil, ""
	}
	bps, err := ParseInfoBreakpoints(p.bpRaw)
	if err != nil {
		return nil, err.Error()
	}
	return bps, ""
}

// settleBPs 算断点账本并把解析错误并入 errText（必须在构造 execResult **之前**调用，
// 结构体字面量里 BPs 字段排在 Err 前面，正是为了让这条错误赶得上）。
func (p *pendingCmd) settleBPs() []Breakpoint {
	bps, e := p.infoBPs()
	if e != "" && p.errText == "" {
		p.errText = e
	}
	return bps
}

const (
	waitPrompt = iota // 等裸 (fgldb) 提示符
	waitMarker        // 等 "Continuing."(continue)
	waitQuiet         // 等输出安静(run)
)

type stopCollect struct {
	reason string
	bpnum  int
	fn     string
	file   string
	line   int
	frames []Frame
	source []SourceLine
}

// Session 一条调试会话 = 一条 SSH PTY 里跑的 fglrun -d
type Session struct {
	ID      string
	Module  string
	Prog    string
	RunProg string // gzzz_t 解析出的实体程序编号(gendbg 语义);空 = 与 Prog 相同
	// gendbg 原版启动引用:gzza004 去 "$FGLRUN" 前缀(如 "$CINi/ainq120_wf"),
	// 启动命令原样交给选区后的 shell,$变量 展开为权威 42r 路径(标准/客制由环境决定);空 = 回退本地探测
	LaunchRef    string
	ExtraArgs    string // gzzz004 额外参数(gendbg 拼在程序名之后)
	ArgsOverride string // 启动参数覆盖(接口日志重放调试:报文文件对);空 = 用 LaunchArgs 模板

	cfg  *Config
	conn *host.SSHConn
	pty  *host.PTYSession

	mu       sync.Mutex
	state    State
	started  bool // 程序是否已 run 过(入口停站时步进类命令非法)
	pending  *pendingCmd
	collect  *stopCollect
	cur      StopInfo
	curFrame int // 当前选中的栈帧(frame N),新停站自动回到栈顶
	bps      map[int]*Breakpoint
	quitReq  bool
	startAt  time.Time // 进入 stopped 时刻(看门狗/停留计时)

	// quitting/quitDone：收口（结束调试/关闭会话）的"执行者与等待者"协议。
	//
	// 为什么要有 quitDone：原来 `quitting` 为真时后来者**立刻 return nil** —— 报"成功"，
	// 可那一次收口还在跑，会话仍是 running。调用方于是紧接着做下一步（设 TOPENT / 切环境），
	// 撞上"仅会话空闲(idle)时可设置 TOPENT(当前 运行中)"。实测 2026-10-16：前一条命令
	// 与另一条命令的 quit 撞在一起，后一条就这么静默失败了。现在后来者**等这一次收口做完**。
	quitting bool
	quitDone chan struct{}

	runningSince time.Time // 进入 running 时刻(回答"跑了多久")
	lastOutputAt time.Time // 最近一次有协议输出的时刻(回答"静默了多久")

	lastAutovars []VarItem // 最近一次停站的自动变量求值结果
	autovarsOn   bool      // 停站后自动求值自动变量(前端面板开关,默认关:减少自动调度拖慢命令队列)

	emit  func(Event)
	lines chan string // 全量行广播(启动序列等待用)
	done  chan struct{}

	// execFn 是"发一条调试命令并拿回响应"的**可替换通道**（nil = 真实 SSH PTY）。
	//
	// 生产里它恒为 nil（`execOpt` 只在非 nil 时才走它）。存在的理由只有一条：A36 的两条
	// 路径 —— `bp list` 的停站回灌、`exec "info breakpoints"` 的读后回灌 —— 必须能在
	// **不连机器、不起会话**的前提下钉死（纪律：全部用假内核/HTTP 桩）。
	// 与 `emit` 同类：测试注入的接缝，不参与状态机（状态翻转仍由调用方与真实泵负责）。
	execFn func(kind, cmd string, mode int, timeout time.Duration) (*execResult, error)

	sftpClientCache *sftp.Client
	srcCache        map[string]srcCacheEntry
	// mirrored 已经发起过镜像落盘的 DVM 源名(见 mirrorStopFile):
	// 同一个文件反复停站不重复拉取。尽力而为的标记 —— 失败的不会重试。
	//
	// 它必须与**镜像落盘的目录**同一个作用域（缺陷 X / T5.1）:标记是会话内的,
	// 所以落盘目录也得按会话分开,否则 A 拉过的文件在共享目录里,B 永远看不到。
	// 作用域由 runScope 决定（见 mirrorScope）。
	mirrored map[string]bool
	// runScope 本条会话在本地镜像/命令输出目录里的**运行作用域**:
	// 按会话唯一（NewSession 分配）、跨轮不变（复用的宿主会话继承上一轮的,见
	// inheritMirrorState）。它让"清空本会话的镜像"与"别的会话的镜像"互不相干 ——
	// 从前清的是全局 `srccache/`,并发启动互删（缺陷 X）。
	runScope string

	stopTimer     *time.Timer
	approachTimer *time.Timer
	retryTimer    *time.Timer
	kaStop        func() // 停止 SSH 心跳(主动关闭连接前调用,避免误报掉线)
	restoring     bool   // 入口停站后的断点恢复进行中(对外仍报 loading)

	custModule string // 转客制作业:实际启动的客制模块名(cpm/apm→cpm),空 = 标准模块

	// ---- 单一常驻会话(复用宿主) ----
	envName        string // 会话所连环境名(设置页 envs 名称;空 = 顶层默认/推导名)
	booted         bool   // 是否已完成首次登录(菜单→shell),宿主可复用
	shellReady     bool   // pty 当前停在 shell 提示符,可直接敲 shell 命令
	topentOverride string // 会话内手动设置的 TOPENT(空 = 未设置,按配置默认)

	// ---- 重放调试 ----
	// replayRspPath 本次重放把响应写到哪个服务器临时文件(LaunchReplay 填)。
	// 程序退出前它还不存在,所以只能等跑完再读(见 loadReplayResponse)。
	replayRspPath  string
	replayResponse string // 读回来的响应原文(只读一次)

	// mode 谁在驾驶:ModeSolo(纯人工,默认) | ModeCollab(协作,AI 主导)。
	// 由发起方决定,之后人与 AI 都能切(界面上的接管/交给 AI 按钮,或 AI 的
	// `tt debug mode`),两个方向都放行 —— 不做逐命令协商。
	mode string

	// ---- 会话身份（契约 C5.3 / Q22 的纯新增四键里的三个：n / name / epoch）----
	//
	// 三样都由 Manager 在**登记那一刻**发一次（见 assignIdentityLocked），之后不再重算：
	//
	//	seq   —— 序号 `n`：**同一环境下**的第 N 条 SSH 会话，从 1 起（C2.2 / Q24 / Q16）。
	//	         企业编号不参与 —— 按（环境, 企业编号）自增会让 `环境#序号` 撞车（2.1.3）。
	//	         它进 `ref`（`环境#序号`），所以**生命周期内一个字符都不变**（C2.0）；
	//	         `tt debug topent 88` 改掉 ent 之后它必须照旧（改的是快照，不是地址）。
	//	name  —— 别名（Q17 / C2.5b）：只在创建时给一次，生命周期内不变；**随会话消失即释放**
	//	         （C3.7b / Q19）—— 它就挂在这条会话对象上，会话从池子里没了，名字自然能给别人用。
	//	epoch —— 世代号（C5.3 / F15）：换代的时机与"为什么全进程唯一"见 nextSessionEpoch。
	//
	// 三个都不是"这条会话现在什么样"，而是"这条会话是哪条"（C5.1 末段：只有 id/ref/n 是身份，
	// 其余是快照）—— 读它们一律走下面的取值器，别从别处推导。
	seq   int
	name  string
	epoch string
}

const (
	// ModeSolo 纯人工:只有人能写,AI 只能读。默认模式,界面与旧行为完全一致。
	ModeSolo = "solo"
	// ModeCollab 协作:AI 主导,人只能读 + 请求 AI 代为操作。
	ModeCollab = "collab"
)

// Mode 当前模式(空值视为 solo)
func (s *Session) Mode() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == "" {
		return ModeSolo
	}
	return s.mode
}

// SetMode 设置模式
func (s *Session) SetMode(m string) {
	s.mu.Lock()
	s.mode = m
	s.mu.Unlock()
}

// 能力类（W7）：一次写操作属于哪一类，决定它受哪些轴约束。
//
// 为什么必须分类：有些写操作要"只读"，但编排动作（起作业）还得放行
// 才跑得起来。一刀切（record 什么都不许写）会把它自己挡在门外 —— 真机第一次挂画面
// 就是这么失败的（报"纯人工模式:AI 不可操作"，而真正原因是用途和动作类没分开）。
type Capability int

const (
	// CapInteractive 交互式调试：控制 / 断点 / 变量 / raw / 定位 / 校准。
	CapInteractive Capability = iota
	// CapSetup 编排：在会话里起作业、设 ENT、起/停界面会话。
	CapSetup
	// CapOwn 收尾：close / quit / restart / 停界面会话。**任何用途都允许**。
	CapOwn
)

// MayDo 回答"这次请求能不能对这条 SSH 会话做这一类写"。
//
// 两个轴（W7）：模式 `mode` × 角色 `actor`。
//   - 收尾（CapOwn）任何模式都放行：收不掉比收错了更糟（H5 幂等收尾）。
//   - 其余沿用既有模式闸门（solo：非 ai 可写；collab：只有 ai 可写）。
//
// `Bare()`（测试夹具：没有 pty/conn 的会话）一律放行 —— 与 sessOfWriteGlobal 同一口径。
func (s *Session) MayDo(cap Capability, actor string) bool {
	// 收尾先放行：**任何模式、任何角色**都收得掉（H5：收不掉比收错了更糟）。
	if cap == CapOwn {
		return true
	}
	// 平台自己的进程替这条会话干活：**不受模式约束**。
	if actor == ActorSystem {
		return true
	}
	return s.MayWrite(actor)
}

// ---------- 会话身份（契约 C5.3 / Q22：n / name / epoch；ref 由环境 + n 拼） ----------

// Seq 是序号 `n`：**同一环境下**的第 N 条 SSH 会话，从 1 起（C2.2 / Q24）。
//
// 建立时由 Manager 分配一次（尚未被占用的最小正整数），之后**不再重算** —— 这正是
// `ref` 能在生命周期内恒定不变（C2.0）的关键：`SetTopent` 改掉企业编号不会让它变。
func (s *Session) Seq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seq
}

// Name 是别名（`--name` 给的那个，Q17）：空串 = 未命名。
// 生命周期内不变（C2.5b：要改名就结束这条会话再起一条）；随会话消失即释放（C3.7b / Q19）。
func (s *Session) Name() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.name
}

// Epoch 是世代号（C5.3 / F15）。
//
// 语义（本批定稿，写在这里就不必再去翻文档）：**"这条会话这一代宿主 / 这一轮运行"的令牌**。
// 它由 nextSessionEpoch 发一个**全进程唯一**的 base36 值，换代的三个时机是：
//  1. 会话建立（Manager 登记时）→ 第一代；
//  2. **同一宿主跑新一轮**（SetRun：`tt debug start` 复用空闲宿主 / 按日志重放复用宿主）→ 换代；
//  3. **会话重启/重新登录**（`/api/sessions/{id}/restart`：旧会话收掉、新会话登上来）→ 新世代。
//
// 调用方（F15）把上一次拿到的这个值原样带回来；与服务端当前值不等就是"手上那份是上一代"，
// 报 `SESSION_STALE` 退 3（判据在 `internal/target` 的 `Resolve`：`Request.Epoch`）。
func (s *Session) Epoch() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.epoch
}

// CurrentEnt 是该会话**当前**生效的企业编号（C5.3 的 `ent`，C2.5c 的快照）。
//
// 取值顺序：登录 shell 回读的 `$TOPENT` 优先（那是"这台机器现在连的是哪个企业"的权威值，
// 见 TopentShell），取不到（还没登录 / 没有 Runtime）才退到本会话显式给的值（--ent 或
// 会话内 `tt debug topent` 设的覆盖）。空 = 未取到 —— 它是**快照**，不是地址（C2.5c）。
func (s *Session) CurrentEnt() string {
	if v := s.TopentShell(); v != "" {
		return v
	}
	return s.topentForRun()
}

// setTopentOverride 记下会话级 TOPENT 覆盖（`tt debug topent` 改的那一半）。
//
// 拆出来是为了让"改企业编号**不动**身份"这件事可测：SetTopent 的另一半是把值下发到宿主
// shell（要 pty，测试里碰不得），而身份不变恰恰与那半边无关 —— 变的只有快照 `ent`。
func (s *Session) setTopentOverride(v string) {
	s.mu.Lock()
	s.topentOverride = v
	s.mu.Unlock()
}

// NewSession 建立 SSH 连接并打开 PTY(登录与启动由 Launch 驱动)。
// cfg 做一份快照:会话跨多轮调试存在,设置热更新不应改变其连接目标与启动参数
// (启动一轮时 Manager 会按最新生效配置重刷,见 SetRun)。
func NewSession(cfg *Config, module, prog, runProg, launchRef, extraArgs string, emit func(Event)) (*Session, error) {
	conn, err := host.Dial(cfg.SSH)
	if err != nil {
		return nil, fmt.Errorf("SSH 连接失败: %w", err)
	}
	cc := *cfg
	s := &Session{
		ID:        nextSessionID(),
		Module:    module,
		Prog:      prog,
		RunProg:   runProg,
		LaunchRef: launchRef,
		ExtraArgs: extraArgs,
		cfg:       &cc,
		conn:      conn,
		state:     StateLoading,
		// 业主显式授权（config.aiTakeover=true）时，会话直接以协作模式起：
		// 这时 AI 可写（启动作业/下断点/步进/求值）。默认仍是纯人工（solo）。
		mode: func() string {
			if cfg != nil && cfg.AITakeover {
				return ModeCollab
			}
			return ModeSolo
		}(),
		envName:  cfg.EnvName(),
		bps:      map[int]*Breakpoint{},
		srcCache: map[string]srcCacheEntry{},
		// 世代号的**第一代**（C5.3 / F15）：会话建立那一刻发一个。序号 `seq` 与别名 `name`
		// 不在这里 —— 它们由 Manager 在**登记**时分配（要跨会话查重，见 assignIdentityLocked）。
		epoch: nextSessionEpoch(),
		// 运行作用域按会话唯一（缺陷 X / T5.1:镜像与命令输出按会话分开,
		// 并发启动互删不了）。它是**内部标识**,不出现在命令输出里（会话可见的
		// ID 仍是上面那个 `s<毫秒>` 的既有形状）。
		runScope: nextRunScope(),
		emit:     emit,
		lines:    make(chan string, 1024),
		done:     make(chan struct{}),
	}
	s.kaStop = conn.StartKeepalive(func(err error) {
		s.emitEvent(Event{Type: "dead", Text: "SSH 连接断开: " + err.Error()})
		s.forceExit()
	})
	pty, err := conn.NewPTY(cfg.TermWidth, cfg.TermHeight)
	if err != nil {
		s.kaStop()
		conn.Close()
		return nil, fmt.Errorf("打开 PTY 失败: %w", err)
	}
	s.pty = pty
	go s.pump()
	return s, nil
}

// ---------- 生命周期 ----------

// runProgName 取本轮的实体程序名(优先 RunProg,回退 Prog)
func (s *Session) runProgName() string {
	if s.RunProg != "" {
		return s.RunProg
	}
	return s.Prog
}

// EnvName 返回会话所属环境名
func (s *Session) EnvName() string {
	return s.EnvNameNow()
}

// EnvNameNow 读**当前**环境名(跨多个调试轮次,可能已被 SetRun 刷新)。
// 与会话创建时那一次快照不同 —— 它是判"这个会话现在连的是哪个环境"的权威值。
func (s *Session) EnvNameNow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.envName
}

// ZoneNow 读**当前**登录区域(与 EnvNameNow 同理:会话跨轮次,配置可被 SetRun 刷新)。
func (s *Session) ZoneNow() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg == nil {
		return ""
	}
	return s.cfg.Zone
}

// FGLServer 读**本轮**生效的 FGLSERVER 覆盖值(空 = 未覆盖,由 T100 按 SSH 来源 IP 自动设置)。
//
// 与 EnvNameNow 同理:会话跨轮次,SetRun 会把本轮的配置刷进来,所以它回答的是
// "这一轮启动时导出的是什么",而不是会话创建那一刻的值。
func (s *Session) FGLServer() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg == nil {
		return ""
	}
	return s.cfg.FGLServer
}

// ScreenTarget 用一句人话说明"这一轮作业界面会画到哪儿"。
//
// TG 只跑 TOP GP，画面由 fgldb 接管：没有 T100 的 FGLSERVER / 本机 GDC 那条线，
// 也就没有"接线没接好、-6300 退出"这类告警可言。
func ScreenTarget(fglserver string) string {
	return "画面去向: 由 fgldb 接管（GP 没有 FGLSERVER 这条线）"
}

// LocalGDCPresent 本机有没有 GDC 在听(TCP 连得上就算)。只连不发数据，两个端口各 200ms。
//
// 端口按 ScreenTarget 那条约定来：7400 是 GDC 的默认前端端口，6400 是它的控制端口。
// 结果带 10 秒缓存：这条探测最多 400ms，而它的调用点之一是会话的输出解析循环
// （退出那一枪），不能让 dial 卡住输出处理。
func LocalGDCPresent() bool {
	gdcProbeMu.Lock()
	if time.Since(gdcProbeAt) < 10*time.Second {
		v := gdcProbeVal
		gdcProbeMu.Unlock()
		return v
	}
	gdcProbeMu.Unlock()

	found := gdcProbe()

	gdcProbeMu.Lock()
	gdcProbeAt, gdcProbeVal = time.Now(), found
	gdcProbeMu.Unlock()
	return found
}

// gdcProbe 是真实探针，测试里可替换（见 session_test.go 的 TestScreenWarn*）。
var gdcProbe = dialGDC

func dialGDC() bool {
	for _, p := range []string{"7400", "6400"} {
		c, err := net.DialTimeout("tcp", "127.0.0.1:"+p, 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}
	}
	return false
}

var (
	gdcProbeMu  sync.Mutex
	gdcProbeAt  time.Time
	gdcProbeVal bool
)

// ScreenWarn 在"本轮画面交回 T100 自动指"而**本机又没有 GDC 在听**时，给一句能照做的说明；
// 其它情况返回空串（本轮有 FGLSERVER 覆盖时去向由调用方负责，dbgScreenVerify 会认账）。
//
// 为什么必须有这句（实测 2026-10-16）：用户报"debug 走两步自动退出了"。真因是本机没起
// GDC —— 画面交回 T100 自动指 = T100 把画面发回 **SSH 来源 IP 的本机 GDC**，本机 7400/6400
// 上没有监听者时，作业会在**第一次画界面**时直接退出（用户那条会话正好：`tbreak main`
// 停在 MAIN 首条可执行语句 OPTIONS，一次单步走到初始化、要开窗口，当场退），而界面上只有
// "程序已退出(作业窗口被关闭或正常结束)" —— 与"用户自己关了窗口"完全分不清。
//
// A35（第四次真机）把话说得更死：**没有接线的会话在 `main` 之后**立刻** `Fatal error -6300`
// 退出，守卫行（`main` 之后那些行）来不及停**；接线就绪的会话则不会。所以这句既是"画面看不到"
// 的解释，也是"断点怎么设都命不中"的解释 —— 必须给到能照做的动作（起 GDC / 用 --fglserver）。
// ScreenWarn / ScreenWarnWith 在 TG 里恒为空串：GP 的画面由 fgldb 接管，
// 没有 T100 的 FGLSERVER / 本机 GDC 那条线，也就没有"没接线会 -6300"这回事。
func ScreenWarn(fglserver string) string { return "" }

func ScreenWarnWith(fglserver string, gdcPresent bool) string { return "" }

// screenExitTail 给"程序已退出"那句话补线索：GP 不需要。
func (s *Session) screenExitTail() string { return "" }

// sameConn 判断另一配置是否指向**同一台机器同一区域**(host/port/user/zone 全同)。
// 它只回答"连接能不能复用",不回答"是不是同一个环境" —— 同机多环境(同库不同
// 企业)在这四样上完全一样,判环境必须用 sameEnvAs。
func (s *Session) sameConn(cfg *Config) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg == nil || s.cfg == nil {
		return false
	}
	a, b := s.cfg.SSH, cfg.SSH
	return a.Host == b.Host && a.Port == b.Port && a.User == b.User && s.cfg.Zone == cfg.Zone
}

// sameTarget 判断"这次启动的目标环境与该会话当前所在环境是同一个"。
//
// 判据是**环境名**(不是连接四元组):同一台机器上可以有多个环境(恒烁正式区/测试1012,
// 同库不同企业),它们的 host/port/user/zone 完全一样,只有环境名与企业编号不同。
// 只比四元组会把它们判成同一个,于是点名切环境后仍旧复用旧会话、继续用旧的企业编号跑
// (实测踩过:所有按企业编号取数的查询返回 0 行,看起来像"配置缺失",结论会完全反向)。
//
// 连接四元组仍是前置条件:同名但连接不同(设置页把某环境的 host 改了)也必须是"换目标",
// 不能拿旧连接继续跑。
//
// 两条判据任意一条为空时的取向:连接四元组为空 → 不敢复用,**换**;
// 环境名算不出(Config.EnvName 的 host-zone 兜底理论上兜得出非空值,这里是防御)→ 允许复用。
// 宁可多登录一次,也不能上错号。
func (s *Session) sameTarget(cfg *Config) bool {
	if cfg == nil {
		return false
	}
	if !s.sameConn(cfg) {
		return false
	}
	a, b := s.EnvNameNow(), cfg.EnvName()
	if a == "" || b == "" {
		return true
	}
	return a == b
}

// sameEnvAs 就是 sameTarget 的别名:调用点用它读起来更像"落在同一个环境上吗"。
func (s *Session) sameEnvAs(cfg *Config) bool { return s.sameTarget(cfg) }

// Launch 启动一轮调试(单一常驻会话):
// - 首次调用:先完成登录(区域菜单 → shell → 回读运行时环境),之后宿主常驻;
// - 后续调用:复用已登录宿主,直接从「进目录 → fglrun」开始,免去重复登录;
// 一轮结束只回到 idle(enterIdle),SSH/登录态保留;失败则尽力把宿主收回空闲。
func (s *Session) Launch(ctx context.Context) error {
	s.beginRun()
	err := s.launchInner(ctx)
	if err != nil && s.booted {
		// 复用宿主上启动失败(常见:作业名错误 fglrun 未起):回收宿主到空闲,
		// 不销毁连接——避免失败一次就重新登录;真卡死由用户用「重启会话」重建
		s.enterIdle("启动失败,会话保留(可换作业重试或重启会话): " + err.Error())
	}
	return err
}

// launchInner 启动流程主体(见 Launch 说明)
func (s *Session) launchInner(ctx context.Context) error {
	launchProg := s.runProgName()
	log.Printf("[debug] launchInner 会话 %s product=%q prog=%q runProg=%q module=%q",
		s.ID, s.cfg.Product, s.Prog, launchProg, s.Module)

	// TOP GP：模块名就是作业编号的前三位，不需要按 T100 的 42r 目录去探测。
	// 走通用路径会在"自动解析模块失败"那里直接返回，根本到不了 startRun。
	if strings.EqualFold(s.cfg.Product, "gp") {
		if !s.booted {
			if err := s.bootHost(ctx); err != nil {
				return err
			}
		}
		// **用作业编号本身，不用 runProgName()**：后者的模块是查 T100 的 gzzz 目录得出来的，
		// GP 上那套目录对不上（csf_csfi300_icd 被解析成 asf），模块一错就 cd 不到目录。
		launchProg = s.Prog
		// 和 T100 的显式模块规则一致：表单给了 csf 就用 csf，不再从 prog 猜。
		// 从 csfi300_icd 取第一个下划线前会得到 csfi300，真实模块是 csf（实测踩过）。
		if strings.TrimSpace(s.Module) == "" {
			s.Module = gpModuleOf(launchProg)
		}
		return s.startRun(ctx, launchProg)
	}

	var resolveErr error

	// 模块解析:已有动态路径(复用宿主登录环境/探测缓存)先试;登录后再用权威环境补一次
	if s.Module == "" && s.cfg.Runtime != nil && s.cfg.Runtime.Valid() {
		if mod, err := s.resolveModule(launchProg); err == nil {
			s.setResolvedModule(launchProg, mod)
		} else {
			resolveErr = err
		}
	}

	if !s.booted {
		// 预登录解析(探针缓存动态路径,可能尚不可用);失败不中断——登录后真实环境再试
		if s.Module == "" {
			if mod, err := s.resolveModule(launchProg); err == nil {
				s.setResolvedModule(launchProg, mod)
			} else {
				resolveErr = err
			}
		}
		if err := s.bootHost(ctx); err != nil {
			return err
		}
		// 登录后补一次模块解析:选区后的 TOP/ERP/COM 最权威,预登录探测不可靠时
		// (如区域与服务器菜单映射不一致)在这里用真实环境纠正
		if s.Module == "" && s.cfg.Runtime != nil && s.cfg.Runtime.Valid() {
			if mod, err := s.resolveModule(launchProg); err == nil {
				s.setResolvedModule(launchProg, mod)
			} else {
				resolveErr = err
			}
		}
	} else if !s.shellReady {
		if err := s.ensureShellForRun(); err != nil {
			return err
		}
	}

	if s.Module == "" {
		if resolveErr == nil {
			resolveErr = fmt.Errorf("在各模块 42r 目录中未找到作业 %s(请检查作业名)", launchProg)
		}
		return fmt.Errorf("自动解析模块失败: %w", resolveErr)
	}
	return s.startRun(ctx, launchProg)
}

// setResolvedModule 记录模块解析结果并广播日志
func (s *Session) setResolvedModule(prog, mod string) {
	s.Module = mod
	s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("按作业名解析模块:%s → %s", prog, mod)})
}

// beginRun 开启新一轮调试前的复位:清上一轮现场,置 loading(宿主可能复用)。
// 注意:不动 booted/shellReady(宿主登录态跨轮保留)。
func (s *Session) beginRun() {
	s.mu.Lock()
	prev := s.state
	s.state = StateLoading
	s.started = false
	s.pending = nil
	s.collect = nil
	s.quitReq = false
	s.cur = StopInfo{}
	s.curFrame = 0
	s.bps = map[int]*Breakpoint{}
	s.lastAutovars = nil
	s.restoring = false
	s.custModule = ""
	s.mu.Unlock()
	s.stopWatchdog()
	if prev != StateLoading {
		s.emitEvent(Event{Type: "state", State: string(StateLoading)})
	}
}

// bootHost 首次登录:区域菜单 → 选区 → shell → 回读运行时环境 → 下发环境默认 TOPENT。
// 仅调用一次。
func (s *Session) bootHost(ctx context.Context) error {
	// TOP GP 的登录器问的是实例菜单，不是区域菜单 —— 两套流程各走各的
	log.Printf("[debug] 会话 %s 登录: product=%q env=%q zone=%q", s.ID, s.cfg.Product, s.envName, s.cfg.Zone)
	if strings.EqualFold(s.cfg.Product, "gp") {
		if err := s.bootGP(ctx); err != nil {
			log.Printf("[debug] 会话 %s TOP GP 登录失败: %v", s.ID, err)
			return err
		}
		return nil
	}
	// 1. 等区域菜单
	// 菜单格式两种:109 那台 `(*)Exit`,金仓这台 `*)Exit`(无左括号)
	if err := s.waitRegexp(host.ReLoginMenu, 25*time.Second, "区域菜单"); err != nil {
		return err
	}
	s.pty.Write(s.cfg.Zone + "\r")
	// 2. 等 shell 提示符
	if err := s.waitRegexp(host.ReShellPrompt, 25*time.Second, "shell 提示符"); err != nil {
		return err
	}
	s.markShellReady()
	// 2.5 连接即下发**用户显式给的企业编号**(tt debug start --ent / 界面面板输入)。
	//
	// 环境配置里已经**没有**企业编号了(hosts.sshs[].topent 已移除)——所以这里不再有
	// "环境默认企业"可下发。空值就什么都不做:**保留登录 shell 自己的 $TOPENT**
	// (选区/环境脚本给的那个),它会被下面 2.6 的回读如实抓成 TopentShell。
	// 换句话说:不给 --ent = 沿用这台机器登录后的企业;给了 = 以给的那个为准。
	if ent := s.topentForRun(); ent != "" {
		if err := s.applyTopentToShell(ent); err != nil {
			return fmt.Errorf("登录 %s(环境 %s)后下发 TOPENT=%q 失败: %w",
				s.cfg.SSH.Host, s.envName, ent, err)
		}
	}
	// 2.6 回读登录后的权威 T100 路径(选区后环境变量,与标准 debug 同源)。
	// T100 路径只来自登录动态获取:回读失败即登录失败,无静态配置可回退
	if err := s.readRuntimeEnv(); err != nil {
		return fmt.Errorf("登录 %s(环境 %s,zone %s)后回读 T100 路径失败: %w",
			s.cfg.SSH.Host, s.envName, s.cfg.Zone, err)
	}
	s.mu.Lock()
	s.booted = true
	s.mu.Unlock()
	s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("已登录 %s(环境 %s),会话将常驻复用", s.cfg.SSH.Host, s.envName)})
	return nil
}

// ensureShellForRun 复用宿主开跑前,确保 pty 当前停在可用的 shell 提示符:
// 若上一轮结束在 (fgldb) 残留(fglrun 未随程序退出),先补发 quit 回到 shell。
func (s *Session) ensureShellForRun() error {
	s.mu.Lock()
	if s.shellReady {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	s.emitEvent(Event{Type: "log", Text: "上一轮调试器未完全退出,先回到 shell"})
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ln := <-s.lines:
			if host.ReShellPrompt.MatchString(ln) {
				s.markShellReady()
				return nil
			}
			if host.IsBarePrompt(ln) {
				if err := s.pty.Write("quit\r"); err != nil {
					return err
				}
				if err := s.waitRegexp(host.ReShellPrompt, 20*time.Second, "shell 提示符(退出残留调试器)"); err != nil {
					return err
				}
				s.markShellReady()
				return nil
			}
		case <-deadline:
			// 没有提示符可循:保守按已停在 shell 处理(直接敲命令,若真卡住由超时回落)
			s.markShellReady()
			return nil
		}
	}
}

// startRun 运行启动段:进模块目录 → fglrun -d → 入口停站就绪。
// 前提:已登录(首次 bootHost 或复用),shellReady 为真,Module 已解析。
func (s *Session) startRun(ctx context.Context, launchProg string) error {
	// TOP GP：进 4gl 目录 → r.d2 <去掉模块前缀的程序名>（真机验证过的唯一启动方式）
	if strings.EqualFold(s.cfg.Product, "gp") {
		return s.startRunGP(ctx, launchProg)
	}
	// 3. 进模块目录 + 源码路径
	// 转客制的作业 42r/源码部署在 c<module> 目录(原版 T100 从 c** 启动,FGLLDPATH 也 c 优先),
	// 这里探测:客制目录有同名 42r 就用客制,否则用标准模块目录
	modDir := s.pickLaunchDir(launchProg)
	log.Printf("[debug] 会话 %s 启动目录: %s (prog=%s module=%s)", s.ID, modDir, launchProg, s.Module)
	setup := fmt.Sprintf("cd %s\r\nexport FGLSOURCEPATH=%s\r\n",
		modDir, s.fglsourcePathOf(modDir))
	if s.cfg.FGLServer != "" {
		setup += "export FGLSERVER=" + s.cfg.FGLServer + "\r\n"
	}
	// 画面去哪必须**显式说出来**:接线错了的表现是"放行之后什么都不弹",与程序卡死
	// 在界面上无法区分,而启动段本身不会报任何错(见 ScreenTarget 的注释)。
	s.emitEvent(Event{Type: "log", Text: ScreenTarget(s.cfg.FGLServer)})
	// 再把"这条去向能不能通"说出来。实测 2026-10-16:用户报"debug 走两步自动退出了",
	// 真因是本机没起 GDC —— 画面交回 T100 自动指(默认本机 GDC)、而本机 7400/6400 上
	// 没有监听者,作业第一次画界面就死了,界面上只有"程序已退出(作业窗口被关闭或正常结束)",
	// 与"用户自己关了窗口"分不清。这一句把真因提前摆出来。
	//
	// 类型用 `log` + "⚠ " 前缀,而不是自定义一个 `warn`:前端的时间线只认 log/stop/command
	// 这几种事件类型(见 web/app/src/store.ts 的 WS 分发),自造类型会被**静默丢掉**
	// ——实测过一次,事件发了、界面上什么都没有。既有约定见 manager.go 的重放警告。
	if w := ScreenWarn(s.cfg.FGLServer); w != "" {
		s.emitEvent(Event{Type: "log", Text: "⚠ " + w})
	}
	// TOPENT:会话内手动设置优先,其次配置企业(ENT);都没有则不导出,沿用选区登录默认
	// (选区输出的是机器默认值如「TOPENT = 99」;作业运行/数据库连接都以 TOPENT 为准)
	// 连接时(bootHost)已下发过一次,这里再下发是为了覆盖"会话开着时在设置页改了配置"的情况。
	if ent := s.topentForRun(); ent != "" {
		// 单引号包裹并剔除内嵌单引号(值不限文本,防注入/拆词),与 topentShellCmd 同规则
		setup += "export TOPENT='" + strings.ReplaceAll(ent, "'", "") + "'\r\n"
		// 本轮起 shell 里的 TOPENT 即 ent:同步运行时回读值,界面显示的"当前会话 TOPENT"才不滞后
		s.mu.Lock()
		if s.cfg.Runtime != nil {
			s.cfg.Runtime.Topent = ent
		}
		s.mu.Unlock()
	}
	s.pty.Write(setup)
	if err := s.waitRegexp(host.ReShellPrompt, 15*time.Second, "shell 提示符(cd)"); err != nil {
		return err
	}
	// 4. 启动调试(launchProg 为 gzzz_t 解析出的实体程序;{prog} 仍传作业编号,与 gendbg 一致;
	//    接口重放调试时 ArgsOverride = "'<req>' '<rsp>'" 报文文件对)
	args := strings.ReplaceAll(s.cfg.LaunchArgs, "{prog}", s.Prog)
	if s.ArgsOverride != "" {
		args = s.ArgsOverride
	}
	if s.ExtraArgs != "" {
		args += " " + s.ExtraArgs // gzzz004 额外参数(gendbg 语义:拼在程序名后)
	}
	// 启动引用:gzza004 原版优先(如 "$CINi/ainq120_wf"),$变量由选区后 shell 展开为
	// 权威路径——标准/客制由 T100 环境决定,与 r.d/gendbg 完全同源;无引用才回退本地探测
	var runCmd string
	if s.LaunchRef != "" {
		runCmd = fmt.Sprintf("fglrun -d %s.42r %s", s.LaunchRef, args)
	} else {
		runCmd = fmt.Sprintf("fglrun -d 42r/%s.42r %s", launchProg, args)
	}
	log.Printf("[debug] 会话 %s 启动: %s", s.ID, runCmd)
	s.mu.Lock()
	s.shellReady = false // 离开 shell 进入 fglrun/程序
	s.mu.Unlock()
	s.pty.Write(runCmd + "\r")
	if err := s.waitBarePrompt(90*time.Second, "(fgldb) 提示符"); err != nil {
		return err
	}
	// 置停站态但暂不宣告:断点恢复要在提示符上发命令(exec 依赖 stopped 态),
	// 恢复期间 State() 对外仍报 loading,恢复完才发 stopped 事件。否则前端见到
	// stopped 即停止轮询,首帧快照里断点还是空的,要步进一次才能看到缓存断点
	s.mu.Lock()
	s.state = StateStopped
	s.startAt = time.Now()
	s.restoring = true
	s.mu.Unlock()
	s.setStop(s.entryStopInfo())
	// print 元素上限(防 T100 大数组 print 刷爆输出,fgldeb 同款防护)
	if _, err := s.exec("other", fmt.Sprintf("set print elements %d", s.cfg.PrintElements), waitPrompt, 10*time.Second); err != nil {
		s.emitEvent(Event{Type: "log", Text: "set print elements 失败(不影响使用): " + err.Error()})
	}
	s.restoreBreakpoints()
	s.mu.Lock()
	s.restoring = false
	s.mu.Unlock()
	s.emitEvent(Event{Type: "state", State: string(StateStopped)})
	s.emitEvent(Event{Type: "log", Text: "调试会话就绪: " + s.Prog + "@" + s.cfg.Zone})
	return nil
}

// markShellReady 标记 pty 已停在 shell 提示符
func (s *Session) markShellReady() {
	s.mu.Lock()
	s.shellReady = true
	s.mu.Unlock()
}

// Booted 宿主是否已完成首次登录(可复用)
func (s *Session) Booted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.booted
}

// CloneCfg 返回会话连接配置的快照(同环境重启/重连用)
func (s *Session) CloneCfg() *Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	cc := *s.cfg
	return &cc
}

// BootIdle 仅完成登录并回到空闲(不启动调试)——会话「重启/切换/连接」用
func (s *Session) BootIdle(ctx context.Context) error {
	if !s.booted {
		if err := s.bootHost(ctx); err != nil {
			return err
		}
	}
	s.enterIdle("会话已就绪(空闲),可直接启动调试")
	return nil
}

// SetRun 复用宿主的下一轮运行参数(由 Manager 在复用启动前按最新生效配置调用)
func (s *Session) SetRun(cfg *Config, module, prog, runProg, launchRef, extra string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg != nil {
		cc := *cfg
		cc.Runtime = s.cfg.Runtime // 会话内动态路径保留(同一宿主/区域)
		// **别把这条会话的 ENT 弄丢**：整份替换会把它抹成空（本轮请求常常没带 --ent，
		// 前端「运行/重新开始」就是这样）。变空之后 `requireExplicitEnt` 的三个来源
		// （本次 --ent / 会话 override / 会话建立时的值）全部落空 —— 于是**第一次启动成功、
		// 之后每次启动都 409**，界面停在 idle，人看到的是"重新开始之后步入/步过全灰"
		// （实测：鼎捷1/鼎捷2 首次启动后 cfg.Topent 变空，再启动必 409）。
		// 会话建立时显式给的 ENT 是这条会话自己的事实（TopentSession 的语义），保留它。
		if strings.TrimSpace(string(cc.Topent)) == "" {
			cc.Topent = s.cfg.Topent
		}
		s.cfg = &cc
		s.envName = cc.EnvName() // 环境名跟着配置走,不然快照会报旧环境
	}
	s.Module = module
	s.Prog = prog
	s.RunProg = runProg
	s.LaunchRef = launchRef
	s.ExtraArgs = extra
	s.ArgsOverride = ""
	s.custModule = ""
	// 复用宿主跑新一轮 = **换代**（契约 C5.3 / F15）：宿主没有重登，但"这一轮运行"是新的，
	// 调用方手上那个旧句柄指的是上一轮 —— 换了代，它当场失效（SESSION_STALE）。
	// 序号 `n` 与别名 `name` 不动：它们标识的是"这条会话"，跨轮不变（C2.0 / C2.5b）。
	s.epoch = nextSessionEpoch()
}

// maxReplayRespBytes 回显重放响应时的字节上限(超了只回头部)
const maxReplayRespBytes = 32 << 10

// ReplayResponse 本次重放产生的响应原文(空 = 还没跑完 / 不是重放会话)
func (s *Session) ReplayResponse() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.replayResponse
}

// loadReplayResponse 程序跑完后把本次重放的响应报文读回来,只读一次。
//
// 为什么只能在这里读:重放的响应是**程序退出时**才写到服务器临时文件里的,
// 停在入口或断点时那个文件还不存在。以前想看它只能"停在收尾断点再 continue 到 exit",
// 从程序 stdout 里抓 —— 这次把它接成一条正常通道。
func (s *Session) loadReplayResponse() {
	s.mu.Lock()
	p := s.replayRspPath
	already := s.replayResponse != ""
	s.mu.Unlock()
	if p == "" || already {
		return
	}
	data, _, err := s.ReadFile(p)
	if err != nil || len(data) == 0 {
		return
	}
	txt := string(data)
	if len(txt) > maxReplayRespBytes {
		txt = txt[:maxReplayRespBytes] + "…(响应过长,已截断)"
	}
	s.mu.Lock()
	s.replayResponse = txt
	s.mu.Unlock()
	s.emitEvent(Event{Type: "log", Text: "本次重放产生的响应:" + txt})
}

// TopentOverride 会话内手动设置的 TOPENT(空 = 未设置,按配置默认)
func (s *Session) TopentOverride() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.topentOverride
}

// TopentSession 本会话的 TOPENT(用户显式给的那个:tt debug start --ent / 界面面板)。
//
// **不再是"配置级"值**:环境配置里的企业编号已移除(hosts.sshs[].topent),所以这个值
// 只有两个来源 —— 启动时显式给,或会话内用 SetTopent 改。空 = 用户没给,
// 此时企业由**登录 shell 自己的 $TOPENT** 决定(见 TopentShell)。
func (s *Session) TopentSession() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.TrimSpace(string(s.cfg.Topent))
}

// TopentShell 登录后 shell 里实际回读到的 $TOPENT(选区/环境脚本给的企业;空 = 登录未导出)。
// 与 TopentSession 的区别:用户没显式给 TOPENT 时,会话不会去 export ——
// 所以"当前这台机器连的是哪个企业"以本值为准。
func (s *Session) TopentShell() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cfg.Runtime == nil {
		return ""
	}
	return strings.TrimSpace(s.cfg.Runtime.Topent)
}

// topentForRun 本轮运行生效的 TOPENT:会话内手动设置优先,其次启动时显式给的值,
// 再否则**不导出**(沿用登录 shell 自己的 $TOPENT)。
func (s *Session) topentForRun() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.topentOverride != "" {
		return s.topentOverride
	}
	return strings.TrimSpace(string(s.cfg.Topent))
}

// TopentIntForDB 该会话实际生效的**企业编号**,供只读 SQL 决定用哪个账号连库。
//
// 非数字(例如把据点码填进了企业编号的位置)返回 0 —— 调用方要据此明确拒绝,
// 而不是拿一个错的编号去查库:查错 schema 的结果是 0 行,而 0 行会被读成
// "这条业务数据不存在",那是最危险的错误结论。
func (s *Session) TopentIntForDB() int {
	n, _ := host.EntValue(s.topentForRun()).Int()
	return n
}

// SetTopent 空闲态重新设置宿主 shell 的 TOPENT(空值 = 清除手动覆盖,回到环境默认 TOPENT)。
// 立即 export 到 shell(清除时下发环境默认值,环境也没配才 unset),随后的各轮调试仍按
// topentForRun 优先导出覆盖值。
func (s *Session) SetTopent(value string) error {
	s.setTopentOverride(value)
	// 清除覆盖(空值)不能简单 unset:连接时已把环境默认下发进 shell,unset 会掉回选区机器默认,
	// 与"会话企业 = 设置页配置"不一致。这里直接下发 topentForRun()(覆盖值 || 配置企业;
	// 都没有才是空 = unset),让宿主 shell 的 TOPENT 恒等于当前生效值。
	eff := s.topentForRun()
	if err := s.applyTopentToShell(eff); err != nil {
		return err
	}
	if value == "" {
		if eff != "" {
			s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("已清除 TOPENT 覆盖,回到环境默认 %q", eff)})
		} else {
			s.emitEvent(Event{Type: "log", Text: "已清除 TOPENT 覆盖(环境未配置,shell 内已 unset)"})
		}
		return nil
	}
	s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("TOPENT 已设置为 %q(会话内,立即生效)", value)})
	return nil
}

// applyTopentToShell 把 value 直接 export/unset 到宿主 shell(空值 = 清除)。
// 只动 shell、不记录 override:SetTopent(会话内手动设置)与 bootHost(连接时下发环境默认)共用。
// 回执超时只记日志不报错(命令通常已生效);写入终端失败才返回错误。
func (s *Session) applyTopentToShell(value string) error {
	if err := s.pty.Write(topentShellCmd(value)); err != nil {
		return fmt.Errorf("写入终端失败: %w", err)
	}
	// 等待 shell 回执(确认命令已被执行)
	if err := s.waitRegexp(reTopentOK, 5*time.Second, "TOPENT 设置回执"); err != nil {
		s.emitEvent(Event{Type: "log", Text: "TOPENT 已下发但未等到 shell 回执: " + err.Error()})
	}
	// shell 里的 TOPENT 已变成 value(或已 unset):同步运行时回读值,
	// 使界面显示的"当前会话 TOPENT"就是 shell 真实值,而不是登录那一刻的旧值。
	s.mu.Lock()
	if s.cfg.Runtime != nil {
		s.cfg.Runtime.Topent = value
	}
	s.mu.Unlock()
	return nil
}

// topentShellCmd 拼出把 TOPENT 下发到宿主 shell 的命令(空值 = unset)。
// 值不限数字/文本:shell 单引号包裹并剔除内嵌单引号,防注入/拆词。
func topentShellCmd(value string) string {
	if value == "" {
		return "unset TOPENT; echo " + reTopentOKMark + "\r"
	}
	q := "'" + strings.ReplaceAll(value, "'", "") + "'"
	return "export TOPENT=" + q + "; echo " + reTopentOKMark + "\r"
}

// reTopentOKMark TOPENT 下发回执标记(TDBG 是本工具自己的本地标记,非远端变量名)
const reTopentOKMark = "TDBG-TOPENT-OK"

// reTopentOK 回执标记:必须匹配独占一行的回执。
// PTY 会把我方发去的整条命令原样回显,该行同样含有本标记字符串 ——
// 用子串匹配会在命令被 shell 执行之前就"等到回执",锚定整行才能排除命令回显本身。
var reTopentOK = regexp.MustCompile(`^\s*` + reTopentOKMark + `\s*$`)

// enterIdle 把会话置为空闲(宿主 shell 保留、无调试运行)并广播 state=idle。
// 程序自然退出 / 结束调试 / 复用启动失败都会回到这个状态。
func (s *Session) enterIdle(logMsg string) {
	s.mu.Lock()
	if s.state == StateExit {
		s.mu.Unlock()
		return
	}
	wasIdle := s.state == StateIdle
	s.state = StateIdle
	s.pending = nil
	s.collect = nil
	s.quitReq = false
	s.cur = StopInfo{}
	s.curFrame = 0
	s.lastAutovars = nil
	s.mu.Unlock()
	s.stopWatchdog()
	// 重放调试:程序跑完了,把它的响应报文读回来(以前只能靠"停在收尾断点再 continue"
	// 从 stdout 里抓,很绕)。异步读,别堵住协议泵。
	go s.loadReplayResponse()
	if !wasIdle {
		s.emitEvent(Event{Type: "state", State: string(StateIdle)})
	}
	if logMsg != "" {
		s.emitEvent(Event{Type: "log", Text: logMsg})
	}
}

// reProgName 作业名白名单(用于拼 shell 命令,防注入)
var reProgName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

// pickLaunchDir 决定启动目录:客制目录 42r 有同名 42r 用客制,否则标准模块目录。
// 与原版 T100 一致:转客制作业从 c** 启动,客制目录 = 模块首字母 a→c(ain→cin、apm→cpm)
func (s *Session) pickLaunchDir(prog string) string {
	modDir := s.cfg.ModuleDir(s.Module)
	if s.Module == "" || !reProgName.MatchString(s.Module) || !strings.HasPrefix(s.Module, "a") || !strings.HasSuffix(modDir, "/"+s.Module) {
		return modDir
	}
	cust := strings.TrimSuffix(modDir, "/"+s.Module) + "/c" + s.Module[1:]
	out, _ := s.conn.Output(fmt.Sprintf("ls %s/42r/%s.42r 2>/dev/null", cust, prog), 10*time.Second)
	if strings.Contains(out, prog+".42r") {
		s.mu.Lock()
		s.custModule = "c" + s.Module[1:] // 客制模块名(apm→cpm):DVM 停站报的就是它
		s.mu.Unlock()
		s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("转客制作业:使用客制目录 %s", cust)})
		return cust
	}
	return modDir
}

// entryStopInfo 构造入口停站:带上实际启动的源文件(标准/客制),让前端入口就加载
// 与 DVM 一致的文件,第一次步进不再跨文件重载
func (s *Session) entryStopInfo() *StopInfo {
	si := &StopInfo{Reason: "entry"}
	mod := s.Module
	// 实体程序(gzzz_t 解析结果)优先;解析不到就退回启动时用的程序名。
	//
	// **WS 服务程序(wssp*/awsp*)根本不在 gzzz_t 里**(实测该表 5416 行中 wssp% 命中 0 条),
	// 所以按作业名启动/重放这类程序时 RunProg 必定为空。以前这里只看 RunProg,于是这类
	// 会话的入口停站不带文件名 —— 而前端"起好会话、页面再接上来"走的是 syncFromSessions,
	// 它只在有文件名时才去取源码,结果代码区一片空白(只有行号 1)。
	// 前端 refreshSource 早就写着 `runProg || prog` 的同样兜底,这里补齐,两边语义一致。
	prog := s.RunProg
	if prog == "" {
		prog = s.Prog
	}
	if mod == "" || !reProgName.MatchString(mod) || prog == "" {
		return si
	}
	// GP 的 prog 自带模块前缀（csf_csfi300_icd），再拼一次会变成 csf_csf_csfi300_icd.4gl ——
	// 停站文件的镜像与源码定位都会因此落空。已经带前缀就不再加。
	if (s.cfg != nil && strings.EqualFold(s.cfg.Product, "gp")) ||
		strings.HasPrefix(strings.ToLower(prog), strings.ToLower(mod)+"_") {
		si.File = prog + ".4gl"
		return si
	}
	if s.custModule != "" {
		si.File = s.custModule + "_" + prog + ".4gl" // cpm_apmt520_wf.4gl
	} else {
		si.File = mod + "_" + prog + ".4gl" // apm_apmt520_wf.4gl
	}
	return si
}

// readRuntimeEnv 在选区后的 shell 里回显关键 T100 环境变量并写入 cfg.Runtime。
// 选区后是真实登录环境(与标准 debug 完全同源),是路径的最权威来源;
// 失败由调用方(bootHost)作为登录失败报错——T100 路径无静态配置可回退。
// 探针与解析共用 host 层的 TEnvProbe/TEnvParser:定界符之间只回显真实变量名,
// 且只有区间内的取值行才被采纳(命令回显与终端重绘碎片都不会污染取值)。
func (s *Session) readRuntimeEnv() error {
	if err := s.pty.Write(host.TEnvProbe() + "\r"); err != nil {
		return err
	}
	var p host.TEnvParser
	deadline := time.After(10 * time.Second)
	for {
		select {
		case ln := <-s.lines:
			if p.Feed(strings.TrimSpace(ln)) {
				env := p.Env()
				if !env.Valid() {
					return fmt.Errorf("未回显到 TOP/ERP(环境脚本未设置 $TOP?)")
				}
				env.FetchedAt = time.Now()
				s.mu.Lock()
				s.cfg.Runtime = env
				s.mu.Unlock()
				return nil
			}
		case <-deadline:
			return fmt.Errorf("回读 T100 环境变量超时")
		}
	}
}

// fglsourcePathOf 按实际启动目录拼源码搜索路径(公共库仍在 TOP/com 下,TOP 为登录动态获取)
func (s *Session) fglsourcePathOf(dir string) string {
	top := s.cfg.TopDirActual()
	dirs := []string{
		dir + "/4gl",
		dir + "/42m",
		top + "/com/lib/42m",
		top + "/com/sub/42m",
		top + "/com/qry/42m",
	}
	out := ""
	for i, d := range dirs {
		if i > 0 {
			out += ":"
		}
		out += d
	}
	return out
}

// resolveModule 按程序名在登录区源码目录各模块的 42r 目录中搜索 <prog>.42r:
// 唯一命中 → 返回模块;标准/客制(ain/cin)并存 → 选客制(转客制后原版从 c** 目录启动);
// 其余多命中 → 报出候选;无命中 → 报错
func (s *Session) resolveModule(prog string) (string, error) {
	if !reProgName.MatchString(prog) {
		return "", fmt.Errorf("作业名含非法字符: %q", prog)
	}
	found := searchModule42r(s.conn, s.cfg.ModuleRootsActual(), prog)
	switch {
	case len(found) == 0:
		// 防御:会话登录(bootHost)已保证动态环境解析成功;此处仅兜底提示
		if s.cfg.Runtime == nil || !s.cfg.Runtime.Valid() {
			return "", fmt.Errorf("会话缺少登录后动态解析的 T100 路径(环境 %s,zone %s),无法定位作业 %s;请重启会话后重试", s.envName, s.cfg.Zone, prog)
		}
		return "", fmt.Errorf("在各模块 42r 目录中未找到作业 %s(请检查作业名)", prog)
	case len(found) > 1:
		// 标准模块与它的客制目录(首字母 a→c,如 ain/cin)并存视为同一作业,取客制
		for _, m := range found {
			if strings.HasPrefix(m, "c") {
				for _, std := range found {
					if "c"+std[1:] == m {
						return m, nil
					}
				}
			}
		}
		return "", fmt.Errorf("作业 %s 存在于多个模块(%s),请指定模块", prog, strings.Join(found, ", "))
	}
	return found[0], nil
}

// searchModule42r 在各模块根的 42r 目录中按 <prog>.42r 搜索,返回去重后的模块列表
// (模块根直挂的 42r 记为 "";ls 部分 glob 未命中返回非零退出码,不能当作失败,只按输出解析)
func searchModule42r(conn *host.SSHConn, roots []string, prog string) []string {
	if !reProgName.MatchString(prog) {
		return nil
	}
	var pats []string
	for _, root := range roots {
		pats = append(pats, root+"/*/42r/"+prog+".42r", root+"/42r/"+prog+".42r")
	}
	out, _ := conn.Output("ls -d "+strings.Join(pats, " ")+" 2>/dev/null", 20*time.Second)
	mods := map[string]bool{}
	var found []string
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimSpace(ln)
		if !strings.HasSuffix(ln, "/42r/"+prog+".42r") {
			continue
		}
		parts := strings.Split(ln, "/")
		mod := ""
		for i, p := range parts {
			if p == "42r" && i > 0 {
				mod = parts[i-1]
				break
			}
		}
		if mod == "erp" || mod == "com" {
			mod = "" // 模块根直挂的 42r
		}
		if !mods[mod] {
			mods[mod] = true
			found = append(found, mod)
		}
	}
	return found
}

// modHas42r 校验 <topDir>/erp/<mod>/42r/<prog>.42r 是否存在
func modHas42r(conn *host.SSHConn, topDir, mod, prog string) bool {
	if mod == "" || !reProgName.MatchString(mod) || !reProgName.MatchString(prog) {
		return false
	}
	out, _ := conn.Output(
		fmt.Sprintf("ls %s/erp/%s/42r/%s.42r 2>/dev/null", topDir, mod, prog), 10*time.Second)
	return strings.Contains(out, prog+".42r")
}

// EndRun 结束本轮调试,保留宿主会话(回到 idle):
// 运行中先中断拿回控制权,再向 fgldb 发 quit 回到 shell,连接不释放。
// 若拿不回控制权(卡死)或状态异常,降级为整体断开 Close()。
func (s *Session) EndRun() error {
	if s.State() == StateExit || s.State() == StateIdle {
		return nil
	}
	// 收口互斥（执行者 / 等待者）由 withQuit 承包：**复位写在那里，调用方漏不掉**。
	return s.withQuit(s.endRunAsOwner)
}

// endRunAsOwner 是"已经拿到执行权"之后的本体。
func (s *Session) endRunAsOwner() error {
	st := s.State()
	if st == StateRunning {
		_ = s.Interrupt()
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if s.State() == StateStopped {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
	if s.State() != StateStopped {
		// running 拿不回控制权 / loading 启动中:没有可保留的干净宿主,整体断开
		return s.closeNow()
	}
	r, err := s.exec("other", "quit", waitPrompt, 20*time.Second)
	if err != nil || !r.SawShell {
		// quit 未正常回到 shell(状态不可靠):整体断开兜底
		if err == nil {
			err = fmt.Errorf("quit 后未回到 shell")
		}
		s.emitEvent(Event{Type: "log", Text: "结束本轮调试异常,改为断开会话: " + err.Error()})
		return s.closeNow()
	}
	s.markShellReady()
	s.enterIdle("结束调试:本轮运行已结束,会话保留(可直接再次启动)")
	return nil
}

// beginQuit 抢占"收口执行者"这个角色。
// mine=true 表示本次调用是执行者（拿到做了要 close 的 done）；false 表示已有人在收口，
// 调用方应当 waitQuit(done) 等它做完 —— 而不是立刻回"成功"（见 quitDone 的注释）。
func (s *Session) beginQuit() (chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quitting {
		return s.quitDone, false
	}
	s.quitting = true
	s.quitDone = make(chan struct{})
	return s.quitDone, true
}

// withQuit 以"收口执行者"的身份跑 fn：**保证**无论 fn 从哪条路径返回都会复位标志。
//
// 为什么不让调用方自己 beginQuit/endQuit：2026-10-23 的 bug 正是"只置 true、忘了复位"，
// 结果每条会话的「结束调试 / 重新开始」只有第一次真的生效，之后全返回假成功（200），
// 状态留在 stopped，紧接着的启动撞"已存在调试会话"。靠"记得写 defer"防不住下一个人
// 再加一条返回路径，所以把配对关系收进这个函数 —— 结构上漏不掉。
func (s *Session) withQuit(fn func() error) error {
	done, mine := s.beginQuit()
	if !mine {
		return s.waitQuit(done)
	}
	defer s.endQuit(done)
	return fn()
}

// endQuit 收口执行者返回时调用：**先清标志，再关信号**。
//
// 顺序不能反。等待者（`beginQuit` 拿到 `!mine` 的那一方）是在 `<-done` 上等的，
// 一旦被唤醒它会立刻按"收口已完成"往下走；若此刻 `quitting` 还是 true，它会再进
// `beginQuit` 又拿到那个**已经关闭**的 done，于是 `waitQuit` 立刻返回 nil —— 报成功，
// 而实际上一次收口都没发生。
//
// 这个 bug 的实际后果（2026-10-23 实测）：`quitting` 从前**只置 true、从不复位**，
// 于是每条会话的「结束调试 / 重新开始」只有**第一次**真的生效；之后再点都返回 200
// 假成功，状态留在 stopped，紧接着的启动撞上"已存在调试会话"，界面停在 idle，
// 人看到的是"重新开始之后步入/步过全灰"。
func (s *Session) endQuit(done chan struct{}) {
	s.mu.Lock()
	s.quitting = false
	s.quitDone = nil
	s.mu.Unlock()
	if done != nil {
		close(done)
	}
}

// waitQuit 等一次在飞的收口做完，并把"做完之后状态是否真的可以干活"如实报出来。
func (s *Session) waitQuit(done chan struct{}) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		return fmt.Errorf("等待上一次结束调试超时(30s)")
	}
	// 收口做完了：调用方要看到真实状态（idle / stopped / exit 都算结束，running / loading 不算）
	if st := s.State(); st == StateRunning || st == StateLoading {
		return fmt.Errorf("结束调试没有生效(当前 %s)", st)
	}
	return nil
}

// Close 彻底结束会话并释放 SSH(切换环境/重启会话/服务退出/异常兜底用):
// 必要时先中断,再向 fgldb quit,最后关闭连接。
func (s *Session) Close() error {
	return s.withQuit(s.closeNow)
}

// closeNow 是 Close 的实体：**不带**"已有人在收口"的前置判断。
// 拆出来是因为 EndRun 已经当过执行者了，它的兜底路径要能直接断开而不自锁。
func (s *Session) closeNow() error {
	st := s.State()

	if st == StateRunning {
		s.Interrupt()
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			if s.State() == StateStopped {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
	}
	if s.State() == StateStopped {
		s.exec("other", "quit", waitPrompt, 20*time.Second)
	}
	s.forceExit()
	return nil
}

// Interrupt 运行中发 SIGINT 拿回控制权
func (s *Session) Interrupt() error {
	s.mu.Lock()
	st := s.state
	started := s.started
	s.mu.Unlock()
	if !started {
		return fmt.Errorf("程序尚未启动,无需中断:直接「继续」即可运行,或点行号下断点")
	}
	if st != StateRunning {
		return fmt.Errorf("仅运行中可中断(当前 %s)", st)
	}
	s.emitEvent(Event{Type: "log", Text: "发送 SIGINT 中断"})
	return s.pty.Write("\x03")
}

// ForceExit 供 API 层终结失败会话(置 exit 态并释放资源)
func (s *Session) ForceExit() { s.forceExit() }

// WhyResult 「程序此刻到底在干什么」的探测结论
type WhyResult struct {
	WaitingForUser bool   `json:"waitingForUser"`
	Kind           string `json:"kind,omitempty"`   // 交互语句类别(仅 WaitingForUser 时非空)
	Evidence       string `json:"evidence"`         // 判断依据(人话,可直接转述给用户)
	Reason         string `json:"reason,omitempty"` // 停站原因
	Func           string `json:"func,omitempty"`
	File           string `json:"file,omitempty"`
	Line           int    `json:"line,omitempty"`
	SourceText     string `json:"sourceText,omitempty"` // 停站那一行的原文
	Resumed        bool   `json:"resumed"`              // 探测后是否已放回运行
	Risk           string `json:"risk,omitempty"`       // 未放回时的提醒
}

// Why 探测程序此刻是在等用户操作、还是在空转 —— 把 SKILL.md 里教 AI 人肉做的
// 三步(interrupt → where → 看停在哪一行)搬进服务端,一次调用给出结论。
//
// 三个要点:
//  1. 已经停在停站态就直接分类,**不发任何命令**(零副作用),不会打扰正在等用户的程序。
//  2. 只有运行中才发 SIGINT。语义随前端模式而变:TUI 的 singular dialog 会被
//     **取消**(int_flag 置位、离开该语句);GUI/GDC 不会取消当前对话框。
//     另外没有 DEFER INTERRUPT 时 SIGINT 默认直接终止进程 —— 现在只挂起,
//     是因为 fglrun -d 的调试器拦截了它,这个安全边界是调试器给的,不是协议保证的。
//  3. 默认探测完自动放回运行:探测本身会暂停程序,而 GDC 上用户的对话框还等着,
//     挂着不动用户就没法操作。要保留现场传 resume=false。
func (s *Session) Why(resume bool) (*WhyResult, error) {
	s.mu.Lock()
	st, started := s.state, s.started
	s.mu.Unlock()
	if !started {
		return nil, fmt.Errorf("程序尚未启动:先 start 或 run 再探测")
	}
	if st != StateStopped && st != StateRunning {
		return nil, fmt.Errorf("当前状态 %s 不可探测", st)
	}

	// 只有"是我们把它中断下来的"才需要我们负责放回。本来就停在断点上的程序
	// 绝不能因为一次探测被放跑 —— 那会直接冲过调用方精心设的断点。
	interrupted := false
	if st == StateRunning {
		if err := s.Interrupt(); err != nil {
			return nil, err
		}
		if _, err := s.WaitForStop(15 * time.Second); err != nil {
			return nil, fmt.Errorf("中断后 15s 内没拿回控制权(程序可能没响应 SIGINT):%w", err)
		}
		interrupted = true
	}

	stop := s.Cur()
	// interrupt 停站常常没有位置头(File 为空,行号只能从 `->` 行补):补一次 where
	// 拿 Frame[0] 的 File:Line。这正是 SKILL.md 让 AI 手工敲 where 的那一步。
	if stop.File == "" || stop.Line == 0 {
		if frames, err := s.Where(); err == nil && len(frames) > 0 {
			f := frames[0]
			if stop.File == "" {
				stop.File = f.File
			}
			if stop.Line == 0 {
				stop.Line = f.Line
			}
			if stop.Func == "" {
				stop.Func = f.Func
			}
		}
	}

	text := CurSourceText(stop.Source)
	if text == "" {
		text = s.sourceLineAt(stop.File, stop.Line)
	}
	kind := ClassifyInteractiveLine(text)

	res := &WhyResult{
		WaitingForUser: kind != "",
		Kind:           kind,
		Reason:         stop.Reason,
		Func:           stop.Func,
		File:           stop.File,
		Line:           stop.Line,
		SourceText:     strings.TrimSpace(text),
	}
	switch {
	case kind != "":
		res.Evidence = fmt.Sprintf("停在交互语句 %s 上:程序已把控制权交给界面,在等用户操作", kind)
	case strings.TrimSpace(text) == "":
		res.Evidence = "拿不到停站那一行的源码,无法判断(不下结论)"
	case stop.Reason == "interrupt":
		res.Evidence = "停在普通语句上,不是在等用户 —— 多半是慢查询/慢循环或长流程"
	default:
		res.Evidence = "停在普通语句上(断点命中),不是在等用户"
	}

	// 只对"是我们把它中断下来的"负责放回
	if interrupted && resume && s.State() == StateStopped {
		if _, err := s.Continue(); err != nil {
			res.Evidence += fmt.Sprintf("(自动放回失败,程序仍停在调试器上: %v)", err)
		}
	}
	res.Resumed = s.State() == StateRunning
	if interrupted && !res.Resumed {
		res.Risk = "程序已停在调试器上:TUI 前端的对话框可能已被取消;GDC 上用户在程序挂起期间也无法继续操作。" +
			"处理完请 continue 放回,或下次直接 why(默认自动放回)"
	}
	return res, nil
}

// sourceLineAt 读远端源码第 line 行(1-based)的原文;取不到返回空串。
// 走 ResolveSource,命中 mtime 校验的 srcCache —— 反复停站不会反复拉 SFTP。
func (s *Session) sourceLineAt(dvmFile string, line int) string {
	if dvmFile == "" || line <= 0 {
		return ""
	}
	sf, err := s.ResolveSource(dvmFile, s.Module)
	if err != nil || sf == nil {
		return ""
	}
	lines := strings.Split(sf.Content, "\n")
	if line > len(lines) {
		return ""
	}
	return lines[line-1]
}

// forceExit 强制进入退出态并释放资源
func (s *Session) forceExit() {
	s.mu.Lock()
	already := s.state == StateExit
	s.state = StateExit
	p := s.pending
	s.pending = nil
	s.mu.Unlock()
	s.stopWatchdog() // 停掉全部看门狗计时（主/预警/重试）
	if p != nil {
		p.res <- &execResult{Cmd: p.cmd, SawShell: true}
	}
	if !already {
		s.emitEvent(Event{Type: "state", State: string(StateExit)})
		s.emitEvent(Event{Type: "log", Text: "会话已断开,SSH 连接已释放(下次启动将重新登录)"})
	}
	// 先停心跳再关连接:这条连接是本会话独占的,关闭属于正常释放而非掉线
	if s.kaStop != nil {
		s.kaStop()
	}
	// 两条 nil 判据不是"防御性冗余"：**会话池里的会话未必都开过 pty/conn** ——
	// 单测里手工造的会话（`idleOn` / `fakeIdleSession`）就是这种形状，而 WP6 之后
	// `prepareSession` 的收口路径（换目标/换名字时收掉一条空闲宿主）会真的走到这里
	// （`Close` → `closeNow` → `forceExit`）。从前它只在"单会话 + 换目标"时可达，
	// 如今那是一条常走的路（`session_reuse_test.go` 那条"别把手工造的 Session 交给
	// prepareSession"的告诫说的就是这件事）。nil 时无资源可释放，跳过即可。
	if s.pty != nil {
		s.pty.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
}

// ---------- 命令执行 ----------

// execOpts 命令执行的可选语义。零值 = 原有行为。
type execOpts struct {
	// SoftWait > 0 时启用「软等待」:到点后若程序仍在跑,直接返回带 SoftTimeout 的
	// 结果,**不发 SIGINT、不取消命令**。区别于 timeout 的硬语义(超时发 \x03 探测)。
	// 只给 resume 类命令(continue/run/step/finish/until)用:print 之类的请求/响应
	// 命令永远走硬路径,这样 autovars 后台求值不受影响。
	SoftWait time.Duration
}

// exec 在 STOPPED 态发送一条命令并等待响应。
// 若上一条命令仍在执行(如 autovars 后台求值),最多宽限 1.5s 再报忙。
func (s *Session) exec(kind, cmd string, mode int, timeout time.Duration) (*execResult, error) {
	return s.execOpt(kind, cmd, mode, timeout, execOpts{})
}

// execOpt 带可选语义的 exec(见 execOpts)。21 个既有调用点走 exec,语义不变。
func (s *Session) execOpt(kind, cmd string, mode int, timeout time.Duration, o execOpts) (*execResult, error) {
	// 注入通道（见 Session.execFn 的注释）：nil 时才往下走真实 PTY。
	if fn := s.execFn; fn != nil {
		return fn(kind, cmd, mode, timeout)
	}
	busyDeadline := time.Now().Add(1500 * time.Millisecond)
	var p *pendingCmd
	for {
		s.mu.Lock()
		if s.state != StateStopped {
			st := s.state
			s.mu.Unlock()
			if st == StateRunning {
				return nil, ErrNotStopped
			}
			// 空闲/已断开会话:不是"参数写错了",是"这轮已经没有了" —— 别让调用方
			// 对着 usage 反复改参数(真机上有人这么绕过两回)
			if st == StateIdle || st == StateExit {
				return nil, fmt.Errorf("这轮调试已经结束(会话 %s),没有可发命令的停站现场;"+
					"请重新 tt debug wsdebug <rowid> 或 tt debug start <作业> 起一轮", st)
			}
			return nil, fmt.Errorf("当前状态 %s 不可发送命令", st)
		}
		if s.pending == nil {
			p = &pendingCmd{cmd: cmd, kind: kind, mode: mode, res: make(chan *execResult, 1), startedAt: time.Now()}
			s.pending = p
			s.mu.Unlock()
			break
		}
		s.mu.Unlock()
		if time.Now().After(busyDeadline) {
			return nil, fmt.Errorf("上一条命令仍在执行")
		}
		time.Sleep(50 * time.Millisecond)
	}

	if err := s.pty.Write(cmd + "\r"); err != nil {
		s.clearPending()
		return nil, fmt.Errorf("写入终端失败: %w", err)
	}

	// 放行类命令一发出,程序就不在调试器上了 —— 状态必须如实翻 running。
	// 不能只靠输出判定:fgldb 只有 continue 会打 "Continuing.",step/next 不打。
	// 假装还停在 stopped 的后果很实际:界面上显示"已停站 · 停留 Ns"、interrupt
	// 被自己的 state==Running 检查拒掉 —— 程序卡在对话框里时连中断都发不出去。
	if IsResumeCmd(cmd) {
		s.setState(StateRunning)
	}

	hardCtx, hardCancel := context.WithTimeout(context.Background(), timeout)
	defer hardCancel()

	if o.SoftWait <= 0 {
		select {
		case r := <-p.res:
			return r, nil
		case <-hardCtx.Done():
			s.clearPending()
			s.recoverAfterTimeout(cmd)
			return nil, fmt.Errorf("命令超时(%s): %s", timeout, cmd)
		}
	}

	soft := time.NewTimer(o.SoftWait)
	defer soft.Stop()
	select {
	case r := <-p.res:
		return r, nil
	case <-soft.C:
		// 放弃等待(不是取消命令):标记后 onStop 会把这条停站走异步分支,
		// 程序照常运行,后续停站与事件都不丢。不发 \x03。
		if !s.abandonPending(p) {
			// 竞态:标记前命令刚被完成/清理,退回等它的结果(或硬超时)
			select {
			case r := <-p.res:
				return r, nil
			case <-hardCtx.Done():
				s.clearPending()
				s.recoverAfterTimeout(cmd)
				return nil, fmt.Errorf("命令超时(%s): %s", timeout, cmd)
			}
		}
		return &execResult{Cmd: cmd, SoftTimeout: true}, nil
	case <-hardCtx.Done():
		s.clearPending()
		s.recoverAfterTimeout(cmd)
		return nil, fmt.Errorf("命令超时(%s): %s", timeout, cmd)
	}
}

// abandonPending 标记「已放弃等待这条命令」,并如实把状态翻成 running。
// 槽位**不在这里清**:命令仍在飞,onStop 收口时要靠 softAbandoned 区分 sync/async 路径。
func (s *Session) abandonPending(p *pendingCmd) bool {
	s.mu.Lock()
	if s.pending != p {
		s.mu.Unlock()
		return false
	}
	p.softAbandoned = true
	changed := false
	if s.state == StateStopped {
		// 必须如实翻 running:否则下一条 exec 会在 state==Stopped 的假象下
		// 把命令写进正在运行程序的输入缓冲,在下一个提示符处被消费 = 延迟命令注入。
		s.state = StateRunning
		s.runningSince = time.Now()
		changed = true
	}
	s.mu.Unlock()
	if changed {
		s.stopWatchdog() // 与 setState(StateRunning) 一致:离开停站态取消看门狗
		s.emitEvent(Event{Type: "state", State: string(StateRunning)})
	}
	return true
}

// RunningSince 进入 running 的时刻(零值 = 当前不在运行)
func (s *Session) RunningSince() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != StateRunning {
		return time.Time{}
	}
	return s.runningSince
}

// InflightInfo 当前正在执行的那条命令 —— 对外回答"AI 此刻在干什么、跑了多久"。
// 在此之前它只藏在私有的 pending 槽位里,界面完全看不见。
type InflightInfo struct {
	Cmd     string    `json:"cmd"`
	Since   time.Time `json:"since"`
	Elapsed float64   `json:"elapsed"` // 秒
}

// Inflight 取在飞的命令;空闲返回 nil。
func (s *Session) Inflight() *InflightInfo {
	s.mu.Lock()
	p := s.pending
	s.mu.Unlock()
	if p == nil || p.startedAt.IsZero() {
		return nil
	}
	return &InflightInfo{Cmd: p.cmd, Since: p.startedAt, Elapsed: time.Since(p.startedAt).Seconds()}
}

// SilentSeconds 距最近一次协议输出的秒数(用来判断"跑了很久但一点动静都没有")。
// 从未有过输出时退回到 runningSince;都不在运行态返回 0。
func (s *Session) SilentSeconds() float64 {
	s.mu.Lock()
	st, last, since := s.state, s.lastOutputAt, s.runningSince
	s.mu.Unlock()
	if st != StateRunning {
		return 0
	}
	base := last
	if base.IsZero() || (!since.IsZero() && base.Before(since)) {
		base = since
	}
	if base.IsZero() {
		return 0
	}
	return time.Since(base).Seconds()
}

// recoverAfterTimeout 命令超时后探测真实状态:发 SIGINT,
// 若回到 (fgldb) 提示符说明已停站;否则按运行中处理
func (s *Session) recoverAfterTimeout(cmd string) {
	s.emitEvent(Event{Type: "log", Text: "命令超时,发送中断探测状态: " + cmd})
	_ = s.pty.Write("\x03")
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s.State() == StateStopped {
			return
		}
		time.Sleep(150 * time.Millisecond)
	}
	s.setState(StateRunning)
}

func (s *Session) clearPending() {
	s.mu.Lock()
	p := s.pending
	s.pending = nil
	s.mu.Unlock()
	if p != nil {
		select {
		case p.res <- &execResult{Cmd: p.cmd}:
		default:
		}
	}
}

// completePending 归还命令响应(由输出泵调用)
func (s *Session) completePending(r *execResult) {
	s.mu.Lock()
	p := s.pending
	s.pending = nil
	s.mu.Unlock()
	if p == nil {
		return
	}
	if r.Cmd == "" {
		r.Cmd = p.cmd
	}
	if len(r.Lines) == 0 {
		r.Lines = p.lines
	}
	// 截断标记统一在这里带上:调用方(断点/步进/软超时…)有好几处,逐个加容易漏
	if p.linesTrunc {
		r.Truncated = true
		if r.TruncReason == "" {
			r.TruncReason = "lines"
		}
	}
	if p.valueTrunc {
		r.Truncated = true
		r.TruncReason = "value"
	}
	select {
	case p.res <- r:
	default:
	}
}

// Continue 继续运行到下一停站
func (s *Session) Continue() (*execResult, error) {
	if !s.Started() {
		// 入口停站:程序尚未 run,fgldb pre-run 态 continue 非法("The program is not being run"),
		// 透明等效 run(从入口启动,跑到第一个停站),符合"点继续=让程序跑起来"的直觉
		return s.Run()
	}
	r, err := s.exec("continue", "continue", waitMarker, 30*time.Second)
	if err == nil {
		s.markStarted()
	}
	return r, err
}

// Run 从入口启动程序
func (s *Session) Run() (*execResult, error) {
	r, err := s.exec("run", "run", waitQuiet, 60*time.Second)
	if err == nil {
		s.markStarted()
	}
	return r, err
}

// markStarted 标记程序已启动(步进类命令此后才合法)
func (s *Session) markStarted() {
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()
}

// Started 程序是否已 run 过
func (s *Session) Started() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

// Bare 空宿主会话:登录上了,但还没挂任何程序(module/prog 都为空)。
// 它没有"一轮运行"可保护 —— 前端一连上来就会建出这么一个,而它是 human 身份建的
// (→ 纯人工),若把它也算进模式闸门,AI 就再也启动不了任何调试。
func (s *Session) Bare() bool {
	return s.Prog == "" && s.Module == ""
}

// Break 下断点(位置: 行号 / 函数名 / file:line)。
// 注意 fgldb 会把非可执行行的断点自动调整到下一条可执行语句,
// 返回的 bp.Line 才是真实生效位置(可能与请求行号不同)。
// 调整后与既有断点落在同一位置时,撤销新增并返回既有断点(带 Note)。
//
// 带条件/忽略次数/监视点的版本是 BreakWith(A20)。
func (s *Session) Break(loc string) (*Breakpoint, error) {
	return s.BreakWith(BreakpointOpts{Location: loc})
}

// BreakWith 下断点并追加 A20 的选项(条件 / 忽略次数 / 监视点；条件按 A26 内联进 break)。
//
// **校验在发命令之前**:BreakpointOpts 不过就直接返回错误 —— 这条路径上一个字节都不发
// (生成命令的函数是纯函数,拒绝时返回 nil 切片,所以"不发"是结构保证,不是自觉)。
//
// 位置里的模块前缀先过 BPLocContext.ResolveLocation：现场实测 `break asf_axmt500_wf.4gl:2141`
// 必然 `No source file named`（真名 axm_...）。修正用的是**本会话登录后解析到的模块**，
// 事实不足就不改，而是把"怎么给"写进 Notes 里（A15 不猜）。
func (s *Session) BreakWith(o BreakpointOpts) (*Breakpoint, error) {
	o, locNote := s.resolveBPLocation(o)
	first, err := o.BreakCmd()
	if err != nil {
		return nil, err
	}
	r, err := s.exec("break", first, waitPrompt, 20*time.Second)
	if err != nil {
		return nil, err
	}
	var bp *Breakpoint
	for _, ln := range r.Lines {
		if m := reBPSet.FindStringSubmatch(ln); m != nil {
			num, _ := strconv.Atoi(m[1])
			line, _ := strconv.Atoi(m[3])
			bp = &Breakpoint{Num: num, File: m[2], Line: line, Enabled: true}
		} else if m := reWatchSet.FindStringSubmatch(ln); m != nil {
			// 监视点没有 file:line,被监视的表达式存在 Func 里(落盘/恢复都靠它)。
			num, _ := strconv.Atoi(m[1])
			bp = &Breakpoint{Num: num, Func: m[2], Enabled: true}
		}
	}
	if bp == nil {
		return nil, fmt.Errorf("断点设置失败: %s", strings.Join(r.Lines, " | "))
	}
	bp.Cond, bp.Ignore = o.Cond, o.Ignore
	bp.Kind = o.kind()
	if locNote != "" {
		bp.Note = locNote
	}
	if err := s.syncBreakpoints(); err != nil {
		// 账本没重建起来不致命（断点确实下了）：如实记一笔，别让调用方以为"什么都没发生"。
		s.emitEvent(Event{Type: "log", Text: "断点已下，但账本未重建：" + err.Error()})
	}
	// syncBreakpoints 以 `info breakpoints` 为真相重建账本,它**不回显条件**（A27）。
	// 所以这里把发起方给的三项**按编号挂回**账本 —— 否则条件在快照与落盘里都会凭空消失。
	// 删除后下一次 sync 会把它从账本里自然抹掉。
	s.mu.Lock()
	if cur, ok := s.bps[bp.Num]; ok {
		cur.Cond, cur.Ignore, cur.Kind = bp.Cond, bp.Ignore, bp.Kind
		if locNote != "" {
			cur.Note = locNote
		}
	} else {
		s.bps[bp.Num] = bp
	}
	s.mu.Unlock()
	if err := s.applyOptCmds(bp.Num, o); err != nil {
		// 断点本身已经下了:报错但把断点交回去,调用方才知道现状。
		return bp, err
	}
	// 重复防护:fgldb 调整行号后可能与既有断点撞在同一位置(监视点没有行,跳过)
	if bp.Kind != BPKindWatch {
		s.mu.Lock()
		var dup *Breakpoint
		for _, b := range s.bps {
			if b.Num != bp.Num && b.File == bp.File && b.Line == bp.Line {
				dup = b
				break
			}
		}
		s.mu.Unlock()
		if dup != nil {
			if err := s.DeleteBreakpoint(bp.Num); err != nil {
				return dup, fmt.Errorf("重复断点清理失败: %w", err)
			}
			dup.Note = fmt.Sprintf("该位置已有断点 #%d(%s:%d),未重复添加", dup.Num, dup.File, dup.Line)
			return dup, nil
		}
	}
	s.persistBPs()
	return bp, nil
}

// bpLocContext 收集修正断点位置所需的会话事实（登录后解析到的模块 / 客制模块 / 本会话作业名）。
func (s *Session) bpLocContext() BPLocContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := BPLocContext{Module: s.Module, CustModule: s.custModule}
	for _, p := range []string{s.Prog, s.RunProg} {
		if p = strings.TrimSpace(p); p != "" {
			c.Progs = append(c.Progs, p)
		}
	}
	// 入口文件可能是 `<模块>_<RunProg>.4gl` 里那个 RunProg（gzzz_t 解析出的实体程序），
	// 也可能就是启动时给的作业名 —— 两个都收，别的作业不碰。
	if rp := strings.TrimSpace(s.runProgName()); rp != "" {
		c.Progs = append(c.Progs, rp)
	}
	return c
}

// resolveBPLocation 把位置里的模块前缀按会话已知事实修正；返回（新选项, 说明）。
func (s *Session) resolveBPLocation(o BreakpointOpts) (BreakpointOpts, string) {
	if s == nil || o.kind() == BPKindWatch {
		return o, "" // 监视点的"位置"是表达式，没有文件名前缀
	}
	fixed, note := s.bpLocContext().ResolveLocation(o.Location)
	if fixed == o.Location {
		return o, note // 没改位置，但可能带一句"怎么给"的说明
	}
	o.Location = fixed
	return o, note
}

// applyOptCmds 发断点落定后的追加命令（A26：只剩 ignore —— 条件已内联进 break）。
// Ignore=0 ⇒ 这里根本没有 ignore 命令可发(OptCmds 不生成)。
func (s *Session) applyOptCmds(num int, o BreakpointOpts) error {
	cmds, err := o.OptCmds(num)
	if err != nil {
		return err
	}
	for _, c := range cmds {
		if _, err := s.exec("other", c, waitPrompt, 15*time.Second); err != nil {
			return fmt.Errorf("%s 执行失败: %w", c, err)
		}
	}
	return nil
}

// DeleteBreakpoint 删除断点
func (s *Session) DeleteBreakpoint(num int) error {
	_, err := s.exec("other", fmt.Sprintf("delete %d", num), waitPrompt, 15*time.Second)
	if err == nil {
		s.syncBreakpoints()
		s.persistBPs()
	}
	return err
}

// SetBPEnabled 启用/禁用断点(enable/disable)
func (s *Session) SetBPEnabled(num int, enabled bool) error {
	cmd := "disable"
	if enabled {
		cmd = "enable"
	}
	if _, err := s.exec("other", fmt.Sprintf("%s %d", cmd, num), waitPrompt, 15*time.Second); err != nil {
		return err
	}
	s.syncBreakpoints()
	s.persistBPs()
	return nil
}

// inheritMissingBPFields 用旧账本给"这次没回显的字段"补空缺 —— **解析到的优先**。
//
// 判据（Q2，承接 A30）：
//
//	Cond   —— 解析到非空就用解析到的；只有解析到空才用旧账本补；
//	Ignore —— 同上：解析到非 0 优先，解析到 0 才用旧账本补。
//
// 为什么必须单向：A30 之后 fgldb **真的会回显** `stop only if <expr>`。修前这里是
// 无条件覆盖（`nbp.Cond = obp.Cond`），等于刚读到的条件被旧账本换掉，或者（旧账本
// 没这条条件时）被抹成空串 —— 界面与落盘于是长期显示一个早就不成立的条件。
//
// 匹配按**位置 + 种类**（File+Line+Kind；Num 会随 delete/add 漂移，不能当键）。
// 两条细节：
//   - 老存档的 `Kind` 是空串（= break，见 Breakpoint.Kind 的注释），比较前先归一，
//     否则从旧存档恢复出来的账本**永远**补不到字段（静默丢条件）；
//   - 监视点的 File/Line 是空的（位置是表达式本身），所以监视点还要比 `Func`，
//     否则两个不同表达式的监视点会互相串值；同一位置在旧账本里有多条时按 **Num 最小**
//     那条取 —— 与 map 遍历顺序无关，同一份报文两次同步得到同一账本。
func inheritMissingBPFields(parsed []Breakpoint, old map[int]*Breakpoint) {
	if len(old) == 0 {
		return
	}
	olds := make([]*Breakpoint, 0, len(old))
	for _, o := range old {
		if o != nil {
			olds = append(olds, o)
		}
	}
	sort.Slice(olds, func(i, j int) bool { return olds[i].Num < olds[j].Num })
	for i := range parsed {
		nbp := &parsed[i]
		kind := bpKindOf(nbp.Kind)
		for _, obp := range olds {
			if obp.File != nbp.File || obp.Line != nbp.Line || bpKindOf(obp.Kind) != kind {
				continue
			}
			if kind == BPKindWatch && obp.Func != nbp.Func {
				continue // 监视点靠表达式定位，别的表达式不算同一条
			}
			if nbp.Cond == "" {
				nbp.Cond = obp.Cond
			}
			if nbp.Ignore == 0 {
				nbp.Ignore = obp.Ignore
			}
			break
		}
	}
}

// bpKindOf 把断点种类归一：空串就是 break（见 Breakpoint.Kind 的注释）。
func bpKindOf(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return BPKindBreak
	}
	return kind
}

// syncBreakpoints 用 `info breakpoints` 重建断点账本（fgldb 为唯一真相，杜绝漂移）。
//
// A27：解析走纯函数 ParseInfoBreakpoints —— watchpoint 行没有 `hw`、ignore 回显在第二行、
// breakpoint 行是 `in <模块> at <文件>:<行>`。解析不出来时：**账本保持原样**（不拿空表覆盖，
// 否则一次格式意外就会把用户手上的断点全部清空），并把"行格式不认识 + 原文"记进事件日志。
//
// A30/Q2：条件与忽略次数既然都读得回来，就**以 fgldb 为准**，旧账本只补空缺
// （判据全在纯函数 `inheritMissingBPFields` 里）。
func (s *Session) syncBreakpoints() error {
	r, err := s.exec("other", "info breakpoints", waitPrompt, 15*time.Second)
	if err != nil {
		return err
	}
	nbps, perr := ParseInfoBreakpoints(r.Lines)
	if perr != nil {
		s.emitEvent(Event{Type: "log", Text: "断点账本未重建：" + perr.Error()})
		return perr
	}
	ledger := make(map[int]*Breakpoint, len(nbps))
	for i := range nbps {
		ledger[nbps[i].Num] = &nbps[i]
	}
	s.mu.Lock()
	// 旧账本只补空缺：解析到的 Cond / Ignore 优先（Q2）。指针指进 nbps，所以
	// 先建好 ledger 再合并，补上的值对 ledger 同样可见。
	inheritMissingBPFields(nbps, s.bps)
	s.bps = ledger
	s.mu.Unlock()
	return nil
}

// BreakpointsNow 交出**当下**的断点账本（A36，2026-10-18 真机冻结口径）。
//
// 停站态：**先** `syncBreakpoints()` 再返回 —— 用内核此刻的 `breakpoint already hit N time(s)`
// / `ignore next N hits` 覆盖账本。修前的现场是：`info breakpoints` 原文早有
// `already hit 1 time`、点击后确实停站，可 `bp list` 前后都写 `命中 0 次` —— 因为账本只在
// "改断点"那刻回灌，`命中 N 次` 表达的其实是"上次改断点时的计数"。
//
// 非停站态（running / loading / idle / exit）：**一个字节都不发**。运行期 fgldb 对任何调试
// 命令都回 400「程序运行中,禁止发送调试命令(仅允许中断)」（A34），而且允许的自动化只有
// interrupt —— 所以这里退回账本副本，并**如实**给出 fresh=false 与"可能不是最新"的说明，
// 拿一句可照做的话（先 interrupt）代替假装它是最新。
//
// 返回值：断点列表 / 是否已与内核同步 / 说明（fresh 为真时说明为空）。
func (s *Session) BreakpointsNow() (bps []Breakpoint, fresh bool, note string) {
	st := s.State()
	if st != StateStopped {
		return s.Breakpoints(), false, bpListStaleNote(st)
	}
	if err := s.syncBreakpoints(); err != nil {
		// 读不回来就**不假装**：账本保持原样（syncBreakpoints 不拿空表覆盖），
		// 而这份账本此刻确定不是最新 —— 说清原因比给一个漂亮数字有用。
		// 这条路径上会话本来就在停站态，所以不劝人 interrupt，而是给"看原文"的下一步。
		return s.Breakpoints(), false, fmt.Sprintf(
			"账本未能从内核回灌(%v) —— 下面这份**可能不是最新**(命中次数可能落后于内核);"+
				"先用 tt debug exec %q 看内核原文(解析不了会带「行格式不认识」并保留原文),"+
				"再 tt debug bp list", err, "info breakpoints")
	}
	return s.Breakpoints(), true, ""
}

// bpListStaleNote 是"这份账本为什么不是最新 + 下一步做什么"的那句话（A36：运行态必须说）。
func bpListStaleNote(st State) string {
	if st == StateRunning {
		return "程序运行中,读不到内核断点表 —— 下面这份是账本,**可能不是最新**(命中次数可能落后于内核)。" +
			"先 tt debug interrupt 停下,再 tt debug bp list 就是内核当下的计数" +
			"(运行期 fgldb 只受理中断)"
	}
	return fmt.Sprintf("会话当前状态 %s,读不到内核断点表 —— 下面这份是账本,**可能不是最新**"+
		"(命中次数可能落后于内核);等它停下(或先 tt debug interrupt)再 tt debug bp list", st)
}

// Print 求值表达式
func (s *Session) Print(expr string) (string, error) {
	r, err := s.exec("print", "print "+expr, waitPrompt, 30*time.Second)
	if err != nil {
		return "", err
	}
	if r.Err != "" {
		return r.Err, fmt.Errorf("%s", r.Err)
	}
	return strings.TrimSpace(r.Value), nil
}

// Where 调用栈
func (s *Session) Where() ([]Frame, error) {
	r, err := s.exec("where", "where", waitPrompt, 20*time.Second)
	if err != nil {
		return nil, err
	}
	if r.Err != "" {
		return nil, fmt.Errorf("%s", r.Err)
	}
	if len(r.Frames) == 0 && s.cur.Frames != nil {
		return s.cur.Frames, nil
	}
	return r.Frames, nil
}

// ---------- 上下文查询(fgldeb 同款命令面,格式来自 3.21.03 实测) ----------

// Locals 当前帧全部局部变量(info locals,`名字 = 值`,值可续行)。
// 入口停站/无局部变量时返回空列表。随 Frame(n) 切换上下文。
func (s *Session) Locals() ([]VarItem, error) {
	r, err := s.exec("other", "info locals", waitPrompt, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if r.Err != "" {
		return nil, fmt.Errorf("%s", r.Err)
	}
	var out []VarItem
	for _, ln := range r.Lines {
		if host.IsBarePrompt(ln) {
			continue
		}
		if m := reLocalVar.FindStringSubmatch(ln); m != nil {
			out = append(out, VarItem{Expr: m[1], Value: strings.TrimSpace(m[2])})
			continue
		}
		// 值跨行的续行(非 name=value 且非提示符)聚合到上一项
		if len(out) > 0 && strings.TrimSpace(ln) != "" {
			out[len(out)-1].Value += "\n" + ln
		}
	}
	return out, nil
}

// Globals 全部全局变量声明(info variables,实测格式:`Globals:` 头 + `名字 类型` 行)。
// T100 全局量可达数千行,limit<=0 时默认 4000;返回 (截取结果, 总数)。
func (s *Session) Globals(limit int) ([]VarDecl, int, error) {
	if limit <= 0 {
		limit = 4000
	}
	r, err := s.exec("other", "info variables", waitPrompt, 60*time.Second)
	if err != nil {
		return nil, 0, err
	}
	if r.Err != "" {
		return nil, 0, fmt.Errorf("%s", r.Err)
	}
	var out []VarDecl
	for _, ln := range r.Lines {
		if strings.TrimSpace(ln) == "" || host.IsBarePrompt(ln) || ln == "info variables" {
			continue // 空行/提示符/PTY 命令回显行
		}
		if m := reGlobalDecl.FindStringSubmatch(ln); m != nil {
			out = append(out, VarDecl{Name: m[1], Type: strings.TrimSpace(m[2])})
		}
		// 非 `名字 类型` 的行(如分节头)跳过
	}
	total := len(out)
	if total > limit {
		out = out[:limit]
	}
	return out, total, nil
}

// Sources 作业已加载符号的全部模块名(info sources,逗号分隔可换行)
func (s *Session) Sources() ([]string, error) {
	r, err := s.exec("other", "info sources", waitPrompt, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if r.Err != "" {
		return nil, fmt.Errorf("%s", r.Err)
	}
	var out []string
	seen := map[string]bool{}
	for _, ln := range r.Lines {
		ln = strings.TrimSpace(ln)
		if ln == "" || host.IsBarePrompt(ln) {
			continue
		}
		if strings.HasSuffix(ln, ":") && strings.Contains(ln, "Source files") {
			continue // 分节头
		}
		for _, name := range strings.Split(ln, ",") {
			name = strings.TrimSpace(name)
			if name == "" || !strings.HasSuffix(name, ".4gl") || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	return out, nil
}

// Functions 全部函数名(info functions,每行 `name ()`;T100 大作业可达上万行)。
// limit<=0 时默认 5000;返回 (截取结果, 总数)。
func (s *Session) Functions(limit int) ([]string, int, error) {
	if limit <= 0 {
		limit = 5000
	}
	r, err := s.exec("other", "info functions", waitPrompt, 90*time.Second)
	if err != nil {
		return nil, 0, err
	}
	if r.Err != "" {
		return nil, 0, fmt.Errorf("%s", r.Err)
	}
	var out []string
	for _, ln := range r.Lines {
		if m := reFuncList.FindStringSubmatch(ln); m != nil {
			out = append(out, m[1])
		}
	}
	total := len(out)
	if total > limit {
		out = out[:limit]
	}
	return out, total, nil
}

// CalibrateOffset 行号校准:fgldb 报的行号来自 .42r 编译产物的行号表,若服务器上
// .4gl 源码与编译产物版本不一致(如文件头部增删行),DVM 行号会与磁盘文件整体偏移。
// 用停站源码块(行号+文本)在磁盘文件中反查真实行号,返回 offset(DVM 行号 = 磁盘行号 + offset)。
// offset > 0 时前端在源码顶部前插 offset 个空行,即可让 Monaco 行号与协议流对齐。
func (s *Session) CalibrateOffset() (int, error) {
	if s.State() != StateStopped {
		return 0, fmt.Errorf("需停站后才能校准(当前未停站)")
	}
	st := s.Cur()
	if st.File == "" {
		return 0, fmt.Errorf("当前停站缺少源文件信息(如人工中断),请步进到具体代码行后再校准")
	}
	if len(st.Source) == 0 {
		return 0, fmt.Errorf("当前停站没有源码上下文(仅紧凑停站),请步进后再校准")
	}
	// 参考行:IsCur 优先;否则取第一个非空文本行
	ref := SourceLine{}
	for _, sl := range st.Source {
		if sl.IsCur && strings.TrimSpace(sl.Text) != "" {
			ref = sl
			break
		}
	}
	if ref.Num == 0 {
		for _, sl := range st.Source {
			if strings.TrimSpace(sl.Text) != "" {
				ref = sl
				break
			}
		}
	}
	if ref.Num == 0 || strings.TrimSpace(ref.Text) == "" {
		return 0, fmt.Errorf("源码上下文没有可匹配的行文本")
	}
	sf, err := s.ResolveSource(st.File, s.Module)
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.ReplaceAll(sf.Content, "\r\n", "\n"), "\n")
	// 验证集:源码块其余行相对 ref 的 DVM 行号差 → 文本,提高匹配唯一性
	verify := map[int]string{}
	for _, sl := range st.Source {
		if sl.Num == ref.Num {
			continue
		}
		if t := strings.TrimSpace(sl.Text); t != "" {
			verify[sl.Num-ref.Num] = t
		}
	}
	target := strings.TrimSpace(ref.Text)
	// 滑动搜索 offset ∈ [-5,5]:磁盘 idx = DVM行号 - offset - 1(0-based)
	for offset := -5; offset <= 5; offset++ {
		idx := ref.Num - offset - 1
		if idx < 0 || idx >= len(lines) || strings.TrimSpace(lines[idx]) != target {
			continue
		}
		ok := true
		for d, t := range verify {
			vi := idx + d
			if vi < 0 || vi >= len(lines) || strings.TrimSpace(lines[vi]) != t {
				ok = false
				break
			}
		}
		if ok {
			return offset, nil
		}
	}
	return 0, fmt.Errorf("未能在源码文件中匹配到停站行(文件与编译产物版本差异过大)")
}

// InfoLine 让 fgldb 解析位置(函数名/模块名/file:line)为 DVM 源文件名与行号,
// 是源码/符号定位的权威兜底(fgldeb 的 get_full_module_name 同款)
func (s *Session) InfoLine(loc string) (string, int, error) {
	r, err := s.exec("other", "info line "+loc, waitPrompt, 20*time.Second)
	if err != nil {
		return "", 0, err
	}
	if r.Err != "" {
		return "", 0, fmt.Errorf("%s", r.Err)
	}
	return parseInfoLine(r.Lines)
}

// SymbolNotDefinedError 表示"调试器认识这条命令,但符号表里没有这个名字" ——
// 它是**结论**(没找到),不是解析失败。调用方据此给 NOT_FOUND,而不是当成输出看不懂。
type SymbolNotDefinedError struct {
	Word string
}

func (e *SymbolNotDefinedError) Error() string {
	return fmt.Sprintf("符号 %q 不在当前调试进程里（名字不分大小写，但要写全：带程序前缀，如 p_zta_menu）", e.Word)
}

// parseInfoLine 解析 `info line` 的响应行:
//   - `Line 860 of "azz_p_zta.4gl" starts at ...` → (文件, 行号)
//   - `Function "zta_menu" not defined.`         → SymbolNotDefinedError
//   - 其它                                       → 解析失败(带原始行,便于排查)
func parseInfoLine(lines []string) (string, int, error) {
	for _, ln := range lines {
		if m := reInfoLine.FindStringSubmatch(ln); m != nil {
			n, _ := strconv.Atoi(m[1])
			return m[2], n, nil
		}
	}
	for _, ln := range lines {
		if m := reFuncNotDefined.FindStringSubmatch(strings.TrimSpace(ln)); m != nil {
			return "", 0, &SymbolNotDefinedError{Word: m[1]}
		}
	}
	return "", 0, fmt.Errorf("无法解析 info line 输出: %s", strings.Join(lines, " | "))
}

// Frame 选择栈帧(影响 print/locals 求值上下文);新停站自动回到栈顶
func (s *Session) Frame(n int) error {
	r, err := s.exec("other", fmt.Sprintf("frame %d", n), waitPrompt, 15*time.Second)
	if err != nil {
		return err
	}
	if r.Err != "" {
		return fmt.Errorf("%s", r.Err)
	}
	s.mu.Lock()
	s.curFrame = n
	s.mu.Unlock()
	return nil
}

// CurFrame 当前选中栈帧
func (s *Session) CurFrame() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.curFrame
}

// Autovars 最近一次停站的自动变量求值结果
func (s *Session) Autovars() []VarItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAutovars
}

// errSoftWait 内部哨兵:软等待到点、程序仍在运行(不是错误,命令没被取消)。
// 用哨兵让 Step 主体的既有 `return nil, err` 原样传播,不必逐个改返回点。
var errSoftWait = errors.New("软等待到点:程序仍在运行")

// Step 步过/步入/步出/直到
func (s *Session) Step(cmd string) (*StopInfo, error) {
	stop, _, err := s.StepSoft(cmd, 0)
	return stop, err
}

// StepSoft 同 Step,softWait > 0 时启用软等待。
// 返回 soft=true 表示到点返回、程序仍在运行(未发 SIGINT)。步进可能撞上交互语句
// 而长时间等用户,所以步进同样需要软等待这条退路。
func (s *Session) StepSoft(cmd string, softWait time.Duration) (*StopInfo, bool, error) {
	stop, err := s.stepWith(cmd, softWait)
	if errors.Is(err, errSoftWait) {
		return nil, true, nil
	}
	return stop, false, err
}

func (s *Session) stepWith(cmd string, softWait time.Duration) (*StopInfo, error) {
	if !s.Started() {
		// 入口停站:fgldb 在 run 之前不支持步进,透明等效为
		// 「tbreak main + run」= 启动并停在 MAIN 首条语句,符合常规调试器的直觉
		return s.stepFromEntry()
	}
	if cmd != "next" && cmd != "step" && cmd != "finish" && cmd != "until" && !strings.HasPrefix(cmd, "until ") {
		return nil, fmt.Errorf("不支持的步进命令: %s", cmd)
	}
	r, err := s.execOpt("step", cmd, waitPrompt, 60*time.Second, execOpts{SoftWait: softWait})
	if err != nil {
		return nil, err
	}
	if r.SoftTimeout {
		return nil, errSoftWait
	}
	if r.Err != "" {
		return nil, fmt.Errorf("%s", r.Err)
	}
	if r.Stop != nil {
		return r.Stop, nil
	}
	if r.SawShell {
		// 步进过程中程序结束回 shell:本轮运行结束,会话保留(免重新登录)
		s.markShellReady()
		s.enterIdle("程序已退出,会话保留")
		return nil, nil
	}
	// 解析步进结果:优先带源码上下文的块(-> 行);
	// 源码不可用时 fgldb 只输出紧凑形式「89	 in lib_cl_ap.4gl」
	// 跨文件步入:fgldb 会输出停站头「func() at <file>:<line>」,
	// 文件名必须取自停站头,否则永远沿用上一次的文件(只跳行号不切文件)
	var src []SourceLine
	curLine := 0
	curFile := ""
	for _, ln := range r.Lines {
		if m := reStopHeader.FindStringSubmatch(ln); m != nil {
			curFile = m[2] // 步入公共函数时给出新文件
		}
		if m := reSource.FindStringSubmatch(ln); m != nil {
			num, _ := strconv.Atoi(m[2])
			src = append(src, SourceLine{Num: num, Text: m[3], IsCur: m[1] != ""})
			if m[1] != "" {
				curLine = num
			}
		}
	}
	var stop *StopInfo
	if curLine > 0 {
		file := curFile
		if file == "" {
			file = s.Cur().File // 无停站头(同文件步进)沿用当前文件
		}
		stop = &StopInfo{Reason: "step", File: file, Line: curLine, Source: src}
	} else {
		for _, ln := range r.Lines {
			if m := reCompactStep.FindStringSubmatch(ln); m != nil {
				n, _ := strconv.Atoi(m[1])
				stop = &StopInfo{Reason: "step", File: m[2], Line: n}
				break
			}
		}
	}
	if stop != nil {
		s.setStop(stop)
		s.autovarsSettle(stop) // 步进同步完成路径不走 onStop,这里补处理自动变量
		return stop, nil
	}
	return nil, nil
}

// WaitForStop 等待程序停站(断点命中/人工中断),供 AI 与 probe 在 run 之后使用
func (s *Session) WaitForStop(timeout time.Duration) (*StopInfo, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		switch s.State() {
		case StateStopped:
			st := s.Cur()
			return &st, nil
		case StateExit, StateIdle:
			return nil, fmt.Errorf("程序已退出")
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("等待停站超时(%s)", timeout)
}

// stepFromEntry 入口停站的步进:临时断点在 main 首条语句 + run,
// 等效"从入口往下走一步";之后 started=true,后续步进为真实单步
// (注意:入口 pre-run 态下 continue 与 next 一样非法,必须用 run)
func (s *Session) stepFromEntry() (*StopInfo, error) {
	if _, err := s.exec("other", "tbreak main", waitPrompt, 15*time.Second); err != nil {
		return nil, fmt.Errorf("入口步进失败(tbreak main): %w", err)
	}
	s.markStarted()
	r, err := s.exec("run", "run", waitQuiet, 60*time.Second)
	if err != nil {
		return nil, err
	}
	var stop *StopInfo
	if r.Stop != nil {
		// run 在静默判定前就命中了临时断点:同步完成路径,停站事件已由 onStop 发出
		stop = r.Stop
	} else {
		stop, err = s.WaitForStop(60 * time.Second)
		if err != nil {
			return nil, err
		}
	}
	s.emitEvent(Event{Type: "log", Text: "入口步进:已停在 MAIN 首条语句,后续步进为真实单步"})
	s.startWatchdog()
	return stop, nil
}

// Raw 透传任意调试命令(供 AI 高级用法),返回原始行
func (s *Session) Raw(cmd string, timeout time.Duration) ([]string, error) {
	r, err := s.RawSoft(cmd, timeout, 0)
	if err != nil {
		return nil, err
	}
	return r.Lines, nil
}

// RawSoft 同 Raw,softWait > 0 时启用软等待:到点若程序仍在跑就返回
// (execResult.SoftTimeout 为真),**不发 SIGINT、不取消命令**。
// 这是 AI 侧 `exec "continue" --wait N` 的落点 —— 替代原先"超时就偷偷发 \x03"
// 的推荐用法(那个副作用会打断 GDC 上用户的输入)。
func (s *Session) RawSoft(cmd string, timeout, softWait time.Duration) (*execResult, error) {
	return s.execOpt("other", cmd, waitPrompt, timeout, execOpts{SoftWait: softWait})
}

// ---------- 输出泵 ----------

func (s *Session) pump() {
	defer close(s.done)
	pr := &host.LineParser{}
	buf := make([]byte, 8192)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			for _, ln := range pr.Feed(buf[:n]) {
				s.onLine(ln)
			}
			// 提示符后面没有换行:半行若已是提示符形态,立即出行
			// (PS1 与 (fgldb) 都是"无换行"输出,否则永远等不到)
			if partial := pr.PartialStr(); partial != "" {
				if host.IsBarePrompt(partial) || host.ReShellPrompt.MatchString(partial) {
					s.onLine(pr.FlushPartial())
				}
			}
		}
		if err != nil {
			pr.Rest()
			s.handleEOF()
			return
		}
	}
}

func (s *Session) handleEOF() {
	s.mu.Lock()
	st := s.state
	already := st == StateExit
	s.mu.Unlock()
	if !already {
		s.emitEvent(Event{Type: "dead", Text: "调试后端连接断开(终端流结束),会话已终止,可重新启动"})
		s.forceExit()
	}
}

// pushLine 把行推给启动序列等待者(非阻塞,防泵卡死)
func (s *Session) pushLine(ln string) {
	select {
	case s.lines <- ln:
	default:
	}
}

func (s *Session) onLine(ln string) {
	if ln == "" {
		return
	}
	if rawDebug {
		log.Println("RAW " + strconv.Quote(ln))
	}
	s.pushLine(ln)
	s.emitEvent(Event{Type: "output", Text: ln})

	s.mu.Lock()
	pending := s.pending
	collect := s.collect
	state := s.state
	quitReq := s.quitReq
	s.lastOutputAt = time.Now()
	s.mu.Unlock()

	// ---- 停站块开启 ----
	if collect == nil {
		if rawDebug {
			log.Println("DECIDE state=" + string(state) + " pending=" + strconv.FormatBool(pending != nil) + " line=" + strconv.Quote(ln))
		}
		// SIGINT 的 ^C 回显粘在后续输出上,且该 DVM 中断时不输出 INTERRUPT 字样,
		// 只输出源码块——^C 前缀本身就是中断标记;剥离后本行仍需按内容收集
		sigint := false
		if state == StateRunning && strings.HasPrefix(ln, "^C") {
			ln = strings.TrimPrefix(ln, "^C")
			collect = &stopCollect{reason: "interrupt"}
			sigint = true
		}
		if !sigint {
			if m := reBreakHit.FindStringSubmatch(ln); m != nil {
				num, _ := strconv.Atoi(m[1])
				line, _ := strconv.Atoi(m[4])
				collect = &stopCollect{reason: "breakpoint", bpnum: num, fn: m[2], file: m[3], line: line}
			} else if m := reSource.FindStringSubmatch(ln); m != nil && m[1] != "" &&
				pending != nil && IsResumeCmd(pending.cmd) {
				// 带箭头的源码行 = 当前停站行。放行类命令(step/next/continue…)停下时,
				// fgldb 可能**只打源码窗、不打任何位置头** —— 以前这个分支什么都不做,
				// 于是 collect 从不创建、onStop 从不被调用:停站事件不发、s.cur 不更新,
				// 界面上的代码画面也就完全不会跟随 AI 的步进。必须把这种也当成一次停站。
				//
				// 两个限定条件都必要:只认**带箭头**的行(list 之类输出没有箭头),
				// 且只认**放行类命令在飞**时(排除别的命令恰好带回一行源码)。
				collect = &stopCollect{reason: "step"}
			} else if reSource.MatchString(ln) {
				// 其余源码块行(如 `-> 1532 DEFER INTERRUPT`):行内容可能含 INTERRUPT
				// 这类关键字,必须在这里吃掉 —— 否则会落到下面被误判成人工中断标记。
				// 这是原有行为,别删。
			} else if reInterrupt.MatchString(ln) {
				collect = &stopCollect{reason: "interrupt"}
			} else if state == StateRunning && !reFrame.MatchString(ln) {
				if m := reStopHeader.FindStringSubmatch(ln); m != nil {
					line, _ := strconv.Atoi(m[3])
					collect = &stopCollect{reason: "step", fn: m[1], file: m[2], line: line}
				}
			}
		}
		if collect != nil {
			s.mu.Lock()
			s.collect = collect
			s.mu.Unlock()
		}
	}

	// ---- 停站块收口:裸提示符 ----
	if collect != nil {
		if host.IsBarePrompt(ln) {
			// 没有位置头的停站(中断块、以及 step/next 只打源码窗的那一种):
			// 从箭头行补行号(文件名协议未给,由 where/前端兜底)
			if collect.file == "" && collect.line == 0 {
				for _, sl := range collect.source {
					if sl.IsCur {
						collect.line = sl.Num
						break
					}
				}
			}
			// 无位置头的 step/next 只打源码窗,协议不给文件名。这种情况必然是**同文件**
			// 移动(跨文件步进 fgldb 会打位置头),所以沿用停站前的位置 —— 与 stepWith
			// 的兜底一致。不补的话:前端要额外发一次 where,停站文件的本地副本也拿不到文件名。
			// 中断块同属"无位置头",但中断可能停在另一个文件上,不能瞎猜,仍交给 where。
			if collect.file == "" && collect.line > 0 && collect.reason == "step" {
				collect.file = s.Cur().File
			}
			stop := &StopInfo{
				Reason: collect.reason, BPNum: collect.bpnum,
				Func: collect.fn, File: collect.file, Line: collect.line,
				Frames: collect.frames, Source: collect.source,
			}
			// 就地判定「当前停站行是不是交互语句」:停站源码块本来就带着当前行原文,
			// 不需要读远端源码、不需要发任何命令 —— 零副作用。
			if kind := ClassifyInteractiveLine(CurSourceText(collect.source)); kind != "" {
				stop.WaitingForUser = true
				stop.WaitingKind = kind
			}
			s.mu.Lock()
			s.collect = nil
			s.cur = *stop
			pending := s.pending
			s.mu.Unlock()
			s.onStop(stop, pending, quitReq)
			return
		}
		// 内容收集
		if m := reFrame.FindStringSubmatch(ln); m != nil {
			idx, _ := strconv.Atoi(m[1])
			line, _ := strconv.Atoi(m[4])
			collect.frames = append(collect.frames, Frame{Idx: idx, Func: m[2], File: m[3], Line: line})
		} else if m := reSource.FindStringSubmatch(ln); m != nil {
			num, _ := strconv.Atoi(m[2])
			collect.source = append(collect.source, SourceLine{Num: num, Text: m[3], IsCur: m[1] != ""})
		}
		return
	}

	// ---- pending 响应处理(非停站块) ----
	if pending != nil {
		pending.addLine(ln)

		// "Continuing." = 程序已放行。这与哪条命令通道无关,状态必须如实翻 running ——
		// 原先只有 waitMarker(Continue())路径翻,于是 Raw(waitPrompt)发出的
		// `exec "continue"` 之后 state 仍是 stopped:Interrupt() 会被自己的
		// state==Running 检查拒掉,SKILL.md 推荐的「拿不准就 interrupt 探测」直接失效。
		// 这里只修状态、不 complete、不 return,各 mode 的完成判定继续走各自分支
		// (waitQuiet 的 run 仍要在 1908 附近按首个输出收口)。
		if reContinuing.MatchString(ln) {
			s.setState(StateRunning)
		}

		// 程序退出(作业窗口被关闭或正常结束):立即结束本轮命令,回空闲保留宿主,
		// 避免 waitMarker/waitPrompt 等不到完成信号而超时,也避免被裸提示符误判成停站
		if reProgramExited.MatchString(ln) {
			s.completePending(&execResult{Cmd: pending.cmd, Lines: pending.lines, Err: "program exited"})
			s.enterIdle("程序已退出(作业窗口被关闭或正常结束),会话保留" + s.screenExitTail())
			return
		}

		if e := matchFdbErr(ln); e != "" && pending.errText == "" {
			pending.errText = e
			// 兜底回滚:放行类命令在 pre-run 态会被 fgldb 拒("The program is not being run")。
			// execOpt 发出命令时把状态**乐观**翻成 running 是对的(命令确实发出去了),
			// 但被拒之后那条"乐观"就是假的:状态会永远停在 running,于是 interrupt 因
			// "未启动"被拒、exec 因"运行中"被拒、EndRun 的中断同样失败 —— 只能整体断开会话。
			// 真机踩过。这里把状态退回 stopped,并把拒绝如实留在 pending.errText 里。
			if reNotRunning.MatchString(e) && IsResumeCmd(pending.cmd) && !s.started {
				s.setState(StateStopped)
			}
		}

		switch pending.kind {
		case "where":
			if m := reFrame.FindStringSubmatch(ln); m != nil {
				idx, _ := strconv.Atoi(m[1])
				line, _ := strconv.Atoi(m[4])
				pending.frames = append(pending.frames, Frame{Idx: idx, Func: m[2], File: m[3], Line: line})
			}
		case "print":
			if m := rePrintVal.FindStringSubmatch(ln); m != nil {
				pending.valueOn = true
				pending.value.Reset()
				pending.writeValue(m[2])
			} else if pending.valueOn && !host.IsBarePrompt(ln) {
				pending.writeValue("\n" + ln)
			}
		case "break":
			if m := reBPSet.FindStringSubmatch(ln); m != nil {
				num, _ := strconv.Atoi(m[1])
				line, _ := strconv.Atoi(m[3])
				s.mu.Lock()
				s.bps[num] = &Breakpoint{Num: num, File: m[2], Line: line, Enabled: true}
				s.mu.Unlock()
			}
		case "breakpoints":
			// A27：一条断点可能占两行（第二行是缩进的 `ignore next N hits`），
			// 所以这里只收原始行，解析统一交给 ParseInfoBreakpoints（见 settleBPs）。
			pending.bpRaw = append(pending.bpRaw, ln)
		}

		// 完成条件
		if host.ReShellPrompt.MatchString(ln) {
			// fgldb 已退出回到 shell(quit 后)
			s.completePending(&execResult{Cmd: pending.cmd, SawShell: true, Err: pending.errText})
			return
		}
		if host.IsBarePrompt(ln) && pending.mode == waitPrompt {
			s.completePending(&execResult{
				Cmd: pending.cmd, Lines: pending.lines,
				Frames: pending.frames, Value: pending.value.String(), BPs: pending.settleBPs(),
				Err: pending.errText,
			})
			return
		}
		if reContinuing.MatchString(ln) && pending.mode == waitMarker {
			r := &execResult{Cmd: pending.cmd, Lines: pending.lines, Continuing: true}
			s.completePending(r)
			s.setState(StateRunning)
			return
		}
		// waitQuiet:出现任意实质输出即认为已接受(异步停站由泵兜底)
		if pending.mode == waitQuiet && len(pending.lines) > 0 && !host.IsBarePrompt(ln) && !strings.HasPrefix(ln, "(fgldb)") {
			s.completePending(&execResult{Cmd: pending.cmd, Lines: pending.lines})
			s.setState(StateRunning)
			return
		}
		if host.IsBarePrompt(ln) && pending.mode == waitQuiet {
			// run 失败/立即停站:回到 stopped
			s.completePending(&execResult{Cmd: pending.cmd, Lines: pending.lines})
			s.setState(StateStopped)
			s.startWatchdog()
			return
		}
		return
	}

	// ---- 无 pending 时的异步状态迁移 ----
	// idle 收尾:本轮已结束,等待宿主回 shell。残留的 (fgldb)(自然退出后
	// fglrun 未随程序退出)不在 idle 期补 quit——统一交给下次启动的
	// ensureShellForRun 清理,避免与启动流程重复发 quit
	if state == StateIdle {
		if host.ReShellPrompt.MatchString(ln) {
			s.markShellReady()
		}
		return // idle 期其它输出(退出回显等)直接忽略
	}
	if reProgramExited.MatchString(ln) {
		// 程序退出(作业窗口被关闭或正常结束):本轮运行结束,回空闲保留宿主;
		// 不能当停站处理,否则后续裸提示符会被下面的"保守置 stopped"误判
		s.enterIdle("程序已退出(作业窗口被关闭或正常结束),会话保留" + s.screenExitTail())
		return
	}
	if reContinuing.MatchString(ln) && state == StateStopped {
		s.setState(StateRunning)
		return
	}
	if host.IsBarePrompt(ln) && state == StateRunning {
		// 程序停下但没有可识别的停站头(罕见)——保守置为 stopped
		s.setState(StateStopped)
		s.startWatchdog()
		return
	}
	if host.ReShellPrompt.MatchString(ln) && state == StateRunning {
		// 程序运行中看到 shell 提示符 = 程序已退出回到 shell
		s.markShellReady()
		s.enterIdle("程序已退出,会话保留")
	}
}

// onStop 停站收口:同步(命令响应)或异步(断点/中断命中)
func (s *Session) onStop(stop *StopInfo, pending *pendingCmd, quitReq bool) {
	// 停站文件落本地副本(异步、尽力而为)。放这里是因为它是断点命中/中断的必经收口;
	// 入口停站与同文件步进不走 onStop,由 setStop 那条覆盖。
	s.mirrorStopFile(stop)
	// 调用方传来的是 onLine 顶部的快照,可能已被软等待(abandonPending)或硬超时
	// (clearPending)作废。**必须**以锁内实况复核:sync 分支只回填响应、不置 stopped、
	// 也不发 stopped 事件(直接 return),一旦让已无人等待的停站误走 sync,
	// 会话就会永久卡在 running 且停站事件永久丢失。
	s.mu.Lock()
	if pending != nil && s.pending != pending {
		pending = s.pending // 快照已过期:以实况为准
	}
	if pending != nil && pending.softAbandoned {
		s.pending = nil
		pending = nil
	}
	s.mu.Unlock()

	if pending != nil {
		// 停站发生在命令执行中(step/continue 类)。先把状态与事件收口,再回填响应 ——
		// 这样调用方拿到响应时看到的快照已经是新位置。
		//
		// 状态在这里必须显式置回 stopped:放行类命令发出时会把状态翻成 running
		// (见 execOpt),而 sync 分支以前假定"状态本来就是 stopped"。
		//
		// stopped 事件同样必须发。AI 发起的 next/continue 走的正是这条路径,
		// 以前只有异步路径发事件,于是浏览器端收不到任何通知、代码画面不会跟随
		// (人类自己的步进靠 HTTP 响应里的 stop 落位,所以这个缺口一直没暴露)。
		s.setState(StateStopped)
		s.emitEvent(Event{Type: "stopped", Stop: stop})
		s.completePending(&execResult{
			Cmd: pending.cmd, Lines: pending.lines, Stop: stop,
			Frames: pending.frames, Value: pending.value.String(), BPs: pending.settleBPs(),
			Err: pending.errText,
		})
		s.startWatchdog()
		if !quitReq {
			s.autovarsSettle(stop)
		}
		return
	}
	// 异步命中:断点/SIGINT
	if quitReq {
		return // 正在退出,不用理会
	}
	s.setState(StateStopped)
	s.emitEvent(Event{Type: "stopped", Stop: stop})
	s.startWatchdog()
	s.autovarsSettle(stop)
}

// ---------- 状态与看门狗 ----------

func (s *Session) setState(st State) {
	s.mu.Lock()
	old := s.state
	s.state = st
	if st == StateStopped {
		s.startAt = time.Now()
	}
	if st == StateRunning {
		s.runningSince = time.Now()
	}
	s.mu.Unlock()
	if st == StateRunning {
		s.stopWatchdog() // 离开停站态,取消看门狗计时
	}
	if old != st {
		s.emitEvent(Event{Type: "state", State: string(st)})
	}
}

func (s *Session) setStop(stop *StopInfo) {
	s.mu.Lock()
	s.cur = *stop
	s.curFrame = 0 // 新停站回到栈顶帧
	s.mu.Unlock()
	s.mirrorStopFile(stop)
}

// resetMirrored 清掉"已镜像"记录,必须与 clearMirrorForSession 成对调用。
//
// 复用的宿主会话(idle 复用免重登录)会把这份记录带进下一轮运行,
// 而镜像目录在启动时已经清空了 —— 不同步清掉,新一轮就再也不会
// 落任何副本(mirrorStopFile 以为"这个文件早就镜像过了")。
func (s *Session) resetMirrored() {
	s.mu.Lock()
	s.mirrored = map[string]bool{}
	s.mu.Unlock()
}

// mirrorScope 本条会话在本地镜像/命令输出目录里的目标作用域（缺陷 X / T5.1）:
// 环境段（测试区/正式区隔离,与从前一致）+ 运行作用域（会话之间隔离）。
//
// 两条纪律:
//   - **同一个作用域**同时决定"文件落到哪"（mirrorWrite / writeExecLog）
//     与"新一轮清哪棵子树"（clearMirrorForSession）——两件事分家就会出现
//     "清了 A 的目录、标记说 B 的文件已经镜像过"这种找不到副本的状态;
//   - runScope 为空（手工造的会话）时退化成只有环境段,与从前的路径逐字相同。
func (s *Session) mirrorScope() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	scope := s.runScope
	envSeg := s.mirrorEnvSegNow()
	s.mu.Unlock()
	return mirrorScopeOf(envSeg, scope)
}

// mirrorEnvSegNow 环境段（调用方持有 s.mu）。
func (s *Session) mirrorEnvSegNow() string {
	env := s.envName
	host, zone := "", ""
	if s.cfg != nil {
		if env == "" {
			env = s.cfg.EnvName()
		}
		host, zone = s.cfg.SSH.Host, s.cfg.Zone
	}
	return mirrorEnvSeg(env, host, zone)
}

// nextSessionID 会话的可见 ID（既有形状 `s<毫秒>`）+ **进程内唯一**保证。
//
// 从前是裸的 `time.Now().UnixMilli()`:同一个毫秒里起的第二条会话会拿到同一个 ID,
// 于是 `m.sessions[id]` 把第一条**顶掉**（并发放开之后这就是"另一条会话莫名其妙没了"）。
//
// 这里把它做成**单调计数器**:候选值 = 当前毫秒,但必须严格大于上一次发出去的值。
// 于是
//   - 形状不变（H2:仍是 `s<数字>`,数量级仍是毫秒时间戳,老脚本与用户照抄的 ID 不作废）;
//   - 同一毫秒里连起两条 → 第二条拿到"前一毫秒"的下一个数,**不撞**;
//   - 时钟回拨也不会撞（计数器只增不减）。
func nextSessionID() string {
	for {
		last := sessionSeq.Load()
		next := time.Now().UnixMilli()
		if next <= last {
			next = last + 1
		}
		if sessionSeq.CompareAndSwap(last, next) {
			return "s" + strconv.FormatInt(next, 10)
		}
	}
}

var (
	// sessionSeq 上一次发出去的会话 ID 数值（见 nextSessionID）。
	sessionSeq atomic.Int64
	// runScopeSeq 运行作用域的单调序号（见 nextRunScope）。
	runScopeSeq atomic.Int64
)

// nextSessionEpoch 发一个新的**会话世代号**（契约 C5.3 的 `epoch`，F15 的判据）。
//
// # 语义（本批定稿）
//
// 世代号 = "这条会话**这一代宿主 / 这一轮运行**"的令牌，取值是**进程内单调且全进程唯一**
// 的 base36 计数。**唯一**这一条是刻意的：令牌不重复 ⇒ 复用/重启之后**不可能**凑巧等于
// 调用方手上的旧值，于是"旧句柄当场失效"（SESSION_STALE）不需要任何时钟比较或额外状态。
//
// 换代的三个时机（都在本包，改一处就够）：
//  1. **会话建立**：Manager 登记时发第一代（`assignIdentityLocked`）；
//  2. **同一宿主跑新一轮**：`SetRun`（复用空闲宿主 / 按日志重放复用宿主）→ 换代；
//  3. **会话重启/重新登录**：`/api/sessions/{id}/restart` 是"收掉旧的 + 新的登上来"，
//     新会话在登记时自然拿到新的一代（旧会话已从池里移除）。
//
// **与 `Manager.Epoch` 不是一回事，别混**：`manager.go` 的 `Manager.Epoch` 是**事件流游标**
// 的进程级世代（前端靠它知道"服务重启了、`Seq` 要归零"，见 `Event.Seq` 的注释）；
// 本条是**会话**的世代。两者同名前缀只是巧合，一个在 `Manager` 上、一个在 `Session` 上。
func nextSessionEpoch() string {
	// 发号规则住在 `internal/session`：这里只是转发。
	return sessionid.NewEpoch()
}

// nextRunScope 分配一个进程内唯一的运行作用域（镜像/命令输出的目录段,见 mirrorScope）。
// 只做路径段,不面向用户:*不*放进任何输出去改既有形状。
func nextRunScope() string {
	return "run" + strconv.FormatInt(runScopeSeq.Add(1), 10)
}

// inheritMirrorState 复用的宿主会话进入新一轮时,把上一轮的镜像作用域与"已镜像"记录
// 一起接过来（缺陷 X / T5.1）。
//
// 为什么两个都要接:
//   - 作用域:复用的宿主会话要清的是**自己上一轮**那棵子树（clearMirrorForSession 按它算）,
//     另起一个新作用域就等于把上一轮的副本留在盘上（"只活一轮"这条语义会丢）;
//   - 记录:新一轮会重新落副本（Manager 随后调 resetMirrored）,但在此之前
//     `mirrorStopFile` 可能已经在别的路径上判过"这个文件镜像过了"。
//
// 新建的会话（NewSession）本来就有自己的作用域,这里什么都不做。
func (s *Session) inheritMirrorState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.runScope == "" {
		s.runScope = nextRunScope()
	}
	if s.mirrored == nil {
		s.mirrored = map[string]bool{}
	}
}

// mirrorStopFile 把停站文件的源码落进本地镜像。
//
// 这是"缓存调试过程中经历过的文件"那一半:只靠 tt debug source 的话,镜像里只有
// 显式读过的文件;程序停过的文件才是完整的一份经历记录。
//
// 异步且尽力而为:绝不能给停站加一次 SFTP 往返(连续 next 时那是逐步的延迟),
// 失败也只是少一份副本 —— 要看的文件随时可以 tt debug source 显式读。
func (s *Session) mirrorStopFile(stop *StopInfo) {
	if stop == nil || stop.File == "" || s.cfg == nil || s.cfg.DataDir == "" {
		return
	}
	s.mu.Lock()
	if s.mirrored == nil {
		s.mirrored = map[string]bool{}
	}
	seen := s.mirrored[stop.File]
	s.mirrored[stop.File] = true
	s.mu.Unlock()
	if seen {
		return
	}
	dvmFile := stop.File
	go func() {
		// ResolveSource 自带 mtime 缓存与镜像落盘
		if _, err := s.ResolveSource(dvmFile, s.Module); err != nil {
			log.Printf("[srccache] 停站文件镜像失败 %s: %v", dvmFile, err)
		}
	}()
}

// watchdogRetryDelay 看门狗自动 continue 失败后，隔多久再试一次。
//
// 为什么必须重试：定时器到点那一刻正是“连接可能半死”的时刻（心跳刚好卡住、SSH 通道
// 正在超时）。从前 `go s.Continue()` 失败就完了：不重试、不告警，timer 也已烧掉，
// 于是会话继续停在断点上持锁，**从此再没有任何东西会提醒**。
const watchdogRetryDelay = 60 * time.Second

// watchdogApproachDelay 返回“快要自动放行”的提醒时刻：总时长的 80%。
//
// 0.8 而不是固定 60 秒：超时被人设成 60 秒时，固定提前量会把整段停站都变成预警噪音。
func watchdogApproachDelay(total time.Duration) time.Duration {
	if total <= 0 {
		return 0
	}
	return total - total/5 // 80% 处
}

func (s *Session) startWatchdog() {
	if s.cfg.WatchdogSeconds <= 0 {
		return
	}
	s.stopWatchdog()
	cur := s.Cur()
	if cur.Reason == "entry" {
		return // 入口停留不持事务锁,不自动 continue
	}
	total := time.Duration(s.cfg.WatchdogSeconds) * time.Second
	if lead := total - watchdogApproachDelay(total); lead > 0 {
		s.approachTimer = time.AfterFunc(watchdogApproachDelay(total), func() {
			if s.State() != StateStopped {
				return
			}
			// 有人正对着屏幕读这段代码时，忽然被放行很恼火。提前说一声，让他
			// 决定是继续看（那就手动 continue）还是就这么放着。
			s.emitEvent(Event{Type: "watchdog-approaching",
				Text: fmt.Sprintf("停站已接近 %d 秒上限，%d 秒后看门狗会自动继续（要保住现场请手动操作）",
					s.cfg.WatchdogSeconds, int(lead.Round(time.Second)/time.Second))})
		})
	}
	s.stopTimer = time.AfterFunc(total, func() { s.fireWatchdog(1) })
}

// fireWatchdog 尝试自动放行一次（attempt=1 首次，2 是重试）。
func (s *Session) fireWatchdog(attempt int) {
	if s.State() != StateStopped {
		return
	}
	if attempt == 1 {
		s.emitEvent(Event{Type: "watchdog",
			Text: fmt.Sprintf("停站停留超过 %d 秒,看门狗自动继续运行(防生产库行锁)", s.cfg.WatchdogSeconds)})
	}
	if _, err := s.Continue(); err == nil {
		return
	} else if attempt >= 2 {
		// 两次都不成：把话说到能照着做为止 —— 这时候锁还在，人必须亲自处理。
		s.emitEvent(Event{Type: "watchdog-failed",
			Text: fmt.Sprintf("看门狗自动继续连续两次失败(%v)：程序仍停在断点并持锁。"+
				"请人工处理：tt debug quit（立刻放锁）或 tt session rm（断开连接）", err)})
		return
	} else {
		s.emitEvent(Event{Type: "watchdog-failed",
			Text: fmt.Sprintf("看门狗自动继续失败(%v)，%d 秒后重试一次", err, int(watchdogRetryDelay.Seconds()))})
		s.retryTimer = time.AfterFunc(watchdogRetryDelay, func() { s.fireWatchdog(2) })
	}
}

// StopTimer 停掉看门狗计时(继续运行时)
func (s *Session) stopWatchdog() {
	for _, t := range []*time.Timer{s.stopTimer, s.approachTimer, s.retryTimer} {
		if t != nil {
			t.Stop()
		}
	}
}

// ---------- 等待辅助(启动序列用) ----------

func (s *Session) waitRegexp(re *regexp.Regexp, timeout time.Duration, what string) error {
	deadline := time.After(timeout)
	have := false
	for !have {
		select {
		case ln := <-s.lines:
			if re.MatchString(ln) {
				have = true
			}
		case <-deadline:
			return fmt.Errorf("等待 %s 超时", what)
		}
	}
	return nil
}

func (s *Session) waitBarePrompt(timeout time.Duration, what string) error {
	deadline := time.After(timeout)
	for {
		select {
		case ln := <-s.lines:
			if host.IsBarePrompt(ln) {
				return nil
			}
		case <-deadline:
			return fmt.Errorf("等待 %s 超时", what)
		}
	}
}

// ---------- 源码读取(SFTP) ----------

// sftpClient 懒加载 SFTP 通道
func (s *Session) sftpClient() (*sftp.Client, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sftpClientCache == nil {
		c, err := s.conn.SFTP()
		if err != nil {
			return nil, fmt.Errorf("打开 SFTP 失败: %w", err)
		}
		s.sftpClientCache = c
	}
	return s.sftpClientCache, nil
}

// ReadFile 读取服务器文件
func (s *Session) ReadFile(path string) ([]byte, time.Time, error) {
	cl, err := s.sftpClient()
	if err != nil {
		return nil, time.Time{}, err
	}
	f, err := cl.Open(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	defer f.Close()
	var mtime time.Time
	if fi, err := f.Stat(); err == nil {
		mtime = fi.ModTime()
	}
	data, err := io.ReadAll(f)
	return data, mtime, err
}

// SourceFile 解析后的源码文件
type SourceFile struct {
	DVMFile string    `json:"dvmFile"` // DVM 报告的模块源名,如 asf_bsft001_wf.4gl
	Path    string    `json:"path"`    // 实际读取的服务器路径
	Content string    `json:"content"`
	ModTime time.Time `json:"modTime"`
	// LocalPath 本地镜像路径(<DataDir>/srccache/…),本次调试读过就有一份;
	// 让 AI 能在本地整读/搜索,不必每次 SSH 往返。镜像每轮调试开始时清空。
	LocalPath string `json:"localPath,omitempty"`
}

// mirrorWrite 会话内读路径的镜像落盘(与会话外 ReadSourceStandalone 共用同一目录口径)
//
// 落点用**本会话的作用域**（缺陷 X / T5.1）:从前是环境段一个路径段,
// 同环境下两条会话共用一个目录 —— A 拉过的文件 B 的 `mirrored` 标记里没有,
// 但盘上已经有了,于是"B 永远拉不到 A 拉过的文件"这个状态同时成立。
func (s *Session) mirrorWrite(serverPath string, data []byte) string {
	if s.cfg == nil {
		return ""
	}
	return writeMirrorScoped(s.cfg.DataDir, s.mirrorScope(), serverPath, data)
}

type srcCacheEntry struct {
	modTime time.Time
	file    SourceFile
}

// sourceBaseName 取文件名部分(不依赖 path 包,避免和参数名 path 打架)。
func sourceBaseName(file string) string {
	if i := strings.LastIndexAny(file, "/\\"); i >= 0 {
		return file[i+1:]
	}
	return file
}

// sourceCandidatePaths 生成源码查找候选路径(自由函数,会话与预取共用)。
//
// TOP GP 只有一种布局:<root>/<模块>/4gl/<程序>.4gl,$TOP 与 $CUST 两棵树。
// 模块按 modules 的顺序逐个试(p_link 解析出来的在前、用户填的在后),
// 这样"模块留空"或"模块填错"都能落到同一份实际源码上。
func sourceCandidatePaths(roots, modules []string, dvmFile string) []string {
	name := strings.TrimSuffix(path.Base(dvmFile), ".4gl")
	var out []string
	seen := map[string]bool{}
	for _, mod := range modules {
		mod = strings.ToLower(strings.TrimSpace(mod))
		if mod == "" || !reProgName.MatchString(mod) {
			continue
		}
		for _, root := range roots {
			for _, n := range sourceNamesForModule(mod, name) {
				p := root + "/" + mod + "/4gl/" + n
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
	}
	return out
}

// sourceNamesForModule 一个模块目录里要试的文件名。
//
// GP 的调试器(r.d2)报的是短名 csfi300_icd,磁盘上是 csf_csfi300_icd.4gl;
// 会话里也可能拿到全名,所以三种写法都试。
func sourceNamesForModule(mod, name string) []string {
	var out []string
	add := func(n string) {
		if n == "" {
			return
		}
		for _, s := range out {
			if s == n {
				return
			}
		}
		out = append(out, n)
	}
	add(name + ".4gl")
	if !strings.HasPrefix(name, mod+"_") {
		add(mod + "_" + name + ".4gl")
	}
	if len(name) > 4 && name[3] == '_' {
		short := name[4:]
		add(short + ".4gl")
		if !strings.HasPrefix(short, mod+"_") {
			add(mod + "_" + short + ".4gl")
		}
	}
	return out
}

// sourceModuleCandidates 模块候选:p_link(本地字典) 解析结果优先,其次用户填的,
// 最后按作业名推一个(与启动时 gpModuleOf 同一套规则)。
func sourceModuleCandidates(lookup func(string) (string, bool), module, name string) []string {
	var mods []string
	add := func(m string) {
		m = strings.ToLower(strings.TrimSpace(m))
		if m == "" || !reProgName.MatchString(m) {
			return
		}
		for _, s := range mods {
			if s == m {
				return
			}
		}
		mods = append(mods, m)
	}
	if m, ok := lookup(name); ok {
		add(m)
	}
	add(module)
	add(gpModuleOf(name))
	return mods
}

// sourceCandidates 生成源码查找候选路径
func (s *Session) sourceCandidates(dvmFile, module string) []string {
	name := strings.TrimSuffix(path.Base(dvmFile), ".4gl")
	mods := sourceModuleCandidates(s.pLinkModule, module, name)
	return sourceCandidatePaths(s.cfg.ModuleRootsActual(), mods, dvmFile)
}

// ResolveSource 把 DVM 报告的源文件名解析为真实路径并读取(带 mtime 缓存,避免每次停站全量拉取)
func (s *Session) ResolveSource(dvmFile, module string) (*SourceFile, error) {
	cl, err := s.sftpClient()
	if err != nil {
		return nil, err
	}
	var lastErr error
	for _, p := range s.sourceCandidates(dvmFile, module) {
		fi, err := cl.Stat(p)
		if err != nil {
			lastErr = err
			continue
		}
		mt := fi.ModTime()
		s.mu.Lock()
		e, ok := s.srcCache[p]
		s.mu.Unlock()
		if ok && e.modTime.Equal(mt) {
			sf := e.file
			return &sf, nil
		}
		data, rmt, err := s.ReadFile(p)
		if err != nil {
			lastErr = err
			continue
		}
		sf := SourceFile{DVMFile: dvmFile, Path: p, Content: string(data), ModTime: rmt,
			LocalPath: s.mirrorWrite(p, data)}
		s.mu.Lock()
		if len(s.srcCache) > 24 {
			s.srcCache = map[string]srcCacheEntry{}
		}
		s.srcCache[p] = srcCacheEntry{modTime: mt, file: sf}
		s.mu.Unlock()
		return &sf, nil
	}
	return nil, fmt.Errorf("源码未找到(%s): %w", dvmFile, lastErr)
}

// ReadPath 读取任意路径(限制在登录区源码目录内,防止越权读文件)
func (s *Session) ReadPath(path string) (*SourceFile, error) {
	ok := false
	for _, root := range s.cfg.ModuleRootsActual() {
		if strings.HasPrefix(path, root+"/") {
			ok = true
			break
		}
	}
	if !ok {
		return nil, fmt.Errorf("路径不在登录区源码目录内: %s", path)
	}
	data, mt, err := s.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &SourceFile{Path: path, Content: string(data), ModTime: mt,
		LocalPath: s.mirrorWrite(path, data)}, nil
}

// ---------- 快照 ----------

// State 当前状态
func (s *Session) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 入口停站后的断点恢复期间,对外仍报 loading:
	// 前端见到 stopped 即停轮询,提前宣告会让首帧快照缺断点(要步进一次才可见)
	if s.state == StateStopped && s.restoring {
		return StateLoading
	}
	return s.state
}

// Cur 最近停站现场
func (s *Session) Cur() StopInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Breakpoints 断点列表
func (s *Session) Breakpoints() []Breakpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Breakpoint, 0, len(s.bps))
	for _, b := range s.bps {
		out = append(out, *b)
	}
	// map 遍历顺序随机:按 文件+行号 稳定排序,保证快照/列表刷新(如步进后)顺序不变
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// HoldingSeconds 已停留秒数(仅 stopped 态有意义)
func (s *Session) HoldingSeconds() float64 {
	s.mu.Lock()
	st := s.state
	start := s.startAt
	s.mu.Unlock()
	if st != StateStopped {
		return 0
	}
	return time.Since(start).Seconds()
}

func (s *Session) emitEvent(ev Event) {
	if s.emit == nil {
		return
	}
	ev.SessionID = s.ID
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	s.emit(ev)
}

// ---------- 自动变量(fgldeb Auto 面板等价物) ----------

// extractVarNames 从停站源码窗提取候选变量名:当前行优先,其余按距离;
// 剥注释与字符串字面量,排除 4gl 关键字与函数调用,去重,上限 max
func extractVarNames(source []SourceLine, curLine, max int) []string {
	if curLine <= 0 || len(source) == 0 {
		return nil
	}
	win := make([]SourceLine, len(source))
	copy(win, source)
	sort.SliceStable(win, func(i, j int) bool {
		di, dj := absInt(win[i].Num-curLine), absInt(win[j].Num-curLine)
		if di != dj {
			return di < dj
		}
		return win[i].Num < win[j].Num
	})
	seen := map[string]bool{}
	var out []string
	for _, sl := range win {
		for _, name := range lineVarNames(sl.Text) {
			if !seen[name] {
				seen[name] = true
				out = append(out, name)
				if len(out) >= max {
					return out
				}
			}
		}
	}
	return out
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// reIdent 标识符(含记录链 a.b.c)
var reIdent = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)*`)

// lineVarNames 从一行 4gl 源码提取候选变量名
func lineVarNames(line string) []string {
	if i := strings.Index(line, "--"); i >= 0 {
		line = line[:i]
	}
	if i := strings.Index(line, "#"); i >= 0 {
		line = line[:i]
	}
	line = stripQuoted(line)
	var out []string
	seen := map[string]bool{}
	for _, loc := range reIdent.FindAllStringIndex(line, -1) {
		name := line[loc[0]:loc[1]]
		// 函数调用排除:紧跟 "("
		if strings.HasPrefix(strings.TrimLeft(line[loc[1]:], " \t"), "(") {
			continue
		}
		skip := false
		for _, seg := range strings.Split(name, ".") {
			if glKeywords[strings.ToLower(seg)] {
				skip = true
				break
			}
		}
		if skip || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

// stripQuoted 把 '...' 与 "..." 字面量替换为空格(处理反斜杠转义)
func stripQuoted(s string) string {
	var b strings.Builder
	in := byte(0)
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if in != 0 {
			if ch == '\\' && i+1 < len(s) {
				i++
				b.WriteByte(' ')
				continue
			}
			if ch == in {
				in = 0
			}
			b.WriteByte(' ')
			continue
		}
		if ch == '\'' || ch == '"' {
			in = ch
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// glKeywords 4gl 关键字表(变量名排除用,取自 BDL 语言关键字)
var glKeywords = map[string]bool{}

func init() {
	kws := []string{
		"main", "function", "return", "if", "then", "else", "elif", "for", "to", "step",
		"while", "case", "when", "otherwise", "define", "record", "array", "dynamic", "like",
		"type", "constant", "let", "call", "display", "input", "construct", "by", "name",
		"on", "from", "menu", "command", "continue", "exit", "next", "field", "before",
		"after", "row", "key", "options", "defer", "interrupt", "whenever", "error",
		"warning", "open", "window", "form", "close", "current", "dialog", "attributes",
		"unbuffered", "without", "defaults", "accept", "cancel", "insert", "delete",
		"update", "select", "foreach", "execute", "immediate", "prepare", "declare",
		"fetch", "free", "rollback", "work", "commit", "run", "sleep", "import", "fgl",
		"public", "private", "returns", "returning", "null", "true", "false", "not",
		"and", "or", "is", "in", "goto", "label", "end", "with", "use", "every", "clear",
		"show", "prompt", "arrange", "grid", "scroll", "touch", "touching", "timer",
		"idle", "action", "interact", "connect", "disconnect", "set", "transaction",
		"begin", "count", "terminate", "report", "output", "order", "group", "having",
		"where", "values", "into", "database", "schema", "table", "temp", "validate",
	}
	for _, w := range kws {
		glKeywords[w] = true
	}
}

// AutovarsOn 停站后自动求值自动变量的开关状态(默认关,由前端面板开关打开)
func (s *Session) AutovarsOn() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.autovarsOn
}

// SetAutovarsOn 开关停站后自动求值自动变量。默认关:每次停站自动调度对窗口变量
// 逐个 print 求值会占住命令队列、拖慢步进(见 autovarsSettle)。
// 开启且当前已停站时立即补一次求值,结果照常以 autovars 事件推送。
func (s *Session) SetAutovarsOn(on bool) {
	s.mu.Lock()
	s.autovarsOn = on
	cur := s.cur
	st := s.state
	s.mu.Unlock()
	if on && st == StateStopped && cur.Line > 0 {
		go s.evaluateAutovars(&cur)
	}
}

// autovarsSettle 停站收口处理自动变量:开关开 → 后台求值;关 → 清掉旧值并广播空,
// 避免面板残留上一停站的结果误导(默认关时每次停站不产生求值命令)。
func (s *Session) autovarsSettle(stop *StopInfo) {
	if s.AutovarsOn() {
		if stop != nil {
			go s.evaluateAutovars(stop)
		}
		return
	}
	s.mu.Lock()
	had := len(s.lastAutovars) > 0
	s.lastAutovars = nil
	s.mu.Unlock()
	if had {
		s.emitEvent(Event{Type: "autovars", Vars: nil})
	}
}

// evaluateAutovars 停站后自动提取当前源码窗中的变量并求值。
// 尽力而为:程序已继续则静默放弃,绝不干扰主流程;结果广播 autovars 事件。
func (s *Session) evaluateAutovars(stop *StopInfo) {
	defer func() { _ = recover() }() // 后台任务,任何异常不影响会话
	if stop == nil || stop.Line <= 0 || !s.Started() {
		return
	}
	names := extractVarNames(stop.Source, stop.Line, 8)
	out := make([]VarItem, 0, len(names))
	for _, name := range names {
		if len(out) >= 8 {
			break
		}
		v, err := s.Print(name)
		if err != nil {
			if errors.Is(err, ErrNotStopped) {
				return // 程序已继续
			}
			continue // No symbol 等:不是有效变量名,丢弃
		}
		out = append(out, VarItem{Expr: name, Value: v})
	}
	s.mu.Lock()
	s.lastAutovars = out
	s.mu.Unlock()
	s.emitEvent(Event{Type: "autovars", Vars: out})
}

// ---------- 断点持久化(fgldeb 状态文件思路) ----------

// persistBPs 把当前断点(fgldb 真相)连同行文本落盘;尽力而为,失败仅记日志
func (s *Session) persistBPs() {
	if !s.cfg.BPsPersisted() {
		return
	}
	s.mu.Lock()
	bps := make([]Breakpoint, 0, len(s.bps))
	for _, b := range s.bps {
		bps = append(bps, *b)
	}
	s.mu.Unlock()
	stored := make([]StoredBP, 0, len(bps))
	srcCache := map[string][]string{}
	for _, b := range bps {
		lines, ok := srcCache[b.File]
		if !ok {
			if sf, err := s.ResolveSource(b.File, s.Module); err == nil {
				lines = strings.Split(sf.Content, "\n")
			}
			srcCache[b.File] = lines
		}
		lt := ""
		if lines != nil && b.Line >= 1 && b.Line <= len(lines) {
			lt = strings.TrimSpace(lines[b.Line-1])
		}
		stored = append(stored, StoredBP{File: b.File, Line: b.Line, Func: b.Func, Enabled: b.Enabled,
			LineText: lt, Cond: b.Cond, Ignore: b.Ignore, Kind: b.Kind})
	}
	if err := saveBPs(s.cfg.DataDir, s.cfg.Product, s.Module, s.Prog, stored); err != nil {
		s.emitEvent(Event{Type: "log", Text: "断点保存失败: " + err.Error()})
	}
}

// restoreBreakpoints 启动后从持久化文件恢复断点:
// 按保存的行文本在源码中重新定位(±15 行),源码变更不恢复错位置,定位不到跳过
func (s *Session) restoreBreakpoints() {
	if !s.cfg.BPsPersisted() {
		return
	}
	st, err := loadBPs(s.cfg.DataDir, s.cfg.Product, s.Module, s.Prog)
	if err != nil || len(st.Breakpoints) == 0 {
		return
	}
	restored, skipped := 0, 0
	srcCache := map[string][]string{}
	for _, sb := range st.Breakpoints {
		loc := fmt.Sprintf("%s:%d", sb.File, sb.Line)
		opts := BreakpointOpts{Location: loc, Cond: sb.Cond, Ignore: sb.Ignore, Kind: sb.Kind}
		if opts.Kind == BPKindWatch {
			// 监视点的"位置"就是被监视的表达式(存档在 Func 里),没有 file:line 可重定位。
			opts.Location = sb.Func
		} else if sb.LineText != "" {
			lines, ok := srcCache[sb.File]
			if !ok {
				if sf, err := s.ResolveSource(sb.File, s.Module); err == nil {
					lines = strings.Split(sf.Content, "\n")
				}
				srcCache[sb.File] = lines
			}
			if lines != nil {
				nl, found := relocateLine(lines, sb.Line, sb.LineText)
				if !found {
					skipped++
					s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("断点恢复跳过:%s:%d 与源码不再匹配", sb.File, sb.Line)})
					continue
				}
				if nl != sb.Line {
					s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("断点重定位:%s %d → %d(源码已变更)", sb.File, sb.Line, nl)})
				}
				opts.Location = fmt.Sprintf("%s:%d", sb.File, nl)
			}
		}
		// 存档可能被手改坏(或带了本版本拒绝的条件):降级为**纯位置断点**,
		// 不因为一个条件表达式把整条断点丢掉。
		//
		// 例外：监视点 + 条件（A26 明确不支持条件监视点）。降级成"纯位置断点"会把监视点
		// 本身丢掉（它的 Location 是表达式，不是 file:line），所以这里只丢**条件**，保住监视点。
		if opts.Kind == BPKindWatch && strings.TrimSpace(opts.Cond) != "" {
			s.emitEvent(Event{Type: "log", Text: fmt.Sprintf(
				"断点恢复降级:%s 是监视点，本机 fgldb 不支持条件监视点，已丢条件只恢复监视", opts.Location)})
			opts.Cond = ""
		}
		if err := opts.Validate(); err != nil {
			s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("断点恢复降级:%s 选项不合法(%v),只恢复位置", opts.Location, err)})
			opts = BreakpointOpts{Location: loc}
		}
		bp, err := s.BreakWith(opts)
		if err != nil {
			skipped++
			s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("断点恢复失败:%s: %v", opts.Location, err)})
			continue
		}
		restored++
		// 存档里"已停用"的断点必须**恢复成停用**。
		//
		// 原来的条件还要求 `bp.Note == ""`（位置被自动修正、或"该位置已有断点 #N"时就不停用）——
		// 那等于"用户明明关了断点，重开一轮它又生效了"，而 903 行这种入口断点一命中就是
		// "作业停在入口、画面不再前进"。定位修正只是**换了行号**，不该把开关一起改掉。
		if !sb.Enabled {
			if err := s.SetBPEnabled(bp.Num, false); err != nil {
				s.emitEvent(Event{Type: "log", Text: fmt.Sprintf(
					"断点恢复: %s 在存档里是停用的，但停用失败(%v) —— 它现在会生效", opts.Location, err)})
			}
		}
	}
	if restored > 0 || skipped > 0 {
		s.emitEvent(Event{Type: "log", Text: fmt.Sprintf("断点恢复完成:%d 成功,%d 跳过(共 %d)", restored, skipped, len(st.Breakpoints))})
	}
	s.persistBPs() // 回写(行号可能已重定位)
}

// relocateLine 按行文本重新定位行号:先精确匹配,再 ±15 行窗口内找相同文本
func relocateLine(lines []string, line int, text string) (int, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return line, true // 无行文本存档(旧文件):按原行号直下,由 fgldb 自行调整
	}
	if line >= 1 && line <= len(lines) && strings.TrimSpace(lines[line-1]) == text {
		return line, true
	}
	for d := 1; d <= 15; d++ {
		for _, idx := range []int{line - d, line + d} {
			if idx >= 1 && idx <= len(lines) && strings.TrimSpace(lines[idx-1]) == text {
				return idx, true
			}
		}
	}
	return 0, false
}
