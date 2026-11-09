package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// DB holds the database connection.
type DB struct {
	conn *sql.DB
}

// Open opens the SQLite database at the given path.
//
// 两处刻意为之:
//
//  1. **先建父目录**。SQLite 在父目录不存在时只回一句 "unable to open database file (14)",
//     不告诉你是哪一层缺了;调用方(如同步的临时库)如果忘了建目录,报错就落在很远的地方。
//  2. 失败时把**父目录及其状态**一起报出来。真实踩过:同步报
//     `创建临时数据库失败 (...\erp_data.db.sync.tmp): ping database: unable to open
//     database file (14)`,而那个目录当时明明存在、也能写 —— 只给文件路径的话,
//     排查只能靠猜。带上 dir 与 exists 之后,下次一眼能看出是"目录没了"还是"目录在但打不开"。
func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据库目录失败 (%s): %w", dir, err)
		}
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database (%s): %w", path, err)
	}
	conn.SetMaxOpenConns(1) // SQLite works best with single writer
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping database (%s, 父目录 %s 存在=%v): %w",
			path, filepath.Dir(path), dirExists(filepath.Dir(path)), err)
	}
	// 临时表/临时索引走内存。两件事：
	//  1. 本地库都是只读镜像查询，临时存储放内存少一次磁盘往返（大 GROUP BY 明显更快）；
	//  2. 趟过一个环境问题：exe 放在某些目录下时 SQLite 建临时文件会失败，
	//     大 GROUP BY 直接报 `unable to open database file (14)`（2026-10 实测，
	//     与本包代码无关，换一个目录跑同一个二进制就正常）。
	// 失败就按 SQLite 默认走 —— 一条 pragma 不该让打开库本身失败。
	_, _ = conn.Exec("PRAGMA temp_store=MEMORY")
	return &DB{conn: conn}, nil
}

// dirExists 供错误信息用:父目录在不在,是这类 "unable to open" 最要紧的一条线索。
func dirExists(dir string) bool {
	st, err := os.Stat(dir)
	return err == nil && st.IsDir()
}

// Close closes the database connection.
func (d *DB) Close() {
	_ = d.conn.Close()
}

// TableInfo holds a single field's information from a table dictionary query.
type TableInfo struct {
	Seq       string `json:"序号"`
	FieldName string `json:"字段名"`
	FieldDesc string `json:"字段说明"`
	DataType  string `json:"数据类型"`
	Length    string `json:"长度"`
	IsPK      string `json:"主键"`
	Required  string `json:"必填"`
	Remark    string `json:"备注"`
}

// QueryTable returns the field dictionary for the given table name.
func (d *DB) QueryTable(tableName string) ([]TableInfo, error) {
	query := `
		SELECT b.dzeb021, b.dzeb002,
		       COALESCE(bl.dzebl003, b.dzeb003, ''),
		       COALESCE(b.dzeb007, ''),
		       COALESCE(b.dzeb008, ''),
		       CASE WHEN b.dzeb004 = 'Y' THEN 'PK' ELSE '' END,
		       CASE WHEN b.dzeb005 = 'Y' THEN 'Y' ELSE '' END,
		       COALESCE(b.dzeb024, '')
		FROM dzeb_t b
		LEFT JOIN dzebl_t bl ON bl.dzebl001 = b.dzeb002 AND bl.dzebl002 = 'zh_CN'
		WHERE b.dzeb001 = ?
		ORDER BY CAST(b.dzeb021 AS INTEGER)
	`
	rows, err := d.conn.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("query table %s: %w", tableName, err)
	}
	defer rows.Close()

	var result []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Seq, &t.FieldName, &t.FieldDesc,
			&t.DataType, &t.Length, &t.IsPK, &t.Required, &t.Remark); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// FieldInfo holds information about a field found across tables.
type FieldInfo struct {
	TableName string `json:"表名"`
	FieldName string `json:"字段名"`
	FieldDesc string `json:"字段说明"`
	DataType  string `json:"数据类型"`
	Length    string `json:"长度"`
	IsPK      string `json:"主键"`
	Required  string `json:"必填"`
	Remark    string `json:"备注"`
}

