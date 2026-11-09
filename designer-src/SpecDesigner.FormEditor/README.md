# SpecDesigner.FormEditor —— 表单所见即所得编辑器

类库（csproj:7），T100 画面（.tzp Form）的布局编辑器：`ManagedForm` 是核心设计画布（1603 行），
`WidgetBox` 是控件工具箱，DBStructure/ 支持从数据库表结构反向生成控件，Relation/ 提供表关系图。
结构性修改（增删/移动/缩放）一律经 UndoRedoManager 命令包装，贯穿 ManagedForm、ResizeControl、
ProgRelProgramsDataGrid 等处。

## 目录地图

| 子目录 / 根散文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 5 | `FormEditorMainWindow`（主窗体，实现 IAvalonFormPropertiesLayout/IAvalonFormSpecEditorLayout/IAvalonFormStructureLayout，FormEditorMainWindow.xaml.cs:20）、ToolBarItem（:14）、DragDropScrollViewer（:10，DragDirection 枚举 :135）、FormDataTemplateSelector（:10）、MouseUtilities（:10） |
| Views/ | 20 | 画布与交互视图层 → [README.md](./Views/README.md) |
| Helpers/ | 15 | 辅助层（控件添加出口、tab 序服务）→ [README.md](./Helpers/README.md) |
| Behaviors/ | 11 | Interaction Behavior 群：DragSelectionBehavior（:16，画布框选）、DragDropBehavior、DatabaseSourceDragBehavior（:17）、GeneroComponentSelectionBehavior（:11）、MouseWheelScaleBehavior（:11）、IgnoreMouseWheelBehavior（:9）、KeyBehavior、AdornedControlBehavior（:11）、RepeatAdornerBehavior（:10）、TreeViewItemHelper（:8）、DragSelectionRectAdorner（:9） |
| Converters/ | 9 | 表单布局转换器：ResizeDirectionConverter、TabIndexBackground/VisibilityConverter（tab 序号标色）、AggregateBackgroundConverter、IsCitedForegroundConverter、BooleanOrConverter、ActionVisibilityConverter、FormVisibilityConverter、IsEqualOrGreaterThanConverter |
| DBStructure/ | 6 | 从库表生成 UI：`DBStructureCreator`（向导 Window，DBStructureCreator.xaml.cs:20，323 行）、DBStructureUISelector（:16）、`UICreator`（生成器，:13，342 行）、ContainerViewModel（:7）、ContainerType（:6）、PrepareAddColumn（:7）+ images/ 14 个 |
| Relation/ | 8 | 表关系图：RelationshipSpace（画布，RelationshipSpace.xaml.cs:27）、TBLControl（实体框，:21）、TBLRelation（连线，:14）、TBLModifyBox（:20）、RelationshipCommands（:7）、ConnectionType（:6）、TableComboBoxItem（:7）、IEntityControl |
| ActionDefaults/ | 4 | 按钮动作默认值编辑：ActionDefaults（容器，:18）、ActionDefault（单条，:18）、ActionDefaultGroup（分组，:14）、ActionDefaultCommands（:12） |
| Events/ | 3 | CancelAllActivesEvent（:8，CompositePresentationEvent<PackageKey>）、ComponentMeasureChangedEventArgs（:8）、Parse4fdEventArgs（:7） |
| Petzold/Media2D/ | 4 | Charles Petzold 书中 Media2D 箭头绘图示例的移植（ArrowLine/ArrowPolyline/ArrowLineBase :9 : Shape/ArrowEnds，命名空间 SpecDesigner.FormEditor.Petzold.Media2D），专门给 Relation/ 画关系连线 |
| Test/ | 2 | StringToXamlConverter（:11）、TestPropertyWindow（:12）—— 遗留的调试/试验窗口，**不是单元测试** |
| ViewModels/ | 1 | TabIndexMenuItemViewModel（:8，tab 排序菜单项） |
| Themes/ | 0 | Generic.xaml：157 个 x:Key 的表单控件模板大字典 |
| images/ · Properties/ | — | 121 个图标文件（另有 DBStructure/images/ 14 个）；AssemblyInfo.cs |

## 热点索引（引擎注释引用的、无子 README 子目录里的文件）

Views/ 与 Helpers/ 的热点在各自 README；本表收 DBStructure/ 的两个。

| 位置 | 那里是什么 |
|---|---|
| DBStructure/DBStructureCreator.xaml.cs:178 | `FinishButton_Click`（:177 起）：向导完成时校验选中字段非空后调 `UICreator.Create(LayoutRoot, uiSelector.Container, 选中字段集, ProgramKey)`，把所选库列按容器类型（None/Grid/Group/ScrollGrid…）批量生成控件 |
| DBStructure/UICreator.cs:16 | `Create` 静态方法体开头：按 containerViewModel.Container.Type 分派到 CreateNoneContainerWidget/CreateGridContainer/CreateGroupContainer 等私有生成器，产出 List<XmlElement> 后统一交 AddWidgetAdornerHelper.Show 展示到画布 |

## 细节去哪

- 子包：[./Views/README.md](./Views/README.md) · [./Helpers/README.md](./Helpers/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
