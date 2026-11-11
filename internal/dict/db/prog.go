package db

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// 程序与作业字典(azzi900 程式基本資料設定作業 / azzi910 作業基本資料維護):
//
//	gzza_t  程序档:gzza001 程序编号(PK)、gzza002 程序类别、gzza003 归属模块、
//	        gzza008 引用主程序编号、gzza011 客制
//	gzzal_t 程序名称多语言:gzzal001 程序编号 + gzzal002 语言别 → gzzal003 程序名称
//	gzzz_t  作业编号设置表:gzzz001 作业编号(PK)、gzzz002 程序编号(作业挂的程序,
//	        一个程序可被多个作业使用)、gzzz003 应用参数组编号、gzzz004 额外参数、
//	        gzzz005 归属模块
//	gzzk_t  程序应用参数组设置表:gzzk001 程序编号 + gzzk002 参数组编号 → gzzk003 说明
//
// 作业的显示名称既不是独立字段也不是独立表:工具链(azzi910)用
// gzzal_t.gzzal003 按 gzzz002 关联取得,即"作业名称 = 它挂的程序名称"。

// ProgInfo 一个编号作为"程序"的登记信息;若该编号同时是"作业"也一并给出它挂的程序。
type ProgInfo struct {
	Code      string `json:"编号"`
	IsProg    bool   `json:"是程序"`
	Name      string `json:"程序名称"` // gzzal003(指定语言)
	ShortName string `json:"程序简称"` // gzzal005
	Category  string `json:"程序类别"` // gzza002
	// CategoryName 是程序类别码的中文名。产品口径不同(T100 与 TOP GP 的 M/T 含义
	// 不一样)，所以由数据源自己给：T100 侧留空、由命令层按 T100 码表翻译；
	// GP 侧由 gpdb 填 GP 码表的名字。空值不出现在输出里。
	CategoryName string `json:"程序类别名称,omitempty"`
	Module       string `json:"归属模块"`   // gzza003
	Cust         string `json:"客制"`     // gzza011
	RefMain      string `json:"引用主程序"`  // gzza008
	RunCmd       string `json:"系统运行指令"` // gzza004
	Status       string `json:"状态码"`    // gzzastus
	IsJob        bool   `json:"是作业"`
	JobProg      string `json:"作业挂的程序"` // gzzz002(该编号作为作业登记时)
	// 下面两项只在“这个编号本身是作业”时有值：同名作业靠它们区分。
	JobParamGrp string `json:"应用参数组"` // gzzz003
	JobExtra    string `json:"额外参数"`  // gzzz004
	// Memo 程序/作业的用途说明。T100 的程序档没有这个字段，GP 的 gaz_file.gaz04 有
	// (如「維護所有資料檔案之關連性」)，所以同样由数据源决定填不填。
	Memo string `json:"作用说明,omitempty"`
	// TableCount 这个程序用到的表格数。只在数据源顺手能算出来时给(如 GP 的
	// zr_file)，T100 侧留空 —— 要看明细还得走 --tables。
	TableCount int `json:"表格总数,omitempty"`
}

// ProgJob 一个使用某程序的作业(gzzz_t 按 gzzz002 关联到程序)。
type ProgJob struct {
	JobCode   string `json:"作业编号"`   // gzzz001
	JobName   string `json:"作业名称"`   // gzzal003(经 gzzz002 关联的程序名称)
	Module    string `json:"归属模块"`   // gzzz005
	ParamGrp  string `json:"应用参数组"`  // gzzz003
	Extra     string `json:"额外参数"`   // gzzz004(拼在程序名后的运行参数)
	ParamDesc string `json:"参数组说明"`  // gzzk003(经 gzzk001/gzzk002)
	DocType   string `json:"默认单据性质"` // gzzz006
	Status    string `json:"状态码"`    // gzzzstus
}

// ProgListItem 程序列表/搜索(prog --kw)的一行。
type ProgListItem struct {
	Code     string `json:"程序编号"`
	Name     string `json:"程序名称"`
	Category string `json:"程序类别"`
	// CategoryName 同 ProgInfo.CategoryName：产品口径不同，由数据源决定填不填。
	CategoryName string `json:"程序类别名称,omitempty"`
	Module       string `json:"归属模块"`
	Cust         string `json:"客制"`
	JobCount     int    `json:"作业数"`
}

