# internal/cli/common — 命令组共享的 CLI 上下文

三个命令组（`debug` / `dev` / `dict`）共用的东西：全局开关、内嵌前端、配置路径解析、
统一输出约定。

**它是叶子包**：只 import [config](../../config/README.md) 与 [output](../../output/README.md)，
不 import 任何命令包 —— 所以三个命令组都能安全依赖它，不会与根命令成环。

## 全局开关

由根命令的持久开关绑定，**各命令组只读**：`--config`、`--json`、`--csv`、`--format`、`-v`、
`--env`（`--conn` 是它的别名，仍然接受）。

输出形态的解析规则：`--json` / `--csv` 是 `--format` 的语法糖；**缺省是 JSON**
（表格类格式对模型的理解力实测更低，而 JSON 又省掉解析歧义）；`table` / `text` 给人读。

## 注入点

| 注入点 | 谁写 | 谁读 | 用途 |
|---|---|---|---|
| `WebFS` | 根命令（值来自 `main.go` 的内嵌产物） | `WebFrontend()` → `tt serve` | 前端产物；**取不到返回 nil**，调用方回落到引导页而不是 panic（前端没构建是正常状态） |
| `Version` | 构建时由链接器注入 | 版本横幅、服务 | 版本号 |
| `MetaProvider` | `cli/dict` | `CurrentMeta()` → 根命令的错误出口 | 出错时补上"连的是哪个环境"—— 那比错误本身更难猜 |

## 两个容易踩的约定

**`UnknownSubcommand` 是必需的，不是可选的。** 不设 `RunE` 的组命令会让 cobra 打一份帮助
然后**退 0** —— 而 `tt dict` 自己的契约写着"1 = 用法错"。把"我打错了命令"读成成功，是调用方
（尤其 AI）最难自己发现的失败形态：它会以为活干完了。所以每个组都要挂上这个兜底，
它把错误写成"未知子命令 X（组名）；可用：…"，并列出全部子命令名。

**`PrintJSON` 不是查询命令的输出出口。** 查询要走 `output.Emit` —— 裸值没有地方放
"这次查的是哪个环境、哪个账号"。

## 判据

```bash
go test ./internal/cli/common -count=1
```

**覆盖**：`OutputFormat` 那 11 种开关合流（`--json` / `--csv` 压过 `--format`，
认不出的形态落回 json 而不是报错）、`CurrentMeta` 没注入 provider 时返回零值、
`ConfigHint` / `ResolveConfig` 的**接线**（取值依赖环境，所以钉转发而不是钉具体值）、
`PrintJSON` 的输出形态、`UnknownSubcommand` 要列出可用子命令。
`resetCommon(t)` 自己也有断言 —— 它是本包所有用例的地基，坏了会让别处的失败长得很怪。

**故意不覆盖**：`Fatal` —— 它 `os.Exit`，一调就带走整个测试进程。

## 细节去哪

- 环境与账号那条链、输出契约 → [../../host/README.md](../../host/README.md)、[../../output/README.md](../../output/README.md)
- 根命令怎么装配 → [../README.md](../README.md)
