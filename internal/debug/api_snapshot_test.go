package debug

// `GET /api/sessions/{id}` 快照的**键集**（契约 C5.1 / C5.3 / C5.4 / C5.9 / C5.17–C5.19）。
//
// 这一组钉两件事，两件都是"手抄第二份事实"最容易静默漂坏的地方：
//
//  1. **身份键恒定出现**（`ref` / `display` / `name` / `ent` / `n` / `kind` / `epoch`）——
//     它们是 C5.1 / C5.3 的恒定键：取不到给空串/零值，**不省键**。前端与 CLI 按这份
//     形状解包，少一个键它们就得写第二套兜底（上一版少了 `ent`，CLI 那边就长出了一层
//     "退回 `topentSession`"的兜底 —— C5.9 的身份行要的正是 `ent`）。
//  2. **快照的既有键一个不少、语义一字不改**（H2）—— 身份键之外，键集必须
//     **恰好**是今天那一份：21 个恒定键（本文件 `snapshotBaseKeys`）+ 7 个身份键 = 28，
//     另有 4 个按状态出现的既有键（`snapshotOptionalKeys`）。多一个键也是坏的：
//     没人认识的键会让人以为服务端学会了新东西。
//
// 身份键的**值**不在这里另算一遍：它们由 `identityKeysOf` 从 `SessionBrief` 的投影搬过来，
// 因此这里断言的是"搬对了"（值等于 `manager.Snapshot()` 里那一条，且 ref/display 的
// 形态与 C5.4 一致），而不是"服务端会拼 ref" —— 拼法只有 `briefOf` 一处（C5.17–C5.19）。

import (
	"encoding/json"
	"net/http"
	"testing"

	"tt/internal/target"
)

// snapshotBaseKeys 是**新增身份七键之前**快照的恒定键集（21 个），逐字摘自本次改动前的
// `hSnapshot`。它是 H2 那半条判据的靶子：这 21 个键一个都不许少。
//
// `inflight` / `runningSince` / `waitingForUser` / `waitingKind` 也是既有键，
// 但它们按状态出现（见 `snapshotOptionalKeys`），所以不在"恒定"这一列。
var snapshotBaseKeys = []string{
	"ok", "id", "module", "prog", "runProg", "env", "state", "stop", "zone",
	"started", "breakpoints", "holdingSeconds", "watchdogSeconds",
	"topent", "topentSession", "topentShell", "replayResponse",
	"silentSeconds", "mode",
	"fglserver", "screenTarget",
}

// snapshotOptionalKeys 是**既有**的按状态出现的键（本次一个字都没动）。
var snapshotOptionalKeys = []string{"inflight", "runningSince", "waitingForUser", "waitingKind"}

// snapshotIdentityKeys 是身份键（C5.1 / C5.3 的恒定键，无 omitempty）。
//
// `ent` 是第七个：C5.9 的身份行 `会话 <display> · <name> · <state> · ENT <ent>` 要它，
// 而它本来就在 `SessionBrief` 上（`SessionBrief.Ent` = `Session.CurrentEnt()`）——
// 快照只是把它搬过来，CLI 那边就不必再从 `topentSession` 兜底（C5.17–C5.19）。
var snapshotIdentityKeys = []string{"ref", "display", "name", "ent", "n", "kind", "epoch"}

// snapshotServerWith 起一个不拨真机的调试服务，往池里放**一条**带身份的会话，返回它的句柄。
//
// `assignIdentityLocked` 是服务端分配 `ref` / `n` / `epoch` 的**唯一**入口（WP4c 起
// 四条造会话的路径都过它），所以这里走它 —— 手写 ref 就等于把被测的那条路绕开了。
func snapshotServerWith(t *testing.T, env string, prog string, alias string) (*Server, string) {
	t.Helper()
	cfg := cfgFor("h1", "36")
	if env != "" {
		cfg = envCfg(env)
	}
	s := NewServer(cfg, nil, "")
	sess := idleOn("s1", cfg)
	sess.Prog = prog
	m := s.mgr
	m.mu.Lock()
	if err := m.assignIdentityLocked(sess, alias); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions[sess.ID] = sess
	m.mu.Unlock()
	return s, sess.ID
}

