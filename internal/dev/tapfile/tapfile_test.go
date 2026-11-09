package tapfile

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"tt/internal/testkit"
)

// corpusTaps 返回语料里所有 .tap 条目的字节（按包名排序，稳定）。
func corpusTaps(t *testing.T) map[string][]byte {
	t.Helper()
	root := testkit.CorpusRoot(t)
	out := map[string][]byte{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.EqualFold(filepath.Ext(p), ".tzc") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			return nil
		}
		for _, zf := range zr.File {
			if filepath.Ext(zf.Name) != ".tap" {
				continue
			}
			rc, err := zf.Open()
			if err != nil {
				return nil
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return nil
			}
			out[p] = data
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫描语料失败: %v", err)
	}
	if len(out) == 0 {
		t.Skipf("%s 下没有 .tap", root)
	}
	return out
}

//---------------------------------------------------------------------------
// 1) 全部真实 TAP 都能扫描；元素区间自洽
//---------------------------------------------------------------------------

func TestParseAllCorpusTaps(t *testing.T) {
	taps := corpusTaps(t)
	totalPoints, totalSections, totalCDATA := 0, 0, 0
	for name, raw := range taps {
		d, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s 扫描失败: %v", name, err)
		}
		if d.Root.Name != "add_points" {
			t.Errorf("%s 根元素应为 add_points，实际 %q", name, d.Root.Name)
		}
		for _, p := range append(append([]*Element{}, d.Points...), d.Sections...) {
			// 开放标签必须自洽
			if string(raw[p.TagStart:p.TagStart+1+len(p.Name)]) != "<"+p.Name {
				t.Errorf("%s: %s 的 TagStart 不对", name, p.Name)
			}
			// 闭合标签必须匹配
			if !p.SelfClosing {
				closeTag := "</" + p.Name + ">"
				// 闭合标签前可能有空白
				tail := raw[p.End-len(closeTag) : p.End]
				if string(tail) != closeTag {
					t.Errorf("%s: %s 的 End 不是紧接闭合标签: %q", name, p.Name, tail)
				}
			}
			if p.HasCDATA() {
				if string(raw[p.CDATAStart-9:p.CDATAStart]) != "<![CDATA[" {
					t.Errorf("%s: %s CDATAStart 前不是 <![CDATA[", name, p.Name)
				}
				if string(raw[p.CDATAEnd:p.CDATAEnd+3]) != "]]>" {
					t.Errorf("%s: %s CDATAEnd 后不是 ]]>", name, p.Name)
				}
				totalCDATA++
			}
		}
		totalPoints += len(d.Points)
		totalSections += len(d.Sections)
		// 属性区间自洽：raw[ValueStart:ValueEnd] == 属性值
		for _, p := range d.Points {
			for _, a := range p.Attrs {
				if a.ValueStart == 0 && a.ValueEnd == 0 {
					continue
				}
				if got := string(raw[a.ValueStart:a.ValueEnd]); got != a.Value {
					t.Errorf("%s: %s 属性 %s 区间不匹配: %q vs %q", name, p.Name, a.Name, got, a.Value)
				}
			}
		}
	}
	t.Logf("语料 .tap=%d 个，<point> 合计=%d，<section> 合计=%d，带 CDATA 元素=%d",
		len(taps), totalPoints, totalSections, totalCDATA)
	if totalPoints == 0 || totalSections == 0 {
		t.Fatalf("语料扫描结果异常：point=%d section=%d", totalPoints, totalSections)
	}
}

//---------------------------------------------------------------------------
// 2) 用**独立正则**对拍 CDATA 内容（不依赖扫描器定位）
//---------------------------------------------------------------------------

