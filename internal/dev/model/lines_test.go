package model

import "testing"

func TestLineOfAndLineAt(t *testing.T) {
	// 三行，行尾 CRLF（真实 .4gl 的常见形态）
	text := []byte("aaa\r\nbbb\r\nccc")
	cases := []struct {
		off      int
		wantLine int
		wantText string
	}{
		{0, 1, "aaa"},
		{1, 1, "aaa"},
		{3, 1, "aaa"},  // 指向 \r：仍属第 1 行
		{4, 1, "aaa"},  // 指向 \n：仍属第 1 行
		{5, 2, "bbb"},  // 第 2 行第一个字节
		{9, 2, "bbb"},  // 第 2 行的 \r
		{10, 3, "ccc"}, // 第 3 行第一个字节
		{13, 3, "ccc"}, // 末尾（EOF）
		{99, 3, "ccc"}, // 越界 → 夹到最后一行
		{-5, 1, "aaa"}, // 负数 → 夹到第一行
	}
	for _, c := range cases {
		line, txt := LineAt(text, c.off)
		if line != c.wantLine || txt != c.wantText {
			t.Errorf("LineAt(off=%d) = (%d, %q)，想要 (%d, %q)", c.off, line, txt, c.wantLine, c.wantText)
		}
		if got := LineOf(text, c.off); got != c.wantLine {
			t.Errorf("LineOf(off=%d) = %d，想要 %d", c.off, got, c.wantLine)
		}
	}

	// 空文档 / 只有一个换行
	if line, txt := LineAt(nil, 0); line != 1 || txt != "" {
		t.Errorf("空文档 = (%d, %q)，想要 (1, \"\")", line, txt)
	}
	if line, _ := LineAt([]byte("\n"), 1); line != 2 {
		t.Errorf("纯换行文档 off=1 的行号 = %d，想要 2", line)
	}
	// 纯 CR 不算换行（与围栏层口径一致）
	if line, txt := LineAt([]byte("aa\rbb"), 3); line != 1 || txt != "aa\rbb" {
		t.Errorf("纯 CR 文档 = (%d, %q)，想要 (1, %q)", line, txt, "aa\rbb")
	}
	// 末尾无换行的最后一行也要能定位
	if line, txt := LineAt([]byte("a\nbc"), 3); line != 2 || txt != "bc" {
		t.Errorf("末行 = (%d, %q)，想要 (2, %q)", line, txt, "bc")
	}
}

func TestFirstDiff(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"", "", -1},
		{"abc", "abc", -1},
		{"abc", "abd", 2},
		{"abc", "axc", 1},
		{"abc", "ab", 2}, // b 是前缀
		{"ab", "abc", 2}, // a 是前缀
		{"abc", "", 0},
		{"", "abc", 0},
	}
	for _, c := range cases {
		got := FirstDiff([]byte(c.a), []byte(c.b))
		if got != c.want {
			t.Errorf("FirstDiff(%q, %q) = %d，想要 %d", c.a, c.b, got, c.want)
		}
		// 对称性：先有的差异位置必须一致
		rev := FirstDiff([]byte(c.b), []byte(c.a))
		if rev != c.want {
			t.Errorf("FirstDiff 反向不对称：(%q, %q) = %d，正向 %d", c.b, c.a, rev, c.want)
		}
	}
}

// TestLineOfOnCRLFDiff 钉住「EOL 归一化」场景：CRLF 与 LF 的差异位置能定位到行。
func TestLineOfOnCRLFDiff(t *testing.T) {
	crlf := []byte("line1\r\nline2\r\nline3\r\n")
	lf := []byte("line1\nline2\nline3\n")
	off := FirstDiff(crlf, lf)
	if off != 5 {
		t.Fatalf("首个差异应在第 1 行行尾（offset 5），实际 %d", off)
	}
	if line := LineOf(crlf, off); line != 1 {
		t.Errorf("差异行号 = %d，想要 1", line)
	}
	// 只差行尾 → EOL 口径下没有差异
	if got := FirstDiffEOL(crlf, lf); got != -1 {
		t.Errorf("FirstDiffEOL(纯行尾差异) = %d，想要 -1", got)
	}
}

// TestFirstDiffEOL 跳过行尾差异，把位置指到真正的改动。
func TestFirstDiffEOL(t *testing.T) {
	base := []byte("aa\r\nbb\r\ncc\r\n")
	// 编辑器把整份文件归一成 LF，同时改了第 2 行
	edited := []byte("aa\nbb-改\ncc\n")
	got := FirstDiffEOL(base, edited)
	if got < 0 {
		t.Fatalf("FirstDiffEOL 应找到真实差异")
	}
	if line := LineOf(edited, got); line != 2 {
		t.Errorf("差异行号 = %d，想要 2（FirstDiffEOL=%d）", line, got)
	}
	// 单独 CR（老式行尾）与 LF 等价
	if got := FirstDiffEOL([]byte("a\rb"), []byte("a\nb")); got != -1 {
		t.Errorf("单独 CR vs LF = %d，想要 -1", got)
	}
	// 只在末尾多出行尾：视为无差异
	if got := FirstDiffEOL([]byte("x\r\n"), []byte("x\n")); got != -1 {
		t.Errorf("末尾行尾差异 = %d，想要 -1", got)
	}
	// 长度不同且不是行尾 → 返回 b 的末尾下标（仍可定位到末行）
	if got := FirstDiffEOL([]byte("abc"), []byte("ab")); got != 2 {
		t.Errorf("b 是 a 的前缀 = %d，想要 2", got)
	}
}
