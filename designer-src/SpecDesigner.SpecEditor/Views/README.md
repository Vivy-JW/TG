# SpecDesigner.SpecEditor/Views —— 数据编辑弹窗层

规格栏位/项的批量编辑、动作类型表格、程序关联、差异对照与预览窗口。单层平铺（11 个 .cs + 8 个 .xaml）。
所有增删改都经 UndoRedoManager 命令包装提交，不做直接写。

## 目录地图

| 分组 | 代表 |
|---|---|
| 模型 | ActionTypeData（:8，动作类型行数据）、ItemData（:7，栏位项行数据）、GridLineHelper（:9，DataGrid 网格线辅助） |
| 窗口/控件 | `ItemsPropertyEditor`（:19，Window，栏位项批量编辑，233 行）、ItemPropertyEditor（:11，单项编辑）、`ActionTypeDataGrid`（:21，UserControl）、`ProgRelProgramsDataGrid`（:17，关联程序表格）、DiffListWindow（:16）、PreviewSpecWindow（:12）、SpecExcludeView（:16）、SpecFieldTextControl（:16，IDisposable） |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| ItemsPropertyEditor.xaml.cs:171 | `saveChanged()`（:168 起）提交 datagrid1 编辑后清理 itemSource：Name/Text 均空的行删除、Name 空则取 Text 补齐，再逐条写回 SettingManager 中该 ProgramKey 的 SpecificationInfo；其上方 btnCancel_Click（:160-169）关窗前弹 Message_ConfirmBeforeClose 询问是否放弃改动 |
| ProgRelProgramsDataGrid.xaml.cs:49 | `Add_Click` 开头：`ProgRelProgram.Create(this.ProgRelNode)` 新建关联程序并包成 ChangeProgRelProgramUndoRedoCommand 提交 UndoRedoManager；Delete_Click 走同样管道删除（其上方 ：43-45 是空的依赖属性回调） |

## 值得知道

- ItemsPropertyEditor 内嵌"编辑器自带数据清洗"（空行剔除、Name 补省，:168 起），是少见的编辑器侧
  数据修正点 —— 引擎侧对齐它的行为时以这段为准。
- 本目录操作的对象模型在 [../../SpecDesignerCommon/ViewModel/README.md](../../SpecDesignerCommon/ViewModel/README.md)。

## 细节去哪

- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
