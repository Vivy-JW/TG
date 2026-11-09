package dict

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/dict/db"
	"tt/internal/output"
)

// ACC（应用分类码）不是一张字典表，而是一个由 gzaa/gzac/gzad 描述、
// gzzz 注册为作业、oocq 存值的小型可配置应用。
//
// 本命令只做两件事：
//  1. show：把元模型翻译成 Agent 能读的字段目录；
//  2. query：输出受约束的查询规格，或校验/执行填充后的规格。
//
// 值查询的过滤、选择和分页都在 Go 层完成；查询规格里的字段只能映射到
// 固定的 value/desc/status/parent/refN，永远不拼 SQL 标识符。

func newAccCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "acc",
		Short: "应用分类码作业（ACC）：元模型与动态值查询",
		Long: `应用分类码（ACC）不是 SCC 那种固定下拉字典，而是由配置生成的维护作业：
gzaa/gzaal 定义分类，gzac/gzad 定义参考字段，gzzz/gzzal 注册维护作业，
oocq/oocql 保存按企业隔离的值。

  tt dict acc show  <作业|分类>       看元模型、维护作业、参考字段
  tt dict acc query <作业|分类> --template
                                     输出受约束的查询框架（AI 填 query）
  tt dict acc query <作业|分类> --spec <文件|->
                                     校验并执行填充后的查询规格

show 默认只展示可见参考字段；--full 展示全部 20 个字段。
query --template 不要求企业；query --spec 需要 --at <环境>@<企业>（或 --ent）。
不使用代码扫描推断“谁在用 ACC”；那只能产生候选证据，不是元数据事实。`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	c.AddCommand(newAccShowCmd(), newAccQueryCmd())
	return c
}

func newAccShowCmd() *cobra.Command {
	var lang string
	var full bool
	c := &cobra.Command{
		Use:   "show <作业|分类>",
		Short: "ACC 元模型：分类、维护作业、参考字段",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if IsCSV() {
				return fmt.Errorf("acc show 不支持 CSV；用默认 JSON 或 --human")
			}
			meta, err := queryACCMeta(args[0], lang)
			if err != nil {
				return err
			}
			if meta == nil {
				return notFoundErr("ACC 分类/作业 %s 不存在", args[0]).
					WithHint("确认编号：分类号在 gzaa_t，作业编号在 gzzz_t；可用 tt dict prog <作业> 查登记")
			}
			srcDetail = "summary"
			if full {
				srcDetail = "full"
			}
			data := accShowData(meta, full)
			// --human 只换渲染，不丢信封：来源、计数、notes 都先落进 m。
			m := srcMeta()
			m.TotalRows, m.Returned = 1, 1
			m.Notes = append(m.Notes, "值查询：tt dict acc query "+args[0]+" --template")
			if Format() != output.FormatTable {
				return emitWith(m, data, nil, nil)
			}
			if err := output.WriteMeta(os.Stdout, m); err != nil {
				return err
			}
			printAccShow(meta, full)
			return nil
		},
	}
	c.Flags().StringVar(&lang, "lang", "zh_CN", "名称/标题语言别")
	c.Flags().BoolVar(&full, "full", false, "展示全部 20 个参考字段（默认只展示可见字段）")
	return c
}

