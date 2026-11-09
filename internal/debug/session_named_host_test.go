package debug

// 「起会话」那条链上两个 bug 的**判据测试**（Bug A：`--session` 只校验不生效；
// Bug B：`wsdebug` 没有 `--name`）。
//
// # Bug A 钉住的三件事（服务端这一侧）
//
//   - **两条候选 + 显式指名 → 就落那一条，而且不报歧义**：指名是精确通道，不是"又一个候选"
//     （`prepareSessionOn` 的 `hostID` 一进来就把候选集收成一条）；
//   - **指名的不存在 → `SESSION_NOT_FOUND`，绝不回落成"那就按候选再收窄一次"**：
//     回落会把"我点的那条没了"变成"落到另一条上，而输出看起来一切正常" ——
//     这正是这次重构开头要根除的那句话；
//   - **不给目标 + 两条候选 → 仍然 `SESSION_AMBIGUOUS` + 候选清单**：判据挂在**新的九参入口**
//     上，所以"顺手把歧义分支改掉"这件事会当场变红（这条与 wp6 那条同源，放在这里是为了
//     让"给了就落 / 不给仍歧义"这对反向判据出现在同一处）。
//
// 另外两条"指名不等于跳过校验"的判据：状态处置仍走 `decideHost`（忙碌就报忙碌），
// 目标/别名对不上时报错而**不把点名的会话收掉重建**。
//
// # Bug B 钉住的两件事
//
//   - 非法别名在 hWSLogDebug 里**先于拨号/收口**被拒（400 + `USAGE`），
//     合法别名则放行到拨号那一层（判据是"拨了几次号"）；
//   - `name` 从 hWSLogDebug 一路走到会话登记（接线文本判据，理由见那条测试的注释）。
//
// # 为什么不拨真机
//
// 复用/忙碌/歧义/目标不符四种处置都在**拨号之前**判完；需要"起新会话"的那几支刻意不碰
// （`h1` 解不出来，失败了也不算这条用例的事）。HTTP 层的用例用
// `actionTestDebugServer` + `idleOn`（不碰网络），与 wp6_multisession_test.go 同一套。

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"tt/internal/host"
	"tt/internal/target"
)

// ---------------------------------------------------------------------------
// Bug A：显式指名（`--session`）就是那一条
// ---------------------------------------------------------------------------

// TestNamedHostWinsOverTwoCandidates：同环境两条空闲会话 + 显式指名 → 落到指名那一条。
//
// 这一条是 Bug A 的核心：从前服务端只按"环境+名字"收窄，两条候选时必报歧义，
// `--session` 那个标志在本机校验完之后就被丢掉了。
func TestNamedHostWinsOverTwoCandidates(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	a, b := idleOn("sA", cfg), idleOn("sB", cfg)
	m.mu.Lock()
	// 身份要真的登记过（序号 / 世代 / 审计行都要它，C2.0 / C5.3）。
	for _, s := range []*Session{a, b} {
		if err := m.assignIdentityLocked(s, ""); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记身份失败：%v", err)
		}
	}
	m.sessions["sA"], m.sessions["sB"] = a, b
	m.mu.Unlock()

	got, err := m.prepareSessionOn(cfg, "m1", "p1", "", "", "", "", "", "sB")
	if err != nil {
		t.Fatalf("显式指名了 sB，却落不上（多候选歧义在指名档根本不该成立）：%v", err)
	}
	if got != b {
		t.Fatalf("该落在指名的 sB 上，实得 %v —— 两条候选时落到别人身上就是上错号", got.ID)
	}
	// 指名不是"顺手把另一条也收了"：另一条一个字都不该动。
	if a.Module != "" || a.Prog != "" {
		t.Errorf("没被指名的那条会话被改动了：module=%q prog=%q", a.Module, a.Prog)
	}
	if n := len(m.Snapshot()); n != 2 {
		t.Errorf("指名复用不该改动会话池的条数，实得 %d 条", n)
	}
}

