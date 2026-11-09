# SpecDesigner.TableViewer —— 数据表/列只读查看器

表/字段查看器程序集（RootNamespace `SpecDesigner.TableViewer`）：TableViewerWindow 单例窗口经
SpecDesignerCommon 的 TableColumnHelper.GetTables() 载入表结构 XML 并按 module 分组过滤
（TableViewerWindow.xaml.cs:84-103）；ColumnViewerWindow 额外提供"复制全部列名/复制到 record"命令
（ColumnViewerWindow.xaml.cs:59-61）。**只查看、不编辑**表结构。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 3 | TableViewerWindow（:21，双检索单例 This :23-34）、ColumnViewerWindow（:22，单例，订阅 LoadSpecReferFilesEvent 重载 :43-56）、TableColumnCommands（:12，复制列名/record 命令） |
| Helper/ | 2 | BooleanConverter（:8）、DataTypeConverter（:12）—— 列显示转换 |
| Properties/ | 1 | AssemblyInfo.cs |

## 值得知道

- 数据源在另一个程序集：SpecDesignerCommon/Helpers/TableColumnHelper（FindTableColumns 的热点见
  [../SpecDesignerCommon/Helpers/README.md](../SpecDesignerCommon/Helpers/README.md)）。

## 细节去哪

- 上层索引：[../README.md](../README.md)
