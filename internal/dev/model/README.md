# internal/dev/model — 领域模型与定位

`.tzc` 管线的**共用底座**：领域模型（`Region` / `Document` / `EnvContext` / `Span`）与一组
纯函数工具（字节偏移 → 行号、行尾归一比较、sha256、稳定 JSON）。

**本包只有纯数据与纯函数** —— 不含 IO、不含业务逻辑，所以 synth / fence / verify / split /
store / cli 都能依赖它而不产生循环。

## 关键内容

| 东西 | 作用 |
|---|---|
| `Document` / `Region` / `RegionKind` / `Span` / `ByteRange` | 合成与围栏共用的一套坐标：Region 带名字、类型、权限、原因、字节区间 |
| `EnvContext` | 权限判定链要吃的那一小撮环境事实（env / login_user / topind / type…） |
| `SectionState` + `EffectiveSectionState` | 区段解锁状态：**包级事实优先** —— TAP 根 `section_flag=="Y"` 就是已解开，否则看工作区是否已 `unlock` 迁移过 |
| `DenyReasonText` / `UnlockedByOf` | 把内部判定码翻成人能读的中文原因（写进 `manifest.json` 的 `reason`） |
| `LineOf` / `LineAt` | 字节偏移 → 1-based 行号；`LineAt` 同时给该行内容，是「报错指到第几行」的出口 |
| `FirstDiff` / `FirstDiffEOL` | 两段字节的第一处差异；`FirstDiffEOL` 按**行尾等价**比较（受保护字节 CRLF↔LF 不算改动） |
| `NormalizeEOL` / `EqualEOL` | 行尾归一与等价判定，同上 |
| `Sha256Bytes` / `Sha256File` | 基线比对（`base.sha256`、源包未变校验） |
| `MarshalJSONStable` | 固定两空格缩进的 JSON（`manifest.json` / `regions.json` 用它，保证同一输入逐字节相同） |

## 契约与不变量

- **行尾等价的边界**：只有**受保护字节**（围栏行、只读区、结构行、围栏外）适用行尾等价；
  可编辑区里除了行尾归一之外的任何差异都照旧拦截。
- **`SectionState` 是单向的**：解锁是一次显式、有代价、不可逆的状态迁移，不是开关。
  它取代了早期版本里那个 `--allow-sec` 标志。
- **不含时钟**：本包所有序列化都不写时间戳（同一输入必须产出逐字节相同的产物）。

## 判据

```bash
go test ./internal/dev/model -count=1
```
覆盖 `LineOf`/`LineAt` 的边界、`FirstDiff`/`FirstDiffEOL` 在 CRLF 与 LF 下的差异。

## 细节去哪

- 报错定位怎么串到用户面前 → [../README.md](../README.md) 与 [cli](../../cli/dev/README.md)
- 权限判定链本身（G1–G7 / S1–S5） → [../synth/README.md](../synth/README.md)
