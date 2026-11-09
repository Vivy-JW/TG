package target

// 契约值的**守卫**（契约 5.1 / 5.2 / 4.1.1 / C5.1–C5.13 / Q15）。
//
// 为什么这里还要逐字钉一遍：**值是定义在 `contract.go` 里的生产常量**（唯一出处），
// 而这一条测试守住它 —— 常量名是给人读的、取值是给 Agent 读的，改错了必须有东西响。
// （WP2 的 `internal/debug/debugcontract_const_test.go` 已经从本包引用同一套值，
//   所以这里钉的是"唯一那份"，不是"第二份"。）

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// TestSessionEnvelopeKeyLiterals 逐字钉住 C5.1：`session` 块的**恒定键**（11 个）。
func TestSessionEnvelopeKeyLiterals(t *testing.T) {
	want := []string{"id", "ref", "display", "name", "env", "ent", "n", "kind", "prog", "state", "epoch"}
	assertSameSet(t, "C5.1 的 session 键集", SessionEnvelopeKeys, want)

	// C5.3 / Q22 的纯新增键必须是它的子集（契约测试用它判"旧键有没有被改名"）。
	if !isSubset(NewSessionKeys, SessionEnvelopeKeys) {
		t.Errorf("C5.3 的新增键 %v 不是 session 键集的子集", NewSessionKeys)
	}
	// 恒定键与 omitempty 键互斥（一个键只能属一边）。
	for _, k := range SessionOptionalKeys {
		if contains(SessionEnvelopeKeys, k) {
			t.Errorf("%q 同时出现在恒定键与 omitempty 键里 —— 一个键只能属一边", k)
		}
	}
	// 两者合起来必须正好是 SessionBrief 的 JSON 键（C5.2：键名一个都不许改）。
	// 27 = 26（原定）+ watchdogDisabled：2026-10-29 真机验证后新增的“这一轮不会
	// 自动放行”标记。按 C5.2 新增键必须显式登记（不能靠 omitempty 飘进去），
	// 所以这个数从 26 改成 27 是本契约的一次有记录的变更，不是测试漂移。
	if got := len(SessionEnvelopeKeys) + len(SessionOptionalKeys); got != 27 {
		t.Errorf("恒定键 + omitempty 键 = %d 个，应该是 27 个（C5.1/C5.2）", got)
	}
	// 顶层恒定键（5.1）：session 与 requested。
	assertSameSet(t, "5.1 的顶层恒定键", SessionTopKeys, []string{"session", "requested"})
	// 失败信封的四键恒定 + 两键按条件（C5.10）。
	assertSameSet(t, "C5.10 的失败信封恒定键", FailureRequiredKeys, []string{"ok", "code", "error", "exitCode"})
	assertSameSet(t, "C5.10/C5.11 的失败信封条件键", FailureOptionalKeys, []string{"hint", "candidates"})
	// 候选九键 —— **全契约的唯一出处**（C5.12）。
	assertSameSet(t, "C5.12 的候选键集", CandidateKeys,
		[]string{"id", "ref", "display", "name", "env", "ent", "n", "prog", "state"})
	if len(CandidateKeys) != 9 {
		t.Errorf("C5.12 写死候选是**九键**，实得 %d 个", len(CandidateKeys))
	}
	if !isSubset(CandidateKeys, SessionEnvelopeKeys) {
		t.Errorf("C5.4b：候选九键必须是 session 键集的**真子集**（同一个渲染函数两处都用）")
	}
}

