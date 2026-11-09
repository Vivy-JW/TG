// Package outline 实现 tt outline：把一个 4GL 文件读成块树。
//
// 它只做本地文件/镜像读取，不拨 SSH，也不调用设计器。解析器在
// internal/dev/fgl（ParseOutline），本包只负责数据源选择、投影和输出。
package outline

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/dev/fgl"
	"tt/internal/output"
)

const schemaVersion = "tt.outline/v1"

var validKinds = []fgl.OutlineKind{
	fgl.KindFunction, fgl.KindMain, fgl.KindReport, fgl.KindDialog,
	fgl.KindInput, fgl.KindDisplay, fgl.KindConstruct, fgl.KindMenu, fgl.KindSub,
}

type options struct {
	dir   string
	kind  []string
	depth string
	flat  bool
	brief bool
}

// Register 把 tt outline 挂到根命令。
func Register(root *cobra.Command) { root.AddCommand(newCmd()) }

func newCmd() *cobra.Command {
	o := &options{depth: "all"}
	c := &cobra.Command{
		Use:   "outline <file.4gl>",
		Short: "4GL 源码大纲：函数、UI 块与行范围",
		Long: `读一个 .4gl 文件，输出块树：FUNCTION / MAIN / REPORT / DIALOG /
INPUT / DISPLAY / CONSTRUCT / MENU / SUB，带 1-based 行范围。

数据源（优先级从高到低）：
  位置参数 <file.4gl>    本地显式文件；存在就直接读
  --dir <目录>           在该环境镜像目录里按文件名查找
  --at <环境>            用该环境配置的镜像目录查找（--env 同义）

默认输出 JSON；--human 出树或表格；--flat 改成带 depth/parent 的数组。
--brief 省掉 selStart/selEnd；--kind 只保留指定块类型（含祖先路径）；
--depth 限制树层级，截断节点会带 childrenTruncated。
CSV 不提供。`,
		Args: exactFileArg,
		RunE: func(_ *cobra.Command, args []string) error {
			return o.run(args[0])
		},
	}
	f := c.Flags()
	f.StringVar(&o.dir, "dir", "", "环境镜像目录（最终目录，内含 4gl/）")
	f.StringVar(&o.depth, "depth", "all", "树最大层级：正整数或 all")
	f.StringSliceVar(&o.kind, "kind", nil, "只保留这些块类型：FUNCTION,MAIN,REPORT,DIALOG,INPUT,DISPLAY,CONSTRUCT,MENU,SUB（可逗号分隔或重复）")
	f.BoolVar(&o.flat, "flat", false, "输出扁平数组（带 depth/parent），而不是树")
	f.BoolVar(&o.brief, "brief", false, "省略 selectionRange（selStart/selEnd）；人类树省略行号")
	return c
}

func exactFileArg(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return usage("需要且只接受一个 .4gl 文件；示例：tt outline aapt320.4gl --at 109tst")
	}
	return nil
}

func (o *options) run(file string) error {
	if common.OutputFormat() == output.FormatCSV {
		return usage("outline 不支持 CSV：用默认 JSON，或加 --human")
	}
	kinds, err := parseKinds(o.kind)
	if err != nil {
		return err
	}
	depth, err := parseDepth(o.depth)
	if err != nil {
		return err
	}
	src, err := resolveSource(sourceOpts{File: file, Dir: o.dir, Env: common.Env})
	if err != nil {
		return err
	}
	b, err := os.ReadFile(src.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return notFound("4GL 文件不存在: %s", src.Path)
		}
		return output.Errorf(output.CodeLocalIO, 1, "读取 4GL 文件失败: %v", err)
	}
	nodes := fgl.ParseOutline(string(b))
	total := countNodes(nodes)
	selected := filterKinds(nodes, kinds)
	tree, truncated := buildTree(selected, 1, depth)
	data := resultData(src, o.flat, o.brief, total, tree, truncated)
	if common.IsJSON() {
		return writeJSON(os.Stdout, envelope{OK: true, Schema: schemaVersion, Data: data})
	}
	if o.flat {
		return writeFlatHuman(os.Stdout, tree, o.brief, len(kinds) > 0)
	}
	return writeTreeHuman(os.Stdout, tree, o.brief, len(kinds) > 0)
}

func parseDepth(s string) (int, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "all" {
		return -1, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, usage("--depth 要是正整数或 all（当前 %q）", s)
	}
	return n, nil
}

