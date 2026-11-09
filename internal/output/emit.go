package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// Format 输出形态。默认 JSON —— 依据是表格类格式对 LLM 的理解力实测
// (JSON 52.3% / Markdown 表格 51.9% / CSV 44.3%),而 JSON 同时又省掉了解析歧义:
// 引号、换行、分隔符都不用猜。
type Format int

const (
	FormatJSON  Format = iota // 带环境信息的信封(默认)
	FormatCSV                 // `# ` 环境头 + 裸 CSV 主体
	FormatTable               // 人读对齐表格
)

// Options 一次输出的全部输入。
type Options struct {
	Format Format
	Meta   Meta
	// Data 是 JSON 载荷(放进信封的 data)。文本/CSV 模式不用它。
	Data any
	// Columns/Rows 是 CSV 模式的主体;文本模式也用它们渲染人读表格。
	Columns []string
	Rows    [][]string
}

// Emit 唯一的输出出口。w 注入,所以能像 tt debug sql 的 writeSQLResult 那样直接单测 ——
// 这是本仓库唯一可行的输出测试手法(见 emit_test.go)。
func Emit(w io.Writer, opt Options) error {
	switch opt.Format {
	case FormatCSV:
		return WriteCSV(w, opt.Meta, opt.Columns, opt.Rows)
	case FormatTable:
		return WriteTable(w, opt.Columns, opt.Rows)
	default:
		return WriteJSON(w, opt)
	}
}

// envelope --json 的顶层形状:成功标志 + 环境信息(平铺)+ 载荷。
//
// 平铺而不是塞进嵌套的 meta:tt debug sql 的 DBQueryResult 就是平的,且被 web 前端
// 消费 —— 改成嵌套是白改一遍。`ok` 与 tt dev 既有的 {"ok":false,...} 对齐,
// 让只读 JSON、不看退出码的消费者也能判成败。
type envelope struct {
	OK bool `json:"ok"`
	Meta
	Data any `json:"data"`
}

// WriteJSON 输出成功信封。
func WriteJSON(w io.Writer, opt Options) error {
	// JSON 输出一律紧凑单行：这些输出主要给 agent/管道消费，缩进只增加 token，不增加信息。
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // 保留 < > & 原样输出(校验 SQL 含 <field> 等标签)
	return enc.Encode(envelope{OK: true, Meta: opt.Meta, Data: opt.Data})
}

// WriteMeta 只写 `# ` 信息块,不写主体。
// 人读表格模式也要在前面带上它 —— "这份数据从哪来"和格式无关。
func WriteMeta(w io.Writer, m Meta) error {
	for _, ln := range m.headerLines() {
		if _, err := fmt.Fprintf(w, "# %s\n", ln); err != nil {
			return err
		}
	}
	return nil
}

// WriteCSV 写 `# ` 环境头 + 裸 CSV 主体。
//
// 井号**必须带一个空格**:值本身以 `#` 开头是常事(如颜色 #FF0000),
// 少了空格 `grep -v '^# '` 会把数据行一起切掉。
//
// 元信息留在 stdout 是刻意的例外 —— CSV 的头必须随数据一起走,管道里没法
// 把 stderr 和 stdout 重新配对;JSON 模式则相反,stdout 只有那一份信封。
func WriteCSV(w io.Writer, meta Meta, columns []string, rows [][]string) error {
	if err := WriteMeta(w, meta); err != nil {
		return err
	}
	if len(columns) == 0 {
		_, err := io.WriteString(w, "# (无结果集)\n")
		return err
	}
	cw := csv.NewWriter(w)
	if err := cw.Write(columns); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write(r); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