func newAccQueryCmd() *cobra.Command {
	var lang string
	var template bool
	var specPath string
	c := &cobra.Command{
		Use:   "query <作业|分类> (--template | --spec <文件|->)",
		Short: "ACC 动态查询：生成或执行受约束的 query spec",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if template == (strings.TrimSpace(specPath) != "") {
				return output.Errorf(output.CodeUsage, 2,
					"需要且只能给 --template 或 --spec（二选一）")
			}
			if IsCSV() {
				return fmt.Errorf("acc query 不支持 CSV；用默认 JSON 或 --human")
			}
			meta, err := queryACCMeta(args[0], lang)
			if err != nil {
				return err
			}
			if meta == nil {
				return notFoundErr("ACC 分类/作业 %s 不存在", args[0]).
					WithHint("确认编号；ACC 分类通常在 gzaa_t，作业在 gzzz_t")
			}
			if template {
				if Format() == output.FormatTable {
					return output.Errorf(output.CodeUsage, 2, "--template 只输出 JSON；去掉 --human")
				}
				srcDetail = ""
				return emitOneNotes(accTemplateData(meta),
					"填 query 后：tt dict acc query "+args[0]+" --spec <文件|->")
			}
			ent, err := accEnt()
			if err != nil {
				return err
			}
			spec, err := readACCSpec(specPath)
			if err != nil {
				return err
			}
			if err := validateACCSpec(meta, spec); err != nil {
				return err
			}
			values, err := GetDB().QueryACCValues(meta.Category, ent, lang, spec.IncludeInactive)
			if err != nil {
				return accDBError(err)
			}
			data, total, returned, truncated, err := runACCQuery(meta, spec, values)
			if err != nil {
				return err
			}
			srcDetail = ""
			notes := []string{}
			if spec.IncludeInactive {
				notes = append(notes, "已包含 oocqstus 非 Y 的停用值")
			}
			if truncated {
				notes = append(notes, fmt.Sprintf("结果已截断：返回 %d / %d；加大 query.limit 或 query.offset 继续", returned, total))
			}
			// 先建信封，再按格式分流。--human 只换渲染，绝不丢计数/截断/notes。
			m := srcMeta()
			m.TotalRows, m.Returned = total, returned
			m.Truncated = truncated
			if truncated {
				m.TruncHint = "加大 query.limit 或 query.offset 继续"
			}
			m.Notes = append(m.Notes, notes...)
			if Format() == output.FormatTable {
				if err := output.WriteMeta(os.Stdout, m); err != nil {
					return err
				}
				printAccQuery(meta, spec, data)
				return nil
			}
			return emitWith(m, data, nil, nil)
		},
	}
	c.Flags().StringVar(&lang, "lang", "zh_CN", "说明/标题语言别")
	c.Flags().BoolVar(&template, "template", false, "只输出可填充的 query 框架")
	c.Flags().StringVar(&specPath, "spec", "", "填充后的 query JSON 文件；- 表示 stdin")
	return c
}

func queryACCMeta(input, lang string) (*db.ACCMeta, error) {
	meta, err := GetDB().QueryACCMeta(input, lang)
	if err != nil {
		if db.IsMissingTable(err) {
			return nil, missingTableErr("应用分类码 (gzaa/gzac/gzad/oocq 等表)")
		}
		return nil, err
	}
	return meta, nil
}

func accDBError(err error) error {
	if db.IsMissingTable(err) {
		if srcLocal {
			return output.Errorf(output.CodeTableMissing, output.ExitMissing,
				"本地库没有 ACC 值表 oocq/oocql；ACC 值是业务数据，不在默认字典同步范围").
				WithHint("用 --at <环境>@<企业> 直查，例如 --at 109tst@2222")
		}
		return missingTableErr("应用分类码值 (oocq/oocql 等表)")
	}
	return err
}

func accEnt() (int, error) {
	s := strings.TrimSpace(dictEnt)
	if s == "" {
		return 0, output.Errorf(output.CodeUsage, 2,
			"ACC 值按企业隔离：需要 --at <环境>@<企业> 或 --ent <企业>").
			WithHint("例：tt dict acc query aapi010 --spec q.json --at 109tst@2222")
	}
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return 0, output.Errorf(output.CodeUsage, 2, "企业编号不是正整数: %q", s)
	}
	return n, nil
}

func accShowData(meta *db.ACCMeta, full bool) map[string]any {
	fields := make([]accFieldView, 0, len(meta.Fields))
	for _, f := range meta.Fields {
		if !full && !f.Visible {
			continue
		}
		fields = append(fields, accFieldViewOf(f))
	}
	return map[string]any{
		"kind":      "acc.show",
		"scope":     map[string]any{"definition": "global", "values": "ent"},
		"acc":       accMetaMap(meta),
		"jobs":      meta.Jobs,
		"fields":    fields,
		"matchedBy": meta.MatchedBy,
	}
}

