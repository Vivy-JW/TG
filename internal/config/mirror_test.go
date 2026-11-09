package config

import (
	"path/filepath"
	"testing"
)

func TestNamedSshMirrorPath(t *testing.T) {
	root := filepath.Join(t.TempDir(), "mirror")
	e := &NamedSsh{Name: "109tst"}
	if got, want := e.MirrorPath(root), filepath.Join(root, "109tst"); got != want {
		t.Fatalf("默认镜像目录 = %q, want %q", got, want)
	}
	custom := filepath.Join(t.TempDir(), "custom-109tst")
	e.MirrorDir = custom
	if got := e.MirrorPath(root); got != custom {
		t.Fatalf("环境单独目录 = %q, want %q", got, custom)
	}
	e.MirrorDir = ""
	if got := e.MirrorPath(""); got != "" {
		t.Fatalf("根目录和环境目录都为空时应返回空串，得到 %q", got)
	}
}
