package debug

// 这组测试盯的是"点名了环境就必须按它连"这条线。
//
// 从前 `tt debug start … --env X` 既不报错也不生效(--env 是根命令的开关,debug 组里
// 没人读它),方向永远取服务端当前环境;同机多环境(同库不同企业)时那就是"上错号",
// 所有按企业编号取数的查询返回 0 行,看起来与"配置缺失"一模一样。这些用例把三件事钉住:
// 点名怎么解析、生效身份怎么表述、不一致时怎么报。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tg/internal/cli/common"
	"tg/internal/target"
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
	t.Setenv("TG_CONFIG", "") // 隔离真实配置
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := `{"schemaVersion":4,"hosts":{"activeEnv":"22主机","sshs":[
	  {"name":"22主机","host":"172.16.1.22","port":22,"user":"tiptop","product":"gp","instance":"toptest",
	   "db":{"type":"oracle","host":"172.16.1.22","port":1521,"service":"toptest",
	         "accounts":[{"account":"digiwin","password":"x"}]}}]}}`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	oldPath := common.ConfigPath
	common.ConfigPath = cfgPath
	t.Cleanup(func() { common.ConfigPath = oldPath })

	got := dbgIdentity(&dbgSnapView{
		Session: target.Session{Env: "22主机", N: 1, Prog: "p_zta", State: "stopped"},
		Reused:  true,
	})
	// TG 只跑 GP：身份行没有 ENT 那一段，第二行也没有"区域"。
	if strings.Contains(got, "ENT") {
		t.Errorf("GP 身份行不该再打 ENT：\n%s", got)
	}
	if strings.Contains(got, "区域") {
		t.Errorf("GP 第二行不该再有区域：\n%s", got)
	}
	for _, want := range []string{
		"会话 22主机/p_zta#1 · (未命名) · 已停站",
		"账号 digiwin", "库 172.16.1.22:1521/toptest", "复用了已登录的宿主",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("身份行缺少 %q:\n%s", want, got)
		}
	}
	if dbgIdentity(nil) != "" {
		t.Error("没有快照时应返回空串（不能打出半行）")
	}
}
