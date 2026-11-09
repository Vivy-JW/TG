package split

import (
	"strings"
	"testing"

	"tt/internal/dev/model"
	"tt/internal/dev/tapfile"
)

// 本包的 README 说"正确性由端到端用例证明" —— 那些用例（改名事务、新增点、删除授权）
// 确实在跑，而且批 1 之后**默认档就跑**（tt dev tzc selftest 的 31 项已接进 go test）。
// 所以 Split 本身不用在这里重测一遍。
//
// 这里补的是**端到端钉不到的纯函数**：点名怎么拆、签名行怎么拼、order 怎么分配。
// 它们各自都有"差一个字就静默走偏"的地方（缩进、大小写、PUBLIC 省不省限定符），
// 而端到端只会告诉你"整条链最后不对"。

func TestIsSelfDef(t *testing.T) {
	yes := []string{"function.a", "dialog.b", "report.c", "function.", "function.a(1)"}
	for _, n := range yes {
		if !isSelfDef(n) {
			t.Errorf("%q 该算自订定义点", n)
		}
	}
	no := []string{"", "global.memo", "main.define", "other.function", "FUNCTION.a", "function", "xfunction.a"}
	for _, n := range no {
		if isSelfDef(n) {
			t.Errorf("%q 不该算自订定义点", n)
		}
	}
}

// TestBare 去掉种类前缀**与参数列表** —— 参数列表那条容易忘，
// 而点名带参数是设计器里的常态（`function.x(1)`）。
func TestBare(t *testing.T) {
	cases := map[string]string{
		"function.calc":    "calc",
		"dialog.popup":     "popup",
		"report.list":      "list",
		"function.calc(1)": "calc",
		"function.a(1,2)":  "a",
		"global.memo":      "global.memo", // 不是自订前缀，原样
		"calc":             "calc",
		"":                 "",
		"function.":        "",
	}
	for in, want := range cases {
		if got := bare(in); got != want {
			t.Errorf("bare(%q) = %q，想 %q", in, got, want)
		}
	}
}

// TestPrefixOfDefaultsToFunction 认不出前缀时返回 "function." ——
// 这是刻意的兜底，因为设计器只认这三类，认不出说明输入有问题，往最保守那边靠。
func TestPrefixOfDefaultsToFunction(t *testing.T) {
	cases := map[string]string{
		"function.a": "function.",
		"dialog.a":   "dialog.",
		"report.a":   "report.",
		"global.a":   "function.",
		"":           "function.",
	}
	for in, want := range cases {
		if got := prefixOf(in); got != want {
			t.Errorf("prefixOf(%q) = %q，想 %q", in, got, want)
		}
	}
}

