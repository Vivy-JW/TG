package tzs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"tt/internal/testkit"
)

// engine_probe_test.go —— 盯住 engine/test/ 下那些探测程序**不许再依赖作者本机的东西**。
//
// ## 为什么需要它
//
// `RoundTrip.exe` 是 `TestCorpus*` / `TestMiniCorpus*` / `TestFnsGate` 全都要起的程序。
// 而它（连同另外 7 个探测程序）曾经各自硬编码了一行：
//
//	const string SRC = @"D:\我的项目\T100设计器";
//	… File.OpenRead(Path.Combine(SRC, "SpecDesignerCommon", "langs", "zh-cn.xaml"))
//
// `SRC` 是 `const`，任何 `TZSCLI_*` 变量都改不了它，所以在**没有那棵反编译源码树**的机器上，
// RoundTrip 一跑就 `DirectoryNotFoundException`。2026-09-27 一位评估者就是这么撞上的：
// 他干净克隆、把引擎建好，然后 `TestMiniCorpus*` 三条全红。
//
// 最阴的一处是**照文档做才会红**：`engineExeAndDir` 在引擎不存在时 Skip，
// 所以**不**建引擎反而是绿的（绿跳）。不修的话，"冒烟回归不再依赖某台机器"这句话就是假的 ——
// 只是把依赖从"有客户语料那台"换成了"有源码树那台"。
//
// 现在那 8 个调用点共用 `engine/test/DesignerLang.cs`（从 DLL 内嵌资源取，与
// src/Designer/Bootstrap.MergeLanguages 同一套做法）。这条测试守着它别退回去。
//
// ## 判据为什么是"代码模式"而不是那个中文字面量
//
// 反馈原话是"断言这些探测程序里不再出现 `D:\我的项目` 字面量"。这里收紧成**两种会被编译的写法**：
// `DesignerLang.cs` 的注释里就有那个路径（它是这件事的来历，该留着），
// 而注释不会让程序在别人机器上崩 —— 拿字面量当判据会误伤它，然后被下一个人"顺手"删掉这条测试。
func TestEngineProbesDoNotHardcodeAuthorPaths(t *testing.T) {
	dir := filepath.Join(testkit.RepoRoot(t), "engine", "test")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读不了 %s：%v（engine/test 是仓库本体，缺了就是坏了）", dir, err)
	}

	// 两种写法：把作者本机路径声明成常量，以及拿那个常量去拼路径。
	// 后者是真正的要害（常量声明在那里是无害的，用了才出事），但两个一起禁更省事，
	// 也堵住了"换个常量名再拼一次"。
	banned := []string{`const string SRC`, `Path.Combine(SRC,`}

	var hits []string
	files := 0
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".cs") {
			continue
		}
		files++
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读不了 %s：%v", name, err)
		}
		for i, ln := range strings.Split(string(b), "\n") {
			// 只认**会被编译的代码**：注释里出现这两行是好事（那件事的来历），
			// 而注释不会让程序在别人机器上崩。第一版没跳注释，于是它把
			// DesignerLang.cs 文件头里那段"以前长这样"的引文抓了出来 —— 守卫误伤自己的说明。
			if t := strings.TrimSpace(ln); strings.HasPrefix(t, "//") ||
				strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
				continue
			}
			for _, pat := range banned {
				if strings.Contains(ln, pat) {
					hits = append(hits, name+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(ln))
				}
			}
		}
	}
	if files == 0 {
		t.Fatalf("%s 下一个 .cs 都没有 —— 探测程序被搬走了？这条守卫会变成空转", dir)
	}
	if len(hits) > 0 {
		t.Errorf("探测程序又依赖作者本机路径了（%d 处）：\n  %s\n\n"+
			"那个路径是 `D:\\我的项目\\T100设计器`（作者本机的设计器反编译源码树），"+
			"任何 TZSCLI_* 都改不了它 —— 于是**任何没有那棵树的机器上，RoundTrip 一跑就红**，\n"+
			"而 RoundTrip 是 TestCorpus* / TestMiniCorpus* / TestFnsGate 全都要起的。\n"+
			"改法：删掉那个常量，把那段 xaml 载入换成一句\n"+
			"  DesignerLang.Merge(app, INSTALL);\n"+
			"它是这 8 个调用点共用的那一份（见 engine/test/DesignerLang.cs 的文件头）。",
			len(hits), strings.Join(hits, "\n  "))
	}
}

