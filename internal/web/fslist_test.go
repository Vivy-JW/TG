package web

import (
	"net/http/httptest"
	"runtime"
	"testing"
)

// TestHostIsLocal 钉住 /api/fs/list 的 Host 校验。
//
// 只有 RemoteAddr 校验时，DNS rebinding 能绕过：域名解析到 127.0.0.1 后请求确实来自本机，
// 但 Host 是攻击者的域名，同源策略下攻击者读得到响应。本用例就是为了不让这条校验被删掉。
func TestHostIsLocal(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"127.0.0.1:8848", true},
		{"127.0.0.1", true},
		{"localhost:8848", true},
		{"LOCALHOST:8848", true},
		{"localhost.:8848", true},
		{"[::1]:8848", true},
		{"[::1]", true},
		{"evil.com", false},
		{"evil.com:8848", false},
		{"127.0.0.1.evil.com:8848", false},
		{"10.0.0.5:8848", false},
		{"", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/api/fs/list", nil)
		r.Host = c.host
		if got := hostIsLocal(r); got != c.want {
			t.Errorf("hostIsLocal(%q) = %v, want %v", c.host, got, c.want)
		}
	}
}

// TestNormalizeDrivePath 钉住四种盘符形态归一（"D:" 直接给 os.ReadDir 会读该盘的当前目录）。
func TestNormalizeDrivePath(t *testing.T) {
	if runtime.GOOS != "windows" {
		// 非 Windows 上不做盘符改写：原样返回（选择器只给 "/" 一个根）。
		if got := normalizeDrivePath("D:"); got != "D:" {
			t.Fatalf("非 Windows 不该改写: %q", got)
		}
		return
	}
	for _, in := range []string{"D", "D:", "D:.", `D:\`, "d:", `d:\`} {
		if got := normalizeDrivePath(in); got != `D:\` {
			t.Errorf("normalizeDrivePath(%q) = %q, want %q", in, got, `D:\`)
		}
	}
	if got := normalizeDrivePath(`D:\work\a`); got != `D:\work\a` {
		t.Errorf("已有子路径不该被改写: %q", got)
	}
}
