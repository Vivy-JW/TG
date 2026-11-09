package cli

// tt update —— tt 自身的更新：查有没有新版、下载并校验制品、把"替换二进制"交给一个
// 脱离的更新器进程。
//
// 四条边界（读代码前先读它们）：
//
//   - **默认不装**。`tt update` 会问一句，`--yes` 才免确认；非交互环境（agent/CI）
//     没给 `--yes` 就拒绝，而不是猜。
//   - **不对源码态动手**。`make build` 出的仓库根 tt.exe 只允许查，装会被拒（exit 4）。
//   - **只在显式命令里联网**。`tt serve` 与日常命令都不查；`tt version` 只读缓存。
//   - 引擎与设计器 dll 随包走，不单独升 —— 更新单位是整包/MSI。
//
// 退出码（这条线自己的一张表，与 tt dev / tt dict 各不相干，见根帮助的退出码表）：
//
//	0  已是最新 / 更新已交棒 / 用户取消了安装
//	10 有可用更新（新号：3 在这条线上是"校验失败"，不能混用）
//	2  用法错（含非交互环境没给 --yes）
//	3  校验失败（摘要不符、制品自报版本不符）
//	4  拒绝写入（这份 tt 不让自装）
//	5  出网失败（代理、限流、超时）或落盘失败
//
// 10 不是错误：`tt update check` 已经把自己的结论打到 stdout 了，所以它带着这个码返回
// 时**不许**再打一份错误信封 —— 记号见 exitStatus（与 internal/cli/dev 的
// AlreadyReported 同一个机制，理由写在 root.go 的 reportError 上）。

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/cli/debug"
	"tt/internal/config"
	"tt/internal/dev/tzs"
	"tt/internal/output"
	"tt/internal/update"
	"tt/internal/winproc"
)

// exitUpdateAvailable "有可用更新"。它不是失败，见上面那段说明。
const exitUpdateAvailable = 10

// exitStatus 表达"命令已经把自己的结论打出去了，只需要用一个非零码收场"。
// 根命令认 AlreadyReported 这个记号并跳过 reportError，否则 stdout 上会多出一份
// 错误信封（JSON 消费者会看到两个对象）。
type exitStatus int

func (e exitStatus) Error() string         { return fmt.Sprintf("退出码 %d", int(e)) }
func (e exitStatus) ExitCode() int         { return int(e) }
func (e exitStatus) AlreadyReported() bool { return true }

// updateFlags tt update 家族的命令行开关。
type updateFlags struct {
	yes       bool
	dryRun    bool
	checkOnly bool
	pre       bool
	proxy     string
	plan      string
}

