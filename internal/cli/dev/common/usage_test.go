// usage_test.go —— `tt dev tzs formcheck` 那一段的多方机械防线（台账 D14）。
//
// 为什么这条测试住在 common 的**外部测试包**、以及为什么第三条腿读**源文本**而不是 import：
//
//	同一段用法文案有三个落点：README（基准）、总帮助（common.Usage）、命令面（tzsUsage，
//	在 ../tzs/tzs.go）。任何一处改了措辞而另一处没跟，
//	都没有编译错误、也没有任何测试会响 —— 症状是"帮助里写的退出码和 README 写的不一样"，
//	而脚本正是照着这几个数字分支的。比对基准取 **README**（人读的那份先被改，且它旁边就是
//	判据与实测基线）；其余三处各自与它比一次，于是谁漂都红，而没有人需要手抄第五份。
//
//	第三条腿本来可以 import tzs 直接读它的 Usage 常量，但那会把 tzs → form42f 整棵树拖进本包的
//	测试构建：本包只该为"一段文案"负责，不该因为邻包正在改 form42f 而红。所以读它的**源文本** ——
//	常量就是那段 raw string 的逐字内容，抠行逻辑同一条。
package common_test

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"tt/internal/cli/dev/common"
)

// formcheckBlockAnchor 是这段用法块的锚：**子命令**那一行（不是退出码散文里提到的那次）。
const formcheckBlockAnchor = "tt dev tzs formcheck"

// TestUsageTextHasFormcheck 是最基本的一条：总帮助里**必须有** formcheck。
//
// 这条测试红过：切片加 `formcheck` 子命令时只改了 tzs 线自己的文本（tzsUsage）与 README，
// 总帮助（裸 `tt dev` 打出来的那份）漏了 —— 于是 `tt dev` 里看不到这条命令存在。
func TestUsageTextHasFormcheck(t *testing.T) {
	block := formcheckBlockLines(common.Usage)
	if len(block) == 0 {
		t.Fatalf("总帮助（common.Usage）里没有以 %q 开头的用法行。\n"+
			"它是 `tt dev` 打出来的那张命令面清单：少一条就等于这条命令在帮助里不存在。\n"+
			"现有总帮助：\n%s", formcheckBlockAnchor, common.Usage)
	}

	// 用法契约的三块必须都在：名字与两个输入、结构化输出、退出码 0/2/6 的语义。
	for _, want := range []string{
		formcheckBlockAnchor,
		"--form42f <表单.42f>",
		"--tzs <包.tzs|表单.4fd>",
		"[--json]",
		"0 一致",
		"2 参数或读取错",
		"6 有确定差异",
	} {
		if !strings.Contains(strings.Join(block, "\n"), want) {
			t.Errorf("总帮助的 formcheck 段缺 %q。\n现有这一段：\n%s", want, strings.Join(block, "\n"))
		}
	}
}

// TestUsageAndReadmeAgreeOnFormcheckUsage 是主判据：总帮助里的 formcheck 段与 README 里的
// 对应段**逐行一致**（只允许共同缩进不同）—— 手法定型于切片 B 的
// TestRootHelpAndReadmeAgreeOnFormExitCodes。
//
// 不抄第三份字面量的意思就是：这条测试不写"应该是什么文案"，它只量"两处是不是同一份"。
func TestUsageAndReadmeAgreeOnFormcheckUsage(t *testing.T) {
	readme := readTzsReadme(t)
	want := formcheckBlockLines(readme)
	if len(want) == 0 {
		t.Fatalf("%s 里没有以 %q 开头的用法行 —— 基准被删了，或者行的写法变了。\n"+
			"README 是这段文案的**基准**（另一处是总帮助）；先把它写回来。", tzsReadmePath(), formcheckBlockAnchor)
	}
	if drift := formcheckBlockDrift(common.Usage, readme); drift != "" {
		t.Errorf("%s", drift)
	}
}

// TestReadmeAndCommandFaceAgreeOnFormcheckUsage 是第三条腿：命令面（`tt dev tzs --help` 打出来的
// 那份，即 ../tzs/tzs.go 的 tzsUsage 常量）与 README 也要逐行一致。
//
// 三处里 README 是基准，另外两处各自被比对一次 ⇒ 任何一处漂了都会红，而没有人需要
// 手抄第二份文案去"看着对齐"。真正的判定口径（哪些规则、退什么码）在实现里；
// 这里只管**文案**不许分叉。
func TestReadmeAndCommandFaceAgreeOnFormcheckUsage(t *testing.T) {
	readme := readTzsReadme(t)
	if len(formcheckBlockLines(readme)) == 0 {
		t.Fatalf("%s 里没有 formcheck 用法段（基准缺了）", tzsReadmePath())
	}
	if drift := formcheckBlockDrift(readTzsSource(t), readme); drift != "" {
		t.Errorf("命令面（tzsUsage，../tzs/tzs.go）与 README 的 formcheck 段不一致：\n%s", drift)
	}
}

