package tzc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/dev/fence"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/tapfile"
	"tt/internal/dev/verify"
	"tt/internal/testkit"
)

// requireDeep 把「对全部真实包」的深度回归设为**显式开关**。
//
// 为什么需要这个开关：下面两个用例要对 166 个真实包各跑一遍
// export + verify / export + apply 仿真，实测合计 9–11 分钟；而
// `go test` 默认 -timeout=10m，go 命令会在 10m+1m 处直接杀掉测试进程并报
// 「*** Test killed: ran too long (11m0s)」——表现为**随机器负载时好时坏的假失败**
// （同一份代码有时 530 s 通过、有时 660 s 被杀），会污染「go test ./... 全绿」这条验收证据。
// 注意：这个杀进程来自 go 命令，测试二进制内部改 -test.timeout 是拦不住的。
//
// 所以默认 `go test ./...` 跳过这两个用例（会打印跳过原因），
// 需要全量证据时显式打开并给足超时：
//
//	$env:TDEV_DEEP="1"; go test ./... -timeout 30m
func requireDeep(t *testing.T) {
	t.Helper()
	if os.Getenv("TDEV_DEEP") == "" {
		t.Skip("深度语料回归未启用：设 TDEV_DEEP=1 并给足 -timeout 30m（见 README「测试与验收」）")
	}
}

// copyPkg 把真实语料里的包拷进临时目录（回归测试永不写真实语料）。
func copyPkg(t *testing.T, dir string, i int, src string) string {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("读 %s 失败: %v", src, err)
	}
	dst := filepath.Join(dir, fmt.Sprintf("pkg%03d.tzc", i))
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("写副本失败: %v", err)
	}
	return dst
}

// silent 已收敛为 testkit.Silent（tzc 与 tzs 两个测试包共用，理由同 CaptureStdout）。

// 语料发现不在这里：根怎么定、怎么走、哪些草稿文件不算语料，都在 internal/dev/testutil，
// 跳过文案在 internal/testkit。从前本文件自己留了两个薄壳，现在连壳也去掉了 ——
// 各包各留一层壳，就是"再抄一份"的入口。

// gateReports 复算 gate1+gate2（失败诊断用）。
func gateReports(t *testing.T, ws *wsHandle, base *model.Document, parsed *fence.ParseResult) *verify.Report {
	t.Helper()
	mf, err := ws.Manifest()
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tapDoc, err := tapfile.Parse(pkg.Tap().Data)
	if err != nil {
		t.Fatal(err)
	}
	rep := &verify.Report{}
	g1 := verify.Gate1(base, parsed)
	rep.Findings = append(rep.Findings, g1.Findings...)
	rep.Errors += g1.Errors
	rep.Warns += g1.Warns
	rep.Infos += g1.Infos
	g2 := verify.Gate2(base, parsed, tapDoc)
	rep.Findings = append(rep.Findings, g2.Findings...)
	rep.Errors += g2.Errors
	rep.Warns += g2.Warns
	rep.Infos += g2.Infos
	return rep
}

func reportString(rep *verify.Report) string {
	var b strings.Builder
	for _, f := range rep.Findings {
		if f.Severity != verify.SevError {
			continue
		}
		fmt.Fprintf(&b, "[%s] %s %s; ", f.Severity, f.Code, f.Message)
	}
	return b.String()
}

//---------------------------------------------------------------------------
// S3：export + verify 对全部真实包零 error
//---------------------------------------------------------------------------

