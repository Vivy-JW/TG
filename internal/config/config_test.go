package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// EditSection 只能动自己那一节，其他顶层键（含未知键）必须原样保留。
func TestEditSection_PreservesOtherKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	seed := `{
  "schemaVersion": 2,
  "listen": "127.0.0.1:28670",
  "hosts": {"activeEnv": "a", "sshs": [{"name": "a", "host": "h", "port": 22, "user": "u", "password": "p"}]},
  "query": {"source": "local"},
  "未来的键": {"x": 1}
}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	err := EditSection(path, "query", nil, func(sec map[string]any) error {
		sec["source"] = "生产环境"
		return nil
	})
	if err != nil {
		t.Fatalf("EditSection: %v", err)
	}

	root, _ := Open(path)
	if root["未来的事"] != nil {
		t.Error("不该出现这个键")
	}
	if _, has := root["未来的键"]; !has {
		t.Error("未知顶层键被抹掉了")
	}
	if root["listen"] != "127.0.0.1:28670" {
		t.Error("其他已知节被改动了")
	}
	q, _ := root["query"].(map[string]any)
	if q["source"] != "生产环境" {
		t.Errorf("目标节未写入: %v", q)
	}
	hosts, _ := root["hosts"].(map[string]any)
	if hosts["activeEnv"] != "a" {
		t.Error("hosts 节被误改")
	}
}

// EditHosts 读-改-写环境清单。
func TestEditHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// 文件不存在时也能写（首次运行路径）
	err := EditHosts(path, func(h *Hosts) error {
		h.SSHs = append(h.SSHs, NamedSsh{
			Name:      "新环境",
			SSHConfig: SSHConfig{Host: "h", Port: 22, User: "u", Password: "p"},
		})
		return nil
	})
	if err != nil {
		t.Fatalf("EditHosts: %v", err)
	}

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r.Hosts.SSHs) != 1 || r.Hosts.SSHs[0].Name != "新环境" {
		t.Fatalf("环境写入不正确: %+v", r.Hosts.SSHs)
	}
	if r.Hosts.ByName("新环境") == nil {
		t.Error("新写入的环境应当能按名解析到")
	}
	// 原先这里断言 "activeEnv 未兜底为首条" —— 默认环境已移除,
	// 环境不再有任何落盘的默认值(见 Hosts 的注释)。
	if r.SchemaVersion != SchemaVersion {
		t.Errorf("写入未打上 schemaVersion: %d", r.SchemaVersion)
	}
}

// SaveHosts 整体替换 hosts 节，其他节不动。
func TestSaveHosts_ReplacesOnlyHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	seed := `{"schemaVersion":2,"hosts":{"sshs":[]},"query":{"source":"local"}}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	err := SaveHosts(path, Hosts{
		SSHs: []NamedSsh{{
			Name:      "e1",
			SSHConfig: SSHConfig{Host: "h", Port: 2222, User: "u", Password: "p"},
		}},
	})
	if err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}

	root, _ := Open(path)
	q, _ := root["query"].(map[string]any)
	if q["source"] != "local" {
		t.Error("SaveHosts 误改了 query 节")
	}
	r, _ := Load(path)
	if len(r.Hosts.SSHs) != 1 || r.Hosts.SSHs[0].Port != 2222 {
		t.Errorf("hosts 未正确写入: %+v", r.Hosts)
	}
}

// Open 对缺失文件返回空配置，对非法 JSON 报错，对空文件返回空配置。
func TestOpen(t *testing.T) {
	dir := t.TempDir()

	root, err := Open(filepath.Join(dir, "missing.json"))
	if err != nil {
		t.Fatalf("缺失文件应当返回空配置而不是错误: %v", err)
	}
	if len(root) != 0 {
		t.Errorf("缺失文件应返回空 map, 得到 %v", root)
	}

	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if root, err = Open(empty); err != nil || len(root) != 0 {
		t.Errorf("空文件应视为空配置: root=%v err=%v", root, err)
	}

	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{oops"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(bad); err == nil {
		t.Error("非法 JSON 应当报错")
	}
}

// Save 必须是原子的：写完不留 .tmp 残渣，且保留原文件权限位。
func TestSave_AtomicAndKeepsMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := Save(path, map[string]any{"a": 1}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("目录里应当只有一个文件, 实为 %v", names)
	}

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows 的权限模型只有只读位，Go 的 os.Stat 一律报 0666/0444，
	// 断言 Unix 权限位没有意义；那里只验证文件可写（即没被写成只读）。
	if runtime.GOOS == "windows" {
		if fi.Mode().Perm()&0o200 == 0 {
			t.Errorf("新建配置不应是只读: %v", fi.Mode().Perm())
		}
	} else if fi.Mode().Perm() != 0o600 {
		t.Errorf("新建配置权限 = %v, 期望 0600（含明文口令）", fi.Mode().Perm())
	}
}

