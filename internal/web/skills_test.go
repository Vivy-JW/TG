package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/config"
)

// withOfficialSkills 在临时目录造一棵官方技能树，并让本包把它当成 <exe>/skills。
func withOfficialSkills(t *testing.T, names ...string) string {
	t.Helper()
	dir := t.TempDir()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + n + "\ndescription: 官方技能\n---\n# " + n + "\n"
		if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old := officialSkillsDirFn
	officialSkillsDirFn = func() (string, error) { return dir, nil }
	t.Cleanup(func() { officialSkillsDirFn = old })
	return dir
}

// mkExternal 造一个第三方技能目录。
func mkExternal(t *testing.T, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: " + name + "\ndescription: 第三方技能\n---\n# " + name + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func rows(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	raw, ok := out["externals"].([]any)
	if !ok {
		t.Fatalf("响应里没有 externals 数组: %v", out)
	}
	rows := make([]map[string]any, 0, len(raw))
	for _, it := range raw {
		m, _ := it.(map[string]any)
		rows = append(rows, m)
	}
	return rows
}

// 列表：官方（只读）+ 第三方（带状态）+ 上次装到过哪些目录。
func TestSkillsGetListsBothSides(t *testing.T) {
	withOfficialSkills(t, "tt-erp-read")
	ok := mkExternal(t, "kb-faq")
	gone := filepath.Join(t.TempDir(), "gone")
	s, _ := newTestServer(t, `{"schemaVersion":4,"skills":{"externals":[
		{"path":`+jsonQuote(ok)+`,"label":"知识库"},
		{"path":`+jsonQuote(gone)+`}
	]}}`)

	rec, out := doJSON(t, s, "GET", "/api/skills", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, body = %s", rec.Code, rec.Body.String())
	}
	official, _ := out["official"].([]any)
	if len(official) != 1 {
		t.Fatalf("官方该有 1 条: %v", out["official"])
	}
	rs := rows(t, out)
	if len(rs) != 2 {
		t.Fatalf("第三方该有 2 条: %v", rs)
	}
	if rs[0]["name"] != "kb-faq" || rs[0]["state"] != "ok" || rs[0]["label"] != "知识库" {
		t.Errorf("第一条 = %v, want kb-faq/ok/知识库", rs[0])
	}
	if rs[1]["state"] != "missing" || rs[1]["why"] == "" {
		t.Errorf("第二条该是 missing 且带原因: %v", rs[1])
	}
}

// 添加：不合规直接拒（当场反馈），失联允许存下但要说一声。
func TestSkillsAddValidation(t *testing.T) {
	withOfficialSkills(t, "tt-erp-read")
	s, path := newTestServer(t, `{"schemaVersion":4}`)

	bad := filepath.Join(t.TempDir(), "bad")
	if err := os.MkdirAll(bad, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "SKILL.md"), []byte("---\nname: 别的名字\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rec, _ := doJSON(t, s, "POST", "/api/skills/add", map[string]any{"path": bad}); rec.Code == http.StatusOK {
		t.Fatalf("名字与目录名不符的技能不该登记成功: %s", rec.Body.String())
	}

	// 与官方撞名：也拒（官方优先，登记了也永远装不上）
	clash := mkExternal(t, "tt-erp-read")
	if rec, _ := doJSON(t, s, "POST", "/api/skills/add", map[string]any{"path": clash}); rec.Code == http.StatusOK {
		t.Fatalf("与官方同名的技能不该登记成功: %s", rec.Body.String())
	}

	// 失联：允许存，但带一句说明
	gone := filepath.Join(t.TempDir(), "gone")
	rec, out := doJSON(t, s, "POST", "/api/skills/add", map[string]any{"path": gone})
	if rec.Code != http.StatusOK {
		t.Fatalf("失联路径该允许登记（移动硬盘没插也得能先存着）: %s", rec.Body.String())
	}
	if out["warning"] == "" || out["warning"] == nil {
		t.Errorf("失联登记该带一句说明: %v", out)
	}

	root, err := loadConfig(t, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Skills.Externals) != 1 {
		t.Fatalf("配置里该有 1 条登记: %+v", root.Skills)
	}
}

// 同一路径重复添加是幂等的（返回 unchanged），不会变成两条登记。
func TestSkillsAddDedup(t *testing.T) {
	withOfficialSkills(t, "tt-erp-read")
	kb := mkExternal(t, "kb-faq")
	s, _ := newTestServer(t, `{"schemaVersion":4,"skills":{"externals":[{"path":`+jsonQuote(kb)+`}]}}`)

	rec, out := doJSON(t, s, "POST", "/api/skills/add", map[string]any{"path": strings.ToUpper(kb)})
	if rec.Code != http.StatusOK {
		t.Fatalf("重复添加该是成功的幂等操作: %s", rec.Body.String())
	}
	if out["unchanged"] != true {
		t.Errorf("重复添加该报 unchanged: %v", out)
	}
}

func TestSkillsRemove(t *testing.T) {
	withOfficialSkills(t, "tt-erp-read")
	kb := mkExternal(t, "kb-faq")
	s, path := newTestServer(t, `{"schemaVersion":4,"skills":{"externals":[{"path":`+jsonQuote(kb)+`}]}}`)

	rec, out := doJSON(t, s, "POST", "/api/skills/remove", map[string]any{"path": kb})
	if rec.Code != http.StatusOK || out["removed"] != true {
		t.Fatalf("删除该成功: %s", rec.Body.String())
	}
	root, err := loadConfig(t, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(root.Skills.Externals) != 0 {
		t.Fatalf("登记表该空了: %+v", root.Skills)
	}

	// 再删一次：幂等，不是错误
	rec, out = doJSON(t, s, "POST", "/api/skills/remove", map[string]any{"path": kb})
	if rec.Code != http.StatusOK || out["removed"] != false {
		t.Fatalf("重复删除该是 ok/removed=false: %s", rec.Body.String())
	}
}

// jsonQuote 把 Windows 路径塞进 JSON 字符串。
func jsonQuote(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

// loadConfig 读回配置类型化视图。
func loadConfig(t *testing.T, path string) (*config.Root, error) {
	t.Helper()
	return config.Load(path)
}
