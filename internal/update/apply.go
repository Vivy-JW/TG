package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/winproc"
)

// Plan 一次升级的全部输入。它由前台命令写好、交给脱离的更新器进程读 —— 所以是**跨进程
// 契约**：字段名改动要同时想清楚 `tt update log` 与排障。
//
// 为什么非要把活儿交给另一个进程：Windows 上运行中的 exe 不能替换自己。前台进程能做的
// 最有用的一件事，就是把自己复制到临时目录、把这份 plan 交出去、然后退出。
type Plan struct {
	Kind     string `json:"kind"`     // KindPortable / KindMSI 的 String()
	From     string `json:"from"`     // 升级前版本
	Target   string `json:"target"`   // 目标版本
	Dir      string `json:"dir"`      // 便携包：要就地覆盖的目录
	ExePath  string `json:"exePath"`  // 替换完成后该在哪跑新版（校验用）
	Stage    string `json:"stage"`    // 便携包：已解好的载荷根（copyTree 的 src）
	Artifact string `json:"artifact"` // MSI：下载好的 .msi
	Digest   string `json:"digest"`   // 制品 sha256（留档，排障时能对回发布）
	DataDir  string `json:"dataDir"`
	WaitPID  int    `json:"waitPid"` // 前台进程；更新器等它退出再动手

	// StopPIDs 是升级前要停掉的进程（后台服务 + 各工作区的引擎守护进程）。
	// **由命令层算好**：状态文件的形状与"哪些进程会锁着安装目录"是命令层的知识
	// （`.tt-serve.json` 在 internal/cli/debug、`.tt-tzs.json` 在 internal/dev/tzs），
	// 这里只负责把它们杀掉 —— 不然这个包就得持有两份状态文件的第二份形状。
	StopPIDs []int `json:"stopPids,omitempty"`
	// RestartServe 为真时，更新器做完要把后台服务拉回来（升级前它在跑）。
	// RestartArgs 由命令层算好：更新器不认识 serve 的参数面，也不该认识。
	RestartServe bool     `json:"restartServe,omitempty"`
	RestartArgs  []string `json:"restartArgs,omitempty"`
	// BackupExe 便携包：旧 tt.exe 的备份位置（失败时改回来）。
	BackupExe string `json:"backupExe,omitempty"`

	// SkillsTargets 是要刷新的技能目录（agent 会去读的那些）。
	// 由命令层从`.tt-skills.json`算好：装了哪些处只有那份戳知道。
	SkillsTargets []string `json:"skillsTargets,omitempty"`
}

// PlanPath 计划文件落点（缓存目录内；它不是长期状态，是这一次交接的信物）。
func PlanPath(dataDir string) string {
	d := UpdateDir(dataDir)
	if d == "" {
		return ""
	}
	return filepath.Join(d, "plan.json")
}

// SavePlan 落盘（原子写）。
func SavePlan(dataDir string, p *Plan) error {
	path := PlanPath(dataDir)
	if path == "" {
		return errors.New("数据目录不可用，升级计划无处可落")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(path, append(b, '\n'))
}

// LoadPlan 读计划。
func LoadPlan(path string) (*Plan, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读不到升级计划 %s: %w", path, err)
	}
	var p Plan
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("升级计划 %s 解析失败: %w", path, err)
	}
	if p.Target == "" || p.ExePath == "" {
		return nil, fmt.Errorf("升级计划 %s 不完整", path)
	}
	return &p, nil
}

// Handoff 把活儿交给一个脱离的更新器进程，返回它的 pid。
//
// 关键一步是把**自己复制到临时目录再起**：更新器要替换的正是当前这个 tt.exe，原地起
// 子进程会由父进程自己锁住目标文件，替换必然失败。复制到 %TEMP% 之后，更新器与安装
// 目录再无关系，那个目录可以随便被动。
func Handoff(p *Plan) (int, error) {
	self := p.ExePath
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("tt-update-%d", os.Getpid()))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("建更新器目录 %s 失败: %w", dir, err)
	}
	helper := filepath.Join(dir, "tt.exe")
	if err := copyFile(self, helper); err != nil {
		return 0, fmt.Errorf("复制更新器到 %s 失败: %w", helper, err)
	}

	args := []string{"update", "apply", "--plan", PlanPath(p.DataDir)}
	if p.DataDir != "" {
		// 显式带上配置路径：更新器是另一个进程，不该再依赖环境变量与缺省落点推断。
		args = append(args, "--config", filepath.Join(p.DataDir, "config.json"))
	}
	pid, err := winproc.SpawnDetached(helper, args, LogPath(p.DataDir))
	if err != nil {
		return 0, fmt.Errorf("拉起更新器失败: %w", err)
	}
	return pid, nil
}

