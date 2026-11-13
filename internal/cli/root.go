// Package cli 装配 tg 的根命令。
//
// 各能力组并存，各自实现在独立子包里，互不 import：
//
//	tg debug …   原 tdebug（AI 人机协同调试）
//	tg dev …     原 tdev（设计器包安全编辑）
//	tg dict …    原 tdict（ERP 数据字典查询）
//	tg db …      数据库连通性测试与只读 SQL
//	tg file …    服务器文件只读浏览与离线源码镜像
//	tg interface … 用 .42f + gae_file 合成前端页面/字段/中文标签
//	tg outline …  4GL 源码大纲（块树 / 行范围）
//
// 共享的 CLI 上下文在 internal/cli/common（叶子包，避免循环依赖）。
package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/at"
	"tg/internal/cli/common"
	"tg/internal/cli/db"
	"tg/internal/cli/debug"
	"tg/internal/cli/dict"
	"tg/internal/cli/file"
	"tg/internal/cli/interfacecmd"
	"tg/internal/cli/outline"
	"tg/internal/cli/session"
	"tg/internal/cli/site"
	"tg/internal/output"
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "tg",
	Short: "TG — TOP GP 开发 CLI",
	Long: `TG 是面向 TOP GP 的命令行工具：查数据字典、看营运据点、连库看数、读服务器源码、
看前端画面、拉离线镜像，收进同一个二进制、同一份 config.json、同一个本地 Web 服务
（tg serve）。

给人和 AI agent 共用：输出简体中文，数据类命令默认 JSON 信封；目标一律点名
（--at <环境>[@<据点编号>] 或 --env，没有默认环境；不知道填什么就加 --at-list）。

能力分组：
  tg dict …      数据字典：表/程序/消息/开窗（r.q）+ 本地字典库同步与状态
  tg site …      营运据点（azp_file）：列全部据点 / 看一个（据点编号 → 资料库编号 → schema）
  tg db …        数据库：连通性测试（ping / probe）与只读 SQL（sql）
  tg file …      服务器文件：ls/find/grep/head/cat 只读浏览（$TOP 与 $CUST 两棵树）
                 + 离线镜像（4gl / 42f / inc）
  tg interface … 前端画面：.42f + gae_file（页面 / 字段 / 中文标签）
  tg outline …   4GL 源码大纲：函数 / UI 块与行范围
  tg debug …     作业调试（T100 的 gendbg 那一套；GP 的调试目前只在网页里）
  tg session …   会话：建 / 列 / 看 / 改 / 收
  tg env / config / serve / install / update   环境、配置、本地服务、安装、自更新

远程操作只读：file 没有 put/rm；镜像用 SFTP 逐个文件下载，不在服务器上留东西。

退出码按命令线各自定义（同一个数字在不同组含义不同，包装脚本按子命令辨认）。
部分命令的码表：
  tg interface  0 成功 / 1 表单不合法 / 2 用法或同名歧义 / 4 精确页面/字段不存在
  tg outline    0 成功 / 2 用法或文件名歧义 / 4 文件不存在`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// 旧命令名别名：合并前各自的二进制名，让既有脚本与 AI skill 不改即可用。
