package host

import (
	"strings"
	"testing"
)

// 命令行超时的错误信息里**绝不能带口令** —— 它会被打给终端、服务日志,以及 AI 的上下文。
//
// 这里直接拿 SqlplusCmd / KbCmd 的真实产出当输入,而不是手写样例:手写的会跟实现漂移,
// 而这两个构造器正是口令出现在命令行里的唯一两条路。
func TestRedactCmd(t *testing.T) {
	ora, err := SqlplusCmd("36", "", "/u2/oracle/product/19.0.0/dbhome_1/bin/sqlplus",
		"dsdemo/Secret123@//10.0.0.5:1521/t35prd", 0)
	if err != nil {
		t.Fatalf("SqlplusCmd: %v", err)
	}
	kb, err := KbCmd("/u2/kingbase/bin/ksql", "10.0.0.5", "54321", "topprd", "dsdemo/Secret456")
	if err != nil {
		t.Fatalf("KbCmd: %v", err)
	}

	for _, c := range []struct{ name, built, secret, keep string }{
		{"oracle", ora, "Secret123", "@//10.0.0.5:1521/t35prd"},
		{"kingbase", kb, "Secret456", "topprd"},
	} {
		if !strings.Contains(c.built, c.secret) {
			t.Fatalf("[%s] 构造出的命令里没有口令,这个测试什么也没验: %s", c.name, c.built)
		}
		got := redactCmd(c.built)
		if strings.Contains(got, c.secret) {
			t.Errorf("[%s] 口令没被抹掉: %s", c.name, got)
		}
		if !strings.Contains(got, c.keep) {
			t.Errorf("[%s] 排障要看的 %q 不该被抹掉: %s", c.name, c.keep, got)
		}
		// 账号留着(它决定"连的是哪个 schema",超时排障常要看)
		if !strings.Contains(got, "dsdemo") {
			t.Errorf("[%s] 账号不该被抹掉: %s", c.name, got)
		}
	}
}
