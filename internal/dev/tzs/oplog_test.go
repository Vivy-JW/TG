package tzs

// oplog_test.go —— `op` + `list_ops`：超时之后"那次写到底进去没有"的机械答案。
//
// 为什么这条值得一份文件：这三态（没有记录 = 请求从未到达；pending = 看见了、还在做；
// ok/error = 已经结束）是**唯一**能让调用方决定"能不能重发"的东西。协议里没有幂等键，从前的
// 规则只有一句"请求一旦上线绝不重试" —— 那条治的是重复，没治"不知道"。所以这里把三态里能
// 在测试里造出来的两态都钉住，第三态（pending）只在超时里出现，见下面注释。
//
// 判据里有一条不是关于日志的，而是关于**谁会被记**：读动词带着 op 不许长日志。`list_ops`
// 自己就有一个叫 op 的参数（它是过滤器），而调度器第一版按"参数里有没有 op"来认写动词 ——
// 那次查询于是把自己当成一次写操作记进了日志（实测）。修法是在 SpecFn 上立 Traced 标志，
// 这条判据守的就是它。

import (
	"os"
	"path/filepath"
	"testing"
)

type opEntry struct {
	Seq     int64          `json:"seq"`
	Op      string         `json:"op"`
	Fn      string         `json:"fn"`
	Handle  string         `json:"handle"`
	Program string         `json:"program"`
	State   string         `json:"state"`
	DryRun  bool           `json:"dryRun"`
	At      string         `json:"at"`
	Ms      float64        `json:"ms"`
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Summary map[string]any `json:"summary"`
}

type listOpsReply struct {
	DaemonStartedAt string    `json:"daemonStartedAt"`
	Count           int       `json:"count"`
	Returned        int       `json:"returned"`
	Truncated       bool      `json:"truncated"`
	Dropped         int       `json:"dropped"`
	Ops             []opEntry `json:"ops"`
	Note            string    `json:"note"`
}

