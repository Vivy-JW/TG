package debug

// 会话寻址的**接线层**（WP4a）：CLI 侧"取第一个会话"换成"向目标层要目标"。
//
// 从前 CLI 拿会话的路径是：`GET /api/sessions` → 取回包数组的**第一项**的 id
// → 拼进 `"/api/sessions/"+id+"/…"`（契约 B10，计划开头 A–E 表的 A 一行）。
// 本文件把中间那一步换成 `internal/target` 的目标层：
//
//	列会话（HTTP）→ 转成 target.Session（**这就是那个薄适配**）→ target.ParseSelector
//	+ target.Resolve → 拿到的 id 才拼进 URL。
//
// 那个"取第一项"的旧入口函数在 WP4a 里被整个删掉（不留副本），它的调用点全部改到
// 本文件的 `dbgSessionID` —— 所以静态检查（T2.3）里那一族的命中数现在是 0。
//
// 三条纪律写死在这里（契约第 8 节 / Q2 / Q4 / Q12）：
//
//  1. **单会话不许变**（Q2 / C8.1）：池里 0 条或 1 条时"省略目标"照旧自动落位。
//     `Resolve` 的 pickUnique 就是这条：唯一就落，绝不多问一个字。
//     "唯一就落"这条便利**住在 CLI**（Q12 / C8.2）—— 服务端不保留隐式寻址，
//     所以先列再发这件事必须在这里做。
//  2. **多会话且没给目标 → 报错并列候选**（F1），不许取第一个。
//  3. **给的目标对不上 → 报错**（F3–F8），不许静默取其一。
//
// 分层纪律（Q15 的"分档生效"，判据见 4.2 本轮改正后的口径）：**判据是"这个场景是不是
// 新场景"**，不是"命令行里有没有 `--session` 这个字"。落在今天不可能发生的新场景里
// （F1/F3–F5/F7–F9/F15/F18/F20…）→ 无条件走契约 4.1 那张码表；今天的既有失败
// （0 条会话、环境名写错、少给必填参数、标志挂错命令…）→ 维持今天的码（1），
// 只有真用了新写法才走码表。前者由 `target.Failure.NewScenario` 一位决定，
// 后者由 `dbgNewSyntax()` 决定，两处在 `dbgTargetError.ExitCode` 合流。

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/output"
	"tt/internal/target"
)

var (
	// dbgSessionFlag 是 `--session <选择器>`（契约 C3.1：它是**会话**的权威）。
	// 契约 1. 的命令角色表定它挂会话级操作命令；今天没被任何命令注册（B19）。
	dbgSessionFlag string
	// dbgNameFlag 是 `--name <别名>`（契约 C2.5 / Q17）：给这条 SSH 会话起别名，
	// 别名**就是幂等键**（Q23 / C1.6）。只在起会话的命令上有意义（C3.5）。
	dbgNameFlag string
)

