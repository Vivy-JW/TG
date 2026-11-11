# internal/dict — ERP 数据字典

查 T100 的元数据：表与字段、校验带值、系统分类码、字段画面规格、可复用开窗、系统消息、
参数定义、程序与作业。**两种数据源、一个接口、一份数据族清单。**

## 两种数据源

| 源 | 实现 | 说明 |
|---|---|---|
| **本地镜像** | [./db](./db/README.md) | 本地 SQLite 副本，离线可用、快 |
| **远程直查** | [./live](./live/README.md) | 连客户的 ERP 库现查，数据总是最新的 |

**两者实现同一组查询方法、返回同一批结构**（[./db](./db/README.md) 的 `Source` 接口），
所以命令层不区分本地与远程 —— 由数据源策略决定，命令本身不关心。

数据源的选取顺序（`cli/dict` 的钩子）：

1. `--env` / `--conn <环境名|local>` —— **显式指定即强制**；
2. 配置里的查询数据源（环境名或 `local`）；
3. 都没有 → **在线优先**（只有连环境都没配时才落到本地）。

## 数据族

**`Families` 是唯一的事实来源**：哪些表属于哪一族、谁依赖它、同步顺序 ——
同步清单、数据覆盖状态、各命令帮助里的"本地数据齐不齐"提示全部从它生成。

| 族 | 中文名 | 依赖命令 | 表 |
|---|---|---|---|
| `table` | 表字典 | `r.t` | 表档 / 字段档 / 键值档 / 索引档等 9 张 |
| `scc` | 系统分类码 | `scc` | `gzca_t` 系 4 张 |
| `win` | 可复用开窗 | `r.q` | `dzca_t` 系 5 张 |
| `msg` | 系统消息 | `msg` | `gzze_t`、`gzzal_t` |
| `param` | 参数定义 | `sysp` / `docp` | `gzsz_t` 系 3 张 |
| `prog` | 程序与作业 | `prog` | 程序档 / 作业表 / 应用参数组等 4 张 |
| `progtable` | 程序与表格 | `prog` / `r.t` | `gzdg_t` |
| `subprog` | 子程序与元件 | `prog` | `gzde_t`、`gzdel_t` |

**缺族不是"部分可用"**：依赖它的命令会**整体报错** —— 所以状态表里那种情况行数显示为
"无意义"，而不是给一个会让人误以为能用的数。

## 长跑动作

| 动作 | 实现 |
|---|---|
| 离线镜像拉取（source/interface profile） | [../host/README.md](../host/README.md) |
| 字典同步（远程库 → 本地 SQLite） | [./dbsync](./dbsync/README.md) |

服务端只为这两个动作提供接口（[./server](./server/README.md)），它们也是设置页里那两张卡片。

## 判据

```bash
go test ./internal/dict/... -count=1
go test ./internal/cli/dict -count=1
```

## 细节去哪

- 查询命令的用法与逐条说明 → [`skills/tt-dict/SKILL.md`](../../skills/tt-dict/SKILL.md)
- 输出的形状（信封里为什么有那么多字段） → [../output/README.md](../output/README.md)
- 结果的落盘与翻页 → [./db/README.md](./db/README.md) 与 `tt dict spill`
