package tzs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

//---------------------------------------------------------------------------
// 假 Conn / 假 Dialer
//
// 只用来测**我们自己的**错误路径与判定（退出码、EOF、超时、解析）。
// 特意**不用**它们去断言「引擎会怎么答」—— 那是在把自己的假设测一遍：
// 帧的形状由契约（和 engine/src/Designer/Rpc.cs）定义，不由这里的假实现定义。
//---------------------------------------------------------------------------

type fakeConn struct {
	sent     [][]byte
	recv     [][]byte // 依次交给 Recv 的帧
	recvErr  error
	sendErr  error
	blockRec bool // Recv 一直阻塞（测读超时）
	closed   bool
}

func (c *fakeConn) Send(b []byte) error {
	if c.sendErr != nil {
		return c.sendErr
	}
	c.sent = append(c.sent, append([]byte(nil), b...))
	return nil
}

func (c *fakeConn) Recv() ([]byte, error) {
	if c.blockRec {
		select {} // 由测试的超时终结；进程退出时一起走
	}
	if len(c.recv) == 0 {
		if c.recvErr != nil {
			return nil, c.recvErr
		}
		return nil, io.EOF
	}
	f, err := c.recv[0], error(nil)
	c.recv = c.recv[1:]
	return f, err
}

func (c *fakeConn) Close() error { c.closed = true; return nil }

type fakeDialer struct {
	conn Conn
	err  error
	got  string // 记下被请求的管道名
}

func (d *fakeDialer) Dial(ctx context.Context, pipeName string) (Conn, error) {
	d.got = pipeName
	if d.err != nil {
		return nil, d.err
	}
	return d.conn, nil
}

//---------------------------------------------------------------------------
// 退出码映射表（契约表逐行）
//---------------------------------------------------------------------------

// TestExitCodeTable 是契约的退出码表本身。
//
// 顺序上的坑是 E_FATAL_LOAD_TIMEOUT：它的 kind 是 designer，照 kind 走会得到 4
// 「写入被拒」（=「表单的规则说不」），而真相是「这个包根本加载不完」。
// 所以 code 的判定必须在 kind 之前 —— 这几行就是钉它的。
func TestExitCodeTable(t *testing.T) {
	cases := []struct {
		name  string
		frame string
		want  int
	}{
		{"帧 ok:true", `{"id":1,"ok":true,"result":{},"ms":0.1}`, 0},
		{"kind=validation", `{"id":1,"ok":false,"error":{"code":"E_BAD_PARAM","kind":"validation","message":"x"}}`, 2},
		{"kind=not_found", `{"id":1,"ok":false,"error":{"code":"E_NOT_FOUND","kind":"not_found","message":"x"}}`, 2},
		{"kind=designer", `{"id":1,"ok":false,"error":{"code":"E_DESIGNER","kind":"designer","message":"x"}}`, 4},
		{"kind=designer 的 E_KEY_IN_USE", `{"id":1,"ok":false,"error":{"code":"E_KEY_IN_USE","kind":"designer","message":"x"}}`, 4},
		{"kind=internal", `{"id":1,"ok":false,"error":{"code":"E_INTERNAL","kind":"internal","message":"x"}}`, 1},
		{"E_NOT_IMPLEMENTED", `{"id":1,"ok":false,"error":{"code":"E_NOT_IMPLEMENTED","kind":"internal","message":"x"}}`, 1},
		// E_NO_OP 曾经由 set_local_string / set_spec_description 的 NoOp 分支发出（kind=internal）。
		// 2026-09-25 起引擎不再有那条路径，但表的这一行不动：这种帧真出现时该退 1，
		// 而**它不是「报 bug」**（见 IsSuccessCode）。
		{"E_NO_OP（错误帧，兜底）", `{"id":1,"ok":false,"error":{"code":"E_NO_OP","kind":"internal","message":"已经是这个内容"}}`, 1},
		{"E_ATTR_CLAMPED（错误帧，兜底）", `{"id":1,"ok":false,"error":{"code":"E_ATTR_CLAMPED","kind":"internal","message":"被吸附"}}`, 1},
		// code 在 kind 之前判的两行。
		{"E_FATAL_LOAD_TIMEOUT（kind 是 designer）", `{"id":7,"ok":false,"error":{"code":"E_FATAL_LOAD_TIMEOUT","kind":"designer","message":"超时"}}`, 5},
		{"帧里出现 E_SERVER_DIED", `{"id":1,"ok":false,"error":{"code":"E_SERVER_DIED","kind":"internal","message":"x"}}`, 5},
		// 兜底：缺少/陌生的 kind。
		{"kind 缺失", `{"id":1,"ok":false,"error":{"code":"E_WEIRD","message":"x"}}`, 1},
		{"kind 陌生", `{"id":1,"ok":false,"error":{"code":"E_WEIRD","kind":"??","message":"x"}}`, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := ParseReply([]byte(c.frame))
			if err != nil {
				t.Fatalf("帧该能解析: %v", err)
			}
			if got := ExitCode(r, nil); got != c.want {
				t.Errorf("退出码想要 %d，得 %d", c.want, got)
			}
		})
	}
}

