# 字典层接线表：`db.Source` 的 30 个查询方法 × GP 侧实测数据

> 这份表把 `internal/dict/db/source.go` 的 `Source` 接口逐个方法对到 GP 的数据上，
> 标明**哪些能直接接、哪些只能给一半、哪些 GP 根本没有**。
>
> **数据来源：2026-11-09 对一套 TOPTEST 环境（主机 / 端口 / 账号见本地 `config.json`，本仓库不记录连接信息）的只读实测**，
> 不是从 T100 推断的。全部是 `SELECT`，没有建表、没有写库。

## 0. 怎么验的

- 连接：`oracle://<库账号>@<主机>:<端口>/<服务名>`（正是 `tt gp` 字典同步用的那条路径：
  `internal/gp/dict.go` 取 `env.DB.Accounts[0]`，不查企业目录）。
- 查询面：`all_tables` / `all_tab_columns` / `all_indexes` / `all_constraints`（物理字典）
  + GP 自己的 `GA*_FILE`、`ZT*_FILE`、`ZE_FILE`。
- 本文所有行数都是实测值；标 **⬜** 的方法，是**实测确认 GP 侧没有对应表**，不是"没查到"。

## 1. 先给结论

| 类别 | 方法数 | 含义 |
|---|---|---|
| ✅ 直连 | 5 | GP 侧数据完整，换个 SQL 就能实现 |
| 🟡 扩同步后能接 | 4 | 数据在 GP 库里，但现有本地表没这一列/这张表 |
| 🟠 只能给一半 | 3 | 能查，但 DTO 里有一部分字段 GP 没有来源 |
| ⬜ GP 没有 | 18 | 实测不存在对应元数据；按 §4.D 的约定返回空、不报错 |
| ⚙️ 通用 | 1 | `Close()` |

**GP 的字典是"物理表结构 + 表/字段中文名 + 程序目录 + 错误消息"**，
比原先估计的乐观一点（表结构这一层很扎实：2,922 张表、61,823 个字段），
但在"程序↔表功能分析、校验带值、分类码、ACC、参数、单据性质"这些 T100 5.x 概念上一片空白。

## 2. GP 侧到底有什么（实测）

### 2.1 物理字典（Oracle 原生，永远和库一致）

| 对象 | 实测数量 | 用途 |
|---|---|---|
| `all_tables`（owner=DS） | **2,922** 张表 | 表清单 |
| `all_tab_columns`（DS） | **61,823** 个字段 | 字段清单（类型/长度/精度/可空/默认值） |
| `all_indexes` + `all_ind_columns` | 有，未全量计数 | 索引 |
| `all_constraints` + `all_cons_columns` | P **2,790**、U **1**、C **15,751**、**R 0** | 主键/唯一/检查；**没有任何外键** |
| `all_views`（DS） | 3 | 视图 |

### 2.2 GP 自己的字典表（`GA*_FILE` / `ZT*_FILE` / `ZE_FILE`）

