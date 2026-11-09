package testkit

// T2.3 静态检查：**全仓库不许再出现"取第一个会话"**。
//
// 样板是 `single_source_test.go` 的 `TestHelpersHaveNoLocalCopies`（"同一件东西不许有第二份实现"
// 的扫法），这里是它的同族问题：**同一件东西不许有第二个来源** ——
// "哪条会话"这件事只许有一个来源（目标层），不许散落成"取列表第一个"。
//
// ## 为什么是"扫源码"而不是"跑起来看"
//
// 这条规矩的失败方式是**静默的**：多会话下取第一个 = 命令跑到另一条会话上，
// 而输出看起来一切正常。计划开头把它写成了一句可验证的话：
//
//	**不会出现"命令跑在另一条会话上却看起来成功"；落不上目标时，一定是报错。**
//
// 跑测试验不了这句话（要真 SSH + 两条真会话 + 并发），但**数点**能验：
// 每个"取第一个会话"的点都是一个可能静默落错的地方，数清楚它、只许减少不许增加。
//
// ## 豁免规则（**唯一的豁免**，写在注释里也写在代码里）
//
// 今天的合法来源是**会话池自己的实现**：`internal/debug/manager.go` 里的
// `func (m *Manager) Current() *Session`。契约 B11 / C7.1 的要求是
// **服务端那五处直呼点全删掉**（`api.go` 的 367 / 816 / 1679 / 1724 / 1754），
// `Current()` 这个方法本身在 WP3/WP4 期间还需要存在（闸门还在用它），
// 所以：
//
//   - **定义处**（`.go` 里 `func (… *…Manager) Current() *Session {` 这一行）豁免；
//   - **调用处一律不豁免** —— 包括 `internal/debug` 内部自己调的（那正是 B11 的五处）；
//   - `sessions[0]` 这种直接下标**一律不豁免**（取第一个会话的那一族，
//     两类写法都已经在 WP4a 里被目标层取代，见下面普查表里那两条 `min: 0`）。
//
// 另一条能自证的纪律：`Current()` 的**接收者必须还是会话池**。
// 今天 `rg -n '\.Current\(\)' internal/` 恰好 5 处、全是 `s.mgr.Current()`。
// 将来别人给别的类型加一个 `.Current()`，本条会把接收者打出来让人自己判断 ——
// 那**不该**由我一条正则替他决定（而且 `sessOf` 那族才是做对了的写法，见 B11b）。
//
// ## 今天它是**断言**（WP4b 把最后 5 处也清掉了）
//
// WP2 交付时全仓库还有 25 处（那三族写法的基线），所以这条测试当时**只是跳过**：
// 它数完、把命中数逐处打进 `t.Skipf` 的文案，进度靠"文案里的数字 + 跳过台账"被看见
// （每迁一批，那个数字就变小）。WP4a 迁掉 CLI 侧 19 处，只剩服务端五个
// `Manager.Current()` 兜底；**WP4b 删掉那五处之后命中数到 0** —— 于是跳过消失，
// 这条测试变成**真断言**：再出现一处"取第一个会话"就是红的。
// 台账里那条 `layPending` 也随之删掉（它等的东西已经交付）。
//
// **两条不许动的纪律**：
//
//  1. **不要为了让测试变绿去改这个文件里的基线** —— 基线是"今天的事实"；
//  2. **不要为了放行它去改构建脚本或加环境变量开关** —— `make test` 是主闸门，
//     闸门迁就测试等于把闸门拆了。
//
// 命中数被两处守着：这里（`min` 那一列 + 下面那句断言）+ 跳过台账
// （`skipledger_test.go`：WP4b 之后这一族**不再有**台账条目，因为它不再跳过 ——
// "不再跳过"这件事本身由台账的"少一处"那条断言反向盯着）。

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// firstSourceCase 是一族"取第一个会话"的写法。
type firstSourceCase struct {
	name string
	// anchor 是这一族写法的**字面锚点**，用来在源码里找到候选位置。
	//
	// 为什么要单独给一个锚点，而不是让 `re` 自己去扫：这一族的正则**没有**右边界，
	// 于是 `FindAllIndex`（不重叠）会在
	// `sessions[0].ID, sessions[0].Env, sessions[0].State` 这种一行多处的写法上少数
	// —— 后两次的起始位置落在上一次的匹配区间里，被吃掉了。
	// 数少的后果是"还没清干净"看起来像"已经清干净"。
	//
	// 用**字面串**扫就不存在这个问题（`Index` 从上一个匹配的**末尾**继续找，
	// 而锚点串本身在每一处都完整出现）。找到之后再让 `re` 从那个位置核一遍，
	// 保证"数到的确实是这一族"。
	anchor string
	// re 是这一族写法的**完整形状**：从锚点起点起必须匹配（模式以 `^` 锚定）。
	// 注意锚点**不**等于正则要匹配的整段：`Current()` 那一族的左边界是"点前面有一个
	// 接收者"，扫描时从锚点（那个点）切下去，正则自己要把接收者吃回来。
	// 所以取数法的自检用 `selfSample`（= samplePrefix + anchor）而不是光 anchor。
	re *regexp.Regexp
	// min 是今天的实测命中数（出现次数，含豁免）—— **只许减少，不许增加**。
	min int
	// exempt 是今天被豁免的条数（写死；多一条就要有人解释为什么）。
	exempt int
	// why 是"这条为什么是坏味道 + 该怎么改"，会原样进失败信息。
	why string
}

