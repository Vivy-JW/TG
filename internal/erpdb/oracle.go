package erpdb

// OracleConnector:Oracle 在线查询连接(go-ora 驱动,纯 Go)。
// 与 KingbaseConnector 对齐 Connector 接口:Type/ServerVersion/Query/Close。
// 查询统一走只读单语句校验(见 checkReadOnlyStatement),列值一律转字符串返回。

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"tt/internal/dbconfig"

	_ "github.com/sijms/go-ora/v2" // 注册 sql 驱动 "oracle"
)

// OracleConnector 用 go-ora(database/sql)连 Oracle,查询文本化返回。
type OracleConnector struct {
	addr string
	typ  string
	db   *sql.DB
}

// OpenOracle 连接 Oracle。统一显式模型:host/port/service(service 也接受 database 旧值);
// service 为空时用 host 的 database 字段兜底。
func OpenOracle(ctx context.Context, c dbconfig.Connection) (*OracleConnector, error) {
	server, port, service := c.Host, c.Port, c.Svc()
	if port == 0 {
		port = 1521
	}
	if server == "" || service == "" {
		return nil, fmt.Errorf("Oracle 连接缺少地址或服务名 (需填 host/port/service)")
	}
	dsn := goOraDSN(server, port, service, c.User, c.Password)
	db, err := sql.Open("oracle", dsn)
	if err != nil {
		return nil, fmt.Errorf("解析 Oracle 连接配置失败: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 Oracle 失败 (%s:%d/%s): %w", server, port, service, err)
	}
	return &OracleConnector{addr: c.Address(), typ: c.Type, db: db}, nil
}

func goOraDSN(host string, port int, service, user, password string) string {
	return fmt.Sprintf("oracle://%s:%s@%s/%s", urlEscape(user), urlEscape(password),
		net.JoinHostPort(host, strconv.Itoa(port)), urlEscape(service))
}

func urlEscape(s string) string {
	r := strings.NewReplacer("%", "%25", ":", "%3A", "@", "%40", "/", "%2F", "?", "%3F")
	return r.Replace(s)
}

func (o *OracleConnector) Type() string { return o.typ }

func (o *OracleConnector) ServerVersion(ctx context.Context) (string, error) {
	_, rows, err := o.queryReadOnly(ctx, "SELECT banner FROM v$version WHERE ROWNUM = 1")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return "", fmt.Errorf("v$version 未返回结果")
	}
	return rows[0][0], nil
}

// Query 只读单语句查询;列值统一转字符串(与 Kingbase 文本协议语义一致)。
//
// 两层防线都在这个函数里，**顺序不能反**（契约：`internal/erpdb/README.md` 的"两层"）：
//
//	① 文本层（`checkReadOnlyStatement` → `internal/safesql.Check`）：挡 `set_config`
//	   这类"把只读自己拆掉"的语句，也挡前导注释绕过、多语句、特权视图、ddl 等；
//	② 库侧（`BeginTx` → `SET TRANSACTION READ ONLY`）：让真去写数据的语句由**库**拒
//	   （常规 DML 得 `ORA-01456`）。
//
// 为什么 Oracle 侧是"每次查询包一层事务"而不是像金仓那样一行连接参数 ——
// **驱动明确拒绝** `sql.TxOptions{ReadOnly:true}`（go-ora 的 `Connection.BeginTx`
// 直接 `return nil, errors.New("readonly transaction is not supported")`），
// 所以只能自己发 `SET TRANSACTION READ ONLY`。
func (o *OracleConnector) Query(ctx context.Context, sql string) ([]string, [][]string, error) {
	if err := checkReadOnlyStatement(sql); err != nil {
		return nil, nil, err
	}
	return o.queryReadOnly(ctx, sql)
}

// queryReadOnly 是**带库侧只读事务**的执行体（`Query` 与 `ServerVersion` 共用）。
//
// # `SET TRANSACTION READ ONLY` 的两条硬约束（都是 Oracle 官方文档原文）
//
//  1. "it must be the **first statement** in your transaction" —— 所以顺序是
//     `BeginTx` → `SET TRANSACTION READ ONLY` → 查询 → `Rollback`，**中间不能先发别的**；
//  2. DDL 会**隐式提交**、从而终结这个只读事务（所以它也不是万能兜底）。
//
// # 代价（如实写，见 README）
//
//   - **多一次往返**：`SET TRANSACTION` 是一条真 RPC，与查询共用同一个 `ctx`
//     （不额外引入超时；副作用是 ctx 可能在建事务那一步就超时）；
//   - `database/sql` 的 `Tx` **独占一条连接**直到 `Rollback`（今天是 `db.QueryContext`
//     在 `rows.Close()` 时就还池）。本包没设过池参数，走 Go 默认（`MaxOpenConns` 无限、
//     `MaxIdleConns=2`），所以并发查询下只是多占几条连接，不会死锁；
//   - **所以 `defer tx.Rollback()` 必须写**：漏了就是连接泄漏（`Tx` 不还池）。
//
// 返回前 `rows` 已经读完/关闭，`Rollback` 打在一条没有未完成结果的连接上 —— 安全。
func (o *OracleConnector) queryReadOnly(ctx context.Context, sql string) ([]string, [][]string, error) {
	// **不能**传 `&sql.TxOptions{ReadOnly: true}`：go-ora 会直接报
	// "readonly transaction is not supported"（见上面的注释）。
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	// 只读事务的终局一律是 Rollback（没有要提交的东西）；**它也是连接归还点** ——
	// `database/sql` 的 `Tx` 独占一条连接直到这里，漏了这句就是连接泄漏。
	// 写成闭包而不是裸的 `defer tx.Rollback()`，只为把"返回值刻意丢掉"写明白：
	// 回滚失败改不了这个函数的成败（查询结果已经定了），但也不该看起来像忘了处理。
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, "SET TRANSACTION READ ONLY"); err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}

	rows, err := tx.QueryContext(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	var result [][]string
	vals := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for rows.Next() {
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, err
		}
		row := make([]string, len(cols))
		for i, v := range vals {
			row[i] = scanString(v)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return cols, result, nil
}

// scanString 把 database/sql 扫到的任意值文本化(供 Oracle 走 interface 扫描)。
func scanString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func (o *OracleConnector) Close() {
	if o.db != nil {
		o.db.Close()
	}
}
