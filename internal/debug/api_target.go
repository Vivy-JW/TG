package debug

// 服务端的**目标解析**（WP4b）与**别名/世代的判定**（WP4c）：契约 C7.1 五处
// `Manager.Current()` 兜底删掉之后，"这次要操作哪条会话"必须由**请求显式给出**，
// 或者由会话池**唯一确定**。
//
// # 钉住的条款
//
//   - **Q4 / C8.2「服务端不保留隐式寻址」**：服务端不再"没给目标就替你挑一条"。
//     从前那五处（`sessOfWriteGlobal`:367 / `hSessionSwitch`:816 / `hWSLogDebug`:1679 /
//     `wslogTopent`:1724 / `hDBSQL`:1754）直呼 `Manager.Current()`，而它取的是 map 里
//     **任意一条**（`internal/debug/manager.go` 的实现 + `session_pool_behavior_test.go`
//     的实测）—— 多会话下那是"随机落一条，而输出看起来一切正常"，也就是计划开头那句
//     要根除的话：**不会出现"命令跑在另一条会话上却看起来成功"；落不上目标时，一定是报错。**
//   - **C8.1「只在多会话时收紧」**：池里 **0 条或 1 条**时，"省略目标"照旧确定性地落位
//     （0 条 = 没有东西可保护；1 条 = 唯一，不必问）；**>=2 条**才按 F1 报错并给候选。
//     这条与 CLI 侧 `internal/cli/debug/target.go` 的"唯一就落"同源：**唯一**是确定性的，
//     不是"替调用方挑一条"。
//   - **C5.15 / C5.16**：失败信封在现行 `{"ok":false,"error":…}` 形状上**增加** `code`（必给，
//     通用失败那一半在 `fail()` 里给）与 `candidates`（只有会话寻址这几族判得出来），
//     `ok` / `error` 一个字都不改；缺目标/歧义用 **409 Conflict**。
//   - **C5.12**：候选行的键集是那**九键** —— 唯一出处是 `internal/target`（WP4c 起服务端
//     直接用那个类型，见下面 `sessionView`）。
//   - **C5.3 / Q22 / Q23 / F15 / F18 / F20**（WP4c）：`ref`/`display`/`name`/`ent`/`n`/`epoch`
//     由服务端自己产出；`--name` 的三种结局（空闲复用 / 忙碌 `SESSION_TARGET_BUSY` /
//     名字不存在起新的）与别名撞车（`SESSION_NAME_TAKEN`）都在这一层判。
//
// # 它不做什么
//
//   - **不复刻 `internal/target` 的选择器文法**：服务端只认它**自己**的原始句柄 `{id}`
//     与"池内唯一"这两种目标来源 —— "`--session` 选择器怎么解析"这件事住在 CLI 的目标层里
//     （Q12 / C8.2）。
//   - **但它用目标层的类型与码表**（WP4c 改）：`target.Candidate` / `target.Code*` /
//     `target.RefOf` / `target.ResolveName` 都是那个包的生产定义，服务端**不另立一套**
//     （C5.12 的"同一件规矩写两遍、改一处漏一处"）。这一层能这么做，是因为
//     `internal/target` **零 `tt/internal` 依赖**（只 import 标准库，见它的包注释）：
//     谁引用它都不成环。

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"tt/internal/target"
)

// sessionView 把一条服务端会话快照投影成目标层的**只读视图**（契约 C5.1 的十键）。
//
// 这是服务端这一侧唯一的适配点：`internal/target` 的 `Session` 是"一条 SSH 会话长什么样"
// 的唯一定义处（Q25），服务端不许另立第二份。WP4c 之前那五个键服务端还没有，
// 于是视图里是零值、由调用方现算；现在服务端自己产出它们（C5.3 / Q22）。
func sessionView(b SessionBrief) target.Session {
	return target.Session{
		ID: b.ID, Ref: b.Ref, Display: b.Display, Name: b.Name,
		Env: b.Env, Ent: b.Ent, N: b.N, Prog: b.Prog, State: b.State, Epoch: b.Epoch,
	}
}

// sessionViews 把整池快照投影成目标层视图（顺序即目标层的输入顺序；它的判定不依赖顺序）。
func sessionViews(briefs []SessionBrief) []target.Session {
	out := make([]target.Session, 0, len(briefs))
	for _, b := range briefs {
		out = append(out, sessionView(b))
	}
	return out
}

