package debug

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ---------- fgldb 协议锚点正则(全部来自 192.0.2.109 真机实测输出) ----------
// 终端共享件(LineParser/提示符判定/ANSI 剥离)已迁 tt/internal/host 包。

var (
	// 断点命中:`Breakpoint 1, asf_bsft001_wf.bsft001_wf_construct() at asf_bsft001_wf.4gl:4452`
	reBreakHit = regexp.MustCompile(`^Breakpoint (\d+), (.+?) at (.+?):(\d+)\s*$`)
	// 人工中断(SIGINT 后;终端 ^C 回显可能与字样合并为一行,用子串匹配)
	reInterrupt = regexp.MustCompile(`INTERRUPT`)
	// 步进/位置头:`cl_ap_code_fuzzyquery() at lib_cl_ap_code.4gl:4593`(函数变化时出现)
	reStopHeader = regexp.MustCompile(`^([A-Za-z_][\w.]*)\(\) at (.+?):(\d+)\s*$`)
	// 调用栈帧:`#0 asf_bsft001_wf.bsft001_wf_construct() at asf_bsft001_wf.4gl:4452`
	reFrame = regexp.MustCompile(`^#(\d+)\s+(.+?)\s+at\s+(.+?):(\d+)\s*$`)
	// 断点设置成功:`Breakpoint 1 at 0x00000000: file asf_bsft001_wf.4gl, line 4452.`
	reBPSet = regexp.MustCompile(`^Breakpoint (\d+) at \S+: file (.+), line (\d+)\.`)
	// 监视点设置成功(A20):GDB 原样 `Hardware watchpoint 2: g_total`;
	// 没有 file:line,位置就是被监视的表达式本身(存在 Breakpoint.Func 里)。
	reWatchSet = regexp.MustCompile(`^(?:Hardware |Software )?(?:[Aa]ccess |[Rr]ead )?watchpoint (\d+): (.+?)\s*$`)
	// print 结果:`$1 = "CNJ-ICC-250800000012"`(记录/数组为多行)
	rePrintVal = regexp.MustCompile(`^\$(\d+) = (.*)$`)
	// 变量不存在:`No symbol "lp_str" in current context.`
	reNoSymbol = regexp.MustCompile(`No symbol "(.+)" in current context`)
	// 恢复运行
	reContinuing = regexp.MustCompile(`^Continuing\.`)
	// 停站上下文中的源码行:`   4452   LET ...` / `-> 4452   LET ...`
	reSource = regexp.MustCompile(`^(->)?\s*(\d+)\s{2,}(.*)$`)
	// 步进的紧凑停站(源码不可用时):`89	 in lib_cl_ap.4gl`
	reCompactStep = regexp.MustCompile(`^(\d+)\s+in\s+(\S+)\s*$`)
	// ---- info breakpoints 的真实行格式(A27,主机109 实测原文;续行见 A30 / A33) ----
	//
	//	Num Type           Disp Enb Address    What
	//	1 watchpoint keep y  g_current_row
	//	2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141
	//		stop only if 1=1
	//		ignore next 1 hits
	//		breakpoint already hit 1 time
	//
	// 五条铁律(全部来自实测,不是照 GDB 文档抄的):
	//   - watchpoint 行**没有 `hw`** 字样、Address 列为空 ⇒ 解析不许依赖 `hw`(带 hw 的老格式仍容忍);
	//   - breakpoint 行形如 `0x00000000 in <模块> at <文件>:<行>`;
	//   - **`Address` 恒是 `0x00000000`(A33 第四次真机:命中的那条也是 0) ⇒ 它不是"装没装上"的
	//     判据,不得用于判定"断点是否生效";命中与否只看 Hits(那条 `already hit` 续行)与停站事件**;
	//   - `ignore` / `stop only if` / `already hit` 三种回显都是**独立缩进续行**(制表符开头)
	//     ⇒ 必须能并进上一条,不能只按行切;
	//   - 条件断点**多一行** `stop only if <expr>`(A30)、命中过的断点**多一行**
	//     `breakpoint already hit N time(s)`(A33,单复数都要认) ⇒ 三种续行都要认,且**顺序不定**。
	reBPInfoWatch = regexp.MustCompile(`^(\d+)\s+(?:hw\s+|Hardware\s+|Software\s+)?watchpoint\s+(\S+)\s+([yn])\s*(.*)$`)
	// Num Type Disp Enb Address What —— 列头不是记录
	reBPInfoHeader = regexp.MustCompile(`^\s*Num\s+Type\s+Disp\b`)
	// `No breakpoints or watchpoints.` —— 空表,**不是**格式错误
	reBPInfoEmpty = regexp.MustCompile(`^No breakpoints or watchpoints`)
	reBPInfoBreak = regexp.MustCompile(`^(\d+)\s+breakpoint\s+(\S+)\s+([yn])\s*(.*)$`)
	// breakpoint 行里的 `in <模块> at <文件>:<行>`
	reBPInfoIn = regexp.MustCompile(`in\s+(\S+)\s+at\s+(.+):(\d+)\s*$`)
	// 缩进的 `ignore next N hits`(也可能是 crossings / hit)
	reBPInfoIgnore = regexp.MustCompile(`^\s+ignore next (\d+)\s+(?:hits|crossings|hit)\s*$`)
	// 缩进的 `stop only if <expr>`(A30,条件断点的回显) —— 与 ignore 一样是**续行**,
	// 要并进上一条断点的 Cond。空条件(`stop only if` 后面什么都没有)**不算**
	// 这条续行:让它掉到最后那条"行格式不认识"里,原文一并留下。
	reBPInfoStopOnly = regexp.MustCompile(`^\s+stop only if (.+)$`)
	// 缩进的 `breakpoint already hit N time(s)`(A33,第四次真机原文:命中过的断点才打印这一行,
	// 未命中过的**不打印**)。单复数都认:`1 time` / `2 times`。
	// 它是**第三种续行**,与 `stop only if` / `ignore next` 并列,顺序不定。
	reBPInfoHits = regexp.MustCompile(`^\s+breakpoint already hit (\d+)\s+times?\s*$`)
	// info locals 变量行:`lp_str = "CNJ-ICC-250800000012"`(记录/数组值可续行)
	reLocalVar = regexp.MustCompile(`^([A-Za-z_][\w.]*)\s*=\s*(.*)$`)
	// info variables 声明行(fgldb 实测为 `Globals:` 头 + `名字 类型` 两列,非 name=value)
	reGlobalDecl = regexp.MustCompile(`^([A-Za-z_][\w.]*)\s+([A-Za-z_][\w.\[\], ]*)$`)
	// info functions 函数行:`bsft001_wf_construct ()`
	reFuncList = regexp.MustCompile(`^([A-Za-z_][\w.]*)\s*\(\s*\)\s*$`)
	// info line 位置行:`Line 801 of "asf_bsft001_wf.4gl" starts at address 0x0 <main+0>`
	reInfoLine = regexp.MustCompile(`^Line (\d+) of "([^"]+)"`)
	// until 目标行不存在:`No line 999999 in file asf_bsft001_wf.`
	reNoLine = regexp.MustCompile(`^No line \d+ in file`)
	// until 目标文件不存在:`No source file named bsft001_wf.4gl.`
	reNoSourceFile = regexp.MustCompile(`^No source file named`)
	// 程序退出:`Program exited normally.` / `Program exited with code N.`(作业窗口被关闭或正常结束)
	reProgramExited = regexp.MustCompile(`^Program exited`)
	// reNotRunning fgldb 在 pre-run 态拒绝放行类命令时回的话。
	// 单独具名是因为它还要用在一处**状态回滚**上(见 session.go 的 matchFdbErr 调用点):
	// 命令被拒了,可 execOpt 发命令时已经把状态乐观翻成了 running。
	reNotRunning = regexp.MustCompile(`^(The program is not being run|Program not being run)`)
)

