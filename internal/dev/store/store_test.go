package store

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
)

//---------------------------------------------------------------------------
// 测试夹具（只依赖标准库；不需要真实包语料）
//---------------------------------------------------------------------------

// testEntries 是一个最小 .tzc 的条目集：.tap / .tgl 必需，ver 必需且主次版本 = 1.0。
var testEntries = []struct {
	Name string
	Data []byte
}{
	{"aapp131.tap", []byte(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\n" +
		`<add_points prog="aapp131" std_prog="aapp131" erpver="3.0" module="AAP" ver="66" env="c" topind="sd">` + "\n" +
		"</add_points>\n")},
	{"aapp131.tgl", []byte("{<point name=\"other.function\"/>}\n")},
	{"aapp131.4gl", []byte("MAIN\n  DISPLAY 1\nEND MAIN\n")},
	{"aapp131.bdx", []byte("<bdx/>\n")},
	{"ver", []byte("1.0\n")},
}

func testFenced() []byte {
	return []byte("// tdev fenced document\nFUNCTION aapp131_f1()\n  DISPLAY 1\nEND FUNCTION\n")
}

// tempDir 建一个临时目录。优先系统临时目录；沙箱里不可写时退回包目录下。
func tempDir(t *testing.T) string {
	t.Helper()
	d, err := os.MkdirTemp("", "tdevstore-")
	if err != nil {
		d, err = os.MkdirTemp(".", ".tdevstore-")
		if err != nil {
			t.Fatalf("创建临时目录失败：%v（系统临时目录不可写且包目录也不可写）", err)
		}
	}
	if abs, aerr := filepath.Abs(d); aerr == nil {
		d = abs
	}
	t.Cleanup(func() { _ = os.RemoveAll(d) })
	return d
}

