package web

import (
	"net/http"
	"path/filepath"
	"testing"

	"tg/internal/config"
)

// 设置页那张「Genero BDL 手册」卡片读写的是同一节：GET /api/hosts 给出 bdldoc.dir，
// PUT 回去要落进配置，/api/config/status 要跟着报新的生效目录。
func TestHostsBDLDocRoundTrip(t *testing.T) {
	s, cfgPath := newTestServer(t, `{"schemaVersion":5,"hosts":{"sshs":[]}}`)
	dir := filepath.Join(t.TempDir(), "docs")

	rec, _ := doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"bdldoc": map[string]any{"dir": dir},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT HTTP %d: %s", rec.Code, rec.Body.String())
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if root.BDLDoc.Dir != dir {
		t.Fatalf("配置里的 bdldoc.dir = %q, want %q", root.BDLDoc.Dir, dir)
	}

	_, out := doJSON(t, s, http.MethodGet, "/api/hosts", nil)
	bd, _ := out["bdldoc"].(map[string]any)
	if bd["dir"] != dir {
		t.Fatalf("GET 的 bdldoc.dir = %v, want %q", bd, dir)
	}

	_, st := doJSON(t, s, http.MethodGet, "/api/config/status", nil)
	sb, _ := st["bdldoc"].(map[string]any)
	if sb["dir"] != dir {
		t.Fatalf("status 的 bdldoc.dir = %v, want %q", sb, dir)
	}
	if sb["installed"] != false {
		t.Fatalf("目录还没建，installed 应为 false: %v", sb)
	}

	// 清空：回到缺省落点（配置里那条键被清掉，卡片显示"留空"）。
	rec, _ = doJSON(t, s, http.MethodPut, "/api/hosts", map[string]any{
		"bdldoc": map[string]any{"dir": ""},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT 清空 HTTP %d: %s", rec.Code, rec.Body.String())
	}
	_, out = doJSON(t, s, http.MethodGet, "/api/hosts", nil)
	bd, _ = out["bdldoc"].(map[string]any)
	if bd["dir"] != "" {
		t.Fatalf("清空后 GET 的 bdldoc.dir = %v, want 空", bd)
	}
	_, st = doJSON(t, s, http.MethodGet, "/api/config/status", nil)
	sb, _ = st["bdldoc"].(map[string]any)
	if want := filepath.Join(config.LocationsAt(cfgPath).DataDir(), config.BDLDocDirName); sb["dir"] != want {
		t.Fatalf("清空后 status 的生效目录 = %v, want %q", sb["dir"], want)
	}
}
