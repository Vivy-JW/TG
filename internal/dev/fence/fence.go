// Package fence 实现设计指南 §4 的围栏协议：Document ⇄ FencedText。
//
// 契约（设计指南 §5.3）：render 与 parse_fenced 互为逆——
// 对任意合法 Package，parse_fenced(render(synthesize(pkg))) ≡ synthesize(pkg)（Region 序列逐项相等）。
//
// 与设计指南 §4 规则 1 的偏差（实现计划 D-2）：允许 section → point 的一层嵌套，
// 因为真实 TGL 里自订点就住在区段内（3 个锚点区段的正文就是注入的点块）。
// 导出时校验嵌套深度 ≤ 2。
package fence

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"tt/internal/dev/fgl"
	"tt/internal/dev/model"
	"tt/internal/dev/tglfile"
)

// ExitCodeError 让 CLI 把错误映射为退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// FenceError = 退出码 3（配对错误 / 无法归属的散行 / 结构行残缺）。
type FenceError struct {
	Msg    string
	Line   int
	Detail []string
}

func (e *FenceError) Error() string {
	loc := ""
	if e.Line > 0 {
		loc = fmt.Sprintf("（第 %d 行）", e.Line)
	}
	if len(e.Detail) == 0 {
		return e.Msg + loc
	}
	return e.Msg + loc + "：" + strings.Join(e.Detail, "; ")
}

func (e *FenceError) ExitCode() int { return 3 }

var (
	reBegin = regexp.MustCompile(`^\{\/\/@tdev:begin\s+(point|section)\s+(\S+)\s*(?:\[(.*)\])?\}\s*$`)
	reEnd   = regexp.MustCompile(`^\{\/\/@tdev:end\s+(point|section)\}\s*$`)
	reKV    = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)=("[^"]*"|\S+)`)
)

// FencePrefix 是围栏行的固定前缀。导出时若文档里已出现它，直接拒绝（无绕过）。
const FencePrefix = "{//@tdev:"

// 旗标（设计指南 §4.1/§4.2）。
const (
	FlagEditable    = "EDITABLE"     // 可编辑的点
	FlagReadonly    = "READONLY"     // 只读
	FlagEditableSec = "EDITABLE-SEC" // 已解开框架后可编辑的区段
	FlagAppend      = "APPEND"       // 锚点区段：允许追加新 point 块
	FlagPlain       = "PLAIN"        // 裸名插入点：没有函数身份，整块即正文
)

// structFields 是围栏行里**允许改动**的结构事务字段（设计指南 §4.1）。
//
// 它们是唯一不受 gate1「锚定部分逐字节」保护的字段；改动会立即触发 V1–V7 校验。
var structFields = map[string]bool{"fn": true, "scope": true, "desc": true}

// kvOrder 是围栏行关键字的稳定输出顺序。
var kvOrder = []string{
	"deny", "reason", "status", "src", "new", "order", "ver", "cite_std", "edit",
	"fn", "scope", "desc", "origin", "anchor",
}

func quoteKV(k, v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	v = strings.ReplaceAll(v, "\r\n", `\n`)
	v = strings.ReplaceAll(v, "\n", `\n`)
	v = strings.ReplaceAll(v, "\r", `\n`)
	return k + `="` + v + `"`
}

// unquoteKV 还原 quoteKV 的转义。
func unquoteKV(v string) string {
	v = strings.Trim(v, `"`)
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' && i+1 < len(v) {
			switch v[i+1] {
			case 'n':
				b.WriteByte('\n')
				i++
				continue
			case '\\':
				b.WriteByte('\\')
				i++
				continue
			case '"':
				b.WriteByte('"')
				i++
				continue
			}
		}
		b.WriteByte(v[i])
	}
	return b.String()
}

// FenceMeta 是一条 begin 围栏行的结构化视图。
type FenceMeta struct {
	Flags map[string]bool
	KV    map[string]string
}

