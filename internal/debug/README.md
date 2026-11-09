# internal/debug — 调试内核

T100 作业调试的核心：经 SSH 驱动服务器上的 4GL 调试协议，管理会话，对外提供 HTTP 面。
命令行的 `tt debug …` 只是它的**薄客户端**（见 [../cli/debug/README.md](../cli/debug/README.md)）。

## 协议

服务器上跑的是 `fglrun -d` 的文本调试协议：**纯文本、只有提示符、没有区间结构**，
所以驱动侧靠一组锚点正则识别"断点命中 / 中断 / 步进头 / 错误"。那些正则来自真机实测输出，
改动它们等于改协议解析。

终端层的东西（ANSI 剥离、提示符判定、行解析）在 [../host/README.md](../host/README.md)，
与源码镜像等功能共用。

## 会话模型

| 角色 | 职责 |
|---|---|
| **管理器** | 持有会话池、向订阅者广播事件、保留最近若干条事件供迟到的订阅者补发 |
| **会话** | 一台状态机：启动、运行、停止；维护"命令-应答配对" |

**会话事件是外部观察程序状态变化的唯一途径**：界面与命令行都从同一条事件流取数，
只是消费方式不同（一个长连接、一个短调用）。

**建会话有两条路，别用错**：

| 端点 | 语义 | 池里已有同环境的会话时 |
|---|---|---|
| `POST /api/sessions/switch` | "**连到这个环境**"（`tt debug env`） | 复用那一条；名字不同且它空闲 → **收掉旧的按新名重建** |
| `POST /api/sessions/new` | "**再给我一条**"（环境页的「创建会话」） | **一条都不动**；同名（同环境）则原样返回它（幂等），否则新建 |

多会话（几个 Agent 同时用、同一环境并发调试）只能用第二条 —— 用 `switch` 建第二条会把
上一条静默收掉，而调用方看到的是"新的建好了"。

### 会话身份：`id` / `ref` / `display` / `n` / `name` / `epoch`

会话列表（`GET /api/sessions`，也就是 `tt debug status` 的第二份文档）里每个元素是
`SessionBrief`。它的键集是契约定死的一份：**十一个恒定键**（`id` / `ref` / `display` /
`name` / `env` / `ent` / `n` / `kind` / `prog` / `state` / `epoch`）加一批按 `omitempty`
出现的快照键 —— 恒定键**取不到就给空串/零值，不省键**。

三样是"这条会话是哪条"，生命周期内**一个字符都不变**：

| 键 | 是什么 | 谁分配、什么时候 |
|---|---|---|
| `id` | 服务端原始句柄（`s<毫秒>`） | 建会话时；REST 路径 `{id}` 只认它 |
| `n` | 序号：**同一环境下**的第 N 条，从 1 起 | 登记时按"尚未被占用的最小正整数"分配，之后不重算 |
| `ref` | 规范 id `环境#序号` | 由 `环境` + `n` 拼出来（`n` 不变它就一定不变） |

**企业编号不参与序号**：按（环境, 企业编号）自增会让企业 99 的 `#1` 与企业 88 的 `#1`
都叫 `恒烁正式区#1`，同一个地址指向两条会话。`ent` 是可变量（`tt debug topent 88` 会改它），
所以它只当**属性**、不进地址 —— 改掉 `ent` 之后 `ref` 与 `n` 必须原样不变。

`display`（`环境/作业#序号`）比 `ref` 多一段**当前作业快照**：作业中途换掉它就跟着变。
**要恒定用 `ref`，要好读用 `display`**，两者不混用（拿这两个去拼 REST 路径都是错的）。

**`epoch` 的语义**（本批定稿，实现见 `nextSessionEpoch`）：它是"这条会话**这一代宿主 /
这一轮运行**"的令牌，取值是进程内单调且**全进程唯一**的 base36 计数。唯一性是判据的一部分：
令牌不重复 ⇒ 复用/重启之后**不可能**凑巧等于调用方手上的旧值，"旧句柄当场失效"就不需要
任何时钟比较。换代的三个时机：

