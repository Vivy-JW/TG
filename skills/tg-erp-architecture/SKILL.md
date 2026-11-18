---
name: tg-erp-architecture
description: 阅读、修改或排查鼎捷 TOP GP（Genero BDL / 4GL）的程序与数据库时触发；覆盖环境与三区、目录与文件类型、程序/表/字段/索引命名、单档与双档骨架、画面与报表、数据字典(p_zta)与客制规范，以及用 tg 命令查这些内容的入口。
---

# TOP GP 架构

TOP GP 是鼎捷 ERP 的 Genero BDL（4GL）版：服务端在 Linux 上跑 `fgldb`（DVM），源码是 `.4gl`，画面是 Genero Studio 做的 `.4fd` → 编译成 `.42f`，客户端是 GDC（Qt，Windows/Linux/macOS）。数据库可挂 Oracle 或金仓。

和 T100 的关系：T100 是同一套业务内核的上一代（设计器 + 代码生成器那一套）。**TG 只面向 TOP GP**，本文件讲的是 GP；最后一节列出读码时最容易踩混的 T100 差异。

---

## 1 服务器与环境

### 1.1 三区（正式 / 测试 / 标准）

| 区 | 实例根（例） | 用途 |
|---|---|---|
| 正式区 | `/u1/topprod` | 生产，只读观察 |
| 测试区 | `/u1/toptest` | 开发与验证，TG 默认连这里 |
| 标准区 | `/u1/topstd` | 标准程序与标准表的对照基准 |

三区各自独立：各有一套实例目录、各自连一个库（同一套 schema 名，不同库实例）。所以「这个字段标准区有没有」要用标准区环境去查，不要拿测试区的库推断。

### 1.2 实例下的两棵树

```
/u1/<实例>/            ← 不带参数的 tg file ls 列的就是这里
  ├─ tiptop/           $TOP：标准代码，模块直接在第一层（abm/4gl/…）
  ├─ topcust/          $CUST：客制代码，同样是 <模块>/4gl/…
  └─ …                 work/ log/ 等运行期目录
```

- 路径不带前缀 = `$TOP`（`abm/4gl/abmi100.4gl`）；客制写 `topcust/<模块>/4gl/<程序>.4gl`。
- 同一支作业可能两处都有：标准版在 `tiptop/`，客制版在 `topcust/`。**先看哪边有、再比差异**，不要默认只有一份。
- 环境与据点：`tg env list` 看环境；`tg site list --env <环境>` 看据点编号（`--at <环境>@<据点编号>` 只换 schema，账号恒为环境里配的 `db.accounts[0]`）。

### 1.3 常用环境变量

| 变量 | 含义 |
|---|---|
| `FGLSERVER` | 画面去向（GDC 的 ip:port）。TOP GP 由 `r.d2` 起 `fgldb`，画面由 fgldb 交给 GDC 直连 |
| `TOPENT` / 企业编号 | **GP 没有这一层**：一个库一个账号，schema 由据点（`azp_file.azp03`）决定 |
| `LANG` / 语系 | 多语言走 `p_perlang`，画面文字与字段说明都按语系取 |

---

## 2 目录与文件类型

### 2.1 模块目录（以 `abm` 为例，15 个子目录）

| 子目录 | 内容 | 读码时的用处 |
|---|---|---|
| `4gl` | `.4gl` 源码 | **主战场**，业务逻辑都在这里 |
| `4fd` | Genero Studio 画面源（XML，`ManagedForm`/`RecordField`） | 想知道画面上有哪些字段、顺序、属性 |
| `per` | 由 `4fd` 导出的 `.per` 画面规格（文本） | 只想快速看布局时比 4fd 好读 |
| `42f` | 编译后的画面 | 运行期实际加载的，改布局要回头改 4fd |
| `42m` | 编译后的模块；另有 `abm_abmf001.4gl` 这种链接形态 | 判断某支程序有没有被链接进模块 |
| `42r` | 报表可执行（编译产物） | 报表类作业的产物 |
| `sdd` | `.sdd` 作業說明書（繁体 Big5） | **改代码前先读**：作業目的、操作功能、逐栏位说明、资料处理 |
| `rep` | `.out` 报表定义（CR 报表数据源） | 报表取数逻辑 |
| `sch` / `schm` | 排程与权限相关档 | 批处理类作业 |
| `4pw` / `per` / 其它 | 工具产物 | 一般不用读 |

另外模块目录下还有 `.4dd`（数据结构导出），可以当「这支程序涉及哪些表」的快速索引。

