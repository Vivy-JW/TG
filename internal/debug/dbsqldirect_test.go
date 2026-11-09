package debug

// 客户端直连这条路的**契约**测试：与服务器 sqlplus 路线同一套行数/截断/字段上限，
// 以及"建连失败可回退、查询失败不可回退"的分界。全部离线（不需要真库）。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tt/internal/dbconfig"
	"tt/internal/safesql"
)

type fakeDirect struct {
	typ  string
	cols []string
	rows [][]string
	err  error
	got  string
}

func (f *fakeDirect) Type() string { return f.typ }

func (f *fakeDirect) Query(_ context.Context, sql string) ([]string, [][]string, error) {
	f.got = sql
	return f.cols, f.rows, f.err
}

func manyRows(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{"x"}
	}
	return rows
}

func TestDirectQueryTruncatesAtMaxRows(t *testing.T) {
	f := &fakeDirect{typ: "oracle", cols: []string{"X"}, rows: manyRows(safesql.MaxRows + 50)}
	res, err := directQuery(context.Background(), f, "select x from t")
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	if !strings.Contains(strings.ToUpper(f.got), "ROWNUM <= 201") {
		t.Errorf("直连也必须在库侧限行（多取一行判“还有更多”），实发: %s", f.got)
	}
	if res.TotalRows != safesql.MaxRows+50 {
		t.Errorf("totalRows 应是库侧回来的行数 %d，得到 %d", safesql.MaxRows+50, res.TotalRows)
	}
	if len(res.Rows) != safesql.MaxRows || !res.Truncated || res.TruncReason != "rows" {
		t.Errorf("应截到 %d 行并标 truncated/rows，得到 %d 行 truncated=%v reason=%q",
			safesql.MaxRows, len(res.Rows), res.Truncated, res.TruncReason)
	}
	if !res.ServerLimited {
		t.Error("包装过的语句应标 serverLimited=true")
	}
}

func TestDirectQueryUnwrappableWithStaysUnlimited(t *testing.T) {
	// 以 WITH 开头的语句不能安全包装（Oracle 11g 不支持子查询里的 WITH）：
	// 必须如实标注“未在库侧限流”，且不能因此少给行。
	sql := "WITH a AS (select 1 x from dual) select * from a"
	f := &fakeDirect{typ: "oracle", cols: []string{"X"}, rows: manyRows(safesql.MaxRows + 5)}
	res, err := directQuery(context.Background(), f, sql)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	if f.got != sql {
		t.Errorf("不可包装的语句应原样执行，实发: %s", f.got)
	}
	if res.ServerLimited {
		t.Error("没包装就不该标 serverLimited")
	}
	if len(res.Rows) != safesql.MaxRows+5 {
		t.Errorf("未包装时不该按 200 行截，得到 %d 行", len(res.Rows))
	}
	if !hasNote(res.Notes, "未在数据库侧限流") {
		t.Errorf("必须如实标注未在库侧限流，notes=%v", res.Notes)
	}
}

func TestDirectQueryCapsValue(t *testing.T) {
	big := strings.Repeat("x", safesql.MaxValue*2)
	f := &fakeDirect{typ: "oracle", cols: []string{"BIG"}, rows: [][]string{{big}}}
	res, err := directQuery(context.Background(), f, "select big from t")
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	got := res.Rows[0][0]
	if !strings.HasPrefix(got, big[:safesql.MaxValue]) || !strings.HasSuffix(got, "…(本字段被截断)") {
		t.Errorf("超长字段应被截到 %d 字节并加标记，得到长度 %d", safesql.MaxValue, len(got))
	}
	if !hasNote(res.Notes, "被截断") {
		t.Errorf("截断要进 notes，notes=%v", res.Notes)
	}
}

func TestDirectQueryPassesQueryErrorThrough(t *testing.T) {
	boom := errors.New("ORA-00942: table or view does not exist")
	f := &fakeDirect{typ: "oracle", err: boom}
	_, err := directQuery(context.Background(), f, "select x from nope")
	if !errors.Is(err, boom) {
		t.Fatalf("查询错误应原样透出，得到 %v", err)
	}
	// 查询失败不是“连不上”：不能被包装成可回退的错误。
	var unreachable *DirectUnreachableError
	if errors.As(err, &unreachable) {
		t.Error("查询失败不该被包成 DirectUnreachableError（那会让调用方白跑第二遍）")
	}
}

func TestRunDirectQueryHonoursReadonlySwitch(t *testing.T) {
	off := false
	c := dbconfig.Connection{Type: "oracle", Host: "127.0.0.1", Service: "nope", ReadonlySQL: &off}
	_, err := RunDirectQuery(context.Background(), c, "select 1 from dual", 5)
	if err == nil || !strings.Contains(err.Error(), "只读 SQL 已在配置里关闭") {
		t.Fatalf("只读开关关掉时应直接拒绝（不建连），得到 %v", err)
	}
}

func TestRunDirectQueryRejectsNonSelectBeforeConnecting(t *testing.T) {
	// Host 是空的：真去连会先报“缺地址”；这里必须是**先**被只读闸门拦下。
	c := dbconfig.Connection{Type: "oracle"}
	_, err := RunDirectQuery(context.Background(), c, "delete from imaa_t", 5)
	if err == nil || !strings.Contains(err.Error(), "只读") && !strings.Contains(strings.ToLower(err.Error()), "select") {
		t.Fatalf("写语句应被只读闸门拒绝，得到 %v", err)
	}
	var unreachable *DirectUnreachableError
	if errors.As(err, &unreachable) {
		t.Error("被闸门拒绝不该是“连不上”")
	}
}

func hasNote(notes []string, sub string) bool {
	for _, n := range notes {
		if strings.Contains(n, sub) {
			return true
		}
	}
	return false
}
