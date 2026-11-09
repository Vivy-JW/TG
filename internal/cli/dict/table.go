package dict

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	tableKW     string
	tableLang   string
	tableBrief  bool
	tableWho    bool
	tableCols   string
	tableFields bool
)

var tableCmd = &cobra.Command{
	Use:     "r.t [table_name]",
	Aliases: []string{"rt", "table"},
	Short:   "查询数据表字典 (r.t)",
	Long: `查询一张或多张数据表的字典:这张表在系统里做什么(表说明/所属模块/表类型),
以及字段(中文含义/数据类型/长度/主键/必填)、键值与索引。
读代码、看 SQL、查界面字段含义时用它。
支持逗号分隔多个表名;输出简体中文。
无参数时列出全部表(--kw 按表名/中文表说明搜索,从业务词找表);
指定表名默认显示 summary:只给表级信息;--fields 加字段骨架(字段名/说明/类型/必填),
--full 才给完整字段/键值/索引,--brief 只给表级信息。
多表同查时用 --brief 避免打出上千行字段明细。
--who 反查"哪些程序在用这张表"(数据来自 gzdg_t 程序与应用表格功能分析表,由 T100
自己维护;参考作业 azzq902「程式編號對應表格查詢」),并给出各程序对它的操作类别
(S=查询/I=新增/U=修改/D=删除)——改表前的影响分析用它。
--brief 与 --who 只能二选一。
指定表名时可用字段投影：--col 精确选字段（逗号分隔），--kw 按字段名/说明筛选；
投影只缩短 fields[]，表级信息与每个字段对象仍完整。

JSON 形状（单表响应会在 notes 里提示批量写法）：
  详情（默认）  data = [{表名, 表说明, 模块, 类型, 字段数, 字段:[]}]
  --fields      data = [表级项 + 字段骨架（不含键值/索引）]
  --full        data = [{表名, 表说明, 模块, 类型, 字段[], 键值[], 索引[]}]
  --brief       data = [{表名, 表说明, 模块, 类型, 字段数}]
  --kw/无参数   data = [同上表级项]
  --col         data = [表级项 + 只含指定字段的 字段[]]
  --kw(有表名)  data = [表级项 + 按字段名/说明/备注筛选后的 字段[]]
  多表用逗号：tt dict r.t "imaa_t,imaj_t" --brief
  计数：--brief 的 totalRows 是表数；完整详情的 totalRows 是展平后的字段行数。
  精确表名不存在时返回 NOT_FOUND（退出码 4）。
命令名对齐 T100 原生工具 r.t(旧名 rt/table 仍可用)。`,
	Example: `  tt dict r.t dzea_t
  tt dict r.t "dzea_t,dzeb_t,dzed_t"
  tt dict r.t --kw 应收        # 按中文说明找表(列表模式)
  tt dict r.t --brief "apca_t,apcb_t,glab_t"   # 多表只解语义,不打字段明细
  tt dict r.t glab_t --who                     # 反查:哪些程序在用 glab_t(含操作类别)
  tt dict r.t apca_t --fields                  # 字段骨架
  tt dict r.t apca_t --full                    # 完整字典
  tt dict r.t apca_t --col apca035,apca036     # 只看两个字段
  tt dict r.t apca_t --kw 暂估                 # 按字段名/说明筛字段
  tt dict r.t dzea_t --json
  tt dict r.t dzea_t --env 正式区          # 直查该环境的远程库
  tt dict r.t dzea_t --env local:正式区    # 查该环境的本地库`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			srcDetail = "summary"
			if dictFull {
				srcDetail = "full"
			}
		}
		if len(args) == 0 {
			return runTableList()
		}
		tables := splitNames(args[0])
		if len(tables) == 0 {
			return fmt.Errorf("未指定有效的表名")
		}

		// --who 与 --brief 回答的是两件事，同时给会让人以为 brief 能裁剪 who 的输出。
		if tableWho && tableBrief {
			return fmt.Errorf("--brief 与 --who 只能二选一")
		}
		if dictFull && tableBrief {
			return fmt.Errorf("--full 与 --brief 只能二选一")
		}
		if tableBrief && tableCols != "" {
			return fmt.Errorf("--brief 与 --col 只能二选一：--brief 是表级摘要，--col 是字段投影")
		}
		if tableBrief && tableFields {
			return fmt.Errorf("--brief 与 --fields 只能二选一：--brief 是表级信息，--fields 是字段骨架")
		}
		if tableKW != "" && tableCols != "" {
			return fmt.Errorf("--kw 与 --col 只能二选一：--kw 按字段名/说明筛，--col 按列名精确选")
		}
		// --who:反查"哪些程序在用这些表"(gzdg_t,参考作业 azzq902),与看字典是两回事
		if tableWho {
			return runTableWho(tables)
		}

		var dicts []*db.TableDict
		var missing []string
		for _, t := range tables {
			d, found, err := queryTableDict(t)
			if err != nil {
				return err
			}
			if !found {
				missing = append(missing, t)
				continue
			}
			dicts = append(dicts, d)
		}
		if len(missing) > 0 {
			e := notFoundErr("表不存在: %s", strings.Join(missing, ", "))
			if len(tables) == 1 {
				e = e.WithHint("批量查多张表：tt dict r.t \"a,b,c\" --brief")
			}
			return e
		}

		// 字段投影只改 fields[]；表级“字段数”仍报投影前的总数。
		fieldCounts := make(map[string]int, len(dicts))
		for _, d := range dicts {
			fieldCounts[d.TableName] = len(d.Fields)
		}
		if err := projectTableFields(dicts, tableCols, tableKW); err != nil {
			return err
		}

		// --brief:只要表级信息。多表同查时默认会打每张表的全部字段(7 张表 ≈ 700 行),
		// 只想确认"这几张表分别是什么"时用 --brief,和列表模式同样的表。
		if tableBrief {
			headers := []string{"表名", "表说明", "模块", "类型", "字段数"}
			items := make([]db.TableListItem, 0, len(dicts))
			rows := make([][]string, 0, len(dicts))
			for _, d := range dicts {
				items = append(items, db.TableListItem{
					TableName: d.TableName, TableDesc: d.TableDesc, Module: d.Module,
					TableType: d.TableType, FieldCnt: fieldCounts[d.TableName],
				})
				rows = append(rows, []string{d.TableName, d.TableDesc, d.Module, d.TableType,
					fmt.Sprintf("%d", fieldCounts[d.TableName])})
			}
			var notes []string
			if len(tables) == 1 {
				notes = append(notes, "批量查多张表：tt dict r.t \"a,b,c\" --brief（多表只看表级信息）")
			}
			if Format() != output.FormatTable {
				return emitDetailNotes(items, headers, rows, notes...)
			}
			output.PrintTable(headers, rows)
			for _, n := range notes {
				fmt.Println("提示:" + n)
			}
			return nil
		}

		if !dictFull {
			includeFields := tableFields || tableCols != "" || tableKW != ""
			items := make([]tableSummary, 0, len(dicts))
			for _, d := range dicts {
				items = append(items, tableSummaryOf(d, includeFields, fieldCounts[d.TableName]))
			}
			notes := []string{"字段骨架：加 --fields"}
			if Format() != output.FormatTable {
				return emitDetailNotes(items, tableSummaryColumns, tableSummaryRows(dicts, fieldCounts), notes...)
			}
			for _, item := range items {
				printTableSummary(item)
			}
			for _, n := range notes {
				fmt.Println("提示:" + n)
			}
			return nil
		}

		if Format() != output.FormatTable {
			var notes []string
			if len(tables) == 1 {
				notes = append(notes, "批量查多张表：tt dict r.t \"a,b,c\" --brief（多表只看表级信息）")
			}
			return emitDetailNotes(dicts, tableDetailColumns, tableDetailRows(dicts), notes...)
		}

		for _, d := range dicts {
			printTableDict(d)
		}
		if len(tables) == 1 {
			fmt.Println("提示:批量查多张表：tt dict r.t \"a,b,c\" --brief（多表只看表级信息）")
		}
		return nil
	},
}

