package session

// 这组判据盯的是"命令 → 端点"的对应关系：新命令面每个动词打哪个端点、带什么参数。
//
// 为什么不做成"跑真服务"的集成测试：会话的动作都要真 SSH + 真 fgldb，单测跑不起；
// 而真正容易写错的是**接线**（打错端点、参数名写错、rm 用了 quit 的端点），
// 用一个假服务就能钉死。语义对不对由服务端自己的判据负责。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/debugclient"
	"tt/internal/testkit"
)

type recFake struct {
	mu    sync.Mutex
	calls []string
	bod   []map[string]any
}

func (f *recFake) install(t *testing.T) {
	t.Helper()
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		f.bod = append(f.bod, body)
		f.mu.Unlock()

		switch {
		case r.URL.Path == "/api/sessions" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"ok":true,"sessions":[
				{"id":"s1","ref":"主机109#1","name":"A","env":"主机109","ent":"99","n":1,"epoch":"3","state":"idle","prog":"bsft001_wf"}]}`))
		case r.URL.Path == "/api/sessions" && r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s9"}`))
		case r.URL.Path == "/api/sessions/switch":
			_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s1","state":"idle"}`))
		case strings.HasSuffix(r.URL.Path, "/job"):
			_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s1","session":"主机109#1"}`))

		case strings.HasPrefix(r.URL.Path, "/api/sessions/s1") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"ok":true,"id":"s1","ref":"主机109#1","state":"stopped","stop":{"file":"a.4gl","line":812}}`))
		case strings.HasPrefix(r.URL.Path, "/api/sessions/s9") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"ok":true,"id":"s9","ref":"主机109#2","state":"stopped"}`))

		case strings.HasSuffix(r.URL.Path, "/raw"):
			_, _ = w.Write([]byte(`{"ok":true,"lines":["Breakpoint 1 at a.4gl:812"]}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	t.Cleanup(hs.Close)

	old := newClient
	newClient = func() (*debugclient.Client, error) { return debugclient.New(hs.URL), nil }
	t.Cleanup(func() { newClient = old })
}

func (f *recFake) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.calls))
	copy(out, f.calls)
	return out
}

func (f *recFake) bodies() []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]map[string]any, len(f.bod))
	copy(out, f.bod)
	return out
}

// resetFlags 把命令级标志清回默认（同一个进程里反复 Execute 时必做）。
func resetFlags() {
	nsEnv, nsEnt, nsName, nsProg, nsModule = "", "", "", "", ""
	lsEnv, lsSession, entValue = "", "", ""
	lsAll, entClear = false, false
}

// runCmd 跑一次子命令，返回 stdout。
func runCmd(t *testing.T, name string, args ...string) string {
	t.Helper()
	// 用组本身当入口（cobra 支持的用法）：直接 Execute 子命令会走另一条解析路径，
	// 而且会把 -- 之后的透传参数吃掉。
	//
	// **每个用例先清标志**：cobra 的 flag 值挂在命令对象上，同一个测试进程里再执行一次
	// 会继承上一次的值（`--env` 没给却还是旧的）—— 那会让"必填项校验"这条判据假绿。
	resetFlags()
	group.SetArgs(append([]string{name}, args...))
	_, out := testkit.CaptureStdout(t, func() int {
		if err := group.Execute(); err != nil {
			t.Fatalf("%s %v 报错：%v", name, args, err)
		}
		return 0
	})
	return out
}

// TestSessionNewIdle：不給 --prog → 建**空闲**会话（走 switch），并回显身份。
func TestSessionNewIdle(t *testing.T) {
	f := &recFake{}
	f.install(t)

	out := runCmd(t, "new", "--env", "主机109", "--ent", "99", "--name", "A")
	if !strings.Contains(out, `"ref":"主机109#1"`) {
		t.Fatalf("应回显落在哪条会话（ref）：%s", out)
	}
	got := f.snapshot()
	if len(got) < 1 || got[0] != "POST /api/sessions/switch" {
		t.Fatalf("空闲会话该走 switch，实得 %v", got)
	}
	if b := f.bodies()[0]; b["ent"] != "99" || b["name"] != "A" || b["env"] != "主机109" {
		t.Fatalf("switch 请求体缺字段：%+v", b)
	}
}

// TestSessionNewWithProg：给了 --prog → 走起会话 + 等入口停站。
func TestSessionNewWithProg(t *testing.T) {
	f := &recFake{}
	f.install(t)

	runCmd(t, "new", "--env", "主机109", "--ent", "99", "--prog", "bsft001_wf", "-m", "asf")
	got := f.snapshot()
	if len(got) < 2 || got[0] != "POST /api/sessions" {
		t.Fatalf("有作业时该先 POST /api/sessions，实得 %v", got)
	}
	if b := f.bodies()[0]; b["prog"] != "bsft001_wf" || b["ent"] != "99" || b["module"] != "asf" {
		t.Fatalf("起会话请求体缺字段：%+v", b)
	}
}

// TestSessionNewRequiresEnvAndEnt：两个必填项缺一不可（ENT 决定账号与据点校验）。
func TestSessionNewRequiresEnvAndEnt(t *testing.T) {
	f := &recFake{}
	f.install(t)

	for _, args := range [][]string{
		{"--ent", "99"},    // 缺 env
		{"--env", "主机109"}, // 缺 ent
	} {
		resetFlags() // 同一进程里反复 Execute：先清掉上一次的标志值，否则"缺 env"测不出来
		group.SetArgs(append([]string{"new"}, args...))
		if err := group.Execute(); err == nil {
			t.Fatalf("%v 应当报错（必填项缺失）", args)
		}
	}
}

// TestSessionLsAndShow：列表与详情各自打对端点。
func TestSessionLsAndShow(t *testing.T) {
	f := &recFake{}
	f.install(t)

	out := runCmd(t, "ls")
	if !strings.Contains(out, `"ref":"主机109#1"`) {
		t.Fatalf("ls 应列出会话：%s", out)
	}

	out = runCmd(t, "show", "--session", "A")
	if !strings.Contains(out, `"line":812`) {
		t.Fatalf("show 应给出停站现场：%s", out)
	}

}

// TestSessionEntQuitRm：三条动作分别打 topent / quit(DELETE) / close(POST)。
//
// 特别钉住 rm：它必须走 `/close` 而不是 `DELETE` —— 后者是"结束本轮、会话保留"。
// 两个动作混用的后果是"以为收掉了，其实还留着一个开着的会话"。
func TestSessionEntQuitRm(t *testing.T) {
	f := &recFake{}
	f.install(t)

	runCmd(t, "ent", "99", "--session", "A")
	runCmd(t, "quit", "--session", "A")
	runCmd(t, "rm", "--session", "A")

	got := f.snapshot()
	joined := strings.Join(got, " | ")
	if !strings.Contains(joined, "POST /api/sessions/s1/topent") {
		t.Errorf("ent 应打 topent：%s", joined)
	}
	if !strings.Contains(joined, "DELETE /api/sessions/s1") {
		t.Errorf("quit 应打 DELETE /api/sessions/{id}（结束本轮、会话保留）：%s", joined)
	}
	if !strings.Contains(joined, "POST /api/sessions/s1/close") {
		t.Errorf("rm 应打 POST /api/sessions/{id}/close（收掉会话）：%s", joined)
	}
}

// TestSessionExecPassthrough：透传命令原样送出去，输出原样回。
func TestSessionExecPassthrough(t *testing.T) {
	f := &recFake{}
	f.install(t)

	out := runCmd(t, "exec", "--session", "A", "--", "break 812")
	if !strings.Contains(out, "Breakpoint 1 at a.4gl:812") {
		t.Fatalf("透传输出丢了：%s", out)
	}
	bodies := f.bodies()
	var rawBody map[string]any
	for i, c := range f.snapshot() {
		if strings.HasSuffix(c, "/raw") {
			rawBody = bodies[i]
		}
	}
	if rawBody == nil || rawBody["command"] != "break 812" {
		t.Fatalf("透传命令没送到 /raw：%+v", rawBody)
	}
}

// TestSessionJob：`session job` 打的是 `/sessions/{id}/job`，带 prog/module/ent。
//
// 这条命令补的是"会话已经在了，只在它里面换作业跑"——与 `new --prog` 的分工见命令帮助。
func TestSessionJob(t *testing.T) {
	f := &recFake{}
	f.install(t)

	out := runCmd(t, "job", "--session", "A", "--prog", "bsft001_wf", "-m", "asf", "--ent", "99")
	if !strings.Contains(out, `"ok":true`) || !strings.Contains(out, "bsft001_wf") {
		t.Fatalf("输出不对：%s", out)
	}
	var hit map[string]any
	for i, c := range f.snapshot() {
		if strings.HasSuffix(c, "/job") {
			hit = f.bodies()[i]
		}
	}
	if hit == nil {
		t.Fatalf("没有打到 /job：%v", f.snapshot())
	}
	if hit["prog"] != "bsft001_wf" || hit["module"] != "asf" || hit["ent"] != "99" {
		t.Fatalf("请求体不对：%+v", hit)
	}

	// 缺 --prog：一个请求都不该发
	before := len(f.snapshot())
	resetFlags()
	group.SetArgs([]string{"job", "--session", "A"})
	if err := group.Execute(); err == nil {
		t.Fatal("缺 --prog 应报错")
	}
	if len(f.snapshot()) != before {
		t.Fatalf("缺 --prog 时不该发请求：%v", f.snapshot())
	}
}

// TestSessionLsFiltersAndFormats：`--env` 过滤、`-a` 含已断开、table/csv 输出。
//
// 这三件事都属"参数给了却没生效"那一类 —— 人不会报错，只会觉得命令怪。
func TestSessionLsFiltersAndFormats(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"sessions":[
			{"id":"s1","ref":"主机109#1","name":"A","env":"主机109","ent":"99","n":1,"state":"idle","prog":"p1"},
			{"id":"s2","ref":"测试1012#1","name":"B","env":"测试1012","ent":"88","n":1,"state":"exit","prog":"p2"}]}`))
	}))
	defer hs.Close()
	old := newClient
	newClient = func() (*debugclient.Client, error) { return debugclient.New(hs.URL), nil }
	t.Cleanup(func() { newClient = old })

	resetFlags()
	group.SetArgs([]string{"ls", "--env", "主机109"})
	_, out := testkit.CaptureStdout(t, func() int {
		if err := group.Execute(); err != nil {
			t.Fatalf("ls --env 报错：%v", err)
		}
		return 0
	})
	if strings.Contains(out, "测试1012") || !strings.Contains(out, "主机109#1") {
		t.Fatalf("--env 没生效：%s", out)
	}

	// 默认不含已断开
	resetFlags()
	group.SetArgs([]string{"ls"})
	_, out = testkit.CaptureStdout(t, func() int { _ = group.Execute(); return 0 })
	if strings.Contains(out, "测试1012#1") {
		t.Fatalf("默认不该列出已断开的会话：%s", out)
	}
	// -a 含
	resetFlags()
	group.SetArgs([]string{"ls", "-a"})
	_, out = testkit.CaptureStdout(t, func() int { _ = group.Execute(); return 0 })
	if !strings.Contains(out, "测试1012#1") {
		t.Fatalf("-a 应含已断开的会话：%s", out)
	}
}

