# SpecDesigner.Search —— 全局查找/替换面板

查找/替换控件程序集（RootNamespace `SpecDesigner.Search`）：全局 SearchBox/ReplaceBox 面板与搜索命令。
**实际的文本搜索逻辑不在本程序集** —— 在 CodeEditWindow 的 BaseTextEditor（BaseTextEditor.cs:184-207），
两者经全局 EventAggregator 的 SearchKeywordEvent/ReplaceKeywordEvent 解耦（BaseTextEditor.cs:103-107）。
仅引用 SpecDesignerCommon。本项目内少见的标准 MVVM（Control 与 ViewModel 分离）。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 4 | SearchBoxControl（SearchBoxControl.xaml.cs:22，ContentID="SearchBox"，IsSearchSpec 属性）、ReplaceBoxControl（:21）、SearchCommands（:7，static 懒加载 RoutedCommand）、ReplaceCommands（:7） |
| ViewModel/ | 2 | `SearchBoxViewModel`（管理 SearchHistory 与 "x hits" 计数，SearchBoxViewModel.cs:12，:31-42）、ReplaceBoxViewModel（:13） |
| Helper/ | 2 | MatchColorConverter（:9，命中高亮）、MatchStringConverter（:9） |
| Properties/ | 3 | AssemblyInfo、Resources.Designer、Settings.Designer（模板文件） |

## 细节去哪

- 搜索的真正实现：[../CodeEditWindow/View/README.md](../CodeEditWindow/View/README.md)（BaseTextEditor 热点）
- 上层索引：[../README.md](../README.md)
