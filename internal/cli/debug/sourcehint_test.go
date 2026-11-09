package debug

// 现场缺陷 #2：`tt debug source` 只回"登录区源码目录未知(环境路径解析失败)"。
// 这里断言**可操作性**：命中这条失败时必须能给出具体的下一步命令。

import (
	"strings"
	"testing"
)

// TestSourceDirHowToIsActionable 命中已知失败 ⇒ 给出可复制粘贴的下一步。
func TestSourceDirHowToIsActionable(t *testing.T) {
	got := sourceDirHowTo("登录区源码目录未知(环境路径解析失败)")
	if len(got) == 0 {
		t.Fatal("命中这条失败时不该返回空提示")
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"tt debug start", "tt debug env", "$ERP", "不猜"} {
		if !strings.Contains(joined, want) {
			t.Errorf("提示里缺 %q：%s", want, joined)
		}
	}
}

// TestSourceDirHowToIgnoresOtherErrors 别的错误不许被套上这套提示（套错比不套更误导）。
func TestSourceDirHowToIgnoresOtherErrors(t *testing.T) {
	for _, e := range []string{
		"源码未找到(axm_axmt500_wf.4gl): file does not exist",
		"路径不在登录区源码目录内: /etc/passwd",
		"",
	} {
		if got := sourceDirHowTo(e); got != nil {
			t.Errorf("%q 不该命中源码目录提示，得 %v", e, got)
		}
	}
}