1. **会话建立**（登记时发第一代）；
2. **同一宿主跑新一轮**（`SetRun`：`tt debug start` 复用空闲宿主 / 按日志重放复用宿主）；
3. **会话重启/重新登录**（`/api/sessions/{id}/restart`：旧会话收掉、新会话登上来）。

调用方把上一次拿到的 `session.epoch` 原样带回来，与服务端当前值不等就是上一代 →
`SESSION_STALE` 退 3（判据在 `internal/target` 的 `Resolve`，服务端只负责**产出**它）。
**注意别与 `Manager.Epoch` 混**：那是**事件流游标**的进程级世代（前端靠它知道服务重启了、
`Seq` 要归零），与会话的世代是两件事。

### 别名（`--name`）的生命周期

别名是会话的一个属性，不是独立登记的长期资源：

- **只在创建时给一次**（生命周期内不变）；
- **随会话消失即释放**：会话池里没有这条会话 ⇒ 别名立刻可以给别人用。判据锚在
  "**池子里还有没有它**"（`SweepExited` 把已断开的残留清出去），所以 **SSH 掉线、
  进程被杀、服务重启**与"人敲了 quit"走**同一条**判据 —— 这正是"会话已死但别名仍占着、
  重起同名被拒"那类极难查的故障的根治点；
- **当幂等键**：同名会话空闲 → 复用它（同一条会话，不是第二条）；正在跑 / 已停站 →
  报 `SESSION_TARGET_BUSY` 退 3；名字不存在 → 起新的。

## 人机协同：模式与写闸门

一个会话有一个**模式**：纯人工或协作（协作者看操作来源判定）。写操作过一道闸门：

- **协作模式下只有 AI 能写**；纯人工模式下 AI 只能读。
- 闸门是**会话级的静态标志，不做逐命令协商**。原因很实际：服务端看不见 AI 的"一轮"
  （命令行每次调用都是独立进程），要做逐命令的租约/续期就得发明一套服务端看不见的时间轴。
- **拒绝文案要能直接照做**：不是"没权限"，而是告诉对方去点界面上的哪个按钮、或把操作委托给谁。

## 两个专有问题

**一、阻塞在交互语句 vs 空转。** 程序跑到等人输入的语句时会停在界面上，从调试器看**与死循环
无法区分**。所以有一条专门的探测动作来回答"它在等人还是在空转"（判据是停站行是不是交互语句）。
这也是人机交接的判据来源。

**二、不能让程序停在断点上过夜。** 停站会占着库里的行锁；有一条看门狗在停站停留超时后
自动放行（默认半小时，可配）。

## HTTP 面

一个处理器对外给出全部接口，统一服务把它挂在 `/debug/` 前缀下：

| 组 | 端点 |
|---|---|
| 服务 | 状态、优雅停止、事件流（查询） |
| 会话 | 建立（可顺带起作业）/ **再建一条**（`/new`）/ 列表 / 快照 / 关闭 / 重启 / 切换 / 企业编号 / 模式 |
| 画面通道 | 查询 / 起（`POST …/screen`）/ 停 —— 实现由接线层注入，调试线不认识 GDC 与代理 |
| 断点 | 增 / 删 / 列 |
| 控制与观察 | 控制（继续/步进等）、求值、取栈、裸命令、**为何停住** |
| 数据 | 局部与全局变量、源码与函数列表、自动变量、调用栈帧、定位 |
| 源码 | 读服务器源码 / 预览（只读白名单） |
| 报文 | 接口日志查询与内容、按报文重放调试、服务测试 |
| 库 | 只读 SQL |
| 实时 | 一条 WebSocket（**纯推送**，带重放缓冲） |

接口前缀的存在是为了与共享层的同名路径区分开（共享层也有 `/api/status`、`/api/hosts` 等）。

## 运行时产物