func TestOpLogAnswersWhatHappened(t *testing.T) {
	exe, ws, _ := requireE2E(t)
	// normalizeWS 与命令层用的是同一个：工作区那一侧由我们统一（server.go 的注释解释了
	// 为什么 —— 设计器归一化包路径但不归一化工作区字符串，正斜杠的 TTZS_WS 会让它拒掉
	// 自己算出来的每一个包路径）。
	ws = normalizeWS(ws)
	s := startStdio(t, exe, ws, perFileBudget)
	defer s.close()

	// 包从 list_packages 找 —— 这样这条用例只需要 TTZS_WS，不必再要一份语料。
	lp, ok := ask[listPackagesReply](t, s, "listpackages", "list_packages", map[string]any{"limit": 1})
	if !ok || len(lp.Packages) == 0 {
		t.Skipf("工作区 %s 里没有包，这条用例没法跑", ws)
	}
	pkg := lp.Packages[0].Path
	o, ok := ask[openReply](t, s, "open", "open", map[string]any{"path": pkg})
	if !ok {
		t.Fatalf("打不开 %s", pkg)
	}
	h := o.Handle
	defer s.call("close", map[string]any{"handle": h})

	out := filepath.Join(ws, "_tdev_oplog_"+itoaPID()+".tzs")
	defer os.Remove(out)

	// ---- ① 成功的一次写：日志必须能答"写进去没有"，而且要带上摘要
	sv, ok := ask[saveReply](t, s, "save ok", "save", map[string]any{"handle": h, "out": out, "op": "op-ok"})
	if !ok {
		t.Fatal("save 失败")
	}
	if sv.Sha256 == "" {
		t.Fatal("save 没回 sha256，后面没法比对")
	}
	rep := opsFor(t, s, "op-ok")
	if rep.Count != 1 || len(rep.Ops) != 1 {
		t.Fatalf("op=op-ok 该只有一条，得到 count=%d returned=%d", rep.Count, len(rep.Ops))
	}
	e := rep.Ops[0]
	if e.State != "ok" {
		t.Errorf("state=%q，该是 ok", e.State)
	}
	if e.Fn != "save" || e.Handle != h {
		t.Errorf("记录里的 fn/handle 不对：fn=%q handle=%q（打开的是 %q）", e.Fn, e.Handle, h)
	}
	if e.Program != o.Program {
		t.Errorf("记录里的 program=%q，open 回的是 %q", e.Program, o.Program)
	}
	if got, _ := e.Summary["sha256"].(string); got != sv.Sha256 {
		t.Errorf("摘要里的 sha256=%q，save 回的是 %q —— 超时之后要拿它去比盘上的字节", got, sv.Sha256)
	}

	// ---- ② 读动词带着 op 不长日志（`list_ops` 自己的参数就叫 op，它是过滤器）
	if got := opsFor(t, s, "op-ok").Count; got != 1 {
		t.Errorf("问了一次 op-ok 之后它变成 %d 条 —— 读动词把自己记进日志了", got)
	}
	all := opsAll(t, s, 0)
	for _, x := range all.Ops {
		if x.Fn == "list_ops" {
			t.Errorf("日志里出现了 list_ops 自己（seq=%d op=%q）—— 查询不是写操作", x.Seq, x.Op)
		}
	}

	// ---- ③ 失败的一次写也要有记录（"重发安全"要靠它）
	//
	// 这里不能用 ask：那个 helper 见到 error 帧就 t.Errorf（对绝大多数用例是对的），
	// 而这一条要的**就是**一次拒绝。
	errFrame, err := s.call("set_spec_attr", map[string]any{
		"handle": h, "path": "managedform/nope/nothing", "kind": "field",
		"attr": "can_edit", "value": "N", "op": "op-err"})
	if err != nil || errFrame == nil {
		t.Fatalf("发请求失败：%v", err)
	}
	if errFrame.OK {
		t.Fatal("往一个不存在的路径写居然成功了？这条用例需要一次失败")
	}
	rep = opsFor(t, s, "op-err")
	if rep.Count != 1 {
		t.Fatalf("失败的写没有留下记录（count=%d）—— 那超时之后就没有答案", rep.Count)
	}
	if e := rep.Ops[0]; e.State != "error" || e.Code == "" {
		t.Errorf("失败的记录该是 state=error 且带 code，得到 state=%q code=%q", e.State, e.Code)
	}

	// ---- ④ 干跑也记，并且标明是干跑；它不写盘
	dryOut := filepath.Join(ws, "_tdev_oplog_dry_"+itoaPID()+".tzs")
	drep, ok := ask[dryReply](t, s, "save dry", "save", map[string]any{
		"handle": h, "out": dryOut, "op": "op-dry", "dry_run": true})
	if !ok {
		t.Fatal("save --dry-run 失败")
	}
	if _, err := os.Stat(dryOut); err == nil {
		t.Errorf("dry-run 的 save 把 %s 写出来了", dryOut)
		_ = os.Remove(dryOut)
	}
	rep = opsFor(t, s, "op-dry")
	if rep.Count != 1 || !rep.Ops[0].DryRun {
		t.Errorf("干跑的记录该带 dryRun:true，得到 count=%d dryRun=%v", rep.Count, rep.Ops[0].DryRun)
	}
	if got, _ := rep.Ops[0].Summary["written"].(bool); got {
		t.Errorf("干跑的摘要里 written=true —— 那正是它该拦下来的东西（reverted=%v）", drep.Reverted.Mode)
	}

	// ---- ⑤ 没发生过的 op：count=0，而且 note 要说清"这个进程没看见过"
	rep = opsFor(t, s, "op-never-sent")
	if rep.Count != 0 || len(rep.Ops) != 0 {
		t.Errorf("没发过的 op 回了 %d 条", rep.Count)
	}
	if rep.DaemonStartedAt == "" {
		t.Errorf("没给 daemonStartedAt —— 换过守护进程时这是唯一能自证的字段（日志只记本进程）")
	}
	if len(rep.Note) < 10 {
		t.Errorf("count=0 时的 note 太短，说不清「没有记录」是什么意思：%q", rep.Note)
	}

	// ---- ⑥ 翻页算术：count 是匹配到的条数，returned 是回给你的条数
	all = opsAll(t, s, 1)
	if all.Returned != 1 {
		t.Errorf("limit=1 该只回 1 条，回了 %d", all.Returned)
	}
	if all.Count < all.Returned {
		t.Errorf("count=%d 比 returned=%d 还小", all.Count, all.Returned)
	}
	if all.Truncated != (all.Count > all.Returned) {
		t.Errorf("truncated=%v 与 count=%d returned=%d 对不上", all.Truncated, all.Count, all.Returned)
	}
	for _, x := range all.Ops {
		if x.State != "pending" && x.State != "ok" && x.State != "error" {
			t.Errorf("state=%q 不在三态里（pending/ok/error）", x.State)
		}
		if x.At == "" {
			t.Errorf("每条都该有时间戳（seq=%d）", x.Seq)
		}
	}
	t.Logf("日志：daemonStartedAt=%s count=%d，最新的 op=%s(%s)",
		all.DaemonStartedAt, all.Count, all.Ops[0].Op, all.Ops[0].State)
}

func opsFor(t *testing.T, s *stdioSession, op string) listOpsReply {
	t.Helper()
	r, ok := ask[listOpsReply](t, s, "list_ops "+op, "list_ops", map[string]any{"op": op})
	if !ok {
		t.Fatalf("list_ops op=%s 失败", op)
	}
	return r
}

func opsAll(t *testing.T, s *stdioSession, limit int) listOpsReply {
	t.Helper()
	args := map[string]any{}
	if limit > 0 {
		args["limit"] = limit
	}
	r, ok := ask[listOpsReply](t, s, "list_ops all", "list_ops", args)
	if !ok {
		t.Fatalf("list_ops 失败")
	}
	return r
}
