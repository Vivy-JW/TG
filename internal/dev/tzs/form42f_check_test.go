package tzs

// form42f_check_test.go —— `.42f`（应然）↔ tzs 生成物（实然）静态校验的守护测试。
//
// 契约：`.agents/plan/tt-genero-formdef-contract.md` 的 **C1/C7**（`.42f` 是编译期权威）、
// **A15**（启发式不得参与判定：拿不准就进"未判定"，且**不计入错误**）；用户决策 **D-NEXT**
// （M3 = 把离线字段表接到消费面，前端不做适配）。
//
// 七条（与任务书一一对应）：
//  1. TestCheckForm42F_Aligned            一致 → 零差异
//  2. TestCheckForm42F_FieldIDMismatch   编号不一致 → 归"编号"组 + "可能是另一张表单/版本不同"
//  3. TestCheckForm42F_MissingColName     tzs 多出一个 colName → 归"存在性"组
//  4. TestCheckForm42F_Writability        可写性冲突 → 归"可写性"组，**方向写清**
//  5. TestCheckForm42F_UnknownWidget     控件拿不准 → 进"未判定"，不计入错误（**反证对象**）
//  6. TestCheckForm42F_T100Sample        真样本门控（axmt500_wf.42f，缺失 t.Skip 写明原因）
//  7. 反证：把"拿不准就是未判定"改成"当成冲突" → 第 5 条必须红（命令与输出见报告）
//
// 另加三条低成本护栏（同属本文件要守的口径）：
//  - TestParseTzsForm4FD_RecordFieldIsNotAField：`.4fd` 的 `<RecordField>` 不算字段（它带的是 fieldIdRef）
//  - TestCheckForm42F_DuplicateColNameIsUndecided：两侧任一有重名 colName ⇒ "编号"归未判定（无法唯一对应）
//  - TestReadTzsFormPackage_MiniCorpus：真实设计器产出的 `.tzs` 包读得动（仓库自带的 3.2 MB 夹具）
//
// 第 8–12 条是**规则修正**（真实配对实测之后，契约 A37/A38）：
//  8. TestCheckForm42F_FieldIDRadixShiftIsOneNote  两侧恒差 +1 ⇒ 0 条"编号差异" + 1 条聚合说明（**反证对象 ①**）
//  9. TestCheckForm42F_FieldIDScatteredIsPerPair   差值不恒定（混一个 +3）⇒ 才逐条报
//  10. TestCheckForm42F_FieldIDSameRadixNoNote     两侧同编号 ⇒ 0 条、无聚合说明
//      TestCheckForm42F_FieldIDSinglePairIsStillADifference 只有 1 对 ⇒ 不谈基数（恒真），仍是确定差异
//  11. TestCheckForm42F_WritabilitySameAttributeIsCertain  **同一属性**反向（两者 notNull 都 false）⇒ 确定差异
//  12. TestCheckForm42F_WritabilityCrossAttributeIsUndecided `notNull=true`/`required=false` 跨属性组合 ⇒ 未判定（**反证对象 ②**）

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tt/internal/form42f"
)

// ── 合成夹具（内联；遵循 internal/form42f/form42f_test.go 的先例 的先例，B14）───────────────
//
// 三张字段：pmdl001（FormField + Edit，必填）、pmdl002（TableColumn + Edit）、
// pmdl003（FormField + CheckBox）。两侧**完全对齐**，任何一条差异都只能是测试改出来的。
const (
	// checkF42Text 是"应然"侧：编译后的表单（`.42f`）。注意 `.42f` 的字段条目**没有 widget
	// 属性** —— 控件类型来自它的第一个子元素（实测 axmt500_wf.42f 就是这样）。
	checkF42Text = `<?xml version='1.0' encoding='UTF-8'?>
<Form name="apcs001_wf.tmp" build="3.21.03" style="gui" width="80" height="24">
  <RecordView tabName="pmdl_t">
    <Link colName="pmdl001" fieldIdRef="18"/>
  </RecordView>
  <Page>
    <FormField name="formonly.pmdl001" colName="pmdl001" fieldId="18" sqlType="CHAR" sqlTabName="pmdl_t" notNull="1" required="1" noEntry="0" tabIndex="2">
      <Edit width="8" height="1" posX="7" posY="3" gridWidth="8" comment="cmt_pmdl001" text="lbl_pmdl001"/>
    </FormField>
    <Table name="pmdl_table">
      <TableColumn name="pmdl_t.pmdl002" colName="pmdl002" fieldId="23" sqlType="VARCHAR" sqlTabName="pmdl_t" notNull="0" required="0" noEntry="0">
        <Edit width="10" height="1" gridWidth="10"/>
      </TableColumn>
    </Table>
    <FormField name="formonly.pmdl003" colName="pmdl003" fieldId="31" sqlType="CHAR" notNull="0" required="0" noEntry="0">
      <CheckBox width="4" height="1" gridWidth="4" valueChecked="Y" valueUnchecked="N"/>
    </FormField>
  </Page>
</Form>
`

	// checkTzsText 是"实然"侧：tzs 生成物里的表单描述（`.tzs` 包内的 `.4fd`，明文 XML）。
	// 与 `.42f` 的差别（都是实测事实）：`widget` 是**属性**；`<RecordField>` 用 `fieldIdRef`
	// 指回字段而不带 `fieldId`，所以它不是字段条目。
	checkTzsText = `<?xml version='1.0' encoding='UTF-8'?>
<ManagedForm databaseName="ds" fileVersion="23200" name="managedform">
  <AGSettings />
  <Record name="Undefined" active="false">
    <RecordField name="pmdl001" fieldType="COLUMN_LIKE" colName="pmdl001" sqlType="CHAR" sqlTabName="pmdl_t" fieldIdRef="18" />
  </Record>
  <Form name="apcs001_wf">
    <Edit name="formonly.pmdl001" colName="pmdl001" fieldId="18" widget="Edit" noEntry="false" required="true" notNull="true" comment="cmt_pmdl001" text="lbl_pmdl001" tabIndex="2" posX="7" posY="3" gridWidth="8" />
    <Edit name="pmdl_t.pmdl002" colName="pmdl002" fieldId="23" widget="Edit" noEntry="false" required="false" notNull="false" />
    <CheckBox name="formonly.pmdl003" colName="pmdl003" fieldId="31" widget="CheckBox" noEntry="false" required="false" notNull="false" />
  </Form>
</ManagedForm>
`
)

