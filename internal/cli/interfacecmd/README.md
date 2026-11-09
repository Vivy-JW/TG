# internal/cli/interfacecmd — `tt interface`

把 `.42f`（编译后的表单）与 `.42s`（编译字符串）合成前端画面模型，供 Agent/人理解
“这屏有哪些页、每页有哪些字段、字段中文是什么”。

## 命令

```text
tt interface show  <form>     页面结构；默认 depth=2，不展开字段；--fields 展开
tt interface pages <form>     页面扁平索引
tt interface fields <form>    字段扁平表；默认最多 30 个字段（--page/--col/--kw/--limit/--full）
tt interface labels <form>    隐藏诊断：42s token 表
```

数据源优先级：`--42f/--42s` > `--dir <环境镜像目录>` > `--at <环境>`（配置里的镜像目录）。
`--dir` 是最终环境目录，不是全局根；镜像由 `tt file mirror pull --profile interface` 建立。

## 页面选择

`--page` 只接受 42f 页面 `id`（XML `name`），不接受 `textKey`。样本里
`master_folder_page` 的 `textKey=page_1`，另有一个页面 `id=page_1`；用 textKey 会双关。
`textKey` 只出现在输出里用于查标签。

## JSON 契约

成功信封固定在 `schema: tt.interface/v1`，`data.kind` 为：

```text
interface.show | interface.pages | interface.fields | interface.labels
```

页面 id/textKey/label/fieldCount；字段 fieldId/colName/label/labelState/widget/sqlType/required；
标签问题在 `labelState`、`labelReason`、`unresolved[]` 中显式给出，不猜。

`fields` 默认是 summary（最多 30 条，`data.detail=summary`），超过时带
`truncated/returned/total` 与 `hint`；`--full` 等同 `--limit 0`，`--brief` 继续控制字段属性。

## 退出码

```text
0 成功
1 42f/42s 不合法或读不到（FORM_UNKNOWN / LABELS_INVALID）
2 用法错、表单/42s 同名歧义（FORM_AMBIGUOUS / LABELS_AMBIGUOUS）
4 精确页面/字段不存在（PAGE_NOT_FOUND / FIELD_NOT_FOUND）
```

`--kw` 零命中是成功空数组；`--col` 零命中是 `NOT_FOUND`。

## 判据

```bash
go test ./internal/cli/interfacecmd -count=1
```

单测覆盖镜像目录解析、模块消歧、显式文件输入和路径推导。真实样本回读：

```bash
tt interface show --42f <bsft001_wf.42f> --42s <bsft001_wf.42s>
tt interface fields --42f <...> --42s <...> --page page_4 --limit 3 --brief
# 或直接走环境镜像：tt interface fields aapt320 --at 109tst --page page2 --limit 3 --brief
```

## 细节去哪

- `.42f` 布局/标签关联 → [../../form42f/README.md](../../form42f/README.md)
- `.42s` 格式 → [../../str42s/README.md](../../str42s/README.md)
- 合成模型 → [../../screen/README.md](../../screen/README.md)
