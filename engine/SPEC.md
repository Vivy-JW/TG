# T100 `.tzs` 格式规格（代码推导版）

> 本文全部结论来自源码推导，非样本猜测。每条都标了出处。
> 样本（`D:\t100_wrok_dir\hengshuo\prd` 里的 63 个 `.tzs` 及三组前后对）只用于**验收**，不作为推导依据。
> 已知与样本的偏差单独列在 §9。

---

## 1. 容器层

`.tzs` 是标准 ZIP。类型由扩展名决定（`TzpManager.Type`，TzpManager.cs:154-181）：

| 扩展名 | TzpType | 必需条目 |
|---|---|---|
| `.tzs` | Form | `.tsd` `.4fd` |
| `.tzv` | Form（简单表单） | `.tsd` `.4fd` `.4fdref` |
| `.tzc` | Code | `.tap` `.tgl` |
| `.tzx` | Code（差分） | `.tap` `.tgl` `.src` `.apt` |
| `.tzf` | Code（独立函数） | `.tap` `.tgl` |
| `.tzd` | CodeSpec | `.csd` |
| `.tzr` | ReportSpec | `.rsd` |
| `.tzt` / `.tzg` | Report / ReportCode | `.tap` `.tgl` |

必需条目缺失会抛 `FileNotFoundException`（`PackageManager.Unpacking`，PackageManager.cs:194）。

可选条目：`.tap2` `.tsd2` `.4gl` `.4ad` `.merge` `.bdx` `.xml` `.src` `.apt` `.str`，外加每包一个 `ver`（纯文本，首行读作版本，`SeekReleaseVersion`）。

**重打包行为**（`PackageManager.Packing`）：
- 不认识的条目原样透传（IL_08DB 分支）
- 所有条目 `DateTime` 刷成 `DateTime.Now`，统一 `SetLevel(3)` 重压
- 按 13 种扩展名分派重写，`.bdx` 落盘名是 `ProgramName + ".bdx"`
- **结论：设计器的保存不是最小改动，CLI 应当只重写变更条目并保留原始压缩字节与时间戳**

---

## 2. 组件词汇

来自工作区 `<workspace>/mta/`：

| 文件 | 内容 | 加载点 |
|---|---|---|
| `core-br.spec` | Genero 基础规格（Four Js 版权） | `SettingManager.LoadCommonData`，SettingManager.cs:598 |
| `mod-fd.spec` | 自定义表单定义 | 同上，:601 |
| `tsd.xsd` | `.tsd` 的 XSD（**已与实现脱节，见 §9**） | 同上 |
| `tables.xml` `datatypes.xml` `zooms.xml` `prog_rel.xml` | 领域数据 | 同上 |
| `tiptop.4ad` | action defaults | 同上，:604 |
| `ver` | 工作区版本，**必须等于设计器主次版本** | `SettingManager.Version` = `Assembly.GetEntryAssembly().GetName().Version`（SettingManager.cs:441） |

`ComponentFactory.SetSpecification(coreBr, modFd)`（ComponentFactory.cs:19）把两者合成目录：`NodeInfoList` + `PropertyInfoList`。实测 **57 个 NodeInfo / 295 个 PropertyInfo**。

`mimeType` 两组：`BR/*`（Record / RecordField / Relation / Root）与 `modFD/*`（表单元素）。

---

## 3. 一个字段的三份表示

| # | 位置 | 标识 | 关键属性 |
|---|---|---|---|
| 1 | `.tsd` `<field>` | `name` | `table` `column` `widget` `type` `attribute` `req` `can_edit` `status` |
| 2 | `.4fd` `<Record>` 下的 `<RecordField>` | `name` + `fieldIdRef` | `fieldType` `sqlTabName` `colName` `sqlType` |
| 3 | `.4fd` 布局元素（`Edit`/`ButtonEdit`/…） | `name` + `fieldId` | `sqlTabName` `colName` `fieldType` `widget` |

关联：布局元素的 `fieldId` ↔ 同 `<Record>` 作用域内 `RecordField.fieldIdRef`。
`.tsd` 的 `<field>` 与 `.4fd` 通过 `name` + `table`/`column` 对应，**无 id 关联**。

### 3.1 命名规则（代码推导）

布局元素的 `name` 有三种形态，由字段是否绑定数据列决定：

- 绑定真实列：**`{sqlTabName}.{colName}`**，如 `pmdl_t.pmdlud001`
- 绑定但不是完整列（见下）：`COLUMN_LIKE`，名字可任意
- 表单局部字段：任意名，`fieldType=NON_DATABASE`

`fieldType` 的**判定规则**（`SpecNodeTransform.TransformFieldType`，SpecNodeTransform.cs:70）：

```
colName 为空                       → NON_DATABASE
colName 非空:
    父节点名以 "s_browse" 开头      → COLUMN_LIKE
    元素名 != "{sqlTabName}.{colName}" → COLUMN_LIKE
    否则                            → TABLE_COLUMN
```

标签与注释的命名（`TransformReferenceNode`，SpecNodeTransform.cs:169；`TransformProgRelNode`，:132）：

```
formElement.title   = "lbl_" + <列名>
formElement.comment = "cmt_" + <列名>
```

**注意**：`lbl_` / `cmt_` 来自这里，**不是** `ComponentFactory.GetNewName`。
裸拖控件（未绑定）走的是另一条通用命名路径，得到 `{控件名}_{N}` 与 `{控件名}_{N}_1`。

---

## 4. `.tsd` 数据结构

由 `tsd.xsd` + 各 `Create` 方法共同确定。

### 4.1 根元素 `<spec>`

必需属性：`prog` `std_prog` `erpver` `ver` `module` `booking` `class` `env` `zone` `type` `identity` `designer_ver` `pre_compile` `topind`

子元素（`tsd.xsd`）：`toolbar` `table` `all` `db_all` `di_all` `mi_all` `tree` `field` `prog_rel` `ref_field` `multi_lang` `help_code` `act` `strings` `sa_spec` `exclude`

### 4.2 各节点的规范属性表

**`<field>`** —— `SpecFieldNode.Create`（SpecFieldNode.cs:529），20 个属性，顺序固定：

```
src ver column name table attribute type req i_zoom c_zoom
chk_ref items default max min can_edit can_query widget cite_std status
```

两个分支：`code_template` 为 `p`/`r` 时 `can_edit="N"`，否则 `"Y"`；两者 `can_query="Y"`。
初始值见代码，关键默认：`column="" table="" widget="" type="" cite_std="N"`，`status=CREATE`。

**`<hfield>`** —— `SpecHelpCodeNode.Create`（SpecHelpCodeNode.cs:327），11 个属性：

```
src ver name help_table help_find help_dlang help_field mapping_widget help_wc cite_std status
```

**`<pfield>`** —— `SpecProgRelNode.Create`（SpecProgRelNode.cs:221），8 个属性：

```
src ver name depend_field program type="1" cite_std status
```

注意属性名是 `depend_field`（XSD 里误写为 `depend_filed`）。

**`<sfield>`** —— `SpecFieldStringNode.Create`（SpecFieldStringNode.cs:19），3 个属性：

```
name text lstr
```

**`<rfield>`**（`ref_field` 下）：`cite_std ver name depend_field correspon_key ref_table ref_fk ref_dlang ref_rtn ch status`

### 4.3 `status` / `src` / `lstr` 的取值

**`SpecStatus` 是位标志枚举**（SpecStatus.cs:8），落盘值是 `[Description]`：

| 枚举 | 落盘 | 含义 |
|---|---|---|
| `NULL` = 0 | — | 无变更（属性缺失时即此值） |
| `CREATE` = 2 | `"c"` | 新建 |
| `DELETE` = 4 | `"d"` | 删除 |
| `MODIFY` = 8 | `"u"` | 修改 |

**`CREATE` 从不落盘**（全语料 0 次）——它是纯内存瞬态。新建字段在保存前必然被改成 `"u"`。

**`src` 不是常量**，是 `info.Env`（`SpecFieldNode.Create` 首参 `info.Env`）。
`IsTopstdMode` 为真时写 `"s"`，否则写环境标识（见 `AddPointModel.cs:71` 同款逻辑）。
语料里 `src="s"` 5449 次、`src="c"` 655 次。

**`lstr`** 在 `<sfield>` 上，同样取 `SpecStatus` 的落盘值。

---

## 5. 三个映射方向

`SpecNodeTransform.cs`（206 行）是双向映射的完整定义。

### 5.1 `.4fd` → `.tsd`（`TransformFormToSpecField`，:118）

```
若 元素.fieldType ∈ {TABLE_COLUMN, COLUMN_LIKE}:
    spec.table  = 元素.sqlTabName
    spec.column = 元素.colName
```

**只有这两个属性被搬运。** 其余（`widget` `type` `attribute`）的来源尚未在源码中定位，见 §8。

### 5.2 `.tsd` → `.4fd`

| 方法 | 出处 | 规则 |
|---|---|---|
| `TransformTableColumn` | :47 | `sqlTabName` ← `spec.Table`；`colName` ← `spec.Column`；`format`/`case` ← `TableColumnHelper.GetDefaultAttributeValue(table, column, …)` |
| `TransformCanEdit` | :38 | `noEntry` ← `spec.CanEdit`（`Y` → `"false"`，否则 `"true"`） |
| `TransformRequired` | :15 | `IsRequired` ← `spec.Req == "y"` |
| `TransformReferenceNode` | :169 | `title` ← `"lbl_" + RefRtn`；`comment` ← `"cmt_" + RefRtn` |
| `TransformProgRelNode` | :132 | `text` ← `"lbl_" + depend_field 的列部分`；`LocalString` ← `GetFieldLocalStringText` 或 `TableColumnHelper.GetColumnTextByFullName` |
| `TransformMultilangNode` | :180 | `sqlTabName` ← `LangTable`；`colName` ← `LangRTN`；再走 `TransformFieldType` |

---

## 6. 增 / 删 / 改 语义

### 6.1 增：`SpecificationInfo.Add(XmlElement, bool)`（SpecificationInfo.cs:1648）

```csharp
FormSpecModel m = new FormSpecModel(Key, xmlElement);
if (FormSpeDictionary.ContainsKey(m.Name)) throw new Exception("Name is Exist");
FindAllFieldsSpecByName(m);              // → FindFieldSpecById / FindHelpCodeSpecById / FindProgRelSpecById …
FormSpeDictionary.Add(m.Name, m);
if (m.SpecNodeType == SpecNodeType.FIELD) {
    SpecNodeTransform.TransformFormToSpecField(m.SpecField, xmlElement);
    if (isModifiedStatus && Table != "" && Column != "")
        m.SpecField.Status |= SpecStatus.MODIFY;      // ← "u" 的唯一来源
    DatabaseSource?.RefreshUsed(FormSpeDictionary);
}
_screenRecordManager.AddRecordField(xmlElement);
```

**`status="u"` 的充分必要条件**：`isModifiedStatus == true` **且** `table` 与 `column` 均非空。
这精确解释了两组样本的差异：局部字段（`table`/`column` 空）拿不到 `u`。

### 6.2 哪些组件会产生规格字段（`FindFieldSpecById`，:1268）

早返回（**不生成**）：

```
Folder Grid Group HBox ScrollGrid Table VBox Label Canvas Image HLine
```

其余类型（Form / Page / RadioGroup / Tree / Edit / ButtonEdit / ComboBox / CheckBox /
DateEdit / DateTimeEdit / TextEdit / SpinEdit / TimeEdit / Button / ProgressBar /
FFLabel / FFImage / Phantom / Item / RadioGroupItem / Slider / WebComponent …）会生成。

生成逻辑：

```
在 this.fields 里找同名且未标 DELETE 的 → 找到则从 fields 摘出并复用
否则 SpecFieldNode.Create(this, name)
       若 CheckUndoRedoManager(Key) 为真:
           SetDefaultProperties(Key, Type.ToString(), node)
           TransformRequired(node, GeneroComponent)
```

`SetDefaultProperties`（SpecFieldNode.cs:584）只做一件事：

```
widgetType ∈ {ComboBox, RadioGroup, CheckBox} → spec.req = "Y"
```

### 6.3 删：软删除

**删除不移除任何节点，只按位或上 `DELETE`**（`DeleteFieldLocalStringText`，SpecificationInfo.cs:2354）：

```csharp
var live = fieldStrings.FirstOrDefault(x => x.Name == name && (x.Status & DELETE) == NULL);
if (live != null) live.Status |= SpecStatus.DELETE;
```

所以 `.tsd` 会**持续累积墓碑**，与设计器的"保存只增不减"一致。
`status="d"` 的节点在读取时被各处的 `(x.Status & DELETE) != DELETE` 过滤掉。

### 6.4 改：条目在 `SpecificationInfo.cs` 1100-1150 一带

`Status = SpecStatus.MODIFY`（直接赋值，非按位或）用于：
`SpecProgRelNode` :1110、`SpecReferenceNode` :1118、`SpecMultiLangNode` :1126、
`SpecActionNode` :1133、`SpecFieldNode` :1138、`SpecHelpCodeNode` :1145。

### 6.5 id 重编号

`fieldId` / `fieldIdRef` **不是稳定标识**。实测每保存一次全部重编号（aapp320：56/56；apmt500：353 处）。
分配算法尚未定位，见 §8。**写入方与地址解析都必须只用 `name`。**

`.bdx` 的 `TBinding@ID` **稳定**（实测 0/50 变化），与 `fieldIdRef` 行为相反。

---

## 7. Headless（CLI）与 GUI 的行为差异

这些是调用设计器自身代码时必须补偿的点，**不补偿就会产出与设计器不同的结果**：

| 差异点 | 出处 | GUI 行为 | Headless 风险 |
|---|---|---|---|
| 默认值套用 | `FindFieldSpecById`:1319 | 有条件 | `CheckUndoRedoManager(Key)` 为假时**跳过** `SetDefaultProperties` 与 `TransformRequired`，产出的字段缺 `req` |
| 重叠校验 | `XmlElement.CheckOverlapping`:2516 | 读 `PreferenceManager.Current.Settings.ValidateForm` | 未注入 `PreferenceModel` 时 NRE |
| 错误提示 | 各处 | `DesignerMessageBox.Show` 弹模态框 | 自动化场景会**挂死**，必须预检或超时保护 |
| 版本门禁 | `SettingManager.LoadCommonData`:550 | 比对 `mta/ver` | CLI 程序集版本必须等于设计器版本 |
| 注册 | `SettingManager.OpenSpecFiles`:469 | 注册 `tzpMap` | 未注册时 `XmlElement.IsCantDel` NRE |

**已验证可行的 headless 加载链**（探针 `gen.exe`）：

```
new Application() → 合并 langs/zh-cn.xaml → SettingManager.Get()
→ 设 CurrentSetting.Connection.Workspace → LoadCommonData()
→ new TzpManager(tzs) → 注册进 tzpMap
→ 注入 PreferenceModel{ValidateForm=false}
→ SpecificationInfo.Create(tzp)
结果：Env=c，FormSpeDictionary = 114 项
```

---

## 8. 保存管线与无界面驱动（已实测）

### 8.1 完整保存路径

`SpecificationInfo.SaveSpecificationInfo(key)`（SpecificationInfo.cs:243）：

```
SaveToTSD()     模型 → TSDElement
SaveToForm()    模型 → FormElement
SaveBinding()   模型 → .bdx
InitTSDValidateWorker() / InitFormValidateWorker()
清理：移除空的 RecordField@colAliasName
```

