// Package fgl 提供两件与 Genero 4GL(BDL)源码结构有关的能力：
//
//   - outline.go —— 源码大纲解析（块节点树）。
//   - block.go   —— **一个块信封**的判定与行/字节范围（FUNCTION / MAIN / DIALOG / REPORT；
//     真实语料里 `function.` / `dialog.` / `report.` 点装的都是同形状的信封）。
//
// outline.go 是 TDebug `web/src/fgloutline.ts`（491 行）的 1:1 Go 移植；后者又是
// BDL 扩展（`D:\我的项目\BDL`，同作者，MIT）`src/extension.ts` 的移植，只保留大纲、
// 不保留折叠（FOLD_ONLY / scanFoldOnly / RE_FOLD_* 一律未移植）。
//
// 结构：maskLines(掩码) → scanBlocks(块栈扫描) → fixup(范围修正) → toOutlineNodes(转对外节点)。
// 设计要点（逐条都有真实语料实证，详见 BDL docs/DESIGN.md）：
//   - 必须用块栈而不是缩进：真实源码的缩进会谎报层级（同类子块 ON ACTION 比宿主还浅）。
//   - 扫描前把字符串内容与三类注释（#、--、{ }）抹成等长空格：否则跨行字符串里的 SQL
//     关键字、块注释里的 ON ACTION、注释里的撇号（会开启跨行字符串状态）都会造幻影或
//     吃掉成片节点。
//   - 块开头要按构造消歧：RECORD 里名叫 report/construct 的字段不是块开头。
//   - 深嵌套一律用显式栈，不用递归（畸形文件约 2000 层嵌套会让递归栈溢出、大纲全空）。
//
// 与 BDL 的一致性约定：标签文本、块范围、以及 BDL 已声明的已知偏差（见
// testdata/fgl-fixtures 里 *.expected.json 的 deviation 字段）都照原样保留，
// 以便用同一批文档推导夹具 1:1 对拍。
//
// # 移植到 Go 时的两处**有意偏差**（其余逐行等价）
//
//  1. RE2 没有前瞻/后顾。fgloutline.ts 里的 NB = `(?![A-Za-z0-9_])` 按位置分两类处理：
//     - NB 后面还跟着「首字符集与 [A-Za-z0-9_] 不相交」的东西（W_P、W_S+$）时，
//     前瞻恒真，直接删掉；
//     - NB 位于分支末尾时，换成消费一个边界字符的 `(?P<nbX>[^A-Za-z0-9_]|$)`，
//     再用该捕获组的起点把匹配长度退回去（见 reMatch.effEnd），效果与零宽前瞻一致。
//     两处「正向/负向前瞻」是构造消歧（construct 的 `(?!(TYPE_KEYWORDS)NB)` 与
//     `(?=W_S(ON|FROM|ATTRIBUTE(S))NB|$)`），改为匹配后在 describe() 里显式判定。
//  2. 按**字节**而不是 UTF-16 码元推进（Go 字符串本来就是字节）。code/disp 两份数组仍然
//     逐字节对齐，正则下标与 selStart/selEnd 因此是**字节列**；对纯 ASCII 行与 TS 完全一致
//     （夹具 function-method-receiver-namecol 正是这种情形）。
package fgl

import (
	"regexp"
	"strings"
)

// OutlineKind 是块种类，与 fgloutline.ts:16-19 的 OutlineKind 一一对应。
type OutlineKind string

const (
	KindFunction  OutlineKind = "FUNCTION"
	KindMain      OutlineKind = "MAIN"
	KindReport    OutlineKind = "REPORT"
	KindDialog    OutlineKind = "DIALOG"
	KindInput     OutlineKind = "INPUT"
	KindDisplay   OutlineKind = "DISPLAY"
	KindConstruct OutlineKind = "CONSTRUCT"
	KindMenu      OutlineKind = "MENU"
	KindSub       OutlineKind = "SUB"
)

// OutlineNode 是对外的大纲节点。
type OutlineNode struct {
	Label   string
	Kind    OutlineKind
	Line    int // 1-based，原文行号（含）
	EndLine int // 1-based，块结束行（含）；= 配对 END <TYPE> 所在行

	// SelStart/SelEnd 是名称的落点（selectionRange 语义），0-based、半开区间。
	//
	// **注意单位：字节列**。TS 那边是 UTF-16 码元列（Monaco 的语义）；Go 这边刻意改成
	// 字节列 —— 本包的下游是「按字节改写 T100 包」的 CLI，字节偏移可以直接切片，而
	// 字节↔UTF-16 的换算才是额外负担。含中文的行上两者会差（每个 CJK 字符多 2 字节），
	// 纯 ASCII 行上逐位一致（已用 14 个真实模块、12,836 个节点对拍确认：除含非 ASCII
	// 字符的行的 SelEnd 外，label/kind/line/endLine/SelStart 全部与 TS 相同）。
	SelStart int
	SelEnd   int

	Children []OutlineNode
}

