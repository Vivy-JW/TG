package debug

// 企业目录(ENT → 账号)的**唯一**取值处:
//
//	进程内缓存 → 落盘快照 → 现查(SSH + ds 读 gzou_t)
//
// 为什么要有快照:agent 回答"这个环境有哪些企业、各用哪个账号"这件事要反复问,而每问一次
// 都得 SSH 握手 + 探测工具路径 + 连库查 gzou_t(秒级)。快照让第二次之后秒回,也让断网 /
// 没起 serve 时仍能答 —— 代价是可能过期,所以**新鲜度必须如实带出去**(Source/Stale/FetchedAt),
// 让调用方自己决定信不信。
//
// 快照是**可丢弃的缓存,不是数据源**:读坏了 / 写不出 / 指纹不符,一律当没有,绝不因此报错
// (与 srcmirror.go 的 writeMirror 同一条原则)。
//
// 注意"企业 → 数据库"这个词是不准的:库是**环境级**的(hosts.sshs[].db,一对一挂),
// 企业编号只决定**库里的账号 / schema**(gzou_t.gzou003)。所以这里解析出来的永远是账号。

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"tt/internal/entdir"
	"tt/internal/host"
)

const (
	// EntSnapshotTTL 快照与进程内缓存共用的新鲜期。
	//
	// 与只读 SQL 的映射缓存(sqlEntCacheTTL)共用同一个值:两条命令必须对外表现出同一份
	// 新鲜度,否则 agent 会拿到两条互相矛盾的时间线(一边说企业 99 在,一边说不在)。
	// 唯一实现在 internal/entdir —— tt dict 的客户端直连路径也读同一份快照。
	EntSnapshotTTL = entdir.TTL

	entSnapshotVer = entdir.Version
)

// errEmptyCatalog 查询成功、但结果里没有一个启用企业。
//
// 这**不是**"连不上" —— 它是"库答了,答案是空"。调用方必须把它和网络/权限错误分开:
// 前者用旧快照兜底会得出"有 128 个企业"这种与当下事实相反的结论,比直接失败更坏。
var errEmptyCatalog = errors.New("gzou_t 无有效企业记录")

// EntCurrent 本次生效的企业编号,以及它能不能定位到账号。
// 它解释的是"你这次问的是哪个企业",不是"有哪些企业"。
type EntCurrent struct {
	Ent      int    `json:"ent"`      // 0 = 未定(TOPENT 非数字 / 没配)
	Raw      string `json:"raw"`      // 原始值,非数字时一眼可见(如把据点码填错位置的 "SITE01")
	Source   string `json:"source"`   // flag(--ent) | config(环境 topent) | none
	Resolved bool   `json:"resolved"` // 能否据此定位账号
	Account  string `json:"account,omitempty"`
	Reason   string `json:"reason,omitempty"` // Resolved=false 时说明为什么
}

// EntListing 一次查询的完整结果,也是 CLI --json 的直接载荷。
type EntListing struct {
	Env       string       `json:"env"`
	Zone      string       `json:"zone"`
	Dialect   string       `json:"dialect"` // oracle | kingbase
	Target    string       `json:"target"`  // oracle service / kingbase 库名
	TopentRaw string       `json:"topentRaw"`
	Count     int          `json:"count"`
	Ents      []EntMapping `json:"ents"`
	Current   *EntCurrent  `json:"current,omitempty"`

	// 新鲜度。agent 判断"这份答案能不能信"全靠这几个字段。
	FetchedAt   time.Time `json:"fetchedAt"`
	AgeSeconds  int       `json:"ageSeconds"`
	TTLSeconds  int       `json:"ttlSeconds"`
	Stale       bool      `json:"stale"`  // true = 已过期,且是现查失败后降级拿到的
	Source      string    `json:"source"` // live | snapshot | snapshot-stale
	Fingerprint string    `json:"fingerprint"`

	Notes []string `json:"notes,omitempty"`
}

// EntSnapshot 落盘的企业目录快照(<DataDir>/ents/<环境段>.json)。
// 定义在 internal/entdir:tt dict 的客户端直连路径读写同一个文件,两边必须是同一份格式。
type EntSnapshot = entdir.Snapshot

