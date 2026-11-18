package zipx

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// makeZip 造一个测试用压缩包。entries 的 key 是条目名（含反斜杠也能给，解包器要认）。
func makeZip(t *testing.T, dir string, entries map[string]string) string {
	t.Helper()
	p := filepath.Join(dir, "pkg.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

// 解包器必须自己挡住越界条目：解出来的东西是要覆盖用户程序目录的，
// 一个 `..\` 条目就能往别处写文件。
func TestExtractZipRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	escape := makeZip(t, dir, map[string]string{
		"tg-portable/tg.exe":     "ok",
		"../outside-marker.txt":  "不该被写出来",
		"tg-portable/../../x.md": "也不行",
	})
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExtractZip(escape, dest); err == nil {
		t.Fatal("带 .. 的条目该被拒绝")
	}
	// 拒绝之后：越界文件一个都不该出现，且解包在遇到坏条目时**不继续**往下写。
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "outside-marker.txt")); err == nil {
		t.Error("越界文件被写出来了")
	}
	for _, p := range []string{filepath.Join(dir, "outside-marker.txt"), filepath.Join(dest, "..", "outside-marker.txt")} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s 不该存在", p)
		}
	}

	abs := makeZip(t, t.TempDir(), map[string]string{"/absolute.txt": "x"})
	dest2 := filepath.Join(dir, "out2")
	_ = os.MkdirAll(dest2, 0o755)
	if err := ExtractZip(abs, dest2); err == nil {
		t.Error("绝对路径条目该被拒绝")
	}
}

// 反斜杠分隔的条目名（Windows 打包工具会这么写）要能正常解开。
func TestExtractZipHandlesBackslashEntries(t *testing.T) {
	dir := t.TempDir()
	p := makeZip(t, dir, map[string]string{
		`tg-portable\tzs\designer\a.dll`: "dll",
		`tg-portable/tg.exe`:             "exe",
	})
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExtractZip(p, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "tg-portable", "tzs", "designer", "a.dll"))
	if err != nil || string(got) != "dll" {
		t.Fatalf("反斜杠条目没解开: %v / %q", err, got)
	}
	root, err := PayloadRoot(dest)
	if err != nil || filepath.Base(root) != "tg-portable" {
		t.Fatalf("载荷根 = %q, %v；想要 .../tg-portable", root, err)
	}
}

// 平的包（顶层直接是文件）也要认：别人重打的包不一定带 tg-portable/ 那一层。
func TestPayloadRootFlatAndEmpty(t *testing.T) {
	flat := t.TempDir()
	if err := os.WriteFile(filepath.Join(flat, "tg.exe"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(flat, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := PayloadRoot(flat)
	if err != nil || root != flat {
		t.Errorf("平的包载荷根该是它自己: %q, %v", root, err)
	}
	empty := t.TempDir()
	if _, err := PayloadRoot(empty); err == nil {
		t.Error("空目录该报错")
	}
}

// CopyTree 的 skip 拿到的是相对 src 的斜杠路径：调用方靠它保护用户自己的文件。
func TestCopyTreeSkipAndOverwrite(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	write := func(dir, name, body string) {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(src, "tg.exe", "新版")
	write(src, filepath.Join("config.json"), `{"模板":true}`)
	write(dst, "config.json", `{"我的":"口令"}`)
	write(dst, "我的笔记.txt", "别动我")
	var seen []string
	n, err := CopyTree(src, dst, func(rel string) bool {
		seen = append(seen, rel)
		return rel == "config.json"
	})
	if err != nil {
		t.Fatalf("CopyTree: %v", err)
	}
	if n != 1 {
		t.Errorf("复制了 %d 个文件，想要 1（只 tg.exe）", n)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "config.json")); string(b) != `{"我的":"口令"}` {
		t.Errorf("skip 命中却被覆盖了: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "我的笔记.txt")); string(b) != "别动我" {
		t.Errorf("src 里没有的文件该原样留着，得到 %q", b)
	}
	if len(seen) == 0 || seen[0] != "." {
		t.Errorf("skip 该收到相对路径（首条是 \".\"），得到 %v", seen)
	}
}