func accTemplateData(meta *db.ACCMeta) map[string]any {
	fields := make([]accFieldView, 0, len(meta.Fields)+4)
	fields = append(fields, accCoreFields()...)
	for _, f := range meta.Fields {
		fields = append(fields, accFieldViewOf(f))
	}
	spec := accQuerySpec{
		Where: []accCondition{},
		Order: []accOrder{{Field: "value", Dir: "asc"}},
		Limit: 20,
	}
	return map[string]any{
		"kind":   "acc.query.template",
		"acc":    accMetaMap(meta),
		"jobs":   meta.Jobs,
		"fields": fields,
		"query":  spec,
		"hint":   "只填 query；field 必须是 fields[].key 之一，工具会校验并注入 ent/category",
	}
}

func accMetaMap(meta *db.ACCMeta) map[string]any {
	return map[string]any{
		"分类号":  meta.Category,
		"名称":   meta.Name,
		"模块":   meta.Module,
		"系统标准": meta.System,
		"树状":   meta.Tree,
		"显示单身": meta.ShowLines,
		"状态":   meta.Status,
	}
}

type accFieldView struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	Visible    bool     `json:"visible"`
	Source     int      `json:"数据源,omitempty"`
	SourceName string   `json:"数据源名称,omitempty"`
	Target     string   `json:"目标,omitempty"`
	Open       string   `json:"开窗,omitempty"`
	Check      string   `json:"校验,omitempty"`
	Default    string   `json:"默认值,omitempty"`
	Operators  []string `json:"operators"`
}

func accFieldViewOf(f db.ACCField) accFieldView {
	title := f.Title
	if title == "" {
		title = fmt.Sprintf("参考字段%d", f.Seq)
	}
	return accFieldView{
		Key: f.Key, Title: title, Visible: f.Visible,
		Source: f.DataSource, SourceName: accSourceName(f.DataSource),
		Target: f.Target, Open: f.OpenProgram, Check: f.Check, Default: f.Default,
		Operators: []string{"eq", "in", "prefix", "contains", "is_null", "not_null"},
	}
}

func accCoreFields() []accFieldView {
	ops := []string{"eq", "in", "prefix", "contains", "is_null", "not_null"}
	return []accFieldView{
		{Key: "value", Title: "值", Visible: true, Operators: ops},
		{Key: "desc", Title: "说明", Visible: true, Operators: ops},
		{Key: "status", Title: "状态", Visible: true, SourceName: "oocqstus", Operators: []string{"eq", "in"}},
		{Key: "parent", Title: "上层", Visible: false, Operators: []string{"eq", "in", "is_null", "not_null"}},
	}
}

func accSourceName(n int) string {
	switch n {
	case 1:
		return "SCC"
	case 2:
		return "开窗程序"
	case 4:
		return "ACC"
	default:
		return ""
	}
}

type accQuerySpec struct {
	Select          []string       `json:"select"`
	Where           []accCondition `json:"where"`
	Order           []accOrder     `json:"order"`
	Tree            bool           `json:"tree"`
	Root            string         `json:"root,omitempty"`
	IncludeInactive bool           `json:"includeInactive"`
	Limit           int            `json:"limit"`
	Offset          int            `json:"offset"`
}

type accCondition struct {
	Field string `json:"field"`
	Op    string `json:"op"`
	Value any    `json:"value,omitempty"`
}

type accOrder struct {
	Field string `json:"field"`
	Dir   string `json:"dir"`
}

