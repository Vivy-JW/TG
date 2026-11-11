package gp

import (
	"database/sql"
	"os"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	dictlib "tt/internal/dict"
	"tt/internal/output"
)

type tableInfo struct {
	Name        string `json:"name"`
	NameCN      string `json:"nameCn,omitempty"`
	Module      string `json:"module,omitempty"`
	TableType   string `json:"tableType,omitempty"`
	ColumnCount int    `json:"columnCount"`
	Remark      string `json:"remark,omitempty"`
}

type columnInfo struct {
	Seq       int    `json:"seq"`
	Name      string `json:"name"`
	NameCN    string `json:"nameCn,omitempty"`
	Type      string `json:"type,omitempty"`
	Length    int    `json:"length,omitempty"`
	Precision int    `json:"precision,omitempty"`
	Scale     int    `json:"scale,omitempty"`
	Nullable  string `json:"nullable,omitempty"`
	Default   string `json:"default,omitempty"`
	Remark    string `json:"remark,omitempty"`
	Format    string `json:"format,omitempty"`
}

type indexInfo struct {
	Name       string   `json:"name"`
	Unique     bool     `json:"unique"`
	UniqueRaw  string   `json:"uniqueRaw,omitempty"`
	Columns    []string `json:"columns"`
	ColumnList string   `json:"-"`
}

type constraintInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type,omitempty"`
	Status     string   `json:"status,omitempty"`
	Validated  string   `json:"validated,omitempty"`
	Columns    []string `json:"columns"`
	ColumnList string   `json:"-"`
}

type tableResult struct {
	Table       tableInfo        `json:"table"`
	Columns     []columnInfo     `json:"columns"`
	Indexes     []indexInfo      `json:"indexes"`
	Constraints []constraintInfo `json:"constraints"`
}

// briefTableResult 是 --brief 的 JSON 形状：只留表名和字段骨架，
// 不把备注、格式、索引、约束这些长内容塞给调用方。
type briefTableResult struct {
	Table   briefTableInfo    `json:"table"`
	Columns []briefColumnInfo `json:"columns"`
}

type briefTableInfo struct {
	Name        string `json:"name"`
	NameCN      string `json:"nameCn,omitempty"`
	Module      string `json:"module,omitempty"`
	TableType   string `json:"tableType,omitempty"`
	ColumnCount int    `json:"columnCount"`
}

type briefColumnInfo struct {
	Seq      int    `json:"seq"`
	Name     string `json:"name"`
	NameCN   string `json:"nameCn,omitempty"`
	Type     string `json:"type,omitempty"`
	Length   int    `json:"length,omitempty"`
	Nullable string `json:"nullable,omitempty"`
}

