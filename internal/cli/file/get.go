package file

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/spf13/cobra"

	"tt/internal/host"
	"tt/internal/output"
)

func newGetCmd() *cobra.Command {
	var outPath string
	var maxMB int64
	c := &cobra.Command{
		Use:   "get <路径> [-o <本地|->]",
		Short: "下载一个文件（默认同名落当前目录）",
		Long: `把一个文件从服务器下载到本地（服务器侧仍是只读）。

  -o <本地>   写到指定路径；"-" 表示 stdout
  --max MB    大小上限（默认 200 MB；0 = 不限）

先写 .part 再 rename：中断不会留下半截文件。`,
		Example: `  tt file get --at 主机109 erp/apm/4gl/apm_apmt520.4gl
  tt file get --at 主机109 erp/apm/4gl/apm_apmt520.4gl -o x.4gl
  tt file get --at 主机109 per/apm/apmt520.rpt -o -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := cur.argPath(args[0])
			if err != nil {
				return err
			}
			p, err = cur.resolveSafe(p)
			if err != nil {
				return err
			}
			self, err := cur.fs.Lstat(p)
			if err != nil {
				if host.NotFound(err) {
					return notFoundErr(cur.top, p)
				}
				return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
			}
			if self.IsDir() {
				return usageErr("get 不支持目录：%s（先用 tt file find <目录> 收窄，或 tt file ls <目录> 看文件）", display(cur.top, p))
			}
			limit := maxMB << 20
			if maxMB > 0 && self.Size > limit {
				return usageErr("文件 %s（%s）超过 --max %d MB；确要下载用 --max 0",
					display(cur.top, p), humanBytes(self.Size), maxMB)
			}
			f, err := cur.fs.Open(p)
			if err != nil {
				return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
			}
			defer f.Close()

			if outPath == "-" {
				_, truncated, err := host.CopyN(os.Stdout, f, limit)
				if err != nil {
					return ioErr("读取失败: %v", err)
				}
				if truncated {
					return usageErr("超过 --max %d MB：输出已截断", maxMB)
				}
				// -o - 是“把字节流交给调用方”（管道/重定向），不能再追加 JSON，
				// 否则文件内容后面会多一段信封，二进制直接损坏。
				return nil
			}

			dst := outPath
			if dst == "" {
				dst = path.Base(p)
			}
			abs, err := filepath.Abs(dst)
			if err != nil {
				return output.Errorf(output.CodeLocalIO, 5, "解析本地路径失败: %v", err)
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
				return output.Errorf(output.CodeLocalIO, 5, "创建本地目录失败: %v", err)
			}
			tmp, err := os.CreateTemp(filepath.Dir(abs), ".tt-get-*.part")
			if err != nil {
				return output.Errorf(output.CodeLocalIO, 5, "创建本地临时文件失败: %v", err)
			}
			tmpName := tmp.Name()
			n, truncated, copyErr := host.CopyN(tmp, f, limit)
			closeErr := tmp.Close()
			if copyErr != nil || truncated || closeErr != nil {
				os.Remove(tmpName)
				if truncated {
					return usageErr("超过 --max %d MB：下载中止（未落盘）", maxMB)
				}
				if copyErr != nil {
					return ioErr("读取失败: %v", copyErr)
				}
				return output.Errorf(output.CodeLocalIO, 5, "写本地文件失败: %v", closeErr)
			}
			if err := os.Rename(tmpName, abs); err != nil {
				os.Remove(tmpName)
				return output.Errorf(output.CodeLocalIO, 5, "保存失败: %v", err)
			}
			if wantJSON(cmd) {
				return commonPrintJSON(map[string]any{"ok": true, "path": display(cur.top, p), "local": abs, "bytes": n})
			}
			fmt.Fprintf(os.Stderr, "已保存 %s（%s）\n", abs, humanBytes(n))
			return nil
		},
	}
	c.Flags().StringVarP(&outPath, "output", "o", "", `本地路径（"-" = stdout）`)
	c.Flags().Int64Var(&maxMB, "max", 200, "大小上限（MB；0 = 不限）")
	return c
}
