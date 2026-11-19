# TG — TOP GP 开发工具箱

把 TOP GP（Genero 4GL）的**读代码、查字典、调作业**从"开 ERP 一层层点、上服务器 vim/grep"搬到
命令行 + 一个本地网页。不用装 Genero Studio，不碰生产库。

## 30 秒上手

1. 下载 `tg-portable.zip` 解压到任意目录（或用 MSI 安装）
2. 命令行跑 `tg serve`
3. 浏览器打开 <http://127.0.0.1:28680> → 「设置 → 站点管理」加一个环境（服务器地址 + 库账号）

加完环境，左树点进去就能建会话、起作业、下断点。

## 能干什么

- **读代码**：`tg outline` 看 4GL 结构与函数、`tg file grep` 在服务器源码里搜、`tg sdd` 直接读作業說明書。
- **查字典**：表/字段/索引/约束/程序/消息可同步到本地离线查，`tg dict r.q` 按 `gab_file`+`gac_file` 把开窗 SQL 现拼出来。
- **排错调试**：`tg debug` 起作业、停在入口、下断点、步过、看变量，与网页调试工作台同一套会话。
- **查环境与库**：`tg site list` 看营运据点与对应 schema，`tg db sql` 跑只读 SQL，`tg file mirror` 把服务器源码镜像到本地。

常用几条：

```cmd
tg env list --format table                   # 配置里的环境
tg site list --env 22主机                    # 该环境的据点与 schema
tg dict r.t ima_file --at 22主机             # 表定义
tg dict r.q cp_ima --at 22主机               # 开窗定义 + 现拼的 SQL
tg sdd axmt410 --at 22主机                   # 作業說明書
tg debug start abmi600 --env 22主机          # 起作业，停在入口
tg session ls                                # 当前调试会话
```

## 安装

- **便携包**：解压即用，配置优先读包内 `config.json`（不存在就就地新建）。
- **MSI**：装到 `%LOCALAPPDATA%\Programs\TG`，配置在 `%APPDATA%\tg`。

随包还有两份 Markdown 手册（查原文用，不嵌进二进制）：

```cmd
tg install bdl        # Genero BDL 官方手册（4GL 语法、画面、SQL、fgldb）
tg install gpdoc      # TOP GP 技术手册（环境、命名、单档双档、客制规范）
tg docs               # 两份手册解到哪了
```

设置页「应用设置」里能看生效目录、也能改落点；`tg install … --force` 只覆盖同名文件，不会删你另外放的东西。

## 它不做什么

- **只读**：不改服务器上的文件，不写生产库（SQL 走只读闸门）。
- **只面向 TOP GP**：T100 那条线（ENT 企业编号、`.42s`、`.tzc`/`.tzs` 设计器）已整体移除。
- **需要能 SSH 到 GP 服务器**；目前以 Windows 为主，Linux/macOS 未验证。

## 文档与技能

- [docs/gp-dict-wiring.md](docs/gp-dict-wiring.md) — 字典线的数据源与对端接线
- [docs/gp-scope.md](docs/gp-scope.md) — 范围与取舍（为什么没有 ENT / 42s / 设计器）
- [docs/tg-plan.md](docs/tg-plan.md)、[docs/tg-tasks.md](docs/tg-tasks.md) — 改造计划与逐项状态
- [skills/](skills/) — 三份技能（架构 / 调试 / 阅读），可 `tg install skills` 装给 agent 用
- [CHANGELOG.md](CHANGELOG.md) — 从 0.1.0 起的变更

<details>
<summary>开发与构建</summary>

需要 Go 1.26+；打便携包还要 Node.js 与 pnpm（前端会被编译进二进制）。

```cmd
make build          # 编 tg.exe
make test           # go test ./...
build_portable.bat  # 便携包 → dist\tg-portable\ 与 dist\tg-portable.zip
build_msi.bat       # MSI → dist\TG-<版本>-x64.msi（需 WiX v3）
```

目录结构：

```
TG/
├─ main.go                程序入口
├─ internal/              Go 后端（cli / config / web / host / output / update …）
├─ web/                   前端（React + vite，构建产物被 Go 嵌入）
├─ installer/             MSI 安装包定义（WiX v3）
├─ BDL-Markdown.zip       Genero BDL 手册（tg install bdl 解）
├─ TOP-GP-Markdown.zip    TOP GP 手册（tg install gpdoc 解）
├─ skills/                对外技能手册
└─ testdata/              测试语料（4GL 大纲夹具）
```

</details>

## 致谢与许可

- 架构参考 [TT](https://github.com/luyaogod/TT)，面向 TOP GP 重新实现；T100 分支已整体移除。
- 上游 TT 仓库未声明开源许可证，因此本仓库目前不沿用任何开源许可条款，仅作个人开发使用。