// TestExitCodeTransportAndUsage 是表里的另外两栏：传输失败 → 5，用法错 → 2。
func TestExitCodeTransportAndUsage(t *testing.T) {
	if got := ExitCode(nil, &UsageError{Msg: "未知函数 foo"}); got != ExitUsage {
		t.Errorf("UsageError 该退 2，得 %d", got)
	}
	if got := ExitCode(nil, &TransportError{Code: CodeServerDied, Msg: "连不上"}); got != ExitTransport {
		t.Errorf("TransportError 该退 5，得 %d", got)
	}
	if got := ExitCode(nil, nil); got != ExitTransport {
		t.Errorf("既没有帧也没有错误时该退 5（不能静默退 0），得 %d", got)
	}
	if got := ExitCode(nil, errors.New("随便一个错")); got != ExitFrameErr {
		t.Errorf("未分类错误该退 1，得 %d", got)
	}
}

// TestSyntheticFailure 钉住"合成的帧"与错误本身的**退出码一致**。
//
// 这条等式是硬约束，理由是消费方只会读其中一个：脚本读退出码、agent 读帧，
// 两处不一致就是同一个失败有两种说法。按退出码挑 (code, kind) 让等式**构造上成立**，
// 这条测试负责证明它确实成立 —— 逐个错误类型验，不是抽查。
func TestSyntheticFailure(t *testing.T) {
	if SyntheticFailure(nil) != nil {
		t.Error("没有错误时不该合成出帧")
	}

	cases := []struct {
		name string
		err  error
		code string
		kind string
	}{
		{"传输失败", &TransportError{Code: CodeServerDied, Msg: "守护进程没有应答就退出了"}, CodeServerDied, KindInternal},
		{"用法错（含 detail 时也要能合成）", &UsageError{Msg: "未知函数 foo", Detail: []string{"是不是想写 bar"}}, CodeBadRequest, KindValidation},
		{"未分类的错", errors.New("随便一个错"), CodeInternal, KindInternal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := SyntheticFailure(c.err)
			if f == nil || f.Error == nil {
				t.Fatal("该合成出一个带 error 的帧")
			}
			if f.OK {
				t.Error("合成的帧必须是 ok:false")
			}
			if f.Error.Code != c.code || f.Error.Kind != c.kind {
				t.Errorf("想要 (%s, %s)，得 (%s, %s)", c.code, c.kind, f.Error.Code, f.Error.Kind)
			}
			if f.Error.Message == "" {
				t.Error("帧里要有 message —— 人读的就是它")
			}
			if f.ID != nil {
				t.Errorf("id 该留 null（帧级错误不回显 id），得 %s", f.ID)
			}

			// ① 帧算出来的退出码 == 错误自己的退出码
			fromFrame := ExitCode(f, nil)
			fromErr := ExitFrameErr
			var ec interface{ ExitCode() int }
			if errors.As(c.err, &ec) {
				fromErr = ec.ExitCode()
			}
			if fromFrame != fromErr {
				t.Errorf("帧说退出码 %d、错误自己是 %d —— 同一个失败两种说法", fromFrame, fromErr)
			}

			// ② 我们自己合成的帧，自己的解析器必须认得
			b, err := json.Marshal(f)
			if err != nil {
				t.Fatalf("合成帧序列化失败：%v", err)
			}
			r, err := ParseReply(b)
			if err != nil {
				t.Fatalf("自己合成的帧自己的解析器不认得（%s）：%v", b, err)
			}
			if r.OK || r.Error == nil || r.Error.Code != c.code {
				t.Errorf("往返之后帧变了：%s", b)
			}
		})
	}
}

