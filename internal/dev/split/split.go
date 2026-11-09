// Package split 实现设计指南 §5.6「回写层 —— Document' → TapOp + TglPatch」。
//
// 依据：
//   - 设计指南 §5.6、§4 规则 4/5、§7 红线 R1/R2/R4
//   - 反编译源码（TglTag 折叠是命门；改框架要写两处）：CodeEditorManager.cs:361-397（SaveADPContent）
//     CodeEditorManager.cs:400-496（SaveSectionContent，逐行把 EditObject 折回 TglTag）
//     CodeEditorManager.cs:314-352（GenerateTGL）+ :408（section_flag="Y"）
//     AddPointModel.cs:1086-1112（ToXML：status 只有 ' '/u/d；CREATE 直接 return null）
//     ProgramInformation.cs:96-108（新增点的 order = max+1）
package split

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"tt/internal/dev/fence"
	"tt/internal/dev/fgl"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/tapfile"
)

// ExitCodeError 让 CLI 把错误映射为退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// VerifyError = 退出码 3。
type VerifyError struct {
	Msg    string
	Detail []string
}

func (e *VerifyError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *VerifyError) ExitCode() int { return 3 }

// DeniedError = 退出码 4。
type DeniedError struct {
	Msg    string
	Detail []string
}

func (e *DeniedError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *DeniedError) ExitCode() int { return 4 }

// TglPatch 是要打进 .tgl 的区段正文补丁。
type TglPatch struct {
	ID   string
	Body []byte
}

// Plan 是一次写回计划。
type Plan struct {
	Ops []tapfile.Op
	// TglPatches 只有 --allow-sec 下改过区段时才非空。
	TglPatches []TglPatch
	// SetSectionFlag 表示需要把根 section_flag 置 "Y"。
	SetSectionFlag bool

	Changed  []string // 内容变了的点
	Renamed  []string // 结构事务：改名（"旧 → 新"）
	Added    []string // 新增的点
	Deleted  []string // 删除的点
	Sections []string // 改过正文的区段
}

// maxOrder 复刻 ProgramInformation.Add 的 order 分配：
// max(非删除自订定义点的 order) + 1；没有则 1。
func maxOrder(tapDoc *tapfile.Doc) int {
	max := 0
	for _, p := range tapDoc.Points {
		if tapfile.IsDeleted(p) {
			continue
		}
		n, _ := p.Attr("name")
		if !isSelfDef(n) {
			continue
		}
		v, ok := p.Attr("order")
		if !ok {
			continue
		}
		if k, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && k > max {
			max = k
		}
	}
	if max == 0 {
		return 1
	}
	return max + 1
}

func isSelfDef(name string) bool {
	return strings.HasPrefix(name, "function.") ||
		strings.HasPrefix(name, "dialog.") ||
		strings.HasPrefix(name, "report.")
}

