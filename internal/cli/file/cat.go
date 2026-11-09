package file

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

type catFile struct {
	Path     string `json:"path"`
	Content  string `json:"content,omitempty"`
	Encoding string `json:"encoding,omitempty"` // base64 = 二进制
	Bytes    int    `json:"bytes"`
}

func newCatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cat <文件...>",
		Short: "输出文件内容（多文件直接拼接）",
		Long: `输出一个或多个文件的内容。多个文件按参数顺序直接拼接（Unix cat 语义，无分隔头）。
参数支持最后一段通配（如 erp/apm/4gl/*.4gl），由本工具展开。

默认 JSON（files[].content）；--human 才是原样文本。
含 NUL 字节的文件默认拒绝；确要输出加 --binary（JSON 里用 base64）。`,
		Example: `  tt file cat --at 主机109 erp/apm/4gl/apmi004.4gl
  tt file cat --human --at 主机109 erp/apm/4gl/apmi004.4gl
  tt file cat --at 主机109 'erp/apm/4gl/apmi004*.4gl'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var files []catFile
			for _, arg := range args {
				matches, err := cur.expandArg(arg)
				if err != nil {
					return err
				}
				for _, one := range matches {
					data, p, err := readWhole(one)
					if err != nil {
						return err
					}
					f := catFile{Path: display(cur.top, p), Bytes: len(data)}
					if isBinary(data) || !utf8.Valid(data) {
						if !allowBinary {
							return usageErr("二进制文件（含 NUL 或非 UTF-8），不往终端喷：%s", f.Path).
								WithHint("要落盘用 tt file get <路径> -o <本地文件>；确要输出原始字节加 --human --binary（或 tt file get <路径> -o -）")
						}
						f.Content = base64.StdEncoding.EncodeToString(data)
						f.Encoding = "base64"
					} else {
						f.Content = string(data)
					}
					files = append(files, f)
				}
			}
			if wantJSON(cmd) {
				return commonPrintJSON(map[string]any{"ok": true, "files": files})
			}
			for _, f := range files {
				if f.Encoding == "base64" {
					data, _ := base64.StdEncoding.DecodeString(f.Content)
					if _, err := os.Stdout.Write(data); err != nil {
						return ioErr("%v", err)
					}
					continue
				}
				if _, err := fmt.Fprint(os.Stdout, f.Content); err != nil {
					return ioErr("%v", err)
				}
			}
			return nil
		},
	}
}

// readWhole 读整份文件（cat 要完整内容；二进制检查在这里做）。
func readWhole(arg string) ([]byte, string, error) {
	p, err := cur.argPath(arg)
	if err != nil {
		return nil, "", err
	}
	p, err = cur.resolveSafe(p)
	if err != nil {
		return nil, "", err
	}
	self, err := cur.fs.Lstat(p)
	if err != nil {
		if host.NotFound(err) {
			return nil, "", notFoundErr(cur.top, p)
		}
		return nil, "", ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
	}
	if self.IsDir() {
		return nil, "", usageErr("cat 不支持目录：%s", display(cur.top, p)).
			WithHint("用 tt file ls <目录> 看有哪些文件；逐一下载用 tt file get <路径> -o <本地文件>")
	}
	f, _, err := openChecked(p)
	if err != nil {
		return nil, "", err
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, "", ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
	}
	return data, p, nil
}

func newHeadCmd() *cobra.Command {
	var n int
	c := &cobra.Command{
		Use:     "head [-n N] <文件...>",
		Short:   "前 N 行（默认 10）",
		Long:    `输出每个文件的前 N 行；默认 JSON（files[].lines），--human 才是带 ==> 头的文本。`,
		Example: `  tt file head -n 40 --at 主机109 erp/apm/4gl/apmi004.4gl`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runText(cmd, args, false, n)
		},
	}
	c.Flags().IntVarP(&n, "lines", "n", 10, "输出前 N 行")
	return c
}

func newTailCmd() *cobra.Command {
	var n int
	c := &cobra.Command{
		Use:     "tail [-n N] <文件...>",
		Short:   "后 N 行（默认 10）",
		Long:    `输出每个文件的后 N 行；只读文件尾部若干块，不整份下载。`,
		Example: `  tt file tail -n 40 --at 主机109 erp/apm/4gl/apmi004.4gl`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runText(cmd, args, true, n)
		},
	}
	c.Flags().IntVarP(&n, "lines", "n", 10, "输出后 N 行")
	return c
}
