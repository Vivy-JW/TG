package tzs

// server.go —— 守护进程的生命周期。
//
// 形状是照 internal/cli/debug/servebg.go 抄的（spawn → 等就绪 → 记状态 → stop），
// 但**三条前提与我们相反**，所以不能复用那一份：
//
//	servebg 的实例用 HTTP 探活        我们探的是一条命名管道，且要问引擎才拿得到名字
//	servebg 的状态文件全局一条         我们是按工作区多条（一个工作区一个守护进程）
//	servebg 的失败是「没起来」         我们的失败还分「有致命帧的确定性失败」与「无帧的 EOF」
//
// 相同的只有「起子进程 / 判活 / 杀」这三件纯进程原语 —— 那部分已经抽在 internal/winproc，
// 两边都用它。
//
// 就绪握手（契约给的三条数，全部来自实测）：
//
//	每 250 ms 试连一次，单次 connect 超时 300 ms；
//	**连得上就算就绪**（不发任何帧 —— 理由见 probeReady）；
//	预算 60 s（Boot 约 900 ms + 首次加载，剩下的余量是给慢磁盘的 ——
//	真正卡死的加载由引擎内部 90 s 的看门狗负责，不是这个预算）。
//
// 探针不碰任何引擎函数：管道是 Boot 之后才创建的、实例只有一个，所以"连上"本身就证明了
// "Boot 做完且服务端正等在这里"。这条边界很重要 —— Go 侧不该在传输层写死一个业务函数名。

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tt/internal/config"
	"tt/internal/winproc"
)

// 就绪握手与轮询的常数（理由见文件注释）。
const (
	// ReadinessBudget 是冷启动预算。
	ReadinessBudget = 60 * time.Second
	// ConnectPollGap 是就绪轮询的间隔。
	ConnectPollGap = 250 * time.Millisecond
	// StopTimeout 是发 `stop` 之后的等待上限。
	StopTimeout = 10 * time.Second
)

// Options 是守护进程的四个落点。
type Options struct {
	// Exe 是 tzs-server.exe 的路径。
	Exe string
	// InstallDir 是**开发期覆盖**的设计器目录，作为 TZSCLI_INSTALL 传给子进程。
	// 留空是正常情形：引擎默认用它自己旁边的 `designer\`（随包分发的那份）。
	// 见 engine/src/Designer/Bootstrap.cs 的 Designer.Install。
	InstallDir string
	// Workspace 是 .tzs 工作区。**绝不留空**：引擎的缺省是一个真实客户目录
	// （Rpc.DefaultWorkspace），留空就等于拿客户的表单当草稿纸。解析顺序由命令层定：
	// --workspace flag > TZSCLI_WS 环境变量 > **当前环境的 workspace**
	//（hosts.sshs[].workspace，见 config.Root.ActiveWorkspace），末端**拒绝**。
	Workspace string
	// WorkspaceSrc 工作区是**从哪来的**（"--workspace" / "环境变量 TZSCLI_WS" /
	// "环境「X」"，只给 doctor 的报告用 —— 三个来源都可能生效时，"这个路径是谁给的"
	// 是排查的第一步：查的是别的客户的工作区时，报错信息本身看不出任何异常。
	WorkspaceSrc string
	// WorkDir 是状态文件与守护进程日志的落点，一般传 config.json 所在目录
	// （与 .tt-serve.json 同目录）。留空则退回 config 的默认落点。
	WorkDir string
}

// DaemonInfo 是「这个工作区现在有没有守护进程」的一份快照（给 status / doctor / stop 文案用）。
type DaemonInfo struct {
	Workspace string
	Pipe      string // 现在生效的管道名（问引擎得到的）
	Recorded  string // 状态文件里记的管道名（可能为空或已过期）
	PID       int    // 状态文件里记的 pid（0 = 不知道）
	Running   bool   // 管道真的连得上（= 守护进程在跑；见 probeReady）
	Log       string
	Since     string
}

