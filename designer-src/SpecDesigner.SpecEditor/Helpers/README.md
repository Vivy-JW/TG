# SpecDesigner.SpecEditor/Helpers —— 转换器/行为工具箱

命名空间 `SpecDesigner.SpecEditor.Helpers`：43 个文件里约 35 个是 IValueConverter/IMultiValueConverter，
服务于规格树与属性面板的显示开关。单层平铺，无子目录。

## 主题分组

| 分组 | 代表 |
|---|---|
| 基类与工具 | `ConverterMarkupExtension`（`ConverterMarkupExtension<T> : MarkupExtension, IValueConverter`，转换器免资源注册的基类，ConverterMarkupExtension.cs:9）、`ExportTools`（:12）、TextBoxEnterBehavior（:11，Behavior<TextBox>）、Visual_ExtensionMethods（:8）、ActionStatus（:7） |
| 转换器（约 37 个） | ActionDisabledCheckedConverter（:9）、AggregateVisibilityConverter（:10）、CantDelTagReadonlyConverter（:7）、CiteEnabledConverter（:10）、ColumnAttributeConverter（:12）、ColumnTypeConverter（:12）、MatchSpecStatusConverter（:9）、ProgRelVisibleConverter（:9）、RepeatVisibilityConverter（:10）、SpecTreeAttVisibilityConverter（:10）、TableAssociationConverter/TableAssociationTextConverter、TitleAttributeVisibilityConverter、MutiComboValueConveter（:11）/MutiTextValueConveter（:11）等 —— 其中 9 个继承 ConverterMarkupExtension<T> |

## 搜索提示

- 类名里的拼写遗留是原样反编译的，搜索时别"顺手纠正"：`Conveter`、`Muti`、`Visbility` 都查得到，
  写对了反而查不到。

## 细节去哪

- 两种转换器写法的分工：继承 ConverterMarkupExtension<T>（免注册，本目录为主）与直接实现
  IValueConverter（FormEditor/Helpers 为主 → [../../SpecDesigner.FormEditor/Helpers/README.md](../../SpecDesigner.FormEditor/Helpers/README.md)）
- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
