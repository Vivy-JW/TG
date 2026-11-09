package file

import (
	"strings"
	"testing"
)

func TestArgPathStaysUnderTop(t *testing.T) {
	tr := &target{top: "/u1/t35prd"}
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
		{"/u1/t35prd2/x", "", false}, // 前缀必须落在段边界上
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
	tr := &target{top: "/u1/t35prd"}
	if _, err := tr.argPath("erp/a\x00b"); err == nil || !strings.Contains(err.Error(), "NUL") {
		t.Fatalf("含 NUL 的路径应拒绝，得到 %v", err)
	}
}

func TestDisplay(t *testing.T) {
	top := "/u1/t35prd"
	cases := map[string]string{
		top:              ".",
		top + "/erp/apm": "erp/apm",
		"/etc/passwd":    "/etc/passwd",
	}
	for in, want := range cases {
		if got := display(top, in); got != want {
			t.Errorf("display(%q) = %q，期望 %q", in, got, want)
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
