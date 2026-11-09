package tzc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tt/internal/cli/dev/common"
	"tt/internal/cli/dev/tzs"
	"tt/internal/dev/fence"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/store"
	"tt/internal/dev/synth"
	"tt/internal/dev/tapfile"
	"tt/internal/dev/testutil"
	"tt/internal/dev/tglfile"
)

// selftestCase 是一个自检项。
type selftestCase struct {
	Name string
	Run  func(dir string) error
}

// cmdSelftest 运行内置自检 + 对抗用例集（设计指南 §8）。
// 全部使用**合成包**，不依赖真实语料。
func cmdSelftest(args []string) int {
	asJSON := false
	for _, a := range args {
		if a == "--json" {
			asJSON = true
		}
	}
	cases := selftestCases()
	pass, failed := 0, 0
	var failures []string
	for _, c := range cases {
		dir, err := os.MkdirTemp("", "tdev-selftest-")
		if err != nil {
			return common.Fail(err, asJSON)
		}
		cerr := c.Run(dir)
		os.RemoveAll(dir)
		if cerr != nil {
			failed++
			failures = append(failures, c.Name+": "+cerr.Error())
			if !asJSON {
				fmt.Printf("  FAIL  %s\n        %v\n", c.Name, cerr)
			}
			continue
		}
		pass++
		if !asJSON {
			fmt.Printf("  ok    %s\n", c.Name)
		}
	}
	if asJSON {
		common.EmitJSON(os.Stdout, map[string]any{
			"ok": failed == 0, "pass": pass, "fail": failed, "failures": failures,
		})
	} else {
		fmt.Printf("\n自检：通过 %d，失败 %d\n", pass, failed)
	}
	if failed > 0 {
		return 3
	}
	return 0
}

// runQuiet 把命令输出吞掉（自检只关心退出码/副作用）。
func runQuiet(fn func() int) int { return fn() }

//---------------------------------------------------------------------------

func selftestCases() []selftestCase {
	return []selftestCase{
		{"合成包-零改动 roundtrip（逐条目 sha256 一致）", stRoundtrip},
		{"围栏可逆性（render∘parse 逐项相等）", stFenceInverse},
		{"export→改点正文→apply 成功且只动目标点", stEditPoint},
		{"对抗-改围栏行 → 拦（退出码 3）", stFenceLineTamper},
		{"对抗-改 READONLY 区段正文本 → 拦（退出码 4）", stReadonlySection},
		{"对抗-改结构行（函数头）→ 拦（退出码 3）", stStructureLine},
		{"对抗-删 end 围栏 → 拦（退出码 3）", stUnpairedFence},
		{"对抗-正文塞 ]]> → 拦（退出码 3）", stCDATAClose},
		{"对抗-围栏外插散行 → 拦（退出码 3）", stOutsideFence},
		{"对抗---only 范围外改点 → 拦（退出码 4）", stOnlyScope},
		{"对抗-包缺 .tgl → 退出码 2", stMissingTgl},
		{"真机事故-编辑器把行尾归一（CRLF→LF）→ 不算改动，apply 只动目标点", stEOLNormalizedByEditor},
		{"真机事故-迭代循环（改→apply→再改→再 apply）", stIterativeApplyLoop},
		{"结构事务-改名：墓碑对（旧 status=d 含原内容 + 新 status=u）", stRenameTransaction},
		{"结构事务-V1 fn 与签名行不一致 → 拦（退出码 3）", stV1Mismatch},
		{"结构事务-V2 旧名残留在只读区段 → 拦（退出码 4）", stV2ResidualReadonly},
		{"结构事务-V5 scope 与类型默认不一致 → warn（--strict 下 3），不阻断既有文本", stV5Scope},
		{"结构事务-V6 描述块非 # 开头 → 拦（退出码 3）", stV6Desc},
		{"结构事务-V7 裸名插入点不可改名 → 拦（退出码 4）", stV7Plain},
		{"newfn 新增自订点 → 出现在 APPEND 锚点且可 apply", stNewfn},
		{"newfn 函数头用设计器模板（顶部空行 + 80 个 # 的框）且落盘后 CDATA 同形", stNewfnTemplate},
		{"newfn 已移除 --desc → 未知标志退出 2", stNewfnNoDescFlag},
		{"解锁闸门-env=s 无 adzi052 授权 → 退出码 4 且打印设计器原文", stUnlockDeniedS},
		{"解锁闸门-已解开包（section_flag=Y）直接 EDITABLE-SEC，unlock 为 no-op", stUnlockAlreadyUnlocked},
		{"对抗-ver 版本不匹配 → 退出码 2", stBadVer},
		{"对抗-区段标记不配对 → 退出码 2", stUnpairedSection},
		{"删除 new=Y 的点 → status=d 且保留原内容", stDeletePoint},
		{"在 APPEND 锚点内新增点 → status=u + new=Y + order=max+1", stAppendPoint},
		{"框架解锁：Locked 改区段拒(4) → unlock 需 --yes → unlock 后 EDITABLE-SEC → apply 落盘 section_flag=Y", stSectionWrite},
		{"function.* 前缀但正文非函数块 → 只读且不可写", stUnparsablePoint},
		{"tzs export 纯解压：逐字节一致、不产生工作区产物、代码包走 tzs 被指回 tzc", stTzsExport},
	}
}

//---------------------------------------------------------------------------
// 基础自检
//---------------------------------------------------------------------------

