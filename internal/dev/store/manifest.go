package store

import (
	"bytes"
	"encoding/xml"
	"path/filepath"
	"strings"

	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
)

//---------------------------------------------------------------------------
// manifest.json / regions.json 的数据结构（人机共读）
//---------------------------------------------------------------------------

// Manifest 是 manifest.json 的顶层结构：让 AI / 人一眼看懂「这是什么包、
// 有哪些条目、哪些 Region 能改、为什么不能改」。
//
// 全部字段都由输入推导，**不含任何时间戳**（红线 R7）。
type Manifest struct {
	Tool        string `json:"tool"` // "tdev tzc"
	ToolVersion string `json:"tool_version"`
	Prog        string `json:"prog"`
	Module      string `json:"module"`
	ErpVer      string `json:"erpver"`
	Env         string `json:"env"`
	Kind        string `json:"kind"`
	Pkg         PkgRef `json:"pkg"`
	// Section 是框架解开状态（设计指南 §3 / §4.2）。v1 的 export.allow_sec 已移除。
	Section SectionInfo     `json:"section"`
	Export  ExportMode      `json:"export"`
	Anchors map[string]bool `json:"anchors"`
	Entries []EntryRef      `json:"entries"`
	Regions []RegionSummary `json:"regions"`
	Spans   []SpanSummary   `json:"spans"`
	Stats   Stats           `json:"stats"`
}

// PkgRef 指向导出时用的原包。
type PkgRef struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
	Ver    string `json:"ver"`
}

// SectionInfo 是 manifest.section：框架解开状态的机器可读视图。
type SectionInfo struct {
	// State = Locked | Unlocked（由 TAP 根 section_flag 与 pending_unlock 合并得出）。
	State string `json:"state"`
	// SectionFlag / StdSectionVerify 是包里的原始事实（便于人类核对授权来源）。
	SectionFlag      string `json:"section_flag"`
	StdSectionVerify string `json:"std_section_verify"`
	// UnlockedBy = pkg（包本来就是）/ unlock-cmd（workspace 迁移）/ ""。
	UnlockedBy string `json:"unlocked_by"`
	// PendingUnlock 为真表示 unlock 已执行但尚未 apply 落盘（TAP 根 section_flag 还是 N）。
	PendingUnlock bool `json:"pending_unlock"`
}

// ExportMode 记录导出范围（--only）。
type ExportMode struct {
	Only []string `json:"only"`
	// AllowSecLegacy 只用于**读** v1 工作区的 `export.allow_sec`（迁移用）。
	// 永不写出：nil 时 JSON 里不出现该键。
	AllowSecLegacy *bool `json:"allow_sec,omitempty"`
}

// EntryRef 是 snapshot/index.json 的一条：原包条目清单。
//
// 说明：条目名 / 摘要 / 大小 / 角色四件套足够 apply 时定位与校验；
// crc32 / method / modtime / order 属于 zip 细节，写回由 pkgfile.Build 透传原字节，
// 工作区不需要它们（保留 EntryRef 的既定字段集）。
type EntryRef struct {
	Name   string `json:"name"`
	Sha256 string `json:"sha256"`
	Size   int    `json:"size"`
	Role   string `json:"role"` // tap|tgl|4gl|ver|bdx|other
}

// RegionSummary 是 manifest 里的 Region 条目：常用 meta 直接展开，便于 AI 阅读。
type RegionSummary struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Editable bool   `json:"editable"`
	DenyCode string `json:"deny_code"`
	Reason   string `json:"reason"`
	// 以下展开自 model.Region.Meta（缺失键 → 空串）
	Status  string `json:"status"`
	Src     string `json:"src"`
	New     string `json:"new"`
	Order   string `json:"order"`
	CiteStd string `json:"cite_std"`
	Edit    string `json:"edit"`
	Mark    string `json:"mark"`

	Origin     string `json:"origin"`
	AnchorType string `json:"anchor_type"`
	Append     bool   `json:"append"`
	Sha256     string `json:"sha256"`
}

