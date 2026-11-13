# TG 范围界定：底座 / Genero 层 / T100 领域层

> 状态：**草案**（2026-11-09），等拍板再动代码。
> 配套：`.agents/plan/handoff.md`（跨会话状态）、`internal/README.md`（包职责地图）。
> 本文件只回答一个问题：**TT 0.5.2 的哪一块在 GP 世界里还有用，哪一块没有。**

## 0. 一句话结论

基线 7.45 万行 Go 分成六块：

| 块 | 规模（非测试行 / 包数） | 对 GP 的意义 | 动作 |
|---|---|---|---|
| **底座** | ≈15.0k / 18 | 全都要：SSH、配置、输出、Web 外壳、更新、安装 | 留，只改名（改名推迟到阶段 3） |
| **Genero 层** | ≈16.5k / 7 | GP 的主战场：4GL 大纲、`.42f` 画面、fgldb 调试 | 留，**拔掉夹在里面的 T100 假设** |
| **GP 字典层** | ≈10.7k / 2（`internal/dict` 4587 + `cli/dict` 6108） | GP 也需要字典（表/字段/索引/程序/开窗/消息） | **改造**：留机制，把 T100 的 24 张表换成 GP 的 `zta/gat/gaq/gaz/gao/gal`（2026-11-09 拍板） |
| **T100 领域层** | ≈26.1k / 8 + 两张资产树 | 用不上 | 最终删（并行 → 替换 → 剥离，删除放最后） |
| **GP 领域层** | ≈2.9k / 2（`internal/gp` + `cli/gp`） | 就是 GP 版本身 | 补完（见 §7） |
| **半通用** | ≈3.3k / 2（`cli/db`、`cli/file`） | 机制通用、路径与白名单是 T100 的 | 改造 |

三条不要混的前提：

1. **"层"有两套编号**。`internal/README.md` 里的 L0–L7 是**依赖深度**（谁 import 谁，零环），本文件的"底座 / Genero / T100"是**产品归属**。同一个包可以是"L0 但属于 T100 领域"（`entdir`）。
2. **删除放最后**。TT-src 只是只读副本、没有 git 历史，删掉的代码在本仓库里就回不来了；而现在 `go test ./...` 还有 5 个包是红的，删之前得先能测。
3. **"删"和"改"要分开 commit**。一次 commit 里既挪文件又删模块，事后没法看。
4. **"T100 的表"不等于"T100 的机制"**。`internal/dict` 的 `db.Source`（表/字段/键/索引/程序/作业/校验/规格/分类码/开窗/消息/参数 25 个查询方法）是**通用字典语义**，换一套表就是另一套 ERP 的字典；所以它进"改造"而不是"删"。

## 1. 判定口径：一个包值不值得留，只问三句

1. **GP 用得上吗？** 用不上就是 T100 领域层——不管它写得多好。
2. **它是 Genero 的规范，还是 T100 的约定？** 前者（4GL 语法、`.42f` 格式、fgldb 提示符）GP 一定用；后者（企业 ENT、`.tzc`）GP 一定不用。混着的那几个包就是 §5 的手术；`zta_file` 之类的**表**则进 §4.D 的换表，不删包。
3. **谁 import 它？** 光看名字会骗人：`dict/live` 名字在 T100 字典包下，实际是"直连 ERP 库执行一条只读 SQL"，GP 的 `tt dict prog --links` 与 `tt dict msg` 正在用它（`internal/cli/dict/prog.go`、`internal/dict/live`）。

## 2. 底座（留，只改名）

