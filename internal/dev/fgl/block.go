package fgl

import (
	"fmt"
	"regexp"
	"strings"
)

/* ============================================================================
 * 块信封（block envelope）
 *
 * 本文件只判定**块信封**是否良构，不做任何语句级语法检查：
 *   FUNCTION 头行 → 块体 → `END FUNCTION` 终止行
 * 目的：CLI 重写 T100 设计器包时，机械保证永不改坏函数的结构行（头行与终止行）。
 *
 * 与 outline.go 的关系：信封判定完全建立在 ParseOutline 的块树之上（不另写一套
 * 词法），所以掩码、注释、字符串、CRLF/孤立 CR 的行为与大纲解析**必然一致**。
 * ========================================================================== */

// Range 描述一段行区间与对应的字节区间。
//
// 行：1-based、闭区间 [StartLine, EndLine]（EndLine < StartLine 表示空区间）。
// 字节：半开区间 [StartByte, EndByte)。
type Range struct {
	StartLine, EndLine int
	StartByte, EndByte int
}

// Empty 报告该区间是否为空（StartLine > EndLine）。
func (r Range) Empty() bool { return r.StartLine > r.EndLine }

// Block 是模块级块的枚举结果。
type Block struct {
	// Kind 是本次请求的信封种类："FUNCTION" | "MAIN" | "DIALOG" | "REPORT"。
	Kind string
	// Name 是块的标签（沿大纲节点的 Label，**不额外加工**）：
	//   FUNCTION → 裸函数名，如 "capt110_apaauc002_ref"（不含 public/private、不含括号）
	//   MAIN     → "MAIN"
	//   REPORT   → "REPORT rep_fmt"（声明式）/ "REPORT"（无名字时）
	//   DIALOG   → "DIALOG dlg_a"（声明式）/ "DIALOG"（过程式）
	Name  string
	Scope string // "PUBLIC" | "PRIVATE" | ""（头行没写限定符时为空）

	Header     Range // 只有块头那一行
	Body       Range // 头行与终止行之间的行；**允许为空**（StartLine > EndLine）
	Terminator Range // 只有 `END <Kind>` 那一行

	// NameSpan 是块名在源码里的落点：Line 为 1-based（与 Range 一致），
	// StartCol/EndCol 为 0-based、半开区间的**字节列**（Go 正则子匹配就是字节下标）。
	NameSpan struct{ Line, StartCol, EndCol int }
}

/* 字节范围的取值口径（**约定，必须一致**）：
 *
 *   行的 StartByte = 该行首字节在原文中的偏移；
 *   行的 EndByte   = 该行**内容**末尾（不含行尾 EOL 字节）；
 *   最后一行没有 EOL 时 EndByte == len(text)。
 *
 * 也就是说 EndByte **不含**行尾的 \r\n / \n / \r。这样做的理由：范围是给「替换整行」
 * 用的，EOL 由调用方按原样保留（红线：不改 EOL 风格），把它含进来只会让拼接处
 * 多出/少掉一个换行。 */

// ParseError 是信封判定的失败结果。ExitCode 固定 3（验证失败）。
type ParseError struct {
	Code     string
	AtLine   int // 1-based；0 表示与具体行无关
	Expected string
	Found    string
}

func (e *ParseError) Error() string {
	var b strings.Builder
	b.WriteString("fgl: ")
	b.WriteString(e.Code)
	if e.AtLine > 0 {
		fmt.Fprintf(&b, "（第 %d 行）", e.AtLine)
	}
	if e.Expected != "" || e.Found != "" {
		b.WriteString("：期望 ")
		b.WriteString(e.Expected)
		b.WriteString("，实际 ")
		b.WriteString(e.Found)
	}
	return b.String()
}

// ExitCode = 退出码 3：验证失败（设计指南 §2）。
func (e *ParseError) ExitCode() int { return 3 }

