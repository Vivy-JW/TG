package debug

// 本地服务的前台/后台运行支持 —— 调试服务与统一服务共用这一份。
//
//   - 默认后台常驻(单实例):打印实际地址后立即返回,终端/会话不被占用,可继续输入其它命令;
//   - `--foreground` 前台运行(调试用,日志直接到终端);
//   - `--stop` 停止后台实例;
//   - 运行状态记录在 config.json 同目录的 .tt-serve.json(pid/地址/API 前缀/日志),
//     端口被占用自动顺延时,控制端命令据状态文件自动找到真实地址。
//
// 两个服务是同一套机制的两个实例,只差三样东西(见 ServeSpec):
//
//	tt debug serve   独立调试服务,调试 API 直接在根(/api/…)
//	tt serve         统一服务(工作台+设置页),调试面挂在 /debug/api/…
//
// 所以状态文件里要记 APIBase:控制端命令照着它拼地址,不必知道对面是哪个服务。
// 漏了它,`tt debug status` 就会拿 /api/status 去问一个挂在 /debug/api/ 的服务,
// 表现是 404「未知接口」—— 看着像服务没起来,其实只是问错了地方。
//
// 状态文件的格式、探活方式、停止逻辑只有这一份:两个服务都走
// RunForeground / StartBackground / StopBackground,不再各写一套"读状态文件 + 杀 pid"。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/debug"
)

const (
	// APIBaseAtRoot / APIBaseUnderDebug 是两种挂载布局下调试 API 的前缀。
	// 独立调试服务直接在根;统一服务把子系统挂在前缀下(见 internal/web 的 routes)。
	APIBaseAtRoot     = ""
	APIBaseUnderDebug = "/debug"
)

var (
	dbgServeForeground bool
	dbgServeStop       bool
)

// ServeSpec 描述一次服务运行 —— 两个服务只差这几项文案与参数。
type ServeSpec struct {
	Prefix   string   // 提示语前缀,如 "[tt debug]"
	StopHint string   // 停止方式,如 "tt debug serve --stop"
	Args     []string // 后台子进程重放的子命令(如 ["debug","serve"]);StartBackground 会补 --foreground 与 --config
	APIBase  string   // 调试 API 前缀,记进状态文件供控制端寻址(见上)
}

// ServeInfo 是后台实例状态:pid/站点地址/调试 API 前缀/日志,供单实例判断与控制端寻址。
type ServeInfo struct {
	PID     int    `json:"pid"`
	URL     string `json:"url"`
	APIBase string `json:"apiBase,omitempty"`
	Log     string `json:"log"`
	Since   string `json:"since"`
}

// serveInfoFile 返回状态文件路径(dir = config.json 所在目录)。
// 文件名与落点规则在 config(统一路径管理器)——这里只传数据目录。
func serveInfoFile(dir string) string { return filepath.Join(dir, config.ServeStateFileName) }

func writeServeInfo(st *ServeInfo, dir string) error {
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(serveInfoFile(dir), b, 0o644)
}

func readServeInfo(dir string) (*ServeInfo, error) {
	b, err := os.ReadFile(serveInfoFile(dir))
	if err != nil {
		return nil, err
	}
	var st ServeInfo
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, err
	}
	if st.PID <= 0 || st.URL == "" {
		return nil, fmt.Errorf("状态文件无效: %s", serveInfoFile(dir))
	}
	return &st, nil
}

func removeServeInfo(dir string) { _ = os.Remove(serveInfoFile(dir)) }

// serveProbe 是探活时依次尝试的路径与应答里的标记。
//
// 两种布局的服务共用同一份状态文件,探活必须都认:只看根下的 /api/status 会把正在
// 运行的统一服务误判成"没在跑" —— 于是单实例判断失效(会再拉一个起来抢端口),
// --stop 也会说找不到它。标记取自各服务 /api/status 里的 server 字段。
var serveProbe = []struct{ Path, Mark string }{
	{"/api/status", "tdebug-debug"},       // 独立调试服务(根布局)
	{"/debug/api/status", "tdebug-debug"}, // 统一服务:调试面在前缀下
	{"/api/health", "tt-unified"},         // 统一服务自己的健康检查
}

// serveUp 探活:任一布局命中已知标记才算运行中,
// 避免把占用同端口的其它 HTTP 服务误判为本实例。
func serveUp(url string) bool {
	if url == "" {
		return false
	}
	root := strings.TrimRight(url, "/")
	cli := &http.Client{Timeout: 2 * time.Second}
	for _, p := range serveProbe {
		resp, err := cli.Get(root + p.Path)
		if err != nil {
			continue
		}
		buf := make([]byte, 512)
		n, _ := resp.Body.Read(buf)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK && strings.Contains(string(buf[:n]), p.Mark) {
			return true
		}
	}
	return false
}