// TestNamedHostMissingNeverFallsBack：指名一个不存在的句柄 → `SESSION_NOT_FOUND`，
// 而且**池子里一条都不许被碰**（这就是"绝不退化成随便落一条"的判据）。
func TestNamedHostMissingNeverFallsBack(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	a, b := idleOn("sA", cfg), idleOn("sB", cfg)
	m.mu.Lock()
	m.sessions["sA"], m.sessions["sB"] = a, b
	m.mu.Unlock()

	_, err := m.prepareSessionOn(cfg, "m1", "p1", "", "", "", "", "", "s-不存在")
	if err == nil {
		t.Fatal("点名了一条不存在的会话却落了位 —— 那是「随便落一条」")
	}
	var f *target.Failure
	if !errors.As(err, &f) {
		t.Fatalf("该是目标层的失败（服务端要按 code 回信封），实得 %T: %v", err, err)
	}
	if f.Code != target.CodeNotFound {
		t.Errorf("code = %q，指名不存在要求 %q（C2.7 / C5.13）", f.Code, target.CodeNotFound)
	}
	// 回落成"随便落一条"的机器判据：两条候选的 Prog 都还是空（一条都没被 SetRun 换轮）。
	for _, s := range []*Session{a, b} {
		if s.Prog != "" || s.Module != "" {
			t.Errorf("指名的会话不存在时**不许**退而落到 %s 上（prog=%q）", s.ID, s.Prog)
		}
	}
}

// TestOmittedTargetStillAmbiguousOnTheNewEntryPoint：不给目标 + 两条候选 → 仍报歧义。
//
// 判据刻意挂在新入口 `prepareSessionOn` 上（指名那一路的新家），并且与八参的
// `prepareSession` 对一遍：**没指名时两条入口必须同解** —— 这样"改指名的分支时
// 顺手把歧义那条改掉"会当场变红，而 H2 靠的正是这条同解。
func TestOmittedTargetStillAmbiguousOnTheNewEntryPoint(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	for _, s := range []*Session{idleOn("sA", cfg), idleOn("sB", cfg)} {
		m.mu.Lock()
		if err := m.assignIdentityLocked(s, ""); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记身份失败：%v", err)
		}
		m.sessions[s.ID] = s
		m.mu.Unlock()
	}

	_, errNew := m.prepareSessionOn(cfg, "m1", "p1", "", "", "", "", "", "")
	if errNew == nil {
		t.Fatal("两条候选 + 不给目标必须报歧义 —— 替调用方挑一条就是这次重构要根除的事（F1 / C8.1）")
	}
	var fNew *target.Failure
	if !errors.As(errNew, &fNew) || fNew.Code != target.CodeAmbiguous {
		t.Fatalf("该报 %s，实得 %T: %v", target.CodeAmbiguous, errNew, errNew)
	}
	if len(fNew.Candidates) != 2 {
		t.Fatalf("F1 必须带候选清单（C5.11），实得 %d 条", len(fNew.Candidates))
	}

	// 八参入口（没指名那一路的旧调用点）必须同解：同一码、同一批候选。
	_, errOld := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
	var fOld *target.Failure
	if !errors.As(errOld, &fOld) {
		t.Fatalf("八参入口也该报目标层失败，实得 %T: %v", errOld, errOld)
	}
	if fOld.Code != fNew.Code || len(fOld.Candidates) != len(fNew.Candidates) {
		t.Errorf("新旧入口在「没指名」这一档必须同解：%q/%d 条 vs %q/%d 条",
			fOld.Code, len(fOld.Candidates), fNew.Code, len(fNew.Candidates))
	}
	for i := range fOld.Candidates {
		if fOld.Candidates[i].ID != fNew.Candidates[i].ID {
			t.Errorf("候选顺序/内容该一致（C5.14 稳定排序），第 %d 条：%s vs %s",
				i, fOld.Candidates[i].ID, fNew.Candidates[i].ID)
		}
	}
}

