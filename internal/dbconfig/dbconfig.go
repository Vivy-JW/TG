// Package dbconfig 定义统一的数据库连接类型（config.json hosts.sshs[].db）。
// 每个 SSH 环境一对一挂一个库：显式 类型/主机/端口/服务名(oracle)|库名(kingbase)，
// 账号全部在 Accounts 列表 —— 运行时用哪个账号由 TOPENT 决定（服务器侧经 gzou_t
// 解析账号名后在列表查密码，未收录回退 账号=密码）；客户端直连（ping/在线查询/sync）
// 与 TOPENT 无关，取列表首项（DialCred）。
// 服务器侧执行工具（sqlplus/ksql 路径）一律自动探测，不存入配置。
//
// 合并自 TDebug/dbconfig 与 TDictCli/dbconfig；两者除注释外只差 TDebug 多出的
// ReadonlySQL 字段，故以 TDebug 版为准（它是超集）。
package dbconfig

import "fmt"

// SystemAccount 查 T100 系统表(gzou_t 企业目录 / gzzz_t 作业解析 / wsfa_t 接口日志)
// 固定用的账号名。
//
// 为什么写死一个名字而不是"取账号清单首项":这三张表是**跨企业共享**的系统表,
// 它们的归属只由 T100 的安装惯例决定(ds);账号清单的首项是"人顺手把哪个账号写在了前面",
// 两者恰好相同不等于同一条规则——换一份配置、调一下顺序,"查企业目录"就会悄悄换到
// 另一个 schema 上,而结果依旧是一堆数字(或零行),看不出换了地方。
//
// 密码依旧查账号清单(按名字 ds),未收录回退 T100 惯例 账号=密码。
const SystemAccount = "ds"

// ViaSSH SSH 端口转发隧道：客户端不可达 DB、但 DB 对 SSH 服务器（或同网）可达时，
// 客户端先建到 SSH 的隧道（本地 bindPort → remoteHost:port），驱动改连 127.0.0.1:bindPort。
type ViaSSH struct {
	Host       string `json:"host"`                 // SSH 服务器地址
	Port       int    `json:"port,omitempty"`       // SSH 端口(0=22)
	User       string `json:"user"`                 // SSH 账号
	Password   string `json:"password,omitempty"`   // SSH 密码
	BindPort   int    `json:"bindPort,omitempty"`   // 本地监听端口(0=自动选空闲)
	RemoteHost string `json:"remoteHost,omitempty"` // DB 在 SSH 侧的真实地址(默认取连接 host)
	RemotePort int    `json:"remotePort,omitempty"` // DB 在 SSH 侧的真实端口(默认取连接 port)
}

// EffectiveRemote 返回隧道远端转发目标（缺省回退到连接自身 host/port）。
func (v *ViaSSH) EffectiveRemote(fallbackHost string, fallbackPort int) (string, int) {
	h := v.RemoteHost
	if h == "" {
		h = fallbackHost
	}
	p := v.RemotePort
	if p <= 0 {
		p = fallbackPort
	}
	return h, p
}

// DBAcct 数据库账号凭据（账号即 schema 名，如 ds/dsdemo/dsdata）。
type DBAcct struct {
	Account  string `json:"account"`  // 账号/schema 名
	Password string `json:"password"` // 登录密码
}

