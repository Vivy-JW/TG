// Package debug 承载 tt debug 命令组（原 TDebug 的 CLI）。
//
// 合并前的 TDebug 是独立二进制、独立 cli 包；现在并入 tt 后以子命令组形式存在，
// 源码由 internal/cli/debug 下的文件实现，调试内核在 internal/debug。
package debug

import (
	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

// Group 是 tt debug 命令组。各命令文件在自己的 init() 里往它挂子命令。
var Group = &cobra.Command{
	Use:   "debug",
	Short: "TDebug - T100 作业调试器",
	Long: `通过 SSH 在 T100 服务器上驱动 fglrun -d 的 (fgldb) 文本调试协议，
提供本地 Web 调试界面（源码/断点/调用栈/变量/接口日志）与命令行控制端
（tt debug start/exec/…），实现"人操作 GDC 界面 + AI 借助命令行检查分析"的
人机协同调试。

需要 config.json 中至少一个 hosts.sshs 环境；首次使用先执行 tt serve 启动本地服务。`,

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