func newUpdateCmd() *cobra.Command {
	var f updateFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "更新 tt 自身（查新版、下载校验、交棒给更新器）",
		Long: `更新 tt 自身。

默认只做"查 + 下载 + 校验"，然后**问你一句**再装；--check-only 只查不装，
--dry-run 走到交棒前一步全停。非交互环境（agent / 脚本）必须显式给 --yes。

  三种安装形态各自怎么装：
    便携包（exe 同目录有 .portable）  下载 zip，解到旁边校验后**就地覆盖**目录
    MSI 安装（%LOCALAPPDATA%\Programs\TT）  下载 MSI，交给 msiexec /qb
    认不出的布局 / 源码态（make build）    只允许查，装会被拒（退出码 4）

  凭什么信得过：
    1. 更新源写死在代码里（` + update.RepoPage + `），没有指向别处的开关；
    2. 下载完对发布方给的 sha256 摘要，不符就删掉；
    3. 装之前跑一遍制品的 ` + "`tt version`" + `，自报版本必须等于目标版本。

  装的动作由**另一个进程**完成：Windows 上运行中的 exe 不能替换自己，所以
  ` + "`tt update`" + ` 把自己复制到 %TEMP% 再交棒，前台随即退出。升级日志落在数据目录的
  .tt-update.log，` + "`tt update log`" + ` 可以回读它走到哪一步。

退出码: 0 已是最新或已交棒 | 10 有可用更新 | 2 用法错 | 3 校验失败 |
        4 拒绝自装 | 5 出网或落盘失败`,
		Example: `  tt update check              # 只查（有新版时退出码 10）
  tt update --dry-run          # 走到交棒前停，看它打算做什么
  tt update                    # 查 + 下载 + 校验，问一句再装
  tt update --yes              # 不问，直接装（脚本用）
  tt update --pre --proxy http://127.0.0.1:10808
  tt update log                # 上一次升级走到哪一步了`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUpdate(cmd, f, false)
		},
	}
	fl := cmd.Flags()
	fl.BoolVar(&f.yes, "yes", false, "不问，直接安装")
	fl.BoolVar(&f.dryRun, "dry-run", false, "只走到交棒前一步，不安装")
	fl.BoolVar(&f.checkOnly, "check-only", false, "--check-only 的别名（只查不装）")
	fl.BoolVar(&f.pre, "pre", false, "连预发布一起看（缺省只看正式发布）")
	fl.StringVar(&f.proxy, "proxy", "", "代理地址（缺省取配置的 net.proxy，再看环境变量）")

	cmd.AddCommand(newUpdateCheckCmd(), newUpdateLogCmd(), newUpdateApplyCmd())
	return cmd
}

func newUpdateCheckCmd() *cobra.Command {
	var f updateFlags
	cmd := &cobra.Command{
		Use:   "check",
		Short: "查有没有新版（不下载、不安装）",
		Long: `查一次更新源，与手上这份 tt 比较。结论写进缓存（<数据目录>/update/check.json），
` + "`tt version`" + ` 与设置页离线读它。

只查不动手，所以这份 tt 是什么形态都不影响它。

退出码: 0 已是最新 | 10 有可用更新 | 5 出网失败`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f.checkOnly = true
			return runUpdate(cmd, f, true)
		},
	}
	cmd.Flags().BoolVar(&f.pre, "pre", false, "连预发布一起看")
	cmd.Flags().StringVar(&f.proxy, "proxy", "", "代理地址（缺省取配置的 net.proxy，再看环境变量）")
	return cmd
}