func newTableCmd() *cobra.Command {
	var brief bool
	cmd := &cobra.Command{
		Use:     "table <表名>",
		Aliases: []string{"p_zta"},
		Short:   "p_zta：查表结构（本地 GP 字典）",
		Long: `p_zta 的“表结构”查询（本地 GP 字典）。

  tt gp table zta_file --env 22主机
  tt gp p_zta zta_file --env 22主机
  tt gp table zta_file --env 22主机 --brief   # 只看字段名/类型/可空

数据来自 GP 字典同步生成的 <dictDir>/<环境>.db：
  t_table / t_column / t_index / t_constraint

注意：p_zta 查的是“表”，不是作业。abmi600 是作业，应该用
tt gp link abmi600 查它落在哪个模块。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			setHumanDefault(cmd)
			return runTable(args[0], brief)
		},
	}
	cmd.Flags().BoolVar(&brief, "brief", false, "精简输出：字段只留序号/字段名/中文名/类型/长度/可空，不输出备注、格式、索引和约束")
	return cmd
}

func runTable(name string, brief bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return output.Errorf("USAGE", 2, "表名不能为空")
	}
	t, err := resolveTarget(true)
	if err != nil {
		return err
	}
	t.local = true
	t.meta.Source = "local"
	t.meta.Dialect = "sqlite"
	t.meta.Target = ""
	t.meta.ViaTunnel = false
	t.meta.LocalDBPath = dictlib.DictPath(t.env)
	if _, err := os.Stat(t.meta.LocalDBPath); err != nil {
		return envErr(t, "NOT_SYNCED", 3, "环境 %q 还没有本地 GP 字典：%s", t.env.Name, t.meta.LocalDBPath).
			WithHint("先在设置 → 数据字典同步该环境，或运行 tt serve 后在页面同步")
	}

	db, err := sql.Open("sqlite", t.meta.LocalDBPath)
	if err != nil {
		return envErr(t, "LOCAL_DB", 3, "打开本地 GP 字典失败: %v", err)
	}
	defer db.Close()

	res := tableResult{}
	var colCount int
	err = db.QueryRow(`
SELECT COALESCE(tab,''), COALESCE(tab_cn,''), COALESCE(module,''),
       COALESCE(tab_type,''), COALESCE(col_count,0), COALESCE(remark,'')
  FROM t_table
 WHERE lower(tab) = lower(?)
 LIMIT 1`, name).Scan(
		&res.Table.Name, &res.Table.NameCN, &res.Table.Module,
		&res.Table.TableType, &colCount, &res.Table.Remark,
	)
	if err == sql.ErrNoRows {
		return envErr(t, "NOT_FOUND", 4, "本地 GP 字典里没有表 %q", name).
			WithHint("确认表名；p_zta 查表，作业代号用 tt gp link <作业>")
	}
	if err != nil {
		return envErr(t, "LOCAL_DB", 3, "查询表 %q 失败: %v", name, err)
	}

	cols, err := db.Query(`
SELECT COALESCE(seq,0), COALESCE(col,''), COALESCE(col_cn,''),
       COALESCE(dtype,''), COALESCE(dlen,0), COALESCE(dprec,0),
       COALESCE(dscale,0), COALESCE(nullable,''), COALESCE(dflt,''),
       COALESCE(remark,''), COALESCE(fmt,'')
  FROM t_column
 WHERE lower(tab) = lower(?)
 ORDER BY seq, col`, name)
	if err != nil {
		return envErr(t, "LOCAL_DB", 3, "查询字段失败: %v", err)
	}
	for cols.Next() {
		var c columnInfo
		if err := cols.Scan(&c.Seq, &c.Name, &c.NameCN, &c.Type,
			&c.Length, &c.Precision, &c.Scale, &c.Nullable,
			&c.Default, &c.Remark, &c.Format); err != nil {
			cols.Close()
			return envErr(t, "LOCAL_DB", 3, "读取字段失败: %v", err)
		}
		res.Columns = append(res.Columns, c)
	}
	if err := cols.Err(); err != nil {
		cols.Close()
		return envErr(t, "LOCAL_DB", 3, "读取字段失败: %v", err)
	}
	cols.Close()

	idx, err := db.Query(`
SELECT COALESCE(idx,''), COALESCE(uniqueness,''), COALESCE(cols,'')
  FROM t_index
 WHERE lower(tab) = lower(?)
 ORDER BY idx`, name)
	if err != nil {
		return envErr(t, "LOCAL_DB", 3, "查询索引失败: %v", err)
	}
	for idx.Next() {
		var x indexInfo
		var raw string
		if err := idx.Scan(&x.Name, &raw, &x.ColumnList); err != nil {
			idx.Close()
			return envErr(t, "LOCAL_DB", 3, "读取索引失败: %v", err)
		}
		x.UniqueRaw = raw
		x.Unique = strings.EqualFold(strings.TrimSpace(raw), "unique")
		x.Columns = splitCols(x.ColumnList)
		res.Indexes = append(res.Indexes, x)
	}
	if err := idx.Err(); err != nil {
		idx.Close()
		return envErr(t, "LOCAL_DB", 3, "读取索引失败: %v", err)
	}
	idx.Close()

	cons, err := db.Query(`
SELECT COALESCE(cons,''), COALESCE(ctype,''), COALESCE(status,''),
       COALESCE(validated,''), COALESCE(cols,'')
  FROM t_constraint
 WHERE lower(tab) = lower(?)
 ORDER BY cons`, name)
	if err != nil {
		return envErr(t, "LOCAL_DB", 3, "查询约束失败: %v", err)
	}
	for cons.Next() {
		var c constraintInfo
		if err := cons.Scan(&c.Name, &c.Type, &c.Status, &c.Validated, &c.ColumnList); err != nil {
			cons.Close()
			return envErr(t, "LOCAL_DB", 3, "读取约束失败: %v", err)
		}
		c.Columns = splitCols(c.ColumnList)
		res.Constraints = append(res.Constraints, c)
	}
	if err := cons.Err(); err != nil {
		cons.Close()
		return envErr(t, "LOCAL_DB", 3, "读取约束失败: %v", err)
	}
	cons.Close()

	res.Table.ColumnCount = len(res.Columns)
	if res.Table.ColumnCount == 0 {
		res.Table.ColumnCount = colCount
	}

	meta := metaNotes(t.meta,
		"索引 "+strconv.Itoa(len(res.Indexes))+"，约束 "+strconv.Itoa(len(res.Constraints)))
	meta.TotalRows = len(res.Columns)
	meta.Returned = len(res.Columns)

	if brief {
		meta.Notes = append(meta.Notes, "brief：索引/约束明细已省略")
		b := briefTableResult{
			Table: briefTableInfo{
				Name:        res.Table.Name,
				NameCN:      res.Table.NameCN,
				Module:      res.Table.Module,
				TableType:   res.Table.TableType,
				ColumnCount: res.Table.ColumnCount,
			},
			Columns: briefColumns(res.Columns),
		}
		return output.Emit(stdout, output.Options{
			Format:  common.OutputFormat(),
			Meta:    meta,
			Data:    b,
			Columns: []string{"序号", "字段", "中文名", "类型", "长度", "可空"},
			Rows:    briefRows(res.Columns),
		})
	}

	return output.Emit(stdout, output.Options{
		Format: common.OutputFormat(),
		Meta:   meta,
		Data:   res,
		Columns: []string{
			"序号", "字段", "中文名", "类型", "长度", "精度", "小数", "可空", "默认", "备注", "格式",
		},
		Rows: tableRows(res.Columns),
	})
}

func splitCols(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func tableRows(cols []columnInfo) [][]string {
	out := make([][]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, []string{
			strconv.Itoa(c.Seq), c.Name, c.NameCN, c.Type,
			itoa(c.Length), itoa(c.Precision), itoa(c.Scale),
			c.Nullable, c.Default, oneLine(c.Remark), c.Format,
		})
	}
	return out
}

func briefColumns(cols []columnInfo) []briefColumnInfo {
	out := make([]briefColumnInfo, 0, len(cols))
	for _, c := range cols {
		out = append(out, briefColumnInfo{
			Seq: c.Seq, Name: c.Name, NameCN: c.NameCN,
			Type: c.Type, Length: c.Length, Nullable: c.Nullable,
		})
	}
	return out
}

func briefRows(cols []columnInfo) [][]string {
	out := make([][]string, 0, len(cols))
	for _, c := range cols {
		out = append(out, []string{
			strconv.Itoa(c.Seq), c.Name, c.NameCN, c.Type,
			itoa(c.Length), c.Nullable,
		})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// oneLine 把字典备注里的换行压成空格。备注原文常带 \r\n（如 zta01 的
// “Table #...\r\nEx:...”），直接进 tabwriter 会被当成新行，把第一列撑宽。
func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
