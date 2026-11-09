# SpecDesigner.FormEditor/Helpers —— 表单编辑器辅助层

控件添加漂浮层、tab 序号服务、通用装饰器/视觉树工具与少量状态/图片转换器。单层平铺（15 个 .cs）。

## 目录地图

| 分组 | 代表 |
|---|---|
| 核心工具 | `AddWidgetAdornerHelper`（public static，AddWidgetAdornerHelper.cs:18，Show(...) 各重载在画布上预览待放控件）、`ComponentTabIndexService`（IDisposable，:12，299 行，tabIndex 重排）、VisualComponentsAdorner（:9，选中控件装饰）、VisualTreeHelperEx（:11，视觉树查找扩展）、ComponentNotFoundException（:6，internal） |
| 转换器 | ActionStatusImageConverter（:9）、ButtonEditImageConverter（:12）、IconImageControlConverter（:12）、CantDelOpacityConverter（:8）、IsCitedVisibilityConverter（:11）、DateTimeEditVisiblityConverter（:10，拼写 Visiblity 即原样）、CheckedValueConverter（:8）、LocalStringsConverter（:10）、RadioGroupItemConverter（:12）、RequiredStyleToBooleanConverter（:9） |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| ComponentTabIndexService.cs:81 | `SetAsCurrent`（:80 起）：把 _tabIndexedList 并回 _sourceList 后按目标元素位置切分两段并 ArrangeTabIndex()；随后 SetAsFirst/SetAsNext（:92 起）提供置首/顺次调整 —— 全部围绕 XmlElement 的 tabIndex 属性重排，是表单 tab 序交互的唯一实现 |

## 值得知道

- `AddWidgetAdornerHelper.Show` 是"把控件放上画布"的**统一出口**：DBStructure/UICreator.cs:31-51 与
  WidgetBox.xaml.cs:82-105 都汇到这里（两处的热点见 [../README.md](../README.md) 与
  [../Views/README.md](../Views/README.md)）。
- 与 SpecEditor/Helpers 不同，本目录转换器直接实现 IValueConverter，未用 ConverterMarkupExtension 基类
  （两种写法的分工见 [../../SpecDesigner.SpecEditor/Helpers/README.md](../../SpecDesigner.SpecEditor/Helpers/README.md)）。

## 细节去哪

- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