func newUpdateLogCmd() *cobra.Command {
	var tail int
	cmd := &cobra.Command{
		Use:   "log",
		Short: "回读最近一次升级的阶段与结果",
		Long: `回读升级状态（<数据目录>/.tt-update.json）与日志尾部（<数据目录>/.tt-update.log）。

更新器是脱离了终端跑的进程：装完它改的是已经装好的那份 tt，前台早退出了。
"到底装成了没有、卡在哪一步"只能从这里回答。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, err := common.ResolveConfig(true)
			if err != nil {
				return err
			}
			dataDir := filepath.Dir(cfgPath)
			st := update.LoadState(dataDir)
			if common.IsJSON() {
				var tailText string
				if st != nil {
					logPath := st.LogPath
					if logPath == "" {
						logPath = update.LogPath(dataDir)
					}
					if b, err := os.ReadFile(logPath); err == nil {
						tailText = tailLines(string(b), tail)
					}
				}
				return common.PrintJSON(map[string]any{"ok": true, "state": st, "log": tailText})
			}
			if st == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "还没有升级记录。")
				return nil
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "阶段: %s\n", st.Phase)
			if st.From != "" || st.Target != "" {
				fmt.Fprintf(out, "版本: %s → %s\n", orDash(st.From), orDash(st.Target))
			}
			if st.Kind != "" {
				fmt.Fprintf(out, "形态: %s\n", st.Kind)
			}
			if st.Artifact != "" {
				fmt.Fprintf(out, "制品: %s\n", st.Artifact)
			}
			if !st.StartedAt.IsZero() {
				fmt.Fprintf(out, "开始: %s\n", st.StartedAt.Local().Format("2006-01-02 15:04:05"))
			}
			if !st.FinishedAt.IsZero() {
				fmt.Fprintf(out, "结束: %s\n", st.FinishedAt.Local().Format("2006-01-02 15:04:05"))
			}
			if st.Error != "" {
				fmt.Fprintf(out, "失败原因: %s\n", st.Error)
			}
			for _, w := range st.Warnings {
				fmt.Fprintf(out, "警告: %s\n", w)
			}
			logPath := st.LogPath
			if logPath == "" {
				logPath = update.LogPath(dataDir)
			}
			if b, err := os.ReadFile(logPath); err == nil {
				fmt.Fprintf(out, "\n日志尾部（%s）:\n", logPath)
				fmt.Fprint(out, tailLines(string(b), tail))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&tail, "tail", 20, "日志尾部回读多少行")
	return cmd
}

// newUpdateApplyCmd 是更新器本体。**隐藏**：它不是给人敲的，参数由前台算好写进计划文件。
// 留在命令树里是因为更新器就是同一个 tt.exe 的副本（见 update.Handoff 的说明）。
func newUpdateApplyCmd() *cobra.Command {
	var f updateFlags
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "（内部）执行升级计划",
		Long: `（内部）执行升级计划。不手工运行：参数由 tt update 算好写进计划文件。

进展**只写日志**（数据目录的 .tt-update.log）：这个进程由 SpawnDetached 拉起，
它的 stdout/stderr 已经被重定向到同一个日志文件，再往 stderr 写一遍会让每一行
出现两次。看进展用 ` + "`tt update log`" + `。`,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if f.plan == "" {
				return usageErr("缺少 --plan", "这个子命令由 tt update 交棒时调用，不手工运行")
			}
			plan, err := update.LoadPlan(f.plan)
			if err != nil {
				return applyErr(err)
			}
			logf := func(format string, a ...any) {
				update.AppendLog(update.LogPath(plan.DataDir), format, a...)
			}
			if err := update.Apply(cmd.Context(), plan, logf); err != nil {
				return applyErr(err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&f.plan, "plan", "", "升级计划文件（由 tt update 生成）")
	return cmd
}

// checkOnlyReport 只查不装时的输出（--json 下进信封，table 下是人读几行）。
type checkOnlyReport struct {
	Kind        string    `json:"kind"`
	Current     string    `json:"current"`
	Latest      string    `json:"latest"`
	Newer       bool      `json:"newer"`
	Prerelease  bool      `json:"prerelease,omitempty"`
	PublishedAt time.Time `json:"publishedAt,omitempty"`
	CheckedAt   time.Time `json:"checkedAt"`
	Proxy       string    `json:"proxy,omitempty"`
	Source      string    `json:"source"`
	CanInstall  bool      `json:"canInstall"`
	Refusal     string    `json:"refusal,omitempty"`
	Assets      []string  `json:"assets,omitempty"`
}

// runUpdate 是 tt update / tt update check 的公共主体。
// checkOnlyMode 由子命令写死（tt update check 只查），而 --check-only 是同一个语义的开关。
func runUpdate(cmd *cobra.Command, f updateFlags, checkOnlyMode bool) error {
	ctx := cmd.Context()
	errOut := cmd.ErrOrStderr()
	checkOnly := checkOnlyMode || f.checkOnly || f.dryRun

	env, err := resolveUpdateEnv(f.proxy)
	if err != nil {
		return err
	}

	// 一次运行清一次上次留下的临时目录：更新器删不掉自己（Windows 上运行中的文件删不掉），
	// 失败的升级也可能留下没来得及清掉的暂存目录。两者都留给下一次启动来清。
	update.CleanStaleHelpers(time.Now())
	update.CleanStaleStages(env.exe, time.Now())
	rep := &checkOnlyReport{
		Kind: env.kind.String(), Current: env.current, Proxy: env.proxy,
		Source: update.RepoSlug, CheckedAt: time.Now(),
	}
	rep.CanInstall = update.InstallRefusal(env.kind, env.exe, env.current) == ""
	if !rep.CanInstall {
		rep.Refusal = update.InstallRefusal(env.kind, env.exe, env.current)
	}

	res, err := update.Check(ctx, env.client, env.current, f.pre)
	if err != nil {
		return netErr(err)
	}
	// 缓存只服务离线提示（tt version / 设置页），落不下去不影响这次检查的结论。
	_ = update.SaveCheck(env.dataDir, res)

	rep.Latest, rep.Newer, rep.Prerelease = res.Latest, res.Newer, res.Prerelease
	rep.PublishedAt, rep.CheckedAt = res.PublishedAt, res.CheckedAt
	for _, a := range res.Assets {
		rep.Assets = append(rep.Assets, a.Name)
	}
	if err := emitCheck(cmd, rep, env); err != nil {
		return err
	}

	if !res.Newer {
		return nil // 已是最新，退 0
	}
	if checkOnly {
		// 有新版但只让查：这不是失败，所以带码返回且不再打错误信封。
		return exitStatus(exitUpdateAvailable)
	}
	if !rep.CanInstall {
		return &output.Error{Code: output.CodeUpdateRefused, Exit: 4,
			Message: rep.Refusal, Hint: "把发行包装到它认得的位置再来更新，或手动替换"}
	}

	// ---- 下载 + 校验 ----
	asset, err := update.PickAsset(res.Assets, env.kind)
	if err != nil {
		return &output.Error{Code: output.CodeUpdateChecksum, Exit: 3, Message: err.Error()}
	}
	hex := asset.HexDigest()
	if hex == "" {
		return &output.Error{Code: output.CodeUpdateChecksum, Exit: 3,
			Message: fmt.Sprintf("发布 %s 的资产 %s 没有 sha256 摘要，无法校验", res.Latest, asset.Name),
			Hint:    "没有摘要就没有\"下到的字节是发布方发的\"这条判据，拒绝安装；请联系发布方重发"}
	}
	dest := filepath.Join(update.UpdateDir(env.dataDir), asset.Name)
	fmt.Fprintf(errOut, "下载 %s（%s）→ %s\n", asset.Name, update.HumanBytes(asset.Size), dest)
	if err := env.client.Download(ctx, asset, dest, hex, progressTo(errOut, asset.Name)); err != nil {
		return netErr(err)
	}
	fmt.Fprintf(errOut, "校验通过: sha256 %s\n", hex)

	plan := &update.Plan{
		Kind: env.kind.Token(), From: env.current, Target: res.Latest,
		ExePath: env.exe, Digest: hex, DataDir: env.dataDir,
		Artifact: dest, WaitPID: os.Getpid(),
	}

	// ---- 就位：便携包先解到旁边、验一遍制品自报版本 ----
	if env.kind == update.KindPortable {
		stage, err := stagePortable(ctx, dest, env.exe, res.Latest, errOut)
		if err != nil {
			return err
		}
		plan.Dir = filepath.Dir(env.exe)
		plan.Stage = stage
	} else if asset.Size > 0 {
		if fi, err := os.Stat(dest); err == nil && fi.Size() != asset.Size {
			return &output.Error{Code: output.CodeUpdateChecksum, Exit: 3,
				Message: fmt.Sprintf("下载的 %s 大小 %d 与发布声明 %d 不符", asset.Name, fi.Size(), asset.Size)}
		}
	}

	// ---- 交棒前停一步：这部分是"升级前在跑的东西" ----
	if st := debug.RunningServe(env.dataDir); st != nil {
		plan.RestartServe = true
		plan.StopPIDs = append(plan.StopPIDs, st.PID)
		plan.RestartArgs = []string{"serve", "--config", env.cfgPath}
		if addr := strings.TrimPrefix(st.URL, "http://"); addr != "" {
			plan.RestartArgs = append(plan.RestartArgs, "--listen", addr)
		}
	}
	plan.StopPIDs = append(plan.StopPIDs, enginePIDs(env.dataDir)...)

	// 技能与二进制要同版：agent 目录里那一份不随升级自动变（见 update.SkillsStamp）。
	// 目标由前台算（戳文件在数据目录），更新器只负责拿新装的 exe 去刷。
	plan.SkillsTargets = update.StaleSkillsTargets(env.dataDir, res.Latest)

	if f.dryRun {
		fmt.Fprintf(errOut, "\n--dry-run：以下动作没有执行\n")
		fmt.Fprintf(errOut, "  形态: %s\n  目标: %s → %s\n  制品: %s\n", env.kind, env.current, res.Latest, dest)
		if plan.Stage != "" {
			fmt.Fprintf(errOut, "  载荷: %s（已解好并验过自报版本）\n", plan.Stage)
		}
		if len(plan.StopPIDs) > 0 {
			fmt.Fprintf(errOut, "  会先停: pid %v\n", plan.StopPIDs)
		}
		if plan.RestartServe {
			fmt.Fprintf(errOut, "  升完会把后台服务拉回来\n")
		}
		if len(plan.SkillsTargets) > 0 {
			fmt.Fprintf(errOut, "  升完会刷新 %d 处技能目录: %v\n", len(plan.SkillsTargets), plan.SkillsTargets)
		}
		if !f.yes {
			fmt.Fprintf(errOut, "  会问一句确认；非交互环境需要 --yes\n")
		}
		return nil
	}

	// ---- 确认 ----
	if !f.yes {
		ok, err := confirm(cmd, fmt.Sprintf("要把 tt 从 %s 升级到 %s 吗？", env.current, res.Latest))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(errOut, "已取消，什么都没改。")
			return nil
		}
	}

	// ---- 交棒 ----
	if err := update.SavePlan(env.dataDir, plan); err != nil {
		return applyErr(err)
	}
	st := &update.State{Phase: update.PhaseHandedOff, From: env.current, Target: res.Latest,
		Kind: env.kind.String(), StartedAt: time.Now(), LogPath: update.LogPath(env.dataDir)}
	if err := update.SaveState(env.dataDir, st); err != nil {
		return applyErr(err)
	}
	pid, err := update.Handoff(plan)
	if err != nil {
		return applyErr(err)
	}
	st.Error = ""
	_ = update.SaveState(env.dataDir, st)

	res2 := *rep
	res2.CheckedAt = time.Now()
	fmt.Fprintf(errOut, "\n更新器已接手（pid %d）。它会等本进程退出后替换文件，装完自报版本必须等于 %s。\n",
		pid, res.Latest)
	fmt.Fprintf(errOut, "看进度: tt update log    （日志 %s）\n", update.LogPath(env.dataDir))
	return emit(cmd, map[string]any{
		"kind": rep.Kind, "current": env.current, "target": res.Latest,
		"artifact": dest, "digest": hex, "helperPid": pid, "handedOff": true,
		"canInstall": true, "source": update.RepoSlug,
	})
}

// updateEnv 一次更新运行需要的外部条件。
type updateEnv struct {
	client  *update.Client
	cfgPath string
	dataDir string
	exe     string
	current string
	kind    update.Kind
	proxy   string // 生效的**显式**代理；空 = 按环境变量
}

// resolveUpdateEnv 解析配置、形态、版本与 HTTP 客户端。
//
// 配置允许缺失（第一次用还没配环境的人也该能查更新），配置坏掉时只失去 net.proxy，
// 不让整条命令失败 —— 出网失败会自己报出来。
func resolveUpdateEnv(proxyFlag string) (*updateEnv, error) {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return nil, err
	}
	cfgProxy := ""
	if root, err := config.Load(cfgPath); err == nil && root != nil {
		cfgProxy = root.Net.Proxy
	}
	proxy := update.ExplicitProxy(proxyFlag, cfgProxy)
	agent := "tt-update/" + Version
	if Version == "" {
		agent = "tt-update/devel"
	}
	client, err := update.NewClient(proxy, agent)
	if err != nil {
		return nil, usageErr(err.Error(), "代理地址要形如 http://127.0.0.1:10808")
	}
	exe, err := os.Executable()
	if err != nil {
		return nil, applyErr(fmt.Errorf("取不到当前可执行文件路径: %w", err))
	}
	return &updateEnv{
		client: client, cfgPath: cfgPath, dataDir: filepath.Dir(cfgPath),
		exe: exe, current: Version, kind: update.DetectKind(exe), proxy: proxy,
	}, nil
}

// stagePortable 把下载好的便携包解到**目标目录的兄弟位置**（同一卷，后面只做文件覆盖），
// 并跑一遍载荷里的 tt.exe 自报版本。
//
// 为什么先解到旁边而不是直接覆盖：解包/校验失败必须发生在动手之前 —— 覆盖到一半失败会
// 留下一个半新半旧的 tt。
func stagePortable(ctx context.Context, zipPath, exePath, target string, errOut io.Writer) (string, error) {
	dir := filepath.Dir(exePath)
	stage := filepath.Join(filepath.Dir(dir), fmt.Sprintf(".tt-update-stage-%d", time.Now().Unix()))
	if err := os.RemoveAll(stage); err != nil {
		return "", applyErr(err)
	}
	if err := os.MkdirAll(stage, 0o755); err != nil {
		return "", applyErr(fmt.Errorf("建暂存目录 %s 失败: %w", stage, err))
	}
	fmt.Fprintf(errOut, "解包 → %s\n", stage)
	if err := update.ExtractZip(zipPath, stage); err != nil {
		_ = os.RemoveAll(stage)
		return "", &output.Error{Code: output.CodeUpdateChecksum, Exit: 3,
			Message: err.Error(), Hint: "下载到的 zip 解不开，已丢弃，这次升级中止"}
	}
	root, err := update.PayloadRoot(stage)
	if err != nil {
		_ = os.RemoveAll(stage)
		return "", &output.Error{Code: output.CodeUpdateChecksum, Exit: 3, Message: err.Error()}
	}
	got, err := update.VersionOf(ctx, filepath.Join(root, "tt.exe"))
	if err != nil {
		_ = os.RemoveAll(stage)
		return "", &output.Error{Code: output.CodeUpdateChecksum, Exit: 3,
			Message: fmt.Sprintf("载荷里的 tt.exe 跑不起来: %v", err)}
	}
	fmt.Fprintf(errOut, "载荷自报版本: %s\n", got)
	if !update.OutputHasVersion(got, target) {
		_ = os.RemoveAll(stage)
		return "", &output.Error{Code: output.CodeUpdateChecksum, Exit: 3,
			Message: fmt.Sprintf("载荷里的 tt.exe 自报版本是 %q，与目标 %s 不一致（文件名不算数）", got, target),
			Hint:    "发布资产与 tag 对不上；这次升级中止，请反馈"}
	}
	return root, nil
}

// enginePIDs 收集各工作区正在跑的 .tzs 引擎守护进程。
//
// 必须停：引擎的 exe 就在安装目录里（<exe 目录>\tzs\tzs-server.exe），它锁着文件，
// 覆盖会失败。状态文件的形状归 internal/dev/tzs，这里只借它的读入口。
func enginePIDs(dataDir string) []int {
	st, err := tzs.LoadState(dataDir)
	if err != nil || st == nil {
		return nil
	}
	var pids []int
	for _, ws := range st.Keys() {
		e := st.Entry(ws)
		if e == nil {
			continue
		}
		if e.PID > 0 && winproc.Alive(e.PID) {
			pids = append(pids, e.PID)
		}
		for _, orphan := range e.Orphans {
			if orphan > 0 && winproc.Alive(orphan) {
				pids = append(pids, orphan)
			}
		}
	}
	return pids
}

// confirm 问一句。非交互环境（stdin 不是终端）**拒绝**而不是当作"是"：
// 升级会替换用户的二进制，猜错方向的代价不可逆。
func confirm(cmd *cobra.Command, question string) (bool, error) {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return false, usageErr("这是交互式确认，但当前 stdin 不是终端",
			"非交互环境（agent / CI）请显式给 --yes，或用 --dry-run 先看它打算做什么")
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N] ", question)
	var ans string
	if _, err := fmt.Fscanln(os.Stdin, &ans); err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes", nil
}

// emitCheck 打检查结论：JSON 进信封（默认），table 下人读几行。
func emitCheck(cmd *cobra.Command, rep *checkOnlyReport, env *updateEnv) error {
	if common.OutputFormat() != output.FormatTable {
		return emit(cmd, rep)
	}
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "当前版本: %s   形态: %s\n", rep.Current, rep.Kind)
	fmt.Fprintf(out, "更新源:   %s   %s\n", rep.Source, env.client.Where())
	if rep.Newer {
		fmt.Fprintf(out, "最新版本: %s（%s 发布）\n", rep.Latest, rep.PublishedAt.Local().Format("2006-01-02 15:04"))
	} else {
		fmt.Fprintf(out, "最新版本: %s（已是最新）\n", rep.Latest)
	}
	if !rep.CanInstall {
		fmt.Fprintf(out, "自装:     不允许 —— %s\n", rep.Refusal)
	}
	return nil
}

// emit 走唯一输出出口（JSON 信封 / CSV / 表格都由它决定）。
func emit(cmd *cobra.Command, data any) error {
	return output.Emit(cmd.OutOrStdout(), output.Options{
		Format: common.OutputFormat(), Meta: output.Meta{}, Data: data,
	})
}

// progressTo 下载进度：约每 10% 或每 10 秒打一行，别把终端刷爆。
//
// 两个条件都要有：“每 10%”是为了不会因为一条卡住的连接而长时间不说话，
// “每 10 秒”是为了在慢线路上不会因为 1% 走了很久而反复打同一行（真机跑 22 MB 的
// MSI 时，只有百分比条件会打出一串 69%）。
func progressTo(w io.Writer, name string) func(got, total int64) {
	var lastAt time.Time
	lastPct := -1
	return func(got, total int64) {
		now := time.Now()
		if total <= 0 {
			// 不知道总量（没有 Content-Length）就只按时间节流。
			if !lastAt.IsZero() && now.Sub(lastAt) < 10*time.Second {
				return
			}
			lastAt = now
			fmt.Fprintf(w, "  %s  %s\n", name, update.HumanBytes(got))
			return
		}
		pct := int(got * 100 / total)
		if lastPct >= 0 && got < total && pct/10 == lastPct/10 && now.Sub(lastAt) < 10*time.Second {
			return
		}
		lastAt, lastPct = now, pct
		fmt.Fprintf(w, "  %s  %s / %s (%d%%)\n", name, update.HumanBytes(got), update.HumanBytes(total), pct)
	}
}

// usageErr / applyErr / netErr 三个出口把这条线的退出码钉在一处：
// 用法错 2、出网失败 5、落地动作失败 5（"环境与 IO"那一档）。
//
// 这三者的错误码分得比退出码细：`UPDATE_NETWORK` 只给出网，覆盖不进去 / msiexec 失败
// 这类是 `UPDATE_APPLY` —— 两者退出码都是 5，但 agent 靠码分支时不能混。
func usageErr(msg, hint string) error {
	return &output.Error{Code: output.CodeUpdateUsage, Exit: 2, Message: msg, Hint: hint}
}

func applyErr(err error) error {
	return &output.Error{Code: output.CodeUpdateApply, Exit: 5, Message: err.Error()}
}
func netErr(err error) error {
	return &output.Error{Code: output.CodeUpdateNetwork, Exit: 5, Message: err.Error(),
		Hint: "若这台机器要经代理出网：tt update --proxy http://127.0.0.1:8080，或写进配置的 net.proxy"}
}

func tailLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}
