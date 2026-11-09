package tzs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"tt/internal/testenv"
)

// fns_test.go —— 引擎**函数面**的覆盖关卡。
//
// 它回答的问题与 corpus_test.go 那几条不同：那几条问"这一包写回去还对不对"（回归网），
// 这一条问"**声明出来的每个写函数，到底被驱动过没有**"（覆盖网）。
// engine/README.md 自己写着「引擎的语料回归驱动的是 8–10 个函数，是回归网，不是函数面的
// 覆盖网…函数面覆盖目前是空的，要补得在 Go 侧补」。这个文件就是那个"在 Go 侧补"。
//
// ## 参数是**问引擎要的**，不是抄的
//
// 参数不再逐个函数手写：`--manifest` 把每个函数的参数声明的**类型**、**角色**
// （`component-path` / `package-path` / `direction` / `action-id` / `new-name`）、
// **enum 的合法值集**都给了出来，于是合成器（`synthArgs`）按"角色优先于类型"这条
// 已有原则填参 —— 那个原则是 CLI 的 `placeholder` 立的（见 tzs_engine.go 的注释）。
//
// 手写的只有"给这个函数挑哪个靶子"（`fnsCases` 里的 pick）—— 那件事没有通用答案。
//
// ## 为什么是 36 而不是 33
//
// `engine/gate-w3-fns.py`（那份跑不起来的 Python 关卡，缺 `gate-w3.py`）的文件头写的是
// 33 —— 那是它写下来时的数。**今天引擎的 manifest 声明 36 个 `writes:true`**。
// 本文件不抄那个数：`TestFnsRegistryMatchesManifest` 每次跑都去问引擎逐条对账。
//
// ## 一个进程跑完整个序列（这条判据必须保住）
//
// `ComponentTabIndexService.Register` 不清它的两个列表，所以**同一进程里第二次 tab 操作
// 会接着第一次编号**（实测：第二次 clear 编 8..15 而不是 1..8）。
// 一个"每次调用一个进程"的装置看不见这个 —— 所以每个包**只起一个 `--stdio` 进程**。
//
// ## 四类判决（与 gate-w3-fns.py 逐字对齐，那四类是对的）
//
//	CHANGED = ok:true 且没报 noop
//	NOOP    = ok:true 且 noop（**是声明的成功，不是失败**，SPEC 11.24 (a)）
//	REFUSED = 被**契约**拒了（退出码 2「参数/环境不对，可以自纠」或 4「表单规则说不」）
//	FAIL    = 其余一切，**包括 E_NOT_IMPLEMENTED 与 E_INTERNAL**
//	SKIP    = 这一包没有这条用例需要的前置 —— **必须说清缺什么**，裸 skip 不算证据
//
// ## 三条判据（TestFnsGate）
//
//	一  36 个函数没有一个归到 FAIL
//	二  每个登记的函数至少成功驱动过一次（CHANGED 或 NOOP）
//	三  判据二里被豁免的那几个，由**引擎的 detail.reason**背书（fnsNoPositiveCase）
//
// 判据三不是"放宽判据二"：豁免表里每一条都要证明它确实一次没成功过、确实被**试过**、
// 且每一次拒绝的理由码都恰好是声明的那一个。语料一变（真出现一个自订程序），
// 理由码就不再是它 —— 那条红。判据二的立场不变：「从来没做成过」与「做过且对」是两件事，
// 而"这份语料里没有可做成的对象"是第三件事，它必须自己拿证据。
//
// ## 开关
//
//	TTZS_FNS=1 TTZS_CORPUS=%TEMP%\ttws TTZS_EXE=<repo>\engine\out\tzs-server.exe \
//	  go test ./internal/dev/tzs -run TestFnsGate -timeout 30m -v
//
// **两条硬纪律跟着守**：只在副本上跑；整轮跑的时候不要重建引擎。

const fnsEnv = "TTZS_FNS"

type fnsVerdict string

func (v fnsVerdict) String() string { return string(v) }

const (
	verdictChanged fnsVerdict = "CHANGED"
	verdictNoop    fnsVerdict = "NOOP"
	verdictRefused fnsVerdict = "REFUSED"
	verdictFail    fnsVerdict = "FAIL"
	verdictSkip    fnsVerdict = "SKIP"
)

// fnsTarget 是一条用例挑出来的靶子。
type fnsTarget struct {
	// path 是 name-path（角色 component-path 的那几个参数用它）。
	path string
	// paths 给 path[] 型参数；空就退化成 []string{path}。
	paths []string
	// extra 按**参数名**预给的值 —— 合成器认不出语义的那些（element/property/table/
	// program/content/id/…）。给了就优先于一切推断。
	extra map[string]any
}

func (tg fnsTarget) pathList() []string {
	if len(tg.paths) > 0 {
		return tg.paths
	}
	if tg.path != "" {
		return []string{tg.path}
	}
	return nil
}

func (tg fnsTarget) get(name string) (any, bool) {
	v, ok := tg.extra[name]
	return v, ok
}

// fnsCase 是"一个写函数怎么驱动"。
type fnsCase struct {
	fn    string
	group string
	// pick 挑**候选靶子**（按优先级）；返回非空 why = 这一包跑不了这一条
	// （**必须说清缺什么**）。nil = 用空靶子。
	//
	// 为什么是列表：好几条用例要的是**某一类**元素（add_action 要 Button、
	// set_items 要 ComboBox/RadioGroup、insert_semantic 要 Table/Tree 的父节点……），
	// 而"哪个元素合格"只有引擎知道。于是给一串候选、逐个试，取第一个不被拒的 ——
	// 引擎拒了 A 就换 B，而不是把"A 不合格"算成这一条没驱动成。
	pick func(*fnsRun) ([]fnsTarget, string)
	// custom 完全接管（调用在它里面发生，所以它把判决也返回来）。
	// 给那些"要按当前状态算值"的 —— 属性四件套（值必须与当前不同）。
	// 返回 (判决, 跳过原因)：判决为空串表示跳过。
	custom func(*fnsRun, *SpecFn) (fnsVerdict, string)
}

// fnsRun 是一条用例跑的时候手里有的东西。
type fnsRun struct {
	t     *testing.T
	s     *stdioSession
	label string
	h     string
	root  *treeNode
	// dirty 表示结构类函数动过树；下一条跑之前要重读（拿旧路径去操作新结构会得到
	// E_NOT_FOUND，而那条红是装置的错，不是引擎的）。
	dirty bool
	// specOf 是引擎的函数声明（按名查），合成参数用。
	specOf func(string) *SpecFn
	// extraVals 是按名字给的全局值（目前只有 packagePath）。
	extraVals map[string]string
}

func (r *fnsRun) refresh() {
	r.root = formTree(r.t, r.s, r.label+" form_tree(重读)", r.h)
	r.dirty = false
}

// call 发一帧并归到四类里。
func (r *fnsRun) call(fn string, args map[string]any) (*Reply, fnsVerdict) {
	r.t.Helper()
	rep, err := r.s.call(fn, args)
	if err != nil {
		r.t.Errorf("%s: %s 传输层失败：%v", r.label, fn, err)
		return nil, verdictFail
	}
	if rep.OK {
		if replyNoop(rep) {
			return rep, verdictNoop
		}
		return rep, verdictChanged
	}
	if isContractRefusal(rep) {
		return rep, verdictRefused
	}
	return rep, verdictFail
}

// replyWhy 把一次失败应答压成一行可读的原因（诊断用）。
//
// **这一行是接下来修参数的唯一线索** —— 只看"被拒了"没法判断是靶子挑错了、
// 还是某个参数的值不在合法集里。所以码与消息都要。
func replyWhy(rep *Reply) string {
	if rep == nil {
		return "没有应答"
	}
	if rep.Error != nil {
		return rep.Error.Code + "：" + firstLine(rep.Error.Message)
	}
	return "ok"
}

// replyReason 取 `error.detail.reason` —— 引擎自己的拒收分类键（SPEC §11.24 (a)）。
//
// 为什么不拿 error.code：一级码只有 E_DESIGNER 那么粗，而"这张表单是标准程序"
// 与"这个元素没有 CitedSpec"是**两件不同的事**，它们的区别只在 detail.reason 里
// （Semantic.cs:85 的 Refused(reason, message, detail) 把 reason 塞进 detail）。
// 拿 error.code 去断言，等于断言"被拒了"，那什么都证明不了。
//
// 取不到就返回空串（不是零值猜测）：detail 的形状按函数而变，没有 reason 就是没有。
func replyReason(rep *Reply) string {
	if rep == nil || rep.Error == nil || len(rep.Error.Detail) == 0 {
		return ""
	}
	var d struct {
		Reason string `json:"reason"`
	}
	if json.Unmarshal(rep.Error.Detail, &d) != nil {
		return ""
	}
	return d.Reason
}

// firstLine 取第一行（引擎的消息常是多行，报告里一行就够）。
func firstLine(s string) string {
	if i := strings.IndexByte(s, 0x0a); i >= 0 {
		return s[:i]
	}
	return s
}

// replyNoop 读 result.noop —— E_NO_OP 是**成功码**，但它可能走 error 字段
// （见 client.go 的 CodeNoOp 注释），所以两个地方都要看。
func replyNoop(rep *Reply) bool {
	if rep.Error != nil && rep.Error.Code == CodeNoOp {
		return true
	}
	var m map[string]any
	if len(rep.Result) > 0 && json.Unmarshal(rep.Result, &m) == nil {
		if b, ok := m["noop"].(bool); ok {
			return b
		}
	}
	return false
}

// isContractRefusal 判断这次拒绝是不是"**契约**里声明过的拒绝"。
//
// 判据用**退出码**，不自列码表 —— 那正是仓库既有的分类（client.go 的 frameExitCode）：
//
//	退 2（ExitUsage）    参数/环境不对，调用方可以自纠 —— 引擎在正确地拒绝坏用法
//	退 4（ExitDesigner） 表单自己的规则说不 —— 同上
//	退 1（ExitFrameErr） **"声明了却没实现"（E_NOT_IMPLEMENTED）与"内部炸了"**
//	                     正是这个关卡要暴露的那两件事
//
// 自列码表的话，引擎加一个新码就得同步一次，漏掉的那个会静默变成 REFUSED。
func isContractRefusal(rep *Reply) bool {
	switch ExitCode(rep, nil) {
	case ExitUsage, ExitDesigner:
		return true
	}
	return false
}

