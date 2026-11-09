// Package tzs 是 `.tzs` 表单设计器引擎（C# 外部 exe：tzs-server.exe）的
// JSON-RPC 客户端与守护进程管理。
//
// 它只做两件事，**都不做业务判断**：
//
//	client.go   一次调用：连 → 发 → 收 → 把结果归类成类型化错误
//	server.go   守护进程生命周期：spawn / 就绪握手 / 状态文件 / stop / reap
//
// 依赖面刻意窄：只 import internal/config（找状态文件落点）、internal/winproc
// （起 / 判活 / 杀）和标准库。**它不 import internal/cli/dev** —— 那边 import 它，
// 反过来就成环；命令层（动词分发、退出码、文案）住在 internal/cli/dev/tzs*.go，
// 本包只提供 exitCode 映射这类纯函数供它调用。
//
// 三条纪律，每条都有对应的坑（详见各文件注释）：
//
//  1. **管道名问引擎，不自己算、不缓存。** 名字里混了 TzsCli.Designer.dll 的 MVID，
//     每次重编都变。自己重算（或跨调用缓存）的后果是连到一个跑陈旧字节的守护进程 ——
//     设计器把「客户端绝不允许连到陈旧守护进程」写成了架构性质（Rpc.PipeName 注释），
//     所以我们只用 `--pipe-name` 问。
//  2. **工作区绝不留缺省。** 引擎的默认工作区是一个真实客户目录
//     （Rpc.DefaultWorkspace = D:\t100_wrok_dir\hengshuo\prd）。留空就等于拿客户的
//     表单当草稿纸，所以 Options.Workspace 解析不出来时一律拒绝（退出码 2），**绝不 spawn**。
//  3. **请求一旦上线绝不重试。** 协议没有幂等键，而这些函数都在改设计器内存里的模型；
//     中途失败时重试是在赌「上一次写进去了没有」。要区分两种失败：
//     先来一帧 `E_FATAL_LOAD_TIMEOUT` 然后 EOF（确定性，退 5，**不重试**）vs
//     一帧都没有的 EOF（E_SERVER_DIED，退 5，可由**下一个**命令重启守护进程）。
package tzs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
)

//---------------------------------------------------------------------------
// 协议常量
//---------------------------------------------------------------------------

// 错误分类（引擎 error.kind 的四个取值，Rpc.Map 是唯一产地）。
//
// 调用方该按 kind 而不是 code 做决定：code 是「具体哪里不对」，kind 才是
// 「你还能不能自己改对」。
const (
	KindValidation = "validation" // 自己改参数就能过
	KindNotFound   = "not_found"  // 换个名字就能过
	KindDesigner   = "designer"   // 表单本身的规则说不（换参数也过不去）
	KindInternal   = "internal"   // 不是你的错
)

// 两个由客户端在传输层判定的码。
const (
	// CodeServerDied 是「守护进程在没有应答的情况下退出」以及一切传输失败的码。
	// 引擎自己也会用同名的码（tzs-cli 的 Died()），那是它作为客户端的判定；
	// 我们直接连 tzs-server，所以这个码只会由本包合成。
	CodeServerDied = "E_SERVER_DIED"
	// CodeFatalLoadTimeout 由引擎的看门狗线程在 exit(3) 之前写出，**没有 ms、没有 result**。
	CodeFatalLoadTimeout = "E_FATAL_LOAD_TIMEOUT"
)

// 另外三个引擎会用的码，在本包里只作为**合成帧**的取值出现
// （帧的 code 由 syntheticFor 按错误自己的退出码挑，见 SyntheticFailure）。
//
// 取这几个而不是新编名字：它们是 Rpc.Map（引擎侧唯一产地）里同一种失败会用的同一批码 ——
// "行不是合法 JSON 对象"（E_BAD_REQUEST）、"设计器自己拒绝"（E_DESIGNER）、
// "未预期异常"（E_INTERNAL）。自造一批只在客户端存在的名字，等于让 `code` 这个稳定契约
// 从此有两套词汇。
const (
	CodeBadRequest = "E_BAD_REQUEST"
	CodeDesigner   = "E_DESIGNER"
	CodeInternal   = "E_INTERNAL"
)