// TestNamedHostBusyIsStillBusy：指名**不跳过校验** —— 那条会话在跑/已停站就报那句老话。
func TestNamedHostBusyIsStillBusy(t *testing.T) {
	cfg := cfgFor("h1", "36")
	m := NewManager(cfg)
	busy := idleOn("sBusy", cfg)
	busy.mu.Lock()
	busy.state = StateStopped
	busy.mu.Unlock()
	m.mu.Lock()
	m.sessions["sBusy"] = busy
	m.mu.Unlock()

	_, err := m.prepareSessionOn(cfg, "m1", "p1", "", "", "", "", "", "sBusy")
	if err == nil {
		t.Fatal("指名的会话已停站却落了位 —— 显式指名不是「跳过所有校验」")
	}
	if !strings.Contains(err.Error(), "请先结束当前调试") {
		t.Errorf("忙碌那一档的文案该与今天一致（H2），实得 %q", err.Error())
	}
}

// TestNamedHostMismatchedTargetIsRefusedNotCollected：指名的会话与这次目标的环境对不上 →
// 报错，**不许把它收掉再在别处起一条**。
//
// 为什么这一条重要：候选档的 `hostCollect` 会 `Close` 掉旧会话重建（空转宿主没什么可丢的），
// 但对**点名的那一条**，那等于把"用这一条"读成"用别处起一条"，用户手上的 ref/别名当场失效。
// 正常路径上 CLI 的 `target.Resolve` 已经按 F12 拦下，这里是服务端的兜底。
func TestNamedHostMismatchedTargetIsRefusedNotCollected(t *testing.T) {
	here, there := envCfg("恒烁正式区"), envCfg("测试1012")
	m := NewManager(here)
	other := idleOn("sThere", there)
	m.mu.Lock()
	if err := m.assignIdentityLocked(other, "别的环境"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["sThere"] = other
	m.mu.Unlock()

	_, err := m.prepareSessionOn(here, "m1", "p1", "", "", "", "", "", "sThere")
	if err == nil {
		t.Fatal("指名的会话在另一个环境，却落了位 —— 那会把这一轮跑到别人的环境上")
	}
	var f *target.Failure
	if !errors.As(err, &f) || f.Code != target.CodeUsage {
		t.Fatalf("该报 %q（F12 那一族的环境矛盾），实得 %T: %v", target.CodeUsage, err, err)
	}
	if !strings.Contains(f.Message, "恒烁正式区") || !strings.Contains(f.Message, "测试1012") {
		t.Errorf("文案该把两个环境都写出来（人要靠它看出是哪边不一致），实得 %q", f.Message)
	}
	if m.Get("sThere") == nil {
		t.Error("指名的会话**不许**因为目标对不上就被收掉（那是「收口重建」，不是「指名」）")
	}
}

// TestLaunchNamedSessionOverridesAmbiguity 是同一件事的 **HTTP 半边**（hLaunch）：
// 请求体带 `session` → 200 且回的就是那条句柄，而不是 409 歧义。
func TestLaunchNamedSessionOverridesAmbiguity(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	for _, id := range []string{"s1", "s2"} {
		s.mgr.sessions[id] = idleOn(id, s.cfg)
	}

	resp := doReq(t, "POST", hs.URL+"/api/sessions", "ai", `{"prog":"my job","session":"s2","ent":"99"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("两条候选 + 显式指名该直接落那一条（Bug A），实得 %d（%s）",
			resp.StatusCode, respError(t, resp))
	}
	var out struct {
		OK        bool   `json:"ok"`
		SessionID string `json:"sessionId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("回包不是合法 JSON：%v", err)
	}
	if !out.OK || out.SessionID != "s2" {
		t.Errorf("该落在指名的 s2 上，实得 ok=%v sessionId=%q", out.OK, out.SessionID)
	}
	if other := s.mgr.Get("s1"); other == nil {
		t.Error("没被指名的那条会话不该被收掉")
	} else if other.Prog != "" {
		t.Errorf("没被指名的那条会话被换了一轮（prog=%q）—— 提名是「用这一条」，不是「收掉别的」", other.Prog)
	}
}

// TestLaunchNamedSessionMissingIsNotFound 是 hLaunch 的失败半边：
// 指名一条不在池里的会话 → **404 + `SESSION_NOT_FOUND`**，池子一个字不动。
func TestLaunchNamedSessionMissingIsNotFound(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	for _, id := range []string{"s1", "s2"} {
		s.mgr.sessions[id] = idleOn(id, s.cfg)
	}

	resp := doReq(t, "POST", hs.URL+"/api/sessions", "ai", `{"prog":"my job","session":"s-没有"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("指名一条不存在的会话该 404（C5.16 的 `SESSION_NOT_FOUND`），实得 %d（%s）",
			resp.StatusCode, respError(t, resp))
	}
	var env struct {
		OK    bool   `json:"ok"`
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v", err)
	}
	if env.OK || env.Code != target.CodeNotFound {
		t.Errorf("信封该是 ok=false + code=%q，实得 %+v", target.CodeNotFound, env)
	}
	if env.Error == "" {
		t.Error("信封的 error 不许为空（C5.15 那句正文）")
	}
	// 绝不退化成"那就按候选再收窄一次"：两条会话一条都没被换轮、也没被收掉。
	if n := len(s.mgr.Snapshot()); n != 2 {
		t.Errorf("指名的会话不存在时池子一个字都不该动，实得 %d 条", n)
	}
	for _, id := range []string{"s1", "s2"} {
		if sess := s.mgr.Get(id); sess == nil || sess.Prog != "" {
			t.Errorf("会话 %s 不该被这次失败的请求改动", id)
		}
	}
}

// TestLaunchWithoutSessionStillReportsAmbiguity：**反向判据** —— 同一个请求去掉 `session`
// 之后仍然 409 + `SESSION_AMBIGUOUS` + 两条候选。
//
// 与上面那条成对存在：加了 `session` 就落那一条，不加就照旧报歧义。少了这一条，
// "把歧义分支顺手改成挑一条"不会红（那是这次重构的根）。
func TestLaunchWithoutSessionStillReportsAmbiguity(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	for _, id := range []string{"s1", "s2"} {
		s.mgr.sessions[id] = idleOn(id, s.cfg)
	}

	resp := doReq(t, "POST", hs.URL+"/api/sessions", "ai", `{"prog":"my job"}`)
	if resp.StatusCode != 409 {
		t.Fatalf("不给目标 + 两条候选该 409（C5.16），实得 %d（%s）",
			resp.StatusCode, respError(t, resp))
	}
	var env struct {
		Code       string             `json:"code"`
		Candidates []target.Candidate `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v", err)
	}
	if env.Code != target.CodeAmbiguous {
		t.Errorf("code = %q，F1 要求 %q", env.Code, target.CodeAmbiguous)
	}
	if len(env.Candidates) != 2 {
		t.Fatalf("F1 必须带两条候选（C5.11），实得 %d 条", len(env.Candidates))
	}
	if n := len(s.mgr.Snapshot()); n != 2 {
		t.Errorf("报歧义时不许改动会话池，实得 %d 条", n)
	}
}

// ---------------------------------------------------------------------------
// 「创建会话」的冷启动档：池子空也算正常起点
// ---------------------------------------------------------------------------

// TestSessionSwitchColdStartIsNotNoSession：**池空 + 没点名 = 创建会话**，不是 `NO_SESSION`。
//
// 真机踩到的正是这一条：`tt debug env <环境>`（Web 上叫「创建会话」）在会话池为空时
// 直接报 `NO_SESSION`，于是"先建会话 → 再设 ENT → 再调试"这条正确流程**第一步**就断了，
// 用户只能先"起个作业"去凑一条会话，顺序完全反了。
//
// 判据不钉状态码（越过闸门后它会真去拨号，而测试的拨号桩必然失败 → 500），
// 只钉"**不许**是 409 + `NO_SESSION`"这件事本身 —— 那才是被修掉的错误。
func TestSessionSwitchColdStartIsNotNoSession(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1"}`)
	var raw struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if resp.StatusCode == 409 && raw.Code == target.CodeNoSession {
		t.Fatalf("池空且没点名时「创建会话」不该报 %s —— 那是把冷启动当成排错", target.CodeNoSession)
	}
	// 失败也只能是因为拨号（测试桩），不能因为"没有会话可供收掉"。
	if n := len(s.mgr.Snapshot()); n != 0 {
		t.Errorf("拨号失败不该留下半条会话，实得 %d 条", n)
	}
}

// TestSessionSwitchColdStartStillRejectsNamedMissing：冷启动档**没有放宽点名**。
//
// 与上一条成对：池空时"没点名"放行、"点名了但不存在"仍然 `SESSION_NOT_FOUND`。
// 少了这一条，把 mustTargetOrEmpty 写成"池空就无条件放行"也不会红。
func TestSessionSwitchColdStartStillRejectsNamedMissing(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1","session":"s-没有"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("点名一条不存在的会话该 404，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var rawn struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&rawn)
	if rawn.Code != target.CodeNotFound {
		t.Errorf("code = %q，要求 %q", rawn.Code, target.CodeNotFound)
	}
}