| 包 | 行数 | 为什么 GP 也要 |
|---|---|---|
| `cli`（根 9 文件） | 2961 | 根命令 + 全局开关 + env/config/cache/serve/install/update/version，与 ERP 无关 |
| `config` | 2209 | 唯一配置读写层；GP 环境也住在同一份 `config.json` 的 `hosts.sshs`（`Product == "gp"`） |
| `host` | 2261 | 唯一 SSH/SFTP/PTY 能力层；GP 镜像、字典同步、调试会话全靠它 |
| `web` | 2042 | 统一本地服务：SPA、设置页后端、路由表。**只有它需要改**：路由表要能拆挂（见 §5.4） |
| `update` | 1567 | 自身更新 + 技能树一致性，与 ERP 无关 |
| `skill` | 871 | 技能安装/清单，与 ERP 无关（但**分发内容**要换，见 §7.5） |
| `output` | 473 | 唯一输出出口（JSON/CSV/表格） |
| `erpdb` | 447 | Oracle / 金仓直连，GP 是 Oracle，长期要 |
| `winproc` | 389 | 起脱离进程、判活、杀 |
| `safesql` | 359 | 只读 SQL 白名单。GP 的 `tt dict` 查询同样走它 |
| `pathinstall` | 263 | 用户 PATH 增删 |
| `envmove` | 243 | 环境改名计划 / 回滚（GP 环境改名也在用） |
| `dbconfig` | 240 | 连接字段类型，纯类型 |
| `testenv` / `testkit` | 182 / 180 | 测试环境与跨包测试辅助 |
| `cli/common` | 132 | 命令组共享上下文（叶子包，防成环） |
| `sshtun` | 115 | SSH 端口转发（客户端直连库走隧道） |
| `session` | 71 | 会话序号 / 世代号，71 行纯规则 |

外带：`installer/wix`（MSI）、`tools/`、`build_*.bat`、`Makefile`、`web/` 前端外壳（`Shell/Sidebar/PanelKit/toast` 等）。

**底座里唯一有 GP 味道的改造点**是 `web` 的路由表——现在它硬写死三条线：

```go
m.Handle("/debug/api/", http.StripPrefix("/debug", s.opt.Debug))  // T100 调试
m.Handle("/api/", s.opt.Dict)                                     // T100 字典
m.Handle("/api/gp/", s.opt.GP)                                    // GP
```

阶段 3 删掉前两条之后，`web` 不该再编译失败，所以它们是"可拆的挂件"而不是"底座的一部分"。

## 3. Genero 层（留，但要拔 T100 假设）

| 包 | 行数 | 内容 | 备注 |
|---|---|---|---|
| `debug` | 13265 | fgldb 调试子系统的**内核**：会话/断点/停站/执行日志/源码镜像 + HTTP & WS 服务 | 机制是 Genero 的，**接线是 T100 的**（见 §5.1） |
| `cli/interfacecmd` | 1001 | `tg interface`：`.42f` + `gae_file` → 画面 | GP 的编译表单同族，但命名/标签约定要核 |
| `cli/outline` | 610 | `tt outline`：4GL → 块树 | 纯语法，GP 直接可用 |
| `form42f` | 630 | `.42f` → 字段表 + 布局 | 纯函数，零 IO |
| `debugclient` | 581 | 调试面 REST 的唯一客户端 | 跟着 §5.1 的结论走 |
| `screen` | 292 | `.42f` 布局 + `Labels` 接口 → 前端画面模型 | 纯函数；标签由调用方给（`gae_file`） |
| ~~`str42s`~~ | ~~141~~ | ~~`.42s` → 中文标签~~ | **已于 2026-11-12 删除**：TG 不用 `.42s`，画面文字改查 `gae_file`（`t_ui_text`） |
| `dev/fgl` | 2242 | 4GL 源码分块与语义 | 已留下（2026-11-11 删 `dev/` 其余包时保留），`tt outline` 与 GP 的 4gl 共用 |

两条判断依据：

- **`.42f` / `.42s` 是 Genero 编译产物，不是 T100 私有格式。** GP 每个模块下有 `42f/`，T100 有 `42f/`，解析器是同一套。这条决定了 `form42f`/`screen`/`cli/interfacecmd` 属于语言层；`str42s` 已于 2026-11-12 删除（TG 不用 `.42s`）。
- **fgldb 是 Genero 调试器**，提示符 `(fgldb) ` 是 Genero 的，不是鼎捷的（`internal/gp/session.go:27` 的 `const fgldbPrompt`）。所以"调试"这件事本身属于 Genero 层。

## 4. T100 领域层（最终删）

### 4.A 纯领域：删了不影响 GP

