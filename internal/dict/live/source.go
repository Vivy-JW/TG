package live

// 远程查询实现:让 *Live 实现 db.Source —— 与本地 SQLite 镜像(db 包)同一组
// 语义查询与 DTO,直接连 GP 的 Oracle 库执行。
//
// TOP GP 的字典表与 T100 完全不同，这一层只认 GP 的那几张：
//
//	zta_file      表字典（按公司别 zta02 分行）
//	gat_file      表的中文名/模块/类型（gat02 语言 0=繁 2=简）
//	gaq_file      字段中文名（gaq02 语言 0/2）
//	all_*         Oracle 的字段/索引/约束视图（按 owner 过滤）
//	gaz_file      程序目录（gaz02 语言；gaz03 名称、gaz04 说明）
//	zz_file       程序类别 zz03 / 系统运行指令 zz08
//	gal_file      程序链接（gal03 == gal01 那行的 gal02 是入口模块）
//	zr_file       程序→表（p_get_zr 扫 4GL 源码生成的 zr01/zr02/zr03）
//	ze_file       错误消息（ze01 编号 / ze02 语言 / ze03 文本）
//
// 值经 QuoteLit 内联（erpdb simple protocol 无绑定参数），表名先过 ValidIdent。

import (
	"context"
	"fmt"
	"strings"

	"tt/internal/dict/db"
	"tt/internal/erpdb"
)

// q 执行单条只读 SQL(查询命令无需外部 ctx,沿用连接自身生命周期)。
func (l *Live) q(sql string) ([][]string, error) {
	_, rows, err := l.conn.Query(context.Background(), sql)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	return rows, nil
}

// get 取行内第 i 列,越界/为空返回空串(与本地全 TEXT 镜像的 NULL→"" 一致)。
func get(row []string, i int) string {
	if i < len(row) {
		return row[i]
	}
	return ""
}

// lit 把用户值转 SQL 字面量(单引号翻倍);表名等标识符先过 ValidIdent 防注入。
func lit(s string) string { return erpdb.QuoteLit(s) }

// tableLit 表/程序名(标识符形态)校验后转字面量;非法返回 ""(查不到,不报错)。
func tableLit(t string) string {
	if !erpdb.ValidIdent(t) {
		return "''"
	}
	return lit(t)
}

// kwWhere 生成大小写不敏感的关键字过滤(本地 LIKE 不区分大小写;远程统一 UPPER)。
// 返回空串表示不加过滤。
func kwWhere(idExpr, descExpr, kw string) string {
	if kw == "" {
		return ""
	}
	k := lit("%" + kw + "%")
	return fmt.Sprintf(" AND (UPPER(%s) LIKE UPPER(%s) OR UPPER(COALESCE(%s, '')) LIKE UPPER(%s))",
		idExpr, k, descExpr, k)
}

// progNameCol 按语言挑程序名称列：zh_TW 取繁体（gaz02='0'），其余取简体（gaz02='2'）。
func progNameCol(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh_tw", "zhtw", "tw", "0":
		return "0"
	default:
		return "2"
	}
}

// ---- 表字典 ----