// TestEngineDocsCiteRealProbeHashes 盯住 engine 文档里对探测程序 sha256 的引用。
//
// ## 为什么加它
//
// `engine/SPEC.md` 里有一句：`test/ProbeReopen.cs`（sha256 `4f8c06b8…`）在真实包上跑完三个测试。
// 2026-09-27 一查，**那个 sha 和该文件的任何一个历史版本都对不上**（四个版本分别是
// c9c006c9 / ba1b5bbc / 97cda936 / 07ca3419）。也就是说它从写下的那天起就没对过，
// 而**没有任何东西在检查它** —— 一条没人验的 pin，写了等于没写，还会让读到的人以为
// "这段结论是对着那个字节版本得出的"。
//
// 这类引用本身是好东西（它把"这些数出自哪一份源码"钉住），坏的只是没人核对。
// 所以这里把"核对"变成断言，而不是把那句删掉。
//
// ## 判据
//
// 扫 engine/*.md，找 `test/<名字>.cs`（sha256 `<至少 8 位十六进制>`…）这种引用，
// 逐条拿**文件实际内容的 sha256** 去比。哈希按 **LF 归一后**算：`.cs` 没有被 .gitattributes
// 罩住，autocrlf=true 的机器上会 checkout 成 CRLF，那时按原样算会对不上 —— 那不是 pin 错了，
// 是行尾转换，归一之后与 git 里存的那份逐字节相同。
func TestEngineDocsCiteRealProbeHashes(t *testing.T) {
	root := testkit.RepoRoot(t)
	eng := filepath.Join(root, "engine")
	docs, err := filepath.Glob(filepath.Join(eng, "*.md"))
	if err != nil {
		t.Fatalf("列 engine/*.md 失败：%v", err)
	}
	if len(docs) == 0 {
		t.Fatalf("%s 下一份 .md 都没有 —— 文档被搬走了？这条守卫会变成空转", eng)
	}

	// `test/名字.cs`（sha256 `十六进制…`）—— 全角括号与反引号都是原文的写法。
	cite := regexp.MustCompile("`test/([A-Za-z_][A-Za-z0-9_]*\\.cs)`（sha256 `([0-9a-f]{8,})(?:…|\\.\\.\\.|·)*`）")

	checked := 0
	for _, doc := range docs {
		b, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("读不了 %s：%v", doc, err)
		}
		for _, m := range cite.FindAllStringSubmatch(string(b), -1) {
			name, want := m[1], m[2]
			p := filepath.Join(eng, "test", name)
			src, err := os.ReadFile(p)
			if err != nil {
				t.Errorf("%s 引用了 test/%s 的 sha256，但那个文件不存在：%v\n"+
					"删掉探针就要连这句引用一起处置 —— 别让它悬在那儿。",
					filepath.Base(doc), name, err)
				continue
			}
			got := sha256Hex(normalizeLF(src))
			if !strings.HasPrefix(got, want) {
				t.Errorf("%s 说 test/%s 的 sha256 是 %s…，实际是 %s…\n"+
					"（按 LF 归一后的内容算）改过那个探针，就要连这句引用一起改 ——\n"+
					"这条 pin 的意义正是「那些数出自哪一份源码」，漂了它就没有意义了。",
					filepath.Base(doc), name, want, got[:len(want)])
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatalf("engine/*.md 里一条 test/*.cs 的 sha256 引用都没找到（%d 份文档）——\n"+
			"要么引用被删了，要么写法变了让这条守卫空转。空转的守卫比没有更糟。", len(docs))
	}
}

// sha256Hex 算内容的十六进制摘要。
func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// normalizeLF 把 CRLF 折成 LF —— 让上面对哈希的比对与工作树的行尾无关。
func normalizeLF(b []byte) []byte {
	return bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
}
