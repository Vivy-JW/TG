# testdata — 测试数据

一套夹具：

| | 是什么 |
|---|---|
| [fgl-fixtures](./fgl-fixtures/README.md) | 4GL 大纲解析器的文档级对拍用例（61 组 `.4gl`  + 期望值） |

**fgl-fixtures 有两份**：本目录是 Go 侧那份，前端侧还有一份在
[../web/app/scripts/fgl-fixtures/](../web/app/scripts/fgl-fixtures/README.md)（它的检查项
跑在 Node 里，从自己的目录读夹具）。**两份内容必须逐字节相同**，改一处要同步另一处。

`.tzs` 语料（`tzs-mini` 最小工作区与客户的真实 `.tzc` / `.tzs` 包）2026-11-12 已随
`tt dev` 删除 —— TG 不做设计器包，GP 侧也没有对应用法。

## 细节去哪

- 夹具的来源、判据与改动约定 → [fgl-fixtures/README.md](./fgl-fixtures/README.md)