### 2.2 顶层公用目录

| 目录 | 内容 |
|---|---|
| `lib` | `cl_*` 共享程序（一档一函式），如 `cl_chk_act_auth`、`cl_ui_locale` |
| `sub` | 一般子程序 `s_*`、查询子程序 `q_*`、报表片段 `*_rep` |
| `qry` | 查询/开窗程序与开窗定义相关的程序 |
| `config` | `top.global`（全局变量档） |
| `schema` | 表结构导出/读入用的 schema 文件 |
| `bin` | 可执行与工具 |
| `ora` / `work` / `rest` / `trigger` / `doc` / `tool` | 数据库相关/工作区/接口/触发器/文档/工具 |

读一支作业的顺序：`sdd`（干什么）→ `4gl`（怎么干）→ `4fd`/`per`（长什么样）→ `rep`（报表取数）→ `lib`/`sub`（复用的公共逻辑）。

---

## 3 命名规则

### 3.1 程序：七个字符（全小写）

```
a b m i 1 0 0
│ │ │ │ └─┴── 作业代码（三位数）
│ │ │ └────── 第四码：程序类型
│ └─┴──────── 模块代码（三位字母）
```

- **前三码 = 模块**：`aap`、`aim`、`apm`（采购）、`axm`（销售）、`abm`（产品结构）、`asf`、`azz`（系统）…（模块清单见服务器 `tiptop/` 第一层目录；模块中文名看 `tg sdd` 第一行，例 `abm` = 產品結構管理系統、`axm` = 銷售管理系統）。
- **第四码 = 程序类型**：手册列六类 —— 维护作业、报表作业、参数设置、批处理、交易处理、查询作业。实际代码里常见 `f` / `i` / `p` / `t` / `r`，同一个字母在不同模块语义偏宽（`abmi100` 是「工程 B O M 模擬維護作業」、`abmi720` 是「工程變異單資料維護作業」、`axmt410` 是「訂單維護作業」、`abmp610` 是「產品結構偵錯作業」、`abmr710` 是报表）。
  **判断这支作业是干嘛的，权威是 `tg sdd <作业>` 里的作業名稱，不是第四码。**
- **第五~七码 = 作业代码**：三位数，同一模块内唯一。
- **附属档**：七码后加 `_01`、`_02`…（手册例 `aimi1001h` 表示附属于 `aimi100` 的子档案）；变体后缀如 `_icd`（查询/输入变体）、`_dis`（显示）、`_slk`（关联/多档）、`_wf`（流程）。
- **客制程序**：把标准程序复制到客制模块下同名，再改「程序链接记录维护作业」的链接模块。所以 `topcust/cbm/4gl/abmi600.4gl` 和 `tiptop/abm/4gl/abmi600.4gl` 可能是同一支作业的两份实现 —— 改之前先确认运行期链接到哪边。

### 3.2 表、字段、索引

| 对象 | 规则 | 实例 |
|---|---|---|
| 表 | 三码小写 + `_file` | `ima_file`（料件）、`bma_file`（BOM）、`gef_file` |
| 客制表 | `tc_` + 三码 + `_file` | `TC_AAA_FILE`、`TC_BAS_FILE`、`TC_ADM_FILE` |
| 字段 | 表代码 + 两位流水，流水不足可扩三位 | `IMA01`、`IMA02`、`IMA021`、`IMA1001`、`IMA17_FAC` |
| 索引 | 表代码 + `_PK` / `_nn` | `IMA_PK`、`IMA_03`、`IMA_12` |
| 自定义栏位 | 放客制表里，命名 `tc_` + 基名 + `ud` + 序号 | `tc_aaud01`、`tc_aaud02` |

- 标准表里**没有** `tc_` 前缀的自定义栏位（`IMA_FILE` 实测 0 个 `tc%` 栏位）：客制栏位一律进客制表，不要指望在 `ima_file` 上多一列。
- 主档表普遍带审计/归属栏位（客制规范要求单档、双档单头/单身表都要有）：资料有效码、资料所有者、资料所有群、资料建立日期、资料修改者、资料修改日、所属工厂、所属法人。
- 查表/字段最快：`tg dict r.t ima_file`（字段、类型、说明）、`tg dict r.t ima_file --who`（哪些程序在用）、`tg dict prog abmi100 --tables full`（这支程序用了哪些表）。

### 3.3 变量

