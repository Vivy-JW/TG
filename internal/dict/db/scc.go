package db

import (
	"database/sql"
	"fmt"
)

// SccListRow holds one row of the SCC list (gzca_t LEFT JOIN gzcal_t + value count).
type SccListRow struct {
	ID     string // gzca001
	Group  string // gzca004
	Status string // gzcastus
	Name   string // gzcal003 (指定语言别)
	ValCnt int    // gzcb_t 中的分类值数
}

// QuerySccList returns all system classification codes (gzca_t), optionally
// filtered by keyword (matched against code and description).
func (d *DB) QuerySccList(lang, keyword string) ([]SccListRow, error) {
	like := "%" + keyword + "%"
	query := `
		SELECT a.gzca001, COALESCE(a.gzca004, ''), COALESCE(a.gzcastus, ''),
		       COALESCE(al.gzcal003, ''),
		       (SELECT COUNT(*) FROM gzcb_t WHERE gzcb001 = a.gzca001)
		FROM gzca_t a
		LEFT JOIN gzcal_t al ON al.gzcal001 = a.gzca001 AND al.gzcal002 = ?
		WHERE (? = '' OR a.gzca001 LIKE ? OR COALESCE(al.gzcal003, '') LIKE ?)
		ORDER BY CAST(a.gzca001 AS INTEGER)
	`
	rows, err := d.conn.Query(query, lang, keyword, like, like)
	if err != nil {
		return nil, fmt.Errorf("query scc list: %w", err)
	}
	defer rows.Close()

	var result []SccListRow
	for rows.Next() {
		var r SccListRow
		if err := rows.Scan(&r.ID, &r.Group, &r.Status, &r.Name, &r.ValCnt); err != nil {
			return nil, fmt.Errorf("scan scc row: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// SccHeaderRow holds the SCC header (gzca_t + gzcal_t).
type SccHeaderRow struct {
	ID     string // gzca001
	Group  string // gzca004
	Status string // gzcastus
	Aux2   string // gzca002
	Aux3   string // gzca003
	Name   string // gzcal003 (指定语言别)
	Desc   string // gzcal004
	Desc2  string // gzcal005
}

// QuerySccHeader returns the header of one SCC. Returns (nil, nil) if not found.
func (d *DB) QuerySccHeader(id, lang string) (*SccHeaderRow, error) {
	query := `
		SELECT a.gzca001, COALESCE(a.gzca004, ''), COALESCE(a.gzcastus, ''),
		       COALESCE(a.gzca002, ''), COALESCE(a.gzca003, ''),
		       COALESCE(al.gzcal003, ''), COALESCE(al.gzcal004, ''), COALESCE(al.gzcal005, '')
		FROM gzca_t a
		LEFT JOIN gzcal_t al ON al.gzcal001 = a.gzca001 AND al.gzcal002 = ?
		WHERE a.gzca001 = ?
	`
	var r SccHeaderRow
	err := d.conn.QueryRow(query, lang, id).Scan(&r.ID, &r.Group, &r.Status, &r.Aux2, &r.Aux3,
		&r.Name, &r.Desc, &r.Desc2)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query scc header %s: %w", id, err)
	}
	return &r, nil
}

// SccValueRow holds one classification value (gzcb_t LEFT JOIN gzcbl_t).
// B3..B15 are the generic data columns whose meaning depends on the SCC.
type SccValueRow struct {
	Val   string // gzcb002
	Sort  string // gzcb012
	Cust  string // gzcb013 (标准/客制)
	Desc  string // gzcbl004 (指定语言别)
	Desc2 string // gzcbl005
	B3    string // gzcb003
	B4    string // gzcb004
	B5    string // gzcb005
	B6    string // gzcb006
	B7    string // gzcb007
	B8    string // gzcb008
	B9    string // gzcb009
	B10   string // gzcb010
	B11   string // gzcb011
	B14   string // gzcb014
	B15   string // gzcb015
}

// QuerySccValues returns all classification values of one SCC, ordered by gzcb012.
func (d *DB) QuerySccValues(id, lang string) ([]SccValueRow, error) {
	query := `
		SELECT COALESCE(b.gzcb002, ''), COALESCE(b.gzcb012, ''), COALESCE(b.gzcb013, ''),
		       COALESCE(bl.gzcbl004, ''), COALESCE(bl.gzcbl005, ''),
		       COALESCE(b.gzcb003, ''), COALESCE(b.gzcb004, ''), COALESCE(b.gzcb005, ''),
		       COALESCE(b.gzcb006, ''), COALESCE(b.gzcb007, ''), COALESCE(b.gzcb008, ''),
		       COALESCE(b.gzcb009, ''), COALESCE(b.gzcb010, ''), COALESCE(b.gzcb011, ''),
		       COALESCE(b.gzcb014, ''), COALESCE(b.gzcb015, '')
		FROM gzcb_t b
		LEFT JOIN gzcbl_t bl ON bl.gzcbl001 = b.gzcb001 AND bl.gzcbl002 = b.gzcb002 AND bl.gzcbl003 = ?
		WHERE b.gzcb001 = ?
		ORDER BY CAST(b.gzcb012 AS INTEGER)
	`
	rows, err := d.conn.Query(query, lang, id)
	if err != nil {
		return nil, fmt.Errorf("query scc values %s: %w", id, err)
	}
	defer rows.Close()

	var result []SccValueRow
	for rows.Next() {
		var r SccValueRow
		if err := rows.Scan(&r.Val, &r.Sort, &r.Cust, &r.Desc, &r.Desc2,
			&r.B3, &r.B4, &r.B5, &r.B6, &r.B7, &r.B8, &r.B9, &r.B10, &r.B11, &r.B14, &r.B15); err != nil {
			return nil, fmt.Errorf("scan scc value: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
