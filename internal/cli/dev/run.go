// 根开关桥：把 tt 根命令的全局开关从透传给动词的原始参数里摘出来、翻译成这条线认识的东西。
//
// 为什么还需要它：动词叶命令是 DisableFlagParsing（参数契约冻结在 tdev 自己的解析器上，
// 见 register.go 的包注释），cobra 不解析这条线上的任何 flag —— 所以 --config/--json/
// --format/--csv/--env 无论写在哪个位置，都会原样落进这里，由本文件翻译。
package dev

import (
	"fmt"
	"os"
	"strings"
)

// takeRootFlags 把根命令的常驻开关从透传给动词的参数里摘出来，翻译成这条线认识的东西。
//
// `--config` 走环境变量而不是直接把值传给解析器：动词核心只经 config.ResolvePath 读配置，
// 而 ResolvePath 本就认 TT_CONFIG —— 不必为这一个 flag 给这条线开新接口。
//
// `--json` / `--csv` / `--format`：根命令的 --help 把它们写作全局开关，但这条线只有
// --json 一个机器可读开关（实测 2026-09-25：`tt dev tzc selftest --format csv` 与
// `tt --format csv dev tzc selftest` 两种位置都只得到「未知子命令」）。翻译表见 devFormat：
// json → --json；table/text → 丢掉（缺省就是人读文本）；csv → **明确拒绝**（这条线没有
// CSV，而"要 CSV 拿到 JSON"正是最该避免的静默走偏）。
//
// `--env/--conn` **现在被接受**（从前这里明确拒绝）：工作区是按环境配的，而"默认环境"
// 已经移除，所以这条线必须能接收一个点名的环境 —— tzs 用它去取该环境的 workspace
// （见 tzs_engine.go 的 envWorkspace）。摘到后经 SetEnvNameFunc 注入的 setter 落到
// cli/common.Env，与 tt debug/tt dict 读的是同一个值。
func takeRootFlags(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	extra := make([]string, 0, 2)
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--config" && i+1 < len(args):
			_ = os.Setenv("TT_CONFIG", args[i+1])
			i++
		case strings.HasPrefix(a, "--config="):
			_ = os.Setenv("TT_CONFIG", strings.TrimPrefix(a, "--config="))
		case a == "--json":
			extra = append(extra, "--json")
		case a == "--csv":
			return nil, devFormatErr("csv")
		case a == "--env" || a == "--conn":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s 需要参数（环境名；可用环境见 tt env list）", a)
			}
			envSetFn(strings.TrimSpace(args[i+1]))
			i++
		case strings.HasPrefix(a, "--env=") || strings.HasPrefix(a, "--conn="):
			envSetFn(strings.TrimSpace(strings.SplitN(a, "=", 2)[1]))
		case a == "--format" && i+1 < len(args):
			v, err := devFormat(args[i+1])
			if err != nil {
				return nil, err
			}
			i++
			if v != "" {
				extra = append(extra, v)
			}
		case strings.HasPrefix(a, "--format="):
			v, err := devFormat(strings.TrimPrefix(a, "--format="))
			if err != nil {
				return nil, err
			}
			if v != "" {
				extra = append(extra, v)
			}
		default:
			out = append(out, a)
		}
	}
	return append(out, extra...), nil
}

// devFormat 把根命令的 --format 取值翻译成这条线认识的东西。返回空串 = 这个取值在本线
// 就是"没有开关"（人读文本是缺省）。
func devFormat(v string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "json":
		return "--json", nil
	case "table", "text":
		return "", nil
	case "csv":
		return "", devFormatErr("csv")
	default:
		return "", fmt.Errorf("--format 不认识 %q：可用 json / table（`tt dev` 这条线不支持 csv）", v)
	}
}

func devFormatErr(v string) error {
	return fmt.Errorf("`tt dev` 这条线没有 %s 输出：要机器可读用 --format json（或 --json），要看人读文本用 --format table", strings.ToUpper(v))
}

// exitCode 是带退出码的错误，由根命令的 exitCodeOf 解出。
type exitCode int

func (e exitCode) Error() string { return fmt.Sprintf("退出码 %d", int(e)) }

// ExitCode 实现根命令期望的退出码接口。
func (e exitCode) ExitCode() int { return int(e) }

// AlreadyReported 告诉根命令「这次失败我已经打出去了」，让它别再抄一份。
//
// 为什么：`tt dev` 是一套完整的 CLI —— 自己的 Usage、自己的退出码、自己把失败打成
// stdout 的信封（或 .tzs 那条线的帧）。失败路径返回前**都**打过一份（`fail()` 或各自的
// 文案），根命令的 reportError 再打一次的后果是 **stdout 上出现两个 JSON 对象**：
// `jq .error.code` 会打出两行，其中一行还是 null。
//
// 记号由命令组自己声明，根命令只问"你报过了吗"（见 internal/cli/root.go 的
// alreadyReported），所以两边不必共享一个类型。
func (e exitCode) AlreadyReported() bool { return true }
