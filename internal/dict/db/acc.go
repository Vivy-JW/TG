package db

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// ACCMeta 应用分类码（ACC）的元模型。
//
// 这张结构刻意不叫 TableDict：ACC 不是一张字典表，而是一个由
// gzaa（分类）+ gzac/gzad（参考字段）+ gzzz（作业注册）共同描述的
// “可配置维护作业”。值在 oocq/oocql，按企业隔离。
type ACCMeta struct {
	Input     string     `json:"输入"`
	Category  string     `json:"分类号"`
	Name      string     `json:"名称"`
	Module    string     `json:"模块"`
	System    string     `json:"系统标准"`
	Tree      bool       `json:"树状"`
	ShowLines bool       `json:"显示单身"`
	Status    string     `json:"状态"`
	MatchedBy string     `json:"匹配方式"` // category | job
	Jobs      []ACCJob   `json:"维护作业"`
	Fields    []ACCField `json:"参考字段"`
}

// ACCJob 一个把 ACC 分类挂到渲染程序上的作业。
// 同一个分类可以有多个作业（例如普通版与 BPM 版），因此必须是数组。
type ACCJob struct {
	Code    string `json:"作业编号"`
	Program string `json:"程序"`
	Name    string `json:"作业名称,omitempty"`
	Extra   string `json:"额外参数,omitempty"`
	Status  string `json:"状态,omitempty"`
}

// ACCField 是 gzac（标题）+ gzad（控制）合并后的一个参考字段。
type ACCField struct {
	Seq         int    `json:"序号"`
	Key         string `json:"key"`
	Column      string `json:"-"`
	Title       string `json:"标题"`
	Visible     bool   `json:"显示"`
	AllowBlank  bool   `json:"可空白"`
	DataSource  int    `json:"数据源"`
	Target      string `json:"目标,omitempty"`
	OpenProgram string `json:"开窗程序,omitempty"`
	Check       string `json:"校验带值,omitempty"`
	Kind        string `json:"形态,omitempty"`
	Default     string `json:"默认值,omitempty"`
}

// ACCValue 一行 oocq 值。Refs 固定 20 项，对应参考字段一~二十。
type ACCValue struct {
	Value  string   `json:"值"`
	Desc   string   `json:"说明"`
	Parent string   `json:"上层"`
	Status string   `json:"状态"`
	Refs   []string `json:"参考字段"`
}

type accHeader struct {
	category  string
	name      string
	module    string
	system    string
	tree      bool
	showLines bool
	status    string
}

// QueryACCMeta 按分类号或作业编号解析 ACC 元模型。
//
// 先按 gzaa001 当分类查；查不到再把输入当 gzzz001 作业查，取 gzzz004
// 的分类号。这样调用方给 3211 或 aapi010 都能落到同一个分类。
func (d *DB) QueryACCMeta(input, lang string) (*ACCMeta, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	category, matchedBy, err := d.resolveACCCategory(input)
	if err != nil {
		return nil, err
	}
	if category == "" {
		return nil, nil
	}
	h, err := d.queryACCHeader(category, lang)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, nil
	}
	jobs, err := d.queryACCJobs(category, lang)
	if err != nil {
		return nil, err
	}
	titles, err := d.queryACCTitles(category, lang)
	if err != nil {
		return nil, err
	}
	controls, err := d.queryACCControls(category)
	if err != nil {
		return nil, err
	}
	fields := make([]ACCField, 0, 20)
	for i := 1; i <= 20; i++ {
		f := ACCField{
			Seq: i, Key: fmt.Sprintf("ref%d", i), Column: fmt.Sprintf("oocq%03d", 3+i),
		}
		if i-1 < len(titles) {
			f.Title = titles[i-1]
		}
		if c, ok := controls[i]; ok {
			f.Visible = c.visible
			f.AllowBlank = c.allowBlank
			f.DataSource = c.dataSource
			f.Target = c.target
			f.OpenProgram = c.openProgram
			f.Check = c.check
			f.Kind = c.kind
			f.Default = c.defaultValue
		}
		fields = append(fields, f)
	}
	return &ACCMeta{
		Input: input, Category: h.category, Name: h.name, Module: h.module,
		System: h.system, Tree: h.tree, ShowLines: h.showLines, Status: h.status,
		MatchedBy: matchedBy, Jobs: jobs, Fields: fields,
	}, nil
}

