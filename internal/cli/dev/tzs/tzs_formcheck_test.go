package tzs

// tzs_formcheck_test.go —— `tt dev tzs formcheck` 命令层的守护测试。
//
// 命令层只该做三件事：读文件、排版、退出码。判定口径在 internal/dev/tzs（那边有自己的测试），
// 所以这里**不重复**验四组规则，只钉：
//
//	① 一致 → 退 0；
//	② 有确定差异 → 退 **6**（本命令组码表里没有 3，6 借自契约 A13 的「体检不过」）；
//	③ 只有"未判定" → 退 **0**（A15：未判定不计入错误）；
//	④ 参数/读取错 → 退 2，且 --json 下 stdout 恰好一个 {"ok":false,...}；
//	⑤ --json 的键名是契约（改了要同步 internal/dev/tzs/README.md 与调用方）。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/testkit"
)

// 与 internal/dev/tzs 的合成夹具同形（两份各自内联：命令层的测试不该依赖另一个包的内联常量）。
const (
	formcheck42FText = `<?xml version='1.0' encoding='UTF-8'?>
<Form name="apcs001_wf.tmp" build="3.21.03">
  <Page>
    <FormField name="formonly.pmdl001" colName="pmdl001" fieldId="18" notNull="1" required="1" noEntry="0">
      <Edit width="8"/>
    </FormField>
  </Page>
</Form>
`
	formcheck4FDText = `<?xml version='1.0' encoding='UTF-8'?>
<ManagedForm databaseName="ds">
  <Form name="apcs001_wf">
    <Edit name="formonly.pmdl001" colName="pmdl001" fieldId="18" widget="Edit" noEntry="false" required="true" notNull="true" />
  </Form>
</ManagedForm>
`
	// formcheck4FDDiffText 把编号改成 19（`.42f` 是 18）⇒ 一条"编号"差异。
	formcheck4FDDiffText = `<?xml version='1.0' encoding='UTF-8'?>
<ManagedForm databaseName="ds">
  <Form name="apcs001_wf">
    <Edit name="formonly.pmdl001" colName="pmdl001" fieldId="19" widget="Edit" noEntry="false" required="true" notNull="true" />
  </Form>
</ManagedForm>
`
	// formcheck4FDUndecidedText 的控件名不在已知表 ⇒ 只有"未判定"，退 0。
	formcheck4FDUndecidedText = `<?xml version='1.0' encoding='UTF-8'?>
<ManagedForm databaseName="ds">
  <Form name="apcs001_wf">
    <MysteryKnob name="formonly.pmdl001" colName="pmdl001" fieldId="18" widget="MysteryKnob" noEntry="false" required="true" notNull="true" />
  </Form>
</ManagedForm>
`
)

// writeFormcheckFixtures 把三份文本落到临时目录，返回 (.42f, 一致, 编号不同, 未判定) 四个路径。
func writeFormcheckFixtures(t *testing.T) (f42, same, diff, undecided string) {
	t.Helper()
	dir := t.TempDir()
	write := func(name, text string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatalf("写 %s 失败：%v", p, err)
		}
		return p
	}
	return write("apcs001_wf.42f", formcheck42FText),
		write("apcs001_wf.4fd", formcheck4FDText),
		write("diff.4fd", formcheck4FDDiffText),
		write("undecided.4fd", formcheck4FDUndecidedText)
}

// TestCmdTzsFormCheck_ConsistentExitsZero：一致退 0，且四组都打出来（不是一句百分比）。
func TestCmdTzsFormCheck_ConsistentExitsZero(t *testing.T) {
	f42, same, _, _ := writeFormcheckFixtures(t)
	code, out := testkit.CaptureStdout(t, func() int {
		return cmdTzs([]string{"formcheck", "--form42f", f42, "--tzs", same}, "")
	})
	if code != 0 {
		t.Fatalf("一致该退 0，得 %d\n%s", code, out)
	}
	for _, want := range []string{"【存在性】0 条", "【编号】0 条", "【可写性】0 条", "【控件族】0 条", "【未判定】0 条"} {
		if !strings.Contains(out, want) {
			t.Errorf("输出里缺 %q：\n%s", want, out)
		}
	}
}

