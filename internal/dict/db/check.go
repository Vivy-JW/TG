package db

import "fmt"

// CheckListRow holds one row of the check definition list (dzcd_t LEFT JOIN dzcdl_t).
type CheckListRow struct {
	ID       string
	Cust     string // dzcd002
	Desc     string // dzcdl003 (指定语言别)
	TypeCode string // dzcd005
	ErrMsg   string // dzcd004
	Remark   string // dzcdl004
}

// QueryCheckList returns all check definitions (dzcd_t), optionally filtered
// by keyword (matched against id and description).
func (d *DB) QueryCheckList(lang, keyword string) ([]CheckListRow, error) {
	like := "%" + keyword + "%"
	query := `
		SELECT c.dzcd001, COALESCE(c.dzcd002, ''),
		       COALESCE(cl.dzcdl003, ''),
		       COALESCE(c.dzcd005, ''),
		       COALESCE(c.dzcd004, ''),
		       COALESCE(cl.dzcdl004, '')
		FROM dzcd_t c
		LEFT JOIN dzcdl_t cl ON cl.dzcdl001 = c.dzcd001 AND cl.dzcdl002 = ?
		WHERE (? = '' OR c.dzcd001 LIKE ? OR COALESCE(cl.dzcdl003, '') LIKE ?)
		ORDER BY c.dzcd001, CASE c.dzcd002 WHEN 's' THEN 0 ELSE 1 END
	`
	rows, err := d.conn.Query(query, lang, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query check list: %w", err)
	}
	defer rows.Close()

	var result []CheckListRow
	for rows.Next() {
		var r CheckListRow
		if err := rows.Scan(&r.ID, &r.Cust, &r.Desc, &r.TypeCode, &r.ErrMsg, &r.Remark); err != nil {
			return nil, fmt.Errorf("scan check row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CheckHeaderRow holds a check definition header (dzcd_t + dzcdl_t).
type CheckHeaderRow struct {
	ID       string
	Cust     string // dzcd002
	TypeCode string // dzcd005
	ErrMsg   string // dzcd004
	Industry string // dzcd006
	Status   string // dzcdstus
	Desc     string // dzcdl003 (指定语言别)
	Remark   string // dzcdl004
	SQL      string // dzcd003
}

// QueryCheckHeaders returns all header variants (标准/客制) of a check definition.
func (d *DB) QueryCheckHeaders(id, lang string) ([]CheckHeaderRow, error) {
	query := `
		SELECT c.dzcd001, COALESCE(c.dzcd002, ''),
		       COALESCE(c.dzcd005, ''),
		       COALESCE(c.dzcd004, ''),
		       COALESCE(c.dzcd006, ''),
		       COALESCE(c.dzcdstus, ''),
		       COALESCE(cl.dzcdl003, ''),
		       COALESCE(cl.dzcdl004, ''),
		       COALESCE(c.dzcd003, '')
		FROM dzcd_t c
		LEFT JOIN dzcdl_t cl ON cl.dzcdl001 = c.dzcd001 AND cl.dzcdl002 = ?
		WHERE c.dzcd001 = ?
		ORDER BY CASE c.dzcd002 WHEN 's' THEN 0 ELSE 1 END
	`
	rows, err := d.conn.Query(query, lang, id)
	if err != nil {
		return nil, fmt.Errorf("query check header %s: %w", id, err)
	}
	defer rows.Close()

	var result []CheckHeaderRow
	for rows.Next() {
		var r CheckHeaderRow
		if err := rows.Scan(&r.ID, &r.Cust, &r.TypeCode, &r.ErrMsg, &r.Industry,
			&r.Status, &r.Desc, &r.Remark, &r.SQL); err != nil {
			return nil, fmt.Errorf("scan check header: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CheckParamRow holds an external parameter definition (dzce_t + dzcel_t).
type CheckParamRow struct {
	Cust     string // dzce003
	Seq      string // dzce002
	Name     string // dzce004
	DateType string // dzcestus
	Desc     string // dzcel004 (指定语言别)
	Remark   string // dzcel005
}

// QueryCheckParams returns the external parameter (argN) definitions of a check.
func (d *DB) QueryCheckParams(id, lang string) ([]CheckParamRow, error) {
	query := `
		SELECT COALESCE(e.dzce003, ''), e.dzce002,
		       COALESCE(e.dzce004, ''),
		       COALESCE(e.dzcestus, ''),
		       COALESCE(el.dzcel004, ''),
		       COALESCE(el.dzcel005, '')
		FROM dzce_t e
		LEFT JOIN dzcel_t el ON el.dzcel001 = e.dzce001 AND el.dzcel002 = e.dzce002 AND el.dzcel003 = ?
		WHERE e.dzce001 = ?
		ORDER BY CAST(e.dzce002 AS INTEGER)
	`
	rows, err := d.conn.Query(query, lang, id)
	if err != nil {
		return nil, fmt.Errorf("query check params %s: %w", id, err)
	}
	defer rows.Close()

	var result []CheckParamRow
	for rows.Next() {
		var r CheckParamRow
		if err := rows.Scan(&r.Cust, &r.Seq, &r.Name, &r.DateType, &r.Desc, &r.Remark); err != nil {
			return nil, fmt.Errorf("scan check param: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// CheckCondRow holds a condition detail (dzch_t).
type CheckCondRow struct {
	Cust   string // dzch005
	Seq    string // dzch002
	Cond   string // dzch003
	ErrMsg string // dzch004
}

// QueryCheckConds returns the existence condition details of a check.
func (d *DB) QueryCheckConds(id string) ([]CheckCondRow, error) {
	query := `
		SELECT COALESCE(h.dzch005, ''), h.dzch002,
		       COALESCE(h.dzch003, ''),
		       COALESCE(h.dzch004, '')
		FROM dzch_t h
		WHERE h.dzch001 = ?
		ORDER BY CAST(h.dzch002 AS INTEGER)
	`
	rows, err := d.conn.Query(query, id)
	if err != nil {
		return nil, fmt.Errorf("query check conds %s: %w", id, err)
	}
	defer rows.Close()

	var result []CheckCondRow
	for rows.Next() {
		var r CheckCondRow
		if err := rows.Scan(&r.Cust, &r.Seq, &r.Cond, &r.ErrMsg); err != nil {
			return nil, fmt.Errorf("scan check cond: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
