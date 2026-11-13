package web

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"tg/internal/config"
	"tg/internal/designer"
)

// newTestServer 造一个只带共享 API 的服务（不接子系统），指向临时配置。
func newTestServer(t *testing.T, seed string) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if seed != "" {
		if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return New(Options{ConfigPath: path, Version: "test"}), path
}

func doJSON(t *testing.T, s *Server, method, target string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, target, &buf)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	out := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec, out
}

const seededConfig = `{
  "schemaVersion": 2,
  "listen": "127.0.0.1:28670",
  "hosts": {
    "activeEnv": "开发环境",
    "sshs": [{
      "name": "开发环境", "host": "10.0.0.1", "port": 22, "user": "u", "password": "p",
      "db": {
        "type": "oracle", "host": "10.0.0.2", "port": 1521, "service": "t100dev",
        "viaSsh": {"host": "10.0.0.1", "port": 22, "user": "u", "password": "p", "bindPort": 11521},
        "accounts": [{"account": "ds", "password": "x"}]
      }
    }]
  },
  "query": {"source": "local"},
  "debug": {"termWidth": 240},
  "未来的键": {"keep": true}
}`

// 配置页保存不能把表单不管理的 viaSsh 抹掉 —— 否则用户配好的 SSH 隧道
// 会在"打开设置页点一下保存"后静默消失。
func TestHostsPut_PreservesViaSSH(t *testing.T) {
	s, path := newTestServer(t, seededConfig)

	// 模拟配置页提交：它有 type/host/port/service/accounts，但没有 viaSsh
	body := map[string]any{
		"hosts": map[string]any{
			"activeEnv": "开发环境",
			"sshs": []any{map[string]any{
				"name": "开发环境", "host": "10.0.0.1", "port": 22, "user": "u", "password": "p",
				"db": map[string]any{
					"type": "oracle", "host": "10.0.0.9", "port": 1521, "service": "t100dev",
					"accounts": []any{map[string]any{"account": "ds", "password": "x"}},
				},
			}},
		},
	}
	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}

	root, err := config.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	hosts, _ := root["hosts"].(map[string]any)
	sshs, _ := hosts["sshs"].([]any)
	if len(sshs) != 1 {
		t.Fatalf("环境数 = %d", len(sshs))
	}
	db, _ := sshs[0].(map[string]any)["db"].(map[string]any)

	// 表单改的字段要生效
	if db["host"] != "10.0.0.9" {
		t.Errorf("表单提交的 db.host 未生效: %v", db["host"])
	}
	// 表单不管的字段要保留
	via, _ := db["viaSsh"].(map[string]any)
	if via == nil {
		t.Fatal("viaSsh 被抹掉了 —— 用户配的 SSH 隧道静默丢失")
	}
	if via["bindPort"] != float64(11521) {
		t.Errorf("viaSsh 内容不完整: %v", via)
	}
}

// PUT /api/hosts 只该动自己带的节，其他顶层键（含未知键）原样保留。
func TestHostsPut_LeavesOtherSectionsAlone(t *testing.T) {
	s, path := newTestServer(t, seededConfig)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"debug": map[string]any{"termWidth": 300},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}

	root, _ := config.Open(path)
	if _, has := root["未来的键"]; !has {
		t.Error("未知顶层键被抹掉了")
	}
	q, _ := root["query"].(map[string]any)
	if q["source"] != "local" {
		t.Error("本次没提交的 query 节被改动了")
	}
	d, _ := root["debug"].(map[string]any)
	if d["termWidth"] != float64(300) {
		t.Errorf("debug.termWidth 未写入: %v", d["termWidth"])
	}
	l, _ := root["listen"].(string)
	if l != "127.0.0.1:28670" {
		t.Errorf("listen 被误改: %q", l)
	}
}