| 包 / 目录 | 行数 | 是什么 |
|---|---|---|
| `cli/debug` | 4547 | `tt debug`：T100 作业调试器的命令行面（停站、断点、接口日志重放） |
| ~~`dev`（`fence`/`model`/`pkgfile`/`split`/`store`/`synth`/`verify`/`tzs`/`tapfile`/`tglfile`）~~ | 12895 | `.tzc`/`.tzs` 设计器包的读写、围栏、合成、闸门 —— **已于 2026-11-11 删除**（只留 `fgl`、`testutil`） |
| ~~`cli/dev`（`tzc`/`tzs`/`common`）~~ | 5015 | `tt dev tzc` / `tt dev tzs` 命令面 —— **已于 2026-11-11 删除**，`tt dev` 组不复存在 |
| `designer` | 327 | 从设计器 `sites.xml` 导入环境站点 |
| `cli/session` | 494 | `tt session`：ENT 会话的建/列/看/改/收 |
| `cli/at` | 788 | `--at <环境>[@<企业>]` 文法与作用域表 |
| `entdir` | 157 | 企业(ENT) → 账号(`gzou_t`) 的映射、快照、新鲜期 |
| `target` | 1865 | 会话寻址：选择器文法、落位、契约信封 |

### 4.B 资产树（不是 Go 代码，但占比最大）

| 目录 | 规模 | 是什么 |
|---|---|---|
| ~~`designer-src/`~~ | 1188 文件 / 5.7 MB | T100 Designer 的 C# 解决方案（`T100Designer.sln`）—— **已于 2026-11-12 删除**（zip 存在仓库外） |
| ~~`engine/`~~ | 76 文件 / 11.7 MB | `.tzs` 引擎（C#，`TzsRepacker.cs` 等）+ 规格 `SPEC.md` —— **已于 2026-11-12 删除**（含 `designer/` 28 个厂商 DLL） |
| `testdata/` | 62 文件 / 0.06 MB | 只剩 `fgl-fixtures`（Go 侧大纲夹具）；`.tzs` 语料与 `tzs-mini` 2026-11-12 已删 |
| `skills/` | 3 个技能 | 已换 GP 版（`tg-erp-architecture` / `-debug` / `-read`）；`tt-dev-tzc` 已删 |

`designer-src/` 与 `engine/` 曾是**最大的两块死重量**（GP 侧一行都用不上）。2026-11-12 已删，删前打了 zip 存仓库外：`D:\Digiwin\tt_work\archive\TG-t100-designer-assets-2026-11-12.zip`。

### 4.C 领域耦合的共享件：要拆，不是删

`target`（寻址文法）、`cli/at`（`@企业`）、`debug`（被 GP 需要）——这三个是 §5 的主角。`dict` 不走"拆"，它整体进 4.D 的改造。

### 4.D 字典层：留机制，换表（2026-11-09 拍板：`internal/dict` 不删；2026-11-13 已落地）

**为什么留**：`internal/dict/db.Source`（`internal/dict/db/source.go`）是**通用字典语义**——表/字段/键/索引/程序/程序用表/程序链接/开窗/消息/画面文字，本地 SQLite 与远程直查实现同一组方法。换一套表，它就是另一套 ERP 的字典。接口随命令面收口从 30 个方法瘦到 **20 个**（`r.v`/`desc`/`scc`/`acc`/`sysp`/`docp` 六条命令已删，对应方法一并移除）。

**GP 的投影已经在本地库里**（`tg dict db sync`，实测 1 分 48 秒 / 68.7 MB）：`t_table`（2,962）、`t_column`（60,870）、`t_index`（3,318）、`t_constraint`（18,301）、`t_module`（121）、`t_prog`（6,465）、`t_prog_table`（118,420）、`t_prog_link`（17,810）、`t_msg`（42,031）、`t_ui_text`（427,553），另有 `meta`。

**换表清单（逐方法的接线表见 [gp-dict-wiring.md](gp-dict-wiring.md)）**：

