package fgl

import (
	"strings"
	"testing"
)

/* ============================================================================
 * 块信封 API 的用例。
 *
 * 覆盖：6 个错误码、PUBLIC/PRIVATE、头行前的 # 横幅注释、CRLF/LF/混合/无结尾换行
 * 的行号一致性、空体、字节范围口径，以及**纯 CR** 行注释的回归（见 outline.go 里
 * maskLines 的行注释修正）。
 * ========================================================================== */

func TestParseFunctionErrors(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		wantCode   string
		wantAtLine int // 0 表示不校验
		wantFound  string
	}{
		{
			name:     "空字符串",
			text:     "",
			wantCode: CodeEmpty,
		},
		{
			name:     "只有空白（CRLF）",
			text:     "   \r\n\t\r\n",
			wantCode: CodeEmpty,
		},
		{
			name:       "只有一行注释 → 没有块头",
			text:       "# 只有注释，没有任何块\n",
			wantCode:   CodeNoFunctionHeader,
			wantAtLine: 1,
		},
		{
			name:       "只有空白与注释 → 没有块头",
			text:       "\r\n   \r\n# c\r\n",
			wantCode:   CodeNoFunctionHeader,
			wantAtLine: 1,
		},
		{
			name:       "两个函数拼接 → 多块",
			text:       "FUNCTION a()\n  LET x = 1\nEND FUNCTION\n\nFUNCTION b()\n  LET y = 2\nEND FUNCTION\n",
			wantCode:   CodeMultipleBlocks,
			wantAtLine: 5,
		},
		{
			name:       "声明式 DIALOG → 不是函数",
			text:       "DIALOG dlg_a()\n  INPUT BY NAME a\n  END INPUT\nEND DIALOG\n",
			wantCode:   CodeNotAFunction,
			wantAtLine: 1,
			wantFound:  "DIALOG",
		},
		{
			name:       "REPORT → 不是函数",
			text:       "REPORT r_rep(x)\n  FORMAT\n    ON EVERY ROW\n      PRINT x\nEND REPORT\n",
			wantCode:   CodeNotAFunction,
			wantAtLine: 1,
			wantFound:  "REPORT",
		},
		{
			name:       "没有 END → 未终止",
			text:       "FUNCTION a()\n  LET x = 1\n",
			wantCode:   CodeUnterminated,
			wantAtLine: 2,
		},
		{
			name:       "只有头行 → 未终止",
			text:       "FUNCTION a()",
			wantCode:   CodeUnterminated,
			wantAtLine: 1,
		},
		{
			name:       "END 后是别的关键字 → 终止符不匹配",
			text:       "FUNCTION a()\n  LET x = 1\nEND DIALOG\n",
			wantCode:   CodeTerminatorMismatch,
			wantAtLine: 3,
			wantFound:  "END DIALOG",
		},
		{
			name:       "MAIN 被 END FUNCTION 收尾 → 终止符不匹配",
			text:       "MAIN\n  DISPLAY \"x\"\nEND FUNCTION\n",
			wantCode:   CodeTerminatorMismatch,
			wantAtLine: 3,
			wantFound:  "END FUNCTION",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, perr := ParseFunction(tc.text)
			if perr == nil {
				t.Fatalf("期望错误码 %s，实际成功（block=%+v）", tc.wantCode, b)
			}
			if perr.Code != tc.wantCode {
				t.Fatalf("错误码 = %s，期望 %s（%s）", perr.Code, tc.wantCode, perr.Error())
			}
			if tc.wantAtLine != 0 && perr.AtLine != tc.wantAtLine {
				t.Errorf("AtLine = %d，期望 %d", perr.AtLine, tc.wantAtLine)
			}
			if tc.wantFound != "" && !strings.Contains(perr.Found, tc.wantFound) {
				t.Errorf("Found = %q，期望包含 %q", perr.Found, tc.wantFound)
			}
			if perr.ExitCode() != 3 {
				t.Errorf("ExitCode = %d，期望 3", perr.ExitCode())
			}
			if perr.Error() == "" {
				t.Error("Error() 为空")
			}
		})
	}
}