// ParseFenceMeta 解析 `[...]` 里的旗标与 key="值"。
func ParseFenceMeta(bracket string) FenceMeta {
	m := FenceMeta{Flags: map[string]bool{}, KV: map[string]string{}}
	for _, f := range []string{FlagEditable, FlagReadonly, FlagEditableSec, FlagAppend, FlagPlain} {
		// 旗标是裸词：用词边界匹配，避免把 kv 的值误判成旗标。
		if regexp.MustCompile(`(^|\s)` + f + `(\s|$)`).MatchString(bracket) {
			m.Flags[f] = true
		}
	}
	for _, kv := range reKV.FindAllStringSubmatch(bracket, -1) {
		m.KV[strings.ToLower(kv[1])] = unquoteKV(kv[2])
	}
	return m
}

// AnchoredSignature 是围栏行的**锚定部分**签名：kind/name/旗标/除结构事务字段外的 kv。
//
// 设计指南 §4 规则 2：AI 不许增删改围栏行的锚定部分；fn/scope/desc 是唯一例外。
// gate1 靠它把该规则变成机械校验（而不是脆弱的整行字节比对）。
func AnchoredSignature(kind model.RegionKind, name string, m FenceMeta) string {
	var flags []string
	for f := range m.Flags {
		flags = append(flags, f)
	}
	sortStrings(flags)
	var kvs []string
	for k, v := range m.KV {
		if structFields[k] {
			continue // 结构事务字段不参与锚定比对
		}
		kvs = append(kvs, k+"="+v)
	}
	sortStrings(kvs)
	return string(kind) + " " + name + " [" + strings.Join(flags, ",") + "] " + strings.Join(kvs, " ")
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// BeginFenceLine 渲染一条 begin 围栏行（供 render 与诊断复用）。
func BeginFenceLine(r *model.Region) string {
	var b strings.Builder
	b.WriteString(FencePrefix + "begin " + string(r.Kind) + " " + r.Name)
	b.WriteString(" [")
	switch {
	case r.Kind == model.RegionSection && r.Editable:
		b.WriteString(FlagEditableSec)
	case r.Editable:
		b.WriteString(FlagEditable)
	default:
		b.WriteString(FlagReadonly)
	}
	if r.Plain {
		b.WriteString(" " + FlagPlain)
	}
	if r.Append {
		b.WriteString(" " + FlagAppend)
	}
	if r.DenyCode != "" {
		b.WriteString(" " + quoteKV("deny", r.DenyCode))
	}
	if !r.Editable && r.Reason != "" {
		b.WriteString(" " + quoteKV("reason", r.Reason))
	}
	for _, k := range kvOrder {
		if k == "deny" || k == "reason" {
			continue
		}
		// fn/scope/desc 只在自订定义点上有意义（PLAIN 点没有函数身份）。
		if structFields[k] {
			if r.Plain {
				continue
			}
			switch k {
			case "fn":
				if r.Fn != "" {
					b.WriteString(" " + quoteKV(k, r.Fn))
				}
			case "scope":
				if r.Scope != "" {
					b.WriteString(" " + quoteKV(k, r.Scope))
				}
			case "desc":
				if d := strings.TrimRight(r.Desc, "\r\n"); d != "" {
					b.WriteString(" " + quoteKV(k, d))
				}
			}
			continue
		}
		if v, ok := r.Meta[k]; ok && v != "" {
			b.WriteString(" " + quoteKV(k, v))
		}
	}
	b.WriteString("]}")
	return b.String()
}

func beginLine(r *model.Region) string { return BeginFenceLine(r) }

func endLine(k model.RegionKind) string { return FencePrefix + "end " + string(k) + "}" }

//----------------------------------------------------------------------------
// render
//----------------------------------------------------------------------------

// Render 产出带围栏的文档，并返回**围栏坐标系**下的 Region 表与未归属段。
// 返回的 regions 是基线的深拷贝，区间已换算到围栏文本坐标。
func Render(doc *model.Document) ([]byte, []*model.Region, []model.Span, error) {
	if bytes.Contains(doc.Text, []byte(FencePrefix)) {
		return nil, nil, nil, &FenceError{
			Msg: "文档里已出现围栏前缀，协议与内容冲突（拒绝导出，无绕过）",
			Detail: []string{"字面量：" + FencePrefix,
				"请先确认该程序正文里为什么会有 tdev 围栏标记"},
		}
	}
	if err := checkDepth(doc.Regions, 1); err != nil {
		return nil, nil, nil, err
	}

	var out bytes.Buffer
	out.Grow(len(doc.Text) + 4096)
	cursor := 0

	var renderRegion func(r *model.Region) *model.Region
	renderRegion = func(r *model.Region) *model.Region {
		c := r.DeepCopy()
		delta := 0
		c.Children = nil
		c.FullSpan.Start = out.Len()
		c.BeginFence.Start = out.Len()
		out.WriteString(beginLine(r))
		c.BeginFence.End = out.Len()
		out.WriteByte('\n')
		c.ContentSpan.Start = out.Len()
		delta = c.ContentSpan.Start - r.ContentSpan.Start

		if len(r.Children) > 0 {
			pos := r.ContentSpan.Start
			for _, ch := range r.Children {
				if ch.FullSpan.Start > pos {
					out.Write(doc.Text[pos:ch.FullSpan.Start])
				}
				c.Children = append(c.Children, renderRegion(ch))
				pos = ch.FullSpan.End
			}
			if r.ContentSpan.End > pos {
				out.Write(doc.Text[pos:r.ContentSpan.End])
			}
		} else if r.ContentSpan.End > r.ContentSpan.Start {
			out.Write(doc.Text[r.ContentSpan.Start:r.ContentSpan.End])
		}
		c.ContentSpan.End = out.Len()
		c.Sha256 = model.Sha256Bytes(out.Bytes()[c.ContentSpan.Start:c.ContentSpan.End])
		c.BodySpan = shift(r.BodySpan, delta)
		c.StructureSpans = nil
		for _, s := range r.StructureSpans {
			c.StructureSpans = append(c.StructureSpans, model.ByteRange{Start: s.Start + delta, End: s.End + delta})
		}
		// 内容末尾若不是换行，补一个换行让 end 围栏独立成行，并记 PadEOL 以便 parse 剥回
		content := out.Bytes()[c.ContentSpan.Start:c.ContentSpan.End]
		c.PadEOL = len(content) > 0 && content[len(content)-1] != '\n'
		if c.PadEOL {
			out.WriteByte('\n')
		}
		c.EndFence.Start = out.Len()
		out.WriteString(endLine(r.Kind))
		c.EndFence.End = out.Len()
		// FullSpan 覆盖 [begin 围栏行首, end 围栏行尾)，**不含**行尾换行 ——
		// 必须与 Parse 侧的 `ln.end`（同样不含 EOL）严格一致，
		// 否则围栏外字节会被算进/算出于 FullSpan，导致 gate1 误报。
		c.FullSpan.End = c.EndFence.End
		out.WriteByte('\n')
		return c
	}

	var regions []*model.Region
	for _, r := range doc.Regions {
		if r.FullSpan.Start > cursor {
			out.Write(doc.Text[cursor:r.FullSpan.Start])
			cursor = r.FullSpan.Start
		}
		regions = append(regions, renderRegion(r))
		cursor = r.FullSpan.End
	}
	if cursor < len(doc.Text) {
		out.Write(doc.Text[cursor:])
	}

	fenced := out.Bytes()
	return fenced, regions, recomputeSpans(fenced, regions), nil
}

func checkDepth(rs []*model.Region, d int) error {
	if len(rs) == 0 {
		return nil
	}
	if d > 2 {
		return &FenceError{Msg: "围栏嵌套深度超过 2（只允许 section → point）",
			Detail: []string{rs[0].Name}}
	}
	for _, r := range rs {
		if err := checkDepth(r.Children, d+1); err != nil {
			return err
		}
	}
	return nil
}

func shift(r *model.ByteRange, delta int) *model.ByteRange {
	if r == nil {
		return nil
	}
	return &model.ByteRange{Start: r.Start + delta, End: r.End + delta}
}

// recomputeSpans 在围栏文本上重算未归属字节段（prefix / gap / suffix）。
func recomputeSpans(fenced []byte, regions []*model.Region) []model.Span {
	var covered []model.ByteRange
	for _, r := range regions {
		covered = append(covered, r.FullSpan)
	}
	sortRanges(covered)
	var out []model.Span
	cursor, gap := 0, 0
	add := func(name string, s, e int) {
		if e <= s {
			return
		}
		out = append(out, model.Span{
			Name:   name,
			Span:   model.ByteRange{Start: s, End: e},
			Sha256: model.Sha256Bytes(fenced[s:e]),
		})
	}
	for _, c := range covered {
		if c.Start > cursor {
			name := fmt.Sprintf("gap.%d", gap)
			if cursor == 0 {
				name = "prefix"
			}
			add(name, cursor, c.Start)
			gap++
		}
		if c.End > cursor {
			cursor = c.End
		}
	}
	if cursor < len(fenced) {
		add("suffix", cursor, len(fenced))
	}
	return out
}

func sortRanges(rs []model.ByteRange) {
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && rs[j].Start < rs[j-1].Start; j-- {
			rs[j], rs[j-1] = rs[j-1], rs[j]
		}
	}
}

