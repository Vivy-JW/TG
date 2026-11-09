package dbsync

import (
	"path/filepath"
	"slices"
	"testing"

	"tt/internal/dict/db"
)

// TestDictTablesDerivedFromFamilies 同步清单由数据族展平而来:不重不漏。
func TestDictTablesDerivedFromFamilies(t *testing.T) {
	seen := make(map[string]bool, len(DictTables))
	for _, f := range Families {
		for _, tb := range f.Tables {
			seen[tb] = true
		}
	}
	if len(DictTables) != len(seen) {
		t.Errorf("DictTables 有重复: 清单 %d 项,唯一表名 %d 个", len(DictTables), len(seen))
	}
	for _, f := range Families {
		if FamilyByKey(f.Key) == nil {
			t.Errorf("数据族 %s 查不到", f.Key)
		}
		if len(f.Commands) == 0 || len(f.Tables) == 0 {
			t.Errorf("数据族 %s 缺命令或表: %+v", f.Key, f)
		}
	}
	if FamilyByKey("nosuch") != nil {
		t.Error("未知 key 应返回 nil")
	}
}

// TestProgFamilyInSyncList "程序与作业"一族的表必须进同步清单(tt dict prog 的本地数据来源)。
func TestProgFamilyInSyncList(t *testing.T) {
	for _, want := range []string{"gzza_t", "gzzz_t", "gzzk_t", "gzzal_t"} {
		if !slices.Contains(DictTables, want) {
			t.Errorf("%s 不在 DictTables 里,tt dict db sync 不会拉它", want)
		}
	}
	// gzzal_t 同时服务"系统消息"与"程序与作业"两族(展平时去重)
	var hit []string
	for _, f := range Families {
		if slices.Contains(f.Tables, "gzzal_t") {
			hit = append(hit, f.Key)
		}
	}
	if len(hit) != 2 {
		t.Errorf("gzzal_t 应被两族共用, got %v", hit)
	}
}

// TestInspectLocal 覆盖情况:库不存在时全部族缺失;建了部分表则只有该族齐、别的仍然缺。
func TestInspectLocal(t *testing.T) {
	dir := t.TempDir()

	// 库不存在:不报错,全部族缺失
	missing, err := InspectLocal(filepath.Join(dir, "nope.db"))
	if err != nil {
		t.Fatalf("InspectLocal(不存在): %v", err)
	}
	if missing.Exists {
		t.Error("库不存在时 Exists 应为 false")
	}
	for _, f := range missing.Families {
		if f.Complete {
			t.Errorf("库不存在时 %s 不该算已同步", f.Name)
		}
	}
	if got := missing.Summary("prog"); got == "" || got == "全部已同步" {
		t.Errorf("缺失时摘要应给提示, got %q", got)
	}

	// 只建"表字典"一族的表:该族 ✓,程序与作业 ✗
	path := filepath.Join(dir, "part.db")
	d, err := db.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	for _, tb := range FamilyByKey("table").Tables {
		if _, err := d.RebuildTable(tb, []string{"c"}, nil); err != nil {
			t.Fatalf("create %s: %v", tb, err)
		}
	}
	d.Close()

	st, err := InspectLocal(path)
	if err != nil {
		t.Fatalf("InspectLocal: %v", err)
	}
	if !st.Exists {
		t.Fatal("库应存在")
	}
	for _, f := range st.Families {
		switch f.Key {
		case "table":
			if !f.Complete || f.Present != len(f.Tables) {
				t.Errorf("表字典应齐全: %+v", f)
			}
		case "prog":
			if f.Complete || len(f.Missing) != len(f.Tables) {
				t.Errorf("程序与作业应全缺: %+v", f)
			}
		}
	}
	if got := st.Summary("table"); got != "表字典 ✓" {
		t.Errorf("单族摘要 = %q, want %q", got, "表字典 ✓")
	}
	if got := st.Summary(); got == "" || got == "全部已同步" {
		t.Errorf("整体摘要应指出还有未同步的族, got %q", got)
	}
}
