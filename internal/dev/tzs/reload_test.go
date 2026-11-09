package tzs

// reload_test.go —— `reload`：丢弃内存改动、从盘重读，句柄不变。
//
// 它替代的是 `close` + `open`，而那条老路有一个调用方看不见的代价：close 会丢掉 validate 基线
// （Fns/Session.cs 的 Validate.Forget），于是"这一轮改下来新增了什么"要重新问一遍 ——
// 一次 validate 在大表单上是十秒级。reload 的语义是"同一个文件的重读"，所以文件没变时基线
// 依然成立；而"文件变没变"这件事它是**量出来**的（打开时记的摘要 vs 现在的摘要），不是假设。
//
// 这条判据在两个方向上都要成立，缺一个都会变成一句空话：
//   · 文件没变  → baseline=kept、内存改动丢掉、产出回到打开时的字节；
//   · 文件变了  → fileChanged=true、baseline=dropped（它测的是另一份字节了）。
// 第二个方向要真的把文件换掉才能测 —— 用例在**自己拷出来的副本**上做这件事，绝不动语料原件。

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

type reloadReply struct {
	Handle      string `json:"handle"`
	Path        string `json:"path"`
	Program     string `json:"program"`
	Key         string `json:"key"`
	State       string `json:"state"`
	Mutated     bool   `json:"mutated"`
	Baseline    string `json:"baseline"`
	FileChanged *bool  `json:"fileChanged"`
	Discarded   struct {
		UndoCommands int  `json:"undoCommands"`
		LayoutDirty  bool `json:"layoutDirty"`
	} `json:"discarded"`
	Note string `json:"note"`
}

