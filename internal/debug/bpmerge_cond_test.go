package debug

// Q2（架构师授权，承接契约 A30/A31 的后续）：
// `syncBreakpoints` 用 `info breakpoints` 重建账本时，要把"fgldb 没回显的字段"
// 从旧账本补回来。**判据必须是"解析到的优先，旧账本只补空缺"** ——
// 修前 `Cond` 是**无条件**覆盖（`nbp.Cond = obp.Cond`），一会儿把刚解析到的
// `stop only if 1=1` 抹成空串（旧账本没条件时），一会儿把新条件换成旧条件。
//
// 这里钉的是那条判据本身（纯函数 `inheritMissingBPFields`）：`syncBreakpoints`
// 的主流程要真 SSH PTY + 真 fgldb（见 README 的"故意不覆盖"），
// 但"该不该用旧账本的值"这个决定已经**全部**在纯函数里，不需要起会话。

import "testing"

// bpMergeOld 造一份旧账本（`s.bps` 的形状：Num → 断点）。
func bpMergeOld(bps ...Breakpoint) map[int]*Breakpoint {
	m := make(map[int]*Breakpoint, len(bps))
	for i := range bps {
		b := bps[i]
		m[b.Num] = &b
	}
	return m
}

// ① 旧账本 Cond 非空、解析到的 Cond 也非空 ⇒ **必须是解析到的那份**。
//
// 修前的写法在这里失败：把 `stop only if 2=2`（fgldb 现在真的回显了，A30）
// 覆盖成旧账本里的 `1=1` —— 界面与落盘会一直显示一个早就改掉的条件。
func TestInheritMissingBPFieldsKeepsParsedCond(t *testing.T) {
	parsed := []Breakpoint{{
		Num: 2, Kind: BPKindBreak, File: "axm_axmt500_wf.4gl", Line: 2141,
		Enabled: true, Cond: "2=2",
	}}
	old := bpMergeOld(Breakpoint{
		Num: 7, Kind: BPKindBreak, File: "axm_axmt500_wf.4gl", Line: 2141, Cond: "1=1",
	})

	inheritMissingBPFields(parsed, old)

	if parsed[0].Cond != "2=2" {
		t.Fatalf("解析到的 Cond 必须优先（旧账本只补空缺）：实得 %q，想要 2=2 —— "+
			"修前这里是无条件覆盖 `nbp.Cond = obp.Cond`，A30 的回显白读了", parsed[0].Cond)
	}
}

// ② 旧账本 Cond 非空、解析到的 Cond 为空 ⇒ 允许（且应该）用旧账本补。
// 为什么还要这条：真机上 `info breakpoints` 的条件回显是 A30 才确认的，
// 老引擎/老会话可能不回显；那种时候账本里的条件不能丢。
func TestInheritMissingBPFieldsFillsEmptyCond(t *testing.T) {
	parsed := []Breakpoint{{
		Num: 2, Kind: BPKindBreak, File: "axm_axmt500_wf.4gl", Line: 2141, Enabled: true,
	}}
	old := bpMergeOld(Breakpoint{
		Num: 2, Kind: BPKindBreak, File: "axm_axmt500_wf.4gl", Line: 2141, Cond: "1=1",
	})

	inheritMissingBPFields(parsed, old)

	if parsed[0].Cond != "1=1" {
		t.Fatalf("没回显时必须从旧账本补回来：实得 %q", parsed[0].Cond)
	}
}

// ③ Ignore 的判据同向自查（架构师要求的附加自查）：解析到非 0 优先，只有解析到 0
// （= 没有 `ignore next N hits` 续行）才用旧账本补。修前这里就已经是单向的，
// 这条把它钉住，免得下一次改动把它也写成无条件覆盖。
func TestInheritMissingBPFieldsIgnoreIsParsedFirst(t *testing.T) {
	parsed := []Breakpoint{
		{Num: 2, Kind: BPKindBreak, File: "a.4gl", Line: 10, Ignore: 5},
		{Num: 3, Kind: BPKindBreak, File: "a.4gl", Line: 20},
	}
	old := bpMergeOld(
		Breakpoint{Num: 2, Kind: BPKindBreak, File: "a.4gl", Line: 10, Ignore: 1, Cond: "x=1"},
		Breakpoint{Num: 3, Kind: BPKindBreak, File: "a.4gl", Line: 20, Ignore: 2},
	)

	inheritMissingBPFields(parsed, old)

	if parsed[0].Ignore != 5 {
		t.Fatalf("解析到的 Ignore 必须优先：实得 %d，想要 5", parsed[0].Ignore)
	}
	if parsed[1].Ignore != 2 {
		t.Fatalf("解析到 0 时该用旧账本补：实得 %d，想要 2", parsed[1].Ignore)
	}
	// Cond 也是一样的方向：非空优先
	if parsed[0].Cond != "x=1" {
		t.Fatalf("解析到空 Cond 时该从旧账本补：实得 %q", parsed[0].Cond)
	}
}