/* ============================ 词法表 ============================
 * 全部建为包级常量：大 alternation 每次调用重建会多花 ~78ms（fgloutline.ts:31-34）。
 */

const (
	wS = `[ \t]*`
	wP = `[ \t]+`
	// 限定名 / 数组记录前缀：g_pmdl_m.*、s_browse.*、type_t.chr50（fgloutline.ts:38）。
	idLex = `[A-Za-z_][A-Za-z0-9_.]*`
)

/* 注意：所有正则都只对「去掉行尾 \r」的副本求值（fgloutline.ts:42-44）。
 * 本文件常见 100% CRLF，而 JS 的 `.` 不匹配 \r —— 任何以 `.*$` 结尾的正则
 * 在原始行上会静默返回 0 个匹配。见 maskLines 里的 code/disp 两份数组。 */

// END <TYPE>。故意不带 `$`（fgloutline.ts:46-50）。
var reEnd = regexp.MustCompile(`(?i)^` + wS + `END` + wP +
	`(?P<endType>FUNCTION|MAIN|REPORT|DIALOG|INPUT|DISPLAY|CONSTRUCT|MENU)(?:[^A-Za-z0-9_]|$)`)

/* `{ }` 是 BDL 的块注释（不可嵌套），与高亮规则保持一致（fgloutline.ts:52-54）。 */
const (
	chLF   = '\n'
	chCR   = '\r'
	chHash = '#'
	chSQ   = '\''
	chDQ   = '"'
	chBT   = '`'
	chBS   = '\\'
	chDash = '-'
	chLB   = '{'
	chRB   = '}'
)

/* BDL 数据类型关键字。用来把 RECORD 里的字段声明与块开头区分开：
 * `construct INTEGER` 是「名为 construct 的字段，类型 INTEGER」，不是 CONSTRUCT 块。
 * 只在「名字后面什么都没有（行尾）」时才需要它来消歧（fgloutline.ts:56-65）。 */
const typeKeywords = `INTEGER|INT|SMALLINT|BIGINT|TINYINT|SERIAL8|SERIAL|BIGSERIAL|` +
	`CHAR|VARCHAR|NVARCHAR|STRING|TEXT|DECIMAL|NUMERIC|MONEY|` +
	`FLOAT|SMALLFLOAT|DOUBLE|DATE|DATETIME|INTERVAL|BOOLEAN|BYTE|` +
	`DYNAMIC|STATIC|RECORD|LIKE|ARRAY`

/* 子块开头：BEFORE ROW / AFTER FIELD x / ON ACTION controlp INFIELD f / ON CHANGE f
 *
 * 后接的关键字必须是**文档里列出的**事件/控制块，不能是「ON 后跟任意东西」——
 * 否则 SQL 的 JOIN 条件只要被格式化到行首就会被当成 dialog 子块：
 *   `ON t1.col_a = t2.col_a AND t1.col_b = t2.col_b`（fgloutline.ts:67-79）
 */
const subEvent = `ACTION|CHANGE|KEY|IDLE|TIMER|ROW` + wP + `CHANGE|EVERY` + wP + `ROW|LAST` + wP + `ROW|` +
	`FILL` + wP + `BUFFER|SORT|APPEND|INSERT|UPDATE|DELETE|EXPAND|COLLAPSE|` +
	`SELECTION` + wP + `CHANGE|DRAG_START|DRAG_FINISHED|DRAG_ENTER|DRAG_OVER|DROP`

const subControl = `INPUT|CONSTRUCT|DISPLAY|DIALOG|MENU|ROW|FIELD|INSERT|DELETE|UPDATE|GROUP` + wP + `OF`

// REPORT 的 FORMAT 段控制块不属于 `ON …` / `BEFORE|AFTER …` 形状，得单列一支（fgloutline.ts:80-81）。
const subReport = `(?:FIRST` + wP + `)?PAGE` + wP + `(?:HEADER|TRAILER)`

/* `COMMAND [KEY (…)] option` 是 MENU 的 menu-option，与 `ON ACTION` **平级**，
 * 在 DIALOG 里也是 dialog-control-block。不认它，前面的 `BEFORE MENU` 会一路吞到
 * 下一个被认出的子块，把整段菜单项包进自己的范围里（fgloutline.ts:82-87）。 */
const subCommand = `(?:COMMAND)(?:` + wP + `KEY` + wS + `\([^)]*\))?`

const subHeadCore = `(?:(?:BEFORE|AFTER)` + wP + `(?:` + subControl + `))` +
	`|(?:(?:ON)` + wP + `(?:` + subEvent + `))` +
	`|(?:` + subReport + `)` +
	`|(?:` + subCommand + `)`