// QueryField searches for the given field name across all tables.
func (d *DB) QueryField(fieldName string) ([]FieldInfo, error) {
	query := `
		SELECT b.dzeb001, b.dzeb002,
		       COALESCE(bl.dzebl003, b.dzeb003, ''),
		       COALESCE(b.dzeb007, ''),
		       COALESCE(b.dzeb008, ''),
		       CASE WHEN b.dzeb004 = 'Y' THEN 'PK' ELSE '' END,
		       CASE WHEN b.dzeb005 = 'Y' THEN 'Y' ELSE '' END,
		       COALESCE(b.dzeb024, '')
		FROM dzeb_t b
		LEFT JOIN dzebl_t bl ON bl.dzebl001 = b.dzeb002 AND bl.dzebl002 = 'zh_CN'
		WHERE b.dzeb002 = ?
		ORDER BY b.dzeb001
	`
	rows, err := d.conn.Query(query, fieldName)
	if err != nil {
		return nil, fmt.Errorf("query field %s: %w", fieldName, err)
	}
	defer rows.Close()

	var result []FieldInfo
	for rows.Next() {
		var f FieldInfo
		if err := rows.Scan(&f.TableName, &f.FieldName, &f.FieldDesc,
			&f.DataType, &f.Length, &f.IsPK, &f.Required, &f.Remark); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, f)
	}
	return result, rows.Err()
}

// TableListItem holds a row from the table list query.
type TableListItem struct {
	TableName string `json:"表名"`
	TableDesc string `json:"表说明"`
	Module    string `json:"模块"`
	TableType string `json:"类型"`
	FieldCnt  int    `json:"字段数"`
}

// QueryTableList returns all tables, optionally filtered by keyword
// (matched against table name and zh_CN/指定语言 table description).
//
// 字段数用 LEFT JOIN + COUNT 一次算出,而不是逐行相关子查询:
// 3,882 张表 × 15 万行字段表,相关子查询实测 81 秒,改 JOIN 后 0.25 秒(本地无索引的库也如此)。
func (d *DB) QueryTableList(lang, keyword string) ([]TableListItem, error) {
	like := "%" + keyword + "%"
	query := `
		SELECT a.dzea001,
		       COALESCE(al.dzeal003, a.dzea002, ''),
		       COALESCE(a.dzea003, ''),
		       COALESCE(a.dzea004, ''),
		       COUNT(b.dzeb001)
		FROM dzea_t a
		LEFT JOIN dzeal_t al ON al.dzeal001 = a.dzea001 AND al.dzeal002 = ?
		LEFT JOIN dzeb_t b ON b.dzeb001 = a.dzea001
		WHERE (? = '' OR a.dzea001 LIKE ? OR al.dzeal003 LIKE ? OR a.dzea002 LIKE ?)
		GROUP BY a.dzea001, al.dzeal003, a.dzea002, a.dzea003, a.dzea004
		ORDER BY a.dzea001
	`
	rows, err := d.conn.Query(query, lang, keyword, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("query table list: %w", err)
	}
	defer rows.Close()

	var result []TableListItem
	for rows.Next() {
		var t TableListItem
		if err := rows.Scan(&t.TableName, &t.TableDesc, &t.Module, &t.TableType, &t.FieldCnt); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// TableRowCounts 返回给定表中"本地库里存在"那些的行数(不存在的表不出现在结果里,
// 也不会报错)。用于 `tt dict db status` 检查本地库对各数据族的覆盖情况。
func (d *DB) TableRowCounts(names []string) (map[string]int64, error) {
	out := make(map[string]int64, len(names))
	for _, n := range names {
		if !validIdent(n) {
			continue
		}
		var c int64
		if err := d.conn.QueryRow(`SELECT COUNT(*) FROM ` + quoteIdent(n)).Scan(&c); err != nil {
			continue // 表不存在:跳过(缺表由调用方按数据族汇总)
		}
		out[n] = c
	}
	return out, nil
}

// QueryRaw executes a raw SQL query and returns rows as maps.
func (d *DB) QueryRaw(sqlStr string) ([]string, [][]string, error) {
	// Basic safety: reject dangerous write operations
	upper := strings.ToUpper(strings.TrimSpace(sqlStr))
	dangerous := []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE", "REPLACE",
		"ATTACH", "DETACH", "PRAGMA", "REINDEX", "VACUUM", "GRANT", "REVOKE"}
	for _, kw := range dangerous {
		if strings.HasPrefix(upper, kw) {
			return nil, nil, fmt.Errorf("write operation '%s' is not allowed, only SELECT queries are permitted", kw)
		}
	}

	rows, err := d.conn.Query(sqlStr)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("get columns: %w", err)
	}

	var result [][]string
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, nil, fmt.Errorf("scan row: %w", err)
		}
		row := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				row[i] = ""
			} else {
				switch val := v.(type) {
				case []byte:
					row[i] = string(val)
				default:
					row[i] = fmt.Sprintf("%v", val)
				}
			}
		}
		result = append(result, row)
	}
	return columns, result, rows.Err()
}

// validIdent reports whether s is a safe SQL identifier (letters/digits/underscore,
// not starting with a digit).
func validIdent(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return !(s[0] >= '0' && s[0] <= '9')
}

