# Infrastructure —— 加点/分区的数据模型与事件

类库（RootNamespace `Infrastructure`），代码编辑侧的地基：加点（ADP/TAP）与分区（section）的数据模型、
程序信息单例仓库、XML 流式解析与 24 个 Prism 事件。ProjectReference 引 CodeEditor.FglAnalysis 与
SpecDesignerCommon（Infrastructure.csproj）。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 2 | LoadInformation（全局命名空间，仅 PackageKey+IsDiff 两个字段，LoadInformation.cs:5，:9-18）；Properties/AssemblyInfo.cs |
| CodeEditor/Infrastructure/ | 1 | `TreeNodeFactory`（TreeNodeFactory.cs:9）：把 AddPointModel 转成结构树 TreeItem（:14-27） |
| SpecDesigner/Infrastructure/（根） | 9 | `ResourceController`（程序信息单例仓库，ResourceController.cs:18）、EventController（:7）、`XMLParser`（带 ElementStart/Attributes 回调的流式 XmlReader 解析，XMLParser.cs:11，:13-40）、TreeItemCollection（:9）+ 4 个 XML 事件参数类（ElementEventArgs 等） |
| SpecDesigner/Infrastructure/Event/ | 24 | 全部是 Prism `CompositePresentationEvent<T>` 空壳子类（事件即类型）：ADPStatusUpdateEvent（:8）、CreateFunctionEvent（:7）、DeleteFunctionEvent（:7）、ModifyFunctionEvent（:7）、RefreshScreen（:8）、SaveAddPointEvent（:8）、ToggleDiffEditableEvent（:8）、TreeItemSort（:8）+ 载荷类 ModifyInfomation（:7，内嵌 ModifyTypeEnum :30）、CreateFunctionInformation（:6）、RemoveFunctionInformation（:6） |
| SpecDesigner/Infrastructure/Helper/ | 11 | `IntellisenseSourceFactory`（从 XML 源解析出 List<IIntellisenseModel>，:10，:13-21）、CodeSampleSourceFactory（:10）、CitedAddPointHelper（:9）、BookmarkCommands/DiffCommands/FunctionInfoCommands（各 :7）、TreeViewItemBehavior（:8）、ReflectionHelpers（:8）、FunctionTypeConverter/DefinitionTypeConverter/YesNoConverter |
| SpecDesigner/Infrastructure/Extension/ | 2 | IDragable（:6）、IDropable（:7）—— 树节点拖放接口 |
| SpecDesigner/Infrastructure/Model/ | 27 | `AddPointModel`（**1317 行全系统最重模型**：XML 元素 + 4GL 正则解析 + 权限 + 错误收集，AddPointModel.cs:18）、`ProgramInformation`（:19，1162 行）、`SectionModel`（:8，175 行）、IntellisenseModel（:11）、TreeItem（:17，同时实现 IDragable/IDropable）、ParameterModel、TAPRoot、Variable、SelfIntellisenseModel、Keyword、DiffModel、ADPModel + Status/Scope/IntellisenseEnum 等枚举 |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| AddPointModel.cs:71 | 构造初始化 `<point>` XElement，`src` 属性按 topstd 模式取 "s" 或环境 ENV（:58-76）—— 每个加点对应一个 XML 元素 |
| AddPointModel.cs:193 | Name 校验：函数名必须以程序名开头，否则发 DocumentErrorsEvent INFORMATION（:193-203） |
| AddPointModel.cs:253 | Description 校验：非 # 开头的非空行报"必须以 # 开头"（:246-261） |
| AddPointModel.cs:328 | `Content` setter：`\r\n`→`\n` 归一后按 funcReg/dialogReg/reportReg 正则识别定义类型（:326-343） |
| AddPointModel.cs:361 | FUNCTION 分支：不匹配 funcReg 抛 ComplexException，剥离 function/end function 头尾、提取函数名与参数（:359-371） |
| AddPointModel.cs:691 | `IsEditable`：综合 isIndFun/Topind、global.memo_industry、CiteSetting、standard 标志的权限判断（:691-706） |
| AddPointModel.cs:1105 | `ToXElement`：按 IsSelfDefinition 写回 Content 或 ToString()，diff 模式剥离 `\a` 标记（:1094-1111） |
| AddPointModel.cs:1200 | 反序列化：Type 非空且名字与 TAP 名不符时抛 ComplexException"TAP 格式错误"（:1193-1203） |
| AddPointModel.cs:1243 | 私有正则字段区：scopeRegex/funcReg/endFuncReg/dialogReg/reportReg（:1240-1258）—— 4gl 函数/dialog/report 定义语法 |
| SectionModel.cs:101 | `IsEditable`：G 类程序的 other_function/other_report 例外、readonly 属性、topstd 模式 src=c/s/m 规则（:101-132） |
| SectionModel.cs:151 | 构造：从 XElement 读 `src` 属性（:146-152）；ToXElement（:155-162）在 diff 模式剥离 `\a` |
| ProgramInformation.cs:655 | `Initial(name)`：按名取加点，不存在则新建并入列（:655-666）；`FullCode` 属性（:652）为编辑器全文缓存；字段区含 _addPoints/_diffAddPoints/_citeAddPoints（:1103-1112） |
| ResourceController.cs:106 | `SaveFile`：经 SettingManager/GetTzpManger 依次保存 TGL、AddPoint、UpdatedAddPoint、FullCode 并发 ADPStatusUpdateEvent（:106-115） |

## 细节去哪

- 消费方（编辑器总管/主控件）：[../CodeEditWindow/README.md](../CodeEditWindow/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
