package tzs

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

// bomString 是 UTF-8 的 BOM。用包里的字节常量拼，源码里不出字面 BOM —— Go 的
// 词法分析器碰到源码里出现的字面 BOM 会直接报 illegal byte order mark，连编译都过不去。
var bomString = string(utf8BOM)

// readFrameFrom 是 ReadFrame 的测试入口：把一段字节当成已经连好的流。
func readFrameFrom(s string) ([]byte, error) {
	return ReadFrame(bufio.NewReaderSize(strings.NewReader(s), 64*1024))
}

// TestReadFrameCRLF：引擎的 LineReader.Finish 会剥掉一个行尾 \r，
// 我们两个方向必须同样容忍 —— 只认 \n 会把一个 CRLF 的应答读成 `{"ok":true}\r`，
// 而这个尾巴会让 json.Unmarshal 报「invalid character '\r'」，
// 读起来像「引擎回了坏 JSON」，其实是行尾约定不同。
func TestReadFrameCRLF(t *testing.T) {
	got, err := readFrameFrom("{\"ok\":true}\r\n")
	if err != nil {
		t.Fatalf("CRLF 帧读失败: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Errorf("CRLF 没被剥掉: %q", got)
	}
}

// TestReadFrameBOM：行首 BOM（记事本一类工具的默认输出）要能容忍。
func TestReadFrameBOM(t *testing.T) {
	got, err := readFrameFrom(bomString + "{\"ok\":true}\n")
	if err != nil {
		t.Fatalf("BOM 帧读失败: %v", err)
	}
	if string(got) != `{"ok":true}` {
		t.Errorf("BOM 没被剥掉: %q", got)
	}
	// 只剥一个：两个 BOM 的第二个是内容的一部分（引擎也只剥一个）。
	two, _ := readFrameFrom(bomString + bomString + "{}\n")
	if !bytes.HasPrefix(two, []byte(bomString)) {
		t.Errorf("第二个 BOM 不该被剥掉: %q", two)
	}
}

// TestReadFrameEOFWithoutFrame：一帧都没有的 EOF 必须是 io.EOF，
// 而且是**可判别的** —— CallConn 靠它合成 E_SERVER_DIED（退 5）。
// 半帧 + EOF 同样是 io.EOF：引擎不会被半个帧救回来。
func TestReadFrameEOFWithoutFrame(t *testing.T) {
	for _, s := range []string{"", `{"ok":tr`, "\r"} {
		_, err := readFrameFrom(s)
		if !errors.Is(err, io.EOF) {
			t.Errorf("输入 %q：想 io.EOF，得 %v", s, err)
		}
	}
}

// TestReadFrameLimit：上限是 8 MiB，判据是 `> MaxLine`（与引擎的 LineReader 同口径）。
//
// 恰好 MaxLine 必须**能过**：有一条 form_tree 正好 8 MiB 时被判死的话，
// 现象是「大表单读不出来」，而引擎那侧认为它没问题。
func TestReadFrameLimit(t *testing.T) {
	ok := strings.Repeat("a", MaxLine)
	if _, err := readFrameFrom(ok + "\n"); err != nil {
		t.Fatalf("恰好 %d 字节的帧该能读，得 %v", MaxLine, err)
	}
	over := strings.Repeat("a", MaxLine+1)
	_, err := readFrameFrom(over + "\n")
	if err == nil {
		t.Fatalf("%d 字节的帧该被拒", MaxLine+1)
	}
	var te *TransportError
	if !errors.As(err, &te) || te.ExitCode() != ExitTransport {
		t.Errorf("超限该是传输失败（5），得 %v（退出码 %d）", err, ExitCode(nil, err))
	}
}

// TestWriteFrameRejectsRawNewline：帧里有裸换行会把一帧劈成两帧。
//
// 这条闸门必须存在：真帧全部由 encoding/json 产出（换行必转义），所以它只会逮住
// 手搓的帧；而没有它，手搓的那一帧之后的每一次交互都错位一格，
// 表现是「参数明明对，引擎说未知函数」。
func TestWriteFrameRejectsRawNewline(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte("{\"a\":\n1}")); err == nil {
		t.Errorf("含裸 \\n 的帧该被拒")
	}
	if err := WriteFrame(&buf, []byte("{\"a\":1}\r")); err == nil {
		t.Errorf("含裸 \\r 的帧该被拒")
	}
	if buf.Len() != 0 {
		t.Errorf("被拒的帧不该写出去任何字节，实际写了 %q", buf.String())
	}
}

// TestWriteFrameAppendsLFOnly：出方向只发 \n（不多一个 \r）。
func TestWriteFrameAppendsLFOnly(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte("{}")); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{}\n" {
		t.Errorf("想要 %q，得 %q", "{}\n", buf.String())
	}
}

