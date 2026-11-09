// Package interfacecmd 实现 tt interface：把 .42f 的页面/字段结构与 .42s 的
// 中文标签合成可读的前端画面模型。
//
// 本组只读本地镜像/文件，不拨 SSH，也不写服务器。数据默认 JSON；--human
// 才输出树/表格。页面选择器只接受 42f 的 XML name（id），不接受 page_1
// 这类 textKey —— 样例里归一化后的 page_1 可能命中两个页面。
package interfacecmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/form42f"
	"tt/internal/output"
	"tt/internal/screen"
	"tt/internal/str42s"
)

type flags struct {
	module string
	dir    string
	fileF  string
	fileS  string

	brief      bool
	depth      string
	withFields bool

	page  string
	col   string
	kw    string
	limit int
	full  bool
}

var f flags

// Register 把 tt interface 挂到根命令。
func Register(root *cobra.Command) {
	g := &cobra.Command{
		Use:   "interface",
		Short: "合成 .42f + .42s：看前端页面、字段与中文标签",
		Long: `把编译后的表单（.42f）和编译字符串（.42s）合成一张可读的前端画面模型：
页面树、页面里的字段、字段中文标签、控件/类型/必填。只读本地文件，不拨 SSH。

  tt interface show  <form>     页面结构（默认不展开字段）
  tt interface pages <form>     所有页面的扁平索引
  tt interface fields <form>    字段扁平表，可按页/列名/关键词过滤（默认每页最多 30 个字段）

数据源（优先级从上到下）：
  --42f <文件> [--42s <文件>]   直接读一对文件
  --dir <环境镜像目录>          读该环境镜像（内含 erp/、com/）
  --at <环境>                   从 config 取该环境的镜像目录

页面选择用 --page <id>（42f XML name），不是 textKey；样例里 page_1 可能
是另一个页面的 id。--42s 缺失仍能出结构，但标签会标为 unresolved。

数据默认 JSON；--human 出页面树/表格（CSV 暂不支持）。`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return common.UnknownSubcommand(cmd, args)
		},
	}
	pf := g.PersistentFlags()
	pf.StringVar(&f.module, "module", "", "模块名（同名表单/42s 多个时用于消歧）")
	pf.StringVar(&f.dir, "dir", "", "环境镜像目录（最终目录，内含 erp/、com/）")
	pf.StringVar(&f.fileF, "42f", "", "直接指定 .42f 文件")
	pf.StringVar(&f.fileS, "42s", "", "直接指定 .42s 文件（可缺省）")
	pf.BoolVar(&f.brief, "brief", false, "精简字段：省略路径、注释、几何等非核心信息")

	g.AddCommand(
		newShowCmd(),
		newPagesCmd(),
		newFieldsCmd(),
		newLabelsCmd(),
	)
	root.AddCommand(g)
}

func newShowCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "show <form>",
		Short: "页面结构（默认不展开字段）",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := noCSV(); err != nil {
				return err
			}
			depth, err := parseDepth(f.depth)
			if err != nil {
				return err
			}
			v, src, err := load(args)
			if err != nil {
				return err
			}
			out := showData(v, src, depth, f.withFields, f.brief)
			if common.IsJSON() {
				return emitJSON(out)
			}
			return printShowHuman(v, depth, f.withFields)
		},
	}
	c.Flags().StringVar(&f.depth, "depth", "2", "页面树展开深度：正整数或 all")
	c.Flags().BoolVar(&f.withFields, "fields", false, "在页面树里展开字段")
	return c
}

func newPagesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "pages <form>",
		Short: "所有页面的扁平索引",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := noCSV(); err != nil {
				return err
			}
			v, src, err := load(args)
			if err != nil {
				return err
			}
			out := pagesData(v, src, f.brief)
			if !common.IsJSON() {
				return printPagesHuman(v, f.brief)
			}
			return emitJSON(out)
		},
	}
	return c
}

func newFieldsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "fields <form>",
		Short: "字段扁平表，可按页/列名/关键词过滤",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := noCSV(); err != nil {
				return err
			}
			if f.full && cmd.Flags().Changed("limit") {
				return usage("--full 与 --limit 不能同时给：--full 表示取全部字段，--limit 表示截断")
			}
			if f.col != "" && f.kw != "" {
				return usage("--col 与 --kw 不能同时给：--col 是精确列名，--kw 是模糊搜索")
			}
			if f.limit < 0 {
				return usage("--limit 不能为负数（0 表示不截断）")
			}
			v, src, err := load(args)
			if err != nil {
				return err
			}
			selected := v.Fields
			var page *screen.PageView
			if f.page != "" {
				if f.page == "_global" {
					selected = filterPage(selected, "")
				} else {
					page = v.FindPage(f.page)
					if page == nil {
						return fail("PAGE_NOT_FOUND", 4, fmt.Sprintf("页面 %q 不存在（--page 只接受 id，不接受 textKey）", f.page), fmt.Sprintf("看可用页面：tt interface pages %s", v.Form))
					}
					selected = filterPage(selected, page.ID)
				}
			}
			if f.col != "" {
				selected = filterCol(selected, f.col)
				if len(selected) == 0 {
					return fail("FIELD_NOT_FOUND", 4, fmt.Sprintf("列 %q 不在当前范围", f.col), fmt.Sprintf("模糊搜索：tt interface fields %s --kw %s", v.Form, f.col))
				}
			}
			if f.kw != "" {
				selected = filterKW(selected, f.kw)
			}
			limit := f.limit
			if f.full {
				limit = 0
			}
			total := len(selected)
			truncated := false
			if limit > 0 && len(selected) > limit {
				selected = selected[:limit]
				truncated = true
			}
			out := fieldsData(v, src, selected, page, total, truncated, f.brief)
			if f.full {
				out["detail"] = "full"
			} else {
				out["detail"] = "summary"
			}
			if truncated {
				out["hint"] = "默认最多 30 个字段；看全量加 --full 或 --limit 0"
			}
			if !common.IsJSON() {
				return printFieldsHuman(v, selected, f.brief)
			}
			return emitJSON(out)
		},
	}
	c.Flags().StringVar(&f.page, "page", "", "只取某页字段（页面 id；_global 表示未归页字段）")
	c.Flags().StringVar(&f.col, "col", "", "精确列名（colName）；零命中 NOT_FOUND")
	c.Flags().StringVar(&f.kw, "kw", "", "模糊搜索 label/colName/textKey；零命中仍成功")
	c.Flags().IntVar(&f.limit, "limit", 30, "最多返回字段数；0 表示不截断（默认 30）")
	c.Flags().BoolVar(&f.full, "full", false, "完整视图：等同 --limit 0")
	return c
}

func newLabelsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:    "labels <form>",
		Short:  "诊断：列出 42s 的 token → 中文（隐藏命令）",
		Hidden: true,
		Args:   cobra.RangeArgs(0, 1),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := noCSV(); err != nil {
				return err
			}
			v, src, err := load(args)
			if err != nil {
				return err
			}
			var entries []map[string]any
			if src.Form42S != "" {
				b, err := os.ReadFile(src.Form42S)
				if err != nil {
					return fail("LABELS_INVALID", 1, fmt.Sprintf("读 42s 失败: %v", err))
				}
				tbl, err := str42s.Parse(b)
				if err != nil {
					return fail("LABELS_INVALID", 1, err.Error())
				}
				for _, e := range tbl.Entries {
					entries = append(entries, map[string]any{"token": e.Token, "text": e.Text})
				}
			}
			out := map[string]any{
				"kind": "interface.labels", "form": v.Form, "module": v.Module,
				"source": sourceMap(src), "total": len(entries), "entries": entries,
			}
			if !common.IsJSON() {
				return printLabelsHuman(entries)
			}
			return emitJSON(out)
		},
	}
	return c
}

func load(args []string) (*screen.View, *sourceInputs, error) {
	form := ""
	if len(args) == 1 {
		form = strings.TrimSpace(args[0])
	}
	src, err := resolveSources(sourceOpts{
		Form: form, Module: f.module, Dir: f.dir, FileF: f.fileF, FileS: f.fileS,
	})
	if err != nil {
		return nil, nil, err
	}
	data, err := os.ReadFile(src.Form42F)
	if err != nil {
		return nil, nil, fail("FORM_UNKNOWN", 1, fmt.Sprintf("读 42f 失败: %v", err))
	}
	layout, err := form42f.ParseLayout(src.Form, string(data))
	if err != nil {
		return nil, nil, fail("FORM_UNKNOWN", 1, err.Error())
	}
	var labels *str42s.Table
	if src.Form42S != "" {
		b, err := os.ReadFile(src.Form42S)
		if err != nil {
			return nil, nil, fail("LABELS_INVALID", 1, fmt.Sprintf("读 42s 失败: %v", err))
		}
		labels, err = str42s.Parse(b)
		if err != nil {
			return nil, nil, fail("LABELS_INVALID", 1, err.Error())
		}
	}
	v := screen.Build(src.Form, src.Module, layout, labels)
	return v, src, nil
}

// fail 构造带稳定码和修复建议的错误；根命令的 reportError 会认 *output.Error。
func fail(code string, exit int, msg string, hint ...string) *output.Error {
	e := output.Errorf(code, exit, "%s", msg)
	if len(hint) > 0 {
		e = e.WithHint(hint[0])
	}
	return e
}

func parseDepth(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "all" {
		return -1, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, usage("--depth 要是正整数或 all（当前 %q）", s)
	}
	return n, nil
}

func filterPage(in []screen.FieldView, pageID string) []screen.FieldView {
	out := make([]screen.FieldView, 0, len(in))
	for _, f := range in {
		if f.PageID == pageID {
			out = append(out, f)
		}
	}
	return out
}

func filterCol(in []screen.FieldView, col string) []screen.FieldView {
	out := make([]screen.FieldView, 0)
	for _, f := range in {
		if strings.EqualFold(f.ColName, col) {
			out = append(out, f)
		}
	}
	return out
}

func filterKW(in []screen.FieldView, kw string) []screen.FieldView {
	kw = strings.ToLower(strings.TrimSpace(kw))
	if kw == "" {
		return in
	}
	out := make([]screen.FieldView, 0)
	for _, f := range in {
		if strings.Contains(strings.ToLower(f.Label), kw) ||
			strings.Contains(strings.ToLower(f.ColName), kw) ||
			strings.Contains(strings.ToLower(f.TextKey), kw) {
			out = append(out, f)
		}
	}
	return out
}

// noCSV：本组只有 JSON 与 --human 两种稳定视图；CSV 的扁平列还没有契约，
// 明确拒绝比悄悄退回表格更安全。
func noCSV() error {
	if common.OutputFormat() == output.FormatCSV {
		return usage("interface 暂不支持 CSV：用默认 JSON，或加 --human 出树/表格")
	}
	return nil
}
