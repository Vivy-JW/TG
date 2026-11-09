package tzs

// stdio_test.go —— 驱动 `tzs-server --stdio` 的测试专用 harness。
//
// 为什么不直接用 client.go：**容错面是反的**。
//
// 出货客户端（client.go / wire.go）的规则是「不是 JSON 对象就 E_SERVER_DIED」—— 因为那边
// 一行读错就意味着帧界错了位，后面的每一帧都不可信，宁可判死也不能拿一个可疑的应答去改
// 设计器的模型。而这里我们只关心「我要的那一帧回没回来」，多出来的诊断行丢掉即可。
//
// 为什么 --stdio 上真的会有诊断行混进来：`Rpc.Stdio()` 只做 `Console.SetOut(Console.Error)`，
// 它拦得住 `Console.Write*`，拦不住引擎里直接写 `Console.OpenStandardOutput()` 的路径
// （Json.cs 里 info/spec 两个转储就是这么写的 —— 那两处会往 stdout 吐多行 JSON）。
// 实测污染率是 0（7 个包 28 帧一行没丢），但结构上必须防：一次设计器的诊断输出只该让某一行
// 对不上，不该把整轮语料回归判成「守护进程死了」。
//
// 这条宽松**绝不能带回 client.go**。两条规则各自有各自的理由（那边是「帧界错了就不能用」，
// 这边是「多几行不改变我要的那一帧」），合并任何一边都是错的。
//
// 为什么语料回归走 --stdio 而不是真管道：它是引擎自己指定的测试模式（Rpc.Stdio 的注释），
// 而且把「管道名里混着 TzsCli.Designer.dll 的 MVID」这个最高风险的假设从语料回归里摘了出去
// —— 那条路留给 chain_test.go（它就是对「真管道」本身的验收）。

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"tt/internal/testenv"
)

// stdioSession 是一个 `tzs-server --stdio` 进程。
//
// 一个进程一次只跑一个包（或一个包的一段）：请求与应答严格一进一出，所以 nextID 不用加锁 ——
// 调用它的只有测试那一根 goroutine。
type stdioSession struct {
	t      *testing.T
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	br     *bufio.Reader
	stderr *syncBuffer

	budget time.Duration
	timer  *time.Timer

	mu     sync.Mutex
	killed bool

	nextID  int
	sent    int
	dropped int
}

// startStdio 起一个 --stdio 进程并把 TZSCLI_WS 钉死在 ws 上。
//
// 预算用 AfterFunc 而不是 context：命名管道之外这里是一条普通的 os.Pipe，读它会**一直阻塞**，
// 而 Go 的 exec 没有「中途取消」这一说。到期直接 Kill 进程，卡住的读随即以 EOF 返回 ——
// 这正是旧实现那条教训要的东西：「a package that pops a modal dialog and hangs costs us
// its timeout and nothing more」。
func startStdio(t *testing.T, exe, ws string, budget time.Duration) *stdioSession {
	t.Helper()
	cmd := exec.Command(exe, "--stdio")
	cmd.Env = engineEnv(ws)
	// 引擎自己的相对路径按 cwd 解析（旧实现也是 cwd=BIN）。
	cmd.Dir = filepath.Dir(exe)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin 管道建不起来: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout 管道建不起来: %v", err)
	}
	errb := &syncBuffer{}
	cmd.Stderr = errb
	if err := cmd.Start(); err != nil {
		t.Fatalf("起不了 %s --stdio: %v", exe, err)
	}
	s := &stdioSession{
		t: t, cmd: cmd, stdin: stdin,
		br:     bufio.NewReaderSize(stdout, 256*1024),
		stderr: errb, budget: budget,
	}
	s.timer = time.AfterFunc(budget, s.kill)
	return s
}

// engineEnv 给子进程一份显式环境。
//
// TZSCLI_WS 由我们钉死而不是继承：环境里一个别的 TZSCLI_WS 会把整轮语料操作到**另一个工作区**上，
// 而 TzpManager 会因此拒绝每一个包 —— 报出来的是「打开失败」，与真因（工作区不对）无关。
// 同一个键出现两次时 exec 取最后一个（os/exec 的既定行为），所以追加就是覆盖。
// TZSCLI_INSTALL 只在给了 TTZS_INSTALL 时才覆盖：绝不能拿空串把引擎的内置缺省打掉。
//
// TZSCLI_QUIET 关掉 RoundTrip.exe 的明细输出（我们只要那一行 SUMMARY|；不打它会让几十万行
// 日志淹掉 -v 的输出）。
func engineEnv(ws string) []string {
	env := append(os.Environ(), "TZSCLI_WS="+ws, "TZSCLI_QUIET=1")
	if d := testenv.DesignerDir(); d != "" {
		env = append(env, "TZSCLI_INSTALL="+d)
	}
	return env
}

func (s *stdioSession) kill() {
	s.mu.Lock()
	s.killed = true
	s.mu.Unlock()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}

func (s *stdioSession) wasKilled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.killed
}

