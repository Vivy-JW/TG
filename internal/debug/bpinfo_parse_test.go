package debug

// A27（契约 .agents/plan/tt-genero-formdef-contract.md §8 A27）：
// `info breakpoints` 的**真实行格式**——来自
// .agents/plan/L1-结果-2026-10-18.md §5.2 的主机109 原始输出。
//
// 三条实测事实（不是推测）：
//  1. watchpoint 行**没有 `hw`** 字样、Address 列为空；
//  2. breakpoint 行形如 `0x00000000 in <函数> at <文件>:<行>`；
//  3. `ignore` 的回显是**独立的第二行**（制表符缩进 `ignore next N hits`）。
//
// 本文件只喂字符串给纯函数 ParseInfoBreakpoints：不连机器、不起会话。

import (
	"strings"
	"testing"
)

// a27Sample 是契约 A27 里的那段原文（脱敏后逐字）。
var a27Sample = []string{
	"Num Type           Disp Enb Address    What",
	"1 watchpoint keep y  g_current_row",
	"2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141",
	"\tignore next 1 hits",
}

// TestParseInfoBreakpointsA27 契约 A27 要求的**最小往返**：
// 三行原文（含列头）→ 2 个断点，第 2 个带 Ignore=1。
func TestParseInfoBreakpointsA27(t *testing.T) {
	bps, err := ParseInfoBreakpoints(a27Sample)
	if err != nil {
		t.Fatalf("A27 原文该解析得出来，得错误：%v", err)
	}
	if len(bps) != 2 {
		t.Fatalf("断点数 = %d，想 2：%+v", len(bps), bps)
	}

	w := bps[0]
	if w.Num != 1 || w.Kind != BPKindWatch {
		t.Errorf("第 1 条该是 watchpoint #1，得 num=%d kind=%q", w.Num, w.Kind)
	}
	if !w.Enabled {
		t.Errorf("watchpoint 行 Enb=y，Enabled 该为 true：%+v", w)
	}
	// 监视点没有 file:line，被监视的表达式落在 Func 里（落盘/恢复靠它）。
	if w.Func != "g_current_row" || w.File != "" || w.Line != 0 {
		t.Errorf("watchpoint 的表达式/位置不对：%+v", w)
	}

	b := bps[1]
	if b.Num != 2 || b.Kind != BPKindBreak {
		t.Errorf("第 2 条该是 breakpoint #2，得 num=%d kind=%q", b.Num, b.Kind)
	}
	if b.File != "axm_axmt500_wf.4gl" || b.Line != 2141 {
		t.Errorf("breakpoint 的 file:line 不对：%+v", b)
	}
	if b.Module != "axm_axmt500_wf.axmt500_ui_dialog" {
		t.Errorf("`in <module>` 没取出来：%q", b.Module)
	}
	// 第二行的 `ignore next 1 hits` 必须并入上一条断点。
	if b.Ignore != 1 {
		t.Errorf("ignore next 1 hits 没并进第 2 条：Ignore=%d", b.Ignore)
	}
	if !b.Enabled {
		t.Errorf("Enb=y 该为 Enabled=true：%+v", b)
	}
}

// TestParseInfoBreakpointsL1Original L1 实测的四条记录（两条 ignore 第二行）：
// 第 2 条 Ignore=2、第 3 条 Ignore=1，第 1 条（watchpoint）Ignore=0。
func TestParseInfoBreakpointsL1Original(t *testing.T) {
	lines := []string{
		"Num Type           Disp Enb Address    What",
		"1   watchpoint     keep y              g_current_row",
		"2   breakpoint     keep y   0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141",
		"\tignore next 2 hits",
		"3   breakpoint     keep y   0x00000000 in axm_axmt500_wf.axmt500_construct at axm_axmt500_wf.4gl:5043",
		"\tignore next 1 hits",
	}
	bps, err := ParseInfoBreakpoints(lines)
	if err != nil {
		t.Fatalf("L1 原文该解析得出来：%v", err)
	}
	if len(bps) != 3 {
		t.Fatalf("断点数 = %d，想 3：%+v", len(bps), bps)
	}
	if bps[0].Ignore != 0 {
		t.Errorf("第 1 条（watch）不该有 ignore：%+v", bps[0])
	}
	if bps[1].Ignore != 2 || bps[1].Line != 2141 {
		t.Errorf("第 2 条不对：%+v", bps[1])
	}
	if bps[2].Ignore != 1 || bps[2].Line != 5043 {
		t.Errorf("第 3 条不对：%+v", bps[2])
	}
	if bps[2].Module != "axm_axmt500_wf.axmt500_construct" {
		t.Errorf("第 3 条的 module 不对：%q", bps[2].Module)
	}
}