// parseKinds 把 --kind 的重复/逗号写法归一成集合。
// 零个 kind 表示不过滤；非法值直接报错，不能悄悄忽略。
func parseKinds(raw []string) (map[fgl.OutlineKind]bool, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	allowed := make(map[string]fgl.OutlineKind, len(validKinds))
	for _, k := range validKinds {
		allowed[string(k)] = k
	}
	out := make(map[fgl.OutlineKind]bool)
	for _, item := range raw {
		for _, part := range strings.Split(item, ",") {
			part = strings.ToUpper(strings.TrimSpace(part))
			if part == "" {
				continue
			}
			k, ok := allowed[part]
			if !ok {
				return nil, output.Errorf(output.CodeEnumInvalid, 2,
					"未知块类型 %q；可用：%s", part, strings.Join(kindNames(), ", "))
			}
			out[k] = true
		}
	}
	return out, nil
}

func kindNames() []string {
	out := make([]string, 0, len(validKinds))
	for _, k := range validKinds {
		out = append(out, string(k))
	}
	return out
}

func countNodes(nodes []fgl.OutlineNode) int {
	n := 0
	for _, node := range nodes {
		n += 1 + countNodes(node.Children)
	}
	return n
}

// filterKinds 只保留命中的节点，以及通向命中节点的祖先路径。
// 这样 --kind DIALOG 还能看见它属于哪个 FUNCTION；不命中节点的子树不输出。
func filterKinds(nodes []fgl.OutlineNode, kinds map[fgl.OutlineKind]bool) []fgl.OutlineNode {
	if len(kinds) == 0 {
		return nodes
	}
	out := make([]fgl.OutlineNode, 0, len(nodes))
	for _, node := range nodes {
		children := filterKinds(node.Children, kinds)
		if kinds[node.Kind] || len(children) > 0 {
			copy := node
			copy.Children = children
			out = append(out, copy)
		}
	}
	return out
}

type envelope struct {
	OK     bool   `json:"ok"`
	Schema string `json:"schema"`
	Data   any    `json:"data"`
}

type treeNode struct {
	Label             string     `json:"label"`
	Kind              string     `json:"kind"`
	Line              int        `json:"line"`
	EndLine           int        `json:"endLine"`
	SelStart          *int       `json:"selStart,omitempty"`
	SelEnd            *int       `json:"selEnd,omitempty"`
	Children          []treeNode `json:"children"`
	ChildrenTruncated bool       `json:"childrenTruncated,omitempty"`
}

type flatNode struct {
	Label    string `json:"label"`
	Kind     string `json:"kind"`
	Line     int    `json:"line"`
	EndLine  int    `json:"endLine"`
	Depth    int    `json:"depth"`
	Parent   *int   `json:"parent"`
	SelStart *int   `json:"selStart,omitempty"`
	SelEnd   *int   `json:"selEnd,omitempty"`
}

type result struct {
	Kind      string         `json:"kind"`
	File      string         `json:"file"`
	Source    map[string]any `json:"source"`
	LineBase  int            `json:"lineBase"`
	Total     int            `json:"total"`
	Returned  int            `json:"returned"`
	Truncated bool           `json:"truncated,omitempty"`
	Nodes     any            `json:"nodes"`
}

// buildTree 同时完成深度限制和节点投影。level 从 1 开始；
// depth < 0 表示 all。被截断的节点必须显式带 childrenTruncated。
func buildTree(nodes []fgl.OutlineNode, level, depth int) ([]treeNode, bool) {
	out := make([]treeNode, 0, len(nodes))
	truncated := false
	for _, node := range nodes {
		tn := treeNode{
			Label: node.Label, Kind: string(node.Kind),
			Line: node.Line, EndLine: node.EndLine,
			Children: make([]treeNode, 0),
		}
		tn.SelStart, tn.SelEnd = selection(node)
		if len(node.Children) > 0 {
			if depth > 0 && level >= depth {
				tn.ChildrenTruncated = true
				truncated = true
			} else {
				var cut bool
				tn.Children, cut = buildTree(node.Children, level+1, depth)
				truncated = truncated || cut
			}
		}
		out = append(out, tn)
	}
	return out, truncated
}

func selection(node fgl.OutlineNode) (*int, *int) {
	a, b := node.SelStart, node.SelEnd
	return &a, &b
}

