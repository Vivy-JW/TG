---
name: tt-erp-architecture
description: 需要处理任何和鼎捷T100ERP相关的问题，需要触发此技能，Agent必须要了解T100的基本架构。
---

# 鼎捷T100 ERP 架构

鼎捷T100是一款基于Genero BDL语言开发的大型ERP系统，其服务部署在Linux服务器上，可搭配Oracle、Kingbase等数据库。该系统采用跨平台客户端架构，服务器运行时环境和客户端同时维护一棵BOM树，双方通过一套名为TFP的协议进行TCP通信以实现BOM同步，这是其跨平台能力的基础。客户端名为GDC，由Qt开发，可在Windows、Linux和macOS上运行。

鼎捷T100的代码大部分由Genero BDL语言编写，而Genero BDL由4GL语言发展而来，语法与4GL类似。T100的大量代码由设计器生成。设计器的用户界面是鼎捷内部开发的一个C#客户端，主要用于代码编写和前端文件的拖拽操作。设计器本身不生成代码，修改代码后需通过服务器上的代码生成器程序生成代码。因此，在理解T100代码时，需要关注设计器生成的代码框架以及各类`add-point`注释点。

T100的开发工具不仅包括设计器和代码生成器，还包括各类封装好的开发工具，用于管理表字典、程序规格、业务词汇、可复用的校验组件以及可复用的开发组件。因此，在阅读代码时，除了阅读源码，还需结合这些工具获取充足的上下文信息。这些工具大多通过GUI界面操作，但我们已在`tt`这一CLI工具中将它们封装为CLI命令，以便辅助Agent调用。

## T100代码架构

> erp是业务模块(源码+编译产物+元数据)，com是公共代码(qry/wss/sub/lib/lng四大块)，www是
设计器与文档，utl/tool/apse是运行时与工具链;读码主战场是erp/*/4gl+com/*/4gl+*.inc，其余是产
物或运行支撑。

T100在服务器上的代码根目录为`$TOP`，模块业务代码放在`$ERP`

```
 $TOP                            ≈ 14 GB
   ├─ erp/         约105 个模块；源码 + 编译产物 + 元数据
   ├─ www/         设计器(clitool) + 文档(doc) + 组件/菜单/补丁
   ├─ com/         公共库/子程序/查询/接口/多语言（.4gl）
   ├─ utl/         运行时与第三方（JRE、7zip、opencc、graphviz、fbin）
   ├─ apse.bak/    APS 旧备份
   ├─ apse/        APS 排程（java/std/cust）
   ├─ tool/        构建/报表工具（crbuild、fop、report、wtk）
   ├─ res/         资源（img/std）
   ├─ sch/         调度定义（ds.sch、ds.42d、ext/）
   ├─ efgp/        流程模板图片（FlowTemplateLvl*.png）
   ├─ file/         空       运行期交换目录
   ├─ history/      空
   ├─ tmp/          空
   ├─ setup/        空
   ├─ log/          2 个 0 字节文件（NaNaApp.log / Tiptop.log）
   └─ t100_release* 6 个版本戳（
```

erp/:一个模块一套固定布局(以apm为例)

 ```
   erp/apm/
   ├─ 4gl/        Genero BDL 源码（未编译的源代码，主要看他）
   ├─ 4fd/        前端字段描述
   ├─ per/        界面源（per）
   ├─ 42f/        编译后的表单
   ├─ 42m/        编译后的模块
   ├─ 42r/        编译后的报表
   ├─ 42s/        顶层为空；编译字符串在 42s/zh_CN/*.42s
   ├─ zh_CN/      多语言目录，*.cpt（编译字符串的另一种产物）
   ├─ en_US/ es_ES/ ja_JP/ ko_KR/ …（每种语言一套 cpt）
   ├─ sch/ str/ mdl/ dzx/ 4ad/ 4pw/ 4rp/ 4tm/ 4st/ 4tb/ xcf/ …
   └─ 其它模块同构（约105 个）
 ```

com/：公共代码

 ```
   com/
   ├─ qry/  .4gl   ← 公共查询（最大）
   ├─ wss/  .4gl   ← Web Service / 接口
   ├─ sub/  .4gl   ← 公共子程序/元件
   ├─ lib/  .4gl   ← 公共库
   ├─ lng/  .4gl   ← 多语言
   ├─ mta/         ← 元数据/工具
   ├─ inc/  .inc   ← 4GL include
   ├─ cfg/   
   └─ clib/ clng/ cqry/ csub/ cwss/   ← 被客制过会转移到这里
 ```

其它

