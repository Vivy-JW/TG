package tzs

// form42f_realpair_test.go —— **真实配对**门控：真设计器包（`.tzs`）↔ 它对应的真编译表单（`.42f`）。
//
// 为什么要有这一条：`form42f_check_test.go` 里那几条是**合成夹具**与**自投影**
// （`projectF42ToTzs` 把 `.42f` 原样搬成 tzs 侧形状），它们只能证明"代码按我写的判据跑"，
// 证明不了"判据碰上真实差异会不会误报"。真实配对才会暴露那些东西 —— 本文件第一次跑就暴露了
// "编号"这条规则在两个真实文件上**每一条唯一配对都命中**（见 README「已知局限」）。
//
// 三条纪律（与包 README 同一套）：
//
//  1. **不断言"整体零差异"**。真实配对本来就可能不同步：设计器改完没重编译、编译完没重新导出包。
//     所以这里断言的是**结构与不变量**（两侧都读得出来、每条清单都带身份与两侧值与分组、
//     未判定与确定的互补关系、退出码谓词、确定性），条数只**记录**不判定。
//     例外的两处（⑥⑦，A37/A38）断言的是**规则的效果**而不是"文件一致"：真实配对那 90 条差异
//     已被追认为规则误报（编号 86 = 两侧基数恒差 +1、可写性 4 = NOT NULL 归一），所以
//     "编号/可写性的确定差异必须是 0 + 事实以**聚合说明/未判定**的形式留下来"本身就是回归网：
//     规则悄悄变宽（误报回来）或变窄（把事实丢了）都会红。
//  2. **样本不在本机就跳过，并写明原因**（契约 C7.3）。跳过文案是一条字面量（跳过台账按
//     文件+文案登记，见 internal/testkit/skipledger_test.go），不许拿合成件顶替真样本。
//  3. **只读**。两个环境变量指向的东西一律只读：`.tzs` 走 `ReadTzsFormPackage`（内存解压，
//     只取包内那一个 `.4fd`），`.42f` 只 `os.ReadFile`。不写、不解压到磁盘、不动样本目录。
//
// 环境变量（两个都要有，缺一个就跳过）：
//
//	TT_TZS_REAL_DIR  设计器导出目录，里面有 <表单名>*.tzs（实然侧；如 D:\t100_wrok_dir\109\35）
//	TT_FORM_42F      编译后的表单文件（应然侧；如 %TEMP%\...\aimm200_wf.42f）
//
// 一条示例（本机实测用的就是这两个）：
//
//	TT_TZS_REAL_DIR='D:\t100_wrok_dir\109\35' TT_FORM_42F="$env:TEMP\ttfc\aimm200_wf.42f" `
//	  go test ./internal/dev/tzs -run TestForm42FRealPair -count=1 -v

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"tt/internal/form42f"
)

const (
	// envTzsRealDir 指设计器导出目录（`*.tzs` 就在里面）。
	envTzsRealDir = "TT_TZS_REAL_DIR"
	// envForm42F 指编译后的表单文件（应然侧）。
	envForm42F = "TT_FORM_42F"
)

// reIdentitySuffixRealPair 与 cli 层 `common.reIdentitySuffix` 同一形状（设计器的客制/标准
// 身份后缀，如 `aimm200_wf(s).tzs` 的 `(s)`）。
//
// 本包**不 import `internal/cli/dev`**（那边 import 本包，反过来就成环 —— 见 README 第 18 行），
// 所以这三行自带一份；它只影响"从文件名推表单名"，不参与任何判定。
var reIdentitySuffixRealPair = regexp.MustCompile(`\([A-Za-z]\)$`)