func readACCSpec(path string) (*accQuerySpec, error) {
	var raw []byte
	var err error
	if strings.TrimSpace(path) == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, output.Errorf(output.CodeUsage, 2, "读取 query spec 失败: %v", err)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, output.Errorf(output.CodeUsage, 2, "query spec 为空")
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, output.Errorf(output.CodeUsage, 2, "query spec 不是 JSON: %v", err)
	}
	if q, ok := root["query"]; ok {
		raw = q
	} else if data, ok := root["data"]; ok {
		// 用户很可能原样保存了 `tt dict acc query ... --template` 的完整信封；
		// 真正的模板在 data.query，不应该逼他们手工剥掉 ok/source。
		var d map[string]json.RawMessage
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, output.Errorf(output.CodeUsage, 2, "query spec 的 data 不是对象: %v", err)
		}
		if q, ok := d["query"]; ok {
			raw = q
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var spec accQuerySpec
	if err := dec.Decode(&spec); err != nil {
		return nil, output.Errorf(output.CodeUsage, 2, "query spec 字段不合法: %v", err)
	}
	return &spec, nil
}

func validateACCSpec(meta *db.ACCMeta, spec *accQuerySpec) error {
	if spec.Limit < 0 || spec.Offset < 0 {
		return output.Errorf(output.CodeUsage, 2, "query.limit/offset 不能为负数")
	}
	if spec.Limit == 0 {
		spec.Limit = 20
	}
	if spec.Limit > 500 {
		return output.Errorf(output.CodeUsage, 2, "query.limit 最大 500（当前 %d）", spec.Limit)
	}
	fields := accFieldMap(meta)
	for _, k := range spec.Select {
		if _, ok := fields[k]; !ok {
			return accUnknownField(k)
		}
	}
	for i := range spec.Where {
		w := &spec.Where[i]
		if _, ok := fields[w.Field]; !ok {
			return accUnknownField(w.Field)
		}
		if !allowedACCOp(w.Op) {
			return output.Errorf(output.CodeUsage, 2,
				"query.where[%d] 不支持操作符 %q；可用：%s", i, w.Op, strings.Join(accOps(), ", "))
		}
		if w.Op == "is_null" || w.Op == "not_null" {
			continue
		}
		if w.Value == nil {
			return output.Errorf(output.CodeUsage, 2, "query.where[%d].value 不能为空", i)
		}
	}
	for i, o := range spec.Order {
		if _, ok := fields[o.Field]; !ok {
			return accUnknownField(o.Field)
		}
		if o.Dir != "asc" && o.Dir != "desc" {
			return output.Errorf(output.CodeUsage, 2, "query.order[%d].dir 只能是 asc/desc", i)
		}
	}
	if spec.Tree && !meta.Tree {
		return output.Errorf(output.CodeUsage, 2,
			"分类 %s 不是树状（gzaa003 != Y），不能使用 query.tree", meta.Category)
	}
	if spec.Root != "" {
		if _, ok := fields["value"]; !ok {
			return accUnknownField("value")
		}
	}
	return nil
}

func accUnknownField(k string) error {
	return output.Errorf(output.CodeUsage, 2,
		"query 字段 %q 不在该 ACC 的字段目录；先运行 tt dict acc query <号> --template", k)
}

func allowedACCOp(op string) bool {
	switch op {
	case "eq", "in", "prefix", "contains", "is_null", "not_null":
		return true
	}
	return false
}

func accOps() []string {
	return []string{"eq", "in", "prefix", "contains", "is_null", "not_null"}
}

func accFieldMap(meta *db.ACCMeta) map[string]db.ACCField {
	m := map[string]db.ACCField{
		"value":  {Key: "value", Title: "值"},
		"desc":   {Key: "desc", Title: "说明"},
		"status": {Key: "status", Title: "状态"},
		"parent": {Key: "parent", Title: "上层"},
	}
	for _, f := range meta.Fields {
		m[f.Key] = f
	}
	return m
}

