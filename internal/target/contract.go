// Package target 是会话寻址的**目标层**。
//
// 它把"用户写的选择器"在"会话池的只读视图"上落成**唯一一条会话**；落不上时给出
// 稳定的机器码 + 候选清单 + 退出码。依据是 `.agents/plan/tt-refactor-contract.md`
// （WP1 冻结的契约）与 `.agents/plan/tt-refactor-plan.md` 的 Q1–Q28。
//
// # 三条纪律
//
//  1. **零 tt/internal 依赖**（只 import 标准库）。它要干的是纯函数：文法解析、层级校验、
//     歧义判定、候选表、错误码、结果信封；输入是"会话列表的视图"（一组结构体），
//     **不需要知道会话是怎么跑起来的** —— CLI 侧自己把 HTTP 拿到的 JSON 转成视图喂进来。
//     这也是它当初没有并进 `internal/cli/common` 的原因（Q25 推翻了 Q5）：`common`
//     是叶子包，往里塞依赖调试侧的层会破它的定位；而 `internal/debug` 反向依赖
//     `internal/cli` 为 0，所以本包引用调试侧的**契约值**也不成环。
//  2. **不许依赖 `Session`**（会话池那个业务类型）。一旦开始依赖它，说明业务逻辑渗进来了
//     （WP3 的自查）：本包只认下面那个十字段的只读视图。
//  3. **值是唯一一份**：契约里的键名 / 码值 / id 文法 / F1–F20 都定义在本包（生产常量），
//     `internal/debug` 的契约测试从本包引用它们 —— 不许留两份互不知情的副本。
//
// # 本包**不**负责什么
//
//   - 不发请求、不写盘、不读环境变量（`DefaultActor` 取的是注入进来的 getenv）；
//   - 不认识 `--json` / 表格输出：信封是**数据**，`internal/output` 负责渲染；
//     失败转成 `*output.Error`（契约 C4.1）由 WP4 的接线做；
//   - 人话候选块只给**候选行**（C5.14）；F1 那句"请指明其一，例如：tt debug … --session X"
//     要原始命令行，由 CLI 自己拼。
package target

// ===========================================================================
// 一、返回信封的键集（契约 5.1 / 5.2 / C5.1 / C5.10 / C5.12）
// ===========================================================================

// SessionEnvelopeKeys 是**成功信封里 `session` 块的恒定键集**（契约 C5.1）。
//
// 「恒定出现」= 取不到值时给空串/零值，**不省键**。十二个键里只有 `id`/`ref`/`n`
// 是身份的组成部分，其余都是当前值快照 —— 判定身份只许用那三个（C5.1 末段）。
//
// `display` 也在这个块里：契约 5.1 的 JSON 示例里逐字有它，C5.4 又把它与 `ref`/`id`
// 并列成"三分工"。C5.1 的正文那句"十个键"没把它数进去 —— 这是契约自己的一处遗漏
// （示例与 C5.4 都要求它），本次按示例与 C5.4 取齐。`kind` 是同类出入（C5.5 说它恒定，
// 但它今天不在 `SessionBrief` 上）。
//

// **这两处都不是定论**：等契约文档退场、以代码为准的那一天，`display` 要么被正式纳入
// 键集、要么被判冗余下线；`kind` 要么落成真字段、要么退出键集。
var SessionEnvelopeKeys = []string{
	"id", "ref", "display", "name", "env", "ent", "n", "kind", "prog", "state", "epoch",
}

// NewSessionKeys 是 C5.3 / Q22 定稿的**纯新增**键：它们今天不在 `SessionBrief` 上，
// 落地之前契约测试只允许缺这几个。名单写死（而不是"缺什么都放过"），
// 否则有人删掉一个**旧**键（C5.3 明写不许）也不会红。
//
//   - `ref` / `name` / `n` / `epoch` 是 Q22 点名的五个新键里的四个；
//   - `ent` 也是新增（C5.3 第二条：`string`）；
//   - `display` 见 SessionEnvelopeKeys 的注释；
//   - `kind` 见 C5.5（本契约只寻址 SSH 会话，所以取值恒为 KindSSH）；

