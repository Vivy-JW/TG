# SpecDesignerCommon —— 全设计器的共享核心库

WPF 类库（.NET 4.0，SpecDesignerCommon.csproj:10-12），被其余工程共同引用。规格文档模型、tzp/tzs 包的
解包打包、全局设置、Prism 事件总线这四类中枢全在**根目录散文件**里；其余职责分给子包
（视图模型、事件定义、撤销命令、静态工具、服务器连接、站点管理器、首选项、多语言）。

## 目录地图

| 子目录 / 根散文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 34 | 四大管理器：`SpecificationInfo`（2739 行规格文档中枢，SpecificationInfo.cs:22）、`SettingManager`（单例 `Get()` 在 SettingManager.cs:48）、`TzpManager`（单个 tzp 包实例，TzpManager.cs:18）、`PackageManager`（静态解包/打包，PackageManager.cs:15）；`PackageKey`（Program+PackType+Memo 三元键，PackageKey.cs:8）；`EventAggregatorManager`（全局 + 按 PackageKey 两级聚合器，EventAggregatorManager.cs:11,22）；其余是 TzpType/PakageType 枚举、搜索族接口与事件参数、ObjectBindings/SpecBindingInfo 等绑定模型、`Watermark`、`RelayCommand` |
| ViewModel/ | 76 | 设计器的文档对象模型层 → [README.md](./ViewModel/README.md) |
| Events/ | 65 | Prism 事件定义层（48 个事件类 + 17 个参数类）→ [README.md](./Events/README.md) |
| UndoRedo/ | 25 | 表单/组件的撤销命令库（操作对象统一是 ViewModel.XmlElement）→ [README.md](./UndoRedo/README.md) |
| UndoRedoCommands/ | 7 | 并行的"规格节点内容"撤销命令命名空间（SDSpecUndoRedoCommand、SpecCitedUndoRedoCommand 等；FormPosUndoRedoCommand 继承 UndoRedo/ 的 AbstractUndoRedoCommand） |
| Helpers/ | 20 | 静态工具层：组件工厂、表格列缓存、WPF 转换器 → [README.md](./Helpers/README.md) |
| Connection/ | 20 | T100 应用服务器连接层（Telnet/SSH、登录窗口、后台 4GL 程序调用口）→ [README.md](./Connection/README.md) |
| site/ | 31 | 站点管理器：加密 XML 维护 T100 服务器站点树（SiteManager.xaml.cs:26 主窗口）、`SiteFileHelper`（AES 加密，SiteFileHelper.cs:51）、`UpdateManager`（AppCast 更新检查，UpdateManager.cs:10）；下挂 Behaviors/5（拖放行为）、Converters/6、ViewModels/10、Views/2、themes/3 |
| SpecDesignerPreference/ | 12 | 首选项子系统：PreferenceManager.cs:10 + RecentFiles/Language/CommonUsedElement 等五个模型 + 两个窗口 |
| TblUpdate/ | 5 | .tbl 表结构在线更新：TblUpdateManager.cs:10（StartUpdate :27、DownloadFile :137，从 castUrl 下载） |
| Views/ | 4 | 通用窗口：AutoCloseDialog、DetailsMessageBox、LocalItemsSelectionWindow、ProgressBar |
| Exceptions/ | 4 | ComplexException、NameAlreadyInUseException、NotInCurrentWorkspaceException、VersionIncompatibleException（包版本门禁抛出，TzpManager.cs:401） |
| Bookmark/ | 2 | 书签接口 IBookmark.cs:7、IBookmarkMargin.cs:7，供编辑器书签边栏实现 |
| Logger/ | 1 | DSCLogger.cs:11：写 Workspace\log\，无配置回退 C:\TT\log\（DSCLogger.cs:16-22） |
| Extension/ | 1 | StringBuilderExtension |
| Properties/ | 1 | AssemblyInfo.cs |
| images/ · langs/ · preference/ · themes/ | 0 | 纯资源：36 个 png/ico（site_* 三态站点图标）、4 份语言包（en-us/zh-cn/zh-tw/Vi-Vn.xaml）、首选项图标、generic.xaml |

## 热点索引（引擎注释引用的本层文件）

子包内的热点在各自 README；本表只收根目录散文件（含 site/）。

