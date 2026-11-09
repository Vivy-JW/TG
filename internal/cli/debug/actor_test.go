package debug

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TT_ACTOR 是"这次调用算谁的动作"的开关：默认 ai（命令行本来就是 AI 的接口），
// 而**人从界面发起的动作**（平台子进程）会把它设成 human。
//
// 事故背景（实测 2026-10-16）：用户点界面上的按钮，报错信息里躺着
// "纯人工模式:AI 不可操作" —— 那条链路是"serve 起 tt 子进程 → 子进程调调试线 REST"，
// 子进程默认声明 X-Actor: ai，于是被自己设的模式闸门拦下。这条测试钉两件事：
//  1. 不带 TT_ACTOR 时仍然声明 ai（AI 那条既有行为不能变）；
//  2. 带 TT_ACTOR=human 时真的把 human 发出去（这才是修复本身）。
func TestActorHeaderFollowsEnv(t *testing.T) {
	var got string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-Actor")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	oldURL := dbgAPIURL
	dbgAPIURL = ts.URL
	t.Cleanup(func() { dbgAPIURL = oldURL })

	// 空串算"没设"（actorSelf 会 TrimSpace 后回落到 ai），不必真的 Unsetenv
	t.Run("默认 ai", func(t *testing.T) {
		t.Setenv("TT_ACTOR", "")
		got = ""
		if _, err := dbgAPI("GET", "/api/status", nil); err != nil {
			t.Fatalf("调用失败: %v", err)
		}
		if got != "ai" {
			t.Fatalf("没设 TT_ACTOR 时应声明 ai，实际 %q", got)
		}
	})

	t.Run("界面发起的链路声明 human", func(t *testing.T) {
		t.Setenv("TT_ACTOR", "human")
		got = ""
		if _, err := dbgAPI("GET", "/api/status", nil); err != nil {
			t.Fatalf("调用失败: %v", err)
		}
		if got != "human" {
			t.Fatalf("TT_ACTOR=human 时应声明 human，实际 %q", got)
		}
	})
}
