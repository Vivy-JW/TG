package fence

import (
	"bytes"
	"strings"
	"testing"

	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/synth"
	"tt/internal/dev/testutil"
	"tt/internal/testkit"
)

// asBase 把 Render 的产物包装成 Parse 需要的基线 Document。
func asBase(doc *model.Document, fenced []byte, regions []*model.Region, spans []model.Span) *model.Document {
	b := *doc
	b.Text = fenced
	b.Regions = regions
	b.Spans = spans
	return &b
}

//---------------------------------------------------------------------------
// 可逆性公理：parse_fenced(render(synthesize(pkg))) ≡ synthesize(pkg)
// 对全部 166 个真实包成立（Region 序列逐项相等 + 内容逐字节相等）。
//---------------------------------------------------------------------------

func TestFenceRoundTripCorpus(t *testing.T) {
	pkgs := testkit.CorpusFiles(t, ".tzc")
	totalRegions := 0
	for _, p := range pkgs {
		pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
		if err != nil {
			t.Fatalf("%s Open 失败: %v", p, err)
		}
		doc, err := synth.Synthesize(pkg, synth.Options{})
		if err != nil {
			t.Fatalf("%s 合成失败: %v", p, err)
		}
		fenced, regions, spans, err := Render(doc)
		if err != nil {
			t.Fatalf("%s render 失败: %v", p, err)
		}
		base := asBase(doc, fenced, regions, spans)
		res, err := Parse(base, fenced)
		if err != nil {
			t.Fatalf("%s parse_fenced 失败: %v", p, err)
		}
		if len(res.Appended) != 0 || len(res.Deleted) != 0 {
			t.Fatalf("%s 恒等往返不应有增删：appended=%d deleted=%d", p, len(res.Appended), len(res.Deleted))
		}
		a := base.AllRegions()
		b := res.Doc.AllRegions()
		if len(a) != len(b) {
			t.Fatalf("%s Region 数量变了：%d → %d", p, len(a), len(b))
		}
		for i := range a {
			if a[i].Kind != b[i].Kind || a[i].Name != b[i].Name {
				t.Fatalf("%s 第 %d 个 Region 不相等：%s/%s → %s/%s",
					p, i, a[i].Kind, a[i].Name, b[i].Kind, b[i].Name)
			}
			ca := base.Text[a[i].ContentSpan.Start:a[i].ContentSpan.End]
			cb := res.Doc.Text[b[i].ContentSpan.Start:b[i].ContentSpan.End]
			if !bytes.Equal(ca, cb) {
				t.Fatalf("%s Region %s 内容不等（基线 %d 字节，解析 %d 字节）",
					p, a[i].Name, len(ca), len(cb))
			}
			if a[i].Editable != b[i].Editable {
				t.Fatalf("%s Region %s 可编辑性变了：%v → %v", p, a[i].Name, a[i].Editable, b[i].Editable)
			}
			// 区间记账也必须逐项相等（render/parse 的 FullSpan/围栏行口径必须对称，
			// 否则围栏外字节的归属会漂移）
			if a[i].FullSpan != b[i].FullSpan {
				t.Fatalf("%s Region %s FullSpan 不对称：%+v → %+v",
					p, a[i].Name, a[i].FullSpan, b[i].FullSpan)
			}
			if a[i].ContentSpan != b[i].ContentSpan {
				t.Fatalf("%s Region %s ContentSpan 不对称：%+v → %+v",
					p, a[i].Name, a[i].ContentSpan, b[i].ContentSpan)
			}
			if a[i].BeginFence != b[i].BeginFence || a[i].EndFence != b[i].EndFence {
				t.Fatalf("%s Region %s 围栏行区间不对称", p, a[i].Name)
			}
		}
		totalRegions += len(a)
	}
	t.Logf("围栏可逆性：%d 个真实包、%d 个 Region，逐项相等且内容逐字节一致", len(pkgs), totalRegions)
}

