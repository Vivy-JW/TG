package db

import (
	"errors"
	"fmt"
	"strings"
)

// 系统消息档查询(gzze_t):T100 所有提示/报错消息由作业 azzi920 维护,
// 运行时 cl_err/cl_getmsg 按 (编号, 语言) 精确取用。消息编号(gzze001)形如
// std-00001 / azz-00041 / lib-xxxxx / -263(负整数 SQLCODE),语言(gzze002)为
// zh_CN/zh_TW/en_US 等。作业名称辅助表 gzzal_t 未同步时静默降级(名称留空)。
//
// 查询条件照 azzi920 的查询画面来:编号(gzze001)、信息语句(gzze003)、
// 语言别(gzze002)三栏同时成立。只有一处刻意不同 —— 画面里编号栏默认被
// cl_ap_code_fuzzyquery 加上 *…* 变成子串查询,这里反过来:**编号默认精确,
// 写了 * 才当通配**。理由是 CLI 的常见用途之一是查 SQLCODE:模糊匹配 -263
// 会把 -1263 / -22263 / -26300…-26312 一并捞回来(实测 17 条以上),精确才是
// 想要的。信息语句两边一致,默认子串。

// MsgRow 一条消息档记录(编号+语言唯一)。
type MsgRow struct {
	Code     string `json:"编号"`
	Lang     string `json:"语言"`
	Text     string `json:"文本"`   // gzze003 消息文本
	Action   string `json:"建议处理"` // gzze004 建议处理方式
	ExecProg string `json:"建议作业"` // gzze005 建议执行作业编号(:EXEPROG=无)
	ProgName string `json:"作业名称"` // gzzal003(该语言名称;gzzal_t 未同步时为空)
	Detail   string `json:"技术细节"` // gzze006 程式人员详细讯息
	TypeCode string `json:"类型码"`  // gzze007 讯息类型: 0警告/1错误/2资讯 (SCC 106)
	ForceWin string `json:"强制开窗"` // gzze008 Y/N
	Status   string `json:"状态"`   // gzzestus Y=启用/N=停用
}

// MsgQuery 系统消息档的查询条件(全部同时成立;零值字段 = 不设该条件)。
//
// 语言别(Lang)不在"筛行"之列:它决定看哪个语言的份,与"筛哪些编号"是两回事
// —— 见 HasFilter。其余五个都是筛子。
type MsgQuery struct {
	// Codes 是 gzze001 编号条件,多个之间是 OR;不含 * / % 的按精确匹配,
	// 含的按通配模式(经 MsgLike)。
	Codes []string
	// Text 是 gzze003 信息语句条件,子串匹配(经 MsgLike)。
	Text string
	// Lang 是 gzze002 语言别条件,精确匹配。
	Lang string
	// Types 是 gzze007 信息类型条件(SCC 106: 0=警告 1=错误 2=资讯),多个之间是 OR。
	Types []string
	// Status 是 gzzestus 状态码条件(Y=启用 N=停用),多个之间是 OR。
	Status []string
	// Progs 是 gzze005 建议运行作业条件(精确匹配,多个之间是 OR)。
	// "哪些消息建议跑这个作业"就靠它 —— 反过来查 gzze005 是这里独有的用法。
	Progs []string
}

// HasFilter 是否给了筛行的条件(编号/语句/类型/状态/建议作业)。
// 语言别不算:只给 --lang 等于没筛,那是"把整张表换个语言看一遍"。
func (q MsgQuery) HasFilter() bool {
	return len(q.Codes) > 0 || q.Text != "" || len(q.Types) > 0 || len(q.Status) > 0 || len(q.Progs) > 0
}

// MsgLike 把用户输入转成 LIKE 模式:含 * 或 % 视为自带通配(原样使用,* 换成 %),
// 否则前后补 % 走子串匹配。对应 azzi920 里 cl_ap_code_fuzzyquery 的
// "使用者有自己要的查詢條件" 那一支(那里认的是 = * <> | :,CLI 只留 * 与 %)。
func MsgLike(term string) string {
	if strings.ContainsAny(term, "*%") {
		return strings.ReplaceAll(term, "*", "%")
	}
	return "%" + term + "%"
}