// fdbErrRes 已知的调试器错误行(出现在响应中时转成命令错误)
var fdbErrRes = []*regexp.Regexp{
	reNoSymbol,
	reNotRunning,
	regexp.MustCompile(`^No stack\.`),
	regexp.MustCompile(`^Cannot execute this command`),
	regexp.MustCompile(`^invalid argument`),
	reNoLine,
	reNoSourceFile,
}

// resumeCmds 会**让程序跑起来**的命令:发出后程序就不在调试器上了,直到下一次停站
// (或者一直不停)。用于两处关键判定:
//   - 发出后必须如实把会话状态翻成 running(否则状态在说谎:界面上还显示"已停站",
//     interrupt 也会被自己的 state==Running 检查拒掉,卡住时连中断都发不出去)
//   - 只有这类命令在飞时,才把带 `->` 的源码行当成一次停站(见 onLine)
//
// 含 fgldb 的常用缩写;内部调用方(Step/Continue/Run)用的都是全名,缩写只有 /raw 透传才可能出现。
var resumeCmds = map[string]bool{
	"run": true, "continue": true, "c": true,
	"next": true, "step": true, "s": true,
	"until": true, "finish": true,
}

// IsResumeCmd 判断一条 fgldb 命令是否会放行程序(只看首个词,忽略参数)
func IsResumeCmd(cmd string) bool {
	f := strings.Fields(strings.TrimSpace(cmd))
	if len(f) == 0 {
		return false
	}
	return resumeCmds[strings.ToLower(f[0])]
}

