package debug

// 企业目录这层的测试。全部**不连网、不连库** —— "用缓存还是快照还是现查"那部分决策
// 是纯的,现查那一步用 EntCatalog.fetch 换掉。
//
// 要钉的正是那些错了不报错、只表现为"答案不对"的地方:
// 指纹算错 → 拿开发区的清单答正式区;降级判据写宽 → 用旧快照掩盖"连不上";
// 把"确实一个企业都没有"也当连不上 → 拿旧快照得出与事实相反的结论。

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/host"
)

// testCfg 造一份在内存里就能用的环境(不连任何东西)。
func testCfg(dataDir string) *Config {
	return &Config{
		DataDir: dataDir,
		SSH:     host.SSHConfig{Host: "10.0.0.1", User: "tiptop", Port: 22},
		Zone:    "36",
		DB:      &dbconfig.Connection{Type: "oracle", Host: "10.0.0.5", Port: 1521, Service: "t35prd"},
	}
}

var testEnts = []EntMapping{
	{Ent: 99, Account: "dsdemo"},
	{Ent: 907, Account: "-", Placeholder: true},
}

func noSSH(t *testing.T) func(*host.SSHConn, *dbRun) ([]EntMapping, error) {
	t.Helper()
	return func(*host.SSHConn, *dbRun) ([]EntMapping, error) {
		t.Error("这一步不该连服务器")
		return nil, nil
	}
}

// 指纹认的是"哪一份 gzou_t":换机器、换区域、换库都必须是另一个身份。
func TestEntFingerprint_ChangesWithIdentity(t *testing.T) {
	base := entFingerprint(testCfg(""))
	muts := map[string]func(*Config){
		"ssh.host": func(c *Config) { c.SSH.Host = "10.0.0.2" },
		"ssh.user": func(c *Config) { c.SSH.User = "root" },
		"zone":     func(c *Config) { c.Zone = "31" },
		"db.type":  func(c *Config) { c.DB.Type = "kingbase" },
		"db.host":  func(c *Config) { c.DB.Host = "10.0.0.6" },
		"db.port":  func(c *Config) { c.DB.Port = 1522 },
		"db.svc":   func(c *Config) { c.DB.Service = "t35dev" },
	}
	for name, mut := range muts {
		c := testCfg("")
		mut(c)
		if got := entFingerprint(c); got == base {
			t.Errorf("改了 %s 之后指纹没变 —— 换区域/换库会拿旧快照冒充", name)
		}
	}
	// 挂没挂库也算不同身份;且不能 panic
	c := testCfg("")
	c.DB = nil
	if entFingerprint(c) == base {
		t.Error("挂没挂库应当算不同身份")
	}
}

func TestEntSnapshotPath(t *testing.T) {
	dir := t.TempDir()

	cfg := testCfg(dir)
	cfg.envName = "示例正式区"
	p := entSnapshotPath(dir, cfg)
	if !strings.HasSuffix(p, filepath.Join("ents", "示例正式区.json")) {
		t.Errorf("中文环境名应直接可用: %s", p)
	}

	// 环境名为空 → 退化成 主机-区域(与 srccache 的命名约定一致)
	cfg2 := testCfg(dir)
	if p2 := entSnapshotPath(dir, cfg2); !strings.Contains(p2, "10.0.0.1-36") {
		t.Errorf("环境名为空时应退化为 主机-区域: %s", p2)
	}

	// 分隔符不许逃出 ents 目录
	cfg3 := testCfg(dir)
	cfg3.envName = `..\..\evil`
	got := entSnapshotPath(dir, cfg3)
	if filepath.Dir(got) != filepath.Join(dir, "ents") {
		t.Errorf("路径逃出了 ents 目录: %s", got)
	}

	// 没有数据目录 = 不落盘
	if entSnapshotPath("", cfg) != "" {
		t.Error("DataDir 为空时应当返回空路径(不落盘)")
	}
}