| 前缀 | 作用域 | 说明 |
|---|---|---|
| `g_` | 全局 | 定义在 `$TOP/config/top.global` |
| `l_` | 局部 | 仅当前 FUNCTION |
| `p_` | 参数 | `FUNCTION x(p_cmd)` |
| `s_` | 屏幕 | 画面变量；多栏（数组）也用 `s_` |

### 3.4 函数

`<作业代码去模块后的四位>_<动作>`，例：`i100_q()`、`i100_u()`、`i100_i(p_cmd)`、`i100_a()`、`i100_show()`、`i100_fetch(p_flag)`、`i100_set_entry(p_cmd)`、`i100_set_no_entry(p_cmd)`。

手册列的常用动作：`_insert` / `_delete` / `_show` / `_fetch` / `_modify` / `_input` / `_query` / `_set_entry` / `_set_no_entry`。同一主程序里函数名不可重复（要加限定就往后接 `_bmo01`、`_b_fill` 这种）。

看到了 `_b`、`_bp`、`_bmp03` 这类名字，一般是「单身/子档」的维护分支函数（`b` = body/BOM 之类），跟主档那套 `_i/_u/_a` 对应。

---

## 4 程序骨架

一支标准的单档维护作业（手册 9.5 的形态）：

```4gl
# Prog. Version..: 1.0
# Pattern name...: abmi100
# Description....: 工程 B O M 模擬維護作業
# Author.........: xxx
# Modify.........: No.MOD-0001 2026/01/01 By xxx 说明
# Note...........:

GLOBALS "../../config/top.global"

DATABASE ds

DEFINE g_bmo        RECORD LIKE bma_file.*
DEFINE g_bmo_t      RECORD LIKE bma_file.*      # 异动前备份
DEFINE g_wc         STRING                      # 查询条件
DEFINE g_sql        STRING
DEFINE g_bmo_rowid  LIKE type_file.chr18
DEFINE g_forupd_sql STRING
DEFINE g_before_input_done BOOLEAN
DEFINE g_chr        LIKE type_file.chr1000      # 没有对应字段时的通用缓冲
DEFINE g_cnt        LIKE type_file.num5
DEFINE g_i          LIKE type_file.num5
DEFINE g_msg        STRING
DEFINE g_curs_index LIKE type_file.num5
DEFINE g_row_count  LIKE type_file.num5
DEFINE g_jump       LIKE type_file.num5
```

几个固定点：

- `GLOBALS "../../config/top.global"` —— 所有全局变量（`g_prog`、`g_user`、`g_lang`、`g_factory`…）都从这里进。
- `RECORD LIKE xxx_file.*` 是记录变量（表格一行）的标准写法；备份用 `_t` 后缀。
- `g_xxx_rowid LIKE type_file.chr18` 用于 `FOR UPDATE` / 定位当前行。
- 没有对应 DB 字段的中间变量，统一借 `type_file.chrN/numN`（实测有 `chr1`、`chr18`、`chr1000`、`num5`、`num20_6`、`dat`）。`type_file` 是**系统 schema `ds`** 里的通用类型档，各业务 schema 里只是指向 `DS.TYPE_FILE` 的同义字 —— 所以 `tg dict r.t type_file` 会显示 0 个字段，要查字段得看 `ds.type_file`。
- 函数排列惯例：`_q`（清除/初始化）→ `_curs`（游标）→ `_menu`（按键/菜单）→ `_a`（新增）→ `_u`（更改）→ `_i`（输入/查询条件）→ `_fetch` → `_show`（显示）→ `_x`/`_r`/`_b`（删除/报表/单身处理）→ `_set_entry`/`_set_no_entry`（栏位可输入性）→ `MAIN`。

`top.global` 里常用的全局变量：

| 变量 | 含义 |
|---|---|
| `g_prog` | 当前程序名（程序不可改它） |
| `g_user` | 当前用户代码（不可改） |
| `g_clas` | 群组 |
| `g_lang` | 语言别（画面文字/说明的语系） |
| `g_action_choice` | 当前动作（`cl_chk_act_auth` 检核权限前要设） |
| `g_qryparam` | 动态查询函数传参 |
| `g_plant` / `g_site` | 当前工厂别 / 营运据点（`g_plant_new` 是异动后的新值） |
| `g_bgjob` | 是否背景作业 |
| `g_today` | 系统日 |
| `g_win_style` | 画面样式表 |
| `g_idle_seconds` | 闲置超时 |

---

## 5 单档与双档

### 5.1 单档（一张主表）

三种子型：标准单档、假双档、单档多栏。

