package debug

// T2.2 表征测试（CLI 侧）：CLI"向目标层要目标"这件事的**边界**。
//
// # 这条测试的来历（WP4a 改写，改动理由在这里）
//
// WP2 写的是同一族的另一条：它把"CLI 取当前会话 id 的那个旧入口**取回包数组第一项**"
// 这个行为逐条钉住（三条断言：顺序决定答案 / 不看状态 / 空列表报那句现行文案），
// 并在文件末尾写着"改掉之后请**删除或改写**这条测试，而不是让它红着"。
//
// WP4a 把那个入口整个删掉、换成"先列会话再让目标层落位"，所以这条测试按它自己的
// 交代**改写**：从"钉住取第一项"改成"钉住 Q2 要的那两件事"——
//
//  1. **单会话不许变**（Q2 / C8.1 / 写法 ①）：池里恰好一条时照旧成功，拿到它的 id；
//  2. **多会话且没给目标 → 报错并列候选**（F1），不许取第一个：
//     同一个桩、只把列表换成两条，旧写法给 id、新写法给 `SESSION_AMBIGUOUS`。
//
// 第 2 条正是"行为真的该变"的那一条（本次重构第一次改行为），也是"顺序不再是答案"
// 的机器可读表述 —— 旧写法换个顺序就换条会话，新写法两种顺序都报同一句错。
//
// 为什么用 HTTP 桩而不是真服务：这条链路的全部行为就是"列会话 + 列环境 → 落位"，
// 一个桩服务器就能把这两句话变成断言，而且**不依赖任何真实环境**
// （本仓库的既有做法见 actor_test.go 的 `dbgAPIURL`）。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/target"
)

// sessionPoolStub 起一个回答 `/api/sessions` 与 `/api/hosts` 的桩。
//
// 两个端点都要：目标层落位时，"省略目标"要判唯一、给选择器时要判环境存在（F6），
// 所以它要同时拿到会话池与可用环境清单（契约 `target.Pool` 的两个字段）。
func sessionPoolStub(t *testing.T, sessions []map[string]any, envs ...string) {
	t.Helper()
	if envs == nil {
		envs = []string{"恒烁正式区"}
	}
	sshs := make([]map[string]any, 0, len(envs))
	for _, e := range envs {
		sshs = append(sshs, map[string]any{"name": e})
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/api/hosts") {
			_ = json.NewEncoder(w).Encode(map[string]any{"sshs": sshs})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
	}))
	t.Cleanup(ts.Close)

	oldURL := dbgAPIURL
	dbgAPIURL = ts.URL
	t.Cleanup(func() { dbgAPIURL = oldURL })
}

// resetTargetFlags 清掉两个寻址标志与根命令的 `--env`，保证每条用例从"什么都没给"起跑。
//
// 三者都是包级/全局变量（cobra 的标志绑在它们上面），所以必须逐条复位 ——
// 否则前一条用例留下的 `--session` 会把后一条悄悄带跑（那正是"测试之间互相串台"）。
func resetTargetFlags(t *testing.T) {
	t.Helper()
	oldSess, oldName, oldEnv := dbgSessionFlag, dbgNameFlag, common.Env
	dbgSessionFlag, dbgNameFlag, common.Env = "", "", ""
	dbgSelectorFailure = nil
	t.Cleanup(func() {
		dbgSessionFlag, dbgNameFlag, common.Env = oldSess, oldName, oldEnv
		dbgSelectorFailure = nil
	})
}

// TestCharTargetSingleSessionFallsThrough（Q2 / C8.1 / 写法 ①）**单会话行为不许变**。
//
// 这一条是 H2 在 CLI 侧的判据：池里恰好一条、又没给任何目标时，旧写法与新写法
// 必须给出**同一个 id**，而且都是成功 —— "唯一就落"这条便利住在 CLI（Q12 / C8.2）。
func TestCharTargetSingleSessionFallsThrough(t *testing.T) {
	resetTargetFlags(t)
	sessionPoolStub(t, []map[string]any{
		{"id": "sOnly", "state": "idle", "env": "恒烁正式区"},
	})

	id, err := dbgSessionID()
	if err != nil {
		t.Fatalf("单会话且没给目标时必须照旧成功（Q2 要的单会话不变），实得 %v", err)
	}
	if id != "sOnly" {
		t.Fatalf("单会话时该落到那一条，期望 %q，实得 %q", "sOnly", id)
	}
}

