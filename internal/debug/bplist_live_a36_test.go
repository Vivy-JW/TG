package debug

// A36（契约 `.agents/plan/tt-genero-formdef-contract.md` §A36，2026-10-18 真机）：
// `bp list` 必须反映**当下**，不许拿账本冒充。
//
// 现场：`bp add` 正常、`info breakpoints` 原文里有 `breakpoint already hit 1 time`、
// 点击后确实停站（`停站 …:4130 … reason=breakpoint`）—— 但 `bp list` 前后都显示 `命中 0 次`。
// 根因：`hBPList` 直接交出账本，而账本只在"改断点"那刻回灌（`IsBreakpointCmd`）⇒
// `命中 N 次` 表达的是"上次改断点时的计数"。
//
// 本文件钉四件事（**全部用假内核 + HTTP 桩**：不连机器、不起会话）：
//
//	① 停站态 ⇒ 先回灌（拿内核的 `already hit` / `ignore next` 覆盖账本）再返回；
//	② 运行态 ⇒ **一个字节都不发**（A34：运行期只允许 interrupt），并在响应里如实标注
//	   "可能不是最新 / 先 interrupt" —— 不许假装账本是最新；
//	③ `exec "info breakpoints"` 这种**读**命令也触发回灌（它读的就是内核真相）；
//	④ 既有行为不破：add / del / enable 之后 hits 仍与内核一致；hits>0 时 JSON 里必有该键。
//
// 关于 `hits: 0` 的 JSON 形状：服务端 `Breakpoint.Hits` 带 `omitempty`（A33 的账本兼容口径，
// 见 session.go 该字段的注释），**0 命中时服务端不落这个键**是既有约定，本轮一个字都没动；
// "`--json` 的 hits 键永远在、0 不省键"由 CLI 侧 `bpView`（`internal/cli/debug/bp.go`，无
// `omitempty`）保证，那一条钉在 `internal/cli/debug/bp_test.go` 的 TestBPListJSONCarriesHits。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------- 假内核 ----------

// fakeBP 是假内核眼里的一条断点（就是 `info breakpoints` 会打印的那些列）。
type fakeBP struct {
	num     int
	file    string
	line    int
	module  string
	enabled bool
	hits    int
	ignore  int
	cond    string
	watch   bool
	expr    string // watch 的被监视表达式（= Breakpoint.Func）
}

// fakeKernel 顶掉 SSH PTY（注入 `Session.execFn`），按命令名改自己的断点表并回原文。
//
// 它**记录收到的每一条命令** —— "运行态一个字节都不发"这条断言靠的就是这份记录。
type fakeKernel struct {
	mu   sync.Mutex
	sent []string // 收到的命令原文（按到达顺序）
	bps  []fakeBP // 内核当下的断点表（命中的 hits 由测试自己调高，模拟"点击后命中一次"）
	next int      // 下一个断点编号
	fail error    // 非空 ⇒ 读命令直接失败（模拟内核读不回来）
}

// install 把假内核挂到会话上（生产里 execFn 恒为 nil，走真实 PTY）。
func (k *fakeKernel) install(s *Session) { s.execFn = k.exec }

// seed 直接摆好内核里的断点表（测试的"现场"）。
func (k *fakeKernel) seed(bps ...fakeBP) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.bps = bps
	k.next = 0
	for _, b := range bps {
		if b.num > k.next {
			k.next = b.num
		}
	}
}

// hitTimes 把编号 num 的断点在**内核**里命中 times 次（= 界面上点了、程序停站了）。
func (k *fakeKernel) hitTimes(num, times int) {
	k.mu.Lock()
	defer k.mu.Unlock()
	for i := range k.bps {
		if k.bps[i].num == num {
			k.bps[i].hits += times
		}
	}
}

// calls 返回至今收到的命令。
func (k *fakeKernel) calls() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	out := make([]string, len(k.sent))
	copy(out, k.sent)
	return out
}

