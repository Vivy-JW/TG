package debug

// 这组测试盯的是"点名了环境就必须按它连"这条线。
//
// 从前 `tt debug start … --env X` 既不报错也不生效(--env 是根命令的开关,debug 组里
// 没人读它),方向永远取服务端当前环境;同机多环境(同库不同企业)时那就是"上错号",
// 所有按企业编号取数的查询返回 0 行,看起来与"配置缺失"一模一样。这些用例把三件事钉住:
// 点名怎么解析、生效身份怎么表述、不一致时怎么报。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/cli/common"
	"tt/internal/entdir"
	"tt/internal/target"
)

// 环境名解析:子命令上的 --env 优先,其次根命令的持久 --env,再没有就是空(用当前环境)。
func TestDBGTargetEnv(t *testing.T) {
	oldFlag, oldGlobal := dbgEnvFlag, common.Env
	t.Cleanup(func() { dbgEnvFlag, common.Env = oldFlag, oldGlobal })

	dbgEnvFlag, common.Env = "", ""
	if got := dbgTargetEnv(); got != "" {
		t.Fatalf("两条都没给时应为空(用服务端当前环境),got %q", got)
	}
	dbgEnvFlag, common.Env = "", "根命令给的"
	if got := dbgTargetEnv(); got != "根命令给的" {
		t.Fatalf("根命令 --env 应被认(别处能用,这里也得能用),got %q", got)
	}
	dbgEnvFlag, common.Env = "子命令给的", "根命令给的"
	if got := dbgTargetEnv(); got != "子命令给的" {
		t.Fatalf("子命令 --env 应优先,got %q", got)
	}
	dbgEnvFlag, common.Env = "  带空白  ", ""
	if got := dbgTargetEnv(); got != "带空白" {
		t.Fatalf("两侧空白应剔除,got %q", got)
	}
}

// 点名了却没生效:必须报警并判失败;一致或没点名则放行。
func TestDBGVerifyEnv(t *testing.T) {
	oldFlag, oldGlobal := dbgEnvFlag, common.Env
	t.Cleanup(func() { dbgEnvFlag, common.Env = oldFlag, oldGlobal })
	dbgEnvFlag, common.Env = "", ""

	snap := &dbgSnapView{Session: target.Session{Env: "测试1012", Prog: "capp115"}}
	if !dbgVerifyEnv("", snap) {
		t.Fatal("没点名环境时不该报失败")
	}
	if !dbgVerifyEnv("测试1012", snap) {
		t.Fatal("点名与实际一致时不该报失败")
	}
	if dbgVerifyEnv("恒烁正式区", snap) {
		t.Fatal("点名与实际不一致时必须判失败(否则调用方以为跑在了点名的环境上)")
	}
	if !dbgVerifyEnv("恒烁正式区", nil) {
		t.Fatal("拿不到快照时不误报(另一条路会给失败)")
	}
}

// 复用了宿主要有痕迹:只有"回包 id 与启动前那个空闲会话相同"才算复用。
func TestDBGReusedHost(t *testing.T) {
	cases := []struct {
		prev, id string
		want     bool
	}{
		{"s1791770107217", "s1791770107217", true},  // 同一个空闲会话被复用
		{"s1791770107217", "s1791957373680", false}, // 新建了一个
		{"", "s1791957373680", false},               // 启动前没有空闲会话
		{"s1791770107217", "", false},               // 没拿到 id:不猜
	}
	for _, c := range cases {
		if got := dbgReusedHost(c.prev, c.id); got != c.want {
			t.Errorf("dbgReusedHost(%q, %q) = %v, 期望 %v", c.prev, c.id, got, c.want)
		}
	}
}

