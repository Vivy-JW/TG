# 第三方组件与文档声明

本文件列出 TG 中包含的第三方代码与随包分发的第三方文档，以及它们的版权归属。
TG 自身以 [Apache License 2.0](LICENSE) 授权；下列内容**不在**该授权范围内，
各自按其权利人声明的条款使用。

## 一、随包分发的文档

| 文件 | 内容 | 版权归属 | 说明 |
|---|---|---|---|
| `BDL-Markdown.zip` | Genero BDL User Guide（19 章 Markdown，约 5,449 个文件） | FourJs / Actian（Genero 产品文档） | 随安装包分发，供 `tg install bdl` 解到本机阅读 |
| `TOP-GP-Markdown.zip` | TOP GP 技术手册（13 章 Markdown，237 个 .md） | 鼎捷数智（Digiwork） | 经权利人许可随包分发，供 `tg install gpdoc` 解出 |

这两份文档只随安装包一起分发，不参与 TG 的 Apache-2.0 授权。
使用者如不希望使用随包版本，可用 `tg install bdl --from <自己的 zip>` /
`tg install gpdoc --from <自己的 zip>` 换成自有副本，或用 `--to` 改落点。

## 二、代码

本项目包含源自 [TT](https://github.com/luyaogod/TT) 的代码（以 TT 0.5.2 的
`2fd193b` 为基线，此后针对 TOP GP 大量改写与删除）。TT 未声明开源许可证，
该部分代码经其作者许可在本项目中继续使用与分发。

## 三、Go 依赖

直接依赖如下（完整清单与版本见 `go.mod` / `go.sum`）。每个模块按其自身仓库中的
LICENSE 授权，常见为 MIT / BSD-2-Clause / BSD-3-Clause / Apache-2.0。

| 模块 | 用途 |
|---|---|
| `github.com/spf13/cobra`、`github.com/spf13/pflag` | 命令行框架与参数解析 |
| `github.com/coder/websocket` | Web 调试工作台的 WebSocket |
| `github.com/pkg/sftp` | SSH/SFTP 传输（文件读取、镜像拉取） |
| `github.com/sijms/go-ora/v2` | Oracle 直连 |
| `github.com/jackc/pgx/v5` | 金仓/PostgreSQL 直连 |
| `golang.org/x/crypto`、`golang.org/x/sys`、`golang.org/x/text` | SSH、系统调用、Big5/GBK 转码 |
| `modernc.org/sqlite` 及其间接依赖 | 本地字典库（纯 Go SQLite） |
| `github.com/dustin/go-humanize`、`github.com/google/uuid`、`github.com/inconshreveable/mousetrap`、`github.com/kr/fs`、`github.com/jackc/*` 等 | 间接依赖 |

构建产物（`tg.exe`）静态链接了上述模块，其许可证声明随各自模块分发；
如需完整的许可证文本清单，可在仓库内执行 `go mod vendor` 后查看 `vendor/` 下各模块的 LICENSE。
