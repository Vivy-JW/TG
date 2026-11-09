package db

import "fmt"

// WinListRow holds one row of the window (r.q) list (dzca_t LEFT JOIN dzcal_t).
type WinListRow struct {
	ID       string // dzca001 开窗识别码
	Cust     string // dzca002 客制 s/c
	Desc     string // dzcal003 (指定语言别)
	Status   string // dzcastus
	PageSize string // dzca004 每页显现数据笔数
	HardCode string // dzca006 (Y=hardcode 跳过自动产生)
	Industry string // dzca007 行业别
}

// QueryWinList returns all reusable windows (dzca_t), optionally filtered by
// keyword (matched against id and description).
func (d *DB) QueryWinList(lang, keyword string) ([]WinListRow, error) {
	like := "%" + keyword + "%"
	query := `
		SELECT a.dzca001, COALESCE(a.dzca002, ''),
		       COALESCE(al.dzcal003, ''),
		       COALESCE(a.dzcastus, ''),
		       COALESCE(a.dzca004, ''),
		       COALESCE(a.dzca006, ''),
		       COALESCE(a.dzca007, '')
		FROM dzca_t a
		LEFT JOIN dzcal_t al ON al.dzcal001 = a.dzca001 AND al.dzcal002 = ?
		WHERE (? = '' OR a.dzca001 LIKE ? OR COALESCE(al.dzcal003, '') LIKE ?)
		ORDER BY a.dzca001, CASE a.dzca002 WHEN 's' THEN 0 ELSE 1 END
	`
	rows, err := d.conn.Query(query, lang, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query win list: %w", err)
	}
	defer rows.Close()

	var result []WinListRow
	for rows.Next() {
		var r WinListRow
		if err := rows.Scan(&r.ID, &r.Cust, &r.Desc, &r.Status, &r.PageSize, &r.HardCode, &r.Industry); err != nil {
			return nil, fmt.Errorf("scan win row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// WinHeaderRow holds a window definition header (dzca_t + dzcal_t).
type WinHeaderRow struct {
	ID       string // dzca001
	Cust     string // dzca002
	Status   string // dzcastus
	SQL      string // dzca003 (带标签的 SQL 文本)
	PageSize string // dzca004
	Serial   string // dzca005 作业串查编号
	HardCode string // dzca006
	Industry string // dzca007
	Desc     string // dzcal003 (指定语言别)
	Memo     string // dzcal004
}

// QueryWinHeaders returns all header variants (标准/客制) of one window.
func (d *DB) QueryWinHeaders(id, lang string) ([]WinHeaderRow, error) {
	query := `
		SELECT a.dzca001, COALESCE(a.dzca002, ''), COALESCE(a.dzcastus, ''),
		       COALESCE(a.dzca003, ''),
		       COALESCE(a.dzca004, ''), COALESCE(a.dzca005, ''),
		       COALESCE(a.dzca006, ''), COALESCE(a.dzca007, ''),
		       COALESCE(al.dzcal003, ''), COALESCE(al.dzcal004, '')
		FROM dzca_t a
		LEFT JOIN dzcal_t al ON al.dzcal001 = a.dzca001 AND al.dzcal002 = ?
		WHERE a.dzca001 = ?
		ORDER BY CASE a.dzca002 WHEN 's' THEN 0 ELSE 1 END
	`
	rows, err := d.conn.Query(query, lang, id)
	if err != nil {
		return nil, fmt.Errorf("query win header %s: %w", id, err)
	}
	defer rows.Close()

	var result []WinHeaderRow
	for rows.Next() {
		var r WinHeaderRow
		if err := rows.Scan(&r.ID, &r.Cust, &r.Status, &r.SQL, &r.PageSize, &r.Serial,
			&r.HardCode, &r.Industry, &r.Desc, &r.Memo); err != nil {
			return nil, fmt.Errorf("scan win header: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// WinParamRow holds an external parameter definition (dzcb_t + dzcbl_t).
type WinParamRow struct {
	Cust     string // dzcb004
	Seq      string // dzcb002
	Name     string // dzcb003 (参数名称, 对应 argN 序号)
	DateType string // dzcbstus
	Desc     string // dzcbl004 (指定语言别)
	Memo     string // dzcbl005
}

// QueryWinParams returns the external parameter (argN) definitions of a window.
func (d *DB) QueryWinParams(id, lang string) ([]WinParamRow, error) {
	query := `
		SELECT COALESCE(b.dzcb004, ''), b.dzcb002,
		       COALESCE(b.dzcb003, ''),
		       COALESCE(b.dzcbstus, ''),
		       COALESCE(bl.dzcbl004, ''),
		       COALESCE(bl.dzcbl005, '')
		FROM dzcb_t b
		LEFT JOIN dzcbl_t bl ON bl.dzcbl001 = b.dzcb001 AND bl.dzcbl002 = b.dzcb002 AND bl.dzcbl003 = ?
		WHERE b.dzcb001 = ?
		ORDER BY CAST(b.dzcb002 AS INTEGER)
	`
	rows, err := d.conn.Query(query, lang, id)
	if err != nil {
		return nil, fmt.Errorf("query win params %s: %w", id, err)
	}
	defer rows.Close()

	var result []WinParamRow
	for rows.Next() {
		var r WinParamRow
		if err := rows.Scan(&r.Cust, &r.Seq, &r.Name, &r.DateType, &r.Desc, &r.Memo); err != nil {
			return nil, fmt.Errorf("scan win param: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// WinColRow holds a display column setting (dzcc_t).
type WinColRow struct {
	Cust     string // dzcc009
	Seq      string // dzcc002 显现顺序
	Field    string // dzcc003 字段编号
	Alias    string // dzcc008 对应表格别名
	Widget   string // dzcc004 显示控件
	IsRet    string // dzcc005 是否回传 (Y=按顺序构成 return1~9)
	CaseConv string // dzcc006 数据大小写
	Format   string // dzcc010 显示格式
	Label    string // dzcc007 画面标签缀字
}

// QueryWinCols returns the display column settings of one window.
func (d *DB) QueryWinCols(id string) ([]WinColRow, error) {
	query := `
		SELECT COALESCE(c.dzcc009, ''), c.dzcc002,
		       COALESCE(c.dzcc003, ''),
		       COALESCE(c.dzcc008, ''),
		       COALESCE(c.dzcc004, ''),
		       COALESCE(c.dzcc005, ''),
		       COALESCE(c.dzcc006, ''),
		       COALESCE(c.dzcc010, ''),
		       COALESCE(c.dzcc007, '')
		FROM dzcc_t c
		WHERE c.dzcc001 = ?
		ORDER BY CAST(c.dzcc002 AS INTEGER)
	`
	rows, err := d.conn.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("query win cols %s: %w", id, err)
	}
	defer rows.Close()

	var result []WinColRow
	for rows.Next() {
		var r WinColRow
		if err := rows.Scan(&r.Cust, &r.Seq, &r.Field, &r.Alias, &r.Widget, &r.IsRet,
			&r.CaseConv, &r.Format, &r.Label); err != nil {
			return nil, fmt.Errorf("scan win col: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
