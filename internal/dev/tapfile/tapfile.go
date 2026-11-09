// Package tapfile 实现设计指南 §5.4「TAP 层 —— CDATA 感知的字节保真改写」。
//
// 依据：
//   - 设计指南 §5.4、§7 R3（禁止用 XML 库整体序列化 .tap）
//   - 反编译源码：根元素真名 add_points、point/section 结构；混合换行（元素间 CRLF /
//     CDATA 内 LF）；旧工具的教训是只用 CDATA 感知扫描器定位元素区间，改写仅重建
//     CDATA 内部，属性改动「有则改、无则加」，属性顺序/空白/引号全部原样保留
//
// 本包是「不破坏」的物理基础之一：除了显式给定的改写点，其余字节逐字节不动。
package tapfile

import (
	"bytes"
	"fmt"
	"strings"
)

// FormatError = 退出码 2（包格式错误）。
type FormatError struct {
	Msg    string
	At     int
	Detail []string
}

func (e *FormatError) Error() string {
	loc := ""
	if e.At > 0 {
		loc = fmt.Sprintf("（字节偏移 %d）", e.At)
	}
	if len(e.Detail) == 0 {
		return e.Msg + loc
	}
	return e.Msg + loc + "：" + strings.Join(e.Detail, "; ")
}

func (e *FormatError) ExitCode() int { return 2 }

// Attr 是开放标签上的一个属性。ValueStart/ValueEnd 精确指向**引号内的值**，
// 用于「有则改」时只替换值本身，保住引号风格。
type Attr struct {
	Name       string
	Value      string
	ValueStart int // 相对 Raw 的绝对偏移
	ValueEnd   int
}

// Element 是扫描出的一个元素及其字节区间。
type Element struct {
	Name        string
	Start       int // 整个元素（含开闭标签）[Start, End)
	End         int
	TagStart    int // 开放标签 <...> 或 <... />
	TagEnd      int
	SelfClosing bool
	Attrs       []Attr
	CDATAStart  int // CDATA **内容**区间（<![CDATA[ 之后、]]> 之前）；-1 表示无
	CDATAEnd    int
	Children    []*Element
	TextStart   int // 非 CDATA 的文本内容区间（多为空白）；-1 表示无
	TextEnd     int
	Parent      *Element
}

// Attr 取属性值。
func (e *Element) Attr(key string) (string, bool) {
	for _, a := range e.Attrs {
		if a.Name == key {
			return a.Value, true
		}
	}
	return "", false
}

// AttrOr 取属性值，缺失时返回默认值。
func (e *Element) AttrOr(key, def string) string {
	if v, ok := e.Attr(key); ok {
		return v
	}
	return def
}

// HasCDATA 判断元素是否带 CDATA 内容。
func (e *Element) HasCDATA() bool { return e.CDATAStart >= 0 }

// Content 返回元素内容：CDATA 内部字节，或普通文本，或空。
func (e *Element) Content(raw []byte) []byte {
	if e.HasCDATA() {
		return raw[e.CDATAStart:e.CDATAEnd]
	}
	if e.TextStart >= 0 {
		return raw[e.TextStart:e.TextEnd]
	}
	return nil
}

// Doc 是扫描结果的视图。
type Doc struct {
	Raw      []byte
	Root     *Element
	Points   []*Element
	Sections []*Element
	Others   []*Element // <other> 子元素（code_template / free_style / start_arg …）
}

// RootAttr 读根元素属性。
func (d *Doc) RootAttr(key string) (string, bool) { return d.Root.Attr(key) }

// RootAttrOr 读根元素属性，缺失时返回默认值。
func (d *Doc) RootAttrOr(key, def string) string { return d.Root.AttrOr(key, def) }

// PointExact 返回第一个同名 <point>，不看 status。
func (d *Doc) PointExact(name string) *Element {
	for _, p := range d.Points {
		if v, _ := p.Attr("name"); v == name {
			return p
		}
	}
	return nil
}