//----------------------------------------------------------------------------
// 文档内定点编辑（unlock / rename / newfn 共用）
//----------------------------------------------------------------------------

// Edit 是对文档文本的一次替换（半开区间 [Start,End) → Repl）。
type Edit struct {
	Start, End int
	Repl       []byte
}

// ApplyEdits 在文档文本上施加若干互不重叠的替换，并返回**位移修正后**的
// Region 表与未归属段（深拷贝，不改动入参）。
//
// 这是 unlock / rename / newfn 三个命令共用的原语：它们都只改 workspace 里的
// prog.full.4gl（永不碰 .tzc），但改完之后所有区间坐标必须整体校正，
// 否则基线（base/regions.json）就会错位。
func ApplyEdits(doc *model.Document, edits []Edit) ([]byte, []*model.Region, []model.Span, error) {
	if len(edits) == 0 {
		return append([]byte(nil), doc.Text...), doc.Regions, doc.Spans, nil
	}
	// 按起点排序并检查不重叠
	es := append([]Edit(nil), edits...)
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && es[j].Start < es[j-1].Start; j-- {
			es[j], es[j-1] = es[j-1], es[j]
		}
	}
	for i := 1; i < len(es); i++ {
		if es[i].Start < es[i-1].End {
			return nil, nil, nil, &FenceError{
				Msg:    "内部错误：ApplyEdits 的替换区间互相重叠",
				Detail: []string{fmt.Sprintf("[%d,%d) 与 [%d,%d)", es[i-1].Start, es[i-1].End, es[i].Start, es[i].End)},
			}
		}
	}

	var out bytes.Buffer
	out.Grow(len(doc.Text) + 1024)
	cursor := 0
	for _, e := range es {
		if e.Start < cursor || e.End > len(doc.Text) || e.Start > e.End {
			return nil, nil, nil, &FenceError{Msg: "内部错误：ApplyEdits 区间越界",
				Detail: []string{fmt.Sprintf("[%d,%d) len=%d", e.Start, e.End, len(doc.Text))}}
		}
		out.Write(doc.Text[cursor:e.Start])
		out.Write(e.Repl)
		cursor = e.End
	}
	out.Write(doc.Text[cursor:])
	newText := out.Bytes()

	// 位移映射：offset 落在第 k 个替换之后 → 加上累计 delta。
	shift := func(off int) int {
		d := 0
		for _, e := range es {
			if off >= e.End {
				d += len(e.Repl) - (e.End - e.Start)
				continue
			}
			if off >= e.Start {
				// 区间内部：夹到替换后的边界（调用方不应依赖这种情形）
				return e.Start + d
			}
			break
		}
		return off + d
	}

	var fixRange func(r *model.ByteRange) *model.ByteRange
	fixRange = func(r *model.ByteRange) *model.ByteRange {
		if r == nil {
			return nil
		}
		return &model.ByteRange{Start: shift(r.Start), End: shift(r.End)}
	}

	var walk func(rs []*model.Region) []*model.Region
	walk = func(rs []*model.Region) []*model.Region {
		if rs == nil {
			return nil
		}
		out := make([]*model.Region, 0, len(rs))
		for _, r := range rs {
			c := r.DeepCopy()
			c.ContentSpan = *fixRange(&r.ContentSpan)
			c.FullSpan = *fixRange(&r.FullSpan)
			c.BeginFence = *fixRange(&r.BeginFence)
			c.EndFence = *fixRange(&r.EndFence)
			c.BodySpan = fixRange(r.BodySpan)
			c.DescSpan = fixRange(r.DescSpan)
			c.SignatureSpan = fixRange(r.SignatureSpan)
			if r.StructureSpans != nil {
				c.StructureSpans = nil
				for _, s := range r.StructureSpans {
					c.StructureSpans = append(c.StructureSpans, *fixRange(&s))
				}
			}
			c.Sha256 = model.Sha256Bytes(newText[c.ContentSpan.Start:c.ContentSpan.End])
			c.Children = walk(r.Children)
			out = append(out, c)
		}
		return out
	}
	newRegions := walk(doc.Regions)
	newSpans := recomputeSpans(newText, newRegions)
	return newText, newRegions, newSpans, nil
}

