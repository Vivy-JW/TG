package debug

// WP4c：把会话身份的五个新键（`ref` / `display` / `name` / `n` / `epoch`）落成**服务端事实**。
//
// 契约条款逐条对应（`.agents/plan/tt-refactor-contract.md`）：
//
//	| 这个文件测什么 | 条款 / 决策 |
//	|---|---|
//	| 序号按**环境**自增、取最小未占用、企业编号不参与 | C2.2 / 2.1.3 / Q16 / Q24 |
//	| `ref` / `n` 生命周期内恒定（`topent` 改了也不动） | **C2.0**（本批最要紧的一条）/ C5.4 |
//	| 别名随会话消失即释放（含异常结束） | C3.7b / **Q19** |
//	| 别名的查重不区分大小写、撞车报 F18 | C2.5 / F18 |
//	| 世代号全进程唯一（换代才有意义） | C5.3 / F15 |
//
// 端到端那几条（HTTP 信封、Q23 的三种结局、F15 的失效判定）在
// `contract_debug_test.go`；这里放的是"服务端怎么算"的判据本身。

import (
	"testing"

	"tt/internal/host"
)

// TestIdentitySeqIsAllocatedPerEnvironment 钉住序号的**作用域**（C2.2 / 2.1.3 / Q24）：
// 「同一环境下的第 N 条，从 1 起，取尚未被占用的最小正整数；**企业编号不参与**」。
//
// 为什么必须按环境（Q24 的理由）：只有"环境内全局自增"能让 `环境#序号` 成为**唯一**地址。
// 按（环境, 企业编号）自增会让企业 99 的 `#1` 与企业 88 的 `#1` 都叫 `恒烁正式区#1` ——
// 同一个地址指向两条会话，于是要么把可变的 `ent` 塞回地址（破 C2.0）、要么地址退回无意义编码。
func TestIdentitySeqIsAllocatedPerEnvironment(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	register := func(id, env, ent string) SessionBrief {
		t.Helper()
		cfg := cfgFor("h1", "36")
		if env != "" {
			cfg = envCfg(env)
		}
		cfg.Topent = hostEnt(ent)
		s := idleOn(id, cfg)
		m.mu.Lock()
		if err := m.assignIdentityLocked(s, ""); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记 %s 失败：%v", id, err)
		}
		m.sessions[id] = s
		m.mu.Unlock()
		for _, b := range m.Snapshot() {
			if b.ID == id {
				return b
			}
		}
		t.Fatalf("登记后快照里找不到 %s", id)
		return SessionBrief{}
	}

	// ① 同一环境下从 1 起、依次自增。
	a1 := register("s-a1", "恒烁正式区", "99")
	a2 := register("s-a2", "恒烁正式区", "88") // **换了企业编号**：序号照旧往下走（Q24 的关键一条）
	if a1.N != 1 || a2.N != 2 {
		t.Fatalf("同一环境下序号该是 1、2（企业编号不参与），实得 %d、%d", a1.N, a2.N)
	}
	if a1.Ref != "恒烁正式区#1" || a2.Ref != "恒烁正式区#2" {
		t.Fatalf("ref 该是 `环境#序号`，实得 %q、%q", a1.Ref, a2.Ref)
	}
	// ② 换一个环境**重新从 1 起**（不是全局计数器）。
	b1 := register("s-b1", "测试1012", "99")
	if b1.N != 1 || b1.Ref != "测试1012#1" {
		t.Fatalf("换环境该重新从 1 起（序号的作用域是环境），实得 n=%d ref=%q", b1.N, b1.Ref)
	}
	// ③ 会话消失后序号释放：取"尚未被占用的最小正整数"（不是"最大 +1"）。
	m.Remove("s-a1")
	a3 := register("s-a3", "恒烁正式区", "99")
	if a3.N != 1 {
		t.Fatalf("空出来的 %d 该被最小的新会话捡回去（C2.2 的\"尚未被占用的最小正整数\"），实得 %d", a1.N, a3.N)
	}
	if a3.Ref != "恒烁正式区#1" {
		t.Fatalf("拿回 #1 之后 ref 该是 恒烁正式区#1，实得 %q", a3.Ref)
	}
}