// runACCQuery 在内存里过滤/树化/分页；返回 data、total、returned、truncated。
func runACCQuery(meta *db.ACCMeta, spec *accQuerySpec, values []db.ACCValue) (map[string]any, int, int, bool, error) {
	filtered := make([]db.ACCValue, 0, len(values))
	for _, v := range values {
		ok := true
		for _, w := range spec.Where {
			if !matchACCCondition(v, w) {
				ok = false
				break
			}
		}
		if ok {
			filtered = append(filtered, v)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if len(spec.Order) == 0 {
			return filtered[i].Value < filtered[j].Value
		}
		for _, o := range spec.Order {
			a, _ := accValueField(filtered[i], o.Field)
			b, _ := accValueField(filtered[j], o.Field)
			if a == b {
				continue
			}
			if o.Dir == "desc" {
				return a > b
			}
			return a < b
		}
		return false
	})
	if spec.Root != "" {
		filtered = retainACCSubtree(filtered, spec.Root)
	}
	total := len(filtered)
	if spec.Offset > 0 {
		if spec.Offset >= len(filtered) {
			filtered = nil
		} else {
			filtered = filtered[spec.Offset:]
		}
	}
	truncated := false
	if spec.Limit > 0 && len(filtered) > spec.Limit {
		filtered = filtered[:spec.Limit]
		truncated = true
	}
	keys := accSelectedKeys(meta, spec.Select)
	views := make([]map[string]any, 0, len(filtered))
	for _, v := range filtered {
		views = append(views, accValueView(v, keys))
	}
	var outValues any = views
	if spec.Tree {
		outValues = buildACCTree(views, filtered)
	}
	data := map[string]any{
		"kind":  "acc.query",
		"acc":   accMetaMap(meta),
		"total": total, "returned": len(views), "truncated": truncated,
		"values": outValues,
	}
	return data, total, len(views), truncated, nil
}