// getSnapshot 走真实路由取一次快照（不是直接调 `hSnapshot`）：端点注册与路径参数
// 也一并被测到，`{id}` 解不出会话时这里会当成失败。
func getSnapshot(t *testing.T, s *Server, id string) map[string]any {
	t.Helper()
	mux := http.NewServeMux()
	s.routes(mux)
	hs := newTestHTTPServer(t, mux)
	resp := doReq(t, "GET", hs.URL+"/api/sessions/"+id, "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("取快照该退 200，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("快照不是合法 JSON：%v", err)
	}
	return raw
}

// TestSnapshotCarriesIdentityKeysAndKeepsEveryOldKey 钉住键集：**身份键一个不少 + 既有一个不少**，
// 合计**恰好 29 个**（21 个既有恒定键 + 8 个身份键）。
//
// 两向断言（少一个键、多一个键都红）是刻意的：
//   - 少一个既有键 → 前端按老形状解包会静默拿到 nil（H2 破）；少一个身份键（比如 `ent`）
//     → 调用方又得自己拼兜底，那正是本轮收口要收掉的东西（C5.9 / C5.17–C5.19）；
//   - 多一个没人认识的键 → 调用方以为服务端学会了新东西，而契约里没有它。
func TestSnapshotCarriesIdentityKeysAndKeepsEveryOldKey(t *testing.T) {
	s, id := snapshotServerWith(t, "恒烁正式区", "apmt520", "汇率单")
	raw := getSnapshot(t, s, id)

	want := map[string]bool{}
	for _, k := range snapshotBaseKeys {
		want[k] = true
	}
	for _, k := range snapshotIdentityKeys {
		want[k] = true
	}
	got := map[string]bool{}
	for k := range raw {
		got[k] = true
	}
	// 数量写死：21 旧恒定键 + 7 身份 = 28（另外 4 个按状态出现的键不在这一列，见
	// `snapshotOptionalKeys`）。少了会由下面两向断言报出具体是哪个，这里再钉一次总数
	// —— 免得"两张表都少写了同一个键"这种双错互相掩护。
	if len(want) != 28 {
		t.Fatalf("前置不成立：期望键集该是 21 旧 + 7 身份 = 28 个，表里凑出 %d 个", len(want))
	}
	for k := range want {
		if !got[k] {
			t.Errorf("快照少了键 %q —— 既有键一个都不许少（H2），身份键必须恒定出现（C5.1 / C5.9）\n    实得键集：%v", k, keysOf(raw))
		}
	}
	for k := range got {
		if !want[k] {
			t.Errorf("快照多了一个契约没写的键 %q\n    实得键集：%v\n"+
				"    （新增键要走 C5.3 那一处：键集只有 `SessionBrief` 一份定义）", k, keysOf(raw))
		}
	}
	if len(got) != 28 {
		t.Errorf("快照该恰好有 29 个恒定键（21 旧 + 8 身份），实得 %d 个：%v", len(got), keysOf(raw))
	}
}

