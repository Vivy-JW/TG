package dev

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"tt/internal/cli/dev/common"
)

// TestDevCobraRouting 钉住 cobra 树这一层的**路由与退出码契约**。
//
// 单元测试（cmdXxx 直调）覆盖动词核心，但"cobra 把参数送到哪个动词、退出码怎么
// 透传出来"只有这层知道 —— 组命令没有 RunE 时未知子命令会退 0、交给根命令则退 1，
// 都不符合 tdev 契约（usage 错 = 2）。这条测试每次用 newDevCmd() 造一棵**新树**，
// 避免 pflag/cobra 的包级状态在用例间串味。
//
// 根开关翻译（takeRootFlags）也在这层测：它是叶子级透传线上唯一的"根命令 ↔ tdev"接缝。
func TestDevCobraRouting(t *testing.T) {
	// run 执行一条 dev 线命令，返回 (退出码, stdout, stderr)。stdout/stderr 全程捕获，
	// 免得测试输出里混进 Usage。
	run := func(args ...string) (int, string, string) {
		t.Helper()
		var outBuf, errBuf bytes.Buffer
		// 动词核心直接写 os.Stdout / os.Stderr（testkit.CaptureStdout 是全局捕获），
		// 这里换成管道再逐字回读，保证拿到的是这一次执行的输出。
		oldOut, oldErr := os.Stdout, os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout, os.Stderr = wOut, wErr
		done := make(chan struct{})
		// 两条管道**同时**排干：标准输出与标准错误是两个独立的管道，各有自己的缓冲
		// （Windows 上 os.Pipe 的默认容量 4096）。串行排（先等 stdout EOF 再读 stderr）
		// 会与生产端互相死等：裸 `tt dev` 先把 Usage 一次写到 **stderr**，一旦这段文本
		// 超过缓冲容量，写端就阻塞 —— 而这里正卡在"等 stdout EOF"上，stdout 的写端要等
		// 命令返回才关。Usage 曾涨到 4.7 KB（加 `formcheck` 那一段）时踩响过。
		go func() {
			_, _ = io.Copy(&outBuf, rOut)
			done <- struct{}{}
		}()
		go func() {
			_, _ = io.Copy(&errBuf, rErr)
			done <- struct{}{}
		}()

		cmd := newDevCmd()
		cmd.SetArgs(args)
		err := cmd.Execute()

		os.Stdout, os.Stderr = oldOut, oldErr
		_ = wOut.Close()
		_ = wErr.Close()
		<-done
		<-done
		return common.ExitCodeOf(err), outBuf.String(), errBuf.String()
	}

	t.Run("裸dev打Usage退2", func(t *testing.T) {
		code, _, errOut := run()
		if code != 2 {
			t.Errorf("裸 tt dev 退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "tt dev tzc export") {
			t.Error("裸 tt dev 应当打 Usage")
		}
	})

	t.Run("未知子命令退2", func(t *testing.T) {
		code, _, errOut := run("nope")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, `未知子命令 "nope"`) {
			t.Errorf("应当报未知子命令，得到：%s", errOut)
		}
	})

	t.Run("tzc未知动词退2", func(t *testing.T) {
		code, _, errOut := run("tzc", "nope")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, `未知动词 "nope"`) {
			t.Errorf("应当报未知动词，得到：%s", errOut)
		}
	})

	t.Run("裸tzc打Usage退2", func(t *testing.T) {
		code, _, errOut := run("tzc")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "Usage") && !strings.Contains(errOut, "用法") {
			t.Error("裸 tt dev tzc 应当打 Usage")
		}
	})

	t.Run("install墓碑退2给指引", func(t *testing.T) {
		code, _, errOut := run("install")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "tt install") {
			t.Error("墓碑应当指引到 tt install")
		}
	})

	t.Run("export缺参数退2", func(t *testing.T) {
		// 动词核心自己的用法检查（NArg<1 → 用法 + 退 2），经 cobra 透传后不变。
		code, _, errOut := run("tzc", "export")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "用法：tt dev tzc export") {
			t.Errorf("应当打动词用法，得到：%s", errOut)
		}
	})

	t.Run("动词help退0", func(t *testing.T) {
		// 原 stdlib flag 把 --help 当未知开关退 2；现在与 tzs 线、cobra 惯例一致（说明书退 0）。
		for _, h := range []string{"--help", "-h"} {
			code, out, _ := run("tzc", "export", h)
			if code != 0 {
				t.Errorf("%s 退出码 = %d, 期望 0", h, code)
			}
			if !strings.Contains(out, "tt dev tzc export <pkg.tzc>") {
				t.Errorf("%s 应当打动词用法，得到：%s", h, out)
			}
		}
	})

	t.Run("csv被明确拒绝", func(t *testing.T) {
		code, _, errOut := run("tzc", "export", "--csv")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "没有 CSV") {
			t.Errorf("应当点名拒绝 CSV，得到：%s", errOut)
		}
	})

	// --env 从前在这里被**明确拒绝**（"它是 tt dict / tt debug 那条线的开关"）。
	// 现在它被接受:工作区按环境配,而"默认环境"已移除,所以这条线必须能收到一个点名的
	// 环境(tzs 用它取该环境的 workspace)。这里钉住"摘出来后落进了注入的 setter"。
	t.Run("env被桥接到setter", func(t *testing.T) {
		var got string
		oldGet, oldSet := envNameFn, envSetFn
		envNameFn, envSetFn = func() string { return got }, func(v string) { got = v }
		t.Cleanup(func() { envNameFn, envSetFn = oldGet, oldSet })

		run("tzc", "export", "--env", "正式区")
		if got != "正式区" {
			t.Errorf("--env 未被桥接: %q", got)
		}
		// 带等号的写法是同一件事(根开关两种写法都要认)
		got = ""
		run("tzc", "export", "--env=测试")
		if got != "测试" {
			t.Errorf("--env= 写法未被桥接: %q", got)
		}
		// --conn 是别名
		got = ""
		run("tzc", "export", "--conn", "别名区")
		if got != "别名区" {
			t.Errorf("--conn 别名未被桥接: %q", got)
		}
	})

	// 缺参数的 --env 要报错(不能静默当成"没给",那会退回"没点名"再报一个含糊的错)
	t.Run("env缺参数被拒", func(t *testing.T) {
		code, _, errOut := run("tzc", "export", "--env")
		if code != 2 {
			t.Errorf("退出码 = %d, 期望 2", code)
		}
		if !strings.Contains(errOut, "环境名") {
			t.Errorf("应当说清它要一个环境名，得到：%s", errOut)
		}
	})

	t.Run("config翻译成TT_CONFIG", func(t *testing.T) {
		// --config 走 TT_CONFIG 环境变量这条桥（见 takeRootFlags）。用一个不可能存在的
		// 配置路径指向临时目录，验证 setenv 确实发生：export 会因此读到"缺省落点在临时目录"
		// 的配置 —— 这里只验证桥本身，不打真包：给 export 一个缺参数路径，退出码仍是 2，
		// 但桥是否生效用环境变量回读断言。
		dir := t.TempDir()
		t.Setenv("TT_CONFIG", "")
		run("tzc", "export", "--config", dir+`\cfg.json`)
		if got := os.Getenv("TT_CONFIG"); got != dir+`\cfg.json` {
			t.Errorf("TT_CONFIG 未被桥接: %q", got)
		}
	})
}