| 项 | T100 现在 | GP 改成 | 实测 |
|---|---|---|---|
| 表档 / 字段档 | `dzea_t` / `dzeb_t` | `zta_file`+`gat_file` / `all_tab_columns`+`gaq_file` → `t_table` / `t_column`（含 `zta07` 种类：61 张只登记不建列的同义词） | 2,962 张表 / 60,870 字段 |
| 键 / 索引 | `dzed_t` / `dzec_t` | `all_constraints` / `all_indexes` → `t_constraint` / `t_index`；**GP 没有外键**，`RefTable/RefField` 从 `gau_file` 补 | P 2,790 / U 1 / R 0 |
| 程序 / 用表 / 链接 | `gzza_t` / `gzdg_t` / `gzou_t` | `gaz_file`+`zz_file`+`gal_file` → `t_prog`；`zr_file` → `t_prog_table`（带 S/I/U/D）；`gal_file` → `t_prog_link`（p_link 权威）；"作业登记"GP 没有 | 6,465 支 / 118,420 行 / 17,810 行 |
| 开窗 | `dzca_t`/`dzcb_t`/`dzcc_t` | **有**：`gab_file`（单头 + WHERE/DISTINCT/JOIN）+ `gac_file`（栏位）+ `gae_file`/`gaq_file`（标题与显示名）→ `tg dict r.q`，明细里按 `cl_create_qry` 的规则现拼 SQL（5,056 个开窗） | 5,056 / 15,327 |
| 消息 | `gzze_t` | `ze_file` → `t_msg` | 42,031 行 |
| 画面文字 | （T100 用 `.42s`） | `gae_file` → `t_ui_text`（`tg interface` 的标签源，替代 42s） | 427,553 行 |
| 校验 / 规格 | `dzcd_t`…/`dzep_t` | GP 无对应表 —— 命令已删，不留空壳 | — |
| 分类码 / ACC / 参数 / 单据性质 | `gzca_t`/`gzaa_t`/`gzsz_t`/`gzsy_t` | GP 无（实测全 `ORA-00942`）—— 命令已删 | 0 张表 |

**"GP 的 ENT"= 营运据点**：`azp_file.azp01` 是据点编号、`azp03` 是资料库编号（= `zta_file.zta02` = Oracle schema）、`azp053='Y'` 表示该据点有实际资料库。解析链 `--at <环境>@<据点编号>`（schema 也认）→ `gp.ResolveSchema`。**字典同步固定按 `ds` 取**（ds 表档最全）；**实际查库**按环境里选中的营运据点（`hosts.sshs[].company`）取 schema。

**命令面**：`tg dict` 现在只有 `r.t` / `r.q` / `prog` / `msg` / `db` / `spill`，外加 `tg site list|show` 查据点。

**连带影响**：Q6 的"查询数据源（本地镜像 / 远程直查）"已随字典改造接通（设置页里每个 GP 环境都能选本地/远程）。


## 5. 四处手术（§5.1 已定，其余待拍板）

### 5.1 `debug`：**已定 —— 归 TG，拔掉 T100 接线**（2026-11-09 拍板）

**用户的定调**：TG 是与 TT 完全独立的工具，**不需要 T100 的内容**。所以不存在"为共享而抽象"这条路——不做 A（抽接口让 T100 与 GP 共用），也不做"删掉 `debug` 另建一套"。

**先纠正一处我原先写错的事实**（重要，否则会删错东西）：

> 原稿说"GP 调试会话是另起的一套，`internal/gp/session.go` 不 import `debug`，两套 PTY 共存"。
> 这只说对了一半：`internal/gp/session.go` 只是 Web 的**简化 PTY**（`/api/gp/session/{new,exec,close}`，起 `r.d2` 后发命令）。
> **真正在跑的 GP 调试器在 `internal/debug` 里**：`internal/debug/gp.go`（230 行）负责"进实例菜单 → 选实例 → 进模块 4gl 目录 → 校验源码存在 → `r.d2 <短程序名>` → 等 `(fgldb)` → 恢复断点"，
> 另有 12 处 `Product == "gp"` 分支（`config.go:65/273`、`manager.go:177/391/654`、`session.go:778/876/959/1338`、`api.go:2796`）把登录段、模块推导、源码镜像分叉给 GP。
> 也就是说：**按原稿的"B 各留一份、删掉 `internal/debug`"执行，等于删掉唯一能用的 GP 调试器**（断点、停站、事件、WS、前端工作台全在它里面）。

**所以落法是第三种**：`internal/debug` 整体归 TG 所有（它是 fgldb 机制 + GP 接线），只把里面的 T100 领域件拔掉。

**要拔的 T100 接线（4 个文件 + `api.go` 里一支入口，合计不到 13265 行的 1%）**：

