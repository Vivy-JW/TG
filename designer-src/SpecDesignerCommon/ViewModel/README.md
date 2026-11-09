# SpecDesignerCommon/ViewModel —— 文档对象模型层

命名空间 `SpecDesignerCommon.ViewModel`，全设计器共享的数据核心：`XmlElement` 承载表单组件树，
`AbstractSpecNode` 体系承载 .tsd 规格节点树，Fgl* 系列解析 4GL 代码结构。各编辑器（FormEditor /
SpecEditor / CodeEditWindow）都在这层模型上读写，属性修改不直接改值，而是组撤销命令提交
（见 UndoRedo/）。

## 主题分组（76 个 .cs）

| 分组 | 数量 | 代表 |
|---|---|---|
| 规格节点体系 | 20 | `AbstractSpecNode`（抽象基类，AbstractSpecNode.cs:12）、`SpecFieldNode`（字段，:15）、`SpecTreeNode`（树查询，:11）、`SpecActionNode`（按钮动作，:15）、`SpecProgRelNode`（程序关联，:12）、`SpecHelpCodeNode`（帮助档，:12）、SpecProgram 抽象及 All/DB/DI/MI 四子类（:7）、`SpecNodeTransform`（规格↔表单转换，:8）、`SpecStatus`（:8） |
| 表单组件模型 | 6 | `XmlElement`（2662 行全库最大类，XmlElement.cs:20）、`FormSpecModel`（:11）、FormSpecDictionary、`ContainerMapManager`（:8）、`WidgetCommands`（:7）、`ScreenRecordManager`（:10） |
| FGL/4GL 解析 | 15 | `FglBaseNode`（抽象，FglBaseNode.cs:6）及 FglComponent/FglDimension/FglKeyword/FglParameter/FglSpecification/FglTest 等节点族 |
| 数据库/表模型 | 8 | `DatabaseSourceViewModel`（:14）、DSTableModel/DSColumnModel/DSColumnViewModel、`TBLModel`（:15）、TableAssociationModel、ModFdInfo（:8） |
| Avalon 布局/菜单接口 | 10 | IAvalonLayout 及 5 个 IAvalon*Layout、IMainCodeMenu/IMainEditMenu/IMainSpecMenu、ISpecSearchable |
| 选项/枚举 | 13 | AlignOptions、CiteEnum、CodeSpecStatus、ProgRelType、ResizeDirectionEnum、SortPosition 等 |
| 其它 | 4 | ProgRelProgram、ProgressBarModel/ViewModel、ReportSpecification（:8） |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| AbstractSpecNode.cs:355 | `SetAttribute(key,value)`：null 转空串后写回 Source 的 XAttribute（:355-362）；OnPropertyChanged(:344) 自动打 SpecStatus.MODIFY |
| SpecFieldNode.cs:60 | 名称变更校验：s_browse/Q 类程序下非 b_ 前缀发 WARNING，否则要求 l_*_desc 格式（:49-73） |
| SpecFieldNode.cs:342 | `AttributeChanged("column")`：经 TableColumnHelper.GetColumnInfo(:355) 取列属性并组 SpecAttributeUndoRedoCommand（:336-355） |
| SpecFieldNode.cs:362 | column 清空时撤销命令同步记录 column/attribute 两个旧值（:357-363） |
| SpecFieldNode.cs:389 | items 连同 i_zoom/c_zoom/default/max/min/chk_ref 一起写入撤销命令（:389-397） |
| SpecFieldNode.cs:524 | `static Create(info,name)`：按代码模板 p/r 设 can_edit=N，构造 19 属性的 `<field>` XElement（:524-580） |
| SpecFieldNode.cs:529 | 该 `<field>` 的 XAttribute 初始化列表本体（:529-551） |
| SpecFieldNode.cs:584 | `SetDefaultProperties`：widget 为 ComboBox/RadioGroup/CheckBox 时强制 req=Y（:583-593） |
| SpecFieldStringNode.cs:19 | `Create`：建 `<sfield name/text/lstr>`，lstr 取 SpecStatus.CREATE 描述（:17-25） |
| SpecHelpCodeNode.cs:327 | `Create`：建 `<hfield>`，含 help_table/help_find/help_dlang/help_field 等 11 属性（:325-342） |
| SpecTreeNode.cs:197 | `SetType`：把 "table.col" 拆开后组 SpecTreeAttributeUndoRedoCommand 并 AddThenExecute 提交（:187-201） |
| SpecActionNode.cs:281 | `AddActionType`：ActionTypes 为空抛 Message_ActionHasType，"all" 类型做展开（:277-296） |
| SpecProgRelNode.cs:92 | `CitedSpec` 重写：标准程序返回 null，否则从 CiteSTD 的 prog_rel 找同名未删除节点（:92-108） |
| SpecProgRelNode.cs:221 | `Create`：建 `<pfield>`（depend_field/program/type=1）（:219-233） |
| SpecNodeTransform.cs:70 | 表格列默认 format 值写回表单控件（:67-76） |
| SpecNodeTransform.cs:90 | `TransformFieldType`：有 colName 则 fieldType=TABLE_COLUMN，父为 s_browse 改 COLUMN_LIKE（:90-105） |
| SpecNodeTransform.cs:169 | `TransformReferenceNode`：为引用返回值生成 lbl_* 多语言标题写 title（:164-183） |
| SpecStatus.cs:8 | `[Flags] SpecStatus`：CREATE/DELETE/MODIFY = "c"/"d"/"u"，Description 直接充当 XML status 文本（:13-20） |
| DatabaseSourceViewModel.cs:41 | 构造遍历 TableAssociationModel.AliveTBLs 建 DSTableModel，以 TableColumnHelper.FindTableColumns 填列（:41-56） |
| ScreenRecordManager.cs:29 | `AddRecordField`：控件属 Table/Tree/ScrollGrid 挂父 Record，控件本身是这三类则自建 Record，其余归 "Undefined"（:29-56） |
| ScreenRecordManager.cs:59 | `AddRecordFieldTo`：无 colName 跳过，建 RecordField 并分配 fieldIdRef 回写（:59-88） |
| ScreenRecordManager.cs:77 | 给新 RecordField 调 `GetNewFieldIdRef(Records)` 取 fieldId |
| ScreenRecordManager.cs:167 | `GetNewFieldIdRef` 静态实现：收集全部 fieldIdRef 排序找空号（:165-182） |
| FormSpecModel.cs:467 | SpecNodeType 判定：button style=button_qrystr→PROGREL；ButtonEdit image=langmodify.png→MULTILANG；FFLabel tag=sync→PROGREL（:466-482） |
| XmlElement.cs:1461 | `Index` 属性：自身在父 Nodes 中的下标，无父返回 0（:1461-1471） |
| XmlElement.cs:1480 | `AddNodeAt(element,index)`：同位置短路，跨父先 RemoveNode 再 Insert（:1480-1495） |
| XmlElement.cs:1888 | `this[key]` 索引器 set：值相同直接返回，tabIndex 特殊处理后 OnPropertyChanged("")（:1888-1900） |
| XmlElement.cs:2190 | `ReplaceItems`：仅 RadioGroup/ComboBox 有效，重建 Items 并以逗号串写回 items 属性（:2190-2205） |
| XmlElement.cs:2522 | `CheckOverlapping`：PreferenceManager 关闭 ValidateForm 则跳过，容器类排除后交 ContainerMapManager（:2516-2535） |

## 细节去哪

- 撤销命令怎么包这些修改：[../UndoRedo/README.md](../UndoRedo/README.md)；列元数据从哪来：[../Helpers/README.md](../Helpers/README.md)
- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
