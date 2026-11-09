package erpdb

import (
	"context"
	"fmt"
	"time"

	"tt/internal/dbconfig"
	"tt/internal/safesql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connector is a live connection to an ERP database (Kingbase/Oracle).
type Connector interface {
	// Type returns the connection type ("kingbase" | "oracle").
	Type() string
	// ServerVersion runs SELECT version() and returns the server banner.
	ServerVersion(ctx context.Context) (string, error)
	// Query executes a read-only SQL statement and returns columns and rows as strings.
	Query(ctx context.Context, sql string) ([]string, [][]string, error)
	// Close releases the underlying connection pool.
	Close()
}

// Open creates a connector for the given connection config by type.
func Open(ctx context.Context, c dbconfig.Connection) (Connector, error) {
	// 客户端直连凭据 = 账号列表首项(直连与 TOPENT 无关)
	if err := c.FillDialCred(); err != nil {
		return nil, err
	}
	switch c.Type {
	case "kingbase":
		return OpenKingbase(ctx, c)
	case "oracle":
		return OpenOracle(ctx, c)
	default:
		return nil, fmt.Errorf("未知的连接类型 \"%s\"", c.Type)
	}
}

// KingbaseConnector connects to a Kingbase (人大金仓) server via the PostgreSQL wire protocol.
type KingbaseConnector struct {
	addr string
	typ  string
	pool *pgxpool.Pool
}

// OpenKingbase connects to the given Kingbase server and verifies the connection.
func OpenKingbase(ctx context.Context, c dbconfig.Connection) (Connector, error) {
	poolCfg, err := pgxpool.ParseConfig(kingbaseDSN(c))
	if err != nil {
		return nil, fmt.Errorf("解析 Kingbase 连接配置失败: %w", err)
	}
	// Use the simple query protocol: results come back in text format, so every
	// column can be scanned into a string (binary format can't decode into
	// **string), and it also avoids Kingbase's extended-protocol quirks.
	poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// ---------- 第一层（库侧）：**会话级**只读 ----------
	//
	// `default_transaction_read_only=on` 是 PostgreSQL/KingbaseES 的 **USERSET** 级 GUC
	// （任何能连上库的普通用户都能设，不需要 DBA 角色）。写进 `RuntimeParams` 之后
	// **池里每条连接一建立就带上**，此后这个会话上的每个事务都是只读的 ——
	// 常规 DML 由**服务器**拒绝，与文本层怎么判无关。
	//
	// 为什么用会话级而不是事务级：与远端那条通道同一个理由（`internal/debug/dbsql.go`
	// 的 `kbReadonlyScript` 写死了这条）—— **过程内一次 `COMMIT` 之后事务级设置就失效了，
	// 会话级还在**。
	//
	// 对连接复用**零影响**：这是启动参数（libpq 的 `options`），在每条连接**建立时**
	// 生效一次；`Query` 照旧走 `pool.Query`，池的复用语义一个字节没变。
	//
	// **这一层不是全部**（见 `checkReadOnlyStatement` 的注释）：它是会话级 GUC，
	// 一条 `SELECT set_config('default_transaction_read_only','off',false)` 就能拆掉它
	// —— 挡住那一句的是**文本层**（`safesql` 的禁用前缀里有 `set_config`）。
	// 两层合起来才成立：缺了文本层，这把锁能被一句 SELECT 打开；缺了这一层，
	// 前导注释/自治事务就绕过去了。
	if poolCfg.ConnConfig.RuntimeParams == nil { // ParseConfig 走 DSN 合成时会给空 map，这里只是别赌它
		poolCfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	poolCfg.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("创建 Kingbase 连接池失败: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("连接 Kingbase 失败 (%s): %w", c.Address(), err)
	}

	return &KingbaseConnector{addr: c.Address(), typ: c.Type, pool: pool}, nil
}

func kingbaseDSN(c dbconfig.Connection) string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable connect_timeout=10",
		c.Host, c.Port, c.Database, c.User, c.Password)
}

func (k *KingbaseConnector) Type() string { return k.typ }

func (k *KingbaseConnector) ServerVersion(ctx context.Context) (string, error) {
	_, rows, err := k.queryReadOnly(ctx, "SELECT version()")
	if err != nil {
		return "", err
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		return "", fmt.Errorf("SELECT version() 未返回结果")
	}
	return rows[0][0], nil
}

