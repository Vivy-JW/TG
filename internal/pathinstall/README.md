# internal/pathinstall — 用户 PATH 的增删

把可执行文件所在目录加进 / 移出**用户** PATH。

**只动 `HKCU\Environment`，绝不碰系统 PATH，也不需要管理员。** 这条约束没有例外 ——
它决定的是"安装 tt 需不需要 IT 审批"。

两个入口共用这一份实现：Web 设置页的安装端点，与命令行的 `tt install path`。
字符串层面的 PATH 增删是纯函数（不触系统状态），因此可以单独测。

## 判据

```bash
go test ./internal/pathinstall -count=1
```

覆盖：真实折行的 PATH 解析、幂等（装两次不会重复）、不改写用户主目录那些展开式路径、
只写用户级键。

## 细节去哪

- 命令行的用法与选项 → [../cli/README.md](../cli/README.md)
- 设置页怎么调它 → [../web/README.md](../web/README.md)
