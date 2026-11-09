package debug

// 会话寻址契约（`.agents/plan/tt-refactor-contract.md`，WP1 冻结）的**值只在一个地方定义**：
// `tt/internal/target`（WP3 的生产常量，见那里的 `contract.go`）。
//
// **本文件现在只是一层薄引用（WP3 改）**，它存在的唯一理由是：契约测试
// （`contract_debug_test.go`）读的是下面这些名字，而它必须读到**同一套值**。
//
// 为什么把定义搬去 `internal/target`（而不是留在测试里反过来引用）：
//
//   - 键名 / 码值 / id 文法 / F1–F20 从今天起是**生产代码**要用的东西（选择器解析、
//     落位、错误码、信封），定义在 `_test.go` 里就永远只有测试能用；
//   - 同一个值两处各写一份，就是本次重构要根除的「同一件规矩写两遍、改一处漏一处」；
//   - 仓库主人的原则是「**代码即真相，文档是脚手架**」：契约文档（markdown）会过期、
//     `.agents/` 根本不进版本库，所以值不许从文档里读，也不许散落成字面量。
//
// 为什么这样引用**不成环**：`internal/target` **零 `tt/internal` 依赖**（只 import 标准库），
// 它不知道 `internal/debug` 的存在。所以 `debug` 的测试引用它没有环，
// 而它当初也没有并进 `internal/cli/common`（那是叶子包，塞进依赖调试侧的层会破它的定位）——
// 这处偏离与依据记在计划的 **Q25**。
//
// **这个文件只放数据，不放判断** —— 判断（"这条契约今天实现了没有"）在
// `contract_debug_test.go` 与 `internal/testkit` 的扫描里。

import "tt/internal/target"

// ---------------------------------------------------------------------------
// 一、返回信封的键集（契约 5.1 / 5.2 / C5.1 / C5.10 / C5.12）
// ---------------------------------------------------------------------------

// debugContractSessionEnvelopeKeys 是**成功信封里 `session` 块的恒定键集**（契约 C5.1）。
//
// 「恒定出现」= 取不到值时给空串/零值，**不省键**。十一个键里只有 `id`/`ref`/`n` 是身份，
// 其余都是当前值快照 —— 判定身份只许用那三个（C5.1 末段）。
//
// `display` 与 `kind` 是契约里两处已知出入（示例/C5.4/C5.5 都要求它们，而 C5.1 正文
// 那句"十个键"没数进去）—— 两处的来龙去脉写在 `internal/target/contract.go` 的同名变量上，
// **不改契约文档**（它是脚手架）。今天取的齐是"按示例与 C5.4/C5.5 走"，不是定论。
//
// `ui` 块**不在这里**：它只在该会话挂着界面会话时出现（C5.7）。
var debugContractSessionEnvelopeKeys = target.SessionEnvelopeKeys

// debugContractNewSessionKeys 是 C5.3 / Q22 定稿的**纯新增**键：它们今天不在 SessionBrief 上，
// 落地之前 TestContractSessionEnvelopeKeysMatchSessionBrief 只允许缺这几个。
//
// 名单写死在 target 里（而不是让测试"看到缺了就跳过"）：少了这个约束，那条测试就成了
// "缺什么都放过"——有人删掉一个**旧**键（C5.3 明写不许）也不会红。
var debugContractNewSessionKeys = target.NewSessionKeys

// debugContractSessionOptionalKeys 是 `session` 块里**按 omitempty 规则出现**的其余键
// （契约 C5.2：键名与类型一律沿用 `internal/debug/manager.go` 的 `SessionBrief`，一个都不改名）。
//
// 它与上面那组**合起来必须正好等于 `SessionBrief` 的全部 JSON 键**。
var debugContractSessionOptionalKeys = target.SessionOptionalKeys

// debugContractSessionTopKeys 是会话级命令成功信封的顶层恒定键（契约 5.1）。
var debugContractSessionTopKeys = target.SessionTopKeys

// debugContractFailureRequredKeys 是**失败信封里恒定出现**的四键（契约 C5.10）。
var debugContractFailureRequredKeys = target.FailureRequiredKeys

// debugContractFailureOptionalKeys 是失败信封里**按条件出现**的两个键（契约 C5.10 / C5.11）。
var debugContractFailureOptionalKeys = target.FailureOptionalKeys

// debugContractCandidateKeys 是**候选清单每个元素的键集，全契约的唯一出处**
// （契约 C5.12：只这九键，是 `session` 的真子集；C2.6 只引用、不复制）。
var debugContractCandidateKeys = target.CandidateKeys

// debugContractCandidatesRequiredCodes 是**必须带候选清单**的机器码（契约 C5.11）。
var debugContractCandidatesRequiredCodes = target.CandidatesRequiredCodes

