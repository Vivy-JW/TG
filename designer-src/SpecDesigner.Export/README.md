# SpecDesigner.Export —— 画面规格导出 Word

规格导出程序集（RootNamespace `SpecDesigner.Export`）：以适配器模式把 FormEditor 的画面规格导出为
Word 文档。`template.docx` 是内嵌的 Word 模板（docx 骨架/占位样式）：导出时从 pack URI 释放到临时
目录，再由 Xceed DocX 填充。ProjectReference 引 SpecDesigner.FormEditor —— 导出的对象是画面规格
（DocxExportAdapter.cs 顶部 using）。

## 目录地图

| 子目录 / 根文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 1 | `ExportSpecification`（:7，仅持有 IExportAdapter 并转发 Export(templateFilePath)，:11-18） |
| Adapters/ | 2 | IExportAdapter（:6，接口）、`DocxExportAdapter`（:30，构造从 FormEditorMainWindow 抓 ManagedForm :32-37） |
| Properties/ | 2 | AssemblyInfo.cs、Strings.Designer.cs（导出提示文案，含 resx） |
| template.docx | — | 全树唯一非代码资源：Word 导出模板 |

## 导出流程（DocxExportAdapter.Export，:40-88）

SaveFileDialog 选目标 → 释放 template.docx 到 templateFolder（无则写出，:53-79）→ DocX.Load 后依次
RenderDocumentInfo / RenderProgramInfo / RenderSnapshot / RenderFields / RenderAction → Process.Start
打开文档（:80-88）。

## 细节去哪

- 被导出的画面模型：[../SpecDesigner.FormEditor/README.md](../SpecDesigner.FormEditor/README.md)
- 上层索引：[../README.md](../README.md)
