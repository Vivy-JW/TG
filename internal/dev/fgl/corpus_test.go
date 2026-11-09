package fgl

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/testkit"
)

/* ============================================================================
 * 真实语料扫场：D:\t100_wrok_dir 下的 166 个 .tzc 设计器包
 *
 * 目的：**经验下限**。设计器今天能打开这些包，所以我们的信封判定不能把其中任何一个
 * 正常块判成结构性损坏（MULTIPLE_BLOCKS / UNTERMINATED / TERMINATOR_MISMATCH）。
 *
 * 做法：用 archive/zip 打开每个 .tzc，取出 .tap 条目，扫描每个 <point …> 的 CDATA
 * 正文；只测 name 以 function./dialog./report. 开头且正文非空的点。正文**不做任何
 * 规范化**（连首行 `<![CDATA[` 前缀都照原样留给行网格）。
 *
 * 语料目录缺失时整条测试跳过 —— 仓库里没有这 1.6GB。根怎么定只有一处：
 * testkit.CorpusRoot（认 TDEV_CORPUS 与 TTZS_CORPUS 两个变量）。
 * ========================================================================== */

// scanPoints 扫描 XML 文本里的 `<point …>…</point>`，对每个点回调其 name 属性与
// CDATA 正文。回调返回 false 可提前结束。
//
// 手写而不用 encoding/xml：.tap 里出现过未转义的 `&`（设计器能读，严格 XML 解析器不认），
// 而这里只需要两个东西 —— name 属性与 CDATA 区间。
func scanPoints(data []byte, fn func(name, body string) bool) {
	s := string(data)
	pos := 0
	for {
		i := strings.Index(s[pos:], "<point")
		if i < 0 {
			return
		}
		i += pos
		j := i + len("<point")
		if j >= len(s) || !isXMLSpace(s[j]) && s[j] != '>' {
			pos = j
			continue
		}
		// 找标签结束的 '>'（跳过引号里的内容）。
		tagEnd, quote := -1, byte(0)
		for k := j; k < len(s); k++ {
			c := s[k]
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '"' || c == '\'' {
				quote = c
				continue
			}
			if c == '>' {
				tagEnd = k
				break
			}
		}
		if tagEnd < 0 {
			return
		}
		name := attrValue(s[j:tagEnd], "name")
		rel := strings.Index(s[tagEnd:], "</point>")
		if rel < 0 {
			return
		}
		inner := s[tagEnd+1 : tagEnd+rel]
		body := ""
		if c0 := strings.Index(inner, "<![CDATA["); c0 >= 0 {
			rest := inner[c0+len("<![CDATA["):]
			if c1 := strings.Index(rest, "]]>"); c1 >= 0 {
				body = rest[:c1]
			}
		}
		if !fn(name, body) {
			return
		}
		pos = tagEnd + rel + len("</point>")
	}
}

func isXMLSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

// attrValue 从标签属性串里取 `key="value"` / `key='value'`（大小写敏感、要求前置分隔）。
func attrValue(attrs, key string) string {
	for i := 0; ; {
		k := strings.Index(attrs[i:], key)
		if k < 0 {
			return ""
		}
		k += i
		before := k == 0 || isXMLSpace(attrs[k-1])
		p := k + len(key)
		if before && p < len(attrs) && attrs[p] == '=' {
			p++
			if p < len(attrs) && (attrs[p] == '"' || attrs[p] == '\'') {
				q := attrs[p]
				p++
				if e := strings.IndexByte(attrs[p:], q); e >= 0 {
					return attrs[p : p+e]
				}
				return ""
			}
		}
		i = k + len(key)
	}
}

type corpusFail struct {
	pkg, point, code, detail string
}

// 语料里唯一一个「正文根本不含块」的点：s_axmt500(s).tzc 的 function.memo_industry，
// 它的整段 CDATA 只有一行 `#` 注释（2024 年的一条补丁说明），没有任何块头。
//
// 这不是白名单，而是**经验基线**：扫场断言「除了它，一个点都不许失败，且它必须正好
// 失败在 NO_FUNCTION_HEADER」。如果哪天这份语料变了（该点补上了函数、或目录被替换），
// 这条断言会失败 —— 那时照着失败信息更新这两个常量即可。
const (
	baselineDegeneratePkg   = "s_axmt500(s).tzc"
	baselineDegeneratePoint = "function.memo_industry"
)