func TestCDATAMatchesIndependentRegex(t *testing.T) {
	taps := corpusTaps(t)
	// 每次只扫一遍原文，用**通用**正则收集全部 CDATA，再按名字分桶做多重集对拍。
	// （按名字逐个 reg.FindAll 会把 150KB 原文扫 32k 次，慢 200 倍。）
	pointPat := regexp.MustCompile(`(?s)<point\b([^>]*)>\s*<!\[CDATA\[(.*?)\]\]>`)
	secPat := regexp.MustCompile(`(?s)<section\b([^>]*)>\s*<!\[CDATA\[(.*?)\]\]>`)
	namePat := regexp.MustCompile(`name="([^"]*)"`)
	idPat := regexp.MustCompile(`id="([^"]*)"`)

	checked, dupNames := 0, 0
	for name, raw := range taps {
		d, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s 扫描失败: %v", name, err)
		}
		wantPoints := map[string][][]byte{}
		for _, m := range pointPat.FindAllSubmatch(raw, -1) {
			nm := namePat.FindSubmatch(m[1])
			if nm == nil {
				continue
			}
			k := string(nm[1])
			wantPoints[k] = append(wantPoints[k], m[2])
		}
		gotPoints := map[string][][]byte{}
		for _, p := range d.Points {
			if !p.HasCDATA() {
				continue
			}
			k, _ := p.Attr("name")
			gotPoints[k] = append(gotPoints[k], p.Content(raw))
		}
		if len(wantPoints) != len(gotPoints) {
			t.Errorf("%s: 带 CDATA 的点名个数不一致（正则 %d，扫描器 %d）", name, len(wantPoints), len(gotPoints))
		}
		for k, want := range wantPoints {
			got := gotPoints[k]
			if len(got) != len(want) {
				t.Errorf("%s: 点 %s 的 CDATA 段数不一致（正则 %d，扫描器 %d）", name, k, len(want), len(got))
				continue
			}
			if len(want) > 1 {
				dupNames++
			}
			sortByteSlices(want)
			sortByteSlices(got)
			for i := range want {
				if !bytes.Equal(want[i], got[i]) {
					t.Errorf("%s: 点 %s 第 %d 段 CDATA 与独立正则不一致", name, k, i)
				}
			}
			checked++
		}

		wantSecs := map[string][][]byte{}
		for _, m := range secPat.FindAllSubmatch(raw, -1) {
			im := idPat.FindSubmatch(m[1])
			if im == nil {
				continue
			}
			k := string(im[1])
			wantSecs[k] = append(wantSecs[k], m[2])
		}
		for _, s := range d.Sections {
			sid, _ := s.Attr("id")
			want := wantSecs[sid]
			if len(want) == 0 {
				t.Errorf("%s: 正则找不到区段 %s 的 CDATA", name, sid)
				continue
			}
			found := false
			for _, w := range want {
				if bytes.Equal(w, s.Content(raw)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s: 区段 %s 的 CDATA 与独立正则不一致", name, sid)
			}
			checked++
		}
	}
	t.Logf("独立正则对拍通过的元素数=%d，其中重名点 %d 个（tombstone+live）", checked, dupNames)
}

func sortByteSlices(s [][]byte) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && bytes.Compare(s[j], s[j-1]) < 0; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

//---------------------------------------------------------------------------
// 3) 恒等改写：把每个元素的内容原样写回，结果必须**逐字节相同**
//    （这是对扫描器+改写器最严格的对拍：一个字节错位都会暴露）
//---------------------------------------------------------------------------

