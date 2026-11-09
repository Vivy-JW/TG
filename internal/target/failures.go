package target

import (
	"fmt"
	"strings"
)

// 两个新建会话时才出现的标志名（F16 / F17 / C3.5 用它们造错）。
const (
	FlagSession = "--session"
	FlagName    = "--name"
)

// Failure 是一次寻址失败的**数据**（契约 5.2 的失败信封 + 2.4 的文案形状）。
//
// 它同时实现了 `error`：这样解析、落位、层级校验、世代校验可以只用一个错误类型返回，
// WP4 再把它包成 `internal/output` 的 `*Error`（契约 C4.1 要求所有失败路径带 Code + Exit）。
//
// **Message 不带 `错误：` 前缀**：契约 5.2 的信封示例里 `error` 的值是
// `"有 2 条 SSH 会话，无法确定你要哪一条"`，而 2.4 的"文案形状"一列写的是
// `错误：有 <N> 条 SSH 会话…`。那一小段是**人话渲染**加的前缀（`HumanError`），
// 不是信封里的字段 —— 否则调用方（Agent）读到的 machine-ish 文案里会多一段固定噪声。
type Failure struct {
	// Code 见契约 4.1.1（唯一清单）。Agent 靠它分支，**不读数字**（Q15）。
	Code string
	// Exit 是这条失败的退出码（4.1.1 那一档；老路径的 1 由 ExitFor 分档决定）。
	Exit int
	// NewScenario 标记这条失败落在**"今天不可能发生"的新场景**里。
	//
	// 分档看的是**这个场景今天存不存在**，不是"命令行里有没有 --session 这个字"
	// （这是本轮改正的判据，见 `ExitFor`）：
	//
	//   - **今天就有、今天就失败的**（池里 0 条会话、环境名写错、少给必填参数、
	//     标志挂错命令…）→ 老脚本看到的那个 1 仍要保护，这一位是 false；
	//   - **今天压根不会发生**的（F1"多会话不取第一项"、选择器指名的东西不存在、
	//     句柄过期、别名撞车、选择器文法非法…）→ **没有旧行为要保护**，这一位是 true，
	//     退出码一律按 4.1.1 那张表给。
	//
	// 依据是 H2 的**全句**：「**只有一条会话**时，所有命令的输出与今天逐字相同」
	// （`tt-refactor-plan.md` 的 H2 行）。H2 保护的是"今天已有的行为"那一档 ——
	// 新场景今天不存在，H2 的射程够不到它。
	//
	// 反例（也是本轮点名的那处错判）：多会话 + 不给目标（F1）今天**会静默取第一条并成功**
	// —— "成功"不是要保护的旧行为，所以它无条件退 3，与有没有打 `--session` 无关。
	NewScenario bool
	// Message 是人读错误（信封的 `error` 字段）。
	Message string
	// Hint 是修复建议；空 = 信封里不出现这个键（C5.10）。
	Hint string
	// Candidates 只在那五种码下出现（C5.11）；空切片 = "出现了但没有候选"。
	Candidates []Candidate
}

// Error 让 Failure 能当 error 用（纯函数只返回一个错误类型）。
func (f *Failure) Error() string { return f.Message }

// HumanError 是人话那一行（`错误：…`），形状照契约 2.4 的"文案形状"一列。
func (f *Failure) HumanError() string { return "错误：" + f.Message }

// newFailure 造一条失败，退出码按 4.1.1 取（"码与退出码同源"，C4.1）。
//
// 默认**不是**新场景（`NewScenario = false`）：既有失败是多数，也是要小心对待的那一半。
// 落在新场景里的那几族用 newScenarioFailure 造，一处一处写明白。
func newFailure(code, format string, a ...any) *Failure {
	return &Failure{Code: code, Exit: exitForCode(code), Message: fmt.Sprintf(format, a...)}
}

// newScenarioFailure 与 newFailure 同形，只是把这条失败登记成**新场景**
// （"今天不可能发生"，见 Failure.NewScenario 与 contract.go 的 ExitFor）。
func newScenarioFailure(code, format string, a ...any) *Failure {
	f := newFailure(code, format, a...)
	f.NewScenario = true
	return f
}