// projectTableFields 对详情结果做字段投影。--col 是精确列名、全有或全无；
// --kw 是字段名/说明/备注的模糊筛选，零命中保留表级信息并返回空字段数组。
func projectTableFields(dicts []*db.TableDict, colSpec, kw string) error {
	kw = strings.ToLower(strings.TrimSpace(kw))
	if strings.TrimSpace(colSpec) == "" && kw == "" {
		return nil
	}
	var wanted []string
	if strings.TrimSpace(colSpec) != "" {
		wanted = splitNames(colSpec)
		if len(wanted) == 0 {
			return fmt.Errorf("--col 没有有效列名")
		}
	}
	for _, d := range dicts {
		if len(wanted) > 0 {
			byName := make(map[string]db.TableInfo, len(d.Fields))
			for _, f := range d.Fields {
				byName[strings.ToLower(f.FieldName)] = f
			}
			picked := make([]db.TableInfo, 0, len(wanted))
			var missing []string
			for _, name := range wanted {
				f, ok := byName[strings.ToLower(name)]
				if !ok {
					missing = append(missing, d.TableName+"."+name)
					continue
				}
				picked = append(picked, f)
			}
			if len(missing) > 0 {
				return notFoundErr("字段不存在: %s", strings.Join(missing, ", ")).
					WithHint("看可用字段：tt dict r.t " + d.TableName + " --brief")
			}
			d.Fields = picked
			continue
		}
		picked := make([]db.TableInfo, 0)
		for _, f := range d.Fields {
			if strings.Contains(strings.ToLower(f.FieldName), kw) ||
				strings.Contains(strings.ToLower(f.FieldDesc), kw) ||
				strings.Contains(strings.ToLower(f.Remark), kw) {
				picked = append(picked, f)
			}
		}
		d.Fields = picked
	}
	return nil
}

