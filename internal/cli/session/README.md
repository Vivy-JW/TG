# internal/cli/session — `tg session` 命令组

会话是**唯一底座**：它持有身份（`ref` / `name` / `n` / `epoch`）与作业实例（环境 / 作业）。
本组是它的命令面：建、列、看、结束本轮、收掉、透传。

身份只有**环境**：TG 只服务 TOP GP，账号由服务端从 `db.accounts[0]` 取，没有企业编号
（ENT）这一层。所以本组没有 `--ent` / `--fglserver`，也没有 `session ent` 子命令；
连给脚本看的 JSON 里也不再摆 `ent`（见下）。

实现在 [`../../debug`](../../debug)（会话与调试器）与 [`../../debugclient`](../../debugclient)
（调试面 REST 的**唯一**客户端）。本组不自己拼 HTTP，也不直接碰 SSH。

## 与 `tg debug` 的分工

`session` 是**名词**（把台子搭起来），`debug` 是**动词**（上台之后怎么演）：

| | `tg session` | `tg debug` |
|---|---|---|
| 关心 | 会话在不在、连哪台机器、挂哪个作业 | 程序停在哪、下一步往哪走 |
| 典型 | `new` / `ls` / `show` / `quit` / `rm` / `exec` | `bp` / `step` / `print` / `where` / `wait` |
| 频率 | 一次调试里调用几次 | 一次调试里调用几十次 |

两个门、同一个屋子：共用 `debugclient`、同一套 `--session` 寻址（`ref` / 别名 / 句柄）、
同一批 REST 端点，行为不会各说各话。`debug env/start/quit/exec` 与 session 有重叠，
保留是因为老脚本、skills 与 Web 工作台还在用，不在这轮删除。

## 命令

| 命令 | 作用 |
|---|---|
| `tg session new --env <环境> [--name <别名>] [--prog <作业>]` | 建会话：不给 `--prog` 建**空闲**会话；给了同时起作业并等到入口停站 |
| `tg session ls [--env …] [-a]` | 列会话（`-a` 含已断开） |
| `tg session show --session <ref\|别名\|句柄>` | 看一条：身份、作业、停站现场 |
| `tg session job --session … --prog …` | 在**这条空闲会话**里起一个作业（换作业跑，不用重建会话） |
| `tg session quit --session …` | 结束本轮调试，**会话保留**（回空闲） |
| `tg session rm --session …` | 收掉会话（断连接、清资源） |
| `tg session exec --session … -- "<fgldb 命令>"` | 透传一条 fgldb 命令 |

目标一律 `--session`，接受 `ref`（`环境#序号`）、别名、句柄。多条命中**不替你挑**：
报歧义并列出候选（`SESSION_AMBIGUOUS`）。

`--module` 留空时按 GP 的规则解析：本地字典 `t_prog` / `t_prog_link` → 线上 `gal_file`
（8 秒超时）→ 回写 `t_prog_link` → 最后才按作业编号前缀兜底。

## 改代码必须知道的四件事

1. **`quit` 与 `rm` 不是一回事**：`quit` 打在 `DELETE /api/sessions/{id}`（结束本轮、宿主保留，
   下一轮免重新登录）；`rm` 打在 `POST /api/sessions/{id}/close`（收掉会话）。混用的后果是
   "以为收掉了，其实还留着一个开着的会话"。
2. **`new` 的两条路**：不给 `--prog` 走 `/sessions/switch` 建空闲会话；给了走 `POST /sessions`
   起作业。
3. **JSON 里没有 `ent`**：`briefMap` 不输出该键；`show` 的 JSON 在 CLI 这层投影，滤掉服务端
   快照里的 `ent` / `topent` / `topentSession` / `topentShell`（`gpSnapshot`）。服务端字段
   不动 —— 那是 Web 接口日志页的既有契约。
4. **只读视图**：`show` 只读会话本身的快照；拿不到时**不影响**
   会话本身的输出（服务版本不同也能用）。

## 判据

```bash
go test ./internal/cli/session/ -count=1
```

钉住的几条（`session_test.go`，用假服务记录每次调用）：

1. `new` 不带 `--prog` → 第一个调用必须是 `POST /api/sessions/switch`，且请求体带 `env`/`name`、不带 `ent`；
2. `new --prog` → 先 `POST /api/sessions`，请求体带 `prog`/`module`、不带 `ent`；
3. 建会话 → 先建空闲会话，再等它登录完（loading → idle）；
   一个进程都不起就报错；
4. `quit` 打 `DELETE`、`rm` 打 `POST …/close`（两个动作的端点不许互换）；
5. `exec` 透传的命令原样进 `/raw` 的 `command`；
6. `ls` / `show` 的 JSON 不含 `ent` 与三个 `topent*` 键（GP 只认环境）。

## 细节去哪

- 会话与调试器内核：[`../../debug/README.md`](../../debug/README.md)
- 调试面客户端（码、候选、信封）：[`../../debugclient`](../../debugclient)
- 命令树与退出码总表：[`../README.md`](../README.md)
