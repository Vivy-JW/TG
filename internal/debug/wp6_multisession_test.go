package debug

// WP6「放开多会话」的判据测试。
//
// # 它替掉的那个假设
//
// `prepareSession` 从前是"取池里唯一那条 live"（`var live *Session` + `break`）——
// 计划开头 A–E 表里的一类"假设只有一条会话"。本批把它改成"**按目标在池里找**"：
//
//	候选 0 条 → 起新会话；候选 1 条 → 按 decideHost 处置；候选 ≥2 条 → **报歧义**
//	（`SESSION_AMBIGUOUS` + **退 3** + C5.12 的候选九键，**绝不挑一条**）。
//
// “候选”= **与本次目标同环境**（`sameTarget`：连接四元组 + 环境名）的活会话；
// 点名叫了 `--name X` 时再按 `nameCompatible` 收窄（Q23 / C2.5b）。
//
// # 钉住的条款
//
//   - **Q2 / C8.1「只在多会话时收紧」**：0 条或 1 条时"省略目标"照旧自动落位；
//     ≥2 条按 F1 报错并列候选；
//   - **F1**（省略目标 + 多条 → 退 3 + `SESSION_AMBIGUOUS` + 候选）、
//     **F15**（复用宿主换代 → 旧句柄 `SESSION_STALE`）、**F20**（别名指向的会话不可复用）；
//   - **C5.11 / C5.12**（候选在那五种码下必须出现；候选就是那九键）；
//   - **C3.7b / Q19**（会话从池里消失 ⇒ 别名与序号当场释放）。
//
// # 为什么能这么测（不拨真机）
//
// 三条分支里只有"起新会话"要拨 SSH，而它**必然失败**（`h1` 这个主机名解不出来）——
// 那正是这一支的判据：报错文案里带 `SSH 连接失败` 就说明走到了 `NewSession`。
// 复用/收口/歧义三支都在**拨号之前**判完，所以纯内存可测（真机那一半见
// `internal/debug/README.md` 的"故意不覆盖"）。

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"tt/internal/target"
)

// ---------------------------------------------------------------------------
// 三条分支：候选 0 / 1 / ≥2
// ---------------------------------------------------------------------------

// 候选 0 条 → 起新会话（本批 §1 的第一档）。
//
// 判据分两半：① 真的走了 `NewSession`（拨号失败那句）；② **失败时池子一个字都不动**
// —— 空的宿主照旧留着（它在 idle，随时可用），不该因为一次失败的启动被顺手收掉。
func TestWP6PrepareSessionZeroCandidatesCreatesNew(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)

	_, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
	if err == nil {
		t.Fatal("池空时该尝试新建（会真拨 SSH，这里必然失败），却返回成功")
	}
	if !strings.Contains(err.Error(), "SSH 连接失败") {
		t.Fatalf("候选 0 条该走到 NewSession（于是报的是 SSH 连接失败），实得 %q", err.Error())
	}
	if n := len(m.Snapshot()); n != 0 {
		t.Errorf("新建失败后池里不该留下会话，实得 %d 条", n)
	}
}

// 候选 1 条 → 按 decideHost 处置（本批 §1 的第二档）：空闲同目标 ⇒ 复用宿主。
//
// 这一档是 H2 的落点（单会话行为逐字不变），所以它必须与今天同解。
func TestWP6PrepareSessionOneCandidateReuses(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	live := idleOn("s1", cfg)
	m.mu.Lock()
	m.sessions["s1"] = live
	m.mu.Unlock()

	got, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
	if err != nil {
		t.Fatalf("同目标空闲该复用（不拨号），实得 %v", err)
	}
	if got != live {
		t.Fatalf("该复用 s1，实得 %v", got.ID)
	}
}