// TestSessionLsTableAndCsv：`--format table` 出表格、`--format csv` 出纯 CSV（不带 SQL 表头注释）。
func TestSessionLsTableAndCsv(t *testing.T) {
	f := &recFake{}
	f.install(t)
	oldFormat := common.Format
	t.Cleanup(func() { common.Format = oldFormat })

	common.Format = "table"
	resetFlags()
	group.SetArgs([]string{"ls"})
	_, out := testkit.CaptureStdout(t, func() int { _ = group.Execute(); return 0 })
	if !strings.Contains(out, "REF") || !strings.Contains(out, "主机109#1") {
		t.Fatalf("table 形态不对：%s", out)
	}

	common.Format = "csv"
	resetFlags()
	group.SetArgs([]string{"ls"})
	_, out = testkit.CaptureStdout(t, func() int { _ = group.Execute(); return 0 })
	if strings.Contains(out, "只读事务") || strings.Contains(out, "行数") {
		t.Fatalf("csv 不该带 SQL 表头注释：%s", out)
	}
	if !strings.HasPrefix(out, "REF,") {
		t.Fatalf("csv 应以表头开头：%s", out)
	}
}

// TestSessionArgsErrors：缺参/缺命令时都要报错，且**不发请求**。
func TestSessionArgsErrors(t *testing.T) {
	f := &recFake{}
	f.install(t)
	before := len(f.snapshot())

	for _, args := range [][]string{
		{"show"},                   // 缺 --session
		{"quit"},                   // 缺 --session
		{"exec", "--session", "A"}, // 缺命令
		{"new", "--ent", "99"},     // 缺 env
	} {
		resetFlags()
		group.SetArgs(args)
		if err := group.Execute(); err == nil {
			t.Fatalf("%v 应报错", args)
		}
	}
	if len(f.snapshot()) != before {
		t.Fatalf("参数错时不该发请求：%v", f.snapshot())
	}
}

