package dict

// tt dict sysp / docp:查询参数定义档(gzsz_t,由 azzzi990 系统参数定义作业与
// azzi991 单据别参数维护作业维护)。两个作业操作同一张定义表,区别在参数群
// (gzsz001):azzi990 视域 = 非 ooac_t(A 系统级 gzsa_t / E 企业级 ooaa_t /
// S 据点级 ooab_t),azzi991 视域 = 恒 ooac_t(单据别 D 级,子表 gzsy_t 绑定
// 单据性质)。参数编号形如 A-SYS-0100 / E-CIR-0001 / D-MFG-0076。
//
// 本命令只查"定义与说明"(名称/说明来自 gzszl_t 多语言表);参数当前值在
// 客户化值表(gzsa_t/ooaa_t/ooab_t/ooac_t),不在查询范围。语言策略同 msg:
// 默认显示 --lang(zh_CN)行,无该语言行时列出可用语言。

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	paramLang   string
	docListMode string
)

// docpObj 单据别参数的 JSON 展示对象。
type docpObj struct {
	Param    db.ParamDefRow  `json:"参数"`
	Docs     []db.DocTypeRow `json:"单据性质,omitempty"`
	DocCount int             `json:"单据性质数,omitempty"`
}

var syspCmd = &cobra.Command{
	Use:   "sysp <编号>[,<编号>...]",
	Short: "查询系统参数说明",
	Long: `查询系统参数说明:系统/企业/据点级参数的定义与用途(不含单据别参数,
那是 tt dict docp 的范围)。
参数编号 = 型态码 + 领域3码 + 4位流水:A-SYS-0040(A=系统级)、E-CIR-0001
(E=企业级)、S-BAS-0028(S=据点级)。
返回: 名称/说明(多语言)、群与级别、型态(1=Y/N 2=整数选项 3=范围设定
4=字符或SCC 5=日期)、领域、预设值、值域、SCC 选项、校核/开窗引用、
异常处理、修改频度与状态;参数实际设定值不在查询范围。`,
	Example: `  tt dict sysp A-SYS-0100
  tt dict sysp S-BAS-0028 --lang zh_TW
  tt dict sysp "A-SYS-0100,E-CIR-0001" --json
  tt dict sysp D-MFG-0076            # 单据别参数会提示改用 docp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runParamQuery(args[0], false)
	},
}

var docpCmd = &cobra.Command{
	Use:   "docp <编号>[,<编号>...]",
	Short: "查询单据参数说明",
	Long: `查询单据参数说明:单据别参数(编号 D- 开头,如 D-MFG-0076)的定义与用途,
并附它适用的单据性质清单(模块 + 单据性质)。
返回: 名称/说明(多语言)、型态(同 sysp:1=Y/N 2=整数选项 3=范围设定
4=字符或SCC 5=日期)、领域、预设值、值域、SCC 选项、校核/开窗引用等定义,
以及单据性质绑定表（--doclist none|count|full 控制展开程度）;
参数实际设定值在各单据别的参数值维护作业,不在查询范围。`,
	Example: `  tt dict docp D-MFG-0076
  tt dict docp D-BAS-0058 --lang zh_TW
  tt dict docp "D-MFG-0076,D-BAS-0058" --json
  tt dict docp D-MFG-0076 --doclist count  # 只给引用数量，不展开清单
  tt dict docp D-MFG-0076 --doclist none   # 不查引用清单
  tt dict docp A-SYS-0100            # 系统参数会提示改用 sysp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runParamQuery(args[0], true)
	},
}

func init() {
	Group.AddCommand(syspCmd, docpCmd)
	syspCmd.Flags().StringVar(&paramLang, "lang", "zh_CN", "说明语言别 (gzszl003, 缺省 zh_CN)")
	docpCmd.Flags().StringVar(&paramLang, "lang", "zh_CN", "说明语言别 (gzszl003, 缺省 zh_CN)")
	docpCmd.Flags().StringVar(&docListMode, "doclist", "full", "单据性质清单: none | count | full")
}

