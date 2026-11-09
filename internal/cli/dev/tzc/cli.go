// 本文件是 tdev 动词核心（原 internal/dev/cli，已并入 tt 的 cobra 树，见 register.go）：
// tt dev tzc 的 export/status/verify/apply/unlock/rename/newfn（+ selftest）的实现体。
//
// 依据设计指南 §2「命令行接口」与 §6「apply 的执行管线（顺序即契约）」。
//
// 退出码（设计指南 §2）：
//
//	0 成功 / 2 包格式错误 / 3 验证失败 / 4 写入被拒 / 5 IO·环境失败
//	1 = 未分类的内部错误（设计指南未定义，仅用于兜底）
package tzc

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/cli/dev/common"
	"tt/internal/config"
	"tt/internal/dev/fence"
	"tt/internal/dev/fgl"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/split"
	"tt/internal/dev/store"
	"tt/internal/dev/synth"
	"tt/internal/dev/tapfile"
	"tt/internal/dev/tglfile"
	"tt/internal/dev/verify"
)

// stringSlice 是可重复的字符串选项（--only 可给多次）。
type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			*s = append(*s, p)
		}
	}
	return nil
}

//---------------------------------------------------------------------------
// 输出辅助
//---------------------------------------------------------------------------

// printFindings 统一渲染验证发现：等级/编号/消息 → 位置（文件:行 + 该行内容）→ 明细。
//
// apply / verify / status 共用同一份渲染，避免三处漂移；位置来自 verify.Finding 的
// File/Line/Snippet（由 gate1/gate2 在能定位的地方填好），是「报错指到第几行」的出口。
func printFindings(w io.Writer, findings []verify.Finding, includeInfo bool) {
	for _, f := range findings {
		if f.Severity == verify.SevInfo && !includeInfo {
			continue
		}
		common.Line(w, "  [%-5s] %-22s %s", f.Severity, f.Code, f.Message)
		if f.File != "" && f.Line > 0 {
			common.Line(w, "            位置：%s:%d", f.File, f.Line)
			if f.Snippet != "" {
				common.Line(w, "            该行：%s", f.Snippet)
			}
		}
		for _, d := range f.Detail {
			common.Line(w, "            %s", d)
		}
	}
}

