# testdata — 测试数据

两套夹具：

| | 是什么 |
|---|---|
| [fgl-fixtures](./fgl-fixtures/README.md) | 4GL 大纲解析器的文档级对拍用例（61 组 `.4gl` + 期望值） |
| [tzs-mini](./tzs-mini/README.md) | **最小 `.tzs` 工作区**：三个典型表单包 + 它们要的元数据，3.2 MB |

**fgl-fixtures 有两份**：本目录是 Go 侧那份，前端侧还有一份在
[../web/app/scripts/fgl-fixtures/](../web/app/scripts/fgl-fixtures/README.md)（它的检查项
跑在 Node 里，从自己的目录读夹具）。**两份内容必须逐字节相同**，改一处要同步另一处。

**`tzs-mini` 是把真语料筛小了放进仓库的那一份** —— 为的是让 `.tzs` 的冒烟回归不必依赖
「有真客户语料的那台机器」。它**不替代**真语料回归，两者分工见它的 README。

真实语料（客户的 `.tzc` / `.tzs` 包）**不在仓库里**：它是活目录、是客户数据，
由环境变量指向。
（`tzs-mini` 是**筛过的那一份**：只有三个包与它们读取的元数据，不是活目录、不是完整工作区。）

## 细节去哪

- 夹具的来源、判据与改动约定 → [fgl-fixtures/README.md](./fgl-fixtures/README.md)、
  [tzs-mini/README.md](./tzs-mini/README.md)
