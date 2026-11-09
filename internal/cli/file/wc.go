package file

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

type wcCount struct {
	lines int64
	words int64
	bytes int64
	name  string
}

func newWcCmd() *cobra.Command {
	var onlyLines bool
	c := &cobra.Command{
		Use:   "wc [-l] <文件...>",
		Short: "统计行/词/字节",
		Long:  `统计行数、词数、字节数（-l 只输出行数）。默认 JSON；--human 使用 Unix wc 文本格式。`,
		Example: `  tt file wc --at 主机109 erp/apm/4gl/apm_apmt520.4gl
  tt file wc -l --at 主机109 erp/apm/4gl/*.4gl`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var files []string
			for _, arg := range args {
				matches, err := cur.expandArg(arg)
				if err != nil {
					return err
				}
				files = append(files, matches...)
			}
			var counts []wcCount
			var total wcCount
			for _, arg := range files {
				p, err := cur.argPath(arg)
				if err != nil {
					return err
				}
				p, err = cur.resolveSafe(p)
				if err != nil {
					return err
				}
				if self, err := cur.fs.Lstat(p); err == nil && self.IsDir() {
					return usageErr("wc 不支持目录：%s", display(cur.top, p))
				}
				f, err := cur.fs.Open(p)
				if err != nil {
					if host.NotFound(err) {
						return notFoundErr(cur.top, p)
					}
					return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
				}
				cnt, err := countFile(f)
				f.Close()
				if err != nil {
					return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
				}
				cnt.name = display(cur.top, p)
				counts = append(counts, cnt)
				total.lines += cnt.lines
				total.words += cnt.words
				total.bytes += cnt.bytes
			}
			type wcJSON struct {
				Path  string `json:"path"`
				Lines int64  `json:"lines"`
				Words int64  `json:"words"`
				Bytes int64  `json:"bytes"`
			}
			if wantJSON(cmd) {
				out := make([]wcJSON, 0, len(counts))
				for _, c := range counts {
					out = append(out, wcJSON{Path: c.name, Lines: c.lines, Words: c.words, Bytes: c.bytes})
				}
				payload := map[string]any{"ok": true, "files": out}
				if len(counts) > 1 {
					payload["total"] = wcJSON{Path: "总计", Lines: total.lines, Words: total.words, Bytes: total.bytes}
				}
				return commonPrintJSON(payload)
			}
			for _, c := range counts {
				printWc(c, onlyLines)
			}
			if len(counts) > 1 {
				total.name = "总计"
				printWc(total, onlyLines)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&onlyLines, "lines", "l", false, "只输出行数")
	return c
}

func countFile(r io.Reader) (wcCount, error) {
	var cnt wcCount
	cr := &countingReader{r: r}
	sc := bufio.NewScanner(cr)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		cnt.lines++
		cnt.words += int64(len(strings.Fields(sc.Text())))
	}
	if err := sc.Err(); err != nil {
		return cnt, err
	}
	cnt.bytes = cr.n
	return cnt, nil
}

// countingReader 统计实际读到的字节数（Scanner 会吃掉换行，不能靠行内容累加）。
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

func printWc(c wcCount, onlyLines bool) {
	if onlyLines {
		fmt.Fprintf(os.Stdout, "%8d %s\n", c.lines, c.name)
		return
	}
	fmt.Fprintf(os.Stdout, "%8d %8d %8d %s\n", c.lines, c.words, c.bytes, c.name)
}
