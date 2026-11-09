# T100 设计器 AI 集成 — 交接文档

> 目的：让 AI 能正确处理 `.tzs` 文件（T100 设计器的表单/代码包格式）。
> 做法：**不重写格式，而是在无界面环境下驱动设计器自己的代码**，再对最小差异做文本级修补。

---

## 快速开始

```bash
cd "D:/我的项目/T100设计器/.tzs-cli"
./build.sh                    # 编译全部 15 个单元（2 个库 + 13 个程序）→ $OUT（默认临时目录）
```

跑一遍端到端（产出一个真实 `.tzs`）：

```bash
/c/Users/18526/AppData/Local/Temp/dep_probe/AddField.exe \
    "D:\t100_wrok_dir\hengshuo\prd\aapp320(c).tzs" \
    "D:\t100_wrok_dir\hengshuo\prd\aapp320(c)_AIADD.tzs" \
    Table "" apca_t apcaent apca_t apcacomp
```

参数：`<in.tzs> <out.tzs> [容器类型] [落点路径] [表 列]...`（`-h` 看帮助）
容器类型 = `None | Grid | Group | ScrollGrid | Table | Tree`，空串走默认。
落点路径留空即用内置默认路径。

验收产出（全新进程、走设计器加载链）：

```bash
/c/Users/18526/AppData/Local/Temp/dep_probe/CheckOut.exe \
    "D:\t100_wrok_dir\hengshuo\prd\aapp320(c)_AIADD.tzs" apcaent
```

**更强的不动点验收**（不需要人工打开设计器，见 §核心结论 9）：

```bash
/c/Users/18526/AppData/Local/Temp/dep_probe/RoundTrip.exe \
    "D:\t100_wrok_dir\hengshuo\prd\aapp320(c)_AIADD.tzs"
```

改 / 删 / 包 / 增控件（**四条路都只加载一次**）：

```bash
# set  <元素路径> <节点类型>:<属性> <值>
Edit.exe in.tzs out.tzs set <path> field:req Y
# del  一个字段是控件 + 它的标签，一起传
Edit.exe in.tzs out.tzs del <path-to-Edit> <path-to-its-Label>
# wrap 把已有元素包进新容器（HBox/VBox 只接受容器，不接受控件）
Edit.exe in.tzs out.tzs wrap HBox <path-to-A> <path-to-B>
# add  在父容器下新增控件；Button 的 SpecNodeType 就是 ACTION
Edit.exe in.tzs out.tzs add <parent-path> Button:my_action
# act  为【已有】按钮补一个 <act>（面板上的「新增项目」）
Edit.exe in.tzs out.tzs act <path-to-Button>
# tab  不带路径 = 按文档顺序重排 1..N；带路径 = 把这些控件按给定顺序排到最前
Edit.exe in.tzs out.tzs tab
Edit.exe in.tzs out.tzs tab <path-to-A> <path-to-B> <path-to-C>
# info 只读，输出 JSON 到 stdout（UTF-8）。带路径则只输出那个子树
Edit.exe info in.tzs
Edit.exe info in.tzs <path>
# info --find 按控件代号定位，直接给出其它命令要的 path
Edit.exe info in.tzs --find <控件代号>
```

**`info` 是其余命令的入口**：`set`/`del`/`wrap`/`add`/`act`/`tab` 都要调用方先给一个
`elementPath`，而拿到它的唯一办法就是读 XML。`info` 把这棵树连同路径一起吐出来，链条才闭合。

**用户手里只有「控件代号」**（设计器 `字段属性` 面板里那个框，选中控件就能看到、可 Ctrl+C），
所以 `--find` 才是实际入口。**唯一性是设计器强制的**，不是巧合：

```
ComponentFactory.GetNewName      唯一命名出口，冲突自动分配 {默认名}{n}
                                 （新建 / 粘贴 / 改名全走它）
SpecificationInfo.Rename         空名 -> throw "'name' field is required"
                                 重名 -> throw Message_NameAlreadyExist（"名称重复"）
PropertyEditor 的 WatermarkTextBox  IsRequired + ValidatesOnExceptions，把上面两个 throw
                                 显示成红框
```

实测 91 个包 / 18,349 个布局元素 / **0 个无名**；唯一的重复是 `<Item>`（ComboBox 的**选项值**，
`name="1"`/`"2"`），不是组件。**所以一个代号 = 一个组件。**

```json
{"program":"aapp320","env":"c","codeTemplate":"P","specDictionarySize":116,
 "root":{"tag":"Form","name":"aapp320","path":"managedform/aapp320","specNodeType":"NONE",
   "children":[ ... ]},
 "elementCount":670}
```

每个节点：`tag`（控件类型）· `name` · `path`（**可直接喂给其它命令**）· `tabIndex`（只有参与
Tab 的 15 种类型有）· `table`/`column` · `fieldType` · `specNodeType`（决定 `set` 用哪个
`<nodeKind>`）· `specStatus`（见下）· `children`。

`--find` 的输出是**扁平的，不带子树**（`info <path>` 已经能把任意一条展开）：

```json
{"program":"capp113","env":"c","query":"b3_incjwf022","exact":true,"matchCount":1,
 "matches":[{"tag":"Edit","name":"b3_incjwf022","path":"ManagedForm/capp113/.../b3_incjwf022",
             "tabIndex":62,"fieldType":"NON_DATABASE","specNodeType":"FIELD"}]}
```

- **精确命中就不做子串扫描**，所以一个代号即使是另一个名字的前缀也只解析到它自己
- **精确没中才退化成子串**（`--find apca` 会列出 `l_apcasite` / `lbl_apcasite` / `l_apcasite_desc` 整套）
- **查不到不是错误**：`matchCount: 0`、退出码 0（参数是查询，不是位置）
- 另有一路 `actionMatches`（按 `id`）——`<act>` **不在** `FormSpeDictionary` 里，元素遍历看不到它，
  见 §15

