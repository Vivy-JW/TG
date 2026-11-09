# CodeEditWindow/Helper —— 编辑器业务中枢

命名空间 `SpecDesigner.CodeEditWindow.Helper`：编辑命令、diff 渲染、校验规则、行为（Behavior）与
转换器。单层平铺 32 个文件。中枢是 `CodeEditorManager`（1717 行）：事件订阅、保存、生成 TGL 都在这。

## 主题分组

| 分组 | 代表 |
|---|---|
| 总管 | `CodeEditorManager`（CodeEditorManager.cs:27；构造 :36-41 绑定 PackageKey 并订阅 LoadedSettingEvent） |
| 命令 | CodeEditCommands（:8，自定义 RoutedCommand 集合） |
| diff | DiffRenderer（:13，IBackgroundRenderer）、SearchResultBackgroundRenderer、TextSegmentComparer、MarkedSegment（:6） |
| 解析/生成 | `FglParserQuickHelper`（:7，ParseFunction 解析函数体行号）、`FunctionGenerator`（:10，Generate(FglSpecification) 生成带 # 注释头的 4gl 函数骨架 :10-30）、StructureHelper（:15，光标位置驱动的结构联动） |
| Behavior | BookmarkTrackerBehavior（:19，Behavior<CodeTextEditor>）等 |
| 校验/转换器 | EditorLineValidationRule、EditorDiffLineValidationRule；DataTypeConverter、ProgramTypeConverter、SwitchAccountMode* 两个等 8 个 IValueConverter |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| CodeEditorManager.cs:48 | `SubscribeEvent()` 内订阅全局 SaveSettingEvent→SaveFile：保存流程中枢（构造绑定见 :36-41） |
| CodeEditorManager.cs:278 | `SaveFile`：把编辑器全文写回 `programInfo.FullCode`；diff 模式下按设置提示保存 diff 源（:269-294） |
| CodeEditorManager.cs:445 | 遍历 SEC 段内 EditObject，命中 `addPointModel.IsMarkHard` 时记录待清除标记行 —— 保存后清理加点硬标记（:430-449） |
| CodeEditorManager.cs:493 | 保存收尾：ENV=="c" 且已 Booking 时调 `ConnectionManager.RunProgram("adzi520 <prog>")` 触发后端登记（:491-494） |
| CodeEditorManager.cs:1093 | `ProcessSections()`：校验 section 开始/结束正则匹配数量一致，否则抛"无法读取 section"（:1083-1096） |
| CodeEditorManager.cs:1238 | `ProcessEditableArea`：对自定义加点用 `FglParserQuickHelper.ParseFunction` 解析出函数体行号并 AddChild（:1230-1246） |
| CodeEditorManager.cs:1694 | 字段区定义 `SectionStartPattern = "({<section\s+id=\"(?<name>\\S*)\".*>})"` 等 section 正则（:1694-1700）—— TGL 分区语法定义 |
| TextChangedBehavior.cs:31 | `Document_Changed`：文档变更时若当前是 ADP 模式，把对应 AddPointModel 状态置为 MODIFY |
| CustomizedRoutedCommand.cs:7 | 继承 RoutedCommand 仅增加可赋值的 `Parameter` 属性，用于命令间传递 AddPointModel |

## 细节去哪

- 加点模型（AddPointModel/SectionModel）：[../../Infrastructure/README.md](../../Infrastructure/README.md)
- 主编辑控件：[../View/README.md](../View/README.md)；上层索引：[../README.md](../README.md)
- 全树入口：[../../README.md](../../README.md)