// ---- 程序 ↔ 表格(gzdg_t 程序与应用表格功能分析表,由 T100 自己维护) ----
//
//	gzdg001 程序编号 + gzdg002 表格编号 + gzdg003 功能类别(SCC 212:
//	I=INSERT 新增 / S=SELECT 查询 / U=UPDATE 修改 / D=DELETE 删除)构成主键。
//	参考作业 azzq902「程式編號對應表格查詢」:它 join gzzal_t(程序名称)与
//	dzeal_t(表名称)展示两侧。实测正式区 18.2 万行 / 14,448 个程序 / 3,825 张表。

// ProgTableRow 一个程序用到的一张表(同一张表的多个操作合并为一行,如 "S/I/U/D")。
type ProgTableRow struct {
	Table     string `json:"表格编号"`
	TableDesc string `json:"表说明"`
	Ops       string `json:"操作"`
}

// TableProgRow 一个使用某表的程序。
type TableProgRow struct {
	Prog     string `json:"程序编号"`
	ProgName string `json:"程序名称"`
	Ops      string `json:"操作"`
}

// ProgLinkRow 程序链接的一行:gal_file(程序 → 模块)。
// Entry 为真表示 gal03 == gal01(这一行是入口模块,程序名等于作业代号)。
type ProgLinkRow struct {
	Module     string `json:"module"`
	LinkedProg string `json:"linkedProg"`
	LinkFlag   string `json:"linkFlag,omitempty"`
	Entry      bool   `json:"entry"`
}

// QueryProgLinks 本地字典库回答不了:gal_file 没有进同步清单
// (整表 30 万行级,只有查链接时才用得上),所以只给一句明确的指路,
// 而不是返回空列表让人以为"这个程序没有链接"。
func (d *DB) QueryProgLinks(string) ([]ProgLinkRow, error) {
	return nil, ErrProgLinksRemoteOnly
}

// ErrProgLinksRemoteOnly 程序链接只有远程库能回答(见 QueryProgLinks)。
var ErrProgLinksRemoteOnly = errors.New("程序链接(gal_file)只在远程库里,本地字典库不同步这张表")

// mergeOps 把同一目标的多个操作类别合并成固定顺序的字符串(S/I/U/D,再按字母补未知码)。
func mergeOps(ops []string) string {
	seen := make(map[string]bool, len(ops))
	for _, o := range ops {
		seen[o] = true
	}
	var out []string
	for _, code := range []string{"S", "I", "U", "D"} {
		if seen[code] {
			out = append(out, code)
			delete(seen, code)
		}
	}
	var rest []string
	for code := range seen {
		rest = append(rest, code)
	}
	sort.Strings(rest)
	return strings.Join(append(out, rest...), "/")
}

// ---- 子程序与元件(gzde_t 子程序及应用元件基本数据表,参考作业 azzi901) ----
//
//	gzde001 规格编号(PK)、gzde002 归属模块(LIB/SUB 等 $COM 子目录)、
//	gzde003 规格类别(SCC 91:B=应用元件 M=主程序 S=子程序 G=报表元件-GR类
//	X=报表元件-XG/FR类 K=报表组件-XR类 W=WebService元件)、gzde005 程序类别、
//	gzde008 客制、gzde009 归属行业别;说明在 gzdel_t(多语言)。
//
// 与 gzza_t(主程序)的关系:**互不重叠**——实测正式区 gzde_t 4,036 个、
// gzza_t 4,147 个,交集 0(本站 gzde_t 无 M 主程序行)。两者合起来才是完整的
// "这个编号是什么":主程序查 gzza_t,子程序/元件/库查 gzde_t。

// SubProgInfo 一个编号在"子程序/元件/库"登记里的信息。
type SubProgInfo struct {
	Code     string `json:"规格编号"`   // gzde001
	Name     string `json:"说明"`     // gzdel003(指定语言)
	Category string `json:"规格类别"`   // gzde003(SCC 91)
	Module   string `json:"归属模块"`   // gzde002
	Cust     string `json:"客制"`     // gzde008
	Industry string `json:"归属行业别"`  // gzde009
	ProgCat  string `json:"程序类别"`   // gzde005
	GenType  string `json:"程序生成类型"` // gzde006
	Status   string `json:"状态码"`    // gzdestus
}

// SubProgListItem 子程序/元件列表的一行。
type SubProgListItem struct {
	Code     string `json:"规格编号"`
	Name     string `json:"说明"`
	Category string `json:"规格类别"`
	Module   string `json:"归属模块"`
	Cust     string `json:"客制"`
}

// QuerySubProgInfo 查子程序/元件/库的登记。
//
// TOP GP 没有目录中的"子程序"实体：4GL 的 FUNCTION 住在本程序自己的源码里，
// 不登记到字典表。这里固定返回"未收录"，命令层据此走"没有这条记录"的分支。
func (d *DB) QuerySubProgInfo(code, lang string) (*SubProgInfo, error) {
	return nil, nil
}

