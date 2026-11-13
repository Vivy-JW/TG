package file

// tg file mirror：把服务器上的源码/表单文件镜像到本地。
//
//   mirror dir [<目录>]              查看/设置镜像根（config.json 顶层 mirror.dir，或某环境的 mirrorDir）
//   mirror pull [<环境>] [--full]    下载/更新该环境镜像（默认增量）
//   mirror path [<环境>]             打印该环境镜像目录 + 上次拉取时间（AI 直接前往）
//
// 收哪些：两棵树（$TOP 与 $CUST）里的 `*/4gl/*.4gl`、`*/42f/*`、`*.inc`。
// 是"离线口子"：联网时 tg file cat/grep 与 tg outline/interface 直读服务器；
// 断网、或要反复搜同一棵树（tg file grep 是逐文件 SFTP，最吃镜像），读这份镜像。
//
// 不收 42s：GP 上没有这类文件。表单标签在 gae_file 里，由 tg dict db sync
// 落到本地 t_ui_text（tg interface --labels 用），属于字典线，不属于镜像线。

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/mirror"
)

// NewMirrorCmd 构造 mirror 子树（file 组与 dict 的隐藏转发各持一棵，
// 避免同一个 *cobra.Command 被挂到两个父命令下）。
func NewMirrorCmd() *cobra.Command {
	var full bool
	g := &cobra.Command{
		Use:   "mirror",
		Short: "管理本地源码镜像（4gl / 42f / inc，$TOP 与 $CUST 两棵树）",
		Long: `把某环境的源码与表单文件镜像到本地目录，给 tg outline / tg interface / tg file grep 用。

收三类文件（两棵树 $TOP 与 $CUST 都扫）：
  4gl  <模块>/4gl/*.4gl     源码（tg outline 读）
  42f  <模块>/42f/*         编译表单（tg interface 读）
  inc  *.inc                4GL include

不收 42s：GP 没有这类文件；表单标签在 gae_file，由 tg dict db sync 落到本地
t_ui_text（tg interface --labels 用），不在镜像里。

联网时这些命令都能直读服务器，镜像只是"快"和"断网可用"：
一次全量约 400+ MB / 十几分钟，之后增量只拉改过的文件。

子命令：
  dir [<目录>]      查看/设置全局根目录；加 --at <环境> 则设置该环境单独目录
  pull [<环境>]     下载/更新该环境镜像（默认增量；--full 全量重建）
  path [<环境>]     打印该环境镜像目录的绝对路径与上次拉取时间（AI 直接前往）

环境用 --at <环境> 点名（也可给位置参数，二选一）。落点优先级：
环境自己的 mirrorDir > 全局根目录/<环境名> > TG_GP_MIRROR_DIR/<环境名> > ~/.tg/gp-mirror/<环境名>。`,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	pullCmd := newMirrorPullCmd(&full)
	pullCmd.Flags().BoolVar(&full, "full", false, "全量重建（以服务器为准，含删除服务器已不存在的镜像文件）")
	g.AddCommand(newMirrorDirCmd(), pullCmd, newMirrorPathCmd())
	return g
}

// ---- 配置读写 ----

func mirrorConfig() (*config.Hosts, string, error) {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return nil, "", err
	}
	hosts, err := config.LoadHosts(path)
	if err != nil {
		return nil, "", err
	}
	return hosts, path, nil
}

// readMirrorDir 读 config.json 顶层 mirror.dir；返回绝对路径；ok=false=未配置。
func readMirrorDir(cfgPath string) (string, bool) {
	root, err := config.Load(cfgPath)
	if err != nil || root.Mirror.Dir == "" {
		return "", false
	}
	abs, err := filepath.Abs(root.Mirror.Dir)
	if err != nil {
		return root.Mirror.Dir, true
	}
	return abs, true
}

// writeMirrorDir 写回 config.json 顶层 mirror.dir（原子；其余顶层键原样保留）。
func writeMirrorDir(cfgPath, dir string) error {
	return config.EditSection(cfgPath, "mirror", nil, func(sec map[string]any) error {
		sec["dir"] = dir
		return nil
	})
}

// setEnvMirrorDir 写某环境自己的镜像目录（空串 = 删除覆盖，回退全局根目录）。
// 直接改 hosts.sshs[] 的 map，保留环境里本页不认识的键（launchArgs 等）。
func setEnvMirrorDir(cfgPath, env, dir string) error {
	return config.Edit(cfgPath, nil, func(root map[string]any) error {
		hosts, _ := root["hosts"].(map[string]any)
		sshs, _ := hosts["sshs"].([]any)
		for _, item := range sshs {
			m, _ := item.(map[string]any)
			if m == nil || m["name"] != env {
				continue
			}
			if strings.TrimSpace(dir) == "" {
				delete(m, "mirrorDir")
			} else {
				m["mirrorDir"] = dir
			}
			return nil
		}
		return fmt.Errorf("未找到环境 %q", env)
	})
}