// **表单没提交的字段要保留旧值**：请求体是前端照它的表单序列化出来的，
// 表单里没有的字段（后加进 schema 的、只给命令行用的）会整个缺席。
//
// 这条钉的是真实踩到的两次：`debug.aiTakeover` 与 `query.limit` ——
// 两个都在 schema 里、都有代码读，而配置页里没有对应控件。整节替换就等于
// "打开设置页点一次保存，那个字段静默消失"（用户完全看不出发生过什么）。
func TestHostsPut_KeepsFieldsTheFormDoesNotSend(t *testing.T) {
	seed := `{"schemaVersion":3,
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","password":"p","zone":"36"}]},
	  "debug":{"termWidth":240,"aiTakeover":true},
	  "query":{"source":"local","limit":500}}`
	s, path := newTestServer(t, seed)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"debug": map[string]any{"termWidth": 180},
		// 保存时已校验 query.source 只能指向存在的环境（拦悬空引用），
		// 这里用环境清单里真有的「甲」，验证"提交的值生效 + 没提交的字段保留"。
		"query": map[string]any{"source": "甲"},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}
	root, _ := config.Open(path)
	d, _ := root["debug"].(map[string]any)
	if d["termWidth"] != float64(180) {
		t.Errorf("表单提交的 debug.termWidth 未生效: %v", d["termWidth"])
	}
	if d["aiTakeover"] != true {
		t.Errorf("表单没提交的 debug.aiTakeover 被抹掉了（保存一次就丢配置）: %v", d["aiTakeover"])
	}
	q, _ := root["query"].(map[string]any)
	if q["source"] != "甲" {
		t.Errorf("表单提交的 query.source 未生效: %v", q["source"])
	}
	if q["limit"] != float64(500) {
		t.Errorf("表单没提交的 query.limit 被抹掉了: %v", q["limit"])
	}
}

