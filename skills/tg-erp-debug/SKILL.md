---
name: tt-erp-debug
description: 当需要对T100程序进行错误排查和DEBUG时，触发此技能
---

# TDebug T100 作业命令行调试

## 前提和原则

在处理BUG时，通过阅读ERP源代码，以及借助SQL工具可以解决大部分的问题，遇到困难复杂，必须通过还原程序运行现场时才需要使用DEBUG工具，接口类只需用户提供ID就可以DEBUG，带画面的程序可能需要协助用户点击来捕获提前设置的断点

## DEBUG作业的基本流程

> 参数不确定就 `tt debug <命令> --help`；环境/企业取值用 `--at-list`。下面的 `<ref>` 是会话地址（如 `主机109#1`），从 `debug start` 输出或 `tt session ls` 里抄。

1. 起服务（已在跑会提示，不会起第二个）：

```bash
tt serve
tt debug status
```

2. 连环境、起作业、等入口停站：

```bash
tt debug start <作业编号> --at <环境>@<企业> --name <别名> --timeout 120
tt session ls
```

记下这条会话的 `ref`。带画面的作业，用户要先打开 GDC。

3. 在用户操作前下断点：

```bash
tt debug bp add <函数名> --session <ref>
tt debug bp list --session <ref>
```

行号/文件行也可以：`tt debug bp add <文件>.4gl:<行>`。

4. 放行，等用户操作触发断点：

```bash
tt debug exec "continue" --session <ref> --wait 300
```

如果返回“还在跑”，让用户操作 GDC，再等：

```bash
tt debug wait --for stopped --session <ref> --timeout 300
```

5. 停住后看现场：

```bash
tt debug stop --session <ref>
tt debug exec "where" "info locals" --session <ref>
tt debug exec "print <变量>" --session <ref>
```

`debug stop` 只是读快照，不会让程序继续。

6. 单步/继续：

```bash
tt debug exec "next" --session <ref>
tt debug exec "step" --session <ref>
tt debug exec "continue" --session <ref> --wait 300
```

7. 结束：

```bash
tt debug quit --session <ref>     # 结束作业，会话回空闲
tt session rm --session <ref>     # 断连接、清资源
```

`tt serve --stop` 只在用户要求时执行。

带画面的程序停在 GDC 等输入时，不要发 `next`/`step`；先告诉用户点什么，等 `continue`/`wait` 返回。判断“在等用户”还是“在空转”：`tt debug why --session <ref>`。

## 如何DEBUG用户提供的日志ID

> 用户给的 ID 是 `rowid`（`tt debug wslogs` 列表里的那一条）。环境要问清楚；`--at-list` 可以看取值。如果用户给的是作业/服务/时间而不是 `rowid`，先用 `wslogs` 过滤出那一行。

1. 起服务，确认这条日志存在，先看请求/响应：

```bash
tt serve
tt debug wslogs --show <rowid> --at <环境>
```

要改入参时，先把请求原文存下来：

```bash
tt debug wslogs --show <rowid> --at <环境> --save-request req.json
```

按作业/服务/时间找 `rowid`：

```bash
tt debug wslogs --job <作业> --from <yyyy-mm-dd> --to <yyyy-mm-dd> --at <环境>
tt debug wslogs --service <服务名> --fail --at <环境>
```

2. 按这条日志重放，停在入口（企业必填，且必须是日志所在的那个环境）：

```bash
tt debug wsdebug <rowid> --at <环境>@<企业> --name <别名>
```

要改入参再重放，启动时直接给（`--set` 可重复，值为空即清空）：

```bash
tt debug wsdebug <rowid> --at <环境>@<企业> \
  --set digi-body.std_data.parameter.production_item_no=<新值>

# 或整份替换（用上面 --save-request 存下来的文件改完再送）
tt debug wsdebug <rowid> --at <环境>@<企业> --request-file req.json
```

3. 记下 `ref`，按基本流程下断点、放行：

```bash
tt session ls
tt debug bp add <函数名> --session <ref>
tt debug exec "continue" --session <ref> --wait 300
```

4. 程序直接退出/断点不命中：先看 `wsdebug` 输出里的 TOPENT 是不是纯数字（把据点码填进企业编号会跑不到业务逻辑）；不对就改：

