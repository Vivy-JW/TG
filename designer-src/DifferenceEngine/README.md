# DifferenceEngine —— 行级 diff 算法库

独立 diff 库（RootNamespace `DifferenceEngine`，无 ProjectReference）：`DiffEngine` 用"最长匹配子序列"
递归（GetLongestSourceMatch/ProcessRange，DiffEngine.cs:22-70+），提供 FastImperfect / Medium /
SlowPerfect 三档精度（DiffEngineLevel.cs:6-13）—— 经典 CodeProject DiffEngine 的移植。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 13 | `DiffEngine`（公开 ProcessDiff(IDiffList, IDiffList, DiffEngineLevel) 返回耗时 double，DiffEngine.cs:7，:129/:136）、IDiffList（:6）+ 四个输入实现 DiffList_TextFile/DiffList_BinaryFile/DiffList_CharData/DiffList_TextContent（各 :6-7）、DiffState（internal，:6）、DiffStateList（internal，:6）、DiffResultSpan（:6，IComparable）、DiffStatus/DiffResultSpanStatus（internal 枚举，:6）、DiffEngineLevel（:6）、`TextLine`（:6） |
| Properties/ | 1 | AssemblyInfo.cs |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| TextLine.cs:12 | 构造把 Tab 换 4 空格并缓存 `GetHashCode()`；`CompareTo` 只比 hash（:9-19）—— **行等价性由字符串哈希决定** |

## 细节去哪

- UI 消费方（DiffManager/DiffRenderer）：[../CodeEditWindow/README.md](../CodeEditWindow/README.md) ·
  [../CodeEditWindow/Helper/README.md](../CodeEditWindow/Helper/README.md)
- 上层索引：[../README.md](../README.md)