三个 `SaveTo*` 都是可调用的，**这就是 `.tsd` / `.4fd` / `.bdx` 的完整写入路径**。

### 8.2 无界面驱动链（探针 `addprobe.exe`，已跑通到产出 `.tsd` 节点）

```
new Application() → 合并 langs/zh-cn.xaml
SettingManager.Get() → 设 Workspace → LoadCommonData()
new TzpManager(tzs) → 注册 tzpMap
SpecificationInfo.Create(tzp) → 回写 tzp.SpecificationInfo   ← 必须！
注入 PreferenceModel{ValidateForm=false}
SpecificationInfo.FormNode                                   ← ViewModel 对象树根
ComponentFactory.CreateEmptyElementWithName(type, name, key) ← 产 62 属性的完整元素
设 el.Parent + 插入 parent.Nodes（绕过 undo 包装）
设 sqlTabName/colName/fieldType
SpecificationInfo.Add(el, isModifiedStatus: true)
SpecificationInfo.SaveToTSD()                                ← 产出 <field>
```

实测产出（`aapp320`，Edit 绑定 `pmdl_t.pmdlud001`）：

```xml
<field src="c" ver="1" column="pmdlud001" name="probe_edit_1" table="pmdl_t"
       attribute="" type="" req="" i_zoom="" c_zoom="" chk_ref="" items=""
       default="" max="" min="" can_edit="N" can_query="Y" widget="Edit"
       cite_std="N" status="u" />
```

同时自动生成 `<RecordField fieldIdRef="113">`（id 由设计器分配）。

### 8.3 与设计器真实输出的残差

对比 `apmt500_wf` 里设计器亲手产出的那条：

| 属性 | 探针产出 | 设计器产出 | 差异来源 |
|---|---|---|---|
| `attribute` | `""` | `C003` | 列元数据富化步骤 |
| `type` | `""` | `varchar2(40)` | 同上（`tables.xml` / `TableColumnHelper`） |
| `req` | `""` | `N` | 同上 |
| `can_edit` | `N` | `Y` | 取决于 `GetCodeTemplate()`（本例表单为 `p`/`r` 分支） |
| `Name`（大写） | 无 | 有 | 富化步骤追加 |
| `ver` | `1` | `4` | 表单自身版本 |

**结论：残差全部集中在"列元数据富化"这一步**，即从数据字典取列的 `type`/`attribute`/`req` 并回填。
这是 §9 待定位项的主线索——下一步应定位 `TableColumnHelper.GetDefaultAttributeValue` 附近的调用链。

### 8.4 端到端验收（**已通过**）

用同一份输入（`apmt500_wf(c) - 测试备份 - 修改前.tzs`）、同一操作（新增一个绑定
`pmdl_t.pmdlud001` 的 `ButtonEdit`），与设计器亲手产出的 `apmt500_wf(c).tzs` 对比：

```
AI/CLI 产出 : src=c | ver=4 | column=pmdlud001 | name=pmdl_t.pmdlud001 | table=pmdl_t |
              attribute=C003 | type=varchar2(40) | req=N | i_zoom= | c_zoom= | chk_ref= |
              items= | default= | max= | min= | can_edit=Y | can_query=Y | widget=ButtonEdit |
              cite_std=N | status=u | Name=pmdl_t.pmdlud001
设计器产出  : 完全相同（21 个属性，值、顺序均一致）
```

**验收通过的完整配方**（`accept.exe` 实测）：

```
1. headless 引导（§8.2）
2. EventAggregatorManager.CreateInstance(key)          ← 缺口 #9
3. 注册 UndoRedoManager(100) + Init() 进 undoRedoManagerMap
4. new TzpManager(tzs) → 注册 tzpMap → SpecificationInfo.Create → 回写 tzp.SpecificationInfo
5. CreateEmptyElementWithName(type, name, key) → 设 Parent + 插 Nodes
6. 设元素 sqlTabName / colName / fieldType
7. SpecificationInfo.Add(el, isModifiedStatus: true)
8. 按 §8.6 表直接富化：
      spec.widget ← 元素类型名
      spec.attribute / type / req ← GetColumnInfo(table, column)
      spec.i_zoom / c_zoom / default / max / min / chk_ref / items ← GetColumnAttrInfo
      spec.Name（大写） ← "{table}.{column}"
      元素.comment ← "cmt_" + column
      SetFieldLocalStringText("lbl_"|"cmt_" + column, GetColumnTextByFullName(...))
9. SaveToTSD() / SaveToForm() / SaveBinding()
```

**重要**：第 8 步是**直接写**，不驱动 `SpecAttributeUndoRedoCommand`。
undo 命令是 UI 管道，其 `Execute()` 在合成 manager 上不落值（§8.8），
而富化的语义已完整掌握，直接实现更稳。

### 8.5 无界面缺口清单（补全版）

| # | 缺口 | 现象 | 补偿方式 |
|---|---|---|---|
| 1 | `TzpManager.SpecificationInfo` 未赋值 | `AbstractSpecNode.Status` setter 回查时 NRE | `SpecificationInfo.Create` 后回写该属性 |
| 2 | `PreferenceManager.Current.Settings` 为 null | `XmlElement.CheckOverlapping` NRE | 注入 `PreferenceModel{ValidateForm=false}` |
| 3 | `XmlElement.IsCantDel` | `GetTzpManger(Key)` 返回 null → NRE | 把 TzpManager 注册进 `SettingManager.tzpMap` |
| 4 | 程序集版本 | `LoadCommonData` 抛 `VersionIncompatibleException` | CLI 程序集版本必须等于设计器版本 |
| 5 | `AddNode`/`AddNodeAt` | 抛 `"No UndoRedoManager"` | 绕过 undo 包装，直接设 `Parent` + 插 `Nodes` |
| 6 | `ComponentFactory.CreateEmptyComponent` | 数据绑定控件全部 NRE（排版重算） | 改用 `CreateEmptyElementWithName`（对所有类型成功） |
| 7 | `DesignerMessageBox.Show` | 错误路径弹模态框 | 自动化场景必须预检 + 超时保护 |
| 8 | `ModFdInfo.IsIncludeAttribute` | 类型不在 mod-fd.spec 时原生 NRE | 调用前先校验类型在目录内 |
| 9 | `EventAggregatorManager.Get(key)` | 未注册时抛 `"EventAggregator not exist"`，由 `SetFieldLocalStringText` 触发 | 调 `EventAggregatorManager.CreateInstance(key)` |
| 10 | 列元数据路径 | 光有 `mta/` 不够 | 必须能访问 `<workspace>/<模块>/tbl/*.tbl` |

### 8.6 列元数据富化（残差的来源，已完整定位）

**触发点**：`SpecFieldNode.AttributeChanged("column", newValue)`（SpecFieldNode.cs:342）。
只有当 `column` 走**属性 setter** 时触发；`SetAttribute(key, value)`（AbstractSpecNode.cs:355）**直接写 XElement，完全绕过它**。

> 注意：`SpecNodeTransform.TransformFormToSpecField` 用的正是 `SetAttribute`，
> 所以设计器自己的 `Add` 流程**不富化**。真实设计器之所以有 `type`/`attribute`，
> 是因为用户在属性编辑器里改列时走的是属性 setter。

**元数据来源**（已实测在无界面下可用）：

```csharp
XElement columnInfo     = TableColumnHelper.GetColumnInfo(table, column);
XElement columnAttrInfo = TableColumnHelper.GetColumnAttrInfo(table, column);
```

`TableColumnHelper.FindTableColumns(table)`（TableColumnHelper.cs:37）的解析链：

```
tables.xml 里 <table name="pmdl_t" module="APM">   ← 只有表级信息，无列定义
  → Path.Combine(BasePath, module, "tbl", table + ".tbl")
  → <workspace>/APM/tbl/pmdl_t.tbl                 ← 列定义在这里
```

实测：`GetColumnInfo("pmdl_t","pmdlud001")` →
`<column name="pmdlud001" text="自定义字段(文本)001" attribute="C003" type="varchar2(40)" pk="N" req="N" xr_type="string"/>`

**富化的完整动作清单**（SpecFieldNode.cs:362-430）：

| 目标 | 值 | 来源 |
|---|---|---|
| spec.`column` | newValue | |
| spec.`attribute` | `columnInfo@attribute` | 列元数据 |
| spec.`type` | `columnInfo@type` | 列元数据 |
| spec.`req` | `columnInfo@req` | 列元数据 |
| spec.`i_zoom` `c_zoom` `default` `max` `min` `chk_ref` `items` | `columnAttrInfo@*` | 列属性 |
| spec.**`Name`**（大写） | `{table}.{column}`，重名时 `GetNewName` | **这就是那个"凭空出现"的属性** |
| 表单元素 `comment` | `"cmt_" + column` | |
| 绑定元素 `text` `title` | `"lbl_" + column` | |
| 绑定元素 `comment` | `"cmt_" + column` | |
| 本地化串 `lbl_<col>` / `cmt_<col>` | `GetColumnTextByFullName("{table}.{column}")` | |
| 绑定元素重命名 | `GetNewName("lbl_" + column, key)` | |

**前置条件**：SSpecField 所在的 `FormSpecModel` 必须已注册进 `FormSpeDictionary`，
否则 `FindNodeByName(this.Name)` 返回 null，`GeneroComponent.SetAttribute` NRE。

**对 CLI 的结论**：富化可以**直接实现**，不必驱动 undo 框架——用
`TableColumnHelper.GetColumnInfo` / `GetColumnAttrInfo`（已验证无界面可用）取值，
按上表写入即可。undo 命令是 UI 管道，不是语义。

### 8.7 `fieldId` / `fieldIdRef` 的分配算法（已定位）

分配器 `ScreenRecordManager.GetNewFieldIdRef(List<XElement> records)`（ScreenRecordManager.cs:167）：

```csharp
// 收集所有 Record 下 RecordField 的 fieldIdRef，升序排序
// 返回第一个空洞 (num+1)；无空洞则返回 max+1
// 注意：是全局单一编号空间，不是每个 Record 独立
```

**两个调用点**，都在每次保存时执行：

| 位置 | 作用 |
|---|---|
| `ScreenRecordManager.cs:77` | 增量：新建元素时分配 `fieldId` |
| `SpecificationInfo.GetScreenRecord`（:2232） | **全量重编号**：遍历 ViewModel 树，对每个有 `fieldId` 的元素重新分配 |

关联机制（`AddRecordFieldTo`，ScreenRecordManager.cs:59）：

```
componentModel.fieldId = GetNewFieldIdRef(已建记录)
RecordField.fieldIdRef = componentModel.fieldId        ← 同一个值
RecordField.{name, fieldType, sqlTabName, colName} ← 从元素复制
```

**`SaveToForm()`（SpecificationInfo.cs:2201）是 `.4fd` 的完整写入路径**：

```
1. 深拷贝 FormElement
2. 移除 <Form> 布局
3. 移除 <DiagramLayout>                    ← 每次保存都会删掉它
4. RebuildScreenRecord()：删光所有 <Record>，按 ViewModel 树重建
5. 从 FormNode.ToXML() 重新写回 <Form>
```

**Record 的归属规则**（`AddRecordField`，ScreenRecordManager.cs:29）：

```
父节点是 Table/Tree/ScrollGrid → 进该表名的 Record
自己就是 Table/Tree/ScrollGrid → 自建一个同名 Record
其他                          → 进 "Undefined"
```

**门禁**：`ComponentFactory.IsIncludeProperties(NodeName, "colName")` —— 只有拥有 `colName`
属性的组件才会产生 `RecordField`。

**对 CLI 的结论**：`fieldId`/`fieldIdRef` 是**导出量**，不是持久标识，跨保存不可依赖。
`SaveToForm` 会重建整个 Record 段并重编号，且每次都删 `DiagramLayout`。
CLI 若要最小改动，应**自己实现上述算法**（约 30 行）作为对原 XML 文本的补丁，
而不是调用 `SaveToForm()`（后者内容正确但会全量重写格式）。

### 8.8 尚未定位的项

| 项 | 状态 |
|---|---|
| `SpecAttributeUndoRedoCommand.Execute()` 在合成 UndoRedoManager 上为何不落值 | 未定位。**不影响交付**——见 §8.6 结论，富化可直接实现 |
| `<act>` `<sact>` `<strings>` 的生成规则 | 未读 |
| `IsTopstdMode` 的判定条件（决定 `src="s"` 还是 `src=Env`） | 未定位 |
| `ModFdInfo.IsIncludeAttribute` 对目录外类型原生 NRE | 需前置校验 |
| 撤销/回滚路径 | 未验证（目前只测了正向） |

---

## 9. 代码推导 vs 实际文件的已知偏差

**`tsd.xsd` 已与实现脱节，不能用于严格校验**：

| 偏差 | 证据 |
|---|---|
| XSD 声明 `cite_ver` 为 `required` | 实现与语料中 **0 次**出现 |
| XSD 写 `depend_filed` | 实现写 `depend_field`（语料 67 次，前者 0 次） |
| XSD 未声明 `Name` | 真实文件里有 `Name="…"`，追加在 `status` 之后 |
| 设计器产出的文件通不过自己的 XSD | 由上一行推出 |

**结论**：`tsd.xsd` 只作结构参考，校验必须另建（以各 `Create` 方法 + 语料统计为准）。

---

## 10. 对 CLI 架构的推论

1. **`.tsd` 侧不需要重写。** `Add(XmlElement, bool)` 是 public，其内部链条
   （`GetChildNode → FindFieldSpecById → SpecFieldNode.Create`）覆盖了字段/帮助码/程序关联/本地化串的生成。
   CLI 只负责 `.4fd` 的最小改动 + 调用它 + 序列化 `TSDElement`。
2. **`.4fd` 侧必须自己实现。** 设计器的 `Packing` 会整体重压、刷新时间戳，破坏 diff。
3. **地址只用 `name`**，绝不依赖 `fieldId`/`fieldIdRef`。
4. **`status` 是变更日志，不是属性。** 写入时不能"清理"墓碑，那是设计器的语义。
5. **Headless 差异必须显式补偿**（§7），否则产出物与设计器不一致。

---

## 11. `.4fd` 最小写入器：实现状态

### 11.1 已实现并验证

| 组件 | 文件 | 验证结果 |
|---|---|---|
| `ElementIndex` | `src/ElementIndex.cs` | 66 个文件：元素计数与 XML 解析器**完全一致**；跨度无一畸形；名字路径**零重复** |
| `RecordRebuilder` | `src/RecordRebuilder.cs` | 66 个文件：**48 个逐字节复现**设计器的 Record 段；**当前格式文件零差异** |
| `FormWriter` | `src/FormWriter.cs` | 空操作 **66/66 逐字节相同**；`set_attr` 差异恰好 4 字节；`add_node` 元素与 RecordField 均正确生成 |
| `TestRebuild` / `TestWriter` | `test/` | 上述回归测试 |

**Record 段复现的判定标准**：文件最后一次保存是被当前版本设计器写的（`<Record>` 属性为规范顺序）。
18 个差异文件全部是旧格式（属性字母序 + `uid`），当前版本设计器保存时也会同样规范化它们。

测试输出：

```
ElementIndex sanity over 66 files
  element-count mismatches vs XML parser : 0
  malformed spans                        : 0
  duplicate name-paths                   : 0

byte-identical Record section : 48
differing                     : 18
pre-current-format files (alphabetical <Record> attrs): 18  (of which identical: 0)
current-format files that STILL differ: 0
```

