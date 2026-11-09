---
name: tt-pj-test
description: 在TT中涉及任何和测试(test)相关的内容都要激活此SKILL
---

# TT 测试

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
| 生成 `./tt.exe` | `make build` | `selftest` / `doctor` 前必须先跑 |
| `.tzc` 合成对抗 | `make selftest` | 31 项；先用 `make build` |
| `.tzc` 全语料 | `make deep-tzc` | 9–11 分钟；需语料**副本** |
| `.tzs` 环境自检 | `make doctor` | 需引擎 exe、设计器、工作区/语料 |
| `.tzs` 全语料 | `make deep-tzs` | 17 分钟；需 `TTZS_CORPUS` 副本 |
| `.tzs` 函数面 | `make fns` | 19 分钟；需 `TTZS_CORPUS` + 引擎 |
| `.tzs` 真引擎 E2E | `make e2e` | 需 `TTZS_EXE` + `TTZS_WS`，部分子用例还要 `TTZS_PKG` |
| 真实 T100 链路 | 见下方“真机” | 没有 Makefile 目标；必须点名环境并获得授权 |
| 看跑了多少/跳了多少 | `make skips` | 会再跑一遍全量并列出跳过原因 |

规则：

- 所有 `go test` 都带 `-count=1`；缓存会让“没跑”看起来像“跑过了”。
- 跳过不等于通过；先看 `make skips` 的跳过原因，再决定是否补环境。
- 新增 `t.Skip` 必须登记到 `internal/testkit/skipledger_test.go`，否则 `make test` 会失败。

## 深档：只用副本

`make deep-tzc` / `deep-tzs` 的语料入口是环境变量，不是仓库里的真实客户目录：

```bash
TDEV_CORPUS="$TEMP/ttws" make deep-tzc      # TDEV_CORPUS 或 TTZS_CORPUS 均可
TTZS_CORPUS="$TEMP/ttws" make deep-tzs
TTZS_CORPUS="$TEMP/ttws" make fns

TTZS_EXE="D:\...\engine\out\tzs-server.exe" \
TTZS_WS="D:\...\workspace" \
make e2e                                    # 需要时再加 TTZS_PKG
```

- `$TEMP/ttws` 必须是真实语料的**副本**；绝不拿客户语料直接跑。
- 覆盖变量指错目录时不会回落到缺省，测试会跳过；先修正路径。
- `.tzs` 深档可选：`TTZS_VALIDATE=sample|all`、`TTZS_CORPUS_LIMIT=N`。
- `TTZS_INSTALL` 可指定设计器目录；不设就用随包设计器。
- 只有改了 `engine/` 才跑 `make engine`；重编会让正在运行的旧守护进程变成孤儿。

## 真机：没有 make 目标

真实 SSH / fgldb / T100 库默认不在 `make test` 里。需要真链路时：

1. 先向用户确认：**用哪个环境**，并明确授权可以连。
2. 环境一律点名，不猜、不扫、不拿别的环境顶替：

```bash
./tt.exe debug probe -m <模块> -p <作业> --env <环境>
./tt.exe debug ents --env <环境> --json
./tt.exe debug sql "SELECT ..." --ent <ENT>
./tt.exe dict r.t <表> --at <环境>@<企业>
```

- `debug probe` 会真的登录并启动作业；这是最重的动作，只在用户点名的环境跑。
- `debug sql` 只放行单条 SELECT/WITH，最多 200 行；只读开关关掉时不要用。
- 会话结束用 `./tt.exe debug quit`；后台服务用 `./tt.exe serve --stop`。
- 真机作业可能留下服务器行锁或客户端窗口；停站过夜有看门狗自动放行（默认 1800 秒）。

## 配置与隔离

- **默认测试离线**，不读 `config.json`，也不连 T100。
- `config.local.json` 只放四个机器路径：`corpusRoot` / `engineExe` / `workspace` / `designerDir`；它不读 `hosts`，不要拿它存环境或口令。
- 真机配置放 `config.json`，或用 `TT_CONFIG` / `--config` 指向外部文件；配置文件已在 `.gitignore` 里，绝不提交、绝不贴出、绝不对任何输出加 `--raw`。
- 本机没有配置时向用户要环境名、主机、账号、区域、库信息；不要猜，不要扫内网，不要试端口。
- 要新增“真机测试”时，写新测试并登记跳过台账；不要给现有测试加一个“有环境就连真机”的隐藏开关。

## 收口

```bash
git status --porcelain        # 不应出现 config.json / config.local.json
make skips                    # 确认哪些没跑，以及为什么
```

真机测试后额外检查：调试会话已 `quit`、后台服务已 `stop`、没有残留作业或行锁。
