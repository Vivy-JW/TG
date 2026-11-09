# internal/dev/fence — 围栏协议

`Document ⇄ FencedText`：把合成出来的文档渲染成**一个带标注的 4GL 文件**（人/AI 唯一编辑界面），
以及把编辑后的文件解析回来。

**契约：`render` 与 `parse` 互为逆。** 对任意合法包，
`parse(render(synthesize(pkg)))` 必须还原出同一个文档（Region 序列逐项相等）——
这是"不破坏"能被机械证明的前提。

围栏是**单行注释**，不改变 4GL 语义，与框架原生的 `{<point/>}` / `{<section>}` 标记区分开。

## 围栏行

```
{//@tdev:begin <kind> <name> [<旗标> src="s" new="Y" order="" fn="..." scope="..." desc="..."]}
{//@tdev:end <kind>}
```

前缀是固定常量。**导出时如果文档里已经出现这个前缀，直接拒绝**（协议与内容冲突，没有绕过）。

| 旗标 | 含义 |
|---|---|
| `EDITABLE` | 可编辑的自订定义点 |
| `READONLY` | 只读 |
| `EDITABLE-SEC` | 框架解锁后可编辑的**区段** |
| `APPEND` | 锚点区段：允许在其中追加新的点块 |
| `PLAIN` | 裸名插入点：没有函数身份，整块即正文 |

关键字按固定顺序输出，值用统一的转义规则 —— 同一份文档渲染两次必须逐字节相同。

### 锚定部分 vs 可改字段（本包最要紧的一处设计）

围栏行的**锚定部分**（类型、名字、旗标、以及除下面三个之外的所有 `key="值"`）由工具推导，
**不许增删改**；`fn` / `scope` / `desc` 是**唯一允许改**的字段。

gate1 因此不比对整行字节，而是比对 `AnchoredSignature`（锚定部分的签名）——
否则用户改一次函数名就会先撞上"围栏行被改"，而看不出真正的问题在结构事务。
改这三个字段会触发 [../verify/README.md](../verify/README.md) 的 V1 / V5 / V6。

## 规则

1. `begin` / `end` 严格配对；**允许 `section → point` 一层嵌套**（真实框架里自订点就住在
   区段内），深度 > 2 直接报错。这是与上游设计指南的一处**有意偏差**（D-2），导出与解析两侧
   都校验。
2. **围栏行本身属于"围栏外"** —— 不许增删改（可改字段除外，见上）。
3. 自订定义点内部再分**结构行**（注释头 / 签名行 / `END`）与**正文本体**，只有正文本体可改。
4. 新增点只能出现在 `APPEND` 锚点区段的围栏块里，且类型要匹配、名字不与已有 Region 重名。
5. 删除点只允许删掉本次新增的那些；**区段围栏块不可删除**（只能改正文）。

## 坐标系统

`Render` 返回三样东西：带围栏的字节、**围栏文本坐标系下的 Region 表**（基线 Region 的深拷贝，
区间已换算过去）、以及**未归属字节段**（前缀 / 区间之间的空隙 / 后缀）。

未归属段是必需的：真实包里区段与区段之间必然有空行，把它们排除在外就没法做逐字节比对。
渲染完会在围栏文本上重算这些段。

`ApplyEdits` 是文档内定点编辑的共用原语（`unlock` / `rename` / `newfn` 都用它）：
施加若干互不重叠的替换，并返回**位移修正后**的 Region 表与未归属段。
这三个命令都只改工作区里的渲染文档、不碰包，但改完之后所有区间坐标必须整体校正 ——
否则基线（`regions.json`）就会错位。

## 解析

`Parse(base, fenced)` 用**配对**而不是按名字查表：真实包里存在**区段 id 与点名同名**的情形
（同一个名字既是 `<section id>` 又是区段内的裸名插入点），按名字查表会张冠李戴。
配对按 `end` 围栏出现的顺序建立。

`ParseResult` 给出四样东西：编辑后的文档、基线↔编辑后的配对（含两侧围栏行的原文与结构化视图）、
**新增**的点区间、**消失**的基线区间（保留基线元数据，供删除授权判定）。

解析阶段就把这些违反面分别报出来，不留给后面的闸门去猜：

- 嵌套超过两层 / `end` 多于 `begin` / `begin` 没有 `end` / `end` 类型与 `begin` 不匹配
- 无法归属的围栏块（不在基线序列里，也不在 `APPEND` 锚点区段内；或父区段不允许追加）
- 追加点类型与锚点区段不匹配 / 追加点与已有 Region 重名
- 删除了区段围栏块
- 结构行被改（属于可编辑区内的结构部分）

配错类的错误是退出码 3。

## 判据

```bash
go test ./internal/dev/fence -count=1     # 约 48 秒，跑真实语料
# TestFenceRoundTripCorpus            可逆性：语料级 render/parse 往返一致、区间记账对称
# TestRenderContainsFencesAndMetadata 渲染出的围栏与元数据
# TestParseRejectsEditedStructureLine 改结构行被拒
# TestParseRejectsUnpairedFence       未配对围栏被拒
# TestParseDetectsDeletion / TestParseDetectsAppendInsideAnchor / TestParseRejectsAppendOutsideAnchor
```

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 围栏行格式 / 旗标 / 关键字顺序 | 波及 [verify](../verify/README.md) 的 gate1（它比对的是锚定部分签名）、`manifest.json` 的呈现，以及 `selftest` 里那些"改围栏行应被拒"的对抗用例 |
| `structFields`（可改字段集合） | 必须与 [verify](../verify/README.md) 的 V1–V7 同步 —— 这边放开的字段，那边要有对应校验，否则出现"能改但没人管"的缝 |
| 可写区间的划分 | gate1 的判定面就是"全文 − 可写区间"（见 [verify](../verify/README.md)）。这里多划一块，gate1 就少守一块 —— 这是最容易出安全缺口的地方 |
| 嵌套深度规则 | 导出侧与解析侧各有一道深度校验，要一起改；`selftest` 与 `TestParseRejectsUnpairedFence` 覆盖 |
| 渲染文本（空白、换行、缩进） | 会改变工作区字节 → 影响可逆性判据与已有工作区的基线 |

**可逆性判据**（`TestFenceRoundTripCorpus`，约 48 秒）是这一层的总闸门：渲染与解析不再互逆，
整条管线就失去意义。

## 细节去哪

- 行尾：受保护字节的 CRLF↔LF 不算改动，判定在 [../model/README.md](../model/README.md)（`EqualEOL`）
- 闸门怎么用解析结果 → [../verify/README.md](../verify/README.md)
- 文档从哪来 → [../synth/README.md](../synth/README.md)