// Apply 是更新器进程的主体：等前台退出 → 停后台服务 → 替换 → 校验 → （必要时）把服务
// 拉回来 → 落状态。
//
// 每一步都往日志写一行：更新器是脱离了终端跑的东西，日志是唯一能回答"它到底做了什么"
// 的地方。
func Apply(ctx context.Context, p *Plan, log func(string, ...any)) error {
	st := &State{Phase: PhaseApplying, From: p.From, Target: p.Target, Kind: p.Kind, StartedAt: time.Now()}
	if p.Kind == TokenMSI {
		st.Artifact = p.Artifact
	} else {
		st.Artifact = p.Stage
	}
	st.LogPath = LogPath(p.DataDir)
	_ = SaveState(p.DataDir, st)

	fail := func(err error) error {
		st.Phase, st.Error, st.FinishedAt = PhaseFailed, err.Error(), time.Now()
		_ = SaveState(p.DataDir, st)
		log("失败: %v", err)
		RemoveStage(p.Stage)
		return err
	}

	if p.WaitPID > 0 {
		log("等前台进程 %d 退出（它正占着要替换的文件）", p.WaitPID)
		if err := waitProcessGone(ctx, p.WaitPID, 60*time.Second); err != nil {
			return fail(err)
		}
	}

	if len(p.StopPIDs) > 0 {
		stopProcesses(p.StopPIDs, log)
	}

	var applyErr error
	switch p.Kind {
	case TokenPortable:
		applyErr = applyPortable(p, log)
	case TokenMSI:
		applyErr = applyMSI(ctx, p, log)
	default:
		applyErr = fmt.Errorf("升级计划里的安装形态 %q 认不出", p.Kind)
	}
	if applyErr != nil {
		return fail(applyErr)
	}

	// 校验：**用刚装好的那份二进制自报版本**。这是整条链上唯一能证明"装下去的是它"
	// 的动作 —— 摘要只能证明"下到的字节是发布方发的"。
	got, err := VersionOf(ctx, p.ExePath)
	if err != nil {
		return fail(fmt.Errorf("装完后跑 %s version 失败: %w", p.ExePath, err))
	}
	log("装完的版本: %s（目标 %s）", got, p.Target)
	if !OutputHasVersion(got, p.Target) {
		return fail(fmt.Errorf("装完后 %s 自报版本是 %q，与目标 %s 不一致", p.ExePath, got, p.Target))
	}

	if p.RestartServe {
		restartServe(p, log)
	}

	// 技能刷新放在最后：它动的是 agent 会读的目录，失败也不该把"二进制已升级"这件事
	// 变成失败 —— 记成警告，`tt update log` 里看得见。
	st.Warnings = append(st.Warnings, refreshSkills(p, log)...)

	if p.Stage != "" {
		RemoveStage(p.Stage)
		log("已清掉暂存目录 %s", p.Stage)
	}

	st.Phase, st.FinishedAt, st.Error = PhaseDone, time.Now(), ""
	_ = SaveState(p.DataDir, st)
	log("完成: %s → %s", p.From, p.Target)
	return nil
}