// dbgSessionPool 列会话（`GET /api/sessions`）+ 列环境（`GET /api/hosts`），
// 转成目标层的**只读视图**（契约 C5.1 的十一键，见 dbgSnapView 的注释）。
//
// 这是"那个薄适配"：字段名一次都不手抄 —— 声明处是 `internal/target` 的 `Session`。
// `ref` / `display` / `name` / `n` / `epoch` 五个键**由服务端产出**（WP4c 落地，C5.3），
// 所以这里的解包是直接的：服务端给什么就是什么。`target.Session.CanonicalRef()` /
// `DisplayForm()` 仍留着兜底（服务端没给 ref 时退回原始句柄），但**正常情况下不该被用到** ——
// 用到了就说明对面是个不认识这五个键的旧服务端。
func dbgSessionPool() (target.Pool, error) {
	data, err := dbgAPI("GET", "/api/sessions", nil)
	if err != nil {
		return target.Pool{}, err
	}
	var r struct {
		Sessions []dbgSnapView `json:"sessions"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return target.Pool{}, fmt.Errorf("解析会话列表失败: %w", err)
	}
	pool := target.Pool{Sessions: make([]target.Session, 0, len(r.Sessions))}
	for _, v := range r.Sessions {
		pool.Sessions = append(pool.Sessions, v.Session)
	}
	// 环境清单只用于 F6 的"可用环境：…"那一句；拿不到就留空 ——
	// 不为了一个人话提示把"选会话"这件事判失败。
	if names, err := dbgEnvNames(); err == nil {
		pool.Environments = names
	}
	return pool, nil
}

// dbgEnvNames 取配置里的**全部**环境名（`GET /api/hosts` 的 `sshs[].name`）。
//
// F6（环境不存在）要在提示里列出全部可用环境，而那个清单的权威在配置里（B23）：
// CLI 侧读配置要多一套路径解析，服务端那个端点已经把它端出来了 —— 只借它这一件事。
func dbgEnvNames() ([]string, error) {
	data, err := dbgAPI("GET", "/api/hosts", nil)
	if err != nil {
		return nil, err
	}
	var r struct {
		SSHs []struct {
			Name string `json:"name"`
		} `json:"sshs"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("解析环境清单失败: %w", err)
	}
	out := make([]string, 0, len(r.SSHs))
	for _, e := range r.SSHs {
		if e.Name != "" {
			out = append(out, e.Name)
		}
	}
	return out, nil
}

// dbgNewSyntax 判"这次调用有没有用到被冻结的新写法"（Q15 分档的**第二半**）。
//
// 判据与契约 4.2 的定稿第 1 条逐字一致：**命令上接受且出现了 `--session` 或 `--name`**
// → 按 4.1 的完整码表退；两者都没出现（含标志写在**不接受**它的命令上，
// 见 C2.8b / C2.8c）→ 维持今天的码（成功 0，其余 1）。
//
// **"接受"那一半由命令树保证**：本函数只在读得到标志值的命令里被调用，而挂着这两个
// 标志的命令就是"接受它们"的命令（F16 / F17 那两条"挂错命令"的路由 cobra 拦下，
// 根本走不到这里）。所以这里只看"出现没出现"。
//
// # 它只是**第二半**，不是全部判据（本轮改正在这里）
//
// 分档的**主判据是"这个场景是不是新场景"**，那一位来自目标层（`target.Failure.NewScenario`）：
// 今天不可能发生的场景**无条件**按 4.1 的码表退，与命令行里有没有 `--session` 无关。
//
// 为什么必须这样：H2 的全句是「**只有一条会话**时，所有命令的输出与今天逐字相同」——
// 它保护的是**今天已有的行为**。而"多会话 + 不给目标"（F1）今天会**静默取第一条并成功**，
// 没有"退 1"这个旧行为要保护；更关键的是 **Agent 恰恰是在没打 `--session` 时**才撞上歧义，
// 它最需要机器可读码的那一刻，不许因为"它没打标志"就把码降回 1。
//
// 所以本函数只负责"既有失败 + 真用了新写法"这一档（4.2 的 H2 表逐行写着这条两档：
// F2 带 `--session` → 3、F6 写在 `--session` 里 → 2）；新场景那一档走 NewScenario，
// 一个标志都不看。
func dbgNewSyntax() bool {
	return strings.TrimSpace(dbgSessionFlag) != "" || strings.TrimSpace(dbgNameFlag) != ""
}