// 设置页改名必须是一次迁移：按旧名落盘的字典库跟着走，query.source 一起改写。
// 少了这一步，保存"成功"、查询却报「本地库不存在」——与数据不存在无法区分。
func TestHostsPut_RenameMigratesPaths(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","password":"p","zone":"1"}]},
	  "query":{"source":"local:甲"}}`
	s, path := newTestServer(t, seed)
	dir := filepath.Dir(path)
	oldDict := config.DictDBPath(dir, "甲")
	if err := os.MkdirAll(filepath.Dir(oldDict), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldDict, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "乙", "host": "h", "port": 22, "user": "u", "password": "p", "zone": "1",
		}}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := os.Stat(oldDict); !os.IsNotExist(err) {
		t.Errorf("旧名的本地字典库没搬走：%s", oldDict)
	}
	if _, err := os.Stat(config.DictDBPath(dir, "乙")); err != nil {
		t.Errorf("新名的本地字典库不存在：%v", err)
	}
	root, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if root.Query.Source != "local:乙" {
		t.Errorf("query.source = %q，改名后该跟着改成 local:乙", root.Query.Source)
	}
}

// **hosts 节里表单不管理的键也要保留**：`hosts.activeEnv`（"默认环境"）就是这一类 ——
// 配置页上早就没有这个控件了，而 `config.ActiveHost` 还在拿它兜底。
// 保存一次就把它抹掉，等于把用户手工配的默认环境悄悄清空。
func TestHostsPut_KeepsSectionKeysTheFormDoesNotManage(t *testing.T) {
	seed := `{"schemaVersion":3,"hosts":{"activeEnv":"甲","sshs":[
	  {"name":"甲","host":"10.0.0.1","port":22,"user":"u","password":"p","zone":"36"}]}}`
	s, path := newTestServer(t, seed)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{
			"sshs": []any{map[string]any{
				"name": "甲", "host": "10.0.0.2", "port": 22, "user": "u", "password": "p", "zone": "36",
			}},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}
	root, _ := config.Open(path)
	hosts, _ := root["hosts"].(map[string]any)
	if hosts["activeEnv"] != "甲" {
		t.Errorf("hosts.activeEnv 被抹掉了（表单里没有这一栏，可它有代码在读）: %v", hosts["activeEnv"])
	}
	sshs, _ := hosts["sshs"].([]any)
	if len(sshs) != 1 || sshs[0].(map[string]any)["host"] != "10.0.0.2" {
		t.Errorf("表单提交的改动该生效: %v", sshs)
	}
}

// 校验拦在保存之前，且错误信息是给人看的中文。
func TestHostsPut_Validation(t *testing.T) {
	cases := []struct {
		name   string
		sshs   []any
		active string
		want   string
	}{
		{"缺名称", []any{map[string]any{"host": "h", "user": "u"}}, "", "缺少名称"},
		{"重名", []any{
			map[string]any{"name": "a", "host": "h", "user": "u"},
			map[string]any{"name": "a", "host": "h2", "user": "u"},
		}, "", "重复"},
		{"缺主机", []any{map[string]any{"name": "a", "user": "u"}}, "", "缺少服务器地址"},
		{"端口越界", []any{map[string]any{"name": "a", "host": "h", "user": "u", "port": 70000}}, "", "超出范围"},
		{"缺账号", []any{map[string]any{"name": "a", "host": "h"}}, "", "缺少登录账号"},
		{"库类型非法", []any{map[string]any{
			"name": "a", "host": "h", "user": "u",
			"db": map[string]any{"type": "mysql", "host": "d"},
		}}, "", "不支持"},
		// 「默认环境不存在」这条用例已删:activeEnv 本身已移除,
		// 不再有"默认环境指向不存在的环境"这种状态可校验(见 config.Hosts 的注释)。
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := newTestServer(t, seededConfig)
			rec, out := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
				"hosts": map[string]any{"activeEnv": c.active, "sshs": c.sshs},
			})
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("应当被拒，却得到 HTTP %d: %s", rec.Code, rec.Body.String())
			}
			msg, _ := out["error"].(string)
			if !contains(msg, c.want) {
				t.Errorf("错误信息 %q 里没有 %q", msg, c.want)
			}
		})
	}
}

// 校验失败不能落盘 —— 一次失败的保存不该把配置改坏。
func TestHostsPut_RejectedWriteDoesNotTouchFile(t *testing.T) {
	s, path := newTestServer(t, seededConfig)
	before, _ := os.ReadFile(path)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"activeEnv": "", "sshs": []any{map[string]any{"name": "a"}}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("应当被拒, 得到 HTTP %d", rec.Code)
	}

	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Error("被拒绝的写入改动了配置文件")
	}
}

// GET /api/hosts 返回设置页需要的那几个节。
func TestHostsGet(t *testing.T) {
	s, _ := newTestServer(t, seededConfig)
	rec, out := doJSON(t, s, http.MethodGet, "/api/hosts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP %d", rec.Code)
	}
	// activeEnv 已不在响应里:没有默认环境了(环境一律点名)。
	for _, k := range []string{"config", "sshs", "listen", "debug", "query", "mirror", "sync"} {
		if _, has := out[k]; !has {
			t.Errorf("响应缺少 %q", k)
		}
	}
	if _, has := out["activeEnv"]; has {
		t.Error("响应里不该再有 activeEnv(默认环境已移除)")
	}
}

// 没有前端构建产物时不能把请求吞掉，而是给出引导页。
func TestRouting_NoSubsystems(t *testing.T) {
	s, _ := newTestServer(t, seededConfig)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/", nil))
	if rec.Code != http.StatusOK {
		t.Errorf("/debug/ → HTTP %d, 期望 200（引导页）", rec.Code)
	}
	body := rec.Body.String()
	if !contains(body, "前端尚未构建") {
		t.Errorf("/debug/ 没有给出引导页")
	}
	// 标题也要在：这条兜底路径上有过一版把 title 丢了，只剩一个光秃秃的「TT」——
	// 而那正是全新克隆第一次访问看到的页面。
	if !contains(body, "界面未构建") {
		t.Errorf("/debug/ 引导页丢了标题: %.120s", body)
	}
}

// 字典页那套 SPA 已经并进统一设置页，/dict/ 不再存在。
func TestRouting_DictPageRemoved(t *testing.T) {
	s, _ := newTestServer(t, seededConfig)
	for _, p := range []string{"/dict/", "/dict/api/mirror"} {
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s → HTTP %d, 期望 404（字典页已移除）", p, rec.Code)
		}
	}
}

// SPA 的静态服务：挂载点、深链接、真实资源各走各的路，都不能被重定向掉前缀。
func TestSPAHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html":      {Data: []byte("<html>APP</html>")},
		"assets/index.js": {Data: []byte("console.log(1)")},
		"favicon.ico":     {Data: []byte("ico")},
	}
	h := SPAHandler(fsys, "/debug/", "未构建")

	cases := []struct {
		path     string
		wantCode int
		wantBody string
	}{
		// 挂载点本身要给入口 HTML，不能 301 到 "/"（那会丢掉 /debug 前缀）
		{"/debug/", http.StatusOK, "APP"},
		// 深链接（前端路由）回落入口 HTML，而不是 404
		{"/debug/some/deep/link", http.StatusOK, "APP"},
		// 目录同样给入口 HTML
		{"/debug/assets", http.StatusOK, "APP"},
		// 真实文件原样返回
		{"/debug/assets/index.js", http.StatusOK, "console.log"},
		{"/debug/favicon.ico", http.StatusOK, "ico"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, c.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != c.wantCode {
			t.Errorf("%s → HTTP %d, 期望 %d", c.path, rec.Code, c.wantCode)
			continue
		}
		if !contains(rec.Body.String(), c.wantBody) {
			t.Errorf("%s 响应体不含 %q: %.60s", c.path, c.wantBody, rec.Body.String())
		}
	}
}

// 前端没构建（只有占位文件）时给引导页，而不是 404 或 panic。
func TestSPAHandler_NotBuilt(t *testing.T) {
	for _, fsys := range []fs.FS{nil, fstest.MapFS{".gitkeep": {Data: []byte("")}}} {
		h := SPAHandler(fsys, "/debug/", "调试工作台未构建")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/debug/", nil))
		if rec.Code != http.StatusOK {
			t.Errorf("HTTP %d, 期望 200 引导页", rec.Code)
		}
		if !contains(rec.Body.String(), "前端尚未构建") {
			t.Errorf("没有给出引导页: %.80s", rec.Body.String())
		}
	}
}

// 与 viaSsh 同源的另一处:每环境的调试覆盖项(launchArgs / watchdogSeconds)表单里
// 没有输入控件,保存站点会把它们丢掉。服务端按环境名补回。
func TestHostsPut_PreservesEnvDebugOverrides(t *testing.T) {
	seed := `{
  "schemaVersion": 2,
  "hosts": {"activeEnv": "开发环境", "sshs": [{
    "name": "开发环境", "host": "10.0.0.1", "port": 22, "user": "u", "password": "p",
    "zone": "36", "topent": "10001",
    "launchArgs": "BBDL512840855a 9 9 'Y' {prog}",
    "watchdogSeconds": 300
  }]}
}`
	s, path := newTestServer(t, seed)

	// 配置页提交:它只有 SSH/DB 那些字段,没有这两个覆盖项
	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{
			"activeEnv": "开发环境",
			"sshs": []any{map[string]any{
				"name": "开发环境", "host": "10.0.0.9", "port": 22, "user": "u", "password": "p",
				"zone": "36", "topent": "10001",
			}},
		},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
	}

	root, _ := config.Open(path)
	hosts, _ := root["hosts"].(map[string]any)
	sshs, _ := hosts["sshs"].([]any)
	e, _ := sshs[0].(map[string]any)

	if e["host"] != "10.0.0.9" {
		t.Errorf("表单改的 host 未生效: %v", e["host"])
	}
	if e["launchArgs"] != "BBDL512840855a 9 9 'Y' {prog}" {
		t.Errorf("每环境的 launchArgs 被抹掉了: %v", e["launchArgs"])
	}
	if e["watchdogSeconds"] != float64(300) {
		t.Errorf("每环境的 watchdogSeconds 被抹掉了: %v", e["watchdogSeconds"])
	}
}

// 配置文件不存在时也要能答,设置页在首次运行下就得渲染。
func TestConfigStatus_NoConfig(t *testing.T) {
	s, _ := newTestServer(t, "")
	rec, out := doJSON(t, s, http.MethodGet, "/api/config/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP %d", rec.Code)
	}
	if out["ok"] != true {
		t.Error("ok 应为 true")
	}
	for _, k := range []string{"config", "mirror", "sync", "install"} {
		if _, has := out[k]; !has {
			t.Errorf("响应缺少 %q", k)
		}
	}
}

// 派生状态要真的去查磁盘:目录/文件存不存在只有服务端算得出来。
func TestConfigStatus_DerivedFromDisk(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "mirror")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}
	syncFile := filepath.Join(dir, "erp_data.db")
	if err := os.WriteFile(syncFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	seed := `{"schemaVersion":2,"hosts":{"sshs":[]},
	  "mirror":{"dir":` + jsonStr(realDir) + `},
	  "sync":{"target":` + jsonStr(syncFile) + `}}`
	s, _ := newTestServer(t, seed)

	rec, out := doJSON(t, s, http.MethodGet, "/api/config/status", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP %d", rec.Code)
	}
	mirror, _ := out["mirror"].(map[string]any)
	if mirror["exists"] != true {
		t.Errorf("存在的镜像目录应报 exists=true: %v", mirror)
	}
	sync, _ := out["sync"].(map[string]any)
	if sync["exists"] != true {
		t.Errorf("存在的同步目标应报 exists=true: %v", sync)
	}
	if sync["configured"] == "" || sync["defaultTarget"] == "" {
		t.Errorf("应同时给出显式配置值与缺省位置: %v", sync)
	}
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && bytes.Contains([]byte(s), []byte(sub)))
}

// 工作区是**表单管得着**的字段：带了就存下，不带就删掉。
//
// 它刻意**不在** ungovernedEnvKeys 里（那里放的是 launchArgs / watchdogSeconds）：一旦被
// 保留机制补回来，用户在页面上清空工作区、保存、再进来看，那个路径又回来了 —— 而引擎会
// 继续开在它上面。这正是"切了环境就该换工作区"要防的事。
func TestHostsPutWorkspaceGoverned(t *testing.T) {
	s, path := newTestServer(t, seededConfig)
	put := func(extra map[string]any) {
		t.Helper()
		env := map[string]any{
			"name": "开发环境", "host": "10.0.0.1", "port": 22, "user": "u", "password": "p",
			"launchArgs": "BBDL1 {prog}", // 表单不管这个 → 必须被保留
		}
		for k, v := range extra {
			env[k] = v
		}
		rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
			"hosts": map[string]any{"activeEnv": "开发环境", "sshs": []any{env}},
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("PUT 失败 HTTP %d: %s", rec.Code, rec.Body.String())
		}
	}
	envOf := func() map[string]any {
		t.Helper()
		root, err := config.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		hosts, _ := root["hosts"].(map[string]any)
		sshs, _ := hosts["sshs"].([]any)
		if len(sshs) != 1 {
			t.Fatalf("环境数 = %d", len(sshs))
		}
		m, _ := sshs[0].(map[string]any)
		return m
	}

	put(map[string]any{"workspace": `D:\ws\客户甲`})
	if got := envOf()["workspace"]; got != `D:\ws\客户甲` {
		t.Errorf("提交的工作区没存下: %v", got)
	}

	// 清空（表单不再带这个键）→ 配置里也必须没有它，不能被保留机制补回来。
	put(nil)
	env := envOf()
	if _, has := env["workspace"]; has {
		t.Errorf("清空后工作区还在: %v —— 它被当成“表单不管的字段”保留了", env["workspace"])
	}
	if env["launchArgs"] != "BBDL1 {prog}" {
		t.Errorf("表单不管的 launchArgs 该被保留，得到 %v", env["launchArgs"])
	}
}

// GET /api/discover/envs 走通,且**没装设计器时是一份正常响应**(不是 5xx)。
//
// 这条的重点是响应形状:前端据此渲染弹窗 —— 所以字段必须稳定,
// 且 reason 要能表达"没找到"这种正常结论。
func TestDiscoverEnvsEndpoint(t *testing.T) {
	s, _ := newTestServer(t, seededConfig)

	// 不碰用户真实的设计器目录:指向一个不存在的路径。
	t.Setenv(designer.EnvOverride, filepath.Join(t.TempDir(), "没有这个文件.xml"))

	rec, out := doJSON(t, s, http.MethodGet, "/api/discover/envs", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HTTP %d(没装设计器是正常情况,不该当错误)", rec.Code)
	}
	if _, has := out["envs"]; !has {
		t.Error("响应里应当有 envs(哪怕是空数组)")
	}
	if _, has := out["found"]; !has {
		t.Error("响应里应当有 found")
	}
	if out["found"] != false {
		t.Errorf("文件不存在时 found 应为 false,得到 %v", out["found"])
	}
	if r, _ := out["reason"].(string); r == "" {
		t.Error("应当给出 reason,否则用户看到空列表不知道是'没装'还是'读坏了'")
	}
}

// 空清单是合法状态：首次运行没有环境，设置页也要能把最后一个环境删掉。
func TestHostsPut_EmptyListAllowed(t *testing.T) {
	s, path := newTestServer(t, seededConfig)
	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("空清单应允许: HTTP %d %s", rec.Code, rec.Body.String())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	hosts, _ := root["hosts"].(map[string]any)
	sshs, _ := hosts["sshs"].([]any)
	if len(sshs) != 0 {
		t.Fatalf("sshs 应为空，实得 %v", sshs)
	}
}

// 目标落点已被占用：整次拒绝（配置一个字节都不写），不能出现“配置改了、文件没搬”的半套。
func TestHostsPut_RenameConflictLeavesConfigUntouched(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","password":"p","zone":"1"}]},
	  "query":{"source":"auto"}}`
	s, path := newTestServer(t, seed)
	dir := filepath.Dir(path)
	for _, n := range []string{"甲", "乙"} {
		p := config.DictDBPath(dir, n)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(n), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "乙", "host": "h", "port": 22, "user": "u", "password": "p", "zone": "1",
		}}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("目标被占用应 400，实际 HTTP %d：%s", rec.Code, rec.Body.String())
	}
	root, _ := config.Load(path)
	if len(root.Hosts.SSHs) != 1 || root.Hosts.SSHs[0].Name != "甲" {
		t.Fatalf("配置不该被改：%+v", root.Hosts.SSHs)
	}
	if _, err := os.Stat(config.DictDBPath(dir, "甲")); err != nil {
		t.Errorf("旧库不该被动：%v", err)
	}
}