// bpCmds 会改动断点集合的 fgldb 命令。
//
// 为什么要单列:Ai 走 `exec "break X"`(即 /raw,kind="other"),而断点缓存 s.bps
// 只由 `exec(kind="break")` 那条路径维护(onLine 按 pending.kind 分派解析)。
// 不透传这条命令的话,AI 下的断点**在会话里等于不存在** —— 快照里是空的(前端断点列表
// 空白)、persistBPs 也读不到它(存不下来)。
var bpCmds = map[string]bool{
	"break": true, "tbreak": true, "clear": true,
	"delete": true, "disable": true, "enable": true,
}

// IsBreakpointCmd 判断一条 fgldb 命令会不会改动断点集合(只看首个词)
func IsBreakpointCmd(cmd string) bool {
	f := strings.Fields(strings.TrimSpace(cmd))
	if len(f) == 0 {
		return false
	}
	return bpCmds[strings.ToLower(f[0])]
}

// bpReadSub 是 `info` 之后那个词的判定：GDB 系调试器按**前缀缩写**收子命令
// （`info b` / `info break` / `info breakpoints` 等价），这里就按前缀认。
//
// 为什么前缀是安全的：`info` 的子命令表里**只有 `breakpoints` 以 b 开头**
// （`locals` / `variables` / `functions` / `line` / `frame` … 都不冲突，见 `help` 实测表）。
var bpReadSub = "breakpoints"

// IsBreakpointReadCmd 判断一条 fgldb 命令是不是**读**断点表：`info breakpoints`
// （含缩写 `info b` / `info break`，含 `info breakpoints 3` 这种带编号写法）。
//
// 为什么与 `IsBreakpointCmd` 分开，而不是把 `info` 塞进 `bpCmds`（A36 后半，二选一里选的
// 就是"单独判断"）：
//   - `bpCmds` 的语义是「会**改动**断点集合」（它的注释、`/raw` 里那个 `act = "bp."+…`
//     的事件名都建立在这条语义上）。`info breakpoints` 一个字节都不改 —— 塞进去等于让读
//     命令冒充写命令：事件流里会平白出现 `bp.info`，前端按 action 分支时无从判定它其实是只读的；
//   - 影响面：单独判断只影响 `hRaw` 的一个分支，不会牵连"别的 info 子命令是否要回灌断点账本"
//     （`info locals` / `info variables` 读了也不该动账本）；
//   - 既有测试 `TestIsBreakpointCmd` 明确断言 `info breakpoints` **不是**断点类命令 ——
//     改 bpCmds 就得改那条既有判据，而"既有行为不破"是本轮纪律。
//
// 它只描述**命令形状**，不描述"该不该回灌"：运行态不能发命令（A34），那一条由调用方按
// 会话状态判（见 hRaw）。
func IsBreakpointReadCmd(cmd string) bool {
	f := strings.Fields(strings.TrimSpace(cmd))
	if len(f) < 2 || !strings.EqualFold(f[0], "info") {
		return false
	}
	sub := strings.ToLower(f[1])
	return sub != "" && strings.HasPrefix(bpReadSub, sub)
}

// matchFdbErr 返回匹配到的错误行文本
func matchFdbErr(ln string) string {
	for _, re := range fdbErrRes {
		if re.MatchString(ln) {
			return ln
		}
	}
	return ""
}

