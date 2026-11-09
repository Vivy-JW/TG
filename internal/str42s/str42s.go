// Package str42s 解析 T100/Genero 的 .42s 编译字符串表（二进制）。
//
// 文件布局（小端）：
//
//	uint32 count
//	uint32 hashes[count]    // 不参与语义，保留在文件里供原工具查找
//	uint32 offsets[count]   // 同上
//	strings                // 2*count 个 NUL 结尾的 UTF-8 字符串，逐对 token/value
//
// 字符串区固定从 4 + 8*count 开始。这里只读 count 与字符串对；
// 两段索引不解释 —— 标签查找靠 token 精确匹配，不靠未知的哈希/偏移算法。
//
// 解析器是纯函数，零 IO：取文件、判断缓存、报告缺文件都由命令层负责。
package str42s

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Entry 是字符串表里的一条 token → text。
type Entry struct {
	Token string
	Text  string
}

// Table 是一张已经解析好的 .42s。
type Table struct {
	Entries []Entry

	byToken map[string][]string
}

// Parse 解析 .42s 的完整字节。
//
// 容错边界：
//   - count 超过文件能容纳的上限直接报错，避免坏文件让调用方分配巨量内存；
//   - 字符串区少一对、或多出非空尾巴都报错；
//   - 末尾没有 NUL 也接受（只要求刚好 2*count 段）。
func Parse(data []byte) (*Table, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("42s: 文件太短（%d 字节，至少需要 4 字节的头）", len(data))
	}
	count := uint64(binary.LittleEndian.Uint32(data[:4]))
	start := uint64(4) + 8*count
	if start > uint64(len(data)) {
		return nil, fmt.Errorf("42s: 头声明 %d 条，但字符串区起点 %d 超过文件长度 %d", count, start, len(data))
	}
	if count == 0 {
		return &Table{byToken: map[string][]string{}}, nil
	}
	// 每条至少两个 token/value；用这个下界挡住“count 写坏了”的情况。
	if count > uint64(len(data)-int(start))/2+1 {
		return nil, fmt.Errorf("42s: 头声明 %d 条，但字符串区只剩 %d 字节", count, len(data)-int(start))
	}

	parts := bytes.Split(data[start:], []byte{0})
	want := int(count) * 2
	if len(parts) < want {
		return nil, fmt.Errorf("42s: 头声明 %d 对，字符串区只有 %d 段", count, len(parts))
	}
	// 末尾 NUL 会多出一个空段；除此之外的多余内容说明文件布局和预期不符。
	for _, extra := range parts[want:] {
		if len(extra) != 0 {
			return nil, fmt.Errorf("42s: 字符串区在 %d 对之后还有多余内容（%q）", count, truncate(string(extra), 40))
		}
	}

	t := &Table{
		Entries: make([]Entry, 0, count),
		byToken: make(map[string][]string, count),
	}
	for i := 0; i < int(count); i++ {
		if !utf8.Valid(parts[i*2]) || !utf8.Valid(parts[i*2+1]) {
			return nil, fmt.Errorf("42s: 第 %d 对不是合法 UTF-8（文件可能被截断）", i+1)
		}
		token := string(parts[i*2])
		text := string(parts[i*2+1])
		t.Entries = append(t.Entries, Entry{Token: token, Text: text})
		t.byToken[token] = append(t.byToken[token], text)
	}
	return t, nil
}

// Lookup 精确查一个 token；重复 token 返回第一条。第二返回值是“是否存在”。
func (t *Table) Lookup(token string) (string, bool) {
	if t == nil {
		return "", false
	}
	vals := t.byToken[token]
	if len(vals) == 0 {
		return "", false
	}
	return vals[0], true
}

// LookupAll 返回一个 token 的全部文本（保留重复，调用方据此判歧义）。
func (t *Table) LookupAll(token string) []string {
	if t == nil {
		return nil
	}
	vals := t.byToken[token]
	if len(vals) == 0 {
		return nil
	}
	out := make([]string, len(vals))
	copy(out, vals)
	return out
}

// Unique 返回去重后的文本：
//
//	ok=false            token 不存在
//	ambiguous=true      同一 token 对应多个不同的文本
//
// 重复但文本完全相同不算歧义（编译产物里偶见同一对写两次）。
func (t *Table) Unique(token string) (text string, ok, ambiguous bool) {
	vals := t.LookupAll(token)
	if len(vals) == 0 {
		return "", false, false
	}
	seen := map[string]bool{}
	for _, v := range vals {
		seen[v] = true
	}
	if len(seen) > 1 {
		return "", true, true
	}
	return vals[0], true, false
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
