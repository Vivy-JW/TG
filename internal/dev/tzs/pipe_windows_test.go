//go:build windows

package tzs

// 真命名管道的往返测试。
//
// 这是**进程内**的（不 exec 任何二进制）：服务端那一半用 LoadLibrary 的
// CreateNamedPipeW 在本进程里现搭，客户端那一半就是我们要验的 DefaultDialer。
// 它证明的是「我们的 Windows 传输能把一帧发出去、把一帧读回来」，
// 与「引擎会怎么答」无关 —— 后者的证据只能来自 TTZS_E2E=1。
//
// 为什么值得写：连管道这一层的错法是**静默**的。比如把 os.NewFile 换成直接
// syscall.Read，或者给别人一个 SetReadDeadline（命名管道上它返回
// "file type does not support deadline"），都不会编译失败，只会在真机上以
// 「冷启动 60 s 未就绪」的形式出现。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

const (
	pipeAccessDuplex = 0x00000003
	pipeUnlimited    = 255
)

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procCreateNamedPipeW    = kernel32.NewProc("CreateNamedPipeW")
	procConnectNamedPipe    = kernel32.NewProc("ConnectNamedPipe")
	procDisconnectNamedPipe = kernel32.NewProc("DisconnectNamedPipe")
)

// pipeServer 是本进程内的假服务端：接一个客户端、读一行、回一段字节。
type pipeServer struct {
	name  string
	h     syscall.Handle
	once  sync.Once
	done  chan struct{}
	got   string // 读到的请求原文
	reply string
}