// filesMatching 返回 files 里**匹配 re 的文件**（仓库相对路径）。用于"某个东西还在不在"的地基确认。
func filesMatching(t *testing.T, files []string, root string, re *regexp.Regexp) []string {
	t.Helper()
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if !re.Match(b) {
			continue
		}
		rel, relErr := filepath.Rel(root, f)
		if relErr != nil {
			rel = f
		}
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}

// isIdentByte 判断一个字节是否算"标识符的一部分"**（不含 `.`）**，用于左边界判断。
//
// 为什么把 `.` 排除在外：`Current()` 那一族的锚点从 `C` 起，调用处的前一个字节是 `.`
// （`s.mgr.Current()`）。若不排除，调用点会被左边界判据全部挡掉，而定义处
// （`func (m *Manager) Current()`，前面是空格）反而留下 —— 恰好把唯一该豁免的那条
// 和该数的那五条**搞反**。区分调用与定义交给 `re` 里的接收者那一段。
func isIdentByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// TestFirstSourceCensusProbeIsSound 是普查"锚点 × 正则"这套取数法的自检。
//
// 它是整条 T2.3 的地基：数错了，后面所有"还剩多少处"都是错的 ——
// 而数错的两种方向都很隐蔽：
//
//   - **漏数**：一行里连续出现三次只数到一次（不重叠匹配的通病）→
//     "还没清干净"看起来像"已经清干净"，T2.3 的全部价值当场归零；
//   - **多数**：把 `xdbgCurrentID` 或注释里提到的名字也算进来 → 数字虚高，
//     人会以为"要改的比实际多"，改到后来对不上账。
//
// 所以这里用一个小样本把两件事都钉住：**每一处都数到、且只数到该数到的**。
func TestFirstSourceCensusProbeIsSound(t *testing.T) {
	// 第一件事：**每个锚点本身必须能被自己的正则匹配** ——
	// 扫描是"从锚点位置切下去再看正则的"，锚点对不上就一处也数不到。
	// 这条断言把"锚点取哪儿、正则从哪儿切"这件事钉死（手写正则时最容易在这儿出错：
	// 把 `^` 理解成"行首"、或锚点与正则各取了一半）。
	for _, c := range firstSourceCensus {
		if !c.re.MatchString(c.anchor) {
			t.Errorf("普查「%s」的锚点 %q 不能被自己的正则 %v 匹配 —— "+
				"扫描时是把正则在**锚点位置**上切的，锚点都对不上就一处也数不到。",
				c.name, c.anchor, c.re)
		}
	}

	sample := "func f() {\n" +
		"\ta := sessions[0].ID\n" + // 1 处
		"\tb, c := sessions[0].Env, sessions[0].State\n" + // 一行两处
		"\td, e, g := sessions[0].ID, sessions[0].Env, sessions[0].State\n" + // 一行三处
		"\tid, _ := dbgCurrentID()\n" + // 1 处
		"\tif !dbgCurrentIDExists {}\n" + // 前缀相同的**另一个**标识符：不该数
		"\tcur := s.mgr.Current()\n" + // 1 处
		"\tother := s.mgr.Current()\n" + // 1 处
		"}\n"
	want := map[string]int{
		"sessions[0]":   6, // 1 + 2 + 3
		"dbgCurrentID":  1, // 只数真正那个名字
		"mgr.Current()": 2, // 两处调用
	}
	for _, c := range firstSourceCensus {
		if got := countAnchored(sample, c); got != want[c.name] {
			t.Errorf("普查「%s」在样本上数到 %d 处，应当是 %d 处。\n"+
				"    样本是刻意造的：一行一处 / 一行两处 / 一行三处 / 前缀相同的另一个标识符。\n"+
				"    数错的方向两种都坏：漏数让『还没清干净』看起来像『已经清干净』；\n"+
				"    多数让进度条虚高，改到后来对不上账。", c.name, got, want[c.name])
		}
	}
}