**`specStatus` 的四态**（元素和 action 共用同一个映射）：

| 值 | 含义 |
|---|---|
| 缺省 | 加载自磁盘、未改动（唯一不值得写出来的状态） |
| `u` | 改动过，保存时会写 |
| `d` | 墓碑，`ToXml()` 照常输出 |
| `c` | **只在内存里，`.tsd` 中没有**——`ToXml()` 保存时会丢掉 |

`c` 是有用信息，不是噪声。实测三个原生文件的 `c` 全部落在设计器加载期为 Tree 造的脚手架上
（`ComponentFactory.cs:72-76` 的 `id`/`parentid`/`isnode`/`expanded`/`name`，以及一批 `Phantom`），
以及**没有 `<act>` 的按钮**——`FindActSpecById` 找不到同名 `<act>` 时会自己合成一个。
**这正是当初"凡加载期诞生的节点都提升"那个坑：现在它直接可见了。**

紧凑输出、不缩进：670 个元素的表单 **144 KB**，缩进版要 355 KB。

**Tab 顺序**（`Widget_TabIndex`，工具栏那个按钮）：`tabIndex` 是**整个表单一条全局平铺序列**，
只覆盖 `mod-fd.spec` 里 properties 含 `tabIndex` 的 **15 种控件**（Label/FFLabel/Image 等标签不参与）。
默认顺序 = 布局树的文档遍历顺序。实测语料 81 个原生包：49 个"文档顺序即 tab 顺序"、
25 个被手工调过、7 个值集合有缺口。

**「新增 button」和「新增项目」是两个独立事件**，设计器里也是分开的：

| | 入口 | 改哪儿 |
|---|---|---|
| 新增 button | 工具栏 `WidgetBox` | `.4fd` 加一个 `<Button>`；**`.tsd` 不动** |
| 新增项目 | `ActionDefaults` 面板（`FGL_AddItem`） | `.tsd` 加一个 `<act id="action_N">`；**`.4fd` 不动** |

两者**互不产生对方**。绑定只是"`<act id="X">` 与 `<Button name="X">` 同名"，没有任何地方记录。
`capp113(c).tzs` 里两个纯态都在：`desel_same_type`（只有 button）、`output`（只有 act）。

**注意**：本工具的 `add` 是两者合一——建 Button 时把 act 也提升落盘（否则按钮点了没动作）。
设计器的「新增 button」只建 Button。要对齐纯态就用 `add` + 手动 `act` 的组合。

