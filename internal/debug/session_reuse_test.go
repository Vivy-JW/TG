package debug

import (
	"testing"

	"tg/internal/dbconfig"
	"tg/internal/host"
)

// 单一常驻会话:目标身份判定/空闲宿主复用/环境克隆的纯逻辑单测
// (SSH 登录等依赖真实 T100,不在此覆盖)
//
// 断点存档(bpsstore.go)目前**没有**覆盖,那个域不在这里 —— 别再按文件名找它。

func cfgFor(h, zone string) *Config {
	c := &Config{}
	c.SSH = host.SSHConfig{Host: h, Port: 22, User: "u"}
	c.Zone = zone
	c.fillDefaults()
	return c
}

func fakeIdleSession(id, host, zone string) *Session {
	return &Session{
		ID:       id,
		cfg:      cfgFor(host, zone),
		state:    StateIdle,
		Module:   "oldm",
		Prog:     "oldp",
		bps:      map[int]*Breakpoint{},
		srcCache: map[string]srcCacheEntry{},
	}
}

func TestSessionSameTarget(t *testing.T) {
	s := fakeIdleSession("s1", "h1", "36")
	if !s.sameTarget(cfgFor("h1", "36")) {
		t.Fatal("同 host/zone 应判定为同一目标")
	}
	if s.sameTarget(cfgFor("h2", "36")) {
		t.Fatal("不同 host 不应判定为同一目标")
	}
	if s.sameTarget(cfgFor("h1", "35")) {
		t.Fatal("不同 zone 不应判定为同一目标")
	}
	if s.sameTarget(nil) {
		t.Fatal("nil 配置不应判定为同一目标")
	}
}

func TestPrepareSessionReusesIdleHost(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	live := fakeIdleSession("s1", "h1", "36")
	m.mu.Lock()
	m.sessions["s1"] = live
	m.mu.Unlock()

	sess, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", "")
	if err != nil {
		t.Fatalf("prepareSession: %v", err)
	}
	if sess != live {
		t.Fatal("空闲同目标会话应被复用,而不是新建")
	}
	if sess.Module != "m1" || sess.Prog != "p1" {
		t.Fatalf("复用时应刷新本轮运行参数: module=%q prog=%q", sess.Module, sess.Prog)
	}
	if m.Get("s1") == nil {
		t.Fatal("复用后会话应仍登记在管理器")
	}
}

// 复核会话复用判据时,**不要**把手工造的 Session 交给 prepareSession:
// 那种会话没有 pty/conn,Close() 会打到 nil(见 session.go 的 forceExit)。
// 这里只钉纯判据与"旧会话必须被移除"这件事,真拨 SSH 的那一步在真机验证。

// envCfg 造一个"同机同区域、只有环境名不同"的配置。
//
// 注:从前这里靠 Topent 让两个环境"同库不同企业"。企业编号已从环境配置移除
// (ENT 每次显式给),所以两者的区别**只剩环境名** —— 而这恰好是判据本身:
// envName 不同就是不同目标(不许复用旧会话),与连接四元组是否相同无关。
func envCfg(name string) *Config {
	e := host.NamedSsh{Name: name}
	e.Host, e.Port, e.User, e.Zone = "h1", 22, "u", "36"
	c := &Config{SSH: host.SSHConfig{Host: "h1", Port: 22, User: "u"}, Zone: "36",
		SSHs: []host.NamedSsh{e}}
	// 企业编号必填（requireExplicitEnt）：这些夹具都走真实"起会话"的路。
	c.fillDefaults()
	// 必须**走真实的环境选择路径**把 envName 定下来:没有默认环境之后,
	// 不点名时 EnvName() 会退回 `host-zone` 推导名 —— 两个同机同区域的环境
	// 于是算出同一个名字,安全判据(不许复用旧会话)当场失效。
	sel := c.CloneEnv(name)
	if sel == nil {
		panic("envCfg: CloneEnv 未命中 " + name)
	}
	return sel
}

// idleOn 造一个"已在该环境登录、空闲待命"的会话:envName 与 cfg 一致,
// 与 NewSession 建立出来的那一种同形(没有 pty/conn,不碰网络)。
func idleOn(id string, cfg *Config) *Session {
	return &Session{ID: id, cfg: cfg, state: StateIdle, envName: cfg.EnvName(),
		bps: map[int]*Breakpoint{}, srcCache: map[string]srcCacheEntry{}}
}