// 错误码（设计契约，刻意收窄到**块信封良构性**，不做语句级语法校验）。
const (
	CodeEmpty              = "EMPTY"               // 文本为空（或只有空白）
	CodeNoFunctionHeader   = "NO_FUNCTION_HEADER"  // 一个块节点都没识别出来
	CodeMultipleBlocks     = "MULTIPLE_BLOCKS"     // 顶层块多于一个
	CodeNotAFunction       = "NOT_A_FUNCTION"      // 顶层块不是本次请求的信封种类
	CodeUnterminated       = "UNTERMINATED"        // 末尾那行压根没有 END
	CodeTerminatorMismatch = "TERMINATOR_MISMATCH" // 末尾是 END <别的关键字>

	// CodeInvalidKind 只有 **ParseBlock 的调用方传错 wantKind** 时才会出现，
	// ParseFunction 永远不会产生它（后者只接受 FUNCTION/MAIN）。
	// 单独给一个码是为了不把它和「文本的信封不对」混为一谈（那是 NOT_A_FUNCTION）。
	CodeInvalidKind = "INVALID_KIND"
)

var (
	// 终止行：`^\s*END\s+(\w+)`（以 END 开头）。
	reEndStrict = regexp.MustCompile(`(?i)^\s*END\s+(\w+)`)
	// 同一行里任意位置的 `END <关键字>`：用于「同行开闭」`FUNCTION f() END FUNCTION`。
	reEndAnywhere = regexp.MustCompile(`(?i)\bEND\s+(\w+)`)
	// 头行的限定符。
	reScope = regexp.MustCompile(`(?i)^[ \t]*(PUBLIC|PRIVATE)(?:[ \t]|$)`)
)

/* lineTable 是行号 ↔ 字节偏移的前缀表。
 *
 * 断行模型与 splitLines 完全相同（/\r\n|\r|\n/）：**不规范化**输入文本，
 * 行号必须与原始文本的行网格一致（调用方若需要全局偏移，自行相加）。 */
type lineTable struct {
	starts []int // 每行首字节偏移
	ends   []int // 每行内容末尾偏移（不含 EOL）；最后一行无 EOL 时 == len(text)
	lines  []string
}

func buildLineTable(text string) *lineTable {
	n := strings.Count(text, "\n") + strings.Count(text, "\r") + 1
	t := &lineTable{
		starts: make([]int, 0, n),
		ends:   make([]int, 0, n),
		lines:  make([]string, 0, n),
	}
	start := 0
	for i := 0; i < len(text); {
		switch text[i] {
		case '\r':
			t.lines = append(t.lines, text[start:i])
			t.starts = append(t.starts, start)
			t.ends = append(t.ends, i)
			if i+1 < len(text) && text[i+1] == '\n' {
				i += 2
			} else {
				i++
			}
			start = i
		case '\n':
			t.lines = append(t.lines, text[start:i])
			t.starts = append(t.starts, start)
			t.ends = append(t.ends, i)
			i++
			start = i
		default:
			i++
		}
	}
	t.lines = append(t.lines, text[start:])
	t.starts = append(t.starts, start)
	t.ends = append(t.ends, len(text))
	return t
}

func (t *lineTable) count() int { return len(t.lines) }

// line 取 1-based 行内容；越界返回 ""。
func (t *lineTable) line(n int) string {
	if n < 1 || n > len(t.lines) {
		return ""
	}
	return t.lines[n-1]
}

func (t *lineTable) lineRange(n int) Range {
	if n < 1 || n > len(t.lines) {
		return Range{StartLine: n, EndLine: n, StartByte: 0, EndByte: 0}
	}
	return Range{StartLine: n, EndLine: n, StartByte: t.starts[n-1], EndByte: t.ends[n-1]}
}