// PointsAll 返回所有同名 <point>，按文档顺序。
//
// 真实语料里同名点会出现两次：先是 tombstone（status="d"），随后是 live（status="u"）。
// 这正是设计器 ProgramInformation.AddPoint getter 的组装顺序（先删除点、后存活点，
// 见 ProgramInformation.cs:553-613）。
func (d *Doc) PointsAll(name string) []*Element {
	var out []*Element
	for _, p := range d.Points {
		if v, _ := p.Attr("name"); v == name {
			out = append(out, p)
		}
	}
	return out
}

// PointAt 返回包含字节偏移 off 的 <point>；off 不在任何 point 内时返回 nil。
//
// 用途：TAP 层的报错（BOM / 0x07 / 解析失败）现在只给一个字节偏移，
// 有了它就能说清「落在哪个点上」，报错才可定位。
func (d *Doc) PointAt(off int) *Element {
	if d == nil || off < 0 || off > len(d.Raw) {
		return nil
	}
	for _, p := range d.Points {
		if off >= p.Start && off < p.End {
			return p
		}
	}
	return nil
}

// SectionAt 返回包含字节偏移 off 的 <section>；不在任何 section 内时返回 nil。
func (d *Doc) SectionAt(off int) *Element {
	if d == nil || off < 0 || off > len(d.Raw) {
		return nil
	}
	for _, s := range d.Sections {
		if off >= s.Start && off < s.End {
			return s
		}
	}
	return nil
}

// IsDeleted 判断点/区段是否被标记删除（status="d"）。
func IsDeleted(el *Element) bool {
	v, ok := el.Attr("status")
	return ok && v == "d"
}

// Point 按设计器语义解析点名：取第一个 **未删除** 的同名 <point>
// （CodeEditorManager.cs:1188 `AddPoints.Where(a => a.Name == name && (a.Status & DELETE) == NULL)`）。
// 全部被删除时返回第一个（调用方应据此判定该点不在文档中）。
//
// 所有写操作都必须经由此解析，绝不能误改 tombstone。
func (d *Doc) Point(name string) *Element {
	var first *Element
	for _, p := range d.Points {
		if v, _ := p.Attr("name"); v != name {
			continue
		}
		if first == nil {
			first = p
		}
		if !IsDeleted(p) {
			return p
		}
	}
	return first
}

// Section 按 id 属性取 <section>。
func (d *Doc) Section(id string) *Element {
	for _, s := range d.Sections {
		if v, _ := s.Attr("id"); v == id {
			return s
		}
	}
	return nil
}

// AppendAnchor 返回新增 <point> 的插入偏移与分隔符样式：
// 最后一个 <point> 之后（无 point 则 <other> 之后，再无则根标签之后）。
func (d *Doc) AppendAnchor() (offset int, sep []byte) {
	anchor := d.Root.TagEnd
	if len(d.Others) > 0 {
		anchor = d.Others[len(d.Others)-1].End
	}
	if len(d.Points) > 0 {
		anchor = d.Points[len(d.Points)-1].End
	}
	return anchor, whitespaceRunFrom(d.Raw, anchor)
}

