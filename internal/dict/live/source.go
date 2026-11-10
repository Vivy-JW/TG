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
	return get(rows[0], 0)
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
			Code: get(r, 0), Name: get(r, 1), Category: get(r, 2), Module: get(r, 3),
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

// ---- GP 没有的字典族：返回空集（命令层据此说“GP 没有这类数据”） ----

func (l *Live) QuerySubProgInfo(code, lang string) (*db.SubProgInfo, error) { return nil, nil }
func (l *Live) QuerySubProgList(lang, keyword string) ([]db.SubProgListItem, error) {
	return []db.SubProgListItem{}, nil
}
func (l *Live) QueryCheckList(lang, keyword string) ([]db.CheckListRow, error) {
	return []db.CheckListRow{}, nil
}
func (l *Live) QueryCheckHeaders(id, lang string) ([]db.CheckHeaderRow, error) {
	return []db.CheckHeaderRow{}, nil
}
func (l *Live) QueryCheckParams(id, lang string) ([]db.CheckParamRow, error) {
	return []db.CheckParamRow{}, nil
}
func (l *Live) QueryCheckConds(id string) ([]db.CheckCondRow, error) {
	return []db.CheckCondRow{}, nil
}
func (l *Live) QuerySpecRows(tableName, lang string) ([]db.SpecRow, error) {
	return []db.SpecRow{}, nil
}
func (l *Live) QuerySccList(lang, keyword string) ([]db.SccListRow, error) {
	return []db.SccListRow{}, nil
}
func (l *Live) QuerySccHeader(id, lang string) (*db.SccHeaderRow, error) { return nil, nil }
func (l *Live) QuerySccValues(id, lang string) ([]db.SccValueRow, error) {
	return []db.SccValueRow{}, nil
}
func (l *Live) QueryACCMeta(input, lang string) (*db.ACCMeta, error) { return nil, nil }
func (l *Live) QueryACCValues(category string, ent int, lang string, includeInactive bool) ([]db.ACCValue, error) {
	return []db.ACCValue{}, nil
}
func (l *Live) QueryWinList(lang, keyword string) ([]db.WinListRow, error) {
	return []db.WinListRow{}, nil
}
func (l *Live) QueryWinHeaders(id, lang string) ([]db.WinHeaderRow, error) {
	return []db.WinHeaderRow{}, nil
}
func (l *Live) QueryWinParams(id, lang string) ([]db.WinParamRow, error) {
	return []db.WinParamRow{}, nil
}
func (l *Live) QueryWinCols(id string) ([]db.WinColRow, error) { return []db.WinColRow{}, nil }
func (l *Live) QueryParam(code string) ([]db.ParamDefRow, error) {
	return []db.ParamDefRow{}, nil
}
func (l *Live) QueryDocTypes(code, lang string) ([]db.DocTypeRow, error) {
	return []db.DocTypeRow{}, nil
}
