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
tt dict prog <程序编号> --evn <环境>
tt dict prog --kw <业务词> --env <环境>
```

3. 优先从本地环境读取代码，如果用户指定环境没有代码镜像可以从本地的其他环境镜像获取代码，代码总体差异不会太大，本地没有镜像，不要帮用户通过 `mirror` 命令去下载代码，时间会很长，彻底没有办法时才尝试从远程获取代码，优先提醒用户拉取环境代码镜像。

获取本地源码镜像目录的命令：
```bash
tg file mirror path --env <环境>
```

从远程SFTP读取源码的命令：
```bash
tg file ls --env <环境> erp/<模块>/4gl
tg file find --env <环境> erp -name '*<程序>*'
tg file grep -rn --env <环境> --include '*.4gl' '<关键字>' erp/<模块>/4gl
tg file head -n 80 --env <环境> erp/<模块>/4gl/<程序>.4gl
```

4.使用代码大纲工具`tg outline`可以辅助快速阅读代码

```bash
tg outline testdata/fgl-fixtures/input-on-key.4gl --human
```

```text
FUNCTION inp_key  [2-7]
  INPUT BY NAME g_cust  [3-6]
    ON KEY (F1, Control-z)  [4-5]
```

## 获取和阅读程序画面

通过程序编号（同上，不是作业编号哦）在本地或远程中找到对应同样编号的`.42f`文件（一份xml组件树）和`.42s`文件（语言档），然后借助`tt interface`可以通过这两份文件合成组件树，借助其提供的一些列API，可以帮助Agent快速理解画面结构

合成画面的命令：
 ```bash
   tg interface fields bsft001_wf `
     --42f "C:\Users\18526\Desktop\bsft001_wf.42f" `
     --42s "C:\Users\18526\Desktop\bsft001_wf.42s" `
     --page page_4 `
     --limit 20 `
     --brief
 ```

## 通过数据字典理解字段含义

在TOP GP的数据库中，所有的表和字段都是流水号，无法直接理解其含义。我们需要借助数据字典来获取其真实的中文含义。TOP GP中跟数据库相关的字段有很显而易见的特征，具体的编码规则请参考 [tg-erp-architecture/SKILL.md](../tt-erp-architecture/SKILL.md)。

例如下面一段代码中，`pmds007`、`pmdt036`、`apba010`、`apba103`、`apba104`、`apba105` 都是数据库中的字段编号，我们需要通过数据字典来查询其对应的中文含义，才能理解这段代码的逻辑。
```
INSERT INTO capp115_tmp (sel,docno,seq,pmds007,pmdt036,apba010,apba103,apba104,apba105)
VALUES (gr_qry[li_i].check,gr_qry[li_i].pmdsdocno,gr_qry[li_i].pmdtseq,gr_qry[li_i].pmds007,
        l_price,gr_qry[li_i].apba010,l_amt_ex,l_tax,l_amt_in)
```

从字典中查询表和字段的中文含义的命令：
```bash
tg dict r.t --brief "capp115_tmp,pmds_t,pmdt_t,apba_t" --at <环境>@<企业>
tg dict r.t apba_t --at <环境>@<企业>
tg dict r.t capp115_tmp --at <环境>@<企业>
```

> Tips: 需要配置环境的命令都可以通过`--at-list`命令查看可以使用哪些环境

## 系统分类码 / 应用分类码 / 参数编号：GP 没有这套

T100 的三套"编号字典"在 GP 库里**一张表都没有**（`gzca_t` 系、`gzaa/gzac/gzad/gzzz/oocq/oocql` 系、`gzsz_t/gzszl_t/gzsy_t/ooac_t` 系全是 `ORA-00942`），所以 TG 不再提供 `tt dict scc` / `acc` / `sysp` / `docp` 四条命令。

对应的 T100 概念，读 GP 代码时要换成这些线索：

- **没有分类码/应用分类码**：GP 的下拉选项多半写死在程序里，或来自该模块自己的小档（如 `*a_file` 的维护作业）。看到 `LIKE gzcbl_t`、`cl_set_combo_scc`、`s_desc_get_acc_desc` 这类写法时，说明这段是 T100 遗留逻辑，GP 侧没有对端。
- **没有"参数编号"**：GP 的参数就是各模块**参数档里的列** —— `aaa_file`（账套参数档）、`aaz_file`（总账会计系统参数档 128 列）、`apz_file`（应付账款）、`aza_file`（整体管理系统参数档 148 列）、`azm_file`（会计期间参数）… 表说明含"参数"的一共 77 张。想知道某一列是什么意思，直接查字段说明：

```bash
tt dict r.t aaz_file --fields --at <环境>@<ds>
```

- **没有单据性质绑定**：T100 的"单据别参数绑一堆单据性质"这层在 GP 不存在。

## 从系统消息解读报错和提示

T100中很多弹窗、消息的内容是可以复用的，特征也比较明显，比如模块+流水的 `aap-00228`、`aap-00245`，也有很多纯数字的 `-3101`、`100`。在它们附近往往伴随着一些弹窗、提示相关的组件，不借助工具查询也是看不出来含义的。

```
INITIALIZE g_errparam TO NULL
LET g_errparam.code = 'asf-00150'
LET g_errparam.popup = TRUE
CALL cl_err()
```

查询系统消息、错误消息的命令：

```bash
tt dict msg asf-00150 --at <环境>@<企业>
tt dict msg --at <环境>@<企业> -- -3101
```

## 开窗和校验带值

T100中的开窗查询组件是可以复用到不同的字段的，特征为 `q_xxx()`，它是由设计器生成的。

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

> 开窗是由实体代码的，可以在 `com/qry` 看到生成的代码，但基本上不需要，因为通过工具就可以查到SQL，也要结合查询组件附近的参数看，因为会注入参数。


查询开窗对应的SQL等信息的命令：
```bash
tt dict r.q q_sfajwfdocno_01_wf --at <环境>@<企业>
tt dict r.q --kw sfajwfdocno --at <环境>@<企业>
```

过栏位的判断也是可以复用的，特征是 `v_xx`。（这一层 GP 侧没有 `tt dict r.v`；`v_*` 封装的 SQL 直接在源码里看，或按它过滤的表/字段用 `tt dict r.t` 反查。）

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
tg db sql --at <环境>@<企业> "select count(*) from capp115_tmp"
tg db sql --at <环境>@<企业> "select bmaa001,bmaastus from bmaa_t where bmaa001='FCPU010100003'"
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
tt db sql --at <环境>@<企业> --batch q.json
```
批次只输出 JSON：成败看逐条 `items[].ok`，行数看 `items[].rows`（顶层
`totalRows/returned` 是语句条数）；单条失败不影响其余。