| 表 | 行数 | 是什么 | 对哪个方法有用 |
|---|---|---|---|
| `zta_file` | **137,270** | **按公司别**的表清单（`zta02` = 公司代码，50 个公司）；本公司与 `ds` 有关的是 2,962 行 | `QueryTableList/Meta` |
| `gat_file` | **5,880** | 表名 → 语言(0 繁/2 简) → 中文名 `gat03`、模块 `gat06`、类型 `gat07` | `QueryTableMeta` |
| `zt_file` | **1,322** | 表主档：`zt01` 表名、`zt02` 繁体名、`zt02e` 英文名、`zt02c` 简中名 | 补表名（覆盖 2,922 张里的 998 张） |
| `gaq_file` | **126,853** | **字段级**：`gaq01` 字段名、`gaq02` 语言、`gaq03` 中文名、`gaq07` 格式码 | `QueryTable.FieldDesc`、`QuerySpecRows` |
| `gau_file` | **1,957** | **字段 → 引用表/引用字段**（`gau01/gau02` → `gau03/gau04`），即"该字段的取值从哪张表来" | `QueryKeys.RefTable/RefField`、`QueryWin*` |
| `gac_file` | **15,327** | 查询（`q_*`）的字段定义：`gac01` 查询名、`gac05/06` 表/字段 | `QueryWinList/Cols`（半个） |
| `gao_file` | **121** | 模块目录：模块、程序目录、报表目录、标志 | `QueryProgInfo.Module`（已有 `t_module`） |
| `gaz_file` | **13,446** | 程序说明：`gaz01` 程序、`gaz02` 语言、`gaz03` 名称、`gaz06` 简中名、`gaz04` 说明 | `QueryProgInfo/List` |
| `gal_file` | **17,810** | 程序 → 模块链接（`gal01` 程序、`gal02` 模块、`gal03` 被链接程序、`gal04` 标志） | `t_prog_link` |
| `gaa_file` | 5,408 | 用户执行程序记录 | 无 |
| `gab_file` | 5,056 | 查询语句档（`q_*` 的 SQL） | 无 |
| `gae_file` | 427,553 | 程序执行/输出记录 | 无 |
| `gak_file` | 5,595 | 程序群组 | 无 |
| `gap_file` | **112,360** | 程序的按键动作（`accept/cancel/control*` …） | 无（可做"程序有哪些动作"的新功能） |
| `gav_file` | **176,082** | 程序用到的变量/字段（`gav01` 程序、`gav02` 变量、`gav12` 长度） | `QuerySpecRows` 的 DisplayLen 可参考 |
| `gax_file` | 3,514 | 程序执行权限 | 无 |
| `gay_file` | **2** | 语言档：**0 = 繁體中文，2 = 简体中文** | `QueryMsgLangs`、`QueryMsgs.Lang` |
| `ze_file` | **42,031** | 错误消息：`ze01` 编号、`ze02` 语言(0/2)、`ze03` 文本、`ze04` 日期、`ze05` 标志 | `QueryMsgs`、`QueryMsgLangs` |
| `ooc_file` | 4 | 会计科目对照（不是 ACC） | 无 |
| `gzl_file` / `gzln_file` | 3 / 1 | 用户→程序链接（不是分类码） | 无 |

### 2.3 实测确认**不存在**的

| 找什么 | 结果 |
|---|---|
| T100 5.x 的 `dz*` 字典家族（`dzca_t/dzcb_t/dzcc_t/dzcd_t/dzce_t/dzch_t/dzep_t/dzeb_t/dzea_t`） | `all_tables like 'dz%'` → **0 张** |
| 分类码/ACC/参数/单据性质（`gzca_t`、`gzaa_t`、`gzsz_t`、`gzsy_t`、`oocq`、`oocql`、`gzzz`） | `gz%` 只有 `gzl_file`/`gzln_file` 两张，且分别 3 行/1 行 |
| 程序↔表使用关系（`gzdg_t`） | 不存在 |
| 子程序元件（`gzde_t`） | 不存在 |
| 外键（R 约束） | **0 条**（全库只有 P/U/C） |
| `ZTB/ZTC/ZTD/ZTE_FILE` | 存在，但只覆盖 **62/34/50/12** 张 `tc_*` 客制表，**不是**全量字段/索引/键定义 |

### 2.4 跑一次真实同步（`POST /api/gp/sync`，2026-11-09 18:00）

不是推测：在 TOPTEST 上跑完整同步，**1 分 7 秒**，产物 11.7 MB：

| 本地表 | 行数 | 备注 |
|---|---|---|
| `t_table` | 2,962 | 来自 `zta_file` 公司 `ds`；其中 **61 张在本实例没有物理表**；96 张**没有中文表名**、64 张**没有模块** |
| `t_column` | 60,870 | 覆盖 **2,901** 张表；2,132 个字段**没有中文名** |
| `t_index` | 3,318 | |
| `t_constraint` | 18,301 | P 2,790 + U 1 + C 15,510 左右；**R 0** |
| `t_module` | 121 | `gao_file` 121 行 |
| `t_prog` | 6,465 | `gaz02='2'`（简体名）；键是**作业编号**（`07448`/`123`），不是 4GL 程序名 |
| `t_prog_link` | **0** | 建表但同步**不填**（见 §5.3） |

同步完成后 `tt gp table ima_file --human` 直接可用：314 个字段，中文名、类型、可空、格式码（`001`/`036`…）都在。

**对账结论（拿 Oracle 逐个表比）**：`all_tab_columns` 在 `DS` 下是 2,925 个对象/61,823 个字段，
本地是 2,901 张表/60,870 个字段，**逐表比没有一个表少字段**——差的 24 个对象（953 个字段）全是
`zta_file` 里没有的视图与备份表（`ac`、`ida_bak0216`、`idd_bak1213`、`tc_imb_bak1205` …）。
所以"表档"和"物理字典"的差不是 bug，是**两套口径**：表档跟着公司别走，字段跟着 schema 走（§5.2）。

## 3. 方法级接线表