// TestCharTargetMultiSessionRefusesToPick（F1 / C8.1）**多会话且没给目标 → 报错并列候选**。
//
// 这是本次重构第一次改行为的落点，也是计划开头那句可验证的话的机器判据：
// "不会出现「命令跑在另一条会话上却看起来成功」；落不上目标时，一定是报错"。
//
// 四条断言：
//
//  1. 报错（不许静默取第一项）；
//  2. `code` 是契约 4.1.1 的 `SESSION_AMBIGUOUS`（Agent 读 `code`，不读数字 —— Q15）；
//  3. 候选清单**必须出现**（C5.11），且两种列表顺序给出的候选**同一批** ——
//     "顺序不再是答案"；
//  4. 退出码是 **F1 那一档（3）**，而且**无条件**是 —— 判据是"**这个场景是不是新场景**"，
//     不是"命令行里有没有 `--session` 这个字"（本轮改正，见下面断言处的注释）。
func TestCharTargetMultiSessionRefusesToPick(t *testing.T) {
	for name, order := range map[string][]map[string]any{
		"甲在前": {{"id": "sA", "state": "idle", "env": "恒烁正式区", "n": 1, "prog": "apmt520"},
			{"id": "sB", "state": "idle", "env": "恒烁正式区", "n": 2, "prog": "apmt520"}},
		"乙在前": {{"id": "sB", "state": "idle", "env": "恒烁正式区", "n": 2, "prog": "apmt520"},
			{"id": "sA", "state": "idle", "env": "恒烁正式区", "n": 1, "prog": "apmt520"}},
	} {
		t.Run(name, func(t *testing.T) {
			resetTargetFlags(t)
			sessionPoolStub(t, order)

			id, err := dbgSessionID()
			if err == nil {
				t.Fatalf("多会话且没给目标时必须报错，不许取第一个（实得 id=%q）——\n"+
					"    这正是计划开头那句可验证的话：多会话下取第一个 = 命令跑到另一条会话上，"+
					"而输出看起来一切正常。", id)
			}
			var te *dbgTargetError
			if !asTargetError(err, &te) {
				t.Fatalf("失败要带 code 与候选（契约 C4.1 / C5.11），实得 %T: %v", err, err)
			}
			if te.fail.Code != target.CodeAmbiguous {
				t.Errorf("code = %q，F1 要求 %q（Agent 读 code 不读数字，Q15）",
					te.fail.Code, target.CodeAmbiguous)
			}
			if len(te.fail.Candidates) != 2 {
				t.Fatalf("候选清单该有 2 条（C5.11：这个码下必须出现），实得 %d 条", len(te.fail.Candidates))
			}
			// 候选按"序号 → 原始句柄"稳定排序（C5.14 的可重放要求）：换列表顺序不该换候选顺序。
			got := []string{te.fail.Candidates[0].ID, te.fail.Candidates[1].ID}
			if got[0] != "sA" || got[1] != "sB" {
				t.Errorf("候选顺序 = %v，期望按序号稳定排序 [sA sB] —— 同一个错误两次跑出的文案必须一样", got)
			}
			// 退出码按 Q15 分档 —— 但**判据是"落在新场景里"，不是"有没有打标志"**
			// （本轮改正）：F1 = 多会话 + 不给目标，这个场景**今天不存在**
			// （今天静默取列表第一个并成功返回），H2 保护的是"今天已有的行为"，
			// 够不到它 —— 所以**无条件**退 F1 那一档（3）。
			//
			// 按错的判据（"没打 --session → 退今天那一档 1"）会得出荒谬结论：
			// 多会话歧义只要不打 --session 就退 1，而 Agent 恰恰是在**没打** --session
			// 时才撞上歧义，它最需要机器可读码的那一刻拿不到码。
			if ec := te.ExitCode(); ec != target.ExitNoSession {
				t.Errorf("没给 --session 时退出码 = %d，F1 要求 %d —— 判据是「这个场景今天存不存在」"+
					"（它不存在），不是「命令行里有没有 --session 这个字」", ec, target.ExitNoSession)
			}
			// 反向：**同一个场景**带上 --session 还是同一个码 —— "无条件"这三个字的意思。
			// （它同时钉住"码一样、数字也一样"：分档看场景，不看写法。）
			dbgSessionFlag = "恒烁正式区#9"
			if ec := dbgTargetErrorOf(te.fail).ExitCode(); ec != target.ExitNoSession {
				t.Errorf("给了 --session 时退出码 = %d，同一个新场景该给同一个码（F1 的 %d）",
					ec, target.ExitNoSession)
			}
		})
	}
}

