package form42f

// WP2：`.42f` 解析的**测试先行**（契约 C7）。
//
// 纪律：
//   - 仓库内只放**合成小夹具**（`testdata/form42f/`，无真实业务数据）；
//   - 真样本走 `TT_FORM_SAMPLE_DIR`（默认指向成果目录的 `fixtures/forms/`），缺失就 `Skip` 并说明原因（C7.3）；
//   - 其中 `TestParseForm42F_CollectsTableColumns` 是**反证测试**：把 `TableColumn` 过滤掉必须红（C7.5）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── 夹具读取 ────────────────────────────────────────────────────────────────

func readFixture42F(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "form42f", name))
	if err != nil {
		t.Fatalf("读合成夹具 %s 失败: %v", name, err)
	}
	return string(b)
}

// sampleDir42F 真样本目录（仓库外，契约 C7.3 / 变更记录 A5）。
func sampleDir42F(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("TT_FORM_SAMPLE_DIR"); d != "" {
		return d
	}
	return filepath.Join("..", "..", ".agents", "Genero Original Factory Compatible", "fixtures", "forms")
}

// corpusMap42F 归档语料表（契约 A5 的基线）。
func corpusMap42F(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("TT_FORM_CORPUS_MAP"); p != "" {
		return p
	}
	return filepath.Join("..", "..", ".agents", "Genero Original Factory Compatible",
		"out", "semantic", "colname-fieldid.json")
}

// ── 合成夹具的六条 ──────────────────────────────────────────────────────────