| 产物 | 位置与规则 |
|---|---|
| **源码镜像** | 数据目录下按环境隔离。**只活一轮调试**：每轮启动前清空 —— 服务器上的源码会改，留着上一轮的文件只会让 AI 拿着过期代码推理。权威性始终在"每次真读服务器"那一侧 |
| **断点持久化** | 按模块与程序一份；恢复时**按源码行文本重新定位**，源码变更不会把断点恢复到错误位置 |
| **接口日志** | 查服务器上的报文日志表；列集对齐 T100 原生页。可按日志里内嵌的报文重跑服务程序（重放调试） |
| **企业目录快照** | 见下 |

## 断点选项：条件 / 忽略次数 / 监视点（A20 结构 · **A26 口径**）

`POST /api/sessions/{id}/breakpoints` 除 `location` 外还收三个字段，**可以都不给**（省略 = 旧行为）：

| 字段 | 含义 | 生成的 fgldb 命令（**A26**） | 拒绝条件 |
|---|---|---|---|
| `cond` | 条件表达式（空 = 无条件） | **`break <loc> if <expr>`**（条件内联进 break，GDB 的 "stop only if %s"） | 含 `\n`/`\r`、以 `end` 结尾、**> 512 字节**；`kind=watch` 时**一律拒**（见下） |
| `ignore` | 忽略前 N 次命中（0 = 不忽略） | `ignore <n> <count>` | 负数；**0 ⇒ 不发这条命令**（其它照发） |
| `kind` | `break`（默认）/ `watch` | `watch <location>` | `location` 含**白名单外字符**（只接受标识符/表达式） |

命令顺序固定：`break`（带 `if`）/ `watch` → `ignore`（见 `BreakpointOpts.Commands`）。

**为什么不是 `condition <n> <expr>`**（A20 的原映射）——主机109 真机实测
（`../.agents/plan/L1-结果-2026-10-18.md` §5.1）：
`tt debug exec "condition 1 g_current_sw = 1"` → `Undefined command: "condition". Try "help".`，
`help` 的命令表里也没有它；本地 `libfglvm` 里 `condition` 的**唯一**命中是 DAP 的 JSON 字段名。
⇒ 内核是 GDB 的**子集**，条件只能写在 break 指令里。

**`watch` + `cond` ⇒ 明确报「不支持条件监视点」**：`watch X if Y` 认不认，一次都没验过；
不拿"大概能行"的猜法发命令（失败会从"下断点失败"推迟成"命中时机不对"，更难查）。

**拒绝路径为什么保证"不发字节"**：校验（`Validate`）在生成命令之前，不过就直接返错，
`Commands`/`BreakCmd` 返回 nil/空串 —— 调用方按"没有命令"处理。这是**结构保证**，不是自觉；
`hBPAdd` 也在取会话之后、发命令之前先校验一次，返回 400 并点名是哪个字段、为什么。
**刻意不做**"先拼再 sanitize"：把危险字符偷偷删掉会静默改变用户表达式的语义，比拒绝坏得多。

存档（`debug-bps/*.json`）的 `cond` / `ignore` / `kind` 三个键**全部 `omitempty`**：

* 旧存档缺这三个键 ⇒ 读成零值（`""` / `0` / `""`），照常可用；
* 没设的字段不会多出 `"cond":""` 之类的噪声键；
* 恢复时按新字段重建（监视点的"位置"是存档 `func` 里那个被监视表达式，没有 file:line 可重定位）；
  存档里的选项若不合法，**降级为纯位置断点**并记一条日志 —— 不因一个条件表达式丢掉整条断点。
  **例外**：`kind=watch` 且带 `cond`（本机不支持条件监视点）时只丢**条件**、保住监视点
  —— 降级成"纯位置断点"会连监视点本身一起丢掉（它的 location 是表达式，不是 file:line）。

## `info breakpoints` 的真实行格式（A27 + A30 + A33）

主机109 实测原文（脱敏）：

```
Num Type           Disp Enb Address    What
1 watchpoint keep y  g_current_row
2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141
	ignore next 1 hits
```

条件断点（A30）下同一张表**多一行**续行：

```
2 breakpoint keep y 0x00000000 in axm_axmt500_wf.axmt500_ui_dialog at axm_axmt500_wf.4gl:2141
	stop only if 1=1
	ignore next 2 hits
```

**命中过的断点多第三行**（A33，第四次真机；未命中的断点**不打印**这一行）：