// reads 数一下收到过几条读数（`info breakpoints`）。
func (k *fakeKernel) reads() int {
	n := 0
	for _, c := range k.calls() {
		if IsBreakpointReadCmd(c) {
			n++
		}
	}
	return n
}

func (k *fakeKernel) exec(kind, cmd string, mode int, timeout time.Duration) (*execResult, error) {
	k.mu.Lock()
	k.sent = append(k.sent, cmd)
	if k.fail != nil && IsBreakpointReadCmd(cmd) {
		err := k.fail
		k.mu.Unlock()
		return nil, err
	}
	lines := k.replyLocked(cmd)
	k.mu.Unlock()
	return &execResult{Cmd: cmd, Lines: append([]string{cmd}, lines...)}, nil
}

// replyLocked 按 fgldb 的**真实报文形状**回话（A27 / A30 / A33 的实测格式）。
func (k *fakeKernel) replyLocked(cmd string) []string {
	lower := strings.ToLower(cmd)
	switch {
	case IsBreakpointReadCmd(cmd):
		return k.renderLocked()
	case strings.HasPrefix(lower, "break "):
		loc := strings.TrimSpace(cmd[len("break "):])
		if i := strings.Index(lower, " if "); i > 0 {
			loc = strings.TrimSpace(cmd[len("break "):i])
		}
		file, line := "axm_axmt500_wf.4gl", 4130
		if i := strings.LastIndex(loc, ":"); i > 0 {
			file = loc[:i]
			if n, err := strconv.Atoi(loc[i+1:]); err == nil {
				line = n
			}
		}
		k.next++
		k.bps = append(k.bps, fakeBP{num: k.next, file: file, line: line,
			module: "axm_axmt500_wf.axmt500_ui_dialog", enabled: true})
		return []string{fmt.Sprintf("Breakpoint %d at 0x00000000: file %s, line %d.", k.next, file, line)}
	case strings.HasPrefix(lower, "delete "):
		num, _ := strconv.Atoi(strings.TrimSpace(cmd[len("delete "):]))
		kept := k.bps[:0]
		for _, b := range k.bps {
			if b.num != num {
				kept = append(kept, b)
			}
		}
		k.bps = kept
		return nil
	case strings.HasPrefix(lower, "disable "), strings.HasPrefix(lower, "enable "):
		num, _ := strconv.Atoi(strings.TrimSpace(cmd[strings.Index(cmd, " ")+1:]))
		for i := range k.bps {
			if k.bps[i].num == num {
				k.bps[i].enabled = strings.HasPrefix(lower, "enable ")
			}
		}
		return nil
	}
	return nil
}

// renderLocked 打出 `info breakpoints` 的原文（未命中的断点**不打印** already hit 那一行，A33）。
func (k *fakeKernel) renderLocked() []string {
	if len(k.bps) == 0 {
		return []string{"No breakpoints or watchpoints."}
	}
	out := []string{"Num Type           Disp Enb Address    What"}
	for _, b := range k.bps {
		yn := "n"
		if b.enabled {
			yn = "y"
		}
		if b.watch {
			out = append(out, fmt.Sprintf("%d watchpoint keep %s  %s", b.num, yn, b.expr))
			continue
		}
		out = append(out, fmt.Sprintf("%d breakpoint keep %s 0x00000000 in %s at %s:%d",
			b.num, yn, b.module, b.file, b.line))
		if b.cond != "" {
			out = append(out, "\tstop only if "+b.cond)
		}
		if b.ignore > 0 {
			out = append(out, fmt.Sprintf("\tignore next %d hits", b.ignore))
		}
		if b.hits > 0 {
			plural := "time"
			if b.hits > 1 {
				plural = "times"
			}
			out = append(out, fmt.Sprintf("\tbreakpoint already hit %d %s", b.hits, plural))
		}
	}
	return out
}

// ---------- 测试台 ----------

