// Package verify 实现设计指南 §5.5 的三道闸门。
//
//	gate1：字节恒等 + 结构行未改 + 写入授权（不可编辑字节一个都不能动）
//	gate2：README §3.8 的不变量 I1–I15 全集
//	gate3：装载模拟 —— 对**将要产出的新包**重跑 synthesize，等价于「设计器打得开吗」
//
// 退出码约定（设计指南 §2）：
//
//	gate1/gate2/gate3 的 error 级发现 → 3
//	权限类（改 READONLY 区、未开 --allow-sec 改区段、--only 范围外改）→ 4
package verify

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"tt/internal/dev/fence"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/synth"
	"tt/internal/dev/tapfile"
)

// ExitCodeError 让 CLI 把错误映射为退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// DeniedError = 退出码 4（写入被拒：权限不足）。
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

// Severity 是发现等级。
type Severity string

const (
	SevInfo  Severity = "info"
	SevWarn  Severity = "warn"
	SevError Severity = "error"
)

// Finding 是一条验证发现。
type Finding struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Region   string   `json:"region,omitempty"`
	Detail   []string `json:"detail,omitempty"`
	// Denied 为真表示该发现属于「写入被拒」（退出码 4）而不是「验证失败」（退出码 3）。
	Denied bool `json:"denied,omitempty"`

	// 位置：出错的文件与 1-based 行号（Line==0 表示无位置可言，例如包级发现），
	// Snippet 是该行内容（已 clip）。有了它，apply/status 的报错能直接指到
	// `prog.full.4gl` 的第 N 行，而不是只说「某个 Region 被改了」。
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Snippet string `json:"snippet,omitempty"`
}

// 位置出处的两个文件名（工作区里固定的两份可编辑面）。
const (
	FileEdited = "prog.full.4gl"       // 用户编辑的渲染文档
	FileBase   = ".tdev/base.full.4gl" // gate1 的比对基线；删除类问题只能用它的行号
)

// locate 用字节偏移给 Finding 补位置（返回新值，便于链式写）。
func locate(f Finding, d *model.Document, file string, off int) Finding {
	if d == nil || off < 0 || len(d.Text) == 0 {
		return f
	}
	line, txt := model.LineAt(d.Text, off)
	if line <= 0 {
		return f
	}
	f.File, f.Line, f.Snippet = file, line, clip(txt)
	return f
}

// AddAt 是「带位置的 Add」：off 是 doc 里的字节偏移（<0 表示没有位置）。
func (r *Report) AddAt(f Finding, d *model.Document, file string, off int) {
	r.Add(locate(f, d, file, off))
}

// docDiffOff 返回两篇文档第一处差异在**编辑后文档**里的偏移；完全相同返回 -1。
func docDiffOff(base, edited *model.Document) int {
	if base == nil || edited == nil {
		return -1
	}
	i := model.FirstDiffEOL(base.Text, edited.Text)
	if i < 0 || i > len(edited.Text) {
		return -1
	}
	return i
}

// contentDiffOff 返回某个 Region 内容里第一处差异在编辑后文档里的偏移。
// 「只读区被改动」这类发现靠它指到具体那一行，而不是只报区段名。
func contentDiffOff(base, edited *model.Document, b, e *model.Region) int {
	if base == nil || edited == nil || b == nil || e == nil {
		return -1
	}
	bs, es := b.ContentSpan, e.ContentSpan
	if bs.Start < 0 || bs.End > len(base.Text) || es.Start < 0 || es.End > len(edited.Text) {
		return -1
	}
	i := model.FirstDiffEOL(base.Text[bs.Start:bs.End], edited.Text[es.Start:es.End])
	if i < 0 {
		return -1
	}
	off := es.Start + i
	if off > len(edited.Text) {
		off = len(edited.Text)
	}
	return off
}

// spanStartOf 取区间起点（nil → -1）。
func spanStartOf(sp *model.ByteRange) int {
	if sp == nil {
		return -1
	}
	return sp.Start
}

// regionOffOf 在文档里按名字找 Region 的内容起点（TAP 侧的问题用它换行号）。
func regionOffOf(d *model.Document, name string) int {
	if d == nil || name == "" {
		return -1
	}
	if r := d.FindRegion(name); r != nil {
		return r.ContentSpan.Start
	}
	return -1
}