// TestSnapshotIdentityKeysAreConstant 钉住 C5.1 / C5.3 那半条：**取不到也给空串/零值，不省键**。
//
// 三种最容易"省键"的现场各来一次：
//
//	① 已命名的会话（正常形状，值都要对）；
//	② **未命名 + 作业为空**：`name` 给空串、`display` 退成 `环境#序号` —— 都不许消失；
//	③ `epoch` 是登记时发的（它不该为空，空 = 前端没法判 F15 的失效）。
//
// 身份键的**值**必须与 `Manager.Snapshot()` 里那一条逐字相等 —— 那是"没有第二份事实"的判据：
// 快照与列表是同一份投影，不是两处各算一遍（C5.17–C5.19）。
func TestSnapshotIdentityKeysAreConstant(t *testing.T) {
	cases := []struct {
		name  string
		env   string
		prog  string
		alias string
	}{
		{"已命名", "恒烁正式区", "apmt520", "汇率单"},
		{"未命名且作业为空", "恒烁正式区", "", ""},
		{"默认环境（没点名）", "", "apmt520", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, id := snapshotServerWith(t, c.env, c.prog, c.alias)
			raw := getSnapshot(t, s, id)

			briefs := s.mgr.Snapshot()
			if len(briefs) != 1 {
				t.Fatalf("前置不成立：池里该恰好一条会话，实得 %d 条", len(briefs))
			}
			b := briefs[0]

			// ① 身份键**都在**（C5.1 / C5.3：恒定出现，不省键）—— 这是本测试的主断言。
			for _, k := range snapshotIdentityKeys {
				if _, ok := raw[k]; !ok {
					t.Errorf("快照缺了恒定键 %q（C5.1 / C5.3：取不到给空串/零值，**不省键**）\n    实得：%v", k, raw)
				}
			}
			if t.Failed() {
				return
			}
			// ② 值与列表快照逐字相同：身份键只有 `briefOf` 一处投影。
			if got := raw["ref"]; got != b.Ref {
				t.Errorf("ref = %v，`SessionBrief` 给的是 %q —— 快照不许自己再拼一遍（C5.17–C5.19）", got, b.Ref)
			}
			if got := raw["display"]; got != b.Display {
				t.Errorf("display = %v，`SessionBrief` 给的是 %q —— 同上", got, b.Display)
			}
			if got := raw["name"]; got != b.Name {
				t.Errorf("name = %v，`SessionBrief` 给的是 %q", got, b.Name)
			}
			// `ent` 是 C5.9 身份行要的那一个：它的值就是 `SessionBrief.Ent` 这一份事实
			// （= `Session.CurrentEnt()`），客户端不许在别处另取一份。
			if got := raw["ent"]; got != b.Ent {
				t.Errorf("ent = %v，`SessionBrief` 给的是 %q —— 快照不许自己再拼一遍，"+
					"它也不能是 `topentSession` 的另一种写法（C5.9 / C5.17–C5.19）", got, b.Ent)
			}
			if _, ok := raw["ent"].(string); !ok {
				t.Errorf("ent 该是字符串（取不到 = 空串，C5.1），实得 %T", raw["ent"])
			}
			if got := raw["n"]; got != float64(b.N) { // JSON 数字解出来是 float64
				t.Errorf("n = %v，`SessionBrief` 给的是 %d", got, b.N)
			}
			if got := raw["kind"]; got != target.KindSSH {
				t.Errorf("kind = %v，C5.5 要求恒为 %q（本契约只寻址 SSH 会话）", got, target.KindSSH)
			}
			if got := raw["epoch"]; got != b.Epoch || b.Epoch == "" {
				t.Errorf("epoch = %v，`SessionBrief` 给的是 %q（登记时必发一个，空 = F15 的失效判定没法用）", got, b.Epoch)
			}

			// ③ 取值形态按 C5.4 分工：ref 是 `环境#序号`（不含 `/`），display 是人读形态。
			ref, _ := raw["ref"].(string)
			if ref == "" || !containsAt(ref, target.SeqSep) {
				t.Errorf("ref = %q，C2.1.2 要求形如 `环境#序号`", ref)
			}
			if containsAt(ref, target.SelectorSep) {
				t.Errorf("ref = %q 含 %q —— 那是 display 的分隔符，ref 只由不可变部分拼（C2.0 / C5.4）", ref, target.SelectorSep)
			}
			// ④ `name` / `ent` 取不到时是**空串**（是"空串"这个值在，不是键不在）。
			if _, ok := raw["name"].(string); !ok {
				t.Errorf("name 该是字符串（未命名 = 空串，C5.1），实得 %T", raw["name"])
			}
		})
	}
}