// resolveACCCategory 返回分类号与匹配方式（category/job）。
func (d *DB) resolveACCCategory(input string) (category, matchedBy string, err error) {
	// 先当分类号查。分类号在 gzaa_t 是 NUMBER，Oracle 拿
	// 作业编号（aapi010）直接比会 ORA-01722，所以只对纯数字输入走这一支。
	if isDigits(input) {
		var got string
		err = d.conn.QueryRow(`SELECT COALESCE(gzaa001, '') FROM gzaa_t WHERE gzaa001 = ?`, input).Scan(&got)
		switch {
		case err == nil && got != "":
			return got, "category", nil
		case err != nil && err != sql.ErrNoRows:
			return "", "", fmt.Errorf("查询 ACC 分类 %s: %w", input, err)
		}
	}

	// 再当作业编号查，分类号在 gzzz004（可能带引号）。
	var extra string
	err = d.conn.QueryRow(`SELECT COALESCE(gzzz004, '') FROM gzzz_t WHERE gzzz001 = ?`, input).Scan(&extra)
	switch {
	case err == sql.ErrNoRows:
		return "", "", nil
	case err != nil:
		return "", "", fmt.Errorf("查询 ACC 作业 %s: %w", input, err)
	}
	cat := NormalizeACCJobExtra(extra)
	if cat == "" {
		return "", "", nil
	}
	return cat, "job", nil
}

func (d *DB) queryACCHeader(category, lang string) (*accHeader, error) {
	var h accHeader
	var tree, show, status string
	err := d.conn.QueryRow(`
		SELECT gzaa001, COALESCE(gzaa002, ''), COALESCE(gzaa003, ''),
		       COALESCE(gzaa004, ''), COALESCE(gzaa005, ''), COALESCE(gzaastus, '')
		FROM gzaa_t WHERE gzaa001 = ?`, category).
		Scan(&h.category, &h.system, &tree, &h.module, &show, &status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 分类 %s: %w", category, err)
	}
	h.tree = isY(tree)
	h.showLines = isY(show)
	h.status = status

	var name string
	err = d.conn.QueryRow(`SELECT COALESCE(gzaal003, '') FROM gzaal_t WHERE gzaal001 = ? AND gzaal002 = ?`,
		category, lang).Scan(&name)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("查询 ACC 名称 %s: %w", category, err)
	}
	h.name = name
	return &h, nil
}