// TestParseFunctionScope 覆盖 PUBLIC / PRIVATE / 无限定符，以及头行前的 # 横幅注释。
func TestParseFunctionScope(t *testing.T) {
	const banner = "################################################################################\n" +
		"# Descriptions...: 清空&給預設 #150127-00007#1\n" +
		"# Memo...........: it's a banner with ' apostrophes and \" quotes\n" +
		"# Usage..........: CALL capt110_apaauc002_ref()\n" +
		"################################################################################\n"

	cases := []struct {
		name      string
		text      string
		wantKind  string
		wantName  string
		wantScope string
		wantHdr   int
		wantTerm  int
	}{
		{
			name:      "PUBLIC 带横幅注释",
			text:      banner + "PUBLIC FUNCTION capt110_apaauc002_ref()\n  LET x = 1\nEND FUNCTION\n",
			wantKind:  "FUNCTION",
			wantName:  "capt110_apaauc002_ref",
			wantScope: "PUBLIC",
			wantHdr:   6,
			wantTerm:  8,
		},
		{
			name:      "PRIVATE 无横幅",
			text:      "PRIVATE FUNCTION p_ref()\nEND FUNCTION\n",
			wantKind:  "FUNCTION",
			wantName:  "p_ref",
			wantScope: "PRIVATE",
			wantHdr:   1,
			wantTerm:  2,
		},
		{
			name:      "小写限定符 → 归一为大写",
			text:      "  private FUNCTION q_ref()\nEND FUNCTION\n",
			wantKind:  "FUNCTION",
			wantName:  "q_ref",
			wantScope: "PRIVATE",
			wantHdr:   1,
			wantTerm:  2,
		},
		{
			name:      "无限定符",
			text:      "FUNCTION r_ref()\nEND FUNCTION\n",
			wantKind:  "FUNCTION",
			wantName:  "r_ref",
			wantScope: "",
			wantHdr:   1,
			wantTerm:  2,
		},
		{
			name:      "MAIN",
			text:      "MAIN\n  DISPLAY \"x\"\nEND MAIN\n",
			wantKind:  "MAIN",
			wantName:  "MAIN",
			wantScope: "",
			wantHdr:   1,
			wantTerm:  3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, perr := ParseFunction(tc.text)
			if perr != nil {
				t.Fatalf("不该失败: %s", perr.Error())
			}
			if b.Kind != tc.wantKind {
				t.Errorf("Kind = %q，期望 %q", b.Kind, tc.wantKind)
			}
			if b.Name != tc.wantName {
				t.Errorf("Name = %q，期望 %q", b.Name, tc.wantName)
			}
			if b.Scope != tc.wantScope {
				t.Errorf("Scope = %q，期望 %q", b.Scope, tc.wantScope)
			}
			if b.Header.StartLine != tc.wantHdr || b.Header.EndLine != tc.wantHdr {
				t.Errorf("Header 行 = [%d,%d]，期望 [%d,%d]", b.Header.StartLine, b.Header.EndLine, tc.wantHdr, tc.wantHdr)
			}
			if b.Terminator.StartLine != tc.wantTerm || b.Terminator.EndLine != tc.wantTerm {
				t.Errorf("Terminator 行 = [%d,%d]，期望 [%d,%d]",
					b.Terminator.StartLine, b.Terminator.EndLine, tc.wantTerm, tc.wantTerm)
			}
			if tc.wantName != "MAIN" && b.NameSpan.Line != tc.wantHdr {
				t.Errorf("NameSpan.Line = %d，期望 %d", b.NameSpan.Line, tc.wantHdr)
			}
		})
	}
}

