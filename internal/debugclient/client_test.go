package debugclient

// 这组测试盯的是"客户端把服务端的话**原样**传下去"这件事：
// 失败码、候选、hint、退出码一个都不能在客户端这一层被改写或吞掉 ——
// 从前的子进程实现只能靠退出码反推，正是这里要消灭的东西。

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/target"
)

// TestFailureKeepsCodeCandidatesAndHint：失败信封的三个关键部分都要留住。
func TestFailureKeepsCodeCandidatesAndHint(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"ok":false,"code":"SESSION_AMBIGUOUS",
			"error":"有 2 条 SSH 会话","hint":"照抄 ref 重发",
			"candidates":[{"id":"s1","ref":"主机109#1","name":"A","n":1},
			              {"id":"s2","ref":"主机109#2","name":"B","n":2}]}`))
	}))
	defer hs.Close()

	_, err := New(hs.URL).Do(http.MethodGet, "/api/sessions", nil)
	var f *Failure
	if !errors.As(err, &f) {
		t.Fatalf("该是 *Failure，实得 %T: %v", err, err)
	}
	if f.Code != target.CodeAmbiguous {
		t.Errorf("code 被改了：%q", f.Code)
	}
	if len(f.Candidates) != 2 || f.Candidates[0].Ref != "主机109#1" {
		t.Errorf("候选没留住：%+v", f.Candidates)
	}
	if f.Hint == "" {
		t.Error("hint 被丢了（它才是'下一步怎么做'）")
	}
	// 退出码要按码表算（歧义 = 3），不是 HTTP 409。
	if got := f.ExitCode(); got != 3 {
		t.Errorf("ExitCode = %d，要 3", got)
	}
}

// TestNonEnvelopeErrorDoesNotBecomeFailure：不是信封的失败不许伪装成带码失败。
func TestNonEnvelopeErrorDoesNotBecomeFailure(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	}))
	defer hs.Close()

	_, err := New(hs.URL).Do(http.MethodGet, "/api/sessions", nil)
	var f *Failure
	if errors.As(err, &f) {
		t.Fatalf("没有信封时不该造一个 *Failure：%+v", f)
	}
	if err == nil {
		t.Fatal("非 2xx 必须报错")
	}
}

// TestResolveIsStrict：别名 / ref / 句柄都能解析，但**多条命中不自己挑**。
func TestResolveIsStrict(t *testing.T) {
	body := `{"ok":true,"sessions":[
		{"id":"s1","ref":"主机109#1","name":"A","env":"主机109","ent":"99","state":"idle","n":1},
		{"id":"s2","ref":"主机109#2","name":"B","env":"主机109","ent":"99","state":"idle","n":2}]}`
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer hs.Close()
	c := New(hs.URL)

	if b, err := c.Resolve("A"); err != nil || b.ID != "s1" {
		t.Fatalf("按别名解析失败：%v %+v", err, b)
	}
	if b, err := c.Resolve("主机109#2"); err != nil || b.ID != "s2" {
		t.Fatalf("按 ref 解析失败：%v %+v", err, b)
	}
	if b, err := c.Resolve("s1"); err != nil || b.ID != "s1" {
		t.Fatalf("按句柄解析失败：%v %+v", err, b)
	}
	if _, err := c.Resolve("没有这条"); err == nil {
		t.Fatal("找不到时必须报错")
	} else {
		var f *Failure
		if !errors.As(err, &f) || f.Code != target.CodeNotFound {
			t.Fatalf("该报 %q，实得 %v", target.CodeNotFound, err)
		}
	}
}

// TestWaitStoppedPollsUntilStopped：轮询到 stopped 就返回。
func TestWaitStoppedPollsUntilStopped(t *testing.T) {
	var calls int
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		state := "loading"
		if calls >= 2 {
			state = "stopped"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "state": state})
	}))
	defer hs.Close()

	snap, ok, err := New(hs.URL).WaitStopped("s1", 5*time.Second)
	if err != nil || !ok {
		t.Fatalf("应等到停站：ok=%v err=%v", ok, err)
	}
	if stateOf(snap) != "stopped" {
		t.Fatalf("快照状态 = %v", snap["state"])
	}
	if calls < 2 {
		t.Fatalf("应轮询至少两次，实得 %d", calls)
	}
}

// TestWaitStoppedTimeoutIsNotAnError：超时不是错误（调用方照常往下走并如实报超时）。
func TestWaitStoppedTimeoutIsNotAnError(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"state":"running"}`))
	}))
	defer hs.Close()

	_, ok, err := New(hs.URL).WaitStopped("s1", 0)
	if err != nil {
		t.Fatalf("超时不该是错误：%v", err)
	}
	if ok {
		t.Fatal("还在运行不该报停站")
	}
}