// dbgTargetSession 向目标层要**目标**：列池 → 解析选择器 → 落位 → 唯一一条会话。
//
// 失败返回 `*dbgTargetError`（它同时是 `*output.Error`，带契约 4.1.1 的 `code`
// 与分档后的退出码）。今天"没有活动会话"那句话的文案由目标层给（F2），
// 与从前那句意思相同、措辞不同 —— 这一条与下面 dbgSessionID 的旧文案分支需要
// 什么条件才保留，见那两处的注释。
func dbgTargetSession(environments []string, now target.Request) (target.Session, error) {
	// ⓪ 文法非法（F9–F11）在**发请求之前**判掉（C2.8）：dbgSelector 解析失败时把失败
	//    挂在那里，这里一进来就取走 —— 这一个字节都不该发给服务端。
	if dbgSelectorFailure != nil {
		fail := dbgSelectorFailure
		dbgSelectorFailure = nil
		return target.Session{}, dbgTargetErrorOf(fail)
	}
	pool, err := dbgSessionPool()
	if err != nil {
		return target.Session{}, err
	}
	if environments != nil {
		pool.Environments = environments
	}
	sess, fail := target.Resolve(pool, now)
	if fail != nil {
		return target.Session{}, dbgTargetErrorOf(fail)
	}
	return sess, nil
}

// dbgSessionID 是会话级命令要的那个 id（拼进 REST 路径 `{id}`，C5.4）。
//
// 它取代了 CLI 那个"列一次会话、取回包数组第一项"的旧入口函数（契约 B10 点名的就是它，
// 那是计划开头 A–E 表的 A 一行）。与旧入口的行为差别只有一处：
// **多会话时报错并列候选**，而不是静默取列表第一个 —— 这正是本次重构要的。
//
// `--session` 给了就按它落位（F3–F8 / C3.3 的冲突在目标层判）；没给就是"省略目标"，
// 走 2.2 的写法 ①（唯一就落，Q2 要的单会话不变）。
func dbgSessionID() (string, error) {
	sess, err := dbgTargetSession(nil, dbgRequest())
	if err != nil {
		return "", err
	}
	return sess.ID, nil
}

// dbgAmbientSession 是**尽力而为**版的 dbgSessionID：落不上目标时返回 nil，
// 由调用方决定"没有当前会话"怎么表达。
//
// 用途只有一处：`debug start` 启动前记下那个空闲会话，用来在输出里标出"复用了宿主"。
// 那件事是**痕迹性**的（`dbgReusedHost` 判不出来就什么都不标），它不该把
// 一条本来能成功的 `start` 判失败 —— 所以多会话/没会话时它是 nil，不是错误。
func dbgAmbientSession() *target.Session {
	sess, err := dbgTargetSession(nil, dbgRequest())
	if err != nil {
		return nil
	}
	return &sess
}

// dbgRequest 把三个权威标志 + 调用方手上的世代号装成一次寻址请求（契约 C3.1）。
//
// **世代号这一位今天仍然是空的**（F15 因此不校验），但原因变了：服务端从 WP4c 起
// **已经产出** `session.epoch`（C5.3），缺的是 CLI 这一侧的持有方式 —— 每次 `tt` 调用
// 都是一个新进程，没有地方存"上一次拿到的那一代"。谁要跨轮做 F15 的失效判定，
// 就得把上一次信封里的 `session.epoch` 原样带回来填进这里（接口已经留好：
// `target.Request.Epoch`，空 = 不校验）。**不要**为了让它有值去编一个。
func dbgRequest() target.Request {
	return target.Request{
		Selector: dbgSelector(),
		Env:      dbgTargetEnv(), // `--env` / `--conn` 权威；`--ssh` 由各命令自己并进 dbgEnvFlag
		Ent:      dbgEntFlag(),
	}
}

// dbgSelector 把 `--session` 的取值解析成选择器（F9–F11 在**发请求之前**判定，C2.8）。
//
// 没给这个标志 → nil（= "省略"，写法 ①，不是空串：F11 明写空串按文法非法处理）。
// 解析失败返回 nil 并把失败挂在 dbgSelectorFailure 上 —— 这样 dbgRequest 不必
// 把错误往上传（调用链上每一处都只关心"要哪条会话"这一件事）。
func dbgSelector() *target.Selector {
	dbgSelectorFailure = nil
	raw := strings.TrimSpace(dbgSessionFlag)
	if raw == "" {
		return nil
	}
	sel, fail := target.ParseSelector(raw)
	if fail != nil {
		dbgSelectorFailure = fail
		return nil
	}
	return &sel
}