// RE_SUB：TS 里 head 之后是零宽 NB，再 `(?P<rest>.*)`；这里把 NB 变成消费一个字符的
// 具名组，于是 head+nb+rest 与 TS 的 head+rest 完全相同（fgloutline.ts:88-93）。
var reSub = regexp.MustCompile(`(?i)^` + wS + `(?P<head>` + subHeadCore + `)` +
	`(?P<nb>[^A-Za-z0-9_]|$)(?P<rest>.*)`)

/* 危险点：`INPUT NO WRAP` / `INPUT WRAP` 是 OPTIONS 子句，不是 dialog 块，
 * 且它落点和真正的 INPUT 同一个缩进层级。整行精确匹配，避免把
 * 「记录名恰好以 NO 开头」的真 INPUT 误跳（fgloutline.ts:95-98）。 */
var reInputOption = regexp.MustCompile(`(?i)^` + wS + `INPUT` + wP + `(?:NO` + wP + `)?WRAP` + wS + `$`)

/* 同上，但对应 RE_OPEN 里 input 分支的负向前瞻 `(?!W_P(?:NO W_P)?WRAP NB)`：
 * 命中它就等于那个前瞻失败；而 fgloutline.ts 里该分支失败后其余分支也不可能命中
 * 同一行（只有 input / inputBare 以 INPUT 开头），所以「整行不做块开头」等价。 */
var reInputWrapGuard = regexp.MustCompile(`(?i)^` + wS + `INPUT` + wP + `(?:NO` + wP + `)?WRAP(?:[^A-Za-z0-9_]|$)`)

/* 全部块开头。备选项顺序有意义：带参数的形式必须排在裸形式之前（fgloutline.ts:100-140）。
 *
 * 与 TS 的差异逐条列在包注释里，另有两点顺序说明：
 *   - menuAttr 被提到 menu 之前。TS 靠 `(?!(?:ATTRIBUTES?)NB)` 把 `MENU ATTRIBUTE(...)`
 *     推给 menuAttr 分支；而 menuAttr 分支（`MENU W_P ATTRIBUTES? NB`）在前瞻成立时
 *     必定能匹配，所以交换顺序等价，且省掉一个前瞻。
 *   - construct 的两个前瞻改为匹配后在 describe() 里判定。 */
var reOpen = regexp.MustCompile(`(?i)^` + wS + `(?:(?:PUBLIC|PRIVATE)` + wP + `)?(?:` +
	`(?P<fn>FUNCTION)` + wS + `(?:\([^)]*\)` + wS + `)?(?P<fnName>` + idLex + `)` + wS + `\(` +
	`|(?P<main>MAIN)(?P<nbMain>[^A-Za-z0-9_]|$)` +
	// REPORT 必须带 name( —— 否则 RECORD 里名叫 `report` 的字段会被当成块开头（fgloutline.ts:105-109）。
	`|(?P<report>REPORT)` + wP + `(?P<reportName>` + idLex + `)` + wS + `\(` +
	// DIALOG：`(?![A-Za-z0-9_.])` 挡掉 DIALOG.getCurrentRow / ui.DIALOG / FGL_DIALOG_*
	// 顺序有意义：ATTRIBUTES 必须排在带名形式之前（fgloutline.ts:110-115）。
	`|(?P<dialog>DIALOG)(?:` + wP + `ATTRIBUTES?(?P<nbDlg>[^A-Za-z0-9_]|$)` +
	`|` + wP + `(?P<dialogName>` + idLex + `)` + wS + `\(` +
	`|` + wS + `$)` +
	// INPUT：自带 (?!…WRAP…) 守卫（改用 reInputWrapGuard），使 `INPUT NO WRAP` 落到
	// 「不匹配」而非把 NO 当记录名（fgloutline.ts:116-120）。
	`|(?P<input>INPUT)` + wP + `(?:(?P<inputArray>ARRAY` + wP + idLex + `)` +
	`|(?P<inputByName>BY` + wP + `NAME` + wP + idLex + `)` +
	`|(?P<inputRecord>` + idLex + `))` +
	`|(?P<display>DISPLAY)` + wP + `ARRAY` + wP + `(?P<displayArray>` + idLex + `)` +
	// CONSTRUCT 后面跟 ON / FROM / ATTRIBUTE(S) 或行尾（多行声明，fgloutline.ts:122-128）。
	// 注意：TS 这里用的是**零宽**前瞻，所以 mo[0] 停在名字之后、不含尾随空白。
	`|(?P<construct>CONSTRUCT)` + wP + `(?P<constructBy>BY` + wP + `NAME` + wP + `)?` +
	`(?P<constructName>` + idLex + `)` +
	// 无标题 MENU 直接跟属性表，必须单列一支（fgloutline.ts:133-135）。
	`|(?P<menuAttr>MENU)` + wP + `ATTRIBUTES?(?P<nbMenuAttr>[^A-Za-z0-9_]|$)` +
	// MENU 的标题可以是双引号、单引号、反引号或不加引号的标识符（fgloutline.ts:129-132）。
	`|(?P<menu>MENU)` + wP + `(?P<menuText>"[^"]*"|'[^']*'|` + "`[^`]*`" + `|` + idLex + `)` +
	`|(?P<inputBare>INPUT)` + wS + `$` +
	`|(?P<menuBare>MENU)` + wS + `$` +
	`)`)

