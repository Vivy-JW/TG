package dict

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/output"

	"github.com/spf13/cobra"
)

var (
	progKW         string
	progLang       string
	progSub        bool
	progTablesMode string
)

const progBatchTip = `批量查多个程序：tt dict prog "a,b,c"`

var progCmd = &cobra.Command{
	Use:     "prog [程序编号...]",
	Aliases: []string{"program", "job"},
	Short:   "查询程序/作业字典 (程序做什么、被哪些作业使用)",
	Long: `查询 T100 的程序与作业登记(azzi900 程式基本資料 / azzi910 作業基本資料):
程序编号对应的中文作业名、程序类别、归属模块、是否客制、引用主程序,
以及**哪些作业用了这个程序**——作业通过 gzzz_t.gzzz002 挂到程序上,
一个程序可以被多个作业使用(作业名称取它所挂程序的名称,与 azzi910 一致)。

读代码时的第一个问题"这个程序做什么"就用它;也可以拿作业编号去查它挂的是哪个程序。
主程序查不到时会继续查**子程序/元件登记**(gzde_t,参考作业 azzi901):子程序(aapq110_01)、
应用元件/库(cl_abi、s_xxx)都能给出它自己的说明与规格类别——两类登记互不重叠。
子程序/元件的列表与搜索用 --sub。
无参数时列出全部程序(--kw 按程序编号/中文名称搜索,从业务词找程序)。
程序名称有简体(--lang zh_CN,默认)与繁体(--lang zh_TW)两份,搜索用字要对应;
一个程序被很多作业使用时(如共用维护程序可达数百个),作业表默认只列前 --limit 行。
默认是 summary：保留程序/作业身份，用表只回编号。
用 --tables none|names|full 控制用表矩阵：none 不查/不返回，names 只回表格编号，
full 保留表说明与操作类别。--full 回到完整视图（默认用表也展开为 full）。
编号可用逗号分隔批量查询：单编号返回程序详情对象，多编号返回 items[] 数组
（每个编号带 found；缺失的列在 missing[]，单编号缺失则报 NOT_FOUND）。

JSON 形状（单编号响应会在 notes 里提示批量写法）：
  单编号 data = {程序?, 子程序?, 作业[], 表格[]}
  作业参数：程序对象上的“应用参数组/额外参数”；程序详情里的作业行也带同名字段
  多编号 data = {items: [{编号, found, 程序?, 子程序?, 作业[], 表格[]}], missing[]}
  搜索     data = [{程序编号, 程序名称, 程序类别, 归属模块, 客制, 作业数}]
数据族为"程序与作业/程序与表格/子程序与元件"(--help 末尾会显示本地是否已同步)。`,
	Example: `  tt dict prog aapi011          # 程序做什么 + 被哪些作业使用 + 用了哪些表
  tt dict prog aapq110_01       # 子程序:自己的说明 + 它的主程序
  tt dict prog cl_abi           # 库/元件:说明 + 用表清单
  tt dict prog --kw 对帐        # 按中文作业名找程序
  tt dict prog --sub --kw ABI   # 在子程序/元件里按业务词搜
  tt dict prog aapi011 --tables names  # 只要用表编号（默认）
  tt dict prog aapi011 --tables none   # 不看用表矩阵
  tt dict prog aapi011 --full          # 完整程序/作业/用表矩阵
  tt dict prog aapi011 --json
  tt dict prog "aapi011,aapq110_01,cl_abi"   # 批量；每个编号一条结果`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := normalizeProgTables(progTablesMode); err != nil {
			return err
		}
		if len(args) > 0 {
			srcDetail = "summary"
			if dictFull {
				srcDetail = "full"
				if !cmd.Flags().Changed("tables") {
					progTablesMode = "full"
				}
			}
		}
		if len(args) == 0 {
			if progSub {
				return runSubProgList()
			}
			return runProgList()
		}
		codes := splitNames(args[0])
		if len(codes) == 0 {
			return fmt.Errorf("未指定有效的程序编号")
		}
		if len(codes) > 1 {
			return runProgBatch(codes)
		}
		return runProgDetail(codes[0])
	},
}