/*
parseBlockEnvelope 是 ParseFunction / ParseBlock 共用的信封判定内核。

want 是**本次接受的信封种类集合**（大写），例如

	ParseFunction → {"FUNCTION", "MAIN"}
	ParseBlock(k) → {k}

输入契约：每次调用只应包含恰好一个块。文本可以是纯 CRLF / 纯 LF / 混合 EOL，
可以没有结尾换行，也可以从半截内容开始（典型场景是 TAP `<point>` 的 CDATA 正文：
同一行前面可能还压着 `<![CDATA[` 前缀）。**不规范化输入文本**，行号一律对原始文本
的行网格（splitLines 的 /\r\n|\r|\n/ 模型）计数。

检查顺序（每一步都可能直接返回）：
 1. 空文本                          → EMPTY
 2. 顶层块数为 0                    → NO_FUNCTION_HEADER
 3. 顶层块数 > 1                    → MULTIPLE_BLOCKS
 4. 顶层块种类不在 want 里          → NOT_A_FUNCTION（Found = 实际种类）
 5. 终止符：先看顶层块的 EndLine 那一行里有没有 `\bEND\s+(\w+)`
    （不必在行首 —— 覆盖同行开闭 `FUNCTION f() END FUNCTION`）；
    关键字不是本块的种类 → TERMINATOR_MISMATCH。
    那一行里没有 END（说明块是被 EOF 兜底收尾的），就在 [头行, EndLine] 里自后向前找
    一条以 `END <kw>` 开头的**掩码**行（注释掉的 END 不算）：找到 → TERMINATOR_MISMATCH；
    一条都没有 → UNTERMINATED。

注意第 4 步先于第 5 步：所以 `FUNCTION f() / END DIALOG` 在 want=FUNCTION 时得到
TERMINATOR_MISMATCH（头行是 FUNCTION，种类对得上），而 `DIALOG d() / END DIALOG`
在 want=FUNCTION 时得到 NOT_A_FUNCTION（种类就先不对）。这两条都有用例钉住。
*/
func parseBlockEnvelope(text string, want []string) (*Block, *ParseError) {
	wantText := strings.Join(want, " 或 ")

	if strings.TrimSpace(text) == "" {
		return nil, &ParseError{
			Code:     CodeEmpty,
			AtLine:   0,
			Expected: "至少一个 " + wantText + " 块",
			Found:    "空文本（或只有空白）",
		}
	}

	nodes, code := parseOutlineMasked(text)

	if len(nodes) == 0 {
		return nil, &ParseError{
			Code:     CodeNoFunctionHeader,
			AtLine:   1,
			Expected: wantText + " 的块头",
			Found:    firstContentLine(text),
		}
	}

	if len(nodes) > 1 {
		second := nodes[1]
		return nil, &ParseError{
			Code:     CodeMultipleBlocks,
			AtLine:   second.Line,
			Expected: "恰好一个顶层块",
			Found: fmt.Sprintf("%d 个顶层块（第 2 个是第 %d 行的 %s）",
				len(nodes), second.Line, second.Kind),
		}
	}

	top := nodes[0]
	if !kindIn(string(top.Kind), want) {
		return nil, &ParseError{
			Code:     CodeNotAFunction,
			AtLine:   top.Line,
			Expected: wantText,
			Found:    string(top.Kind),
		}
	}

	lt := buildLineTable(text)
	headerLine := top.Line
	termLine := top.EndLine
	if termLine < 1 {
		termLine = 1
	}
	if n := lt.count(); termLine > n {
		termLine = n
	}
	if headerLine < 1 {
		headerLine = 1
	}
	if headerLine > lt.count() {
		headerLine = lt.count()
	}

	// 期望的终止关键字 = 顶层节点自己的种类。第 4 步已经保证它一定在 want 里，
	// 所以 `END <wantKind>` 与 `END <top.Kind>` 在这里是同一件事。
	wantKw := string(top.Kind) // "FUNCTION" | "MAIN" | "DIALOG" | "REPORT"

	/* 终止符判定 —— 这是本文件唯一需要绕开大纲终局结果的地方。
	 *
	 * 大纲解析里 RE_END 的自上而下配对「绝不猜」：`END DIALOG` 收不了 FUNCTION 帧，
	 * 于是该帧一直挂到 EOF，被 closeTopRepair 拉到**文本末行**。所以对一个
	 * `FUNCTION … / END DIALOG` 的文本，top.EndLine 指向的是末行（常常是空行），
	 * 而不是那行 `END DIALOG`。判信封时必须把这一点补回来：
	 *
	 *  1. 终止行（top.EndLine）里找 `\bEND\s+(\w+)`（不必在行首 —— 覆盖同行开闭
	 *     `FUNCTION f() END FUNCTION`）。找到且关键字相符 → 良构；不符 → 终止符不匹配。
	 *  2. 终止行里没有 END：块是被 EOF 修复收尾的。在 [H, EndLine] 里**自后向前**找
	 *     一条以 `END <kw>` 开头的掩码行（用掩码行，注释掉的 END 不算）。
	 *     找到 → 作者显然写了 END、只是关键字不对 → 终止符不匹配；
	 *     一条都没有 → 未终止。
	 */
	termLineIdx := termLine - 1
	if m := reEndAnywhere.FindStringSubmatch(code[termLineIdx]); m != nil {
		if !strings.EqualFold(m[1], wantKw) {
			return nil, &ParseError{
				Code:     CodeTerminatorMismatch,
				AtLine:   termLine,
				Expected: "END " + wantKw,
				Found:    "END " + strings.ToUpper(m[1]),
			}
		}
	} else {
		mismatchAt, mismatchKw := 0, ""
		for i := termLineIdx - 1; i >= headerLine-1; i-- {
			if i < 0 || i >= len(code) {
				break
			}
			if m := reEndStrict.FindStringSubmatch(code[i]); m != nil {
				mismatchAt, mismatchKw = i+1, m[1]
				break
			}
		}
		if mismatchAt != 0 {
			return nil, &ParseError{
				Code:     CodeTerminatorMismatch,
				AtLine:   mismatchAt,
				Expected: "END " + wantKw,
				Found:    "END " + strings.ToUpper(mismatchKw),
			}
		}
		// 报「最后一条有内容的行」，比报 EOF 处的空行有用。
		at := headerLine
		for i := termLine - 1; i >= headerLine-1; i-- {
			if i >= 0 && i < len(code) && strings.TrimSpace(code[i]) != "" {
				at = i + 1
				break
			}
		}
		return nil, &ParseError{
			Code:     CodeUnterminated,
			AtLine:   at,
			Expected: "END " + wantKw,
			Found:    fmt.Sprintf("第 %d 行起没有 END %s（块在文本结束处仍未收尾）", headerLine, wantKw),
		}
	}

	b := &Block{
		Kind:       wantKw,
		Name:       top.Label,
		Scope:      scopeOf(lt.line(headerLine)),
		Header:     lt.lineRange(headerLine),
		Terminator: lt.lineRange(termLine),
	}
	// Body：头行 H、终止行 T → [H+1, T-1]。T <= H+1 时为空（允许，不是错误）；
	// 空体时两个字节边界都取头行的 EndByte（= 空体紧贴头行之后）。
	if bs, be := headerLine+1, termLine-1; bs > be {
		b.Body = Range{StartLine: bs, EndLine: be, StartByte: b.Header.EndByte, EndByte: b.Header.EndByte}
	} else {
		b.Body = Range{
			StartLine: bs,
			EndLine:   be,
			StartByte: lt.starts[bs-1],
			EndByte:   lt.ends[be-1],
		}
	}
	b.NameSpan = struct{ Line, StartCol, EndCol int }{headerLine, top.SelStart, top.SelEnd}
	return b, nil
}

