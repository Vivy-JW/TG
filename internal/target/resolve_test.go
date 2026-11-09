package target

// 落位（契约 2.2 的写法 ①–⑫、2.4 的 F1–F8 / F12–F15 / F20、C2.10–C2.12、C8.1、Q23）。
//
// 每条测试的注释里写明它钉住的条款 / 决策编号；消息文案按 2.4 的"文案形状"断言
// （`错误：`那一段前缀由人话渲染加，见 envelope.go 的 Failure.HumanError）。

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 测试夹具（契约 2.2 的前置，逐字照它写）
// ---------------------------------------------------------------------------

// contractPool 是契约 2.2 的前置：环境 `恒烁正式区` / `测试1012` 已配置；
// `恒烁正式区` 下有一条 SSH 会话，作业 `apmt520`、ENT 99、序号 1、别名 `汇率单`。
func contractPool() Pool {
	return Pool{
		Environments: []string{"恒烁正式区", "测试1012"},
		Sessions: []Session{{
			ID: "s1000", Ref: "恒烁正式区#1", Display: "恒烁正式区/apmt520#1",
			Name: "汇率单", Env: "恒烁正式区", Ent: "99", N: 1,
			Prog: "apmt520", State: StateIdle, Epoch: "k1",
		}},
	}
}

// envPool 是"同环境两条会话"（F1 / F4 / F8 / C2.11 的输入）。
func envPool() Pool {
	p := contractPool()
	p.Sessions = append(p.Sessions, Session{
		ID: "s2000", Ref: "恒烁正式区#2", Display: "恒烁正式区/bsft001#2",
		Env: "恒烁正式区", Ent: "88", N: 2,
		Prog: "bsft001", State: StateStopped, Epoch: "k1",
	})
	return p
}

// ---------------------------------------------------------------------------
// 写法 ①（2.2 / C8.1）：省略目标 + 唯一就落
// ---------------------------------------------------------------------------

// TestResolveUniqueFallsThrough 钉住 2.2 的写法 ① 与 C8.1：
// 会话池里 **0 条或 1 条**时，"省略目标"照旧自动落位（Q2 的单会话不变）。
func TestResolveUniqueFallsThrough(t *testing.T) {
	got, f := Resolve(contractPool(), Request{})
	if f != nil {
		t.Fatalf("唯一一条会话时省略目标该落位（C8.1），却报 %s：%s", f.Code, f.Message)
	}
	if got.Ref != "恒烁正式区#1" {
		t.Errorf("落到 %q，期望 恒烁正式区#1", got.Ref)
	}
}

// TestResolveEnvIsTheEnvironmentAuthority 钉住 C3.1 + C3.10 的一条读法：
// `--env` 是**环境**的权威，所以给了它就把候选集收窄到该环境 ——
// 别的环境里还有几条会话，不影响"这个环境里唯一就落"。
//
// 这是本次落地的**一处判读**（契约没有逐字写"收窄"两个字）：依据是 C3.1
// （三者各管一层，`--env` 管环境）与 C3.10（环境仍是一律点名）。
func TestResolveEnvIsTheEnvironmentAuthority(t *testing.T) {
	p := contractPool()
	p.Sessions = append(p.Sessions, Session{
		ID: "s3000", Ref: "测试1012#1", Display: "测试1012/apmt520#1",
		Env: "测试1012", Ent: "99", N: 1, Prog: "apmt520", State: StateIdle, Epoch: "k1",
	})
	got, f := Resolve(p, Request{Env: "恒烁正式区"})
	if f != nil {
		t.Fatalf("`--env 恒烁正式区` 下只有一条会话，该落它（C3.1），却报 %s：%s", f.Code, f.Message)
	}
	if got.Env != "恒烁正式区" {
		t.Errorf("落到 %q 环境的会话，期望 恒烁正式区（--env 是环境的权威）", got.Env)
	}
}

// ---------------------------------------------------------------------------
// F1–F8：非法/歧义写法
// ---------------------------------------------------------------------------

