package debug

import (
	"errors"
	"strings"
	"testing"
)

// TestParseInfoLine 钉住 `info line` 的三种回答（地址 0x0 是 fgldb 的常态，别当异常）：
//   - 命中：`Line 860 of "azz_p_zta.4gl" starts at ...`
//   - 没这个符号：`Function "zta_menu" not defined.` —— 这条必须是**结论**（SymbolNotDefinedError），
//     不能掉进"无法解析"；两者对使用者是两件完全不同的事。
//   - 其它：真异常，保留原始行。
func TestParseInfoLine(t *testing.T) {
	t.Run("hit", func(t *testing.T) {
		lines := []string{
			`Line 860 of "azz_p_zta.4gl" starts at address 0x0 <main+0>`,
			`   and ends at 0x0 <main+1>.`,
		}
		file, line, err := parseInfoLine(lines)
		if err != nil {
			t.Fatalf("parseInfoLine 报错: %v", err)
		}
		if file != "azz_p_zta.4gl" || line != 860 {
			t.Fatalf("得到 %q:%d，期望 azz_p_zta.4gl:860", file, line)
		}
	})

	t.Run("symbol-not-defined", func(t *testing.T) {
		lines := []string{
			`info line zta_menu`,
			`Function "zta_menu" not defined.`,
			`(fgldb)`,
		}
		_, _, err := parseInfoLine(lines)
		var miss *SymbolNotDefinedError
		if !errors.As(err, &miss) {
			t.Fatalf("期望 SymbolNotDefinedError，得到 %T: %v", err, err)
		}
		if miss.Word != "zta_menu" {
			t.Fatalf("Word=%q，期望 zta_menu", miss.Word)
		}
		// 提示里要带"带程序前缀"这件事：这是 GP 实测最常见的写错方式。
		if !strings.Contains(miss.Error(), "前缀") {
			t.Fatalf("错误文案没提前缀: %s", miss.Error())
		}
	})

	t.Run("unknown", func(t *testing.T) {
		_, _, err := parseInfoLine([]string{"(fgldb)"})
		if err == nil {
			t.Fatal("空输出不该被当成成功")
		}
		var miss *SymbolNotDefinedError
		if errors.As(err, &miss) {
			t.Fatalf("未知输出不该被当成符号不存在: %v", err)
		}
		if !strings.Contains(err.Error(), "无法解析 info line 输出") {
			t.Fatalf("未知输出要走「解析失败」: %v", err)
		}
	})
}
