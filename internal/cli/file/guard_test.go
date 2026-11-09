package file

// 围栏的**源码级**守卫：
//  1. cli/file 不许 import os/exec 或 pkg/sftp —— 前者等于“能不能执行命令”，
//     后者等于“能不能绕过 host.FS 直接写远端”；
//  2. host/fsops.go（只读原语）不许出现写方法名。
//
// 只读不靠 review 记忆，靠这条测试变红。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilePackageHasNoExecOrSftp(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		s := string(b)
		for _, bad := range []string{`"os/exec"`, `"github.com/pkg/sftp"`} {
			if strings.Contains(s, bad) {
				t.Errorf("%s 引入了 %s —— tt file 必须只经 host.FS 只读访问，且不执行服务器命令", name, bad)
			}
		}
	}
}

func TestFSOpsHasNoRemoteWrites(t *testing.T) {
	p := filepath.Join("..", "..", "host", "fsops.go")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, bad := range []string{".Create(", ".Remove(", ".RemoveAll(", ".Rename(", ".Mkdir(", ".MkdirAll(", ".Chmod(", ".Truncate(", ".Write("} {
		if strings.Contains(s, bad) {
			t.Errorf("host/fsops.go 出现了写操作 %s —— 只读原语里不许有写", bad)
		}
	}
}
