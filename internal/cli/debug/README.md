# internal/cli/debug — 调试命令组

`tt debug` 的命令面与后台服务控制。**命令本体都是 HTTP 客户端**：内核在
[../../debug/README.md](../../debug/README.md)，本包只负责参数、寻址与输出。

## 寻址

控制端命令**自动发现**后端地址：从运行状态文件里读它记下的接口前缀与端口。
所以对着"统一服务下的调试面"与"单独起的调试服务"两种装法都能用，调用方不必知道对面是哪种。

## 命令分组（按职责，不按实现文件）

| 组 | 命令 | 说明 |
|---|---|---|
| 服务 | `serve` / `status` | 起（默认后台常驻、单实例）、看状态、停 |
| 生命周期 | `start` / `quit` / `interrupt` / `stop` | 建会话、结束会话、中断运行中的程序、取当前停站现场 |
| 断点 | `bp list` / `bp add` / `bp del` | 结构化断点面：**每条带命中计数**（`hits`）、条件、忽略次数 |
| 标准命令透传 | `exec "<命令>" [更多…]` | 透传协议自带命令并原样返回输出；可一次给多条，省掉每条一次进程启动 |
| 等人还是空转 | `why` / `wait` | 判断程序在等用户操作还是空转；等会话事件（到点返回"超时"而不是错误） |
| 看 | `source` / `logs` / `locate` / `resolve` | 读服务器源码、会话事件、定位函数、把作业编号解析成程序 |
| 人机协同 | `mode` | 切换纯人工 / 协作模式 |
| 环境 | `env` / `topent` / `ents` | 列环境与切换、看与设企业编号、看当前环境有哪些企业及各自账号 |
| 库 | `sql` / `db` | 只读 SQL、库连接与账号映射探查 |
| 报文 | `wslogs` / `wsdebug` | 接口日志检索、按报文重放调试 |
| 自检 | `probe` | 协议驱动器自检（登录 → 启动 → 下断点 → 步进 → 求值） |

**用法、参数与典型流程见** [`skills/tt-debug/SKILL.md`](../../../skills/tt-debug/SKILL.md)；
这里只记契约。

## 两个现场口径（L1 真机后修正）

**`--module` 不再有默认值**（缺陷 #3）：以前 `start`/`probe` 把默认值写死成 `asf`，
它把服务端**登录后按 `<作业>.42r` 在各模块 42r 目录里做的权威解析**整条短路 ——
实测 `debug start axmt500_wf` 于是报出 `asf_axmt500_wf.4gl`，真名是 `axm_axmt500_wf.4gl`，
`break asf_…:行` 必然 `No source file named`。现在：

* 留空（默认）= 让会话自己解析，这是唯一权威来源；
* 显式给出 = 按它拼 `<模块>_<作业>.4gl`（说清它只是"我要这个模块"，不是"我不在乎"）；
* 帮助文本由内核包的 `ModuleFlagHelp()` 提供，两个命令一份文案。

按作业名前缀推模块是**猜**（`axmt500_wf→axm` 对，`bsft001_wf→asf` 就错），所以不做。

**`tt debug source` 失败要可排障**（缺陷 #2）：登录区源码目录是登录后回读的
`$TOP/$ERP/$COM`，取不到时服务端错误已带三条可操作的路；客户端再补一段命令行视角的
"怎么办"（`sourcehint.go`，写 stderr，不污染 stdout 的正文/JSON）。判据只认这一条已知失败，
别的错误不套这套提示。

## 断点面与两条 running/接线纪律（A33 / A34 / A35）

`tt debug bp` 是**结构化**的断点面，不是 `exec "info breakpoints"` 的复读机：

```
断点 3 个（会话 sOnly）
  #1   break     axm_axmt500_wf.4gl:4130 (axm_axmt500_wf.axmt500_ui_dialog)  命中 1 次  启用  条件 1=1  还忽略 2 次
  #2   watch     g_current_row  命中 0 次  启用
  #3   break     axm_axmt500_wf.4gl:1553 (main)  命中 3 次  已禁用  该位置已有断点 #1
```

* **命中计数必须打出来**（A33 / D33）：`hits` 是内核给的唯一命中判据 —— `Address` 恒
  `0x00000000`（命中的断点也是 0），拿它判"装没装上"会得出反向结论。所以人读行里永远有
  `命中 N 次`，`--json` 里 `hits` **键永远存在**（没命中是 `0`，不是缺键）—— 那个键由
  `bpView` 保证，用的不是内核 `Breakpoint` 的 `omitempty`。
* **running 时下断会被明确拒**（A34）：服务端回 `400 {"code":"USAGE","error":"程序运行中,
  禁止发送调试命令(仅允许中断)"}`。客户端**不重试、不静默**，错误里直接给下一步：
  `tt debug interrupt`（这是 running 时**唯一**允许的命令）→ `tt debug wait --for stopped`
  → 再下断。判据只看那句报文原文，别的 400（位置不存在等）不套这套指引。
  **允许的自动化只有 interrupt**，而且本工具**选择不自动中断**：程序常常正停在 GDC 界面等
  用户输入，自动 SIGINT 会把用户正在填的对话框直接取消。**绝不**自动 continue / run ——
  这三条都由 `bp_test.go` 钉住（桩只收到一次 POST，且没有任何 raw/continue/run 请求）。
* **先接线再调试**（A35）：没有 GDC 接线的会话在 `main` 之后**立刻** `Fatal error -6300` 退出，
  守卫行与断点都来不及停。`start`（以及 `wsdebug`）在拿到快照后**当场**把这句话打出来
  （`wiring.go`：判据是快照里的 `fglserver`，那是本轮真正导出的那个值；文案唯一来源是内核的
  `ScreenWarnWith`，本包不另抄一份）。`--json` 时这句走 stderr，stdout 仍是一份完整 JSON。

已知边界（A32）：`ignore` 的**清零不受支持**（报文分不清"没设过"与"刚清零"）——
要改成"不忽略"请 `bp del` + `bp add` 重建，这一点写在 `tt debug bp` 的帮助里。

## 输出

- 控制端命令默认输出**给人看的文本**。
- **唯一例外是只读 SQL**：它默认输出 CSV（`# ` 注释头 + 裸主体），因为那条命令的主要用途是
  进管道；要结构化详情才加 `--json`。
- 失败的形状与退出码见 [../README.md](../README.md)（这条线的码是 0 成功 / 1 失败）。

## 判据

```bash
go test ./internal/cli/debug -count=1
```

覆盖：控制端的寻址与参数、后台服务的启动/停止判定、
`source` 失败时的可操作提示（`sourcehint_test.go`：命中/不命中两半）、
断点面与两条现场纪律（`bp_test.go`：`bp list` 人读与 `--json` 都带命中计数、
running 的 400 必须给 `interrupt` 指引（**反证**：改回裸错误 → 红）、
"不重试 + 绝不自动 continue/run"（桩只收到一次 POST）、无 GDC 时 `start` 输出里有 `-6300`
而本轮有 FGLSERVER 覆盖时不吵）。

## 细节去哪

- 内核（会话、协议、看门狗、运行时产物） → [../../debug/README.md](../../debug/README.md)
- 环境与账号 → [../../host/README.md](../../host/README.md)、[../../entdir/README.md](../../entdir/README.md)
- 怎么用（含人机交接的话术） → [`skills/tt-debug/SKILL.md`](../../../skills/tt-debug/SKILL.md)