### 11.2 实现要点（踩过的坑）

| 坑 | 现象 | 处理 |
|---|---|---|
| `colName` 用排除法 | id 序列被多分配（差 2 / 差 14） | 改用**肯定列表**（目录里 18 个含 `colName` 的类型）。排除法会放进 `Button`/`Text`/`Spacer` |
| `SetAttributeValue(k, null)` | 新建 `RecordField` 时 `name` 被丢到属性末尾 | 用 `""` 占位，保证规范顺序 |
| Record 段跨度 | 按深度计数会在第一个 `</Record>` 处提前结束 | 改为锚定 `<Form>` **向前**找最后一个 `</Record>` |
| `Func` 默认值被 null 覆盖 | `FormWriter.HasColName = null` 冲掉了 `RecordRebuilder` 的默认实现 → NRE | 仅在非 null 时赋值 |

### 11.3 设计要点

**新元素的属性模板从文件内同类型元素克隆**，而不是硬编码或调用 DLL：

```csharp
w.CloneTemplate("Edit", overrides)   // 找文件里第一个同 tag 元素，复制其属性集与顺序
```

理由：文件是由**某个特定版本**的设计器写的，其属性集与顺序正是该版本能往返的形态。
从文件克隆自动匹配版本，比调用当前 DLL 更稳。DLL 路径作为可选覆盖保留。

**布局编辑全部是文本拼接**，只有 Record 段是重新生成的：

| 操作 | 手法 |
|---|---|
| `SetAttribute` | 只替换属性值那一段字节（实测差异 4 字节） |
| `AddNode` | 按 `ElementIndex` 偏移插入 `\r\n{indent}{element}`；父元素自闭合时先展开 |
| `RemoveNode` | 整行删除 |
| Record 段 | 重建（设计器每次保存也重建，无法避免） |
| `DiagramLayout` | 跟随设计器行为一并删除 |

### 11.4 端到端验收（**已通过**）

`test/E2E.cs`：`FormWriter` 改 `.4fd` → 注入 `TzpManager.GeneroFormString` → 设计器加载并自行注册 →
按 §8.6 富化 → 提升 Status → `SaveToTSD` / `SaveBinding` → 与设计器手工产出的结果对比。

```
[1] .4fd in : 543610 chars  →  out: 545217 chars  (delta 1607)
[3] FormSpeDictionary["pmdl_t.pmdlud001"] = present   ← 加载自动注册，无需手工 Add
[6] .tsd 节点比对（全部节点类型）
    --- name=pmdl_t.pmdlud001   mine=2 designer=2
        OK   <field>
        OK   <hfield>
    --- name=lbl_pmdlud001   mine=1 designer=1
        OK   <sfield>
    --- name=cmt_pmdlud001   mine=1 designer=1
        OK   <sfield>
    => identical nodes: 4   differing: 0
```

**四类节点全部逐属性一致。** `pfield` 在双方都正确缺席——父容器是 `Grid` 而非 `Table`，
符合 `FindAllFieldsSpecByName` 的条件（`Parent.Type ∈ {Table, Tree, ScrollGrid}` 才生成）。

**关键认识**：`SpecificationInfo.Create` 在加载时会遍历表单树、对每个元素调一次 `Add`，
所以 CLI **不需要手工调 `Add`**——写好 `.4fd` 交给设计器加载即可。
`.tsd` 的各类节点（field / hfield / pfield / sfield）随之自动生成。

### 11.5 本轮新增的两个 headless 缺口

| # | 缺口 | 现象 | 补偿 |
|---|---|---|---|
| 11 | `SetInitGridX` 的注册时序 | `GetChildNode` → `SetInitGridX()` 在 UndoRedoManager **已注册**时抛 `"illeagal call SetInitGridX"` | **加载前不注册**，`SpecificationInfo.Create` 之后再注册。加载要"无"，富化要"有" |
| 12 | `ToXml()` 过滤 `CREATE` | `AbstractSpecNode.ToXml()`：`if (Status == CREATE) return null` → 加载期诞生的节点是瞬态，不参与序列化 | CLI 必须把新增节点的**内存 Status** 提升为 `MODIFY`。只改 XML 的 `status` 属性无效 |

缺口 12 是整条链最后卡住的地方，也是最容易误判的：节点的 XML 看起来完全正确，但序列化时被静默丢弃。
`addprobe` 之所以没遇到，是因为它调了 `Add(el, true)`，那里会 `Status |= MODIFY`。

### 11.7 重打包（**已通过**）

`src/TzsRepacker.cs` + `test/TestRepack.cs` + `test/VerifyRepack.cs`。

**目标**：只重写变更条目，其余条目的压缩字节与时间戳原样保留——这正是设计器的
`PackageManager.Packing` 做不到的（它把所有条目 `DateTime` 刷成 `DateTime.Now`、
统一 `SetLevel(3)` 重压，一次编辑会让整包在 VCS 里全红）。

**验收结果**：

```
=== 空操作重打包（66 个文件）===
  byte-identical : 66
  changed        : 0

=== 单条目替换（apmt500_wf(c).tzs，改写 .tsd）===
  entry order preserved: True
  untouched entries byte-identical: 4   changed: 0
  untouched entry timestamps preserved: 4   changed: 0
  replaced entry round-trips: True

=== 设计器自己的读取路径（ZipInputStream，即 PackageManager.Unpacking）===
  read apmt500_wf.4fd           546900 chars
  read apmt500_wf.4fd.merge     66 chars
  read apmt500_wf.tsd           166927 chars
  read ver                      4 chars
  read apmt500_wf.bdx           9190 chars
  entries identical: 4   changed: 1   (expect exactly 1)

=== SpecDesignerCommon.PackageManager.SeekReleaseVersion ===
  ver entry read back as: 1.0
```

**实现要点（踩过的坑）**：

| 坑 | 现象 | 处理 |
|---|---|---|
| **归档是 ZIP64** | SharpZipLib 对小文件也写 ZIP64：CD 的 32 位尺寸字段是 `0xFFFFFFFF` 占位，真值在 `0x0001` 扩展字段 | 解析 ZIP64 extra 取值；**未变更条目保留占位符**，只重写会移动的 `localOffset` |
| 逐字段重建 CD 头 | `externalAttrs` 被静默清零 | 改为**整块复制原 46 字节头**，只补丁真正变化的字段 |
| 已改条目残留 ZIP64 extra | 新旧尺寸互相矛盾 | 重写条目时剥掉 `0x0001` 记录，写普通 32 位尺寸 |
| EOCD | 逐字段重建会丢注释/磁盘号 | 整块复制原 EOCD，只补丁条目数与 CD 位置 |
| CRC32 | .NET 4.0 无内置实现 | 自带表驱动实现 |

**注意**：产物比原文小（50071 → 47809），因为 .NET 的 `DeflateStream` 在**被改条目**上
压得比 SharpZipLib level 3 好。只有该条目的字节变了，其余原样。

### 11.8 `add_field` 走设计器的布局引擎（**已通过**）

原实现自己拼控件、自己算位置、自己造标签——等于重写设计器已有的东西。
正确做法是调用 `UICreator.CreateNoneContainerWidget`（`SpecDesigner.FormEditor/DBStructure/UICreator.cs`）。

**控件箱的权威清单**（`WidgetBox.xaml`，共 31 项）：

| 族 | 内容 |
|---|---|
| 语义节点（3） | 字段参考 `REFERENCE`、多语言 `MULTILANG`、查询按钮 `PROGREL` |
| 容器（6） | `Folder` `Grid` `Group` `ScrollGrid` `Table` `Tree` |
| 从数据字典加（1） | `AddWidgetByDataControl` → `DBStructureCreator` → **`UICreator.Create`** |
| 基础控件（20） | `Button` `Canvas` `HLine` `Image` `Label` `ButtonEdit` `CheckBox` `ComboBox` `DateEdit` `Edit` `FFImage` `FFLabel` `ProgressBar` `RadioGroup` `Slider` `SpinEdit` `TextEdit` `TimeEdit` `DateTimeEdit` `WebComponent` |
| 工具（1） | `SetTabIndex`（非创建） |

**可容纳拖放的容器**（`WidgetBox.xaml.cs:138` 硬编码）：`Grid` `HBox` `VBox` `Group` `Folder` `Page`

**`UICreator` 按容器类型分派**：`CreateNoneContainerWidget` / `CreateGridContainer` /
`CreateGroupContainer` / `CreateScrollGridContainer` / `CreateTableContainer` / `CreateTreeContainer`

**调用契约**（实测）：

```
输入：
  ContainerViewModel { Container = new ContainerType { Type = "Grid" },
                       MaximumWidthStrValue = "20", NumberOfFieldsStrValue = "1" }
  IEnumerable<PrepareAddColumn> { Table, Column, Description, Label,
                                  Widget ← <table>.tbl 的 <col_attr>@widget,
                                  Width  ← <col_attr>@widget_width }
  PackageKey

输出：List<XmlElement>  —— 成对的 Label + 控件，已完成：
  命名（<表>.<列> / lbl_<列>）、fieldType、标签伴随节点、
  两列布局（标签 posX=1、控件 posX=12）、以及 .bdx 绑定（AddSpecBinding）

前置：
  UndoRedoManager 必须已注册（布局 setter 走撤销命令）
  但 SpecificationInfo.Create 期间必须先摘掉（SetInitGridX/Y 会抛）
```

**`add_field` 的最终形态**（`test/AddField.cs`）：

```
1. 加载设计器上下文；注册 UndoRedoManager
2. UICreator.CreateNoneContainerWidget(...) → 成对的 Label + 控件
3. FormWriter 把它们按 NextFreeRow 拼进 .4fd 文本（最小改动）
4. 重新加载编辑后的 .4fd（加载期间临时摘掉 UndoRedoManager）
5. 富化 + 提升 Status + AddSpecBinding（.bdx 里没有 .bdx 时不会自动重建）
6. SaveToTSD + SaveBinding
7. TzsRepacker 只重写变更条目
```

**实测产出**（`aapp320`，加 `apca_t.apcaent`）：

```
apca_t.apcaent  → 表名「企业编号」 widget=Edit width=10     ← widget 由列元数据决定
[2] UICreator.CreateNoneContainerWidget produced 2 element(s)
[3] spliced at row 9;  .4fd 92258 → 93813 chars (+1555)
    <Edit name="apca_t.apcaent" posX=12 posY=9 w=10 h=1>  children=0
    <Label name="lbl_apcaent"   posX=1  posY=9 w=10 h=1>  children=0
[4] reloaded; FormSpeDictionary=116 entries
    1 node(s) enriched, 2 promoted out of CREATE
    bound lbl_apcaent ↔ apca_t.apcaent
[5] .tsd 29252 chars;  .bdx 617 chars   （原 449，新增 ID=2 的绑定）
[6] repacked: 13138 → 11008 bytes
验证：<Edit> fieldId=67  /  RecordField fieldIdRef=67  /  .bdx ID=2 lbl_apcaent ↔ apca_t.apcaent
```

**取元素用 `ToXML()` 而不是 `Source`**（本轮更正）。`XmlElement.Source` 是私有属性，
构造时 `RemoveNodes()` 过，**永远不带子节点**——容器模式必须用 `ToXML()`。
而且 `ToXML()` 才是权威序列化：`SpecificationInfo.SaveToForm()` 末行就是
`xelement.Add(this.FormNode.ToXML())`。它按设计器自己的规则过滤属性
（丢弃值为空的 `style`/`format`/`picture`/`width`/`height`/`unhidable`/`unmovable`/
`unsizable`/`unsortable`，丢弃 `fontPitch`/`sqlType`/`lookup`/`sample`/`ref_usage`/
`guid`/`displayTabName`/`displayColName`/`validateTabName`/`validateColName`，
丢弃 `NOSET` 哨兵值），结果与文件里既有元素的属性集一致
（实测：既有 `<Edit>` 同样没有 `justify`/`picture`/`fontPitch`/`style`）。

**相比手写版的改进**：控件类型由设计器从列元数据选（手写版猜错了 `ButtonEdit`，
实际应是 `Edit`）；标签伴随节点、`.bdx` 绑定全部由设计器产生。

### 11.9 缺口 #6 的更正

早期记录的「`ComponentFactory.CreateEmptyComponent` 对数据绑定控件全部 NRE」**是误报**。
实测：在完整初始化下（`Application` + 语言资源 + `LoadCommonData` + `tzpMap` +
`EventAggregatorManager` + `SpecificationInfo`），它对 `Edit`/`ButtonEdit`/`ComboBox`/
`Label`/`Table` 全部成功，有无 UndoRedoManager 都一样。当时的 NRE 来自探针初始化不完整。

### 11.10 六种落点：全部接通（**已通过**）

`UICreator.Create`（UICreator.cs:16）按 `ContainerType.Type` 分派。**六条支路全是
`private static`，签名都是 `(ContainerViewModel, IEnumerable<PrepareAddColumn>, PackageKey)`**，
所以反射调用只需要换方法名：

| `Type` | 方法 | 产出结构 |
|---|---|---|
| `None` | `CreateNoneContainerWidget` | 扁平同级：控件 + 标签，两列（标签 posX=1，控件 posX=12） |
| `Grid` | `CreateGridContainer` | **新建一个 `Grid`**，字段作为它的子节点 |
| `Group` | `CreateGroupContainer` | **新建 `Group` → 内含新建 `Grid` → 内含字段**（两层） |
| `ScrollGrid` | `CreateScrollGridContainer` | **新建 `ScrollGrid`**，字段带 `repeat=true`/`rowCount`/`columnCount`/`stepX`/`stepY` |
| `Table` | `CreateTableContainer` | **新建 `Table`**，字段的 `LocalString` 落到 `title` |
| `Tree` | `CreateTreeContainer` | **新建 `Tree`**（自带 `name`/`id`/`parentid`/`isnode`/`expanded` 五个构件） |

**与前五者相对 `None` 的唯一结构差异**：除 `None` 外，返回值是**一个新的容器元素**，
字段在它内部。所以拼接目标从"同级兄弟"变成"新容器本身"，且**只有它自己的
`posX`/`posY` 需要摆放**——内部子节点的坐标是相对新容器的，原样即可。

**定位**：`None` 模式 UICreator 已按"从第 1 行开始"排好相对坐标，所以取
`NextFreeRow` 后整体**平移**；容器模式则直接置于 `NextFreeRow`。

**`.bdx` 绑定不能靠名字猜**：`None` 模式 UICreator 内部调了
`AddSpecBinding(label, widget)`（:169），而 `ScrollGrid` 的标签是裸 `Label`，
**不建绑定**；`Table`/`Tree` 更是把文本写进字段自身的 `title`。所以绑定对必须从
`XmlElement.BindElement` 反读。且方向要归一成 **label 在前**——`TBinding.Equals`
不区分顺序，先加入的那条决定 `ObjectName1`，设计器的约定是
`AddSpecBinding(label, field)`。

### 11.11 规格节点的提升规则（本轮的关键更正）

`AbstractSpecNode.ToXml()` 对 `CREATE` 状态返回 null，所以新增元素默认**不会**进 `.tsd`。
但"凡加载期诞生的节点都提升"是**错的**：`Tree` 自带的五个构件
（`id`/`parentid`/`isnode`/`expanded`，以及那个名为 `name` 的 `Edit`）在加载期同样
是 `CREATE`，而语料里 37 个含 `Tree` 构件的包里，**设计器自建的**
（`src="c"` 且 `status` 非空）`.tsd` 节点数为 **0**。