// ---------- 交互语句分类 ----------
//
// fgldb 协议看不到前端界面:程序执行到交互语句就会阻塞在用户界面上等人操作,
// 而调试器里它只是 running —— 与死循环在协议层无法区分。唯一可靠的证据是
// 「程序停在哪一行」:停在这类语句上,就是在等用户。
// 官方文档两处实证:attach 示例直接打出 `108  DISPLAY ARRAY contlist TO sr.*`,
// Ctrl-C 示例打出 `-> 2  MENU "Test"`。
//
// 判定顺序有讲究:复合形式必须排在裸形式之前(INPUT ARRAY / INPUT BY NAME 先于 INPUT)。
var interactiveKeywords = []struct {
	kind string
	re   *regexp.Regexp
}{
	{"input_array", regexp.MustCompile(`(?i)^INPUT\s+ARRAY\b`)},
	{"input_by_name", regexp.MustCompile(`(?i)^INPUT\s+BY\s+NAME\b`)},
	{"input", regexp.MustCompile(`(?i)^INPUT\b`)},
	{"display_array", regexp.MustCompile(`(?i)^DISPLAY\s+ARRAY\b`)},
	{"menu", regexp.MustCompile(`(?i)^MENU\b`)},
	// DIALOG 是 Genero 的通用交互指令(INPUT/MENU/CONSTRUCT 的现代写法),
	// 也是 T100 客制里最常见的那种 —— 真机验证时程序正停在
	// `DIALOG ATTRIBUTES(UNBUFFERED,FIELD ORDER FORM)` 上,漏了它这一整类就全判不出来。
	{"dialog", regexp.MustCompile(`(?i)^DIALOG\b`)},
	{"construct", regexp.MustCompile(`(?i)^CONSTRUCT\b`)},
	{"prompt", regexp.MustCompile(`(?i)^PROMPT\b`)},
	{"window", regexp.MustCompile(`(?i)^OPEN\s+WINDOW\b`)},
}