//---------------------------------------------------------------------------
// 账
//---------------------------------------------------------------------------

type fnsTally struct {
	verdicts map[string]map[fnsVerdict]int
	// reasons 记的是 `error.detail.reason` —— **不是** error.code。
	//
	// 两件事必须分开：`standard_program` / `no_cited_spec` / `not_found` 这些是引擎
	// SPEC §11.24 (a) 的分类键，落在 detail 里；而 error.code 一级只有 E_DESIGNER 那么大
	// （见 engine/src/Designer/Fns/Semantic.cs:85 的 Refused(reason, …) 与 Rpc.cs 的映射表）。
	// 拿 E_DESIGNER 去判"是不是被这个理由拒的"什么也判不出来。
	reasons map[string]map[string]int
	notes   map[string]string // 每个函数**第一条**非成功判决的原因（诊断线索）
	order   []string
}

func newFnsTally(cases []fnsCase) *fnsTally {
	t := &fnsTally{
		verdicts: map[string]map[fnsVerdict]int{},
		reasons:  map[string]map[string]int{},
		notes:    map[string]string{},
	}
	for _, c := range cases {
		t.verdicts[c.fn] = map[fnsVerdict]int{}
		t.reasons[c.fn] = map[string]int{}
		t.order = append(t.order, c.fn)
	}
	return t
}

// add 记一条判决。rep 可以为 nil（跳过、合成不出参数这类没有应答的情形）——
// 那就没有理由码可记，理由表里不会多出一项（**不是**记一个空串，
// 否则"每一个理由码都必须是 standard_program"那条断言会被跳过项误伤）。
func (t *fnsTally) add(fn string, v fnsVerdict, rep *Reply, why string) {
	t.verdicts[fn][v]++
	if r := replyReason(rep); r != "" {
		t.reasons[fn][r]++
	}
	if v == verdictChanged || v == verdictNoop || why == "" {
		return
	}
	// 非成功判决一律留一条原因 —— 报告里要看得见"为什么没成"，
	// 否则一条 REFUSED 只会说"引擎拒了"，而拒的理由恰恰是接下来要修的线索。
	if t.notes == nil {
		t.notes = map[string]string{}
	}
	// **REFUSED/FAIL 的备注优先于 SKIP**：SKIP 在某些包上本来就是正常的（"本包没那种元素"），
	// 而拒绝/失败才是接下来要修的线索。第一版只留第一条，于是 set_tree_source 的备注
	// 被某个包的 SKIP 占住了，真正的拒绝原因一个字也看不到。
	if old, seen := t.notes[fn]; !seen || (old != "" && strings.HasPrefix(old, string(verdictSkip))) {
		t.notes[fn] = v.String() + "：" + why
	}
}

// succeeded 是这个函数**成功做到过**的次数（CHANGED 或 NOOP）。
//
// 这才是"被驱动过"的判据：REFUSED 只说明引擎拒了这次调用，没说明它做成过任何事 ——
// 而参数是我们合成的，被拒很可能是参数挑得不对，不能算作覆盖。
func (t *fnsTally) succeeded(fn string) int {
	vs := t.verdicts[fn]
	return vs[verdictChanged] + vs[verdictNoop]
}

func (t *fnsTally) report() string {
	var b strings.Builder
	for _, fn := range t.order {
		vs := t.verdicts[fn]
		fmt.Fprintf(&b, "  %-24s 成功 %3d  CHANGED %3d  NOOP %3d  REFUSED %3d  FAIL %3d  SKIP %3d",
			fn, t.succeeded(fn), vs[verdictChanged], vs[verdictNoop], vs[verdictRefused], vs[verdictFail], vs[verdictSkip])
		if s := t.notes[fn]; s != "" {
			b.WriteString("   ← " + s)
		}
		b.WriteString("\n")
	}
	return b.String()
}

//---------------------------------------------------------------------------
// 开关与前置
//---------------------------------------------------------------------------

// requireFns 是这条关卡的显式开关。
//
// 它**不**要求 TTZS_DEEP：这条关卡有自己的开关，不该被迫把整个 17 分钟的深档回归打开。
// 前置（引擎 exe / 语料 / 工作区）与别处一样，缺哪样说哪样。
func requireFns(t *testing.T) *corpusEnv {
	t.Helper()
	if strings.TrimSpace(os.Getenv(fnsEnv)) == "" {
		t.Skipf("函数面关卡未启用：设 %s=1（它会对每个语料包驱动一遍全部写函数，见 fns_test.go 顶部）", fnsEnv)
	}
	return requireCorpusEnv(t)
}

//---------------------------------------------------------------------------
// 参数合成：角色优先于类型
//---------------------------------------------------------------------------

// synthArgs 按引擎自己的参数声明填一份参数。
//
// **角色优先于类型**：`component-path` 不论声明成 path 还是 string 都是 name-path；
// `new-name` / `action-id` 各有各的来源。这条原则是 CLI 的 `placeholder` 立的
// （见 tzs_engine.go 的注释），这里沿用。
//
// 只填**必填**参数；可选参数一律不给（`dry_run` / `op` 由调用方按需加）——
// 少给一个可选参数最多是走得浅一点，乱给一个却可能把调用变成另一种意思。
func (r *fnsRun) synthArgs(spec *SpecFn, tg fnsTarget) (map[string]any, string) {
	args := map[string]any{"handle": r.h}
	for _, p := range spec.Args {
		if p.Name == "handle" {
			continue
		}
		// 靶子明确给了值的，**不论必填可选都给** —— 它比任何推断都权威。
		// （实测踩到过：`add_field.column` 是可选参数，靶子给了值却在必填判断那一步
		// 被跳过，于是引擎回"需要 column 或 columns 之一"。）
		if v, ok := tg.get(p.Name); ok {
			args[p.Name] = v
			continue
		}
		if !p.Required {
			// 其余可选参数默认不给，**但 component-path 型的例外**：实测 align / nudge /
			// fit_size / wrap / delete / move 的 path 都是可选的，不给就等于调用了一个
			// "对空选择集操作"的动词 —— 引擎会拒，而那条 REFUSED 什么也没证明。
			if p.Role != RoleComponentPath || len(tg.pathList()) == 0 {
				continue
			}
		}
		switch p.Role {
		case RoleComponentPath:
			if p.Type == TypePathList {
				args[p.Name] = tg.pathList()
			} else {
				args[p.Name] = tg.path
			}
			continue
		case RoleNewName:
			args[p.Name] = "tdev_fns_new"
			continue
		case RolePackagePath:
			args[p.Name] = r.extra("packagePath")
			continue
		case RoleActionID:
			return nil, "参数 " + p.Name + " 是 action-id，但靶子没给 id"
		}
		switch p.Type {
		case TypeEnum:
			if len(p.Values) == 0 {
				return nil, "参数 " + p.Name + " 是 enum 但引擎没给合法值集"
			}
			args[p.Name] = p.Values[0]
		case TypeBool:
			args[p.Name] = true
		case TypeInt:
			args[p.Name] = 1
		case TypePath:
			if tg.path == "" {
				return nil, "参数 " + p.Name + " 要一个 name-path，但靶子没给"
			}
			args[p.Name] = tg.path
		case TypePathList:
			if len(tg.pathList()) == 0 {
				return nil, "参数 " + p.Name + " 要 name-path 列表，但靶子没给"
			}
			args[p.Name] = tg.pathList()
		case TypeKind:
			k, why := r.pickKind()
			if why != "" {
				return nil, why
			}
			args[p.Name] = k
		case TypeKindOrLayout:
			args[p.Name] = "layout"
		default:
			// string / string[] / attr / attrs 这些**认不出语义**的，只能由靶子给 ——
			// 编一个值进去会让这次调用变成"用假参数试探引擎"，那不是覆盖。
			return nil, "参数 " + p.Name + "（" + p.Type + "）合成器认不出语义，靶子也没给"
		}
	}
	return args, ""
}

// extra 取一条靶子之外、按名字给的全局值（目前只有包路径）。
func (r *fnsRun) extra(name string) string {
	if r.extraVals == nil {
		return ""
	}
	return r.extraVals[name]
}

//---------------------------------------------------------------------------
// 挑靶子
//---------------------------------------------------------------------------

// findPath 问引擎"这个名字的元素在哪"，返回 name-path。
func (r *fnsRun) findPath(name string) (string, bool) {
	fc, ok := ask[findReply](r.t, r.s, r.label+" find_component "+name, "find_component",
		map[string]any{"handle": r.h, "query": name})
	if !ok {
		return "", false
	}
	for _, m := range fc.Matches {
		if m.Name == name && m.Path != "" {
			return m.Path, true
		}
	}
	return "", false
}

// fnsWalkTree 深度优先遍历结构树。
func fnsWalkTree(n *treeNode, fn func(*treeNode) bool) bool {
	if n == nil {
		return false
	}
	if fn(n) {
		return true
	}
	for _, c := range n.Children {
		if fnsWalkTree(c, fn) {
			return true
		}
	}
	return false
}

// pickByTag 挑第一个满足条件的元素。
func (r *fnsRun) pickByTag(pred func(*treeNode) bool, what string) (string, string) {
	if r.root == nil {
		return "", "form_tree 没拿到结构树"
	}
	var path string
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if n.Path != "" && pred(n) {
			path = n.Path
			return true
		}
		return false
	})
	if path == "" {
		return "", "结构树里没有 " + what
	}
	return path, ""
}

// isContainerTag 是能收子元素的标签（设计器的 mime 门禁只允许这些收 Button）。
func isContainerTag(tag string) bool {
	switch tag {
	case "Grid", "Group", "VBox", "HBox", "Folder":
		return true
	}
	return false
}

// isLeafTag 是可写的叶子控件。
func isLeafTag(tag string) bool {
	switch tag {
	case "Form", "Folder", "Page", "Grid", "Group", "VBox", "HBox":
		return false
	}
	return true
}