```
2 breakpoint keep y 0x00000000 in main at axm_axmt500_wf.4gl:1553
	breakpoint already hit 1 time
```

| 事实 | 后果 |
|---|---|
| **watchpoint 行没有 `hw`**、Address 列为空 | 解析**不依赖** `hw`（带 `hw` 的老格式仍容忍）；监视点没有 file:line，表达式落在 `Breakpoint.Func` |
| breakpoint 行 `0x00000000 in <模块> at <文件>:<行>` | `Module` / `File` / `Line` 三段分别取出 |
| **`Address` 恒 `0x00000000`**（A33：**命中的那条也是 0**） | 它**不是**"装没装上/生不生效"的判据，解析只是跳过它；**命中与否只看 `Hits` 与停站事件** |
| `ignore next N hits` 是**独立缩进续行** | 解析必须能跨行，并进上一条断点的 `Ignore` |
| 条件断点多一行 `stop only if <expr>`（A30，A26 `break … if <expr>` 的回声） | 解析必须并进上一条断点的 `Cond` |
| 命中的断点多一行 `breakpoint already hit N time(s)`（A33，单复数都认） | 解析必须并进上一条断点的 `Hits`（未命中 = 0，**不是**解析失败） |
| **三种续行顺序不定** | 都按"并进上一条"处理，不按固定 if-else 链切；空条件的 `stop only if` 不算续行（掉进"行格式不认识"） |

落点：纯函数 `ParseInfoBreakpoints`（`parser.go`，不 IO、可全覆盖），
`syncBreakpoints`（重建账本）与泵里的 `pendingCmd.settleBPs`（`exec info breakpoints` 类响应）
都走它 —— 一处格式知识，两个入口。

**`Hits` 的账本兼容（A33 新增字段）**：`Breakpoint.Hits` 带 `omitempty`，且**不进**
`StoredBP`（`bpsstore.go` 的 JSON 契约一字未动）—— 它是 fgldb 的**实时**事实，不是要在磁盘上
记住的东西。所以老存档（没有 `hits` 键）照旧读、照旧恢复，零迁移；新报文里 0 命中也不落键。

**解析不出来 ⇒ 报「行格式不认识」并保留原文**：`syncBreakpoints` **不拿空表覆盖账本**
（一次格式意外会把用户手上的断点全清掉），错误进事件日志；流式路径把错误放进 `execResult.Err`。
静默吞掉是最坏的处理：一张空断点表 + 没有任何线索，等于让人从零开始猜。

**`Cond` 与 `Ignore` 都以 fgldb 回显为准**（两种续行都能读回来，A30）：`syncBreakpoints`
重建账本时旧账本**只补空缺**，判据全在纯函数 `inheritMissingBPFields`（`session.go`，
不碰会话、可全覆盖 —— 见 `bpmerge_cond_test.go`）：

| 字段 | 判据 |
|---|---|
| `Cond` | 解析到非空 ⇒ 用解析到的；解析到空（老引擎/老会话不回显）⇒ 才用旧账本补 |
| `Ignore` | 同上：解析到非 0 优先，`0` 才用旧账本补 |

修前 `Cond` 是**无条件**覆盖（`nbp.Cond = obp.Cond`）—— A30 之后 fgldb 真会回显
`stop only if 1=1`，那一行等于把刚读到的条件换成旧值，或者（旧账本没条件时）抹成空串。
匹配上的两条细节：老存档的 `Kind` 是空串（= break）⇒ 比较前归一，否则从旧存档恢复的账本
**永远**补不到；监视点的 `File/Line` 是空的（位置就是表达式）⇒ 还要比 `Func`，否则两个
监视点会互相串 `ignore`；同一位置在旧账本里有多条时按 **Num 最小**取（与 map 遍历顺序无关，
同一份报文两次同步得到同一账本）。

**残余歧义（已知，未改）**：解析到 `Ignore == 0` 既可能是"没有 `ignore next N hits` 续行"，
也可能是"用户刚 `ignore N 0` 清零"—— 两者在报文上无法区分，所以清零后可能被旧账本的值补回来。
要精确只能让解析器带一个"这次回显过没有"的标记（`Breakpoint` 加字段 = 账本结构变更）。

