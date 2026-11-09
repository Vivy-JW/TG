package update

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		"tt-portable/tt.exe":     "ok",
		"../outside-marker.txt":  "不该被写出来",
		"tt-portable/../../x.md": "也不行",
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
		`tt-portable\tzs\designer\a.dll`: "dll",
		`tt-portable/tt.exe`:             "exe",
	})
	dest := filepath.Join(dir, "out")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ExtractZip(p, dest); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "tt-portable", "tzs", "designer", "a.dll"))
	if err != nil || string(got) != "dll" {
		t.Fatalf("反斜杠条目没解开: %v / %q", err, got)
	}
	root, err := PayloadRoot(dest)
	if err != nil || filepath.Base(root) != "tt-portable" {
		t.Fatalf("载荷根 = %q, %v；想要 .../tt-portable", root, err)
	}
}

// 平的包（顶层直接是文件）也要认：别人重打的包不一定带 tt-portable/ 那一层。
func TestPayloadRootFlatAndEmpty(t *testing.T) {
	flat := t.TempDir()
	if err := os.WriteFile(filepath.Join(flat, "tt.exe"), []byte("x"), 0o644); err != nil {
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

// 就地覆盖：src 里没有的保持原样；keepLocal 里的（用户的 config.json）绝不能被模板盖掉。
// 这条对着的是最贵的一次事故：便携形态的配置就住在被覆盖的那个目录里。
func TestCopyTreeKeepsLocalFiles(t *testing.T) {
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
	write(src, "tt.exe", "新版")
	write(src, "config.json", `{"hosts":{}}`) // 包里的空模板
	write(src, filepath.Join("skills", "a", "SKILL.md"), "技能")
	write(dst, "tt.exe", "旧版")
	write(dst, "config.json", `{"我的":"口令"}`)
	write(dst, "我的笔记.txt", "别动我")

	n, err := copyTree(src, dst)
	if err != nil {
		t.Fatalf("copyTree: %v", err)
	}
	if n != 2 {
		t.Errorf("复制了 %d 个文件，想要 2（tt.exe 与技能；config.json 被跳过）", n)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "tt.exe")); string(b) != "新版" {
		t.Errorf("tt.exe 该被覆盖，得到 %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "config.json")); !strings.Contains(string(b), "我的") {
		t.Errorf("用户的 config.json 被模板盖掉了: %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "我的笔记.txt")); string(b) != "别动我" {
		t.Errorf("包内没有的文件该原样留着，得到 %q", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dst, "skills", "a", "SKILL.md")); string(b) != "技能" {
		t.Errorf("新增的技能没落地: %q", b)
	}
}

// `tt version` 的输出比对按词比：子串比会让 0.2.2 匹配上 0.2.20。
func TestOutputHasVersion(t *testing.T) {
	out := "tt 0.2.2 (commit d569d10, 2026-10-07)"
	for _, want := range []string{"0.2.2", "v0.2.2"} {
		if !OutputHasVersion(out, want) {
			t.Errorf("OutputHasVersion(%q, %q) = false，想要 true", out, want)
		}
	}
	for _, want := range []string{"0.2.20", "0.2", "0.2.3", ""} {
		if OutputHasVersion(out, want) {
			t.Errorf("OutputHasVersion(%q, %q) = true，想要 false", out, want)
		}
	}
	if !OutputHasVersion("tt devel (无版本信息)", "devel") {
		t.Error("devel 该被认出来（它也是版本字面量）")
	}
	// 预发布 tag 的版本号要能对上。
	if !OutputHasVersion("tt 0.3.0-rc1 (commit x)", "0.3.0-rc1") {
		t.Error("预发布版本自报该能对上")
	}
}

func TestPlanRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := &Plan{
		Kind: TokenPortable, From: "0.2.1", Target: "0.2.2",
		Dir: dir, Stage: filepath.Join(dir, "stage"), ExePath: filepath.Join(dir, "tt.exe"),
		DataDir: dir, StopPIDs: []int{1, 2}, RestartServe: true,
		RestartArgs:   []string{"serve", "--listen", "127.0.0.1:28670"},
		SkillsTargets: []string{filepath.Join(dir, "skills")},
	}
	if err := SavePlan(dir, p); err != nil {
		t.Fatalf("SavePlan: %v", err)
	}
	got, err := LoadPlan(PlanPath(dir))
	if err != nil {
		t.Fatalf("LoadPlan: %v", err)
	}
	if got.Target != "0.2.2" || got.Kind != TokenPortable || len(got.StopPIDs) != 2 ||
		len(got.SkillsTargets) != 1 || !got.RestartServe {
		t.Errorf("计划没读全: %+v", got)
	}
	// 不完整的计划要当场报错，而不是让更新器跑到一半才发现没东西可装。
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte(`{"kind":"portable"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPlan(bad); err == nil {
		t.Error("缺 target/exePath 的计划该报错")
	}
	if _, err := LoadPlan(filepath.Join(dir, "nope.json")); err == nil {
		t.Error("计划文件不存在该报错")
	}
}

func TestCleanStaleStagesAndRemoveStage(t *testing.T) {
	base := t.TempDir()
	exeDir := filepath.Join(base, "tt")
	if err := os.MkdirAll(exeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(exeDir, "tt.exe")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	fresh := filepath.Join(base, ".tt-update-stage-1")
	old := filepath.Join(base, ".tt-update-stage-2")
	for _, d := range []string{fresh, old} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// 把旧的装作一天前建的。
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	CleanStaleStages(exe, time.Now())
	if _, err := os.Stat(old); err == nil {
		t.Error("一天前的暂存目录该被清掉")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("新暂存目录不该被清（可能正有一次升级在用它）")
	}
	RemoveStage(fresh)
	if _, err := os.Stat(fresh); err == nil {
		t.Error("RemoveStage 该删掉它")
	}
	RemoveStage("") // 空路径不该炸
}

// 更新器临时目录的清理：只清一天前的（正在跑的更新器就在新目录里）。
func TestCleanStaleHelpers(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	fresh := filepath.Join(os.TempDir(), "tt-update-fresh")
	old := filepath.Join(os.TempDir(), "tt-update-old")
	for _, d := range []string{fresh, old} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		// 改不了目录时间就没法构造"一天前"这个前提。这不该靠 Skip 蒙过去：
		// 临时目录的时间一定改得动（仓库支持的平台上都是），改不动就是断言前提坏了。
		t.Fatalf("改 %s 的时间失败: %v", old, err)
	}
	CleanStaleHelpers(time.Now())
	if _, err := os.Stat(old); err == nil {
		t.Error("一天前的更新器目录该被清掉")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Error("新建的更新器目录不该被清")
	}
}

// waitProcessGone 对已经不存在的 pid 立刻返回：前台早就退了，别白等 60 秒。
func TestWaitProcessGoneImmediate(t *testing.T) {
	start := time.Now()
	if err := waitProcessGone(context.Background(), 0, time.Second); err != nil {
		t.Errorf("pid 0 该当作已退出: %v", err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Errorf("等了 %s，该立刻返回", d)
	}
}

// HumanBytes 是给使用者看的：别把 41 MB 写成 41297555。
func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		512:      "512 B",
		2048:     "2.0 KB",
		41297555: "39.4 MB",
		0:        "0 B",
	}
	for in, want := range cases {
		if got := HumanBytes(in); got != want {
			t.Errorf("HumanBytes(%d) = %q，想要 %q", in, got, want)
		}
	}
}
