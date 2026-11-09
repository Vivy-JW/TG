# internal/dev/store — 工作区

工作区的落盘与读取：`export` 写出它，`apply` 从它读基线、成功后把它推进一格。布局固定，
不随环境变化。

```
<dir>/
  prog.full.4gl        唯一编辑文件：带围栏的渲染文档
  manifest.json        人机共读索引（每个 Region 的权限与不可编辑原因）
  snapshot/
    index.json         原包条目清单：name / sha256 / size / role
    entries/<条目名>    原包每个条目的逐字节拷贝（恢复源；apply 不从这里读）
  .tdev/
    base.full.4gl      导出时 prog.full.4gl 的字节拷贝（gate1 的比对基线）
    base.sha256        该基线的 sha256
    regions.json       Region 表 + 字节区间 + 环境摘要（含包级解锁标志）
    section-state      工作区的解锁**意图**（可能不存在）
    lock               互斥锁（防两个进程同时操作）
  .git/                export 时 init 并首次提交；apply 成功后再提交
```

## 三条硬约束

| | 约束 | 实现 |
|---|---|---|
| R6 | 一切落盘走**原子写**（同目录临时文件 → Rename），禁止先删后建 | `AtomicWrite` |
| R7 | 时间戳不是功能：同一输入必须产出逐字节相同的工作区 | `manifest.json` / `regions.json` 里不含时钟 |
| — | **不用库调 git**：git 不是依赖，只是版本留痕 | 走 `git` 命令行，并显式关掉行尾转换，让 git 里的字节 == 工作区里的字节 |

git 是**可选**的：环境里没有 git 可执行文件时，`export` 照常产出完整工作区，只是不建版本库。

## 解锁状态有三个来源，合并规则只有一条

| 来源 | 是什么 |
|---|---|
| 包的根标志（`regions.json` 里记着） | **包级事实**：这个包本身是不是解开态 |
| `.tdev/section-state` | 工作区的**意图**：`unlock` 命令迁移出来的（记着是谁解锁的） |
| `manifest.json` 里的旧字段 | 早期版本用开关放行区段时留下的；**只读**，用来把老工作区认成已解锁 |

`EffectiveUnlockState()` 合并它们，规则是：**包级事实优先**，否则看工作区意图。
文件不存在且旧字段也没有 → 只读。

## apply 成功之后：基线前进一格

`UpdateAfterApply(doc, fenced)` 把工作区推进到"刚写出去的那一版"：

1. 重写 `base.full.4gl` 与 `base.sha256`（新的比对基线）
2. 重建 `regions.json`（由新文档与新基线重算区间）
3. 刷新 `manifest.json`

**为什么要重写基线而不是只记一个 hash**：改名一类的操作会改变点的身份，只有从写出的新包重新
合成，基线才与包一致；这一步不做，下一次 `apply` 就会拿错基线去比对。

## 快照与备份

| 入口 | 作用 |
|---|---|
| `SnapshotPkg(pkg)` | 把原包每个条目逐字节写进 `snapshot/entries/`，刷新 `index.json` |
| `RefreshPackage(pkg)` | apply 之后用新包刷新条目清单与包的 sha256 |
| `BackupPackage(srcPath)` | 覆盖前把**上一版包**整文件备份一份（可回滚） |
| `SnapshotEntry(name)` | 取某个条目的原始字节（恢复源） |

## manifest 里的工具标识

`manifest.json` 记着两个字段：`tool` 与 `tool_version`。**`tool` 的值写作 `"tdev tzc"`**，
而且**全仓没有任何代码读它** —— 保持这个字符串不变是有意的：它是持久化字段，
改名只会让已有工作区的 manifest 看起来"换了身份"，没有收益。

## 锁

`Lock()` 用 `O_CREATE|O_EXCL` 原子抢占 `.tdev/lock`（两个进程不可能同时成功）。
已存在且未陈旧 → IO 错误（退出码 5）；**文件时间超过 30 分钟**视为上次进程已死 → 删掉重试一次。
锁记录里带着进程号与主机名，只用于诊断归属 —— 那是工作区里唯一允许出现时钟的地方。

## 入口

| 入口 | 作用 |
|---|---|
| `Create(dir, doc, pkg, fenced)` / `Open(dir)` | 建 / 开工作区（目标已存在、或工作区不完整 → 拒绝） |
| `ReadEdited` / `WriteEdited` / `ReadBase` | 编辑文件与基线 |
| `Manifest()` / `Regions()` / `EffectiveUnlockState()` | 读索引、Region 表、综合两处事实后的解锁状态 |
| `WriteSectionState` / `ReadSectionState` | 工作区的解锁意图 |
| `UpdateAfterApply(doc, fenced)` | 基线前进一格（见上） |
| `Lock` / `GitInit` / `GitCommit` / `GitHead` | 互斥与版本留痕 |
| `IOError` | 环境 / IO 类错误，退出码 5 |

## 判据

```bash
go test ./internal/dev/store -count=1
# TestCreateWorkspaceLayout / TestCreateRefusesExistingWorkspace / TestOpenRejectsIncompleteWorkspace
# TestAtomicWriteOverwrite / TestAtomicWriteRenameOntoDirectoryFails
# TestLockMutualExclusionAndStale / TestGitLifecycle
# TestRefreshPackageUpdatesRecordedPkg / TestBackupPackageKeepsPrevious
```

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 工作区布局（目录名、文件名） | **已经存在的工作区会失效**，并且兼容逻辑（旧字段、旧状态文件）要一起改；本节的布局表就是它的权威描述，改了要同步 |
| `UpdateAfterApply` | 少刷新一样东西，第二次 `apply` 就会拿错基线（改名一类的操作尤其明显） |
| 序列化结构（`manifest.json` / `regions.json`） | 它们是**人机共读**的：字段变了，读工作区的人与 AI 都会受影响；同时 `MarshalJSONStable` 保证的"同一输入逐字节相同"必须保持 |
| 锁策略 | 并发语义与"上次进程已死"的判定；改错了会表现为"明明没有别的进程却锁不上" |
| 快照 / 备份策略 | 回滚能力（`BackupPackage` 是覆盖前的最后一道保险） |

红线 R6（原子写）与 R7（无时钟）在这一层落地；判据里那条"同一输入产出逐字节相同的工作区"
主要靠本包守住。

## 细节去哪

- 工作区里那份文档是怎么合成与渲染的 → [../synth/README.md](../synth/README.md)、[../fence/README.md](../fence/README.md)
- 逐条目写回计划从哪来 → [../split/README.md](../split/README.md)
- 解锁命令怎么迁移状态 → [cli](../../cli/dev/README.md)