// TestSessionViewTagsAreTheEnvelopeKeys 钉住 C5.1 + C5.2 + C5.5：
// 视图 `Session` 的 json 标签必须**恰好**是 `session` 块的键集减去 `kind`
// （`kind` 由 C5.5 定成常量 `ssh`，不由视图携带）；`SessionEnvelope` 再把两者合起来。
func TestSessionViewTagsAreTheEnvelopeKeys(t *testing.T) {
	tags := jsonTagsOf(t, reflect.TypeOf(Session{}))
	want := without(SessionEnvelopeKeys, "kind")
	assertSameSet(t, "视图 Session 的 json 标签", tags, want)

	// 合起来必须正好是 11 键（多一个少一个都不行）。
	assertSameSet(t, "SessionEnvelope 的键集（视图 + kind）", append(append([]string{}, tags...), "kind"), SessionEnvelopeKeys)

	// C5.5 / Q9：本契约只寻址 SSH 会话，所以 kind 恒为 "ssh"。
	if KindSSH != "ssh" {
		t.Errorf("C5.5：kind 的取值该是 \"ssh\"，实得 %q", KindSSH)
	}
	if got := NewSessionEnvelope(Session{}).Kind; got != KindSSH {
		t.Errorf("信封里的 kind 该恒为 %q，实得 %q", KindSSH, got)
	}

}

// TestCandidateTagsMatchTheCandidateKeys 钉住 C5.12 的唯一出处纪律：
// `Candidate` 这个类型的 json 标签必须与 `CandidateKeys` 逐字一致 ——
// 常量表给测试用、struct 标签给序列化用，两者必须是**同一套键**（否则就有两份互不知情的定义）。
func TestCandidateTagsMatchTheCandidateKeys(t *testing.T) {
	assertSameSet(t, "Candidate 的 json 标签", jsonTagsOf(t, reflect.TypeOf(Candidate{})), CandidateKeys)
}

// TestCodeLiteralsAndExits 逐字钉住 4.1.1 那张表（码 + 退出码）。
//
// 这是**对外契约**：C5.13 明写"加新码可以，改旧码的名字不行" —— 调用方靠它分支。
func TestCodeLiteralsAndExits(t *testing.T) {
	// 4.1.1 的"code"一列 + 退出码一列（新写法那一档）。
	want := map[string]int{
		"SESSION_AMBIGUOUS":   3,
		"SESSION_NOT_FOUND":   3,
		"SESSION_STALE":       3,
		"SESSION_TARGET_BUSY": 3,
		"NO_SESSION":          3,
		"ENV_UNKNOWN":         2,
		"USAGE":               2,
		"SESSION_NAME_TAKEN":  2,
		"SESSION_STATE":       4,
		// W1（工作站）：用途撞名 / 只读。两条都是 4.1.1 表里的新行，取值是**对外契约**。
		"SESSION_PURPOSE_MISMATCH": 2,
		"SESSION_READONLY":         4,
	}
	if len(CodeExits) != len(want) {
		t.Errorf("4.1.1 那张表有 %d 行，CodeExits 有 %d 行", len(want), len(CodeExits))
	}
	for literal, exit := range want {
		got, ok := CodeExits[literal]
		if !ok {
			t.Errorf("4.1.1 的码 %q 不在 CodeExits 里（Agent 靠它分支）", literal)
			continue
		}
		if got != exit {
			t.Errorf("码 %q 的退出码 = %d，4.1.1 写的是 %d", literal, got, exit)
		}
	}
	// 常量本身的取值也要钉（常量名是给人读的，取值是给 Agent 读的）。
	for constVal, literal := range map[string]string{
		CodeAmbiguous:  "SESSION_AMBIGUOUS",
		CodeNotFound:   "SESSION_NOT_FOUND",
		CodeStale:      "SESSION_STALE",
		CodeTargetBusy: "SESSION_TARGET_BUSY",
		CodeNameTaken:  "SESSION_NAME_TAKEN",
		CodeNoSession:  "NO_SESSION",
		CodeState:      "SESSION_STATE",
		CodeEnvUnknown: "ENV_UNKNOWN",
		CodeUsage:      "USAGE",

		CodePurposeMismatch: "SESSION_PURPOSE_MISMATCH",
		CodeReadOnly:        "SESSION_READONLY",
	} {
		if constVal != literal {
			t.Errorf("机器码常量的取值 = %q，4.1.1 / C5.13 写的是 %q（改旧码的名字不行）", constVal, literal)
		}
	}
	// 4.1 的其余档位：0 成功 / 1 运行失败 / 2 用法 / 3 没有会话 / 4 状态不符 / 5 目标不存在 / 6 体检不过。
	if ExitOK != 0 || ExitRunFailed != 1 || ExitUsage != 2 || ExitNoSession != 3 ||
		ExitState != 4 || ExitMissing != 5 || ExitPreflight != 6 {
		t.Errorf("4.1 的码表 = 0/1/2/3/4/5/6，常量是 %d/%d/%d/%d/%d/%d/%d",
			ExitOK, ExitRunFailed, ExitUsage, ExitNoSession, ExitState, ExitMissing, ExitPreflight)
	}
	// C2.8b / Q18：F16 / F17 走老路径 → 退 1（不是 2）。
	if FlagOnWrongCommandExit != ExitRunFailed {
		t.Errorf("C2.8b / Q18：标志挂错命令该退 %d，常量写的是 %d", ExitRunFailed, FlagOnWrongCommandExit)
	}
}

