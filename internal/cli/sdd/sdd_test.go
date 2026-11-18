package sdd

import (
	"os"
	"path/filepath"
	"testing"

	"tg/internal/config"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// fixture 造一棵镜像树：两份 axmt410*、一份 axmt400*、一份不在 sdd/ 里。
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "axm", "sdd", "axmt410dis.sdd"), "单据说明")
	writeFile(t, filepath.Join(root, "axm", "sdd", "axmt410icd.sdd"), "维护说明")
	writeFile(t, filepath.Join(root, "axm", "sdd", "axmt400dis.sdd"), "别的作业")
	writeFile(t, filepath.Join(root, "axm", "other", "axmt410.sdd"), "不在 sdd 目录")
	writeFile(t, filepath.Join(root, "axm", "sdd", "readme.txt"), "不是 sdd")
	return root
}

func TestNormalizeJob(t *testing.T) {
	cases := map[string]string{
		"  AXMT410 ":       "axmt410",
		"axmt410.sdd":      "axmt410",
		"axm/axmt410dis":   "axm/axmt410dis",
		"axm\\axmt410dis":  "axm/axmt410dis",
		"./axm/axmt410dis": "axm/axmt410dis",
	}
	for in, want := range cases {
		if got := normalizeJob(in); got != want {
			t.Errorf("normalizeJob(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsSDD(t *testing.T) {
	good := []string{"axm/sdd/axmt410dis.sdd", "qry/sdd/x.SDD", "a/b/sdd/c.sdd"}
	bad := []string{"axm/sdd/readme.txt", "axm/sdd2/x.sdd", "axm/4gl/x.sdd", "sdd/x.sdd"}
	for _, p := range good {
		if !isSDD(p) {
			t.Errorf("isSDD(%q) = false, want true", p)
		}
	}
	for _, p := range bad {
		if isSDD(p) {
			t.Errorf("isSDD(%q) = true, want false", p)
		}
	}
}

func TestScanLocalMatchesPrefixOnly(t *testing.T) {
	root := fixture(t)
	got, err := scanLocal(root, "axmt410")
	if err != nil {
		t.Fatalf("scanLocal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("命中 %d 份，want 2: %+v", len(got), got)
	}
	want := []string{"axm/sdd/axmt410dis.sdd", "axm/sdd/axmt410icd.sdd"}
	for i, m := range got {
		if m.Path != want[i] {
			t.Errorf("[%d] path = %q, want %q", i, m.Path, want[i])
		}
		if m.Source != "mirror" {
			t.Errorf("[%d] source = %q, want mirror", i, m.Source)
		}
		if m.Bytes == 0 {
			t.Errorf("[%d] bytes = 0，应该带文件大小", i)
		}
	}
}

func TestScanLocalWithPathPrefix(t *testing.T) {
	root := fixture(t)
	got, err := scanLocal(root, "axm/axmt410")
	if err != nil {
		t.Fatalf("scanLocal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("命中 %d 份，want 2: %+v", len(got), got)
	}
}

func TestPickOnePrefersExactThenFirst(t *testing.T) {
	ms := []match{
		{Path: "axm/sdd/axmt410dis.sdd"},
		{Path: "axm/sdd/axmt410icd.sdd"},
	}
	if got := pickOne(ms, "axmt410").Path; got != "axm/sdd/axmt410dis.sdd" {
		t.Errorf("没有精确命中时应取第一份，得到 %q", got)
	}
	ms = append(ms, match{Path: "axm/sdd/axmt410.sdd"})
	if got := pickOne(ms, "axmt410").Path; got != "axm/sdd/axmt410.sdd" {
		t.Errorf("有精确命中时应取它，得到 %q", got)
	}
}

func TestReadMatchFromMirror(t *testing.T) {
	root := fixture(t)
	got, err := readMatch(match{Path: "axm/sdd/axmt410dis.sdd", Source: "mirror"}, root, nil)
	if err != nil {
		t.Fatalf("readMatch: %v", err)
	}
	if string(got) != "单据说明" {
		t.Errorf("内容 = %q，want 单据说明", string(got))
	}
	var e *config.NamedSsh
	if _, err := readMatch(match{Path: "axm/sdd/x.sdd", Source: "server"}, root, e); err == nil {
		t.Error("没有环境却要读服务器文件，应该报错")
	}
}