func accSelectedKeys(meta *db.ACCMeta, selected []string) []string {
	if len(selected) > 0 {
		out := make([]string, 0, len(selected)+1)
		if !containsString(selected, "value") {
			out = append(out, "value")
		}
		return append(out, selected...)
	}
	keys := []string{"value", "desc", "status"}
	for _, f := range meta.Fields {
		if f.Visible {
			keys = append(keys, f.Key)
		}
	}
	return keys
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

func accValueView(v db.ACCValue, keys []string) map[string]any {
	m := make(map[string]any, len(keys))
	for _, k := range keys {
		if s, ok := accValueField(v, k); ok {
			m[k] = s
		}
	}
	return m
}

func accValueField(v db.ACCValue, key string) (string, bool) {
	switch key {
	case "value":
		return v.Value, true
	case "desc":
		return v.Desc, true
	case "status":
		return v.Status, true
	case "parent":
		return v.Parent, true
	}
	if strings.HasPrefix(key, "ref") {
		n, err := strconv.Atoi(strings.TrimPrefix(key, "ref"))
		if err == nil && n >= 1 && n <= len(v.Refs) {
			return v.Refs[n-1], true
		}
	}
	return "", false
}

func matchACCCondition(v db.ACCValue, w accCondition) bool {
	got, ok := accValueField(v, w.Field)
	if !ok {
		return false
	}
	switch w.Op {
	case "is_null":
		return got == ""
	case "not_null":
		return got != ""
	case "eq":
		return got == fmt.Sprint(w.Value)
	case "prefix":
		return strings.HasPrefix(strings.ToLower(got), strings.ToLower(fmt.Sprint(w.Value)))
	case "contains":
		return strings.Contains(strings.ToLower(got), strings.ToLower(fmt.Sprint(w.Value)))
	case "in":
		for _, one := range accCondValues(w.Value) {
			if got == one {
				return true
			}
		}
		return false
	}
	return false
}

func accCondValues(v any) []string {
	switch x := v.(type) {
	case []any:
		out := make([]string, 0, len(x))
		for _, one := range x {
			out = append(out, fmt.Sprint(one))
		}
		return out
	case []string:
		return x
	case string:
		if strings.Contains(x, ",") {
			return splitNames(x)
		}
		return []string{x}
	default:
		return []string{fmt.Sprint(v)}
	}
}

func retainACCSubtree(values []db.ACCValue, root string) []db.ACCValue {
	byValue := make(map[string]db.ACCValue, len(values))
	for _, v := range values {
		byValue[v.Value] = v
	}
	if _, ok := byValue[root]; !ok {
		return nil
	}
	keep := map[string]bool{}
	var walk func(string)
	walk = func(v string) {
		if keep[v] {
			return
		}
		keep[v] = true
		for _, one := range values {
			if one.Parent == v && one.Value != v {
				walk(one.Value)
			}
		}
	}
	walk(root)
	out := make([]db.ACCValue, 0, len(keep))
	for _, v := range values {
		if keep[v.Value] {
			out = append(out, v)
		}
	}
	return out
}

func buildACCTree(views []map[string]any, values []db.ACCValue) []map[string]any {
	byValue := make(map[string]map[string]any, len(views))
	for i := range values {
		byValue[values[i].Value] = views[i]
	}
	roots := make([]map[string]any, 0)
	for i, v := range values {
		view := views[i]
		parent := v.Parent
		if parent == "" || parent == v.Value || byValue[parent] == nil {
			roots = append(roots, view)
			continue
		}
		pv := byValue[parent]
		children, _ := pv["children"].([]map[string]any)
		pv["children"] = append(children, view)
	}
	return roots
}

func printAccQuery(meta *db.ACCMeta, spec *accQuerySpec, data map[string]any) {
	fmt.Printf("=== ACC %s ===\n", meta.Category)
	if meta.Name != "" {
		fmt.Printf("名称: %s\n", meta.Name)
	}
	if spec.Tree {
		if roots, ok := data["values"].([]map[string]any); ok {
			for _, root := range roots {
				printACCNode(root, 0)
			}
		}
		return
	}
	rows, _ := data["values"].([]map[string]any)
	keys := accSelectedKeys(meta, spec.Select)
	cols := make([]string, 0, len(keys))
	for _, k := range keys {
		cols = append(cols, accTitle(meta, k))
	}
	table := make([][]string, 0, len(rows))
	for _, row := range rows {
		line := make([]string, 0, len(keys))
		for _, k := range keys {
			line = append(line, fmt.Sprint(row[k]))
		}
		table = append(table, line)
	}
	output.PrintTable(cols, table)
}

func printACCNode(n map[string]any, level int) {
	fmt.Printf("%s%s  %s\n", strings.Repeat("  ", level), fmt.Sprint(n["value"]), fmt.Sprint(n["desc"]))
	if children, ok := n["children"].([]map[string]any); ok {
		for _, child := range children {
			printACCNode(child, level+1)
		}
	}
}

func accTitle(meta *db.ACCMeta, key string) string {
	switch key {
	case "value":
		return "值"
	case "desc":
		return "说明"
	case "status":
		return "状态"
	case "parent":
		return "上层"
	}
	for _, f := range meta.Fields {
		if f.Key == key && f.Title != "" {
			return f.Title
		}
	}
	return key
}

func printAccShow(meta *db.ACCMeta, full bool) {
	fmt.Printf("=== ACC %s ===\n", meta.Category)
	fmt.Printf("名称: %s\n模块: %s\n树状: %v\n状态: %s\n", meta.Name, meta.Module, meta.Tree, meta.Status)
	fmt.Printf("\n维护作业 (%d):\n", len(meta.Jobs))
	if len(meta.Jobs) == 0 {
		fmt.Println("  (无维护作业；该分类可能只作为参考字段来源)")
	}
	for _, j := range meta.Jobs {
		fmt.Printf("  %s  %s  %s\n", j.Code, j.Program, j.Name)
	}
	fmt.Printf("\n参考字段:\n")
	rows := make([][]string, 0, len(meta.Fields))
	for _, f := range meta.Fields {
		if !full && !f.Visible {
			continue
		}
		rows = append(rows, []string{strconv.Itoa(f.Seq), f.Key, f.Title,
			yesNo(f.Visible), yesNo(f.AllowBlank), accSourceName(f.DataSource),
			f.Target, f.OpenProgram, f.Check})
	}
	output.PrintTable([]string{"序号", "key", "标题", "显示", "可空白", "数据源", "目标", "开窗", "校验"}, rows)
	fmt.Println()
}

func yesNo(b bool) string {
	if b {
		return "Y"
	}
	return "N"
}

func init() {
	Group.AddCommand(newAccCmd())
}