func TestCorpusEnvelope(t *testing.T) {
	files := testkit.CorpusFiles(t, ".tzc")

	var (
		points      int
		nFunction   int
		nDialog     int
		nReport     int
		ok          int
		byKind      = map[string]int{}
		unreadable  int
		fails       []corpusFail
		absentKind  int
		emptyPoints int
	)
	for _, p := range files {
		zr, err := zip.OpenReader(p)
		if err != nil {
			unreadable++
			t.Logf("打不开包（跳过）: %s: %v", p, err)
			continue
		}
		base := filepath.Base(p)
		for _, zf := range zr.File {
			if !strings.EqualFold(filepath.Ext(zf.Name), ".tap") {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				unreadable++
				continue
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				unreadable++
				continue
			}
			scanPoints(data, func(name, body string) bool {
				// 调用方要走的正是这条路径：前缀决定信封种类。
				wantKind, isEnvelope := EnvelopeKindFor(name)
				if !isEnvelope {
					return true
				}
				switch wantKind {
				case "FUNCTION":
					nFunction++
				case "DIALOG":
					nDialog++
				case "REPORT":
					nReport++
				}
				if strings.TrimSpace(body) == "" {
					emptyPoints++
					return true
				}
				points++

				b, perr := ParseBlock(body, wantKind)
				if perr == nil {
					ok++
					byKind[b.Kind]++
					return true
				}
				if perr.Code == CodeInvalidKind {
					// EnvelopeKindFor 只会吐出 ParseBlock 认识的种类；真出现就是实现 bug。
					absentKind++
				}
				fails = append(fails, corpusFail{base, name, perr.Code, perr.Error()})
				return true
			})
		}
		zr.Close()
	}

	t.Logf("语料扫描：包 %d 个（不可读 %d），点 %d 个（function. %d / dialog. %d / report. %d；空正文另计 %d）",
		len(files), unreadable, points, nFunction, nDialog, nReport, emptyPoints)
	t.Logf("ParseBlock(EnvelopeKindFor(点名前缀))：通过 %d %v，失败 %d",
		ok, byKind, len(fails))

	if points == 0 {
		t.Fatalf("一个点都没扫到 —— 语料目录结构可能变了")
	}
	if nFunction == 0 {
		t.Fatalf("没有扫到 function. 点 —— 语料目录结构可能变了")
	}
	if absentKind != 0 {
		t.Fatalf("EnvelopeKindFor 吐出了 %d 次 ParseBlock 不认识的种类 —— 两个函数的前缀表不一致", absentKind)
	}

	// 断言：失败集合**必须恰好**是那一个退化点，且错误码必须是 NO_FUNCTION_HEADER。
	const show = 20
	n := len(fails)
	if n > show {
		n = show
	}
	for _, f := range fails[:n] {
		if f.pkg != baselineDegeneratePkg || f.point != baselineDegeneratePoint {
			t.Errorf("不该失败的点：包 %s 点 %s：%s（%s）", f.pkg, f.point, f.code, f.detail)
			continue
		}
		if f.code != CodeNoFunctionHeader {
			t.Errorf("基线退化点 %s 的错误码 = %s，期望 %s", f.point, f.code, CodeNoFunctionHeader)
		}
	}
	if len(fails) > n {
		t.Errorf("… 另有 %d 个失败点未列出", len(fails)-n)
	}
	if len(fails) != 1 {
		t.Fatalf("语料扫场：%d/%d 个点被判定为信封不良构，期望恰好 1 个（基线退化点 %s / %s）",
			len(fails), points, baselineDegeneratePkg, baselineDegeneratePoint)
	}
	if fails[0].pkg != baselineDegeneratePkg || fails[0].point != baselineDegeneratePoint {
		t.Fatalf("唯一允许失败的点变了：实际是 %s / %s（%s），期望 %s / %s",
			fails[0].pkg, fails[0].point, fails[0].code,
			baselineDegeneratePkg, baselineDegeneratePoint)
	}
	if fails[0].code != CodeNoFunctionHeader {
		t.Fatalf("基线退化点 %s 的错误码 = %s，期望 %s",
			fails[0].point, fails[0].code, CodeNoFunctionHeader)
	}

	if ok+len(fails) != points {
		t.Fatalf("分类计数不一致：通过 %d + 失败 %d != 点 %d", ok, len(fails), points)
	}
}