// dbgSelectorFailure 是 dbgSelector 解析出来的文法失败（F9/F10/F11）。
//
// 它是"一次调用里最多一条"的临时槽：dbgTargetSession 一进来就把它取走并转成错误，
// 不让它活到第二次寻址。用包级变量而不是返回值，是为了让 dbgRequest 这条
// "装请求"的路保持无错误返回（与旧入口一样好接）。
var dbgSelectorFailure *target.Failure

// dbgEntFlag 取 `--ent` 的权威取值（契约 C3.1：它是**企业**的权威）。
//
// 从前这里恒返回 ""（"会话级命令今天没有 --ent"），于是目标层为它准备的两种冲突判据
// （F13：`--session` 里的企业 vs `--ent`；F14：落到的会话实际企业 vs `--ent`）**永远触发不了**
// —— 那两条规则只活在文档里。现在 start / wsdebug 都有这个标志，判据跟着活过来。
//
// 其余命令（exec / quit / topent …）仍然没有它：它们不改变"这一轮跑在哪个企业"，
// 企业在那条会话建立时就已经钉住了。
func dbgEntFlag() string { return strings.TrimSpace(dbgEntRun) }

// dbgTargetError 是目标层失败在 CLI 这一侧的**信封**（契约 5.2 / C4.1 / C4.2）。
//
// 它同时满足三件事：
//
//   - 实现 `ExitCode()` → 根命令（`internal/cli/version.go` 的 exitCodeOf）按它退码；
//   - 实现 `AlreadyReported()` → 根命令不再打第二份信封（本类型自己打 JSON，见 Write）；
//   - 包着 `target.Failure` → `code` / `hint` / `candidates` 全从契约那一处来。
//
// 为什么不直接用 `internal/output.Error`：那个类型没有 `candidates` 字段，而
// C5.11 要求五种码下候选块**必须出现**（F1/F3/F4/F5/F7/F8/F15/F18/F20）。
// 本类型是"output.Error 的形状 + 候选清单"，`Resolve` 之外的地方仍用 output.Error。
type dbgTargetError struct {
	fail *target.Failure
	// newSyntax 见 Q15 的**第二半**：真用了新写法（`--session` / `--name`）。
	// 分档的主判据在 `fail.NewScenario`（"这个场景是不是新场景"），不是这一位 ——
	// 见 `dbgNewSyntax` 与 `target.ExitFor` 的注释。
	newSyntax bool
}

// dbgTargetErrorOf 把目标层的失败包成可退出的 CLI 错误。
func dbgTargetErrorOf(fail *target.Failure) *dbgTargetError {
	return &dbgTargetError{fail: fail, newSyntax: dbgNewSyntax()}
}

// Error 是人话那一行（契约 2.4 的"文案形状"：`错误：…` + 空行 + 候选块 + 提示）。
//
// **为什么不带 `错误：` 前缀**：根命令（`internal/cli/root.go` 的 reportError）在
// 非 JSON 模式下自己会打 `错误: %v` —— 前缀写在这里就变成 `错误: 错误：…`。
// 所以这里给的是"正文"，前缀与 `提示:` 那一行由根命令加（那是它一贯的做法）。
func (e *dbgTargetError) Error() string {
	var b strings.Builder
	b.WriteString(e.fail.Message)
	if lines := e.fail.CandidateLines(); len(lines) > 0 {
		b.WriteString("\n\n候选（照抄 <ref> 那一列重发）：\n")
		b.WriteString(strings.Join(lines, "\n"))
	}
	return b.String()
}

// ExitCode 给退出码（契约 4.1 / 4.2 的定稿；判据是"落在新场景里"，见 target.ExitFor）。
//
// 两个入参各管一半：`fail.NewScenario` 管"今天不可能发生的场景"（无条件走 4.1.1），
// `newSyntax` 管"既有场景 + 真用了新写法"（4.2 的 H2 表逐行写着这条两档）。
func (e *dbgTargetError) ExitCode() int {
	return target.ExitFor(e.fail.Code, e.fail.NewScenario, e.newSyntax)
}