// a36StopBP 是现场那条断点（`停站 …axm_axmt500_wf.4gl:4130 … reason=breakpoint`）。
func a36StopBP() fakeBP {
	return fakeBP{num: 1, file: "axm_axmt500_wf.4gl", line: 4130,
		module: "axm_axmt500_wf.axmt500_ui_dialog", enabled: true}
}

// a36Rig 造一台"起了服务 + 池里一条会话 + 挂了假内核"的测试台。
type a36Rig struct {
	srv  *Server
	sess *Session
	k    *fakeKernel
	ts   *httptest.Server
}

func newA36Rig(t *testing.T, st State, seed ...fakeBP) *a36Rig {
	t.Helper()
	cfg := cfgFor("h1", "36")
	cfg.WatchdogSeconds = 0 // 关掉看门狗，免得测试里冒出定时器
	sess := &Session{
		ID: "s1", cfg: cfg, state: st, started: true,
		bps:      map[int]*Breakpoint{},
		srcCache: map[string]srcCacheEntry{},
	}
	k := &fakeKernel{}
	k.seed(seed...)
	k.install(sess)

	srv := NewServer(cfg, nil, "")
	srv.mgr.sessions[sess.ID] = sess
	mux := http.NewServeMux()
	srv.routes(mux)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return &a36Rig{srv: srv, sess: sess, k: k, ts: ts}
}

// bpList 走**真实路由**取一次 `bp list`（端点注册与路径参数一并被测到）。
func (r *a36Rig) bpList(t *testing.T) map[string]any {
	t.Helper()
	resp := doReq(t, "GET", r.ts.URL+"/api/sessions/s1/breakpoints", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bp list 该退 200，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("bp list 不是合法 JSON：%v", err)
	}
	return raw
}

// post 发一条写请求（人身份 = 纯人工模式，闸门放行）。
func (r *a36Rig) post(t *testing.T, path string, body string) map[string]any {
	t.Helper()
	resp := doReq(t, "POST", r.ts.URL+path, "", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s 该退 200，得 %d（%s）", path, resp.StatusCode, respError(t, resp))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("响应不是合法 JSON：%v", err)
	}
	return raw
}

// bpOf 取一条断点的 JSON 对象（按编号找）。
func bpOf(t *testing.T, raw map[string]any, num int) map[string]any {
	t.Helper()
	list, _ := raw["breakpoints"].([]any)
	for _, it := range list {
		b, _ := it.(map[string]any)
		if n, ok := b["num"].(float64); ok && int(n) == num {
			return b
		}
	}
	t.Fatalf("响应里没有断点 #%d：%v", num, raw)
	return nil
}

// hitsOf 读一条断点的命中计数（键不在 = 0，这正是 A33 的 omitempty 口径）。
func hitsOf(t *testing.T, raw map[string]any, num int) int {
	t.Helper()
	b := bpOf(t, raw, num)
	if v, ok := b["hits"]; ok {
		f, _ := v.(float64)
		return int(f)
	}
	return 0
}

// ---------- ① 停站 + 账本陈旧 ----------

