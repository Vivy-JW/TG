package fgl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

/* ============================================================================
 * 夹具对拍：testdata/fgl-fixtures/*.4gl ↔ *.expected.json
 *
 * 夹具目录是 **BDL 扩展**（D:\我的项目\BDL，同作者，MIT）test/fixtures/ 的原样副本，
 * 期望值由 BDL 官方文档逐条推导，**不是**实现跑出来的快照；本文件不改动它们。
 *
 * 判据逐条镜像 BDL scripts/fixture-check.cjs：
 *   - 输出与 nodes 完全一致            → 通过
 *   - 输出不一致，但 JSON 里有 deviation → 已知偏差（不算失败）
 *   - 输出不一致且没有 deviation        → 不符（失败）
 * 已声明 deviation 的用例若现在通过了，就是普通「通过」（语义不变）。
 *
 * 基线（与 BDL 自身实测一致）：用例 61 个：通过 58，已知偏差 3，不符 0
 * 3 个已知偏差：dialog-subdialog、input-by-name-record-star、input-nested-semicolon。
 * ========================================================================== */

// kindName 复刻 extension.ts:442-451 的 SYM_KIND 映射（DocumentSymbol 种类名）。
var kindName = map[OutlineKind]string{
	KindFunction:  "Function",
	KindMain:      "Module",
	KindReport:    "Module",
	KindDialog:    "Interface",
	KindInput:     "Object",
	KindConstruct: "Object",
	KindDisplay:   "Array",
	KindMenu:      "Namespace",
	KindSub:       "Event",
}

type expNode struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	StartLine int       `json:"startLine"`
	EndLine   int       `json:"endLine"`
	Selection []int     `json:"selection"`
	Children  []expNode `json:"children"`
}

type expFixture struct {
	Doc       string    `json:"doc"`
	About     string    `json:"about"`
	Fixture   string    `json:"fixture"`
	Deviation string    `json:"deviation"`
	Nodes     []expNode `json:"nodes"`
}

// actNode 是与 expNode 同形的“实现输出”投影（fixture-check.cjs:90-97 的 toPlain）。
type actNode struct {
	Name      string
	Kind      string
	StartLine int
	EndLine   int
	Selection [2]int
	Children  []actNode
}

func toActNodes(nodes []OutlineNode) []actNode {
	out := make([]actNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, actNode{
			Name:      n.Label,
			Kind:      kindLabel(n.Kind),
			StartLine: n.Line - 1,
			EndLine:   n.EndLine - 1,
			Selection: [2]int{n.SelStart, n.SelEnd},
			Children:  toActNodes(n.Children),
		})
	}
	return out
}

// kindLabel 把 OutlineKind 映射成夹具里用的种类名；未知种类原样回落，便于报错时看清。
func kindLabel(k OutlineKind) string {
	if s, ok := kindName[k]; ok {
		return s
	}
	return string(k)
}

// compare 复刻 fixture-check.cjs:110-128，差异文案也保持一致（方便与 BDL 输出对齐）。
func compare(exp []expNode, act []actNode, at string, diffs *[]string) {
	n := len(exp)
	if len(act) > n {
		n = len(act)
	}
	for i := 0; i < n; i++ {
		where := fmt.Sprintf("%s[%d]", at, i)
		if i >= len(exp) {
			*diffs = append(*diffs, fmt.Sprintf("%s 多出节点 %s [%s L%d]",
				where, act[i].Name, act[i].Kind, act[i].StartLine+1))
			continue
		}
		if i >= len(act) {
			*diffs = append(*diffs, fmt.Sprintf("%s 缺少节点 %s [%s L%d]",
				where, exp[i].Name, exp[i].Kind, exp[i].StartLine+1))
			continue
		}
		e, a := exp[i], act[i]
		label := fmt.Sprintf("%s %s@L%d", where, e.Name, e.StartLine+1)
		if e.Name != a.Name {
			*diffs = append(*diffs, fmt.Sprintf("%s: name 期望 %q，实际 %q", label, e.Name, a.Name))
		}
		if e.Kind != a.Kind {
			*diffs = append(*diffs, fmt.Sprintf("%s: kind 期望 %s，实际 %s", label, e.Kind, a.Kind))
		}
		if e.StartLine != a.StartLine {
			*diffs = append(*diffs, fmt.Sprintf("%s: startLine 期望 %d，实际 %d", label, e.StartLine, a.StartLine))
		}
		if e.EndLine != a.EndLine {
			*diffs = append(*diffs, fmt.Sprintf("%s: endLine 期望 %d，实际 %d", label, e.EndLine, a.EndLine))
		}
		if e.Selection != nil && (e.Selection[0] != a.Selection[0] || e.Selection[1] != a.Selection[1]) {
			*diffs = append(*diffs, fmt.Sprintf("%s: selectionRange 期望 [%d,%d]，实际 [%d,%d]",
				label, e.Selection[0], e.Selection[1], a.Selection[0], a.Selection[1]))
		}
		compare(e.Children, a.Children, where+".children", diffs)
	}
}

