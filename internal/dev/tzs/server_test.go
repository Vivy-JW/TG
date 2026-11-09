package tzs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/winproc"
)

// TestWorkspaceRefusedWhenUnset 是四条纪律里的第二条：**工作区绝不留缺省**。
//
// 引擎的缺省工作区是一个真实客户目录（Rpc.DefaultWorkspace = D:\t100_wrok_dir\hengshuo\prd），
// 留空就等于拿客户的表单当草稿纸。所以解析不出来时必须拒绝（退出 2），而且**绝不 spawn**。
func TestWorkspaceRefusedWhenUnset(t *testing.T) {
	t.Setenv("TZSCLI_WS", "")
	ws, err := Options{}.workspace()
	if err == nil {
		t.Fatalf("没有工作区时该拒绝，却给出了 %q", ws)
	}
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("该是用法错，得 %T", err)
	}
	if ue.ExitCode() != ExitUsage {
		t.Errorf("该退 2，得 %d", ue.ExitCode())
	}
	if !strings.Contains(err.Error(), "工作区") {
		t.Errorf("文案该点名「工作区」：%v", err)
	}
	// 三种给法都得在文案里（否则用户只知道失败了，不知道去哪配）。
	// 环境那一份要点名是"哪个环境给的"，并指向它的家（按环境配，不是全局键）。
	for _, want := range []string{"--workspace", "TZSCLI_WS", "--env", "工作区目录"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("文案里该有 %q：%v", want, err)
		}
	}
}

// TestWorkspaceFromEnvAndTrim：TZSCLI_WS 是引擎自己也认的那个变量，这里退到它是一致的；
// 两端空白要去掉（命令行/环境变量里多一个空格不该变成另一个工作区 —— 注意只去空白，
// 不做 filepath.Clean：尾斜杠会改变引擎 hash 出来的管道名，那是另一个身份）。
func TestWorkspaceFromEnvAndTrim(t *testing.T) {
	t.Setenv("TZSCLI_WS", `  D:\ws\x  `)
	ws, err := Options{}.workspace()
	if err != nil {
		t.Fatal(err)
	}
	if ws != `D:\ws\x` {
		t.Errorf("去掉两端空白后该是 D:\\ws\\x，得 %q", ws)
	}

	// 显式给的优先于环境变量（命令层的解析顺序里 flag 最高，这里是它的兜底）。
	ws, err = Options{Workspace: `D:\explicit`}.workspace()
	if err != nil {
		t.Fatal(err)
	}
	if ws != `D:\explicit` {
		t.Errorf("显式值该赢，得 %q", ws)
	}
}

