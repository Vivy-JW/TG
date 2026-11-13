---
name: tg-pj-test
description: 在 TG 中涉及任何和测试（test）相关的内容都要激活此 SKILL。触发词：测试、test、用例、Makefile、跑测试、真机验证。
---

# TG 测试

> **命名说明**：项目已更名为 TG，本文命令前缀写作 `tg`。但代码里的模块名与二进制名尚未改名，
> **在代码改名提交之前，实际执行请把 `tg` 换成 `tt`**（当前产物仍是 `tg.exe`）。

## 关于Makefile

`Makefile`在本项目中是集中管理test命令的入口也是一份文档，能通过`Makefile`管理的测试命令尽可能封装到Makefile中

## 先按改动选档位

| 改动 | 命令 | 前提与代价 |
|---|---|---|
| 单个 Go 包 | `go test ./internal/<pkg> -count=1` | 最快；只验这个包 |
| 单条用例 | `go test ./internal/<pkg> -run <用例名> -count=1 -v` | 钉一个行为 |
| 全仓离线测试 | `make test` | 约 1 分钟；不连 T100 |
| 前端逻辑 | `make check-web` | 需要 `web/node_modules`；不做 tsc |
| 前端类型 + 构建 | `make web-build` | 含 tsc；会重写 `web/dist` |
| 生成 `./tg.exe` | `make build` | 打包与手工实测前先跑 |
| 真实 T100 链路 | 见下方"真机" | 没有 Makefile 目标；必须点名环境并获得授权 |
| 看跑了多少/跳了多少 | `make skips` | 会再跑一遍全量并列出跳过原因 |

规则：

- 所有 `go test` 都带 `-count=1`；缓存会让"没跑"看起来像"跑过了"。
- 跳过不等于通过；先看 `make skips` 的跳过原因，再决定是否补环境。
- 新增 `t.Skip` 必须登记到 `internal/testkit/skipledger_test.go`，否则 `make test` 会失败。

> `.tzc` / `.tzs` 那几条深档目标（`selftest` / `deep-tzc` / `doctor` / `deep-tzs` / `fns` / `e2e`）
> 已随 `tt dev` 与 `engine/` 在 2026-11-11/12 删除，`make` 里不再有它们。

## 真机：没有 make 目标

真实 SSH / fgldb / T100 库默认不在 `make test` 里。需要真链路时：

1. 先向用户确认：**用哪个环境**，并明确授权可以连。
2. 环境一律点名，不猜、不扫、不拿别的环境顶替：

```bash
./tg.exe debug probe -m <模块> -p <作业> --env <环境>
./tg.exe debug ents --env <环境> --json
./tg.exe debug sql "SELECT ..." --at <环境>@<企业>
./tg.exe dict r.t <表> --at <环境>@<据点编号>
./tg.exe site list --env <环境>
```

- `debug probe` 会真的登录并启动作业；这是最重的动作，只在用户点名的环境跑。
- `debug sql` 只放行单条 SELECT/WITH，最多 200 行；只读开关关掉时不要用。
- GP 环境的 `--at` 右侧是**据点编号**（`azp_file.azp01`，大小写不敏感，`schema` 也认），不是数字企业号。
- 会话结束用 `./tg.exe debug quit`；后台服务用 `./tg.exe serve --stop`。
- 真机作业可能留下服务器行锁或客户端窗口；停站过夜有看门狗自动放行（默认 1800 秒）。

## 配置与隔离

- **默认测试离线**，不读 `config.json`，也不连 T100。
- `config.local.json` 只放机器路径（现在只剩 `corpusRoot` 还在用：Go 侧夹具语料的根）；它不读 `hosts`，不要拿它存环境或口令。
- 真机配置放 `config.json`，或用 `TT_CONFIG` / `--config` 指向外部文件；配置文件已在 `.gitignore` 里，绝不提交、绝不贴出、绝不对任何输出加 `--raw`。
- 本机没有配置时向用户要环境名、主机、账号、区域、库信息；不要猜，不要扫内网，不要试端口。
- 要新增"真机测试"时，写新测试并登记跳过台账；不要给现有测试加一个"有环境就连真机"的隐藏开关。

## 收口

```bash
git status --porcelain        # 不应出现 config.json / config.local.json
make skips                    # 确认哪些没跑，以及为什么
```

真机测试后额外检查：调试会话已 `quit`、后台服务已 `stop`、没有残留作业或行锁。
