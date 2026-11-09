package debug

// WP5 的并发缺陷判据（T5.1–T5.5 里 X / Y / Z 三处的**进程内**那一半）。
//
// 为什么这些能在这里测:三处缺陷的病灶都是**进程内共享态** ——
//   - X:清的是全局 `srccache/` 目录,而"已镜像"标记是会话内的;
//   - Y:动态路径被写回共享的 `m.cfg.Runtime`;
//   - Z:会话外读取只有**一条**连接 + 一把全局锁,且 key 不分环境。
//
// 它们都不需要真 SSH:判据是"共享态有没有被改""两处能不能同时进"。
// 真拨 SSH 的那一半（登录、读远端文件）不在这里 —— 见 `internal/debug/README.md`
// 的"故意不覆盖"一节。

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tt/internal/host"
)

// ---------- Y（T5.2）:会话级操作不许改写共享配置 ----------

// 跑一轮 `LaunchWith` 的**共享态代价**:它拿到的 cfg 与 m.cfg 是两份，
// 调用结束后共享配置必须一字未动（缺陷 Y）。
//
// 修前:`resolveJobWith` / `ensureRuntimeEnv` 直接写 `m.cfg.Runtime`
// ⇒ 这里的第一条断言红（第二条更狠:它连调用方给的那一份也改，于是"两条会话
// 互相污染"只需要一次作业解析就成立）。
func TestLaunchDoesNotMutateSharedConfig(t *testing.T) {
	shared := cfgFor("h1", "36") // 没有 db ⇒ resolveJobWith 会走"动态路径"这一步
	m := NewManager(shared)
	req := cfgFor("h1", "36") // 本轮请求克隆出来的那一份
	req.DB = nil

	if _, prog, _, _, err := m.resolveJobWith(req, "asf", "bspt001_wf"); err != nil {
		t.Fatalf("resolveJobWith 不该报错（没有 db 时应静默返回空）: %v", err)
	} else if prog != "" {
		t.Fatalf("没配 db 时解析不出实体程序（这是既有语义），实得 %q", prog)
	}

	if shared.Runtime != nil {
		t.Fatalf("共享配置的 Runtime 被会话级操作改写了（%+v）—— 这就是缺陷 Y:"+
			"另一条会话随后读到的是别人的动态路径", shared.Runtime)
	}
}

// 共享的 `m.cfg` 必须**始终**是 nil Runtime:两条会话各自持自己那一份。
func TestSharedConfigNeverLearnsDynamicPaths(t *testing.T) {
	shared := cfgFor("h1", "36")
	m := NewManager(shared)
	// 会话外读取路径拿的是副本（sourceCfg），不是共享那一份
	if got := m.sourceCfg(); got == shared {
		t.Fatal("会话外读取该拿一份配置副本 —— 直接改共享的 m.cfg 就是缺陷 Y 的入口")
	}
	// `resolveJob`（无 With 的那条）也必须走副本
	if _, _, _, _, err := m.resolveJob("asf", "bspt001_wf"); err != nil {
		t.Fatalf("resolveJob 不该报错: %v", err)
	}
	if shared.Runtime != nil {
		t.Fatalf("resolveJob 把动态路径写回了共享配置（%+v）", shared.Runtime)
	}
	if s := srcConnScopeSeg(shared); s == "" {
		t.Fatal("共享配置仍要能给出环境段（会话外镜像按目标落盘）")
	}
}

// 会话级操作把动态路径写进**调用方自己那一份**配置,共享那一份始终是 nil（缺陷 Y / T5.2）。
//
// 这条钉的是**写入点**:`ensureRuntimeEnvOn(conn, cfg)` 必须写 cfg,不能写 `m.cfg`。
// 修前的实现在这里没有可对应的函数（那正是"写入点是共享字段"的证据），
// 而它落回共享那条路（`m.cfg.Runtime == nil` 时的兜底）就是缺陷 Y:
// 谁先启动，另一条会话读到的就是谁的 TOP/ERP。
func TestRuntimeEnvWritesOnlyToTheCallersConfig(t *testing.T) {
	shared := cfgFor("h1", "36")
	m := NewManager(shared)

	own := cfgFor("h1", "36")
	own.Runtime = &host.RuntimeEnv{TOP: "/u1/t35prd", ERP: "/u1/t35prd/erp"}
	if _, err := m.ensureRuntimeEnvOn(nil, own); err != nil {
		t.Fatalf("已有有效 Runtime 时不该再探（无连接也必须能返回）: %v", err)
	}
	if own.Runtime == nil || own.Runtime.TOP != "/u1/t35prd" {
		t.Fatal("本份配置的有效 Runtime 不该被清掉")
	}
	if shared.Runtime != nil {
		t.Fatalf("共享配置被写入了 Runtime（%+v）—— 另一条会话随后读到的就是这条的路径（缺陷 Y）", shared.Runtime)
	}

	// 第二条会话独立走一遍:两条各自的动态路径互不可见
	own2 := m.sourceCfg()
	own2.Runtime = &host.RuntimeEnv{TOP: "/u1/t35dev", ERP: "/u1/t35dev/erp"}
	if _, err := m.ensureRuntimeEnvOn(nil, own2); err != nil {
		t.Fatalf("第二次也不该再探: %v", err)
	}
	if own.Runtime.TOP != "/u1/t35prd" || own2.Runtime.TOP != "/u1/t35dev" {
		t.Fatalf("两条会话的动态路径互相污染了: %q / %q", own.Runtime.TOP, own2.Runtime.TOP)
	}
	if shared.Runtime != nil {
		t.Fatalf("共享配置始终该是 nil,实得 %+v", shared.Runtime)
	}
}

