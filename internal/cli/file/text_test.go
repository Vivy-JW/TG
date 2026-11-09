package file

import (
	"bytes"
	"strings"
	"testing"
)

func TestHeadLines(t *testing.T) {
	got, err := headLines(strings.NewReader("a\nb\nc\n"), 2)
	if err != nil || len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("headLines = %v (err=%v)", got, err)
	}
	// 最后一行没有换行也要算
	got, _ = headLines(strings.NewReader("a\nb"), 5)
	if len(got) != 2 || got[1] != "b" {
		t.Fatalf("无尾换行被吞了: %v", got)
	}
}

func TestTailLines(t *testing.T) {
	data := []byte("a\nb\nc\nd\n")
	got, err := tailLines(bytes.NewReader(data), int64(len(data)), 2)
	if err != nil || len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Fatalf("tailLines = %v (err=%v)", got, err)
	}
	// 无尾换行
	data = []byte("a\nb\nc")
	got, _ = tailLines(bytes.NewReader(data), int64(len(data)), 2)
	if len(got) != 2 || got[0] != "b" || got[1] != "c" {
		t.Fatalf("无尾换行: %v", got)
	}
	// n 大于行数
	got, _ = tailLines(bytes.NewReader([]byte("only\n")), 5, 10)
	if len(got) != 1 || got[0] != "only" {
		t.Fatalf("n 大于行数: %v", got)
	}
	if got, _ := tailLines(bytes.NewReader(nil), 0, 3); len(got) != 0 {
		t.Fatalf("空文件应为空: %v", got)
	}
}

func TestIsBinary(t *testing.T) {
	if isBinary([]byte("plain text\n")) {
		t.Error("纯文本被判成二进制")
	}
	if !isBinary([]byte("abc\x00def")) {
		t.Error("含 NUL 应判成二进制")
	}
}

func TestIncludeOK(t *testing.T) {
	if !includeOK("x.4gl", nil) {
		t.Error("不给模式 = 全收")
	}
	if !includeOK("x.4gl", []string{"*.4gl", "*.inc"}) {
		t.Error("*.4gl 应命中")
	}
	if includeOK("x.42m", []string{"*.4gl"}) {
		t.Error("*.4gl 不该命中 .42m")
	}
	if includeOK("x.4gl", []string{"[bad"}) {
		t.Error("非法模式应不命中（不 panic）")
	}
}