// runTableList 列表模式:列出/搜索表(3,882 张),按中文表说明找表时用它。
func runTableList() error {
	list, err := GetDB().QueryTableList(tableLang, tableKW)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("表字典 (dzea_t/dzeb_t 等表)")
		}
		return err
	}
	headers := []string{"表名", "表说明", "模块", "类型", "字段数"}
	if len(list) == 0 {
		if Format() != output.FormatTable {
			return emit([]db.TableListItem{}, headers, nil)
		}
		if tableKW != "" {
			fmt.Printf("没有匹配 '%s' 的表。\n提示: 表说明有简繁两种写法(zh_CN 与原始繁体档),用字可能是异体(如 对账/对帐/對帳);换个写法或更短的词再试。\n", tableKW)
		} else {
			fmt.Println(emptyHint("表字典"))
		}
		return nil
	}

	var rows [][]string
	for _, t := range list {
		rows = append(rows, []string{t.TableName, t.TableDesc, t.Module, t.TableType, fmt.Sprintf("%d", t.FieldCnt)})
	}
	if Format() != output.FormatTable {
		return emit(list, headers, rows)
	}
	output.PrintTable(headers, rows)
	fmt.Printf("\n共 %d 张表", len(list))
	if tableKW == "" {
		fmt.Print(";用 --kw <业务词> 按表名/中文说明过滤,如 tt dict r.t --kw 应收")
	}
	fmt.Println()
	return nil
}

// queryTableDict assembles a table's complete dictionary (meta + fields + keys + indexes).
func queryTableDict(name string) (*db.TableDict, bool, error) {
	d := &db.TableDict{TableName: name}

	meta, err := GetDB().QueryTableMeta(name)
	if err != nil {
		return nil, false, err
	}
	if meta != nil {
		d.TableName = meta.TableName
		d.TableDesc = meta.TableDesc
		d.Module = meta.Module
		d.TableType = meta.TableType
	}

	fields, err := GetDB().QueryTable(name)
	if err != nil {
		return nil, false, err
	}
	d.Fields = fields
	found := meta != nil || len(fields) > 0
	if !found {
		return d, false, nil
	}

	// 键值/索引依赖 dzed_t/dzec_t，部分库可能未含这两张表
	keys, err := GetDB().QueryKeys(name)
	if err != nil {
		if db.IsMissingTable(err) {
			keys = nil
		} else {
			return nil, false, err
		}
	}
	d.Keys = keys

	indexes, err := GetDB().QueryIndexes(name)
	if err != nil {
		if db.IsMissingTable(err) {
			indexes = nil
		} else {
			return nil, false, err
		}
	}
	d.Indexes = indexes

	return d, true, nil
}