- 画面把「所有要维护的字段」按序放进画面、分页分群；栏位顺序 = 输入顺序 = Excel 导出顺序。
- 流程：输入条件 → `_fetch` 取一列 → 上一笔/下一笔浏览 → `_a`/`_u`/`_x` 异动 → 存盘后重新 `_fetch`。
- 「假双档」= 单头与单身其实是同一张表（用序号栏区分），代码形态像双档但只有一张表。
- 客制流程三种：标准单档客制、标准单档修改、单档多栏客制。

### 5.2 双档（单头 + 单身）

- 单头表 + 单身表，例：采购料件询价 `pmw_file`（单头）/ `pmx_file`（单身）。
- 画面上下分块：上半单头（字段逐栏）、下半单身（表格，一屏多行）。
- 功能流程：新增 / 更改 / 删除 / 复制 / 维护单身 / 打印 / 汇出 Excel / 查询 / 条件查询 / 条件储存。
- 读码要点：主档那套 `_i/_u/_a` 之外，单身逻辑集中在 `_b*` 系列函数（`_b`、`_bp`、`_b_fill`、`_bmp03`），单头异动经常连带更新单身与其它表（审核逻辑尤其注意）。
- 改「共用程序」前先 grep 谁在用：手册的例子是 `saxmt400` 被 `axmt400` / `axmt410` / `axmt810` 三支作业共用。

---

## 6 画面：4fd → per → 42f

1. `.4fd`：Genero Studio 画面源（XML，看 `ManagedForm`/`RecordField`），设计师改这个。
2. `.per`：由 4fd 导出的画面规格文本（文件头有 "File created from … by Genero Studio"），要做差异比对优先看它。
3. `.42f`：编译后的画面，运行期加载的是它 —— 改了 4fd 必须重新编译才会变。

多语言：画面文字与字段说明走 `p_perlang`（语言档），**简体与繁体都要维护**；`tg interface fields <表单> --at <环境>` 能看到字段、控件、标签（标签来自 `gae_file`，`--labels` 可选 `auto|local|live|off`）。

---

## 7 报表

### 7.1 CR 报表（Crystal Reports）

顺序：明确需求 → 做画面 → 写程序 → 编译与维护链接档 → 生成数据源 → 放到 CR 服务器 → CR 模板 → 报表格式设置（`p_zaw`）→ 运行。

常见错误：索引无效（报表取数走错索引/没建索引）、载入报表失败、CertID 验证失败、登入失败、报表印不出来。看报表问题时先在 `rep/` 找 `.out` 定义，再回 `4gl` 看取数 SQL。

### 7.2 自定义查询报表

另一条线：使用者自己定义查询条件与输出栏，套用报表模板执行（可外部呼叫）。查这类问题看 `qry/` 与报表参数表，不要按 CR 报表的思路找模板。

---

## 8 数据字典与建表（p_zta）

`p_zta` = 系统管理模块的「档案架构修正记录维护作业」，也是 GP 的表字典本体：

- 能做：查表结构、新增/修改表格/字段/索引、查看档案修改记录、产生数据库手册。
- 操作前先选数据库（会先跳选择画面）；要异动表结构，得选「资料库管理者」并输入密码（预设 `manager` / `sys`）。
- 「建立表格」会按画面栏位在模块目录下生成建表 `.sch`，再用批次在其它营运中心建表。

命令式建表（手册 7.3）：建 `.sh` → 建 `.sch` → 执行异动表结构 → 更新 `p_zta` 数据字典。`.sch` 里就是标准 DDL：

```sql
create table tc_hmm_file (
  ...
);
alter table tc_hmm_file add constraint tpc_hmm_pk primary key (...);
grant select, insert, update, delete on tc_hmm_file to tiptopgp;
```

TG 里查字典不用连 p_zta 画面：`tg dict db sync --env <环境>` 把表/字段/索引/约束/程序/程序用表/消息/画面文字同步成本地 SQLite（22主机 实测 68 MB、1m45s），之后 `tg dict r.t`、`tg dict prog`、`tg dict msg`、`tg dict r.q` 都是本地的，断网也能查。

---

## 9 权限与菜单

四层（手册 1.5）：

1. **人员**：用户基本数据、是否多工厂。
2. **用户数据库权限**：可用哪些工厂别。
3. **程序基本权限**（`p_base_act`）：程序是否注册、权限类别、个人特殊权限。
4. **群组执行权限**：单支或整组更新；另有画面组件权限。

菜单：`p_zm`（主菜单更新）—— 新作业写完不仅要能跑，还要挂进 `p_zm` 才在菜单上看得到。

---

## 10 调试

