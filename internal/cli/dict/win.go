package dict

import (
	"fmt"
	"strconv"
	"strings"

	"tt/internal/dict/db"
	dictlib "tt/internal/dict"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	winKW   string
	winLang string
)

// winListItem is the display form of one window in list mode.
type winListItem struct {
	ID       string `json:"开窗识别码"`
	Cust     string `json:"客制"`
	Desc     string `json:"说明"`
	Status   string `json:"状态"`
	PageSize string `json:"每页笔数"`
	HardCode string `json:"HardCode"`
	Industry string `json:"行业别"`
}

// winHeaderView is the display form of one window header variant (标准/客制).
type winHeaderView struct {
	Cust     string         `json:"客制"`
	Status   string         `json:"状态"`
	SQL      string         `json:"SQL指令"`
	PageSize string         `json:"每页笔数"`
	Serial   string         `json:"作业串查编号"`
	HardCode string         `json:"HardCode"`
	Industry string         `json:"行业别"`
	Desc     string         `json:"说明"`
	Memo     string         `json:"助记码"`
	Cols     []winColView   `json:"显现设定"`
}

type winColView struct {
	Seq      string `json:"显现顺序"`
	Field    string `json:"字段编号"`
	Alias    string `json:"表格别名"`
	Widget   string `json:"显示控件"`
	IsRet    string `json:"是否回传"`
	CaseConv string `json:"大小写"`
	Format   string `json:"显示格式"`
	Label    string `json:"标签缀字"`
}

// winDetail is the display form of a complete window definition.
type winDetail struct {
	ID      string          `json:"开窗识别码"`
	Headers []winHeaderView `json:"开窗定义"`
}