// pickWidgetContainer 挑一个**能收控件**的容器。
//
// 设计器的 mime 门禁只允许 Grid / Group 收 Button（见 corpus_test.go 的 actionOp 注释：
// "设计器的 mime 门禁只允许 Grid/Group 收 Button"）—— 所以这里不能用"任意容器"，
// 否则 add_widget 会被引擎正当地拒掉，而那条 REFUSED 什么也没证明。
func pickWidgetContainer(r *fnsRun) ([]fnsTarget, string) {
	path, why := r.pickByTag(func(n *treeNode) bool {
		return (n.Tag == "Grid" || n.Tag == "Group") && n.Path != ""
	}, "能收控件的 Grid / Group")
	return []fnsTarget{{path: path}}, why
}

// pickContainer 挑一个有子元素的容器。
func pickContainer(r *fnsRun) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var path string
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if n.Path != "" && isContainerTag(n.Tag) && len(n.Children) > 0 {
			path = n.Path
			return true
		}
		return false
	})
	if path == "" {
		return nil, "结构树里没有带子元素的容器"
	}
	return []fnsTarget{{path: path}}, ""
}

// pickBound 挑一个**绑了列的字段**的布局元素。
//
// 为什么不是"任意叶子"：实测拿 Label 去调 set_cited / set_excluded / set_code_template /
// break_layout / convert_widget / set_table_association 一律被引擎正当地拒 ——
// 那些动词说的是**数据字段**的事。corpus_test.go 的 resolveEditTarget 也是这么挑的
// （它要求 `column` 非空，"`column` 非空才是「数据绑定」这件事的权威答案"）。
func pickBound(r *fnsRun) (name, path, why string) {
	return pickSpecNode(r, "field")
}

// pickBoundTarget 是 pickBound 的靶子包装。
func pickBoundTarget(r *fnsRun) ([]fnsTarget, string) {
	_, path, why := pickBound(r)
	return []fnsTarget{{path: path}}, why
}

// pickLeaf 挑一个叶子控件 —— **但它必须在引擎的 ElementIndex 里能被寻址**。
//
// 实测踩到过：form_tree 里有 Label（`lbl_apcasite`），把它当路径传给 delete / nudge /
// fit_size / wrap / reparent 一律回 `E_PATH_NOT_FOUND：路径 不存在` —— 设计器的
// ElementIndex 不收那类元素。所以挑完之后**问一次引擎**（find_component），
// 寻址不到就换下一个。那一次问是值得的：少了它，六条用例会以"路径不存在"被拒，
// 而那条 REFUSED 指着的是靶子而不是被测代码。
func pickLeaf(r *fnsRun) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var path string
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if path != "" || n.Path == "" || !isLeafTag(n.Tag) {
			return false
		}
		if _, ok := r.findPath(n.Name); ok {
			path = n.Path
			return true
		}
		return false
	})
	if path == "" {
		return nil, "结构树里的叶子控件没有一个能被引擎寻址（find_component 全落空）"
	}
	return []fnsTarget{{path: path}}, ""
}

// pickPage 挑一个页面（标签页那一组用）。
//
// 它**不用 form_tree**：那棵树里 Page 是 Folder 的子节点，而标签页那组函数要的是
// 页面自己的 name-path —— 从树里取同一条路径就够（Page 的 Path 就是它的 name-path）。
func pickPage(r *fnsRun) ([]fnsTarget, string) {
	path, why := r.pickByTag(func(n *treeNode) bool { return n.Tag == "Page" }, "页面（标签页）")
	return []fnsTarget{{path: path}}, why
}

// pickSpecNode 挑一个指定 kind 的规格节点，并定位到布局元素上。
//
// "哪个元素是这个规格节点"这条映射**问引擎**（find_component），不自己解析 .4fd ——
// 自己拼就多一处会漂的第二实现（corpus_test.go 的 resolveEditTarget 同一个理由）。
func pickSpecNode(r *fnsRun, kind string) (name, path string, why string) {
	rep, ok := ask[specNodesReply](r.t, r.s, r.label+" list_spec_nodes "+kind, "list_spec_nodes",
		map[string]any{"handle": r.h, "kind": kind})
	if !ok {
		return "", "", "问不到 " + kind + " 节点"
	}
	if len(rep.Nodes) == 0 {
		return "", "", "本包没有 " + kind + " 节点"
	}
	for _, nd := range rep.Nodes {
		if p, ok := r.findPath(nd.Name); ok {
			return nd.Name, p, ""
		}
	}
	return "", "", "本包的 " + kind + " 节点一个都没能定位到布局元素上"
}

// pickKind 挑一个本包**真有节点**的规格 kind（给 kind 型参数用）。
func (r *fnsRun) pickKind() (string, string) {
	var firstWhy string
	for _, k := range specKinds {
		if _, _, why := pickSpecNode(r, k); why == "" {
			return k, ""
		} else if firstWhy == "" {
			firstWhy = why
		}
	}
	return "", firstWhy
}

// specKinds 是引擎认的那七个规格节点 kind（它自己在 E_BAD_PARAM 里报的原文：
// field|hfield|pfield|rfield|mlfield|tree|act），按覆盖面从大到小排。
var specKinds = []string{"field", "hfield", "pfield", "rfield", "mlfield", "tree", "act"}

//---------------------------------------------------------------------------
// 应答形状
//---------------------------------------------------------------------------

type specNodesReply struct {
	Program string `json:"program"`
	Count   int    `json:"count"`
	Nodes   []struct {
		Kind string `json:"kind"`
		Name string `json:"name"`
	} `json:"nodes"`
}

type kindMapReply struct {
	Count  int `json:"count"`
	Layout []struct {
		Name    string   `json:"name"`
		Type    string   `json:"type"`
		Values  []string `json:"values"`
		Initial string   `json:"initial"`
	} `json:"layout"`
	Field []string `json:"field"`
}

// componentReply 是 get_component 的返回（只用 layout 那一段）。
//
// **`layout` 就是"这个元素自己允许的属性表"** —— 引擎拒绝时会把它列出来
// （"元素 没有属性 \"action\"（可用的：tag, posX, posY, …）"）。所以挑属性要问它，
// 而不是拿 describe_kind layout 的**并集**去猜：那张并集是整张表单的，元素只允许其中一部分。
type componentReply struct {
	Tag   string            `json:"tag"`
	Name  string            `json:"name"`
	Path  string            `json:"path"`
	Attrs map[string]string `json:"layout"`
}

type localStringsReply struct {
	Program string `json:"program"`
	Count   int    `json:"count"`
	Strings []struct {
		Name string `json:"name"`
		Text string `json:"text"`
	} `json:"strings"`
}

type recordsReply struct {
	Program string `json:"program"`
	Count   int    `json:"count"`
	Records []struct {
		Attrs  map[string]string `json:"attrs"`
		Fields []struct {
			Name  string            `json:"name"`
			Attrs map[string]string `json:"attrs"`
		} `json:"fields"`
	} `json:"records"`
}

type tablesReply struct {
	Count  int `json:"count"`
	Tables []struct {
		Name string `json:"name"`
		Desc string `json:"desc"`
	} `json:"tables"`
}

type columnsReply struct {
	Table   string `json:"table"`
	Count   int    `json:"count"`
	Columns []struct {
		Name string `json:"name"`
	} `json:"columns"`
}

//---------------------------------------------------------------------------
// 登记表：36 个写函数各怎么挑靶子
//---------------------------------------------------------------------------

// fnsCases 返回登记表。
//
// **它必须与引擎 manifest 里 `writes:true` 的集合完全一致** —— 由
// TestFnsRegistryMatchesManifest 盯着。加了新写函数而没在这里登记，那条会红。
func fnsCases() []fnsCase {
	return []fnsCase{
		// ---- 属性（spec / layout 两侧）----
		{fn: "set_spec_attr", group: "属性", custom: customSpecAttr},
		{fn: "set_spec_attrs", group: "属性", custom: customSpecAttrs},
		{fn: "set_layout_attr", group: "属性", custom: customLayoutAttr},
		{fn: "set_layout_attrs", group: "属性", custom: customLayoutAttrs},
		{fn: "set_tree_source", group: "属性", pick: pickTreeSource},
		{fn: "rename_component", group: "属性", pick: pickLeaf},

		// ---- 结构 ----
		{fn: "add_widget", group: "结构", pick: pickWidgetContainer},
		{fn: "add_field", group: "结构", pick: pickAddField},
		{fn: "insert_at", group: "结构", pick: pickWidgetContainer},
		{fn: "delete", group: "结构", pick: pickLeaf},
		{fn: "move", group: "结构", pick: pickMove},
		{fn: "reparent", group: "结构", pick: pickReparent},
		{fn: "nudge", group: "结构", pick: pickLeaf},
		{fn: "align", group: "结构", pick: pickPaths2},
		{fn: "fit_size", group: "结构", pick: pickLeaf},
		{fn: "wrap", group: "结构", pick: pickWrapTarget},
		{fn: "break_layout", group: "结构", pick: pickLayoutBox},
		{fn: "convert_widget", group: "结构", pick: pickConvertWidget},
		{fn: "convert_container", group: "结构", pick: pickConvertContainer},

		// ---- 标签页 ----
		{fn: "add_page", group: "标签", pick: pickPage},
		{fn: "delete_page", group: "标签", pick: pickDeletePage},

		// ---- 动作 ----
		{fn: "insert_semantic", group: "动作", pick: pickInsertSemantic},
		{fn: "add_action", group: "动作", pick: pickAddAction},
		{fn: "set_action_types", group: "动作", pick: pickActionTypes},
		// 顺序有意义：delete_action 会把那个动作标成 status=d，之后 set_action_types
		// 再改它就是"对已删除的动作改型态"（引擎会正当地拒）。所以它排在上面。
		{fn: "delete_action", group: "动作", pick: pickAction},

		// ---- 数据源 / 选项 / 描述 ----
		{fn: "set_local_string", group: "数据", pick: pickLocalString},
		{fn: "set_items", group: "数据", pick: pickItems},
		{fn: "set_progrel_programs", group: "数据", pick: pickProgrel},
		{fn: "set_table_association", group: "数据", pick: pickTableAssoc},
		{fn: "set_spec_description", group: "数据", custom: customSpecDesc},
		{fn: "set_cited", group: "数据", pick: pickBoundTarget},

		// ---- Tab 顺序 ----
		{fn: "set_tab_order", group: "Tab", pick: pickTabControl},
		{fn: "tab_action", group: "Tab", pick: pickTabControl},

		// ---- 校验 / 代码模板 ----
		{fn: "set_excluded", group: "校验", pick: pickBoundTarget},
		{fn: "set_code_template", group: "校验", pick: pickBoundTarget},

		// ---- 工作流 ----
		{fn: "field_add", group: "工作流", pick: pickFieldAdd},
	}
}

