package dict

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var specLang string

// specRowView is the display form of one field spec row.
type specRowView struct {
	Seq        string `json:"序号"`
	Field      string `json:"字段"`
	FieldName  string `json:"字段名"`
	FieldDesc  string `json:"字段说明"`
	Widget     string `json:"控件"`
	Scc        string `json:"SCC码"`
	SccName    string `json:"SCC名称"`
	Required   string `json:"必填"`
	Width      string `json:"显示宽度"`
	Format     string `json:"格式"`
	CaseConv   string `json:"大小写"`
	DefaultVal string `json:"默认值"`
	MaxSym     string `json:"最大值符号"`
	MaxVal     string `json:"最大值"`
	MinSym     string `json:"最小值符号"`
	MinVal     string `json:"最小值"`
	OpenEdit   string `json:"编辑开窗"`
	OpenQuery  string `json:"查询开窗"`
	ChkVal     string `json:"校验带值"`
	LookupType string `json:"串查型态"`
	LookupProg string `json:"串查程序"`
	RepWidth   string `json:"报表栏宽"`
	RepDigit   string `json:"报表小数位"`
	Env        string `json:"客制识别"`
}

// specTable is the display form of a table's field specs.
type specTable struct {
	TableName string        `json:"表名"`
	TableDesc string        `json:"表说明"`
	Rows      []specRowView `json:"字段规格"`
}

