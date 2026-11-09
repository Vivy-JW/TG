# internal/dev — 设计器包管线

`.tzc` 代码包（`.tzc` / `.tzf` / `.tzx`）的编辑管线：把包渲染成一个**带围栏**的 4GL 工作区，
人和 AI 只改那一个文件，改完经**三道闸门**写回原包。核心承诺是**不破坏可机械证明** ——
围栏之外的字节一个都不许动，写回之前先对"将要产出的新包"跑一遍装载模拟。

```
tt dev tzc export  →  工作区：prog.full.4gl + manifest.json + snapshot/ + .tdev/ + .git/
        （改 prog.full.4gl）
tt dev tzc verify  →  解析围栏 → gate1 → gate2（不写盘）
tt dev tzc apply   →  解析围栏 → gate1 → 源包未变校验 → I15 拒绝 → split → gate2
                      → 改写字节 → gate3 → 原子写 → git commit
```

（顺序取自 `cli.go` 的 `cmdApply`；任何一步失败，原包字节不变。）

## 两条管线别用错

本目录的包只服务 `.tzc`。表单包 `.tzs` 走的是另一条线（[./tzs](./tzs/README.md)），
它的格式由设计器自己的引擎算，本目录的围栏/闸门与它无关。

| | `tt dev tzc`（本目录） | `tt dev tzs`（[./tzs](./tzs/README.md)） |
|---|---|---|
| 包 | `.tzc` / `.tzf` / `.tzx` | `.tzs` / `.tzv` |
| 产物 | 工作区（可编辑，走 `apply` 写回） | 纯解压（**只读参考**，没有 tzs apply） |
| 怎么改 | 改 `prog.full.4gl` → `verify` → `apply` | 用引擎动词（`open` / `set_spec_attr` / `save`…） |
| 需要引擎吗 | 不需要 | 需要（C# 引擎 + 设计器程序集） |

拿错入口会被挡住：`.tzc` 跑 `tt dev tzs export` 报错并指回 `tt dev tzc export`（退出码 2）。

## 分层与数据流

```
pkgfile ─┬─ tapfile    包与条目（只读视图 + 字节级重建）
         ├─ tglfile    框架骨架里的标记：区段边界 / 占位符 / 集合锚点
         └─ fgl        4GL 结构：块信封（FUNCTION/MAIN/DIALOG/REPORT）与大纲
            ↓
         synth        Package → Document（合成 + 权限判定链）
            ↓
         fence        Document ⇄ FencedText（围栏渲染与解析，互为逆）
            ↓
         verify       gate1 字节恒等 · gate2 不变量 · gate3 装载模拟
            ↓
         split        Document' → TapOp + TglPatch（回写拆分）
            ↓
         store        工作区落盘（原子写）、锁、manifest、git 留痕
```

`model` 是上面所有包共用的纯数据层；`testutil` 提供合成包（不依赖真语料）与真实语料发现。

## 编号体系

这些编号是**代码里的 `Finding.Code` 与注释**，改这条线的代码时会反复遇到：

| 编号 | 是什么 | 在哪 |
|---|---|---|
| gate1 / gate2 / gate3 | 三道闸门 | [verify](./verify/README.md) |
| I1–I15 | gate2 的不变量全集（`I2a`/`I2b`/`I2c`/`I11b` 是子项） | [verify](./verify/README.md) |
| V1–V7 | 结构事务校验（改名 / 改 scope / 改描述），全部 error 级 | [verify](./verify/README.md)（`structtx.go`） |
| G1–G7 | 自订定义点的可编辑性判定 | [synth](./synth/README.md)（`ResolvePoint`） |
| S1–S5 | 区段的可编辑性判定 | [synth](./synth/README.md)（`ResolveSection`） |
| D-n | 与设计器实际行为之间的**有意偏差** | 各包自己的 README |
| R1–R7 | 红线，见下表 | 散在代码注释里 |

| | 红线 | 依据 |
|---|---|---|
| R1 | `.4gl` 永不改写（它是服务器产出） | `pkgfile.go:441`、`pkgfile.go:482` |
| R2 | 条目集合不增不减、名字与顺序不变 | `pkgfile.go:481` |
| R3 | `.tap` 不许用 XML 库整体序列化 | `tapfile.go:5` |
| R4 | 区段正文回写要把内嵌点折回 TglTag | `model/model.go:198`、`split` |
| R5 | `Package` 不可变，写回只走 `Build()` | `pkgfile.go:12` |
| R6 | 一切落盘走原子写（同目录临时文件 → Rename） | `store/atomic.go` |
| R7 | 时间戳不是功能：同一输入必须产出逐字节相同的工作区 | `store.go`、`cli/newfn_template.go:17` |

