package config

// 本文件钉住原子写的**失败路径**:出错时不留半截文件、不留 .tmp 残渣。
//
// 为什么值得单独测:实测(2026-10-15)在 Windows 上,只要目标 config.json 那一刻
// 被另一个句柄占着,os.Rename 就报 `Access is denied`。于是保存失败的现场是
// "用户按了保存 / 删了环境,却得到一个看不懂的拒绝访问",同时目录里可能留下一堆
// config.json.tmpNNN —— 下次再写又要靠 CreateTemp 随机名绕开它们,越积越多。
// 所以这里断言两件事:失败要报错(不能假装成功),以及**不留下残渣**。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 目标文件被占用时:AtomicWrite 必须失败,且不留下 .tmp 残渣。
//
// 用 os.Open 持有的句柄就足以让 Go 的 os.Rename 在 Windows 上拿到
// ERROR_ACCESS_DENIED —— 这正是真实场景(杀软/索引器/另一个读者)的形态。
func TestAtomicWriteLeavesNoLitterWhenDestinationBusy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	orig := []byte(`{"schemaVersion": 3}`)
	if err := os.WriteFile(path, orig, 0o600); err != nil {
		t.Fatal(err)
	}

	// 占住目标文件
	hold, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	err = AtomicWrite(path, []byte(`{"schemaVersion": 3, "changed": true}`))
	_ = hold.Close()

	if err == nil {
		// 值得单独说一句:如果哪天 Windows 允许了这种替换,这条测试会红 ——
		// 那时应当把"被占用时重试"的策略与这条断言一起重新审一遍,而不是删掉它。
		t.Fatal("目标被占用时应当失败,而不是默默成功")
	}
	if !strings.Contains(err.Error(), "Access is denied") &&
		!strings.Contains(err.Error(), "拒绝访问") &&
		!strings.Contains(err.Error(), "更新配置文件失败") {
		t.Errorf("错误信息应当指向替换失败那一步,得到: %v", err)
	}

	// 原文件必须完好(没被写坏)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(orig) {
		t.Errorf("失败时改动了原文件: %s", got)
	}

	// **不留残渣**:目录里不该有 .tmp 文件
	assertNoTmpLeft(t, dir)
}

// 成功路径同样不留残渣(与上面那条一起把"清理"这件事钉死)。
func TestAtomicWriteLeavesNoTmpOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		if err := AtomicWrite(path, []byte(`{"a":2}`)); err != nil {
			t.Fatalf("第 %d 轮失败: %v", i, err)
		}
	}
	assertNoTmpLeft(t, dir)
}

// 重命名替换在目标空闲时必须是**一次就成功**的:重试只是兜瞬时占用,
// 不该把正常路径也拖慢(拖慢的话每次保存都要多等几十到几百毫秒)。
func TestAtomicWriteFastPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(path, []byte(`{"a":2}`)); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.TrimSpace(string(got)) != `{"a":2}` {
		t.Errorf("内容不对: %s", got)
	}
}

func assertNoTmpLeft(t *testing.T, dir string) {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("留下了临时文件残渣: %s", e.Name())
		}
	}
}