// runProgDetail 打印一个程序编号的详情:程序信息 + 使用它的作业。
// 编号只登记为作业时,跟进它挂的程序再打印一遍(一眼看到"谁在用它")。
func runProgDetail(code string) error {
	info, err := GetDB().QueryProgInfo(code, progLang)
	if err != nil {
		if db.IsMissingTable(err) {
			fmt.Println(missingHint("程序与作业 (gzza_t/gzzz_t 等表)"))
			return printProgTables(code, true) // 用表索引是另一族,可能已同步
		}
		return err
	}
	if info == nil {
		// 主程序登记(gzza_t)里没有 → 查子程序/元件/库登记(gzde_t,参考作业 azzi901)。
		// 两张表互不重叠:实测 gzde_t 4,036 个 / gzza_t 4,147 个、交集 0。
		sub, serr := GetDB().QuerySubProgInfo(code, progLang)
		if serr != nil && !db.IsMissingTable(serr) {
			return serr
		}
		if sub != nil {
			tables, _ := GetDB().QueryProgTables(code, progLang)
			if Format() != output.FormatTable {
				item := map[string]any{
					"编号": code, "found": true, "程序": nil, "子程序": sub,
					"表格": tables, "作业": []any{},
				}
				if err := shapeProgTables(item); err != nil {
					return err
				}
				return emitOneNotes(item, progBatchTip)
			}
			printSubProgInfo(sub)
			if base := progBaseCode(code); base != "" {
				if bi, err := GetDB().QueryProgInfo(base, progLang); err == nil && bi != nil && bi.IsProg {
					fmt.Println("它的主程序: " + base + "(" + bi.Name + ");子程序自己做的事以文件头 `#+ Description:` 为准。")
				}
			}
			return printProgTables(code, true)
		}
		tables, _ := GetDB().QueryProgTables(code, progLang)
		if len(tables) == 0 {
			return notFoundErr("程序/作业/元件 %s 不存在", code).WithHint(progBatchTip)
		}
		if Format() != output.FormatTable {
			item := map[string]any{
				"编号": code, "found": false, "程序": nil, "作业": []any{},
				"表格": tables, "note": "未登记为程序/作业；gzdg_t 有用表索引",
			}
			if err := shapeProgTables(item); err != nil {
				return err
			}
			return emitOneNotes(item, progBatchTip)
		}
		fmt.Println("未找到程序/作业/元件 '" + code + "' 的登记；但 gzdg_t 有用表索引：")
		return printProgTables(code, true)
	}
	if Format() != output.FormatTable {
		jobs, jerr := GetDB().QueryProgJobs(code, progLang)
		if jerr != nil && !db.IsMissingTable(jerr) {
			return jerr
		}
		tables, terr := GetDB().QueryProgTables(code, progLang)
		if terr != nil && !db.IsMissingTable(terr) {
			return terr
		}
		notes := []string{progBatchTip}
		if len(jobs) == 0 && !info.IsJob {
			notes = append(notes, "程序存在，但没有作业登记（不是 NOT_FOUND）")
		}
		if len(tables) == 0 {
			notes = append(notes, "gzdg_t 没有该程序的用表登记（可能只用了动态 SQL）")
		}
		item := map[string]any{"程序": info, "作业": jobs, "表格": tables}
		if err := shapeProgTables(item); err != nil {
			return err
		}
		return emitOneNotes(item, notes...)
	}

	if info.IsProg {
		printProgInfo(info)
	} else {
		fmt.Printf("=== %s ===\n该编号不是程序登记,是作业:挂的程序 = %s\n", code, orDash(info.JobProg))
		if info.JobParamGrp != "" || info.JobExtra != "" {
			fmt.Printf("作业参数: 应用参数组=%s 额外参数=%s\n", orDash(info.JobParamGrp), orDash(info.JobExtra))
		}
		if info.JobProg != "" && info.JobProg != code {
			// 跟到它挂的程序:程序信息 + 作业清单(本作业会出现在其中)
			pi, err := GetDB().QueryProgInfo(info.JobProg, progLang)
			if err != nil && !db.IsMissingTable(err) {
				return err
			}
			if pi != nil && pi.IsProg {
				fmt.Println()
				printProgInfo(pi)
			}
		}
		return nil
	}

	jobs, err := GetDB().QueryProgJobs(code, progLang)
	if err != nil {
		if db.IsMissingTable(err) {
			fmt.Println(missingHint("程序与作业 (gzzz_t 等表)"))
			return printProgTables(code, false) // 程序与表格可能是另一族,继续试
		}
		return err
	}
	fmt.Printf("\n使用它的作业 (%d):\n", len(jobs))
	if len(jobs) == 0 {
		fmt.Println("  (无;该程序未被作业登记,或是被作其他用途引用)")
		return printProgTables(code, false)
	}
	headers := []string{"作业编号", "作业名称", "归属模块", "应用参数组", "额外参数", "参数组说明", "默认单据性质"}
	shown := jobs
	if limitOf() > 0 && len(jobs) > limitOf() {
		shown = jobs[:limitOf()]
	}
	var rows [][]string
	for _, j := range shown {
		rows = append(rows, []string{j.JobCode, j.JobName, j.Module, j.ParamGrp, j.Extra, j.ParamDesc, j.DocType})
	}
	output.PrintTable(headers, rows)
	if len(shown) < len(jobs) {
		fmt.Printf("\n… 还有 %d 个未显示(共 %d 个);--limit 0 显示全部,或 --json 导出\n",
			len(jobs)-len(shown), len(jobs))
	}
	return printProgTables(code, false)
}

