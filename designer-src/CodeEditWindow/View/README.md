# CodeEditWindow/View —— 主编辑控件与对话窗口

命名空间 `SpecDesigner.CodeEditWindow.View`：主编辑控件（CodeTextEditor / BaseTextEditor /
CodeEditorMainWindow）与约 12 个对话窗口（书签、跳转、参数、Globals、函数信息、SQL 模板、标准版对比）。
单层平铺 25 个文件。

## 目录地图

| 分组 | 代表 |
|---|---|
| 主编辑控件 | `CodeTextEditor`（**2085 行**，CodeTextEditor.cs:48；内嵌 DiffContentStructure/DiffBaseOnStandardStructure 两个私有结构类 :2038/:2059）、`BaseTextEditor`（抽象基类，:30，681 行）、`CodeEditorMainWindow`（:30，898 行）、DiffTextViewer（:22）、CodeSpecificationMainWindow（:25，实现 IAvalonFormSpecEditorLayout）、TreeView（:22，结构树） |
| 对话窗口/杂项 | ErrorCheckWindow（:31，132 行）、ButtonEdit（:8，带按钮 TextBox）、SimpleDiffTextViewer、ErrorList、JoinTableDetail、ParameterGroup、DSCTextColumn、CodeEditorWindowCommands（:7，窗口级命令） |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| BaseTextEditor.cs:91 | 构造中按偏好设置行号、制表符转空格、ToolTip 提示（ShowHintBehavior），并订阅全局搜索/替换事件（:90-107） |
| BaseTextEditor.cs:190 | `SearchKeyword(e, isPublishEvent)`：按方向/起点执行文本与正则搜索（:184-207） |
| BaseTextEditor.cs:281 | 搜索结果构造处：`EditableAreaOnly` 时跳过不可编辑区的匹配（:277-285） |
| CodeTextEditor.cs:1171 | `OnReturnToStandard`：调 `ConnectionManager.RunProgram("adzp064 <prog> <type>")` 恢复标准程序 |
| CodeTextEditor.cs:1400 | `IsEditable(editor)`：遍历选区逐偏移检查 `SectionProvider.CanInsert`，决定命令可执行性（:1400-1415） |
| CodeTextEditor.cs:1760 | `LoadSyntaxHighlighting()`：按主题 pack URI 加载 FGL_Mode_Default/DarkTheme.xshd（:1756-1775） |
| CodeEditorMainWindow.xaml.cs:211 | 构造：new CodeTextEditor + CodeEditorManager + StructureHelper，并按 IsDiff 决定布局（:207-219） |
| CodeEditorMainWindow.xaml.cs:235 | 注册 CommandBindings：Vi 模式切换、CreateFunction/DeleteFunction |
| CodeEditorMainWindow.xaml.cs:521 | CreateFunction 的 CanExecute：要求加点 IsSelfDefinition/IsNew/IsEditable |
| CodeEditorMainWindow.xaml.cs:557 | `checkBox_PreviewMouseLeftButtonDown`：SEC/ADP 模式切换开关，含 topstd 权限校验（:557-572） |
| ErrorCheckWindow.xaml.cs:31 | 用 `^ *IF` / `^ *END *IF` 正则逐行配对检查 IF/END IF 缺漏（:31-47） |

## 值得知道

- 保存链路会顺带调用后端程序：adzi520（保存登记，Helper/CodeEditorManager.cs:493 →
  [../Helper/README.md](../Helper/README.md)）与 adzp064（恢复标准，本目录 CodeTextEditor.cs:1171）。

## 细节去哪

- 编辑器总管与分区正则：[../Helper/README.md](../Helper/README.md)；上层索引：[../README.md](../README.md)
- 全树入口：[../../README.md](../../README.md)
