# SpecDesigner.FormDataEditor —— 画面数据 XML 编辑器

画面（Form）XML 数据的编辑工具窗口（RootNamespace `SpecDesigner.FormDataEditor`）：FormDataEditorWindow
用 XmlDataProvider 把画面数据 XML 绑定到主从 DataGrid（MasterDetailView.xaml.cs:127-151），DetailView
特判 `zoom` 节点（DetailView.xaml.cs:49）—— 编辑的是画面数据源/zoom 结构，不是画面布局。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 6 | FormDataEditorWindow（:12，标题 dataView_WindowTitle :15-17）、MainWindow（:13，构造中 LoadCommonData :16-19）、`MasterDetailView`（:21，369 行，四个重载构造 :124/:134/:156/:165/:200 支持层级 xpath、过滤、多选）、DetailView（:16，动态建列）、SelectionMode（:6，枚举）、VisibilityConverter（:10） |
| Helper/ | 1 | VisualHelper（:7，视觉树辅助） |
| Properties/ | 3 | AssemblyInfo、Resources.Designer、Settings.Designer（模板文件） |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| MasterDetailView.xaml.cs:134 | `MasterDetailView(string source, int layer)`：载入 XML 后按 layer 生成 `/*/*…` 的 XPath 并绑到 dataGrid（:134-151）—— layer 参数实现主从钻取 |

## 值得知道

- 整个编辑基于 XmlNode/DataGrid 而非强类型模型 —— 与其它编辑器的 ViewModel 化路线不同。

## 细节去哪

- 上层索引：[../README.md](../README.md)