// runParamQuery 按视域查询并输出:docOnly=true 走 docp(ooac_t 群 + gzsy 子行)。
func runParamQuery(arg string, docOnly bool) error {
	if docOnly {
		if _, err := normalizeDocListMode(docListMode); err != nil {
			return err
		}
	}
	codes := splitNames(arg)
	if len(codes) == 0 {
		return fmt.Errorf("未指定有效的参数编号")
	}
	other := "docp"
	if docOnly {
		other = "sysp"
	}

	structured := Format() != output.FormatTable
	var pickedSys []db.ParamDefRow
	var pickedDoc []docpObj
	for _, code := range codes {
		rows, err := GetDB().QueryParam(code)
		if err != nil {
			if db.IsMissingTable(err) {
				return missingTableErr("参数档 (gzsz_t/gzszl_t)")
			}
			return err
		}
		if len(rows) == 0 {
			if !structured {
				fmt.Printf("未找到参数编号 '%s'。\n", code)
			}
			continue
		}
		// 视域过滤:sysp 排除 ooac_t;docp 只要 ooac_t
		var kept []db.ParamDefRow
		for i := range rows {
			if (rows[i].Group == "ooac_t") == docOnly {
				kept = append(kept, rows[i])
			}
		}
		if len(kept) == 0 {
			if structured {
				continue
			}
			fmt.Printf("编号 '%s' 是%s参数;请用 tt dict %s 查询。\n", code, paramKind(rows[0].Group), other)
			continue
		}
		match, avail := pickLangRow(kept, paramLang, func(r db.ParamDefRow) string { return r.Lang })
		if match == nil {
			if structured {
				continue
			}
			fmt.Printf("%s\n", langMissMsg(code, paramLang, avail))
			continue
		}
		if !docOnly {
			pickedSys = append(pickedSys, *match)
			if !structured {
				printParam(match, nil, 0)
			}
			continue
		}
		// docp:附带单据性质绑定(gzsy_t 缺表时静默降级为空)。
		// --doclist none 不查；count 查但只留数量；full 保留明细。
		mode, _ := normalizeDocListMode(docListMode)
		var docs []db.DocTypeRow
		docCount := 0
		if mode != "none" {
			docs, err = GetDB().QueryDocTypes(code, match.Lang)
			if err != nil && !db.IsMissingTable(err) {
				return err
			}
			if err != nil {
				docs = nil
			}
			docCount = len(docs)
			if mode == "count" {
				docs = nil
			}
		}
		if structured {
			pickedDoc = append(pickedDoc, docpObj{Param: *match, Docs: docs, DocCount: docCount})
		} else {
			printParam(match, docs, docCount)
		}
	}

	if structured {
		if docOnly {
			return emitOne(pickedDoc)
		}
		return emitOne(pickedSys)
	}
	return nil
}

// normalizeDocListMode 校验 docp 的引用清单投影。
func normalizeDocListMode(s string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(s))
	switch mode {
	case "", "full":
		return "full", nil
	case "none", "count":
		return mode, nil
	default:
		return "", fmt.Errorf("--doclist 只接受 none | count | full（当前 %q）", s)
	}
}

// paramKind 参数群 → 级别说明(命令提示用)。
func paramKind(group string) string {
	switch group {
	case "ooac_t":
		return "单据别"
	case "gzsa_t":
		return "系统级"
	case "ooaa_t":
		return "企业级"
	case "ooab_t":
		return "据点级"
	}
	return group
}

// paramLevelLabel 参数群 → 中文级别标签(展示用)。
func paramLevelLabel(group string) string {
	switch group {
	case "gzsa_t":
		return "系统级参数"
	case "ooaa_t":
		return "企业级参数"
	case "ooab_t":
		return "据点级参数"
	case "ooac_t":
		return "单据别参数"
	}
	return ""
}