// QueryTableMeta 表档:按公司别查 zta_file，未收录返回 (nil, nil)。
//
// 模块/类型与本地同步取同一口径：模块 = zta03（空才退 gat06）、类型 = zta09
// （空才退 gat07）。从前这里只取 gat_file 的 gat06/gat07，而 gat06 在库里
// 基本都是空的 —— 于是同一条 `dict r.t` 本地有模块、远程没有，看起来像"远程查不到"。
func (l *Live) QueryTableMeta(table string) (*db.TableMeta, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(z.zta01), COALESCE(g.gat03, ''),
		       COALESCE(NULLIF(TRIM(z.zta03), ''), g.gat06, ''),
		       COALESCE(NULLIF(TRIM(z.zta09), ''), g.gat07, ''),
		       COALESCE(z.zta07, '')
		  FROM zta_file z
		  LEFT JOIN gat_file g ON g.gat01 = z.zta01 AND g.gat02 = '2'
		 WHERE z.zta02 = %s AND lower(z.zta01) = lower(%s)`, lit(l.company), tableLit(table)))
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	r := rows[0]
	return &db.TableMeta{
		TableName: get(r, 0),
		TableDesc: get(r, 1),
		Module:    strings.ToUpper(strings.TrimSpace(get(r, 2))),
		TableType: strings.ToUpper(strings.TrimSpace(get(r, 3))),
		Kind:      strings.ToUpper(strings.TrimSpace(get(r, 4))),
	}, nil
}

// pkColumns 取一张表的主键字段(小写)。
func (l *Live) pkColumns(table string) map[string]bool {
	out := map[string]bool{}
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(cc.column_name)
		  FROM all_constraints c
		  JOIN all_cons_columns cc ON cc.owner = c.owner AND cc.constraint_name = c.constraint_name
		 WHERE upper(c.owner) = upper(%s) AND c.constraint_type = 'P'
		   AND lower(c.table_name) = lower(%s)`, lit(l.owner), tableLit(table)))
	if err != nil {
		return out
	}
	for _, r := range rows {
		out[strings.ToLower(get(r, 0))] = true
	}
	return out
}

// QueryTable 字段字典:all_tab_columns + gaq_file(字段中文名)。
func (l *Live) QueryTable(table string) ([]db.TableInfo, error) {
	pk := l.pkColumns(table)
	rows, err := l.q(fmt.Sprintf(`
		SELECT TO_CHAR(c.column_id), lower(c.column_name), COALESCE(g.gaq03, ''),
		       lower(c.data_type),
		       CASE WHEN c.data_length IS NULL THEN '' ELSE TO_CHAR(c.data_length) END,
		       CASE WHEN c.nullable = 'N' THEN 'Y' ELSE '' END
		  FROM all_tab_columns c
		  LEFT JOIN gaq_file g ON g.gaq01 = c.column_name AND g.gaq02 = '2'
		 WHERE upper(c.owner) = upper(%s) AND lower(c.table_name) = lower(%s)
		 ORDER BY c.column_id`, lit(l.owner), tableLit(table)))
	if err != nil {
		return nil, err
	}
	var result []db.TableInfo
	for _, r := range rows {
		info := db.TableInfo{
			Seq: get(r, 0), FieldName: get(r, 1), FieldDesc: get(r, 2),
			DataType: get(r, 3), Length: get(r, 4), Required: get(r, 5),
		}
		if pk[strings.ToLower(info.FieldName)] {
			info.IsPK = "PK"
		}
		result = append(result, info)
	}
	return result, nil
}

// QueryKeys 键值:all_constraints + all_cons_columns，一个约束一行。
func (l *Live) QueryKeys(table string) ([]db.KeyInfo, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(c.constraint_name), c.constraint_type,
		       LISTAGG(lower(cc.column_name), ',') WITHIN GROUP (ORDER BY cc.position)
		  FROM all_constraints c
		  JOIN all_cons_columns cc ON cc.owner = c.owner AND cc.constraint_name = c.constraint_name
		 WHERE upper(c.owner) = upper(%s) AND lower(c.table_name) = lower(%s)
		 GROUP BY lower(c.constraint_name), c.constraint_type
		 ORDER BY lower(c.constraint_name)`, lit(l.owner), tableLit(table)))
	if err != nil {
		return nil, err
	}
	var result []db.KeyInfo
	for _, r := range rows {
		result = append(result, db.KeyInfo{
			KeyName: get(r, 0), KeyType: db.KeyTypeLabel(get(r, 1)), KeyField: get(r, 2),
		})
	}
	return result, nil
}

// QueryIndexes 索引:all_indexes + all_ind_columns。
func (l *Live) QueryIndexes(table string) ([]db.IndexInfo, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(i.index_name), i.uniqueness,
		       LISTAGG(lower(ic.column_name), ',') WITHIN GROUP (ORDER BY ic.column_position)
		  FROM all_indexes i
		  JOIN all_ind_columns ic ON ic.index_owner = i.owner AND ic.index_name = i.index_name
		 WHERE upper(i.owner) = upper(%s) AND lower(i.table_name) = lower(%s)
		 GROUP BY lower(i.index_name), i.uniqueness
		 ORDER BY lower(i.index_name)`, lit(l.owner), tableLit(table)))
	if err != nil {
		return nil, err
	}
	var result []db.IndexInfo
	for _, r := range rows {
		result = append(result, db.IndexInfo{IndexName: get(r, 0), IndexType: get(r, 1), IndexField: get(r, 2)})
	}
	return result, nil
}

