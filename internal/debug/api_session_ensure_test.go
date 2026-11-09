package debug

// **按别名点名会话**的语义（真机 2026-10-18 13:15 的现场）。
//
// 现场：环境里有 2 条会话，调用方拿别名去"点名环境"的那一步走的是
// `/api/sessions/switch` —— 那个端点的语义是**"把这个环境切到某一条"**，
// 于是"池里有 2 条"当场 SESSION_AMBIGUOUS，别名一个字节都没看。
//
// 修法：这类调用改用 `/api/sessions/new` —— 它的语义正是「同名幂等、异名新建、
// **绝不收口别人**」（D6：switch 可回收，new 不收口）。
//
// 这一组钉两件事：
//  1. 池里有两条时，点名已有别名 → **复用那一条**，另一条不动、也不拨号；
//  2. 点名一个新别名 → 新建，已有两条都还在。
//
// 老的 `switch` 语义（同名幂等 / 异名重建 / 忙时拒 / 多条下仍报歧义）由既有的
// `TestSwitch*` 继续守着 —— 这次**没有**动它。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newEnsuredResp 解 `/sessions/new` 的应答。
func newEnsuredResp(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("应答不是合法 JSON：%v", err)
	}
	return out
}

// manyFixture 造两条同环境、各有名字的空闲会话。
func manyFixture(t *testing.T) (*Server, *httptest.Server, *Session, *Session) {
	t.Helper()
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	a := idleOn("s-a", s.cfg.CloneEnv("h1"))
	a.setNameInTest("会话甲")
	b := idleOn("s-b", s.cfg.CloneEnv("h1"))
	b.setNameInTest("会话乙")
	s.mgr.sessions[a.ID] = a
	s.mgr.sessions[b.ID] = b
	return s, hs, a, b
}

// 池里有两条时，点名已有别名 → 复用那一条，另一条不动，**不拨号**。
func TestSessionNewEnsuresNamedSessionAmongMany(t *testing.T) {
	s, hs, a, b := manyFixture(t)

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "",
		`{"env":"h1","ent":"99","name":"会话乙","purpose":"debug"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("点名已有会话应 200（幂等），实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	out := newEnsuredResp(t, resp)
	if got := out["sessionId"]; got != b.ID {
		t.Fatalf("该复用「会话乙」(%s)，实得 %v", b.ID, got)
	}
	if reused, _ := out["reused"].(bool); !reused {
		t.Error("幂等复用要回 reused=true（调用方靠它区分「新建」与「就是那条」）")
	}
	if s.mgr.Get(a.ID) == nil {
		t.Fatal("点名一条会话时，同环境另一条不该被收掉")
	}
}

// 点名一个**新的**别名 → 新建；已有的两条都还在（多会话的意义就在这儿）。
func TestSessionNewCreatesNewNameWithoutTouchingOthers(t *testing.T) {
	s, hs, a, b := manyFixture(t)
	made := 0
	s.mgr.newSession = func(cfg *Config, _, _, _, _, _ string, _ func(Event)) (*Session, error) {
		made++
		return idleOn("s-new", cfg), nil
	}

	resp := doReq(t, "POST", hs.URL+"/api/sessions/new", "",
		`{"env":"h1","ent":"99","name":"会话丙","purpose":"debug"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("点名新别名应 200（新建），实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	out := newEnsuredResp(t, resp)
	id, _ := out["sessionId"].(string)
	if id == "" || id == a.ID || id == b.ID {
		t.Fatalf("该新建一条，实得 sessionId=%v", out["sessionId"])
	}
	if made != 1 {
		t.Fatalf("该只建一条，实际建了 %d 条", made)
	}
	if s.mgr.Get(a.ID) == nil || s.mgr.Get(b.ID) == nil {
		t.Fatal("点名新别名时不该收掉已有会话")
	}
}
