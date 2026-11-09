package db

import (
	"database/sql"
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
	Name      string `json:"程序名称"`   // gzzal003(指定语言)
	ShortName string `json:"程序简称"`   // gzzal005
	Category  string `json:"程序类别"`   // gzza002
	Module    string `json:"归属模块"`   // gzza003
	Cust      string `json:"客制"`     // gzza011
	RefMain   string `json:"引用主程序"`  // gzza008
	RunCmd    string `json:"系统运行指令"` // gzza004
	Status    string `json:"状态码"`    // gzzastus
	IsJob     bool   `json:"是作业"`
	JobProg   string `json:"作业挂的程序"` // gzzz002(该编号作为作业登记时)
	// 下面两项只在“这个编号本身是作业”时有值：同名作业靠它们区分。
	JobParamGrp string `json:"应用参数组"` // gzzz003
	JobExtra    string `json:"额外参数"`  // gzzz004
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
	Module   string `json:"归属模块"`
	Cust     string `json:"客制"`
	JobCount int    `json:"作业数"`
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

// QuerySubProgInfo 按编号查子程序/元件/库的登记;未收录返回 (nil, nil)。
func (d *DB) QuerySubProgInfo(code, lang string) (*SubProgInfo, error) {
	var p SubProgInfo
	err := d.conn.QueryRow(`
		SELECT g.gzde001, COALESCE(l.gzdel003, ''), COALESCE(g.gzde003, ''),
		       COALESCE(g.gzde002, ''), COALESCE(g.gzde008, ''), COALESCE(g.gzde009, ''),
		       COALESCE(g.gzde005, ''), COALESCE(g.gzde006, ''), COALESCE(g.gzdestus, '')
		FROM gzde_t g
		LEFT JOIN gzdel_t l ON l.gzdel001 = g.gzde001 AND l.gzdel002 = ?
		WHERE g.gzde001 = ?`, lang, code).
		Scan(&p.Code, &p.Name, &p.Category, &p.Module, &p.Cust, &p.Industry,
			&p.ProgCat, &p.GenType, &p.Status)
	switch err {
	case nil:
		return &p, nil
	case sql.ErrNoRows:
		return nil, nil
	default:
		return nil, fmt.Errorf("query subprog info %s: %w", code, err)
	}
}

// QuerySubProgList 列出/搜索子程序与元件(--kw 按编号或说明过滤)。
func (d *DB) QuerySubProgList(lang, keyword string) ([]SubProgListItem, error) {
	like := "%" + keyword + "%"
	rows, err := d.conn.Query(`
		SELECT g.gzde001, COALESCE(l.gzdel003, ''), COALESCE(g.gzde003, ''),
		       COALESCE(g.gzde002, ''), COALESCE(g.gzde008, '')
		FROM gzde_t g
		LEFT JOIN gzdel_t l ON l.gzdel001 = g.gzde001 AND l.gzdel002 = ?
		WHERE (? = '' OR g.gzde001 LIKE ? OR COALESCE(l.gzdel003, '') LIKE ?)
		ORDER BY g.gzde001`, lang, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query subprog list: %w", err)
	}
	defer rows.Close()

	var result []SubProgListItem
	for rows.Next() {
		var p SubProgListItem
		if err := rows.Scan(&p.Code, &p.Name, &p.Category, &p.Module, &p.Cust); err != nil {
			return nil, fmt.Errorf("scan subprog row: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
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

// QueryProgTables 查"这个程序用了哪些表"(gzdg_t 按 gzdg001)。表名取 dzeal_t(指定语言)。
func (d *DB) QueryProgTables(code, lang string) ([]ProgTableRow, error) {
	rows, err := d.conn.Query(`
		SELECT t.gzdg002, COALESCE(al.dzeal003, ''), COALESCE(t.gzdg003, '')
		FROM gzdg_t t
		LEFT JOIN dzeal_t al ON al.dzeal001 = t.gzdg002 AND al.dzeal002 = ?
		WHERE t.gzdg001 = ?
		ORDER BY t.gzdg002, t.gzdg003`, lang, code)
	if err != nil {
		return nil, fmt.Errorf("query prog tables %s: %w", code, err)
	}
	defer rows.Close()

	var raw [][]string
	for rows.Next() {
		var table, desc, op string
		if err := rows.Scan(&table, &desc, &op); err != nil {
			return nil, fmt.Errorf("scan prog table: %w", err)
		}
		raw = append(raw, []string{table, desc, op})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return GroupProgTables(raw, GetCol), nil
}

// QueryTablePrograms 反查"哪些程序在用这张表"(gzdg_t 按 gzdg002)。程序名取 gzzal_t(指定语言)。
func (d *DB) QueryTablePrograms(table, lang string) ([]TableProgRow, error) {
	rows, err := d.conn.Query(`
		SELECT t.gzdg001, COALESCE(pl.gzzal003, ''), COALESCE(t.gzdg003, '')
		FROM gzdg_t t
		LEFT JOIN gzzal_t pl ON pl.gzzal001 = t.gzdg001 AND pl.gzzal002 = ?
		WHERE t.gzdg002 = ?
		ORDER BY t.gzdg001, t.gzdg003`, lang, table)
	if err != nil {
		return nil, fmt.Errorf("query table programs %s: %w", table, err)
	}
	defer rows.Close()

	var raw [][]string
	for rows.Next() {
		var prog, name, op string
		if err := rows.Scan(&prog, &name, &op); err != nil {
			return nil, fmt.Errorf("scan table program: %w", err)
		}
		raw = append(raw, []string{prog, name, op})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return GroupTablePrograms(raw, GetCol), nil
}

// QueryProgInfo 按编号查程序登记信息(附"该编号是否也是作业、挂的哪个程序")。
// 程序与作业都未收录时返回 (nil, nil)。
func (d *DB) QueryProgInfo(code, lang string) (*ProgInfo, error) {
	info := &ProgInfo{Code: code}

	row := d.conn.QueryRow(`
		SELECT COALESCE(a.gzza002, ''), COALESCE(a.gzza003, ''), COALESCE(a.gzza004, ''),
		       COALESCE(a.gzza008, ''), COALESCE(a.gzza011, ''), COALESCE(a.gzzastus, ''),
		       COALESCE(l.gzzal003, ''), COALESCE(l.gzzal005, '')
		FROM gzza_t a
		LEFT JOIN gzzal_t l ON l.gzzal001 = a.gzza001 AND l.gzzal002 = ?
		WHERE a.gzza001 = ?`, lang, code)
	var category, module, runCmd, refMain, cust, status, name, shortName string
	switch err := row.Scan(&category, &module, &runCmd, &refMain, &cust, &status, &name, &shortName); err {
	case nil:
		info.IsProg = true
		info.Category, info.Module, info.RunCmd = category, module, runCmd
		info.RefMain, info.Cust, info.Status = refMain, cust, status
		info.Name, info.ShortName = name, shortName
	case sql.ErrNoRows:
		// 不是程序,再看看是不是作业
	default:
		return nil, fmt.Errorf("query prog info %s: %w", code, err)
	}

	var jobProg, jobParam, jobExtra string
	if err := d.conn.QueryRow(
		`SELECT COALESCE(gzzz002, ''), COALESCE(gzzz003, ''), COALESCE(gzzz004, '')
		 FROM gzzz_t WHERE gzzz001 = ?`, code).Scan(&jobProg, &jobParam, &jobExtra); err == nil {
		info.IsJob = true
		info.JobProg, info.JobParamGrp, info.JobExtra = jobProg, jobParam, jobExtra
	} else if err != sql.ErrNoRows {
		return nil, fmt.Errorf("query job info %s: %w", code, err)
	}

	if !info.IsProg && !info.IsJob {
		return nil, nil
	}
	return info, nil
}

// QueryProgJobs 查"哪些作业用了这个程序"(gzzz_t.gzzz002 = code)。
// 作业名称取它所挂程序的名称(gzzal_t),与 azzi910 的取法一致。
func (d *DB) QueryProgJobs(code, lang string) ([]ProgJob, error) {
	rows, err := d.conn.Query(`
		SELECT z.gzzz001,
		       COALESCE(l.gzzal003, ''),
		       COALESCE(z.gzzz005, ''),
		       COALESCE(z.gzzz003, ''),
		       COALESCE(z.gzzz004, ''),
		       COALESCE(k.gzzk003, ''),
		       COALESCE(z.gzzz006, ''),
		       COALESCE(z.gzzzstus, '')
		FROM gzzz_t z
		LEFT JOIN gzzal_t l ON l.gzzal001 = z.gzzz002 AND l.gzzal002 = ?
		LEFT JOIN gzzk_t k ON k.gzzk001 = z.gzzz002 AND k.gzzk002 = z.gzzz003
		WHERE z.gzzz002 = ?
		ORDER BY z.gzzz001`, lang, code)
	if err != nil {
		return nil, fmt.Errorf("query prog jobs %s: %w", code, err)
	}
	defer rows.Close()

	var result []ProgJob
	for rows.Next() {
		var j ProgJob
		if err := rows.Scan(&j.JobCode, &j.JobName, &j.Module, &j.ParamGrp, &j.Extra,
			&j.ParamDesc, &j.DocType, &j.Status); err != nil {
			return nil, fmt.Errorf("scan prog job: %w", err)
		}
		result = append(result, j)
	}
	return result, rows.Err()
}

// QueryProgList 列出/搜索程序(--kw 按程序编号或程序名称过滤),附各自的作业数。
func (d *DB) QueryProgList(lang, keyword string) ([]ProgListItem, error) {
	like := "%" + keyword + "%"
	rows, err := d.conn.Query(`
		SELECT a.gzza001,
		       COALESCE(l.gzzal003, ''),
		       COALESCE(a.gzza002, ''),
		       COALESCE(a.gzza003, ''),
		       COALESCE(a.gzza011, ''),
		       (SELECT COUNT(*) FROM gzzz_t z WHERE z.gzzz002 = a.gzza001)
		FROM gzza_t a
		LEFT JOIN gzzal_t l ON l.gzzal001 = a.gzza001 AND l.gzzal002 = ?
		WHERE (? = '' OR a.gzza001 LIKE ? OR COALESCE(l.gzzal003, '') LIKE ?)
		ORDER BY a.gzza001`, lang, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query prog list: %w", err)
	}
	defer rows.Close()

	var result []ProgListItem
	for rows.Next() {
		var p ProgListItem
		if err := rows.Scan(&p.Code, &p.Name, &p.Category, &p.Module, &p.Cust, &p.JobCount); err != nil {
			return nil, fmt.Errorf("scan prog row: %w", err)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