**工作区里现成的样例**（都在 `D:\t100_wrok_dir\hengshuo\prd\`，都通过不动点检验）：

```
_ai_None / _ai_Grid / _ai_Group / _ai_ScrollGrid / _ai_Table / _ai_Tree   ← 加字段的 6 种容器落点
_ai_SET.tzs    把 l_apbbdocdt 设为必填（规格 + 布局同步改）
_ai_DEL.tzs    删掉 AI 加的那个字段对（布局 + 规格 + .bdx 一并处理）
_ai_WRAP.tzs   把 hbox_3 下的两个 Group 包进新建的 HBox
_ai_ACT.tzs    新增一个 Button（ACTION）—— .tsd 里多出一个 id=按钮名的 <act> 节点
```

---

## 现状

### 已实现并验证

| 组件 | 文件 | 验证结果 |
|---|---|---|
| 文本级元素索引 | `src/ElementIndex.cs` | 67 个文件：元素计数与 XML 解析器**完全一致**，跨度无畸形，名字路径零重复 |
| Record 段重建 | `src/RecordRebuilder.cs` | 48/66 逐字节复现设计器产出；**当前格式文件零差异** |
| `.4fd` 最小写入器 | `src/FormWriter.cs` | 空操作 **67/67 逐字节相同**；改一个属性差异**恰好 4 字节** |
| 容器重打包 | `src/TzsRepacker.cs` | 空操作 **67/67 逐字节相同**；未触碰条目字节与时间戳全保留 |
| **`add_field` 全链** | `test/AddField.cs` | 走设计器布局引擎；**6 种容器落点全部产出**，不动点检验 0 增 0 减 |
| **改 / 删 / 包 / 增控件** | `test/Edit.cs` | `set` 走 `SpecAttributeUndoRedoCommand`；`del` 留 `status="d"` 墓碑；`wrap` 走 `AddToContainerUndoRedoCommand`；`add` 走 `AddComponetsUndoRedoCommand`（Button→ACTION）。**四条路都只加载一次** |
| **读侧 `info` / `--find`** | `test/Edit.cs` | 整棵树带 `path` 输出 JSON；`--find <控件代号>` 解析成 `path`——10 个包 20 次查询，报的路径与从 `.4fd` 独立算出的**逐字相同（20/20）**；`--find` 出来的 path 直接喂 `set`，不动点 0 增 0 减 |
| **全语料批量验证** | `batch.sh` / `batch-write.sh` / `batch-edit.sh` / `batch-action.sh` | 读 **81/81**、写 **78/1/2**、改 **79/0/2**、增 **81/0/0**（**这是 81 个文件时的数**；2026-09-19 语料降到 67 个，这四个基线因此作废、待重采，见 §18 开头的语料变动注） |

### 八个测试程序

| 程序 | 验证什么 | 独立可跑 |
|---|---|---|
| `TestRebuild` | Record 段能否复现设计器产出 + `ElementIndex` 自检 | ✓ |
| `TestWriter` | `.4fd` 空操作逐字节相同 + `set_attr` 影响范围 | ✓ |
| `TestRepack` | 重打包空操作 + 单条目替换的局部性 | ✓ |
| `VerifyRepack` | 设计器自己的 `ZipInputStream` 读回 + `SeekReleaseVersion` | ✓ |
| `CheckOut` | 任意 `.tzs` 的全链加载验证 | ✓ |
| **`RoundTrip`** | **不动点：设计器重新生成的是不是还是它** | ✓ |
| `AddField` | 端到端 `add_field`（6 种容器） | ✓ |
| **`Edit`** | **端到端 `set` / `del` / `wrap` / `add`** | ✓ |

另有 `E2E.cs`（写入器与设计器模型的全链比对）、`MakeTestFile.cs`（早期手写版驱动，已被 `AddField` 取代但保留作对照）。

---

## 核心结论（详细版在 `SPEC.md`）

### 1. 格式本身

`.tzs` 是 ZIP，**且是 ZIP64**（SharpZipLib 对小文件也写 ZIP64：CD 的 32 位尺寸字段是
`0xFFFFFFFF` 占位，真值在 `0x0001` 扩展字段）。

按扩展名分类型：`.tzs`=Form、`.tzc`=Code、`.tzd`=CodeSpec、`.tzr`=ReportSpec、
`.tzt`=Report、`.tzg`=ReportCode、`.tzx`=差分、`.tzv`=简单表单、`.tzf`=独立函数。
每包有 `ver` 文本条目。必需条目见 `SPEC.md §1`。

内容：`.4fd` = Genero managed form XML（布局）；`.tsd` = T100 规格文档；
`.bdx` = 元素绑定；`.tsd` 里有正式 schema `mta/tsd.xsd`——**但它已和实现脱节，不能用于校验**。

### 2. 一个字段有三份表示

```
.tsd  <field name="apca_t.apcaent" table="apca_t" column="apcaent" .../>
.4fd  <RecordField name="apca_t.apcaent" fieldIdRef="67" .../>     ← 在 <Record> 下
.4fd  <Edit name="apca_t.apcaent" fieldId="67" .../>               ← 在布局树里
```

关联：`fieldId` ↔ `fieldIdRef`（**每次保存全量重编号，不是持久标识**）。
**寻址只能用 `name`**（名字路径，已验证零碰撞）。

### 3. 三条硬约束

- **程序集版本必须等于设计器版本**（`SettingManager.Version` 读 `GetEntryAssembly`）
- **加载要"无" UndoRedoManager，操作要"有"**（`SetInitGridX/Y` 在已注册时抛异常）
- **`ToXml()` 会静默丢弃内存 Status 为 `CREATE` 的节点**——只改 XML 的 `status` 属性无效，
  必须提升内存对象

### 4. 不要重写，要调用

设计器里已有完整实现，直接调：

| 需求 | 调用 |
|---|---|
| 加字段（含控件选择、命名、标签、位置、绑定） | `UICreator.CreateNoneContainerWidget` |
| 按容器类型加字段 | `UICreator.CreateGridContainer` / `Group` / `ScrollGrid` / `Table` / `Tree` |
| 语义节点展开 | `ComponentFactory.CreateEmptyComponent(key, SpecNodeType)` |
| 列元数据 | `TableColumnHelper.GetColumnInfo` / `GetColumnAttrInfo` |
| 生成 `.tsd` | `SpecificationInfo.SaveToTSD` |
| 生成 `.bdx` | `SpecificationInfo.SaveBinding` |
| 取元素的 XML（**含子节点**） | `XmlElement.ToXML()` —— **不要用 `Source`** |
| 完整的保存（会全量重写 `.4fd`） | `SpecificationInfo.SaveSpecificationForm` / `SaveToForm` |

**六个容器创建者全部已接通**（全是 `private static`，签名一致，反射只换方法名）：

```
None        CreateNoneContainerWidget   扁平同级（控件 + 标签，两列）
Grid        CreateGridContainer         新建 Grid，字段在它内部
Group       CreateGroupContainer        新建 Group → 内含新建 Grid → 内含字段
ScrollGrid  CreateScrollGridContainer   新建 ScrollGrid，字段带 repeat 系属性
Table       CreateTableContainer        新建 Table，字段文本落到 title
Tree        CreateTreeContainer         新建 Tree，自带 5 个构件
```

**除 `None` 外，返回值是"一个新的容器元素"，字段在它内部** —— 所以拼接目标是新容器本身，
且只有它自己的 `posX`/`posY` 需要摆放（内部坐标相对新容器，原样即可）。

**`add_field` 的正确形态**（`test/AddField.cs` 已实现）：

```
1. 加载设计器上下文，注册 UndoRedoManager
2. UICreator.Create<X>Container(...)  → 新容器（内含成对的 Label + 控件）
3. FormWriter 按 NextFreeRow 拼进 .4fd 文本（最小改动），取 ToXML() 拿含子节点的 XML
4. 重新加载编辑后的 .4fd（加载期间临时摘掉 UndoRedoManager）
5. 富化 + 提升 Status + AddSpecBinding
6. SaveToTSD + SaveBinding
7. TzsRepacker 只重写变更条目
```

### 5. 控件箱的权威清单（31 项）

`WidgetBox.xaml` 里数出来的，不是猜的。分四族：

- **语义节点（3）**：字段参考 `REFERENCE` / 多语言 `MULTILANG` / 查询按钮 `PROGREL`
  —— 这三项不是"控件类型"，是"用途"，展开后是特定的属性组合
- **容器（6）**：`Folder` `Grid` `Group` `ScrollGrid` `Table` `Tree`
- **从数据字典加（1）**：`AddWidgetByDataControl` → `UICreator.Create`
- **基础控件（20）**：`Edit` `ButtonEdit` `ComboBox` `CheckBox` `DateEdit` `TextEdit`
  `Button` `Label` `FFLabel` `FFImage` `Image` `ProgressBar` `RadioGroup` `Slider`
  `SpinEdit` `TimeEdit` `DateTimeEdit` `WebComponent` `Canvas` `HLine`

**可容纳拖放的容器**：`Grid` `HBox` `VBox` `Group` `Folder` `Page`

### 6. 用途 ≠ 控件类型

`SpecNodeType`（`FIELD`/`ACTION`/`FORMONLY`/`PROGREL`/`REFERENCE`/`MULTILANG`/`TREE`/`NONE`）
是**从属性组合推导出来的**，不是选的：

```
有 colName            → FIELD，没有 → FORMONLY
ButtonEdit + image="16/langmodify.png"   → MULTILANG
FFLabel    + style="reference"           → REFERENCE
Button     + style="button_qrystr"       → PROGREL
```

**所以对 AI 暴露的应该是"用途"，展开交给设计器。**

### 7. 取 XML 用 `ToXML()`，不要用 `Source`

`XmlElement.Source` 是私有属性，构造时 `RemoveNodes()` 过，**永远不带子节点** —— 容器模式
必须用 `ToXML()`。而且 `ToXML()` 才是权威序列化：`SaveToForm()` 末行就是
`xelement.Add(this.FormNode.ToXML())`，它按设计器自己的规则过滤属性，产出与文件里
既有元素的属性集**一致**（实测：既有 `<Edit>` 同样没有 `justify`/`picture`/`fontPitch`/`style`）。

### 8. 提升规则：只有 `colName` 非空的元素才进 `.tsd`

`ToXml()` 对 `CREATE` 状态返回 null，所以新增元素要提升 Status 才能进 `.tsd`。
但**"凡加载期诞生的都提升"是错的**：`Tree` 自带的五个构件（`id`/`parentid`/`isnode`/
`expanded`，以及名为 `name` 的 `Edit`）加载期同样是 `CREATE`，而语料里 37 个含 `Tree`
构件的包中，**设计器自建**的（`src="c"` 且 `status` 非空）相关节点数为 **0**。

规则直接用设计器自己的谓词（`SpecNodeTransform.TransformFieldType`，:90）：

```csharp
string col = formElement.GetAttribute("colName");
if (!string.IsNullOrEmpty(col))  fieldType = "TABLE_COLUMN" / "COLUMN_LIKE";
else                             fieldType = "NON_DATABASE";
```

`Tree` 修正前提升 6 个、修正后 1 个。

### 9. 不动点检验：不需要人工开设计器的验收手段

问法是：**把产出重新加载，让设计器自己 `SaveToForm`/`SaveToTSD`/`SaveBinding` 重生成，
写出来的是不是还是它？**（`test/RoundTrip.cs`）

6 个单字段产出 + 3 个三字段产出，实测全部：

```
.tsd / .bdx : 逐字节相同（规范化后）
.tsd 规格节点: 0 增 0 减
.4fd 路径    : 0 增 0 减
```

`.4fd` 的**字符数**会变，但那是设计器自己的重序列化 —— **未改动的原生文件同样如此**
（`aapt110(c).tzs`：333301 → 327136），所以判据是**路径集合**而不是字符数。
先用一份原生文件跑出 `0 增 0 减`，确认判据可用，再拿它验产出。

### 10. 全语料批量验证：读 86/86、写 77/80

两个驱动脚本，把"单表单验证"升级成"全语料验证"：

```
./batch.sh              读侧：89/89 干净（能否加载 + 0 增 0 减 + 报属性值偏差）
./batch-write.sh        写侧：77/80 干净（每包加一个真字段后是否仍是不动点）
./batch-edit.sh         改侧：78/80 干净（每包翻一个字段的 can_edit 后是否仍是不动点）
./batch-action.sh       增侧：81/81 干净（每包加一个 Button/ACTION 后是否仍是不动点）
```

**判据必须相对基线**：原生设计器文件里有 3 个（`capt111`/`cpmp530`/`cpmq001`）本来就与
自己的模型差 2–3 个 `posX`（detail 容器的 `posX: 2 → 0`）。所以写/改侧的门槛是
**"产出不劣于源文件"**，不是绝对 0——驱动会先对源文件跑一遍量出基线。硬卡绝对 0 会产生
3 个假警报，把真问题淹没。

语料分布决定了哪些分支真被跑到：`code_template` F=61/P=19/Q=4/W=2；
容器 Grid 924 / Page 453 / Group 315 / Table 239 / Tree 68 / **ScrollGrid 仅 1**；
布局元素数 10 ~ 1297。

- **`ScrollGrid` 全语料只有 1 处** —— 那个分支实现了但实质上没被验证过，别当它是验证过的。
- **工作区是每模块独立的**（各带自己的 `mta/`），`TzpManager` 拒绝工作区外的包。
  判定规则：最近一个含 `mta/` 的祖先目录，程序读 `TZSCLI_WS`。
- **`code_template` 这条轴无害**：P/R 与其余分支逐属性相同，只差 `can_edit` 一个默认值，
  且随后会被列元数据覆盖。

### 11. 无界面会撞上设计器的模态框（新缺口，已加看门狗）

`SpecificationInfo..ctor` **无条件**构造 `DatabaseSourceViewModel`，它在表查不到时弹
`DesignerMessageBox`。无界面没有消息泵 → **挂死，stdout 什么都不说**。

`AddField` 现在把两次加载都包在看门狗里（默认 90s，`TZSCLI_RELOAD_TIMEOUT` 可调），
超时打印诊断并 `Environment.Exit(3)`。**看门狗在独立线程，加载必须留在 STA 主线程**
（`Application.Current` 线程亲和，跨线程 `FindResource` 会抛）。

触发条件：表单引用的表没登记在 `<工作区>/mta/tables.xml`。`FindTableColumns` **先查这份
索引**，查不到连 `.tbl` 都不看。实测 `xiyuan/tst` 那份缺 4 张表而 `.tbl` 其实都在
（索引过期，刷新基础资料即可）。

### 12. 改 / 删：都走设计器，都只要一次加载

**`Modify`**：直接构造并执行 `SpecAttributeUndoRedoCommand`。它一次做完三件事——写规格属性、
按 key 分派 `Transform*` 把值推到配对表单元素上（`req`→`IsRequired`、`can_edit`→`noEntry`、
`table`/`column`→`TransformTableColumn`、…）、并 `OnPropertyChanged` 把 `Status` 置为 `u`。
**所以不要自己维护属性映射表。** 落盘仍是最小改动：命令前后做属性快照，只写回动了的那些。

**`Delete`**：`ClearBinding()`（先解绑，`SaveBinding` 不过滤）→ `SpecificationInfo.Remove(name)`
→ `FormWriter.RemoveNode(path)`。`Remove` 内部 `CloneModelToDeleted` → `Remove(node, oldName)`
会造一个**只剩名字**的同类节点、置 `Status=DELETE`，塞进 `this.fields`/`help_codes`/…。
墓碑能出现在 `.tsd` 里的原因：**`ToXml()` 只对 `CREATE` 返回 null，`DELETE` 照常输出**。

**与"增"最大的区别：只需要一次加载。** "增"要第二次，只是因为设计器得为它从没见过的元素
铸造规格节点；改和删都作用于模型已经认识的元素。

实测：删掉一个字段对（Edit + Label）后 `.4fd` 93813 → **92258**，正好等于加字段之前的长度。

### 13. 删除留下的墓碑长什么样

```xml
<field src="c" ver="1" column="" name="apca_t.apcaent" table="" attribute=""
       type="" req="" ... widget="" cite_std="N" status="d" />