// 三个"自纠"码：它们的意思是「换一个输入就能过」，而且**由声明推导**（收 handle / 收
// 组件路径 / 写动词收 kind）—— 引擎侧是 Manifest.AdvertisedErrors，Go 侧是
// tzs_verb_e2e_test.go 的 TestE2EManifestAdvertisesImpliedCodes（外部复核）。
//
// 为什么在这边也给它们名字：`--help` 的 errors[] 与错误帧里的 code 是同一套词汇，
// 测试和命令层照着名字引用，比照着字符串引用少一次拼错的机会。
const (
	CodeNoHandle     = "E_NO_HANDLE"
	CodePathNotFound = "E_PATH_NOT_FOUND"
	CodeNoSpecNode   = "E_NO_SPEC_NODE"
)

// E_NO_OP / E_ATTR_CLAMPED 是**成功码**：它们出现在 `result.code` 里（ok:true），
// 表示「你要的状态已经是这样了」/「写进去了但被栅格吸附过」。SPEC §11.24 (a) 把它们
// 定义成**结局**而不是失败，三种结局各带恰好一个正向标记（applied / clamped / noop）。
//
// 2026-09-25 起引擎里**没有任何路径**再把它当错误帧发：最后一处偏离是
// `set_local_string` / `set_spec_description` 的 NoOp 分支（Fns/Semantic.cs 的 NoOp()），
// 它抛 DetailedError，而 Rpc.Map 的 default 分支给未知 E_* 码的 kind 是 `internal`
// —— 于是调用方看到的 `kind:internal` 意思是「未预期异常，上报，别重试」，正好是反的。
// 那条路已改成和 Attr.cs / Action.cs 一样的成功帧（`set_code_template` 也补上了它一直
// 缺的 `noop:true` + `code`）。
//
// 下面两个常量与 IsSuccessCode 因此变成**兜底**：判据留着，是因为旧引擎仍可能在跑
// （守护进程按 MVID 命名，重编前后各有一批），而且"成功码被扔进 error"这件事本身
// 值得在文案上认出来，而不是当成「报 bug」。
const (
	CodeNoOp        = "E_NO_OP"
	CodeAttrClamped = "E_ATTR_CLAMPED"
)

// 读超时（契约给的三个数，全部来自 tzs-cli 的实测常量）。
const (
	// ConnectTimeout 是单次 connect 的超时。小是故意的：热守护进程 1 ms 内应答，
	// 再长只是把 spawn 路径往后拖。
	ConnectTimeout = 300 * time.Millisecond
	// DefaultTimeout 是一次调用的读超时。
	//
	// 为什么是 300 s 而不是几秒：`open` 在某些包上会阻塞到**引擎自己的 90 s 加载看门狗**
	// （TZSCLI_RELOAD_TIMEOUT 缺省 90），而 `validate` 在 670 元素的表单上实测 10.4 s。
	// 给 30 s 会把一次正常的 open 判成传输失败，用户看到的是「守护进程无响应」——
	// 一个和真实原因（包大）毫无关系的结论。
	DefaultTimeout = 300 * time.Second
	// MinTimeout 是命令层允许 `--timeout` 收到的最小值。
	//
	// 下限存在的唯一理由是上面那条 90 s 看门狗：允许 `--timeout 5` 就等于允许用户
	// 把每个慢包都判成「守护进程死了」，然后把守护进程反复重建。
	MinTimeout = 120 * time.Second
)

// ExitCode 是 tdev 的退出码（与 internal/cli/dev 的 0/2/3/4/5 同一套口径）。
//
// 注意这里只有 0/1/2/4/5 五个取值，3 是 tzc 那条管线的「验证失败」，
// tzs 这条线上没有对应物 —— 帧级失败一律落到 2（可自纠）或 4（设计器拒绝）。
const (
	ExitOK        = 0 // 帧 ok:true
	ExitFrameErr  = 1 // kind=internal（含 E_NOT_IMPLEMENTED）
	ExitUsage     = 2 // kind=validation / not_found、本地参数错、未知函数、manifest 拉不到
	ExitDesigner  = 4 // kind=designer（含 E_KEY_IN_USE）
	ExitTransport = 5 // 传输失败、E_FATAL_LOAD_TIMEOUT、环境/IO 失败
)

//---------------------------------------------------------------------------
// 类型化错误
//---------------------------------------------------------------------------

