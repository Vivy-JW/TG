// Package db 是“数据库”命令组：**连通性测试** + **只读 SQL**。
//
// TG 只面向 TOP GP，身份只有“环境”一个维度：一个库一个登录账号
// （config.json 的 hosts.sshs[].db.accounts[0]），没有 T100 的 ENT 企业清单，
// 所以 `--at` 只写 `<环境>`（写 `@<据点>` 会被 at 层拒掉 —— 查数据用哪个 schema
// 由账号决定，见下）。
//
// 一个 GP 库里装几十套表（ds / digiwin / cof_demo …，清单见 `tg site list`），
// 裸表名解析到“登录账号的 schema”：`select count(*) from ima_file` 在 digiwin
// 账号下读的是 DIGIWIN 的 850 行，要读别的库自己写 `cof_demo.ima_file`。
// 输出信封里的 schema 字段写的就是这个落点，别把它当成“你选的据点”。
//
// 三条命令的边界：
//
//	sql    对库执行一条只读 SQL（业务数据）
//	ping   从**本机**连一次库（客户端直连，可经 viaSsh 隧道）
//	probe  从**服务器**连一次库（SSH + sqlplus/ksql）
package db

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/dbconfig"
	"tt/internal/debug"
)

// Register 把 db 组挂到根命令。
func Register(root *cobra.Command) {
	g := &cobra.Command{
		Use:   "db",
		Short: "数据库：连通性测试与只读 SQL",
		Long: `数据库：测试连不连得上，以及直接执行一条只读 SQL。

  tg db sql     执行一条只读 SQL（单条 SELECT/WITH，只读事务，最多 200 行；
                默认客户端直连，建连失败回退服务器 sqlplus，--route 可指定）
  tg db sql --batch <json>  一次建连跑 N 条（JSON 数组 [{id?,sql}]，只输出 JSON）
  tg db ping    从本机连一次库（客户端直连；支持 viaSsh 隧道）
  tg db probe   从服务器连一次库（SSH + sqlplus/ksql）

身份只有一个维度：--at <环境>。账号取环境配置里的数据库账号（accounts[0]）——
GP 一个库一个登录账号，没有 T100 的 ENT 企业清单。

**裸表名落在登录账号的 schema**（信封的 schema 字段写的就是它）：
  tg db sql --at 22主机 "select count(*) from ima_file"             # DIGIWIN 的
  tg db sql --at 22主机 "select count(*) from cof_demo.ima_file"   # COF 的
不知道有哪些库：tg site list --env 22主机（资料库编号那一列就是 schema）。

退出码：0 成功 / 2 用法或身份错（环境不在配置、账号没配）/ 3 只读闸门拒绝 /
4 连接或查询失败。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	g.AddCommand(newSQLCmd(), newPingCmd(), newProbeCmd())
	root.AddCommand(g)
}

// loadCfg 读配置并切到本次点名的环境（--at/--env 已落到 common.Env）。
//
// 与 tt debug 的读法一致：CloneEnv 把该环境的 SSH/Zone/DB 装进运行时字段，
// 但不 ApplyDefaultEnv —— 那次合并会把“没点名就落首条”的老行为带回来。
func loadCfg() (*debug.Config, string, error) {
	env := strings.TrimSpace(common.Env)
	if env == "" {
		return nil, "", fmt.Errorf("需要 --at <环境> 或 --env <环境>（可用：tg env list 看环境名）")
	}
	path, err := common.ResolveConfig(false)
	if err != nil {
		return nil, "", err
	}
	cfg, err := debug.LoadConfig(path)
	if err != nil {
		return nil, "", err
	}
	sub := cfg.CloneEnv(env)
	if sub == nil {
		return nil, "", fmt.Errorf("环境 %q 不存在（tg env list 看全部）", env)
	}
	sub.DataDir = filepath.Dir(path)
	return sub, path, nil
}

// connFor 取环境的库连接副本，账号槽位留空（= 取账号清单首项）。
//
// 返回副本：Accounts 只读共享，Account 是“这次用哪个账号”的一次性槽位，
// 改它不能影响配置本体。
func connFor(cfg *debug.Config) (*dbconfig.Connection, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("环境 %q 未挂数据库连接（设置 → 站点管理 → 数据库）", cfg.EnvName())
	}
	cc := *cfg.DB
	cc.Account = ""
	return &cc, nil
}

// primaryAccount 取账号清单首项的名字（GP 的“这个库用哪个账号登录”）。
func primaryAccount(cfg *debug.Config) (string, error) {
	cc, err := connFor(cfg)
	if err != nil {
		return "", err
	}
	user, _, ok := cc.DialCred()
	if !ok {
		return "", fmt.Errorf("环境 %q 的数据库连接里没有账号（设置 → 站点管理 → 数据库 → 数据库账号）", cfg.EnvName())
	}
	return user, nil
}

// entAccount 解析本次要用哪个账号：>0 按 ENT 清单（T100 那条线在用），
// <=0 取账号清单首项（TG/GP 的正常路径）。
//
// 实现只有一处（debug.Config.EntAccount）：Web 的 /api/dbsql 与 CLI 必须按同一份
// 清单落到同一个 schema —— 两处各写一遍，“同名不同号”迟早会漂移。
func entAccount(cfg *debug.Config, ent int) (string, error) {
	if ent > 0 {
		return cfg.EntAccount(ent)
	}
	return primaryAccount(cfg)
}

// dbConnFor 把该环境的库连接取出来，并在给了企业编号时换成那个企业的账号。
func dbConnFor(cfg *debug.Config, ent int) (*dbconfig.Connection, error) {
	cc, err := connFor(cfg)
	if err != nil {
		return nil, err
	}
	if ent > 0 {
		acct, err := cfg.EntAccount(ent)
		if err != nil {
			return nil, err
		}
		cc.Account = acct
	}
	return cc, nil
}