// ErrorMessage 是**机器可读**那一份 `error` 字段（契约 5.2 的信封示例）。
//
// 它与 `Error()` 刻意不同：信封里的 `error` 只有那句正文（`"有 2 条 SSH 会话，
// 无法确定你要哪一条"`），候选在 `candidates`、建议在 `hint` —— 各就各位。
// 人话那一路才需要把候选块拼进同一段文字（终端上没有第二个字段可放）。
func (e *dbgTargetError) ErrorMessage() string { return e.fail.Message }

// WriteEnvelope 让根命令把这条失败**按本类型的形状**打出去（见 internal/cli/root.go
// 的 `jsonErrorWriter` 接口）：`ok` / `code` / `error` / `hint` / `exitCode`，
// 外加 C5.11 那五种码下的 `candidates`（键集 = C5.12 的九键，唯一出处是 `internal/target`）。
//
// 为什么不复用 `output.WriteError`：它没有 `candidates` 字段，而候选清单是 F1
// 这类失败**必须**给的东西（"请指明其一"没有候选就无从指明）。形状的其余部分
// 与它逐字一致，所以读信封的那一方不必学第二种形状。
func (e *dbgTargetError) WriteEnvelope(w io.Writer) error {
	return output.WriteJSONValue(w, target.FailureEnvelope{
		OK:       false,
		Code:     e.fail.Code,
		Error:    e.ErrorMessage(),
		Hint:     e.fail.Hint,
		ExitCode: e.ExitCode(),
		Candidates: func() *[]target.Candidate {
			if !target.CodeRequiresCandidates(e.fail.Code) {
				return nil
			}
			lines := e.fail.Candidates
			if lines == nil {
				lines = []target.Candidate{}
			}
			return &lines
		}(),
	})
}

// dbgServerError 是**服务端失败信封**在 CLI 这一侧的落地（契约 C5.15 / 4.1.1 / Q15）。
//
// 为什么要它：从 WP4c 起，服务端在**所有**失败信封上给稳定的 `code`（C5.15），
// 而 F18（`SESSION_NAME_TAKEN`）/ F20（`SESSION_TARGET_BUSY`）这两个判据**住在服务端** ——
// 别名是服务端的一条事实（D1：服务端是目标的唯一真相源），CLI 不能自己拍。
// 于是退出码必须由 CLI 按 **code** 分档，而不是按"HTTP 是不是 4xx"猜一个数字。
//
// 分档口径与 `dbgTargetError` 同一套（Q15 的"分档生效"，判据是**场景**不是标志）：
//
//	① **新场景**（今天不可能发生的那些）：无条件按 4.1.1 的码表退 —— 判据是
//	   `target.CodeRequiresCandidates(code)`，它列的那五种码恰好就是契约里
//	   "今天不可能发生"的那五族（F1/F3–F5/F7/F8/F15/F18/F20）。多会话歧义（F1）尤其要点名：
//	   Agent 恰恰是在**没打** `--session` 时才撞上它。
//	② **既有场景**（0 条会话 / 环境名写错 / 参数错 / 会话状态不符 / 运行失败）：
//	   维持今天的码（1），只有**真用了新写法**（命令行上出现了 `--session` / `--name`）
//	   才按 4.1.1 升档（契约 4.2 的 H2 表逐行写着这两档：F2 带 `--session` → 3、
//	   F6 写在 `--session` 里 → 2）。
type dbgServerError struct {
	// code 是服务端给的机器码（空 = 对面是个不认识 code 的老服务端或非信封错误）。
	code string
	// msg 是信封里的 `error` 正文（**不带**候选块：候选在 candidates，见 WriteEnvelope）。
	msg string
	// candidates 是 C5.12 的九键候选清单，服务端给什么透传什么（C5.11 的政策在服务端判）。
	candidates []target.Candidate
	// exit 见 dbgServerErrorOf。
	exit int
}

