---
name: tg-erp-read
description: 正在阅读或需要阅读鼎捷TOP GP代码时自动触发本技能
---

# TOP GP 代码阅读

> 4GL 语句、画面指令、报表与数据库接口的权威说明在 **Genero BDL 官方手册**（本机 Markdown）：
> 目录跑 `tg install bdl --dry-run` 问（打印 `{"dir": ...}`，不写盘；已装过就是 `skipped`），
> 没装先 `tg install bdl`，再到那个目录按章找（`01_genero-bdl-user-guide` … `19_copyrights-and-trademarks`）。
> 这是查"语言本身怎么规定"，不替代 `tg sdd` / `tg interface` 查某支作业的业务语义。

TOP GP 里表名、字段、画面文字全是流水号（`ima01`、`bma_file`、`gae_file`），
光看代码读不出业务含义。所以顺序是：**定环境 → 定程序 → 读作業說明書 → 看画面 → 读代码 → 查字典/消息/库验证**。

| 步骤 | 目的 | 命令 |
|---|---|---|
| 1 | 定环境 | `tg env list` / 任意命令加 `--at-list` |
| 2 | 作业 → 程序 | `tg dict prog <编号>` |
| 3 | 作业目的与字段用法 | `tg sdd <程序>` |
| 4 | 画面结构与字段表 | `tg interface fields <程序>` |
| 5 | 读源码 | `tg file …` / `tg outline …` |
| 6 | 字段/消息/开窗/库验证 | `tg dict r.t` / `tg dict msg` / `tg dict r.q` / `tg db sql` |

## 1. 先定环境

用户的电脑上常配了多个客户环境。提问里带了客户名（"看一下 XX 公司的 XX 作业"）就直接去对应环境；不明确**先问**，别猜。

```bash
tg env list                        # 有哪些环境
tg dict prog --at-list             # 本命令能用的 --at 取值、来源与示例（纯本地，不联网）
```

`--at` 两种写法：`<环境>`、`<环境>@<据点编号>`（后者决定查哪个 schema，账号仍是 `db.accounts[0]`）。
据点编号是字母开头（`COF`、`digiwin`），写纯数字会被拒 —— GP 没有企业编号（ENT）。

## 2. 作业 ≠ 程序

用户在客户端点的是**作业**，干活的是**程序**；一个作业可能调多个程序，一个程序也可能被多个作业复用。
先把编号落实：

```bash
tg dict prog abmi600 --env <环境>          # 作业名/模块/程序类别/执行指令/涉及哪些表
tg dict prog --kw 产品结构 --env <环境>     # 只记得中文名
tg dict prog abmi600 --links --env <环境>   # 这个程序链接到哪个模块（客制换版看这条）
```

