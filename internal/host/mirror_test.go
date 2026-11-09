package host

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMirrorEnvDirAndReady(t *testing.T) {
	dir := t.TempDir()
	if got := MirrorEnvDir(dir, "envA"); got != filepath.Join(dir, "envA") {
		t.Fatalf("MirrorEnvDir = %q", got)
	}
	if MirrorEnvDir("", "envA") != "" || MirrorEnvDir(dir, "") != "" {
		t.Fatal("空参数应返回空串")
	}
	if MirrorReady(dir, "envA") {
		t.Fatal("无基线标记应为 false")
	}
	if err := writeLocalMark(filepath.Join(dir, "envA"), MirrorSource); err != nil {
		t.Fatalf("writeLocalMark: %v", err)
	}
	if !MirrorReady(dir, "envA") {
		t.Fatal("有基线标记应为 true")
	}
	if !MirrorReadyDir(filepath.Join(dir, "envA")) {
		t.Fatal("MirrorReadyDir 应识别直接目录的基线标记")
	}
	if MirrorReady("", "envA") {
		t.Fatal("空镜像根应为 false")
	}
}

// countingReader 统计压缩字节,并在 EOF 时回调一次进度(小数据只触发收尾回调)。
func TestCountingReaderProgress(t *testing.T) {
	var got []MirrorProgress
	cr := &countingReader{
		r:      strings.NewReader("hello world"),
		total:  11,
		report: func(p MirrorProgress) { got = append(got, p) },
	}
	buf := make([]byte, 4)
	for {
		if _, err := cr.Read(buf); err != nil {
			break
		}
	}
	if cr.n != 11 {
		t.Fatalf("已读字节 = %d, want 11", cr.n)
	}
	if len(got) == 0 {
		t.Fatal("应至少回调一次进度")
	}
	last := got[len(got)-1]
	if last.Phase != "download" || last.Bytes != 11 || last.Total != 11 {
		t.Fatalf("末次进度 = %+v", last)
	}
}

// 备份/临时文件识别:用户反馈的实际样例 + 正常源码不应误判。
func TestIsBackupFile(t *testing.T) {
	backups := []string{
		"s_icd_wf_s02.4fd.bak", "s_icd_wf_s04.4fd.bak",
		"cs_aapp320.bck", "cs_aapp320.bck1", "cs_aapp320.bck2",
		"cs_axmt500_wf_excel.bck", "cs_axmt500_wf_excel.bck1", "cs_axmt500_wf_excel.bck2",
		"foo.4gl~", "foo.old", "foo.orig", "bar.tmp", "x.swp", "y.bak3",
	}
	for _, n := range backups {
		if !isBackupFile(n) {
			t.Errorf("应判为备份: %s", n)
		}
	}
	kept := []string{
		"s_icd_wf_s02.4fd", "s_icd_wf_s04.4fd",
		"cs_aapp320.4gl", "cs_axmt500_wf_excel.4gl",
		"apmt520_bak_service.4gl",          // 含 "_bak" 但不是备份文件
		"lib_common.inc", "b_sysparam.inc", // .inc 是源码,不是备份
		localMarkName, "dzea_t",
	}
	for _, n := range kept {
		if isBackupFile(n) {
			t.Errorf("不应判为备份: %s", n)
		}
	}
}

// 服务器侧 find 排除条件按同一份后缀列表生成。
func TestMirrorBackupFindExpr(t *testing.T) {
	expr := mirrorBackupFindExpr()
	for _, suf := range mirrorBackupSuffixes {
		if !strings.Contains(expr, `-not -iname '*`+suf+`*'`) {
			t.Errorf("排除条件缺少 %s: %s", suf, expr)
		}
	}
	if !strings.Contains(expr, `-not -iname '*~'`) {
		t.Errorf("排除条件缺少 ~: %s", expr)
	}
}

