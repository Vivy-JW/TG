package db

// 批量（--batch）的契约：输入解析、逐条成败不互相拖累、以及“只输出 JSON”。
// 全部离线：执行部分用假会话喂，不连任何库。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/debug"
	"tt/internal/output"
)

type fakeQuerier struct {
	res  map[string]*debug.DBQueryResult
	errs map[string]error
	seen []string
}

func (f *fakeQuerier) Query(_ context.Context, sql string, _ int) (*debug.DBQueryResult, error) {
	f.seen = append(f.seen, sql)
	if err := f.errs[sql]; err != nil {
		return nil, err
	}
	if r, ok := f.res[sql]; ok {
		return r, nil
	}
	return &debug.DBQueryResult{Columns: []string{"X"}, Rows: [][]string{{"1"}}, TotalRows: 1}, nil
}

func TestParseBatchDefaultsAndValidation(t *testing.T) {
	stmts, err := parseBatch([]byte(`[{"sql":"select 1 from dual"},{"id":"b","sql":"select 2 from dual"}]`))
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	if len(stmts) != 2 || stmts[0].ID != "1" || stmts[1].ID != "b" {
		t.Errorf("id 缺省应按序号补（1..N）: %+v", stmts)
	}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空数组", `[]`, "没有任何语句"},
		{"非数组", `{"sql":"select 1 from dual"}`, "JSON 数组"},
		{"缺 sql", `[{"id":"a"}]`, "缺 sql"},
		{"id 重复", `[{"id":"a","sql":"select 1 from dual"},{"id":"a","sql":"select 2 from dual"}]`, "id 重复"},
	}
	for _, c := range cases {
		_, err := parseBatch([]byte(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: 期望错误含 %q，得到 %v", c.name, c.want, err)
		}
	}
}

func TestRunBatchQueriesIsolatesPerStatementFailure(t *testing.T) {
	boom := errors.New("ORA-00942: table or view does not exist")
	q := &fakeQuerier{
		res: map[string]*debug.DBQueryResult{
			"select 1 from dual": {
				Columns: []string{"X"}, Rows: [][]string{{"1"}}, TotalRows: 1, ServerLimited: true,
				Truncated: false,
			},
		},
		errs: map[string]error{"select bad from nope": boom},
	}
	stmts := []batchStmt{
		{ID: "ok", SQL: "select 1 from dual"},
		{ID: "boom", SQL: "select bad from nope"},
		{ID: "write", SQL: "delete from imaa_t"},
	}
	items, failed := runBatchQueries(context.Background(), q, stmts, 30)

	if len(items) != 3 || failed != 2 {
		t.Fatalf("应有 3 条、2 条失败，得到 %d 条、failed=%d", len(items), failed)
	}
	if !items[0].OK || items[0].ID != "ok" || items[0].TotalRows != 1 || !items[0].ServerLimited {
		t.Errorf("成功那条的结果没带全: %+v", items[0])
	}
	if items[1].OK || items[1].Code != "DB_FAILED" || !strings.Contains(items[1].Error, "ORA-00942") {
		t.Errorf("查询失败应记 DB_FAILED 且带原因: %+v", items[1])
	}
	if items[2].OK || items[2].Code != "SQL_REJECTED" {
		t.Errorf("写语句应被只读闸门逐条拒掉（不是整批拒）: %+v", items[2])
	}
	// 被闸门拒的那条不该发到库上：发出去的只有前两条。
	if len(q.seen) != 2 || strings.Contains(strings.Join(q.seen, "|"), "delete") {
		t.Errorf("被拒语句不应执行，实发: %v", q.seen)
	}
}

func TestRunBatchQueriesKeepsOrderAndIDs(t *testing.T) {
	q := &fakeQuerier{}
	stmts := []batchStmt{{ID: "a", SQL: "select 1 from dual"}, {ID: "b", SQL: "select 2 from dual"}}
	items, failed := runBatchQueries(context.Background(), q, stmts, 30)
	if failed != 0 || items[0].ID != "a" || items[1].ID != "b" {
		t.Fatalf("顺序与 id 必须原样回: %+v failed=%d", items, failed)
	}
	if q.seen[0] != "select 1 from dual" || q.seen[1] != "select 2 from dual" {
		t.Errorf("执行顺序应保持输入顺序: %v", q.seen)
	}
}

func TestQueryBatchRefusesNonJSONFormats(t *testing.T) {
	oldCSV, oldJSON := common.CSV, common.JSON
	common.CSV, common.JSON = true, false
	t.Cleanup(func() { common.CSV, common.JSON = oldCSV, oldJSON })

	err := QueryBatch("q.json", 99, 30, SQLRouteAuto)
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Exit != 2 || oe.Code != "USAGE" {
		t.Fatalf("--csv 下的批量应是 USAGE/2，得到 %v", err)
	}
	if !strings.Contains(oe.Error(), "批量只输出 JSON") {
		t.Errorf("错误要说明原因，得到 %q", oe.Error())
	}
}