// whitespaceRunFrom 返回从 off 开始的连续空白（分隔符），无空白则返回 "\n"。
func whitespaceRunFrom(raw []byte, off int) []byte {
	i := off
	for i < len(raw) {
		c := raw[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		break
	}
	if i == off {
		return []byte("\n")
	}
	return raw[off:i]
}

//---------------------------------------------------------------------------
// 扫描器
//---------------------------------------------------------------------------

func isNameByte(c byte) bool {
	return c == '_' || c == '-' || c == '.' || c == ':' ||
		(c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// Parse 扫描 TAP 字节，只做「结构定位」，不做任何 XML 规范化。
func Parse(raw []byte) (*Doc, error) {
	d := &Doc{Raw: raw, Others: nil}
	i := 0
	// 跳过 XML 声明 / 注释 / DOCTYPE / 空白，直到根元素。
	for {
		i = skipMisc(raw, i)
		if i >= len(raw) {
			return nil, &FormatError{Msg: "TAP 里没有找到根元素", At: i}
		}
		break
	}
	root, next, err := parseElement(raw, i, nil)
	if err != nil {
		return nil, err
	}
	d.Root = root
	i = next

	// 根元素的直接子元素分类。
	for _, c := range root.Children {
		switch c.Name {
		case "point":
			d.Points = append(d.Points, c)
		case "section":
			d.Sections = append(d.Sections, c)
		case "other":
			d.Others = append(d.Others, c.Children...)
		}
	}
	return d, nil
}

// skipMisc 跳过空白、XML 声明、注释、DOCTYPE。
func skipMisc(raw []byte, i int) int {
	for i < len(raw) {
		c := raw[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case bytes.HasPrefix(raw[i:], []byte("<?")):
			j := bytes.Index(raw[i:], []byte("?>"))
			if j < 0 {
				return len(raw)
			}
			i += j + 2
		case bytes.HasPrefix(raw[i:], []byte("<!--")):
			j := bytes.Index(raw[i:], []byte("-->"))
			if j < 0 {
				return len(raw)
			}
			i += j + 3
		case bytes.HasPrefix(raw[i:], []byte("<!")):
			j := bytes.IndexByte(raw[i:], '>')
			if j < 0 {
				return len(raw)
			}
			i += j + 1
		default:
			return i
		}
	}
	return i
}

// parseOpenTag 解析开放标签，返回标签结束偏移（紧接 > 或 /> 之后）与属性表。
func parseOpenTag(raw []byte, start int) (name string, attrs []Attr, selfClosing bool, tagEnd int, err error) {
	if start >= len(raw) || raw[start] != '<' {
		return "", nil, false, 0, &FormatError{Msg: "内部错误：开放标签起始位置不是 '<'", At: start}
	}
	i := start + 1
	ns := i
	for i < len(raw) && isNameByte(raw[i]) {
		i++
	}
	if i == ns {
		return "", nil, false, 0, &FormatError{Msg: "标签名缺失", At: start}
	}
	name = string(raw[ns:i])

	for i < len(raw) {
		// 跳过空白
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r' || raw[i] == '\n') {
			i++
		}
		if i >= len(raw) {
			break
		}
		if raw[i] == '>' {
			return name, attrs, false, i + 1, nil
		}
		if raw[i] == '/' {
			if i+1 < len(raw) && raw[i+1] == '>' {
				return name, attrs, true, i + 2, nil
			}
			return "", nil, false, 0, &FormatError{Msg: "开放标签里出现裸 '/'", At: i}
		}
		// 属性名
		as := i
		for i < len(raw) && isNameByte(raw[i]) {
			i++
		}
		if i == as {
			return "", nil, false, 0, &FormatError{Msg: "无法解析的属性名", At: i}
		}
		an := string(raw[as:i])
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r' || raw[i] == '\n') {
			i++
		}
		if i >= len(raw) || raw[i] != '=' {
			// 无值属性（本语料未见，保守处理为空值）
			attrs = append(attrs, Attr{Name: an, Value: "", ValueStart: i, ValueEnd: i})
			continue
		}
		i++
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\r' || raw[i] == '\n') {
			i++
		}
		if i >= len(raw) {
			break
		}
		switch raw[i] {
		case '"', '\'':
			q := raw[i]
			vs := i + 1
			ve := bytes.IndexByte(raw[vs:], q)
			if ve < 0 {
				return "", nil, false, 0, &FormatError{Msg: "属性值引号未闭合", At: vs}
			}
			ve += vs
			attrs = append(attrs, Attr{Name: an, Value: string(raw[vs:ve]), ValueStart: vs, ValueEnd: ve})
			i = ve + 1
		default:
			vs := i
			for i < len(raw) && raw[i] != ' ' && raw[i] != '\t' && raw[i] != '\r' && raw[i] != '\n' && raw[i] != '>' && raw[i] != '/' {
				i++
			}
			attrs = append(attrs, Attr{Name: an, Value: string(raw[vs:i]), ValueStart: vs, ValueEnd: i})
		}
	}
	return "", nil, false, 0, &FormatError{Msg: "开放标签未闭合（缺 '>'）", At: start}
}