// construct 的两个前瞻替身（fgloutline.ts:126-128）。
var (
	// `(?!(?:TYPE_KEYWORDS)NB)`：名字不得以类型关键字 + 词边界开头。
	reConstructTypeKw = regexp.MustCompile(`(?i)^(?:` + typeKeywords + `)(?:[^A-Za-z0-9_]|$)`)
	// `(?=W_S(?:(?:ON|FROM|ATTRIBUTE|ATTRIBUTES)NB|$))`：名字之后要么行尾，要么这些关键字。
	reConstructTail = regexp.MustCompile(`(?i)^[ \t]*(?:(?:ON|FROM|ATTRIBUTE|ATTRIBUTES)(?:[^A-Za-z0-9_]|$)|$)`)
)

// SUB 标签里 COMMAND 只留 option-name（fgloutline.ts:366-373）。
var reCommandHead = regexp.MustCompile(`(?i)^(?P<kw>COMMAND(?:[ \t]+KEY[ \t]*\([^)]*\))?)[ \t]+` +
	`(?P<opt>"[^"]*"|'[^']*'|` + "`[^`]*`" + `|[A-Za-z_][A-Za-z0-9_]*(?:\[[^\]]*\])*)`)

// 结束一个块用哪个 END <TYPE>（fgloutline.ts:142-146）。
var terminatorOf = map[OutlineKind]string{
	KindFunction: "FUNCTION", KindMain: "MAIN", KindReport: "REPORT", KindDialog: "DIALOG",
	KindInput: "INPUT", KindDisplay: "DISPLAY", KindConstruct: "CONSTRUCT", KindMenu: "MENU",
}

// 模块级单元：永不嵌套，且它们的 END 能修复夹在中间未闭合的 UI 块（fgloutline.ts:147-148）。
func isModuleKind(k string) bool {
	return k == "FUNCTION" || k == "MAIN" || k == "REPORT"
}

// 同行开闭（文档中合法）：MENU "t" COMMAND "Quit" EXIT MENU END MENU（fgloutline.ts:150-154）。
var reSameLineEnd = map[OutlineKind]*regexp.Regexp{}

// 未闭合的非模块帧在 EOF 处的跨度上限，防止被截断的文件吞掉整个文档（fgloutline.ts:156-157）。
const maxUnclosedSpan = 10000

const chSpace = ' '

