package mirror

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneStaleOnlyFullSet(t *testing.T) {
	dst := t.TempDir()
	keep := filepath.Join(dst, "keep.4gl")
	stale := filepath.Join(dst, "stale.4gl")
	subStale := filepath.Join(dst, "sub", "old.4gl")
	other := filepath.Join(dst, "note.txt")
	for _, p := range []string{keep, stale, subStale, other} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	n, err := pruneStale(dst, []file{{rrel: "keep.4gl"}})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("pruned = %d, want 2", n)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("keep 被误删: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("非 .4gl 被误删: %v", err)
	}
	for _, p := range []string{stale, subStale} {
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