// runProgBatch 批量查询多个程序编号。
// 单个编号缺失是 NOT_FOUND；批量时每个编号带 found，缺失的列在 missing[]，
// 这样 agent 能一次拿到全部可用的结果，又能准确知道哪些编号没登记。
func runProgBatch(codes []string) error {
	headers := []string{"程序编号", "登记状态", "名称", "作业数", "表数"}
	items := make([]map[string]any, 0, len(codes))
	rows := make([][]string, 0, len(codes))
	var missing []string
	for _, code := range codes {
		item, found, err := progDetailItem(code)
		if err != nil {
			return err
		}
		items = append(items, item)
		status := "未登记"
		if found {
			status = "已登记"
		} else {
			missing = append(missing, code)
		}
		var name string
		switch p := item["程序"].(type) {
		case *db.ProgInfo:
			name = p.Name
		}
		if name == "" {
			switch p := item["子程序"].(type) {
			case *db.SubProgInfo:
				name = p.Name
			}
		}
		var jobN, tableN int
		switch v := item["作业"].(type) {
		case []db.ProgJob:
			jobN = len(v)
		}
		switch v := item["表格"].(type) {
		case []db.ProgTableRow:
			tableN = len(v)
		case []string:
			tableN = len(v)
		}
		rows = append(rows, []string{code, status, name, fmt.Sprint(jobN), fmt.Sprint(tableN)})
	}
	data := map[string]any{"items": items}
	if len(missing) > 0 {
		data["missing"] = missing
	}
	var notes []string
	if len(missing) > 0 {
		notes = append(notes, "未登记编号："+strings.Join(missing, ", ")+"（单编号查询会报 NOT_FOUND）")
	}
	if Format() != output.FormatTable {
		return emitDetailNotes(data, headers, rows, notes...)
	}
	output.PrintTable(headers, rows)
	for _, n := range notes {
		fmt.Println("注意:" + n)
	}
	return nil
}

// normalizeProgTables 校验 --tables 投影。
func normalizeProgTables(s string) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(s))
	switch mode {
	case "":
		return "names", nil
	case "full", "none", "names":
		return mode, nil
	default:
		return "", fmt.Errorf("--tables 只接受 none | names | full（当前 %q）", s)
	}
}

// shapeProgTables 把详情里的“表格”按投影改成 []string / []ProgTableRow / 空。
func shapeProgTables(item map[string]any) error {
	mode, err := normalizeProgTables(progTablesMode)
	if err != nil {
		return err
	}
	switch mode {
	case "none":
		item["表格"] = []string{}
		return nil
	case "full":
		return nil
	}
	var names []string
	switch rows := item["表格"].(type) {
	case []db.ProgTableRow:
		for _, r := range rows {
			names = append(names, r.Table)
		}
	case []string:
		names = rows
	}
	item["表格"] = names
	return nil
}