// QueryTableList 表清单:zta_file(公司别) + 字段数(按 owner 聚合的 all_tab_columns)。
func (l *Live) QueryTableList(lang, keyword string) ([]db.TableListItem, error) {
	owner := lit(l.owner)
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(z.zta01), COALESCE(g.gat03, ''), COALESCE(g.gat06, ''), COALESCE(g.gat07, ''),
		       COALESCE(c.n, 0)
		  FROM zta_file z
		  LEFT JOIN gat_file g ON g.gat01 = z.zta01 AND g.gat02 = '2'
		  LEFT JOIN (SELECT lower(table_name) AS tn, COUNT(*) AS n
		               FROM all_tab_columns WHERE upper(owner) = upper(%s)
		              GROUP BY table_name) c ON c.tn = lower(z.zta01)
		 WHERE z.zta02 = %s%s
		 ORDER BY z.zta01`,
		owner, lit(l.company), kwWhere("z.zta01", "g.gat03", keyword)))
	if err != nil {
		return nil, err
	}
	var result []db.TableListItem
	for _, r := range rows {
		item := db.TableListItem{
			TableName: get(r, 0), TableDesc: get(r, 1), Module: get(r, 2), TableType: get(r, 3),
		}
		if n, err := parseCount(get(r, 4)); err == nil {
			item.FieldCnt = n
		}
		result = append(result, item)
	}
	return result, nil
}

// parseCount 把 COUNT(*) 的字符串结果转成整数。
func parseCount(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("不是数字: %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// ---- 程序 / 作业 ----

// progName 取程序名称(按语言)。
func (l *Live) progName(prog, lang string) (name, memo string) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT COALESCE(gaz03, ''), COALESCE(gaz04, '')
		  FROM gaz_file
		 WHERE gaz02 = '%s' AND lower(gaz01) = lower(%s)`, progNameCol(lang), tableLit(prog)))
	if err != nil || len(rows) == 0 {
		return "", ""
	}
	return get(rows[0], 0), get(rows[0], 1)
}

// progModule 取入口模块(gal_file 里 gal03 == gal01 那行的 gal02)。
func (l *Live) progModule(prog string) string {
	rows, err := l.q(fmt.Sprintf(`
		SELECT COALESCE(MAX(gal02), '')
		  FROM gal_file
		 WHERE lower(gal01) = lower(%s) AND lower(gal03) = lower(gal01)`, tableLit(prog)))
	if err != nil || len(rows) == 0 {
		return ""
	}
	// gal_file 的模块名一律大写（实测 17,810 行全是 AWS/AXM/APM…），而本地 t_prog.module
	// 与 GP 磁盘目录（$TOP/cbm/4gl）一律小写。模块会进路径，出口统一转小写，
	// 免得拼出 $TOP/CBM/4gl 这种在区分大小写的文件系统上找不到的路径。
	return strings.ToLower(strings.TrimSpace(get(rows[0], 0)))
}

