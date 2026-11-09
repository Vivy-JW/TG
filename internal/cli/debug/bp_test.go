package debug

// A33 / A34 的命令面（`tt debug bp`）：两条现场纪律 + 一条接线提示。
//
//	A33 —— 命中计数是内核给的**唯一**命中判据（`Address` 恒 0，不能当判据），
//	       所以 `bp list` 必须把它打出来：内核给了却不显示，等于让人猜。
//	A34 —— running 时下断被服务端 **400** 明确拒绝（"程序运行中,禁止发送调试命令(仅允许中断)"）。
//	       收到它**不重试、不静默**，要给能照做的指引：先 `tt debug interrupt`，停下后再下断。
//	       允许的自动化**只有 interrupt**，绝不自动 continue / run —— 本工具选择"只给指引、
//	       不自动中断"（人在界面上打字时，自动 SIGINT 会把他的对话框取消），并在这里钉住。
//	A35 —— 没有 GDC 接线的会话在 `main` 之后**立刻** `Fatal error -6300` 退出，
//	       所以 `debug start` 就要把这句话说出来，而不是等人下完断点才发现程序已经退了。
//
// 全部用 HTTP 桩（假通道）：不连机器、不起会话。

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/debug"
	"tt/internal/testkit"
)

// bpStub 是本文件的"假通道"：一个只认断点面那几条路径的 HTTP 桩，
// 同时把收到的**每一次**请求记下来 —— "不重试"这条断言靠的就是这份记录。
type bpStub struct {
	mu    sync.Mutex
	calls []string

	sess []map[string]any // GET /api/sessions
	bps  []map[string]any // GET …/breakpoints
	snap map[string]any   // GET /api/sessions/{id}

	// reject 非空 ⇒ 一律回 400 + 这个信封正文（模拟 A34 的 running 拒绝）。
	reject string
}

func (s *bpStub) record(method, uri string) {
	s.mu.Lock()
	s.calls = append(s.calls, method+" "+uri)
	s.mu.Unlock()
}

// count 返回见过的请求（按 METHOD 前缀过滤）。
func (s *bpStub) count(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if strings.HasPrefix(c, method+" ") {
			n++
		}
	}
	return n
}

func (s *bpStub) all() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.calls))
	copy(out, s.calls)
	return out
}

// install 起桩并把它挂成调试服务地址（既有做法见 actor_test.go 的 dbgAPIURL）。
func (s *bpStub) install(t *testing.T) {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.record(r.Method, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		p := r.URL.Path
		switch {
		case s.reject != "":
			w.WriteHeader(400)
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "code": "USAGE", "error": s.reject})
		case strings.HasSuffix(p, "/api/hosts"):
			_ = json.NewEncoder(w).Encode(map[string]any{"sshs": []any{}})
		case strings.HasSuffix(p, "/api/sessions"):
			if r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "sessionId": "sOnly", "state": "loading"})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"sessions": s.sess})
		case strings.HasSuffix(p, "/wait"):
			_ = json.NewEncoder(w).Encode(map[string]any{"timedOut": false})
		case strings.HasSuffix(p, "/breakpoints"):
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "breakpoints": s.bps})
		default:
			if s.snap != nil {
				_ = json.NewEncoder(w).Encode(s.snap)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		}
	}))
	t.Cleanup(ts.Close)

	oldURL := dbgAPIURL
	dbgAPIURL = ts.URL
	t.Cleanup(func() { dbgAPIURL = oldURL })
}

// 两条样例断点：一条命中过（Hits=1，带条件与忽略次数），一条没有（Hits=0，监视点）。
func bpStubSample() []map[string]any {
	return []map[string]any{
		{"num": 1, "kind": "break", "file": "axm_axmt500_wf.4gl", "line": 4130,
			"func": "axm_axmt500_wf.axmt500_ui_dialog", "module": "axm_axmt500_wf.axmt500_ui_dialog",
			"enabled": true, "cond": "1=1", "ignore": 2, "hits": 1},
		{"num": 2, "kind": "watch", "func": "g_current_row", "enabled": true},
	}
}