// tapLoc 把 TAP 里的字节偏移说清落在哪个 point/section 上（TAP 层报错用）。
func tapLoc(d *tapfile.Doc, off int) string {
	if d == nil || off < 0 || off > len(d.Raw) {
		return fmt.Sprintf("TAP 字节偏移 %d", off)
	}
	if p := d.PointAt(off); p != nil {
		n, _ := p.Attr("name")
		return fmt.Sprintf("TAP 字节偏移 %d，位于 <point name=%q>", off, n)
	}
	if s := d.SectionAt(off); s != nil {
		id, _ := s.Attr("id")
		return fmt.Sprintf("TAP 字节偏移 %d，位于 <section id=%q>", off, id)
	}
	return fmt.Sprintf("TAP 字节偏移 %d（不在任何 point/section 内）", off)
}

// regionAnchorOff 取「这个 Region 最值得指的那一行」：优先签名行（函数头），
// 其次描述块，最后正文起点。结构事务类问题都落在函数头上，指签名行最有用。
func regionAnchorOff(r *model.Region) int {
	if r == nil {
		return -1
	}
	if r.SignatureSpan != nil {
		return r.SignatureSpan.Start
	}
	if r.DescSpan != nil {
		return r.DescSpan.Start
	}
	return r.ContentSpan.Start
}

// fillPositions 是位置信息的**兜底**：凡是报了 Region 名却还没有行号的发现，
// 就到编辑后文档（再退到基线）里按名字定位。
//
// 这样 V1–V7、I1–I5 这类「按区段报」的发现不必在每处都手写偏移，
// 也能拿到 `prog.full.4gl:1234` 这种可点击的定位；删除类发现已在基线侧定位，不会被动到。
func fillPositions(rep *Report, base, doc *model.Document) {
	for i := range rep.Findings {
		f := &rep.Findings[i]
		if f.Line > 0 || f.Region == "" {
			continue
		}
		type side struct {
			doc  *model.Document
			file string
		}
		for _, s := range []side{{doc, FileEdited}, {base, FileBase}} {
			if s.doc == nil {
				continue
			}
			r := s.doc.FindRegion(f.Region)
			if r == nil {
				continue
			}
			if nf := locate(*f, s.doc, s.file, regionAnchorOff(r)); nf.Line > 0 {
				*f = nf
				break
			}
		}
	}
}

// Report 是一次验证的完整结果。
type Report struct {
	Findings []Finding `json:"findings"`
	Errors   int       `json:"errors"`
	Warns    int       `json:"warns"`
	Infos    int       `json:"infos"`
	Denied   int       `json:"denied_count"`
}

// Add 追加一条发现。
func (r *Report) Add(f Finding) {
	r.Findings = append(r.Findings, f)
	switch f.Severity {
	case SevError:
		r.Errors++
	case SevWarn:
		r.Warns++
	default:
		r.Infos++
	}
	if f.Denied && f.Severity == SevError {
		r.Denied++
	}
}

// HasError 判断是否有 error 级发现。
func (r *Report) HasError() bool { return r.Errors > 0 }

// HasDenied 判断是否存在「写入被拒」类 error。
func (r *Report) HasDenied() bool { return r.Denied > 0 }

// Summary 返回一行人类可读摘要。
func (r *Report) Summary() string {
	return fmt.Sprintf("error=%d warn=%d info=%d", r.Errors, r.Warns, r.Infos)
}

//----------------------------------------------------------------------------
// gate1：字节恒等 + 授权
//----------------------------------------------------------------------------

func contentOf(d *model.Document, r *model.Region) []byte {
	if r == nil || r.ContentSpan.End > len(d.Text) || r.ContentSpan.Start > r.ContentSpan.End {
		return nil
	}
	return d.Text[r.ContentSpan.Start:r.ContentSpan.End]
}

func clip(s string) string {
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}

func firstDiffDesc(a, b []byte) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			lo := i - 20
			if lo < 0 {
				lo = 0
			}
			return fmt.Sprintf("偏移 %d，原 %q 新 %q", i, clip(string(a[lo:i+20])), clip(string(b[lo:i+20])))
		}
	}
	return fmt.Sprintf("长度不同：原 %d 新 %d", len(a), len(b))
}

// outsideBytes 返回围栏外字节的拼接（prefix / gap / suffix，按文档顺序）。
func outsideBytes(d *model.Document) []byte {
	var out bytes.Buffer
	for _, s := range d.Spans {
		if s.Span.End > s.Span.Start && s.Span.End <= len(d.Text) {
			out.Write(d.Text[s.Span.Start:s.Span.End])
		}
	}
	return out.Bytes()
}

