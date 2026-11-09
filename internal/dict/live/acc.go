package live

import (
	"fmt"
	"strconv"
	"strings"

	"tt/internal/dict/db"
)

// QueryACCMeta 远程实现。SQL 与本地 SQLite 版本同源；值经 QuoteLit 内联。
func (l *Live) QueryACCMeta(input, lang string) (*db.ACCMeta, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	category, matchedBy, err := l.resolveACCCategory(input)
	if err != nil {
		return nil, err
	}
	if category == "" {
		return nil, nil
	}
	h, err := l.queryACCHeader(category, lang)
	if err != nil {
		return nil, err
	}
	if h == nil {
		return nil, nil
	}
	jobs, err := l.queryACCJobs(category, lang)
	if err != nil {
		return nil, err
	}
	titles, err := l.queryACCTitles(category, lang)
	if err != nil {
		return nil, err
	}
	controls, err := l.queryACCControls(category)
	if err != nil {
		return nil, err
	}
	fields := make([]db.ACCField, 0, 20)
	for i := 1; i <= 20; i++ {
		f := db.ACCField{
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
	return &db.ACCMeta{
		Input: input, Category: h.category, Name: h.name, Module: h.module,
		System: h.system, Tree: h.tree, ShowLines: h.showLines, Status: h.status,
		MatchedBy: matchedBy, Jobs: jobs, Fields: fields,
	}, nil
}

func (l *Live) resolveACCCategory(input string) (category, matchedBy string, err error) {
	if onlyDigits(input) {
		rows, err := l.q(`SELECT gzaa001 FROM gzaa_t WHERE gzaa001 = ` + lit(input))
		if err != nil {
			return "", "", fmt.Errorf("查询 ACC 分类 %s: %w", input, err)
		}
		if len(rows) > 0 && get(rows[0], 0) != "" {
			return get(rows[0], 0), "category", nil
		}
	}

	rows, err := l.q(`SELECT gzzz004 FROM gzzz_t WHERE gzzz001 = ` + lit(input))
	if err != nil {
		return "", "", fmt.Errorf("查询 ACC 作业 %s: %w", input, err)
	}
	if len(rows) == 0 {
		return "", "", nil
	}
	cat := db.NormalizeACCJobExtra(get(rows[0], 0))
	if cat == "" {
		return "", "", nil
	}
	return cat, "job", nil
}

func (l *Live) queryACCHeader(category, lang string) (*accHeader, error) {
	rows, err := l.q(`SELECT gzaa001, COALESCE(gzaa002, ''), COALESCE(gzaa003, ''),
		COALESCE(gzaa004, ''), COALESCE(gzaa005, ''), COALESCE(gzaastus, '')
		FROM gzaa_t WHERE gzaa001 = ` + numLit(category))
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 分类 %s: %w", category, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	r := rows[0]
	h := &accHeader{
		category: get(r, 0), system: get(r, 1), module: get(r, 3),
		status: get(r, 5), tree: isY(get(r, 2)), showLines: isY(get(r, 4)),
	}
	n, err := l.q(`SELECT COALESCE(gzaal003, '') FROM gzaal_t
		WHERE gzaal001 = ` + numLit(category) + ` AND gzaal002 = ` + lit(lang))
	if err != nil {
		return nil, err
	}
	if len(n) > 0 {
		h.name = get(n[0], 0)
	}
	return h, nil
}

func (l *Live) queryACCJobs(category, lang string) ([]db.ACCJob, error) {
	rows, err := l.q(`SELECT z.gzzz001, z.gzzz002, COALESCE(z.gzzz004, ''),
		COALESCE(z.gzzzstus, ''), COALESCE(gzzal003, '')
		FROM gzzz_t z
		LEFT JOIN gzzal_t l ON l.gzzal001 = z.gzzz002 AND l.gzzal002 = ` + lit(lang) + `
		ORDER BY z.gzzz001`)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 作业注册: %w", err)
	}
	out := make([]db.ACCJob, 0)
	for _, r := range rows {
		extra := get(r, 2)
		if db.NormalizeACCJobExtra(extra) != category {
			continue
		}
		job := db.ACCJob{
			Code: get(r, 0), Program: get(r, 1), Extra: extra,
			Status: get(r, 3), Name: get(r, 4),
		}
		if job.Name == "" {
			job.Name = job.Code
		}
		out = append(out, job)
	}
	return out, nil
}

func (l *Live) queryACCTitles(category, lang string) ([]string, error) {
	rows, err := l.q(`SELECT COALESCE(gzac003, ''), COALESCE(gzac004, ''), COALESCE(gzac005, ''),
		COALESCE(gzac006, ''), COALESCE(gzac007, ''), COALESCE(gzac008, ''),
		COALESCE(gzac009, ''), COALESCE(gzac010, ''), COALESCE(gzac011, ''),
		COALESCE(gzac012, ''), COALESCE(gzac013, ''), COALESCE(gzac014, ''),
		COALESCE(gzac015, ''), COALESCE(gzac016, ''), COALESCE(gzac017, ''),
		COALESCE(gzac018, ''), COALESCE(gzac019, ''), COALESCE(gzac020, ''),
		COALESCE(gzac021, ''), COALESCE(gzac022, '')
		FROM gzac_t WHERE gzac001 = ` + numLit(category) + ` AND gzac002 = ` + lit(lang))
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 字段标题 %s: %w", category, err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	r := rows[0]
	out := make([]string, 20)
	for i := range out {
		out[i] = get(r, i)
	}
	return out, nil
}

func (l *Live) queryACCControls(category string) (map[int]accControl, error) {
	rows, err := l.q(`SELECT gzad002, COALESCE(gzad003, ''), COALESCE(gzad004, ''),
		COALESCE(gzad005, ''), COALESCE(TO_CHAR(gzad006), ''), COALESCE(gzad007, ''),
		COALESCE(gzad008, ''), COALESCE(gzad009, ''), COALESCE(gzad014, '')
		FROM gzad_t WHERE gzad001 = ` + numLit(category) + ` ORDER BY gzad002`)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 字段控制 %s: %w", category, err)
	}
	out := make(map[int]accControl)
	for _, r := range rows {
		n, _ := strconv.Atoi(strings.TrimSpace(get(r, 0)))
		if n < 1 || n > 20 {
			continue
		}
		c := accControl{
			target: get(r, 4), openProgram: get(r, 5), check: get(r, 6),
			kind: get(r, 7), defaultValue: get(r, 8),
		}
		c.dataSource, _ = strconv.Atoi(strings.TrimSpace(get(r, 3)))
		c.visible = isY(get(r, 1))
		c.allowBlank = isY(get(r, 2))
		out[n] = c
	}
	return out, nil
}

