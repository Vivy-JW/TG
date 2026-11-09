# internal/config — 统一配置层与路径管理器

**只有一份配置文件**，所有命令组共用：位置解析、读写、校验、打码都在这一层。
tt 访问的一切自有外部资源（缓存、服务状态、本地字典库、引擎 exe）的落点也在这里统一解析。

## 配置文件放在哪

按优先级取**第一个存在**的：

1. `TT_CONFIG` 环境变量
2. `--config <路径>`
3. `<exe 目录>\.portable` 存在 → 便携包，配置留在包内
4. `%APPDATA%\tt\config.json` —— 默认位置

**没有旧位置兜底，也不做自动迁移。** 旧版的各种落点（`%APPDATA%\T100\tt`、tdebug/tdict、
exe 或当前目录的 `config.json`）一律不读 —— 数据统一在 `%APPDATA%\tt`，不向后兼容。
`tt config validate` 会对 `schemaVersion` 低于当前的配置给出提示（不再自动迁移）。

**显式指定的路径就是答案。** `--config` / `TT_CONFIG` 指向一个还不存在的文件时，**不**悄悄
改用默认落点 —— 否则便携版会把配置写进用户目录，测试脚本也会落在别处。

**数据目录 = 配置所在目录**，所以缓存、快照、状态文件都跟着落在同一个目录下。

## 统一路径管理器（locations.go）

所有"东西放哪 / 去哪读"的问题都从 `Locations` 取答案，调用方不得自己拼这些路径 ——
那会复制出第二份位置解析，某天两边悄悄分叉（历史上"服务写一处、命令行读另一处"就是这样来的）。

```go
loc, err := config.ResolveLocations(configFlag, allowMissing) // 从头解析
loc := config.LocationsAt(configPath)                        // 已有配置路径时包装
```

| 方法 | 给什么 |
|---|---|
| `DataDir()` | 数据目录 = `config.json` 所在目录 |
| `CacheDir(name)` | 数据目录下的缓存子目录；名字必须在 `CacheSubdirs` 清单里，否则空串 |
| `StateFile()` / `LogFile()` | `.tt-serve.json` / `.tt-serve.log`（后台服务状态与日志） |
| `DictDBDir()` / `DictDBPath(env)` | 本地字典库目录与某环境的库文件：`<数据目录>/dict/<环境>.db`（一个环境一个文件） |
| `DictDBCandidatesFor(env, flagPath)` / `ResolveDictDBFor(env, flagPath)` | 某环境本地库的候选与读侧定位：`TDICT_DB` → `-d` → `<数据目录>/dict/<环境>.db` |
| `DictDBCandidates(flagPath)` / `ResolveDictDB(flagPath)` / `SyncTarget(configured, flagPath)` | **旧版单文件口径**，只用于显式 `-d/TDICT_DB` 的兼容路径；新代码走 `DictDB*For(env)` |
| `config.EngineExe(override)` | `.tzs` 引擎 exe：`tzs.serverExe` 覆盖 → `<tt.exe 目录>\tzs\tzs-server.exe` |

两个配套约束：

- **缓存子目录名只允许 `CacheSubdirs` 清单里的**（`ents` / `srccache` / `execlog` /
  `debug-bps` / `spill` / `update`）。凡是 `<数据目录>/<名字>` 这种拼接都走 `config.CacheDir`，
  写错名字当场得到空串，而不是悄悄建出一个清单外的新目录。
- **本地字典库缺省在数据目录下**（便携包的数据目录就是包内，两种形态都成立）。
  读侧（查询命令）要求文件存在，全不存在就报错；写侧（同步）永不报错，由同步过程创建。

## 一份文件、一个写入口

```go
config.Edit(path, validate, mutate)   // Open → validate → mutate → Save
```

- **`Open`** 读成原始键值结构：文件不存在**不是错误**（首次运行就是没有），空文件视为空配置，
  非法 JSON 报错。**未知顶层键一律保留** —— 调用方只该改自己那一节，其余原样带走。
- **`Save`** 是原子写：同目录临时文件 + 落盘 + 重命名，失败不留半截文件，
  读者也看不到只写了一半的内容。配置目录不存在时自动创建。