func newBPStub(t *testing.T) *bpStub {
	t.Helper()
	resetTargetFlags(t)
	s := &bpStub{
		sess: []map[string]any{{"id": "sOnly", "state": "stopped", "env": "主机109", "prog": "axmt500_wf"}},
		bps:  bpStubSample(),
	}
	s.install(t)
	return s
}

// ③ 人读输出必须带命中计数（两条都要：命中过的与没命中的）。
func TestBPListShowsHitsHuman(t *testing.T) {
	newBPStub(t)
	common.JSON = false

	_, out := testkit.CaptureStdout(t, func() int {
		if err := debugBPListCmd.RunE(debugBPListCmd, nil); err != nil {
			t.Fatalf("bp list 不该失败：%v", err)
		}
		return 0
	})

	for _, want := range []string{"命中 1 次", "命中 0 次", "#1", "#2"} {
		if !strings.Contains(out, want) {
			t.Errorf("bp list 的输出里缺 %q（A34：命中计数是内核给的，不显示等于让人猜）：\n%s", want, out)
		}
	}
}

// ③ --json 里也必须带 hits —— 而且 **0 也要出现**（不能让 jq 取不到键）。
func TestBPListJSONCarriesHits(t *testing.T) {
	newBPStub(t)
	oldJSON := common.JSON
	common.JSON = true
	t.Cleanup(func() { common.JSON = oldJSON })

	_, out := testkit.CaptureStdout(t, func() int {
		if err := debugBPListCmd.RunE(debugBPListCmd, nil); err != nil {
			t.Fatalf("bp list --json 不该失败：%v", err)
		}
		return 0
	})

	var got struct {
		Breakpoints []map[string]any `json:"breakpoints"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json 输出不是 JSON：%v\n%s", err, out)
	}
	if len(got.Breakpoints) != 2 {
		t.Fatalf("断点数 = %d，想 2：%s", len(got.Breakpoints), out)
	}
	if v, ok := got.Breakpoints[0]["hits"]; !ok || v != float64(1) {
		t.Errorf("--json 里第 1 条该有 hits=1，实得 %v（键存在=%v）", v, ok)
	}
	// 没命中过的断点也必须带这个键（0 而不是缺失）。
	if v, ok := got.Breakpoints[1]["hits"]; !ok || v != float64(0) {
		t.Errorf("--json 里没命中的断点也该有 hits=0（不许省成缺键，调用方要能直接读）：%v（键存在=%v）",
			v, ok)
	}
}

// ④ 400「程序运行中」⇒ 错误里必须有**可执行指引**（含 interrupt），不是一句"失败"。
// 同时钉住"不重试"与"绝不自动 continue/run"：桩只该收到一次 POST，且没有任何放行命令。
func TestBPRunningRejectTellsInterrupt(t *testing.T) {
	resetTargetFlags(t)
	s := &bpStub{reject: "程序运行中,禁止发送调试命令(仅允许中断)"}
	s.install(t)

	err := bpAdd("sOnly", debug.BreakpointOpts{Location: "axm_axmt500_wf.4gl:4130"})
	if err == nil {
		t.Fatal("被 400 拒绝时该报错（不许静默当成功）")
	}
	msg := err.Error()
	if !strings.Contains(msg, "程序运行中") {
		t.Errorf("错误里要保留服务端原文（现场据此判断是状态问题而不是参数问题）：%s", msg)
	}
	if !strings.Contains(msg, "interrupt") {
		t.Fatalf("错误里必须给可执行指引「先 tt debug interrupt，停下后再下断」（A34 冻结行为）：%s", msg)
	}

	if n := s.count("POST"); n > 1 {
		t.Errorf("A34：收到 400 不许自动重试，POST 次数 = %d，想 1\n请求记录：%v", n, s.all())
	}
	for _, c := range s.all() {
		if strings.Contains(c, "raw") || strings.Contains(c, "/interrupt") ||
			strings.Contains(c, "continue") || strings.Contains(c, "run") {
			t.Errorf("本工具只给指引、不自动中断/放行，却发了：%s", c)
		}
	}
}

// 同一条纪律也覆盖 `info breakpoints`（running 时它同样被拒）—— bp list 也要给指引。
func TestBPListRunningRejectTellsInterrupt(t *testing.T) {
	resetTargetFlags(t)
	s := &bpStub{
		sess:   []map[string]any{{"id": "sOnly", "state": "running", "env": "主机109"}},
		reject: "程序运行中,禁止发送调试命令(仅允许中断)",
	}
	s.install(t)

	_, err := bpList("sOnly")
	if err == nil {
		t.Fatal("running 时查断点被拒，该报错")
	}
	if !strings.Contains(err.Error(), "interrupt") {
		t.Fatalf("错误里必须给「先 tt debug interrupt」这条指引：%v", err)
	}
}

// ⑤ A35：没有 GDC 接线时，`debug start` 当场就要说"main 之后立刻 -6300 退出"，
// 而不是等人下完断点才发现程序已经退了。
func TestStartWarnsWhenNoGDCListening(t *testing.T) {
	oldProbe := gdcLocal
	t.Cleanup(func() { gdcLocal = oldProbe })

	runStart := func(t *testing.T, fglserver string) string {
		t.Helper()
		resetTargetFlags(t)
		s := &bpStub{
			sess: []map[string]any{{"id": "sOnly", "state": "stopped", "env": "主机109", "prog": "axmt500_wf"}},
			snap: map[string]any{
				"ok": true, "id": "sOnly", "env": "主机109", "state": "stopped", "prog": "axmt500_wf",
				"module": "axm", "fglserver": fglserver,
				"screenTarget": "画面去向: 未覆盖 FGLSERVER —— 由 T100 按 SSH 来源 IP 自动设置(默认回本机 GDC)",
				"stop":         map[string]any{"file": "axm_axmt500_wf.4gl", "reason": "entry"},
			},
		}
		s.install(t)
		_, out := testkit.CaptureStdout(t, func() int {
			if err := debugStartCmd.RunE(debugStartCmd, []string{"axmt500_wf"}); err != nil {
				t.Fatalf("start 不该失败：%v", err)
			}
			return 0
		})
		return out
	}

	t.Run("没有 GDC 在听 ⇒ 启动输出里有 -6300", func(t *testing.T) {
		gdcLocal = func() bool { return false }
		out := runStart(t, "")
		if !strings.Contains(out, "-6300") {
			t.Fatalf("A35：没有接线时启动输出要说清「main 之后立刻 Fatal error -6300 退出」：\n%s", out)
		}
		if !strings.Contains(out, "GDC") {
			t.Errorf("提示里要能照做（先起 GDC 或用 --fglserver）：\n%s", out)
		}
	})

	t.Run("本轮有 FGLSERVER 覆盖 ⇒ 不吵", func(t *testing.T) {
		gdcLocal = func() bool { return false }
		out := runStart(t, "10.1.25.88:1200")
		if strings.Contains(out, "-6300") {
			t.Fatalf("本轮有接线（FGLSERVER 覆盖）时不该嚷 -6300：\n%s", out)
		}
	})
}

// 假通道自身的守卫：桩必须被真的用过（否则上面几条断言都是"空对空"）。
func TestBPStubServesBreakpointList(t *testing.T) {
	s := newBPStub(t)
	views, err := bpList("sOnly")
	if err != nil {
		t.Fatalf("列断点失败：%v", err)
	}
	if len(views) != 2 || views[0].Hits != 1 || views[1].Hits != 0 {
		t.Fatalf("假通道回的账本没读对：%+v", views)
	}
	if s.count("GET") == 0 {
		t.Fatal("桩一次 GET 都没收到 —— 说明断言没打在那条链路上")
	}
}

// 服务端错误信封要按契约渲染（这里只钉"没被 bp 层吞掉"）。
func TestBPAddEnvelopeErrorSurvives(t *testing.T) {
	resetTargetFlags(t)
	s := &bpStub{reject: "服务端炸了"}
	s.install(t)
	err := bpAdd("sOnly", debug.BreakpointOpts{Location: "1"})
	if err == nil || !strings.Contains(err.Error(), "服务端炸了") {
		t.Fatalf("服务端错误要原样带到调用方：%v", err)
	}
	if bytes.Count([]byte(err.Error()), []byte("interrupt")) > 0 {
		t.Errorf("非「程序运行中」的错误不该套 interrupt 指引：%v", err)
	}
}