func init() {
	for k, term := range terminatorOf {
		reSameLineEnd[k] = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])END` + wP + term + `(?:[^A-Za-z0-9_]|$)`)
	}
}

/* ============================ 剥注释 / 剥字符串 ============================ */

/*
maskLines 生成两份结构相同（等长、逐字节对齐）的行数组（fgloutline.ts:161-245）：

  - code：注释**和字符串内容**都抹成空格 —— 用于结构匹配。
    抹字符串内容是必须的：语料里有跨行 SQL 字面量，其续行以关键字开头，
    例如多行 "SELECT …" 字面量续行上的
    `ON t1.col_a = t2.col_a AND ...`（JOIN 条件），不抹就会被当成 dialog 子块。
  - disp：只抹注释、保留字符串 —— 用于取标签文字（例如 MENU 的标题）。

两份都在同一趟里产生，保证下标完全一致。引号状态跨行保持（跨行字面量）。
*/
func maskLines(text string) (code, disp []string) {
	c := make([]byte, 0, len(text))
	d := make([]byte, 0, len(text))
	var quote byte
	inBrace := false
	n := len(text)

	both := func(b byte) { c = append(c, b); d = append(d, b) }
	both2 := func(b1, b2 byte) { c = append(c, b1, b2); d = append(d, b1, b2) }

	for i := 0; i < n; i++ {
		ch := text[i]

		if ch == chLF {
			code = append(code, string(c))
			disp = append(disp, string(d))
			c = c[:0]
			d = d[:0]
			continue
		}

		// 行尾的 `\r` 必须丢掉（匹配用的副本带 `\r` 会让 `.*$` 这类静默失效），
		// 而且**单独的 `\r` 也是行终止符** —— 与 Monaco 的文档行模型
		// (`/\r\n|\r|\n/`) 一致（fgloutline.ts:189-197）。
		if ch == chCR {
			if i+1 < n && text[i+1] == chLF {
				continue // CRLF：丢掉 \r，由下一轮的 \n 断行
			}
			code = append(code, string(c)) // 孤立 \r：断行
			disp = append(disp, string(d))
			c = c[:0]
			d = d[:0]
			continue
		}

		// `{ … }` 是块注释，不可嵌套。内容两份都抹掉（fgloutline.ts:199-206）。
		if inBrace {
			if ch == chRB {
				inBrace = false
			}
			both(chSpace)
			continue
		}

		if quote != 0 {
			// `\` 在 ' 和 " 里也是转义（只有反引号是原始字符串）。
			// 漏掉这一步的后果非常严重：'\''（转义引号紧跟收尾引号）里的第一个 '
			// 会被「成对引号是字面引号」规则连同一个收尾 ' 一起吃掉，引号状态从此
			// 永久卡在字符串里（fgloutline.ts:208-219）。
			if ch == chBS {
				if i+1 < n {
					nx := text[i+1]
					if quote == chBT || nx == quote || nx == chBS {
						both2(chSpace, chSpace)
						d[len(d)-2] = chBS
						d[len(d)-1] = nx
						i++
						continue
					}
				}
				// TS 在 i+1 == n（字符串末尾一个孤立反斜杠）时会写出长度不一致的两行
				// （code 多一个空格、disp 追加 "undefined"），从而破坏两份数组的对齐。
				// 这里按「普通字符」处理：既不破坏对齐，也不改变任何可解析输入的语义。
			}
			if quote == chBT && ch == chBT {
				quote = 0
				both(ch)
				continue
			}
			if ch == quote {
				if quote != chBT && i+1 < n && text[i+1] == quote {
					both2(chSpace, chSpace)
					d[len(d)-2] = ch
					d[len(d)-1] = ch
					i++
					continue
				}
				quote = 0
				both(ch)
				continue
			}
			c = append(c, chSpace) // 字符串内容：code 抹成空格，disp 原样
			d = append(d, ch)
			continue
		}

		if ch == chSQ || ch == chDQ || ch == chBT {
			quote = ch
			both(ch)
			continue
		}
		if ch == chLB {
			inBrace = true
			both(chSpace)
			continue
		}
		// `--` 与 `#` 同为行注释。不认 `--` 的后果很隐蔽：注释里的撇号会开启**跨行**
		// 字符串状态，其后整片代码被掩成空格（fgloutline.ts:232-234）。
		if ch == chHash || (ch == chDash && i+1 < n && text[i+1] == chDash) {
			both(chSpace)
			// **有意修掉的 bug**：fgloutline.ts:237 的 `while (i+1 < n && text.charCodeAt(i+1) !== CH_LF)`
			// 只在 LF 处停下。行尾模型是 /\r\n|\r|\n/，所以纯 CR（老 Mac 风格）文件里
			// 第一个 `#`/`--` 注释会把**其后整份文件**都吞成空格 —— 大纲全空。
			// 这里补上 CH_CR：CR 也是行尾，注释到 CR 就必须停。
			// 回归用例见 block_test.go 的 TestParseOutlineCommentCROnly 与
			// TestParseFunctionCRLineComment，夹具见 fold-eol-midcr。
			for i+1 < n && text[i+1] != chLF && text[i+1] != chCR {
				both(chSpace)
				i++
			}
			continue
		}
		both(ch)
	}
	code = append(code, string(c))
	disp = append(disp, string(d))
	return code, disp
}

/* ============================ 节点 ============================ */

type rawNode struct {
	kind      OutlineKind
	label     string
	startLine int // 0-based
	endLine   int // 0-based
	selStart  int
	selEnd    int
	children  []*rawNode
}

var (
	reSpaceRun  = regexp.MustCompile(`[ \t]+`)
	reTrailPunc = regexp.MustCompile(`[.,]+$`)
)

func normalize(s string) string {
	s = reSpaceRun.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	s = reTrailPunc.ReplaceAllString(s, "")
	return s
}

/* reMatch 是「匹配 + 具名组下标」的薄封装：Go 的 FindStringSubmatchIndex 给的是
 * 字节下标，正好用来复刻 TS 里 `mo.index` / `mo[0].length` / `groups.x` 三种用法。 */
type reMatch struct {
	s     string
	idx   []int
	names []string
}

func match(re *regexp.Regexp, s string) *reMatch {
	idx := re.FindStringSubmatchIndex(s)
	if idx == nil {
		return nil
	}
	return &reMatch{s: s, idx: idx, names: re.SubexpNames()}
}

func (m *reMatch) text() string { return m.s[m.idx[0]:m.idx[1]] }
func (m *reMatch) start() int   { return m.idx[0] }
func (m *reMatch) end() int     { return m.idx[1] }

func (m *reMatch) span(name string) (int, int, bool) {
	for i, n := range m.names {
		if n == name {
			if m.idx[2*i] < 0 {
				return 0, 0, false
			}
			return m.idx[2*i], m.idx[2*i+1], true
		}
	}
	return 0, 0, false
}

func (m *reMatch) group(name string) string {
	s, e, ok := m.span(name)
	if !ok {
		return ""
	}
	return m.s[s:e]
}

func (m *reMatch) has(name string) bool {
	_, _, ok := m.span(name)
	return ok
}