// TestSnapshotOptionalKeysAreUntouched 钉住另外那四个**既有**的按状态出现的键：
// 本次一行都没动它们，但它们最容易在"顺手多搬几个字段"时被改成恒定出现（那也是破 H2）。
//
// 空闲、没有在跑的东西、没有停站现场时：四个键一个都不该出现。
func TestSnapshotOptionalKeysAreUntouched(t *testing.T) {
	s, id := snapshotServerWith(t, "恒烁正式区", "apmt520", "")
	raw := getSnapshot(t, s, id)
	for _, k := range snapshotOptionalKeys {
		if v, ok := raw[k]; ok {
			t.Errorf("空闲会话不该有 %q 键（它不是恒定键，一直是按状态出现的），实得 %v", k, v)
		}
	}
}

// TestSnapshotEntIsItsOwnFactAndTopentSessionSurvives 钉住 `ent` 与 `topentSession`
// 在快照里是**两件事**（C5.1 / C5.3 / C5.9 / H2）。
//
// 为什么值得单钉一条：`ent` 是身份行 `… · ENT <ent>`（C5.9）要的那个值，
// 而 `topentSession` 是"本会话显式给的企业编号"这**另一个**事实（C2.5c）。
// 两者常常同值（都从 `--ent` 来），于是"拿 `topentSession` 顶 `ent`"这种写法看起来毫无破绽 ——
// 上一批 CLI 那处兜底就是这么长出来的，而它一顶就会在**会话内 `tt debug topent` 改过**的
// 会话上取错值（那是本次要根除的"第二份事实"）。
//
// 两条判据：
//
//	① 两个键**可以不同**：`ent` 跟着会话内覆盖走，`topentSession` 坚持启动时给的那个；
//	② `topentSession` 这个**既有键还在**（收口只删 CLI 那层取舍，不许顺手删服务端的键，H2）。
func TestSnapshotEntIsItsOwnFactAndTopentSessionSurvives(t *testing.T) {
	s, id := snapshotServerWith(t, "恒烁正式区", "apmt520", "汇率单")
	// 起会话时给的 `--ent`（`tt debug start --ent 99` 落到 cfg.Topent 上）：
	// `ent` 与 `topentSession` 现在是同一个值。
	s.mgr.sessions[id].cfg.Topent = "99"
	if got := getSnapshot(t, s, id); got["ent"] != "99" || got["topentSession"] != "99" {
		t.Fatalf("前置不成立：起会话给的 TOPENT 该让两个键都是 99，实得 ent=%v topentSession=%v",
			got["ent"], got["topentSession"])
	}
	// `tt debug topent 88` 的状态那一半（另一半是把值下发到宿主 shell，要 pty）：
	// 只有 `ent` 这份**快照**该变，`topentSession` 不动。
	s.mgr.sessions[id].setTopentOverride("88")

	raw := getSnapshot(t, s, id)
	if got := raw["ent"]; got != "88" {
		t.Errorf("`ent` 该跟着会话内覆盖变成 88（C2.5c：它是快照），实得 %v", got)
	}
	if got := raw["topentSession"]; got != "99" {
		t.Errorf("`topentSession` 是**另一个**事实（本会话显式给的那个），这里是启动时给的 99，实得 %v ——\n"+
			"    它不能是 `ent` 的别名，也不能被 `ent` 顶掉（C5.9 / H2：既有键语义一字不改）", got)
	}
}

// keysOf 用同包既有的那一份（`contract_debug_test.go`：排序后返回，失败信息里要用）。

// containsAt 是本文件的小工具（不 import strings 只为这一句，
// 免得与同包别的测试文件的 import 块打架）：子串判定，空子串算命中。
func containsAt(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