func TestParseForm42F_Synthetic(t *testing.T) {
	f, err := ParseForm42F("demoform", readFixture42F(t, "basic.xml"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if f.Name != "demoform" {
		t.Errorf("Name = %q, want demoform", f.Name)
	}
	if f.Build != "3.21.03" || f.Style != "dialog4" || f.Spacing != "normal" {
		t.Errorf("表头属性没带出来: build=%q style=%q spacing=%q", f.Build, f.Style, f.Spacing)
	}
	if f.Width != 58 || f.Height != 4 {
		t.Errorf("尺寸 = %dx%d, want 58x4", f.Width, f.Height)
	}
	if len(f.Fields) != 3 {
		t.Fatalf("字段条目 = %d, want 3（2 个 FormField + 1 个 TableColumn）: %+v", len(f.Fields), f.Fields)
	}

	// demo001：普通 Edit，几何要原样带出
	one := f.ByColName["demo001"]
	if one != 0 {
		t.Errorf("demo001 的 fieldId = %d, want 0", one)
	}
	if f.Fields[0].Widget != "Edit" {
		t.Errorf("demo001 的控件 = %q, want Edit", f.Fields[0].Widget)
	}
	if g := f.Fields[0].Geom; g.Width != 20 || g.PosX != 6 || g.PosY != 0 {
		t.Errorf("demo001 几何 = %+v, want width20 posX6 posY0", g)
	}

	// demo002：CheckBox + required/notNull + comment（在子控件上）+ text（在条目上）
	if f.ByColName["demo002"] != 1 {
		t.Errorf("demo002 的 fieldId = %d, want 1", f.ByColName["demo002"])
	}
	two := f.Fields[1]
	if two.Widget != "CheckBox" {
		t.Errorf("demo002 控件 = %q, want CheckBox", two.Widget)
	}
	if !two.Required || !two.NotNull {
		t.Errorf("demo002 该是 required+notNull: %+v", two)
	}
	if two.Comment != "cmt_demo002" {
		t.Errorf("demo002 comment = %q, want cmt_demo002（子控件上的属性也要收）", two.Comment)
	}
	if two.TextKey != "lbl_demo002" {
		t.Errorf("demo002 textKey = %q, want lbl_demo002", two.TextKey)
	}
	if two.SQLType != "CHAR" || two.TabIndex != 2 {
		t.Errorf("demo002 sqlType/tabIndex = %q/%d", two.SQLType, two.TabIndex)
	}

	// demo003：明细列（TableColumn）
	three := f.Fields[2]
	if three.Tag != "TableColumn" || three.ColName != "demo003" || three.FieldID != 2 {
		t.Errorf("demo003 = %+v, want TableColumn/demo003/2", three)
	}
	if three.Comment != "cmt_demo003" || three.TextKey != "lbl_demo003" {
		t.Errorf("demo003 comment/textKey = %q/%q", three.Comment, three.TextKey)
	}

	// 双键（契约 C1.5）：定位要走 ByKey，避免重名歧义
	if _, ok := f.ByKey["demo003#2"]; !ok {
		t.Errorf("ByKey 缺 demo003#2；现有键: %v", keysOf42(f.ByKey))
	}

	// RecordView + Link
	if len(f.RecordViews) != 1 || f.RecordViews[0].TabName != "demo_t" {
		t.Fatalf("RecordViews = %+v, want 1 个 demo_t", f.RecordViews)
	}
	if l := f.RecordViews[0].Links; len(l) != 1 || l[0].ColName != "demo003" || l[0].FieldIDRef != 2 {
		t.Errorf("RecordView links = %+v, want demo003→2", f.RecordViews[0].Links)
	}

	// LStr 键要收（后续可接字典）
	if !contains42(f.LStrKeys, "demo.title") || !contains42(f.LStrKeys, "demo.lbOne") {
		t.Errorf("LStrKeys 不全: %v", f.LStrKeys)
	}

	if f.Skipped != 0 {
		t.Errorf("Skipped = %d, want 0（这张表没有残缺条目）", f.Skipped)
	}
	if len(f.Duplicates) != 0 {
		t.Errorf("Duplicates = %v, want 空", f.Duplicates)
	}
}

// TestParseForm42F_CollectsTableColumns 是**反证测试**（契约 C1.3 / C7.5）。
//
// 它存在的全部意义：如果有人把 `TableColumn` 从"字段载体"里过滤掉（例如只认 `FormField`），
// 这条必须红。真实样本里 563 条有 397 条是 TableColumn —— 漏掉它们会得到
// "看起来对、其实错"的字段表（这是本轮唯一一个会被误判成成功的错误）。
func TestParseForm42F_CollectsTableColumns(t *testing.T) {
	f, err := ParseForm42F("onlycols", readFixture42F(t, "tablecolumn_only.xml"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(f.Fields) != 2 {
		t.Fatalf("字段条目 = %d, want 2（这张表**只有** TableColumn/PhantomColumn）: %+v", len(f.Fields), f.Fields)
	}
	if f.Fields[0].Tag != "TableColumn" {
		t.Errorf("第 1 条 Tag = %q, want TableColumn", f.Fields[0].Tag)
	}
	if f.Fields[1].Tag != "PhantomColumn" {
		t.Errorf("第 2 条 Tag = %q, want PhantomColumn", f.Fields[1].Tag)
	}
	if f.ByColName["demo101"] != 7 || f.ByColName["demo102"] != 8 {
		t.Errorf("TableColumn 的 colName→fieldId 没收到: %v", f.ByColName)
	}
}

func TestParseForm42F_ReportsDuplicateColName(t *testing.T) {
	f, err := ParseForm42F("dupform", readFixture42F(t, "duplicate_colname.xml"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(f.Fields) != 2 {
		t.Fatalf("字段条目 = %d, want 2（重名也要都留下）", len(f.Fields))
	}
	if len(f.Duplicates) != 1 || f.Duplicates[0] != "demo201" {
		t.Errorf("Duplicates = %v, want [demo201]（重名必须报出来，不能静默覆盖）", f.Duplicates)
	}
	if _, ok := f.ByKey["demo201#5"]; !ok {
		t.Errorf("ByKey 缺 demo201#5")
	}
	if _, ok := f.ByKey["demo201#6"]; !ok {
		t.Errorf("ByKey 缺 demo201#6")
	}
}

func TestParseForm42F_ToleratesMissingAttrs(t *testing.T) {
	f, err := ParseForm42F("partial", readFixture42F(t, "missing_attrs.xml"))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(f.Fields) != 1 || f.Fields[0].ColName != "a001" {
		t.Fatalf("应只留下 1 条完整条目(a001): %+v", f.Fields)
	}
	if f.Skipped != 3 {
		t.Errorf("Skipped = %d, want 3（缺 colName / 缺 fieldId / fieldId 非数字）", f.Skipped)
	}
}

func TestParseForm42F_RejectsNonXML(t *testing.T) {
	if _, err := ParseForm42F("notxml", readFixture42F(t, "not_xml.txt")); err == nil {
		t.Fatal("非 XML 必须报错，不能硬解")
	} else if !strings.Contains(err.Error(), "不是 XML") {
		t.Errorf("错误应说清「不是 XML」: %v", err)
	}
}

func TestParseForm42F_RejectsEmpty(t *testing.T) {
	if _, err := ParseForm42F("empty", readFixture42F(t, "empty.xml")); err == nil {
		t.Fatal("空内容必须报错")
	} else if !strings.Contains(err.Error(), "空") {
		t.Errorf("错误应说清是空的: %v", err)
	}
}

// ── 真样本（门控）────────────────────────────────────────────────────────────

func TestParseForm42F_T100Samples(t *testing.T) {
	dir := sampleDir42F(t)
	path := filepath.Join(dir, "t100", "axmt500_wf.42f")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("没有真样本（%s 打不开：%v）—— 设 TT_FORM_SAMPLE_DIR 指向 fixtures/forms/", path, err)
	}
	f, err := ParseForm42F("axmt500_wf", string(b))
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	if len(f.Fields) != 563 {
		t.Errorf("字段条目 = %d, want 563（TableColumn 397 + FormField 132 + PhantomFormField 22 + PhantomColumn 12）", len(f.Fields))
	}
	// 代表性锚点：这些值同时也是"运行期语料里观测到的值"（§18）
	want := map[ColName]FieldID{
		"b_xmda001":   18,
		"b_xmdadocdt": 17,
		"b_index":     25,
		"b_count":     26,
		"h_index":     559,
		"h_count":     560,
		"help":        14,
	}
	for col, id := range want {
		if got, ok := f.ByColName[col]; !ok {
			t.Errorf("表单里没有 %s（说明解析漏了）", col)
		} else if got != id {
			t.Errorf("表单里 %s = %d, want %d", col, got, id)
		}
	}
	if len(f.Duplicates) != 0 {
		t.Errorf("这张表单内不该有重名: %v", f.Duplicates)
	}
	t.Logf("真样本解析：%d 条字段、%d 个 RecordView、%d 个 LStr 键、跳过 %d 条",
		len(f.Fields), len(f.RecordViews), len(f.LStrKeys), f.Skipped)

	// Genero 自带的 4 个小表单也应能解析（它们小，顺序与条数都固定）
	for name, wantN := range map[string]int{"fglfind.42f": 4, "fglfind_tui.42f": 3, "fglhelp.42f": 1, "fglreport.42f": 2} {
		bb, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Logf("跳过 %s（打不开：%v）", name, err)
			continue
		}
		ff, err := ParseForm42F(strings.TrimSuffix(name, ".42f"), string(bb))
		if err != nil {
			t.Errorf("解析 %s 失败: %v", name, err)
			continue
		}
		if len(ff.Fields) != wantN {
			t.Errorf("%s 字段条目 = %d, want %d", name, len(ff.Fields), wantN)
		}
	}
}

// TestFormMatchesCorpus 是 M1 的验收测试（契约 C7.4 / 变更记录 A5）：
// 离线解析出来的 `colName→fieldId` 必须与归档语料表**逐条 100% 一致**。
//
// 口径（写死，避免"看起来对"）：
//   - **可比** = 语料里有、且这张表单里也有的 colName → **必须相等**（0 处不一致）；
//   - **仅在语料里** = 语料有、这张表单没有 → **允许**（同一个窗口里还有别的表单/开窗/动态节点）；
//   - 语料里标了 `ambiguous` 的 → 表单里的编号必须**落在候选里**。
func TestFormMatchesCorpus(t *testing.T) {
	raw, err := os.ReadFile(corpusMap42F(t))
	if err != nil {
		t.Skipf("没有归档语料表（%v）—— 设 TT_FORM_CORPUS_MAP 指向 colname-fieldid.json", err)
	}
	var doc struct {
		Map map[string]map[string]json.RawMessage `json:"map"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("语料表解析失败: %v", err)
	}
	win, ok := doc.Map["w_axmt500"]
	if !ok {
		t.Skip("归档语料表里没有 w_axmt500（换台机器/换份语料就跳）")
	}
	formPath := filepath.Join(sampleDir42F(t), "t100", "axmt500_wf.42f")
	b, err := os.ReadFile(formPath)
	if err != nil {
		t.Skipf("没有真样本（%v）", err)
	}
	f, err := ParseForm42F("axmt500_wf", string(b))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	var comparable, mismatches, corpusOnly, ambiguousOK, ambiguousBad int
	for col, v := range win {
		cn := ColName(col)
		formID, inForm := f.ByColName[cn]

		var n int
		if err := json.Unmarshal(v, &n); err == nil {
			if !inForm {
				corpusOnly++
				continue
			}
			comparable++
			if FieldID(n) != formID {
				mismatches++
				if mismatches <= 10 {
					t.Errorf("%s: 语料=%d 表单=%d", col, n, formID)
				}
			}
			continue
		}
		var amb struct {
			Ambiguous []struct {
				FieldID int `json:"fieldId"`
				Count   int `json:"count"`
			} `json:"ambiguous"`
		}
		if err := json.Unmarshal(v, &amb); err == nil && len(amb.Ambiguous) > 0 {
			if !inForm {
				continue
			}
			hit := false
			for _, c := range amb.Ambiguous {
				if FieldID(c.FieldID) == formID {
					hit = true
				}
			}
			if hit {
				ambiguousOK++
			} else {
				ambiguousBad++
				t.Errorf("%s: 表单=%d 不在候选里 %+v", col, formID, amb.Ambiguous)
			}
		}
	}

	if comparable < 500 {
		t.Errorf("可比的太少（%d）—— 说明解析或语料至少有一边不对", comparable)
	}
	if mismatches != 0 {
		t.Errorf("不一致 %d 处（必须 0）", mismatches)
	}
	if ambiguousBad != 0 {
		t.Errorf("歧义项里表单编号没落候选的 %d 处（必须 0）", ambiguousBad)
	}
	t.Logf("与归档语料表逐条比对：可比 %d 条 → **一致 %d 条（%.1f%%）**；不一致 %d；"+
		"歧义项命中候选 %d；仅在语料里 %d（同一个窗口里还有别的表单/开窗，允许）",
		comparable, comparable-mismatches, 100*float64(comparable-mismatches)/float64(comparable),
		mismatches, ambiguousOK, corpusOnly)
}

// ── 小工具 ──────────────────────────────────────────────────────────────────

func keysOf42(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func contains42(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