// TestIdentityRefAndSeqSurviveTopentChange 是**本批最要紧的一条**：C2.0 不变量。
//
// 「一条 SSH 会话的规范 id 在它的生命周期内恒定不变」—— 而 `tt debug topent 88`
// 会改掉这条会话当前生效的企业编号（`SetTopent` → `topentOverride`）。判据：
// **只有 `ent` 字段变，`ref` 与 `n`（以及 `display` 的环境段/序号段）一个字符都不许变**。
//
// 为什么它是最要紧的：`ref` 是本次重构新立的地址。它要是在生命周期里会漂，
// 那么"照着上一次的地址再发一枪"就会落到别的地方 —— 而那正是这次要根除的那类错
// （见计划开头那句：不会出现"命令跑在另一条会话上却看起来成功"）。
func TestIdentityRefAndSeqSurviveTopentChange(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	live := idleOn("s1", cfgFor("h1", "36"))
	live.Prog = "apmt520"
	live.cfg.Topent = "99" // 起会话时给的 --ent
	m.mu.Lock()
	if err := m.assignIdentityLocked(live, "汇率单"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"] = live
	m.mu.Unlock()

	before := m.Snapshot()[0]
	if before.Ent != "99" {
		t.Fatalf("前置不成立：`ent` 该是启动时给的企业编号 99，实得 %q", before.Ent)
	}
	// `tt debug topent 88` 的状态那一半（另一半是把值下发到宿主 shell，要 pty）。
	live.setTopentOverride("88")
	after := m.Snapshot()[0]

	if after.Ref != before.Ref || after.N != before.N {
		t.Errorf("C2.0：改企业编号之后地址不许变 —— ref %q→%q，n %d→%d",
			before.Ref, after.Ref, before.N, after.N)
	}
	if after.Display != before.Display {
		t.Errorf("display 的**环境段与序号段**也不该变（变的是 ent，不是作业）—— %q→%q",
			before.Display, after.Display)
	}
	if after.Name != before.Name {
		t.Errorf("别名生命周期内不变（C2.5b）—— %q→%q", before.Name, after.Name)
	}
	if after.Epoch != before.Epoch {
		t.Errorf("改 ent **不是**换代（F15 的三处换代见 nextSessionEpoch）—— %q→%q", before.Epoch, after.Epoch)
	}
	// 变的只有那个快照字段：这正是 C2.5c 说的"ent 是快照，不是地址"。
	if after.Ent != "88" {
		t.Errorf("`ent` 该跟着 `tt debug topent 88` 变成 88（C5.3 / C2.5c），实得 %q", after.Ent)
	}
}

// TestIdentityAliasReleasedWhenSessionGone 钉住 Q19 / C3.7b：**别名随会话消失即释放**。
//
// 判据锚在"**会话池里还有没有它**"，与"人敲没敲 quit"无关 —— 所以三种结束方式走同一条：
//
//	① 正常结束（quit / close：会话从池里被移除）；
//	② **异常结束**（SSH 掉线、进程被杀、断电：`forceExit` 把状态置成 `StateExit`，
//	   会话随后被清扫出池）；
//	③ 服务重启（整池没了 = 新进程里一条会话都没有）。
//
// 为什么这条值得单测：反过来（别名不随会话释放）会产出一类**极难查的故障** ——
// "会话已死但别名仍占着，重起同名被拒"，报的还是"别名已被 <ref> 占用"，
// 而 `<ref>` 指向一条早就没了的会话（C3.7b 原话）。
func TestIdentityAliasReleasedWhenSessionGone(t *testing.T) {
	const alias = "复现-1"

	newNamed := func(t *testing.T, m *Manager, id string) *Session {
		t.Helper()
		s := idleOn(id, cfgFor("h1", "36"))
		m.mu.Lock()
		if err := m.assignIdentityLocked(s, alias); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记 %s 失败：%v", id, err)
		}
		m.sessions[id] = s
		m.mu.Unlock()
		return s
	}

	t.Run("① 正常结束（从池里移除）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		newNamed(t, m, "s1")
		m.Remove("s1")
		if err := m.checkNameFree(alias); err != nil {
			t.Fatalf("会话已从池里移除，别名该当场可复用（Q19），实得：%v", err)
		}
	})

	t.Run("② 异常结束（掉线 → StateExit → 清扫出池）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		s := newNamed(t, m, "s1")
		// 掉线那条路做的事（`forceExit`，见 session.go）：状态置成 StateExit；
		// 这里不调 ForceExit 本身 —— 手工造的会话没有 pty，它会打到 nil。
		s.mu.Lock()
		s.state = StateExit
		s.mu.Unlock()
		// 判据一：还没清扫时，别名就已经不算"被占"（占用者只认活着的会话）。
		if err := m.checkNameFree(alias); err != nil {
			t.Fatalf("已断开的会话不许再占着别名（Q19：判据是池子里还有没有它），实得：%v", err)
		}
		// 判据二：清扫之后，`tt debug status` 里不再有它的 ref —— 别名必须可复用。
		m.SweepExited()
		if got := len(m.Snapshot()); got != 0 {
			t.Fatalf("清扫后池里该一条都没有，实得 %d 条", got)
		}
		fresh := idleOn("s2", cfgFor("h1", "36"))
		m.mu.Lock()
		err := m.assignIdentityLocked(fresh, alias)
		m.mu.Unlock()
		if err != nil {
			t.Fatalf("同一个别名该能直接给新会话（Q19 的可验证判据），实得：%v", err)
		}
	})

	t.Run("③ 服务重启（新进程里整池为空）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		if err := m.checkNameFree(alias); err != nil {
			t.Fatalf("新进程里没有任何会话，别名一定是空的，实得：%v", err)
		}
	})
}