// TestBPListStoppedSyncsFromKernelFirst 是 A36 的主判据：停站态下 `bp list` **先回灌再返回**。
//
// 现场复刻：账本里 Hits=0（上次改断点那刻的计数），内核此刻已经报 `already hit 1 time`
// （程序停过站）。修前 hBPList 直接交账本 ⇒ 打成 `命中 0 次`，与真机表现一模一样。
func TestBPListStoppedSyncsFromKernelFirst(t *testing.T) {
	rig := newA36Rig(t, StateStopped, a36StopBP())
	// 前置：账本陈旧（Hits=0）—— 手工摆成"上次改断点那刻"的样子
	rig.sess.bps[1] = &Breakpoint{Num: 1, File: "axm_axmt500_wf.4gl", Line: 4130,
		Module: "axm_axmt500_wf.axmt500_ui_dialog", Enabled: true, Kind: BPKindBreak}
	// 内核此刻的真相：这条断点已经命中过 1 次
	rig.k.hitTimes(1, 1)

	raw := rig.bpList(t)

	// ① **先**回灌：假内核必须收到一条读数（`info breakpoints`）
	if n := rig.k.reads(); n == 0 {
		t.Fatalf("停站态 bp list 必须先读内核再返回，实际一条读数都没发（命令记录：%v）", rig.k.calls())
	}
	// ② 返回值就是内核当下的计数
	if got := hitsOf(t, raw, 1); got != 1 {
		t.Fatalf("停站态 bp list 应报内核当下的 hits=1，得 %d（A36：不许拿账本冒充）\n    %v", got, raw)
	}
	// ③ 账本自己也被覆盖了（后续快照/落盘读的是同一份事实）
	if got := rig.sess.Breakpoints()[0].Hits; got != 1 {
		t.Fatalf("回灌应覆盖账本：账本 Hits=%d，想 1", got)
	}
	// ④ 如实标注"这份是最新的"
	if fresh, _ := raw["fresh"].(bool); !fresh {
		t.Fatalf("停站态回灌成功后该标 fresh=true，得 %v\n    %v", raw["fresh"], raw)
	}
	if _, ok := raw["note"]; ok {
		t.Fatalf("最新的一份不该带「可能不是最新」的说明：%v", raw["note"])
	}
}

// 内核读不回来（格式不认识/被拒）时：**不许假装是最新**，账本保持原样（不拿空表覆盖）。
func TestBPListStoppedSyncFailureSaysStale(t *testing.T) {
	rig := newA36Rig(t, StateStopped, a36StopBP())
	rig.sess.bps[1] = &Breakpoint{Num: 1, File: "axm_axmt500_wf.4gl", Line: 4130, Enabled: true}
	rig.k.mu.Lock()
	rig.k.fail = fmt.Errorf("行格式不认识（info breakpoints）")
	rig.k.mu.Unlock()

	raw := rig.bpList(t)
	if fresh, _ := raw["fresh"].(bool); fresh {
		t.Fatalf("回灌失败却标 fresh=true（那是在假装账本最新）：%v", raw)
	}
	note, _ := raw["note"].(string)
	if !strings.Contains(note, "可能不是最新") {
		t.Fatalf("回灌失败要如实说「可能不是最新」：%q", note)
	}
	if len(rig.sess.Breakpoints()) != 1 {
		t.Fatalf("回灌失败不该动账本，得 %d 条", len(rig.sess.Breakpoints()))
	}
}

// ---------- ② 运行态 ----------

// TestBPListRunningSendsNoCommandAndSaysStale 钉 A34 + A36 的交界：
// 运行期 fgldb 直接 400「程序运行中,禁止发送调试命令(仅允许中断)」，所以 bp list
// **一个字节都不许发**，同时必须说清"这份账本可能不是最新、先 interrupt"。
func TestBPListRunningSendsNoCommandAndSaysStale(t *testing.T) {
	rig := newA36Rig(t, StateRunning, a36StopBP())
	rig.sess.bps[1] = &Breakpoint{Num: 1, File: "axm_axmt500_wf.4gl", Line: 4130, Enabled: true}
	rig.k.hitTimes(1, 1) // 内核里已经命中过 —— 但运行态读不到，账本仍是 0

	raw := rig.bpList(t)

	if calls := rig.k.calls(); len(calls) != 0 {
		t.Fatalf("运行态 bp list 不许发任何调试命令（A34：只允许 interrupt），实际发了 %v", calls)
	}
	if fresh, _ := raw["fresh"].(bool); fresh {
		t.Fatalf("运行态不可能读内核，却标 fresh=true：%v", raw)
	}
	note, _ := raw["note"].(string)
	for _, want := range []string{"可能不是最新", "interrupt"} {
		if !strings.Contains(note, want) {
			t.Errorf("运行态的说明里必须出现 %q（不许假装账本是最新）：%q", want, note)
		}
	}
	if got, _ := raw["state"].(string); got != string(StateRunning) {
		t.Errorf("响应要带上会话状态让调用方自己也能判断：state=%v", raw["state"])
	}
	// 账本照原样交出（0），不假装读到了内核的 1
	if got := hitsOf(t, raw, 1); got != 0 {
		t.Errorf("运行态只能交账本（Hits=0），得 %d", got)
	}
}

