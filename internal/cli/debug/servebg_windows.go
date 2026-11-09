//go:build windows

package debug

// Windows 后台常驻：实现在 internal/winproc。
//
// 抽出去是因为多了第二个用它的地方（tt dev tzs 托管 C# 引擎）。这三个函数是纯进程原语，
// 和 HTTP、和 serve 的语义都无关，两处各留一份就是「清掉重复」那条纪律要清的东西。
// spawnDetached 的 TT_SERVE_LOG 注入由 winproc 自己完成，行为与抽走前逐字一致。

import "tt/internal/winproc"

func spawnDetached(exe string, args []string, logPath string) (int, error) {
	return winproc.SpawnDetached(exe, args, logPath)
}

func pidAlive(pid int) bool { return winproc.Alive(pid) }

func killProcess(pid int) error { return winproc.Kill(pid) }
