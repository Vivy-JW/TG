// 行号与首个差异字节：报错定位的公共地基。
//
// 为什么放在 model：Region 已经带全套字节区间（ContentSpan / BeginFence / SignatureSpan /
// DescSpan / BodySpan），Document 带 Text，把「字节偏移 → 行号 + 该行内容」这一步做成
// 纯函数后，verify 的 Finding、cli 的 status/apply 输出、fence 的解析错误都能共用同一口径。
package model

import "bytes"

// LineOf 返回字节偏移 off 所在的 1-based 行号。
//
// off 越界（<0 或 >len(text)）时夹到合法范围，绝不返回 0 —— 调用方可以据此
// 用 `Line > 0` 判断「有没有位置信息」。
func LineOf(text []byte, off int) int {
	line, _ := LineAt(text, off)
	return line
}

// LineAt 返回 1-based 行号与「那一行的文本」（不含行尾 CR/LF）。
//
// 换行口径与围栏层一致：只认 \n（单独出现的 \r 不算换行，见 outline 的纯 CR 行注释修正）；
// 行尾的 \r 会被剥掉，便于直接打印/比对。
func LineAt(text []byte, off int) (int, string) {
	if off < 0 {
		off = 0
	}
	if off > len(text) {
		off = len(text)
	}
	line := bytes.Count(text[:off], []byte("\n")) + 1
	start := bytes.LastIndexByte(text[:off], '\n') + 1
	end := len(text)
	if i := bytes.IndexByte(text[off:], '\n'); i >= 0 {
		end = off + i
	}
	if start > end {
		start = end
	}
	return line, string(bytes.TrimRight(text[start:end], "\r"))
}

// FirstDiff 返回 a 与 b 首个不同字节的下标；完全相同返回 -1。
//
// 用于把「只读区被改动」「围栏外被插行」这类发现定位到具体字节，
// 再换算成行号（比只报 Region 名有用得多）。
func FirstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}

// normEOLIndexed 与 NormalizeEOL 同口径（CRLF 丢 CR；单独 CR → LF），
// 同时给出「归一化后的第 i 个字节对应原切片的哪个下标」，用于把差异位置映射回去。
func normEOLIndexed(b []byte) (norm []byte, idx []int) {
	norm = make([]byte, 0, len(b))
	idx = make([]int, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] == '\r' {
			if i+1 < len(b) && b[i+1] == '\n' {
				continue // CRLF：CR 丢掉
			}
			norm = append(norm, '\n') // 单独 CR → LF
			idx = append(idx, i)
			continue
		}
		norm = append(norm, b[i])
		idx = append(idx, i)
	}
	return norm, idx
}

// FirstDiffEOL 返回 b 里第一处「非行尾差异」对应的**原始下标**；只差行尾时返回 -1。
//
// 为什么要它：编辑器（VS Code 等）保存时会把整份文件的 CRLF 归一成 LF，
// 那时逐字节 FirstDiff 会在 Region 的第 0 个字节就报差异，把位置指到区段开头
// 而不是用户真正改的那一行。gate1 本身按 EqualEOL 判等价，这里保持同一口径。
func FirstDiffEOL(a, b []byte) int {
	if bytes.Equal(a, b) {
		return -1
	}
	na, _ := normEOLIndexed(a)
	nb, ib := normEOLIndexed(b)
	i := FirstDiff(na, nb)
	if i < 0 {
		return -1
	}
	if i >= len(ib) {
		return len(b) // 差异出现在 b 的末尾之后（b 是 a 的前缀）
	}
	return ib[i]
}