// mustF42 解析合成 `.42f`（解析器是 internal/form42f 的，本包不重写）。
func mustF42(t *testing.T, text string) *form42f.F42Form {
	t.Helper()
	f, err := form42f.ParseForm42F("apcs001_wf", text)
	if err != nil {
		t.Fatalf("解析合成 .42f 失败：%v", err)
	}
	return f
}

// mustTzs 解析合成 tzs 侧描述。
func mustTzs(t *testing.T, text string) *TzsForm {
	t.Helper()
	f, err := ParseTzsForm4FD("apcs001_wf", text)
	if err != nil {
		t.Fatalf("解析合成 .4fd 失败：%v", err)
	}
	return f
}

// replace1 在夹具里做**恰好一次**替换；命中 0 次或多次都当场失败（免得测试悄悄不再测它想测的东西）。
func replace1(t *testing.T, s, old, new string) string {
	t.Helper()
	if n := strings.Count(s, old); n != 1 {
		t.Fatalf("夹具里 %q 出现 %d 次，want 1（夹具被改坏了）", old, n)
	}
	return strings.Replace(s, old, new, 1)
}

// ── 1. 一致：零差异 ─────────────────────────────────────────────────────────

func TestCheckForm42F_Aligned(t *testing.T) {
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, checkTzsText))

	if got := len(c.Findings); got != 0 {
		t.Fatalf("两侧完全对齐时不该有任何条目，得 %d 条：\n%s", got, c)
	}
	if !c.Consistent() {
		t.Errorf("两侧对齐时 Consistent() 该为 true")
	}
	if c.Pairs != 3 {
		t.Errorf("真正比过的 colName 对数 = %d，want 3（三张字段都该配上对）", c.Pairs)
	}
	if c.TzsUnnamed != 0 {
		t.Errorf("tzs 侧未命名条目 = %d，want 0", c.TzsUnnamed)
	}
	if len(c.F42Only) != 0 {
		t.Errorf(".42f 独有字段 = %v，want 空", c.F42Only)
	}
	// 渲染出来必须是"四组都在、都是 0 条"，而不是只有一句百分比。
	out := c.String()
	for _, want := range []string{"【存在性】0 条", "【编号】0 条", "【可写性】0 条", "【控件族】0 条", "未判定】0 条"} {
		if !strings.Contains(out, want) {
			t.Errorf("输出里缺 %q：\n%s", want, out)
		}
	}
}

// ── 2. 编号不一致 → "编号"组，且提示"另一张表单/版本不同" ────────────────────────