## 有意的偏差（D-n）

设计指南假设的几条规则，与真实包或真实行为不符，于是按**实测**做了偏离。编号出现在代码注释里，
所以列在这里对照：

| 编号 | 偏差 | 在哪落地 |
|---|---|---|
| **D-1** | 新增点的状态写 `"u"` 而**不是** `"c"` —— 写 `"c"` 会被设计器在下次保存时静默丢弃 | [fence](./fence/README.md)、[split](./split/README.md)、[cli](../cli/dev/README.md)（自检用例盯着） |
| **D-2** | 允许"区段 → 点"**一层嵌套**（指南要求围栏不嵌套）；真实框架里自订点就住在区段内 | [fence](./fence/README.md)、[model](./model/README.md) |
| **D-3** | 允许**未归属字节段**（前缀 / 区段之间的空行 / 后缀），但全部纳入字节恒等比对 | [model](./model/README.md)、[fence](./fence/README.md) |
| **D-6** | 写回前校验**源包自导出后未被改动**，变了就拒绝 | [cli](../cli/dev/README.md) |
| **DV-3** | "类型 → scope"的映射是**新建点对话框的默认值**，不是对既有文本的硬约束 → 只给 warn | [verify](./verify/README.md)（`structtx.go`） |

## 退出码

```
0 成功 / 2 包格式或用法错 / 3 验证失败 / 4 写入被拒 / 5 IO·环境失败
```

退出码映射在 [./cli](../cli/dev/README.md)。同一条命令的 `--json` 输出里也带 `exit_code` 字段。

## 工作区布局

固定布局，不随环境变化（`store/store.go` 的注释是权威）：

```
<dir>/
  prog.full.4gl        唯一编辑文件：带围栏的渲染文档
  manifest.json        人机共读索引（每个 Region 的权限与不可编辑原因）
  snapshot/
    index.json         原包条目清单：name / sha256 / size / role
    entries/<条目名>    原包每个条目的逐字节拷贝（恢复源；apply 不从这里读）
  .tdev/
    base.full.4gl      导出时 prog.full.4gl 的字节拷贝（gate1 的比对基线）
    base.sha256        基线的 sha256
    regions.json       Region 表 + 字节区间 + 基线摘要
    lock               互斥锁（防两个进程同时操作）
  .git/                export 时 init 并首次提交；apply 成功后再提交
```

## 与外部材料的关系

- [`designer-src/`](../../designer-src/README.md)：设计器反编译源码，**在仓库里**，注释里的 `file:line` 指它。
- 注释里的「设计指南 §x.y」指一份**不在本仓库**的前身文档。R1–R7 的条文在代码注释里都有，
  D-n 的含义写在使用处 —— 读代码即可，不必去找那份文档。

## 判据

```bash
go test ./internal/dev/...            # 各包单测（fence 与 tapfile 各约 48 秒）
./tt.exe dev tzc selftest             # 内置对抗用例，不需要真实语料：通过 31，失败 0
TDEV_DEEP=1 go test ./internal/cli/dev -run 'TestCorpusExportVerify|TestCorpusApplySimulation'
                                      # 全语料回归（9–11 分钟）
```

## 子目录

| 包 | 一句话 |
|---|---|
| [model](./model/README.md) | 领域模型与字节偏移→行号定位（纯数据、纯函数） |
| [pkgfile](./pkgfile/README.md) | 包的只读视图、条目分派、zip 字节级重建 |
| [tapfile](./tapfile/README.md) | `.tap` 的 CDATA 感知字节保真改写 |
| [tglfile](./tglfile/README.md) | `.tgl` 的区段边界、占位符、集合锚点 |
| [fgl](./fgl/README.md) | 4GL 块信封判定与源码大纲 |
| [synth](./synth/README.md) | 合成层与两条权限判定链（G1–G7 / S1–S5） |
| [fence](./fence/README.md) | 围栏协议（渲染与解析，互为逆） |
| [verify](./verify/README.md) | 三道闸门、不变量 I1–I15、结构事务 V1–V7 |
| [split](./split/README.md) | 回写拆分：Document' → TapOp + TglPatch |
| [store](./store/README.md) | 工作区落盘、原子写、锁、manifest、git |
| [testutil](./testutil/README.md) | 合成测试包与真实语料发现 |
| [cli](../cli/dev/README.md) | 命令行入口与各动词 |
| [tzs](./tzs/README.md) | `.tzs` 表单包的引擎客户端（另一条线） |