// QueryACCValues 读取指定企业、指定分类的全部值；过滤/分页在命令层做。
func (l *Live) QueryACCValues(category string, ent int, lang string, includeInactive bool) ([]db.ACCValue, error) {
	if ent <= 0 {
		return nil, fmt.Errorf("ACC 值查询需要企业编号")
	}
	cols := make([]string, 20)
	for i := range cols {
		cols[i] = fmt.Sprintf("COALESCE(q.oocq%03d, '')", i+4)
	}
	sql := `SELECT q.oocq002, COALESCE(l.oocql004, ''), COALESCE(q.oocq003, ''),
		COALESCE(q.oocqstus, ''), ` + strings.Join(cols, ", ") + `
		FROM oocq_t q
		LEFT JOIN oocql_t l ON l.oocqlent = q.oocqent AND l.oocql001 = q.oocq001
			AND l.oocql002 = q.oocq002 AND l.oocql003 = ` + lit(lang) + `
		WHERE q.oocqent = ` + lit(strconv.Itoa(ent)) + ` AND q.oocq001 = ` + numLit(category)
	if !includeInactive {
		sql += ` AND COALESCE(q.oocqstus, '') = 'Y'`
	}
	sql += ` ORDER BY q.oocq002`
	rows, err := l.q(sql)
	if err != nil {
		return nil, fmt.Errorf("查询 ACC 值 %s: %w", category, err)
	}
	var out []db.ACCValue
	for _, r := range rows {
		refs := make([]string, 20)
		for i := range refs {
			refs[i] = get(r, i+4)
		}
		out = append(out, db.ACCValue{
			Value: get(r, 0), Desc: get(r, 1), Parent: get(r, 2), Status: get(r, 3),
			Refs: refs,
		})
	}
	return out, nil
}

// 与本地 db/acc.go 保持同形的内部中间结构；不导出，避免把 SQL 细节带出 live 包。
type accHeader struct {
	category, name, module, system, status string
	tree, showLines                        bool
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

func isY(s string) bool { return strings.EqualFold(strings.TrimSpace(s), "Y") }

func onlyDigits(s string) bool {
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

func numLit(s string) string {
	if onlyDigits(s) {
		return s
	}
	return lit(s)
}
