package debug

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"tt/internal/host"
	"tt/internal/target"
)

// 四个"动作型"端点的**错误半边**：POST /api/wslogs/debug、POST /api/dbsql、
// GET /api/wslogs、GET /api/wslogs/content。
//
// 从前这里是五个（还含 `POST /api/wstest`）—— 那条端点本批已**从产品里取消**
// （仓库主人定性：它不是标准接口，是临时测试工具），实现与只测它的测试一并删掉。
//
// 为什么只测错误半边：它们的**快乐路径**要 `*host.SSHConn` —— 那个结构体的 `cli` 是私有
// 字段（host/ssh.go:18），要从外部伪造得实现 8 个方法、其中三个返回不可构造的类型
// （`*PTYSession` / `*sftp.Client`）。为一个测试付那个代价不划算，所以 Server 上开了个
// `dial` 字段（生产默认 host.Dial，行为一字不差），这里注入一个**必然失败**的实现。
//
// 买到的四类东西，都是端到端测不到的：
//   1. 请求体解析失败 → 400（而不是拿半个请求去拨号）
//   2. 拨号失败 → 500 且信封形状一致（"SSH 连接失败: %w"）
//   3. 未注册的 API 路径 → 404 + JSON 信封（**不是** 405，也不是 200 + HTML ——
//      路由里有一条 catch-all，Go 的 mux 就不会自己产生 405 了。见 hStatic 的注释）
//   4. **入参校验在拨号之前** —— hWSLogDebug 里那处注释专门说过的顺序性质
//      （"非法请求不该先把当前会话收口"）。今天没有任何东西守着它。
//
// WP4b 追加第 5 类：**"要一条会话却落不上目标" → 按契约 4.1.1 的码报错并给候选**
// （见 TestActionEndpointsRequireExplicitTarget）。

// errTestNoDial 是注入进去的拨号失败。文案刻意与真实失败不同，
// 这样测试断言里出现它就说明**确实走进了被测的那条分支**，而不是碰巧撞上别的错。
var errTestNoDial = errors.New("测试：不拨号")

// actionTestDebugServer 起一个不拨真机的调试服务，并**返回那个 Server**
// （有些用例要往会话池里放一条会话：WP4b 起"切环境 / 按日志重放"要求能落一条目标，见 api_target.go）。
// 复用 mode_test.go 的 doReq / respError 发请求收错误文案。
func actionTestDebugServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	s := NewServer(&Config{Listen: "127.0.0.1:0"}, nil, "")
	s.dial = func(host.SSHConfig) (*host.SSHConn, error) { return nil, errTestNoDial }
	hs := httptest.NewServer(s.Handler())
	t.Cleanup(hs.Close)
	return hs, s
}

// actionTestServer 是只要一个 URL 的那一半（不往池里放会话）。
func actionTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	hs, _ := actionTestDebugServer(t)
	return hs
}

// actionTestServerWithSession 起服务并往池里放**一条**空宿主会话（`StateIdle`），
// 返回它的句柄。用途：让"要一条目标"的那些端点能走到拨号那一层。
func actionTestServerWithSession(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	hs, s := actionTestDebugServer(t)
	sess := fakeIdleSession("s-one", "h1", "36")
	s.mgr.sessions[sess.ID] = sess
	return hs, sess.ID
}

// giveServerNamedEnv 给测试服务配一个**具名环境**（`config.json` 的 `hosts.sshs[].name`）。
//
// 为什么需要：`hSessionSwitch` 的判定顺序是"先校验环境存在（404）→ 再落目标"，
// 所以在一个没有任何环境的服务上测"缺目标"只会测到环境那条错（实测踩过）。
func giveServerNamedEnv(s *Server, name, hostAddr, zone string) {
	e := host.NamedSsh{Name: name, SSHConfig: host.SSHConfig{Host: hostAddr, Port: 22, User: "u"}, Zone: zone}
	s.cfg.SSHs = append(s.cfg.SSHs, e)
	s.cfg.fillDefaults()
	s.cfg.SSH, s.cfg.Zone = e.SSHConfig, zone
}

