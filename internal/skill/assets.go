package skill

import (
	"bytes"
	"path/filepath"
)

// 静态资源的处理：第三方技能根级 assets/ **不复制**到目标目录，改为在目标 SKILL.md
// 正文首行写一行绝对路径指针。
//
// 为什么不像其它文件一样复制走：知识库型技能的 assets/ 可能几百 MB 到几 GB，
// 而 install skills 是个随手会跑的动作（升级后会自己再跑一次）。复制等于把用户的磁盘
// 按技能数量乘一遍。
//
// 代价（已知且写进文档）：指针是绝对路径，源目录挪了就失效；某些 agent 的文件工具
// 只允许读工程目录内的路径，可能读不到工程外的绝对路径。这时的正确做法是用户自己
// 把资源挪进工程，tt 不做自动复制。

const (
	assetsDirName = "assets"

	// assetsPointerSentinel 是那一行指针的**哨兵前缀**。安装时先找它：找到就替换，
	// 找不到才插入。这样 --force 刷十次也只有一行，源路径变了还能自动纠正。
	//
	// 用可见的引用行而不是 HTML 注释：这一行的读者是模型（要让它知道资源在哪），
	// 人在设置页/GitHub 上看到它也是好事（知道这份技能指向硬盘上哪个目录）。
	assetsPointerSentinel = "> 静态文件位置："
)

// AssetsDir 返回某技能目录的根级 assets/ 路径（不保证它存在）。
func AssetsDir(skillDir string) string { return filepath.Join(skillDir, assetsDirName) }

// AssetsPointerLine 是指针行的正文（不含换行）。
func AssetsPointerLine(assetsDir string) string {
	return assetsPointerSentinel + assetsDir + "（未复制到本目录，请按此路径读取）"
}

// InjectAssetsPointer 在 SKILL.md 的**正文首行**写入静态文件指针。
//
// 只做字节级插入/替换，且只碰正文：
//   - frontmatter（含 BOM 与它自己的换行风格）一个字节不动，所以 name/description 的
//     合规判定不受影响；
//   - 原文的换行风格（CRLF/LF）沿用，不做统一；
//   - 指针行已存在（正文第一个非空行就是它）时替换整行，不追加；
//   - 内容没变化时 changed=false，调用方可以不写盘。
//
// 没有 frontmatter 的文件原样返回：那本来就不该被装出去（Inspect* 已经拦过了）。
func InjectAssetsPointer(src []byte, assetsDir string) (out []byte, changed bool) {
	bom, body := splitBOM(src)
	eol := detectEOL(body)
	bodyStart, ok := bodyStartAfterFrontmatter(body)
	if !ok {
		return src, false
	}

	ptr := []byte(AssetsPointerLine(assetsDir))
	sentinel := []byte(assetsPointerSentinel)

	// 找正文第一个非空行：指针要么插在它前面，要么替换它（如果它已经是指针）。
	pos := bodyStart
	firstNonEmpty := -1
	for pos < len(body) {
		line, lineEnd, next := nextLine(body, pos)
		if len(bytes.TrimSpace(line)) > 0 {
			firstNonEmpty = pos
			if bytes.HasPrefix(bytes.TrimSpace(line), sentinel) {
				trimmed := bytes.TrimSpace(line)
				if bytes.Equal(trimmed, ptr) {
					return src, false // 已经是这一行，什么都不用做
				}
				res := concat(bom, body[:pos], ptr, body[lineEnd:])
				return res, true
			}
			break
		}
		pos = next
	}

	var prefix []byte
	atEOF := firstNonEmpty < 0 || firstNonEmpty == len(body)
	if firstNonEmpty < 0 { // 正文是空的：追加在文件末尾
		firstNonEmpty = len(body)
	}
	// 指针前后各留一个空行。原文在插入点前已经空行就沿用，不重复添。
	if !bytes.Contains(body[bodyStart:firstNonEmpty], []byte("\n")) {
		prefix = []byte(eol)
	}
	suffix := []byte(eol + eol)
	if atEOF { // 末尾不留下一个孤立的空行
		suffix = []byte(eol)
	}
	res := concat(bom, body[:firstNonEmpty], prefix, ptr, suffix, body[firstNonEmpty:])
	return res, true
}

// splitBOM 把 UTF-8 BOM 与其余内容分开。
func splitBOM(b []byte) (bom, body []byte) {
	if bytes.HasPrefix(b, []byte("\xef\xbb\xbf")) {
		return b[:3], b[3:]
	}
	return nil, b
}

// detectEOL 取文件第一处换行的风格；没有换行时按 LF。
func detectEOL(b []byte) string {
	i := bytes.IndexByte(b, '\n')
	if i > 0 && b[i-1] == '\r' {
		return "\r\n"
	}
	return "\n"
}

// bodyStartAfterFrontmatter 返回闭合 `---` 那一行之后（含换行）的偏移。
// 结构不像 SKILL.md 时 ok=false。
func bodyStartAfterFrontmatter(body []byte) (int, bool) {
	nl := bytes.IndexByte(body, '\n')
	if nl < 0 {
		return 0, false
	}
	if string(bytes.TrimSpace(body[:nl])) != "---" {
		return 0, false
	}
	pos := nl + 1
	for pos < len(body) {
		line, _, next := nextLine(body, pos)
		if string(bytes.TrimSpace(line)) == "---" {
			return next, true
		}
		pos = next
	}
	return 0, false
}

// nextLine 返回 pos 处那一行的内容（不含换行）、该行内容的结束偏移、下一行的起始偏移。
func nextLine(b []byte, pos int) (line []byte, lineEnd, next int) {
	nl := bytes.IndexByte(b[pos:], '\n')
	if nl < 0 {
		line = b[pos:]
		lineEnd = len(b)
		return line, lineEnd, len(b)
	}
	lineEnd = pos + nl
	if lineEnd > pos && b[lineEnd-1] == '\r' {
		return b[pos : lineEnd-1], lineEnd - 1, lineEnd + 1
	}
	return b[pos:lineEnd], lineEnd, lineEnd + 1
}

// concat 拼接若干片段，返回新分配的切片（不共享入参的底层数组）。
func concat(parts ...[]byte) []byte {
	n := 0
	for _, p := range parts {
		n += len(p)
	}
	out := make([]byte, 0, n)
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}