/* effEnd 把「消费一个边界字符」的 NB 替身退回去，使 mo[0].length 与 TS 的零宽前瞻一致。
 * 这些 nbX 组恒为所属分支的最后一组，所以命中时匹配长度减 1 即可（$ 命中时为 0 长度）。 */
func (m *reMatch) effEnd() int {
	for _, name := range []string{"nbMain", "nbDlg", "nbMenuAttr"} {
		if s, e, ok := m.span(name); ok && e-s == 1 {
			return s
		}
	}
	return m.idx[1]
}

type desc struct {
	kind    OutlineKind
	label   string
	nameCol int // -1 表示无名称列（TS 里的 undefined）
}

/* describe 复刻 fgloutline.ts:261-290，另加 construct 的两个前瞻替身判定。 */
func describe(m *reMatch) (*desc, bool) {
	if m.has("fn") {
		name := m.group("fnName")
		// 函数名在参数表的 '(' 之前，必须从整个匹配里定位，否则会得到负偏移。
		return &desc{KindFunction, name, m.start() + maxInt(0, strings.LastIndex(m.text(), name))}, true
	}
	if m.has("main") {
		return &desc{KindMain, "MAIN", -1}, true
	}
	if m.has("report") {
		label := "REPORT"
		if n := m.group("reportName"); n != "" {
			label = "REPORT " + n
		}
		return &desc{KindReport, label, -1}, true
	}
	// 过程式 DIALOG 没有名字；模块级声明式 DIALOG name(params) 有，必须带上，
	// 否则同一模块里的多个声明式 DIALOG 会全部同名。
	if m.has("dialog") {
		label := "DIALOG"
		if n := m.group("dialogName"); n != "" {
			label = "DIALOG " + n
		}
		return &desc{KindDialog, label, -1}, true
	}
	if m.has("input") || m.has("inputBare") {
		arg := ""
		for _, g := range []string{"inputByName", "inputArray", "inputRecord"} {
			if v := m.group(g); v != "" {
				arg = v
				break
			}
		}
		return &desc{KindInput, normalize("INPUT " + arg), -1}, true
	}
	if m.has("display") {
		return &desc{KindDisplay, normalize("DISPLAY ARRAY " + m.group("displayArray")), -1}, true
	}
	if m.has("construct") {
		name := m.group("constructName")
		// (?!(?:TYPE_KEYWORDS)NB)：`construct LIKE …` 是 RECORD 字段，不是块。
		if reConstructTypeKw.MatchString(name) {
			return nil, false
		}
		// (?=W_S(?:(?:ON|FROM|ATTRIBUTE|ATTRIBUTES)NB|$))：多行声明的尾巴。
		if !reConstructTail.MatchString(m.s[m.effEnd():]) {
			return nil, false
		}
		by := ""
		if m.has("constructBy") {
			by = "BY NAME "
		}
		return &desc{KindConstruct, normalize("CONSTRUCT " + by + name), -1}, true
	}
	// menuAttr 是无标题 MENU 直接跟属性表，menuText 为 undefined → 标签就是 "MENU"。
	if m.has("menu") {
		return &desc{KindMenu, normalize("MENU " + m.group("menuText")), -1}, true
	}
	if m.has("menuAttr") || m.has("menuBare") {
		return &desc{KindMenu, normalize("MENU "), -1}, true
	}
	return nil, false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

/* scanBlocks：单趟前向扫描 + 显式栈。栈底是虚拟 ROOT，使每个节点都有父节点
 * （fgloutline.ts:292-421）。
 *
 * 推入：块开头作为栈顶的子节点推入且**不弹** SUB 帧（这正是 MENU 能嵌进
 * ON ACTION 的原因）；子块开头先弹掉所有 SUB 帧（它们是兄弟，永不互相嵌套）。
 * 弹出：END <TYPE> 自上而下跳过 SUB 帧，第一个非 SUB 帧的 kind 必须匹配，
 * 否则忽略该终止符（绝不猜）。 */
func scanBlocks(code, disp []string) []*rawNode {
	root := &rawNode{kind: KindFunction, startLine: 0, endLine: len(code) - 1}

	type frame struct {
		kind string
		node *rawNode
	}
	stack := []frame{{kind: "ROOT", node: root}}

	closeTop := func(endLine int) {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		f.node.endLine = maxInt(endLine, f.node.startLine)
	}

	/* 模块级 END 修复 / EOF 兜底时关闭栈顶（fgloutline.ts:312-333）。
	 *
	 * 块帧若**一条子节点都没有**，说明它是「无 control block 的裸语句」形式：
	 * 文档把 `END <TYPE>` 和 control block 放在**同一个可选组**里 —— 没有 control block
	 * 就没有 END，它只是一条单行语句。此时 range 不该延伸到父块末尾。 */
	closeTopRepair := func(endLine int) {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		isModule := isModuleKind(string(f.kind))
		bare := f.kind != string(KindSub) && !isModule && len(f.node.children) == 0
		var end int
		switch {
		case bare:
			end = f.node.startLine
		case isModule:
			end = endLine
		default:
			end = endLine
			if cap := f.node.startLine + maxUnclosedSpan; end > cap {
				end = cap
			}
		}
		f.node.endLine = maxInt(end, f.node.startLine)
	}

	for i := 0; i < len(code); i++ {
		line := code[i]
		if reInputOption.MatchString(line) {
			continue
		}

		/* --- 终止符 --- */
		if me := match(reEnd, line); me != nil {
			typ := strings.ToUpper(me.group("endType"))
			k := len(stack) - 1
			if isModuleKind(typ) {
				// 函数体不能嵌套：向下扫时不限定只跳 SUB，以此修复中间未闭合的 UI 块。
				for k > 0 && stack[k].kind != typ {
					k--
				}
			} else {
				for k > 0 && stack[k].kind == "SUB" {
					k--
				}
			}
			if k > 0 && stack[k].kind == typ {
				// 之间被夹住的帧用 repair 口径收尾（裸语句块收成单行）。
				for len(stack)-1 > k {
					closeTopRepair(i - 1)
				}
				closeTop(i)
			}
			continue
		}

		/* --- 子块开头：隐式结束上一个子块 --- */
		if ms := match(reSub, line); ms != nil {
			for len(stack) > 1 && stack[len(stack)-1].kind == "SUB" {
				closeTop(i - 1)
			}
			// 标签取 disp（保留字符串原文）—— 用 code 会让 `COMMAND "Quit" "Leave"`
			// 变成 `COMMAND " " " "`，因为字符串内容在 code 里已被抹成空格。
			msd := match(reSub, disp[i])
			if msd == nil {
				msd = ms
			}
			full := msd.group("head") + msd.group("nb") + msd.group("rest")
			head := msd.group("head") + msd.group("nb")
			label := normalize(strings.ToUpper(head) + msd.group("rest"))
			// COMMAND 的完整形式是 `COMMAND [KEY (…)] option-name [option-comment] [HELP n]`，
			// 只把 option-name 放进标签（fgloutline.ts:366-373）。
			if cmd := match(reCommandHead, full); cmd != nil {
				label = normalize(strings.ToUpper(cmd.group("kw")) + " " + cmd.group("opt"))
			}

			node := &rawNode{
				kind:      KindSub,
				label:     label,
				startLine: i,
				endLine:   i,
				selStart:  ms.start(),
				selEnd:    ms.end(),
			}
			stack[len(stack)-1].node.children = append(stack[len(stack)-1].node.children, node)
			// SUB 也必须是栈帧：MENU 会真实地嵌在 ON ACTION 里。
			stack = append(stack, frame{kind: "SUB", node: node})
			continue
		}

		/* --- 块开头 --- */
		mo := match(reOpen, line)
		if mo == nil {
			continue
		}
		if mo.has("input") && reInputWrapGuard.MatchString(line) {
			// 对应 TS 里 input 分支的 `(?!W_P(?:NO W_P)?WRAP NB)`：守卫命中则该分支失败。
			continue
		}
		// 检测用 code（字符串已抹），标签用 disp（保留字符串原文，例如 MENU 标题）。
		// 两份等长同构，匹配的下标与跨度完全一致，只差字符串内容。
		md := match(reOpen, disp[i])
		if md == nil {
			md = mo
		}
		d, ok := describe(md)
		if !ok {
			continue
		}
		// 匹配长度一律取**检测用**的 code 匹配（fgloutline.ts:394-412 用的就是 mo）：
		// disp 里保留了字符串原文，`MENU 'a\'b'` 这类行的 disp 匹配会比 code 匹配短一截。
		effEnd := mo.effEnd()

		if isModuleKind(string(d.kind)) {
			for len(stack) > 1 {
				closeTop(i - 1)
			}
		}

		node := &rawNode{
			kind:      d.kind,
			label:     d.label,
			startLine: i,
			endLine:   i,
			children:  nil,
		}
		if d.nameCol >= 0 {
			node.selStart = d.nameCol
			node.selEnd = d.nameCol + len(d.label)
		} else {
			node.selStart = mo.start()
			node.selEnd = effEnd
		}
		stack[len(stack)-1].node.children = append(stack[len(stack)-1].node.children, node)

		// 同行开闭，如 MENU "t" COMMAND "Quit" EXIT MENU END MENU
		if reSameLineEnd[d.kind].MatchString(line[effEnd:]) {
			node.endLine = i
		} else {
			stack = append(stack, frame{kind: string(d.kind), node: node})
		}
	}

	// EOF：仍未闭合的帧结束于此（裸语句块收成单行）。
	for len(stack) > 1 {
		closeTopRepair(len(code) - 1)
	}

	fixup(root)
	return root.children
}

/* fixup：只可能把父节点的 endLine 往大改，永不改子节点 —— 于是
 * child.range ⊆ parent.range 由构造保证（fgloutline.ts:423-444）。
 *
 * 用显式栈而非递归：嵌套足够深时递归会抛 Maximum call stack size exceeded
 * （正常语料最大深度只有 7，但粘贴一份畸形文件就能触发）。 */
func fixup(root *rawNode) {
	order := make([]*rawNode, 0, 64)
	stack := []*rawNode{root}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if n.endLine < n.startLine {
			n.endLine = n.startLine
		}
		order = append(order, n)
		stack = append(stack, n.children...)
	}
	// 前序的逆序 = 子节点必先于其父节点处理。
	for i := len(order) - 1; i >= 0; i-- {
		n := order[i]
		for _, c := range n.children {
			if c.endLine > n.endLine {
				n.endLine = c.endLine
			}
		}
	}
}