// runningInstance 返回本 config 目录下正在运行的后台实例;无则返回 nil。
func runningInstance(dir string) *ServeInfo {
	st, err := readServeInfo(dir)
	if err != nil {
		return nil
	}
	if !serveUp(st.URL) {
		return nil
	}
	return st
}

// RunningServe 报告这个数据目录下的后台服务是否**真的在跑**：不在跑返回 nil。
//
// 供 tt update 在升级前后判断"要不要先把它停掉、升完要不要拉回来"。
// 注意判据是"连得上"而不是"状态文件里有 pid"（见 runningInstance），
// 否则一个残留的 pid 会让升级去杀一个不相干的进程。
func RunningServe(dir string) *ServeInfo { return runningInstance(dir) }

// configDir 定位 config.json 所在目录(状态文件/日志存放处)。
func configDir() string {
	if p, err := resolveConfigPath(); err == nil {
		return config.LocationsAt(p).DataDir()
	}
	return config.LocationsAt(config.DefaultConfigPath()).DataDir()
}

// apiBaseFromState 读状态文件里记的调试 API 前缀。
// 读不到(没实例/旧状态文件)按空处理 —— 那是独立调试服务的根布局,与旧行为一致。
func apiBaseFromState() string {
	if st, err := readServeInfo(configDir()); err == nil {
		return st.APIBase
	}
	return ""
}

// configListen 读配置里的监听地址(尽力而为,失败返回空)。
//
// 合并前读的是 debug.listen(那时监听地址挤在 debug 节里);拆节后监听地址在
// 顶层 listen,由统一配置层 config.Load 读出 —— 它还保留了对旧 debug.listen 的
// 回落,所以老配置文件不用改也能读到。
func configListen() string {
	p, err := resolveConfigPath()
	if err != nil {
		return ""
	}
	root, err := config.Load(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(root.Listen)
}

// logTail 返回日志文件末尾内容(后台启动失败时给出线索)。
func logTail(logPath string, max int) string {
	f, err := os.Open(logPath)
	if err != nil {
		return "(无日志文件)"
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
	buf := make([]byte, int(st.Size()-off))
	if _, err := f.ReadAt(buf, off); err != nil {
		return string(buf)
	}
	return string(buf)
}

// printRunning 打印实例地址块(前台/后台启动共用;两个服务只差前缀与停止方式)。
//
// 日志路径非空 = 服务由 StartBackground 脱离终端跑着,不提示 Ctrl+C;
// 为空 = 前台运行,日志就在当前终端上。
func printRunning(spec ServeSpec, st *ServeInfo, cfgPath string) {
	fmt.Printf("%s 服务已启动  %s  (pid %d)\n", spec.Prefix, st.URL, st.PID)
	fmt.Printf("  配置文件: %s\n", cfgPath)
	if st.Log != "" {
		fmt.Printf("  运行日志: %s\n", st.Log)
		fmt.Printf("  停止服务: %s\n", spec.StopHint)
		return
	}
	fmt.Println("  按 Ctrl+C 停止")
}

// RunForeground 前台阻塞运行一个服务:Ctrl+C 优雅收尾,并写状态文件供 --stop 与
// 控制端命令自动寻址。
//
// start 负责建监听并服务,ctx 取消时返回;它在监听建好后回调 onReady(addr),让本函数
// 写盘并打印地址块 —— 端口顺延时真实地址只有 start 知道。addr 用能直接访问的形式
// (0.0.0.0 会被换成 127.0.0.1),因为状态文件里的地址是要给客户端点的。
//
// 同一份配置目录下同时只允许一个实例:两条进程会争同一份状态文件,
// 后写的那条会覆盖前一条的记录,--stop 与自动寻址就都指错人了。
func RunForeground(cfgPath string, spec ServeSpec, start func(ctx context.Context, onReady func(addr string)) error) error {
	dir := filepath.Dir(cfgPath)
	if st := runningInstance(dir); st != nil {
		return fmt.Errorf("服务已在运行: %s (pid %d),如需重启请先执行 %s", st.URL, st.PID, spec.StopHint)
	}
	removeServeInfo(dir) // 清理残留状态

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	err := start(ctx, func(addr string) {
		st := &ServeInfo{
			PID: os.Getpid(), URL: "http://" + addr, APIBase: spec.APIBase,
			Log: os.Getenv("TT_SERVE_LOG"), Since: time.Now().Format(time.RFC3339),
		}
		_ = writeServeInfo(st, dir)
		printRunning(spec, st, cfgPath)
	})
	removeServeInfo(dir)
	if err != nil {
		return err
	}
	fmt.Printf("%s 服务已停止\n", spec.Prefix)
	return nil
}

// StartBackground 默认入口:单实例,后台常驻;实例已运行则打印现状直接返回。
//
// 拉起一个脱离终端的自身跑前台,等它建好监听、写出状态文件再返回 —— 端口被占用会顺延,
// 真实地址只有子进程知道,不等就只能瞎猜。
func StartBackground(cfgPath string, spec ServeSpec) (*ServeInfo, error) {
	dir := filepath.Dir(cfgPath)
	if st := runningInstance(dir); st != nil {
		fmt.Printf("%s 服务已在后台运行(单实例):\n", spec.Prefix)
		fmt.Printf("  地址: %s  (pid %d)\n", st.URL, st.PID)
		if st.Log != "" {
			fmt.Printf("  运行日志: %s\n", st.Log)
		}
		fmt.Printf("  停止服务: %s\n", spec.StopHint)
		return st, nil
	}
	removeServeInfo(dir)

	logPath := filepath.Join(dir, config.ServeLogFileName)
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("定位可执行文件失败: %w", err)
	}
	args := append(append([]string{}, spec.Args...), "--foreground", "--config", cfgPath)
	pid, err := spawnDetached(exe, args, logPath)
	if err != nil {
		return nil, fmt.Errorf("启动后台服务失败: %w", err)
	}

	// 等子进程建好监听并写状态(端口被占用会顺延,需读回真实地址)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if st := runningInstance(dir); st != nil && st.PID == pid {
			printRunning(spec, st, cfgPath)
			return st, nil
		}
		if !pidAlive(pid) {
			return nil, fmt.Errorf("后台服务进程提前退出,日志末尾:\n%s", logTail(logPath, 2000))
		}
		time.Sleep(250 * time.Millisecond)
	}
	return nil, fmt.Errorf("后台服务 15 秒内未就绪(端口可能全部被占用),日志末尾:\n%s", logTail(logPath, 2000))
}

