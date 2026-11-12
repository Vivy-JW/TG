package mirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKindOf(t *testing.T) {
	cases := map[string]string{
		"azz/4gl/p_zta.4gl":       "4gl",
		"azz/42f/p_zta.42f":       "42f",
		"azz/42f/1":               "42f",
		"azz/4gl/include/xx.inc":  "inc",
		"azz/42s/zh_CN/p_zta.42s": "",
		"azz/4gl/readme.txt":      "",
	}
	for rel, want := range cases {
		if got := KindOf(rel); got != want {
			t.Errorf("KindOf(%q) = %q, want %q", rel, got, want)
		}
	}
}

func TestPruneStaleOnlyFullSet(t *testing.T) {
	dst := t.TempDir()
	keep := filepath.Join(dst, "azz", "4gl", "keep.4gl")
	stale := filepath.Join(dst, "azz", "4gl", "stale.4gl")
	subStale := filepath.Join(dst, "cbm", "42f", "old.42f")
	incStale := filepath.Join(dst, "cbm", "4gl", "old.inc")
	other := filepath.Join(dst, "note.txt")
	legacy42s := filepath.Join(dst, "azz", "42s", "zh_CN", "x.42s")
	for _, p := range []string{keep, stale, subStale, incStale, other, legacy42s} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := pruneStale(dst, []file{{rrel: filepath.Join("azz", "4gl", "keep.4gl")}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("pruned = %d, want 3", n)
	}
	for _, p := range []string{keep, other, legacy42s} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("%s 被误删: %v", p, err)
		}
	}
	for _, p := range []string{stale, subStale, incStale} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("stale 未删除: %s", p)
		}
	}
	if _, err := os.Stat(filepath.Dir(subStale)); !os.IsNotExist(err) {
		t.Fatalf("空目录未清理: %s", filepath.Dir(subStale))
	}
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("镜像根被误删: %v", err)
	}
}