// countAnchored 数 `anchor` 在 text 里出现、且**该处 `re` 也匹配**的次数（供上面的自检使用）。
//
// `from++` 而不是 `from = pos + len(anchor)`：后者会漏掉**相邻**的两次出现
// （`sessions[0].ID, sessions[0].Env` 里第二处的 `s` 紧挨着第一处匹配的末尾之后，
// 但 `Index` 是从上一个匹配的**末尾**继续找，只要锚点串本身没有被跨过就不会漏；
// 逐字节推进是"宁可慢一点也不数错"的写法 —— 这条检查的全部价值就是这个数）。
func countAnchored(text string, c firstSourceCase) int {
	if c.anchor == "" {
		return 0
	}
	n := 0
	for from := 0; from+len(c.anchor) <= len(text); {
		i := strings.Index(text[from:], c.anchor)
		if i < 0 {
			return n
		}
		pos := from + i
		if pos == 0 || !isIdentByte(text[pos-1]) {
			if c.re.MatchString(text[pos:]) {
				n++
			}
		}
		from = pos + 1
	}
	return n
}

// firstSourceCensus 是这几族写法的**普查基线**。
//
//	min 是脚本数出来的（`internal/` 下**非测试**的 .go，按"出现次数"逐族统计，不是我估的）。
//	WP2 交付时的基线（三族合计 25 处）：
//
//		取当前会话那族       →  9 处（debugctl.go 6 + debugctx.go 3；其中 1 处是 doc 注释那行）
//		直接下标取第一条那族 → 10 处（debugctl.go 3 + debugenv.go 7，含一行两处与一行三处）
//		mgr.Current()        →  5 处调用（api.go 的 367/816/1679/1724/1754，B11）+ 定义处 1 处（豁免）
//		合计                 → 25 处命中（不含豁免），其中 **1 处豁免**（会话池定义行）
//
//	**WP4a 之后**：前两族在**生产代码**里一处不剩，所以它们的 `min` 从 9/10 改成 **0**；
//	第三族那 5 处一行未动（那是 WP4b 的活）。合计 25 → **5**。
//
//	**WP4b 之后（今天）**：第三族的 5 处调用也全部删除 —— 服务端的隐式寻址兜底清零
//	（Q4 / C8.2 硬切落地，见 `internal/debug/api_target.go`）。于是：
//
//		mgr.Current() → `min` 从 6 改成 **1**（只剩会话池那个**定义行**，它是唯一豁免）
//		合计          → **0 处调用**（`exempt` 那一列的 1 不计进来）
//
//	**为什么退役的那两族不删掉、而是留成 `min: 0`**：它们从"今天要清的账"变成
//	"**不许再出现**"的判据 —— 有人哪天再写一句"取列表第一个"，下面那条断言当场红。
//
// 豁免那一条**不进普查**：它的锚点 `mgr.Current()` 找不到会话池那个方法的定义行
// （那是另一族写法），所以由 `reCurrentDefinition` 单独确认，`exempt` 那一列的 1 就是它。
//
// 为什么逐"出现次数"数而不是逐"行"数：从前 debugenv.go 有一行里三处下标，
// 逐行数会漏掉两处，于是"还没清干净"看起来像"已经清干净"。
var firstSourceCensus = []firstSourceCase{
	{
		name:   "dbgCurrentID",
		anchor: "dbgCurrentID",
		re:     regexp.MustCompile(`^dbgCurrentID\b`),
		// WP4a：CLI 那个"取列表第一个"的入口已经由目标层取代（target.go 的 dbgSessionID），
		// 生产代码里一处不剩 —— 基线从 9 改成 0，这一族从此是"不许再出现"。
		min:    0,
		exempt: 0,
		why: "CLI 取当前会话 id 的旧入口（它**取列表第一个**，计划开头 A 表的 A 一行）。\n" +
			"    WP4a 已把它整个换成「向目标层要目标」（`internal/cli/debug/target.go`）。\n" +
			"    今天基线 0：再出现这个名字就是有人把旧写法搬回来了。",
	},
	{
		name:   "sessions[0]",
		anchor: "sessions[0]",
		re:     regexp.MustCompile(`^sessions\[0\]`),
		// WP4a：CLI 侧 10 处直接下标全部换成目标层落位的结果 —— 基线从 10 改成 0。
		min:    0,
		exempt: 0,
		why: "直接下标取第一条会话（写法 ①「唯一就落」的**旧**实现方式）。\n" +
			"    多会话下这个下标取到哪一条是完全不确定的 —— 与上一族同一个病。\n" +
			"    WP4a 已全部改成向目标层要目标（唯一就落住在 CLI，Q12/C8.1）。",
	},
	{
		name: "mgr.Current()",
		// 锚点取**完整的接收者 + 方法**（`mgr.Current()`），与 `re` 逐字对齐。
		//
		// 为什么不按"任意接收者 + `.Current()`"来数：锚点与正则必须指向**同一个东西**
		// （扫描是"从锚点位置切下去再看正则的"，而正则的 `^` 是**文本开头**）。
		// 从前全仓库那五处都是 `s.mgr.Current()`，方法那一段就是 `mgr.Current()`，
		// 所以锚点取它最直白；正则再钉一遍保证"数到的确实是这一族的调用"。
		// 左边界判据（前一个字节不能是标识符字符）照样生效：`xmgr.Current()` 不会被数进来。
		//
		// 会话池那个**定义**行写的是 `func (m *Manager) Current() *Session`
		// （没有 `mgr.` 这一段），所以锚点找不到它；它由豁免规则
		// `reCurrentDefinition` 单独认出来 —— 那是**唯一豁免**。
		anchor: "mgr.Current()",
		re:     regexp.MustCompile(`^mgr\.Current\(\)`),
		// WP4b：**5 处调用全部删除**（Q4 / C8.2：服务端不保留隐式寻址）。
		// 于是 `min` 从 6（5 处调用 + 1 处定义）改成 **1**（只剩那个定义行，它豁免）。
		min:    1,
		exempt: 1,
		why: "服务端在**没有目标**时回落「当前会话」（B11：恰好五处，C7.1 要求全删）。\n" +
			"    WP4b 已把那五处全删掉（`internal/debug/api_target.go` 的 resolveTarget / mustTarget\n" +
			"    把\"要哪条会话\"改成\"请求显式给，或池内唯一\"；>=2 条时报 SESSION_AMBIGUOUS 并列候选）。\n" +
			"    今天基线 1：只剩会话池自己的**定义处**（`manager.go` 的 `func (m *Manager) Current()`）。\n" +
			"    豁免：只有那一处定义豁免；**调用处一律不豁免** —— 包括 internal/debug 自己调的。",
	},
}

