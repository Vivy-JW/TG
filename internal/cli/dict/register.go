// Package dict 承载 tt dict 命令组（原 tdict 的 CLI）。
//
// 合并前的 tdict 是独立二进制、独立 cli 包；现在并入 tt 后以子命令组形式存在，
// 数据源实现在 internal/dict（db 本地镜像 / live 远程直查 / dbsync 同步）。
package dict

import (
	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

// Group 是 tt dict 命令组。各命令文件在自己的 init() 里往它挂子命令。
var Group = &cobra.Command{
	Use:   "dict",
	Short: "TDict - ERP 数据字典查询工具",
	Long: `查询 ERP 数据字典。

查询数据源（r.t/r.v/desc/scc/r.q）：环境一律点名。
  --env <环境>              直查该环境的远程库（金仓/Oracle）
  --env local:<环境>        查该环境自己的本地 SQLite 镜像
                            （<数据目录>/dict/<环境>.db，由 tt dict db sync --env <环境> 同步）
  本地库按环境分文件，各环境互不覆盖。
ACC（应用分类码）：tt dict acc show / query --template / query --spec
也可用 config.json 顶层 query.source 固定数据源（local:<环境> 或环境名）。`,

	// RunE：**收到不认识的子命令必须非 0**（见 common.UnknownSubcommand）。
	// 没有这个 RunE，cobra 会打一份帮助就退 0 —— 而"我打错了命令"读成成功，
	// 是调用方最难自己发现的失败形态（实测 2026-09-25：`tt dict nope` 退 0，
	// 与它自己文档里写的「1 = 用法错」相矛盾）。
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return common.UnknownSubcommand(cmd, args)
	},
}

// Register 把本命令组挂到根命令上。
func Register(root *cobra.Command) { root.AddCommand(Group) }