约定：**"空"= 返回零值，不报错**（`IsMissingTable` 已经在做同样的降级）。

### 3.1 表结构

| # | 方法 | 用途 | DTO 要的字段 → GP 来源 | 结论 |
|---|---|---|---|---|
| 1 | `QueryTableMeta` | 表头 | `tab/tab_cn/module/tab_type` ← `t_table`（已由 zta+gat 同步） | ✅ |
| 2 | `QueryTable` | 字段清单 | `seq,col,col_cn,dtype,dlen,remark` ← `t_column`；`IsPK`←`t_constraint(ctype='P')`；`Required`←`nullable='N'` | ✅ |
| 3 | `QueryKeys` | 键值 | `KeyType/KeyName/KeyField` ← `t_constraint`（P/U）；**`RefTable/RefField` 改从 `gau_file` 取**（GP 没有外键，但有 1,957 行"字段→引用表"） | 🟡 扩同步后能接 |
| 4 | `QueryIndexes` | 索引 | `idx/uniqueness/cols` ← `t_index` | ✅ |
| 5 | `QueryTableList` | 列表/搜索 | 同 #1 | ✅ |

### 3.2 程序与作业

| # | 方法 | 用途 | GP 来源 | 结论 |
|---|---|---|---|---|
| 6 | `QueryProgInfo` | 程序详情 | `t_prog`（`prog,name,memo,name2,module`，来自 `gaz_file`）；`Category/Cust/RefMain/RunCmd/Status` **无** | 🟠 一半 |
| 7 | `QueryProgJobs` | 作业登记 | GP 没有"作业" | ⬜ |
| 8 | `QueryProgList` | 程序列表 | 同 #6；`Category/Cust` 空、`JobCount` 恒 0 | 🟠 一半 |
| 9 | `QueryProgTables` | 程序用了哪些表 | 库里没有；只能扫 4GL 源码（属新功能） | ⬜ |
| 10 | `QueryTablePrograms` | 反查程序 | 同上 | ⬜ |
| 11 | `QuerySubProgInfo` | 子程序详情 | GP 的子程序是 4GL `FUNCTION`，不在字典 | ⬜ |
| 12 | `QuerySubProgList` | 子程序列表 | 同上 | ⬜ |

### 3.3 校验 / 规格

| # | 方法 | GP 来源 | 结论 |
|---|---|---|---|
| 17 | `QuerySpecRows` | 只有 `t_column.fmt/dflt` + `gaq_file.gaq07`（格式码）；22 列里能给 6～8 列 | 🟠 一半 |
| 13–16 | `QueryCheck*` | GP 没有 `r.v` 校验带值这套 | ⬜ |

### 3.4 分类码 / ACC / 开窗 / 参数

| # | 方法 | GP 来源 | 结论 |
|---|---|---|---|
| 18–20 | `QueryScc*` | 没有分类码字典（`gzca_t` 不存在） | ⬜ |
| 21–22 | `QueryACC*` | 没有（`oocq/oocql/gzzz` 不存在；`ooc_file` 只有 4 行会计科目） | ⬜ |
| 23 | `QueryWinList` | **可以映射**到 `gac_file` 的 `q_*` 查询（15,327 行里的 distinct 查询名） | 🟡 半个 |
| 24/25 | `QueryWinHeaders/Params` | 没有"开窗头/参数"这一层（GP 的开窗是 4GL 代码 + 查询） | ⬜ |
| 26 | `QueryWinCols` | `gac_file` 给了每个查询用了哪些表的哪些字段 | 🟡 半个 |
| 27 | `QueryParam` | 没有参数定义档 | ⬜ |
| 28 | `QueryDocTypes` | 没有单据性质档 | ⬜ |

### 3.5 消息

| # | 方法 | GP 来源 | 结论 |
|---|---|---|---|
| 29 | `QueryMsgs` | `ze_file`：`ze01→Code`、`ze02→Lang`、`ze03→Text`、`ze04→日期`；`Action/ExecProg/Detail/TypeCode/ForceWin/Status` **无** | 🟡 扩同步（建 `t_msg`）后能接 |
| 30 | `QueryMsgLangs` | `SELECT DISTINCT ze02 FROM ze_file` + `gay_file` 给语言名（0 繁體中文 / 2 简体中文） | ✅ |

## 4. 除了这 30 个方法，还有三处必须先改