| 目录 | 内容 | 说明 |
|---|---|---|---|
| www/clitool 1.4 GB | 设计器前端（designer / layouteditor / xrdesigner / xrpreviewer） | 占 www 一半 |
| www/doc 1.3 GB | 文档 | 与运行无关 |
| www/components 362 MB | 前端组件库（chart/gantt/calendar/…） | |
| www/{erpmenu,b2bmenu,entlist,oa,t100app,tplinc} | 菜单、门户、模板 | 文件很少 |
| utl/3rd | JRE 1.7、7zip、opencc、graphviz、fgldeb | 运行时依赖 |
| utl/bin | fbin/{4gl,4fd,42m}、sqldebug、appqs | 工具程序 |
| apse/java + apse/std + apse/cust | APS 排程引擎（Java） | apse.bak 是旧备份 |
| tool/ | crbuild、fop、report、wtk | 打包/报表/工具链 |
| sch/ds.sch |  调度定义 | ds.42d 是其编译形态 |
| efgp/*.png | 流程模板图片（5 KB 级） | 体积很小 |

## 企业、据点、账套

T100 是适用于大型集团的 ERP 系统，其核心数据架构由企业（`ent`）、据点（`site`）和账套（`ld`）构成。在前端表现为：同一大型集团下的用户可登录至不同据点，据点通常代表一个工厂或法人公司，而账套则是可跨据点使用的账务主键。`ent` 一方面是各业务表的数据主键，另一方面，不同 `ent` 还与数据库 schema 相关联；据点与账套同样是许多单据的业务主键。理解这些数据架构，通常通过观察主键即可得到答案。

## 命名规范

### 模块命名原则

模块名称由三个英文字母组成（适用于 ERP 模块及共用 COM 模块等）。

**第一码**为标准识别码：A=标准 ERP，B=行业专用 ERP，C=客制 ERP，D=客制行业专用 ERP，E=客制自创 ERP。**第二、三码**采用系统英文缩写，若已被使用则第二码改以 X 或其他相关缩写替代。

格式：**AXX**（XX 为 ERP 模块英文缩写代码）

**范例**：AIM（料件主档及库存管理，IM 取自 Item Master / Inventory Management）、AXM（销售管理，因 ASM 已被系统基本资料管理使用而改之）。

| 模块 | 名称 |
|---|---|
| AAP | 应付账款管理系统 |
| ABC | 条码管理系统 |
| ABG | 预算管理系统 |
| ABM | 产品结构系统 |
| ABX | 保税管理系统 |
| ACO | 合同管理系统 |
| ACR | 客户关系管理 |
| ADB | 配销管理系统 |
| ADE | 日结管理系统 |
| ADZ | 程序生成设置管理系统 |
| AEC | 工艺管理模块 |
| AEP | 费用管理系统 |
| AFA | 固定资产管理模块 |
| AFC | 销售预测管理 |
| AFM | 融资投资管理系统 |
| AGC | 券管理 |
| AGL | 总账管理系统 |
| AIC | 企业间交易管理 |
| AIM | 料件管理系统 |
| AIN | 库存管理系统 |
| AIS | 发票税务管理系统 |
| AIT | 研发测试模块 |
| AIV | 电子发票方案 |
| ALO | 系统纪录数据模块 |
| ALR | 批次需求计划模块 |
| AMH | 招商管理系统 |
| AMM | 会员管理 |
| AMR | 工厂资源管理系统 |
| ANM | 资金管理 |
| AOO | 整体系统管理 |
| APC | POS接口 |
| APJ | 项目管理系统 |
| APM | 采购管理系统 |
| APR | 价格管理模块 |
| APS | 高端MRP |
| AQC | 质量控制系统 |
| ARM | RMA退品维修管理 |
| ARP | 移动化设备管理 |
| ART | 零售管理系统 |
| ASF | 生产管理模块 |
| ASL | 鞋服管理模块 |
| ASR | 重复性生产 |
| AST | 费用管理模块 |
| AWM | 仓储管理系统 |
| AWS | 集成管理模块 |
| AXC | 成本管理模块 |
| AXM | 销售管理系统 |
| AXR | 应收账款管理系统 |
| AZZ | 系统管理系统 |
| BWF | IC行业专用系统 |
| ENU | 客制模块 |

**共用模块（$COM）代码**：

| 代码 | 说明 | 英文名称 |
| :---: | :--- | :--- |
| lib | 共用程序 | Common Library |
| lng | 语言资料与提速档副程序 | Lingual sub function |
| sub | 一般副程序 | Common Sub function |
| qry | 查询副程序 | Common Query Function |
| wss | 整合与 Web Service 程序 | Web Service Subsystem |


### 程序命名原则

#### 主程序编号

格式：**SSSQ999_MM**（七码：三码英文 + 四码数字，均小写）

- SSS：模块代码
- Q：程序类别 — i=基本资料维护，m=主档维护，t=交易处理，s=参数设定，p=批次处理，q=查询，r=报表
- 999：三码流水号
- _MM：行业别专用代码（如 _ic=IC 设计业，_sc=鞋服饰，_ph=制药业；省略则通用）

**范例**：aimi100（料件基本资料维护）、cimi001（客制料件维护）、bphi100_ph（食品添加物登记证维护）

#### 子程序与子画面

- **子程序**：SSSQ999**_01**（一般,多支依序 `_02`、`_03`…）/ SSSQ999_MM**_01**（行业专用）
- **子画面**：SSSQ999_s**01** 或 SSSQ999_**01**_s**01**

**范例**：aimi100_01、cimi100_01；aimi100_s01、aimi100_01_s01

> **后缀是 `_01` 系列,不是 `_88`**：实测 `_01` 575 个、`_02` 258 个、`_03` 130 个,而 `_88` 全树只有 1 个(旧规格写法)。按 `_88` 去找子程序会找不到文件。

#### Library 文件

- **标准**：cl_xxxxxxx（共用程序，如 cl_err、cl_about），长度 1–17 字符
- **客制**：ccl_xxxxxxxxx（如 ccl_trim），长度 1–17 字符

> 客制共用程序暂不开放，lib 功能已完备。

#### 应用元件

- **标准**：s_XXXXXXX 或 s_SSSQ999_XXXXXX（如 s_transaction）
- **客制**：CS_XXXXXXXXX 或 cs_SSSQ999_XXXXXX（如 cs_date）

#### 报表相关文件

报表主程序命名同 2.1。报表元件格式（后缀同为 `_01` 系列,`_g88`/`_x88` 在本树不存在）：

| 类型 | 标准 | 行业专用 |
| :---: | :--- | :--- |
| 报表元件 | SSSQ999_g01（多件依序 `_g02`…） | SSSQ999_MM_g01 |
| 凭单列印元件 | SSSQ999_k01 | SSSQ999_MM_k01 |
| 查询列印元件 | SSSQ999_x01（多件依序 `_x02`…） | SSSQ999_MM_x01 |

**实测例**：`aapp350_g01`(退貨折讓單列印作業)、`aapr110_k01`(供應商貨款對帳憑單列印)、`aapq110_x01`(供應商對帳單明細查詢列印)。

- **报表结构档（.rdd）**：与元件同名，如 axmr500_g01
- **报表样板（.4rp）**：主报表 SSSQ999_g01（多样板加 _77），子报表加 _subrep66

#### 其他常见后缀

| 后缀 | 含义 | 例 | 本树数量 |
| :---: | :--- | :--- | ---: |
| `_wf` | IC半导体行业（`com/lib/cl_bpm_wf`、`q_*_wf`） | axmi125_wf、q_bxmi002_wf | 487 |
| `_rep` | 报表子程序片段，在 `com/inc/erp/<模块>/<母程序>_<NN>_rep.4gl` | aapt110_01_rep | 1,342 |
| `_01`…`_09` | 子程序（见上） | aapq110_01 | 575 / 258 / 130 |

**范例汇总**：

| 场景 | 主程序 | 报表元件 | 样板 |
| :--- | :--- | :--- | :--- |
| 标准报表 | axmr500 | axmr500_g01 | axmr500_g01 / axmr500_g01_subrep01 |
| 多样件 | axmr500 | axmr500_g02 | axmr500_g02 / axmr500_g02_subrep01 |
| 多样板 | axmr500 | axmr500_g01 | axmr500_g01_02 / axmr500_g01_02_subrep01 |
| 行业别 | axmr500_ph | bxmr500_ph_g01 | bxmr500_ph_g01 / bxmr500_ph_g01_subrep01 |
| 客制标准 | axmr500 | axmr500_g01 | axmr500_g01 / axmr500_g01_subrep01 |
| 新增客制 | cxmr501 | cxmr501_g01 | cxmr501_g01 / cxmr501_g01_subrep01 |

#### 查询元件

- **标准**：q_XXXXXXX（如 q_imaa001），长度 1–18 字符，以表格名命名
- **客制**：cq_XXXXXXXXX，长度 1–17 字符
- **行业专用**：q_XXXXXXXXX_mm（如 q_ooea001_ph），前缀长度 1–15 字符

> 避免与动态查询副程序名称冲突。

#### Web Service 程序

- **主程序**：wssp999（标准）/ cwssp999（客制）
- **子程序**：wssp999**_01**（标准）/ cwssp999**_01**（客制）（实测 `com/wss/4gl/wssp000_01.4gl`）

存放于 $COM/WSS，子程序不对外供一般 ERP 调用。

#### 共用参数文件（.inc）

- **一般用途**：SSSQ999_**01**.inc（放 4g1 目录）
- **跨模块**：top_XXXXXX.inc（放 $ERP/cfg，链接至 $COM/cfg）

#### 扩展名说明

| 类别 | 扩展名 | 说明 |
| :--- | :--- | :--- |
| 原始程序 | 4g1 / inc / 42m / 42r | 源码 / 共用参数 / 编译目标 / 可执行 |
| 画面 | 4fd / per / 42f | 设计档 / 对照档 / 编译档 |
| 报表 | rdd / 4rp | 结构档 / 样板档 |
| 资源 | str / 42s / sch / 4tm/4ad/4tb / 4pw | 翻译原始档 / 翻译编译档 / 参考内容 / Genero 设定 / 专案设定 |


### 函数命名原则

格式：**xxxxxxxxx_yyyy**（前 7 码为程序文件名，yyyy 为功能描述）

**范例**：azzi100_insert

**常用函数**：_insert、_delete、_show、fetch、_modify、_input、_query、set_entry、set_no_entry

> 同一主程序下函数名不可重复。


### 变量命名原则

| 类型 | 格式 | 说明 |
| :--- | :--- | :--- |
| 全局变量 | g_XXXXXXXXX | 定义于 $TOP/cfg/top_global.inc，如 g_gui_type、g_errno |
| 局部变量 | l_XXXXXXXXX | 仅限当前 Function 有效 |
| 传递参数 | p_XXXXXXXXX | 跨函数传递 |
| 屏幕变量 | s_detailN / s_browser | N 为流水号，异动可能影响代码生成 |


### 数据库表格命名原则

#### 表格名称

格式：**xxxx_t**（4 码小写 + _t），前两码为模块名，后两码流水号。如 imaa_t（料件主档）。

- **行业包辅助表**：xxxxmm_t（如 imaaic_t、imaasl_t）
- **客制表**：标准表不可删，新建客制表加 uc（如 imaauc_t）

#### 字段名称

格式：**xxxx999z**（表格编号 + 3 码流水号 + 阶层码）。如 imaa001（料件编号）、imaa001a（子阶）。

- **客制字段**：客制表用 apqquc001，标准表加 ud（如 apqqua001、imaaud001）
- **行业包辅助字段**：xxxxmm099（如 imaaic001）
- **自定义字段**：主/明/交易档加 ud，001–010 文字型（C003），011–020 数值型（N101），021–030 日期型（D002）

**固定字尾规范**：

| 用途 | 字尾 | 说明 |
| :--- | :--- | :--- |
| 建立 | crtid / crtdt / crtdp | 员工 / 日期 / 部门 |
| 拥有 | ownid / owndp | 员工 / 部门 |
| 修改 | modid / moddt | 员工 / 日期 |
| 确认 | cnfid / cnfdt | 员工 / 日期（adp 负责） |
| 过账 | pstd / pstdt | 员工 / 日期（adp 负责） |

**通用字段**：

| 字段 | 字尾 | 类型 | 备注 |
| :--- | :--- | :--- | :--- |
| 状态码 | stus | C001 | |
| 单号 | docno | C203 | |
| 单据日期 | docdt | D001 | T 类表单号后必跟日期 |
| 项次 | seq / seq1 / seq2 | N004 | 需逐层依赖 |
| 企业编号 | ent | N802 | |
| 法人 | comp | C813 | |
| 营运据点 | site | C813 | |
| 账别 | ld | C501 | 小写 |
| 组织对象 | unit | C007 | |
| 账务归属 | orga | C007 | |
| 核算组织 | legl | C007 | |
| 时间戳记 | stamp | D003 | 不可为 KEY |

**必要字段**：主/参数/基础资料档需 crtid、crtdp、crtdt、ownid、owndp、modid、moddt、stus；交易档另加 cnfid、cnfdt。

#### 索引与键值

- **索引**：xxxx_Qy（Q=n/u，y 流水号），如 imaa_n01、imaa_u01
  - 客制新表比照标准（imaauc_n01）；标准表加客制索引前缀 tic_（tic_imaa_n01）
  - 行业包辅助表：imaaic_n01
- **Primary Key**：xxxx_pk（r.t 自动命名，唯一不可改），如 imaa_pk
  - 客制：imaauc_pk；行业包：imaaic_pk
- **Footing Data**：xxxxxx_fdy（r.t 自动命名，仅供 r.a 及产生器参考，不实际建库），如 imaa_fd
  - 客制：imaauc_fd；行业包：imaaic_fd