var winCmd = &cobra.Command{
	Use:     "r.q [开窗码]",
	Aliases: []string{"rq", "win"},
	Short:   "查询可复用开窗 (r.q)",
	Long: `查询可复用开窗 (r.q):代码里 CALL q_xxx() 弹出的查寻选单定义——带占位符的
选取 SQL(<field>/<table>/<wc> 标记)、外部参数(arg1~9)与显现/回传列。
无参数时列出全部开窗(--kw 按说明过滤);指定开窗码显示完整详情:
SQL 指令(附标签图例)、参数、显现设定。
开窗码形如 q_apca001。命令名对齐 T100 原生工具 r.q(旧名 rq/win 仍可用)。`,
	Example: `  tg dict r.q
  tg dict r.q --kw 料号
  tg dict r.q q_apca001
  tg dict r.q q_apca001 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return runWinDetail(args[0])
		}
		return runWinList()
	},
}

func runWinList() error {
	rows, err := GetDB().QueryWinList(winLang, winKW)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("开窗定义 (dzca_t 等表)")
		}
		return err
	}
	if srcProduct == "gp" {
		return printWinListGP(rows)
	}
	items := mergeWinList(rows)
	if len(items) == 0 {
		fmt.Println(emptyHint("开窗定义"))
		return nil
	}

	headers := []string{"开窗识别码", "客制", "说明", "状态", "每页笔数", "HardCode", "行业别"}
	var tableRows [][]string
	for _, it := range items {
		tableRows = append(tableRows, []string{it.ID, it.Cust, it.Desc, it.Status,
			it.PageSize, it.HardCode, it.Industry})
	}
	if Format() != output.FormatTable {
		return emit(items, headers, tableRows)
	}
	output.PrintTable(headers, tableRows)
	return nil
}

// mergeWinList merges the 标准/客制 variants of the same dzca001 into one row.
func mergeWinList(rows []db.WinListRow) []winListItem {
	var items []winListItem
	for _, r := range rows {
		label := custLabel(r.Cust)
		if len(items) > 0 && items[len(items)-1].ID == r.ID {
			it := &items[len(items)-1]
			if !strings.Contains(it.Cust, label) {
				it.Cust += "," + label
			}
			continue
		}
		items = append(items, winListItem{
			ID:       r.ID,
			Cust:     custLabel(r.Cust),
			Desc:     r.Desc,
			Status:   r.Status,
			PageSize: r.PageSize,
			HardCode: winHardCodeLabel(r.HardCode),
			Industry: r.Industry,
		})
	}
	return items
}

func runWinDetail(id string) error {
	if IsCSV() {
		return fmt.Errorf("详情模式不支持 --csv，请使用列表模式或 --json")
	}

	headers, err := GetDB().QueryWinHeaders(id, winLang)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("开窗定义 (dzca_t 等表)")
		}
		return err
	}
	if len(headers) == 0 {
		return notFoundErr("开窗定义 %s 不存在", id).WithHint("列出全部开窗定义：tg dict r.q")
	}
	if srcProduct == "gp" {
		return runWinDetailGP(id, headers)
	}

	cols, err := GetDB().QueryWinCols(id, winLang)
	if err != nil && !db.IsMissingTable(err) {
		return err
	}

	detail := winDetail{ID: id}
	for _, h := range headers {
		view := winHeaderView{
			Cust:     custLabel(h.Cust),
			Status:   h.Status,
			SQL:      h.SQL,
			PageSize: h.PageSize,
			Serial:   h.Serial,
			HardCode: winHardCodeLabel(h.HardCode),
			Industry: h.Industry,
			Desc:     h.Desc,
			Memo:     h.Memo,
		}
		for _, c := range cols {
			if c.Cust != h.Cust {
				continue
			}
			view.Cols = append(view.Cols, winColView{
				Seq:      c.Seq,
				Field:    c.Field,
				Alias:    c.Alias,
				Widget:   specWidgetLabel(c.Widget),
				IsRet:    c.IsRet,
				CaseConv: c.CaseConv,
				Format:   c.Format,
				Label:    c.Label,
			})
		}
		detail.Headers = append(detail.Headers, view)
	}

	if Format() == output.FormatTable {
		printWinDetail(&detail)
		return nil
	}
	return emitOne(detail)
}

func printWinDetail(d *winDetail) {
	fmt.Printf("=== %s ===\n", d.ID)
	for _, h := range d.Headers {
		fmt.Printf("\n[%s]\n", h.Cust)
		fmt.Printf("状态:     %s\n", h.Status)
		fmt.Printf("每页笔数: %s\n", h.PageSize)
		fmt.Printf("串查编号: %s\n", h.Serial)
		fmt.Printf("HardCode: %s\n", h.HardCode)
		fmt.Printf("行业别:   %s\n", h.Industry)
		fmt.Printf("说明:     %s\n", h.Desc)
		fmt.Printf("助记码:   %s\n", h.Memo)

		fmt.Println("\nSQL 指令 (dzca003):")
		if h.SQL == "" {
			fmt.Println("  (无)")
		} else {
			for _, line := range strings.Split(h.SQL, "\n") {
				fmt.Println("  " + line)
			}
		}

		if len(h.Cols) > 0 {
			fmt.Printf("\n显现设定 (dzcc_t, %d):\n", len(h.Cols))
			headers := []string{"显现顺序", "字段编号", "表格别名", "显示控件", "是否回传", "大小写", "显示格式", "标签缀字"}
			var rows [][]string
			for _, c := range h.Cols {
				rows = append(rows, []string{c.Seq, c.Field, c.Alias, c.Widget, c.IsRet,
					c.CaseConv, c.Format, c.Label})
			}
			output.PrintTable(headers, rows)
		}
	}
	fmt.Println(winTagLegend)
}

// winHardCodeLabel maps dzca006 codes to Chinese labels.
func winHardCodeLabel(code string) string {
	switch code {
	case "Y":
		return "Y (跳过自动产生)"
	case "N":
		return "N"
	default:
		return code
	}
}

const winTagLegend = `
标签说明 (开窗 SQL 中的占位符):
  <field>...</field>   选取/显示字段区 (生成时按 dzcc_t 输出字段列表)
  <table>...</table>   查询的表区
  <wc>...</wc>         外部 WHERE 条件的插入点 (g_qryparam.where)
  <inwc>...</inwc>     input 段条件 (state='i'/'m' 时并入查询)
  arg1~arg9            外部参数, 按 dzcb_t 定义代入 (日期参数自动转 TO_DATE)
  :DEPT :SITE :USER :LANG :DLANG :ENT :LEGAL   全局变量
  :TODAY               当前日期

回传约定: dzcc005='Y' 的字段按显现顺序构成 g_qryparam.return1~return9 (多选时 '|' 分隔)。`

// ---- TOP GP：gab_file + gac_file 的开窗定义 ----
//
// GP 没有 T100 那种"整段 SQL 模板"(dzca003)：SQL 是运行期由 cl_create_qry() 按
// gab02/gab05 的条件片段 + gac 的栏位表清单现拼的。这里按同一套规则拼出来给人看
// （规则见 dictlib.QryDef.BuildSQL），栏位清单则直接来自 gac_file。

// gpWinListItem 是 GP 开窗列表的一行。
type gpWinListItem struct {
	Code     string `json:"开窗代号"`
	Title    string `json:"标题"`
	ColCount string `json:"栏位数"`
	Cust     string `json:"客制"`
	HardCode string `json:"HardCode"`
}

// gpWinColItem 是 GP 开窗里的一个显现栏位。
type gpWinColItem struct {
	Seq        string `json:"顺序"`
	Table      string `json:"表格"`
	Field      string `json:"字段"`
	Label      string `json:"显示名"`
	Width      string `json:"宽度"`
	Return     string `json:"回传"`
	Widget     string `json:"控件"`
	Outer      string `json:"OUTER"`
	Constraint string `json:"输入约束"`
	ConnectTo  string `json:"串查"`
	DataType   string `json:"列类型"`
}

// gpWinDef 是 GP 开窗的一个版本（标准 / 客制）。
type gpWinDef struct {
	Cust        string         `json:"客制"`
	Title       string         `json:"标题"`
	HardCode    string         `json:"HardCode"`
	Unlocked    string         `json:"单身可改"`
	Distinct    string         `json:"DISTINCT"`
	JoinTables  string         `json:"JOIN 表"`
	InputWhere  string         `json:"Input WHERE"`
	ConstrWhere string         `json:"Construct WHERE"`
	Memo        string         `json:"备注"`
	SQL         string         `json:"SQL"`
	Cols        []gpWinColItem `json:"开窗栏位"`
}

// gpWinDetail 是 GP 开窗的完整定义。
type gpWinDetail struct {
	Code string     `json:"开窗代号"`
	Defs []gpWinDef `json:"开窗定义"`
}

const gpWinNoSQL = "（这条开窗是 HardCode 程序：SQL 由程序自己写，不由 gac_file 拼）"

func printWinListGP(rows []db.WinListRow) error {
	items := mergeWinListGP(rows)
	if len(items) == 0 {
		fmt.Println(emptyHint("开窗定义"))
		return nil
	}
	headers := []string{"开窗代号", "标题", "栏位数", "客制", "HardCode"}
	var tableRows [][]string
	for _, it := range items {
		tableRows = append(tableRows, []string{it.Code, it.Title, it.ColCount, it.Cust, it.HardCode})
	}
	if Format() != output.FormatTable {
		return emit(items, headers, tableRows)
	}
	output.PrintTable(headers, tableRows)
	return nil
}

// mergeWinListGP 把同一开窗的标准/客制两行并成一行。
func mergeWinListGP(rows []db.WinListRow) []gpWinListItem {
	var items []gpWinListItem
	for _, r := range rows {
		label := custLabel(r.Cust)
		if len(items) > 0 && strings.EqualFold(items[len(items)-1].Code, r.ID) {
			it := &items[len(items)-1]
			if !strings.Contains(it.Cust, label) {
				it.Cust += "," + label
			}
			continue
		}
		items = append(items, gpWinListItem{
			Code:     r.ID,
			Title:    r.Desc,
			ColCount: r.ColCount,
			Cust:     label,
			HardCode: winHardCodeLabel(r.HardCode),
		})
	}
	return items
}

func runWinDetailGP(id string, headers []db.WinHeaderRow) error {
	if IsCSV() {
		return fmt.Errorf("详情模式不支持 --csv，请使用列表模式或 --json")
	}
	cols, err := GetDB().QueryWinCols(id, winLang)
	if err != nil && !db.IsMissingTable(err) {
		return err
	}

	detail := gpWinDetail{Code: id}
	for _, h := range headers {
		def := gpWinDef{
			Cust:        custLabel(h.Cust),
			Title:       h.Desc,
			HardCode:    winHardCodeLabel(h.HardCode),
			Unlocked:    yesNoLabel(h.Unlocked, "可改", "不可改"),
			Distinct:    yesNoLabel(h.Distinct, "是", "否"),
			JoinTables:  h.JoinTables,
			InputWhere:  h.InputWhere,
			ConstrWhere: h.ConstrWhere,
			Memo:        h.Memo,
		}
		defSQL := dictlib.QryDef{
			Code:        h.ID,
			Title:       h.Desc,
			InputWhere:  h.InputWhere,
			JoinTables:  h.JoinTables,
			Distinct:    strings.EqualFold(h.Distinct, "Y"),
			ConstrWhere: h.ConstrWhere,
			Memo:        h.Memo,
			HardCode:    strings.EqualFold(h.HardCode, "Y"),
			Cust:        strings.TrimSpace(h.Cust) != "",
		}
		for _, c := range cols {
			if c.Cust != h.Cust {
				continue
			}
			def.Cols = append(def.Cols, gpWinColItem{
				Seq:        c.Seq,
				Table:      c.Table,
				Field:      c.Field,
				Label:      c.Label,
				Width:      c.Width,
				Return:     yesNoLabel(c.Return, "Y", "N"),
				Widget:     gpWidgetLabel(c.Type),
				Outer:      yesNoLabel(c.Outer, "Y", ""),
				Constraint: c.Constraint,
				ConnectTo:  c.ConnectTo,
				DataType:   c.DataType,
			})
			defSQL.Cols = append(defSQL.Cols, dictlib.QryCol{
				No:         atoiOr(c.Seq, 0),
				Table:      c.Table,
				Field:      c.Field,
				Label:      c.Label,
				Width:      atoiOr(c.Width, 0),
				Return:     strings.EqualFold(c.Return, "Y"),
				Type:       atoiOr(c.Type, 0),
				Outer:      strings.EqualFold(c.Outer, "Y"),
				Constraint: c.Constraint,
				ConnectTo:  c.ConnectTo,
				DateType:   strings.EqualFold(c.DataType, "DATE"),
			})
		}
		if defSQL.HardCode {
			def.SQL = gpWinNoSQL
		} else {
			def.SQL = defSQL.BuildSQL(dictlib.QrySQLOpt{Schema: srcCompany, Oracle: true})
		}
		detail.Defs = append(detail.Defs, def)
	}

	if Format() == output.FormatTable {
		printWinDetailGP(&detail)
		return nil
	}
	return emitOne(detail)
}

func printWinDetailGP(d *gpWinDetail) {
	fmt.Printf("=== %s ===\n", d.Code)
	for _, h := range d.Defs {
		fmt.Printf("\n[%s]\n", h.Cust)
		fmt.Printf("标题:      %s\n", orDash(h.Title))
		fmt.Printf("HardCode:  %s\n", h.HardCode)
		fmt.Printf("单身可改:  %s\n", h.Unlocked)
		fmt.Printf("DISTINCT:  %s\n", h.Distinct)
		fmt.Printf("JOIN 表:   %s\n", orDash(h.JoinTables))
		fmt.Printf("Input WHERE:     %s\n", orDash(h.InputWhere))
		fmt.Printf("Construct WHERE: %s\n", orDash(h.ConstrWhere))
		fmt.Printf("备注:      %s\n", orDash(h.Memo))

		fmt.Println("\nSQL:")
		for _, line := range strings.Split(h.SQL, "\n") {
			fmt.Println("  " + line)
		}

		if len(h.Cols) > 0 {
			fmt.Printf("\n开窗栏位 (gac_file, %d):\n", len(h.Cols))
			headers := []string{"顺序", "表格", "字段", "显示名", "宽度", "回传", "控件", "OUTER", "输入约束", "串查"}
			var rows [][]string
			for _, c := range h.Cols {
				rows = append(rows, []string{c.Seq, c.Table, c.Field, c.Label, c.Width,
					c.Return, c.Widget, c.Outer, c.Constraint, c.ConnectTo})
			}
			output.PrintTable(headers, rows)
		}
	}
}

// gpWidgetLabel 把 gac07 的栏位型态翻成人话（0=Edit 1=CheckBox 2=DateEdit 3=ComboBox）。
func gpWidgetLabel(code string) string {
	switch strings.TrimSpace(code) {
	case "0":
		return "Edit"
	case "1":
		return "CheckBox"
	case "2":
		return "DateEdit"
	case "3":
		return "ComboBox"
	default:
		return code
	}
}

// yesNoLabel 只把 'Y' 翻成 yes，其余（含空）翻成 no。
func yesNoLabel(v, yes, no string) string {
	if strings.EqualFold(strings.TrimSpace(v), "Y") {
		return yes
	}
	return no
}

func atoiOr(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

func init() {
	winCmd.Flags().StringVar(&winKW, "kw", "", "按识别码/说明过滤 (仅列表模式)")
	winCmd.Flags().StringVar(&winLang, "lang", "zh_CN", "说明语言别 (dzcal002/dzcbl003, 默认 zh_CN)")
	Group.AddCommand(winCmd)
}
