package gp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/output"
)

func TestNormalizeLang(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"", "", true},
		{"0", "0", true},
		{"zh_TW", "0", true},
		{"zh-Hant", "0", true},
		{"2", "2", true},
		{"zh_CN", "2", true},
		{"zh-Hans", "2", true},
		{"1", "1", true},
		{"zh_JP", "", false},
		{"traditional", "", false},
	}
	for _, c := range cases {
		got, err := normalizeLang(c.in)
		if (err == nil) != c.ok {
			t.Fatalf("normalizeLang(%q) err=%v, want ok=%v", c.in, err, c.ok)
		}
		if got != c.want {
			t.Errorf("normalizeLang(%q)=%q, want %q", c.in, got, c.want)
		}
	}
}

func TestSplitCols(t *testing.T) {
	got := splitCols(" zta01, zta02 ,,zta03 ")
	want := []string{"zta01", "zta02", "zta03"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("splitCols=%v, want %v", got, want)
	}
}

func TestLinkRowsMarksEntry(t *testing.T) {
	rows := linkRows([]linkRow{
		{Module: "cbm", LinkedProg: "abmi600", LinkFlag: "Y", Entry: true},
		{Module: "abm", LinkedProg: "abmi600_2", LinkFlag: "Y"},
	})
	if len(rows) != 2 || rows[0][3] != "Y" || rows[1][3] != "" {
		t.Fatalf("linkRows 入口标记不对: %v", rows)
	}
}

func TestCellOutOfRange(t *testing.T) {
	if got := cell([]string{"a"}, 2); got != "" {
		t.Fatalf("cell 越界应返回空串，得 %q", got)
	}
}

func TestOneLineRemark(t *testing.T) {
	got := oneLine("Table #, (Table name)\r\nEx:ima_file, zz_file")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("备注仍有换行: %q", got)
	}
	if got != "Table #, (Table name) Ex:ima_file, zz_file" {
		t.Fatalf("备注 = %q", got)
	}
}

func TestBriefTableOutput(t *testing.T) {
	cols := []columnInfo{{
		Seq: 1, Name: "zta01", NameCN: "档案代码", Type: "varchar2",
		Length: 15, Precision: 2, Scale: 1, Nullable: "N",
		Default: "x", Remark: "很长的备注", Format: "999",
	}}

	brief := briefColumns(cols)
	if len(brief) != 1 || brief[0].Name != "zta01" || brief[0].Length != 15 || brief[0].Nullable != "N" {
		t.Fatalf("briefColumns 不对: %+v", brief)
	}

	rows := briefRows(cols)
	if len(rows) != 1 || len(rows[0]) != 6 {
		t.Fatalf("briefRows 应为 6 列: %v", rows)
	}
	if rows[0][1] != "zta01" || rows[0][3] != "varchar2" || rows[0][5] != "N" {
		t.Fatalf("briefRows 内容不对: %v", rows[0])
	}

	raw, err := json.Marshal(briefTableResult{
		Table:   briefTableInfo{Name: "zta_file", ColumnCount: 1},
		Columns: brief,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{`"remark"`, `"format"`, `"default"`, `"precision"`, `"scale"`} {
		if strings.Contains(string(raw), banned) {
			t.Fatalf("brief JSON 不应包含 %s: %s", banned, raw)
		}
	}
}

func TestSetHumanDefault(t *testing.T) {
	oldFormat, oldHuman, oldJSON, oldCSV := common.Format, common.Human, common.JSON, common.CSV
	defer func() {
		common.Format, common.Human, common.JSON, common.CSV = oldFormat, oldHuman, oldJSON, oldCSV
	}()

	reset := func() {
		common.Format, common.Human, common.JSON, common.CSV = "json", false, false, false
	}

	// 没显式指定输出格式：默认给 table。
	reset()
	setHumanDefault(&cobra.Command{Use: "table"})
	if got := common.OutputFormat(); got != output.FormatTable {
		t.Fatalf("默认格式 = %v，want table", got)
	}

	// 显式 --format json：保持 json。
	reset()
	cmd := &cobra.Command{Use: "table"}
	cmd.Flags().String("format", "json", "")
	if err := cmd.Flags().Set("format", "json"); err != nil {
		t.Fatal(err)
	}
	setHumanDefault(cmd)
	if got := common.OutputFormat(); got != output.FormatJSON {
		t.Fatalf("显式 --format json = %v，want json", got)
	}

	// 显式 --json：保持 json。
	reset()
	common.JSON = true
	setHumanDefault(&cobra.Command{Use: "table"})
	if got := common.OutputFormat(); got != output.FormatJSON {
		t.Fatalf("显式 --json = %v，want json", got)
	}
}
