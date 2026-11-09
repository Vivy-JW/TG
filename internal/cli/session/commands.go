package session

// `tt session` 的子命令：建 / 列 / 看 / 改 ENT / 结束本轮 / 收掉 / 透传。
//
// 每一个都只做"参数 → 一次客户端调用 → 输出"，不含编排。命令层保持薄。

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/debugclient"
	"tt/internal/output"
)

var (
	// new
	nsEnv, nsEnt, nsName, nsProg, nsModule string
	nsTimeout                              int
	nsFGLServer                            string // --fglserver：本轮画面接线（会话 job 用）
	// 定位
	lsSession, lsEnv string
	lsAll            bool
	// 属性
	entValue string
	entClear bool
	// 透传

)

func init() {
	newCmd := &cobra.Command{
		Use:   "new",
		Short: "建会话（可同时起作业）",
		Long: `建一条会话。

不给 --prog 建出来的是**空闲会话**（已登录、无作业）：之后可以 tt session exec / tt debug 操作它，
也可以再用 tt debug start 在里面起作业。
给了 --prog 则同时起作业并等到入口停站。

ENT 必填：企业编号决定用哪个账号连库与据点校验，不再沿用登录 shell 的 $TOPENT。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			env := strings.TrimSpace(nsEnv)
			if env == "" {
				env = strings.TrimSpace(common.Env) // --at 落到 common.Env；本命令自己的 --env 优先
			}
			ent := strings.TrimSpace(nsEnt)
			if env == "" {
				return failUsage("需要 --at <环境>@<企业> 或 --env <环境名>（tt env list 看全部）")
			}
			if ent == "" {
				return failUsage("需要 --at <环境>@<企业> 或 --ent <企业编号>（决定用哪个账号连库；tt debug ents 可查）")
			}
			cl, err := newClient()
			if err != nil {
				return err
			}
			prog := strings.TrimSpace(nsProg)

			if prog == "" {
				// 空闲会话：走"切到该环境并命名"，服务端会建一条 idle 的宿主。
				id, serr := cl.SwitchToIdle(env, ent, strings.TrimSpace(nsName), time.Duration(nsTimeout)*time.Second)
				if serr != nil {
					return serr
				}
				return emitSession(cl, id)
			}
			// 有作业：建会话 + 起作业 + 等入口停站。
			id, lerr := cl.Launch(launchReq(env, ent, prog))
			if lerr != nil {
				return lerr
			}
			if _, ok, werr := cl.WaitStopped(id, time.Duration(nsTimeout)*time.Second); werr != nil {
				return werr
			} else if !ok {
				return fmt.Errorf("起作业后没等到入口停站（%s）", prog)
			}
			return emitSession(cl, id)
		},
	}

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "列会话",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newClient()
			if err != nil {
				return err
			}
			list, lerr := cl.Sessions()
			if lerr != nil {
				return lerr
			}
			out := []map[string]any{}
			var rows [][]string
			wantEnv := strings.TrimSpace(lsEnv)
			if wantEnv == "" {
				wantEnv = strings.TrimSpace(common.Env) // --at 落到 common.Env
			}
			for _, b := range list {
				if wantEnv != "" && b.Env != wantEnv {
					continue
				}
				if !lsAll && b.State == "exit" {
					continue
				}
				out = append(out, briefMap(b))
				rows = append(rows, []string{b.Ref, b.Name, b.Env, b.Ent, b.Prog, b.State, b.Epoch})
			}
			payload := map[string]any{"ok": true, "sessions": out}
			headers := []string{"REF", "名字", "环境", "ENT", "作业", "用途", "状态", "世代"}
			switch common.OutputFormat() {
			case output.FormatJSON:
				return emit(payload)
			case output.FormatCSV:
				// 自己写 CSV：`output.WriteCSV` 会带上 SQL 那一套表头注释（"只读事务 / 行数 0"），
				// 对会话列表是误导 —— 这里明明有行，注释却说 0 行。
				cw := csv.NewWriter(os.Stdout)
				_ = cw.Write(headers)
				for _, row := range rows {
					_ = cw.Write(row)
				}
				cw.Flush()
				return cw.Error()
			default:
				return output.WriteTable(os.Stdout, headers, rows)
			}
		},
	}

	showCmd := &cobra.Command{
		Use:   "show",
		Short: "看一条会话的详情（身份 / 作业 / 停站位置）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			snap, serr := cl.Snapshot(b.ID)
			if serr != nil {
				return serr
			}
			return emitFormatted(snap, func() {
				fmt.Printf("会话  : %s（%s）\n", b.Ref, b.Name)
				fmt.Printf("环境  : %s / ENT %s\n", b.Env, b.Ent)
				fmt.Printf("作业  : %s（%s）\n", b.Prog, b.State)
			})
		},
	}

	entCmd := &cobra.Command{
		Use:   "ent <企业编号>",
		Short: "改会话的企业编号（仅空闲时可设）",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			value := strings.TrimSpace(entValue)
			if len(args) == 1 && value == "" {
				value = strings.TrimSpace(args[0])
			}
			if entClear {
				value = ""
			}
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			if terr := cl.Topent(b.ID, value); terr != nil {
				return terr
			}
			return emitFormatted(map[string]any{"ok": true, "session": b.Ref, "ent": value}, func() {
				if value == "" {
					fmt.Printf("已清除会话 %s 的企业编号覆盖（回到环境默认）\n", b.Ref)
				} else {
					fmt.Printf("会话 %s 的企业编号：%s\n", b.Ref, value)
				}
			})
		},
	}

	quitCmd := &cobra.Command{
		Use:   "quit",
		Short: "结束本轮调试（会话保留，回空闲）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			if qerr := cl.Quit(b.ID); qerr != nil {
				return qerr
			}
			return emitFormatted(map[string]any{"ok": true, "session": b.Ref, "state": "idle"}, func() {
				fmt.Printf("已结束会话 %s 的本轮调试（会话保留，回空闲）\n", b.Ref)
			})
		},
	}

	rmCmd := &cobra.Command{
		Use:   "rm",
		Short: "收掉会话（断连接、清资源）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			if cerr := cl.Close(b.ID); cerr != nil {
				return cerr
			}
			return emitFormatted(map[string]any{"ok": true, "session": b.Ref, "removed": true}, func() {
				fmt.Printf("已收掉会话 %s\n", b.Ref)
			})
		},
	}

	jobCmd := &cobra.Command{
		Use:   "job --prog <作业>",
		Short: "在这条会话里起一个作业（会话本来就空闲）",
		Long: "在指定会话里起作业。\n\n" +
			"与 `tt session new --prog` 的分工：那条是“建会话（顺带起作业）”；这条是“会话已经在了，\n" +
			"只在它里面起作业”。典型用法是同一个宿主上换作业跑：`tt session quit` → `tt session job --prog …`。",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			prog := strings.TrimSpace(nsProg)
			if prog == "" {
				return failUsage("需要 --prog：在这条会话里起的是哪个作业")
			}
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			id, jerr := cl.StartJob(b.ID, prog, nsModule, nsEnt, nsFGLServer)
			if jerr != nil {
				return jerr
			}
			return emitFormatted(map[string]any{"ok": true, "session": b.Ref, "sessionId": id, "prog": prog}, func() {
				fmt.Printf("已在会话 %s 里起作业 %s\n", b.Ref, prog)
			})
		},
	}

	execCmd := &cobra.Command{
		Use:   "exec -- \"<fgldb 命令>\"",
		Short: "透传一条 fgldb 命令（等价 tt debug exec）",
		Long: `透传一条 fgldb 原生命令，原样返回输出。

  tt session exec --session A -- "break 812"
  tt session exec --session A -- "print g_qryparam.cond"

-- 之后整段原样交给调试器，不再由 tt 解析。`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			command := strings.TrimSpace(strings.Join(args, " "))
			if command == "" {
				return failUsage(`需要一条命令：tt session exec --session A -- "break 812"`)
			}
			cl, err := newClient()
			if err != nil {
				return err
			}
			b, rerr := resolve(cl, lsSession)
			if rerr != nil {
				return rerr
			}
			lines, xerr := cl.Exec(b.ID, command)
			if xerr != nil {
				return xerr
			}
			return emitFormatted(map[string]any{"ok": true, "session": b.Ref, "command": command, "lines": lines}, func() {
				// 透传的输出原样给：它就是调试图输出，加壳反而不好读。
				for _, line := range lines {
					fmt.Println(line)
				}
			})
		},
	}

	newCmd.Flags().StringVar(&nsEnv, "env", "", "环境名（必填）")
	newCmd.Flags().StringVar(&nsEnt, "ent", "", "企业编号（必填）")
	newCmd.Flags().StringVar(&nsName, "name", "", "别名（跨轮追踪用它；1–64 字符，禁 / 与 #）")
	newCmd.Flags().StringVar(&nsProg, "prog", "", "作业程序名（给了就同时起作业）")
	newCmd.Flags().StringVarP(&nsModule, "module", "m", "", "T100 模块目录名（留空由作业名自动解析）")
	newCmd.Flags().IntVar(&nsTimeout, "timeout", 120, "等待空闲 / 入口停站的超时（秒）")

	lsCmd.Flags().StringVar(&lsEnv, "env", "", "只看这个环境")
	lsCmd.Flags().BoolVarP(&lsAll, "all", "a", false, "含已断开的会话")

	showCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")

	entCmd.Flags().StringVar(&entValue, "value", "", "企业编号（也可用位置参数）")
	entCmd.Flags().BoolVar(&entClear, "clear", false, "清除会话级覆盖（回到登录默认）")
	entCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")

	quitCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")
	rmCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")
	jobCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")
	jobCmd.Flags().StringVar(&nsProg, "prog", "", "作业程序名（必填）")
	jobCmd.Flags().StringVarP(&nsModule, "module", "m", "", "T100 模块目录名（留空用 asf）")
	jobCmd.Flags().StringVar(&nsEnt, "ent", "", "企业编号（留空用会话自己的）")
	jobCmd.Flags().StringVar(&nsFGLServer, "fglserver", "", "本轮画面接线（<ip>:<port>；留空按登录 shell 的 FGLSERVER 走）")
	execCmd.Flags().StringVar(&lsSession, "session", "", "目标会话（ref / 别名 / 句柄）")

	group.AddCommand(newCmd, jobCmd, lsCmd, showCmd, entCmd, quitCmd, rmCmd, execCmd)
}

// launchReq 组装"建会话 + 起作业"的请求（ENT 必填，模块可留空）。
func launchReq(env, ent, prog string) debugclient.LaunchReq {
	return debugclient.LaunchReq{
		Env: env, Ent: ent, Prog: prog,
		Module: strings.TrimSpace(nsModule),
		Name:   strings.TrimSpace(nsName),
	}
}

// emitSession 按句柄取一次会话并输出（建完/改完之后用它回显"落在哪条"）。
func emitSession(cl *debugclient.Client, id string) error {
	list, err := cl.Sessions()
	if err != nil {
		return err
	}
	for _, b := range list {
		if b.ID == id {
			return emit(map[string]any{"ok": true, "session": briefMap(b)})
		}
	}
	// 建完立刻列不到（极少数竞态）：至少把句柄给出来，别让调用方以为没建成。
	return emit(map[string]any{"ok": true, "session": map[string]any{"id": id}})
}
