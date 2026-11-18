---
name: tg-erp-read
description: 正在阅读或需要阅读鼎捷TOP GP代码时自动触发本技能
---

# ERP 代码阅读

## 获取源码：在线现读 / 离线镜像

1. 区分环境。tg的用户主要是ERP顾问或者ERP开发，他们的电脑上可能配置了多个不同的客户环境。用户提问时可能带了环境信息，比如："帮我看一下XX公司的XX作业的代码"。如果足够明确，可以判断出是哪个客户环境的作业，那么就可以直接去对应的环境中获取源码；如果不够明确，则需要先询问用户，确认客户环境后再去获取源码。

查看环境列表的命令：
```bash
tg env list
```

2. 找到程序编号。作业编号是用户在客户端看到的功能入口，而程序编号是作业的实现代码。一个作业可能调用多个程序，而一个程序也可能被多个作业复用。作业不等于程序，一个程序也可能被多个作业复用。在处理问题时，我们要优先判断用户说的是作业还是程序。用户大概率讲的是作业，我们需要借助 `tg dict prog <程序编号>` 来找到对应的程序，才能确定去看哪一份源代码。

```bash
tg dict prog <程序编号> --env <环境>
tg dict prog --kw <业务词> --env <环境>
```

程序编号到手后，**先花十秒看一眼作業說明書**（`tg sdd <程序编号>`，见下面「先读作業說明書」一节）：操作功能、逐字段说明、会动哪些档，一次就能拿到，比直接读代码快。

3. 优先从本地环境读取代码，如果用户指定环境没有代码镜像可以从本地的其他环境镜像获取代码，代码总体差异不会太大，本地没有镜像，不要帮用户通过 `mirror` 命令去下载代码，时间会很长，彻底没有办法时才尝试从远程获取代码，优先提醒用户拉取环境代码镜像。

获取本地源码镜像目录的命令：
```bash
tg file mirror path --env <环境>
```

从远程SFTP读取源码的命令（GP 的树是 `<实例>/tiptop` 与 `<实例>/topcust` 并列，模块直接在第一层，**没有 T100 的 `erp/` 那层**）：

```bash
tg file ls --at <环境> <模块>/4gl
tg file find --at <环境> . -name '*<程序>*' -maxdepth 3
tg file grep -rn --at <环境> --include '*.4gl' '<关键字>' <模块>/4gl
tg file head -n 80 --at <环境> <模块>/4gl/<程序>.4gl
```

客制代码在另一棵树，前面加 `topcust/`（如 `topcust/cbm/4gl/abmi600.4gl`）；路径不带前缀就是 `$TOP`。

4.使用代码大纲工具`tg outline`可以辅助快速阅读代码（有镜像时按文件名在镜像里找，没有就直接读服务器）。读大纲前如果还没看过作業說明書，先 `tg sdd <程序编号> --at <环境> --human`（见下节），它会告诉你这份程序的作业目的、操作功能与字段用法，能省掉大量猜字段的时间。

```bash
tg outline abmi100.4gl --at <环境> --human
```

```text
FUNCTION inp_key  [2-7]
  INPUT BY NAME g_cust  [3-6]
    ON KEY (F1, Control-z)  [4-5]
```

## 先读作業說明書：`tg sdd`

每个模块的 `sdd/` 目录里放的是**作業說明書**（`.sdd`，繁体 Big5）。它按 `▼---44---` 分段：
系統名稱 / 程式名稱 / 執行指令 / 作業目的 / 操作功能（A 輸入、Q 查詢、U 更改…逐功能键）/
欄位說明（`●訂單單號 [oea01] (C10)(R)` 这种逐欄位用法）/ 資料處理（检核、连动、UPDATE 哪些档）/
單身欄位控制。

读一个陌生作业时，**先看说明书画出的业务流程与字段用法，再进代码**，比直接啃 4GL 快很多。

```bash
tg sdd axmt410 --at <环境>            # 默认打第一份（有 <作業>.sdd 就优先它）
tg sdd axmt410 --at <环境> --list     # 只列这份作业有几份说明书
tg sdd axmt410 --at <环境> --human    # 打原文（默认是 JSON 信封）
tg sdd axmt410 --dir <镜像目录>       # 只看本地镜像，连环境都不需要
```

- 一个作业号常对应几份：`axmt410` → `axmt410dis`（单据）、`axmt410icd`（资料维护）、`axmt410slk`（查询）。
  `--list` 先看清有哪几份，再按场景 `tg sdd axmt410dis --at <环境>` 精确取。
- 数据源与 `tg outline` 一致：有镜像读镜像（快），没镜像就直接连服务器在 `$TOP`/`$CUST` 的
  `*/sdd/` 下按前缀找。JSON 里 `source` 会写 `mirror` 还是 `server`，镜像里的 `path` 带
  `tiptop/` 或 `topcust/` 前缀。
