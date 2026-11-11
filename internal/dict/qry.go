package dict

import (
	"regexp"
	"strings"
)

// QryDef 是一次开窗（动态查询画面）的完整定义，来自 gab_file 一行 + gac_file 若干行。
//
// 字段含义以 GP 的 42f 标签为准（T100 的 sdd 已过时）：
//
//	gab01 开窗代号（运行期塞进 g_qryparam.form，程序里写 LET g_qryparam.form = "cp_ima"）
//	gab02 Input Mode WHERE     —— 一般模式的查询条件片段，arg1~arg9 会在生成时替换
//	gab03 Joint Tables         —— 逗号分隔，要额外 join 的表
//	gab04 Using Unique Setting —— 'Y' 时 SELECT 加 UNIQUE
//	gab05 Construct Mode WHERE —— 开窗（CONSTRUCT）模式下换用这段条件
//	gab06 Memo
//	gab07 Hard Coding Program  —— 'Y' 表示 SQL 由程序硬写，不由 gac_file 拼
//	gab11 Customerlized        —— 'C'/'Y' 为客制版本
type QryDef struct {
	Code        string
	Title       string
	InputWhere  string
	JoinTables  string
	Distinct    bool
	ConstrWhere string
	Memo        string
	HardCode    bool
	Cust        bool
	Cols        []QryCol
}

// QryCol 是 gac_file 的一行：开窗里的一个显示栏位。
//
//	gac02 序号（-1 是 title 行，不参与查询）
//	gac05 Table / gac06 Field
//	gac07 Type（0=Edit 1=CheckBox 2=DateEdit 3=ComboBox）
//	gac09 Width / gac10 Return（'Y' 表示回传给调用程序，也用来定 ORDER BY）
//	gac11 Outer（'Y' 表示该表要走外连接）
//	gac13 Input Constraint（遮罩；Type=3 时是 ComboBox 的取值来源）
//	gac15 Connect to（串查的下一段开窗代号）
type QryCol struct {
	No         int
	Table      string
	Field      string
	Label      string
	Width      int
	Return     bool
	Type       int
	Outer      bool
	Constraint string
	ConnectTo  string
	// DateType 由调用方按 schema 的列类型填：Oracle 下 DateEdit 栏位要包 to_char 才能查。
	DateType bool
}

// QrySQLOpt 是拼 SQL 时的运行期参数。
type QrySQLOpt struct {
	// Schema 是资料库编号（azp03，通常等于 Oracle schema 名）。非空时给表名加前缀。
	Schema string
	// Construct 为 true 时用 gab05（CONSTRUCT 模式），否则用 gab02。
	Construct bool
	// Args 是 arg1~arg9 的代入值，条件片段里出现 arg1 就换成 Args[0]。
	Args [9]string
	// Where 是调用方附加的查询条件（已经带上比较表达式），会括起来 AND 进去。
	Where string
	// OrderBy 覆盖默认排序（默认按 Return 栏位）。
	OrderBy string
	// DateFormat 是 Oracle 下 to_char 的格式，空则 "yy/mm/dd"。
	DateFormat string
	// Oracle 为 false 时不做 to_char、不做 OUTER 字段补表名。
	Oracle bool
}

var closeBracketRe = regexp.MustCompile(`\s+`)

// BuildSQL 按 T100 cl_create_qry() 的规则，用 gab_file + gac_file 拼出这条开窗的 SELECT。
//
// 对应 4GL 的 cl_qry_prep_result_set()：
//
//	SELECT [UNIQUE] 'N',<gac06 列> FROM <gac05 去重>,<gab03 join>[, OUTER <gac11='Y' 的表>]
//	 WHERE (<gab02 或 gab05，arg1~9 已代入>) [AND (<调用方条件>)] [AND <CONSTRUCT 条件>]
//	 ORDER BY <gac10='Y' 的栏位>
//
// 与 4GL 的差异（有意为之）：
//   - 4GL 的 ms_cons_where 是画面上用户输入的每个栏位条件，这里由调用方用 Where 传进来；
//   - 4GL 用 sch_file 判断栏位是否日期型，这里用 QryCol.DateType（调用方从 t_column 拿）；
//   - 4GL 会再套一层权限过滤条件（cl_get_extra_cond_for_qry），GP 没有对应表，不实现。
func (d QryDef) BuildSQL(opt QrySQLOpt) string {
	dateFmt := opt.DateFormat
	if dateFmt == "" {
		dateFmt = "yy/mm/dd"
	}

	var cols []string
	var tables []string
	var outer []string
	seenTable := map[string]bool{}
	seenOuter := map[string]bool{}

	for _, c := range d.Cols {
		if c.No == -1 || c.Field == "" {
			continue // title 行不参与查询
		}
		expr := c.Field
		if opt.Oracle && c.Type == 2 && c.DateType {
			expr = "to_char(" + c.Field + ",'" + dateFmt + "')"
		}
		cols = append(cols, expr)

		t := strings.TrimSpace(c.Table)
		if t == "" || strings.EqualFold(t, "formonly") {
			continue
		}
		if c.Outer {
			if !seenOuter[t] {
				seenOuter[t] = true
				outer = append(outer, t)
			}
			continue
		}
		if !seenTable[t] {
			seenTable[t] = true
			tables = append(tables, t)
		}
	}

	var b strings.Builder
	b.WriteString("SELECT ")
	if d.Distinct {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("'N'")
	if len(cols) > 0 {
		b.WriteString(",")
		b.WriteString(strings.Join(cols, ","))
	}
	b.WriteString(" FROM ")
	b.WriteString(strings.Join(prefixTables(tables, opt.Schema), ","))

	// gab03 的 join 表
	for _, t := range splitTrim(d.JoinTables) {
		b.WriteString(",")
		b.WriteString(tableRef(t, opt.Schema))
	}
	for _, t := range outer {
		if opt.Oracle {
			// Oracle 没有 OUTER 关键字，外连接改用 WHERE 里的 (+) 标记（见 qualifyOuter）
			b.WriteString(",")
		} else {
			b.WriteString(", OUTER ")
		}
		b.WriteString(tableRef(t, opt.Schema))
	}

	conds := d.conditions(opt)
	b.WriteString(" WHERE ")
	if len(conds) == 0 {
		b.WriteString("1=1")
	} else {
		b.WriteString(strings.Join(conds, " AND "))
	}

	order := opt.OrderBy
	if order == "" {
		var ret []string
		for _, c := range d.Cols {
			if c.No == -1 || !c.Return || c.Field == "" {
				continue
			}
			ret = append(ret, c.Field)
		}
		order = strings.Join(ret, ",")
	}
	if order != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(order)
	}

	sql := strings.TrimSpace(closeBracketRe.ReplaceAllString(b.String(), " "))
	if opt.Oracle && len(outer) > 0 {
		sql = qualifyOuter(sql, outer, opt.Schema)
	}
	return sql
}