// StopBackground 停止 dir(config.json 所在目录)记录的后台服务,并清理状态文件。
//
// 这是"停后台实例"的唯一实现:两个服务的 `--stop` 都只是拿配置目录调它 ——
// 状态文件格式与 pid 处置只有这一份。
//
// 状态文件损坏 / 进程已不在时:清掉残留文件并给出可照做的报错,
// 不让一个陈旧的状态文件把下一次启动挡住。
func StopBackground(dir string) error {
	st, err := readServeInfo(dir)
	if err != nil {
		return fmt.Errorf("没有后台服务在运行(未找到状态文件 %s)", serveInfoFile(dir))
	}
	if !pidAlive(st.PID) {
		removeServeInfo(dir)
		return fmt.Errorf("后台服务已不在运行(pid %d),已清理状态文件", st.PID)
	}
	if err := killProcess(st.PID); err != nil {
		return fmt.Errorf("停止服务失败(pid %d): %w", st.PID, err)
	}
	for i := 0; i < 50 && pidAlive(st.PID); i++ {
		time.Sleep(200 * time.Millisecond)
	}
	removeServeInfo(dir)
	fmt.Printf("已停止后台服务 (pid %d)\n", st.PID)
	return nil
}

// debugServeForeground 前台运行:阻塞到 Ctrl+C;同时写状态文件供 --stop 使用。
func debugServeForeground(cfg *debug.Config, cfgPath string) error {
	requested := cfg.Listen
	cfg.DataDir = filepath.Dir(cfgPath)
	srv := debug.NewServer(cfg, common.WebFrontend(), cfgPath)

	return RunForeground(cfgPath, ServeSpec{
		Prefix:   "[tt debug]",
		StopHint: "tt debug serve --stop",
		APIBase:  APIBaseAtRoot,
	}, func(ctx context.Context, onReady func(addr string)) error {

		ln, addr, err := srv.Listen()
		if err != nil {
			return err
		}
		if addr != requested {
			fmt.Printf("[tt debug] 端口 %s 被占用,自动改用 %s\n", requested, addr)
		}
		onReady(addr)
		return srv.Serve(ctx, ln)
	})
}

// debugServeBackground 默认入口:单实例,后台常驻。
func debugServeBackground(cfgPath string) error {
	args := []string{"debug", "serve"}
	if dbgListen != "" {
		args = append(args, "--listen", dbgListen)
	}
	_, err := StartBackground(cfgPath, ServeSpec{
		Prefix:   "[tt debug]",
		StopHint: "tt debug serve --stop",
		APIBase:  APIBaseAtRoot,
		Args:     args,
	})
	return err
}

// debugServeStop 停止后台实例(按状态文件 pid)。
func debugServeStop() error { return StopBackground(configDir()) }
