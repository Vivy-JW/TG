// Package dev 承载 tt dev 命令组（原 tdev 的设计器包管线）。
//
// 目录即分层：
//
//	dev/          本包：cobra 树（路由、组级帮助、未知命令退 2、install 墓碑）
//	              + 根开关桥 takeRootFlags + 退出码透传 exitCode
//	dev/tzc/      .tzc 线：八个动词核心 + 两条线共用的输出/解析脚手架（导出面见其 scaffold.go）
//	dev/tzs/      .tzs 线：整线 DisableFlagParsing 透传，动词面由引擎 manifest 定义
//
// 为什么要留一层"不归 cobra 解析"：参数写法契约冻结在 tdev 的解析器上（common.Usage：
// "参数写法完全不变"，位置无关、接受单杠长形式），pflag 接手会把 `-only` 静默错解析成
// `-o nly`。退出码契约（0/2/3/4/5；tzs 线没有 3）经 exitCode 透传给根命令。
package dev

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tt/internal/cli/dev/common"
	"tt/internal/cli/dev/tzc"
	"tt/internal/cli/dev/tzs"
)

// Group 是 tt dev 命令组。
var Group = newDevCmd()

// envNameFn 由 internal/cli 注入根命令的 --env 取值。
//
// 为什么要这一层:本包**不能**直接 import internal/cli/common —— 它已经 import 了
// internal/cli/dev/common(包名同为 common),两个 common 撞名;而改名会牵动本包所有文件。
// 所以沿用与 WebFS 同样的做法:上游给一对存取函数,这里只转手。
var envNameFn = func() string { return "" }

// envSetFn 由 internal/cli 注入"把 --env 写进全局开关"的方式。
// takeRootFlags 摘到 --env 时调它 —— 与 --config 走 TT_CONFIG 同一个思路:
// 摘出来的根开关不重发给动词解析器,而是落到这条线真正会读的地方。
var envSetFn = func(string) {}

// SetEnvNameFunc 由 internal/cli 在装配命令树时调用,注入 --env 的读写方式。
func SetEnvNameFunc(get func() string, set func(string)) {
	if get != nil {
		envNameFn = get
	}
	if set != nil {
		envSetFn = set
	}
}

// Register 把本命令组挂到根命令上。
func Register(parent *cobra.Command) { parent.AddCommand(Group) }

// devLong 是命令组的 --help 文本。参数与退出码契约沿用 tdev 自己的说明（common.Usage），
// 这里只补一句在 tt 里的调用方式。
const devLong = `T100 设计器包工具：.tzc 代码包 + .tzs 表单包

在 tt 里，原来 tdev 的命令整体后移一级：
  tdev tzc export …   →  tt dev tzc export …
  tdev tzs export …   →  tt dev tzs export …
  tdev install …      →  tt install …（安装不再住在这条线上）

.tzc 的退出码与参数写法完全不变（0 成功 / 2 包格式或用法错 / 3 校验失败 /
4 拒绝写入 / 5 IO 与环境失败）。.tzs 那条线**没有 3**，另有一个 1（引擎内部错）。

` + common.Usage

func newDevCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dev",
		Short: "T100 设计器包工具（.tzc 代码包 + .tzs 表单包）",
		Long:  devLong,
		// 错误已由动词/本组自己打好（exitCode.AlreadyReported），cobra 不再重复输出；
		// 根命令的 Silence* 也兜着这一层。
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 裸 tt dev 与未知子命令保持 tdev 契约：Usage + 退 2。
			// 不能交给 cobra 缺省行为 —— 组命令没有 RunE 时未知子命令退 0，
			// 交给根命令 reportError 则退 1，都不符合这条线的契约。
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "未知子命令 %q\n\n%s", args[0], common.Usage)
			} else {
				fmt.Fprint(os.Stderr, common.Usage)
			}
			return exitCode(2)
		},
	}
	cmd.AddCommand(newTzcCmd(), newTzsCmd(), newInstallTombstone())
	return cmd
}

// newTzcCmd 是 tt dev tzc 命令组：动词清单来自 tzc.Verbs()（展示面与核心都住在 tzc），
// 未知动词落进本组 RunE 保持"未知动词退 2"的契约。
func newTzcCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tzc",
		Short: ".tzc 代码包：export 渲染围栏工作区，apply 三道闸门写回",
		Long:  tzc.TzcLong,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				fmt.Fprintf(os.Stderr, "未知动词 %q（export/status/verify/apply/unlock/rename/newfn/selftest）\n", args[0])
				return exitCode(2)
			}
			fmt.Fprint(os.Stderr, common.Usage)
			return exitCode(2)
		},
	}
	for _, v := range tzc.Verbs() {
		v := v
		cmd.AddCommand(newTzcVerbCmd(v.Use, v.Short, v.Long, v.Run))
	}
	return cmd
}

// newTzcVerbCmd 把一个 tzc 动词接进 cobra 树。
//
// DisableFlagParsing 是刻意的：参数解析权在 tdev 自己的解析器（见包注释），
// 根开关翻译见 takeRootFlags。cobra 在动词这层只提供 Use/Short/Long（进组级帮助）。
func newTzcVerbCmd(use, short, long string, run func(args []string) int) *cobra.Command {
	return &cobra.Command{
		Use:                use,
		Short:              short,
		Long:               long,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 说明书本身不该失败：原 stdlib flag 把 -h/--help 当未知开关退 2，
			// 与 tzs 线（`<动词> --help` 退 0，它就是说明书本身）和 cobra 惯例都不一致。
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				fmt.Fprint(cmd.OutOrStdout(), long)
				return nil
			}
			bridged, err := takeRootFlags(args)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return exitCode(2)
			}
			if code := run(bridged); code != 0 {
				return exitCode(code)
			}
			return nil
		},
	}
}

// newTzsCmd 是 tt dev tzs：整条线 DisableFlagParsing 原样透传给 tzs.Run。
// 动词面由引擎 manifest 定义、`<动词> --help` 是**引擎**的说明书（附动词索引），
// 所以连动词都不挂成 cobra 子命令 —— 见 tzs 包顶部的说明。
func newTzsCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "tzs",
		Short:              ".tzs 表单包：export 纯解压只读；读写表单走引擎动词",
		Long:               tzs.Usage,
		DisableFlagParsing: true,
		Args:               cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			bridged, err := takeRootFlags(args)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return exitCode(2)
			}
			// 环境名来自根命令的 --env(--conn 是别名),经 SetEnvNameFunc 注入。
			// 工作区按环境配,所以 tzs 这条线必须拿到它 —— 没有"默认环境"可回落。
			if code := tzs.Run(bridged, envNameFn()); code != 0 {
				return exitCode(code)
			}
			return nil
		},
	}
}

// newInstallTombstone 是 tt dev install 的墓碑：安装已并入 tt install。
// 保留子命令是为给出指引而不是"未知子命令"——退出码仍是 2，脚本行为不变。
func newInstallTombstone() *cobra.Command {
	return &cobra.Command{
		Use:    "install",
		Short:  "（已并入 tt install，这里是墓碑）",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(os.Stderr, "tt dev install 已并入 tt install：\n"+
				"  tt install skills [--to <dir>|auto] [--agent <名字>] [--force]\n"+
				"  tt install path   [--dry-run]\n")
			return exitCode(2)
		},
	}
}