// reCurrentDefinition 是**唯一豁免**：会话池自己那个方法的定义行。
//
// 只匹配定义，不匹配调用；并且钉住接收者必须还是会话池（`*Manager`）。
// 用了别名 `.Current()` 的调用（如 `mgr.Current()` 赋给别的变量再点）不在豁免之列。
var reCurrentDefinition = regexp.MustCompile(`func\s*\([^)]*\*Manager\)\s*Current\(\)\s*\*Session`)

// reCurrentCall 与 currentCallReceiver 是这两族的"认出来给人看"的判据。
//
// 它们曾经只服务于旧的失败信息最后那一段（把 `.Current()` 的**接收者**列出来：
// 调用处不豁免，但接收者是谁要让人自己判断 —— 见 firstSourceCensus 里 `mgr.Current()` 那条 why）。
// 现在那条检查改成**跳过 + 台账登记**，跳过文案里改用逐处 `文件:行`（比只列接收者更可执行，
// 见 TestNoSecondSourceForCurrentSession），这两条正则就暂时没有调用方了。
// **留着它们是有意的**：WP4 迁完、total == 0 之后，收尾动作之一是"改成正向断言"，
// 那时要判的正是"`.Current()` 有没有被用在别的类型上" —— 判据就是这两条（改法写在上面那两行）。
var (
	reCurrentCall       = regexp.MustCompile(`\.Current\(\)\s*;?`)
	currentCallReceiver = regexp.MustCompile(`([A-Za-z_][\w.\[\]]*)\.Current\(\)`)
)

