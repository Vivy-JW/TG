package dict

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	checkKW   string
	checkLang string
)

// checkListItem is the display form of one check definition in list mode.
type checkListItem struct {
	ID     string `json:"识别码"`
	Cust   string `json:"客制"`
	Desc   string `json:"说明"`
	Type   string `json:"型态"`
	ErrMsg string `json:"错误讯息"`
	Remark string `json:"备注"`
}

// checkHeaderView is the display form of one check header variant (标准/客制).
type checkHeaderView struct {
	Cust     string           `json:"客制"`
	Type     string           `json:"型态"`
	ErrMsg   string           `json:"错误讯息"`
	Industry string           `json:"行业别"`
	Status   string           `json:"状态码"`
	Desc     string           `json:"说明"`
	Remark   string           `json:"备注"`
	SQL      string           `json:"SQL指令"`
	Params   []checkParamView `json:"参数"`
	Conds    []checkCondView  `json:"判断条件"`
}

type checkParamView struct {
	Seq      string `json:"顺序"`
	Name     string `json:"参数名称"`
	DateType string `json:"日期型态"`
	Desc     string `json:"说明"`
	Remark   string `json:"备注"`
}

type checkCondView struct {
	Seq    string `json:"顺序"`
	Cond   string `json:"条件"`
	ErrMsg string `json:"错误讯息"`
}

// checkDetail is the display form of a complete check definition.
type checkDetail struct {
	ID      string            `json:"识别码"`
	Headers []checkHeaderView `json:"校验定义"`
}

var checkCmd = &cobra.Command{
	Use:     "r.v [识别码]",
	Aliases: []string{"rv", "check"},
	Short:   "查询校验带值定义 (r.v)",
	Long: `查询字段校验规则 (r.v):系统保存数据前的检查是可复用的校验模板,每条校验一个
识别码,含要执行的校验 SQL(SQL 内用 <field>、arg1~9、:TODAY 等占位符,运行时代入)、
外部参数与判断条件。
无参数时列出全部校验定义(--kw 按识别码/说明过滤);指定识别码显示完整详情:
校验 SQL 原文(附标签图例)、参数、判断条件与错误讯息。
识别码形如 v_ooba002_07。命令名对齐 T100 原生工具 r.v(旧名 rv/check 仍可用)。`,
	Example: `  tt dict r.v
  tt dict r.v --kw 料号
  tt dict r.v v_ooba002_07
  tt dict r.v v_ooba002_07 --json`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return runCheckDetail(args[0])
		}
		return runCheckList()
	},
}

func runCheckList() error {
	rows, err := GetDB().QueryCheckList(checkLang, checkKW)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("校验定义 (dzcd_t 等表)")
		}
		return err
	}
	items := mergeCheckList(rows)
	if len(items) == 0 {
		if Format() != output.FormatTable {
			return emit([]checkListItem{}, []string{"识别码", "客制", "说明", "型态", "错误讯息", "备注"}, nil)
		}
		fmt.Println(emptyHint("校验定义"))
		return nil
	}

	headers := []string{"识别码", "客制", "说明", "型态", "错误讯息", "备注"}
	var tableRows [][]string
	for _, it := range items {
		tableRows = append(tableRows, []string{it.ID, it.Cust, it.Desc, it.Type, it.ErrMsg, it.Remark})
	}
	if Format() != output.FormatTable {
		return emit(items, headers, tableRows)
	}
	output.PrintTable(headers, tableRows)
	return nil
}

// mergeCheckList merges the 标准/客制 variants of the same dzcd001 into one row.
func mergeCheckList(rows []db.CheckListRow) []checkListItem {
	var items []checkListItem
	for _, r := range rows {
		label := custLabel(r.Cust)
		if len(items) > 0 && items[len(items)-1].ID == r.ID {
			it := &items[len(items)-1]
			if !strings.Contains(it.Cust, label) {
				it.Cust += "," + label
			}
			if it.Desc == "" {
				it.Desc = r.Desc
			}
			continue
		}
		items = append(items, checkListItem{
			ID:     r.ID,
			Cust:   custLabel(r.Cust),
			Desc:   r.Desc,
			Type:   typeWithCode(checkTypeLabel(r.TypeCode), r.TypeCode),
			ErrMsg: r.ErrMsg,
			Remark: r.Remark,
		})
	}
	return items
}