// allByPred 按条件列出**所有**候选（每个都先问过引擎能不能寻址）。
//
// 为什么"先问一次"：form_tree 里有 Label 那类元素，路径传过去引擎回
// `E_PATH_NOT_FOUND：路径 不存在` —— 设计器的 ElementIndex 不收它们。少了这一次问，
// 六条用例会以"路径不存在"被拒，而那条 REFUSED 指着的是靶子而不是被测代码。
func (r *fnsRun) allByPred(pred func(*treeNode) bool, none string) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var out []fnsTarget
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if n.Path != "" && pred(n) {
			if _, ok := r.findPath(n.Name); ok {
				out = append(out, fnsTarget{path: n.Path})
			}
		}
		return false
	})
	if len(out) == 0 {
		return nil, none
	}
	return out, ""
}

// allByTag 按标签列出候选。
func (r *fnsRun) allByTag(tags ...string) ([]fnsTarget, string) {
	want := map[string]bool{}
	for _, t := range tags {
		want[t] = true
	}
	return r.allByPred(func(n *treeNode) bool { return want[n.Tag] },
		"结构树里没有 "+strings.Join(tags, " / "))
}

// allByParentTag 列出"父元素是指定标签"的候选。
func (r *fnsRun) allByParentTag(tags ...string) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	want := map[string]bool{}
	for _, t := range tags {
		want[t] = true
	}
	var out []fnsTarget
	var walk func(n *treeNode)
	walk = func(n *treeNode) {
		for _, c := range n.Children {
			if c.Path != "" && want[n.Tag] {
				if _, ok := r.findPath(c.Name); ok {
					out = append(out, fnsTarget{path: c.Path})
				}
			}
			walk(c)
		}
	}
	walk(r.root)
	if len(out) == 0 {
		return nil, "结构树里没有父元素是 " + strings.Join(tags, " / ") + " 的控件"
	}
	return out, ""
}

// pickMove 给 move 挑"挪动谁 + 挪到哪"。
//
// 引擎报过："Grid 里的元素不能改 Z 序（设计器只在 Folder/HBox/VBox/Table/Tree 里提供
// 这个操作）" —— 所以候选**每个叶子一个**，让驱动逐个试到合格的那个。
// `to` 不是路径：引擎报的合法集是 `first|prev|next|last`（设计器的"移到…"菜单）。
func pickMove(r *fnsRun) ([]fnsTarget, string) {
	leaves, why := r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
		"结构树里没有可寻址的叶子控件")
	if why != "" {
		return nil, why
	}
	for i := range leaves {
		leaves[i].extra = map[string]any{"to": "next"}
	}
	return leaves, ""
}

// pickReparent 给 reparent 挑"挪谁们 + 进哪个容器"。
//
// 引擎报过：`into` 的容器有自己的 mime 白名单，会拒绝不收的元素类型 ——
// 所以候选是**叶子 × 容器**的组合（封顶 12 条，别让一个包生成上千条）。
func pickReparent(r *fnsRun) ([]fnsTarget, string) {
	leaves, why := r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
		"结构树里没有可寻址的叶子控件")
	if why != "" {
		return nil, why
	}
	conts, why := r.allByPred(func(n *treeNode) bool { return isContainerTag(n.Tag) },
		"结构树里没有容器")
	if why != "" {
		return nil, why
	}
	var out []fnsTarget
	for _, lf := range leaves {
		for _, ct := range conts {
			if ct.path == lf.path {
				continue
			}
			tg := lf
			tg.extra = map[string]any{"into": ct.path}
			out = append(out, tg)
			if len(out) >= 12 {
				return out, ""
			}
		}
	}
	return out, ""
}

// pickAction 挑一个已有动作（delete_action / set_action_types 要 action-id）。
func pickAction(r *fnsRun) ([]fnsTarget, string) {
	rep, ok := ask[specNodesReply](r.t, r.s, r.label+" list_spec_nodes act", "list_spec_nodes",
		map[string]any{"handle": r.h, "kind": "act"})
	if !ok || len(rep.Nodes) == 0 {
		return nil, "本包没有 act 节点（没有可改的动作）"
	}
	return []fnsTarget{{extra: map[string]any{"id": rep.Nodes[0].Name}}}, ""
}

// pickTreeSource 给 set_tree_source 挑一棵 Tree + 一组(元素, 属性, 值)。
//
// Tree 是七种规格节点之一；本包没有就跳过（说清缺什么）。
func pickTreeSource(r *fnsRun) ([]fnsTarget, string) {
	name, path, why := pickSpecNode(r, "tree")
	if why != "" {
		return nil, why
	}
	_ = name
	// 引擎报过：`E_NOT_FOUND：tree 元素 不存在: root（可用: type, type2, type3, type4,
	// type5, type6, id, pid, desc, speed, stype, sid, spid）` —— 它**把合法元素名列出来了**。
	// 所以照着它给，别自己编一个 "root"。
	return []fnsTarget{{path: path, extra: map[string]any{
		"element": "type", "property": "table", "value": "tdev_fns",
	}}}, ""
}

// pickLocalString 挑一条本地字符串（列表为空就跳过）。
func pickLocalString(r *fnsRun) ([]fnsTarget, string) {
	rep, ok := ask[localStringsReply](r.t, r.s, r.label+" list_local_strings", "list_local_strings",
		map[string]any{"handle": r.h})
	if !ok || len(rep.Strings) == 0 {
		return nil, "本包没有本地字符串"
	}
	s := rep.Strings[0]
	_, path, why := pickBound(r)
	if why != "" {
		return nil, why
	}
	return []fnsTarget{{path: path, extra: map[string]any{
		"name": s.Name, "text": s.Text + "-tdev",
	}}}, ""
}

// pickTableAssoc 给 set_table_association 挑一张表 + 一列。
func pickTableAssoc(r *fnsRun) ([]fnsTarget, string) {
	leaves, why := r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
		"结构树里没有可寻址的叶子控件")
	if why != "" {
		return nil, why
	}
	cols, why := r.firstTableColumn()
	if why != "" {
		return nil, why
	}
	for i := range leaves {
		// `column` 在这里**不是列名** —— 引擎报过："只能是 insert/delete/append 之一
		// （它在这里表示 INSERT/DELETE/APPEND ROW 那个勾选框）"。
		leaves[i].extra = map[string]any{"table": cols[0], "column": "insert"}
	}
	return leaves, ""
}

// pickWrapTarget 给 wrap 挑靶子。
//
// 引擎报过："<Label> 不能被 HBox 接受（HBox 的 mime 表只收容器，不收控件）" ——
// 所以 wrap 的靶子是**容器**（把一个容器包进新盒子），不是控件。
func pickWrapTarget(r *fnsRun) ([]fnsTarget, string) {
	return r.allByPred(func(n *treeNode) bool {
		switch n.Tag {
		case "Grid", "Group", "HBox", "VBox":
			return n.Path != ""
		}
		return false
	}, "结构树里没有可被包起来的容器")
}

// pickInsertSemantic 给 insert_semantic 挑靶子。
//
// 引擎报过："reference 栏位只能加在 Table / Tree 里" —— 所以靶子必须是
// **父元素是 Table 或 Tree** 的控件。
func pickInsertSemantic(r *fnsRun) ([]fnsTarget, string) {
	// 引擎报过："reference 栏位只能加在 Table / Tree 里" —— 但要判断"哪个元素的父是
	// Table/Tree"，布局树的标签（Grid/Group/VBox/HBox…）里看不出 Table 这一层，
	// 而引擎的模型里才有。所以候选给**所有可寻址叶子**，让它自己筛。
	return r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
		"结构树里没有可寻址的叶子控件")
}

// pickDeletePage 给 delete_page 挑靶子。
//
// 引擎报过：有的页签"CanDelIncludeChildren 为 false"（Form 自己、Form 的直接子节点、
// 被标记 IsCantDel 的元素，或任何一个不可删的后代）—— 哪个能删只有引擎知道，
// 所以把**所有页面**都给出去让它筛。
func pickDeletePage(r *fnsRun) ([]fnsTarget, string) {
	return r.allByTag("Page")
}

// pickTabControl 给 set_tab_order / tab_action 挑靶子。
//
// 引擎报过：页面"没有 tabIndex 属性——只有 mod-fd.spec 里列了 tabIndex 的那十几种才有"。
// 那十几种是**可输入控件**（Edit / ComboBox / Button 之类），不是页面 —— 所以候选给
// 所有可寻址的叶子控件，让引擎自己去筛。`paths` 给全（这一组是"排 Tab 顺序"，
// 单独一条没有意义）。
func pickTabControl(r *fnsRun) ([]fnsTarget, string) {
	// 引擎报过："只有 mod-fd.spec 里列了 tabIndex 的那十几种才有" —— 那份清单是
	// **设计器自己的规格文件**（<工作区>/mta/mod-fd.spec 的 NodeInfo properties），
	// 所以这里按标签筛，不自己编一张表。
	leaves, why := r.allByPred(func(n *treeNode) bool { return tabIndexTags[n.Tag] },
		"结构树里没有参与 Tab 的控件（"+strings.Join(tabIndexTagList, "/")+"）")
	if why != "" {
		return nil, why
	}
	paths := make([]string, 0, len(leaves))
	for _, tg := range leaves {
		paths = append(paths, tg.path)
	}
	out := make([]fnsTarget, 0, len(leaves))
	for _, tg := range leaves {
		tg.paths = paths
		out = append(out, tg)
	}
	return out, ""
}

