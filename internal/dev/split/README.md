# internal/dev/split — 回写拆分

`Document' → TapOp + TglPatch`：把"编辑后的文档"翻译成**具体要在包里打哪些补丁**。
本包只产出计划，不动任何字节；真正写盘的是 [../pkgfile](../pkgfile/README.md)（包容器）与
[../store](../store/README.md)（工作区）。

## 一次写回计划里有什么

| 字段 | 含义 |
|---|---|
| `Ops` | 要施加到 `.tap` 上的操作序列（见 [../tapfile/README.md](../tapfile/README.md) 的 Op 表） |
| `TglPatches` | 要打进 `.tgl` 的区段正文补丁（**改区段时才有**） |
| `SetSectionFlag` | 是否要把根上的解锁标志置位 |
| `Changed` / `Renamed` / `Added` / `Deleted` | 按种类列出的变动点，用来生成报告与 `--dry-run` 的说明 |

## 两条要点

**区段正文要写两处。** `.tap` 的 `<section>` 与 `.tgl` 里同一区段的正文必须逐字节一致 ——
所以区段改动会同时产出 `SetSectionCDATA` 与 `TglPatch`。只写一处会让包自相矛盾。

**改名是一次事务，不是一个属性改动。** 设计器把自订定义点的身份拆在多处（点名、签名行、
作用域、描述块、墓碑、调用点），任何一处单独漂移都是故障。所以改名落盘时复刻设计器的事务序列：

```
.tap 里一次改名 = 一对 <point>
  function.旧名   status="d"    ← 墓碑，保留原正文逐字节
  function.新名   status="u"    ← 活点，签名行已改为新名
```

配套的校验（V1–V7）在 [../verify/README.md](../verify/README.md)；改名/新增的命令入口在
[cli](../../cli/dev/README.md)。

## 入口

| 入口 | 作用 |
|---|---|
| `Split(base, parsed, pkg)` | 产出写回计划 |
| `Plan` / `TglPatch` | 计划与补丁的结构 |
| `BuildSignature(oldSig, kind, scope, fn)` | 由旧签名行与新身份拼出新签名行 |
| `VerifyError` / `DeniedError` | 拒绝类错误退出码 4 |

## 判据

端到端那一层（改名事务、新增点、删除授权、真实语料写回仿真）**已经在默认档里跑** ——
`tt dev tzc selftest` 的 31 项被接进了 `go test`。所以本包的单测
**不重测那些**，只钉端到端钉不到的纯函数：

```bash
go test ./internal/dev/split -count=1
```

**覆盖**：点名怎么拆（`isSelfDef` / `bare` 去前缀与参数列表 / `prefixOf`）、
**签名行怎么拼**（`BuildSignature`：FUNCTION 恒写限定符，DIALOG/REPORT 在 PUBLIC 时
**省略** —— 错一个字就是静默走偏）、order 怎么分配（跳过已删除与非自订点）、
`Plan.Describe` 的摘要按种类报数、`content` 的区间越界兜底、两个错误类型的退出码。

**故意不覆盖**：`Split` 本身（它要 `Document` + `ParseResult` + `Package` 三件套）。
那条链由 `tt dev tzc selftest` 的端到端用例与 `TDEV_DEEP=1` 的真实语料写回仿真负责：

```bash
TDEV_DEEP=1 go test ./internal/cli/dev -run TestCorpusApplySimulation
```

## 细节去哪

- 补丁落进包之后怎么被重建 → [../pkgfile/README.md](../pkgfile/README.md)
- 不变量 I1–I15 具体怎么判 → [../verify/README.md](../verify/README.md)
