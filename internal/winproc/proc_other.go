//go:build !windows

package winproc

// 非 Windows 的后台子进程：以 Setsid 脱离会话，输出重定向到日志文件，父进程退出后继续运行。
//
// C# 引擎本身只有 Windows 版（它 LoadFrom 的是 .NET Framework/WPF 的程序集），所以这条路径
// 实际只服务 `tt serve` 的后台常驻。留着是为了让 internal/cli/debug 不必再分平台维护一份。

import (
	"os"
	"os/exec"
	"syscall"
)

// HasConsole 报告本进程是不是挂着一个终端。非 Windows 上"控制台窗口"这件事不成立
// （子进程不会因为父进程没有终端就弹窗），恒返回 true，于是 HideConsole 是空操作。
func HasConsole() bool { return true }

// HideConsole 在非 Windows 上是空操作（见 HasConsole 与 Windows 版那份注释）。
func HideConsole(c *exec.Cmd) {}

// SpawnDetached 启动一个脱离子进程（不等待），返回 pid。语义与 Windows 版一致。
func SpawnDetached(exe string, args []string, logPath string, extraEnv ...string) (int, error) {
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return 0, err
	}
	defer logf.Close()

	cmd := exec.Command(exe, args...)
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.Stdin = nil
	cmd.Env = append(append(os.Environ(), "TT_SERVE_LOG="+logPath), extraEnv...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go cmd.Wait()
	return cmd.Process.Pid, nil
}

// Alive 判断 pid 是否存活。
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

// Kill 结束指定进程。语义与 Windows 版一致：进程已经不在了算成功（收尾要的是
// "它别再挡路"），真的杀不掉才报错。
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
	if kerr := p.Kill(); kerr != nil {
		if !Alive(pid) {
			return nil
		}
		return kerr
	}
	return nil
}
