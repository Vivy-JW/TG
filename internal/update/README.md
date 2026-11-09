# internal/update

`tt update` 的全部实现：查有没有新版、下载并校验制品、把"替换二进制"交给一个脱离的
更新器进程。

**在链条的哪一环**：它不碰 ERP，只碰 tt 自己。上游是 GitHub Releases（`luyaogod/TT`），
下游是三种安装形态的文件替换；命令层（`internal/cli/update.go`）只做参数解析、输出与
确认，业务都在这里。

## 做什么 / 不做什么

做：

- **查**。读发布元数据（tag、发布时间、资产的 sha256 摘要），与手上这份版本比较。
- **下**。流式下载 + 边下边算 sha256 + 摘要不符就删；已存在且摘要对得上就跳过（幂等）。
- **验**。便携包先解到旁边、跑一遍载荷里的 `tt.exe version`，自报版本必须等于目标版本；
  MSI 靠摘要 + 装完再跑一遍装上去的那个 exe 验版本。
- **换**。便携包就地覆盖（保留包外文件），MSI 交给 `msiexec /qb`。**由另一个进程做**。
- **收口**。记状态与日志；把 agent 目录里的技能树刷成与二进制同版（见 `skills.go`）。

不做（边界比职责更值得写）：

- **不自动更新**。没有任何后台轮询：`tt serve` 与日常命令都不查，检查只发生在显式命令
  （或设置页上人点按钮）时。
- **不对源码态动手**。`make build` 出的仓库根 tt.exe 只允许查（`KindOther` + 空版本号），
  装会被拒 —— 见 `kind.go` 的 `InstallRefusal`。
- **不合并版本号的两个来源**。二进制自报的版本来自构建注入（根 `VERSION` 文件，严格
  `MAJOR.MINOR.PATCH`），发布 tag 允许带预发布后缀；两者用同一个解析器，因为比较它们
  时必须知道后缀的存在（`0.3.0-rc1` 比 `0.3.0` 旧）。
- 不管 tt 之外的软件：引擎 exe 与设计器 dll 随包走，更新单位是整包 / MSI。

## 独有机制与不变量

**自替换必须由另一个进程完成。** Windows 上运行中的 exe 不能替换自己，而安装目录里还有
`.tzs` 引擎守护进程与设计器 dll 锁着文件。所以 `tt update` 把自己**复制到 `%TEMP%`**
再 `winproc.SpawnDetached` 拉起它（`apply.go` 的 `Handoff`），前台随即退出；更新器等前台
退出、停掉记在计划里的那些 pid、再动手。

**计划是跨进程契约。** `.tt-update.json`（状态）、`update/plan.json`（计划）、
`.tt-update.log`（日志）三份文件的字段名改动要同时想清楚 `tt update log` 与排障。

**落点分两处**（见 `internal/config/README.md` 的缓存清单）：

| 东西 | 落点 | 性质 |
|---|---|---|
| 下载的安装包、检查结果、升级计划 | `<数据目录>/update/` | 缓存（在 `config.CacheSubdirs` 里），删了重新下 |
| 升级状态、日志、技能安装戳 | `<数据目录>/.tt-update.{json,log}`、`.tt-skills.json` | **不是缓存**：排障要看，与 `.tt-serve.json` 同类 |

**便携包不能整目录换掉。** 便携形态的 `config.json` 就住在被覆盖的那个目录里，而 zip 带的
是一份空模板 —— 覆盖它等于删掉用户的口令与环境。`zipx.go` 的 `keepLocal` 是那条防线，
它只管一个文件，但那个文件最贵。代价：上一版删掉/改名的文件会留在原地。

**只有显式代理才自带规则。** `ExplicitProxy` 只回答"要不要显式用某个代理"，其余情况交给
Go 的 `http.ProxyFromEnvironment`（连同 `NO_PROXY` 的既有语义）；自己读一遍环境变量等于
把 `NO_PROXY` 丢掉。

**没有摘要就不装。** `Asset.HexDigest()` 为空时 `Download` 直接拒绝：整条路径上唯一不靠
信任的判断就是"下到的字节 == 发布方发布的字节"。

## 判据

① 可重放的命令（本机需能出网；这台机器要 `--proxy http://127.0.0.1:10808`）：

```
go test ./internal/update -count=1 -v
tt update check                      # 只看结论：退出码 10 = 有新版，0 = 已最新
```

② 期望结果：`go test` 全绿（27 条）；`tt update check --format table` 打出
当前版本 / 形态 / 更新源 / 最新版本四行，退出码按上表。真装一次的可重放判据在便携形态上：
造一个带 `.portable` 的旧版目录，跑 `tt update --yes`，等 `.tt-update.json` 的 `phase`
变成 `done`，再断言目录里的 `tt.exe version` 等于目标版本、`config.json` 的用户内容还在。

③ 这一层负责证明哪件事：**"装下去的东西就是发布方发布的那一份，而且装完真的能跑"**。
逐条对应：摘要校验证明字节（`client.go` 的 `Download`）；制品自报版本证明"这份字节是那次
发布"（`stagePortable` 与 `Apply` 的装后校验）；换完之后 `config.json` 与包外文件仍在
（`zipx.go` 的 `keepLocal` 与 `copyTree`）；失败时旧 `tt.exe` 回得来（`applyPortable`）。

④ 故意不覆盖什么：**MSI 形态的真机替换没有单测**（要一次真装，见 `tt update log` 的
`msiexec` 退出码），这是结构上难测而不是留给别的层；`winproc` 的杀进程/探测语义由它自己
的包负责；HTTP 层只用 `httptest` 假服务，真联网只由 `make update-live-check` 覆盖。

## 细节去哪

- 命令面、退出码、`tt update` 的四种用法 → `tt update --help`
- 落点与缓存清单的规矩 → `internal/config/README.md`
- 技能戳为什么不是 `SKILL.md` 里的一个字段 → `skills.go` 顶部的注释
- 计划的字段含义与更新器做事的顺序 → `apply.go`