// TestNoSecondSourceForCurrentSession 就是那条扫描。
func TestNoSecondSourceForCurrentSession(t *testing.T) {
	root := RepoRoot(t)
	files := repoScanFiles(t, root)

	// 防空转：扫描坏了（路径变了、跳过规则写错）的话，下面所有比对都会"通过"。
	// 判据与 single_source_test.go 的 TestRepoRootFindsTheModule 同一个思路。
	if len(files) < 200 {
		t.Fatalf("只扫到 %d 个非测试 .go —— 扫描本身坏了（或跳过规则把 internal/ 也跳了），"+
			"此后的比对等于空转。仓库里非测试 .go 有 700+ 个。", len(files))
	}

	hits := map[string][]string{}
	allHits := map[string]int{}
	exempt := map[string]int{}

	// 唯一豁免那一条**不在普查里**（它的锚点找不到自己）—— 单独确认真身还在，
	// 并把它算进基线：`exempt` 那一列的 1 就是它。
	defSites := filesMatching(t, files, root, reCurrentDefinition)
	exempt["mgr.Current()"] = len(defSites)
	if len(defSites) == 0 {
		t.Errorf("找不到会话池自己的定义行（`func (m *Manager) Current() *Session`）——\n" +
			"    豁免规则是对着**它**写的。找不到只有两种可能：方法被删了（那太好了，请把这条\n" +
			"    普查连同豁免一起删掉），或者它改了名字/签名（那要同步 `reCurrentDefinition`）。")
	}
	if len(defSites) > 0 {
		allHits["mgr.Current()"] += 1
	}

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		text := string(b)
		rel, relErr := filepath.Rel(root, f)
		if relErr != nil {
			rel = f
		}
		rel = filepath.ToSlash(rel)

		for _, c := range firstSourceCensus {
			// 逐字节推进（`pos+1`）而不是跳到锚点末尾 —— 见 countAnchored 的注释。
			for from := 0; from+len(c.anchor) <= len(text); {
				i := strings.Index(text[from:], c.anchor)
				if i < 0 {
					break
				}
				pos := from + i
				from = pos + 1
				// 左边界：前一个字节不能是标识符字符（否则 `xdbgCurrentID` 也会被数进来）。
				// `Current()` 那一族从 `C` 起，调用时前一个字节是 `.`（也算标识符字符）——
				// 所以它靠 `re` 里的接收者那一段来区分调用与定义（见 firstSourceCensus 的注释）。
				if pos > 0 && isIdentByte(text[pos-1]) {
					continue
				}
				if !c.re.MatchString(text[pos:]) {
					continue
				}
				allHits[c.name]++
				// 唯一豁免：会话池自己那个定义行。
				if c.name == "mgr.Current()" && reCurrentDefinition.MatchString(lineAt(text, pos)) {
					exempt[c.name]++
					continue
				}
				hits[c.name] = append(hits[c.name], rel+":"+lineNoAt(text, pos))
			}
		}
	}

	// 基线与**豁免**对账：豁免的条数也必须与基线里写的一致 ——
	// 豁免行是**写死的一条**，多出一条只有两种可能：有人又加了一个 `Current()` 定义
	// （那不该被豁免），或者正则被改松了（那会让调用点被误免，检查当场失效）。
	// 这一条就是"豁免规则本身"的判据（`single_source_test.go` 的同族手法）。
	for _, c := range firstSourceCensus {
		wantExempt := c.exempt
		if exempt[c.name] != wantExempt {
			t.Errorf("「%s」的豁免条数 = %d，基线说该是 %d。\n"+
				"    豁免只有一条：`internal/debug/manager.go` 的 `func (m *Manager) Current() *Session`。\n"+
				"    多出来说明有人加了第二个 Current() 定义（该被数进来），或豁免正则 `reCurrentDefinition` 被改松了\n"+
				"    （那会让**调用点**被误免，这条检查当场失效）。",
				c.name, exempt[c.name], wantExempt)
		}
		if allHits[c.name] < c.min {
			t.Errorf("「%s」今天一共数到 %d 处（含豁免），基线是 %d 处 —— 有人**动过**这些点了。\n"+
				"    变小是好事，但基线必须同步更新：把 firstSourceCensus 里 %s 的 min 改成 %d，\n"+
				"    并把这次的改动写进提交说明（这条测试的作用就是让「少了一处」必须被看见）。",
				c.name, allHits[c.name], c.min, c.name, allHits[c.name])
		}
	}

	total := 0
	var report strings.Builder
	for _, c := range firstSourceCensus {
		got := hits[c.name]
		sort.Strings(got)
		total += len(got)
		if len(got) > 0 {
			report.WriteString("\n  【" + c.name + "】" + itoa(len(got)) + " 处：\n")
			for _, h := range got {
				report.WriteString("      " + h + "\n")
			}
		}
	}

	// ---- 到这里就是**真断言**（WP4b 之后不再跳过） ----
	//
	// WP2 交付时全仓库还有 25 处待清，所以这条检查当时**必须不红**：它数完把命中数
	// 打进 `t.Skipf` 的文案，进度靠"数字 + 跳过台账"被看见。WP4a 把 CLI 侧 19 处迁掉
	// 之后还剩服务端那 5 处；**WP4b 删掉那 5 处，命中数到 0** ——
	// 跳过消失，这条测试从"记录未落地"变成"守住已落地"：
	// **全仓库再出现一处「取第一个会话」就是红的**（台账里那条 layPending 也一并删了）。
	//
	// 失败信息要能照做：逐处 `文件:行` + 为什么必须清零 + 该往哪儿改。
	if total != 0 {
		t.Errorf("全仓库不该再出现「取第一个会话」，实得 %d 处。%s\n"+
			"    为什么必须清零（计划开头那句可验证的话）：多会话下「取第一个」= 命令跑到\n"+
			"    **另一条**会话上，而输出看起来一切正常；落不上目标时必须是报错。\n"+
			"    怎么改：CLI 侧向目标层要目标（WP4a 的 `internal/cli/debug/target.go`）；\n"+
			"    服务端那一族由 `internal/debug/api_target.go` 的 resolveTarget / mustTarget\n"+
			"    取代（请求显式给目标，或池内唯一；>=2 条报 SESSION_AMBIGUOUS 并列候选）。\n"+
			"    唯一豁免是会话池自己的**定义行**（`func (m *Manager) Current() *Session`），\n"+
			"    它不进这份普查，也不算在上面这个数里。",
			total, report.String())
	}
}