// exitForCode 取 4.1.1 表里的退出码；表外的现行码归"运行失败"档。
func exitForCode(code string) int {
	if exit, ok := CodeExits[code]; ok {
		return exit
	}
	return ExitRunFailed
}

// ---------------------------------------------------------------------------
// F9 / F10 / F11：选择器文法非法（都在**发请求之前**判定，C2.8）
// ---------------------------------------------------------------------------

// failureSeqInvalid 是 F9：序号非十进制正整数（`#x` / `#1.5` / `#` / `#0` / `#-1` / `#01`）。
// **新场景**：今天没有 `--session` 这个标志（B19），这一族错今天不可能发生 → 退 2。
func failureSeqInvalid(seq string) *Failure {
	return newScenarioFailure(CodeUsage, "选择器里的序号必须是 >=1 的十进制整数：%q", SeqSep+seq)
}

// failureSelectorGrammar 是 F10 / F11：段数不对 / 空段 / 空串（空串**不**当成省略）。
// **新场景**：同上，只有读了 `--session` 才判得出 → 退 2。
func failureSelectorGrammar(raw string) *Failure {
	return newScenarioFailure(CodeUsage,
		"选择器文法非法：%q（写法：环境 | 环境/作业 | 环境/作业#序号 | 环境/企业编号）", raw)
}

// ---------------------------------------------------------------------------
// F1–F8：落位失败
// ---------------------------------------------------------------------------

// failureAmbiguous 是 F1 / F3 / F4：选择器（或省略）命中 >=2 条。
//
// **新场景**（F1 那一支尤其要点明）：今天"多条会话时取哪一条"这个**问题不存在** ——
// 旧入口静默取列表第一条并**成功返回**。也就是说今天没有"退 1"这个行为要保护，
// 所以它无条件走 4.1.1（退 3），与有没有打 `--session` 无关。
func failureAmbiguous(raw string, hits []Session) *Failure {
	f := newScenarioFailure(CodeAmbiguous, "%s 命中 %d 条 SSH 会话，无法确定你要哪一条", raw, len(hits))
	f.Hint = "请用 --session <环境>/<作业>#<序号> 指明"
	f.Candidates = candidatesOf(hits)
	return f
}

// failureAmbiguousOmitted 是 F1：省略目标而池子（或收窄后的环境）里有多条。
// 它与 F3/F4 只有文案不同（F1 没有"<选择器>"可写）。
//
// 这一条是**本轮判据改正的落点**：判据是"这个场景是不是新场景"，不是"命令行里有没有
// `--session` 这个字"。Agent 恰恰是在**没打** `--session` 时才撞上歧义 ——
// 它最需要机器可读码（Q15 的 `code` 与"落不上目标就报错"）的那一刻，
// 不许因为"它没打标志"就把码降回 1。
func failureAmbiguousOmitted(hits []Session) *Failure {
	f := newScenarioFailure(CodeAmbiguous, "有 %d 条 SSH 会话，无法确定你要哪一条", len(hits))
	f.Hint = "用 --session <别名> 或 --session <ref> 或 --session <环境>/<作业>#<序号> 指明"
	f.Candidates = candidatesOf(hits)
	return f
}

// failureNoSession 是 F2：会话池里 0 条 SSH 会话（候选列**不出现**）。
//
// **既有失败**：今天池里一条会话都没有时，旧入口也报"没有活动会话"并退 1
// （契约 4.2 的 H2 表第一行逐字记着这件事）。这一档要保护 → `NewScenario` 是 false：
// 老写法（没打 `--session`/`--name`）维持 1；真用了新写法才按 4.1.1 退 3（4.1.1 的 F2 行）。
func failureNoSession() *Failure {
	return newFailure(CodeNoSession,
		"没有 SSH 会话：先运行 tt debug start <作业> --env <环境> 起一条")
}