// tabIndexTagList 是**参与 Tab 顺序**的控件类型。
//
// 来源是设计器自己的规格文件 `<工作区>/mta/mod-fd.spec`：其中
// `<NodeInfo mimeType="modFD/X" properties="…;tabIndex;…">` 的那些 X。引擎的报错把它
// 指了出来（"只有 mod-fd.spec 里列了 tabIndex 的那十几种才有"），所以照它来。
var tabIndexTagList = []string{
	"Button", "ButtonEdit", "CheckBox", "ComboBox", "DateEdit", "DateTimeEdit",
	"Edit", "Field", "RadioGroup", "RadioGroupWithItem", "Slider", "SpinEdit",
	"TimeEdit", "TextEdit", "WebComponent",
}

var tabIndexTags = func() map[string]bool {
	m := map[string]bool{}
	for _, t := range tabIndexTagList {
		m[t] = true
	}
	return m
}()

// pickAddAction 给 add_action 挑靶子 + 一个类型。
//
// `type` 在声明里是可选的，但引擎运行时要求它（实测回"缺少参数 type"）；
// 靶子必须是 **Button**（引擎报过："SpecNodeType 是 FIELD，不是 ACTION（只有 Button、
// 且 style 不是 button_qrystr 才是）"）。
func pickAddAction(r *fnsRun) ([]fnsTarget, string) {
	btns, why := r.allByTag("Button")
	if why != "" {
		return nil, why
	}
	for i := range btns {
		btns[i].extra = map[string]any{"type": "none"}
	}
	return btns, ""
}

// pickLayoutBox 给 break_layout 挑一个 HBox/VBox（它只能拆这两种）。
func pickLayoutBox(r *fnsRun) ([]fnsTarget, string) {
	// 引擎报过："父容器是 Form，设计器不允许在这里拆箱" —— 所以候选要**每个**合格的
	// HBox/VBox 一个（各自的父容器不同），让驱动试到父容器不是 Form 的那个。
	//
	// 第一版只挑"第一个 HBox/VBox"，而那个恰好是 Form 的直接子节点 —— 于是 65 个包
	// 一个都没成。**列表 + 逐个试**才是这一层的正解。
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var out []fnsTarget
	var walk func(parent *treeNode)
	walk = func(parent *treeNode) {
		for _, c := range parent.Children {
			if c.Path == "" {
				continue
			}
			if (c.Tag == "HBox" || c.Tag == "VBox") && parent.Tag != "Form" {
				if _, ok := r.findPath(c.Name); ok {
					out = append(out, fnsTarget{path: c.Path})
				}
			}
			walk(c)
		}
	}
	walk(r.root)
	if len(out) == 0 {
		return nil, "没有父容器不是 Form 的 HBox / VBox"
	}
	return out, ""
}

// pickPaths2 挑**两条**可寻址的路径（align 要求至少 2 个元素）。
func pickPaths2(r *fnsRun) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var paths []string
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if len(paths) >= 2 || n.Path == "" || !isLeafTag(n.Tag) {
			return false
		}
		if _, ok := r.findPath(n.Name); ok {
			paths = append(paths, n.Path)
		}
		return len(paths) >= 2
	})
	if len(paths) < 2 {
		return nil, "可寻址的叶子少于两个（align 要求至少 2 个元素）"
	}
	return []fnsTarget{{path: paths[0], paths: paths}}, ""
}

// pickConvertWidget 给 convert_widget 挑一个能转过去的目标类型。
//
// 目标类型**从引擎给的合法集里挑**，且避开 Button —— 实测"不支持转到 Button"。
func pickConvertWidget(r *fnsRun) ([]fnsTarget, string) {
	tgs, why := pickBoundTarget(r)
	if why != "" {
		return nil, why
	}
	for i := range tgs {
		tgs[i].extra = map[string]any{"type": convertWidgetTarget}
	}
	return tgs, ""
}

// pickConvertContainer 给 convert_container 挑一个**本身就是容器**的源元素。
func pickConvertContainer(r *fnsRun) ([]fnsTarget, string) {
	path, why := r.pickByTag(func(n *treeNode) bool {
		return (n.Tag == "Grid" || n.Tag == "Group") && n.Path != ""
	}, "Grid / Group（convert_container 要求源元素本身是容器）")
	return []fnsTarget{{path: path, extra: map[string]any{"type": "Grid"}}}, why
}

// convertWidgetTarget 是 convert_widget 的目标类型。
//
// 取自引擎报的合法集（"设计器的转换菜单只有这 14 种"）里最普通的一个 ——
// **不用 Button**：实测引擎回"不支持转到 Button"，那条 REFUSED 指着的是我们的靶子。
const convertWidgetTarget = "Edit"

// pickAddField 给 add_field 挑容器 + 一张表。
func pickAddField(r *fnsRun) ([]fnsTarget, string) {
	c, why := pickWidgetContainer(r)
	if why != "" {
		return nil, why
	}
	cols, why := r.firstTableColumn()
	if why != "" {
		return nil, why
	}
	for i := range c {
		c[i].extra = map[string]any{"table": cols[0], "column": cols[1]}
	}
	return c, ""
}

// pickActionTypes 给 set_action_types 挑一个动作 id 与一组类型。
//
// types 的合法集是"该程序自己的 vocabulary"，而契约里写明 **"none" 恒合法**（表示空集）——
// 所以这个值不需要问引擎。
func pickActionTypes(r *fnsRun) ([]fnsTarget, string) {
	tgs, why := pickAction(r)
	if why != "" {
		return nil, why
	}
	for i := range tgs {
		tgs[i].extra["types"] = "none"
	}
	return tgs, ""
}

// pickItems 给 set_items 挑一个绑字段 + 一组"name|text|description"。
func pickItems(r *fnsRun) ([]fnsTarget, string) {
	// 引擎报过："元素 X 是 Edit，不是 ComboBox/RadioGroup：XmlElement.ReplaceItems 对这种
	// 类型直接 return（不抛异常、不写任何东西），所以这里拒绝而不是假装成功"。
	//
	// 所以**先按该有的类型给**，找不到再退回"所有可寻址叶子"，让引擎自己筛 ——
	// "哪个元素合格"只有它知道。
	tgs, why := r.allByTag("ComboBox", "RadioGroup")
	if why != "" {
		tgs, why = r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
			"结构树里没有可寻址的叶子控件")
		if why != "" {
			return nil, why
		}
	} else {
		// 再**剔掉列派生的那些**，这一步是这个函数唯一不显然的地方。
		//
		// 列派生元素（fieldType 是 COLUMN_LIKE / TABLE_COLUMN）的 `items` 是**派生值**：
		// 设计器换列时用新列的 col_attr 覆盖它（SpecFieldNode.cs:389-397，反编译源码直证），
		// 加载时也会重算，所以写进去的值下次加载就没了。**那是忠实复现设计器，不是缺陷** ——
		// 设计器自己的 items 编辑器同样写得住，因为属性可见性按 **ComboBox 控件**分、
		// 不问是否列绑定（SpecPropertyEditor.xaml.cs:272）。引擎也知道这件事，
		// 返回里带了 `derivedFromColumn:true` 与说明（Semantic.cs 的 SetItems）。
		//
		// 但本关卡的判据四是「写完之后产出还是不是设计器自己的不动点」，而 RoundTrip 的
		// `stale` 量的正是"模型记的属性值与文件里存的对不上" —— 它会把这件**不是缺陷**的事
		// 报成红。实测：aapt110 上 13 个候选全是列派生的，写 4 个 → stale 从 0 变 4。
		//
		// 所以这里只挑非列派生的：写得住，判据才有意义。一个包**只有**列派生的候选时
		// 不换别的靶子、不静默，把理由说出来（它会进 tally 的 SKIP 那一栏，报告里看得见）。
		kept := make([]fnsTarget, 0, len(tgs))
		colBacked := 0
		for _, tg := range tgs {
			gc, ok := ask[componentReply](r.t, r.s, r.label+" get_component "+filepath.Base(tg.path),
				"get_component", map[string]any{"handle": r.h, "query": filepath.Base(tg.path)})
			if !ok {
				continue
			}
			switch gc.Attrs["fieldType"] {
			case "COLUMN_LIKE", "TABLE_COLUMN":
				colBacked++
			default:
				kept = append(kept, tg)
			}
		}
		if len(kept) == 0 {
			return nil, fmt.Sprintf("本包的 %d 个 ComboBox/RadioGroup 全是列派生的：items 对它们是"+
				"派生值，写进去下次加载会被列定义覆盖（SpecFieldNode.cs:389-397，设计器自己也这样）；"+
				"拿它当靶子只会让判据四报一个不是缺陷的红", colBacked)
		}
		tgs = kept
	}
	for i := range tgs {
		tgs[i].extra = map[string]any{"items": []string{"tdev_a|甲|", "tdev_b|乙|"}}
	}
	return tgs, ""
}

// pickProgrel 给 set_progrel_programs 挑一个靶子 + 一个程序名。
//
// 程序名用**这张表单自己的程序名**（form_tree 的根名）：prog_rel 里有没有这条关系不重要
// —— 这一条要驱动的是那个动词，不是要造出一条真关系。
func pickProgrel(r *fnsRun) ([]fnsTarget, string) {
	// 引擎报过："元素 X 没有 pfield（串查）节点；串查只存在于有 SpecProgRel 的字段上"
	// —— 哪个字段有只有它知道，所以候选给**所有可寻址叶子**，让它筛。
	tgs, why := r.allByPred(func(n *treeNode) bool { return isLeafTag(n.Tag) },
		"结构树里没有可寻址的叶子控件")
	if why != "" {
		return nil, why
	}
	name := ""
	if r.root != nil {
		name = r.root.Name
	}
	if name == "" {
		return nil, "结构树根没有程序名"
	}
	for i := range tgs {
		tgs[i].extra = map[string]any{"program": name}
	}
	return tgs, ""
}

