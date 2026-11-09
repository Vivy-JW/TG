package debug

// T2.2 表征测试：把**今天**的行为逐条钉住，防后面误伤（H2 的判据就在这些断言里）。
//
// 这些测试的对象全部是**即将被改掉**的东西 —— 它们的价值不是"证明今天对"，
// 而是"改的时候知道自己改了什么"：
//
//   - `Manager.Current()`（manager.go:434）取会话池里的**任意一条**（今天池里只可能有一条），
//     契约 C7.1 要把服务端那五处直呼点（B11）全删掉；
//   - `prepareSession`（manager.go:242）的"取唯一 live，否则 Close 旧的"（A–E 表里的第二处实例），
//     WP6 才放开多会话；
//   - `Session.sameTarget`（session.go:539）的复用判据（A–E 表里的 C），WP3/WP4 要把它搬进目标层。
//
// `sameTarget` 的基础用例（host/zone/nil/同机多环境）已经在 `session_reuse_test.go` 里钉过，
// 本文件只补它**还没被钉住**的那两条语义：连接四元组必须全同、名算不出时的取向。
//
// 断言一律写成"今天是这样 + 改了要改这里"，不写成"这样是对的"。

import (
	"errors"
	"strings"
	"testing"

	"tt/internal/target"
)

// sameTarget 的两条**还没被钉住**的语义。
//
// 契约依据：`tt-refactor-plan.md` 开头 A–E 表的 **C** 一行 —— 它是"判错后果最隐蔽"的一处
// （同机多环境时上错号，按企业编号取数的查询全返回 0 行，而 0 行会被读成"这条数据不存在"，
// **结论完全反向**）。原文的判据写在 session.go:526-538 的注释里，这里把它变成断言。
func TestSameTargetSemanticsArePinned(t *testing.T) {
	t.Run("连接四元组必须全同（host/port/user/zone）", func(t *testing.T) {
		// 同名但换了端口 / 用户 → 仍是"换目标"。设置页改了一个环境就属于这一类。
		base := cfgFor("h1", "36")
		s := &Session{ID: "s1", cfg: base, state: StateIdle, envName: base.EnvName()}

		same := cfgFor("h1", "36")
		same.envName = base.EnvName()
		if !s.sameTarget(same) {
			t.Fatal("同名同连接该判为同一目标（这是复用宿主的前提）")
		}

		for _, mut := range []struct {
			name string
			f    func(c *Config)
		}{
			{"换端口", func(c *Config) { c.SSH.Port = 2222 }},
			{"换用户", func(c *Config) { c.SSH.User = "other" }},
			{"换主机", func(c *Config) { c.SSH.Host = "h2" }},
			{"换区域", func(c *Config) { c.Zone = "35" }},
		} {
			c := cfgFor("h1", "36")
			c.envName = base.EnvName() // 环境名保持一样，只动连接
			mut.f(c)
			if s.sameTarget(c) {
				t.Errorf("%s 之后仍被判成同一目标 —— 会拿旧连接继续跑（session.go:516-524 的 sameConn 是前置条件）。\n"+
					"    今天的行为是：连接四元组任一不同 → 换目标。改这条要同时改这里。", mut.name)
			}
		}
	})

	t.Run("环境名算不出时按「允许复用」取向", func(t *testing.T) {
		// 依据在 session.go:536-538：**连接四元组为空 → 不敢复用（换）**；
		// **环境名算不出 → 允许复用**。这两句方向相反，所以必须各钉一条：
		// 混起来读就会把「防御」读成「放行」。
		//
		// 判据是 `s.EnvNameNow()` 与 `cfg.EnvName()` 两个值，两支各自的分界必须分开钉：
		//   - **连接四元组为空** → 不敢复用（换）；
		//   - **一侧环境名算不出** → 那一侧走 "算不出就允许复用" 的**防御分支**（session.go:536-538）。
		// 第二支今天**可达**，而且只在一侧空时可达 —— 见下。
		conn := cfgFor("h1", "36")

		// (a) 会话侧空、配置侧算得出 → `a == ""` → **允许复用**（防御分支）。
		// 读起来反直觉：空的那一侧反而让判据放宽了。所以必须钉住。
		sEmptyName := &Session{ID: "s1", cfg: conn, state: StateIdle, envName: ""}
		if !sEmptyName.sameTarget(conn) {
			t.Errorf("会话 envName 为空、配置算出 %q 时今天是**允许复用**（session.go:547-549 的防御分支）。\n"+
				"    实得 false —— 说明判据变了：这一支放宽的理由是「宁可多登录一次，也不上错号」，\n"+
				"    收严之前先想清楚算不出名字的会话是哪种（session.go:526-538）。", conn.EnvName())
		}
		// (b) 两侧都算得出且相等 → 复用。
		sNamed := &Session{ID: "s2", cfg: conn, state: StateIdle, envName: conn.EnvName()}
		if !sNamed.sameTarget(conn) {
			t.Errorf("两侧环境名都是 %q 时该判同一目标；实得 false", conn.EnvName())
		}
		// (c) 两侧都不相同 → 换。
		sOther := &Session{ID: "s3", cfg: conn, state: StateIdle, envName: "测试1012"}
		if sOther.sameTarget(conn) {
			t.Errorf("会话在 测试1012、目标是 %q 时该判「换目标」；实得 true", conn.EnvName())
		}
		// (d) `Config.EnvName()` 的兜底是 `host + "-" + Zone`，**永远给不出空串**
		//     （host/zone 都空时给 "-"）。这条本身也是今天的语义，一并钉住：
		//     所以"两侧都算不出"这种组合今天到达不了，防御分支只在**会话侧**为空时走。
		noHost := &Config{}
		if got := noHost.EnvName(); got != "-" {
			t.Errorf("零值 Config 的 EnvName() 今天 = %q（兜底 host+`-`+zone），实得 %q。\n"+
				"    这条值不值得知道：它意味着「两侧都算不出」的组合今天到达不了，"+
				"    防御分支只在会话侧为空时走（见 (a)）。", "-", got)
		}
	})

	t.Run("nil 配置不判同一目标", func(t *testing.T) {
		s := &Session{ID: "s1", cfg: cfgFor("h1", "36"), state: StateIdle}
		if s.sameTarget(nil) {
			t.Error("nil 配置该判 false（session.go:540-542）—— 否则 nil 解引用会顺着复用路径漏下去")
		}
	})
}

