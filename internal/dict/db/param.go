package db

import "fmt"

// 参数定义档查询(gzsz_t + gzszl_t):系统参数由 azzi990(参数资料定义作业)维护、
// 单据别参数由 azzi991 维护 —— 同一张定义表,仅参数群(gzsz001)不同:
//   azzi990 视域:gzsz001 != 'ooac_t'(A/E/S 级,值表 gzsa_t/ooaa_t/ooab_t)
//   azzi991 视域:gzsz001 =  'ooac_t'(单据别 D 级,值表 ooac_t;子表 gzsy_t 记绑定单据性质)
// 参数编号(gzsz002)形如 A-SYS-0100 / E-CIR-0001 / D-MFG-0076(<型态1码>-<领域3码>-<4位流水>)。
// 本查询只读"定义与说明";参数当前值在客户化值表(gzsa_t 等),不在本包范围。

// ParamDefRow 一条参数定义(每语言一行,来自 gzszl_t)。
type ParamDefRow struct {
	Code      string `json:"编号"`    // gzsz002
	Group     string `json:"参数群"`   // gzsz001 (值表名+_t)
	GroupName string `json:"群名称"`   // dzeal003 (该语言;未注册时空)
	Lang      string `json:"语言"`    // gzszl003
	Name      string `json:"名称"`    // gzszl004
	Desc      string `json:"说明"`    // gzszl005
	Desc2     string `json:"备注一"`   // gzszl006 (长文本页签)
	Desc3     string `json:"备注二"`   // gzszl007 (长文本页签)
	TypeCode  string `json:"型态码"`   // gzsz003 SCC'89': 1=Y/N 2=整数选项 3=范围设定 4=字符或SCC 5=日期
	AreaCode  string `json:"领域码"`   // gzsz011 SCC'90'(BAS/CIR/.../B+行业)
	Default   string `json:"预设值"`   // gzsz008
	RangeVal  string `json:"值域"`    // gzsz009
	DateFmt   string `json:"日期格式"`  // gzsz015
	SccCode   string `json:"SCC代号"` // gzsz016 (型态4字符时的选项清单)
	RVCode    string `json:"校核带值"`  // gzsz013 (r.v dzcd001)
	RQCode    string `json:"开窗程序"`  // gzsz014 (r.q dzca001)
	Except    string `json:"异常处理码"` // gzsz017 SCC'160'
	Freq      string `json:"修改频度"`  // gzsz018 SCC'258' Y=未限制/N=配置后不可修改
	Live      string `json:"即时抓取"`  // gzsz019 Y/N
	ValueProg string `json:"值维护作业"` // gzsz004 (单据别参数恒 aooi200)
	Seq       string `json:"行序"`    // gzsz005
	Status    string `json:"状态"`    // gzszstus Y/N
}