// 非停站的非运行态（idle/loading）同样不发命令，同样标注"可能不是最新"。
func TestBPListNonStoppedStatesSendNoCommand(t *testing.T) {
	for _, st := range []State{StateIdle, StateLoading, StateExit} {
		t.Run(string(st), func(t *testing.T) {
			rig := newA36Rig(t, st, a36StopBP())
			raw := rig.bpList(t)
			if calls := rig.k.calls(); len(calls) != 0 {
				t.Fatalf("状态 %s 下 bp list 一个字节都不该发，实际发了 %v", st, calls)
			}
			if note, _ := raw["note"].(string); !strings.Contains(note, "可能不是最新") {
				t.Fatalf("状态 %s 下要如实标注「可能不是最新」：%q", st, note)
			}
		})
	}
}

// ---------- ③ 读命令也回灌 ----------

// TestExecInfoBreakpointsRefreshesLedger 钉 A36 后半：`exec "info breakpoints"` 这条路
// （`/raw`，kind="other"）也要把账本刷新 —— 它读的就是内核真相。
//
// 走的是"单独判断"路线（`IsBreakpointReadCmd`），所以 `act` 仍是 `raw`：
// 读命令没有改动断点集合，`IsBreakpointCmd` 的语义（会改断点）一个字都不许借用。
func TestExecInfoBreakpointsRefreshesLedger(t *testing.T) {
	rig := newA36Rig(t, StateStopped, a36StopBP())
	rig.sess.bps[1] = &Breakpoint{Num: 1, File: "axm_axmt500_wf.4gl", Line: 4130, Enabled: true}
	rig.k.hitTimes(1, 2) // 内核：已命中 2 次；账本：0

	rig.post(t, "/api/sessions/s1/raw", `{"command":"info breakpoints"}`)

	if got := rig.sess.Breakpoints()[0].Hits; got != 2 {
		t.Fatalf("exec \"info breakpoints\" 之后账本应被回灌成内核的 2，得 %d（命令记录：%v）",
			got, rig.k.calls())
	}
	// 紧接着的 bp list 不必再读一次也能给出同一个数（账本已经是最新的）
	before := rig.k.reads()
	if got := hitsOf(t, rig.bpList(t), 1); got != 2 {
		t.Fatalf("回灌后 bp list 应给出 2，得 %d", got)
	}
	if rig.k.reads() <= before {
		t.Errorf("停站态 bp list 仍应先读内核（A36 第一条），实际读数没增加：%v", rig.k.calls())
	}
}

// 读命令的**判定**：只认 `info breakpoints` 一族，`info locals` 之类不许触发断点回灌。
func TestIsBreakpointReadCmd(t *testing.T) {
	for _, c := range []string{"info breakpoints", "INFO BREAKPOINTS", "info b", "info break",
		"  info breakpoints 2  "} {
		if !IsBreakpointReadCmd(c) {
			t.Errorf("%q 应判定为读断点表", c)
		}
	}
	for _, c := range []string{"info locals", "info variables", "info functions", "info",
		"break 1", "delete 1", "print info breakpoints", ""} {
		if IsBreakpointReadCmd(c) {
			t.Errorf("%q 不该判定为读断点表", c)
		}
	}
	// 与 IsBreakpointCmd 的边界必须互斥：读命令**不是**"会改动断点集合"的那一族
	// （既有语义与既有测试 `TestIsBreakpointCmd` 都钉着这一条）。
	if IsBreakpointCmd("info breakpoints") {
		t.Fatal("`info breakpoints` 不改断点集合，不许被 IsBreakpointCmd 认领")
	}
}