| 文件 | 依赖 | 处置 |
|---|---|---|
| `internal/debug/ents.go` | `entdir` 企业清单 | 删（GP 没有 ENT） |
| `internal/debug/db.go` | `entdir`（企业→账号）+ 只读 SQL | 删；账号改由环境配置给 |
| `internal/debug/dbsqldirect.go` | `dict/live` | 删（T100 字典直查） |
| `internal/debug/srcmirror.go` | `entdir` | 改成纯 GP 路径（`$TOP/$CUST` 镜像），去掉企业段 |
| `internal/debug/api.go` 的企业/作业入口 | `target` 的 ENT 形态 | 摘掉 `环境/99`、`@企业` 那一支（见 §5.3） |

**同时要处理的重复**：`internal/gp/session.go` 与 `internal/debug` 的两套 PTY 不能长期并存。阶段 2 二选一——要么把 `internal/gp/session.go` 降级成 `debug` 的薄封装（推荐，前端/WS/断点只有一条路），要么明确它只服务 `/api/gp/session` 的"快速 exec"，不再长大。

**连带删除（阶段 3）**：`cli/debug`（4547 行，T100 作业调试命令面）、`debugclient`（581 行，`cli/debug` 与 `cli/session` 共用的 REST 客户端）。GP 要命令行面的话，在 `cli/gp` 里新写，不复用这两个。


### 5.2 `dict/live` 其实不是字典包

### 5.2 `dict/live` —— **已作废**（2026-11-09：`internal/dict` 不删）

原稿的 Q2 是"把 `live.Open` 那 74 行搬进 `erpdb`，好让阶段 3 删字典时 `cli/gp` 的 import 不断"。

用户拍板：**`internal/dict` 不删，改成 GP 的字典层**（§4.D）。既然字典树留着，`cli/gp` 继续 import `dict/live` 就没有问题，搬家这件事整个作废。

留一句备查：`live.go` 的 74 行确实只做 `erpdb.Open` + 可选 SSH 隧道（`sshtun.Dial`），将来若真要拆，搬进 `erpdb` 仍是最省事的路子。

### 5.3 `target` 的 ENT 语义 → GP 没有企业

`target` 是纯函数（零内部依赖，只 import 标准库），选择器文法有六种形态：`--session 别名`、`环境#序号`、`环境/作业#序号`、`环境/作业`、`环境/99`（ENT）、省略。

GP **没有 ENT**（`cli/gp/register.go` 的包注释明写"GP 没有 T100 的 ENT，所以这里不碰 `--at` / 企业账号"）。所以：

- `环境#序号` / `环境/作业` 这两条对 GP 仍然成立 → 文法值得留；
- `环境/99`、`@企业` 是 T100 的 → 跟着 `cli/at`、`cli/session` 一起删。

处置：`target` 标"Genero 层（文法）+ T100 层（ENT 一种形态）"，阶段 2 把 ENT 那一支摘掉，包本身留下。

### 5.4 `web` 的路由表要可拆

现在三条线硬挂在 `server.go`。阶段 3 删掉 `debug`（字典要改成 GP 的，不删）之后，`web` 要么编译不过，要么留一堆死引用。处置：`web.Options` 的 `Debug`/`Dict`/`GP` 三个 handler 已经是可空接口，只要把"挂载行"收进一个 `mount()`，删的时候是删一行而不是改文件。

## 6. 阶段与闸门

### 阶段 0（现在）：先让基线是绿的

- [x] Go 环境 + `go build ./...`（本会话已做）
- [x] 补回 90 份被 robocopy `/XF README.md` 漏掉的嵌套 README（`f4a02df`）
- [ ] 把 `go test ./...` 的 5 个红包装回绿（其中 4 个是沙箱限制，不是代码问题，见 §8）
- [ ] 补 GP 技能到 `skills/`（现在 4 个全是 T100 的）
- [ ] 修 §7 的接线缺口（4 个是"没接线"，不是"没设计"）

### 阶段 1（并行）：GP 自给自足

TT 的 GP 面现在只有"环境 / 镜像 / 字典 / 调试会话"四件事，**且全在 Web API 上**。命令行只有 `tt gp table|link|msg` 三条查询（**2026-11-11 已改**：`tt gp` 删除，改为 `tt dict r.t` / `tt dict prog --links` / `tt dict msg`，据点清单用 `tt site`）。这个阶段把 GP 侧该有的补上，T100 代码一行不动。