// 新增/改名的环境名会进 dict/<名>.db、ents/<名>.json 与 local:<名>：
// 路径分隔符/冒号必须当场拒绝，老配置里已有的名字则不追溯。
func TestHostsPut_RejectsUnsafeNewName(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","password":"p","zone":"1"}]},
	  "query":{"source":"auto"}}`
	s, path := newTestServer(t, seed)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{
			map[string]any{"name": "甲", "host": "h", "port": 22, "user": "u", "password": "p", "zone": "1"},
			map[string]any{"name": "a/b", "host": "h2", "port": 22, "user": "u", "password": "p", "zone": "1"},
		}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("非法环境名应 400，实际 HTTP %d：%s", rec.Code, rec.Body.String())
	}
	root, _ := config.Load(path)
	if len(root.Hosts.SSHs) != 1 || root.Hosts.SSHs[0].Name != "甲" {
		t.Fatalf("配置不该被改：%+v", root.Hosts.SSHs)
	}
}

// 删一个环境 + 加一个无关环境（同一次保存）：不能被当成改名而搬走旧数据。
func TestHostsPut_DeletePlusAddDoesNotMoveData(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"old-host","port":22,"user":"u","password":"p","zone":"1"}]},
	  "query":{"source":"auto"}}`
	s, path := newTestServer(t, seed)
	dir := filepath.Dir(path)
	oldDB := config.DictDBPath(dir, "甲")
	if err := os.MkdirAll(filepath.Dir(oldDB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldDB, []byte("甲的数据"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "丙", "host": "new-host", "port": 22, "user": "u", "password": "p", "zone": "9",
		}}},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("删一个加一个应允许保存，实际 HTTP %d：%s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(oldDB); err != nil {
		t.Errorf("被删环境的库不该搬走：%v", err)
	}
	if _, err := os.Stat(config.DictDBPath(dir, "丙")); !os.IsNotExist(err) {
		t.Errorf("新环境名下不该出现旧数据：%v", err)
	}
}

