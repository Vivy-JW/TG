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

// makeGPDocZip 造一份"像发版包那样"的 GP 手册 zip：顶层目录 TOP-GP-Markdown/ + 若干 md。
func makeGPDocZip(t *testing.T, path string, files map[string]string) string {
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
		w, cerr := zw.Create(config.DocGP.DirName + "/" + name)
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

// runInstallGPDoc 跑一次命令，返回退出码与 stdout。
func runInstallGPDoc(t *testing.T, args ...string) (int, string) {
	t.Helper()
	return testkit.CaptureStdout(t, func() int {
		cmd := newInstallGPDocCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Errorf("install gpdoc %v: %v", args, err)
			return 1
		}
		return 0
	})
}

// 落点三级：--to > config 的 gpdoc.dir > <数据目录>/TOP-GP-Markdown。
func TestResolveGPDocDirPrecedence(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":6}`)
	dir, why, err := resolveGPDocDir("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dataDir, config.DocGP.DirName); dir != want {
		t.Fatalf("缺省落点 = %q, want %q", dir, want)
	}
	if why == "" {
		t.Fatal("缺省落点应当说明'为什么是它'")
	}

	writeConfig(t, `{"schemaVersion":6,"gpdoc":{"dir":"D:/docs/gp"}}`)
	dir, why, err = resolveGPDocDir("")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Clean("D:/docs/gp"); dir != want {
		t.Fatalf("配置落点 = %q, want %q", dir, want)
	}
	if why == "" {
		t.Fatal("配置落点应当说明来源")
	}

	dir, why, err = resolveGPDocDir("rel-docs")
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
func TestResolveGPDocZip(t *testing.T) {
	got, err := resolveGPDocZip("some/dir/topgp.zip")
	if err != nil {
		t.Fatal(err)
	}
	wd, _ := os.Getwd()
	if want := filepath.Join(wd, "some", "dir", "topgp.zip"); got != want {
		t.Fatalf("--from = %q, want %q", got, want)
	}

	old := gpDocZipSource
	gpDocZipSource = func() (string, error) { return `C:\app\` + config.DocGP.ZipName, nil }
	defer func() { gpDocZipSource = old }()
	if got, err = resolveGPDocZip(""); err != nil {
		t.Fatal(err)
	}
	if want := `C:\app\` + config.DocGP.ZipName; got != want {
		t.Fatalf("缺省 zip = %q, want %q", got, want)
	}
}

// 首次安装：解到目标，报文件数；gpdoc 与 bdl 各装各的目录，互不影响。
func TestInstallGPDocFresh(t *testing.T) {
	writeConfig(t, `{"schemaVersion":6}`)
	tmp := t.TempDir()
	src := makeGPDocZip(t, filepath.Join(tmp, config.DocGP.ZipName), map[string]string{
		"01_环境架构/0001-overview.md": "# overview\n",
		"02_操作说明/0001-main.md":     "# main\n",
	})
	dst := filepath.Join(tmp, "gp-docs")

	code, out := runInstallGPDoc(t, "--from", src, "--to", dst)
	if code != 0 {
		t.Fatalf("退出码 = %d\n%s", code, out)
	}
	var rep struct {
		OK      bool `json:"ok"`
		Files   int  `json:"files"`
		Skipped bool `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !rep.OK || rep.Files != 2 || rep.Skipped {
		t.Fatalf("报告不对: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(dst, "01_环境架构", "0001-overview.md")); err != nil {
		t.Fatalf("没解出文件: %v", err)
	}
	// 只拆一层顶层目录：目标里不该再套一层 TOP-GP-Markdown。
	if _, err := os.Stat(filepath.Join(dst, config.DocGP.DirName)); !os.IsNotExist(err) {
		t.Fatal("不该在目标里再套一层顶层目录")
	}
}

// 目标非空：默认跳过。
func TestInstallGPDocSkipsExisting(t *testing.T) {
	writeConfig(t, `{"schemaVersion":6}`)
	tmp := t.TempDir()
	src := makeGPDocZip(t, filepath.Join(tmp, "gp.zip"), map[string]string{"01/a.md": "# new\n"})
	dst := filepath.Join(tmp, "docs")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "mine.md"), []byte("notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runInstallGPDoc(t, "--from", src, "--to", dst)
	if code != 0 {
		t.Fatalf("跳过应当成功退出，got %d\n%s", code, out)
	}
	var rep struct {
		Skipped bool `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !rep.Skipped {
		t.Fatalf("报告不对: %+v", rep)
	}
}

// 随包 zip 不在：稳定的 NOT_FOUND，并提示发版包/--from。
func TestInstallGPDocMissingZip(t *testing.T) {
	writeConfig(t, `{"schemaVersion":6}`)
	old := gpDocZipSource
	gpDocZipSource = func() (string, error) { return filepath.Join(t.TempDir(), "nope.zip"), nil }
	defer func() { gpDocZipSource = old }()

	err := newInstallGPDocCmd().Execute()
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

// 配置读侧：gpdoc.dir 能被 Root.DocDir 取到（web/status 与 --dry-run 都靠它）。
func TestRootDocDirGPDoc(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":6,"gpdoc":{"dir":"D:/docs/gp"},"bdldoc":{"dir":"D:/docs/bdl"}}`)
	root, err := config.Load(filepath.Join(dataDir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := root.DocDir(config.DocGP.Key); got != "D:/docs/gp" {
		t.Fatalf("gpdoc.dir = %q", got)
	}
	if got := root.DocDir(config.DocBDL.Key); got != "D:/docs/bdl" {
		t.Fatalf("bdldoc.dir = %q", got)
	}
	if got := root.DocDir("nope"); got != "" {
		t.Fatalf("未知节名应当回空串，got %q", got)
	}
}