// parseElement 解析一个元素（含子元素/CDATA/文本），返回元素与下一个偏移。
func parseElement(raw []byte, start int, parent *Element) (*Element, int, error) {
	name, attrs, selfClosing, tagEnd, err := parseOpenTag(raw, start)
	if err != nil {
		return nil, 0, err
	}
	el := &Element{
		Name:        name,
		Start:       start,
		TagStart:    start,
		TagEnd:      tagEnd,
		SelfClosing: selfClosing,
		Attrs:       attrs,
		CDATAStart:  -1,
		CDATAEnd:    -1,
		TextStart:   -1,
		TextEnd:     -1,
		Parent:      parent,
	}
	if selfClosing {
		el.End = tagEnd
		return el, tagEnd, nil
	}
	i := tagEnd
	textStart := -1
	for i < len(raw) {
		switch {
		case bytes.HasPrefix(raw[i:], []byte("<![CDATA[")):
			if textStart >= 0 && el.TextStart < 0 {
				el.TextStart, el.TextEnd = textStart, i
			}
			cs := i + len("<![CDATA[")
			j := bytes.Index(raw[cs:], []byte("]]>"))
			if j < 0 {
				return nil, 0, &FormatError{Msg: "CDATA 未闭合", At: i}
			}
			ce := cs + j
			// 本实现只支持「元素内最多一段 CDATA」，与真实 TAP 形态一致。
			if el.CDATAStart >= 0 {
				return nil, 0, &FormatError{Msg: "元素内出现多段 CDATA（本实现不支持）", At: i}
			}
			el.CDATAStart, el.CDATAEnd = cs, ce
			i = ce + 3
			textStart = -1
		case bytes.HasPrefix(raw[i:], []byte("</")):
			if textStart >= 0 && el.TextStart < 0 {
				el.TextStart, el.TextEnd = textStart, i
			}
			i += 2
			ns := i
			for i < len(raw) && isNameByte(raw[i]) {
				i++
			}
			cn := string(raw[ns:i])
			for i < len(raw) && raw[i] != '>' {
				i++
			}
			if i >= len(raw) {
				return nil, 0, &FormatError{Msg: "闭合标签未终结", At: ns}
			}
			i++
			if cn != el.Name {
				return nil, 0, &FormatError{
					Msg:    "闭合标签与开放标签不匹配",
					At:     ns,
					Detail: []string{"期待 </" + el.Name + ">，实际 </" + cn + ">"},
				}
			}
			el.End = i
			return el, i, nil
		case raw[i] == '<' && i+1 < len(raw) && isNameByte(raw[i+1]):
			if textStart >= 0 && el.TextStart < 0 {
				el.TextStart, el.TextEnd = textStart, i
			}
			child, next, err := parseElement(raw, i, el)
			if err != nil {
				return nil, 0, err
			}
			el.Children = append(el.Children, child)
			i = next
			textStart = -1
		default:
			if textStart < 0 {
				textStart = i
			}
			i++
		}
	}
	return nil, 0, &FormatError{Msg: "元素未闭合：<" + el.Name + ">", At: el.Start}
}

//---------------------------------------------------------------------------
// 改写
//---------------------------------------------------------------------------

// Op 是一次改写操作。所有操作只改「显式指定的字节」，其余原样保留。
type Op interface {
	apply(raw []byte, d *Doc) ([]byte, error)
	describe() string
}

