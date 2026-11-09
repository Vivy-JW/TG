package db

import "fmt"

// SpecRow holds one field's spec row (dzep_t LEFT JOIN dzeb_t/dzebl_t).
// It records how the screen designer builds a form for this field:
// widget type, SCC code for dropdowns, format, required flag, etc.
type SpecRow struct {
	Seq        string // dzeb021 (显示顺序, 来自字段档)
	Field      string // dzep002
	FieldName  string // dzebl003 (指定语言别)
	FieldDesc  string // dzebl004
	Widget     string // dzep010 (控件类型代码, 见 specWidgetLabels)
	Scc        string // dzep011 (系统分类码, 下拉数据源)
	Required   string // dzep005 (Y/N)
	Width      string // dzep009 (画面显示宽度 "整数,小数")
	Format     string // dzep021 (显示格式)
	CaseConv   string // dzep023 (字段大小写 U/L)
	DefaultVal string // dzep012 (默认值)
	MaxSym     string // dzep025 (最大值比较符号)
	MaxVal     string // dzep013 (最大值)
	MinSym     string // dzep026 (最小值比较符号)
	MinVal     string // dzep014 (最小值)
	OpenEdit   string // dzep017 (编辑时开窗程序代码)
	OpenQuery  string // dzep018 (查询时开窗程序代码)
	ChkVal     string // dzep019 (r.v 校验带值代码)
	LookupType string // dzep022 (串查型态 '1'=一般)
	LookupProg string // dzep020 (串查/参考程序代码)
	RepWidth   string // dzep027 (报表栏宽)
	RepDigit   string // dzep028 (报表小数位数)
	Env        string // dzepstus (环境/客制识别 s/c)
}

// QuerySpecRows returns all field spec rows of one table, ordered by the
// field sequence (dzeb_t.dzeb021) as in the adzi150 maintenance screen.
func (d *DB) QuerySpecRows(tableName, lang string) ([]SpecRow, error) {
	query := `
		SELECT COALESCE(eb.dzeb021, ''), ep.dzep002,
		       COALESCE(ebl.dzebl003, ''), COALESCE(ebl.dzebl004, ''),
		       COALESCE(ep.dzep010, ''), COALESCE(ep.dzep011, ''),
		       COALESCE(ep.dzep005, ''), COALESCE(ep.dzep009, ''),
		       COALESCE(ep.dzep021, ''), COALESCE(ep.dzep023, ''),
		       COALESCE(ep.dzep012, ''), COALESCE(ep.dzep025, ''), COALESCE(ep.dzep013, ''),
		       COALESCE(ep.dzep026, ''), COALESCE(ep.dzep014, ''),
		       COALESCE(ep.dzep017, ''), COALESCE(ep.dzep018, ''),
		       COALESCE(ep.dzep019, ''), COALESCE(ep.dzep022, ''), COALESCE(ep.dzep020, ''),
		       COALESCE(ep.dzep027, ''), COALESCE(ep.dzep028, ''),
		       COALESCE(ep.dzepstus, '')
		FROM dzep_t ep
		LEFT JOIN dzeb_t eb ON eb.dzeb001 = ep.dzep001 AND eb.dzeb002 = ep.dzep002
		LEFT JOIN dzebl_t ebl ON ebl.dzebl001 = ep.dzep002 AND ebl.dzebl002 = ?
		WHERE ep.dzep001 = ?
		ORDER BY CAST(COALESCE(eb.dzeb021, '0') AS INTEGER), ep.dzep002
	`
	rows, err := d.conn.Query(query, lang, tableName)
	if err != nil {
		return nil, fmt.Errorf("query spec rows %s: %w", tableName, err)
	}
	defer rows.Close()

	var result []SpecRow
	for rows.Next() {
		var r SpecRow
		if err := rows.Scan(&r.Seq, &r.Field, &r.FieldName, &r.FieldDesc,
			&r.Widget, &r.Scc, &r.Required, &r.Width, &r.Format, &r.CaseConv,
			&r.DefaultVal, &r.MaxSym, &r.MaxVal, &r.MinSym, &r.MinVal,
			&r.OpenEdit, &r.OpenQuery, &r.ChkVal, &r.LookupType, &r.LookupProg,
			&r.RepWidth, &r.RepDigit, &r.Env); err != nil {
			return nil, fmt.Errorf("scan spec row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
