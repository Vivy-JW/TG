package winproc

// Kill 的边界：**进程已经不在了要算成功**。
//
// 为什么值得钉住：收工（`tt session rm`、面板那个「收工」按钮）的时机恰好是会话
// 正在退场的时候 —— 登记的 pid 可能刚退完、也可能正处在退场中。原来这里把
// `os.FindProcess` 的失败直接往上抛，于是用户看到"收尾有失败: OpenProcess: Access is denied."，
// 而现场其实干干净净（只剩一个窗口都没留下的空登记）。这条测试就是那次现场的最小复现。

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestKillGonePIDIsNotAnError(t *testing.T) {
	for _, pid := range []int{0, -1} {
		if err := Kill(pid); err != nil {
			t.Fatalf("Kill(%d) 应当当成功（本来就没有这个进程）: %v", pid, err)
		}
	}

	// 起一个立刻退出的探针进程，等它真的退掉，再"杀"它一次。
	exe, args := "cmd", []string{"/c", "exit"}
	if runtime.GOOS != "windows" {
		exe, args = "true", nil
	}
	pid, err := SpawnDetached(exe, args, filepath.Join(t.TempDir(), "probe.log"))
	if err != nil {
		// 不跳过：SpawnDetached 是这个包唯一的生产用途（serve 后台常驻、引擎托管），
		// 它起不来就是这台机器上的真问题，不该静默变成一次空转。
		t.Fatalf("起不了探针进程（SpawnDetached 坏了吗？）: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for Alive(pid) && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if Alive(pid) {
		t.Fatalf("探针进程 %d 十秒都没退（它只是 `%s %v`）", pid, exe, args)
	}
	if err := Kill(pid); err != nil {
		t.Fatalf("已经退掉的进程应当当成功（收尾的常见时机），实得: %v", err)
	}
}