// 候选 ≥2 条 → **报歧义**（F1）：`SESSION_AMBIGUOUS`、**退 3**、带**九键**候选清单，
// 而且**一条都不许动**（不挑、不收）。
//
// 逐条钉住契约的四处：码（C5.13 / 4.1.1）、退出码（F1 那一档 = 3，**无条件** ——
// 判据是"这个场景今天存不存在"，不是"命令行里有没有 --session"，见 Q15 与本轮改正）、
// 候选清单（C5.11 必须出现 + C5.12 的九键 + C5.14 的稳定顺序）、池子不变。
func TestWP6PrepareSessionTwoCandidatesReportsAmbiguity(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	// 顺序刻意乱着放：候选块必须按 id 排（`m.sessions` 是 map，顺序随机 —— 同一个失败
	// 两次跑出的文案必须一样，C5.14）。
	for _, s := range []*Session{idleOn("sB", cfg), idleOn("sA", cfg)} {
		m.mu.Lock()
		// 身份要真的登记过（序号/别名/世代）：候选行的九键里就有 ref 与 n（C5.12）。
		if err := m.assignIdentityLocked(s, ""); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记身份失败：%v", err)
		}
		m.sessions[s.ID] = s
		m.mu.Unlock()
	}

	got, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", "")
	if err == nil {
		t.Fatalf("同环境两条候选该报歧义（C8.1 / Q2），却落到了 %v —— 绝不挑一条", got.ID)
	}
	var f *target.Failure
	if !errors.As(err, &f) {
		t.Fatalf("要的是目标层的失败（带 code 与候选），实得 %T: %v", err, err)
	}
	if f.Code != target.CodeAmbiguous {
		t.Errorf("code = %q，F1 要求 %q（Agent 读 code 不读数字，Q15）", f.Code, target.CodeAmbiguous)
	}
	// 退出码：F1 那一档（3），而且**无条件** —— 这个场景（多会话 + 不给目标）
	// 今天不存在（今天只有一条会话），H2 射程够不到它。
	if got := target.ExitFor(f.Code, f.NewScenario, false); got != 3 {
		t.Errorf("F1 的退出码 = %d，该是 3（契约 2.4 / Q15：新场景无条件按码表退）", got)
	}
	if !f.NewScenario {
		t.Error("F1 必须登记成新场景（`target.Resolve` 的 failureAmbiguousOmitted 负责这件事）")
	}
	if len(f.Candidates) != 2 {
		t.Fatalf("F1 必须带候选清单（C5.11），实得 %d 条", len(f.Candidates))
	}
	// 顺序按"序号 → 原始句柄"稳定排序（`internal/target` 的 `sortCandidates`，
	// 唯一出处）—— 判据写成**不依赖插入顺序**的形状：序号必须升序，同号时句柄升序。
	// 这比断言"哪条在前"更结实：它同时钉住了"同一个失败两次跑出的顺序一样"（C5.14）。
	if f.Candidates[0].N > f.Candidates[1].N ||
		(f.Candidates[0].N == f.Candidates[1].N && f.Candidates[0].ID > f.Candidates[1].ID) {
		t.Errorf("候选该按「序号 → 原始句柄」稳定排序，实得 [%s(n=%d) %s(n=%d)]",
			f.Candidates[0].ID, f.Candidates[0].N, f.Candidates[1].ID, f.Candidates[1].N)
	}
	// 九键：候选行的键集唯一出处是 C5.12（`internal/target` 的 Candidate）。
	for _, c := range f.Candidates {
		if c.Ref == "" {
			t.Errorf("候选 %s 的 ref 是空的 —— C5.12 的九键里它是最重要的一把地址（C2.0）", c.ID)
		}
		if c.Env == "" || c.N == 0 {
			t.Errorf("候选 %s 缺 env/n（九键）：%+v", c.ID, c)
		}
	}
	if n := len(m.Snapshot()); n != 2 {
		t.Errorf("报歧义时不许改动会话池（不挑、不收），实得 %d 条", n)
	}
}

// 别名是**精确通道**（Q23 / C2.5b）：要 X 而池里那条叫 X ⇒ 只针对它判（不是歧义）；
// 池里那条叫 Y ⇒ **不复用它**（`nameCompatible` 的第二半），走"收口重建"那条路。
//
// 这一条解释"候选 ≥2"为什么不会把 `--name` 的正常用法也判成歧义 ——
// 名字一旦点名，候选就被收窄到"叫这个名字的那一条"（或"还没名字的宿主"）。
func TestWP6NamedTargetNarrowsCandidates(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	named := idleOn("s1", cfg)
	m.mu.Lock()
	if err := m.assignIdentityLocked(named, "复现-1"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	other := idleOn("s2", cfg)
	if err := m.assignIdentityLocked(other, "别的名字"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"], m.sessions["s2"] = named, other
	m.mu.Unlock()

	// 同环境两条，但点名了其中一条的名字 ⇒ 候选只有它一条 ⇒ 复用，不是歧义。
	got, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "复现-1")
	if err != nil {
		t.Fatalf("点名了别名时该落在那一条上（不是歧义），实得 %v", err)
	}
	if got != named {
		t.Fatalf("该复用 %s，实得 %v", named.ID, got.ID)
	}
	// 反向：点名一个**还不存在**的别名 + 池里有两条 ⇒ 也报歧义
	// （不是"随便挑一条空闲宿主给它起名"，见 sessionCandidates 的判据）。
	if _, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "新名字"); err == nil {
		t.Fatal("池里两条同环境会话时，给一条会话起个新名字也该报歧义（那是替调用方挑一条）")
	}
}

// ---------------------------------------------------------------------------
// Q2 / C8.1：只在多会话时收紧（单会话放行 / 多会话拒绝）
// ---------------------------------------------------------------------------

// 这条判据**两半各一条断言**，就是 Q2 的全文：单会话放行、多会话拒绝。
func TestWP6C8_1SingleSessionAllowsOmittedTarget(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	live := idleOn("s1", cfg)
	m.mu.Lock()
	m.sessions["s1"] = live
	m.mu.Unlock()

	// 不带任何目标（name 为空、也没点名）：**照旧成功**，而且落的就是那一条。
	got, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
	if err != nil {
		t.Fatalf("C8.1：池里只有一条时「省略目标」必须照旧自动落位，实得 %v", err)
	}
	if got != live {
		t.Fatalf("该落到唯一那条 s1，实得 %v", got.ID)
	}
}

func TestWP6C8_1MultiSessionRejectsOmittedTarget(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	for _, s := range []*Session{idleOn("sA", cfg), idleOn("sB", cfg)} {
		m.mu.Lock()
		m.sessions[s.ID] = s
		m.mu.Unlock()
	}

	_, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
	if err == nil {
		t.Fatal("C8.1：池里有两条时「省略目标」必须报歧义（不许挑一条）")
	}
	var f *target.Failure
	if !errors.As(err, &f) || f.Code != target.CodeAmbiguous {
		t.Fatalf("该报 %s，实得 %T: %v", target.CodeAmbiguous, err, err)
	}
	if len(f.Candidates) != 2 {
		t.Fatalf("多会话拒绝时必须带候选（C5.11），实得 %d 条", len(f.Candidates))
	}
}

