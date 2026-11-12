# TG 开发计划（大纲）

> 目标：把鼎捷 T100 的开发 CLI（TT 0.5.2）改造成 TOP GP（Genero 4GL）版，命名 TG。
> 独立项目、个人维护、对外只发自己的安装包；T100 专有模块最终全部剥离。
>
> 具体事项在 [tg-tasks.md](tg-tasks.md)（`☐` 未做 / `☑` 已完成）。
> 依据文档：[gp-scope.md](gp-scope.md)（分层删留）、[gp-dict-wiring.md](gp-dict-wiring.md)
> （字典接线）、[tt-commands.xlsx](tt-commands.xlsx)（118 条命令 + 处置初判）。

## 范围与原则

- **只保留 GP 用得上的东西**：T100 的字典表、设计器包、接口合成、企业账号体系都不进 TG。
- **删与改分开提交**：删除放最后，因为要能单独回滚。
- **删除前先备份**：`TT-src` 没有 git 历史，动大件前先打 zip 存仓库外。
- **对外标识与代码内部标识分三批改**（见阶段 5），每批一个 commit。
- **单人维护、未发版**：不需要开分支和 PR，改动直接进 `main`，保持 `main` 可用即可。

## 阶段大纲

| 阶段 | 目标 | 主要交付物 |
|---|---|---|
| 0 基线 | 把 TT 0.5.2 搬进 TG 并能构建、能测试、有文档 | 干净基线 + 三份依据文档 |
| 1 字典层 GP 化 | `tt dict` 成为 GP 字典的唯一入口，本地/远程两条都只认 GP 表 | 同步扩表、`db` 与 `live` 两套 GP SQL、接口瘦身 |
| 2 调试面 | `tt debug` 能对 GP 起会话、下断点、步进 | `internal/debug` 拔掉 T100 接线，PTY 合成一套 |
| 3 删 T100 领域层 | 设计器包工具、接口合成、企业账号体系全部消失 | 代码 + `engine/` + `designer-src/` 一起删 |
| 4 安全与发版 | 能对外发的安装包 | 口令不回吐、dist/exe 重打、MSI 更名 |
| 5 改名三批 | 对外自称与代码内部标识都叫 TG | 二进制名 / 数据目录迁移 / module path |
| 6 验证与发布 | 全量测试 + 真环境端到端 + 打 tag | release |

## 已定决策（不再推翻）

| 编号 | 决策 | 影响 |
|---|---|---|
| Q1 | GP 调试归 TG，`internal/debug` 保留、拔掉 T100 接线 | 不与 TT 共用抽象；连带删 `cli/debug` 的 ENT 命令 |
| Q2 | `internal/dict` 保留，改造成 GP 字典层 | 换表不删包；原"删整个 dict"作废 |
| Q3 | GP 镜像不拆 profile | 统一收 4gl + 42f + inc（两棵树）；42s 在 GP 不存在，标签走 `gae_file`（2026-11-12 改） |
| Q4 | `designer-src/` + `engine/` 阶段 3 同批删 | 删前 TT-src 打 zip；厂商 DLL 不进 TG 发行包 |
| Q5 | 改名分三批 | 对外 / 数据与配置 / 代码内部 |
| Q6 | 「查询数据源」控件先隐藏，字典改造后接通 | 原控件是从未挂载的死代码，已删 |
| Q7 | ~~`tt gp table` 后续废弃，`tt dict` 是唯一入口~~ 已执行：2026-11-11 删掉整个 `tt gp` 组 | 远程直查保留并改写 GP SQL |

## 规范

- commit 标题：`feat(scope):` / `fix(scope):` / `chore(scope):` / `refactor(scope):`
- 计划与任务成对维护：本文件写大纲，`tg-tasks.md` 写勾选清单
- 全部完成后，两个文件名前加 `✓`

## 环境备忘

- Go：`C:\Program Files\Go`，不在 PATH；沙箱下必须
  `$env:PATH="C:\Program Files\Go\bin;$env:PATH"; $env:GOCACHE="D:\Digiwin\tt_work\.gocache"`
- `make` / `bash` 不可用（msys fork 报 `couldn't create signal pipe`）→ 直接 `go` / `node`
- node：`D:\nodejs\node.exe` v24.22.1；`vite build` 需 `danger-full-access` + `TMP/TEMP` 指到工作区
- 后台 `go test` 别在管道末尾接 `Select-Object`（会整段缓冲，看不到进度）
- Git 推送由用户在 cmd 里执行（沙箱连不上 `ssh.github.com:443`）
