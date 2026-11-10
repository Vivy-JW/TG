package dict

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"tt/internal/cli/common"
	"tt/internal/dict/db"
)

// gpConfig 造一份只有一个 GP 环境的 config.json，字典库目录指向 dir。
func gpConfig(t *testing.T, dir string) string {
	t.Helper()
	esc := strings.ReplaceAll(dir, `\`, `\\`)
	return writeConfig(t, `{
  "hosts": {
    "sshs": [
      {
        "name": "TOPTEST",
        "product": "gp",
        "instance": "toptest",
        "host": "10.0.0.1",
        "port": 22,
        "user": "u",
        "dictDir": "`+esc+`",
        "db": {"type": "oracle", "host": "10.0.0.1", "port": 1521, "service": "toptest",
               "accounts": [{"account": "ds", "password": "x"}]}
      }
    ]
  }
}`)
}

// makeGPDict 在 dir 下造一个最小的 <环境>.db（tt gp sync 产出的那几张表）。
func makeGPDict(t *testing.T, dir, env string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, env+".db")
	conn, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()
	for _, stmt := range []string{
		`CREATE TABLE t_prog(prog TEXT PRIMARY KEY, name TEXT, module TEXT, memo TEXT,
			name_tw TEXT, category TEXT, run_cmd TEXT)`,
		`CREATE TABLE t_table(tab TEXT PRIMARY KEY, tab_cn TEXT, module TEXT, tab_type TEXT, col_count TEXT)`,
		`CREATE TABLE t_column(tab TEXT, seq TEXT, col TEXT, col_cn TEXT, dtype TEXT, dlen TEXT,
			dprec TEXT, dscale TEXT, nullable TEXT, dflt TEXT, remark TEXT, fmt TEXT)`,
		`CREATE TABLE t_prog_table(prog TEXT, tab TEXT, ops TEXT, PRIMARY KEY(prog,tab))`,
		`INSERT INTO t_prog VALUES('apmt420','请购单维护作业','apm','','請購單維護作業','T','$FGLRUN $APMi/apmt420')`,
		`INSERT INTO t_table VALUES('ima_file','料件主档','inv','1','314')`,
		`INSERT INTO t_prog_table VALUES('apmt420','ima_file','S/U/D')`,
	} {
		if _, err := conn.Exec(stmt); err != nil {
			t.Fatalf("seed %q: %v", stmt, err)
		}
	}
	return p
}

// TestLocalGPEnvReadsDict：local:<GP 环境> 落到 tt gp sync 产出的本地库，
// 不需要企业（ENT）、不连远程库。
func TestLocalGPEnvReadsDict(t *testing.T) {
	resetDictPackageState(t)

	dir := t.TempDir()
	// 打开的是真文件句柄：必须在 TempDir 清理前关掉（后注册的先跑），否则 Windows 删不掉。
	t.Cleanup(closeQuerySource)
	makeGPDict(t, dir, "TOPTEST")
	gpConfig(t, dir)

	common.Env = "local:TOPTEST"
	if err := openQuerySource(); err != nil {
		t.Fatalf("openQuerySource: %v", err)
	}
	if srcProduct != "gp" {
		t.Errorf("srcProduct = %q，期望 gp", srcProduct)
	}
	if !srcLocal {
		t.Error("本地字典库，srcLocal 应为 true")
	}
	if srcTarget == nil || srcTarget.Env != "TOPTEST" {
		t.Errorf("srcTarget = %+v，期望点名 TOPTEST", srcTarget)
	}

	info, err := GetDB().QueryProgInfo("apmt420", "zh_CN")
	if err != nil || info == nil {
		t.Fatalf("QueryProgInfo = (%v, %v)，期望命中", info, err)
	}
	if info.CategoryName != "异动单据" {
		t.Errorf("CategoryName = %q，GP 的 T 是异动单据（不是 T100 的交易处理）", info.CategoryName)
	}
	if info.TableCount != 1 {
		t.Errorf("TableCount = %d，期望 1", info.TableCount)
	}
	// 校验带值这一族 GP 没有：要么空集、要么"缺表"（命令层据此说 GP 没有这类数据）。
	if rows, err := GetDB().QueryCheckList("zh_CN", ""); err != nil && !db.IsMissingTable(err) {
		t.Errorf("GP 没有校验带值族，应 空集 或 缺表，得到 (%d, %v)", len(rows), err)
	}
}

// TestLocalGPEnvMissingDict：还没同步过要直接说清楚怎么补，而不是
// 让它去查一个空库、最后报"程序不存在"。
func TestLocalGPEnvMissingDict(t *testing.T) {
	resetDictPackageState(t)

	dir := t.TempDir() // 有目录、没有 .db
	gpConfig(t, dir)

	common.Env = "local:TOPTEST"
	err := openQuerySource()
	if err == nil {
		t.Fatal("字典库不存在时应报错")
	}
	msg := err.Error()
	if !strings.Contains(msg, "tt dict db sync") {
		t.Errorf("报错里应给出补救命令 tt dict db sync，实际: %s", msg)
	}
	if !strings.Contains(msg, "TOPTEST") {
		t.Errorf("报错里应点名环境，实际: %s", msg)
	}
}

// TestRemoteGPEnvIsRemote：不带 local: 前缀 = 直查远程库，不再走 T100 的
// "环境@企业 → entdir 换账号"（GP 没有 ENT）。这里只验路由形态：
// 因为测试环境连不上 10.0.0.1，它必须报"打开远程数据源"，而不是企业解析失败。
func TestRemoteGPEnvIsRemote(t *testing.T) {
	resetDictPackageState(t)
	t.Cleanup(closeQuerySource)

	dir := t.TempDir()
	gpConfig(t, dir)

	common.Env = "TOPTEST"
	err := openQuerySource()
	if err == nil {
		t.Fatal("连不上远程库时应报错")
	}
	msg := err.Error()
	if !strings.Contains(msg, "打开远程数据源") {
		t.Errorf("应走远程直查路径，实际: %s", msg)
	}
	if strings.Contains(msg, "企业") {
		t.Errorf("GP 没有企业概念，报错不该提企业: %s", msg)
	}
}