// applyPortable 就地覆盖便携包目录。
//
// 顺序：先备份旧 tt.exe（失败时能改回来）→ 覆盖 → 覆盖失败则把旧 exe 放回。
// 备份只留这一个文件：它是"坏了就完全不能用"的那个；skills/ 与引擎 dll 可以从下载好的
// 载荷里再覆盖一次。
func applyPortable(p *Plan, log func(string, ...any)) error {
	if p.Stage == "" || p.Dir == "" {
		return errors.New("便携包升级计划缺少目录或载荷")
	}
	if err := os.MkdirAll(p.BackupDir(), 0o755); err != nil {
		return fmt.Errorf("建备份目录失败: %w", err)
	}
	old := filepath.Join(p.Dir, "tt.exe")
	backup := filepath.Join(p.BackupDir(), "tt.exe")
	if _, err := os.Stat(old); err == nil {
		if err := copyFile(old, backup); err != nil {
			return fmt.Errorf("备份旧 tt.exe 失败（没有它就没法回退）: %w", err)
		}
		log("已备份旧 tt.exe → %s", backup)
	}
	n, err := copyTree(p.Stage, p.Dir)
	if err != nil {
		log("覆盖目录失败，回退旧 tt.exe")
		if _, serr := os.Stat(backup); serr == nil {
			_ = copyFile(backup, old)
		}
		return fmt.Errorf("覆盖 %s 失败: %w", p.Dir, err)
	}
	log("已就位：%d 个文件覆盖到 %s（包内没有的文件保持原样，config.json 在内）", n, p.Dir)
	return nil
}

// BackupDir 旧 tt.exe 的备份目录（便携包目录下的隐藏子目录，跟着包走）。
func (p *Plan) BackupDir() string { return filepath.Join(p.Dir, ".tt-update-prev") }

// applyMSI 把新的 MSI 交给 msiexec。
//
// per-user 安装（见 installer/tt.wxs）所以不需要管理员权限，也不该弹 UAC；
// `/qb` 给最简进度界面而不是 `/qn` —— 升级是几分钟的事，全静默时用户看不到任何东西，
// 只能等。退出码 0 成功 / 3010 成功但需重启 / 1641 成功且已安排重启。
func applyMSI(ctx context.Context, p *Plan, log func(string, ...any)) error {
	if runtime.GOOS != "windows" {
		return errors.New("MSI 安装形态只在 Windows 上可用")
	}
	if p.Artifact == "" {
		return errors.New("MSI 升级计划缺少安装包路径")
	}
	msiexec := filepath.Join(os.Getenv("SystemRoot"), "System32", "msiexec.exe")
	if _, err := os.Stat(msiexec); err != nil {
		msiexec = "msiexec.exe" // 交给 PATH 解析，报错时至少信息是清楚的
	}
	log("运行 %s /i %s /qb /norestart", msiexec, p.Artifact)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, msiexec, "/i", p.Artifact, "/qb", "/norestart")
	out, err := cmd.CombinedOutput()
	code := cmd.ProcessState.ExitCode()
	if len(out) > 0 {
		log("msiexec 输出: %s", strings.TrimSpace(string(out)))
	}
	if err != nil && code != 3010 && code != 1641 {
		return fmt.Errorf("msiexec 失败（退出码 %d）: %w", code, err)
	}
	log("msiexec 退出码 %d", code)
	return nil
}

