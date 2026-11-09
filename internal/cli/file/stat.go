package file

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"tt/internal/host"
)

type statJSON struct {
	Path    string `json:"path"`
	Type    string `json:"type"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"modTime"`
	Link    string `json:"link,omitempty"`
}

func newStatCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stat <路径...>",
		Short: "看类型/大小/时间/权限",
		Long:  `看一个或多个路径的属性（不跟随符号链接；--follow 时才解析目标）。`,
		Example: `  tt file stat --at 主机109 erp/apm/4gl/apm_apmt520.4gl
  tt file stat --at 主机109 erp/apm/4gl/link.4gl --follow`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var out []statJSON
			for _, arg := range args {
				p, err := cur.argPath(arg)
				if err != nil {
					return err
				}
				p, err = cur.resolveSafe(p)
				if err != nil {
					return err
				}
				e, err := cur.fs.Lstat(p)
				if err != nil {
					if host.NotFound(err) {
						return notFoundErr(cur.top, p)
					}
					return ioErr("%v", host.WrapPathErr(display(cur.top, p), err))
				}
				out = append(out, statJSON{
					Path: display(cur.top, p), Type: entryType(*e), Size: e.Size,
					Mode: e.ModeText(), ModTime: e.ModTime.Format("2006-01-02 15:04:05"),
					Link: e.Link,
				})
			}
			if wantJSON(cmd) {
				if len(out) == 1 {
					return commonPrintJSON(out[0])
				}
				return commonPrintJSON(out)
			}
			for i, e := range out {
				if i > 0 {
					fmt.Fprintln(os.Stdout)
				}
				fmt.Fprintf(os.Stdout, "  File: %s\n", e.Path)
				fmt.Fprintf(os.Stdout, "  Type: %s\n", e.Type)
				fmt.Fprintf(os.Stdout, "  Size: %d bytes (%s)\n", e.Size, humanBytes(e.Size))
				fmt.Fprintf(os.Stdout, "Modify: %s\n", e.ModTime)
				fmt.Fprintf(os.Stdout, "  Mode: %s\n", e.Mode)
				if e.Link != "" {
					fmt.Fprintf(os.Stdout, "  Link: -> %s\n", e.Link)
				}
			}
			return nil
		},
	}
}