// EntCatalog 企业目录。零值不可用,用 NewEntCatalog 构造。
type EntCatalog struct {
	dataDir string
	mu      sync.Mutex
	mem     map[string]entMemEntry

	// fetch 是"现查 gzou_t"这一步。生产路径恒为 dbAllMappings;
	// 单测替换它就能覆盖"查得到 / 查得空 / 连不上"三种结果,不必真有 SSH 与库。
	fetch func(conn *host.SSHConn, d *dbRun) ([]EntMapping, error)
}

type entMemEntry struct {
	listing *EntListing
	at      time.Time
}

// NewEntCatalog dataDir 为空表示只走内存、不落盘(与 writeMirror 同款兜底:
// 单测里 NewManager(&Config{}) 就是这种情形,必须安全)。
func NewEntCatalog(dataDir string) *EntCatalog {
	return &EntCatalog{dataDir: dataDir, mem: map[string]entMemEntry{}, fetch: dbAllMappings}
}

// ---- 纯函数(不碰 SSH / 磁盘,便于单测) ----

// entFingerprint 环境指纹:换机器 / 换区域 / 换库 = 换了另一份 gzou_t,快照立即失效。
// 算法在 internal/entdir(tt dict 用同一份,两个路径必须对同一环境算出同一个指纹)。
func entFingerprint(cfg *Config) string {
	ident := "none"
	if cfg.DB != nil {
		ident = entdir.DBIdent(cfg.DB.Type, cfg.DB.Host, cfg.DB.Port, cfg.DB.Svc())
	}
	return entdir.Fingerprint(cfg.SSH.Host, cfg.SSH.User, cfg.Zone, ident)
}

// entSnapshotPath 快照路径:<DataDir>/ents/<环境段>.json。环境段复用镜像那套命名。
func entSnapshotPath(dataDir string, cfg *Config) string {
	return entdir.Path(dataDir, entdir.EnvSeg(cfg.EnvName(), cfg.SSH.Host, cfg.Zone))
}

// readEntSnapshot 读快照。不存在 / 坏掉 / 版本不符 / 指纹不符一律当没有(返回 nil)。
// **不判过期** —— 过期快照在"现查失败"时是唯一的答案来源,用不用由调用方定。
func readEntSnapshot(path, fingerprint string) *EntSnapshot {
	return entdir.Read(path, fingerprint)
}

// writeEntSnapshot 原子落盘。失败只返回错误,由调用方降级成一条 note ——
// 快照写不出去不该让"查到了"这件事变成失败。
func writeEntSnapshot(path string, s *EntSnapshot) error {
	return entdir.Write(path, s)
}

// ---- 目录 ----

// EntListOpt 一次查询的选项。
type EntListOpt struct {
	// Ent 调用方显式指定的企业编号(--ent);>0 时回填 Current 的 Source=flag。
	Ent int
	// Refresh 跳过内存与快照,强制现查;**现查失败即失败**,不回退旧快照
	// (调用方明确要"新的",给旧的等于骗人)。
	Refresh bool
	// Cached 只用内存/快照,完全不连网;没有可用快照就报错。
	// 给 agent 一条确定性的快回路:不必猜 SSH 超时要等多久。
	Cached bool
	// Offline 禁止现查:与 Cached 的区别是**没有快照也只是查不到**(返回 nil,不报错)。
	//
	// 给"顺手补一行环境信息"这类调用方用:`tt debug start` 的「生效身份」要顺带说出
	// 企业编号对应哪个账号,但那只是输出里的说明,不值得为它拨一次 SSH(慢时 40 秒),
	// 更不该因为它没查到就把启动判成失败。要权威答案请用 tt debug ents。
	Offline bool
}

// LookupEnts 企业目录查询的**唯一对外入口**(CLI 与其它调用方用)。
//
// 每次新建一个 EntCatalog 是刻意的:CLI 是一次性进程,它要的是快照而不是进程内缓存;
// 长驻的 serve 侧自己去 NewManager 里那份,那份才有跨调用缓存。
//
// 不需要调用方管连接:命中缓存/快照时**一次网络都不发**,只有真要现查才拨 SSH。
func LookupEnts(cfg *Config, opt EntListOpt) (*EntListing, error) {
	return NewEntCatalog(cfg.DataDir).list(nil, nil, cfg, opt)
}