var NewSessionKeys = []string{"ref", "display", "kind", "name", "ent", "n", "epoch"}

// SessionOptionalKeys 是 `session` 块里**按 omitempty 规则出现**的其余键
// （C5.2：键名与类型一律沿用 `internal/debug/manager.go` 的 `SessionBrief`，一个都不改名）。
//
// 它与 SessionEnvelopeKeys **合起来必须正好等于 `SessionBrief` 的全部 JSON 键**：
// 少一个说明旧键被改了名（C5.3 明写「一个旧键都不改、不改名、不删」），
// 多一个说明有人往信封里塞了新形状。
var SessionOptionalKeys = []string{
	"module", "runProg", "started", "file", "line", "func", "reason",
	"holdingSeconds", "breakpoints", "watchdogSeconds", "watchdogDisabled",
	"waitingForUser", "waitingKind", "silentSeconds", "mode", "inflight",
}

// SessionTopKeys 是会话级命令成功信封的顶层恒定键（契约 5.1）。
// `ok` 与平铺的环境信息（`output.Meta`）沿用现行成功信封（B7），不在这里重复声明。
var SessionTopKeys = []string{"session", "requested"}

// KindSSH 是 `session.kind` 的取值（C5.5 / Q9）：本契约只寻址 SSH 会话，
// 界面会话是它的**属性**（走 `ui` 块），不是第二个目标。
const KindSSH = "ssh"

// FailureRequiredKeys 是**失败信封里恒定出现**的四键（C5.10）。
var FailureRequiredKeys = []string{"ok", "code", "error", "exitCode"}

// FailureOptionalKeys 是失败信封里**按条件出现**的两个键（C5.10 / C5.11）：
// `hint` 有修复建议才出现；`candidates` 只在 C5.11 列的那五种码下必须出现。
var FailureOptionalKeys = []string{"hint", "candidates"}

// CandidateKeys 是**候选清单每个元素的键集，全契约的唯一出处**
// （C5.12：只这九键，是 `session` 的真子集；2.4 的 C2.6 只引用、不复制）。
//
// 契约测试会断言 "Candidate 这个类型的 json 标签与它逐字一致" ——
// 常量表给测试用、struct 标签给序列化用，两者必须是同一套键。
var CandidateKeys = []string{
	"id", "ref", "display", "name", "env", "ent", "n", "prog", "state",
}

// CandidatesRequiredCodes 是**必须带候选清单**的机器码（C5.11）：
// 歧义 / 指名不存在 / 句柄过期 / 别名撞车 / 别名指向的会话不可复用。
// 与 2.4 的 F 表（F1/F3/F4/F5/F7/F8/F15/F18/F20）逐条对得上。
var CandidatesRequiredCodes = []string{
	CodeAmbiguous, CodeNotFound, CodeStale, CodeNameTaken, CodeTargetBusy,
}

// ===========================================================================
// 二、机器码 + 退出码（契约 4.1 码表 / 4.1.1 code 取值表 / C5.13 / Q15）
// ===========================================================================

