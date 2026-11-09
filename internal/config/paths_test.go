package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// isolateEnv 把配置发现相关的环境变量全部指到临时目录，让测试不受本机
// 真实存在的 %APPDATA% 内容影响。返回统一用户目录的**上级**（%APPDATA% 指向的临时目录，
// 即 UserConfigDir = <返回值>\tt 的父目录）。
func isolateEnv(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("AppData", home)
	default:
		t.Setenv("XDG_CONFIG_HOME", home)
	}
	t.Setenv("TT_CONFIG", "")
	return home
}

// 默认落点：%APPDATA%\tt —— 中间不再有 T100 一层。
func TestUserConfigDir(t *testing.T) {
	home := isolateEnv(t)

	want := filepath.Join(home, ToolDirName)
	if got := UserConfigDir(); got != want {
		t.Errorf("UserConfigDir() = %q, 期望 %q", got, want)
	}
	if got := UserConfigPath(); got != filepath.Join(want, DefaultConfigName) {
		t.Errorf("UserConfigPath() = %q", got)
	}
}

// TT_CONFIG 优先级最高。
func TestResolvePath_EnvVar(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "my.json")
	if err := os.WriteFile(path, []byte(`{"hosts":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	isolateEnv(t)
	t.Setenv("TT_CONFIG", path)

	got, err := ResolvePath("", false)
	if err != nil {
		t.Fatalf("ResolvePath: %v", err)
	}
	if got != path {
		t.Errorf("解析到 %q, 期望 %q", got, path)
	}
}

// --config 显式指定：绝对路径直接用，相对路径按 exe 目录 / 当前目录找。
func TestResolvePath_ExplicitFlag(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	isolateEnv(t)

	rel := "custom.json"
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(`{"hosts":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ResolvePath(rel, false)
	if err != nil {
		t.Fatalf("相对路径未按当前目录解析: %v", err)
	}
	if filepath.Base(got) != rel {
		t.Errorf("解析到 %q", got)
	}
}

// 全部落空时报错，错误里要含尝试过的路径与设置环境变量的提示。
func TestResolvePath_NoneFound(t *testing.T) {
	isolateEnv(t)

	_, err := ResolvePath("", false)
	if err == nil {
		t.Fatal("找不到配置时应当报错")
	}
	msg := err.Error()
	for _, want := range []string{"配置文件未找到", "TT_CONFIG", "尝试了以下路径"} {
		if !contains(msg, want) {
			t.Errorf("错误信息缺少 %q:\n%s", want, msg)
		}
	}

	// allowMissing 用于写路径：首次保存要能拿到落点而不是报错
	got, err := ResolvePath("", true)
	if err != nil {
		t.Fatalf("allowMissing=true 不应报错: %v", err)
	}
	if got != UserConfigPath() {
		t.Errorf("落点 = %q, 期望 %q", got, UserConfigPath())
	}
}

// 旧位置兜底已经移除：哪怕当前目录、旧统一目录（%APPDATA%\T100\tt）、
// 旧工具目录（tdebug / tdict）里摆着**长得完全像本工具配置**的诱饵，
// 解析也不许捡走任何一份 —— 数据统一在 %APPDATA%\tt，不向后兼容。
func TestResolvePath_NoLegacyFallback(t *testing.T) {
	home := isolateEnv(t)

	seed := `{"schemaVersion":2,"hosts":{"activeEnv":"诱饵","sshs":[{"name":"诱饵","host":"h","user":"u"}]}}`
	legacyDirs := []string{
		"T100\\tt",     // 旧版默认落点（中间多一层 T100）
		"T100\\tdebug", // 合并前的调试工具
		"T100\\tdict",  // 合并前的字典工具
		"TDebug",       // 更早的桌面版目录
	}
	for _, sub := range legacyDirs {
		p := filepath.Join(home, sub, DefaultConfigName)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(seed), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// 当前目录的 config.json 同样是诱饵
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	workdir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workdir, DefaultConfigName), []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workdir); err != nil {
		t.Fatal(err)
	}

	// 读路径：统一目录（%APPDATA%\tt）没有文件 → 必须报错，而不是捡走诱饵
	if got, err := ResolvePath("", false); err == nil {
		t.Fatalf("旧位置兜底被捡走了: %q", got)
	}
	// 写路径：落点必须是 %APPDATA%\tt\config.json，而不是任何诱饵
	got, err := ResolvePath("", true)
	if err != nil {
		t.Fatalf("allowMissing=true 不应报错: %v", err)
	}
	if want := filepath.Join(home, ToolDirName, DefaultConfigName); got != want {
		t.Errorf("落点 = %q, 期望 %q", got, want)
	}
}

// 显式指定的路径即使文件还不存在，也必须原样用 —— 那是"在这里新建"，不是"没找到"。
// 写偏了会让便携版把配置写到用户目录，也会让 --config <临时目录> 的调用落在别处。
func TestResolvePath_ExplicitWinsWhenMissing(t *testing.T) {
	isolateEnv(t)

	dir := t.TempDir()
	want := filepath.Join(dir, "sub", "config.json")

	got, err := ResolvePath(want, true)
	if err != nil {
		t.Fatalf("allowMissing 时不该报错: %v", err)
	}
	if got != want {
		t.Errorf("落点 = %q, 期望用户显式指定的 %q", got, want)
	}
	if got == DefaultConfigPath() {
		t.Error("退回了默认落点 —— 显式路径被忽略")
	}
}

// 环境变量指定的路径同理（TT_CONFIG 指向一个还不存在的文件）。
func TestResolvePath_EnvWinsWhenMissing(t *testing.T) {
	isolateEnv(t)

	want := filepath.Join(t.TempDir(), "from-env.json")
	t.Setenv("TT_CONFIG", want)

	got, err := ResolvePath("", true)
	if err != nil {
		t.Fatalf("allowMissing 时不该报错: %v", err)
	}
	if got != want {
		t.Errorf("落点 = %q, 期望 %q", got, want)
	}
}

// ---------- 统一路径管理器（locations.go） ----------

// CacheDir 只认缓存清单里的名字：清单外的名字得到空串，不许悄悄建新目录。
func TestCacheDir_ValidatesName(t *testing.T) {
	dir := t.TempDir()
	if got := CacheDir(dir, "ents"); got != filepath.Join(dir, "ents") {
		t.Errorf("CacheDir(ents) = %q", got)
	}
	if got := CacheDir(dir, "not-in-list"); got != "" {
		t.Errorf("清单外的名字应当得到空串, 得到 %q", got)
	}
	if got := CacheDir("", "ents"); got != "" {
		t.Errorf("数据目录为空应当得到空串, 得到 %q", got)
	}
}

// Locations 从配置路径派生：数据目录、状态文件、日志。
func TestLocations_DataDir(t *testing.T) {
	l := LocationsAt(filepath.Join(t.TempDir(), "tt", "config.json"))
	want := filepath.Dir(l.ConfigPath)
	if l.DataDir() != want {
		t.Errorf("DataDir = %q, 期望 %q", l.DataDir(), want)
	}
	if l.StateFile() != filepath.Join(want, ".tt-serve.json") {
		t.Errorf("StateFile = %q", l.StateFile())
	}
	if l.LogFile() != filepath.Join(want, ".tt-serve.log") {
		t.Errorf("LogFile = %q", l.LogFile())
	}
	if l.CacheDir("spill") != filepath.Join(want, "spill") {
		t.Errorf("CacheDir(spill) = %q", l.CacheDir("spill"))
	}
	if l.CacheDir("config.json") != "" {
		t.Error("CacheDir 对清单外的名字应当返回空串")
	}
}

// 字典库读侧：TDICT_DB → -d → 数据目录，取第一个存在的；全不存在报错。
func TestLocations_ResolveDictDB(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "real.db")
	if err := os.WriteFile(existing, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	l := LocationsAt(filepath.Join(dir, "tt", "config.json"))

	// TDICT_DB 最高优先级
	t.Setenv("TDICT_DB", existing)
	if got, _, err := l.ResolveDictDB(""); err != nil || got != existing {
		t.Errorf("TDICT_DB 未生效: got=%q err=%v", got, err)
	}

	// -d 次之（相对路径按当前目录绝对化）
	rel := "flag.db"
	if err := os.WriteFile(filepath.Join(dir, rel), []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TDICT_DB", "")
	if got, _, err := l.ResolveDictDB(rel); err != nil || got != filepath.Join(dir, rel) {
		t.Errorf("-d 未按当前目录解析: got=%q err=%v", got, err)
	}

	// 都没有 → 数据目录下的缺省名（这里不存在 → 报错并列出全部尝试）
	if _, tried, err := l.ResolveDictDB(""); err == nil {
		t.Error("数据目录没有库时应当报错")
	} else if len(tried) == 0 {
		t.Error("报错时应当列出尝试过的路径")
	} else {
		want := filepath.Join(filepath.Dir(l.ConfigPath), "erp_data.db")
		if tried[len(tried)-1] != want {
			t.Errorf("最后一个候选应当是数据目录缺省名: %q, 期望 %q", tried[len(tried)-1], want)
		}
	}
}

// 字典库写侧（SyncTarget）：sync.target 优先 → 已存在的库 → 绝对 -d → 数据目录缺省。
func TestLocations_SyncTarget(t *testing.T) {
	t.Setenv("TDICT_DB", "") // 隔离开发机上的真实 TDICT_DB,否则缺省目标会变成它
	dir := t.TempDir()
	l := LocationsAt(filepath.Join(dir, "tt", "config.json"))
	dataDir := filepath.Dir(l.ConfigPath)

	// 1. 配置值优先，存在与否都一样
	cfgured := filepath.Join(dir, "configured.db")
	if got := l.SyncTarget(cfgured, ""); got != cfgured {
		t.Errorf("sync.target 未生效: %q", got)
	}

	// 4. 什么都不给 → 数据目录缺省名
	if got := l.SyncTarget("", ""); got != filepath.Join(dataDir, "erp_data.db") {
		t.Errorf("缺省目标 = %q", got)
	}

	// 2. 已存在的 -d 目标优先于缺省落点
	flagDB := filepath.Join(dir, "flag.db")
	if err := os.WriteFile(flagDB, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TDICT_DB", "")
	if got := l.SyncTarget("", flagDB); got != flagDB {
		t.Errorf("已存在的 -d 库未生效: %q", got)
	}

	// 3. 不存在的绝对 -d 也写那里
	absent := filepath.Join(dir, "absent.db")
	if got := l.SyncTarget("", absent); got != absent {
		t.Errorf("绝对 -d 未生效: %q", got)
	}
}

// 引擎 exe：覆盖优先，否则 <tt.exe 目录>\tzs\tzs-server.exe。
func TestEngineExe(t *testing.T) {
	override := filepath.Join(t.TempDir(), "custom-tzs-server.exe")
	if got := EngineExe(override); got != override {
		t.Errorf("覆盖未生效: %q", got)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("拿不到测试二进制路径: %v", err)
	}
	want := filepath.Join(filepath.Dir(self), "tzs", "tzs-server.exe")
	if got := EngineExe(""); got != want {
		t.Errorf("缺省引擎路径 = %q, 期望 %q", got, want)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