// TestCharTargetSingleEnvNotConfusedWithAlias 钉住目标层那条"环境名赢"的分流（C2.4 / promoteEnvName）。
//
// `--session 恒烁正式区`（写法 ⑧：按环境找）与"按别名找"在字面上完全同形，
// 能分开它们的只有"它是否精确等于一个已配置的环境名"。池里两条、环境一条会话：
// 按环境落到那一条（不是报别名不存在）。
func TestCharTargetSingleEnvNotConfusedWithAlias(t *testing.T) {
	resetTargetFlags(t)
	sessionPoolStub(t, []map[string]any{
		{"id": "sA", "state": "idle", "env": "恒烁正式区", "n": 1},
		{"id": "sB", "state": "idle", "env": "测试1012", "n": 1},
	}, "恒烁正式区", "测试1012")
	dbgSessionFlag = "恒烁正式区"

	id, err := dbgSessionID()
	if err != nil {
		t.Fatalf("按环境找（写法 ⑧）该落到该环境下那一条，实得 %v", err)
	}
	if id != "sA" {
		t.Fatalf("落到 %q，期望 %q", id, "sA")
	}
}

// TestCharTargetSelectorGrammarIsRejectedBeforeRequest（F11 / C2.8 / C5.4）。
//
// 两件"发请求之前就该判掉"的事：
//
//  1. `--session ""` 按**文法非法**处理（F11），**不当成"省略"** —— 判据是
//     `target.ParseSelector("")` 直接返回 `USAGE`（纯函数，C2.8：一个字节都不发）；
//  2. 会话级 REST 路径**只认原始句柄**（C5.4）：id 为空时 `dbgSessionURL` 给空串，
//     不许拼出一个 `/api/sessions/…` 这种少了 id 的路径 —— 那种路径在服务端是 404，
//     会被读成"服务端说不存在"，而真相是"这次压根没拿到目标"。
func TestCharTargetSelectorGrammarIsRejectedBeforeRequest(t *testing.T) {
	if _, fail := target.ParseSelector(""); fail == nil || fail.Code != target.CodeUsage {
		t.Fatalf("`--session \"\"` 该按文法非法处理（F11 / USAGE），实得 %v", fail)
	}
	if got := dbgSessionURL("", ""); got != "" {
		t.Errorf("id 为空时路径该是空串（C5.4），实得 %q", got)
	}
	if got, want := dbgSessionURL("s1", "/wait"), "/api/sessions/s1/wait"; got != want {
		t.Errorf("路径该逐字是 %q，实得 %q —— 路径只认 id，不许拿 ref/display 去拼", want, got)
	}
}