//----------------------------------------------------------------------------
// parse_fenced
//----------------------------------------------------------------------------

// Pair 是一次「基线与编辑后」的配对。
//
// 必须用配对而不是按名字查表：真实包里存在**区段 id 与点名同名**的情形
// （例如 `wssp00316.process` 既是 `<section id>` 又是区段内的裸名插入点），
// 按名字查表会张冠李戴。
type Pair struct {
	Base   *model.Region
	Edited *model.Region
	// BaseFence / EditedFence 是两侧 begin 围栏行的原文（诊断用）。
	BaseFence   string
	EditedFence string
	// BaseMeta / EditedMeta 是两侧 begin 围栏行的结构化视图。
	// gate1 用 AnchoredSignature 比对**锚定部分**；fn/scope/desc 由结构事务校验。
	BaseMeta   FenceMeta
	EditedMeta FenceMeta
}

// StructFieldChanged 判断某结构事务字段相对基线是否被改动。
func (p Pair) StructFieldChanged(field string) (from, to string, changed bool) {
	b := p.BaseMeta.KV[field]
	e := p.EditedMeta.KV[field]
	return b, e, b != e
}

// ParseResult 是解析结果。
type ParseResult struct {
	// Doc 是编辑后的文档（Text = 围栏文本，Regions 为围栏坐标）。
	Doc *model.Document
	// Pairs 是基线↔编辑后的配对（按 end 围栏出现顺序）。
	Pairs []Pair
	// Appended 是新增的 point 区间（在 APPEND 锚点内追加）。
	Appended []*model.Region
	// Deleted 是围栏块消失的基线区间（保留基线元数据，供删除授权判定）。
	Deleted []*model.Region
}

