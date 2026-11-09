//go:build windows

package tzs

// Windows 命名管道客户端。
//
// 为什么用 syscall.CreateFile 而不是 os.OpenFile：命名管道的 `dwShareMode` 必须是 0，
// 而 os.OpenFile 走的 syscall.Open 会带上 FILE_SHARE_READ|WRITE|DELETE。CreateFile
// 是标准库里现成的那个（syscall_windows.go 里 os 自己也在用），只差一个参数。
//
// 读超时**不在这里做**：实测（临时探针，本机跑过）os.NewFile 包住的管道句柄上
// SetReadDeadline 返回 "file type does not support deadline" —— 同步打开的命名管道
// 没有可用的取消手段（.NET 那边也一样：PipeStream 拒绝 ReadTimeout）。
// 所以超时由 CallConn 用「goroutine + 定时器」实现，与本文件无关。

import (
	"context"
	"errors"
	"os"
	"syscall"
	"time"
	"unsafe"
)

// errorPipeBusy = ERROR_PIPE_BUSY(231)。标准库 syscall 只为少数几个 Win32 错误码
// 建了具名常量，这个不在其中；写数字而不写名字会让这里变成一个魔法值，所以起个名字。
//
// 它的含义是「名字有，但所有实例都正忙」—— 引擎每台机器只开一个实例
// （NamedPipeServerStream 的 maxNumberOfServerInstances=1），所以第二个客户端必然
// 撞上它。契约里那句「第二个客户端会在 Connect 里等」说的就是这件事。
const errorPipeBusy syscall.Errno = 231

// 「对端走了」的码（ERROR_BROKEN_PIPE 在标准库里有，另外三个在这里起名字）。
//
// 为什么这些都要收：Windows 命名管道的文档只讲 ERROR_BROKEN_PIPE，实测却不是
// （见 client.go 的 Recv 与 pipe_windows_test.go 的 EOF 用例）—— 对端 CloseHandle
// 还是 DisconnectNamedPipe、以及是不是还有未读数据，都会换一个码。
// 漏掉任何一个的后果都一样：守护进程死掉会被报成「连接中断 + 一句内核原文」，
// 而不是契约要的「EOF，无帧」。
const (
	errorNoData           syscall.Errno = 232 // ERROR_NO_DATA: 管道正在关闭
	errorPipeNotConnected syscall.Errno = 233 // ERROR_PIPE_NOT_CONNECTED: 对端没有进程了
	errorOperationAborted syscall.Errno = 995 // ERROR_OPERATION_ABORTED: 操作被取消（断开时会看到）
)

// isPeerGone 报告这个读错误是不是「对端没了」（等价于 io.EOF）。
//
// 不把 ERROR_INVALID_HANDLE 收进来：那只会在我们自己 Close 与 Read 抢的时候出现，
// 而那时调用方早就拿到超时结果了 —— 把它算成「对端走了」会掩盖真正的句柄错误。
func isPeerGone(err error) bool {
	switch {
	case errors.Is(err, syscall.ERROR_BROKEN_PIPE),
		errors.Is(err, errorNoData),
		errors.Is(err, errorPipeNotConnected),
		errors.Is(err, errorOperationAborted):
		return true
	}
	return false
}

var procWaitNamedPipeW = syscall.NewLazyDLL("kernel32.dll").NewProc("WaitNamedPipeW")

// DefaultDialer 返回真管道上的 Dialer。
func DefaultDialer() Dialer { return pipeDialer{} }

type pipeDialer struct{}

func (pipeDialer) Dial(ctx context.Context, pipeName string) (Conn, error) {
	if pipeName == "" {
		return nil, &TransportError{Code: CodeServerDied, Msg: "管道名为空"}
	}
	path := `\\.\pipe\` + pipeName
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, &TransportError{Code: CodeServerDied, Msg: "管道名无法转成 UTF-16: " + pipeName, Err: err}
	}

	deadline := ctxDeadline(ctx, ConnectTimeout)
	for {
		h, err := syscall.CreateFile(p,
			syscall.GENERIC_READ|syscall.GENERIC_WRITE,
			0, nil, syscall.OPEN_EXISTING, 0, 0)
		if err == nil {
			f := os.NewFile(uintptr(h), path)
			if f == nil {
				syscall.CloseHandle(h)
				return nil, &TransportError{Code: CodeServerDied, Msg: "管道句柄无法包成 *os.File"}
			}
			return streamConn(f), nil
		}
		switch err {
		case syscall.ERROR_FILE_NOT_FOUND, syscall.ERROR_PATH_NOT_FOUND:
			// 没有服务端实例。**不要把它当成「等一下就有了」**：Ensure 靠这个错误
			// 判定「守护进程还没起来」，然后自己决定是轮询还是失败。
			return nil, &TransportError{Code: CodeServerDied,
				Msg: "没有守护进程在监听 " + pipeName, Err: err}
		case errorPipeBusy:
			// 忙：另一个客户端占着那唯一的实例。等一小段再试，预算由 ctx 决定。
			remain := time.Until(deadline)
			if remain <= 0 {
				return nil, &TransportError{Code: CodeServerDied,
					Msg: "管道 " + pipeName + " 一直忙（另一个客户端占着唯一的实例）"}
			}
			wait := remain
			if wait > 100*time.Millisecond {
				wait = 100 * time.Millisecond
			}
			procWaitNamedPipeW.Call(uintptr(unsafe.Pointer(p)), uintptr(wait.Milliseconds()))
			continue
		default:
			return nil, &TransportError{Code: CodeServerDied, Msg: "连不上管道 " + pipeName, Err: err}
		}
	}
}

// ctxDeadline 是「取 ctx 的截止时间和 now+d 中更早的那个」。
//
// 两个来源都要认：调用方给的 ctx 是他愿意等多久（一般是 300 ms），d 是这个函数
// 自己的上限（同一个 300 ms）。少了 ctx 那边，一个已经取消的调用会继续等满 d。
func ctxDeadline(ctx context.Context, d time.Duration) time.Time {
	dl := time.Now().Add(d)
	if t, ok := ctx.Deadline(); ok && t.Before(dl) {
		return t
	}
	return dl
}