// TestWSLogDebugColdStartIsNotNoSession：「调试此调用」（池空 + 没点名）= **自己造那一条**，
// 不是 `NO_SESSION`。
//
// 真机踩到的（界面「接口日志」页 → 点「调试此调用」，命令行是 `tt debug wsdebug <rowid>`）：
// 一台还没有任何会话的机器上，点下去当场报
// `查询失败 按接口日志重放：没有 SSH 会话 —— 先运行 tt debug start <作业> --env <环境> 起一条`。
// 而重放干的就是**创建**一条会话来重跑日志（`LaunchReplay` → `prepareSession`）：
// 它要的那一条本来就该由它自己建（与「创建会话」`tt debug env <环境>` 同一个理由，
// 见上面那条 `TestSessionSwitchColdStartIsNotNoSession`）。
//
// 判据同上：不钉状态码（越过闸门后它会真去拨号，测试的拨号桩必然失败 → 500），
// 只钉"**不许**是 409 + `NO_SESSION`"这一件事本身。
func TestWSLogDebugColdStartIsNotNoSession(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	// 界面那条请求的形状：rowid + ent，**没有** session 键（池空，前端拿不到旧会话）。
	resp := doReq(t, "POST", hs.URL+"/api/wslogs/debug", "", `{"rowid":"1","ent":"99"}`)
	var raw struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&raw)
	if resp.StatusCode == 409 && raw.Code == target.CodeNoSession {
		t.Fatalf("池空且没点名时按日志重放不该报 %s —— 重放自己就是造会话的命令",
			target.CodeNoSession)
	}
	// 失败也只能是因为拨号（测试桩），不能因为"没有会话可供收掉"；更不能留下半条会话。
	if n := len(s.mgr.Snapshot()); n != 0 {
		t.Errorf("拨号失败不该留下半条会话，实得 %d 条", n)
	}
}