// VersionOf 跑一次 `<exe> version` 拿回输出。
//
// 跑刚下到的二进制是**有意为之**：摘要证明"字节是发布方发的"，而它自报的版本证明
// "这确实是那一次发布"，两者互补。它只跑 `version`，不碰配置、不联网。
func VersionOf(ctx context.Context, exe string) (string, error) {
	if _, err := os.Stat(exe); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, exe, "version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w（输出：%s）", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// OutputHasVersion 判断 `tt version` 的输出里有没有目标版本号。
// 输出形如 `tt 0.2.1 (commit abc1234, 2026-10-07)`，所以按**词**比而不是子串比：
// 子串会让 0.2.2 匹配上 0.2.20。两边的 v 前缀都抹掉 —— tag 带它、注入的版本号不带。
func OutputHasVersion(out, want string) bool {
	want = strings.TrimPrefix(strings.TrimSpace(want), "v")
	for _, f := range strings.Fields(out) {
		f = strings.Trim(f, "(),")
		if strings.TrimPrefix(f, "v") == want {
			return true
		}
	}
	return false
}

// waitProcessGone 等一个进程退出（最多 timeout）。进程不在了或被复用都不是问题：
// 我们等的是"目标文件不再被它占用"这个效果，退出后由调用方自己写来决定成败。
func waitProcessGone(ctx context.Context, pid int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !winproc.Alive(pid) {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("前台进程 %d 在 %s 内没有退出：不替换了，避免留下半截状态", pid, timeout)
}

// stopProcesses 停掉升级前会锁着安装目录的那些进程（后台服务与引擎守护进程）。
// 停不下来不阻断升级：真正的判据是后面替换能不能成功，而替换失败会自己报出来。
func stopProcesses(pids []int, log func(string, ...any)) {
	for _, pid := range pids {
		if pid <= 0 {
			continue
		}
		if err := winproc.Kill(pid); err != nil {
			log("停 pid %d 失败: %v（继续尝试替换）", pid, err)
			continue
		}
		log("已停 pid %d", pid)
		for i := 0; i < 50 && winproc.Alive(pid); i++ {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// refreshSkills 用**刚装好的** exe 刷新 agent 目录里的技能树，返回警告（不是错误）。
//
// 为什么要另起一个进程而不是直接调函数：安装技能的实现住在命令层（internal/cli），
// 而那一层 import 本包 —— 反过来 import 就成环。走子进程还有个好处：用的是**新版**的
// 安装逻辑与新版自带的那棵技能树，而不是正要被替换掉的旧版那棵。
func refreshSkills(p *Plan, log func(string, ...any)) []string {
	var warnings []string
	for _, dir := range p.SkillsTargets {
		if dir == "" {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		cmd := exec.CommandContext(ctx, p.ExePath, "install", "skills", "--force", "--to", dir)
		out, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			w := fmt.Sprintf("刷新技能目录 %s 失败: %v（输出：%s）—— 手动跑 tt install skills --force --to %s",
				dir, err, strings.TrimSpace(string(out)), dir)
			log("警告: %s", w)
			warnings = append(warnings, w)
			continue
		}
		log("已把技能刷到同版: %s", dir)
	}
	return warnings
}

// restartServe 用升级前记下的参数把后台服务拉回来。
//
// 日志落到**服务自己的那个日志**（`.tt-serve.log`）而不是更新日志：`tt serve` 自己
// 起的那些进程都把输出写在那里，混到更新日志里会让“服务为什么没起来”要在两个文件里找。
func restartServe(p *Plan, log func(string, ...any)) {
	args := p.RestartArgs
	if len(args) == 0 {
		args = []string{"serve"}
	}
	serveLog := filepath.Join(p.DataDir, config.ServeLogFileName)
	pid, err := winproc.SpawnDetached(p.ExePath, args, serveLog)
	if err != nil {
		log("拉回后台服务失败: %v（用 tt serve 手动起）", err)
		return
	}
	log("已拉回后台服务 pid %d（输出在 %s）", pid, serveLog)
}

// CleanStaleStages 清掉失败/中断的升级留在安装目录旁边的暂存目录。
//
// 暂存目录是**目标目录的兄弟**（同一卷，覆盖时才便宜），它不是数据、删了只是下次重解，
// 所以只保留一天内新建的那些（可能有一次升级正在用它）。
func CleanStaleStages(exePath string, now time.Time) {
	if exePath == "" {
		return
	}
	pattern := filepath.Join(filepath.Dir(filepath.Dir(exePath)), ".tt-update-stage-*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || !fi.IsDir() || now.Sub(fi.ModTime()) < 24*time.Hour {
			continue
		}
		_ = os.RemoveAll(m)
	}
}

// RemoveStage 删掉这次升级用的暂存目录（成功与失败都删）。
// 下载到的 zip 还在缓存目录里，重跑一次升级会重新解出来，所以这里不必留。
func RemoveStage(stage string) {
	if stage == "" {
		return
	}
	_ = os.RemoveAll(stage)
}

// CleanStaleHelpers 清掉上一次升级留下的临时目录。
//
// 为什么要专门做：更新器就是那个目录里的 exe，它不能删自己（Windows 上运行中的文件删
// 不掉），所以留给**下一次**启动来清。只清一天前的，免得把正在跑的更新器删了。
func CleanStaleHelpers(now time.Time) {
	matches, err := filepath.Glob(filepath.Join(os.TempDir(), "tt-update-*"))
	if err != nil {
		return
	}
	for _, m := range matches {
		fi, err := os.Stat(m)
		if err != nil || !fi.IsDir() {
			continue
		}
		if now.Sub(fi.ModTime()) < 24*time.Hour {
			continue
		}
		_ = os.RemoveAll(m)
	}
}