// failureEnvelope 是 **HTTP 侧**的失败信封（契约 C5.15）。
//
// 它与 `fail()` 那个形状（B8：`{"ok":false,"error":…}`）的关系是"**增加**"：
// `ok` 与 `error` 一字不改，多出来的只有 `code`（必给）与 `candidates`（按 C5.11）。
// 两个键在**所有**失败上都由 `fail()` / 本文件写出（C5.15 的"必给"），
// 差别只在 `candidates`：它是**会话候选清单**，只有会话寻址那几族失败才知道该列谁。
//
// `Candidates` 用指针是刻意的（与 `target.FailureEnvelope` 同一条理由）：C5.11 要求那五种
// 码下这个键**必须出现**（可能是空数组），而 `omitempty` 连空切片一起丢 ——
// 只有 `*[]target.Candidate` 能区分"不出现"与"出现且为空"。
type failureEnvelope struct {
	OK         bool                `json:"ok"`
	Code       string              `json:"code"`
	Error      string              `json:"error"`
	Candidates *[]target.Candidate `json:"candidates,omitempty"`
}

// resolveReasons 是"解析目标"的四种结局。
type resolveReasons int

const (
	// targetOK：拿到了一条确定的会话。
	targetOK resolveReasons = iota
	// targetEmptyPool：会话池里 0 条 —— **不是错误**（C7.1-1：没有东西可保护，放行）。
	//
	// 为什么它是"放行"而不是"报 NO_SESSION"：`hLaunch`（起会话）走的就是这条 ——
	// 池子空的时候恰恰是最需要能起会话的时候。若把它判失败，"第一条会话怎么来的"
	// 就没有答案了（C8.1 的单会话便利会当场死掉）。
	targetEmptyPool
	// targetMissing：调用方点名了一条会话，而它不在池里。→ 404 + `SESSION_NOT_FOUND`。
	targetMissing
	// targetAmbiguous：池里 >=2 条且调用方没点名。→ **409 + `SESSION_AMBIGUOUS` + 候选清单**（F1 / C5.16）。
	targetAmbiguous
)

// resolveTarget 在会话池上落一条**确定**的会话（C8.1：0 条或 1 条确定，>=2 条报错）。
//
// 两条来源，按顺序判：
//
//	① 点名的原始句柄（`targetID`，来自 URL 的 `{id}` 或请求体的 `session`）——
//	   它是**权威**：给了就只认它，找不到就报 `SESSION_NOT_FOUND`（不回落池子，
//	   否则"我点名的那条没了"会被静默换成另一条，正是要根除的那件事）；
//	② 省略目标 —— 池里**恰好一条**时落它（唯一即确定，与 CLI 的"唯一就落"同源）；
//	   0 条 → `targetEmptyPool`（放行）；>=2 条 → `targetAmbiguous` 并列出全部候选。
//
// `cands` 只在 `targetAmbiguous` 时非空（C5.11：候选只在那几种码下出现）。
func (s *Server) resolveTarget(targetID string) (sess *Session, reason resolveReasons, cands []target.Candidate) {
	if id := strings.TrimSpace(targetID); id != "" {
		if sess := s.mgr.Get(id); sess != nil {
			return sess, targetOK, nil
		}
		return nil, targetMissing, nil
	}
	briefs := s.mgr.Snapshot()
	switch len(briefs) {
	case 0:
		return nil, targetEmptyPool, nil
	case 1:
		return s.mgr.Get(briefs[0].ID), targetOK, nil
	default:
		return nil, targetAmbiguous, sessionsToCandidates(briefs)
	}
}