// TestResolveF1AmbiguousWhenOmitted 钉住 F1：省略目标而服务端有 >=2 条 → 退 3、
// `SESSION_AMBIGUOUS`、**必须带候选清单**。
func TestResolveF1AmbiguousWhenOmitted(t *testing.T) {
	_, f := Resolve(envPool(), Request{})
	assertFailure(t, f, "F1", CodeAmbiguous, ExitNoSession, true)
	if !strings.Contains(f.Message, "有 2 条 SSH 会话") || !strings.Contains(f.Message, "无法确定你要哪一条") {
		t.Errorf("F1 的文案 = %q，形状是「有 <N> 条 SSH 会话，无法确定你要哪一条」", f.Message)
	}
	if strings.Contains(f.Message, "错误：") {
		t.Errorf("信封的 error 字段不带 `错误：` 前缀（契约 5.2 的示例逐字如此）：%q", f.Message)
	}
	if len(f.Candidates) != 2 {
		t.Errorf("F1 该列 2 条候选（C5.11），实得 %d 条", len(f.Candidates))
	}
}

// TestResolveF2NoSession 钉住 F2：会话池 0 条 → 退 3、`NO_SESSION`、**不带候选**。
// 表里写明"`--session` 省略或给了名字，服务端 0 条 SSH 会话"，所以两种都给一条。
func TestResolveF2NoSession(t *testing.T) {
	empty := Pool{Environments: []string{"恒烁正式区", "测试1012"}}
	_, f := Resolve(empty, Request{})
	assertFailure(t, f, "F2", CodeNoSession, ExitNoSession, false)
	if !strings.Contains(f.Message, "没有 SSH 会话") || !strings.Contains(f.Message, "tt debug start") {
		t.Errorf("F2 的文案 = %q，形状是「没有 SSH 会话：先运行 tt debug start <作业> --env <环境> 起一条」", f.Message)
	}

	sel := mustSelector(t, "汇率单")
	_, f2 := Resolve(empty, Request{Selector: &sel})
	assertFailure(t, f2, "F2（给了名字）", CodeNoSession, ExitNoSession, false)
}

// TestResolveF3AmbiguousEnvJob 钉住 F3：（环境, 作业）下 >=2 条 → 退 3 + 候选块。
func TestResolveF3AmbiguousEnvJob(t *testing.T) {
	p := envPool()
	p.Sessions[1].Prog = "apmt520" // 同一个作业的两条（序号不同）
	p.Sessions[1].Display = "恒烁正式区/apmt520#2"
	sel := mustSelector(t, "恒烁正式区/apmt520")
	_, f := Resolve(p, Request{Selector: &sel})
	assertFailure(t, f, "F3", CodeAmbiguous, ExitNoSession, true)
	if !strings.Contains(f.Message, "恒烁正式区/apmt520 命中 2 条 SSH 会话") {
		t.Errorf("F3 的文案 = %q，形状是「<选择器> 命中 <N> 条 SSH 会话，无法确定你要哪一条」", f.Message)
	}
}

// TestResolveF4AmbiguousEnv 钉住 F4：环境名下有 >=2 条 → 退 3 + 候选块。
func TestResolveF4AmbiguousEnv(t *testing.T) {
	sel := mustSelector(t, "恒烁正式区")
	_, f := Resolve(envPool(), Request{Selector: &sel})
	assertFailure(t, f, "F4", CodeAmbiguous, ExitNoSession, true)
	if !strings.Contains(f.Message, "恒烁正式区 命中 2 条 SSH 会话") {
		t.Errorf("F4 的文案 = %q（<选择器> 换成环境名）", f.Message)
	}
}