// msgWhere 拼 MsgQuery 的 WHERE 子句与绑定参数。一个条件都没给时返回空串。
// 编号两端 UPPER:工具里其它查询一律大小写不敏感,这里跟上。
//
// GP 的 ze_file 只有四个列(ze01 编号 / ze02 语言 / ze03 文本 / ze04 日期),
// 所以只有编号、文本、语言三个条件能落到 SQL 上;类型/状态/建议作业
// 在 QueryMsgs 里提前返回空集,不会走到这里。
func msgWhere(q MsgQuery) (string, []any) {
	var conds []string
	var args []any

	if len(q.Codes) > 0 {
		ors := make([]string, 0, len(q.Codes))
		for _, c := range q.Codes {
			if strings.ContainsAny(c, "*%") {
				ors = append(ors, "UPPER(COALESCE(code, '')) LIKE UPPER(?)")
				args = append(args, MsgLike(c))
			} else {
				ors = append(ors, "UPPER(COALESCE(code, '')) = UPPER(?)")
				args = append(args, c)
			}
		}
		conds = append(conds, "("+strings.Join(ors, " OR ")+")")
	}
	if q.Text != "" {
		conds = append(conds, "UPPER(COALESCE(text, '')) LIKE UPPER(?)")
		args = append(args, MsgLike(q.Text))
	}
	if q.Lang != "" {
		conds = append(conds, "lang = ?")
		args = append(args, LangCode(q.Lang))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// errMsgNoCond 空条件查询的拒绝理由 —— 消息档有 3 万条,不设条件地全捞没有意义,
// 命令层会先给出更具体的用法提示,这里只兜住将来别的调用方。
var errMsgNoCond = errors.New("消息查询至少需要一个条件(编号/语句/类型/状态/建议作业)")

// MsgResult 一次消息查询的结果。
//
// Rows 是命中的 (编号, 语言) 行；Langs 只在**没有命中**时填 —— 是同一组条件
// （只是不看语言）下实际存在的语言，用来回答"这条消息是不是只有别的语系"。
// 命中时 Langs 留空：那时命令层直接列详情，不需要再花一次 DISTINCT 去数语言。
type MsgResult struct {
	Rows  []MsgRow
	Langs []string
}

// QueryMsgs 按条件查消息:命中几行就是几条 (编号, 语言) 记录,按编号排序。
//
// 源是 t_msg(同步自 ze_file)。GP 的 ze_file 没有"讯息类型/启用状态/建议作业"
// 这三列,所以带这三类条件时直接返回空集 —— 空是准确答案,不是查询失败。
func (d *DB) QueryMsgs(q MsgQuery) (MsgResult, error) {
	if !q.HasFilter() {
		return MsgResult{}, errMsgNoCond
	}
	if len(q.Types) > 0 || len(q.Status) > 0 || len(q.Progs) > 0 {
		res := MsgResult{Rows: []MsgRow{}}
		if langs, err := d.msgLangs(q); err == nil {
			res.Langs = langs
		}
		return res, nil
	}
	where, args := msgWhere(q)
	rows, err := d.conn.Query(`
		SELECT COALESCE(code, ''), COALESCE(lang, ''), COALESCE(text, '')
		FROM t_msg`+where+`
		ORDER BY code, lang`, args...)
	if err != nil {
		return MsgResult{}, fmt.Errorf("query msg: %w", err)
	}
	defer rows.Close()

	var result []MsgRow
	for rows.Next() {
		var r MsgRow
		if err := rows.Scan(&r.Code, &r.Lang, &r.Text); err != nil {
			return MsgResult{}, fmt.Errorf("scan msg row: %w", err)
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return MsgResult{}, err
	}
	res := MsgResult{Rows: result}
	if len(result) == 0 {
		if langs, err := d.msgLangs(q); err == nil {
			res.Langs = langs
		}
	}
	return res, nil
}

// msgLangs 按同样的条件(只去掉语言)返回有记录的语言清单。
// 用途:命令层在"该编号没有 --lang 那一行"时列出可用语言。
func (d *DB) msgLangs(q MsgQuery) ([]string, error) {
	if !q.HasFilter() {
		return nil, errMsgNoCond
	}
	q.Lang = "" // 这一步问的就是"除当前语言外还有哪些"
	where, args := msgWhere(q)
	rows, err := d.conn.Query(`SELECT DISTINCT COALESCE(lang, '') AS lang FROM t_msg`+where+` ORDER BY lang`, args...)
	if err != nil {
		return nil, fmt.Errorf("query msg langs: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var lang string
		if err := rows.Scan(&lang); err != nil {
			return nil, fmt.Errorf("scan msg lang: %w", err)
		}
		if lang != "" {
			out = append(out, lang)
		}
	}
	return out, rows.Err()
}