// dbgServerErrorOf 把服务端失败信封转成可退出的 CLI 错误。
func dbgServerErrorOf(code, msg string, cands []target.Candidate) error {
	if strings.TrimSpace(code) == "" {
		// 对面没给码（老服务端 / 非 debug 面的失败）：保持今天的行为 —— 一句散文、退 1。
		return fmt.Errorf("%s", msg)
	}
	return &dbgServerError{
		code: code, msg: msg, candidates: cands,
		exit: target.ExitFor(code, target.CodeRequiresCandidates(code), dbgNewSyntax()),
	}
}

// Error 是人话那一行：正文 + 候选块（照 C5.14 渲染）。
//
// **与信封里的 `error` 字段刻意不同**：信封只有那句正文（候选在 `candidates`），
// 终端上没有第二个字段可放，所以人话这一路把候选块拼进同一段文字 —— 与
// `dbgTargetError.Error()` 的形状一致（两处形状不同会让同一个失败看起来像两种）。
func (e *dbgServerError) Error() string {
	var b strings.Builder
	b.WriteString(e.msg)
	if lines := dbgCandidateLines(e.candidates); len(lines) > 0 {
		b.WriteString("\n\n候选（照抄 <ref> 那一列重发）：\n")
		b.WriteString(strings.Join(lines, "\n"))
	}
	return b.String()
}

// ExitCode 给退出码（契约 4.1.1 那一档；分档见 dbgServerErrorOf）。
func (e *dbgServerError) ExitCode() int { return e.exit }

// ErrorMessage 是信封里的 `error` 字段（只有正文）。
func (e *dbgServerError) ErrorMessage() string { return e.msg }

// WriteEnvelope 让根命令按**契约 5.2 的形状**打这条失败（`internal/cli/root.go` 的
// `jsonErrorWriter`）：`ok` / `code` / `error` / `exitCode`，`hint` 有才出现，
// `candidates` 只在 C5.11 那五种码下出现（判据用目标层的唯一清单，
// **不在这里再抄一遍码表**）。
//
// 这样才能把服务端给的候选清单原样交给调用方：F20 的"占用者那一条"与 F1 的候选
// 是同一件事，缺了它就只剩一句"别名指向的会话不能复用"，而那句话没法照着重发。
func (e *dbgServerError) WriteEnvelope(w io.Writer) error {
	return output.WriteJSONValue(w, target.FailureEnvelope{
		OK:       false,
		Code:     e.code,
		Error:    e.ErrorMessage(),
		ExitCode: e.ExitCode(),
		Candidates: func() *[]target.Candidate {
			if !target.CodeRequiresCandidates(e.code) {
				return nil
			}
			list := e.candidates
			if list == nil {
				list = []target.Candidate{}
			}
			return &list
		}(),
	})
}

// dbgCandidateLines 按 C5.14 渲染人话候选块（唯一实现是 `target.CandidateLine`）。
func dbgCandidateLines(cands []target.Candidate) []string {
	if len(cands) == 0 {
		return nil
	}
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, target.CandidateLine(c))
	}
	return out
}

// dbgResolveSessionFlag 在**起会话**之前把 `--session` 落位一次，并把落上的那一条回出去。
//
// 为什么要独立一步：`debug start` 的目标是"要起的那条会话"，而 `--session` 是"落在
// 哪条现有会话上"（Q23 的复用宿主）—— 它给了就先把那条会话找到（落不上当场报，见
// F3–F8，且**不发那个 POST**）。会话级命令不需要这一步：它们自己那次 `dbgSessionID`
// 就是落位，重复一次白跑一趟 HTTP。
//
// # 为什么要把落位结果**回出去**（Bug A）
//
// 从前这里只判"落得上/落不上"，落上了却把结果丢掉 —— POST 的 body 里没有目标，
// 于是服务端仍按"环境+名字"收窄候选：同环境有 2 条空闲会话时，本地校验通过、
// 服务端照样报 `SESSION_AMBIGUOUS`，`--help` 里写着的这个标志等于帮不上忙。
// 现在调用方拿到那条会话本身，把它的**服务端原始句柄**（C5.4：只认 id）随请求体送过去，
// 服务端 `prepareSession` 于是不再有"候选集"这一步 —— 歧义连成立的机会都没有。
//
// 空标志 → (nil, nil)：没打这个标志时调用方一个字节都不多送（H2）。
func dbgResolveSessionFlag() (*target.Session, error) {
	if strings.TrimSpace(dbgSessionFlag) == "" {
		return nil, nil
	}
	sess, err := dbgTargetSession(nil, dbgRequest())
	if err != nil {
		return nil, err
	}
	return &sess, nil
}