// call 发一帧请求并等它那一帧应答。
//
// 严格一进一出（发了才读、读到才发下一帧）而不是像 gate-w3.py 那样把整批 payload 先写完：
// 引擎的 Serve 是一行一请求一应答的循环，所以逐帧驱动是它支持的形态，而它买到的东西很实在
// —— 后面每一帧都能用**前面真实拿到的**句柄，不必假设「第一个 open 一定回 h1」。
// （旧实现只能写死 h1，因为它是先把整批请求写出去的。）
func (s *stdioSession) call(fn string, args map[string]any) (*Reply, error) {
	raw := json.RawMessage("{}")
	if args != nil {
		b, err := json.Marshal(args)
		if err != nil {
			return nil, fmt.Errorf("用例给的参数不是合法 JSON: %w", err)
		}
		raw = b
	}
	id := s.nextID
	s.nextID++
	req, err := MarshalRequest(id, fn, raw)
	if err != nil {
		return nil, err
	}
	frame, err := CompleteFrame(req)
	if err != nil {
		return nil, err
	}
	if _, err := s.stdin.Write(frame); err != nil {
		return nil, s.transport("写请求失败（"+fn+"）", err)
	}
	s.sent++
	return s.readReply(id)
}

// readReply 读到 id 匹配的那一帧为止，中途**丢掉**不像应答的行（见文件头注释）。
func (s *stdioSession) readReply(id int) (*Reply, error) {
	want := strconv.Itoa(id)
	for {
		body, err := ReadFrame(s.br)
		if err != nil {
			return nil, s.transport("读应答失败", err)
		}
		r, perr := ParseReply(body)
		if perr != nil {
			// 不是本协议的帧（设计器的诊断输出混进了这条流）。丢掉继续读。
			s.dropped++
			continue
		}
		if string(r.ID) != want {
			// 别的 id。一进一出之下不该发生；发生了说明流上多了东西，同样丢掉。
			s.dropped++
			continue
		}
		return r, nil
	}
}

// transport 把一次传输失败说成人话：超时被杀是最常见的一种，它的真因在 stderr 里。
func (s *stdioSession) transport(msg string, err error) error {
	why := err.Error()
	if s.wasKilled() {
		why = fmt.Sprintf("超过每包预算 %s，进程已被杀（典型原因：这个包弹了一个模态对话框，卡住了）", s.budget)
	} else if errors.Is(err, io.EOF) {
		why = "进程在应答之前就没了（EOF 无帧）"
	}
	return fmt.Errorf("%s: %s；stderr 末尾：\n%s", msg, why, tailOf(s.stderr.String(), 800))
}

// close 收摊：关 stdin（引擎读到 EOF 自己退）→ 等一会儿 → 还没走就杀。
func (s *stdioSession) close() {
	s.timer.Stop()
	_ = s.stdin.Close()
	done := make(chan struct{})
	go func() { _ = s.cmd.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(15 * time.Second):
		s.kill()
		<-done
	}
}

//---------------------------------------------------------------------------
// 断言小工具
//---------------------------------------------------------------------------

// replyResult 解一帧的 result。帧不是 ok:true 时 t.Errorf 并返回 false。
//
// 每一处都用同一个入口，是为了让「引擎拒绝了这次调用」在报告里长得一样：带 label（哪个包、
// 哪一步）、带 code/kind、带 message 与 detail。分开写的后果是某一处只报一句“失败了”。
func replyResult[T any](t *testing.T, label string, r *Reply, err error) (T, bool) {
	t.Helper()
	var v T
	if err != nil {
		t.Errorf("%s: %v", label, err)
		return v, false
	}
	if r == nil {
		t.Errorf("%s: 没有应答（nil 帧）", label)
		return v, false
	}
	if !r.OK {
		b, _ := json.Marshal(r.Error)
		t.Errorf("%s: 引擎拒绝: %s", label, b)
		return v, false
	}
	if len(r.Result) > 0 {
		if e := json.Unmarshal(r.Result, &v); e != nil {
			t.Errorf("%s: result 解不开: %v（%s）", label, e, clip(r.Result))
			return v, false
		}
	}
	return v, true
}

// ask 发一次调用并把 result 解进 T —— 「发」与「解」合成一句，调用点于是只有一个形状。
func ask[T any](t *testing.T, s *stdioSession, label, fn string, args map[string]any) (T, bool) {
	t.Helper()
	r, err := s.call(fn, args)
	return replyResult[T](t, label, r, err)
}

//---------------------------------------------------------------------------

// syncBuffer 是给 cmd.Stderr 用的并发安全缓冲。
//
// 不能直接用 bytes.Buffer：exec 会在自己的 goroutine 里往它写，而我们在主 goroutine 里读
// （超时时要把 stderr 末尾贴进报错）。那是一次数据竞争，`-race` 会当场抓住。
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// tailOf 取一段文本的末尾（报错时给出线索，但不把整篇日志拖进报告）。
func tailOf(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
