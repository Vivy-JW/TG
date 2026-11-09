# internal/dev/fgl — 4GL 结构判定

两件与 Genero 4GL（BDL）源码结构有关的能力：

| 文件 | 能力 |
|---|---|
| `block.go` | **一个块信封**的判定与行/字节范围：`FUNCTION` / `MAIN` / `DIALOG` / `REPORT`，以及 `PRIVATE`/`PUBLIC` 作用域 |
| `outline.go` | 源码大纲：块节点树（给编辑器与工具用） |

机械重写为什么需要它：自订定义点内部要区分**结构行**（注释头、`PUBLIC FUNCTION x(...)`、
`END FUNCTION`）与**正文本体** —— 只有正文本体可改，结构行被改一律拦下。
`ParseBlock` 就是那条边界线的判定者（见 [../synth/README.md](../synth/README.md) 与
[../verify/README.md](../verify/README.md)）。

## 关系

- 被 [synth](../synth/README.md)、[verify](../verify/README.md)、[split](../split/README.md)、
  [fence](../fence/README.md)、[cli](../../cli/dev/README.md) 使用；本包不 import 组内任何包。
- 夹具在 [`testdata/fgl-fixtures/`](../../../testdata/fgl-fixtures/README.md)（61 组，
  期望值由 BDL 语言文档推导）。

## 实现要点

`outline.go` 是 TDebug 那份 `fgloutline.ts` 的 1:1 Go 移植（后者又是 BDL 扩展的移植，
MIT，同作者），只保留大纲、不保留折叠。管线是
`maskLines（掩码）→ scanBlocks（块栈扫描）→ fixup（范围修正）→ toOutlineNodes`。

四条设计要点，每条都有真实语料实证：

1. **用块栈，不用缩进** —— 真实源码的缩进会谎报层级（同类子块 `ON ACTION` 比宿主还浅）。
2. **扫描前把字符串内容与三类注释（`#`、`--`、`{ }`）抹成等长空格** —— 否则跨行字符串里的
   SQL 关键字、块注释里的 `ON ACTION`、注释里的撇号（会开启跨行字符串状态）都会造幻影节点
   或吃掉成片节点。
3. **块开头按构造消歧** —— `RECORD` 里名叫 `report` / `construct` 的字段不是块开头。
4. **深嵌套用显式栈，不用递归** —— 畸形文件里约 2000 层嵌套会让递归栈溢出、大纲全空。

**移植到 Go 时的两处有意偏差**（其余逐行等价）：

- RE2 没有前瞻/后顾，原正则里的 `(?!…)` 按位置分两类处理（恒真的直接删；位于分支末尾的
  改成消费一个边界字符再用捕获组起点把匹配长度退回去）。
- 按**字节**而不是 UTF-16 码元推进，因此正则下标与选择区是**字节列**；对纯 ASCII 行与
  前端实现完全一致（夹具 `function-method-receiver-namecol` 就是这种情形）。

标签文本、块范围、以及上游已声明的已知偏差都照原样保留，以便与夹具 1:1 对拍。

## 判据

```bash
go test ./internal/dev/fgl -count=1 -v
# 夹具对拍：用例 61 个，通过 58，已知偏差 3，不符 0
#   已知偏差（夹具里带 deviation 字段，不算失败）：
#   dialog-subdialog（SUBDIALOG 未识别为节点）、
#   input-by-name-record-star（标签丢失整记录 vs 单变量的区别）、
#   input-nested-semicolon（`;` 不作终止符）
```

语料侧的信封判定另有 `TestCorpusEnvelope`（跟默认测试一起跑）。
`deviation` 字段的语义与"不许把实现输出回写当期望"的约定见夹具的 README。

## 细节去哪

- 期望值与 `deviation` 的约定 → [../../../testdata/fgl-fixtures/README.md](../../../testdata/fgl-fixtures/README.md)
- 前端那份同源实现 → [../../../web/app/src/fgloutline.ts](../../../web/app/src/fgloutline.ts)
