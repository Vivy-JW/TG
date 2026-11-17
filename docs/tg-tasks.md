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
- ☑ `tg dict db sync` 的 GP 分支（进度打 stderr，JSON 整包）
- ☑ 类别名 GP 码表（`db.CategoryName`）
- ☑ 语言码映射 `db.LangCode`（zh_TW→0、zh_CN→2）
- ☑ 「表格总数」按表去重（`COUNT(DISTINCT zr02)`，224 → 159）
- ☑ 测试按 GP 语义重写（`dict/db`、`dict/live`、`cli/dict` 全绿）
- ☑ 删剩下六族 T100 SQL（`check/scc/win/param/acc/spec`）与其 DTO（2026-11-11~13：`check`/`scc`/`param`/`acc`/`spec` 连命令带方法一起删；`win` 是 r.q，GP 有 `gab_file`/`gac_file`，改造成 GP 版留着）
- ☑ `Source` 接口瘦身（2026-11-13 起 30 → 17；2026-11-17 再并两条 —— `QueryMsgLangs`→`QueryMsgs`、`QueryProgJobs`→`QueryProgInfo.Jobs`。现在 `internal/dict/db/source.go` 正好 15 个 `Query*`，没有空壳方法）
- ☑ `internal/dict/live/README.md` 与包注释去 T100 表述（2026-11-13 重写；全文已无 `T100`/`dz*`/`gz*`/`ENT`/`企业` 字样）
- ☑ 更新 `docs/gp-dict-wiring.md` 的 ✅/⬜ 标记与 `docs/gp-scope.md` §4.D（2026-11-13：wiring 表 15 ✅ / 0 空壳，§4.D 标「已落地」）
- ☑ 提交字典层改动（`a75acb3`）
- ☑ 清理 `tmp\gp-probe.json`（含明文口令）、`tmp\gpdict`、`tmp\tg.exe`
- ☑ TG 起自己的运行实例（`run\`，独立端口），不再借用 TT 的 28670

## 阶段 2 调试面

- ☑ `tg debug` 接 GP（2026-11-16~17 真机实测：`tg session job --prog abmi600` → `stopped`、`module=cbm`、`stop.file=abmi600.4gl`、`reason=entry`；断点/步进/变量与网页调试台同走 `/api/sessions/*`）
- ☑ 摘 T100 接线（2026-11-17 `24b2c45`）：`ents.go`/`ent_required_test.go` 删除、`dbsqldirect.go` 已不存在、`db.go` 去掉 `entdir` 与 `ProbeDB(ent)`、`srcmirror.go` 改纯 `$TOP/$CUST`、`api.go` 的 ENT 入口与 `/api/ents*` 一起删
- ☑ 连带删 `tt debug ents` / `tt debug topent`（2026-11-16 `c0e3974` 删 `--fglserver`，2026-11-17 `24b2c45` 把 ENT 管线整条删掉：`--ent`、`/api/ents*`、`cfg.Topent`、`requireExplicitEnt`、快照 `ent`/`topent*`）
- ☑ `tg session` 收成 GP 语义（2026-11-16）：去掉 `--ent` / `session ent` / `--fglserver`，只要环境；`ls`/`show` 的 JSON 不再带 `ent` 与三个 `topent*` 键；`session new` 的作用域从 `EnvEntReq` 降成 `Env`
- ☑ 两套 PTY 二选一（2026-11-17）：**决定删，不做薄封装** —— 删掉 `internal/gpsession` 与 `/api/gp/session/{new,list,exec,close}`（前端/CLI/文档 0 调用方，能力是 `internal/debug` 的子集）；三条真机经验并进 `internal/debug/gp.go` 的 `bootGP`/`startRunGP`，并把两处盲等改成「先等 shell 提示符、等不到再兜底睡」
- ☑ 删设置页「发现环境 — 从 T100 设计器导入」（2026-11-17）：`internal/designer` 整包、`GET /api/discover/envs`、`api.discoverEnvs` 与前端弹窗/状态一起删；GP 没有设计器站点，那份 `sites.xml` 还带明文 SSH 口令
- ☑ 调试工作台前端在 GP 下验一遍（2026-11-17 真浏览器 CDP 驱动无头 Edge 实测）：界面「作业编号 abmi600」→ 建会话 → 自动跳调试页并停在入口（`module=cbm`、`stop.reason=entry`）；「继续 (F5)」→ `state=running`、步过按钮正确置灰；点编辑器**行号栏**下断点 → `cbm_abmi600.4gl:785` 进断点表、「继续」停在 785（`reason=breakpoint`、`func=main()`）；右栏变量监视/断点/调用栈渲染正常、源码区显示 `topcust/cbm/4gl/abmi600.4gl`；「结束调试」可收掉会话。顺带查出并修掉两个真 bug：① 左树判空只看 `envs`（`loadEnvs` 里 T100 时代的 `product==='gp'` 过滤让 `envs` 恒空），明明有环境却显示「还没有环境。」——改按 `envs+gpEnvs` 判空；② 入口停站的 `stop.file` 在 GP 上是裸作业名（`abmi600.4gl`），按它下断点会被 fgldb 拒（`No source file named abmi600.4gl.`），改成 `<模块>_<作业>.4gl`（`internal/debug/session.go` 的 `entryStopInfo`，带 `TestEntryStopInfoFileGPProgHasNoPipePrefix`）。
- ☑ 真机测试后确认（2026-11-17）：`tg session ls` 空表、`fgldb` 进程 0 个、28680 上只有一个 `tg serve`（pid 27128），没有残留作业

## 阶段 3 删 T100 领域层

- ☑ `tt dev tzc` / `tt dev tzs`（10 条命令，2026-11-11 连同 `internal/dev` 的 fence/model/pkgfile/split/store/synth/tapfile/tglfile/tzs/verify 一起删了；只留 `dev/fgl` 与 `dev/testutil`）
- ☑ 删旧命令手册 `.agents/docs/architecture/commands.html`（2026-11-17，TT 时代的命令快照，含 .tzc/.tzs 两个整组已删命令与 `tt env ents` 样例，全仓无引用）`n- ☑ 删 `engine/` + `designer-src/`（1264 文件 ≈17 MB），删前打了 zip 到仓库外（2026-11-12，commit 见 handoff 第 35 条）
- ☑ 确认 `engine/designer/` 28 个厂商 DLL（10.5 MB）不进 TG 发行包（engine 整个删了，`build_portable.bat` 的 staging 段也删了）
- ☑ `tg interface`（3 条）：标签改查 gae_file，`.42s` 路径已删（2026-11-12）
- ☑ `tg outline` 换 GP 目录结构：不换结构，改成"本地文件 > `--dir` > 镜像 > 直读服务器"（与 `tg interface` 同源，2026-11-12）
- ☑ `tg file mirror` 改 GP 口径：两棵树 + 4gl/42f/inc，不做 profile（2026-11-12，见 handoff 第 39 条）
- ☑ `tg db sql` 去掉"强制 `环境@企业`"：身份只有环境，账号取 `accounts[0]`，裸表名落在账号的 schema（信封的 schema 字段）
- ☑ 从命令层摘掉 `target` / `entdir`（2026-11-17 `800554e`）：`internal/entdir` 整包删除、`ents` 缓存目录从清单移除；`target` **有意保留** —— 它已不含 ENT，只剩寻址文法与信封契约（`internal/debug`、`debugclient`、`cli/debug` 在用）
- ☑ `skills/` 换成 GP 版（`tt-dev-tzc` 已删，`tt-erp-*` 已改名 `tg-erp-*`，commit a42a1a2）

## 阶段 4 安全与发版

- ☑ `GET /api/gp/meta` 回明文口令 —— **决定保留**（只监听 127.0.0.1；`run\config.json` 本身就必须存明文，设置页要能编辑口令并回显；TT 侧同样明文）
- ☑ 前端 `dist` 重编 + exe 重打（多轮；2026-11-17 最新产物 `web/dist/assets/index-*.js` ≈4.04 MB，exe `0.5.2-2026-11-17h`）
- ☑ `build_portable.bat` / `installer/tg.wxs` 改 TG 名称、产品名（2026-11-13 `1685d08`；MSI 2026-11-16 实跑产出 `dist\TG-0.5.2-x64.msi`，UpgradeCode 换成 TG 自己的）
- ☑ 版本号与 CHANGELOG（2026-11-17）：TG 作为新工具重新起版本，`VERSION` = **0.1.0**（不沿用 TT 的 0.5.2）；同步 `web/package.json`、`web/app/package.json`、`web/package-lock.json`（3 处 version + 2 处 `tg-web` 名字），`make version-check` 的 node 脚本增查 `packages["app"]`；新建 `CHANGELOG.md`（Keep a Changelog，只记 0.1.0 这一版）；README 的「当前状态/常用命令」同步到 GP 口径

## 阶段 5 改名三批

- ☑ ①对外（2026-11-13）：`Makefile` / `build_portable.bat` / `build_msi.bat` 出 `tg.exe`、`dist\tg-portable\`、`dist\TG-<ver>-x64.msi`；`installer/tt.wxs` → `installer/tg.wxs`（产品名/目录/注册表 `Software\TG`/开始菜单名/`install bdl` 全改）；`internal/update` 的 `Programs\TG`、备份目录 `.tg-update-prev`、临时前缀 `tg-update-`、UA `tg-update/`、状态与日志 `.tg-update.json|log`、技能戳 `.tg-skills.json` 同步。MSI 已实跑（2026-11-16）：WiX v3.14.1 放在 `D:\tt-build-tools\wix3`（`build_msi.bat` 的默认 `WIX_BIN`），产出 `dist\TG-0.5.2-x64.msi`（19.9 MB）；`installer/tg.wxs` 的 `UpgradeCode` 已换成 TG 自己的 `B4E6D2A9-5C3F-4E18-9A7B-2D6F8C1E5A03`（原来沿用 TT 的，装 TG 会把机器上的 TT 当旧版本升级掉）。
- ☑ ②数据与配置（2026-11-13）：`%APPDATA%\tt` → `%APPDATA%\tg`、`~/.tt` → `~/.tg`（字典/镜像兜底）、`.tt-serve.json|log` → `.tg-serve.json|log`、`TT_*` 环境变量全改 `TG_*`。**不做旧名兼容**（TG 没有已发版的旧脚本/旧数据要接）
- ☑ ③代码内部（2026-11-13）：`module tt` → `module tg`，176 个 Go 文件的 import 全量重写、`-ldflags -X tg/internal/cli.Version` 同步；`go build ./...` 与 `go test ./internal/...` 通过（`internal/host` 仍因沙箱无 bash 恒红）

## 阶段 6 验证与发布

- ☑ 全量 `go test ./...`（2026-11-17：装上 Git for Windows 补出 bash 之后全绿，exit 0 —— 原先那 3 红确是环境缺件）
- ☑ 真环境端到端（2026-11-17，22主机/TOPTEST）：`tg dict db sync --env 22主机` 1m45s 写入 68.7 MB（表 2,962 / 字段 60,870 / 关系 118,420 / 消息 42,031 / 画面文字 427,553 / 程序链接 17,810）→ `tg dict r.t ima_file`、`tg dict prog abmi600`、`tg dict msg -263` 均 `ok:true` → `tg session job --prog abmi600` 真机停在入口
- ☐ 打 tag、README、发 release

## 待决策

- ☑ 阶段 2 起手方式（2026-11-17）：先接 GP 跑通（`internal/debug` 直接拥有 GP 路径），两套 PTY 的残留随后删掉，不保留第二条路
- ☑ 把 `.agents/skills/` 挂进 DSH 技能目录，让工作区规范能自动触发（已生效：会话可用技能里有 `tg-pj-add-cli`/`tg-pj-git`/`tg-pj-plan`/`tg-pj-test`）