// TestParseBlockEnvelopeKinds 覆盖 ParseBlock 的四种信封种类与几条交叉用例。
//
// 检查顺序是「先种类、后终止符」，所以：
//   - want=FUNCTION 遇到 `DIALOG d() / END DIALOG` → NOT_A_FUNCTION（种类就先不对）
//   - want=FUNCTION 遇到 `FUNCTION f() / END DIALOG` → TERMINATOR_MISMATCH（种类对得上）
func TestParseBlockEnvelopeKinds(t *testing.T) {
	const dialogSrc = "DIALOG dlg_a()\n  INPUT BY NAME a\n  END INPUT\nEND DIALOG\n"
	const reportSrc = "REPORT rep_fmt(cust_num)\n  DEFINE cust_num INTEGER\n  FORMAT EVERY ROW\n    PRINT cust_num\nEND REPORT\n"

	cases := []struct {
		name          string
		text          string
		wantKind      string
		wantOK        bool
		wantCode      string
		wantKindField string
		wantName      string
	}{
		{
			name:          "DIALOG 信封 ok（want=DIALOG）",
			text:          dialogSrc,
			wantKind:      "DIALOG",
			wantOK:        true,
			wantKindField: "DIALOG",
			wantName:      "DIALOG dlg_a",
		},
		{
			name:          "DIALOG 信封 ok（wantKind 小写 + 首尾空白）",
			text:          dialogSrc,
			wantKind:      "  dialog  ",
			wantOK:        true,
			wantKindField: "DIALOG",
			wantName:      "DIALOG dlg_a",
		},
		{
			name:          "REPORT 信封 ok（want=REPORT）",
			text:          reportSrc,
			wantKind:      "REPORT",
			wantOK:        true,
			wantKindField: "REPORT",
			wantName:      "REPORT rep_fmt",
		},
		{
			name:          "顶层裸 DIALOG 信封 ok",
			text:          "DIALOG\n  INPUT BY NAME a\n  END INPUT\nEND DIALOG\n",
			wantKind:      "DIALOG",
			wantOK:        true,
			wantKindField: "DIALOG",
			wantName:      "DIALOG",
		},
		{
			name:     "拿 FUNCTION 去要 DIALOG 信封 → NOT_A_FUNCTION",
			text:     dialogSrc,
			wantKind: "FUNCTION",
			wantOK:   false,
			wantCode: CodeNotAFunction,
		},
		{
			name:     "拿 FUNCTION 去要 REPORT 信封 → NOT_A_FUNCTION",
			text:     reportSrc,
			wantKind: "FUNCTION",
			wantOK:   false,
			wantCode: CodeNotAFunction,
		},
		{
			name:     "拿 DIALOG 去要 FUNCTION 信封 → NOT_A_FUNCTION",
			text:     "FUNCTION f()\nEND FUNCTION\n",
			wantKind: "DIALOG",
			wantOK:   false,
			wantCode: CodeNotAFunction,
		},
		{
			// 头行种类对得上，坏在终止行 —— 这时是 TERMINATOR_MISMATCH，不是 NOT_A_FUNCTION。
			name:     "want=FUNCTION 但收尾是 END DIALOG → TERMINATOR_MISMATCH",
			text:     "FUNCTION f()\n  LET x = 1\nEND DIALOG\n",
			wantKind: "FUNCTION",
			wantOK:   false,
			wantCode: CodeTerminatorMismatch,
		},
		{
			name:     "want=DIALOG 但收尾是 END REPORT → TERMINATOR_MISMATCH",
			text:     "DIALOG dlg_a()\n  INPUT BY NAME a\n  END INPUT\nEND REPORT\n",
			wantKind: "DIALOG",
			wantOK:   false,
			wantCode: CodeTerminatorMismatch,
		},
		{
			name:     "want=REPORT 但收尾是 END DIALOG → TERMINATOR_MISMATCH",
			text:     "REPORT rep_fmt(cust_num)\n  FORMAT EVERY ROW\n    PRINT cust_num\nEND DIALOG\n",
			wantKind: "REPORT",
			wantOK:   false,
			wantCode: CodeTerminatorMismatch,
		},
		{
			name:     "want=DIALOG 但文本里根本没有块 → NO_FUNCTION_HEADER",
			text:     "# 只有注释\n",
			wantKind: "DIALOG",
			wantOK:   false,
			wantCode: CodeNoFunctionHeader,
		},
		{
			name:     "want=DIALOG 但文本为空 → EMPTY",
			text:     "  \r\n",
			wantKind: "DIALOG",
			wantOK:   false,
			wantCode: CodeEmpty,
		},
		{
			name:     "want=DIALOG 但有两个块 → MULTIPLE_BLOCKS",
			text:     dialogSrc + "\n" + dialogSrc,
			wantKind: "DIALOG",
			wantOK:   false,
			wantCode: CodeMultipleBlocks,
		},
		{
			name:     "wantKind 不认识（MENU）→ INVALID_KIND",
			text:     dialogSrc,
			wantKind: "MENU",
			wantOK:   false,
			wantCode: CodeInvalidKind,
		},
		{
			name:     "wantKind 为空 → INVALID_KIND",
			text:     dialogSrc,
			wantKind: "",
			wantOK:   false,
			wantCode: CodeInvalidKind,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, perr := ParseBlock(tc.text, tc.wantKind)
			if tc.wantOK {
				if perr != nil {
					t.Fatalf("不该失败: %s", perr.Error())
				}
				if b.Kind != tc.wantKindField {
					t.Errorf("Kind = %q，期望 %q", b.Kind, tc.wantKindField)
				}
				if b.Name != tc.wantName {
					t.Errorf("Name = %q，期望 %q", b.Name, tc.wantName)
				}
				if b.Header.StartLine < 1 || b.Terminator.StartLine <= b.Header.StartLine {
					t.Errorf("Header/Terminator 行不对: %+v / %+v", b.Header, b.Terminator)
				}
				if b.Body.StartLine != b.Header.StartLine+1 || b.Body.EndLine != b.Terminator.StartLine-1 {
					t.Errorf("Body = %+v，期望 [%d,%d]",
						b.Body, b.Header.StartLine+1, b.Terminator.StartLine-1)
				}
				if got := tc.text[b.Header.StartByte:b.Header.EndByte]; !strings.Contains(got, b.Kind) {
					t.Errorf("Header 切片 = %q，期望含 %s", got, b.Kind)
				}
				if got := tc.text[b.Terminator.StartByte:b.Terminator.EndByte]; !strings.HasSuffix(strings.TrimSpace(got), b.Kind) {
					t.Errorf("Terminator 切片 = %q，期望以 %s 收尾", got, b.Kind)
				}
				return
			}
			if perr == nil {
				t.Fatalf("期望错误码 %s，实际成功（block=%+v）", tc.wantCode, b)
			}
			if perr.Code != tc.wantCode {
				t.Fatalf("错误码 = %s，期望 %s（%s）", perr.Code, tc.wantCode, perr.Error())
			}
			if perr.ExitCode() != 3 {
				t.Errorf("ExitCode = %d，期望 3", perr.ExitCode())
			}
		})
	}
}