// 身份行（契约 C5.9）里必须同时有 会话/display/别名/状态/企业，第二行有 账号/库/区域，
// 以及"复用了宿主"的痕迹。
//
// **这是本批唯一改了单会话可见输出的地方**（契约 C8.4 登记的第二处变化）：
// 从前这里是一行 `生效身份: 环境 … / 区域 … / 企业 … / 账号 … / 库 …`，
// 现在按 C5.9 定稿成
//
//	会话 <display> · <name> · <state> · ENT <ent>
//	  账号 <acct> · 库 <target> · 区域 <zone>
//
// 两行**缺一个字段都不行**（C5.9：取不到写字面量，不省整段 —— 否则字段位置会随会话漂）。
func TestDBGIdentity(t *testing.T) {
	t.Setenv("TT_CONFIG", "") // 隔离真实 TT_CONFIG,否则 dbgIdentity 查库会读到生产 config.json
	oldPath := common.ConfigPath
	common.ConfigPath = writeIdentityConfig(t)
	t.Cleanup(func() { common.ConfigPath = oldPath })

	snap := &dbgSnapView{
		Session: target.Session{Env: "恒烁正式区", N: 1, Prog: "apmt520", State: "stopped", Ent: "9999"},
		Zone:    "1",
		// TopentCf 是**既有**的一栏（第二行"账号"要靠它从企业快照里查），留在这里正是要钉住
		// "它不进第一行的 `ENT`"：第一行的 `ENT` 只认快照的 `ent`（C5.9 / C5.1 / C5.3）。
		TopentCf: "9999",
		Reused:   true,
	}
	got := dbgIdentity(snap)
	for _, want := range []string{
		"会话 恒烁正式区/apmt520#1 · (未命名) · 已停站 · ENT 9999", // 第一行：C5.9 逐字
		"账号 -", "库 172.19.93.55:54321/topprd", "区域 1",
		"复用了已登录的宿主",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("身份行缺少 %q:\n%s", want, got)
		}
	}
	// 没有企业快照时不许拨 SSH(慢时 40 秒),但要说明查不到并给出自己核对的命令
	if !strings.Contains(got, "没有该环境的企业快照") || !strings.Contains(got, "tt debug ents --env 恒烁正式区") {
		t.Errorf("没有快照时应说明为什么没有账号、并给出核对命令:\n%s", got)
	}
	// 三个裸字段全空时也不能打出空白:企业编号要写 `?`(C5.9 明写取不到写 `?`，不省整段)
	if g := dbgIdentity(&dbgSnapView{Session: target.Session{Env: "恒烁正式区", N: 1, Prog: "capp115"}}); !strings.Contains(g, "ENT ?") {
		t.Errorf("企业三个字段都空时该写 ENT ?（C5.9），实得:\n%s", g)
	}
	// 未命名 / 状态取不到时同样写字面量，四个字段的位置不漂。
	un := dbgIdentity(&dbgSnapView{Session: target.Session{Env: "恒烁正式区", N: 2, Prog: "capp115"}})
	if !strings.HasPrefix(un, "会话 恒烁正式区/capp115#2 · (未命名) · ") {
		t.Errorf("未命名该写 (未命名)、display 该是 环境/作业#序号（C5.9），实得:\n%s", un)
	}
	if dbgIdentity(nil) != "" {
		t.Error("没有快照时应返回空串(不能打出半行)")
	}
}

// TestDBGIdentityEntIsTheSnapshotsEntOnly 钉住 C5.9 身份行里 `<ENT>` 的**唯一来源**：
// 快照的 `ent`（服务端 `SessionBrief.Ent`，C5.1 / C5.3）。
//
// 上一批这里还有一层兼容兜底（`dbgIdentityEnt`：快照没给 `ent` 时按
// shell 回读 / topent / topentSession / topentCfg 的顺序凑一个）。那层已按验收判定**删净** ——
// 留着它就等于让客户端手上同时有 `ent` 与 `topentSession` 两个近似的东西、还要在它们之间
// 做取舍，而那正是 C5.17–C5.19 一路在拆的"第二份事实"。快照现在直接给 `ent`，所以：
//
//	① `ent` 与 `topentSession` **不同值**时，身份行必须取 `ent`；
//	② 只有 `topentSession` / `topentShell` / `topent` / `topentCfg`、没有 `ent` 时，
//	   身份行必须写 C5.9 的占位符 `?` —— 一个字节都不许拿别的字段顶。
//
// 输入走**真实的快照 JSON**（`json.Unmarshal` 进 `dbgSnapView`，与 `dbgSnapOf` 同一条路），
// 不是手工拼 struct：这条断言要的正是"`ent` 这个键真的接到身份行上"。
func TestDBGIdentityEntIsTheSnapshotsEntOnly(t *testing.T) {
	oldPath := common.ConfigPath
	common.ConfigPath = writeIdentityConfig(t)
	t.Cleanup(func() { common.ConfigPath = oldPath })

	// ① 服务端同时给了两个键、且值不同：`ENT` 取 `ent`（99），不是 `topentSession`（66）。
	var snap dbgSnapView
	raw := `{"id":"s1","env":"恒烁正式区","n":1,"prog":"apmt520","state":"stopped",
	         "ent":"99","topentSession":"66","topent":"77","topentShell":"88","zone":"1"}`
	if err := json.Unmarshal([]byte(raw), &snap); err != nil {
		t.Fatalf("快照 JSON 解不开：%v", err)
	}
	if snap.Ent != "99" {
		t.Fatalf("前置不成立：快照的 `ent` 该解到 %q，实得 %q —— 这条断言靠的就是它", "99", snap.Ent)
	}
	first := strings.SplitN(dbgIdentity(&snap), "\n", 2)[0]
	if !strings.Contains(first, "ENT 99") {
		t.Errorf("身份行的 `ENT` 必须取自快照的 `ent`（C5.9 / C5.1）：期望 %q，实得 %q", "ENT 99", first)
	}
	for _, other := range []string{"66", "77", "88"} {
		if strings.Contains(first, "ENT "+other) {
			t.Errorf("身份行第一行的 `ENT` 取到了别的字段（%s）—— 客户端不许在 `ent` 与 "+
				"`topentSession` / `topentShell` / `topent` 之间做取舍（C5.9 / C5.17–C5.19），实得 %q",
				other, first)
		}
	}

	// ② 只有裸字段、没有 `ent`：写占位符 `?`（C5.9 明写取不到写 `?`，不省整段）。
	var bare dbgSnapView
	rawBare := `{"id":"s2","env":"恒烁正式区","n":2,"prog":"capp115","state":"idle",
	             "topentSession":"66","topent":"77","topentShell":"88","topentCfg":"55","zone":"1"}`
	if err := json.Unmarshal([]byte(rawBare), &bare); err != nil {
		t.Fatalf("快照 JSON 解不开：%v", err)
	}
	if bare.Ent != "" {
		t.Fatalf("前置不成立：这份快照里没有 `ent`，该解成空串，实得 %q", bare.Ent)
	}
	firstBare := strings.SplitN(dbgIdentity(&bare), "\n", 2)[0]
	if !strings.Contains(firstBare, "ENT ?") {
		t.Errorf("快照没给 `ent` 时要写 C5.9 的占位符 `ENT ?`，实得 %q\n"+
			"    （上一批那种「退回 topentSession」的兜底已删净，不许复活）", firstBare)
	}
	for _, other := range []string{"66", "77", "88", "55"} {
		if strings.Contains(firstBare, "ENT "+other) {
			t.Errorf("快照没给 `ent`，`ENT` 却取到了别的字段（%s）—— 兜底链不许复活（C5.1 / C5.9），实得 %q",
				other, firstBare)
		}
	}
}