func printParam(m *db.ParamDefRow, docs []db.DocTypeRow, docCount int) {
	fmt.Printf("=== %s (%s) ===\n", m.Code, m.Lang)
	if m.Name != "" {
		fmt.Printf("名称:     %s\n", m.Name)
	}
	if m.Desc != "" {
		fmt.Printf("说明:     %s\n", m.Desc)
	}
	level := paramLevelLabel(m.Group)
	if level != "" && m.GroupName != "" {
		level += "(" + m.GroupName + ")"
	}
	line := "群:       " + m.Group
	if level != "" {
		line += "  " + level
	}
	fmt.Println(line)
	fmt.Printf("型态:     %s\n", typeWithCode(paramTypeLabel(m.TypeCode), m.TypeCode))
	if m.AreaCode != "" {
		fmt.Printf("领域:     %s\n", m.AreaCode)
	}
	if m.Default != "" {
		fmt.Printf("预设值:   %s\n", m.Default)
	}
	if m.RangeVal != "" {
		fmt.Printf("值域:     %s\n", m.RangeVal)
	}
	if m.DateFmt != "" {
		fmt.Printf("日期格式: %s\n", m.DateFmt)
	}
	if m.SccCode != "" {
		fmt.Printf("SCC选项:  清单 %s(型态 4 字符时的可选项来源)\n", m.SccCode)
	}
	if m.RVCode != "" {
		fmt.Printf("校核带值: %s\n", m.RVCode)
	}
	if m.RQCode != "" {
		fmt.Printf("开窗程序: %s\n", m.RQCode)
	}
	if m.Except != "" {
		fmt.Printf("异常处理: %s\n", typeWithCode(paramExceptLabel(m.Except), m.Except))
	}
	if m.Freq != "" {
		fmt.Printf("修改频度: %s (%s)\n", paramFreqLabel(m.Freq), m.Freq)
	}
	if m.Live != "" {
		fmt.Printf("即时抓取: %s\n", m.Live)
	}
	if m.ValueProg != "" {
		fmt.Printf("值维护作业: %s\n", m.ValueProg)
	}
	if m.Status != "" {
		fmt.Printf("状态:     %s\n", statusLabel(m.Status))
	}
	for _, text := range [][2]string{{"备注一", m.Desc2}, {"备注二", m.Desc3}} {
		if text[1] == "" {
			continue
		}
		fmt.Printf("\n%s (gzszl006/007):\n", text[0])
		for _, l := range strings.Split(text[1], "\n") {
			fmt.Println("  " + l)
		}
	}
	if len(docs) > 0 {
		fmt.Printf("\n单据性质绑定 (gzsy_t, %d):\n", len(docs))
		headers := []string{"单据性质", "名称", "模块", "已抛转"}
		var rows [][]string
		for _, d := range docs {
			rows = append(rows, []string{d.DocType, d.DocName, d.Module, d.Generate})
		}
		output.PrintTable(headers, rows)
	}
	fmt.Println()
}

// paramTypeLabel 输入型态 (SCC'89'): 1=Y/N 2=整数选项 3=范围设定 4=字符或SCC 5=日期。
func paramTypeLabel(code string) string {
	switch code {
	case "1":
		return "Y/N"
	case "2":
		return "整数选项"
	case "3":
		return "范围设定"
	case "4":
		return "字符或SCC"
	case "5":
		return "日期"
	default:
		return code
	}
}

// paramExceptLabel 取参异常处理 (SCC'160'): 1=改抓设定时的预设值 2=显示错误回传NULL 3=不回传错误直接NULL。
func paramExceptLabel(code string) string {
	switch code {
	case "1":
		return "改抓设定时的预设值"
	case "2":
		return "显示错误并回传NULL"
	case "3":
		return "不回传错误直接回传NULL"
	default:
		return code
	}
}

// paramFreqLabel 修改频度 (SCC'258'): Y=未限制 N=配置后不可修改。
func paramFreqLabel(code string) string {
	switch code {
	case "Y":
		return "未限制"
	case "N":
		return "配置后不可修改"
	default:
		return code
	}
}
