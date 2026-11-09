package testkit

import (
	"io"
	"os"
	"testing"
)

// CaptureStdout 把一段写到 os.Stdout 的输出收回来，连同它的退出码。
//
// 命令层直接写 os.Stdout（internal/cli/dev 的 `line(w, …)`、internal/cli/dict 的
// `emitWith` 都是），所以只能换掉 os.Stdout 来抓。
//
// **必须边写边读**（2026-09-25 实测的事故）。进程内管道的缓冲约 4 KB，而
// `tt dev tzs --help` 在**引擎可达**时会附上动词索引，整份输出 4,574 字节，恰好越过缓冲 ——
// 于是"先跑完 fn、关掉写端、再读"那种写法会让写端阻塞在管道上，而它在等 fn() 返回，
// fn() 在等写成功：一条测试挂满 10 分钟被 go test 判超时。
// 这里用 goroutine 边读边收，从根上避开。判据是 TestCaptureStdoutSurvivesBigOutput
// （64 KB 输出，远超任何管道缓冲）—— 那一条**必须见过它红**。
//
// fn 返回退出码，与命令函数的签名一致（internal/cli/dev 的 `cmd*` 都是 `func([]string) int`）。
func CaptureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("建管道失败: %v", err)
	}
	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()
	// 用 defer 还原而不是直接赋值：fn 里 panic 时也要把 os.Stdout 还回去，
	// 否则**同一个包后续的用例**都会把输出写进这个管道（进程级状态，跨用例污染）。
	old := os.Stdout
	defer func() { os.Stdout = old }()
	os.Stdout = w
	code := fn()
	w.Close()
	out := <-done
	r.Close()
	return code, out
}
