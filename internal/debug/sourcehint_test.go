package debug

// 缺陷 #2（.agents/plan/L1-结果-2026-10-18.md §7-2）：
// `tt debug source` 只回「登录区源码目录未知(环境路径解析失败)」。
//
// A15（不猜）：ModuleRootsActual 只认登录后回读的 $TOP/$ERP/$COM，静态配置里不许有，
// 所以取不到时**不回退默认目录**，而是把"取不到"与三条可操作的路一起说清楚。
// 本文件只断言文案里的可操作内容：不连线、不读盘。

import (
	"strings"
	"testing"
)

// TestSourceRootsUnknownErrIsActionable 错误里必须给出：取不到（说清是取不到，不是别的原因）、
// 怎么让它有值（起会话回读）、以及源码目录到底长什么样。
func TestSourceRootsUnknownErrIsActionable(t *testing.T) {
	err := sourceRootsUnknownErr(nil)
	if err == nil {
		t.Fatal("该返回错误")
	}
	msg := err.Error()
	for _, want := range []string{
		"登录区源码目录未知",
		"取不到",            // A15：明确说取不到
		"tt debug start", // 怎么让它有值
		"$ERP",           // 源码目录口径
		"tt debug env",   // 已经有值时的自查
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误文案缺 %q（不可操作就无法排障）：%v", want, msg)
		}
	}
	// 不许偷偷给一个"默认目录"当退路：那会把"读错机器上的文件"伪装成成功。
	if strings.Contains(msg, "/u1/") {
		t.Errorf("不许在文案里出现具体默认路径（不猜）：%v", msg)
	}
}

// TestSourceRootsUnknownErrNamesEnv 点了名就带上环境名（多环境时"哪个环境没值"是第一个问题）。
func TestSourceRootsUnknownErrNamesEnv(t *testing.T) {
	c := &Config{envName: "主机109"}
	msg := sourceRootsUnknownErr(c).Error()
	if !strings.Contains(msg, "主机109") {
		t.Errorf("该点出当前环境名：%v", msg)
	}
}