// TestParseFunctionEqualsParseBlock 钉住「ParseFunction = ParseBlock("FUNCTION") 外加接受 MAIN」。
func TestParseFunctionEqualsParseBlock(t *testing.T) {
	texts := []string{
		"FUNCTION f()\n  LET x = 1\nEND FUNCTION\n",
		"PRIVATE FUNCTION g()\nEND FUNCTION\n",
		"MAIN\n  DISPLAY \"x\"\nEND MAIN\n",
		"FUNCTION f()\n  LET x = 1\nEND DIALOG\n",
		"DIALOG d()\nEND DIALOG\n",
		"# 只有注释\n",
		"",
		"FUNCTION a()\nEND FUNCTION\nFUNCTION b()\nEND FUNCTION\n",
	}
	for _, text := range texts {
		fnBlock, fnErr := ParseFunction(text)
		pbBlock, pbErr := ParseBlock(text, "FUNCTION")
		if (fnErr == nil) != (pbErr == nil) {
			// MAIN 是唯一的有意例外，单独在下面断言。
			if !strings.Contains(text, "MAIN") {
				t.Errorf("%q: ParseFunction 与 ParseBlock(FUNCTION) 成败不一致：%v / %v", text, fnErr, pbErr)
			}
			continue
		}
		if fnErr != nil {
			if fnErr.Code != pbErr.Code {
				t.Errorf("%q: 错误码不一致 %s / %s", text, fnErr.Code, pbErr.Code)
			}
			continue
		}
		if *fnBlock != *pbBlock {
			t.Errorf("%q: Block 不一致\n fn=%+v\n pb=%+v", text, *fnBlock, *pbBlock)
		}
	}
	// 唯一的例外：MAIN 只有 ParseFunction 接受。
	if b, err := ParseFunction("MAIN\n  DISPLAY \"x\"\nEND MAIN\n"); err != nil {
		t.Errorf("ParseFunction 必须继续接受 MAIN: %s", err.Error())
	} else if b.Kind != "MAIN" {
		t.Errorf("ParseFunction(MAIN).Kind = %q", b.Kind)
	}
	if _, err := ParseBlock("MAIN\n  DISPLAY \"x\"\nEND MAIN\n", "FUNCTION"); err == nil || err.Code != CodeNotAFunction {
		t.Errorf("ParseBlock(MAIN 文本, FUNCTION) 期望 NOT_A_FUNCTION，实际 %v", err)
	}
	if b, err := ParseBlock("MAIN\n  DISPLAY \"x\"\nEND MAIN\n", "MAIN"); err != nil {
		t.Errorf("ParseBlock(MAIN 文本, MAIN) 不该失败: %s", err.Error())
	} else if b.Kind != "MAIN" {
		t.Errorf("ParseBlock(MAIN).Kind = %q", b.Kind)
	}
}