// 另一条同族判据：**不同环境**的两条会话彼此不算候选。
//
// 这是 `sameTarget` 那套判据（连接四元组 + 环境名）保住语义的落点 ——
// 判错就是上错号（同机多环境时按企业编号取数的查询全返回 0 行，结论完全反向）。
func TestWP6CandidatesAreScopedToTheTargetEnvironment(t *testing.T) {
	here, there := envCfg("恒烁正式区"), envCfg("测试1012")
	m := NewManager(here)
	a := idleOn("sHere", here)
	b := idleOn("sThere", there) // 同机同区域、**只有环境名不同**
	m.mu.Lock()
	m.sessions["sHere"], m.sessions["sThere"] = a, b
	m.mu.Unlock()

	got, err := m.prepareSession(here, "m1", "p1", "", "", "", "", "")
	if err != nil {
		t.Fatalf("另一个环境的那条不是候选：本环境只有一条，该复用而不是报歧义，实得 %v", err)
	}
	if got != a {
		t.Fatalf("该复用它自己环境里的 %s，实得 %v", a.ID, got.ID)
	}
}

// ---------------------------------------------------------------------------
// F15：复用路径产生新一代（旧句柄当场失效）
// ---------------------------------------------------------------------------

// 复用宿主跑新一轮 ⇒ 换代（`SetRun`），于是**手上那个旧 epoch 落位就报 `SESSION_STALE`**。
//
// 判据走**生产链路**（`prepareSession` → `SetRun` → `Snapshot` → `target.Resolve`），
// 不拼假数据：只有服务端真的换了代，F15 才判得出来（C5.3 / Q11 的 F15 行）。
func TestWP6ReuseProducesANewEpochAndOldHandleGoesStale(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	live := idleOn("s1", cfg)
	m.mu.Lock()
	if err := m.assignIdentityLocked(live, ""); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"] = live
	m.mu.Unlock()

	old := m.Snapshot()[0].Epoch
	if strings.TrimSpace(old) == "" {
		t.Fatal("C5.3：登记时必发一代，实得空串")
	}
	if _, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", ""); err != nil {
		t.Fatalf("复用空闲宿主该成功（不拨号），实得 %v", err)
	}
	after := m.Snapshot()
	if len(after) != 1 || after[0].ID != "s1" {
		t.Fatalf("复用不该换会话，实得 %+v", after)
	}
	if after[0].Epoch == old {
		t.Fatalf("复用宿主跑新一轮之后 epoch 没换（%q）—— F15 的「旧句柄当场失效」判不出来", old)
	}

	// F15：拿旧 epoch 落位 ⇒ SESSION_STALE + 退 3 + 当前代那一条候选。
	pool := target.Pool{Environments: []string{cfg.EnvName()}, Sessions: sessionViews(after)}
	sel := selectorOf(t, target.RefOf(cfg.EnvName(), after[0].N))
	_, f := target.Resolve(pool, target.Request{Selector: sel, Epoch: old})
	if f == nil {
		t.Fatal("拿上一代的 epoch 落位该被判失效（F15），却落上了")
	}
	if f.Code != target.CodeStale {
		t.Errorf("F15 的码该是 %q，实得 %q（%s）", target.CodeStale, f.Code, f.Message)
	}
	if got := target.ExitFor(f.Code, f.NewScenario, false); got != 3 {
		t.Errorf("F15 的退出码 = %d，契约 2.4 写的是 3", got)
	}
	if len(f.Candidates) != 1 {
		t.Errorf("F15 必须带当前代那一条候选（C5.11），实得 %d 条", len(f.Candidates))
	}
	// 反向：当前代照旧落得上（不是"凡带 epoch 就拒"）。
	sel2 := selectorOf(t, target.RefOf(cfg.EnvName(), after[0].N))
	if _, f2 := target.Resolve(pool, target.Request{Selector: sel2, Epoch: after[0].Epoch}); f2 != nil {
		t.Errorf("当前代的 epoch 该正常落位，实得 %v", f2.Message)
	}
}

// ---------------------------------------------------------------------------
// 审计行：建立 / 复用 / 收口 / 歧义拒绝（WP6 的验收项）
// ---------------------------------------------------------------------------

