package dict

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/output"
)

// tt dict spill —— 读「结果被截断时落盘的那一份完整结果」。
//
// 为什么有这条命令（2026-09-25，§11.9 的 Go 侧清单）：`emitCapped` 在截断**之前**会把完整结果
// 落一份到 <配置目录>/spill/，并在信封的 notes 里给出路径 —— 那份数据**已经在盘上了**。
// 可从前"要看全量"只有两条路：重跑一次查询（远程库上可能很贵，而且结果可能已经变了），
// 或者自己 cat/jq 那个文件（得先知道它是 JSON、还得自己数行）。于是"已经躺在盘上的结果"
// 反而没有入口。本命令就是那个入口：**不重查、不连数据源**，只按页读盘上那一份。
//
// 与查询共用同一套上限语义：`--limit` 仍是"本次最多返回多少条"（0 = 不限，
// 缺省跟随 config.json 的 query.limit），`--offset` 是"从第几条开始"。

func init() {
	Group.AddCommand(spillCmd())
}

func spillCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "spill",
		Short: "查看被截断查询落盘的完整结果（截断时信封里的 localPath）",
		Long: `查询结果超过返回上限时，完整的那一份**先落盘、再截断**（<配置目录>/spill/），
截断后的信封里给 totalRows 与 localPath。本命令是那个 localPath 的入口：
不重跑查询、不连数据源，只把已经躺在盘上的那一份按页读出来。

子命令:
  list                                       列出落盘过的结果（新 → 旧）
  show <文件|latest> [--offset N] [--limit M] 读一份；默认从第 0 条起，
                                             返回条数跟随 --limit / config.json query.limit

要看全量用 --limit 0（与查询那边同一套上限语义）。`,
		Example: `  tt dict spill list
  tt dict spill show latest
  tt dict spill show latest --offset 20 --limit 20
  tt dict spill show latest --limit 0        # 全量`,
		// 只读文件，不碰数据源：与 mirror/db 一样覆盖组的查询钩子。
		PersistentPreRunE: skipQuerySource,
	}
	c.AddCommand(spillListCmd(), spillShowCmd())
	return c
}

// spillDir 是落盘目录 <配置目录>/spill。
//
// 与 spillResult（root.go）用同一套解析：配置路径 + 固定子目录名，不另立一份。
// 配置读不到时报错 —— 那说明连落盘的地方都没有，而这里的调用方正要去找那些文件。
func spillDir() (string, error) {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("定位不到配置目录（--config / TT_CONFIG / 用户目录都读不到）")
	}
	return filepath.Join(filepath.Dir(path), spillDirName), nil
}

// emitSpill 按查询**同一个信封**输出，但出处写 "spill"。
//
// 为什么不复用 srcMeta()：这一条命令没有连任何数据源（PreRun 就跳过了），套上"哪个环境、
// 哪个账号"会让人以为它刚查过库 —— 而它读的是盘上那份**当时**的结果。`source: "spill"`
// 是诚实的出处，`localPath` 则让调用方可以再翻页而不必重新解析路径。
func emitSpill(m output.Meta, data any, columns []string, rows [][]string) error {
	if m.Source == "" {
		m.Source = "spill"
	}
	return output.Emit(os.Stdout, output.Options{
		Format: Format(), Meta: m, Data: data, Columns: columns, Rows: rows,
	})
}

// ---------------------------------------------------------------- list

type spillFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	Time string `json:"time"`
	// mod 是**排序用的原始时间**，不进 JSON：Time 只有秒精度，同一秒内落的两份文件
	// 按字符串排会变成"看 ReadDir 的心情"，而 "latest" 正是靠这个顺序 —— 实测
	// 2026-09-25 被自己的测试抓出来（两个文件同一秒，latest 取错了那份）。
	mod time.Time `json:"-"`
}

func spillListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出落盘过的结果（新 → 旧）",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := spillDir()
			if err != nil {
				return err
			}
			files, err := readSpillDir(dir)
			if err != nil {
				return err
			}
			m := output.Meta{}
			if len(files) == 0 {
				m.Notes = append(m.Notes,
					"还没有落盘过——只有查询被截断时才会落盘（"+dir+" 里没有文件）")
				return emitSpill(m, []spillFile{}, nil, nil)
			}
			m.TotalRows, m.Returned = len(files), len(files)
			m.Notes = append(m.Notes,
				fmt.Sprintf("%d 份；看某一份：tt dict spill show <名字|latest>（目录 %s）", len(files), dir))
			cols := []string{"文件", "字节", "时间"}
			rows := make([][]string, 0, len(files))
			for _, f := range files {
				rows = append(rows, []string{f.Name, fmt.Sprint(f.Size), f.Time})
			}
			return emitSpill(m, files, cols, rows)
		},
	}
}

// readSpillDir 读落盘目录，新 → 旧。
//
// 目录不存在不算错：那只是"从没截断过"，而一条把"没有"报成失败的 list 会让第一次用它的
// 人以为环境坏了。
func readSpillDir(dir string) ([]spillFile, error) {
	ents, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	out := make([]spillFile, 0, len(ents))
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, spillFile{
			Name: e.Name(),
			Path: filepath.Join(dir, e.Name()),
			Size: info.Size(),
			Time: info.ModTime().Format("2006-01-02 15:04:05"),
			mod:  info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		// 先按落盘时间（真正的时间，不是格式化到秒的字符串）。
		if !out[i].mod.Equal(out[j].mod) {
			return out[i].mod.After(out[j].mod)
		}
		// 时间戳**完全相等**时再按文件名排：实测背靠背写的两个文件 mtime 会一样
		//（自己的测试就是这么抓住"latest 取错了一份"的），而文件名本身带着秒级时间戳
		//（query-20260101-120000.json），所以这个次序与人的直觉一致。
		return out[i].Name > out[j].Name
	})
	return out, nil
}

// ---------------------------------------------------------------- show