// TestCmdTzsFormCheck_DiffExitsSix：有确定差异退 6（而不是 1/2/4/5）。
func TestCmdTzsFormCheck_DiffExitsSix(t *testing.T) {
	f42, _, diff, _ := writeFormcheckFixtures(t)
	code, out := testkit.CaptureStdout(t, func() int {
		return cmdTzs([]string{"formcheck", "--form42f", f42, "--tzs", diff}, "")
	})
	if code != 6 {
		t.Fatalf("有差异该退 6，得 %d\n%s", code, out)
	}
	if !strings.Contains(out, "【编号】1 条") || !strings.Contains(out, "pmdl001") {
		t.Errorf("输出里该有【编号】组与 colName：\n%s", out)
	}
}

// TestCmdTzsFormCheck_UndecidedExitsZero：只有未判定 ⇒ 退 0（A15：未判定不计入错误）。
func TestCmdTzsFormCheck_UndecidedExitsZero(t *testing.T) {
	f42, _, _, undecided := writeFormcheckFixtures(t)
	code, out := testkit.CaptureStdout(t, func() int {
		return cmdTzs([]string{"formcheck", "--form42f", f42, "--tzs", undecided}, "")
	})
	if code != 0 {
		t.Fatalf("只有未判定该退 0，得 %d\n%s", code, out)
	}
	if !strings.Contains(out, "【未判定】1 条") {
		t.Errorf("未判定要打印出来（不判也要看得见）：\n%s", out)
	}
}

// TestCmdTzsFormCheck_JSONKeys：`--json` 的键名是契约。
func TestCmdTzsFormCheck_JSONKeys(t *testing.T) {
	f42, _, diff, _ := writeFormcheckFixtures(t)
	code, out := testkit.CaptureStdout(t, func() int {
		return cmdTzs([]string{"formcheck", "--form42f", f42, "--tzs", diff, "--json"}, "")
	})
	if code != 6 {
		t.Fatalf("--json 下退出码不变（该退 6），得 %d\n%s", code, out)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json 输出不是合法 JSON：%v\n%s", err, out)
	}
	for _, k := range []string{"ok", "form", "declared_name_42f", "prog_tzs", "f42_fields", "tzs_fields",
		"tzs_unnamed", "pairs", "diff_count", "undecided_count", "groups", "undecided", "f42_only"} {
		if _, has := got[k]; !has {
			t.Errorf("JSON 少了键 %q：\n%s", k, out)
		}
	}
	if ok, _ := got["ok"].(bool); ok {
		t.Errorf("有差异时 ok 该是 false：\n%s", out)
	}
	groups, _ := got["groups"].(map[string]any)
	for _, k := range []string{"exist", "fieldid", "writable", "widget"} {
		if _, has := groups[k]; !has {
			t.Errorf("groups 少了分组 %q：\n%s", k, out)
		}
	}
}

// TestCmdTzsFormCheck_UsageErrors：缺参数/位置参数/坏输入一律退 2。
func TestCmdTzsFormCheck_UsageErrors(t *testing.T) {
	f42, same, _, _ := writeFormcheckFixtures(t)
	dir := t.TempDir()
	junk := filepath.Join(dir, "junk.4fd")
	if err := os.WriteFile(junk, []byte("这不是 XML"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := [][]string{
		{"formcheck"},                   // 两个输入都没给
		{"formcheck", "--form42f", f42}, // 缺 --tzs
		{"formcheck", "--form42f", f42, "--tzs", same, "x"}, // 位置参数一律报错
		{"formcheck", "--form42f", filepath.Join(dir, "nope.42f"), "--tzs", same},
		{"formcheck", "--form42f", f42, "--tzs", junk},
		{"formcheck", "--form42f", f42, "--tzs", filepath.Join(dir, "x.txt")},
	}
	for _, args := range cases {
		if code := testkit.Silent(t, func() int { return cmdTzs(args, "") }); code != 2 {
			t.Errorf("%v 该退 2，得 %d", args, code)
		}
	}
}