// TestActionEndpointsRejectMalformedBody 请求体不是 JSON → 400，
// 且不能是"拨号失败"那种 500 —— 半个请求根本不该走到拨号那一步。
func TestActionEndpointsRejectMalformedBody(t *testing.T) {
	hs := actionTestServer(t)
	for _, path := range []string{"/api/wslogs/debug", "/api/dbsql"} {
		t.Run(path, func(t *testing.T) {
			resp := doReq(t, "POST", hs.URL+path, "", "{ 这不是 json")
			if resp.StatusCode != 400 {
				t.Fatalf("该退 400，得 %d（%s）", resp.StatusCode, respError(t, resp))
			}
			if msg := respError(t, resp); !strings.Contains(msg, "请求体解析失败") {
				t.Errorf("错误文案该说清是请求体的问题，得 %q", msg)
			}
		})
	}
}

// TestActionEndpointsReportDialFailure 拨号失败 → 500，信封是
// `{"ok":false,"code":…,"error":…}`（C5.15：`code` 必给，`ok`/`error` 的形状一字不改）。
//
// 这条钉的是信封形状：前端与 CLI 都按它解析，改了形状它们会静默拿不到错误原因。
//
// **WP4b 之后这两条请求体必须各带一个前置条件**（否则它们会被更早的那道闸门拦下，
// 测不到拨号那一层）：
//
//   - `/api/wslogs/debug` 现在要求"能落一条会话"（`mustTargetOrEmpty`：池空 = 冷启动 → 放行，
//     多条 → 409 + `SESSION_AMBIGUOUS`）。这里先放一条空宿主会话（`actionTestServerWithSession`）
//     并带上它的句柄，只是为了把这一层闸门过得干净：这条测试要测的是拨号那一层。
//   - `/api/dbsql` 现在要求 `ent` 必填（C7.1-5），给 0 会被 400 拦在拨号之前。
func TestActionEndpointsReportDialFailure(t *testing.T) {
	hs, sid := actionTestServerWithSession(t)
	cases := []struct{ method, path, body string }{
		{"POST", "/api/wslogs/debug", `{"rowid":"1","session":"` + sid + `"}`},
		{"POST", "/api/dbsql", `{"sql":"select 1 from dual","ent":99}`},
		{"GET", "/api/wslogs", ""},
		{"GET", "/api/wslogs/content?rowid=1", ""},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			resp := doReq(t, c.method, hs.URL+c.path, "", c.body)
			if resp.StatusCode != 500 {
				t.Fatalf("拨号失败该退 500，得 %d（%s）", resp.StatusCode, respError(t, resp))
			}
			msg := respError(t, resp)
			if !strings.Contains(msg, "SSH 连接失败") {
				t.Errorf("文案该说清是 SSH 连不上，得 %q", msg)
			}
			// 注入的那句话必须出现在里面 —— 证明走的是我们替换的那个 dial。
			if !strings.Contains(msg, errTestNoDial.Error()) {
				t.Errorf("该是由注入的 dial 报的错（否则这条测试没测到它），得 %q", msg)
			}
		})
	}
}

