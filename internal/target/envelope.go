package target

import (
	"fmt"
	"strings"
)

// 结果信封与人话渲染（契约 5.1 / 5.2 / C5.1 / C5.5–C5.7 / C5.9 / C5.10 / C5.11 / C5.14）。
//
// 这里给的是**数据**（可以 json.Marshal 的 struct）与人话的两行渲染；
// `--json` 的落盘、`internal/output` 的 Meta 平铺、以及"失败转 *output.Error"（C4.1）
// 都归 WP4 的接线 —— 本包不 import `internal/output`。

// 人话渲染里两个占位字面量（C5.9 / C5.14 明写要写字面量，**不省整段**）。
const (
	// UnnamedPlaceholder 是未命名别名的人话写法（`(未命名)`）。
	UnnamedPlaceholder = "(未命名)"
	// UnknownPlaceholder 是取不到企业编号/状态时的人话写法（`?`）。
	UnknownPlaceholder = "?"
)

// SessionEnvelope 是成功信封里的 `session` 块（契约 5.1 / C5.1）：视图的全部身份键 + `kind`。
//
// `kind` 不由视图携带：C5.5 定的是"本契约只寻址 SSH 会话"，所以它是常量 `KindSSH`。
// 这样 `session` 块**恒 11 键**，而视图恰好是"减去 kind 的那十个"（测试双向钉住）。
type SessionEnvelope struct {
	Session
	Kind string `json:"kind"`
}

// NewSessionEnvelope 把一条视图包成信封里的 `session` 块。
func NewSessionEnvelope(s Session) SessionEnvelope {
	return SessionEnvelope{Session: s, Kind: KindSSH}
}

// Requested 是 `requested` 块：把调用方**原本写的**选择器原样回显（C5.6）。
//
// 用来一眼比出"我要的"与"我拿到的"；省缺（没给 `--session`）时是空串。
type Requested struct {
	Session string `json:"session"`
}

// SuccessEnvelope 是会话级命令的成功信封（契约 5.1）。
//
// `ok` 与平铺的环境信息（`output.Meta`）沿用现行成功信封（B7）；这里落的是新增的三个块。
type SuccessEnvelope struct {
	OK        bool            `json:"ok"`
	Session   SessionEnvelope `json:"session"`
	Requested Requested       `json:"requested"`
	Data      any             `json:"data,omitempty"`
}

// NewSuccessEnvelope 造一个成功信封。requestedSelector 是调用方**原本写的**选择器（C5.6）。
func NewSuccessEnvelope(s Session, requestedSelector string, data any) SuccessEnvelope {
	return SuccessEnvelope{
		OK:        true,
		Session:   NewSessionEnvelope(s),
		Requested: Requested{Session: requestedSelector},
		Data:      data,
	}
}

// FailureEnvelope 是失败信封的**渲染形状**（契约 5.2 / C5.10 / C5.11）。
//
// 四键恒定：`ok` / `code` / `error` / `exitCode`；`hint` 有修复建议才出现；
// `candidates` 只在 C5.11 那五种码下出现。
type FailureEnvelope struct {
	OK    bool   `json:"ok"`
	Code  string `json:"code"`
	Error string `json:"error"`
	Hint  string `json:"hint,omitempty"`
	// ExitCode 是 4.1.1 那一档的退出码（老路径的 1 由 ExitFor 分档决定，见 Q15）。
	ExitCode int `json:"exitCode"`
	// Candidates 用**指针**是刻意的：C5.11 要求那五种码下这个键**必须出现**
	// （可能是空数组），而 encoding/json 的 `omitempty` 连空切片一起丢 ——
	// 只有 `*[]Candidate` 能区分"不出现"（nil）与"出现且为空"（指向空切片）。
	Candidates *[]Candidate `json:"candidates,omitempty"`
}

// Envelope 把失败数据转成可序列化的信封（四键恒定 + 按条件出现的两键）。
func (f *Failure) Envelope() FailureEnvelope {
	e := FailureEnvelope{OK: false, Code: f.Code, Error: f.Message, Hint: f.Hint, ExitCode: f.Exit}
	if CodeRequiresCandidates(f.Code) {
		list := f.Candidates
		if list == nil {
			list = []Candidate{}
		}
		e.Candidates = &list
	}
	return e
}

// CandidateLines 按 C5.14 渲染人话候选块（每条一行，前面两格缩进由 CandidateLine 带）。
//
// 渲染前按"序号 → 原始句柄"定序（不原地改调用方的切片）：同一个错误两次跑出的文案必须一样。
func (f *Failure) CandidateLines() []string {
	list := append([]Candidate{}, f.Candidates...)
	sortCandidates(list)
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, CandidateLine(c))
	}
	return out
}

// CandidateLine 按 C5.14 渲染**一行**人话候选块：
//
//	<ref>  <name>  <state>  ENT <ent>
//
// 两格缩进、字段间两个空格；未命名写 `(未命名)`；ent 取不到写 `?`。
// 第一个字段是 **ref**（恒定不变的规范 id），不是 `display` —— 候选行的用途是
// "照着它重发一枪"，照抄一个会漂的字段会让人抄到过期写法（C5.4 / C5.14）。
func CandidateLine(c Candidate) string {
	return fmt.Sprintf("  %s  %s  %s  ENT %s",
		c.Ref, orPlaceholder(c.Name, UnnamedPlaceholder), StateLabel(c.State),
		orPlaceholder(c.Ent, UnknownPlaceholder))
}

// IdentityLine 按 C5.9 渲染人话**身份行**（会话级命令唯一的"我在哪儿"那一行）：
//
//	会话 <display> · <name> · <state> · ENT <ent>
//
// 行首是字面量 `会话 `，四个字段用 ` · `（空格 + U+00B7 + 空格）分隔。
// 身份行写 `display`（人读形态），但**对账一律用 `ref`**（C5.9 的确切规则）。
//
// 契约 C5.9 的第二行（`  账号 <acct> · 库 <target> · 区域 <zone>`）不在本包：
// 那三样来自 `output.Meta`，不是会话的身份键。
func IdentityLine(s Session) string {
	return fmt.Sprintf("会话 %s · %s · %s · ENT %s",
		s.DisplayForm(), orPlaceholder(s.Name, UnnamedPlaceholder), StateLabel(s.State),
		orPlaceholder(s.Ent, UnknownPlaceholder))
}

// StateLabel 把 B30 的五个会话状态取值翻成人话标签（C5.9 明写 `<state>` 用它）。
//
// **一处已知的重复，如实写明**：`internal/cli/debug/debugenv.go` 的 `dbgStateLabel`
// 是同一份值的生产实现（WP2 的契约测试也记过这处重复）。本包零 `tt/internal` 依赖，
// 没法引用它，所以在这里落一份**生产定义**；WP4 接线时应当让 `dbgStateLabel`
// 改成引用本函数，把那份重复收掉（只留一处）。
func StateLabel(state string) string {
	switch state {
	case StateIdle:
		return "空闲"
	case StateLoading:
		return "连接中…"
	case StateStopped:
		return "已停站"
	case StateRunning:
		return "运行中"
	case StateExit:
		return "已断开"
	}
	if strings.TrimSpace(state) == "" {
		return UnknownPlaceholder
	}
	// 认不出的取值按原样打：人话行不能因为一个没见过的状态就少一段。
	return state
}

// orPlaceholder 在取不到值时给字面量占位（**不省整段**，否则字段位置会随会话漂）。
func orPlaceholder(v, placeholder string) string {
	if strings.TrimSpace(v) == "" {
		return placeholder
	}
	return v
}