// ④ 老存档的 `Kind` 为空串（`Breakpoint.Kind` 的注释写着"空 = break"）也要能对上：
// 否则从旧存档恢复出来的账本**永远**补不到 Cond/Ignore（一次静默丢条件）。
func TestInheritMissingBPFieldsMatchesLegacyEmptyKind(t *testing.T) {
	parsed := []Breakpoint{{
		Num: 2, Kind: BPKindBreak, File: "asf_bsft001_wf.4gl", Line: 4452, Enabled: true,
	}}
	old := bpMergeOld(Breakpoint{
		Num: 2, Kind: "", File: "asf_bsft001_wf.4gl", Line: 4452, Cond: "g_x=1", Ignore: 3,
	})

	inheritMissingBPFields(parsed, old)

	if parsed[0].Cond != "g_x=1" || parsed[0].Ignore != 3 {
		t.Fatalf("Kind 为空的老账本（= break）必须能对上：%+v", parsed[0])
	}
}

// ⑤ 位置不同（file/line）或种类不同（watch vs break）时**不许串味**。
func TestInheritMissingBPFieldsDoesNotBleedAcrossLocations(t *testing.T) {
	parsed := []Breakpoint{
		{Num: 2, Kind: BPKindBreak, File: "a.4gl", Line: 10},
		{Num: 3, Kind: BPKindBreak, File: "a.4gl", Line: 20},
		{Num: 4, Kind: BPKindWatch, Func: "g_current_row"},
	}
	old := bpMergeOld(
		Breakpoint{Num: 9, Kind: BPKindBreak, File: "a.4gl", Line: 30, Cond: "别的行", Ignore: 9},
		Breakpoint{Num: 8, Kind: BPKindWatch, File: "a.4gl", Line: 10, Cond: "监视点才有的"},
	)

	inheritMissingBPFields(parsed, old)

	for _, b := range parsed {
		if b.Cond != "" || b.Ignore != 0 {
			t.Fatalf("不相干的位置/种类不许把值串过来：%+v", b)
		}
	}
}

// ⑥ 重复位置（同 file:line:kind 在旧账本里有多条）时结果必须**确定**：
// 按 Num 取最小的那条（不是 map 遍历顺序 —— 那会让"同一个报文两次同步得到两个账本"）。
func TestInheritMissingBPFieldsIsDeterministic(t *testing.T) {
	parsed := []Breakpoint{{Num: 2, Kind: BPKindBreak, File: "a.4gl", Line: 10}}
	old := bpMergeOld(
		Breakpoint{Num: 9, Kind: BPKindBreak, File: "a.4gl", Line: 10, Cond: "大的", Ignore: 9},
		Breakpoint{Num: 1, Kind: BPKindBreak, File: "a.4gl", Line: 10, Cond: "小的", Ignore: 1},
	)
	for i := 0; i < 20; i++ {
		parsed[0].Cond, parsed[0].Ignore = "", 0
		inheritMissingBPFields(parsed, old)
		if parsed[0].Cond != "小的" || parsed[0].Ignore != 1 {
			t.Fatalf("第 %d 次合并用了 Num 大的那条（%q/%d）—— 合并结果必须与 map 遍历顺序无关",
				i+1, parsed[0].Cond, parsed[0].Ignore)
		}
	}
}

// ⑦ 两个监视点的 File/Line 都是空的（位置就是表达式本身）⇒ 必须按表达式区分。
// 不区分的话：`watch g_a` 的 `ignore` 次数会跑到 `watch g_b` 头上（A27 的原样报文里
// 监视点就带 `ignore next 1 hits`，所以这不是假想的形状）。
func TestInheritMissingBPFieldsKeepsWatchpointsApart(t *testing.T) {
	parsed := []Breakpoint{
		{Num: 1, Kind: BPKindWatch, Func: "g_current_row"},
		{Num: 2, Kind: BPKindWatch, Func: "g_total"},
	}
	old := bpMergeOld(
		Breakpoint{Num: 1, Kind: BPKindWatch, Func: "g_current_row", Ignore: 1},
	)
	inheritMissingBPFields(parsed, old)

	if parsed[0].Ignore != 1 {
		t.Fatalf("同一个表达式该补到 ignore：%+v", parsed[0])
	}
	if parsed[1].Ignore != 0 {
		t.Fatalf("另一个监视点（g_total）不许继承 g_current_row 的 ignore：%+v", parsed[1])
	}
}

// ⑧ 空参数不炸（syncBreakpoints 首次同步时旧账本是空的）。
func TestInheritMissingBPFieldsHandlesEmpty(t *testing.T) {
	inheritMissingBPFields(nil, nil) // 不许 panic
	parsed := []Breakpoint{{Num: 1, File: "a.4gl", Line: 1, Cond: "1=1"}}
	inheritMissingBPFields(parsed, nil)
	if parsed[0].Cond != "1=1" {
		t.Fatalf("没有旧账本时解析结果原样保留：%+v", parsed[0])
	}
}