/* makeNode：转对外节点，行号 0-based → 1-based，并沿用 BDL 的钳位
 * （非法范围会被宿主拒绝，fgloutline.ts:446-460）。 */
func makeNode(nd *rawNode, raw []string) OutlineNode {
	endLine0 := nd.endLine
	if endLine0 > len(raw)-1 {
		endLine0 = len(raw) - 1
	}
	lineLen := 0
	if nd.startLine >= 0 && nd.startLine < len(raw) {
		lineLen = len(raw[nd.startLine])
	}
	selEnd := nd.selEnd
	if selEnd > lineLen {
		selEnd = lineLen
	}
	selStart := nd.selStart
	if selStart > selEnd {
		selStart = selEnd
	}
	return OutlineNode{
		Label:    nd.label,
		Kind:     nd.kind,
		Line:     nd.startLine + 1,
		EndLine:  endLine0 + 1,
		SelStart: selStart,
		SelEnd:   selEnd,
	}
}

// toOutlineNodes 同样用显式栈，理由见 fixup：深嵌套不得让遍历栈溢出（fgloutline.ts:462-478）。
func toOutlineNodes(roots []*rawNode, raw []string) []OutlineNode {
	nodes := make(map[*rawNode]*OutlineNode, len(roots)*4)
	order := make([]*rawNode, 0, len(roots)*4)
	stack := make([]*rawNode, 0, len(roots))
	for i := len(roots) - 1; i >= 0; i-- {
		stack = append(stack, roots[i])
	}
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		order = append(order, n)
		on := makeNode(n, raw)
		nodes[n] = &on
		for i := len(n.children) - 1; i >= 0; i-- {
			stack = append(stack, n.children[i])
		}
	}
	// 逆前序 = 子节点先于父节点定稿。必须先定稿子节点，父节点才能把「已经带上孙节点」
	// 的子节点副本拷进去（Go 的值语义不像 TS 的对象引用会自动级联）。
	for i := len(order) - 1; i >= 0; i-- {
		n := order[i]
		if len(n.children) == 0 {
			continue
		}
		kids := make([]OutlineNode, 0, len(n.children))
		for _, c := range n.children {
			kids = append(kids, *nodes[c])
		}
		nodes[n].Children = kids
	}
	out := make([]OutlineNode, 0, len(roots))
	for _, r := range roots {
		out = append(out, *nodes[r])
	}
	return out
}

