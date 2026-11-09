# CustomException —— 两个自定义异常类

全树最小的程序集（RootNamespace `CustomException`，csproj 无项目引用）：两个从 Exception 派生的标准
异常类（含 format/innerException 全套构造重载，纯样板无逻辑），供 SpecDesignerCommon 等引用
（如 SettingManager.cs:553 抛 SettingFileNotFoundException）。

## 目录地图

| 子目录 | .cs | 是什么 |
|---|---|---|
| SpecDesigner/CustomException/ | 1 | `SettingFileNotFoundException`（:6，命名空间 `SpecDesigner.CustomException`）—— 找不到设置文件时抛出 |
| SpecDesignerCustomException/ | 1 | `FileFormatErrorException`（:6，命名空间 `SpecDesignerCustomException`，连写无点）—— 文件格式错误时抛出 |
| Properties/ | 1 | AssemblyInfo.cs |

## 值得知道

- 两个子目录是**两个平级命名空间的镜像**：`SpecDesigner.CustomException`（带点分目录）与
  `SpecDesignerCustomException`（连写）风格不一致 —— 命名空间与目录不匹配的反编译/工程整理遗留，
  搜索时两个名字都要试。

## 细节去哪

- 上层索引：[../README.md](../README.md)