func (d *DB) queryACCJobs(category, lang string) ([]ACCJob, error) {
	rows, err := d.conn.Query(`
		SELECT z.gzzz001, z.gzzz002, COALESCE(z.gzzz004, ''),
		       COALESCE(z.gzzzstus, ''), COALESCE(l.gzzal003, '')
		FROM gzzz_t z
		LEFT JOIN gzzal_t l ON l.gzzal001 = z.gzzz002 AND l.gzzal002 = ?
		ORDER BY z.gzzz001`, lang)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 作业注册: %w", err)
	}
	defer rows.Close()
	out := make([]ACCJob, 0)
	for rows.Next() {
		var job ACCJob
		if err := rows.Scan(&job.Code, &job.Program, &job.Extra, &job.Status, &job.Name); err != nil {
			return nil, err
		}
		if NormalizeACCJobExtra(job.Extra) != category {
			continue
		}
		if job.Name == "" {
			job.Name = job.Code
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

func (d *DB) queryACCTitles(category, lang string) ([]string, error) {
	row := d.conn.QueryRow(`
		SELECT COALESCE(gzac003, ''), COALESCE(gzac004, ''), COALESCE(gzac005, ''),
		       COALESCE(gzac006, ''), COALESCE(gzac007, ''), COALESCE(gzac008, ''),
		       COALESCE(gzac009, ''), COALESCE(gzac010, ''), COALESCE(gzac011, ''),
		       COALESCE(gzac012, ''), COALESCE(gzac013, ''), COALESCE(gzac014, ''),
		       COALESCE(gzac015, ''), COALESCE(gzac016, ''), COALESCE(gzac017, ''),
		       COALESCE(gzac018, ''), COALESCE(gzac019, ''), COALESCE(gzac020, ''),
		       COALESCE(gzac021, ''), COALESCE(gzac022, '')
		FROM gzac_t WHERE gzac001 = ? AND gzac002 = ?`, category, lang)
	var titles [20]string
	args := make([]any, len(titles))
	for i := range titles {
		args[i] = &titles[i]
	}
	err := row.Scan(args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 字段标题 %s: %w", category, err)
	}
	return titles[:], nil
}

type accControl struct {
	visible      bool
	allowBlank   bool
	dataSource   int
	target       string
	openProgram  string
	check        string
	kind         string
	defaultValue string
}

func (d *DB) queryACCControls(category string) (map[int]accControl, error) {
	rows, err := d.conn.Query(`
		SELECT gzad002, COALESCE(gzad003, ''), COALESCE(gzad004, ''),
		       COALESCE(gzad005, ''), COALESCE(gzad006, ''), COALESCE(gzad007, ''),
		       COALESCE(gzad008, ''), COALESCE(gzad009, ''), COALESCE(gzad014, '')
		FROM gzad_t WHERE gzad001 = ? ORDER BY gzad002`, category)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 字段控制 %s: %w", category, err)
	}
	defer rows.Close()
	out := make(map[int]accControl)
	for rows.Next() {
		var seq, visible, allowBlank, source string
		var c accControl
		if err := rows.Scan(&seq, &visible, &allowBlank, &source, &c.target,
			&c.openProgram, &c.check, &c.kind, &c.defaultValue); err != nil {
			return nil, err
		}
		n, _ := strconv.Atoi(strings.TrimSpace(seq))
		if n < 1 || n > 20 {
			continue
		}
		c.dataSource, _ = strconv.Atoi(strings.TrimSpace(source))
		c.visible = isY(visible)
		c.allowBlank = isY(allowBlank)
		out[n] = c
	}
	return out, rows.Err()
}

// QueryACCValues 读取一个分类在指定企业下的全部值。
// 过滤/选择/分页由命令层在内存里完成：ACC 的值数量有限，而这样能保证
// 查询规格里的字段永远不会被拼进 SQL。
func (d *DB) QueryACCValues(category string, ent int, lang string, includeInactive bool) ([]ACCValue, error) {
	if ent <= 0 {
		return nil, fmt.Errorf("ACC 值查询需要企业编号")
	}
	cols := make([]string, 20)
	for i := range cols {
		cols[i] = fmt.Sprintf("COALESCE(q.oocq%03d, '')", i+4)
	}
	sqlStr := `SELECT q.oocq002, COALESCE(l.oocql004, ''), COALESCE(q.oocq003, ''),
		COALESCE(q.oocqstus, ''), ` + strings.Join(cols, ", ") + `
		FROM oocq_t q
		LEFT JOIN oocql_t l ON l.oocqlent = q.oocqent AND l.oocql001 = q.oocq001
			AND l.oocql002 = q.oocq002 AND l.oocql003 = ?
		WHERE q.oocqent = ? AND q.oocq001 = ?`
	if !includeInactive {
		sqlStr += ` AND COALESCE(q.oocqstus, '') = 'Y'`
	}
	sqlStr += ` ORDER BY q.oocq002`
	rows, err := d.conn.Query(sqlStr, lang, ent, category)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 值 %s: %w", category, err)
	}
	defer rows.Close()
	dest := make([]any, 24)
	vals := make([]string, 24)
	for i := range dest {
		dest[i] = &vals[i]
	}
	var out []ACCValue
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		out = append(out, ACCValue{
			Value: vals[0], Desc: vals[1], Parent: vals[2], Status: vals[3],
			Refs: append([]string(nil), vals[4:]...),
		})
	}
	return out, rows.Err()
}

// NormalizeACCJobExtra 去掉 gzzz004 的引号/空白，取出分类号。
// gzzz004 是“额外参数”，实际可能写成 "3211"、'3211' 或 3211。
func NormalizeACCJobExtra(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
		s = s[:i]
	}
	return strings.Trim(s, `"'`)
}

func isY(s string) bool { return strings.EqualFold(strings.TrimSpace(s), "Y") }

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
