package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
)

// splitHeader 把输出切成 `# ` 注释行与主体 —— 与 tt debug sql 的
// TestWriteSQLResult 同一套判据:注释**只能出现在主体之前**。
func splitHeader(t *testing.T, out string) (comments []string, body []string) {
	t.Helper()
	seenBody := false
	for _, ln := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if strings.HasPrefix(ln, "# ") {
			if seenBody {
				t.Errorf("注释行出现在主体之后: %q\n完整输出:\n%s", ln, out)
			}
			comments = append(comments, strings.TrimPrefix(ln, "# "))
			continue
		}
		seenBody = true
		body = append(body, ln)
	}
	return comments, body
}

func sampleMeta() Meta {
	return Meta{
		Source: "live", Env: "主机正式区", SSHHost: "172.16.1.109", Zone: "36",
		Ent: 99, Account: "dsdemo", AccountSource: "ent(gzou_t)",
		Target: "172.16.1.109:1521/t35prd", Dialect: "oracle", Route: "client-direct",
		TotalRows: 2, Returned: 2, Elapsed: 1.5,
	}
}

// TestEmitJSONEnvelope 默认形态是 JSON 信封:环境信息平铺在顶层,载荷在 data。
func TestEmitJSONEnvelope(t *testing.T) {
	var buf bytes.Buffer
	if err := Emit(&buf, Options{
		Format: FormatJSON, Meta: sampleMeta(),
		Data: []map[string]string{{"编号": "std-00006"}},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	if strings.Contains(strings.TrimSpace(buf.String()), "\n") {
		t.Fatalf("JSON 信封应是紧凑单行: %s", buf.String())
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("输出不是合法 JSON: %v\n%s", err, buf.String())
	}
	for k, want := range map[string]any{
		"ok": true, "env": "主机正式区", "ent": float64(99),
		"account": "dsdemo", "accountSource": "ent(gzou_t)",
		"target": "172.16.1.109:1521/t35prd", "dialect": "oracle",
		"totalRows": float64(2), "returned": float64(2),
	} {
		if got[k] != want {
			t.Errorf("信封 %q = %v, 期望 %v", k, got[k], want)
		}
	}
	data, ok := got["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("data 应是 1 条数组: %#v", got["data"])
	}
	if row, _ := data[0].(map[string]any); row["编号"] != "std-00006" {
		t.Errorf("data[0] 不符: %#v", data[0])
	}
}

// TestEmitCSVHeaderBeforeBody CSV 的 `# ` 头必须在主体之前,且主体能被标准 CSV 读回。
func TestEmitCSVHeaderBeforeBody(t *testing.T) {
	var buf bytes.Buffer
	if err := Emit(&buf, Options{
		Format: FormatCSV, Meta: sampleMeta(),
		Columns: []string{"编号", "文本"},
		Rows:    [][]string{{"std-00006", "输入的资料已存在"}, {"azz-00041", "含,逗号\"与\"引号"}},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	comments, body := splitHeader(t, buf.String())
	if len(comments) < 2 {
		t.Fatalf("至少应有环境行与行数行, got %v", comments)
	}
	if !strings.Contains(comments[0], "环境 主机正式区") || !strings.Contains(comments[0], "区域 36") {
		t.Errorf("第一行应回显环境与区域: %q", comments[0])
	}
	// 企业编号与账号同在一行 —— 它们是一条链,拆开看会被误读成两件独立的事
	if !strings.Contains(comments[1], "企业(ENT) 99 → 账号 dsdemo") {
		t.Errorf("第二行应把企业与账号连在一句话里: %q", comments[1])
	}
	if !strings.Contains(comments[1], "来源 ent(gzou_t)") {
		t.Errorf("第二行要说清账号是怎么定下来的: %q", comments[1])
	}

	recs, err := csv.NewReader(strings.NewReader(strings.Join(body, "\n"))).ReadAll()
	if err != nil {
		t.Fatalf("主体不是合法 CSV: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("应 1 行表头 + 2 行数据, got %d", len(recs))
	}
	if recs[0][0] != "编号" || recs[0][1] != "文本" {
		t.Errorf("表头不符: %v", recs[0])
	}
	if recs[2][1] != "含,逗号\"与\"引号" {
		t.Errorf("引号/逗号未正确转义: %q", recs[2][1])
	}
}

// TestEmitMetaSkipsEmptySegments 一个值都没有的段不打印。
// 本地源没有 SSH/区域,绝不能出现 "环境  · SSH  · 区域 " 这种空壳行。
func TestEmitMetaSkipsEmptySegments(t *testing.T) {
	var buf bytes.Buffer
	if err := Emit(&buf, Options{
		Format:  FormatCSV,
		Meta:    Meta{Source: "local", Env: "local", Route: "local-sqlite", LocalDBPath: `D:\x.db`},
		Columns: []string{"表名"}, Rows: [][]string{{"dzea_t"}},
	}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	comments, _ := splitHeader(t, buf.String())
	for _, c := range comments {
		if strings.Contains(c, "SSH") || strings.Contains(c, "区域") {
			t.Errorf("本地源不该出现 SSH/区域段: %q", c)
		}
		if strings.HasSuffix(strings.TrimSpace(c), "·") {
			t.Errorf("不该留下悬空的分隔符: %q", c)
		}
	}
}

// TestEmitCountLine 行数行的四种情形:0 行要点名"上错号也是 0 行";
// 截断要给总数,且**区分**库侧限流的"≥下界"与本地按条数截断的"共 N 条"。
func TestEmitCountLine(t *testing.T) {
	cases := []struct {
		name string
		m    Meta
		want string
	}{
		{"零行", Meta{TotalRows: 0}, "先核对上面的企业编号"},
		{"正常", Meta{TotalRows: 7}, "行数 7"},
		// 库侧包装多取一行判"还有更多",所以 totalRows 只是下界
		{"库侧截断", Meta{TotalRows: 5000, Returned: 200, Truncated: true, ServerLimited: true}, "≥5000"},
		// 本地按返回条数截断:总数是准的,且必须说清只回了多少
		{"本地截断", Meta{TotalRows: 28005, Returned: 20, Truncated: true}, "共 28005 条,只回了前 20 条"},
		{"本地截断带落盘", Meta{TotalRows: 28005, Returned: 20, Truncated: true, LocalPath: `D:\x.json`}, "完整结果见落盘文件"},
	}
	for _, c := range cases {
		if got := c.m.countLine(); !strings.Contains(got, c.want) {
			t.Errorf("%s: countLine() = %q, 期望含 %q", c.name, got, c.want)
		}
	}
}

// TestEmitNoResultSet 没有列定义时不写 CSV 主体(只留一句说明)。
func TestEmitNoResultSet(t *testing.T) {
	var buf bytes.Buffer
	if err := Emit(&buf, Options{Format: FormatCSV, Meta: sampleMeta()}); err != nil {
		t.Fatalf("Emit: %v", err)
	}
	_, body := splitHeader(t, buf.String())
	for _, ln := range body {
		if !strings.HasPrefix(ln, "# ") {
			t.Errorf("没有列定义时不该有主体行: %q", ln)
		}
	}
}

// TestWriteError 错误信封与成功信封共用同一组环境键。
func TestWriteError(t *testing.T) {
	var buf bytes.Buffer
	err := Errorf(CodeTableMissing, ExitMissing, "本地库没有该族数据").
		WithHint("先 tt dict db sync --env <环境>").WithMeta(sampleMeta())
	if e := WriteError(&buf, err); e != nil {
		t.Fatalf("WriteError: %v", e)
	}
	if strings.Contains(strings.TrimSpace(buf.String()), "\n") {
		t.Fatalf("错误信封应是紧凑单行: %s", buf.String())
	}
	var got map[string]any
	if e := json.Unmarshal(buf.Bytes(), &got); e != nil {
		t.Fatalf("错误输出不是合法 JSON: %v", e)
	}
	if got["ok"] != false || got["code"] != CodeTableMissing || got["exitCode"] != float64(ExitMissing) {
		t.Errorf("错误信封不符: %#v", got)
	}
	if got["hint"] != "先 tt dict db sync --env <环境>" {
		t.Errorf("hint 应单独给出: %#v", got["hint"])
	}
	if got["env"] != "主机正式区" {
		t.Errorf("错误也要带上出事的那个环境: %#v", got)
	}
	if err.ExitCode() != ExitMissing {
		t.Errorf("ExitCode() = %d, 期望 %d", err.ExitCode(), ExitMissing)
	}
}

// TestWriteTableCJKWidth 中文表头的分隔线按显示宽度算(一个字两列),
// 否则分隔行会比表头短一截,整行看着是歪的。
func TestWriteTableCJKWidth(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, []string{"编号", "文本"}, [][]string{{"std-00006", "xx"}}); err != nil {
		t.Fatalf("WriteTable: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("输出行数不足: %q", buf.String())
	}
	sep := strings.Fields(lines[1])[0] // 第一条分隔线
	// 表头"编号"两个汉字占 4 列,数据 "std-00006" 9 列,取大者 9
	if len(sep) != 9 {
		t.Errorf("分隔线宽度 = %d, 期望 9(按显示宽度取两列最大值): %q", len(sep), lines[1])
	}
}

// TestDetailIsVisibleInJSONAndCSV 钉住 detail 契约：JSON 顶层可见，CSV 头也回显。
func TestDetailIsVisibleInJSONAndCSV(t *testing.T) {
	var buf bytes.Buffer
	m := sampleMeta()
	m.Detail = "summary"
	if err := Emit(&buf, Options{Format: FormatJSON, Meta: m, Data: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["detail"] != "summary" {
		t.Fatalf("JSON 应带 detail: %#v", got)
	}

	buf.Reset()
	if err := Emit(&buf, Options{Format: FormatCSV, Meta: m, Columns: []string{"列"}, Rows: [][]string{{"值"}}}); err != nil {
		t.Fatal(err)
	}
	comments, _ := splitHeader(t, buf.String())
	found := false
	for _, c := range comments {
		if c == "视图 summary" {
			found = true
		}
	}
	if !found {
		t.Fatalf("CSV 头应回显视图: %v", comments)
	}
}

// TestTruncHintOnlyAffectsHumanHeader 截断后的“下一步”因命令而异：
// ACC 用 query.limit，SQL 用 --limit 0。TruncHint 只改人读/CSV 头，不进 JSON。
func TestTruncHintOnlyAffectsHumanHeader(t *testing.T) {
	m := Meta{TotalRows: 22, Returned: 3, Truncated: true, TruncHint: "加大 query.limit 或 query.offset 继续"}
	line := m.countLine()
	if !strings.Contains(line, "加大 query.limit") || strings.Contains(line, "--limit 0") {
		t.Fatalf("人读头应使用命令自己的提示: %q", line)
	}
	if !strings.Contains(line, "共 22 条") || !strings.Contains(line, "只回了前 3 条") {
		t.Fatalf("截断头必须同时给总数和返回数: %q", line)
	}

	var buf bytes.Buffer
	if err := Emit(&buf, Options{Format: FormatJSON, Meta: m, Data: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "truncHint") {
		t.Fatalf("TruncHint 不应进入 JSON 信封: %s", buf.String())
	}
}

// TestWriteMetaKeepsNotes 人读模式也必须保留 notes；--human 只换渲染，不丢信息。
func TestWriteMetaKeepsNotes(t *testing.T) {
	var buf bytes.Buffer
	m := Meta{
		TotalRows: 22, Returned: 3, Truncated: true,
		TruncHint: "加大 query.limit 或 query.offset 继续",
		Notes:     []string{"已包含停用值"},
	}
	if err := WriteMeta(&buf, m); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"# 行数 3", "加大 query.limit", "# 注:已包含停用值"} {
		if !strings.Contains(out, want) {
			t.Errorf("人读头缺少 %q:\n%s", want, out)
		}
	}
}
