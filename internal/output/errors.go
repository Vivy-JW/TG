package output

import (
	"encoding/json"
	"fmt"
	"io"
)

// 稳定错误码。**这是对外契约,不随版本改名** —— agent 靠 Code 分支,
// 人读 Message。加新码可以,改旧码的名字不行。
const (
	CodeUsage        = "USAGE"          // 用法/参数错
	CodeEnumInvalid  = "ENUM_INVALID"   // 枚举取值非法
	CodeEnvUnknown   = "ENV_UNKNOWN"    // 环境名不存在
	CodeConfigBroken = "CONFIG_BROKEN"  // 配置缺失/坏了
	CodeTableMissing = "TABLE_MISSING"  // 字典表不在库里(该数据族未同步)
	CodeConnFailed   = "CONNECT_FAILED" // 连不上库/服务器
	CodeSQLRejected  = "SQL_REJECTED"   // 语句没过白名单
	CodeQueryFailed  = "QUERY_FAILED"   // 查询本身失败
	CodeNotFound     = "NOT_FOUND"      // 查到了,但没有这条数据

	// 会话那条线的目标族：会话、目标、体检。
	CodeNoSession     = "NO_SESSION"     // 没有会话（未登记，或登记已过期）
	CodeSessionState  = "SESSION_STATE"  // 会话状态不符
	CodeTargetMissing = "TARGET_MISSING" // 目标（连接号/窗口/对象）不存在
	CodePreflight     = "PREFLIGHT"      // 环境体检不过

	// 会话寻址那条线（`.agents/plan/tt-refactor-contract.md` 4.1.1 的 code 取值表，
	// WP4b 落地）。**加新码可以，改旧码的名字不行**（C5.13，与上面那两族同一条取向）。
	//
	// 它们今天落在**服务端**：HTTP 侧那三个"要一条会话"的端点（会话池里 >=2 条又没给目标）
	// 用 `SESSION_AMBIGUOUS` + 候选清单拒绝（C5.16：409），这正是 Q4「服务端不保留隐式寻址」
	// 的判据从"没有码的错误"变成"有码的信封"。
	//
	// 后三个（`SESSION_STALE` / `SESSION_NAME_TAKEN` / `SESSION_TARGET_BUSY`）**WP4c 补齐**：
	// 它们要 `epoch` / `name` 先成为服务端的事实（契约 C5.3 的纯新增键）—— 那一批落地后，
	// 服务端自己就会产出这三个码（起会话时的别名三种结局 + 世代失效）。
	// 取值一律与 `internal/target` 的 `Code*` 同源（那边是契约 4.1.1 的唯一出处）。
	CodeSessionAmbiguous = "SESSION_AMBIGUOUS" // 选择器/省略命中 >=2 条（F1/F3/F4）
	CodeSessionNotFound  = "SESSION_NOT_FOUND" // 选择器指名的那条不存在（F5/F7/F8）
	// CodeSessionStale 句柄是上一代（F15：环境与序号都对，但 epoch 过期）。
	CodeSessionStale = "SESSION_STALE"
	// CodeSessionNameTaken 别名已被占用（F18）。**退 2 不退 3**（4.1.1 明写）。
	CodeSessionNameTaken = "SESSION_NAME_TAKEN"
	// CodeSessionTargetBusy 别名指向的会话不能复用（F20 / Q23 / C1.6）：正在跑或已停站。
	CodeSessionTargetBusy = "SESSION_TARGET_BUSY"

	// tt update 那条线。它管的是"tt 自己"，与字典无关，所以码也自成一族。
	CodeUpdateUsage    = "UPDATE_USAGE"    // 参数组合不成立（如非交互环境没给 --yes）
	CodeUpdateRefused  = "UPDATE_REFUSED"  // 这份 tt 不让自装（源码态、认不出的布局）
	CodeUpdateChecksum = "UPDATE_CHECKSUM" // 摘要/制品自报版本校验不过
	CodeUpdateNetwork  = "UPDATE_NETWORK"  // 出网失败（代理、限流、超时）
	CodeUpdateApply    = "UPDATE_APPLY"    // 落地动作失败（覆盖不进去、msiexec 失败、计划读不出）

	// tt file 那条线（只读服务器文件）。本机落盘失败与远端读取失败分开，
	// 免得把「磁盘满了」报成「服务器连不上」。
	CodeLocalIO = "LOCAL_IO" // 本地写失败（get -o）
)

// 退出码。0 与 1 **沿用 tt 既有语义**(1 = 用法/参数错),只新增分类码 ——
// 重新编号会把既有脚本的一次性判据改掉,而换来的只是对齐一个没有权威性的约定。
const (
	ExitOK      = 0
	ExitUsage   = 1 // 用法 / 参数校验错
	ExitSource  = 2 // 数据源错(连不上、配置坏)
	ExitMissing = 3 // 缺表 / 查不到该族数据
)

// Error 带稳定码与退出码的错误。它让 --json 下的错误也是机器可读的:
// 在此之前,所有 cobra 命令的错误都是 stderr 上的一句中文散文。
type Error struct {
	Code    string
	Exit    int
	Message string
	Hint    string // 修复建议;空则不打
	Meta    Meta   // 出错前已解析出的环境信息 —— 排查"哪个环境连不上"全靠它
}

func (e *Error) Error() string { return e.Message }

// ExitCode 让 tt 的退出码管线(internal/cli 的 exitCoder)认得出它。
func (e *Error) ExitCode() int { return e.Exit }

// Errorf 构造一个带码的错误。
func Errorf(code string, exit int, format string, a ...any) *Error {
	return &Error{Code: code, Exit: exit, Message: fmt.Sprintf(format, a...)}
}

// WithHint 补一句修复建议,返回自身便于链式。
func (e *Error) WithHint(h string) *Error { e.Hint = h; return e }

// WithMeta 补上出错时的环境信息,返回自身便于链式。
func (e *Error) WithMeta(m Meta) *Error { e.Meta = m; return e }

// errorEnvelope --json 时的错误形状。Meta 匿名嵌入 = 字段平铺,
// 与成功信封共用同一组键,"哪个环境出的错"在两种情况下写法一致。
type errorEnvelope struct {
	OK    bool   `json:"ok"`
	Code  string `json:"code"`
	Error string `json:"error"`
	Hint  string `json:"hint,omitempty"`
	Exit  int    `json:"exitCode"`
	Meta
}

// WriteError 把错误写成 JSON 信封。与成功信封共用 Meta 的字段,
// 所以"哪个环境出的错"在两种情况下是同一组键。
func WriteError(w io.Writer, e *Error) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(errorEnvelope{
		OK: false, Code: e.Code, Error: e.Message, Hint: e.Hint, Exit: e.Exit, Meta: e.Meta,
	})
}
