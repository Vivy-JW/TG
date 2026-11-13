package config

// GP 环境的补丁语义（只改提交的字段）。
//
// 这一层原来住在 internal/gp（已删，执行面在 internal/web），但补丁合并是纯配置运算：不碰网络、
// 不认识 HTTP、不依赖 job 或会话。设置页的 500ms 自动保存全靠它 —— 前端只发
// "这次真正变了的字段"，这里保证没提交的键（口令、service、dictDir…）不被清掉。

import (
	"strings"

	"tg/internal/dbconfig"
)

// EnvPatch 是 GP 环境的补丁。指针字段为 nil = 本次没提交，保留旧值；
// 非 nil 的空串/0 = 明确改成空。
type EnvPatch struct {
	Name        *string `json:"name,omitempty"`
	Host        *string `json:"host,omitempty"`
	Port        *int    `json:"port,omitempty"`
	User        *string `json:"user,omitempty"`
	Password    *string `json:"password,omitempty"`
	Instance    *string `json:"instance,omitempty"`
	Workspace   *string `json:"workspace,omitempty"`
	CustRoot    *string `json:"custRoot,omitempty"`
	Company     *string `json:"company,omitempty"`
	CompanyName *string `json:"companyName,omitempty"`
	// CompanySyncedAt 只在「从服务器刷新」成功后由前端回填，RFC3339。
	CompanySyncedAt *string  `json:"companySyncedAt,omitempty"`
	MirrorDir       *string  `json:"mirrorDir,omitempty"`
	DictDir         *string  `json:"dictDir,omitempty"`
	Zone            *string  `json:"zone,omitempty"`
	DB              *DBPatch `json:"db,omitempty"`
}

// DBPatch 是数据库那一节的“只改提交的字段”补丁。
type DBPatch struct {
	Type        *string            `json:"type,omitempty"`
	Host        *string            `json:"host,omitempty"`
	Port        *int               `json:"port,omitempty"`
	Service     *string            `json:"service,omitempty"`
	Database    *string            `json:"database,omitempty"`
	Accounts    *[]dbconfig.DBAcct `json:"accounts,omitempty"`
	ReadonlySQL *bool              `json:"readonlySql,omitempty"`
}

// PatchFromEnv 是旧版“整条 env”请求的兼容转换：所有字段都算作提交，
// 但空密码仍按“不改”处理，避免老客户端一次保存把口令抹掉。
func PatchFromEnv(e NamedSsh) EnvPatch {
	p := EnvPatch{
		Name: &e.Name, Host: &e.Host, Port: &e.Port, User: &e.User,
		Workspace: &e.Workspace, CustRoot: &e.CustRoot, Company: &e.Company,
		CompanyName: &e.CompanyName, CompanySyncedAt: &e.CompanySyncedAt,
		MirrorDir: &e.MirrorDir, DictDir: &e.DictDir, Zone: &e.Zone,
	}
	// 旧客户端的“整条 env”没有 instance；空值不能当“清空实例”处理，
	// 否则会把已有的 workspace/custRoot 一起清掉。非空才交给推导。
	if e.Instance != "" {
		p.Instance = &e.Instance
	}
	if e.Password != "" {
		p.Password = &e.Password
	}
	if e.DB != nil {
		p.DB = &DBPatch{
			Type: &e.DB.Type, Host: &e.DB.Host, Port: &e.DB.Port,
			Service: &e.DB.Service, Database: &e.DB.Database,
			Accounts: &e.DB.Accounts, ReadonlySQL: e.DB.ReadonlySQL,
		}
	}
	return p
}

