//go:build windows

// Package winproc 只做三件事：起一个不随本进程死的子进程、判它活着、杀掉它。
//
// 单独成包是因为有两处要用：`tt serve` 把自己重新拉起来做后台常驻（internal/cli/debug），
// 和 `tt dev tzs` 托管 C# 引擎（internal/dev/tzs）。原来只有前者，实现就住在
// internal/cli/debug/servebg_windows.go 里；第二处出现时按「清掉重复」的纪律抽出来。
package winproc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
)

// createNoWindow 是 Windows 的 CREATE_NO_WINDOW：给**控制台程序**起一个没有控制台的子进程。
// 注意它只能"不给控制台"，不能把已有的窗口藏起来（SpawnDetached 那边用的是 DETACHED_PROCESS）。
const createNoWindow = 0x08000000

// HasConsole 报告本进程是不是挂着一个控制台。
//
// 判据是 kernel32 的 GetConsoleWindow()（没有控制台时返回 0）。x/sys/windows 没包这个函数，
// 所以自己取一次过程地址。它要回答的场景见 HideConsole。
func HasConsole() bool {
	h, _, _ := procGetConsoleWindow.Call()
	return h != 0
}

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow")
)

// HideConsole 让**控制台子程序**不要在父进程没有控制台时弹出一个新的控制台窗口。
//
// 为什么需要它（用户实测：点界面上的一个按钮就弹出一大堆终端窗口/标签页）：
// `tt serve` 起后台子进程走的是 winproc.SpawnDetached（DETACHED_PROCESS）——
// **没有控制台**。而它内部用 runSelf/runSelfOut 起的是一串控制台程序（`tt debug start`、
// `tt debug exec`、`tt debug quit`…）：父进程没有控制台时，Windows 会给**每一个**子进程
// **新分配一个控制台**，于是每起一个子进程就多一个窗口（Windows Terminal 下是多一个标签页）。
//
// 父进程**已经有**控制台时不加这个标志：那时子进程照旧继承同一个控制台 ——
// 用户在自己终端里跑同一条命令时，交互与输出一点都不变。
//
// 只动 CreationFlags 的这一个位，别的位（调用方可能已经设了 DETACHED_PROCESS 之类）原样保留。
func HideConsole(c *exec.Cmd) {
	if HasConsole() {
		return
	}
	if c.SysProcAttr == nil {
		c.SysProcAttr = &syscall.SysProcAttr{}
	}
	c.SysProcAttr.CreationFlags |= createNoWindow
}

// SpawnDetached 启动一个脱离子进程（不等待），返回 pid。输出追加到 logPath，stdin 是空设备。
// extraEnv 追加在 os.Environ() 之后（同名后者胜），用来把工作区之类的设置传给子进程。
//
// 关于句柄继承 —— 这里**故意不做** C# 侧那种「扫全句柄表清 HANDLE_FLAG_INHERIT」的封印，
// 因为 Go 不需要：syscall.StartProcess 只要会继承句柄，就一定会设
// PROC_THREAD_ATTRIBUTE_HANDLE_LIST，把继承限制在 ProcAttr.Files 那三个上
// （exec_windows.go:420-427，注释原文 "Do not accidentally inherit more than these handles."）。
//
// 这一步在 C# 那边是必须的，而且踩得很深：Process.Start 无差别复制每一个可继承句柄，于是
// spawn 出来的守护进程拿到了调用方捕获 stdout 的那个管道写端，调用方（subprocess 的
// capture_output、CI runner、`| head`）永远等不到 EOF。只清 GetStdHandle(STD_*) 三个还不够——
// .NET 运行时在 Main 之前给自己复制了第二个可继承的 stdout 句柄，点不到它。
//
// 本包实测过这个差异（临时探针，两种模式都试了）：父进程以 capture_output 起一个 Go 进程，
// 后者 DETACHED_PROCESS 拉起真正的 tzs-server，父进程 0.38 s 拿到 EOF，不挂。
func SpawnDetached(exe string, args []string, logPath string, extraEnv ...string) (int, error) {
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer logf.Close()

	cmd := exec.Command(exe, args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.Stdin = nil // os.DevNull：子进程不该等着谁的输入
	cmd.Env = append(append(os.Environ(), "TT_SERVE_LOG="+logPath), extraEnv...)

	// DETACHED_PROCESS(0x8) 脱离控制台；CREATE_NEW_PROCESS_GROUP(0x200) 独立进程组
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x8 | 0x200}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait() // 回收句柄，不阻塞调用方
	return cmd.Process.Pid, nil
}

// Alive 判断 pid 是否存活。用于快速失败，**不用于身份判断**：pid 会被复用，
// 而且「进程活着」不等于「它在监听」。
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == 259 // STILL_ACTIVE
}

// Kill 结束指定进程。
//
// 两种失败要分开对待，因为处置完全不同：
//   - 进程**已经不在了**：当作成功 —— 收尾要的是"它别再挡路"，不是"我亲手杀了它"。
//     收工的时机恰恰是会话正在退场的时候（pid 也可能刚被复用），把这一下当成失败，
//     用户看到的就是"收尾有失败"，可现场其实干干净净。
//   - **拒绝访问**：它是提权启动的（T100 客户端拉起的那个 gdc 就是，登记里出现它的
//     唯一可能是配置指错了端口），普通权限结束不了 —— 照实报出来并给出处置，
//     而不是甩一句 `OpenProcess: Access is denied.` 让人去猜。
func Kill(pid int) error {
	if pid <= 0 {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		if !Alive(pid) {
			return nil
		}
		return err
	}
	kerr := p.Kill()
	if kerr == nil {
		return nil
	}
	// 退场中的进程会短暂拒绝 PROCESS_TERMINATE：等一下再看一眼它还在不在。
	time.Sleep(300 * time.Millisecond)
	if !Alive(pid) {
		return nil
	}
	if errors.Is(kerr, windows.ERROR_ACCESS_DENIED) {
		return fmt.Errorf("结束进程 %d 被拒（错误 5:拒绝访问）—— 它多半是提权启动的,"+
			"普通权限结束不了:请用管理员身份结束它,或确认登记的 pid 是不是这一轮的", pid)
	}
	return kerr
}
