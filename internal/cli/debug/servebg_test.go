package debug

// 这组测试盯的是"后台常驻"赖以工作的两个判断：探活认不认得出在跑的实例、
// 寻址拼不拼得对挂载前缀。两个都是错了不报错、只表现为"找不到服务"的那种。

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// serveUp 必须同时认两种布局：独立调试服务(API 在根)与统一服务(调试面在 /debug/api/)。
// 漏认统一服务，单实例判断就失效（会再拉一个起来抢端口），--stop 也会说找不到它。
func TestServeUp_BothLayouts(t *testing.T) {
	standalone := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/status" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"server":"tdebug-debug"}`))
	}))
	defer standalone.Close()

	unified := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/debug/api/status":
			_, _ = w.Write([]byte(`{"server":"tdebug-debug"}`))
		case "/api/health":
			_, _ = w.Write([]byte(`{"ok":true,"server":"tt-unified"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer unified.Close()

	// 恰好占着同一端口的别的服务：不能被认领成自己人
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"server":"something-else"}`))
	}))
	defer other.Close()

	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"独立调试服务", standalone.URL, true},
		{"统一服务", unified.URL, true},
		{"别的 HTTP 服务", other.URL, false},
		{"空地址", "", false},
		{"没人监听", "http://127.0.0.1:1", false},
	}
	for _, c := range cases {
		if got := serveUp(c.url); got != c.want {
			t.Errorf("serveUp(%q) [%s] = %v, 期望 %v", c.url, c.name, got, c.want)
		}
	}
}

// joinAPIBase 给服务地址补挂载前缀；但 URL 自己带了 API 路径时不能重复拼。
func TestJoinAPIBase(t *testing.T) {
	cases := []struct {
		url, prefix, want string
	}{
		{"http://h:28670", "", "http://h:28670"},
		{"http://h:28670", "/debug", "http://h:28670/debug"},
		{"http://h:28670/", "/debug", "http://h:28670/debug"},
		// 已经指到 API 上了（--url 的两种写法）→ 原样用
		{"http://h:28670/debug/api", "/debug", "http://h:28670/debug/api"},
		{"http://h:28670/api", "", "http://h:28670/api"},
		{"http://h:28670/api/", "/debug", "http://h:28670/api"},
	}
	for _, c := range cases {
		if got := joinAPIBase(c.url, c.prefix); got != c.want {
			t.Errorf("joinAPIBase(%q, %q) = %q, 期望 %q", c.url, c.prefix, got, c.want)
		}
	}
}

// 状态文件的 APIBase 要能原样读回来 —— 控制端寻址全靠它，
// 老状态文件没有这个字段时按 "" 处理(独立调试服务的根布局)。
func TestServeInfo_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	st := &ServeInfo{PID: 1234, URL: "http://127.0.0.1:28670", APIBase: APIBaseUnderDebug, Log: "x.log"}
	if err := writeServeInfo(st, dir); err != nil {
		t.Fatalf("writeServeInfo: %v", err)
	}
	got, err := readServeInfo(dir)
	if err != nil {
		t.Fatalf("readServeInfo: %v", err)
	}
	if got.PID != st.PID || got.URL != st.URL || got.APIBase != APIBaseUnderDebug {
		t.Errorf("读回不一致: %+v", got)
	}

	removeServeInfo(dir)
	if _, err := readServeInfo(dir); err == nil {
		t.Error("状态文件已删除，读回应当报错")
	}
}