## `bp list` 的实时性：账本 vs 内核（A36）

**现场（2026-10-18 真机）**：`bp add` 正常、`info breakpoints` 原文里有 `breakpoint already hit 1 time`、
点击后**确实停站**（`停站 …:4130 … reason=breakpoint`）—— 可 `bp list` 前后都写 `命中 0 次`。

**根因（代码已复核）**：`hBPList` 直接交出账本（`sess.Breakpoints()`），一个字节都不重读内核；
而回灌只在"改断点"那刻发生（`/raw` 里 `IsBreakpointCmd(cmd)` 为真时 `syncBreakpoints()`）
⇒ `命中 N 次` 表达的是**上次改断点时的计数**，不是当下。

| 状态 | `GET …/breakpoints` 的行为 | 响应里的标记 |
|---|---|---|
| `stopped` | **先** `syncBreakpoints()`（用内核当下的 `already hit N time(s)` / `ignore next N hits` 覆盖账本）**再**返回 | `fresh: true`，不带 `note` |
| `running` | **一个字节都不发**（A34：运行期任何调试命令都被 400 拒，只允许 interrupt） | `fresh: false` + `note`（含"可能不是最新"与"先 `tt debug interrupt`"） |
| `idle` / `loading` / `exit` | 同上（没停站现场，发不出去） | `fresh: false` + `note` |
| 停站但内核读不回来（格式不认识/被拒） | 账本**保持原样**（不拿空表覆盖），如实说不新鲜 | `fresh: false` + `note`（带错误原文） |

响应形状是**纯新增键**：`ok` / `breakpoints` 一字不改，多出 `fresh`（恒出现）与 `state`，
`note` 只在 `fresh=false` 时出现 —— 调用方（CLI / 前端 / AI）据此**知道自己在看什么**，
而不是把一份旧账本当内核真相。

**读命令也回灌（A36 后半）**：`exec "info breakpoints"` 走的是 `/raw`（`kind="other"`），
`onLine` 那条按 kind 分派的解析不会进，所以它读到的命中计数原本只留在响应文本里。
现在 `hRaw` 在命令成功后按 `IsBreakpointReadCmd(cmd)` **再回灌一次**（仍然只在停站态）。

**为什么是"单独判断"而不是把 `info` 塞进 `bpCmds`**（二选一，选前者）：

* `bpCmds` 的语义是「会**改动**断点集合」，`info breakpoints` 一个字节都不改 —— 塞进去等于让
  读命令冒充写命令：`/raw` 的 `act` 会变成 `bp.info`，事件流里平白多出一条"改断点"的动作；
* 单独判断的影响面只有 `hRaw` 一个分支，`info locals` / `info variables` 读了也不会去动断点账本；
* 既有判据 `TestIsBreakpointCmd` 明确断言 `info breakpoints` **不是**断点类命令（既有行为不破）。

**测试口径**（`bplist_live_a36_test.go`）：假内核（注入 `Session.execFn`）+ HTTP 桩，不连机器、
不起会话。四条主线：停站先回灌（且**先**于返回）、运行态零命令且带说明、读命令后账本被刷新、
add / del / enable 之后 hits 仍与内核一致。两条反证各钉一次：去掉停站态回灌 ⇒ 第一条红；
让运行态也去 sync ⇒ 第二条红（它真的会发出 `info breakpoints`）。

**`hits: 0` 的 JSON 形状没动**：服务端 `Breakpoint.Hits` 带 `omitempty`（A33 的账本兼容口径），
所以 0 命中时服务端不落这个键；"`--json` 的 `hits` 键永远在、0 不省键"是 CLI 侧
`bpView`（`internal/cli/debug/bp.go`，无 `omitempty`）的保证，钉在 `bp_test.go`。

## `asf_` 从哪来 / 断点位置的模块前缀（缺陷 #3）

