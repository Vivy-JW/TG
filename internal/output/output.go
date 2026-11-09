// Package output tt 各命令的输出出口:JSON 信封 / CSV(带 `# ` 环境头)/ 人读表格。
//
// 合并前这里是四套并存的东西:JSON 发射器四个(其中两个字节相同地重复)、
// CSV 写入器三个(一个死代码、一个吞掉所有错误、行尾还各不相同)、表格渲染四种。
// 现在只剩一个 Emit 加它下面的三个渲染器,且都能注入 io.Writer 以便单测。
//
// 默认形态是 **JSON 信封**(见 emit.go 的 Format)。
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// PrintTable 人读对齐表格(写 os.Stdout)。需要注入 writer 时用 WriteTable。
func PrintTable(headers []string, rows [][]string) { _ = WriteTable(os.Stdout, headers, rows) }

// WriteTable 人读对齐表格。
//
// 分隔行的宽度按**显示宽度**算而不是字符数:中文一个字占两个显示列,
// 按字符数算出来的分隔线会比表头短一截,整行看着是歪的(旧实现就是这个毛病)。
func WriteTable(w io.Writer, headers []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(headers, "\t")); err != nil {
		return err
	}
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = strings.Repeat("-", maxInt(displayWidth(headers[i]), maxColWidth(rows, i)))
	}
	if _, err := fmt.Fprintln(tw, strings.Join(sep, "\t")); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// displayWidth 字符串在终端里占几列:CJK 与全角字符按 2 列算,其余按 1 列。
// 这是个够用的近似(不查 Unicode 表),覆盖中日韩与全角标点这些实际会遇到的字符。
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

func runeWidth(r rune) int {
	switch {
	case r >= 0x1100 && r <= 0x115F, // 韩文字母
		r >= 0x2E80 && r <= 0xA4CF, // 中日韩部首、假名、汉字
		r >= 0xAC00 && r <= 0xD7A3, // 韩文音节
		r >= 0xF900 && r <= 0xFAFF, // 兼容汉字
		r >= 0xFE30 && r <= 0xFE6F, // 中日韩兼容形式
		r >= 0xFF00 && r <= 0xFF60, // 全角形式
		r >= 0xFFE0 && r <= 0xFFE6:
		return 2
	}
	return 1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxColWidth(rows [][]string, col int) int {
	max := 0
	for _, row := range rows {
		if col < len(row) {
			if l := displayWidth(row[col]); l > max {
				max = l
			}
		}
	}
	return max
}

// PrintJSON 输出一个裸 JSON 值(不带信封)。
//
// 查询命令请用 Emit —— 裸值没有地方放"这次查的是哪个环境/哪个账号",
// 而那正是排查"查错了地方"唯一的信息。这里保留它是给非查询类输出用的。
func PrintJSON(v interface{}) error { return WriteJSONValue(os.Stdout, v) }

// WriteJSONValue 把任意值写成紧凑 JSON（单行，末尾换行）。
func WriteJSONValue(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false) // 保留 < > & 原样输出(校验 SQL 含 <field> 等标签)
	return enc.Encode(v)
}