// Connection 数据库连接（内嵌于 hosts.sshs[].db，与 SSH 环境一对一）。
// oracle 用 Service(SERVICE_NAME)，kingbase 用 Database(库名)。
type Connection struct {
	Type     string `json:"type"`               // "kingbase" | "oracle"
	Host     string `json:"host,omitempty"`     // 主机地址(客户端与服务器侧均可达)
	Port     int    `json:"port,omitempty"`     // 端口(0=默认 1521/54321)
	Service  string `json:"service,omitempty"`  // oracle: SERVICE_NAME(如 t35prd)
	Database string `json:"database,omitempty"` // kingbase: 库名(如 topprd)
	// Accounts 账号列表（全部账号，无主账号标记）。服务器侧按 TOPENT→gzou_t 解析出的
	// 账号名在列表中查密码；客户端直连取列表首项（DialCred）。
	Accounts []DBAcct `json:"accounts,omitempty"`
	ViaSSH   *ViaSSH  `json:"viaSsh,omitempty"` // 可省：客户端不可达时经 SSH 隧道转发再直连

	// ReadonlySQL 是否允许 AI 执行只读 SQL（排查业务数据用）。
	// 用指针是为了区分"没配"（nil = 默认开启）与"显式关掉"（false）——
	// 与 debug.PersistBreakpoints 同款范式。关掉时端点直接 403，文案告诉用户改哪里。
	ReadonlySQL *bool `json:"readonlySql,omitempty"`

	// 内部直连凭据槽位（不持久化）：客户端直连前由 DialCred 填入
	User     string `json:"-"`
	Password string `json:"-"`

	// Account 本次要用哪个账号（不持久化）。空 = 客户端直连惯例：取账号列表首项。
	//
	// 非空时按它取密码（PasswordFor），未收录则回退 T100 惯例 账号=密码 ——
	// 这是"按企业编号查 gzou_t 解析出的账号"落到连接上的方式，与服务器侧
	// （internal/debug 的 acctConnStr）同一条规则。两条路径必须对一个环境给出
	// 同一个账号，否则同一张字典表会被两个 schema 读到而没人察觉。
	Account string `json:"-"`
}

// ReadonlySQLEnabled 只读 SQL 是否开启（未配置 = 开启）
func (c *Connection) ReadonlySQLEnabled() bool {
	if c == nil || c.ReadonlySQL == nil {
		return true
	}
	return *c.ReadonlySQL
}

// DialCred 返回客户端直连凭据 = 账号列表首项（直连与 TOPENT 无关）；
// ok=false 表示尚未配置任何账号。
func (c *Connection) DialCred() (user, pass string, ok bool) {
	if c == nil || len(c.Accounts) == 0 || c.Accounts[0].Account == "" {
		return "", "", false
	}
	return c.Accounts[0].Account, c.Accounts[0].Password, true
}

// FillDialCred 客户端直连前填充内部凭据槽。
// Account 非空时用它（密码查清单，未收录回退 账号=密码）；否则取列表首项。
func (c *Connection) FillDialCred() error {
	if c.Account != "" {
		pass, ok := c.PasswordFor(c.Account)
		if !ok {
			pass = c.Account // T100 惯例：未收录的账号，密码=账号
		}
		c.User, c.Password = c.Account, pass
		return nil
	}
	u, p, ok := c.DialCred()
	if !ok {
		return fmt.Errorf("数据库连接未配置账号(accounts)")
	}
	c.User, c.Password = u, p
	return nil
}

// PasswordFor 按账号名查密码（服务器侧调试：TOPENT→gzou_t 解析出账号后取密码）；
// ok=false 表示未收录（调用方回退 账号=密码 惯例）
func (c *Connection) PasswordFor(account string) (string, bool) {
	if c == nil {
		return "", false
	}
	for _, a := range c.Accounts {
		if a.Account == account && a.Password != "" {
			return a.Password, true
		}
	}
	return "", false
}

// Svc 返回 oracle 的服务名（兼容 database 字段旧值兜底）
func (c *Connection) Svc() string {
	if c.Service != "" {
		return c.Service
	}
	return c.Database
}

// DefaultPort 返回该类型数据库的缺省端口。
func (c *Connection) DefaultPort() int {
	if c.Type == "kingbase" {
		return 54321
	}
	return 1521
}

// Address 返回展示用地址 host:port/svc(库名)
func (c *Connection) Address() string {
	if c.Host == "" {
		return ""
	}
	port := c.Port
	if port == 0 {
		port = c.DefaultPort()
	}
	return fmt.Sprintf("%s:%d/%s", c.Host, port, c.Svc())
}