// TestActionEndpointsRequireExplicitTarget 是 WP4b 加的那一类：**要一条会话却落不上目标**。
//
// 契约依据：Q4「服务端不保留隐式寻址」/ C8.2（"唯一就落"只住在 CLI 里）/ C7.1-2 / C7.1-3 /
// C7.1-5 / C5.15 / C5.16 / 4.1.1 的 code 取值表。
//
// 从前这几条路静默取"当前会话"（`Manager.Current()` 取的是 map 里**任意一条**），
// 多会话下就是"随机落一条而输出看起来成功"。现在它们必须报错、且信封里带稳定 `code`：
//
//	① 池里 2 条 + 不带目标      → 409 + SESSION_AMBIGUOUS + 候选清单（F1 的形状）
//	② 点名了一条不在池里的会话   → 404 + SESSION_NOT_FOUND
//	③ /api/dbsql 不给 ent       → 400 + USAGE（C7.1-5：不再回落会话的 TOPENT）
//
// **「池空 + 不带目标 → 409 NO_SESSION」这一档已经不在这张表里**，因为它现在**没有任何端点**：
// 从前占着它的是两条路线，而它们都是**创建**会话的命令，池子空恰恰是最正常的起点 ——
//   - 切环境：Web 上那个按钮就叫「创建会话」，命令行是 `tt debug env <环境>`；
//   - 按日志重放：「调试此调用」/ `tt debug wsdebug <rowid>`，它自己建那条会话来重跑一遍。
//
// 报 `NO_SESSION` 等于让"先建会话、再设 ENT、再调试"这条正确流程**第一步**就断，
// 用户只能先起个作业去凑一条会话（顺序完全反了）。两处的新判据都在
// `session_named_host_test.go` 的 `TestSessionSwitchColdStartIsNotNoSession` /
// `TestWSLogDebugColdStartIsNotNoSession`；F2 这一档仍活在 CLI 目标层
// （`internal/target` 的 `failureNoSession`），信封形状见
// `contract_debug_test.go` 的 `TestContractNoSessionEnvelopeIsStillPinned`。
//
// 为什么断言 `code` 而不是状态码：Q15 写死了「Agent 读 `code`，不读数字」。
func TestActionEndpointsRequireExplicitTarget(t *testing.T) {
	// 池里两条（并给它一个具名环境：切环境那条会把"环境不存在"判在"缺目标"之前）
	hsTwo, sTwo := actionTestDebugServer(t)
	giveServerNamedEnv(sTwo, "h1", "127.0.0.1", "36")
	for _, id := range []string{"s-a", "s-b"} {
		sTwo.mgr.sessions[id] = fakeIdleSession(id, "h1", "36")
	}
	// 池空 + **配了环境**的那套服务不再需要：切环境与按日志重放的冷启动判据都在
	// `session_named_host_test.go`（它们都是"创建会话"，池空不是错误）。

	cases := []struct {
		name     string
		hs       *httptest.Server
		path     string
		body     string
		status   int
		code     string
		candLen  int
		wantText string
	}{
		{"两条 + 切环境（歧义）", hsTwo, "/api/sessions/switch", `{"env":"h1","ent":"99"}`,
			http.StatusConflict, target.CodeAmbiguous, 2, "无法确定你要哪一条"},
		{"两条 + 按日志重放（歧义）", hsTwo, "/api/wslogs/debug", `{"rowid":"1"}`,
			http.StatusConflict, target.CodeAmbiguous, 2, "无法确定你要哪一条"},
		{"点名了一条不存在的会话", hsTwo, "/api/wslogs/debug", `{"rowid":"1","session":"s-没了"}`,
			http.StatusNotFound, target.CodeNotFound, -1, "会话不存在"},
		{"只读 SQL 不给 ent", hsTwo, "/api/dbsql", `{"sql":"select 1 from dual"}`,
			http.StatusBadRequest, debugContractCodeUsage, -1, "ent 必填"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp := doReq(t, "POST", c.hs.URL+c.path, "", c.body)
			if resp.StatusCode != c.status {
				t.Fatalf("该退 %d，得 %d（%s）", c.status, resp.StatusCode, respError(t, resp))
			}
			var raw map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
				t.Fatalf("失败信封不是合法 JSON：%v", err)
			}
			// C5.15：`ok` 与 `error` 的形状一个字都不改，只在上面**增加** code / candidates。
			if raw["ok"] != false {
				t.Errorf("ok 必须是 false（C5.15），实得 %v", raw["ok"])
			}
			if s, _ := raw["error"].(string); strings.TrimSpace(s) == "" {
				t.Errorf("error 必须非空（C5.15），实得 %v", raw["error"])
			}
			if got := raw["code"]; got != c.code {
				t.Errorf("code = %v，契约 4.1.1 要求 %q\n    信封：%v", got, c.code, raw)
			}
			if c.code == "" {
				return
			}
			if msg, _ := raw["error"].(string); c.wantText != "" && !strings.Contains(msg, c.wantText) {
				t.Errorf("文案里该有一句 %q，实得 %q", c.wantText, msg)
			}
			// C5.11：候选清单只在"歧义/指名不存在"那几种码下出现，且必须是九键的投影。
			cands, hasCands := raw["candidates"]
			if c.candLen < 0 {
				if hasCands {
					t.Errorf("这个码下不该有 candidates（C5.11），实得 %v", cands)
				}
				return
			}
			if !hasCands {
				t.Fatalf("这个码下**必须**有 candidates（C5.11），实得 %v", raw)
			}
			list, ok := cands.([]any)
			if !ok || len(list) != c.candLen {
				t.Fatalf("candidates 该是 %d 条，实得 %v", c.candLen, cands)
			}
			for _, one := range list {
				row, _ := one.(map[string]any)
				for _, k := range []string{"id", "ref", "display", "name", "env", "ent", "n", "prog", "state"} {
					if _, ok := row[k]; !ok {
						t.Errorf("候选行少了 %q 键（C5.12 的九键），实得 %v", k, row)
					}
				}
			}
		})
	}
}

