package testkit

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestHelpersHaveNoLocalCopies 钉住"这几个跨包 helper 只有一处实现"。
//
// 为什么值得一条测试：这个仓库反复出现"图省事再抄一份"的坏味道，
// 而**抄错语料根的后果是静默的** —— internal/dev/testutil/corpus.go 与它的 README 都写着：
// "两边各自漂移的后果是一边认 TDEV_CORPUS、另一边只认 TTZS_CORPUS，于是同一条命令
// 在一台机器上跑全量、在另一台上**静默跑零个包**（'0 个包全部通过'是最坏的一种假绿）。"
//
// 收敛之前，internal/dev/{fence,pkgfile,tapfile,fgl} 各有一份 corpusRoot，全都只认
// TDEV_CORPUS；captureStdout 也有两份，一份返回退出码、一份不返回。
// 这条测试把那段散文变成断言：**再抄一份就是红的**。
//
// 文案不是新写的 —— 上面那句引文就是仓库里已有的原话。
func TestHelpersHaveNoLocalCopies(t *testing.T) {
	root := RepoRoot(t)
	cases := []struct {
		name string
		re   *regexp.Regexp
		why  string
	}{
		{
			name: "corpusRoot",
			re:   regexp.MustCompile(`(?m)^func corpusRoot\(`),
			why: "语料根发现只有一处：testutil.CorpusRoot（本包只加 skip 语境）。\n" +
				"    抄第二份的后果不是多几行 —— 是一边认 TDEV_CORPUS、另一边只认 TTZS_CORPUS，\n" +
				"    于是同一条命令在一台机器上跑全量、在另一台上静默跑零个包。\n" +
				"    改成 testkit.CorpusRoot(t)。",
		},
		{
			name: "captureStdout",
			re:   regexp.MustCompile(`(?m)^func captureStdout\(`),
			why: "stdout 捕获只有一处：testkit.CaptureStdout（边写边读，且有 64 KB 的边界判据）。\n" +
				"    从前那份'写临时文件'的实现没有退出码，两边的签名不一样。\n" +
				"    改成 testkit.CaptureStdout(t, func() int { …; return 0 })。",
		},
		{
			name: "silent",
			re:   regexp.MustCompile(`(?m)^func silent\(`),
			why: "吞输出只有一处：testkit.Silent。tzc 与 tzs 两个测试包都要吞输出，\n" +
				"    各抄一份就会有一天只改一份 —— 与 captureStdout 收敛到这里是同一个理由。\n" +
				"    改成 testkit.Silent(t, func() int { … })。",
		},
	}

	found := map[string][]string{}
	for _, c := range cases {
		found[c.name] = scanTestFiles(t, root, c.re)
	}
	for _, c := range cases {
		if hits := found[c.name]; len(hits) > 0 {
			t.Errorf("这些文件自己定义了 %s：\n  %s\n\n  %s",
				c.name, strings.Join(hits, "\n  "), c.why)
		}
	}
}

// scanTestFiles 返回 internal/ 下所有 `_test.go` 里命中 re 的文件（仓库相对路径，升序）。
func scanTestFiles(t *testing.T, root string, re *regexp.Regexp) []string {
	t.Helper()
	var out []string
	base := filepath.Join(root, "internal")
	err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		if re.Match(b) {
			rel, relErr := filepath.Rel(root, p)
			if relErr != nil {
				rel = p
			}
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("扫 %s 失败: %v", base, err)
	}
	return out
}

// TestRepoRootFindsTheModule 上溯要真的停在含 go.mod 的那一级 ——
// 这条是上面那条测试的地基：它定位错了，等于扫描扫了个空目录然后报"全部合格"
// （"0 个文件全部通过"正是我们要防的那种假绿）。
func TestRepoRootFindsTheModule(t *testing.T) {
	root := RepoRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("RepoRoot 返回的 %s 下没有 go.mod：%v", root, err)
	}
	// 反面判据：internal/ 必须真的在它下面，且扫得出文件 ——
	// 否则上面那条"没有本地副本"就是空转。
	if got := len(scanTestFiles(t, root, regexp.MustCompile(`(?m)^package `))); got < 50 {
		t.Fatalf("在 %s 下只扫到 %d 个 _test.go —— 定位错了，那条防重复的断言等于空转", root, got)
	}
}