// failureNoSessionInScope 是"池里有会话，但这个环境里一条都没有"：
// 码用 `SESSION_NOT_FOUND`（指名的那一层不存在，而不是整池空 —— C2.7）。
//
// **新场景**：今天"你要的那个环境里没有会话"这件事**不会失败** —— 旧入口根本不看
// `--env`，它取列表第一条（可能是**另一个环境**的会话）并看起来成功。没有旧行为要保护。
func failureNoSessionInScope(env string, all []Session) *Failure {
	f := newScenarioFailure(CodeNotFound, "环境 %q 下没有 SSH 会话", env)
	f.Hint = fmt.Sprintf("先运行 tt debug start <作业> --env %s 起一条", env)
	f.Candidates = candidatesOf(all)
	return f
}

// failureAliasNotFound 是 F5：别名不存在。候选块**只列有名字的那些**（2.4 的 F5 行）。
// **新场景**：别名寻址要 `--session <别名>`（B19/B20）→ 无条件走 4.1.1（退 3）。
func failureAliasNotFound(alias string, named []Session) *Failure {
	f := newScenarioFailure(CodeNotFound, "没有别名为 %q 的 SSH 会话", alias)
	f.Hint = "去掉 --session 可看全部（tt debug status）"
	f.Candidates = candidatesOf(named)
	return f
}

// failureEnvUnknown 是 F6：环境名不在配置里。候选列是**环境名清单**（不是会话候选块）。
//
// **既有失败**：环境名写错今天就会失败（`--env 无此环境` → 服务端 404 → 退 1），
// 这一档要保护 → `NewScenario` 是 false：老写法维持 1；写在 `--session` 里（新写法）
// 才按 4.1.1 退 2（契约 4.2 的 H2 表逐行写着这条两档）。
func failureEnvUnknown(env string, known []string) *Failure {
	f := newFailure(CodeEnvUnknown, "环境 %q 不存在", env)
	f.Hint = "可用环境：" + strings.Join(known, "、")
	return f
}

// failureJobNotFound 是 F7（也是 C2.12：写法 B 的作业段核对不过）。
// **新场景**：作业段只可能写在 `--session` 里 → 退 3。
func failureJobNotFound(env, job string, inEnv []Session) *Failure {
	f := newScenarioFailure(CodeNotFound, "环境 %q 下没有作业 %q 的 SSH 会话", env, job)
	f.Candidates = candidatesOf(inEnv)
	return f
}

// failureEntNotFound 是 `环境/企业编号` 形态的"没有这条"。
// 契约没有单列一条 F（F7 是作业那一支），但它与 F7 同类：指名的东西压根不存在（C2.7）→
// `SESSION_NOT_FOUND`、候选块列**该环境下的全部会话**。**新场景**（选择器形态）→ 退 3。
func failureEntNotFound(env, ent string, inEnv []Session) *Failure {
	f := newScenarioFailure(CodeNotFound, "环境 %q 下没有企业编号 %s 的 SSH 会话", env, ent)
	f.Candidates = candidatesOf(inEnv)
	return f
}

// failureSeqOutOfRange 是 F8：序号越界。`scope` 是 `<环境>` 或 `<环境>/<作业>`；
// `m` 是**同一环境下**的会话条数（C2.11）；候选块列的就是这 m 条。
// **新场景**（`#序号` 只可能写在 `--session` 里）→ 退 3。
func failureSeqOutOfRange(scope string, m, n int, inEnv []Session) *Failure {
	f := newScenarioFailure(CodeNotFound, "%s 只有 %d 条 SSH 会话，没有 %s%d", scope, m, SeqSep, n)
	f.Hint = fmt.Sprintf("先运行 tt debug status 看该环境下现有 %d 条", m)
	f.Candidates = candidatesOf(inEnv)
	return f
}

// ---------------------------------------------------------------------------
// F12–F14：标志冲突（C3.3 的四种组合里的三种）
// ---------------------------------------------------------------------------

// failureEnvConflict 是 F12 / C3.3-1：`--session` 里的环境与 `--env` 矛盾。
// **新场景**：这一族冲突今天不可能存在（`--session` 今天没被注册，B19）→ 退 2。
func failureEnvConflict(fromSelector, fromFlag string) *Failure {
	return newScenarioFailure(CodeUsage,
		"--session 里的环境是 %q，与 --env 给的 %q 矛盾；两者只能有一个是权威",
		fromSelector, fromFlag)
}