- 原文是 Big5，命令会自动转成 UTF-8（JSON 的 `encoding` 字段标明原文编码），终端不会乱码。
- 有些是占位空文档（只有 100 多字节），别把它当"这段没逻辑"的证据 —— 以源码为准；
  说明书本身也可能滞后于代码，冲突时同样以源码为准。

## 获取和阅读程序画面

通过程序编号（同上，不是作业编号哦）在本地或远程找到同名的`.42f`文件（一份 xml 组件树）。GP 没有 T100 的`.42s`语言档，画面文字放在 `gae_file` 里：`tg interface` 默认先查本地字典库 `t_ui_text`，没有再直查远程 `gae_file`（`--labels auto|local|live|off` / `--lang zh_CN|zh_TW`）。借助它给出的组件树与字段表，可以帮助Agent快速理解画面结构。

合成画面的命令：
 ```bash
   tg interface fields bsft001_wf `
     --at <环境> `
     --page page_4 `
     --limit 20 `
     --brief
 ```

手上已有 `.42f` 文件时用 `--42f <文件>` 直接指定，不必连环境。

## 通过数据字典理解字段含义

在TOP GP的数据库中，所有的表和字段都是流水号，无法直接理解其含义。我们需要借助数据字典来获取其真实的中文含义。TOP GP中跟数据库相关的字段有很显而易见的特征，具体的编码规则请参考 [tg-erp-architecture/SKILL.md](../tg-erp-architecture/SKILL.md)。

例如下面一段代码中，`pmds007`、`pmdt036`、`apba010`、`apba103`、`apba104`、`apba105` 都是数据库中的字段编号，我们需要通过数据字典来查询其对应的中文含义，才能理解这段代码的逻辑。
```
INSERT INTO capp115_tmp (sel,docno,seq,pmds007,pmdt036,apba010,apba103,apba104,apba105)
VALUES (gr_qry[li_i].check,gr_qry[li_i].pmdsdocno,gr_qry[li_i].pmdtseq,gr_qry[li_i].pmds007,
        l_price,gr_qry[li_i].apba010,l_amt_ex,l_tax,l_amt_in)
```

从字典中查询表和字段的中文含义的命令：
```bash
tg dict r.t --brief "capp115_tmp,pmds_t,pmdt_t,apba_t" --at <环境>@<据点编号>
tg dict r.t apb_file --at <环境>@<据点编号>
tg dict r.t capp115_tmp --at <环境>@<据点编号>
```

> Tips: 需要配置环境的命令都可以通过`--at-list`命令查看可以使用哪些环境

## 系统分类码 / 应用分类码 / 参数编号：GP 没有这套

T100 的三套"编号字典"在 GP 库里**一张表都没有**（`gzca_t` 系、`gzaa/gzac/gzad/gzzz/oocq/oocql` 系、`gzsz_t/gzszl_t/gzsy_t/ooac_t` 系全是 `ORA-00942`），所以 TG 不再提供 `tg dict scc` / `acc` / `sysp` / `docp` 四条命令。

对应的 T100 概念，读 GP 代码时要换成这些线索：

- **没有分类码/应用分类码**：GP 的下拉选项多半写死在程序里，或来自该模块自己的小档（如 `*a_file` 的维护作业）。看到 `LIKE gzcbl_t`、`cl_set_combo_scc`、`s_desc_get_acc_desc` 这类写法时，说明这段是 T100 遗留逻辑，GP 侧没有对端。
- **没有"参数编号"**：GP 的参数就是各模块**参数档里的列** —— `aaa_file`（账套参数档）、`aaz_file`（总账会计系统参数档 128 列）、`apz_file`（应付账款）、`aza_file`（整体管理系统参数档 148 列）、`azm_file`（会计期间参数）… 表说明含"参数"的一共 77 张。想知道某一列是什么意思，直接查字段说明：

```bash
tg dict r.t aaz_file --fields --at <环境>@<ds>
```

- **没有单据性质绑定**：T100 的"单据别参数绑一堆单据性质"这层在 GP 不存在。

## 从系统消息解读报错和提示

TOP GP中很多弹窗、消息的内容是可以复用的，特征也比较明显，比如模块+流水的 `aap-00228`、`aap-00245`，也有很多纯数字的 `-3101`、`100`。在它们附近往往伴随着一些弹窗、提示相关的组件，不借助工具查询也是看不出来含义的。

报错在 GP 里只有固定的几种形态（按 `22主机` 镜像 7,316 个 `.4gl` 的统计），读代码时先认形态，再决定要不要去查消息：

