package debug

// 缺陷 #3 / A26 的现场半：`asf_` 前缀从哪来、`break asf_…:行` 为什么必然
// `No source file named`（.agents/plan/L1-结果-2026-10-18.md §5.1 / §7-3）。
//
// 结论（读代码得出的，不是猜的）：
//   - `Session.Module` 在**登录后**由 resolveModule（session.go）按 `<作业>.42r`
//     在各模块 42r 目录里搜出来 —— 那是唯一权威来源；
//   - 但 CLI 的 `--module` 默认值写死 "asf"（debugctl.go），于是权威解析被整条短路，
//     入口停站的文件名就成了 `asf_<作业>.4gl`，而真名是 `axm_axmt500_wf.4gl`。
//
// 所以修法不是"把 asf 换成 axm"（那对 bsft001_wf 会换来另一个错名），而是：
// ① 不再提供写死的默认模块；② 用**会话已知事实**（权威模块 / 本会话作业名）
// 去修正用户写错前缀的位置；事实不足就**明确说取不到**，并告诉他怎么给（A15 不猜）。

import (
	"strings"
	"testing"
)

// TestResolveLocationFixesWrongModule 给 asf_<本会话作业>.4gl:行 ⇒ 换成权威模块前缀。
func TestResolveLocationFixesWrongModule(t *testing.T) {
	c := BPLocContext{Module: "axm", Progs: []string{"axmt500_wf"}}
	fixed, note := c.ResolveLocation("asf_axmt500_wf.4gl:2141")
	if fixed != "axm_axmt500_wf.4gl:2141" {
		t.Fatalf("修正后 = %q，想 axm_axmt500_wf.4gl:2141", fixed)
	}
	if !strings.Contains(note, "asf") || !strings.Contains(note, "axm") {
		t.Errorf("说明里该写清改了什么：%q", note)
	}
}

// TestResolveLocationKeepsRightModule 前缀本来就对（bsft001_wf 真的在 asf 模块里）⇒ 一字不改。
// 这条是"别猜"的护栏：把 asf 无脑换成别的名字会把本来对的断点改坏。
func TestResolveLocationKeepsRightModule(t *testing.T) {
	c := BPLocContext{Module: "asf", Progs: []string{"bsft001_wf"}}
	for _, loc := range []string{"asf_bsft001_wf.4gl:4452", "asf_bsft001_wf.4gl"} {
		fixed, note := c.ResolveLocation(loc)
		if fixed != loc || note != "" {
			t.Errorf("%q 不该被改：得 (%q, %q)", loc, fixed, note)
		}
	}
}

// TestResolveLocationNoFactsDoesNotGuess 模块未知（登录后也没解析出来）⇒ **不改**，
// 并且说明里要给可操作的办法（A15：取不到就说取不到，不猜）。
func TestResolveLocationNoFactsDoesNotGuess(t *testing.T) {
	c := BPLocContext{Progs: []string{"axmt500_wf"}} // Module 空
	fixed, note := c.ResolveLocation("asf_axmt500_wf.4gl:2141")
	if fixed != "asf_axmt500_wf.4gl:2141" {
		t.Fatalf("事实不足时不许改位置，得 %q", fixed)
	}
	for _, want := range []string{"info breakpoints", "模块", "行号"} {
		if !strings.Contains(note, want) {
			t.Errorf("说明里该提到 %q（怎么给才不改错）：%q", want, note)
		}
	}
}

// TestResolveLocationIgnoresForeignFiles 不是本会话作业名的文件（公共库 lib_*）一律不碰。
func TestResolveLocationIgnoresForeignFiles(t *testing.T) {
	c := BPLocContext{Module: "axm", Progs: []string{"axmt500_wf"}}
	for _, loc := range []string{
		"lib_cl_ap_code.4gl:4593",
		"axm_axmt500_wf.4gl.orig:12",
		"2141",
		"axmt500_ui_dialog",
		"",
	} {
		if fixed, note := c.ResolveLocation(loc); fixed != loc || note != "" {
			t.Errorf("%q 不该被改：得 (%q, %q)", loc, fixed, note)
		}
	}
}

// TestResolveLocationPrefersCustModule 会话解析到客制目录时，源文件名前缀就是客制模块。
func TestResolveLocationPrefersCustModule(t *testing.T) {
	c := BPLocContext{Module: "apm", CustModule: "cpm", Progs: []string{"apmt520_wf"}}
	fixed, note := c.ResolveLocation("apm_apmt520_wf.4gl:88")
	if fixed != "cpm_apmt520_wf.4gl:88" {
		t.Fatalf("客制模块下应改成 cpm 前缀：%q（note=%q）", fixed, note)
	}
	if note == "" {
		t.Error("改了就要说明（沉默改位置是不可接受的）")
	}
}

// TestModuleForCmdlineHelp 给出"不给 --module 会怎样"的可操作口径：
// 空 = 登录后按 <作业>.42r 自动解析（权威），不是"默认 asf"。
func TestModuleFlagHelpNoHardcodedDefault(t *testing.T) {
	h := ModuleFlagHelp()
	if strings.Contains(h, "如 asf") {
		t.Errorf("帮助文本不该再举写死的模块名当默认：%q", h)
	}
	if !strings.Contains(h, "自动解析") {
		t.Errorf("帮助文本该说明空值会自动解析：%q", h)
	}
}