现场：`debug start axmt500_wf --module asf` 报的停站文件是 **`asf_axmt500_wf.4gl`**，
而 `info breakpoints` 里的真名是 **`axm_axmt500_wf.4gl`** ⇒ `break asf_…:行` 必然
`No source file named`。

**根因（读代码得出，不是猜）**：`Session.Module` 本来是**登录后**由 `resolveModule`
（`session.go`，按 `<作业>.42r` 在各模块 42r 目录里搜）解析出来的 —— 这是唯一权威来源；
但 CLI 的 `--module` 默认值被写死成 `"asf"`，把权威解析整条短路了，文件名于是由
`<模块>_<作业>.4gl` 这个拼接规则编出来。

**修法**：

1. `--module` 默认改成**空**（`debug start` 与 `debug probe` 都是），帮助文本由
   `ModuleFlagHelp()` 统一给：留空 = 登录后自动解析；显式给出则按它拼名；
2. `BPLocContext.ResolveLocation` 用**会话已知事实**修正用户写错的前缀
   （`resolveModule` 的结果 / 客制模块 / 本会话作业名）：只改"本会话作业"的文件名，
   公共库（`lib_*.4gl`）与别人的文件一概不碰；
3. 事实不足（模块还没解析出来）**不改位置**，而是把"怎么给"写进 `Breakpoint.Note`：
   `info breakpoints` 看真名 / `debug stop` 看停站 `file` 字段 / 直接 `break <行号>`。

**按作业名前缀推模块是猜，不许**：`axmt500_wf → axm`（实测）但 `bsft001_wf → asf`
（同台机器的另一支作业，`asf_bsft001_wf.4gl`）—— 同一套"前三字母"规则立刻给出错答案。
唯一能回答的是 42r 目录，所以答案只能来自登录后的解析。

## 登录区源码目录未知（`tt debug source`，缺陷 #2）

`ReadSourceStandalone` 的白名单根目录是 `Config.ModuleRootsActual()` =
**登录后回读**的 `$TOP/$ERP/$COM`（`$ERP/<模块>/4gl` 与 `$COM/{lib,sub,qry}/42m`）。
静态配置里没有、也不许有 T100 路径，所以取不到时**不回退默认目录**
（那会把"读错机器上的文件"伪装成成功），而是由 `sourceRootsUnknownErr` 把三件事说清：

1. 怎么让它有值：`tt debug start <作业> --env <环境名>`（会话登录后把 `$TOP/$ERP/$COM` 写进该环境 Runtime）；
2. Runtime 是**进程内**状态：换环境/区域后要在同一个 serve 实例里重新落一次（start 或 switch）；
3. 已经有值却仍报：`tt debug env` 看该环境到底有没有 TOP/ERP/COM（有值⇒问的是另一个环境段；没值⇒选区后没回读到）。

CLI 侧再补一段命令视角的下一步（`internal/cli/debug/sourcehint.go`，stdout 不污染）。

## 企业目录的取值顺序

```
进程内缓存 → 落盘快照 → 现查（SSH + 读映射表）
```

快照的存在理由：这个问题会被反复问，而每问一次都要 SSH 握手 + 探工具路径 + 连库查。
**代价是可能过期，所以新鲜度必须如实带出去**（来源、是否过期、抓取时间），让调用方自己决定信不信。
快照是**可丢弃的缓存，不是数据源** —— 读坏了、写不出一律当没有，绝不因此报错。

## 只读 SQL

给 AI 查一眼业务数据用。**三层防线，每层都要如实说明它挡住什么、挡不住什么**：

| 层 | 挡 |
|---|---|
| 文本白名单 | 手写的写操作、多语句、客户端命令、受限包 |
| 库侧只读事务 | 白名单没看出来的常规写 |
| 资源与输出上限 | "读太多"把库和上下文一起打爆 |

**挡住不的**：自治事务、函数副作用，以及"只读 ≠ 只读该企业的数据"。

**账号必须由企业编号解析**：企业编号与库账号强关联，上错号查不到数据 ——
而"查不到"会被读成"这条业务数据不存在"，那是最危险的错误结论。所以绝不硬编码账号，
解析不到就直接拒绝，并把"企业 → 账号"回显在结果里。