// Gate1 对比基线（导出时的围栏文档）与编辑后的文档。
//
// 判定面 = 全文 − ∪可写字节区间。可写区间只包含：
//   - 自订定义点的**正文本体**（函数头/注释头/END 行都在其外）
//   - 裸名插入点的整块内容
//   - --allow-sec 下可写区段的正文
//
// 因此围栏行、未归属段、只读区内容、结构行全部被物理保护。
func Gate1(base *model.Document, parsed *fence.ParseResult) *Report {
	rep := &Report{}
	edited := parsed.Doc

	// sameBytes 做「行尾等价」的字节比对：受保护字节 tdev 从不写回包，
	// 所以编辑器（VS Code 等）把 CRLF 归一成 LF 不算改动；任何其它字节差异照旧拦下。
	eolOnly := 0
	sameBytes := func(a, b []byte) bool {
		if bytes.Equal(a, b) {
			return true
		}
		if model.EqualEOL(a, b) {
			eolOnly++
			return true
		}
		return false
	}

	// 1) 区段序列与数量必须与基线一致（fence.Parse 已保证配对与顺序；
	//    这里核对数量：基线总数 = 有基线配对的 Region + 被删除的 Region；
	//    新增点（Base == nil）不计入基线）
	matched := 0
	for _, pr := range parsed.Pairs {
		if pr.Base != nil {
			matched++
		}
	}
	if matched+len(parsed.Deleted) != len(base.AllRegions()) {
		rep.AddAt(Finding{Code: "gate1.region-count", Severity: SevError,
			Message: fmt.Sprintf("Region 数量不守恒：基线 %d，配对 %d + 删除 %d",
				len(base.AllRegions()), matched, len(parsed.Deleted))},
			edited, FileEdited, docDiffOff(base, edited))
	}

	// 2) 被整块删除的 Region：只有 new="Y" 的点可删（设计指南 §4 规则 5）
	//    删除类问题在编辑后的文档里已经不存在，行号只能给基线（FileBase）。
	for _, d := range parsed.Deleted {
		off := d.ContentSpan.Start
		if d.Kind != model.RegionPoint {
			rep.AddAt(Finding{Code: "gate1.delete-section", Severity: SevError, Region: d.Name,
				Message: "区段不可删除（只能改正文）"}, base, FileBase, off)
			continue
		}
		if !strings.EqualFold(d.Meta["new"], "Y") {
			rep.AddAt(Finding{Code: "gate1.delete-not-new", Severity: SevError, Denied: true, Region: d.Name,
				Message: "只有 new=\"Y\" 的自订点才能删除（与设计器 CodeEditorMainWindow.CanDeleteFunction 一致）",
				Detail:  []string{"该点 new=" + d.Meta["new"]}}, base, FileBase, off)
			continue
		}
		if !d.Editable {
			rep.AddAt(Finding{Code: "gate1.delete-readonly", Severity: SevError, Denied: true, Region: d.Name,
				Message: "删除被拒：该点不可编辑（deny=" + d.DenyCode + "）",
				Detail:  []string{d.Reason}}, base, FileBase, off)
			continue
		}
		rep.AddAt(Finding{Code: "gate1.delete-ok", Severity: SevInfo, Region: d.Name,
			Message: "删除自订点（new=Y，已授权）"}, base, FileBase, off)
	}

	// 3) 配对的 Region：逐项对比「不可写字节」（用配对而不是按名字查表 —— 真实包里
	//    存在区段 id 与点名同名的情况）
	for _, pr := range parsed.Pairs {
		b, e := pr.Base, pr.Edited
		if b == nil || e == nil {
			continue
		}
		if b.Kind != e.Kind {
			rep.AddAt(Finding{Code: "gate1.kind", Severity: SevError, Region: b.Name,
				Message: "Region 类型变了"}, edited, FileEdited, e.BeginFence.Start)
			continue
		}
		// 3-pre) 围栏行的**锚定部分**不许增删改（设计指南 §4 规则 2）。
		//
		// v2 起不再逐字节比整行：`fn`/`scope`/`desc` 是唯一允许改的字段，
		// 它们由结构事务校验（V1/V5/V6）治理；其余（kind/name/旗标/status/src/new/…）
		// 用结构化签名比对 —— 既不误伤字段顺序/空白/EOL，又抓得住任何真实改动。
		baseSig := fence.AnchoredSignature(b.Kind, b.Name, pr.BaseMeta)
		editSig := fence.AnchoredSignature(e.Kind, e.Name, pr.EditedMeta)
		if baseSig != editSig {
			rep.AddAt(Finding{Code: "gate1.fence-line", Severity: SevError, Region: b.Name,
				Message: "围栏行的锚定部分被改动（point/section 名、旗标、status/src/new 等不许改；只有 fn/scope/desc 可改）",
				Detail:  []string{"原：" + clip(baseSig), "新：" + clip(editSig)}},
				edited, FileEdited, e.BeginFence.Start)
		}
		if b.EndFence.End > b.EndFence.Start && e.EndFence.End > e.EndFence.Start {
			be := base.Text[b.EndFence.Start:b.EndFence.End]
			ee := edited.Text[e.EndFence.Start:e.EndFence.End]
			if !sameBytes(be, ee) {
				rep.AddAt(Finding{Code: "gate1.fence-line", Severity: SevError, Region: b.Name,
					Message: "end 围栏行被改动",
					Detail:  []string{"原：" + clip(string(be)), "新：" + clip(string(ee))}},
					edited, FileEdited, e.EndFence.Start)
			}
		}
		// 3a) 不可编辑区：**自身字节**必须逐字节相等。
		//
		// 「自身字节」= 区段内容剔除所有子区间（含其行尾换行）后的字节。
		// 子区间有各自的可编辑性与授权，不能因为子区间合法改动而把父区段判成被改；
		// 反过来，父区段自己的任何字节改动都跑不掉。
		if !b.Editable {
			if !sameBytes(base.OwnBytes(b), edited.OwnBytes(e)) {
				off := contentDiffOff(base, edited, b, e)
				if b.Append {
					rep.AddAt(Finding{Code: "gate1.append-anchor-body", Severity: SevError, Denied: true, Region: b.Name,
						Message: "锚点区段只能追加完整的新点块，不能改区段自身字节",
						Detail:  []string{"deny=" + b.DenyCode}}, edited, FileEdited, off)
				} else {
					detail := []string{"deny=" + b.DenyCode, b.Reason}
					if b.DenyCode == model.DenySecLocked {
						detail = append(detail, "先执行 `tt dev tzc unlock <dir>`（解锁是一次单向、有代价的状态迁移）")
					}
					rep.AddAt(Finding{Code: "gate1.readonly-region", Severity: SevError, Denied: true, Region: b.Name,
						Message: "只读 Region 内容被改动（写入被拒）",
						Detail:  detail}, edited, FileEdited, off)
				}
			}
			continue
		}
		// 3b) 可编辑区：结构行必须逐字节相等
		if len(b.StructureSpans) != len(e.StructureSpans) {
			rep.AddAt(Finding{Code: "I-structure", Severity: SevError, Region: b.Name,
				Message: "结构行数量变了（函数头 / END 行缺失或重复）"}, edited, FileEdited, e.ContentSpan.Start)
			continue
		}
		for i := range b.StructureSpans {
			bs := base.Text[b.StructureSpans[i].Start:b.StructureSpans[i].End]
			es := edited.Text[e.StructureSpans[i].Start:e.StructureSpans[i].End]
			if !sameBytes(bs, es) {
				rep.AddAt(Finding{Code: "I-structure", Severity: SevError, Region: b.Name,
					Message: "结构行被改动（设计器只允许改正文本体，对齐 CanInsert 行为）",
					Detail: []string{
						"结构行 " + fmt.Sprint(i+1) + " 原：" + clip(string(bs)),
						"结构行 " + fmt.Sprint(i+1) + " 新：" + clip(string(es)),
					}}, edited, FileEdited, e.StructureSpans[i].Start)
			}
		}
	}

	// 4) 围栏外字节（prefix / gap / suffix）：整段拼接后逐字节相等。
	//
	// 用**拼接**而不是逐段对拍：区间位移或空段增删（Region 内容变长变短）会让
	// gap.N 的编号错位，逐段比会误报；而拼接序列对这类位移免疫，
	// 同时对「在 Region 之外增删/改动任何字节」依然敏感。
	baseOut := outsideBytes(base)
	editOut := outsideBytes(edited)
	if !sameBytes(baseOut, editOut) {
		rep.AddAt(Finding{Code: "gate1.outside-fence", Severity: SevError,
			Message: "围栏外字节被改动（Region 之外不许增删改任何内容）",
			Detail: []string{
				fmt.Sprintf("原 %d 字节，新 %d 字节", len(baseOut), len(editOut)),
				firstDiffDesc(baseOut, editOut),
			}}, edited, FileEdited, docDiffOff(base, edited))
	}

	// 5) 新增点：必须在 APPEND 锚点内（parse 已强制），且 new=Y
	for _, a := range parsed.Appended {
		if !strings.EqualFold(a.Meta["new"], "Y") {
			rep.AddAt(Finding{Code: "gate1.append-not-new", Severity: SevError, Denied: true, Region: a.Name,
				Message: "新增点必须 new=\"Y\"", Detail: []string{"实际 new=" + a.Meta["new"]}},
				edited, FileEdited, a.ContentSpan.Start)
			continue
		}
		if a.BodySpan == nil {
			rep.AddAt(Finding{Code: "I4", Severity: SevWarn, Region: a.Name,
				Message: "新增点正文不是可解析的函数块，apply 会拒绝写回"},
				edited, FileEdited, a.ContentSpan.Start)
			continue
		}
		rep.AddAt(Finding{Code: "gate1.append-ok", Severity: SevInfo, Region: a.Name,
			Message: "在 APPEND 锚点内新增自订点"}, edited, FileEdited, a.ContentSpan.Start)
	}
	if eolOnly > 0 {
		rep.Add(Finding{Code: "gate1.eol-normalized", Severity: SevInfo,
			Message: "受保护字节存在行尾差异（CRLF↔LF），已按等价处理：这些字节不会写回包",
			Detail:  []string{fmt.Sprintf("%d 处比较仅行尾不同", eolOnly)}})
	}
	fillPositions(rep, base, edited) // 兜底：凡是只报了 Region 名的发现补上行号
	return rep
}