func spillShowCmd() *cobra.Command {
	var offset int
	c := &cobra.Command{
		Use:   "show <文件|latest>",
		Short: "读一份落盘结果（可翻页）",
		Long: `读一份落盘结果。参数可以是 spill list 里的文件名、完整路径，或 "latest"（最新那份）。

--offset 从第几条开始（默认 0）；返回条数用组的 --limit（缺省跟随 config.json 的
query.limit；0 = 不限）。列序按字母序：落盘的是**数据**，原始列序不在那个文件里。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveSpillPath(args[0])
			if err != nil {
				return err
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("读不到 %s：%w", path, err)
			}
			var data any
			if err := json.Unmarshal(raw, &data); err != nil {
				return fmt.Errorf("%s 不是合法 JSON：%w", path, err)
			}
			if offset < 0 {
				return fmt.Errorf("--offset 不能是负数：%d", offset)
			}

			total := spillLen(data)
			page, from, to := spillPage(data, offset, limitOf())
			if offset > 0 && offset >= total {
				return fmt.Errorf("--offset %d 越界：这份结果共 %d 条（从 0 数起）", offset, total)
			}

			m := output.Meta{
				LocalPath: path,
				TotalRows: total,
				Returned:  spillLen(page),
				Offset:    from,
				Truncated: to < total,
			}
			if m.Truncated {
				m.TruncReason = "spill-page"
				// 说清三件事：这一页从哪起、几条、下一条的 offset 是多少。
				// 不写"第 N–M 条"那种区间：to 是开区间，写成区间就得解释含不含尾，
				// 而这种说明读的人一定会跳过去（实测：自己都差点把它读成 3 条）。
				m.Notes = append(m.Notes, fmt.Sprintf(
					"这一页：offset=%d，%d 条（共 %d 条）；下一页：tt dict spill show %s --offset %d",
					from, spillLen(page), total, filepath.Base(path), to))
			}
			if total == 1 && reflect.ValueOf(data).Kind() != reflect.Slice {
				m.Notes = append(m.Notes, "这份落盘结果是一个单实体（不是结果集），没有分页可言")
			} else {
				m.Notes = append(m.Notes,
					"列序按字母序（落盘的是数据，原始列序不在文件里）")
			}
			cols, rows := spillRows(page)
			return emitSpill(m, page, cols, rows)
		},
	}
	c.Flags().IntVar(&offset, "offset", 0, "从第几条开始（0 = 从头；与查询的 --limit 配合翻页）")
	return c
}

// resolveSpillPath 把参数解析成落盘文件的绝对路径。
func resolveSpillPath(arg string) (string, error) {
	dir, err := spillDir()
	if err != nil {
		return "", err
	}
	if strings.EqualFold(arg, "latest") {
		files, err := readSpillDir(dir)
		if err != nil {
			return "", err
		}
		if len(files) == 0 {
			return "", fmt.Errorf("%s 里没有落盘结果——只有查询被截断时才会落盘", dir)
		}
		return files[0].Path, nil
	}
	if filepath.IsAbs(arg) || strings.ContainsAny(arg, `/\`) {
		if _, err := os.Stat(arg); err != nil {
			return "", fmt.Errorf("读不到 %s：%w", arg, err)
		}
		return arg, nil
	}
	p := filepath.Join(dir, arg)
	if _, err := os.Stat(p); err != nil {
		return "", fmt.Errorf("读不到 %s：%w（先 tt dict spill list 看有哪些）", p, err)
	}
	return p, nil
}

// spillLen 是这份载荷有多少"条" —— 与 root.go 的 payloadLen 同一个判据（slice 按元素数，
// 其余算一条），因为翻页与截断必须是同一种数法，否则 --limit 5 在两边会给出不同的页。
func spillLen(v any) int { return payloadLen(v) }

// spillPage 取第 offset 条起、最多 n 条（n<=0 = 不限）。
// 返回 (这一页, 起始下标, 结束下标) —— 结束下标是**开区间**，用于告诉调用方下一页从哪开始。
func spillPage(v any, offset, n int) (any, int, int) {
	if v == nil {
		return nil, 0, 0
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return v, 0, 1 // 单实体：没有页，from/to 只为让计数自洽
	}
	total := rv.Len()
	from := offset
	if from > total {
		from = total
	}
	to := total
	if n > 0 && from+n < total {
		to = from + n
	}
	return rv.Slice(from, to).Interface(), from, to
}

// spillRows 从载荷推出一张表（人读表格 / CSV 用）。列名取各元素键的并集并排序 ——
// map 的迭代顺序在 Go 里是随机的，不排序就会让同一份文件两次输出不同的列序。
func spillRows(v any) ([]string, [][]string) {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil, nil
	}
	var items []any
	switch rv.Kind() {
	case reflect.Slice:
		items = make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			items = append(items, rv.Index(i).Interface())
		}
	default:
		items = []any{v}
	}
	if len(items) == 0 {
		return nil, nil
	}

	seen := map[string]bool{}
	var cols []string
	objs := make([]map[string]any, len(items))
	allObj := true
	for i, it := range items {
		o, ok := it.(map[string]any)
		if !ok {
			allObj = false
			break
		}
		objs[i] = o
		for k := range o {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}
	if !allObj {
		// 不是"对象的数组"（比如字符串数组）：一列，值原样 toString。
		rows := make([][]string, 0, len(items))
		for _, it := range items {
			rows = append(rows, []string{spillCell(it)})
		}
		return []string{"值"}, rows
	}
	sort.Strings(cols)
	rows := make([][]string, 0, len(objs))
	for _, o := range objs {
		row := make([]string, 0, len(cols))
		for _, c := range cols {
			row = append(row, spillCell(o[c]))
		}
		rows = append(rows, row)
	}
	return cols, rows
}

// spillCell 把一个单元格转成字符串：字符串原样，其余走紧凑 JSON（数字/布尔/嵌套都对得上），
// nil 给空串而不是 "<nil>"。
func spillCell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strings.TrimSuffix(fmt.Sprintf("%v", t), ".0")
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(b)
	}
}