func quoteIdent(s string) string { return `"` + s + `"` }

// RebuildTable drops the given table (if it exists), recreates it with all-TEXT
// columns matching the given column names, then bulk-inserts the rows.
// It returns the number of rows inserted.
func (d *DB) RebuildTable(tableName string, columns []string, rows [][]string) (int, error) {
	if !validIdent(tableName) {
		return 0, fmt.Errorf("非法表名: %s", tableName)
	}
	colDefs := make([]string, len(columns))
	quotedCols := make([]string, len(columns))
	for i, c := range columns {
		if !validIdent(c) {
			return 0, fmt.Errorf("非法列名: %s", c)
		}
		colDefs[i] = quoteIdent(c) + " TEXT"
		quotedCols[i] = quoteIdent(c)
	}

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdent(tableName))
	if _, err := d.conn.Exec(dropSQL); err != nil {
		return 0, fmt.Errorf("删除表 %s: %w", tableName, err)
	}
	createSQL := fmt.Sprintf("CREATE TABLE %s (%s)", quoteIdent(tableName), strings.Join(colDefs, ", "))
	if _, err := d.conn.Exec(createSQL); err != nil {
		return 0, fmt.Errorf("创建表 %s: %w", tableName, err)
	}

	if len(rows) == 0 {
		return 0, nil
	}

	// Multi-row bulk insert. journal_mode=OFF + synchronous=OFF should be set
	// beforehand via SetBulkWriteMode for maximum speed on the throwaway temp DB.
	chunk := 500
	if len(columns) > 0 {
		if m := 10000 / len(columns); m < chunk {
			chunk = m
		}
		if chunk < 1 {
			chunk = 1
		}
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?, ", len(columns)), ", ")

	inserted := 0
	for start := 0; start < len(rows); start += chunk {
		end := start + chunk
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[start:end]

		var sb strings.Builder
		fmt.Fprintf(&sb, "INSERT INTO %s (%s) VALUES ", quoteIdent(tableName), strings.Join(quotedCols, ", "))
		vals := make([]interface{}, 0, len(batch)*len(columns))
		for ri := range batch {
			if ri > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString("(")
			sb.WriteString(placeholders)
			sb.WriteString(")")
			for _, v := range batch[ri] {
				vals = append(vals, v)
			}
		}
		if _, err := d.conn.Exec(sb.String(), vals...); err != nil {
			return inserted, fmt.Errorf("插入表 %s: %w", tableName, err)
		}
		inserted += len(batch)
	}
	return inserted, nil
}

// SetBulkWriteMode configures the connection for fast bulk writes
// (journal off + no fsync). Only safe for a throwaway database being built
// from scratch; a crash may corrupt it.
func (d *DB) SetBulkWriteMode() error {
	if _, err := d.conn.Exec("PRAGMA journal_mode=OFF"); err != nil {
		return fmt.Errorf("set journal_mode: %w", err)
	}
	if _, err := d.conn.Exec("PRAGMA synchronous=OFF"); err != nil {
		return fmt.Errorf("set synchronous: %w", err)
	}
	return nil
}

// CreateUniqueIndex creates a UNIQUE index on the given table and columns.
// It is a no-op (with nil error) if an index with the same name already exists.
func (d *DB) CreateUniqueIndex(indexName, tableName string, columns []string) error {
	if !validIdent(indexName) || !validIdent(tableName) {
		return fmt.Errorf("非法索引/表名: %s / %s", indexName, tableName)
	}
	quoted := make([]string, len(columns))
	for i, c := range columns {
		if !validIdent(c) {
			return fmt.Errorf("非法列名: %s", c)
		}
		quoted[i] = quoteIdent(c)
	}
	// 先确认列真的在表里。不查的话 SQLite 会走"双引号串"的历史兼容行为：
	// 解析不到标识符就当成字符串字面量，于是在 (常量) 上建出一个**假索引** ——
	// 单行表静默成功（不加速，只挡插入），多行表则报 "UNIQUE constraint failed"，
	// 而那个提示把人引向"数据重复"，根本没指向"列名写错了"。dzed_t 的列名与
	// 实际表对不上时（大小写/命名差异）正是这么误导人的。
	have, err := d.tableColumns(tableName)
	if err != nil {
		return fmt.Errorf("创建索引 %s: 检查表 %s 的列失败: %w", indexName, tableName, err)
	}
	for _, c := range columns {
		if !have[strings.ToLower(c)] {
			return fmt.Errorf("创建索引 %s: 表 %s 没有列 %q（索引列名与表结构对不上；"+
				"SQLite 会把不存在又带引号的列当字符串字面量，建出来的是假索引）", indexName, tableName, c)
		}
	}
	sql := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s)",
		quoteIdent(indexName), quoteIdent(tableName), strings.Join(quoted, ", "))
	if _, err := d.conn.Exec(sql); err != nil {
		return fmt.Errorf("创建索引 %s: %w", indexName, err)
	}
	return nil
}

