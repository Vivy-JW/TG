package dict

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/config"

	_ "modernc.org/sqlite"
)

// 每个环境必须落到自己的 .db；同步/查询都不该再碰共享的 erp_data.db。
func TestLocalDBTargetsArePerEnv(t *testing.T) {
	resetDictPackageState(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	body := `{"schemaVersion":2,"hosts":{"sshs":[
      {"name":"主机109","host":"1.2.3.4","user":"u","password":"p"},
      {"name":"尚阳通","host":"5.6.7.8","user":"u","password":"p"}]}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	common.ConfigPath = cfgPath
	srcConfigPath = cfgPath
	t.Setenv("TDICT_DB", "")
	dbPath = ""

	a := dictDBSyncTarget("主机109")
	b := dictDBSyncTarget("尚阳通")
	if a == b {
		t.Fatalf("两个环境指向同一个库: %s", a)
	}
	if filepath.Dir(a) != filepath.Join(dir, "dict") || filepath.Dir(b) != filepath.Join(dir, "dict") {
		t.Fatalf("库应在 <数据目录>/dict 下: %s / %s", a, b)
	}
	if filepath.Base(a) == "erp_data.db" || filepath.Base(b) == "erp_data.db" {
		t.Fatal("不该回落到旧的共享库名")
	}
}

// 本地源必须点名环境：openLocalSource("主机109") 只读该环境的文件。
func TestLocalSourceUsesNamedEnvDB(t *testing.T) {
	resetDictPackageState(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	body := `{"schemaVersion":2,"hosts":{"sshs":[
      {"name":"主机109","host":"1.2.3.4","user":"u","password":"p"}]}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	common.ConfigPath = cfgPath
	srcConfigPath = cfgPath
	t.Setenv("TDICT_DB", "")
	dbPath = ""

	want := config.DictDBPath(dir, "主机109")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	makeLocalDict(t, want)

	if err := openLocalSource("主机109"); err != nil {
		t.Fatalf("按环境打开本地库: %v", err)
	}
	defer closeQuerySource()
	if srcTarget == nil || srcTarget.Path != want || srcTarget.Env != "主机109" {
		t.Fatalf("srcTarget = %+v, want path=%s env=主机109", srcTarget, want)
	}
}

// 裸 local（没有环境、也没有 -d/TDICT_DB）必须报错并给出 local:<环境> 的写法。
func TestBareLocalNeedsEnv(t *testing.T) {
	resetDictPackageState(t)
	t.Setenv("TDICT_DB", "")
	dbPath = ""
	err := openLocalSource("")
	if err == nil {
		t.Fatal("裸 local 不该猜环境")
	}
	if !strings.Contains(err.Error(), "local:<环境>") {
		t.Errorf("报错应给出 local:<环境> 的写法: %v", err)
	}
}

// --env local:<环境> 必须解析成该环境的本地库（而不是回落到共享库）。
func TestOpenQuerySourceParsesLocalEnv(t *testing.T) {
	resetDictPackageState(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	body := `{"schemaVersion":2,"hosts":{"sshs":[
      {"name":"主机109","host":"1.2.3.4","user":"u","password":"p"}]}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	common.ConfigPath = cfgPath
	srcConfigPath = cfgPath
	t.Setenv("TDICT_DB", "")
	dbPath = ""

	want := config.DictDBPath(dir, "主机109")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	makeLocalDict(t, want)

	common.Env = "local:主机109"
	if err := openQuerySource(); err != nil {
		t.Fatalf("openQuerySource: %v", err)
	}
	defer closeQuerySource()
	if !srcLocal || srcTarget == nil || srcTarget.Path != want {
		t.Fatalf("local source = %v target=%+v, want path=%s", srcLocal, srcTarget, want)
	}
}

// query.source = local:<环境> 也能选中该环境的本地库。
func TestQuerySourceConfigLocalEnv(t *testing.T) {
	resetDictPackageState(t)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	body := `{"schemaVersion":2,"query":{"source":"local:主机109"},"hosts":{"sshs":[
      {"name":"主机109","host":"1.2.3.4","user":"u","password":"p"}]}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	common.ConfigPath = cfgPath
	srcConfigPath = cfgPath
	t.Setenv("TDICT_DB", "")
	dbPath = ""

	want := config.DictDBPath(dir, "主机109")
	if err := os.MkdirAll(filepath.Dir(want), 0o755); err != nil {
		t.Fatal(err)
	}
	makeLocalDict(t, want)

	common.Env = ""
	if err := openQuerySource(); err != nil {
		t.Fatalf("openQuerySource: %v", err)
	}
	defer closeQuerySource()
	if srcTarget == nil || srcTarget.Path != want || srcTarget.Env != "主机109" {
		t.Fatalf("target = %+v, want path=%s env=主机109", srcTarget, want)
	}
}

// makeLocalDict 建一张最小可用的字典库，供本地源测试打开。
func makeLocalDict(t *testing.T, path string) {
	t.Helper()
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("建本地字典库失败：%v", err)
	}
	defer conn.Close()
	for _, stmt := range []string{
		`CREATE TABLE dzeb_t (dzeb001 TEXT, dzeb002 TEXT, dzeb003 TEXT, dzeb004 TEXT,
			dzeb005 TEXT, dzeb007 TEXT, dzeb008 TEXT, dzeb021 TEXT, dzeb024 TEXT)`,
		`CREATE TABLE dzebl_t (dzebl001 TEXT, dzebl002 TEXT, dzebl003 TEXT)`,
		`INSERT INTO dzeb_t (dzeb001, dzeb002, dzeb003, dzeb021)
			VALUES ('demo_t', 'demo001', '演示栏位', '1')`,
	} {
		if _, err := conn.Exec(stmt); err != nil {
			t.Fatalf("建本地字典库失败（%s）：%v", stmt, err)
		}
	}
}