type frame struct {
	reg          *model.Region
	base         *model.Region
	contentStart int
	// 围栏行的原文与结构化视图（gate1 锚定比对 + 结构事务检测用）。
	baseFence   string
	editedFence string
	baseMeta    FenceMeta
	editedMeta  FenceMeta
}

// bracketOf 取基线 Region 的 begin 围栏行里的 `[...]` 内容。
func bracketOf(base *model.Document, r *model.Region) string {
	line := baseFenceLineOf(base, r)
	i := strings.IndexByte(line, '[')
	j := strings.LastIndexByte(line, ']')
	if i < 0 || j <= i {
		return ""
	}
	return line[i+1 : j]
}

// baseFenceLineOf 取基线文本里该 Region 的 begin 围栏行原文。
func baseFenceLineOf(base *model.Document, r *model.Region) string {
	if base == nil || r.BeginFence.End <= r.BeginFence.Start || r.BeginFence.End > len(base.Text) {
		return ""
	}
	return string(base.Text[r.BeginFence.Start:r.BeginFence.End])
}

type srcLine struct{ start, end int }

func splitLines(b []byte) []srcLine {
	var lines []srcLine
	i := 0
	for i <= len(b) {
		j := bytes.IndexByte(b[i:], '\n')
		if j < 0 {
			lines = append(lines, srcLine{i, len(b)})
			break
		}
		e := i + j
		if e > i && b[e-1] == '\r' {
			lines = append(lines, srcLine{i, e - 1})
		} else {
			lines = append(lines, srcLine{i, e})
		}
		i = e + 1
	}
	return lines
}