// makePkg 在 dir 下造一个合成 .tzc 并用 pkgfile.Open 打开。
func makePkg(t *testing.T, dir string) *pkgfile.Package {
	t.Helper()
	p := filepath.Join(dir, "aapp131.tzc")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, e := range testEntries {
		hdr := &zip.FileHeader{Name: e.Name, Method: zip.Deflate}
		hdr.SetModTime(fixed)
		w, err := zw.CreateHeader(hdr)
		if err != nil {
			t.Fatalf("zip CreateHeader(%s): %v", e.Name, err)
		}
		if _, err := w.Write(e.Data); err != nil {
			t.Fatalf("zip Write(%s): %v", e.Name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip Close: %v", err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatalf("写合成包失败：%v", err)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatalf("pkgfile.Open: %v", err)
	}
	return pkg
}

// makeDoc 造一个含 3 个 Region（含 1 个子区间）+ 1 个 Span 的文档。
func makeDoc(pkg *pkgfile.Package, fenced []byte) *model.Document {
	body := model.ByteRange{Start: 30, End: 60}
	inner := model.ByteRange{Start: 95, End: 120}
	pkgSha, _ := model.Sha256File(pkg.Path)
	return &model.Document{
		Text: fenced,
		Prog: "aapp131",
		Env: model.EnvContext{
			Env: "c", Topind: "sd", LoginUser: "tiptop", ProgType: "M",
			IsStandard: true, SectionFlag: false,
		},
		Regions: []*model.Region{
			{
				Name: "function.aapp131_f1", Kind: model.RegionPoint, Editable: true,
				Meta: map[string]string{
					"status": "", "src": "c", "new": "Y", "order": "1",
					"cite_std": "N", "edit": "c", "mark": "Y",
				},
				ContentSpan: model.ByteRange{Start: 25, End: 65},
				BodySpan:    &body,
				Sha256:      model.Sha256Bytes([]byte("body-1")),
				Origin:      "placeholder",
			},
			{
				Name: "aapp131.main", Kind: model.RegionSection, Editable: false,
				DenyCode:    model.DenySectionReadonly,
				ContentSpan: model.ByteRange{Start: 90, End: 140},
				Sha256:      model.Sha256Bytes([]byte("section-main")),
				Origin:      "placeholder",
				Children: []*model.Region{
					{
						Name: "function.aapp131_inner", Kind: model.RegionPoint, Editable: true,
						Meta: map[string]string{
							"status": "u", "src": "s", "new": "N", "order": "2",
						},
						ContentSpan: model.ByteRange{Start: 95, End: 130},
						BodySpan:    &inner,
						Sha256:      model.Sha256Bytes([]byte("body-2")),
						Origin:      "anchor-injected",
						AnchorType:  "function",
					},
				},
			},
		},
		Spans: []model.Span{
			{Name: "prefix", Span: model.ByteRange{Start: 0, End: 25}, Sha256: model.Sha256Bytes(fenced[:25])},
		},
		PkgPath:   pkg.Path,
		PkgSha256: pkgSha,
		Ver:       "66",
		Anchors:   map[string]bool{"other.function": true},
	}
}

func mustCreate(t *testing.T, pkg *pkgfile.Package, doc *model.Document, fenced []byte, name string) *Workspace {
	t.Helper()
	ws, err := Create(filepath.Join(tempDir(t), name), doc, pkg, fenced)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	return ws
}

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func gitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

//---------------------------------------------------------------------------
// Create / 布局
//---------------------------------------------------------------------------

func TestCreateWorkspaceLayout(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	doc := makeDoc(pkg, fenced)

	ws, err := Create(filepath.Join(root, "ws"), doc, pkg, fenced)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ws.Dir != filepath.Join(root, "ws") {
		t.Fatalf("Workspace.Dir = %q", ws.Dir)
	}

	// 全部预期文件都在
	for _, rel := range []string{
		FileProg, FileManifest, FileSnapshotIndex, FileGitignore,
		FileBase, FileBaseSha256, FileRegions,
	} {
		p := filepath.Join(ws.Dir, filepath.FromSlash(rel))
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("缺少文件 %s：%v", rel, err)
		}
		if st.IsDir() {
			t.Fatalf("%s 应为文件", rel)
		}
	}

	// 快照逐字节等于 zip 条目
	for _, e := range testEntries {
		got, err := os.ReadFile(filepath.Join(ws.Dir, DirSnapshotEntries, e.Name))
		if err != nil {
			t.Fatalf("读快照 %s：%v", e.Name, err)
		}
		if !bytes.Equal(got, e.Data) {
			t.Fatalf("快照 %s 字节不一致：got %q want %q", e.Name, got, e.Data)
		}
		if se, err := ws.SnapshotEntry(e.Name); err != nil || !bytes.Equal(se, e.Data) {
			t.Fatalf("SnapshotEntry(%s) = %q, %v", e.Name, se, err)
		}
	}
	if _, err := ws.SnapshotEntry("nope.tap"); err == nil {
		t.Fatal("读取不存在的快照条目应当报错")
	} else {
		var ioe *IOError
		if !errors.As(err, &ioe) || ioe.ExitCode() != 5 {
			t.Fatalf("快照缺失错误应为 *IOError/5：%v", err)
		}
	}

	// 条目清单
	idx, err := ws.SnapshotIndex()
	if err != nil {
		t.Fatalf("SnapshotIndex: %v", err)
	}
	wantRoles := map[string]string{
		"aapp131.tap": "tap", "aapp131.tgl": "tgl", "aapp131.4gl": "4gl",
		"aapp131.bdx": "bdx", "ver": "ver",
	}
	if len(idx) != len(testEntries) {
		t.Fatalf("SnapshotIndex 条目数 = %d, want %d", len(idx), len(testEntries))
	}
	for _, e := range idx {
		if wantRoles[e.Name] != e.Role {
			t.Errorf("条目 %s role = %q, want %q", e.Name, e.Role, wantRoles[e.Name])
		}
		if e.Size != len(pkg.Entry(e.Name).Data) {
			t.Errorf("条目 %s size = %d", e.Name, e.Size)
		}
		if e.Sha256 != pkgfile.Sha256Hex(pkg.Entry(e.Name).Data) {
			t.Errorf("条目 %s sha256 不对", e.Name)
		}
	}

	// manifest 往返
	m, err := ws.Manifest()
	if err != nil {
		t.Fatalf("Manifest: %v", err)
	}
	if m.Tool != ToolName || m.ToolVersion != ToolVersion {
		t.Errorf("tool = %q/%q", m.Tool, m.ToolVersion)
	}
	if m.Prog != "aapp131" || m.Module != "AAP" || m.ErpVer != "3.0" || m.Env != "c" || m.Kind != "Code" {
		t.Errorf("manifest 头部字段不对：%+v", m)
	}
	if m.Pkg.Path != pkg.Path || m.Pkg.Ver != "66" || m.Pkg.Sha256 != doc.PkgSha256 {
		t.Errorf("manifest.pkg 不对：%+v", m.Pkg)
	}
	if !m.Anchors["other.function"] {
		t.Errorf("manifest.anchors = %v", m.Anchors)
	}
	if len(m.Entries) != len(testEntries) {
		t.Errorf("manifest.entries = %d", len(m.Entries))
	}
	if len(m.Export.Only) != 0 {
		t.Errorf("manifest.export 不对：%+v", m.Export)
	}
	if m.Section.State == "" {
		t.Errorf("manifest.section.state 不能为空：%+v", m.Section)
	}

	// manifest.json 是稳定的两空格缩进 JSON，且能往返
	rawM, err := os.ReadFile(filepath.Join(ws.Dir, FileManifest))
	if err != nil {
		t.Fatal(err)
	}
	again, err := model.MarshalJSONStable(m)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rawM, again) {
		t.Errorf("manifest.json 不满足 RoundTrip（%d vs %d 字节）", len(rawM), len(again))
	}
	var m2 Manifest
	if err := json.Unmarshal(rawM, &m2); err != nil {
		t.Fatalf("manifest.json 不是合法 JSON：%v", err)
	}

	// Stats 含嵌套子区间：2 point（1 顶层 + 1 嵌套）+ 1 section
	if m.Stats.Points != 2 || m.Stats.Sections != 1 || m.Stats.Editable != 2 || m.Stats.Readonly != 1 {
		t.Errorf("Stats = %+v, want {2 1 2 1}", m.Stats)
	}
	if len(m.Regions) != 3 {
		t.Fatalf("manifest.regions = %d, want 3（含嵌套）", len(m.Regions))
	}
	byName := map[string]RegionSummary{}
	for _, r := range m.Regions {
		byName[r.Name] = r
	}
	f1 := byName["function.aapp131_f1"]
	if f1.Kind != string(model.RegionPoint) || !f1.Editable ||
		f1.Src != "c" || f1.New != "Y" || f1.Order != "1" || f1.CiteStd != "N" ||
		f1.Edit != "c" || f1.Mark != "Y" || f1.Status != "" || f1.Origin != "placeholder" {
		t.Errorf("RegionSummary(function.aapp131_f1) = %+v", f1)
	}
	main := byName["aapp131.main"]
	if main.Editable || main.DenyCode != model.DenySectionReadonly {
		t.Errorf("只读区段摘要不对：%+v", main)
	}
	if main.Reason == "" || main.Reason != model.DenyReasonText(model.DenySectionReadonly) {
		t.Errorf("reason 未由 DenyReasonText 填充：%q", main.Reason)
	}
	inner := byName["function.aapp131_inner"]
	if inner.Status != "u" || inner.AnchorType != "function" || inner.Origin != "anchor-injected" {
		t.Errorf("嵌套点摘要不对：%+v", inner)
	}
	if len(m.Spans) != 1 || m.Spans[0].Name != "prefix" || m.Spans[0].End != 25 {
		t.Errorf("manifest.spans = %+v", m.Spans)
	}

	// regions.json 往返
	rf, err := ws.Regions()
	if err != nil {
		t.Fatalf("Regions: %v", err)
	}
	sha := model.Sha256Bytes(fenced)
	if rf.Prog != "aapp131" || rf.BaseSha256 != sha || rf.TextLen != len(fenced) {
		t.Errorf("regions.json 头部不对：prog=%q sha=%q len=%d", rf.Prog, rf.BaseSha256, rf.TextLen)
	}
	if rf.Env.Env != "c" || rf.Env.LoginUser != "tiptop" {
		t.Errorf("regions.json env 不对：%+v", rf.Env)
	}
	if len(rf.Regions) != 2 || len(rf.Regions[1].Children) != 1 {
		t.Fatalf("regions.json 区间表不对：%d 个顶层", len(rf.Regions))
	}
	if rf.Regions[1].Children[0].Name != "function.aapp131_inner" {
		t.Errorf("嵌套子区间丢失：%+v", rf.Regions[1].Children)
	}
	if len(rf.Spans) != 1 {
		t.Errorf("regions.json spans = %+v", rf.Spans)
	}
	rawR, err := os.ReadFile(filepath.Join(ws.Dir, FileRegions))
	if err != nil {
		t.Fatal(err)
	}
	againR, err := model.MarshalJSONStable(rf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(rawR, againR) {
		t.Errorf("regions.json 不满足 RoundTrip")
	}

	// 基线
	base, err := ws.ReadBase()
	if err != nil {
		t.Fatalf("ReadBase: %v", err)
	}
	if !bytes.Equal(base, fenced) {
		t.Errorf("base.full.4gl != fenced")
	}
	edited, err := ws.ReadEdited()
	if err != nil {
		t.Fatalf("ReadEdited: %v", err)
	}
	if !bytes.Equal(edited, fenced) {
		t.Errorf("prog.full.4gl != fenced")
	}
	shaFile, err := os.ReadFile(filepath.Join(ws.Dir, FileBaseSha256))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(shaFile)) != sha {
		t.Errorf("base.sha256 = %q, want %q", strings.TrimSpace(string(shaFile)), sha)
	}

	// 确定性：manifest.json / regions.json 不含时钟（R7）
	for _, raw := range [][]byte{rawM, rawR} {
		for _, needle := range []string{"timestamp", "created_at", "2024-", "2025-", "2026-"} {
			if bytes.Contains(raw, []byte(needle)) {
				t.Errorf("工作区文件里出现了时钟痕迹 %q", needle)
			}
		}
	}

	// Open 能打开自己产出的工作区
	if _, err := Open(ws.Dir); err != nil {
		t.Fatalf("Open(刚 Create 的工作区): %v", err)
	}
}