// TestEnvelopeKindFor 钉住 TAP point 前缀 → 信封种类的映射。
func TestEnvelopeKindFor(t *testing.T) {
	cases := []struct {
		point string
		kind  string
		ok    bool
	}{
		{"function.aapp131_qbe_clear", "FUNCTION", true},
		{"dialog.aooi350_01_display", "DIALOG", true},
		{"report.r_fmt", "REPORT", true},
		{"Function.Upper", "FUNCTION", true},
		{"DIALOG.Upper", "DIALOG", true},
		{"Report.Mixed", "REPORT", true},
		// 同为 T100 组名、但正文不是「一个块信封」的，必须一律不认。
		{"main.aapp131", "", false},
		{"input.aapp131", "", false},
		{"construct.aapp131", "", false},
		{"menu.aapp131", "", false},
		{"global.g_x", "", false},
		{"before_delete.x", "", false},
		{"set_act_visible_b.x", "", false},
		// 必须带点，避免前缀撞车。
		{"functions.x", "", false},
		{"dialogs.x", "", false},
		{"reports.x", "", false},
		{"function", "", false},
		{"functionx.y", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		kind, ok := EnvelopeKindFor(tc.point)
		if ok != tc.ok || kind != tc.kind {
			t.Errorf("EnvelopeKindFor(%q) = (%q,%v)，期望 (%q,%v)", tc.point, kind, ok, tc.kind, tc.ok)
		}
	}
}

// TestParseFunctionNameSpan 校验名称的 0-based 字节列。
func TestParseFunctionNameSpan(t *testing.T) {
	text := "FUNCTION (r_area Rect) area() RETURNS FLOAT\n  RETURN r_area.w\nEND FUNCTION\n"
	b, perr := ParseFunction(text)
	if perr != nil {
		t.Fatalf("不该失败: %s", perr.Error())
	}
	if b.Name != "area" {
		t.Fatalf("Name = %q，期望 area", b.Name)
	}
	if b.NameSpan.Line != 1 || b.NameSpan.StartCol != 23 || b.NameSpan.EndCol != 27 {
		t.Fatalf("NameSpan = %+v，期望 {Line:1 StartCol:23 EndCol:27}", b.NameSpan)
	}
}

/* TestParseFunctionEOLInvariance：同内容的 LF / CRLF / 混合 / 无结尾换行 / 纯 CR
 * 必须给出**完全相同的行号**（行尾风格不得影响行网格）。 */
func TestParseFunctionEOLInvariance(t *testing.T) {
	type variant struct {
		name string
		text string
	}
	variants := []variant{
		{"LF", "# 横幅\nFUNCTION f_eol()\n  LET x = 1\n  LET y = 2\nEND FUNCTION\n"},
		{"LF 无结尾换行", "# 横幅\nFUNCTION f_eol()\n  LET x = 1\n  LET y = 2\nEND FUNCTION"},
		{"CRLF", "# 横幅\r\nFUNCTION f_eol()\r\n  LET x = 1\r\n  LET y = 2\r\nEND FUNCTION\r\n"},
		{"CRLF 无结尾换行", "# 横幅\r\nFUNCTION f_eol()\r\n  LET x = 1\r\n  LET y = 2\r\nEND FUNCTION"},
		{"混合行尾", "# 横幅\r\nFUNCTION f_eol()\n  LET x = 1\r\n  LET y = 2\nEND FUNCTION\r\n"},
		{"纯 CR", "# 横幅\rFUNCTION f_eol()\r  LET x = 1\r  LET y = 2\rEND FUNCTION\r"},
		{"纯 CR 无结尾换行", "# 横幅\rFUNCTION f_eol()\r  LET x = 1\r  LET y = 2\rEND FUNCTION"},
	}

	type shape struct {
		name      string
		header    Range
		body      Range
		terminatr Range
	}
	shapes := make([]shape, 0, len(variants))
	for _, v := range variants {
		b, perr := ParseFunction(v.text)
		if perr != nil {
			t.Fatalf("%s: 不该失败: %s", v.name, perr.Error())
		}
		if b.Name != "f_eol" {
			t.Fatalf("%s: Name = %q，期望 f_eol", v.name, b.Name)
		}
		shapes = append(shapes, shape{v.name, b.Header, b.Body, b.Terminator})
	}
	base := shapes[0]
	for _, s := range shapes[1:] {
		if s.header.StartLine != base.header.StartLine || s.header.EndLine != base.header.EndLine {
			t.Errorf("%s: Header 行 = [%d,%d]，与 %s 的 [%d,%d] 不一致",
				s.name, s.header.StartLine, s.header.EndLine, base.name, base.header.StartLine, base.header.EndLine)
		}
		if s.body.StartLine != base.body.StartLine || s.body.EndLine != base.body.EndLine {
			t.Errorf("%s: Body 行 = [%d,%d]，与 %s 的 [%d,%d] 不一致",
				s.name, s.body.StartLine, s.body.EndLine, base.name, base.body.StartLine, base.body.EndLine)
		}
		if s.terminatr.StartLine != base.terminatr.StartLine || s.terminatr.EndLine != base.terminatr.EndLine {
			t.Errorf("%s: Terminator 行 = [%d,%d]，与 %s 的 [%d,%d] 不一致",
				s.name, s.terminatr.StartLine, s.terminatr.EndLine, base.name, base.terminatr.StartLine, base.terminatr.EndLine)
		}
	}
	if base.header.StartLine != 2 || base.terminatr.StartLine != 5 {
		t.Fatalf("基准行号不对：Header=%d Terminator=%d（期望 2 / 5）", base.header.StartLine, base.terminatr.StartLine)
	}
}

// TestParseFunctionByteRanges 钉死字节口径：EndByte **不含**行尾 EOL，末行无 EOL 时 == len(text)。
func TestParseFunctionByteRanges(t *testing.T) {
	t.Run("LF 有结尾换行", func(t *testing.T) {
		text := "FUNCTION f()\n  LET x = 1\n  LET y = 2\nEND FUNCTION\n"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		wantHeader := Range{1, 1, 0, 12}
		wantBody := Range{2, 3, 13, 36}
		wantTerm := Range{4, 4, 37, 49}
		if b.Header != wantHeader {
			t.Errorf("Header = %+v，期望 %+v", b.Header, wantHeader)
		}
		if b.Body != wantBody {
			t.Errorf("Body = %+v，期望 %+v", b.Body, wantBody)
		}
		if b.Terminator != wantTerm {
			t.Errorf("Terminator = %+v，期望 %+v", b.Terminator, wantTerm)
		}
		if b.Terminator.EndByte == len(text) {
			t.Errorf("有结尾换行时 Terminator.EndByte 不该等于 len(text)=%d", len(text))
		}
		if got := text[b.Header.StartByte:b.Header.EndByte]; got != "FUNCTION f()" {
			t.Errorf("Header 切片 = %q", got)
		}
		if got := text[b.Terminator.StartByte:b.Terminator.EndByte]; got != "END FUNCTION" {
			t.Errorf("Terminator 切片 = %q", got)
		}
	})

	t.Run("LF 无结尾换行 → EndByte == len(text)", func(t *testing.T) {
		text := "FUNCTION f()\n  LET x = 1\n  LET y = 2\nEND FUNCTION"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if b.Terminator.EndByte != len(text) {
			t.Errorf("Terminator.EndByte = %d，期望 len(text) = %d", b.Terminator.EndByte, len(text))
		}
		if got := text[b.Terminator.StartByte:b.Terminator.EndByte]; got != "END FUNCTION" {
			t.Errorf("Terminator 切片 = %q", got)
		}
	})

	t.Run("CRLF 的 EOL 不计入", func(t *testing.T) {
		text := "FUNCTION f()\r\n  LET x = 1\r\nEND FUNCTION\r\n"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if got := text[b.Header.StartByte:b.Header.EndByte]; got != "FUNCTION f()" {
			t.Errorf("Header 切片 = %q", got)
		}
		if got := text[b.Body.StartByte:b.Body.EndByte]; got != "  LET x = 1" {
			t.Errorf("Body 切片 = %q", got)
		}
		if got := text[b.Terminator.StartByte:b.Terminator.EndByte]; got != "END FUNCTION" {
			t.Errorf("Terminator 切片 = %q", got)
		}
	})

	t.Run("孤立 CR 的 EOL 不计入", func(t *testing.T) {
		text := "FUNCTION f()\r  LET x = 1\rEND FUNCTION\r"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if got := text[b.Header.StartByte:b.Header.EndByte]; got != "FUNCTION f()" {
			t.Errorf("Header 切片 = %q", got)
		}
		if got := text[b.Body.StartByte:b.Body.EndByte]; got != "  LET x = 1" {
			t.Errorf("Body 切片 = %q", got)
		}
	})
}

// TestParseFunctionEmptyBody：头行紧接终止行（含同行开闭）时 Body 为空，且不是错误。
func TestParseFunctionEmptyBody(t *testing.T) {
	t.Run("相邻两行", func(t *testing.T) {
		text := "FUNCTION f()\nEND FUNCTION\n"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if !b.Body.Empty() {
			t.Errorf("Body = %+v，期望为空", b.Body)
		}
		if b.Body.StartLine != 2 || b.Body.EndLine != 1 {
			t.Errorf("Body 行 = [%d,%d]，期望 [2,1]", b.Body.StartLine, b.Body.EndLine)
		}
		if b.Body.StartByte != b.Header.EndByte || b.Body.EndByte != b.Header.EndByte {
			t.Errorf("空体字节 = [%d,%d]，期望都等于 Header.EndByte=%d",
				b.Body.StartByte, b.Body.EndByte, b.Header.EndByte)
		}
	})

	t.Run("同行开闭", func(t *testing.T) {
		text := "FUNCTION f() END FUNCTION\n"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if !b.Body.Empty() {
			t.Errorf("Body = %+v，期望为空", b.Body)
		}
		if b.Header.StartLine != 1 || b.Terminator.StartLine != 1 {
			t.Errorf("头/终止行 = %d/%d，期望 1/1", b.Header.StartLine, b.Terminator.StartLine)
		}
	})
}

/* TestParseFunctionCRLineComment 是**回归用例**（fgloutline.ts:237 的缺陷）：
 * 原实现在行注释里只在 LF 处停下，于是纯 CR 文件里第一个 # / -- 注释会把其后整份
 * 文件吞成空格 —— 大纲全空、信封判定直接 NO_FUNCTION_HEADER。
 * 修正为「!== LF && !== CR」后，CR 同样是行尾。 */
func TestParseFunctionCRLineComment(t *testing.T) {
	cases := []struct {
		name       string
		text       string
		wantName   string
		wantHeader int
		wantTerm   int
	}{
		{"# 注释在头行之前", "# 横幅注释\rFUNCTION f_cr()\r  LET x = 1\rEND FUNCTION\r", "f_cr", 2, 4},
		{"# 注释在函数体内", "FUNCTION f_cr()\r  # 体内注释\r  LET x = 1\rEND FUNCTION\r", "f_cr", 1, 4},
		{"-- 注释在头行之前", "-- 横幅注释\rFUNCTION f_cr()\r  LET x = 1\rEND FUNCTION\r", "f_cr", 2, 4},
		{"-- 注释在函数体内", "FUNCTION f_cr()\r  -- 体内注释 with ' apostrophe\r  LET x = 1\rEND FUNCTION\r", "f_cr", 1, 4},
		{"无结尾 CR", "# 横幅\rFUNCTION f_cr()\r  LET x = 1\rEND FUNCTION", "f_cr", 2, 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, perr := ParseFunction(tc.text)
			if perr != nil {
				t.Fatalf("不该失败（CR 行注释吞掉了后续内容？）: %s", perr.Error())
			}
			if b.Name != tc.wantName {
				t.Fatalf("Name = %q，期望 %s", b.Name, tc.wantName)
			}
			if b.Header.StartLine != tc.wantHeader {
				t.Errorf("Header.StartLine = %d，期望 %d", b.Header.StartLine, tc.wantHeader)
			}
			if b.Terminator.StartLine != tc.wantTerm {
				t.Errorf("Terminator.StartLine = %d，期望 %d", b.Terminator.StartLine, tc.wantTerm)
			}
		})
	}
}