func TestCorpusExportVerify(t *testing.T) {
	requireDeep(t)
	pkgs := testkit.CorpusFiles(t, ".tzc")
	tmp := t.TempDir()
	totalRegions, totalPoints, totalSections := 0, 0, 0
	totalWarns, totalInfos := 0, 0
	for i, p := range pkgs {
		// 一律在临时目录里对**副本**操作：真实语料是只读输入，绝不能被测试写到
		p = copyPkg(t, tmp, i, p)
		wsDir := filepath.Join(tmp, fmt.Sprintf("ws%03d", i))
		if code := testkit.Silent(t, func() int { return cmdExport([]string{p, "-o", wsDir}) }); code != 0 {
			t.Fatalf("%s export 退出码 %d", p, code)
		}
		ws, err := storeOpen(wsDir)
		if err != nil {
			t.Fatalf("%s 打开工作区失败: %v", p, err)
		}
		mf, err := ws.Manifest()
		if err != nil {
			t.Fatal(err)
		}
		totalRegions += mf.Stats.Points + mf.Stats.Sections
		totalPoints += mf.Stats.Points
		totalSections += mf.Stats.Sections

		code := testkit.Silent(t, func() int { return cmdVerify([]string{wsDir}) })
		base, _, err := loadBase(ws)
		if err != nil {
			t.Fatal(err)
		}
		edited, err := ws.ReadEdited()
		if err != nil {
			t.Fatal(err)
		}
		parsed, perr := fence.Parse(base, edited)
		if perr != nil {
			t.Fatalf("%s parse_fenced 失败: %v", p, perr)
		}
		rep := gateReports(t, ws, base, parsed)
		totalWarns += rep.Warns
		totalInfos += rep.Infos
		if code != 0 || rep.HasError() {
			t.Errorf("%s verify 退出码 %d，error 级发现：%s\n%s", p, code, reportString(rep), dumpRegions(parsed))
		}
	}
	t.Logf("S3 语料 export+verify：%d 个包、%d 个 Region（point %d / section %d）、verify 全部 0 error；warn=%d info=%d",
		len(pkgs), totalRegions, totalPoints, totalSections, totalWarns, totalInfos)
}

// dumpRegions 在失败时给出可编辑点清单，便于定位。
func dumpRegions(parsed *fence.ParseResult) string {
	var b strings.Builder
	n := 0
	for _, r := range parsed.Doc.AllRegions() {
		if r.Kind != model.RegionPoint || !r.Editable {
			continue
		}
		fmt.Fprintf(&b, "    可编辑点 %s origin=%s deny=%s\n", r.Name, r.Origin, r.DenyCode)
		n++
		if n > 20 {
			b.WriteString("    …\n")
			break
		}
	}
	return b.String()
}

//---------------------------------------------------------------------------
// S5：apply 仿真 —— 确定性挑一个可编辑自订函数点，机械插入一行注释，写回并核对
//---------------------------------------------------------------------------

