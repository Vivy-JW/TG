# UndoRedoFramework —— 通用撤销/重做框架

通用命令框架（RootNamespace `UndoRedoFramework`）：`UndoRedoManager` 维护 Undo/Redo 双栈，命令可按组
（GeneralComplexCommand）聚合，栈深可配。SpecDesignerCommon 的撤销命令库（UndoRedo/、UndoRedoCommands/）
与表单编辑器的拖拽缩放（StartGroup/EndGroup）都建在它上面。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 2 | `UndoRedoManager`（UndoRedoManager.cs:7，238 行）、UndoRedoStack（:8，Push/Pop/索引器的简单栈） |
| Commands/ | 4 | IUndoRedoCommand（:6，Undo/Execute/Clear 三方法）、IGroupUndoRedoCommand（:6）、`GeneralComplexCommand`（:8，持 MainCommand 与子命令列表 :10-36）、AbstractComplexTriggerUndoRedoCommand（:6，abstract） |
| Properties/ | 3 | AssemblyInfo、Resources.Designer、Settings.Designer（模板文件） |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| UndoRedoManager.cs:17 | `MAXSTACKSIZE` 属性（-1 表示不限）；AdjustUndoStack 超限时对最老命令 Clear 并移除（:17-23、:142-149） |

## 对外面（怎么用）

`new UndoRedoManager(maxStackSize)`（:20-23）→ `Init()` 清双栈（:26-30）→ `AddThenExecute(cmd)` 执行并入栈
（:135-139）；`StartGroup/EndGroup(complexCommand)` 把多条命令合并成一组入 Undo 栈（:33-78，EndGroup 支持
两个不同 complexCommand 合并 ：44-60）；`Undo()` 弹栈执行并压 Redo 栈（:81-108）、`Redo()` 反向（:111-132，
其中 AbstractComplexTriggerUndoRedoCommand 不回压 Undo 栈 ：124-127 —— 触发器类命令的特殊语义）；
`UndoStackChanged` 事件（:12）通知 UI 刷新按钮状态。

## 细节去哪

- 命令库（谁在用它）：[../SpecDesignerCommon/UndoRedo/README.md](../SpecDesignerCommon/UndoRedo/README.md)
- 上层索引：[../README.md](../README.md)