// ApplyEnvPatch 把补丁合并进 dst。它只碰补丁里出现的字段，
// 因此“只改主机”不会顺带清掉 service / 密码 / dictDir / mirrorDir。
func ApplyEnvPatch(dst *NamedSsh, p EnvPatch) {
	if p.Name != nil {
		dst.Name = strings.TrimSpace(*p.Name)
	}
	if p.Host != nil {
		dst.Host = strings.TrimSpace(*p.Host)
	}
	if p.Port != nil {
		dst.Port = *p.Port
	}
	if p.User != nil {
		dst.User = strings.TrimSpace(*p.User)
	}
	// GP 表单：密码留空 = 不改（不是“清空”）。
	if p.Password != nil && *p.Password != "" {
		dst.Password = *p.Password
	}
	if p.Instance != nil {
		inst := strings.TrimSpace(*p.Instance)
		dst.Instance = inst
		if inst == "" {
			// 清空实例名 = 不再保留旧实例推导出来的路径，避免旧路径继续生效。
			dst.Workspace, dst.CustRoot = "", ""
		} else {
			dst.Workspace, dst.CustRoot = RootsForInstance(inst)
		}
	}
	// 旧客户端（或高级配置）仍可直接给路径；给了 instance 时以 instance 推导为准。
	if p.Workspace != nil && p.Instance == nil {
		dst.Workspace = strings.TrimSpace(*p.Workspace)
	}
	if p.CustRoot != nil && p.Instance == nil {
		dst.CustRoot = strings.TrimSpace(*p.CustRoot)
	}
	// 营运据点：空 = 回到 "ds"（只影响直查；本地字典库同步固定按 ds）。
	if p.Company != nil {
		dst.Company = strings.ToLower(strings.TrimSpace(*p.Company))
	}
	// 据点名字只是显示用的抄本；随据点一起清空，避免代号换了名字还留着旧的。
	if p.CompanyName != nil {
		dst.CompanyName = strings.TrimSpace(*p.CompanyName)
	}
	// 刷新时间原样存（前端给 RFC3339），不裁时区。
	if p.CompanySyncedAt != nil {
		dst.CompanySyncedAt = strings.TrimSpace(*p.CompanySyncedAt)
	}
	if p.MirrorDir != nil {
		dst.MirrorDir = strings.TrimSpace(*p.MirrorDir)
	}
	if p.DictDir != nil {
		dst.DictDir = strings.TrimSpace(*p.DictDir)
	}
	if p.Zone != nil {
		dst.Zone = strings.TrimSpace(*p.Zone)
	}
	if p.DB != nil {
		applyDBPatch(dst, p.DB)
	}
}

// applyDBPatch 合并数据库补丁。DB 不存在时新建一节，字段逐个覆盖。
func applyDBPatch(dst *NamedSsh, p *DBPatch) {
	if p == nil {
		return
	}
	if dst.DB == nil {
		dst.DB = &dbconfig.Connection{Type: "oracle", Port: 1521}
	}
	if p.Type != nil {
		dst.DB.Type = strings.TrimSpace(*p.Type)
	}
	if p.Host != nil {
		dst.DB.Host = strings.TrimSpace(*p.Host)
	}
	if p.Port != nil {
		dst.DB.Port = *p.Port
	}
	if p.Service != nil {
		dst.DB.Service = strings.TrimSpace(*p.Service)
	}
	if p.Database != nil {
		dst.DB.Database = strings.TrimSpace(*p.Database)
	}
	if p.ReadonlySQL != nil {
		v := *p.ReadonlySQL
		dst.DB.ReadonlySQL = &v
	}
	if p.Accounts != nil {
		MergeAccounts(dst.DB, *p.Accounts)
	}
	if dst.DB.Type == "" {
		dst.DB.Type = "oracle"
	}
}

// MergeAccounts 按账号名合并口令：
//   - 单账号 ↔ 单账号：视为同一条（允许改账号名），空密码沿用旧密码；
//   - 多账号：同账号更新、空密码保留旧密码，未提交的账号原样保留。
//
// 这样 GP 表单只编辑首个账号时，不会把配置里其余账号一起抹掉。
func MergeAccounts(db *dbconfig.Connection, incoming []dbconfig.DBAcct) {
	if db == nil || len(incoming) == 0 {
		return
	}
	if len(db.Accounts) == 1 && len(incoming) == 1 {
		a := incoming[0]
		a.Account = strings.TrimSpace(a.Account)
		if a.Account == "" {
			return
		}
		if a.Password == "" {
			a.Password = db.Accounts[0].Password
		}
		db.Accounts = []dbconfig.DBAcct{a}
		return
	}
	out := append([]dbconfig.DBAcct(nil), db.Accounts...)
	for _, in := range incoming {
		account := strings.TrimSpace(in.Account)
		if account == "" {
			continue
		}
		found := -1
		for i := range out {
			if out[i].Account == account {
				found = i
				break
			}
		}
		if found >= 0 {
			if in.Password == "" {
				in.Password = out[found].Password
			}
			out[found] = dbconfig.DBAcct{Account: account, Password: in.Password}
			continue
		}
		out = append(out, dbconfig.DBAcct{Account: account, Password: in.Password})
	}
	db.Accounts = out
}

