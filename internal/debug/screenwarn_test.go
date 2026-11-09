package debug

import (
	"strings"
	"testing"
	"time"
)

// 画面没接线要说出来（bug 1 的真因）。
//
// 事故背景（用户 2026-10-16 报"debug 走两步自动退出了"）：会话把画面交回 T100 自动指
// = T100 把画面发回 **SSH 来源 IP 的本机 GDC**。本机没起 GDC 时，作业在**第一次画界面**
// 时直接退出：`tbreak main` 停在 MAIN 首条可执行语句（`OPTIONS`），一次单步就走到初始化
// 要开窗口，当场退 —— 而界面上只有"程序已退出(作业窗口被关闭或正常结束)"，与"用户自己
// 关了窗口"分不清。所以这条规则要钉住：**没覆盖 FGLSERVER 且本机没有 GDC 时必须给警告**。
func TestScreenWarnOnlyWhenAutoRouteLooksDead(t *testing.T) {
	// 探针可替换：本机现在有没有 GDC 不该决定这条测试的结果
	old := gdcProbe
	t.Cleanup(func() { gdcProbe = old })

	setProbe := func(present bool) {
		gdcProbe = func() bool { return present }
		gdcProbeMu.Lock() // 绕过 10 秒缓存，让下一次调用真的走探针
		gdcProbeAt = time.Time{}
		gdcProbeMu.Unlock()
	}

	t.Run("没覆盖 + 本机没 GDC → 必须警告且能照做", func(t *testing.T) {
		setProbe(false)
		got := ScreenWarn("")
		if got == "" {
			t.Fatal("本机没有 GDC 又没有 FGLSERVER 覆盖时必须警告")
		}
		for _, want := range []string{"GDC", "7400", "fglserver"} {
			if !strings.Contains(got, want) {
				t.Errorf("警告里该提到 %q（要能照做，而不是只说「没接线」）：%s", want, got)
			}
		}
	})

	t.Run("空白串也算没覆盖", func(t *testing.T) {
		setProbe(false)
		if ScreenWarn("   ") == "" {
			t.Fatal("纯空白的 FGLSERVER 与空串同义（本轮交回 T100），不能当成「有覆盖」")
		}
	})

	t.Run("本机有 GDC → 不吵", func(t *testing.T) {
		setProbe(true)
		if got := ScreenWarn(""); got != "" {
			t.Fatalf("本机明明有 GDC 就不该警告：%s", got)
		}
	})

	t.Run("本轮有覆盖 → 去向由调用方负责，不由这里警告", func(t *testing.T) {
		setProbe(false)
		if got := ScreenWarn("10.1.25.33:1000"); got != "" {
			t.Fatalf("有覆盖时不该由 ScreenWarn 说话（认账的是 dbgScreenVerify）：%s", got)
		}
	})
}

// 退出文案补的那句线索：只在"本轮画面交回 T100 自动指、而且本机没有 GDC"时补。
func TestScreenExitTailOnlyForAutoRoute(t *testing.T) {
	old := gdcProbe
	t.Cleanup(func() { gdcProbe = old })
	gdcProbe = func() bool { return false }
	gdcProbeMu.Lock()
	gdcProbeAt = time.Time{}
	gdcProbeMu.Unlock()

	s := &Session{cfg: &Config{}}
	if got := s.screenExitTail(); !strings.Contains(got, "GDC") {
		t.Fatalf("自动接线 + 本机没 GDC 时，退出文案该补一句线索，实际 %q", got)
	}
	s2 := &Session{cfg: &Config{FGLServer: "10.1.25.33:1000"}}
	if got := s2.screenExitTail(); got != "" {
		t.Fatalf("有覆盖时不该补：%q", got)
	}
}
