package tzs

// 帧编解码 —— 协议的最底层，逐字节对齐引擎的 Rpc.cs（LineReader / WriteLine）。
//
// 帧 = 一行 JSON + 一个 `\n`，UTF-8 无 BOM。两个方向共用同一条规则，但容错面不同：
//
//   - **出方向**只发 `\n`，且**拒绝**帧里出现裸的 `\r`/`\n`。理由不是洁癖：一个裸换行会把
//     一帧劈成两帧，对端在下一次请求上读到上一帧的尾巴，之后每一帧都错位一格 ——
//     表现是「明明参数对，引擎却报 E_BAD_REQUEST / 未知函数」，而看代码怎么都看不出来。
//     我们的帧全部由 encoding/json 产出（换行必转义成 \n 两字符），所以这条闸门永远不会
//     误伤真帧，只会逮住手搓的帧。
//   - **入方向**容忍行尾 `\r`（引擎的 LineReader.Finish 会剥掉一个）与行首 BOM
//     （Serve 里会显式剥掉行首的一个 U+FEFF）。这两个容错不是可选项：设计器那侧
//     的日志/诊断可能混到同一条流上（stdio 模式），而 BOM 是记事本一类工具的默认输出。
//
// 行长上限 8 MiB（引擎的 Rpc.MaxLine，双向）。**不要用默认 64 KiB 上限的 bufio.Scanner**：
// `form_tree` 在大表单上一次就几十万字节，Scanner 会以 `bufio.Scanner: token too long`
// 这种和表单毫无关系的错误终止，读起来像表单坏了。这里用 bufio.Reader + 跨块累积。

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

// MaxLine 是一帧（不含结尾 `\n`）的字节上限，与引擎的 Rpc.MaxLine 同值。
//
// 边界口径照抄引擎的 LineReader：判据是 `长度 > MaxLine` 才拒绝，所以**恰好** MaxLine
// 的一帧是合法的。写错成 `>=` 会让一条正好 8 MiB 的 form_tree 在客户端被判死，
// 而引擎那侧认为它没问题。
const MaxLine = 8 * 1024 * 1024

// utf8BOM 是 UTF-8 的 BOM 字节。只剥行首的一个。
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// CompleteFrame 把一帧**正文**补成完整帧（含结尾 `\n`）并校验。
//
// 两个方向的约定在这里定死，因为它们是**反的**（Conn 的注释里也写了）：
//
//	Conn.Send 收的是**完整帧**（含结尾 \n）——「一行一帧」里那一行连换行在内；
//	Conn.Recv 给的是**正文**（不含结尾 \n）—— 行尾是帧界，不是内容。
//
// 正文里不许出现裸 `\r`/`\n`：见包注释（一个裸换行会把一帧劈成两帧，
// 之后每次交互都错位一格，而报出来的是「未知函数」这种与真因无关的错）。
func CompleteFrame(body []byte) ([]byte, error) {
	if len(body) > MaxLine {
		return nil, &TransportError{Code: CodeServerDied, Msg: "帧超过 8 MiB 上限（" + itoa(len(body)) + " B）"}
	}
	if bytes.IndexAny(body, "\r\n") >= 0 {
		return nil, &TransportError{Code: CodeServerDied,
			Msg: "帧里有裸换行：一行一帧的协议下它会把这一帧劈成两帧，之后每一帧都错位"}
	}
	buf := make([]byte, 0, len(body)+1)
	buf = append(buf, body...)
	buf = append(buf, '\n')
	return buf, nil
}

// validateFrame 检查一个「完整帧」（末尾必须恰好是 \n，正文里不许再有换行）。
func validateFrame(frame []byte) error {
	if len(frame) == 0 || frame[len(frame)-1] != '\n' {
		return &TransportError{Code: CodeServerDied,
			Msg: "Conn.Send 收的是**完整帧**（含结尾 \\n）；少了那个换行，对端会把两帧并成一帧读"}
	}
	_, err := CompleteFrame(frame[:len(frame)-1])
	return err
}

// WriteFrame 把一帧正文补上结尾 `\n` 写出去。
//
// 与 Conn.Send 的分工：Send 的约定是「已经完整的帧」，这个函数是「给正文补换行」，
// 给不需要 Conn 的地方（测试、工具）用。
func WriteFrame(w io.Writer, body []byte) error {
	frame, err := CompleteFrame(body)
	if err != nil {
		return err
	}
	_, err = w.Write(frame)
	return err
}