// failureEntConflict 是 F13 / C3.3-2：`--session` 里的企业编号与 `--ent` 矛盾。
// **新场景** → 退 2。
func failureEntConflict(fromSelector, fromFlag string) *Failure {
	return newScenarioFailure(CodeUsage,
		"--session 里的企业是 %s，与 --ent 给的 %s 矛盾；两者只能有一个是权威",
		fromSelector, fromFlag)
}

// failureSessionEntConflict 是 F14 / C3.3-3：落到的那条会话实际 ENT 与 `--ent` 不同。
// **新场景** → 退 2。
func failureSessionEntConflict(sel string, actual, fromFlag string) *Failure {
	return newScenarioFailure(CodeUsage,
		"%s 的 SSH 会话企业是 %s，与 --ent 给的 %s 矛盾", sel, actual, fromFlag)
}

// ---------------------------------------------------------------------------
// F15：世代失效
// ---------------------------------------------------------------------------

// failureStale 是 F15：环境与序号都对，但句柄是上一代。候选块给**当前代**那一条。
//
// **新场景**：今天没有"代"这个概念（B31），这个失败今天无从发生 → 无条件退 3。
func failureStale(s Session, wantEpoch string) *Failure {
	f := newScenarioFailure(CodeStale,
		"这条 SSH 会话已失效（%s 当前是第 %s 代，你手上的是第 %s 代）；重新 tt debug status 取一次",
		s.CanonicalRef(), s.Epoch, wantEpoch)
	f.Candidates = candidatesOf([]Session{s})
	return f
}

// ---------------------------------------------------------------------------
// F16 / F17：标志写在**不接受**它的命令上（老路径，退 1；Q18 / C2.8b / C2.8c）
// ---------------------------------------------------------------------------

// FailureFlagNotAccepted 造 F16 / F17 的失败：`--session` / `--name` 挂到了不接受它的命令上。
//
// 退出码是**老路径那一档**（1）：`tt debug` 组不补 `FlagErrorFunc`（Q18），
// cobra 拦下之后由根命令兜底 —— 与今天 `--bogus` 完全一致。
// 信封里仍然是 `code: USAGE`：Agent 读 `code`，不读数字（Q15 / C2.8c）。
//
// **既有失败**：`NewScenario` 是 false —— "打错标志"今天就有（`--bogus` 退 1），
// 这一档正是 Q18 有意保留的旧行为。（它也不该走到 `ExitFor`：这条路由 cobra 拦，
// 连 `dbgTargetError` 都进不来，所以 `Exit` 在这里写死。）
func FailureFlagNotAccepted(flag string) *Failure {
	f := &Failure{Code: CodeUsage, Exit: FlagOnWrongCommandExit}
	switch flag {
	case FlagName:
		f.Message = fmt.Sprintf("unknown flag: %s", FlagName)
		f.Hint = "--name 只在起会话的命令上有意义（tt debug start / tt debug wsdebug / tt debug start / tt debug exec）"
	default:
		// cobra 对未注册标志的原话是单横线的名字（`flag provided but not defined: -session`）。
		f.Message = "flag provided but not defined: -" + strings.TrimLeft(flag, "-")
		f.Hint = "该命令不接受 --session（要列全部会话用 tt debug status）"
	}
	return f
}

// ---------------------------------------------------------------------------
// F18 / F19 / F20：别名
// ---------------------------------------------------------------------------

// FailureNameTaken 是 F18：`--name` 的别名已被占用。
//
// 与 Q23（C1.6）的关系：`--name X` 指到的那条会话**空闲就复用**、正在跑/已停站报
// `SESSION_TARGET_BUSY`（F20），所以在"起会话"这条路径上 F18 已经不可达。
// 它仍然是契约 4.1.1 里的一行（**退 2**），留给**复用不了**的那些路径 ——
// 构造器在这里，取值与退出码由测试守住。
//
// **新场景**：`--name` 今天没被服务端接受（B20/C5.3 的新键还没落地），别名撞车今天
// 不可能发生 → 这一位是 true（无条件走 4.1.1）。注意它的码值本身是 2 不是 3：
// Q23 的**两个**别名码不是一个数 —— `SESSION_NAME_TAKEN` 归"用法错"那一行（2，
// 4.1.1 明写"退 2 不退 3"，早先把它与 3 并列是笔误），`SESSION_TARGET_BUSY` 才退 3。
func FailureNameTaken(name string, occupant Session) *Failure {
	f := newScenarioFailure(CodeNameTaken, "别名 %q 已被 %s 占用；换一个名字，或先结束那条会话",
		name, occupant.CanonicalRef())
	f.Candidates = candidatesOf([]Session{occupant})
	return f
}

