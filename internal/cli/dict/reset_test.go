package dict

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/dbconfig"
	"tt/internal/testkit"
)

// 本包是**单例**：约 35 个包级 var（各命令的 flag 值、数据源落脚点、镜像配置……）
// 加一个 cobra 命令树 Group。真机上一条命令一个进程，所以这不是产品缺陷 ——
// 是测试必须自己补上的那一步。
//
// 这里提供两件东西：
//  1. resetDictPackageState(t)：快照/还原，用 t.Cleanup。**动过状态就必须调它。**
//  2. TestDictGlobalsAreAllInReset：读源码断言"没有哪个包级 var 落在清单之外"。
//     光有 (1) 是不够的 —— 漏一个 var 是**静默**泄漏，下一条用例拿到的值像是
//     "被测函数算错了"。

// snapVars 把一批同类型变量当前的值收下来，返回还原函数。
// 用泛型写一次，省得 35 个变量写 70 行存/取。
func snapVars[T any](ptrs ...*T) func() {
	old := make([]T, len(ptrs))
	for i, p := range ptrs {
		old[i] = *p
	}
	return func() {
		for i, p := range ptrs {
			*p = old[i]
		}
	}
}

// resetDictPackageState 让下一条用例看到一个**全新进程**该有的样子。
//
// 两类东西：
//   - cobra 的 flag 值挂在命令对象上 → testkit.ResetFlags(Group)（递归整棵树）
//   - 包级 var → 按类型分组快照，t.Cleanup 还原
//
// 还顺带还原 common.ConfigPath —— 它跨包，但 dict 的每条查询路径都要它，
// 而从前这里是**直接赋值不恢复**的（漏到过下一条用例）。
func resetDictPackageState(t *testing.T) {
	t.Helper()
	testkit.ResetFlags(Group)

	// 字符串
	t.Cleanup(snapVars(
		&dbPath, &srcConfigPath, &dictEnt, &srcDetail,
		&checkKW, &checkLang,
		&dbSyncTable, &discoverType, &discoverHost,
		&msgLang, &msgText, &msgType, &msgStatus, &msgProg,
		&paramLang, &specLang, &docListMode,
		&progKW, &progLang, &progTablesMode,
		&sccKW, &sccLang,
		&tableKW, &tableLang, &tableCols,
		&winKW, &winLang,
		&common.ConfigPath, &common.Env, &common.Format, &common.Version,
	))
	// 布尔
	t.Cleanup(snapVars(
		&discoverSave, &dbListSecrets, &msgFull,
		&progSub, &tableBrief, &tableWho, &tableFields, &sccBrief, &srcLocal, &dictFull,
		&common.JSON, &common.CSV, &common.Verbose,
	))
	// 整数
	t.Cleanup(snapVars(&queryLimit, &srcQueryLimit))
	// 剩下的几种类型各占一组（snapVars 是泛型的，每组只能放同一种类型）
	t.Cleanup(snapVars(&dbCfg))      // *host.Hosts
	t.Cleanup(snapVars(&dataSrc))    // db.Source（接口）
	t.Cleanup(snapVars(&srcTarget))  // *dbconfig.Target
	t.Cleanup(snapVars(&srcNotes))   // []string
	t.Cleanup(snapVars(&srcStarted)) // time.Time
	t.Cleanup(snapVars(&common.MetaProvider))
	t.Cleanup(snapVars(&common.WebFS))
}

// ---- 拒绝路径：这两条不需要库、不需要真机，却从来没被跑过 ----

// writeConfig 在临时目录写一份 config.json 并把 common.ConfigPath 指过去。
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	common.ConfigPath = p
	return p
}

// TestOpenQuerySourceRejectsMissingLocalDB `--env local` 而本地库文件不在：
// 报错要点名**找过哪些路径**、并给出下一步（设 TDICT_DB 或用 -d），
// 而不是一句"打不开"。
func TestOpenQuerySourceRejectsMissingLocalDB(t *testing.T) {
	resetDictPackageState(t)
	writeConfig(t, `{}`)
	common.Env = "local"
	// 指一个确定不存在的路径（不要依赖当前目录里恰好有没有 erp_data.db）。
	dbPath = filepath.Join(t.TempDir(), "肯定不存在.db")

	err := openQuerySource()
	if err == nil {
		t.Fatal("本地库不在时该报错")
	}
	msg := err.Error()
	for _, want := range []string{"本地数据库文件未找到", "TDICT_DB"} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误文案里该有 %q，得 %q", want, msg)
		}
	}
	// 失败时不该留下半个数据源。
	if dataSrc != nil {
		t.Error("打开失败不该留下 dataSrc")
	}
}