## 配置

环境的类型与连接由 [../host](../host/README.md) 承载；本包在它之上追加调试专属参数，
并把三段配置（环境清单 / 调试设置 / 监听地址）读成一个配置对象。
**T100 的路径不允许静态配置** —— 登录后按区域动态获取，失败即报错。

**画面接线（FGLSERVER）只由单轮决定**：配置里的 `debug.fglserver` 是默认值，启动请求可按轮覆盖
（`tt debug start --fglserver <ip:port>`；空串 = 本轮交回 T100 按 SSH 来源 IP 自动设置），
覆盖不写回配置，也不改变会话复用判据。它是"界面去哪"的唯一开关，而失败是**静默**的 ——
放行之后什么都不弹、启动段也不报错，所以每轮启动都在事件流与快照里报一句画面去向
（`ScreenTarget`：端口 = 前端监听端口 - 6400）。

**没接线就不只是"看不到画面"，是程序根本活不过 `main`**（A35，第四次真机三次复现）：
自动接线而本机 7400/6400 上没有 GDC 在听时会话在 `main` 之后**立刻**
`Fatal error -6300 Can not connect to GUI` 退出（`main:1553` 之后的守卫行一次都没停）。
所以 `ScreenWarn` 必须在启动段就说这句话、并给出动作（起 GDC / `--fglserver`）；
文案的唯一来源是 `ScreenWarnWith(fglserver, gdcPresent)` —— 探针结果由调用方给，
因为 CLI 要在 `debug start` 那一刻也打同一句（见 [../cli/debug/README.md](../cli/debug/README.md) 的 A35 段）。

## 起作业的路径

会话起的作业是 `fglrun -d <LaunchRef>.42r <LaunchArgs>`（`session.go:668`）：`LaunchRef` 取自
T100 元数据（gzza004 的启动引用，如 `$ASFi/bsft001_wf`），`LaunchArgs` 是配置模板
（默认 `config.DefaultLaunchArgs`，`{prog}` 换成作业名）。这条路径**直接跑目标程序**，
不经过 T100 自己的启动器（`r.r` 的 `azzp000`、`r.d` 的 `gendbg`/`fgldeb`）。

运行期分配的 AUI id 是**启动历史的函数**：三条路径互不相同。

## 判据

```bash
go test ./internal/debug -count=1
```

**覆盖**：协议行分类与锚点、会话状态机与命令-应答配对、看门狗、企业目录取值顺序、
只读 SQL 的防线、配置合并；**画面接线的单轮覆盖**（`api_test.go`：覆盖值不写回内存配置、
空串 = 本轮交回自动设置、与环境点名各归各的，以及 `ScreenTarget` 的措辞）；
**四个"动作"端点的错误半边**（`api_action_test.go`：
请求体坏了 → 400、拨号失败 → 500 信封、入参校验**在拨号之前**那处顺序性质；
从前是五个 —— 那个"服务测试"端点（`/api/wstest`）本批已从产品里取消）；
**断点存档的落盘与读取**（`bpsstore_test.go`：路径怎么拼、`dataDir` 为空时不落盘也不报错、
`Read` 的每一条拒绝理由）；**断点选项的命令生成与拒绝**（`breakpointopts_test.go` +
`breakpointopts_a26_test.go`：`break … if …` / `ignore` / `watch` 逐字与顺序、非法条件各自报错且
**不产生命令**、`ignore 0` 不发命令、`watch+cond` 明确拒、**生成器里不许出现 `condition`**、
存档新字段往返与**旧存档零值**）；**A27/A30/A33 行格式**（`bpinfo_parse_test.go` +
`bpinfo_stoponly_test.go` + `bphits_parse_test.go`：L1 原文两种形态、`watchpoint` 无 `hw`、
`ignore` 第二行并入、`stop only if` 续行并入 `Cond`、`breakpoint already hit N time(s)` 并入
`Hits`（单复数、无该行 = 0、三种续行顺序互换同一份账本、计数不串到别的断点、
孤儿续行/不认识的行报错并保留原文）；**反证**：拿掉 `already hit` 续行 ⇒ 第一条红）；**账本重建的补空缺判据**（`bpmerge_cond_test.go`：解析到的 `Cond` 优先、
解析到空才用旧账本补（**反证**：改回无条件覆盖 → 红）、`Ignore` 同向、老存档空 `Kind` 能对上、
监视点按表达式区分、同位置多条按 Num 最小且与 map 顺序无关）；
**断点位置的前缀修正**（`bplocation_test.go`：`asf_` 前缀按会话已知模块改正、
前缀本来就对时一字不改、事实不足时不猜并给出怎么给）；
**登录区源码目录未知的文案**（`sourcehint_test.go`：可操作、点名环境、不出现默认路径）。