// TestManagerCurrentPicksAnyLiveSession 钉住 `Manager.Current()` 今天的语义。
//
// 契约依据：B11（服务端**恰好五处**直呼它）+ C7.1-1…C7.1-5（五处全部要改）。
// 这一条是那五处的**共同地基**：它今天取的是"会话池里任意一条"，
// 而且**不看状态**（`StateExit` 的残留也照取，扫除是 `prepareSession` 干的事）。
//
// 为什么值得先钉住：多会话之后"任意一条"会变成"随机一条" ——
// map 遍历顺序在 Go 里是随机的，于是同一个请求可能落到 A 会话也可能落到 B 会话。
// 那正是计划开头那句"不会出现命令跑在另一条会话上却看起来成功"要根除的东西。
func TestManagerCurrentPicksAnyLiveSession(t *testing.T) {
	t.Run("池空返回 nil", func(t *testing.T) {
		m := NewManager(&Config{})
		if cur := m.Current(); cur != nil {
			t.Fatalf("空池该返回 nil（api.go 的 sessOfWriteGlobal 靠它判「没有东西可保护」），实得 %v", cur.ID)
		}
	})

	t.Run("一条时返回那一条", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		s := fakeIdleSession("s1", "h1", "36")
		m.mu.Lock()
		m.sessions["s1"] = s
		m.mu.Unlock()
		if cur := m.Current(); cur != s {
			t.Fatalf("池里只有一条时必须返回它，实得 %v", cur)
		}
	})

	t.Run("StateExit 的残留照样被返回（它不问状态）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		s := fakeIdleSession("s1", "h1", "36")
		s.mu.Lock()
		s.state = StateExit
		s.mu.Unlock()
		m.mu.Lock()
		m.sessions["s1"] = s
		m.mu.Unlock()
		if cur := m.Current(); cur == nil {
			t.Error("Current() 今天不扫 StateExit 的残留（扫除在 prepareSession 里）—— 返回 nil 会改变 sessOfWriteGlobal 的放行判定。\n" +
				"    改这条之前先看 C7.1-1：那处闸门今天靠 cur == nil || cur.Bare() 放行。")
		}
	})

	t.Run("多条时返回其中任意一条（这就是要被改掉的那条）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		a := fakeIdleSession("sA", "h1", "36")
		b := fakeIdleSession("sB", "h1", "36")
		m.mu.Lock()
		m.sessions["sA"] = a
		m.sessions["sB"] = b
		m.mu.Unlock()

		seen := map[string]bool{}
		for i := 0; i < 200; i++ {
			cur := m.Current()
			if cur == nil {
				t.Fatal("池里有两条时不该返回 nil")
			}
			seen[cur.ID] = true
		}
		// 注意：这条**不断言**"必须两条都出现过"。Go 的 map 遍历顺序对 2 元素而言
		// 常常稳定，把它写死会变成一条时红时绿的测试。它断言的是"取到的一定是池里的某一条"，
		// 并且把"哪一条不确定"这件事**写在注释与失败信息里**，让人知道风险在哪儿。
		for id := range seen {
			if id != "sA" && id != "sB" {
				t.Errorf("Current() 返回了池外的会话 %q", id)
			}
		}
		if len(seen) > 1 {
			t.Logf("注意：本机这一轮里 Current() 先后返回过 %d 条不同的会话 —— "+
				"多会话下「任意一条」就是这个意思，服务端五处直呼点必须按 C7.1 改成「按目标判」。", len(seen))
		}
	})
}

