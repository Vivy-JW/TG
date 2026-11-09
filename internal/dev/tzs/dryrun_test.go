package tzs

// dryrun_test.go —— `dry_run` 的回滚判据，在**每一份真实语料包**上跑。
//
// 判据只有一条，但它必须是这一条：**save → dry-run → save，两次产出的 sha256 相同**。
// 换句话说：干跑之后，这个句柄保存出来的东西，与干跑之前保存出来的东西逐字节一样。
//
// 为什么不逐字段比对：能观察模型的路径只有"渲染成文本"（save 的 .4fd/.tsd/.bdx 三件事，
// 见 Save.Run），所以"逐字节一样"就是模型没变的**定义**，而且是可机械复现的那个定义。
//
// 为什么还要跑一次真的写（非空转保证）：如果那个动词在这个包上什么都没做，上面那条判据
// 会全绿而它什么都没证明 —— 与 corpus_test.go 里 mustDiffer 是同一条纪律。所以最后再用
// 同样的参数**真写一次**，产出的 sha 必须变。
//
// 为什么这条判据是这个样子而不是"撤销栈回到了调用前的高度"（2026-09-26 的教训）：
// 落法原本记的是"应用 → 读 diff → 批量 undo，撤销栈是现成的"。实测**不成立** —— 撤销栈
// 管的是设计师的 command，而写动词有相当一部分写在 command 之外（UICreator 就是工厂而不是
// 命令，它建进模型的规格节点从来没进过栈）。十五个动词跑这一条判据，只有四个纯布局的
// 逐字节回得去，其余各留各的残留（set_spec_attr 多一个 status="u"，add_field 多出整套
// 规格节点与一条 TBinding）。回滚因此改成"把模型渲染成临时包、再从它重读回来"，而这一条
// 判据就是那个改动的验收 —— 它对**每个**动词都成立，不是逐个动词碰运气。
//
// 它还顺带守住两件容易漏的小事：
//   · 引擎的临时包必须自己清掉（_tt_dry_*.tzs 不能留给下一次语料遍历）；
//   · 干跑之后的句柄仍然可用（回滚会重建会话，句柄串不许换）。

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// dryReply 是 dry-run 信封（Rpc 的 DryRun.Envelope）。
type dryReply struct {
	DryRun   bool   `json:"dryRun"`
	Verb     string `json:"verb"`
	Handle   string `json:"handle"`
	Preview  any    `json:"preview"`
	Note     string `json:"note"`
	Reverted struct {
		Mode      string  `json:"mode"`
		Complete  bool    `json:"complete"`
		RenderMs  float64 `json:"renderMs"`
		LoadMs    float64 `json:"loadMs"`
		Sha256    string  `json:"sha256"`
		Error     string  `json:"error"`
		Scratch   string  `json:"scratch"`
		Recovered bool    `json:"recovered"`
	} `json:"reverted"`
}

// driedUp 清掉一个目录下所有引擎临时包，并返回清掉的数量。
//
// 引擎自己会删（DryRun.Scope.Rollback），所以正常情况下这里是 0 —— 这一句是"没留下东西"
// 的机械检查，而不是清理。真删不掉的（进程崩在回滚中间）也要在这里收掉，否则它会跟着
// 语料遍历跑到下一个工作区去。
func driedUp(t *testing.T, dir string) int {
	t.Helper()
	hits, _ := filepath.Glob(filepath.Join(dir, "_tt_dry_*.tzs"))
	for _, p := range hits {
		_ = os.Remove(p)
	}
	return len(hits)
}