// repoScanFiles 返回仓库里**所有非测试的 `.go`**（绝对路径，按目录序走）。
//
// 三处刻意的取舍：
//
//   - **不含 `_test.go`**：测试文件里出现 `sessions[0]` 是在**构造一个多会话的输入**，
//     不是去取第一个会话。把它们算进来会让这条检查失去意义（真正该清的点会被噪声淹没）。
//   - **跳过 `web/dist`**：那是构建产物（嵌进二进制的 SPA），不是源码。
//   - **跳过 `.git`**：不是源码。
//
// 其余一律扫（`internal/`、`tools/`、`engine/` 之外的 Go、根目录的 `main.go`）——
// "全仓库"这三个字就是字面意思，少扫一个目录就等于给"取第一个会话"留了个后门。
func repoScanFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		out = append(out, p)
		return nil
	})
	if err != nil {
		t.Fatalf("扫 %s 失败：%v", root, err)
	}
	return out
}

// TestRepoScanFilesExcludesTestsAndDist 是上一条的地基：扫描的**边界**本身也要有判据。
//
// 边界写错的表现是"扫到了不该扫的"或"漏了该扫的"，两种都让上一条变成假绿：
// 前者被噪声淹没（测试里到处都在造 sessions[0]），后者留后门（漏掉的目录里的点永远数不到）。
func TestRepoScanFilesExcludesTestsAndDist(t *testing.T) {
	root := RepoRoot(t)
	files := repoScanFiles(t, root)

	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		rel = filepath.ToSlash(rel)
		if strings.HasSuffix(rel, "_test.go") {
			t.Errorf("扫描把测试文件也算进来了：%s\n"+
				"    测试里出现 sessions[0] 是在**构造多会话输入**，不是「取第一个会话」。", rel)
		}
		if strings.HasPrefix(rel, "web/dist/") {
			t.Errorf("扫描把构建产物也算进来了：%s（那是嵌进二进制的 SPA，不是源码）", rel)
		}
	}

	// 反向：「该扫到的必须扫到」—— 下面这几处的命中是今天的**已知事实**，
	// 少扫一处，基线就对不上（上面那条测试也会红）。
	must := []string{
		"internal/cli/debug/debugctl.go", // dbgCurrentID 的定义与调用点
		"internal/cli/debug/debugctx.go", // dbgCurrentID 的调用点
		"internal/debug/manager.go",      // Current() 的定义处（豁免对象）
		"internal/debug/api.go",          // B11 的五处直呼点
	}
	seen := map[string]bool{}
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		seen[filepath.ToSlash(rel)] = true
	}
	for _, m := range must {
		if !seen[m] {
			t.Errorf("%s 没被扫到 —— 那个文件里有「取第一个会话」的点（见 firstSourceCensus 的基线），\n"+
				"    漏扫它等于给这条检查开了个后门。", m)
		}
	}
}

// ---------------------------------------------------------------------------
// 小工具（single_source_test.go 里那份 scanTestFiles 只扫 `_test.go`，
// 这里要扫的是非测试的 .go，所以单开一份）
// ---------------------------------------------------------------------------

// lineAt 返回 offset 所在那一行的文本。
func lineAt(text string, offset int) string {
	start := strings.LastIndexByte(text[:offset], '\n') + 1
	end := strings.IndexByte(text[offset:], '\n')
	if end < 0 {
		return text[start:]
	}
	return text[start : offset+end]
}

// lineNoAt 返回 offset 所在的行号（1 基）。
func lineNoAt(text string, offset int) string {
	return itoa(strings.Count(text[:offset], "\n") + 1)
}

// itoa 是 strconv.Itoa 的薄封装，避免为了几个失败信息多引一个 import。
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