// ---------------------------------------------------------------------------
// Bug B：`wsdebug` 的 `--name`
// ---------------------------------------------------------------------------

// TestWSLogDebugRejectsBadAliasBeforeDialing：非法别名在**拨号/收口之前**被拒。
//
// 判据用"拨了几次号"而不是只看状态码：`beginNamedLaunch` 的位置就在 `mustTarget`
// 与 `dialSSH` 之前，判错位置（比如挪到 LaunchReplay 里）时状态码可能还一样，
// 但那时当前会话已经被收掉了 —— 那正是 C2.8 反复要防的顺序。
func TestWSLogDebugRejectsBadAliasBeforeDialing(t *testing.T) {
	for name, bad := range map[string]string{
		"含斜杠":   "a/b",
		"含井号":   "a#b",
		"超过 64": strings.Repeat("长", target.AliasMaxRunes+1),
	} {
		t.Run(name, func(t *testing.T) {
			hs, s := actionTestDebugServer(t)
			s.mgr.sessions["s1"] = idleOn("s1", s.cfg)
			var dialed int32
			s.dial = func(host.SSHConfig) (*host.SSHConn, error) {
				atomic.AddInt32(&dialed, 1)
				return nil, errTestNoDial
			}
			body, _ := json.Marshal(map[string]any{"rowid": "1", "session": "s1", "name": bad})
			resp := doReq(t, "POST", hs.URL+"/api/wslogs/debug", "ai", string(body))
			if resp.StatusCode != 400 {
				t.Fatalf("非法别名 %q 该 400 + USAGE（F19），实得 %d（%s）", bad, resp.StatusCode, respError(t, resp))
			}
			var env struct {
				Code string `json:"code"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
				t.Fatalf("失败信封不是合法 JSON：%v", err)
			}
			if env.Code != target.CodeUsage {
				t.Errorf("code = %q，F19 要求 %q", env.Code, target.CodeUsage)
			}
			if n := atomic.LoadInt32(&dialed); n != 0 {
				t.Errorf("非法别名拨了 %d 次号 —— 判定要赶在拨号与收口之前（C2.8）", n)
			}
			if s.mgr.Get("s1") == nil {
				t.Error("非法别名不该把当前会话收掉")
			}
		})
	}
}

// TestWSLogDebugAcceptsValidAlias：合法别名**放行到拨号那一层**（不再被别名闸门拦下）。
//
// 它证明 hWSLogDebug 真的认这个键（而不是"悄悄忽略未知字段"—— 服务端恰好有这种静默行为，
// 见 dbgScreenVerify 的注释）。"登记"那一步见下一条接线判据。
func TestWSLogDebugAcceptsValidAlias(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	s.mgr.sessions["s1"] = idleOn("s1", s.cfg)
	var dialed int32
	s.dial = func(host.SSHConfig) (*host.SSHConn, error) {
		atomic.AddInt32(&dialed, 1)
		return nil, errTestNoDial
	}

	resp := doReq(t, "POST", hs.URL+"/api/wslogs/debug", "ai",
		`{"rowid":"1","session":"s1","name":"调试-甲"}`)
	if resp.StatusCode != 500 {
		t.Fatalf("合法别名该放行到拨号那一层（这里注入的拨号必然失败 → 500），实得 %d（%s）",
			resp.StatusCode, respError(t, resp))
	}
	if n := atomic.LoadInt32(&dialed); n != 1 {
		t.Errorf("合法别名该走到拨号（拨 %d 次号），说明它没被当成非法别名拦下", n)
	}
}

// TestWSLogDebugPassesNameIntoSessionRegistration 钉住 `name` 的**接线**：
// `hWSLogDebug` 收到它之后必须一路送到会话登记（hWSLogDebug → LaunchReplay → prepareSession），
// 缺任何一环别名都不会真的登记上，而**从外面看不出来**（会话照常起来，只是 name 是空串）。
//
// 为什么只能读源码：这条路上 `hWSLogDebug` 必须先拨 SSH 去读 `wsfa_t`（`WSLogDetail`），
// 而 `LaunchReplay` 里的 `host.Dial` 与 `WriteReplayFiles` 都没有可注入的缝
// （`api_action_test.go` 的注释写着为什么 `*host.SSHConn` 伪造不了）。于是"参数有没有
// 往下传"这一环，在不连真机的测试里只能钉接线的文本 —— 与 `session_mode_source_test.go`
// （"每条造会话的路径都要 SetMode"）同一路数：把注释里的纪律写成断言。
func TestWSLogDebugPassesNameIntoSessionRegistration(t *testing.T) {
	handler := funcBodyOf(t, "api.go", "func (s *Server) hWSLogDebug(")
	if !regexp.MustCompile(`s\.mgr\.LaunchReplay\([^)]*\bname\b`).MatchString(handler) {
		t.Errorf("hWSLogDebug 没有把 name 交给 LaunchReplay —— 别名会在这里断线，\n" +
			"    会话照常起得来，只是它永远不会被登记成有名字的会话")
	}
	replay := funcBodyOf(t, "manager.go", "func (m *Manager) LaunchReplay(")
	if !regexp.MustCompile(`m\.prepareSession\([^)]*\bname\b`).MatchString(replay) {
		t.Errorf("LaunchReplay 没有把 name 交给 prepareSession —— 它是唯一登记别名的地方\n" +
			"    （assignIdentityLocked / nameSession），漏了这一步等于 wsdebug 的 --name 白给")
	}
}

// funcBodyOf 取出一个顶层函数的源码（从签名到行首的收尾 `}`）。
//
// 只给源码判据用：gofmt 之后顶层函数体的收尾括号在第 0 列，所以 `\n}\n` 就是函数尾。
func funcBodyOf(t *testing.T, file, sig string) string {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("读不到 %s: %v", file, err)
	}
	src := string(b)
	i := strings.Index(src, sig)
	if i < 0 {
		t.Fatalf("%s 里找不到 %q —— 函数改名了就把这条判据一起改", file, sig)
	}
	rest := src[i:]
	if j := strings.Index(rest, "\n}\n"); j >= 0 {
		return rest[:j]
	}
	return rest
}

// ---------------------------------------------------------------------------
// 「创建即命名」：`debug env --name X` 的四种落点（P1）
// ---------------------------------------------------------------------------

// TestSwitchNamesAnUnnamedSession：池里已有一条同环境**无名**的空闲宿主 → 把名字登记给它，不重连。
func TestSwitchNamesAnUnnamedSession(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	live := idleOn("s-live", s.cfg.CloneEnv("h1"))
	s.mgr.sessions[live.ID] = live

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1","name":"录像A"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("无名宿主应被就地命名（不该重连）：%d（%s）", resp.StatusCode, respError(t, resp))
	}
	var out struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.SessionID != live.ID {
		t.Fatalf("应落回原来那条（%s），实得 %s", live.ID, out.SessionID)
	}
	if live.Name() != "录像A" {
		t.Fatalf("名字没登记上：%q", live.Name())
	}
}

// TestSwitchSameNameIsIdempotent：同名再来一次 = 幂等（同一条会话）。
func TestSwitchSameNameIsIdempotent(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	live := idleOn("s-live", s.cfg.CloneEnv("h1"))
	live.setNameInTest("录像A")
	s.mgr.sessions[live.ID] = live

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1","name":"录像A"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("同名应幂等：%d（%s）", resp.StatusCode, respError(t, resp))
	}
	if n := len(s.mgr.Snapshot()); n != 1 {
		t.Fatalf("不该多出一条会话，实得 %d", n)
	}
}

// TestSwitchDifferentNameRebuildsWhenIdle：叫别的名字且空闲 → 收掉旧的按新名重建。
//
// 新建要走拨号（测试桩必然失败），所以这里断言的是"旧的被收掉了、请求走到了新建"，
// 而不是结果 200。
func TestSwitchDifferentNameRebuildsWhenIdle(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	old := idleOn("s-old", s.cfg.CloneEnv("h1"))
	old.setNameInTest("旧名")
	s.mgr.sessions[old.ID] = old

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1","name":"录像A"}`)
	if resp.StatusCode == 200 {
		t.Fatal("名字不同且旧会话叫别的名字：那是改名，不该成功")
	}
	if s.mgr.Get(old.ID) != nil {
		t.Fatal("旧会话应被收掉（它是空闲的，收口重建）")
	}
}

// TestSwitchDifferentNameRefusesWhenBusy：叫别的名字且**不空闲** → 409，不许悄悄收掉。
func TestSwitchDifferentNameRefusesWhenBusy(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	busy := idleOn("s-busy", s.cfg.CloneEnv("h1"))
	busy.setNameInTest("旧名")
	busy.mu.Lock()
	busy.state = StateStopped
	busy.mu.Unlock()
	s.mgr.sessions[busy.ID] = busy

	resp := doReq(t, "POST", hs.URL+"/api/sessions/switch", "", `{"env":"h1","name":"录像A"}`)
	if resp.StatusCode != 409 {
		t.Fatalf("忙会话不许被收掉重建，该 409，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	if s.mgr.Get(busy.ID) == nil {
		t.Fatal("被拒时那条会话必须原样留着")
	}
}

// setNameInTest 直接给会话命名（测试用）：走字段与锁，不经过服务端登记流程。
func (s *Session) setNameInTest(name string) {
	s.mu.Lock()
	s.name = name
	s.mu.Unlock()
}