| 位置 | 那里是什么 |
|---|---|
| SpecificationInfo.cs:148-149 | 载入 TSD 后向全局聚合器订阅 SaveSettingEvent→SaveSpecificationInfo、TzpFileClose |
| SpecificationInfo.cs:243 | `SaveSpecificationInfo`：SaveToTSD/SaveToForm 后起 TSD 与 Form 两个后台校验线程（:243-251） |
| SpecificationInfo.cs:260 | 把 TSDElement 写回包（`SaveSpecificationInfo(this.TSDElement.ToString())`） |
| SpecificationInfo.cs:374 | TSD 校验出错时组 DocumentErrorsEventArgs（ERROR 级、含行号）发布 DocumentErrorsEvent（:367-374） |
| SpecificationInfo.cs:1314 | `FindFieldSpecById` 内：移除与表单控件同名的旧 SpecFieldNode，找不到就 `SpecFieldNode.Create` 新建并 SetSpecNode 绑定（:1309-1327） |
| SpecificationInfo.cs:1648 | `Add(XmlElement,…)`：包成 FormSpecModel 入 FormSpeDictionary，重名抛 "Name is Exist"（:1650-1664） |
| SpecificationInfo.cs:1913 | `SaveToTSD()`：克隆 TSDElement，先删 tree/field/act/multi_lang/help_code 旧内容再按模型重建（:1911-1930） |
| SpecificationInfo.cs:2201 | `SaveToForm()`：克隆 FormElement、删 Form/DiagramLayout，`RebuildScreenRecord` 后重加 `FormNode.ToXML()`（:2199-2211） |
| SpecificationInfo.cs:2232 | `GetScreenRecord`：递归控件树，带 fieldId 的控件重新编号并 AddRecordField（:2233-2241） |
| SpecificationInfo.cs:2341 | `SetFieldLocalStringText`：在 fieldStrings 找/建 SpecFieldStringNode 写 Text（:2341-2353） |
| SpecificationInfo.cs:2354 | 写完发布 SpecPropertiesChangedEvent（:2354-2357） |
| SettingManager.cs:79 | `CheckUndoRedoManager(key)`：undoRedoManagerMap 是否含该 key（缺失时 ：69 的 GetUndoRedoManager 抛 "No UndoRedoManager"） |
| SettingManager.cs:441 | `Version` 属性：懒取入口程序集版本并缓存（:435-445） |
| SettingManager.cs:517 | `CloseFile(key)`：从 tzpMap/undoRedoManagerMap 移除并清 ActionDefaults（:517-526） |
| SettingManager.cs:531 | `SaveSetting(key)`：经全局聚合器 Publish SaveSettingEvent（:529-532） |
| SettingManager.cs:587 | LoadCommonData 按 specReferFiles 读共享文件（固定 12 个，:92-94），core-br.spec / mod-fd.spec / tiptop.4ad / subroutines.xml 等读入 Info_* 字段（:587-613） |
| SettingManager.cs:598 | 同上遍历的实现段（core-br/mod-fd 的具体读取） |
| PackageManager.cs:68 | `Unpacking(zipFilePath, tzpManager)`：按 TzpType 决定必需扩展名再逐条分发（:68-83） |
| PackageManager.cs:132 | 扩展名 switch：`.4gl` → `tzpManager.Load4glFile`（:132-133） |
| PackageManager.cs:194 | 解包完仍有必需文件缺失则抛 FileNotFoundException（:194-203） |
| PackageManager.cs:208 | `Packing`：ZipInputStream→ZipOutputStream（SetLevel(3)）重打包（:208-214） |
| PackageManager.cs:320 | 取主文件名决定写回引用包的 CiteTAP 还是自身 TAP（:320-329） |
| PackageManager.cs:589 | `SeekReleaseVersion`：读 zip 内 "ver" 条目首行作包版本（:589-604） |
| TzpManager.cs:150 | `Type` 属性按 .tzs/.tzc/.tzd/.tzr/.tzg/.tzt 映射 TzpType（:150-169） |
| TzpManager.cs:154 | 即该 switch 的取值行 `Path.GetExtension(this.ZipFile)` |
| TzpManager.cs:225 | 构造尾部订阅 TzpFileClose（构造主体：CheckReleaseVersion→InCurrentWorkspace→Unpacking→LoadProgramInfomation，:216-226） |
| TzpManager.cs:229 | `CreateKey`：参数非法抛 NullReferenceException，否则 new PackageKey（:229-236） |
| TzpManager.cs:378 | `Load4glFile`：只置 _readingFlags["4gl"]=false，**不保存** 4gl 内容（:378-381） |
| TzpManager.cs:395 | `CheckReleaseVersion`：包版本与设计器 Major/Minor 不符抛 VersionIncompatibleException（:395-404） |
| PackageKey.cs:43 | `Equals` 按 Program+PackType+Memo 三元判等（:43-67） |
| site/SiteFileHelper.cs:51 | `Encrypt`：Rfc2898DeriveBytes（口令 "T100 SiteManager.SiteFile"、盐 "T100 SpecDesigner"）派生 AES 密钥加密站点文件（:51-60） |

## 细节去哪

- 子包地图：[./ViewModel/README.md](./ViewModel/README.md) · [./Events/README.md](./Events/README.md) · [./UndoRedo/README.md](./UndoRedo/README.md) · [./Helpers/README.md](./Helpers/README.md) · [./Connection/README.md](./Connection/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