// mustTarget 要一条**确定的**会话,不对就当场写好失败信封并返回 false。
//
// 它把"落位 → 判模式闸门"这组动作收成一处:两个端点（切环境 / 按日志重放）都要
// "先落位、再按那一条判谁能写",两处各写一遍必然漂。
//
// 池子空时**报错**（`targetEmptyPool` → 409 + `NO_SESSION`）—— 但只在这些端点
// **真的要一条既有会话**时才成立。冷启动（池子空、也没点名）不是这种情况:那时
// 没有"要收掉的那一条",却有"要新建的那一条"。这种调用方用 `mustTargetOrEmpty`，
// 不要用这里。
//
// `what` 见 `failTarget`；`targetID` 是调用方点名的原始句柄（空 = 省略目标）。
func (s *Server) mustTarget(w http.ResponseWriter, r *http.Request, targetID, what string) (*Session, bool) {
	sess, reason, cands := s.resolveTarget(targetID)
	if reason == targetOK {
		if !s.sessOfWriteGlobal(w, r, sess) {
			return nil, false
		}
		return sess, true
	}
	failTarget(w, reason, cands, what)
	return nil, false
}

// mustTargetOrEmpty 是 `mustTarget` 的**冷启动档**：池子空且没点名时返回 (nil, true)。
//
// 为什么需要它（真机踩到的，两次、同一形状）：**要新建一条会话的端点**在池子空时撞上
// `mustTarget` 的 `NO_SESSION`，而池空恰恰是它们最正常的起点 ——
//   - 「连到该环境」= 创建一条会话（Web 上那个按钮就叫「创建会话」，命令行是
//     `tt debug env <环境>`）：报 `NO_SESSION` 等于让"先建会话、再设 ENT、再调试"
//     这条正确流程在**第一步**就断；
//   - 按接口日志重放（`hWSLogDebug` / `tt debug wsdebug`）= 用它自己建的那条会话重跑一遍
//     （`LaunchReplay` → `prepareSession`）：报 `NO_SESSION` 等于让人先去"起个作业"
//     凑一条会话，而重放**不需要**一条既有会话。
//
// 两处的错误提示都把人往"先运行 tt debug start"引，顺序被彻底反过来。
//
// 边界一字不让:
//   - **点名了**（targetID 非空）→ 交给 `mustTarget`：点名的不存在就是 `SESSION_NOT_FOUND`；
//   - **池里多条且没点名** → 也交给它：那是真歧义（"收哪一条"确实不明确）；
//   - 只有"池空 + 没点名"这一种才放行，返回 nil —— 调用方原本就写着 `if cur != nil`。
func (s *Server) mustTargetOrEmpty(w http.ResponseWriter, r *http.Request, targetID, what string) (*Session, bool) {
	if strings.TrimSpace(targetID) == "" && len(s.mgr.Snapshot()) == 0 {
		// 没有目标可保护，但写闸门仍要过（nil 目标 = 放行，见 sessOfWriteGlobal 的注释）。
		if !s.sessOfWriteGlobal(w, r, nil) {
			return nil, false
		}
		return nil, true
	}
	return s.mustTarget(w, r, targetID, what)
}