// pickTabPaths 给 set_tab_order / tab_action 挑**页面**路径（Tab 顺序说的是标签页）。
func pickTabPaths(r *fnsRun) ([]fnsTarget, string) {
	if r.root == nil {
		return nil, "form_tree 没拿到结构树"
	}
	var pages []string
	fnsWalkTree(r.root, func(n *treeNode) bool {
		if n.Path != "" && n.Tag == "Page" {
			pages = append(pages, n.Path)
		}
		return false
	})
	if len(pages) == 0 {
		return nil, "结构树里没有页面（标签页那组函数要的靶子）"
	}
	if len(pages) == 1 {
		// 只有一个页面：paths 给一条也算"排过序"，只是没什么可排 —— 仍然驱动了那个动词。
		return []fnsTarget{{path: pages[0], paths: pages}}, ""
	}
	return []fnsTarget{{path: pages[0], paths: pages}}, ""
}

// customSpecDesc 驱动 set_spec_description。
//
// **引擎自己给了修法**：被引用的节点（cite_std != "N"）改描述会被直接 return
// （"在这里报出来而不是假成功"），要先用 `set_cited cited:false` 取消引用。
// 所以这一条不是"挑个更好的靶子"，而是**按依赖顺序**先解引用再改 ——
// 它顺带证明了那两个动词在真实链路上能配合。
func customSpecDesc(r *fnsRun, spec *SpecFn) (fnsVerdict, string) {
	name, path, why := pickSpecNode(r, "field")
	if why != "" {
		return "", why
	}
	_ = name
	// 先解引用。失败不算错 —— 本来就未被引用时它会 noop 或被正当地拒。
	if _, v := r.call("set_cited", map[string]any{
		"handle": r.h, "path": path, "cited": false,
	}); v == verdictFail {
		return v, "set_cited(cited:false) 报 FAIL"
	}
	rep, v := r.call("set_spec_description", map[string]any{
		"handle": r.h, "path": path, "kind": "field", "content": "tdev_fns 注入的说明",
	})
	if v == verdictFail {
		return v, "set_spec_description 报 FAIL：" + replyWhy(rep)
	}
	return v, fmt.Sprintf("path=%s %s", path, replyWhy(rep))
}

func pickSpecDesc(r *fnsRun) ([]fnsTarget, string) {
	name, path, why := pickSpecNode(r, "field")
	if why != "" {
		return nil, why
	}
	_ = name
	return []fnsTarget{{path: path, extra: map[string]any{
		"content": "tdev_fns 注入的说明", "kind": "field",
	}}}, ""
}

// pickFieldAdd 给 field_add（工作流那一个）挑"哪张表的哪几列"。
func pickFieldAdd(r *fnsRun) ([]fnsTarget, string) {
	cols, why := r.firstTableColumn()
	if why != "" {
		return nil, why
	}
	return []fnsTarget{{extra: map[string]any{
		"table":   cols[0],
		"columns": []string{cols[1]},
	}}}, ""
}

// firstTableColumn 挑一张有列的表，返回 (表名, 第一列名)。
//
// 走的是引擎的 list_tables → list_columns（那 3886 张表的元数据在 mta/ 里），
// 不自己去读 base_data 的文件 —— 同一件事不要两条路。
func (r *fnsRun) firstTableColumn() ([2]string, string) {
	tr, ok := ask[tablesReply](r.t, r.s, r.label+" list_tables", "list_tables",
		map[string]any{"limit": 20})
	if !ok || len(tr.Tables) == 0 {
		return [2]string{}, "list_tables 没给出表"
	}
	for _, t := range tr.Tables {
		cr, ok := ask[columnsReply](r.t, r.s, r.label+" list_columns "+t.Name, "list_columns",
			map[string]any{"table": t.Name})
		if !ok || len(cr.Columns) == 0 {
			continue
		}
		return [2]string{t.Name, cr.Columns[0].Name}, ""
	}
	return [2]string{}, "前 20 张表里没有一张能列出列"
}

//---------------------------------------------------------------------------
// 四个"属性"类的专用实现
//
// 它们**必须按当前值算新值**（同值改动会被引擎当空操作丢掉，那这一条就静默变成什么都
// 没证明），所以合成器覆盖不了。
//---------------------------------------------------------------------------

// customSpecAttr 驱动 set_spec_attr。
func customSpecAttr(r *fnsRun, spec *SpecFn) (fnsVerdict, string) {
	var firstWhy string
	for _, kind := range specKinds {
		attrs := specAttrsOf(r, kind)
		if len(attrs) == 0 {
			if firstWhy == "" {
				firstWhy = "describe_kind 没给出 " + kind + " 的属性名"
			}
			continue
		}
		_, path, why := pickSpecNode(r, kind)
		if why != "" {
			if firstWhy == "" {
				firstWhy = why
			}
			continue
		}
		for _, a := range attrs {
			old, ok := dryRunSpecAttr(r, path, kind, a)
			if !ok {
				continue
			}
			nv, pok := flipValue("", nil, old)
			if !pok {
				continue
			}
			rep, v := r.call("set_spec_attr", map[string]any{
				"handle": r.h, "path": path, "kind": kind, "attr": a, "value": nv,
			})
			if v == verdictFail {
				firstWhy = "set_spec_attr 报 FAIL：" + replyWhy(rep)
				continue
			}
			if v == verdictRefused {
				firstWhy = replyWhy(rep)
				continue
			}
			return v, fmt.Sprintf("kind=%s attr=%s %q→%q", kind, a, old, nv)
		}
		if firstWhy == "" {
			firstWhy = "kind=" + kind + " 的属性一个都读不出旧值"
		}
	}
	if firstWhy == "" {
		firstWhy = "找不到任何一个能改出不同值的规格属性"
	}
	return "", firstWhy
}

// customSpecAttrs 驱动 set_spec_attrs（一次改多个）。
func customSpecAttrs(r *fnsRun, spec *SpecFn) (fnsVerdict, string) {
	var firstWhy string
	for _, kind := range specKinds {
		attrs := specAttrsOf(r, kind)
		if len(attrs) < 2 {
			continue
		}
		_, path, why := pickSpecNode(r, kind)
		if why != "" {
			if firstWhy == "" {
				firstWhy = why
			}
			continue
		}
		patch := map[string]string{}
		for _, a := range attrs {
			if len(patch) >= 2 {
				break
			}
			old, ok := dryRunSpecAttr(r, path, kind, a)
			if !ok {
				continue
			}
			if nv, pok := flipValue("", nil, old); pok {
				patch[a] = nv
			}
		}
		if len(patch) == 0 {
			continue
		}
		_, v := r.call("set_spec_attrs", map[string]any{
			"handle": r.h, "path": path, "kind": kind, "attrs": patch,
		})
		if v == verdictFail {
			continue
		}
		return v, fmt.Sprintf("kind=%s attrs=%v", kind, patch)
	}
	if firstWhy == "" {
		firstWhy = "找不到能一次改两个属性的规格节点"
	}
	return "", firstWhy
}

// customLayoutAttr 驱动 set_layout_attr。
func customLayoutAttr(r *fnsRun, spec *SpecFn) (fnsVerdict, string) {
	tgs, why := pickLeaf(r)
	if why != "" {
		return "", why
	}
	var lastV fnsVerdict
	var lastWhy string
	for _, tg := range tgs {
		attrs := layoutAttrsOf(r, tg.path)
		for name, cur := range attrs {
			value, ok := flipValue("", nil, cur)
			if !ok {
				continue
			}
			rep, v := r.call("set_layout_attr", map[string]any{
				"handle": r.h, "path": tg.path, "attr": name, "value": value,
			})
			if v == verdictFail {
				return v, "set_layout_attr 报 FAIL"
			}
			if v != verdictRefused {
				return v, fmt.Sprintf("path=%s attr=%s %q→%q", tg.path, name, cur, value)
			}
			lastV, lastWhy = v, replyWhy(rep)
		}
	}
	if lastWhy == "" {
		lastWhy = "没有元素给出可改的布局属性"
	}
	return lastV, lastWhy
}

// layoutAttrsOf 问引擎要**这个元素自己的**布局属性表（get_component 的 layout 段）。
func layoutAttrsOf(r *fnsRun, path string) map[string]string {
	rep, ok := ask[componentReply](r.t, r.s, r.label+" get_component "+path, "get_component",
		map[string]any{"handle": r.h, "path": path})
	if !ok {
		return nil
	}
	return rep.Attrs
}

// customLayoutAttrs 驱动 set_layout_attrs（一次改多个）。
func customLayoutAttrs(r *fnsRun, spec *SpecFn) (fnsVerdict, string) {
	tgs, why := pickLeaf(r)
	if why != "" {
		return "", why
	}
	var lastV fnsVerdict
	var lastWhy string
	for _, tg := range tgs {
		// 每个元素**自己的**属性表（并集挑出来的会被 E_ATTR_NOT_WHITELIST 拒）。
		attrs := layoutAttrsOf(r, tg.path)
		patch := map[string]string{}
		for name, cur := range attrs {
			if len(patch) >= 2 {
				break
			}
			if v, pok := flipValue("", nil, cur); pok {
				patch[name] = v
			}
		}
		if len(patch) == 0 {
			continue
		}
		rep, v := r.call("set_layout_attrs", map[string]any{
			"handle": r.h, "path": tg.path, "attrs": patch,
		})
		if v == verdictFail {
			return v, "set_layout_attrs 报 FAIL"
		}
		if v != verdictRefused {
			return v, fmt.Sprintf("path=%s attrs=%v", tg.path, patch)
		}
		lastV, lastWhy = v, replyWhy(rep)
	}
	return lastV, lastWhy
}

// specAttrsOf 取某个 kind 的规格属性名（describe_kind <kind> → field 那一列）。
func specAttrsOf(r *fnsRun, kind string) []string {
	rep, ok := ask[kindMapReply](r.t, r.s, r.label+" describe_kind "+kind, "describe_kind",
		map[string]any{"handle": r.h, "kind": kind})
	if !ok {
		return nil
	}
	return rep.Field
}

