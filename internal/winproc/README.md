# internal/winproc — 起一个不随本进程死的子进程

只做三件事：**起一个不随本进程死的子进程、判它活着、杀掉它。**

平台差异全部封在这里：一个平台实现（Windows 走进程与管道原语），另有一个其它平台的空实现
（本项目的发行目标是 Windows）。

## 两处用途

| 用途 | 谁在用 |
|---|---|
| 把自己重新拉起来做后台常驻 | [../cli/debug/README.md](../cli/debug/README.md)（`tt serve`） |
| 托管 C# 引擎守护进程 | [../dev/tzs/README.md](../dev/tzs/README.md) |

**就这两处，所以它刻意很薄。** 两边的失败语义、状态文件、停止方式都不同（一个探 HTTP、
一个探命名管道），那些都属于各自的包 —— 共用的只有这三件纯进程原语。

## 判据

本包没有自己的测试文件；行为由上面两处使用者的用例覆盖。

## 细节去哪

- 后台常驻与单实例 → [../cli/debug/README.md](../cli/debug/README.md)
- 引擎守护进程的生命周期 → [../dev/tzs/README.md](../dev/tzs/README.md)