// mirrorEnvName 取本次环境：--at 优先，位置参数也可（两者给得不一致就报错）。
func mirrorEnvName(args []string) (string, error) {
	flagEnv := strings.TrimSpace(common.Env)
	name := flagEnv
	if len(args) == 1 {
		arg := strings.TrimSpace(args[0])
		if flagEnv != "" && arg != "" && flagEnv != arg {
			return "", usageErr("--at 的环境（%s）与位置参数（%s）不一致：二选一", flagEnv, arg)
		}
		name = arg
	}
	if name == "" {
		return "", usageErr("需要点名环境：tg file mirror pull --at <环境>（或 tg file mirror path --at <环境>；可用环境见 tg env list）")
	}
	return name, nil
}

func mirrorEnvOf(hosts *config.Hosts, name string) (*config.NamedSsh, error) {
	ref, err := hosts.Resolve(name)
	if err != nil {
		return nil, usageErr("%v", err)
	}
	return ref.Env, nil
}

// ---- mirror dir ----

func newMirrorDirCmd() *cobra.Command {
	var clear bool
	c := &cobra.Command{
		Use:   "dir [<目录>]",
		Short: "查看/设置镜像目录（全局根目录，或 --at 指定环境单独目录）",
		Long: `不带 --at：查看或设置全局镜像根目录（config.json 顶层 mirror.dir）。
带 --at <环境>：查看或设置该环境自己的镜像目录；不设置时回退到全局根目录/<环境名>。
--clear 只清除某环境的单独目录（必须同时给 --at <环境>），回退到全局根目录。
两者都没有时，落点退到 TG_GP_MIRROR_DIR/<环境名> 或 ~/.tg/gp-mirror/<环境名>。

目录可为相对路径，保存时转为绝对路径。`,
		Example: `  tg file mirror dir
  tg file mirror dir D:\dev\erp-src
  tg file mirror dir D:\dev\erp-src-22 --at 22主机
  tg file mirror dir --at 22主机 --clear`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := common.ResolveConfig(false)
			if err != nil {
				return err
			}
			env := strings.TrimSpace(common.Env)
			if clear && env == "" {
				return usageErr("--clear 需要同时给 --at <环境>（只清除该环境的单独镜像目录）")
			}
			if env != "" {
				hosts, err := config.LoadHosts(cfgPath)
				if err != nil {
					return err
				}
				e := hosts.ByName(env)
				if e == nil {
					return usageErr("未找到环境 %q（tg env list 看全部）", env)
				}
				if clear {
					if len(args) == 1 {
						return usageErr("--clear 与目录参数不能同时给")
					}
					if err := setEnvMirrorDir(cfgPath, env, ""); err != nil {
						return err
					}
					e.MirrorDir = "" // 本地副本也要清，下面算回退路径时才是真的回退
					dir := mirror.DirFor(cfgPath, *e)
					if common.IsJSON() {
						return common.PrintJSON(map[string]any{"ok": true, "env": env, "dir": dir, "cleared": true})
					}
					fmt.Printf("已清除环境 %s 的单独镜像目录，现落点: %s\n", env, dir)
					return nil
				}
				if len(args) == 1 {
					abs, err := filepath.Abs(args[0])
					if err != nil {
						return usageErr("解析目录路径失败: %v", err)
					}
					if err := setEnvMirrorDir(cfgPath, env, abs); err != nil {
						return err
					}
					if common.IsJSON() {
						return common.PrintJSON(map[string]any{"ok": true, "env": env, "dir": abs, "updated": true})
					}
					fmt.Printf("环境 %s 的镜像目录已设置: %s\n", env, abs)
					return nil
				}
				dir := mirror.DirFor(cfgPath, *e)
				custom := strings.TrimSpace(e.MirrorDir) != ""
				if common.IsJSON() {
					return common.PrintJSON(map[string]any{"ok": true, "env": env, "dir": dir, "custom": custom})
				}
				fmt.Println(dir)
				return nil
			}
			if len(args) == 1 {
				abs, err := filepath.Abs(args[0])
				if err != nil {
					return usageErr("解析目录路径失败: %v", err)
				}
				if err := writeMirrorDir(cfgPath, abs); err != nil {
					return err
				}
				if common.IsJSON() {
					return common.PrintJSON(map[string]any{"ok": true, "dir": abs, "updated": true})
				}
				fmt.Printf("镜像根目录已设置: %s\n(镜像将保存在 %s\\<环境名>\\ 下；执行 tg file mirror pull --at <环境> 拉取)\n", abs, abs)
				return nil
			}
			dir, ok := readMirrorDir(cfgPath)
			if common.IsJSON() {
				return common.PrintJSON(map[string]any{"ok": true, "dir": dir, "configured": ok})
			}
			if !ok {
				fmt.Printf("镜像根目录未设置。请执行: tg file mirror dir <目录>（或加 --at <环境> 只给该环境）\n" +
					"未设置时落点退到 TG_GP_MIRROR_DIR/<环境名> 或 ~/.tg/gp-mirror/<环境名>。\n")
				return nil
			}
			fmt.Println(dir)
			return nil
		},
	}
	c.Flags().BoolVar(&clear, "clear", false, "清除某环境自己的镜像目录，回退到全局根目录（需 --at）")
	return c
}