（另有 7 个匹配的节点，但全是**继承来的**：`src="s"`、`status=""`，6 个 `hfield` + 1 个
`rfield`，来自 `_wf(c)` 包的 `.tsd` 里携带的标准版节点。它们不是设计器为本程序元素
新建的，`FindNodeByName` 会复用它们——所以也不该由我们制造。）

判定规则直接用设计器自己的谓词 —— `SpecNodeTransform.TransformFieldType`
（SpecNodeTransform.cs:90）：

```csharp
string col = formElement.GetAttribute("colName");
if (!string.IsNullOrEmpty(col))  fieldType = "TABLE_COLUMN" / "COLUMN_LIKE";
else                             fieldType = "NON_DATABASE";
```

**只有 `colName` 非空的元素才提升。** 实测修正前后的新增节点数：

| 模式 | 修正前 | 修正后 | 应该是 |
|---|---|---|---|
| `None` / `Grid` / `Group` | 2 | 2 | `field` + `hfield`（真列，有帮助码） |
| `ScrollGrid` / `Table` | 3 | 3 | `field` + `hfield` + `pfield` |
| `Tree` | **6** | **1** | 只有 `field:apcaent`；五个构件不该有节点 |

两个把这条规则写错的坑：
- `AttachDefaultAttributes` 给**每个**元素都写了 `colName=""`。所以判空必须用
  `IsNullOrEmpty`，只挡 `null` 会让空串通过。
- `XmlElement` 的属性**只能通过 `GetAttribute(key)` 取**，不是 .NET 属性；
  `Prop(el, "colName")` 永远是 null（`NodeName` 能取到是因为它确实是属性）。

### 11.12 不动点检验：新的验收方法（**已通过**）

"样本只做验收"需要一个**不需要人工开设计器**的验收手段。做法是问：
**把产出重新加载，让设计器自己 `SaveToForm`/`SaveToTSD`/`SaveBinding` 重生成，
写出来的是不是还是它？**（`test/RoundTrip.cs`）

对 6 个产出（单字段）与 3 个产出（三字段）实测：

```
.tsd      : IDENTICAL      ← 逐字节（规范化后）
.bdx      : IDENTICAL
.tsd 规格节点: 0 增 0 减
.4fd 路径      : 0 增 0 减
```

`.4fd` 的**字符数**会变（如 93813 → 91081），但那是设计器自己的重序列化
（空白 / 属性顺序规范化）——**未改动的原生文件同样如此**（`aapt110(c).tzs`：
333301 → 327136），所以字符数不是判据，**路径集合**才是。

基线校验：先对一份原生文件跑，得到 `0 增 0 减`，才说明这个判据可用。

另有一个更强的单点实验：把树探针做成"**我的 `.4fd` + 原始 `.tsd`**"，若模型重建出
的节点数与原始 `.tsd` 相同（实测 167 = 167，0 增 0 减），就证明该子树
**贡献零个规格节点**——这正是判定 `Tree` 五个构件不该提升的依据。

### 11.13 批量验证：全语料（**已通过**）

单表单验证不够——语料有 86 个包。两个驱动脚本：

| 脚本 | 问什么 | 结果 |
|---|---|---|
| `batch.sh` | 能否无界面加载？空操作重生成是否 0 增 0 减？ | **86/86 干净**（`.tsd` 节点集、`.4fd` 路径集各 0 增 0 减） |
| `batch-write.sh` | 每个包加一个真字段后，产出还是不是不动点？ | **77/80 干净**，1 失败（有明确原因），2 跳过（取样器取不到列） |

**语料分布**（决定了哪些分支被真正跑到）：

```
code_template : F=61  P=19  Q=4  W=2
容器元素      : Grid 924  Page 453  Group 315  Table 239  Folder 125  Tree 68  ScrollGrid 1
布局元素数    : 10（xiyuan/tst/cs_excel_in_xmdl_s01）~ 1297（hengshuo/prd/axmt500_wf）
```

- **`code_template` 这条轴无害**：`SpecFieldNode.Create`（SpecFieldNode.cs:524）的
  P/R 分支与其余分支**逐属性相同，只差一个默认值**（`can_edit` 是 `"N"` 还是 `"Y"`），
  且该值随后被列元数据富化覆盖。
- **`ScrollGrid` 全语料只有 1 处**。那个分支实现了但实质上无法验证——不要声称它已验证。
- **工作区是每个模块独立的**：`xiyuan/prd`、`xiyuan/tst`、`hengshuo/prd` 各有自己的
  `mta/`。`TzpManager` 拒绝打开配置工作区之外的包（`NotInCurrentWorkspaceException`）。
  判定规则：**最近一个含 `mta/` 的祖先目录**。第一次跑出的 13 个"失败"全是这个原因，
  两个程序都必须读 `TZSCLI_WS`（最初 `AddField` 漏了，12 个包因此误报）。

### 11.14 无界面下会弹模态框（新缺口）

`SpecificationInfo..ctor` 第 145 行**无条件**构造 `DatabaseSourceViewModel`，而它的
构造函数（DatabaseSourceViewModel.cs:41-82）会对 `AssociateTable.AliveTBLs` 里每张表调
`FindTableColumns`，查不到就 `catch` → **`DesignerMessageBox.Show` 模态框**。

无界面下没有消息泵，`Dispatcher.Invoke` 永久阻塞——**进程挂死，stdout 什么都不说**。
`AddField` 现在用看门狗把两次加载都包住：超时（默认 90s，`TZSCLI_RELOAD_TIMEOUT` 可调）
就打印诊断并 `Environment.Exit(3)`。看门狗必须在**独立线程**上，加载本身**必须留在
STA 主线程**（`Application.Current` 线程亲和，跨线程 `FindResource` 会抛）。

**触发条件**（`batch-write.sh` 里那唯一 1 个失败）：表单引用的表没有登记在
`<工作区>/mta/tables.xml` 里。`FindTableColumns(tableName)`（TableColumnHelper.cs:37）
**先查这份索引**，查不到直接返回 null，连 `.tbl` 都不看。实测
`xiyuan/tst/mta/tables.xml` 有 3345 张表，`hengshuo/prd` 有 3886 张——`tst` 那份缺
`sfamwf_t`/`sfanwf_t`/`sfbcwf_t`/`wfapwf_t`，而这 4 张的 `.tbl` 在 `tst` 里**是存在的**
（`asf/tbl/`、`bwf/tbl/`）。所以是**索引过期**，不是数据缺失，刷新基础资料即可。

**未查清**：同一份原始文件在无界面下**不会**走到这一步，改动过的会。两者 `.tsd` 相同
（第 4 步只替换 `.4fd`），所以差异出在 `ConvertTSDToModel()`（第 143 行，建
`AliveTBLs` 之前会重建 `<table>` 段）里对表单内容的敏感性。没有追到底，不要假装追到了。

### 11.15 没有控件映射的列

列的 `<col_attr>` 里 `<field widget="">` 为空时，`CreateFieldComponent` 会在
`ComponentFactory` → `ModFdInfo.IsIncludeAttribute` 深处 NRE。`type_t`（"数据型态参照档"，
列名就是 SQL 类型名）整张表都是这种列。**设计器自己也拖不了这种列**，所以工具应当拒绝并
说明原因，而不是让栈冒出来。`AddField` 已在建 `PrepareAddColumn` 时挡住；
`pick-column.py` 取样时跳过。

### 11.16 改 / 删（**已通过**）

`test/Edit.cs`。**与"增"最大的不同：只需要一次加载**——改和删都作用于模型已经认识的元素；
"增"要第二次加载只是因为设计器得为它从没见过的元素铸造规格节点。

**`Modify` —— 直接跑设计器自己的命令** `SpecAttributeUndoRedoCommand`：

```
Execute() 逐个属性：
  _specNode.SetAttribute(key, value)          ← 写规格
  按 key 分派 Transform*                       ← 把规格值推到配对表单元素上
     table/column          → TransformTableColumn
     req                   → TransformRequired     （→ formElement.IsRequired）
     can_edit              → TransformCanEdit      （→ noEntry）
     lang_table/lang_rtn   → MULTILANG 那套
     ref_table/ref_rtn     → REFERENCE 那套
  _specNode.OnPropertyChanged("")             ← Status |= MODIFY
```

它同时改两边，所以**不要自己维护属性映射表**。落盘仍是最小改动：在命令前后对表单元素
做属性快照，只把动了的写回 `.4fd`。

**`Delete` —— 布局真删、规格留墓碑**：

```
formElement.ClearBinding()        ← 先解绑：SaveBinding 只是序列化 SpecBinding，不会过滤
SpecificationInfo.Remove(name)    ← CloneModelToDeleted → Remove(node, oldName)
                                    造一个【只剩名字】的同类节点、Status=DELETE、
                                    塞进 this.fields / help_codes / …；
                                    再 RemoveRecordField；Table/Tree/ScrollGrid 还 Remove 表关联
FormWriter.RemoveNode(path)       ← 布局文本删除
```

墓碑长这样（其余属性全被剥空）：

```xml
<field src="c" ver="1" column="" name="apca_t.apcaent" table="" attribute=""
       type="" req="" ... widget="" cite_std="N" status="d" />
```

机制根源：`ToXml()` **只对 `CREATE` 返回 null**，`DELETE` 节点照常输出 → 墓碑。
另外 `SaveToTSD` 产出**两份**文档：`TSDElement`（全量，写回包里的就是它）和
`TSD2Element`（差量覆盖层，只收 `status` 为 `u`/`d` 的节点）。

**实测**：对 `apca_t.apcaent` 所在的字段对（Edit + Label）做删除，`.4fd` 93813 → **92258**，
正好等于加字段之前的长度；`.tsd` 从 117 节点变 119（两个墓碑，零丢失）；`.bdx` 里那一对
绑定消失；RoundTrip 0 增 0 减。

**全语料批量**（`batch-edit.sh`：逐包挑一个可编辑字段、把 `can_edit` 翻面、再验不动点）：
**78 clean / 0 failed / 2 skipped**（80 个设计器原生包）。两个 skip 是同一个原因——那两个
表单太小，表里带控件映射的列已被用尽，取样器挑不出目标。

顺带一条反证：`xiyuan/tst/bsft001_wf(c).tzs` 在**写侧**因 `mta/tables.xml` 缺表而弹框失败，
在**改侧**却通过——因为 `set` 不往表关联里加新表。

**两个坑**：

- `AddAttributeChanged(attr, old, new)` 开头 `if (old == new) return;`——**改成和原值相同会被
  静默跳过**，只剩 `OnPropertyChanged` 把状态置成 `u`。工具必须自己判并报"无需修改"。
- 反射选重载**不能只看方法名+参数个数**：`SpecificationInfo.Remove` 同时有
  `Remove(string)` 和 `Remove(SpecActionNode)`，按个数匹配会随机撞上后者并抛类型错误。
  必须先按参数类型可赋值性筛选。

**已知缺口（属于"增"，本轮暴露）**：`add_field` 用 `Enrich` 直接写规格属性，**从不跑
spec→form 的 `Transform*`**，所以新字段的布局可能与规格不一致——实测新字段
`req="Y"` 而布局 `style` 里没有 `required`。下次被设计器驱动编辑时会自愈，但文件本身
是不同步的。修法：加字段后按 `Edit.cs` 的办法跑一遍对应的 `Transform*`。

### 11.17 HBox / VBox：不是"新建容器"，是"把已有元素包起来"

`UICreator.Create` 的分派里**没有** HBox/VBox，控件箱里也**没有**——用户根本拖不出来。
设计器造它们的地方是 `ManagedForm.CanLayoutCommand` / `ExecuteHBoxLayout` / `ExecuteVBoxLayout`
（ManagedForm.xaml.cs:930-980），用的是：

```csharp
new AddToContainerUndoRedoCommand(selection, parent, ComponentType.HBox).Execute();
```

所以这是**布局重构**（把选中的元素整体挪进一个新盒子），不是"加字段"。`Edit.cs` 的
`wrap` 操作对应它，**只需一次加载**（命令末尾自己 `SpecificationInfo.Add(_newContainer)`）。
对称操作是 `BreakLayoutUndoRedoCommand`（拆盒子）。

**前置条件**（`CanLayoutCommand` 同样这三条，缺一不可）：

1. 有选中元素
2. 第一个选中元素的父节点存在且**不是 `Form`**
3. `AcceptMimes(child.NodeName, …)` 与 `AcceptMimes(parent.NodeName, HBox)` 都通过

**`AcceptMimes` 读的是 `core-br.spec` 的元数据**：
`NodeInfo[mimeType="modFD/<父>"]@acceptedMimes` 里是否含 `modFD/<子>`（ComponentFactory.cs:1232）。
实测语料的 `mta/mod-fd.spec`：

**注意：不能只读 `acceptedMimes` 那张表，`AcceptMimes` 还叠了两条额外规则**
（`parentNodeName == "Page"` 时要求 `IsContainer(child)`；`AcceptMimes(ComponentType,…)`
对 `Grid`/`Group` 父节点挡掉容器子节点）。第一版就是只读了 spec 文本，把 `Page` 那行写错了。
下表是按**代码规则 + 实测**给出的**有效**结果：

| 父容器 | 有效接受 |
|---|---|
| `Grid` / `Group` | **收控件**（`Edit`/`Label`/`Button`/`ComboBox`/… 一整批）；容器被额外规则挡掉 |
| `Folder` | **只有 `Page`** |
| `Page` | **只收容器**——spec 列表里虽写着控件，但 `IsContainer` 门槛把它们全挡了 |
| **`HBox` / `VBox`** | **只收容器**（Folder/Grid/Group/HBox/VBox/ScrollGrid/Table/Tree） |
| `Table` / `Tree` | 部分控件（`Phantom` 只在这两者下合法） |
| `ScrollGrid` | 控件 + 部分容器 |

实测：`add <Grid> Button` ✓、`add <Group> Button` ✓、`add <Folder> Button` ✗、
`add <Page> Button` ✗、`add <HBox> Button` ✗。

**结论：HBox/VBox 是"摆放容器块的横/纵布局"，不能把字段直接包进去。**
实测对两个 `Edit`/`Label` 调 `wrap HBox` 被 `AcceptMimes` 正确拒绝；对同一父节点下的两个
`Group` 调 `wrap HBox` 成功，新建 `hbox_11` 并重排父 HBox 的横坐标。

**`wrap` 落盘**：命令会改模型树（摘下元素、重定基准、插进新盒子、重排父容器坐标），所以
文本侧要「先按路径删掉被包裹的元素，再把新盒子的 `ToXML()` 插进父路径」。

### 11.18 新增 ACTION（按钮）

**`Button` 元素的 `SpecNodeType` 就是 `ACTION`**（`FormSpecModel.cs:467`）：

```csharp
case ComponentType.Button:
    specNodeType = (("button_qrystr" == ce.GetAttribute("style")) ? SpecNodeType.PROGREL : SpecNodeType.ACTION);
```

于是 `SpecificationInfo.Add(XmlElement)` → `FindAllFieldsSpecByName` → `case SpecNodeType.ACTION:
FindActSpecById`。而 **`FindActSpecById`（:1331）在找不到 `<act>` 时会自己造一个，且
`id` = 按钮元素的 `name`**：

```csharp
foreach (SpecActionNode a in this._acts)
    if (name == a.Name && (a.Status & DELETE) == NULL) { found = a; break; }
if (found == null) { found = SpecActionNode.Create(this, name); this._acts.Add(found); }
formSpecModel.SetSpecNode(found);
```

