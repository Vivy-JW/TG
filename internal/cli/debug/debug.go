package debug

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	cdb "tt/internal/cli/db"
	"tt/internal/debug"
)

var (
	dbgModule string
	dbgProg   string
	dbgListen string
	dbEnt     int
)

// debugProbeCmd 是调试链路自检：全自动跑 登录→启动→下断点→命中→源码→栈帧→步进→求值，
// 再把"验了什么 / 没验什么"如实报出来。
//
// 收尾判据在 `probe_verdict.go`：**前六步必须每一步都真的走通**才算通过；步进与求值
// 可能因为"程序自己跑完"而没覆盖 —— 那不是链路故障（A35：没有 GDC 接线时作业在 main
// 之后当场 Fatal error -6300 退出），要如实说明并给出病因，而不是一律 exit 1。
var debugProbeCmd = &cobra.Command{
	Use:   "probe",
	Short: "调试链路自检:全自动验一遍协议驱动器(登录/断点/命中/源码/栈帧/步进/求值)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := resolveConfigPath()
		if err != nil {
			return err
		}
		cfg, err := debug.LoadConfig(cfgPath)
		if err != nil {
			return err
		}
		// **环境必须点名**(--env):hosts 里已经没有"默认环境"可回落了,
		// 不点名就没有该连哪台机器可言。与 tt debug start 走同一条解析(见 dbgTargetEnv)。
		if envName := dbgTargetEnv(); envName != "" {
			if sel := cfg.CloneEnv(envName); sel != nil {
				cfg = sel
			} else {
				return fmt.Errorf("环境 %q 不存在(用 tt env list 看全部)", envName)
			}
		}
		cfg.ApplyDefaultEnv() // 环境的连接/参数合并到运行时字段
		if cfg.SSH.Host == "" {
			return fmt.Errorf("没有可连的环境:请用 --env <环境名> 点名(tt env list 看全部)")
		}
		fmt.Printf("[probe] 服务器 %s:%d 区域 %s\n", cfg.SSH.Host, cfg.SSH.Port, cfg.Zone)

		sess, err := debug.NewSession(cfg, dbgModule, dbgProg, "", "", "", func(ev debug.Event) {
			switch ev.Type {
			case "state":
				fmt.Printf("[state] %s\n", ev.State)
			case "watchdog":
				fmt.Printf("[watchdog] %s\n", ev.Text)
			case "output":
				if common.Verbose {
					fmt.Printf("[out] %q\n", ev.Text)
				}
			}
		})
		if err != nil {
			return err
		}
		defer sess.Close()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() { // Ctrl-C 优雅退出
			sig := make(chan os.Signal, 1)
			signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
			<-sig
			cancel()
		}()

		// reached 记**真的走通**的步骤：它既喂收尾判据，也是最后那份"验了什么"的清单。
		// 只在某一步确实成功后追加 —— 失败路径上不许凑数，否则"自检通过"又变回一句空话。
		reached := make([]string, 0, len(probeRequired)+len(probeOptional))

		fmt.Println("[probe] 启动调试会话(Launch)...")
		if err := sess.Launch(ctx); err != nil {
			return fmt.Errorf("Launch 失败: %w", err)
		}
		reached = append(reached, probeLogin)
		fmt.Printf("[probe] 已到 (fgldb) 入口停站 ✓\n\n")
		reached = append(reached, probeEntry)

		fmt.Println("[probe] break main(函数断点):")
		bp, err := sess.Break("main")
		if err != nil {
			return err
		}
		reached = append(reached, probeBreak)
		fmt.Printf("  Breakpoint %d at %s:%d ✓\n", bp.Num, bp.File, bp.Line)

		fmt.Println("\n[probe] run(等待命中断点)...")
		if _, err := sess.Run(); err != nil {
			return err
		}
		stop, err := sess.WaitForStop(60 * time.Second)
		if err != nil {
			return err
		}
		reached = append(reached, probeRun)
		fmt.Printf("  命中: %s:%d (%s) ✓\n", stop.File, stop.Line, stop.Func)
		for _, sl := range stop.Source {
			mark := "   "
			if sl.IsCur {
				mark = "-> "
			}
			fmt.Printf("  %s%-6d %s\n", mark, sl.Num, sl.Text)
		}
		// 源码镜像也算一步：取不到就说明"停站处周边的源码"这条链路没通（源码目录未知那类
		// 故障正是这样表现）—— 空列表不许当成"验过了"。
		if len(stop.Source) > 0 {
			reached = append(reached, probeSource)
		}

		fmt.Println("\n[probe] where:")
		frames, err := sess.Where()
		if err != nil {
			return err
		}
		if len(frames) > 0 {
			reached = append(reached, probeWhere)
		}
		for _, f := range frames {
			fmt.Printf("  #%d %s at %s:%d\n", f.Idx, f.Func, f.File, f.Line)
		}

		// 程序"自己跑完"从这一步起可能随时发生：这个作业在 main 之后要开界面，而画面没接线时
		// 它会当场 Fatal error -6300 退出（A35，真机三次复现）。**那不是链路故障** ——
		// 但也不许拿它当成功糊过去：如实说"步进与求值没覆盖"，并给出能照做的病因。
		exitAt := ""

		fmt.Println("\n[probe] next:")
		stop2, err := sess.Step("next")
		switch {
		case err == nil:
			reached = append(reached, probeStep)
			if stop2 != nil {
				fmt.Printf("  -> %s:%d (%s)\n", stop2.File, stop2.Line, stop2.Func)
				for _, sl := range stop2.Source {
					mark := "   "
					if sl.IsCur {
						mark = "-> "
					}
					fmt.Printf("  %s%-6d %s\n", mark, sl.Num, sl.Text)
				}
			}
		case probeProgramExited(err, sess.State()):
			exitAt = probeStep
			fmt.Printf("  程序自己跑完退出了（不是链路故障）：%v\n", err)
		default:
			return err
		}

		if exitAt == "" {
			fmt.Println("\n[probe] print num_args():")
			v, perr := sess.Print("num_args()")
			switch {
			case perr == nil:
				reached = append(reached, probePrint)
				fmt.Printf("  %s\n", v)
			case probeProgramExited(perr, sess.State()):
				exitAt = probePrint
				fmt.Printf("  程序自己跑完退出了（不是链路故障）：%v\n", perr)
			default:
				// 求值失败与前面的协议步骤无关，不算链路故障；但也绝不装作拿到了值。
				fmt.Printf("  未取到：%v\n", perr)
			}
		}

		ok, missing := probeVerdict(reached)
		printProbeVerdict(reached, missing, ok, exitAt)
		if !ok {
			return fmt.Errorf("probe 未通过：%s 没真的走通（链路故障，与「程序自己跑完」不同）",
				strings.Join(missing, "、"))
		}
		// 只在「程序自己跑完」时给病因：这不是链路问题，但会让人误以为断点没装上。
		if exitAt != "" {
			if hint := debug.ScreenWarnWith(sess.FGLServer(), gdcLocal()); hint != "" {
				fmt.Println("\n  ⚠ " + hint)
			}
		}
		return nil
	},
}