// regionRef 是「改动清单」里的一条：名字 + 可点击的位置（JSON 直接给 AI 用）。
type regionRef struct {
	Name    string `json:"name"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// changedRefs 返回改动过的 Region（带行号）。行号取**首个差异字节**所在行，
// 最贴近用户实际改的那一处，而不是笼统地指到区段开头。
func changedRefs(base *model.Document, parsed *fence.ParseResult) []regionRef {
	var out []regionRef
	// 用 parse 给出的配对，不按名字查表（真实包里存在区段 id 与点名同名）。
	// 区段比「自身字节」：子点的合法改动不应把父锚点区段也算成改动。
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil {
			continue
		}
		var bc, ec []byte
		if b.Kind == model.RegionSection {
			bc, ec = base.OwnBytes(b), parsed.Doc.OwnBytes(e)
		} else {
			bc = base.Text[b.ContentSpan.Start:b.ContentSpan.End]
			ec = parsed.Doc.Text[e.ContentSpan.Start:e.ContentSpan.End]
		}
		if model.EqualEOL(bc, ec) {
			continue
		}
		ref := regionRef{Name: b.Name}
		off := e.ContentSpan.Start
		if i := model.FirstDiffEOL(bc, ec); i >= 0 {
			if off = e.ContentSpan.Start + i; off > len(parsed.Doc.Text) {
				off = len(parsed.Doc.Text)
			}
		}
		if ln, txt := model.LineAt(parsed.Doc.Text, off); ln > 0 {
			ref.File, ref.Line, ref.Snippet = verify.FileEdited, ln, clipLine(txt)
		}
		out = append(out, ref)
	}
	return out
}

// appendedRefs 返回新增点（只存在于编辑后文档）。
func appendedRefs(parsed *fence.ParseResult) []regionRef {
	var out []regionRef
	for _, r := range parsed.Appended {
		ref := regionRef{Name: r.Name}
		if ln, txt := model.LineAt(parsed.Doc.Text, r.ContentSpan.Start); ln > 0 {
			ref.File, ref.Line, ref.Snippet = verify.FileEdited, ln, clipLine(txt)
		}
		out = append(out, ref)
	}
	return out
}

// deletedRefs 返回被删除的点（只存在于基线，行号只能给基线文件）。
func deletedRefs(base *model.Document, parsed *fence.ParseResult) []regionRef {
	var out []regionRef
	for _, r := range parsed.Deleted {
		ref := regionRef{Name: r.Name}
		if ln, txt := model.LineAt(base.Text, r.ContentSpan.Start); ln > 0 {
			ref.File, ref.Line, ref.Snippet = verify.FileBase, ln, clipLine(txt)
		}
		out = append(out, ref)
	}
	return out
}

// clipLine 把一行内容裁短（清单里只给个提示）。
func clipLine(s string) string {
	s = strings.ReplaceAll(s, "\t", "  ")
	if len(s) > 100 {
		return s[:100] + "…"
	}
	return s
}

// printRegionRefs 打印清单：名字（文件:行），无位置时只打名字。
func printRegionRefs(w io.Writer, label string, refs []regionRef) {
	if len(refs) == 0 {
		common.Line(w, "%s无", label)
		return
	}
	parts := make([]string, 0, len(refs))
	for _, r := range refs {
		if r.File != "" && r.Line > 0 {
			parts = append(parts, fmt.Sprintf("%s（%s:%d）", r.Name, r.File, r.Line))
			continue
		}
		parts = append(parts, r.Name)
	}
	common.Line(w, "%s%s", label, strings.Join(parts, ", "))
}

// loadBase 从工作区重建基线 Document（gate1 的左操作数）。
func loadBase(ws *store.Workspace) (*model.Document, *store.Manifest, error) {
	rf, err := ws.Regions()
	if err != nil {
		return nil, nil, err
	}
	text, err := ws.ReadBase()
	if err != nil {
		return nil, nil, err
	}
	mf, err := ws.Manifest()
	if err != nil {
		return nil, nil, err
	}
	// 解锁状态：三处来源合并（section-state 文件 / v1 的 allow_sec / 包级 section_flag）。
	us, err := ws.EffectiveUnlockState()
	if err != nil {
		return nil, nil, err
	}
	d := &model.Document{
		Text:          text,
		Regions:       rf.Regions,
		Spans:         rf.Spans,
		Env:           rf.Env,
		Prog:          rf.Prog,
		SectionState:  us.State,
		PendingUnlock: us.PendingUnlock,
		UnlockedBy:    us.UnlockedBy,
		Only:          rf.Only,
		Anchors:       mf.Anchors,
		PkgPath:       mf.Pkg.Path,
		Ver:           mf.Pkg.Ver,
	}
	d.Env.SectionState = us.State
	return d, mf, nil
}

//---------------------------------------------------------------------------
// 工作区定位：位置参数省略时默认当前目录
//---------------------------------------------------------------------------

// IsWorkspaceDir 判断目录是否是 tdev 工作区（看两个最稳定的标志文件）。
func IsWorkspaceDir(dir string) bool {
	if dir == "" {
		return false
	}
	if st, err := os.Stat(filepath.Join(dir, "prog.full.4gl")); err != nil || st.IsDir() {
		return false
	}
	if st, err := os.Stat(filepath.Join(dir, "manifest.json")); err != nil || st.IsDir() {
		return false
	}
	return true
}

// resolveWorkspaceDir 解析工作区目录：给了就用给的；没给就用**当前目录**。
//
// 目的是让你 `cd` 进工作区之后直接敲 `tt dev tzc apply`：
//
//	cd D:\pkg\capt110-ws
//	tt dev tzc status / verify / apply / unlock / rename / newfn    ← 都不用再写 <dir>
//
// 当前目录不是工作区时给出明确指引（而不是让你猜为什么报错）。
func resolveWorkspaceDir(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		abs, err := filepath.Abs(explicit)
		if err != nil {
			return explicit, nil
		}
		return abs, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", &store.IOError{Msg: "取不到当前目录", Err: err}
	}
	if !IsWorkspaceDir(cwd) {
		return "", &store.IOError{
			Msg: "当前目录不是 tdev 工作区：" + cwd,
			Err: errors.New("请先 `cd` 进工作区（含 prog.full.4gl + manifest.json），或显式给出 <dir>：" +
				"`tt dev tzc <动词> <dir>`"),
		}
	}
	return cwd, nil
}

// looksLikeWorkspaceArg 判断第一个位置参数是不是"工作区目录"。
// 用于 rename 这类「第一个位置参数可能缺省」的命令，避免把函数名当成目录。
func looksLikeWorkspaceArg(a string) bool {
	if a == "" || strings.HasPrefix(a, "-") {
		return false
	}
	st, err := os.Stat(a)
	if err != nil || !st.IsDir() {
		return false
	}
	return IsWorkspaceDir(a)
}

//---------------------------------------------------------------------------
// export
//---------------------------------------------------------------------------

// defaultWorkspaceDir 是 -o 省略时的默认工作区目录：<包所在目录>/<程序名>-ws。
//
// 为什么是「包旁边的子目录」而不是「就在包旁边平铺」：
// 工作区会写入 prog.full.4gl / manifest.json / snapshot/ / .tdev/ / .git/，
// 若直接平铺在包所在目录，同一目录里放第二个包时就会撞上同一个 prog.full.4gl ——
// 第二次 export 要么被拒（Create 拒绝覆盖已有工作区），要么把上一次未 apply 的编辑挤掉。
func defaultWorkspaceDir(pkgPath string) string {
	return workspaceDirWithSuffix(pkgPath, config.DefaultWorkspaceSuffix)
}

// workspaceDirWithSuffix 是 defaultWorkspaceDir 的可配置版本：后缀来自 config.json
// 的 tdev.workspaceSuffix（缺省 -ws），**只在 -o 省略时**才被问到 —— -o 给了就用 -o。
//
// tdev 里默认值的来源只有两种：写死的缺省，或合并后新增的 tdev 节；后者缺失时
// config.WorkspaceSuffixOrDefault 会自己退回 -ws，所以这里不必再判空。
func workspaceDirWithSuffix(pkgPath, suffix string) string {
	return filepath.Join(filepath.Dir(pkgPath), common.StripIdentitySuffix(pkgPath)+suffix)
}

func cmdExport(args []string) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	out := fs.String("o", "", "输出工作区目录（可省略，默认 <包所在目录>/<程序名>-ws）")
	only := stringSlice{}
	fs.Var(&only, "only", "部分导出：只渲染这些点为可编辑（可重复/逗号分隔）")
	// v2：--allow-sec 已由 unlock 状态机取代（设计指南 §4.2）。保留标志只为给出明确指引。
	allowSec := fs.Bool("allow-sec", false, "已废弃：请改用 `tt dev tzc unlock <dir>`")
	acceptVer := fs.String("accept-ver", "1.0", "可接受的 ver 主次版本")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "o", "only", "accept-ver"); err != nil {
		return 2
	}
	if *allowSec {
		return common.Fail(&store.IOError{
			Msg: "--allow-sec 已废弃（v2 起由框架解锁状态机取代）",
			Err: errors.New("请先 export，再执行 `tt dev tzc unlock <dir>`；" +
				"解锁是一次单向、有代价的状态迁移，需要显式确认（--yes）"),
		}, *asJSON)
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc export <pkg.tzc> [-o <dir>] [--only <点名>...] [--json]")
		fmt.Fprintln(os.Stderr, "      -o 省略时默认导出到 <包所在目录>/<程序名>-ws")
		fmt.Fprintln(os.Stderr, "      要改框架区段：export 后执行 `tt dev tzc unlock <dir>`（不再用 --allow-sec）")
		return 2
	}
	pkgPath := fs.Arg(0)
	outDir := *out
	if outDir == "" {
		// -o 缺省：先用 config.json 的 tdev.workspaceSuffix（缺省仍为 -ws）。
		// 配置读不出来时是零值，WorkspaceSuffixOrDefault 会退回 -ws。
		ts := common.LoadTdevSettings()
		outDir = workspaceDirWithSuffix(pkgPath, ts.WorkspaceSuffixOrDefault())
	}
	w := os.Stdout

	pkg, err := pkgfile.Open(pkgPath, pkgfile.OpenOptions{AcceptVer: *acceptVer})
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	doc, err := synth.Synthesize(pkg, synth.Options{Only: only})
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	fenced, regions, spans, err := fence.Render(doc)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	doc.Text = fenced
	doc.Regions = regions
	doc.Spans = spans

	ws, err := store.Create(outDir, doc, pkg, fenced)
	if err != nil {
		var ioe *store.IOError
		if errors.As(err, &ioe) && strings.Contains(ioe.Msg, "拒绝覆盖已存在的工作区") {
			return common.Fail(&store.IOError{
				Msg: "工作区已存在：" + outDir,
				Err: errors.New("默认目录被占用；换一个位置：-o <新目录>，或先自行删除该工作区"),
			}, *asJSON)
		}
		return common.Fail(err, *asJSON)
	}
	// git 提交（store.Create 已 init；这里补一次以确保 commit 信息带程序名）
	hash, gerr := ws.GitCommit("tdev export: " + doc.Prog)
	if gerr != nil {
		fmt.Fprintf(os.Stderr, "警告：git 提交失败（工作区仍可用）：%v\n", gerr)
	}
	mf, _ := ws.Manifest()

	rep := map[string]any{
		"ok": true, "dir": outDir, "pkg": pkgPath, "prog": doc.Prog,
		"kind": pkg.Kind.String(), "ver": pkg.Ver.Raw,
		"regions": len(doc.AllRegions()), "stats": mf.Stats,
		"anchors": mf.Anchors, "editable": mf.Stats.Editable,
		"section": mf.Section, "only": only, "commit": hash,
		"files": []string{"prog.full.4gl", "manifest.json", ".tdev/base.full.4gl",
			".tdev/base.sha256", ".tdev/regions.json", ".tdev/section-state", "snapshot/"},
	}
	if *asJSON {
		common.EmitJSON(w, rep)
		return 0
	}
	common.Line(w, "已导出工作区：%s", outDir)
	common.Line(w, "  程序：%s（%s，ver %s）", doc.Prog, pkg.Kind.String(), pkg.Ver.Raw)
	common.Line(w, "  Region：%d 个（可编辑 %d，只读 %d）", mf.Stats.Points+mf.Stats.Sections, mf.Stats.Editable, mf.Stats.Readonly)
	common.Line(w, "  锚点：other.function=%v other.dialog=%v other.report=%v",
		mf.Anchors["other.function"], mf.Anchors["other.dialog"], mf.Anchors["other.report"])
	if mf.Section.State == string(model.SectionUnlocked) {
		common.Line(w, "  框架：已解开（section_flag=%s，来源 %s）→ 可编辑区段标 [EDITABLE-SEC]",
			mf.Section.SectionFlag, mf.Section.UnlockedBy)
	} else {
		common.Line(w, "  框架：未解开（Locked）→ SectionRegion 全部只读")
		common.Line(w, "       要改框架区段：`tt dev tzc unlock %s`（单向、有代价：解开后规格调整不再自动生成程序代码）", outDir)
	}
	if len(only) > 0 {
		common.Line(w, "  --only：只允许改 %s（其它点一律 READONLY not-exported）", strings.Join(only, ", "))
	}
	common.Line(w, "")
	common.Line(w, "唯一编辑文件：%s", filepath.Join(outDir, "prog.full.4gl"))
	common.Line(w, "下一步：编辑它，然后 `tt dev tzc verify %s` 或 `tt dev tzc apply %s`", outDir, outDir)
	return 0
}

//---------------------------------------------------------------------------
// unlock（框架解锁状态迁移，只改 workspace）
//---------------------------------------------------------------------------

// unlockRegions 以 Unlocked 状态重判所有区段的可编辑性，并生成新的 begin 围栏行。
func unlockRegions(doc *model.Document) (map[string]string, []*model.Region) {
	env := doc.Env
	env.SectionState = model.SectionUnlocked
	repl := map[string]string{}
	var updated []*model.Region
	for _, r := range doc.AllRegions() {
		if r.Kind != model.RegionSection {
			continue
		}
		c := r.DeepCopy()
		isReadonly := false
		if v, ok := c.Meta["tgl_readonly"]; ok && strings.EqualFold(v, "Y") {
			isReadonly = true
		}
		editable, deny := synth.ResolveSection(c.Name, isReadonly, c.Meta, env, doc.Prog)
		if len(doc.Only) > 0 {
			editable, deny = false, model.DenyOnlyPoints
		}
		c.Editable = editable
		c.DenyCode = deny
		c.Reason = model.DenyReasonText(deny)
		updated = append(updated, c)
	}
	// 逐个生成新的 begin 围栏行（只替换真的变了的）
	for _, c := range updated {
		old := fence.BeginFenceLine(regionByName(doc, c.Name))
		nw := fence.BeginFenceLine(c)
		if old != nw {
			repl[c.Name] = nw
		}
	}
	return repl, updated
}

// applyEditable 把重判后的可编辑性写回 Region 表（ApplyEdits 只搬坐标）。
func applyEditable(regions []*model.Region, updated []*model.Region) {
	for _, r := range regions {
		r.Walk(func(x *model.Region) {
			for _, u := range updated {
				if x.Name == u.Name && x.Kind == u.Kind {
					x.Editable, x.DenyCode, x.Reason = u.Editable, u.DenyCode, u.Reason
				}
			}
		})
	}
}

func regionByName(doc *model.Document, name string) *model.Region {
	for _, r := range doc.AllRegions() {
		if r.Name == name {
			return r
		}
	}
	return nil
}

func cmdUnlock(args []string) int {
	fs := flag.NewFlagSet("unlock", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "确认「解开框架」的代价警告（单向、不可逆）")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "o", "only", "accept-ver"); err != nil {
		return 2
	}
	dir, derr := resolveWorkspaceDir(fs.Arg(0)) // 省略 <dir> → 用当前目录
	if derr != nil {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc unlock [<dir>] [--yes] [--json]   # <dir> 省略时用当前目录")
		return common.Fail(derr, *asJSON)
	}
	w := os.Stdout

	ws, err := store.Open(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	release, err := ws.Lock()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	defer release()

	base, mf, err := loadBase(ws)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// ① 授权闸门（逐条重放设计器三分支；R9：无绕过）
	dec := synth.CheckUnlockPermission(base.Env, *yes)
	if !dec.Allow {
		if dec.NeedYes {
			// 需要二次确认：打印设计器原文，退出码 4
			if *asJSON {
				common.EmitJSON(w, map[string]any{"ok": false, "stage": "unlock-gate",
					"reason": dec.Reason, "detail": dec.Detail, "need_yes": true})
				return 4
			}
			common.Line(w, "拒绝解锁（退出码 4）：%s", dec.Reason)
			for _, d := range dec.Detail {
				common.Line(w, "  %s", d)
			}
			common.Line(w, "  确认后加 --yes 重新执行（AI 不得自主加 --yes）")
			return 4
		}
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": false, "stage": "unlock-gate",
				"reason": dec.Reason, "detail": dec.Detail})
			return 4
		}
		common.Line(w, "拒绝解锁（退出码 4）：%s", dec.Reason)
		for _, d := range dec.Detail {
			common.Line(w, "  %s", d)
		}
		return 4
	}
	// 已经是解开态 → no-op（设计器对这类包不设拦截）
	if base.SectionState == model.SectionUnlocked && !base.PendingUnlock {
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": true, "dir": dir, "changed": false,
				"state": string(model.SectionUnlocked), "reason": dec.Reason})
			return 0
		}
		common.Line(w, "框架本来就是解开态（%s），无需 unlock", dec.Reason)
		return 0
	}
	if base.SectionState == model.SectionUnlocked && base.PendingUnlock {
		common.Line(w, "框架已处于待落盘的解锁态（pending_unlock）；apply 时会把 section_flag 置 Y")
	}

	// ② 读编辑后的文档，按 Unlocked 重判区段
	edited, err := ws.ReadEdited()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	parsed, err := fence.Parse(base, edited)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	repl, updated := unlockRegions(parsed.Doc)

	// ③ 只改围栏行（工具自身唯一被允许改写围栏行的操作）。
	//
	// 关键：**编辑文件与基线两处都要翻旗标**，但基线绝不能被 AI 的正文改动「吸收」。
	// 基线代表「包里的样子」——它是 gate1 的左操作数；把待落盘的正文编辑写进基线
	// 会让 gate1 直接失明（那正是本命令第一版实现踩到的坑）。
	applyFenceFlags := func(doc *model.Document) ([]byte, []*model.Region, []model.Span, error) {
		var es []fence.Edit
		for _, r := range doc.AllRegions() {
			nw, ok := repl[r.Name]
			if !ok || r.Kind != model.RegionSection {
				continue
			}
			es = append(es, fence.Edit{Start: r.BeginFence.Start, End: r.BeginFence.End, Repl: []byte(nw)})
		}
		txt, regs, spans, err := fence.ApplyEdits(doc, es)
		if err != nil {
			return nil, nil, nil, err
		}
		applyEditable(regs, updated)
		return txt, regs, spans, nil
	}
	newText, newRegions, _, err := applyFenceFlags(parsed.Doc)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	_ = newRegions
	newBase, newBaseRegions, newBaseSpans, err := applyFenceFlags(base)
	if err != nil {
		return common.Fail(err, *asJSON)
	}

	// ④ 落盘：prog.full.4gl（保留 AI 的编辑）+ 基线（只翻旗标）+ regions + section-state
	if err := ws.WriteEdited(newText); err != nil {
		return common.Fail(err, *asJSON)
	}
	newDoc := *base
	newDoc.Text = newBase
	newDoc.Regions = newBaseRegions
	newDoc.Spans = newBaseSpans
	newDoc.SectionState = model.SectionUnlocked
	newDoc.PendingUnlock = true
	newDoc.UnlockedBy = model.UnlockedByUnlockCmd
	newDoc.Env.SectionState = model.SectionUnlocked
	if err := ws.UpdateAfterApply(&newDoc, newBase); err != nil {
		return common.Fail(err, *asJSON)
	}
	if err := ws.WriteSectionState(&store.SectionStateFile{
		State: string(model.SectionUnlocked), PendingUnlock: true, UnlockedBy: model.UnlockedByUnlockCmd,
	}); err != nil {
		return common.Fail(err, *asJSON)
	}
	hash, gerr := ws.GitCommit("tdev unlock: " + base.Prog)
	if gerr != nil {
		fmt.Fprintf(os.Stderr, "警告：git 提交失败：%v\n", gerr)
	}
	editableSections := 0
	for _, r := range newRegions {
		r.Walk(func(x *model.Region) {
			if x.Kind == model.RegionSection && x.Editable {
				editableSections++
			}
		})
	}
	_ = mf
	rep := map[string]any{
		"ok": true, "dir": dir, "changed": true, "state": string(model.SectionUnlocked),
		"pending_unlock": true, "unlocked_by": model.UnlockedByUnlockCmd,
		"reason": dec.Reason, "editable_sections": editableSections, "commit": hash,
	}
	if *asJSON {
		common.EmitJSON(w, rep)
		return 0
	}
	common.Line(w, "已解开框架（workspace 状态迁移，未碰 .tzc）")
	common.Line(w, "  工作区：%s", dir)
	common.Line(w, "  依据：%s", dec.Reason)
	common.Line(w, "  可编辑区段：%d 个（其余仍 READONLY：锚点区段 / readonly=\"Y\" / topstd 规则）", editableSections)
	common.Line(w, "  围栏已重渲染：可编辑区段标 [EDITABLE-SEC]")
	common.Line(w, "  pending_unlock=true（apply 时才会把 TAP 根 section_flag 置 Y）")
	common.Line(w, "")
	common.Line(w, "代价提醒：解开框架后，规格的任何调整都不会再产生对应的程序代码（不可逆）。")
	common.Line(w, "下一步：编辑区段正文 → `tt dev tzc apply %s`（已在该目录内可省 <dir>）", dir)
	return 0
}

func cmdStatus(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "o", "only", "accept-ver"); err != nil {
		return 2
	}
	dir, derr := resolveWorkspaceDir(fs.Arg(0)) // 省略 <dir> → 用当前目录
	if derr != nil {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc status [<dir>] [--json]   # <dir> 省略时用当前目录")
		return common.Fail(derr, *asJSON)
	}
	w := os.Stdout

	ws, err := store.Open(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	base, mf, err := loadBase(ws)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	edited, err := ws.ReadEdited()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	parsed, err := fence.Parse(base, edited)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	g1 := verify.Gate1(base, parsed)

	// 预估写回面
	var predicted []pkgfile.EntryAction
	if mf.Pkg.Path != "" {
		if pkg, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{}); err == nil {
			tapDoc, _ := tapfile.Parse(pkg.Tap().Data)
			if tapDoc != nil {
				plan, perr := split.Split(base, parsed, pkg)
				if perr == nil {
					newTap, terr := tapfile.Rewrite(pkg.Tap().Data, plan.Ops...)
					if terr == nil {
						newTgl := applyTglPatches(pkg, plan)
						if acts, err := pkg.Plan(pkgfile.Rebuild{Tap: newTap, Tgl: newTgl}); err == nil {
							predicted = acts
						}
					}
				}
			}
		}
	}

	changed, added, deleted := changedRefs(base, parsed), appendedRefs(parsed), deletedRefs(base, parsed)
	rep := map[string]any{
		"ok":       true,
		"dir":      dir,
		"prog":     base.Prog,
		"findings": g1.Findings,
		"summary":  g1.Summary(),
		"entries":  predicted,
		// 改动清单带行号：name + file:line + 该行内容，AI 可直接跳到该行
		"changed_regions": changed,
		"added_regions":   added,
		"deleted_regions": deleted,
	}
	if *asJSON {
		common.EmitJSON(w, rep)
	} else {
		common.Line(w, "工作区：%s（程序 %s）", dir, base.Prog)
		common.Line(w, "基线：%s  sha256=%s", filepath.Join(dir, ".tdev", "base.full.4gl"), common.Short(mf.Pkg.Sha256))
		common.Line(w, "")
		common.Line(w, "改动：")
		printRegionRefs(w, "  改动的点  ", changed)
		printRegionRefs(w, "  新增的点  ", added)
		printRegionRefs(w, "  删除的点  ", deleted)
		common.Line(w, "")
		common.Line(w, "验证（gate1）：%s", g1.Summary())
		printFindings(w, g1.Findings, false)
		if len(predicted) > 0 {
			common.Line(w, "")
			common.Line(w, "预估写回面：")
			for _, a := range predicted {
				mark := "不变"
				if a.Changed {
					mark = "改写"
				}
				common.Line(w, "  %-28s %s  %d → %d 字节", a.Name, mark, a.OldSize, a.NewSize)
			}
		}
	}
	if g1.HasDenied() {
		return 4
	}
	if g1.HasError() {
		return 3
	}
	return 0
}

//---------------------------------------------------------------------------
// rename / newfn（编辑辅助：把设计器弹窗 CLI 化，只改 workspace）
//---------------------------------------------------------------------------

// loadEditable 打开工作区、取锁、读基线 + 编辑后的文档并解析。
func loadEditable(dir string) (*store.Workspace, func(), *model.Document, *store.Manifest, []byte, *fence.ParseResult, error) {
	ws, err := store.Open(dir)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	release, err := ws.Lock()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	base, mf, err := loadBase(ws)
	if err != nil {
		release()
		return nil, nil, nil, nil, nil, nil, err
	}
	edited, err := ws.ReadEdited()
	if err != nil {
		release()
		return nil, nil, nil, nil, nil, nil, err
	}
	parsed, err := fence.Parse(base, edited)
	if err != nil {
		release()
		return nil, nil, nil, nil, nil, nil, err
	}
	return ws, release, base, mf, edited, parsed, nil
}

// editFenceFn 原子同步「围栏 fn/scope/desc 字段 + 签名行 + 描述块」三处。
func editFenceFn(doc *model.Document, reg *model.Region, newFn, newScope, newDesc string, descSet bool) ([]fence.Edit, error) {
	var edits []fence.Edit
	// ① 围栏行：重算 begin 行（fn/scope/desc 都是它的字段）
	c := reg.DeepCopy()
	if newFn != "" {
		c.Fn = newFn
	}
	if newScope != "" {
		c.Scope = newScope
	}
	if descSet {
		c.Desc = newDesc
	}
	newLine := fence.BeginFenceLine(c)
	edits = append(edits, fence.Edit{Start: reg.BeginFence.Start, End: reg.BeginFence.End, Repl: []byte(newLine)})

	// ② 签名行：按新 scope/fn 重写（DIALOG/REPORT 的 PUBLIC 省略限定符）
	if reg.SignatureSpan != nil && (newFn != "" || newScope != "") {
		old := string(doc.Text[reg.SignatureSpan.Start:reg.SignatureSpan.End])
		kind, _ := fgl.EnvelopeKindFor(reg.Name)
		sig := split.BuildSignature(old, kind, pickScope(newScope, reg.Scope), pickFn(newFn, reg.Fn))
		edits = append(edits, fence.Edit{Start: reg.SignatureSpan.Start, End: reg.SignatureSpan.End, Repl: []byte(sig)})
	}
	// ③ 描述块：desc 字段是权威声明 → 由它重写块（每行确保以 "# " 开头）
	if descSet && reg.DescSpan != nil {
		block := renderDescBlock(newDesc)
		edits = append(edits, fence.Edit{Start: reg.DescSpan.Start, End: reg.DescSpan.End, Repl: []byte(block)})
	}
	return edits, nil
}

func pickFn(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func pickScope(a, b string) string {
	if a != "" {
		return a
	}
	if b != "" {
		return b
	}
	return "PRIVATE"
}

// renderDescBlock 把 desc 字段展开成描述块文本（每行以 "# " 开头，末尾带换行）。
func renderDescBlock(desc string) string {
	d := strings.TrimRight(strings.ReplaceAll(desc, "\r\n", "\n"), "\n")
	if strings.TrimSpace(d) == "" {
		return ""
	}
	var b strings.Builder
	for _, ln := range strings.Split(d, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			b.WriteString("#\n")
			continue
		}
		if strings.HasPrefix(t, "#") {
			b.WriteString(t + "\n")
			continue
		}
		b.WriteString("# " + t + "\n")
	}
	return b.String()
}

func cmdRename(args []string) int {
	fs := flag.NewFlagSet("rename", flag.ContinueOnError)
	scope := fs.String("scope", "", "新的 scope：PUBLIC | PRIVATE（省略则不改）")
	desc := fs.String("desc", "", "新的描述块（省略则不改）")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "scope", "desc"); err != nil {
		return 2
	}
	// 位置参数两种形态：`rename <dir> <旧> <新>` 或（cd 进工作区后）`rename <旧> <新>`
	var dir, oldName, newName string
	switch {
	case fs.NArg() >= 3:
		dir, oldName, newName = fs.Arg(0), fs.Arg(1), fs.Arg(2)
	case fs.NArg() == 2:
		d, derr := resolveWorkspaceDir("")
		if derr != nil {
			fmt.Fprintln(os.Stderr, "用法：tt dev tzc rename [<dir>] <旧函数名> <新函数名> [--scope …] [--desc …]")
			return common.Fail(derr, *asJSON)
		}
		dir, oldName, newName = d, fs.Arg(0), fs.Arg(1)
	default:
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc rename [<dir>] <旧函数名> <新函数名> [--scope PUBLIC|PRIVATE] [--desc <描述>] [--json]")
		fmt.Fprintln(os.Stderr, "      只改 workspace（围栏 fn + 签名行 + 描述块三处原子同步）；落盘仍走 apply")
		return 2
	}
	if fs.NArg() == 3 && !looksLikeWorkspaceArg(fs.Arg(0)) {
		fmt.Fprintf(os.Stderr, "错误（退出码 5）：第一个参数 %q 不是 tdev 工作区目录；"+
			"若要省略 <dir>，请先 cd 进工作区\n", fs.Arg(0))
		return 5
	}
	w := os.Stdout

	ws, release, base, mf, _, parsed, err := loadEditable(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	defer release()

	// 定位目标：允许给「点全名」或「裸函数名」
	var reg *model.Region
	for _, r := range parsed.Doc.AllRegions() {
		if r.Kind != model.RegionPoint || r.Plain {
			continue
		}
		if r.Name == oldName || strings.HasSuffix(r.Name, "."+oldName) || bareOf(r.Fn) == oldName {
			reg = r
			break
		}
	}
	if reg == nil {
		return common.Fail(&synth.DeniedError{
			Msg:    "找不到要改名的自订定义点",
			Detail: []string{oldName, "可用 `tt dev tzc status` 查看 Region 清单（改名只支持 function./dialog./report. 点）"},
		}, *asJSON)
	}
	if !reg.Editable {
		return common.Fail(&synth.DeniedError{
			Msg:    "目标点不可编辑，改名被拒（写入被拒）",
			Detail: []string{reg.Name, "deny=" + reg.DenyCode, reg.Reason},
		}, *asJSON)
	}
	// 前缀与参数列表保持不变（只换函数名主体）
	oldFn := reg.Fn
	suffix := ""
	if i := strings.IndexByte(oldFn, '('); i >= 0 {
		suffix = oldFn[i:]
	}
	targetFn := newName
	if !strings.Contains(targetFn, "(") {
		targetFn = newName + suffix
	}

	edits, err := editFenceFn(parsed.Doc, reg, targetFn, *scope, *desc, *desc != "")
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	newText, _, _, err := fence.ApplyEdits(parsed.Doc, edits)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// 预检：把新的文本当成「已编辑文档」跑一遍 gate1 + 结构事务校验
	if err := preflight(base, parsed, newText, tapDocOf(mf)); err != nil {
		return common.Fail(err, *asJSON)
	}
	if err := ws.WriteEdited(newText); err != nil {
		return common.Fail(err, *asJSON)
	}
	rep := map[string]any{
		"ok": true, "dir": dir, "region": reg.Name,
		"from": oldFn, "to": targetFn, "scope": pickScope(*scope, reg.Scope),
	}
	if *asJSON {
		common.EmitJSON(w, rep)
		return 0
	}
	common.Line(w, "已改名（只改 workspace，未碰 .tzc）")
	common.Line(w, "  Region：%s", reg.Name)
	common.Line(w, "  签名：%s → %s", oldFn, targetFn)
	if *scope != "" {
		common.Line(w, "  scope：%s → %s", reg.Scope, strings.ToUpper(*scope))
	}
	common.Line(w, "")
	common.Line(w, "注意：围栏的锚定点名（%s）不变 —— 点身份的切换由 apply 的改名事务完成", reg.Name)
	common.Line(w, "下一步：改完调用点后 `tt dev tzc verify %s`，再 `tt dev tzc apply %s`（已在该目录内可省 <dir>）", dir, dir)
	return 0
}

// preflight 在不落盘的前提下，对「即将写入的文本」跑 gate1 + 结构事务校验。
//
// 必须**重新解析**新文本（而不是复用 ApplyEdits 的位移结果）：只有重新解析才会
// 发现 `newfn` 追加的新 Region —— 否则新点块的字节会被算进锚点区段的「自身字节」，
// gate1 会误判成"锚点区段被改"。
func preflight(base *model.Document, parsed *fence.ParseResult, newText []byte, tapDoc *tapfile.Doc) error {
	nb := *base
	res, err := fence.Parse(&nb, newText)
	if err != nil {
		return err
	}
	if g1 := verify.Gate1(base, res); g1.HasError() {
		return &verify.VerifyErrorFromFindings{Findings: g1.Findings}
	}
	if st := verify.ValidateStructural(base, res, tapDoc); st.HasError() {
		return &verify.VerifyErrorFromFindings{Findings: st.Findings}
	}
	return nil
}

// tapDocOf 读工作区当前源包的 TAP（预检用）。
func tapDocOf(mf *store.Manifest) *tapfile.Doc {
	if mf == nil || mf.Pkg.Path == "" {
		return nil
	}
	pkg, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{})
	if err != nil {
		return nil
	}
	td, _ := tapfile.Parse(pkg.Tap().Data)
	return td
}

func bareOf(fn string) string {
	if i := strings.IndexByte(fn, '('); i >= 0 {
		return fn[:i]
	}
	return fn
}

func cmdNewfn(args []string) int {
	fs := flag.NewFlagSet("newfn", flag.ContinueOnError)
	typ := fs.String("type", "", "FUNCTION | DIALOG | REPORT（必填）")
	name := fs.String("name", "", "函数名（省略则默认 <prog>_newfunc，自动去重）")
	scope := fs.String("scope", "", "PUBLIC | PRIVATE（省略按程序类型默认）")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "type", "name", "scope"); err != nil {
		return 2
	}
	if *typ == "" {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc newfn [<dir>] --type FUNCTION|DIALOG|REPORT [--name <名>] [--scope …] [--json]")
		fmt.Fprintln(os.Stderr, "      <dir> 省略时用当前目录")
		return 2
	}
	dir, derr := resolveWorkspaceDir(fs.Arg(0)) // 省略 <dir> → 用当前目录
	if derr != nil {
		return common.Fail(derr, *asJSON)
	}
	w := os.Stdout
	kind := strings.ToUpper(*typ)
	if kind != "FUNCTION" && kind != "DIALOG" && kind != "REPORT" {
		return common.Fail(&synth.DeniedError{Msg: "--type 必须是 FUNCTION | DIALOG | REPORT", Detail: []string{*typ}}, *asJSON)
	}

	ws, release, base, mf, _, parsed, err := loadEditable(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	defer release()

	// 锚点区段（other.<type>）必须存在且可追加
	anchor := map[string]string{"FUNCTION": "other.function", "DIALOG": "other.dialog", "REPORT": "other.report"}[kind]
	secName := base.Prog + "." + strings.Replace(anchor, "other.", "other_", 1)
	var sec *model.Region
	for _, r := range parsed.Doc.AllRegions() {
		if r.Kind == model.RegionSection && r.Name == secName {
			sec = r
			break
		}
	}
	if sec == nil || !sec.Append {
		return common.Fail(&verify.VerifyErrorFromMsg{
			Msg: "该程序的 TGL 没有 " + anchor + " 锚点，无法注入新函数（I2b）",
			Detail: []string{"锚点区段：" + secName,
				"请在设计器里确认该程序模板是否包含该锚点"},
		}, *asJSON)
	}
	// 函数名：默认 <prog>_newfunc，并自动去重
	fn := *name
	if fn == "" {
		fn = base.Prog + "_newfunc"
	}
	used := map[string]bool{}
	for _, r := range parsed.Doc.AllRegions() {
		used[bareOf(r.Fn)] = true
		used[bareOf(r.Name)] = true
	}
	if used[fn] {
		if *name != "" {
			return common.Fail(&synth.DeniedError{Msg: "函数名已存在", Detail: []string{fn}}, *asJSON)
		}
		for i := 2; ; i++ {
			cand := fmt.Sprintf("%s%d", fn, i)
			if !used[cand] {
				fn = cand
				break
			}
		}
	}
	sc := *scope
	if sc == "" {
		sc = defaultScope(base.Env.ProgType)
	}
	block := buildNewBlock(kind, sc, fn, base.Prog)
	// 插到锚点区段正文末尾（`{</section>}` 行之前）
	at := sec.ContentSpan.End
	blockText := fence.BeginFenceLine(appendedRegionStub(kind, fn, sc)) + "\n" + block + "\n" + fence.FencePrefix + "end point}\n"
	edits := []fence.Edit{{Start: at, End: at, Repl: []byte(blockText)}}
	newText, _, _, err := fence.ApplyEdits(parsed.Doc, edits)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	if err := preflight(base, parsed, newText, tapDocOf(mf)); err != nil {
		return common.Fail(err, *asJSON)
	}
	if err := ws.WriteEdited(newText); err != nil {
		return common.Fail(err, *asJSON)
	}
	rep := map[string]any{"ok": true, "dir": dir, "type": kind, "name": fn, "scope": sc, "anchor": secName}
	if *asJSON {
		common.EmitJSON(w, rep)
		return 0
	}
	common.Line(w, "已新增自订点（只改 workspace，未碰 .tzc）")
	common.Line(w, "  类型/名字：%s %s", kind, fn)
	common.Line(w, "  scope：%s", sc)
	common.Line(w, "  注入位置：%s（锚点末尾）", secName)
	common.Line(w, "下一步：编辑它的正文 → `tt dev tzc verify %s` → `tt dev tzc apply %s`（已在该目录内可省 <dir>）", dir, dir)
	return 0
}

// defaultScope 按程序类型给默认 scope（FunctionInfoWindow.xaml.cs:143-192）。
func defaultScope(progType string) string {
	switch strings.ToUpper(progType) {
	case "B":
		return "PUBLIC"
	case "M", "G", "X", "Z", "Q", "S":
		return "PRIVATE"
	case "W":
		return "PUBLIC"
	}
	return "PRIVATE"
}

// appendedRegionStub 造一个用于渲染围栏行的临时 Region（只取旗标与结构字段）。
func appendedRegionStub(kind, fn, scope string) *model.Region {
	k := model.RegionPoint
	prefix := "function."
	switch kind {
	case "DIALOG":
		prefix = "dialog."
	case "REPORT":
		prefix = "report."
	}
	return &model.Region{
		Name: prefix + fn, Kind: k, Editable: true,
		Fn: fn + "()", Scope: scope, Meta: map[string]string{"new": "Y", "status": "u", "src": ""},
	}
}

// buildNewBlock 造新点块（描述块 + 签名行 + 空正文 + END 行）。
//
// FUNCTION 用设计器形态的函数头模板（见 newfn_template.go：顶部空行 + 80 个 # 的框）；
// DIALOG/REPORT 维持单行 `#+ Description:` 描述块与设计器注入的默认正文。
func buildNewBlock(kind, scope, fn, prog string) string {
	var sig string
	switch kind {
	case "DIALOG":
		if strings.EqualFold(scope, "PUBLIC") {
			sig = "DIALOG " + fn + "()"
		} else {
			sig = strings.ToUpper(scope) + " DIALOG " + fn + "()"
		}
	case "REPORT":
		if strings.EqualFold(scope, "PUBLIC") {
			sig = "REPORT " + fn + "()"
		} else {
			sig = strings.ToUpper(scope) + " REPORT " + fn + "()"
		}
		if true { // REPORT 空正文：设计器注入默认 FORMAT 模板（AddPointModel.cs:1048-1051）
			body := "    FORMAT\r\n           \r\n        ON EVERY ROW\r\n            PRINTX g_grNumFmt.*\r\n            PRINTX"
			return dialogDescBlock(prog) + sig + "\n" + body + "\nEND REPORT"
		}
	default:
		sig = strings.ToUpper(scope) + " FUNCTION " + fn + "()"
		// FUNCTION：用设计器形态的函数头模板（顶部空行 + 80 个 # 的框 + 占位行）
		return newFnHeaderTemplate(fn) + sig + "\n" + "END FUNCTION"
	}
	return dialogDescBlock(prog) + sig + "\n" + "END " + kind
}

// dialogDescBlock 是 DIALOG/REPORT 沿用的单行描述块（FUNCTION 已改用
// newFnHeaderTemplate；这两种类型本次不动，避免把函数头模板套到对话框上）。
func dialogDescBlock(prog string) string {
	return renderDescBlock("#+ Description: " + prog + " 自訂函式")
}

func cmdVerify(args []string) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "输出 JSON")
	strict := fs.Bool("strict", false, "把 warn 级发现也算作失败（供 CI 使用）")
	if err := common.ParseArgs(fs, args, "o", "only", "accept-ver"); err != nil {
		return 2
	}
	dir, derr := resolveWorkspaceDir(fs.Arg(0)) // 省略 <dir> → 用当前目录
	if derr != nil {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc verify [<dir>] [--json] [--strict]   # <dir> 省略时用当前目录")
		return common.Fail(derr, *asJSON)
	}
	w := os.Stdout

	ws, err := store.Open(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	base, mf, err := loadBase(ws)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	edited, err := ws.ReadEdited()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	parsed, err := fence.Parse(base, edited)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	g1 := verify.Gate1(base, parsed)

	rep := &verify.Report{}
	rep.Findings = append(rep.Findings, g1.Findings...)
	rep.Errors += g1.Errors
	rep.Warns += g1.Warns
	rep.Infos += g1.Infos
	rep.Denied += g1.Denied

	// gate2 需要「编辑后将要写回的 TAP」。这里在内存里算出来，不落盘。
	tapDoc := (*tapfile.Doc)(nil)
	if mf.Pkg.Path != "" {
		if pkg, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{}); err == nil {
			td, _ := tapfile.Parse(pkg.Tap().Data)
			if td != nil {
				if plan, perr := split.Split(base, parsed, pkg); perr == nil {
					if newTap, terr := tapfile.Rewrite(pkg.Tap().Data, plan.Ops...); terr == nil {
						tapDoc, _ = tapfile.Parse(newTap)
					}
				}
			}
		}
	}
	if tapDoc != nil {
		g2 := verify.Gate2(base, parsed, tapDoc)
		rep.Findings = append(rep.Findings, g2.Findings...)
		rep.Errors += g2.Errors
		rep.Warns += g2.Warns
		rep.Infos += g2.Infos
		rep.Denied += g2.Denied
	}

	ok := !rep.HasError()
	if *strict && rep.Warns > 0 {
		ok = false
	}
	if *asJSON {
		common.EmitJSON(w, map[string]any{
			"ok": ok, "dir": dir, "prog": base.Prog,
			"summary": rep.Summary(), "findings": rep.Findings,
			"errors": rep.Errors, "warns": rep.Warns, "infos": rep.Infos,
		})
	} else {
		common.Line(w, "验证工作区：%s（程序 %s）", dir, base.Prog)
		printFindings(w, rep.Findings, true) // verify 是全量视图：info 也打印
		common.Line(w, "")
		common.Line(w, "结论：%s%s", rep.Summary(), map[bool]string{true: "（通过）", false: "（失败）"}[ok])
	}
	if !ok {
		if rep.HasDenied() {
			return 4
		}
		return 3
	}
	return 0
}

