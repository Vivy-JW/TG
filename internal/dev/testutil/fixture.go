// Package testutil 提供**合成**测试包（不依赖真实语料），供 selftest 与各包单测使用。
//
// 合成包的结构照着真实包的最小可用集：8 个区段（含 3 个强制只读锚点区段）、
// 一个自订定义点（function.）、几个裸名插入点、以及 TAP 里的 <other>。
//
// corpus.go 里的东西是另一件事：它**只发现**真实语料（`.tzc` / `.tzs` 两条管线共用一套
// 「根怎么定、怎么走」），不合成任何包。合成与发现分开，是为了让「不依赖真实语料」这条
// 承诺在 fixture.go 里继续成立 —— 需要真语料的用例显式去 corpus.go 拿文件。
package testutil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// WritePackage 把给定条目写成 .tzc，返回路径。
func WritePackage(dir, name string, entries map[string]string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	p := filepath.Join(dir, name)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// 条目顺序照真实包：.4gl / .tap / .tgl / ver
	order := []string{".4gl", ".tap", ".tgl"}
	prog := strings.TrimSuffix(name, filepath.Ext(name))
	for _, ext := range order {
		key := prog + ext
		v, ok := entries[key]
		if !ok {
			continue
		}
		w, err := zw.Create(key)
		if err != nil {
			return "", err
		}
		if _, err := w.Write([]byte(v)); err != nil {
			return "", err
		}
	}
	for k, v := range entries {
		if k == prog+".4gl" || k == prog+".tap" || k == prog+".tgl" {
			continue
		}
		w, err := zw.Create(k)
		if err != nil {
			return "", err
		}
		if _, err := w.Write([]byte(v)); err != nil {
			return "", err
		}
	}
	if err := zw.Close(); err != nil {
		return "", err
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return p, nil
}

// NormalEntries 返回一个「正常包」的全部条目。
//
// 结构要点（对齐真实包）：
//   - TGL 是框架骨架，区段标记独占一行，锚点区段带 readonly="Y"
//   - 区段正文里放 {<point name="…"/>} 占位符
//   - TAP 根元素是 <add_points>，子元素顺序 <other> → <point> ×N → <section> ×N
//   - .tap 的 CDATA 存**整块**（自订定义点）或正文（裸名点）
func NormalEntries(prog string) map[string]string {
	tgl := NormalTGL(prog)
	tap := NormalTAP(prog)
	four := Normal4GL(prog)
	return map[string]string{
		prog + ".tap": tap,
		prog + ".tgl": tgl,
		prog + ".4gl": four,
		"ver":         "1.0\n",
	}
}

// NormalTGL 是框架骨架。
func NormalTGL(prog string) string {
	var b strings.Builder
	b.WriteString("#該程式未解開Section, 採用最新樣板產出!\n")
	b.WriteString("#該程式非freestyle程式!\n")
	b.WriteString(`{<section id="` + prog + `.description" type="s" >}` + "\n")
	// 真实包里 TAP 与 TGL 的同名区段正文逐字节相同，且**混用行尾**
	// （capt110 实测：866 个 CRLF + 8466 个 LF）。这里照抄这个特征，
	// 让「编辑器归一而行尾」这类真机事故能在自检里复现。
	b.WriteString("#應用 a00 樣板自動產生(Version:3)\r\n")
	b.WriteString("#+ Description: 合成測試程式\r\n")
	b.WriteString("{</section>}\n")
	b.WriteString(`{<section id="` + prog + `.global" type="s" >}` + "\n")
	b.WriteString("#add-point:填寫註解說明 name=\"global.memo\"\n")
	b.WriteString(`{<point name="global.memo" edit="s"/>}` + "\n")
	b.WriteString("#end add-point\n")
	b.WriteString("#add-point:增加匯入項目 name=\"global.import\"\n")
	b.WriteString(`{<point name="global.import"/>}` + "\n")
	b.WriteString("#end add-point\n")
	b.WriteString("IMPORT os\n")
	b.WriteString("{</section>}\n")
	b.WriteString(`{<section id="` + prog + `.main" type="s" >}` + "\n")
	b.WriteString("MAIN\n")
	b.WriteString("  #add-point:初始設定(客製用) name=\"main.define_customerization\"\n")
	b.WriteString(`  {<point name="main.define_customerization" edit="c"/>}` + "\n")
	b.WriteString("  #end add-point\n")
	b.WriteString("  DIALOG\n")
	b.WriteString("    INPUT BY NAME g_x\n")
	b.WriteString("  END DIALOG\n")
	b.WriteString("END MAIN\n")
	b.WriteString("{</section>}\n")
	for _, s := range []string{"other_function", "other_dialog", "other_report"} {
		b.WriteString(`{<section id="` + prog + "." + s + `" readonly="Y" type="s" >}` + "\n")
		kind := strings.Replace(s, "other_", "", 1)
		b.WriteString(`{<point name="other.` + kind + `"/>}` + "\n")
		b.WriteString("{</section>}\n")
	}
	return b.String()
}

// NormalTAP 是客制增量 XML（混合换行：元素间 CRLF、CDATA 内 LF —— 照真实包）。
func NormalTAP(prog string) string {
	// TAP 根 type="M" → 设计器强制 PRIVATE（FunctionInfoWindow.xaml.cs:176-179）
	// 自订定义点的 CDATA 存**整块**：描述块 + 签名行 + 正文 + END
	fn := "#+ Description: 合成測試用函式\n#+ Usage......: CALL " + prog + `_calc(1)
PRIVATE FUNCTION ` + prog + `_calc(p_a)
   DEFINE p_a INTEGER
   LET p_a = p_a + 1
   RETURN p_a
END FUNCTION`
	memo := "   # 客制註解\n"
	imports := "IMPORT util\n"
	defineCus := "  CALL ccl_init()\n"
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="no"?>` + "\r\n")
	b.WriteString(`<add_points prog="` + prog + `" std_prog="` + prog + `" erpver="3.0" module="TST" ver="1"` +
		` env="c" zone="topprd" booking="N" type="M" identity="c" section_flag="N" designer_ver="1.0"` +
		` login_user="tiptop" std_section_verify="N" std_to_cus="" pre_compile="Y" topind="sd">` + "\r\n")
	b.WriteString("  <other>\r\n")
	b.WriteString(`    <code_template value="F" status=""/>` + "\r\n")
	b.WriteString(`    <free_style value="N" status=""/>` + "\r\n")
	b.WriteString(`    <start_arg value="" status=""/>` + "\r\n")
	b.WriteString("  </other>\r\n")
	// 自订定义点（CDATA 存整块）
	b.WriteString(`  <point name="function.` + prog + `_calc" order="1" ver="1" cite_std="N" new="Y"` +
		` ind_fun="sd" ind_extra="N" ch="" status="" src="s" readonly="" mark_hard="N"` +
		` modi_by_topstd="" mapping="function.` + prog + `_calc">` + "\r\n")
	b.WriteString("<![CDATA[" + fn + "]]>\r\n")
	b.WriteString("  </point>\r\n")
	// 裸名插入点（CDATA 存正文）
	b.WriteString(`  <point name="global.memo" order="" ver="" cite_std="N" new="Y" ind_fun="sd"` +
		` ind_extra="N" ch="" status="" src="s" readonly="" mark_hard="N" modi_by_topstd="">` + "\r\n")
	b.WriteString("<![CDATA[" + memo + "]]>\r\n")
	b.WriteString("  </point>\r\n")
	b.WriteString(`  <point name="global.import" order="" ver="" cite_std="N" new="Y" ind_fun="sd"` +
		` ind_extra="N" ch="" status="" src="s" readonly="" mark_hard="N" modi_by_topstd="">` + "\r\n")
	b.WriteString("<![CDATA[" + imports + "]]>\r\n")
	b.WriteString("  </point>\r\n")
	b.WriteString(`  <point name="main.define_customerization" order="" ver="" cite_std="N" new="Y"` +
		` ind_fun="sd" ind_extra="N" ch="" status="" src="s" readonly="" mark_hard="N" modi_by_topstd="">` + "\r\n")
	b.WriteString("<![CDATA[" + defineCus + "]]>\r\n")
	b.WriteString("  </point>\r\n")
	// 空的自订定义点（TGL 有锚点、TAP 无对应点 → 设计器造空点）
	// 区段：与 TGL 一一对应
	sections := []struct{ id, body string }{
		{prog + ".description", "\r\n#應用 a00 樣板自動產生(Version:3)\r\n#+ Description: 合成測試程式\r\n"},
		{prog + ".global", "\r\n#add-point:填寫註解說明 name=\"global.memo\"\r\n{<point name=\"global.memo\" edit=\"s\"/>}\r\n#end add-point\r\n" +
			"#add-point:增加匯入項目 name=\"global.import\"\r\n{<point name=\"global.import\"/>}\r\n#end add-point\r\nIMPORT os\r\n"},
		{prog + ".main", "\r\nMAIN\r\n  #add-point:初始設定(客製用) name=\"main.define_customerization\"\r\n" +
			"  {<point name=\"main.define_customerization\" edit=\"c\"/>}\r\n  #end add-point\r\n  DIALOG\r\n" +
			"    INPUT BY NAME g_x\r\n  END DIALOG\r\nEND MAIN\r\n"},
		{prog + ".other_function", "\r\n{<point name=\"other.function\"/>}\r\n"},
		{prog + ".other_dialog", "\r\n{<point name=\"other.dialog\"/>}\r\n"},
		{prog + ".other_report", "\r\n{<point name=\"other.report\"/>}\r\n"},
	}
	for _, s := range sections {
		b.WriteString(`  <section id="` + s.id + `" src="s" status="" ver="1" ch="" readonly="" modi_by_topstd="">` + "\r\n")
		b.WriteString("<![CDATA[" + s.body + "]]>\r\n")
		b.WriteString("  </section>\r\n")
	}
	b.WriteString("</add_points>\r\n")
	return b.String()
}

// Normal4GL 模拟服务器产出（客户端永不读回、永不改写 —— 红线 R1）。
func Normal4GL(prog string) string {
	return "#+ Description: 合成測試程式（非真實 T100 資料）\n" +
		"MAIN\n  CALL ccl_init()\nEND MAIN\n"
}

// NormalEntriesWith 在正常包基础上改写 TAP 根属性（用于解锁闸门等用例）。
//
// 例：NormalEntriesWith("adzi999", map[string]string{"env": "s", "login_user": "tiptop"})
func NormalEntriesWith(prog string, patch map[string]string) map[string]string {
	e := NormalEntries(prog)
	tap := e[prog+".tap"]
	for k, v := range patch {
		re := regexp.MustCompile(`(^|\s)` + regexp.QuoteMeta(k) + `="[^"]*"`)
		if re.MatchString(tap) {
			tap = re.ReplaceAllString(tap, "${1}"+k+`="`+v+`"`)
		}
	}
	e[prog+".tap"] = tap
	return e
}

// WriteEntries 把给定条目写成包（NormalEntriesWith 的配套）。
func WriteEntries(dir, name string, entries map[string]string) (string, error) {
	return WritePackage(dir, name, entries)
}

// NormalPkgPath 是便捷入口：在 dir 下写出一个正常包。
func NormalPkgPath(dir, prog string) (string, error) {
	if _, err := os.Stat(dir); err != nil {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("建目录失败: %w", err)
		}
	}
	return WritePackage(dir, prog+".tzc", NormalEntries(prog))
}
