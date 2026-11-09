package dict

import (
	"fmt"
	"os"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict/dbsync"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var dbStatusCmd = &cobra.Command{
	Use:   "status [<环境名>|local:<环境名>]",
	Short: "查看各环境本地字典库的覆盖情况（缺什么、各多少行）",
	Long: `本地字典库按环境分文件：<数据目录>/dict/<环境>.db。这里按环境列出各自的库，
不传环境看总览；点名环境看该环境各数据族是否已同步、各多少行、缺哪些表。

查询命令报「本地库尚未包含 XXX 数据」时用它定位；补齐用 tt dict db sync --env <环境>。
-d/--db 显式给路径时，只检查那一份（不按环境）。
只读，不改任何文件。`,
	Example: `  tt dict db status
  tt dict db status --env 主机109
  tt dict db status --env local:主机109
  tt dict db status -d D:\data\old_share.db`,
	Args:              cobra.MaximumNArgs(1),
	PersistentPreRunE: skipQuerySource,
	RunE: func(cmd *cobra.Command, args []string) error {
		env := strings.TrimSpace(common.Env)
		if len(args) == 1 {
			env = strings.TrimSpace(args[0])
		}
		if name, ok := localEnvName(env); ok {
			env = name
		} else if strings.EqualFold(env, "local") {
			env = ""
		}

		// 显式路径：只看这一份，不按环境。TDICT_DB 是旧单文件的全局开关，
		// 不该挡住“各环境总览”，所以这里只认 -d/--db。
		if p := strings.TrimSpace(dbPath); p != "" && env == "" {
			return inspectOne(config.AbsPath(p), "")
		}
		if env != "" {
			p := config.DictDBPathForEnv(dictCfgPath(), env)
			if x := strings.TrimSpace(dbPath); x != "" {
				p = config.AbsPath(x)
			}
			return inspectOne(p, env)
		}

		rows, dir, err := listLocalDBs()
		if err != nil {
			return err
		}
		if Format() != output.FormatTable {
			return emitOne(map[string]any{"dir": dir, "envs": rows})
		}
		fmt.Printf("本地字典库目录: %s\n\n", dir)
		if len(rows) == 0 {
			fmt.Println("(还没有配置环境；先 tt env list 或设置页添加环境)")
			return nil
		}
		headers := []string{"环境", "本地库", "状态", "大小", "同步时间"}
		out := make([][]string, 0, len(rows))
		for _, r := range rows {
			state, size, when := "未同步", "—", "—"
			if r.Exists {
				state, size, when = "已同步", humanSize(r.Size), r.Modified
			}
			out = append(out, []string{r.Env, r.Path, state, size, when})
		}
		output.PrintTable(headers, out)
		fmt.Println("\n同步某个环境: tt dict db sync --env <环境>")
		return nil
	},
}

type localDBRow struct {
	Env      string `json:"env"`
	Path     string `json:"path"`
	Exists   bool   `json:"exists"`
	Size     int64  `json:"size"`
	Modified string `json:"modified,omitempty"`
}

// listLocalDBs 列出配置里每个环境的本地库文件状态。
func listLocalDBs() ([]localDBRow, string, error) {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return nil, "", err
	}
	hosts, err := config.LoadHosts(path)
	if err != nil {
		return nil, "", err
	}
	dir := config.DictDBDirFor(path)
	rows := make([]localDBRow, 0, len(hosts.SSHs))
	for _, name := range hosts.Names() {
		p := config.DictDBPathForEnv(path, name)
		r := localDBRow{Env: name, Path: p}
		if st, err := os.Stat(p); err == nil {
			r.Exists = true
			r.Size = st.Size()
			r.Modified = st.ModTime().Format("2006-01-02 15:04")
		}
		rows = append(rows, r)
	}
	return rows, dir, nil
}

// inspectOne 看一份本地库的数据族覆盖；env 为空表示这是显式路径（文案不带环境）。
func inspectOne(path, env string) error {
	st, err := dbsync.InspectLocal(path)
	if err != nil {
		return err
	}
	if Format() != output.FormatTable {
		return emitOne(st)
	}
	if !st.Exists {
		if env != "" {
			fmt.Printf("环境 %s 的本地库不存在: %s\n", env, st.Path)
			fmt.Printf("建立: tt dict db sync --env %s\n", env)
		} else {
			fmt.Printf("本地库不存在: %s\n", st.Path)
			fmt.Println("建立: tt dict db sync --env <环境> -d <路径>")
		}
		return nil
	}
	printLocalStatus(st)
	if env != "" {
		fmt.Printf("\n环境: %s\n", env)
	}
	return nil
}

func printLocalStatus(st *dbsync.LocalStatus) {
	fmt.Printf("本地库: %s  (%s, %s)\n\n", st.Path, humanSize(st.Size), st.Modified.Format("2006-01-02 15:04"))

	headers := []string{"状态", "数据族", "行数", "缺表", "依赖命令"}
	rows := make([][]string, 0, len(st.Families))
	missingAny := false
	for _, f := range st.Families {
		mark, missing, cnt := "✓", "-", fmt.Sprintf("%d", f.Rows)
		if !f.Complete {
			missingAny = true
			missing = strings.Join(f.Missing, ",")
			cnt = "—"
			mark = "✗"
			if f.Present > 0 {
				mark = "✗ 部分"
			}
		}
		rows = append(rows, []string{mark, f.Name, cnt, missing, strings.Join(f.Commands, ",")})
	}
	output.PrintTable(headers, rows)

	if missingAny {
		fmt.Println("\n✗ = 该族的字典表不全，依赖它的命令会**整体**报错（不是部分可用）；行数显示 — 表示该族不全、行数无意义。")
		fmt.Println("补齐: tt dict db sync --env <环境>；临时也可 `tt dict <命令> --env <环境>` 远程直查。")
	}
}

// humanSize 把字节数写成人类可读（与 mirror 进度里的写法一致）。
func humanSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.2f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func init() {
	dbCmd.AddCommand(dbStatusCmd)
}