// TestExitForIsStagedByScenario 钉住 Q15「分档生效」的**判据**：
// 分档看的是"**这个场景是不是新场景**"，不是"命令行里有没有 `--session` 这个字"。
//
// 三条（本轮改正后的口径，见 contract.go 的 ExitFor）：
//
//  1. 新场景（`newScenario = true`）→ **无条件**走 4.1.1，哪怕一个标志都没打；
//  2. 既有场景 + 老写法 → 维持今天的码（1）—— 这是 H2 的落点；
//  3. 既有场景 + 真用了新写法 → 也走 4.1.1（4.2 的 H2 表逐行写着 F2 带 `--session` → 3、
//     F6 写在 `--session` 里 → 2）。
func TestExitForIsStagedByScenario(t *testing.T) {
	// 1. 新场景那一档：不管有没有打标志，都按 4.1.1 退。
	for code, exit := range CodeExits {
		if got := ExitFor(code, true, false); got != exit {
			t.Errorf("ExitFor(%q, 新场景, 没打标志) = %d，4.1.1 写的是 %d —— "+
				"新场景没有旧行为要保护，不许因为没打标志就降回 1", code, got, exit)
		}
		if got := ExitFor(code, true, true); got != exit {
			t.Errorf("ExitFor(%q, 新场景, 打了标志) = %d，4.1.1 写的是 %d", code, got, exit)
		}
	}
	// 判据改正的那一处，单独写死一遍（它就是本轮被打回的那条）：
	// 多会话 + 不给目标（F1）= 新场景 + 没打 `--session` → 3，不是 1。
	if got := ExitFor(CodeAmbiguous, true, false); got != ExitNoSession {
		t.Errorf("F1（多会话 + 不给目标、没打 --session）= %d，要的是 %d —— "+
			"判据是「这个场景今天存不存在」（今天不存在：它会静默取第一条并成功）", got, ExitNoSession)
	}

	// 2. 既有场景 + 老写法：不管码是什么，一律 1（这是守住 H2 的那一半）。
	for _, code := range append(append([]string{}, CandidatesRequiredCodes...),
		CodeNoSession, CodeEnvUnknown, CodeUsage, CodeState, "QUERY_FAILED") {
		if got := ExitFor(code, false, false); got != ExitRunFailed {
			t.Errorf("ExitFor(%q, 既有场景, 老写法) = %d，Q15 要的是 %d（老脚本今天看到的就是它）",
				code, got, ExitRunFailed)
		}
	}
	// F16 / F17 这一档（USAGE + 老路径）就是 1，这条是 C2.8c 那句"退出码分辨不了，靠 code"的落点。
	if got := ExitFor(CodeUsage, false, false); got != 1 {
		t.Errorf("F16/F17（标志挂错命令）该退 1，实得 %d", got)
	}
	// 3. 既有场景 + 新写法：F2 带 `--session` → 3、F6 写在 `--session` 里 → 2（4.2 的 H2 表）。
	if got := ExitFor(CodeNoSession, false, true); got != ExitNoSession {
		t.Errorf("F2 带 --session 该退 %d（4.2 的 H2 表），实得 %d", ExitNoSession, got)
	}
	if got := ExitFor(CodeEnvUnknown, false, true); got != ExitUsage {
		t.Errorf("F6 写在 --session 里该退 %d（4.2 的 H2 表），实得 %d", ExitUsage, got)
	}
	// 4.1.1 表外的**现行码**（QUERY_FAILED / CONNECT_FAILED / NOT_FOUND …）恒为"运行失败"档。
	if got := ExitFor("CONNECT_FAILED", true, true); got != ExitRunFailed {
		t.Errorf("4.1.1 表外的现行码该退 %d（运行失败档），实得 %d", ExitRunFailed, got)
	}
	for _, code := range []string{CodeNoSession, CodeEnvUnknown, CodeUsage, CodeState} {
		if _, ok := CodeExits[code]; !ok {
			t.Errorf("4.1.1 复用现行常量的那几个码（%s）也要在这张表里 —— 否则分档就没有依据", code)
		}
	}
	// 表里的码**自洽**：ExitFor 与 Failure 的 Exit 必须同源（C4.1）。
	for _, code := range CandidatesRequiredCodes {
		f := newFailure(code, "x")
		if f.Exit != ExitFor(code, true, false) {
			t.Errorf("码 %s：Failure.Exit = %d，ExitFor = %d —— C4.1 要求「码与退出码同源」",
				code, f.Exit, ExitFor(code, true, false))
		}
	}
}