// 快照是"可丢弃的缓存":读不出、对不上、版本换了,一律当没有 —— 绝不能因此报错。
func TestEntSnapshot_RejectsWhatItShould(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	path := entSnapshotPath(dir, cfg)
	fp := entFingerprint(cfg)

	write := func(s *EntSnapshot) {
		t.Helper()
		if err := writeEntSnapshot(path, s); err != nil {
			t.Fatalf("writeEntSnapshot: %v", err)
		}
	}
	fresh := func() *EntSnapshot {
		return &EntSnapshot{Version: entSnapshotVer, Env: "e", Fingerprint: fp,
			Zone: "36", Dialect: "oracle", Target: "t35prd",
			Mappings: testEnts, FetchedAt: time.Now()}
	}

	write(fresh())
	if s := readEntSnapshot(path, fp); s == nil || len(s.Mappings) != 2 {
		t.Fatalf("读回失败: %+v", s)
	}
	if s := readEntSnapshot(path, fp+"|别的库"); s != nil {
		t.Error("指纹不符应当当没有(那是另一份 gzou_t)")
	}
	s := fresh()
	s.Version = entSnapshotVer + 1
	write(s)
	if got := readEntSnapshot(path, fp); got != nil {
		t.Error("版本不符应当当没有")
	}
	if err := os.WriteFile(path, []byte(`{"version":1,`), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readEntSnapshot(path, fp); got != nil {
		t.Error("半个 JSON 应当当没有,而不是让调用方拿到半份数据")
	}
	os.Remove(path)
	if got := readEntSnapshot(path, fp); got != nil {
		t.Error("文件不存在应当当没有")
	}
}

// 快照还新鲜时:**一次网络都不发**。这条靠"给 nil 连接也不报错"来证明 ——
// 真去拨号的话会连不上并报错。
func TestEntCatalog_FreshSnapshotNeedsNoNetwork(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	cfg.SSH.Host, cfg.SSH.Port = "127.0.0.1", 1 // 万一真拨号:立刻失败,不挂 20 秒

	if err := writeEntSnapshot(entSnapshotPath(dir, cfg), &EntSnapshot{
		Version: entSnapshotVer, Env: "e", Fingerprint: entFingerprint(cfg),
		Zone: "36", Dialect: "oracle", Target: "t35prd",
		Mappings: testEnts, FetchedAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	cat := NewEntCatalog(dir)
	cat.fetch = noSSH(t)
	l, err := cat.list(nil, nil, cfg, EntListOpt{})
	if err != nil {
		t.Fatalf("命中快照不该失败: %v", err)
	}
	if l.Source != "snapshot" || l.Stale {
		t.Errorf("应当是新鲜快照: source=%s stale=%v", l.Source, l.Stale)
	}
	if l.Count != 2 || l.TTLSeconds != int(EntSnapshotTTL.Seconds()) {
		t.Errorf("清单/新鲜期不对: count=%d ttl=%d", l.Count, l.TTLSeconds)
	}
}

// 降级规则:只有"连不上"才拿旧快照兜底;"答案是空"与"明确要新的"都不许回退。
func TestEntCatalog_DegradeRules(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	cfg.SSH.Host, cfg.SSH.Port = "127.0.0.1", 1
	conn, d := &host.SSHConn{}, &dbRun{}

	// ① 现查成功 → live,并且落盘
	cat := NewEntCatalog(dir)
	cat.fetch = func(*host.SSHConn, *dbRun) ([]EntMapping, error) { return testEnts, nil }
	l, err := cat.list(conn, d, cfg, EntListOpt{})
	if err != nil {
		t.Fatalf("首次现查: %v", err)
	}
	if l.Source != "live" || l.Stale {
		t.Fatalf("首次应当 live: %+v", l)
	}
	if l.Current == nil || l.Current.Resolved {
		t.Errorf("据点码式的非数字 topent 不该被解析成账号: %+v", l.Current)
	}

	// ② 新进程(内存空)+ 快照新鲜 → 不连网
	cat2 := NewEntCatalog(dir)
	cat2.fetch = noSSH(t)
	if l2, err := cat2.list(nil, nil, cfg, EntListOpt{}); err != nil || l2.Source != "snapshot" {
		t.Fatalf("应当命中刚写的快照: err=%v source=%v", err, l2.Source)
	}

	// ③ 快照过期 + 现查失败 → 降级,并如实标注
	if err := writeEntSnapshot(entSnapshotPath(dir, cfg), &EntSnapshot{
		Version: entSnapshotVer, Env: "e", Fingerprint: entFingerprint(cfg),
		Zone: "36", Dialect: "oracle", Target: "t35prd",
		Mappings: testEnts, FetchedAt: time.Now().Add(-EntSnapshotTTL - time.Minute),
	}); err != nil {
		t.Fatal(err)
	}
	cat3 := NewEntCatalog(dir)
	cat3.fetch = func(*host.SSHConn, *dbRun) ([]EntMapping, error) { return nil, errors.New("SSH 断了") }
	l3, err := cat3.list(conn, d, cfg, EntListOpt{})
	if err != nil {
		t.Fatalf("有旧快照时不该直接失败: %v", err)
	}
	if l3.Source != "snapshot-stale" || !l3.Stale {
		t.Errorf("应当是降级的过期快照: source=%s stale=%v", l3.Source, l3.Stale)
	}
	if len(l3.Notes) == 0 || !strings.Contains(l3.Notes[0], "SSH 断了") {
		t.Errorf("降级必须说明为什么不是实时数据: %v", l3.Notes)
	}

	// ④ --refresh + 失败 → 直接失败(明确要新的,给旧的等于骗人)
	if _, err := cat3.list(conn, d, cfg, EntListOpt{Refresh: true}); err == nil {
		t.Error("--refresh 时现查失败应当直接报错,不该回退旧快照")
	}

	// ⑤ "库答了,答案是空" → 不能拿旧快照冒充(那会得出与事实相反的结论)
	cat4 := NewEntCatalog(dir)
	cat4.fetch = func(*host.SSHConn, *dbRun) ([]EntMapping, error) { return nil, errEmptyCatalog }
	if _, err := cat4.list(conn, d, cfg, EntListOpt{}); err == nil {
		t.Error("查得空应当报错,而不是拿旧快照冒充")
	}

	// ⑥ --cached 且没有快照 → 报错,而不是偷偷连网
	empty := t.TempDir()
	cat5 := NewEntCatalog(empty)
	cat5.fetch = noSSH(t)
	if _, err := cat5.list(nil, nil, testCfg(empty), EntListOpt{Cached: true}); err == nil {
		t.Error("--cached 且没有快照时应当报错")
	}

	// ⑦ Offline 且没有快照 → (nil, nil):只是查不到,不报错也不连网。
	// 给"顺手补一行环境信息"的调用方用(tt debug start 的生效身份),别为它拨 SSH。
	cat6 := NewEntCatalog(t.TempDir())
	cat6.fetch = noSSH(t)
	l6, err := cat6.list(nil, nil, testCfg(""), EntListOpt{Offline: true})
	if err != nil {
		t.Errorf("Offline 没有快照时不该报错(它只是一句说明),got %v", err)
	}
	if l6 != nil {
		t.Errorf("Offline 没有快照时应返回 nil,got %+v", l6)
	}

	// ⑧ Offline 且有过期快照 → 照常给(调用方自己看新鲜度),同样不连网
	cat7 := NewEntCatalog(dir)
	cat7.fetch = noSSH(t)
	l7, err := cat7.list(nil, nil, cfg, EntListOpt{Offline: true})
	if err != nil || l7 == nil {
		t.Fatalf("Offline 命中快照时应当给出结果: err=%v", err)
	}
	if l7.Count != 2 || l7.Current == nil {
		t.Errorf("Offline 的结果内容应与现查同形: %+v", l7)
	}
}

// 快照写不出去(数据目录是个文件)不该让"查到了"变成失败 —— 它只是下次还得现查。
func TestEntCatalog_WriteFailureIsNonFatal(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := testCfg(blocker)

	cat := NewEntCatalog(blocker)
	cat.fetch = func(*host.SSHConn, *dbRun) ([]EntMapping, error) { return testEnts, nil }
	l, err := cat.list(&host.SSHConn{}, &dbRun{}, cfg, EntListOpt{})
	if err != nil {
		t.Fatalf("快照写不出去不该让查询失败: %v", err)
	}
	if l.Count != 2 {
		t.Errorf("结果应当照常返回: %+v", l)
	}
	if len(l.Notes) == 0 {
		t.Error("写不出去应当留一条说明,否则用户以为下次也能秒回")
	}
}

// 有企业、但账号列是空:这一行**必须留住**。
// 旧正则 `(\S+)` 会让它整行匹配不上,于是"企业 907 没配账号"被读成"企业 907 不存在"。
func TestParseEntLine_EmptyAccountIsKept(t *testing.T) {
	cases := []struct {
		in   string
		ent  int
		acct string
		ph   bool
	}{
		{"99|dsdemo", 99, "dsdemo", false},
		{"907|-", 907, "-", true},
		{"1001|", 1001, "-", true}, // ← 金仓下 gzou003 是空串,以前会被整行丢掉
		{"1002|   ", 1002, "-", true},
		{"  88|dsx  ", 88, "dsx", false},
	}
	for _, c := range cases {
		m, ok := parseEntLine(c.in)
		if !ok {
			t.Fatalf("%q 应当解析得出(丢掉这行就会把企业读成不存在)", c.in)
		}
		if m.Ent != c.ent || m.Account != c.acct || m.Placeholder != c.ph {
			t.Errorf("parseEntLine(%q) = %+v, 期望 ent=%d acct=%q placeholder=%v",
				c.in, m, c.ent, c.acct, c.ph)
		}
	}
	for _, noise := range []string{"", "表头", "ORA-00942: table or view does not exist", "企业|账号"} {
		if m, ok := parseEntLine(noise); ok {
			t.Errorf("%q 不该被当成一行数据: %+v", noise, m)
		}
	}
}

// --ent N = 只看这一个:收窄到一条;找不到 / 没配账号就报错(文案与 tt debug sql 一致);
// 而且**不许把缓存里的清单裁掉** —— 缓存那份是共享的,就地裁剪会让下一次不带 --ent 的
// 调用也只见一个企业。
func TestEntCatalog_EntFilter(t *testing.T) {
	dir := t.TempDir()
	cfg := testCfg(dir)
	cat := NewEntCatalog(dir)
	cat.fetch = func(*host.SSHConn, *dbRun) ([]EntMapping, error) { return testEnts, nil }
	conn, d := &host.SSHConn{}, &dbRun{}

	l, err := cat.list(conn, d, cfg, EntListOpt{Ent: 99})
	if err != nil {
		t.Fatalf("--ent 99: %v", err)
	}
	if l.Count != 1 || len(l.Ents) != 1 || l.Ents[0].Ent != 99 || l.Ents[0].Account != "dsdemo" {
		t.Fatalf("--ent 99 应当只留那一条: %+v", l.Ents)
	}
	if l.Current == nil || !l.Current.Resolved || l.Current.Source != "flag" {
		t.Errorf("当前企业应当标成来自 --ent 且已解析: %+v", l.Current)
	}

	// 再问全部:必须还是两条(收窄没改到共享的那份)
	all, err := cat.list(conn, d, cfg, EntListOpt{})
	if err != nil {
		t.Fatalf("再问全部: %v", err)
	}
	if all.Count != 2 {
		t.Errorf("收窄把缓存里的清单裁掉了: count=%d, 期望 2", all.Count)
	}

	// 不在清单里 → 报错,并带上可用企业号
	if _, err := cat.list(conn, d, cfg, EntListOpt{Ent: 8888}); err == nil {
		t.Error("--ent 不在清单里应当报错,而不是原样列出全部")
	} else if !strings.Contains(err.Error(), "8888") || !strings.Contains(err.Error(), "99") {
		t.Errorf("报错应当点出该企业号并列出可用的: %v", err)
	}

	// 有企业但没配账号 → 报错,不能把 "-" 当账号给出去
	// (这两种错要分开:"不存在"和"没配账号"要修的地方不一样)
	if _, err := cat.list(conn, d, cfg, EntListOpt{Ent: 907}); err == nil {
		t.Error("没配账号的企业应当报错")
	} else if !strings.Contains(err.Error(), "没有配可用账号") {
		t.Errorf("应当说清是「没配账号」而不是「不存在」: %v", err)
	}
}

// 配置里的 ENT 清单优先：存在就不联网（Offline 也必须有答案），Source=config，
// 且不按 TTL 判过期 —— 否则 10 分钟后它又会“变旧”，把用户推回连库。
func TestEntCatalogPrefersConfig(t *testing.T) {
	cfg := testCfg(t.TempDir())
	cfg.envName = "A"
	cfg.SSHs = []host.NamedSsh{{
		Name: "A", Zone: "36",
		Ents: []config.EntEntry{
			{Ent: 99, Account: "dsdemo"},
			{Ent: 907, Account: "dsdata"},
		},
		EntsSyncedAt: time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339), // 一个月前
	}}
	c := NewEntCatalog(cfg.DataDir)
	c.fetch = noSSH(t) // 一旦走到现查就是错

	l, err := c.list(nil, nil, cfg, EntListOpt{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	if l == nil || l.Source != "config" {
		t.Fatalf("应优先返回配置清单，得到 %+v", l)
	}
	if len(l.Ents) != 2 || l.Ents[0].Ent != 99 {
		t.Fatalf("清单不对: %+v", l.Ents)
	}
	if l.Stale {
		t.Error("配置清单不该按 TTL 判过期（新鲜度靠 entsSyncedAt 展示）")
	}
	acct, err := c.account(nil, nil, cfg, 907)
	if err != nil || acct != "dsdata" {
		t.Fatalf("account(907) = %q, %v", acct, err)
	}
	if _, err := c.account(nil, nil, cfg, 12345); err == nil {
		t.Error("配置里没有的企业应当报错（并列候选）")
	}
}