// debugServeCmd M1+:启动本地调试服务(REST+WS+Web 前端)。
// 默认在后台常驻(单实例):打印实际地址后立即返回,终端不被占用;
// 端口被占用时自动顺延到下一个空闲端口,并把真实地址写入状态文件供 debugctl 自动发现。
var debugServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "后台常驻调试服务(tt debug start/exec/status 等控制命令自动发现的守护进程)",
	Long: `后台常驻调试服务 —— tt debug 各条控制命令(start/exec/status/quit/wait/…)自动发现的守护进程。

它与 tt serve 是同一个服务的两种装法,服务端能力完全相同:
  · tt debug serve  只服务调试面:工作台 API 直接在根的 /api/… 下;
  · tt serve        统一服务:调试工作台 /debug/ + 统一设置页,调试面在 /debug/api/… 下。
两者默认都后台常驻、都写同一份状态文件(.tt-serve.json),控制端命令据此自动寻址
(各自的前缀也记在里面),所以 tt debug start/exec/status 对着哪个跑着的实例都能用。
同一个配置目录下只能起一个 —— 两条件会争同一份状态文件。

默认后台运行(单实例):命令打印服务地址与 pid 后立即返回,当前会话可继续输入其它命令;
已有一个实例在跑时打印它的地址后直接返回。
停止用 tt debug serve --stop;前台运行(日志直出终端)用 --foreground。
监听地址取 config.json 顶层 listen(兼容旧结构的 debug.listen;默认 127.0.0.1:28670,不常用端口);
端口被占用时自动顺延到下一个空闲端口并打印真实地址。`,
	Example: `  tt debug serve                  # 后台启动(单实例),打印地址后返回
  tt debug serve --stop          # 停止后台实例
  tt debug serve --foreground    # 前台运行,日志直出终端
  tt debug serve --listen 127.0.0.1:9123`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if dbgServeStop {
			return debugServeStop()
		}
		cfgPath, err := resolveConfigPath()
		if err != nil {
			return err
		}
		cfg, err := debug.LoadConfig(cfgPath)
		if err != nil {
			return err
		}
		cfg.ApplyDefaultEnv() // 默认环境的连接/参数合并到顶层
		if dbgListen != "" {
			cfg.Listen = dbgListen
		}
		// 数据目录 = config.json 所在目录(断点持久化等)
		cfg.DataDir = filepath.Dir(cfgPath)
		if dbgServeForeground {
			return debugServeForeground(cfg, cfgPath)
		}
		return debugServeBackground(cfgPath)
	},
}