// TestForm42FRealPair 是真实配对的门控测试。
//
// 它断言的是**不变量**，不是"好看的数字"：
//
//	① 两侧都真的读出了字段（`F42Fields>0`、`TzsFields>0`、`Pairs>0`）—— 防"空对空"通过；
//	② `Findings` 每一条都带分组（`Rule` 必须是 `RuleOrder` 里的一条）+ `colName` + 两侧值 + 为什么；
//	③ `Diffs()` 与 `Undecided()` 严格互补：两者之和 = `Findings` 总数，各组之和 = `Diffs()` 总数；
//	④ 退出码谓词（CLI 的 6/0 就是从它取的）：确定差异>0 ⇔ `Consistent()==false`；
//	⑤ 确定性：同一份输入跑两次，渲染出来的清单**逐字节相同**（"同一份差异每次都长一样"）。
func TestForm42FRealPair(t *testing.T) {
	f42Path, tzsPath, reason := locateRealPair()
	if reason != "" {
		t.Skipf("没有现场真配对样本：%s —— 设 TT_TZS_REAL_DIR 与 TT_FORM_42F 指向真 .tzs 目录与真 .42f（契约 C7.3：缺样本写明原因，不拿合成件顶替）", reason)
	}

	b, err := os.ReadFile(f42Path)
	if err != nil {
		t.Fatalf("读 %s 失败：%v", f42Path, err)
	}
	formName := formNameFrom42F(f42Path)
	want, err := form42f.ParseForm42F(formName, string(b))
	if err != nil {
		t.Fatalf("解析 %s 失败（样本在、但不是合法 .42f）：%v", f42Path, err)
	}
	got, err := ReadTzsFormPackage(tzsPath)
	if err != nil {
		t.Fatalf("读 %s 失败（样本在、但不是合法表单包）：%v", tzsPath, err)
	}

	c := CheckForm42F(want, got)

	// ① 两侧都真的读出了字段。
	if c.F42Fields <= 0 {
		t.Fatalf("应然侧 .42f 字段数 = %d，want > 0（%s 没解出任何条目）", c.F42Fields, f42Path)
	}
	if c.TzsFields <= 0 {
		t.Fatalf("实然侧 .4fd 字段数 = %d，want > 0（%s 没解出任何条目）", c.TzsFields, tzsPath)
	}
	if c.Pairs <= 0 {
		t.Fatalf("配对对数 = %d，want > 0（一条都没配上，后面的规则等于空转）", c.Pairs)
	}
	if c.Pairs > c.F42Fields || c.Pairs > c.TzsFields {
		t.Errorf("配对对数 %d 超过了某一侧的字段数（.42f=%d / tzs=%d）—— 配对逻辑出错了",
			c.Pairs, c.F42Fields, c.TzsFields)
	}

	// ② 每条清单都带身份、两侧值与分组。
	for i, f := range c.Findings {
		if !isKnownRule(f.Rule) {
			t.Errorf("第 %d 条的分组 %q 不在 RuleOrder（%v）里：%+v", i, f.Rule, RuleOrder, f)
		}
		if strings.TrimSpace(string(f.ColName)) == "" {
			t.Errorf("第 %d 条没有 colName（没有身份就没法定位到栏位）：%+v", i, f)
		}
		if strings.TrimSpace(f.F42) == "" {
			t.Errorf("第 %d 条（%s）没写应然侧的值：%+v", i, f.ColName, f)
		}
		if strings.TrimSpace(f.Tzs) == "" {
			t.Errorf("第 %d 条（%s）没写实然侧的值：%+v", i, f.ColName, f)
		}
		if strings.TrimSpace(f.Why) == "" {
			t.Errorf("第 %d 条（%s）没写为什么：%+v", i, f.ColName, f)
		}
	}

	// ③ 未判定与确定差异互补，且分组不漏条。
	if got, want := len(c.Diffs())+len(c.Undecided()), len(c.Findings); got != want {
		t.Errorf("Diffs+Undecided = %d，want %d（= Findings 总数）—— 有清单条目既不算差异也不算未判定",
			got, want)
	}
	sum := 0
	for _, r := range RuleOrder {
		sum += len(c.Group(r))
	}
	if sum != len(c.Diffs()) {
		t.Errorf("四组之和 = %d，want %d（= 确定差异总数）—— 有差异没有归到任何一组", sum, len(c.Diffs()))
	}

	// ④ 退出码谓词：CLI 的「有确定差异 ⇒ 6 / 只有未判定 ⇒ 0」就建在这条上。
	if got, want := c.Consistent(), len(c.Diffs()) == 0; got != want {
		t.Errorf("Consistent() = %t，want %t（= 确定差异为 0）—— 退出码 6/0 会跟着一起错",
			got, want)
	}

	// ⑤ 确定性：同一份输入两次跑出同一份渲染。
	if again := CheckForm42F(want, got); again.String() != c.String() {
		t.Errorf("同一份输入跑两次结果不同（差异清单的顺序或内容不稳定）：\n第一次：\n%s\n第二次：\n%s",
			c.String(), again.String())
	}

	// ⑥ A37：编号必须**基数感知** —— 这条真实配对的 86 条"编号不同"全部来自"两侧基数恒差 +1"，
	// 所以确定差异必须是 **0**，而事实以**一条聚合说明**留下来（不是被删掉）。
	// `TestCheckForm42F_FieldIDRadixShiftIsOneNote` 是合成夹具上的同一条判据；这里是真实文件上的。
	if n := len(c.Group(RuleFieldID)); n != 0 {
		t.Errorf("「编号」确定差异 = %d 条，want 0（A37：两侧 fieldId 恒差 +1 是基数问题，"+
			"逐条报就是对同一件事重复 86 遍、对「是不是另一张表单」零分辨力）：\n%v", n, c.Group(RuleFieldID))
	}
	radix := c.NoteFor(RuleFieldID)
	if len(radix) != 1 {
		t.Fatalf("基数聚合说明 = %d 条，want 1（把 86 条降级成一条说明，不是删掉）：%+v", len(radix), c.Notes)
	}
	for _, kw := range []string{"恒差", "+1", "基数"} {
		if !strings.Contains(radix[0].Text, kw) {
			t.Errorf("聚合说明里该写清 %q（措辞只陈述事实、不猜哪边是 0 基）：%s", kw, radix[0].Text)
		}
	}
	if radix[0].Pairs <= 0 || radix[0].Pairs > c.Pairs {
		t.Errorf("聚合说明的配对数 = %d，want 落在 (0, %d]（覆盖多少对必须可核）", radix[0].Pairs, c.Pairs)
	}
	if len(radix[0].Cols) == 0 {
		t.Errorf("聚合说明该带样例 colName（降级≠丢线索）：%+v", radix[0])
	}
	// 说明真的被渲染出来（"改到不报就完事"的反面：报告里必须看得见）。
	if out := c.String(); !strings.Contains(out, "【说明】1 条") || !strings.Contains(out, "恒差 +1") {
		t.Errorf("渲染里没有【说明】段或没写「恒差 +1」—— 事实被丢了：\n%s", out)
	}

	// ⑦ A38：可写性必须**同属性对齐** —— 那 4 条（`imaa036/037/038`、`imac002`）两侧 `notNull` 一致、
	// 只有 `required` 反向，属「疑似编译器把 NOT NULL 归一成必填」，只能是**未判定**。
	if n := len(c.Group(RuleWritable)); n != 0 {
		t.Errorf("「可写性」确定差异 = %d 条，want 0（A38：跨属性组合只记疑似，不计入确定差异）：\n%v",
			n, c.Group(RuleWritable))
	}
	reqUnd := 0
	for _, u := range c.Undecided() {
		if u.Rule != RuleWritable {
			continue
		}
		blob := u.F42 + u.Tzs + u.Why
		if !strings.Contains(blob, "required") {
			continue
		}
		reqUnd++
		for _, kw := range []string{"notNull", "归一", "疑似"} {
			if !strings.Contains(blob, kw) {
				t.Errorf("可写性未判定条目（%s）该说清 %q（疑似编译器把 NOT NULL 归一成必填）：%+v",
					u.ColName, kw, u)
			}
		}
	}
	if reqUnd < 4 {
		t.Errorf("带 required 的可写性未判定 = %d 条，want >= 4（真实配对里那 4 个 CheckBox 必须留在这里，"+
			"不是消失）：\n%v", reqUnd, c.Undecided())
	}

	// 条数只记录、不判定（真实配对本来就可能不同步）。
	t.Logf("真实配对 %s：.42f %d 条 / tzs %d 条 / 配对 %d 对；差异 %d 条（存在性 %d、编号 %d、可写性 %d、控件族 %d）；未判定 %d 条；聚合说明 %d 条；仅 .42f 有 %d 条；tzs 无 colName %d 条；Consistent=%t",
		formName, c.F42Fields, c.TzsFields, c.Pairs,
		len(c.Diffs()), len(c.Group(RuleExist)), len(c.Group(RuleFieldID)),
		len(c.Group(RuleWritable)), len(c.Group(RuleWidget)),
		len(c.Undecided()), len(c.Notes), len(c.F42Only), c.TzsUnnamed, c.Consistent())
	for _, n := range c.Notes {
		t.Logf("聚合说明[%s]：%s", n.Rule.Title(), n.Text)
	}
}

