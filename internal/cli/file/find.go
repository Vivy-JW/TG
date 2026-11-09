package file

import (
	"os"
	"path"
	"sort"
	"sync"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

type findOptions struct {
	name     string
	typ      string
	maxDepth int
}

func newFindCmd() *cobra.Command {
	var opt findOptions
	var maxDepth int
	c := &cobra.Command{
		Use:   "find <路径...> [-name G] [-type f|d|l] [-maxdepth N]",
		Short: "按名字/类型找文件（只读元数据，服务端零进程）",
		Long: `遍历目录找文件。默认 JSON（数组）；--human 才一行一个路径（相对 $TOP，可直接回喂给本组命令）。

  -name G       基名匹配 G（支持 * ? 通配）
  -type f|d|l   只看普通文件 / 目录 / 符号链接
  -maxdepth N   最多向下 N 层（0 = 只看起点本身）

不接受 -exec/-delete/-fprint 这类会写或执行的动作 —— 本条命令只读。`,
		Example: `  tt file find --at 主机109 erp/apm -name '*apmi004*'
  tt file find --at 主机109 erp/apm -type f -maxdepth 2 -name '*.4gl'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opt.maxDepth = maxDepth
			if opt.typ != "" && opt.typ != "f" && opt.typ != "d" && opt.typ != "l" {
				return usageErr("-type 只支持 f / d / l（收到 %q）", opt.typ)
			}
			var roots []string
			for _, arg := range args {
				p, err := cur.argPath(arg)
				if err != nil {
					return err
				}
				p, err = cur.resolveSafe(p)
				if err != nil {
					return err
				}
				roots = append(roots, p)
			}
			found, err := findRun(roots, opt)
			if err != nil {
				return err
			}
			if wantJSON(cmd) {
				type item struct {
					Path string `json:"path"`
					Type string `json:"type"`
					Size int64  `json:"size"`
				}
				out := make([]item, 0, len(found))
				for _, e := range found {
					out = append(out, item{Path: display(cur.top, e.Path), Type: entryType(e), Size: e.Size})
				}
				return commonPrintJSON(out)
			}
			for _, e := range found {
				os.Stdout.WriteString(display(cur.top, e.Path) + "\n")
			}
			return nil
		},
	}
	c.Flags().StringVar(&opt.name, "name", "", "基名匹配（支持 * ?）")
	c.Flags().StringVar(&opt.typ, "type", "", "只看 f(文件) / d(目录) / l(链接)")
	c.Flags().IntVar(&maxDepth, "maxdepth", -1, "最多向下 N 层（0 = 只看起点；默认不限）")
	return c
}

// findItem 遍历队列里的一项：属性已由 List/Lstat 带回，**不再逐项 Lstat**
// （多一次往返在 VPN 上就是几十毫秒，整棵树能差出一分钟）。
type findItem struct {
	e     host.Entry
	depth int
}

// findRun 逐层并发遍历：同一层的目录一起 List（12 路），匹配结果最后按路径排序，
// 输出稳定。SFTP 元数据只能一条条问，串行遍历大目录会非常慢。
func findRun(roots []string, opt findOptions) ([]host.Entry, error) {
	matches := make([]host.Entry, 0, 64)
	for _, root := range roots {
		self, err := cur.fs.Lstat(root)
		if err != nil {
			if host.NotFound(err) {
				return nil, notFoundErr(cur.top, root)
			}
			return nil, ioErr("%v", host.WrapPathErr(display(cur.top, root), err))
		}
		level := []findItem{{e: *self, depth: 0}}
		seen := map[string]bool{}
		for len(level) > 0 {
			var dirs []findItem
			for _, it := range level {
				self := it.e
				real := it.e
				linkResolved := ""
				if self.IsSymlink() && followLinks {
					target, err := cur.resolveSafe(self.Path)
					if err != nil {
						continue // 指向 $TOP 之外 / 层数过多：跳过这一条，不中断整次查找
					}
					if st, err := cur.fs.Lstat(target); err == nil {
						real = *st
						linkResolved = target
					}
				}
				if findMatch(self, real, opt) {
					e := self
					e.Path = self.Path
					matches = append(matches, e)
				}
				isDir := self.IsDir() || (linkResolved != "" && real.IsDir())
				if !isDir {
					continue
				}
				if opt.maxDepth >= 0 && it.depth >= opt.maxDepth {
					continue
				}
				dirPath := self.Path
				if linkResolved != "" {
					dirPath = linkResolved
				}
				if seen[dirPath] {
					continue
				}
				seen[dirPath] = true
				dirs = append(dirs, findItem{e: host.Entry{Path: dirPath, Mode: real.Mode}, depth: it.depth + 1})
			}
			if len(dirs) == 0 {
				break
			}
			level, err = listDirs(dirs)
			if err != nil {
				return nil, err
			}
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Path < matches[j].Path })
	return matches, nil
}

// listDirs 并发列一批目录，把子项拼成下一层（保持确定的目录顺序）。
func listDirs(dirs []findItem) ([]findItem, error) {
	paths := make([]string, len(dirs))
	for i := range dirs {
		paths[i] = dirs[i].e.Path
	}
	lists, err := listPathsConcurrent(paths)
	if err != nil {
		return nil, err
	}
	var next []findItem
	for i, entries := range lists {
		for _, e := range entries {
			next = append(next, findItem{e: e, depth: dirs[i].depth})
		}
	}
	return next, nil
}

// listPathsConcurrent 并发列多个目录（12 路）。
//
// SFTP 的目录列表只能一条条问；串行遍历大目录在 VPN 上会慢到不可用
// （实测 erp/apm 全树串行 73 秒，并发后约 1 秒）。
func listPathsConcurrent(paths []string) ([][]host.Entry, error) {
	lists := make([][]host.Entry, len(paths))
	errs := make([]error, len(paths))
	sem := make(chan struct{}, 12)
	var wg sync.WaitGroup
	for i := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			lists[i], errs[i] = cur.fs.List(paths[i])
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			return nil, ioErr("%v", host.WrapPathErr(display(cur.top, paths[i]), err))
		}
	}
	return lists, nil
}

// findMatch 名字 + 类型都要满足（未给的条件不参与判定）。
func findMatch(self, real host.Entry, opt findOptions) bool {
	if opt.name != "" {
		ok, err := path.Match(opt.name, self.Name)
		if err != nil || !ok {
			return false
		}
	}
	switch opt.typ {
	case "f":
		return !self.IsDir() && !self.IsSymlink() && real.Mode.IsRegular()
	case "d":
		return self.IsDir() || (followLinks && self.IsSymlink() && real.IsDir())
	case "l":
		return self.IsSymlink()
	default:
		return true
	}
}

// NormalizeFindArgs 把 "tt file find -name X" 里的 Unix 单横线长选项改成 pflag 认的
// 双横线（pflag 把 -name 当成 -n -a -m -e 一串缩写）。
//
// 只在 file find 这一段生效：其他动词的长选项本来就是双横线（--include），
// 单字母缩写（-r/-n/-m）不受影响。写操作的选项也一并改成双横线 —— 目的不是支持它们，
// 而是让拒绝时的报错说清是哪个选项被拒（unknown flag: --exec）。
func NormalizeFindArgs(args []string) []string {
	start := -1
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "file" && args[i+1] == "find" {
			start = i + 2
			break
		}
	}
	if start < 0 {
		return args
	}
	out := append([]string(nil), args...)
	rename := map[string]string{
		"-name": "--name", "-type": "--type", "-maxdepth": "--maxdepth",
		"-exec": "--exec", "-execdir": "--execdir", "-delete": "--delete",
		"-ok": "--ok", "-fprint": "--fprint", "-fprintf": "--fprintf",
	}
	for i := start; i < len(out); i++ {
		if to, ok := rename[out[i]]; ok {
			out[i] = to
		}
	}
	return out
}
