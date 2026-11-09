package db

// tt db ping —— 从**本机**连一次库（客户端直连，可经 viaSsh 隧道）。
// 原 tt dict db ping；区别是身份用 --at，且给 @企业 时用配置清单里的那个账号
// （不给企业就按客户端惯例用账号清单首项 —— 这一档只回答“库通不通”）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/dict/live"
	"tt/internal/output"
)

func newPingCmd() *cobra.Command {
	var entFlag int
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "从本机连一次库（客户端直连；支持 viaSsh 隧道）",
		Long: `从**本机**连一次库，验证网络/VPN/隧道与账号口令是否可用（客户端直连视角）。

  tt db ping --at 主机109          # 用环境配置里的账号清单（首项）连一次
  tt db ping --at 主机109@99       # 用配置 ENT 清单里企业 99 的账号连一次
  tt db ping --at 主机109@99 --json

对端能不能连用 tt db probe（服务器侧视角）；查数据用 tt db sql。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, _, err := loadCfg()
			if err != nil {
				return output.Errorf("TARGET", 2, "%v", err)
			}
			ent := entOf(entFlag)
			cc, err := dbConnFor(cfg, ent)
			if err != nil {
				return output.Errorf("TARGET", 2, "%v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			l, err := live.Open(ctx, *cc)
			if err != nil {
				return output.Errorf("DB_UNREACHABLE", 4, "%v", err)
			}
			defer l.Close()
			ver, err := l.Connector().ServerVersion(ctx)
			if err != nil {
				return output.Errorf("DB_UNREACHABLE", 4, "%v", err)
			}
			oneLine := strings.TrimSpace(strings.SplitN(ver, "\n", 2)[0])

			if common.IsJSON() {
				return common.PrintJSON(map[string]any{
					"ok": true, "env": cfg.EnvName(), "ent": ent,
					"account": cc.Account, "dialect": cc.Type, "target": cc.Address(),
					"version": oneLine,
				})
			}
			who := cfg.EnvName()
			if ent > 0 {
				who = fmt.Sprintf("%s@%d", cfg.EnvName(), ent)
			}
			acct := cc.Account
			if acct == "" {
				acct = "账号清单首项"
			}
			fmt.Printf("连接正常: %s (%s, %s) %s\n", who, acct, cc.Type, oneLine)
			return nil
		},
	}
	cmd.Flags().IntVar(&entFlag, "ent", 0, "企业编号（也可用 --at <环境>@<企业>；不给=账号清单首项）")
	return cmd
}
