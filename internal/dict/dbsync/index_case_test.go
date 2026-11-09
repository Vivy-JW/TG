package dbsync

import (
	"path/filepath"
	"testing"

	"tt/internal/dict/db"
)

// dzed_t 的列名在 Oracle 侧是大写（DZED001…），而建索引时查的是小写。
// 这条钉住"大小写不敏感"：否则一条索引都建不出来，且 createPKIndexes 返回 nil
// 警告 —— 同步看起来成功，查询却仍是全表扫。
func TestCreatePKIndexesCaseInsensitive(t *testing.T) {
	sq, err := db.Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sq.Close()

	if _, err := sq.RebuildTable("dzeb_t", []string{"DZEB001", "DZEB002"}, [][]string{{"a", "1"}}); err != nil {
		t.Fatal(err)
	}
	dzedCols := []string{"DZEDSTUS", "DZED001", "DZED002", "DZED003", "DZED004"}
	dzedRows := [][]string{{"", "dzeb_t", "dzeb_pk", "P", "dzeb001,dzeb002"}}

	if warns := createPKIndexes(sq, dzedCols, dzedRows, map[string]bool{"dzeb_t": true}); len(warns) != 0 {
		t.Fatalf("不该有警告: %v", warns)
	}

	_, rows, err := sq.QueryRaw("select name from sqlite_master where type='index' and tbl_name='dzeb_t'")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("大写列名的 dzed_t 没建出主键索引（大小写比较写错了？）")
	}
}
