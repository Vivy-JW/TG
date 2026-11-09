# SpecDesigner.SpecEditor —— 规格树编辑器

类库（csproj:7），"规格（.tzs 等）树状结构"的编辑器：根目录的 `SpecEditor` 用户控件是规格树编辑主界面
（SpecEditor.xaml.cs:21，392 行），`SpecPropertyEditor` 是右侧属性面板（SpecPropertyEditor.xaml.cs:31，
1292 行），`SpecificationViewWindow` 提供整规格只读查看。

## 目录地图

| 子目录 / 根散文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 7 | `SpecEditor`（规格树编辑主控件，:21）、`SpecPropertyEditor`（属性编辑面板，:31；ComponentPropertyVisibilityMap_Init 在 :76）、SpecificationViewWindow（:23）、MenuCommands（:10，仅 43 行 SetCiteCommand —— 与 T100Designer 那份 3924 行的同名文件无关）、SpecPropertyCommands（:9）、SpecNodeLoadedEvent/EventArgs（:7/:8，Prism 事件对） |
| Helpers/ | 43 | 转换器/行为工具箱 → [README.md](./Helpers/README.md) |
| Views/ | 11 | 数据编辑弹窗层 → [README.md](./Views/README.md) |
| Controls/ | 10 | 表格/字段模型控件：Tables（:11）、TblModel（:17）、SRModel（:8）、BaseModel（:8）、ModelHierarchyDataTemplateSelector（:8）、NumericTextBox、ResetComboBox、ResetTextBox、ObjectToTypeStringConverter、TableSchemaAssociationWindow |
| ViewModel/ | 3 | 泛型数据源（`where T : AbstractSpecNode`）：FieldsSourceViewModel、SpecNodesSourceViewModel（:12）、UncitedSpecNodesViewModel（:9） |
| SpecEditor/ | 1 | MainWindow（:11）：独立运行时的主窗口壳（与根的 UserControl 主控件区分层级） |
| Themes/ | 0 | Generic.xaml 控件样式字典 |
| images/ | 0 | 19 个文件 |
| Properties/ | 1 | AssemblyInfo.cs |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| SpecPropertyEditor.xaml.cs:272 | `ComponentPropertyVisibilityMap_Init`（:76-279）填充静态字典的段落：逐控件类型（Reference/Edit/ButtonEdit/CheckBox/ComboBox/DateEdit/TextEdit/Label/ProgRelField）登记 属性→"L"（Label 形式显示）映射，决定属性面板里哪些属性可见/可编辑；:272 附近是 `ComponentPropertyVisibilityMap.Add("Reference", dictionary)` 一类调用（:268-274） |
| SpecPropertyEditor.xaml.cs:836 | `SRAttributesCB_Click`（:821-852）：SR（栏位 insert/delete/append）复选框点击后，从 SpecificationInfo.AssociateTable.Source 找本窗体对应 `sr` 节点（排除 DELETE 状态），按 Tag 把 Y/N 写回属性 |

Helpers/ 与 Views/ 的热点在各自 README。

## 细节去哪

- 子包：[./Helpers/README.md](./Helpers/README.md) · [./Views/README.md](./Views/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
