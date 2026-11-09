# internal — 包职责地图

55 个包。这份文档回答三个问题：**每个包只该管什么**、**它在第几层**、**加新东西时该放哪**。

层号来自 `go list` 实测的依赖图，不是设计意图。跑一遍：

```bash
go list -f '{{.ImportPath}}|{{join .Imports " "}}' ./internal/... | grep 'tt/internal'
```

## 一、分层：8 层，零环

**层号 = 它到最底层的依赖链长度，不代表重要性。** 规矩只有一条：上层可以 import 下层，
下层绝不 import 上层。层号越小越该稳定——改第 0 层等于改所有东西的地基。

### L0 · 基座（16 个）：没有内部依赖，谁都能用，它谁也不依赖

可独立测试、可整体替换。

| 包 | 只该管这些 |
|---|---|
| `dbconfig` | 一次数据库连接该有哪些字段（类型 / 主机 / 端口 / 服务名或库名 / 账号清单）。纯类型，无逻辑 |
| `output` | **唯一输出出口**：JSON 信封、CSV（带环境头）、人读表格。谁都不许自己拼 JSON |
| `safesql` | 只读 SQL 的文本防线：白名单、禁多语句、禁 DML/DDL |
| `pathinstall` | 用户 PATH（HKCU）的增删。**永不碰系统 PATH** |
| `winproc` | 起一个不随本进程死的子进程 / 判它活着 / 杀掉 |
| `sshtun` | SSH 端口转发隧道（本地监听 → 远端 host:port） |
| `session` | 会话身份规则：序号怎么分、世代号怎么发。71 行纯规则 |
| `target` | 会话寻址的目标层：层级定义、选择器解析、契约表、失败信封。全是纯函数——不发请求、不写盘 |
| `form42f` | `.42f` 编译表单 → 字段表 + 页面布局。纯函数，零 IO |
| `str42s` | `.42s` 编译字符串表 → 中文标签。纯函数，零 IO |
| `dict/db` | 本地 SQLite 字典查询 + `Source` 接口 + ACC 元模型 |
| `dev/fgl` | 4GL 源码结构：分块、语义 |
| `dev/model` | tzc 领域模型 Region / Document / EnvContext + 行号定位 |
| `dev/tapfile` | TAP 文件的 CDATA 感知、字节保真改写 |
| `dev/tglfile` | `.tgl` 框架骨架里的标记：区段边界、插入点、锚点 |
| `testenv` | 本机测试环境的配置（`config.local.json` + 环境变量） |

### L1 · 文件与格式（7 个）

| 包 | 只该管这些 |
|---|---|
| `config` | **配置文件的唯一读写层**：位置解析、`Open`/`Save`/`Edit`、原子写、打码、缓存清单、缓存清理、字典库落点。只认「节 + 键」，**不认业务字段** |
| `dev/fence` | 围栏协议：Document ⇄ 带围栏的文本，字节级写回 |
| `dev/pkgfile` | `.tzc` / `.tzs` 包格式的只读视图 |
| `dev/testutil` | 合成测试包夹具（不依赖真语料） |
| `erpdb` | 客户端直连库封装：Oracle / 金仓，一个接口两个实现 |
| `screen` | `.42f` 布局 + `.42s` 标签 → 前端画面模型 |

### L2 · 共享服务（13 个）

| 包 | 只该管这些 |
|---|---|
| `host` | **唯一 SSH / 远程服务器能力层**：连接、登录、服务器侧只读探测、源码镜像引擎、服务器侧库探测 |
| `entdir` | 企业目录（ENT → 账号）的共享件：映射表、落盘快照、环境指纹、新鲜期 |
| `designer` | 从设计器 `sites.xml` 只读导入环境站点 |
| `update` | tt 自身更新：查新版、下载校验、交棒给脱离进程；外加「技能树一致性」检查 |
| `testkit` | 跨包测试辅助：语料发现、stdout 捕获、flag 重置、仓库本体定位 |
| `dict/dbsync` | 远程字典表 → 本地 SQLite 同步 |
| `dict/live` | 远程 ERP 库（Oracle / 金仓）直查，实现 `dict/db` 的 `Source` 接口 |
| `dev/store` | 工作区落盘：manifest、基线、原子写、锁 |
| `dev/split` | Document' → TapOp + TglPatch 的写回计划 |
| `dev/synth` | Package → Document 合成，复刻设计器 `LoadContent` 的行为 |
| `dev/tzs` | `.tzs` 引擎（外部 C# exe）客户端：管道协议、manifest、doctor、守护进程生命周期 |
| `cli/common` | CLI 命令组共享的上下文：全局开关、MetaProvider、WebFS。叶子包，用来防成环 |
| `cli/dev/common` | tzc / tzs 两条线共用脚手架：错误信封、稳定紧凑 JSON、参数重排、身份后缀 |

### L3 · 能力内核（9 个）

| 包 | 只该管这些 |
|---|---|
| `debug` | 调试子系统：会话、断点、执行日志、企业快照、源码镜像；外加 HTTP+WS 服务（`api.go`）与只读 SQL |
| `dict/server` | 字典侧 REST：源码镜像拉取、字典同步 |
| `dev/verify` | 三道闸门 + 不变量 I1–I15 |
| `web` | 统一本地服务的 HTTP 层：REST/WS 路由、嵌 SPA、设置页后端 |
| `cli/file` | 服务器文件命令组：只读看/取文件 + 离线镜像管理 |
| `cli/interfacecmd` | `tt interface`：42f + 42s → 画面 |
| `cli/outline` | `tt outline`：4GL → 块树 |
| `cli/dev/tzs` | `tt dev tzs` 的动词（经引擎读写表单） |

### L4–L7 · 命令层（9 个）