// TestDiscoverUsesStateFile：状态文件里的地址 + 前缀优先。
func TestDiscoverUsesStateFile(t *testing.T) {
	dir := t.TempDir()
	st := `{"pid":1,"url":"http://127.0.0.1:29000","apiBase":"/debug/api"}`
	if err := os.WriteFile(filepath.Join(dir, ".tt-serve.json"), []byte(st), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := Discover(dir, ""), "http://127.0.0.1:29000/debug/api"; got != want {
		t.Fatalf("Discover = %q，要 %q", got, want)
	}
}

// TestDiscoverFallsBackToListen：没有状态文件 → 配置 listen；也没有 → 内置默认。
func TestDiscoverFallsBackToListen(t *testing.T) {
	dir := t.TempDir()
	if got, want := Discover(dir, "127.0.0.1:1234"), "http://127.0.0.1:1234"; got != want {
		t.Fatalf("Discover = %q，要 %q", got, want)
	}
	if got := Discover(dir, ""); got == "" || got == "http://" {
		t.Fatalf("没有状态文件也没有 listen 时必须给内置默认，实得 %q", got)
	}
}

// TestLifecycleEndpoints：生命周期动作打到**哪个端点、什么方法、什么请求体**。
//
// 这张映射表是 P2 的关键：从前靠起子进程表达这些动作，现在直连；
// 端点或请求体写错，表现是"动作起不来"，而错误信息往往指向别处。
func TestLifecycleEndpoints(t *testing.T) {
	type call struct {
		method, path string
		body         map[string]any
	}
	var got []call
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		got = append(got, call{r.Method, r.URL.Path, b})
		switch {
		case r.URL.Path == "/api/sessions/switch":
			_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s1","state":"idle"}`))
		case r.URL.Path == "/api/sessions":
			_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s9"}`))
		case strings.HasSuffix(r.URL.Path, "/raw"):
			_, _ = w.Write([]byte(`{"ok":true,"lines":["ok"]}`))
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer hs.Close()
	c := New(hs.URL)

	id, err := c.SwitchToIdle("主机109", "99", "录像A", time.Second)
	if err != nil || id != "s1" {
		t.Fatalf("SwitchToIdle = %q, %v", id, err)
	}
	if err := c.Topent("s1", "99"); err != nil {
		t.Fatal(err)
	}
	fgl := "10.0.0.1:1200"
	sid, err := c.Launch(LaunchReq{Env: "主机109", Prog: "bsft001_wf", Module: "asf", Name: "录像A", Ent: "99", FGLServer: &fgl})
	if err != nil || sid != "s9" {
		t.Fatalf("Launch = %q, %v", sid, err)
	}
	if lines, err := c.Exec("s9", "break 812"); err != nil || len(lines) != 1 {
		t.Fatalf("Exec = %v, %v", lines, err)
	}
	if err := c.Continue("s9"); err != nil {
		t.Fatal(err)
	}
	if err := c.Quit("s9"); err != nil {
		t.Fatal(err)
	}

	want := []struct {
		method, path string
	}{
		{"POST", "/api/sessions/switch"},
		{"POST", "/api/sessions/s1/topent"},
		{"POST", "/api/sessions"},
		{"POST", "/api/sessions/s9/raw"},
		{"POST", "/api/sessions/s9/control"},
		{"DELETE", "/api/sessions/s9"},
	}
	if len(got) != len(want) {
		t.Fatalf("打了 %d 次，要 %d 次：%+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].method != w.method || got[i].path != w.path {
			t.Errorf("第 %d 次 = %s %s，要 %s %s", i, got[i].method, got[i].path, w.method, w.path)
		}
	}
	// 请求体里几个关键字段：起会话必须带 ent / ssh / fglserver，放行必须是 continue。
	if got[2].body["ent"] != "99" || got[2].body["ssh"] != "主机109" || got[2].body["fglserver"] != fgl {
		t.Errorf("起会话请求体不对：%+v", got[2].body)
	}
	if got[4].body["action"] != "continue" {
		t.Errorf("放行动作不对：%+v", got[4].body)
	}
	if got[3].body["command"] != "break 812" {
		t.Errorf("exec 命令体不对：%+v", got[3].body)
	}
}

// TestFailureWritesItsOwnEnvelope：**机器码必须原样传到 CLI 输出**。
//
// 这条防的是"REST 上是 SESSION_NOT_FOUND、命令行里变成 USAGE"那种漂：
// 靠 code 分支的调用方只能去啃中文，而那正是契约要消灭的东西。
func TestFailureWritesItsOwnEnvelope(t *testing.T) {
	f := &Failure{
		Code:    target.CodeAmbiguous,
		Message: "有 2 条会话",
		Hint:    "照抄 ref 重发",
		Candidates: []target.Candidate{
			{ID: "s1", Ref: "主机109#1"}, {ID: "s2", Ref: "主机109#2"},
		},
	}
	var buf bytes.Buffer
	if err := f.WriteEnvelope(&buf); err != nil {
		t.Fatal(err)
	}
	var out struct {
		OK         bool               `json:"ok"`
		Code       string             `json:"code"`
		Error      string             `json:"error"`
		Hint       string             `json:"hint"`
		ExitCode   int                `json:"exitCode"`
		Candidates []target.Candidate `json:"candidates"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("信封不是合法 JSON：%v", err)
	}
	if out.OK || out.Code != target.CodeAmbiguous || out.Error != "有 2 条会话" || out.Hint != "照抄 ref 重发" {
		t.Fatalf("信封字段不对：%+v", out)
	}
	if len(out.Candidates) != 2 || out.Candidates[0].Ref != "主机109#1" {
		t.Fatalf("候选丢了：%+v", out.Candidates)
	}
	if out.ExitCode != f.ExitCode() {
		t.Fatalf("退出码不一致：%d vs %d", out.ExitCode, f.ExitCode())
	}
}

// TestStartJobEndpoint：`POST /sessions/{id}/job` 的路径与请求体（fglserver 可选）。
func TestStartJobEndpoint(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte(`{"ok":true,"sessionId":"s1"}`))
	}))
	defer hs.Close()

	id, err := New(hs.URL).StartJob("s1", "bsft001_wf", "asf", "99", "10.0.0.1:1200")
	if err != nil || id != "s1" {
		t.Fatalf("StartJob = %q, %v", id, err)
	}
	if gotPath != "POST /api/sessions/s1/job" {
		t.Fatalf("路径 = %q", gotPath)
	}
	if gotBody["prog"] != "bsft001_wf" || gotBody["module"] != "asf" || gotBody["ent"] != "99" || gotBody["fglserver"] != "10.0.0.1:1200" {
		t.Fatalf("请求体 = %+v", gotBody)
	}

	// 没回 sessionId 要报错（而不是让调用方拿着空句柄继续）
	hs2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer hs2.Close()
	if _, err := New(hs2.URL).StartJob("s1", "p", "", "", ""); err == nil {
		t.Fatal("没回 sessionId 该报错")
	}
}

// TestCloseUsesCloseEndpoint：`Close` 打的是 `/close`（收掉会话），不是 DELETE（结束本轮）。
func TestCloseUsesCloseEndpoint(t *testing.T) {
	var got string
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer hs.Close()
	if err := New(hs.URL).Close("s9"); err != nil {
		t.Fatal(err)
	}
	if got != "POST /api/sessions/s9/close" {
		t.Fatalf("Close 打的是 %q", got)
	}
}

// TestNilFailureExitCode：nil 失败不许 panic（收尾路径上会碰到）。
func TestNilFailureExitCode(t *testing.T) {
	var f *Failure
	if got := f.ExitCode(); got != 1 {
		t.Fatalf("nil Failure 的退出码 = %d，要 1", got)
	}
	if err := f.WriteEnvelope(io.Discard); err != nil {
		t.Fatalf("nil Failure 写信封不该报错：%v", err)
	}
}

// TestJoinBaseDoesNotDoublePrefix：状态文件给的地址已经指到 API 上时，不再拼前缀。
func TestJoinBaseDoesNotDoublePrefix(t *testing.T) {
	if got := joinBase("http://h:1/debug/api", "/debug"); got != "http://h:1/debug/api" {
		t.Fatalf("joinBase = %q", got)
	}
	if got := joinBase("http://h:1", "/debug"); got != "http://h:1/debug" {
		t.Fatalf("joinBase = %q", got)
	}
}

// TestWaitStoppedExitIsAnError：会话断开不是"超时"，要报错（调用方据此不再等）。
func TestWaitStoppedExitIsAnError(t *testing.T) {
	hs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true,"state":"exit"}`))
	}))
	defer hs.Close()
	if _, ok, err := New(hs.URL).WaitStopped("s1", time.Second); err == nil || ok {
		t.Fatalf("exit 该报错：ok=%v err=%v", ok, err)
	}
}

// TestResolveEmptyTarget：空目标直接判用法错（不去列会话）。
func TestResolveEmptyTarget(t *testing.T) {
	c := New("http://127.0.0.1:1")
	var f *Failure
	if _, err := c.Resolve("  "); !errors.As(err, &f) || f.Code != target.CodeUsage {
		t.Fatalf("空目标该报 %s，实得 %v", target.CodeUsage, err)
	}
}

// TestDoUnreachable：连不上服务时的错误要带地址与"先起服务"的提示。
func TestDoUnreachable(t *testing.T) {
	c := New("http://127.0.0.1:1")
	_, err := c.Sessions()
	if err == nil || !strings.Contains(err.Error(), "无法连接调试服务") {
		t.Fatalf("连不上该有明确提示：%v", err)
	}
}

// TestFailureErrorIncludesHint：人读错误要把 hint 带出来（命令行散文模式就靠它）。
func TestFailureErrorIncludesHint(t *testing.T) {
	f := &Failure{Code: target.CodeAmbiguous, Message: "有两条", Hint: "照抄 ref 重发"}
	if got := f.Error(); !strings.Contains(got, "有两条") || !strings.Contains(got, "照抄 ref 重发") {
		t.Fatalf("Error() = %q", got)
	}
}