```

其余属性全被剥空——这是 `Remove(node, oldName)` 造的形状，不是"把原节点改成 d"。
另外 `SaveToTSD` 产出**两份**文档：`TSDElement`（全量，写回包里的就是它）和
`TSD2Element`（差量覆盖层，只收 `status` 为 `u`/`d` 的节点）。

### 14. HBox / VBox 不是"新建容器"，是"把已有元素包起来"

**`UICreator.Create` 里没有它们，控件箱里也没有** —— 用户拖不出来。设计器造它们靠
`AddToContainerUndoRedoCommand(selection, parent, ComponentType.HBox)`（`ManagedForm` 的
HBox/VBox 布局命令），是**布局重构**，不是加字段。`Edit.cs` 的 `wrap` 对应它，同样只需一次加载。

**关键：它们只接受容器，一个控件都不接受。** 这个门槛来自 `core-br.spec` 的
`NodeInfo@acceptedMimes`（`ComponentFactory.AcceptMimes`，:1232）：

| 容器 | 接受控件 | 接受容器 |
|---|---|---|
| `Grid` / `Group` | ✓ 全部 | Grid 被额外规则挡掉容器子节点 |
| `Folder` | ✗ | **只有 `Page`** |
| `Page` | ✓ | ✓（含 HBox/VBox） |
| **`HBox` / `VBox`** | **✗ 一个都不接受** | **只接受容器** |
| `Table` / `Tree` | ✓ 部分 | ✗ |

所以"把字段包进 HBox"在设计上就是不可能的——`wrap HBox <两个 Edit>` 会被正确拒绝，
`wrap HBox <两个 Group>` 才行。想做之前先查这张表。

### 15. 新增 ACTION（按钮）：一次加载，且 `<act>` 是设计器自己造的

**`Button` 元素的 `SpecNodeType` 就是 `ACTION`**（`FormSpecModel.cs:467`，除非
`style="button_qrystr"` → PROGREL）。所以加一个按钮，设计器会自动为它铸出规格节点——
`FindActSpecById`（:1331）找不到同名 `<act>` 时自己造一个，**`id` = 按钮元素的 `name`**。

`<act>` 的规范属性（`SpecActionNode.Create`，:334）：
`src ver id cite_std gen_code type status`，`status` 初值是 `CREATE`，**必须提升**否则
`ToXml()` 丢弃。

**只需一次加载**（与"增字段"不同）：设计器的 `AddComponetsUndoRedoCommand.Execute()` =
`container.AddNode(el)` + `AddChildrenForSpec(el)`（递归 `Add(el, true)`）+ 列元数据富化——
**这正是 `add_field` 靠"第二次加载"间接达到的效果**，设计器一次做完。

`Edit.exe <in> <out> add <父路径> <控件类型>[:名字]`（默认 `Button`，默认名 `button_1`）。
被测过的父容器：`Grid` ✓、`Group` ✓、`Folder` ✗、`Page` ✗、`HBox` ✗。

### 16. 验收判据要能区分"值陈旧"

`RoundTrip` 原来的 `.4fd` 判据是规范化 XML 相等，而设计器每次保存都重排属性顺序，
所以它**永远报 DIFFERS**、**区分不出值陈旧**——`add_field` 的 `req`/`style` 不同步就是这么
溜过去的。现在补了**属性值级**比对（按名字路径逐属性比值）。

**但 `fieldId` 必须排除**：第一版没排除，结果 89 个包里 **20 个原生设计器文件**报不符，
差异全是 `fieldId`（`1 → 57` 之类）——那不是缺陷，是"每次保存全量重编号、非持久标识"。
排除后归 0。

**教训：新判据先对基线跑出 0，再拿去验产出。** 基线非零说明它在量实现细节；直接拿它判
"我的产出有问题"会得到一堆假警报，真正要抓的问题反而被淹没。

---

### 17. P0 探针：长驻进程 / 校验器 / 布局属性写路径

`test/Probe.cs`（`./build.sh Probe`，`Probe.exe <A.tzs> [B.tzs]`）。全程只动内存，不写工作区。
五个疑问，全部有答案。

**① 一个长驻进程能持多个包、交错操作吗 —— 能。**

```
[A] aapp320    加载 185 ms      [B] capp113  加载 101 ms
tzpMap 同时持有 2 个包
B 加载后重读 A：元素数 114（一致）
交错写：A=<upper>  B=<lower>     两包互不串扰
5 次内存操作共 0 ms
```

对照：**单独起一次进程约 1000 ms**。所以长驻服务把每次调用从 ~1 秒降到 ~0——这就是
JSON-RPC 进程形态的全部理由。（670 元素的大表单加载 821 ms，即加载本身与表单大小有关，
但那 1 秒里约 890 ms 是设计器初始化。）

**新发现的约束**：`UndoRedoManager` 还注册着的 key **不能再加载**——`SetInitGridY` 会抛
`illeagal call`。所以同一个 key 的"加载"和"操作"两个阶段不能交错，服务要按 key 管理生命周期。

`TzpManager.Current` 确实会被改成最后加载的那个包，但我们所有寻址都走 key，不受影响。

**② 能用设计器自己的校验器吗 —— 能，但有两个坑。**

```csharp
Call(si, "SaveToForm");            // ← 见坑 1
Call(si, "SaveToTSD");
Call(si, "InitTSDValidateWorker");
Call(si, "InitFormValidateWorker");
// 订阅 EventAggregatorManager.Global.GetEvent<DocumentErrorsEvent>() 收结果
```

- **坑 1：必须先 `SaveToForm()`/`SaveToTSD()`。** `FormElement`/`TSDElement` 是**构造时
  一次性赋值的快照**（私有 setter），那两个 worker 校验的是它们。不先重建，校验的就是
  "加载时的文件"，**任何编辑都不可见**。
- **坑 2：原生文件不是干净的。** `axmt500_wf(c).tzs` 未经修改就报 **11 条 WARNING**
  （`xmda009_desc - Column：'2' 不存在`）。所以校验判据**必须相对基线**，跟 §16 那个
  属性值判据同一个教训。

实测：改出一个重名 → 抓到 2 条 ERROR，消息就是设计器自己的"控件编号重复，请删除后新增并重新命名!"。
`Info_TsdXsd` 已加载（XSD 那一段有效）。**代价**：114 元素 1.6 s、670 元素 **10.4 s**，不便宜。

**③ 布局属性该走哪条路 —— 只能用 `XmlElement` 索引器，绝不能用命令类。**

| 路径 | 写已存在属性 | 写不存在属性 | 夹紧 |
|---|---|---|---|
| `el[attr] = v`（索引器） | ✓ 生效 | **✓ 拒绝**（`<null>`） | ✓ `gridWidth 1557→1` 被夹到 **1555** |
| `new FormAttributesUndoRedoCommand(...)` | ✓ | **✗ 照样写进去** | 部分 |

索引器是 UI 自己的写路径：白名单（属性不存在就拒绝）→ 同值短路 → `repeat`/`stepX`/`rowCount`
门禁 → `MinGrid*` 夹紧 → **最后才**调 `FormAttributesUndoRedoCommand`。直接调那个类会跳过前面全部。
这正是 `Edit.exe set` 污染 `.tsd` 的同一类错误，只是发生在布局侧。

**④ 写入能到达输出吗 —— 能。** 索引器写 `case=upper` → `SaveToForm()` 重生成 → 
`<ButtonEdit name="l_apcasite" case="upper">`。保存路径认这个改动。

**⑤ `ValidateForm` 这个偏好管什么 —— 管重叠检测，不是死的。**
`XmlElement.cs:2522`：`if (!PreferenceManager.Current.Settings.ValidateForm) return;` 直接
门控 `CheckOverlapping`，而它每次 `SetAttribute` 都跑。**我们所有工具一直设 `false`，
等于一直悄悄关着重叠检测。** 探针验证 `true` 时无界面可正常运行。

**这一轮的自我更正**（都是我先前的错误结论）：

| 我说过 | 实际 |
|---|---|
| "`ValidateForm` 没有任何地方读它" | **错**。`XmlElement.cs:2522` 在读。我当时用了 `\| head -10`，输出被截断就下了结论 |
| "`RoundTrip` 已经在跑校验只是没收结果" | **错**。它调 `SaveToForm`/`SaveToTSD` 但从不调那两个 worker，**从来没校验过** |
| "命令类 25 个" | **32 个**——漏了 `UndoRedoCommands/` 目录 |
| "`CreateEmptyComponentForBody` 是 `internal static`" | 是 **`public static`** |

另外探针自己出了 4 次错（同值写入、挑了 `MinGridWidth=1` 的元素、把 Form 自己的名字当重名对象、
提取时先撞上 `<RecordField>`）。**每次都是我的测试写错，不是产品有问题**——这个比例值得记住。

### 18. W3 门：函数全部实现之后的收官验收

三个部分。**没有一个判据是绝对阈值**——全部相对基线，这是本轮反复救命的规则。

**(a) 四个批脚本 vs 固定基线**（`batch*.sh`，81 个固定文件）

| 脚本 | 驱动 | 构造 `SpecificationInfo` | 结果 |
|---|---|---|---|
| `batch.sh` | RoundTrip (`none`) | **否** | **逐字节相同**（81/81 clean） |
| `batch-write` | AddField (`link`) | 是 | 78/1/2 一致；唯一差异是 `bsft001` 那行的**自由文本** |
| `batch-edit` | Edit (`link`) | 是 | **79/0/2 → 78/1/2** |
| `batch-action` | Edit (`link`) | 是 | **81/0/0 → 80/1/0** |

**四行差异全是同一个文件：`xiyuan/tst/bsft001_wf(c).tzs`。** 会复现的驱动全都构造
`SpecificationInfo`；唯一不构造的那个逐字节相同。

**那不是回归，是"有人在不在机器前"。** `DatabaseSourceViewModel.cs:41-82`：遍历表单引用的每张活表，
`TableColumnHelper.FindTableColumns` 抛异常就把表名累积，最后**无条件** `DesignerMessageBox.Show`。
无界面下没有消息泵 → `Show` **阻塞 STA 线程**。四条证据：

1. `xiyuan/tst/mta/tables.xml`（2025-08-14，我们从未写它）缺 `sfamwf_t`/`sfanwf_t`/`wfapwf_t`，
   弹窗里正是这几张表。
2. 设计器自己的日志 `xiyuan/tst/log/errlog_20260919.log` 里，**同一条缺失记录从 16:00 记到 21:53**，
   跨过基线采集时段（19:05–19:26）——条件从头到尾没变。
3. 把看守放宽到 110 秒，`Edit.exe` 在这个文件上**挂满 112 秒才被杀**（`rc=3`）。弹窗从不自行消失。
   所以基线那两个 `ok`（824 / 825 节点）**只能**来自 45 秒内有人点掉弹窗。
4. 不构造 `SpecificationInfo` 的 `RoundTrip` 驱动**逐字节相同**。

**没有点掉时**：`open` 会一直阻塞（`abmm200_wf(c).tzs` 那次观察到的 ~9 分钟是另一回事，见下），
所以**全语料扫描里 `bsft001` 是通的，而那是人手点的结果**。这是无人值守运行的已知限制，不是 W3 的缺陷。

**(b) `gate-w3.py`——出货路径 + 全语料**

- **段 A 9/9**：六个请求经**真正的 `tzs-cli`**（自动 spawn 命名管道守护进程）跑通，**stdout 与 stderr
  同时捕获**。这正是当初挂死的那种组合（守护进程继承调用方的捕获管道），W3-E 修好后第一次有门证明。
  冷开 1.54 s → 热调用 **0.13 s**；`stop` 0.06 s。`gate-w2.py` 里"被真实缺陷堵住"那句注释据此退休。
- **段 B 81/81 文件、21/21 断言**：每个文件一个 `tzs-server --stdio` 进程，`open` + 全读函数 +
  `nudge` 真写 + `validate` + `save` + RoundTrip。
  > **更正（2026-09-20，把门重写成 Go 测试时发现）：这一段的 B9「写后 validate 无新增错误」
  > 是恒真的，它永远不会失败。** `Fns/Validate.cs:139` 的基线是**按句柄懒捕获**的
  > （首调即自己的基线），而句柄是进程内的；段 B 的两个载荷各在一个 `--stdio` 进程里，
  > 所以第二个载荷里那次 `validate` 必然是"首调"，`newErrors` 构造上就是空。
  > **段 B 真正有信息量的判据是 B8（`nudge` 产生了真实 delta）、B11（RoundTrip 基线相对
  > 不动点）和 B-neg（被拒的写逐字节不变）**——那三条站得住，整体结论不变。
  > 要让它有意义，必须在**同一个句柄**上先量一次基线、再做操作、再量一次。
  > 新的 Go 语料测试就是这么做的（`TestCorpusWritePath` 等每次调两次 validate）。**B11 相对基线**：语料里 3 个文件原始就报 `stale`
  （`cpmp530` 报 2），要求 0 会冤枉它们——初版就是这么写的，被抓到了。
- **B-neg 逐字节**：对同一个包"不写直接存"与"被拒绝的写之后存"，两个文件 `sha256` 相同。
  「看起来没报错」在这个项目里藏过三次静默故障。
- 81 个文件 1110.3 s（13.7 s/文件）。**这是 `validate` 的成本，不是卡死**——每文件 2 次 `open` +
  2 次 `validate` + 2 次 RoundTrip，而 `validate` 在大表单上是 8–12 s（W2 测过 670 元素 10.4 s）。
  81 × ~13.7 ≈ 1134 s，与实测吻合。（我先曾把一段 9 分钟的停顿归因于 `abmm200_wf` 卡死并推测
  "PID 变了所以进程被换掉"——**两处都错**：该文件实测 pristine RoundTrip 1 s、`open` 1–2 s，
  而且每个 `srv()` 本来就新起进程，160 多次里 PID 当然会变。）

**(c) `gate-w3-fns.py`——34 个写函数全驱动**

`F1–F7` 全过：34/34 被驱动（0 个从未执行）、**125/125 个负向对照都以契约码拒绝**、
拒绝后 `sha256(pre) == sha256(mid)`、四个文件 RoundTrip **相对基线**不动点、`newErrors = 0`。
`F8` **FAIL**——见下面的缺陷 2。

**门挖出的缺陷**（这才是门的价值，不是绿灯）：

| # | 缺陷 | 状态 |
|---|---|---|
| 1 | **`list_tables` / `list_columns` 完全不可达**：声明 `NeedsHandle=false`，函数体却解引用 `s.Tzp` → 4/4 文件 `E_INTERNAL NullReferenceException`；而 `Manifest.Check` 又因 `handle` 不在 `Params` 里拒绝传入。**没有任何调用方式能成功** | **已修**（`Read.cs` 5 处改为 `s == null ? null : s.Tzp`，即 `PageTab.BaseData:526` 已有的写法）。修后实测 `list_tables` → 3886 张表、`list_columns apca_t` → 156 列 |
| 2 | **`copy_component` 无用，已摘除。** 直接实测：它**不复制**——同容器下节点数 114→114、元素只从第一个挪到最后一个，是设计器 cut+paste 的重排。根源在 `TargetContainer`（`Semantic.cs:1102`）取的是**源容器自己**的相对路径，于是只有两格：同一张表单 → 必然解析回同一个容器（纯重排）；另一张表单 → 路径不存在 → 回落到 `<Form>` → `CanExecutePaste` 一律拒。**没有任何参数能换容器。** 四个用途全不成立：复制（从不复制）· 重排（`move {to:"last"}` 更清楚）· 换容器（不可达）· 跨表单搬运（被拒，且清空源表单） | **已摘除**（49→48 函数）。它还会让设计器的 NRE（`GetActLocalStringText`，act 型 Button）逃成 `E_INTERNAL`，随函数一起消失。全表单内复制**是成功的**——早期结论"没有可达的成功路径"只试了跨表单，是错的 |
| 3 | **`set_items` 在列绑定元素上过不了一次重载——查清了，是设计器自己的行为，我们忠实复现。** `items` 对列绑定元素是**派生值**：换列时 `SpecFieldNode.cs:389-397` 用新列的 `col_attr` 覆盖它，加载时也会重算。设计器**自己的** items 编辑器同样如此——属性可见性按 **ComboBox 控件**分（`SpecPropertyEditor.xaml.cs:272`，九个字典里只有它列了 `items`），不问是否列绑定。所以不是我们的缺陷。但「写成功、值蒸发」是静默失败，正是这个项目反复挖到的那一类：`verify` **也看不见**（它比的是内存模型与磁盘文件，不是与一次重载） | **已加告知**：列绑定元素上 `set_items` 的返回多一个 `derivedFromColumn:true` 与说明，调用方不必靠猜 |
| 4 | **函数表有四份拷贝**：`Fns/{Session,Validate,Verify}.cs` 各带一个 `Descriptors` 数组，**全仓零引用**，且与 `Manifest.cs` 不一致。它们来自并行波次——注释写着"Manifest.cs 是别人的文件，先park在这里"——波次结束后没人并回去。**死的那份反而更准**：`Manifest` 的 `open` 漏了 `timeout`，而实现读它（`Session.cs:220`）且 `Manifest.Check` 拒绝未声明的参数，**等于这个参数从线上根本传不进去**；`close` 的 `Mutating` 死的写作 false 并附了论证，活的写作 true | **已修**：删掉三个数组（95 行），`timeout` 补进 `Manifest`，`close` 按论证改为 `Mutating=false`。实测 `open --timeout 30` 现在通得过 |

| 5 | **`FormWriter.Indent()` 不给元素自己那行加缩进**（`src/FormWriter.cs:255`），`AddNode` 靠"\r\n" + block + "\r\n" + 父缩进 拼接，于是新元素的标签落在第 0 列而它的子树缩进正常 | **已修**：调用处补上 `new string(' ', indent)`（递归里子节点本来就自己补了，所以只改调用方一行）。实测产出的 `.4fd` 里唯一不缩进的行只剩 XML 声明与根元素，且 RoundTrip 仍 `0/0/0/0` |

**顺带查清**：`reconcile.py` 报 80 而语料是 81，是因为它按设计剥掉非 ASCII 再比路径，而
`aapp320(c) - <中文>-修改前.tzs` 与 `…-删除前.tzs` **只在中文字符上不同**，归一化后撞名。无害。

---

## 下一步

W0–W3 走完，函数全部实现并经门验收（§18）；此后又补了 `reparent`，现共 **49 个**。**门挖出五个缺陷，四个已处理**：
`list_tables`/`list_columns` 不可达→已修；`copy_component` 无用→已摘除；`Session/Validate/Verify` 三个死 `Descriptors` 表→已删并按论证解决了漂移；`FormWriter.Indent()` 不缩进→已修。
剩下的一个（`set_items`）查清是**设计器自身行为**，我们忠实复现，已在返回里明确告知。

按价值排序的待办：

1. **`bsft001_wf` 那条无人值守限制**（§18(a)）：包缺基础资料时设计器弹模态框，无消息泵即永久阻塞。
   长驻守护进程被一个坏包卡住，比单次进程严重得多——**值得给 `open` 一条能真正生效的终止路径**。
2. **重采四个批脚本的基线**：`xiyuan/tst` 被删后语料降到 67 个，而基线
   TSV 记录的 81 个里有 14 个已不存在——`git diff --stat -- '*.tsv'` 这个关卡目前是坏的。一次约 20 分钟。
3. **`FormWriter` 的改动需要一次真正的关卡**（§18 缺陷 5）：它的输出变了，而目前只验了单文件的
   RoundTrip 不动点。折进第 2 条一起做。
4. **旧程序未迁 `ref`**：`Edit.cs` / `AddField.cs` 仍是 `link` 模式。迁移后 15 个构建单元只剩两个
   副本，但**必须在关卡做**——四个批脚本是它们的回归网。
5. **`move` 的绝对序号**无法通过冻结的 manifest 表达（`to ∈ {first,prev,next,last}`）。
6. 原有的低优先级项不变：写侧输出编码、`add_field` 的规格/布局同步、`del` 的容器语义、
   CLI `apply` 批处理（实测批处理解决的是不存在的性能问题）。