func runCheckDetail(id string) error {
	if IsCSV() {
		return fmt.Errorf("详情模式不支持 --csv，请使用列表模式或 --json")
	}

	headers, err := GetDB().QueryCheckHeaders(id, checkLang)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("校验定义 (dzcd_t 等表)")
		}
		return err
	}
	if len(headers) == 0 {
		return notFoundErr("校验定义 %s 不存在", id).WithHint("列出全部校验定义：tt dict r.v")
	}

	params, err := GetDB().QueryCheckParams(id, checkLang)
	if err != nil && !db.IsMissingTable(err) {
		return err
	}
	conds, err := GetDB().QueryCheckConds(id)
	if err != nil && !db.IsMissingTable(err) {
		return err
	}

	detail := checkDetail{ID: id}
	for _, h := range headers {
		view := checkHeaderView{
			Cust:     custLabel(h.Cust),
			Type:     typeWithCode(checkTypeLabel(h.TypeCode), h.TypeCode),
			ErrMsg:   h.ErrMsg,
			Industry: h.Industry,
			Status:   h.Status,
			Desc:     h.Desc,
			Remark:   h.Remark,
			SQL:      h.SQL,
		}
		for _, p := range params {
			if p.Cust != h.Cust {
				continue
			}
			view.Params = append(view.Params, checkParamView{
				Seq:      p.Seq,
				Name:     p.Name,
				DateType: typeWithCode(checkDateTypeLabel(p.DateType), p.DateType),
				Desc:     p.Desc,
				Remark:   p.Remark,
			})
		}
		for _, c := range conds {
			if c.Cust != h.Cust {
				continue
			}
			view.Conds = append(view.Conds, checkCondView{Seq: c.Seq, Cond: c.Cond, ErrMsg: c.ErrMsg})
		}
		detail.Headers = append(detail.Headers, view)
	}

	if Format() == output.FormatTable {
		printCheckDetail(&detail)
		return nil
	}
	return emitOne(detail)
}

func printCheckDetail(d *checkDetail) {
	fmt.Printf("=== %s ===\n", d.ID)
	for _, h := range d.Headers {
		fmt.Printf("\n[%s]\n", h.Cust)
		fmt.Printf("型态:     %s\n", h.Type)
		fmt.Printf("错误讯息: %s\n", h.ErrMsg)
		fmt.Printf("行业别:   %s\n", h.Industry)
		fmt.Printf("状态码:   %s\n", h.Status)
		fmt.Printf("说明:     %s\n", h.Desc)
		fmt.Printf("备注:     %s\n", h.Remark)

		fmt.Println("\nSQL 指令 (dzcd003):")
		if h.SQL == "" {
			fmt.Println("  (无)")
		} else {
			for _, line := range strings.Split(h.SQL, "\n") {
				fmt.Println("  " + line)
			}
		}

		if len(h.Params) > 0 {
			fmt.Printf("\n参数 (dzce_t, %d):\n", len(h.Params))
			headers := []string{"顺序", "参数名称", "日期型态", "说明", "备注"}
			var rows [][]string
			for _, p := range h.Params {
				rows = append(rows, []string{p.Seq, p.Name, p.DateType, p.Desc, p.Remark})
			}
			output.PrintTable(headers, rows)
		}

		if len(h.Conds) > 0 {
			fmt.Printf("\n判断条件 (dzch_t, %d):\n", len(h.Conds))
			headers := []string{"顺序", "条件", "错误讯息"}
			var rows [][]string
			for _, c := range h.Conds {
				rows = append(rows, []string{c.Seq, c.Cond, c.ErrMsg})
			}
			output.PrintTable(headers, rows)
		}
	}
	fmt.Print(checkTagLegend)
}

// custLabel maps dzcd002/dzce003/dzch005 cust flags to Chinese labels.
func custLabel(code string) string {
	switch code {
	case "s":
		return "标准"
	case "c":
		return "客制"
	default:
		return code
	}
}

// checkTypeLabel maps dzcd005 type codes to Chinese labels
// (1=检查存在, 2=带值, 3=检查存在并带值).
func checkTypeLabel(code string) string {
	switch code {
	case "1":
		return "检查存在"
	case "2":
		return "带值"
	case "3":
		return "检查存在并带值"
	default:
		return code
	}
}

// checkDateTypeLabel maps dzcestus codes to Chinese labels
// (1=年月日, 2=年月日时分秒, 3=毫秒, 其他=普通字符串).
func checkDateTypeLabel(code string) string {
	switch code {
	case "1":
		return "年月日"
	case "2":
		return "年月日时分秒"
	case "3":
		return "年月日时分秒毫秒"
	default:
		return "字符串"
	}
}

// typeWithCode formats a label with its raw code, e.g. "检查存在并带值 (3)".
func typeWithCode(label, code string) string {
	if code == "" || code == label {
		return label
	}
	return fmt.Sprintf("%s (%s)", label, code)
}

const checkTagLegend = `
标签说明 (校验 SQL 中的占位符):
  <field>...</field>   回传字段 (型态 2/3 时由 SQL 查出带回)
  <table>...</table>   查询的表
  <wc>...</wc>         外部追加 WHERE 条件的插入点
  <count>...</count>   COUNT 区间, 用于判断存在
  arg1~arg9            外部参数, 按 dzce_t 定义代入 (日期参数自动转 TO_DATE)
  :DEPT :SITE :USER :LANG :ENT :LEGAL :DLANG   全局变量
  :TODAY               当前日期
`

func init() {
	checkCmd.Flags().StringVar(&checkKW, "kw", "", "按识别码/说明过滤 (仅列表模式)")
	checkCmd.Flags().StringVar(&checkLang, "lang", "zh_CN", "说明语言别 (dzcdl002/dzcel003, 默认 zh_CN)")
	Group.AddCommand(checkCmd)
}
