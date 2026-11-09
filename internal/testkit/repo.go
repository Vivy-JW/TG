package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

// RepoRoot 从当前包目录逐级上溯，返回含 go.mod 的那一级（仓库根）。
//
// 测试的工作目录是**包目录**，而"仓库本体"的东西（根 README、夹具目录、别的包的源码）
// 是按仓库根定位的。上溯比硬编码 `../../../` 稳：包挪了层级不用跟着改。
func RepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("取不到工作目录: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("从 %s 逐级上溯都没找到 go.mod —— 仓库布局变了，或者测试不在仓库里跑", dir)
		}
		dir = parent
	}
}