// tableColumns 取一张表的全部列名(小写),供建索引前做列存在性检查。
// 表名已过 validIdent,拼进 PRAGMA 不会跑出标识符以外的东西。
func (d *DB) tableColumns(tableName string) (map[string]bool, error) {
	rows, err := d.conn.Query("PRAGMA table_info(" + quoteIdent(tableName) + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		// PRAGMA table_info 的列: cid, name, type, notnull, dflt_value, pk
		var (
			cid     int
			name    string
			typ     string
			notnull int
			dflt    any
			pk      int
		)
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return nil, err
		}
		out[strings.ToLower(name)] = true
	}
	return out, rows.Err()
}

// TableMeta holds the table's basic information.
type TableMeta struct {
	TableName string `json:"表名"`
	TableDesc string `json:"表说明"`
	Module    string `json:"模块"`
	TableType string `json:"类型"`
}

// QueryTableMeta returns the basic info (name, zh_CN description, module, type)
// for the given table. Returns (nil, nil) if the table is not in the registry.
func (d *DB) QueryTableMeta(tableName string) (*TableMeta, error) {
	query := `
		SELECT a.dzea001, COALESCE(al.dzeal003, a.dzea002, ''), COALESCE(a.dzea003, ''), COALESCE(a.dzea004, '')
		FROM dzea_t a
		LEFT JOIN dzeal_t al ON al.dzeal001 = a.dzea001 AND al.dzeal002 = 'zh_CN'
		WHERE a.dzea001 = ?`
	var m TableMeta
	err := d.conn.QueryRow(query, tableName).Scan(&m.TableName, &m.TableDesc, &m.Module, &m.TableType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query table meta %s: %w", tableName, err)
	}
	return &m, nil
}

// KeyInfo holds a key (主键/外键) definition from dzed_t.
type KeyInfo struct {
	KeyName  string `json:"键名"`
	KeyType  string `json:"类型"`
	KeyField string `json:"键值字段"`
	RefTable string `json:"外键表"`
	RefField string `json:"外键字段"`
}

// QueryKeys returns the key definitions (键值档 dzed_t) for the given table.
func (d *DB) QueryKeys(tableName string) ([]KeyInfo, error) {
	rows, err := d.conn.Query(`
		SELECT dzed002, COALESCE(dzed003, ''), COALESCE(dzed004, ''),
		       COALESCE(dzed005, ''), COALESCE(dzed006, '')
		FROM dzed_t WHERE dzed001 = ? ORDER BY dzed002`, tableName)
	if err != nil {
		return nil, fmt.Errorf("query keys %s: %w", tableName, err)
	}
	defer rows.Close()

	var result []KeyInfo
	for rows.Next() {
		var k KeyInfo
		if err := rows.Scan(&k.KeyName, &k.KeyType, &k.KeyField, &k.RefTable, &k.RefField); err != nil {
			return nil, fmt.Errorf("scan key row: %w", err)
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

// IndexInfo holds an index definition from dzec_t.
type IndexInfo struct {
	IndexName  string `json:"索引名"`
	IndexType  string `json:"类型"`
	IndexField string `json:"索引字段"`
}

// QueryIndexes returns the index definitions (索引档 dzec_t) for the given table.
func (d *DB) QueryIndexes(tableName string) ([]IndexInfo, error) {
	rows, err := d.conn.Query(`
		SELECT dzec002, COALESCE(dzec003, ''), COALESCE(dzec004, '')
		FROM dzec_t WHERE dzec001 = ? ORDER BY dzec002`, tableName)
	if err != nil {
		return nil, fmt.Errorf("query indexes %s: %w", tableName, err)
	}
	defer rows.Close()

	var result []IndexInfo
	for rows.Next() {
		var ix IndexInfo
		if err := rows.Scan(&ix.IndexName, &ix.IndexType, &ix.IndexField); err != nil {
			return nil, fmt.Errorf("scan index row: %w", err)
		}
		result = append(result, ix)
	}
	return result, rows.Err()
}

// TableDict holds a table's complete dictionary: basic info, fields, keys, indexes.
type TableDict struct {
	TableName string      `json:"表名"`
	TableDesc string      `json:"表说明"`
	Module    string      `json:"模块"`
	TableType string      `json:"类型"`
	Fields    []TableInfo `json:"字段"`
	Keys      []KeyInfo   `json:"键值"`
	Indexes   []IndexInfo `json:"索引"`
}
