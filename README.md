# TG（TOP GP 开发 CLI）

面向鼎捷数智 TOP GP（Genero 4GL）ERP 的命令行工具集。

> 本项目在架构上参考了 [TT](https://github.com/luyaogod/TT)，面向 TOP GP 重新实现（T100 分支已整体移除）。
> 当前版本 **0.1.0** —— TG 是新工具，版本号从 0.1.0 起算，不沿用 TT 的 0.5.x（变更见 [CHANGELOG.md](CHANGELOG.md)）。
> 可执行文件 `tg.exe`、安装目录 `%LOCALAPPDATA%\Programs\TG`、命令前缀 `tg`、数据目录 `%APPDATA%\tg`。

---

## 当前状态

只面向 **鼎捷 TOP GP（Genero 4GL）**，T100 的内容已删干净：

- ✅ 站点管理：按环境读营运据点（`azp_file`），据点编号与 schema 分开
- ✅ 数据字典：表/字段/索引/约束/程序/消息/画面文字，可同步成本地 SQLite 离线查
- ✅ 开窗定义：读 `gab_file`+`gac_file`，按 `cl_create_qry` 的规则现拼 SQL
- ✅ 代码阅读：`tg outline`、`tg file`、`tg interface`、`tg dict prog --links`、`tg sdd`（作業說明書）
- ✅ 调试：`tg session` / `tg debug`（`r.d2` + `fgldb`），以及 `tg serve` 起的 Web 调试工作台
- ❌ 已移除：T100 的 ENT 企业编号、`.42s`、`.tzc`/`.tzs` 设计器管线、设计器导入环境、T100 字典命令

## 从源码构建

需要 Go 1.26+；构建便携包时还需要 Node.js 与 pnpm（前端会被编译进二进制）。

```cmd
build_portable.bat
```

## 使用

当前可用的主要命令：

```cmd
tg env list --format table     # 列出配置里的环境（TOP GP）
tg site list --env 22主机      # 该环境的营运据点与 schema
tg dict r.t ima_file --at 22主机     # 读表定义（本地字典没有就直查）
tg dict r.q cp_ima --at 22主机       # 开窗定义 + 现拼出来的 SQL
tg outline abmi100.4gl --at 22主机 --human   # 4GL 大纲
tg sdd axmt410 --at 22主机           # 作業說明書（有镜像读镜像，没有就直读服务器）
tg debug start abmi600 --env 22主机  # 起作业并停在入口
tg session ls                  # 看当前调试会话

tg serve              # 启动本地配置页 / 调试工作台（http://127.0.0.1:28680）
tg install path       # 把 tg 加入用户 PATH
tg install skills     # 把内置技能安装到指定目录
tg update check       # 检查是否有新版本（不下载）
tg version            # 查看当前版本
```

## 目录结构

```
TG/
├─ main.go              程序入口
├─ internal/            Go 后端全部实现（cli / config / web / host / output / update …）
├─ web/                 前端（React + vite，构建产物被 Go 嵌入）
├─ installer/           MSI 安装包定义（WiX v3）
├─ skills/              对外技能手册
└─ testdata/            测试语料（4GL 大纲夹具）
```

## 关于许可证

上游 TT 仓库**未声明开源许可证**（无 `LICENSE` 文件），因此本仓库不沿用任何开源许可条款，
目前仅作个人开发使用。若后续获得上游授权，或 GP 部分已完全重写，再决定采用哪种许可证。

## 致谢

- [TT](https://github.com/luyaogod/TT) — 本项目的架构参考与代码基线