// list 取企业目录,并在给了 --ent 时收窄到那一条。
//
// 收窄放在这里而不是 resolve 里,是为了**不改到缓存里的那份**:缓存的 listing 是共享的,
// 就地裁剪会让下一次不带 --ent 的调用也只看到一个企业。
func (c *EntCatalog) list(conn *host.SSHConn, d *dbRun, cfg *Config, opt EntListOpt) (*EntListing, error) {
	l, err := c.resolve(conn, d, cfg, opt)
	// l == nil 是 Offline 模式"没有快照、也没去查"的正常结果(见 EntListOpt.Offline):
	// 调用方拿到 (nil, nil) 就是"这次没有目录可用",不该再往下走去做收窄。
	if err != nil || l == nil || opt.Ent <= 0 {
		return l, err
	}
	// --ent N = "只看这一个"。找不到 / 没配账号就**报错**,而且复用 pickEntAccount,
	// 让文案与 tt debug sql 完全一致 —— 同一件事两条命令说两样话,最容易被当成两个问题。
	acct, err := pickEntAccount(l.Ents, opt.Ent)
	if err != nil {
		return nil, err
	}
	cp := *l
	cp.Ents = []EntMapping{{Ent: opt.Ent, Account: acct}}
	cp.Count = 1
	return &cp, nil
}

// resolve 取企业目录(不处理 --ent 的收窄)。
//
// 取值顺序:内存 → 快照 → 现查。降级规则见 errEmptyCatalog 的注释:
// 只有"连不上 / 查不动"才回退旧快照,且回退后 Stale=true、Notes 里带上失败原因。
//
// conn 为 nil = 调用方没现成的连接,需要现查时自己拨一个(用完关);
// conn 非 nil = 调用方借的(如会话已有的那条),**本函数绝不去关它**。
func (c *EntCatalog) resolve(conn *host.SSHConn, d *dbRun, cfg *Config, opt EntListOpt) (*EntListing, error) {
	fp := entFingerprint(cfg)
	path := entSnapshotPath(c.dataDir, cfg)
	now := time.Now()

	// 配置里的 ENT 清单优先：它是人显式同步过的那一份（设置页「ENT」页签 /
	// tt env ents --refresh），存在即权威，**一次网络都不发**。
	// Refresh 明确要求“现查”时才跳过它 —— 那是同步动作，不是日常读取。
	if !opt.Refresh {
		if l := configListing(cfg, now); l != nil {
			return finish(l, cfg, opt, now), nil
		}
	}

	if !opt.Refresh {
		if l := c.memGet(fp, now); l != nil {
			return finish(l, cfg, opt, now), nil
		}
	}

	snap := readEntSnapshot(path, fp)
	if snap != nil && !opt.Refresh {
		l := entSnapshotToListing(snap, cfg, opt, now, "snapshot")
		if !l.Stale {
			c.memPut(fp, l, now)
			return finish(l, cfg, opt, now), nil
		}
		// 过期:不直接返回,继续往下尝试现查(下面失败时才拿它兜底)
	}

	if opt.Cached {
		if snap != nil {
			return finish(entSnapshotToListing(snap, cfg, opt, now, "snapshot-stale"), cfg, opt, now), nil
		}
		return nil, fmt.Errorf("没有可用的企业目录快照(%s);先联网跑一次 tt env ents --refresh", path)
	}

	// Offline 在 Cached 之后:有快照就照常给(哪怕是过期的,调用方自己看新鲜度),
	// 没快照就**查不到** —— 不报错也不拨 SSH。见 EntListOpt.Offline。
	if opt.Offline {
		if snap != nil {
			return finish(entSnapshotToListing(snap, cfg, opt, now, "snapshot"), cfg, opt, now), nil
		}
		return nil, nil
	}

	// 到这里确定要连服务器了。--cached 那条路在上面就返回了,所以"完全不连网"
	// 这个承诺是由这个位置兑现的,不是靠调用方自觉。
	if conn == nil {
		c2, err := host.Dial(cfg.SSH)
		if err != nil {
			return nil, fmt.Errorf("SSH 连接失败: %w", err)
		}
		defer c2.Close()
		conn = c2
		if d, err = resolveDBRun(conn, cfg); err != nil {
			return nil, err
		}
	}

	fetch := c.fetch
	if fetch == nil {
		fetch = dbAllMappings
	}
	live, err := fetch(conn, d)
	if err != nil {
		if errors.Is(err, errEmptyCatalog) || opt.Refresh {
			return nil, err // 见 errEmptyCatalog:答案为空 / 明确要新的,都不回退
		}
		if snap != nil {
			l := entSnapshotToListing(snap, cfg, opt, now, "snapshot-stale")
			l.Stale = true
			l.Notes = append(l.Notes, "现查失败,以下是过期快照: "+err.Error())
			return finish(l, cfg, opt, now), nil
		}
		return nil, err
	}

	l := &EntListing{
		Env: cfg.EnvName(), Zone: zoneOr36(cfg.Zone), Dialect: dialectOf(cfg),
		Target: targetOf(cfg), TopentRaw: string(cfg.Topent),
		Ents: live, Source: "live", FetchedAt: now, Fingerprint: fp,
	}
	if err := writeEntSnapshot(path, &EntSnapshot{
		Version: entSnapshotVer, Env: l.Env, Fingerprint: fp, Zone: l.Zone,
		Dialect: l.Dialect, Target: l.Target, Mappings: live, FetchedAt: now,
	}); err != nil {
		// 快照只是给下次的便利,写不出去不该让这次查询失败
		l.Notes = append(l.Notes, "快照未能落盘(下次仍要现查): "+err.Error())
	}
	c.memPut(fp, l, now)
	return finish(l, cfg, opt, now), nil
}