//----------------------------------------------------------------------------
// gate2：不变量 I1–I15
//----------------------------------------------------------------------------

// Gate2 检查不变量。tapDoc 是**编辑后将要写回**的 TAP（apply 时是新 TAP；
// verify 时是原 TAP）。doc 是编辑后的文档。
func Gate2(base *model.Document, parsed *fence.ParseResult, tapDoc *tapfile.Doc) *Report {
	rep := &Report{}
	doc := parsed.Doc

	// I15：TAP 条目基名 ≠ 根 prog（引用标准程序的包，设计器 Packing 走 CiteTAP）
	// 由调用方用 pkgfile 提供，这里检查基名一致性。

	// I1：区段配对 + id 集合一致
	tapSections := map[string]bool{}
	for _, s := range tapDoc.Sections {
		id, _ := s.Attr("id")
		tapSections[id] = true
	}
	seen := map[string]int{}
	for _, r := range doc.AllRegions() {
		if r.Kind != model.RegionSection {
			continue
		}
		seen[r.Name]++
		if seen[r.Name] > 1 {
			rep.AddAt(Finding{Code: "I1", Severity: SevError, Region: r.Name,
				Message: "区段 id 在文档里重复"}, doc, FileEdited, r.ContentSpan.Start)
		}
		if !tapSections[r.Name] {
			// 设计器会从 src=c/src=m 的区段 clone；clone 出来的 status="u" 是允许的
			if r.DenyCode == model.DenySectionTemplateMissing {
				rep.AddAt(Finding{Code: "I1", Severity: SevError, Region: r.Name,
					Message: "TGL 有区段但 TAP 没有，且无模板可 clone（设计器会抛「找不到区段」）"},
					doc, FileEdited, r.ContentSpan.Start)
			} else {
				rep.AddAt(Finding{Code: "I1", Severity: SevWarn, Region: r.Name,
					Message: "TAP 缺该区段（设计器会 clone 一个 src=c 的模板）"},
					doc, FileEdited, r.ContentSpan.Start)
			}
		}
	}
	for id := range tapSections {
		if seen[id] == 0 {
			rep.Add(Finding{Code: "I1", Severity: SevWarn, Region: id,
				Message: "TAP 有区段但 TGL 里没有对应标记（该区段正文不会显示）"})
		}
	}

	// I2b/I2c：命名空间三层的错配
	anchors := base.Anchors
	liveByKind := map[string]int{}
	bareNoPlaceholder := 0
	for _, p := range tapDoc.Points {
		if tapfile.IsDeleted(p) {
			continue
		}
		n, _ := p.Attr("name")
		switch {
		case strings.HasPrefix(n, "function."):
			liveByKind["function"]++
		case strings.HasPrefix(n, "dialog."):
			liveByKind["dialog"]++
		case strings.HasPrefix(n, "report."):
			liveByKind["report"]++
		default:
			// 裸名插入点：TGL 里通常有同名占位符；没有则是惰性数据（孤儿点）
			if base.FindRegion(n) == nil {
				bareNoPlaceholder++
			}
		}
	}
	for k, c := range liveByKind {
		if c > 0 && !anchors["other."+k] {
			rep.Add(Finding{Code: "I2b", Severity: SevError,
				Message: "TAP 有 " + k + ". 自订定义点，但 TGL 没有 other." + k + " 锚点 —— 这些代码永远不会出现在文档里",
				Detail:  []string{fmt.Sprintf("%d 个点", c)}})
		}
	}
	if bareNoPlaceholder > 0 {
		rep.Add(Finding{Code: "I2c", Severity: SevWarn,
			Message: "TAP 裸插入点在 TGL 里没有同名占位符（惰性数据：看不到但不会丢，必须原样透传）",
			Detail:  []string{fmt.Sprintf("%d 个点", bareNoPlaceholder)}})
	}
	// I2a：TGL 占位符在 TAP 里没有对应点 —— 正常（设计器造空点）
	emptyCreated := 0
	for _, r := range doc.AllRegions() {
		if r.Kind == model.RegionPoint && r.Origin == "placeholder-empty" {
			emptyCreated++
		}
	}
	if emptyCreated > 0 {
		rep.Add(Finding{Code: "I2a", Severity: SevInfo,
			Message: "TGL 有占位符而 TAP 无同名点：设计器会造空点（常态，不是错误）",
			Detail:  []string{fmt.Sprintf("%d 处", emptyCreated)}})
	}

	// I3：区段正文不得出现真实自订点的完整函数块（精确判据：拿已知函数名匹配）
	var names []string
	for _, p := range tapDoc.Points {
		n, _ := p.Attr("name")
		if strings.HasPrefix(n, "function.") || strings.HasPrefix(n, "dialog.") || strings.HasPrefix(n, "report.") {
			names = append(names, strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(n, "function."), "dialog."), "report."))
		}
	}
	reI3 := buildI3(names)
	if reI3 != nil {
		for _, r := range doc.AllRegions() {
			if r.Kind != model.RegionSection {
				continue
			}
			c := doc.OwnBytes(r)
			if m := reI3.FindSubmatch(c); m != nil {
				rep.AddAt(Finding{Code: "I3", Severity: SevError, Region: r.Name,
					Message: "区段正文里出现了展开后的自订函数块（TglTag 折叠被破坏，下次装载会重复展开 —— 红线 R4）",
					Detail:  []string{"命中：" + clip(string(m[0]))}}, doc, FileEdited, r.ContentSpan.Start)
			}
		}
	}

	// I4：自订点结构（结构行无法解析 → warn，设计器装载会抛异常）
	for _, r := range doc.AllRegions() {
		if r.Kind == model.RegionPoint && r.DenyCode == model.DenyStructureUnparsable {
			rep.AddAt(Finding{Code: "I4", Severity: SevWarn, Region: r.Name,
				Message: "正文不是可解析的函数块信封（设计器装载时会抛 FormatException，已知 :function.* 前缀不保证是函数块）",
				Detail:  []string{"tdev 已将该点标为不可写回"}}, doc, FileEdited, r.ContentSpan.Start)
		}
	}

	// §4.1 结构事务校验（V1–V7）：gate2 的一部分（设计指南 §5.5）。
	// detect 判据 = 围栏 fn/scope/desc 或签名行相对基线变化。
	if edits := DetectStructural(base, parsed); len(edits) > 0 {
		sub := validateStructural(edits, base, parsed, tapDoc, base.Prog)
		rep.Findings = append(rep.Findings, sub.Findings...)
		rep.Errors += sub.Errors
		rep.Warns += sub.Warns
		rep.Infos += sub.Infos
		rep.Denied += sub.Denied
		rep.Add(Finding{Code: "structtx", Severity: SevInfo,
			Message: fmt.Sprintf("检测到 %d 项结构事务（改名/scope/描述/签名）", len(edits))})
	}

	// I5：status 取值 + 删点保留内容
	validStatus := map[string]bool{"": true, " ": true, "c": true, "u": true, "d": true}
	for _, p := range tapDoc.Points {
		v, ok := p.Attr("status")
		n, _ := p.Attr("name")
		off := regionOffOf(doc, n) // 同名 Region 存在时就能给出行号
		if !ok {
			rep.AddAt(Finding{Code: "I5", Severity: SevError, Region: n, Message: "point 缺 status 属性"},
				doc, FileEdited, off)
			continue
		}
		if !validStatus[v] {
			rep.AddAt(Finding{Code: "I5", Severity: SevError, Region: n,
				Message: "status 取值非法", Detail: []string{"status=" + fmt.Sprint(v)}}, doc, FileEdited, off)
		}
		if v == "d" && !p.HasCDATA() {
			rep.AddAt(Finding{Code: "I5", Severity: SevWarn, Region: n,
				Message: "标记删除的点没有保留原内容（设计器会保留）"}, doc, FileEdited, off)
		}
	}

	// I11：UTF-8 无 BOM；CDATA 内不得残留 0x07；新内容不得含 ]]>
	for _, e := range []struct {
		name string
		b    []byte
	}{
		{"tap", tapDoc.Raw},
	} {
		if len(e.b) >= 3 && e.b[0] == 0xEF && e.b[1] == 0xBB && e.b[2] == 0xBF {
			rep.Add(Finding{Code: "I11", Severity: SevError, Message: "TAP 带 UTF-8 BOM"})
		}
		if bytes.IndexByte(e.b, 0x07) >= 0 {
			off := bytes.IndexByte(e.b, 0x07)
			rep.Add(Finding{Code: "I11", Severity: SevError, Message: "TAP 里残留 0x07(\\a) diff 标记",
				Detail: []string{tapLoc(tapDoc, off)}})
		}
	}
	// I11b：可编辑内容里出现 ]]> —— XML CDATA 装不下（设计器会拆成两段 CDATA），必须拦在写之前
	for _, r := range doc.AllRegions() {
		if !r.Editable {
			continue
		}
		c := contentOf(doc, r)
		if bytes.Contains(c, []byte("]]>")) {
			rep.AddAt(Finding{Code: "I11b", Severity: SevError, Region: r.Name,
				Message: "内容里出现 ]]>，无法作为 CDATA 写入（请改成字符串拼接或转义，例如 ']' + ']>'）",
				Detail:  []string{"XML 不允许 CDATA 内含 ]]>"}}, doc, FileEdited, r.ContentSpan.Start)
		}
	}
	for _, a := range parsed.Appended {
		if bytes.Contains(contentOf(doc, a), []byte("]]>")) {
			rep.AddAt(Finding{Code: "I11b", Severity: SevError, Region: a.Name,
				Message: "新增点内容里出现 ]]>，无法作为 CDATA 写入"}, doc, FileEdited, a.ContentSpan.Start)
		}
	}

	// I10：.4gl 与合成结果不一致是**正常**（82/105 实测）→ info
	rep.Add(Finding{Code: "I10", Severity: SevInfo,
		Message: ".4gl 是服务器 build 产物，与「TGL + 展开点」不一致是常态，不是 bug（红线 R1；实测 82/105）"})
	fillPositions(rep, base, doc) // 兜底：V1–V7 / I1–I5 等按区段报的发现补上行号
	return rep
}

