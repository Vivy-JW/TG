# SpecDesignerCommon/UndoRedo —— 表单/组件撤销命令库

命名空间 `SpecDesignerCommon.UndoRedo`：基于外部 UndoRedoFramework 的 `IUndoRedoCommand` 接口
（AbstractUndoRedoCommand.cs:3,8），操作对象统一是 ViewModel.XmlElement。命令由 SettingManager 的
undoRedoManagerMap 按 PackageKey 管理（SettingManager.cs:69-82 → [../README.md](../README.md)）；
属性修改一律先组命令再 `GetUndoRedoManager(key).AddThenExecute` 提交，不直接改值。

## 主题分组（25 个 .cs）

| 分组 | 数量 | 代表 |
|---|---|---|
| 基类 | 1 | `AbstractUndoRedoCommand`（持 Element，提供 GetSpecificationInfo()，AbstractUndoRedoCommand.cs:8） |
| 组件增删改/移动 | 8 | AddComponets / CutComponent / PasteComponent / DeleteComponents / DeleteAct / DragComponents / MoveComponents / ChangeChildIndex 等 UndoRedoCommand |
| 布局/容器/类型 | 7 | Align、BreakLayout、ChangeSize、AddToContainer、ConvertContainerType、ConvertWidgetType、MultiHide |
| 属性/名称/尺寸 | 7 | `SpecAttributeUndoRedoCommand`（被 SpecFieldNode.cs:347、SpecProgRelNode.cs:84 大量构造）、`SpecTreeAttributeUndoRedoCommand`（SpecTreeNode.cs:197 使用）、FormAttributes、MultiFormAttributes、FormSize、Rename、ChangeProgRelProgram |
| 支撑类 | 2 | `DesignerClipboardData`（剪贴板序列化，DesignerClipboardData.cs:14，内嵌 StringFormSpecModel:243、LocalSetting:392）、BooleanAttributeValue |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| DeleteComponentsUndoRedoCommand.cs:28 | 构造函数内：若删除列表恰好是 Table/Tree/Folder 的全部子节点，升级为删除容器本身并把 parent 上移（:14-33） |

## 相邻目录

- `../UndoRedoCommands/`（7 个）是并行的"规格节点内容"撤销命令命名空间：SDSpecUndoRedoCommand（替换节点
  CDATA 内容，SDSpecUndoRedoCommand.cs:10）、SpecCitedUndoRedoCommand（内嵌 SpecSource :228）、
  ActLocalStringUndoRedoCommand、ProgRelProgramAttributeUndoRedoCommand、ExcludedUndoRedoCommand、
  FormPosUndoRedoCommand（继承本目录的 AbstractUndoRedoCommand）、FormSizeComplexUndoRedoCommand。
  见 [../README.md](../README.md) 目录地图。

## 细节去哪

- 框架本体（双栈/分组）：[../../UndoRedoFramework/README.md](../../UndoRedoFramework/README.md)；被操作的对象模型：[../ViewModel/README.md](../ViewModel/README.md)
- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