- **GDC Debugger**（图形）：客户端里下断点、看变量。
- **`r.d2`**（文字）：`r.d2 <程序>`，起 `fgldb` 后进 `(fgldb)` 提示符，`run` / `break` / `next` / `print` 都在这里。
- DVM 的「源名」是 **`<模块>_<作业>.4gl`**（`info sources` 能看到 `cbm_abmi600.4gl`），磁盘上是 `topcust/cbm/4gl/abmi600.4gl`。**下断点要用 DVM 源名**：`break cbm_abmi600.4gl:785`；只写 `abmi600.4gl` 会被 fgldb 拒（`No source file named …`）。
- TG 侧：`tg debug start <作业> --env <环境>`、`tg debug bp add …`、`tg debug exec …`、`tg session ls`，网页调试台走同一套 `/api/sessions/*`。

---

## 11 客制规范（改代码前必看）

- 新客制表里的自定义栏位按 `tc_` + 基名 + `ud` + 序号（`tc_aaud01`/`tc_aaud02`）；客制表名 `tc_<三码>_file`。
- 单档、单档多栏、双档单头/单身表默认要有：资料有效码、资料所有者、资料所有群、资料建立日期、资料修改者、资料修改日、所属工厂、所属法人。
- 字段说明的简体与繁体都要维护（`p_perlang`），否则切语系会看到空白或旧文字。
- 画面栏位顺序影响输入顺序与 Excel 导出顺序，加栏位不要随手插中间。
- 修改共用程序（`lib/cl_*`、`sub/s_*`、`sub/q_*`）前先 grep 有哪些作业共用，改完确认链接档指向正确模块。
- 按钮管控/审核逻辑经常连带更新其它表（应收应付、库存异动），改之前把下游表列出来。
- 作业说明书（`sdd`）常在源码改动后没同步，**以源码为准**；但字段语义、操作步骤仍以 sdd 最权威。

---

## 12 用 tg 读这套代码

| 想知道 | 命令 |
|---|---|
| 有哪些环境 / 据点 | `tg env list`、`tg site list --env <环境>` |
| 这支作业是干什么的 | `tg sdd <作业> --at <环境> --human`（先看作業目的、操作功能、欄位說明） |
| 程序属于哪个模块、用了哪些表 | `tg dict prog <作业> --at <环境> --tables full` |
| 表有哪些字段、谁在用 | `tg dict r.t <表> --at <环境>`、`--who` |
| 字段中文说明 | `tg dict r.t`（`r.t --kw 关键字` 反查字段） |
| 开窗（查询画面）定义与 SQL | `tg dict r.q <开窗代号> --at <环境>` |
| 直接看源码 | `tg file cat <模块>/4gl/<程序>.4gl --at <环境>`、`tg file head/tail --lines N` |
| 搜代码 | `tg file grep -rn --at <环境> --include '*.4gl' '<关键字>' <模块>/4gl` |
| 找文件 | `tg file find --at <环境> . -name '*<关键字>*' -maxdepth 3` |
| 函数/块结构 | `tg outline <程序>.4gl --at <环境> --human` |
| 画面字段与标签 | `tg interface fields <表单> --at <环境>` |
| 本地镜像（离线快查） | `tg file mirror pull --at <环境>`、`tg file mirror path` |
| 只看远端库数据 | `tg db sql "select …" --at <环境>`（只读） |

镜像里收四类：`*/4gl/*.4gl`、`*/42f/*`、`*.inc`、`*/sdd/*.sdd`（22主机 实测 1.8 万个 / 464 MB），读单文件不如直读服务器，反复 grep 或断网时用镜像。

---

## 13 与 T100 的差异（读码最容易踩混的几条）

| T100 | TOP GP |
|---|---|
| `$TOP/erp/<模块>/4gl`、`com/` | `$TOP/<模块>/4gl`；公共代码在顶层 `lib/`、`sub/`、`qry/` |
| 设计器 + 代码生成器 + `add-point` 注释点 | Genero Studio 编 `.4fd`，直接产 `.per`/`.42f`，没有代码生成器 |
| `.42s` 存画面文字 | 画面文字在 `gae_file`，多语言走 `p_perlang` |
| 企业编号 ENT → 账号 | 没有 ENT：一个库一个账号，schema 由据点 `azp03` 决定 |
| 表名 `xxxx_t`、字段 `imaa001` | 表 `xxx_file`、字段 `ima01`、客制表 `tc_xxx_file` |
| `.tzc`/`.tzs` 设计器包 | 没有这层打包格式，改源码就是改 `.4gl` |