// TestNewScenarioClassification 逐条钉住"**哪些失败是新场景**"这张判定表
// （本轮改正的判据就落在 `Failure.NewScenario` 这一位上）。
//
// 判据一句话：**这个场景今天会不会发生**。
//
//   - 今天压根不会发生的（F1 多会话不取第一项、选择器指名的东西不存在、句柄过期、
//     别名撞车、选择器文法非法…）→ 新场景 → 无条件按 4.1.1 退；
//   - 今天就有、今天就失败的（池里 0 条会话、环境名写错、少给必填参数、标志挂错命令）
//     → 既有失败 → 老写法维持 1（除非契约明文另有规定：F2 带 --session → 3、
//     F6 写在 --session 里 → 2）。
//
// 这条测试是"判据"本身的守卫：把某一族错标成既有（或反过来），数字当场就变。
func TestNewScenarioClassification(t *testing.T) {
	occupant := Session{Env: "恒烁正式区", N: 1, State: StateRunning}
	two := []Session{
		{Env: "恒烁正式区", N: 1, State: StateIdle},
		{Env: "恒烁正式区", N: 2, State: StateIdle},
	}
	cases := []struct {
		label string
		fail  *Failure
		// want 是**没打任何标志**时的退出码：新场景走 4.1.1，既有失败维持 1。
		want int
	}{
		// —— 新场景（今天不可能发生）——
		{"F1 多会话 + 不给目标", failureAmbiguousOmitted(two), ExitNoSession},
		{"F3/F4 选择器命中多条", failureAmbiguous("恒烁正式区", two), ExitNoSession},
		{"F5 别名不存在", failureAliasNotFound("汇率单X", two), ExitNoSession},
		{"F6 环境名写错（`--env` 写错，这一档今天就有）", failureEnvUnknown("无此环境", []string{"恒烁正式区"}), ExitRunFailed},
		{"F7 该环境下没有该作业", failureJobNotFound("恒烁正式区", "apmt520", two), ExitNoSession},
		{"F8 序号越界", failureSeqOutOfRange("恒烁正式区", 2, 7, two), ExitNoSession},
		{"F9 序号不是十进制正整数", failureSeqInvalid("x"), ExitUsage},
		{"F10/F11 选择器文法非法", failureSelectorGrammar(""), ExitUsage},
		{"F12 环境矛盾", failureEnvConflict("测试1012", "恒烁正式区"), ExitUsage},
		{"F13 企业矛盾", failureEntConflict("88", "99"), ExitUsage},
		{"F14 会话企业不符", failureSessionEntConflict("恒烁正式区/88", "88", "99"), ExitUsage},
		{"F15 句柄过期", failureStale(occupant, "1"), ExitNoSession},
		{"F18 别名已被占用", FailureNameTaken("汇率单", occupant), ExitUsage},
		{"F19 别名取值非法", failureAliasInvalid(), ExitUsage},
		{"F20 别名指向的会话不能复用", FailureTargetBusy("汇率单", occupant), ExitNoSession},
		// —— 既有失败（今天就有、今天就退 1）——
		{"F2 池里 0 条会话", failureNoSession(), ExitRunFailed},
		{"该环境下没有会话（池里有别的环境）", failureNoSessionInScope("恒烁正式区", two), ExitNoSession},
		{"F16/F17 标志挂到不接受它的命令上", FailureFlagNotAccepted(FlagSession), ExitRunFailed},
		{"层数不够（少给必填参数）", failureLayerMismatch("tt debug exec", Layer3, Layer1), ExitRunFailed},
	}
	for _, c := range cases {
		if c.fail == nil {
			t.Errorf("%s：构造器返回了 nil", c.label)
			continue
		}
		// 没打任何标志：新场景按 4.1.1 退，既有失败维持 1。
		if got := ExitFor(c.fail.Code, c.fail.NewScenario, false); got != c.want {
			t.Errorf("%s：码 %s、NewScenario=%v → 没打标志时退 %d，判定表要的是 %d\n"+
				"    判据是「这个场景今天会不会发生」：不会 → 新场景（无条件走 4.1.1）；\n"+
				"    会 → 既有失败（老写法维持 1，这是 H2 的落点）。",
				c.label, c.fail.Code, c.fail.NewScenario, got, c.want)
		}
	}
	// 反例的**另一半**：既有失败里唯一"契约明文另有规定"的那两处 —— 真用了新写法才升档。
	if got := ExitFor(CodeNoSession, false, true); got != ExitNoSession {
		t.Errorf("F2 带 --session 该升到 %d（4.2 的 H2 表逐行写着），实得 %d", ExitNoSession, got)
	}
	if got := ExitFor(CodeEnvUnknown, false, true); got != ExitUsage {
		t.Errorf("F6 写在 --session 里该升到 %d（4.2 的 H2 表逐行写着），实得 %d", ExitUsage, got)
	}
	// F1 是**新场景**，所以它的 3 与上面那两处"升档"无关：一个标志都不打也一样。
	if got := ExitFor(CodeAmbiguous, failureAmbiguousOmitted(two).NewScenario, false); got != ExitNoSession {
		t.Errorf("F1 没打标志时该退 %d，实得 %d —— 它是新场景，不靠 --session 才升档", ExitNoSession, got)
	}
}