// TestFailureKindHelpers 报告「哪两种 kind 是调用方能自纠的」。
// 用错这一条的表现是：一个自己打错的参数名被报成「内部错误，请报 bug」。
func TestFailureKindHelpers(t *testing.T) {
	for _, k := range []string{KindValidation, KindNotFound} {
		if frameExitCode("", k) != ExitUsage {
			t.Errorf("kind=%s 该退 2", k)
		}
	}
	if frameExitCode("", KindDesigner) != ExitDesigner {
		t.Errorf("kind=%s 该退 4", KindDesigner)
	}
	if frameExitCode("", KindInternal) != ExitFrameErr {
		t.Errorf("kind=%s 该退 1", KindInternal)
	}
	// E_NO_OP / E_ATTR_CLAMPED 在 ok:false 的帧里是「什么也没改」，不是「坏了」。
	// 引擎已经不发这种帧了（2026-09-25），这条断言守的是那句文案的判据本身。
	for _, code := range []string{CodeNoOp, CodeAttrClamped} {
		e := &WireError{Code: code, Kind: KindInternal, Message: "x"}
		if !e.IsSuccessCode() {
			t.Errorf("%s 该被认成成功码", code)
		}
	}
	if (&WireError{Code: "E_INTERNAL", Kind: KindInternal}).IsSuccessCode() {
		t.Errorf("E_INTERNAL 不是成功码")
	}
}

//---------------------------------------------------------------------------
// Call 的传输错误路径
//---------------------------------------------------------------------------

// TestCallDialFailure：连不上 → 传输失败（退 5）。
func TestCallDialFailure(t *testing.T) {
	d := &fakeDialer{err: errors.New("no pipe")}
	_, err := Call(context.Background(), d, "tzs-cli-00000000-00000000", 1, "list_open", nil, time.Second)
	if err == nil {
		t.Fatal("连不上该报错")
	}
	if ExitCode(nil, err) != ExitTransport {
		t.Errorf("该退 5，得 %d（%v）", ExitCode(nil, err), err)
	}
	if d.got == "" {
		t.Errorf("Dialer 该被调用")
	}
}

// TestCallEOFFrameLess：一帧都没有的 EOF → E_SERVER_DIED（退 5）。
//
// 这是「下一次命令会重启守护进程」的那一类；与「先来一帧 E_FATAL_LOAD_TIMEOUT 再 EOF」
// （确定性、绝不重试）必须分开。两者都退 5，区别在于**有没有帧**：有帧的那条路
// 会从 ParseReply 正常返回，根本不会走到这里。
func TestCallEOFFrameLess(t *testing.T) {
	conn := &fakeConn{} // recv 空 + 没有 recvErr → Recv 返回 io.EOF
	d := &fakeDialer{conn: conn}
	_, err := Call(context.Background(), d, "p", 1, "list_open", nil, time.Second)
	var te *TransportError
	if !errors.As(err, &te) {
		t.Fatalf("该是运输错误，得 %T（%v）", err, err)
	}
	if te.Code != CodeServerDied {
		t.Errorf("码该是 %s，得 %s", CodeServerDied, te.Code)
	}
	if !strings.Contains(te.Error(), "EOF") {
		t.Errorf("文案该说清是 EOF 无帧：%s", te.Error())
	}
	if !conn.closed {
		t.Errorf("连接该被关掉（否则后台的 Recv goroutine 会一直挂着）")
	}
}

// TestCallReadTimeout：守护进程不答 → 读超时（退 5），且**不重试**。
//
// 重试是危险的：请求可能已经上线并写进了设计器内存，重试就是在赌
// 「上一次写进去了没有」。所以这里断言只发了一次。
func TestCallReadTimeout(t *testing.T) {
	conn := &fakeConn{blockRec: true}
	d := &fakeDialer{conn: conn}
	start := time.Now()
	_, err := Call(context.Background(), d, "p", 1, "set_layout_attr", json.RawMessage(`{"value":"x"}`), 50*time.Millisecond)
	if err == nil {
		t.Fatal("读超时该报错")
	}
	if ExitCode(nil, err) != ExitTransport {
		t.Errorf("该退 5，得 %d", ExitCode(nil, err))
	}
	if !strings.Contains(err.Error(), "读超时") {
		t.Errorf("文案该说清是读超时：%v", err)
	}
	if len(conn.sent) != 1 {
		t.Errorf("请求只该发一次（无重试），实际 %d 次", len(conn.sent))
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("超时该很快返回，实际等了 %v", elapsed)
	}
}