//---------------------------------------------------------------------------
// Ensure
//---------------------------------------------------------------------------

// Ensure 保证某个工作区有一个就绪的守护进程：已有就复用，没有就 spawn 并等就绪。
// 返回本构建该工作区的管道名（调用方拿它去 Call）。
//
// **它不替调用方选工作区**：Workspace 解析不出来时直接拒绝（退出 2），绝不 spawn。
func Ensure(ctx context.Context, o Options) (string, error) {
	ws, err := o.workspace()
	if err != nil {
		return "", err
	}
	pipe, err := PipeName(ctx, o.Exe, ws)
	if err != nil {
		return "", err
	}
	dir := o.StateDir()
	st := loadState(dir)
	// 先记一份「我见过这个管道」——即使后面就绪失败，记录也得留着：
	// 一个 wedged 的守护进程正是 Reap 要能看见的东西。
	prev := st.Entry(ws)

	if probeReady(ctx, pipe) {
		e := &DaemonEntry{Pipe: pipe, Workspace: ws, Exe: o.Exe, Log: o.DaemonLogPath()}
		if prev != nil {
			e.PID, e.Since, e.Orphans = prev.PID, prev.Since, prev.Orphans
		}
		if e.Since == "" {
			e.Since = time.Now().Format(time.RFC3339)
		}
		st.SetEntry(ws, e)
		_ = SaveState(dir, st)
		return pipe, nil
	}

	// 冷启动。子进程必须显式带 --daemon：不传且它的 stdin 被重定向时，引擎会自己
	// 判定成 stdio 模式（tzs-server.cs 的 Console.IsInputRedirected），管道永远不会出现。
	pid, err := winproc.SpawnDetached(o.Exe, o.spawnArgs(ws), o.DaemonLogPath(), o.extraEnv(ws)...)
	if err != nil {
		return "", &TransportError{Code: CodeServerDied,
			Msg: "启动守护进程失败（" + o.Exe + "）", Err: err}
	}
	e := &DaemonEntry{
		PID: pid, Pipe: pipe, Workspace: ws, Exe: o.Exe, Log: o.DaemonLogPath(),
		Since: time.Now().Format(time.RFC3339),
	}
	// 旧记录的管道名与新名字不同 = 那是**上一次构建**的守护进程。它既不会被我们连上
	// （设计如此：不允许连到跑陈旧字节的守护进程），也不会自己退出。
	// 不在这儿杀它（可能有别的客户端正在请求它），只把 pid 留给 Reap。
	if prev != nil && prev.PID > 0 && prev.PID != pid && prev.Pipe != pipe && winproc.Alive(prev.PID) {
		e.Orphans = append(append([]int{}, prev.Orphans...), prev.PID)
	} else if prev != nil {
		e.Orphans = prev.Orphans
	}
	st.SetEntry(ws, e)
	_ = SaveState(dir, st)

	deadline := time.Now().Add(ReadinessBudget)
	for time.Now().Before(deadline) {
		if probeReady(ctx, pipe) {
			return pipe, nil
		}
		if !winproc.Alive(pid) {
			// 提前失败：进程都没了，再等满 60 s 只是把结论推迟。
			st.DropEntry(ws)
			_ = SaveState(dir, st)
			return "", &TransportError{Code: CodeServerDied,
				Msg: fmt.Sprintf("守护进程提前退出（pid %d）", pid) + "；日志末尾：\n" + logTail(o.DaemonLogPath(), 2000)}
		}
		if !sleepCtx(ctx, ConnectPollGap) {
			return "", ctxFail(ctx)
		}
	}
	// 超时。pid 还活着就**保留**记录（它是个 wedged 的守护进程，Reap 应该看得见）；
	// 已经死了就删掉，免得下一条命令又对着一条死记录发呆。
	if !winproc.Alive(pid) {
		st.DropEntry(ws)
		_ = SaveState(dir, st)
	}
	return "", &TransportError{Code: CodeServerDied,
		Msg: fmt.Sprintf("守护进程 %s 内没有就绪（pid %d）", ReadinessBudget, pid) + "；日志末尾：\n" + logTail(o.DaemonLogPath(), 2000)}
}