// TestParseOutlineCommentCROnly 是大纲层的同一回归：纯 CR 文件里 # / -- 注释
// 必须只吃到行尾，否则整份文件被掩成空格、一个节点都不剩。
func TestParseOutlineCommentCROnly(t *testing.T) {
	const text = "# 横幅\rmodule_something\r-- 第二条注释\rFUNCTION f_a()\rEND FUNCTION\rFUNCTION f_b()\rEND FUNCTION\r"
	nodes := ParseOutline(text)
	if len(nodes) != 2 {
		t.Fatalf("顶层节点 = %d，期望 2（%+v）", len(nodes), nodes)
	}
	if nodes[0].Label != "f_a" || nodes[0].Line != 4 || nodes[0].EndLine != 5 {
		t.Errorf("nodes[0] = %+v，期望 f_a L4-5", nodes[0])
	}
	if nodes[1].Label != "f_b" || nodes[1].Line != 6 || nodes[1].EndLine != 7 {
		t.Errorf("nodes[1] = %+v，期望 f_b L6-7", nodes[1])
	}
}

// TestParseOutlineLineCommentStopsAtCRLF 保证 CRLF 下 `#`/`--` 注释只吃掉本行
// （CR 被丢掉、LF 断行，不会多吞一行）。
func TestParseOutlineLineCommentStopsAtCRLF(t *testing.T) {
	const text = "# 注释一\r\nFUNCTION f_x()\r\n  # 注释二\r\n  LET y = 1\r\nEND FUNCTION\r\n"
	nodes := ParseOutline(text)
	if len(nodes) != 1 {
		t.Fatalf("顶层节点 = %d，期望 1", len(nodes))
	}
	n := nodes[0]
	if n.Label != "f_x" || n.Line != 2 || n.EndLine != 5 {
		t.Fatalf("节点 = %+v，期望 f_x L2-5", n)
	}
}