// TestCandidatesRequiredCodes 钉住 C5.11：只有这五种码**必须**带候选清单。
func TestCandidatesRequiredCodes(t *testing.T) {
	want := []string{CodeAmbiguous, CodeNotFound, CodeStale, CodeNameTaken, CodeTargetBusy}
	assertSameSet(t, "C5.11 的五种码", CandidatesRequiredCodes, want)
	for _, c := range want {
		if !CodeRequiresCandidates(c) {
			t.Errorf("C5.11：%s 必须带候选清单", c)
		}
	}
	for _, c := range []string{CodeNoSession, CodeEnvUnknown, CodeUsage, CodeState, "QUERY_FAILED"} {
		if CodeRequiresCandidates(c) {
			t.Errorf("C5.11：%s 不在那五种码里，不该带候选清单", c)
		}
	}
}

// TestGrammarConstants 钉住 2.1.2 / 2.1.4 / C2.3 / C2.5 的那几个文法常量。
func TestGrammarConstants(t *testing.T) {
	if SeqSep != "#" || SelectorSep != "/" {
		t.Errorf("分隔符常量 = %q / %q，契约 2.1.2 / 2.1.4 写的是 \"#\" / \"/\"", SeqSep, SelectorSep)
	}
	// 规范 id 的文法：`环境#序号`，序号 >= 1 且不写前导零（C2.1.2 / F9）。
	if RefPattern != `^[^#/]+#[1-9][0-9]*$` {
		t.Errorf("规范 id 的正则 = %q，C2.1.2 的文法要求 `^[^#/]+#[1-9][0-9]*$`", RefPattern)
	}
	if AliasMinRunes != 1 || AliasMaxRunes != 64 || AliasForbid != "/#" {
		t.Errorf("别名纪律 = %d–%d 字符 / 排除 %q，C2.5 写的是 1–64 / \"/#\"",
			AliasMinRunes, AliasMaxRunes, AliasForbid)
	}
	if SecondSegEnt != "ent" || SecondSegJob != "job" {
		t.Errorf("第二段的两种形态 = %q / %q，C2.3 写的是 \"ent\" / \"job\"", SecondSegEnt, SecondSegJob)
	}
	if JobPattern != `^[A-Za-z][A-Za-z0-9_-]{0,63}$` {
		t.Errorf("作业名正则 = %q，B29 写的是 `^[A-Za-z][A-Za-z0-9_-]{0,63}$`", JobPattern)
	}
	// 状态取值（B30）。
	assertSameSet(t, "B30 的五个会话状态",
		[]string{StateIdle, StateLoading, StateStopped, StateRunning, StateExit},
		[]string{"idle", "loading", "stopped", "running", "exit"})
}