// QuerySubProgList 列出/搜索子程序与元件。GP 无子程序目录，恒为空。
func (d *DB) QuerySubProgList(lang, keyword string) ([]SubProgListItem, error) {
	return []SubProgListItem{}, nil
}

// progNameColumn 按语言挑程序名称列：zh_TW 用 gaz_file 的繁体名（同步时落到 name_tw），
// 其余语言用简体名（name）。GP 的 gaz02 只有 0（繁）/2（简）两种。
func progNameColumn(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh_tw", "zhtw", "tw", "0":
		return "COALESCE(NULLIF(p.name_tw,''), p.name, '')"
	default:
		return "COALESCE(p.name, '')"
	}
}

// CategoryName 把 GP 的 zz_file.zz03 程序类别码翻成中文注解。
// 取值来自 azz/4gl/../../azz/sdd/p_zz.sdd 的「欄位說明」；GP 只有这九种。
func CategoryName(code string) string {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "M":
		return "目录程式"
	case "I":
		return "建档维护"
	case "U":
		return "公用程式"
	case "P":
		return "处理类"
	case "R":
		return "报表类"
	case "Q":
		return "查询"
	case "T":
		return "异动单据"
	case "S":
		return "设定"
	case "O":
		return "其他"
	default:
		return ""
	}
}

// SubProgCategoryLabel 规格类别(SCC 91)的中文注解。
func SubProgCategoryLabel(code string) string {
	switch strings.ToUpper(code) {
	case "B":
		return "(应用元件)"
	case "M":
		return "(主程序)"
	case "S":
		return "(子程序)"
	case "G":
		return "(报表元件-GR类)"
	case "X":
		return "(报表元件-XG/FR类)"
	case "K":
		return "(报表组件-XR类)"
	case "W":
		return "(WebService元件)"
	default:
		return ""
	}
}