```bash
tt debug topent <企业> --session <ref>
```

5. 结束同基本流程：

```bash
tt debug quit --session <ref>
tt session rm --session <ref>
```

## 原生 fgldb 参考(Genero BDL User Guide 6.00 节选)

> 来源:Genero BDL User Guide 6.00 的 `13_programming-tools/2520-fgldb.md`(fgldb 工具页)
> 与 `2586-debugger-commands.md`(Debugger commands 清单),内容已内联于本技能。
> `tt debug exec "<命令>"` 等价于在原生 `(fgldb)` 提示符下逐条输入,输出为原生文本。

### fgldb — interface program for remote debugging

The fgldb command line tool is an interface tool to attach to the FGL integrated
debugger remotely.

**Syntax 1: Debugging an application running on the computer**

```
fgldb -p process-id
```

- `process-id` is the process identifier of the `fglrun` process.

**Syntax 2: Debugging an app running on a mobile device**

```
fgldb -m host[:port]
```

- `host` is the hostname or IP address of the mobile device where the program executes.
- `port` is the TCP debug port number to connect to, default is 6400.

**Options**

| Option | Description |
| --- | --- |
| `-V` | Displays version information. |
| `-h` | Displays options for the tool. |
| `-p process-id` | Attach to an `fglrun` process running on the same computer, by using its process id. |
| `-m host[:port]` | Attach to a running mobile app to debug, using the mobile device hostname/IP and debug port. |

The fgldb tool can be used to:

- Debug an `fglrun` process currently running on the same computer, by using its process id.
- Debug an application on a mobile device, by using the mobile device IP address and its TCP debug port.

### Debugger commands — full list

All commands below are passed through with `tt debug exec "..."`.
As noted in the workflow above, `continue`/`run`/`until` (and similar commands that
resume program execution) block until the program stops again — add `--timeout 300`
(or larger) when running them.

| Command | Description |
| --- | --- |
| `backtrace` / `where` | Prints a summary of how your program reached the current state. |
| `break` | Defines a breakpoint to stop the program execution at a given line or function. |
| `call` | Calls a function in the program. |
| `clear` | Clears the breakpoint at a specified line or function. |
| `continue` | Continues the execution of the program after a breakpoint. |
| `delete` | Removes breakpoints that you have specified in your debugger session. |
| `detach` | Closes the TCP connection of a remote debug session. |
| `disable` | Disables the specified breakpoint. |
| `display` | Displays the specified expression's value each time program execution stops. |
| `down` | Moves down in the call stack. |
| `echo` | Prints the specified text as prompt. |
| `enable` | Enables breakpoints that have previously been disabled. |
| `finish` | Continues the execution of a program until the current function returns normally. |
| `frame` | Selects and prints a stack frame. |
| `help` | Provides information about debugger commands. |
| `ignore` | Defines the number of times a breakpoint must be ignored. |
| `info` | Describes the current state of your program (e.g. `info breakpoints`, `info locals`, `info sources`). |
| `list` | Prints source code lines of the program being executed. |
| `next` | Continues running the program by executing the next source line in the current stack frame, and then stops. |
| `output` | Prints only the value of the specified expression, suppressing any other output. |
| `print` | Displays the current value of the specified expression. |
| `ptype` | Prints the data type or structure of a variable. |
| `quit` | Terminates the debugger session. |
| `run` | Starts the program. |
| `set` | Configures your debugger session and changes program variable values (e.g. `set verbose on`). |
| `source` | Executes a file of debugger commands. |
| `signal` | Sends an interruption signal to the program. |
| `step` | Continues running the program by executing the next line of source code, and then stops. |
| `tbreak` | Sets a temporary breakpoint. |
| `tty` | Resets the default program input and output for future `run` commands. |
| `undisplay` | Cancels expressions to be displayed when the program execution stops. |
| `until` | Continues running the program until the specified location is reached. |
| `up` | Selects and prints the function that called this one, or the function specified by the frame number in the call stack. |
| `watch` | Sets a watchpoint for an expression (optionally with a condition, e.g. `watch i if i >= 3`). |
| `whatis` | Prints the data type of a variable. |