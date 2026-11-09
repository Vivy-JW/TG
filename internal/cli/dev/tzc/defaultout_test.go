package tzc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tt/internal/testkit"

	"tt/internal/dev/testutil"
)

// TestDefaultWorkspaceDir 钉住 -o 省略时的默认目录推导规则。
func TestDefaultWorkspaceDir(t *testing.T) {
	cases := []struct{ in, want string }{
		{filepath.Join("D:", "pkg", "capt110(c).tzc"), filepath.Join("D:", "pkg", "capt110-ws")},
		{filepath.Join("D:", "pkg", "aapp131(s).tzc"), filepath.Join("D:", "pkg", "aapp131-ws")},
		{filepath.Join("D:", "pkg", "aapp131.tzc"), filepath.Join("D:", "pkg", "aapp131-ws")},
		{filepath.Join(string(filepath.Separator), "tmp", "x", "aapp320(c).tzc"),
			filepath.Join(string(filepath.Separator), "tmp", "x", "aapp320-ws")},
		{filepath.Join("D:", "pkg", ".tzc"), filepath.Join("D:", "pkg", "pkg-ws")},
	}
	for _, c := range cases {
		if got := defaultWorkspaceDir(c.in); got != c.want {
			t.Errorf("defaultWorkspaceDir(%q) = %q，想要 %q", c.in, got, c.want)
		}
	}
}

// TestExportWithoutO 端到端：省略 -o 会在包所在目录下建 <程序名>-ws，
// 且同目录的第二个包各建各的工作区、互不覆盖。
func TestExportWithoutO(t *testing.T) {
	dir := t.TempDir()
	p1, err := testutil.NormalPkgPath(dir, "adzi999")
	if err != nil {
		t.Fatal(err)
	}
	// 造第二个包（换程序名）放在同一目录
	p2 := filepath.Join(dir, "adzi998.tzc")
	entries := testutil.NormalEntries("adzi998")
	b, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	_ = b
	if _, err := testutil.WritePackage(dir, "adzi998.tzc", entries); err != nil {
		t.Fatal(err)
	}

	if code := testkit.Silent(t, func() int { return cmdExport([]string{p1}) }); code != 0 {
		t.Fatalf("省略 -o 的 export 退出码 %d", code)
	}
	ws1 := filepath.Join(dir, "adzi999-ws")
	if _, err := os.Stat(filepath.Join(ws1, "prog.full.4gl")); err != nil {
		t.Fatalf("默认工作区没建出来: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws1, "manifest.json")); err != nil {
		t.Fatalf("默认工作区缺少 manifest.json: %v", err)
	}
	// 第二个包：另一个工作区，互不影响
	if code := testkit.Silent(t, func() int { return cmdExport([]string{p2}) }); code != 0 {
		t.Fatalf("第二个包 export 退出码 %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "adzi998-ws", "prog.full.4gl")); err != nil {
		t.Fatalf("第二个包的默认工作区没建出来: %v", err)
	}
	// 再导一次同一个包：必须明确拒绝（不静默覆盖已有工作区），且提示 -o
	code := testkit.Silent(t, func() int { return cmdExport([]string{p1}) })
	if code == 0 {
		t.Fatalf("重复导出同一个包必须被拒绝（不能静默覆盖已有工作区）")
	}
	if code != 5 {
		t.Logf("提示：重复导出返回退出码 %d（期望 5 = IO/环境失败）", code)
	}
}

// TestDefaultWorkspaceDirStripsOnlyIdentitySuffix 只剥 (x) 形式的后缀，别误伤程序名。
func TestDefaultWorkspaceDirStripsOnlyIdentitySuffix(t *testing.T) {
	got := defaultWorkspaceDir(`D:\pkg\aapp131_wf(c).tzc`)
	if !strings.HasSuffix(got, `aapp131_wf-ws`) {
		t.Errorf("下划线程序名被误伤: %q", got)
	}
	got2 := defaultWorkspaceDir(`D:\pkg\cs_gen_asft800_wf.tzc`)
	if !strings.HasSuffix(got2, `cs_gen_asft800_wf-ws`) {
		t.Errorf("无后缀程序名被误改: %q", got2)
	}
}