func nextLineStart(b []byte, lineEnd int) int {
	if lineEnd < len(b) {
		if b[lineEnd] == '\n' {
			return lineEnd + 1
		}
		if b[lineEnd] == '\r' {
			if lineEnd+1 < len(b) && b[lineEnd+1] == '\n' {
				return lineEnd + 2
			}
			return lineEnd + 1
		}
	}
	return lineEnd
}

// lineNo 返回字节偏移所在行（1-based）。口径统一到 model.LineOf：
// 只认 \n、越界偏移夹到合法范围（旧实现用 b[:off] 计数，off 越界会 panic）。
func lineNo(b []byte, off int) int { return model.LineOf(b, off) }

func matchKey(kind model.RegionKind, name string) string { return string(kind) + " " + name }

func pointTypeOf(name string) (tglfile.AnchorKind, bool) {
	switch {
	case strings.HasPrefix(name, "function."):
		return tglfile.AnchorFunction, true
	case strings.HasPrefix(name, "dialog."):
		return tglfile.AnchorDialog, true
	case strings.HasPrefix(name, "report."):
		return tglfile.AnchorReport, true
	}
	return "", false
}

// Parse 解析编辑后的围栏文本，与基线 Region 表逐项对拍。
func Parse(base *model.Document, fenced []byte) (*ParseResult, error) {
	expected := base.AllRegions()
	ei := 0

	res := &ParseResult{Doc: &model.Document{
		Text: fenced,
		Env:  base.Env, Prog: base.Prog, Ver: base.Ver,
		PkgPath: base.PkgPath, PkgSha256: base.PkgSha256,
		SectionState: base.SectionState, PendingUnlock: base.PendingUnlock,
		UnlockedBy: base.UnlockedBy, Only: base.Only, Anchors: base.Anchors,
	}}

	var stack []frame
	for _, ln := range splitLines(fenced) {
		text := string(fenced[ln.start:ln.end])

		if m := reBegin.FindStringSubmatch(text); m != nil {
			kind := model.RegionKind(m[1])
			name := m[2]
			if len(stack) >= 2 {
				return nil, &FenceError{Msg: "围栏嵌套超过两层（只允许 section → point）", Line: lineNo(fenced, ln.start), Detail: []string{name}}
			}

			var matched *model.Region
			if ei < len(expected) && matchKey(expected[ei].Kind, expected[ei].Name) == matchKey(kind, name) {
				matched = expected[ei]
				ei++
			} else {
				// 只有「点」可以被整块跳过（= 删除）；区段不允许被跳过。
				j := -1
				for k := ei; k < len(expected); k++ {
					if matchKey(expected[k].Kind, expected[k].Name) == matchKey(kind, name) {
						j = k
						break
					}
				}
				if j >= 0 {
					for k := ei; k < j; k++ {
						res.Deleted = append(res.Deleted, expected[k])
					}
					matched = expected[j]
					ei = j + 1
				}
			}

			if matched == nil {
				// 新增点：只允许出现在 APPEND 锚点区段内，且类型匹配
				if len(stack) == 0 {
					return nil, &FenceError{
						Msg:    "无法归属的围栏块（不在基线序列里，也不在 APPEND 锚点区段内）",
						Line:   lineNo(fenced, ln.start),
						Detail: []string{matchKey(kind, name)},
					}
				}
				parent := stack[len(stack)-1].reg
				if kind != model.RegionPoint || !parent.Append {
					return nil, &FenceError{
						Msg:    "无法归属的围栏块（父区段不允许追加）",
						Line:   lineNo(fenced, ln.start),
						Detail: []string{matchKey(kind, name), "父区段：" + parent.Name},
					}
				}
				pt, ok := pointTypeOf(name)
				if !ok || string(pt) != parent.AnchorType {
					return nil, &FenceError{
						Msg:    "追加点的类型与锚点区段不匹配",
						Line:   lineNo(fenced, ln.start),
						Detail: []string{name, "锚点类型：" + parent.AnchorType},
					}
				}
				if base.FindRegion(name) != nil {
					return nil, &FenceError{
						Msg:    "追加点与已有 Region 重名",
						Line:   lineNo(fenced, ln.start),
						Detail: []string{name},
					}
				}
				reg := newAppendedRegion(name, m[3])
				reg.FullSpan.Start = ln.start
				reg.BeginFence = model.ByteRange{Start: ln.start, End: ln.end}
				attach(&res.Doc.Regions, stack, reg)
				res.Appended = append(res.Appended, reg)
				stack = append(stack, frame{reg: reg, contentStart: nextLineStart(fenced, ln.end)})
				continue
			}

			reg := matched.DeepCopy()
			reg.Deleted = false
			// 子区间必须来自**编辑后的围栏文本**重新解析的结果，
			// 不能保留基线深拷贝里的子区间（否则子区间会被计两次）。
			reg.Children = nil
			reg.FullSpan.Start = ln.start
			reg.BeginFence = model.ByteRange{Start: ln.start, End: ln.end}
			// 记录两侧围栏行的结构化视图：gate1 比锚定部分，结构事务比 fn/scope/desc。
			fr := frame{reg: reg, base: matched, contentStart: nextLineStart(fenced, ln.end)}
			fr.editedMeta = ParseFenceMeta(m[3])
			fr.editedFence = text
			fr.baseMeta = ParseFenceMeta(bracketOf(base, matched))
			fr.baseFence = baseFenceLineOf(base, matched)
			attach(&res.Doc.Regions, stack, reg)
			stack = append(stack, fr)
			continue
		}

		if m := reEnd.FindStringSubmatch(text); m != nil {
			if len(stack) == 0 {
				return nil, &FenceError{Msg: "围栏 end 多于 begin", Line: lineNo(fenced, ln.start)}
			}
			f := stack[len(stack)-1]
			if string(f.reg.Kind) != m[1] {
				return nil, &FenceError{
					Msg:    "围栏 end 类型与 begin 不匹配",
					Line:   lineNo(fenced, ln.start),
					Detail: []string{"begin=" + string(f.reg.Kind), "end=" + m[1]},
				}
			}
			content := fenced[f.contentStart:ln.start]
			if f.reg.PadEOL && len(content) > 0 && content[len(content)-1] == '\n' {
				content = content[:len(content)-1]
				// 编辑器可能把我们补的那个换行写成 CRLF；必须连 CR 一起剥掉
				if len(content) > 0 && content[len(content)-1] == '\r' {
					content = content[:len(content)-1]
				}
			}
			f.reg.ContentSpan = model.ByteRange{Start: f.contentStart, End: f.contentStart + len(content)}
			f.reg.EndFence = model.ByteRange{Start: ln.start, End: ln.end}
			f.reg.FullSpan.End = ln.end
			f.reg.Sha256 = model.Sha256Bytes(content)
			recomputeBody(f.reg, content)
			res.Pairs = append(res.Pairs, Pair{
				Base: f.base, Edited: f.reg,
				BaseFence: f.baseFence, EditedFence: f.editedFence,
				BaseMeta: f.baseMeta, EditedMeta: f.editedMeta,
			})
			stack = stack[:len(stack)-1]
			continue
		}
		// 其余行是围栏外字节：不解释、不改动，交由 gate1 逐字节比对。
	}
	if len(stack) != 0 {
		return nil, &FenceError{Msg: "围栏 begin 没有对应的 end", Detail: []string{stack[len(stack)-1].reg.Name}}
	}
	for ; ei < len(expected); ei++ {
		res.Deleted = append(res.Deleted, expected[ei])
	}
	for _, r := range res.Deleted {
		if r.Kind != model.RegionPoint {
			return nil, &FenceError{
				Msg:    "区段围栏块被删除（区段不可删除，只能改正文）",
				Detail: []string{r.Name},
			}
		}
	}
	for _, d := range res.Deleted {
		d.Deleted = true
	}
	res.Doc.Spans = recomputeSpans(fenced, res.Doc.Regions)
	return res, nil
}