//---------------------------------------------------------------------------
// 合成包单测：编辑正文 / 删点 / 追加点
//---------------------------------------------------------------------------

func synthDoc(t *testing.T, prog string) (*model.Document, []byte, []*model.Region, []model.Span) {
	t.Helper()
	p, err := testutil.NormalPkgPath(t.TempDir(), prog)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := synth.Synthesize(pkg, synth.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fenced, regions, spans, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	return doc, fenced, regions, spans
}

func TestRenderContainsFencesAndMetadata(t *testing.T) {
	_, fenced, _, _ := synthDoc(t, "adzi999")
	s := string(fenced)
	if !strings.Contains(s, FencePrefix+"begin point function.adzi999_calc [EDITABLE") {
		t.Errorf("缺少自订定义点的 EDITABLE 围栏行：\n%s", firstLines(s, 40))
	}
	if !strings.Contains(s, `status=""`) && !strings.Contains(s, `src="s"`) {
		t.Errorf("围栏行应带元数据")
	}
	if !strings.Contains(s, `deny="section-locked"`) {
		t.Errorf("框架未解开（Locked）时区段应为 READONLY 且 deny=section-locked")
	}
	if !strings.Contains(s, `APPEND`) {
		t.Errorf("锚点区段应标 APPEND")
	}
	var calcEditable bool
	doc := mustSynth(t, "adzi999")
	for _, r := range doc.AllRegions() {
		if r.Name == "function.adzi999_calc" {
			calcEditable = r.Editable
			if r.DenyCode != "" {
				t.Errorf("自订点不应被拒：%s（%s）", r.DenyCode, r.Reason)
			}
			if r.BodySpan == nil {
				t.Errorf("自订定义点必须有 BodySpan")
			}
		}
	}
	if !calcEditable {
		t.Errorf("env=c 下自订点应可编辑")
	}
}

func TestParseRejectsEditedStructureLine(t *testing.T) {
	_, fenced, _, spans := synthDoc(t, "adzi999")
	// 改函数头（结构行）
	edited := bytes.Replace(fenced, []byte("PRIVATE FUNCTION adzi999_calc(p_a)"),
		[]byte("PRIVATE FUNCTION adzi999_calc_RENAMED(p_a)"), 1)
	if bytes.Equal(edited, fenced) {
		t.Fatal("测试自身失效：没有替换到函数头")
	}
	doc := mustSynth(t, "adzi999")
	_, regions, _, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	base := asBase(doc, fenced, regions, spans)
	res, err := Parse(base, edited)
	if err != nil {
		t.Fatalf("结构行被改时 Parse 仍应成功（由 gate1/gate2 判定），实际报错: %v", err)
	}
	// 结构行区间必须落在可写区间之外 → verify 会拦。这里断言区间性质。
	for _, r := range res.Doc.Regions {
		if r.Name != "function.adzi999_calc" {
			continue
		}
		if len(r.StructureSpans) != 2 {
			t.Fatalf("应有两个结构行区间，实际 %d", len(r.StructureSpans))
		}
		editable := res.Doc.EditableSpans()
		for _, sp := range r.StructureSpans {
			for _, e := range editable {
				if e.Overlaps(sp) {
					t.Fatalf("结构行区间与可写区间重叠：%+v vs %+v", sp, e)
				}
			}
		}
	}
}

func TestParseDetectsDeletion(t *testing.T) {
	doc := mustSynth(t, "adzi999")
	fenced, regions, spans, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	base := asBase(doc, fenced, regions, spans)
	// 删掉 function.adzi999_calc 的整个围栏块（begin 到**它自己的** end）
	beginMark := []byte(FencePrefix + "begin point function.adzi999_calc")
	start := bytes.Index(fenced, beginMark)
	if start < 0 {
		t.Fatal("测试自身失效：找不到围栏块")
	}
	// 从该 begin 行向后找下一个 end 围栏行
	endRel := bytes.Index(fenced[start:], []byte(FencePrefix+"end point}"))
	if endRel < 0 {
		t.Fatal("测试自身失效：找不到该块的 end 围栏行")
	}
	end := start + endRel + len(FencePrefix+"end point}")
	if end < len(fenced) && fenced[end] == '\n' {
		end++
	}
	edited := append(append([]byte(nil), fenced[:start]...), fenced[end:]...)
	res, err := Parse(base, edited)
	if err != nil {
		t.Fatalf("删除点块应被接受（授权由 split 判定），实际报错: %v", err)
	}
	if len(res.Deleted) != 1 || res.Deleted[0].Name != "function.adzi999_calc" {
		t.Fatalf("未识别到删除：%+v", names(res.Deleted))
	}
}

func TestParseDetectsAppendInsideAnchor(t *testing.T) {
	doc := mustSynth(t, "adzi999")
	fenced, regions, spans, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	base := asBase(doc, fenced, regions, spans)
	// 在 other_function 锚点区段末尾追加一个新点块
	anchorEnd := bytes.Index(fenced, []byte(FencePrefix+"end section}"))
	if anchorEnd < 0 {
		t.Fatal("测试自身失效：找不到区段结束")
	}
	// 找 other_function 那一段的结束标记
	idx := bytes.Index(fenced, []byte("adzi999.other_function"))
	if idx < 0 {
		t.Fatal("测试自身失效：找不到锚点区段")
	}
	closeIdx := bytes.Index(fenced[idx:], []byte(FencePrefix+"end section}"))
	if closeIdx < 0 {
		t.Fatal("测试自身失效：找不到锚点区段结束标记")
	}
	at := idx + closeIdx
	newBlock := []byte(FencePrefix + "begin point function.adzi999_new [EDITABLE status=\"u\" src=\"c\" new=\"Y\"]}\n" +
		"PRIVATE FUNCTION adzi999_new()\n   RETURN 1\nEND FUNCTION\n" +
		FencePrefix + "end point}\n")
	edited := append(append(append([]byte(nil), fenced[:at]...), newBlock...), fenced[at:]...)

	res, err := Parse(base, edited)
	if err != nil {
		t.Fatalf("在 APPEND 锚点内追加点应被接受，实际报错: %v", err)
	}
	if len(res.Appended) != 1 || res.Appended[0].Name != "function.adzi999_new" {
		t.Fatalf("未识别到追加：%+v", names(res.Appended))
	}
}

func TestParseRejectsAppendOutsideAnchor(t *testing.T) {
	doc := mustSynth(t, "adzi999")
	fenced, regions, spans, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	base := asBase(doc, fenced, regions, spans)
	// 在正文最前面（围栏外）偷偷塞一个围栏块
	block := []byte(FencePrefix + "begin point function.adzi999_evil [EDITABLE]}\nX\n" + FencePrefix + "end point}\n")
	edited := append(block, fenced...)
	if _, err := Parse(base, edited); err == nil {
		t.Fatalf("锚点外追加必须被拒绝")
	}
}

func TestParseRejectsUnpairedFence(t *testing.T) {
	doc := mustSynth(t, "adzi999")
	fenced, regions, spans, err := Render(doc)
	if err != nil {
		t.Fatal(err)
	}
	base := asBase(doc, fenced, regions, spans)
	// 删掉一个 end 围栏行
	edited := bytes.Replace(fenced, []byte(FencePrefix+"end point}\n"), []byte(""), 1)
	if _, err := Parse(base, edited); err == nil {
		t.Fatalf("围栏不配对必须被拒绝")
	}
}

//---------------------------------------------------------------------------

func mustSynth(t *testing.T, prog string) *model.Document {
	t.Helper()
	p, err := testutil.NormalPkgPath(t.TempDir(), prog)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := synth.Synthesize(pkg, synth.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func names(rs []*model.Region) []string {
	var out []string
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