// QueryParam 返回指定参数编号的全部语言行(按群、语言排序)。编号跨群出现时
// (gzsz002 全表唯一检查,理论上不会)返回全部命中行,由命令层按视域过滤。
// 群名称(dzeal_t)单独查询:镜像库未同步 dzeal_t 时静默降级为空,不阻断主结果。
func (d *DB) QueryParam(code string) ([]ParamDefRow, error) {
	rows, err := d.conn.Query(`
		SELECT t0.gzsz002, t0.gzsz001,
		       COALESCE(t0.gzszstus, ''), COALESCE(t0.gzsz011, ''),
		       COALESCE(t0.gzsz003, ''), COALESCE(t0.gzsz008, ''),
		       COALESCE(t0.gzsz009, ''), COALESCE(t0.gzsz015, ''),
		       COALESCE(t0.gzsz016, ''), COALESCE(t0.gzsz013, ''),
		       COALESCE(t0.gzsz014, ''), COALESCE(t0.gzsz017, ''),
		       COALESCE(t0.gzsz018, ''), COALESCE(t0.gzsz019, ''),
		       COALESCE(t0.gzsz004, ''), COALESCE(t0.gzsz005, ''),
		       COALESCE(l.gzszl003, ''), COALESCE(l.gzszl004, ''),
		       COALESCE(l.gzszl005, ''), COALESCE(l.gzszl006, ''),
		       COALESCE(l.gzszl007, '')
		FROM gzsz_t t0
		LEFT JOIN gzszl_t l ON l.gzszl001 = t0.gzsz001 AND l.gzszl002 = t0.gzsz002
		WHERE t0.gzsz002 = ? ORDER BY t0.gzsz001, l.gzszl003`, code)
	if err != nil {
		return nil, fmt.Errorf("query param %s: %w", code, err)
	}
	defer rows.Close()

	var result []ParamDefRow
	groups := map[string]bool{}
	for rows.Next() {
		var r ParamDefRow
		if err := rows.Scan(&r.Code, &r.Group, &r.Status, &r.AreaCode,
			&r.TypeCode, &r.Default, &r.RangeVal, &r.DateFmt, &r.SccCode,
			&r.RVCode, &r.RQCode, &r.Except, &r.Freq, &r.Live,
			&r.ValueProg, &r.Seq, &r.Lang, &r.Name, &r.Desc, &r.Desc2,
			&r.Desc3); err != nil {
			return nil, fmt.Errorf("scan param row: %w", err)
		}
		if r.Group != "" {
			groups[r.Group] = true
		}
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return result, nil
	}

	// 群名称(dzea_t/dzeal_t 的表说明):缺表静默降级
	names, err := d.paramGroupNames(groups)
	if err != nil {
		names = map[string]string{}
	}
	for i := range result {
		result[i].GroupName = names[result[i].Group+"\x00"+result[i].Lang]
	}
	return result, nil
}

// paramGroupNames 查参数群名称(表说明):key = "<群>\x00<语言>"。
func (d *DB) paramGroupNames(groups map[string]bool) (map[string]string, error) {
	if len(groups) == 0 {
		return map[string]string{}, nil
	}
	ids := make([]any, 0, len(groups))
	ph := ""
	for g := range groups {
		ids = append(ids, g)
		ph += "?,"
	}
	ph = ph[:len(ph)-1]
	rows, err := d.conn.Query(`
		SELECT al.dzeal001, COALESCE(al.dzeal002, ''), COALESCE(al.dzeal003, '')
		FROM dzeal_t al WHERE al.dzeal001 IN (`+ph+`)`, ids...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var g, lang, name string
		if err := rows.Scan(&g, &lang, &name); err != nil {
			return nil, err
		}
		out[g+"\x00"+lang] = name
	}
	return out, rows.Err()
}

// DocTypeRow 单据别参数绑定的单据性质行(gzsy_t;gzsy001='ooac_t')。
type DocTypeRow struct {
	DocType  string `json:"单据性质"` // gzsy004 (SCC'24' 代码,如 aapt110)
	DocName  string `json:"性质名称"` // gzcbl004 (指定语言)
	Module   string `json:"模块"`   // gzsy003 (gzsy004 前 3 码大写)
	Generate string `json:"已抛转"`  // gzsy005 Y/N
}

// QueryDocTypes 返回单据别参数(code)绑定的单据性质(名称按指定语言)。
func (d *DB) QueryDocTypes(code, lang string) ([]DocTypeRow, error) {
	rows, err := d.conn.Query(`
		SELECT COALESCE(g.gzsy004, ''), COALESCE(g.gzsy003, ''),
		       COALESCE(g.gzsy005, ''), COALESCE(b.gzcbl004, '')
		FROM gzsy_t g
		LEFT JOIN gzcbl_t b ON b.gzcbl001 = '24' AND b.gzcbl002 = g.gzsy004 AND b.gzcbl003 = ?
		WHERE g.gzsy001 = 'ooac_t' AND g.gzsy002 = ? ORDER BY g.gzsy004`, lang, code)
	if err != nil {
		return nil, fmt.Errorf("query doc types %s: %w", code, err)
	}
	defer rows.Close()

	var result []DocTypeRow
	for rows.Next() {
		var r DocTypeRow
		if err := rows.Scan(&r.DocType, &r.Module, &r.Generate, &r.DocName); err != nil {
			return nil, fmt.Errorf("scan doc type: %w", err)
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