// SpanSummary 是不能归属任何 Region 的字节段。
type SpanSummary struct {
	Name   string `json:"name"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Sha256 string `json:"sha256"`
}

// Stats 是导出规模统计（含嵌套子区间）。
type Stats struct {
	Points   int `json:"points"`
	Sections int `json:"sections"`
	Editable int `json:"editable"`
	Readonly int `json:"readonly"`
}

// RegionsFile 是 .tdev/regions.json 的顶层结构。
//
// 它是 apply 的**唯一区间来源**（连同守门规则）：Region 表 + spans + 文本长度 +
// 基线摘要 + 环境 / 导出模式。
type RegionsFile struct {
	Prog       string           `json:"prog"`
	BaseSha256 string           `json:"base_sha256"`
	TextLen    int              `json:"text_len"`
	Env        model.EnvContext `json:"env"`
	// Section 与 manifest.section 同源（regions.json 是 apply 的区间来源，
	// 而区段可编辑性依赖 state，所以这里也必须带上）。
	Section SectionInfo     `json:"section"`
	Only    []string        `json:"only"`
	Regions []*model.Region `json:"regions"`
	Spans   []model.Span    `json:"spans"`
}

//---------------------------------------------------------------------------
// 构造
//---------------------------------------------------------------------------

// buildManifest 由 doc / pkg / 条目清单构造 manifest。
func buildManifest(doc *model.Document, pkg *pkgfile.Package, prog string, idx []EntryRef) *Manifest {
	module, erpver, _ := tapRootAttrs(pkg)
	m := &Manifest{
		Tool:        ToolName,
		ToolVersion: ToolVersion,
		Prog:        prog,
		Module:      module,
		ErpVer:      erpver,
		Env:         doc.Env.Env,
		Kind:        pkg.Kind.String(),
		Pkg: PkgRef{
			Path:   pkgPath(doc, pkg),
			Sha256: pkgSha256(doc, pkg),
			Ver:    pkgVer(doc, pkg),
		},
		Export:  ExportMode{Only: onlyOrEmpty(doc.Only)},
		Section: buildSectionInfo(doc),
		Anchors: anchorsOrEmpty(doc.Anchors),
		Entries: nonNilEntries(idx),
	}
	refreshManifest(m, doc)
	return m
}

// buildSectionInfo 由 doc 的状态 + TAP 根事实构造 manifest.section。
func buildSectionInfo(doc *model.Document) SectionInfo {
	state := doc.SectionState
	if state == "" {
		state = model.EffectiveSectionState(doc.Env.SectionFlag, doc.PendingUnlock)
	}
	sf := "N"
	if doc.Env.SectionFlag {
		sf = "Y"
	}
	ssv := "N"
	if doc.Env.StdSectionVerify {
		ssv = "Y"
	}
	by := doc.UnlockedBy
	if by == "" {
		by = model.UnlockedByOf(doc.Env.SectionFlag, doc.PendingUnlock)
	}
	return SectionInfo{
		State:            string(state),
		SectionFlag:      sf,
		StdSectionVerify: ssv,
		UnlockedBy:       by,
		PendingUnlock:    doc.PendingUnlock,
	}
}

// refreshManifest 只刷新 regions / spans / stats（apply 后重算），其余字段原样保留。
func refreshManifest(m *Manifest, doc *model.Document) {
	m.Regions = regionSummaries(doc)
	m.Spans = spanSummaries(doc)
	m.Stats = buildStats(doc)
	m.Section = buildSectionInfo(doc)
}

// buildRegionsFile 由 doc / 基线构造 regions.json。
//
// old 非 nil 时（apply 后刷新）保留导出范围 only：导出范围在 export 时就固定了，
// apply 不改变它。env / section / regions / spans 一律以新 doc 为准。
func buildRegionsFile(doc *model.Document, prog string, fenced []byte, old *RegionsFile) *RegionsFile {
	env := doc.Env
	only := doc.Only
	if old != nil {
		if only == nil {
			only = old.Only
		}
		// 调用方若没填 Env（零值），沿用工作区里已记录的上下文。
		if env == (model.EnvContext{}) {
			env = old.Env
		}
	}
	return &RegionsFile{
		Prog:       prog,
		BaseSha256: model.Sha256Bytes(fenced),
		TextLen:    len(fenced),
		Env:        env,
		Section:    buildSectionInfo(doc),
		Only:       onlyOrEmpty(only),
		Regions:    regionsOrEmpty(doc.Regions),
		Spans:      spansOrEmpty(doc.Spans),
	}
}

func regionSummaries(doc *model.Document) []RegionSummary {
	all := doc.AllRegions()
	out := make([]RegionSummary, 0, len(all))
	for _, r := range all {
		out = append(out, regionSummary(r))
	}
	return out
}

// regionSummary 把 model.Region 摊平成便于阅读的摘要。
func regionSummary(r *model.Region) RegionSummary {
	s := RegionSummary{
		Name:       r.Name,
		Kind:       string(r.Kind),
		Editable:   r.Editable,
		DenyCode:   r.DenyCode,
		Reason:     r.Reason,
		Status:     metaOf(r, "status"),
		Src:        metaOf(r, "src"),
		New:        metaOf(r, "new"),
		Order:      metaOf(r, "order"),
		CiteStd:    metaOf(r, "cite_std"),
		Edit:       metaOf(r, "edit"),
		Mark:       metaOf(r, "mark"),
		Origin:     r.Origin,
		AnchorType: r.AnchorType,
		Append:     r.Append,
		Sha256:     r.Sha256,
	}
	if s.Reason == "" {
		s.Reason = model.DenyReasonText(s.DenyCode)
	}
	return s
}

func spanSummaries(doc *model.Document) []SpanSummary {
	out := make([]SpanSummary, 0, len(doc.Spans))
	for _, s := range doc.Spans {
		out = append(out, SpanSummary{Name: s.Name, Start: s.Span.Start, End: s.Span.End, Sha256: s.Sha256})
	}
	return out
}

// buildStats 统计**全部**区间（含嵌套子区间，走 doc.AllRegions()）。
func buildStats(doc *model.Document) Stats {
	var s Stats
	for _, r := range doc.AllRegions() {
		switch r.Kind {
		case model.RegionPoint:
			s.Points++
		case model.RegionSection:
			s.Sections++
		}
		if r.Editable {
			s.Editable++
		} else {
			s.Readonly++
		}
	}
	return s
}

// metaOf 读 Region.Meta；缺失键返回空串。
func metaOf(r *model.Region, key string) string {
	if r.Meta == nil {
		return ""
	}
	return r.Meta[key]
}

// effectiveSectionState 合并 doc 上的状态载体（Document.SectionState / EnvContext.SectionState /
// pending_unlock / 包级 section_flag），得到唯一状态。v1 的 allow_sec 已由 unlock 状态机取代。
func effectiveSectionState(doc *model.Document) model.SectionState {
	if doc == nil {
		return model.SectionLocked
	}
	if doc.SectionState != "" {
		return doc.SectionState
	}
	if doc.Env.SectionState != "" {
		return doc.Env.SectionState
	}
	return model.EffectiveSectionState(doc.Env.SectionFlag, doc.PendingUnlock)
}

func pkgPath(doc *model.Document, pkg *pkgfile.Package) string {
	if doc.PkgPath != "" {
		return doc.PkgPath
	}
	return pkg.Path
}

func pkgSha256(doc *model.Document, pkg *pkgfile.Package) string {
	if doc.PkgSha256 != "" {
		return doc.PkgSha256
	}
	if pkg.Path == "" {
		return ""
	}
	sum, err := model.Sha256File(pkg.Path)
	if err != nil {
		return ""
	}
	return sum
}

func pkgVer(doc *model.Document, pkg *pkgfile.Package) string {
	if doc.Ver != "" {
		return doc.Ver
	}
	return pkg.Ver.Raw
}

func onlyOrEmpty(only []string) []string {
	if only == nil {
		return []string{}
	}
	return only
}

func anchorsOrEmpty(a map[string]bool) map[string]bool {
	if a == nil {
		return map[string]bool{}
	}
	return a
}

func nonNilEntries(idx []EntryRef) []EntryRef {
	if idx == nil {
		return []EntryRef{}
	}
	return idx
}

func regionsOrEmpty(rs []*model.Region) []*model.Region {
	if rs == nil {
		return []*model.Region{}
	}
	return rs
}

func spansOrEmpty(ss []model.Span) []model.Span {
	if ss == nil {
		return []model.Span{}
	}
	return ss
}

//---------------------------------------------------------------------------
// .tap 根属性（best-effort）
//---------------------------------------------------------------------------

// tapRootAttrs 从 .tap 条目里读根元素的 module / erpver / prog 属性。
//
// model.Document 不带 module / erpver，而 manifest 想让 AI 直接看到它们，
// 因此这里做一次只读的 best-effort 提取（encoding/xml 宽松模式，只取第一个起始标签）。
// 解析失败 / 没有 .tap → 返回空串，**绝不让 export 失败**。
func tapRootAttrs(pkg *pkgfile.Package) (module, erpver, prog string) {
	if pkg == nil {
		return "", "", ""
	}
	e := pkg.Tap()
	if e == nil || len(e.Data) == 0 {
		return "", "", ""
	}
	dec := xml.NewDecoder(bytes.NewReader(e.Data))
	dec.Strict = false
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", "", ""
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		for _, a := range se.Attr {
			switch a.Name.Local {
			case "module":
				module = strings.TrimSpace(a.Value)
			case "erpver":
				erpver = strings.TrimSpace(a.Value)
			case "prog":
				prog = strings.TrimSpace(a.Value)
			}
		}
		return module, erpver, prog
	}
}

// progFromPath 仅用于兜底（包文件名去掉扩展名）。
func progFromPath(p string) string {
	b := filepath.Base(p)
	return strings.TrimSuffix(b, filepath.Ext(b))
}
