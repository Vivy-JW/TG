package db

import (
	"fmt"
	"strings"
)

// 画面元件译文(gae_file)。
//
// T100 的 .42s(编译字符串表)在 GP 上不存在 —— GP 的画面文字是运行期由
// lib/4gl/cl_ui_locale.4gl 从 gae_file 取的(该 lib 里三处 SELECT 都注释了
// #p_perlang)。p_perlang「畫面元件顯示多語言設定作業」维护的就是这张表:
//
//	gae01 程序/表单名 | gae02 元件名(页面/字段/按钮) | gae03 语系(0 繁 2 简 …)
//	gae04 译文 | gae10 异动日期 | gae11 客制标记 | gae12 行业别(std 为一般)
//
// 因此 .42s 的 token → 译文，在 GP 上等价于 (gae01, gae02, gae03) → gae04。
// 本地字典库把它投影成 t_ui_text，与同步来的其它 t_* 表同源。

// UIElementText 一条画面元件译文。
type UIElementText struct {
	Prog    string `json:"程序"`
	Element string `json:"元件"`
	Lang    string `json:"语系"`
	Text    string `json:"文字"`
	Cust    string `json:"客制"`
	Profile string `json:"行业别"`
}

// QueryUITexts 查某程序在某语系下的全部元件译文，按元件名排序。
//
// 同一元件在客制/行业别维度上可能有多行；命令层按「客制优先、非空优先」挑一条，
// 这里不做取舍，把原始行交给调用方。
func (d *DB) QueryUITexts(prog, lang string) ([]UIElementText, error) {
	prog = strings.ToLower(strings.TrimSpace(prog))
	lang = strings.TrimSpace(lang)
	if prog == "" || lang == "" {
		return nil, nil
	}
	rows, err := d.conn.Query(`SELECT COALESCE(prog,''), COALESCE(elem,''), COALESCE(lang,''),
		COALESCE(text,''), COALESCE(cust,''), COALESCE(profile,'')
		FROM t_ui_text WHERE prog = ? AND lang = ? ORDER BY elem, cust DESC, profile`, prog, lang)
	if err != nil {
		if IsMissingTable(err) {
			return nil, fmt.Errorf("本地字典库里没有 t_ui_text（这份库还没同步画面文字）：%w", err)
		}
		return nil, err
	}
	defer rows.Close()
	var out []UIElementText
	for rows.Next() {
		var r UIElementText
		if err := rows.Scan(&r.Prog, &r.Element, &r.Lang, &r.Text, &r.Cust, &r.Profile); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
