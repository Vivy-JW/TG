package db

// 只读 SQL 的输出契约（从旧 tt debug sql 的测试搬来，随渲染一起搬）：
//  1. 头部必须回显本次查询的环境与企业编号 —— 读错它会把"上错号/上错环境"读成"数据不存在"；
//  2. CSV 主体必须纯净（只有表头 + 数据行，杂质全在 # 注释里）。

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/debug"
)

func testResult() *debug.DBQueryResult {
	return &debug.DBQueryResult{
		Ent: 99, Account: "dsdemo", AccountSource: "ent(config)",
		Env: "开发环境", SSHHost: "10.0.0.1", Zone: "31",
		DBTarget: "10.0.0.2:1521/t100dev", Dialect: "oracle",
		Columns:   []string{"BMAA001", "BMAASTUS"},
		Rows:      [][]string{{"FCPU010100003", "Y"}, {"FCPU010100004", "N"}},
		TotalRows: 2, ServerLimited: true, Elapsed: 1.67,
	}
}

func renderCSV(t *testing.T, res *debug.DBQueryResult) (comments, body string) {
	t.Helper()
	oldCSV, oldJSON := common.CSV, common.JSON
	common.CSV, common.JSON = true, false
	t.Cleanup(func() { common.CSV, common.JSON = oldCSV, oldJSON })

	var b bytes.Buffer
	if err := writeSQLResult(&b, res); err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	var head []string
	var rows []string
	seenBody := false
	for _, ln := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if strings.HasPrefix(ln, "# ") {
			if seenBody {
				t.Fatalf("CSV 主体之后又出现注释行（注释必须全在头部）:\n%s", b.String())
			}
			head = append(head, ln)
			continue
		}
		seenBody = true
		rows = append(rows, ln)
	}
	return strings.Join(head, "\n"), strings.Join(rows, "\n")
}

func TestWriteSQLResultHeader(t *testing.T) {
	head, _ := renderCSV(t, testResult())
	for _, must := range []string{"开发环境", "10.0.0.1", "31", "企业(ENT) 99", "dsdemo", "oracle", "10.0.0.2:1521/t100dev"} {
		if !strings.Contains(head, must) {
			t.Errorf("头部缺少 %q:\n%s", must, head)
		}
	}
	if !strings.Contains(head, "企业(ENT) 99 → 账号 dsdemo") {
		t.Errorf("「企业→账号」应在同一行:\n%s", head)
	}
}

func TestWriteSQLResultCSVBody(t *testing.T) {
	_, body := renderCSV(t, testResult())
	recs, err := csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatalf("主体应当是合法 CSV: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("应有 1 行表头 + 2 行数据，得到 %d 行: %v", len(recs), recs)
	}
	if strings.Join(recs[0], ",") != "BMAA001,BMAASTUS" || recs[1][0] != "FCPU010100003" || recs[1][1] != "Y" {
		t.Errorf("表头或数据错了: %v", recs)
	}

	// 值里的逗号/引号/换行要按 RFC4180 转义
	res := testResult()
	res.Columns = []string{"a", "b"}
	res.Rows = [][]string{{`x,y`, `has"quote`}, {"line1\nline2", ""}}
	_, body = renderCSV(t, res)
	recs, err = csv.NewReader(strings.NewReader(body)).ReadAll()
	if err != nil {
		t.Fatalf("转义后仍应是合法 CSV: %v", err)
	}
	if recs[1][0] != "x,y" || recs[1][1] != `has"quote` || recs[2][0] != "line1\nline2" {
		t.Errorf("含逗号/引号/换行的值被写坏了: %v", recs)
	}
}
