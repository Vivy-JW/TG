//go:build !windows

package debug

// 非 Windows 后台常驻：实现在 internal/winproc（见 servebg_windows.go 的说明）。

import "tt/internal/winproc"

func spawnDetached(exe string, args []string, logPath string) (int, error) {
	return winproc.SpawnDetached(exe, args, logPath)
}

func pidAlive(pid int) bool { return winproc.Alive(pid) }

func killProcess(pid int) error { return winproc.Kill(pid) }