// 两条会话同时启动:各自的动态路径互不污染（T5.2 的验收）。
//
// 判断方式是"每一份 cfg 只可能拿到自己那一份值" —— 共享字段一旦存在，
// 后启动的那条就会看到前一条的路径（修前:两个 goroutine 都写 m.cfg.Runtime，
// 断言 shared.Runtime 仍是 nil 会红）。
func TestTwoSessionsKeepTheirOwnRuntime(t *testing.T) {
	shared := cfgFor("h1", "36")
	m := NewManager(shared)
	const n = 16
	var bad atomic.Int64
	done := make(chan struct{})
	for i := 0; i < n; i++ {
		go func(i int) {
			defer func() { done <- struct{}{} }()
			own := m.sourceCfg()
			own.Runtime = &host.RuntimeEnv{TOP: "/u1/z" + string(rune('a'+i%26)), ERP: "/u1/erp"}
			if _, err := m.ensureRuntimeEnvOn(nil, own); err != nil {
				bad.Add(1)
				return
			}
			if own.Runtime.TOP != "/u1/z"+string(rune('a'+i%26)) {
				bad.Add(1) // 被别人的显式设置顶掉了
			}
		}(i)
	}
	for i := 0; i < n; i++ {
		<-done
	}
	if bad.Load() != 0 {
		t.Fatalf("有 %d 条会话的动态路径被污染（各自那一份 cfg 之间不该互相看见）", bad.Load())
	}
	if shared.Runtime != nil {
		t.Fatalf("共享配置被写入了 Runtime（%+v）—— 两条会话的动态路径就此互相污染", shared.Runtime)
	}
}

// ---------- Z（T5.3）:连接 key 要分环境,锁要按 key 分 ----------

// 同机同账号、只差环境名/区域的两个目标**不能**共用一条会话外连接（缺陷 Z）。
//
// 修前 `srcConnKey()` = `host:port/user`,上面两个目标算出**同一个 key**:
// 于是换了环境仍在用旧连接，按旧区域解析出的 TOP/ERP 去读源码。
func TestSrcConnKeyIncludesEnvAndZone(t *testing.T) {
	e1 := host.NamedSsh{Name: "恒烁正式区"}
	e1.Host, e1.Port, e1.User, e1.Zone = "h1", 22, "u", "36"
	e2 := host.NamedSsh{Name: "测试1012"}
	e2.Host, e2.Port, e2.User, e2.Zone = "h1", 22, "u", "36"

	base := &Config{SSH: host.SSHConfig{Host: "h1", Port: 22, User: "u"}, Zone: "36",
		SSHs: []host.NamedSsh{e1, e2}}
	base.fillDefaults()
	c1 := base.CloneEnv("恒烁正式区")
	c2 := base.CloneEnv("测试1012")
	if c1 == nil || c2 == nil {
		t.Fatal("两个环境都该能克隆出来")
	}

	k1, k2 := srcConnKey(c1), srcConnKey(c2)
	if k1 == k2 {
		t.Fatalf("同机同账号的两个环境算出了同一个连接 key（%q）—— 换了环境仍在用旧连接（缺陷 Z）", k1)
	}
	if !strings.Contains(k1, "恒烁正式区") || !strings.Contains(k2, "测试1012") {
		t.Fatalf("key 里要看得见环境维度: %q / %q", k1, k2)
	}
	// 区域也要进 key:同一环境换区域解析出的 TOP/ERP 不同
	same := *c1
	same.Zone = "35"
	if srcConnKey(&same) == k1 {
		t.Fatalf("区域没进 key（%q）—— 换了区域仍用旧连接", k1)
	}
	// 目标或账号变了同样要换（既有语义,不许丢）
	other := *c1
	other.SSH.Host = "h2"
	if srcConnKey(&other) == k1 {
		t.Fatal("换主机仍算出同一个 key")
	}
	if srcConnKey(nil) != "" {
		t.Error("nil 配置该得到空 key（调用方不必再判一次）")
	}
}

// 两条会话并发读源码**不阻塞**（T5.3 的验收判据）。
//
// 口径是"按连接排队,不按全局排队":不同目标各借各的连接,所以两个 goroutine
// 必须能**同时**进到借用里。修前的写法是一条全局连接 + 一把全局锁,
// 后进的那一条会卡在锁上（下面的 2s 兜底会响）。
func TestSrcConnDifferentTargetsRunInParallel(t *testing.T) {
	shared := cfgFor("h1", "36")
	m := NewManager(shared)

	// 手工放一个"已借出"的标记:两个不同 key 的锁必须互不相干。
	release1 := m.lockSrcKey("targetA")
	defer release1()
	got := make(chan bool, 1)
	go func() {
		release2 := m.lockSrcKey("targetB")
		defer release2()
		got <- true
	}()
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("两个不同目标的连接锁互相阻塞了 —— 并发读源码会串行（缺陷 Z）")
	}

	// 同一个 key 仍然要排队（一条 SSH 连接不能被两个借用者同时用）
	releaseSame := m.lockSrcKey("targetSame")
	blocked := make(chan bool, 1)
	go func() {
		r := m.lockSrcKey("targetSame")
		defer r()
		blocked <- true
	}()
	select {
	case <-blocked:
		releaseSame()
		t.Fatal("同一个目标的两处借用必须排队（共用一条 SSH 连接与一个 SFTP 客户端）")
	case <-time.After(150 * time.Millisecond):
	}
	releaseSame() // 放行:租约是**释放**才还,不是"超时就还"
	select {
	case <-blocked:
	case <-time.After(2 * time.Second):
		t.Fatal("释放之后同一个 key 的等候者该立刻拿到")
	}
}