// UsageError 是**用法错**：未知函数、本地参数错（类型/枚举/必填/未知参数）、
// manifest 拉不到。退出码 2。
//
// 为什么把 manifest 拉不到也算用法错：到这一步工具连函数表都没有，
// 无法判断调用方写的东西是否合法 —— 它和「参数名打错」是同一类处境，
// 都是「这次调用根本没能成立」，而不是「引擎拒绝了这次调用」。
type UsageError struct {
	Msg    string
	Detail []string
}

func (e *UsageError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "（" + strings.Join(e.Detail, "；") + "）"
}
func (e *UsageError) ExitCode() int { return ExitUsage }

// TransportError 是**传输/环境失败**：连不上、读超时、EOF 无帧、非 JSON 应答、
// 引擎 exe 起不来、管道名问不到。退出码 5。
//
// 它**不代表**「请求失败了」：请求可能已经上线并写进了设计器内存，只是应答没回来。
// 所以调用方绝不能拿它当重试依据（见包注释纪律 3）。
type TransportError struct {
	Code string // 一般是 CodeServerDied
	Msg  string
	Err  error // 底层错误（可能为 nil）
}

func (e *TransportError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}
func (e *TransportError) Unwrap() error { return e.Err }
func (e *TransportError) ExitCode() int { return ExitTransport }

// WireError 是一帧 `error` 对象的类型化视图。
//
// Detail 用 json.RawMessage 而不是 map：引擎的 detail 形状按函数而变
// （`param/reason/message`、`candidates`、`legal`、`vocabulary`、`exception/at`…），
// 在 Go 侧抄一份 map 结构就多了一处会漂移的第二实现。要取字段就自己解。
type WireError struct {
	Code    string          `json:"code"`
	Kind    string          `json:"kind"`
	Message string          `json:"message"`
	Detail  json.RawMessage `json:"detail"`
}

func (e *WireError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Kind == "" {
		return e.Code + ": " + e.Message
	}
	return e.Code + " (" + e.Kind + "): " + e.Message
}

// ExitCode 把一帧错误映射成 tdev 退出码（契约表）。
//
// 顺序是有讲究的：**先看 code 再看 kind**。E_FATAL_LOAD_TIMEOUT 的 kind 是
// `designer`（引擎 WriteFatal 的注释解释了为什么：调用方不能重试，该修的是上游的
// mta/tables.xml），照 kind 走会得到 4「写入被拒」——那是在说「表单的规则拒绝了」，
// 而真相是「这个包根本加载不完」。所以这两行的先后不能换。
func (e *WireError) ExitCode() int {
	if e == nil {
		return ExitTransport
	}
	return frameExitCode(e.Code, e.Kind)
}

// IsSuccessCode 报告这个码在语义上是不是成功（见 CodeNoOp 的注释）。
//
// 现在没有任何引擎路径会以 ok:false 发出它（2026-09-25 起，见上）—— 留着是为了让
// "成功码出现在 error 里"这件事在文案上仍是「什么也没改」而不是「真的坏了」。
func (e *WireError) IsSuccessCode() bool {
	if e == nil {
		return false
	}
	return e.Code == CodeNoOp || e.Code == CodeAttrClamped
}

// frameExitCode 是契约的退出码表本身，抽出来是为了能直接对表打表测试。
func frameExitCode(code, kind string) int {
	switch code {
	case CodeFatalLoadTimeout:
		// 契约单列这一行（5）。它的 kind 是 designer，所以必须在 kind 之前判。
		return ExitTransport
	case CodeServerDied:
		// 传输层自己合成的码；引擎不会在 data 帧里发它。
		return ExitTransport
	}
	switch kind {
	case KindValidation, KindNotFound:
		return ExitUsage
	case KindDesigner:
		return ExitDesigner
	}
	// kind=internal 走这里（含 E_NOT_IMPLEMENTED）。E_NO_OP 曾经也从这里走（引擎把它
	// 发成错误帧时），2026-09-25 起引擎不再有那条路径 —— 表不动：真收到这种帧，按
	// 「未分类的内部错误」退 1 仍是对的兜底。
	// 未知/缺失的 kind 也落在这里：契约表没给它行，按「未分类的内部错误」兜底。
	return ExitFrameErr
}