// probeReady 探一次就绪：**连得上就算就绪**。
//
// 为什么不发一帧：管道实例只有一个（引擎 `maxNumberOfServerInstances=1`），而服务端的循环是
// `WaitForConnection → Serve → Disconnect`；我们连得上，就说明服务端此刻正等在这个实例上
// （即 Boot 已经做完 —— `tzs-server` 是**先 `Designer.Boot` 再 `Rpc.Daemon`**，
// 管道在 Boot 之后才创建，见 engine/test/tzs-server.cs）。
//
// 以前这里发 `list_open` 探活：为了问一句"你醒着吗"，去调一个**业务函数**（列打开的句柄）。
// 那让 Go 侧把一个引擎函数名写死在传输层里 —— 业务面与传输面混在一起，引擎改名就得改这里。
// 连上即就绪没有这个问题，而且更便宜。
func probeReady(ctx context.Context, pipe string) bool {
	dctx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	defer cancel()
	conn, err := DefaultDialer().Dial(dctx, pipe)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

//---------------------------------------------------------------------------
// Stop
//---------------------------------------------------------------------------

// Stop 停掉某个工作区的守护进程。**绝不 spawn**（让「停止」去启动服务是个笑话），
// 且幂等：本来就没有守护进程时返回 nil。
//
// 返回 nil 有两种含义：「停掉了」和「本来就没有」。要区分就先调 LookupDaemon
// 取一份快照再决定文案 —— 但**不要**用返回错误来表达「本来就没有」：
// 那会让命令层退出 5 或 1，而 stop 是幂等的退 0。
func Stop(ctx context.Context, o Options) error {
	ws, err := o.workspace()
	if err != nil {
		return err
	}
	dir := o.StateDir()
	st := loadState(dir)

	// 管道名优先问引擎（状态文件里的可能已经过期 —— 重编引擎会让名字变）。
	// 问不到时退回记录里的那个：连不上就按「没有守护进程」处理，一样是幂等退 0。
	pipe := ""
	if rec := st.Entry(ws); rec != nil {
		pipe = rec.Pipe
	}
	if p, perr := PipeName(ctx, o.Exe, ws); perr == nil {
		pipe = p
	} else if pipe == "" {
		return perr
	}

	dctx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	conn, derr := DefaultDialer().Dial(dctx, pipe)
	cancel()
	if derr == nil {
		func() {
			defer conn.Close()
			// 超时/中断都当「已经停止了」：引擎收到 stop 会写好应答再退出，
			// 客户端没等到应答不代表它没收到。这里不把结果当判据。
			_, _ = CallConn(ctx, conn, 1, FnStop, nil, StopTimeout)
		}()
	}
	// 无论连没连上，这条记录都作废了：连不上说明进程已经没了（或换了管道名），
	// 留着它会让下一次 Reap 对着一个死 pid 发呆。
	st.DropEntry(ws)
	_ = SaveState(dir, st)
	return nil
}

//---------------------------------------------------------------------------
// Reap
//---------------------------------------------------------------------------

// Reap 收拾**孤儿守护进程**：跑着旧构建字节、谁也连不上的那些。
//
// 它们是怎么来的：管道名含 TzsCli.Designer.dll 的 MVID，每次重编引擎名字就变，
// 上一次构建起的守护进程于是永远等不到客户端（设计如此：不允许连到跑陈旧字节的
// 守护进程），也不自己退出。
//
// 判据只有一条：**状态文件里记的管道名 ≠ 这个工作区现在该有的管道名**。
// 不能用「连不上」当判据 —— 孤儿在它自己的管道名上答得好好的，连不上的是我们。
//
// yes=false 只报告（dry-run），yes=true 才真杀。返回涉及到的 pid 列表（排序稳定）。
//
// 问不到当前管道名时（引擎 exe 没了/坏了）**一个都不动**：那会儿我们没法把孤儿和
// 现役区分开，猜错就是杀掉一个正在被别的客户端用的守护进程。
func Reap(ctx context.Context, o Options, yes bool) ([]int, error) {
	dir := o.StateDir()
	st := loadState(dir)
	var pids []int

	for _, k := range st.Keys() {
		e := st.Daemons[k]
		if e == nil {
			continue
		}
		// pid 记着、人已经没了、也没有待收的孤儿 → 这条账已经平了，删掉。
		// 注意只删「pid 明确死了」的：pid<=0 的记录我们连它死没死都不知道
		// （比如守护进程是别的进程起的，我们只记到了管道名），删掉就等于把一个
		// 活着的东西忘掉 —— 而忘掉它正是下次 Reap 收不到它的原因。
		if e.PID > 0 && !winproc.Alive(e.PID) && len(e.Orphans) == 0 {
			delete(st.Daemons, k)
			continue
		}

		// 「这个工作区现在该有的管道名」。问不到（exe 没了/坏了）就整条跳过：
		// 那会儿孤儿和现役分不开，猜错的代价是杀掉一个正被别的客户端用的守护进程。
		cur := ""
		if e.Workspace != "" {
			if p, err := PipeName(ctx, o.Exe, e.Workspace); err == nil {
				cur = p
			}
		}
		if cur == "" {
			continue
		}

		// ① 记录本身指向一个旧管道名，而进程还活着 = 跑陈旧字节的孤儿。
		if e.PID > 0 && winproc.Alive(e.PID) && e.Pipe != cur {
			pids = append(pids, e.PID)
			if yes {
				_ = winproc.Kill(e.PID)
				e.PID = 0
				e.Pipe = ""
			}
		}

		// ② 被后来者挤掉的那些（spawn 时挪进 Orphans 的 pid），判据同上。
		var still []int
		for _, p := range e.Orphans {
			if !winproc.Alive(p) {
				continue
			}
			pids = append(pids, p)
			if yes {
				_ = winproc.Kill(p)
				continue
			}
			still = append(still, p)
		}
		if yes {
			e.Orphans = nil
			if e.PID == 0 {
				// 自己也被收了、且没有别的线索 → 整条删掉。
				delete(st.Daemons, k)
			}
		} else {
			e.Orphans = still
		}
	}

	if yes {
		// dry-run 绝不落盘：Reap(ctx, o, false) 必须是纯粹的「看一眼」。
		_ = SaveState(dir, st)
	}
	return pids, nil
}

//---------------------------------------------------------------------------
// LookupDaemon
//---------------------------------------------------------------------------

// LookupDaemon 报告某个工作区当前的守护进程状态（不 spawn，也不改状态文件）。
//
// 它存在的理由是文案：Stop 为了幂等必须对「停掉了」和「本来就没有」都返回 nil，
// 命令层要说出正确的那一句就得先看一眼。doctor 也用它。
func LookupDaemon(ctx context.Context, o Options) (*DaemonInfo, error) {
	ws, err := o.workspace()
	if err != nil {
		return nil, err
	}
	info := &DaemonInfo{Workspace: ws}
	if rec := loadState(o.StateDir()).Entry(ws); rec != nil {
		info.Recorded, info.PID, info.Log, info.Since = rec.Pipe, rec.PID, rec.Log, rec.Since
	}
	if pipe, perr := PipeName(ctx, o.Exe, ws); perr == nil {
		info.Pipe = pipe
	} else {
		// 问不到就用记录里的：连得上就说明它在跑，这比报错有用。
		info.Pipe = info.Recorded
	}
	if info.Pipe != "" {
		info.Running = probeReady(ctx, info.Pipe)
	}
	return info, nil
}

//---------------------------------------------------------------------------
// Options 的派生
//---------------------------------------------------------------------------

// workspace 解析出生效的工作区，解析不出来就拒绝（退出 2）。
//
// **绝不留缺省**：引擎的缺省工作区是一个真实客户目录（Rpc.DefaultWorkspace），
// 留空就等于默默对着客户的表单干活。
//
// 退到 TZSCLI_WS 是刻意的：引擎自己也认这个变量（Rpc.PipeName 的解析顺序里有它），
// 所以认它只会让两侧一致。命令层应当已经把 --workspace / 配置都解析好传进来，
// 这一步是兜底，不是替代。
func (o Options) workspace() (string, error) {
	if ws := normalizeWS(o.Workspace); ws != "" {
		return ws, nil
	}
	if ws := normalizeWS(os.Getenv("TZSCLI_WS")); ws != "" {
		return ws, nil
	}
	return "", &UsageError{
		Msg: "未配置工作区（拒绝启动引擎）",
		Detail: []string{
			"引擎的默认工作区是一个真实客户目录，工具绝不替你选",
			"三种给法：--workspace <dir> / 环境变量 TZSCLI_WS / --env <环境名> 的工作区",
			"（环境那一份在设置页「站点管理 → 环境 → 工作区目录」，即 hosts.sshs[].workspace）",
		},
	}
}

// normalizeWS 把工作区路径里的 `/` 换成 `\`。
//
// 为什么这一个替换是必要的：这个字符串同时喂给两处，而**用 `/` 写的那一种，
// 两处都过不去**（实测，不是推理）——
//
//  1. **设计器**的 `TzpManager.InCurrentWorkspace` 是纯**字符串前缀比较**（取包所在的
//     目录、两边各补一个尾 `\`）。工作区写成 `C:/ws` 时，每一次 `open` 都被拒
//     （`NotInCurrentWorkspaceException: … 不在工作目录之下`），而 `doctor` 对同一个值
//     说「工作区 ok」（它只 `stat` 目录）—— 症状与原因离得很远。
//  2. **管道名** = hash(工作区字符串) 前 8 位。同一个目录两种拼法 = **两个守护进程身份**，
//     于是"只把配置里的斜杠方向改一下"就孤儿化一个正在跑的 daemon：实测状态文件里是
//     `tzs-cli-3637cf53-…`，改斜杠后 doctor 算出 `tzs-cli-29fc047f-…`，`stop` 与 `reap`
//     都够不到它，最后只能手工 taskkill。
//
// 两个身份的合并之所以只会更好：`/` 那一种**根本装不进任何包**，所以它不是"另一种用法"，
// 是一个坏掉的值。
//
// **不做的事**（有意的，别顺手加）：不 `filepath.Clean`、不去尾分隔符、不改大小写。
// `TestWorkspaceFromEnvAndTrim` 的注释立过这条规矩：尾斜杠会改变 hash，那是另一个身份。
// 实测过（2026-09-24）：**尾分隔符形态是被设计器接受的**（工作区 `C:\ws\` + 包
// `C:\ws\x.tzs` → `ok:true`），所以它不是陷阱，不去它只是保留一个身份差异 ——
// 代价是"只改尾斜杠"同样会孤儿化一个 daemon，与斜杠方向那条不同，这一条**留着没修**。
//
// 顺带把机制说准（两处行为不一样，别推错）：设计器**会归一化包路径**、**不会**归一化
// 工作区字符串 —— 工作区 `C:\ws` + 包 `C:/ws/x.tzs` 通过；工作区 `C:/ws` + 包任意形式
// 被拒。所以 `--args` 里的包路径写正斜杠是安全的（三个干净执行者都这么绕过转义问题），
// 而**工作区**那一侧必须由我们统一。
//
// 大小写不必管：引擎 hash 之前先 `ToLowerInvariant()`，而前缀比较也是大小写不敏感的。
//
// Windows 的文件名里不允许出现 `/`，所以这个替换不会误伤；UNC（`\\server\share`）也不受影响。
func normalizeWS(p string) string {
	return strings.ReplaceAll(strings.TrimSpace(p), "/", `\`)
}

// StateDir 是状态文件与日志的落点（导出给命令层：它要写 last、要渲染路径）。
//
// 命令层应当把解析好的配置目录经 WorkDir 传进来（只有它知道 --config）；
// 留空时退回 config 的默认落点，再退到用户目录。
func (o Options) StateDir() string {
	if d := strings.TrimSpace(o.WorkDir); d != "" {
		return d
	}
	if p, err := config.ResolvePath("", true); err == nil && p != "" {
		return filepath.Dir(p)
	}
	if d := config.UserConfigDir(); d != "" {
		return d
	}
	return "."
}

// DaemonLogPath 是守护进程日志的落点（导出给命令层渲染）。
func (o Options) DaemonLogPath() string { return LogPath(o.StateDir()) }

// spawnArgs 是拉起守护进程的参数。
//
// `--daemon` **必须显式给**：不传且子进程的 stdin 被重定向时（winproc 把 stdin 接成
// 空设备，就是重定向），引擎会自己选 stdio 模式，管道永远不会出现 ——
// 现象是每次调用都「冷启动 60 s 未就绪」，而日志里只多一行 "模式: stdio"。
//
// `--workspace` 也显式给：不依赖环境变量，命令行的值在引擎的参数解析里优先级更高。
func (o Options) spawnArgs(ws string) []string {
	return []string{"--daemon", "--workspace", ws}
}

// extraEnv 是给子进程补的环境变量。
//
// TZSCLI_WS 与 --workspace 一起给是冗余但故意的：它保证任何一条**再派生**的路径
// （引擎内部 spawn、将来的子工具）也落在同一个工作区上。
// TZSCLI_INSTALL 只在给了 InstallDir 时才覆盖：留空表示「用引擎旁边随包分发的那份」，
// 所以要原样不传，而不是传一个空串去把它顶掉。
func (o Options) extraEnv(ws string) []string {
	env := []string{"TZSCLI_WS=" + ws}
	if d := strings.TrimSpace(o.InstallDir); d != "" {
		env = append(env, "TZSCLI_INSTALL="+d)
	}
	return env
}

//---------------------------------------------------------------------------
// 小工具
//---------------------------------------------------------------------------

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func ctxFail(ctx context.Context) error {
	return &TransportError{Code: CodeServerDied, Msg: "等待守护进程就绪时被取消", Err: ctx.Err()}
}

// logTail 取日志末尾（启动失败时给出线索）。
func logTail(path string, max int) string {
	f, err := os.Open(path)
	if err != nil {
		return "(无日志文件 " + path + ")"
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return "(无法读取日志)"
	}
	off := int64(0)
	if st.Size() > int64(max) {
		off = st.Size() - int64(max)
	}
	buf := make([]byte, st.Size()-off)
	if _, err := f.ReadAt(buf, off); err != nil && len(buf) > 0 {
		return string(buf)
	}
	return string(buf)
}

// 断言：三种错误都能被命令层用 errors.As 取出来，并各自实现 ExitCode()。
// 少了任何一条，命令层就得自己重写一遍退出码表 —— 那份表一定会漂移。
var (
	_ interface{ ExitCode() int } = (*UsageError)(nil)
	_ interface{ ExitCode() int } = (*TransportError)(nil)
	_ interface{ ExitCode() int } = (*WireError)(nil)
	_ error                       = (*UsageError)(nil)
	_ error                       = (*TransportError)(nil)
	_ error                       = (*WireError)(nil)
)
