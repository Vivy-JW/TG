package dbsync

import (
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/dict/db"
)

// 建索引这件事有两种静默失败，两种都必须被看见：
//  1. dzed_t 的列名大小写与代码里的比较对不上（PR #27 修的那个，已由
//     TestCreatePKIndexesCaseInsensitive 钉住）；
//  2. 该建的都该建，结果一个都没建成 —— 返回 nil 警告等于告诉用户"同步成功"。
//
// 第二类在没有这条用例之前是没人会发现的：查得慢只会被当成"库就是这样"。
func TestCreatePKIndexesWarnsWhenNothingCreated(t *testing.T) {
	sq, err := db.Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sq.Close()

	// 表里根本没有 dzeb001/dzeb002 这两列：CREATE UNIQUE INDEX 必失败。
	if _, err := sq.RebuildTable("dzeb_t", []string{"OTHER1"}, [][]string{{"a"}}); err != nil {
		t.Fatal(err)
	}
	dzedCols := []string{"DZED001", "DZED002", "DZED003", "DZED004"}
	dzedRows := [][]string{{"dzeb_t", "dzeb_pk", "P", "dzeb001,dzeb002"}}

	warns := createPKIndexes(sq, dzedCols, dzedRows, map[string]bool{"dzeb_t": true})
	if len(warns) == 0 {
		t.Fatal("一个索引都没建成，却没有任何警告（用户会以为同步成功）")
	}
	joined := strings.Join(warns, "\n")
	if !strings.Contains(joined, "实际建成 0 个") {
		t.Errorf("应当有一条\"建成 0 个\"的汇总警告，实得：\n%s", joined)
	}
}

// 只同步了无主键表：应建数本来就是 0，不该报警（否则每次 --table 单表都刷噪声）。
func TestCreatePKIndexesNoWarnWhenNothingExpected(t *testing.T) {
	sq, err := db.Open(filepath.Join(t.TempDir(), "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer sq.Close()

	if _, err := sq.RebuildTable("dzeb_t", []string{"DZEB001"}, [][]string{{"a"}}); err != nil {
		t.Fatal(err)
	}
	dzedCols := []string{"DZED001", "DZED002", "DZED003", "DZED004"}
	dzedRows := [][]string{
		{"dzed_t", "dzed_pk", "P", "dzed001"}, // dzed_t 本身本次没同步
		{"dzec_t", "dzec_pk", "P", "dzec001"}, // 同样没同步
		{"dzeb_t", "dzeb_pk", "U", "dzeb001"}, // 不是主键
	}

	if warns := createPKIndexes(sq, dzedCols, dzedRows, map[string]bool{"dzeb_t": true}); len(warns) != 0 {
		t.Fatalf("没有该建的索引时不该有警告：%v", warns)
	}
}

// `--table` 允许用户手输表名：传大写的 DZED_T 时，dzedCols 也必须被认出来，
// 否则 PK 定义整份丢失，索引一个都建不出来（与列名大小写是两个独立的洞）。
func TestDzedTableNameCaseInsensitive(t *testing.T) {
	for _, name := range []string{"dzed_t", "DZED_T", "Dzed_T", " dzed_t "} {
		if !isDzedTable(name) {
			t.Errorf("%q 应被认成 dzed_t（否则 --table 手输大写会漏建全部索引）", name)
		}
	}
	for _, name := range []string{"dzeb_t", "dzed", "dz ed_t", ""} {
		if isDzedTable(name) {
			t.Errorf("%q 不是 dzed_t，不该被当成 PK 定义表", name)
		}
	}
}
