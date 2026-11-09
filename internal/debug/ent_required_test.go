package debug

// 企业编号(ENT)**必填**的判据（requireExplicitEnt）。
//
// 这是本轮的行为变更：从前不给 ENT 会**静默**沿用登录 shell 的 `$TOPENT`，于是"忘了点名"
// 与"点错了"在输出上完全一样 —— 作业照跑，只是跑在另一家企业上，而 ENT 决定用哪个账号
// 连库、以及据点校验过不过。现在两个合法来源之外一律拒绝：
//
//   - 本次启动显式给的 `--ent`（落在 cfg.Topent）；
//   - 这条会话此前显式定下的企业编号（`tt debug topent` 的 override，或它建立时的 cfg.Topent）。

import (
	"errors"
	"strings"
	"testing"

	"tt/internal/host"
	"tt/internal/target"
)

// TestRequireExplicitEntRejectsSilentFallback：谁都没说 → **必须拒绝**，且码是 USAGE。
//
// 这一条就是"移除静默行为"的判据：从前它会返回 nil（然后 export 都不发，shell 那个
// 默认值就悄悄生效了）。
func TestRequireExplicitEntRejectsSilentFallback(t *testing.T) {
	cfg := envCfg("恒烁正式区")
	cfg.Topent = "" // 没人点名

	err := requireExplicitEnt(cfg, nil)
	if err == nil {
		t.Fatal("没给 ENT 时必须拒绝，不许静默沿用登录 shell 的 $TOPENT")
	}
	var f *target.Failure
	if !errors.As(err, &f) || f.Code != target.CodeUsage {
		t.Fatalf("该报 %q，实得 %T: %v", target.CodeUsage, err, err)
	}
	// 文案要能照做：说清两条合法路径，并明说不再沿用 shell 的值。
	for _, want := range []string{"--ent", "tt debug topent", "不再"} {
		if !strings.Contains(f.Message, want) {
			t.Errorf("文案里缺 %q（人要靠它知道下一步怎么给）：%q", want, f.Message)
		}
	}
}

// TestRequireExplicitEntAcceptsRequestValue：本次启动给了 `--ent` → 放行。
func TestRequireExplicitEntAcceptsRequestValue(t *testing.T) {
	cfg := envCfg("恒烁正式区")
	cfg.Topent = host.EntValue("99")
	if err := requireExplicitEnt(cfg, nil); err != nil {
		t.Fatalf("显式给了 ENT 不该被拒：%v", err)
	}
}

// TestRequireExplicitEntAcceptsSessionOverride：会话此前用 `tt debug topent` 定过 → 放行。
//
// 这是"先建设会话、再设 ENT、再用这条会话调试"那条流程能成立的前提。
func TestRequireExplicitEntAcceptsSessionOverride(t *testing.T) {
	cfg := envCfg("恒烁正式区")
	cfg.Topent = ""

	sess := idleOn("s1", cfg)
	sess.setTopentOverride("99")

	if err := requireExplicitEnt(cfg, sess); err != nil {
		t.Fatalf("会话已显式定过 ENT，不该被拒：%v", err)
	}
}

// TestRequireExplicitEntAcceptsSessionBuildEnt：会话建立时就带了企业编号 → 放行。
func TestRequireExplicitEntAcceptsSessionBuildEnt(t *testing.T) {
	reqCfg := envCfg("恒烁正式区")
	reqCfg.Topent = "" // 本次请求没给

	// 这条会话**建立时**就带着显式企业编号（99）—— 注意它必须是一份独立的配置，
	// 否则改了请求那一份就等于把会话那一份也改了，测不到想测的东西。
	built := envCfg("恒烁正式区")
	sess := idleOn("s1", built)

	if err := requireExplicitEnt(reqCfg, sess); err != nil {
		t.Fatalf("会话自身带着显式企业编号，不该被拒：%v", err)
	}
}

// TestPrepareSessionWithoutEntIsRejected：端到端 —— 起会话那条路真的会拒绝（不只是纯函数）。
func TestPrepareSessionWithoutEntIsRejected(t *testing.T) {
	cfg := envCfg("恒烁正式区")
	cfg.Topent = ""

	m := NewManager(cfg)
	if _, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", ""); err == nil {
		t.Fatal("不给 ENT 起会话必须被拒")
	} else {
		var f *target.Failure
		if !errors.As(err, &f) || f.Code != target.CodeUsage {
			t.Fatalf("该报 %q，实得 %T: %v", target.CodeUsage, err, err)
		}
	}
	if n := len(m.Snapshot()); n != 0 {
		t.Errorf("被拒的请求不该留下会话，实得 %d 条", n)
	}
}