// ---------- ④ 既有行为不破 ----------

// TestBPMutationsKeepHitsRight：add / enable / del 之后 `bp list` 的 hits 仍与内核一致
// （这几条本来就是"改断点即回灌"的路径，本轮的改动不许把它们带坏）。
func TestBPMutationsKeepHitsRight(t *testing.T) {
	rig := newA36Rig(t, StateStopped, a36StopBP())
	rig.k.hitTimes(1, 2) // 现场那条已经命中 2 次
	rig.sess.bps[1] = &Breakpoint{Num: 1, File: "axm_axmt500_wf.4gl", Line: 4130, Enabled: true}

	// bp add：新下的那条内核没有命中过 ⇒ 0；老那条仍是 2
	added := rig.post(t, "/api/sessions/s1/breakpoints",
		`{"location":"axm_axmt500_wf.4gl:5043"}`)
	bp, _ := added["breakpoint"].(map[string]any)
	if n, _ := bp["num"].(float64); int(n) != 2 {
		t.Fatalf("内核该给出 #2，得 %v", added)
	}
	raw := rig.bpList(t)
	if got := hitsOf(t, raw, 1); got != 2 {
		t.Fatalf("bp add 之后老断点的 hits 应仍是 2，得 %d", got)
	}
	if got := hitsOf(t, raw, 2); got != 0 {
		t.Fatalf("新下的断点没命中过，hits 应是 0，得 %d", got)
	}

	// bp enable/disable：不动计数
	rig.post(t, "/api/sessions/s1/breakpoints/1/enabled", `{"enabled":false}`)
	if got := hitsOf(t, rig.bpList(t), 1); got != 2 {
		t.Fatalf("enable/disable 不该改变命中计数，得 %d", got)
	}

	// bp del：剩下的那条计数照旧
	resp := doReq(t, "DELETE", rig.ts.URL+"/api/sessions/s1/breakpoints/2", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bp del 该退 200，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	raw = rig.bpList(t)
	if list, _ := raw["breakpoints"].([]any); len(list) != 1 {
		t.Fatalf("删掉 #2 之后该只剩 1 条，得 %d：%v", len(list), raw)
	}
	if got := hitsOf(t, raw, 1); got != 2 {
		t.Fatalf("bp del 之后剩下的断点 hits 应仍是 2，得 %d", got)
	}
}

// hits>0 时 JSON 里必须有这个键（没命中过的按 A33 的 omitempty 口径不落键，本轮没动）。
func TestBPListJSONCarriesHitsWhenHit(t *testing.T) {
	rig := newA36Rig(t, StateStopped, a36StopBP())
	rig.k.hitTimes(1, 1)

	b := bpOf(t, rig.bpList(t), 1)
	if v, ok := b["hits"]; !ok || v != float64(1) {
		t.Fatalf("命中过 1 次的断点必须带 hits=1，得 %v（键存在=%v）\n    %v", v, ok, b)
	}
}

// 假内核自身的守卫：它必须真的建模了内核（否则上面几条断言都是"空对空"）。
func TestFakeKernelModelsInfoBreakpoints(t *testing.T) {
	k := &fakeKernel{}
	k.seed(fakeBP{num: 1, file: "a.4gl", line: 7, module: "m.f", enabled: true, hits: 3,
		cond: "1=1", ignore: 2})
	bps, err := ParseInfoBreakpoints(k.renderLocked())
	if err != nil {
		t.Fatalf("假内核打出来的原文必须能被真解析器读懂：%v", err)
	}
	if len(bps) != 1 || bps[0].Hits != 3 || bps[0].Cond != "1=1" || bps[0].Ignore != 2 {
		t.Fatalf("假内核的报文与解析器对不上：%+v", bps)
	}
}
