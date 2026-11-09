# SpecDesigner.Output —— 错误/警告输出面板

输出面板程序集（RootNamespace `SpecDesigner.Output`）：OutputPanelControl 后台线程订阅全局
DocumentErrorsEvent，把错误/警告/信息汇集到 ObservableCollection 展示；Tzp 文件关闭时清理对应条目
（OutputPanelControl.xaml.cs:26-38）。纯展示组件，无业务规则。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 3 | OutputPanelControl（:20）、DocumentErrorsSource（空的 ObservableCollection<DocumentErrorsEventArgs>，:8-10）、ErrorComparer（:8，IEqualityComparer 去重） |
| images/ | 0 | deleteall/error/information/warning 四个 png（面板按钮与级别图标） |
| Properties/ | 3 | AssemblyInfo、Resources.Designer、Settings.Designer（模板文件） |

## 值得知道

- 与事件生产端构成**生产-消费对**：DocumentErrorsEvent 的发布点在 AddPointModel.cs:202 与
  CodeEditorManager.cs:482（见 [../Infrastructure/README.md](../Infrastructure/README.md) 与
  [../CodeEditWindow/Helper/README.md](../CodeEditWindow/Helper/README.md)）、
  SpecFieldNode.cs:60（见 [../SpecDesignerCommon/ViewModel/README.md](../SpecDesignerCommon/ViewModel/README.md)）。

## 细节去哪

- 上层索引：[../README.md](../README.md)