// 配置目录不存在时要能自动创建 —— 缺省位置在新机器上还没有。
func TestSave_CreatesMissingDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "T100", "tt", "config.json")
	if err := Save(path, map[string]any{"a": 1}); err != nil {
		t.Fatalf("Save 应当自动创建目录: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("文件未创建: %v", err)
	}
}

// LoadHosts 是读环境清单的入口（合并前 TDictCli/host.LoadHosts 的替代）。
// 它要求至少配了一个环境 —— CLI 命令都按"有环境可连"的前提工作。
//
// 合并前那两份 LoadHosts 测试里有一条断言"旧键 debug 也能读"，那条随旧结构一起
// 取消了：兼容性由 migrate.go 一次性做掉，读路径只认 hosts。
func TestLoadHosts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultConfigName)
	seed := `{
  "schemaVersion": 2,
  "hosts": {"sshs": [
    {"name": "a", "host": "1.1.1.1", "user": "u"},
    {"name": "b", "host": "2.2.2.2", "user": "u", "db": {"type": "oracle", "host": "d", "service": "t35prd"}}
  ]}
}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	h, err := LoadHosts(path)
	if err != nil {
		t.Fatalf("LoadHosts: %v", err)
	}
	if len(h.SSHs) != 2 {
		t.Fatalf("环境数 = %d, 期望 2", len(h.SSHs))
	}
	// 空名**不再**回退到任何环境:默认环境已移除,环境一律点名。
	if got := h.ByName(""); got != nil {
		t.Errorf("ByName(\"\") 不该取到环境(没有默认环境),得到 %+v", got)
	}
	if got := h.ByName("b"); got == nil {
		t.Error("ByName(\"b\") 应当取到 b")
	}
	// db 要能正常解析出来
	e := h.ByName("b")
	if e.DB == nil || e.DB.Type != "oracle" || e.DB.Svc() != "t35prd" {
		t.Errorf("db 未正确解析: %+v", e.DB)
	}
}

func TestLoadHosts_Errors(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// 没有 hosts 节
	if _, err := LoadHosts(write("a.json", `{"schemaVersion":2,"query":{"source":"local"}}`)); err == nil {
		t.Error("缺少 hosts 节应报错")
	}
	// 有节但一个环境都没有
	if _, err := LoadHosts(write("b.json", `{"schemaVersion":2,"hosts":{"sshs":[]}}`)); err == nil {
		t.Error("sshs 为空应报错")
	}
}

// 首次运行要能落一份骨架：配置页得有个文件可编辑。
func TestEnsureExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "config.json")

	if err := EnsureExists(path); err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}
	r, err := Load(path)
	if err != nil {
		t.Fatalf("骨架应当可被读回: %v", err)
	}
	if r.SchemaVersion != SchemaVersion {
		t.Errorf("骨架的 schemaVersion = %d, 期望 %d（写对了才不会被就地迁移）", r.SchemaVersion, SchemaVersion)
	}
	if r.Listen != DefaultListen {
		t.Errorf("骨架的 listen = %q, 期望 %q", r.Listen, DefaultListen)
	}
	if len(r.Hosts.SSHs) != 0 {
		t.Errorf("骨架不该预置环境: %+v", r.Hosts.SSHs)
	}

	// 已存在时原样不动
	if err := Save(path, map[string]any{"schemaVersion": SchemaVersion, "listen": "127.0.0.1:9999", "hosts": map[string]any{"sshs": []any{}}}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := EnsureExists(path); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("EnsureExists 覆盖了已存在的配置")
	}
}

// ByName 只认**显式名字**:没有默认环境,所以空名一律取不到东西。
// 这是"环境一律点名"的最后一道:任何地方想靠空名拿到一个环境都会被这里挡住。
func TestHostsByName(t *testing.T) {
	h := &Hosts{
		SSHs: []NamedSsh{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
	}
	if got := h.ByName("c"); got == nil || got.Name != "c" {
		t.Error("按名取环境失败")
	}
	if got := h.ByName(""); got != nil {
		t.Errorf("空名不该取到环境(没有默认环境,也不回退首条),得到 %+v", got)
	}
	if got := h.ByName("不存在"); got != nil {
		t.Error("未命中应返回 nil")
	}
}

// TDev 原来完全没有配置系统；这个节是新增的，缺省值必须能兜住。
func TestTdevSettingsDefault(t *testing.T) {
	var s TdevSettings
	if got := s.WorkspaceSuffixOrDefault(); got != DefaultWorkspaceSuffix {
		t.Errorf("缺省后缀 = %q, 期望 %q", got, DefaultWorkspaceSuffix)
	}
	s.WorkspaceSuffix = "-work"
	if got := s.WorkspaceSuffixOrDefault(); got != "-work" {
		t.Errorf("显式后缀 = %q", got)
	}
}

// net 节是可选节：写了的读出代理，没写的（或整节缺失的）取零值 —— 缺省就是"没配代理"。
// 这条同时钉住 schema 3 的读侧：老配置（schemaVersion 2、没有 net 节）必须照读不报错。
func TestNetSettingsProxy(t *testing.T) {
	dir := t.TempDir()

	full := filepath.Join(dir, "full.json")
	seed := `{"schemaVersion": 4, "net": {"proxy": "http://127.0.0.1:10808"}}`
	if err := os.WriteFile(full, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Load(full)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if r.Net.Proxy != "http://127.0.0.1:10808" {
		t.Errorf("net.proxy = %q，想要 http://127.0.0.1:10808", r.Net.Proxy)
	}
	if r.SchemaVersion != SchemaVersion {
		t.Errorf("schemaVersion = %d，想要 %d", r.SchemaVersion, SchemaVersion)
	}

	old := filepath.Join(dir, "old.json")
	if err := os.WriteFile(old, []byte(`{"schemaVersion": 2, "listen": "127.0.0.1:28670"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r2, err := Load(old)
	if err != nil {
		t.Fatalf("旧结构（没有 net 节）该照读，却报错: %v", err)
	}
	if r2.Net.Proxy != "" {
		t.Errorf("缺省的 net.proxy 该是空串，得到 %q", r2.Net.Proxy)
	}
}