// TestForm42FUndecidedOnlyKeepsConsistent 是"未判定不计入差异"的**反证对象**。
//
// 把 `Diffs()` 的过滤条件去掉（未判定也返回）⇒ 本文件第一条断言就红、`Consistent()` 变 false
// ⇒ 退出码从 0 变 6。所以它是这条判据唯一的牙齿（真实配对里"未判定"总和确定差异同时存在，
// 分不开两件事，所以这里用合成夹具把"只有未判定"单独造出来）。
//
// 同一条测试里做**对照**：把同一份夹具改出一条确定差异（编号不一致），`Consistent()` 必须 false
// —— 否则"只有未判定 ⇒ 0"可能只是因为 `Consistent()` 永远为真。
func TestForm42FUndecidedOnlyKeepsConsistent(t *testing.T) {
	// 合成夹具（checkF42Text / checkTzsText，见 form42f_check_test.go）：两侧完全对齐。
	// 只把 tzs 侧的控件名换成一个不在 widgetCanonTable 里的名字 ⇒ 只可能是"未判定"。
	undecidedText := replace1(t, checkTzsText, `widget="CheckBox"`, `widget="MysteryWidgetForTest"`)
	c := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, undecidedText))

	if len(c.Findings) == 0 {
		t.Fatalf("夹具没造出任何清单条目（这条测试等于空转）：\n%s", c)
	}
	if len(c.Undecided()) == 0 {
		t.Fatalf("未知控件名该进未判定，得 0 条：\n%s", c)
	}
	if diffs := c.Diffs(); len(diffs) != 0 {
		t.Fatalf("未判定被算进了确定差异（%d 条）—— A15 的「不计入错误」破了：\n%v", len(diffs), diffs)
	}
	if !c.Consistent() {
		t.Errorf("只有未判定时 Consistent() 必须为 true（⇒ CLI 退 0），得 false：\n%s", c)
	}
	for _, u := range c.Undecided() {
		if u.ColName == "" || u.F42 == "" || u.Tzs == "" || u.Why == "" {
			t.Errorf("未判定条目也要带 colName + 两侧值 + 为什么：%+v", u)
		}
	}

	// 对照：同一份夹具里造一条**确定**差异（编号不一致）⇒ Consistent()=false（⇒ 退 6）。
	diffText := replace1(t, checkTzsText, `fieldId="23"`, `fieldId="24"`)
	c2 := CheckForm42F(mustF42(t, checkF42Text), mustTzs(t, diffText))
	if len(c2.Diffs()) == 0 {
		t.Fatalf("夹具没造出确定差异（对照失效，这条测试证明不了任何事）：\n%s", c2)
	}
	if c2.Consistent() {
		t.Errorf("有确定差异时 Consistent() 必须为 false（⇒ CLI 退 6），得 true：\n%s", c2)
	}
	if got := c2.Group(RuleFieldID); len(got) != 1 {
		t.Errorf("编号不一致该归「编号」组 1 条，得 %d 条：\n%v", len(got), got)
	}
}

