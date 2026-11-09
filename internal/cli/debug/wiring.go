package debug

// A35（契约 .agents/plan/tt-genero-formdef-contract.md §9 A35）：
// **调试必须先接线**，否则 `main` 之后**立刻** `Fatal error -6300` 退出。
//
// 第四次真机实测（L1-断点排查-2026-10-18.md §4.4 卡点 B / N5，三次复现）：没有 GDC 接线的会话
// 在 `main` 之后立刻死 —— `Fatal error -6300 Can not connect to GUI`；`main:1553` 之后的
// 守卫行（1657 / 1561）**一次都没停**。"断点怎么设都不命"于是被读成"断点装不上"，
// 而真因是**程序已经退了**。
//
// 服务端早就把这句话写进事件日志了（`debug.ScreenWarn`），但**`tt debug start` 不显示事件日志** ——
// 人只看到"入口停站成功"，然后放行、然后程序消失。所以这里在 `start` **启动那一刻**把它打出来，
// 并且明说后果（-6300），让它成为一条能照做的提示而不是一句事后解释。
//
// 文案的**唯一来源**仍是内核包的 `ScreenWarnWith`：本文件不另抄一份，只负责"什么时候说"。

import (
	"fmt"
	"os"

	"tt/internal/debug"
)

// gdcLocal 探"本机有没有 GDC 在听"（7400/6400 连得上就算）。
//
// 包级变量是为了可替换：探针是**真实网络**动作，测试里不能让它决定断言结果
// （既有做法见 internal/debug 的 `gdcProbe`）。
var gdcLocal = debug.LocalGDCPresent

// dbgWiringHint 启动那一刻该说的接线提示（空串 = 不必说）。
//
// 判据用**服务端回包里的 `fglserver`**（`snap.FGLServer`），不用命令行上那个标志：
// 标志"没给"不等于"本轮交回 T100 自动指" —— 环境配置里也可能写着 FGLSERVER。
// 快照里那个值才是这一轮真正导出的那一个。
func dbgWiringHint(fglserver string) string {
	return debug.ScreenWarnWith(fglserver, gdcLocal())
}

// printWiringHint 把提示打到**该去的流**上：人读模式进 stdout（与"画面去向"那行挨着），
// `--json` 模式进 stderr —— stdout 上的 JSON 必须仍然是一份能被 jq 吃掉的完整文档。
func printWiringHint(hint string) {
	if hint == "" {
		return
	}
	hint = "⚠ " + hint
	if IsJSON() {
		fmt.Fprintln(os.Stderr, hint)
		return
	}
	fmt.Println(hint)
}