var legacyAliases = map[string]string{
	"tdebug": "debug",
	"tdict":  "dict",
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&common.ConfigPath, "config", "", "配置文件路径 (JSON;缺省取统一用户目录 "+common.ConfigHint()+")")
	pf.BoolVar(&common.JSON, "json", false, "以 JSON 输出（--format json 的语法糖，也是默认）")
	pf.BoolVar(&common.CSV, "csv", false, "以 CSV 输出（--format csv 的语法糖）")
	pf.BoolVar(&common.Human, "human", false, "以人读格式输出（默认 JSON；等价 --format table）")
	pf.StringVar(&common.Format, "format", "json",
		"输出形态: json(默认,带环境信息的信封) | csv(带 # 环境头) | table(人读表格)")
	pf.BoolVarP(&common.Verbose, "verbose", "v", false, "显示解析细节（如实际使用的数据源路径）")
	// 环境一律点名:没有"默认环境"可回落(见 config.Hosts 的注释)。认它的命令见各自 --help。
	pf.StringVar(&common.Env, "env", "", "指定环境名（config.json hosts.sshs 中的 name；环境一律点名，没有默认环境可回落）")
	// --conn 是合并前 tdict 的叫法，保留为别名，但不出现在 --help 里
	pf.StringVar(&common.Env, "conn", "", "指定环境名（--env 的别名）")
	_ = pf.MarkHidden("conn")

	// --at / --at-list：环境+企业的**统一写法**。
	// --at 的值最终落到 common.Env 与命令自己的 --ent 上（见 at.Apply）——
	// 各命令不需要改读法，也不会出现“--at 与 --env 各读一份”。
	// --at-list 在 Execute 的预扫描里处理（cobra 的 ValidateArgs 早于 pre-run）。
	pf.StringVar(&at.Raw, "at", "",
		"这次操作落在哪个目标：<环境> 或 <环境>@<据点编号>（不知道填什么：加 --at-list）")
	pf.BoolVar(&at.List, "at-list", false,
		"列出本命令可用的 --at 取值、来源与示例（纯本地，不联网）")

	debug.Register(rootCmd)
	file.Register(rootCmd)
	db.Register(rootCmd)
	dict.Register(rootCmd)
	// 顶层 site：营运据点（azp_file）。--at 到处要写据点编号，所以它自己一层。
	rootCmd.AddCommand(site.NewSiteCmd())
	session.Register(rootCmd)
	interfacecmd.Register(rootCmd)
	outline.Register(rootCmd)

	rootCmd.AddCommand(newEnvCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newCacheCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newInstallCmd())
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newUpdateCmd())
	// --at 的校验/落位：先于任何 RunE，且子命令自己的 --ent 还没被读。
	// 目标填错属于“用法/身份”错，统一退 2（各命令线的运行期错误另有自己的码）。
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := at.Apply(cmd, args); err != nil {
			return output.Errorf("USAGE", 2, "%v", err)
		}
		return nil
	}
}

// Execute 运行根命令。
// web 承载内嵌的前端构建产物（web/dist）；未构建时可能为 nil 或空目录。
func Execute(web fs.FS) {
	common.WebFS = web
	common.Version = Version

	// 旧命令名兼容：tt tdebug … 等价 tt debug …
	if len(os.Args) > 1 {
		if target, ok := legacyAliases[os.Args[1]]; ok {
			os.Args[1] = target
		}
	}

	// Unix 写法（find -name/-type/-maxdepth）在 pflag 里是非法缩写，先归一化。
	os.Args = file.NormalizeFindArgs(os.Args)

	// --at-list：cobra 解析之前就处理掉（ValidateArgs 比 pre-run 早，
	// 而 tt debug start --at-list 本来就不带作业名）。
	//
	// 它比 flag 解析还早，所以 --config 得在这里先认一遍 —— 否则
	// `tg dict r.t --at-list --config D:\...\config.json` 列的是默认配置里的环境，
	// 用户会照着别处的清单填 --at，然后在校验时才发现"环境不存在"。
	if strings.Contains(strings.Join(os.Args[1:], " "), "--at-list") {
		if p := scanConfigFlag(os.Args[1:]); p != "" {
			common.ConfigPath = p
		}
	}
	if cmd, ok := at.ListTarget(rootCmd, os.Args[1:]); ok {
		code, err := at.PrintContract(os.Stdout, cmd)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if code != 0 {
			os.Exit(code)
		}
		return
	}

	if err := rootCmd.Execute(); err != nil {
		// cobra 的 SilenceErrors 让错误只由这里打一次（各子命令只管 return err）。
		// 少了这一行，命令失败就**一声不响**地退出 1 —— 用户看不到原因，
		// 而这恰恰是最需要说清楚的时候（"为什么服务没起来/为什么没输出"）。
		// 带退出码的错误（既有 TDev 的 0/2/3/4/5 契约，以及 output.Error 的
		// 2=数据源 / 3=缺表）按码退出。
		//
		// **例外**：自己已经把失败打出去了的命令组（见 alreadyReported）跳过这一步 ——
		// 它们各有自己完整的输出契约，再抄一份会让 stdout 上出现两个 JSON 对象。
		var reported alreadyReported
		if !errors.As(err, &reported) || !reported.AlreadyReported() {
			reportError(err)
		}
		if code := exitCodeOf(err); code != 0 {
			os.Exit(code)
		}
		os.Exit(1)
	}
}