// TestOpenQuerySourceRejectsUnknownEnv `--env 没这个环境`：错误要来自环境解析，
// 而不是走到连接那一步才失败（那会先卡一次超时）。
func TestOpenQuerySourceRejectsUnknownEnv(t *testing.T) {
	resetDictPackageState(t)
	// 一份**没有 hosts** 的配置：解析环境时就会报"没有配置服务器环境"。
	writeConfig(t, `{"schemaVersion": 2}`)
	common.Env = "并不存在的环境"

	err := openQuerySource()
	if err == nil {
		t.Fatal("环境不存在时该报错")
	}
	if dataSrc != nil {
		t.Error("打开失败不该留下 dataSrc")
	}
	// 这条路径不该碰到网络 —— 错误文案里出现连接类词就说明走远了。
	if strings.Contains(err.Error(), "dial") || strings.Contains(err.Error(), "connect") {
		t.Errorf("该在解析阶段就失败，不该走到连接：%q", err.Error())
	}
}

// TestOpenQuerySourceLocalSucceedsWhenDBExists 正面对照：库文件在时能打开，
// 且 srcLocal / srcTarget 被如实填上（信封里的"这次查的是哪儿"全靠它）。
//
// 造一个空的 SQLite 文件就够 —— 这一步只验"打开"与"标注"，不查询。
func TestOpenQuerySourceLocalSucceedsWhenDBExists(t *testing.T) {
	resetDictPackageState(t)
	writeConfig(t, `{}`)
	common.Env = "local"
	// 空文件就够：resolveDBPath 只要求"文件在"，db.Open 走的是
	// sql.Open + Ping，SQLite 自己会把空文件初始化成库。
	p := filepath.Join(t.TempDir(), "erp_data.db")
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath = p

	if err := openQuerySource(); err != nil {
		t.Fatalf("空库该能打开：%v", err)
	}
	if dataSrc == nil {
		t.Fatal("该留下 dataSrc")
	}
	if !srcLocal {
		t.Error("本地源该标 srcLocal")
	}
	if srcTarget == nil || srcTarget.Route != dbconfig.RouteLocalSQLite || srcTarget.Path != p {
		t.Errorf("落脚点没标对：%+v", srcTarget)
	}
	if srcStarted.IsZero() {
		t.Error("该记下打开时刻（信封里的'用时'要用）")
	}
	closeQuerySource()
	if dataSrc != nil {
		t.Error("关闭之后该把 dataSrc 清掉")
	}
}

// ---- 清单齐全断言 ----

// dictMutableVars 是 resetDictPackageState **承诺**覆盖的包级 var。
//
// 改 resetDictPackageState 时必须同步这里 —— 下面那条测试就是逼你同步的。
// 清单本身写在这里而不是从函数里反射出来（Go 拿不到函数体的局部变量名），
// 所以它有残余缺口：往清单里加了名字却忘了真的在函数里还原，这条测不出来。
// 但它挡得住最常见的那种：**新加一个包级 var，两处都忘了**。
var dictMutableVars = []string{
	"dbPath", "srcConfigPath", "queryLimit", "srcQueryLimit", "dictEnt", "dictFull", "srcDetail",
	"checkKW", "checkLang",
	"dbSyncTable", "discoverType", "discoverHost", "discoverSave", "dbListSecrets",
	"msgLang", "msgText", "msgType", "msgStatus", "msgProg", "msgFull",
	"paramLang", "specLang", "docListMode",
	"progKW", "progLang", "progTablesMode", "progSub",
	"sccKW", "sccLang",
	"tableKW", "tableLang", "tableCols", "tableBrief", "tableWho", "tableFields", "sccBrief",
	"winKW", "winLang",
	"dataSrc", "srcLocal", "srcTarget", "srcNotes", "srcStarted",
	"dbCfg",
}

