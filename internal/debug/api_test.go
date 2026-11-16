package debug

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"tg/internal/host"
)

// launchCfg:点名的环境是**权威**的。
//
// 从前 /api/sessions 的 ssh 走 SSHByName:拼错了会**静默回落**当前连接,于是
// "环境名打错"与"确实连上了那个环境"在输出上无法区分 —— 同一台机器上有多个环境
// (同库不同企业)时,那就是上错号。这里钉住:认不出就报错并列出可用名。
func TestLaunchCfgResolvesByName(t *testing.T) {
	cfg := &Config{
		SSH:  host.SSHConfig{Host: "top", Port: 22, User: "u"},
		Zone: "36",
		SSHs: []host.NamedSsh{
			{Name: "恒烁正式区", SSHConfig: host.SSHConfig{Host: "h1", Port: 22, User: "u"}, Zone: "1"},
			{Name: "测试1012", SSHConfig: host.SSHConfig{Host: "h1", Port: 22, User: "u"}, Zone: "1"},
		},
	}
	cfg.fillDefaults()
	cfg.ApplyDefaultEnv()
	s := NewServer(cfg, nil, "")

	// 不点名:用当前生效配置(与从前一致)
	if got, err := s.launchCfg("", ""); err != nil || got != s.cfg {
		t.Fatalf("不点名应回当前配置: got=%v err=%v", got, err)
	}
	// 点名:换成该环境。
	// 注:企业编号不再从环境里取(hosts.sshs[].topent 已移除)—— 所以这里只断言环境名
	// 与连接换了;ENT 由调用方显式给(launchCfg 的 ent 参数 / --ent)。
	got, err := s.launchCfg("测试1012", "")
	if err != nil {
		t.Fatalf("点名存在的环境不该失败: %v", err)
	}
	if got.EnvName() != "测试1012" {
		t.Fatalf("点名 测试1012 应换成该环境: env=%q", got.EnvName())
	}
	if got.Topent != "" {
		t.Fatalf("没显式给 ENT 时不该从环境里继承一个,got %q", got.Topent)
	}
	if got.SSH.Host != "h1" {
		t.Fatalf("点名后应连该环境的机器, got %q", got.SSH.Host)
	}
	// 区域覆盖仍可用
	if z, err := s.launchCfg("测试1012", "35"); err != nil || z.Zone != "35" {
		t.Fatalf("zone 覆盖应生效: zone=%q err=%v", z.Zone, err)
	}
	// 点错了:报错并列出可用环境名(绝不静默回落)
	_, err = s.launchCfg("并不存在的环境", "")
	if err == nil {
		t.Fatal("点名不存在的环境必须报错(静默回落就是上错号)")
	}
	for _, want := range []string{"并不存在的环境", "恒烁正式区", "测试1012"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误信息里应出现 %q: %v", want, err)
		}
	}
}

// Listen 必须报出真实监听端口:--listen 端口写 0 时由系统分配,只能回读。
func TestListenReportsRealPort(t *testing.T) {
	s := NewServer(&Config{Listen: "127.0.0.1:0"}, nil, "")
	ln, addr, err := s.Listen()
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("监听到非 host:port 地址 %q: %v", addr, err)
	}
	if host != "127.0.0.1" || port == "0" {
		t.Fatalf("端口未回读(期望 127.0.0.1:<非0>): %s", addr)
	}
	if want := ln.Addr().String(); want != addr {
		t.Fatalf("上报地址应与真实监听一致: 上报 %s,实际 %s", addr, want)
	}
	if s.ListenAddr() != addr {
		t.Fatalf("ListenAddr 应为 %s,实际 %s", addr, s.ListenAddr())
	}

	// 该端口再起一个 → 自动顺延到**另一个**空闲端口(且报出的是顺延后的真实地址)
	//
	// 只钉"顺延了、落在契约范围内(最多往后 maxPortTries 个)、报的地址是真的",
	// **不钉正好 +1**:顺延到哪个端口取决于这台机器此刻哪些端口空着 —— 别的进程,
	// 甚至刚释放、还在 TIME_WAIT 的端口都算。钉死 +1 会在那种时候**假失败**
	// (实测本机单跑 5 次挂 2 次),因为那断言的是机器状态,不是代码行为。
	s2 := NewServer(&Config{Listen: addr}, nil, "")
	ln2, addr2, err := s2.Listen()
	if err != nil {
		t.Fatalf("顺延监听失败: %v", err)
	}
	defer ln2.Close()
	if got := ln2.Addr().String(); got != addr2 {
		t.Fatalf("顺延后上报地址应与真实监听一致: 上报 %s,实际 %s", addr2, got)
	}
	p1, _ := strconv.Atoi(port)
	_, port2, _ := net.SplitHostPort(addr2)
	p2, _ := strconv.Atoi(port2)
	if p2 <= p1 || p2 > p1+maxPortTries {
		t.Fatalf("端口 %d 被占用时应向后顺延(最多 %d 个),实际落到 %s", p1, maxPortTries, addr2)
	}
}

