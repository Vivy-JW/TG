package dict

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/dict/db"
	"tt/internal/output"
)

func sampleTable() *db.TableDict {
	return &db.TableDict{
		TableName: "apca_t",
		Fields: []db.TableInfo{
			{FieldName: "apca035", FieldDesc: "暂估科目"},
			{FieldName: "apca036", FieldDesc: "暂估说明"},
			{FieldName: "apca037", FieldDesc: "币别"},
		},
	}
}

func TestProjectTableFieldsCol(t *testing.T) {
	d := sampleTable()
	if err := projectTableFields([]*db.TableDict{d}, "apca036,apca035", ""); err != nil {
		t.Fatal(err)
	}
	if len(d.Fields) != 2 || d.Fields[0].FieldName != "apca036" || d.Fields[1].FieldName != "apca035" {
		t.Fatalf("字段投影顺序不对: %+v", d.Fields)
	}
}

func TestProjectTableFieldsKW(t *testing.T) {
	d := sampleTable()
	if err := projectTableFields([]*db.TableDict{d}, "", "暂估"); err != nil {
		t.Fatal(err)
	}
	if len(d.Fields) != 2 {
		t.Fatalf("--kw 应命中 2 个字段: %+v", d.Fields)
	}
}

func TestProjectTableFieldsMissing(t *testing.T) {
	err := projectTableFields([]*db.TableDict{sampleTable()}, "apca999", "")
	oe, ok := err.(*output.Error)
	if !ok || oe.Code != "NOT_FOUND" {
		t.Fatalf("缺失字段应为 NOT_FOUND: %#v", err)
	}
	if !strings.Contains(oe.Message, "apca_t.apca999") {
		t.Fatalf("错误应点名具体字段: %s", oe.Message)
	}
}

func TestBriefSccDetail(t *testing.T) {
	d := &sccDetail{ID: "8504", Header: sccHeaderView{Name: "状态"}, Values: []sccValueView{{Val: "1", Desc: "启用", Sort: "1"}}}
	b := briefSccDetail(d)
	if b.ID != "8504" || b.Name != "状态" || len(b.Values) != 1 || b.Values[0].Desc != "启用" {
		t.Fatalf("SCC 投影不对: %+v", b)
	}
	if b.Values[0].Val != "1" {
		t.Fatalf("值不对: %+v", b.Values[0])
	}
}

func TestNormalizeProgTables(t *testing.T) {
	for in, want := range map[string]string{"": "names", "full": "full", "none": "none", "names": "names"} {
		if got, err := normalizeProgTables(in); err != nil || got != want {
			t.Fatalf("normalizeProgTables(%q) = %q,%v", in, got, err)
		}
	}
	if _, err := normalizeProgTables("bad"); err == nil {
		t.Fatal("非法 --tables 应报错")
	}
}

func TestShapeProgTablesNames(t *testing.T) {
	old := progTablesMode
	progTablesMode = "names"
	t.Cleanup(func() { progTablesMode = old })
	item := map[string]any{"表格": []db.ProgTableRow{{Table: "apca_t"}, {Table: "apcb_t"}}}
	if err := shapeProgTables(item); err != nil {
		t.Fatal(err)
	}
	names, ok := item["表格"].([]string)
	if !ok || len(names) != 2 || names[0] != "apca_t" {
		t.Fatalf("--tables names 形状不对: %#v", item["表格"])
	}
}

func TestNormalizeDocListMode(t *testing.T) {
	for in, want := range map[string]string{"": "full", "full": "full", "none": "none", "count": "count"} {
		if got, err := normalizeDocListMode(in); err != nil || got != want {
			t.Fatalf("normalizeDocListMode(%q) = %q,%v", in, got, err)
		}
	}
	if _, err := normalizeDocListMode("bad"); err == nil {
		t.Fatal("非法 --doclist 应报错")
	}
}

func TestTableSummaryOnlyIncludesFieldsWhenAsked(t *testing.T) {
	d := sampleTable()
	summary := tableSummaryOf(d, false, len(d.Fields))
	if summary.FieldCnt != 3 || len(summary.Fields) != 0 {
		t.Fatalf("默认 summary 只应有字段数: %+v", summary)
	}
	withFields := tableSummaryOf(d, true, len(d.Fields))
	if len(withFields.Fields) != 3 || withFields.Fields[0].FieldName != "apca035" {
		t.Fatalf("--fields 应包含字段骨架: %+v", withFields)
	}
}

func TestTableWhoSummaryAggregates(t *testing.T) {
	progs := []db.TableProgRow{
		{Prog: "aapp131", ProgName: "暂估应付", Ops: "S"},
		{Prog: "aapp132", ProgName: "对账", Ops: "S/I/U/D"},
	}
	s := tableWhoSummaryOf("apca_t", "应付凭单单头", progs)
	if s.ProgramCount != 2 || s.Ops["S"] != 2 || s.Ops["D"] != 1 || len(s.Sample) != 2 {
		t.Fatalf("影响摘要不对: %+v", s)
	}
}

func TestSummaryCommandsExposeFullFlag(t *testing.T) {
	for name, cmd := range map[string]interface{ Flags() interface{} }{} {
		_ = name
		_ = cmd
	}
	for name, cmd := range map[string]*cobra.Command{"prog": progCmd, "scc": sccCmd, "r.t": tableCmd} {
		if cmd.Flags().Lookup("full") == nil {
			t.Errorf("%s 缺少 --full", name)
		}
	}
}
