package debug

import (
	"strings"
	"testing"

	"tt/internal/host"
)

// ApplyDefaultEnv 在 sshs 被删空时不得 panic。
// 回归场景(合并前由 hSettingsPut 触发):设置页删光所有环境后保存,请求体 sshs 为空,
// 但服务端会把运行时 envName(非空)代入,修复前回落分支对空切片取下标越界。
// 环境清单的保存入口现在统一在 tt serve 的 PUT /api/hosts(它仍拦下空列表),
// 但配置文件被手改成空、或首次运行还没配时,这条兜底仍然必须成立。
func TestApplyDefaultEnvEmptySSHs(t *testing.T) {
	c := &Config{envName: "示例正式区", SSH: host.SSHConfig{Host: "10.0.0.2"}, Zone: "36"}
	c.ApplyDefaultEnv()
	if c.envName != "" {
		t.Fatalf("sshs 为空应清空当前环境,got %q", c.envName)
	}
	if c.SSH.Host != "" || c.Zone != "" {
		t.Fatalf("sshs 为空运行时字段应清空,got host=%q zone=%q", c.SSH.Host, c.Zone)
	}
	// EnvName() 会用 SSH.Host+Zone 兜底;残留会拼出幽灵环境名 "10.0.0.2-36"
	if got := c.EnvName(); strings.Contains(got, "10.0.0.2") {
		t.Fatalf("sshs 为空不应拼出幽灵环境名,got %q", got)
	}
}

// 当前环境已被删除时:**清空运行时字段,不再回落到首条**。
//
// 原先这里是"回落到首条(原有语义不变)"。默认环境移除之后这条回落变成了危险行为:
// 它会静默把会话切到另一台机器上,而输出看起来一切正常 —— 与"上错号"无法区分。
// 现在解不出环境就什么都不连(见 ApplyDefaultEnv)。
func TestApplyDefaultEnvClearsWhenEnvMissing(t *testing.T) {
	c := &Config{
		SSHs: []host.NamedSsh{
			{Name: "A", SSHConfig: host.SSHConfig{Host: "1.1.1.1"}},
			{Name: "B", SSHConfig: host.SSHConfig{Host: "2.2.2.2"}},
		},
		envName: "已删除的环境",
	}
	c.ApplyDefaultEnv()
	if c.envName != "" {
		t.Fatalf("环境不存在时应清空当前环境(不回落到首条),got %q", c.envName)
	}
	if c.SSH.Host != "" {
		t.Fatalf("环境不存在时不该合并任何 SSH(否则会连到别人机器上),got %q", c.SSH.Host)
	}
}

// envName 命中时应合并该环境的连接(原有语义不变)。
func TestApplyDefaultEnvMatchesByName(t *testing.T) {
	c := &Config{
		SSHs: []host.NamedSsh{
			{Name: "A", SSHConfig: host.SSHConfig{Host: "1.1.1.1"}},
			{Name: "B", SSHConfig: host.SSHConfig{Host: "2.2.2.2", Port: 2222}},
		},
		envName: "B",
	}
	c.ApplyDefaultEnv()
	if c.envName != "B" || c.SSH.Host != "2.2.2.2" || c.SSH.Port != 2222 {
		t.Fatalf("应按名合并环境 B,got %q %q:%d", c.envName, c.SSH.Host, c.SSH.Port)
	}
}