- `validate` / `mutate` 任一失败都**不落盘**。

**一切修改都应走 `Edit`。** 自己读-改-写的代价不是多几行代码，而是错误语义与"未知键被抹掉"
这类行为在各处不一致。

## 结构版本与缺省值

`SchemaVersion = 4`，随新配置文件写入（骨架与 `ensureSchemaVersion`）；旧结构不做自动迁移。
以下两版是"加可选节"：3 加了 `net`，4 加了 `skills`（第三方技能登记表）。

各节都有自己的缺省常量（监听地址、调试参数、终端宽高、打印上限、SSH 端口、工作区后缀…），
由 `FillDefaults` 在内存里补齐 —— **不写盘**。缺省值只在一处定义。

## 打码

`RedactSecrets` / `RedactValue` **按键名判断**（`password` / `passwd` / `pwd` 一类），
不看标记、也不看它在哪一节。这样**未知节里的未知键**同样会被遮住 —— 那是这类泄漏最容易发生
的地方。所有"把配置吐出来"的输出（`config show` / `config get` / 数据库列表 / 服务的
`GET /api/hosts`）共用这一份口径。

**`GET /api/hosts` 返回明文口令**（设置页要能就地编辑与探测）—— 任何 dump 或粘贴这个响应的
行为都是泄密，包括贴进聊天窗口与 issue。

## 哪些目录算缓存

`CacheSubdirs` 是**唯一定义**：CLI、服务、启动清理都从这一份拿。这样不会出现某处漏清、
或某处手滑把配置一起删掉。

明确**不是**缓存的三个：

| 不列入的 | 为什么 |
|---|---|
| 配置文件本身 | 用户数据（口令、环境、路径），删了就没了 |
| 服务状态文件 | **运行中**服务的进程记录，删了 `--stop` 就找不着它 |
| 服务日志 | 排障要看；想清可以用系统手段 |

`CleanCache` 有两种模式：**启动清理**只删"修改时间早于阈值"的文件；
**用户手动清缓存**整个子目录删掉。它只碰清单里列出的目录。

## 路径型取值

配置里那些"这是个路径"的值（`sync.target`、`mirror.dir`、`dict.dir`、`hosts.sshs[].workspace`…），
解析规则同样只有一份（绝对化、派生状态、同步目标缺省位置）。
理由：同一个值在两处各算一遍，会解析成不同结果 —— 那是最难查的一类 bug。

`skills.externals[].path` 是唯一**解析规则不在这里**的路径值：它由
[../skill](../skill/README.md) 自己绝对化并使用（可能指向一个今天没插的移动硬盘，
所以**存下来时不做存在性校验** —— 那是安装时才判的事，且只告警）。配置层只负责
原样读写与保持顺序（顺序即优先级）。

## 判据

```bash
go test ./internal/config -count=1
```

覆盖：位置解析的优先级与"显式路径就是答案"、**旧位置兜底已移除**（当前目录与旧工具目录里
的诱饵配置不被捡走）、原子写、打码口径（含未知节）、缓存清单的边界、路径管理器的各条
优先级链（字典库读/写、引擎 exe、状态文件）。

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 位置解析的优先级 | 便携版与安装版的落点都靠它；改错会让配置"消失"（其实是写到了别处） |
| `Locations` 的各条优先级链 | 字典库读写、服务状态、引擎定位——调用方已经不再自己算，改这里=改所有地方 |
| `Edit` / `Save` 的语义 | 全仓所有写配置的路径；原子写退化成直接写会重新引入"半截文件" |
| 缺省值常量 | 各命令组的默认行为（调试参数、终端尺寸…） |
| `CacheSubdirs` | 清理逻辑与 `CacheDir` 校验的唯一依据；**多列一项可能删掉用户数据** |
| 打码规则 | 一切把配置吐出来的地方 |

## 细节去哪

- 环境与账号那条链 → [../host/README.md](../host/README.md)、[../entdir/README.md](../entdir/README.md)
- 配置在界面上怎么被读写（校验在服务端） → [../web/README.md](../web/README.md)
- 各节字段的完整说明 → 根 [README.md](../../README.md) 与 [config.example.json](../../config.example.json)
