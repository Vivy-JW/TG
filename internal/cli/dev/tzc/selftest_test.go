package tzc

import "testing"

// TestSelftestCases 把 `tt dev tzc selftest` 那 31 项接进 `go test`。
//
// **只写驱动，不写断言。** 那 31 项的形状本来就是 t.Run + t.TempDir：cmdSelftest 自己
// 就是一条驱动循环（selftest.go 里 MkdirTemp → c.Run → RemoveAll，逐项计数），
// 这里把它的循环原样搬过来，只把临时目录换成 t.TempDir（由 go test 负责清理）。
//
// 为什么值得这一条：**单一来源**。selftestCases() 是那 31 项的唯一定义处 —— 谁给
// `tt dev tzc selftest` 加一项，`go test` 自动跟着长，不存在"两份清单各自漂移"。
// 反过来，如果这里另抄一份清单，加一项就要改两处，那正是本仓库反复出现的坏味道。
//
// 覆盖面：这 31 项里有十几项走的是 cmdExport / cmdApply / cmdVerify / cmdRename /
// cmdNewfn / cmdUnlock 的**完整命令路径**（约 1200 行），而在此之前默认档里没有一条
// 断言碰到它们 —— 默认 `go test ./...` 要么跳过（语料不在），要么只调纯 helper。
//
// 反向判据（改这一层之前必须重做一次）：把 selftest.go 里 stRoundtrip 的那行
// sha256 比对注掉，这条必须红。**没见过它红的断言不算数**（pos_test.go 立的规矩）。
func TestSelftestCases(t *testing.T) {
	cases := selftestCases()
	if len(cases) == 0 {
		// 驱动还在、被测对象没了 —— 那种情况下这条测试会"全绿"，比红更坏。
		t.Fatal("selftestCases() 返回空清单：驱动还在，被测的用例没了")
	}
	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			dir := t.TempDir()
			if err := c.Run(dir); err != nil {
				t.Fatalf("%s：%v", c.Name, err)
			}
		})
	}
}