// scanConfigFlag 从原始参数里取 --config 的值（cobra 之前先认一遍，只给 --at-list 用）。
//
// 支持 `--config X`、`--config=X`、`-c X`、`-c=X` 四种写法 —— 与 pflag 对同名
// 长短标志的接受范围一致；找不到返回空串。
func scanConfigFlag(args []string) string {
	for i, a := range args {
		for _, name := range []string{"--config", "-c"} {
			if a == name {
				if i+1 < len(args) {
					return args[i+1]
				}
				return ""
			}
			if strings.HasPrefix(a, name+"=") {
				return strings.TrimPrefix(a, name+"=")
			}
		}
	}
	return ""
}

// alreadyReported 由"自己已经把失败打出去了"的命令组实现 —— 根命令据此跳过 reportError。
//
// 为什么需要它：`tt dev` 是完整的一套 CLI（自己的 Usage、自己的退出码、自己把失败打成
// stdout 的信封或帧），而 reportError 在 JSON 模式下也往 stdout 写信封。两份叠在一起的
// 后果是 **stdout 上恰好两个 JSON 对象**：`jq .error.code` 打出两行，其中一行是 null，
// 而"`--json` 下 stdout 恰好一个对象"正是 .tzs 那条线刚立下的规则。
//
// 用结构化接口而不是共享一个类型：记号由命令组自己声明（`exitCode.AlreadyReported`），
// 根命令只问"你报过了吗" —— 不必为了一个记号把 internal/cli/dev 的类型导出。
type alreadyReported interface{ AlreadyReported() bool }

// jsonErrorWriter 由"自己知道失败信封长什么样"的错误实现（`tt debug` 的会话寻址失败
// 就是这一类：它的信封里除了 output.Error 那五键还要带 `candidates`，见契约 5.2 / C5.11）。
//
// 为什么不把 candidates 塞进 internal/output.Error：那是全仓共用的形状，给它加一个
// "只有会话寻址才用得上"的键，等于让每个命令组成员都以为自己该填它。这里用一个
// 只被 `tt debug` 实现的小接口把差异留在它自己包里，根命令只问"你会不会自己打"。
type jsonErrorWriter interface{ WriteEnvelope(w io.Writer) error }

// reportError 打一次错误。
//
// JSON 模式下错误也走 **stdout 的 JSON 信封**（与 tt dev 既有的 {"ok":false,...}
// 同一条路子），让 agent 靠 Code 分支而不是靠正则啃中文；其余模式保持 stderr 散文，
// 并多打一行「提示:」把修复建议单独拎出来。
//
// ⚠️ 这会改变"stderr 有内容 = 失败"这个既有判据：JSON 模式下失败信息在 stdout。
//
// **`tt dev` 不经过这里**：它自己是一套完整的 CLI（自己的 Usage、自己的退出码、
// 自己把失败打成 stdout 的信封或帧），所以它返回的错误实现了 alreadyReported ——
// 否则同一个失败会在 stdout 上出现两份信封，而 `jq .error.code` 会打出两行（一行是 null）。
func reportError(err error) {
	format := common.OutputFormat()
	// 自己会打信封的错误（目前只有会话寻址那一族）：交给它，免得这里的形状
	// 把它多带的键（candidates）吃掉 —— 那是 F1 这类失败的核心信息。
	if format == output.FormatJSON {
		var jw jsonErrorWriter
		if errors.As(err, &jw) {
			if werr := jw.WriteEnvelope(os.Stdout); werr == nil {
				return
			}
		}
	}
	var oe *output.Error
	if errors.As(err, &oe) {
		// 错误自己带了环境信息就用它的,否则补上当前数据源的 ——
		// 连不上库时,"连的是哪个环境"比错误文本本身更难猜。
		if oe.Meta.Env == "" && oe.Meta.Source == "" {
			oe.Meta = common.CurrentMeta()
		}
	} else {
		oe = &output.Error{Code: output.CodeUsage, Exit: exitCodeOf(err),
			Message: err.Error(), Meta: common.CurrentMeta()}
	}
	if oe.Exit == 0 {
		oe.Exit = 1
	}
	if format == output.FormatJSON {
		if werr := output.WriteError(os.Stdout, oe); werr == nil {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "错误: %v\n", oe)
	if oe.Hint != "" {
		fmt.Fprintf(os.Stderr, "提示: %s\n", oe.Hint)
	}
}
