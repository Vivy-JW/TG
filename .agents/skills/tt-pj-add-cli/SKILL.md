---
name: tt-pj-add-cli
description: 在 TT 中新增或改造 CLI 命令时使用。触发词：加命令、命令组、Agent 友好、JSON 输出、--brief、投影、批量、错误码、CLI 契约。
---

# 给 TT 加 CLI：只做 Agent 真正需要的命令

TT 的 CLI 是 Agent 的查询 API，不是数据导出器。  
每条命令只回答一个问题，返回足够的信息，并允许调用方继续裁剪。

## 设计四问

1. Agent 要用它回答什么问题？
2. 数据从哪里来，默认是否联网？
3. 默认输出多大，怎么缩小？
4. 查不到、查不了、查错地方，分别怎么返回？

答不清，不要开始写 Go。

## 三条原则

**明确**

- 环境、企业、文件、目录一律点名，没有默认环境。
- 默认只读；远程写操作必须显式、单独设计。
- 未知子命令非 0；精确 ID 缺失是 `NOT_FOUND`。
- 不静默切换数据源，不猜企业，不猜列名。

**可控**

- 默认输出有界：`--limit`、`--depth`、字段级截断。
- 超限返回 `totalRows / returned / truncated` 和查看全量的下一步。
- 投影是选择，不是截断：
  - `--col` / `--kw`：字段
  - `--tables` / `--doclist`：子对象
  - `--brief`：明确的摘要视图
- 被选中的字段/节点保持完整，只缩短数组。

**可信**

- `--human` 只改变展示形态，不能比 JSON 少内容。
- 摘要必须叫 `--brief`，不能靠 `--human` 偷偷降级。
- `--brief` 在 JSON、CSV、表格三种形态都生效。
- 不从定宽文本、截断表头、猜测 token 里取结构；拿不到可信元数据就明确失败。
- 批量中单个缺失不拖垮整批；单项精确缺失仍报 `NOT_FOUND`。

## 命令放在哪

**单动词不要新建组。**

- 一个稳定动作：放现有组，或顶层单命令。
- 只有多个独立数据面才建组，不是多个视图就建组。
- 例：`interface` 有页面/字段/标签三个数据面 → 组；`outline` 只有一棵块树 → 单命令 + flags。
- 不要把“都读文件”当成同组理由；`tt file` 是服务器文件线，纯本地源码分析优先顶层。

新组必须有至少两个稳定动作，在 `internal/cli/at/scopes.go` 认领 `--at` 作用域，并用 `common.UnknownSubcommand` 处理未知子命令。

## 输出与批量

数据命令默认 JSON；`--human` 是同一份数据的人读形态。

统一信封：

```text
ok / source / env / ent / account / target
totalRows / returned / truncated / localPath / notes
data
```

错误用稳定结构：

```json
{"ok":false,"code":"NOT_FOUND","error":"...","hint":"可复制的下一步","exitCode":4}
```

批量统一：

```json
{"items":[{"id":"a","found":true,"data":{}}],"missing":["b"]}
```

规则：

- 批量单个缺失进 `missing[]`，整批仍成功。
- 单个精确 ID 缺失 → `NOT_FOUND` / exit 4。
- `--kw` 搜索无结果 → 成功空数组。
- 每项带 `found`，不要靠数组长度猜。

## 帮助与验收

`--help` 至少包含：2–4 条可复制示例、默认输出、关键 JSON 形状、`--brief`/投影语义、退出码、数据源选择。

测试至少覆盖：

```text
空输入 / 缺目标 / 未知子命令
精确缺失 / 搜索空结果
batch items + missing + found
--brief 在 JSON/CSV/table 都生效
--human 不丢详情
截断元数据
```

默认验证：

```bash
go test ./... -count=1
```

手工回读：

```bash
tt <cmd> --help
tt <cmd> --at-list
tt <cmd> ...
tt <cmd> ... --human
tt <cmd> ... --brief
tt <cmd> <不存在 ID>
tt <cmd> --kw <无匹配词>
```

## 提交前检查

- [ ] 单动词没有新建组
- [ ] 默认 JSON，`--human` 不丢数据
- [ ] `--brief` 是显式摘要，所有格式生效
- [ ] 有字段/子对象投影，精确缺字段报 `NOT_FOUND`
- [ ] 默认有界，截断不静默
- [ ] 批量有 `items[]` / `missing[]` / `found`
- [ ] 错误有稳定 code / exit / hint
- [ ] `--at` 已登记 scopes，没有默认环境
- [ ] 本地/远程不静默切换
- [ ] 帮助、README、测试同步

## 反模式

- 默认打印表格。
- 用 `--human` 省 token，结果只回摘要。
- 只给表格版 `--brief`，JSON 仍是全量。
- 默认输出全部字段/全部行。
- 用行号、数组下标、可重复 textKey 当唯一 ID。
- 从 sqlplus/psql 定宽表头猜列名。
- 查不到就静默换数据源。
- 未知子命令退 0。
- 为“功能丰富”堆只换名字的子命令。

## 参考

- [internal/cli/README.md](../../../internal/cli/README.md)
- [internal/output/README.md](../../../internal/output/README.md)
- [internal/cli/at/scopes.go](../../../internal/cli/at/scopes.go)
