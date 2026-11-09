package debug

// A33（契约 .agents/plan/tt-genero-formdef-contract.md §9 A33）：
// `info breakpoints` 的**命中计数**才是"这条断点命中过没有"的真判据。
//
// 第四次真机实测（L1-断点排查-2026-10-18.md §4.3）推翻了两条旧推论：
//
//	· 命中的断点会多一行 `breakpoint already hit N time(s)`（未命中过的**不打印**这一行）；
//	· `Address` 恒为 `0x00000000`，命中与否都是 0 ⇒ **它不是"装没装上"的证据**。
//
// 所以解析器要认**第三种续行**，并与 `stop only if …` / `ignore next N hits` 并列
// （顺序无关，都按"并进上一条"处理）。
//
// 本文件只喂字符串给纯函数 ParseInfoBreakpoints：不连机器、不起会话。

import (
	"strings"
	"testing"
)

// a33Sample 是第四次真机实测的四行原文（脱敏后逐字）：
// 断点行 + 条件回显 + 忽略回显 + 命中计数。
var a33Sample = []string{
	"1   breakpoint     keep y   0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:4130",
	"\tstop only if 1=1",
	"\tignore next 2 hits",
	"\tbreakpoint already hit 1 time",
}

// ① 两种续行 + 命中计数混在同一段原文里 ⇒ Cond/Ignore/Hits 三样都对上。
func TestParseInfoBreakpointsA33HitsAndContinuations(t *testing.T) {
	bps, err := ParseInfoBreakpoints(a33Sample)
	if err != nil {
		t.Fatalf("A33 原文该解析得出来（修前：`breakpoint already hit 1 time` 不认识 ⇒ 整张表重建失败）：%v", err)
	}
	if len(bps) != 1 {
		t.Fatalf("断点数 = %d，想 1：%+v", len(bps), bps)
	}
	b := bps[0]
	if b.Cond != "1=1" {
		t.Errorf("`stop only if 1=1` 没并进 Cond：%q", b.Cond)
	}
	if b.Ignore != 2 {
		t.Errorf("`ignore next 2 hits` 没并进 Ignore：%d", b.Ignore)
	}
	if b.Hits != 1 {
		t.Errorf("`breakpoint already hit 1 time` 没并进 Hits：%d（A33：内核给的计数不读，等于让人猜命中）", b.Hits)
	}
	if b.Num != 1 || b.File != "axm_axmt500_wf.4gl" || b.Line != 4130 {
		t.Errorf("记录行本身的字段被续行搞坏了：%+v", b)
	}
}

// ② 复数 `2 times` 也认（内核按 1 次/多次换单复数）。
func TestParseInfoBreakpointsA33HitsPlural(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"2 breakpoint keep y 0x00000000 in main at axm_axmt500_wf.4gl:1553",
		"\tbreakpoint already hit 2 times",
	})
	if err != nil {
		t.Fatalf("`breakpoint already hit 2 times` 该认得出来：%v", err)
	}
	if len(bps) != 1 || bps[0].Hits != 2 {
		t.Fatalf("复数形态没读对：%+v", bps)
	}
}

// ③ 没有那一行 ⇒ Hits = 0（"没命中过"与"解析失败"必须分得开）。
func TestParseInfoBreakpointsA33NoHitsLine(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"2 breakpoint keep y 0x00000000 in main at axm_axmt500_wf.4gl:1553",
	})
	if err != nil {
		t.Fatalf("没有命中行不该报错：%v", err)
	}
	if len(bps) != 1 || bps[0].Hits != 0 {
		t.Fatalf("没有命中行时 Hits 该是 0：%+v", bps)
	}
}

// ④ 三种续行顺序互换 ⇒ 同一份账本（顺序是调试器实现的细节，不是契约）。
func TestParseInfoBreakpointsA33AnyOrder(t *testing.T) {
	lines := []string{
		"1   breakpoint     keep y   0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:4130",
		"\tbreakpoint already hit 3 times",
		"\tignore next 2 hits",
		"\tstop only if 1=1",
	}
	bps, err := ParseInfoBreakpoints(lines)
	if err != nil {
		t.Fatalf("三种续行换顺序也必须认：%v", err)
	}
	if len(bps) != 1 {
		t.Fatalf("断点数 = %d，想 1：%+v", len(bps), bps)
	}
	if b := bps[0]; b.Cond != "1=1" || b.Ignore != 2 || b.Hits != 3 {
		t.Fatalf("顺序互换后解析不对：%+v", b)
	}
}

// ⑤ 命中计数只并进**上一条**：第一条已经命中过、第二条刚下 ⇒ 两条各不相同。
func TestParseInfoBreakpointsA33HitsPerBreakpoint(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{
		"1 breakpoint keep y 0x00000000 in main at axm_axmt500_wf.4gl:1553",
		"\tbreakpoint already hit 1 time",
		"2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:4130",
	})
	if err != nil {
		t.Fatalf("两条断点该解析得出来：%v", err)
	}
	if len(bps) != 2 {
		t.Fatalf("断点数 = %d，想 2：%+v", len(bps), bps)
	}
	if bps[0].Hits != 1 || bps[1].Hits != 0 {
		t.Fatalf("命中计数串到别的断点上了：%+v", bps)
	}
}

// ⑥ 孤立的命中续行（上面没有断点记录）仍按 A27 报"行格式不认识"+原文：
// 宁可报错，也不要把计数挂到别人的断点上。
func TestParseInfoBreakpointsA33OrphanHits(t *testing.T) {
	bps, err := ParseInfoBreakpoints([]string{"\tbreakpoint already hit 1 time"})
	if err == nil {
		t.Fatalf("孤立的命中续行该报错，得 %+v", bps)
	}
	if !strings.Contains(err.Error(), "行格式不认识") || !strings.Contains(err.Error(), "already hit 1 time") {
		t.Fatalf("错误要带「行格式不认识」与原文：%v", err)
	}
}

// ⑦ 流式路径（onLine 逐行喂）走同一条解析：命中计数在那里也不许丢。
func TestPendingCmdInfoBPsKeepsHits(t *testing.T) {
	p := &pendingCmd{kind: "breakpoints"}
	p.bpRaw = append(p.bpRaw, a33Sample...)
	bps, errText := p.infoBPs()
	if errText != "" {
		t.Fatalf("流式路径该解析得出来，得错误：%s", errText)
	}
	if len(bps) != 1 || bps[0].Hits != 1 {
		t.Fatalf("流式路径丢了命中计数：%+v", bps)
	}
}