// kindIn 判断大纲节点种类是否落在允许集合里（均为大写枚举字符串）。
func kindIn(kind string, want []string) bool {
	for _, w := range want {
		if kind == w {
			return true
		}
	}
	return false
}

// envelopeKinds 是 ParseBlock 认识的信封种类。
var envelopeKinds = []string{"FUNCTION", "MAIN", "DIALOG", "REPORT"}

/*
ParseFunction 解析**一个函数块**的文本，判定信封是否良构。

等价于 ParseBlock(text, "FUNCTION")，但**额外接受 MAIN** —— 保持历史行为不变：
T100 的 `function.` 点里出现过 `MAIN` 形式的正文，之前的调用方依赖它被接受。
除此之外两者完全一致（错误码、检查顺序、Range 口径都相同）。
*/
func ParseFunction(text string) (*Block, *ParseError) {
	return parseBlockEnvelope(text, []string{"FUNCTION", "MAIN"})
}

/*
ParseBlock 解析**一个块信封**的文本，判定它是否是指定种类的良构信封。

wantKind ∈ {"FUNCTION","MAIN","DIALOG","REPORT"}（大小写不敏感，首尾空白忽略）。
需要它是因为真实语料里的 `dialog.` / `report.` 点装的是同样形状的信封
（`DIALOG x() … END DIALOG`、`REPORT r(x) … END REPORT`）：调用方要把它们
和函数块一样处理（头行结构性、块体可编辑、`END <KIND>` 行结构性），
所以需要一个能声明「我要的是哪种信封」的入口。

wantKind 不是上面四种时 → `INVALID_KIND`（调用方用错 API，不是文本的问题）。
文本顶层块存在但种类不匹配 → `NOT_A_FUNCTION`（**保留这个码名**：调用方用它表示
「文本的信封种类不是我请求的那种」，而不是字面上的「不是函数」）。
终止符要求是 `END <wantKind>`，不匹配 → `TERMINATOR_MISMATCH`。

返回的 Block.Kind 就是本次请求的（大写）种类，例如 "DIALOG"。
*/
func ParseBlock(text string, wantKind string) (*Block, *ParseError) {
	want := strings.ToUpper(strings.TrimSpace(wantKind))
	if !kindIn(want, envelopeKinds) {
		return nil, &ParseError{
			Code:     CodeInvalidKind,
			AtLine:   0,
			Expected: strings.Join(envelopeKinds, "、"),
			Found:    fmt.Sprintf("%q", wantKind),
		}
	}
	return parseBlockEnvelope(text, []string{want})
}

