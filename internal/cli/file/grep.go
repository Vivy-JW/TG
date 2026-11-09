package file

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

// grepFileMax 单文件扫描上限：源码远小于此；超过的按“跳过”处理并提示。
const grepFileMax = 64 << 20

type grepOptions struct {
	ignoreCase bool
	lineNumber bool
	listOnly   bool
	recursive  bool
	maxMatches int
	include    []string
}

type grepMatch struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

type grepFileResult struct {
	idx     int
	out     []byte
	matches []grepMatch
	matched bool
	binary  bool
	skipped bool
	err     error
}

func newGrepCmd() *cobra.Command {
	var opt grepOptions
	var maxMatches int
	c := &cobra.Command{
		Use:   "grep [-rinl] [-m N] [--include G] <模式> <路径...>",
		Short: "在文件里找内容（RE2 语法；服务端零进程）",
		Long: `在文件里按正则找内容。默认 JSON（matches[]）；--human 才是 "路径:内容" 文本。

  -r             递归目录
  -i             忽略大小写
  -n             显示行号
  -l             只列有匹配的文件名
  -m N           每个文件最多 N 处匹配（0 = 不限）
  --include G    只扫基名匹配 G 的文件（可重复；如 --include '*.4gl'）

路径必须显式给（不会默认扫整个 $TOP）。二进制文件按 NUL 判断后跳过。
实现是纯 SFTP：文件逐个读过来在本地匹配，服务器上不起 grep 进程。`,
		Example: `  tt file grep -rn --at 主机109 'apmt520' erp/apm/4gl
  tt file grep -rln --at 主机109 --include '*.4gl' 'SELECT .*bmaa_t' erp/apm`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := args[0]
			paths := args[1:]
			opt.maxMatches = maxMatches
			if opt.ignoreCase {
				pattern = "(?i)" + pattern
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return usageErr("正则写错了（RE2 语法）：%v", err)
			}
			var files []string
			for _, arg := range paths {
				matches, err := cur.expandArg(arg)
				if err != nil {
					return err
				}
				for _, one := range matches {
					p, err := cur.argPath(one)
					if err != nil {
						return err
					}
					p, err = cur.resolveSafe(p)
					if err != nil {
						return err
					}
					collected, err := collectFiles(p, opt)
					if err != nil {
						return err
					}
					files = append(files, collected...)
				}
			}
			if len(files) == 0 {
				return silentExit{code: 1}
			}
			results := grepAll(files, re, opt)
			if wantJSON(cmd) {
				matches := []grepMatch{}
				for _, r := range results {
					if r.err == nil && r.matched {
						matches = append(matches, r.matches...)
					}
				}
				return commonPrintJSON(map[string]any{"ok": true, "matches": matches, "count": len(matches)})
			}
			any := false
			for _, r := range results {
				if r.err != nil {
					fmt.Fprintf(os.Stderr, "tt file grep: %s: %v\n", display(cur.top, files[r.idx]), r.err)
					continue
				}
				if r.binary {
					continue
				}
				if r.skipped {
					fmt.Fprintf(os.Stderr, "tt file grep: %s: 文件超过 %s，跳过\n",
						display(cur.top, files[r.idx]), humanBytes(grepFileMax))
					continue
				}
				if r.matched {
					any = true
					os.Stdout.Write(r.out)
				}
			}
			if !any {
				return silentExit{code: 1}
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&opt.recursive, "recursive", "r", false, "递归目录")
	c.Flags().BoolVarP(&opt.ignoreCase, "ignore-case", "i", false, "忽略大小写")
	c.Flags().BoolVarP(&opt.lineNumber, "line-number", "n", false, "显示行号")
	c.Flags().BoolVarP(&opt.listOnly, "files-with-matches", "l", false, "只列有匹配的文件名")
	c.Flags().IntVarP(&maxMatches, "max-count", "m", 0, "每个文件最多 N 处匹配（0 = 不限）")
	c.Flags().StringSliceVar(&opt.include, "include", nil, "只扫基名匹配该模式的文件（可重复）")
	return c
}

// collectFiles 把起点展开成文件列表：文件直接收；目录按 -r 递归。
func collectFiles(root string, opt grepOptions) ([]string, error) {
	self, err := cur.fs.Lstat(root)
	if err != nil {
		if host.NotFound(err) {
			return nil, notFoundErr(cur.top, root)
		}
		return nil, ioErr("%v", host.WrapPathErr(display(cur.top, root), err))
	}
	if !self.IsDir() {
		if includeOK(self.Name, opt.include) {
			return []string{root}, nil
		}
		return nil, nil
	}
	if !opt.recursive {
		return nil, usageErr("%s 是目录：加 -r 递归（grep -rn <模式> <目录>）", display(cur.top, root))
	}
	var files []string
	var walk func(dir string) error
	walk = func(dir string) error {
		entries, err := cur.fs.List(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.IsSymlink() {
				if !followLinks {
					continue
				}
				target, err := cur.resolveSafe(e.Path)
				if err != nil {
					continue // 指向 $TOP 之外 / 层数过多：跳过，不中断整次搜索
				}
				e.Path = target
				if st, err := cur.fs.Lstat(target); err == nil {
					e.Mode = st.Mode
				}
			}
			if e.IsDir() {
				if err := walk(e.Path); err != nil {
					return err
				}
				continue
			}
			if includeOK(e.Name, opt.include) {
				files = append(files, e.Path)
			}
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, ioErr("%v", host.WrapPathErr(display(cur.top, root), err))
	}
	return files, nil
}

func includeOK(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, pat := range patterns {
		if ok, err := path.Match(pat, name); err == nil && ok {
			return true
		}
	}
	return false
}

// grepAll 并发读文件、按原顺序汇总（输出稳定，AI 不会因并发看到乱序）。
func grepAll(files []string, re *regexp.Regexp, opt grepOptions) []grepFileResult {
	results := make([]grepFileResult, len(files))
	jobs := make(chan int)
	workers := 12
	if len(files) < workers {
		workers = len(files)
	}
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				r := grepOne(files[i], re, opt)
				r.idx = i
				results[i] = r
			}
		}()
	}
	for i := range files {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return results
}