// printTableDict prints a table's dictionary as plain text sections.
func printTableDict(d *db.TableDict) {
	if d.TableDesc == "" && d.Module == "" && d.TableType == "" && len(d.Fields) == 0 {
		fmt.Printf("未找到表 '%s' 或该表无字段定义。\n\n", d.TableName)
		return
	}
	fmt.Printf("=== %s ===\n", d.TableName)
	if d.TableDesc != "" {
		fmt.Printf("表说明: %s\n", d.TableDesc)
	}
	var metaParts []string
	if d.Module != "" {
		metaParts = append(metaParts, "模块: "+d.Module)
	}
	if d.TableType != "" {
		metaParts = append(metaParts, "类型: "+d.TableType)
	}
	if len(metaParts) > 0 {
		fmt.Printf("%s\n", strings.Join(metaParts, "    "))
	}

	fmt.Printf("\n字段 (%d):\n", len(d.Fields))
	if len(d.Fields) == 0 {
		fmt.Println("  (无)")
	} else {
		headers := []string{"序号", "字段名", "字段说明", "数据类型", "长度", "主键", "必填", "备注"}
		var rows [][]string
		for _, f := range d.Fields {
			rows = append(rows, []string{f.Seq, f.FieldName, f.FieldDesc,
				f.DataType, f.Length, f.IsPK, f.Required, f.Remark})
		}
		output.PrintTable(headers, rows)
	}

	fmt.Printf("\n键值 (%d):\n", len(d.Keys))
	if len(d.Keys) == 0 {
		fmt.Println("  (无)")
	} else {
		headers := []string{"键名", "类型", "键值字段", "外键表", "外键字段"}
		var rows [][]string
		for _, k := range d.Keys {
			rows = append(rows, []string{k.KeyName, keyTypeLabel(k.KeyType), k.KeyField, k.RefTable, k.RefField})
		}
		output.PrintTable(headers, rows)
	}

	fmt.Printf("\n索引 (%d):\n", len(d.Indexes))
	if len(d.Indexes) == 0 {
		fmt.Println("  (无)")
	} else {
		headers := []string{"索引名", "类型", "索引字段"}
		var rows [][]string
		for _, ix := range d.Indexes {
			rows = append(rows, []string{ix.IndexName, ix.IndexType, ix.IndexField})
		}
		output.PrintTable(headers, rows)
	}
	fmt.Println()
}

// printTableCSV flattens the field definitions across tables into CSV rows.
// tableDetailColumns / tableDetailRows 表字典详情的列定义与摊平。
// 列定义只有一处:CSV 表头与 JSON 的键同源于 db.TableDict 这一份 DTO。
var tableDetailColumns = []string{"表名", "序号", "字段名", "字段说明", "数据类型", "长度", "主键", "必填", "备注"}

func tableDetailRows(dicts []*db.TableDict) [][]string {
	var rows [][]string
	for _, d := range dicts {
		for _, f := range d.Fields {
			rows = append(rows, []string{d.TableName, f.Seq, f.FieldName, f.FieldDesc,
				f.DataType, f.Length, f.IsPK, f.Required, f.Remark})
		}
	}
	return rows
}

type tableSummaryField struct {
	FieldName string `json:"字段名"`
	FieldDesc string `json:"字段说明"`
	DataType  string `json:"数据类型"`
	Required  string `json:"必填,omitempty"`
}

// tableSummary 是 r.t 的 summary 视图：表级信息；includeFields 时追加字段骨架。
// 它省略键值/索引/备注，但保留字段名、说明、类型、必填；--full 才回完整字典。
type tableSummary struct {
	TableName string              `json:"表名"`
	TableDesc string              `json:"表说明"`
	Module    string              `json:"模块"`
	TableType string              `json:"类型"`
	FieldCnt  int                 `json:"字段数"`
	Fields    []tableSummaryField `json:"字段"`
}

var tableSummaryColumns = []string{"表名", "表说明", "模块", "类型", "字段数"}

func tableSummaryOf(d *db.TableDict, includeFields bool, fieldCnt int) tableSummary {
	t := tableSummary{
		TableName: d.TableName, TableDesc: d.TableDesc, Module: d.Module,
		TableType: d.TableType, FieldCnt: fieldCnt,
		Fields: make([]tableSummaryField, 0),
	}
	if !includeFields {
		return t
	}
	t.Fields = make([]tableSummaryField, 0, len(d.Fields))
	for _, f := range d.Fields {
		t.Fields = append(t.Fields, tableSummaryField{
			FieldName: f.FieldName, FieldDesc: f.FieldDesc,
			DataType: f.DataType, Required: f.Required,
		})
	}
	return t
}

