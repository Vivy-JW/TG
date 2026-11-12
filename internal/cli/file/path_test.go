package file

import (
	"strings"
	"testing"
)

// newTestTarget 造一个典型的 GP 目标：实例目录 /u1/toptest，下挂 tiptop 与 topcust。
func newTestTarget() *target {
	return &target{
		inst: "/u1/toptest",
		top:  "/u1/toptest/tiptop",
		cust: "/u1/toptest/topcust",
	}
}

func TestArgPathTwoTrees(t *testing.T) {
	tr := newTestTarget()
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		// 空参数 = 实例目录：一眼能看到 tiptop/ 与 topcust/
		{"", "/u1/toptest", true},
		// 老用法：不带树名前缀 → 仍按 $TOP 解释
		{"azz/4gl/p_zta.4gl", "/u1/toptest/tiptop/azz/4gl/p_zta.4gl", true},
		{"azz/../cbm", "/u1/toptest/tiptop/cbm", true},
		// 显式写树名 → 从实例目录起算
		{"tiptop/azz", "/u1/toptest/tiptop/azz", true},
		{"topcust/cbm/4gl/x.4gl", "/u1/toptest/topcust/cbm/4gl/x.4gl", true},
		{"TOPCUST/cbm", "/u1/toptest/topcust/cbm", true}, // 首段大小写不敏感，换成规范写法
		// 往上走一层仍在实例目录内：允许
		{"../topcust/cbm", "/u1/toptest/topcust/cbm", true},
		// 绝对路径落在实例目录内：允许
		{"/u1/toptest/topcust/cbm", "/u1/toptest/topcust/cbm", true},
		{"/u1/toptest/tiptop/azz", "/u1/toptest/tiptop/azz", true},
		// 越界
		{"../../etc/passwd", "", false},
		{"/etc/passwd", "", false},
		{"/u1/toptest2/x", "", false}, // 前缀必须落在段边界上
		{"/u1/x", "", false},
	}
	for _, c := range cases {
		got, err := tr.argPath(c.in)
		if c.ok && err != nil {
			t.Errorf("argPath(%q) 不该报错: %v", c.in, err)
			continue
		}
		if !c.ok {
			if err == nil {
				t.Errorf("argPath(%q) 应越界报错，实得 %q", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("argPath(%q) = %q，期望 %q", c.in, got, c.want)
		}
	}
}

// 没有 instance / custRoot 的老环境：根退回 $TOP，行为与以前一致。
func TestArgPathTopOnlyFallback(t *testing.T) {
	tr := &target{inst: "/u1/t35prd", top: "/u1/t35prd"}
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "/u1/t35prd", true},
		{"erp/apm", "/u1/t35prd/erp/apm", true},
		{"/u1/t35prd/erp/apm", "/u1/t35prd/erp/apm", true},
		{"erp/../com", "/u1/t35prd/com", true},
		{"../etc/passwd", "", false},
		{"erp/../../etc/passwd", "", false},
		{"/etc/passwd", "", false},
		{"/u1/t35prd2/x", "", false},
	}
	for _, c := range cases {
		got, err := tr.argPath(c.in)
		if c.ok && err != nil {
			t.Errorf("argPath(%q) 不该报错: %v", c.in, err)
			continue
		}
		if !c.ok {
			if err == nil {
				t.Errorf("argPath(%q) 应越界报错，实得 %q", c.in, got)
			}
			continue
		}
		if got != c.want {
			t.Errorf("argPath(%q) = %q，期望 %q", c.in, got, c.want)
		}
	}
}

func TestArgPathRejectsNUL(t *testing.T) {
	tr := newTestTarget()
	if _, err := tr.argPath("erp/a\x00b"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("含 NUL 的路径应拒绝，得到 %v", err)
	}
}

func TestShort(t *testing.T) {
	tr := newTestTarget()
	cases := map[string]string{
		tr.top:                       ".",
		tr.top + "/azz/4gl/x.4gl":    "azz/4gl/x.4gl",
		tr.inst:                      ".",
		tr.inst + "/topcust/cbm/4gl": "topcust/cbm/4gl",
		tr.inst + "/tiptop/azz":      "azz", // 在 $TOP 内就按 $TOP 显示
		"/etc/passwd":                "/etc/passwd",
	}
	for in, want := range cases {
		if got := tr.short(in); got != want {
			t.Errorf("short(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestTreeArg(t *testing.T) {
	tr := newTestTarget()
	yes := []string{"tiptop", "tiptop/azz", "Tiptop/azz", "topcust", "topcust/cbm/4gl"}
	no := []string{"", ".", "..", "../topcust", "azz", "tiptopx", "topcustx/a"}
	for _, s := range yes {
		if !tr.treeArg(s) {
			t.Errorf("treeArg(%q) 应为真", s)
		}
	}
	for _, s := range no {
		if tr.treeArg(s) {
			t.Errorf("treeArg(%q) 应为假", s)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{0: "0", 1023: "1023", 1024: "1.0K", 1 << 20: "1.0M", 3 << 30: "3.0G"}
	for in, want := range cases {
		if got := humanBytes(in); got != want {
			t.Errorf("humanBytes(%d) = %q，期望 %q", in, got, want)
		}
	}
}