| 形态 | 规模（文件 / 处） | 典型写法 |
|---|---|---|
| `CALL cl_err(msg, code, n)` | 6,109 / 170,658 | `CALL cl_err(g_bma.bma01,'abm-123',0)` |
| `WHENEVER ERROR CALL cl_err_msg_log` | 6,725 / 7,214 | 程序头一行 |
| `CALL cl_err3(action, table, keys…, sqlcode, …)` | 3,420 / 37,240 | `CALL cl_err3("ins","bma_file",g_bma.bma01,…,SQLCA.sqlcode,"","",1)` |
| `CALL s_errmsg(field, value, msg, code, n)` | 1,104 / 22,062 | `CALL s_errmsg('bma01',g_bma.bma01,'','abm-123',0)` |
| `CALL cl_err_msg(title, code, replace, sec)` | 653 / 1,982 | `CALL cl_err_msg(NULL,'asf-001',g_user||'\|'||'oea01',10)` |
| `IF SQLCA.sqlcode <> 0` | 251 / 1,324 | 自己判库错 |
| 原生 `ERROR` / `MESSAGE` | 2,714 / 3,169、3,383 / 17,191 | `ERROR ""`、`MESSAGE 'UPDATE O.K'` |

**① 主出口 `cl_err`** —— 第二个参数才是消息本体，一眼能分出「业务错」还是「系统错」：

```
CALL cl_err('','abm-123',0)                    # 模块-流水码：走 ze_file 消息档，按语系取原文
CALL cl_err(g_bma.bma01,'mfg2634',0)           # 同上，第一个参数是附加说明（业务值，会拼在原文前/后）
CALL cl_err('',-400,0)                         # Oracle 错误号：原文由 finderr 给（-263 就是资料被锁住）
CALL cl_err("OPEN i600_cl:",STATUS,1)          # STATUS：游标/连线这类系统态错误
```

第三个参数 `p_n` 控制停留：`0` 不停留、`1` 等 CR（GUI 下等价于弹窗确认）、`>1` 停几秒、`-1` 文本模式等同 `0`。

**② 兜底 `WHENEVER ERROR CALL cl_err_msg_log`** —— 几乎每个程序头一行；真跑飞时写错误日志（`g_need_err_log='Y'`、`g_bgjob='Y'`、cron 才写），`STATUS=-1109` 直接 `ERR_QUIT`，`LOG_ERRTIMES` 防爆。看到它不用找消息码，它不产生业务文案。

**③ 库动作失败 `cl_err3`** —— `INSERT/UPDATE/DELETE` 之后紧跟，比 `cl_err` 多带动作、表名与键值，日志能直接定位到哪一行数据：

```
CALL cl_err3("ins","bma_file",g_bma.bma01,g_bma.bma06,SQLCA.sqlcode,"","",1)
```

源码不在 4gl 树里（只有 `lib/42f/cl_err3.42f` 与 `lib/sdd/cl_err3.sdd`），读不到实现时按名字理解即可。

**④ 多栏位一起报 `s_errmsg` + `s_showmsg`** —— 先逐栏位攒，最后一次性弹「错误清单」表格：

```
CALL s_errmsg('bma01',g_bma.bma01,'','abm-123',0)   # 栏位名、当前值、附加说明、消息码、停留
CALL s_errmsg('bma02',g_bma.bma02,'','abm-456',0)
CALL s_showmsg()                                     # 把 g_err_msg[] 用 cl_show_array 列成表（lib-314 当标题）
```

`p_n=0` 时 `s_errmsg` 直接 return（等于这条不报）；`s_showmsg` 只在 `g_bgerr=TRUE` 时才弹。

**⑤ 直接弹窗 `cl_err_msg`** —— 跳过 `cl_err`，`|` 分隔替换 `%1..%n`：

```
CALL cl_err_msg(NULL,'asf-001',g_user||'|'||'oea01',10)
```

**⑥ 自己判库错** —— 新代码偏多，常配 `SQLCA.SQLERRD[2]` 与 `cl_sql_dup_value()`（唯一键冲突）。

**⑦ 原生 `ERROR` / `MESSAGE`** —— `ERROR ""` 多半是**用它把控制流从 `INPUT`/`MENU` 里弹出来**（真实原因在 `STATUS` 里，别当成"空错误"），`MESSAGE 'UPDATE O.K'` 是状态行提示，都不是给人看的错误文案。

> 两种不存在的写法，别照抄 T100：
> - `INITIALIZE g_errparam … LET g_errparam.code/popup = TRUE` —— GP 全库只有 `lib/4gl/cl_soapcli.4gl` 一处，还是被 `{}` 注释掉的；
> - `CALL cl_err()` 无参 —— GP 的 `cl_err` 固定三参 `(p_msg, err_code, p_n)`。

