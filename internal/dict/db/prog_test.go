package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildProgDB 建一个只含"程序与作业"四张表的临时库,列全 TEXT(与本地镜像同构)。
func buildProgDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "prog.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE gzza_t (gzza001 TEXT, gzza002 TEXT, gzza003 TEXT, gzza004 TEXT,
		   gzza008 TEXT, gzza011 TEXT, gzzastus TEXT)`,
		`CREATE TABLE gzzal_t (gzzal001 TEXT, gzzal002 TEXT, gzzal003 TEXT, gzzal005 TEXT)`,
		`CREATE TABLE gzzz_t (gzzz001 TEXT, gzzz002 TEXT, gzzz003 TEXT, gzzz004 TEXT, gzzz005 TEXT,
		   gzzz006 TEXT, gzzzstus TEXT)`,
		`CREATE TABLE gzzk_t (gzzk001 TEXT, gzzk002 TEXT, gzzk003 TEXT)`,
		// 程序↔表格(参考作业 azzq902):主键 程序编号+表格编号+操作类别
		`CREATE TABLE gzdg_t (gzdg001 TEXT, gzdg002 TEXT, gzdg003 TEXT)`,
		// 子程序/元件登记(参考作业 azzi901):与 gzza_t(主程序)互不重叠
		`CREATE TABLE gzde_t (gzde001 TEXT, gzde002 TEXT, gzde003 TEXT, gzde005 TEXT,
		   gzde006 TEXT, gzde008 TEXT, gzde009 TEXT, gzdestus TEXT)`,
		`CREATE TABLE gzdel_t (gzdel001 TEXT, gzdel002 TEXT, gzdel003 TEXT)`,
		`INSERT INTO gzde_t VALUES ('aapq110_01','AAP','S','Q','','s','sd','Y')`,
		`INSERT INTO gzde_t VALUES ('cl_abi','LIB','B','X','','s','sd','Y')`,
		`INSERT INTO gzdel_t VALUES ('aapq110_01','zh_CN','供应商对账单明细查询报表打印')`,
		`INSERT INTO gzdel_t VALUES ('cl_abi','zh_CN','ABI Library')`,
		`CREATE TABLE dzeal_t (dzeal001 TEXT, dzeal002 TEXT, dzeal003 TEXT)`,
		`INSERT INTO dzeal_t VALUES ('glab_t','zh_CN','账套应用会计科目设置档')`,
		`INSERT INTO dzeal_t VALUES ('ooag_t','zh_CN','员工数据档')`,
		// aapi011 对 glab_t 是完整读写,对 ooag_t 只读;aapi201 也读写 glab_t
		`INSERT INTO gzdg_t VALUES ('aapi011','glab_t','S')`,
		`INSERT INTO gzdg_t VALUES ('aapi011','glab_t','U')`,
		`INSERT INTO gzdg_t VALUES ('aapi011','glab_t','I')`,
		`INSERT INTO gzdg_t VALUES ('aapi011','glab_t','D')`,
		`INSERT INTO gzdg_t VALUES ('aapi011','ooag_t','S')`,
		`INSERT INTO gzdg_t VALUES ('aapi201','glab_t','S')`,
		`INSERT INTO gzdg_t VALUES ('aapi201','glab_t','U')`,
		// aooi301:被两个作业使用的共用维护程序;aapi011:作业编号=程序编号
		`INSERT INTO gzza_t VALUES ('aooi301','i','AOO','$FGLRUN $AOOi/aooi301','','s','Y')`,
		`INSERT INTO gzza_t VALUES ('aapi011','i','AAP','$FGLRUN $AAPi/aapi011','','s','Y')`,
		`INSERT INTO gzzal_t VALUES ('aooi301','zh_CN','应用分类码维护作业(单档多栏)','YYFLMWH')`,
		`INSERT INTO gzzal_t VALUES ('aapi011','zh_CN','应付账款类别依账套设置科目作业','YFZKLB')`,
		`INSERT INTO gzzal_t VALUES ('aooi301','zh_TW','應用分類碼維護作業','YYFLMWH')`,
		`INSERT INTO gzzz_t VALUES ('aapi011','aapi011','0','','AAP','','Y')`,
		`INSERT INTO gzzz_t VALUES ('aooi701','aooi301','0','','AOO','','Y')`,
		`INSERT INTO gzzz_t VALUES ('aooi707','aooi301','1','vendor','AOO','','Y')`,
		`INSERT INTO gzzk_t VALUES ('aooi301','0','预设参数组')`,
		`INSERT INTO gzzk_t VALUES ('aooi301','1','单档多栏参数组')`,
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

// TestQueryProgInfo 程序编号与作业编号两种形态,以及未收录。
func TestQueryProgInfo(t *testing.T) {
	d := buildProgDB(t)

	// 程序:aooi301
	info, err := d.QueryProgInfo("aooi301", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgInfo(aooi301): %v", err)
	}
	if info == nil || !info.IsProg {
		t.Fatalf("aooi301 应登记为程序, got %+v", info)
	}
	if info.Name != "应用分类码维护作业(单档多栏)" || info.Module != "AOO" || info.Category != "i" {
		t.Errorf("aooi301 字段不符: %+v", info)
	}
	if info.IsJob {
		t.Errorf("aooi301 不是作业: %+v", info)
	}

	// 繁体名称:换 --lang 应取到繁体那一行
	tw, err := d.QueryProgInfo("aooi301", "zh_TW")
	if err != nil {
		t.Fatalf("QueryProgInfo(zh_TW): %v", err)
	}
	if tw == nil || tw.Name != "應用分類碼維護作業" {
		t.Errorf("zh_TW 名称不符: %+v", tw)
	}

	// 作业编号 != 程序编号:aooi701 挂 aooi301,自身不是程序
	job, err := d.QueryProgInfo("aooi701", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgInfo(aooi701): %v", err)
	}
	if job == nil || job.IsProg || !job.IsJob || job.JobProg != "aooi301" {
		t.Errorf("aooi701 应只是作业且挂 aooi301: %+v", job)
	}

	// 未收录:返回 nil,nil(不报错)
	none, err := d.QueryProgInfo("nosuch01", "zh_CN")
	if err != nil || none != nil {
		t.Errorf("未收录应返回 (nil,nil), got %+v, %v", none, err)
	}
}

// TestQueryProgJobs 一个程序被多个作业使用,且参数组说明按 gzzk 关联带出。
func TestQueryProgJobs(t *testing.T) {
	d := buildProgDB(t)

	jobs, err := d.QueryProgJobs("aooi301", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgJobs: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("aooi301 应有 2 个作业, got %d: %+v", len(jobs), jobs)
	}
	if jobs[0].JobCode != "aooi701" || jobs[1].JobCode != "aooi707" {
		t.Errorf("作业应按编号升序: %+v", jobs)
	}
	// 作业名称取所挂程序的名称(azzi910 的取法)
	if jobs[0].JobName != "应用分类码维护作业(单档多栏)" {
		t.Errorf("作业名称应取程序名称: %+v", jobs[0])
	}
	if jobs[0].ParamGrp != "0" || jobs[0].ParamDesc != "预设参数组" {
		t.Errorf("应用参数组关联有误: %+v", jobs[0])
	}
	if jobs[1].ParamGrp != "1" || jobs[1].Extra != "vendor" || jobs[1].ParamDesc != "单档多栏参数组" {
		t.Errorf("第二个作业的参数组/额外参数关联有误: %+v", jobs[1])
	}

	only, err := d.QueryProgJobs("aapi011", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgJobs(aapi011): %v", err)
	}
	if len(only) != 1 || only[0].JobCode != "aapi011" {
		t.Errorf("aapi011 应只有 1 个作业: %+v", only)
	}
}

// TestQuerySubProgInfo 子程序/元件登记:能答"这个编号是什么",且主程序不在其中
// (与 gzza_t 互不重叠——实测正式区两表交集 0)。
func TestQuerySubProgInfo(t *testing.T) {
	d := buildProgDB(t)

	sub, err := d.QuerySubProgInfo("aapq110_01", "zh_CN")
	if err != nil {
		t.Fatalf("QuerySubProgInfo: %v", err)
	}
	if sub == nil {
		t.Fatal("aapq110_01 应登记在 gzde_t")
	}
	if sub.Name != "供应商对账单明细查询报表打印" || sub.Category != "S" || sub.Module != "AAP" {
		t.Errorf("字段不符: %+v", sub)
	}
	if sub.ProgCat != "Q" || sub.Status != "Y" || sub.Cust != "s" {
		t.Errorf("字段不符: %+v", sub)
	}

	// 主程序(aapi011)不在 gzde_t —— 两个登记表互补
	if p, err := d.QuerySubProgInfo("aapi011", "zh_CN"); err != nil || p != nil {
		t.Errorf("主程序不该出现在 gzde_t: %+v, %v", p, err)
	}
	// 未收录
	if p, err := d.QuerySubProgInfo("nosuch", "zh_CN"); err != nil || p != nil {
		t.Errorf("未收录应返回 (nil,nil): %+v, %v", p, err)
	}
}

// TestQuerySubProgList 子程序/元件列表与搜索,--kw 按编号或说明。
func TestQuerySubProgList(t *testing.T) {
	d := buildProgDB(t)

	all, err := d.QuerySubProgList("zh_CN", "")
	if err != nil {
		t.Fatalf("QuerySubProgList: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("应有 2 个, got %d", len(all))
	}
	byName, err := d.QuerySubProgList("zh_CN", "ABI")
	if err != nil {
		t.Fatalf("QuerySubProgList(ABI): %v", err)
	}
	if len(byName) != 1 || byName[0].Code != "cl_abi" || byName[0].Category != "B" {
		t.Errorf("按说明搜索应命中 cl_abi: %+v", byName)
	}
}

// TestSubProgCategoryLabel 规格类别(SCC 91)注解。
func TestSubProgCategoryLabel(t *testing.T) {
	cases := map[string]string{"B": "(应用元件)", "S": "(子程序)", "M": "(主程序)",
		"G": "(报表元件-GR类)", "X": "(报表元件-XG/FR类)", "K": "(报表组件-XR类)",
		"W": "(WebService元件)", "": "", "z": ""}
	for in, want := range cases {
		if got := SubProgCategoryLabel(in); got != want {
			t.Errorf("SubProgCategoryLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestQueryProgTables 程序 → 表格:同一张表的多个操作合并成一行(固定顺序 S/I/U/D),
// 表名走 dzeal_t(指定语言),按表名升序。
func TestQueryProgTables(t *testing.T) {
	d := buildProgDB(t)

	rows, err := d.QueryProgTables("aapi011", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgTables: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("aapi011 应命中 2 张表, got %d: %+v", len(rows), rows)
	}
	if rows[0].Table != "glab_t" || rows[0].Ops != "S/I/U/D" {
		t.Errorf("glab_t 操作应为 S/I/U/D: %+v", rows[0])
	}
	if rows[0].TableDesc != "账套应用会计科目设置档" {
		t.Errorf("表说明应取 dzeal_t: %+v", rows[0])
	}
	if rows[1].Table != "ooag_t" || rows[1].Ops != "S" {
		t.Errorf("ooag_t 应只有 S: %+v", rows[1])
	}

	// 另一程序对同一张表只读写
	rows2, err := d.QueryProgTables("aapi201", "zh_CN")
	if err != nil {
		t.Fatalf("QueryProgTables(aapi201): %v", err)
	}
	if len(rows2) != 1 || rows2[0].Ops != "S/U" {
		t.Errorf("aapi201 对 glab_t 应为 S/U: %+v", rows2)
	}

	// 未登记用表的程序:空结果,不报错
	if none, err := d.QueryProgTables("nosuch", "zh_CN"); err != nil || len(none) != 0 {
		t.Errorf("未登记者应返回空: %+v, %v", none, err)
	}
}

// TestQueryTablePrograms 表 → 程序:反查哪些程序在用这张表(含操作类别)。
func TestQueryTablePrograms(t *testing.T) {
	d := buildProgDB(t)

	rows, err := d.QueryTablePrograms("glab_t", "zh_CN")
	if err != nil {
		t.Fatalf("QueryTablePrograms: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("glab_t 应被 2 个程序使用, got %d: %+v", len(rows), rows)
	}
	if rows[0].Prog != "aapi011" || rows[0].Ops != "S/I/U/D" {
		t.Errorf("aapi011 行不符: %+v", rows[0])
	}
	if rows[1].Prog != "aapi201" || rows[1].Ops != "S/U" {
		t.Errorf("aapi201 行不符: %+v", rows[1])
	}
	// 程序名取 gzzal_t(该编号在 gzzal_t 里没有名称时为空的用例见下)
	if none, err := d.QueryTablePrograms("nosuch_t", "zh_CN"); err != nil || len(none) != 0 {
		t.Errorf("没人用的表应返回空: %+v, %v", none, err)
	}
}

// TestMergeOps 操作类别合并:固定顺序 S/I/U/D,未知码按字母序追加。
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

// TestQueryProgList 列表带作业数,--kw 按编号或中文名称过滤。
func TestQueryProgList(t *testing.T) {
	d := buildProgDB(t)

	all, err := d.QueryProgList("zh_CN", "")
	if err != nil {
		t.Fatalf("QueryProgList: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("应有 2 个程序, got %d", len(all))
	}
	if all[0].Code != "aapi011" || all[0].JobCount != 1 {
		t.Errorf("aapi011 作业数应为 1: %+v", all[0])
	}
	if all[1].Code != "aooi301" || all[1].JobCount != 2 {
		t.Errorf("aooi301 作业数应为 2(一个程序被多个作业使用): %+v", all[1])
	}

	byName, err := d.QueryProgList("zh_CN", "分类码")
	if err != nil {
		t.Fatalf("QueryProgList(按名称): %v", err)
	}
	if len(byName) != 1 || byName[0].Code != "aooi301" {
		t.Errorf("按中文名称搜索应命中 aooi301: %+v", byName)
	}

	byCode, err := d.QueryProgList("zh_CN", "aapi")
	if err != nil {
		t.Fatalf("QueryProgList(按编号): %v", err)
	}
	if len(byCode) != 1 || byCode[0].Code != "aapi011" {
		t.Errorf("按编号搜索应命中 aapi011: %+v", byCode)
	}

	if none, err := d.QueryProgList("zh_CN", "无此程序"); err != nil || len(none) != 0 {
		t.Errorf("无匹配应返回空: %+v, %v", none, err)
	}
}