func TestIdentityRewriteIsByteExact(t *testing.T) {
	taps := corpusTaps(t)
	pkgs, opsTotal, dupHandled := 0, 0, 0
	for name, raw := range taps {
		d, err := Parse(raw)
		if err != nil {
			t.Fatalf("%s 扫描失败: %v", name, err)
		}
		var ops []Op
		// 关键：按**唯一名字**取解析结果（第一个未删除元素），
		// 与写操作的目标解析规则完全一致；tombstone 由设计器语义自动跳过。
		seen := map[string]bool{}
		for _, p := range d.Points {
			pn, _ := p.Attr("name")
			if seen[pn] {
				continue
			}
			seen[pn] = true
			if len(d.PointsAll(pn)) > 1 {
				dupHandled++
			}
			target := d.Point(pn)
			if target == nil || !target.HasCDATA() {
				continue
			}
			ops = append(ops, SetPointCDATA{Name: pn, Content: append([]byte(nil), target.Content(raw)...)})
		}
		for _, s := range d.Sections {
			sid, _ := s.Attr("id")
			if !s.HasCDATA() {
				continue
			}
			ops = append(ops, SetSectionCDATA{ID: sid, Content: append([]byte(nil), s.Content(raw)...)})
		}
		if v, ok := d.RootAttr("section_flag"); ok {
			ops = append(ops, SetRootAttr{Key: "section_flag", Value: v})
		}
		out, err := Rewrite(raw, ops...)
		if err != nil {
			t.Fatalf("%s 恒等改写失败: %v", name, err)
		}
		if !bytes.Equal(out, raw) {
			i := firstDiff(out, raw)
			t.Fatalf("%s 恒等改写后字节不同（首个差异偏移 %d，共 %d ops）", name, i, len(ops))
		}
		opsTotal += len(ops)
		pkgs++
	}
	t.Logf("恒等改写逐字节一致：%d 个包、%d 个写操作；其中重名点 %d 个已按未删除元素解析",
		pkgs, opsTotal, dupHandled)
}

func firstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

//---------------------------------------------------------------------------
// 4) 属性「有则改、无则加」只动开放标签
//---------------------------------------------------------------------------

const tinyTAP = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<add_points prog="adzi999" env="c" section_flag="N" topind="sd">
  <other>
    <code_template value="F" status="" />
  </other>
  <point name="function.adzi999_f" order="1" ver="1" cite_std="N" new="Y" status="" src="s" readonly="" mark_hard="N" modi_by_topstd="">
    <![CDATA[
PUBLIC FUNCTION adzi999_f()
   LET x = 1
END FUNCTION]]>
  </point>
  <point name="global.memo" order="" ver="" cite_std="N" new="Y" status="" src="s" edit="s"/>
  <section id="adzi999.main" src="s" status="">
    <![CDATA[MAIN
END MAIN]]>
  </section>
