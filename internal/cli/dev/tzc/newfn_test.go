package tzc

import (
	"strings"
	"testing"
	"tt/internal/testkit"
)

// TestNewFnHeaderTemplate 逐字节钉住新函数模板：
// 顶部空行、两条 80 个 # 的框、固定字段行、Usage 填真实函数名、日期/作者留占位符。
func TestNewFnHeaderTemplate(t *testing.T) {
	got := newFnHeaderTemplate("adzi999_added")
	want := "\n" +
		"################################################################################\n" +
		"# Descriptions...: 描述说明\n" +
		"# Memo...........:\n" +
		"# Usage..........: CALL adzi999_added(传入参数)\n" +
		"#                  RETURNING 回传参数\n" +
		"# Input parameter: 传入参数变量1   传入参数变量说明1\n" +
		"#                : 传入参数变量2   传入参数变量说明2\n" +
		"# Return code....: 回传参数变量1   回传参数变量说明1\n" +
		"#                : 回传参数变量2   回传参数变量说明2\n" +
		"# Date & Author..: 日期 By 作者\n" +
		"# Modify.........:\n" +
		"################################################################################\n"
	if got != want {
		t.Fatalf("模板与设计器形态不一致：\n得：%q\n想：%q", got, want)
	}
	// ① 顶部空行（落进 .tap 后新函数不会与上一个函数挨着）
	if !strings.HasPrefix(got, "\n#") {
		t.Error("模板必须以空行开头")
	}
	// ② 两条框各 80 个 #
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	rules := 0
	for _, ln := range lines {
		if strings.HasPrefix(ln, "####") {
			rules++
			if len(ln) != 80 {
				t.Errorf("分隔线长度 %d，想要 80：%q", len(ln), ln)
			}
		}
	}
	if rules != 2 {
		t.Errorf("分隔线 %d 条，想要 2 条", rules)
	}
	// ③ 占位符保留（不自动填日期/作者 —— 红线 R7 不引入时钟）
	if !strings.Contains(got, "日期 By 作者") {
		t.Error("应保留「日期 By 作者」占位符")
	}
	// ④ 确定性：两次调用逐字节相同
	if again := newFnHeaderTemplate("adzi999_added"); again != got {
		t.Error("模板必须是确定性的（两次调用结果不同）")
	}
	// ⑤ 只填已知值：函数名进 Usage 行，其余示例名不得出现
	if strings.Contains(got, "s_aooi150_ins") {
		t.Error("不应把示例函数名固化进模板")
	}
}

// TestNewfnRejectsDescFlag 钉住「newfn 不再有 --desc」。
func TestNewfnRejectsDescFlag(t *testing.T) {
	if code := testkit.Silent(t, func() int {
		return cmdNewfn([]string{"--type", "FUNCTION", "--desc", "#+ 旧用法"})
	}); code != 2 {
		t.Errorf("newfn --desc 应作为未知标志被拒（退出码 2），实际 %d", code)
	}
}