// attach 把区间挂到当前栈顶区段（或顶层）。
func attach(top *[]*model.Region, stack []frame, reg *model.Region) {
	if len(stack) > 0 {
		parent := stack[len(stack)-1].reg
		parent.Children = append(parent.Children, reg)
		return
	}
	*top = append(*top, reg)
}

// recomputeBody 依据编辑后的内容重算三个权限区（绝对坐标）。
//
// 与 synth.buildPointRegion 的口径必须逐字一致，否则 gate1 的结构行比对会错位：
//
//	描述块 = 签名行之前（可编辑，V6 校验形态）
//	签名行 = SignatureSpan（结构事务治理，gate1 豁免）
//	END 行 = StructureSpans（绝对不可改，gate1 逐字节）
//	正文   = BodySpan（自由编辑）
func recomputeBody(r *model.Region, content []byte) {
	r.StructureSpans = nil
	r.SignatureSpan = nil
	r.DescSpan = nil
	if r.Kind != model.RegionPoint {
		r.BodySpan = nil
		return
	}
	if _, selfDef := pointTypeOf(r.Name); selfDef {
		kind, _ := fgl.EnvelopeKindFor(r.Name)
		blk, err := fgl.ParseBlock(string(content), kind)
		if err != nil || blk == nil {
			r.BodySpan = nil
			r.DenyCode = model.DenyStructureUnparsable
			r.Reason = model.DenyReasonText(model.DenyStructureUnparsable)
			return
		}
		base := r.ContentSpan.Start
		if blk.Header.StartByte > 0 {
			d := model.ByteRange{Start: base, End: base + blk.Header.StartByte}
			r.DescSpan = &d
			r.Desc = string(content[:blk.Header.StartByte])
		} else {
			z := model.ByteRange{Start: base, End: base}
			r.DescSpan = &z
			r.Desc = ""
		}
		sg := model.ByteRange{Start: base + blk.Header.StartByte, End: base + blk.Header.EndByte}
		r.SignatureSpan = &sg
		r.Scope = blk.Scope
		r.Fn = signatureNameOf(string(content[blk.Header.StartByte:blk.Header.EndByte]))
		r.StructureSpans = []model.ByteRange{
			{Start: base + blk.Terminator.StartByte, End: base + blk.Terminator.EndByte},
		}
		if blk.Body.EndByte > blk.Body.StartByte {
			r.BodySpan = &model.ByteRange{
				Start: base + blk.Body.StartByte,
				End:   base + blk.Body.EndByte,
			}
		} else {
			z := model.ByteRange{Start: base + blk.Header.EndByte, End: base + blk.Header.EndByte}
			r.BodySpan = &z
		}
		return
	}
	r.Plain = true
	z := model.ByteRange{Start: r.ContentSpan.Start, End: r.ContentSpan.End}
	r.BodySpan = &z
}