// Split 依据基线 + 编辑后的文档产出写回操作。
//
// 授权检查在这里再做一遍（gate1 已报错，但 split 是写路径的最后一道）：
//   - 不可编辑区内容变了 → DeniedError
//   - 非 new="Y" 的点被删 → DeniedError
//   - 新增点不在 APPEND 锚点内 → fence.Parse 已经拦住
func Split(base *model.Document, parsed *fence.ParseResult, pkg *pkgfile.Package) (*Plan, error) {
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return nil, &VerifyError{Msg: "TAP 解析失败", Detail: []string{err.Error()}}
	}
	if _, perr := tapfile.Parse(base.Text); perr == nil {
		// 基线是围栏文本，不是 TAP；这里只是占位，无实际作用
		_ = perr
	}
	env := base.Env
	plan := &Plan{}

	// 逐点比对内容：用 parse 给出的**配对**，不按名字查表 ——
	// 真实包里存在区段 id 与点名同名（例如 wssp00316.process）的情形。
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil || b.Kind != model.RegionPoint {
			continue
		}
		bc := content(base, b)
		ec := content(parsed.Doc, e)
		// 行尾等价：编辑器把 CRLF 归一成 LF 不算改动（受保护字节从不写回包）
		if model.EqualEOL(bc, ec) {
			continue
		}
		if !b.Editable {
			return nil, &DeniedError{
				Msg:    "内容被改动，但该点不可编辑（写入被拒）",
				Detail: []string{b.Name, "deny=" + b.DenyCode, b.Reason},
			}
		}
		if e.BodySpan == nil && isSelfDef(b.Name) {
			return nil, &VerifyError{
				Msg:    "改动后的正文不是可解析的函数块，拒绝写回",
				Detail: []string{b.Name},
			}
		}
		if tapDoc.Point(b.Name) == nil {
			// TAP 里没有这个点（TGL 有占位符 → 设计器造空点）。写回 = 新增一个 <point>。
			plan.Ops = append(plan.Ops, tapfile.AddPoint{
				Attrs:   newPointAttrs(b.Name, nextOrder(tapDoc, plan), env, b.Meta),
				Content: ec,
			})
			plan.Added = append(plan.Added, b.Name)
			continue
		}
		plan.Ops = append(plan.Ops,
			tapfile.SetPointCDATA{Name: b.Name, Content: ec},
			tapfile.SetPointAttr{Name: b.Name, Key: "status", Value: "u"},
		)
		plan.Changed = append(plan.Changed, b.Name)
	}

	// 1b) 结构事务：改名（设计指南 §4.1）
	//
	// 复刻 ProgramInformation.Modify（ProgramInformation.cs:116-134）：
	//   ① 清掉同名旧墓碑  ③ Clone 一份 status="d" 的墓碑（保留旧名与原 CDATA）
	//   ② 活点置 MODIFY 并把 Name 改为 function.新名
	// 产出的 .tap 与设计器手工改名的产物语义等价：一对 <point>（旧 status=d + 新 status=u）。
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil || b.Kind != model.RegionPoint || b.Plain {
			continue
		}
		_, toFn, renamed := pr.StructFieldChanged("fn")
		if !renamed {
			continue
		}
		oldName := b.Name
		if bare(toFn) == "" {
			return nil, &VerifyError{
				Msg:    "改名目标为空",
				Detail: []string{b.Name, "围栏 fn 字段不能为空"},
			}
		}
		newName := prefixOf(oldName) + bare(toFn)
		if newName == oldName {
			continue
		}
		// ② 新块 = 编辑后内容里把签名行换成新签名（逐字节保留其余部分）
		block, err := renameBlock(parsed.Doc, b, e, toFn, pr)
		if err != nil {
			return nil, err
		}
		// 旧内容：**逐字节**取基线的 CDATA（Clone 保留旧名与原内容）
		oldContent := base.Text[b.ContentSpan.Start:b.ContentSpan.End]
		if tapDoc.Point(oldName) == nil {
			return nil, &VerifyError{
				Msg:    "要改名的点在 TAP 里不存在，无法生成墓碑对",
				Detail: []string{oldName},
			}
		}
		// ③ 墓碑：复制基线的全部属性，只把 status 置 d、name 保持旧名
		tombAttrs := map[string]string{}
		if el := tapDoc.Point(oldName); el != nil {
			for _, a := range el.Attrs {
				tombAttrs[a.Name] = a.Value
			}
		}
		tombAttrs["name"] = oldName
		tombAttrs["status"] = "d"
		if _, ok := tombAttrs["new"]; !ok {
			tombAttrs["new"] = "Y"
		}
		plan.Ops = append(plan.Ops,
			// ① 清掉同名旧墓碑（没有则 no-op）
			tapfile.RemoveTombstones{Name: oldName},
			// ② 活点改名 → function.新名（name 属性 + 整块重写 + status=u）
			tapfile.RenamePoint{From: oldName, To: newName},
			tapfile.SetPointCDATA{Name: newName, Content: block},
			tapfile.SetPointAttr{Name: newName, Key: "status", Value: "u"},
			// ③ 补齐墓碑（旧名 + 原 CDATA 逐字节 + status=d）。
			//    放在改名之后：此时已无同名活点，AddPoint 的重名检查自然通过。
			tapfile.AddPoint{Attrs: tombAttrs, Content: oldContent},
		)
		plan.Renamed = append(plan.Renamed, oldName+" → "+newName)
		plan.Changed = append(plan.Changed, newName)
	}

	// 2) 删除的点
	for _, d := range parsed.Deleted {
		if d.Kind != model.RegionPoint {
			return nil, &VerifyError{Msg: "区段不可删除", Detail: []string{d.Name}}
		}
		if !strings.EqualFold(d.Meta["new"], "Y") {
			return nil, &DeniedError{
				Msg:    "只有 new=\"Y\" 的自订点才能删除（与设计器一致）",
				Detail: []string{d.Name, "new=" + d.Meta["new"]},
			}
		}
		if !d.Editable {
			return nil, &DeniedError{
				Msg:    "删除被拒：该点不可编辑",
				Detail: []string{d.Name, "deny=" + d.DenyCode, d.Reason},
			}
		}
		if tapDoc.Point(d.Name) == nil {
			return nil, &VerifyError{
				Msg:    "要删除的点在 TAP 里不存在（无内容可标记删除）",
				Detail: []string{d.Name},
			}
		}
		// 设计器语义：删点 = status="d" 且**保留原内容**（I5）
		plan.Ops = append(plan.Ops, tapfile.MarkDeleted{Name: d.Name})
		plan.Deleted = append(plan.Deleted, d.Name)
	}

	// 3) 新增的点（APPEND 锚点内）
	for _, a := range parsed.Appended {
		ec := content(parsed.Doc, a)
		if a.BodySpan == nil && isSelfDef(a.Name) {
			return nil, &VerifyError{
				Msg:    "新增点的正文不是可解析的函数块，拒绝写回",
				Detail: []string{a.Name},
			}
		}
		meta := map[string]string{}
		for k, v := range a.Meta {
			meta[k] = v
		}
		plan.Ops = append(plan.Ops, tapfile.AddPoint{
			Attrs:   newPointAttrs(a.Name, nextOrder(tapDoc, plan), env, meta),
			Content: ec,
		})
		plan.Added = append(plan.Added, a.Name)
	}

	// 4) 区段正文（仅 --allow-sec 下可编辑区段）
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil || b.Kind != model.RegionSection {
			continue
		}
		bc := base.OwnBytes(b)
		ec := parsed.Doc.OwnBytes(e)
		// 行尾等价：编辑器把 CRLF 归一成 LF 不算改动（受保护字节从不写回包）
		if model.EqualEOL(bc, ec) {
			continue
		}
		if !b.Editable {
			detail := []string{b.Name, "deny=" + b.DenyCode, b.Reason}
			if b.DenyCode == model.DenySecLocked {
				detail = append(detail, "先执行 `tt dev tzc unlock <dir>`（解锁改变的是门，不是所有房间）")
			}
			return nil, &DeniedError{
				Msg:    "区段正文被改动，但该区段不可编辑（写入被拒）",
				Detail: detail,
			}
		}
		folded, err := foldSection(parsed.Doc, b, e)
		if err != nil {
			return nil, err
		}
		plan.Ops = append(plan.Ops,
			tapfile.SetSectionCDATA{ID: b.Name, Content: folded},
			tapfile.SetSectionAttr{ID: b.Name, Key: "status", Value: "u"},
		)
		plan.TglPatches = append(plan.TglPatches, TglPatch{ID: b.Name, Body: folded})
		plan.Sections = append(plan.Sections, b.Name)
	}
	if len(plan.Sections) > 0 {
		// 设计器副作用：改过区段 → TAP 根 section_flag="Y"（CodeEditorManager.cs:408，只置 Y）
		plan.Ops = append(plan.Ops, tapfile.SetRootAttr{Key: "section_flag", Value: "Y"})
		plan.SetSectionFlag = true
	}
	return plan, nil
}