// TestIdentityNameCollisionIsCaseInsensitive 钉住 C2.5 的查重口径：
// **两个只差大小写的同名别名算撞车**（F18），而查找按原样比（别名参与寻址）。
//
// 中文名与拼音名是两个名字（`汇率单` 与 `HuiLvDan` 不撞车），这一点也钉下来 ——
// 它证明查重用的是"不区分大小写的**同一个名字**"，不是"看起来像"。
func TestIdentityNameCollisionIsCaseInsensitive(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	first := idleOn("s1", cfgFor("h1", "36"))
	m.mu.Lock()
	if err := m.assignIdentityLocked(first, "HuiLvDan"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记失败：%v", err)
	}
	m.sessions["s1"] = first
	m.mu.Unlock()

	if err := m.checkNameFree("huilvdan"); err == nil {
		t.Errorf("只差大小写的别名该算撞车（C2.5 / F18），却放行了")
	}
	if err := m.checkNameFree("汇率单"); err != nil {
		t.Errorf("`汇率单` 与 `HuiLvDan` 是两个名字，不该撞车（C2.5），实得：%v", err)
	}
	// 撞车时的错误得带上占用者（F18 的候选清单要靠它）。
	err := m.checkNameFree("HUILVDAN")
	if err == nil {
		t.Fatalf("大小写不同的同名该撞车")
	}
	var taken *ErrNameTaken
	if !asNameTaken(err, &taken) {
		t.Fatalf("该报 *ErrNameTaken（F18），实得 %T", err)
	}
	if taken.Occupant != "s1" {
		t.Errorf("占用者该是 s1，实得 %q", taken.Occupant)
	}
}

// TestIdentityEpochIsGloballyUnique 钉住世代号那半边"**全进程唯一**"的性质。
//
// 为什么唯一性是判据的一部分（而不是实现细节）：F15 的判据是"手上的 epoch 与服务端当前值
// **不相等**就是上一代"。若世代号会重复（比如每条会话各从 1 数），那么"旧会话被收掉、
// 新会话拿到同一个 ref"之后，上一代的句柄会**凑巧**等于新会话的世代号 —— 于是
// "旧句柄当场失效"这件事就静默失效了，而没有任何东西会响。
func TestIdentityEpochIsGloballyUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		e := nextSessionEpoch()
		if e == "" {
			t.Fatalf("第 %d 次发出的世代号是空的", i)
		}
		if seen[e] {
			t.Fatalf("世代号重复了：%q（第 %d 次）—— F15 的失效判定依赖它全进程唯一", e, i)
		}
		seen[e] = true
	}
}

