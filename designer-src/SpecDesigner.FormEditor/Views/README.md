# SpecDesigner.FormEditor/Views —— 画布与交互视图层

表单编辑器的画布与视图：ManagedForm（设计画布）、WidgetBox/WidgetToolItem（工具箱）、
TabIndexControl（tab 排序）、ResizeControl（尺寸手柄）及各类查看器（表单结构、数据库源、屏幕记录）。
单层平铺（20 个 .cs + 13 个 .xaml）。

## 目录地图

| 分组 | 代表 |
|---|---|
| 画布核心 | `ManagedForm`（UserControl/IDisposable，**1603 行**，ManagedForm.xaml.cs:31）、WorkSpace（:13）、ColumnCanvas（Canvas 派生，:13）、`ResizeControl`（:18，缩放手柄，263 行）、DropDirection（:6，枚举） |
| 工具箱/装饰 | `WidgetBox`（:22，186 行）、SimpleFormWidgetBox（:14）、WidgetToolItem（:13，ICommandSource/IDisposable）、WidgetAdornerView（:12）、WidgetRepeatAdorner（:15，Adorner）、HaveSpecImage（:10）、ToolBar（:20） |
| 命令 | WidgetToolbarCommands（:7）、SimpleFormWidgetToolbarCommands（:7）、TabIndexCommands（:7，internal static） |
| tab 排序 | TabIndexControl（:15） |
| 查看器 | FormStructureViewer（:22，表单结构树）、DatabaseSourceViewer（:12）、ScreenRecordViewer（:17）、FormZoomer（:15，Window，缩放） |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| WidgetBox.xaml.cs:82 | `ExecutedAdd`（:73 起）内：工具箱添加控件命令 —— DateTimeEdit 在 ErpVer=="1.0" 时禁用；参数可解析为 ComponentType 或特判 ReferenceLabel/MultiLangButtonEdit 等 SpecNodeType；最终都调 AddWidgetAdornerHelper.Show(formWindow.LayoutRoot, …) |
| WidgetBox.xaml.cs:138 | `ShrinkContainer`（:135 起）的 foreach：递归遍历 FormNode 下 Grid/HBox/VBox/Group/Folder/Page 容器，把各容器 gridHeight/gridWidth 收成 "1"；入口 ExecutedAdjustContainerBlankArea（:121）先弹繁体确认框 |
| ManagedForm.xaml.cs:716 | `CanExecutedInsertQuery`：仅当代码模板为 "Q" 时允许插入 Query(PROGREL) 控件；其前 ExecutedInsertMultiLanguage（:707-714）用 AddComponetsUndoRedoCommand + ComponentFactory.CreateEmptyComponentForBody 插 MULTILANG 控件 |
| ManagedForm.xaml.cs:851 | `ExecuteMoveToFirst`：把选中控件包进 ChangeChildIndexUndoRedoCommand 移到父容器第 0 位并提交（:860/:871/:883/:892 是 MoveToPrevious/CanMoveToNext/MoveToNext/MoveToLast 同族） |
| ManagedForm.xaml.cs:906 | `canMoveInBox`（:901 起）：判定选中控件能否在盒内移动 —— 须唯一选中、有父容器、父类型为 Folder/HBox/Table/Tree/VBox 且父下不止一个节点 |
| ManagedForm.xaml.cs:930 | `CanLayoutCommand`：布局类命令的 CanExecute 校验 —— 需有选中项、有父容器且父不是 Form，并经 ComponentFactory.AcceptMimes(HBox, 节点名) 判定可接受性 |
| ResizeControl.xaml.cs:58 | `WhenDragStarted`：拖拽缩放开始即取 DataContext 为 XmlElement，构造 ChangeSizeUndoRedoCommand + FormSizeComplexUndoRedoCommand 并 StartGroup 记入 UndoRedo；WhenDragCompleted（:61 起）在 DragCompleted 时 SetFinalSize 并 EndGroup |

## 细节去哪

- AddWidgetAdornerHelper.Show（统一"把控件放上画布"出口）：[../Helpers/README.md](../Helpers/README.md)
- 命令包装的对象：[../../SpecDesignerCommon/UndoRedo/README.md](../../SpecDesignerCommon/UndoRedo/README.md)
- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