func TestCorpusDryRunRollsBack(t *testing.T) {
	env := requireCorpus(t)
	checked, skipped := 0, 0
	for i, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" {
			skipped++
			continue
		}
		// 挑选不需要引擎（can_edit 在 .tsd 里），所以缺前置的包在起进程之前就能说清。
		cands, why := editCandidates(t, src)
		if why != "" {
			skipped++
			continue
		}
		label := "dry/" + filepath.Base(src)
		dir := filepath.Dir(src)
		pre := scratchPath(src, i, "dry", "pre")
		post := scratchPath(src, i, "dry", "post")
		real := scratchPath(src, i, "dry", "real")
		defer func() { _ = os.Remove(pre); _ = os.Remove(post); _ = os.Remove(real) }()

		s := startStdio(t, env.exe, ws, perFileBudget)
		o, ok := ask[openReply](t, s, label+" open", "open", map[string]any{"path": src})
		if !ok {
			s.close()
			continue
		}
		h := o.Handle
		if !saveTo(t, s, label+" save(pre)", h, pre) {
			s.close()
			continue
		}
		shaPre, err := sha16(pre)
		if err != nil {
			t.Errorf("%s: 算 pre 的 sha 失败：%v", label, err)
			s.close()
			continue
		}

		// resolveEditTarget 会自己报它为什么挑不中（引擎拒绝、路径不存在…），挑不中就跳过这一包。
		name, path, was, flip, ok := resolveEditTarget(t, s, label, h, cands)
		if !ok {
			driedUp(t, dir)
			s.close()
			skipped++
			continue
		}
		args := map[string]any{"handle": h, "path": path, "kind": "field",
			"attr": "can_edit", "value": flip}

		// ---- ① 干跑
		dryArgs := map[string]any{}
		for k, v := range args {
			dryArgs[k] = v
		}
		dryArgs["dry_run"] = true
		d, ok := ask[dryReply](t, s, label+" dry", "set_spec_attr", dryArgs)
		if !ok {
			driedUp(t, dir)
			s.close()
			continue
		}
		if !d.DryRun {
			t.Errorf("%s: dry_run:true 的调用没有回 dryRun:true（信封丢了，调用方无从分辨这是一次预演）", label)
		}
		if d.Verb != "set_spec_attr" {
			t.Errorf("%s: 信封里的 verb=%q", label, d.Verb)
		}
		if d.Handle != h {
			t.Errorf("%s: 信封里的 handle=%q，调用方手里的是 %q —— 回滚重建了会话，句柄串必须不变", label, d.Handle, h)
		}
		if d.Preview == nil {
			t.Errorf("%s: preview 是空的 —— 干跑的全部意义就是把这次调用本会返回的东西留下", label)
		}
		if !d.Reverted.Complete {
			t.Errorf("%s: 回滚没走完：%s（scratch=%s）", label, d.Reverted.Error, d.Reverted.Scratch)
		}

		// ---- ② 判据：干跑之后的产出必须与干跑之前逐字节相同
		if !saveTo(t, s, label+" save(post)", h, post) {
			driedUp(t, dir)
			s.close()
			continue
		}
		shaPost, err := sha16(post)
		if err != nil {
			t.Errorf("%s: 算 post 的 sha 失败：%v", label, err)
			s.close()
			continue
		}
		if shaPre != shaPost {
			t.Errorf("%s: 干跑改变了模型（pre=%s post=%s）—— dry_run 的承诺是「什么都没发生」。\n"+
				"  这条判据失败时先看两处：一是句柄那个 writer 有没有跟着回滚（Fns.Session.InstallLayout，"+
				"它曾经漏过：模型回来了而 .4fd 文本没回来），二是回滚到底走了哪条路（reverted.mode 只能是 session-rebuild）。",
				label, shaPre, shaPost)
		}

		// ---- ③ 非空转：同样的参数真写一次，产出必须变
		rd, ok := ask[deltaRef](t, s, label+" real", "set_spec_attr", args)
		if !ok {
			t.Errorf("%s: 真写被拒了，而干跑刚说它会成功 —— 两条路不一致", label)
			driedUp(t, dir)
			s.close()
			continue
		}
		if rd.Noop {
			t.Errorf("%s: 真写回了 noop（%q 已经是 %q）—— 上面那条判据在空转", label, rd.Attr, rd.Value)
		}
		if !saveTo(t, s, label+" save(real)", h, real) {
			driedUp(t, dir)
			s.close()
			continue
		}
		shaReal, err := sha16(real)
		if err != nil {
			t.Errorf("%s: 算 real 的 sha 失败：%v", label, err)
			s.close()
			continue
		}
		if shaReal == shaPre {
			t.Errorf("%s: 真写之后产出没变（%s）—— 这个包上的这次写是个空操作，上面第 ② 条判据什么也没证明",
				label, shaReal)
		}

		// ---- ④ 临时包不许留下
		if left := driedUp(t, dir); left > 0 {
			t.Errorf("%s: 回滚留下了 %d 个 _tt_dry_*.tzs（引擎自己该删掉；留下就会被下一次语料遍历当成包）",
				label, left)
		}
		note := ""
		if d.Reverted.Recovered {
			// 不该发生的事被兜住了：临时包在回滚前不见了，是用留底字节重写的（见 SPEC §11.24 (o)）。
			// 判据不算它失败（模型确实回来了），但一定要在报告里看得见 —— 藏起来就是让 flake 变成谜。
			note = " [recovered：临时包在回滚前丢了，用留底字节重写过]"
			t.Logf("%s: 回滚时临时包不见了，用留底字节重写后成功（这是机器的问题，不是这次请求的）", label)
		}
		t.Logf("%s: can_edit %s %s→%s 干跑回滚 OK（render=%.1fms load=%.1fms，pre=%s real=%s）%s",
			label, name, was, flip, d.Reverted.RenderMs, d.Reverted.LoadMs, shaPre[:12], shaReal[:12], note)
		checked++
		s.call("close", map[string]any{"handle": h})
		s.close()
	}
	if checked == 0 {
		t.Fatalf("一个包都没验到（跳过 %d 个）：判据或前置条件写错了，不是语料的问题", skipped)
	}
	t.Logf("干跑回滚：%d 个包逐字节回得去、%d 个跳过", checked, skipped)
}