// TestParseFunctionLeadingIndentAndTrailingGarbage 覆盖头行前的缩进、头行后的
// RETURNS 子句，以及第一行压着 CDATA 前缀（调用方从半截内容开始）的情形。
func TestParseFunctionLeadingIndentAndTrailingGarbage(t *testing.T) {
	t.Run("头行缩进 + RETURNS 子句", func(t *testing.T) {
		text := "   PUBLIC FUNCTION calc(a INTEGER) RETURNS INTEGER\n      RETURN a\n   END FUNCTION\n"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if b.Name != "calc" || b.Scope != "PUBLIC" {
			t.Fatalf("Name/Scope = %q/%q", b.Name, b.Scope)
		}
		if b.Header.StartLine != 1 || b.Terminator.StartLine != 3 {
			t.Fatalf("头/终止行 = %d/%d，期望 1/3", b.Header.StartLine, b.Terminator.StartLine)
		}
	})

	t.Run("正文从半截开始（首行前压着 CDATA 前缀后的内容）", func(t *testing.T) {
		text := "\n# 横幅\nPRIVATE FUNCTION p_x()\nEND FUNCTION"
		b, perr := ParseFunction(text)
		if perr != nil {
			t.Fatalf("不该失败: %s", perr.Error())
		}
		if b.Header.StartLine != 3 || b.Terminator.StartLine != 4 {
			t.Fatalf("头/终止行 = %d/%d，期望 3/4", b.Header.StartLine, b.Terminator.StartLine)
		}
	})
}