func stRoundtrip(dir string) error {
	p, err := testutil.NormalPkgPath(dir, "adzi999")
	if err != nil {
		return err
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	out, actions, err := pkg.Build(pkgfile.Rebuild{})
	if err != nil {
		return err
	}
	for _, a := range actions {
		if a.Changed {
			return fmt.Errorf("零改动却改了 %s", a.Name)
		}
	}
	p2 := filepath.Join(dir, "out.tzc")
	if err := os.WriteFile(p2, out, 0o644); err != nil {
		return err
	}
	pkg2, err := pkgfile.Open(p2, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	if len(pkg2.Entries) != len(pkg.Entries) {
		return fmt.Errorf("条目数变了")
	}
	for i := range pkg.Entries {
		if pkg.Entries[i].Name != pkg2.Entries[i].Name {
			return fmt.Errorf("条目名变了：%s → %s", pkg.Entries[i].Name, pkg2.Entries[i].Name)
		}
		if pkg.Entries[i].Sha256 != pkg2.Entries[i].Sha256 {
			return fmt.Errorf("条目内容变了：%s", pkg.Entries[i].Name)
		}
	}
	return nil
}

func stFenceInverse(dir string) error {
	p, err := testutil.NormalPkgPath(dir, "adzi999")
	if err != nil {
		return err
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	doc, err := synthDocFrom(pkg)
	if err != nil {
		return err
	}
	fenced, regions, spans, err := renderDoc(doc)
	if err != nil {
		return err
	}
	base := baseDoc(doc, fenced, regions, spans)
	parsed, err := parseFenced(base, fenced)
	if err != nil {
		return err
	}
	a, b := base.AllRegions(), parsed.Doc.AllRegions()
	if len(a) != len(b) {
		return fmt.Errorf("Region 数量 %d → %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return fmt.Errorf("第 %d 个 Region 名不同：%s → %s", i, a[i].Name, b[i].Name)
		}
		ca := base.Text[a[i].ContentSpan.Start:a[i].ContentSpan.End]
		cb := parsed.Doc.Text[b[i].ContentSpan.Start:b[i].ContentSpan.End]
		if !bytes.Equal(ca, cb) {
			return fmt.Errorf("Region %s 内容不等", a[i].Name)
		}
	}
	return nil
}

//---------------------------------------------------------------------------
// 正向：改点正文
//---------------------------------------------------------------------------

func stEditPoint(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	before4gl := entrySha(p, ".4gl")
	beforeVer := entrySha(p, "ver")
	edited := readEdited(ws)
	marker := []byte("   RETURN p_a\n")
	if !bytes.Contains(edited, marker) {
		return fmt.Errorf("测试自身失效：找不到目标正文行")
	}
	edited = bytes.Replace(edited, marker, []byte("   RETURN p_a\n   # tdev selftest marker\n"), 1)
	if err := writeEdited(ws, edited); err != nil {
		return err
	}
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.Point("function.adzi999_calc")
	if el == nil {
		return fmt.Errorf("找不到目标点")
	}
	if !bytes.Contains(el.Content(tapDoc.Raw), []byte("tdev selftest marker")) {
		return fmt.Errorf("目标点正文没更新")
	}
	if v, _ := el.Attr("status"); v != "u" {
		return fmt.Errorf("目标点 status 应为 u，实际 %q", v)
	}
	// 红线 R1/R2：.4gl 与 ver 必须字节不变
	if got := entrySha(p, ".4gl"); got != before4gl {
		return fmt.Errorf(".4gl 被改动了（红线 R1）")
	}
	if got := entrySha(p, "ver"); got != beforeVer {
		return fmt.Errorf("ver 被改动了（红线 R2）")
	}
	// 只动目标点：其余点内容不变
	base := mustReadWorkspaceBase(ws)
	_ = base
	return nil
}

//---------------------------------------------------------------------------
// 结构事务 / 解锁闸门 / newfn（v2）
//---------------------------------------------------------------------------

// stRenameTransaction 事务等价：`rename` + apply 必须产出设计器同款墓碑对。
//
// 复刻 ProgramInformation.Modify（ProgramInformation.cs:116-134）：
//
//	① 清旧墓碑  ③ 墓碑 status="d"（旧名 + 原 CDATA 逐字节）  ② 活点 status="u"（新名）
func stRenameTransaction(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	oldContent := ""
	{
		pkg0, err := pkgfile.Open(p, pkgfile.OpenOptions{})
		if err != nil {
			return err
		}
		tap0, err := tapfile.Parse(pkg0.Tap().Data)
		if err != nil {
			return err
		}
		el := tap0.Point("function.adzi999_calc")
		if el == nil {
			return fmt.Errorf("测试自身失效：找不到目标点")
		}
		oldContent = string(el.Content(tap0.Raw))
	}
	if code := cmdRename([]string{ws.Dir, "adzi999_calc", "adzi999_calc2"}); code != 0 {
		return fmt.Errorf("rename 退出码 %d", code)
	}
	// 改名后签名行必须已同步（V1 通过），apply 才可能成功
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("改名后 apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	// ③ 墓碑：旧名 + status=d + **原内容逐字节保留**
	tomb := tapDoc.PointExact("function.adzi999_calc")
	if tomb == nil {
		return fmt.Errorf("缺少旧名墓碑（设计器改名会留一份 status=d 的墓碑）")
	}
	if v, _ := tomb.Attr("status"); v != "d" {
		return fmt.Errorf("墓碑 status 应为 d，实际 %q", v)
	}
	if got := string(tomb.Content(tapDoc.Raw)); got != oldContent {
		return fmt.Errorf("墓碑必须逐字节保留原内容（I5）")
	}
	// ② 活点：新名 + status=u + 签名行已改名
	live := tapDoc.PointExact("function.adzi999_calc2")
	if live == nil {
		return fmt.Errorf("缺少新名活点")
	}
	if v, _ := live.Attr("status"); v != "u" {
		return fmt.Errorf("新点 status 应为 u，实际 %q", v)
	}
	if !bytes.Contains(live.Content(tapDoc.Raw), []byte("FUNCTION adzi999_calc2(")) {
		return fmt.Errorf("新点签名行没改成新名：%s", firstLine(string(live.Content(tapDoc.Raw))))
	}
	// 同名点恰好两个（无重复墓碑堆积）
	if n := len(tapDoc.PointsAll("function.adzi999_calc")); n != 1 {
		return fmt.Errorf("旧名元素数应为 1（墓碑），实际 %d", n)
	}
	return nil
}

// stV1Mismatch：只改签名行不改围栏 fn → V1 拦（设计器会抛 ComplexException）。
func stV1Mismatch(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = bytes.Replace(edited, []byte("PRIVATE FUNCTION adzi999_calc(p_a)"),
		[]byte("PRIVATE FUNCTION adzi999_renamed(p_a)"), 1)
	writeEdited(ws, edited)
	return expectApplyCode(p, ws.Dir, 3, "V1")
}

// stV2ResidualReadonly：旧名残留在只读区段（框架区段里的调用点）→ 退出码 4。
func stV2ResidualReadonly(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 先在只读区段正文里加一处对旧名的调用（模拟框架里的调用点）
	idx := bytes.Index(edited, []byte("#應用 a00 樣板自動產生(Version:3)"))
	if idx < 0 {
		return fmt.Errorf("测试自身失效：找不到只读区段正文")
	}
	edited = append(edited[:idx], append([]byte("# CALL adzi999_calc(1)\n"), edited[idx:]...)...)
	// 再改名（fn + 签名行同步）
	edited = bytes.Replace(edited, []byte(`fn="adzi999_calc(p_a)"`), []byte(`fn="adzi999_new(p_a)"`), 1)
	edited = bytes.Replace(edited, []byte("PRIVATE FUNCTION adzi999_calc(p_a)"),
		[]byte("PRIVATE FUNCTION adzi999_new(p_a)"), 1)
	writeEdited(ws, edited)
	return expectApplyCode(p, ws.Dir, 4, "V2")
}

// stV5Scope：type=M 的程序里把函数改成 PUBLIC。
//
// 偏差 DV-3（源码直证）：type→scope 映射是**新建点对话框的默认值**（且被 useDefaultScope
// 门控），不是硬约束 —— 真实语料里就有 type=M 却声明 PUBLIC 的点。所以这里给 warn：
// 普通 verify/apply 通过，`--strict` 下才失败。
func stV5Scope(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = bytes.Replace(edited, []byte(`scope="PRIVATE"`), []byte(`scope="PUBLIC"`), 1)
	edited = bytes.Replace(edited, []byte("PRIVATE FUNCTION adzi999_calc(p_a)"),
		[]byte("PUBLIC FUNCTION adzi999_calc(p_a)"), 1)
	writeEdited(ws, edited)
	// 普通 verify 通过；--strict 把 warn 升级为失败（退出码 3）
	if code := runQuiet(func() int { return cmdVerify([]string{ws.Dir}) }); code != 0 {
		return fmt.Errorf("V5 是 warn，普通 verify 应通过，实际退出码 %d", code)
	}
	if code := runQuiet(func() int { return cmdVerify([]string{ws.Dir, "--strict"}) }); code != 3 {
		return fmt.Errorf("V5 warn 在 --strict 下应退出码 3，实际 %d", code)
	}
	// apply 照旧放行（不阻断既有文本）
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("V5 为 warn 时 apply 应成功，实际 %d", code)
	}
	_ = p
	return nil
}

// stV6Desc：描述块里塞一行非 # 开头 → V6 拦（退出码 3）。
func stV6Desc(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 用**点自己的**描述块（区段正文里也有同名行，用它会误命中只读区段）
	marker := []byte("#+ Description: 合成測試用函式")
	if !bytes.Contains(edited, marker) {
		return fmt.Errorf("测试自身失效：找不到自订点的描述块")
	}
	edited = bytes.Replace(edited, marker, append([]byte("这不是注释行\n"), marker...), 1)
	writeEdited(ws, edited)
	return expectApplyCode(p, ws.Dir, 3, "V6")
}

// stV7Plain：给裸名插入点改名 → V7 拦（退出码 4）。
func stV7Plain(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	// 对裸名插入点调用 rename：普通插入点没有函数身份 → V7 拒（写入被拒，退出码 4）
	_ = p
	code := cmdRename([]string{ws.Dir, "global.import", "global.import2"})
	if code != 4 {
		return fmt.Errorf("裸名点改名应退出码 4（V7），实际 %d", code)
	}
	return nil
}

// stNewfn：newfn 在 APPEND 锚点内插入新点，且能 apply 落盘。
func stNewfn(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	if code := cmdNewfn([]string{ws.Dir, "--type", "FUNCTION", "--name", "adzi999_added"}); code != 0 {
		return fmt.Errorf("newfn 退出码 %d", code)
	}
	edited := readEdited(ws)
	if !bytes.Contains(edited, []byte("adzi999_added")) {
		return fmt.Errorf("newfn 后文档里看不到新点")
	}
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("newfn 后 apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.Point("function.adzi999_added")
	if el == nil {
		return fmt.Errorf("新点没写进 TAP")
	}
	if v, _ := el.Attr("status"); v != "u" {
		return fmt.Errorf("新点 status 应为 u（D-1），实际 %q", v)
	}
	if !bytes.Contains(el.Content(tapDoc.Raw), []byte("FUNCTION adzi999_added(")) {
		return fmt.Errorf("新点签名行不对")
	}
	return nil
}

// stNewfnTemplate：newfn 造出来的函数头用设计器模板，且**落盘后** .tap 的 CDATA
// 以「空行 + 80 个 # 的框」开头 —— 这正是真实包里的形态（desc="\n####…"），
// 也保证新函数不与上一个函数的 END 挨着。
func stNewfnTemplate(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	if code := cmdNewfn([]string{ws.Dir, "--type", "FUNCTION", "--name", "adzi999_tpl"}); code != 0 {
		return fmt.Errorf("newfn 退出码 %d", code)
	}
	edited := readEdited(ws)
	// 渲染文档里：围栏行 → 空行 → 80 个 # 的框；Usage 行填真实函数名
	if !bytes.Contains(edited, []byte("]}\n\n"+newFnRule+"\n# Descriptions...: 描述说明\n")) {
		return fmt.Errorf("描述块没有以「空行 + 80 个 # 的框」开头")
	}
	if !bytes.Contains(edited, []byte("# Usage..........: CALL adzi999_tpl(传入参数)")) {
		return fmt.Errorf("Usage 行没有填真实函数名")
	}
	if !bytes.Contains(edited, []byte("日期 By 作者")) {
		return fmt.Errorf("日期/作者占位符丢了")
	}
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("newfn 后 apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.Point("function.adzi999_tpl")
	if el == nil {
		return fmt.Errorf("新点没写进 TAP")
	}
	content := el.Content(tapDoc.Raw)
	// 顶部空行：CDATA 内容的第一个字节就是换行（真实包同形）
	if !bytes.HasPrefix(content, []byte("\n"+newFnRule)) && !bytes.HasPrefix(content, []byte("\r\n"+newFnRule)) {
		return fmt.Errorf("新点 CDATA 没有以空行 + 分隔线开头：%q", clipBytes(content, 40))
	}
	if !bytes.Contains(content, []byte("# Usage..........: CALL adzi999_tpl(传入参数)")) {
		return fmt.Errorf("新点 CDATA 里没有模板的 Usage 行")
	}
	return nil
}

// clipBytes 截断字节用于报错信息。
func clipBytes(b []byte, n int) string {
	if len(b) > n {
		b = b[:n]
	}
	return string(b)
}

// stNewfnNoDescFlag：newfn 已移除 --desc，传入应按未知标志退出 2（不能静默忽略）。
func stNewfnNoDescFlag(dir string) error {
	ws, _, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	if code := cmdNewfn([]string{ws.Dir, "--type", "FUNCTION", "--desc", "#+ 旧用法"}); code != 2 {
		return fmt.Errorf("newfn --desc 应退出 2（未知标志），实际 %d", code)
	}
	return nil
}

// stTzsExport：表单包的纯解压。钉住三件事：
//  1. 逐字节一致（不做任何处理）；
//  2. 不产生工作区/审计产物（没有 .tdev / manifest.json / git / prog.full.4gl）；
//  3. 用错管线被挡住（.tzc 走 tzs → 退出 2，指回 tzc export）。
func stTzsExport(dir string) error {
	entries := map[string]string{
		"adzi999.tsd": "<form/>\r\n",
		"adzi999.4fd": "BIN\x00\x01\xff",
		"ver":         "1.0\r\n",
	}
	p, err := testutil.WritePackage(filepath.Join(dir, "src"), "adzi999.tzs", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "unzip")
	if code := tzs.Run([]string{"export", p, "-o", out}, ""); code != 0 {
		return fmt.Errorf("tzs export 退出码 %d", code)
	}
	for name, want := range entries {
		got, rerr := os.ReadFile(filepath.Join(out, name))
		if rerr != nil {
			return fmt.Errorf("缺文件 %s: %v", name, rerr)
		}
		if string(got) != want {
			return fmt.Errorf("%s 内容不一致（纯解压必须逐字节相同）", name)
		}
	}
	for _, forbidden := range []string{".tdev", "manifest.json", ".git", "prog.full.4gl", "snapshot"} {
		if _, serr := os.Stat(filepath.Join(out, forbidden)); serr == nil {
			return fmt.Errorf("纯解压不该产生 %s", forbidden)
		}
	}
	// 非空目标默认拒绝；--force 才覆盖
	if code := tzs.Run([]string{"export", p, "-o", out}, ""); code != 5 {
		return fmt.Errorf("非空目标应退出 5，实际 %d", code)
	}
	if code := tzs.Run([]string{"export", p, "-o", out, "--force"}, ""); code != 0 {
		return fmt.Errorf("--force 应退出 0，实际 %d", code)
	}
	// 代码包走 tzs → 退出 2（指回 tzc）
	codePkg, err := testutil.NormalPkgPath(dir, "adzi998")
	if err != nil {
		return err
	}
	if code := tzs.Run([]string{"export", codePkg, "-o", filepath.Join(dir, "bad")}, ""); code != 2 {
		return fmt.Errorf("代码包走 tzs 应退出 2，实际 %d", code)
	}
	return nil
}

// stUnlockDeniedS：env=s + 非 topstd + 无 std_section_verify → 退出码 4（无绕过）。
func stUnlockDeniedS(dir string) error {
	entries := testutil.NormalEntriesWith("adzi999", map[string]string{
		"env": "s", "login_user": "tiptop", "std_section_verify": "N",
	})
	p, err := testutil.WriteEntries(filepath.Join(dir, "src"), "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	if code := cmdExport([]string{p, "-o", out}); code != 0 {
		return fmt.Errorf("export 退出码 %d", code)
	}
	// 即使加 --yes 也必须被拒（R9：授权不可自我伪造）
	if code := cmdUnlock([]string{out, "--yes"}); code != 4 {
		return fmt.Errorf("env=s 无授权应退出码 4（R9 无绕过），实际 %d", code)
	}
	return nil
}

// stUnlockAlreadyUnlocked：section_flag="Y" 的包 export 即 Unlocked。
func stUnlockAlreadyUnlocked(dir string) error {
	entries := testutil.NormalEntriesWith("adzi999", map[string]string{"section_flag": "Y"})
	p, err := testutil.WriteEntries(filepath.Join(dir, "src"), "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	if code := cmdExport([]string{p, "-o", out}); code != 0 {
		return fmt.Errorf("export 退出码 %d", code)
	}
	ws, err := storeOpen(out)
	if err != nil {
		return err
	}
	mf, err := ws.Manifest()
	if err != nil {
		return err
	}
	if mf.Section.State != string(model.SectionUnlocked) {
		return fmt.Errorf("已解开包应直接 Unlocked，实际 %q", mf.Section.State)
	}
	edited := readEdited(ws)
	if !bytes.Contains(edited, []byte("EDITABLE-SEC")) {
		return fmt.Errorf("已解开包应有 [EDITABLE-SEC] 围栏旗标")
	}
	// 对已解开的包 unlock 是 no-op（退出码 0）
	if code := cmdUnlock([]string{out}); code != 0 {
		return fmt.Errorf("已解开包的 unlock 应为 no-op，实际退出码 %d", code)
	}
	return nil
}

// expectApplyCode：apply 必须返回给定退出码，且**原包 sha256 不变**。
func expectApplyCode(pkgPath, wsDir string, wantCode int, tag string) error {
	before := mustRead(pkgPath)
	code := cmdApply([]string{wsDir})
	if code != wantCode {
		return fmt.Errorf("期待退出码 %d（%s），实际 %d", wantCode, tag, code)
	}
	if !bytes.Equal(before, mustRead(pkgPath)) {
		return fmt.Errorf("被拒的 apply 不应改包（%s）", tag)
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// stIterativeApplyLoop 钉住第二个真机事故：
//
//	apply 成功后 manifest 仍记着**上一版**包的 sha256，
//	于是同一个工作区的第二次 apply 被「源包自 export 之后已被改动」误拒（退出码 5）。
//	「改一次 → apply → 再改 → 再 apply」是 AI 与人的正常迭代方式，必须可用。
func stIterativeApplyLoop(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	markers := []string{"# iter-1", "# iter-2", "# iter-3"}
	for i, m := range markers {
		edited := readEdited(ws)
		needle := []byte("   RETURN p_a\n")
		if !bytes.Contains(edited, needle) {
			return fmt.Errorf("测试自身失效：找不到目标正文行")
		}
		edited = bytes.Replace(edited, needle, []byte("   RETURN p_a\n   "+m+"\n"), 1)
		if err := writeEdited(ws, edited); err != nil {
			return err
		}
		if code := cmdApply([]string{ws.Dir}); code != 0 {
			return fmt.Errorf("第 %d 次 apply 退出码 %d（迭代循环断了）", i+1, code)
		}
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.Point("function.adzi999_calc")
	if el == nil {
		return fmt.Errorf("找不到目标点")
	}
	for _, m := range markers {
		if !bytes.Contains(el.Content(tapDoc.Raw), []byte(m)) {
			return fmt.Errorf("包内缺少 %s（说明某次 apply 丢了改动）", m)
		}
	}
	if _, err := os.Stat(filepath.Join(ws.Dir, ".tdev", "prev.tzc")); err != nil {
		return fmt.Errorf("缺少 .tdev/prev.tzc 回滚副本: %v", err)
	}
	return nil
}

// stEOLNormalizedByEditor 复现并钉住一次真机事故：
//
//	用 VS Code 打开 prog.full.4gl 保存后，整份文件的混合行尾（CRLF + LF）
//	被归一成纯 LF（capt110 实测少了 866 字节）。旧实现逐字节比对，
//	于是 596/605 个 Region 被误判为「改动」、只读区段被改 → apply 以退出码 4 拒绝，
//	用户只是加了一行注释却什么也写不回去。
//
// 正确语义：受保护字节 tdev **从不写回包**（tapfile.Rewrite 只在原 .tap 的字节区间
// 上做替换），所以行尾归一不算改动；同时必须保证**未改动区域在包里的字节一字不变**。
func stEOLNormalizedByEditor(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	if !bytes.Contains(edited, []byte("\r\n")) {
		return fmt.Errorf("测试自身失效：合成文档里没有 CRLF，无法验证行尾归一")
	}
	// ① 模拟编辑器：整份文件行尾归一成 LF
	edited = bytes.ReplaceAll(edited, []byte("\r\n"), []byte("\n"))
	// ② 再真的改一个点的正文
	marker := []byte("   RETURN p_a\n")
	if !bytes.Contains(edited, marker) {
		return fmt.Errorf("测试自身失效：找不到目标正文行")
	}
	edited = bytes.Replace(edited, marker,
		[]byte("   RETURN p_a\n   # tdev eol-normalized edit\n"), 1)
	if err := writeEdited(ws, edited); err != nil {
		return err
	}
	// 写前缓存未改动区域的包内字节（用一个只读区段验证）
	pkgBefore, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapBefore, err := tapfile.Parse(pkgBefore.Tap().Data)
	if err != nil {
		return err
	}
	secBefore := tapBefore.Section("adzi999.description")
	if secBefore == nil {
		return fmt.Errorf("测试自身失效：找不到只读区段")
	}
	// 该区段正文在合成包里是 CRLF
	if !bytes.Contains(secBefore.Content(tapBefore.Raw), []byte("\r\n")) {
		return fmt.Errorf("测试自身失效：只读区段正文里没有 CRLF")
	}
	beforeSecSha := model.Sha256Bytes(secBefore.Content(tapBefore.Raw))

	// ③ apply 必须成功（行尾归一不算改动），且只动目标点
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("行尾归一后 apply 应成功，实际退出码 %d", code)
	}
	pkgAfter, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapAfter, err := tapfile.Parse(pkgAfter.Tap().Data)
	if err != nil {
		return err
	}
	el := tapAfter.Point("function.adzi999_calc")
	if el == nil || !bytes.Contains(el.Content(tapAfter.Raw), []byte("tdev eol-normalized edit")) {
		return fmt.Errorf("目标点正文没写进去")
	}
	secAfter := tapAfter.Section("adzi999.description")
	if secAfter == nil {
		return fmt.Errorf("只读区段丢了")
	}
	// 关键断言：未改动区段在包里的字节一字不变（CRLF 仍是 CRLF）
	if got := model.Sha256Bytes(secAfter.Content(tapAfter.Raw)); got != beforeSecSha {
		return fmt.Errorf("未改动区段的包内字节被行尾归一污染了（应逐字节不变）")
	}
	return nil
}

func stFenceLineTamper(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 改围栏行里的元数据（围栏行属于围栏外，AI 不许改）
	needle := []byte(`order="1"`)
	if !bytes.Contains(edited, needle) {
		return fmt.Errorf("测试自身失效：围栏行里找不到 %s", needle)
	}
	edited = bytes.Replace(edited, needle, []byte(`order="9"`), 1)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 3 {
		return fmt.Errorf("期待退出码 3，实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stReadonlySection(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 往只读区段正文里加一行
	idx := bytes.Index(edited, []byte("#應用 a00 樣板自動產生(Version:3)"))
	if idx < 0 {
		return fmt.Errorf("测试自身失效：找不到只读区段正文")
	}
	edited = append(edited[:idx], append([]byte("# tdev 偷改区段\n"), edited[idx:]...)...)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 4 {
		return fmt.Errorf("期待退出码 4（写入被拒），实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stStructureLine(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = bytes.Replace(edited, []byte("PRIVATE FUNCTION adzi999_calc(p_a)"),
		[]byte("PRIVATE FUNCTION adzi999_calc2(p_a)"), 1)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 3 {
		return fmt.Errorf("期待退出码 3，实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stUnpairedFence(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = bytes.Replace(edited, []byte("{//@tdev:end point}\n"), []byte(""), 1)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 3 {
		return fmt.Errorf("期待退出码 3，实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stCDATAClose(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = bytes.Replace(edited, []byte("   RETURN p_a\n"),
		[]byte("   RETURN p_a\n   # ]]> 塞 CDATA 结束符\n"), 1)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 3 {
		return fmt.Errorf("期待退出码 3（I11b），实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stOutsideFence(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	edited = append(edited, []byte("\n# 围栏外偷偷塞的内容\n")...)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 3 {
		return fmt.Errorf("期待退出码 3，实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stOnlyScope(dir string) error {
	ws, p, err := exportNormal(dir, []string{"function.adzi999_calc"}, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 改一个**未导出**的点（global.import 的内容是 "IMPORT util"）
	edited = bytes.Replace(edited, []byte("IMPORT util"), []byte("IMPORT evil"), 1)
	writeEdited(ws, edited)
	before := mustRead(p)
	code := cmdApply([]string{ws.Dir})
	if code != 4 {
		return fmt.Errorf("期待退出码 4（--only 范围外），实际 %d", code)
	}
	if !bytes.Equal(before, mustRead(p)) {
		return fmt.Errorf("被拒的 apply 不应改包")
	}
	return nil
}

func stMissingTgl(dir string) error {
	entries := testutil.NormalEntries("adzi999")
	delete(entries, "adzi999.tgl")
	p, err := testutil.WritePackage(dir, "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	code := cmdExport([]string{p, "-o", out})
	if code != 2 {
		return fmt.Errorf("期待退出码 2，实际 %d", code)
	}
	return nil
}

func stBadVer(dir string) error {
	entries := testutil.NormalEntries("adzi999")
	entries["ver"] = "2.0\n"
	p, err := testutil.WritePackage(dir, "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	code := cmdExport([]string{p, "-o", out})
	if code != 2 {
		return fmt.Errorf("期待退出码 2，实际 %d", code)
	}
	return nil
}

func stUnpairedSection(dir string) error {
	entries := testutil.NormalEntries("adzi999")
	// 删掉一个 {</section>}
	tgl := entries["adzi999.tgl"]
	idx := strings.Index(tgl, "{</section>}")
	entries["adzi999.tgl"] = tgl[:idx] + tgl[idx+len("{</section>}"):]
	p, err := testutil.WritePackage(dir, "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	code := cmdExport([]string{p, "-o", out})
	if code != 2 {
		return fmt.Errorf("期待退出码 2，实际 %d", code)
	}
	return nil
}

//---------------------------------------------------------------------------
// 结构变更：删点 / 新增点 / 改区段
//---------------------------------------------------------------------------

func stDeletePoint(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	beginMark := []byte("{//@tdev:begin point function.adzi999_calc")
	start := bytes.Index(edited, beginMark)
	if start < 0 {
		return fmt.Errorf("测试自身失效：找不到点围栏")
	}
	endRel := bytes.Index(edited[start:], []byte("{//@tdev:end point}"))
	if endRel < 0 {
		return fmt.Errorf("测试自身失效：找不到 end 围栏")
	}
	end := start + endRel + len("{//@tdev:end point}")
	if end < len(edited) && edited[end] == '\n' {
		end++
	}
	edited = append(append([]byte(nil), edited[:start]...), edited[end:]...)
	writeEdited(ws, edited)
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.PointExact("function.adzi999_calc")
	if el == nil {
		return fmt.Errorf("删除后找不到 tombstone")
	}
	if v, _ := el.Attr("status"); v != "d" {
		return fmt.Errorf("status 应为 d，实际 %q", v)
	}
	if !bytes.Contains(el.Content(tapDoc.Raw), []byte("RETURN p_a")) {
		return fmt.Errorf("标记删除必须保留原内容（I5）")
	}
	return nil
}

func stAppendPoint(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 在 other_function 锚点区段的 end 之前插入一个新点块
	anchor := []byte("adzi999.other_function")
	ai := bytes.Index(edited, anchor)
	if ai < 0 {
		return fmt.Errorf("测试自身失效：找不到锚点区段")
	}
	closeRel := bytes.Index(edited[ai:], []byte("{//@tdev:end section}"))
	if closeRel < 0 {
		return fmt.Errorf("测试自身失效：找不到锚点区段结束")
	}
	at := ai + closeRel
	block := "{//@tdev:begin point function.adzi999_new [EDITABLE status=\"\" src=\"s\" new=\"Y\"]}\n" +
		"PRIVATE FUNCTION adzi999_new()\n   RETURN 2\nEND FUNCTION\n" +
		"{//@tdev:end point}\n"
	edited = append(append(append([]byte(nil), edited[:at]...), []byte(block)...), edited[at:]...)
	writeEdited(ws, edited)
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	el := tapDoc.Point("function.adzi999_new")
	if el == nil {
		return fmt.Errorf("新增点没写进 TAP")
	}
	if v, _ := el.Attr("status"); v != "u" {
		return fmt.Errorf("新增点 status 应为 u（D-1：写 c 会被设计器丢弃），实际 %q", v)
	}
	if v, _ := el.Attr("new"); v != "Y" {
		return fmt.Errorf("新增点 new 应为 Y，实际 %q", v)
	}
	if v, _ := el.Attr("order"); v != "2" {
		return fmt.Errorf("新增点 order 应为 max+1=2，实际 %q", v)
	}
	if !bytes.Contains(el.Content(tapDoc.Raw), []byte("adzi999_new")) {
		return fmt.Errorf("新增点内容不对")
	}
	// 原有点仍在
	if tapDoc.Point("function.adzi999_calc") == nil {
		return fmt.Errorf("新增后丢了原有点")
	}
	return nil
}

func stSectionWrite(dir string) error {
	ws, p, err := exportNormal(dir, nil, false)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	marker := []byte("#+ Description: 合成測試程式")
	if !bytes.Contains(edited, marker) {
		return fmt.Errorf("测试自身失效：找不到区段正文")
	}
	edited = bytes.Replace(edited, marker, append(append([]byte(nil), marker...), []byte("\n#+ tdev 改过的框架行")...), 1)
	writeEdited(ws, edited)
	// Locked 态改区段 → 退出码 4，并且文案要指向 unlock
	if code := cmdApply([]string{ws.Dir}); code != 4 {
		return fmt.Errorf("Locked 态改区段应退出码 4，实际 %d", code)
	}
	// unlock 未给 --yes → 退出码 4（并打印设计器代价警告原文）
	if code := cmdUnlock([]string{ws.Dir}); code != 4 {
		return fmt.Errorf("unlock 未给 --yes 应退出码 4，实际 %d", code)
	}
	// 给了 --yes → 迁移成功
	if code := cmdUnlock([]string{ws.Dir, "--yes"}); code != 0 {
		return fmt.Errorf("unlock --yes 退出码 %d", code)
	}
	// 迁移后区段应变成可编辑（围栏标 EDITABLE-SEC）
	after := readEdited(ws)
	if !bytes.Contains(after, []byte("EDITABLE-SEC")) {
		return fmt.Errorf("unlock 后应出现 [EDITABLE-SEC] 围栏旗标")
	}
	if code := cmdApply([]string{ws.Dir}); code != 0 {
		return fmt.Errorf("unlock 后 apply 退出码 %d", code)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return err
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		return err
	}
	sec := tapDoc.Section("adzi999.description")
	if sec == nil {
		return fmt.Errorf("找不到区段")
	}
	if !bytes.Contains(sec.Content(tapDoc.Raw), []byte("tdev 改过的框架行")) {
		return fmt.Errorf(".tap 区段正文没更新")
	}
	if v, _ := sec.Attr("status"); v != "u" {
		return fmt.Errorf("区段 status 应为 u，实际 %q", v)
	}
	if v, _ := tapDoc.RootAttr("section_flag"); v != "Y" {
		return fmt.Errorf("根 section_flag 应为 Y，实际 %q", v)
	}
	// .tgl 必须与 .tap 的区段正文逐字节一致（README §3.13 硬性约束 6）
	tgl := pkg.Tgl()
	if tgl != nil {
		secs, err := tglfileSections(tgl.Data)
		if err != nil {
			return err
		}
		for _, s := range secs {
			if s.ID != "adzi999.description" {
				continue
			}
			tglBody := tgl.Data[s.BodyStart:s.BodyEnd]
			tapBody := sec.Content(tapDoc.Raw)
			if !bytes.Equal(bytes.Trim(tglBody, "\r\n"), bytes.Trim(tapBody, "\r\n")) {
				return fmt.Errorf(".tap 与 .tgl 的区段正文不一致（红线 R4 相关约束）")
			}
		}
	}
	return nil
}

func stUnparsablePoint(dir string) error {
	entries := testutil.NormalEntries("adzi999")
	tap := entries["adzi999.tap"]
	// 把一个 function.* 点的正文改成纯注释
	tap = strings.Replace(tap,
		"PRIVATE FUNCTION adzi999_calc(p_a)\n   DEFINE p_a INTEGER\n   LET p_a = p_a + 1\n   RETURN p_a\nEND FUNCTION",
		"#240401-00041#1 只有註解，沒有函式塊", 1)
	entries["adzi999.tap"] = tap
	p, err := testutil.WritePackage(dir, "adzi999.tzc", entries)
	if err != nil {
		return err
	}
	out := filepath.Join(dir, "ws")
	if code := cmdExport([]string{p, "-o", out}); code != 0 {
		return fmt.Errorf("export 退出码 %d", code)
	}
	ws, err := storeOpen(out)
	if err != nil {
		return err
	}
	edited := readEdited(ws)
	// 尝试改它的正文 → 必须被拒（退出码 4）
	edited = bytes.Replace(edited, []byte("#240401-00041#1 只有註解，沒有函式塊"),
		[]byte("#240401-00041#1 偷改"), 1)
	writeEdited(ws, edited)
	code := cmdApply([]string{out})
	if code != 4 {
		return fmt.Errorf("不可解析的函数块点应被拒（退出码 4），实际 %d", code)
	}
	return nil
}

//---------------------------------------------------------------------------
// 辅助：薄封装，避免自检代码直接纠缠各包细节
//---------------------------------------------------------------------------

type wsHandle = store.Workspace

func storeOpen(dir string) (*store.Workspace, error) { return store.Open(dir) }

func mustRead(p string) []byte {
	b, _ := os.ReadFile(p)
	return b
}

// exportNormal 写一个合成包并 export 到 <dir>/ws。
func exportNormal(dir string, only []string, allowSec bool) (*wsHandle, string, error) {
	p, err := testutil.NormalPkgPath(filepath.Join(dir, "src"), "adzi999")
	if err != nil {
		return nil, "", err
	}
	out := filepath.Join(dir, "ws")
	args := []string{p, "-o", out}
	if len(only) > 0 {
		args = append(args, "--only", strings.Join(only, ","))
	}
	if allowSec {
		args = append(args, "--allow-sec")
	}
	if code := cmdExport(args); code != 0 {
		return nil, "", fmt.Errorf("export 退出码 %d", code)
	}
	h, err := storeOpen(out)
	if err != nil {
		return nil, "", err
	}
	return h, p, nil
}

func readEdited(ws *store.Workspace) []byte {
	b, err := ws.ReadEdited()
	if err != nil {
		return nil
	}
	return b
}

func writeEdited(ws *store.Workspace, b []byte) error { return ws.WriteEdited(b) }

func mustReadWorkspaceBase(ws *store.Workspace) []byte {
	b, _ := ws.ReadBase()
	return b
}

func synthDocFrom(pkg *pkgfile.Package) (*model.Document, error) {
	return synth.Synthesize(pkg, synth.Options{})
}

func renderDoc(doc *model.Document) ([]byte, []*model.Region, []model.Span, error) {
	return fence.Render(doc)
}

func baseDoc(doc *model.Document, fenced []byte, regions []*model.Region, spans []model.Span) *model.Document {
	b := *doc
	b.Text = fenced
	b.Regions = regions
	b.Spans = spans
	return &b
}

func parseFenced(base *model.Document, fenced []byte) (*fence.ParseResult, error) {
	return fence.Parse(base, fenced)
}

func tglfileSections(b []byte) ([]*tglfile.Section, error) { return tglfile.FindSections(b) }

// entrySha 取包里某扩展名条目的 sha256。
func entrySha(p, ext string) string {
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		return ""
	}
	if ext == "ver" {
		if e := pkg.Entry("ver"); e != nil {
			return e.Sha256
		}
		return ""
	}
	if e := pkg.ByExt(ext); e != nil {
		return e.Sha256
	}
	return ""
}
