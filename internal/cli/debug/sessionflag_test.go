package debug

// 起会话那一族命令上两个标志的**接线判据**（Bug A / Bug B）。
//
// # 这里钉什么
//
//  1. **Bug A**：`tt debug start x --session <选择器>` 落上的那一条必须**随请求体送到服务端**
//     （`body["session"]` = 服务端原始句柄，C5.4）。从前这一步只在本机"落位校验"、
//     一个字节都不送，于是服务端仍按"环境+名字"收窄候选 —— 同环境两条空闲会话时
//     `--session 恒一#1` 本地校验通过、POST 却不带目标，服务端照旧报
//     `SESSION_AMBIGUOUS`：标志在 --help 里写着，却帮不上忙。
//  2. **环境由指名的那条会话决定**：只给 `--session`、没给 `--env`/`--ssh` 时，
//     `body["ssh"]` 必须是那条会话的 env。不补这一句，服务端会拿"它自己当前的环境"
//     去组装本轮配置，点名恒一#1 却把这一轮落到另一个环境上。
//  3. **H2：空值不送**。没打 `--session` / `--name` 时，请求体与今天**逐字相同**
//     （那几个键根本不出现）—— 这是"新增字段不许改变旧行为"的机器判据。
//  4. **Bug B**：`tt debug wsdebug --help` 里看得见 `--name`，且它的取值纪律与
//     `debug start` **同一份**（同一份 `dbgValidateNameFlag`）；非法别名在**发请求之前**
//     被拒（C2.8：一个字节都不发），退出码按 F19 退 2。
//
// # 为什么用 HTTP 桩
//
// 这三件事全都是"CLI 到底送了哪些字段"，一个能记下请求体的桩就能把它们变成断言，
// 而且不依赖真 SSH（与 currentsession_char_test.go 的 `sessionPoolStub` 同一路数）。
// 桩对 POST 一律回一条失败信封：这几条判据只关心**请求体**，不关心成功之后那一串
// （等停站 / 快照 / 身份行）—— 记完账就收场，测试不会多跑一步。

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"tt/internal/target"
)

// recordedBody 收下桩看到的 POST 请求体（nil = 一次 POST 都没来）。
type recordedBody struct{ body map[string]any }

