//go:build !windows

package tzs

// 非 Windows：没有命名管道，也就没有引擎可连。
//
// 这里**刻意不实现**一个「用 unix socket 假装一下」的版本：C# 引擎本身只有 Windows 版
// （它 LoadFrom 的是 .NET Framework/WPF 的程序集，winproc/proc_other.go 的注释也是
// 同一个理由），假装能连只会把一个「环境不对」的错误推迟成「管道上没人应答」。
// 留着这个文件是为了让本包在非 Windows 上能编译（包里的 pure 逻辑、manifest、argmap、
// wire 的单测都能跑）。

import "context"

// DefaultDialer 在非 Windows 上永远失败：没有管道可言。
func DefaultDialer() Dialer { return unsupportedDialer{} }

type unsupportedDialer struct{}

func (unsupportedDialer) Dial(ctx context.Context, pipeName string) (Conn, error) {
	return nil, &TransportError{Code: CodeServerDied,
		Msg: "本平台没有命名管道：.tzs 引擎（tzs-server.exe）只有 Windows 版"}
}

// isPeerGone 在非 Windows 上没有「命名管道断开」这套码：unix socket 的对端关闭
// 本来就读成 io.EOF，不需要归一。
func isPeerGone(error) bool { return false }
