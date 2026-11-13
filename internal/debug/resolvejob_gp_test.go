package debug

import (
	"strings"
	"testing"
)

// GP 的作业解析只看本地字典：没有字典时不连库、不报错，给一句话说明怎么办。
func TestResolveJobGPWithoutDict(t *testing.T) {
	m := &Manager{cfg: &Config{Product: "gp"}}
	j := m.ResolveJob("", "p_zta")
	if j.Prog != "p_zta" || j.RunProg != "p_zta" {
		t.Fatalf("GP 的作业就是程序本身，期望 RunProg=p_zta，得到 %+v", j)
	}
	if j.Module != "" {
		t.Fatalf("没有字典时模块应为空，得到 %q", j.Module)
	}
	if !strings.Contains(j.Note, "tg dict db sync") {
		t.Fatalf("note 应指路到字典同步，得到 %q", j.Note)
	}
}

// 用户显式给模块时保留它（小写），并说明"本地没有记录、用你给的"。
func TestResolveJobGPExplicitModule(t *testing.T) {
	m := &Manager{cfg: &Config{Product: "gp"}}
	j := m.ResolveJob("CBM", "p_zta")
	if j.Module != "cbm" {
		t.Fatalf("显式模块应小写保留，得到 %q", j.Module)
	}
	if !strings.Contains(j.Note, "用你给的模块") {
		t.Fatalf("note 应说明用的是显式模块，得到 %q", j.Note)
	}
}

// 非标准标识符不进字典查询。
func TestResolveJobGPBadName(t *testing.T) {
	m := &Manager{cfg: &Config{Product: "gp"}}
	j := m.ResolveJob("", "1bad-name")
	if j.Module != "" || !strings.Contains(j.Note, "不是标准标识符") {
		t.Fatalf("非法作业名应跳过解析，得到 %+v", j)
	}
}