// SetRootAttr 改/加根元素属性（SetAttributeValue 语义：有则原位改，无则追加到标签末尾）。
type SetRootAttr struct{ Key, Value string }

// SetPointAttr 改/加某个 <point> 的属性。
type SetPointAttr struct{ Name, Key, Value string }

// SetPointCDATA 只重建某个 <point> 的 CDATA 内部。
type SetPointCDATA struct {
	Name    string
	Content []byte
}

// MarkDeleted 把点标记为删除：只改 status 属性，**保留原内容**（I5）。
type MarkDeleted struct{ Name string }

// AddPoint 追加一个全新的 <point> 元素。
//
// 元素字节以**同包内已有的 <point> 为模板**（缩进、引号风格、CDATA 前置换行、
// 属性顺序全部继承），避免自创风格；无模板时用设计器 ctor 的属性顺序兜底。
type AddPoint struct {
	Attrs   map[string]string // 至少含 name；其余为设计器 ctor 的字段集
	Content []byte
}

// RemoveTombstones 删除同名 <point> 里**所有 status="d" 的墓碑**（含其行尾）。
//
// 用于复刻设计器改名事务的第 ① 步（ProgramInformation.Modify:118-122：
// `Where(a => a.Name == Source && (a.Status & Status.DELETE) == Status.DELETE)`）。
// 没有墓碑时是 **no-op**（不是错误）—— 首次改名时本来就没有墓碑。
type RemoveTombstones struct{ Name string }

// RenamePoint 改 <point> 的 name 属性（元素本身与 CDATA 不动）。
//
// 用于复刻改名事务的第 ② 步：活点的 Name 随 FunctionName 变为 function.新名
// （SetName，AddPointModel.cs:810-830）。
type RenamePoint struct{ From, To string }

// SetSectionCDATA 只重建某个 <section> 的 CDATA 内部。
type SetSectionCDATA struct {
	ID      string
	Content []byte
}

// SetSectionAttr 改/加某个 <section> 的属性。
type SetSectionAttr struct{ ID, Key, Value string }

func (o SetRootAttr) describe() string { return "set root attr " + o.Key }
func (o SetPointAttr) describe() string {
	return "set point attr " + o.Name + "." + o.Key
}
func (o SetPointCDATA) describe() string { return "set point cdata " + o.Name }
func (o MarkDeleted) describe() string   { return "mark deleted " + o.Name }
func (o RemoveTombstones) describe() string {
	return "remove tombstones " + o.Name
}
func (o RenamePoint) describe() string { return "rename point " + o.From + " → " + o.To }
func (o AddPoint) describe() string    { return "add point " + o.Attrs["name"] }
func (o SetSectionCDATA) describe() string {
	return "set section cdata " + o.ID
}
func (o SetSectionAttr) describe() string {
	return "set section attr " + o.ID + "." + o.Key
}

// setAttr 在元素开放标签里改/加属性，返回新的整段字节。
// 已有属性：只替换引号内的值（引号风格、属性顺序、周边空白全不变）。
// 新属性：追加到开放标签末尾（若原标签在 '>' 前有空白则沿用，否则补一个空格）。
func setAttr(raw []byte, d *Doc, el *Element, key, value string) ([]byte, error) {
	tag := raw[el.TagStart:el.TagEnd]
	var out []byte
	var found *Attr
	for i := range el.Attrs {
		if el.Attrs[i].Name == key {
			found = &el.Attrs[i]
			break
		}
	}
	if found != nil {
		out = make([]byte, 0, len(tag)+len(value))
		out = append(out, raw[el.TagStart:found.ValueStart]...)
		out = append(out, []byte(escapeAttrValue(value))...)
		out = append(out, raw[found.ValueEnd:el.TagEnd]...)
	} else {
		// 开放标签最后一个字节是 '>'，自闭合则是 '/>'。
		insertAt := len(tag) - 1
		if el.SelfClosing {
			insertAt = len(tag) - 2
		}
		sep := " "
		if insertAt > 0 {
			prev := tag[insertAt-1]
			if prev == ' ' || prev == '\t' || prev == '\r' || prev == '\n' {
				sep = ""
			}
		}
		out = make([]byte, 0, len(tag)+len(key)+len(value)+4)
		out = append(out, tag[:insertAt]...)
		out = append(out, []byte(sep+key+"=\""+escapeAttrValue(value)+"\"")...)
		out = append(out, tag[insertAt:]...)
	}
	// 用新标签替换原标签字节。
	res := make([]byte, 0, len(raw)+len(out)-len(tag))
	res = append(res, raw[:el.TagStart]...)
	res = append(res, out...)
	res = append(res, raw[el.TagEnd:]...)
	return res, nil
}