**故意不覆盖**（两处，都是结构上难测，不是漏了）：

- **`session.go` 的主流程**：要真 SSH PTY + 真 fgldb 交互。那里 19 处 `time.Now()`、
  多处硬编码 `time.Sleep` —— 断言只能落在"最终态"，而最终态要真引擎。
  要测得先造一个 fgldb 协议模拟器，那是**一次依赖反转**；而买家是一个已被 87 个用例
  与真机自检覆盖的包的**内部行覆盖**。同样的理由让"把 debug 拆成子包"那件事也被否掉了。
- **`api.go` 的快乐路径**：要 `*host.SSHConn`，而它的 `cli` 是私有字段（`host/ssh.go:18`），
  从外部伪造得实现 8 个方法、其中三个返回不可构造的类型。所以 `Server` 上开了个 `dial`
  字段（生产默认 `host.Dial`，行为一字不差），测试只注入**必然失败**的实现 ——
  买到的是错误半边，不是快乐路径。

**一处缺口，2026-09-27 已修**：未注册的 `/api/*` 路径曾返回 **200 + HTML** —— 路由里那条
`mux.HandleFunc("/", s.hStatic)` 的 SPA 兜底把任何未注册路径都吞掉，调用方打错一个 API
路径时拿到的是**看起来成功**的 HTML 页（前端构建过时是一张真的 index.html，更看不出异样）。
现在 `hStatic` 对 `/api` 或 `/api/` 前缀回 **404 + JSON 信封**，与 `internal/web`
（`hRoot` 对非 `/` 路径回 404）和 `internal/dict/server`（`TestSharedEndpointsNotServedHere`）
两侧的规矩对齐。

判据两条，都在 `api_action_test.go`：`TestUnregisteredAPIPathsReturnJSON404` 覆盖
**前端构建 / 未构建两种状态**；反向的 `TestStaticFallbackStillServesSPA` 守住
"收窄兜底不能顺手把 SPA 打掉"（`/apifoo`、`/api2/x` 这两个边界证明判据是
`/api` 或 `/api/` 前缀，不是松的 `HasPrefix(p, "/api")`）。

**405 是刻意没做的**：`GET /api/dbsql`（方法不对）现在得 404 而不是 405。路由里有
catch-all 时 Go 的 mux 不会自己产生 405，要拿它得手写一张注册表；而 404 与 405 对调用方
是同一件事——"这个路径上你没有可用的接口"。理由也写在 `hStatic` 的注释里。

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 协议锚点正则 | 整个调试功能；它们来自真机输出，改错会让"停站"识别不出来 |
| 模式与写闸门 | 人机协同的边界；文案是给人看的操作指引，不是错误码 |
| 看门狗默认值 | 生产区行锁的持有时间 |
| 源码镜像的清理时机 | AI 读到的代码是否过期 |
| HTTP 面 | 前端与命令行的控制端都依赖；前缀语义要与统一服务一致 |
| 只读 SQL 的防线 | 直连客户库的安全性 |

## 细节去哪

- 命令行控制端与用法 → [../cli/debug/README.md](../cli/debug/README.md)、[`skills/tt-debug/SKILL.md`](../../skills/tt-debug/SKILL.md)
- 服务器连接与登录后的环境 → [../host/README.md](../host/README.md)
- 结果输出的形状 → [../output/README.md](../output/README.md)
- 服务怎么挂它 → [../web/README.md](../web/README.md)
