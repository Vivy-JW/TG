// Package session 是 `tg session` 命令组：**会话是一级对象**的那套命令面。
//
// # 它和 `tg debug` 的关系
//
// `tg debug` 是"调试器动词"（break/step/where/…），`tg session` 是"会话本身"
// （建 / 列 / 看 / 收）。两者都直连同一个调试面（`internal/debugclient`），
// 也都用同一个地址体系（`ref` / `name` / 句柄）。
//
// # 身份只有环境
//
// TOP GP 没有企业编号那一层：账号由服务端取 db.accounts[0]，所以这里不提供
// `--ent` / `--fglserver`，也没有 `session ent`。这一批覆盖会话主体：
// 建、列、看、结束本轮、收掉、透传。它们不依赖任何新端点。
package session

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/debugclient"
	"tg/internal/output"
)

// group 是 `tg session` 的命令组。
var group = &cobra.Command{
	Use:   "session",
	Short: "会话：建 / 列 / 看 / 收（会话是一级对象）",
	Long: `会话（session）是唯一底座：它持有身份（ref / name / n / epoch）与作业实例（环境 / 作业）。

  tg session new   --env 22主机 --name A [--prog bsft001_wf]
  tg session ls
  tg session show  --session A
  tg session quit  --session A     # 结束本轮，会话保留
  tg session rm    --session A     # 收掉会话（断连接）
  tg session exec  --session A -- "break 812"

只要环境：TOP GP 的账号由服务端取 db.accounts[0]，没有企业编号要填。

目标一律用 --session，接受 ref（环境#序号）、别名、句柄；不给则只有一条时才落位，
多条会报歧义并列候选（绝不替你挑一条）。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return failUsage("未知子命令 %q", args[0])
	},
}

// init 让**标志/参数解析错**也按这条线的表退 2。
//
// cobra 默认把它当普通错误 → 根命令退 1，而这张表里 1 = 运行失败、2 = 用法错。
// 少了它，`tg session ls --bogus` 会被脚本当成"运行失败"重试。
func init() {
	group.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return failUsage("%v", err)
	})
}

// Register 把这一组挂到根命令上（与其它命令组同一套注册方式）。
func Register(root *cobra.Command) { root.AddCommand(group) }

// ---------- 公共 ----------

// failUsage 造一个"用法错"（退出码 2）。
func failUsage(format string, args ...any) error {
	return &exitErr{code: 2, msg: fmt.Sprintf(format, args...)}
}

// exitErr 带退出码的错误（根命令读 ExitCode）。
type exitErr struct {
	code int
	msg  string
}

func (e *exitErr) Error() string { return e.msg }
func (e *exitErr) ExitCode() int { return e.code }

// newClient 是**可替换的**客户端构造（测试用它指向假服务）。
//
// 生产实现按配置目录算地址与前缀（都来自服务状态文件）；测试里把它换掉即可，
// 不必去摆弄真实的配置目录与端口。
var newClient = func() (*debugclient.Client, error) {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return nil, err
	}
	loc := config.LocationsAt(cfgPath)
	dir := loc.DataDir()
	if dir == "" {
		return nil, failUsage("配置路径没解析出数据目录，先跑 tg config path 看看")
	}
	return debugclient.New(debugclient.Discover(dir, "")), nil
}

// resolve 把 --session 的取值解析成一条会话。
func resolve(cl *debugclient.Client, want string) (debugclient.Brief, error) {
	if strings.TrimSpace(want) == "" {
		return debugclient.Brief{}, failUsage("需要 --session（别名 / 环境#序号 / 句柄）；先 tg session ls 看现有会话")
	}
	return cl.Resolve(want)
}

// emit 打印 JSON（本组统一：默认 JSON，人读另说）。
func emit(v any) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(v)
}

// briefMap 把一条会话摊成"输出里该有的那几个键"（身份 + 作业状态）。
//
// 不含 `ent`：TG 只服务 TOP GP，会话身份只有环境；服务端快照里仍带 ent 是
// 既有契约（Web 接口日志页还在读），但命令行输出不把它摆出来。
func briefMap(b debugclient.Brief) map[string]any {
	return map[string]any{
		"id": b.ID, "ref": b.Ref, "display": b.Display, "name": b.Name,
		"env": b.Env, "n": b.N, "epoch": b.Epoch,
		"prog": b.Prog, "state": b.State,
	}
}

// gpSnapshot 去掉服务端快照里的 T100 身份键（`ent` 与三个 `topent*`）。
//
// 为什么在 CLI 这层投影、而不是让服务端不返回：那几个键是 Web 接口日志页/调试
// 工作台的既有契约（`internal/debug` 的 api_snapshot_test.go 钉着），删了是另一批事。
// `tg session show` 是给人/脚本看的 GP 视图，T100 的企业编号在这里没有含义。
func gpSnapshot(snap map[string]any) map[string]any {
	out := make(map[string]any, len(snap))
	for k, v := range snap {
		switch k {
		case "ent", "topent", "topentSession", "topentShell":
			continue
		}
		out[k] = v
	}
	return out
}

// emitFormatted 按全局 `--format` 输出：JSON 默认；table/text/csv 走人读形态。
//
// 为什么要它：新命令一开始一律打 JSON，而全局 `--format table` 被**静默忽略** ——
// 那正是这套工具反复在灭的毛病（"我明明给了参数，它当没看见"）。
func emitFormatted(v any, text func()) error {
	if common.OutputFormat() == output.FormatJSON {
		return emit(v)
	}
	text()
	return nil
}

// errf 把用法/运行错按线里约定的码返回。
func errf(code int, format string, args ...any) error {
	return &exitErr{code: code, msg: fmt.Sprintf(format, args...)}
}