// TestBuildSignature 钉住签名行的口径（对齐设计器 AddPointModel.ToString）。
//
// 两条容易写错的规则：
//   - FUNCTION **恒写**限定符；
//   - DIALOG/REPORT 在 PUBLIC 时**省略**限定符（写出来是 `DIALOG x` 而不是 `PUBLIC DIALOG x`）。
//
// 缩进从旧签名行原样继承 —— 它决定函数体在源码里的层叠，丢了就看不出嵌套。
func TestBuildSignature(t *testing.T) {
	cases := []struct {
		name         string
		oldSig, kind string
		scope, fn    string
		want         string
	}{
		{"FUNCTION 恒写限定符", "PRIVATE FUNCTION old(a)", "FUNCTION", "PRIVATE", "new", "PRIVATE FUNCTION new"},
		{"FUNCTION 即便 PUBLIC 也写出来", "PRIVATE FUNCTION old(a)", "FUNCTION", "PUBLIC", "new", "PUBLIC FUNCTION new"},
		{"DIALOG + PUBLIC 省略限定符", "DIALOG old()", "DIALOG", "PUBLIC", "new", "DIALOG new"},
		{"DIALOG + PRIVATE 写出限定符", "DIALOG old()", "DIALOG", "PRIVATE", "new", "PRIVATE DIALOG new"},
		{"REPORT + PUBLIC 同理省略", "REPORT old()", "REPORT", "PUBLIC", "new", "REPORT new"},
		{"REPORT + PRIVATE 写出限定符", "REPORT old()", "REPORT", "PRIVATE", "new", "PRIVATE REPORT new"},
		{"缩进原样继承（制表）", "\t\tPRIVATE FUNCTION old()", "FUNCTION", "PRIVATE", "new", "\t\tPRIVATE FUNCTION new"},
		{"缩进原样继承（空格）", "    DIALOG old()", "DIALOG", "PRIVATE", "new", "    PRIVATE DIALOG new"},
		{"旧行没有缩进就不加", "FUNCTION old()", "FUNCTION", "PRIVATE", "new", "PRIVATE FUNCTION new"},
		{"kind 为空时兜底 FUNCTION", "PRIVATE FUNCTION old()", "", "PRIVATE", "new", "PRIVATE FUNCTION new"},
		{"kind 大小写不敏感", "PRIVATE FUNCTION old()", "function", "PRIVATE", "new", "PRIVATE FUNCTION new"},
		{"scope 大小写不敏感（写出来一律大写）", "FUNCTION old()", "FUNCTION", "private", "new", "PRIVATE FUNCTION new"},
		{"旧行是小写关键字也能取到缩进", "\tprivate function old()", "FUNCTION", "PRIVATE", "new", "\tPRIVATE FUNCTION new"},
		{"参数列表不参与拼接（新签名只有名字）", "PRIVATE FUNCTION old(a, b, c)", "FUNCTION", "PRIVATE", "new", "PRIVATE FUNCTION new"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := BuildSignature(c.oldSig, c.kind, c.scope, c.fn); got != c.want {
				t.Errorf("得 %q，想 %q", got, c.want)
			}
		})
	}
}

// ---- order 分配 ----

// tapWith 造一份最小的 .tap 并解析出 Doc（走真实解析器，不手搓 Element ——
// 手搓的话测的是"我以为的 Element 形状"，不是解析器给的那个）。
func tapWith(t *testing.T, points string) *tapfile.Doc {
	t.Helper()
	raw := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\n" +
		`<add_points prog="p" type="M">` + "\n" + points + "\n</add_points>\n")
	doc, err := tapfile.Parse(raw)
	if err != nil {
		t.Fatalf("夹具解析失败：%v", err)
	}
	return doc
}

// TestMaxOrder 复刻 ProgramInformation.Add 的分配规则，四个筛子都得对：
// 跳过已删除的点、跳过非自订定义点、跳过没有 order 的点，最后取最大值 + 1。
func TestMaxOrder(t *testing.T) {
	doc := tapWith(t,
		`  <point name="function.a" order="2" status=""/>`+"\n"+
			`  <point name="function.b" order="5" status="d"/>`+"\n"+ // 已删除 → 跳过
			`  <point name="global.memo" order="9" status=""/>`+"\n"+ // 非自订 → 跳过
			`  <point name="function.c" order="3"/>`)
	if got := maxOrder(doc); got != 4 {
		t.Errorf("得 %d，想 4（max(2,3)+1；已删除的 5 与非自订的 9 都不算）", got)
	}

	// 一个自订定义点都没有 → 1（不是 0：order 从 1 起算）。
	if got := maxOrder(tapWith(t, `  <point name="global.memo" order="9" status=""/>`)); got != 1 {
		t.Errorf("没有自订定义点该给 1，得 %d", got)
	}
	// order 不是数字就跳过那一条，而不是报错。
	if got := maxOrder(tapWith(t, `  <point name="function.a" order="abc" status=""/>`)); got != 1 {
		t.Errorf("order 不是数字该跳过，得 %d", got)
	}
	// 带空白的 order 要能认（属性值里常见）。
	if got := maxOrder(tapWith(t, `  <point name="function.a" order=" 7 " status=""/>`)); got != 8 {
		t.Errorf("带空白的 order 该认，得 %d", got)
	}
}