// dictReadOnlyVars 是**只读**的包级 var：常量表、列名、预编译的 help 函数。
// 它们不需要快照（没人改），列在这里是为了让上面那条"没落在清单之外"是**有意义**的。
var dictReadOnlyVars = []string{
	"helpCmdFamilies", "baseHelpFunc",
	"msgTypeValues", "msgStatusValues",
	"msgColumns", "specCSVHeaders", "tableDetailColumns", "tableSummaryColumns",
}

var (
	reSingleVar = regexp.MustCompile(`(?m)^var\s+([A-Za-z_]\w*)\s`)
	// `\r?\n` 里的 `\r?` 不是装饰：少写它，CRLF 工作树上**整块 var ( … ) 会静默消失**。
	// 实测（2026-09-27，一份最小复现）：同一份源码 LF 版找到 2 个名字，CRLF 版找到 **0 个**。
	// 触发条件只是开发者本机 `core.autocrlf=true`（Windows 常见），仓库里的 blob 一直是 LF，
	// 所以 Linux/CI 绿、他那台机器红 —— 症状与原因离得极远。
	//
	// 后果是**两条守卫同时失效**：TestDictVarListsHaveNoTypos 报假红（清单里的名字"不存在"），
	// 而 TestDictGlobalsAreAllInReset 报假绿（漏掉的 var 自然不会被判为"没有归属"）。
	// 见 TestScanVarsInIsEOLInvariant。
	reVarBlock  = regexp.MustCompile(`(?m)^var\s*\(\r?\n([\s\S]*?)^\)`)
	reBlockName = regexp.MustCompile(`(?m)^\t([A-Za-z_]\w*)[\s=]`)
)

// scanDictPackageVars 从本包的非测试源码里扫出全部包级 var 名。
//
// 扫源码而不是反射：反射拿不到"没被引用到的"包级 var（编译进二进制但没人提）。
// 扫出来的还包括 cobra 命令单例（Group 与各 *Cmd）—— 它们的 flag 由
// ResetFlags(Group) 递归覆盖，所以单独归一类。
func scanDictPackageVars(t *testing.T) []string {
	t.Helper()
	root := testkit.RepoRoot(t)
	dir := filepath.Join(root, "internal", "cli", "dict")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读 %s 失败：%v", dir, err)
	}
	seen := map[string]bool{}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读 %s 失败：%v", name, err)
		}
		for _, n := range scanVarsIn(string(b)) {
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	sort.Strings(out)
	return out
}

