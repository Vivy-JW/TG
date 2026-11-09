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
	write(t, filepath.Join(root, "erp", "asf", "42s", "zh_CN", "demo.42s"), "x")
	src, err := resolveSources(sourceOpts{Form: "demo", Dir: root})
	if err != nil {
		t.Fatal(err)
	}
	if src.Module != "asf" || src.Form != "demo" {
		t.Fatalf("src = %+v", src)
	}
	if filepath.Base(src.Form42F) != "demo.42f" || filepath.Base(src.Form42S) != "demo.42s" {
		t.Fatalf("paths = %q / %q", src.Form42F, src.Form42S)
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

func TestResolveSourcesExplicitFiles(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "demo.42f")
	s := filepath.Join(dir, "demo.42s")
	write(t, f, "<Form/>")
	write(t, s, "x")
	src, err := resolveSources(sourceOpts{FileF: f, FileS: s})
	if err != nil {
		t.Fatal(err)
	}
	if src.Form != "demo" || src.Mode != "files" || src.Form42S != s {
		t.Fatalf("src = %+v", src)
	}
}

func TestModuleFromPath(t *testing.T) {
	for in, want := range map[string]string{
		"root/erp/asf/42f/demo.42f":           "asf",
		"root/com/ima/42s/zh_CN/demo.42s":     "ima",
		"root/erp/asf/42s/nested/zh_CN/a.42s": "asf",
		"root/other/demo.42f":                 "",
	} {
		if got := moduleFromPath(in); got != want {
			t.Errorf("moduleFromPath(%q) = %q, want %q", in, got, want)
		}
	}
}