/* splitLines 复刻 JS 的 `text.split(/\r\n|\r|\n/)`（fgloutline.ts:486-488）：
 * 与 Monaco / VS Code 的文档行模型一致。用 split('\n') 会让 CRLF 文件的行尾
 * 留着 `\r`，含孤立 `\r` 的文件还会整体偏 1 行。 */
func splitLines(text string) []string {
	out := make([]string, 0, strings.Count(text, "\n")+1)
	start := 0
	for i := 0; i < len(text); {
		switch text[i] {
		case chCR:
			out = append(out, text[start:i])
			if i+1 < len(text) && text[i+1] == chLF {
				i += 2
			} else {
				i++
			}
			start = i
		case chLF:
			out = append(out, text[start:i])
			i++
			start = i
		default:
			i++
		}
	}
	out = append(out, text[start:])
	return out
}

// ParseOutline 解析 4GL 源码为大纲树（fgloutline.ts:480-491）。
//
// 传入的应是**未做行号补偿**的原文：调试页顶部前插的空行不计入 Line，
// 跳转时由调用方自行加上偏移。
func ParseOutline(text string) []OutlineNode {
	nodes, _ := parseOutlineMasked(text)
	return nodes
}

// parseOutlineMasked 顺带把掩码后的 code 行交给调用方（block.go 的信封判定要按
// 「注释不算代码」的口径去找 END 行，复用同一份掩码可避免两套词法漂移）。
func parseOutlineMasked(text string) ([]OutlineNode, []string) {
	raw := splitLines(text)
	code, disp := maskLines(text)
	return toOutlineNodes(scanBlocks(code, disp), raw), code
}