// TestUnregisteredAPIPathsReturnJSON404 钉住"打错的 API 路径不是 200 + HTML"。
//
// 这条断言本来长得不一样：它原先写的是"用 GET 打 POST 端点应当 405"，而那是**错的**
// —— 实测 200，因为路由里那条 `mux.HandleFunc("/", s.hStatic)` 的 SPA 兜底把任何
// 未注册路径都吞掉，未构建前端时回"前端尚未构建"那页、构建过就回 index.html。
//
// 当时撤掉了断言而不是把 200 钉下来（把已知不对的行为写成契约更坏），并把它记成一条
// 待定项。现在兜底收窄了，断言以它本来的意图回来，只把期望值从 405 改成 404 ——
// 405 要手写一张注册表才拿得到，而 404 与 405 对调用方是同一件事。理由见 hStatic 注释。
//
// 两种前端状态都测，因为**构建过的那半更坏**：它回的是一张真的 index.html，
// 调用方看不出任何异样；未构建那半至少文案里写着"前端尚未构建"。
func TestUnregisteredAPIPathsReturnJSON404(t *testing.T) {
	built := func() *httptest.Server {
		web := fstest.MapFS{"index.html": {Data: []byte("<html>APP</html>")}}
		hs := httptest.NewServer(NewServer(&Config{Listen: "127.0.0.1:0"}, web, "").Handler())
		t.Cleanup(hs.Close)
		return hs
	}()
	servers := map[string]*httptest.Server{"前端未构建": actionTestServer(t), "前端已构建": built}

	cases := []struct{ method, path string }{
		{"GET", "/api/dbsql"}, // 路径注册了，方法是 POST
		{"GET", "/api/根本不存在"}, // 路径压根没有
		{"POST", "/api/根本不存在"},
		{"GET", "/api"}, // 前缀本身
		{"GET", "/api/"},
		{"DELETE", "/api/sessions/xyz/nope"}, // 注册前缀之下的未知子路径
	}
	for name, hs := range servers {
		for _, c := range cases {
			t.Run(name+" "+c.method+" "+c.path, func(t *testing.T) {
				resp := doReq(t, c.method, hs.URL+c.path, "", "")
				if resp.StatusCode != http.StatusNotFound {
					t.Fatalf("未注册的 API 路径该退 404，得 %d（%s）", resp.StatusCode, respError(t, resp))
				}
				if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
					t.Errorf("该回 JSON 信封，Content-Type 得 %q", ct)
				}
				if msg := respError(t, resp); msg == "" {
					t.Errorf("信封里该有错误原因（前端与 CLI 按它显示）,否则等于没说")
				}
			})
		}
	}
}

