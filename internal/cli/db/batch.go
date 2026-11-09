package db

// 批量只读 SQL：一次建连、一个进程跑 N 条。
//
// 要解决的问题：每条 `tt db sql` 都是一个新进程 + 一次建连（直连 ~140ms，服务器路线
// ~1.3s）。查十个问题就是十次固定成本。批量把固定成本摊成一次：进程、建连、登录各一次，
// 之后每条只付自己的执行时间。
//
// 契约（与全仓批量同形）：
//   - 输入是 JSON 数组 [{"id"?: "名字", "sql": "..."}]，**不从 SQL 文本切分号**
//     —— 字符串里的分号、q-quote、注释都不可靠地能切对，切错了就是查错数据；
//   - id 可省（默认按序号 1..N），但必须唯一：回包要靠它取结果；
//   - 单条失败不拖垮整批：该条 ok=false + code/error，其余照跑；
//   - 只输出 JSON（多结果集拼 CSV/表格只会误导），--format csv|table 直接拒。

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"tt/internal/cli/common"
	"tt/internal/debug"
	"tt/internal/output"
	"tt/internal/safesql"
)

// batchStmt 批量输入里的一条语句。
type batchStmt struct {
	ID  string `json:"id"`
	SQL string `json:"sql"`
}

// batchItem 批量回包里的一条结果。
type batchItem struct {
	ID            string     `json:"id"`
	OK            bool       `json:"ok"`
	Columns       []string   `json:"columns,omitempty"`
	Rows          [][]string `json:"rows,omitempty"`
	TotalRows     int        `json:"totalRows"`
	Returned      int        `json:"returned"`
	Truncated     bool       `json:"truncated,omitempty"`
	TruncReason   string     `json:"truncReason,omitempty"`
	ServerLimited bool       `json:"serverLimited,omitempty"`
	Elapsed       float64    `json:"elapsedSeconds,omitempty"`
	Code          string     `json:"code,omitempty"`
	Error         string     `json:"error,omitempty"`
}

// parseBatch 解析并校验批量输入。
func parseBatch(data []byte) ([]batchStmt, error) {
	var stmts []batchStmt
	if err := json.Unmarshal(data, &stmts); err != nil {
		return nil, fmt.Errorf("批量输入必须是 JSON 数组 [{\"sql\":\"...\"}]：%v", err)
	}
	if len(stmts) == 0 {
		return nil, fmt.Errorf("批量输入里没有任何语句")
	}
	seen := make(map[string]bool, len(stmts))
	for i := range stmts {
		stmts[i].SQL = strings.TrimSpace(stmts[i].SQL)
		if stmts[i].SQL == "" {
			return nil, fmt.Errorf("第 %d 条缺 sql", i+1)
		}
		if stmts[i].ID == "" {
			stmts[i].ID = strconv.Itoa(i + 1)
		}
		if seen[stmts[i].ID] {
			return nil, fmt.Errorf("id 重复: %q（回包按 id 取结果，必须唯一）", stmts[i].ID)
		}
		seen[stmts[i].ID] = true
	}
	return stmts, nil
}

// sqlQuerier 是 runBatchQueries 需要的最小能力面（离线单测用它注入假会话）。
type sqlQuerier interface {
	Query(ctx context.Context, sql string, timeout int) (*debug.DBQueryResult, error)
}

// runBatchQueries 顺序执行，返回逐条结果与失败条数。
//
// 顺序执行而不是并发：同一个会话上并发查，行序与错误归属都会变得难解释；
// 而批量省的是固定成本，不是墙钟并行度。
func runBatchQueries(ctx context.Context, q sqlQuerier, stmts []batchStmt, timeout int) ([]batchItem, int) {
	items := make([]batchItem, 0, len(stmts))
	failed := 0
	for _, st := range stmts {
		it := batchItem{ID: st.ID}
		// 每条都过只读闸门：批量不是绕闸门的后门，被拒的那条只是它自己失败。
		if err := safesql.Check(st.SQL); err != nil {
			it.Code, it.Error = "SQL_REJECTED", err.Error()
			failed++
			items = append(items, it)
			continue
		}
		res, err := q.Query(ctx, st.SQL, timeout)
		if err != nil {
			it.Code, it.Error = "DB_FAILED", err.Error()
			failed++
			items = append(items, it)
			continue
		}
		it.OK = true
		it.Columns, it.Rows = res.Columns, res.Rows
		it.TotalRows, it.Returned = res.TotalRows, len(res.Rows)
		it.Truncated, it.TruncReason = res.Truncated, res.TruncReason
		it.ServerLimited, it.Elapsed = res.ServerLimited, res.Elapsed
		items = append(items, it)
	}
	return items, failed
}

// QueryBatch 执行一批只读 SQL 并渲染。
func QueryBatch(arg string, ent, timeout int, route string) error {
	if common.OutputFormat() != output.FormatJSON {
		return output.Errorf("USAGE", 2, "批量只输出 JSON（多结果集拼成 CSV/表格会误导）：去掉 --format/--csv/--human")
	}
	cfg, err := prepareTarget(ent, route)
	if err != nil {
		return err
	}
	data, err := readBatchInput(arg)
	if err != nil {
		return output.Errorf("USAGE", 2, "%v", err)
	}
	stmts, err := parseBatch(data)
	if err != nil {
		return output.Errorf("USAGE", 2, "%v", err)
	}

	sess, note, err := openSession(cfg, ent, route)
	if err != nil {
		return err
	}
	defer sess.Close()

	start := time.Now()
	items, failed := runBatchQueries(context.Background(), sess, stmts, timeout)
	return printBatchResult(cfg, sess, note, items, failed, time.Since(start).Seconds())
}

// readBatchInput 读批量输入：`-` 表示 stdin，其余按本地文件。
func readBatchInput(arg string) ([]byte, error) {
	if arg == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("读 stdin 失败: %w", err)
		}
		return b, nil
	}
	b, err := os.ReadFile(arg)
	if err != nil {
		return nil, fmt.Errorf("读取批量文件失败: %s: %w", arg, err)
	}
	return b, nil
}

// printBatchResult 渲染批量信封。
//
// totalRows/returned 在批量里是**语句条数**（不是行数）—— 行数是逐条的，只能看
// items[].rows。这一点写进 notes，免得"行数 3"被读成"一共 3 行数据"。
func printBatchResult(cfg *debug.Config, sess *sqlSession, note string, items []batchItem, failed int, elapsed float64) error {
	notes := []string{"批量：top-level totalRows/returned 是语句条数，逐条行数看 items[].totalRows/rows"}
	if note != "" {
		notes = append(notes, note)
	}
	if failed > 0 {
		notes = append(notes, fmt.Sprintf("%d/%d 条失败（看 items[].ok / code / error）；单条失败不拖垮整批", failed, len(items)))
	}
	meta := output.Meta{
		Source: "live", Env: cfg.EnvName(), Zone: cfg.Zone,
		Ent: sess.ent, Account: sess.account, AccountSource: sess.acctSource(),
		Target: sess.target, Dialect: sess.dialect, Route: sess.routeName(), Readonly: true,
		ViaTunnel: sess.viaTun,
		TotalRows: len(items), Returned: len(items), Elapsed: elapsed, Notes: notes,
	}
	return output.Emit(os.Stdout, output.Options{
		Format: output.FormatJSON, Meta: meta,
		Data: map[string]any{
			"statements": len(items), "failed": failed, "items": items,
		},
	})
}

// acctSource、routeName 见 session.go（批量与单条共用同一份落脚点信息）。