func escapeAttrValue(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			b.WriteString("&quot;")
		case '<':
			b.WriteString("&lt;")
		case '&':
			b.WriteString("&amp;")
		case '\n':
			b.WriteString("&#xA;")
		case '\r':
			b.WriteString("&#xD;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// setCDATA 只替换 CDATA 内部字节。若元素是自闭合且尚无 CDATA，则按模板展开。
func setCDATA(raw []byte, d *Doc, el *Element, content []byte, template *Element) ([]byte, error) {
	if bytes.Contains(content, []byte("]]>")) {
		return nil, &FormatError{
			Msg: "内容含 ]]>，无法作为 CDATA 写入（XML 不允许；设计器会拆成两段）",
			Detail: []string{
				"元素：" + el.Name + " " + elementKey(el),
				"请在代码里改成字符串拼接或转义，不要依赖 CDATA 里的 ]]>",
			},
		}
	}
	if el.HasCDATA() {
		res := make([]byte, 0, len(raw)+len(content)-(el.CDATAEnd-el.CDATAStart))
		res = append(res, raw[:el.CDATAStart]...)
		res = append(res, content...)
		res = append(res, raw[el.CDATAEnd:]...)
		return res, nil
	}
	if !el.SelfClosing {
		// 形如 <point ...></point>：把内容塞进空元素体，不加额外空白。
		res := make([]byte, 0, len(raw)+len(content)+12)
		res = append(res, raw[:el.TagEnd]...)
		res = append(res, []byte("<![CDATA[")...)
		res = append(res, content...)
		res = append(res, []byte("]]>")...)
		res = append(res, raw[el.TagEnd:]...)
		return res, nil
	}
	// 自闭合 → 展开。用模板（同包内已有带 CDATA 的同类元素）继承前置/后置字节。
	openTag := raw[el.TagStart : el.TagEnd-2] // 去掉 "/>"
	pre, post := []byte("\r\n<![CDATA["), []byte("]]>\r\n")
	if template != nil && template.HasCDATA() {
		pre = raw[template.TagEnd:template.CDATAStart]
		post = raw[template.CDATAEnd:template.End]
		// 模板的 post 含 `]]>` + 换行 + 缩进 + 闭合标签；重建时闭合标签用本元素名。
		post = rewriteClosingTag(post, template.Name, el.Name)
	}
	var b bytes.Buffer
	b.Write(openTag)
	b.WriteByte('>')
	b.Write(pre)
	b.Write(content)
	b.Write(post)
	res := make([]byte, 0, len(raw)-(el.End-el.Start)+b.Len())
	res = append(res, raw[:el.Start]...)
	res = append(res, b.Bytes()...)
	res = append(res, raw[el.End:]...)
	return res, nil
}

// rewriteClosingTag 把模板的尾部字节里的闭合标签名换成目标名。
func rewriteClosingTag(post []byte, from, to string) []byte {
	old := []byte("</" + from + ">")
	if !bytes.Contains(post, old) {
		return post
	}
	return bytes.ReplaceAll(post, old, []byte("</"+to+">"))
}

func elementKey(el *Element) string {
	if v, ok := el.Attr("name"); ok {
		return "name=" + v
	}
	if v, ok := el.Attr("id"); ok {
		return "id=" + v
	}
	return ""
}

// firstCDATATemplate 找同包内第一个带 CDATA 的同类元素，作为展开/新增的字节模板。
func firstCDATATemplate(d *Doc, name string) *Element {
	var list []*Element
	switch name {
	case "point":
		list = d.Points
	case "section":
		list = d.Sections
	}
	// 优先取最后一个（更接近当前写入风格），退而取第一个。
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].HasCDATA() {
			return list[i]
		}
	}
	return nil
}

