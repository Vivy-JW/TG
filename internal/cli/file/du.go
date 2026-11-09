package file

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/host"
)

func newDuCmd() *cobra.Command {
	var summary bool
	c := &cobra.Command{
		Use:   "du [-s] [--human] [<路径>]",
		Short: "统计占用大小（按文件大小累计，1K 块）",
		Long: `统计一个目录（默认 $TOP）占用的大小。默认 JSON；--human 逐目录打印文本。

  -s        只输出汇总
  --human   人性化大小（-h 被 help 占用）

SFTP 拿不到磁盘块占用，所以这里按文件大小累计（与 du 的默认口径略有差异）。`,
		Example: `  tt file du -s --at 主机109 erp/apm/4gl
  tt file du --at 主机109 erp/apm`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			arg := ""
			if len(args) == 1 {
				arg = args[0]
			}
			p, err := cur.argPath(arg)
			if err != nil {
				return err
			}
			p, err = cur.resolveSafe(p)
			if err != nil {
				return err
			}
			sums, err := duSums(p)
			if err != nil {
				return err
			}
			if wantJSON(cmd) {
				type duEntry struct {
					Path   string `json:"path"`
					Bytes  int64  `json:"bytes"`
					Blocks int64  `json:"blocks"`
				}
				entry := func(d string, n int64) duEntry {
					return duEntry{Path: display(cur.top, d), Bytes: n, Blocks: (n + 1023) / 1024}
				}
				if summary {
					return commonPrintJSON(map[string]any{"ok": true, "total": entry(p, sums[p])})
				}
				dirs := make([]string, 0, len(sums))
				for d := range sums {
					dirs = append(dirs, d)
				}
				sort.Strings(dirs)
				out := make([]duEntry, 0, len(dirs))
				for _, d := range dirs {
					out = append(out, entry(d, sums[d]))
				}
				return commonPrintJSON(map[string]any{"ok": true, "dirs": out})
			}
			if summary {
				printDu(sums[p], display(cur.top, p), common.Human)
				return nil
			}
			dirs := make([]string, 0, len(sums))
			for d := range sums {
				dirs = append(dirs, d)
			}
			sort.Strings(dirs)
			for _, d := range dirs {
				printDu(sums[d], display(cur.top, d), common.Human)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&summary, "summarize", "s", false, "只输出汇总")
	return c
}

// duSums 逐层并发遍历，返回每个目录的累计大小。
//
// 先按目录收集直属文件大小，再按深度从深到浅把子目录的和加到父目录
// （并发的 BFS 拿不到后序遍历，这一步补上）。
func duSums(root string) (map[string]int64, error) {
	self, err := cur.fs.Lstat(root)
	if err != nil {
		if host.NotFound(err) {
			return nil, notFoundErr(cur.top, root)
		}
		return nil, ioErr("%v", host.WrapPathErr(display(cur.top, root), err))
	}
	if !self.IsDir() {
		return map[string]int64{root: self.Size}, nil
	}
	sums := map[string]int64{root: 0}
	parent := map[string]string{}
	type lvl struct {
		path  string
		depth int
	}
	level := []lvl{{path: root, depth: 0}}
	seen := map[string]bool{}
	for len(level) > 0 {
		paths := make([]string, len(level))
		for i, it := range level {
			paths[i] = it.path
		}
		lists, err := listPathsConcurrent(paths)
		if err != nil {
			return nil, err
		}
		next := make([]lvl, 0, len(paths))
		for i, entries := range lists {
			dir := level[i].path
			for _, e := range entries {
				if e.IsSymlink() {
					continue // 不跟随、也不计链接本身的大小
				}
				if e.IsDir() {
					if seen[e.Path] {
						continue
					}
					seen[e.Path] = true
					sums[e.Path] = 0
					parent[e.Path] = dir
					next = append(next, lvl{path: e.Path, depth: level[i].depth + 1})
					continue
				}
				sums[dir] += e.Size
			}
		}
		level = next
	}
	// 由深到浅汇总子目录
	dirs := make([]string, 0, len(sums))
	for d := range sums {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.Count(dirs[i], "/") > strings.Count(dirs[j], "/")
	})
	for _, d := range dirs {
		if p, ok := parent[d]; ok {
			sums[p] += sums[d]
		}
	}
	return sums, nil
}

func printDu(bytes int64, name string, human bool) {
	if human {
		fmt.Fprintf(os.Stdout, "%s\t%s\n", humanBytes(bytes), name)
		return
	}
	blocks := (bytes + 1023) / 1024
	fmt.Fprintf(os.Stdout, "%d\t%s\n", blocks, name)
}
