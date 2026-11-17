package db

// tg db probe —— 从**服务器**连一次库（SSH + sqlplus/ksql），验证该企业的账号。
// 原 tg debug db：它读 gzou_t 映射并真连一次；现在映射来自配置的 ENT 清单（不联网解析）。

import (
	"fmt"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/debug"
	"tg/internal/output"
)

func newProbeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "probe",
		Short: "从服务器连一次库（SSH + sqlplus/ksql）",
		Long: `登录服务器，用环境配置里的数据库账号真连一次库（Oracle 走 sqlplus，金仓走 ksql），
回答"服务器这一侧现在连不连得上、这个账号能不能用"。

  tg db probe --at 22主机
  tg db probe --at 22主机 --json

与本机视角的 tg db ping 是两条不同的路：
  ping  你 → 库（客户端直连，可经隧道）
  probe 服务器 → 库（这条正是 tg db sql / 调试时实际走的链路）`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return ProbeOnce(0)
		},
	}
	return cmd
}

// ProbeOnce 服务器侧连接体检（tg db probe 与旧的 tg debug db 转发共用）。
func ProbeOnce(ent int) error {
	cfg, _, err := loadCfg()
	if err != nil {
		return output.Errorf("TARGET", 2, "%v", err)
	}
	acct, aerr := primaryAccount(cfg)
	if aerr != nil {
		return output.Errorf("TARGET", 2, "%v", aerr)
	}
	rep, err := debug.ProbeDBAccount(cfg, acct)
	if err != nil {
		return output.Errorf("DB_FAILED", 4, "%v", err)
	}
	if common.IsJSON() {
		return common.PrintJSON(rep)
	}

	if kdb := rep.Env["database"]; kdb != "" {
		fmt.Printf("数据库: 人大金仓 %s@%s:%s\n", kdb, rep.Env["host"], rep.Env["port"])
		if ks := rep.Env["ksql"]; ks != "" {
			fmt.Printf("ksql: %s\n", ks)
		}
	} else {
		fmt.Printf("Oracle: %s:%s (service %s)\n", rep.Env["host"], rep.Env["port"], rep.TNS)
		if sp := rep.Env["sqlplus"]; sp != "" {
			fmt.Printf("sqlplus: %s\n", sp)
		}
		if oh := rep.Env["oracleHome"]; oh != "" {
			fmt.Printf("ORACLE_HOME: %s\n", oh)
		}
	}
	if rep.Probe != nil {
		p := rep.Probe
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
