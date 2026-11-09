# AutoUpdater —— 独立更新器 EXE

独立小 EXE（RootNamespace `AutoUpdater`，.NET 4.0）：主程序 T100Designer 退出后由它接管升级 —— 杀掉
T100Designer 进程、用 xcopy 把升级包覆盖到安装目录、再重启设计器。引用 SpecDesignerCommon
（Program.cs:5）仅为 DesignerMessageBox/资源文案。

## 目录地图

| 子目录 / 根文件 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 1 | `Program`（Program.cs:10，internal）：`Main` 要求 3 个参数 —— 新版本目录、PID、唤醒参数（:12-26）；流程 KillDesigner→CopyUpgradeFiles→WakeApplication→Exit |
| 根配置 | — | app.config（supportedRuntime v4.0）、app.manifest（UAC/兼容性清单） |
| Properties/ | 1 | AssemblyInfo.cs |

## 值得知道

- 更新对象即 T100Designer 主程序目录，**进程名硬编码 "T100Designer"**（Program.cs:102）；
  KillDesigner 杀进程后等待 2 秒（:97-110）。
- CopyFiles 起隐藏 xcopy `/R/Y/I`，并把其 stderr 输出当"更新完成/报错"提示（:70-94）；
  KillOtherUpdate 防止两个 AutoUpdater 并行（:121-136）。

## 细节去哪

- 上层索引：[../README.md](../README.md)