// 会话操作必须留审计行（谁 / 哪条 / 干了什么），形状取 `internal/target` 的
// `AuditEntry`（一行一条 JSON、时间 RFC3339、`ref` 是**规范 id**：C5.4）。
//
// 落点：`<DataDir>/audit/sessions.jsonl`（见 audit.go）。
func TestWP6AuditLinesCoverCreateReuseAndAmbiguity(t *testing.T) {
	cfg := cfgFor("h1", "36")
	cfg.DataDir = t.TempDir()
	m := NewManager(cfg)
	m.auditActor = func(string) string { return "agent" } // 注入身份，不必改进程环境

	// ① 建立：登记身份那一刻留一行（四条造会话的路径都过 assignIdentityLocked）。
	live := idleOn("s1", cfg)
	m.mu.Lock()
	if err := m.assignIdentityLocked(live, "复现-1"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"] = live
	m.mu.Unlock()

	// ② 复用：复用宿主跑新一轮。
	if _, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "复现-1"); err != nil {
		t.Fatalf("复用该成功，实得 %v", err)
	}

	// ④ 歧义拒绝：再放一条同环境的会话，然后不带目标地起一轮。
	second := idleOn("s2", cfg)
	m.mu.Lock()
	if err := m.assignIdentityLocked(second, "另一个"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s2"] = second
	m.mu.Unlock()
	if _, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", ""); err == nil {
		t.Fatal("前置不成立：同环境两条该报歧义")
	}

	lines := auditLines(m.auditPath)
	if len(lines) < 4 {
		t.Fatalf("审计该至少有 建立/复用/建立/歧义 四行，实得 %d 行：%v", len(lines), lines)
	}
	seen := map[string]int{}
	for _, ln := range lines {
		e, err := auditEntryOf(ln)
		if err != nil {
			t.Fatalf("审计行不是合法 JSON（形状取 internal/target 的 AuditEntry）：%v\n%s", err, ln)
		}
		if e.Actor != "agent" {
			t.Errorf("审计行的「谁」= %q，该是注入的 agent（缺省来源是 TT_ACTOR）", e.Actor)
		}
		if strings.TrimSpace(e.Ref) == "" {
			t.Errorf("审计行的「哪条」是空的 —— 它写的是规范 id（C5.4 / C2.0）")
		}
		if strings.Contains(e.Ref, "/") {
			t.Errorf("审计行的 ref = %q 含 `/`：那是 display（作业段会漂，C5.4）", e.Ref)
		}
		if e.Time.IsZero() {
			t.Errorf("审计行的时间是零值（该补当前时间）：%s", ln)
		}
		seen[e.Action]++
	}
	for _, want := range []string{auditCreate, auditReuse, auditAmbiguous} {
		if seen[want] == 0 {
			t.Errorf("审计里没有 %q 这一行 —— WP6 要求 建立/复用/收口/歧义拒绝 至少这几处都有\n"+
				"    实得：%v", want, seen)
		}
	}
	if seen[auditAmbiguous] != 2 {
		t.Errorf("歧义拒绝时**每条候选**都该留一行（谁被卷进这次歧义要查得到），实得 %d 行", seen[auditAmbiguous])
	}
	// 谁 / 哪条 / 干了什么 三要素都要在行里（形状逐字是 AuditEntry 的 JSON）。
	first := lines[0]
	for _, k := range []string{`"time"`, `"actor"`, `"ref"`, `"action"`} {
		if !strings.Contains(first, k) {
			t.Errorf("审计行少了 %s 这个键：%s", k, first)
		}
	}
}