// TestParseReplyNullIDAndMissingMs 钉住两件实测事实：
//
//	id:null  —— 帧级错误（E_BAD_REQUEST / E_UNKNOWN_METHOD）的 id 就是 null，
//	            反序列化不能因此崩，也不能把 null 当成 0。
//	没有 ms、没有 result —— WriteFatal 的致命帧就是这种形状。
//	            ms 用 float64 写会把「没有 ms」和「0.0 ms」混为一谈。
func TestParseReplyNullIDAndMissingMs(t *testing.T) {
	frame := `{"id":null,"ok":false,"error":{"code":"E_BAD_REQUEST","kind":"validation","message":"空行不是请求"}}`
	r, err := ParseReply([]byte(frame))
	if err != nil {
		t.Fatalf("id:null 的帧该能解析: %v", err)
	}
	if string(r.ID) != "null" {
		t.Errorf("id 该原样是 null，得 %q", r.ID)
	}
	if r.Ms != nil {
		t.Errorf("这一帧没有 ms，Ms 该是 nil，得 %v", *r.Ms)
	}
	if r.Result != nil {
		t.Errorf("这一帧没有 result，Result 该是 nil，得 %q", r.Result)
	}
	if r.Error == nil || r.Error.Code != "E_BAD_REQUEST" || r.Error.Kind != KindValidation {
		t.Errorf("error 没解对：%+v", r.Error)
	}

	fatal := `{"id":7,"ok":false,"error":{"code":"E_FATAL_LOAD_TIMEOUT","kind":"designer","message":"加载超时 90s","detail":{"path":"x.tzs","timeoutSeconds":90}}}`
	r2, err := ParseReply([]byte(fatal))
	if err != nil {
		t.Fatalf("致命帧该能解析: %v", err)
	}
	if r2.Ms != nil || r2.Result != nil {
		t.Errorf("致命帧没有 ms/result：%v %q", r2.Ms, r2.Result)
	}
	if r2.Error.ExitCode() != ExitTransport {
		t.Errorf("致命帧该退 5，得 %d", r2.Error.ExitCode())
	}
}

// TestParseReplyNotAFrame：四类「不是帧」的输入，一律是传输失败（退 5）。
//
// 绝不能让它们变成「某个业务错误」：一句解析错误读起来像一条关于表单的事实，
// 会让人断定「这个元素不存在」—— 这是引擎 Rpc.Classify 的注释里最强调的一条。
func TestParseReplyNotAFrame(t *testing.T) {
	cases := []struct {
		name  string
		frame string
	}{
		{"数字", "123"},
		{"数组", `[{"ok":true}]`},
		{"null", "null"},
		{"字符串", `"hello"`},
		{"不是 JSON", "tzs-server: 引导中"},
		{"没有 ok 字段", `{"id":1,"result":{}}`},
		{"ok 不是布尔", `{"id":1,"ok":"yes"}`},
		{"ok:false 但没有 error", `{"id":1,"ok":false}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := ParseReply([]byte(c.frame))
			if err == nil {
				t.Fatalf("该被当成传输失败，却解出了 %+v", r)
			}
			if r != nil {
				t.Errorf("失败时 reply 该是 nil")
			}
			var te *TransportError
			if !errors.As(err, &te) {
				t.Fatalf("该是 *TransportError，得 %T", err)
			}
			if te.Code != CodeServerDied {
				t.Errorf("码该是 %s，得 %s", CodeServerDied, te.Code)
			}
			if ExitCode(nil, err) != ExitTransport {
				t.Errorf("退出码该是 5，得 %d", ExitCode(nil, err))
			}
		})
	}
}

// TestParseReplyOKTrue：成功帧的三种形状（result 是对象 / 是 null / 没有 result）。
func TestParseReplyOKTrue(t *testing.T) {
	for _, frame := range []string{
		`{"id":1,"ok":true,"result":{"handle":"h1"},"ms":0.5}`,
		`{"id":1,"ok":true,"result":null,"ms":0.0}`,
		`{"id":1,"ok":true,"ms":12}`,
	} {
		r, err := ParseReply([]byte(frame))
		if err != nil {
			t.Fatalf("%s: %v", frame, err)
		}
		if !r.OK || ExitCode(r, nil) != ExitOK {
			t.Errorf("%s: 该是 ok 且退 0，得 ok=%v code=%d", frame, r.OK, ExitCode(r, nil))
		}
	}
	// ms 是可读的浮点（引擎用 InvariantCulture 的 "0.###"）。
	r, _ := ParseReply([]byte(`{"id":1,"ok":true,"ms":0.5}`))
	if r.Ms == nil || *r.Ms != 0.5 {
		t.Errorf("ms 该是 0.5，得 %v", r.Ms)
	}
}