func startPipeServer(t *testing.T, name, reply string) *pipeServer {
	t.Helper()
	path, err := syscall.UTF16PtrFromString(`\\.\pipe\` + name)
	if err != nil {
		t.Fatal(err)
	}
	h, _, e := procCreateNamedPipeW.Call(uintptr(unsafe.Pointer(path)),
		pipeAccessDuplex, 0, pipeUnlimited, 8192, 8192, 0, 0)
	if syscall.Handle(h) == syscall.InvalidHandle {
		t.Fatalf("CreateNamedPipeW(%s) 失败: %v", name, e)
	}
	s := &pipeServer{name: name, h: syscall.Handle(h), done: make(chan struct{}), reply: reply}
	t.Cleanup(func() { s.close() })

	go func() {
		defer close(s.done)
		procConnectNamedPipe.Call(h, 0)
		var acc []byte
		buf := make([]byte, 4096)
		for !bytes.Contains(acc, []byte("\n")) {
			var n uint32
			if err := syscall.ReadFile(s.h, buf, &n, nil); err != nil || n == 0 {
				break
			}
			acc = append(acc, buf[:n]...)
		}
		s.got = string(acc)
		if s.reply == "" {
			// 一声不响地断开：**必须真的关掉句柄**。只让这个 goroutine return 的话，
			// 客户端的 Read 会一直挂着、最后走到读超时 —— 于是测试名义上测的是
			// 「EOF 无帧」，实际测的是「超时」，而两者是不同的失败（虽然都退 5）。
			// 客户端会在 ReadFile 上拿到 ERROR_BROKEN_PIPE，Go 的 poll 层把它映射成 io.EOF。
			s.close()
			return
		}
		w := []byte(s.reply)
		var wn uint32
		_ = syscall.WriteFile(s.h, w, &wn, nil)
	}()
	return s
}

func (s *pipeServer) close() {
	s.once.Do(func() {
		procDisconnectNamedPipe.Call(uintptr(s.h))
		syscall.CloseHandle(s.h)
	})
}

// wait 等服务端那一半收工（超时就失败，免得测试挂住）。
func (s *pipeServer) wait(t *testing.T) {
	t.Helper()
	select {
	case <-s.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("假服务端没在 5 s 内收工")
	}
}

func testPipeName() string {
	return fmt.Sprintf("tzs-cli-%08x-%08x", uint32(os.Getpid()), 0x11c46f9d)
}

// TestDefaultDialerRealNamedPipe 是最核心的一条：真的连上、真的发一帧、真的收一帧。
func TestDefaultDialerRealNamedPipe(t *testing.T) {
	name := testPipeName()
	srv := startPipeServer(t, name, `{"id":1,"ok":true,"result":{"handle":"h1"},"ms":0.25}`+"\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := DefaultDialer().Dial(ctx, name)
	if err != nil {
		t.Fatalf("连本进程刚建的管道失败: %v", err)
	}
	defer conn.Close()

	if err := conn.Send([]byte("{\"id\":1,\"fn\":\"open\",\"args\":{\"path\":\"x.tzs\"}}\n")); err != nil {
		t.Fatalf("发帧失败: %v", err)
	}
	frame, err := conn.Recv()
	if err != nil {
		t.Fatalf("收帧失败: %v", err)
	}
	r, err := ParseReply(frame)
	if err != nil {
		t.Fatalf("收到的帧解不开: %v（原文 %q）", err, frame)
	}
	if !r.OK {
		t.Fatalf("该是 ok:true：%+v", r)
	}
	if !bytes.Contains(r.Result, []byte(`"handle":"h1"`)) {
		t.Errorf("result 没解对：%s", r.Result)
	}

	srv.wait(t)
	// 出方向：一整帧 + 一个 \n，不多不少。
	if srv.got != "{\"id\":1,\"fn\":\"open\",\"args\":{\"path\":\"x.tzs\"}}\n" {
		t.Errorf("服务端收到的是 %q", srv.got)
	}
}

// TestDefaultDialerToleratesCRLFReply：真管道上回来的 CRLF 也要能剥掉
// （引擎的 LineReader 在写的时候只发 \n，但 stdio 混合流上 \r 是常见污染）。
func TestDefaultDialerToleratesCRLFReply(t *testing.T) {
	name := testPipeName()
	srv := startPipeServer(t, name, "{\"id\":1,\"ok\":true,\"result\":{},\"ms\":0.1}\r\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := DefaultDialer().Dial(ctx, name)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.Send([]byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	frame, err := conn.Recv()
	if err != nil {
		t.Fatalf("CRLF 的应答该能读：%v", err)
	}
	if strings.HasSuffix(string(frame), "\r") {
		t.Errorf("\\r 没被剥掉：%q", frame)
	}
	if r, err := ParseReply(frame); err != nil || !r.OK {
		t.Errorf("该是 ok 帧：%v %+v", err, r)
	}
	srv.wait(t)
}

// TestDialNoSuchPipe：没有服务端实例时**当场**失败（不是等满超时）。
//
// 这个区分是 Ensure 的整个轮询逻辑依赖的：它靠「连不上」判定守护进程还没起来，
// 要是连不上要等 300 ms，冷启动的 240 次轮询就变成 72 秒。
func TestDialNoSuchPipe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	start := time.Now()
	_, err := DefaultDialer().Dial(ctx, "tzs-cli-ffffffff-ffffffff")
	if err == nil {
		t.Fatal("不该连上")
	}
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("该是传输错误，得 %T：%v", err, err)
	}
	if te.ExitCode() != ExitTransport {
		t.Errorf("该退 5，得 %d", te.ExitCode())
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("该立刻失败，实际等了 %v", elapsed)
	}
}

// TestCallOverRealPipeWithEOF：服务端读到请求后一声不响地断开 →
// 我们收到「有请求发出去、没有任何帧」→ E_SERVER_DIED（退 5）。
//
// 这是「传输失败」里最要紧的一类：重试它是在赌「上一次写进去了没有」。
func TestCallOverRealPipeWithEOF(t *testing.T) {
	name := testPipeName()
	srv := startPipeServer(t, name, "") // 空 reply = 读到请求后直接关掉

	start := time.Now()
	r, err := Call(context.Background(), DefaultDialer(), name, 1, "set_layout_attr",
		[]byte(`{"value":"x"}`), 3*time.Second)
	if err == nil {
		t.Fatalf("该报传输失败，却得到 %+v", r)
	}
	if ExitCode(nil, err) != ExitTransport {
		t.Errorf("该退 5，得 %d（%v）", ExitCode(nil, err), err)
	}
	var te *TransportError
	if !errors.As(err, &te) || te.Code != CodeServerDied {
		t.Errorf("该是 %s：%v", CodeServerDied, err)
	}
	// 必须是**看到了 EOF**，不是等满了 3 s 的读超时：两者文案不同
	// （一个是「EOF，无帧」，一个是「读超时」），而只有前者说明
	// 「守护进程走了」这件事真的被判断出来了。
	if !strings.Contains(err.Error(), "EOF") {
		t.Errorf("该报 EOF 无帧，得 %v", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("该立刻看到 EOF，实际等了 %v（说明走到读超时那条路了）", elapsed)
	}
	srv.wait(t)
	// 请求确实发出去了（只是没回来）—— 这正是「不能重试」的理由。
	if !strings.Contains(srv.got, "set_layout_attr") {
		t.Errorf("服务端该收到请求：%q", srv.got)
	}
}

// TestCallOverRealPipeSuccess 把 Call 的整条路径在真管道上跑一遍：
// Dial → MarshalRequest → Send → Recv → ParseReply → 退出码 0。
func TestCallOverRealPipeSuccess(t *testing.T) {
	name := testPipeName()
	srv := startPipeServer(t, name,
		`{"id":1,"ok":true,"result":[{"handle":"h1","program":"aapp320"}],"ms":0.1}`+"\n")

	r, err := Call(context.Background(), DefaultDialer(), name, 1, "list_open", nil, 3*time.Second)
	if err != nil {
		t.Fatalf("整条路径该成功：%v", err)
	}
	if got := ExitCode(r, nil); got != ExitOK {
		t.Errorf("该退 0，得 %d", got)
	}
	srv.wait(t)
	if srv.got != "{\"id\":1,\"fn\":\"list_open\",\"args\":{}}\n" {
		t.Errorf("发出去的帧不对：%q", srv.got)
	}
}