// QueryProgInfo 程序登记:gaz_file(名称/说明) + zz_file(类别/运行指令) + gal_file(模块)。
func (l *Live) QueryProgInfo(code, lang string) (*db.ProgInfo, error) {
	name, memo := l.progName(code, lang)
	rows, err := l.q(fmt.Sprintf(`
		SELECT COALESCE(zz03, ''), COALESCE(zz08, '')
		  FROM zz_file WHERE lower(zz01) = lower(%s)`, tableLit(code)))
	if err != nil {
		return nil, err
	}
	if name == "" && len(rows) == 0 {
		return nil, nil
	}
	info := &db.ProgInfo{Code: strings.ToLower(strings.TrimSpace(code)), IsProg: true, Name: name, Memo: memo}
	if len(rows) > 0 {
		info.Category = get(rows[0], 0)
		info.RunCmd = get(rows[0], 1)
		info.CategoryName = db.CategoryName(info.Category)
	}
	info.Module = l.progModule(code)
	// zr_file 是"程序 × 表 × 操作码"的行集合，同一张表可能有多行 → 按表去重。
	// 两侧都 TRIM：zr_file 是 p_get_zr 扫源码生成的，同一条程序名/表名可能带尾空格
	// （本地同步用 clean() 去空白后归并，这里不 TRIM 就会比本地少几张表 —— 实测
	// apmt420 本地 164 张、远程 159 张，差的 5 张就在带空格的那几行里）。
	if cnt, err := l.q(fmt.Sprintf(`SELECT COUNT(DISTINCT TRIM(lower(zr02))) FROM zr_file WHERE TRIM(lower(zr01)) = lower(%s)`, tableLit(code))); err == nil && len(cnt) > 0 {
		if n, err := parseCount(get(cnt[0], 0)); err == nil {
			info.TableCount = n
		}
	}
	return info, nil
}

// QueryProgJobs GP 里作业就是程序本身，恒返回 1 条。
func (l *Live) QueryProgJobs(code, lang string) ([]db.ProgJob, error) {
	info, err := l.QueryProgInfo(code, lang)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return []db.ProgJob{}, nil
	}
	return []db.ProgJob{{JobCode: info.Code, JobName: info.Name, Module: info.Module}}, nil
}

// QueryProgList 程序清单(名称、类别、模块)，--kw 按编号或名称过滤。
func (l *Live) QueryProgList(lang, keyword string) ([]db.ProgListItem, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(g.gaz01), COALESCE(g.gaz03, ''), COALESCE(z.zz03, ''), COALESCE(m.mod, '')
		  FROM gaz_file g
		  LEFT JOIN zz_file z ON lower(z.zz01) = lower(g.gaz01)
		  LEFT JOIN (SELECT lower(gal01) AS p, MAX(gal02) AS mod
		               FROM gal_file WHERE lower(gal03) = lower(gal01)
		              GROUP BY gal01) m ON m.p = lower(g.gaz01)
		 WHERE g.gaz02 = '%s'%s
		 ORDER BY g.gaz01`,
		progNameCol(lang), kwWhere("g.gaz01", "g.gaz03", keyword)))
	if err != nil {
		return nil, err
	}
	var result []db.ProgListItem
	for _, r := range rows {
		item := db.ProgListItem{
			Code: get(r, 0), Name: get(r, 1), Category: get(r, 2),
			Module: strings.ToLower(strings.TrimSpace(get(r, 3))), // 同上：出口统一小写
		}
		item.CategoryName = db.CategoryName(item.Category)
		item.JobCount = 1
		result = append(result, item)
	}
	return result, nil
}

// QueryProgTables 程序 → 表:zr_file（一个 (程序,表) 可能多行操作码，按表合并）。
func (l *Live) QueryProgTables(code, lang string) ([]db.ProgTableRow, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(TRIM(r.zr02)), COALESCE(g.gat03, ''), upper(r.zr03)
		  FROM zr_file r
		  LEFT JOIN gat_file g ON lower(TRIM(g.gat01)) = lower(TRIM(r.zr02)) AND g.gat02 = '2'
		 WHERE TRIM(lower(r.zr01)) = lower(%s)
		 ORDER BY 1`, tableLit(code)))
	if err != nil {
		return nil, err
	}
	return db.GroupProgTables(rows, get), nil
}