// queryProgTablesForDetail 在 --tables none 时不查用表矩阵。
func queryProgTablesForDetail(code string) ([]db.ProgTableRow, error) {
	mode, err := normalizeProgTables(progTablesMode)
	if err != nil {
		return nil, err
	}
	if mode == "none" {
		return []db.ProgTableRow{}, nil
	}
	return GetDB().QueryProgTables(code, progLang)
}

// progDetailItem 取一个编号的详情，供批量查询使用；不打印任何东西。
// found=false 仍可能带“用表索引”（gzdg_t 覆盖库/元件/开窗），但不算程序登记。
func progDetailItem(code string) (map[string]any, bool, error) {
	info, err := GetDB().QueryProgInfo(code, progLang)
	if err != nil && !db.IsMissingTable(err) {
		return nil, false, err
	}
	if info != nil {
		jobs, jerr := GetDB().QueryProgJobs(code, progLang)
		if jerr != nil && !db.IsMissingTable(jerr) {
			return nil, false, jerr
		}
		tables, terr := queryProgTablesForDetail(code)
		if terr != nil && !db.IsMissingTable(terr) {
			return nil, false, terr
		}
		item := map[string]any{"编号": code, "found": true, "程序": info, "作业": jobs, "表格": tables}
		if err := shapeProgTables(item); err != nil {
			return nil, false, err
		}
		return item, true, nil
	}
	sub, serr := GetDB().QuerySubProgInfo(code, progLang)
	if serr != nil && !db.IsMissingTable(serr) {
		return nil, false, serr
	}
	tables, terr := queryProgTablesForDetail(code)
	if terr != nil && !db.IsMissingTable(terr) {
		return nil, false, terr
	}
	if sub != nil {
		item := map[string]any{"编号": code, "found": true, "程序": nil, "子程序": sub, "作业": []any{}, "表格": tables}
		if err := shapeProgTables(item); err != nil {
			return nil, false, err
		}
		return item, true, nil
	}
	item := map[string]any{"编号": code, "found": false, "程序": nil, "作业": []any{}, "表格": tables}
	if err := shapeProgTables(item); err != nil {
		return nil, false, err
	}
	if len(tables) > 0 {
		item["note"] = "未登记为程序/作业；gzdg_t 有用表索引"
	} else {
		item["note"] = "没有任何程序/作业/子程序/用表登记"
	}
	return item, false, nil
}

// printSubProgInfo 打印子程序/元件/库的登记(gzde_t + gzdel_t,参考作业 azzi901)。
func printSubProgInfo(p *db.SubProgInfo) {
	fmt.Printf("=== %s ===\n", p.Code)
	if p.Name != "" {
		fmt.Printf("说明: %s\n", p.Name)
	}
	var parts []string
	if p.Category != "" {
		parts = append(parts, "规格类别: "+p.Category+db.SubProgCategoryLabel(p.Category))
	}
	if p.Module != "" {
		parts = append(parts, "归属模块: "+p.Module)
	}
	if p.ProgCat != "" {
		parts = append(parts, "程序类别: "+p.ProgCat+categoryLabel(p.ProgCat))
	}
	if p.Cust != "" {
		parts = append(parts, "客制: "+p.Cust)
	}
	if p.Status != "" {
		parts = append(parts, "状态码: "+p.Status)
	}
	for _, x := range parts {
		fmt.Println(x)
	}
	if p.Industry != "" {
		fmt.Printf("归属行业别: %s\n", p.Industry)
	}
	fmt.Println("(登记在「子程序及应用元件基本数据表」gzde_t,不在程序/作业登记里)")
}