// 同一份池子两次跑出的文案必须一样（否则报错信息没法当判据用）。
//
// "按 id 排序"是刻意的：`Manager.Snapshot()` 走的是 map 遍历，顺序本身就是随机的；
// 不排一下，同一次失败在两次请求里会列出两种顺序。
//
// WP4c：候选行改用 `internal/target` 的 `Candidate`（C5.12 的九键**唯一出处**），
// 投影走 `target.Session.Candidate()` —— 它把 `ref` / `display` 的兜底规则也一并继承了
// （服务端没给 ref 时退回原始句柄，绝不给空串）。
func sessionsToCandidates(briefs []SessionBrief) []target.Candidate {
	out := make([]target.Candidate, 0, len(briefs))
	for _, b := range briefs {
		if strings.TrimSpace(b.ID) == "" {
			continue
		}
		out = append(out, sessionView(b).Candidate())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// failTarget 把"要一条会话却拿不到目标"这一族写成 C5.15 / C5.16 的失败信封。
//
//   - `targetAmbiguous` → **409** + `SESSION_AMBIGUOUS` + 候选清单（F1 的形状）；
//   - `targetMissing`   → **404** + `SESSION_NOT_FOUND`（指名的那条不在池里）；
//   - `targetEmptyPool` → **409** + `NO_SESSION`（调用方这一步**必须**有一条会话才做得下去）。
//     **今天没有端点产出这一档**：要新建会话的两条端点（切环境 / 按日志重放）都走
//     `mustTargetOrEmpty`（见它的注释）。这一支仍留着，是因为 `resolveTarget` 的四种结局
//     与四种信封一一对应 —— 将来新增一条真的"必须有既有会话"的端点，拿 `mustTarget`
//     得到的码当场就是对的。形状由 `TestContractNoSessionEnvelopeIsStillPinned` 钉住。
//   - `targetOK`        → 调用方判错了状态，如实说一句（不当成静默成功）。
//
// `what` 是"哪个端点在哪一步要目标"和（点名时的）那个句柄，两样都要进文案：
// 读错误的人得能一眼看出是切换环境、按日志重放、还是查库那一步缺目标。
func failTarget(w http.ResponseWriter, reason resolveReasons, cands []target.Candidate, what string) {
	what = strings.TrimSpace(what)
	switch reason {
	case targetAmbiguous:
		list := cands
		if list == nil {
			list = []target.Candidate{}
		}
		msg := what + "：有 " + strconv.Itoa(len(list)) + " 条 SSH 会话，无法确定你要哪一条（请带上目标会话的 id）"
		writeJSON(w, http.StatusConflict, failureEnvelope{
			OK: false, Code: target.CodeAmbiguous, Error: msg, Candidates: &list,
		})
	case targetMissing:
		writeJSON(w, http.StatusNotFound, failureEnvelope{
			OK: false, Code: target.CodeNotFound,
			Error: "会话不存在: " + what,
		})
	case targetEmptyPool:
		writeJSON(w, http.StatusConflict, failureEnvelope{
			OK: false, Code: target.CodeNoSession,
			Error: what + "：没有 SSH 会话 —— 先运行 tt debug start <作业> --env <环境> 起一条",
		})
	default:
		writeJSON(w, http.StatusConflict, failureEnvelope{
			OK: false, Code: target.CodeState,
			Error: "这次请求要的会话状态不对: " + what,
		})
	}
}

// writeCandidateFailure 把一条**带候选清单**的失败写成 C5.11 / C5.15 的信封。
//
// 用它的是**会话寻址**那几族（别名 / 世代）：它们的失败天然有一条"占用者/当前代"的候选，
// 而没找到的那族（F19 取值非法之类）不带候选 —— 判据是 `target.CodeRequiresCandidates`
// （C5.11 的唯一出处），不是在这里再抄一遍码表。
func writeCandidateFailure(w http.ResponseWriter, status int, code, msg string, cands []target.Candidate) {
	e := failureEnvelope{OK: false, Code: code, Error: msg}
	if target.CodeRequiresCandidates(code) {
		list := cands
		if list == nil {
			list = []target.Candidate{}
		}
		e.Candidates = &list
	}
	writeJSON(w, status, e)
}

// writeLaunchFailure 是**造会话那两条路**（`hLaunch` / `hWSLogDebug`）的错误出口。
//
// 三种失败各给各的信封，返回是否已写（写了调用方就 return）：
//
//	① F18（`ErrNameTaken`）：别名在"判定 → 登记"之间被别的请求抢走。它**不是**
//	   TARGET_BUSY（4.1.1 明写退 2 不退 3），单独一份信封；
//	② **F1 / F20（WP6 新增的这一支）**：`prepareSession` 落不上目标 —— 多会话 + 没点名要
//	   哪条宿主 → 歧义并列候选（`SESSION_AMBIGUOUS` + 退 3 + C5.12 九键），或者点名的别名
//	   不可复用（`SESSION_TARGET_BUSY`）。这两条**必须**带 `code` 与候选清单，
//	   不许降级成一句散文（C5.11：那五种码下候选**必须出现**）；
//	③ 其余（拨号失败之类的运行失败）：维持今天那一档，由调用方的 `fail(w, 409, err)` 写。
//
// 收成一个函数是为了让"这两种特殊信封"只有一个落点：两条路各写一遍必然漂。
func (s *Server) writeLaunchFailure(w http.ResponseWriter, err error) bool {
	var taken *ErrNameTaken
	if errors.As(err, &taken) {
		s.writeNameTaken(w, taken)
		return true
	}
	return writeTargetFailure(w, err)
}

// writeTargetFailure 把一条目标层失败写成 C5.15 / C5.16 的信封并返回 true；
// 不是目标层失败时返回 false，由调用方按它自己那一档报。
//
// 用途（WP6）：**造会话那两条路**（`hLaunch` / `hWSLogDebug`）的失败从前一律是
// "409 + 一句散文"（`fail(w, 409, err)`），调用方拿不到 `code`、更拿不到候选清单。
// 而 `prepareSession` 现在会报 F1 的歧义（多会话 + 不给目标 → `SESSION_AMBIGUOUS`
// + 候选九键），那条失败**必须**带候选 —— 否则调用方只知道"有 N 条会话"却不知道是哪几条，
// C5.11 的"必须出现候选"当场落空。
//
// 状态码取 409（C5.16）：与"会话状态不符、缺目标、歧义"同一档 —— 请求体是合法 JSON，
// 冲突的是**会话池当前的状态**。
func writeTargetFailure(w http.ResponseWriter, err error) bool {
	var f *target.Failure
	if !errors.As(err, &f) || f == nil {
		return false
	}
	writeCandidateFailure(w, http.StatusConflict, f.Code, f.Message, f.Candidates)
	return true
}

// beginNamedLaunch 判 `--name` 的三件判定，返回"要带下去的名字"与是否放行（Q23 / C1.6 / F19）。
//
// 判定顺序（写死，因为它决定"同时踩两条规矩时报哪一条"）：
//
//	① **取值纪律**（F19 → 400 + `USAGE`）：非空、1–64 字符、只排除 `/` 与 `#`（Q17）。
//	   纯函数，判在**发请求之前**那一半（C2.8）由 CLI 做；服务端这一份是防线，不是重复。
//	② **别名的三种结局**（Q23，判据用目标层的 `ResolveName`，与 CLI 同源）：
//	   - `NameFree`  → 放行（名字不存在 ⇒ 起新的 / 落在这条宿主上）；
//	   - `NameReuse` → 放行（那条会话**空闲** ⇒ 复用它：同一条会话、同一个 ref、同一个 n）；
//	   - `NameBusy`  → **409 + `SESSION_TARGET_BUSY` + 候选（占用者那一条）**，退 3。
//
// 别名的作用域是**整个会话池**（目标层的 `ResolveName` 就是这么写的）：它跨环境查重，
// 于是不会出现"起的时候说没撞、找的时候找到两条"。
func (s *Server) beginNamedLaunch(w http.ResponseWriter, rawName string) (string, bool) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		return "", true
	}
	if f := target.ValidateAlias(name); f != nil {
		writeCandidateFailure(w, http.StatusBadRequest, f.Code, f.Message, nil)
		return "", false
	}
	// Q19 / C3.7b：别名随会话消失即释放 —— 判"名字有没有人用"之前先把已断开的残留清掉。
	// 不清的话，一条早就掉了线的会话会继续占着名字，重起同名被拒（那个人人都会踩的坑）。
	s.mgr.SweepExited()
	outcome, _, f := target.ResolveName(sessionViews(s.mgr.Snapshot()), name)
	switch outcome {
	case target.NameReuse, target.NameFree:
		return name, true
	default: // NameBusy（F20 / Q23）：别名指向的那条会话正在跑 / 已停站，不能复用
		writeCandidateFailure(w, http.StatusConflict, f.Code, f.Message, f.Candidates)
		return "", false
	}
}

// writeNameTaken 写 F18（别名已被占用）的信封：**409 + `SESSION_NAME_TAKEN` + 占用者那一条**。
//
// 4.1.1 明写它归"用法错"那一行（**退 2 不退 3**）：服务端这一侧只给码与候选，
// 退出码由调用方按 `target.ExitFor` 分档（CLI 侧读 `code`，不读数字：Q15）。
//
// 它落在这条路上只有一个原因：**竞态**。Q23 的三种结局已经判过一次，但"判定 → 登记"
// 之间有窗口；晚到的那一个请求必须报错，不许把名字从别人手里抢过来（那会让先到的
// 调用方手上的别名当场指到另一条会话上）。
func (s *Server) writeNameTaken(w http.ResponseWriter, taken *ErrNameTaken) {
	var cands []target.Candidate
	if occ := s.mgr.Get(taken.Occupant); occ != nil {
		for _, b := range s.mgr.Snapshot() {
			if b.ID == occ.ID {
				cands = []target.Candidate{sessionView(b).Candidate()}
				break
			}
		}
	}
	writeCandidateFailure(w, http.StatusConflict, target.CodeNameTaken, taken.Error(), cands)
}