//---------------------------------------------------------------------------
// apply
//---------------------------------------------------------------------------

func cmdApply(args []string) int {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	outPkg := fs.String("o", "", "写到另一个包路径（默认覆盖原包）")
	dryRun := fs.Bool("dry-run", false, "跑完管线并输出逐条目 diff，不落盘、不 commit")
	yes := fs.Bool("yes", false, "对「写区段（解开框架）」做二次确认")
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args, "o", "only", "accept-ver"); err != nil {
		return 2
	}
	dir, derr := resolveWorkspaceDir(fs.Arg(0)) // 省略 <dir> → 用当前目录
	if derr != nil {
		fmt.Fprintln(os.Stderr, "用法：tt dev tzc apply [<dir>] [-o <pkg.tzc>] [--dry-run] [--yes] [--json]   # <dir> 省略时用当前目录")
		return common.Fail(derr, *asJSON)
	}
	w := os.Stdout

	ws, err := store.Open(dir)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	release, err := ws.Lock()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	defer release()

	base, mf, err := loadBase(ws)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// 1) 读取 edited
	edited, err := ws.ReadEdited()
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// 2) parse_fenced：结构性第一道（配对 / 归属 / 嵌套深度）
	parsed, err := fence.Parse(base, edited)
	if err != nil {
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": false, "stage": "parse_fenced", "message": err.Error()})
		} else {
			common.Line(w, "apply 失败（parse_fenced）：%v", err)
		}
		return common.Fail(err, *asJSON)
	}
	// 3) gate1：字节恒等（围栏行/围栏外/只读区/结构行）+ 写入授权
	g1 := verify.Gate1(base, parsed)
	if g1.HasDenied() || g1.HasError() {
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": false, "stage": "gate1", "findings": g1.Findings,
				"summary": g1.Summary()})
		} else {
			common.Line(w, "apply 被拒绝（gate1 %s）：", g1.Summary())
			printFindings(w, g1.Findings, false)
		}
		if g1.HasDenied() {
			return 4
		}
		return 3
	}

	// 源包必须未变（D-6）
	pkgPath := mf.Pkg.Path
	if pkgPath == "" {
		return common.Fail(fmt.Errorf("manifest 里没有记录源包路径"), *asJSON)
	}
	curSha, err := model.Sha256File(pkgPath)
	if err != nil {
		return common.Fail(&store.IOError{Msg: "读不到源包 " + pkgPath, Err: err}, *asJSON)
	}
	if mf.Pkg.Sha256 != "" && curSha != mf.Pkg.Sha256 {
		return common.Fail(&store.IOError{Msg: "源包自 export 之后已被改动，拒绝写回",
			Err: fmt.Errorf("manifest 记录 %s，当前 %s；请重新 export", common.Short(mf.Pkg.Sha256), common.Short(curSha))}, *asJSON)
	}
	pkg, err := pkgfile.Open(pkgPath, pkgfile.OpenOptions{})
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// I15：TAP 条目基名 ≠ 根 prog → 引用标准程序的包，设计器 Packing 会走 CiteTAP，拒绝整体重写
	if tap := pkg.Tap(); tap != nil {
		baseName := strings.TrimSuffix(filepath.Base(tap.Name), filepath.Ext(tap.Name))
		tapDoc0, _ := tapfile.Parse(tap.Data)
		if tapDoc0 != nil {
			prog := tapDoc0.RootAttrOr("prog", "")
			if prog != "" && baseName != prog {
				me := &pkgfile.FormatError{
					Msg: "TAP 条目基名与程序名不一致（引用标准程序的包），拒绝整体重写 .tap（I15）",
					Detail: []string{"条目基名：" + baseName, "根 prog：" + prog,
						"设计器 Packing 在这种情况下会改用 CiteTAP 内容"},
				}
				return common.Fail(me, *asJSON)
			}
		}
	}

	// 4) split + rewrite
	plan, err := split.Split(base, parsed, pkg)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	// 4b) gate2：不变量 I1–I15（对**原包**的 TAP 与编辑后的文档做一致性判定）
	if tapDoc0, terr := tapfile.Parse(pkg.Tap().Data); terr == nil && tapDoc0 != nil {
		g2 := verify.Gate2(base, parsed, tapDoc0)
		if g2.HasError() {
			if *asJSON {
				common.EmitJSON(w, map[string]any{"ok": false, "stage": "gate2", "findings": g2.Findings})
			} else {
				common.Line(w, "apply 失败（gate2 不变量 %s）：磁盘未动", g2.Summary())
				printFindings(w, g2.Findings, false)
			}
			if g2.HasDenied() {
				return 4
			}
			return 3
		}
	}
	// 写区段的前提是 workspace 处于 Unlocked（由 `tt dev tzc unlock` 显式迁移并二次确认）。
	// 设计指南 §6：apply 不重复弹警告；Locked 态改区段在 gate1/split 已被拒（退出码 4）。
	_ = yes
	newTap, err := tapfile.Rewrite(pkg.Tap().Data, plan.Ops...)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	newTgl := applyTglPatches(pkg, plan)

	// 5) 逐条目写回计划
	actions, err := pkg.Plan(pkgfile.Rebuild{Tap: newTap, Tgl: newTgl})
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	if *dryRun {
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": true, "dry_run": true, "plan": plan.Describe(),
				"entries": actions, "changed": plan.Changed, "added": plan.Added,
				"deleted": plan.Deleted, "sections": plan.Sections})
			return 0
		}
		common.Line(w, "--dry-run：将写入的逐条目 diff（未落盘、未 commit）")
		for _, a := range actions {
			mark := "不变"
			if a.Changed {
				mark = "改写"
			}
			common.Line(w, "  %-28s %s  %8d → %-8d  %s → %s", a.Name, mark, a.OldSize, a.NewSize,
				common.Short(a.OldSha256), common.Short(a.NewSha256))
		}
		common.Line(w, "计划：%s", plan.Describe())
		return 0
	}
	// 6) gate3：对将要产出的新包重跑合成
	g3, err := verify.Gate3(pkg, newTap, newTgl)
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	if g3.HasError() {
		if *asJSON {
			common.EmitJSON(w, map[string]any{"ok": false, "stage": "gate3", "findings": g3.Findings})
		} else {
			common.Line(w, "apply 失败（gate3 装载模拟）：磁盘未动")
			printFindings(w, g3.Findings, false)
		}
		return 3
	}
	// 7) 原子写
	newBytes, _, err := pkg.Build(pkgfile.Rebuild{Tap: newTap, Tgl: newTgl})
	if err != nil {
		return common.Fail(err, *asJSON)
	}
	writePath := pkgPath
	if *outPkg != "" {
		writePath = *outPkg
	}
	// 包是原地覆盖的：覆盖前留一份「上一步」副本，便于回滚
	backupPath := ""
	if writePath == pkgPath {
		if bp, berr := ws.BackupPackage(pkgPath); berr == nil {
			backupPath = bp
		} else {
			fmt.Fprintf(os.Stderr, "警告：备份上一步的包失败（继续写）：%v\n", berr)
		}
	}
	if err := store.AtomicWrite(writePath, newBytes); err != nil {
		return common.Fail(err, *asJSON)
	}
	// 8) 重算基线：**以刚写出的包为准**重新 synthesize + render（规范化）。
	//
	// 为什么不用编辑后的文本当基线：改名事务会改变点的身份（region 名
	// function.旧名 → function.新名），而围栏的锚定名在编辑时是不变的；
	// 只有从新包重渲染，基线的身份才与包一致。这一步同时规范化围栏字段
	// （fn/scope/desc）与行尾，并让 desc 镜像自动与描述块同步。
	newBaselineOK := false
	if np, nerr := pkgfile.Open(writePath, pkgfile.OpenOptions{}); nerr == nil {
		// pending_unlock 必须带过去：否则解锁后的围栏旗标会在重渲染时退回 READONLY。
		if ndoc, serr := synth.Synthesize(np, synth.Options{
			Only: base.Only, PendingUnlock: base.PendingUnlock,
		}); serr == nil {
			if nf, nregs, nspans, rerr := fence.Render(ndoc); rerr == nil {
				ndoc.Text, ndoc.Regions, ndoc.Spans = nf, nregs, nspans
				if uerr := ws.UpdateAfterApply(ndoc, nf); uerr == nil {
					// 工作文件也要换成规范化重渲染结果：否则它与基线不再逐字节对齐
					// （apply 会把点的 status 置 u，围栏行随之变化），
					// 下一次 apply 的 gate1 就会把「基线已有、工作文件还没有」当成被改动。
					if werr := ws.WriteEdited(nf); werr != nil {
						fmt.Fprintf(os.Stderr, "警告：刷新 prog.full.4gl 失败：%v\n", werr)
					}
					newBaselineOK = true
				} else {
					fmt.Fprintf(os.Stderr, "警告：重算基线失败：%v\n", uerr)
				}
			} else {
				fmt.Fprintf(os.Stderr, "警告：新包重渲染失败：%v\n", rerr)
			}
		} else {
			fmt.Fprintf(os.Stderr, "警告：新包重新合成失败：%v\n", serr)
		}
		// 刷新 manifest.pkg.sha256 / entries / snapshot：
		// 不刷的话，同一工作区第二次 apply 会被「源包自 export 之后已被改动」误拒（退出码 5）。
		if rerr := ws.RefreshPackage(np); rerr != nil {
			fmt.Fprintf(os.Stderr, "警告：刷新工作区包记录失败（下次 apply 可能被拒，请重新 export）：%v\n", rerr)
		}
	} else {
		// 退化路径：读不回新包时，至少用编辑后的文本刷新基线（保持旧行为）
		if uerr := ws.UpdateAfterApply(parsed.Doc, edited); uerr != nil {
			return common.Fail(uerr, *asJSON)
		}
		fmt.Fprintf(os.Stderr, "警告：写出的包无法读回（%v），基线按编辑文本刷新\n", nerr)
	}
	if !newBaselineOK {
		fmt.Fprintf(os.Stderr, "提示：基线未能从新包重算，建议 `tt dev tzc export` 重建工作区\n")
	}
	hash, gerr := ws.GitCommit("tdev apply: " + base.Prog)
	if gerr != nil {
		fmt.Fprintf(os.Stderr, "警告：git 提交失败（包已写成功）：%v\n", gerr)
	}
	newSha := model.Sha256Bytes(newBytes)

	rep := map[string]any{
		"ok": true, "pkg": writePath, "prog": base.Prog, "plan": plan.Describe(),
		"changed": plan.Changed, "added": plan.Added, "deleted": plan.Deleted,
		"sections": plan.Sections, "section_flag": plan.SetSectionFlag,
		"entries": actions, "pkg_sha256": newSha, "commit": hash, "prev_pkg": backupPath,
	}
	if *asJSON {
		common.EmitJSON(w, rep)
		return 0
	}
	common.Line(w, "apply 成功")
	common.Line(w, "  包：%s", writePath)
	common.Line(w, "  新包 sha256：%s", newSha)
	if backupPath != "" {
		common.Line(w, "  上一步的包已备份到：%s", backupPath)
	}
	common.Line(w, "  计划：%s", plan.Describe())
	for _, a := range actions {
		mark := "不变"
		if a.Changed {
			mark = "改写"
		}
		common.Line(w, "    %-28s %s  %8d → %-8d", a.Name, mark, a.OldSize, a.NewSize)
	}
	if plan.SetSectionFlag {
		common.Line(w, "  副作用：TAP 根 section_flag=\"Y\"（包从此永久处于已解开状态）")
	}
	if len(plan.Sections) > 0 {
		common.Line(w, "")
		common.Line(w, "提示：本包改过框架区段，**下次上传会触发服务器侧 adzi520 联动**（tdev 是本地工具，不代为执行）。")
	}
	common.Line(w, "  基线已重算、git 已提交：%s", common.Short(hash))
	return 0
}

// applyTglPatches 把区段正文补丁打进 .tgl（与 .tap 保持逐字节一致的同一份内容）。
func applyTglPatches(pkg *pkgfile.Package, plan *split.Plan) []byte {
	tgl := pkg.Tgl()
	if tgl == nil || len(plan.TglPatches) == 0 {
		return nil
	}
	body := append([]byte(nil), tgl.Data...)
	for _, p := range plan.TglPatches {
		out, err := tglfile.PatchSection(body, p.ID, p.Body, []byte("\r\n"))
		if err != nil {
			// gate3 会重新合成并报告；这里保留原样以免破坏 .tgl
			continue
		}
		body = out
	}
	return body
}

//---------------------------------------------------------------------------