func tableSummaryRows(dicts []*db.TableDict, fieldCounts map[string]int) [][]string {
	rows := make([][]string, 0, len(dicts))
	for _, d := range dicts {
		rows = append(rows, []string{d.TableName, d.TableDesc, d.Module, d.TableType,
			fmt.Sprintf("%d", fieldCounts[d.TableName])})
	}
	return rows
}

func printTableSummary(s tableSummary) {
	fmt.Printf("=== %s ===\n", s.TableName)
	if s.TableDesc != "" {
		fmt.Printf("表说明: %s\n", s.TableDesc)
	}
	var parts []string
	if s.Module != "" {
		parts = append(parts, "模块: "+s.Module)
	}
	if s.TableType != "" {
		parts = append(parts, "类型: "+s.TableType)
	}
	if len(parts) > 0 {
		fmt.Println(strings.Join(parts, "    "))
	}
	fmt.Printf("\n字段 (%d):\n", s.FieldCnt)
	if len(s.Fields) == 0 {
		if s.FieldCnt > 0 {
			fmt.Println("  (用 --fields 展开字段骨架)")
		} else {
			fmt.Println("  (无)")
		}
		return
	}
	rows := make([][]string, 0, len(s.Fields))
	for _, f := range s.Fields {
		rows = append(rows, []string{f.FieldName, f.FieldDesc, f.DataType, f.Required})
	}
	output.PrintTable([]string{"字段名", "字段说明", "数据类型", "必填"}, rows)
	fmt.Println()
}

// tableWhoSummary 是 r.t --who 的默认视图：聚合影响面，不倾倒程序明细。
type tableWhoSummary struct {
	TableName    string            `json:"表格编号"`
	TableDesc    string            `json:"表说明,omitempty"`
	ProgramCount int               `json:"使用程序数"`
	Ops          map[string]int    `json:"操作统计"`
	Sample       []db.TableProgRow `json:"样例"`
}

func tableWhoSummaryOf(t, desc string, progs []db.TableProgRow) tableWhoSummary {
	ops := map[string]int{"S": 0, "I": 0, "U": 0, "D": 0}
	for _, p := range progs {
		for _, code := range []string{"S", "I", "U", "D"} {
			if strings.Contains(p.Ops, code) {
				ops[code]++
			}
		}
	}
	sample := progs
	if len(sample) > 5 {
		sample = sample[:5]
	}
	return tableWhoSummary{
		TableName: t, TableDesc: desc, ProgramCount: len(progs), Ops: ops, Sample: sample,
	}
}

func printTableWhoSummary(s tableWhoSummary) {
	fmt.Printf("=== %s ===\n", s.TableName)
	if s.TableDesc != "" {
		fmt.Printf("表说明: %s\n", s.TableDesc)
	}
	fmt.Printf("使用它的程序: %d\n", s.ProgramCount)
	fmt.Printf("操作统计: S=%d I=%d U=%d D=%d\n", s.Ops["S"], s.Ops["I"], s.Ops["U"], s.Ops["D"])
	if len(s.Sample) > 0 {
		fmt.Println("\n样例:")
		rows := make([][]string, 0, len(s.Sample))
		for _, p := range s.Sample {
			rows = append(rows, []string{p.Prog, p.ProgName, p.Ops})
		}
		output.PrintTable([]string{"程序编号", "程序名称", "操作"}, rows)
	}
	fmt.Println()
}

