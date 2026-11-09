package form42f

import "testing"

const layoutXML = `<?xml version="1.0" encoding="UTF-8"?>
<Form name="demo.tmp" build="3.21.03" width="100" height="50">
  <Page name="p1" text="page_1" comment="cmt_page_1">
    <Grid name="g1">
      <Label name="lbl_a" text="lbl_a" posX="1" posY="0" width="10"/>
      <FormField name="t.a" colName="a" sqlType="VARCHAR(10)" fieldId="1" tabIndex="1">
        <Edit posX="12" posY="0" width="8"/>
      </FormField>
      <FormField name="t.a_desc" colName="a_desc" sqlType="VARCHAR(80)" fieldId="2" tabIndex="2">
        <Label posX="21" posY="0" width="10"/>
      </FormField>
    </Grid>
    <Table name="t2">
      <TableColumn name="t.b" colName="b" sqlType="CHAR" fieldId="3" tabIndex="3" text="lbl_b"/>
    </Table>
    <Page name="p2" text="demo.page_2">
      <FormField name="t.c" colName="c" sqlType="DATE" fieldId="4" tabIndex="4">
        <DateEdit posX="4" posY="0"/>
      </FormField>
    </Page>
  </Page>
  <FormField name="t.g" colName="g" sqlType="CHAR" fieldId="5">
    <Edit posX="1" posY="1"/>
  </FormField>
</Form>`

func TestParseLayoutPagesAndLabels(t *testing.T) {
	l, err := ParseLayout("demo", layoutXML)
	if err != nil {
		t.Fatal(err)
	}
	if l.Form.DeclaredName != "demo.tmp" || l.Form.Build != "3.21.03" {
		t.Fatalf("form 元数据不对: %+v", l.Form)
	}
	if len(l.Pages) != 2 {
		t.Fatalf("pages = %d, want 2", len(l.Pages))
	}
	p1, p2 := l.Pages[0], l.Pages[1]
	if p1.ID != "p1" || p1.Depth != 1 || p1.TextKey != "page_1" {
		t.Fatalf("p1 = %+v", p1)
	}
	if p2.ParentID != "p1" || p2.Depth != 2 || p2.Path != "p1/p2" {
		t.Fatalf("p2 = %+v", p2)
	}
	if len(p1.FieldIndexes) != 3 || len(p2.FieldIndexes) != 1 {
		t.Fatalf("页面字段数 p1=%d p2=%d", len(p1.FieldIndexes), len(p2.FieldIndexes))
	}

	byCol := map[string]F42LayoutField{}
	for _, f := range l.Fields {
		byCol[string(l.Form.Fields[f.Index].ColName)] = f
	}
	if f := byCol["a"]; f.LabelKey != "lbl_a" || f.LabelSource != "label-token" {
		t.Fatalf("a 标签 = %+v", f)
	}
	if f := byCol["a_desc"]; f.LabelKey != "" || f.LabelSource != "none" {
		t.Fatalf("a_desc 不应抢走 a 的 Label: %+v", f)
	}
	if f := byCol["b"]; f.LabelKey != "lbl_b" || f.LabelSource != "field" {
		t.Fatalf("b 标签 = %+v", f)
	}
	if f := byCol["g"]; f.PageID != "" {
		t.Fatalf("页面外字段应无 PageID: %+v", f)
	}
}

func TestParseLayoutRejectsBadXML(t *testing.T) {
	if _, err := ParseLayout("demo", "<Form><Page>"); err == nil {
		t.Fatal("截断 XML 应报错")
	}
}
