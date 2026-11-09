# tools — 打包辅助脚本与评测装置

全部是 Go（`go run` 直跑，随 `go build ./...` 编译期把关）。2026-10 从 Python 脚本
整体迁移。

| 项 | 是什么 |
|---|---|
| `zip/` | 把便携包暂存目录打成 zip |
| `wixremovefolders/` | 给安装包的文件清单补上"卸载时要删的目录" |
| `tzsmini/` | 重新生成 `testdata/tzs-mini/ws` 最小语料夹具（含 `manifest.txt`） |
| `eval/` | 工具面评测装置（见它自己的 README） |

## `zip/`

**必须递归。** 早先是在批处理里塞一句 `python -c`，用的是"列目录 + 逐个写"的写法 ——
那个组合**不会递归**：目录只会被写成一个空条目，于是包里明明该有技能文档，却只有一个空的
技能目录，而技能文档正是给 AI 用的说明书。单独成工具就是为了这一段能被读、被改、被审。

路径分隔符固定用 `/`（zip 规范），不依赖平台。**没有降级回退**：打包失败就该让构建停下来，
而不是悄悄换一个会把条目名写成反斜杠的降级实现（旧 PowerShell 回退正是这么干的）。

## `wixremovefolders/`

安装包是**用户级**的（目录都落在用户配置目录下）。MSI 的校验要求这类目录必须登记进
"卸载时删除"表，否则卸载后目录会原样留在磁盘上。而采集文件的工具只生成组件与文件、
不生成这个登记，所以构建流水线里补这一道。

三条规则：目录自己有组件时往那个组件里挂一条；目录自己只有子目录时补一个只挂注册表键的
组件专门负责删空目录；那种组件用"按 keypath 推 GUID"的写法，重打包装升级时不会被认成另一个组件。
另有一条：文件做 KeyPath 的组件要改成 HKCU 注册表键做 KeyPath（用户级安装的判定语义）。

这四条 MSI 语义全部有单元测试钉着（[main_test.go](./wixremovefolders/main_test.go)，
夹具模拟 heat 输出、逐条断言）—— 重写实现时丢了哪条，测试替你红。

## `tzsmini/`

从一份真工作区重新生成 `testdata/tzs-mini/ws` 与 `manifest.txt`（一次做两件事）。
生成规则与"为什么留哪些文件"见 [main.go](./tzsmini/main.go) 的文件头（每条都注明是实测）。

## 判据

```bash
go build ./...                                        # 三个工具随全仓编译把关
go test ./tools/wixremovefolders/                     # 四条 MSI 语义逐条过
go run ./tools/zip                                    # 打一次，检查包里确实有 skills/<名字>/SKILL.md
build_msi.bat                                         # 走完整条打包链
go run ./tools/tzsmini --src <真工作区> --out %TEMP%\tzsmini-check   # 对照 manifest 行数与内容
```

## 细节去哪

- 安装包定义与用户级安装的理由 → [../installer/README.md](../installer/README.md)
- 最小语料夹具本身 → [../testdata/tzs-mini/README.md](../testdata/tzs-mini/README.md)