// renderTree 与 fixture-check.cjs:99-106 的 renderTree 一致，失败时打印实际树。
func renderTree(nodes []actNode, depth int) []string {
	out := []string{}
	for _, n := range nodes {
		out = append(out, fmt.Sprintf("%s%s  [%s L%d-%d]",
			strings.Repeat("  ", depth), n.Name, n.Kind, n.StartLine+1, n.EndLine+1))
		out = append(out, renderTree(n.Children, depth+1)...)
	}
	return out
}

// fixturesDir 返回夹具目录（go test 的工作目录是包目录）。
//
// 合并进 tt 后包多了一层（internal/dev/fgl），写死的 "../.." 不再成立；
// 这里改为从包目录逐级上溯，找到含 testdata/fgl-fixtures 的那一级
// （夹具落在仓库根的 testdata/fgl-fixtures，与合并前位置约定一致）。
func fixturesDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("取不到当前目录: %v", err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "testdata", "fgl-fixtures")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("缺少夹具目录 testdata/fgl-fixtures（从包目录上溯未找到）")
	return ""
}

func TestParseOutlineFixtures(t *testing.T) {
	dir := fixturesDir(t)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取夹具目录失败: %v", err)
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".expected.json") {
			names = append(names, strings.TrimSuffix(e.Name(), ".expected.json"))
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("夹具目录 %s 里没有 *.expected.json", dir)
	}

	var pass, known, fatal int
	report := []string{}

	for _, name := range names {
		srcPath := filepath.Join(dir, name+".4gl")
		src, err := os.ReadFile(srcPath)
		if err != nil {
			fatal++
			report = append(report, fmt.Sprintf("\n❌ %s\n    - 缺少源文件 %s.4gl", name, name))
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name+".expected.json"))
		if err != nil {
			fatal++
			report = append(report, fmt.Sprintf("\n❌ %s\n    - 读取期望 JSON 失败: %v", name, err))
			continue
		}
		var exp expFixture
		if err := json.Unmarshal(raw, &exp); err != nil {
			fatal++
			report = append(report, fmt.Sprintf("\n❌ %s\n    - 期望 JSON 解析失败: %v", name, err))
			continue
		}

		actual := toActNodes(ParseOutline(string(src)))

		diffs := []string{}
		compare(exp.Nodes, actual, "root", &diffs)

		if len(diffs) == 0 {
			pass++
			continue
		}
		isKnown := exp.Deviation != ""
		if isKnown {
			known++
		} else {
			fatal++
		}
		head := "❌"
		if isKnown {
			head = "⚠️ "
		}
		var b strings.Builder
		fmt.Fprintf(&b, "\n%s %s  (%s)\n", head, name, exp.Doc)
		fmt.Fprintf(&b, "    考什么: %s\n", exp.About)
		if isKnown {
			fmt.Fprintf(&b, "    已知偏差: %s\n", exp.Deviation)
		}
		const maxDiffs = 12
		shown := diffs
		if len(shown) > maxDiffs {
			shown = shown[:maxDiffs]
		}
		for _, d := range shown {
			fmt.Fprintf(&b, "    - %s\n", d)
		}
		if len(diffs) > len(shown) {
			fmt.Fprintf(&b, "    … 另有 %d 处差异\n", len(diffs)-len(shown))
		}
		b.WriteString("    实际树:\n")
		for _, l := range renderTree(actual, 0) {
			fmt.Fprintf(&b, "      %s\n", l)
		}
		report = append(report, b.String())
	}

	t.Logf("用例 %d 个：通过 %d，已知偏差 %d，不符 %d", len(names), pass, known, fatal)
	if len(report) > 0 {
		t.Log(strings.Join(report, ""))
	}
	if fatal != 0 {
		t.Fatalf("失败: %d 个用例与文档推导的期望树不符（判据是文档，不是实现）", fatal)
	}
}

//---------------------------------------------------------------------------
// 两份副本的守卫
//---------------------------------------------------------------------------

// webFixturesDir 返回**前端那侧**的同名夹具目录（web/app/scripts/fgl-fixtures）。
//
// 与 fixturesDir 一样逐级上溯，只是找的是另一条路径。两份副本的存在是刻意的：
// Go 侧的对拍读仓库根的 testdata/，前端侧的 `npm run check:outline` 读它自己目录下那份
// （前端脚本在 esbuild 打包后跑，不该往仓库根去取数据）。
func webFixturesDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("取不到当前目录: %v", err)
	}
	for i := 0; i < 8; i++ {
		cand := filepath.Join(dir, "web", "app", "scripts", "fgl-fixtures")
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("缺少夹具目录 web/app/scripts/fgl-fixtures（从包目录上溯未找到）")
	return ""
}

