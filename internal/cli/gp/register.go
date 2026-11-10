// Package gp 是 TOP GP 的查询命令组。
//
// 它刻意不并进 tt dict：T100 的 r.t / desc / r.v / scc / r.q / prog 查的是
// T100 字典表，GP 的 p_zta / p_link / p_ze 查的是另一套 Oracle 表，字段和语义
// 都不一样。并进去只会让 tt dict 到处长 GP 特判。
//
// 三条命令的分工：
//
//	tt gp table   p_zta 的“表结构”：本机 GP 字典 SQLite（t_table/t_column/...）
//	tt gp link    p_link 的“程序链接”：远程 gal_file，保留全部行并标入口
//	tt gp msg     p_ze 的“错误消息”：远程 ze_file
//
// 环境一律用 --env 点名。GP 没有 T100 的 ENT，所以这里不碰 --at / 企业账号；
// 直接连该环境配置里的 db（dbconfig.Connection）。
package gp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/dict/live"
	"tt/internal/gp"
	"tt/internal/output"
)

// Register 把 gp 组挂到根命令。
func Register(root *cobra.Command) {
	g := &cobra.Command{
		Use:   "gp",
		Short: "TOP GP：p_zta 表结构 / p_link 程序链接 / p_ze 错误消息 / 本地字典同步",
		Long: `TOP GP 查询（不是 T100 字典）。

  tt gp table <表> --env <环境>      查 p_zta 的表结构（本地 GP 字典）
  tt gp link <作业> --env <环境>     查 p_link 的完整程序链接（远程 Oracle）
  tt gp msg <编号> --env <环境>      查 p_ze 的错误消息（远程 Oracle）
  tt gp sync --env <环境>            把 GP 字典同步到本地 SQLite（Web 的「同步字典」同款）

别名：table=p_zta，link=p_link，msg=p_ze。

环境是 config.json hosts.sshs 里的 name，必须点名，没有默认环境；
表结构默认读本机 GP 字典，link/msg 默认远程查该环境的 Oracle。

为什么单独一组：T100 的 tt dict 查 dzea_t/dzeb_t 等，GP 查
zta_file/ztb_file/gal_file/ze_file，数据源、字段、语义都不同。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	g.AddCommand(newTableCmd(), newLinkCmd(), newMsgCmd(), newSyncCmd())
	root.AddCommand(g)
}

// setHumanDefault 让 tt gp 没显式指定输出格式时默认给人看的表格；
// 显式 --json/--csv/--human/--format 仍按用户的来。
//
// T100 的数据查询默认 JSON（方便 agent 和管道）；GP 这三条更像 p_zta/p_link/p_ze
// 的维护画面，第一次用的人往往直接看结果，所以单独把默认值改成 human。
func setHumanDefault(cmd *cobra.Command) {
	if common.Human || common.JSON || common.CSV {
		return
	}
	if cmd.Flags().Changed("format") || cmd.InheritedFlags().Changed("format") {
		return
	}
	common.Format = "table"
}

// target 一次查询解析出的环境与数据源。
type target struct {
	env   config.NamedSsh
	local bool
	meta  output.Meta
}

// resolveTarget 解析 --env。local 前缀只允许 table：
//
//	table:   --env 22主机        → 本地字典（p_zta 的数据就在本地）
//	         --env local:22主机  → 同上，兼容 T100 的 local: 写法
//	link/msg --env 22主机        → 远程 Oracle
//
// 不允许“本地没有就走远程”的静默降级：查的是哪一份数据必须写在输出头里。
func resolveTarget(allowLocal bool) (*target, error) {
	raw := strings.TrimSpace(common.Env)
	if raw == "" {
		return nil, output.Errorf("USAGE", 2, "需要 --env <环境名>（GP 没有默认环境）").
			WithHint("用 tt env list 看可用环境；例如 --env 22主机")
	}

	local := false
	if strings.HasPrefix(strings.ToLower(raw), "local:") {
		if !allowLocal {
			return nil, output.Errorf("USAGE", 2, "该命令只支持远程环境，不能用 local: 前缀")
		}
		local = true
		raw = strings.TrimSpace(raw[len("local:"):])
	}
	if raw == "" {
		return nil, output.Errorf("USAGE", 2, "local: 后面必须跟环境名").
			WithHint("例如 --env local:22主机")
	}

	path, err := common.ResolveConfig(false)
	if err != nil {
		return nil, err
	}
	root, err := config.Load(path)
	if err != nil {
		return nil, output.Errorf("CONFIG", 2, "读取配置失败: %v", err)
	}
	ref, err := root.Hosts.Resolve(raw)
	if err != nil {
		return nil, output.Errorf("ENV", 2, "%v", err)
	}
	env := *ref.Env
	if !strings.EqualFold(env.Product, "gp") {
		return nil, output.Errorf("GP_ENV", 2, "环境 %q 不是 TOP GP 环境（product=%q）", env.Name, env.Product).
			WithHint("GP 命令只用于 product=gp 的环境；用 tt env list 查看").
			WithMeta(output.Meta{Env: env.Name})
	}

	t := &target{env: env, local: local}
	t.meta.Env = env.Name
	t.meta.Source = "live"
	t.meta.Readonly = true // 三条命令都只读：本地 SQLite 与远程查询都不写库
	if local {
		t.meta.Source = "local"
		t.meta.Dialect = "sqlite"
		t.meta.LocalDBPath = gp.DictPath(env)
	}
	if !local && env.DB != nil {
		t.meta.Dialect = env.DB.Type
		t.meta.Target = dbTarget(env.DB)
		t.meta.ViaTunnel = env.DB.ViaSSH != nil
	}
	return t, nil
}

func dbTarget(c *dbconfig.Connection) string {
	if c == nil {
		return ""
	}
	port := c.Port
	if port == 0 {
		if c.Type == "kingbase" {
			port = 54321
		} else {
			port = 1521
		}
	}
	name := c.Service
	if name == "" {
		name = c.Database
	}
	return fmt.Sprintf("%s:%d/%s", c.Host, port, name)
}

// openLive 建立远程 GP 库连接。调用方负责 Close。
func openLive(ctx context.Context, t *target) (*live.Live, error) {
	if t == nil || t.local {
		return nil, output.Errorf("USAGE", 2, "该命令需要远程 GP 环境，不能用 local: 前缀")
	}
	if t.env.DB == nil {
		return nil, output.Errorf("DB", 2, "环境 %q 没有配置数据库连接", t.env.Name).
			WithHint("在设置 → 站点管理 → 数据库里配置该环境的 Oracle 连接").
			WithMeta(t.meta)
	}
	l, err := live.Open(ctx, *t.env.DB)
	if err != nil {
		return nil, output.Errorf("CONNECT", 5, "连接 GP 数据库失败: %v", err).
			WithHint("先 tt db ping --env " + t.env.Name + " 验证连通性").
			WithMeta(t.meta)
	}
	return l, nil
}

// queryLive 执行一条只读 SQL，返回列名/行。错误统一带上环境信息。
func queryLive(ctx context.Context, l *live.Live, t *target, sql string) ([]string, [][]string, error) {
	cols, rows, err := l.Connector().Query(ctx, sql)
	if err != nil {
		return nil, nil, output.Errorf("QUERY", 5, "查询 GP 数据库失败: %v", err).
			WithMeta(t.meta)
	}
	return cols, rows, nil
}

// withTimeout 统一限制一次远程查询的时间。GP 库偶发慢查询时，
// 不能让 CLI 一直挂着；45 秒足够 gal_file/ze_file 的点查。
func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 45*time.Second)
}

// emitError 的辅助：把已解析出的环境信息补进出错信封。
func envErr(t *target, code string, exit int, format string, args ...any) *output.Error {
	e := output.Errorf(code, exit, format, args...)
	if t != nil {
		e = e.WithMeta(t.meta)
	}
	return e
}

// stdout 便于测试时替换；命令默认写 os.Stdout。
var stdout = os.Stdout

// metaNotes 复制一份 Meta，避免调用方修改共享值；Notes 为空时原样返回。
func metaNotes(m output.Meta, notes ...string) output.Meta {
	for _, n := range notes {
		if strings.TrimSpace(n) != "" {
			m.Notes = append(m.Notes, n)
		}
	}
	return m
}