// ExitCode 把一次调用的结果映射成 tdev 退出码 —— 命令层只该调这一个。
//
//	err != nil  → err 自己的退出码（UsageError 是 2，TransportError 是 5）
//	reply == nil → 5（没有应答就当成传输失败，不要静默退 0）
//	reply.OK    → 0
//	否则        → reply.Error.ExitCode()
func ExitCode(reply *Reply, err error) int {
	if err != nil {
		var ec interface{ ExitCode() int }
		if errors.As(err, &ec) {
			return ec.ExitCode()
		}
		return ExitFrameErr
	}
	if reply == nil {
		return ExitTransport
	}
	if reply.OK {
		return ExitOK
	}
	return reply.Error.ExitCode()
}

// SyntheticFailure 把一次"没能成为帧"的失败**合成成一帧**，形状与引擎帧完全一致。
//
// 为什么需要它：`--json` 的消费方读的是帧（`{id,ok,result,error,ms}`），而这条链上有几处
// 失败根本到不了引擎 —— 连不上、本地参数错、manifest 拉不到。从前它们各写各的：
// 两处写的是 `fail()` 那份**另一个形状**的信封（`{ok,exit_code,error}`，那是 tzc 那条线的），
// 一处什么都不写，而传输失败那处往 stdout 打了个字面量 `null`（`json.Marshal(nil)`）。
// 现在一律走这里。
//
// **不在 Go 侧另写一个 struct**：形状就是 `Reply` 本身，逐字段对齐是抄第二份、会漂。
// `id` 留 null 而不是 0 —— 帧级错误本来就不回显 id（引擎的 WriteFatal 也是这样）。
func SyntheticFailure(err error) *Reply {
	if err == nil {
		return nil
	}
	code, kind := syntheticFor(err)
	return &Reply{OK: false, Error: &WireError{Code: code, Kind: kind, Message: err.Error()}}
}

// syntheticFor 按 err **自己的退出码**挑一对 (code, kind)。
//
// 硬约束：`frameExitCode(code, kind)` 必须等于 `err.ExitCode()`。不然同一个失败会有两种
// 说法（"帧说 5、进程退 2"），而消费方只读其中一个。按退出码而不是按错误类型挑，
// 这条约束就是**构造上成立**的，不靠人记得对齐 —— TestSyntheticFailure 会对四种错误
// 逐个验这条等式。
func syntheticFor(err error) (code, kind string) {
	var ec interface{ ExitCode() int }
	if !errors.As(err, &ec) {
		return CodeInternal, KindInternal // 未分类：按"不是你的错"兜底（退 1）
	}
	switch ec.ExitCode() {
	case ExitUsage:
		return CodeBadRequest, KindValidation // 退 2：参数/环境不对，可以自纠
	case ExitDesigner:
		return CodeDesigner, KindDesigner // 退 4：表单自己的规则说不
	case ExitTransport:
		// 退 5。E_SERVER_DIED 就是本包给"一切传输失败"的码（见上面的常量注释），
		// 包括守护进程没应答就退出。
		return CodeServerDied, KindInternal
	default:
		// ExitFrameErr（退 1）以及任何没见过的码：内部错。
		return CodeInternal, KindInternal
	}
}

// NormalizeTimeout 把 `--timeout` 收到的值规范化：0/负数 → 默认 300 s；// 正数但小于 120 s → 抬到 120 s（理由见 MinTimeout）。
//
// 命令层应当在**解析完 flag 之后**、调用 Call 之前过一遍它。
func NormalizeTimeout(d time.Duration) time.Duration {
	if d <= 0 {
		return DefaultTimeout
	}
	if d < MinTimeout {
		return MinTimeout
	}
	return d
}

//---------------------------------------------------------------------------
// 连接与调用
//---------------------------------------------------------------------------

// Dialer 连到一个管道名。抽成接口是为了让默认测试不碰真实管道。
type Dialer interface {
	Dial(ctx context.Context, pipeName string) (Conn, error)
}

// Conn 是一条已经连好的双向帧流。
//
// **两个方向的约定是反的**，照契约来：
//
//	Send(frame)  —— frame 是**完整帧**，含结尾 `\n`（「一行一帧」里那一行连换行在内）。
//	                少了那个换行，对端会把两帧并成一帧读；多了也不会被原谅（见下面的校验）。
//	Recv()       —— 返回**正文**，不含结尾 `\n`（行尾是帧界，不是内容）。
//
// 之所以要写清楚：把 Send 当成「发正文」的实现在单测里能过（假 Conn 不校验），
// 真机上表现为「第一次调用就超时」。
type Conn interface {
	Send([]byte) error
	Recv() ([]byte, error)
	Close() error
}