// TestResolveF5AliasNotFound 钉住 F5：别名不存在 → 退 3、`SESSION_NOT_FOUND`、
// 候选块**只列有名字的那些** + 那句"去掉 --session 可看全部"。
func TestResolveF5AliasNotFound(t *testing.T) {
	sel := mustSelector(t, "汇率单X")
	_, f := Resolve(envPool(), Request{Selector: &sel})
	assertFailure(t, f, "F5", CodeNotFound, ExitNoSession, true)
	if !strings.Contains(f.Message, `没有别名为 "汇率单X" 的 SSH 会话`) {
		t.Errorf("F5 的文案 = %q，形状是「没有别名为 \"汇率单X\" 的 SSH 会话」", f.Message)
	}
	if len(f.Candidates) != 1 {
		t.Fatalf("F5 的候选块该**只列有名字的那些**（2 条里只有 1 条有名字），实得 %d 条", len(f.Candidates))
	}
	if f.Candidates[0].Name != "汇率单" {
		t.Errorf("F5 的候选该是那条有名字的（汇率单），实得 %q", f.Candidates[0].Name)
	}
	if !strings.Contains(f.Hint, "tt debug status") {
		t.Errorf("F5 该提示「去掉 --session 可看全部（tt debug status）」，实得 hint = %q", f.Hint)
	}
}

// TestResolveF6EnvUnknown 钉住 F6：环境名不在配置里 → 退 **2**、`ENV_UNKNOWN`、
// 候选列是**环境名清单**（不是会话候选块）。
func TestResolveF6EnvUnknown(t *testing.T) {
	sel := mustSelector(t, "不存在的环境/apmt520")
	_, f := Resolve(contractPool(), Request{Selector: &sel})
	assertFailure(t, f, "F6", CodeEnvUnknown, ExitUsage, false)
	if !strings.Contains(f.Message, `环境 "不存在的环境" 不存在`) {
		t.Errorf("F6 的文案 = %q，形状是「环境 \"不存在的环境\" 不存在」", f.Message)
	}
	if !strings.Contains(f.Hint, "恒烁正式区") || !strings.Contains(f.Hint, "测试1012") {
		t.Errorf("F6 该在 hint 里给出**全部**环境名（可用环境：恒烁正式区、测试1012），实得 %q", f.Hint)
	}
	if len(f.Candidates) != 0 {
		t.Errorf("F6 的候选列是环境名清单、不是会话候选块（C5.11 里 ENV_UNKNOWN 不带 candidates），实得 %d 条会话候选",
			len(f.Candidates))
	}
}

// TestResolveF7JobNotFound 钉住 F7：该环境下该作业无会话 → 退 3 + 候选块（该环境下的全部会话）。
func TestResolveF7JobNotFound(t *testing.T) {
	sel := mustSelector(t, "恒烁正式区/不存在的作业")
	_, f := Resolve(envPool(), Request{Selector: &sel})
	assertFailure(t, f, "F7", CodeNotFound, ExitNoSession, true)
	if !strings.Contains(f.Message, `环境 "恒烁正式区" 下没有作业 "不存在的作业" 的 SSH 会话`) {
		t.Errorf("F7 的文案 = %q", f.Message)
	}
	if len(f.Candidates) != 2 {
		t.Errorf("F7 的候选块该列该环境下的全部会话（2 条），实得 %d 条", len(f.Candidates))
	}
}

// TestResolveF8SequenceOutOfRange 钉住 F8 + C2.11：
// `<M>` 是**同一环境下**的会话条数，不是"同一作业下的条数"；候选块列的就是这 M 条。
func TestResolveF8SequenceOutOfRange(t *testing.T) {
	sel := mustSelector(t, "恒烁正式区/apmt520#7")
	_, f := Resolve(envPool(), Request{Selector: &sel})
	assertFailure(t, f, "F8", CodeNotFound, ExitNoSession, true)
	if !strings.Contains(f.Message, "恒烁正式区/apmt520 只有 2 条 SSH 会话，没有 #7") {
		t.Errorf("F8 的文案 = %q，C2.11 定的是「<环境>/<作业> 只有 <M> 条 SSH 会话，没有 #7」，\n"+
			"    而 <M> 按**环境**算（该环境下共 2 条）", f.Message)
	}
	if len(f.Candidates) != 2 {
		t.Errorf("C2.11：F8 的候选块列的是该环境下的 2 条会话，实得 %d 条", len(f.Candidates))
	}

	// 规范 id 形态（写法 ④）越界也要报：没有作业段可写时，作用域就是环境名。
	selRef := mustSelector(t, "恒烁正式区#7")
	_, f2 := Resolve(envPool(), Request{Selector: &selRef})
	assertFailure(t, f2, "F8（规范 id 形态）", CodeNotFound, ExitNoSession, true)
	if !strings.Contains(f2.Message, "恒烁正式区 只有 2 条 SSH 会话，没有 #7") {
		t.Errorf("F8（写法 ④）的文案 = %q", f2.Message)
	}
}