// ---- mirror pull ----

func newMirrorPullCmd(full *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "pull [<环境>]",
		Short: "下载/更新该环境镜像（默认增量；--full 全量重建）",
		Long: `把该环境两棵树（$TOP 与 $CUST）里的 4gl / 42f / inc 拉到本地镜像目录。
默认增量：按大小 + 修改时间只补有差异的文件（大小相同但改过的源码也能认出来）。
--full 全量重建：以服务器为准，服务器上已删的文件本地也删掉；不属于白名单的残留不动。

不收 42s：GP 没有这类文件；表单标签在 gae_file（tg dict db sync → t_ui_text）。`,
		Example: `  tg file mirror pull --at 22主机
  tg file mirror pull --at 22主机 --full`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hosts, cfgPath, err := mirrorConfig()
			if err != nil {
				return err
			}
			name, err := mirrorEnvName(args)
			if err != nil {
				return err
			}
			e, err := mirrorEnvOf(hosts, name)
			if err != nil {
				return err
			}
			envDir := mirror.DirFor(cfgPath, *e)
			e.MirrorDir = envDir // 固定落点：Pull 内部用 mirror.Dir(env)
			mode := "incr"
			if *full {
				mode = "full"
			}
			if !common.IsJSON() {
				label := "增量"
				if *full {
					label = "全量"
				}
				fmt.Printf("正在拉取 %s 镜像（%s）: %s -> %s\n", e.Name, label, e.Host, envDir)
			}
			out, err := mirror.Pull(*e, mode, func(done, total int, msg string) {
				if common.IsJSON() || total == 0 {
					return
				}
				if done == total || done%500 == 0 {
					fmt.Printf("  %d/%d 个%s\n", done, total, msg)
				}
			})
			if err != nil {
				return outputErrConn(err)
			}
			if common.IsJSON() {
				return common.PrintJSON(map[string]any{"ok": true, "env": e.Name, "dir": envDir, "mode": mode, "trees": out})
			}
			files, bytes, failed := 0, int64(0), 0
			firstFail := ""
			for label, v := range out {
				m, _ := v.(map[string]any)
				n, _ := m["files"].(int)
				b, _ := m["bytes"].(int64)
				files += n
				bytes += b
				failed += intOr(m["failed"])
				if s, _ := m["firstFail"].(string); s != "" && firstFail == "" {
					firstFail = s
				}
				fmt.Printf("  %s: 下载 %d 个（跳过 %d 个），%.1f MB\n", label, n, intOr(m["skipped"]), float64(b)/1048576)
			}
			fmt.Printf("镜像完成: %d 个文件, %s\n", files, fmtBytes(bytes))
			fmt.Printf("镜像目录: %s\n", envDir)
			if mf, ok := mirror.ReadManifest(envDir); ok {
				fmt.Printf("上次拉取: %s（%d 个文件, %.1f MB）\n", mf.Updated, mf.Files, mf.SizeMB)
			}
			if failed > 0 {
				fmt.Printf("注意: %d 个文件没拉下来（首个: %s）；再跑一次增量会重试这些\n", failed, firstFail)
			}
			return nil
		},
	}
}

// ---- mirror path ----

func newMirrorPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path [<环境>]",
		Short: "打印该环境镜像目录的绝对路径与上次拉取时间（AI 直接前往）",
		Long: `打印该环境镜像目录的绝对路径（第一行即路径本身，便于 AI/脚本取用），并附上次拉取时间。
目录尚未拉取时会在路径后给提示。`,
		Example: `  tg file mirror path --at 22主机`,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hosts, cfgPath, err := mirrorConfig()
			if err != nil {
				return err
			}
			name, err := mirrorEnvName(args)
			if err != nil {
				return err
			}
			e, err := mirrorEnvOf(hosts, name)
			if err != nil {
				return err
			}
			p := mirror.DirFor(cfgPath, *e)
			mf, ok := mirror.ReadManifest(p)
			if common.IsJSON() {
				v := map[string]any{"ok": true, "env": e.Name, "path": p, "exists": ok}
				if ok {
					v["files"] = mf.Files
					v["sizeMB"] = mf.SizeMB
					v["at"] = mf.Updated
					v["kinds"] = mf.Kinds
				}
				return common.PrintJSON(v)
			}
			fmt.Println(p)
			if !ok {
				fmt.Printf("(该环境尚未拉取；先执行 tg file mirror pull --at %s)\n", e.Name)
				return nil
			}
			fmt.Printf("(上次拉取 %s，%d 个文件，%.1f MB)\n", mf.Updated, mf.Files, mf.SizeMB)
			return nil
		},
	}
}

// intOr 把 map 里的数字转 int（JSON 解出来的可能是 float64，这里只管本进程内构造的值）。
func intOr(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// fmtBytes 字节数人性化显示。
func fmtBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
