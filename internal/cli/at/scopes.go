package at

// 作用域表：**这张表是 --at 语义的唯一出处**。新增命令时到这里认领一档；
// 路径按“相对根命令”写，子命令会沿路径向上继承（`dict` 认领的档，`dict r.t` 自动继承）。
//
// 认领的判据不是“它需不需要环境”，而是“它把环境拿去干什么”：
//   EnvEnt    —— 要连库，并且**按企业解析账号**（gzou_t → gzou003），或按企业收窄；
//   Env       —— 只连服务器/库，不按企业解析账号；
//   EnvFilter —— 会话寻址：环境只用来在会话池里收窄，@企业 拒绝（会话自带企业）；
//   不写      —— None：不接受 --at（给了直接报错，不静默忽略）。

import (
	"strings"

	"github.com/spf13/cobra"
)

var scopes = map[string]Scope{
	// ---- 数据库：连通性测试与只读 SQL ----
	"db sql":   Env, // GP：一个库一个账号，@ 后面没有企业也没有据点可写
	"db probe": Env,
	"db ping":  Env,
	// ---- 建立连接 / 解析账号（<环境>@<企业>）----
	"session new":   EnvEntReq,
	"debug start":   EnvEntReq,
	"debug wsdebug": EnvEntReq,
	// debug sql / debug db 是已弃用的转发壳，身份跟 tg db sql / tg db probe 一致：
	// 只要环境，账号取 db.accounts[0]（GP 没有企业编号那层）。
	"debug sql":  Env,
	"debug db":   Env,
	"debug ents": EnvEnt,
	// dict 的查询子命令（r.t/r.q/prog/msg）要有企业才能解析账号
	"dict": EnvEnt,
	// 但 dict 下的“连接/镜像/本地文件”这几支不要企业
	"dict db":         Env,
	"dict db status":  None, // 只看本地 SQLite，不碰环境
	"dict mirror":     Env,  // 隐藏转发到 tg file mirror（下版删）
	"dict mirror dir": Env,  // 转发到 tg file mirror dir；带 --at 时只改该环境

	// ---- 服务器文件：只读，只认环境 ----
	"file":            Env,
	"file mirror dir": Env, // 不带 --at=全局根；带 --at=该环境单独目录
	// 本地目录/本地文件类：连环境都不需要
	"dict spill": None,

	// ---- 只连环境（<环境>）----
	"debug wslogs": Env,
	"debug probe":  Env,
	"session ls":   Env,
	"interface":    Env,
	"outline":      Env,
	// 顶层 site：连库读 azp_file，只认环境（据点编号是位置参数，不走 --at）
	"site": Env,

	// ---- 会话寻址：<环境> 只用来收窄 ----
	"debug exec":      EnvFilter,
	"debug quit":      EnvFilter,
	"debug interrupt": EnvFilter,
	"debug locate":    EnvFilter,
	"debug mode":      EnvFilter,
	"debug topent":    EnvFilter,
	"debug stop":      EnvFilter,
	"debug wait":      EnvFilter,
	"debug why":       EnvFilter,
	"debug bp":        EnvFilter,
}

// ScopeOf 取命令的作用域：先看自己，再沿路径向上找。
func ScopeOf(cmd *cobra.Command) Scope {
	p := pathOf(cmd)
	for {
		if s, ok := scopes[p]; ok {
			return s
		}
		if p == "" {
			return None
		}
		if i := strings.LastIndex(p, " "); i >= 0 {
			p = p[:i]
		} else {
			p = ""
		}
	}
}
