package web

// 更新端点（/api/update*）的离线断言：GET 不联网、只读缓存与状态；POST check 在没有
// 出网条件时也要给一个 ok:false 而不是把服务端算成 500。
//
// 真联网那一半不在这里测：它由 make update-live-check 与 internal/update 的 httptest
// 覆盖（同一个判据只放一处）。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/update"
)

func TestUpdateGetReadsCacheOffline(t *testing.T) {
	s, cfgPath := newVersionedServer(t, `{"schemaVersion":3,"hosts":{"sshs":[]}}`, "0.2.1")
	dataDir := filepath.Dir(cfgPath)

	// 先放一份"上次检查"的缓存：它的当前版本必须等于服务自报的版本（不然会作废）。
	if err := update.SaveCheck(dataDir, &update.CheckResult{
		Current: "0.2.1", Latest: "9.9.9", Newer: true, CheckedAt: time.Now(),
		Source: update.RepoSlug,
	}); err != nil {
		t.Fatalf("SaveCheck: %v", err)
	}

	rec, body := doJSON(t, s, http.MethodGet, "/api/update", nil)
	if rec.Code != http.StatusOK || body["ok"] != true {
		t.Fatalf("GET /api/update = %d %v", rec.Code, body)
	}
	if body["version"] != "0.2.1" || body["source"] != update.RepoSlug {
		t.Errorf("该回报当前版本与更新源: %v", body)
	}
	check, _ := body["check"].(map[string]any)
	if check == nil || check["latest"] != "9.9.9" {
		t.Errorf("缓存里的检查结果该被读出来: %v", body["check"])
	}
	if hint, _ := body["hint"].(string); hint == "" {
		t.Error("有新版本时该给一句提示")
	}
	// 形态是服务自己的 exe 决定的（测试二进制不在便携标记载里 → 不让自装）。
	if can, _ := body["canInstall"].(bool); can {
		t.Error("测试二进制不在任何可自装的布局里，canInstall 该是 false")
	}
	if refusal, _ := body["refusal"].(string); refusal == "" {
		t.Error("不让自装时该给出原因")
	}
}

// newVersionedServer 与 newTestServer 一样，但指定服务自报的版本号。
// 版本号在这里很要紧：检查缓存的作废条件之一就是"记的版本 != 现在这个"。
// 两处都设：Version 是给页面看的（完整串），BareVersion 是拿来比较的（见 Options）。
func newVersionedServer(t *testing.T, seed, version string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	return New(Options{ConfigPath: path, Version: version, BareVersion: version}), path
}

func TestUpdateGetWithoutCache(t *testing.T) {
	s, _ := newTestServer(t, `{"schemaVersion":3,"hosts":{"sshs":[]}}`)
	rec, body := doJSON(t, s, http.MethodGet, "/api/update", nil)
	if rec.Code != http.StatusOK || body["ok"] != true {
		t.Fatalf("GET /api/update = %d %v", rec.Code, body)
	}
	if _, has := body["check"]; has {
		t.Error("没有缓存时不该凭空给一个检查结果")
	}
	if _, has := body["hint"]; has {
		t.Error("没有缓存时不该给提示")
	}
}

func TestUpdateApplyRefusesUnknownLayout(t *testing.T) {
	s, _ := newTestServer(t, `{"schemaVersion":3,"hosts":{"sshs":[]}}`)
	// 测试二进制的布局认不出来 → 必须拒绝交棒（而不是拉一个子进程去乱装）。
	rec, body := doJSON(t, s, http.MethodPost, "/api/update/apply", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("认不出的布局该返回 400，得到 %d %v", rec.Code, body)
	}
	if body["ok"] != false {
		t.Errorf("该是 ok:false: %v", body)
	}
}

// 技能漂移也要在状态里看得见（它就是"agent 读的手册跟二进制不同版"这件事）。
func TestUpdateGetReportsSkillsDrift(t *testing.T) {
	s, cfgPath := newTestServer(t, `{"schemaVersion":3,"hosts":{"sshs":[]}}`)
	dataDir := filepath.Dir(cfgPath)
	target := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.RecordSkillsTarget(dataDir, target, "0.0.1"); err != nil {
		t.Fatal(err)
	}
	rec, body := doJSON(t, s, http.MethodGet, "/api/update", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/update = %d", rec.Code)
	}
	drift, _ := body["skillsDrift"].(string)
	if drift == "" || !strings.Contains(drift, "0.0.1") {
		t.Errorf("该报告技能漂移并指明是哪个版本装的: %q", drift)
	}
	// 响应必须可 JSON 序列化（上面已经解过一遍，这里再钉一次类型没漏）。
	if _, err := json.Marshal(body); err != nil {
		t.Errorf("响应不可序列化: %v", err)
	}
}

// 这个端点必须**不联网**：拿一个不可解析的代理地址当配置，GET 仍然要成功
// （它只读本地缓存；联网只发生在 POST check 上）。
func TestUpdateGetNeverTouchesNetwork(t *testing.T) {
	s, _ := newTestServer(t, `{"schemaVersion":3,"hosts":{"sshs":[]},"net":{"proxy":"http://127.0.0.1:1"}}`)
	req := httptest.NewRequest(http.MethodGet, "/api/update", nil)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.Handler().ServeHTTP(rec, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("GET /api/update 卡住了：它不该联网")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/update = %d", rec.Code)
	}
}

// 回归：更新接口拿的必须是**裸版本**（`0.2.2`），不是给人看的完整串
// （`0.2.2 (commit 25a639d, 2026-10-07)`）。
//
// 真机上踩过：serve 把 versionString() 传给了 web.Options.Version，于是设置页的
// 「检查更新」永远失败，而错误文案是"版本号 … 不是 MAJOR.MINOR.PATCH" —— 看起来像发布方
// 写错了版本号，实际是这一层把展示串当成了可比的值。
func TestUpdateUsesBareVersionNotDisplayString(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":3,"hosts":{"sshs":[]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{
		ConfigPath:  path,
		Version:     "0.2.2 (commit 25a639d, 2026-10-07)",
		BareVersion: "0.2.2",
	})

	// 缓存是以裸版本为键写的（命令行那条路就是这么写的）——页面必须读得到它。
	if err := update.SaveCheck(dir, &update.CheckResult{
		Current: "0.2.2", Latest: "0.2.3", Newer: true, CheckedAt: time.Now(), Source: update.RepoSlug,
	}); err != nil {
		t.Fatal(err)
	}
	rec, body := doJSON(t, s, http.MethodGet, "/api/update", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/update = %d", rec.Code)
	}
	if body["version"] != "0.2.2 (commit 25a639d, 2026-10-07)" {
		t.Errorf("页面上的当前版本该是完整串（带 commit），得到 %v", body["version"])
	}
	if _, ok := body["check"].(map[string]any); !ok {
		t.Errorf("该用裸版本读得到缓存：%v", body["check"])
	}
	if hint, _ := body["hint"].(string); hint == "" {
		t.Error("该给出有新版本的提示")
	}

	// 技能漂移同理：戳里记的是裸版本。
	target := filepath.Join(t.TempDir(), "skills")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.RecordSkillsTarget(dir, target, "0.2.1"); err != nil {
		t.Fatal(err)
	}
	_, body = doJSON(t, s, http.MethodGet, "/api/update", nil)
	if drift, _ := body["skillsDrift"].(string); drift == "" {
		t.Errorf("技能漂移该按裸版本比对出来：%v", body["skillsDrift"])
	}
}
