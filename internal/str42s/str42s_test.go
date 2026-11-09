package str42s

import (
	"encoding/binary"
	"testing"
)

// encode 造一份最小合法 .42s：两段索引只占位（解析器不读它们）。
func encode(pairs [][2]string) []byte {
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
	return b
}

func TestParsePairs(t *testing.T) {
	data := encode([][2]string{
		{"page_1", "基本数据"},
		{"bsft001_wf.page_4", "工艺工单明细"},
		{"lbl_wfapwfseq1", "项序"},
	})
	tbl, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(tbl.Entries))
	}
	if got, ok := tbl.Lookup("lbl_wfapwfseq1"); !ok || got != "项序" {
		t.Fatalf("Lookup = %q, %v", got, ok)
	}
	if got, ok, amb := tbl.Unique("page_1"); !ok || amb || got != "基本数据" {
		t.Fatalf("Unique = %q, %v, %v", got, ok, amb)
	}
}

func TestParseWithoutTrailingNUL(t *testing.T) {
	data := encode([][2]string{{"a", "甲"}})
	data = data[:len(data)-1] // 去掉最后一个 NUL
	if _, err := Parse(data); err != nil {
		t.Fatalf("末尾无 NUL 应接受: %v", err)
	}
}

func TestParseRejectsTruncated(t *testing.T) {
	data := encode([][2]string{{"a", "甲"}, {"b", "乙"}})
	if _, err := Parse(data[:len(data)-3]); err == nil {
		t.Fatal("截断文件应报错")
	}
	if _, err := Parse([]byte{1, 2}); err == nil {
		t.Fatal("短头应报错")
	}
}

func TestUniqueAmbiguous(t *testing.T) {
	tbl, err := Parse(encode([][2]string{
		{"x", "甲"}, {"x", "乙"}, {"y", "同"}, {"y", "同"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, amb := tbl.Unique("x"); !ok || !amb {
		t.Fatalf("x 应歧义: ok=%v amb=%v", ok, amb)
	}
	if got, ok, amb := tbl.Unique("y"); !ok || amb || got != "同" {
		t.Fatalf("y 重复同值不算歧义: %q %v %v", got, ok, amb)
	}
	if _, ok, _ := tbl.Unique("missing"); ok {
		t.Fatal("不存在的 token 应 ok=false")
	}
}
