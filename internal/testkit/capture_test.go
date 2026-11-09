package testkit

import (
	"fmt"
	"os"
	"testing"
)

// TestCaptureStdoutSurvivesBigOutput 钉住 CaptureStdout 的边界面（约 4 KB）。
//
// 判据是"它回来了" —— 而这一条**必须见过它红**：把 CaptureStdout 退回
// "先跑完 fn、关掉写端、再读"的写法，这条测试不会失败，它会**挂住**
// （写端阻塞在管道缓冲上，读端在等 fn 返回），最后靠 go test 的超时兜底。
// 所以输出量给到 64 KB，远超任何管道缓冲。
//
// 这个坑的来历见 CaptureStdout 的注释：它是在"环境变量恰好把引擎指通了"之后
// 偶然翻出来的，默认跑测试时引擎不可达、输出小，所以一直没显形。
func TestCaptureStdoutSurvivesBigOutput(t *testing.T) {
	const n = 64 * 1024
	code, out := CaptureStdout(t, func() int {
		for i := 0; i < n; i++ {
			fmt.Fprint(os.Stdout, "x")
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("退出码该原样传出来，得 %d", code)
	}
	if len(out) != n {
		t.Fatalf("收了 %d 字节，想 %d —— 大的没全回来（管道那侧漏了）", len(out), n)
	}
}

// TestCaptureStdoutReturnsCode 退出码必须原样出来：内部/dev/cli 的几条用例
// 正是靠它断言"这条命令该退几"。
func TestCaptureStdoutReturnsCode(t *testing.T) {
	for _, want := range []int{0, 2, 5} {
		code, out := CaptureStdout(t, func() int { return want })
		if code != want {
			t.Errorf("退出码想 %d，得 %d", want, code)
		}
		if out != "" {
			t.Errorf("没写东西时该是空串，得 %q", out)
		}
	}
}
