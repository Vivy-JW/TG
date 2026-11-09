package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/envmove"
)

// newEnvCmd 是统一的环境管理命令组。
//
// 合并前 TDictCli 有 `tdict env [<环境名>]`、TDebug 有 `tdebug env` 与 `tdebug topent`，
// 三者操作的是同一份数据（一个环境清单），只是各自实现了一遍。合并后只有这一份。
func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "管理 SSH 环境（三个工具共用同一份环境清单）",
		Long: `列出/查看/切换 config.json 里 hosts.sshs 定义的 SSH 环境。

环境清单是三个工具共用的：调试、字典查询、源码镜像都从同一份 hosts 里取连接，
所以在这里加一台机器，tt debug 和 tt dict 都能立刻用。`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runEnvList(cmd)
			}
			return runEnvShow(cmd, args[0])
		},
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "列出全部环境（同不带参数）",
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, args []string) error { return runEnvList(cmd) },
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show <环境名>",
		Short: "显示某个环境的连接详情（口令打码）",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runEnvShow(cmd, args[0]) },
	})

	// 企业清单：--at 的取值来源（环境 × 企业）。用它替代“记住 ENT 号”。
	cmd.AddCommand(newEnvEntsCmd())

	// env rename：环境名是跨文件的持久引用，改名必须是一次迁移，不是一个字段赋值
	// （dict/<环境>.db、ents/<环境>.json、srccache/<环境>/ 都拿它当键）。
	cmd.AddCommand(&cobra.Command{
		Use:   "rename <旧环境名> <新环境名>",
		Short: "改环境名，并把按名字落盘的字典库/企业快照/源码镜像一起迁移",
		Long: `把环境改成一个新名字，**同时**迁移所有按旧名落盘的东西：

  dict/<旧名>.db       本地字典库（改完不用重新同步；尊重 dict.dir / 环境 dictDir）
  dict/<旧名>.db.bak   同步留下的上一版备份
  ents/<旧名>.json     企业目录快照
  srccache/<旧名>/     调试源码镜像
  execlog/<旧名>/      调试跑过命令的大输出副本
  mirror.dir/<旧名>/   tt file mirror 的源码镜像（环境自配 mirrorDir 时不动）
  query.source         配置里指向旧名的数据源（local:<旧名> 或裸名字）

不做迁移的改名会留下悬空引用：查询报「本地库不存在」或「查不到数据」，
与「数据真的不存在」在输出上分不出来 —— 这类问题只能靠事后排查。

目标名已存在、或目标落点已有文件时整体拒绝，一个都不搬。`,
		Example: `  tt env rename 正式区 正式
  tt env rename 测试区 测试`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error { return runEnvRename(cmd, args[0], args[1]) },
	})

	// env use 是**墓碑**,不是被删掉的命令:它背后的 hosts.activeEnv 已经整个移除
	// (环境一律点名 —— 见 config.Hosts 的注释)。留一个会解释原因、并给出替代用法的
	// 命令,好过让用户撞上"unknown command env use"而以为环境管理坏了。
	cmd.AddCommand(&cobra.Command{
		Use:   "use <环境名>",
		Short: "(已移除) 默认环境这个概念已取消 —— 请每次用 --env 点名",
		Long: `"默认环境"已移除(hosts.activeEnv 不再存在)。

原因:同机多环境时,"回落到默认环境"与"上错环境"在输出上完全无法区分 ——
而用错环境的典型症状是"查不到数据",与"数据不存在"一模一样。所以环境
改为**一律点名**:

  tt debug start <作业> --env <环境名>      # 调试:点名环境
  tt dict <命令> --env <环境名>             # 字典:点名环境(或 --env local:<环境> 查该环境本地副本)
  tt dev tzs <动词> --env <环境名>          # 表单:环境决定用哪个工作区

Web 端在会话面板里选环境(那个选择是运行时的,同样不写配置文件)。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("`tt env use` 已移除:默认环境不再存在,请用 --env %s 点名(tt env list 看全部)", args[0])
		},
	})

	// env topent 同样是墓碑:环境级企业编号(topent)已从配置移除,ENT 只能每次显式给。
	cmd.AddCommand(&cobra.Command{
		Use:   "topent <环境名> <企业编号>",
		Short: "(已移除) 企业编号不再存在环境配置里 —— 请每次用 --ent 给",
		Long: `环境级的默认企业编号(hosts.sshs[].topent)已移除。

原因:企业编号决定**用哪个账号(schema)**连库;把它登记在环境上等于给它一个
会被静默沿用的默认值,而用错账号的症状是"查不到数据",与"数据不存在"一样。
所以 ENT 改为**每次显式给**:

  tt debug start <作业> --env <环境名> --ent <企业编号>
  tt db sql --at <环境>@<企业> "<SQL>"
  tt dict <命令> --env <环境名> --ent <企业编号>

不给 ENT 时:调试沿用登录 shell 里已有的 TOPENT(不覆盖),字典查询用账号列表首项
(并在输出信封的 notes 里说明)。`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("`tt env topent` 已移除:企业编号不再存配置,请用 --ent %s 在命令上给", args[1])
		},
	})
	return cmd
}

// envView 是环境在 --json 输出里的形状。口令一律不出现在输出里。
type envView struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Zone      string `json:"zone,omitempty"`
	DBType    string `json:"dbType,omitempty"`
	DBAddress string `json:"dbAddress,omitempty"`
}

func envViews(h *config.Hosts) []envView {
	out := make([]envView, 0, len(h.SSHs))
	for i := range h.SSHs {
		e := &h.SSHs[i]
		// 原先这里带 Topent 与 Active("是不是默认环境")—— 两者都已移除:
		// 企业编号不再存配置,默认环境这个概念也不存在了(环境一律点名)。
		v := envView{
			Name: e.Name,
			Host: e.Host,
			Port: e.Port,
			User: e.User,
			Zone: e.Zone,
		}
		if e.DB != nil {
			v.DBType = e.DB.Type
			v.DBAddress = e.DB.Address()
		}
		out = append(out, v)
	}
	return out
}

func runEnvList(cmd *cobra.Command) error {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return err
	}
	root, err := config.Load(path)
	if err != nil {
		return err
	}
	h := root.Hosts

	if common.IsJSON() {
		return common.PrintJSON(map[string]any{
			"config": path,
			"sshs":   envViews(&h),
		})
	}

	if len(h.SSHs) == 0 {
		cmd.Printf("尚未配置任何环境。\n配置文件：%s\n运行 tt serve 打开配置页添加，或编辑其中的 hosts.sshs。\n", path)
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	// 「企业」这一列已去掉:企业编号不再存配置(每次 --ent 给),列在这里只会误导。
	fmt.Fprintln(w, "名称\tSSH\t区域\t数据库")
	for _, v := range envViews(&h) {
		ssh := fmt.Sprintf("%s@%s:%d", v.User, v.Host, v.Port)
		db := v.DBAddress
		if db == "" {
			db = "-"
		} else if v.DBType != "" {
			db = v.DBType + " " + db
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", v.Name, ssh, orDash(v.Zone), db)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	// 没有"默认环境"可标注了 —— 环境一律由 --env 点名。
	cmd.Printf("\n配置文件：%s\n", path)
	return nil
}

func runEnvShow(cmd *cobra.Command, name string) error {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return err
	}
	root, err := config.Load(path)
	if err != nil {
		return err
	}
	h := root.Hosts
	e := h.ByName(name)
	if e == nil {
		return fmt.Errorf("没有名为 %q 的环境（用 tt env list 看全部）", name)
	}

	if common.IsJSON() {
		v := envViews(&h)
		for i := range v {
			if v[i].Name == e.Name {
				return common.PrintJSON(v[i])
			}
		}
	}

	cmd.Printf("名称      %s\n", e.Name)
	cmd.Printf("SSH       %s@%s:%d\n", e.User, e.Host, e.Port)
	cmd.Printf("口令      %s\n", mask(e.Password))
	cmd.Printf("区域      %s\n", orDash(e.Zone))
	// 「企业」一行已去掉:企业编号不再存在环境配置里(每次 --ent 给)。
	if e.LaunchArgs != "" {
		cmd.Printf("启动参数  %s\n", e.LaunchArgs)
	}
	// 看门狗：负数 = 关闭（`config.WatchdogProblems` 会把它当问题报出来）。
	// 这里**无条件**打印：从前只在 >0 时输出，于是“被关掉”和“没配”在 env show
	// 里长得一模一样 —— 而这两件事的后果差得很远（一个不自动放行，一个 30 分钟放行）。
	switch {
	case config.WatchdogDisabled(e.WatchdogSeconds):
		cmd.Printf("停站超时  已关闭（停站后不会自动放行，行锁只能靠人工处理）\n")
	case e.WatchdogSeconds > 0:
		cmd.Printf("停站超时  %d 秒\n", e.WatchdogSeconds)
	default:
		cmd.Printf("停站超时  %d 秒（默认）\n", config.DefaultWatchdogSeconds)
	}
	if e.DB == nil {
		cmd.Printf("数据库    未配置\n")
		return nil
	}
	cmd.Printf("数据库    %s %s\n", e.DB.Type, e.DB.Address())
	cmd.Printf("只读 SQL  %s\n", onOff(e.DB.ReadonlySQLEnabled()))
	if e.DB.ViaSSH != nil {
		cmd.Printf("隧道      %s@%s:%d\n", e.DB.ViaSSH.User, e.DB.ViaSSH.Host, e.DB.ViaSSH.Port)
	}
	if len(e.DB.Accounts) > 0 {
		names := make([]string, 0, len(e.DB.Accounts))
		for _, a := range e.DB.Accounts {
			names = append(names, a.Account)
		}
		sort.Strings(names)
		cmd.Printf("账号      %s\n", strings.Join(names, ", "))
	}
	return nil
}

// mask 把口令打码，只保留长度信息 —— 终端输出常被贴进 issue 或聊天记录。
func mask(s string) string {
	if s == "" {
		return "（未设置）"
	}
	return strings.Repeat("*", len([]rune(s)))
}

func onOff(b bool) string {
	if b {
		return "开启"
	}
	return "关闭"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// runEnvRename 执行一次带迁移的改名：先搬落点，再改配置；配置写失败就把落点搬回去。
func runEnvRename(cmd *cobra.Command, from, to string) error {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return err
	}
	root, err := config.Load(path)
	if err != nil {
		return err
	}
	e := root.Hosts.ByName(from)
	if e == nil {
		return fmt.Errorf("没有名为 %q 的环境（用 tt env list 看全部）", from)
	}
	if root.Hosts.ByName(to) != nil {
		return fmt.Errorf("环境 %q 已存在，不能改名成它", to)
	}

	moves, err := envmove.Plan(path, from, to, e.Host, e.Zone)
	if err != nil {
		return err
	}
	if err := envmove.Execute(moves, to, from); err != nil {
		return err
	}
	rewritten, err := config.RenameHost(path, from, to)
	if err != nil {
		// 配置没改成 → 把落点搬回旧名，别留下"文件在新名、配置在旧名"的半套。
		// 回滚失败必须报出来：只报配置错误会让用户以为文件还是旧的。
		if uerr := envmove.Undo(moves, from); uerr != nil {
			return fmt.Errorf("%w；同时回滚落点失败：%v", err, uerr)
		}
		return err
	}

	if common.IsJSON() {
		return common.PrintJSON(map[string]any{
			"ok":                   true,
			"config":               path,
			"from":                 from,
			"to":                   to,
			"querySourceRewritten": rewritten,
			"moved":                moves,
		})
	}
	cmd.Printf("已把环境 %q 改名为 %q。\n", from, to)
	if len(moves) == 0 {
		cmd.Printf("没有需要迁移的落点（本地字典库/企业快照/源码镜像都不存在）。\n")
	}
	for _, m := range moves {
		cmd.Printf("  迁移 %-8s %s → %s\n", m.Kind, m.From, m.To)
	}
	if rewritten {
		cmd.Printf("query.source 已同步指向新名字。\n")
	}
	// 运行中的 tt serve 把配置读进了内存（web 保存路径有 Reloaders，CLI 改名没有）。
	// 不替用户重启服务，但必须告诉他：否则下一轮查询可能还按旧名寻址。
	if _, err := os.Stat(config.LocationsAt(path).StateFile()); err == nil {
		cmd.Printf("\n提示：检测到 tt serve 的状态文件。若服务正在运行，请重启它（tt serve --stop && tt serve）以加载新环境名。\n")
	}
	cmd.Printf("\n配置文件：%s\n", path)
	return nil
}