// failureAliasInvalid 是 F19：别名取值非法（空 / 含 `/` 或 `#` / 超长）。
// **新场景**（`--name` 是新标志）→ 退 2。
func failureAliasInvalid() *Failure {
	return newScenarioFailure(CodeUsage, `别名不能为空、也不能含 "/" 或 "#"`)
}

// FailureTargetBusy 是 F20 / Q23：`--name` 指到的那条会话不能复用（正在跑 / 已停站）。
// **新场景**（同上，别名体系是新的）→ 无条件退 3。
func FailureTargetBusy(name string, occupant Session) *Failure {
	f := newScenarioFailure(CodeTargetBusy, "别名 %q 指向的会话 %s 当前是 %s，不能复用",
		name, occupant.CanonicalRef(), StateLabel(occupant.State))
	f.Hint = "换一个别名，或先停掉那条会话（tt debug stop）"
	f.Candidates = candidatesOf([]Session{occupant})
	return f
}

// NewFailure 按 4.1.1 的码造一条失败（`code` → `Exit` 同源，C4.1）。
//
// 它给**服务端**那一侧的调用方用（WP6：`internal/debug` 的会话准备要报 F1 的歧义，
// 而那个包不重写一句码表 / 退出码表）。判据与默认口径与包内的 `newFailure` 一致：
//
//   - `NewScenario` **不置位**（false）：调用方要"这是新场景"必须自己说清 ——
//     免得服务端拿一个默认的 true 把老路径的退出码档位悄悄改掉（Q15 的分档判据）；
//   - 退出码取自 `CodeExits`（表外的现行码归"运行失败"档）。
//
// **不造候选清单**：候选是"会话池里长什么样"的结论，只有拿得到池子的调用方拼得出来
// （C5.12 的九键只有一处定义：`Candidate` + `Session.Candidate()`）。
func NewFailure(code, format string, a ...any) *Failure {
	return newFailure(code, format, a...)
}

// ---------------------------------------------------------------------------
// 层级校验（契约第 7 节 / `tt-target-levels.md` (b) 的报错文案）
// ---------------------------------------------------------------------------

// failureLayerMismatch 是"这条命令要求 Lx 而目标只给到 Ly"。
//
// 码用 `USAGE`：4.1.1 是**唯一清单**，里面没有"层级不足"这一条 —— 不许为了这个另立新码。
//
// **既有失败**（`NewScenario` 是 false）：这就是"少给必填参数"，今天少给 `--env`
// 一样是失败退 1 —— 旧脚本看到的那一档要保护。（三个 `--session` / `--name`
// 专属的选择器错是另一族，它们由 newScenarioFailure 造，见上面 F9–F14。）
func failureLayerMismatch(cmdPath string, need, have Layer) *Failure {
	f := newFailure(CodeUsage, "%s 需要 %s（%s），这次只给到 %s",
		cmdPath, need, LayerNeed(need), have)
	switch need {
	case Layer1:
		f.Hint = "加 --env <环境>"
	case Layer2:
		f.Hint = "加 --env <环境> 与 --ent <企业编号>"
	case Layer3:
		f.Hint = "加 --session <选择器>，或先起一条会话（tt debug start <作业> --env <环境>）"
	default:
		f.Hint = "加 --env <环境>"
	}
	return f
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// candidatesOf 把一组会话投影成候选行（C5.12 的九键），
// 并按"序号 → 原始句柄"稳定排序 —— 同一个错误两次跑出的文案必须一样。
func candidatesOf(sessions []Session) []Candidate {
	out := make([]Candidate, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, s.Candidate())
	}
	sortCandidates(out)
	return out
}
