package db

import (
	"path/filepath"
	"testing"
)

func TestACCQueries(t *testing.T) {
	d, err := Open(filepath.Join(t.TempDir(), "acc.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()

	put := func(name string, cols []string, rows [][]string) {
		t.Helper()
		if _, err := d.RebuildTable(name, cols, rows); err != nil {
			t.Fatalf("建 %s: %v", name, err)
		}
	}
	put("gzaa_t", []string{"gzaa001", "gzaa002", "gzaa003", "gzaa004", "gzaa005", "gzaastus"},
		[][]string{{"3211", "Y", "N", "4", "N", "Y"}})
	put("gzaal_t", []string{"gzaal001", "gzaal002", "gzaal003"},
		[][]string{{"3211", "zh_CN", "应付账务类型"}})

	titles := []string{"3211", "zh_CN"}
	for i := 0; i < 20; i++ {
		titles = append(titles, "标题"+string(rune('A'+i)))
	}
	put("gzac_t", []string{"gzac001", "gzac002", "gzac003", "gzac004", "gzac005", "gzac006", "gzac007", "gzac008", "gzac009", "gzac010", "gzac011", "gzac012", "gzac013", "gzac014", "gzac015", "gzac016", "gzac017", "gzac018", "gzac019", "gzac020", "gzac021", "gzac022"},
		[][]string{titles})
	put("gzad_t", []string{"gzad001", "gzad002", "gzad003", "gzad004", "gzad005", "gzad006", "gzad007", "gzad008", "gzad009", "gzad014"},
		[][]string{{"3211", "1", "Y", "Y", "4", "3212", "q_oocq002", "v_oocq002", "", ""}})
	put("gzzz_t", []string{"gzzz001", "gzzz002", "gzzz004", "gzzzstus"},
		[][]string{{"aapi010", "aooi301", `"3211"`, "Y"}})
	put("gzzal_t", []string{"gzzal001", "gzzal002", "gzzal003"},
		[][]string{{"aooi301", "zh_CN", "应用分类码维护作业"}})

	oc := []string{"oocqent", "oocqstus", "oocq001", "oocq002", "oocq003"}
	for i := 4; i <= 23; i++ {
		oc = append(oc, "oocq0"+itoa2(i))
	}
	refs := make([]string, 20)
	refs[0] = "类型一"
	row := []string{"2222", "Y", "3211", "P001", "3211"}
	row = append(row, refs...)
	row2 := []string{"2222", "N", "3211", "P002", "3211"}
	row2 = append(row2, refs...)
	put("oocq_t", oc, [][]string{row, row2})
	put("oocql_t", []string{"oocqlent", "oocql001", "oocql002", "oocql003", "oocql004"},
		[][]string{{"2222", "3211", "P001", "zh_CN", "说明一"}, {"2222", "3211", "P002", "zh_CN", "说明二"}})

	meta, err := d.QueryACCMeta("aapi010", "zh_CN")
	if err != nil {
		t.Fatal(err)
	}
	if meta == nil || meta.Category != "3211" || meta.Name != "应付账务类型" || len(meta.Jobs) != 1 {
		t.Fatalf("meta = %+v", meta)
	}
	if meta.MatchedBy != "job" || meta.Fields[0].Title != "标题A" || !meta.Fields[0].Visible {
		t.Fatalf("fields = %+v matchedBy=%s", meta.Fields[0], meta.MatchedBy)
	}
	vals, err := d.QueryACCValues("3211", 2222, "zh_CN", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 1 || vals[0].Value != "P001" || vals[0].Desc != "说明一" || vals[0].Refs[0] != "类型一" {
		t.Fatalf("values = %+v", vals)
	}
	all, err := d.QueryACCValues("3211", 2222, "zh_CN", true)
	if err != nil || len(all) != 2 {
		t.Fatalf("includeInactive values = %+v, %v", all, err)
	}
}

func itoa2(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
