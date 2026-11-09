package tzs

// chain_test.go —— 六请求链，走**真命名管道守护进程**（旧实现：gate-w3.py 的段 A）。
//
// 为什么这一条必须走管道，而语料回归走 --stdio：这条链要验的正是「管道」这一层本身 ——
// 起守护进程、就绪握手、跨请求复用同一个进程、stop 收摊。管道名里混着
// TzsCli.Designer.dll 的 MVID（每次重编都变），所以它是整条线上风险最高的一个假设；
// 语料回归用 --stdio 正是为了把这个假设从 67 个包里摘出去，而这里就是它的归属地。
//
// 六请求链（gate-w3.py 段 A 的形状，逐条对应）：
//
//	open → find_component → get_component → set_layout_attr → validate → save
//
// 加一条**不属于链条本身**的观测：再发一次 find_component 并计时。那是 A8/A9 的要害 ——
// 守护进程的全部意义是「只有第一次调用付那 ~900 ms 的设计器初始化」。如果第二次调用也是
// ~1 s，说明守护进程根本没被复用，整套长驻进程的设计退化成了「一次调用一个进程」，
// 而这种退化在功能上完全看不出来（每条断言都还是绿的）。
//
// 开启方式同 corpus_test.go：TTZS_DEEP=1 + 引擎 + 语料（TTZS_WS 可省，见下）。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tt/internal/testenv"
)

// chainTargetName 是这一条链的靶包。选它是因为它是文档里每个例子都用的那张表单
// （tpl=P，且带一个真实的数据绑定叶子 l_apcasite），gate-w3.py 段 A 用的也是它。
//
// 靶包不换：这条链验的是**管道本身**，不是某一批表单的性质，换来换去只会让「上一次
// 还好好的」变成一种碰运气。
const chainTargetName = "aapp320(c).tzs"

// chainQuery 是 find_component 用的代号。它必须在靶包里存在 —— 找不到时这一条**跳过**
// 并说清原因（旧实现在同一处的做法：报一句然后停掉那一段，而不是拿一个空路径继续往下
// 跑出四条与真因无关的红）。
const chainQuery = "l_apcasite"