// 工作区按**环境**存：往返不能丢，且要按名取到对应环境的那一份。
//
// 这条是"切了环境就该换工作区"的地基：机器级那份已经没有了（见 TzsSettings），所以一旦
// 环境级这份在读-改-写里丢掉一个，切换环境就会退回"未配置工作区"。
//
// 注：原先这里顺带断言 ActiveWorkspace 取的是 activeEnv 那一份。默认环境已移除
// （见 Hosts 的注释），所以现在直接按名取 —— 环境一律点名。
func TestPerEnvWorkspace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	seed := `{
  "schemaVersion": 3,
  "hosts": {
    "sshs": [
      {"name": "甲", "host": "h1", "port": 22, "user": "u", "password": "p", "workspace": "D:\\ws\\jia"},
      {"name": "乙", "host": "h2", "port": 22, "user": "u", "password": "p"}
    ]
  }
}`
	if err := os.WriteFile(path, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := r.Hosts.SSHs[0].Workspace; got != `D:\ws\jia` {
		t.Errorf("环境「甲」的工作区 = %q", got)
	}
	// 按名取：乙没配 → 空。**不回落**到别的环境（那是"拿别人的表单当草稿纸"）。
	if got := r.Hosts.ByName("乙").Workspace; got != "" {
		t.Errorf("环境「乙」没配工作区时该是空串，得到 %q", got)
	}
	if got := r.Hosts.ByName("甲").Workspace; got != `D:\ws\jia` {
		t.Errorf("按名取「甲」的工作区，得到 %q", got)
	}

	// 写回（整节替换）不能丢掉工作区：SaveHosts 走的是同一份类型化结构。
	if err := SaveHosts(path, r.Hosts); err != nil {
		t.Fatalf("SaveHosts: %v", err)
	}
	r2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := r2.Hosts.ByName("甲").Workspace; got != `D:\ws\jia` {
		t.Errorf("往返之后工作区丢了: %q", got)
	}
	// 机器级那一份不再存在：配置里不该有 tzs.workspace。
	root, _ := Open(path)
	if sec, ok := root["tzs"].(map[string]any); ok {
		if _, has := sec["workspace"]; has {
			t.Error("机器级的 tzs.workspace 又回来了 —— 工作区只该按环境配")
		}
	}
}

// skills 节：第三方技能登记是可选的，缺节读成空表；顺序必须原样保留（顺序即优先级）。
func TestSkillsExternals(t *testing.T) {
	dir := t.TempDir()

	empty := filepath.Join(dir, "empty.json")
	if err := os.WriteFile(empty, []byte(`{"schemaVersion": 4}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := Load(empty)
	if err != nil {
		t.Fatalf("缺 skills 节该照读: %v", err)
	}
	if len(r.Skills.Externals) != 0 || len(r.Skills.Paths()) != 0 {
		t.Errorf("缺省该是空表，得到 %+v", r.Skills)
	}

	full := filepath.Join(dir, "full.json")
	seed := `{"schemaVersion":4,"skills":{"externals":[
		{"path":"D:/my-skills/b","label":"b"},
		{"path":"D:/my-skills/a"},
		{"path":"  "}
	]}}`
	if err := os.WriteFile(full, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	r2, err := Load(full)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := r2.Skills.Paths()
	want := []string{`D:/my-skills/b`, `D:/my-skills/a`} // 空白路径被丢掉，顺序不变
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Paths() = %v, want %v", got, want)
	}
	if r2.Skills.Externals[0].Label != "b" {
		t.Errorf("label 没读出来: %+v", r2.Skills.Externals[0])
	}
}
