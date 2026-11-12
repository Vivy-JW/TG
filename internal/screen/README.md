# internal/screen — `.42f` + `Labels` 的前端画面模型

把 [form42f](../form42f/README.md) 的页面/字段布局与调用方给的 `Labels`（TG 是 gae_file 的译文）合成
`View`：页面树、字段表、每个标签的解析状态。纯函数，零 IO。

## 合成规则

- 页面 id 用 42f XML `name`；`textKey` 只用于查标签。归一化后的 `page_1` 可能重复，不能当 id。
- 字段结构、控件、类型、必填来自 42f；标签只来自 `Labels`。GP 没有 `.42s`，画面文字在 `gae_file`（`p_perlang` 维护，运行期由 `lib/4gl/cl_ui_locale.4gl` 取）。
- 查标签先精确匹配，再只去掉 `<form>.` 前缀或 `.<form>` 后缀；不做模糊匹配。
- 找不到标签、没有关联到标签、同一 token 多个文本，分别标
  `labelState=unresolved|ambiguous` 与 `labelReason`，不拿 token 冒充中文。
- `Labels` 为 nil 时仍返回结构，`Warnings` 说明标签未解析。
- 页面外字段计入 `GlobalFieldCount`，不静默丢弃。

## 不变量

- `PageView.ID` 唯一；`TextKey` 不保证唯一。
- `Fields` 保持 42f XML 顺序；`FieldsInPage` 只返回直接挂在该页的字段。
- `LabelStats` 的计数覆盖所有页面/字段，即使命令只展开了其中一部分。

## 判据

```bash
go test ./internal/screen -count=1
```

合成夹具钉住：页面标签、字段标签、未关联标签、页面外字段、没有标签。

## 细节去哪

- 页面/字段布局 → [../form42f/README.md](../form42f/README.md)
- 标签来源（gae_file）→ [../dict/README.md](../dict/README.md)
- 命令与 JSON 形状 → [../cli/interfacecmd/README.md](../cli/interfacecmd/README.md)