func resultData(src *sourceInputs, flat, brief bool, total int, nodes []treeNode, truncated bool) result {
	out := result{
		Kind:      "outline.tree",
		File:      src.Path,
		Source:    sourceMap(src),
		LineBase:  1,
		Total:     total,
		Returned:  countTree(nodes),
		Truncated: truncated,
	}
	if flat {
		out.Kind = "outline.flat"
		out.Nodes = flatten(nodes, 0, nil, brief)
		return out
	}
	out.Nodes = treeView(nodes, brief)
	return out
}

func treeView(nodes []treeNode, brief bool) []treeNode {
	out := make([]treeNode, 0, len(nodes))
	for _, n := range nodes {
		if brief {
			n.SelStart, n.SelEnd = nil, nil
		}
		n.Children = treeView(n.Children, brief)
		out = append(out, n)
	}
	return out
}

// flatten 生成稳定的扁平关系：parent 是同一返回数组里的下标；
// 它只用于本次输出，不是跨调用的稳定 ID。
func flatten(nodes []treeNode, depth int, parent *int, brief bool) []flatNode {
	out := make([]flatNode, 0)
	flattenInto(&out, nodes, depth, parent, brief)
	return out
}

// flattenInto 用一个共享的输出切片算全局 parent 下标；每层各建一份切片
// 会让子节点拿到本层从 0 开始的下标，父子关系就错了。
func flattenInto(out *[]flatNode, nodes []treeNode, depth int, parent *int, brief bool) {
	for i := range nodes {
		n := nodes[i]
		var selStart, selEnd *int
		if !brief {
			selStart, selEnd = n.SelStart, n.SelEnd
		}
		idx := len(*out)
		*out = append(*out, flatNode{
			Label: n.Label, Kind: n.Kind, Line: n.Line, EndLine: n.EndLine,
			Depth: depth, Parent: parent, SelStart: selStart, SelEnd: selEnd,
		})
		childParent := idx
		flattenInto(out, n.Children, depth+1, &childParent, brief)
	}
}

func countTree(nodes []treeNode) int {
	n := 0
	for _, node := range nodes {
		n += 1 + countTree(node.Children)
	}
	return n
}

func writeJSON(w io.Writer, v any) error {
	return output.WriteJSONValue(w, v)
}

func writeTreeHuman(w io.Writer, nodes []treeNode, brief, filtered bool) error {
	if len(nodes) == 0 {
		msg := "(没有识别到块)"
		if filtered {
			msg = "(没有匹配的块)"
		}
		_, err := fmt.Fprintln(w, msg)
		return err
	}
	return writeTreeNode(w, nodes, 0, brief)
}

func writeTreeNode(w io.Writer, nodes []treeNode, level int, brief bool) error {
	for _, n := range nodes {
		label := n.Label
		if n.Kind == string(fgl.KindFunction) {
			label = n.Kind + " " + label
		}
		line := ""
		if !brief {
			line = fmt.Sprintf("  [%d-%d]", n.Line, n.EndLine)
		}
		if _, err := fmt.Fprintf(w, "%s%s%s\n", strings.Repeat("  ", level), label, line); err != nil {
			return err
		}
		if err := writeTreeNode(w, n.Children, level+1, brief); err != nil {
			return err
		}
		if n.ChildrenTruncated {
			if _, err := fmt.Fprintf(w, "%s  ... 已截断\n", strings.Repeat("  ", level+1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeFlatHuman(w io.Writer, nodes []treeNode, brief, filtered bool) error {
	rows := make([][]string, 0)
	var walk func([]treeNode, int)
	walk = func(ns []treeNode, depth int) {
		for _, n := range ns {
			row := []string{strconv.Itoa(depth), n.Kind, n.Label}
			if !brief {
				row = append(row, fmt.Sprintf("%d-%d", n.Line, n.EndLine))
			}
			rows = append(rows, row)
			walk(n.Children, depth+1)
		}
	}
	walk(nodes, 0)
	if len(rows) == 0 {
		msg := "(没有识别到块)"
		if filtered {
			msg = "(没有匹配的块)"
		}
		_, err := fmt.Fprintln(w, msg)
		return err
	}
	cols := []string{"深度", "类型", "块"}
	if !brief {
		cols = append(cols, "行")
	}
	return output.WriteTable(w, cols, rows)
}

// sortedHits 只用于错误提示：让同名文件的候选顺序稳定。
func sortedHits(hits []string) []string {
	out := append([]string(nil), hits...)
	sort.Strings(out)
	return out
}