// ReadFrame 读一整帧（**不含**结尾 `\n`），容忍行尾 `\r` 与行首 BOM。
//
// 没读到任何字节就 EOF 时返回 io.EOF —— 调用方据此区分「守护进程一声不响地走了」
// （E_SERVER_DIED）与「回了一帧空内容」（后者是引擎的 E_BAD_REQUEST，不是传输失败）。
func ReadFrame(r *bufio.Reader) ([]byte, error) {
	var acc []byte
	for {
		chunk, err := r.ReadSlice('\n')
		acc = append(acc, chunk...)
		// 先卡上限再看 err：ReadSlice 在 ErrBufferFull 时已经把一块交给我们了，
		// 不在这里拦，一条无换行的巨流会被无限累积到 OOM。
		if len(acc) > MaxLine+1 { // +1 是给结尾那个 \n 留位置
			return nil, &TransportError{Code: CodeServerDied,
				Msg: "应答超过 8 MiB 上限，已放弃读取（引擎那边不会发这种帧，多半是流被别的东西污染了）"}
		}
		switch {
		case err == nil:
			// 拿到 \n，落下去处理
		case err == bufio.ErrBufferFull:
			continue
		case err == io.EOF:
			if len(acc) == 0 {
				return nil, io.EOF
			}
			// 有半帧但没等到 \n：引擎不会被 EOF 前的半个帧救回来，这就是 EOF 无帧。
			return nil, io.EOF
		default:
			return nil, err
		}
		break
	}
	frame := acc[:len(acc)-1] // 去掉 \n
	frame = bytes.TrimSuffix(frame, []byte{'\r'})
	frame = bytes.TrimPrefix(frame, utf8BOM)
	if len(frame) > MaxLine {
		return nil, &TransportError{Code: CodeServerDied, Msg: "应答超过 8 MiB 上限"}
	}
	return frame, nil
}

// ParseReply 把一帧（ReadFrame 的产物）解析成 Reply。
//
// 三个「不是帧」的判定，全部归到传输失败（E_SERVER_DIED，退出 5），**绝不合成一个
// 看起来像业务错误的应答**：一句「不是 JSON 对象」的解析错误读起来像一条关于表单的
// 事实，AI 会据此断定「这个元素不存在」——引擎的 Rpc.Classify 注释把这条讲得最清楚，
// 这里的口径与它逐字对齐。
//
//   - 顶层不是 JSON 对象（数字、数组、null、字符串）
//   - 没有 `ok` 字段，或 `ok` 不是布尔（引擎用 ok 做唯一判别依据；
//     只看 error 字段判断成败的调用方会在成功路径上判错）
//   - `ok:false` 却没有 `error` 对象（引擎任何一条错误路径都会带 error）
func ParseReply(frame []byte) (*Reply, error) {
	// 中间类型用 *bool：既能区分「ok 缺席/null」与「ok:false」，又只解析一次。
	// 直接解码成 Reply 的 OK bool 就分不出来了 —— 一个 {"foo":1} 会被当成 ok:false 的帧。
	var wf struct {
		ID     json.RawMessage `json:"id"`
		OK     *bool           `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *WireError      `json:"error"`
		Ms     *float64        `json:"ms"`
	}
	if err := json.Unmarshal(frame, &wf); err != nil {
		return nil, notAFrame("应答不是 JSON 对象（已按 E_SERVER_DIED 处理）: " + clip(frame))
	}
	if wf.OK == nil {
		return nil, notAFrame("应答没有 ok 字段（不是本协议的帧，已按 E_SERVER_DIED 处理）: " + clip(frame))
	}
	r := &Reply{ID: wf.ID, OK: *wf.OK, Result: wf.Result, Error: wf.Error, Ms: wf.Ms}
	if !r.OK && r.Error == nil {
		return nil, notAFrame("应答 ok:false 但没有 error 对象（帧不成形，已按 E_SERVER_DIED 处理）: " + clip(frame))
	}
	return r, nil
}

// notAFrame 造一个传输失败。Code 固定 E_SERVER_DIED：与 tzs-cli 在同一个位置
// （Died()）用的码一致 —— 它说的是「传输层没能给你一个可判读的应答」，
// 而不是「表单里没有这个东西」。
func notAFrame(msg string) error {
	return &TransportError{Code: CodeServerDied, Msg: msg}
}

// clip 把一帧裁到可读长度（只用于报错文案）。
func clip(b []byte) string {
	const max = 200
	s := string(b)
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