// LoadConfig 严格要求至少一个环境;LoadConfigAllowEmpty(服务启动用)允许为空。
// 配置是合并后的拆分结构:环境清单在顶层 hosts.sshs,监听地址在顶层 listen,
// 调试设置在 debug 节 —— 不再是从前那种 {debug:{sshs,listen,…}} 的合体节。
func TestLoadConfigAllowEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	fixture := `{"schemaVersion":2,"hosts":{"sshs":[]},"listen":"127.0.0.1:1234","debug":{"termWidth":123}}`
	if err := os.WriteFile(p, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(p); err == nil {
		t.Fatal("LoadConfig 应因 sshs 为空报错(CLI 语义不能松)")
	}
	cfg, err := LoadConfigAllowEmpty(p)
	if err != nil {
		t.Fatalf("LoadConfigAllowEmpty 应通过: %v", err)
	}
	if cfg.Listen != "127.0.0.1:1234" {
		t.Fatalf("listen 未从顶层 listen 读回: %s", cfg.Listen)
	}
	if cfg.TermWidth != 123 {
		t.Fatalf("debug 节的设置未读回: termWidth=%d", cfg.TermWidth)
	}
	if len(cfg.SSHs) != 0 {
		t.Fatalf("sshss 应为空: %d", len(cfg.SSHs))
	}
}

// 拆节后环境清单来自 hosts.sshs,默认环境取 hosts.activeEnv(可被 debug.activeEnv 覆盖)。
func TestLoadConfigSplitSections(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	fixture := `{
	  "schemaVersion": 2,
	  "hosts": {"activeEnv": "B", "sshs": [
	    {"name": "A", "host": "1.1.1.1", "port": 22, "user": "u"},
	    {"name": "B", "host": "2.2.2.2", "port": 22, "user": "u"}
	  ]},
	  "listen": "127.0.0.1:1234",
	  "debug": {"launchArgs": "X {prog}", "watchdogSeconds": 9}
	}`
	if err := os.WriteFile(p, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.SSHs) != 2 {
		t.Fatalf("环境清单应来自 hosts.sshs, got %d", len(cfg.SSHs))
	}
	if cfg.LaunchArgs != "X {prog}" || cfg.WatchdogSeconds != 9 {
		t.Fatalf("调试设置未读回: %+v", cfg)
	}
	// 原先这里断言 hosts.activeEnv / debug.activeEnv 决定"默认环境"以及两者的覆盖关系。
	// 两个字段都已移除(环境一律点名,见 config.Hosts 的注释),所以整段删掉。
	// 现在的等价保证:没有点名环境时**不挑任何环境**(ApplyDefaultEnv 清空运行时字段)。
	cfg.ApplyDefaultEnv()
	if cfg.envName != "" || cfg.SSH.Host != "" {
		t.Fatalf("没有点名环境时不该自动挑一个,got env=%q host=%q", cfg.envName, cfg.SSH.Host)
	}
}

// NewDefaultConfig 落盘后应能被宽松加载读回(首次运行的配置骨架)。
func TestNewDefaultConfigRoundTrip(t *testing.T) {
	cfg := NewDefaultConfig()
	if cfg.Listen == "" || cfg.LaunchArgs == "" || cfg.WatchdogSeconds == 0 ||
		cfg.TermWidth == 0 || cfg.TermHeight == 0 || cfg.PrintElements == 0 {
		t.Fatalf("默认值缺失: %+v", cfg)
	}
}

// POST /api/shutdown 取消 Serve 上下文(走这条优雅停止)。
func TestShutdownEndpoint(t *testing.T) {
	srv := NewServer(&Config{Listen: "127.0.0.1:0"}, nil, "")
	ln, addr, err := srv.Listen()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(context.Background(), ln) }()

	base := "http://" + addr
	if err := waitStatus(base, 3*time.Second); err != nil {
		t.Fatalf("服务未就绪: %v", err)
	}
	// 空 body:解析失败 → 400(与全站 readBody 语义一致)
	resp, err := http.Post(base+"/api/shutdown", "application/json", bytes.NewBufferString(""))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("空请求体应返回 400,实际 %d", resp.StatusCode)
	}
	resp, err = http.Post(base+"/api/shutdown", "application/json", bytes.NewBufferString("{}"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("shutdown 应返回 200,实际 %d", resp.StatusCode)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve 应正常返回,实际 %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve 未在 5s 内退出")
	}
}

// 未运行/未就绪的服务调 shutdown → 409。
func TestShutdownNotReady(t *testing.T) {
	srv := NewServer(&Config{}, nil, "")
	w := httptest.NewRecorder()
	srv.hShutdown(w, httptest.NewRequest("POST", "/api/shutdown", bytes.NewBufferString("{}")))
	if w.Code != http.StatusConflict {
		t.Fatalf("未就绪应返回 409,实际 %d", w.Code)
	}
}

func waitStatus(base string, d time.Duration) error {
	deadline := time.Now().Add(d)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/api/status")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
			lastErr = err
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	return lastErr
}

// ScreenTarget 在 TG 里恒为 GP 那一句：画面由 fgldb 接管，界面走 GDC 直连。
func TestScreenTargetSaysWhereTheScreenGoes(t *testing.T) {
	got := ScreenTarget()
	for _, want := range []string{"fgldb", "GDC"} {
		if !strings.Contains(got, want) {
			t.Errorf("画面去向里应出现 %q: %s", want, got)
		}
	}
}
