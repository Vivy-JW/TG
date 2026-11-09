package cli

// root_test.go —— 根命令帮助与 README 的漂移防线。
//
// **为什么这条测试住在这个包**：`internal/cli/dev` 里的 TestUsageTextHasNoStaleAdvice
// 覆盖不到 rootCmd.Long，而那个包不能 import 本包（`internal/cli/root.go` →
// `internal/cli/dev` → `internal/cli/dev`，反过来就成环）。rootCmd 是本包的包级变量，
// 同包测试才直接看得到它 —— 所以这条断言只能住在这里。
//
// 它要抓的是**同一个事实写在两处、只改了一处**：`tt --help` 与 README 的首屏都有一张
// "四块能力"的清单，`.tzs` 那行曾经两处都还写着已删除的 `call` 网关，而当时的漂移测试
// 只看 tzsUsage/Usage 两个常量 —— 改一处不影响另一处，也没有任何测试会响。

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestBuildScriptsAreASCIIAndNotMixed 钉住两个 .bat 的两条硬约束。
//
// 为什么值得一条测试：cmd.exe 用 OEM 代码页解析 .bat 的**字节**，于是
//
//  1. 文件里不能有非 ASCII 字节。UTF-8 中文会被从半个字符处切开而把行拆坏 —— 脚本
//     顶部的 `chcp 65001` 救不回来，因为出问题的行在它后面。这一条两个脚本的头部都写了。
//  2. 行尾不能**混着来**。两个脚本今天的形态其实不一样：build_portable.bat 在仓库里
//     是 LF（`git show HEAD:build_portable.bat` 里一个 CR 都没有），build_msi.bat 是
//     CRLF。两种都能跑 —— 打包链一直在跑 —— 但“一部分行 LF、一部分行 CRLF”不是这两种
//     中的任何一种：cmd 对 `if ( … )` 块与标签的解析会错位，而一次编辑器的部分重写
//     （或一次 sed 只动了几行）就能造出这个状态，脚本本身看不出任何异样。
//
// 所以这里不强求统一成 CRLF（那是一次有意的规范化提交，不归这条测试管），只禁止混行。
func TestBuildScriptsAreASCIIAndNotMixed(t *testing.T) {
	for _, name := range []string{"build_portable.bat", "build_msi.bat"} {
		p := filepath.Join("..", "..", name)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("读不到 %s：%v（打包脚本是仓库本体的一部分，缺了就是仓库坏了）", p, err)
		}
		for i, c := range b {
			if c >= 0x80 {
				t.Fatalf("%s 第 %d 字节是 0x%02X：.bat 必须 ASCII-only（cmd 会按 OEM 代码页切字节）", name, i, c)
			}
		}
		lines := bytes.Split(b, []byte("\n"))
		if n := len(lines); n > 0 && len(lines[n-1]) == 0 {
			lines = lines[:n-1] // 文件末尾换行产生的空片不算一行
		}
		crlf, lf := 0, 0
		for i, ln := range lines {
			switch {
			case len(ln) > 0 && ln[len(ln)-1] == '\r':
				crlf++
			case i == len(lines)-1:
				lf++ // 末行没有换行也算 LF 那一派
			default:
				lf++
			}
		}
		if crlf > 0 && lf > 0 {
			t.Errorf("%s 混了行尾（CRLF %d 行 / LF %d 行）：cmd 会错位解析块与标签", name, crlf, lf)
		}
	}
}

// TestVersionHasOneHome 钉住“版本号只有 VERSION 一个出处”。
//
// 版本号是更新机制、`tt version` 输出与资产文件名的共同输入：一旦 `VERSION` 与脚本里
// 的第二份字面量分叉，发出去的包会报一个从来没有过的版本，而当场看不出任何异常。
// 跨文件的一致性（VERSION ↔ web 的 package.json）由 `make version-check` 把关；
// 这条只钉“脚本里不许再出现字面量”，两者管的是不同的事。
func TestVersionHasOneHome(t *testing.T) {
	p := filepath.Join("..", "..", "VERSION")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读不到 %s：%v", p, err)
	}
	if v := strings.TrimSpace(string(b)); !versionShape.MatchString(v) {
		t.Errorf("VERSION 内容 %q 不是 MAJOR.MINOR.PATCH", v)
	}
	hardcoded := regexp.MustCompile(`(?m)^\s*set VERSION=[0-9]`)
	for _, name := range []string{"build_portable.bat", "build_msi.bat"} {
		src, err := os.ReadFile(filepath.Join("..", "..", name))
		if err != nil {
			t.Fatalf("读不到 %s：%v", name, err)
		}
		if loc := hardcoded.Find(src); loc != nil {
			t.Errorf("%s 里又写死了版本号（%q）：它必须读根目录的 VERSION 文件，否则两份会分叉", name, loc)
		}
	}
}

// versionShape 版本号形状：MAJOR.MINOR.PATCH，无 v 前缀（tag 才带 v）。
var versionShape = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

// TestRootHelpAndReadmeHaveNoStaleAdvice 断言根帮助与 README 都不再教已删除的调用。
//
// 读包外的文件是有先例的（`internal/dev/tzs/corpus_test.go` 读 `../../../engine/corpus.manifest`），
// 而 README.md 属于仓库本体、不是外部语料 —— 所以它**不在**就判失败，不用 Skip 糊过去。
func TestRootHelpAndReadmeHaveNoStaleAdvice(t *testing.T) {
	readmePath := filepath.Join("..", "..", "README.md")
	b, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("读不到 %s：%v（README.md 是仓库本体的一部分，缺了就是仓库坏了）", readmePath, err)
	}

	stale := []struct{ bad, why string }{
		{"走 call", "call 网关已删除：动词就是函数名，参数用 JSON 给"},
		{"call <fn>", "call 网关已删除"},
		{"tzs call", "call 网关已删除"},
	}
	for name, text := range map[string]string{
		"rootCmd.Long": rootCmd.Long,
		"README.md":    string(b),
	} {
		for _, s := range stale {
			if strings.Contains(text, s.bad) {
				t.Errorf("%s 里还在教 %s（%s）", name, s.bad, s.why)
			}
		}
	}
}

// TestRootHelpPointsAtNamedVerbs 断言根帮助把 .tzs 的入口说成"具名动词"。
//
// 只禁掉旧写法（上一条）不够：把那一行整句删掉也能让它通过，而调用方从此不知道
// 表单包该怎么改 —— 那比教错更坏。所以要有一条**正向**断言。
func TestRootHelpPointsAtNamedVerbs(t *testing.T) {
	long := rootCmd.Long
	for _, want := range []string{"tt dev tzs", "tt dev tzc", "tt dict", "tt debug"} {
		if !strings.Contains(long, want) {
			t.Errorf("根帮助里该出现 %s（四块能力的入口要一眼看全）", want)
		}
	}
	if !strings.Contains(long, "动词") {
		t.Errorf("根帮助里该说清 .tzs 的入口是具名动词，而不是删掉的 call 网关")
	}
}