// TestDryRunIsOnlyAdvertisedOnTracingVerbs —— `dry_run` 只该出现在**会留下痕迹**的动词上。
//
// 这条不是形式主义：它守的是"这个开关在这条命令上有意义吗"。dry_run 的语义是"做一遍但别留下"，
// 挂在读动词上就是一句空话，而调用方会以为它有意义。
//
// 关于 `op`：这里**不**做逐动词的断言，那是有意的。`op` 在写动词上是"给这次写起个名字"，
// 在 `list_ops` 上却是**过滤器**（"我要问哪个名字"）—— 同一个词、同一个概念的两个方向，
// 这不是冲突。第一版调度器按"参数里有没有 op"来认写动词，于是那次查询把自己当成一次写操作
// 记进了日志；修法是在 SpecFn 上立一个 Traced 标志（声明与行为同源），而不是按名字猜。
// 那条性质是**行为**上的，所以它的判据在 oplog_test.go 里（读动词带 op 不会长日志）。
func TestDryRunIsOnlyAdvertisedOnTracingVerbs(t *testing.T) {
	exe, _, _ := requireE2E(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m, err := FetchManifest(ctx, exe)
	if err != nil {
		t.Fatalf("--manifest 失败: %v", err)
	}
	writes, withDry := 0, 0
	for _, f := range m.Fns {
		has := f.Param("dry_run") != nil
		if has {
			withDry++
		}
		if f.Writes {
			writes++
			if !has || f.Param("op") == nil {
				t.Errorf("%s 是写动词却没有 dry_run/op（dry_run=%v）", f.Name, has)
			}
			continue
		}
		// save 不改模型，却真的落盘 —— 它是唯一一个"要写盘但不算写动词"的例外，
		// 两个开关都在它身上手工声明（Manifest.cs 的 Traced）。它必须**双份**都在。
		if f.Name == "save" {
			if !has || f.Param("op") == nil {
				t.Errorf("save 少了 dry_run/op（dry_run=%v）", has)
			}
			continue
		}
		if has {
			t.Errorf("%s 不写东西（writes=false）却挂了 dry_run —— 它的语义是「这次写」，"+
				"挂在读动词上只会让调用方以为它有意义", f.Name)
		}
	}
	if writes == 0 {
		t.Fatalf("manifest 里一个写动词都没有？函数表拉错了")
	}
	if withDry != writes+1 {
		t.Errorf("挂了 dry_run 的动词有 %d 个，写动词 %d 个 + save 一个 —— 两边对不上",
			withDry, writes)
	}
	t.Logf("dry_run 的分布：%d 个写动词 + save 有；函数表共 %d 个动词", writes, len(m.Fns))
}

// TestCorpusDryRunDoesNotTouchTheSource —— 干跑连源包的一个字节都不许改。
//
// 上面那条判据比的是**产出**，这一条比的是**输入**：临时包是写在源包**旁边**的
// （TzpManager 只在工作区内加载，所以它没法写到别处），写错路径就会把源包覆盖掉 ——
// 那正是 save 的两道闸门（Session.CheckOutPath）存在的理由，而干跑是绕过 SaveFn 直接
// 调 Save.Run 的唯一一条路。
func TestCorpusDryRunDoesNotTouchTheSource(t *testing.T) {
	env := requireCorpus(t)
	checked := 0
	for _, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" {
			continue
		}
		cands, why := editCandidates(t, src)
		if why != "" {
			continue
		}
		before, err := sha256full(src)
		if err != nil {
			continue
		}
		label := "srcsafe/" + filepath.Base(src)
		s := startStdio(t, env.exe, ws, perFileBudget)
		o, ok := ask[openReply](t, s, label+" open", "open", map[string]any{"path": src})
		if !ok {
			s.close()
			continue
		}
		_, path, _, flip, ok := resolveEditTarget(t, s, label, o.Handle, cands)
		if ok {
			ask[dryReply](t, s, label+" dry", "set_spec_attr", map[string]any{
				"handle": o.Handle, "path": path, "kind": "field", "attr": "can_edit",
				"value": flip, "dry_run": true})
			after, err := sha256full(src)
			if err != nil {
				t.Errorf("%s: 回读源包失败：%v", label, err)
			} else if before != after {
				t.Errorf("%s: 干跑改了源包（%s -> %s）—— 临时包必须写新文件，绝不覆盖源包", label, before[:12], after[:12])
			} else {
				checked++
			}
		}
		driedUp(t, filepath.Dir(src))
		s.call("close", map[string]any{"handle": o.Handle})
		s.close()
	}
	if checked == 0 {
		t.Skip("没验到（语料里挑不出可写的目标）")
	}
	t.Logf("源包未被触碰：%d 个", checked)
}