// dryRunSpecAttr 用 `dry_run` 读一个规格属性的**当前值**。
//
// 为什么不自己解析 .tsd：那样就多一处会漂的第二实现，而且只对 field 那一族有效。
// dry_run 是引擎给的"只算不做"通道 —— 真跑一遍这个动词，答案原样放进结果，模型回滚。
// 它顺带把 dry_run 这条路径也驱动了一遍（那是**声明过的**一个参数，不是附属品）。
func dryRunSpecAttr(r *fnsRun, path, kind, attr string) (old string, ok bool) {
	rep, err := r.s.call("set_spec_attr", map[string]any{
		"handle": r.h, "path": path, "kind": kind, "attr": attr, "value": "", "dry_run": true,
	})
	if err != nil || rep == nil || !rep.OK {
		return "", false
	}
	var d deltaRef
	if json.Unmarshal(rep.Result, &d) != nil {
		return "", false
	}
	return d.Old, true
}

// flipValue 造一个与给定值**不同**的合法值。给不出就 ok=false（调用方换一个）。
//
// 类型是软提示：describe_kind 的 layout 那侧报得出 type/values，spec 那侧只报属性名，
// 所以默认分支必须能对付任意字符串。
func flipValue(typ string, values []string, cur string) (string, bool) {
	switch strings.ToUpper(typ) {
	case "BOOLEAN":
		if strings.EqualFold(cur, "true") {
			return "false", true
		}
		return "true", true
	case "ENUM":
		for _, v := range values {
			if v != cur {
				return v, true
			}
		}
		return "", false
	case "INTEGER":
		n, err := strconv.Atoi(strings.TrimSpace(cur))
		if err != nil {
			return "1", true
		}
		return strconv.Itoa(n + 1), true
	}
	switch strings.ToLower(strings.TrimSpace(cur)) {
	case "y":
		return "N", true
	case "n":
		return "Y", true
	// 布尔值也是**有合法集**的（引擎回 `E_ATTR_VALUE_ILLEGAL：属性 "hidden" 的值
	// "false-tdev" 不是它的合法表达（ENUM）合法值: false, true`）——
	// 所以默认分支不能无脑加后缀。
	case "true":
		return "false", true
	case "false":
		return "true", true
	}
	if n, err := strconv.Atoi(strings.TrimSpace(cur)); err == nil {
		return strconv.Itoa(n + 1), true
	}
	if cur == "" {
		return "tdev-fns", true
	}
	if strings.HasSuffix(cur, "-tdev") {
		return cur + "2", true
	}
	return cur + "-tdev", true
}

// pickLayoutAttr 挑一个布局属性并给出一个与它当前值不同的值。
func pickLayoutAttr(r *fnsRun) (attr, value, why string) {
	rep, ok := ask[kindMapReply](r.t, r.s, r.label+" describe_kind layout", "describe_kind",
		map[string]any{"handle": r.h, "kind": "layout"})
	if !ok || len(rep.Layout) == 0 {
		return "", "", "describe_kind layout 没给出可改的属性"
	}
	for _, a := range rep.Layout {
		if v, pok := flipValue(a.Type, a.Values, a.Initial); pok {
			return a.Name, v, ""
		}
	}
	return "", "", "每个布局属性都给不出一个不同的值"
}

//---------------------------------------------------------------------------
// 驱动
//---------------------------------------------------------------------------

// fnsPass 对每个语料包跑一遍登记表。
func fnsPass(t *testing.T, env *corpusEnv, m *Manifest) *fnsTally {
	t.Helper()
	cases := fnsCases()
	tally := newFnsTally(cases)
	pkgs := 0

	for i, src := range env.pkgs {
		ws := workspaceOf(src)
		if ws == "" {
			continue // 推不出工作区：与 corpusPass 同一个理由，不替调用方选
		}
		pkgs++
		fnsOnePackage(t, env, m, i, src, ws, cases, tally)
	}
	if pkgs == 0 {
		t.Fatalf("一个包都没跑起来 —— 前置条件写错了，不是语料的问题")
	}
	t.Logf("函数面：%d 个包跑过\n%s", pkgs, tally.report())
	return tally
}

// fnsOnePackage 是一个包的全程序列。**全程只起一个进程**（见文件头）。
func fnsOnePackage(t *testing.T, env *corpusEnv, m *Manifest, idx int, src, ws string, cases []fnsCase, tally *fnsTally) {
	t.Helper()
	label := "fns/" + filepath.Base(src)
	// 这一行是**心跳**：整轮 65 个包要二十多分钟，而报告只在最后打一次 ——
	// 没有它，中途看日志只能看到一片安静（第一版就是这样）。
	t.Logf("%s: 开始（包 %d/%d）", label, idx+1, len(env.pkgs))

	s := startStdio(t, env.exe, ws, perFileBudget)
	defer s.close()

	opened, ok := ask[openReply](t, s, label+" open", "open", map[string]any{"path": src})
	if !ok || opened.Handle == "" {
		t.Errorf("%s: open 失败", label)
		return
	}
	h := opened.Handle

	// validate 的第一次调用**就是** baseline（见 validate 的注释），所以它必须在改动之前。
	if validate(t, s, label+" validate(baseline)", h) == nil {
		t.Errorf("%s: baseline validate 失败", label)
		return
	}

	pre := scratchPath(src, idx, "fns", "pre")
	defer os.Remove(pre)
	if !saveTo(t, s, label+" save(pre)", h, pre) {
		return
	}

	// 负向对照：每分组一条 bogus args，必须被**契约**拒。
	fnsNegativeControls(t, s, label, h, m, cases)

	mid := scratchPath(src, idx, "fns", "mid")
	defer os.Remove(mid)
	if !saveTo(t, s, label+" save(mid)", h, mid) {
		return
	}
	// 负向对照不该改动任何东西 —— 这是"REFUSED 是真的被拒"唯一的证据：
	// 只看 error.code 的话，一个"拒了但已经改坏了"的实现也能过。
	if same, err := filesEqual(pre, mid); err != nil {
		t.Errorf("%s: 比对 pre/mid 失败：%v", label, err)
	} else if !same {
		t.Errorf("%s: 负向对照跑完包已经变了（mid ≠ pre）—— 被拒的调用不该落盘", label)
	}

	run := &fnsRun{t: t, s: s, label: label, h: h, specOf: m.ByName}
	run.extraVals = map[string]string{"packagePath": src}
	run.refresh()
	for _, c := range cases {
		if run.dirty {
			run.refresh()
		}
		spec := m.ByName(c.fn)
		if spec == nil {
			tally.add(c.fn, verdictSkip, nil, "引擎的 manifest 里没有这个函数")
			continue
		}
		if c.custom != nil {
			v, why := c.custom(run, spec)
			tally.add(c.fn, v, nil, why)
			if v == verdictFail {
				t.Errorf("%s: %s 归到 FAIL：%s", label, c.fn, why)
			}
			continue
		}
		targets := []fnsTarget{{}}
		if c.pick != nil {
			var why string
			targets, why = c.pick(run)
			if why != "" {
				tally.add(c.fn, verdictSkip, nil, why)
				continue
			}
		}
		// 逐个候选试，取第一个**不是被拒**的。被拒说明靶子不合格（元素类型不对），
		// 换下一个；全被拒才记一条 REFUSED（并留下最后一个原因，好知道该补什么候选）。
		var lastRep *Reply
		var lastArgs map[string]any
		done := false
		for _, tg := range targets {
			args, why := run.synthArgs(spec, tg)
			if why != "" {
				lastArgs, lastRep = args, nil
				continue
			}
			rep, v := run.call(c.fn, args)
			lastRep, lastArgs = rep, args
			if v == verdictRefused {
				continue
			}
			tally.add(c.fn, v, rep, fmt.Sprintf("%s（参数：%v）", replyWhy(rep), args))
			if v == verdictFail {
				t.Errorf("%s: %s 归到 FAIL：%s（参数：%v）", label, c.fn, replyWhy(rep), args)
			}
			// **任何一次成功改动都可能弄脏结构树**，于是把 dirty 置上、下一条跑之前重读。
			// 这一条踩过：`delete` 先跑（成功删了一个元素），之后 nudge / align / fit_size /
			// wrap / move / reparent 拿的还是**候选表里算好的旧路径** —— 六条一起报
			// `E_PATH_NOT_FOUND`，而那条红指着的是装置而不是被测代码。
			run.dirty = true
			done = true
			break
		}
		if !done {
			tally.add(c.fn, verdictRefused, lastRep, fmt.Sprintf("%s（参数：%v）", replyWhy(lastRep), lastArgs))
		}
	}
	if run.dirty {
		run.refresh()
	}

	// 改完之后：新增 error 必须为 0（与 corpus_test.go 同一个判据）。
	if v := validate(t, s, label+" validate(after)", h); v == nil {
		t.Errorf("%s: after validate 失败", label)
	} else if len(v.NewErrors) != 0 {
		t.Errorf("%s: 驱动写函数之后冒出 %d 个新 ERROR：%s", label, len(v.NewErrors), dumpFindings(v.NewErrors))
	}

	post := scratchPath(src, idx, "fns", "post")
	defer os.Remove(post)
	if !saveTo(t, s, label+" save(post)", h, post) {
		return
	}
	// 这一条是"没有一条写是静默空操作"的总闸门（gate-w3-fns.py 用同一招）：
	// 登记表里的函数跑完，产出与 pristine 必须不同。
	if !mustDiffer(t, label+" 写函数跑完之后产出必须与 pristine 不同", pre, post) {
		return
	}
	base := runRoundTrip(t, env, ws, src, label+" pristine")
	if got := runRoundTrip(t, env, ws, post, label+" post"); got != nil && base != nil {
		requireFixedPoint(t, label, got, base)
	}
}

// filesEqual 比两份文件逐字节相同。
func filesEqual(a, b string) (bool, error) {
	x, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	y, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return string(x) == string(y), nil
}

