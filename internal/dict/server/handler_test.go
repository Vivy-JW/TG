package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Handler 的路径是**绝对**的(/api/…):合并后字典页那套 SPA 没了,本包的端点
// 直接挂在共享层 /api/ 下,不再经 StripPrefix。
func TestHandlerRoutes(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"activeEnv":"a","sshs":[{"name":"a","host":"1.2.3.4"}]}}`)
	s := New(p)

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/api/mirror", nil))
	if rec.Code != 200 {
		t.Fatalf("GET /api/mirror code=%d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"envs"`) {
		t.Fatalf("应返回镜像状态: %s", rec.Body.String())
	}
}

// 共享给 internal/web 的端点不在本包注册(同进程里两边都有会撞名):
// /api/dbprobe、/api/dbaccverify、/api/conntest、/api/status、/api/config、/api/install
// 应明确报"未知接口",而不是本包的 JSON 接口。
//
// 这里断言 404 JSON 而不是某个 HTML 页:API 路径落到 HTML 页会让调用方拿到
// 200 + HTML,解析失败时报的错与真实原因(接口不存在)完全对不上。
func TestSharedEndpointsNotServedHere(t *testing.T) {
	s := New(writeTempConfig(t, `{}`))
	h := s.Handler()

	for _, path := range []string{
		"/api/dbprobe", "/api/dbaccverify", "/api/conntest", "/api/status", "/api/config",
		"/api/install", // 应用级动作,在 internal/web
		"/api/hosts",   // 共享的环境清单
	} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("POST", path, nil))
		if strings.Contains(rec.Body.String(), `"configPath"`) {
			t.Errorf("%s 不该由本包提供: %s", path, rec.Body.String())
		}
		if rec.Code != 404 {
			t.Errorf("%s 应回 404, got %d %s", path, rec.Code, rec.Body.String())
		}
		if strings.Contains(rec.Body.String(), "<!doctype html>") {
			t.Errorf("%s 不该返回 HTML: %s", path, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "未知接口") {
			t.Errorf("%s 应给出可读的说明, got %s", path, rec.Body.String())
		}
	}
}

// 非 API 路径不由本包负责:合并后没有兜底页了(internal/web 会把 /debug/ 之外的
// 非 API 路径 404,页面由它统一提供)。本包只认 /api/ 下的路径。
func TestNonAPIPathNotServedHere(t *testing.T) {
	s := New(writeTempConfig(t, `{}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/whatever", nil))
	if rec.Code != 404 {
		t.Fatalf("非 API 路径应 404(页面由 internal/web 提供): code=%d body=%s", rec.Code, rec.Body.String())
	}
}