### 阶段 2（替换）：共享件去 T100 化

按这个顺序：**字典换表（§4.D）** → `target` 摘 ENT → `debug` 拔接线（§5.1）→ `web` 路由收成 `mount()`。每做完一处，`go build ./...` + `go test ./...` 都要绿。

### 阶段 3（剥离）：删 + 改名

删 §4 的代码与资产 → 删 T100 技能 → 改 `module tt` / 二进制名 / MSI 产品名 / 数据目录 / `TG_WEB_DIR` 这类内部标识。

### 每阶段的闸门

| 闸门 | 命令 | 说明 |
|---|---|---|
| 编译 | `go build ./...` | 必须 |
| 单测 | `go test ./...` | 沙箱里有 4 个包恒红，见 §8；真机应为绿 |
| 前端 | `cd web && npm run check:app` | 只在改前端时 |
| T100 自检 | ~~`./tg.exe dev tzc selftest`（31 项）~~ | 已随 `tt dev` 删除消失（2026-11-11） |
| GP 全链路 | 配环境 → 测连通 → 同步字典 → 拉镜像 → 起 fgldb → 跑一条 `tt dict r.t` | **手工**，没有自动化；这条是"TG 还能用"的唯一判据 |

## 7. 今天实测出来的缺口（不是设计缺失，是没接线）

这八条都有代码证据，改起来都不大，但会决定用户第一次打开 TG 的印象。

1. **`tt debug` 完全不认 GP 环境。** `internal/cli/debug` 全包搜不到一处 `Product` 判断。对着 GP 环境跑 `tt debug`，它会按 T100 的路径去解析作业、企业、源码镜像——报错大概是"找不到作业"，而不是"这是 GP 环境"。
2. **GP 调试没有命令行面。** 只有 `/api/gp/session/{new,list,exec,close}`；~~`tt gp` 只有 `table|link|msg`~~（已删，现为 `tt dict …`）。也就是说：**GP 版的"调试"今天只有 Web 能做，命令行做不了**。而 TT 的定位是 CLI-first。
3. ~~**`tt file mirror` 是 T100 专用。**~~ —— **2026-11-12 已改**：`tg file mirror` 不再走 T100 的 zone/`erp|com`/`tar.gz`/profile，改为 `internal/mirror`（两棵树 `$TOP`+`$CUST`、白名单 `*/4gl/*.4gl` + `*/42f/*` + `*.inc`、SFTP 逐文件 + mtime 增量、`--full` 剪枝、`manifest.json` 记规模与 kinds）。实测规模：4gl 6,971 个 / 376.5 MB、42f 6,771 个 / 57.0 MB、inc 24 个 / 0.3 MB，合计 **433.8 MB / 13,766 个**，首拉约 10–12 分钟。**没有按 profile 拆**：一个镜像目录要装齐所有读者的并集（`tg outline` 读 4gl、`tg interface` 读 42f、`grep/cat` 读 inc），拆开只会让某条命令找不到文件；42s 在 GP 恒为 0（标签已改由 `gae_file` → `t_ui_text` 提供）。网页 `/api/gp/mirror` 与命令行同源。
4. **前端有两个端点不存在（GP 页的"查询数据源"其实是三处断线）。**
   - `web/app/src/gpEnvs.ts:40` 的 `gpPickDir` POST `/api/gp/pickdir` → 后端只有 `POST /api/pickdir`（`internal/web/server.go:161` 挂的 `internal/web/pickdir.go`，本机回环、返回 `{ok,path,canceled}`）。返回 404，函数吞掉异常返回空串，**表现是"选目录按钮没反应"**。T100 侧 `web/app/src/api.ts:487` 调的正是 `/api/pickdir`，改一行 URL 就对齐。
   - `web/app/src/GpSettings.tsx:576` 的 `setSource` POST `/api/gp/settings` → `gp.Service.ServeHTTP` 的 switch 里没有这一个，**表现是"改查询源报错"**。T100 的对应实现是共享端点 `PUT /api/hosts` 的 `query.source` 节（`SettingsView.tsx:748`；`api.ts:194` 的 `HostsPatch`）。
   - 更麻烦的是第三处：`GpSettings.tsx:137` 从 `GET /api/gp/meta` 读 `m.querySource`，而 `gp.go:262` 的 `hMeta` 只吐 `envs`（name/host/user/mirror/dict/db/job），**没有 `querySource` 字段** → 下拉框永远显示"在线"。而且 `internal/gp` 全包搜不到 `query`/`source` 的读取，`cli/gp/register.go:164` 的 `openLive` 也只有一个在线路径。**这个控件读不到、存不下、也没人消费。**
   两处 404 都是"服务从两进程并成一进程"时漏搬的端点（`gpEnvs.ts` 的注释还写着"GP 服务的 /api/pickdir"）。
   **pickdir 已修**（改调 `/api/pickdir`，前端 `tsc --noEmit` 与 `check:app` 全绿，dist 已重建）。`/api/gp/settings` 与 `querySource` 两处按 **Q6 = a** 处理：控件先**不渲染**（`GpSettings.tsx` 顶部的 `SHOW_QUERY_SOURCE = false`，代码保留），等 §4.D 字典改造落地后连同后端一起接通——现在渲染它只会给用户一个点了报错的死控件。