// runSubProgList 列出/搜索子程序与元件(--sub,--kw 按编号或说明过滤)。
func runSubProgList() error {
	list, err := GetDB().QuerySubProgList(progLang, progKW)
	if err != nil {
		if db.IsMissingTable(err) {
			return missingTableErr("子程序与元件 (gzde_t/gzdel_t)")
		}
		return err
	}
	headers := []string{"规格编号", "说明", "规格类别", "归属模块", "客制"}
	if len(list) == 0 {
		if Format() != output.FormatTable {
			return emit([]db.SubProgListItem{}, headers, nil)
		}
		if progKW != "" {
			fmt.Printf("没有匹配 '%s' 的子程序/元件。\n提示: 说明可能只有繁体或其它语言,换 --lang 或更短的词再试。\n", progKW)
		} else {
			fmt.Println(emptyHint("子程序与元件"))
		}
		return nil
	}
	var rows [][]string
	for _, p := range list {
		rows = append(rows, []string{p.Code, p.Name, p.Category, p.Module, p.Cust})
	}
	if Format() != output.FormatTable {
		return emit(list, headers, rows)
	}
	output.PrintTable(headers, rows)
	fmt.Printf("\n共 %d 个", len(list))
	if progKW == "" {
		fmt.Print(";用 --kw <业务词> 过滤")
	}
	fmt.Println()
	return nil
}

// printProgTables 打印"这个编号用了哪些表"(gzdg_t,由 T100 自己维护;参考作业 azzq902)。
// 该数据族缺失时只提示、不影响上面的程序与作业信息。
// fromNotFound=true 表示调用方刚说过"未登记为程序/作业":索引里没记录就静默,
// 有记录则说明它是库/元件/开窗之类——照样给出来(实测 cl_abi 22 条、s_apcp300 8 条)。
func printProgTables(code string, fromNotFound bool) error {
	mode, err := normalizeProgTables(progTablesMode)
	if err != nil {
		return err
	}
	if mode == "none" {
		return nil
	}
	tables, err := GetDB().QueryProgTables(code, progLang)
	if err != nil {
		if db.IsMissingTable(err) {
			fmt.Println()
			fmt.Println(missingHint("程序与表格 (gzdg_t)"))
			return nil
		}
		return err
	}
	if len(tables) == 0 {
		if fromNotFound {
			return nil
		}
		fmt.Println("\n使用的表格 (0):\n  (无登记;该程序未在 gzdg_t 登记用表,或只用了动态 SQL)")
		return nil
	}
	if mode == "names" {
		fmt.Printf("\n使用的表格 (%d): ", len(tables))
		for i, t := range tables {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Print(t.Table)
		}
		fmt.Println()
		return nil
	}
	fmt.Printf("\n使用的表格 (%d):\n", len(tables))
	shown := tables
	if limitOf() > 0 && len(tables) > limitOf() {
		shown = tables[:limitOf()]
	}
	var rows [][]string
	for _, t := range shown {
		rows = append(rows, []string{t.Table, t.TableDesc, t.Ops})
	}
	output.PrintTable([]string{"表格编号", "表说明", "操作"}, rows)
	if len(shown) < len(tables) {
		fmt.Printf("\n… 还有 %d 张未显示(共 %d 张);--limit 0 显示全部\n", len(tables)-len(shown), len(tables))
	}
	fmt.Println("操作: S=SELECT 查询 / I=INSERT 新增 / U=UPDATE 修改 / D=DELETE 删除")
	return nil
}

// printProgInfo 打印程序登记信息。
func printProgInfo(p *db.ProgInfo) {
	fmt.Printf("=== %s ===\n", p.Code)
	if p.Name != "" {
		fmt.Printf("程序名称: %s\n", p.Name)
	}
	if p.ShortName != "" {
		fmt.Printf("程序简称: %s\n", p.ShortName)
	}
	parts := make([]string, 0, 5)
	if p.Category != "" {
		parts = append(parts, "程序类别: "+p.Category+categoryLabel(p.Category))
	}
	if p.Module != "" {
		parts = append(parts, "归属模块: "+p.Module)
	}
	if p.Cust != "" {
		parts = append(parts, "客制: "+p.Cust)
	}
	if p.Status != "" {
		parts = append(parts, "状态码: "+p.Status)
	}
	for _, s := range parts {
		fmt.Println(s)
	}
	if p.RefMain != "" {
		fmt.Printf("引用主程序: %s\n", p.RefMain)
	}
	if p.RunCmd != "" {
		fmt.Printf("系统运行指令: %s\n", p.RunCmd)
	}
}

