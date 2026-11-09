# internal/cli/session — `tt session` 命令组

会话是**唯一底座**：它持有身份（`ref` / `name` / `n` / `epoch`）与作业实例（环境 / ENT / 作业）。
本组是它的命令面：建、列、看、改 ENT、结束本轮、收掉、透传。

实现在 [`../../debug`](../../debug)（会话与调试器）与 [`../../debugclient`](../../debugclient)
（调试面 REST 的**唯一**客户端）。本组不自己拼 HTTP，也不直接碰 SSH。

## 命令

| 命令 | 作用 |
|---|---|
| `tt session new --env <环境> --ent <ENT> [--name <别名>] [--prog <作业>]` | 建会话：不给 `--prog` 建**空闲**会话；给了同时起作业并等到入口停站 |
| `tt session ls [--env …] [-a]` | 列会话（`-a` 含已断开） |
| `tt session show --session <ref|别名|句柄>` | 看一条：身份、作业、停站现场、附件（画面通道 / 正在录的那一份） |
| `tt session job --session … --prog …` | 在**这条空闲会话**里起一个作业（换作业跑，不用重建会话） |
| `tt session ent <值> --session …` | 改会话级企业编号（仅空闲可设；`--clear` 清除覆盖） |
| `tt session quit --session …` | 结束本轮调试，**会话保留**（回空闲） |
| `tt session rm --session …` | 收掉会话（断连接、清资源） |
| `tt session exec --session … -- "<fgldb 命令>"` | 透传一条 fgldb 命令 |

目标一律 `--session`，接受 `ref`（`环境#序号`）、别名、句柄。多条命中**不替你挑**：
报歧义并列出候选（`SESSION_AMBIGUOUS`）。

## 改代码必须知道的三件事

1. **`quit` 与 `rm` 不是一回事**：`quit` 打在 `DELETE /api/sessions/{id}`（结束本轮、宿主保留，
   下一轮免重新登录）；`rm` 打在 `POST /api/sessions/{id}/close`（收掉会话）。混用的后果是
   "以为收掉了，其实还留着一个开着的会话"。
2. **`new` 的两条路**：不给 `--prog` 走 `/sessions/switch` 建空闲会话；给了走 `POST /sessions`
   起作业。
3. **只读视图**：`show` 只读会话本身的快照；拿不到时**不影响**
   会话本身的输出（服务版本不同也能用）。

## 判据

```bash
go test ./internal/cli/session/ -count=1
```

钉住的几条（`session_test.go`，用假服务记录每次调用）：

1. `new` 不带 `--prog` → 第一个调用必须是 `POST /api/sessions/switch`，且请求体带 `env`/`ent`/`name`；
2. `new --prog` → 先 `POST /api/sessions`，请求体带 `prog`/`ent`/`module`；
3. 建会话 → 先建空闲会话，再等它登录完（loading → idle）；
   一个进程都不起就报错；
4. `quit` 打 `DELETE`、`rm` 打 `POST …/close`（两个动作的端点不许互换）；
5. `exec` 透传的命令原样进 `/raw` 的 `command`；


## 细节去哪

- 会话与调试器内核：[`../../debug/README.md`](../../debug/README.md)
- 调试面客户端（码、候选、信封）：[`../../debugclient`](../../debugclient)

- 命令树与退出码总表：[`../README.md`](../README.md)
