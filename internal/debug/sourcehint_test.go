package debug

// `tg debug source` 的源码根取不到时的文案。
//
// GP 的源码根就是环境里配的 $TOP / $CUST（config.ModuleRootsActual），
// 取不到时**不回退默认目录**（那会把"读错机器上的文件"伪装成成功），
// 而是说清"哪一步没配 + 去哪里配"。
// 本文件只断言文案里的可操作内容：不连线、不读盘。

import (
	"strings"
	"testing"
)

// TestSourceRootsUnknownErrIsActionable 错误里必须给出：没有 $TOP、去哪配、$TOP 长什么样。
func TestSourceRootsUnknownErrIsActionable(t *testing.T) {
	err := sourceRootsUnknownErr(nil)
	if err == nil {
		t.Fatal("该返回错误")
	}
	msg := err.Error()
	for _, want := range []string{
		"$TOP",
		"workspace",
		"设置页",
		"<实例>",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("错误文案缺 %q（不可操作就无法排障）：%v", want, msg)
		}
	}
	// 不许出现某个具体实例的默认路径（那等于猜一台机器）。
	for _, bad := range []string{"/u1/toptest", "/u1/t35prd"} {
		if strings.Contains(msg, bad) {
			t.Errorf("不许在文案里出现具体实例路径（不猜）：%v", msg)
		}
	}
}

// TestSourceRootsUnknownErrNamesEnv 点了名就带上环境名（多环境时"哪个环境没值"是第一个问题）。
func TestSourceRootsUnknownErrNamesEnv(t *testing.T) {
	c := &Config{envName: "22主机"}
	msg := sourceRootsUnknownErr(c).Error()
	if !strings.Contains(msg, "22主机") {
		t.Errorf("该点出当前环境名：%v", msg)
	}
}