// Query executes a read-only SQL statement and returns columns and rows as strings.
// NULL values are returned as empty strings.
//
// 这条路是 `Connector` 接口上唯一的执行口，所以**文本层那道防线就挂在这里**
// （`checkReadOnlyStatement`）：库侧的会话级只读是兜底，它拦不住"把只读自己关掉"
// 的那一句 —— 见那两个函数的注释。
func (k *KingbaseConnector) Query(ctx context.Context, sql string) ([]string, [][]string, error) {
	if err := checkReadOnlyStatement(sql); err != nil {
		return nil, nil, err
	}
	return k.queryReadOnly(ctx, sql)
}

// queryReadOnly 是不带文本层检查的执行体。
//
// 用它的只有 `ServerVersion`（`SELECT version()` 是**代码字面量**，不是调用方给的语句）——
// 于是"检查只做一遍、且做在唯一那个入口上"这件事是显式的，不是靠"没人从别处调"。
func (k *KingbaseConnector) queryReadOnly(ctx context.Context, sql string) ([]string, [][]string, error) {
	rows, err := k.pool.Query(ctx, sql)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fields := rows.FieldDescriptions()
	columns := make([]string, len(fields))
	for i, f := range fields {
		columns[i] = f.Name
	}

	// Simple protocol returns results in text format, so RawValues() gives the
	// raw text bytes per value — no decode/alloc per value. This is an order of
	// magnitude faster than scanning into **string.
	var result [][]string
	for rows.Next() {
		raw := rows.RawValues()
		row := make([]string, len(raw))
		for i, v := range raw {
			if v != nil {
				row[i] = string(v)
			}
		}
		result = append(result, row)
	}
	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}
	return columns, result, nil
}

func (k *KingbaseConnector) Close() {
	if k.pool != nil {
		k.pool.Close()
	}
}

// SelectAllSQL returns the SELECT * statement for the given table in the
// connection's schema. In Oracle/Kingbase the username maps to the schema, so
// the table is qualified as "user.table".
func SelectAllSQL(c dbconfig.Connection, table string) (string, error) {
	if !validIdent(c.User) || !validIdent(table) {
		return "", fmt.Errorf("非法标识符 (user/table): %q / %q", c.User, table)
	}
	return "SELECT * FROM " + c.User + "." + table, nil
}

func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return !(s[0] >= '0' && s[0] <= '9')
}

// checkReadOnlyStatement 是**文本层**那道防线：把要去执行的语句交给
// `internal.safesql.Check` 判一次，nil 才放行。
//
// # 为什么不是本包自己那份前缀匹配（这次换掉的就是它）
//
// 从前这个位置是 `checkReadOnlySQL`：`TrimSpace` + `ToUpper` 看**开头**是不是写操作、
// 看有没有分号。它挡得住"手写的 INSERT/DROP 直接贴进来"，但**挡不住**前面加一段注释的
// 写法（README 的旧版自己承认这一点），更挡不住这一条：
//
//	SELECT set_config('default_transaction_read_only','off',false)
//
// 那一句**能拆掉库侧的只读事务**（见 `OpenKingbase` 的注释）。`safesql` 是词法级扫描
// （先剥注释、按词边界/标识符前缀判），`set_config` 就在它的禁用前缀表里
// —— 原话是"能在一条 SELECT 里把 default_transaction_read_only 改回 off，直接拆掉只读事务"。
//
// # 与库侧只读事务的关系（**这两层合起来才成立**，缺一不可）
//
//   - 库侧只读事务（金仓的 `default_transaction_read_only`、Oracle 的
//     `SET TRANSACTION READ ONLY`）是**兜底**：它不认文本，只要语句真去写数据就被库拒；
//   - 但它拦不住"把只读这个状态本身关掉"的语句 —— 金仓那条是**会话级** GUC，
//     一句 `set_config` 就能改回 off，改完之后这一层的兜底当场不存在；
//   - 所以**挡住 `set_config` 的正是这一处文本层**。反过来，文本层也拦不住
//     "披着 SELECT 的写"（自治事务、函数副作用）—— 那要靠库侧事务。
//
// 一句话：**文本层负责不让只读被拆掉，库侧事务负责让写真的失败**。
// 两层各自都挡不住全部，这不是"双保险"这种客套话，是它们的分工。
//
// 挡不住的（如实写，别粉饰）：自治事务 / 函数副作用（`safesql` 的包注释同款）；
// Oracle 的独立事务与 `EXECUTE IMMEDIATE` 这类"披着 SELECT 的写"。
func checkReadOnlyStatement(sql string) error {
	return safesql.Check(sql)
}