// TestResolveC2_12JobSegmentMustMatch 钉住 C2.12：
// `环境/作业#序号` 里序号存在、但作业段与会话当前作业不符 → 按 **F7** 报，**不静默落**。
func TestResolveC2_12JobSegmentMustMatch(t *testing.T) {
	sel := mustSelector(t, "恒烁正式区/apmt520#1")
	p := contractPool()
	p.Sessions[0].Prog = "bsft001" // 会话中途换了作业（SetRun 那类改写）
	p.Sessions[0].Display = "恒烁正式区/bsft001#1"
	_, f := Resolve(p, Request{Selector: &sel})
	assertFailure(t, f, "C2.12", CodeNotFound, ExitNoSession, true)
	if !strings.Contains(f.Message, `没有作业 "apmt520" 的 SSH 会话`) {
		t.Errorf("C2.12 要按 F7 报「该环境该作业无会话」，实得 %q", f.Message)
	}
}

// ---------------------------------------------------------------------------
// F12–F15：标志冲突与世代失效
// ---------------------------------------------------------------------------

// TestResolveF12EnvConflict 钉住 F12 / C3.3-1：`--session` 里的环境与 `--env` 矛盾 → 退 2。
func TestResolveF12EnvConflict(t *testing.T) {
	sel := mustSelector(t, "测试1012/apmt520")
	_, f := Resolve(contractPool(), Request{Selector: &sel, Env: "恒烁正式区"})
	assertFailure(t, f, "F12", CodeUsage, ExitUsage, false)
	if !strings.Contains(f.Message, `--session 里的环境是 "测试1012"`) ||
		!strings.Contains(f.Message, `与 --env 给的 "恒烁正式区" 矛盾`) {
		t.Errorf("F12 的文案 = %q", f.Message)
	}
}

// TestResolveF13EntConflict 钉住 F13 / C3.3-2：`--session 环境/企业编号` 与 `--ent` 矛盾 → 退 2。
func TestResolveF13EntConflict(t *testing.T) {
	sel := mustSelector(t, "恒烁正式区/88")
	_, f := Resolve(contractPool(), Request{Selector: &sel, Ent: "99"})
	assertFailure(t, f, "F13", CodeUsage, ExitUsage, false)
	if !strings.Contains(f.Message, "--session 里的企业是 88") || !strings.Contains(f.Message, "与 --ent 给的 99 矛盾") {
		t.Errorf("F13 的文案 = %q", f.Message)
	}
}

// TestResolveF14SessionEntConflict 钉住 F14 / C3.3-3：落到的那条会话实际 ENT 与 `--ent` 不同 → 退 2。
func TestResolveF14SessionEntConflict(t *testing.T) {
	p := contractPool()
	p.Sessions[0].Ent = "88"
	sel := mustSelector(t, "恒烁正式区/apmt520")
	_, f := Resolve(p, Request{Selector: &sel, Ent: "99"})
	assertFailure(t, f, "F14", CodeUsage, ExitUsage, false)
	if !strings.Contains(f.Message, "的 SSH 会话企业是 88") || !strings.Contains(f.Message, "与 --ent 给的 99 矛盾") {
		t.Errorf("F14 的文案 = %q", f.Message)
	}
	// C2.9：企业矛盾只在**同一环境**下比 —— 环境不同时先撞 F12 那条环境矛盾。
	// （这里的"环境不同"是 `--session` 里的环境 与 `--env` 给的环境：测试1012 vs 恒烁正式区。）
	selOther := mustSelector(t, "测试1012/88")
	_, f12 := Resolve(p, Request{Selector: &selOther, Env: "恒烁正式区", Ent: "99"})
	if f12 == nil {
		t.Fatalf("C2.9：环境不同 + 企业不同时该报错（先撞环境那条），实得成功")
	}
	if !strings.Contains(f12.Message, "--session 里的环境是") {
		t.Errorf("C2.9：环境不同时该先报 F12 的环境矛盾，实得 %q", f12.Message)
	}
}