// fnsNegativeControls 每个分组至少一条 bogus args，必须被契约拒。
//
// 为什么按分组而不是按函数：每条各来一条负向对照会让整轮翻倍，而这一条要证明的是
// "**这一组**的参数校验还在"。按组一条是够的，也是刻意的摩擦上限。
func fnsNegativeControls(t *testing.T, s *stdioSession, label, h string, m *Manifest, cases []fnsCase) {
	t.Helper()
	seen := map[string]bool{}
	for _, c := range cases {
		if seen[c.group] {
			continue
		}
		spec := m.ByName(c.fn)
		if spec == nil {
			continue
		}
		seen[c.group] = true
		// 一个必然非法的大杂烩：不存在的路径 + 不存在的属性名 + 不该有的值。
		// 只填**引擎声明过的**参数名 —— 填一个它不认的键会先被本地参数校验拦下，
		// 那就测不到引擎那一侧了。
		args := map[string]any{}
		for _, p := range spec.Args {
			switch p.Name {
			case "handle":
				args[p.Name] = h
			case "path":
				args[p.Name] = "managedform/__tdev_no_such_element__"
			case "paths":
				args[p.Name] = []string{"managedform/__tdev_no_such_element__"}
			default:
				if !p.Required {
					continue
				}
				switch p.Type {
				case TypeEnum:
					if len(p.Values) > 0 {
						args[p.Name] = p.Values[0]
					}
				case TypeBool:
					args[p.Name] = true
				case TypeInt:
					args[p.Name] = 1
				default:
					args[p.Name] = "__tdev_no_such_thing__"
				}
			}
		}
		rep, err := s.call(c.fn, args)
		if err != nil {
			t.Errorf("%s: 负向对照 %s 传输层失败：%v", label, c.fn, err)
			continue
		}
		if rep.OK {
			t.Errorf("%s: 负向对照 %s 竟然成功了（传的是不存在的路径）—— 参数校验有洞：%v",
				label, c.fn, args)
			continue
		}
		if !isContractRefusal(rep) {
			t.Errorf("%s: 负向对照 %s 被拒了，但退出码是 %d（该是 2 或 4）",
				label, c.fn, ExitCode(rep, nil))
		}
	}
}

//---------------------------------------------------------------------------
// 入口
//---------------------------------------------------------------------------

func TestFnsGate(t *testing.T) {
	env := requireFns(t)
	m := fetchManifestForFns(t, env.exe)
	t.Logf("函数面关卡：%d 个包，每个包一个 --stdio 进程（预算 %s/包）", len(env.pkgs), perFileBudget)

	started := time.Now()
	tally := fnsPass(t, env, m)
	t.Logf("整轮耗时 %s", time.Since(started).Round(time.Second))

	total := map[fnsVerdict]int{}
	for _, fn := range tally.order {
		for v, n := range tally.verdicts[fn] {
			total[v] += n
		}
	}
	if total[verdictFail] > 0 {
		t.Errorf("有 %d 条归到 FAIL（见上面每个函数的表）—— FAIL 含 E_NOT_IMPLEMENTED 与意外异常，"+
			"正是这个关卡要暴露的两件事", total[verdictFail])
	}

	// 判据二：**每个登记的函数都至少成功做到过一次**（CHANGED 或 NOOP）。
	// 只有 REFUSED/SKIP 说明它从来没做成过任何事 —— 而"从来没做成过"与"做过且对"
	// 是两件完全不同的事。
	//
	// 先从这条里摘出"语料里根本没有可做成的对象"那几个 —— 但不是无条件摘：
	// 每一个都要过 checkNoPositiveCase，用**引擎给的理由码**证明它确实没有。
	exempt := map[string]bool{}
	for _, e := range fnsNoPositiveCase {
		exempt[e.fn] = true
		checkNoPositiveCase(t, tally, e)
	}

	var neverDone []string
	for _, fn := range tally.order {
		if exempt[fn] || tally.succeeded(fn) > 0 {
			continue
		}
		neverDone = append(neverDone, fmt.Sprintf("%s（%s）", fn, tally.notes[fn]))
	}
	if len(neverDone) > 0 {
		sort.Strings(neverDone)
		t.Errorf("这些写函数一次都没成功做到过：\n  %s\n\n"+
			"要么补上它的靶子（见 fnsCases 的 pick），要么把跳过原因写清楚 ——\n"+
			"「从来没做成过」与「做过且对」是两件完全不同的事。", strings.Join(neverDone, "\n  "))
	}
}

// fnsNoCase 是一条"这份语料里没有正面用例"的声明。
//
// **它是可证伪的，不是一句注释**：reason 是引擎自己在 detail.reason 里给的分类键，
// checkNoPositiveCase 要求这个函数的每一次 REFUSED 都恰好带这个码。语料一变
// （真出现一个自订程序、或某个元素真的有了 CitedSpec），码就不再是它，
// 或者干脆变成成功 —— 两种都会让这条红。
type fnsNoCase struct {
	fn     string
	reason string // 引擎每一次拒它用的理由码（detail.reason，不是 error.code）
	why    string // 为什么这份语料里没有正面用例
}

// fnsNoPositiveCase 是那张表。今天只有一条，写在这里是为了**下一条也有地方放**，
// 而且放进来比"把判据二放宽一点"难 —— 这是故意的摩擦。
var fnsNoPositiveCase = []fnsNoCase{
	{
		fn:     "set_cited",
		reason: "standard_program",
		why: "它只对 prog != std_prog 的自订程序有意义：AbstractSpecNode.CitedSpec 的第一句就是" +
			"「if (IsStandardProgram) return null」，所以标准程序连可引用的来源都没有。" +
			"引擎源码里已经写死了这件事（engine/src/Designer/Fns/Semantic.cs 的 SetCited 注释：" +
			"「all 91 corpus files have prog == std_prog and none carries cite_std=\"Y\", " +
			"so this function has no positive case in the corpus -- only the negative one. " +
			"That is reported rather than worked around.」）。" +
			"本关卡跑的那份语料同样如此 —— 而且判据不是我去读 .tsd 说的，是**引擎对每一次调用都回这个理由码**。" +
			"（顺带记一条踩过的坑：`<spec prog= std_prog=>` 在 .tsd 条目里，不在 .tap 里 —— " +
			"本仓库的包里根本没有 .tap 那个后缀，早先按 .tap 去找，于是「正则没匹配上」被误当成了线索。）",
	},
}

// checkNoPositiveCase 把一条豁免变成断言。
//
// 三件都要成立，缺一条就是"拿豁免盖住了没做成的函数"：
//  1. 它确实一次都没成功过（成功了就说明这条豁免是假的，该把它从表里删掉）
//  2. 它确实被**试过**（REFUSED 次数 > 0）—— 一条"试都没试"的豁免是裸 skip
//  3. 记到的每一个理由码都恰好是声明的那一个
func checkNoPositiveCase(t *testing.T, tally *fnsTally, e fnsNoCase) {
	t.Helper()
	if n := tally.succeeded(e.fn); n > 0 {
		t.Errorf("%s 在豁免表里，却成功做到过 %d 次 —— 这条豁免已经过期了，把它从 fnsNoPositiveCase 里删掉",
			e.fn, n)
	}
	if n := tally.verdicts[e.fn][verdictFail]; n > 0 {
		t.Errorf("%s 在豁免表里，却有 %d 条归到 FAIL —— FAIL 不是「这份语料没有对象」，是被暴露的缺陷",
			e.fn, n)
	}
	if n := tally.verdicts[e.fn][verdictRefused]; n == 0 {
		t.Errorf("%s 在豁免表里，却一次都没被拒过（SKIP %d 次）—— 这条豁免没有证据，"+
			"它只是把「从来没做成过」重新藏了起来", e.fn, tally.verdicts[e.fn][verdictSkip])
		return
	}
	for code, n := range tally.reasons[e.fn] {
		if code != e.reason {
			t.Errorf("%s 被拒的理由码出现了 %s（×%d），但豁免表声明的是 %s ——\n"+
				"语料变了：这一次拒绝指的是别的事情，%s 的那条豁免不再成立。\n%s",
				e.fn, code, n, e.reason, e.fn, e.why)
		}
	}
	t.Logf("%s：语料里没有正面用例，已由引擎的理由码背书（%s ×%d，成功 0 次）",
		e.fn, e.reason, tally.verdicts[e.fn][verdictRefused])
}

// fetchManifestForFns 拉一次函数表。
func fetchManifestForFns(t *testing.T, exe string) *Manifest {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m, err := FetchManifest(ctx, exe)
	if err != nil {
		t.Fatalf("拉函数表失败：%v", err)
	}
	return m
}

// TestFnsRegistryMatchesManifest 登记表必须与引擎自己声明的写函数集合**完全一致**。
//
// **这是"问引擎，别抄"那条规矩的落地**：33 还是 36 这种数字不该靠人维护 ——
// 每次跑都去问引擎，多一个少一个立刻红。
func TestFnsRegistryMatchesManifest(t *testing.T) {
	exe := testenv.EngineExe()
	if exe == "" {
		exe = filepath.Join("..", "..", "..", "engine", "out", "tzs-server.exe")
	}
	if abs, err := filepath.Abs(exe); err == nil {
		exe = abs
	}
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("找不到引擎 exe（TTZS_EXE=%s）：%v；先在 engine/ 里跑 build.sh", exe, err)
	}
	m := fetchManifestForFns(t, exe)

	declared := map[string]bool{}
	for _, f := range m.Fns {
		if f.Writes {
			declared[f.Name] = true
		}
	}
	registered := map[string]bool{}
	groups := map[string]string{}
	for _, c := range fnsCases() {
		registered[c.fn] = true
		if g, ok := groups[c.fn]; ok && g != c.group {
			t.Errorf("%s 在登记表里有两个分组：%q 与 %q", c.fn, g, c.group)
		}
		groups[c.fn] = c.group
	}

	var missing, extra []string
	for fn := range declared {
		if !registered[fn] {
			missing = append(missing, fn)
		}
	}
	for fn := range registered {
		if !declared[fn] {
			extra = append(extra, fn)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 {
		t.Errorf("引擎声明了这些写函数，但登记表里没有（新加的？补进 fnsCases）：\n  %s",
			strings.Join(missing, "\n  "))
	}
	if len(extra) > 0 {
		t.Errorf("登记表里有这些，但引擎不认它们是写函数（改名了？还是 manifest 变了）：\n  %s",
			strings.Join(extra, "\n  "))
	}
	t.Logf("引擎声明 %d 个写函数，登记表 %d 条 —— 一一对上", len(declared), len(registered))
}
