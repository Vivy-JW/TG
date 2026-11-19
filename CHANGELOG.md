# 变更记录

本文件记录 TG 的版本变更。格式参照 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

TG 是 TT 的继任者，只面向 **鼎捷 TOP GP**（T100 分支已整体移除），因此版本号从 `0.1.0` 重新起算，
不沿用 TT 的 `0.5.x`。

## [0.1.0] - 2026-11-19

首个面向 TOP GP 的版本。

### 新增

- **站点管理**：按环境读取营运据点清单（`azp_file`），据点编号与 schema（`azp03`）分开显示；
  `tg site list/show` 与设置页「站点管理」页签同源。
- **数据字典**（`tg dict`）：表、字段、索引、约束、程序、程序与表的双向关系、消息、画面文字，
  全部改为读 GP 的 `zta/ztb/ztc/ztd/zte/zs`、`gal_file`、`gae_file`、`ze_file`，
  并可同步成本地 SQLite（`tg dict db sync`）供离线查询。
- **开窗定义**（`tg dict r.q`）：读 `gab_file` + `gac_file`，按 GP 的 `cl_create_qry` 规则现拼开窗 SQL
  （含 `SELECT UNIQUE`、Oracle 外连接 `col(+)`、HardCode 开窗识别）。
- **代码阅读**：`tg outline`（4GL 大纲）、`tg file`（按 `$TOP`/`$CUST` 两棵树读文件）、
  `tg interface`（`.42f` 画面 + `gae_file` 标签）、`tg dict prog --links`（p_link 模块关系）。
- **调试**：`tg session` 与 `tg debug` 走 GP 的 `r.d2` + `fgldb` 链路 —— 建会话、起作业、
  断点、步进、变量、停站现场、原始协议流；身份只要「环境」，账号固定取 `db.accounts[0]`。
- **Web 调试工作台**：`tg serve` 起本地服务，浏览器里建会话、下断点、看变量、看界面日志、改设置。
- **源码镜像**：`tg file mirror pull` 把服务器上的 `*/4gl/*.4gl`、`*/42f/*`、`*.inc` 拉到本地，
  供离线检索（`--at <环境>`，默认增量）。
- **随包的两份 Markdown 手册**：`BDL-Markdown.zip`（Genero BDL User Guide，19 章）与
  `TOP-GP-Markdown.zip`（TOP GP 技术手册，13 章 / 237 个 md）跟安装包一起发，用
  `tg install bdl` / `tg install gpdoc` 解到本机（`--to` 换位置、`--from` 换 zip、`--force` 覆盖、
  `--dry-run` 只看落点）；落点可用 `bdldoc.dir` / `gpdoc.dir` 长期改；反查目录用 `tg docs`（`--json` 给脚本读）。设置页「应用设置」
  两张卡显示生效目录与装没装。便携包与 MSI 都会 stage 两个 zip，MSI 装完自动各跑一次
  （`Return="ignore"`，装失败不影响安装本身）。

### 变更

- 命令前缀由 `tt` 改为 `tg`，可执行文件为 `tg.exe`；数据目录改为 `%APPDATA%\tg`、`~/.tg`，
  环境变量前缀改为 `TG_*`。
- Go module 名由 `tt` 改为 `tg`。
- 安装包换成 TG 自己的 `UpgradeCode`，与本机可能装着的 TT 0.5.x 互不影响。

### 移除

- **企业编号（ENT）整条管线**：`--ent` 及各命令的 ENT 语义、`/api/ents*`、`/api/sessions/{id}/ent|topent`、
  快照里的 `ent`/`topent*` 键、`internal/entdir`、企业快照缓存。GP 一个环境一个库，账号就是 `db.accounts[0]`。
- **`.42s` 与 FGLSERVER**：GP 没有编译字符串表，标签改读 `gae_file`；画面由 fgldb 交给 GDC 直连，
  `--fglserver` 及本机 GDC 接线探测全部删除。
- **设计器相关**：`.tzc` / `.tzs` 两条包管线（`tg dev` 整组）、`engine/`、`designer-src/`、
  `tools/tzsmini`、`testdata/tzs-mini`、从设计器 `sites.xml` 导入环境的链路（`internal/designer`、`/api/discover/envs`）。
- **TT 时代的其它命令**：`tt gp` 整组（并入 `tg dict` / `tg site`）、`tt debug ents`/`topent`、
  `tt env ents`、T100 字典命令（`r.v`/`desc`/`scc`/`acc`/`sysp`/`docp`）、`/api/mirror` 与 `internal/host/mirror.go`。
- **重复的 PTY 实现**：`internal/gpsession` 与 `/api/gp/session/*`，只保留 `internal/debug` 一套。

### 已知限制

- 断点命中后要继续操作，需要在 GDC 里点开对应作业；纯步进的自动化验证建议用不弹窗的批处理作业。
- 接口日志页（`wsfa_t`）在 GP 库里没有对应表，该页查询会报「查询出错」。
- `tg update` 的自动更新链路尚未对着真实 release 做过端到端验证。
