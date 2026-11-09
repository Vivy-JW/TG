# CodeEditor.FglAnalysis —— 4GL 词法 + 语法分析库

手写的 FGL（Genero/Informix 4GL，.4gl）分析库（RootNamespace `CodeEditor.FglAnalysis`）：FglReader
逐字符读取（FglReader.cs:11-14），FglScanner 状态机分词产出 FglToken（DISPLAY/DIALOG/INPUT 等
4GL 关键字，FglScanner.cs:23-32，内嵌 ScanState 枚举 :520），FglParser 带前瞻递归建 AST（:15-37）。
csproj 无 ProjectReference —— 纯解析库，供结构树与函数解析消费（如 CodeEditWindow/Helper 的
FglParserQuickHelper，CodeEditorManager.cs:1238）。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| 根散文件 | 9 | `FglScanner`（状态机分词，FglScanner.cs:7）、`FglParser`（:8）、FglReader（:6）、FglToken（:6）、FglTokenNode（:7）、TokenType（:6，枚举）、AST（:7）、FunctionAST（:6，继承 AST）、Nodes（:6）—— 手写语法树，不是 Roslyn |
| Properties/ | 3 | AssemblyInfo.cs、Resources.Designer.cs、Settings.Designer.cs（模板生成文件，含 resx/settings） |

## 热点索引（引擎注释引用的本程序集文件）

| 位置 | 那里是什么 |
|---|---|
| FglScanner.cs:350 | 分词状态机某分支的返回点：buffer 为空时返回 `MakeToken(TokenType.PEROID_TOKEN, ",")`，否则回退一字符走关键字识别（:336-358） |

## 细节去哪

- 消费方（编辑器、加点模型）：[../CodeEditWindow/README.md](../CodeEditWindow/README.md) · [../Infrastructure/README.md](../Infrastructure/README.md)
- 上层索引：[../README.md](../README.md)