func grepOne(p string, re *regexp.Regexp, opt grepOptions) grepFileResult {
	res := grepFileResult{}
	f, err := cur.fs.Open(p)
	if err != nil {
		res.err = err
		return res
	}
	defer f.Close()
	if st, err := f.Stat(); err == nil && st.Size > grepFileMax {
		res.skipped = true
		return res
	}
	probe := make([]byte, binaryProbe)
	n, _ := f.Read(probe)
	prefix := probe[:n]
	if isBinary(prefix) {
		res.binary = true
		return res
	}
	reader := bufio.NewReader(io.MultiReader(bytes.NewReader(prefix), f))
	sc := bufio.NewScanner(reader)
	sc.Buffer(make([]byte, 64<<10), 2<<20)

	var out bytes.Buffer
	displayPath := display(cur.top, p)
	line := 0
	matches := 0
	for sc.Scan() {
		line++
		text := sc.Text()
		if !re.MatchString(text) {
			continue
		}
		res.matched = true
		if opt.listOnly {
			res.matches = append(res.matches, grepMatch{Path: displayPath})
			out.WriteString(displayPath)
			out.WriteByte('\n')
			break
		}
		res.matches = append(res.matches, grepMatch{Path: displayPath, Line: line, Text: text})
		if opt.lineNumber {
			out.WriteString(displayPath)
			out.WriteByte(':')
			out.WriteString(strconv.Itoa(line))
			out.WriteByte(':')
		} else {
			out.WriteString(displayPath)
			out.WriteByte(':')
		}
		out.WriteString(text)
		out.WriteByte('\n')
		matches++
		if opt.maxMatches > 0 && matches >= opt.maxMatches {
			break
		}
	}
	if err := sc.Err(); err != nil {
		res.err = err
		return res
	}
	res.out = out.Bytes()
	return res
}