/*
EnvelopeKindFor 返回 TAP point 名字前缀所隐含的信封种类：

	function. → "FUNCTION"
	dialog.  → "DIALOG"
	report.  → "REPORT"

其余一律 ("", false) —— 包括 T100 里同级但形状不同的另一些组：`main.`、`input.`、
`construct.`、`menu.`、`global.` 等，它们的 CDATA 不是「一个块信封」，
调用方拿这个返回值去 ParseBlock 也没有意义。

比对大小写不敏感，且**必须带点**，所以 `functions.xxx` / `dialogs.xxx` 不会被误认。
*/
func EnvelopeKindFor(pointName string) (string, bool) {
	lower := strings.ToLower(pointName)
	switch {
	case strings.HasPrefix(lower, "function."):
		return "FUNCTION", true
	case strings.HasPrefix(lower, "dialog."):
		return "DIALOG", true
	case strings.HasPrefix(lower, "report."):
		return "REPORT", true
	}
	return "", false
}

// scopeOf 从块头行取 PUBLIC/PRIVATE 限定符（大小写不敏感，输出大写）。
func scopeOf(header string) string {
	if m := reScope.FindStringSubmatch(header); m != nil {
		return strings.ToUpper(m[1])
	}
	return ""
}

// firstContentLine 取第一条非空行（去掉首尾空白，超长截断），用于报错时给出线索。
func firstContentLine(text string) string {
	for _, ln := range splitLines(text) {
		if s := strings.TrimSpace(ln); s != "" {
			return clip(s, 60)
		}
	}
	return "（没有非空行）"
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