// dbgRegisterSessionFlag 给一组命令挂上 `--session`（契约 1. 的命令角色表：
// 会话级操作命令认它；`tt debug status` / `sql` / `wslogs` / `wsdebug` 不认它）。
func dbgRegisterSessionFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&dbgSessionFlag, "session", "",
			"指定 SSH 会话（Q9：只寻址 SSH 会话）：别名 / <环境>#<序号> / <环境>/<作业>[#<序号>] / <环境>/<企业编号> / <环境>；不给则唯一就落（多条时报错并列候选）")
	}
}

// dbgRegisterNameFlag 给**起会话**的命令挂上 `--name`（契约 C3.5 / C2.4 / F19）。
//
// 它做两件事，顺序写死（C2.8 的同一条纪律：一个字节都不该白花）：
//
//  1. **取值纪律的校验**（非空、1–64 字符、只排除 `/` 与 `#`）在**发请求之前**判 ——
//     不合法时连 POST 都不发（F19）；
//  2. 合法就**随请求体送给服务端**（`debug start` 的 body["name"]、`tt debug wsdebug` 的
//     body["name"]，两条都是**起会话**的路）—— 别名从那一刻起是**服务端的一条事实**
//     （`SessionBrief.name`，C5.3）：它参与寻址（`--session <别名>`）、当幂等键
//     （Q23：空闲复用 / 忙碌报 `SESSION_TARGET_BUSY`）、随会话消失即释放（Q19）。
//
// 为什么不是 CLI 自己存一份映射表：D1 写死了"服务端是目标的唯一真相源"，
// 本地缓存会漂移，而漂移的表现恰好是"静默落错"。
func dbgRegisterNameFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&dbgNameFlag, "name", "",
			"给这条 SSH 会话起别名（契约 C2.4/C2.5：非空、1–64 字符、只排除 / 与 #）。别名参与寻址，跨轮追踪用它而不是序号（Q20/C3.7c）")
	}
}

// dbgValidateNameFlag 校验 `--name` 的取值（F19 / C2.4 / C2.5 / Q17）。
//
// 落点刻意放在"发请求之前"：别名不合法时一个字节都不该发给服务端
// （与 F9–F11 同一条纪律，C2.8）。
func dbgValidateNameFlag() error {
	raw := strings.TrimSpace(dbgNameFlag)
	if raw == "" {
		return nil
	}
	if fail := target.ValidateAlias(raw); fail != nil {
		return dbgTargetErrorOf(fail)
	}
	return nil
}

// dbgSessionURL 拼会话级 REST 路径（C5.4：路径**只认** `id`，不认 ref/display）。
//
// 单独一个函数是为了让"路径只认原始句柄"这条规矩只有一个落点 ——
// 拿 ref（`恒烁正式区#1`）去拼路径是本契约点名的错法。
//
// id 为空时返回空串：那种情况下打出去的一定是 `/api/sessions/…` 这种少了 id 的路径，
// 服务端会当成"不存在的会话"报 404；空串让调用方能一眼看出"这次根本没拿到目标"
// （而不是把它读成"服务端说不存在"）。
func dbgSessionURL(id, suffix string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	return "/api/sessions/" + url.PathEscape(id) + suffix
}