// ClassifyInteractiveLine 判断一行 4GL 源码是否为「会停下来等用户操作」的交互语句,
// 返回语句类别(input|input_by_name|input_array|display_array|menu|construct|prompt|window),
// 不是交互语句返回空串。
// 先剥字符串与注释再匹配,避免 `DISPLAY 'INPUT'`、`#add-point:...INPUT` 之类误命中。
func ClassifyInteractiveLine(text string) string {
	code := stripQuoted(text)
	if i := strings.Index(code, "--"); i >= 0 { // 4GL 行注释
		code = code[:i]
	}
	if i := strings.Index(code, "#"); i >= 0 { // 预处理指令(#add-point 之类),与 lineVarNames 同款截断
		code = code[:i]
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	for _, k := range interactiveKeywords {
		if k.re.MatchString(code) {
			return k.kind
		}
	}
	return ""
}

// CurSourceText 取停站源码块里当前行(`->` 箭头那行)的原文;无当前行标记时返回空串。
func CurSourceText(src []SourceLine) string {
	for _, sl := range src {
		if sl.IsCur {
			return sl.Text
		}
	}
	return ""
}

// ---------- info breakpoints 解析(A27 + A30 + A33) ----------

// ParseInfoBreakpoints 解析 `info breakpoints` 的输出为断点账本(**纯函数**,不 IO、不碰 *Session)。
//
// 认两种记录行 + 三种缩进续行:
//
//	<N> watchpoint <Disp> <Enb>  <表达式>                             ← 无 hw、无地址(A27 实测)
//	<N> breakpoint <Disp> <Enb> <地址> in <模块> at <文件>:<行>
//	\tstop only if <expr>                                            ← 条件断点的回显(A30) ⇒ Cond
//	\tignore next <n> hits                                           ← 忽略次数(A27) ⇒ Ignore
//	\tbreakpoint already hit <n> time(s)                             ← 命中计数(A33) ⇒ Hits
//
// 续行**顺序不定**(先 ignore 后 stop only if、命中计数夹在中间都行):三种都按"并进上一条"处理,
// 不按固定 if-else 链切 —— 顺序是调试器实现的细节,不是契约。
//
// **`Address` 恒为 `0x00000000`(A33:命中的断点也是 0)⇒ 它不是"装没装上"的判据**,
// 本函数只是把它跳过去。命中与否只看 Hits 与停站事件。
//
// 解析不出来(既不是记录、也不是已知的列头/空表/回显/续行)⇒ 返回错误,且**错误里带原文**:
// 静默吞掉是最坏的处理 —— 现场拿到一张空断点表,却没有任何线索说为什么。
func ParseInfoBreakpoints(lines []string) ([]Breakpoint, error) {
	var out []Breakpoint
	last := -1 // 上一条记录的索引,`stop only if` / `ignore` 续行要并进它
	for _, raw := range lines {
		ln := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(ln)
		switch {
		case trimmed == "":
			continue // 空行
		case strings.HasPrefix(trimmed, "(fgldb)"):
			continue // 提示符残行
		case reBPInfoHeader.MatchString(ln):
			continue // 列头
		case reBPInfoEmpty.MatchString(trimmed):
			continue // 空表提示
		case trimmed == "info breakpoints" || strings.HasSuffix(trimmed, " info breakpoints"):
			continue // 命令回显
		}
		// 续行 ①:`stop only if <expr>` ⇒ 并进上一条的 Cond(A30)
		if m := reBPInfoStopOnly.FindStringSubmatch(ln); m != nil {
			if expr := strings.TrimSpace(m[1]); expr != "" {
				if last < 0 {
					return nil, fmt.Errorf("行格式不认识（info breakpoints）：%q —— 这是 `stop only if` 续行,"+
						"但它上面没有断点记录;原文保留: %q", ln, ln)
				}
				out[last].Cond = expr
				continue
			}
		}
		// 续行 ②:`ignore next N hits` ⇒ 并进上一条的 Ignore(A27)
		if m := reBPInfoIgnore.FindStringSubmatch(ln); m != nil {
			if last < 0 {
				return nil, fmt.Errorf("行格式不认识（info breakpoints）：%q —— 这是 ignore 续行,但它上面没有断点记录;"+
					"原文保留: %q", ln, ln)
			}
			n, _ := strconv.Atoi(m[1])
			out[last].Ignore = n
			continue
		}
		// 续行 ③:`breakpoint already hit N time(s)` ⇒ 并进上一条的 Hits(A33)。
		//
		// 这是"命中过没有"的**唯一**报文判据 —— `Address` 恒 0,看不出任何东西。
		// 未命中过的断点**不打印**这一行,所以"没有这一行"= Hits 0,不是解析失败。
		if m := reBPInfoHits.FindStringSubmatch(ln); m != nil {
			if last < 0 {
				return nil, fmt.Errorf("行格式不认识（info breakpoints）：%q —— 这是命中计数续行"+
					"(`breakpoint already hit N time(s)`),但它上面没有断点记录;原文保留: %q", ln, ln)
			}
			n, _ := strconv.Atoi(m[1])
			out[last].Hits = n
			continue
		}
		// 监视点行(无 hw;带 hw 的老格式也认)
		if m := reBPInfoWatch.FindStringSubmatch(ln); m != nil {
			num, _ := strconv.Atoi(m[1])
			out = append(out, Breakpoint{
				Num: num, Kind: BPKindWatch, Func: strings.TrimSpace(m[4]),
				Enabled: m[3] == "y",
			})
			last = len(out) - 1
			continue
		}
		// 断点行:`… in <模块> at <文件>:<行>`
		if m := reBPInfoBreak.FindStringSubmatch(ln); m != nil {
			num, _ := strconv.Atoi(m[1])
			w := reBPInfoIn.FindStringSubmatch(m[4])
			if w == nil {
				return nil, fmt.Errorf("行格式不认识（info breakpoints）：%q —— breakpoint 行里没有 "+
					"`in <模块> at <文件>:<行>`;原文保留: %q", ln, ln)
			}
			line, _ := strconv.Atoi(w[3])
			out = append(out, Breakpoint{
				Num: num, Kind: BPKindBreak, Module: w[1], Func: w[1],
				File: w[2], Line: line, Enabled: m[3] == "y",
			})
			last = len(out) - 1
			continue
		}
		// 剩下的都算不认识 —— 报错并保留原文
		return nil, fmt.Errorf("行格式不认识（info breakpoints）：%q —— 原文保留: %q;"+
			"已知形态是 `1 watchpoint keep y  <表达式>` 与 `2 breakpoint keep y 0x… in <模块> at <文件>:<行>`"+
			"(可跟三行缩进续行：`stop only if <expr>` / `ignore next N hits` / "+
			"`breakpoint already hit N time(s)`，顺序不定)", ln, ln)
	}
	return out, nil
}