// TestWorkspaceSeparatorsAreUnified 钉住 `/` → `\` 这一条替换。
//
// 为什么值得一条测试：`/` 那一种**装不进任何包** —— 实测每一次 `open` 都被设计器拒
// （`NotInCurrentWorkspaceException`），而 `doctor` 对同一个值说「工作区 ok」；
// 同一个字符串还 hash 出**另一个管道名**，于是改一下配置里的斜杠方向就能孤儿化一个
// 正在跑的守护进程（`stop` 与 `reap` 都够不到，只能手工 taskkill）。
//
// 同时钉住**不做**的事：尾分隔符不动、不去尾、不 `filepath.Clean` ——
// 上一条测试的注释立过这条规矩（尾斜杠改变 hash，是另一个身份），而只有正斜杠这一种
// 形态我实测过。空白的处理也与从前一致。
func TestWorkspaceSeparatorsAreUnified(t *testing.T) {
	cases := []struct{ in, want string }{
		{`C:\ws`, `C:\ws`},                   // 反斜杠原样
		{`C:/ws`, `C:\ws`},                   // 正斜杠统一成反斜杠
		{`C:/ws/`, `C:\ws\`},                 // 只换分隔符，**尾斜杠不动**
		{`  C:/ws  `, `C:\ws`},               // 两端空白仍然去掉
		{`\\server\share`, `\\server\share`}, // UNC 不受影响
	}
	for _, c := range cases {
		ws, err := Options{Workspace: c.in}.workspace()
		if err != nil {
			t.Fatalf("%q 该能解析：%v", c.in, err)
		}
		if ws != c.want {
			t.Errorf("%q → %q，想要 %q", c.in, ws, c.want)
		}
	}

	// 同一个目录的两种斜杠必须给**同一个**工作区 —— 这条等式才是这个函数的全部意义
	// （管道名是它的哈希，两个字符串就是两个身份）。
	a, _ := Options{Workspace: `C:\ws\x`}.workspace()
	b, _ := Options{Workspace: `C:/ws/x`}.workspace()
	if a != b {
		t.Errorf("同一个目录两种拼法该给同一个工作区，得 %q vs %q", a, b)
	}
}

// TestEnsureRefusesBeforeTouchingEngine：Ensure 在没有工作区时**连 exe 都不该碰**。
//
// exe 故意给一个不存在的路径：如果它被用到，错误会变成「问不到管道名」，
// 而那说明我们在拒绝之前就已经跑去 exec 了 —— 那正是「绝不 spawn」这条要防的。
func TestEnsureRefusesBeforeTouchingEngine(t *testing.T) {
	t.Setenv("TZSCLI_WS", "")
	pipe, err := Ensure(context.Background(), Options{Exe: filepath.Join(t.TempDir(), "不存在.exe")})
	if err == nil {
		t.Fatalf("该拒绝，却给出了管道名 %q", pipe)
	}
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("该是用法错（退 2），得 %T：%v", err, err)
	}
	if ExitCode(nil, err) != ExitUsage {
		t.Errorf("该退 2，得 %d", ExitCode(nil, err))
	}
}

// TestStopRefusesWithoutWorkspace：stop 不 spawn，但连管道名都不知道时只能拒绝。
func TestStopRefusesWithoutWorkspace(t *testing.T) {
	t.Setenv("TZSCLI_WS", "")
	err := Stop(context.Background(), Options{Exe: filepath.Join(t.TempDir(), "不存在.exe")})
	var ue *UsageError
	if !errors.As(err, &ue) {
		t.Fatalf("该是用法错，得 %T：%v", err, err)
	}
	if ExitCode(nil, err) != ExitUsage {
		t.Errorf("该退 2，得 %d", ExitCode(nil, err))
	}
}

// TestReapEmptyStateIsNoop：状态文件里一条记录都没有时，Reap 不改盘、也不碰 exe。
//
// 一条记录都没有时它连 `PipeName` 都不该调（那是唯一会 exec 的地方）——
// 所以这条同时是「dry-run 绝不写盘」的看门：状态文件在 dry-run 之后必须还是不存在。
func TestReapEmptyStateIsNoop(t *testing.T) {
	dir := t.TempDir()
	pids, err := Reap(context.Background(), Options{
		WorkDir: dir, Exe: filepath.Join(dir, "不存在.exe"),
	}, false)
	if err != nil {
		t.Fatalf("空状态不该报错：%v", err)
	}
	if len(pids) != 0 {
		t.Errorf("空状态该没有可收的 pid，得 %v", pids)
	}
	if _, err := os.Stat(StatePath(dir)); err == nil {
		t.Errorf("dry-run 不该写出状态文件")
	}
}

// TestReapDoesNotKillWhenItCannotTell：**判不了就不动**。
//
// 判据是「记录里的管道名 ≠ 这个工作区现在该有的管道名」，而后者要问引擎。
// 问不出来时（记录里连工作区都没有、或者引擎 exe 没了），孤儿与现役分不开 ——
// 猜错的代价是杀掉一个正在被别的客户端用的守护进程，所以宁可一个都不动。
//
// pid 用**自己这个测试进程**：如果实现不小心把「问不出来」当成「都是孤儿」，
// 这条会当场杀掉测试进程（测试直接崩），这比任何断言都有力。
//
// 「引擎在但问不出名字」那条路线（exe 路径坏了）由 TTZS_E2E 覆盖 ——
// 走到那儿需要真的跑一次 exe，而默认测试一个 exec 都不许有。
func TestReapDoesNotKillWhenItCannotTell(t *testing.T) {
	dir := t.TempDir()
	st := newState()
	st.SetEntry(`D:\ws`, &DaemonEntry{
		PID: os.Getpid(), Pipe: "tzs-cli-00000000-00000000",
		// Workspace 故意留空：Reap 于是无从知道「现在该有的管道名」。
	})
	if err := SaveState(dir, st); err != nil {
		t.Fatal(err)
	}
	pids, err := Reap(context.Background(), Options{WorkDir: dir}, true)
	if err != nil {
		t.Fatalf("不该报错：%v", err)
	}
	if len(pids) != 0 {
		t.Errorf("问不到当前管道名时一个都不该动，得 %v", pids)
	}
	// 记录也该还在（没判成孤儿就不该删）。
	if got, _ := LoadState(dir); got.Entry(`D:\ws`) == nil {
		t.Errorf("记录不该被删掉")
	}
}

// TestReapDropsDeadRecords：pid 明确死掉、又没有孤儿的记录该被清掉（账要平），
// 而且这不需要问引擎。
func TestReapDropsDeadRecords(t *testing.T) {
	dir := t.TempDir()
	st := newState()
	st.SetEntry(`D:\ws`, &DaemonEntry{PID: deadPID, Pipe: "tzs-cli-00000000-00000000", Workspace: `D:\ws`})
	if err := SaveState(dir, st); err != nil {
		t.Fatal(err)
	}
	pids, err := Reap(context.Background(), Options{WorkDir: dir, Exe: filepath.Join(dir, "不存在.exe")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(pids) != 0 {
		t.Errorf("死记录不是回收对象，得 %v", pids)
	}
	if got, _ := LoadState(dir); got.Entry(`D:\ws`) != nil {
		t.Errorf("死记录该被清掉：%+v", got.Daemons)
	}
}

// TestAliveMatchesThisProcess 顺带把 winproc 的判活用一次：我们这个进程一定是活的。
// （不这么来一下，测试里就完全没有「活着」的那一路，判据写反了也看不出来。）
func TestAliveMatchesThisProcess(t *testing.T) {
	if !winproc.Alive(os.Getpid()) {
		t.Fatal("自己这个进程该被判成活着")
	}
	if winproc.Alive(deadPID) {
		t.Fatalf("pid %d 不该被判成活着", deadPID)
	}
}

// TestSpawnArgs：`--daemon` **必须显式给**。
//
// 不传且子进程 stdin 被重定向时（winproc 把 stdin 接成空设备，那就是重定向），
// 引擎会自己选 stdio 模式，管道永远不会出现 —— 现象是每次调用都「冷启动 60 s 未就绪」，
// 而日志里只多一行「模式: stdio」。
func TestSpawnArgs(t *testing.T) {
	got := Options{}.spawnArgs(`D:\ws\x`)
	want := []string{"--daemon", "--workspace", `D:\ws\x`}
	if len(got) != len(want) {
		t.Fatalf("参数个数不对：%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("参数 %d 该是 %q，得 %q", i, want[i], got[i])
		}
	}
	if got[0] != "--daemon" {
		t.Fatal("--daemon 必须是第一个（也是唯一一个模式开关）")
	}
}

// TestExtraEnv：TZSCLI_WS 冗余但故意（任何再派生的路径也落在同一个工作区）；
// TZSCLI_INSTALL 只在给了 InstallDir 时才写，免得用空串把引擎的内置缺省打掉。
func TestExtraEnv(t *testing.T) {
	env := Options{}.extraEnv(`D:\ws\x`)
	if strings.Join(env, "|") != `TZSCLI_WS=D:\ws\x` {
		t.Errorf("只该有 TZSCLI_WS：%v", env)
	}
	for _, e := range env {
		if strings.HasPrefix(e, "TZSCLI_INSTALL=") {
			t.Errorf("没给 InstallDir 时不该写 TZSCLI_INSTALL：%v", env)
		}
	}
	env = Options{InstallDir: `D:\APPS\设计器`}.extraEnv("w")
	if !contains(env, `TZSCLI_INSTALL=D:\APPS\设计器`) {
		t.Errorf("给了 InstallDir 就该带上：%v", env)
	}
}

// TestStateDirPrefersWorkDir：命令层解析好的配置目录（只有它知道 --config）优先；
// 留空时退到 config 的默认落点。
//
// 退到 config 的那一支用 TT_CONFIG 指到临时目录来测 —— 否则这一段会去动
// 开发者机器上真实的 config 目录（那是测试绝不该有的副作用）。
func TestStateDirPrefersWorkDir(t *testing.T) {
	dir := t.TempDir()
	if got := (Options{WorkDir: dir}).StateDir(); got != dir {
		t.Errorf("WorkDir 该被直接用：%q", got)
	}
	cfg := filepath.Join(dir, "config.json")
	t.Setenv("TT_CONFIG", cfg)
	if got := (Options{}).StateDir(); got != dir {
		t.Errorf("留空时该退到 TT_CONFIG 所在目录 %q，得 %q", dir, got)
	}
	if LogPath(dir) != filepath.Join(dir, LogFileName) {
		t.Errorf("日志该落在同一个目录")
	}
}

// TestLogTail：启动失败时要贴得出日志末尾；文件不存在时也要给一句人话。
func TestLogTail(t *testing.T) {
	dir := t.TempDir()
	if got := logTail(filepath.Join(dir, "没有.log"), 100); !strings.Contains(got, "无日志文件") {
		t.Errorf("缺文件该给一句人话，得 %q", got)
	}
	p := filepath.Join(dir, "a.log")
	if err := os.WriteFile(p, []byte("第一行\n第二行\n第三行\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := logTail(p, 100); !strings.Contains(got, "第三行") {
		t.Errorf("该贴出末尾：%q", got)
	}
	// 只取末尾 max 个字节（日志可能很长）。
	if got := logTail(p, 4); len([]byte(got)) > 4 {
		t.Errorf("该只取末尾 4 字节，得 %q", got)
	}
}

// deadPID 是一个**一定不存在**的 pid。用 0x7FFFFFFF 而不是 -1：
// Reap 的判据里有 `PID > 0`（-1 走不到「判断它死没死」那一步），
// 而这个值大到 Windows 的 pid 空间里不可能出现，OpenProcess 必然失败。
const deadPID = 0x7FFFFFFF