// launchAPIStub 起一个"记请求体"的桩：GET 会话池 / 环境，POST 记账后回失败信封。
//
// 会话池用 `map[string]any` 而不是 `target.Session`：这里要顺带钉住"CLI 解的是
// **服务端产出的那五个键**"（ref/env/n…，C5.3），写成 Go 结构体会把字段名藏起来。
func launchAPIStub(t *testing.T, sessions []map[string]any, envs ...string) *recordedBody {
	t.Helper()
	if envs == nil {
		envs = []string{"恒一"}
	}
	sshs := make([]map[string]any, 0, len(envs))
	for _, e := range envs {
		sshs = append(sshs, map[string]any{"name": e})
	}
	rec := &recordedBody{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost:
			b, _ := io.ReadAll(r.Body)
			m := map[string]any{}
			_ = json.Unmarshal(b, &m)
			rec.body = m
			// 回一条**带 code** 的信封：这样 RunE 走的是 `dbgServerErrorOf` 那条路，
			// 不会退成"HTTP 409: …"那种散文（判据只关心请求体，但回包要像真的）。
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"ok":false,"code":"SESSION_NOT_FOUND","error":"测试桩：到此为止"}`))
		case strings.HasSuffix(r.URL.Path, "/api/hosts"):
			_ = json.NewEncoder(w).Encode(map[string]any{"sshs": sshs})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"sessions": sessions})
		}
	}))
	t.Cleanup(ts.Close)

	oldURL := dbgAPIURL
	dbgAPIURL = ts.URL
	t.Cleanup(func() { dbgAPIURL = oldURL })
	return rec
}

// resetLaunchFlags 复位**这一族**起会话命令读的全部包级开关。
//
// 除了两个寻址标志与根命令的 `--env`（`resetTargetFlags` 管那三个），这里还要复位
// 子命令自己的 `--env` / `--ssh` / `--module`：它们都是包级变量（cobra 绑在上面），
// 前一条用例留下的值会把后一条悄悄带跑。
func resetLaunchFlags(t *testing.T) {
	t.Helper()
	resetTargetFlags(t)
	oldEnv, oldSSH, oldMod := dbgEnvFlag, dbgSSHName, dbgModule
	dbgEnvFlag, dbgSSHName, dbgModule = "", "", ""
	t.Cleanup(func() { dbgEnvFlag, dbgSSHName, dbgModule = oldEnv, oldSSH, oldMod })
}

// twoIdleSessions 是同环境两条空闲会话（Bug A 的现场：候选 ≥2）。
func twoIdleSessions() []map[string]any {
	return []map[string]any{
		{"id": "s1", "ref": "恒一#1", "display": "恒一/p1#1", "env": "恒一", "n": 1, "state": "idle", "prog": "p1"},
		{"id": "s2", "ref": "恒一#2", "display": "恒一/p2#2", "env": "恒一", "n": 2, "state": "idle", "prog": "p2"},
	}
}

// TestCharStartSendsResolvedSessionToServer 是 Bug A 的 CLI 半边：
// `--session 恒一#2` 必须把那条会话的**服务端原始句柄**送进 POST，且环境也由那条决定。
func TestCharStartSendsResolvedSessionToServer(t *testing.T) {
	resetLaunchFlags(t)
	rec := launchAPIStub(t, twoIdleSessions())
	dbgSessionFlag = "恒一#2"

	if err := debugStartCmd.RunE(debugStartCmd, []string{"my job"}); err == nil {
		t.Fatal("前置不成立：桩对 POST 回的是失败信封，RunE 不该成功")
	}
	if rec.body == nil {
		t.Fatal("没收到 POST /api/sessions —— --session 落位之后必须真的发出这一次启动")
	}
	// 最关键的一条：目标真的送到服务端了（不送 = 服务端照旧按候选收窄 = 本次要修的 bug）。
	if got := rec.body["session"]; got != "s2" {
		t.Errorf("请求体的 session = %v，期望 \"s2\"（`--session 恒一#2` 落上的那一条，C5.4 只认原始句柄）——\n"+
			"    body 里没有它，服务端就会重新按「环境+名字」收窄候选，两条空闲会话时照样报歧义", got)
	}
	// 环境由指名的那条会话决定：只给 --session 时不该让服务端用它自己的当前环境。
	if got := rec.body["ssh"]; got != "恒一" {
		t.Errorf("请求体的 ssh = %v，期望 \"恒一\"（点名一条会话时环境由它决定；不补这一句，\n"+
			"    服务端会用它自己当前的环境组装本轮配置，于是点名恒一#2 却落到别处）", got)
	}
	if got := rec.body["prog"]; got != "my job" {
		t.Errorf("prog = %v，期望 \"my job\"", got)
	}
	// 没打 --name：这个键不该出现（H2 的"空值不送"）。
	if v, ok := rec.body["name"]; ok {
		t.Errorf("没打 --name 时请求体不该有 name 键，实得 %v", v)
	}
}

// TestCharStartEnvConflictStillFailsLocally 是 Bug A 旁边那条**没有被改**的纪律（F12 / C3.3-1）：
// `--session <另一个环境>#1` 与 `--env <本环境>` 并存时，在**发请求之前**就报 USAGE / 退 2。
//
// 为什么改了这一处之后要专门钉它一次：新增的接线让"落位成功"这条路上多了一次 POST，
// 而 F12 判在落位那一步 —— 一旦有人把落位挪到 POST 之后（"反正服务端也会判"），
// 这条本地防线就悄悄失效了，表现是"点名的会话与 --env 矛盾却真的启动了"。
func TestCharStartEnvConflictStillFailsLocally(t *testing.T) {
	resetLaunchFlags(t)
	rec := launchAPIStub(t, []map[string]any{
		{"id": "s1", "ref": "恒一#1", "env": "恒一", "n": 1, "state": "idle"},
		{"id": "s2", "ref": "测试1012#1", "env": "测试1012", "n": 1, "state": "idle"},
	}, "恒一", "测试1012")
	dbgSessionFlag = "测试1012#1"
	dbgEnvFlag = "恒一"

	err := debugStartCmd.RunE(debugStartCmd, []string{"my job"})
	if err == nil {
		t.Fatal("--session 里的环境与 --env 矛盾却启动了 —— F12 必须判在发请求之前")
	}
	var te *dbgTargetError
	if !asTargetError(err, &te) {
		t.Fatalf("该是目标层的失败（带 code 与退出码），实得 %T: %v", err, err)
	}
	if te.fail.Code != target.CodeUsage {
		t.Errorf("code = %q，F12 要求 %q", te.fail.Code, target.CodeUsage)
	}
	if got := te.ExitCode(); got != 2 {
		t.Errorf("退出码 = %d，F12 那一档是 2", got)
	}
	if rec.body != nil {
		t.Errorf("环境矛盾时一个字节都不该发给服务端（C2.8），实得请求体 %v", rec.body)
	}
}

// TestCharStartWithoutSessionKeepsRequestBody 是 H2 的判据：
// 没打任何新标志时，请求体**逐字**等于今天那一个（只有 prog / module 这些老键）。
func TestCharStartWithoutSessionKeepsRequestBody(t *testing.T) {
	resetLaunchFlags(t)
	rec := launchAPIStub(t, twoIdleSessions())

	if err := debugStartCmd.RunE(debugStartCmd, []string{"my job"}); err == nil {
		t.Fatal("前置不成立：桩对 POST 回的是失败信封，RunE 不该成功")
	}
	if rec.body == nil {
		t.Fatal("没收到 POST /api/sessions")
	}
	if len(rec.body) != 1 || rec.body["prog"] != "my job" {
		t.Errorf("没打 --session / --name 时请求体该只有 prog（与今天逐字相同，H2），实得 %v ——\n"+
			"    多出来的键哪怕只是空串，也会让服务端读到一次「新写法」", rec.body)
	}
}

// TestCharWsdebugNameFlagIsVisibleInHelp 是 Bug B 的 --help 半边。
//
// `wsdebug` 同样是**起会话**的命令（C3.5），所以 `--name` 必须出现在它的帮助里，
// 文案要写着取值纪律（与 `debug start` 同一段话，见 dbgRegisterNameFlag）。
// 反向同时钉住 C1.3：`--session` 仍然**不**挂在这条命令上（它不认会话级目标）。
func TestCharWsdebugNameFlagIsVisibleInHelp(t *testing.T) {
	f := debugWsdebugCmd.Flags().Lookup("name")
	if f == nil {
		t.Fatal("`tt debug wsdebug` 上没有 --name —— 它是起会话的命令（C3.5），\n" +
			"    没有这个标志它起的会话只能用 ref 寻址、也不能当幂等键")
	}
	if !strings.Contains(f.Usage, "别名") {
		t.Errorf("--name 的 help 文案 = %q，该说清它是别名（取值纪律要在 --help 里读得到）", f.Usage)
	}

	// 真正的判据是"敲 --help 看得见"，所以把帮助渲染出来看字面，而不只看 Lookup。
	var b strings.Builder
	debugWsdebugCmd.SetOut(&b)
	debugWsdebugCmd.SetErr(&b)
	t.Cleanup(func() {
		debugWsdebugCmd.SetOut(nil)
		debugWsdebugCmd.SetErr(nil)
	})
	if err := debugWsdebugCmd.Help(); err != nil {
		t.Fatalf("渲染 wsdebug 的帮助失败：%v", err)
	}
	if !strings.Contains(b.String(), "--name") {
		t.Errorf("`tt debug wsdebug --help` 里看不到 --name：\n%s", b.String())
	}
	if debugWsdebugCmd.Flags().Lookup("session") != nil {
		t.Error("`tt debug wsdebug` 不该认 --session（C1.3：它的目标不是某一条会话）")
	}
}

// TestCharWsdebugRejectsBadNameBeforeAnyRequest 钉住 F19 的两个半边：
// **码/退出码**（USAGE / 2）与**一个字节都不发**（C2.8）。
func TestCharWsdebugRejectsBadNameBeforeAnyRequest(t *testing.T) {
	cases := map[string]string{
		"含斜杠":   "a/b",
		"含井号":   "a#b",
		"超过 64": strings.Repeat("长", target.AliasMaxRunes+1),
	}
	for name, bad := range cases {
		t.Run(name, func(t *testing.T) {
			resetLaunchFlags(t)
			var hits int32
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&hits, 1)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"sessions":[]}`))
			}))
			t.Cleanup(ts.Close)
			old := dbgAPIURL
			dbgAPIURL = ts.URL
			t.Cleanup(func() { dbgAPIURL = old })

			dbgNameFlag = bad
			err := debugWsdebugCmd.RunE(debugWsdebugCmd, []string{"12345"})
			if err == nil {
				t.Fatalf("非法别名 %q 该在发请求之前被拒（F19）", bad)
			}
			var te *dbgTargetError
			if !asTargetError(err, &te) {
				t.Fatalf("失败该带 code（Agent 读 code 不读数字，Q15），实得 %T: %v", err, err)
			}
			if te.fail.Code != target.CodeUsage {
				t.Errorf("code = %q，F19 要求 %q", te.fail.Code, target.CodeUsage)
			}
			if got := te.ExitCode(); got != 2 {
				t.Errorf("退出码 = %d，F19 那一档是 2（真用了新写法 --name）", got)
			}
			if n := atomic.LoadInt32(&hits); n != 0 {
				t.Errorf("非法别名发出了 %d 次请求 —— 取值纪律要判在**发请求之前**（C2.8）", n)
			}
		})
	}
}

// TestCharWsdebugSendsNameToServer 是 Bug B 的 CLI 半边：`--name` 随这次起会话送出去。
func TestCharWsdebugSendsNameToServer(t *testing.T) {
	resetLaunchFlags(t)
	rec := launchAPIStub(t, twoIdleSessions())
	dbgNameFlag = "调试-甲"

	if err := debugWsdebugCmd.RunE(debugWsdebugCmd, []string{"12345"}); err == nil {
		t.Fatal("前置不成立：桩对 POST 回的是失败信封，RunE 不该成功")
	}
	if rec.body == nil {
		t.Fatal("没收到 POST /api/wslogs/debug")
	}
	if got := rec.body["name"]; got != "调试-甲" {
		t.Errorf("请求体的 name = %v，期望 \"调试-甲\" —— 不送的话服务端登记不上别名，\n"+
			"    这条会话就只能用 ref 寻址、也不能当幂等键（Q23）", got)
	}
	if got := rec.body["rowid"]; got != "12345" {
		t.Errorf("rowid = %v，期望 \"12345\"（老键不许被新键顶掉）", got)
	}
	// 同环境两条候选、又没点名 session：**不替调用方挑一条**，这个键一个都不该出现。
	if v, ok := rec.body["session"]; ok {
		t.Errorf("落不上目标时不该带 session（那是替调用方挑一条），实得 %v", v)
	}
}

// TestCharWsdebugWithoutNameKeepsRequestBody 是 wsdebug 那一侧的 H2：
// 没打 --name 时请求体逐字等于今天那一个。
func TestCharWsdebugWithoutNameKeepsRequestBody(t *testing.T) {
	resetLaunchFlags(t)
	rec := launchAPIStub(t, twoIdleSessions())

	if err := debugWsdebugCmd.RunE(debugWsdebugCmd, []string{"12345"}); err == nil {
		t.Fatal("前置不成立：桩对 POST 回的是失败信封，RunE 不该成功")
	}
	if rec.body == nil {
		t.Fatal("没收到 POST /api/wslogs/debug")
	}
	if len(rec.body) != 1 || rec.body["rowid"] != "12345" {
		t.Errorf("没打 --name 时请求体该只有 rowid（与今天逐字相同，H2），实得 %v", rec.body)
	}
}