// TestCharDBGSessionsEndpointShape 钉住 `dbgSessions`（debugenv.go:130）读的
// 服务端字段：`sessions[].{id,env,state}`。
//
// 它与会话寻址是同一个端点的两个消费者，所以上一条测试钉行为，这条钉**形状**：
// 服务端字段名一改，两处一起坏。
func TestCharDBGSessionsEndpointShape(t *testing.T) {
	resetTargetFlags(t)
	sessionPoolStub(t, []map[string]any{
		{"id": "s1", "env": "恒烁正式区", "state": "idle", "ref": "恒烁正式区#1"},
	})
	got, err := dbgSessions()
	if err != nil {
		t.Fatalf("该成功，得 %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("该解析出 1 条会话，实得 %d 条", len(got))
	}
	if got[0].ID != "s1" || got[0].Env != "恒烁正式区" || got[0].State != "idle" {
		t.Errorf("字段解析不符：%+v\n"+
			"    今天读的是 id/env/state 三个字段（debugenv.go:130-134）。\n"+
			"    C5.17–C5.19 要求把客户端手抄的第二份事实收掉：那时这几个键的唯一声明处"+
			"    是 internal/debug 的 SessionBrief。", got[0])
	}
	// 服务端多给了 ref（契约 C5.3 的新键）也不影响今天的解析 —— 这是"纯新增"该有的样子。
	if got[0].ID != "s1" {
		t.Error("新增键不该把旧键的解析顶掉")
	}
}

// TestCharDBGStateLabels 钉住会话状态的**中文标签**（`dbgStateLabel`，debugenv.go:351）。
//
// 契约依据：C5.9 的身份行 `会话 <display> · <name> · <state> · ENT <ent>` 明确说
// `<state>` 用**现行中文标签**。所以这不是"顺手测的函数"—— 它是那条身份行的一半输入。
//
// 五个取值来自 B30（session.go:28-34 的五个 State 常量），一个不多一个不少：
// 少一个标签就会在身份行里漏出一个英文单词。
func TestCharDBGStateLabels(t *testing.T) {
	// 契约 C5.9 的身份行 `会话 … · <state> · ENT …` 明确要求 `<state>` 用**现行中文标签**，
	// 所以生产实现（dbgStateLabel）与这五个标签必须逐字一致。
	//
	// 为什么标签在这边再写一份：那张表的唯一出处是
	// `internal/debug/contract_debug_test.go` 的 `stateLabelOf()`（文件靠后，注释里写了这处重复），
	// 但 Go 里另一个包**看不见**它的 `_test.go` 符号（实测编译不过）。
	//
	// **防漂移靠什么**（如实说清，别夸大）：目标层（`internal/target` 的 `StateLabel`）
	// 现在是**生产实现**那一份，WP4a 接线时本包的 `dbgStateLabel` 仍是旧的那一份
	// （这一处重复已由 target 的注释登记：接线时应当让它引用目标层，只留一处）。
	// 在那之前，这一条（拿生产实现去比这五对）是它的判据。
	want := map[string]string{
		"idle":    "空闲",
		"loading": "连接中…",
		"stopped": "已停站",
		"running": "运行中",
		"exit":    "已断开",
	}
	if len(want) != 5 {
		t.Fatalf("B30 的会话状态是**五个**取值，表里有 %d 个 —— 少一个就会在身份行里漏出一个英文单词", len(want))
	}
	for st, label := range want {
		if got := dbgStateLabel(st); got != label {
			t.Errorf("dbgStateLabel(%q) = %q，C5.9 写的是 %q\n"+
				"    这条标签进了身份行 `会话 … · <state> · ENT …`，改它等于改那一行的逐字格式。",
				st, got, label)
		}
		// 两份实现必须同值（目标层那份是 C5.9 的身份行要用的）。
		if got, want := target.StateLabel(st), label; got != want {
			t.Errorf("target.StateLabel(%q) = %q，C5.9 写的是 %q", st, got, want)
		}
	}
	// 不认识的取值原样透传（今天的 default 分支）—— 兜底成空串会让身份行少一个字段，
	// 四个字段的位置就漂了（C5.9 明写不许省整段）。
	if got := dbgStateLabel("没见过"); got != "没见过" {
		t.Errorf("不认识的取值今天原样透传，实得 %q", got)
	}
	if got := dbgStateLabel(""); got != "" {
		t.Errorf("空取值今天原样返回空串，实得 %q", got)
	}
}

// asTargetError 是 errors.As 的小包装（免得每条用例都拉一个 errors 导入）。
func asTargetError(err error, dst **dbgTargetError) bool {
	if te, ok := err.(*dbgTargetError); ok {
		*dst = te
		return true
	}
	return false
}