func (o SetRootAttr) apply(raw []byte, d *Doc) ([]byte, error) {
	return setAttr(raw, d, d.Root, o.Key, o.Value)
}

func (o SetPointAttr) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Point(o.Name)
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到点，无法改属性", Detail: []string{o.Name}}
	}
	return setAttr(raw, d, el, o.Key, o.Value)
}

func (o SetPointCDATA) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Point(o.Name)
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到点，无法改写正文", Detail: []string{o.Name}}
	}
	return setCDATA(raw, d, el, o.Content, firstCDATATemplate(d, "point"))
}

func (o MarkDeleted) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Point(o.Name)
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到点，无法标记删除", Detail: []string{o.Name}}
	}
	return setAttr(raw, d, el, "status", "d")
}

func (o RenamePoint) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Point(o.From) // 第一个未删除元素
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到点，无法改名", Detail: []string{o.From}}
	}
	if d.Point(o.To) != nil {
		return nil, &FormatError{Msg: "TAP 里已存在目标点名，改名会撞名", Detail: []string{o.To}}
	}
	return setAttr(raw, d, el, "name", o.To)
}

func (o RemoveTombstones) apply(raw []byte, d *Doc) ([]byte, error) {
	var els []*Element
	for _, p := range d.PointsAll(o.Name) {
		if IsDeleted(p) { // 只删墓碑
			els = append(els, p)
		}
	}
	if len(els) == 0 {
		return raw, nil // 首次改名没有墓碑：no-op
	}
	// 从后往前删，避免偏移失效
	for i := len(els) - 1; i >= 0; i-- {
		el := els[i]
		if el.Name != "point" {
			continue
		}
		end := el.End
		if end < len(raw) && raw[end] == '\n' {
			end++
		}
		raw = append(append([]byte(nil), raw[:el.Start]...), raw[end:]...)
	}
	return raw, nil
}

func (o SetSectionCDATA) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Section(o.ID)
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到区段，无法改写正文", Detail: []string{o.ID}}
	}
	return setCDATA(raw, d, el, o.Content, firstCDATATemplate(d, "section"))
}

func (o SetSectionAttr) apply(raw []byte, d *Doc) ([]byte, error) {
	el := d.Section(o.ID)
	if el == nil {
		return nil, &FormatError{Msg: "TAP 里找不到区段，无法改属性", Detail: []string{o.ID}}
	}
	return setAttr(raw, d, el, o.Key, o.Value)
}

func (o AddPoint) apply(raw []byte, d *Doc) ([]byte, error) {
	name := o.Attrs["name"]
	if name == "" {
		return nil, &FormatError{Msg: "AddPoint 缺少 name"}
	}
	if d.Point(name) != nil {
		return nil, &FormatError{
			Msg:    "TAP 里已存在同名点，不能新增",
			Detail: []string{name, "应改用 SetPointCDATA"},
		}
	}
	if bytes.Contains(o.Content, []byte("]]>")) {
		return nil, &FormatError{Msg: "新增点内容含 ]]>，无法作为 CDATA 写入", Detail: []string{name}}
	}

	tpl := firstCDATATemplate(d, "point")
	var elem []byte
	if tpl != nil {
		// 用模板的开放标签骨架：先按模板属性顺序生成，再套用目标属性值。
		openTag := append([]byte(nil), raw[tpl.TagStart:tpl.TagEnd]...)
		pre := raw[tpl.TagEnd:tpl.CDATAStart]
		post := rewriteClosingTag(raw[tpl.CDATAEnd:tpl.End], "point", "point")
		elem = buildFromTemplate(openTag, o.Attrs, pre, o.Content, post)
	} else {
		elem = buildCanonicalPoint(o.Attrs, o.Content)
	}

	off, sep := d.AppendAnchor()
	res := make([]byte, 0, len(raw)+len(sep)+len(elem))
	res = append(res, raw[:off]...)
	res = append(res, sep...)
	res = append(res, elem...)
	res = append(res, raw[off:]...)
	return res, nil
}