5. **`skills/` 没有 GP 技能。** 分发的 4 个技能全是 T100 的；GP 那套（`gp-erp-architecture`/`-debug`/`-read`）现在只在本机 `.agents/skills` 之外。用户装完 TG，Agent 拿到的是 T100 文档——这是**对外发版前必须换的**。
6. **`internal/README.md` 的包地图里没有 `internal/gp`。** 55 个包的清单从 `cli/at` 到 `cli` 都有，唯独这个 2067 行、带 9 个端点的子系统没登记（整份文档里 `gp` 出现 0 次）。
7. **GP 字典只有三条命令，本地库与 Oracle 的口径还不一致。** ~~`tt gp` 只有 `table|link|msg`~~（2026-11-11 已改为 `tt dict`）；~~本地库已同步七张表~~ → **2026-11-13 现状**：本地库 10 张数据表（`t_table`/`t_column`/`t_index`/`t_constraint`/`t_module`/`t_prog`/`t_prog_table`/`t_prog_link`/`t_msg`/`t_ui_text`）+ `meta`，实测 2,962 张表 / 60,870 字段 / 6,465 支程序 / 118,420 条程序用表 / 17,810 条链接 / 42,031 条消息 / 427,553 条画面文字，同步 1 分 48 秒、68.7 MB；`db.Source` 的 20 个方法已全部接上（逐方法接线见 [gp-dict-wiring.md](gp-dict-wiring.md)）。~~实测还暴露三处不一致~~ → **三处已修**（`068b19f`/`14413d9`/`92d6a95`/`2110c95`）：①同步固定按 `ds`、直查按营运据点；②`t_table` 记 `zta07` 种类，61 张同义词只登记不建列；③`t_prog_link` 照常填（17,810 行）。
8. **`GET /api/gp/meta` 明文回吐口令。** 实测响应体里带环境的**明文 SSH 口令**与库账号的**明文口令**（形如 `"password":"<明文>"`，本文档不记录其值）——前端只用到 `hasPassword: true`，根本不需要明文。虽然服务只听 127.0.0.1，但本机任何进程/浏览器页面都能读到；`hMeta` 应改成只回 `hasPassword`。这是**对外发版前必须扫掉**的一条。

## 8. 沙箱跑不出来的测试（别误判成代码坏了）

`go test ./...` 在本会话有 5 个包红，其中 4 个与代码无关：

| 包 | 报错 | 真实原因 |
|---|---|---|
| `internal/host` | `exec: "bash": executable file not found in %PATH%` | 测试用 bash 反解 shell 引号；本机 bash 起不来（msys fork 限制） |
| `internal/pathinstall` | `创建临时注册表键失败: Access is denied` | 沙箱不许写 HKCU |
| ~~`internal/dev/tzs`~~ | `连不上管道 tzs-cli-...: Access is denied` | 沙箱禁命名管道（包已于 2026-11-11 删除） |
| ~~`internal/cli/dev/tzc`~~ | 同上（121s 超时后失败） | 同上（包已于 2026-11-11 删除） |

