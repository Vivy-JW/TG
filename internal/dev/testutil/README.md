# internal/dev/testutil — 测试夹具与语料发现

两件互不相干的事，刻意放在一起的两个文件里：

| 文件 | 做什么 |
|---|---|
| `fixture.go` | **合成**测试包：不依赖真实语料，供 `selftest` 与各包单测使用 |
| `corpus.go` | **发现**真实语料：只回答"语料在哪、有哪些文件"，不读包、不解压、不判好坏 |

合成与发现分开，是为了让"不依赖真实语料"这条承诺在 `fixture.go` 里继续成立 ——
需要真语料的用例显式去 `corpus.go` 拿文件。

## `corpus.go`：语料根只有一个真源

两条管线（`.tzc` 与 `.tzs`）找的是**同一批目录**，只是扩展名不同。所以"根怎么定、怎么走"
各写一次，两份共用：

- **根**：按顺序认 `TDEV_CORPUS`、`TTZS_CORPUS`（两个名字都认，**谁先设谁说话**），
  都没设时用缺省 `D:\t100_wrok_dir`。
- **覆盖变量设了但指不到目录 → 返回空**，不回落到缺省。那表示调用方想指出一份语料而指错了；
  静默换一份别的语料去跑，会留下"我以为跑的是这份"这种错误结论。
- **发现**：递归收集扩展名匹配（大小写不敏感）的文件，路径升序；`excludePrefix` 用来跳过
  驱动脚本自己留下的草稿产物（`_ai` / `_dw` / `_ed` / `_del` / `_ac` 等）——
  它们会在两次运行之间出现或消失，算进基线基线就不可复现。

**这是一处单一来源**：抄第二份的后果不是多几行，而是两边各自漂移 —— 一边认 `TDEV_CORPUS`、
另一边只认 `TTZS_CORPUS`，于是同一条命令在一台机器上跑全量、在另一台上静默跑零个包，
而 "0 个包全部通过" 是最坏的一种假绿。

## `fixture.go`：合成包长什么样

照着真实包的最小可用集构造：多个区段（含强制只读的锚点区段）、一个自订定义点
（`function.`）、几个裸名插入点，以及 TAP 里的 `<other>`。导出的构造器有
`NormalEntries` / `NormalTGL` / `NormalTAP` / `Normal4GL` / `NormalEntriesWith` /
`WritePackage` / `WriteEntries` / `NormalPkgPath`。

## 判据

本包没有自己的测试（`go test ./internal/dev/testutil` 会报 `no test files`）——
它是夹具提供方，判据由使用者承担：

```bash
./tt.exe dev tzc selftest                 # 合成包的端到端对抗用例
go test ./internal/dev/...                # 各包单测
```

## 细节去哪

- **「根怎么定」不在这里** → [../../testenv/README.md](../../testenv/README.md)
  （环境变量 > 仓库根的 `config.local.json` > 内置缺省）。本包只用它给的根去**遍历**。
- **跳过文案与 `*testing.T` 那层语境**（"没有语料该怎么收场"） → [../../testkit/README.md](../../testkit/README.md)。
  本包**不能** import 它：本包被生产代码 import（`internal/cli/dev/selftest.go`），而它带 `testing`。
- 语料清单的固定（`engine/corpus.manifest`） → [../../../engine/BUILD.md](../../../engine/BUILD.md)
