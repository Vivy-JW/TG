package tzs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/dev/testutil"
	"tt/internal/testkit"
)

// formPkg 造一个合成的表单包（.tzs）：纯文本 + 二进制各一，用于验证「纯解压逐字节一致」。
func formPkg(t *testing.T, dir, name string) (string, map[string][]byte) {
	t.Helper()
	entries := map[string]string{
		"adzi999.tsd":     "<form spec=\"x\">\r\n  <field/>\r\n</form>\r\n",
		"adzi999.4fd":     "BIN\x00\x01\x02\xff\r\n",
		"adzi999.4fd.ref": "",
		"ver":             "1.0\r\n",
	}
	p, err := testutil.WritePackage(dir, name, entries)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][]byte{}
	for k, v := range entries {
		want[k] = []byte(v)
	}
	return p, want
}

// TestTzsExportUnzipsVerbatim 纯解压：逐字节一致，且绝不产生工作区/审计产物。
func TestTzsExportUnzipsVerbatim(t *testing.T) {
	dir := t.TempDir()
	p, want := formPkg(t, filepath.Join(dir, "src"), "adzi999.tzs")
	out := filepath.Join(dir, "out")

	if code := testkit.Silent(t, func() int { return cmdTzs([]string{"export", p, "-o", out}, "") }); code != 0 {
		t.Fatalf("tzs export 退出码 %d", code)
	}
	for name, wb := range want {
		got, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("缺文件 %s: %v", name, err)
		}
		if string(got) != string(wb) {
			t.Errorf("%s 内容不一致：得 %q 想 %q", name, got, wb)
		}
	}
	// 纯解压 = 没有工作区/审计/渲染产物
	for _, forbidden := range []string{".tdev", "manifest.json", ".git", "prog.full.4gl", "snapshot", "regions.json"} {
		if _, err := os.Stat(filepath.Join(out, forbidden)); err == nil {
			t.Errorf("纯解压不该产生 %s", forbidden)
		}
	}
}

// TestTzsExportDefaultDir 默认目录 = <包目录>/<程序名>-unzip（身份后缀去掉）。
func TestTzsExportDefaultDir(t *testing.T) {
	cases := []struct{ in, want string }{
		{filepath.Join("D:", "pkg", "aapp320(c).tzs"), filepath.Join("D:", "pkg", "aapp320-unzip")},
		{filepath.Join("D:", "pkg", "capt110(s).tzs"), filepath.Join("D:", "pkg", "capt110-unzip")},
		{filepath.Join("D:", "pkg", "plain.tzv"), filepath.Join("D:", "pkg", "plain-unzip")},
		{filepath.Join("D:", "pkg", ".tzs"), filepath.Join("D:", "pkg", "pkg-unzip")},
	}
	for _, c := range cases {
		if got := defaultUnzipDir(c.in); got != c.want {
			t.Errorf("defaultUnzipDir(%q) = %q，想要 %q", c.in, got, c.want)
		}
	}
}

// TestTzsExportRefusesWrongKind 用错管线要当场指回 tzc（这类简单错误必须被挡住）。
func TestTzsExportRefusesWrongKind(t *testing.T) {
	dir := t.TempDir()
	code, err := testutil.NormalPkgPath(dir, "adzi999")
	if err != nil {
		t.Fatal(err)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", code, "-o", filepath.Join(dir, "a")}, "") }); got != 2 {
		t.Errorf("代码包走 tzs 应退出 2，实际 %d", got)
	}
	// 非包扩展名
	junk := filepath.Join(dir, "x.zip")
	if err := os.WriteFile(junk, []byte("not a package"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", junk, "-o", filepath.Join(dir, "b")}, "") }); got != 2 {
		t.Errorf("未知扩展名应退出 2，实际 %d", got)
	}
	// 用法错误
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export"}, "") }); got != 2 {
		t.Errorf("缺参数应退出 2，实际 %d", got)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"apply"}, "") }); got != 2 {
		t.Errorf("tzs 没有 apply 动词，应退出 2，实际 %d", got)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs(nil, "") }); got != 2 {
		t.Errorf("无子命令应退出 2，实际 %d", got)
	}
}

// TestTzsExportNonEmptyTarget 目标非空 → 拒绝（5）；--force → 覆盖（0）。
func TestTzsExportNonEmptyTarget(t *testing.T) {
	dir := t.TempDir()
	p, want := formPkg(t, filepath.Join(dir, "src"), "adzi999.tzs")
	out := filepath.Join(dir, "out")
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", p, "-o", out}, "") }); got != 0 {
		t.Fatalf("首次解压退出码 %d", got)
	}
	// 改脏一个文件，再解一次：默认拒绝，--force 覆盖回原样
	target := filepath.Join(out, "ver")
	if err := os.WriteFile(target, []byte("脏了"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", p, "-o", out}, "") }); got != 5 {
		t.Errorf("目标非空应退出 5，实际 %d", got)
	}
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", p, "-o", out, "--force"}, "") }); got != 0 {
		t.Errorf("--force 应退出 0，实际 %d", got)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want["ver"]) {
		t.Errorf("--force 没把文件覆盖回原样：%q", got)
	}
}

// TestTzsExportRejectsZipSlip 条目名带 ../ 的包必须整体拒绝，且不产生越界文件。
func TestTzsExportRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	p, err := testutil.WritePackage(filepath.Join(dir, "src"), "evil.tzs", map[string]string{
		"ok.tsd":         "fine",
		"../escaped.txt": "should not escape",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if got := testkit.Silent(t, func() int { return cmdTzs([]string{"export", p, "-o", out}, "") }); got != 2 {
		t.Errorf("zip-slip 包应退出 2，实际 %d", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "escaped.txt")); err == nil {
		t.Errorf("有文件逃出目标目录：%s", filepath.Join(dir, "escaped.txt"))
	}
}

// TestTzsExportJSON --json 的形状（AI 直接消费）。
func TestTzsExportJSON(t *testing.T) {
	dir := t.TempDir()
	p, _ := formPkg(t, filepath.Join(dir, "src"), "adzi999.tzs")
	out := filepath.Join(dir, "out")
	code, text := testkit.CaptureStdout(t, func() int {
		return cmdTzs([]string{"export", p, "-o", out, "--json"}, "")
	})
	if code != 0 {
		t.Fatalf("--json 退出码 %d", code)
	}
	for _, want := range []string{`"ok":true`, `"readonly":true`, `"count":4`, `"entries"`, `adzi999.tsd`} {
		if !strings.Contains(text, want) {
			t.Errorf("JSON 里缺 %s：\n%s", want, text)
		}
	}
}