// TestInvalidFormTableIsComplete 钉住 WP2 那张 F1–F20 的表**仍然完整**：
// 它现在是本包的生产常量（唯一出处），`internal/debug` 的契约测试引用它 ——
// 少一条就是少验一种"说不清的错"。
func TestInvalidFormTableIsComplete(t *testing.T) {
	if len(InvalidForms) != 20 {
		t.Fatalf("非法写法表该有 20 条（F1–F20），实得 %d 条", len(InvalidForms))
	}
	for i, f := range InvalidForms {
		if want := "F" + itoa(i+1); f.Key != want {
			t.Errorf("第 %d 条该是 %s，表里写的是 %q（键必须连号）", i+1, want, f.Key)
		}
		if strings.TrimSpace(f.Write) == "" || strings.TrimSpace(f.Code) == "" {
			t.Errorf("%s 的输入描述或机器码空着 —— 它是将来照着敲的输入集", f.Key)
		}
		if _, ok := CodeExits[f.Code]; !ok {
			t.Errorf("%s 用的机器码 %q 不在 CodeExits（4.1.1 的唯一清单）里", f.Key, f.Code)
		}
		if CodeRequiresCandidates(f.Code) != f.Candidates {
			t.Errorf("%s：candidates=%v，而 C5.11 对 %s 的规定是 %v",
				f.Key, f.Candidates, f.Code, CodeRequiresCandidates(f.Code))
		}
	}
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// jsonTagsOf 收集一个 struct 的 json 键（嵌入字段递归展开，跳过 `-` 与空标签）。
func jsonTagsOf(t *testing.T, rt reflect.Type) []string {
	t.Helper()
	out := make([]string, 0, rt.NumField())
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if f.Anonymous {
			out = append(out, jsonTagsOf(t, f.Type)...)
			continue
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			t.Errorf("%s.%s 没有 json 标签 —— 它是对外键集的一部分，没有标签就会被按字段名输出",
				rt.Name(), f.Name)
			continue
		}
		out = append(out, tag)
	}
	return out
}

// assertSameSet 比较两个键集（忽略顺序，但数量必须相同）。
func assertSameSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	g := append([]string{}, got...)
	w := append([]string{}, want...)
	sort.Strings(g)
	sort.Strings(w)
	if !reflect.DeepEqual(g, w) {
		t.Errorf("%s = %v\n    期望 %v", label, g, w)
	}
}

// without 返回 keys 里去掉 drop 之后的那些。
func without(keys []string, drop string) []string {
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != drop {
			out = append(out, k)
		}
	}
	return out
}

// contains 判断 keys 里有没有 k。
func contains(keys []string, k string) bool {
	for _, x := range keys {
		if x == k {
			return true
		}
	}
	return false
}

// isSubset 判断 sub 的每个元素都在 all 里。
func isSubset(sub, all []string) bool {
	for _, k := range sub {
		if !contains(all, k) {
			return false
		}
	}
	return true
}

// itoa 是 strconv.Itoa 的薄封装（只为失败信息少引一个 import）。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
