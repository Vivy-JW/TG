// Package testkit 是**跨包的测试辅助**：语料发现、stdout 捕获、flag 重置、仓库本体定位。
//
// 为什么需要它：`_test.go` 不能被别的包 import（Go 语言规则），所以"多个包都要用的
// 测试 helper"只能住在一个**普通包**里，由各包的 `_test.go` 去 import。
//
// 两条硬约束，都是本包存在的理由与它不能长大的边界：
//
//  1. **本包只被 `_test.go` import。** 它 import `testing`，一旦有生产代码引它，
//     整个 `testing` 包就会进 `tt` 二进制。所以它不进任何生产 import 图。
//  2. **依赖方向只能 testkit → testutil。** `internal/dev/testutil` 被**生产代码**
//     import（internal/cli/dev/selftest.go:16，`tt dev tzc selftest` 的实现），
//     所以它不能反过来引本包 —— 那会把 `testing` 拖进二进制。
//
// 于是分工是：**发现逻辑**（语料根怎么定、文件怎么走）单源在 `testutil`，
// 本包只加**"取不到该怎么收场"**这层语境（跳过还是判失败、文案怎么给）。
//
// 本包不提供断言库，也不假装是。它只做"取到东西"与"取不到就按约定的方式停"。
package testkit