// nextOrder 给新增点分配 order（max+1），同一次 apply 里多个新增点依次递增。
func nextOrder(tapDoc *tapfile.Doc, plan *Plan) int {
	base := maxOrder(tapDoc)
	extra := 0
	for _, n := range plan.Added {
		if isSelfDef(n) {
			extra++
		}
	}
	return base + extra
}

// newPointAttrs 生成新增 <point> 的属性集。
//
// 关键偏差 D-1：status 写 "u" 而**不是** "c"。
// AddPointModel.ToXML() 在 Status==CREATE 时直接 return null（AddPointModel.cs:1090-1093），
// 所以写 status="c" 的点会在设计器下一次保存时被静默丢弃。
// 属性集合与顺序取自设计器 ctor（AddPointModel.cs:64-75）。
func newPointAttrs(name string, order int, env model.EnvContext, meta map[string]string) map[string]string {
	attrs := map[string]string{
		"name":      name,
		"order":     strconv.Itoa(order),
		"ver":       "",
		"cite_std":  "N",
		"new":       "Y",
		"src":       env.Env,
		"status":    "u",
		"ch":        "",
		"ind_fun":   env.Topind,
		"ind_extra": "N",
	}
	if !isSelfDef(name) {
		// 裸名插入点没有 order 概念（设计器 SortIndex setter 对非自订定义点拒绝赋值）
		attrs["order"] = ""
	}
	if env.IsTopstdMode {
		attrs["src"] = "s"
	}
	return attrs
}

