package debug

// A30（契约 .agents/plan/tt-genero-formdef-contract.md §9）：
// A27 的续行不止一种。真机在**条件断点**下多打一行 `stop only if <expr>`：
//
//	2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141
//		stop only if 1=1
//		ignore next 2 hits
//
// 两句缩进续行的语义：
//
//	`stop only if <expr>`  ⇒ 并进上一条的 Cond（条件回显，A26 的 `break … if <expr>` 的回声）
//	`ignore next N hits`   ⇒ 并进上一条的 Ignore（A27 已支持）
//
// 认不出来仍按 A27：**报"行格式不认识" + 保留原文**（不许静默丢）。
//
// 本文件只喂字符串给纯函数 ParseInfoBreakpoints：不连机器、不起会话。

import (
	"strings"
	"testing"
)

// a30Sample 是契约 A30 里的那段原文（脱敏后逐字）。
var a30Sample = []string{
	"2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141",
	"\tstop only if 1=1",
	"\tignore next 2 hits",
}

// ④ A30 原文三行 → 1 个断点，Cond="1=1"、Ignore=2。
func TestParseInfoBreakpointsA30StopOnlyIf(t *testing.T) {
	bps, err := ParseInfoBreakpoints(a30Sample)
	if err != nil {
		t.Fatalf("A30 原文该解析得出来（修前：第二个续行不认识 ⇒ 断点账本重建失败）：%v", err)
	}
	if len(bps) != 1 {
		t.Fatalf("断点数 = %d，想 1：%+v", len(bps), bps)
	}
	b := bps[0]
	if b.Cond != "1=1" {
		t.Errorf("`stop only if 1=1` 没并进 Cond：%q（整个断点 %+v）", b.Cond, b)
	}
	if b.Ignore != 2 {
		t.Errorf("`ignore next 2 hits` 没并进 Ignore：%d", b.Ignore)
	}
	if b.Num != 2 || b.Kind != BPKindBreak || b.File != "axm_axmt500_wf.4gl" || b.Line != 2141 {
		t.Errorf("记录行本身的字段被续行搞坏了：%+v", b)
	}
	if b.Module != "axm_axmt500_wf.axmt500_ui_dialog" || !b.Enabled {
		t.Errorf("`in <模块>`/Enb 没读对：%+v", b)
	}
}

// ⑤ 两种续行顺序互换也认（`ignore` 在前、`stop only if` 在后）。
//
// 为什么要钉：续行的顺序是调试器实现的细节，不是契约；按"先 ignore 再 stop only if"
// 写死一个 if-else 链，换个顺序就整张表重建失败。
func TestParseInfoBreakpointsContinuationsAnyOrder(t *testing.T) {
	lines := []string{
		"2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141",
		"\tignore next 2 hits",
		"\tstop only if 1=1",
	}
	bps, err := ParseInfoBreakpoints(lines)
	if err != nil {
		t.Fatalf("两种续行互换顺序也必须认：%v", err)
	}
	if len(bps) != 1 || bps[0].Cond != "1=1" || bps[0].Ignore != 2 {
		t.Fatalf("顺序互换后解析不对：%+v", bps)
	}

	// 反过来（stop only if 在前、ignore 在后，即 A30 原文的顺序）也得是同一份账本。
	rev := []string{
		"2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141",
		"\tstop only if 1=1",
		"\tignore next 2 hits",
	}
	bps2, err := ParseInfoBreakpoints(rev)
	if err != nil {
		t.Fatalf("A30 原文顺序也必须认：%v", err)
	}
	if len(bps2) != 1 || bps2[0].Cond != "1=1" || bps2[0].Ignore != 2 {
		t.Fatalf("两种顺序该算出同一份账本，实得：%+v", bps2)
	}
}

// 空条件（`stop only if` 后面什么都没有）不是记录也不是合法续行 ⇒
// 报"行格式不认识"并保留原文（不许把 Cond 写成一个空串蒙过去）。
func TestParseInfoBreakpointsStopOnlyIfWithoutExpr(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"2 breakpoint keep y 0x00000000 in m.f at f.4gl:2141",
		"\tstop only if   ",
	})
	if err == nil {
		t.Fatalf("空条件该报错，得 %+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") {
		t.Fatalf("错误要说清是行格式问题：%v", err)
	}
	if !strings.Contains(err.Error(), "stop only if") {
		t.Fatalf("错误里必须保留原文：%v", err)
	}
}

// ⑥ 不认识的续行：报"行格式不认识" + 保留原文（A27 的另一半，不许静默丢）。
func TestParseInfoBreakpointsUnknownContinuationKeepsOriginal(t *testing.T) {
	const bad = "\tthread 2 (Thread 0x1 (LWP 4321))"
	bps, err := ParseInfoBreakpoints([]string{
		"2 breakpoint keep y 0x00000000 in m.f at f.4gl:2141",
		bad,
	})
	if err == nil {
		t.Fatalf("不认识的续行该报错，得 %+v", bps)
	}
	if len(bps) != 0 {
		t.Errorf("报错时不该给出半截结果：%+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") {
		t.Errorf("错误没说清是行格式问题：%v", err)
	}
	if !strings.Contains(err.Error(), "thread 2 (Thread 0x1 (LWP 4321)") {
		t.Errorf("错误里必须保留原文（否则现场无从下手）：%v", err)
	}
}

// 孤立的条件续行（上面没有断点记录）也算不认识 —— 与孤儿 `ignore` 同一个口径：
// 宁可报错，也不要把条件挂到别的断点上。
func TestParseInfoBreakpointsOrphanStopOnlyIf(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{"\tstop only if 1=1"})
	if err == nil {
		t.Fatalf("孤立的条件续行该报错，得 %+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") || !strings.Contains(err.Error(), "stop only if 1=1") {
		t.Fatalf("错误要带原文与「行格式不认识」：%v", err)
	}
}

// 流式路径（onLine 逐行喂）走同一条解析：三种行在那里也不许丢 Cond（A57 的现场：
// "断点账本未重建"就是从这条路上报出来的）。
func TestPendingCmdInfoBPsKeepsStopOnlyIf(t *testing.T) {
	p := &pendingCmd{kind: "breakpoints"}
	p.bpRaw = append(p.bpRaw, a30Sample...)
	bps, errText := p.infoBPs()
	if errText != "" {
		t.Fatalf("流式路径该解析得出来，得错误：%s", errText)
	}
	if len(bps) != 1 || bps[0].Cond != "1=1" || bps[0].Ignore != 2 {
		t.Fatalf("流式路径解析不对：%+v", bps)
	}
}
