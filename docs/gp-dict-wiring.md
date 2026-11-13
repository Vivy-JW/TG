# 字典层接线表：`db.Source` 的查询方法 × GP 侧实测数据

> 这份表把 `internal/dict/db/source.go` 的 `Source` 接口逐个方法对到 GP 的数据上，
> 标明**哪些能直接接、哪些只能给一半、哪些 GP 根本没有**。
>
> **数据来源：2026-11-09 对一套 TOPTEST 环境（主机 / 端口 / 账号见本地 `config.json`，本仓库不记录连接信息）的只读实测**，
> 不是从 T100 推断的。全部是 `SELECT`，没有建表、没有写库。
>
> **2026-11-13 复核**：接口随命令面收口从 30 个方法瘦到 **20 个** —— `r.v`(check)、`desc`(spec)、
> `scc`、`acc`、`sysp`/`docp`(param) 六条命令已删（GP 侧那几张表实测全 `ORA-00942`），
> 它们的方法也一并从接口移除；§3 表里对应的行标成「已删」。同期新增了 `QueryUITexts`（画面文字，`gae_file` →
> `t_ui_text`）与 `QueryProgLinks`（程序链接，`gal_file` → `t_prog_link`，本地也能查），
> `QueryProgTables`/`QueryTablePrograms` 也从「⬜ 只能扫源码」变成 ✅（同步 `zr_file` 后本地与远程都出数，实测 164 张）。

## 0. 怎么验的

- 连接：`oracle://<库账号>@<主机>:<端口>/<服务名>`（正是 `tg dict db sync` 字典同步用的那条路径：
  `internal/dict/dbsync/gp.go` 取 `env.DB.Accounts[0]`，不查企业目录）。
- 查询面：`all_tables` / `all_tab_columns` / `all_indexes` / `all_constraints`（物理字典）
  + GP 自己的 `GA*_FILE`、`ZT*_FILE`、`ZE_FILE`。
- 本文所有行数都是实测值；标 **⬜** 的方法，是**实测确认 GP 侧没有对应表**，不是"没查到"。

## 1. 先给结论

| 类别 | 方法数 | 含义 |
|---|---|---|
| ✅ 直连 | 16 | GP 侧数据完整（含同步进本地库的 `t_*` 与直查 `GA*/ZT*/ZE_FILE`） |
| ⬜ GP 没有 | 4 | 实测不存在对应元数据；按 §4.D 的约定返回空、不报错 |

**GP 的字典是"物理表结构 + 表/字段中文名 + 程序目录与程序用表 + 错误消息 + 画面文字 + 开窗定义"**，
表结构这一层很扎实（2,962 张表 / 60,870 个字段，含 61 张只登记不建列的同义词），
但"校验带值、分类码、ACC、参数、单据性质、作业登记"这些 T100 5.x 概念上一片空白 —— 相关命令已下线，不再留空壳。

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

同步完成后 `tt dict r.t ima_file --format table` 直接可用：314 个字段，中文名、类型、可空、格式码（`001`/`036`…）都在。

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
| 3 | `QueryKeys` | 键值 | `KeyType/KeyName/KeyField` ← `t_constraint`（P/U）；`RefTable/RefField` 从 `gau_file` 取（GP 没有外键，但有"字段→引用表"行） | ✅ |
| 4 | `QueryIndexes` | 索引 | `idx/uniqueness/cols` ← `t_index` | ✅ |
| 5 | `QueryTableList` | 列表/搜索 | 同 #1 | ✅ |

### 3.2 程序与作业

| # | 方法 | 用途 | GP 来源 | 结论 |
|---|---|---|---|---|
| 6 | `QueryProgInfo` | 程序详情 | `t_prog`（`prog,name,memo,name2,module,category,run_cmd,name_tw`，来自 `gaz_file`/`zz_file`/`gal_file`） | ✅ |
| 7 | `QueryProgJobs` | 作业登记 | GP 没有"作业登记"这一层（`gzou_t` 之类不存在） | ⬜ |
| 8 | `QueryProgList` | 程序列表 | 同 #6，另带 `p_link` 解析出的模块 | ✅ |
| 9 | `QueryProgTables` | 程序用了哪些表 | `t_prog_table` ← `zr_file`（118,420 行，含 S/I/U/D 操作码）；远程直查同源 | ✅ |
| 10 | `QueryTablePrograms` | 反查程序 | 同上（按表反查） | ✅ |
| 11 | `QuerySubProgInfo` | 子程序详情 | GP 的子程序是 4GL `FUNCTION`，不在字典 | ⬜ |
| 12 | `QuerySubProgList` | 子程序列表 | 同上 | ⬜ |
| 13 | `QueryProgLinks` | 程序链接 | `t_prog_link` ← `gal_file`（17,810 行，入口行 `gal03=gal01`）；本地 3ms | ✅ |
| 14 | `QueryUITexts` | 画面文字 | `t_ui_text` ← `gae_file`（427,553 行，多语系）；`tg interface` 的标签源 | ✅ |

### 3.3 校验 / 规格 —— 已下线

| # | 方法 | 结论 |
|---|---|---|
| 17 | `QuerySpecRows`（`desc`） | **命令已删**（2026-11-11）。GP 的字段规格 22 列里只能给 6～8 列，值不值不足以撑一个命令 |
| 13–16 | `QueryCheck*`（`r.v`） | **命令已删**。GP 没有校验带值这套 |

### 3.4 分类码 / ACC / 开窗 / 参数（SCC、ACC 已删；开窗已接）