// foldSection 把编辑后的区段正文折回占位符版（复刻 SaveSectionContent 的行为）。
//
// 规则（红线 R4）：区段正文里出现的每个点块，必须换回它**原始的 TglTag 占位符原文**；
// 找不到 TglTag 的点块一律报错，绝不把展开后的函数写进区段。
func foldSection(edited *model.Document, baseSection, editedSection *model.Region) ([]byte, error) {
	// 基线子区间按顺序对应编辑后的前 N 个子区间
	baseChildren := make([]*model.Region, 0, len(baseSection.Children))
	for _, c := range baseSection.Children {
		if c.Kind == model.RegionPoint {
			baseChildren = append(baseChildren, c)
		}
	}
	editedChildren := make([]*model.Region, 0, len(editedSection.Children))
	for _, c := range editedSection.Children {
		if c.Kind == model.RegionPoint {
			editedChildren = append(editedChildren, c)
		}
	}
	var out bytes.Buffer
	pos := editedSection.ContentSpan.Start
	for i, ch := range editedChildren {
		if ch.FullSpan.Start > pos {
			out.Write(edited.Text[pos:ch.FullSpan.Start])
		}
		var tag []byte
		if i < len(baseChildren) {
			tag = baseChildren[i].TglTag
		}
		if len(tag) == 0 {
			return nil, &VerifyError{
				Msg: "区段正文里有点块没有原始 TglTag，无法折叠回占位符（红线 R4：绝不把展开后的函数写进区段）",
				Detail: []string{
					"区段：" + baseSection.Name,
					"点：" + ch.Name,
				},
			}
		}
		out.Write(tag)
		pos = ch.FullSpan.End
	}
	if editedSection.ContentSpan.End > pos {
		out.Write(edited.Text[pos:editedSection.ContentSpan.End])
	}
	return out.Bytes(), nil
}

func content(d *model.Document, r *model.Region) []byte {
	if r == nil || r.ContentSpan.End > len(d.Text) || r.ContentSpan.Start > r.ContentSpan.End {
		return nil
	}
	return d.Text[r.ContentSpan.Start:r.ContentSpan.End]
}

// Describe 返回人类可读的写回计划摘要。
func (p *Plan) Describe() string {
	var parts []string
	if len(p.Renamed) > 0 {
		parts = append(parts, fmt.Sprintf("改名 %d 个点（墓碑对：旧 status=d + 新 status=u）", len(p.Renamed)))
	}
	if len(p.Changed) > 0 {
		parts = append(parts, fmt.Sprintf("改动 %d 个点", len(p.Changed)))
	}
	if len(p.Added) > 0 {
		parts = append(parts, fmt.Sprintf("新增 %d 个点", len(p.Added)))
	}
	if len(p.Deleted) > 0 {
		parts = append(parts, fmt.Sprintf("删除 %d 个点", len(p.Deleted)))
	}
	if len(p.Sections) > 0 {
		parts = append(parts, fmt.Sprintf("改动 %d 个区段（%s）", len(p.Sections),
			"TglTag 折叠 + .tgl 补丁 + section_flag=Y"))
	}
	if len(parts) == 0 {
		return "无改动"
	}
	return strings.Join(parts, "、")
}