// conditions 拼 WHERE 里的各个条件片段（顺序与 4GL 一致：调用方条件、gab 条件、CONSTRUCT 条件）。
func (d QryDef) conditions(opt QrySQLOpt) []string {
	var conds []string
	if w := strings.TrimSpace(opt.Where); w != "" {
		conds = append(conds, "( "+w+" )")
	}
	base := d.InputWhere
	if opt.Construct {
		base = d.ConstrWhere
	}
	if w := strings.TrimSpace(substArgs(base, opt.Args)); w != "" {
		conds = append(conds, "( "+w+" )")
	}
	return conds
}

var argRe = regexp.MustCompile(`arg([1-9])`)

// substArgs 把条件片段里的 arg1~arg9 换成实参（对应 4GL 的 cl_replace_str_by_index 循环）。
func substArgs(s string, args [9]string) string {
	if s == "" {
		return s
	}
	return argRe.ReplaceAllStringFunc(s, func(m string) string {
		i := int(m[len(m)-1] - '1')
		return args[i]
	})
}

func splitTrim(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func tableRef(t, schema string) string {
	if schema == "" || strings.Contains(t, ".") {
		return t
	}
	return schema + "." + t
}

func prefixTables(tables []string, schema string) []string {
	out := make([]string, len(tables))
	for i, t := range tables {
		out[i] = tableRef(t, schema)
	}
	return out
}

// qualifyOuter 对应 4GL 的 cl_qry_replace_sql：WHERE 里凡引用外连接表的栏位都要带 (+)，否则 Oracle 不认；
// 裸字段名（ima01）先补表名，已经是 ima_file.ima01 这种写法的只补 (+)。
func qualifyOuter(sql string, outer []string, schema string) string {
	i := strings.Index(sql, " WHERE ")
	j := strings.Index(sql, " ORDER BY ")
	if i < 0 || j < i {
		return sql
	}
	where := sql[i+7 : j]

	bases := make(map[string]string, len(outer)) // 字段前缀 -> 表引用
	refs := make(map[string]bool, len(outer))    // 外连接表的引用名（含 schema 前缀）
	for _, t := range outer {
		ref := tableRef(t, schema)
		refs[strings.ToLower(ref)] = true
		refs[strings.ToLower(t)] = true
		refs[strings.ToLower(t2s(t))] = true
		if base := strings.TrimSuffix(t, "_file"); base != "" {
			bases[base] = ref
		}
	}

	var b strings.Builder
	for k := 0; k < len(where); {
		if !isIdentStart(where[k]) {
			b.WriteByte(where[k])
			k++
			continue
		}
		s := k
		for k < len(where) && isIdentPart(where[k]) {
			k++
		}
		ident := where[s:k]
		if s > 0 && where[s-1] == '.' && refs[strings.ToLower(prevIdent(where, s-1))] {
			b.WriteString(ident)
			b.WriteString("(+)")
			continue
		}
		if qualify := bareOuterField(ident, bases); qualify != "" {
			b.WriteString(qualify)
			b.WriteByte('.')
			b.WriteString(ident)
			b.WriteString("(+)")
			continue
		}
		b.WriteString(ident)
	}
	return sql[:i+7] + b.String() + sql[j:]
}

// t2s 把 schema 前缀去掉，得到裸表名（"cof_demo.ima_file" -> "ima_file"）。
func t2s(t string) string {
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

// bareOuterField 判断 ident 是不是外连接表里的字段（前缀 + 数字），是则返回该表的引用名。
func bareOuterField(ident string, bases map[string]string) string {
	best := ""
	for base := range bases {
		if len(ident) > len(base) && strings.HasPrefix(ident, base) &&
			ident[len(base)] >= '0' && ident[len(base)] <= '9' && len(base) > len(best) {
			best = base
		}
	}
	if best == "" {
		return ""
	}
	return bases[best]
}

// prevIdent 从 end（指向 "."）往前取一个标识符。
func prevIdent(s string, end int) string {
	i := end
	for i > 0 && isIdentPart(s[i-1]) {
		i--
	}
	return s[i:end]
}

func isIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9') || c == '$' || c == '#'
}