// 审计的**收口**那一行：换目标时收掉旧会话 ⇒ 一条 `session.collect`。
//
// 收口走 `Close()`（释放 SSH），所以这里只能测到"判据走对了"这一层：
// 收口是**新会话建成之后**才做的（本批改的时机），于是拨号失败时一条都不该动 ——
// 那正是这一条断言。
func TestWP6CollectHappensOnlyAfterTheNewSessionIsUp(t *testing.T) {
	here, there := envCfg("恒烁正式区"), envCfg("测试1012")
	here.DataDir = t.TempDir()
	m := NewManager(here)
	m.auditActor = func(string) string { return "agent" }
	old := idleOn("sOld", there) // 另一个环境：换目标 ⇒ hostCollect
	m.mu.Lock()
	if err := m.assignIdentityLocked(old, "旧的"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["sOld"] = old
	m.mu.Unlock()

	// 新环境必然拨不通（`h1` 解不出来）⇒ 新会话建不成 ⇒ 旧会话**一个字都不许动**。
	if _, err := m.prepareSession(here, "m1", "p1", "", "", "", "", ""); err == nil {
		t.Fatal("前置不成立：拨不通的目标该失败")
	}
	if m.Get("sOld") == nil {
		t.Fatal("新会话没建成时**不许**先收掉旧会话（那样失败路径上会少一条会话）——\n" +
			"    本批把收口挪到 NewSession 成功之后，就是为这一条")
	}
	for _, ln := range auditLines(m.auditPath) {
		if e, err := auditEntryOf(ln); err == nil && e.Action == auditCollect {
			t.Errorf("新会话没建成却记了收口审计行：%s", ln)
		}
	}
}

// ---------------------------------------------------------------------------
// Q11：事件按会话筛（发送那头筛）—— 含 hWait 那处 E 类修正
// ---------------------------------------------------------------------------

// `SubscribeScoped` 的判据：**范围在发送那头筛**（Q11 / C6.4），
// 而且无会话事件只发给"全部"订阅者（C6.5）。
func TestWP6SubscribeScopedFiltersOnTheSendingSide(t *testing.T) {
	m := NewManager(&Config{})
	chA, cancelA := m.SubscribeScoped("ws", "sA")
	defer cancelA()
	chB, cancelB := m.SubscribeScoped("ws", "sB")
	defer cancelB()
	chAll, _, cancelAll := m.SubscribeWithReplayScoped("ws", "", 10)
	defer cancelAll()

	m.emit(Event{Type: "log", SessionID: "sA", Text: "给 A"})
	m.emit(Event{Type: "log", SessionID: "sB", Text: "给 B"})
	m.emit(Event{Type: "log", Text: "无会话（服务级）"}) // SessionID 空

	// A 只该拿到"给 A"；B 只该拿到"给 B"；无会话事件谁都拿不到（除了 all）。
	wantA := []string{"给 A"}
	wantB := []string{"给 B"}
	wantAll := []string{"给 A", "给 B", "无会话（服务级）"}
	if got := drainTexts(chA); !sameStrings(got, wantA) {
		t.Errorf("订阅 sA 收到 %v，期望 %v —— 筛必须发生在**发送那头**（Q11 / C6.4）", got, wantA)
	}
	if got := drainTexts(chB); !sameStrings(got, wantB) {
		t.Errorf("订阅 sB 收到 %v，期望 %v", got, wantB)
	}
	if got := drainTexts(chAll); !sameStrings(got, wantAll) {
		t.Errorf("全部订阅者收到 %v，期望 %v（无会话事件只发给它，C6.5）", got, wantAll)
	}
}

// 补发也按范围过滤（C6.6），而且**配额不被别的会话吃掉**：过滤发生在取切片之前。
//
// 反例（错的实现）：先取最近 n 条再过滤 —— 单会话订阅者会拿到少于 n 条的历史。
func TestWP6ScopedReplayFiltersBeforeTakingTheSlice(t *testing.T) {
	m := NewManager(&Config{})
	// 别的会话刷 5 条，然后自己要的那条来 2 条 —— 若实现写成"先取 2 条再过滤"，
	// 订阅者就只剩 1 条（另外那条被别人的事件顶出窗口）。
	for i := 0; i < 5; i++ {
		m.emit(Event{Type: "log", SessionID: "sOther", Text: "噪声"})
	}
	for i := 0; i < 2; i++ {
		m.emit(Event{Type: "log", SessionID: "sA", Text: "给 A"})
	}

	_, replay, cancel := m.SubscribeWithReplayScoped("ws", "sA", 2)
	defer cancel()
	if len(replay) != 2 {
		t.Fatalf("补发该给 2 条（C6.6：补发先按范围过滤），实得 %d 条：%+v", len(replay), replay)
	}
	for _, ev := range replay {
		if ev.SessionID != "sA" {
			t.Errorf("补发里混进了别条会话的事件：%+v", ev)
		}
	}
	// Seq 空间不分叉（C6.7）：过滤只影响"给不给"，不影响序号本身。
	if replay[0].Seq >= replay[1].Seq {
		t.Errorf("补发里两条的 Seq 该升序（%d, %d）", replay[0].Seq, replay[1].Seq)
	}
	// 反向：all 的补发照旧拿全部（逐字等于 SubscribeWithReplay 今天的行为）。
	_, allReplay, cancelAll := m.SubscribeWithReplayScoped("ws", "", 3)
	defer cancelAll()
	if len(allReplay) != 3 {
		t.Fatalf("全部订阅者的补发该是 3 条，实得 %d", len(allReplay))
	}
}

// `Subscribe` / `SubscribeWithReplay` **一个字都不改**（Q22 / C6.7 的"纯新增"）：
// 它们等价于"全部"那个范围 —— 既有调用方照旧拿到全部事件。
func TestWP6UnscopedSubscribeStaysAllEvents(t *testing.T) {
	m := NewManager(&Config{})
	// 先攒两条历史（补发读的是环形缓冲），再订阅 —— 订阅之后发的进 channel。
	m.emit(Event{Type: "log", SessionID: "sA", Text: "A"})
	m.emit(Event{Type: "log", SessionID: "sB", Text: "B"})
	ch, replay, cancel := m.SubscribeWithReplay("t", 10)
	defer cancel()
	m.emit(Event{Type: "log", Text: "服务级"})

	got := drainTexts(ch)
	want := []string{"服务级"}
	if !sameStrings(got, want) {
		t.Errorf("不带范围的订阅该照旧拿全部事件（Q22 纯新增），实得 %v", got)
	}
	if len(replay) != 2 {
		t.Fatalf("不带范围的补发该照旧拿全部历史，实得 %d 条", len(replay))
	}
	if replay[0].Text != "A" || replay[1].Text != "B" {
		t.Errorf("补发该按时间升序给全部（含别的会话的），实得 %q / %q", replay[0].Text, replay[1].Text)
	}
}

// `hWait` 那处 E 类写法（契约 C7.2-2）修正的判据：
// **它必须按会话订阅**，而不是订阅全部再自己丢。
//
// 判据落在**订阅登记**这一层（发送那头筛的证据在这里，而不是"客户端有没有写 if"）：
// `hWait` 订阅之后，`m.subs` 里那一条的范围必须等于 URL 里那条会话的 id。
func TestWP6WaitSubscribesWithItsOwnSessionScope(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	sess := fakeIdleSession("s-wait", "h1", "36")
	s.mgr.sessions[sess.ID] = sess

	// 清干净：这条用例只看"hWait 登记了什么范围"。
	s.mgr.subMu.Lock()
	s.mgr.subs = map[chan Event]subscriber{}
	s.mgr.subMu.Unlock()

	// 超时 1 秒：请求自己会收场，我们只看它订阅的那一瞬间。
	done := make(chan struct{})
	go func() {
		defer close(done)
		resp := doReq(t, "GET", hs.URL+"/api/sessions/s-wait/wait?for=stopped&timeout=1", "", "")
		if resp.StatusCode != 200 {
			t.Errorf("hWait 该以 200 收场（超时不是错误），实得 %d（%s）",
				resp.StatusCode, respError(t, resp))
		}
	}()

	deadline := time.Now().Add(3 * time.Second)
	var scopes []string
	for time.Now().Before(deadline) {
		scopes = scopes[:0]
		s.mgr.subMu.Lock()
		for _, sub := range s.mgr.subs {
			scopes = append(scopes, sub.tag+"|"+sub.sessionID)
		}
		s.mgr.subMu.Unlock()
		if len(scopes) > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(scopes) == 0 {
		t.Fatal("等不到 hWait 的订阅登记（测试前置失败）")
	}
	for _, sc := range scopes {
		if sc != "wait|s-wait" {
			t.Errorf("hWait 的订阅范围 = %q，期望 %q —— 按会话订阅是 Q11 的落点，"+
				"它替掉了从前「订阅全部 + 客户端自己丢」那处 E 类写法（C7.2-2）", sc, "wait|s-wait")
		}
	}
	<-done
}

// `GET /api/ws` 的**范围**（契约 6.1 / C6.2 / C6.4 / C6.6 / Q11）：
//
//	?session=<句柄> → 只拿那一条（补发也只给那一条）；
//	?all=1          → 全部；
//	都不给          → 与 `all=1` 同解（**今天的行为逐字不变**：前端那条不带参数的
//	                  `/debug/api/ws` 连的就是它；见 api.go 的 wsScope 注释）。
//
// 这一条同时是"前端不用改也不会断"的判据：`web/` 不在本批的可改范围里。
func TestWP6WebSocketScopeFiltersSessionAndAll(t *testing.T) {
	srv := NewServer(&Config{Listen: "127.0.0.1:0"}, nil, "")
	// 两条会话要**真的在池里**：`?session=` 收的是服务端的原始句柄，找不到就 404
	// （见下一条用例）—— 所以这里先把它们登记上。
	for _, id := range []string{"sA", "sB"} {
		s := fakeIdleSession(id, "h1", "36")
		srv.mgr.sessions[id] = s
	}
	srv.mgr.emit(Event{Type: "log", SessionID: "sA", Text: "历史-A"})
	srv.mgr.emit(Event{Type: "log", SessionID: "sB", Text: "历史-B"})
	srv.mgr.emit(Event{Type: "log", Text: "历史-服务级"})

	mux := http.NewServeMux()
	srv.routes(mux)
	hs := httptest.NewServer(mux)
	defer hs.Close()
	base := "ws" + strings.TrimPrefix(hs.URL, "http")

	readReplay := func(t *testing.T, path string) []Event {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c, _, err := websocket.Dial(ctx, base+path, nil)
		if err != nil {
			t.Fatalf("%s 建连失败: %v", path, err)
		}
		defer c.Close(websocket.StatusNormalClosure, "")
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("%s 读第一帧失败: %v", path, err)
		}
		var frame struct {
			Type   string  `json:"type"`
			Events []Event `json:"events"`
		}
		if err := json.Unmarshal(data, &frame); err != nil {
			t.Fatalf("%s 的第一帧不是合法 JSON: %v", path, err)
		}
		if frame.Type != "replay" {
			t.Fatalf("%s 建连第一帧该是 replay，实得 %q", path, frame.Type)
		}
		return frame.Events
	}

	// ① ?session=sA：补发只给 sA 的历史（C6.6：过滤在补发切片上做一次，不在客户端做）。
	got := readReplay(t, "/api/ws?session=sA")
	if len(got) != 1 || got[0].Text != "历史-A" {
		t.Errorf("?session=sA 的补发该只有 sA 那条历史，实得 %+v", got)
	}
	// ② ?all=1：全部（含无会话的服务级事件）。
	got = readReplay(t, "/api/ws?all=1")
	if len(got) != 3 {
		t.Errorf("?all=1 该拿到全部 3 条历史，实得 %d 条：%+v", len(got), got)
	}
	// ③ 不带参数：**与 all=1 同解**（H2：前端那条老连接一个字都不用改）。
	got = readReplay(t, "/api/ws")
	if len(got) != 3 {
		t.Errorf("不带参数该与 ?all=1 同解（今天的行为），实得 %d 条：%+v", len(got), got)
	}
}

// `?session=` 点名的会话不在池里 ⇒ 当场 404 + `SESSION_NOT_FOUND`（C5.4 的句柄语义 +
// C5.15 的信封），**不许静默回落成"那就全部"**。
func TestWP6WebSocketScopeRejectsUnknownSession(t *testing.T) {
	srv := NewServer(&Config{Listen: "127.0.0.1:0"}, nil, "")
	mux := http.NewServeMux()
	srv.routes(mux)
	hs := httptest.NewServer(mux)
	defer hs.Close()

	resp := doReq(t, "GET", hs.URL+"/api/ws?session=s-不存在", "", "")
	if resp.StatusCode != 404 {
		t.Fatalf("点名的会话不在池里该 404，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var env struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("失败信封读不懂（该是 JSON）：%v", err)
	}
	if env.Code != target.CodeNotFound {
		t.Errorf("code = %q，该是 %q", env.Code, target.CodeNotFound)
	}
}

// ---------------------------------------------------------------------------
// T5.4：`--session <别名>` 真的能落位（WP5 留下的那一步）
// ---------------------------------------------------------------------------

// WP5 让 `form42f` 子进程把固定别名随 `--name` 传下去，当时服务端还不认识别名。
// WP4c 把别名变成服务端事实（`SessionBrief.name`），**本条确认它真的可寻址**：
// 池里有两条同环境会话时，`--session <别名>` 精确落到那一条（不是歧义、不是第一条）。
func TestWP6SessionAliasLands(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	first := idleOn("s1", cfgFor("h1", "36"))
	second := idleOn("s2", cfgFor("h1", "36"))
	m.mu.Lock()
	if err := m.assignIdentityLocked(first, recAliasForTest); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	if err := m.assignIdentityLocked(second, "别人"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"], m.sessions["s2"] = first, second
	m.mu.Unlock()

	pool := target.Pool{
		Environments: []string{"h1-36"},
		Sessions:     sessionViews(m.Snapshot()),
	}
	sel := selectorOf(t, recAliasForTest) // 不含 / 与 # ⇒ 按别名解析（C2.4）
	if sel.Kind != target.SelectorAlias {
		t.Fatalf("别名 %q 该被解析成别名形态，实得 %v", recAliasForTest, sel.Kind)
	}
	got, f := target.Resolve(pool, target.Request{Selector: sel})
	if f != nil {
		t.Fatalf("`--session <别名>` 落不了位：%s（F5 会让人以为别名没登记上）", f.Message)
	}
	if got.ID != "s1" {
		t.Fatalf("别名该落到 s1，实得 %s（多会话下这可就是上错号）", got.ID)
	}
	// 反向判据：**不带目标**时同样的池子必须报歧义 —— 那正是别名存在的理由
	// （它是跨轮追踪一条会话的那把钥匙，Q20 / C3.7c）。
	if _, f := target.Resolve(pool, target.Request{}); f == nil || f.Code != target.CodeAmbiguous {
		t.Fatalf("不带目标时该报歧义（F1），实得 %v", f)
	}
}

// recAliasForTest 是按 WP5 的算法（`recSessionAlias`）算出来的那种形状：
// `<会话别名>`。这里写死一个字面量，免得为了一个字符串把命令层 import 进来。
const recAliasForTest = "rec-20261016-120000"

// ---------------------------------------------------------------------------
// "两条会话能同时跑、互不干扰"（本批的总验收）
// ---------------------------------------------------------------------------

// 同环境两条会话**各自独立**：各自的序号/ref/别名/世代互不串台，收掉一条不动另一条。
//
// 这是本批总验收（WP6 的"放开多会话"）在**不拨真机**这一层能给出的最强判据；
// 真两条 SSH 会话同时跑断点的那一半见 README 的"故意不覆盖"与真机步骤。
func TestWP6TwoSessionsRunSideBySide(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	a := idleOn("sA", cfg)
	b := idleOn("sB", cfg)
	m.mu.Lock()
	// 一条一条登记**并进池**：序号是"取池里尚未被占用的最小正整数"，
	// 不进池的第二条会拿到同一个序号（C2.2 / Q24）。
	if err := m.assignIdentityLocked(a, "甲"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["sA"] = a
	if err := m.assignIdentityLocked(b, "乙"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["sB"] = b
	m.mu.Unlock()

	snaps := m.Snapshot()
	if len(snaps) != 2 {
		t.Fatalf("该有两条会话，实得 %d 条", len(snaps))
	}
	byID := map[string]SessionBrief{}
	for _, s := range snaps {
		byID[s.ID] = s
	}
	aBrief, bBrief := byID["sA"], byID["sB"]
	// 序号按环境自增（C2.2 / Q24）：两条会话必须是**两个不同的** `环境#序号`。
	if aBrief.N == bBrief.N {
		t.Errorf("两条会话拿到了同一个序号 %d —— `环境#序号` 当场撞车（C2.0）", aBrief.N)
	}
	if aBrief.Ref == bBrief.Ref || aBrief.Ref == "" || bBrief.Ref == "" {
		t.Errorf("两条会话的 ref 必须互不相同且非空，实得 %q / %q（C2.0 / C5.4）",
			aBrief.Ref, bBrief.Ref)
	}
	// 世代互不相同（F15 的前提：拿谁的 epoch 就只对谁有效）。
	if aBrief.Epoch == bBrief.Epoch {
		t.Errorf("两条会话的 epoch 撞了（%q）—— F15 就判不出「手上那份是谁的」", aBrief.Epoch)
	}
	// 别名各归各的（Q23 的幂等键）。
	if aBrief.Name != "甲" || bBrief.Name != "乙" {
		t.Errorf("别名串台了：%q / %q", aBrief.Name, bBrief.Name)
	}

	// **收掉一条不动另一条**（这是"互不干扰"的实质）。
	m.Remove("sA")
	left := m.Snapshot()
	if len(left) != 1 || left[0].ID != "sB" {
		t.Fatalf("收掉 sA 之后该只剩 sB，实得 %+v", left)
	}
	if left[0].Name != "乙" || left[0].Ref != bBrief.Ref {
		t.Errorf("收掉 sA 之后 sB 的身份变了：%+v", left[0])
	}
	// 别名随会话消失即释放（C3.7b / Q19）：甲这个名字现在可以给别人用。
	if err := m.checkNameFree("甲"); err != nil {
		t.Errorf("sA 走了之后别名「甲」该被释放（C3.7b / Q19），实得 %v", err)
	}
	if err := m.checkNameFree("乙"); err == nil {
		t.Error("sB 还在，别名「乙」不该被释放")
	}
}

// ---------------------------------------------------------------------------
// HTTP 层：多会话 + 不给目标 ⇒ 409 + SESSION_AMBIGUOUS + 候选清单（C5.15 / C5.16）
// ---------------------------------------------------------------------------

// 造会话那条路（`POST /api/sessions`）落不上目标时，**信封必须带 code 与候选**。
//
// 从前这里一律是 `fail(w, 409, err)`（一句散文、没有 code、没有候选）——
// 调用方只知道"有 N 条会话"却不知道是哪几条，C5.11 的"必须出现候选"当场落空。
func TestWP6LaunchReportsAmbiguousWithCandidates(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	// 两条同目标（服务端 cfg 的 EnvName："h1-36"）的空闲会话。
	// **用 `idleOn(id, s.cfg)` 而不是 `fakeIdleSession`**：后者的 `envName` 是空的，
	// 而 `sameTarget` 在"两侧都算得出且不等"时判**换目标** —— 空名字的会话在
	// "目标侧算得出"时走的是那条"允许复用"的防御分支，但这里要测的是**正常环境匹配**。
	for _, id := range []string{"s1", "s2"} {
		s.mgr.sessions[id] = idleOn(id, s.cfg)
	}
	s.mgr.newSession = fakeNewIdleSession

	// 作业名刻意不是标准标识符：`resolveJobWith` 对非标识符**不连库**（`reProgName` 不匹配
	// → 直接返回空），于是这次请求一路走到会话准备那一步 —— 否则它会在**解析作业**时
	// 就先拨一次 SSH 并报别的错（那就不叫"测歧义"了）。
	resp := doReq(t, "POST", hs.URL+"/api/sessions", "ai", `{"prog":"my job","ent":"99"}`)
	if resp.StatusCode != 409 {
		t.Fatalf("多会话 + 不给目标该 409（C5.16），实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var env struct {
		OK         bool               `json:"ok"`
		Code       string             `json:"code"`
		Error      string             `json:"error"`
		Candidates []target.Candidate `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v", err)
	}
	if env.OK {
		t.Error("失败信封的 ok 必须是 false（C5.15：形状一字不改）")
	}
	if env.Code != target.CodeAmbiguous {
		t.Errorf("code = %q，F1 要求 %q", env.Code, target.CodeAmbiguous)
	}
	if env.Error == "" {
		t.Error("信封的 error 是空的 —— 那句正文一个字都不许少（C5.15）")
	}
	if len(env.Candidates) != 2 {
		t.Fatalf("F1 必须带候选清单（C5.11），实得 %d 条：%+v", len(env.Candidates), env.Candidates)
	}
	// 候选就是 C5.12 的九键（唯一出处是 internal/target 的 Candidate）。
	for _, c := range env.Candidates {
		if c.ID == "" || c.Env == "" || c.State == "" {
			t.Errorf("候选缺键（C5.12 的九键）：%+v", c)
		}
	}
}

// 反向判据（H2 / C8.1）：**池里只有一条**时，同一个请求照旧成功 —— 一次 200，
// 而且返回的就是那条会话的句柄（"唯一就落"不是"挑一条"，唯一是确定性的）。
func TestWP6LaunchSingleSessionStillFallsThrough(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	only := idleOn("s-one", s.cfg)
	s.mgr.sessions[only.ID] = only
	s.mgr.newSession = fakeNewIdleSession

	resp := doReq(t, "POST", hs.URL+"/api/sessions", "ai", `{"prog":"my job","ent":"99"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("单会话 + 不给目标照旧该成功（Q2 / C8.1），实得 %d（%s）",
			resp.StatusCode, respError(t, resp))
	}
	var out struct {
		OK        bool   `json:"ok"`
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("回包不是合法 JSON：%v", err)
	}
	if !out.OK || out.SessionID != only.ID {
		t.Errorf("该落在唯一那条 %s 上，实得 ok=%v sessionId=%q", only.ID, out.OK, out.SessionID)
	}
}

// --- 小工具 ---

// fakeNewIdleSession 是**不拨号**的 `NewSession` 替身（只给判据测试用）。
//
// 为什么需要它：`NewSession` 里是 `host.Dial`，没有可注入的缝 —— 无网环境里
// "单会话放行 / 多会话拒绝"这两条 HTTP 判据只能测到拨号失败那一层，
// 测不到"落位之后到底成不成功"（那才是它们要说的事）。
// 造出来的形状与 `idleOn` 同族（`booted` + idle + 身份齐），所以 `hLaunch` 之后
// 那条 `go sess.Launch(...)` 走的是 `readyForRun` 的复用分支，不会去打 pty/conn。
func fakeNewIdleSession(cfg *Config, module, prog, runProg, launchRef, extra string, emit func(Event)) (*Session, error) {
	s := idleOn(nextSessionID(), cfg)
	s.Module, s.Prog, s.RunProg = module, prog, runProg
	s.booted = true
	s.emit = emit
	return s, nil
}

// drainTexts 把通道里当前积压的事件文案取出来（非阻塞）。
func drainTexts(ch <-chan Event) []string {
	var out []string
	for {
		select {
		case ev := <-ch:
			out = append(out, ev.Text)
		default:
			return out
		}
	}
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
