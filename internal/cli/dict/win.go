package dict

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
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
	Params   []winParamView `json:"参数"`
	Cols     []winColView   `json:"显现设定"`
}

type winParamView struct {
	Seq      string `json:"顺序"`
	Name     string `json:"参数名称"`
	DateType string `json:"日期型态"`
	Desc     string `json:"说明"`
	Memo     string `json:"助记码"`
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
	Example: `  tt dict r.q
  tt dict r.q --kw 料号
  tt dict r.q q_apca001
  tt dict r.q q_apca001 --json`,
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
		return notFoundErr("开窗定义 %s 不存在", id).WithHint("列出全部开窗定义：tt dict r.q")
	}

	params, err := GetDB().QueryWinParams(id, winLang)
	if err != nil && !db.IsMissingTable(err) {
		return err
	}
	cols, err := GetDB().QueryWinCols(id)
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
		for _, p := range params {
			if p.Cust != h.Cust {
				continue
			}
			view.Params = append(view.Params, winParamView{
				Seq:      p.Seq,
				Name:     p.Name,
				DateType: typeWithCode(checkDateTypeLabel(p.DateType), p.DateType),
				Desc:     p.Desc,
				Memo:     p.Memo,
			})
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

		if len(h.Params) > 0 {
			fmt.Printf("\n参数 (dzcb_t, %d):\n", len(h.Params))
			headers := []string{"顺序", "参数名称", "日期型态", "说明", "助记码"}
			var rows [][]string
			for _, p := range h.Params {
				rows = append(rows, []string{p.Seq, p.Name, p.DateType, p.Desc, p.Memo})
			}
			output.PrintTable(headers, rows)
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

func init() {
	winCmd.Flags().StringVar(&winKW, "kw", "", "按识别码/说明过滤 (仅列表模式)")
	winCmd.Flags().StringVar(&winLang, "lang", "zh_CN", "说明语言别 (dzcal002/dzcbl003, 默认 zh_CN)")
	Group.AddCommand(winCmd)
}
