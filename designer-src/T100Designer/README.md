# T100Designer —— 设计器主 EXE

`<OutputType>WinExe</OutputType>`（T100Designer.csproj:7），整个设计器的启动程序；AssemblyInfo 自述
"Specification Designer & Code Editor"（版本 1.0.0.251，2013）。应用入口在
SpecDesigner/Main/SpecDesignerApp.xaml.cs:31（`SpecDesignerApp : Application, ISingleInstanceApp`），
单实例支持由 Microsoft/Shell 提供。主 UI 层在 SpecDesigner/ 子树（另有一份
[README.md](./SpecDesigner/README.md)）。

## 目录地图

| 子目录 / 根文件 | .cs | 是什么 |
|---|---|---|
| SpecDesigner/ | 23 | 主程序 UI 层（入口、主窗口、文档 ViewModel、全局菜单命令）→ [README.md](./SpecDesigner/README.md) |
| Microsoft/Shell/ | 4 | 从微软 SingleInstance 示例搬来的单实例基础设施：ISingleInstanceApp.cs:9（接口）、SingleInstance.cs（跨进程互斥，回调 SignalExternalCommandLineArgs :124）、NativeMethods、WM（Windows 消息常量）。全树唯一的第三方命名空间目录，无业务 |
| Properties/ | 1 | AssemblyInfo.cs |
| Themes/ | 0 | Generic.xaml：窗口外观样式（ShowInTaskbar=False、NoResize 等），不含控件模板；真正的资源合并来自各程序集（SpecDesignerApp.xaml:8-22） |
| images/ | 0 | 80 个文件：菜单/工具栏图标 + 11 个 .ico（tzp/tzd/tzg/tzr/tzs/tzt/tzx 等各规格扩展名的文件图标） |
| 根文件 | — | app.manifest（UAC/兼容性清单）、startup.png、csproj |

热点索引：本层散文件没有被引擎注释引用；引用落在 SpecDesigner/ 子树（MenuCommands.cs、
CodeTemplateSelector.xaml.cs），见 [./SpecDesigner/README.md](./SpecDesigner/README.md) 的热点索引。

## 细节去哪

- 主 UI 子树：[./SpecDesigner/README.md](./SpecDesigner/README.md)
- 上层索引：[../README.md](../README.md)
- 为什么入库、边界在哪：[../../README.md](../../README.md)
