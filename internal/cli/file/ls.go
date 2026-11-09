package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/host"
	"tt/internal/output"
)

// ls 一条记录（--json 用）。
type lsJSON struct {
	OK      bool      `json:"ok"`
	Path    string    `json:"path"`
	Entries []lsEntry `json:"entries"`
}

type lsEntry struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // file | dir | link | other
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
	Link    string `json:"link,omitempty"`
}

func entryType(e host.Entry) string {
	switch {
	case e.IsSymlink():
		return "link"
	case e.IsDir():
		return "dir"
	case e.Mode.IsRegular():
		return "file"
	default:
		return "other"
	}
}

func newLsCmd() *cobra.Command {
	var long, all bool
	c := &cobra.Command{
		Use:   "ls [-la] [--human] [<路径>]",
		Short: "列出目录（默认 $TOP）",
		Long: `列出 $TOP 内的一个目录；不给路径就是 $TOP。

  -l  长格式（权限/大小/时间/名字）
  -a  含隐藏文件（名字以 . 开头）
  --human  大小人性化（配合 -l；-h 被 help 占用）`,
		Example: `  tt file ls --at 主机109
  tt file ls -la --at 主机109 erp/apm/4gl`,
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
			self, err := cur.fs.Lstat(p)
			if err != nil {
				if host.NotFound(err) {
					return notFoundErr(cur.top, p)
				}
				return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
			}
			entries := []host.Entry{*self}
			if self.IsDir() {
				entries, err = cur.fs.List(p)
				if err != nil {
					return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
				}
			}
			shown := entries[:0]
			for _, e := range entries {
				if !all && strings.HasPrefix(e.Name, ".") {
					continue
				}
				shown = append(shown, e)
			}
			if wantJSON(cmd) {
				out := lsJSON{OK: true, Path: display(cur.top, p), Entries: make([]lsEntry, 0, len(shown))}
				for _, e := range shown {
					out.Entries = append(out.Entries, lsEntry{
						Name: e.Name, Type: entryType(e), Size: e.Size,
						Mode: e.ModeText(), ModTime: e.ModTime.Format("2006-01-02 15:04:05"),
						Link: e.Link,
					})
				}
				return commonPrintJSON(out)
			}
			for _, e := range shown {
				name := e.Name
				if e.Link != "" {
					name += " -> " + e.Link
				}
				if !long {
					fmt.Fprintln(os.Stdout, name)
					continue
				}
				size := fmt.Sprintf("%d", e.Size)
				if common.Human {
					size = humanBytes(e.Size)
				}
				fmt.Fprintf(os.Stdout, "%s %8s %s  %s\n",
					e.ModeText(), size, e.ModTime.Format("2006-01-02 15:04"), name)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&long, "long", "l", false, "长格式（权限/大小/时间/名字）")
	c.Flags().BoolVarP(&all, "all", "a", false, "含隐藏文件")
	return c
}

// commonPrintJSON 包一层，避免每个命令都 import output。
func commonPrintJSON(v any) error { return output.WriteJSONValue(os.Stdout, v) }
