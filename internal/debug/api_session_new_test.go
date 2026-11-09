// 「再建一条会话」（`POST /api/sessions/new`）的判据。
//
// 这条端点存在的理由就是**与 `/api/sessions/switch` 不同**：switch 遇到"同环境已有会话、
// 而这次点了别的名字"时，会把那条空闲的收掉再按新名重建（C3.7b / Q19 —— 那是"换目标"的语义）。
// 多会话要的是"再给我一条，一条都别动"，所以这里钉的正是**池子一个字都不动**。
//
// 为什么这几条必须存在：hSessionSwitch 那四条测试（session_named_host_test.go）已经把
// "回收重建"钉死了 —— 拿 switch 当"创建第二条会话"用，旧的会静默消失，而界面/CLI 看到的是
// "新会话建好了"。那正是这次多会话重构要根除的那类错。
package debug

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// envelopeOf 解出失败/成功信封里我们关心的键（`ok` / `code` / `error`）。
func envelopeOf(t *testing.T, resp *http.Response) (ok bool, code, msg string) {
	t.Helper()
	var r struct {
		OK    bool   `json:"ok"`
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Fatalf("信封解不开: %v", err)
	}
	return r.OK, r.Code, r.Error
}

// 同环境已有同名会话 → 原样返回它（幂等），不重连、不多建。
func TestSessionNewSameNameIsIdempotent(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	live := idleOn("s-live", s.cfg.CloneEnv("h1"))
	live.setNameInTest("录像A")
	s.mgr.sessions[live.ID] = live

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "", `{"env":"h1","ent":"99","name":"录像A"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("同名应幂等返回那一条，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var out struct {
		SessionID string `json:"sessionId"`
		Reused    bool   `json:"reused"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("应答解不开: %v", err)
	}
	if out.SessionID != live.ID {
		t.Fatalf("应落回已存在的那条（%s），实得 %q", live.ID, out.SessionID)
	}
	if !out.Reused {
		t.Error("`reused` 该是 true：调用方靠它区分“新建了”与“本来就有”")
	}
	if n := len(s.mgr.Snapshot()); n != 1 {
		t.Fatalf("同名幂等不该多出一条会话，实得 %d 条", n)
	}
}

// **本文件的核心**：新建时不许动池里别的会话（switch 会收掉空闲那条，这条不许）。
//
// 新建要走拨号，而测试桩必然拨不通（`errTestNoDial`），所以这里断言的是
// "请求走到了新建、且旧的两条原样留着"，而不是结果 200。
func TestSessionNewDoesNotTouchOtherSessions(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	busy := idleOn("s-busy", s.cfg.CloneEnv("h1"))
	busy.setNameInTest("旧名")
	busy.mu.Lock()
	busy.state = StateStopped
	busy.mu.Unlock()
	idle := idleOn("s-idle", s.cfg.CloneEnv("h1"))
	s.mgr.sessions[busy.ID] = busy
	s.mgr.sessions[idle.ID] = idle

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "", `{"env":"h1","ent":"99","name":"新录像"}`)
	if resp.StatusCode == 200 {
		t.Fatal("测试环境拨不了号，这条请求不该成功 —— 否则测不到“新建”那一段")
	}
	for _, id := range []string{busy.ID, idle.ID} {
		if s.mgr.Get(id) == nil {
			t.Fatalf("会话 %s 被收掉了：这条端点不许动池里别的会话（那是 /switch 的语义）", id)
		}
	}
}

// ENT 必填：谁都没说 → 400 + USAGE，且文案要能照做（与 requireExplicitEnt 同一份判据）。
func TestSessionNewRequiresExplicitEnt(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "", `{"env":"h1","name":"录像A"}`)
	if resp.StatusCode != 400 {
		t.Fatalf("没给 ENT 该退 400，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	ok, code, msg := envelopeOf(t, resp)
	if ok || code != "USAGE" {
		t.Fatalf("该回 ok=false + code=USAGE，实得 ok=%v code=%q", ok, code)
	}
	if !strings.Contains(msg, "--ent") {
		t.Errorf("文案没指出怎么给 ENT：%q", msg)
	}
	if n := len(s.mgr.Snapshot()); n != 0 {
		t.Errorf("被拒的请求不该留下会话，实得 %d 条", n)
	}
}

// 环境不存在 / 没给环境：两条各自说清楚，都不该先去拨号。
func TestSessionNewValidatesEnv(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	cases := []struct {
		name, body string
		status     int
		want       string
	}{
		{"没给环境", `{"ent":"99"}`, 400, "需要 env"},
		{"环境不存在", `{"env":"没有这个环境","ent":"99"}`, 404, "不存在"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "", c.body)
			if resp.StatusCode != c.status {
				t.Fatalf("该退 %d，实得 %d（%s）", c.status, resp.StatusCode, respError(t, resp))
			}
			if msg := respError(t, resp); !strings.Contains(msg, c.want) {
				t.Errorf("文案该含 %q，实得 %q", c.want, msg)
			}
		})
	}
	if n := len(s.mgr.Snapshot()); n != 0 {
		t.Errorf("校验失败的请求不该留下会话，实得 %d 条", n)
	}
}

// 别名取值纪律（F19）：非法名字在建连接之前就拒掉。
func TestSessionNewRejectsBadAlias(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "", `{"env":"h1","ent":"99","name":"带/斜杠"}`)
	if resp.StatusCode != 400 {
		t.Fatalf("非法别名该退 400，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	if _, code, _ := envelopeOf(t, resp); code != "USAGE" {
		t.Errorf("别名取值非法该报 USAGE，实得 %q", code)
	}
}