// buildFromTemplate 用模板开放标签的属性顺序生成新元素字节。
// 模板里出现的属性按模板位置改写；模板里没有的属性追加；模板里有而目标没给的属性清空。
func buildFromTemplate(openTag []byte, attrs map[string]string, pre, content, post []byte) []byte {
	// 用一个临时 Doc 解析模板标签，复用 setAttr 的「有则改无则加」。
	tmp := &Doc{Raw: openTag}
	el := &Element{Name: "point", TagStart: 0, TagEnd: len(openTag)}
	if bytes.HasSuffix(openTag, []byte("/>")) {
		el.SelfClosing = true
	}
	_, parsed, _, _, err := parseOpenTag(openTag, 0)
	if err == nil {
		el.Attrs = parsed
	}
	cur := append([]byte(nil), openTag...)
	// 先按目标属性改/加。
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		cur, _ = setAttr(cur, tmp, el, k, attrs[k])
		el.TagEnd = len(cur)
		_, parsed, _, _, _ = parseOpenTag(cur, 0)
		el.Attrs = parsed
	}
	// 去掉自闭合斜杠：新元素一定有内容体。
	if bytes.HasSuffix(cur, []byte("/>")) {
		cur = append(cur[:len(cur)-2], '>')
	}
	var b bytes.Buffer
	b.Write(cur)
	b.Write(pre)
	b.Write(content)
	b.Write(post)
	return b.Bytes()
}

// buildCanonicalPoint 是无模板时的兜底：属性顺序取自设计器 ctor
// （AddPointModel.cs:64-75：name order ver cite_std new src status ch ind_fun ind_extra）。
func buildCanonicalPoint(attrs map[string]string, content []byte) []byte {
	order := []string{"name", "order", "ver", "cite_std", "new", "src", "status", "ch", "ind_fun", "ind_extra"}
	var b bytes.Buffer
	b.WriteString(`<point`)
	seen := map[string]bool{}
	for _, k := range order {
		if v, ok := attrs[k]; ok {
			b.WriteString(` ` + k + `="` + escapeAttrValue(v) + `"`)
			seen[k] = true
		}
	}
	rest := make([]string, 0)
	for k := range attrs {
		if !seen[k] {
			rest = append(rest, k)
		}
	}
	sortStrings(rest)
	for _, k := range rest {
		b.WriteString(` ` + k + `="` + escapeAttrValue(attrs[k]) + `"`)
	}
	b.WriteString(">\r\n<![CDATA[")
	b.Write(content)
	b.WriteString("]]>\r\n</point>")
	return b.Bytes()
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// Rewrite 依次施加 ops。每步都基于上一步的结果重新扫描：ops 数量小（通常 < 100），
// 每次重扫 155KB 量级毫秒级完成；用简单换正确，符合设计指南「性能无需优化」。
func Rewrite(orig []byte, ops ...Op) ([]byte, error) {
	buf := append([]byte(nil), orig...)
	for _, op := range ops {
		d, err := Parse(buf)
		if err != nil {
			return nil, err
		}
		buf, err = op.apply(buf, d)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op.describe(), err)
		}
	}
	return buf, nil
}
