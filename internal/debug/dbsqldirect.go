package debug

// 客户端直连的只读查询（Oracle 走 go-ora，金仓走 pgx）。
//
// 为什么要有这条路：服务器 sqlplus 那条路（RunReadonlyQuery）每查一条都要在服务器上
// 新起一次 `bash -lc` + chenv + sqlplus，实测固定成本约 1.0 秒（`select 1 from dual`
// 与 27 万行 count(*) 同速）；同样的库从客户端直连只要 0.2 秒。两条路的差别只在
// **在哪执行**，结果契约必须一模一样，否则"换条路"就等于"换一种答案"：
//   - 单语句只读闸门（safesql.Check，库侧另有 SET TRANSACTION READ ONLY 兜底）；
//   - 库侧限行 200 行（WrapRowLimit）+ 单字段截断（applyValueCap）；
//   - 信封里的 totalRows / returned / truncated / serverLimited 语义与服务器路线同义。
//
// 建连失败与查询失败要分得开（见 DirectUnreachableError）：前者可以回退服务器路线，
// 后者回退只会把同一条错误重跑一遍。

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"tt/internal/dbconfig"
	"tt/internal/dict/live"
	"tt/internal/safesql"
)

// DirectUnreachableError 直连**建不起来**：网络不通、SSH 隧道起不来、登录被拒。
//
// 它刻意与"查询本身失败"（ORA-00942 之类）分开：只有这一类可以回退服务器路线。
// 查询失败回退没有意义——同一条语句在另一条路上会得到同一个错误，而重查询还要
// 再付一次超时（大表上就是两倍代价）。
type DirectUnreachableError struct{ Err error }

func (e *DirectUnreachableError) Error() string { return "客户端直连失败: " + e.Err.Error() }

// Unwrap 让调用方能用 errors.Is/As 继续判底层原因。
func (e *DirectUnreachableError) Unwrap() error { return e.Err }

// RunDirectQuery 从本机直连该企业的账号，执行一条只读 SQL。
//
// 账号由调用方放进 conn.Account（与服务器路线同一个"企业→账号"来源），
// 这里不再解析身份——两条路必须落到同一个 schema，换账号的事只许发生在一处。
func RunDirectQuery(ctx context.Context, conn dbconfig.Connection, sqlText string, timeout int) (*DBQueryResult, error) {
	if err := safesql.Check(sqlText); err != nil {
		return nil, err
	}
	sec := clampSQLTimeout(timeout)
	qctx, cancel := context.WithTimeout(ctx, time.Duration(sec)*time.Second)
	defer cancel()

	d, err := OpenDirectQuery(qctx, conn)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	start := time.Now()
	res, err := d.Query(qctx, sqlText)
	if err != nil {
		return nil, err
	}
	// 与服务器路线的 elapsedSeconds 同义：这次查询从建连到拿全结果用了多久
	// （服务器那边量的也是 sqlplus 启动+登录+执行，不只是库执行时间）。
	res.Elapsed = time.Since(start).Seconds()
	return res, nil
}

// DirectQuery 是**一条进程内可复用**的客户端直连会话。
//
// 存在的理由：建连（TCP + Oracle 登录，实测约 55ms）是固定的，一条 SQL 摊一次
// 完全浪费。批量/常驻场景下先用 OpenDirectQuery 建一次，然后连续 Query —— 这才是
// "把固定成本摊薄"的正确形态，而不是给每条语句各开一个连接。
//
// 只读语义不变：每条语句照样过 safesql.Check，库会话照样是只读事务
// （Oracle 的 SET TRANSACTION READ ONLY 由 erpdb 在每次查询里发）。
type DirectQuery struct {
	l *live.Live
	// closed 保证 Close 真的幂等：池里退出/热替换/上报都可能在同一时刻叫它，
	// 重复关一条 *sql.DB 不致命，但"关了几次"会让"连接被谁抽走"变得没法追。
	closed atomic.Bool
}

// OpenDirectQuery 建一次直连。账号仍由 conn.Account 决定（调用方已按企业解析好）。
func OpenDirectQuery(ctx context.Context, conn dbconfig.Connection) (*DirectQuery, error) {
	if !conn.ReadonlySQLEnabled() {
		return nil, fmt.Errorf("该环境的只读 SQL 已在配置里关闭(debug.sshs[].db.readonlySql=false)")
	}
	l, err := live.Open(ctx, conn)
	if err != nil {
		return nil, &DirectUnreachableError{Err: err}
	}
	return &DirectQuery{l: l}, nil
}

// Dialect 返回底层方言（oracle | kingbase）。
func (d *DirectQuery) Dialect() string { return d.l.Connector().Type() }

// Query 在这条会话上执行一条只读 SQL。
func (d *DirectQuery) Query(ctx context.Context, sqlText string) (*DBQueryResult, error) {
	if err := safesql.Check(sqlText); err != nil {
		return nil, err
	}
	return directQuery(ctx, d.l.Connector(), sqlText)
}

// Close 释放连接（幂等）。
func (d *DirectQuery) Close() {
	if d == nil || !d.closed.CompareAndSwap(false, true) {
		return
	}
	if d.l != nil {
		d.l.Close()
	}
}

// directConnector 是 directQuery 需要的最小能力面。
//
// 用接口而不是具体类型只为一件事：让行数/截断/字段上限这套契约能在**离线单测**里
// 被逐条钉住，而不是只能连着真库手测。
type directConnector interface {
	Type() string
	Query(ctx context.Context, sql string) ([]string, [][]string, error)
}

// directQuery 执行并套用与服务器路线相同的行数/字段上限。
func directQuery(ctx context.Context, q directConnector, sqlText string) (*DBQueryResult, error) {
	sql := sqlText
	serverLimited := false
	if wrapped, ok := safesql.WrapRowLimit(q.Type(), sqlText); ok {
		sql, serverLimited = wrapped, true
	}

	start := time.Now()
	cols, rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	res := &DBQueryResult{
		Dialect:       q.Type(),
		Columns:       cols,
		Rows:          rows,
		TotalRows:     len(rows),
		ServerLimited: serverLimited,
		Elapsed:       time.Since(start).Seconds(),
	}
	if !serverLimited {
		res.Notes = append(res.Notes,
			"未在数据库侧限流(该语句无法安全包装:以 WITH 开头,或自带行限制),行数上限由客户端截断")
	}
	// 包装时多取了一行，专门用来判"还有更多"——与服务器路线同一条规则。
	if serverLimited && len(rows) > safesql.MaxRows {
		res.Rows = rows[:safesql.MaxRows]
		res.Truncated = true
		res.TruncReason = "rows"
	}
	applyValueCap(res)
	return res, nil
}
