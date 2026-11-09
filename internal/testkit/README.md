# internal/testkit — 跨包的测试辅助

## 定位

`_test.go` 不能被别的包 import（Go 语言规则），所以"多个包都要用的测试 helper"只能住在一个
**普通包**里，由各包的 `_test.go` 去 import。本包就是那个普通包：语料发现、stdout 捕获、
flag 重置、仓库本体定位。

## 职责边界

**做**：把"取不到东西"变成一条**约定好的收场**（跳过还是判失败）、把跨包重复的测试小工具
收成一处。

**不做**：

- **不做发现逻辑本身。** 语料根怎么定、文件怎么走、哪些草稿产物不算语料，单源在
  `internal/dev/testutil`；本包只加"没有语料该怎么收场"这层语境。两处各写一份的后果不是
  多几行 —— 是一边认 `TDEV_CORPUS`、另一边只认 `TTZS_CORPUS`，于是同一条命令在一台机器上
  跑全量、在另一台上**静默跑零个包**（"0 个包全部通过"是最坏的一种假绿）。
- **不做断言库。** 仓库不用 testify（`go.mod` 里没有直接从属），断言一律手写。
- **不装 `RequireDir` 一类还没人用的东西。** 为"将来可能要用"预留的 API 就是"只有测试用的
  生产代码"的镜像版 —— `internal/debug/wstest.go` 已经是那类坏味道的一次教训
  （它按"临时测试工具"进来，最后按产品端点留了很久；**本批已从产品里取消、连同它的测试一起删掉**）。

## 与谁发生关系

| 方向 | 谁 | 约束 |
|---|---|---|
| 被谁 import | `internal/dev/{fence,pkgfile,tapfile,fgl,cli}`、`internal/cli/dict` 的 `_test.go` | **只能被 `_test.go` import** |
| 依赖谁 | `internal/dev/testutil`（`CorpusRoot` / `CorpusFiles`，发现逻辑的单源） | 方向单向 |

**依赖方向不能反过来**：`internal/dev/testutil` 被**生产代码** import
（`internal/cli/dev/selftest.go:16`，`tt dev tzc selftest` 的实现），而本包 import `testing` ——
testutil 一旦引本包，整个 `testing` 包就会进 `tt` 二进制。

## 判据

```bash
go test ./internal/testkit -count=1
# TestHelpersHaveNoLocalCopies            跨包 helper 没有本包之外的第二份实现
# TestRepoRootFindsTheModule              上溯定位到含 go.mod 的那一级（防上面那条空转）
# TestCaptureStdoutSurvivesBigOutput      64 KB 输出不会挂住
# TestCaptureStdoutReturnsCode            退出码原样出来
```

**`TestHelpersHaveNoLocalCopies` 必须见过它红**（仓库规矩：没见过它红的断言不算数）：
在任一个 `_test.go` 里写回 `func corpusRoot(` 或 `func captureStdout(`，它就该报出那个文件。

## 跳过台账（`skipledger_test.go`）

**做**：`internal/` 下每一处 `t.Skip` / `t.Skipf` 都必须在 `skipLedger` 里登记
（文件 + 文案字面量 + 层 + 为什么）。多一处、少一处、**改一个标点**都红。

为什么值得：跳过的测试与通过的测试在报告里长得一样无害，而这个仓库默认档的覆盖面正是
由"跳了多少、跳的是哪一类"定义的 —— 悄悄加一处跳过，默认档就少验一块，没有任何东西会响。

分层（取值见 `skipLayer`）：`L3` 真语料 / `L3-缺件` 语料在但没这一件（无开关可点名）/
`L4` 真机 E2E / `L0-环境` 本机没那个东西 / `前提` 测试自造前提没满足 / `未落地` 契约已冻结待实现。
`L3`/`L4` 的 `why` **必须点名开关名**，而开关名要先进 `switchNames`
（只列会出现在跳过文案里的那些，`TestSkipLedgerHasNoStaleSwitchNames` 反向守着"表里没有死名字"）。

```bash
go test ./internal/testkit/ -run 'TestSkipLedger' -count=1
# TestSkipLedger                        台账与代码严格对上（双向）
# TestSkipLedgerL3L4NamesItsSwitch      L3/L4 的 why 点了名
# TestSkipLedgerHasNoStaleSwitchNames   switchNames 里没有死名字
```

扫描器（`reSkipCall`）只抓 `t.Skip/Skipf` 的**第一个**字符串字面量，且**跳过文案里不要用半角双引号**
（`\"` 会让正则提前停下，抓到的与台账对不上）。跨行或拼接的文案它也抓不到 —— 那时该改扫描规则，
不是把台账删掉。

## 细节去哪

- 语料根怎么定、哪些文件算语料 → [`../dev/testutil/README.md`](../dev/testutil/README.md)
