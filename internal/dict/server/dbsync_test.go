package server

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"tt/internal/config"
)

// GET /api/dbsync:库目录 + 各环境自己的 .db（按环境分文件，互不覆盖）。
func TestDBSyncGet(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"sshs":[
      {"name":"a","host":"1.2.3.4","user":"u","password":"p"},
      {"name":"b","host":"5.6.7.8","user":"u","password":"p",
       "db":{"type":"oracle","host":"5.6.7.8","port":1521,"service":"s","accounts":[{"account":"x","password":"x"}]}}]}}`)
	dir := t.TempDir()
	s := New(p)
	s.SetDBDir(dir)

	rec := httptest.NewRecorder()
	s.hDBSyncGet(rec, httptest.NewRequest("GET", "/api/dbsync", nil))
	var resp dbSyncResp
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Dir != dir {
		t.Fatalf("dir = %q, want %q", resp.Dir, dir)
	}
	if len(resp.Envs) != 1 || resp.Envs[0].Name != "b" || resp.Envs[0].Type != "oracle" {
		t.Fatalf("应只列挂库环境: %+v", resp.Envs)
	}
	want := config.DictDBPathIn(dir, "b")
	if resp.Envs[0].DBPath != want {
		t.Fatalf("dbPath = %q, want %q", resp.Envs[0].DBPath, want)
	}
	if resp.Envs[0].Exists {
		t.Fatal("库文件尚不存在，exists 应为 false")
	}
	if resp.Job.Running {
		t.Fatal("初始不应有任务在跑")
	}

	// 建成 b 的库后，状态按文件真实反映
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("sqlite"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	s.hDBSyncGet(rec, httptest.NewRequest("GET", "/api/dbsync", nil))
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Envs[0].Exists || resp.Envs[0].Size != int64(len("sqlite")) {
		t.Fatalf("已存在的库应报 exists/size: %+v", resp.Envs[0])
	}
}

// 不同环境必须落到不同文件（隔离的结构判据）。
func TestDBSyncPerEnvPathsDiffer(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"sshs":[
      {"name":"a","host":"1.2.3.4","user":"u","password":"p",
       "db":{"type":"oracle","host":"1.2.3.4","service":"s","accounts":[{"account":"x","password":"x"}]}},
      {"name":"b","host":"5.6.7.8","user":"u","password":"p",
       "db":{"type":"kingbase","host":"5.6.7.8","database":"d","accounts":[{"account":"x","password":"x"}]}}]}}`)
	dir := t.TempDir()
	s := New(p)
	s.SetDBDir(dir)
	pa, pb := s.dbPathFor("a"), s.dbPathFor("b")
	if pa == pb {
		t.Fatalf("两个环境指向同一个文件: %s", pa)
	}
	if filepath.Dir(pa) != dir || filepath.Dir(pb) != dir {
		t.Fatalf("库文件应在库目录 %s 下: %s / %s", dir, pa, pb)
	}
}

// 拉取前置校验:空环境/未知环境/未挂库均 400。
func TestDBSyncPostValidation(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"sshs":[{"name":"nodb","host":"1.2.3.4","user":"u","password":"p"}]}}`)
	s := New(p)

	for _, body := range []string{`{"env":""}`, `{"env":"nope"}`, `{"env":"nodb"}`} {
		rec := httptest.NewRecorder()
		s.hDBSyncPost(rec, httptest.NewRequest("POST", "/api/dbsync", bytes.NewBufferString(body)))
		if rec.Code != 400 {
			t.Fatalf("body %s 应 400, got %d (%s)", body, rec.Code, rec.Body.String())
		}
	}
}

// 已有任务在跑时再启动返回 409(不触发真实同步)。
func TestDBSyncRunningConflict(t *testing.T) {
	p := writeTempConfig(t, `{"hosts":{"sshs":[{"name":"a","host":"1.2.3.4","user":"u","password":"p",
      "db":{"type":"oracle","host":"1.2.3.4","service":"s","accounts":[{"account":"x","password":"x"}]}}]}}`)
	s := New(p)
	s.sync.Running = true

	rec := httptest.NewRecorder()
	s.hDBSyncPost(rec, httptest.NewRequest("POST", "/api/dbsync", bytes.NewBufferString(`{"env":"a"}`)))
	if rec.Code != 409 {
		t.Fatalf("已有任务应 409, got %d", rec.Code)
	}
}