func TestCorpusApplySimulation(t *testing.T) {
	requireDeep(t)
	pkgs := testkit.CorpusFiles(t, ".tzc")
	tmp := t.TempDir()
	applied, skipped := 0, 0
	for i, p := range pkgs {
		// 一律在临时目录里对**副本**操作：真实语料是只读输入，绝不能被测试写到
		p = copyPkg(t, tmp, i, p)
		wsDir := filepath.Join(tmp, fmt.Sprintf("ws%03d", i))
		if code := testkit.Silent(t, func() int { return cmdExport([]string{p, "-o", wsDir}) }); code != 0 {
			t.Fatalf("%s export 退出码 %d", p, code)
		}
		ws, err := storeOpen(wsDir)
		if err != nil {
			t.Fatal(err)
		}
		base, mf, err := loadBase(ws)
		if err != nil {
			t.Fatal(err)
		}
		edited, err := ws.ReadEdited()
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := fence.Parse(base, edited)
		if err != nil {
			t.Fatalf("%s parse_fenced 失败: %v", p, err)
		}
		var target *model.Region
		for _, r := range parsed.Doc.AllRegions() {
			if r.Kind != model.RegionPoint || !r.Editable || r.BodySpan == nil || r.BodySpan.Empty() {
				continue
			}
			if !strings.HasPrefix(r.Name, "function.") {
				continue
			}
			target = r
			break
		}
		if target == nil {
			skipped++
			continue
		}
		// 写前缓存：旧 TAP 字节 + 关键条目摘要
		pkgBefore, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		oldTap := append([]byte(nil), pkgBefore.Tap().Data...)
		before4gl, beforeVer := "", ""
		if e := pkgBefore.ByExt(".4gl"); e != nil {
			before4gl = e.Sha256
		}
		if e := pkgBefore.Entry("ver"); e != nil {
			beforeVer = e.Sha256
		}

		marker := "\n   # tdev apply-sim"
		at := target.BodySpan.End
		newEdited := append([]byte(nil), edited[:at]...)
		newEdited = append(newEdited, []byte(marker)...)
		newEdited = append(newEdited, edited[at:]...)
		if err := ws.WriteEdited(newEdited); err != nil {
			t.Fatal(err)
		}
		if code := testkit.Silent(t, func() int { return cmdApply([]string{wsDir}) }); code != 0 {
			t.Fatalf("%s apply 退出码 %d（目标点 %s，deny=%s）", p, code, target.Name, target.DenyCode)
		}
		pkg2, err := pkgfile.Open(mf.Pkg.Path, pkgfile.OpenOptions{})
		if err != nil {
			t.Fatalf("%s 写后打开失败: %v", p, err)
		}
		if e := pkg2.ByExt(".4gl"); e == nil || e.Sha256 != before4gl {
			t.Errorf("%s：.4gl 被改动了（红线 R1）", p)
		}
		if e := pkg2.Entry("ver"); e == nil || e.Sha256 != beforeVer {
			t.Errorf("%s：ver 被改动了（红线 R2）", p)
		}
		tapDoc, err := tapfile.Parse(pkg2.Tap().Data)
		if err != nil {
			t.Fatalf("%s 写后 TAP 解析失败: %v", p, err)
		}
		el := tapDoc.Point(target.Name)
		if el == nil {
			t.Fatalf("%s：写后找不到目标点 %s", p, target.Name)
		}
		if !bytes.Contains(el.Content(tapDoc.Raw), []byte("# tdev apply-sim")) {
			t.Errorf("%s：目标点 %s 正文没写进去", p, target.Name)
		}
		if v, _ := el.Attr("status"); v != "u" {
			t.Errorf("%s：目标点 status 应为 u，实际 %q", p, v)
		}
		// 只动目标点：其余点内容与写前一致。
		// 注意真实包里存在**同名两次**的情形（tombstone(status=d) + live），
		// 因此按「同名元素的第 i 个」成对比较，而不是按名字取第一个。
		before, err := tapfile.Parse(oldTap)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, bp := range before.Points {
			n, _ := bp.Attr("name")
			if n == target.Name || seen[n] {
				continue
			}
			seen[n] = true
			bAll := before.PointsAll(n)
			aAll := tapDoc.PointsAll(n)
			if len(bAll) != len(aAll) {
				t.Errorf("%s：点 %s 的元素个数变了 %d → %d", p, n, len(bAll), len(aAll))
				continue
			}
			for i := range bAll {
				if !bytes.Equal(bAll[i].Content(before.Raw), aAll[i].Content(tapDoc.Raw)) {
					t.Errorf("%s：非目标点 %s（第 %d 个元素）内容被改动", p, n, i)
				}
				if bv, ok := bAll[i].Attr("status"); ok {
					if av, _ := aAll[i].Attr("status"); av != bv {
						t.Errorf("%s：非目标点 %s（第 %d 个元素）status 被改动 %q → %q", p, n, i, bv, av)
					}
				}
			}
		}
		// 条目集合与顺序不变
		if len(pkg2.Entries) != len(pkgBefore.Entries) {
			t.Errorf("%s：条目数变了", p)
		}
		applied++
	}
	t.Logf("S5 apply 仿真：应用 %d 个包，跳过 %d（没有合适的自订函数点）；全部只动目标点、.4gl/ver 字节不变",
		applied, skipped)
	if applied == 0 {
		t.Fatalf("没有可仿真的包")
	}
}
