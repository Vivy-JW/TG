package verify

import (
	"fmt"
	"regexp"
	"strings"

	"tt/internal/dev/fence"
	"tt/internal/dev/fgl"
	"tt/internal/dev/model"
	"tt/internal/dev/tapfile"
	"tt/internal/dev/tglfile"
)

// Package structtx（结构事务）的实现放在 verify 里而不是独立包：
// 它要返回 verify.Finding，独立包会造成 verify ↔ structtx 的循环依赖。
// 检测用的原语在 fence.Pair 上（StructFieldChanged），split 侧复用同一原语。

// StructKind 是结构事务的种类（设计指南 §4.1）。
type StructKind string

const (
	StructRename          StructKind = "Rename"
	StructScopeChange     StructKind = "ScopeChange"
	StructDescChange      StructKind = "DescChange"
	StructSignatureChange StructKind = "SignatureChange"
)

// StructEdit 是一次检测到的结构事务。
type StructEdit struct {
	Region string
	Kind   StructKind
	From   string
	To     string
	// 基线 / 编辑后的签名行原文（SignatureChange / V1 用）
	BaseSig, EditedSig string
}

// DetectStructural 找出编辑后文档相对基线的结构事务。
//
// 判据（设计指南 §4.1）：围栏 fn/scope/desc 字段，或签名行相对 base 发生变化。
func DetectStructural(base *model.Document, parsed *fence.ParseResult) []StructEdit {
	var out []StructEdit
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil || b.Kind != model.RegionPoint || b.Plain {
			continue
		}
		if from, to, ch := pr.StructFieldChanged("fn"); ch {
			out = append(out, StructEdit{Region: b.Name, Kind: StructRename, From: from, To: to})
		}
		if from, to, ch := pr.StructFieldChanged("scope"); ch {
			out = append(out, StructEdit{Region: b.Name, Kind: StructScopeChange, From: from, To: to})
		}
		if from, to, ch := pr.StructFieldChanged("desc"); ch {
			out = append(out, StructEdit{Region: b.Name, Kind: StructDescChange, From: from, To: to})
		}
		bs := spanText(base, b.SignatureSpan)
		es := spanText(parsed.Doc, e.SignatureSpan)
		if bs != es {
			out = append(out, StructEdit{
				Region: b.Name, Kind: StructSignatureChange,
				From: bs, To: es, BaseSig: bs, EditedSig: es,
			})
		}
	}
	return out
}

func spanText(d *model.Document, r *model.ByteRange) string {
	if r == nil || r.End <= r.Start || r.End > len(d.Text) {
		return ""
	}
	return string(d.Text[r.Start:r.End])
}

// 程序类型 → 允许的 scope（FunctionInfoWindow.xaml.cs:143-192）。
//
//	B        → 强制 PUBLIC（锁定）
//	M/G/X/Z/Q→ 强制 PRIVATE（锁定）
//	S        → 默认 PRIVATE（可改）
//	W        → 默认 PUBLIC（可改）
func scopeRule(progType string) (must string, free bool) {
	switch strings.ToUpper(progType) {
	case "B":
		return "PUBLIC", false
	case "M", "G", "X", "Z", "Q":
		return "PRIVATE", false
	case "S":
		return "PRIVATE", true
	case "W":
		return "PUBLIC", true
	}
	return "", true
}