`<act>` 的规范属性（`SpecActionNode.Create`，:334）：

```
src ver id cite_std gen_code type status      ← status 初值 CREATE，必须提升，否则 ToXml() 丢弃
```

实测产出：`<act src="c" ver="1" id="my_action" cite_std="N" gen_code="Y" type="all" status="u" />`，
`.tsd` 只多这一个节点。

**只需一次加载**——这与"增字段"不同。原因在 `AddComponetsUndoRedoCommand.Execute()`
（设计器"拖一个控件进来"的规范流程）：

```
1. StartGroup(urm)
2. 对每个子元素： container.AddNodeAt/AddNode(el)   ← 放进布局树
                  AddChildrenForSpec(el)            ← 递归 Add(el, true) 建规格模型
3. 对每个子元素： 从 TableColumnHelper 富化 SpecField 的 11 个属性
4. EndGroup(urm)
```

**这就是 `add_field` 靠"第二次加载"间接达到的效果**——设计器一次做完。所以 `Edit.cs` 的
`add` 直接构造这个命令即可。

命令形状：`Edit.exe <in> <out> add <父路径> <控件类型>[:名字]`
（默认类型 `Button`，默认名走 `ComponentFactory.GetNewName` → `button_1`）。

被测过的父容器：`Grid` ✓、`Group` ✓、`Folder` ✗、`Page` ✗、`HBox` ✗（见 §11.17 的有效接受表）。

**全语料批量**（`batch-action.sh`：逐包往第一个有内容的 Grid 加一个 `Button`，再验不动点）：
**80/80 干净、零失败零跳过**——四个批处理里最干净的一个。两个原因值得记：

- 加按钮**不需要列**，所以写/改侧因"列已用尽"而跳过的两个小表单，这里能跑。
- 它**不动表关联**，所以 `xiyuan/tst/bsft001_wf(c).tzs`（写侧因 `mta/tables.xml` 缺表弹框失败的那个）
  在这里通过。

### 11.19 属性值级比对（验收方法补强）

`RoundTrip` 原来的 `.4fd` 判据是**规范化 XML 相等**——但设计器每次保存都会重排属性顺序，
所以它永远报 DIFFERS，**区分不出"值陈旧"**。这正是让 `add_field` 的 `req`/`style` 不同步
溜过去的原因。

现在补了一层**属性值级**的比对：按名字路径取属性字典，逐个比值，报"有多少个元素的值与
模型不符"。**基线（原生文件）实测 0**，才说明这个判据可用。

**校准过程本身值得记一笔**：第一版把 `fieldId` 也算进去了，结果 89 个包里 **20 个原生
设计器文件**报"不符"，差异**全部是 `fieldId`**（`1 → 57`、`2 → 313` 之类）。这不是缺陷——
`fieldId` 每次保存全量重编号、不是持久标识（正因为如此寻址才只能用 `name`）。排除后归 0。

教训：**新判据必须先对基线跑出 0，再拿去验产出。** 基线非零说明它在量一个实现细节而不是
缺陷；直接拿它判"我的产出有问题"会得到 20 个假警报，而真正要抓的那类问题（属性值陈旧）
反而被淹没。

（顺带解释了为什么工具自身的产出是 0：`FormWriter` 会重建 Record 段，`fieldId` 与模型
一致；原生文件停留在旧编号状态。）

### 11.20 读侧：`info` 与 `info --find`

设计器自己就有这个东西——**「画面结构」页签**（`menu_FormStructure` →
`FormStructureViewer.xaml`）。它的 TreeView 是：

```xml
<TreeView ItemsSource="{Binding Nodes}">
  <HierarchicalDataTemplate ItemsSource="{Binding Nodes}">
    <Image Source="{Binding NodeName, Converter=IconImageControlConverter}" />
    <TextBlock Text="{Binding Name}" />
```

**没有任何独立的 view model**——`ItemsSource` 直接绑到 `XmlElement.Nodes` 递归，节点文字是
`Name`、图标按 `NodeName`。也就是说**设计器自己就把表单呈现为一棵名字树**，这印证了
"寻址只能靠 `name`"（见 §11.11 与 §3）。

`Edit.exe info <in.tzs> [<elementPath>]` 就是把同一棵树吐成 JSON，区别在于**每个节点带上
`path`**——其余命令要的正是它。没有这一步，AI 得先自己解析 `.4fd` 才能调用任何写操作。

节点字段：`tag` `name` `path` `tabIndex` `table` `column` `fieldType` `specNodeType`
`specStatus` `children`。

**`specStatus` 是四态，不是两态**（元素和 action 共用同一个映射）：

| 值 | 含义 |
|---|---|
| 缺省 | 加载自磁盘、未改动 |
| `u` | 改动过，保存时会写 |
| `d` | 墓碑，`ToXml()` 照常输出 |
| `c` | **只在内存里，`.tsd` 中没有**，`ToXml()` 保存时丢弃 |

第一版把 `c` 藏了起来，理由是"报成 `c` 等于宣称磁盘上有个并不存在的节点"——**那个理由是错的**，
而且藏掉之后"磁盘上真有"和"内存里的幻影"变得无法区分。实测三个原生文件的 `c` 全部落在设计器
加载期造的脚手架上：`axmt500_wf(c)` 的 15 个里 11 个是 `Phantom`，其余是 Tree 的
`id`/`parentid`/`isnode`/`expanded`/`name`（`ComponentFactory.cs:72-76`）；`capp113(c)` 的 4 个
里 2 个是**没有 `<act>` 的按钮**（`FindActSpecById` 找不到同名 `<act>` 就自己合成一个
`SpecActionNode`）。**`c` 正好把 §11.14 那个"凡加载期诞生的节点都提升"的坑标了出来**——它现在
在输出里直接可见。代价是 670 元素的表单从 147918 涨到 148173 字节。

**紧凑、不缩进**。缩进版在 670 元素的表单上是 355 KB，紧凑版 144 KB；JSON 阅读器不在乎
空白，而这里每字节都要进 AI 的上下文。

#### `info --find <控件代号>`

用户手里只有「控件代号」——设计器 `字段属性` 面板里那个 `WatermarkTextBox`
（`SpecPropertyEditor.xaml:687`，`IsRequired="True"`，绑 `Name` 双向）。所以这才是实际入口，
`--find` 把它解析成其余命令要的 `elementPath`。

**唯一性是设计器强制的**，三层：

```
ComponentFactory.GetNewName(:438)   唯一命名出口；冲突时枚举 FormSpeDictionary 取
                                    {默认名}{n} 的第一个空闲号。新建 / 粘贴 / 改名全走它
SpecificationInfo.Rename(:1730)     空名 -> throw "'name' field is required"
                                    重名 -> throw Message_NameAlreadyExist（"名称重复"）
                                    查的是 FormSpeDictionary.ContainsKey —— 作用域是整张表单
WatermarkTextBox                    IsRequired + ValidatesOnExceptions，把上面两个 throw
                                    显示成红框
```

实测 **91 个包 / 18,349 个布局元素 / 0 个无名**；唯一的重复是 `<Item>`（ComboBox/RadioGroup 的
**选项值**，`name="1"`/`"2"`，路径形如 `.../xmdc_t.xmdcwf071/1`），不是组件。**所以一个代号
对应一个组件，这条路是类型安全的**——不是模糊匹配，命中了就是它。

输出是**扁平的，不带子树**（`info <path>` 已经能把任意一条展开）：

```json
{"program":"capp113","env":"c","query":"b3_incjwf022","exact":true,"matchCount":1,
 "matches":[{"tag":"Edit","name":"b3_incjwf022","path":"ManagedForm/capp113/.../b3_incjwf022",
             "tabIndex":62,"fieldType":"NON_DATABASE","specNodeType":"FIELD"}]}
```

四条约定：

- **精确命中就不做子串扫描**，所以一个代号即使是另一个名字的前缀也只解析到它自己
- **精确没中才退化成子串**：`--find apca` 会列出 `l_apcasite` / `lbl_apcasite` / `l_apcasite_desc`
  整套（aapp320 每个字段是"输入 + 标签 + 描述"三件套）
- **查不到不是错误**：`matchCount: 0`、退出码 0——参数是查询，不是位置；`info <path>` 才用
  退出码 2 表示"你给的定位不存在"
- **`matches` 与 `actionMatches` 分开**。`<act>` 不在 `FormSpeDictionary` 里（`Rename` 用另一个
  `CheckActionNameIsExists` 单独守），所以元素遍历看不到它。没有 `<Button>` 的纯 action
  （`新增项目` 的产物）只有 `actionMatches`；绑定了按钮的则两边同名各出现一次

正确性验证：对 10 个包各取最深路径与中位路径共 20 次查询，`--find` 报的 `path` 与从 `.4fd`
独立算出的路径**逐字相同（20/20）**。

另：stdout **和 stderr** 都必须写 UTF-8。`Console.Out`/`Console.Error` 走的是 OEM 代码页
（本机 GBK），`Console.OutputEncoding` 对重定向的流不生效——要各自往
`Console.OpenStandardOutput()` / `OpenStandardError()` 包一个 `UTF8Encoding` 的 `StreamWriter`。

### 11.21 尚未实现

| 组件 | 说明 |
|---|---|
| **三个语义用途** | `REFERENCE`（`<rfield>`）/ `MULTILANG`（`<multi_lang>`）/ `PROGREL`（`<pfield>`）。`ComponentFactory.CreateEmptyComponentForBody`（:138）已经写好了这三种"用途"的属性组合，而且它是 **`public static`**（早先记成 `internal` 是错的），可直接调 |
| **`add_field` 的规格/布局同步** | 见 §11.16 末尾的已知缺口：加完字段要补跑 `Transform*` |
| **写侧输出的编码** | `Edit` 除 `info` 外的 stdout 仍走 `Console.WriteLine`（OEM 代码页），管道里是乱码。交互式终端下正常，但要给 AI 读就得和 `info` 一样改走 UTF-8 |
| **目录导出成清单** | 57 个类型各标注：族 / 有无 `colName` / 对应 `SpecNodeType` / per-type fixup |
| **操作层 `apply` + CLI 外壳** | 把 set/del/add 串成一条命令；操作集定型后再做 |
| **`del` 的容器语义** | 只删过叶子字段对。设计器的 `DeleteComponentsUndoRedoCommand` 对 Folder/Table/Tree 有"删光子节点就删容器自己"的特例，以及递归收集子节点——当前 `Edit.cs` 只做单元素删除，没复刻这些 |
| **`wrap` 的对称操作** | 只做了"包"（`AddToContainerUndoRedoCommand`），没做"拆"（`BreakLayoutUndoRedoCommand`） |
| **`wrap` 的 mime 检查不完整** | 用的是字符串重载；`AcceptMimes(ComponentType, XmlElement)`（:1135）还额外挡掉 Grid/Group 的容器子节点，那条没复刻 |

### 11.22 离线能力全景与函数层规划

#### 封装面是 32 个 UndoRedo 命令类，不是那 ~180 个 RoutedCommand

设计器有约 180 个 `RoutedCommand`（`MenuCommands` 96 + `FormCommands` 32 + 其余若干），
但**每个 handler 都只有 3–5 行，收敛到 `SpecDesignerCommon` 的 32 个 UndoRedo 命令类上**：

```csharp
// 移到最前/前一个/下一个/最后 —— 4 个菜单项，1 个类，只差一个 int
new ChangeChildIndexUndoRedoCommand(el, newIndex).Execute();

// 往前/往后新增栏位、新增页签 —— 就是我们 add 已经在用的那个类，只差 index
new AddComponetsUndoRedoCommand(list, parent, index).Execute();
```

命令类分布在**两个目录**：`SpecDesignerCommon/UndoRedo/`（25 个）和
`SpecDesignerCommon/UndoRedoCommands/`（7 个：`SDSpec` / `SpecCited` / `Excluded` /
`ActLocalString` / `FormPos` / `FormSizeComplex` / `ProgRelProgramAttribute`）。

已抽出的构造签名（这份表就是封装清单）：

| 命令类 | 签名 | UI 能力 |
|---|---|---|
| `AddComponetsUndoRedoCommand` | `(list, container[, index｜posX,posY])` | 新增控件 / 往前·往后新增栏位 / 新增页签 |
| `AddToContainerUndoRedoCommand` | `(elements, container, type)` | 包进 HBox/VBox/Grid/Group |
| `BreakLayoutUndoRedoCommand` | `(box)` | 拆箱 |
| `DeleteComponentsUndoRedoCommand` | `(parent, list)` | 删除控件/页签/栏位 |
| `DeleteActUndoRedoCommand` | `(act)` | 删除 Action |
| `ChangeChildIndexUndoRedoCommand` | `(element, newIndex)` | 移到最前/前/后/最后 |
| `MoveComponentsUndoRedoCommand` | `(list, direction, offset)` | 按方向偏移（`Up/Down/Left/Right`） |
| `AlignUndoRedoCommand` | `(elements, option)` | 对齐（`STRETCH/LEFT/RIGHT/TOP/BOTTOM`） |
| `ChangeSizeUndoRedoCommand` | `(elements)` | 尺寸自适应 |
| `ConvertWidgetTypeUndoRedoCommand` | `(element, type)` | 转换控件类型（14 种） |
| `ConvertContainerTypeUndoRedoCommand` | `(element, type)` | 转换容器类型 |
| `RenameUndoRedoCommand` | `(key, oldName, newName)` | 改控件代号 |
| `Cut`/`PasteComponentUndoRedoCommand` + `DesignerClipboardData` | `(container,list)` / `(data,container)` | 跨表单复制粘贴组件（**不暴露**，见清单里"复制"一行） |
| `DragComponentsUndoRedoCommand` | `(sourceContainer)` | 拖拽落点 |
| `MultiFormAttributesUndoRedoCommand` | `(elements, attribute, newValue)` | **批量改属性** |
| `MultiHideUndoRedoCommand` | `(elements, bool)` | 批量隐藏 |
| `FormAttributesUndoRedoCommand` | `(element, attribute, newValue)` | 单个布局属性（**只能在索引器之后**，见 §11.23） |
| `FormSizeUndoRedoCommand` / `FormPosUndoRedoCommand` | `(element, attribute, int)` | Form 尺寸/位置 |
| `SpecAttributeUndoRedoCommand` | `(node)` | 字段规格属性（已用） |
| `SpecTreeAttributeUndoRedoCommand` | `(node, elementName, propertyName)` | Tree 数据来源每一格 |
| `SDSpecUndoRedoCommand` | `(node, newContent)` | SD 规格描述 CDATA |
| `ActLocalStringUndoRedoCommand` | `(actNode, newValue)` | Action 多语言说明 |
| `SpecCitedUndoRedoCommand` | `(node, isCited)` | 引用/取消引用标准 |
| `ExcludedUndoRedoCommand` | `(model, isExcluded)` | 控件排除 |
| `ChangeProgRelProgramUndoRedoCommand` | `(node, program, isDelete)` | 串查程序增删 |
| `ProgRelProgramAttributeUndoRedoCommand` | `(program, progRelNode)` | 串查程序属性 |

命令类之外还有三块**读侧模型 API**，我们基本没用：
`SpecificationInfo`（57 个公开成员，用了约 6 个）、`TableColumnHelper`（约 20 个静态方法，用了 0 个）、
`ComponentTabIndexService`（6 个动作，用了 1 个）。

#### 约 50 个函数的清单

