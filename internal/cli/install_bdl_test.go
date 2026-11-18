package cli

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"tg/internal/config"
	"tg/internal/output"
	"tg/internal/testkit"
)

// makeBDLZip 造一份"像发版包那样"的文档 zip：一个顶层目录 + 若干 md。
func makeBDLZip(t *testing.T, path string, files map[string]string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, cerr := zw.Create("BDL-Markdown/" + name)
		if cerr != nil {
			t.Fatal(cerr)
		}
		if _, werr := w.Write([]byte(body)); werr != nil {
			t.Fatal(werr)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// runInstallBDL 跑一次命令，返回退出码与 stdout。
func runInstallBDL(t *testing.T, args ...string) (int, string) {
	t.Helper()
	return testkit.CaptureStdout(t, func() int {
		cmd := newInstallBDLCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Errorf("install bdl %v: %v", args, err)
			return 1
		}
		return 0
	})
}

// 落点三级：--to > config 的 bdldoc.dir > <数据目录>/BDL-Markdown。
func TestResolveBDLDirPrecedence(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":5}`)
	dir, why, err := resolveBDLDir("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dataDir, config.BDLDocDirName); dir != want {
		t.Fatalf("缺省落点 = %q, want %q", dir, want)
	}
	if why == "" {
		t.Fatal("缺省落点应当说明'为什么是它'")
	}

	// 配置里写了 bdldoc.dir：用它，并说明来源。
	writeConfig(t, `{"schemaVersion":5,"bdldoc":{"dir":"D:/docs/bdl"}}`)
	dir, why, err = resolveBDLDir("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Clean("D:/docs/bdl"); dir != want {
		t.Fatalf("配置落点 = %q, want %q", dir, want)
	}
	if why == "" {
		t.Fatal("配置落点应当说明来源")
	}

	// --to 压过配置，且不再解释为什么（显式给了就不需要解释）。
	dir, why, err = resolveBDLDir("rel-docs")
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if want := filepath.Join(wd, "rel-docs"); dir != want {
		t.Fatalf("--to 落点 = %q, want %q", dir, want)
	}
	if why != "" {
		t.Fatalf("--to 不该带 why，got %q", why)
	}
}

// 文档包：--from 优先，否则 exe 旁边的随包 zip。
func TestResolveBDLZip(t *testing.T) {
	got, err := resolveBDLZip("some/dir/docs.zip")
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if want := filepath.Join(wd, "some", "dir", "docs.zip"); got != want {
		t.Fatalf("--from = %q, want %q", got, want)
	}

	old := bdlZipSource
	bdlZipSource = func() (string, error) { return `C:\app\BDL-Markdown.zip`, nil }
	defer func() { bdlZipSource = old }()
	if got, err = resolveBDLZip(""); err != nil {
		t.Fatal(err)
	}
	if want := `C:\app\BDL-Markdown.zip`; got != want {
		t.Fatalf("缺省 zip = %q, want %q", got, want)
	}
}

// 首次安装：把 zip 里的东西解到目标，报文件数。
func TestInstallBDLFresh(t *testing.T) {
	writeConfig(t, `{"schemaVersion":5}`)
	tmp := t.TempDir()
	src := makeBDLZip(t, filepath.Join(tmp, "bdl.zip"), map[string]string{
		"01_guide/0001-intro.md":  "# intro\n",
		"01_guide/0002-syntax.md": "# syntax\n",
		"02_db/0001-sql.md":       "# sql\n",
	})
	dst := filepath.Join(tmp, "docs")

	code, out := runInstallBDL(t, "--from", src, "--to", dst)
	if code != 0 {
		t.Fatalf("退出码 = %d\n%s", code, out)
	}
	var rep struct {
		OK      bool `json:"ok"`
		Files   int  `json:"files"`
		Skipped bool `json:"skipped"`
		DryRun  bool `json:"dryRun"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !rep.OK || rep.Files != 3 || rep.Skipped || rep.DryRun {
		t.Fatalf("报告不对: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dst, "01_guide", "0002-syntax.md")); err != nil {
		t.Fatalf("没解出文件: %v", err)
	}
}

// 目标非空：默认跳过，一个字节都不写。
func TestInstallBDLSkipsExisting(t *testing.T) {
	writeConfig(t, `{"schemaVersion":5}`)
	tmp := t.TempDir()
	src := makeBDLZip(t, filepath.Join(tmp, "bdl.zip"), map[string]string{"01/a.md": "# new\n"})
	dst := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "mine.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runInstallBDL(t, "--from", src, "--to", dst)
	if code != 0 {
		t.Fatalf("跳过应当是成功退出，got %d\n%s", code, out)
	}
	var rep struct {
		Skipped bool `json:"skipped"`
		Files   int  `json:"files"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !rep.Skipped || rep.Files != 0 {
		t.Fatalf("报告不对: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dst, "01", "a.md")); !os.IsNotExist(err) {
		t.Fatal("跳过时不该往目标里写文件")
	}
}

// --force：覆盖同名文件，但**不删**目标里多出来的东西。
func TestInstallBDLForceOverwritesAndKeepsExtras(t *testing.T) {
	writeConfig(t, `{"schemaVersion":5}`)
	tmp := t.TempDir()
	src := makeBDLZip(t, filepath.Join(tmp, "bdl.zip"), map[string]string{"01/a.md": "# new\n"})
	dst := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(filepath.Join(dst, "01"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "01", "a.md"), []byte("# old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "notes.md"), []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out := runInstallBDL(t, "--from", src, "--to", dst, "--force")
	if code != 0 {
		t.Fatalf("退出码 = %d\n%s", code, out)
	}
	got, err := os.ReadFile(filepath.Join(dst, "01", "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "# new\n" {
		t.Fatalf("同名文件没被覆盖: %q", got)
	}
	if _, err := os.Stat(filepath.Join(dst, "notes.md")); err != nil {
		t.Fatalf("--force 不该删掉目标里多出来的文件: %v", err)
	}
}

// --dry-run：只回答"会解到哪"，不写盘也不改配置。
func TestInstallBDLDryRunWritesNothing(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":5}`)
	tmp := t.TempDir()
	src := makeBDLZip(t, filepath.Join(tmp, "bdl.zip"), map[string]string{"01/a.md": "# x\n"})
	dst := filepath.Join(tmp, "docs")

	code, out := runInstallBDL(t, "--from", src, "--to", dst, "--dry-run")
	if code != 0 {
		t.Fatalf("退出码 = %d\n%s", code, out)
	}
	var rep struct {
		DryRun bool   `json:"dryRun"`
		Dir    string `json:"dir"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !rep.DryRun || rep.Dir != dst {
		t.Fatalf("报告不对: %+v", rep)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("--dry-run 不该创建目标目录")
	}
	// --to 只对这一次生效：不写回配置。
	raw, err := os.ReadFile(filepath.Join(dataDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		t.Fatal(err)
	}
	if bd, ok := root["bdldoc"].(map[string]any); ok && bd["dir"] != "" && bd["dir"] != nil {
		t.Fatalf("--to 不该写回 bdldoc.dir: %v", bd["dir"])
	}
}

// 随包 zip 不在：稳定的 NOT_FOUND，而不是把解压栈抛给用户。
func TestInstallBDLMissingZip(t *testing.T) {
	writeConfig(t, `{"schemaVersion":5}`)
	old := bdlZipSource
	bdlZipSource = func() (string, error) { return filepath.Join(t.TempDir(), "nope.zip"), nil }
	defer func() { bdlZipSource = old }()

	err := newInstallBDLCmd().Execute()
	if err == nil {
		t.Fatal("缺 zip 应当报错")
	}
	var oe *output.Error
	if !errors.As(err, &oe) {
		t.Fatalf("错误类型 = %T, want *output.Error", err)
	}
	if oe.Code != output.CodeNotFound {
		t.Fatalf("code = %q, want %q", oe.Code, output.CodeNotFound)
	}
}