// runProgList 列表模式:列出/搜索程序(附各自的作业数)。
func runProgList() error {
	list, err := GetDB().QueryProgList(progLang, progKW)
	if err != nil {
		if db.IsMissingTable(err) {
			fmt.Println(missingHint("程序与作业 (gzza_t 等表)"))
			return nil
		}
		return err
	}
	headers := []string{"程序编号", "程序名称", "程序类别", "归属模块", "客制", "作业数"}
	if len(list) == 0 {
		if Format() != output.FormatTable {
			return emit([]db.ProgListItem{}, headers, nil)
		}
		if progKW != "" {
			fmt.Printf("没有匹配 '%s' 的程序。\n提示: 程序名称分简体(--lang zh_CN,默认)与繁体(--lang zh_TW)两份,用字可能是异体(如 对账/对帐/對帳);换个写法、换 --lang 或用更短的词再试。\n", progKW)
		} else {
			fmt.Println(emptyHint("程序与作业"))
		}
		return nil
	}

	var rows [][]string
	for _, p := range list {
		rows = append(rows, []string{p.Code, p.Name, p.Category, p.Module, p.Cust, fmt.Sprintf("%d", p.JobCount)})
	}
	if Format() != output.FormatTable {
		return emit(list, headers, rows)
	}
	output.PrintTable(headers, rows)
	fmt.Printf("\n共 %d 个程序", len(list))
	if progKW == "" {
		fmt.Print(";用 --kw <业务词> 按程序编号/中文名称过滤,如 tt dict prog --kw 对帐")
	}
	fmt.Println()
	return nil
}

// categoryLabel 程序类别码的中文补充(取 T100 程序编号第 4 码的类别约定)。
// 注意 ERP 里存的是大写(gzza002='I'),这里统一转小写再比——否则注解永远不出现。
func categoryLabel(code string) string {
	switch strings.ToLower(code) {
	case "i":
		return "(基本资料维护)"
	case "m":
		return "(主档维护)"
	case "t":
		return "(交易处理)"
	case "s":
		return "(参数设定)"
	case "p":
		return "(批次处理)"
	case "q":
		return "(查询)"
	case "r":
		return "(报表)"
	default:
		return ""
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// progBaseCode 把子程序/子元件编号还原成主程序编号:aapq110_01 → aapq110、
// axmi125_wf → axmi125、aapt110_01_rep → aapt110。不是这些形态时返回空串
// (普通程序编号如 aapi011、开窗码如 q_adzi052 都不动)。
func progBaseCode(code string) string {
	base := code
	for {
		i := strings.LastIndex(base, "_")
		if i <= 0 || !isSubSuffix(base[i+1:]) {
			break
		}
		base = base[:i]
	}
	if base == code {
		return ""
	}
	return base
}

// isSubSuffix 判断是否为子程序/子元件后缀:_01/_02…、_x01、_g01、_k01、_s01、_wf、_rep。
func isSubSuffix(s string) bool {
	switch s {
	case "wf", "rep":
		return true
	}
	if s == "" {
		return false
	}
	if isDigits(s) {
		return true
	}
	switch s[0] {
	case 'x', 'g', 'k', 's':
		return len(s) > 1 && isDigits(s[1:])
	}
	return false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func init() {
	progCmd.Flags().StringVar(&progKW, "kw", "", "按程序编号/中文名称搜索 (无编号时的列表模式)")
	progCmd.Flags().StringVar(&progLang, "lang", "zh_CN", "程序名称语言别 (gzzal002, 默认 zh_CN)")
	progCmd.Flags().BoolVar(&progSub, "sub", false, "无编号时:列出/搜索子程序与元件(gzde_t)而不是主程序")
	progCmd.Flags().StringVar(&progTablesMode, "tables", "names", "用表矩阵: none | names | full（默认 names）")
	progCmd.Flags().BoolVar(&dictFull, "full", false, "完整视图：用表展开为 full（等同于显式 --tables full）")
	Group.AddCommand(progCmd)
}
