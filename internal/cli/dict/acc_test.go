package dict

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/dict/db"
)

func sampleACCMeta() *db.ACCMeta {
	return &db.ACCMeta{
		Category: "3211", Name: "应付账务类型", Tree: false,
		Fields: []db.ACCField{
			{Seq: 1, Key: "ref1", Title: "账务类型", Visible: true},
			{Seq: 2, Key: "ref2", Title: "隐藏字段", Visible: false},
		},
	}
}

func TestACCSelectedKeys(t *testing.T) {
	meta := sampleACCMeta()
	got := accSelectedKeys(meta, nil)
	want := []string{"value", "desc", "status", "ref1"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("默认 select = %v, want %v", got, want)
	}
	got = accSelectedKeys(meta, []string{"ref2"})
	if len(got) != 2 || got[0] != "value" || got[1] != "ref2" {
		t.Fatalf("显式 select 应保留 value: %v", got)
	}
}

func TestValidateACCSpec(t *testing.T) {
	meta := sampleACCMeta()
	spec := &accQuerySpec{Limit: 5, Where: []accCondition{{Field: "ref1", Op: "eq", Value: "A"}}}
	if err := validateACCSpec(meta, spec); err != nil {
		t.Fatal(err)
	}
	if err := validateACCSpec(meta, &accQuerySpec{Where: []accCondition{{Field: "ref99", Op: "eq", Value: "A"}}}); err == nil {
		t.Fatal("未知字段应报错")
	}
	if err := validateACCSpec(meta, &accQuerySpec{Where: []accCondition{{Field: "ref1", Op: "bad"}}}); err == nil {
		t.Fatal("未知操作符应报错")
	}
	if err := validateACCSpec(meta, &accQuerySpec{Tree: true}); err == nil {
		t.Fatal("非树状 ACC 不应允许 tree")
	}
}

func TestRunACCQueryFilterAndLimit(t *testing.T) {
	meta := sampleACCMeta()
	vals := []db.ACCValue{
		{Value: "A", Desc: "甲", Status: "Y", Refs: []string{"A1", ""}},
		{Value: "B", Desc: "乙", Status: "Y", Refs: []string{"B1", ""}},
		{Value: "C", Desc: "丙", Status: "Y", Refs: []string{"C1", ""}},
	}
	spec := &accQuerySpec{
		Select: []string{"value", "desc", "ref1"},
		Where:  []accCondition{{Field: "value", Op: "eq", Value: "B"}},
		Limit:  5,
	}
	data, total, returned, truncated, err := runACCQuery(meta, spec, vals)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || returned != 1 || truncated {
		t.Fatalf("过滤/计数不对: total=%d returned=%d truncated=%v", total, returned, truncated)
	}
	rows := data["values"].([]map[string]any)
	if rows[0]["value"] != "B" || rows[0]["ref1"] != "B1" {
		t.Fatalf("结果不对: %#v", rows[0])
	}
}

func TestRunACCQueryTree(t *testing.T) {
	meta := &db.ACCMeta{Category: "216", Tree: true, Fields: []db.ACCField{{Key: "ref1", Visible: true}}}
	vals := []db.ACCValue{
		{Value: "01", Desc: "root", Parent: "216", Refs: []string{""}},
		{Value: "011", Desc: "child", Parent: "01", Refs: []string{""}},
	}
	data, _, returned, _, err := runACCQuery(meta, &accQuerySpec{Tree: true, Limit: 20}, vals)
	if err != nil {
		t.Fatal(err)
	}
	if returned != 2 {
		t.Fatalf("returned=%d", returned)
	}
	roots := data["values"].([]map[string]any)
	if len(roots) != 1 || roots[0]["value"] != "01" {
		t.Fatalf("根节点不对: %#v", roots)
	}
	children := roots[0]["children"].([]map[string]any)
	if len(children) != 1 || children[0]["value"] != "011" {
		t.Fatalf("子节点不对: %#v", children)
	}
}

func TestReadACCSpecAcceptsTemplateShape(t *testing.T) {
	p := filepath.Join(t.TempDir(), "q.json")
	raw := `{"kind":"acc.query.template","query":{"where":[{"field":"value","op":"eq","value":"A"}],"limit":3}}`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := readACCSpec(p)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Limit != 3 || len(spec.Where) != 1 || spec.Where[0].Value != "A" {
		t.Fatalf("spec = %+v", spec)
	}
}

func TestReadACCSpecAcceptsCLIEnvelope(t *testing.T) {
	p := filepath.Join(t.TempDir(), "q.json")
	raw := `{"ok":true,"data":{"kind":"acc.query.template","query":{"limit":2}}}`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	spec, err := readACCSpec(p)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Limit != 2 {
		t.Fatalf("limit = %d", spec.Limit)
	}
}