// asNameTaken 是 errors.As 的一层薄封装（这里不 import errors，免得与别的测试文件的
// import 块打架：直接用类型断言，因为我们就是那个包）。
func asNameTaken(err error, out **ErrNameTaken) bool {
	e, ok := err.(*ErrNameTaken)
	if ok {
		*out = e
	}
	return ok
}

// TestDecideHostMatrix 钉住"池里那条既有会话怎么处置"的全部三种分支（Q23 / C2.5b / H2）。
//
// 为什么值得单独测：三种处置里只有"复用"这一条能靠单测走到（另外两条要真拨 SSH 或真收口），
// 而"什么时候敢收掉别人的会话"恰恰是最不该靠碰运气验的一条 —— 收错了的后果是
// **把用户正在看的调试会话弄没了**，而输出看起来一切正常。
//
// 判据是纯函数（`decideHost`），所以这里逐格铺开：换目标 / 在跑 / 已停站 / 别名对不上，
// 以及 `wantName` 为空时**必须逐字等于今天的行为**（H2 的落点）。
func TestDecideHostMatrix(t *testing.T) {
	base := cfgFor("h1", "36")
	cases := []struct {
		name      string
		prevEnv   string // 既有会话的环境（空 = 与这次同一个；非空 = 换环境，用 envCfg 造）
		prevState State  // 既有会话的状态
		prevName  string // 既有会话的别名（空 = 无名）
		wantName  string // 这次请求要的别名（空 = 没点名）
		want      hostDecision
	}{
		{"同目标 空闲 无名 没点名 → 复用（今天的行为）", "", StateIdle, "", "", hostReuse},
		{"同目标 空闲 无名 点名 X → 复用这件宿主并给它起名", "", StateIdle, "", "汇率单", hostReuse},
		{"同目标 空闲 已叫 X 点名 X → 复用（Q23 的幂等键）", "", StateIdle, "汇率单", "汇率单", hostReuse},
		{"同目标 空闲 已叫 X 点名 Y → 收口重建（别名不能改，C2.5b）", "", StateIdle, "汇率单", "另一个名字", hostCollect},
		{"同目标 已停站 没点名 → 报错（今天的原话）", "", StateStopped, "", "", hostBusy},
		{"同目标 运行中 没点名 → 报错", "", StateRunning, "", "", hostBusy},
		{"同目标 已停站 已叫 X 点名 Y → 仍是报错（不许为抢名字收掉在干活的会话）", "", StateStopped, "汇率单", "另一个名字", hostBusy},
		{"换环境 空闲 → 收口重建（今天就不看状态）", "测试1012", StateIdle, "", "", hostCollect},
		{"换环境 已停站 → 收口重建（今天的行为，一字不改）", "测试1012", StateStopped, "", "", hostCollect},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			prevCfg := base
			if c.prevEnv != "" {
				prevCfg = envCfg(c.prevEnv)
			}
			prev := idleOn("s1", prevCfg)
			prev.Prog = "apmt520"
			prev.mu.Lock()
			prev.state = c.prevState
			prev.name = c.prevName
			prev.mu.Unlock()

			if got := decideHost(prev, base, c.wantName); got != c.want {
				t.Errorf("decideHost = %v，期望 %v\n"+
					"    （换目标 / 在跑 / 空闲、别名对得上 / 对不上 —— 三条规矩见 decideHost 的注释）",
					got, c.want)
			}
		})
	}
	// nil（池空）也走"收口重建"那一支：没有既有会话可收，直落到新建。
	if got := decideHost(nil, base, "随便"); got != hostCollect {
		t.Errorf("池空时 decideHost = %v，期望 hostCollect（等价于「没有可复用的宿主」）", got)
	}
}

// hostEnt 把企业编号塞进配置字段（`Config.Topent` 是 host.EntValue 类型）。
func hostEnt(v string) host.EntValue { return host.EntValue(v) }