// TestCallSendFailure：发不出去 → 传输失败，且不去等应答。
func TestCallSendFailure(t *testing.T) {
	conn := &fakeConn{sendErr: errors.New("broken pipe")}
	d := &fakeDialer{conn: conn}
	_, err := Call(context.Background(), d, "p", 1, "list_open", nil, time.Second)
	if ExitCode(nil, err) != ExitTransport {
		t.Errorf("该退 5，得 %d（%v）", ExitCode(nil, err), err)
	}
}

// TestCallNonJSONReply：应答不是 JSON 对象 → 传输失败（退 5），
// **绝不**报成一个关于表单的业务错误。
func TestCallNonJSONReply(t *testing.T) {
	for _, frame := range []string{"tzs-server: 引导中", "<html>", "{\"oops\":1}"} {
		conn := &fakeConn{recv: [][]byte{[]byte(frame)}}
		d := &fakeDialer{conn: conn}
		_, err := Call(context.Background(), d, "p", 1, "list_open", nil, time.Second)
		if err == nil {
			t.Fatalf("帧 %q 该被当成传输失败", frame)
		}
		if ExitCode(nil, err) != ExitTransport {
			t.Errorf("帧 %q 该退 5，得 %d", frame, ExitCode(nil, err))
		}
	}
}

// TestCallFatalFrameThenEOF：致命帧之后守护进程就 exit(3) 了；我们只读一帧，
// 所以拿到的是那个帧（而不是 EOF），退出码 5，**不重试**。
func TestCallFatalFrameThenEOF(t *testing.T) {
	fatal := `{"id":5,"ok":false,"error":{"code":"E_FATAL_LOAD_TIMEOUT","kind":"designer","message":"加载超时 90s","detail":{"path":"big.tzs","timeoutSeconds":90}}}`
	conn := &fakeConn{recv: [][]byte{[]byte(fatal)}}
	d := &fakeDialer{conn: conn}
	r, err := Call(context.Background(), d, "p", 5, "open", json.RawMessage(`{"path":"big.tzs"}`), time.Second)
	if err != nil {
		t.Fatalf("这是一帧正常应答，不该是 error：%v", err)
	}
	if r.OK {
		t.Fatal("致命帧是 ok:false")
	}
	if got := ExitCode(r, nil); got != ExitTransport {
		t.Errorf("该退 5，得 %d", got)
	}
	// 两次都不许重试：帧已经说明加载是确定性的失败，重试只会再挂一次。
	if len(conn.sent) != 1 {
		t.Errorf("只该发一次，实际 %d 次", len(conn.sent))
	}
	if string(r.ID) != "5" {
		t.Errorf("id 该被回显成 5，得 %q", r.ID)
	}
}

//---------------------------------------------------------------------------
// 请求帧的形状
//---------------------------------------------------------------------------

