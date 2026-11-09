# SpecDesignerCommon/Events —— Prism 事件定义层

命名空间 `SpecDesignerCommon.Events`：48 个继承 `CompositePresentationEvent<T>` 的发布/订阅事件类 +
17 个随行参数/枚举/结果模型。事件本体都是空壳子类（事件即类型，声明多在类第 7 行左右），经
`EventAggregatorManager` 的两级聚合器流通（全局 `Global` 与按 PackageKey 的每文档实例，
EventAggregatorManager.cs:11,22）。

## 主题分组（65 个 .cs）

| 分组 | 数量 | 代表 |
|---|---|---|
| 事件类 | 48 | 文档/设置生命周期约 10（SaveSettingEvent.cs:7、TzpFileLoaded.cs:7、LoadedSettingIncluding* 四件、ActiveDocumentChanged）；搜索/替换约 20（SearchKeyword、ShowSearchBox/ReplaceBox、Replace* 族、SearchFromCaret/SearchFromFocus）；Diff 比对约 8（DiffFunctionSync、RefreshDiffResult、SaveDiffBaseOnStandard 等）；选择类 4（ComponentSelectedEvent.cs:7、MultiComponentSelected、FunctionSelected、ProgramSelectionChanged）；错误/属性 5（DocumentErrorsEvent.cs:7、SpecPropertiesChanged、FormPropertyChanged、HideSpecProperties、SaveSqlResult）；Tab/其它（HideTabIndex、RunCommand、ConnectionStatusChanged、InsertionCode） |
| 参数/枚举/模型 | 17 | DocumentErrorsEventArgs.cs:6、ErrorsType.cs:6（WARNING/ERROR 分级）、SearchKeywordEventArgs.cs:6、ReplaceAllKeywordEventArgs.cs:6、SpecPropertiesChangedEventArgs.cs:7、MultiSelectionArgs、DiffType、CompareResult 等 |

## 与其它层的联动（无被引热点，但发布点都在别的 README 里）

- `SaveSettingEvent` 的发布点在 SettingManager.cs:531，订阅点在 SpecificationInfo.cs:148 → [../README.md](../README.md) 热点索引
- `DocumentErrorsEvent` 的组包在 SpecificationInfo.cs:374 与 SpecFieldNode.cs:60 → [../ViewModel/README.md](../ViewModel/README.md) 热点索引
- 消费端：SpecDesigner.Output 的 OutputPanelControl 订阅 DocumentErrorsEvent 展示错误 → [../../SpecDesigner.Output/README.md](../../SpecDesigner.Output/README.md)

## 细节去哪

- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
