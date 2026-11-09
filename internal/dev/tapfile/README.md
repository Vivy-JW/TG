# internal/dev/tapfile — `.tap` 的字节保真改写

`.tap` 是包里**唯一被服务端消费**的设计文件（`.4gl` 是服务器 build 产物、`.tgl` 是框架骨架）。
本包负责它的解析与改写，方式只有一种：**CDATA 感知的字节级扫描**。

**红线 R3：禁止用 XML 库把 `.tap` 整体序列化。** 解析成 DOM 再序列化会重排属性、改写引号、
归一空白 —— 对设计器来说那是另一份文件。

## 形状

```go
doc, err := tapfile.Parse(raw)                 // 只读视图：元素、属性、CDATA 区间
newRaw, err := tapfile.Rewrite(orig, ops...)   // 产出新字节；除改写点外逐字节不动
```

## 扫描器做什么、不做什么

**只做结构定位**：找出根元素、`<point>`、`<section>` 以及它们各自的字节区间（标签区间、
属性区间、CDATA 区间）。**不做任何 XML 规范化** —— 不合并空白、不统一引号、不重排属性。

扫描时跳过空白、XML 声明、注释与 DOCTYPE。之所以必须自己扫：`.tap` 里同一个文件混着两种换行
（元素之间是 CRLF、CDATA 内部是 LF），任何"读进来再写回去"的通用做法都会改变字节。

## 只读视图

| API | 语义 |
|---|---|
| `Element.Attr / AttrOr` / `Doc.RootAttr / RootAttrOr` | 读属性（`Attr` 同时给**值在原文里的精确区间**） |
| `Element.Content(raw)` / `HasCDATA` | 取正文：CDATA 内部字节、普通文本，或空 |
| `Doc.PointExact(name)` | 第一个同名点，**不看状态** |
| `Doc.PointsAll(name)` | 所有同名点，按文档顺序 |
| `Doc.Point(name)` | 按设计器语义取"**第一个未删除**的同名点" |
| `Doc.Section(id)` | 按 id 取区段 |
| `Doc.PointAt(off)` / `Doc.SectionAt(off)` | 字节偏移落在哪个点/区段 —— 报错定位用 |
| `Doc.AppendAnchor()` | 新增点的插入位置与分隔符样式 |
| `IsDeleted(el)` | 是否被标记删除（`status="d"`） |

两条容易踩的语义：

- **同名点会共存**：真实包里同一个名字先有一个墓碑（`status="d"`）、随后才是活点
  （`status="u"`），这是设计器组装点列表的顺序。所以**所有写操作一律走 `Doc.Point`**，
  绝不能按名字取第一个就改 —— 那会改到墓碑上。
- **新增点的落点**由 `AppendAnchor` 决定：最后一个 `<point>` 之后；没有点则 `<other>` 之后；
  再没有则根标签之后。分隔符沿用该处已有的空白样式。

## 改写

`Rewrite(orig, ops...)` 依次施加显式的改写动作，**没有"改任意元素"的通用口子**：

| Op | 作用 |
|---|---|
| `SetRootAttr{Key, Value}` | 根元素属性（例如区段解锁要写的标志） |
| `SetPointAttr{Name, Key, Value}` | 某个点的属性（`status` / `new` / `order`…） |
| `SetPointCDATA{...}` | 某个点的正文（自订定义点的函数体就在这里） |
| `AddPoint{...}` | 新增点 |
| `MarkDeleted{Name}` | 把点标成删除态（**保留原正文**，形成墓碑） |
| `RemoveTombstones{Name}` | 清掉同名的旧墓碑 |
| `RenamePoint{From, To}` | 改名事务用 |
| `SetSectionCDATA{...}` / `SetSectionAttr{ID, Key, Value}` | 区段正文与属性 |

实现是**逐步重扫**：每施加一个 op 就基于上一步的结果重新扫描。op 数量通常不到 100 个，
单次扫描（十几万字节量级）是毫秒级 —— 用简单换正确，不为此做增量或游标优化。

## 保真规则

- **属性"有则改、无则加"**：改值时只替换**引号内的那一段**（属性记着自己的 `ValueStart/ValueEnd`），
  所以引号风格、属性顺序、属性之间的空白全部原样保留；属性不存在时追加。
- **除显式改写点之外，其余字节逐字节不动** —— 这是"不破坏"的物理基础之一。
- 自闭合元素在被写入内容时展开成开闭标签对。
- 写入的正文里出现 CDATA 结束序列时直接拒绝（退出码 2 的那类包格式错）。

## 判据

```bash
go test ./internal/dev/tapfile -count=1     # 约 48 秒，跑真实语料
# TestIdentityRewriteIsByteExact    恒等改写逐字节一致
# TestParseAllCorpusTaps            语料里的 .tap 全部可解析
# TestCDATAMatchesIndependentRegex  与一条独立正则对拍，确认 CDATA 区间找得对
# TestAttrSetIfPresentElseAdd       "有则改、无则加" 的属性语义
# TestRewriteSelfClosingPointExpands / TestAddPointKeepsSiblingTemplate / TestAddPointRejectsDuplicate
# TestRejectCDATACloseInContent
```

这几条会打印**当前的语料规模**（`.tap` 个数、`<point>` / `<section>` / 带 CDATA 元素数与
恒等写次数）。实测值随语料增删而变，**别把它当判据** —— 判据是"恒等改写逐字节一致"。

## 改动影响面

| 改什么 | 会波及 |
|---|---|
| 扫描器（元素/CDATA 区间） | 上游是 [synth](../synth/README.md) 的点/区段配对，下游是 [verify](../verify/README.md) 的全部 I 系列判定。`TestParseAllCorpusTaps` 与 `TestCDATAMatchesIndependentRegex` 是防线 |
| 改写逻辑 | 恒等写必须仍逐字节一致（`TestIdentityRewriteIsByteExact`，约 40 秒，跑真实语料） |
| 新增一个 `Op` | 要想清楚两件事：**它能不能穿过 gate1**（可写区间的定义在 [fence](../fence/README.md) / verify 那一侧），以及它会不会产生需要回写的框架补丁（那种情况要同时产出 `.tgl` 补丁，见 [split](../split/README.md)） |
| 同名点解析（`Point`） | 写操作全靠它避开墓碑；改错会**静默改到墓碑上**，而包还能打开 —— 最坏的一类错 |

## 细节去哪

- 谁产出这些 Op（回写拆分） → [../split/README.md](../split/README.md)
- 权限链与合成（哪些点可编辑） → [../synth/README.md](../synth/README.md)
- 三方依据 → 设计器反编译源码 [`designer-src/`](../../../designer-src/README.md)：
  根元素与点/区段结构、混合换行、属性改写规则