拿到程序后先看它**是不是个壳**：如果文件很短、只有一句 `CALL t110(...)`，真逻辑在同模块的子程序里
（`s` + 模块三码 + 函数名 + 后缀），追法见 [5.1](#51-入口程序常常只是个壳)。

## 3. 先读作業說明書：`tg sdd`

每个模块的 `sdd/` 下是**作業說明書**（`.sdd`，繁体 Big5），按 `▼---44---` 分段：
系統名稱 / 程式名稱 / 執行指令 / 作業目的 / 操作功能（`A` 輸入、`Q` 查詢、`U` 更改…逐功能键）/
欄位說明（`●訂單單號 [oea01] (C10)(R)` 逐欄位用法）/ 資料處理（检核、连动、更新哪些档）/ 單身欄位控制。

**读陌生作业先看它**：作业目的、逐字段含义、会动哪些表，一次拿全，比啃 4GL 快一个数量级。

```bash
tg sdd axmt410 --at <环境>            # 默认打第一份（有 <作業>.sdd 就优先它）
tg sdd axmt410 --at <环境> --list     # 只列这份作业有几份说明书
tg sdd axmt410 --at <环境> --human    # 打原文（默认是 JSON 信封）
tg sdd axmt410 --dir <镜像目录>       # 只看本地镜像，连环境都不用
```

- 一个作业号常有几份：`axmt410` → `axmt410dis`（单据）、`axmt410icd`（资料维护）、`axmt410slk`（查询）。
  先 `--list` 看清，再按场景精确取。
- 数据源与 `tg outline` 一致：有镜像读镜像（快），没有就连服务器在 `$TOP`/`$CUST` 的 `*/sdd/` 下按前缀找；
  JSON 的 `source` 写 `mirror` 还是 `server`，镜像里的 `path` 带 `tiptop/` 或 `topcust/` 前缀。
- 原文是 Big5，命令自动转 UTF-8（JSON 的 `encoding` 标明原文编码），终端不会乱码。
- 有些是占位空文档（100 多字节），别当成"这段没逻辑"；说明书也可能滞后于代码 —— 冲突时**以源码为准**。

## 4. 读画面：`tg interface`

按程序编号（不是作业编号）找同名 `.42f`（一份 XML 组件树）。GP **没有** T100 的 `.42s` 语言档，
画面文字在 `gae_file` 里：`tg interface` 默认先查本地字典 `t_ui_text`，没有再直查远程 `gae_file`。

```bash
tg interface fields bsft001_wf --at <环境> --page page_4 --limit 20 --brief
tg interface pages abmi100 --at <环境>          # 这个程序有几页
tg interface fields bsft001_wf --42f <本地.42f>  # 手上有 .42f 就不必连环境
```

`--labels auto|local|live|off` 控制标签来源，`--lang zh_CN|zh_TW` 控制语系。
返回里 `labelSource` 会写清这次标签是从本地字典、远程 `gae_file` 还是离线拿的。

### 4.1 栏位的中文说明不在 4GL 里

在 4GL 源码里几乎看不到「主件料件」「规格」这种标题 —— 它们是 `.42f` 里**跟字段成对的 `Label` 元件**，
运行时才按语系从 `gae_file` 取译文。`lib/4gl/cl_set_comp_visible.4gl` 是最好的证据：它的用途是
「動態顯現/隱藏畫面上的元件」（`CALL cl_set_comp_visible("oea01,oea04", FALSE)`），遍历当前 form 的 DOM
按 `colName`/`name` 找到元件，隐藏字段时**必须把紧挨在前面的 Label 一起隐藏**，否则界面上就成了
「字段没了、标题还在」：

```genero
IF (ls_prev_tag.equals("Label")) AND
   (ls_prev_name NOT MATCHES "dummy*" OR cl_null(ls_prev_name)) THEN   # "+" 展开按钮会被跳过
    CALL lnode_prev.setAttribute("hidden",1)
END IF
```

所以读代码时：

- 程序里只有字段代码（`oea01`、`bmo06`），要中文说明就去画面/字典拿：
  ```bash
  tg interface fields abmi100 --at <环境> --limit 0 --brief    # 画面上的栏位标题 + labelReason
  tg dict r.t bma_file --fields --at <环境>@<据点编号>          # 按表查字段中文名
  ```
- `label` 为空不是工具坏了，看 `labelReason`：`token-not-found`（42f 里的 token 在 `gae_file` 查不到）、
  `label-not-associated`（这个元件没有配对的标题，常见于客制程序）—— 这种情况以 `.sdd` 与字段说明为准。
- 顺带一个坑：同一个函数里查 `gav_file`（画面输出字段格式设置档）算出的 `lc_cust_flag`**算完没有使用**，
  是历史遗留（原「必要栏位不可隐藏」逻辑被 `MOD-4C0124` 拿掉）；`gav_file` 在业务 schema 里还是同义词，
  `tg dict r.t gav_file` 会显示 0 字段。

## 5. 读代码：本地镜像优先

**优先读本地镜像**（快、可 grep、断网也能用）。用户环境没有镜像时，可以借**别的环境**的镜像
（同一套标准码，差异通常在客制）；实在没有才直连服务器读。**不要替用户拉镜像**（十几分钟起步），
先提醒他 `tg file mirror pull --at <环境>`。

```bash
tg file mirror path --at <环境>     # 镜像在哪、收了哪些类型（4gl/42f/inc/sdd）
```

直连读源码（GP 的树是 `<实例>/tiptop` 与 `<实例>/topcust` 并列，模块就在第一层，**没有 T100 的 `erp/` 那层**）：

```bash
tg file ls --at <环境> <模块>/4gl
tg file find --at <环境> . -name '*<程序>*' -maxdepth 3
tg file grep -rn --at <环境> --include '*.4gl' '<关键字>' <模块>/4gl
tg file head -n 80 --at <环境> <模块>/4gl/<程序>.4gl
tg file cat --at <环境> <模块>/4gl/<程序>.4gl          # 整个文件
```

客制代码在另一棵树，加 `topcust/` 前缀（`topcust/cbm/4gl/abmi600.4gl`）；不带前缀就是 `$TOP`。

再用 `tg outline` 看结构（有大纲就不用通读几千行）：

```bash
tg outline abmi100.4gl --at <环境> --human
```

```text
FUNCTION inp_key  [2-7]
  INPUT BY NAME g_cust  [3-6]
    ON KEY (F1, Control-z)  [4-5]
```

`--flat` 平铺（带 `depth`/`parent`）、`--depth N` 限层、`--brief` 去掉 `selectionRange`（人读树还会省掉行号）、
`--kind FUNCTION,MAIN` 只看某几类块。

### 5.1 入口程序常常只是个壳

读到的文件很短、只有一个 `CALL t110(...)` 之类，**别以为这段没逻辑** —— 它把活交给了同模块的子程序。
以 `apmt200_icd`（2,265 B）为例，MAIN 里就三步：

```
LET g_prog = 'apmt200_icd'
...
IF (NOT cl_setup("APM")) THEN EXIT PROGRAM END IF
CALL cl_used(g_prog,g_time,1) RETURNING g_time
CALL t110(g_argv1,g_argv2,g_argv3,g_argv4,g_argv5,'1')   #No.FUN-630010 #FUN-840012 #FUN-940083
CALL cl_used(g_prog,g_time,2) RETURNING g_time
```

真逻辑在 `apm/4gl/sapmt110_icd.4gl` 的 `FUNCTION t110(...)`（第 837 行）。规律：

- `CALL <函数名>` 的函数名是 **4 位**（字母 + 3 位数字），对应作业号后 4 位（`apmt200` → `t110`，数字不一定相同，**别按编号猜**）；
- 子程序文件名 = `s` + **模块三码** + 函数名 + **入口的后缀**：`apmt200_icd` → `sapmt110_icd`、`apmt730_icd` → `sapmt720_icd`、`aapt140` → `saapt110`；
- 带下划线的 `s_xxx`（`s_aapt110_gl`、`s_t110_w`）是 `sub`/`lib` 里的共享子程序，不是这一类。

**三步定位（实测有效）**：

```bash
# ① 看链接清单：找 s<模块><函数名>*，有多个时按后缀与入口对齐
tg dict prog apmt200_icd --links --at <环境> --limit 0
#   → apm sapmt110_icd / apm sapmt110_sub / apm sapmt720_sub …（entry=true 那条才是入口自己）

# ② 直接 grep 函数定义：跨模块也能找到，参数表对得上就是它
tg file grep -rn --at <环境> --include '*.4gl' 'FUNCTION t110' apm/4gl
#   → apm/4gl/sapmt110_icd.4gl:837

# ③ 目标可能很大（saapt110.4gl 有 2.6 MB），用大纲读，别 cat
tg outline apm/4gl/sapmt110_icd.4gl --at <环境> --human
```

- 同名函数常有几份（`sapmt110`、`sapmt110_icd`、`sapmt110_slk`、`sapmt110_sub` 都有 `FUNCTION t110`），
  按**入口后缀**挑（`_icd` 对 `_icd`），再核对参数个数（`sapmt110` 是 6 参、`saapt110` 是 5 参且第一参是 `p_aptype`）。
- `--links` 是全量清单，别被默认 20 条截断：加 `--limit 0`（或按 `tg dict spill show <文件>` 翻页）。
- 这类壳程序不算少：`22主机` 镜像里 1,050 个 <6 KB 的 `.4gl` 中有 **245 个**在 `CALL` 4 位函数。
- 反过来也成立：改一处逻辑前先看**谁在调它** —— `tg file grep -rn --at <环境> --include '*.4gl' 'CALL t110' .`。

## 6. 查字段含义：`tg dict r.t`

GP 的字段是「表码 + 两位流水」（`bma01`、`rvv39`），看不懂就去字典查。例如这段真实代码：

```
INSERT INTO bma_file VALUES (g_bma.*)
IF SQLCA.sqlcode THEN      #置入資料庫不成功
    CALL cl_err3("ins","bma_file",g_bma.bma01,g_bma.bma06,SQLCA.sqlcode,"","",1)
    CONTINUE WHILE
END IF
```

`bma_file` 是什么档、`bma01`/`bma06` 是什么、`cl_err3` 的键值为什么挑这两栏，只有字典能回答：

```bash
tg dict r.t bma_file --at <环境>@<据点编号>            # 表说明 + 模块 + 字段数
tg dict r.t bma_file --fields --at <环境>@<据点编号>    # 字段骨架：代号 + 中文名 + 类型
tg dict r.t --brief "bma_file,rvv_file" --at <环境>@<据点编号>   # 一次看多张表的表说明
tg dict r.t --kw 料件 --at <环境>@<据点编号>            # 只记得中文名
```

- `--brief`（表级）与 `--fields`（字段级）**只能二选一**。
- 逗号列表里**只要有一张表不存在，整条报 `表不存在: …`** —— 拼错名字时先单表查。
- 程序里 `CREATE TEMP TABLE` 出来的临时档（`curr_tmp`、`p112_tmp`）字典里没有，读它们的字段只能回源码。
- `--full` 会把完整字段/关系一起带出来，默认是摘要。
- 表名的命名规则、模块代码含义见 [tg-erp-architecture/SKILL.md](../tg-erp-architecture/SKILL.md)。

## 7. 读开窗：先分清 `q_*` 和 `cq_*`

开窗有两类，找法完全不同 —— **看到 `cq_*` 别去 `tg dict r.q` 查**，它不在字典里：

| 类型 | 定义在哪 | 怎么读 |
|---|---|---|
| `q_*` 标准开窗 | `gab_file` / `gac_file`（`p_qry` 作业维护），程序用 `cl_create_qry()` 现拼 | `tg dict r.q <代号>` 直接给定义与 SQL |
| `cq_*` 手写/客制开窗 | 纯 4GL 函数，主要在 `topcust/cqry/4gl/cq_*.4gl` | 读代码（`tg file` / `tg outline`） |

### 7.1 `q_*`：定义在字典里

它**不是设计器生成的**：单头档 `gab_file`（代号、输入条件、JOIN、DISTINCT、强条件、排序、是否 HardCode、客制度）
+ 单身档 `gac_file`（表、字段、控件、宽度、回传、OUTER、约束、串查），程序按这些定义现拼 SQL。

```
ON ACTION controlp
   CASE
     WHEN INFIELD(pmm01)
          CALL cl_init_qry_var()
          LET g_qryparam.form  = "q_pmm_sel"
          LET g_qryparam.state = 'c'
          LET g_qryparam.arg1  = g_rva.rva10
          LET g_qryparam.arg2  = g_rva.rva05
          CALL cl_create_qry() RETURNING g_qryparam.multiret
          DISPLAY g_qryparam.multiret TO pmm01
          NEXT FIELD pmm01
     WHEN INFIELD(pmn04)
          CALL q_sel_ima(TRUE,"q_ima03","","","","","","","",'') RETURNING g_qryparam.multiret
          DISPLAY g_qryparam.multiret TO pmn04
          NEXT FIELD pmn04
   END CASE
```

```bash
tg dict r.q q_rwf01 --at <环境>@<据点编号>       # 明细：定义 + 现拼出来的 SQL
tg dict r.q --kw rwf --at <环境>@<据点编号>      # 按代号找开窗
```

- 明细里给**两套 SQL**：一般查询（`gab02`）与开窗 CONSTRUCT 模式（`gab02` + `gab05`），别只看前者。
- `gab07=Y` 是 HardCode 开窗：SQL 写在程序里，字典不拼。
- `q_*` 的实体代码在公共的 `qry/4gl`（T100 里叫 `com/qry`），基本不用读 —— `tg dict r.q` 更快；
  但要看**调用处注入了哪些参数**（`g_qryparam.arg1/arg2`），SQL 里的条件可能来自那里。

### 7.2 `cq_*`：定义在代码里

`22主机` 镜像里 36 个 `cq_*.4gl`，其中 32 个在 `topcust/cqry/4gl/`，另几个散在模块自己的 `4gl`
（如 `tiptop/qry/4gl/cq_ime_11.4gl`、`topcust/cic/4gl/cq_cict042.4gl`）。它们是**手写的开窗函数**：

```
# topcust/cqry/4gl/cq_bmb01.4gl
FUNCTION cq_bmb01(pi_multi_sel,pi_need_cons,p_tc_sfd01,p_tc_sfd02,p_tc_sfd05)
   ...
   LET ms_cons_where = "1=1"                        # CONSTRUCT 条件攒在这里
   SELECT ze03 INTO li_count FROM ze_file WHERE ze01='qry-001' AND ze02=g_lang   # 每页笔数
   SELECT ze03 INTO li_page  FROM ze_file WHERE ze01='qry-002' AND ze02=g_lang   # 页数
   LET ls_sql = "SELECT ... FROM bmb_file WHERE bmb04 <= to_date('",g_today,"','yy/mm/dd') ..."
   PREPARE cq_bmb01_BP_pre1 FROM ls_sql
   ...
```

- 前两个参数是固定的：`pi_multi_sel`（是否多选）、`pi_need_cons`（要不要 CONSTRUCT 条件）；
  后面才是各作业自己的带入值。多选回传时调用处写 `CALL cq_xxx(...) RETURNING g_qryparam.multiret`。
- SQL 是**代码里手拼的**，条件、翻页、排序都可能是动态的，读 SQL 只能读源码。
- 找文件、找调用点（镜像里直接 grep，不必连服务器）：

```bash
tg file ls --at <环境> topcust/cqry/4gl
tg file grep -rn --at <环境> --include '*.4gl' 'FUNCTION cq_bmb01' topcust/cqry/4gl
tg file grep -rn --at <环境> --include '*.4gl' 'CALL cq_bmb01' .
tg outline topcust/cqry/4gl/cq_bmb01.4gl --at <环境> --human
```

- `tg dict r.q cq_bmb01` 会回 `NOT_FOUND 开窗定义 cq_bmb01 不存在` —— **这是正常的**，不代表这段没逻辑。

## 8. 读报错与提示：`tg dict msg`

报错在 GP 里只有固定的几种形态（`22主机` 镜像 7,316 个 `.4gl` 的统计），读代码时先认形态，再决定要不要查消息：

| 形态 | 规模（文件 / 处） | 典型写法 |
|---|---|---|
| `CALL cl_err(msg, code, n)` | 6,109 / 170,658 | `CALL cl_err(g_bma.bma01,'abm-123',0)` |
| `WHENEVER ERROR CALL cl_err_msg_log` | 6,725 / 7,214 | 程序头一行 |
| `CALL cl_err3(action, table, keys…, sqlcode, …)` | 3,420 / 37,240 | `CALL cl_err3("ins","bma_file",g_bma.bma01,…,SQLCA.sqlcode,"","",1)` |
| `CALL s_errmsg(field, value, msg, code, n)` | 1,104 / 22,062 | `CALL s_errmsg('bma01',g_bma.bma01,'','abm-123',0)` |
| `CALL cl_err_msg(title, code, replace, sec)` | 653 / 1,982 | `CALL cl_err_msg(NULL,'asf-001',g_user||'\|'||'oea01',10)` |
| `IF SQLCA.sqlcode <> 0` | 251 / 1,324 | 自己判库错 |
| 原生 `ERROR` / `MESSAGE` | 2,714 / 3,169、3,383 / 17,191 | `ERROR ""`、`MESSAGE 'UPDATE O.K'` |

**① 主出口 `cl_err`** —— 第二个参数才是消息本体，一眼分出「业务错」还是「系统错」：

```
CALL cl_err('','abm-123',0)                     # 模块-流水码：走 ze_file 消息档，按语系取原文
CALL cl_err(g_bma.bma01,'mfg2634',0)            # 同上，第一参是附加说明（业务值）
CALL cl_err('',-400,0)                          # Oracle 错误号：原文由 finderr 给（-263=资料被锁住）
CALL cl_err("OPEN i600_cl:",STATUS,1)           # STATUS：游标/连线这类系统态错误
```

第三个参数 `p_n` 控制停留：`0` 不停留、`1` 等 CR（GUI 下等价弹窗确认）、`>1` 停秒、`-1` 文本模式同 `0`。

**② 兜底 `WHENEVER ERROR CALL cl_err_msg_log`** —— 几乎每个程序头一行；真跑飞才写日志
（`g_need_err_log='Y'`、`g_bgjob='Y'`、cron），`STATUS=-1109` 直接 `ERR_QUIT`，`LOG_ERRTIMES` 防爆。
它不产生业务文案，看到不用找消息码。

**③ 库动作失败 `cl_err3`** —— 紧跟 `INSERT/UPDATE/DELETE`，多带动作、表名与键值，日志能定位到哪行数据。
源码不在 4gl 树（只有 `lib/42f/cl_err3.42f` 与 `lib/sdd/cl_err3.sdd`）。

**④ 多栏位一起报 `s_errmsg` + `s_showmsg`** —— 先逐栏位攒，最后一次性弹「错误清单」表格：

```
CALL s_errmsg('bma01',g_bma.bma01,'','abm-123',0)   # 栏位名、当前值、附加说明、消息码、停留
CALL s_showmsg()                                     # 把 g_err_msg[] 用 cl_show_array 列成表（lib-314 当标题）
```

`p_n=0` 时 `s_errmsg` 直接丢弃这条；`s_showmsg` 只在 `g_bgerr=TRUE` 时才弹。

**⑤ 直接弹窗 `cl_err_msg`** —— 跳过 `cl_err`，`|` 分隔替换 `%1..%n`：

```
CALL cl_err_msg(NULL,'asf-001',g_user||'|'||'oea01',10)
```

**⑥ 自己判库错** —— 新代码偏多，常配 `SQLCA.SQLERRD[2]` 与 `cl_sql_dup_value()`（唯一键冲突）。

**⑦ 原生 `ERROR` / `MESSAGE`** —— `ERROR ""` 多半是**把控制流从 `INPUT`/`MENU` 里弹出来**
（真实原因在 `STATUS`，别当成"空错误"）；`MESSAGE 'UPDATE O.K'` 是状态行提示。两者都不是给人看的错误文案。

> 两种不存在的写法，别照抄 T100：
> - `INITIALIZE g_errparam … LET g_errparam.code/popup = TRUE` —— GP 全库只有 `lib/4gl/cl_soapcli.4gl` 一处，还被 `{}` 注释掉；
> - `CALL cl_err()` 无参 —— GP 的 `cl_err` 固定三参 `(p_msg, err_code, p_n)`。

### 什么时候该调 `tg dict msg`

它在本地字典 `t_msg`（源 `ze_file`）里按「消息代号 + 语系」查原文与处理建议。**只有这几种情况值得查**：

```bash
tg dict msg abm-123 --at <环境>                    # 知道代号（cl_err/cl_err_msg/s_errmsg 的 code 参数、日志里的 aap-00228）
tg dict msg --text 锁定 --lang zh_CN --at <环境>    # 只记得意思，反查代号
tg dict msg --at <环境> -- -3101                   # 纯数字码：看有没有被客制过文案
```

- **要查**：代码里写死的模块-流水码（`abm-123`、`mfg2634`）、接口/日志里冒出的 `aap-00228`、要确认简繁两种语系下的措辞。
- **不必查**：`-400`/`-239`/`-263`/`9001` 这类 Oracle 原生错（`finderr` 就够）、纯 `STATUS`（值在调用处上下文里）、
  `ERROR ""`/`MESSAGE 'UPDATE O.K'`（不是消息码）。
- 查不到时先看「可用语言: N」，加 `--lang` 重试，别当成没有这条消息。
- 参数顺序：**标志写在 `--` 前面**，代号是唯一的位置参数（`tg dict msg --at <环境> -- -263`）。

## 9. 查库验证：`tg db sql`

读代码有疑问时，用只读查询验证最快（`tg db sql` 只走只读事务）：

```bash
tg db sql --at <环境> "select count(*) c from ima_file"
tg db sql --at <环境> "select ima01,ima02 from ima_file where rownum<=5"
```

默认直连该环境的库，直连建不起来会自动回退服务器 sqlplus —— **回退不静默**：信封里 `route` 写这次实际走哪条，
`notes` 写原因。钉死某条路用 `--route direct` / `--route server`。

**两条路的日期渲染不一样**：直连 `2026-10-28 00:00:00`，服务器 sqlplus `28-OCT-26`。
要和既有报表逐字对齐时用 `--route server`；只是自己看逻辑，默认即可。

一次查好几条用批次，别在 shell 里循环（每条都要重新建连）：

```bash
cat > q.json <<'EOF'
[{"id":"cnt","sql":"select count(*) c from ima_file"},
 {"id":"one","sql":"select 1 x from dual"}]
EOF
tg db sql --at <环境> --batch q.json
```

批次只输出 JSON：成败看逐条 `items[].ok`，行数看 `items[].rows`（顶层 `totalRows/returned` 是语句条数）；
单条失败不影响其余。

## 附：GP 里没有的 T100 概念

读老代码时最容易踩的几个空（对照 [tg-erp-architecture/SKILL.md](../tg-erp-architecture/SKILL.md)）：

- **企业编号（ENT）**：GP 一个环境一个库，账号固定 `db.accounts[0]`；`@` 右边写的是**据点编号**（只换 schema）。
- **`.42s` 语言档**：没有，画面文字在 `gae_file` / 本地 `t_ui_text`。
- **分类码 / 应用分类码 / 参数编号**：`gzca_t`、`gzaa/gzac/gzad/gzzz`、`gzsz_t/gzszl_t/gzsy_t/ooac_t` 一张都没有。
  下拉多半写死在程序里或来自模块自己的小档；「参数」就是各模块参数档里的列（`aaa_file`、`aaz_file`、
  `apz_file`、`aza_file`、`azm_file`…，表说明含"参数"的共 77 张）：

  ```bash
  tg dict r.t aaz_file --fields --at <环境>@<ds>
  ```

- **设计器 / `.tzc` / `.tzs` 包**：TG 不做这条线；开窗定义看 `gab_file`/`gac_file`（见第 7 节）。
- **`g_errparam` 错误参数**：见第 8 节，GP 没有这套。
