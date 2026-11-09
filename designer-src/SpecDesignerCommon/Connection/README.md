# SpecDesignerCommon/Connection —— T100 应用服务器连接层

命名空间 `SpecDesignerCommon.Connection`：登录窗口（多套）、Telnet/SSH 两种网络实现、服务器日志窗口，
以及通过 `RunProgram` 调后台 4GL 程序的静态入口。设计器对 T100 后端的全部触达都汇聚在
`ConnectionManager` 一个类里。

## 主题分组（20 个 .cs）

| 分组 | 数量 | 代表 |
|---|---|---|
| 网络协议 | 3 | ITTNetwork（接口）、`Telnet`（Telnet.cs:13）、`SshNetwork`（SshNetwork.cs:11，IDisposable） |
| 管理/会话 | 2 | `ConnectionManager`（824 行静态门面，ConnectionManager.cs:18）、ConnectionInfo（:7） |
| 窗口 | 5 | LoginWindow、ServiceCloudLoginWindow、TOPSTDLoginWindow、CheckGDCWindow、ServerLog |
| 数据/命令 | 6 | DataReceivedEventArgs、MessageEventArgs、LogData、CheckOptionEnum、InputValidation、ServerLogCommands |
| Adorner/ | 3 | WaterMarkeAdorner、PasswordBoxWatermarkBehavior、AdornerExtensions（登录框水印） |
| Helper/ | 1 | PasswordHelper |

## 热点索引（引擎注释引用的本目录文件）

| 位置 | 那里是什么 |
|---|---|
| ConnectionManager.cs:805 | `GeneralFunction(ProgramKey, Component)` 内拼 `adzi261 -PRGNO {程序} -SPCVER {版本} -DSTCNO {控件} -DSTCON {容器} -SPCCKO {签核Y/N}` 命令行并 RunProgram 执行（:797-808） |

## 值得知道

- ConnectionManager 是后台 4GL 程序的统一调用口：`RunProgram(adzp201 …)`（ConnectionManager.cs:793）、
  `GeneralFunction(adzi261)`（:805）、`GeneralValueFunc(adzi262)`（:819）。
- CodeEditWindow 的保存链路也走这里（adzi520 登记保存 → [../../CodeEditWindow/Helper/README.md](../../CodeEditWindow/Helper/README.md)）。

## 细节去哪

- 上层索引：[../README.md](../README.md)；全树入口：[../../README.md](../../README.md)
