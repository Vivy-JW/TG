package debug

// `tg debug source` 出错时要给"怎么办"：命中已知失败必须给出可复制粘贴的下一步命令，
// 别的错误不许被套上这套提示（套错比不套更误导）。

import (
	"strings"
	"testing"
)

// TestSourceDirHowToIsActionable 命中已知失败 ⇒ 给出可复制粘贴的下一步。
func TestSourceDirHowToIsActionable(t *testing.T) {
	got := sourceDirHowTo("环境 22主机 没有配 $TOP（workspace）：去设置页「SSH 服务器」填上")
	if len(got) == 0 {
		t.Fatal("命中这条失败时不该返回空提示")
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"tg env show", "$TOP", "$CUST", "不猜路径"} {
		if !strings.Contains(joined, want) {
			t.Errorf("提示里缺 %q：%s", want, joined)
		}
	}
}

// TestSourceDirHowToGuidesMissingFile 文件没找到时给的是 p_link 方向的路，而不是配置方向。
func TestSourceDirHowToGuidesMissingFile(t *testing.T) {
	got := sourceDirHowTo(`源码未找到(cbmq100.4gl)：按 p_link 解析出的模块 [cbm] 在 $TOP/$CUST 的 <模块>/4gl 下都没找到`)
	if len(got) == 0 {
		t.Fatal("源码未找到时该给提示")
	}
	joined := strings.Join(got, "\n")
	for _, want := range []string{"p_link", "tg dict db status", "tg file ls"} {
		if !strings.Contains(joined, want) {
			t.Errorf("提示里缺 %q：%s", want, joined)
		}
	}
}

// TestSourceDirHowToIgnoresOtherErrors 别的错误不许被套上这套提示（套错比不套更误导）。
func TestSourceDirHowToIgnoresOtherErrors(t *testing.T) {
	for _, e := range []string{
		"路径不在 $TOP / $CUST 之内: /etc/passwd",
		"需要 file 或 path 参数",
		"",
	} {
		if got := sourceDirHowTo(e); got != nil {
			t.Errorf("%q 不该命中源码提示，得 %v", e, got)
		}
	}
}