// TestNextOrderCountsOnlySelfDefAdded 新增点要**连续**占号：
// 每新增一个自订定义点就在基线上再加一，这样一次写回里新增的多个点不会撞号。
func TestNextOrderCountsOnlySelfDefAdded(t *testing.T) {
	doc := tapWith(t, `  <point name="function.a" order="2" status=""/>`)

	plan := &Plan{Added: []string{"function.x"}}
	if got := nextOrder(doc, plan); got != 4 {
		t.Errorf("基线 3、新增 1 个自订点 → 该是 4，得 %d", got)
	}
	plan.Added = []string{"function.x", "dialog.y", "report.z"}
	if got := nextOrder(doc, plan); got != 6 {
		t.Errorf("新增 3 个自订点 → 该是 6，得 %d", got)
	}
	// 非自订点（裸名插入点）不占号。
	plan.Added = []string{"global.memo"}
	if got := nextOrder(doc, plan); got != 3 {
		t.Errorf("新增的不是自订定义点 → 不该占号，该是 3，得 %d", got)
	}
}

// ---- 错误类型：退出码是契约 ----

func TestErrorExitCodes(t *testing.T) {
	var ve error = &VerifyError{Msg: "校验没过"}
	var de error = &DeniedError{Msg: "没有授权"}
	if ec, ok := ve.(ExitCodeError); !ok || ec.ExitCode() != 3 {
		t.Errorf("VerifyError 该映射退出码 3，得 %v", ve)
	}
	if ec, ok := de.(ExitCodeError); !ok || ec.ExitCode() != 4 {
		t.Errorf("DeniedError 该映射退出码 4，得 %v", de)
	}
	// 渲染：有详情才带分隔符。
	if got := ve.Error(); got != "校验没过" {
		t.Errorf("没有详情时不该多分隔符，得 %q", got)
	}
	got := (&VerifyError{Msg: "校验没过", Detail: []string{"甲", "乙"}}).Error()
	if got != "校验没过：甲; 乙" {
		t.Errorf("得 %q", got)
	}
}

// TestPlanDescribe 摘要要按种类报数，且**没有改动时明说"无改动"** ——
// 空字符串会让人以为"计划没生成"，与"计划生成了但没有改动"是两件事。
func TestPlanDescribe(t *testing.T) {
	if got := (&Plan{}).Describe(); got != "无改动" {
		t.Errorf("空计划该说「无改动」，得 %q", got)
	}
	got := (&Plan{
		Renamed:  []string{"a → b"},
		Changed:  []string{"p1", "p2"},
		Added:    []string{"p3"},
		Deleted:  []string{"p4"},
		Sections: []string{"s1"},
	}).Describe()
	for _, want := range []string{"改名 1 个点", "改动 2 个点", "新增 1 个点", "删除 1 个点", "改动 1 个区段"} {
		if !strings.Contains(got, want) {
			t.Errorf("摘要里该有 %q，得 %q", want, got)
		}
	}
	// 区段那一条要把"同时写两处 + 置标志"说出来 —— 只看"改了 1 个区段"会以为只动了一处。
	if !strings.Contains(got, "TglTag 折叠") || !strings.Contains(got, "section_flag=Y") {
		t.Errorf("区段改动要说明它同时动 .tap 与 .tgl 并置标志，得 %q", got)
	}
}

// TestContentBoundsSafe 取正文时区间越界返回 nil 而不是 panic ——
// 它是给"文档与区间对不上"这种半坏状态兜底的。
func TestContentBoundsSafe(t *testing.T) {
	doc := &model.Document{Text: []byte("0123456789")}

	got := content(doc, &model.Region{ContentSpan: model.ByteRange{Start: 2, End: 5}})
	if string(got) != "234" {
		t.Errorf("得 %q，想 %q", got, "234")
	}
	if got := content(doc, nil); got != nil {
		t.Errorf("nil 区间该返回 nil，得 %q", got)
	}
	if got := content(doc, &model.Region{ContentSpan: model.ByteRange{Start: 2, End: 99}}); got != nil {
		t.Errorf("末端越界该返回 nil，得 %q", got)
	}
	if got := content(doc, &model.Region{ContentSpan: model.ByteRange{Start: 7, End: 3}}); got != nil {
		t.Errorf("起止颠倒该返回 nil，得 %q", got)
	}
}