// 契约 4.1.1 的机器码。取值是**对外契约**（C5.13：加新码可以，改旧码的名字不行），
// 所以这里写的是字面量，并且由 values_test.go 逐字守着。
//
// `NO_SESSION` / `SESSION_STATE` / `ENV_UNKNOWN` / `USAGE` 复用现行常量
// （`internal/output/errors.go` 的 B5 那一批）—— 本包**不** import output，
// 所以这里按契约的取值**再写一次字面量**并核对（WP4 接线时两侧要相等，
// 那一步由 `internal/debug` 的契约测试与 output 的常量表共同保证）。
const (
	CodeAmbiguous  = "SESSION_AMBIGUOUS"   // 选择器命中 >=2 条（F1/F3/F4）
	CodeNotFound   = "SESSION_NOT_FOUND"   // 选择器指名的东西不存在（F5/F7/F8）
	CodeStale      = "SESSION_STALE"       // 环境与序号都对，但句柄是上一代（F15）
	CodeTargetBusy = "SESSION_TARGET_BUSY" // 别名已存在且那条会话不能复用（F20 / Q23）
	CodeNameTaken  = "SESSION_NAME_TAKEN"  // 别名已被占用（F18；**退 2 不退 3**）
	CodeNoSession  = "NO_SESSION"          // 会话池里 0 条 SSH 会话（F2 / B5）
	CodeState      = "SESSION_STATE"       // 会话存在但状态不符（4.1 表 / B5）
	CodeEnvUnknown = "ENV_UNKNOWN"         // 环境名不在配置里（F6 / C2.1 / B5）
	CodeUsage      = "USAGE"               // 文法非法 / 标志冲突 / 别名取值非法（F9–F14、F19）
	// CodePurposeMismatch 是同名 SSH 会话存在但**用途**不同（W1：名字是幂等键，
	// 键撞了而含义不同时不能悄悄返回既有那条）。与 `SESSION_NAME_TAKEN` 同一档（退 2）。
	CodePurposeMismatch = "SESSION_PURPOSE_MISMATCH"
	// CodeReadOnly 是"这条 SSH 会话当前不接受这个写操作"：用途（record）/ 模式 / 角色
	// 任一不允（W7 的三类能力）。与 `SESSION_STATE` 同一档（退 4）：都是"存在但此刻不行"。
	CodeReadOnly = "SESSION_READONLY"
)

// CodeExits 是契约 4.1.1 那张表的**退出码一列**（**新场景**那一档，Q15 分档生效：
// 新场景无条件按它退；既有场景只有真用了新写法才按它退，见 ExitFor）。
//
// `USAGE` 会出现在两个退出码档下（4.1.1 的两条口径）：新写法下退 2，
// 老路径（F16/F17，标志写在**不接受**它的命令上）下退 1 —— 后者由 ExitFor 负责。
var CodeExits = map[string]int{
	CodeAmbiguous:       3,
	CodeNotFound:        3,
	CodeStale:           3,
	CodeTargetBusy:      3,
	CodeNoSession:       3,
	CodeEnvUnknown:      2,
	CodeUsage:           2,
	CodeNameTaken:       2,
	CodePurposeMismatch: 2,
	CodeState:           4,
	CodeReadOnly:        4,
}

// 契约 4.1 的其余档位：0 成功、1 运行失败、5 目标不存在、6 体检不过。
// 5 与 6 是既有那条线**现成**的码（B3），本契约不改它一条。
const (
	ExitOK        = 0
	ExitRunFailed = 1
	ExitUsage     = 2
	ExitNoSession = 3
	ExitState     = 4
	ExitMissing   = 5
	ExitPreflight = 6
)

// FlagOnWrongCommandExit 是 F16/F17 的退出码（C2.8b / C2.8c / Q18）：
// `tt debug` 组**不补** `FlagErrorFunc`，所以"标志挂错了命令"走 cobra 老路径 →
// 根命令兜底 1，与今天 `--bogus` 完全一致。信封里仍然是 `code: USAGE`。
const FlagOnWrongCommandExit = ExitRunFailed

