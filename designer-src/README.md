# designer-src —— T100 设计器的反编译源码（**只读参考**）

这不是本项目的代码，也不是我们要构建的东西。它是 T100 设计器（厂商标识 DSC）的反编译产物，
**入库只为一件事**：让引擎源码注释里那些 `file:line` 在仓库里就能对上。

**只看不改。** 这里没有构建入口（`solution.sln` / `.csproj` 是反编译带出来的，不是我们的构建链），
任何"顺手改这里一下"都等于改了参考基准 —— 那样引擎注释里的行号会静默指错。

## 程序集索引

16 个工程 = 16 个顶层目录（目录名 = 程序集名 = csproj 名）。想看某个目录内部，进它的 README；
实质性子包还有下一层 README，其余子目录由父级 README 的目录地图逐行覆盖（见下方"覆盖规则"）。

| 工程 | 是什么 |
|---|---|
| [T100Designer](./T100Designer/README.md) | 主 EXE：应用入口、主窗口、全局菜单命令（其 SpecDesigner/ 主 UI 子树另有一份 README） |
| [SpecDesignerCommon](./SpecDesignerCommon/README.md) | 共享核心库（全树最大）：规格文档模型、tzp 包管理、设置、事件总线、站点管理器（ViewModel / Events / UndoRedo / Helpers / Connection 五个子包另有 README） |
| [SpecDesigner.FormEditor](./SpecDesigner.FormEditor/README.md) | 表单所见即所得编辑器：画布、控件工具箱、由库表反向生成控件（Views / Helpers 另有 README） |
| [SpecDesigner.SpecEditor](./SpecDesigner.SpecEditor/README.md) | 规格树编辑器与节点属性面板（Helpers / Views 另有 README） |
| [CodeEditWindow](./CodeEditWindow/README.md) | 4GL 代码编辑窗口：分区编辑、加点管理、diff 对比、语法高亮（Helper / View 另有 README） |
| [Infrastructure](./Infrastructure/README.md) | 加点/分区的数据模型、Prism 事件与 XML 流式解析 |
| [CodeEditor.FglAnalysis](./CodeEditor.FglAnalysis/README.md) | 4GL 词法 + 语法分析（手写 scanner/parser/AST，无工程引用） |
| [SpecDesigner.Search](./SpecDesigner.Search/README.md) | 全局查找/替换面板 |
| [SpecDesigner.FormDataEditor](./SpecDesigner.FormDataEditor/README.md) | 画面数据 XML 的主从表格编辑器 |
| [SpecDesigner.Output](./SpecDesigner.Output/README.md) | 错误/警告输出面板 |
| [SpecDesigner.TableViewer](./SpecDesigner.TableViewer/README.md) | 数据表/列只读查看器 |
| [SpecDesigner.Export](./SpecDesigner.Export/README.md) | 画面规格导出 Word（内嵌 template.docx 模板） |
| [DifferenceEngine](./DifferenceEngine/README.md) | 行级 diff 算法库 |
| [UndoRedoFramework](./UndoRedoFramework/README.md) | 通用撤销/重做框架（双栈 + 命令分组） |
| [AutoUpdater](./AutoUpdater/README.md) | 独立更新器 EXE（杀主程序 → xcopy 升级包 → 重启） |
| [CustomException](./CustomException/README.md) | 两个自定义异常类 |

**热点索引按文件所在层落位**：引擎注释引用的 `file:line` 由文件所在的那份 README 收录 ——
散在程序集根目录的进程集 README，在子包里的进子包 README。查一条引用：先定位文件在哪个目录，
再进对应 README 的热点索引表。

## 覆盖规则（每个目录都能被索引到）

- **16 个程序集目录** → 上表各有一份 README.md。
- **12 个实质性子包** → 各有一份 README.md：
  `SpecDesignerCommon/{ViewModel, Events, UndoRedo, Helpers, Connection}`、
  `SpecDesigner.SpecEditor/{Helpers, Views}`、`SpecDesigner.FormEditor/{Views, Helpers}`、
  `CodeEditWindow/{Helper, View}`、`T100Designer/SpecDesigner`。
- **其余约 90 个目录**（`Properties/`、`images/`、`langs/`、`themes/`、`site/` 的叶子、`Petzold/` 等）
  各只有一个 AssemblyInfo.cs 或纯资源，由**父级 README 的目录地图表逐行覆盖**，不单开 README。

## 怎么核对

```bash
# 引擎文档与注释引用的全部 file:line 目标（设计器侧 50 个文件应能逐条在各 README 的
# 热点索引里找到；Edit/AddField/Probe/RoundTrip 等是引擎自己的测试程序，不在设计器侧）：
grep -rhoE "[A-Za-z0-9_.]+\.cs:[0-9]+" engine docs --include="*.md" --include="*.cs" | sed 's/:[0-9]*$//' | sort -u
```