// scanVarsIn 从一份源码文本里扫出全部包级 var 名（升序、去重）。
//
// 单独抽出来，是为了能喂**内存里的**一段源码：scanDictPackageVars 读的是磁盘，
// 而这条扫描器最大的坑恰恰是**行尾** —— 见 TestScanVarsInIsEOLInvariant。
func scanVarsIn(src string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(n string) {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	// `var x = …` 与 `var x T`（单行）
	for _, m := range reSingleVar.FindAllStringSubmatch(src, -1) {
		add(m[1])
	}
	// `var ( … )` 块里的每个名字
	for _, blk := range reVarBlock.FindAllStringSubmatch(src, -1) {
		for _, m := range reBlockName.FindAllStringSubmatch(blk[1], -1) {
			add(m[1])
		}
	}
	sort.Strings(out)
	return out
}

// TestScanVarsInIsEOLInvariant 把"扫描器对行尾敏感"这件事钉死。
//
// **这条测试有来历**：2026-09-27 一位在 Windows 上评估本仓库的执行者报回来 ——
// 他那台机器 `core.autocrlf=true`，248 个 .go 全被 checkout 成 CRLF，于是
// `TestDictVarListsHaveNoTypos` 红了 30+ 行、而 `TestDictGlobalsAreAllInReset` **假绿**
// （漏掉的 var 自然不会被判为"没有归属"）。两条守卫同时失效，而仓库里的 blob 一直是 LF，
// 所以 Linux/CI 上永远看不见。
//
// 光改正则不够 —— 没有这条断言，下一个人把 `\r?` 删掉时不会有任何东西拦住他，
// 而这正是本仓库立的规矩："没见过它红的断言不算数"。
func TestScanVarsInIsEOLInvariant(t *testing.T) {
	lf := "package x\n\n" +
		"var single int\n\n" +
		"var (\n\tmirrorCfg     string\n\tmirrorCfgPath string\n)\n"
	crlf := strings.ReplaceAll(lf, "\n", "\r\n")

	want := "mirrorCfg,mirrorCfgPath,single"
	if got := strings.Join(scanVarsIn(lf), ","); got != want {
		t.Fatalf("LF 版扫出来 [%s]，期望 [%s] —— 连 LF 都不对，先查正则本身", got, want)
	}
	if got := strings.Join(scanVarsIn(crlf), ","); got != want {
		t.Errorf("CRLF 版扫出来 [%s]，期望 [%s]。\n"+
			"差的就是 `var ( … )` 那一块 —— CRLF 下 `var (\\r\\n` 让 `^var\\s*\\(\\n` 匹配不上，\n"+
			"整块静默消失（少了 `\\r?`）。后果不只是这条红：\n"+
			"TestDictGlobalsAreAllInReset 会**假绿**，因为漏掉的 var 不会被判为没有归属。", got, want)
	}
}

// isCobraVar 判断这个包级 var 是不是 cobra 命令单例（Group 与各 *Cmd）。
// 它们的"状态"是挂在命令对象上的 flag 值，由 ResetFlags(Group) 递归覆盖。
func isCobraVar(src string, name string) bool {
	return name == "Group" || strings.HasSuffix(name, "Cmd")
}

// TestDictGlobalsAreAllInReset 读源码断言：**每个包级 var 都有归属**。
//
// 三条出路，缺一不可：
//   - 在 dictMutableVars 里（→ resetDictPackageState 会还原）
//   - 在 dictReadOnlyVars 里（→ 只读，不需要还原）
//   - 是 cobra 命令单例（→ ResetFlags 覆盖它和它的 flag）
//
// 新加一个包级 var 而不选任何一条出路 → 红。这正是这条测试存在的理由：
// **漏一个 var 是静默的**，症状是下一条用例拿到上一条的残留值，看起来像被测函数算错了。
func TestDictGlobalsAreAllInReset(t *testing.T) {
	mutable := map[string]bool{}
	for _, n := range dictMutableVars {
		mutable[n] = true
	}
	readonly := map[string]bool{}
	for _, n := range dictReadOnlyVars {
		readonly[n] = true
	}

	var unaccounted []string
	all := scanDictPackageVars(t)
	for _, n := range all {
		if mutable[n] || readonly[n] || isCobraVar("", n) {
			continue
		}
		unaccounted = append(unaccounted, n)
	}
	if len(unaccounted) > 0 {
		t.Errorf("这些包级 var 没有归属：\n  %s\n\n"+
			"三条出路选一条：\n"+
			"  · resetDictPackageState 会还原它 → 加进 dictMutableVars\n"+
			"  · 它是只读的（常量表 / 列名）   → 加进 dictReadOnlyVars，并注明为什么只读\n"+
			"  · 它是 cobra 命令单例           → 名字以 Cmd 结尾即可\n\n"+
			"不加的话它是**静默**泄漏：下一条用例拿到这条留下的值，\n"+
			"而失败看起来像「被测函数算错了」。",
			strings.Join(unaccounted, "\n  "))
	}
}

// TestDictVarListsHaveNoTypos 反向：清单里的名字必须在源码里真的存在。
//
// 少了这条，删掉一个 var 之后清单会留着一个不存在的名字，
// 而上面那条测试照样绿 —— 清单就慢慢变成一份没人信的历史。
func TestDictVarListsHaveNoTypos(t *testing.T) {
	exists := map[string]bool{}
	for _, n := range scanDictPackageVars(t) {
		exists[n] = true
	}
	for _, n := range append(append([]string{}, dictMutableVars...), dictReadOnlyVars...) {
		if !exists[n] {
			t.Errorf("清单里列了 %q，但本包里没有这个名字（改名了？删了？同步清单）", n)
		}
	}
	// 同一个名字不该同时出现在两张表里。
	inMutable := map[string]bool{}
	for _, n := range dictMutableVars {
		inMutable[n] = true
	}
	for _, n := range dictReadOnlyVars {
		if inMutable[n] {
			t.Errorf("%q 同时在可变与只读两张表里 —— 必须是其中之一", n)
		}
	}
}
