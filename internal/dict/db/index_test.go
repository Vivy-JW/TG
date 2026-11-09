package db

import (
	"path/filepath"
	"strings"
	"testing"
)

// 索引列名与表结构对不上时必须明确报错。SQLite 的行为在这里特别容易骗人：
// 带引号又解析不到标识符的列会被当成**字符串字面量**，于是在 (常量) 上建出一个
// 假索引 —— 单行表静默成功（一点也没加速，还让表只能插一行），多行表报
// "UNIQUE constraint failed"，那句话把人引向"数据重复"，与真正的原因无关。
func TestCreateUniqueIndexRejectsMissingColumn(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := d.RebuildTable("dzeb_t", []string{"OTHER1"}, [][]string{{"a"}}); err != nil {
		t.Fatal(err)
	}

	err = d.CreateUniqueIndex("idx_dzeb_t_dzeb_pk", "dzeb_t", []string{"dzeb001"})
	if err == nil {
		t.Fatal("列不存在却在单行表上\"成功\"建出了假索引")
	}
	if !strings.Contains(err.Error(), "没有列") {
		t.Errorf("错误信息应指明是哪一列不存在（而不是唯一约束之类的误导），实得：%v", err)
	}

	// 假索引不能留在库里：它会挡住第二行插入。
	_, rows, qerr := d.QueryRaw("select name from sqlite_master where type='index' and tbl_name='dzeb_t'")
	if qerr != nil {
		t.Fatal(qerr)
	}
	if len(rows) != 0 {
		t.Errorf("失败时不该留下任何索引，实得 %v", rows)
	}
}

// 列名大小写不同不影响判定（SQLite 标识符本就大小写不敏感）。
func TestCreateUniqueIndexColumnCaseInsensitive(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	if _, err := d.RebuildTable("dzeb_t", []string{"DZEB001", "DZEB002"}, [][]string{{"a", "1"}}); err != nil {
		t.Fatal(err)
	}
	if err := d.CreateUniqueIndex("idx_dzeb_t_dzeb_pk", "dzeb_t", []string{"dzeb001", "dzeb002"}); err != nil {
		t.Fatalf("列存在（只是大小写不同）时不该报错：%v", err)
	}
	_, rows, err := d.QueryRaw("select name from sqlite_master where type='index' and tbl_name='dzeb_t'")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("索引应当真建出来，实得 %v", rows)
	}
}