// 改名同时改了 host：改名识别不出来，旧名从配置里消失而 query.source 还指着它 ——
// 必须当场拒绝，而不是留下悬空引用。
func TestHostsPut_UndetectedRenameWithDanglingSourceRejected(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"h1","port":22,"user":"u","password":"p","zone":"1"}]},
	  "query":{"source":"local:甲"}}`
	s, path := newTestServer(t, seed)
	dir := filepath.Dir(path)
	oldDB := config.DictDBPath(dir, "甲")
	if err := os.MkdirAll(filepath.Dir(oldDB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oldDB, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "乙", "host": "h2", "port": 22, "user": "u", "password": "p", "zone": "1",
		}}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("悬空的 query.source 应拒绝保存，实际 HTTP %d：%s", rec.Code, rec.Body.String())
	}
	root, _ := config.Load(path)
	if len(root.Hosts.SSHs) != 1 || root.Hosts.SSHs[0].Name != "甲" {
		t.Fatalf("配置不该被改：%+v", root.Hosts.SSHs)
	}
	if root.Query.Source != "local:甲" {
		t.Errorf("query.source 不该被改：%q", root.Query.Source)
	}
	if _, err := os.Stat(oldDB); err != nil {
		t.Errorf("旧库不该被动：%v", err)
	}
}

// 设置页能自由写环境的 watchdogSeconds（ungovernedEnvKeys）。负数等于关掉防行锁保护，
// 必须在保存时拦住 —— 一个"看起来还在、实际不生效"的保护开关比没有更危险。
func TestHostsPut_RejectsNegativeWatchdog(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","password":"p","zone":"1"}]}}`
	s, _ := newTestServer(t, seed)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "甲", "host": "h", "port": 22, "user": "u", "password": "p", "zone": "1",
			"watchdogSeconds": -1,
		}}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("负数应被拒绝，实得 HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "关闭看门狗") {
		t.Errorf("错误信息应说清后果：%s", rec.Body.String())
	}
}

