package update

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 技能戳的两个动作：记一次（幂等）、按版本判断该不该刷。
func TestSkillsStampRecordAndStale(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "agent-skills")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := RecordSkillsTarget(dir, target, "0.2.1"); err != nil {
		t.Fatalf("RecordSkillsTarget: %v", err)
	}
	st := LoadSkillsStamp(dir)
	if st == nil || st.Version != "0.2.1" || len(st.Targets) != 1 {
		t.Fatalf("戳没写对: %+v", st)
	}
	if !filepath.IsAbs(st.Targets[0]) {
		t.Errorf("目标该记成绝对路径（升级时的 cwd 未必与安装时相同）: %q", st.Targets[0])
	}

	// 幂等：同一个目录再来一次（哪怕写法不同）不该多出一条。
	if err := RecordSkillsTarget(dir, target, "0.2.1"); err != nil {
		t.Fatal(err)
	}
	if got := len(LoadSkillsStamp(dir).Targets); got != 1 {
		t.Errorf("重复记录产生了 %d 条目标，想要 1 条", got)
	}

	// 同版本 → 不用刷。
	if got := StaleSkillsTargets(dir, "0.2.1"); len(got) != 0 {
		t.Errorf("同版本不该给刷新目标，得到 %v", got)
	}
	// 版本变了 → 要刷。
	got := StaleSkillsTargets(dir, "0.2.2")
	if len(got) != 1 || got[0] != st.Targets[0] {
		t.Errorf("版本变了该给目标，得到 %v", got)
	}
	// 目标目录已经不在了（用户删了）→ 不该给（白跑一趟还会报错）。
	if err := os.RemoveAll(target); err != nil {
		t.Fatal(err)
	}
	if got := StaleSkillsTargets(dir, "0.2.2"); len(got) != 0 {
		t.Errorf("目标没了就不该刷，得到 %v", got)
	}
}

func TestSkillsStampMissingAndBroken(t *testing.T) {
	dir := t.TempDir()
	if LoadSkillsStamp(dir) != nil {
		t.Error("没有戳时该返回 nil")
	}
	if got := StaleSkillsTargets(dir, "0.2.2"); got != nil {
		t.Errorf("没有戳时没有可刷的目标，得到 %v", got)
	}
	if got := SkillsDriftHint(dir, "0.2.2"); got != "" {
		t.Errorf("没有戳时不该提示，得到 %q", got)
	}
	if err := os.WriteFile(SkillsStampPath(dir), []byte("坏 JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if LoadSkillsStamp(dir) != nil {
		t.Error("坏掉的戳该当作没有")
	}
	// 数据目录不可用（空串）时：记录要报错，读要返回空 —— 不该炸。
	if err := RecordSkillsTarget("", "x", "1.0.0"); err == nil {
		t.Error("数据目录为空时记录该报错")
	}
	if LoadSkillsStamp("") != nil || SkillsDriftHint("", "1.0.0") != "" {
		t.Error("数据目录为空时读该得到空")
	}
}

// 漂移提示要说清"是谁装的、装了几处、该敲什么"。
func TestSkillsDriftHint(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "s")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := RecordSkillsTarget(dir, target, "0.2.1"); err != nil {
		t.Fatal(err)
	}
	if got := SkillsDriftHint(dir, "0.2.1"); got != "" {
		t.Errorf("同版不该提示，得到 %q", got)
	}
	got := SkillsDriftHint(dir, "0.2.2")
	for _, want := range []string{"0.2.1", "1 处", "tt install skills --force"} {
		if !strings.Contains(got, want) {
			t.Errorf("提示里该有 %q: %q", want, got)
		}
	}
}