// 服务器侧 find 白名单：source 必须是“所有读者的并集”（4gl + 42f + 42s/zh_CN + *.inc）。
//
// 为什么这么硬：一个环境只有一个镜像目录，tt outline 只认 4gl、tt interface 只认
// 42f/42s，两者读同一个目录。source 少收一类，就有一个命令静默断粮 —— 它只会报
// “找不到文件”，看起来像文件不存在。这条用例就是钉住“不要再收紧回去”。
func TestMirrorWhitelistFindExpr(t *testing.T) {
	expr := mirrorWhitelistFindExpr(MirrorSource)
	for _, want := range []string{
		`-path '*/4gl/*'`, `-iname '*.4gl'`,
		`-path '*/42f/*'`,
		`-path '*/42s/zh_CN/*'`, `-path '*/42s/*/zh_CN/*'`,
		`-iname '*.inc'`,
	} {
		if !strings.Contains(expr, want) {
			t.Errorf("source 白名单缺少 %s: %s", want, expr)
		}
	}
	for _, bad := range []string{"4fd", ".src"} {
		if strings.Contains(expr, bad) {
			t.Errorf("source 白名单不应含 %s: %s", bad, expr)
		}
	}
}

func TestParseMirrorProfile(t *testing.T) {
	for in, want := range map[string]MirrorProfile{"": MirrorSource, "source": MirrorSource, "interface": MirrorInterface} {
		got, err := ParseMirrorProfile(in)
		if err != nil || got != want {
			t.Fatalf("ParseMirrorProfile(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ParseMirrorProfile("bogus"); err == nil {
		t.Fatal("未知 profile 应报错")
	}
}

func TestMirrorWhitelistInterfaceProfile(t *testing.T) {
	expr := mirrorWhitelistFindExpr(MirrorInterface)
	for _, want := range []string{`-path '*/42f/*'`, `-path '*/42s/zh_CN/*'`} {
		if !strings.Contains(expr, want) {
			t.Errorf("interface 白名单缺少 %s: %s", want, expr)
		}
	}
	for _, bad := range []string{"4fd", "4gl", ".inc"} {
		if strings.Contains(expr, bad) {
			t.Errorf("interface 白名单不应含 %s: %s", bad, expr)
		}
	}
}

// 基线标记记录白名单版本;旧格式(纯时间戳)与缺失都视为 0(触发自动全量)。
func TestLocalMarkVersion(t *testing.T) {
	dir := t.TempDir()
	if got := localMarkVersion(dir); got != 0 {
		t.Fatalf("无标记应为 0, got %d", got)
	}
	if err := writeLocalMark(dir, MirrorSource); err != nil {
		t.Fatal(err)
	}
	if got := localMarkVersion(dir); got != mirrorWhitelistVersion {
		t.Fatalf("版本 = %d, want %d", got, mirrorWhitelistVersion)
	}
	// 旧格式(仅 RFC3339 时间戳)
	legacy := t.TempDir()
	if err := os.WriteFile(filepath.Join(legacy, localMarkName), []byte("2026-01-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := localMarkVersion(legacy); got != 0 {
		t.Fatalf("旧格式应为 0, got %d", got)
	}
}

// 本地残留备份清理:只删备份,不碰源码与目录。
func TestPruneBackupFiles(t *testing.T) {
	dir := t.TempDir()
	mk := func(rel string) string {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	keepA := mk("erp/asf/4gl/cs_aapp320.4gl")
	keepB := mk("erp/asf/42f/s_icd_wf_s02.42f")
	mk("erp/asf/4gl/cs_aapp320.bck")
	mk("erp/asf/4gl/cs_aapp320.bck1")
	mk("erp/asf/42f/s_icd_wf_s02.42f.bak")
	mk("erp/asf/4gl/sub/cs_x.4gl~")

	if got := pruneBackupFiles(dir); got != 4 {
		t.Fatalf("应删除 4 个备份, got %d", got)
	}
	for _, p := range []string{keepA, keepB} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("源码不应被删: %s (%v)", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "erp/asf/4gl/cs_aapp320.bck")); err == nil {
		t.Error("备份应已删除")
	}
	// 幂等:再清一次为 0
	if got := pruneBackupFiles(dir); got != 0 {
		t.Fatalf("二次清理应为 0, got %d", got)
	}
	// 不存在目录不报错
	if got := pruneBackupFiles(filepath.Join(dir, "nope")); got != 0 {
		t.Fatalf("不存在目录应为 0, got %d", got)
	}
}

// 服务器 marker / 归档必须是**按 profile 分开**的。
//
// 否则：先用 source 拉一次，服务器 mark 被 touch 到最新；再按 interface 拉时
// `find -newer <marker>` 把 42f/42s 全当老文件排除 → 打包 0 个文件、还报"无变更"。
// 用户看到的是成功，实际一个文件都没拿到 —— 这是最难查的那种"静默空转"。
func TestMirrorBaseIsPerProfile(t *testing.T) {
	const top = "/u1/topprod"
	src := mirrorBase(top, "主机109", MirrorSource)
	ifc := mirrorBase(top, "主机109", MirrorInterface)
	if src == ifc {
		t.Fatalf("两个 profile 的 marker 路径不能相同: %s", src)
	}
	if !strings.HasPrefix(src, top+"/.tdict-mirror-") || !strings.HasSuffix(src, "-source") {
		t.Errorf("source marker 路径形状不对: %s", src)
	}
	if !strings.HasSuffix(ifc, "-interface") {
		t.Errorf("interface marker 路径形状不对: %s", ifc)
	}
	// 空 profile 按 source 处理（与 ParseMirrorProfile 的约定一致）
	if got := mirrorBase(top, "主机109", ""); got != src {
		t.Errorf("空 profile 应等同 source, got %s", got)
	}
	// 环境名不同 → 路径不同（避免同服务器多环境互相覆盖）
	if mirrorBase(top, "正式环境", MirrorSource) == src {
		t.Error("不同环境应得到不同的 marker 路径")
	}
}

// 本地基线标记记录"这份目录拉过哪些 profile"，且取并集：
// 先后拉两种 profile 后，记录里两种都在（否则"--full 会不会清掉另一类"之类的判断会看错）。
func TestLocalMarkProfilesUnion(t *testing.T) {
	dir := t.TempDir()
	if got := readLocalMark(dir); got.Version != 0 || got.Has(MirrorSource) || got.Has(MirrorInterface) {
		t.Fatalf("无标记时应是零值, got %+v", got)
	}
	if err := writeLocalMark(dir, MirrorSource); err != nil {
		t.Fatal(err)
	}
	m := readLocalMark(dir)
	if !m.Has(MirrorSource) || m.Has(MirrorInterface) {
		t.Fatalf("只拉过 source, got %s", m.ProfilesLabel())
	}
	if err := writeLocalMark(dir, MirrorInterface); err != nil {
		t.Fatal(err)
	}
	m = readLocalMark(dir)
	if !m.Has(MirrorSource) || !m.Has(MirrorInterface) {
		t.Fatalf("两种都拉过应都记下, got %s", m.ProfilesLabel())
	}
	if m.Version != mirrorWhitelistVersion {
		t.Fatalf("版本 = %d, want %d", m.Version, mirrorWhitelistVersion)
	}
	// 旧版本的标记（版本不同）读出来的 profile 集合不作数：白名单变了，那一类可能得重拉
	old := t.TempDir()
	if err := os.WriteFile(filepath.Join(old, localMarkName), []byte("v3 source 2026-01-01T00:00:00Z\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readLocalMark(old); got.Version != 3 || !got.Has(MirrorSource) {
		t.Fatalf("旧标记应能读出 v3/source, got %+v", got)
	}
	if err := writeLocalMark(old, MirrorInterface); err != nil {
		t.Fatal(err)
	}
	if got := readLocalMark(old); got.Has(MirrorSource) || !got.Has(MirrorInterface) {
		t.Fatalf("版本升级后旧 profile 记录应被清掉, got %s", got.ProfilesLabel())
	}
}

// 本地分类器与服务器侧 find 表达式必须同源：表达式里出现的每一类，分类器都要认。
//
// 分类器只在"全量解压"里用：属于本 profile 的文件以服务器为准，不属于的原样保留。
// 两者不一致的后果很具体 —— 要么该保留的被删（另一类文件消失），要么该删的留着（镜像过期）。
func TestMirrorPathInProfileMatchesFindExpr(t *testing.T) {
	cases := []struct {
		profile MirrorProfile
		token   string // find 表达式里必须出现的片段
		path    string // 分类器必须认的本地相对路径
	}{
		{MirrorSource, `-iname '*.4gl'`, "erp/asf/4gl/cs_aapp320.4gl"},
		{MirrorSource, `-path '*/42f/*'`, "erp/asf/42f/s_aapp320.42f"},
		{MirrorSource, `-path '*/42s/zh_CN/*'`, "erp/asf/42s/zh_CN/s_aapp320.42s"},
		{MirrorSource, `-path '*/42s/*/zh_CN/*'`, "erp/asf/42s/sub/zh_CN/s_aapp320.42s"},
		{MirrorSource, `-iname '*.inc'`, "com/cfg/top_global.inc"},
		{MirrorSource, `-iname '*.inc'`, "erp/asf/4g1/SSSQ999_01.inc"},
		{MirrorInterface, `-path '*/42f/*'`, "erp/asf/42f/s_aapp320.42f"},
		{MirrorInterface, `-path '*/42s/zh_CN/*'`, "erp/asf/42s/zh_CN/s_aapp320.42s"},
		{MirrorInterface, `-path '*/42s/*/zh_CN/*'`, "erp/asf/42s/sub/zh_CN/s_aapp320.42s"},
	}
	for _, c := range cases {
		if expr := mirrorWhitelistFindExpr(c.profile); !strings.Contains(expr, c.token) {
			t.Errorf("%s 白名单里没有 %s: %s", c.profile, c.token, expr)
		}
		if !mirrorPathInProfile(c.profile, c.path) {
			t.Errorf("分类器(%s)不认 %s —— 全量重建时它会被当成「别人的文件」保留下来", c.profile, c.path)
		}
	}
	// 白名单外的文件：source 与 interface 都不认（保留它们才是对的）
	outside := []string{
		"erp/asf/4gl/cs_aapp320.4gl.src",
		"erp/asf/4gl/cs_aapp320.4fd",
		"erp/asf/42s/en_US/s_aapp320.42s", // 非 zh_CN 语言目录
		"erp/asf/4gl/cs_aapp320.bak",
		"erp/asf/4gl/notes.txt",
	}
	for _, p := range outside {
		if mirrorPathInProfile(MirrorSource, p) {
			t.Errorf("source 分类器不该认 %s", p)
		}
	}
	for _, p := range []string{"erp/asf/4gl/cs_aapp320.4gl", "com/cfg/top_global.inc"} {
		if mirrorPathInProfile(MirrorInterface, p) {
			t.Errorf("interface 分类器不该认 %s", p)
		}
	}
}

// makeTarGz 造一份内存里的 tar.gz（模拟服务器归档），供全量解压用例使用。
func makeTarGz(t *testing.T, files map[string]string) io.Reader {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(1700000000, 0),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(buf.Bytes())
}

// 全量重建只对**本 profile 的白名单**负责：不属于它的既有文件必须原样保留。
//
// 这条钉的是"某次更新后镜像只剩 4gl"的直接成因：旧实现把旧目录整个删掉，
// 于是"按 source 做一次全量"会把已经拉好的 42f/42s（以及 source 不收的其它类型）
// 一起删掉。全量的正确语义是"本 profile 内以服务器为准"，不是"清空目录"。
func TestExtractFullKeepsOtherProfilesFiles(t *testing.T) {
	mkFile := func(root, rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }

	// 情况一：source 全量 —— source 白名单内的旧文件消失（服务器已删），
	// 白名单外的（per 目录的 .4gl、非 zh_CN 的 42s）保留，备份文件不被搬。
	envDir := filepath.Join(t.TempDir(), "envA")
	mkFile(envDir, "erp/asf/4gl/old_gone.4gl", "old")
	mkFile(envDir, "erp/asf/4gl/x.bak", "bak")
	mkFile(envDir, "erp/asf/per/only_per.4gl", "per")
	mkFile(envDir, "erp/asf/42s/en_US/en.42s", "en")
	mkFile(envDir, "erp/asf/42f/kept.42f", "42f") // source 白名单内 → 但归档里没有 → 应消失
	mkFile(envDir, localMarkName, "v5 source 2026-10-26T00:00:00Z\n")

	n, err := extractFull(makeTarGz(t, map[string]string{
		"erp/asf/4gl/new_one.4gl": "new",
	}), envDir, MirrorSource)
	if err != nil {
		t.Fatalf("extractFull: %v", err)
	}
	if n != 1 {
		t.Errorf("归档文件数 = %d, want 1", n)
	}
	if !exists(filepath.Join(envDir, "erp/asf/4gl/new_one.4gl")) {
		t.Error("归档里的文件应落盘")
	}
	for _, rel := range []string{"erp/asf/4gl/old_gone.4gl", "erp/asf/42f/kept.42f"} {
		if exists(filepath.Join(envDir, rel)) {
			t.Errorf("source 白名单内的旧文件应随全量消失（服务器已删）: %s", rel)
		}
	}
	for _, rel := range []string{"erp/asf/per/only_per.4gl", "erp/asf/42s/en_US/en.42s"} {
		if !exists(filepath.Join(envDir, rel)) {
			t.Errorf("不属于本 profile 的文件应原样保留: %s", rel)
		}
	}
	if exists(filepath.Join(envDir, "erp/asf/4gl/x.bak")) {
		t.Error("备份文件不该被搬进新目录")
	}
	if exists(filepath.Join(envDir, localMarkName)) {
		t.Error("基线标记不该被搬（解压完成后由 writeLocalMark 重写）")
	}
	// 临时目录不该留在镜像父目录里
	entries, err := os.ReadDir(filepath.Dir(envDir))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".staging-") || strings.Contains(e.Name(), ".old-") {
			t.Errorf("临时目录未清理: %s", e.Name())
		}
	}

	// 情况二：interface 全量 —— 4gl/inc 不属于它，必须留着（否则 tt outline 断粮）
	envDir2 := filepath.Join(t.TempDir(), "envB")
	mkFile(envDir2, "erp/asf/4gl/keep_outline.4gl", "4gl")
	mkFile(envDir2, "com/cfg/top_global.inc", "inc")
	if _, err := extractFull(makeTarGz(t, map[string]string{
		"erp/asf/42f/s_new.42f":       "42f",
		"erp/asf/42s/zh_CN/s_new.42s": "42s",
	}), envDir2, MirrorInterface); err != nil {
		t.Fatalf("extractFull(interface): %v", err)
	}
	for _, rel := range []string{"erp/asf/4gl/keep_outline.4gl", "com/cfg/top_global.inc"} {
		if !exists(filepath.Join(envDir2, rel)) {
			t.Errorf("interface 全量不该删掉 4gl/inc: %s", rel)
		}
	}
	for _, rel := range []string{"erp/asf/42f/s_new.42f", "erp/asf/42s/zh_CN/s_new.42s"} {
		if !exists(filepath.Join(envDir2, rel)) {
			t.Errorf("interface 归档文件应落盘: %s", rel)
		}
	}
}
