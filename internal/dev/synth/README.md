# internal/dev/synth — 合成与权限判定

把包**合成**成一份可编辑文档（`Package → Document`），并给其中每一块标上"能不能改"。
合成的产物是 [fence](../fence/README.md) 渲染围栏的输入。

**两条权限判定链是本包最重要的东西** —— 机械重写最大的风险不是写错字节，而是**写了一个
设计器本来不许改的地方**。

## 合成做了什么

逐条复刻设计器的装载流程：

1. 取 `.tap` 与 `.tgl`；缺任何一个都是合成错误（该包本来就不完整）。
2. 解析 TAP，取出根上的程序名与解锁标志。
3. **合并解锁状态**：包级事实（根标志已是 `Y`）**优先**；否则看工作区的意图
   （`pending_unlock`，或调用方显式要求已解锁 —— `unlock` 命令的重渲染走这条）。
4. 读环境上下文（环境、登录用户、程序类型、行业别、topstd 模式…），供后面的权限链使用。
5. **锚点展开**：三类集合锚点按**固定顺序**（FUNCTION → DIALOG → REPORT）处理；每一类里的点
   先按 `order` 排序，渲染成 `{<point name="…"/>}` 行注入 TGL 对应锚点。
6. **区段配对**：按序号配对 + 模板克隆（先找客制区段、再找标准区段，都没有就是"找不到区段"），
   并给区段打上强制只读。
7. **占位符处理**：TGL 的占位符与 TAP 的点配对 —— 有占位符没点就**造空点**（常态），
   并记下标记供回写折叠使用。

合成完，每个点与每个区段都带着"可编辑与否 + 原因码"进入 `Region`
（原因码由 [../model/README.md](../model/README.md) 的 `DenyReasonText` 翻成中文写进
`manifest.json` —— 所以看到 `editable:false` 的同时就知道为什么，不必试错探测边界）。

## 点：G1–G7（`ResolvePoint`）

| 门 | 条件 | 结果 |
|---|---|---|
| G1 | 独立功能程序，且行业别与宿主不匹配 | 不可编辑 |
| G2 | 标准环境 + 行业别 `sd`/空 + 行业备忘点 | 不可编辑 |
| G3 | 非标准件且引用了标准代码 | 不可编辑 |
| G4 | 标了只读 | 不可编辑 |
| G5 | 插入点的 `edit=` 与环境组合（客制点遇上标准环境 / 标准点遇上客制环境…） | 不可编辑 |
| G6 | topstd 编辑模式：按来源与状态收窄到只有"新建 / 未改 / 本次修改"可编辑 | 受限 |
| G7 | 登录用户是 `topstd` 且来源是客制且状态不是新建 | 不可编辑 |

外加一条**本工具的政策**：自订定义点的正文解析不出块信封 → 一律不写回
（设计器装载会抛异常，写进去就是个打不开的包）。

`new="Y"` **不参与**可编辑性判定（它只关删除授权）；是否自订定义点也不参与（只关改名）。

## 区段：政策层 + S1–S5（`ResolveSection`）

前两层是本工具自己的政策，**先于**设计器链生效：

1. **三个集合锚点区段的正文永不写** —— 那是注入进来的点块，写它等于把展开后的函数写进框架
   （红线 R4）。设计器在某个组合下对其中一个锚点有放行例外，我们**不复刻**那个例外。
2. **框架未解锁 → 一律只读**（先 `tt dev tzc unlock`）。

然后是设计器链 S1–S5：`type="G"` 且已解开时的例外、运行期只读（锚点硬编码或框架标了只读）、
区段属性标了只读、topstd 模式下的来源规则、登录用户门禁。

> **解锁改变的是门，不是所有房间。** 解锁之后仍然逐条走设计器链 —— 锚点区段、框架标了只读的、
> topstd 模式下的来源规则，一个都不放松。

## 解锁闸门（`CheckUnlockPermission`）

逐条重放设计器那个勾选框的判定，三个分支：

| 情形 | 判定 |
|---|---|
| 包本来就是解开态（根标志已是 `Y`） | 允许（设计器对这类包不设拦截） |
| 标准环境 + 非 topstd，无授权事实 | **拒绝**，附授权原文 |
| 标准环境 + 非 topstd，有授权事实 | 允许，但需二次确认 |
| 客制环境或 topstd | 允许，但需二次确认（打印代价警告） |

**没有绕过**：不提供强制开关，不改环境与登录用户，也不写"假装有授权"的标志 ——
那是服务器侧的权限域，本地工具自我授权等于伪造权限。是否解锁必须由人决定。

## 入口

| 入口 | 作用 |
|---|---|
| `Synthesize(pkg, Options)` | 合成文档 |
| `ResolvePoint` / `ResolveSection` | 判定单个点 / 区段的权限，返回"可否"与**原因码** |
| `CheckUnlockPermission(env, yes)` / `UnlockDecision` | 解锁闸门（含 `NeedYes` 与要打印的原文） |
| `DeniedError` / `SynthesisError` | 拒绝类错误的退出码是 4 |

## 判据

```bash
go test ./internal/dev/synth -count=1
# TestResolvePointTruthTable    点的权限链真值表
# TestResolveSectionTruthTable  区段的权限链真值表
```

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 权限链（`ResolvePoint` / `ResolveSection`） | **这是最贵的一类改动**：放宽会让工具去写设计器不许改的地方，而包仍然能打开 —— 错误要到设计器里才暴露。真值表必跑 |
| 解锁闸门 | 与 `unlock` 命令、工作区的解锁状态（[../store/README.md](../store/README.md)）三方一致；`selftest` 覆盖各分支 |
| 合成步骤（锚点展开 / 占位符 / 区段配对） | 上游是 [pkgfile](../pkgfile/README.md) 与 [tglfile](../tglfile/README.md)，下游是 [fence](../fence/README.md) 的渲染、以及 gate3 的重跑合成 |
| Region 的 `reason` 文案 | 直接出现在 `manifest.json` 里（AI 与人都读），改文案等于改对外说明 |

```bash
go test ./internal/dev/synth -count=1     # 两条权限链的真值表
```

## 细节去哪

- 围栏怎么把权限渲染成可读标注 → [../fence/README.md](../fence/README.md)
- 区段回写（解锁之后才可能发生） → [../split/README.md](../split/README.md)
- 三方依据 → 设计器反编译源码 [`designer-src/`](../../../designer-src/README.md) 的
  `CodeEditorManager.cs` / `AddPointModel.cs` / `SectionModel.cs`