// TestPrepareSessionSingleSessionAssumption 钉住 `prepareSession`（manager.go:242）
// 的"取唯一 live，否则 Close 旧的"。
//
// 契约依据：`tt-refactor-plan.md` 把它与 `rec_session.go:82-88` 并列成**同一个毛病的第二处实例**
// （"取唯一 live"就是计划开头 A–E 表里那一类"假设只有一条会话"）；WP6 才放开多会话。
//
// 能测的三条（都在不碰 pty/conn 的前提下）：
//
//  1. 池空 → **尝试新建**（要拨 SSH）；
//  2. 池里有该目标的 live 会话：
//     - 空闲 → 复用它（这一条 `session_reuse_test.go` 已钉过，这里只做"与 Current() 一致"的交叉判据）；
//     - 非空闲 → 报"已存在调试会话"，**不静默再开一条**；
//  3. 池里那条 live 与目标不同 → 今天会 `Close()` 旧的再新建（**测不了**，见文件头 T2.4 的第 4 条）。
func TestPrepareSessionSingleSessionAssumption(t *testing.T) {
	t.Run("池空时它要新建（会真的去拨 SSH）", func(t *testing.T) {
		// 用一个 127.0.0.1 上必然没人听的端口：拨号当场失败，不会挂住。
		cfg := cfgFor("127.0.0.1", "36")
		cfg.SSH.Port = 1
		m := NewManager(cfg)

		sess, err := m.prepareSession(cfg, "m1", "p1", "", "", "", "", "")
		if err == nil {
			t.Fatalf("池空 + 拨不通时该报错；返回了 %v —— 说明它没走「新建」这条路径", sess)
		}
		if !strings.Contains(err.Error(), "SSH 连接失败") {
			t.Errorf("池空时 prepareSession 应当走到**新建会话**（NewSession → host.Dial），"+
				"于是报的是 SSH 连接失败；实际报的是 %q。\n"+
				"    怎么办：如果它开始返回别的错，说明入口逻辑变了（比如不再新建）；"+
				"    如果它开始静默返回 nil，那是更坏的变更。", err.Error())
		}
		if got := m.Current(); got != nil {
			t.Errorf("新建失败后池里不该留下会话，实得 %v", got.ID)
		}
	})

	t.Run("同目标空闲 → 复用（与 Current() 取到的同一条）", func(t *testing.T) {
		cfg := cfgFor("h1", "36")
		m := NewManager(cfg)
		live := fakeIdleSession("s1", "h1", "36")
		m.mu.Lock()
		m.sessions["s1"] = live
		m.mu.Unlock()

		// 交叉判据：Current() 取的那一条，正是 prepareSession 会复用的那一条。
		// 今天只有一条会话，"取任意一条"与"取唯一的 live"是同一件事 ——
		// 多会话之后它们就会分叉，这两处必须一起改。
		if m.Current() != live {
			t.Fatal("前置不成立：池里只有一条时 Current() 该返回它")
		}
		got, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", "")
		if err != nil {
			t.Fatalf("同目标空闲该复用，却报错：%v", err)
		}
		if got != live {
			t.Fatalf("该复用已登记的宿主会话，却新建了 %v", got.ID)
		}
	})

	t.Run("同目标非空闲 → 报错，不静默再开一条", func(t *testing.T) {
		for _, st := range []State{StateStopped, StateRunning, StateLoading} {
			cfg := cfgFor("h1", "36")
			m := NewManager(cfg)
			live := fakeIdleSession("s1", "h1", "36")
			live.mu.Lock()
			live.state = st
			live.mu.Unlock()
			m.mu.Lock()
			m.sessions["s1"] = live
			m.mu.Unlock()

			_, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", "")
			if err == nil {
				t.Errorf("状态 %s 时该报「已存在调试会话…请先结束当前调试」，却放行了 —— "+
					"那就是「静默再开一条」（Q23 明写不许）", st)
				continue
			}
			if !strings.Contains(err.Error(), "已存在调试会话") {
				t.Errorf("状态 %s 时的报错文案 = %q，该是「已存在调试会话(%s 运行中/已停站),请先结束当前调试」的形状",
					st, err.Error(), live.Prog)
			}
		}
	})

	t.Run("多会话时不再只认一条（WP6 改写：本条原来钉的是「取唯一 live」）", func(t *testing.T) {
		// **改写说明（WP6，2026-10-16）**：这条子用例原来钉的是"池里有 >=2 条 live 时
		// `prepareSession` 必从复用/忙碌两支里出一支"—— 那正是**要被改掉**的单例假设
		// （文件头写着"WP6 才放开多会话"）。
		//
		// WP6 把它改成本批要的行为：同环境两条候选 ⇒ **F1 的歧义**（`SESSION_AMBIGUOUS`
		// + 退 3 + 候选九键），**绝不替调用方挑一条**（C8.1 / Q2 / 契约 1. 的 §1）。
		// 逐条的判据（0 / 1 / ≥2 三条分支、单会话放行 / 多会话拒绝）在
		// `wp6_multisession_test.go` 里，这里只留"入口不再是随机挑一条"这一条。
		cfg := cfgFor("h1", "36")
		m := NewManager(cfg)
		a := fakeIdleSession("sA", "h1", "36")
		b := fakeIdleSession("sB", "h1", "36")
		m.mu.Lock()
		m.sessions["sA"] = a
		m.sessions["sB"] = b
		m.mu.Unlock()

		got, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", "")
		if err == nil {
			t.Fatalf("同环境两条候选时该报歧义（C8.1 / Q2），却放行并落到 %v —— "+
				"map 遍历顺序随机，那就是「挑一条」", got.ID)
		}
		var f *target.Failure
		if !errors.As(err, &f) || f.Code != target.CodeAmbiguous {
			t.Fatalf("该报 %s（F1），实得 %T: %v", target.CodeAmbiguous, err, err)
		}
		if len(f.Candidates) != 2 {
			t.Errorf("F1 必须带候选清单（C5.11：那五种码下必须出现），实得 %d 条", len(f.Candidates))
		}
		// 一条都不许动：报错就是报错，不许顺手收掉谁。
		if n := len(m.Snapshot()); n != 2 {
			t.Errorf("报歧义时不许改动会话池，实得 %d 条", n)
		}
	})
}
