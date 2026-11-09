package debug

// `tt debug source` 现场缺陷（.agents/plan/L1-结果-2026-10-18.md §7-2）：
// 报「登录区源码目录未知(环境路径解析失败)」，人只能猜。
//
// 服务端的错误文案现在自带三条可操作的路（internal/debug/manager.go 的
// sourceRootsUnknownErr）；客户端再补一段**命令行视角**的"怎么办"，
// 两边都敢丢一个也不会变成无法排障的哑错误。
// 本文件只测这段纯字符串逻辑：不连服务、不读盘。

import (
	"fmt"
	"os"
	"strings"
)

// sourceDirHowTo 命中"登录区源码目录未知"时返回命令行视角的补丁说明；不命中返回 nil。
//
// 判据只看**文案关键词**（服务端错误码在信封 Code 里，源码读取没有专用码），
// 所以宁可窄：只认这一条已知的失败，别把别的错误也包装成同一套提示。
func sourceDirHowTo(errText string) []string {
	if !strings.Contains(errText, "登录区源码目录未知") {
		return nil
	}
	return []string{
		"怎么办（登录区源码目录是登录后回读的 $TOP/$ERP/$COM，取不到就不猜路径）：",
		"  1) tt debug start <作业> --env <环境名>   —— 会话登录后会把 $TOP/$ERP/$COM 写进该环境的 Runtime",
		"  2) 同一份 serve 实例里才会有那份 Runtime（换环境后重新 start 或 switch 一次）",
		"  3) tt debug env                          —— 看该环境到底有没有 TOP/ERP/COM",
	}
}

// printSourceDirHowTo 把补丁说明打到 stderr（stdout 留给 JSON/正文，别污染管道）。
func printSourceDirHowTo(errText string) {
	for _, ln := range sourceDirHowTo(errText) {
		fmt.Fprintln(os.Stderr, ln)
	}
}
