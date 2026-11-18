package cli

// tg docs：随包的两份 Markdown 手册（Genero BDL / TOP GP）解到哪了。
//
// 为什么单开一条命令：手册目录随安装形态变 —— 便携包在 exe 旁边、MSI 在 %APPDATA%\tg、
// 用户还能用 `bdldoc.dir` / `gpdoc.dir` 改。所以技能手册与协作者都不能写死路径，只能
// "问一条命令"。`tg install bdl --dry-run` 也能问出来，但那是"别写盘"的意思，不是查询；
// 这里给一个正当入口，一份输出同时回答两份手册。
//
// 判断口径与设置页完全同源（config.DocKind.StatusFor）：不遍历文件、不校验完整性 ——
// 那是 `--long` 的事，几千个文件走一遍要时间。

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/output"
)

// docsSchemaVersion JSON 信封的标识（与 tg sdd / tg outline 同风格）。
const docsSchemaVersion = "tg.docs/v1"

// docEntry 一类文档的落点与状态。
type docEntry struct {
	Key        string `json:"key"`                  // 配置节名：bdldoc / gpdoc
	Title      string `json:"title"`                // 人读标题（与设置页卡片一致）
	Dir        string `json:"dir"`                  // 生效目录（绝对路径）
	Configured string `json:"configured,omitempty"` // 配置里显式写的值；空 = 缺省落点
	Exists     bool   `json:"exists"`               // 目录存在
	Installed  bool   `json:"installed"`            // 目录里有东西
	Files      int    `json:"files,omitempty"`      // --long 才有
	Bytes      int64  `json:"bytes,omitempty"`      // --long 才有
	ZipName    string `json:"zipName"`              // 随包 zip 名（没装时提示去哪找）
}

// docsReport 整份输出。
type docsReport struct {
	Schema     string     `json:"schema"`
	ConfigPath string     `json:"configPath"`
	Docs       []docEntry `json:"docs"`
}

func newDocsCmd() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "查看随包的两份 Markdown 手册解到哪了（BDL / TOP GP）",
		Long: `查看随包的两份 Markdown 手册解到哪了。

  bdldoc  Genero BDL User Guide（tg install bdl）
  gpdoc   TOP GP 技术手册（tg install gpdoc）

"解到哪" 由配置与安装形态决定：便携包在 tg.exe 旁边，普通安装在 %APPDATA%\tg，
也可以用 tg config set bdldoc.dir / gpdoc.dir 换位置。所以**别写死路径** —— 需要原文时
先跑这条命令，拿其中 dir 那一栏。

它只回答"在不在"，不校验完整性；--long 会顺带数一遍文件数与体积（几千个文件，慢一点）。`,
		Example: `  tg docs                # 两份手册：目录 + 装没装
  tg docs --long         # 顺带数文件数与体积
  tg docs --json         # 机器读：{"schema":"tg.docs/v1","docs":[{"key":"bdldoc","dir":...}]}
  tg docs --format table # 强制人读表`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := common.ResolveConfig(false)
			rep := docsReport{Schema: docsSchemaVersion, ConfigPath: cfgPath}
			for _, k := range []config.DocKind{config.DocBDL, config.DocGP} {
				rep.Docs = append(rep.Docs, docEntryOf(k, cfgPath, long))
			}
			if common.OutputFormat() != output.FormatTable {
				return output.Emit(cmd.OutOrStdout(), output.Options{
					Format: common.OutputFormat(), Meta: output.Meta{}, Data: rep,
				})
			}
			out := cmd.OutOrStdout()
			rows := make([][]string, 0, len(rep.Docs))
			for _, d := range rep.Docs {
				state := "未安装"
				switch {
				case d.Installed && d.Files > 0:
					state = fmt.Sprintf("已解出 %s / %d 个文件", humanBytes(d.Bytes), d.Files)
				case d.Installed:
					state = "已解出文档"
				case d.Exists:
					state = "目录存在，但是空的"
				}
				rows = append(rows, []string{d.Title, d.Dir, state})
			}
			if err := output.WriteTable(out, []string{"手册", "目录", "状态"}, rows); err != nil {
				return err
			}
			for _, d := range rep.Docs {
				if d.Installed {
					continue
				}
				fmt.Fprintf(out, "\n%s 还没装：把 %s 放到 tg.exe 旁边（或 %s）后跑 tg install %s\n",
					d.Title, d.ZipName, filepath.Dir(d.Dir), installDocName(d.Key))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&long, "long", false, "顺带数一遍文件数与体积（几千个文件，慢一点）")
	return cmd
}

// docEntryOf 汇总一类文档。count 为真时再遍历一遍算规模。
func docEntryOf(k config.DocKind, cfgPath string, count bool) docEntry {
	st := k.StatusFor(cfgPath)
	e := docEntry{
		Key:        k.Key,
		Title:      k.Title,
		Dir:        st.Dir,
		Configured: st.Configured,
		Exists:     st.Exists,
		Installed:  st.Installed,
		ZipName:    k.ZipName,
	}
	if count && st.Exists {
		e.Files, e.Bytes = docScale(st.Dir)
	}
	return e
}

// docScale 数目录里的文件数与总字节。读不到的子目录直接跳过 —— 这是给人看规模用的，
// 不该因为一个没权限的子目录整条命令失败。
func docScale(root string) (int, int64) {
	var files int
	var bytes int64
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		files++
		if fi, ierr := d.Info(); ierr == nil {
			bytes += fi.Size()
		}
		return nil
	})
	return files, bytes
}

// installDocName 配置节名 → 安装子命令名（bdldoc → bdl，gpdoc → gpdoc）。
func installDocName(key string) string {
	switch key {
	case config.DocBDL.Key:
		return "bdl"
	case config.DocGP.Key:
		return "gpdoc"
	}
	return strings.TrimSuffix(key, "doc")
}
