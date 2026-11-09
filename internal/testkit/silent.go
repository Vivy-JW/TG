package testkit

import (
	"os"
	"testing"
)

// Silent 在测试期间把命令输出吞掉（命令层直接写 os.Stdout / os.Stderr）。
//
// 为什么收进本包：tzc 与 tzs 两个测试包都要吞输出，各自抄一份就是
// "图省事再抄一份"的坏味道（与 CaptureStdout 当年收敛到这里的理由相同）。
// 要看输出用 CaptureStdout；只想去掉噪声用本函数。
func Silent(t *testing.T, fn func() int) int {
	t.Helper()
	dev, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return fn()
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = dev, dev
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr; dev.Close() }()
	return fn()
}