// ValidInstance 实例名只允许普通路径段，不能带 / 或空格。
func ValidInstance(inst string) bool {
	if inst == "" {
		return false
	}
	for _, r := range inst {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

// RootsForInstance 实例名 → GP 的 $TOP / $CUST。
// 现场约定是 /u1/<实例>/tiptop 与 /u1/<实例>/topcust（TOPTEST 即 /u1/toptest/...）。
func RootsForInstance(inst string) (workspace, custRoot string) {
	inst = strings.TrimSpace(inst)
	if inst == "" {
		return "", ""
	}
	base := "/u1/" + inst
	return base + "/tiptop", base + "/topcust"
}

// InstanceRoot 环境对应的实例目录（/u1/<实例>），也就是 $TOP 与 $CUST 的公共父目录。
//
// 为什么需要它：GP 的源码是两棵并列的树（/u1/<实例>/tiptop 与 /u1/<实例>/topcust），
// 而"看服务器文件""读源码"这类命令天生只有一个根。把根提到实例目录，两棵树就成了
// 它的两个子目录：单根模型不用为第二棵树开特例，越界判定也仍然只有一个前缀。
//
// 有 instance 就按约定推 /u1/<实例>；没有就退回 workspace 的上一级
// （/u1/toptest/tiptop → /u1/toptest）；都没有返回空串，由调用方报错。
func InstanceRoot(e NamedSsh) string {
	if inst := strings.TrimSpace(e.Instance); inst != "" {
		return "/u1/" + inst
	}
	ws := strings.TrimSuffix(strings.TrimSpace(e.Workspace), "/")
	if i := strings.LastIndex(ws, "/"); i > 0 {
		return ws[:i]
	}
	return ""
}

// InstanceFromWorkspace 旧配置里只有 workspace 时，从 /u1/<实例>/tiptop 反推实例名。
func InstanceFromWorkspace(workspace string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(workspace), "/"), "/")
	for i, p := range parts {
		if strings.EqualFold(p, "tiptop") && i > 0 {
			return parts[i-1]
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// MergeTestEnv 把“表单里没填的项”用已保存值补齐，供测试连接使用。
//
// 测试连接不是保存：用户可能只改了 DB 主机、口令框留空（表示不改），
// 这时必须拿已保存的口令去测；否则每次测试都要重新输入密码，且自动保存
// 清空密码框后测试必然失败（实测）。
func MergeTestEnv(req, saved NamedSsh) NamedSsh {
	env := req
	if env.Name == "" {
		env.Name = saved.Name
	}
	if env.Password == "" {
		env.Password = saved.Password
	}
	if env.Port == 0 {
		env.Port = saved.Port
	}
	if env.Host == "" {
		env.Host = saved.Host
	}
	if env.User == "" {
		env.User = saved.User
	}
	if env.Instance == "" {
		env.Instance = saved.Instance
	}
	if env.Instance == "" {
		env.Instance = InstanceFromWorkspace(saved.Workspace)
	}
	if env.Workspace == "" {
		env.Workspace = saved.Workspace
	}
	if env.CustRoot == "" {
		env.CustRoot = saved.CustRoot
	}
	if env.Company == "" {
		env.Company = saved.Company
	}
	if env.CompanyName == "" {
		env.CompanyName = saved.CompanyName
	}
	if env.Instance != "" {
		env.Workspace, env.CustRoot = RootsForInstance(env.Instance)
	}

	if env.DB == nil {
		if saved.DB == nil {
			return env
		}
		db := *saved.DB
		db.Accounts = append([]dbconfig.DBAcct(nil), saved.DB.Accounts...)
		env.DB = &db
		return env
	}

	db := *env.DB
	db.Accounts = append([]dbconfig.DBAcct(nil), env.DB.Accounts...)
	if saved.DB != nil {
		if db.Type == "" {
			db.Type = saved.DB.Type
		}
		if db.Host == "" {
			db.Host = saved.DB.Host
		}
		if db.Port == 0 {
			db.Port = saved.DB.Port
		}
		if db.Service == "" {
			db.Service = saved.DB.Service
		}
		if db.Database == "" {
			db.Database = saved.DB.Database
		}
	}
	// 账号：请求没给账号时整组沿用；给了账号但口令为空时按账号名沿用旧口令。
	if len(db.Accounts) == 0 || strings.TrimSpace(db.Accounts[0].Account) == "" {
		if saved.DB != nil {
			db.Accounts = append([]dbconfig.DBAcct(nil), saved.DB.Accounts...)
		}
	} else if db.Accounts[0].Password == "" && saved.DB != nil {
		for _, a := range saved.DB.Accounts {
			if a.Account == db.Accounts[0].Account {
				db.Accounts[0].Password = a.Password
				break
			}
		}
	}
	env.DB = &db
	return env
}
