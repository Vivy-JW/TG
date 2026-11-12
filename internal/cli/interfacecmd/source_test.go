package interfacecmd

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, p, s string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveSourcesFromDir(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "erp", "asf", "42f", "demo.42f"), "<Form/>")
	src, err := resolveSources(sourceOpts{Form: "demo", Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if src.Module != "asf" || src.Form != "demo" {
		t.Fatalf("src = %+v", src)
	}
	if filepath.Base(src.Form42F) != "demo.42f" {
		t.Fatalf("path = %q", src.Form42F)
	}
}

func TestResolveSourcesAmbiguousModule(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "erp", "asf", "42f", "demo.42f"), "<Form/>")
	write(t, filepath.Join(root, "erp", "aim", "42f", "demo.42f"), "<Form/>")
	if _, err := resolveSources(sourceOpts{Form: "demo", Dir: root}); err == nil {
		t.Fatal("同名跨模块应报 FORM_AMBIGUOUS")
	}
	src, err := resolveSources(sourceOpts{Form: "demo", Dir: root, Module: "aim"})
	if err != nil {
		t.Fatal(err)
	}
	if src.Module != "aim" {
		t.Fatalf("module = %q", src.Module)
	}
}

func TestResolveSourcesExplicitFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "demo.42f")
	write(t, f, "<Form/>")
	src, err := resolveSources(sourceOpts{FileF: f})
	if err != nil {
		t.Fatal(err)
	}
	if src.Form != "demo" || src.Mode != "files" || src.Form42F != f {
		t.Fatalf("src = %+v", src)
	}
}

// 标签来源只认 gae_file：--labels off 时完全不查库，也不要求点名环境。
func TestLoadLabelsOff(t *testing.T) {
	f.labels = labelsOff
	m, note, err := loadLabels("p_zta")
	if err != nil || m != nil || note != "off" {
		t.Fatalf("off: m=%v note=%q err=%v", m, note, err)
	}
	f.labels = labelsAuto
}

func TestModuleFromPath(t *testing.T) {
	for in, want := range map[string]string{
		"root/erp/asf/42f/demo.42f": "asf",
		"root/other/demo.42f":       "",
	} {
		if got := moduleFromPath(in); got != want {
			t.Errorf("moduleFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}
