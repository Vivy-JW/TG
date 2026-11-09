package store

import (
	"os"
	"path/filepath"
	"testing"

	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/testutil"
)

// buildWorkspace 造一个最小可用工作区（只为本文件的测试服务）。
func buildWorkspace(t *testing.T, dir string) (*Workspace, string) {
	t.Helper()
	pkgPath, err := testutil.NormalPkgPath(filepath.Join(dir, "src"), "adzi999")
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pkgfile.Open(pkgPath, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	doc := &model.Document{
		Text: []byte("fenced body\n"),
		Regions: []*model.Region{{
			Name: "function.adzi999_calc", Kind: model.RegionPoint, Editable: true,
			ContentSpan: model.ByteRange{Start: 0, End: 11},
			FullSpan:    model.ByteRange{Start: 0, End: 11},
			Meta:        map[string]string{"src": "s", "new": "Y"},
		}},
		Env:          model.EnvContext{Env: "c", Topind: "sd", LoginUser: "tiptop"},
		Prog:         "adzi999",
		SectionState: model.SectionLocked,
	}
	ws, err := Create(filepath.Join(dir, "ws"), doc, pkg, doc.Text)
	if err != nil {
		t.Fatal(err)
	}
	return ws, pkgPath
}

// TestRefreshPackageUpdatesRecordedPkg 钉住一次真实事故：
//
//	apply 成功后若 manifest 仍记着**上一版**包的 sha256，
//	同一个工作区的第二次 apply 会被 D-6 的「源包自 export 之后已被改动」误拒（退出码 5），
//	即「改一次→apply→再改→再 apply」的正常迭代循环会断掉。
func TestRefreshPackageUpdatesRecordedPkg(t *testing.T) {
	dir := t.TempDir()
	ws, pkgPath := buildWorkspace(t, dir)

	before, err := ws.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	if before.Pkg.Sha256 == "" {
		t.Fatal("导出的 manifest 应记录包 sha256")
	}

	// 模拟 apply 写出新包：把 .tap 条目改一点内容，重建成新包文件
	pkg, err := pkgfile.Open(pkgPath, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tap := append([]byte(nil), pkg.Tap().Data...)
	tap = append(tap, []byte("\n<!-- tdev test -->\n")...)
	newBytes, _, err := pkg.Build(pkgfile.Rebuild{Tap: tap})
	if err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(dir, "applied.tzc")
	if err := os.WriteFile(newPath, newBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	newPkg, err := pkgfile.Open(newPath, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.RefreshPackage(newPkg); err != nil {
		t.Fatal(err)
	}

	after, err := ws.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	wantSha, _ := model.Sha256File(newPath)
	if after.Pkg.Sha256 != wantSha {
		t.Errorf("manifest.pkg.sha256 未刷新：%s → 想要 %s", after.Pkg.Sha256, wantSha)
	}
	if after.Pkg.Path != newPath {
		t.Errorf("manifest.pkg.path 未刷新：%s", after.Pkg.Path)
	}
	if after.Pkg.Sha256 == before.Pkg.Sha256 {
		t.Errorf("sha256 没变，测试自身失效")
	}
	// snapshot 与 entries 必须跟着换到新包
	if len(after.Entries) != len(newPkg.Entries) {
		t.Fatalf("entries 条数不对：%d → %d", len(after.Entries), len(newPkg.Entries))
	}
	snapped, err := ws.SnapshotEntry(newPkg.Tap().Name)
	if err != nil {
		t.Fatal(err)
	}
	if string(snapped) != string(newPkg.Tap().Data) {
		t.Errorf("snapshot 未刷新到新包的 .tap")
	}
	// 解锁状态不能被 apply 悄悄改掉（包级事实才是权威）
	if after.Section.State != before.Section.State {
		t.Errorf("apply 不应改变解锁状态：%s → %s", before.Section.State, after.Section.State)
	}
	// 导出范围（--only）原样保留
	if len(after.Export.Only) != len(before.Export.Only) {
		t.Errorf("apply 不应改变导出范围")
	}
}

// TestBackupPackageKeepsPrevious 覆盖前必须留下「上一步」的包副本。
func TestBackupPackageKeepsPrevious(t *testing.T) {
	dir := t.TempDir()
	ws, pkgPath := buildWorkspace(t, dir)
	orig, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	bp, err := ws.BackupPackage(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(bp)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(orig) {
		t.Errorf("备份内容与原包不一致")
	}
	// 原包被覆盖后，备份仍是旧字节
	if err := os.WriteFile(pkgPath, []byte("overwritten"), 0o644); err != nil {
		t.Fatal(err)
	}
	got2, _ := os.ReadFile(bp)
	if string(got2) != string(orig) {
		t.Errorf("原包被覆盖后备份不应变化")
	}
}
