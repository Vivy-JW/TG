package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildProgDB 建一个只含"程序 / 程序用表 / 表档"三张表的临时库，
// 列与 tt gp sync 产出的 t_* 同构（全 TEXT，与本地镜像一致）。
func buildProgDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prog.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		// 程序登记（GP 的 gaz_file + zz_file + gal_file 落到一张表）
		`CREATE TABLE t_prog (prog TEXT PRIMARY KEY, name TEXT, module TEXT, memo TEXT,
		   name_tw TEXT, category TEXT, run_cmd TEXT)`,
		`INSERT INTO t_prog VALUES ('apmt420','请购单维护作业','APM','维护请购单资料',
		   '請購單維護作業','I','$FGLRUN $APMi/apmt420')`,
		`INSERT INTO t_prog VALUES ('aimq421','请购单查询','AIM','',
		   '請購單查詢','Q','$FGLRUN $AIMi/aimq421')`,
		`INSERT INTO t_prog VALUES ('aapf104','入库金额明细表打印','AAP','',
		   '', 'R','')`,
		// 程序用表（zr_file：一个 (程序,表) 多行操作码）
		`CREATE TABLE t_prog_table (prog TEXT, tab TEXT, ops TEXT, PRIMARY KEY (prog, tab))`,
		`INSERT INTO t_prog_table VALUES ('apmt420','ima_file','S/I/U/D')`,
		`INSERT INTO t_prog_table VALUES ('apmt420','ebolog_file','S')`,
		`INSERT INTO t_prog_table VALUES ('apmt420','odm_file','S/U')`,
		`INSERT INTO t_prog_table VALUES ('aimq421','ima_file','S')`,
		// 表档（gp 的 tab 说明：odm_file 故意不在表档里，验 LEFT JOIN）
		`CREATE TABLE t_table (tab TEXT, tab_cn TEXT, module TEXT, tab_type TEXT, col_count TEXT)`,
		`INSERT INTO t_table VALUES ('ima_file','料件主档','INV','1','40')`,
		`INSERT INTO t_table VALUES ('ebolog_file','异动日志档','INV','1','12')`,
	}
	for _, s := range stmts {
		if _, err := conn.Exec(s); err != nil {
			t.Fatalf("setup (%s): %v", s, err)
		}
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close setup conn: %v", err)
	}

	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

// TestQueryProgInfo 程序登记的三个来源：名称/说明、类别/运行指令、模块。
func TestQueryProgInfo(t *testing.T) {
	d := buildProgDB(t)

	info, err := d.QueryProgInfo("apmt420", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgInfo(apmt420): %v", err)
	}
	if info == nil || !info.IsProg {
		t.Fatalf("apmt420 应登记为程序: %+v", info)
	}
	if info.Name != "请购单维护作业" || info.Module != "APM" || info.Category != "I" {
		t.Errorf("字段不符: %+v", info)
	}
	if info.CategoryName != "建档维护" {
		t.Errorf("类别名应为 GP 口径: %+v", info.CategoryName)
	}
	if info.RunCmd != "$FGLRUN $APMi/apmt420" || info.Memo != "维护请购单资料" {
		t.Errorf("运行指令/说明不符: %+v", info)
	}
	if info.TableCount != 3 {
		t.Errorf("用表数应为 3: %+v", info)
	}

	// 繁体名称
	tw, err := d.QueryProgInfo("apmt420", "zh_TW")
	if err != nil {
		t.Fatalf("QueryProgInfo(zh_TW): %v", err)
	}
	if tw == nil || tw.Name != "請購單維護作業" {
		t.Errorf("zh_TW 名称不符: %+v", tw)
	}

	// 没有繁体名时退回简体
	noTW, err := d.QueryProgInfo("aapf104", "zh_TW")
	if err != nil {
		t.Fatalf("QueryProgInfo(aapf104,zh_TW): %v", err)
	}
	if noTW == nil || noTW.Name != "入库金额明细表打印" {
		t.Errorf("无繁体名应退回简体: %+v", noTW)
	}

	// 未收录
	if none, err := d.QueryProgInfo("nosuch01", "zh_CN"); err != nil || none != nil {
		t.Errorf("未收录应返回 (nil,nil): %+v, %v", none, err)
	}
}

// TestQueryProgJobs GP 没有独立作业实体：作业 = 程序，恒 1 条。
func TestQueryProgJobs(t *testing.T) {
	d := buildProgDB(t)

	jobs, err := d.QueryProgJobs("apmt420", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgJobs: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("GP 的作业恒 1 条, got %d: %+v", len(jobs), jobs)
	}
	if jobs[0].JobCode != "apmt420" || jobs[0].JobName != "请购单维护作业" || jobs[0].Module != "APM" {
		t.Errorf("作业应等于程序本身: %+v", jobs[0])
	}

	// 未收录：空数组（不是 nil）
	if none, err := d.QueryProgJobs("nosuch01", "zh_CN"); err != nil || none == nil || len(none) != 0 {
		t.Errorf("未收录应返回空切片: %#v, %v", none, err)
	}
}

