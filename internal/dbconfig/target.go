package dbconfig

// 一次查询的"落脚点"。
//
// 存在的理由:今天"用哪个账号"曾经是隐式的 —— 客户端直连取账号列表首项,
// 服务器侧按企业编号查 gzou_t 解析。两套规则都合理,但结果里看不见,于是同一张
// 字典表可能被两个不同 schema 读到而没有任何信号。
//
// Target 把它变成一等公民:谁连的、连到哪、用的是哪个账号、那个账号怎么定下来的,
// 全部跟着结果一起回去,渲染进查询输出的环境头与 JSON 信封。

// Route 一次查询的执行路径。
const (
	RouteClientDirect = "client-direct"  // 客户端直连(erpdb + 可选 SSH 隧道)
	RouteServerOracle = "server-sqlplus" // 服务器侧 sqlplus,SQL 走 stdin
	RouteServerKB     = "server-ksql"    // 服务器侧 ksql
	RouteLocalSQLite  = "local-sqlite"   // 本地 SQLite 镜像
)

// AccountSource 账号是怎么定下来的(AccountSource 字段的取值)。
const (
	AcctFromPrimary   = "accounts[0]" // 客户端直连:账号列表首项
	AcctFromEnt       = "ent(gzou_t)" // 按企业编号查 gzou_t 解析（快照/现查）
	AcctFromEntConfig = "ent(config)" // 按企业编号查配置里的 ENT 清单（同步过之后的标准路径）
	AcctFromFlag      = "flag"        // 命令行显式指定
)

// Target 一次查询的完整落脚点。
//
// 字段全可空,零值一律表示"这一项不适用"(如本地源没有 SSH / 没有企业编号),
// 渲染时整段跳过,不打印 "环境  · SSH  · 区域 " 这种空壳。
type Target struct {
	Env           string // 环境名
	SSHHost       string // 该环境的 SSH 主机
	SSHUser       string
	Zone          string // 登录区域代码(31 开发 / 35 测试 / 36 正式)
	Topent        string // 环境配的默认企业编号(原样,可能非数字)
	Ent           int    // 本次实际使用的企业编号(0 = 未用)
	Account       string // 本次实际用的库账号/schema
	AccountSource string
	Route         string // Route 之一
	ViaTunnel     bool   // 客户端直连是否经 SSH 隧道
	Path          string // 本地 SQLite 文件路径(仅 RouteLocalSQLite)
	Conn          Connection
}

// NewTarget 由连接与所属环境构造落脚点,账号先按客户端直连惯例取列表首项;
// 解析出企业账号后调 WithAccount 覆盖。
func NewTarget(conn *Connection, env, sshHost, sshUser, zone, topent, route string) *Target {
	t := &Target{Env: env, SSHHost: sshHost, SSHUser: sshUser, Zone: zone,
		Topent: topent, Route: route, AccountSource: AcctFromPrimary}
	if conn != nil {
		t.Conn = *conn
		t.ViaTunnel = conn.ViaSSH != nil
		if u, _, ok := conn.DialCred(); ok {
			t.Account = u
		}
	}
	return t
}

// WithAccount 换上按企业编号解析出的账号,返回自身便于链式。
func (t *Target) WithAccount(acct, source string, ent int) *Target {
	if acct != "" {
		t.Account, t.AccountSource = acct, source
	}
	if ent > 0 {
		t.Ent = ent
	}
	return t
}

// Address 展示用库地址(本地 SQLite 由调用方直接给路径,这里返回连接自身的地址)。
func (t *Target) Address() string { return t.Conn.Address() }

// Dialect 库类型(oracle | kingbase)。
func (t *Target) Dialect() string { return t.Conn.Type }
