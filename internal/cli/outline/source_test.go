package outline

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/output"
)

func write4gl(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("FUNCTION f()\nEND FUNCTION\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSourceExplicitFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "demo.4gl")
	write4gl(t, p)
	src, err := resolveSource(sourceOpts{File: p, Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if src.Mode != "file" || src.Path != p || src.Root != "" {
		t.Fatalf("src = %+v", src)
	}
}

func TestResolveSourceFromDir(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "erp", "asf", "4gl", "demo.4gl")
	write4gl(t, p)
	src, err := resolveSource(sourceOpts{File: "demo.4gl", Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if src.Mode != "mirror" || src.Root != root || filepath.Base(src.Path) != "demo.4gl" {
		t.Fatalf("src = %+v", src)
	}
}

func TestResolveSourceRelativePath(t *testing.T) {
	root := t.TempDir()
	write4gl(t, filepath.Join(root, "erp", "asf", "4gl", "demo.4gl"))
	src, err := resolveSource(sourceOpts{File: "erp/asf/4gl/demo.4gl", Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(src.Path), "erp/asf/4gl/demo.4gl") {
		t.Fatalf("src = %+v", src)
	}
}

func TestResolveSourceAmbiguous(t *testing.T) {
	root := t.TempDir()
	write4gl(t, filepath.Join(root, "erp", "asf", "4gl", "demo.4gl"))
	write4gl(t, filepath.Join(root, "erp", "aim", "4gl", "demo.4gl"))
	_, err := resolveSource(sourceOpts{File: "demo.4gl", Dir: root})
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Code != "FILE_AMBIGUOUS" || oe.Exit != 2 {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(oe.Message, "erp/asf/4gl/demo.4gl") || !strings.Contains(oe.Message, "erp/aim/4gl/demo.4gl") {
		t.Fatalf("候选路径不全: %s", oe.Message)
	}
}

func TestResolveSourceMissing(t *testing.T) {
	_, err := resolveSource(sourceOpts{File: "demo.4gl"})
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Code != output.CodeNotFound || oe.Exit != 4 {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveSourceRejectsNon4GL(t *testing.T) {
	_, err := resolveSource(sourceOpts{File: "demo.txt"})
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Code != output.CodeUsage {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveSourceRejectsDirAndEnv(t *testing.T) {
	_, err := resolveSource(sourceOpts{File: "demo.4gl", Dir: t.TempDir(), Env: "e"})
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Code != output.CodeUsage {
		t.Fatalf("err = %v", err)
	}
}

func TestResolveSourceFromEnvMirror(t *testing.T) {
	root := t.TempDir()
	write4gl(t, filepath.Join(root, "erp", "asf", "4gl", "demo.4gl"))
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.json")
	cfg := `{"hosts":{"sshs":[{"name":"e","mirrorDir":` + strconv.Quote(root) + `}]}}
`
	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	old := common.ConfigPath
	common.ConfigPath = cfgPath
	defer func() { common.ConfigPath = old }()

	src, err := resolveSource(sourceOpts{File: "demo.4gl", Env: "e"})
	if err != nil {
		t.Fatal(err)
	}
	if src.Mode != "mirror" || src.Env != "e" || src.Root != root {
		t.Fatalf("src = %+v", src)
	}
}