// TestStaticFallbackStillServesSPA 是上一条的**反向判据**：收窄兜底不能顺手把 SPA 打掉。
//
// `/apifoo` 与 `/api2/x` 是刻意的边界 —— 判据是 `/api` 或 `/api/` 前缀，不是
// `strings.HasPrefix(p, "/api")` 那种松写法，否则这两个正常的客户端路由会被误判成 API。
func TestStaticFallbackStillServesSPA(t *testing.T) {
	web := fstest.MapFS{"index.html": {Data: []byte("<html>APP</html>")}}
	hs := httptest.NewServer(NewServer(&Config{Listen: "127.0.0.1:0"}, web, "").Handler())
	t.Cleanup(hs.Close)

	for _, p := range []string{"/", "/sessions/abc", "/apifoo", "/api2/x"} {
		resp := doReq(t, "GET", hs.URL+p, "", "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s 该走 SPA 兜底得 200，得 %d", p, resp.StatusCode)
		}
	}
}

// TestWSLogDebugValidatesInputBeforeDialing 钉住 hWSLogDebug 的**判定顺序**。
//
// 那个 handler 里有一句注释专门说了这件事：
//
//	入参先校验:非法请求不该先把当前会话收口(旧顺序是收口后才在 LaunchReplay 里失败)
//
// 顺序是这样的：读体 → checkReplayOverride（入参上限）→ 会话说闸门 → 拨号 → 查库 → 收口旧会话。
// 所以给一个**超限的 request**，应当拿到 **400**（入参被拒），而**不是** 500（拨号失败）。
// 两者都是"失败"，只有状态码能把它们区分开 —— 一旦有人把校验挪到拨号之后，
// 这条会从 400 变成 500 而红。
func TestWSLogDebugValidatesInputBeforeDialing(t *testing.T) {
	hs := actionTestServer(t)

	// maxReplayPayload 是 256 KB，给一个超过它的入参。
	huge := strings.Repeat("x", maxReplayPayload+1)
	body := `{"rowid":"1","request":"` + huge + `"}`

	resp := doReq(t, "POST", hs.URL+"/api/wslogs/debug", "", body)
	if resp.StatusCode == 500 {
		t.Fatalf("超限入参该在拨号**之前**被拒（400），却走到了拨号（500）：%s", respError(t, resp))
	}
	if resp.StatusCode != 400 {
		t.Fatalf("该退 400，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	if msg := respError(t, resp); !strings.Contains(msg, "上限") {
		t.Errorf("文案该说清是入参超限，得 %q", msg)
	}
}

// ---------------------------------------------------------------------------
// 规格补充端点（§2 的命令面）：`?env=` 过滤、`/ent`、`POST /quit`
// ---------------------------------------------------------------------------

// TestListFiltersByEnv：`GET /api/sessions?env=` 只返回那个环境的会话。
//
// 为什么要有它：多会话之后"列全部"在界面上越来越没用 —— 人问的是"这个环境现在有几条"。
func TestListFiltersByEnv(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	giveServerNamedEnv(s, "h2", "127.0.0.1", "36")
	s.mgr.sessions["s1"] = idleOn("s1", s.cfg.CloneEnv("h1"))
	s.mgr.sessions["s2"] = idleOn("s2", s.cfg.CloneEnv("h2"))

	resp := doReq(t, "GET", hs.URL+"/api/sessions?env=h1", "", "")
	if resp.StatusCode != 200 {
		t.Fatalf("list: %d %s", resp.StatusCode, respError(t, resp))
	}
	var out struct {
		Sessions []struct {
			Env string `json:"env"`
		} `json:"sessions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if len(out.Sessions) != 1 || out.Sessions[0].Env != "h1" {
		t.Fatalf("只该剩 h1 那条：%+v", out.Sessions)
	}
}

// TestSessionEntEndpoint：`POST /sessions/{id}/ent` 与 `/topent` 同一件事，
// 只是请求体按规格写成 {"value"} / {"clear":true}。
func TestSessionEntEndpoint(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	sess := idleOn("s1", s.cfg.CloneEnv("h1"))
	s.mgr.sessions["s1"] = sess

	// 假会话没有真 PTY（SetTopent 要往 shell 下发），所以这里钉**接线与闸门**：
	// 非空闲时 /ent 与 /topent 给出同一个拒绝 —— 两条路走的是同一份实现。
	sess.mu.Lock()
	sess.state = StateStopped
	sess.mu.Unlock()
	for _, path := range []string{"/api/sessions/s1/ent", "/api/sessions/s1/topent"} {
		resp := doReq(t, "POST", hs.URL+path, "", `{"value":"88"}`)
		if resp.StatusCode != 409 {
			t.Fatalf("%s 应与非空闲会话一样拒绝，实得 %d（%s）", path, resp.StatusCode, respError(t, resp))
		}
	}
}

// TestSessionQuitByPost：规格里 quit 是 POST；与 DELETE 同一件事（结束本轮、会话保留）。
func TestSessionQuitByPost(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
	sess := idleOn("s1", s.cfg.CloneEnv("h1"))
	s.mgr.sessions["s1"] = sess

	resp := doReq(t, "POST", hs.URL+"/api/sessions/s1/quit", "", "")
	if resp.StatusCode != 200 {
		t.Fatalf("quit: %d %s", resp.StatusCode, respError(t, resp))
	}
	// 会话**还在**（结束本轮 ≠ 收掉会话），这是 DELETE/quit 与 /close 的分界。
	if s.mgr.Get("s1") == nil {
		t.Fatal("quit 之后会话必须保留（收会话走 /close）")
	}
}

// TestSessionJobStartsOnThatSession：`POST /sessions/{id}/job` 必须**钉在那条会话上**。
//
// 关键不是"能不能起作业"（那要真 SSH），而是：非空闲直接拒；空闲时用的是**这条**会话当宿主
// （`hostID`），而不是另找一个同环境的空闲宿主 —— 后者会让"会话里的作业"跑到别人那条上。
func TestSessionJobStartsOnThatSession(t *testing.T) {
	hs, s := actionTestDebugServer(t)
	giveServerNamedEnv(s, "h1", "127.0.0.1", "36")

	running := idleOn("s1", s.cfg.CloneEnv("h1"))
	running.setNameInTest("A")
	running.mu.Lock()
	running.state = StateStopped
	running.mu.Unlock()
	s.mgr.sessions["s1"] = running
	resp := doReq(t, "POST", hs.URL+"/api/sessions/s1/job", "", `{"prog":"bsft001_wf"}`)
	if resp.StatusCode != 409 {
		t.Fatalf("非空闲该 409，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}

	// 缺 prog：400（不拨号、不建会话）
	idle := idleOn("s2", s.cfg.CloneEnv("h1"))
	idle.setNameInTest("B")
	s.mgr.sessions["s2"] = idle
	resp = doReq(t, "POST", hs.URL+"/api/sessions/s2/job", "", `{}`)
	if resp.StatusCode != 400 {
		t.Fatalf("缺 prog 该 400，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}

	// 空闲 + 有 prog：拨号必然失败（测试桩），但**会话没被换掉**，且报的是拨号错
	resp = doReq(t, "POST", hs.URL+"/api/sessions/s2/job", "", `{"prog":"bsft001_wf"}`)
	if resp.StatusCode < 500 {
		t.Fatalf("拨号失败该 5xx，实得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	if got := s.mgr.Get("s2"); got == nil {
		t.Fatal("起作业失败不该把宿主会话删掉（它还是那条空闲会话）")
	}
}