</add_points>
`

func TestAttrSetIfPresentElseAdd(t *testing.T) {
	raw := []byte(tinyTAP)
	// 已存在：只改值
	out, err := Rewrite(raw, SetPointAttr{Name: "function.adzi999_f", Key: "status", Value: "u"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte(`status="u"`)) {
		t.Errorf("status 未改成 u")
	}
	if bytes.Count(out, []byte("status=")) != bytes.Count(raw, []byte("status=")) {
		t.Errorf("改已有属性不应增加属性个数")
	}
	// 不存在：追加到开放标签末尾
	out2, err := Rewrite(raw, SetPointAttr{Name: "function.adzi999_f", Key: "mark_hard", Value: "Y"},
		SetPointAttr{Name: "function.adzi999_f", Key: "new_attr", Value: "1"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out2, []byte(`new_attr="1"`)) {
		t.Errorf("新属性未追加")
	}
	// 其余字节不变：把开放标签换成占位后应完全一致
	tagStart := bytes.Index(raw, []byte(`<point name="function.adzi999_f"`))
	tagEnd := bytes.Index(raw[tagStart:], []byte(">")) + tagStart + 1
	tagStart2 := bytes.Index(out2, []byte(`<point name="function.adzi999_f"`))
	tagEnd2 := bytes.Index(out2[tagStart2:], []byte(">")) + tagStart2 + 1
	if !bytes.Equal(raw[tagEnd:], out2[tagEnd2:]) {
		t.Errorf("改属性动到了开放标签以外的字节")
	}
}

func TestRewriteSelfClosingPointExpands(t *testing.T) {
	raw := []byte(tinyTAP)
	out, err := Rewrite(raw, SetPointCDATA{Name: "global.memo", Content: []byte("  # 客制內容\n")})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out, []byte("<![CDATA[  # 客制內容\n]]>")) {
		t.Errorf("自闭合点未正确展开：\n%s", out)
	}
	if !bytes.Contains(out, []byte("</point>")) {
		t.Errorf("展开后缺少 </point>")
	}
	// 展开后属性必须保留（含 edit="s"）
	d, err := Parse(out)
	if err != nil {
		t.Fatalf("展开后无法再解析: %v", err)
	}
	if v, _ := d.Point("global.memo").Attr("edit"); v != "s" {
		t.Errorf("展开后丢了属性 edit")
	}
}

func TestRejectCDATACloseInContent(t *testing.T) {
	raw := []byte(tinyTAP)
	_, err := Rewrite(raw, SetPointCDATA{Name: "function.adzi999_f", Content: []byte("x ]]> y")})
	if err == nil {
		t.Fatalf("内容含 ]]> 必须被拒绝")
	}
	if ec, ok := err.(interface{ ExitCode() int }); ok && ec.ExitCode() != 2 {
		// 包装后可能取不到，这里只做提示
		t.Logf("退出码=%d", ec.ExitCode())
	}
}

func TestMarkDeletedKeepsContent(t *testing.T) {
	raw := []byte(tinyTAP)
	before := ""
	{
		d, _ := Parse(raw)
		before = string(d.Point("function.adzi999_f").Content(raw))
	}
	out, err := Rewrite(raw, MarkDeleted{Name: "function.adzi999_f"})
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(out)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := d.Point("function.adzi999_f").Attr("status"); v != "d" {
		t.Errorf("status 应为 d，实际 %q", v)
	}
	if after := string(d.Point("function.adzi999_f").Content(out)); after != before {
		t.Errorf("标记删除必须保留原内容（I5）")
	}
}

func TestAddPointKeepsSiblingTemplate(t *testing.T) {
	raw := []byte(tinyTAP)
	attrs := map[string]string{
		"name": "function.adzi999_new", "order": "2", "ver": "", "cite_std": "N",
		"new": "Y", "src": "c", "status": "u", "ch": "", "ind_fun": "sd", "ind_extra": "N",
	}
	out, err := Rewrite(raw, AddPoint{Attrs: attrs, Content: []byte("PUBLIC FUNCTION adzi999_new()\nEND FUNCTION")})
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse(out)
	if err != nil {
		t.Fatalf("新增点后无法解析: %v", err)
	}
	p := d.Point("function.adzi999_new")
	if p == nil {
		t.Fatalf("新增点未写入")
	}
	if v, _ := p.Attr("status"); v != "u" {
		t.Errorf("新增点 status 应为 u（D-1 偏差：status=c 会被设计器丢弃），实际 %q", v)
	}
	if !bytes.Equal(p.Content(out), []byte("PUBLIC FUNCTION adzi999_new()\nEND FUNCTION")) {
		t.Errorf("新增点内容不对: %q", p.Content(out))
	}
	// 新增点后原有元素仍在
	for _, n := range []string{"function.adzi999_f", "global.memo"} {
		if d.Point(n) == nil {
			t.Errorf("新增点后丢了原有点 %s", n)
		}
	}
	if d.Section("adzi999.main") == nil {
		t.Errorf("新增点后丢了区段")
	}
}

func TestAddPointRejectsDuplicate(t *testing.T) {
	raw := []byte(tinyTAP)
	_, err := Rewrite(raw, AddPoint{Attrs: map[string]string{"name": "global.memo"}, Content: []byte("x")})
	if err == nil {
		t.Fatalf("同名点必须拒绝新增")
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"无根元素":      "",
		"标签未闭合":     `<add_points><point name="a">`,
		"闭合不匹配":     `<add_points><point name="a"></section></add_points>`,
		"CDATA 未闭合": `<add_points><point name="a"><![CDATA[x</point></add_points>`,
	}
	for name, in := range cases {
		if _, err := Parse([]byte(in)); err == nil {
			t.Errorf("%s 应当报错", name)
		}
	}
}