只有 `internal/cli/dev/common` 是真缺陷（缺 `internal/cli/dev/tzs/README.md`），已由 `f4a02df` 修好；该包已于 2026-11-11 与 `tt dev` 一起删除，现在跑 `go test ./internal/...` 只剩 `internal/host` 红。

## 9. 数字怎么来的（可复现）

```powershell
# 每包非测试 .go 行数（PowerShell；本仓库只在 Windows 上开发）
Get-ChildItem internal -Directory | ForEach-Object {
  $src = Get-ChildItem $_.FullName -Recurse -Filter *.go |
         Where-Object { $_.Name -notlike '*_test.go' }
  [pscustomobject]@{ 包 = $_.Name; 行 = ($src | Get-Content -Encoding UTF8).Count }
} | Sort-Object 行 -Descending
```

口径：只算 `*.go` 且排除 `*_test.go`；资产树按文件数 / MB；`internal` 合计 ≈74500 行。

注意别用 `Measure-Object -Line` 代替 `.Count`——它在多文件管道上会漏记（同一批文件量出 69000，比真实值少 5000 多）。

## 10. 待拍板

- ~~**Q1**：`debug` 合一还是各留一份~~ —— **已定（2026-11-09）**：TG 与 TT 完全独立，不与 T100 共享代码；`internal/debug` 归 TG，拔掉 T100 接线，不新建第二套（§5.1）。
- ~~**Q2**：`dict/live` 搬到 `erpdb` 还是新建 `livedb`~~ —— **已作废（2026-11-09）**：`internal/dict` 不删，改造成 GP 的字典层（§4.D），`cli/gp` 继续用它没有断 import 的问题。
- ~~**Q3**：GP 镜像要不要扩到 `42f/42s/inc`~~ —— **已改（2026-11-12）：不做 profile，统一收 `4gl + 42f + inc`**。原计划（`debug` 只 4gl / `source` 收全四类）作废：一个镜像目录必须装齐所有读者的并集，否则某条命令会找不到文件；42s 在 GP 上不存在（标签在 `gae_file`，走字典线）。见 §7.3 第 3 条。
- ~~**Q4**：`designer-src/` + `engine/`（16 MB、1264 文件）什么时候删~~ —— **已做（2026-11-12）**：连同 `tools/tzsmini`、`testdata/tzs-mini` 一起删，打包成 `D:\Digiwin\tt_work\archive\TG-t100-designer-assets-2026-11-12.zip`（7.13 MB / 1355 条目）；`build_portable.bat` 里打 engine + designer 的两段（`[3b/5]`/`[3c/5]`）同时删掉，发行包不再带任何厂商 DLL。
- ~~**Q5**：阶段 3 的改名要不要一次性做~~ —— **已定（2026-11-09）：分三批**。①对外标识（二进制名、MSI 产品名、PATH 写入名、文档自称）②数据与配置（`~/.tt`→`~/.tg` 带一次性迁移、`TG_WEB_DIR`、`.tg-serve.json`、`TT_*` 变量读旧名兼容）③代码内部（`module tt`→`module tg` 全仓 import 重写 + 重生成 `commands.html`）。每批一个 commit、可单独回滚。**2026-11-13 进度**：②③ 已做（`%APPDATA%\tt`→`%APPDATA%\tg`、`~/.tt`→`~/.tg`、`.tt-serve.*`→`.tg-serve.*`、`TT_*`→`TG_*`、`module tg` + 176 文件 import 重写；**不保留旧名兼容**，TG 没有已发版的旧数据/旧脚本要接）；①未做（`tg.exe`、`installer/tg.wxs`、`%LOCALAPPDATA%\Programs\TG` 原样）。
- ~~**Q6**：GP 页的"查询数据源"怎么处理（§7.4）~~ —— **已完成（2026-11-13）**：设置页「数据字典 → 查询数据源」现在是真控件，下拉里列的是 GP 环境（数据来自 `/api/gp/meta` 的 `env.dict`），选「本地镜像」即 `tg dict … --env local:<环境>`（读 `dict\<环境>.db`），选「远程直查」即 `--env <环境>`；`query.source` 落 `config.json`，命令行 `--at`/`--env local:` 单次覆盖仍在。原计划的 `SHOW_QUERY_SOURCE = false` 与 `POST /api/gp/settings` 都随 §4.D 落地作废。