var (
	reSigScopeF = regexp.MustCompile(`(?i)^(PUBLIC|PRIVATE)[ \t]+`)
	reSigKindF  = regexp.MustCompile(`(?i)^(FUNCTION|DIALOG|REPORT)[ \t]*`)
)

// signatureNameOf 从签名行取出「名字(参数)」（与 synth.signatureName 同口径）。
func signatureNameOf(headerLine string) string {
	s := strings.TrimLeft(headerLine, " \t")
	if m := reSigScopeF.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	if m := reSigKindF.FindStringSubmatch(s); m != nil {
		s = s[len(m[0]):]
	}
	s = strings.TrimLeft(s, " \t")
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimRight(s, " \t")
}

func newAppendedRegion(name, kvText string) *model.Region {
	meta := map[string]string{}
	for _, m := range reKV.FindAllStringSubmatch(kvText, -1) {
		meta[strings.ToLower(m[1])] = strings.Trim(m[2], `"`)
	}
	if meta["new"] == "" {
		meta["new"] = "Y"
	}
	if meta["status"] == "" {
		meta["status"] = "u" // D-1：新增点写 u，不写 c
	}
	reg := &model.Region{
		Name:     name,
		Kind:     model.RegionPoint,
		Editable: true,
		Meta:     meta,
		Origin:   "appended",
	}
	if _, selfDef := pointTypeOf(name); !selfDef {
		reg.DenyCode = model.DenyStructureUnparsable
	}
	return reg
}
