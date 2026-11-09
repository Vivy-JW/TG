# internal/testenv — 本机测试环境的配置

## 定位

一份**机器本地**的配置文件（仓库根的 `config.local.json`）加环境变量覆盖，回答"这台机器上
引擎在哪、语料在哪、工作区在哪"。优先级一律 **环境变量 > 配置文件 > 内置缺省**。

## 职责边界

**做**：解析这四项，并把"这次是怎么定下来的"讲成一句可引用的话（`CorpusRootDetail`）。

**不做**：

- **不遍历语料文件。** 根怎么定在这里，怎么走（递归找 `.tzc`、跳过哪些草稿产物）在
  `internal/dev/testutil`。
- **不碰口令。** 本包只放**机器路径**。口令在 `config.json` 的 `hosts.sshs[].db` 里，
  那是**凭据**，是另一件事；混在一起等于让测试配置继承凭据的处理规矩。
- **不缓存。** 文件很小，而缓存 + `os.Chdir`（`internal/config` 的测试会 chdir）会得到
  "读的是上一个工作目录的配置"这种难查的错。

## 与谁发生关系

| 方向 | 谁 | 约束 |
|---|---|---|
| 被谁 import | `internal/dev/testutil`、`internal/dev/tzs`（doctor）、各 `_test.go` | **本包不带 `testing`** |
| 依赖谁 | 只用标准库 | — |

**为什么单独一个包、为什么不能带 `testing`**：`internal/dev/testutil` 要用它定语料根，而
testutil 被**生产代码** import（`internal/cli/dev/selftest.go`，`tt dev tzc selftest` 的实现）。
那条链上出现 `testing`，整个 `testing` 包就会进 `tt` 二进制。分层是：

```
testenv（配置与解析，无 testing）
   ↑                    ↑
testutil（夹具 + 语料遍历）   testkit（*testing.T 那层包装：跳过还是判失败）
```

## 怎么配

仓库根放 `config.local.json`（`.gitignore` 已经忽略它）。四项都可以留空，也都可以用环境变量代替：

| 字段 | 环境变量 | 缺省 |
|---|---|---|
| `corpusRoot` | `TDEV_CORPUS` / `TTZS_CORPUS`（谁先设谁说话） | `D:\t100_wrok_dir` |
| `engineExe` | `TTZS_EXE` | 无 |
| `workspace` | `TTZS_WS` | 无 |
| `designerDir` | `TTZS_INSTALL` | 无（用随包分发的那份） |

**覆盖项设了却指不到目录 → 返回空，不落回缺省。** 这是刻意的：静默换一份别的语料去跑，
会留下"我以为跑的是这份"这种错误结论。

## 判据

```bash
go test ./internal/testenv -count=1
# TestCorpusRootEnvWinsOverFile            环境变量压过配置文件
# TestCorpusRootAcceptsBothEnvVars         两个变量都认（从前四个包只认 TDEV_CORPUS）
# TestCorpusRootOverrideWinsWhenBothSet    两个都设时按声明顺序
# TestCorpusRootDoesNotFallBackWhenOverrideIsBroken  指错了就给空，不回落
# TestCorpusRootFileWhenNoEnv              没有环境变量时用文件里的
# TestOtherThreeFollowTheSameOrder         另外三项同一条规矩
# TestMissingOrBrokenConfigIsNotAnError    没有文件 / 坏 JSON 都当空配置，不报错
# TestCorpusRootDetailNamesTheRightCulprit 诊断点名**真正**卡住的那一处
# TestLocalFileIsFoundFromASubdirectory    从包目录上溯定位仓库根

./tt.exe dev tzs doctor          # ⑥ 语料根那一行就是本包的 CorpusRootDetail
```

**最后一条是关键**：`CorpusRootDetail` 的用处是给"还差什么"的文案兜底。自己拿"配置文件在不在"
去猜"是哪一处没指到"一定会猜错，而猜错的结果是**给出的指引正好指向错的那条路**
（明明是环境变量指错了，却说"你没设环境变量"）—— 那比不说更坏。
