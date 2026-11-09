// Package tglfile 处理 .tgl 框架骨架里的标记：区段边界、插入点占位符、集合锚点。
//
// 依据：
//   - 反编译源码（标记正则、合成管线、GenerateTGL、.tgl 的真身）：
//     CodeEditWindow/Helper/CodeEditorManager.cs:1694-1715（四个正则原文）
//     CodeEditorManager.cs:314-352（GenerateTGL 的补丁格式）
//     SpecDesignerCommon/TzpManager.cs:453-457（LoadCodeFile 对 TGL 做一次 TrimEnd）
package tglfile

import (
	"bytes"
	"fmt"
	"regexp"
)

// ExitCodeError 让 CLI 把错误映射为退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// FormatError = 退出码 2。
type FormatError struct {
	Msg    string
	Detail []string
}

func (e *FormatError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + join(e.Detail, "; ")
}

func (e *FormatError) ExitCode() int { return 2 }

func join(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// 正则原文抄自 CodeEditorManager.cs:1694-1715。
var (
	// sectionStartPattern = ({<section\s+id="(?<name>\S*)".*>})
	reSectionStart = regexp.MustCompile(`(?i)(\{<section\s+id="(\S*)".*>\})`)
	// sectionEndPattern = ({</section>})
	reSectionEnd = regexp.MustCompile(`(?i)(\{</section>\})`)
	// addPointRex = {<point\s+name="(\S+)".*\s*/>}   —— 设计器里是**大小写敏感**、无选项
	rePlaceholder = regexp.MustCompile(`\{<point\s+name="(\S+)".*\s*/>\}`)
	// sectionReadOnlyPattern = readonly="(?<value>\w)"
	reReadonly = regexp.MustCompile(`(?i)readonly="(\w)"`)
)

// Marker 是一处标记的字节区间。
type Marker struct {
	Name  string
	Start int
	End   int
	Raw   []byte
}

// Section 是一对 {<section …>} … {</section>} 标记。
type Section struct {
	ID          string
	StartMarker Marker
	EndMarker   Marker
	// Start/End 覆盖两个标记本身；BodyStart/BodyEnd 是两个标记之间的正文。
	Start, End         int
	BodyStart, BodyEnd int
	ReadonlyMarker     bool
}

// TrimEnd 复刻 TzpManager.LoadCodeFile 的 content.TrimEnd(new char[0])：
// 去掉**所有**尾部空白（空格/制表/CR/LF）。这一步只做一次，且在合成之前。
func TrimEnd(b []byte) []byte {
	i := len(b)
	for i > 0 {
		switch b[i-1] {
		case ' ', '\t', '\r', '\n', '\v', '\f':
			i--
			continue
		}
		break
	}
	return b[:i]
}

// FindSections 按序号配对区段标记（设计器 ProcessSections 就是按序号配对，不支持嵌套）。
// 起止数量不等 → FormatError（设计器会抛「区段错误」）。
func FindSections(tgl []byte) ([]*Section, error) {
	starts := reSectionStart.FindAllSubmatchIndex(tgl, -1)
	ends := reSectionEnd.FindAllSubmatchIndex(tgl, -1)
	if len(starts) != len(ends) {
		return nil, &FormatError{
			Msg: "TGL 区段标记不配对（设计器会抛「区段错误」）",
			Detail: []string{
				fmt.Sprintf("{<section ...>} 出现 %d 次", len(starts)),
				fmt.Sprintf("{</section>} 出现 %d 次", len(ends)),
			},
		}
	}
	out := make([]*Section, 0, len(starts))
	for i := range starts {
		sm := starts[i]
		em := ends[i]
		id := string(tgl[sm[4]:sm[5]])
		s := &Section{
			ID:        id,
			Start:     sm[0],
			End:       em[1],
			BodyStart: sm[1],
			BodyEnd:   em[0],
			StartMarker: Marker{
				Name:  id,
				Start: sm[0],
				End:   sm[1],
				Raw:   append([]byte(nil), tgl[sm[0]:sm[1]]...),
			},
			EndMarker: Marker{
				Start: em[0],
				End:   em[1],
				Raw:   append([]byte(nil), tgl[em[0]:em[1]]...),
			},
		}
		if m := reReadonly.FindSubmatch(s.StartMarker.Raw); m != nil {
			s.ReadonlyMarker = bytes.EqualFold(m[1], []byte("Y"))
		}
		out = append(out, s)
	}
	return out, nil
}

// FindPlaceholders 返回 TGL 里所有插入点占位符（设计器 addPointRex）。
func FindPlaceholders(tgl []byte) []Marker {
	var out []Marker
	for _, m := range rePlaceholder.FindAllSubmatchIndex(tgl, -1) {
		out = append(out, Marker{
			Name:  string(tgl[m[2]:m[3]]),
			Start: m[0],
			End:   m[1],
			Raw:   append([]byte(nil), tgl[m[0]:m[1]]...),
		})
	}
	return out
}

// AnchorKind 是集合锚点的三种类型。
type AnchorKind string

const (
	AnchorFunction AnchorKind = "function"
	AnchorDialog   AnchorKind = "dialog"
	AnchorReport   AnchorKind = "report"
)

// anchorRe 复刻 ProcessFunctionTypes 的三个正则：
// {<point\s+name="(other.function)".*\s*/>}，IgnoreCase|Multiline。
// 注意捕获组里的 '.' 是「任意字符」，所以 other_function / otherXfunction 也会命中
// ——这是设计器的真实行为，照抄。
func anchorRe(k AnchorKind) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(\{<point\s+name="(other.` + string(k) + `)".*\s*/>\})`)
}

// FindAnchor 找集合锚点（整行标记）。
func FindAnchor(tgl []byte, k AnchorKind) *Marker {
	re := anchorRe(k)
	m := re.FindSubmatchIndex(tgl)
	if m == nil {
		return nil
	}
	return &Marker{
		Name:  string(tgl[m[4]:m[5]]),
		Start: m[0],
		End:   m[1],
		Raw:   append([]byte(nil), tgl[m[0]:m[1]]...),
	}
}

// ReplaceAnchor 把锚点标记**全部出现**替换为 repl（设计器的 regex.Replace 替换所有出现）。
func ReplaceAnchor(tgl []byte, k AnchorKind, repl []byte) ([]byte, bool) {
	re := anchorRe(k)
	if !re.Match(tgl) {
		return tgl, false
	}
	// 用 ReplaceAllLiteral 避免 repl 里的 $ 被当作替换模板
	// （设计器直接用 repl 作模板，只有点名里出现 $ 才会出问题；点名不允许 $）。
	return re.ReplaceAll(tgl, repl), true
}

// AnchorSectionID 判断区段 id 是否为三个集合锚点之一，并返回类型。
func AnchorSectionID(prog, id string) (AnchorKind, bool) {
	switch id {
	case prog + ".other_function":
		return AnchorFunction, true
	case prog + ".other_dialog":
		return AnchorDialog, true
	case prog + ".other_report":
		return AnchorReport, true
	}
	return "", false
}

// PatchSection 复刻 GenerateTGL（CodeEditorManager.cs:314-352）的补丁格式：
//
//	"{0}{1}{2}{1}{3}" = startMarker + NewLine + content + NewLine + endMarker
//
// 设计器用 Environment.NewLine（Windows = CRLF）。这里保持一致，便于与 .tap 的
// 同名区段文本逐字节相等（README §3.13 硬性约束 6）。
func PatchSection(tgl []byte, id string, content []byte, eol []byte) ([]byte, error) {
	secs, err := FindSections(tgl)
	if err != nil {
		return nil, err
	}
	for _, s := range secs {
		if s.ID != id {
			continue
		}
		var b bytes.Buffer
		b.Grow(len(tgl) + len(content))
		b.Write(tgl[:s.Start])
		b.Write(s.StartMarker.Raw)
		b.Write(eol)
		b.Write(content)
		b.Write(eol)
		b.Write(s.EndMarker.Raw)
		b.Write(tgl[s.End:])
		return b.Bytes(), nil
	}
	return nil, &FormatError{Msg: "TGL 里找不到区段，无法打补丁", Detail: []string{id}}
}

// SectionBody 返回区段正文（两个标记之间的字节）。
func SectionBody(tgl []byte, s *Section) []byte { return tgl[s.BodyStart:s.BodyEnd] }