// ---------------------------------------------------------------------------
// 二、七个机器码 + 退出码（契约 4.1 码表 / 4.1.1 code 取值表 / C5.13）
// ---------------------------------------------------------------------------

// 契约 4.1.1 的机器码（取值定义在 `internal/target/contract.go`，那里写着它们是**对外契约**）。
//
// 命名一律 `debugContractCode*`：**不与 `internal/output` 的常量同名**，
// 这样"新码还没进 output"这件事一眼可见，也逼着契约测试去核对两边的取值相等。
const (
	debugContractCodeAmbiguous  = target.CodeAmbiguous
	debugContractCodeNotFound   = target.CodeNotFound
	debugContractCodeStale      = target.CodeStale
	debugContractCodeTargetBusy = target.CodeTargetBusy
	debugContractCodeNameTaken  = target.CodeNameTaken
	// NO_SESSION 复用现行常量（B5 的 `CodeNoSession`），这里是它在契约里的取值。
	debugContractCodeNoSession = target.CodeNoSession
	// SESSION_STATE 复用现行常量（B5 的 `CodeSessionState`）：会话存在但状态不符（退 4）。
	debugContractCodeState = target.CodeState
	// ENV_UNKNOWN / USAGE 也复用现行常量（B5）。
	debugContractCodeEnvUnknown = target.CodeEnvUnknown
	debugContractCodeUsage      = target.CodeUsage
)

// debugContractCodeExits 是契约 4.1.1 那张表的**退出码一列**（新写法那一档）。
//
// 注意 `USAGE` 会出现在两个退出码档下（契约 4.1.1 的两条口径）：新写法下退 2，
// 老路径（F16/F17）下退 1。表里记的是新写法那一档；老路径由
// `debugContractFlagOnWrongCommandExit` 单独写死。
var debugContractCodeExits = target.CodeExits

// 契约 4.1 的其余档位：0 成功、1 运行失败、5 目标不存在、6 体检不过。
const (
	debugContractExitOK        = target.ExitOK
	debugContractExitRunFailed = target.ExitRunFailed
	debugContractExitUsage     = target.ExitUsage
	debugContractExitNoSession = target.ExitNoSession
	debugContractExitState     = target.ExitState
	debugContractExitMissing   = target.ExitMissing
	debugContractExitPreflight = target.ExitPreflight
)

// debugContractFlagOnWrongCommandExit 是 F16/F17 的退出码（契约 C2.8b / C2.8c / Q18）：
// `tt debug` 组**不补** `FlagErrorFunc`，所以"标志挂错了命令"走 cobra 老路径 → 根命令兜底 1。
// 信封里仍然是 `code: USAGE`。
const debugContractFlagOnWrongCommandExit = target.FlagOnWrongCommandExit

// ---------------------------------------------------------------------------
// 三、规范 id 与选择器的文法（契约 2.1.2 / 2.1.4 / C2.2 / C2.3 / C2.5）
// ---------------------------------------------------------------------------

const (
	// debugContractSeqSep 是规范 id 里的序号分隔段（Q1-b「序号」）。
	debugContractSeqSep = target.SeqSep
	// debugContractSelectorSep 是选择器里的段分隔符。
	debugContractSelectorSep = target.SelectorSep
)

// debugContractRefPattern 是**规范 id 的文法**（契约 2.1.2）：
//
//	规范 id := 环境名 "#" 序号      序号 := 十进制整数，>= 1
//
// 环境名本身不做前缀匹配、不做大小写折叠（C2.1），所以 `[^#/]+` 只排除分隔符本身。
// 序号**不写前导零**（`#01` 与 `#1` 是同一个地址）。
const debugContractRefPattern = target.RefPattern

// 别名（`--name`）的取值纪律（契约 C2.4 / C2.5 / Q17）。
const (
	debugContractAliasMinRunes = target.AliasMinRunes
	debugContractAliasMaxRunes = target.AliasMaxRunes
	debugContractAliasForbid   = target.AliasForbid
)

// 选择器第二段的两种形态（契约 C2.3）与作业名合法字符集（B29）。
const (
	debugContractSecondSegEnt = target.SecondSegEnt
	debugContractSecondSegJob = target.SecondSegJob
	debugContractJobPattern   = target.JobPattern
)

// ---------------------------------------------------------------------------
// 四、非法写法一览 F1–F20（契约 2.4；F20 是 Q23 新增的"起会话"专属写法）
// ---------------------------------------------------------------------------

// debugContractInvalidForm 是 2.4 那张表的一行（类型与数据都在 `internal/target`，
// 那里是**唯一出处**：它同时被生产代码与两处测试读）。
type debugContractInvalidForm = target.InvalidForm

// debugContractInvalidForms 是 F1–F20 逐条（键必须恰好是 F1…F20 共二十条）。
var debugContractInvalidForms = target.InvalidForms