// GetCol 取行内第 i 列(越界返回空串)。语义与 live 包的 get 一致,供共用的合并函数使用。
func GetCol(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

// GroupProgTables 把 (表格编号, 表说明, 操作) 行按表格合并操作类别。
// 本地与远程共用同一份合并逻辑(远程查询返回 [][]string,用 get 取列)。
func GroupProgTables(rows [][]string, get func([]string, int) string) []ProgTableRow {
	var result []ProgTableRow
	idx := make(map[string]int)
	var ops [][]string
	for _, r := range rows {
		table := get(r, 0)
		if i, ok := idx[table]; ok {
			ops[i] = append(ops[i], get(r, 2))
			continue
		}
		idx[table] = len(result)
		result = append(result, ProgTableRow{Table: table, TableDesc: get(r, 1)})
		ops = append(ops, []string{get(r, 2)})
	}
	for i := range result {
		result[i].Ops = mergeOps(ops[i])
	}
	return result
}

// GroupTablePrograms 把 (程序编号, 程序名称, 操作) 行按程序合并操作类别。
func GroupTablePrograms(rows [][]string, get func([]string, int) string) []TableProgRow {
	var result []TableProgRow
	idx := make(map[string]int)
	var ops [][]string
	for _, r := range rows {
		prog := get(r, 0)
		if i, ok := idx[prog]; ok {
			ops[i] = append(ops[i], get(r, 2))
			continue
		}
		idx[prog] = len(result)
		result = append(result, TableProgRow{Prog: prog, ProgName: get(r, 1)})
		ops = append(ops, []string{get(r, 2)})
	}
	for i := range result {
		result[i].Ops = mergeOps(ops[i])
	}
	return result
}

// QueryProgTables 查"这个程序用了哪些表"。
//
// GP 的这条关系不在目录里，而在 zr_file：由 p_get_zr 扫 4GL 源码生成，
// zr01 程序 / zr02 表 / zr03 操作码，同步时合并成 t_prog_table(prog,tab,ops)。
// 表说明取 t_table（可能没有——备份表与视图不进表档，此时留空）。
func (d *DB) QueryProgTables(code, lang string) ([]ProgTableRow, error) {
	rows, err := d.conn.Query(`
		SELECT t.tab, COALESCE(k.tab_cn, ''), COALESCE(t.ops, '')
		FROM t_prog_table t
		LEFT JOIN t_table k ON k.tab = t.tab
		WHERE t.prog = ?
		ORDER BY t.tab`, strings.ToLower(code))
	if err != nil {
		return nil, fmt.Errorf("query prog tables %s: %w", code, err)
	}
	defer rows.Close()

	var result []ProgTableRow
	for rows.Next() {
		var r ProgTableRow
		if err := rows.Scan(&r.Table, &r.TableDesc, &r.Ops); err != nil {
			return nil, fmt.Errorf("scan prog table: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryTablePrograms 反查"哪些程序在用这张表"（t_prog_table 按 tab）。
func (d *DB) QueryTablePrograms(table, lang string) ([]TableProgRow, error) {
	rows, err := d.conn.Query(`
		SELECT t.prog, COALESCE(p.name, ''), COALESCE(t.ops, '')
		FROM t_prog_table t
		LEFT JOIN t_prog p ON p.prog = t.prog
		WHERE t.tab = ?
		ORDER BY t.prog`, strings.ToLower(table))
	if err != nil {
		return nil, fmt.Errorf("query table programs %s: %w", table, err)
	}
	defer rows.Close()

	var result []TableProgRow
	for rows.Next() {
		var r TableProgRow
		if err := rows.Scan(&r.Prog, &r.ProgName, &r.Ops); err != nil {
			return nil, fmt.Errorf("scan table program: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// QueryProgInfo 按编号查程序登记信息。未收录返回 (nil, nil)。
//
// GP 的 gzza_t 对应 t_prog（同步自 gaz_file + zz_file + gal_file）：
//   - category ← zz_file.zz03（程序类别，M/I/U/P/R/Q/T/S/O）
//   - run_cmd  ← zz_file.zz08（系统运行指令）
//   - module   ← gal_file 里 gal03 == gal01 那行的 gal02（入口模块）
//   - name/name_tw ← gaz_file 的简体/繁体名称
//
// GP 没有"作业"这张独立的表：gaz_file 里一条程序就是一条作业，
// 所以 IsJob 恒为 true、JobProg 就是它自己（见 QueryProgJobs）。
func (d *DB) QueryProgInfo(code, lang string) (*ProgInfo, error) {
	info := &ProgInfo{Code: strings.ToLower(strings.TrimSpace(code))}
	row := d.conn.QueryRow(`
		SELECT COALESCE(p.category, ''), COALESCE(p.module, ''), COALESCE(p.run_cmd, ''),
		       COALESCE(p.memo, ''), `+progNameColumn(lang)+`,
		       (SELECT COUNT(*) FROM t_prog_table x WHERE x.prog = p.prog)
		FROM t_prog p
		WHERE p.prog = ?`, info.Code)
	var category, module, runCmd, memo, name string
	var tableCount int
	switch err := row.Scan(&category, &module, &runCmd, &memo, &name, &tableCount); err {
	case nil:
		info.IsProg = true
		info.Category, info.Module, info.RunCmd = category, module, runCmd
		info.Memo, info.TableCount = memo, tableCount
		info.Name = name
		info.CategoryName = CategoryName(category)
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, fmt.Errorf("query prog info %s: %w", code, err)
	}
	return info, nil
}

// QueryProgJobs 查"哪些作业用了这个程序"。
//
// GP 里 gzze_t 不存在，作业就是程序本身（gaz_file 的 gaz03 写着"……作業"），
// 所以这里恒返回 1 条：编号、名称、模块都取该程序自己。
func (d *DB) QueryProgJobs(code, lang string) ([]ProgJob, error) {
	info, err := d.QueryProgInfo(code, lang)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return []ProgJob{}, nil
	}
	return []ProgJob{{JobCode: info.Code, JobName: info.Name, Module: info.Module}}, nil
}

// QueryProgList 列出/搜索程序（--kw 按编号或名称过滤）。
// GP 里每个程序就是一条作业，所以"作业数"恒为 1。
func (d *DB) QueryProgList(lang, keyword string) ([]ProgListItem, error) {
	like := "%" + keyword + "%"
	rows, err := d.conn.Query(`
		SELECT p.prog, `+progNameColumn(lang)+`, COALESCE(p.category, ''), COALESCE(p.module, '')
		FROM t_prog p
		WHERE (? = '' OR lower(p.prog) LIKE lower(?) OR lower(COALESCE(p.name, '')) LIKE lower(?))
		ORDER BY p.prog`, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query prog list: %w", err)
	}
	defer rows.Close()

	var result []ProgListItem
	for rows.Next() {
		var p ProgListItem
		if err := rows.Scan(&p.Code, &p.Name, &p.Category, &p.Module); err != nil {
			return nil, fmt.Errorf("scan prog row: %w", err)
		}
		p.CategoryName = CategoryName(p.Category)
		p.JobCount = 1 // GP：程序即作业
		result = append(result, p)
	}
	return result, rows.Err()
}
