package testutil

// corpus.go —— 真实语料的**遍历**（`.tzc` 与 `.tzs` 两条管线共用）。
//
// 为什么抽到这里：`.tzc` 的语料回归（internal/cli/dev/corpus_test.go）与 `.tzs` 的
// （internal/dev/tzs/corpus_test.go）要找的是**同一批目录**，只是扩展名不同。抄两份的
// 后果不是「多几行」，而是两边会各自漂移：一边认了 TDEV_CORPUS、另一边只认 TTZS_CORPUS，
// 于是同一条命令在一台机器上跑全量、在另一台上静默跑零个包（"0 个包全部通过"是最坏的一种
// 假绿）。
//
// **「根怎么定」不在这里**：那是 internal/testenv（它还要读 config.local.json，而那份
// 配置的读取不能带 `testing` —— 本包被生产代码 import）。本文件只做**遍历与发现**：
// 给一个根，走出一批文件；不读包、不解压、不判好坏。语料的判定（sha pin、RoundTrip 定点、
// validate 零新增）住在各自的管线里。

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tt/internal/testenv"
)

// DefaultCorpusRoot 是语料的内置缺省。**定义在 internal/testenv**，这里只是个转发，
// 好让既有的引用（internal/dev/tzs/corpus_test.go 的跳过文案）不用改。
const DefaultCorpusRoot = testenv.DefaultCorpusRoot

// CorpusRoot 解析真实语料根，解析不出来时返回 ""。
//
// 解析只有一处实现：internal/testenv.CorpusRoot —— 优先级是
// **环境变量（TDEV_CORPUS / TTZS_CORPUS，谁先设谁说话）> config.local.json > 内置缺省**；
// 覆盖项设了却指不到目录时返回 ""，**不回落**（静默换一份别的语料会留下
// "我以为跑的是这份"这种错误结论）。这里只是个转发，好让既有的引用不用改。
func CorpusRoot() string { return testenv.CorpusRoot() }

// CorpusFiles 递归收集 root 下扩展名匹配（大小写不敏感）的语料文件，路径升序。
//
// excludePrefix 里的名字前缀会被跳过 —— 那是驱动脚本自己留下的草稿产物（_ai/_dw/_ed/
// _del/_ac，四个 batch 脚本各自排除集的并集；见 engine/make-manifest.sh）。排除是必须的：
// 那些文件会在两次运行之间出现或消失，把它们算进基线，基线就不可复现 —— 这正是
// engine/corpus.manifest 存在的理由。
func CorpusFiles(root, ext string, excludePrefix ...string) []string {
	var out []string
	_ = filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(p), ext) {
			return nil
		}
		base := filepath.Base(p)
		for _, pre := range excludePrefix {
			if strings.HasPrefix(base, pre) {
				return nil
			}
		}
		out = append(out, p)
		return nil
	})
	sort.Strings(out)
	return out
}
