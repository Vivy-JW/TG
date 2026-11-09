package testkit

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ResetFlags 把整棵命令树上的 flag 恢复成默认值 —— 让下一次调用像**全新进程**一样。
//
// 为什么需要：cobra 的 flag 值挂在命令对象上，而命令树是**包级单例**
// （internal/cli/dict 的 `Group`、internal/cli 的 `rootCmd`）。不重置的话，上一条用例
// 里的 `--offset 2` 会留到下一条 —— internal/cli/dict 从前正因为这个报过
// "越界：这份结果共 1 条"。真机上一条命令一个进程，所以这不是产品缺陷，
// 是测试必须自己补上的那一步。
//
// 递归到子命令是必须的：`--limit` 挂在命令组上、各命令自己还有一批 flag，
// 只重置根那一层会漏掉后者。
func ResetFlags(c *cobra.Command) {
	c.Flags().VisitAll(func(f *pflag.Flag) { _ = f.Value.Set(f.DefValue) })
	c.PersistentFlags().VisitAll(func(f *pflag.Flag) { _ = f.Value.Set(f.DefValue) })
	for _, sub := range c.Commands() {
		ResetFlags(sub)
	}
}