// Renamed 记录本次的改名（人类可读的 "旧 → 新"）。
// （字段声明见 Plan 定义处。）

// bare 去掉点名前缀与参数列表。
func bare(n string) string {
	for _, p := range []string{"function.", "dialog.", "report."} {
		n = strings.TrimPrefix(n, p)
	}
	if i := strings.IndexByte(n, '('); i >= 0 {
		n = n[:i]
	}
	return n
}

// prefixOf 取点名的种类前缀（含点号）。
func prefixOf(n string) string {
	switch {
	case strings.HasPrefix(n, "function."):
		return "function."
	case strings.HasPrefix(n, "dialog."):
		return "dialog."
	case strings.HasPrefix(n, "report."):
		return "report."
	}
	return "function."
}

var (
	reSigScopeR = regexp.MustCompile(`(?i)^([ \t]*)(PUBLIC|PRIVATE)([ \t]+)`)
	reSigKindR  = regexp.MustCompile(`(?i)^([ \t]*)(FUNCTION|DIALOG|REPORT)([ \t]*)`)
)

// renameBlock 生成改名后的点块：把编辑后内容里的签名行换成新签名，其余逐字节保留。
//
// 签名行口径对齐 AddPointModel.ToString()（AddPointModel.cs:1008-1060）：
//   - FUNCTION 恒写 scope（`PRIVATE FUNCTION x(...)`）
//   - DIALOG/REPORT 在 PUBLIC 时**省略** scope 前缀（`DIALOG x`）
func renameBlock(doc *model.Document, base, edited *model.Region, toFn string, pr fence.Pair) ([]byte, error) {
	if edited.SignatureSpan == nil {
		return nil, &VerifyError{
			Msg:    "改动后的点找不到签名行，无法改名",
			Detail: []string{edited.Name},
		}
	}
	sig := doc.Text[edited.SignatureSpan.Start:edited.SignatureSpan.End]
	scope := pr.EditedMeta.KV["scope"]
	if scope == "" {
		scope = edited.Scope
	}
	if scope == "" {
		scope = base.Scope
	}
	if scope == "" {
		scope = "PRIVATE"
	}
	kind, _ := fgl.EnvelopeKindFor(edited.Name)
	newSig := BuildSignature(string(sig), kind, scope, toFn)

	// 用编辑后的内容，替换签名行字节
	content := doc.Text[edited.ContentSpan.Start:edited.ContentSpan.End]
	relS := edited.SignatureSpan.Start - edited.ContentSpan.Start
	relE := edited.SignatureSpan.End - edited.ContentSpan.Start
	if relS < 0 || relE > len(content) || relS > relE {
		return nil, &VerifyError{Msg: "内部错误：签名行区间越界", Detail: []string{edited.Name}}
	}
	out := make([]byte, 0, len(content)+len(newSig))
	out = append(out, content[:relS]...)
	out = append(out, newSig...)
	out = append(out, content[relE:]...)
	return out, nil
}

// BuildSignature 用原签名行的缩进与关键字，拼出新的签名行（改名事务与
// `tt dev tzc rename` 共用同一口径，避免两处漂移）。
func BuildSignature(oldSig, kind, scope, fn string) string {
	indent := ""
	if m := reSigKindR.FindStringSubmatch(oldSig); m != nil {
		indent = m[1]
	} else if m := reSigScopeR.FindStringSubmatch(oldSig); m != nil {
		indent = m[1]
	}
	k := strings.ToUpper(kind)
	if k == "" {
		k = "FUNCTION"
	}
	// DIALOG/REPORT：PUBLIC 省略限定符（对齐设计器 ToString）
	if (k == "DIALOG" || k == "REPORT") && strings.EqualFold(scope, "PUBLIC") {
		return indent + k + " " + fn
	}
	return indent + strings.ToUpper(scope) + " " + k + " " + fn
}