// T100 与 TOP GP 共用 hosts.sshs，环境名是两边的路径键；T100 侧也必须反向拦 GP 同名，
// 否则 T100 保存会静默丢掉/覆盖同名的 GP 环境。
func TestHostsPut_RejectsGPNameCollision(t *testing.T) {
	seed := `{"schemaVersion":4,
	  "hosts":{"sshs":[
	    {"name":"t100","host":"h1","port":22,"user":"u","password":"p","zone":"1"},
	    {"name":"gp","host":"h2","port":22,"user":"u","password":"p","product":"gp"}
	  ]}}`
	s, path := newTestServer(t, seed)

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"hosts": map[string]any{"sshs": []any{map[string]any{
			"name": "gp", "host": "h1", "port": 22, "user": "u", "password": "p", "zone": "1",
		}}},
	})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("T100 与 GP 同名应拒绝，实得 HTTP %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "TOP GP") {
		t.Errorf("错误信息应指明冲突的是 GP 环境：%s", rec.Body.String())
	}

	root, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Hosts.SSHs) != 2 {
		t.Fatalf("拒绝保存后配置不该变：%+v", root.Hosts.SSHs)
	}
	names := []string{root.Hosts.SSHs[0].Name, root.Hosts.SSHs[1].Name}
	if names[0] != "t100" || names[1] != "gp" {
		t.Fatalf("原环境应保持：%v", names)
	}
}