// TestMarshalRequest 钉住请求帧的字节形状：{"id":n,"fn":"...","args":{...}} + 无行尾。
//
// 形状是契约（引擎 Rpc.Request 逐字生成它），不是「我们觉得这样好看」。
func TestMarshalRequest(t *testing.T) {
	b, err := MarshalRequest(1, "list_open", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"id":1,"fn":"list_open","args":{}}` {
		t.Errorf("空 args 该补成 {}：%s", b)
	}
	b, _ = MarshalRequest(7, "set_layout_attr", json.RawMessage(`{"value":"x"}`))
	if string(b) != `{"id":7,"fn":"set_layout_attr","args":{"value":"x"}}` {
		t.Errorf("args 该被逐字节拼进去：%s", b)
	}
	// 帧里不许有换行（WriteFrame 会拒），所以这里也就不会有。
	if bytes.ContainsAny(b, "\r\n") {
		t.Errorf("帧里出现了换行：%q", b)
	}
	// HTML 不转义：`--value "</section>"` 在日志里该能照着读。
	b, _ = MarshalRequest(1, "set_spec_description", json.RawMessage(`{"content":"</section>"}`))
	if !bytes.Contains(b, []byte(`</section>`)) {
		t.Errorf("< 被转义了：%s", b)
	}
}

// TestCallSendsExactlyOneRequest：一整帧发出去、只发一次，并且带结尾 \n。
func TestCallSendsExactlyOneRequest(t *testing.T) {
	conn := &fakeConn{recv: [][]byte{[]byte(`{"id":1,"ok":true,"result":[],"ms":0.1}`)}}
	d := &fakeDialer{conn: conn}
	r, err := Call(context.Background(), d, "p", 1, "list_open", nil, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if !r.OK {
		t.Fatal("该是 ok")
	}
	if len(conn.sent) != 1 {
		t.Fatalf("只该发一帧，实际 %d", len(conn.sent))
	}
	if string(conn.sent[0]) != "{\"id\":1,\"fn\":\"list_open\",\"args\":{}}\n" {
		t.Errorf("发出去的帧不对：%q", conn.sent[0])
	}
}

// TestNormalizeTimeout：`--timeout` 的规范化的三条规则。
//
// 下限 120 s 的唯一理由是引擎内部 90 s 的加载看门狗（TZSCLI_RELOAD_TIMEOUT）：
// 允许更小就等于允许用户把每个慢包都判成「守护进程死了」，然后反复重建守护进程。
func TestNormalizeTimeout(t *testing.T) {
	cases := []struct {
		in, want time.Duration
	}{
		{0, DefaultTimeout},
		{-time.Second, DefaultTimeout},
		{5 * time.Second, MinTimeout},
		{MinTimeout - 1, MinTimeout},
		{MinTimeout, MinTimeout},
		{10 * time.Minute, 10 * time.Minute},
	}
	for _, c := range cases {
		if got := NormalizeTimeout(c.in); got != c.want {
			t.Errorf("NormalizeTimeout(%v) = %v，想要 %v", c.in, got, c.want)
		}
	}
}

// TestStreamConnRoundTrip 走一遍真正的「流 → 帧」路径（用 bytes.Buffer 当管道），
// 确认 streamConn 的 Send/Recv 与 wire.go 的帧规则是同一条。
func TestStreamConnRoundTrip(t *testing.T) {
	var wire bytes.Buffer
	c := streamConn(nopCloser{&wire})
	if err := c.Send([]byte("{\"id\":1,\"fn\":\"list_open\",\"args\":{}}\n")); err != nil {
		t.Fatal(err)
	}
	if wire.String() != "{\"id\":1,\"fn\":\"list_open\",\"args\":{}}\n" {
		t.Fatalf("写出来的字节不对：%q", wire.String())
	}
	// 回来的那一侧带 \r\n 与 BOM，这是引擎/stdio 混合流的样子。
	r := streamConn(nopCloser{bytes.NewBufferString(bomString + "{\"ok\":true}\r\n")})
	got, err := r.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"ok":true}` {
		t.Errorf("读回的帧不对：%q", got)
	}
}

// TestStreamConnSendRequiresCompleteFrame：Conn.Send 收的是**完整帧**（含结尾 \n）。
//
// 少了那个换行必须当场断掉：发出去的话，对端会把这一帧的尾巴和下一帧的头并成一行读，
// 之后每一次交互都错位一格（引擎那侧回的是 E_BAD_REQUEST / 未知函数），
// 而症状看起来与「我忘了换行」毫无关系。
func TestStreamConnSendRequiresCompleteFrame(t *testing.T) {
	var wire bytes.Buffer
	c := streamConn(nopCloser{&wire})
	if err := c.Send([]byte(`{"id":1}`)); err == nil {
		t.Errorf("少结尾 \\n 的帧该被拒")
	}
	if wire.Len() != 0 {
		t.Errorf("被拒的帧不该写出去任何字节：%q", wire.String())
	}
	// 正文里夹一个换行同样该被拒（那会把一帧劈成两帧）。
	if err := c.Send([]byte("{\"a\":1}\n{\"b\":2}\n")); err == nil {
		t.Errorf("正文里夹换行的帧该被拒")
	}
	if wire.Len() != 0 {
		t.Errorf("被拒的帧不该写出去任何字节：%q", wire.String())
	}
}

type nopCloser struct{ io.ReadWriter }

func (nopCloser) Close() error { return nil }
