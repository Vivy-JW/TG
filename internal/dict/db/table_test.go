package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildTableDB 建一个只含 GP 表字典四张表（t_table 表档 / t_column 字段 /
// t_index 索引 / t_constraint 约束）的临时库，列与 tt gp sync 产出的同构。
func buildTableDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tables.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE t_table (tab TEXT, tab_cn TEXT, module TEXT, tab_type TEXT, col_count TEXT)`,
		`CREATE TABLE t_column (tab TEXT, seq TEXT, col TEXT, col_cn TEXT, dtype TEXT,
		   dlen TEXT, dprec TEXT, dscale TEXT, nullable TEXT, dflt TEXT, remark TEXT, fmt TEXT)`,
		`CREATE TABLE t_index (tab TEXT, idx TEXT, uniqueness TEXT, cols TEXT)`,
		`CREATE TABLE t_constraint (tab TEXT, cons TEXT, ctype TEXT, status TEXT, validated TEXT, cols TEXT)`,
		`INSERT INTO t_table VALUES ('ima_file','料件主档','INV','1','2')`,
		`INSERT INTO t_table VALUES ('apca_file','应付凭单单头','APP','1','1')`,
		// ima_file 两个字段，ima01 是主键且必填
		`INSERT INTO t_column VALUES ('ima_file','1','ima01','料件编号','VARCHAR2','40','','','N','','','001')`,
		`INSERT INTO t_column VALUES ('ima_file','2','ima02','料件名称','VARCHAR2','100','','','Y','','','')`,
		`INSERT INTO t_column VALUES ('apca_file','1','apca01','单别','VARCHAR2','4','','','N','','','')`,
		`INSERT INTO t_constraint VALUES ('ima_file','ima_file_pk','P','ENABLED','VALIDATED','ima01')`,
		`INSERT INTO t_constraint VALUES ('ima_file','ima_file_uk','U','ENABLED','VALIDATED','ima02')`,
		`INSERT INTO t_index VALUES ('ima_file','ima_file_pk','UNIQUE','ima01')`,
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

// TestQueryTableList 列表模式：字段数由 JOIN 聚合得出，关键字按表名/中文说明过滤。
func TestQueryTableList(t *testing.T) {
	d := buildTableDB(t)

	all, err := d.QueryTableList("zh_CN", "")
	if err != nil {
		t.Fatalf("QueryTableList: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("应有 2 张表, got %d: %+v", len(all), all)
	}
	if all[0].TableName != "apca_file" || all[0].TableDesc != "应付凭单单头" || all[0].Module != "APP" {
		t.Errorf("apca_file 行不符: %+v", all[0])
	}
	if all[1].TableName != "ima_file" || all[1].FieldCnt != 2 {
		t.Errorf("ima_file 字段数应为 2: %+v", all[1])
	}

	byDesc, err := d.QueryTableList("zh_CN", "凭单")
	if err != nil {
		t.Fatalf("QueryTableList(凭单): %v", err)
	}
	if len(byDesc) != 1 || byDesc[0].TableName != "apca_file" {
		t.Errorf("按表说明过滤应命中 apca_file: %+v", byDesc)
	}

	byName, err := d.QueryTableList("zh_CN", "ima")
	if err != nil || len(byName) != 1 || byName[0].TableName != "ima_file" {
		t.Errorf("按表名过滤应命中 ima_file: %+v, %v", byName, err)
	}

	if none, err := d.QueryTableList("zh_CN", "无此表"); err != nil || len(none) != 0 {
		t.Errorf("无匹配应返回空: %+v, %v", none, err)
	}
}

// TestQueryTableMeta 表档：未收录返回 (nil, nil)。
func TestQueryTableMeta(t *testing.T) {
	d := buildTableDB(t)

	meta, err := d.QueryTableMeta("ima_file")
	if err != nil {
		t.Fatalf("QueryTableMeta: %v", err)
	}
	if meta == nil || meta.TableDesc != "料件主档" || meta.Module != "INV" {
		t.Errorf("ima_file 表档不符: %+v", meta)
	}

	if none, err := d.QueryTableMeta("nosuch_file"); err != nil || none != nil {
		t.Errorf("未收录应返回 (nil,nil): %+v, %v", none, err)
	}
}

// TestQueryTable 字段字典：主键由 t_constraint 推、必填由 nullable 推。
func TestQueryTable(t *testing.T) {
	d := buildTableDB(t)

	cols, err := d.QueryTable("ima_file")
	if err != nil {
		t.Fatalf("QueryTable: %v", err)
	}
	if len(cols) != 2 {
		t.Fatalf("ima_file 应有 2 个字段, got %d: %+v", len(cols), cols)
	}
	if cols[0].FieldName != "ima01" || cols[0].FieldDesc != "料件编号" || cols[0].IsPK != "PK" {
		t.Errorf("ima01 行不符: %+v", cols[0])
	}
	if cols[0].Required != "Y" {
		t.Errorf("ima01 nullable=N 应算必填: %+v", cols[0])
	}
	if cols[0].Length != "40" || cols[0].DataType != "VARCHAR2" {
		t.Errorf("长度/类型不符: %+v", cols[0])
	}
	if cols[1].FieldName != "ima02" || cols[1].IsPK != "" || cols[1].Required != "" {
		t.Errorf("ima02 行不符: %+v", cols[1])
	}
}

// TestQueryKeys 键值：ctype 翻成中文。
func TestQueryKeys(t *testing.T) {
	d := buildTableDB(t)

	keys, err := d.QueryKeys("ima_file")
	if err != nil {
		t.Fatalf("QueryKeys: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("ima_file 应有 2 个约束, got %d: %+v", len(keys), keys)
	}
	if keys[0].KeyName != "ima_file_pk" || keys[0].KeyType != "主键" || keys[0].KeyField != "ima01" {
		t.Errorf("主键行不符: %+v", keys[0])
	}
	if keys[1].KeyType != "唯一" {
		t.Errorf("唯一键应翻成“唯一”: %+v", keys[1])
	}
}

// TestQueryIndexes 索引。
func TestQueryIndexes(t *testing.T) {
	d := buildTableDB(t)

	idx, err := d.QueryIndexes("ima_file")
	if err != nil {
		t.Fatalf("QueryIndexes: %v", err)
	}
	if len(idx) != 1 || idx[0].IndexName != "ima_file_pk" || idx[0].IndexType != "UNIQUE" {
		t.Errorf("索引不符: %+v", idx)
	}
}
