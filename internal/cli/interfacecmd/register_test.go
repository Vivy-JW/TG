package interfacecmd

import "testing"

// TestFieldsDefaultsAndFull 钉住字段列表的默认摘要：默认 30 条，--full 才取全部。
func TestFieldsDefaultsAndFull(t *testing.T) {
	c := newFieldsCmd()
	if got := c.Flags().Lookup("limit").DefValue; got != "30" {
		t.Fatalf("--limit 默认值 = %q, want 30", got)
	}
	if c.Flags().Lookup("full") == nil {
		t.Fatal("fields 缺少 --full")
	}
}
