# T100Designer/SpecDesigner —— 主 EXE 的 UI 子树

主窗口、应用入口与全局命令所在：Main/ 是入口与主窗体，根下 15 个 Window/UserControl 是各功能弹窗
（函数列表、书签、快捷键、上传选项、欢迎页等），ViewModels/ 是基于 AvalonDock 的多文档工作区模型
（LayoutInitializer.cs:10 实现其 ILayoutUpdateStrategy）。

## 目录地图

| 子目录 / 根散文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 23 | 窗口群：BaseDataViewWindow（:16）、BookmarksWindow（:15）、CustomAdpListWindow（:21）、EditorWorkspaceList（:12）、FunctionList（:21）、InformationWindow（:15）、ProgramInfomationWindow（:14）、ReportSpecificationWindow（:14）、ShortcutListWindow（:12）、StartArgSetting（:14）、TabItemHeader（:15）、UploadSpecOptions（:12）、ViCommandListWindow（:12）、WelcomeWindow（:12）、`CodeTemplateSelector`（:20）+ 代码模板（CodeTemplate.cs、CodeTemplateComparer.cs:7 去重比较器）；AvalonDock 支撑：LayoutInitializer（:10）、PanesStyleSelector（:9）、PanesTemplateSelector（:9）、NumbericValidationRule（:10）、DummyArgModel、FunctionListCommands（:7） |
| Main/ | 5 | `SpecDesignerApp`（应用入口，SpecDesignerApp.xaml.cs:31，OnStartup :44 载入设置/多语言，单实例二次启动转发 SignalExternalCommandLineArgs :242）、SpecDesignerMainWindow（:34，523 行）、SpecDesignerInit（:12）、SpecDesignerSplashScreen（:13）、AvalonTitleBinding（:7） |
| Commands/ | 1 | `MenuCommands`（3924 行：全部菜单 RoutedCommand 定义与执行体，MenuCommands.cs:34；含调后端 adzp085 等逻辑） |
| ViewModels/ | 10 | 文档工作区：`FileViewModel`（抽象基类，FileViewModel.cs:12）← PaneViewModel（:7），派生 Code/CodeSpec/Form/ReportSpec 四类文档 ViewModel（CodeViewModel.cs:12、FormViewModel.cs:11、CodeSpecViewModel.cs:10、ReportSpecViewModel.cs:9）；`EditorWorkspace`（工作区/布局，:16）、MenuItemViewModel、SettingViewModel、WelcomeViewModel |
| Converters/ | 6 | XAML 值转换器：ActiveDocumentConverter（:9）、HistoryConverter、ObjectToTypeStringConverter、StringToImageConverter、ValueCheckedConverter、VisbilityToBooleanConverter |
| Helper/ | 2 | FileViewModelFactory（:8，文档 ViewModel 工厂）、ActiveDocoucmentConverter（:9，类名拼写即如此；与 Converters/ 的 ActiveDocumentConverter 疑似重复定义） |
| Views/ | 1 | AnchorableAdapterViewer（:7，AvalonDock 可停靠内容适配） |
| Properties/ | 1 | Settings.Designer.cs（用户设置） |
| Themes/ · images/ | 0 | 属上层 EXE 根（见 [../README.md](../README.md)） |

## 热点索引（引擎注释引用的本子树文件）

| 位置 | 那里是什么 |
|---|---|
| Commands/MenuCommands.cs:2839 | "Free Style（自由样式）"命令处理段（:2824-2856）：确认警告后按 TzpType 分派 —— ReportCode/Code 调 `ConnectionManager.RunProgram("adzp085 …")` 后端转换，Form 调 SpecificationInfo/TzpManager 的 SetFreeStyle()。注意 SpecDesigner.SpecEditor 另有一个同名 MenuCommands.cs 只有 43 行（SetCiteCommand），2839 行的热点落在本文件 |
| CodeTemplateSelector.xaml.cs:46 | 读合法代码模板：从 `<workspace>/mta/code_template.xml`（:52）XElement.Load 后遍历 `kind` 元素，按 codeGroup1/codeGroup2 与当前模板同组过滤，生成 "id.desc" 列表；文件缺失弹 Message_CodeTemplateFileNotFound（:46-75） |

## 细节去哪

- 上层：[../README.md](../README.md)（EXE 根）；全树入口：[../../README.md](../../README.md)
- 文档模型（FileViewModel 族操作的对象）：[../../SpecDesignerCommon/ViewModel/README.md](../../SpecDesignerCommon/ViewModel/README.md)