// debugDBCmd 已迁移到 tt db probe（服务器侧连接体检）；这里只保留隐藏转发，
// 下一个版本移除。企业/账号来自配置的 ENT 清单，不再现查 gzou_t。
var debugDBCmd = &cobra.Command{
	Use:        "db",
	Short:      "（已迁移）数据库连接体检 —— 用 tt db probe",
	Hidden:     true,
	Deprecated: "已迁移：请用 tt db probe --at <环境>@<企业>",
	RunE: func(cmd *cobra.Command, _ []string) error {
		if dbEnt <= 0 {
			return fmt.Errorf("需要 --at <环境>@<企业>（或 --env + --ent）：验的是这个企业的账号")
		}
		return cdb.ProbeOnce(dbEnt)
	},
}

// printJSON 以 JSON 输出结果(统一走 common.PrintJSON,格式与根命令一致)
func printJSON(v any) error { return common.PrintJSON(v) }

func init() {
	// --module 原为 debug 父命令的持久 flag,扁平化后只挂在真正读取它的命令上。
	// 默认留空 = 登录后按 <作业>.42r 自动解析（不再写死 "asf"：那会短路权威解析，
	// 实测 axmt500_wf 因此被拼成 asf_axmt500_wf.4gl，真名是 axm_axmt500_wf.4gl）。
	debugProbeCmd.Flags().StringVarP(&dbgModule, "module", "m", "", debug.ModuleFlagHelp())
	debugProbeCmd.Flags().StringVarP(&dbgProg, "prog", "p", "bsft001_wf", "作业名(如 bsft001_wf)")
	debugServeCmd.Flags().StringVar(&dbgListen, "listen", "", "覆盖监听地址(默认取配置)")
	debugServeCmd.Flags().BoolVar(&dbgServeForeground, "foreground", false, "前台运行,日志直出终端(默认后台常驻)")
	debugServeCmd.Flags().BoolVar(&dbgServeStop, "stop", false, "停止后台运行的调试服务(单实例)")
	debugDBCmd.Flags().IntVar(&dbEnt, "ent", 0, "企业编号(TOPENT),验证该企业账号连接;0=取该环境的 topent")

	Group.AddCommand(debugProbeCmd, debugServeCmd, debugDBCmd)
}