// runTableWho 反查"哪些程序在用这些表"(gzdg_t 程序与应用表格功能分析表,由 T100 自己维护;
// 参考作业 azzq902 程式編號對應表格查詢)。改表前的影响分析用它。
func runTableWho(tables []string) error {
	var jsonOut []map[string]any
	var outRows [][]string
	var missing []string
	structured := Format() != output.FormatTable
	summary := !dictFull
	for _, t := range tables {
		meta, err := GetDB().QueryTableMeta(t)
		if err != nil && !db.IsMissingTable(err) {
			return err
		}
		if meta == nil {
			missing = append(missing, t)
			continue
		}
		desc := meta.TableDesc

		progs, err := GetDB().QueryTablePrograms(t, tableLang)
		if err != nil {
			if db.IsMissingTable(err) {
				return missingTableErr("程序与表格 (gzdg_t)")
			}
			return err
		}

		if summary {
			s := tableWhoSummaryOf(t, desc, progs)
			if structured {
				jsonOut = append(jsonOut, map[string]any{
					"表格编号": t, "表说明": desc, "使用程序数": s.ProgramCount,
					"操作统计": s.Ops, "样例": s.Sample,
				})
				outRows = append(outRows, []string{t, desc, fmt.Sprint(s.ProgramCount),
					fmt.Sprint(s.Ops["S"]), fmt.Sprint(s.Ops["I"]),
					fmt.Sprint(s.Ops["U"]), fmt.Sprint(s.Ops["D"])})
			} else {
				printTableWhoSummary(s)
			}
			continue
		}

		shown := progs
		if limit := limitOf(); limit > 0 && len(shown) > limit {
			shown = shown[:limit]
		}
		if structured {
			jsonOut = append(jsonOut, map[string]any{"表格编号": t, "表说明": desc, "使用程序": shown})
			for _, p := range shown {
				outRows = append(outRows, []string{t, desc, p.Prog, p.ProgName, p.Ops})
			}
			continue
		}

		title := t
		if desc != "" {
			title = fmt.Sprintf("%s (%s)", t, desc)
		}
		fmt.Printf("=== %s ===\n", title)
		if len(progs) == 0 {
			fmt.Println("没有程序登记使用这张表(或该表只被动态 SQL 访问)。")
			fmt.Println()
			continue
		}
		fmt.Printf("使用它的程序 (%d):\n", len(progs))
		var rows [][]string
		for _, p := range shown {
			rows = append(rows, []string{p.Prog, p.ProgName, p.Ops})
		}
		output.PrintTable([]string{"程序编号", "程序名称", "操作"}, rows)
		if len(shown) < len(progs) {
			fmt.Printf("\n… 还有 %d 个未显示(共 %d 个);--limit 0 显示全部,或 --json 导出\n",
				len(progs)-len(shown), len(progs))
		}
		fmt.Println("操作: S=SELECT 查询 / I=INSERT 新增 / U=UPDATE 修改 / D=DELETE 删除")
		fmt.Println()
	}
	if len(missing) > 0 {
		return notFoundErr("表不存在: %s", strings.Join(missing, ", "))
	}
	if structured {
		var notes []string
		if summary {
			notes = append(notes, "完整程序清单：加 --full --limit 20（或 --limit 0 看全部）")
			return emitDetailNotes(jsonOut,
				[]string{"表格编号", "表说明", "使用程序数", "S", "I", "U", "D"}, outRows, notes...)
		}
		notes = append(notes, "完整清单已按 --limit 截断；--limit 0 看全部")
		return emitDetailNotes(jsonOut, []string{"表格编号", "表说明", "程序编号", "程序名称", "操作"}, outRows, notes...)
	}
	if len(tables) == 1 {
		if summary {
			fmt.Println("提示:完整程序清单用 --full；批量反查多张表：tt dict r.t \"a,b\" --who")
		} else {
			fmt.Println("提示:批量反查多张表：tt dict r.t \"a,b\" --who")
		}
	}
	return nil
}

// keyTypeLabel maps dzed_t key type codes to Chinese labels.
func keyTypeLabel(code string) string {
	switch code {
	case "P":
		return "主键"
	case "F":
		return "外键"
	case "U":
		return "唯一"
	default:
		return code
	}
}

func init() {
	tableCmd.Flags().StringVar(&tableKW, "kw", "", "无表名时按表名/说明搜索；有表名时按字段名/说明筛选")
	tableCmd.Flags().StringVar(&tableLang, "lang", "zh_CN", "表说明语言别 (dzeal002, 默认 zh_CN)")
	tableCmd.Flags().BoolVar(&tableBrief, "brief", false, "只给表级信息(表名/说明/模块/类型/字段数),不打字段明细")
	tableCmd.Flags().BoolVar(&dictFull, "full", false, "完整字典：字段/键值/索引全部返回")
	tableCmd.Flags().BoolVar(&tableWho, "who", false, "反查哪些程序在用这些表(含操作类别 S/I/U/D)")
	tableCmd.Flags().StringVar(&tableCols, "col", "", "只看指定字段(逗号分隔，如 apca035,apca036)")
	tableCmd.Flags().BoolVar(&tableFields, "fields", false, "字段骨架：字段名/说明/类型/必填，不含键值与索引")
	Group.AddCommand(tableCmd)
}

func splitNames(s string) []string {
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