func buildI3(names []string) *regexp.Regexp {
	var parts []string
	for _, n := range names {
		if n == "" {
			continue
		}
		parts = append(parts, regexp.QuoteMeta(n))
	}
	if len(parts) == 0 {
		return nil
	}
	// 精确判据：先剥 public/private，再匹配定义头（README §3.8 I3 的教训：
	// 宽正则会误伤框架自己的 DIALOG ATTRIBUTES(...)）
	pat := `(?im)^[ \t]*(?:(?:public|private)[ \t]+)?(?:FUNCTION|DIALOG|REPORT)[ \t]+(?:` +
		strings.Join(parts, "|") + `)[ \t]*\(`
	return regexp.MustCompile(pat)
}

//----------------------------------------------------------------------------
// gate3：装载模拟
//----------------------------------------------------------------------------

// Gate3 对**将要产出的新包字节**重跑合成，等价于问「设计器打得开吗」。
//
// 它不写用户的原包：只写到临时文件再读回。
func Gate3(pkg *pkgfile.Package, newTap, newTgl []byte) (*Report, error) {
	rep := &Report{}
	newBytes, _, err := pkg.Build(pkgfile.Rebuild{Tap: newTap, Tgl: newTgl})
	if err != nil {
		rep.Add(Finding{Code: "gate3.build", Severity: SevError, Message: "新包构建失败", Detail: []string{err.Error()}})
		return rep, nil
	}
	dir, err := os.MkdirTemp("", "tdev-gate3-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	tmp := filepath.Join(dir, filepath.Base(pkg.Path))
	if err := os.WriteFile(tmp, newBytes, 0o644); err != nil {
		return nil, err
	}
	newPkg, err := pkgfile.Open(tmp, pkgfile.OpenOptions{})
	if err != nil {
		rep.Add(Finding{Code: "gate3.open", Severity: SevError,
			Message: "新包无法被重新打开（设计器也打不开）", Detail: []string{err.Error()}})
		return rep, nil
	}

	// 逐条目核对：.4gl / ver / 未知条目必须字节级不变；条目集合与顺序不变
	if len(newPkg.Entries) != len(pkg.Entries) {
		rep.Add(Finding{Code: "I13", Severity: SevError,
			Message: fmt.Sprintf("条目数变了：%d → %d", len(pkg.Entries), len(newPkg.Entries))})
	}
	for i := range pkg.Entries {
		if i >= len(newPkg.Entries) {
			break
		}
		o, n := pkg.Entries[i], newPkg.Entries[i]
		if o.Name != n.Name {
			rep.Add(Finding{Code: "I13", Severity: SevError,
				Message: "条目名字/顺序变了", Detail: []string{o.Name + " → " + n.Name}})
			continue
		}
		ext := strings.ToLower(filepath.Ext(o.Name))
		if ext == ".4gl" || o.Name == "ver" || (ext != ".tap" && ext != ".tgl") {
			if o.Sha256 != n.Sha256 {
				rep.Add(Finding{Code: "I13", Severity: SevError,
					Message: "本应字节透传的条目被改动了：" + o.Name,
					Detail:  []string{o.Sha256 + " → " + n.Sha256}})
			}
		}
	}

	// 重新合成 + 跑不变量
	doc, err := synth.Synthesize(newPkg, synth.Options{})
	if err != nil {
		rep.Add(Finding{Code: "gate3.synthesize", Severity: SevError,
			Message: "新包无法合成（设计器装载会失败）", Detail: []string{err.Error()}})
		return rep, nil
	}
	fenced, regions, spans, err := fence.Render(doc)
	if err != nil {
		rep.Add(Finding{Code: "gate3.render", Severity: SevError,
			Message: "新包渲染失败", Detail: []string{err.Error()}})
		return rep, nil
	}
	rendered := *doc
	rendered.Text = fenced
	rendered.Regions = regions
	rendered.Spans = spans
	parsed, err := fence.Parse(&rendered, fenced)
	if err != nil {
		rep.Add(Finding{Code: "gate3.fence", Severity: SevError,
			Message: "新包围栏回读失败", Detail: []string{err.Error()}})
		return rep, nil
	}
	newTapDoc, err := tapfile.Parse(newTap)
	if err != nil {
		rep.Add(Finding{Code: "gate3.tap", Severity: SevError, Message: "新 TAP 解析失败", Detail: []string{err.Error()}})
		return rep, nil
	}
	sub := Gate2(&rendered, parsed, newTapDoc)
	if sub.HasError() {
		rep.Add(Finding{Code: "gate3.invariants", Severity: SevError,
			Message: "新包违反不变量（设计器装载可能失败）", Detail: []string{sub.Summary()}})
		for _, f := range sub.Findings {
			if f.Severity == SevError {
				rep.Add(f)
			}
		}
		return rep, nil
	}
	rep.Add(Finding{Code: "gate3.ok", Severity: SevInfo,
		Message: "新包装载模拟通过：合成成功、结构行完整、区段配对、锚点注入正常"})
	return rep, nil
}

// VerifyErrorFromFindings 把 gate 发现打包成 error（退出码 3/4），供 CLI 预检直接抛出。
type VerifyErrorFromFindings struct{ Findings []Finding }

func (e *VerifyErrorFromFindings) Error() string {
	var parts []string
	for _, f := range e.Findings {
		if f.Severity != SevError {
			continue
		}
		s := f.Code + " " + f.Message
		if f.File != "" && f.Line > 0 {
			s += fmt.Sprintf("（%s:%d）", f.File, f.Line)
		}
		parts = append(parts, s)
	}
	if len(parts) == 0 {
		return "验证失败"
	}
	return "验证失败：" + strings.Join(parts, "; ")
}

func (e *VerifyErrorFromFindings) ExitCode() int {
	for _, f := range e.Findings {
		if f.Severity == SevError && f.Denied {
			return 4
		}
	}
	return 3
}

// VerifyErrorFromMsg 是带消息与详情的验证错误（退出码 3）。
type VerifyErrorFromMsg struct {
	Msg    string
	Detail []string
}

func (e *VerifyErrorFromMsg) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *VerifyErrorFromMsg) ExitCode() int { return 3 }