// 有企业快照时,身份行要直接给出**账号** —— 企业编号错了就会连到另一个 schema,
// 而"上错号"与"数据不存在"在输出上一模一样,这一行就是用来分开它们的。
//
// C5.9 起账号住在**第二行**（`  账号 <acct> · 库 <target> · 区域 <zone>`）：
// 第一行按契约只有四个身份字段，多塞一段就会让"对账用 ref、认领看 display"那条规矩失效。
func TestDBGIdentityWithEntSnapshot(t *testing.T) {
	t.Setenv("TT_CONFIG", "") // 隔离真实 TT_CONFIG,否则查库/账号会读到生产 config.json
	dir := t.TempDir()
	cfgPath := writeIdentityConfigIn(t, dir)

	// 快照文件名与内容都照 internal/entdir 的契约来(与 tt debug ents 落的是同一份)
	const (
		sshHost, sshUser, zone = "172.19.93.55", "tiptop", "1"
		dialect, dbHost, svc   = "kingbase", "172.19.93.55", "topprd"
	)
	fp := entdir.Fingerprint(sshHost, sshUser, zone, entdir.DBIdent(dialect, dbHost, 54321, svc))
	path := entdir.Path(dir, entdir.EnvSeg("恒烁正式区", sshHost, zone))
	if err := entdir.Write(path, &entdir.Snapshot{
		Version: entdir.Version, Env: "恒烁正式区", Fingerprint: fp, Zone: zone,
		Dialect: dialect, Target: svc, FetchedAt: time.Now(),
		Mappings: []entdir.Mapping{{Ent: 9999, Account: "dsdata"}, {Ent: 1012, Account: "dsdemo"}},
	}); err != nil {
		t.Fatalf("写快照失败: %v", err)
	}

	oldPath := common.ConfigPath
	common.ConfigPath = cfgPath
	t.Cleanup(func() { common.ConfigPath = oldPath })

	got := dbgIdentity(&dbgSnapView{Session: target.Session{Env: "恒烁正式区", N: 1, Prog: "apmt520"}, Zone: "1", TopentCf: "9999"})
	if !strings.Contains(got, "账号 dsdata") {
		t.Errorf("有快照时应给出企业 9999 的账号 dsdata:\n%s", got)
	}
	if strings.Contains(got, "dsdemo") {
		t.Errorf("不该串到别的企业的账号上:\n%s", got)
	}
}

// writeIdentityConfig 落一份"单环境挂库"的配置,返回其路径。
func writeIdentityConfig(t *testing.T) string {
	t.Helper()
	return writeIdentityConfigIn(t, t.TempDir())
}

// writeIdentityConfigIn 同上,但落在指定目录(企业快照的 DataDir 就是配置所在目录)。
func writeIdentityConfigIn(t *testing.T, dir string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"schemaVersion":3,"hosts":{"activeEnv":"恒烁正式区","sshs":[
	  {"name":"恒烁正式区","host":"172.19.93.55","port":22,"user":"tiptop","zone":"1","topent":"9999",
	   "db":{"type":"kingbase","host":"172.19.93.55","port":54321,"service":"topprd"}}
	]}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}
