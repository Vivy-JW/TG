package testkit

import (
	"testing"

	"tt/internal/dev/testutil"
)

// CorpusRoot 解析真实语料根；解析不出来就**跳过**这条用例。
//
// 解析本身只有一处实现（testutil.CorpusRoot：两个覆盖变量都认、设了却指不到目录时
// 不回落缺省），这里只加"没有语料该怎么收场"这层语境。
//
// **为什么非要收敛**：从前 internal/dev/{fence,pkgfile,tapfile,fgl} 各抄了一份，
// 而它们**只认 TDEV_CORPUS、不认 TTZS_CORPUS**（fgl 那份还连着目录都不检查）。
// 这正是 internal/dev/testutil/README.md 亲手警告过的那个事故 ——
// "两边各自漂移的后果是一边认 TDEV_CORPUS、另一边只认 TTZS_CORPUS，于是同一条命令
// 在一台机器上跑全量、在另一台上**静默跑零个包**（'0 个包全部通过'是最坏的一种假绿）"。
// 所以这不是"少几行"，是修一个静默的假绿。
func CorpusRoot(t *testing.T) string {
	t.Helper()
	if root := testutil.CorpusRoot(); root != "" {
		return root
	}
	t.Skipf("没有真实语料：设 TDEV_CORPUS 或 TTZS_CORPUS 指到一份，或准备 %s"+
		"（覆盖变量设了却指不到目录时**不会**回落到缺省 —— 那是刻意的）",
		testutil.DefaultCorpusRoot)
	return ""
}

// CorpusFiles 收集语料里扩展名匹配的文件（ext 带点，如 ".tzc"）；一个都没有就**跳过**。
//
// 遍历规则（含要跳过哪些驱动脚本留下的草稿产物）单源在 testutil.CorpusFiles，
// 这里只加"一个都没找到"的收场。
func CorpusFiles(t *testing.T, ext string) []string {
	t.Helper()
	root := CorpusRoot(t)
	out := testutil.CorpusFiles(root, ext)
	if len(out) == 0 {
		t.Skipf("%s 下没有 %s", root, ext)
	}
	return out
}