| 组 | 函数 | 底层 |
|---|---|---|
| 会话 (5) | `open` `save` `close` `verify` `list_open` | `SettingManager.OpenSpecFiles` / `SaveSetting` |
| 读 (10) | `form_tree` `find_component` `get_component` `list_spec_nodes` `describe_kind` `list_packages` `list_tables` `list_columns` `list_records` `list_local_strings` | `FormNode` / `*ForView` / `Other*` / 工作区目录 / `TableColumnHelper` / `GetRecords` |
| 属性 (6) | `set_spec_attr` `set_spec_attrs` `set_layout_attr`（含批量 `paths`）`set_layout_attrs` `set_tree_source` `rename_component` | `SpecAttribute` / `MultiFormAttributes` / `SpecTreeAttribute` / `Rename` |
| 结构 (12) | `add_widget` `add_field` `insert_at` `delete` `move` `nudge` `align` `fit_size` `wrap` `break_layout` `convert_widget` `convert_container` | 对应 12 个命令类 |
| ~~复制 (1)~~ | **已摘除**：`copy_component` 不复制（它是 cut+paste，同容器下就是重排），而且 `TargetContainer` 取的是**源容器自己**的相对路径，所以换容器不可达、跨表单必被拒。命令类在设计器里存在，但**不暴露**——完整理由见 `HANDOFF.md §18` |`Cut`/`Paste`/`DesignerClipboardData`（不暴露） |
| 页签 (2) | `add_page` `delete_page` | `AddComponets` + `Page` / `DeleteComponents` |
| 语义/Action (4) | `insert_semantic` `add_action` `delete_action` `set_action_types` | `CreateEmptyComponentForBody` / `ActionTypeDataGrid` 规则 |
| 多语言/选项/串查 (6) | `set_local_string` `set_items` `set_progrel_programs` `set_table_association` `set_spec_description` `set_cited` | `SetFieldLocalStringText` / `ReplaceItems` / `<sr>` / `SDSpec` / `SpecCited` |
| Tab 顺序 (2) | `set_tab_order` `tab_action`（6 动作合一） | `ComponentTabIndexService` |
| 校验/工具 (4) | `validate` `base_data` `set_excluded` `set_code_template` | 设计器自己的校验器 / `MasterDetailView` / `Excluded` |

`move` 一个函数覆盖 8 个菜单项，`tab_action` 覆盖 6 个，`insert_at` 覆盖 2 个。

#### 形态：长驻 JSON-RPC 进程

实测单次进程调用 ~1000 ms，其中约 890 ms 是**与表单无关的固定初始化**；而长驻进程里
一次内存操作 **~0 ms**（P0 探针，见 `HANDOFF.md §17`）。

```
tzs-cli       一次调用：起进程 → 一条 JSON → 打印 → 退出
tzs-server    长驻：stdin/stdout 逐行 JSON-RPC（MCP 直接套）
mcp-adapter   薄适配：manifest → MCP tools
```

每个函数在 **manifest** 里声明一次（名字、参数、类型、说明、返回结构），由它生成
CLI 帮助、参数校验、MCP tool 列表。关键字段是 `describe_from`——属性名不是自由字符串，
是运行时从节点自己物化的属性集里取的（`field` 23 个、`hfield` 12 个、`pfield` 7 个、
`rfield` 12 个、`tree` 5 个、`act` 8 个，实测每种都完整物化、一个不漏）。

#### 阶段

| 阶段 | 内容 | 估时 |
|---|---|---|
| ~~P0~~ | 探针：长驻/多包、校验器、布局属性写路径 | 已完成，见 §11.23 |
| P1 | manifest + JSON-RPC 循环 + 会话组 + 读组（`get_component` 必须吐规格属性；`list_columns` 解除 `add_field` 前置阻塞） | 3–5 天 |
| P2 | 属性组 4 个 + `describe_kind` 白名单 + `validate`（含修 `set` 的静默污染） | 2–3 天 |
| P3 | 结构 12 + 页签 2 + Tab 2 | 4–6 天 |
| P4 | 语义 + 多语言 + 选项 + 跨表单复制 | 4–6 天 |
| P5 | MCP 适配 | 1–2 天 |

合计约 15–22 个工作日。

#### 离线范围内的两个边界

- **docx 汇出**：`DocxExportAdapter` 强依赖已渲染的 WPF 视觉树（含截图 + `Thread.Sleep(500)`），
  无界面静默返回。数据结构部分（`FormSpeDictionary` + `Actions`）可自己重写。
- **`.tzc`/`.tzg`/`.tzd` 4GL 代码编辑**：也是离线，但**没有错误列表**（`FglParser.ParseErrorCheck`
  是空壳）、**没有编译能力**、**没有符号表**。`DiffManager.SimpleDiff(string,string)` 是干净的两
  字符串比对面。建议作为**另一条线**，别混进 `.tzs` 的进度。


### 11.23 P0 探针的机理（`test/Probe.cs`）

结论在 `HANDOFF.md §17`，这里只记机制。

#### 为什么校验器必须先 `SaveToForm()` / `SaveToTSD()`

```csharp
public XElement FormElement { get; private set; }   // :62  —— 私有 setter
public XElement TSDElement  { get; private set; }   // :52  —— 私有 setter
// :141 构造时一次性赋值
this.FormElement = XElement.Parse(this._tzpManager.GeneroFormString);
```

两个都是**构造时的快照**，之后只有 `SaveToForm()` / `SaveToTSD()` 会重新赋值
（它们在类内，所以能用私有 setter）。而校验器读的正是它们：

```csharp
this._FormValidater.RunWorkerAsync(this.FormElement);      // :524
this._TSDValidater.RunWorkerAsync(this.TSDElement.ToString());  // :303
```

**不先重建，校验的就是"加载时的文件"，任何编辑都不可见。**

两个 worker 的唯一触发点是 `SaveSettingEvent` 的订阅者
`SaveSpecificationInfo(PackageKey)`（:243），顺序是：

```
SaveToTSD() → SaveToForm() → SaveBinding() → InitTSDValidateWorker() → InitFormValidateWorker()
→ 写 .4fd/.tsd/.tsd2 到包里
```

**所以"只校验不写盘"的配方是：`SaveToForm()` + `SaveToTSD()` + 两个 worker + 订阅
`DocumentErrorsEvent`。**（`RoundTrip` 只调了前两个，从没触发校验。）

#### 布局属性：索引器 vs 命令类

`XmlElement.this[string]`（:1888）的 setter 顺序：

```
1. GetAttribute(key) == null  → return        ← 白名单：不能新建属性
2. attribute == value         → return        ← 同值短路
3. key == "repeat" 且父是 ScrollGrid 且 value != "true"  → throw Message_RepeatInScrollGrid
4. key ∈ {stepX,stepY} 且父非 ScrollGrid → return     ← 静默忽略
5. key ∈ {rowCount,columnCount} 且父非 ScrollGrid → return；否则 Math.Max(...)
6. IsBatchSet 时只处理 hidden
7. new FormAttributesUndoRedoCommand(this, key, value).Execute()
```

而 `FormAttributesUndoRedoCommand.Execute()` 内部**直接** `this._element.SetAttribute(...)`
（只有 `gridWidth`/`gridHeight` 走属性、`rowCount`/`rowHeight`/`totalRows` 走 `MeasureSize`）。
**所以直接构造它 = 跳过 1–6 全部规则。**

实测对照：

| | 写已存在属性 | 写不存在属性 | `gridWidth 1557 → 1` |
|---|---|---|---|
| 索引器 | 生效 | **拒绝**（`<null>`） | 夹到 **1555**（`MinGridWidth`） |
| 直接调命令类 | 生效 | **写进去** | — |

#### `ValidateForm` 的唯一行为消费点

```
SpecDesignerCommon/ViewModel/XmlElement.cs:2522
    if (!PreferenceManager.Current.Settings.ValidateForm) return;   // 在 CheckOverlapping 里
```

`CheckOverlapping` 每次 `SetAttribute` 都跑，所以这个偏好实际管的是**重叠检测**。
我们所有工具一直设 `false`——注释写的是"避免模态框"，真实效果是**一直关着重叠检测**。

`PreferenceManager.Current.Settings` 为 null 时，加载期第一次 `SetAttribute` 就 NRE
（走 `CheckOverlapping`）——这就是新程序初始化必须设 `_preferenceModel` 的原因。

### 11.24 Agent 小组契约（Wave 0 冻结，实现期不许改）

本节是并行实现期的**编码契约**。契约不冻结就并行 = 必然返工。

#### 0.3 两个分歧的裁决（已查证）

| 分歧 | 裁决 | 依据 |
|---|---|---|
| `Call` 重载绑定 | **统一到强版**（按参数类型可赋值性筛，再 fallback `loose`） | `AddField.cs:58` 的弱版只按参数个数选，选中不匹配的重载时抛的是 `ArgumentException`——而弱版只 `catch (TargetInvocationException)`，异常直接冒出去。`SpecificationInfo.Remove(string)` vs `Remove(SpecActionNode)` 就是实例 |
| `NODE_SLOTS` | **7 项为准**，删掉 `SpecItem` | `SpecItem` 在**整个设计器源码里不存在**（只出现在 `AddField.cs:470`）。`Prop()` 对未知名字返回 null，所以它是无害的死条目。`FormSpecModel` 真正持有的是 7 个：`SpecField` `SpecHelpCode` `SpecProgRel` `SpecReference` `SpecMultiLang` `SpecTree` `SpecAction`，另有通用槽 `SpecNode`（`AbstractSpecNode`，指向其中之一）与 `SpecExcludeNode`（由 `ExcludedUndoRedoCommand` 单独设） |

**注意 `Promote` 与 `NODE_SLOTS` 是两件事**，只是共用同一组槽位：`Promote` 是"这个元素是新建的，把它持有的**所有**槽位解除 CREATE"，`NODE_SLOTS` 是"`kind:` 指哪个槽"。共用一个 7 项常量即可。

**`Call` 换强版是行为变更**，不是纯去重——`AddField` 必须重跑 `batch-write.sh` 并由 W1 回归门验证。

#### (a) 请求 / 响应信封

逐行 JSON，stdin/stdout（管道传输见 (f)）。一行一个请求，一行一个响应，**不许跨行**。

```json
{"id":1,"fn":"set_layout_attr","args":{"handle":"h1","path":"...","attr":"case","value":"upper"}}
{"id":1,"ok":true,"result":{...}}
{"id":1,"ok":false,"error":{"kind":"validation","message":"...","detail":{...}}}
```

`error.kind` 的取值与语义——**AI 的自纠能力完全取决于这个区分**：

| kind | 含义 | 调用方该怎么办 |
|---|---|---|
| `validation` | 参数本身不合法（未知属性、类型不对、缺参数） | 可以自纠，`detail` 里给可用值 |
| `not_found` | 路径 / 代号 / 句柄不存在 | 可以自纠，`detail` 里给近似候选 |
| `designer` | 设计器自己的规则拒绝（`repeat` 门禁、重名、被引用不可改） | **不要重试**，转述给用户 |
| `internal` | 未预期的异常 | 上报，不要重试 |

`id` 原样回显；不回显就无法对上。

**完整的错误码表。** W3-F 发现这段契约原先只写了四个 `kind`，而 `E_NO_OP` / `E_ATTR_CLAMPED`
只活在代码里——于是下一个 agent 又会把 no-op 当成错误。补上：

| code | kind | 含义 |
|---|---|---|
| `E_BAD_REQUEST` | validation | 行不是合法 JSON 对象 |
| `E_UNKNOWN_METHOD` | validation | 不在 manifest 里 |
| `E_BAD_PARAM` | validation | 参数缺/类型错/枚举越界；`detail.param` 点名 |
| `E_NO_HANDLE` | not_found | 句柄不存在或已关闭 |
| ~~`E_HANDLE_BUSY`~~ | ~~validation~~ | **已删除（2026-09-25）**：表里写着、实现里从没有任何地方抛它（引擎唯一的句柄状态判定是 Closed，那是 `E_NO_HANDLE`）。一个抛不出来的码是 advertised-but-inert，留着只会让调用方永远为一个不存在的结果写分支。 |
| `E_PATH_NOT_FOUND` | not_found | name-path 解析不到 |
| `E_NO_SPEC_NODE` | not_found | 该元素没有这种规格节点；换一个 kind |
| `E_ATTR_NOT_WHITELIST` | validation | 索引器拒绝——元素身上没有这个属性；`detail.legal` 列出合法集，`detail.hint` 给最接近的名字 |
| `E_ATTR_VALUE_ILLEGAL` | validation | 属性名合法但**值**不在设计器声明的集合里（`<工作区>/mta/mod-fd.spec` 的 `contains:` / `type`）；`detail.legal` 给值集、`detail.hint` 给最接近的值、`detail.source` 说明依据、`detail.written:false` 表示一个字节都没写。**只查布局侧**，原因见 7.5.5 的表 |
| `E_ATTR_PARTIAL` | designer | 复数写入（`set_spec_attrs` / `set_layout_attrs`）写了一半：名字与值都已在写之前全量校验过，所以失败来自模型；`detail.applied` / `detail.failed` 两栏说明哪几个落了、哪几个没有，**模型不是原样了**，重发只该发 `failed` 里那几个 |
| `E_KEY_IN_USE` | designer | 同一个 `ProgramKey` 已被占用；先 close |
| `E_DESIGNER` | designer | 设计器自己的规则拒绝（repeat 门禁、重名、被引用不可改） |
| `E_FATAL_LOAD_TIMEOUT` | designer | 加载超时；**进程随即退出**，见 (h) |
| `E_SERVER_DIED` | — | 仅 CLI 侧：EOF 且无帧 |
| `E_NOT_IMPLEMENTED` | internal | manifest 声明了但没实现 |
| `E_INTERNAL` | internal | 未预期异常；`detail.exception` 是拆到最内层的那一个 |
| **`E_NO_OP`** | **成功** | `old == new`，设计器同值短路，未改动任何值 |
| **`E_ATTR_CLAMPED`** | **成功** | 应用了，但被 `MinGrid*`/`rowCount` 门禁改了值；`result.written` 是真值 |

**最后两条是成功帧（`ok:true`），不是错误。** 它们存在于 `result` 里而不是 `error` 里，
因为 `kind` 的全部意义就是告诉调用方"要不要自纠"——"已经是这个值了"不是失败，不该让调用方去查错。
三者必须可区分，各带恰好一个正向标记：

| 结局 | 标记 | `changed` | `written` |
|---|---|---|---|
| 真应用 | `applied:true` | `true` | 新值 |
| 应用但被夹紧 | `clamped:true` + `code:E_ATTR_CLAMPED` | `written != old` | 模型实际持有的值 |
| 无操作 | `noop:true` + `code:E_NO_OP` | `false` | = 请求值 |

**一个刻意的例外**：`stepX`/`stepY`/`rowCount`/`columnCount` 写在非 `ScrollGrid` 子元素上时，
索引器**静默丢弃**（`XmlElement` 索引器第 4/5 条）。这里**仍然报 `E_DESIGNER` 而不是成功**——
它既不是夹紧也不是同值短路，请求被丢掉了；报成功就正是本项目要消灭的那种静默损坏。

#### (b) 会话 / 句柄模型

```
open  → handle      加载包 + 注册 UndoRedoManager（一次）
ops   → 用 handle
save  → handle + outPath
close → 释放
```