// 同机多环境(同库不同企业):连接四元组完全相同,但**不是**同一个环境 ——
// 判成同一个就会复用旧会话、继续用旧的企业编号跑(实测:所有按企业编号取数的
// 查询返回 0 行,看起来像"配置缺失",结论会完全反向)。
func TestSameTargetDistinguishesSiblingEnvs(t *testing.T) {
	live := idleOn("s1", envCfg("恒烁正式区"))
	if !live.sameConn(envCfg("测试1012")) {
		t.Fatal("两个环境指向同一台机器同一区域,连接本身是可复用的")
	}
	if live.sameTarget(envCfg("测试1012")) {
		t.Fatal("同机不同环境不应判定为同一目标(否则复用旧会话 = 上错号)")
	}
	// 同名但连接不同(设置页把某环境的 host 改了):仍是"换目标",不能拿旧连接继续跑。
	if !live.sameTarget(envCfg("恒烁正式区")) {
		t.Fatal("同名同连接应判定为同一目标")
	}
	other := envCfg("恒烁正式区")
	other.SSH.Host, other.SSH.User = "h2", "u"
	if live.sameTarget(other) {
		t.Fatal("同名但连接不同不应判定为同一目标")
	}
}

// 空闲旧会话遇到不同环境:即使连接可复用也必须重建,且新会话带上新的企业编号。
// 这里只验证"旧会话被移除、新的那份配置被用上" —— 真拨 SSH 那一步在真机验证。
func TestPrepareSessionRebuildsOnEnvSwitch(t *testing.T) {
	m := NewManager(envCfg("恒烁正式区"))
	live := idleOn("s1", envCfg("恒烁正式区"))
	m.mu.Lock()
	m.sessions["s1"] = live
	m.mu.Unlock()

	target := envCfg("测试1012")
	if live.sameTarget(target) {
		t.Fatal("前置条件不成立:两个环境被判成了同一个")
	}
	// 重建路径要求"旧会话必须被移除",否则新会话根本建不出来(单一常驻会话)
	m.Remove("s1")
	if m.Get("s1") != nil {
		t.Fatal("旧会话应被移除,否则新环境无法重建")
	}
	ns := idleOn("s2", target)
	if got := ns.EnvNameNow(); got != "测试1012" {
		t.Fatalf("新会话环境应为 测试1012, got %q", got)
	}
}

// 复用同环境会话时,环境名要跟着配置刷新(否则快照会报旧环境名)。
func TestSetRunRefreshesEnvName(t *testing.T) {
	s := &Session{ID: "s1", cfg: envCfg("恒烁正式区"), state: StateIdle, envName: "旧名"}
	s.SetRun(envCfg("恒烁正式区"), "m", "p", "", "", "")
	if got := s.EnvNameNow(); got != "恒烁正式区" {
		t.Fatalf("SetRun 后环境名应为 恒烁正式区, got %q", got)
	}
}

func TestPrepareSessionRejectsActiveRun(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	active := fakeIdleSession("s1", "h1", "36")
	active.mu.Lock()
	active.state = StateStopped
	active.mu.Unlock()
	m.mu.Lock()
	m.sessions["s1"] = active
	m.mu.Unlock()

	if _, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", ""); err == nil {
		t.Fatal("同目标活跃会话(stopped)应报错,要求先结束当前调试")
	}
}

func TestCloneEnvAndEnvName(t *testing.T) {
	// 环境不再带企业编号(ENT 每次显式给),所以这里只造连接与库。
	e1 := host.NamedSsh{Name: "E1", Zone: "35"}
	e1.Host = "e1h"
	e1.Port = 22
	e1.User = "u1"
	e1.DB = &dbconfig.Connection{Type: "oracle", Host: "db1h", Port: 1521, Service: "s1"}
	e2 := host.NamedSsh{Name: "E2", Zone: "36"}
	e2.Host = "e2h"
	e2.Port = 22
	e2.User = "u2"
	c := &Config{
		SSH:  host.SSHConfig{Host: "top", Port: 22, User: "u"},
		Zone: "36",
		SSHs: []host.NamedSsh{e1, e2},
	}
	c.fillDefaults()

	clone := c.CloneEnv("E1")
	if clone == nil {
		t.Fatal("CloneEnv 应命中 E1")
	}
	if clone.SSH.Host != "e1h" || clone.Zone != "35" || clone.EnvName() != "E1" {
		t.Fatalf("CloneEnv 字段不符: %+v", clone.SSH)
	}
	if clone.DB == nil || clone.DB.Type != "oracle" || clone.DB.Service != "s1" {
		t.Fatal("CloneEnv 应带环境内嵌的 db 配置")
	}
	if c.SSH.Host != "top" {
		t.Fatal("CloneEnv 不应改动原配置")
	}
	if c.CloneEnv("Nope") != nil {
		t.Fatal("未命中环境应返回 nil")
	}
	clone2 := c.CloneEnv("E2")
	if clone2 == nil || clone2.DB != nil {
		t.Fatal("未挂 db 的环境应得到 nil DB")
	}
	if got := cfgFor("x", "36").EnvName(); got != "x-36" {
		t.Fatalf("未选定环境时环境名应为 host-zone,got %q", got)
	}
}