// TestParseInfoBreakpointsToleratesHW 契约说**不得依赖** `hw`，所以带 `hw` 的老格式
// 也得认（本机不打印它，但别把兼容性焊死在一个不存在的前缀上）。
func TestParseInfoBreakpointsToleratesHW(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{"4\thw watchpoint\tkeep\ty\t\tg_total"})
	if err != nil {
		t.Fatalf("带 hw 的 watchpoint 也该认：%v", err)
	}
	if len(bps) != 1 || bps[0].Kind != BPKindWatch || bps[0].Func != "g_total" {
		t.Fatalf("解析不对：%+v", bps)
	}
}

// TestParseInfoBreakpointsSkipsNonRecords 列头 / 命令回显 / 空行不是记录，也不算错误。
func TestParseInfoBreakpointsSkipsNonRecords(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"info breakpoints",
		"",
		"Num Type           Disp Enb Address    What",
		"",
	})
	if err != nil {
		t.Fatalf("非记录行不该报错：%v", err)
	}
	if len(bps) != 0 {
		t.Fatalf("没有记录，得 %d 条：%+v", len(bps), bps)
	}
}

// TestParseInfoBreakpointsUnknownFormat **不认识就报错并保留原文**（不许静默吞掉）。
// 这是 A27 的另一半：解析不出来的时候，人得能从错误里看到那一行到底长什么样。
func TestParseInfoBreakpointsUnknownFormat(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"1   watchpoint     keep y              g_current_row",
		"??? garbage line that no one can read ???",
	})
	if err == nil {
		t.Fatalf("不认识的格式该报错，得 %+v", bps)
	}
	if len(bps) != 0 {
		t.Errorf("报错时不该给出半截结果：%+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") {
		t.Errorf("错误没说清是行格式问题：%v", err)
	}
	if !strings.Contains(err.Error(), "??? garbage line that no one can read ???") {
		t.Errorf("错误里必须保留原文（否则现场无从下手）：%v", err)
	}
}

// TestParseInfoBreakpointsOrphanIgnore 孤立的 `ignore next N hits`（上一条记录不存在）
// 也算不认识——宁可报错，也不要把 ignore 次数挂到别的断点上。
func TestParseInfoBreakpointsOrphanIgnore(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{"\tignore next 3 hits"})
	if err == nil {
		t.Fatalf("孤立 ignore 行该报错，得 %+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") {
		t.Errorf("错误没说清是行格式问题：%v", err)
	}
}

// TestPendingCmdInfoBPs 会话内的流式路径（onLine 逐行喂）走同一条解析：
// 结果是同一个函数算出来的，所以两行形态在那里也不会丢 ignore。
func TestPendingCmdInfoBPs(t *testing.T) {
	p := &pendingCmd{kind: "breakpoints"}
	for _, ln := range a27Sample {
		p.bpRaw = append(p.bpRaw, ln)
	}
	bps, errText := p.infoBPs()
	if errText != "" {
		t.Fatalf("该解析得出来，得错误：%s", errText)
	}
	if len(bps) != 2 || bps[1].Ignore != 1 {
		t.Fatalf("流式路径解析不对：%+v", bps)
	}
	// 解析失败时错误要留在 errText 里（调用方据此报"行格式不认识"并保留原文）。
	bad := &pendingCmd{kind: "breakpoints", bpRaw: []string{"??? nope ???"}}
	if _, e := bad.infoBPs(); !strings.Contains(e, "行格式不认识") {
		t.Fatalf("解析失败该把原文错误带回 errText，得 %q", e)
	}
}