// payloadNames 列出夹具目录里的**负载文件**（.4gl 与 .expected.json），升序。
//
// README 之类**不算负载**，不参与比对 —— 见下面那条测试的注释。
func payloadNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读 %s 失败：%v", dir, err)
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".4gl") || strings.HasSuffix(n, ".expected.json") {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// TestFixtureCopiesAreIdentical 两份 fgl-fixtures 必须**逐字节相同**。
//
// **为什么这条住在 Go 侧**：`go test ./...` 是默认档、会自动跑；前端侧没有默认档
// （`check:app` 是显式命令，web/package.json 里连 test 脚本都没有）。
// 放前端侧等于"只有人记得跑 check:app 时才生效"，那与今天的口头约定没本质差别。
//
// 两份目录的 README 都明写着"两份内容必须逐字节相同，改一处要同步另一处"，
// 而此前**没有任何东西在看** —— 这是全仓唯一一处"写成书面约定却没进测试"的一致性规则。
//
// 三条要留神的：
//   - **不比对 README**：两份 README 本来就不同（一份 Go 侧口吻、一份前端侧口吻），
//     testdata 那侧还多一个 README.source.md。把它们纳进来会立刻误报，
//     然后下一个人会"顺手"把这条测试删掉。
//   - **缺文件不 skip**：这两份属于仓库本体，不是语料那样的外部数据 ——
//     缺了就是仓库坏了（同 internal/cli/dev/tzs_verb_test.go:369 与
//     internal/cli/root_test.go:24 立的规矩）。
//   - 比对的是**集合**再加**逐字节**：只比内容不比名字，会漏掉"一边改名了"。
func TestFixtureCopiesAreIdentical(t *testing.T) {
	goDir := fixturesDir(t)
	webDir := webFixturesDir(t)

	goFiles := payloadNames(t, goDir)
	webFiles := payloadNames(t, webDir)

	if len(goFiles) == 0 {
		t.Fatalf("%s 里一个负载文件都没有 —— 夹具被删光了？", goDir)
	}

	// 1) 集合相等
	goSet := map[string]bool{}
	for _, n := range goFiles {
		goSet[n] = true
	}
	webSet := map[string]bool{}
	for _, n := range webFiles {
		webSet[n] = true
	}
	var onlyGo, onlyWeb []string
	for _, n := range goFiles {
		if !webSet[n] {
			onlyGo = append(onlyGo, n)
		}
	}
	for _, n := range webFiles {
		if !goSet[n] {
			onlyWeb = append(onlyWeb, n)
		}
	}
	if len(onlyGo) > 0 || len(onlyWeb) > 0 {
		t.Errorf("两份夹具的文件名集合不一致：\n"+
			"  只在 testdata/fgl-fixtures：%v\n"+
			"  只在 web/app/scripts/fgl-fixtures：%v\n\n"+
			"两份必须逐字节相同（见两份目录各自的 README）。改一处要同步另一处。",
			onlyGo, onlyWeb)
	}

	// 2) 逐字节相等
	for _, n := range goFiles {
		if !webSet[n] {
			continue // 上面已经报过集合不一致，这里不重复刷屏
		}
		goB, err := os.ReadFile(filepath.Join(goDir, n))
		if err != nil {
			t.Fatalf("读 %s 失败：%v", n, err)
		}
		webB, err := os.ReadFile(filepath.Join(webDir, n))
		if err != nil {
			t.Fatalf("读 %s 失败：%v", n, err)
		}
		if !bytes.Equal(goB, webB) {
			t.Errorf("两份 fgl-fixtures 不一致：%s 逐字节不同（%d vs %d 字节，首个差异在第 %d 字节）\n\n"+
				"两份必须逐字节相同 —— 见两份目录各自的 README。改一处要同步另一处：\n"+
				"  Copy-Item web\\app\\scripts\\fgl-fixtures\\%s testdata\\fgl-fixtures\\ -Force\n\n"+
				"（只比对 .4gl 与 .expected.json；两份 README 本来就不同，别改它们来「修」这条。）",
				n, len(goB), len(webB), firstDiff(goB, webB), n)
		}
	}
}

// firstDiff 返回两段字节第一个不同的位置（1-based）；相同则返回 0。
// 报"第几个字节不同"比报"长度不同"有用 —— 长度一样时后者是 0 信息。
func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i + 1
		}
	}
	if len(a) != len(b) {
		return n + 1
	}
	return 0
}