// account 严格版:企业编号必须解析出账号,否则报错。
//
// **只读 SQL 走这条**,保持 dbsql.go 立下的规矩:解析不到就报错,不静默回退 ds ——
// 回退到 ds 会安安静静地查另一个 schema,拿到 0 行,然后被当成"没有这条数据"。
func (c *EntCatalog) account(conn *host.SSHConn, d *dbRun, cfg *Config, ent int) (string, error) {
	if ent <= 0 {
		return "", fmt.Errorf("企业编号无效: %d", ent)
	}
	l, err := c.list(conn, d, cfg, EntListOpt{Ent: ent})
	if err != nil {
		return "", err
	}
	return pickEntAccount(l.Ents, ent)
}

// ListEnts 用**服务端那份** catalog 查企业目录(与 tt debug ents 同一套实现、同一份缓存)。
//
// 长驻进程里这份 catalog 的内存缓存与快照是跨调用共享的:接口日志页与「创建会话」的
// ENT 下拉每开一次都问一次,命中缓存时**一次网络都不发**(见 EntListOpt / resolve)。
func (m *Manager) ListEnts(cfg *Config, opt EntListOpt) (*EntListing, error) {
	return m.ents.list(nil, nil, cfg, opt)
}

func (c *EntCatalog) memGet(fp string, now time.Time) *EntListing {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.mem[fp]
	if !ok || now.Sub(e.at) >= EntSnapshotTTL {
		return nil
	}
	return e.listing
}

func (c *EntCatalog) memPut(fp string, l *EntListing, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mem[fp] = entMemEntry{listing: l, at: now}
}

// finish 补齐"每次都要重算"的派生字段(年龄 / 新鲜度 / 当前企业),
// 让缓存里存的与返回给调用方的保持同一份。
func finish(l *EntListing, cfg *Config, opt EntListOpt, now time.Time) *EntListing {
	l.Count = len(l.Ents)
	l.TTLSeconds = int(EntSnapshotTTL.Seconds())
	l.AgeSeconds = int(now.Sub(l.FetchedAt).Seconds())
	if l.AgeSeconds < 0 {
		l.AgeSeconds = 0
	}
	switch l.Source {
	case "config":
		// 配置清单本身就是“抄下来的那一份”：不按 TTL 判过期（否则 10 分钟后就
		// 变成 stale，又会去拨网）。新鲜度用 FetchedAt/AgeSeconds 如实展示。
		l.Stale = false
	default:
		l.Stale = l.Source == "snapshot-stale" || now.Sub(l.FetchedAt) >= EntSnapshotTTL
	}
	l.Current = entCurrent(cfg, opt.Ent, l.Ents)
	return l
}