var specCmd = &cobra.Command{
	Use:     "desc <table_name> [field_name]",
	Aliases: []string{"spec"},
	Short:   "查询数据库字段对应的前端规格",
	Long: `查询字段的画面规格:每个字段在画面上用什么控件(输入框/下拉/日期…)、
下拉取哪个系统分类码、显示格式/宽度、必填、默认值、开窗与校验引用等
(画面设计器按它生成画面)。
指定表名列出该表全部字段规格;再指定字段名显示单字段完整规格。`,
	Example: `  tt dict desc oobd_t
  tt dict desc oobd_t oobd002
  tt dict desc oobd_t --json
  tt dict desc oobd_t oobd002 --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 2 {
			return runSpecDetail(args[0], args[1])
		}
		return runSpecList(args[0])
	},
}

func runSpecList(table string) error {
	rows, err := GetDB().QuerySpecRows(table, specLang)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("字段规格 (dzep_t)")
		}
		return err
	}
	tbl, err := querySpecTableMeta(table)
	if err != nil {
		return err
	}
	tbl.Rows = toSpecRowViews(rows)
	if len(tbl.Rows) == 0 {
		fmt.Printf("表 '%s' 无字段规格记录 (dzep_t)。\n", table)
		return nil
	}

	var tableRows [][]string
	for _, r := range tbl.Rows {
		tableRows = append(tableRows, specRowCSV(r))
	}
	if Format() != output.FormatTable {
		return emitDetail(tbl, specCSVHeaders, tableRows)
	}

	fmt.Printf("=== %s ===\n", tbl.TableName)
	if tbl.TableDesc != "" {
		fmt.Printf("表说明: %s\n\n", tbl.TableDesc)
	} else {
		fmt.Println()
	}
	headers := []string{"序号", "字段", "字段名", "控件", "SCC码", "必填", "宽度", "格式", "默认值", "校验带值"}
	var textRows [][]string
	for _, r := range tbl.Rows {
		textRows = append(textRows, []string{r.Seq, r.Field, r.FieldName, r.Widget,
			r.Scc, r.Required, r.Width, r.Format, r.DefaultVal, r.ChkVal})
	}
	output.PrintTable(headers, textRows)
	fmt.Println(specExtNote)
	return nil
}

// specCSVHeaders are the full spec columns for CSV output.
var specCSVHeaders = []string{"序号", "字段", "字段名", "字段说明", "控件", "SCC码", "必填", "显示宽度",
	"格式", "大小写", "默认值", "最大值符号", "最大值", "最小值符号", "最小值", "编辑开窗", "查询开窗",
	"校验带值", "串查型态", "串查程序", "报表栏宽", "报表小数位", "客制识别"}

func specRowCSV(r specRowView) []string {
	return []string{r.Seq, r.Field, r.FieldName, r.FieldDesc, r.Widget, r.Scc, r.Required, r.Width,
		r.Format, r.CaseConv, r.DefaultVal, r.MaxSym, r.MaxVal, r.MinSym, r.MinVal,
		r.OpenEdit, r.OpenQuery, r.ChkVal, r.LookupType, r.LookupProg, r.RepWidth, r.RepDigit, r.Env}
}

func runSpecDetail(table, field string) error {
	rows, err := GetDB().QuerySpecRows(table, specLang)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("字段规格 (dzep_t)")
		}
		return err
	}
	if len(rows) == 0 {
		fmt.Printf("表 '%s' 无字段规格记录 (dzep_t)。\n", table)
		return nil
	}

	var found *specRowView
	for i := range rows {
		if rows[i].Field == field {
			v := toSpecRowViews(rows[i : i+1])[0]
			// SCC 名称: gzca 表已同步时关联查询, 失败静默降级
			if v.Scc != "" {
				if h, err := GetDB().QuerySccHeader(v.Scc, specLang); err == nil && h != nil {
					v.SccName = h.Name
				}
			}
			found = &v
			break
		}
	}
	if found == nil {
		fmt.Printf("字段 '%s' 不在表 '%s' 的规格中。可执行 tt dict desc %s 查看全部字段规格。\n", field, table, table)
		return nil
	}

	tbl, err := querySpecTableMeta(table)
	if err != nil {
		return err
	}

	obj := struct {
		TableName string      `json:"表名"`
		TableDesc string      `json:"表说明"`
		Field     specRowView `json:"字段规格"`
	}{tbl.TableName, tbl.TableDesc, *found}
	if Format() == output.FormatTable {
		printSpecDetail(tbl, found)
		return nil
	}
	return emitOne(obj)
}

// querySpecTableMeta returns the table registry info; a missing registry is
// not an error (the spec rows themselves are the primary content).
func querySpecTableMeta(table string) (*specTable, error) {
	tbl := &specTable{TableName: table}
	if meta, err := GetDB().QueryTableMeta(table); err == nil && meta != nil {
		tbl.TableName = meta.TableName
		tbl.TableDesc = meta.TableDesc
	}
	return tbl, nil
}

// toSpecRowViews converts db rows into display views with interpreted codes.
func toSpecRowViews(rows []db.SpecRow) []specRowView {
	views := make([]specRowView, 0, len(rows))
	for _, r := range rows {
		views = append(views, specRowView{
			Seq:        r.Seq,
			Field:      r.Field,
			FieldName:  r.FieldName,
			FieldDesc:  r.FieldDesc,
			Widget:     specWidgetLabel(r.Widget),
			Scc:        r.Scc,
			Required:   r.Required,
			Width:      r.Width,
			Format:     r.Format,
			CaseConv:   r.CaseConv,
			DefaultVal: r.DefaultVal,
			MaxSym:     r.MaxSym,
			MaxVal:     r.MaxVal,
			MinSym:     r.MinSym,
			MinVal:     r.MinVal,
			OpenEdit:   r.OpenEdit,
			OpenQuery:  r.OpenQuery,
			ChkVal:     r.ChkVal,
			LookupType: r.LookupType,
			LookupProg: r.LookupProg,
			RepWidth:   r.RepWidth,
			RepDigit:   r.RepDigit,
			Env:        custLabel(r.Env),
		})
	}
	return views
}

func printSpecDetail(tbl *specTable, v *specRowView) {
	fmt.Printf("=== %s", tbl.TableName)
	if tbl.TableDesc != "" {
		fmt.Printf(" (%s)", tbl.TableDesc)
	}
	fmt.Println(" ===")
	fmt.Printf("字段:     %s", v.Field)
	if v.FieldName != "" {
		fmt.Printf(" (%s)", v.FieldName)
	}
	fmt.Println()
	if v.FieldDesc != "" {
		fmt.Printf("字段说明: %s\n", v.FieldDesc)
	}
	fmt.Printf("控件:     %s\n", v.Widget)
	if v.Scc != "" {
		line := "SCC码:    " + v.Scc
		if v.SccName != "" {
			line += " (" + v.SccName + ")"
		}
		fmt.Println(line)
	}
	fmt.Printf("必填:     %s\n", v.Required)
	fmt.Printf("显示宽度: %s\n", v.Width)
	fmt.Printf("格式:     %s\n", v.Format)
	fmt.Printf("大小写:   %s\n", v.CaseConv)
	fmt.Printf("默认值:   %s\n", v.DefaultVal)
	if rng := rangeSpec(v); rng != "" {
		fmt.Printf("取值范围: %s\n", rng)
	}
	if v.OpenEdit != "" || v.OpenQuery != "" {
		fmt.Printf("开窗:     编辑 %s / 查询 %s\n", v.OpenEdit, v.OpenQuery)
	}
	if v.ChkVal != "" {
		fmt.Printf("校验带值: %s\n", v.ChkVal)
	}
	if v.LookupType != "" || v.LookupProg != "" {
		fmt.Printf("串查:     型态 %s / 程序 %s\n", v.LookupType, v.LookupProg)
	}
	if v.RepWidth != "" || v.RepDigit != "" {
		fmt.Printf("报表:     栏宽 %s / 小数位 %s\n", v.RepWidth, v.RepDigit)
	}
	if v.Env != "" {
		fmt.Printf("客制识别: %s\n", v.Env)
	}
	fmt.Println(specExtNote)
}

// rangeSpec combines comparison symbols with min/max values.
func rangeSpec(v *specRowView) string {
	var parts []string
	if v.MinSym != "" || v.MinVal != "" {
		parts = append(parts, "最小值 "+strings.TrimSpace(v.MinSym+" "+v.MinVal))
	}
	if v.MaxSym != "" || v.MaxVal != "" {
		parts = append(parts, "最大值 "+strings.TrimSpace(v.MaxSym+" "+v.MaxVal))
	}
	return strings.Join(parts, " / ")
}

// specWidgetLabel maps dzep010 widget codes (dzej_t GENERO_WIDGETS) to names.
func specWidgetLabel(code string) string {
	labels := map[string]string{
		"01": "ButtonEdit", "02": "CheckBox", "03": "ComboBox", "04": "DateEdit",
		"05": "Edit", "06": "FFImage", "07": "FFLabel", "08": "ProgressBar",
		"09": "RadioGroup", "10": "Slider", "11": "SpinEdit", "12": "TextEdit",
		"13": "TimeEdit", "14": "Field", "15": "WebComponent", "34": "DateTimeEdit",
	}
	label, ok := labels[code]
	if !ok {
		return code
	}
	return fmt.Sprintf("%s (%s)", label, code)
}

const specExtNote = `
注: dzep011 (SCC码) 为下拉/单选控件的选项数据源, 仅控件为 ComboBox (03)/RadioGroup (09) 时有效;
    各列含义可对照维护作业 adzi150 (字段规格) 与画面设计器 adzp168。`

func init() {
	specCmd.Flags().StringVar(&specLang, "lang", "zh_CN", "说明语言别 (dzebl002, 默认 zh_CN)")
	Group.AddCommand(specCmd)
}
