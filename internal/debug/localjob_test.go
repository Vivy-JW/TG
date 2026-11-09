package debug

// 本地优先作业解析的回归测试（2026-10-23）。
//
// 背景：用户问"点启动为什么要 3-5 秒 / 本地有 db 为什么不用"。这三条用例钉住：
//  1. 本地库文件本身能查出与远程 ksql 相同口径的结果（含 $FGLRUN 前缀剥离、大小写）；
//  2. 本地命中时**不拨 SSH** —— 用一台必然连不上的假主机验证（拨了就报错）；
//  3. 本地未命中时照旧走远程链路（回退不能把启动弄坏）。

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
	"tt/internal/dbconfig"
)

// makeJobDB 造一份最小可用的本地字典库（gzzz_t JOIN gzza_t 的两张表）。
func makeJobDB(t *testing.T, dir string) string {
	t.Helper()
	dbPath := filepath.Join(dir, "鼎捷.db")
	d, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("打开临时库: %v", err)
	}
	defer d.Close()
	for _, ddl := range []string{
		`create table gzzz_t (gzzz001 text, gzzz002 text, gzzz004 text, gzzz005 text)`,
		`create table gzza_t (gzza001 text, gzza004 text)`,
		`insert into gzza_t values ('axmt500', '$FGLRUN $AXMi/axmt500')`,
		`insert into gzzz_t values ('apmt500', 'axmt500', '1', 'AXM')`,
	} {
		if _, err := d.Exec(ddl); err != nil {
			t.Fatalf("建表/插数: %v", err)
		}
	}
	return dbPath
}

func TestLookupJobLocalDB(t *testing.T) {
	dbPath := makeJobDB(t, t.TempDir())

	jr, ok := lookupJobLocalDB(dbPath, "apmt500")
	if !ok {
		t.Fatal("本地库里有这条作业，应当命中")
	}
	if jr.Prog != "axmt500" {
		t.Fatalf("实体程序: got %q want axmt500", jr.Prog)
	}
	if jr.Module != "AXM" {
		t.Fatalf("模块: got %q want AXM", jr.Module)
	}
	if jr.LaunchRef != "$AXMi/axmt500" || jr.Extra != "1" {
		t.Fatalf("启动引用应剥掉 $FGLRUN 前缀: got ref=%q extra=%q", jr.LaunchRef, jr.Extra)
	}

	// 大小写不敏感
	if _, ok := lookupJobLocalDB(dbPath, "APMT500"); !ok {
		t.Fatal("作业号大小写应当不敏感")
	}
	// 不存在的作业
	if _, ok := lookupJobLocalDB(dbPath, "nope999"); ok {
		t.Fatal("不存在的作业不该命中")
	}
	// 库文件不存在
	if _, ok := lookupJobLocalDB(filepath.Join("Z:", "none.db"), "apmt500"); ok {
		t.Fatal("库文件不存在不该命中")
	}
}

func TestResolveJobPrefersLocalLookup(t *testing.T) {
	m := NewManager(cfgFor("unreachable.invalid", "36")) // 假主机：真去拨 SSH 必失败
	hit := false
	m.LocalJobLookup = func(env, job string) (JobResolve, bool) {
		if env == "" {
			t.Fatal("env 名不该为空：本地库按环境选")
		}
		hit = true
		return JobResolve{Prog: "axmt500", Extra: "1", Module: "AXM", LaunchRef: "$FGLRUN $AXMi/axmt500"}, true
	}
	cfg := cfgFor("unreachable.invalid", "36")
	cfg.DB = &dbconfig.Connection{Type: "kingbase"} // 非 nil 才会进解析链
	mod, prog, ref, extra, err := m.resolveJobWith(cfg, "", "apmt500")
	if err != nil {
		t.Fatalf("本地命中时不应触网: %v", err)
	}
	if !hit {
		t.Fatal("本地查找没被调用")
	}
	if prog != "axmt500" || mod != "axm" || ref != "$AXMi/axmt500" || extra != "1" {
		t.Fatalf("解析结果不符: mod=%q prog=%q ref=%q extra=%q", mod, prog, ref, extra)
	}
}

func TestResolveJobFallsBackWhenLocalMiss(t *testing.T) {
	m := NewManager(cfgFor("unreachable.invalid", "36"))
	m.LocalJobLookup = func(env, job string) (JobResolve, bool) { return JobResolve{}, false }
	cfg := cfgFor("unreachable.invalid", "36")
	cfg.DB = &dbconfig.Connection{Type: "kingbase"}
	_, _, _, _, err := m.resolveJobWith(cfg, "", "apmt500")
	if err == nil || !strings.Contains(err.Error(), "SSH 连接失败") {
		t.Fatalf("本地未命中应当回退远程链路（这台假主机上会报 SSH 连接失败）: %v", err)
	}
}