// Reply 是一帧应答。
//
// 三个字段的形状是被契约钉死的，每一个都对应一条实测事实：
//
//	ID     —— JSON 的 `id` 可以是 null（帧级错误就是 {"id":null,...}），
//	          所以必须是 RawMessage。用 int 会在解 `null` 时失败，用 *int 会丢掉
//	          「原样回显」这个信息。
//	Error  —— 指针：ok:true 的帧没有 error。
//	Ms     —— **指针**：致命帧（WriteFatal）没有 ms 也没有 result。
//	          写成 float64 的话，每一个致命帧都会被解成 ms=0，而「0.0 ms」看着像
//	          「守护进程答得飞快」——最坏的一种误读。
type Reply struct {
	ID     json.RawMessage `json:"id"`
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  *WireError      `json:"error"`
	Ms     *float64        `json:"ms"`
}

// MarshalRequest 造一帧请求：{"id":n,"fn":"...","args":{...}}。
//
// args 以 RawMessage 逐字节拼进去，不重新解析：它就是 argmap 按 manifest 定型好的
// 对象，再解一遍只会多一次出错的机会。args 为空时补 `{}` 而不是 `null`
// （引擎两者都收，但 `{}` 让人读帧时不用去想 null 是什么意思）。
func MarshalRequest(id int, fn string, args json.RawMessage) ([]byte, error) {
	if len(args) == 0 {
		args = json.RawMessage("{}")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	// 关掉 HTML 转义：默认会把 < > & 写成 \u003c 这类。语义等价，但
	// `--value "</section>"` 在日志里变成一坨 \u003c 就没法对着看，
	// 而看帧是排查问题时唯一的手段。
	enc.SetEscapeHTML(false)
	err := enc.Encode(struct {
		ID   int             `json:"id"`
		Fn   string          `json:"fn"`
		Args json.RawMessage `json:"args"`
	}{id, fn, args})
	if err != nil {
		return nil, &UsageError{Msg: "参数不是合法 JSON（本不该发生）", Detail: []string{err.Error()}}
	}
	// Encoder.Encode 会补一个尾随 \n；这里返回的是**正文**，换行由 CompleteFrame 补。
	// 留着它会让「帧里有裸换行」那条闸门当场拒掉我们自己的帧。
	return bytes.TrimSuffix(buf.Bytes(), []byte{'\n'}), nil
}

// Call 做一次调用：连 → 发 → 收 → 返回帧或传输错误。
//
// **不做的事**（每一条都是刻意的）：
//
//   - 不 spawn。起守护进程是 Ensure 的活；Call 连不上就报传输失败，
//     让下一个命令去重启（tzs-cli 的 stop 也是同一条原则：让「停止」去启动服务是个笑话）。
//   - 不重试。见包注释纪律 3。
//   - 不把帧级错误当 error 返回。帧说得清清楚楚（ok:false + error），
//     把它压成一个 error 会丢掉 kind/code/detail，而退出码恰恰只看这三个。
//
// timeout <= 0 时用 DefaultTimeout；调用方要「读超时下限 120 s」请先用 NormalizeTimeout。
func Call(ctx context.Context, d Dialer, pipeName string, id int, fn string, args json.RawMessage, timeout time.Duration) (*Reply, error) {
	if d == nil {
		return nil, &TransportError{Code: CodeServerDied, Msg: "没有 Dialer"}
	}
	dctx, cancel := context.WithTimeout(ctx, ConnectTimeout)
	conn, err := d.Dial(dctx, pipeName)
	cancel()
	if err != nil {
		var te *TransportError
		if errors.As(err, &te) {
			return nil, te
		}
		return nil, &TransportError{Code: CodeServerDied,
			Msg: "连不上守护进程（" + ConnectTimeout.String() + " 内没有应答）", Err: err}
	}
	defer conn.Close()
	return CallConn(ctx, conn, id, fn, args, timeout)
}

// CallConn 在一条**已经连好**的连接上做一次调用。
//
// 单独暴露是因为就绪握手要复用它（那时连接刚建立，再 Dial 一次没有意义）。
// Call = Dial + CallConn。
func CallConn(ctx context.Context, conn Conn, id int, fn string, args json.RawMessage, timeout time.Duration) (*Reply, error) {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	req, err := MarshalRequest(id, fn, args)
	if err != nil {
		return nil, err
	}
	// MarshalRequest 给的是正文（不含换行）；Conn.Send 的约定是完整帧。
	frame, err := CompleteFrame(req)
	if err != nil {
		return nil, err
	}

	type outcome struct {
		frame []byte
		err   error
	}
	ch := make(chan outcome, 1)
	go func() {
		if err := conn.Send(frame); err != nil {
			ch <- outcome{nil, err}
			return
		}
		got, err := conn.Recv()
		ch <- outcome{got, err}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, &TransportError{Code: CodeServerDied, Msg: "调用被取消", Err: ctx.Err()}
	case <-timer.C:
		// 读超时只能这样做：PipeStream 拒绝 ReadTimeout（"Timeouts are not supported
		// on this stream"），也没有别的取消手段 —— tzs-cli 也是拿一条线程等事件。
		// 后台那条 goroutine 会一直卡在 Read 上，这不是问题：本进程一次只跑一个动词，
		// 退出时就没了；而且上面的 conn.Close() 会让它的 Read 立刻失败。
		return nil, &TransportError{Code: CodeServerDied,
			Msg: "守护进程无响应（读超时 " + timeout.String() + "）"}
	case r := <-ch:
		if r.err != nil {
			if errors.Is(r.err, io.EOF) {
				// 一帧都没有的 EOF。它与「先来一帧 E_FATAL_LOAD_TIMEOUT 再 EOF」是两码事，
				// 但两者都是退 5；区分它们靠的是有没有帧，而走到这里就是没有帧。
				return nil, &TransportError{Code: CodeServerDied,
					Msg: "守护进程在没有应答的情况下退出（EOF，无帧）"}
			}
			var te *TransportError
			if errors.As(r.err, &te) {
				return nil, te
			}
			return nil, &TransportError{Code: CodeServerDied, Msg: "连接中断", Err: r.err}
		}
		return ParseReply(r.frame)
	}
}

//---------------------------------------------------------------------------
// 一个通用 Conn 实现（真实管道只是它的一个底层流）
//---------------------------------------------------------------------------

// streamConn 把任意双向字节流包成 Conn。
//
// 命名管道之外的每一处（就绪探测、单测里的假连接）都复用这一份，
// 于是「帧怎么读怎么发」只有一处实现 —— 平台相关的只有「怎么把流连出来」。
func streamConn(rw io.ReadWriteCloser) Conn {
	return &streamConnImpl{rw: rw, br: bufio.NewReaderSize(rw, 64*1024)}
}

type streamConnImpl struct {
	rw io.ReadWriteCloser
	br *bufio.Reader
}

func (c *streamConnImpl) Send(frame []byte) error {
	// 先校验再写：一个少了结尾换行的帧，与其发出去让对端把两帧并成一帧读，
	// 不如在本地就断掉（见 Conn 的注释）。
	if err := validateFrame(frame); err != nil {
		return err
	}
	_, err := c.rw.Write(frame)
	return err
}

func (c *streamConnImpl) Recv() ([]byte, error) {
	frame, err := ReadFrame(c.br)
	// 「对端走了」在 Windows 上**不一定是 io.EOF**：命名管道那侧关掉/断开之后，
	// ReadFile 回的是 ERROR_BROKEN_PIPE / ERROR_PIPE_NOT_CONNECTED / ERROR_NO_DATA
	// （实测拿到的是 233「No process is on the other end of the pipe」，
	// 而这个事实是 pipe_windows_test.go 的 EOF 用例发现的 —— 只看文档会以为一定是
	// ERROR_BROKEN_PIPE，而 Go 的 poll 层只把 BROKEN_PIPE 映射成 io.EOF）。
	// 归一成 io.EOF 之后，上层只需要判一种「没有应答就没了」，
	// 并且文案说的是「守护进程走了」而不是一句内核原文。
	if err != nil && err != io.EOF && isPeerGone(err) {
		return nil, io.EOF
	}
	return frame, err
}

func (c *streamConnImpl) Close() error { return c.rw.Close() }

// NewStreamConn 把一条已经建好的双向流包成 Conn（供命令层做标准输入输出直连等用途）。
func NewStreamConn(rw io.ReadWriteCloser) Conn { return streamConn(rw) }

//---------------------------------------------------------------------------
// 小工具
//---------------------------------------------------------------------------

func itoa(n int) string { return strconv.Itoa(n) }
