// Package common 是三个工具命令组共享的 CLI 上下文：全局开关、内嵌前端、
// 配置路径解析、统一输出约定。
//
// 它是叶子包（不 import 任何 tt 内部命令包），所以 internal/cli 下的
// debug/dev/dict 三个命令组都能安全依赖它，不会与根命令形成循环。
package common

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/config"
	"tt/internal/output"
)

// 全局开关。由根命令的 persistent flags 绑定；各命令组只读。
var (
	// ConfigPath --config 显式指定的配置文件路径（空 = 按规则解析）
	ConfigPath string
	// JSON --json：机器可读输出
	JSON bool
	// CSV --csv：CSV 输出（--format csv 的语法糖）
	CSV bool
	// Format --format：输出形态 json(默认) | csv | table。--json/--csv 是它的语法糖。
	Format string
	// Human --human：人读表格/文本（数据类命令默认 JSON）。
	Human bool
	// Verbose -v：显示解析细节（如实际使用的数据源路径）
	Verbose bool
	// Env --env 指定的环境名；--conn 是其别名（合并前 tdict 的写法）
	Env string
	// WebFS 内嵌的前端构建产物（web/dist），由 main 注入
	WebFS fs.FS
	// Version 版本号，由 build_portable.bat 经 -ldflags 注入
	Version string
)

// ResolveConfig 按统一规则解析配置文件路径。
//
//	allowMissing=false：找不到任何配置时报错（读命令用）
//	allowMissing=true ：返回缺省落点，允许首次保存创建文件（写命令用）
func ResolveConfig(allowMissing bool) (string, error) {
	return config.ResolvePath(ConfigPath, allowMissing)
}

// ConfigHint 给 --help 用的一行配置位置提示。
func ConfigHint() string { return config.DefaultConfigPathHint() }

// WebFrontend 返回内嵌前端的 FS（dist 根，即 web/app 的构建产物）。
//
// 合并前这里是 WebSub(name) —— 当时有两套 SPA（调试工作台与字典页）各占 dist 下一个
// 子目录；字典页已并入统一设置页，只剩一套，所以不需要再按名字取子目录。
// main.go 已把嵌入 FS 的根收敛到 dist，这里直接用。
//
// 取不到时返回 nil —— 调用方应据此回落到引导页，而不是 panic：
// 前端没构建（只有 .gitkeep 占位）是完全正常的状态。
func WebFrontend() fs.FS {
	return WebFS
}

// OutputFormat 解析本次要用的输出形态。
//
// 默认 **json** —— 依据是表格类格式对 LLM 的理解力实测(JSON 52.3% /
// Markdown 表格 51.9% / CSV 44.3%),而 JSON 又省掉了解析歧义。
// --json / --csv 保留为语法糖：既有脚本与 skill 文档里的例子不改即可用。
func OutputFormat() output.Format {
	switch {
	case Human:
		return output.FormatTable
	case JSON:
		return output.FormatJSON
	case CSV:
		return output.FormatCSV
	}
	switch strings.ToLower(strings.TrimSpace(Format)) {
	case "csv":
		return output.FormatCSV
	case "table", "text":
		return output.FormatTable
	default:
		return output.FormatJSON
	}
}

// MetaProvider 由命令组注入：返回当前数据源的落脚点(环境/账号/库)，
// 供错误信封使用 —— 出错时"哪个环境连不上"比错误本身更难猜。
var MetaProvider func() output.Meta

// CurrentMeta 取当前数据源的环境信息(未注入返回零值)。
func CurrentMeta() output.Meta {
	if MetaProvider == nil {
		return output.Meta{}
	}
	return MetaProvider()
}

// IsJSON 本次是否输出 JSON：数据类命令的默认形态（--human 才切文本）。
func IsJSON() bool { return OutputFormat() == output.FormatJSON }

// PrintJSON 以紧凑 JSON 输出 v。查询命令请用 output.Emit —— 裸值没有地方放
// "这次查的是哪个环境、哪个账号"。
func PrintJSON(v any) error { return output.WriteJSONValue(os.Stdout, v) }

// Fatal 打印错误到 stderr 并以退出码 1 结束。
func Fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

// UnknownSubcommand 给「组命令收到一个不认识的子命令」造一个错误。
//
// 为什么需要它（实测 2026-09-25）：**不设 RunE 的组命令会让 cobra 打一份帮助就退 0** ——
// `tt debug nope` 与 `tt dict nope` 都是这样，而 `tt dict` 自己的契约写着「1 = 用法错」。
// 把「我打错了命令」读成成功，是调用方（尤其是 AI）最难自己发现的失败形态：它会以为
// 活干完了。返回一个普通 error，根命令打一份并退 1 —— 与这两个组里 flag 打错的码一致
// （`tt dict r.t --bogus` 也是 1）。
//
// 子命令名一并列出：调用方下一步要的就是那串名字，而 cobra 默认只给一句"unknown command"。
func UnknownSubcommand(cmd *cobra.Command, args []string) error {
	names := make([]string, 0, len(cmd.Commands()))
	for _, c := range cmd.Commands() {
		if c.IsAvailableCommand() && !c.Hidden {
			names = append(names, c.Name())
		}
	}
	return fmt.Errorf("未知子命令 %q（%s）；可用：%s\n  看全部：%s --help",
		args[0], cmd.CommandPath(), strings.Join(names, " / "), cmd.CommandPath())
}