func TestSixRequestChain(t *testing.T) {
	env := requireCorpus(t)
	var target string
	for _, p := range env.pkgs {
		if strings.EqualFold(filepath.Base(p), chainTargetName) {
			target = p
		}
	}
	if target == "" {
		t.Skipf("语料里没有 %s —— 这条链要一个真实的数据绑定叶子；换个语料就得同时改 chainQuery", chainTargetName)
	}
	ws := workspaceOf(target)
	if ws == "" {
		// 推不出工作区时才退到 TTZS_WS / config.local.json（照 batch.sh 的回落）。
		// 注意这里退的仍是**调用方给的值**，不是引擎内置的缺省 ——
		// 引擎的缺省是一个真实客户目录（包注释纪律 2）。
		ws = testenv.Workspace()
	}
	if ws == "" {
		t.Skipf("%s 的祖先目录里没有 mta/，且没给 TTZS_WS —— 工作区绝不替你选", filepath.Base(target))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	o := Options{
		Exe:        env.exe,
		InstallDir: testenv.DesignerDir(),
		Workspace:  ws,
		// 状态文件与守护进程日志落在测试自己的临时目录里：默认落点（config.json 旁边）是
		// 用户的东西，一次测试没理由往那儿写。日志丢了也不影响诊断 —— Ensure 的报错文案
		// 本来就会把日志末尾贴出来。
		WorkDir: t.TempDir(),
	}

	// 先确保干净：上一次跑留下的守护进程不该让这一条的「冷启动」变成暖的。
	if err := Stop(ctx, o); err != nil {
		t.Fatalf("清场 Stop 失败: %v", err)
	}
	// 但 Stop 返回时守护进程可能还在退出的路上（它写好评答才退，而 Stop 不等它死）。
	// 不等干净就直接 Ensure 会撞上一个竞态：新起的守护进程发现管道名还被占着就自己退出
	// （Rpc.Daemon 的注释：「管道已被占用，退出」），于是我们可能接到一台正在消失的守护进程
	// 上 —— 而那会把「冷启动」这个前提悄悄换掉，让下面那条复用判据失去意义。
	settle := time.Now().Add(15 * time.Second)
	for time.Now().Before(settle) {
		if info, _ := LookupDaemon(ctx, o); info == nil || !info.Running {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	t0 := time.Now()
	pipe, err := Ensure(ctx, o)
	tCold := time.Since(t0)
	if err != nil {
		t.Fatalf("冷启动 Ensure 失败: %v", err)
	}
	defer func() {
		// 收摊。Stop 幂等且绝不 spawn（让「停止」去启动服务是个笑话）。
		if err := Stop(ctx, o); err != nil {
			t.Errorf("收尾 Stop 失败: %v", err)
		}
	}()
	info1, err := LookupDaemon(ctx, o)
	if err != nil {
		t.Fatalf("LookupDaemon 失败: %v", err)
	}
	if !info1.Running || info1.PID <= 0 {
		t.Fatalf("Ensure 之后该有一个应答的守护进程：%+v", info1)
	}
	if info1.Pipe != pipe {
		t.Errorf("Ensure 返回的管道名与快照不一致：%q vs %q", pipe, info1.Pipe)
	}

	// 每个请求走完整的 Call 路径（连 → 发 → 收）。id 递增：一进一出之下它们不会撞车，
	// 而递增能让「同一帧被读到两次」这种错法当场露出来。
	call := func(id int, fn string, args map[string]any) (*Reply, time.Duration) {
		t.Helper()
		raw, err := json.Marshal(args)
		if err != nil {
			t.Fatalf("%s 的参数不是合法 JSON: %v", fn, err)
		}
		start := time.Now()
		r, err := Call(ctx, DefaultDialer(), pipe, id, fn, raw, DefaultTimeout)
		dt := time.Since(start)
		if err != nil {
			t.Fatalf("%s 走管道失败: %v", fn, err)
		}
		return r, dt
	}

	// ① open
	r, _ := call(100, "open", map[string]any{"path": target})
	opened, ok := replyResult[openReply](t, "open", r, nil)
	if !ok || opened.Handle == "" {
		t.Fatalf("open 没给出句柄: %s", clip(r.Result))
	}
	h := opened.Handle

	// ② find_component（第一次：链里那一次，也是「第一次暖调用」的计时点）
	r, tWarm1 := call(101, "find_component", map[string]any{"handle": h, "query": chainQuery})
	found, ok := replyResult[findReply](t, "find_component", r, nil)
	if !ok {
		return
	}
	if found.MatchCount < 1 || len(found.Matches) == 0 {
		// 靶包里没有这个代号。说清并停在这里，而不是拿一个空路径继续往下跑出一串
		// 与真因无关的红（旧实现在同一处也是这么做的）。
		t.Skipf("%s 里没有 %s（find_component 回 matchCount=%d）—— 这条链要一个已知存在的代号",
			filepath.Base(target), chainQuery, found.MatchCount)
	}
	leaf := found.Matches[0].Path

	// ③ get_component：读布局属性，好知道 posX 现在是多少
	r, _ = call(102, "get_component", map[string]any{"handle": h, "path": leaf})
	comp, ok := replyResult[getReply](t, "get_component", r, nil)
	if !ok {
		return
	}
	posX, ok := comp.Layout["posX"]
	if !ok {
		// 不是所有控件都有 posX（比如某些容器）。这是**包的性质**，不是管道的问题 ——
		// 说清并跳过，而不是拿一个空串去 set_layout_attr 换来一句 E_BAD_PARAM。
		t.Skipf("%s 的 %s 没有 posX 这个布局属性（可写属性：%s）", filepath.Base(target), leaf, clipMapKeys(comp.Layout))
	}
	n, err := strconv.Atoi(strings.TrimSpace(posX))
	if err != nil {
		t.Skipf("%s 的 posX=%q 不是整数，无法「+1」出一个必然不同的值", leaf, posX)
	}

	// ④ set_layout_attr：真写一个布局属性。写 +1 而不是某个固定值，是为了让「写进去的值与
	// 原值不同」这件事不依赖具体表单 —— 同值会被当作 E_NO_OP 成功（§11.24 (a)），
	// 那一次写就变成了静默的空操作，而下面的定点断言在空操作上也照样绿。
	r, _ = call(103, "set_layout_attr", map[string]any{
		"handle": h, "path": leaf, "attr": "posX", "value": strconv.Itoa(n + 1),
	})
	delta, ok := replyResult[deltaRef](t, "set_layout_attr", r, nil)
	if !ok {
		return
	}
	if delta.Old != posX {
		t.Errorf("set_layout_attr 说旧值是 %q，get_component 读到的却是 %q", delta.Old, posX)
	}
	if !delta.Changed {
		t.Errorf("set_layout_attr ok:true 但没有 changed 标记（%q → %q）—— 静默空操作", delta.Old, delta.Value)
	}

	// ⑤ validate
	//
	// 这一帧是**本句柄上的第一次** validate，而 Fns/Validate.cs 把第一次调用本身就当成
	// baseline，于是 newErrors 在这里恒为空。也就是说这一帧证明的是「校验器跑得完、回得出
	// 一个成形的结果」，**不是**「改动没有引入新错误」—— 后者要在一个进程里先量一遍 pristine
	// 才有意义，那是 corpus_test.go 的四个语料用例在做的事（它们为此每包调两次 validate）。
	// 把这件事写在这里，是因为照抄旧关卡最容易漏掉的就是它：旧 gate-w3.py 段 B 的两个 payload
	// 是两个进程，第二个进程里的 validate 同样是一次「首次调用」，所以那句
	// "newErrors must be 0" 从来没有机会失败。
	r, _ = call(104, "validate", map[string]any{"handle": h})
	v, ok := replyResult[validateDelta](t, "validate", r, nil)
	if !ok {
		return
	}
	if len(v.NewErrors) != 0 {
		t.Errorf("validate 回了 %d 个 newErrors：%s", len(v.NewErrors), dumpFindings(v.NewErrors))
	}
	t.Logf("validate: %d ms，这一版模型上校验器报了 %d 条发现", v.ElapsedMs, len(v.After))

	// ⑥ save
	out := scratchPath(target, 0, "chain", "post")
	defer os.Remove(out)
	r, _ = call(105, "save", map[string]any{"handle": h, "out": out})
	saved, ok := replyResult[saveReply](t, "save", r, nil)
	if !ok {
		return
	}
	st, err := os.Stat(out)
	if err != nil || st.Size() == 0 {
		t.Fatalf("save 说写到了 %s，但那里没有东西（%v）", saved.Out, err)
	}

	// A7：管道产出的文件还得是一个定点（基线相对 —— 见 corpus_test.go 的 requireFixedPoint）。
	base := runRoundTrip(t, env, ws, target, "chain pristine")
	if base == nil {
		return
	}
	if !requireFixedPoint(t, "chain pristine", base, nil) {
		return
	}
	if got := runRoundTrip(t, env, ws, out, "chain post"); got != nil {
		requireFixedPoint(t, "chain post", got, base)
	}

	// ⑦（不是链的一部分）再暖一次：这就是「守护进程被复用了」的证据本身。
	r, _ = call(106, "find_component", map[string]any{"handle": h, "query": chainQuery})
	if _, ok := replyResult[findReply](t, "find_component(warm)", r, nil); !ok {
		return
	}
	tWarm := tWarm1
	// 直接证据：还是同一台守护进程（pid 与管道名都没变）。
	info2, err := LookupDaemon(ctx, o)
	if err != nil {
		t.Fatalf("第二次 LookupDaemon 失败: %v", err)
	}
	if !info2.Running {
		t.Fatalf("链条跑完之后守护进程不该消失：%+v", info2)
	}
	if info2.PID != info1.PID {
		t.Errorf("守护进程换了一台：pid %d → %d（每次调用都要重 Boot 一次了）", info1.PID, info2.PID)
	}
	if info2.Pipe != pipe {
		t.Errorf("管道名变了：%q → %q", pipe, info2.Pipe)
	}

	// 间接证据：第二次调用必须**显著**快于冷启动。冷启动要 spawn + Boot（~1 s 量级），
	// 暖调用只是连一次管道 + 跑一次 find_component（几十 ms 量级），所以 2 倍这个界
	// 宽到不会因为机器负载而抖 —— 它要抓的是「暖调用也是 ~1 s」那种数量级的退化。
	// 旧关卡在同一处用的界是 0.6 s + 1.5 倍；这里换成纯比值，免得把一台慢机器判成回归。
	if !(tCold > 2*tWarm) {
		t.Errorf("暖调用没有明显快于冷启动：冷 %.2fs、暖 %.2fs（第一次暖 %.2fs）—— "+
			"守护进程多半没有被复用，整套长驻进程的设计退化成了一次调用一个进程",
			tCold.Seconds(), tWarm.Seconds(), tWarm1.Seconds())
	}
	t.Logf("链：冷启动 %.2fs（含 spawn+Boot）→ 第一次暖调用 %.2fs → 复用的守护进程 pid=%d，管道 %s",
		tCold.Seconds(), tWarm1.Seconds(), info2.PID, pipe)
}

// getReply 是 get_component 的返回（只列我们读的字段）。
//
// Layout 用 map[string]string 而不是逐个字段：那是**活的**模型属性表（几十上百个键，按元素
// 类型而变），在 Go 侧抄一份字段清单就等于把「有哪些布局属性」也变成会漂的第二实现。
type getReply struct {
	Tag    string            `json:"tag"`
	Name   string            `json:"name"`
	Path   string            `json:"path"`
	Table  string            `json:"table"`
	Column string            `json:"column"`
	Layout map[string]string `json:"layout"`
}

func clipMapKeys(m map[string]string) string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) > 12 {
		keys = keys[:12]
	}
	return strings.Join(keys, ",") + "…"
}