### 什么时候该调 `tg dict msg`

它在本地字典的 `t_msg`（源 `ze_file`）里按「消息代号 + 语系」查原文与处理建议。**只有下面这几种情况值得查**：

```bash
tg dict msg abm-123 --at <环境>                # 知道代号（cl_err/cl_err_msg/s_errmsg 的 code 参数、日志里的 aap-00228）
tg dict msg --text 锁定 --lang zh_CN --at <环境>  # 只记得大概意思，反查代号
tg dict msg -- -3101 --at <环境>                # 纯数字码：查的是自定义文案，Oracle 原生错误用 finderr 读
```

- **要查**：代码里写死的模块-流水码（`abm-123`、`mfg2634`）、接口/日志里冒出的 `aap-00228` 这类码、想确认某个码在简繁两种语系下的措辞；
- **不必查**：`-400`/`-239`/`-263`/`9001` 这类 Oracle 原生错误（`finderr` 就能解释，`tg dict msg -- -263` 只是看有没有被客制过文案）、纯 `STATUS`（值在 `cl_err` 调用处的上下文里）、`ERROR ""`/`MESSAGE 'UPDATE O.K'`（不是消息码）；
- 查不到时用可用语言提示（`可用语言: 2`）加 `--lang` 重试，别当成"没有这条消息"。

## 开窗和校验带值

TOP GP中的开窗查询组件是可以复用到不同的字段的，特征为 `q_xxx()`。它**不是设计器生成**的：开窗定义维护在 `p_qry` 作业的单头档 `gab_file`（代号、输入条件、JOIN、DISTINCT、强条件、排序、是否 HardCode、客制度）与单身档 `gac_file`（表、字段、控件、宽度、回传、OUTER、约束、串查）里，程序用 `cl_create_qry()` 按这些定义现拼 SQL 去查。

```
ON ACTION controlp

CASE
WHEN INFIELD(bmbb05)
                 CALL cl_init_qry_var()
                 LET g_qryparam.arg1 = g_ima941
                 LET g_qryparam.form ="q_agd"
                 CALL cl_create_qry() RETURNING g_bmba_1[l_ac2].bmbb05
                 DISPLAY BY NAME g_bmba_1[l_ac2].bmbb05
                 NEXT FIELD bmbb05
END CASE
```

> 开窗有实体代码，可以在公共的 `qry/4gl` 下看到生成的 q_*（T100 里叫 `com/qry`），但基本上不需要——用 `tg dict r.q` 直接查 SQL 更快；也要结合查询组件附近的参数看，因为会注入参数。


查询开窗对应的SQL等信息的命令：
```bash
tg dict r.q q_rwf01 --at <环境>@<据点编号>
tg dict r.q --kw rwf --at <环境>@<据点编号>
```

过栏位的判断也是可以复用的，特征是 `v_xx`。（这一层 GP 侧没有 `tg dict r.v`；`v_*` 封装的 SQL 直接在源码里看，或按它过滤的表/字段用 `tg dict r.t` 反查。）

```
AFTER FIELD wfapwf023

#add-point:AFTER FIELD wfapwf023 name="input.a.page3.wfapwf023"
IF NOT cl_null(g_sfakwf3_d[l_ac].wfapwf023) THEN 
    ...
    IF cl_chk_exist("v_wfalwf001_wf") THEN
      ...
    ELSE
    ...
END IF   
```

## 查询数据库

查询数据库也是学习和验证代码逻辑的重要手段，为了安全的只读查询，我们也封装了工具

选择环境，执行一句查询的示例
```
tg db sql --at <环境> "select count(*) from capp115_tmp"
tg db sql --at <环境> "select bmaa001,bmaastus from bmaa_t where bmaa001='FCPU010100003'"
```

默认从本机直连该企业的库；直连建不起来会自动回退服务器 sqlplus（不静默：信封里
`route` 写明这次实际走的哪条，`notes` 写明回退原因）。要钉死某条路用
`--route direct`（只直连）或 `--route server`（只走服务器）。

**两条路的日期渲染不一样**：直连是 `2026-10-28 00:00:00`，服务器 sqlplus 是
`28-OCT-26`。要和既有报表逐字对齐时加 `--route server`；只是自己读一眼逻辑，默认值就行。

一次要查好几条时用批次，别在 shell 里循环（每条都要重新建连）：
```
echo '[{"id":"cnt","sql":"select count(*) c from inaj_t"},
       {"id":"one","sql":"select 1 x from dual"}]' > q.json
tg db sql --at <环境> --batch q.json
```
批次只输出 JSON：成败看逐条 `items[].ok`，行数看 `items[].rows`（顶层
`totalRows/returned` 是语句条数）；单条失败不影响其余。