// validateStructural 跑 V1–V7（设计指南 §4.1）。全部 error 级。
func validateStructural(edits []StructEdit, base *model.Document, parsed *fence.ParseResult,
	tapDoc *tapfile.Doc, prog string) *Report {

	rep := &Report{}
	if len(edits) == 0 {
		return rep
	}
	doc := parsed.Doc
	env := doc.Env

	// ---- V1 名字一致：围栏 fn == 签名行函数名（ComplexException 前置）----
	for _, pr := range parsed.Pairs {
		e := pr.Edited
		if e == nil || pr.Base == nil || e.Kind != model.RegionPoint || e.Plain {
			continue // 新增点（Base==nil）没有基线可对，结构事务只针对既有自订定义点
		}
		sig := spanText(doc, e.SignatureSpan)
		if sig == "" {
			continue
		}
		want := signatureName(sig)
		declared := pr.EditedMeta.KV["fn"]
		if declared == "" {
			// 旧工作区（围栏无 fn 字段）：只要求签名行没被改，否则无法做事务校验。
			if bs := spanText(base, pr.Base.SignatureSpan); bs != sig {
				rep.Add(Finding{Code: "V1", Severity: SevError, Region: e.Name,
					Message: "签名行被改动，但围栏没有 fn 字段（工作区由旧版本创建）",
					Detail: []string{"请重新 export 后再做结构事务（改名/改签名）",
						"原：" + clip(bs), "新：" + clip(sig)}})
			}
			continue
		}
		if declared != want {
			rep.Add(Finding{Code: "V1", Severity: SevError, Region: e.Name,
				Message: "围栏 fn 与签名行的函数名不一致（设计器装载时会抛 ComplexException）",
				Detail: []string{"fn=" + declared, "签名行=" + want,
					"用 `tt dev tzc rename` 可原子同步这两处"}})
		}
	}

	// ---- V3 新名查重（含本次会话内的冲突） ----
	existing := map[string]bool{}
	for _, r := range base.AllRegions() {
		if r.Kind == model.RegionPoint {
			existing[bareName(r.Name)] = true
		}
	}
	for _, p := range tapDoc.Points {
		n, _ := p.Attr("name")
		existing[bareName(n)] = true
	}
	targets := map[string]string{}
	for _, e := range edits {
		if e.Kind != StructRename {
			continue
		}
		to := bareName(e.To)
		if to == "" {
			continue
		}
		if prev, dup := targets[to]; dup {
			rep.Add(Finding{Code: "V3", Severity: SevError, Region: e.Region,
				Message: "本次会话里有两次改名指向同一目标",
				Detail:  []string{prev + " 与 " + e.Region + " 都改成 " + to}})
		}
		targets[to] = e.Region
		// 目标名与「别的现存点」冲突（排除自己）
		if existing[to] && to != bareName(e.Region) {
			rep.Add(Finding{Code: "V3", Severity: SevError, Region: e.Region,
				Message: "新函数名与现存点冲突",
				Detail:  []string{"目标 " + to + " 已存在（point 或 TGL 占位符裸名）"}})
		}
		// 交换式改名（A→B 且 B→A）
		for _, other := range edits {
			if other.Region == e.Region || other.Kind != StructRename {
				continue
			}
			if bareName(other.To) == bareName(e.Region) && bareName(e.To) == bareName(other.Region) {
				rep.Add(Finding{Code: "V3", Severity: SevError, Region: e.Region,
					Message: "检测到交换式改名（A→B 同时 B→A），请分两次 apply 完成",
					Detail:  []string{e.Region + " ↔ " + other.Region}})
			}
		}
	}

	// ---- V2 调用点清零（旧名在所有 Region 正文中的残留引用） ----
	for _, e := range edits {
		if e.Kind != StructRename {
			continue
		}
		old := bareName(e.Region)
		if old == "" {
			continue
		}
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`)
		for _, r := range doc.AllRegions() {
			if r.Name == e.Region {
				continue // 自己那一块（签名行/正文头部）不算残留
			}
			if r.Kind == model.RegionPoint && bareName(r.Name) == old {
				continue // 同名墓碑/新点自身
			}
			c := doc.OwnBytes(r)
			if !re.Match(c) {
				continue
			}
			if r.Editable {
				rep.Add(Finding{Code: "V2", Severity: SevError, Region: r.Name,
					Message: "旧函数名仍有残留引用（在可编辑区内）",
					Detail: []string{"改名 " + old + " → " + bareName(e.To),
						"位置：Region " + r.Name + "（可编辑）",
						"先把调用点改成新名，再重新 apply"}})
				continue
			}
			// 只读区段（含 Locked 态的区段）里的调用点改不了 → 拒绝
			detail := []string{"改名 " + old + " → " + bareName(e.To),
				"位置：Region " + r.Name + "（只读，deny=" + r.DenyCode + "）"}
			if r.DenyCode == model.DenySecLocked {
				detail = append(detail, "该区段属于框架且框架未解开：先 `tt dev tzc unlock`，或放弃这次改名")
			} else {
				detail = append(detail, "只读区段里的调用点无法修改：请放弃这次改名，或先由设计器调整框架")
			}
			rep.Add(Finding{Code: "V2", Severity: SevError, Denied: true, Region: r.Name,
				Message: "旧函数名残留在只读区段里（写入被拒）", Detail: detail})
		}
	}

	// ---- V4 命名规范（对齐设计器：仅 INFORMATION 提示 → warn） ----
	prefix := prog + "_"
	if env.IsIndFun {
		prefix = prog + "_" + env.Topind + "_"
	}
	for _, e := range edits {
		if e.Kind != StructRename {
			continue
		}
		to := bareName(e.To)
		if to == "" {
			continue
		}
		if !strings.HasPrefix(to, prefix) {
			rep.Add(Finding{Code: "V4", Severity: SevWarn, Region: e.Region,
				Message: "函数名未以程序名开头（设计器只给 INFORMATION 提示，不阻断）",
				Detail:  []string{"新名：" + to, "建议前缀：" + prefix}})
		}
	}

	// ---- V5 scope 约束（按程序 type）----
	//
	// ⚠️ 与设计指南的偏差 DV-3（源码直证）：`FunctionInfoWindow.xaml.cs:143-192` 的
	// type→scope 映射是**新建点对话框的默认值**（且整段被 `useDefaultScope` 门控），
	// 不是对既有文本的硬约束 —— 真实语料里就有 type="M" 而函数声明为 PUBLIC 的点
	// （aapt300(c).tzc 的 function.aapt300_curr_info_master）。把它当 error 会拒掉真实包。
	// 因此只在**本次改了 scope**时给 warn，提示「设计器对话框会锁住这个字段」。
	locksTo, free := scopeRule(env.ProgType) // free=false 表示该类型在对话框里锁定 scope
	for _, e := range edits {
		if e.Kind != StructScopeChange {
			continue
		}
		if !free && locksTo != "" && !strings.EqualFold(e.To, locksTo) {
			rep.Add(Finding{Code: "V5", Severity: SevWarn, Region: e.Region,
				Message: "scope 与程序类型的对话框默认值不一致（设计器新建点时会锁定该字段）",
				Detail: []string{"程序 type=" + env.ProgType + " 的对话框默认 " + locksTo,
					"本次改为：" + e.To,
					"既有文本仍被接受：真实语料里有 type=M 却声明 PUBLIC 的点"}})
		}
	}

	// ---- V6 描述块合法（非空行必须以 # 开头；**空行合法**，AddPointModel.cs:253） ----
	rePound := regexp.MustCompile(`^\s*#`)
	// 本次改动过描述块的点 → error；存量数据（没改）→ warn：
	// 真实语料里存在历史遗留的不规范描述行，把存量当 error 会拒掉真实包。
	descTouched := map[string]bool{}
	for _, e := range edits {
		if e.Kind == StructDescChange {
			descTouched[e.Region] = true
		}
	}
	for _, r := range doc.AllRegions() {
		if r.Kind != model.RegionPoint || r.Plain || r.DescSpan == nil {
			continue
		}
		lvl := SevWarn
		if descTouched[r.Name] {
			lvl = SevError
		}
		txt := spanText(doc, r.DescSpan)
		off := r.DescSpan.Start // 逐行累加，把「描述块第 N 行」换算成文档里的绝对行号
		for i, ln := range strings.Split(txt, "\n") {
			raw := ln
			ln = strings.TrimRight(ln, "\r")
			if strings.TrimSpace(ln) == "" {
				off += len(raw) + 1
				continue // 空行合法
			}
			if !rePound.MatchString(ln) {
				rep.AddAt(Finding{Code: "V6", Severity: lvl, Region: r.Name,
					Message: "描述块里有非 # 开头的行（设计器表单校验会报错：注释必须以#开头）",
					Detail:  []string{fmt.Sprintf("第 %d 行：%s", i+1, clip(ln))}}, doc, FileEdited, off)
				break
			}
			off += len(raw) + 1
		}
	}

	// ---- V7 目标必须是自订定义点且可编辑（对齐 CanFunctionModify） ----
	for _, e := range edits {
		r := doc.FindRegion(e.Region)
		if r == nil {
			continue
		}
		if r.Plain || r.Kind != model.RegionPoint {
			rep.Add(Finding{Code: "V7", Severity: SevError, Denied: true, Region: e.Region,
				Message: "目标不是自订定义点（普通插入点没有函数身份，不支持改名/改签名/改描述）"})
			continue
		}
		if !r.Editable {
			rep.Add(Finding{Code: "V7", Severity: SevError, Denied: true, Region: e.Region,
				Message: "目标点不可编辑，结构事务被拒（写入被拒）",
				Detail:  []string{"deny=" + r.DenyCode, r.Reason}})
		}
	}

	// ---- 描述冲突：块与 desc 字段同时改且不一致 → 意图不明 ----
	for _, e := range edits {
		if e.Kind != StructDescChange {
			continue
		}
		pr, ok := pairOf(parsed, e.Region)
		if !ok {
			continue
		}
		blockChanged := !model.EqualEOL(base.OwnBytes(pr.Base), doc.OwnBytes(pr.Edited))
		if blockChanged && normalizeDesc(e.To) != normalizeDesc(descBlockText(doc, pr.Edited)) {
			rep.Add(Finding{Code: "V6", Severity: SevError, Region: e.Region,
				Message: "描述块与围栏 desc 字段同时被改且不一致（意图不明）",
				Detail: []string{"请只改一处：改描述块文字，或用 `tt dev tzc rename <dir> <旧名> <新名> --desc <描述>`",
					"desc 字段：" + clip(e.To),
					"描述块：" + clip(descBlockText(doc, pr.Edited))}})
		}
	}
	return rep
}

func pairOf(parsed *fence.ParseResult, name string) (fence.Pair, bool) {
	for _, pr := range parsed.Pairs {
		if pr.Base != nil && pr.Base.Name == name {
			return pr, true
		}
	}
	return fence.Pair{}, false
}

// normalizeDesc 归一描述文本（行尾 + 每行去前导 '# '）用于比较。
func normalizeDesc(s string) string {
	var out []string
	for _, ln := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		ln = strings.TrimSpace(ln)
		ln = strings.TrimPrefix(ln, "#")
		out = append(out, strings.TrimSpace(ln))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// descBlockText 取某 Region 的描述块文本。
func descBlockText(d *model.Document, r *model.Region) string {
	return spanText(d, r.DescSpan)
}

// ValidateStructural 单独暴露结构事务校验（`rename` / `newfn` 的预检用）。
//
// 与 gate2 里的调用同源，避免"命令行预检通过、apply 又拒"的口径漂移。
func ValidateStructural(base *model.Document, parsed *fence.ParseResult, tapDoc *tapfile.Doc) *Report {
	edits := DetectStructural(base, parsed)
	if len(edits) == 0 {
		return &Report{}
	}
	return validateStructural(edits, base, parsed, tapDoc, base.Prog)
}

// bareName 去掉点名前缀与参数列表：`function.aapp131_calc(p_a)` → `aapp131_calc`。
func bareName(n string) string {
	for _, p := range []string{"function.", "dialog.", "report."} {
		n = strings.TrimPrefix(n, p)
	}
	if i := strings.IndexByte(n, '('); i >= 0 {
		n = n[:i]
	}
	return n
}

// signatureName 从签名行取「名字(参数)」（与时/ fence/synth 同口径）。
func signatureName(sigLine string) string {
	s := strings.TrimLeft(sigLine, " \t")
	if m := reSigScopeV.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	if m := reSigKindV.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	s = strings.TrimLeft(s, " \t")
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " \t")
}

var (
	reSigScopeV = regexp.MustCompile(`(?i)^(PUBLIC|PRIVATE)[ \t]+`)
	reSigKindV  = regexp.MustCompile(`(?i)^(FUNCTION|DIALOG|REPORT)[ \t]*`)
)

// UnlockHint 供 gate2 在 Locked 且改区段时给出统一指引。
const UnlockHint = "先执行 `tt dev tzc unlock <dir>`（解锁是一次单向、有代价的状态迁移）"

var _ = fgl.EnvelopeKindFor
var _ = tglfile.AnchorFunction