| 包 | 只该管这些 |
|---|---|
| `cli/at` | `--at` 文法、作用域表、`--at-list` |
| `debugclient` | 调试面 REST 的**唯一客户端** |
| `cli/dev/tzc` | `tt dev tzc` 八个动词 + uuid 生成 + 31 项对抗自检 |
| `cli/dict` | `tt dict` 命令组：表 / 字段 / 校验 / 开窗 / 消息 / 参数 / ACC 查询 + 字典库运维 |
| `cli/db` | `tt db`：本机 ping / 服务器 probe / 只读 SQL |
| `cli/dev` | `tt dev` 分发（tzc / tzs 两条线） |
| `cli/session` | `tt session` 命令组 |
| `cli/debug` | `tt debug` 命令组 |
| `cli` | 根命令 + 全局开关 + 自身管理命令（env / config / cache / serve / install / update / version） |

## 二、三条硬规矩

**规矩 1 · 只朝一个方向依赖。** 上层 import 下层，下层绝不 import 上层。判据：
一条改动如果让下层 import 了上层，那件事就放错了地方，不是"加一行 import"的事。

**规矩 2 · 命令层只做三件事。** 解析参数、调下层、把结果交给 `output`。
命令层里出现 SQL、出现 SSH、出现"重算一遍引擎的结论"，就是越界。

**规矩 3 · L0–L2 不许出现四样东西。**

| 不许 | 因为 |
|---|---|
| `net/http`、`net` | 网络归 L3 的 `debug` / `dict/server` / `web` |
| `spf13/cobra` | 命令框架归命令层 |
| `internal/cli/*`、`internal/web` | 那是上层 |
| 进程 / 窗口 / 抓图 | `winproc`（进程）；窗口与抓图目前没有归属包（见第五节） |

检查命令（手工版，自动版待加）：

```bash
go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./internal/... \
  | grep -E '^tt/internal/(config|dbconfig|output|host|entdir|safesql|target|session|screen|form42f|str42s|dict/db|dev/(model|fgl|fence|pkgfile|store|split|synth|verify|tzs|tapfile|tglfile|testutil)):' \
  | grep -E 'net/http|cobra|internal/cli|internal/web'
# 无输出 = 合格
```

## 三、加新东西时该放哪

| 你要加 | 放哪 |
|---|---|
| 新 CLI 动词 | `cli/<组>/`，同时在 `cli/<组>/register.go` 挂一行 |
| 「查一下 / 算一下」的能力 | 对应子系统包（`dict/db`、`host`、`erpdb`…），**不放 cli** |
| 新输出格式 | `output` |
| 新配置节 | `config` 的 `Section` + 由该节的主人去建强类型模型 |
| 新 HTTP 端点 | 能力的 `*/server` 或 `*/api.go`；`web` 里只挂一行 |
| 新的 4GL / `.tzc` / `.tzs` 格式细节 | `dev/` 下对应的那一层 |
| 测试夹具 | `testkit` 或 `dev/testutil` |

## 四、谁管什么（唯一性清单）

这四件事**各只有一个归属**，别在别处再写一份：

| 东西 | 归谁 | 存在哪 |
|---|---|---|
| SSH 环境（主机 / 用户 / 口令 / 区域 / 企业编号） | `host` + 配置的 `hosts` 节 | `config.json` |
| 数据库连接（类型 / 地址 / 账号清单） | `dbconfig` | `config.json` |
| **用哪个账号** | 解析出来的，不是配的（`entdir`） | 运行时快照 |
| 输出形状 | `output` | —— |
| 缓存目录清单 | `config`（唯一定义处） | `config.json` 旁边 |

## 五、已知偏差（待清理）

下面 4 条是**当前与上表不符**的地方，不是设计意图。修完就删掉对应行，全部修完就删掉这一节。

| # | 位置 | 不符在哪 |
|---|---|---|
| 1 | `cli/db` / `cli/at` | 为了拿「环境 X 的连接」而构造整个调试运行时视图（`debug.Config`），而 `config.Hosts.Resolve` 直接就能回答 |
| 2 | `debug` | 扛着企业清单获取（`ents.go`）和只读 SQL 执行（`db.go`、`dbsql.go`，后者还挂在调试会话管理器 `Manager` 上）。这两件与"调试"无关 |
| 3 | `host` / `debug` / `cli/db` | 数据库探测与连接的能力散在三处，三份都在做「连一次库验证账号」 |
| 4 | `cli/db` / `cli/file` | 住在命令层，却被别的命令组当库用（`cli/debug → cli/db`、`cli/dict → cli/file`）。它们是能力，名字挂在了命令层 |

## 六、四条安全 / 数据规矩（与分层无关，别顺手改）

1. **SSH 主机密钥当前不校验**（与"手工登录内网机器"同一信任级别）。要改这条等于改安全模型，须单独讨论。
2. **口令一律不进命令行**（客户端直连路径）。服务器侧金仓是唯一例外，代码注释里如实标注。
3. **往配置目录写文件时，要么登记进缓存清单，要么说清它为什么不是缓存** —— 那个目录里住着唯一删了不能自愈的文件。
4. **拼远端命令时值必须过白名单或引号；SQL 一律走标准输入**，不拼进命令串。

## 七、加新东西时的自查

- **环境从哪来？** 走配置层的解析入口，不要自己写「名字 → 活动环境 → 首条」。
- **账号从哪来？** 走企业目录解析，不要自己取账号列表第一项。
- **输出怎么走？** 走 `output`，不要自建 JSON/CSV 发射器。
- **写配置？** 走 `config.Edit` 这个唯一写入口。
- **新增位置参数？** 负整数一类的值会被开关解析吃掉，要给出可操作的提示。
