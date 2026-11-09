package envmove

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tt/internal/config"
	"tt/internal/entdir"
)

// writeConfigRaw 写一份最小配置（body 里的 hosts/query 由调用方给）。
func writeConfigRaw(t *testing.T, dir, body string) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func writeConfig(t *testing.T, dir, source string) string {
	t.Helper()
	cfg := map[string]any{
		"schemaVersion": config.SchemaVersion,
		"listen":        "127.0.0.1:1",
		"hosts": map[string]any{"sshs": []any{
			map[string]any{"name": "甲", "host": "10.0.0.1", "port": 22, "user": "u", "zone": "1"},
		}},
		"query": map[string]any{"source": source},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return writeConfigRaw(t, dir, string(b))
}

func touch(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// 改名的落点都要跟着走，配置里 hosts 与 query.source 同时更新。
func TestRenameMigratesAllPaths(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "local:甲")

	dictOld := config.DictDBPath(dir, "甲")
	entsOld := entdir.Path(dir, entdir.EnvSeg("甲", "10.0.0.1", "1"))
	mirrorOld := filepath.Join(dir, "srccache", entdir.EnvSeg("甲", "10.0.0.1", "1"))
	touch(t, dictOld, "dict")
	touch(t, dictOld+".bak", "bak")
	touch(t, entsOld, `{"version":1,"env":"甲","fingerprint":"x","mappings":[{"ent":99,"account":"a","placeholder":false}]}`)
	touch(t, filepath.Join(mirrorOld, "u1", "com", "a.4gl"), "src")

	moves, err := Plan(cfgPath, "甲", "乙", "10.0.0.1", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(moves) != 4 { // dict / dict-bak / ents / srccache
		t.Fatalf("该搬 4 份，实得 %d：%+v", len(moves), moves)
	}
	if err := Execute(moves, "乙", "甲"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rewritten, err := config.RenameHost(cfgPath, "甲", "乙")
	if err != nil {
		t.Fatalf("RenameHost: %v", err)
	}
	if !rewritten {
		t.Error("query.source 指向旧名，改名时该一起改写")
	}

	for _, old := range []string{dictOld, dictOld + ".bak", entsOld, mirrorOld} {
		if _, err := os.Stat(old); !os.IsNotExist(err) {
			t.Errorf("旧落点还在：%s", old)
		}
	}
	dictNew := config.DictDBPath(dir, "乙")
	entsNew := entdir.Path(dir, entdir.EnvSeg("乙", "10.0.0.1", "1"))
	for _, p := range []string{dictNew, dictNew + ".bak", entsNew, filepath.Join(dir, "srccache", "乙")} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("新落点不存在：%s（%v）", p, err)
		}
	}

	root, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if root.Hosts.SSHs[0].Name != "乙" {
		t.Errorf("环境名 = %q，期望 乙", root.Hosts.SSHs[0].Name)
	}
	if root.Query.Source != "local:乙" {
		t.Errorf("query.source = %q，期望 local:乙", root.Query.Source)
	}

	b, err := os.ReadFile(entsNew)
	if err != nil {
		t.Fatal(err)
	}
	var snap map[string]any
	if err := json.Unmarshal(b, &snap); err != nil {
		t.Fatal(err)
	}
	if snap["env"] != "乙" {
		t.Errorf("企业快照里的 env = %v，期望 乙", snap["env"])
	}
}

// dict.dir / 环境 dictDir 指到别处时，也要搬到那个目录（而不是只看 <数据目录>/dict）。
func TestPlanUsesConfiguredDictDir(t *testing.T) {
	dir := t.TempDir()
	custom := filepath.Join(dir, "custom-dict")
	cfgPath := writeConfigRaw(t, dir, `{"schemaVersion":4,
	  "dict":{"dir":"`+jsonEscape(custom)+`"},
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","zone":"1"}]},
	  "query":{"source":"auto"}}`)
	touch(t, filepath.Join(custom, "甲.db"), "db")

	moves, err := Plan(cfgPath, "甲", "乙", "h", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	found := false
	for _, m := range moves {
		if m.Kind == "dict" {
			found = true
			if m.From != filepath.Join(custom, "甲.db") || m.To != filepath.Join(custom, "乙.db") {
				t.Errorf("dict 落点算错：%+v", m)
			}
		}
	}
	if !found {
		t.Fatalf("自定义 dict.dir 下的库没有被规划搬迁：%+v", moves)
	}
}

// 环境自己的 dictDir 优先于顶层 dict.dir。
func TestPlanUsesEnvDictDir(t *testing.T) {
	dir := t.TempDir()
	envDir := filepath.Join(dir, "env-dict")
	cfgPath := writeConfigRaw(t, dir, `{"schemaVersion":4,
	  "dict":{"dir":"`+jsonEscape(filepath.Join(dir, "top-dict"))+`"},
	  "hosts":{"sshs":[{"name":"甲","dictDir":"`+jsonEscape(envDir)+`","host":"h","port":22,"user":"u","zone":"1"}]},
	  "query":{"source":"auto"}}`)
	touch(t, filepath.Join(envDir, "甲.db"), "db")

	moves, err := Plan(cfgPath, "甲", "乙", "h", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, m := range moves {
		if m.Kind == "dict" && m.From != filepath.Join(envDir, "甲.db") {
			t.Errorf("环境 dictDir 没被优先使用：%+v", m)
		}
	}
}

// `tt file mirror` 的镜像根按环境名分目录，改名时要一起搬；环境自己指定 mirrorDir 时跳过。
func TestPlanMovesMirrorUnderRoot(t *testing.T) {
	dir := t.TempDir()
	mirrorRoot := filepath.Join(dir, "mirrors")
	cfgPath := writeConfigRaw(t, dir, `{"schemaVersion":4,
	  "mirror":{"dir":"`+jsonEscape(mirrorRoot)+`"},
	  "hosts":{"sshs":[{"name":"甲","host":"h","port":22,"user":"u","zone":"1"}]},
	  "query":{"source":"auto"}}`)
	touch(t, filepath.Join(mirrorRoot, "甲", "erp", "a.4gl"), "src")

	moves, err := Plan(cfgPath, "甲", "乙", "h", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	found := false
	for _, m := range moves {
		if m.Kind == "mirror" {
			found = true
			if m.From != filepath.Join(mirrorRoot, "甲") || m.To != filepath.Join(mirrorRoot, "乙") {
				t.Errorf("mirror 落点算错：%+v", m)
			}
		}
	}
	if !found {
		t.Fatalf("镜像根下的环境目录没有规划搬迁：%+v", moves)
	}
}

// 环境自配 mirrorDir 是绝对路径、与名字无关：不该搬。
func TestPlanSkipsPerEnvMirrorDir(t *testing.T) {
	dir := t.TempDir()
	own := filepath.Join(dir, "my-mirror")
	cfgPath := writeConfigRaw(t, dir, `{"schemaVersion":4,
	  "mirror":{"dir":"`+jsonEscape(filepath.Join(dir, "root"))+`"},
	  "hosts":{"sshs":[{"name":"甲","mirrorDir":"`+jsonEscape(own)+`","host":"h","port":22,"user":"u","zone":"1"}]},
	  "query":{"source":"auto"}}`)
	touch(t, filepath.Join(own, "x.4gl"), "src")

	moves, err := Plan(cfgPath, "甲", "乙", "h", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	for _, m := range moves {
		if m.Kind == "mirror" {
			t.Errorf("环境自配 mirrorDir 不该搬：%+v", m)
		}
	}
}

// Execute 中途失败时回滚已搬的落点，并把企业快照的 env 改回旧名。
func TestExecuteRollbackRestoresSnapshot(t *testing.T) {
	dir := t.TempDir()
	entsFrom := filepath.Join(dir, "ents", "甲.json")
	touch(t, entsFrom, `{"version":1,"env":"甲"}`)

	moves := []Move{
		{Kind: "ents", From: entsFrom, To: filepath.Join(dir, "ents", "乙.json")},
		{Kind: "dict", From: filepath.Join(dir, "nope.db"), To: filepath.Join(dir, "nope2.db")},
	}
	err := Execute(moves, "乙", "甲")
	if err == nil {
		t.Fatal("第二个落点不存在，Execute 应该失败")
	}
	if _, err := os.Stat(entsFrom); err != nil {
		t.Errorf("回滚后旧快照不在：%v", err)
	}
	b, _ := os.ReadFile(entsFrom)
	var snap map[string]any
	_ = json.Unmarshal(b, &snap)
	if snap["env"] != "甲" {
		t.Errorf("回滚后快照 env = %v，期望 甲", snap["env"])
	}
}

// 目标名已被占：整次拒绝，一个都不搬。
func TestPlanRefusesExistingTarget(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "auto")
	touch(t, filepath.Join(dir, "dict", "甲.db"), "a")
	touch(t, filepath.Join(dir, "dict", "乙.db"), "b")

	if _, err := Plan(cfgPath, "甲", "乙", "h", "1"); err == nil {
		t.Fatal("目标已存在时必须报错，不能覆盖")
	}
	if _, err := os.Stat(filepath.Join(dir, "dict", "甲.db")); err != nil {
		t.Errorf("报错后源文件不该动：%v", err)
	}
}

// 什么都没同步过/调试过：计划为空，但改名本身仍要成立。
func TestPlanWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	cfgPath := writeConfig(t, dir, "auto")
	moves, err := Plan(cfgPath, "甲", "乙", "h", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(moves) != 0 {
		t.Fatalf("不该有落点要搬：%+v", moves)
	}
	if rewritten, err := config.RenameHost(cfgPath, "甲", "乙"); err != nil || rewritten {
		t.Fatalf("RenameHost: rewritten=%v err=%v", rewritten, err)
	}
}

// 保存时校验：指向不存在的环境必须报错，指向存在的放行；裸 local 也不放行。
func TestValidateQuerySource(t *testing.T) {
	root := func(src string) map[string]any {
		return map[string]any{
			"hosts": map[string]any{"sshs": []any{map[string]any{"name": "甲"}}},
			"query": map[string]any{"source": src},
		}
	}
	for _, bad := range []string{"local:乙", "乙", "local:", "local"} {
		if err := config.ValidateQuerySource(root(bad)); err == nil {
			t.Errorf("query.source = %q 该报错", bad)
		}
	}
	for _, ok := range []string{"", "auto", "local:甲", "甲"} {
		if err := config.ValidateQuerySource(root(ok)); err != nil {
			t.Errorf("query.source = %q 不该报错：%v", ok, err)
		}
	}
}

// 新环境名不允许路径分隔符/冒号（会破坏落点与 local: 前缀）。
func TestValidateEnvName(t *testing.T) {
	for _, bad := range []string{"", " 甲", "甲 ", "a/b", `a\b`, "a:b", ".", ".."} {
		if err := config.ValidateEnvName(bad); err == nil {
			t.Errorf("环境名 %q 该被拒绝", bad)
		}
	}
	for _, ok := range []string{"甲", "测试 区", "prod-2"} {
		if err := config.ValidateEnvName(ok); err != nil {
			t.Errorf("环境名 %q 不该被拒绝：%v", ok, err)
		}
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

// execlog 与 srccache 同源（<dataDir>/<种类>/<环境段>/runNN/）：调试跑过命令的
// 大输出副本按环境段落盘，改名不搬就会留下孤儿目录。
func TestPlanMovesExecLog(t *testing.T) {
	dir := t.TempDir()
	seg := entdir.EnvSeg("甲", "10.0.0.1", "1")
	execlogOld := filepath.Join(dir, "execlog", seg, "run1")
	touch(t, filepath.Join(execlogOld, "cmd-1.log"), "out")

	moves, err := Plan(filepath.Join(dir, "config.json"), "甲", "乙", "10.0.0.1", "1")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	var found *Move
	for i := range moves {
		if moves[i].Kind == "execlog" {
			found = &moves[i]
		}
	}
	if found == nil {
		t.Fatalf("execlog 没进计划：%+v", moves)
	}

	if err := Execute(moves, "乙", "甲"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "execlog", seg)); !os.IsNotExist(err) {
		t.Errorf("旧环境段的 execlog 还在：%s", filepath.Join(dir, "execlog", seg))
	}
	if _, err := os.Stat(filepath.Join(dir, "execlog", "乙", "run1", "cmd-1.log")); err != nil {
		t.Errorf("新环境段的 execlog 内容没搬过来：%v", err)
	}
}
