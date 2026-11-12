# TG 开发任务清单

> 大纲见 [tg-plan.md](tg-plan.md)。`☑` 已完成、`☐` 未开始。
> 每完成一项就把 `☐` 改成 `☑`，并在行尾补依据（commit 号 / 实测数字）。

## 阶段 0 基线

- ☑ 导入 TT 0.5.2 源码基线（`2fd193b`，2110 文件，排除 `*.exe`/`*.bak-*`/`node_modules`）
- ☑ `.gitignore` 补 `/tg.exe*`、`/tmp/`
- ☑ README 重写为 TG 版（写明"二进制暂时仍叫 tt"）
- ☑ `.agents/skills/` 四个 `tt-pj-*` 改名 `tg-pj-*`（`89d03ca`）
- ☑ 补回 90 份被 `robocopy /XF` 挡掉的嵌套 README（`f4a02df`）
- ☑ Go 构建与测试跑通（`GOCACHE` 指工作区，避开沙箱；基线 50 绿 / 5 红）
- ☑ 写出 `docs/gp-scope.md`（底座 / Genero 层 / T100 领域层三层删留）
- ☑ 写出 `docs/gp-dict-wiring.md`（30 个 `Source` 方法逐条接线）
- ☑ 写出 `docs/tt-commands.xlsx`（99 个末端命令 + 处置：留 31 / 改 53 / 删 15）
- ☑ 前端：GP 选目录修 404（`b2aa3aa`）
- ☑ 前端：删从未挂载的 `GpDict` 死组件（`0be3167`）
- ☑ 真实 GP 环境实测（TOPTEST：同步 1m18s / 20.5 MB；`zr_file` 16 万行；`zz_file` 类别定义）
- ☑ 计划与任务拆成两个文件（`docs/tg-plan.md` + `docs/tg-tasks.md`）

## 阶段 1 字典层 GP 化

- ☑ 同步扩表：`t_msg`、`t_prog_table`，`t_prog` 加 `category/run_cmd/name_tw`，补抓 `zz_file`/`zr_file`/`ze_file`
- ☑ 本地 `internal/dict/db` 表族改查 `t_*`（`QueryTable/Field/TableList/TableMeta/Keys/Indexes`）
- ☑ 本地 `internal/dict/db` 程序族改查 `t_prog`/`t_prog_table`（作业恒 1 条）
- ☑ 本地 `internal/dict/db` 消息族改查 `t_msg`
- ☑ 远程 `internal/dict/live` 整篇重写为 GP SQL，删 `acc.go`（270 行 T100 ACC）
- ☑ 删掉并行实现 `internal/dict/gpdb/`
- ☑ 路由：GP 环境不经 `entdir`；本地走 `gp.DictPath()`，远程直连用 `accounts[0]`
- ☑ `tt dict db sync` 的 GP 分支（进度打 stderr，JSON 整包）
- ☑ 类别名 GP 码表（`db.CategoryName`）
- ☑ 语言码映射 `db.LangCode`（zh_TW→0、zh_CN→2）
- ☑ 「表格总数」按表去重（`COUNT(DISTINCT zr02)`，224 → 159）
- ☑ 测试按 GP 语义重写（`dict/db`、`dict/live`、`cli/dict` 全绿）
- ☐ 删剩下六族 T100 SQL（`check/scc/win/param/acc/spec`）与其 DTO
- ☐ `Source` 接口瘦身（30 个方法 → 只留有 GP 数据的十来条）
- ☐ `internal/dict/live/README.md` 与包注释去 T100 表述
- ☐ 更新 `docs/gp-dict-wiring.md` 的 ✅/⬜ 标记与 `docs/gp-scope.md` §4.D
- ☑ 提交字典层改动（`a75acb3`）
- ☑ 清理 `tmp\gp-probe.json`（含明文口令）、`tmp\gpdict`、`tmp\tg.exe`
- ☑ TG 起自己的运行实例（`run\`，独立端口），不再借用 TT 的 28670

## 阶段 2 调试面

- ☐ `tt debug` 接 GP（起会话 / 停站 / 断点 / 步进）
- ☐ 摘 T100 接线：`ents.go`、`db.go`、`dbsqldirect.go`、`srcmirror.go`、`api.go` 的 ENT 入口
- ☐ 连带删 `tt debug ents` / `tt debug topent`
- ☐ `tt session` 收成 GP 语义（现在的对象是 ENT/作业形态）
- ☐ 两套 PTY 二选一：把 `internal/gp/session.go` 降级成 `internal/debug` 的薄封装
- ☐ 调试工作台前端在 GP 下验一遍
- ☐ 真机测试后确认：会话已 `quit`、服务已 `stop`、无残留作业或行锁

## 阶段 3 删 T100 领域层

- ☑ `tt dev tzc` / `tt dev tzs`（10 条命令，2026-11-11 连同 `internal/dev` 的 fence/model/pkgfile/split/store/synth/tapfile/tglfile/tzs/verify 一起删了；只留 `dev/fgl` 与 `dev/testutil`）
- ☑ 删 `engine/` + `designer-src/`（1264 文件 ≈17 MB），删前打了 zip 到仓库外（2026-11-12，commit 见 handoff 第 35 条）
- ☑ 确认 `engine/designer/` 28 个厂商 DLL（10.5 MB）不进 TG 发行包（engine 整个删了，`build_portable.bat` 的 staging 段也删了）
- ☑ `tg interface`（3 条）：标签改查 gae_file，`.42s` 路径已删（2026-11-12）
- ☑ `tt outline` 换 GP 目录结构：不换结构，改成"本地文件 > `--dir` > 镜像 > 直读服务器"（与 `tg interface` 同源，2026-11-12）
- ☑ `tg file mirror` 改 GP 口径：两棵树 + 4gl/42f/inc，不做 profile（2026-11-12，见 handoff 第 39 条）
- ☐ `tt db sql` 去掉"强制 `环境@企业`"
- ☐ 从命令层摘掉 `target` / `entdir`
- ☑ `skills/` 换成 GP 版（`tt-dev-tzc` 已删，`tt-erp-*` 已改名 `tg-erp-*`，commit a42a1a2）

## 阶段 4 安全与发版

- ☐ `GET /api/gp/meta` 别回明文口令（前端只要 `hasPassword`）
- ☐ 前端 `dist` 重编 + exe 重打
- ☐ `build_portable.bat` / `tt.wxs` 改 TG 名称、图标、产品名
- ☐ 版本号与 CHANGELOG

## 阶段 5 改名三批

- ☐ ①对外：二进制 `tg.exe`、MSI 产品名、PATH 写入名、文档自称
- ☐ ②数据与配置：`~/.tt` → `~/.tg`（带一次性迁移）、`TT_WEB_DIR`、`.tt-serve.json`、`TT_*` 读旧名兼容
- ☐ ③代码内部：`module tt` → `tg` 全仓 import 重写 + 重生成 `commands.html`

## 阶段 6 验证与发布

- ☐ 全量 `go test ./...`（沙箱外；现有 3 红是沙箱限制）
- ☐ 真环境端到端：sync → `dict prog/table/msg` → 调试会话
- ☐ 打 tag、README、发 release

## 待决策

- ☐ 阶段 2 起手方式：先接 GP 跑通（两套 PTY 并存一阵），还是先合并成一套再接线
- ☐ 把 `.agents/skills/` 挂进 DSH 技能目录，让工作区规范能自动触发
