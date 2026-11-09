package debug

// 三类能力 × 模式 × 角色（W7）。
//
// 为什么要有这一组：收尾（CapOwn）任何时候都得放行（H5：收不掉比收错了更糟），
// 编排（CapSetup）与交互（CapInteractive）按模式判；平台自己的进程（ActorSystem）
// 不受模式约束 —— 少了这一档，平台子进程会被自己的闸门挡下。
// 真值表在这里钉死；端点行为另用一条请求钉住 —— 光有真值表不够，得证明闸门接在端点上。

import (
	"net/http"
	"testing"
)

// capSession 造一条夹具会话：指定模式。
func capSession(id, mode string) *Session {
	s := idleOn(id, envCfg("恒烁正式区"))
	// 夹具要**非 Bare**（Bare = prog/module 都空，是"还没有作业的空宿主"，模式闸门对它放行）。
	// 真值表判的是"有作业的会话"，所以给它一个作业名。
	s.Prog = "apmt520"
	s.SetMode(mode)
	return s
}

// 真值表：mode × actor × capability。
//
// 规则（W7）：
//   - CapOwn 任何时候都放行；
//   - 其余按模式：solo 只有人可写、collab 只有 ai 可写；
//   - ActorSystem（平台进程）不受模式约束。
func TestMayDoTruthTable(t *testing.T) {
	cases := []struct {
		mode, actor string
		cap         Capability
		want        bool
		why         string
	}{
		// CapOwn：谁都收得掉（收不掉比收错了更糟，H5）
		{ModeSolo, "human", CapOwn, true, "solo 下人可以收尾"},
		{ModeSolo, "ai", CapOwn, true, "solo 下 AI 也可以收尾（收尾不受模式闸门）"},
		{ModeCollab, "human", CapOwn, true, "collab 下人也收得掉"},
		{ModeCollab, "ai", CapOwn, true, "collab 下 AI 也收得掉"},

		// CapSetup / CapInteractive：按模式
		{ModeSolo, "human", CapSetup, true, "solo 下人可以起作业"},
		{ModeSolo, "ai", CapSetup, false, "solo 下 AI 不能"},
		{ModeCollab, "ai", CapSetup, true, "collab 下 AI 可以"},
		{ModeSolo, "human", CapInteractive, true, "solo 下人可写"},
		{ModeCollab, "ai", CapInteractive, true, "collab 下 AI 可写"},
		{ModeCollab, "human", CapInteractive, false, "collab 下人只读"},

		// 平台自己（ActorSystem）：不受模式约束。
		{ModeSolo, ActorSystem, CapInteractive, true, "平台进程要放行作业"},
		{ModeCollab, ActorSystem, CapInteractive, true, "同上"},
		{ModeCollab, ActorSystem, CapSetup, true, "平台进程在协作模式下也要能起作业"},
	}
	for _, c := range cases {
		s := capSession(c.mode+"-"+c.actor, c.mode)
		if got := s.MayDo(c.cap, c.actor); got != c.want {
			t.Errorf("MayDo(mode=%s actor=%s cap=%d) = %v，该是 %v —— %s",
				c.mode, c.actor, c.cap, got, c.want, c.why)
		}
	}
}

// debug 会话的交互式闸门保持原样（H2）：solo + 人 → 放行。
func TestDebugSessionKeepsInteractiveGate(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	sess := idleOn("s-dbg", s.cfg.CloneEnv("h1"))
	s.mgr.sessions[sess.ID] = sess

	resp := doReq(t, "POST", hs.URL+"/api/sessions/"+sess.ID+"/control", "", `{"action":"continue"}`)
	if resp.StatusCode == http.StatusForbidden {
		t.Fatalf("debug 会话 + solo + 人不该被拒：%s", respError(t, resp))
	}
}