func TestCreateRefusesExistingWorkspace(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	doc := makeDoc(pkg, fenced)
	dir := filepath.Join(root, "ws")
	if _, err := Create(dir, doc, pkg, fenced); err != nil {
		t.Fatalf("第一次 Create: %v", err)
	}
	// 先记下 prog.full.4gl 的内容，确认第二次调用不会破坏它
	before, err := os.ReadFile(filepath.Join(dir, FileProg))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Create(dir, doc, pkg, []byte("// 第二次的不同内容\n"))
	if err == nil {
		t.Fatal("第二次 Create 到同一目录应当报错")
	}
	var ioe *IOError
	if !errors.As(err, &ioe) || ioe.ExitCode() != 5 {
		t.Fatalf("重复 Create 的错误应为 *IOError/5：%v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, FileProg))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("重复 Create 破坏了已有工作区")
	}
}

func TestOpenRejectsIncompleteWorkspace(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	ws := mustCreate(t, pkg, makeDoc(pkg, fenced), fenced, "ws")

	if err := os.Remove(filepath.Join(ws.Dir, FileSnapshotIndex)); err != nil {
		t.Fatal(err)
	}
	_, err := Open(ws.Dir)
	if err == nil {
		t.Fatal("缺少 snapshot/index.json 时 Open 应当报错")
	}
	var ioe *IOError
	if !errors.As(err, &ioe) || ioe.ExitCode() != 5 {
		t.Fatalf("Open 的错误应为 *IOError/5：%v", err)
	}
	if _, err := Open(filepath.Join(ws.Dir, "does-not-exist")); err == nil {
		t.Fatal("目录不存在时 Open 应当报错")
	}
}

//---------------------------------------------------------------------------
// AtomicWrite
//---------------------------------------------------------------------------

func TestAtomicWriteOverwrite(t *testing.T) {
	dir := tempDir(t)
	p := filepath.Join(dir, "a.txt")
	if err := AtomicWrite(p, []byte("first")); err != nil {
		t.Fatalf("AtomicWrite(create): %v", err)
	}
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(p, []byte("second-longer-content")); err != nil {
		t.Fatalf("AtomicWrite(overwrite): %v", err)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "second-longer-content" {
		t.Fatalf("内容 = %q", b)
	}
	after, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if after.Mode().Perm() != before.Mode().Perm() {
		t.Errorf("权限位被改变：%v → %v", before.Mode().Perm(), after.Mode().Perm())
	}
	// 不留 *.tmp*
	left, err := filepath.Glob(filepath.Join(dir, "*tmp*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		t.Fatalf("残留临时文件：%v", left)
	}
	// 目录也必须干净
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 {
		t.Fatalf("目录里有多余文件：%v", ents)
	}
}

func TestAtomicWriteRenameOntoDirectoryFails(t *testing.T) {
	dir := tempDir(t)
	target := filepath.Join(dir, "sub")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := AtomicWrite(target, []byte("nope"))
	if err == nil {
		t.Fatal("AtomicWrite 到目录应当报错")
	}
	var ioe *IOError
	if !errors.As(err, &ioe) || ioe.ExitCode() != 5 {
		t.Fatalf("错误应为 *IOError/5：%v", err)
	}
	st, serr := os.Stat(target)
	if serr != nil || !st.IsDir() {
		t.Fatalf("目标目录被破坏：%v", serr)
	}
	kb, rerr := os.ReadFile(keep)
	if rerr != nil || string(kb) != "keep" {
		t.Fatalf("目录内文件被破坏：%q %v", kb, rerr)
	}
	left, _ := filepath.Glob(filepath.Join(dir, "*tmp*"))
	if len(left) != 0 {
		t.Fatalf("残留临时文件：%v", left)
	}
	// 目标不存在时也必须清理临时文件
	err = AtomicWrite(filepath.Join(dir, "no-such-parent", "x.txt"), []byte("x"))
	if err == nil {
		t.Fatal("父目录不存在时应当报错")
	}
	if !errors.As(err, &ioe) {
		t.Fatalf("错误应为 *IOError：%v", err)
	}
}

//---------------------------------------------------------------------------
// Lock
//---------------------------------------------------------------------------

func TestLockMutualExclusionAndStale(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	ws := mustCreate(t, pkg, makeDoc(pkg, fenced), fenced, "ws")

	release, err := ws.Lock()
	if err != nil {
		t.Fatalf("第一次 Lock: %v", err)
	}
	lockPath := filepath.Join(ws.Dir, FileLock)
	raw, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("锁文件没落盘：%v", err)
	}
	var rec lockRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("锁文件不是 JSON：%v（%q）", err, raw)
	}
	if rec.Pid != os.Getpid() || rec.Started == "" {
		t.Fatalf("锁记录不对：%+v", rec)
	}

	_, err = ws.Lock()
	if err == nil {
		t.Fatal("同一进程第二次 Lock 应当失败")
	}
	var ioe *IOError
	if !errors.As(err, &ioe) || ioe.ExitCode() != 5 {
		t.Fatalf("占用错误应为 *IOError/5：%v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "lock") || !strings.Contains(msg, "删除") {
		t.Fatalf("错误提示应说明删除锁文件（%s）：%q", lockPath, msg)
	}

	release()
	if _, err := os.Stat(lockPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("release() 之后锁文件应消失：%v", err)
	}
	// 幂等释放
	release()

	release2, err := ws.Lock()
	if err != nil {
		t.Fatalf("release 后应当能再次 Lock: %v", err)
	}
	release2()

	// 陈旧锁（mtime 远早于 30 分钟）应被抢占
	if err := os.WriteFile(lockPath, []byte(`{"pid":999999}`), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	release3, err := ws.Lock()
	if err != nil {
		t.Fatalf("陈旧锁应被清理并抢占：%v", err)
	}
	release3()

	// 清理后锁文件存在，且不再是 999999 那个陈旧的
	raw, err = os.ReadFile(lockPath)
	if err == nil {
		if err := json.Unmarshal(raw, &rec); err == nil && rec.Pid == 999999 {
			t.Fatal("陈旧锁没有被替换")
		}
	}
}

//---------------------------------------------------------------------------
// git
//---------------------------------------------------------------------------

func TestGitLifecycle(t *testing.T) {
	if !gitAvailable() {
		t.Skip("环境里没有 git，跳过 git 相关断言")
	}
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	doc := makeDoc(pkg, fenced)
	ws, err := Create(filepath.Join(root, "ws"), doc, pkg, fenced)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if st, err := os.Stat(filepath.Join(ws.Dir, ".git")); err != nil || !st.IsDir() {
		t.Fatalf("Create 应当 git init：%v", err)
	}
	head, err := ws.GitHead()
	if err != nil {
		t.Fatalf("GitHead: %v", err)
	}
	if head == "" {
		t.Fatal("首次 commit 之后 GitHead 应非空")
	}
	if out := gitOutput(t, ws.Dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Fatalf("Create 之后工作区应当是干净的：%q", out)
	}

	// 干净树上再 commit：返回当前 HEAD + nil error
	head2, err := ws.GitCommit("second")
	if err != nil {
		t.Fatalf("干净树上 GitCommit 不应报错：%v", err)
	}
	if head2 != head {
		t.Fatalf("干净树 commit 应返回当前 HEAD：%q vs %q", head2, head)
	}

	// 改 prog.full.4gl 再 commit：hash 变化，树变干净
	if err := ws.WriteEdited([]byte("// changed\nFUNCTION aapp131_f1()\n  DISPLAY 2\nEND FUNCTION\n")); err != nil {
		t.Fatalf("WriteEdited: %v", err)
	}
	if out := gitOutput(t, ws.Dir, "status", "--porcelain"); strings.TrimSpace(out) == "" {
		t.Fatal("改动之后 status 应为脏")
	}
	head3, err := ws.GitCommit("third")
	if err != nil {
		t.Fatalf("第二次 commit: %v", err)
	}
	if head3 == "" || head3 == head {
		t.Fatalf("hash 应当变化：%q → %q", head, head3)
	}
	if out := gitOutput(t, ws.Dir, "status", "--porcelain"); strings.TrimSpace(out) != "" {
		t.Fatalf("commit 之后 status 应为空：%q", out)
	}
	if h, err := ws.GitHead(); err != nil || h != head3 {
		t.Fatalf("GitHead = %q, %v; want %q", h, err, head3)
	}
	// 身份不依赖全局配置：确认提交者是我们给的身份
	who := gitOutput(t, ws.Dir, "log", "-1", "--format=%an <%ae>")
	if strings.TrimSpace(who) != "tdev <tdev@local>" {
		t.Errorf("提交身份 = %q", strings.TrimSpace(who))
	}
}

//---------------------------------------------------------------------------
// 确定性
//---------------------------------------------------------------------------

func TestCreateDeterministic(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	doc := makeDoc(pkg, fenced)

	a, err := Create(filepath.Join(root, "a"), doc, pkg, fenced)
	if err != nil {
		t.Fatalf("Create(a): %v", err)
	}
	b, err := Create(filepath.Join(root, "b"), doc, pkg, fenced)
	if err != nil {
		t.Fatalf("Create(b): %v", err)
	}
	for _, rel := range []string{FileManifest, FileRegions, FileSnapshotIndex} {
		ba, err := os.ReadFile(filepath.Join(a.Dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		bb, err := os.ReadFile(filepath.Join(b.Dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(ba, bb) {
			t.Errorf("%s 在两次 Create 之间不一致（红线 R7：相同输入必须逐字节相同）", rel)
		}
	}
}

//---------------------------------------------------------------------------
// apply 后刷新
//---------------------------------------------------------------------------

func TestUpdateAfterApply(t *testing.T) {
	root := tempDir(t)
	pkg := makePkg(t, root)
	fenced := testFenced()
	doc := makeDoc(pkg, fenced)
	ws := mustCreate(t, pkg, doc, fenced, "ws")

	before, err := ws.Manifest()
	if err != nil {
		t.Fatal(err)
	}

	// apply 后：新基线 + 新的区间表（嵌套点被删掉、多了个 span）
	fenced2 := []byte("// tdev fenced document v2\nFUNCTION aapp131_f1()\n  DISPLAY 2\nEND FUNCTION\n")
	body := model.ByteRange{Start: 28, End: 60}
	doc2 := &model.Document{
		Text: fenced2,
		Prog: doc.Prog,
		Env:  doc.Env,
		Regions: []*model.Region{
			{
				Name: "function.aapp131_f1", Kind: model.RegionPoint, Editable: true,
				Meta:        map[string]string{"status": "u", "src": "c", "new": "N", "order": "1"},
				ContentSpan: model.ByteRange{Start: 25, End: 65},
				BodySpan:    &body,
				Sha256:      model.Sha256Bytes([]byte("v2")),
				Origin:      "placeholder",
			},
		},
		Spans: []model.Span{
			{Name: "prefix", Span: model.ByteRange{Start: 0, End: 25}, Sha256: model.Sha256Bytes(fenced2[:25])},
			{Name: "suffix", Span: model.ByteRange{Start: 70, End: len(fenced2)}, Sha256: model.Sha256Bytes(fenced2[70:])},
		},
	}
	if err := ws.UpdateAfterApply(doc2, fenced2); err != nil {
		t.Fatalf("UpdateAfterApply: %v", err)
	}

	base, err := ws.ReadBase()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(base, fenced2) {
		t.Fatal("基线没有刷新")
	}
	shaFile, err := os.ReadFile(filepath.Join(ws.Dir, FileBaseSha256))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(shaFile)) != model.Sha256Bytes(fenced2) {
		t.Fatal("base.sha256 没有刷新")
	}

	rf, err := ws.Regions()
	if err != nil {
		t.Fatal(err)
	}
	if rf.BaseSha256 != model.Sha256Bytes(fenced2) || rf.TextLen != len(fenced2) {
		t.Errorf("regions.json 基线没有刷新：%+v", rf)
	}
	if len(rf.Regions) != 1 || len(rf.Spans) != 2 {
		t.Errorf("regions.json 区间/spans 没有刷新：%d / %d", len(rf.Regions), len(rf.Spans))
	}

	after, err := ws.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(after.Regions) != 1 || len(after.Spans) != 2 {
		t.Errorf("manifest 的 regions/spans 没有刷新：%d / %d", len(after.Regions), len(after.Spans))
	}
	if after.Stats.Points != 1 || after.Stats.Sections != 0 || after.Stats.Editable != 1 || after.Stats.Readonly != 0 {
		t.Errorf("manifest.stats 没有刷新：%+v", after.Stats)
	}
	// pkg / entries / export / anchors 必须原样保留
	if after.Pkg != before.Pkg {
		t.Errorf("manifest.pkg 被改动：%+v → %+v", before.Pkg, after.Pkg)
	}
	if len(after.Entries) != len(before.Entries) {
		t.Errorf("manifest.entries 被改动")
	}
	if len(after.Export.Only) != len(before.Export.Only) {
		t.Errorf("manifest.export 被改动：%+v → %+v", before.Export, after.Export)
	}
	if len(after.Anchors) != len(before.Anchors) || !after.Anchors["other.function"] {
		t.Errorf("manifest.anchors 被改动：%+v", after.Anchors)
	}
	if after.Prog != before.Prog || after.Module != before.Module || after.Kind != before.Kind {
		t.Errorf("manifest 头部被改动")
	}
	// 编辑文件本身不被 UpdateAfterApply 改写
	edited, err := ws.ReadEdited()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(edited, fenced) {
		t.Errorf("UpdateAfterApply 不应改写 prog.full.4gl")
	}
}