// TestResolveF15StaleEpoch 钉住 F15：环境与序号都对、但句柄是上一代 → 退 3、
// `SESSION_STALE`、候选块给**当前代**那一条。
func TestResolveF15StaleEpoch(t *testing.T) {
	p := contractPool()
	p.Sessions[0].Epoch = "k2" // 会话被重启/复用后当前是第 2 代
	sel := mustSelector(t, "恒烁正式区#1")
	_, f := Resolve(p, Request{Selector: &sel, Epoch: "k1"})
	assertFailure(t, f, "F15", CodeStale, ExitNoSession, true)
	if !strings.Contains(f.Message, "当前是第 k2 代") || !strings.Contains(f.Message, "你手上的是第 k1 代") {
		t.Errorf("F15 的文案 = %q，形状是「这条 SSH 会话已失效（<ref> 当前是第 <E> 代，你手上的是第 <E-1> 代）；重新 tt debug status 取一次」",
			f.Message)
	}
	if len(f.Candidates) != 1 || f.Candidates[0].Ref != "恒烁正式区#1" {
		t.Errorf("F15 的候选块该给当前代那一条（恒烁正式区#1），实得 %+v", f.Candidates)
	}
	// 世代对得上时不报（否则 F15 会把正常用法也打成失效）。
	if _, ok := Resolve(p, Request{Selector: &sel, Epoch: "k2"}); ok != nil {
		t.Errorf("世代一致时不该报失效：%s", ok.Message)
	}
}

// ---------------------------------------------------------------------------
// Q23 / C1.6：别名当幂等键
// ---------------------------------------------------------------------------

// TestResolveNameIdempotency 钉住 Q23 / C1.6 / C3.7 / C3.7b：
// 别名已存在且那条会话**空闲** → 复用；**正在跑 / 已停站** → `SESSION_TARGET_BUSY` 退 3；
// 名字不存在 → 起新会话（Free）。
func TestResolveNameIdempotency(t *testing.T) {
	p := contractPool() // 那一唯一会话的别名是 `汇率单`，状态 idle

	kind, got, f := ResolveName(p.Sessions, "汇率单")
	if f != nil {
		t.Fatalf("别名已存在且会话空闲时该**复用**（Q23），却报 %s：%s", f.Code, f.Message)
	}
	if kind != NameReuse || got.Ref != "恒烁正式区#1" {
		t.Errorf("Q23：空闲 → 复用那一条（恒烁正式区#1），实得 kind=%v ref=%q", kind, got.Ref)
	}

	kind, _, f = ResolveName(p.Sessions, "汇率单2")
	if f != nil || kind != NameFree {
		t.Errorf("别名不存在时该起新会话（Free），实得 kind=%v failure=%v", kind, f)
	}

	for _, st := range []string{StateRunning, StateStopped} {
		q := contractPool()
		q.Sessions[0].State = st
		kind, occupant, f := ResolveName(q.Sessions, "汇率单")
		assertFailure(t, f, "F20（"+st+"）", CodeTargetBusy, ExitNoSession, true)
		if kind != NameBusy || occupant.Ref != "恒烁正式区#1" {
			t.Errorf("F20：忙碌时该报那一条占用者，实得 kind=%v ref=%q", kind, occupant.Ref)
		}
		if !strings.Contains(f.Message, "不能复用") || !strings.Contains(f.Message, "汇率单") {
			t.Errorf("F20 的文案 = %q，形状是「别名 \"汇率单\" 指向的会话 <ref> 当前是 <状态>，不能复用」", f.Message)
		}
		if !strings.Contains(f.Hint, "tt debug stop") {
			t.Errorf("F20 该提示「换一个别名，或先停掉那条会话（tt debug stop）」，实得 %q", f.Hint)
		}
	}

	// F18 与 Q23 的关系：Q23 定稿之后，"别名已存在且空闲"是**复用**、不是撞车，
	// 所以 F18 的 `SESSION_NAME_TAKEN`（退 2）只在"复用不了"的那些路径上给 ——
	// 它的构造器单独钉在这里，取值与退出码必须与 4.1.1 一致。
	f18 := FailureNameTaken("汇率单", p.Sessions[0])
	assertFailure(t, f18, "F18", CodeNameTaken, ExitUsage, true)
	if !strings.Contains(f18.Message, "已被 恒烁正式区#1 占用") {
		t.Errorf("F18 的文案 = %q，形状是「别名 \"汇率单\" 已被 <ref> 占用；换一个名字，或先结束那条会话」", f18.Message)
	}
}

