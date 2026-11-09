package target

import "strconv"

// Session 是一条 SSH 会话的**只读视图**（契约 C5.1 / C5.3 / C5.4 的身份键）。
//
// 它是本包唯一认识的"会话"形状：**不是** `internal/debug` 的 `Session`。
// 谁把 HTTP 的 JSON 转成它，谁就负责取目标 —— 本包只管在这些字段上做纯函数判定。
//
// # 服务端还没吐的五个键（**不是本包的缺陷**）
//
// `ref` / `display` / `name` / `n` / `epoch` 这五个键今天**不在** `SessionBrief`
// （`internal/debug/manager.go`）上，服务端补它们是 WP5/WP6 的活：
//
//   - `ref`（规范 id `环境#序号`，C2.0 恒定）与 `display`（`环境/作业#序号`，作业段可漂）；
//   - `name`（别名，C2.5b 生命周期内不变）；
//   - `n`（序号，C2.2/Q24 按环境自增）；
//   - `epoch`（世代号，F15 的失效判据）。
//
// 这里先把位置留好：字段齐了，服务端一补就通（`json.Unmarshal` 直接吃）。
// 在那之前 `CanonicalRef` / `DisplayForm` 会用 `Env` + `N` + `Prog` 现算一个 ——
// 这样错误文案与审计行**今天**就能打印出稳定的规范 id，而不是一片空白。
type Session struct {
	// ID 是服务端的**原始句柄**（今天形如 `s<UnixMilli>`，B9）。REST 路径 `{id}` 用它（C5.4）。
	ID string `json:"id"`
	// Ref 是**规范 id**：`环境#序号`。生命周期内恒定不变（C2.0）。
	// 机器对账、错误文案、审计行用它（C5.4）。服务端还没吐这个键（WP5/WP6 补）。
	Ref string `json:"ref"`
	// Display 是**人读形态**：`环境/作业#序号`。它比 `ref` 多一段"当前作业"快照，
	// 所以作业中途变了它就会变 —— **它不能当地址**（C5.4：要恒定用 ref，要好读用 display）。
	Display string `json:"display"`
	// Name 是别名（`--name` 给的那个）。空串 = 未命名。生命周期内不变（C2.5b）。
	Name string `json:"name"`
	// Env 是环境名（`config.json` 的 `hosts.sshs[].name` 精确取值，C2.1）。
	// 它是地址的一部分，会话生命周期内不改（2.1.2）。
	Env string `json:"env"`
	// Ent 是该会话**当前**生效的企业编号（登录后由 shell 的 `$TOPENT` 定，可被 SetTopent 改写）。
	// **它是快照、不是地址**（C2.5c）：不许拿它推导地址或序号。空 = 未取到。
	Ent string `json:"ent"`
	// N 是序号：**同一环境下**的第 N 条 SSH 会话，从 1 起（C2.2 / Q24）。建立时分配，之后不变。
	N int `json:"n"`
	// Prog 是当前作业（快照）。`display` 的作业段就是它，作业中途会变（C2.5c）。
	Prog string `json:"prog"`
	// State 见 B30 的五个取值（StateIdle / StateLoading / StateStopped / StateRunning / StateExit）。
	State string `json:"state"`
	// Epoch 是世代号：会话每次"重新登录/重启"递增（C5.3）。F15 的失效判据用它。空 = 不校验。
	Epoch string `json:"epoch"`

	// 它**不是** `kind`（`kind` 是 `ssh`）。

}

// CanonicalRef 返回这条会话的**规范 id**（C2.0 / C5.4）：服务端给了 `ref` 就用它，
// 没给（今天就是）就用 `环境#序号` 现算；两者都取不到时退回原始句柄，绝不给空串
// —— 错误文案与审计行里一个空字段比一个句柄更难查。
func (s Session) CanonicalRef() string {
	if s.Ref != "" {
		return s.Ref
	}
	if s.Env != "" && s.N > 0 {
		return RefOf(s.Env, s.N)
	}
	return s.ID
}

// DisplayForm 返回人读形态（C5.4）：服务端给了 `display` 就用它，
// 没给就按 `环境/作业#序号` 现算（作业段取不到时退成 `环境#序号`）。
func (s Session) DisplayForm() string {
	if s.Display != "" {
		return s.Display
	}
	if s.Env == "" || s.N <= 0 {
		return s.CanonicalRef()
	}
	if s.Prog == "" {
		return RefOf(s.Env, s.N)
	}
	return s.Env + SelectorSep + s.Prog + SeqSep + strconv.Itoa(s.N)
}

// RefOf 拼一个规范 id：`环境#序号`（C2.1.2）。序号由**服务端**在建立时算一次（D1），CLI 不自己数。
func RefOf(env string, n int) string { return env + SeqSep + strconv.Itoa(n) }

// Candidate 是候选清单的一行（契约 C5.12 的**九键，唯一出处**）。
//
// 它是 `session` 块的真子集：同一个渲染函数既打人话候选块（C5.14）又给机器读。
type Candidate struct {
	ID      string `json:"id"`
	Ref     string `json:"ref"`
	Display string `json:"display"`
	Name    string `json:"name"`
	Env     string `json:"env"`
	Ent     string `json:"ent"`
	N       int    `json:"n"`
	Prog    string `json:"prog"`
	State   string `json:"state"`
}

// Candidate 把一条会话投影成候选行（C5.4b：同一个 struct 的投影，**不许手抄第二份 JSON 定义**）。
func (s Session) Candidate() Candidate {
	return Candidate{
		ID: s.ID, Ref: s.CanonicalRef(), Display: s.DisplayForm(), Name: s.Name,
		Env: s.Env, Ent: s.Ent, N: s.N, Prog: s.Prog, State: s.State,
	}
}

// Pool 是目标层看到的**会话池 + 可用环境**只读视图。
//
// 两个字段来自两个不同的端点：`Sessions` 是 `GET /api/sessions`，
// `Environments` 是配置里的环境清单（F6 的"可用环境：…"那句用它）。
type Pool struct {
	// Environments 是配置里**全部**环境名（顺序即展示顺序）。
	Environments []string
	// Sessions 是会话池的当前快照。
	Sessions []Session
}

// Request 是一次寻址请求：选择器 + 三个权威标志 + 调用方手上的世代号。
type Request struct {
	// Selector 是 `--session` 解析出来的选择器；**nil = 没给这个标志**（省略，"唯一就落"）。
	//
	// 为什么用指针区分"没给"与"给了空串"：F11 明写空串按**文法非法**处理、不当成省略。
	// 空串在 `ParseSelector` 那一步就被拒了，所以到了这里要么是 nil（省略）、
	// 要么是一个合法的选择器。
	Selector *Selector
	// Env 是 `--env`（含 `--conn` / `--ssh`）的权威取值（空 = 没给）。C3.1：它是**环境**的权威。
	Env string
	// Ent 是 `--ent` 的权威取值（空 = 没给）。C3.1：它是**企业**的权威。
	Ent string
	// Epoch 是调用方手上的世代号（空 = 不校验，F15）。
	//
	// 用途：会话被复用/重启后旧句柄当场失效。调用方把上一次拿到的 `session.epoch`
	// 原样带回来，与服务端当前值一比就知道手上这份是不是上一代。
	Epoch string
}
