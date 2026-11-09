# CodeEditWindow —— 4GL 代码编辑窗口

基于 AvalonEdit 的代码编辑程序集（RootNamespace `SpecDesigner.CodeEditWindow`）：FGL（.4gl）源码的
分区（section）编辑、加点（ADP/TAP）管理、diff 对比与语法高亮。ProjectReference 引
CodeEditor.FglAnalysis、DifferenceEngine、Infrastructure 三个内部程序集（CodeEditWindow.csproj）。
编辑区不是普通文本：Document 上有 SectionProvider 管理 SEC/ADP 可编辑分段
（CodeEditorManager.cs:430-448、CodeTextEditor.cs:1400-1415）。

## 目录地图

| 子目录 / 根散文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 3 | CodeMenuCommands、`DiffManager`（双编辑器 diff 对比管理，DiffManager.cs:17）、IComparisonWindow（:7，暴露 Source/Target 两个 DiffTextViewer） |
| Helper/ | 32 | 编辑器业务中枢 → [README.md](./Helper/README.md) |
| View/ | 25 | 主编辑控件与对话窗口 → [README.md](./View/README.md) |
| Model/ | 3 | CodeCompletionData（AvalonEdit ICompletionData 智能感知条目，CodeCompletionData.cs:11）、CommandModel（:8）、ContextMenuCollection（:13） |
| Extension/ | 4 | 编辑器拖放行为：AllowDropAdorner、DragDropExtension（:9，static）、MyDragBehavrior、MyDropBehavior |
| 根的两个 .xshd | 0 | `fgl_mode_default.xshd` / `fgl_mode_darktheme.xshd`：AvalonEdit（SharpDevelop）格式的 FGL 语法高亮定义（`<SyntaxDefinition name="FGL" extensions=".4gl" …>`（第 2 行，首行是 XML 声明）），分别用于浅色/深色主题，运行时按 pack URI 切换（CodeTextEditor.cs:1756-1775） |
| Themes/ | 0 | Generic.xaml，引用 PresentationFramework.Aero（:1-3） |
| images/ · Properties/ | — | 6 个 png（delete/error/folder/lock/unlock/viewmag）；AssemblyInfo.cs |

热点索引：本层散文件没有被引擎注释引用；引用集中在 Helper/（CodeEditorManager 等 3 件）与
View/（CodeTextEditor 等 4 件），见各自 README 的热点索引。

## 细节去哪

- 子包：[./Helper/README.md](./Helper/README.md) · [./View/README.md](./View/README.md)
- 语法/AST 与 diff 算法：[../CodeEditor.FglAnalysis/README.md](../CodeEditor.FglAnalysis/README.md) · [../DifferenceEngine/README.md](../DifferenceEngine/README.md)；加点模型：[../Infrastructure/README.md](../Infrastructure/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
