# SpecDesignerCommon/Helpers —— 静态工具层

命名空间 `SpecDesignerCommon.Helpers`：组件工厂（按 core-br/mod-fd 元数据生成带默认子件的组件）、
表单设计规则、表格列元数据缓存和一批 WPF 值转换器。全部是静态工具或无状态转换器，没有自己的文档模型。

## 主题分组（20 个 .cs）

| 分组 | 数量 | 代表 |
|---|---|---|
| 组件工厂/元数据 | 4 | `ComponentFactory`（static，1292 行，ComponentFactory.cs:12）、ComponentHelper（:12）、ComponentType（:7，组件类型枚举）、ComponentPropertyTypeEnum |
| 表单设计规则 | 5 | `FormDesignSetting`（:7，容器/formFields 白名单）、FormCommands（:7，static 命令）、GridSize、MoveDirection、PosXY |
| 数据表/库 | 3 | `TableColumnHelper`（468 行，.tbl 列信息缓存，TableColumnHelper.cs:12）、TabIndexSortHelper（:8）、CodeLibrariesHelper（:9） |
| UI 转换器 | 6 | ColsConverter、DimensionClassConverter、ForDBValueConverter、InvertBoolConverter、MessageBoxImageConverter、DataGridHelper（:12） |
| 其它 | 2 | `ReflectionHelpers`（static，GetCustomDescription(SpecStatus) 被各 Node.Create 引用，ReflectionHelpers.cs:8）、BusyIndicatorExtension |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| ComponentFactory.cs:19 | `SetSpecification(CoreBrString, modFdInfo)` 内首次缓存 _modFdInfo，随后解析 core-br 抽取 NodeInfo/PropertyInfo 元数据（:15-46） |
| ComponentFactory.cs:49 | `CreateEmptyComponentWithLabel(key, componentType)`：按组件类型生成空组件+标签组合（:49 起） |
| ComponentFactory.cs:72 | componentType==Tree 分支：为 id/parentid/isnode/expanded/name 生成 4 个 Phantom + 1 个 Edit 子组件（:70-91） |
| ComponentFactory.cs:98 | Edit/ComboBox/TextEdit/ButtonEdit/DateEdit 公共分支（可编辑单组件，不加子件）（:98-103） |
| ComponentFactory.cs:1232 | `AcceptMimes(parentNodeName, childName)`：Form 不可作父、Phantom 只进 Tree/Table，再查 _nodeInfoList 的 mimeType 判定嵌套合法性（:1232-1249） |
| FormDesignSetting.cs:114 | 静态字段 `containers`：10 种可作容器的组件名（Folder/Form/Grid/Group/HBox/Page/ScrollGrid/Table/Tree/VBox），formFields 列表在 :117-121 |
| TableColumnHelper.cs:37 | `FindTableColumns(tableName)`：从 tables 缓存取 module 名，拼 `<BasePath>/<module>/tbl/<表名>.tbl` 读列定义（:37-52） |

## 值得知道

- ComponentFactory 是**元数据驱动**的：core-br.spec + mod-fd.spec 提供组件/属性定义（ComponentFactory.cs:35-38），
  由 SettingManager 启动时加载（SettingManager.cs:598 → [../README.md](../README.md) 热点索引）。
- ReflectionHelpers.GetCustomDescription(SpecStatus) 把枚举 Description("c"/"d"/"u") 变成 XML status 文本，
  是所有 Node.Create 的标配（如 SpecFieldNode.cs:550 → [../ViewModel/README.md](../ViewModel/README.md)）。

## 细节去哪

- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