// locateRealPair 解析两个环境变量并定位实然侧那个 `.tzs`。
//
// 返回非空 reason 表示"样本不在这台机器上"—— 调用方**跳过**（不是失败）。
// 环境变量都设了、东西也在，但实然侧有多个候选 ⇒ 那是配置歧义，按"不猜"直接失败。
func locateRealPair() (f42Path, tzsPath, reason string) {
	dir := strings.TrimSpace(os.Getenv(envTzsRealDir))
	f42Path = strings.TrimSpace(os.Getenv(envForm42F))

	if dir == "" && f42Path == "" {
		return "", "", envTzsRealDir + " 与 " + envForm42F + " 都没设"
	}
	if dir == "" {
		return "", "", envTzsRealDir + " 没设（只给了 " + envForm42F + "）"
	}
	if f42Path == "" {
		return "", "", envForm42F + " 没设（只给了 " + envTzsRealDir + "）"
	}
	if _, err := os.Stat(dir); err != nil {
		return "", "", fmt.Sprintf("%s=%s 打不开：%v", envTzsRealDir, dir, err)
	}
	if _, err := os.Stat(f42Path); err != nil {
		return "", "", fmt.Sprintf("%s=%s 打不开：%v", envForm42F, f42Path, err)
	}

	form := formNameFrom42F(f42Path)
	var hits []string
	for _, ext := range []string{".tzs", ".tzv"} {
		m, err := filepath.Glob(filepath.Join(dir, form+"*"+ext))
		if err != nil {
			return "", "", fmt.Sprintf("在 %s 里按 %s*%s 找包失败：%v", dir, form, ext, err)
		}
		hits = append(hits, m...)
	}
	switch len(hits) {
	case 0:
		return "", "", fmt.Sprintf("%s=%s 里没有 %s*.tzs（.42f 推出来的表单名是 %s）",
			envTzsRealDir, dir, form, form)
	case 1:
		return f42Path, hits[0], ""
	}
	// 多个候选：**不替你挑**（挑错了"实然"就整个换了，而报告看起来还一样）。
	return "", "", fmt.Sprintf("%s 里有 %d 个候选（%s）—— 只留一个，或把 %s 指向含唯一候选的目录",
		envTzsRealDir, len(hits), strings.Join(hits, " / "), envTzsRealDir)
}

// formNameFrom42F 从 `.42f` 文件名推表单名（去扩展名 + 去身份后缀，与 CLI 层同一手法）。
func formNameFrom42F(p string) string {
	base := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
	return reIdentitySuffixRealPair.ReplaceAllString(base, "")
}

// isKnownRule 判一个分组是不是 `RuleOrder` 里的那四条之一（防"新加了一条没人渲染的规则"）。
func isKnownRule(r Rule) bool {
	for _, known := range RuleOrder {
		if r == known {
			return true
		}
	}
	return false
}