// ---------------------------------------------------------------------------
// F16 / F17：标志写在**不接受**它的命令上（老路径，退 1）
// ---------------------------------------------------------------------------

// TestFailureFlagNotAcceptedIsTheOldPath 钉住 F16 / F17 与 C2.8b / C2.8c / Q18：
// 这两条走 cobra 老路径 → **退 1**（`tt debug` 组不补 FlagErrorFunc），
// 但信封里仍然是 `code: USAGE`（Agent 读码，不读数字，Q15）。
func TestFailureFlagNotAcceptedIsTheOldPath(t *testing.T) {
	f16 := FailureFlagNotAccepted(FlagSession)
	assertFailure(t, f16, "F16", CodeUsage, FlagOnWrongCommandExit, false)
	if !strings.Contains(f16.Message, "-session") {
		t.Errorf("F16 的文案 = %q，形状是「flag provided but not defined: -session」", f16.Message)
	}
	if !strings.Contains(f16.Hint, "tt debug status") {
		t.Errorf("F16 该提示「该命令不接受 --session（要列全部会话用 tt debug status）」，实得 %q", f16.Hint)
	}
	f17 := FailureFlagNotAccepted(FlagName)
	assertFailure(t, f17, "F17", CodeUsage, FlagOnWrongCommandExit, false)
	if !strings.Contains(f17.Message, "unknown flag") {
		t.Errorf("F17 的文案 = %q，形状是「unknown flag: --name」", f17.Message)
	}
	if !strings.Contains(f17.Hint, "只在起会话的命令上有意义") {
		t.Errorf("F17 该提示「--name 只在起会话的命令上有意义（…）」（C3.5），实得 %q", f17.Hint)
	}
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// mustSelector 解析一个**必须合法**的选择器（夹具用；非法的那一半在上面的 F9–F11 里）。
func mustSelector(t *testing.T, raw string) Selector {
	t.Helper()
	sel, f := ParseSelector(raw)
	if f != nil {
		t.Fatalf("夹具里的选择器 %q 该合法，却报 %s：%s", raw, f.Code, f.Message)
	}
	return sel
}

// assertFailure 把 2.4 那一行（码 / 退出码 / 候选列）三项一次钉下来。
func assertFailure(t *testing.T, f *Failure, label, code string, exit int, candidates bool) {
	t.Helper()
	if f == nil {
		t.Fatalf("%s 该报错，实得成功", label)
	}
	if f.Code != code {
		t.Errorf("%s 的机器码 = %q，契约 2.4 写的是 %q（4.1.1 是唯一清单）", label, f.Code, code)
	}
	if f.Exit != exit {
		t.Errorf("%s 的退出码 = %d，契约 2.4 写的是 %d", label, f.Exit, exit)
	}
	if CodeRequiresCandidates(code) && !candidates {
		t.Errorf("%s 用了必须带候选的码 %s，测试却期望不带 —— 夹具写错了", label, code)
	}
	if candidates && CodeRequiresCandidates(code) {
		if f.Envelope().Candidates == nil {
			t.Errorf("%s（%s）的失败信封**必须出现** candidates 键（C5.11）", label, code)
		}
	}
	if !candidates {
		if len(f.Candidates) != 0 {
			t.Errorf("%s 的候选列是「不出现」，实得 %d 条", label, len(f.Candidates))
		}
	}
	if f.Exit != FlagOnWrongCommandExit && f.Exit != ExitUsage && f.Exit != ExitNoSession {
		t.Errorf("%s 的退出码 %d 不在契约 4.1 的码表里", label, f.Exit)
	}
}