**硬约束**：`UndoRedoManager` 还注册着的 key **不能重新加载**——`SetInitGridY` 抛 `illeagal call`（P0 实测）。
所以 `RegisterUndoRedo` 是 **`open` 阶段的一次性步骤，不是每次操作**。`Edit.cs` 里它被内联 4 次
（:340/:442/:507/:637），那是因为每个进程只做一个操作；迁到 server 必须上移到 `open`。

**一律按 key 寻址，永不读 `TzpManager.Current`**——P0 实测它会被 `GetTzpManger` 的副作用改写成
最后加载的那个包。

**`save` 的返回里带 `sha256`（刚写进 `out` 的字节的摘要）与 `key`（新包会占用的 ProgramKey，与源包
相同）。** 这两样是给调用方**自己闭合**"改动落了盘"这一环用的（2026-09-25 的评测量出来：那一步过去
要三次引擎调用）。链条是：`get_component` 显示内存模型里有这个改动 → `save` 回它写下去那份字节的
摘要 → `sha256sum <out>` 与摘要一比，就证明盘上就是这个包。它成立的前提只有本节的**不动点**性质
（save 渲染的就是当前模型，语料回归在每个包上验它）加一条"摘要就是摘要"。不这么做的代价是
**自证循环**：`get_component` / `verify` 读的都是 save 写盘所依据的那个内存模型，拿它们回读证明
不了盘上的字节。注意 `key` 那一半：新包与源包同 key，所以想 `open` 新包必须先 `close` 源会话，
否则 `E_KEY_IN_USE`。

#### (c) manifest 条目

单一事实来源，由它生成 CLI 帮助与参数校验：

```json
{"fn":"set_spec_attr","group":"属性","desc":"改字段规格属性","writes":true,"slow":false,
 "args":[{"n":"handle","t":"string","req":true},
         {"n":"path","t":"string","req":true},
         {"n":"kind","t":"enum","values":["field","hfield","pfield","rfield","mlfield","tree","act"]},
         {"n":"attr","t":"enum","req":true,"from":"describe_kind"},
         {"n":"value","t":"string","req":true}]}
```

`from:"describe_kind"` 是关键：**属性名不是自由字符串**，是运行时从节点自己物化的属性集里取的
（`field` 23 / `hfield` 12 / `pfield` 7 / `rfield` 12 / `tree` 5 / `act` 8 个，每种都完整物化、一个不漏）。
这同时解决了"AI 该填什么"和"拒绝非法值"。

**布局侧那一半是同一张表的另一个方向**（2026-09-25，§11.9 第 15 条）。`describe_kind --kind layout`
回的**不是**一串名字，而是每条 `{name, type, values[], initial}` —— 名字来自这张表单上出现过的
属性并集，后三项来自工作区 `mta/mod-fd.spec` 的 `<PropertyInfo>`，与 `set_layout_attr` 拒绝坏值时
写进 `detail.legal` 的**是同一张表**（`SpecValues`：一个 `Check` 拒人，一个 `Describe` 报人；
BOOLEAN 的 `true|false` 是一处声明的常量，两边共用）。`values` 缺省表示"没有任何地方声明取值集"
（TEXT / FDSTYLE 之类自由格式，以及 `gridWidth` 这种只有 range 的）。
为什么要有这个出口：在那之前只有"拒绝"那一半 —— 调用方写之前问不到，只能猜，而猜错**不报错**，
只是做了另一件事（`hidden` 与 `invisible` 就是一例：前者 ENUM `false|true`，后者是 BOOLEAN，
4.2 名 `isPassword`）。两半一致由语料回归的
`TestCorpusLayoutValueRuleAgreesWithItsDiscovery` 在每个真实工作区上钉住。

`slow:true` 标记耗时函数（`validate`：114 元素 1.6 s、670 元素 **10.4 s**），提醒不要每步都调。

`errors[]` **是算出来的，不是手写名单**：除了每个动词自己声明的那几个，`AdvertisedErrors` 还会按
它的参数与性质补上蕴含的码 —— 收 `handle` 补 `E_NO_HANDLE`、收 name-path（`role:"component-path"`）
补 `E_PATH_NOT_FOUND`、**写动词**收 `kind` 补 `E_NO_SPEC_NODE`。理由：这三条本来就已经写在声明里，
算出来的忘不掉；手写名单忘过 —— 2026-09-25 实测：十二个能答"什么都没改"的动词里，八个的
`errors[]` 从没出现过 `E_NO_OP`，另一个能答 `E_NO_SPEC_NODE` 的动词也不在表里，于是**照 `--help`
写分支的调用方漏掉那些码**。两个成功码（`E_NO_OP` 与 `E_ATTR_CLAMPED`）同样要出现在 `errors[]` 里：
它们虽然走 `result` 而不是 `error`，却是调用方必须能与 `applied` 区分开的那个结局（见 (a) 的表）。

#### (d) 写入的两条铁律

1. **布局属性一律走 `XmlElement` 索引器**（`el[attr] = value`），**绝不**直接
   `new FormAttributesUndoRedoCommand(...)`。索引器依次做：白名单（属性不存在就拒）→ 同值短路 →
   `repeat`/`stepX`/`rowCount` 门禁 → `MinGrid*` 夹紧 → **最后才**调那个命令类。
   直接调它 = 跳过前面全部（P0 实测：会把不存在的属性写进去）。
   **`name` 是这条规则的例外，而且是唯一一个**：它既是名字路径的构成，又是
   `FormSpeDictionary` 的键。索引器写得到模型、写不到 `.4fd` 文本，所以**布局侧的
   `name` 只改得动一半**。实测（2026-09-27，Go 侧函数面关卡跑 65 个真实包）：写完之后新代号
   查得到、文本里还是旧代号，引擎自己在同一个容器上的 `add_field` 报
   `E_INTERNAL：模型子节点 … 在布局文本里没有对应元素——模型和文本已经分叉，不能继续写`，
   `delete`/`nudge`/`align`/`fit_size` 拿新代号一律 `E_NOT_FOUND`。
   所以**布局侧那四个入口**（`set_layout_attr` 的 `path`/`paths` 两种形态 + `set_layout_attrs`）
   **在入口就拒 `attr="name"`**（`E_BAD_PARAM`，`kind:validation`，实现在 `Attr.RejectRename`）。
   **规格侧不拒，这个不对称是实测的**：`set_spec_attr {attr:"name"}` 走
   `SpecificationInfo.Rename`，元素名一起推过去，收尾的 `Patch()` 也把文本改到了 ——
   改完之后 `find_component` 解析的是**新路径**、旧名一个都不剩、容器里的 `add_field` 照常成功。
   把它一起禁掉等于砍掉一条能用的路。`rename_component` 仍比它多一样东西：**重名预检**
   （`IsExists`）。**不要**在布局侧补第二份改名实现。
2. **规格属性写之前先查白名单**——同一个病在规格侧的形态，就是 `Edit set` 往 `.tsd` 里写
   `case="upper"` 那种垃圾。

#### (e) 函数注册契约

```csharp
public delegate object Fn(Session s, JObject args);   // 返回可序列化对象；抛出即错误
// src/Designer/Fns/*.cs 每个文件暴露：
public static void Register(IDictionary<string, Fn> into)
```

`Server.cs` 里写死调用（**模块清单在 Wave 0 冻结**，否则 Server 得等所有模块写完才知道要调谁）：

```csharp
Read.Register(map); Attr.Register(map); Validate.Register(map);
Verify.Register(map); Session.Register(map);
```

**JSON 用安装目录现成的 `Newtonsoft.Json.dll`（v9.0.0.0，已确认存在）**，不手写解析器。
编译期加 `-r:"D:\APPS\T100设计器_1.0.0.251_免安装\Newtonsoft.Json.dll"`；运行期现有的
`AssemblyResolve` 已会从 `INSTALL` 解析它。

#### (f) 传输：守护进程（这条决定成败）

P0 实测：单次进程启动 **~1000 ms**（其中 ~890 ms 是与表单无关的固定初始化），同进程内一次操作 **~0 ms**。
**如果 `tzs-cli` 每次都起新进程，长驻架构一分钱都省不下来。**

```
tzs-server.exe  常驻，监听 \.\pipe\tzs-cli   （命名管道：无端口冲突，无需鉴权）
tzs-cli.exe     连接；连不上就 spawn 一个再重试（首次约 1 s，之后 ~0 ms）
tzs-cli stop    收工
```

**用户界面只有 CLI**，守护进程是内部形态。自检必须包含**真实的两次 CLI 调用并对比耗时**——
第二次若仍是 ~1 s，说明退化成了一次性进程而没人发现。

#### (g) 会话模型修订：`ProgramKey` 会碰撞，禁止静默驱逐

**这是 Wave 0 论证时才发现的问题，比原设计严重。**

`TzpManager.CreateKey`（TzpManager.cs:229）只用**程序名 + 包类型**建 key：

```csharp
this.ProgramKey = this.CreateKey(this.ProgramName, this.Type);   // :317
private PackageKey CreateKey(string program, TzpType packType)   // :229
    => new PackageKey(program, packType);                        // 不含路径！
```

语料里**四个文件撞同一个 key**（程序名都是 `aapp320`、都是 Form 类型）：

```
aapp320(c).tzs                        aapp320(c) - 测试备份-修改前.tzs
aapp320(c)_AIADD.tzs                  aapp320(c) - 测试备份-删除前.tzs
```

而现有 `LoadPackage` 的处理是 `map.Remove(k); map.Add(k, t)` —— **静默丢弃前一个句柄，
但它的 `UndoRedoManager` 还留在 `undoRedoManagerMap` 里**。单文件测试永远发现不了；
长驻服务里表现为"打开 B 之后 A 的数据变成 B 的"。

**契约：**

```
handle = "h" + 自增整数。永远不是路径，永远不是 ProgramKey。

open(path)  → state = Loaded    （进 tzpMap、EAM.CreateInstance，但【不注册】UndoRedoManager）
首次写操作   → state = Mutable   （注册 undoRedoManagerMap[key]；对句柄不可逆）
close(h)    → 摘 tzpMap[key] / undoRedoManagerMap[key] / EAM[key]；state = Closed，句柄串永不复用

open(path2) 而 key(path2) 已有活句柄 → E_KEY_IN_USE
                                       force:true 已于 2026-09-25 从 manifest 删除（从未实现，
                                       且"接管占用者"正是契约禁止的静默驱逐）；要拿一个被占用的
                                       key 只能先 close 占用者再 open（§11.24 (g-1) 实测可行）
```

- **禁止 `map.Remove(k); map.Add(k,t)` 式静默驱逐**——`E_KEY_IN_USE` 存在的全部意义就是不让一个活的
  `UndoRedoManager` 变成孤儿。
- `list_open` 必须回吐 `key`，让调用方**看得见**两个文件撞了。
- 读操作在 `Loaded`/`Mutable` 都合法且不改变状态；`save` 不改状态、在 `Loaded` 就可用
  （那正是不动点性质）。

**加性扩展（2026-09，`Rpc.FindByKey`）：`args.handle` 也接受"有意义的键"。** 除 `h<N>` 之外，
可以写**程序名**（`aapp320`）或 **ProgramKey**（`aapp320|Form`）；`Rpc.Resolve` 先按句柄精确匹配
（句柄恒为 `h`+数字，不可能与程序名混淆），未命中再按这两者查 `_open`；查不到时错误里列出候选
（含 `key`）并说明三种写法。理由：句柄是易失的随机号，让调用方在每条命令之间搬运它，等于逼它用
随机号说"我要改 aapp320 的表单"。线上形状一个字没变（仍是字符串、仍叫 `handle`），
`Manifest.Check` 不动 —— 这是**允许更多写法**，不是收紧契约。
匹配到多个（同名程序的不同类型同时开着）时返回 `bad_param` + `detail.candidates`，**不猜**。

**新 P0 问题（Wave 0 必答）**：`close` 之后能否重新 `open` 同一个 key？
`CheckUndoRedoManager`（SettingManager.cs:79）只是 `undoRedoManagerMap.ContainsKey(key)` 的裸查表，
所以**理论上**摘掉就能重新加载——但从未验证过，且 `EventAggregatorManager` 的 per-key 实例
现有代码从不释放。若 close/reopen 不干净，会话模型就退化为"**一个 ProgramKey 一个句柄，进程内不允许重开**"，
那是完全不同的 CLI 契约。**必须先答再写 `Fn/Session.cs`。**

#### (h) 致命退出契约（加载超时）

看门狗是后台线程，唯一出路是 `Environment.Exit(3)`；**主线程此刻卡在
`Activator.CreateInstance` 里，超时无法被放弃**。设计器已经在无消息泵的 STA 线程上弹了模态框，回不去。

所以加载超时在服务里是**终结性**的：

1. server 写 `{"id":<id>,"ok":false,"error":{"code":"E_FATAL_LOAD_TIMEOUT",...}}`，**flush stdout**，然后 `Environment.Exit(3)`
2. `tzs-cli` 把"致命帧之后立刻 EOF"识别为 `E_FATAL_LOAD_TIMEOUT`；"完全没有帧的 EOF"识别为 `E_SERVER_DIED`。
   **绝不能让调用方看到 JSON 解析错误**——那会让 AI 以为"这个表单没有该元素"。
3. wrapper **自动重启，但不重试失败的那个请求**（加载超时是确定性的：`mta/tables.xml` 缺表）

#### (i) `validate` 的两条实现约束

1. **`PreferenceManager.Current.Settings.ValidateForm` 必须在 `finally` 里恢复成 `false`。**
   它是**进程级**的，`XmlElement.CheckOverlapping` 在**每次** `SetAttribute` 都读它——包括**加载别的包**的时候。
   忘了恢复，别的包的加载会开始抛异常，而单包测试看不见。
2. **不许沿用 `Probe` 的 `Thread.Sleep(600)`**——那是竞态，只是碰巧过了。改为用
   `DocumentErrorsEvent` 处理器 signal 一个 `CountdownEvent`，带 60 s 上限，再加 300 ms 静默期才判定结束。
3. 判据精确化：`newErrors = multiset(after) − multiset(before)`，**只要 `newErrors` 里没有 ERROR 级即通过**；
   永远报完整 delta，不要压成一个布尔。`axmt500_wf(c).tzs` 未改动就有 11 条 WARNING——这是基线机制的证据，不是文件脏。

#### (j) 移植时不许照搬的东西

| 不要照搬 | 原因 | 改成 |
|---|---|---|
| `Edit.cs` 的 `_specDic` / `_infoCount` 模块静态 | 两个 handle 同时开会串扰 | 作为**参数**传递 |
| `Edit.add` 的晋升逻辑与 `AddField.Promote` 合并 | 两者**规则不同**：`Promote` 额外以 `colName` 非空为门、还要扫 `si.fieldStrings`；合并会导致按钮（无 `colName`）不再晋升而 `act` 失效，或把 Tree 的 5 个脚手架子节点错误晋升（SPEC §11.14 明确不许） | 保持两个具名方法：`PromoteSlots` / `PromoteByColName` |
| `RoundTrip.cs` 的输出格式 | 15 列 `SUMMARY` + `cut -f9` 基线逻辑焊死在四个批脚本里 | Wave 1–3 **不许改**；它是法官，被告受审期间不能换法官 |

#### (k) 里程碑的演示方式（关键）

**必须演示为"一个 server 进程连续处理六个请求"，不是六次 `tzs-cli` 调用。**