1. **数据源解析不认 GP。** `internal/cli/dict/source.go:218 openRemoteSource` 要"环境@企业"经
   `entdir` 换账号。**实测澄清**：GP 里没有 T100 那种"企业→账号"映射，但有**公司别 `zta02`**
   （`ds`/`hk`/`bj`/`demo1` …共 50 个），且**通常等于数据库账号名**（本实例账号就是 `ds`，
   `zta_file` 里公司也是 `ds`）。所以改法是：环境 `product=="gp"` 时，账号直接用
   `db.accounts[0]`，公司别默认取账号名，允许在环境配置里覆盖。
2. **本地库路径两套。** `tt dict` 的 `local:<环境>` 指向 T100 字典库，GP 的库在
   `internal/gp.DictPath()`（`~/.tt/gp-dict/<环境>.db`）。改法同上一条。
3. **命令组还没挂 GP。** `tt gp` 只有 `table|link|msg`，`tt dict` 的 10 个子命令都还没对 GP 开。

## 5. 现有同步的三个口径问题（实测发现）

这三个不是"要不要接线"的问题，是**现在就已经存在的不一致**：

1. **`t_table` 的公司别与 schema 是硬编码的。** `internal/gp/dict.go:186` 写着
   `lib, owner := "ds", "DS"`，紧跟着 `if env.DB.Service != "" { owner = "DS" }` 是**死分支**
   （两边一样）。后果：换个公司/实例，`t_table` 里还是 `ds` 的表。
2. **`t_table` 与 `t_column` 口径不一致。** `t_table` 只收公司 `ds` 的 2,962 张（其中 61 张
   在本实例上**没有物理表**），`t_column` 收整个 `DS` schema 的 2,901 张物理表/60,870 个字段
   （另有 24 个对象——视图与备份表——不在表档里）。→ 查一张"有字段、没表头"的表，`--meta` 会返回空；
   查一张"有表头、没物理表"的表，字段列表为空。
3. **`t_prog_link` 建了表却不填，`LookupModule` 的 p_link 覆盖等于没生效。** `dict.go:111-114` 说
   "`gal_file` 很大，全表扫会让 50 秒的同步卡到数分钟"——实测 **17,810 行**、入口行（`gal03=gal01`）
   **5,561 行**，一条 `SELECT` 就扫完了。现在 `t_prog_link` 恒为 0 行，`internal/gp/lookup.go:44`
   的第一步永远查空，只能掉到 `t_prog.module` 兜底；而 `t_prog` 的键是**作业编号**（`07448`），
   不是程序名（`csfi300_icd`），所以对 `csfi300_icd` 这类程序**两步都查不到**，最终退回
   `gpModuleOf` 取前缀。填了这张表，`csfi300_icd` 才能正确解析到 `CSF`（实测该程序在 `gal_file`
   里有且只有一行 `gal03=gal01`，模块 `CSF`；其余 13 行是 AEC/APM/ASF/AWS 的子程序链接）。

另外 `t_prog` 只同步 `gaz02='2'`（简体名，6,465 支），繁体名（`gaz02='0'`，6,808 支）没进来；
`ze_file` 两种语言各约 2.1 万行，现在完全没同步。

## 6. 建议的实施顺序

1. **第一步（确定性最高）**：把 #1/#2/#4/#5 做成 GP 版 `Source`（`t_table`/`t_column`/`t_index`/
   `t_constraint` 现成）；顺手修 §5 的三处口径（公司别从环境配置来、`t_table`/`t_column`
   对齐到同一批表、`t_prog_link` 照常填）。
2. **第二步**：#3（`gau_file` → `RefTable/RefField`）、#29（新建 `t_msg`，从 `ze_file` 同步）、
   #30（语言名从 `gay_file` 取）、#17（`gaq07` + `gav12` 补规格列）。
3. **第三步**：#23/#26（`gac_file` → 查询/开窗的"半个"）；#9/#10 留给"扫 4GL 源码"的新功能。
4. **第四步**：剩下 18 个 ⬜ 统一返回空 + `notes: ["GP 没有这项元数据"]`，并在 `tt dict` 里
   把这 8 个子命令标成"GP 不支持"。

## 7. 两个取舍

- **DTO 留空 vs 给 GP 窄结构**：倾向留空。T100 的 `ProgInfo` 有 14 个字段，GP 只能给 5 个；
  但脚本/前端已经在按 T100 的形状取，改形状的破坏面更大。
- **空实现报不报错**：不报错（exit 0 + 空数组 + `notes`），否则脚本里 `tt dict scc list`
  在 GP 上就成了一条失败命令——而它本来只是"GP 没有 SCC"。