// QueryTablePrograms 反查哪些程序用这张表:zr_file + gaz_file(程序名)。
func (l *Live) QueryTablePrograms(table, lang string) ([]db.TableProgRow, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT lower(TRIM(r.zr01)), COALESCE(g.gaz03, ''), upper(r.zr03)
		  FROM zr_file r
		  LEFT JOIN gaz_file g ON lower(TRIM(g.gaz01)) = lower(TRIM(r.zr01)) AND g.gaz02 = '2'
		 WHERE lower(TRIM(r.zr02)) = lower(%s)
		 ORDER BY 1`, tableLit(table)))
	if err != nil {
		return nil, err
	}
	return db.GroupTablePrograms(rows, get), nil
}

// ---- 程序链接 ----

// QueryProgLinks 查 gal_file(程序 → 模块)。入口行是 gal03 == gal01,
// 也就是"程序名等于作业代号"的那一行 —— 不能只取 MAX(gal02) 压成一行,
// 否则会选中子程序所在的模块。
func (l *Live) QueryProgLinks(prog string) ([]db.ProgLinkRow, error) {
	prog = strings.TrimSpace(prog)
	if prog == "" {
		return nil, nil
	}
	rows, err := l.q(fmt.Sprintf(`
		SELECT COALESCE(gal01, ''), COALESCE(gal02, ''), COALESCE(gal03, ''), COALESCE(gal04, '')
		  FROM gal_file
		 WHERE UPPER(gal01) = UPPER(%s)
		 ORDER BY CASE WHEN UPPER(gal03) = UPPER(gal01) THEN 0 ELSE 1 END,
		          gal02, gal03`, lit(prog)))
	if err != nil {
		return nil, err
	}
	out := make([]db.ProgLinkRow, 0, len(rows))
	for _, r := range rows {
		head := strings.TrimSpace(get(r, 0))
		module := strings.ToLower(strings.TrimSpace(get(r, 1)))
		linked := strings.TrimSpace(get(r, 2))
		flag := strings.TrimSpace(get(r, 3))
		out = append(out, db.ProgLinkRow{
			Module:     module,
			LinkedProg: linked,
			LinkFlag:   flag,
			Entry:      strings.EqualFold(linked, head),
		})
	}
	return out, nil
}

// ---- 消息 ----

// msgWhereGP 拼 ze_file 的 WHERE(编号/文本/语言)。
func (l *Live) msgWhereGP(q db.MsgQuery) string {
	var conds []string
	if len(q.Codes) > 0 {
		ors := make([]string, 0, len(q.Codes))
		for _, c := range q.Codes {
			if strings.ContainsAny(c, "*%") {
				ors = append(ors, fmt.Sprintf("UPPER(ze01) LIKE UPPER(%s)", lit(db.MsgLike(c))))
			} else {
				ors = append(ors, fmt.Sprintf("UPPER(ze01) = UPPER(%s)", lit(c)))
			}
		}
		conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	}
	if q.Text != "" {
		conds = append(conds, fmt.Sprintf("UPPER(ze03) LIKE UPPER(%s)", lit(db.MsgLike(q.Text))))
	}
	if q.Lang != "" {
		conds = append(conds, fmt.Sprintf("ze02 = %s", lit(db.LangCode(q.Lang))))
	}
	if len(conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conds, " AND ")
}

// QueryMsgs 查错误消息(ze_file)。GP 只有编号/语言/文本/日期，其余条件返回空集。
func (l *Live) QueryMsgs(q db.MsgQuery) ([]db.MsgRow, error) {
	if !q.HasFilter() {
		return nil, fmt.Errorf("消息查询至少需要一个条件(编号/语句)")
	}
	if len(q.Types) > 0 || len(q.Status) > 0 || len(q.Progs) > 0 {
		return []db.MsgRow{}, nil
	}
	rows, err := l.q(`SELECT lower(ze01), COALESCE(ze02, ''), COALESCE(ze03, '') FROM ze_file` +
		l.msgWhereGP(q) + ` ORDER BY ze01, ze02`)
	if err != nil {
		return nil, err
	}
	var result []db.MsgRow
	for _, r := range rows {
		result = append(result, db.MsgRow{Code: get(r, 0), Lang: get(r, 1), Text: get(r, 2)})
	}
	return result, nil
}

// QueryMsgLangs 同条件(去掉语言)下该编号有哪些语言行。
func (l *Live) QueryMsgLangs(q db.MsgQuery) ([]string, error) {
	if !q.HasFilter() {
		return nil, fmt.Errorf("消息查询至少需要一个条件(编号/语句)")
	}
	q.Lang = ""
	rows, err := l.q(`SELECT DISTINCT COALESCE(ze02, '') AS lang FROM ze_file` +
		l.msgWhereGP(q) + ` ORDER BY lang`)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range rows {
		if v := get(r, 0); v != "" {
			out = append(out, v)
		}
	}
	return out, nil
}

// ---- 开窗定义 r.q：GP 用 gab_file + gac_file ----
//
// 与 T100 的差别：GP 没有整段 SQL 模板（dzca003），也没有 arg1~9 的参数档（dzcb_t）。
// 这里只负责把定义读出来；SQL 由命令层按 cl_create_qry() 的规则现拼
// （见 internal/dict/qry.go 的 QryDef.BuildSQL），所以 WinHeaderRow.SQL 留空。

// winLangCode 语系码：0 = 繁中，2 = 简中（gae_file.gae03 / gaq_file.gaq02 同用）。
func winLangCode(lang string) string { return progNameCol(lang) }

// winTitleExpr 是标题的取法：先看 gae_file 的 wintitle 行，没有就退回
// lib-213 的消息文案 + 第一张表的 gat03 —— 与 cl_create_qry.4gl 的垫背逻辑一致。
// codeExpr / custExpr 是外层 gab_file 的代号与客制旗标表达式。
func winTitleExpr(codeExpr, custExpr, lang string) string {
	word := "查询"
	if winLangCode(lang) == "0" {
		word = "查詢"
	}
	return fmt.Sprintf(`TRIM(COALESCE(NULLIF(TRIM(t.gae04), ''),
		%s || ' ' || (SELECT MAX(gf.gat03) KEEP (DENSE_RANK FIRST ORDER BY gc.gac02)
		                FROM gac_file gc
		                JOIN gat_file gf ON TRIM(gf.gat01) = TRIM(gc.gac05) AND gf.gat02 = %s
		               WHERE TRIM(gc.gac01) = %s
		                 AND COALESCE(gc.gac12, ' ') = %s
		                 AND gc.gac05 IS NOT NULL AND TRIM(gc.gac05) <> 'formonly')))`,
		lit(word), lit(winLangCode(lang)), codeExpr, custExpr)
}

// QueryWinList 列出全部开窗定义：代号 / 客制旗标 / 标题 / HardCode / 栏位数。
func (l *Live) QueryWinList(lang, keyword string) ([]db.WinListRow, error) {
	rows, err := l.q(fmt.Sprintf(`
		SELECT TRIM(b.gab01), TRIM(COALESCE(b.gab11, '')),
		       %s, TRIM(COALESCE(b.gab07, '')),
		       (SELECT COUNT(*) FROM gac_file c
		         WHERE TRIM(c.gac01) = TRIM(b.gab01)
		           AND COALESCE(c.gac12, ' ') = COALESCE(b.gab11, ' ')
		           AND c.gac02 <> -1)
		  FROM gab_file b
		  LEFT JOIN gae_file t
		         ON TRIM(t.gae01) = TRIM(b.gab01) AND t.gae02 = 'wintitle'
		        AND t.gae03 = %s AND COALESCE(t.gae11, ' ') = COALESCE(b.gab11, ' ')
		 WHERE 1=1%s
		 ORDER BY TRIM(b.gab01)`,
		winTitleExpr("TRIM(b.gab01)", "COALESCE(b.gab11, ' ')", lang),
		lit(winLangCode(lang)), kwWhere("b.gab01", "t.gae04", keyword)))
	if err != nil {
		return nil, err
	}
	var out []db.WinListRow
	for _, r := range rows {
		out = append(out, db.WinListRow{
			ID:       get(r, 0),
			Cust:     get(r, 1),
			Desc:     get(r, 2),
			HardCode: get(r, 3),
			ColCount: get(r, 4),
		})
	}
	return out, nil
}

// QueryWinHeaders 取一条开窗的单头（标准/客制各一行）。
func (l *Live) QueryWinHeaders(id, lang string) ([]db.WinHeaderRow, error) {
	code := strings.TrimSpace(id)
	if code == "" {
		return []db.WinHeaderRow{}, nil
	}
	rows, err := l.q(fmt.Sprintf(`
		SELECT TRIM(b.gab01), TRIM(COALESCE(b.gab11, '')),
		       TRIM(COALESCE(b.gab02, '')), TRIM(COALESCE(b.gab03, '')),
		       TRIM(COALESCE(b.gab04, '')), TRIM(COALESCE(b.gab05, '')),
		       TRIM(COALESCE(b.gab06, '')), TRIM(COALESCE(b.gab07, '')),
		       TRIM(COALESCE(b.gab08, '')), %s
		  FROM gab_file b
		  LEFT JOIN gae_file t
		         ON TRIM(t.gae01) = TRIM(b.gab01) AND t.gae02 = 'wintitle'
		        AND t.gae03 = %s AND COALESCE(t.gae11, ' ') = COALESCE(b.gab11, ' ')
		 WHERE UPPER(TRIM(b.gab01)) = UPPER(%s)
		 ORDER BY TRIM(COALESCE(b.gab11, ''))`,
		winTitleExpr("TRIM(b.gab01)", "COALESCE(b.gab11, ' ')", lang),
		lit(winLangCode(lang)), lit(code)))
	if err != nil {
		return nil, err
	}
	var out []db.WinHeaderRow
	for _, r := range rows {
		out = append(out, db.WinHeaderRow{
			ID:          get(r, 0),
			Cust:        get(r, 1),
			InputWhere:  get(r, 2),
			JoinTables:  get(r, 3),
			Distinct:    get(r, 4),
			ConstrWhere: get(r, 5),
			Memo:        get(r, 6),
			HardCode:    get(r, 7),
			Unlocked:    get(r, 8),
			Desc:        get(r, 9),
		})
	}
	return out, nil
}

// QueryWinCols 取一条开窗的全部显现栏位：gac_file + gaq_file 显示名 + 列类型。
func (l *Live) QueryWinCols(id, lang string) ([]db.WinColRow, error) {
	code := strings.TrimSpace(id)
	if code == "" {
		return []db.WinColRow{}, nil
	}
	rows, err := l.q(fmt.Sprintf(`
		SELECT TRIM(COALESCE(g.gac12, '')), TRIM(TO_CHAR(g.gac02)),
		       TRIM(COALESCE(g.gac06, '')), TRIM(COALESCE(g.gac05, '')),
		       TRIM(COALESCE(TO_CHAR(g.gac09), '')), TRIM(COALESCE(g.gac10, '')),
		       TRIM(COALESCE(g.gac07, '')), TRIM(COALESCE(g.gac11, '')),
		       TRIM(COALESCE(g.gac13, '')), TRIM(COALESCE(g.gac15, '')),
		       TRIM(COALESCE(q.gaq03, '')), TRIM(COALESCE(c.data_type, ''))
		  FROM gac_file g
		  LEFT JOIN gaq_file q
		         ON TRIM(q.gaq01) = TRIM(g.gac06) AND q.gaq02 = %s
		  LEFT JOIN all_tab_columns c
		         ON c.owner = %s AND c.table_name = UPPER(TRIM(g.gac05))
		        AND c.column_name = UPPER(TRIM(g.gac06))
		 WHERE UPPER(TRIM(g.gac01)) = UPPER(%s)
		 ORDER BY g.gac02`,
		lit(winLangCode(lang)), lit(l.owner), lit(code)))
	if err != nil {
		return nil, err
	}
	var out []db.WinColRow
	for _, r := range rows {
		out = append(out, db.WinColRow{
			Cust:       get(r, 0),
			Seq:        get(r, 1),
			Field:      get(r, 2),
			Table:      get(r, 3),
			Width:      get(r, 4),
			Return:     get(r, 5),
			Type:       get(r, 6),
			Outer:      get(r, 7),
			Constraint: get(r, 8),
			ConnectTo:  get(r, 9),
			Label:      get(r, 10),
			DataType:   get(r, 11),
		})
	}
	return out, nil
}