func TestCheckForm42F_FieldIDMismatch(t *testing.T) {
	// tzs 侧把 pmdl002 的编号写成 24（`.42f` 是 23）。
	tzsText := replace1(t, checkTzsText, `colName="pmdl002" fieldId="23"`, `colName="pmdl002" fieldId="24"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	diffs := c.Diffs()
	if len(diffs) != 1 {
		t.Fatalf("该有且只有 1 条差异，得 %d 条：\n%s", len(diffs), c)
	}
	d := diffs[0]
	if d.Rule != RuleFieldID {
		t.Errorf("该归到 %q 组，得 %q", RuleFieldID, d.Rule)
	}
	if d.ColName != "pmdl002" {
		t.Errorf("colName = %q，want pmdl002", d.ColName)
	}
	if d.F42 != "fieldId=23" || d.Tzs != "fieldId=24" {
		t.Errorf("两侧值要写清：.42f=%q tzs=%q，want fieldId=23 / fieldId=24", d.F42, d.Tzs)
	}
	for _, want := range []string{"另一张表单", "版本不同"} {
		if !strings.Contains(d.Why, want) {
			t.Errorf("理由里该提示 %q：%s", want, d.Why)
		}
	}
	if d.Undecided {
		t.Errorf("编号不一致是**确定**差异，不该是未判定")
	}
	if !strings.Contains(c.String(), "【编号】1 条") {
		t.Errorf("渲染里该把这条挂在【编号】组下：\n%s", c)
	}
}

// ── 3. 缺字段（tzs 多声明）→ "存在性"组 ────────────────────────────────────

func TestCheckForm42F_MissingColName(t *testing.T) {
	tzsText := replace1(t, checkTzsText, `  </Form>`,
		`    <Edit name="formonly.ghost" colName="ghost" fieldId="77" widget="Edit" noEntry="false" required="false" notNull="false" />
  </Form>`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	diffs := c.Diffs()
	if len(diffs) != 1 {
		t.Fatalf("该有且只有 1 条差异，得 %d 条：\n%s", len(diffs), c)
	}
	d := diffs[0]
	if d.Rule != RuleExist {
		t.Errorf("该归到 %q 组，得 %q", RuleExist, d.Rule)
	}
	if d.ColName != "ghost" {
		t.Errorf("colName = %q，want ghost", d.ColName)
	}
	if d.Tzs == "" {
		t.Errorf("实然侧（tzs）该写出它声明了什么：%+v", d)
	}
	if !strings.Contains(d.Why, "不存在") && !strings.Contains(d.Why, "没有") {
		t.Errorf("理由该说清 .42f 里没有这个栏目：%s", d.Why)
	}
	if len(c.F42Only) != 0 {
		t.Errorf("反方向（.42f 独有）该是空的：%v", c.F42Only)
	}
}

// ── 4. 可写性冲突 → "可写性"组，方向明确 ───────────────────────────────────

func TestCheckForm42F_Writability(t *testing.T) {
	// `.42f` 侧 pmdl001 是 noEntry="0"（可写）；tzs 侧改成 noEntry="true"（不可输入）。
	tzsText := replace1(t, checkTzsText, `colName="pmdl001" fieldId="18" widget="Edit" noEntry="false"`,
		`colName="pmdl001" fieldId="18" widget="Edit" noEntry="true"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	diffs := c.Diffs()
	if len(diffs) != 1 {
		t.Fatalf("该有且只有 1 条差异，得 %d 条：\n%s", len(diffs), c)
	}
	d := diffs[0]
	if d.Rule != RuleWritable {
		t.Errorf("该归到 %q 组，得 %q", RuleWritable, d.Rule)
	}
	// 方向：F42 是应然（.42f），Tzs 是实然（tzs）。两侧都必须写出属性名与值。
	if !strings.Contains(d.F42, "noEntry") || !strings.Contains(d.F42, "false") {
		t.Errorf("应然侧该写 noEntry=false，得 %q", d.F42)
	}
	if !strings.Contains(d.Tzs, "noEntry") || !strings.Contains(d.Tzs, "true") {
		t.Errorf("实然侧该写 noEntry=true，得 %q", d.Tzs)
	}
	if !strings.Contains(d.Why, ".42f") || !strings.Contains(d.Why, "tzs") {
		t.Errorf("理由里两侧来源都要点名（哪侧是 .42f、哪侧是 tzs）：%s", d.Why)
	}
	if !strings.Contains(c.String(), "【可写性】1 条") {
		t.Errorf("渲染里该挂在【可写性】组下：\n%s", c)
	}
}

// ── 4b. 控件族**确定**冲突 → "控件族"组（两侧都是已知控件时才报）─────────────────
//
// 这一条是补出来的：第一次跑演示时输出成了 `CheckBox（）` —— 族说明取错了键（拿归一键去查
// 原表），说明"已知控件冲突"这条路当时**没有任何测试走过**。补上它，顺带钉住族说明非空。
func TestCheckForm42F_WidgetFamilyConflict(t *testing.T) {
	// tzs 侧把 pmdl003 写成 Edit（`.42f` 是 CheckBox）：两个名字都在已知表里 ⇒ 确定冲突。
	tzsText := replace1(t, checkTzsText, `colName="pmdl003" fieldId="31" widget="CheckBox"`,
		`colName="pmdl003" fieldId="31" widget="Edit"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	diffs := c.Diffs()
	if len(diffs) != 1 {
		t.Fatalf("该有且只有 1 条差异，得 %d 条：\n%s", len(diffs), c)
	}
	d := diffs[0]
	if d.Rule != RuleWidget {
		t.Errorf("该归到 %q 组，得 %q", RuleWidget, d.Rule)
	}
	if d.Undecided {
		t.Errorf("两侧都是已知控件时该是**确定**差异，不是未判定")
	}
	for _, want := range []string{"布尔勾选", "文本输入"} {
		if !strings.Contains(d.F42+d.Tzs+d.Why, want) {
			t.Errorf("两侧的族说明该出现 %q（族说明取错了键就会空成「（）」）：F42=%q Tzs=%q Why=%s",
				want, d.F42, d.Tzs, d.Why)
		}
	}
	if strings.Contains(d.F42, "（）") || strings.Contains(d.Tzs, "（）") {
		t.Errorf("族说明空了：F42=%q Tzs=%q", d.F42, d.Tzs)
	}
	// 确定等价（FF 前缀）**不该**报：`FFLabel` ↔ `Label` 是同一个东西。
	f42FF := replace1(t, checkF42Text, `<CheckBox width="4"`, `<FFLabel width="4"`)
	tzsLabel := replace1(t, checkTzsText, `widget="CheckBox"`, `widget="Label"`)
	if got := CheckForm42F(mustF42(t, f42FF), mustTzs(t, tzsLabel)).Diffs(); len(got) != 0 {
		t.Errorf("FFLabel ↔ Label 是确定等价，不该报差异：%+v", got)
	}
}

// ── 5. 拿不准 ⇒ "未判定"，**不计入错误**（反证对象）────────────────────────────
//
// 构造：`.42f` 侧 pmdl003 的子控件元素名改成 FancyKnob（一个我们的映射表里没有的名字），
// tzs 侧仍是 widget="CheckBox"。两边都"有控件类型"，但**这个组合我们拿不准**是否等价 ⇒
// 必须进"未判定"，而不是冲突（A15）。
func TestCheckForm42F_UnknownWidget(t *testing.T) {
	f42Text := replace1(t, checkF42Text, `<CheckBox width="4"`, `<FancyKnob width="4"`)
	c := CheckForm42F(mustF42(t, f42Text), mustTzs(t, checkTzsText))

	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("拿不准的控件组合**不得**算作冲突，得 %d 条差异：\n%v", len(diffs), diffs)
	}
	if !c.Consistent() {
		t.Errorf("只有未判定时 Consistent() 该为 true（未判定不计入错误）")
	}
	und := c.Undecided()
	if len(und) != 1 {
		t.Fatalf("该有 1 条未判定，得 %d 条：\n%s", len(und), c)
	}
	u := und[0]
	if u.Rule != RuleWidget {
		t.Errorf("未判定条目该记在控件族这条规则下，得 %q", u.Rule)
	}
	if u.ColName != "pmdl003" {
		t.Errorf("colName = %q，want pmdl003", u.ColName)
	}
	if !strings.Contains(u.F42, "FancyKnob") || !strings.Contains(u.Tzs, "CheckBox") {
		t.Errorf("未判定也要写清两侧值：%+v", u)
	}
	if !strings.Contains(u.Why, "不确定") && !strings.Contains(u.Why, "没有换算") && !strings.Contains(u.Why, "不在") {
		t.Errorf("理由该说清为什么拿不准：%s", u.Why)
	}
	out := c.String()
	if !strings.Contains(out, "【未判定】1 条") {
		t.Errorf("渲染里该有独立的未判定段：\n%s", out)
	}
	if strings.Contains(out, "【控件族】1 条") {
		t.Errorf("未判定不该出现在【控件族】的差异段里：\n%s", out)
	}
}

// ── 6. 真样本门控（axmt500_wf.42f）─────────────────────────────────────────
//
// 这一条为什么这么写（**诚实说明**）：仓库里**没有** `axmt500_wf` 的 tzs 生成物
// （`.tzs` 包 / `.4fd` 一份都没有，只有 7 份 `.42f`，见 fixtures/forms/）。
// 所以"实然"侧用 `.42f` 自己投影出来（同名字段、同一批控件名），断言的是：
//
//	① 真样本 563 条逐条都能配成对（没有一条掉在配对之外）；
//	② 四个差异组全 0；
//	③ "未判定"只会出现在控件族这一条上，而且条数等于 `.42f` 侧**没带控件类型**的条目数
//	   （`TableColumn`/`PhantomColumn` 这类自闭元素，实测它们没有子控件）—— 这条是可反证的量。
//
// 它证明不了"某份真 .4fd 与这份 .42f 一致"（没有那份 .4fd）；那件事要等真包到手。
func TestCheckForm42F_T100Sample(t *testing.T) {
	path := filepath.Join(tzsCheckSampleDir(t), "t100", "axmt500_wf.42f")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("没有真样本（%s 打不开：%v）—— 设 TT_FORM_SAMPLE_DIR 指向 fixtures/forms/", path, err)
	}
	f, err := form42f.ParseForm42F("axmt500_wf", string(b))
	if err != nil {
		t.Fatalf("解析 %s 失败：%v", path, err)
	}
	if len(f.Fields) != 563 {
		t.Fatalf("真样本条目 = %d，want 563（与契约 §0 B3 的实测一致）", len(f.Fields))
	}

	c := CheckForm42F(f, projectF42ToTzs(f))
	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("自投影应当零差异，得 %d 条：\n%v", len(diffs), diffs)
	}
	if c.Pairs != len(f.ByColName) {
		t.Errorf("配对对数 = %d，want %d（= ByColName 的唯一栏目数）", c.Pairs, len(f.ByColName))
	}

	// 可反证的量：未判定只许来自"控件名不在已知表"这一种情况 —— 真样本里有几个控件名我们
	// 还没收录（实测 34/563），**必须**进未判定而不是差异，而且比例要小（"认得的名"覆盖率
	// 是这张表唯一的验收量）。
	wantUndecided := 0
	for _, fld := range f.Fields {
		if _, _, ok := widgetCanon(fld.Widget); !ok {
			wantUndecided++
		}
	}
	if got := len(c.Undecided()); got != wantUndecided {
		t.Errorf("未判定 = %d 条，want %d（= .42f 里控件名不在已知表的条目数）", got, wantUndecided)
	}
	if wantUndecided*10 > len(f.Fields) {
		t.Errorf("未判定 %d/%d 超过 10%%：控件名映射表该补了，否则这条校验大半在「不判」", wantUndecided, len(f.Fields))
	}
	for _, u := range c.Undecided() {
		if u.Rule != RuleWidget {
			t.Errorf("未判定只该出现在控件族规则上，得 %q（%s）", u.Rule, u.ColName)
		}
	}
	hist := map[string]int{}
	for _, fld := range f.Fields {
		hist[fld.Widget]++
	}
	names := make([]string, 0, len(hist))
	for name, n := range hist {
		names = append(names, fmt.Sprintf("%s×%d", name, n))
	}
	sort.Strings(names)
	t.Logf("真样本：.42f %d 条 / 配对 %d 对 / 差异 0 条 / 未判定 %d 条（都是控件名不在已知表）", len(f.Fields), c.Pairs, len(c.Undecided()))
	t.Logf("真样本的控件分布：%s", strings.Join(names, " "))
}

// ── 另加的三条护栏 ────────────────────────────────────────────────────────

// TestParseTzsForm4FD_RecordFieldIsNotAField：`<RecordField>` 带的是 `fieldIdRef` 而不是
// `fieldId` —— 它是**记录布局**，不是表单字段条目。若把它也收成字段，每张字段都会被数两遍。
func TestParseTzsForm4FD_RecordFieldIsNotAField(t *testing.T) {
	f := mustTzs(t, checkTzsText)
	if len(f.Fields) != 3 {
		t.Fatalf("字段条目 = %d，want 3（RecordField 不算）：%+v", len(f.Fields), f.Fields)
	}
	for _, fld := range f.Fields {
		if fld.Tag == "RecordField" {
			t.Errorf("RecordField 不该进字段表：%+v", fld)
		}
	}
	// 缺 fieldId（不可解析）的元素要被跳过并计数，绝不猜（对齐 `.42f` 的 C1.6 口径）。
	// 注意口径：**没有 `fieldId` 属性的元素不算"跳过"** —— 那 127 个 Record/Grid/Label 本来
	// 就不是字段条目（把它们算进"跳过"只会让计数变成噪音）。
	bad := `<ManagedForm><Form name="x"><Edit colName="a" fieldId="oops" widget="Edit"/></Form></ManagedForm>`
	bf, err := ParseTzsForm4FD("x", bad)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if len(bf.Fields) != 0 || bf.Skipped != 1 {
		t.Errorf("fieldId 不是整数的元素该被跳过并计数：fields=%d skipped=%d，want 0/1", len(bf.Fields), bf.Skipped)
	}
	// 非 XML / 空内容要给明确错误（不 panic）
	if _, err := ParseTzsForm4FD("x", "这不是 XML"); err == nil {
		t.Errorf("垃圾输入该报错")
	}
	if _, err := ParseTzsForm4FD("x", "  "); err == nil {
		t.Errorf("空内容该报错")
	}
	if _, err := ParseTzsForm4FD("x", `<ManagedForm><Record/></ManagedForm>`); err == nil {
		t.Errorf("没有 <Form> 的 XML 该报错（不是 .4fd）")
	}
}

// TestCheckForm42F_DuplicateColNameIsUndecided：重名 colName 无法唯一对应 ⇒ 编号归未判定。
//
// 依据：`.42f` 自己也有 Duplicates（同栏位绑多个控件，实测 aapt300.4fd 里 chr80 出现 4 次），
// 所以"多命中"是**正常形状**，不能当冲突。
func TestCheckForm42F_DuplicateColNameIsUndecided(t *testing.T) {
	f42Text := replace1(t, checkF42Text, `</Page>`,
		`  <FormField name="formonly.pmdl003b" colName="pmdl003" fieldId="32" noEntry="0"><Edit width="4"/></FormField>
  </Page>`)
	tzsText := replace1(t, checkTzsText, `  </Form>`,
		`    <Edit name="formonly.pmdl003b" colName="pmdl003" fieldId="32" widget="Edit" noEntry="false" required="false" notNull="false" />
  </Form>`)
	c := CheckForm42F(mustF42(t, f42Text), mustTzs(t, tzsText))

	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("重名时不该报确定差异，得 %d 条：\n%s", len(diffs), c)
	}
	var got []Finding
	for _, f := range c.Undecided() {
		if f.Rule == RuleFieldID && f.ColName == "pmdl003" {
			got = append(got, f)
		}
	}
	if len(got) != 1 {
		t.Fatalf("pmdl003 重名该有 1 条「编号」未判定，得 %d 条：\n%s", len(got), c)
	}
	if !strings.Contains(got[0].Why, "重名") && !strings.Contains(got[0].Why, "重复") {
		t.Errorf("理由该说清是重名导致无法唯一对应：%s", got[0].Why)
	}
}

// TestReadTzsFormPackage_MiniCorpus：真实设计器产出的 `.tzs`（zip）读得动。
//
// 夹具是仓库本体（testdata/tzs-mini/ws，3.2 MB，见 mini_corpus_test.go），所以**缺了就是仓库坏了**；
// 这里仍给 t.Skip 的退路是因为它属于"另一条切片的地盘"，不该让本切片因别人的夹具迁移而红。
func TestReadTzsFormPackage_MiniCorpus(t *testing.T) {
	root := filepath.Join(tzsCheckRepoRoot(t), "testdata", "tzs-mini", "ws")
	pkg := filepath.Join(root, "aapp320(c).tzs")
	if _, err := os.Stat(pkg); err != nil {
		t.Skipf("没有仓库自带的最小语料（%s：%v）—— 它是仓库本体，缺了用 `go run ./tools/tzsmini` 重新生成", pkg, err)
	}
	f, err := ReadTzsFormPackage(pkg)
	if err != nil {
		t.Fatalf("读 %s 失败：%v", pkg, err)
	}
	if f.Prog != "aapp320" {
		t.Errorf("程序名 = %q，want aapp320（从包内 .4fd 文件名推）", f.Prog)
	}
	// 实测（2026-10-17，本文件作者逐元素数过 aapp320.4fd）：**56** 个带 `fieldId` 的元素，
	// 其中 **45** 个有非空 colName（另 11 个是 NON_DATABASE / 无栏位身份的占位控件）；
	// `RecordField` 是 56 条**另一批**元素（带 `fieldIdRef` 不带 `fieldId`），不能算进来。
	if len(f.Fields) != 56 {
		t.Fatalf("字段条目 = %d，want 56", len(f.Fields))
	}
	if got := len(f.Fields) - f.Unnamed(); got != 45 {
		t.Errorf("有 colName 的条目 = %d，want 45", got)
	}
	if f.Skipped != 0 {
		t.Errorf("Skipped = %d，want 0（这个包里没有 fieldId 非整数的元素）", f.Skipped)
	}
	// 写死几个锚点，防止解析器"读到了但字段全错位"。
	anchors := map[form42f.ColName]form42f.FieldID{"apcasite": 57, "apcald": 61, "ooef001": 60}
	idx := map[form42f.ColName]TzsField{}
	for _, fld := range f.Fields {
		idx[fld.ColName] = fld
	}
	for col, id := range anchors {
		fld, ok := idx[col]
		if !ok {
			t.Errorf("包里没有 %s（解析漏了）", col)
			continue
		}
		if fld.FieldID != id {
			t.Errorf("%s 的 fieldId = %d，want %d", col, fld.FieldID, id)
		}
		if fld.Widget == "" {
			t.Errorf("%s 的控件类型是空的（`.4fd` 的 widget 属性没读到）", col)
		}
	}
	t.Logf("%s：%d 条字段（%d 条有 colName，%d 条跳过），程序名 %s", filepath.Base(pkg), len(f.Fields), len(f.Fields)-f.Unnamed(), f.Skipped, f.Prog)
}

// ── 8–10. 编号：基数感知（契约 A37）─────────────────────────────────────────
//
// 为什么这条规则要改（真实配对 `aimm200_wf` 实测，验收清单 §16）：`.42f` 的 `fieldId` 是 0..170
// （0 基）、`.4fd` 的是 1..171（1 基），逐行对齐 **171 行恒差 +1**。裸比两侧 `fieldId` 时
// **每一条唯一配对都命中**（86 条"编号不同"），对"是不是另一张表单"零分辨力。
// 所以判据改成"先看差值是否恒定"：恒定 ⇒ 一条聚合说明（不逐条报）；散 ⇒ 才逐条报。

// radixPair 造一对**同序**夹具：`shifts[i]` 是第 i 个字段在 tzs 侧的编号偏移。
//
// `.42f` 侧编号固定 10,11,12…；tzs 侧 = 该值 + shifts[i]。三侧属性一律对齐（编号是唯一变量），
// 所以除了"编号"这一组，别的规则都不该有话说。
func radixPair(t *testing.T, shifts ...int) (*form42f.F42Form, *TzsForm) {
	t.Helper()
	var f42, tzs strings.Builder
	f42.WriteString(`<?xml version='1.0' encoding='UTF-8'?>` + "\n<Form name=\"apcr001_wf.tmp\">\n  <Page>\n")
	tzs.WriteString(`<?xml version='1.0' encoding='UTF-8'?>` + "\n<ManagedForm>\n  <Form name=\"apcr001_wf\">\n")
	for i, shift := range shifts {
		id, col := 10+i, fmt.Sprintf("prmd%03d", i+1)
		fmt.Fprintf(&f42, `    <FormField name="formonly.%s" colName="%s" fieldId="%d" notNull="0" required="0" noEntry="0">`+"\n      <Edit width=\"8\"/>\n    </FormField>\n",
			col, col, id)
		fmt.Fprintf(&tzs, `    <Edit name="formonly.%s" colName="%s" fieldId="%d" widget="Edit" noEntry="false" required="false" notNull="false" />`+"\n",
			col, col, id+shift)
	}
	f42.WriteString("  </Page>\n</Form>\n")
	tzs.WriteString("  </Form>\n</ManagedForm>\n")
	return mustF42(t, f42.String()), mustTzs(t, tzs.String())
}

// TestCheckForm42F_FieldIDRadixShiftIsOneNote：**恒差 +1** ⇒ 0 条"编号差异" + 1 条聚合说明。
//
// 这是 A37 的正题：真实配对里那 86 条全是这个形状。断言三件事：
//
//	① "编号"组**一条确定差异都没有**（不是"少报几条"，是全部降级）；
//	② 恰好 **1 条**聚合说明，且文案里写清"恒差"与 "+1"（措辞只陈述事实，不猜哪边是 0 基）；
//	③ **信息没丢**：说明里带配对数与样例 colName，渲染里真的印出来。
//
// 反证（报告里有命令）：把基数检测去掉（恒差也逐条报）⇒ ①红。
func TestCheckForm42F_FieldIDRadixShiftIsOneNote(t *testing.T) {
	want, got := radixPair(t, 1, 1, 1, 1)
	c := CheckForm42F(want, got)

	if n := len(c.Group(RuleFieldID)); n != 0 {
		t.Fatalf("两侧恒差 +1 时「编号」不该有确定差异，得 %d 条（A37：恒差 ⇒ 一条聚合说明，不逐条报）：\n%v",
			n, c.Group(RuleFieldID))
	}
	if !c.Consistent() {
		t.Errorf("只有聚合说明时 Consistent() 必须为 true（⇒ CLI 退 0；说明不是差异）：\n%s", c)
	}
	notes := c.NoteFor(RuleFieldID)
	if len(notes) != 1 {
		t.Fatalf("恒差 +1 该有且只有 1 条「编号」聚合说明，得 %d 条（Notes=%v）：\n%s", len(notes), c.Notes, c)
	}
	if len(c.Notes) != 1 {
		t.Errorf("聚合说明总数 = %d，want 1（这条夹具只该造出基数这一条）", len(c.Notes))
	}
	n := notes[0]
	for _, kw := range []string{"恒差", "+1", "基数"} {
		if !strings.Contains(n.Text, kw) {
			t.Errorf("聚合说明里该写清 %q（措辞只陈述事实）：%s", kw, n.Text)
		}
	}
	if n.Rule != RuleFieldID {
		t.Errorf("聚合说明该挂在「编号」这条规则下，得 %q", n.Rule)
	}
	// ③ 信息没丢：条数与样例 colName 都在（不然就是"改到不报就完事"）。
	if n.Pairs != len(want.Fields) {
		t.Errorf("聚合说明里的配对数 = %d，want %d（= 唯一配对数）", n.Pairs, len(want.Fields))
	}
	if len(n.Cols) == 0 {
		t.Errorf("聚合说明该带样例 colName（否则人无法抽查）：%+v", n)
	}
	if !strings.Contains(n.Text, "prmd001") {
		t.Errorf("聚合说明里该出现样例 colName prmd001：%s", n.Text)
	}
	// 渲染：说明真的印出来，而不是只活在结构里。
	out := c.String()
	if !strings.Contains(out, "【说明】1 条") || !strings.Contains(out, "恒差 +1") {
		t.Errorf("渲染里该有【说明】段且写出「恒差 +1」：\n%s", out)
	}
	if !strings.Contains(out, "【编号】0 条") {
		t.Errorf("渲染里「编号」组该是 0 条：\n%s", out)
	}
}

// TestCheckForm42F_FieldIDScatteredIsPerPair：差值**不恒定** ⇒ 才逐条报（这才是真信号）。
//
// 夹具：4 对，差 +1/+1/**+3**/+1 —— 只要有一条不跟着整体平移，就不是基数为题。
func TestCheckForm42F_FieldIDScatteredIsPerPair(t *testing.T) {
	want, got := radixPair(t, 1, 1, 3, 1)
	c := CheckForm42F(want, got)

	if notes := c.NoteFor(RuleFieldID); len(notes) != 0 {
		t.Fatalf("差值不恒定时**不该**有基数聚合说明（那会把真信号吞掉），得 %d 条：%+v", len(notes), notes)
	}
	g := c.Group(RuleFieldID)
	if len(g) != 4 {
		t.Fatalf("差值散开时该逐条报 4 条，得 %d 条：\n%s", len(g), c)
	}
	var gotCols []form42f.ColName
	for _, f := range g {
		if f.Undecided {
			t.Errorf("逐条报的编号差异是确定差异，不该是未判定：%+v", f)
		}
		if !strings.Contains(f.F42, "fieldId=") || !strings.Contains(f.Tzs, "fieldId=") {
			t.Errorf("每条都要写清两侧编号：%+v", f)
		}
		gotCols = append(gotCols, f.ColName)
	}
	sort.Slice(gotCols, func(i, j int) bool { return gotCols[i] < gotCols[j] })
	wantCols := []form42f.ColName{"prmd001", "prmd002", "prmd003", "prmd004"}
	if fmt.Sprint(gotCols) != fmt.Sprint(wantCols) {
		t.Errorf("逐条报的 colName = %v，want %v", gotCols, wantCols)
	}
}

// TestCheckForm42F_FieldIDSameRadixNoNote：两侧完全同编号 ⇒ 0 条、**且没有聚合说明**。
//
// 为什么单独立一条：基数检测最便宜的写法是"只要差值恒定就报一句"，那样"恒差 0"也会冒出一条
// 说明 —— 一份完全一致的表单就该什么都不说（`TestCheckForm42F_Aligned` 的同一口径）。
func TestCheckForm42F_FieldIDSameRadixNoNote(t *testing.T) {
	want, got := radixPair(t, 0, 0, 0, 0)
	c := CheckForm42F(want, got)

	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("两侧同编号时不该有差异，得 %d 条：\n%v", len(diffs), diffs)
	}
	if len(c.Notes) != 0 {
		t.Errorf("两侧同编号时不该有聚合说明（恒差 0 是「没有问题」，不是「基数不同」）：%+v", c.Notes)
	}
	if !c.Consistent() {
		t.Errorf("两侧同编号时 Consistent() 该为 true")
	}
}

// TestCheckForm42F_FieldIDSinglePairIsStillADifference：**只有 1 对**时不谈"基数"。
//
// 为什么单独立一条：一对时"差值恒定"是**恒真**的，若照此判"两侧基数不同"，就会把唯一的信号
// 吞成一条说明（cli 层 `TestCmdTzsFormCheck_DiffExitsSix` 正是这个形状：单字段、编号 18↔19、
// 必须退 6）。这条钉住 `minRadixPairs` 的下界。
func TestCheckForm42F_FieldIDSinglePairIsStillADifference(t *testing.T) {
	want, got := radixPair(t, 1)
	c := CheckForm42F(want, got)

	if len(c.Notes) != 0 {
		t.Errorf("只有 1 对时不该谈「基数差」（恒定恒真）：%+v", c.Notes)
	}
	if g := c.Group(RuleFieldID); len(g) != 1 {
		t.Fatalf("只有 1 对且编号不同 ⇒ 该报 1 条确定差异，得 %d 条：\n%s", len(g), c)
	}
	if c.Consistent() {
		t.Errorf("有确定差异时 Consistent() 该为 false（⇒ CLI 退 6）")
	}
}

// ── 11–12. 可写性：同属性对齐（契约 A38）───────────────────────────────────

// TestCheckForm42F_WritabilitySameAttributeIsCertain：**同一属性**两侧反向 ⇒ 确定差异。
//
// 夹具：pmdl002 的 `required` 在 tzs 侧写成 true（`.42f` 是 0），两侧 `notNull` **都是 false**。
// 没有任何已知蕴含关系能解释它（不是 NOT NULL 归一那条）⇒ 必须留在确定差异里。
func TestCheckForm42F_WritabilitySameAttributeIsCertain(t *testing.T) {
	tzsText := replace1(t, checkTzsText,
		`colName="pmdl002" fieldId="23" widget="Edit" noEntry="false" required="false"`,
		`colName="pmdl002" fieldId="23" widget="Edit" noEntry="false" required="true"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	diffs := c.Diffs()
	if len(diffs) != 1 {
		t.Fatalf("同属性（required）反向该有且只有 1 条确定差异，得 %d 条：\n%s", len(diffs), c)
	}
	d := diffs[0]
	if d.Rule != RuleWritable {
		t.Errorf("该归到 %q 组，得 %q", RuleWritable, d.Rule)
	}
	if d.ColName != "pmdl002" {
		t.Errorf("colName = %q，want pmdl002", d.ColName)
	}
	if d.Undecided {
		t.Errorf("同一属性反向 + 两侧 notNull 都为 false ⇒ 是**确定**差异，不该是未判定：%+v", d)
	}
	if !strings.Contains(d.F42, "required=false") || !strings.Contains(d.Tzs, "required=true") {
		t.Errorf("两侧值要写清 required：F42=%q Tzs=%q", d.F42, d.Tzs)
	}
	if und := c.Undecided(); len(und) != 0 {
		t.Errorf("这条夹具不该有其他未判定条目：%+v", und)
	}
}

// TestCheckForm42F_WritabilityCrossAttributeIsUndecided：跨属性组合 ⇒ **未判定、不计入差异**。
//
// 夹具就是真实配对里那 4 条的形状：两侧 `notNull=true` 一致，只有 `required` 反向
// （tzs 侧 `required=false`）⇒ 出现 `notNull=true 且 required=false` 这个组合。
// A38：编译器可能把 NOT NULL 归一成必填，所以这不是设计变更的证据，只记"疑似"。
//
// 反证（报告里有命令）：把这一支改回"算确定差异"⇒ 本条的 ①②红。
func TestCheckForm42F_WritabilityCrossAttributeIsUndecided(t *testing.T) {
	tzsText := replace1(t, checkTzsText, `required="true" notNull="true"`, `required="false" notNull="true"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, tzsText))

	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("跨属性组合（notNull=true / required=false）**不得**算确定差异，得 %d 条（A38）：\n%v",
			len(diffs), diffs)
	}
	if !c.Consistent() {
		t.Errorf("只有这一条未判定时 Consistent() 该为 true（⇒ CLI 退 0）：\n%s", c)
	}
	var got []Finding
	for _, u := range c.Undecided() {
		if u.Rule == RuleWritable && u.ColName == "pmdl001" {
			got = append(got, u)
		}
	}
	if len(got) != 1 {
		t.Fatalf("pmdl001 该有 1 条「可写性」未判定，得 %d 条：\n%s", len(got), c)
	}
	u := got[0]
	if !u.Undecided {
		t.Errorf("这条必须是未判定：%+v", u)
	}
	// 两侧值都要写出来，而且要能看出是 required / notNull 的组合。
	for _, kw := range []string{"required", "notNull"} {
		if !strings.Contains(u.F42, kw) || !strings.Contains(u.Tzs, kw) {
			t.Errorf("两侧值都该带 %q：F42=%q Tzs=%q", kw, u.F42, u.Tzs)
		}
	}
	for _, kw := range []string{"NOT NULL", "归一", "疑似", "required"} {
		if !strings.Contains(u.Why, kw) {
			t.Errorf("说明里该出现 %q（A38 要求说清「疑似编译器把 NOT NULL 归一成必填」）：%s", kw, u.Why)
		}
	}
	out := c.String()
	if !strings.Contains(out, "【未判定】1 条") || !strings.Contains(out, "【可写性】0 条") {
		t.Errorf("渲染里该是「可写性 0 条 / 未判定 1 条」：\n%s", out)
	}
}

// ── 测试用的小装置 ────────────────────────────────────────────────────────

// projectF42ToTzs 把 `.42f` 的字段表**投影**成 tzs 侧形状（只给门控测试用）。
//
// 它刻意不做任何换算：同名字段、同一编号、同一批布尔与控件名原样搬过去。
// 于是 CheckForm42F 的差异组必然全空 —— 剩下能真正量到的是"配对有没有漏"与
// "未判定是不是只来自没带控件类型的条目"，这两条都是有牙齿的。
func projectF42ToTzs(f *form42f.F42Form) *TzsForm {
	out := &TzsForm{Prog: string(f.Name), DeclaredName: f.DeclaredName}
	for _, fld := range f.Fields {
		out.Fields = append(out.Fields, TzsField{
			Tag:      fld.Tag,
			Name:     fld.Name,
			ColName:  fld.ColName,
			FieldID:  fld.FieldID,
			Widget:   fld.Widget,
			NoEntry:  boolText42F(fld.NoEntry),
			Required: boolText42F(fld.Required),
			NotNull:  boolText42F(fld.NotNull),
		})
	}
	return out
}

// boolText42F 给投影用：`.42f` 的布尔被解析成 bool，投影回 `.4fd` 的 "true"/"false" 写法。
func boolText42F(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// tzsCheckRepoRoot 从包目录上溯找仓库根（夹在 `.agents` 与 `go.mod` 都有的那一层）。
func tzsCheckRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("拿不到工作目录：%v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("从 %s 上溯没找到 go.mod（仓库根）", mustGetwd(t))
		}
		dir = parent
	}
}

// tzsCheckSampleDir 真样本目录：`TT_FORM_SAMPLE_DIR` 优先（与 internal/form42f 的 sampleDir42F
// 同一口径，契约 C7.3），缺省指向成果目录里的 fixtures/forms/。
func tzsCheckSampleDir(t *testing.T) string {
	t.Helper()
	if d := os.Getenv("TT_FORM_SAMPLE_DIR"); d != "" {
		return d
	}
	return filepath.Join(tzsCheckRepoRoot(t), ".agents", "Genero Original Factory Compatible", "fixtures", "forms")
}