// TestSessionActionsRespectFormat：`--format table` 对动作类命令也要生效。
//
// 之前只有 ls/show/job 走了格式化出口，ent/quit/rm/exec 仍硬打 JSON ——
// 全局开关被**静默忽略**，正是这套工具一直在灭的毛病。
func TestSessionActionsRespectFormat(t *testing.T) {
	f := &recFake{}
	f.install(t)
	oldFormat := common.Format
	t.Cleanup(func() { common.Format = oldFormat })
	common.Format = "table"

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"ent", "99", "--session", "A"}, "企业编号：99"},
		{[]string{"quit", "--session", "A"}, "已结束会话"},
		{[]string{"rm", "--session", "A"}, "已收掉会话"},
		{[]string{"exec", "--session", "A", "--", "break 812"}, "Breakpoint 1 at"},
	}
	for _, c := range cases {
		resetFlags()
		group.SetArgs(c.args)
		_, out := testkit.CaptureStdout(t, func() int { _ = group.Execute(); return 0 })
		if !strings.Contains(out, c.want) {
			t.Errorf("%v 的 table 输出应含 %q，实得 %q", c.args, c.want, out)
		}
		if strings.Contains(out, `"ok":`) {
			t.Errorf("%v 在 table 模式下不该打 JSON：%q", c.args, out)
		}
	}
}