// ExitFor 按契约 4.2 的定稿给出退出码（Q15 的"分档生效"，判据经本轮改正）。
//
// # 判据是"这个场景是不是新场景"，不是"命令行里有没有那个标志"
//
//	newScenario = 这条失败落在**新场景**里（今天不可能发生；取值来自 `Failure.NewScenario`）
//	newSyntax   = 命令行里真的出现了 `--session` / `--name`（老脚本不可能有的写法）
//
// 两条规则，按顺序：
//
//  1. **新场景 → 无条件走 4.1.1 那张表。** H2 的全句是「**只有一条会话**时，所有命令的
//     输出与今天逐字相同」—— 它保护的是**今天已有的行为**；新场景今天不存在，
//     H2 的射程够不到它，所以不必（也不许）先等一个 `--session` 才肯给细码。
//     F1 就是这条的落点：多会话 + 不给目标今天会**静默取第一条并成功**，
//     "成功"不是要保护的旧行为。
//  2. **既有场景 → 维持今天的码（1）。** 这是 H2 真正的落点：老脚本今天看到的是 1。
//     只有真的用了新写法（老脚本里没有一个字提到 `--session`/`--name`）才走 4.1.1 ——
//     4.2 的 H2 表逐行写着这条两档（F2 带 `--session` → 3、F6 写在 `--session` 里 → 2）。
//
// 一句话：**今天没有的行为不欠谁一个 1。** 语义一致性仍然由失败信封的 `code` 承担（Q15），
// 调用方的判断一律读 `code`。
func ExitFor(code string, newScenario, newSyntax bool) int {
	if !newScenario && !newSyntax {
		return ExitRunFailed
	}
	// 4.1.1 表外的**现行码**（QUERY_FAILED / CONNECT_FAILED / NOT_FOUND …）都是"运行失败"档。
	return exitForCode(code)
}

// CodeRequiresCandidates 判断某个码下的失败信封**必须**出现 `candidates` 键（C5.11）。
func CodeRequiresCandidates(code string) bool {
	for _, c := range CandidatesRequiredCodes {
		if c == code {
			return true
		}
	}
	return false
}

// ===========================================================================
// 三、规范 id、选择器与别名的文法（契约 2.1.2 / 2.1.4 / C2.2 / C2.3 / C2.5 / Q17）
// ===========================================================================

const (
	// SeqSep 是规范 id 里的序号分隔段（Q1-b「序号」）。规范 id = 环境名 + 它 + 序号。
	SeqSep = "#"
	// SelectorSep 是选择器里的段分隔符。`--session` 的取值里含 `/` 或 `#` 的
	// 一律按选择器解析，不含这两个字符的才按别名解析（C2.4 的落地判据）。
	SelectorSep = "/"
	// SelectorSeps 是上面两个字符的集合（分流判据用）。
	SelectorSeps = SeqSep + SelectorSep
)

// RefPattern 是**规范 id 的文法**（契约 2.1.2）：
//
//	规范 id := 环境名 "#" 序号      序号 := 十进制整数，>= 1
//
// 环境名本身不做前缀匹配、不做大小写折叠（C2.1：写错一个字就是 ENV_UNKNOWN，不猜、不回落），
// 所以 `[^#/]+` 只排除分隔符本身 —— 环境名允许中文、空格与任意其它字符。
//
// 序号**不写前导零**：`#01` 与 `#1` 是同一个地址，写成两个写法对不上账（F9）。
const RefPattern = `^[^#/]+#[1-9][0-9]*$`

// 别名（`--name`）的取值纪律（C2.4 / C2.5 / Q17）：
// 非空、去两侧空白后 1–64 个字符、**只排除 `/` 与 `#`**，其余（中文、空格、标点）按原样允许。
const (
	AliasMinRunes = 1
	AliasMaxRunes = 64
	AliasForbid   = "/#"
)

// 选择器第二段的两种形态（C2.3）：全数字 → 企业编号；否则 → 作业名。
// 作业名合法字符集见 B29（JobPattern），首字符是字母，所以两种形态不会撞。
const (
	SecondSegEnt = "ent" // 全数字
	SecondSegJob = "job" // 其余
	JobPattern   = `^[A-Za-z][A-Za-z0-9_-]{0,63}$`
)

// 会话状态的五个取值（B30）。它们是 `SessionBrief.state` 的取值域，
// 也是 Q23 判"能不能复用别名"的依据。
const (
	StateIdle    = "idle"    // 空闲
	StateLoading = "loading" // 连接中…
	StateStopped = "stopped" // 已停站
	StateRunning = "running" // 运行中
	StateExit    = "exit"    // 已断开
)

// ===========================================================================
// 四、非法写法一览 F1–F20（契约 2.4；F20 是 Q23 新增的"起会话"专属写法）
// ===========================================================================

