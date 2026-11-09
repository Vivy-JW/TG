package screen

import (
	"encoding/binary"
	"strings"
	"testing"

	"tt/internal/form42f"
	"tt/internal/str42s"
)

func table(pairs [][2]string) *str42s.Table {
	var b []byte
	head := make([]byte, 4)
	binary.LittleEndian.PutUint32(head, uint32(len(pairs)))
	b = append(b, head...)
	b = append(b, make([]byte, 8*len(pairs))...)
	for _, p := range pairs {
		b = append(b, p[0]...)
		b = append(b, 0)
		b = append(b, p[1]...)
		b = append(b, 0)
	}
	t, err := str42s.Parse(b)
	if err != nil {
		panic(err)
	}
	return t
}

const formXML = `<Form name="demo.tmp" build="1">
 <Page name="p1" text="page_1">
  <Grid name="g">
   <Label name="lbl_a" text="lbl_a" posX="1" posY="0"/>
   <FormField name="t.a" colName="a" sqlType="VARCHAR(10)" fieldId="1">
    <Edit posX="12" posY="0"/>
   </FormField>
   <FormField name="t.a_desc" colName="a_desc" sqlType="VARCHAR(80)" fieldId="2">
    <Label posX="21" posY="0"/>
   </FormField>
  </Grid>
 </Page>
 <FormField name="t.g" colName="g" sqlType="CHAR" fieldId="3"/>
</Form>`

func TestBuildResolvesLabelsAndKeepsGlobal(t *testing.T) {
	layout, err := form42f.ParseLayout("demo", formXML)
	if err != nil {
		t.Fatal(err)
	}
	v := Build("demo", "m", layout, table([][2]string{
		{"page_1", "基本资料"},
		{"lbl_a", "甲字段"},
	}))
	if len(v.Pages) != 1 || v.Pages[0].Label != "基本资料" {
		t.Fatalf("page = %+v", v.Pages)
	}
	if v.TotalFields != 3 || v.GlobalFieldCount != 1 {
		t.Fatalf("fields=%d global=%d", v.TotalFields, v.GlobalFieldCount)
	}
	byCol := map[string]FieldView{}
	for _, f := range v.Fields {
		byCol[f.ColName] = f
	}
	if f := byCol["a"]; f.Label != "甲字段" || f.LabelState != "resolved" {
		t.Fatalf("a = %+v", f)
	}
	if f := byCol["a_desc"]; f.LabelState != "unresolved" || f.Reason != "label-not-associated" {
		t.Fatalf("a_desc = %+v", f)
	}
	if f := byCol["g"]; f.PageID != "" || f.LabelState != "unresolved" {
		t.Fatalf("g = %+v", f)
	}
	if len(v.Unresolved) != 2 {
		t.Fatalf("unresolved = %d, want 2: %+v", len(v.Unresolved), v.Unresolved)
	}
}

func TestBuildWithoutLabels(t *testing.T) {
	layout, err := form42f.ParseLayout("demo", formXML)
	if err != nil {
		t.Fatal(err)
	}
	v := Build("demo", "m", layout, nil)
	if len(v.Warnings) != 1 || !strings.Contains(v.Warnings[0], "42s") {
		t.Fatalf("warnings = %v", v.Warnings)
	}
	for _, f := range v.Fields {
		if f.LabelState == "resolved" {
			t.Fatalf("无 42s 不应有 resolved 字段: %+v", f)
		}
	}
}
