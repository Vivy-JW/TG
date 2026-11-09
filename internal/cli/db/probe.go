package db

// tt db probe —— 从**服务器**连一次库（SSH + sqlplus/ksql），验证该企业的账号。
// 原 tt debug db：它读 gzou_t 映射并真连一次；现在映射来自配置的 ENT 清单（不联网解析）。

import (
	"fmt"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/debug"
	"tt/internal/output"
)

func newProbeCmd() *cobra.Command {
	var entFlag int
	cmd := &cobra.Command{
		Use:   "probe",
		Short: "从服务器连一次库（SSH + sqlplus/ksql；验证该企业的账号）",
		Long: `登录服务器，用该企业账号真连一次库（Oracle 走 sqlplus，金仓走 ksql），
回答"服务器这一侧现在连不连得上、这个账号能不能用"。

  tt db probe --at 主机109@99
  tt db probe --at 主机109@99 --json

与本机视角的 tt db ping 是两条不同的路：
  ping  你 → 库（客户端直连，可经隧道）
  probe 服务器 → 库（这条正是 tt db sql / 调试时实际走的链路）`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ent := entOf(entFlag)
			if ent <= 0 {
				return output.Errorf("USAGE", 2, "需要 --at <环境>@<企业>：验的是这个企业的账号（tt db probe --at-list 可看取值）")
			}
			return ProbeOnce(ent)
		},
	}
	cmd.Flags().IntVar(&entFlag, "ent", 0, "企业编号（也可用 --at <环境>@<企业>）")
	return cmd
}

// ProbeOnce 服务器侧连接体检（tt db probe 与旧的 tt debug db 转发共用）。
func ProbeOnce(ent int) error {
	cfg, _, err := loadCfg()
	if err != nil {
		return output.Errorf("TARGET", 2, "%v", err)
	}
	if _, err := entAccount(cfg, ent); err != nil {
		return output.Errorf("TARGET", 2, "%v", err)
	}
	rep, err := debug.ProbeDB(cfg, ent, debug.EntListOpt{Ent: ent})
	if err != nil {
		return output.Errorf("DB_FAILED", 4, "%v", err)
	}
	if common.IsJSON() {
		return common.PrintJSON(rep)
	}

	if kdb := rep.Env["database"]; kdb != "" {
		fmt.Printf("区域: %s   数据库: 人大金仓 %s@%s:%s\n", rep.Zone, kdb, rep.Env["host"], rep.Env["port"])
		if ks := rep.Env["ksql"]; ks != "" {
			fmt.Printf("ksql: %s\n", ks)
		}
	} else {
		fmt.Printf("区域: %s   Oracle: %s:%s (service %s)\n",
			rep.Zone, rep.Env["host"], rep.Env["port"], rep.TNS)
		if sp := rep.Env["sqlplus"]; sp != "" {
			fmt.Printf("sqlplus: %s\n", sp)
		}
		if oh := rep.Env["oracleHome"]; oh != "" {
			fmt.Printf("ORACLE_HOME: %s\n", oh)
		}
	}
	fmt.Printf("\n企业(ENT) → 账号(schema)，共 %d 个（来自配置的 ENT 清单）：\n", len(rep.Mappings))
	for _, m := range rep.Mappings {
		mark := " "
		if rep.Probe != nil && m.Ent == rep.Probe.Ent {
			mark = "*"
		}
		fmt.Printf("  %s %-6d → %s\n", mark, m.Ent, m.Account)
	}
	if rep.Probe != nil {
		p := rep.Probe
		fmt.Printf("\n企业 %d 连接验证: %s@%s\n", p.Ent, p.Account, rep.TNS)
		if p.Host != "" {
			fmt.Printf("  主机: %s:%s  服务名: %s\n", p.Host, p.Port, p.Service)
		}
		if p.Connect {
			fmt.Println("  结果: 连接成功 ✓")
		} else {
			fmt.Printf("  结果: 连接失败 ✗  %s\n", p.Error)
			fmt.Println("  (密码优先取账号清单；未收录按 账号=密码 惯例。若失败请在 设置 → 站点管理 → 数据库 的账号清单里维护)")
		}
	}
	return nil
}

// reportedErr 只带退出码：结果已经打印过了，根命令不要再补一条错误。
type reportedErr struct{ code int }

func (e reportedErr) Error() string         { return "连接失败" }
func (e reportedErr) ExitCode() int         { return e.code }
func (e reportedErr) AlreadyReported() bool { return true }