// TestQueryProgTables 程序 → 表：按表名升序，表档没收录的表也照样出（LEFT JOIN）。
func TestQueryProgTables(t *testing.T) {
	d := buildProgDB(t)

	rows, err := d.QueryProgTables("apmt420", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgTables: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("apmt420 应命中 3 张表, got %d: %+v", len(rows), rows)
	}
	if rows[0].Table != "ebolog_file" || rows[0].Ops != "S" {
		t.Errorf("第一行应为 ebolog_file/S: %+v", rows[0])
	}
	if rows[1].Table != "ima_file" || rows[1].Ops != "S/I/U/D" {
		t.Errorf("ima_file 操作应为 S/I/U/D: %+v", rows[1])
	}
	if rows[1].TableDesc != "料件主档" {
		t.Errorf("表说明应取 t_table: %+v", rows[1])
	}
	if rows[2].Table != "odm_file" || rows[2].TableDesc != "" {
		t.Errorf("表档未收录时说明为空、表名照出: %+v", rows[2])
	}

	if none, err := d.QueryProgTables("nosuch", "zh_CN"); err != nil || len(none) != 0 {
		t.Errorf("未登记者应返回空: %+v, %v", none, err)
	}
}

// TestQueryTablePrograms 表 → 程序：反查哪些程序在用这张表。
func TestQueryTablePrograms(t *testing.T) {
	d := buildProgDB(t)

	rows, err := d.QueryTablePrograms("ima_file", "zh_CN")
	if err != nil {
		t.Fatalf("QueryTablePrograms: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("ima_file 应被 2 个程序使用, got %d: %+v", len(rows), rows)
	}
	if rows[0].Prog != "aimq421" || rows[0].Ops != "S" {
		t.Errorf("aimq421 行不符: %+v", rows[0])
	}
	if rows[1].Prog != "apmt420" || rows[1].Ops != "S/I/U/D" {
		t.Errorf("apmt420 行不符: %+v", rows[1])
	}

	if none, err := d.QueryTablePrograms("nosuch_t", "zh_CN"); err != nil || len(none) != 0 {
		t.Errorf("没人用的表应返回空: %+v, %v", none, err)
	}
}

// TestMergeOps 操作类别合并：固定顺序 S/I/U/D，未知码按字母序追加。
func TestMergeOps(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{[]string{"D", "I", "U", "S"}, "S/I/U/D"},
		{[]string{"U", "S"}, "S/U"},
		{[]string{"S", "S"}, "S"},
		{[]string{"Z", "S"}, "S/Z"},
		{nil, ""},
	}
	for _, c := range cases {
		if got := mergeOps(c.in); got != c.want {
			t.Errorf("mergeOps(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestQueryProgList 列表：类别名、作业数恒 1，--kw 按编号或名称过滤。
func TestQueryProgList(t *testing.T) {
	d := buildProgDB(t)

	all, err := d.QueryProgList("zh_CN", "")
	if err != nil {
		t.Fatalf("QueryProgList: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("应有 3 个程序, got %d", len(all))
	}
	if all[0].Code != "aapf104" || all[0].Category != "R" || all[0].CategoryName != "报表类" {
		t.Errorf("aapf104 行不符: %+v", all[0])
	}
	if all[2].Code != "apmt420" || all[2].JobCount != 1 {
		t.Errorf("作业数恒 1: %+v", all[2])
	}

	byName, err := d.QueryProgList("zh_CN", "请购")
	if err != nil {
		t.Fatalf("QueryProgList(按名称): %v", err)
	}
	if len(byName) != 2 {
		t.Errorf("按名称搜索应命中 2 个: %+v", byName)
	}

	byCode, err := d.QueryProgList("zh_CN", "aimq")
	if err != nil {
		t.Fatalf("QueryProgList(按编号): %v", err)
	}
	if len(byCode) != 1 || byCode[0].Code != "aimq421" {
		t.Errorf("按编号搜索应命中 aimq421: %+v", byCode)
	}

	if none, err := d.QueryProgList("zh_CN", "无此程序"); err != nil || len(none) != 0 {
		t.Errorf("无匹配应返回空: %+v, %v", none, err)
	}
}

// TestCategoryName GP 的程序类别码表（来源 p_zz.sdd 的 zz03 定义）。
func TestCategoryName(t *testing.T) {
	cases := map[string]string{
		"M": "目录程式", "I": "建档维护", "U": "公用程式", "P": "处理类",
		"R": "报表类", "Q": "查询", "T": "异动单据", "S": "设定", "O": "其他",
		"m": "目录程式", "": "", "Z": "",
	}
	for in, want := range cases {
		if got := CategoryName(in); got != want {
			t.Errorf("CategoryName(%q) = %q, want %q", in, got, want)
		}
	}
}