// InvalidForm 是 2.4 那张表的一行。每条都写明：**退出码 + 文案形状 + 候选清单是否出现**。
//
// 它是**生产常量**（不只是测试数据）：契约里可断言的形状就落在它上面，
// `internal/debug` 的契约测试从本包引用它 —— 一个规矩只定义一处。
type InvalidForm struct {
	// Key 是 F1 … F20。
	Key string
	// Write 是这条非法写法的输入（人话描述 + 可照着敲的选择器值）。
	Write string
	// Code 见 4.1.1（唯一清单）。
	Code string
	// Exit 见 2.4 那一列。F16/F17 是 1（老路径，见 C2.8b）。
	Exit int
	// Candidates 见 2.4 那一列：true = 候选清单**必须出现**。
	Candidates bool
}

// InvalidForms 是 F1–F20 逐条（键必须恰好连号，契约测试双向核对）。
var InvalidForms = []InvalidForm{
	{Key: "F1", Write: "省略目标，服务端有 >=2 条 SSH 会话",
		Code: CodeAmbiguous, Exit: 3, Candidates: true},
	{Key: "F2", Write: "省略 --session（或给了名字），服务端 0 条 SSH 会话",
		Code: CodeNoSession, Exit: 3, Candidates: false},
	{Key: "F3", Write: "--session 恒烁正式区/apmt520，该（环境, 作业）下有 >=2 条",
		Code: CodeAmbiguous, Exit: 3, Candidates: true},
	{Key: "F4", Write: "--session 恒烁正式区，该环境下有 >=2 条",
		Code: CodeAmbiguous, Exit: 3, Candidates: true},
	{Key: "F5", Write: "--session 汇率单X（别名不存在）",
		Code: CodeNotFound, Exit: 3, Candidates: true},
	{Key: "F6", Write: "--session 不存在的环境/apmt520（环境不存在）",
		Code: CodeEnvUnknown, Exit: 2, Candidates: false},
	{Key: "F7", Write: "--session 恒烁正式区/不存在的作业（该环境下该作业无会话）",
		Code: CodeNotFound, Exit: 3, Candidates: true},
	{Key: "F8", Write: "--session 恒烁正式区/apmt520#7（序号越界；<M> 按环境算，见 C2.11）",
		Code: CodeNotFound, Exit: 3, Candidates: true},
	{Key: "F9", Write: `--session 恒烁正式区/apmt520#x（也含 #1.5 / # / #0 / #-1 / #01）`,
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F10", Write: "--session 恒烁正式区/作业/多一段（段数不对）；也含 恒烁正式区//apmt520（空段）",
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F11", Write: `--session ""（给了空串：按文法非法处理，不当成"省略"）`,
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F12", Write: "--session 测试1012/apmt520 与 --env 恒烁正式区 并存（环境矛盾）",
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F13", Write: "--session 恒烁正式区/88 与 --ent 99 并存（企业矛盾）",
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F14", Write: "--session 恒烁正式区/apmt520 --ent 99，而该会话 ENT 是 88",
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F15", Write: "--session 的环境与序号都对，但句柄是上一代（epoch 过期）",
		Code: CodeStale, Exit: 3, Candidates: true},
	{Key: "F16", Write: "--session 挂到不接受它的命令上（tt debug status --session X / tt debug sql --session X）",
		Code: CodeUsage, Exit: FlagOnWrongCommandExit, Candidates: false},
	{Key: "F17", Write: "--name 挂到非建会话命令上（tt debug exec … --name X）",
		Code: CodeUsage, Exit: FlagOnWrongCommandExit, Candidates: false},
	{Key: "F18", Write: `--name 汇率单 与已存在的别名撞车`,
		Code: CodeNameTaken, Exit: 2, Candidates: true},
	{Key: "F19", Write: `--name 含 "/" 或 "#"，或为空串`,
		Code: CodeUsage, Exit: 2, Candidates: false},
	{Key: "F20", Write: "--name 汇率单，而该别名指向的会话正在跑 / 已停站（Q23：本轮新增，不在输入集 B 里）",
		Code: CodeTargetBusy, Exit: 3, Candidates: true},
}