func TestCorpusReloadKeepsTheBaselineAndDropsTheWork(t *testing.T) {
	env := requireCorpus(t)
	checked, skipped := 0, 0
	for i, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" {
			skipped++
			continue
		}
		cands, why := editCandidates(t, src)
		if why != "" {
			skipped++
			continue
		}
		label := "reload/" + filepath.Base(src)
		dir := filepath.Dir(src)
		// 全部动作发生在这份副本上：源包一个字节都不碰。
		copyPath := filepath.Join(dir, "_tdev_reload_"+itoaPID()+"_"+strconv.Itoa(i)+".tzs")
		other := filepath.Join(dir, "_tdev_reload_other_"+itoaPID()+"_"+strconv.Itoa(i)+".tzs")
		pre := filepath.Join(dir, "_tdev_reload_pre_"+itoaPID()+"_"+strconv.Itoa(i)+".tzs")
		post := filepath.Join(dir, "_tdev_reload_post_"+itoaPID()+"_"+strconv.Itoa(i)+".tzs")
		cleanup := func() {
			for _, p := range []string{copyPath, other, pre, post} {
				_ = os.Remove(p)
			}
		}
		defer cleanup()
		if err := copyFile(src, copyPath); err != nil {
			t.Errorf("%s: 拷副本失败：%v", label, err)
			continue
		}

		s := startStdio(t, env.exe, ws, perFileBudget)
		o, ok := ask[openReply](t, s, label+" open", "open", map[string]any{"path": copyPath})
		if !ok {
			cleanup()
			s.close()
			continue
		}
		h := o.Handle

		// 打开时的产出，就是 reload 之后必须回到的那一份。
		if !saveTo(t, s, label+" save(open)", h, pre) {
			cleanup()
			s.close()
			continue
		}
		shaPre, _ := sha16(pre)

		// 还没 validate 过 → 没有基线可谈。三态里的第三种，单独钉一次。
		r0, ok := ask[reloadReply](t, s, label+" reload(none)", "reload", map[string]any{"handle": h})
		if !ok {
			cleanup()
			s.close()
			continue
		}
		if r0.Baseline != "none" {
			t.Errorf("%s: 没跑过 validate，baseline 该是 none，得到 %q", label, r0.Baseline)
		}
		if r0.Handle != h {
			t.Errorf("%s: reload 换了句柄（%q → %q）—— 句柄串必须不变", label, h, r0.Handle)
		}
		if r0.State != "Loaded" || r0.Mutated {
			t.Errorf("%s: reload 之后该是刚加载的样子（state=%q mutated=%v）", label, r0.State, r0.Mutated)
		}

		// 建立基线，再改点东西。
		if _, ok := ask[validateDelta](t, s, label+" validate", "validate", map[string]any{"handle": h}); !ok {
			cleanup()
			s.close()
			continue
		}
		_, path, _, flip, ok := resolveEditTarget(t, s, label, h, cands)
		if !ok {
			cleanup()
			s.close()
			skipped++
			continue
		}
		if _, ok := ask[deltaRef](t, s, label+" write", "set_spec_attr", map[string]any{
			"handle": h, "path": path, "kind": "field", "attr": "can_edit", "value": flip}); !ok {
			cleanup()
			s.close()
			continue
		}
		// 改动存成另一份包：它既是"有改动"的证据，也是下面换文件要用的那份内容。
		if !saveTo(t, s, label+" save(other)", h, other) {
			cleanup()
			s.close()
			continue
		}
		shaOther, _ := sha16(other)
		if shaOther == shaPre {
			t.Errorf("%s: 改动之后产出没变 —— 这一包上挑中的是一次空写，本条用例什么都证明不了", label)
			cleanup()
			s.close()
			continue
		}

		// ---- 文件没变：丢掉改动、保住基线
		r1, ok := ask[reloadReply](t, s, label+" reload(kept)", "reload", map[string]any{"handle": h})
		if !ok {
			cleanup()
			s.close()
			continue
		}
		if r1.Baseline != "kept" {
			t.Errorf("%s: 文件没动，baseline 该是 kept（保住那次 validate 的结果），得到 %q（note：%s）",
				label, r1.Baseline, r1.Note)
		}
		if r1.FileChanged == nil || *r1.FileChanged {
			t.Errorf("%s: 文件没动，fileChanged 该是 false，得到 %v", label, r1.FileChanged)
		}
		if r1.Discarded.UndoCommands == 0 && !r1.Discarded.LayoutDirty {
			t.Errorf("%s: 刚才那次写既没进撤销栈也没动布局？丢弃计数全 0 说明挑选的写是空转", label)
		}
		if !saveTo(t, s, label+" save(after reload)", h, post) {
			cleanup()
			s.close()
			continue
		}
		shaPost, _ := sha16(post)
		if shaPost != shaPre {
			t.Errorf("%s: reload 之后的产出（%s）与打开时的（%s）不同 —— 内存改动没被丢掉", label, shaPost, shaPre)
		}

		// ---- 文件变了：从头到尾都该说出来
		if err := copyFile(other, copyPath); err != nil {
			t.Errorf("%s: 换文件失败：%v", label, err)
			cleanup()
			s.close()
			continue
		}
		r2, ok := ask[reloadReply](t, s, label+" reload(changed)", "reload", map[string]any{"handle": h})
		if !ok {
			cleanup()
			s.close()
			continue
		}
		if r2.FileChanged == nil || !*r2.FileChanged {
			t.Errorf("%s: 盘上的包被换过了，fileChanged 该是 true，得到 %v", label, r2.FileChanged)
		}
		if r2.Baseline != "dropped" {
			t.Errorf("%s: 文件变了之后基线该丢掉（它测的是另一份字节），得到 %q", label, r2.Baseline)
		}
		if r2.Handle != h {
			t.Errorf("%s: 第二次 reload 换了句柄（%q → %q）", label, h, r2.Handle)
		}
		t.Logf("%s: reload 三态都对（none → kept@%s → dropped）", label, shaPre[:12])
		checked++

		s.call("close", map[string]any{"handle": h})
		s.close()
		cleanup()
	}
	if checked == 0 {
		t.Fatalf("一个包都没验到（跳过 %d 个）：判据或前置条件写错了，不是语料的问题", skipped)
	}
	t.Logf("reload：%d 个包上三个分支都验过", checked)
}

func copyFile(from, to string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, b, 0o644)
}