一次性 wrapper 每次都要付那 ~1000 ms，那正是长驻架构要避免的。用六次 CLI 调用演示，
**等于什么都没证明**。所以 `slice.sh` 要直接对 `tzs-server` 的管道驱动；`tzs-cli` 作为便利壳另行演示。

#### (g-1) close/reopen 已实测：**可以**，但 close 不止三步

`test/ProbeReopen.cs`（sha256 `07ca34193ee74988…`）在真实包上跑完三个测试。
（这个 sha 原先写的是 `4f8c06b8…`，**和那个文件的任何一个历史版本都对不上** ——
一条没人检查的 pin 漂了多久也没人知道。2026-09-27 改成真值，并由
`internal/dev/tzs` 的 `TestEngineDocsCiteRealProbeHashes` 逐条盯着：
以后改这个探针就必须连这句一起改。）

**结论：key 可以释放并重新打开。** §11.24(g) 的会话模型成立，不需要退化成"一个 key 一个句柄、不许重开"。
`CheckUndoRedoManager`（SettingManager.cs:79）确实是裸的 `undoRedoManagerMap.ContainsKey(key)`，
摘掉那一个条目就足以让 key 重新可加载。重开是**真正的新加载**（181 ms → 重开 78 ms）：
重新解包、重读 `.4fd`、内存里的改动**消失**、元素数一致、新句柄完全可写。

**但 close 不是三个 Remove 就够。** 每次 open 会给 Global 聚合器挂上**三个**订阅者，
而只有一件事能摘掉它们：**发布 `TzpFileClose`**（即 `SettingManager.CloseFile`，SettingManager.cs:517）。

| 订阅者 | 订阅 | 退订 |
|---|---|---|
| `TzpManager.TzpFileClosed` | TzpManager.cs:225 | :856 |
| `SpecificationInfo.OnTzpFileClose` | SpecificationInfo.cs:149 | :289 |
| `SpecificationInfo.SaveSpecificationInfo`（挂在 `SaveSettingEvent`） | SpecificationInfo.cs:148 | :287 |

实测订阅者计数：

```
一次 open 之后                        : TzpFileClose=3  SaveSettingEvent=1
只做三个 map Remove 之后              : TzpFileClose=3  SaveSettingEvent=1   ← 泄漏
发布一次 TzpFileClose 之后            : TzpFileClose=1  SaveSettingEvent=0   ← 全部回收
```

**泄漏的 `SaveSettingEvent` 那个最危险**：它的函数体是
`SettingManager.Get().GetTzpManger(this.Key).SaveSpecificationInfo(...)`（SpecificationInfo.cs:260），
按 key 解析——而那时该 key 已经属于**另一个包**。于是下一次保存会把一个**已关闭包的模型写进新包**。

**所以 close = 发布 `TzpFileClose(key)` + `EventAggregatorManager.Remove(key)`。**
（`CloseFile` 自己会做 tzpMap / undoRedoManagerMap 的摘除和 `ActionDefaults = null`，
但**不碰** per-key 的 EAM 实例。）已按此修正 `Session.Close()`。

**另外三条必须吸收的：**

1. **失败的 open 会毒化 key。** `open(B)` 在占用者是 Mutable 时抛
   `Exception: illeagal call SetInitGridY`（**是 Y 不是 X**，§11.24(g) 原文记错了；两者共享
   `GetChildNode` 的守卫，哪个先触发取决于表单）——而且**抛出前 `map.Add(k,t)` 已经执行**，
   留下一个 `SpecificationInfo` 从未赋值的半成品 `TzpManager`，`urmMap[key]`/`EAM[key]` 也都还在。
   **所以失败必须回滚**，否则那个 key 从此永远是"被占用"。已按此修正 `Session.Open()`。
2. **够不着 close 的句柄与占用者不是一回事。** 碰撞周期里 `close(A)` 实际删掉的是 **B**——
   摘除是按 map key 而不是按句柄。`close` 必须容忍"不存在"，且不得假设被删的就是所点名的那个。
   实测里 `close(A)` 之后 `TzpManager.Current` 还指向一个陈旧句柄。
3. **close + reopen 是"关闭 + 全新打开"，不是"恢复"。** 第二次加载建的是**新的** `TzpManager`
   与**新的** `SpecificationInfo`，旧句柄什么都不带过来——调用方想留的东西必须重读。

#### (l) 测试范围：日常自检用 T1，全语料只在关卡

**全语料（2026-09-19 前是 81 个文件，现 67 个——`xiyuan/tst` 模块已删除）是关卡手段，不是迭代手段。** 每个文件起一个完整设计器进程（~1 s 起步），
四个批脚本一轮 20 分钟以上——把它用在每次自检上会让 agent 的反馈延迟毁掉迭代速度。

| 档 | 文件 | 用于 |
|---|---|---|
| **T1**（默认，约 1 s） | `cs_excel_in_xmdl_s01(c).tzs`（14 元素，xiyuan/tst）· `cpmp530(c).tzs`（32，tpl=Q，已知 posX 漂移）· `aapp320(c).tzs`（114，tpl=P） | **每个 agent 的每次自检** |
| **T2**（+1 个） | `aist310_wf(c).tzs`（577，Tree + progrel + 34 act） | 改动碰到 Tree / 晋升 / 结构时 |
| **T3**（单次） | `axmt500_wf(c).tzs`（678，未改动即报 11 条 WARNING） | 只在验校验基线时 |
| 全语料 | `./batch*.sh` | **只在 W1 门 / W2 门 / 合并前** |

T1 三个文件覆盖了跨工作区、tpl 的 F/P/Q、有无 Tree、有无 progrel/rfield、14→114 元素——
覆盖的是**变化轴**而不是数量。

给 agent 写任务说明时必须**写死用哪几个文件**；写"对几个语料文件跑"这种模糊话，agent 会挑最大的那个。

#### (m) W2 挖出的三个"看起来像没错误"的静默故障

两个 agent 在 W2 里各挖出一个，都是**不报错、只是结果悄悄变错**的类型。
它们能被抓到，唯一原因是验收里放了**正向对照**（故意造一个必须被抓到的错误）——这条值得写进规矩。

**1. Prism 单参 `Subscribe(Action<T>)` 弱引用持有处理器。** W2-C 的闭包被 GC 掉，而订阅还活着：
`validate` 有时报 0 条、有时报 1 条；而同进程里另开一个原始订阅则每次都收到。
修法是 `Subscribe(Action<T>, keepSubscriberReferenceAlive: true)`。

> **`test/Probe.cs:400` 有同一个隐患**——这也是它当时需要固定 `Thread.Sleep(600)` 的部分原因。
> P0 探针 2 的结论仍然成立（有正向对照佐证），但那个机制里有一条潜在竞态。

**2. `InitFormValidateWorker` 每次调用都重新挂 `DoWork`，且从不摘除**
（`StartValidateForm` 里唯一的摘除写的是 `-= this.StartValidateTSD`，是个笔误空操作）。
`BackgroundWorker` 的事件是多播委托，所以**第 N 次 `validate` 会把每个发现发布 N 遍**——
`axmt500_wf(c).tzs` 那 11 条 WARNING 基线，第二次 `validate` 会变成 11 条**新增** WARNING，
在一个没改过的文件上伪造出回归。修法是 `Init*` 之前先丢弃已结束的 worker。

**3. `.tsd` 的字节一致性有例外，且不是 `save` 的锅。** `cpmp530(c).tzs` 和 `axmt500_wf(c).tzs`
上，**设计器自己的重生成会清空 `<table>/<tbl>` 表关联段**。证据：`verify`（不写盘）报出同样的差异，
且判官的不动点列完全吻合。所以"`.tsd` 逐字节相同"这个判据**必须带这个例外**。

#### (n) 传输：管道名必须是派生的，不能是常量

`\.\pipe\tzs-cli` 是**机器级全局**的，一下午咬了两次：W2-C 的 CLI **静默连上了 W2-A 的守护进程**，
而那个进程服务的是同一程序集的**另一个构建**（陈旧行为，客户端完全无从分辨）；
W2-B 因此**刻意绕开 `tzs-cli`**，因为守护进程一个进程绑一个 workspace，
而 T1 跨两个 workspace，一个游荡的守护进程会卡住别人的 `open`。

**现在是派生的：`tzs-cli-<workspace 哈希>-<本程序集 MVID 前 8 位>`。**
workspace 进去是因为 `Designer.Boot` 明确不可重定向（一个进程只服务一个模块）；
MVID 进去是因为它每次构建都变，于是**客户端永远连不到服务陈旧字节的守护进程**——
它会去起一个匹配的。代价是旧构建的守护进程会成为孤儿，`tzs-cli stop` 负责当前这个。

#### (o) `dry_run`：把模型放回调用前，靠的是**重建会话**而不是撤销栈

写动词（36 个）与 `save` 都收一个可选的 `dry_run`：真跑一遍这个动词，把它的答案整份放进
`preview`，然后把模型放回调用前。**返回体是一个信封**：

```json
{"dryRun":true,"verb":"add_field","handle":"h1",
 "preview":{ …这次调用本该返回的东西，逐字… },
 "reverted":{"mode":"session-rebuild","complete":true,"renderMs":2.1,"loadMs":71.8,"sha256":"…"},
 "note":"dry-run：preview 是这次调用本该返回的结果（逐字）；模型已回到调用前…"}
```

**`preview` 是键而不是把动词的字段抬到顶层**：抬上去就会出现 `applied:true` 与"什么都没落地"
同框，而 §11.24 (a) 的三态（applied / clamped / noop）每一个本来就只有**一个**正向标记。
`dryRun` 是第四态自己的标记；`preview` 里那三个标记保持原义不变。

**回滚不是 Undo。** 落法原本记的是"应用 → 读 diff → 批量 undo，撤销栈是现成的"，2026-09-26
在语料上量过，**不成立**：撤销栈管的是设计器的 `UndoRedoCommand`，而写动词有相当一部分写在
命令之外（`UICreator` 是工厂不是命令；`status="u"`、`sfield`、`TBinding` 都直接落在模型上）。
判据是 `save → dry-run → save` 比 sha256，拿 15 个写动词量过，只有 4 个纯布局的逐字节回得去。
现在的回滚是三条：

1. 把**当前模型**渲染成一份临时包 `_tt_dry_<pid>_<n>.tzs`，写在**源包旁边**
   （TzpManager 只在工作区内加载，所以临时包必须在工作区里）；
2. 跑动词（`DryRun.Active` 为真时 `Save.Run` **不落盘**，但照样拼包、照样回摘要）；
3. 把会话从**那份临时包**重读回来，句柄串不变（复用 (p) 的 `ReKey`）。

于是"干跑之后保存出来的东西 = 干跑之前保存出来的东西"是**构造上**成立的，判据在
`internal/dev/tzs/dryrun_test.go`（65 个包，另有一条非空转的对照：同样的参数真写一次，产出必须变）。

**要一起动的东西**（都踩过）：句柄的 `Path` 必须拨回源包（否则下一次读 `s.Path` 会 FileNotFound）；
句柄缓存的 `FormWriter` 必须换成快照那一份的（`Fns.Session.InstallLayout`，否则模型回来了而
`.4fd` 文本没回来）；取快照要写文件，所以它会因环境失败（只读目录 / 盘满），**那个异常必须和
动词体走同一个 catch** —— 第一版让它逃出调度器把守护进程打死了。

`_tt_dry_` 已在 `engine/make-manifest.sh` 与 Go 侧 `scratchPrefixes` 的排除集里：守护进程崩在
回滚中间会把它留在工作区，而它绝不该进语料基线。

**快照文件可能丢，所以回滚不许依赖它活着。** 2026-09-26 的全语料跑里出现过一次：65 个包里约 14 个的干跑，回滚时**打不开刚刚写下的那份
临时包**（同一进程、两条语句之前才写、`File.WriteAllBytes` 返回成功）。单独跑那条判据时
**65/65 干净**，重跑一次也复现不出来，而且没找到是谁删的。

查不出原因不等于可以把承诺压在它身上。渲染出来的字节本来就在手上，所以现在：快照在
回滚前**先看还在不在**，不在就用留底字节重写一份再打开（`Scope.EnsureScratch`），并且返回里多一个
`reverted.recovered: true` —— 这件事是**这台机器**的事实，不是调用方请求的事实，藏起来只会让
一次 flake 变成一个谜。连重写都失败才走失败分支（那时 `scratch` 与 `error` 都在返回里，
调用方仍能 `open` 那份包）。

写完之后**立刻**验一次 `File.Exists`，这个事实只在失败分支上报：它把"我们从没写成"与
"写成了、被别的东西吃了"分开。

#### (p) `reload`：从盘重读，句柄不变；`baseline` 是三态

`close` + `open` 的替代品，但多两样东西：

- **保住 validate 基线**。close 会 `Validate.Forget`，于是"这一轮改下来新增了什么"要重问一遍
  （大表单上一次 validate 十秒级）。`open` 时记下源包字节的 sha256，`reload` 时再算一次 ——
  **量出来**，不是假设。返回里的 `baseline` 因而是三态而不是布尔：`kept`（字节相同）/
  `dropped`（文件被换过，基线测的是另一份字节）/ `none`（这个句柄还没 validate 过）。
- **说出"盘上的包在你打开之后被换过"**（`fileChanged:true`）。那之前调用方手里的一切对照
  （基线增量、diff）都在拿两个不同的文档比，而这件事从前完全看不见。

句柄串不变：新铸的号**作废不用**（`Fns.Session.ReKey`），所以 `_nextHandle` 仍然单调，
"句柄串永不复用"((g)) 不被破坏。`discarded` 报出丢掉的撤销步数与布局是否有改动。

**失败方向**（两个都不能含糊）：文件读不出来 → **就地拒绝、句柄原样**；
close 之后 open 失败（包被换成了坏字节）→ 句柄真的没了，如实说 `E_NO_HANDLE` + "重新 open 即可"。

#### (q) `op` + `list_ops`：超时之后"到底写进去没有"

协议没有幂等键，从前的规则只有一句"请求一旦上线绝不重试"—— 那条治**重复**，没治**不知道**。
现在写动词与 `save` 都收一个可选的 `op`（调用方自己起的名字），引擎在**跑动词体之前**记一条
`pending`，跑完改成 `ok` / `error`。`list_ops` 按 `op` 精确匹配（新的在前）读回来，于是超时之后
有三个机械答案：

| 日志里 | 含义 | 调用方 |
|---|---|---|
| 没有这条 | 请求从未到达 | 重发安全 |
| `pending` | 引擎看见了，还在做（或进程已经没了） | **别盲目重发** |
| `ok` / `error` | 已经结束，摘要/错误码在里面 | 重发安全 |

**日志只记本进程，不落盘**（它的答案只在守护进程活着时有意义；落盘会把运维状态塞进客户工作区
或临时目录，两处都不该由引擎自己选），所以返回里带 `daemonStartedAt` —— 换个守护进程问同一个 op
必须能自己看出来。上限 500 条，挤掉的旧条目计数并报（`dropped`，`seq` 会跳号）。

**`SpecFn.Traced` 是这条的承重墙。** `list_ops` 自己有一个叫 `op` 的**过滤器**，
而调度器第一版按"参数里有没有 op"来认写动词 —— 那次查询于是把自己记成了写操作（实测）。
现在"哪些动词收这两个开关"是**声明**（`Manifest.Traced` 挂参数、`SpecFn.Traced` 是标志），
不是按名字猜；判据一分为二：manifest 那一半在 `dryrun_test.go`，行为那一半（读动词带着 `op`
不长日志）在 `oplog_test.go`。
