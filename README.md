# TG（TOP GP 开发 CLI）

TG 是用于开发鼎捷数智旗下 TOP GP（Genero 4GL）ERP 的命令行工具集：读代码、查字典、调作业，
外加一个本地网页调试工作台。只读——不改服务器上的文件、不写生产库。

---

## 安装

参考 https://github.com/Vivy-JW/TG/releases

- 便携包

    ```powershell
    # 把 tg 安装到用户 PATH
    tg install path
    ```

- MSI 安装包

## 升级

```powershell
tg update check     # 查有没有新版（不下载不安装）
tg update           # 查 + 下载校验，问一句再装
tg update log       # 上一次升级走到哪一步
```

`tg update` 按你现在的安装形态装：便携包就地覆盖程序目录（`config.json` 与目录里你自己的文件不碰），
MSI 交给 `msiexec`（用户级安装，不需要管理员）。升级全程需要能访问 GitHub；内网要经代理时用
`tg update --proxy http://host:port`，或把地址写进 `config.json` 的 `net.proxy`。

升级会把装过的技能树一并刷成同版（`tg install skills` 装过哪些地方记在数据目录的 `.tg-skills.json`）。
那一步没成时，`tg version` 会提示手动跑 `tg install skills --force`。

Agent / 脚本里注意：默认会问一句再装，非交互环境要显式 `tg update --yes`；
先看它打算做什么可以 `tg update --dry-run`。

## 初次配置

```
tg serve
```

> 启动后在本地打开配置网页 <http://127.0.0.1:28680>
>
> 至少需要配置一个 TOP GP 环境（可 SSH 的 Linux 服务器 + 库账号）

加完环境，网页左树点进去就能建会话、起作业、下断点。

## 技能（skills）

执行 `tg install skills` 会把 `tg` 官方技能和登记在配置里的第三方技能安装到指定目录，命令执行完毕
在目标目录下创建 `.agents/` 并把技能放进去。

内置三份：`tg-erp-architecture`（产品架构与目录/命名）、`tg-erp-debug`（调试与排错）、
`tg-erp-read`（读代码：结构、开窗、报错、作業說明書）。

## TG SKILL HUB

如果需要贡献 skill，请遵循下面的格式，将你的 skill 仓库链接、摘要和描述通过 PR 的形式推送到本文档本章节。

|name&link|abstract|description|
|---|---|---|
| | | |

## 随包手册

发版包在 `tg.exe` 旁边带两份 Markdown 手册，查原文用，不嵌进二进制：

```powershell
tg install bdl      # Genero BDL 官方手册（4GL 语法、画面、SQL、fgldb）
tg install gpdoc    # TOP GP 技术手册（环境、命名、单档双档、客制规范）
tg docs             # 两份手册解到哪了
```

落点默认在数据目录，设置页「应用设置」里能改；`--force` 只覆盖同名文件，不会删你另外放的东西。

---

<details>
<summary>开发与构建</summary>

需要 Go 1.26+；打便携包还要 Node.js 与 pnpm（前端会被编译进二进制）。

```cmd
make build          # 编 tg.exe
make test           # go test ./...
build_portable.bat  # 便携包 → dist\tg-portable\ 与 dist\tg-portable.zip
build_msi.bat       # MSI → dist\TG-<版本>-x64.msi（需 WiX v3）
```

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