// TestFormcheckBlockDriftCatchesExitCodeChange 是**反证**：把帮助里的 `6` 改成别的数字，
// 上面那条双边比对必须红。
//
// 没有这条，双边比对可能因为抠行逻辑太宽（比如扫了整段输出）而永远为真 —— 那种测试
// 看起来在守，其实什么都没守。这里直接喂一份改过数字的帮助文本给同一个比对函数，
// 要求它**必须**报不一致；顺带验参数行也是同样的道理。
func TestFormcheckBlockDriftCatchesExitCodeChange(t *testing.T) {
	readme := readTzsReadme(t)
	if len(formcheckBlockLines(common.Usage)) == 0 || len(formcheckBlockLines(readme)) == 0 {
		t.Fatal("基准或帮助缺 formcheck 段，反证无从谈起（先让前面的测试变绿）")
	}
	if drift := formcheckBlockDrift(common.Usage, readme); drift != "" {
		t.Fatalf("前置条件不成立：两处本来就不一致，反证无法定位\n%s", drift)
	}

	cases := []struct {
		name string
		from string
		to   string
	}{
		{"退出码 6 被改", "6 有确定差异", "7 有确定差异"},
		{"退出码 2 被改", "2 参数或读取错", "3 参数或读取错"},
		{"参数名被改", "--form42f <表单.42f>", "--form <表单.42f>"},
		{"整段删掉", formcheckBlockAnchor, "tt dev tzs export"},
	}
	for _, c := range cases {
		tampered := strings.Replace(common.Usage, c.from, c.to, 1)
		if tampered == common.Usage {
			t.Fatalf("%s：总帮助里找不到 %q —— 反证的输入已经失效，改这条测试", c.name, c.from)
		}
		if drift := formcheckBlockDrift(tampered, readme); drift == "" {
			t.Errorf("%s：比对函数没报不一致 —— 这条防线是空的（改数字也能绿，等于没守）", c.name)
		}
	}
}

// tzsReadmePath 是基准文本的位置（本包是 internal/cli/dev/common）。
func tzsReadmePath() string { return filepath.Join("..", "tzs", "README.md") }

// tzsSourcePath 是命令面文案的落点：tzsUsage 常量所在的那份源文件。
func tzsSourcePath() string { return filepath.Join("..", "tzs", "tzs.go") }

func readTzsReadme(t *testing.T) string {
	t.Helper()
	return readText(t, tzsReadmePath(), "它是 formcheck 用法文案的基准")
}

func readTzsSource(t *testing.T) string {
	t.Helper()
	return readText(t, tzsSourcePath(), "tzsUsage（命令面文案）住在它里面")
}

func readText(t *testing.T, path, why string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读不到 %s：%v（%s）", path, err, why)
	}
	return string(b)
}

// formcheckBlockDrift 比较两段文本里的 formcheck 用法块；一致返回空串，否则返回人读的差异说明。
func formcheckBlockDrift(help, ref string) string {
	got := formcheckBlockLines(help)
	want := formcheckBlockLines(ref)
	if len(got) == 0 || len(want) == 0 {
		return "有一侧没有 formcheck 用法段（help=" + strconv.Itoa(len(got)) + " 行，基准=" + strconv.Itoa(len(want)) + " 行）"
	}
	if strings.Join(got, "\n") == strings.Join(want, "\n") {
		return ""
	}
	return "两处 formcheck 用法文案不一致（必须逐行相同，只允许共同缩进不同）\n" +
		"--- 总帮助 / 命令面 ---\n" + strings.Join(got, "\n") +
		"\n--- 基准（internal/cli/dev/tzs/README.md） ---\n" + strings.Join(want, "\n")
}

// formcheckBlockLines 从一段文本里抠出 `tt dev tzs formcheck` 那一段用法块：
// **子命令**那一行（`tt dev tzs formcheck` 后面必须是行尾或空白，所以散文里提到
// "formcheck 多一个 6" 不会被当成锚），加上紧跟其后、**以 ≥4 格空白缩进开头**、
// 非空、且不是另起一条命令（`tt …`）的续行；每行去掉首尾空白。
//
// 缩进必须去掉：总帮助整体缩进 2 格、命令面 2 格、README 的代码块不缩进 ——
// 要的是"文案本身逐字一致"，不是"排版也一致"。找不到返回 nil。
func formcheckBlockLines(text string) []string {
	lines := strings.Split(text, "\n")
	for i, ln := range lines {
		trimmed := strings.TrimSpace(ln)
		if !strings.HasPrefix(trimmed, formcheckBlockAnchor) {
			continue
		}
		if rest := trimmed[len(formcheckBlockAnchor):]; rest != "" && rest[0] != ' ' && rest[0] != '\t' {
			continue // `tt dev tzs formcheckx` 之类，不是这条命令
		}
		out := []string{trimmed}
		for j := i + 1; j < len(lines); j++ {
			body := lines[j]
			cont := strings.TrimSpace(body)
			if cont == "" || !hasIndentAtLeast(body, 4) || strings.HasPrefix(cont, "tt ") {
				break
			}
			out = append(out, cont)
		}
		return out
	}
	return nil
}

// hasIndentAtLeast 判断行首是否有至少 n 格空白（空格或制表符，制表符算 4 格）。
func hasIndentAtLeast(s string, n int) bool {
	width := 0
	for _, r := range s {
		switch r {
		case ' ':
			width++
		case '\t':
			width += 4
		default:
			return width >= n
		}
		if width >= n {
			return true
		}
	}
	return false
}
