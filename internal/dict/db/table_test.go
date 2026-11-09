package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildTableDB 建一个只含表字典三张表(dzea_t 表档/dzeal_t 表说明/dzeb_t 字段档)的临时库。
func buildTableDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tables.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE dzea_t (dzea001 TEXT, dzea002 TEXT, dzea003 TEXT, dzea004 TEXT)`,
		`CREATE TABLE dzeal_t (dzeal001 TEXT, dzeal002 TEXT, dzeal003 TEXT)`,
		`CREATE TABLE dzeb_t (dzeb001 TEXT, dzeb002 TEXT, dzeb021 TEXT)`,
		`INSERT INTO dzea_t VALUES ('apca_t','應付憑單','AAP','T')`,
		`INSERT INTO dzea_t VALUES ('glab_t','帳套科目','AGL','D')`,
		`INSERT INTO dzeal_t VALUES ('apca_t','zh_CN','应付凭单单头')`,
		`INSERT INTO dzeal_t VALUES ('glab_t','zh_CN','账套应用会计科目设置档')`,
		// apca_t 两个字段,glab_t 一个字段
		`INSERT INTO dzeb_t VALUES ('apca_t','apca001','1')`,
		`INSERT INTO dzeb_t VALUES ('apca_t','apca002','2')`,
		`INSERT INTO dzeb_t VALUES ('glab_t','glab001','1')`,
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

// TestQueryTableList 列表模式:字段数由 JOIN 聚合得出(不是逐行相关子查询),
// 关键字按表名/中文表说明过滤,空关键字返回全部。
func TestQueryTableList(t *testing.T) {
	d := buildTableDB(t)

	all, err := d.QueryTableList("zh_CN", "")
	if err != nil {
		t.Fatalf("QueryTableList: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("应有 2 张表, got %d: %+v", len(all), all)
	}
	if all[0].TableName != "apca_t" || all[0].TableDesc != "应付凭单单头" || all[0].Module != "AAP" {
		t.Errorf("apca_t 行不符: %+v", all[0])
	}
	if all[0].FieldCnt != 2 {
		t.Errorf("apca_t 字段数应为 2, got %d", all[0].FieldCnt)
	}
	if all[1].TableName != "glab_t" || all[1].FieldCnt != 1 {
		t.Errorf("glab_t 行不符: %+v", all[1])
	}

	// 按中文表说明过滤(说明在 dzeal_t,是 LEFT JOIN 来的)
	byDesc, err := d.QueryTableList("zh_CN", "凭单")
	if err != nil {
		t.Fatalf("QueryTableList(凭单): %v", err)
	}
	if len(byDesc) != 1 || byDesc[0].TableName != "apca_t" {
		t.Errorf("按表说明过滤应命中 apca_t: %+v", byDesc)
	}

	// 按表名过滤
	byName, err := d.QueryTableList("zh_CN", "glab")
	if err != nil {
		t.Fatalf("QueryTableList(glab): %v", err)
	}
	if len(byName) != 1 || byName[0].TableName != "glab_t" {
		t.Errorf("按表名过滤应命中 glab_t: %+v", byName)
	}

	// 换了语言别时取不到 zh_CN 说明,退回原始表档说明(dzea002,繁体)
	tw, err := d.QueryTableList("zh_TW", "")
	if err != nil {
		t.Fatalf("QueryTableList(zh_TW): %v", err)
	}
	if tw[0].TableDesc != "應付憑單" {
		t.Errorf("无该语言行应退回原始表说明, got %q", tw[0].TableDesc)
	}

	// 无匹配:空结果,不报错
	if none, err := d.QueryTableList("zh_CN", "无此表"); err != nil || len(none) != 0 {
		t.Errorf("无匹配应返回空: %+v, %v", none, err)
	}
}