| # | 方法 | GP 来源 | 结论 |
|---|---|---|---|
| 18–20 | `QueryScc*`（`scc`） | **命令已删**（2026-11-11）。GP 没有分类码字典（`gzca_t` 不存在） | — |
| 21–22 | `QueryACC*`（`acc`） | **命令已删**。GP 没有（`oocq/oocql/gzzz` 不存在；`ooc_file` 只有 4 行会计科目） | — |
| 23 | `QueryWinList` | `gab_file` + `gae_file`（wintitle） | ✅ 已接（`r.q` 列表） |
| 24/25 | `QueryWinHeaders/Params` | `gab_file` 单头；GP 没有开窗参数这一层（`dzcb_t` 无对端） | ✅ 单头 / ⬜ 参数 |
| 26 | `QueryWinCols` | `gac_file` + `gaq_file` 显示名 + `all_tab_columns` 列类型 | ✅ 已接 |
| 27 | `QueryParam`（`sysp`） | **命令已删**（2026-11-11）。GP 没有参数定义档：参数是各模块参数档里的列（`aaa_file`/`aaz_file`/`apz_file`…，表说明含"参数"的 77 张） | — |
| 28 | `QueryDocTypes`（`docp`） | **命令已删**。GP 没有单据性质档，也没有"参数绑定单据别"这层 | — |

### 3.5 消息

| # | 方法 | GP 来源 | 结论 |
|---|---|---|---|
| 15 | `QueryMsgs` | `t_msg` ← `ze_file`（42,031 行；`ze01→Code`、`ze02→Lang`、`ze03→Text`） | ✅ |
| 16 | `QueryMsgLangs` | `SELECT DISTINCT ze02` + `gay_file` 给语言名（0 繁體中文 / 2 简体中文） | ✅ |

## 4. 三处前置问题 —— 已全部改完（2026-11）

1. **数据源解析不认 GP**：已改为环境 `product=="gp"` 时账号直接取 `db.accounts[0]`、schema 取
   「营运据点」（`azp_file.azp03`，默认 `ds`），解析链 `--at <环境>@<据点编号>` → `gp.ResolveSchema`
   （`dc9fad8`）；`db sql`/`db probe` 身份只有环境（`735f22d`），`--ent` 已删。
2. **本地库路径两套**：已统一到 `internal/dict.DictPath`（`DictDir` → `TT_GP_DICT_DIR` → `~/.tt/gp-dict/<环境>.db`），
   `local:<环境>` 与网页同一条路径（`ff3215c`）。
3. ~~命令组还没挂 GP~~：`tg gp` 整组已删，表/程序/消息/开窗/画面文字/链接/同步/据点都在
   `tg dict` 与 `tg site`（`5ac4903`）。

## 5. 三个口径问题 —— 已修（2026-11）

1. ~~公司别硬编码 `ds`~~：同步固定按 `ds`（`const SyncCompany`，ds 表档最全 2,962 张），
   直查按「营运据点」配置（`gp.Company`），两件事分开（`068b19f`/`14413d9`）。
2. ~~`t_table` 与 `t_column` 口径不一致~~：`t_table` 记 `zta07` 种类（`T` 实体 / `S` 同义词），
   61 张同义词只登记不建列，`tg dict r.t` 会打「种类: 同义词」并解释为什么不列字段（`92d6a95`）。
3. ~~`t_prog_link` 建了不填~~：现在照常填（17,810 行），本地 `dict prog --links` 3ms 出数（`2110c95`）。

## 6. 当时的实施顺序（2026-11-13 已全部落地）

这一节保留的是 11-09 立的计划与最终结果，方便回头看"当初以为要做什么、实际做成了什么"：

1. **第一步（确定性最高）**：#1/#2/#4/#5 接成 GP 版 `Source`（`t_table`/`t_column`/`t_index`/`t_constraint`）
   + §5 三处口径 → **已完成**（`a75acb3`/`92d6a95`/`068b19f`）。
2. **第二步**：#3（`gau_file` → `RefTable/RefField`）、消息（`t_msg` ← `ze_file`）、语言名（`gay_file`）、
   程序链接（`t_prog_link` ← `gal_file`）→ **已完成**（`e886087` 之后的几笔，行数见 §1/§3.5）。
3. **第三步**：开窗（`gab_file` + `gac_file` + `gae_file`/`gaq_file`）→ **已完成**（`e886087`，明细里能现拼 SQL）；
   程序用表（`zr_file` → `t_prog_table`，118,420 行）也从"扫 4GL 源码"改成了字典查询 → **已完成**（`0e2f790`）。
4. **第四步**：剩下 ⬜ 统一返回空、把没有对端的子命令标成"GP 不支持" → **改成了更彻底的做法**：
   直接删命令（`r.v`/`desc`/`scc`/`acc`/`sysp`/`docp`，`1df9d4f`…`baee36e`），不再留空壳。

**现在只剩 4 个 ⬜**：`QueryProgJobs`（GP 没有作业登记）、`QuerySubProgInfo`/`QuerySubProgList`（子程序不在字典里，
是 4GL `FUNCTION`）、`QueryWinParams`（GP 的开窗没有参数层）。这 4 个返回空集，不报错、不编数据。

## 7. 两个取舍

- **DTO 留空 vs 给 GP 窄结构**：倾向留空。T100 的 `ProgInfo` 有 14 个字段，GP 只给得出其中一部分
  （`prog`/`name`/`name2`/`module`/`category`/`run_cmd`/`name_tw`）；但脚本/前端已经在按这个形状取，
  改形状的破坏面更大。
- **空实现报不报错**：不报错（exit 0 + 空数组 + `notes`），否则脚本里 `tg dict r.t ima_file`
  在 GP 上就成了一条失败命令——而它本来只是"GP 没有某张字典表"。
  实际做法：GP 根本没有对端的族（`r.v` / `desc` / `scc` / `acc` / `sysp` / `docp`）直接删命令，不留空壳。