// entCurrent 说明本次问的是哪个企业、能不能定位到账号。
// 数字但不在清单里、非数字(如把据点码填错位置)、没配 —— 三种都给得出原因。
func entCurrent(cfg *Config, flagEnt int, ents []EntMapping) *EntCurrent {
	if flagEnt > 0 {
		cur := &EntCurrent{Ent: flagEnt, Raw: strconv.Itoa(flagEnt), Source: "flag"}
		return resolveCur(cur, ents)
	}
	raw := strings.TrimSpace(string(cfg.Topent))
	if raw == "" {
		return &EntCurrent{Source: "none", Reason: "环境没有配 topent(企业编号)"}
	}
	n, _ := cfg.Topent.Int()
	return resolveCur(&EntCurrent{Ent: n, Raw: raw, Source: "config"}, ents)
}

func resolveCur(cur *EntCurrent, ents []EntMapping) *EntCurrent {
	if cur.Ent <= 0 {
		cur.Reason = fmt.Sprintf("TOPENT 不是数字(%s),无法据此定位账号", cur.Raw)
		return cur
	}
	for _, m := range ents {
		if m.Ent != cur.Ent {
			continue
		}
		if m.Account == "" || m.Account == "-" {
			cur.Reason = fmt.Sprintf("企业 %d 在 gzou_t 里没有配可用账号", cur.Ent)
			return cur
		}
		cur.Resolved, cur.Account = true, m.Account
		return cur
	}
	cur.Reason = fmt.Sprintf("企业 %d 不在 gzou_t 的启用清单里", cur.Ent)
	return cur
}

// entSnapshotToListing 把快照转成 listing。它是自由函数而不再是 EntSnapshot 的方法:
// EntSnapshot 现在是 internal/entdir 的类型别名,而 Go 不允许在非本地类型上定义方法。
func entSnapshotToListing(s *EntSnapshot, cfg *Config, opt EntListOpt, now time.Time, source string) *EntListing {
	l := &EntListing{
		Env: s.Env, Zone: s.Zone, Dialect: s.Dialect, Target: s.Target,
		TopentRaw: string(cfg.Topent), Ents: s.Mappings, Source: source,
		FetchedAt: s.FetchedAt, Fingerprint: s.Fingerprint,
	}
	_ = finish(l, cfg, opt, now)
	return l
}

func zoneOr36(z string) string {
	if z == "" {
		return "36"
	}
	return z
}

func dialectOf(cfg *Config) string {
	if cfg.DB != nil && cfg.DB.Type == "kingbase" {
		return "kingbase"
	}
	return "oracle"
}

func targetOf(cfg *Config) string {
	if cfg.DB == nil {
		return ""
	}
	return cfg.DB.Svc()
}

// configListing 把 config.json 里该环境的 ENT 清单转成 EntListing。
// 没配置（或清单为空）返回 nil —— 调用方回落到快照/现查（老配置的兼容路径）。
func configListing(cfg *Config, now time.Time) *EntListing {
	name := cfg.EnvName()
	for i := range cfg.SSHs {
		if cfg.SSHs[i].Name != name {
			continue
		}
		e := &cfg.SSHs[i]
		if len(e.Ents) == 0 {
			return nil
		}
		ents := make([]EntMapping, 0, len(e.Ents))
		for _, x := range e.Ents {
			acct := x.Account
			if acct == "" {
				acct = "-"
			}
			ents = append(ents, EntMapping{Ent: x.Ent, Account: acct, Placeholder: acct == "-"})
		}
		l := &EntListing{
			Env: cfg.EnvName(), Zone: zoneOr36(cfg.Zone), Dialect: dialectOf(cfg),
			Target: targetOf(cfg), TopentRaw: string(cfg.Topent),
			Ents: ents, Source: "config", FetchedAt: now, Fingerprint: entFingerprint(cfg),
			Notes: []string{"来自配置 json（ENT 页签/ tt env ents --refresh 同步的那一份），不联网"},
		}
		if e.EntsSyncedAt != "" {
			if t, err := time.Parse(time.RFC3339, e.EntsSyncedAt); err == nil {
				l.FetchedAt = t
				l.Notes = append(l.Notes, "同步于 "+e.EntsSyncedAt)
			}
		}
		return l
	}
	return nil
}

// ForgetEnts 丢掉某环境的进程内缓存。配置清单被同步改写后必须调用，
// 否则下一次读还会命中旧的那份（mem 以指纹为键，与清单内容无关）。
func (m *Manager) ForgetEnts(cfg *Config) {
	fp := entFingerprint(cfg)
	m.ents.mu.Lock()
	delete(m.ents.mem, fp)
	m.ents.mu.Unlock()
}
