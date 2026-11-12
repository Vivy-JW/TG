# TG（TOP GP 开发 CLI）

面向鼎捷数智 TOP GP（Genero 4GL）ERP 的命令行工具集。

> 本项目在架构上参考了 [TT](https://github.com/luyaogod/TT)，面向 TOP GP 重新实现。
> 当前代码基线来自 TT 0.5.2，可执行文件名与命令前缀暂时仍是 `tt`；
> 等 GP 相关能力稳定后统一改名为 `tg`，那时才算 TG 自己的第一个基线版本。

---

## 当前状态

处于**打基础**阶段：

- ✅ 已导入 TT 0.5.2 源码基线（可编译、可运行）
- 🚧 TOP GP（Genero 4GL）适配进行中
- ⏳ T100 专有模块（数据字典、`.tzc`/`.tzs` 管线、引擎二进制、T100 技能包）待逐步替换后移除

## 从源码构建

需要 Go 1.26+；构建便携包时还需要 Node.js 与 pnpm（前端会被编译进二进制）。

```cmd
build_portable.bat
```

## 使用

命令与 TT 保持一致，当前可用的主要命令：

```cmd
tt serve              # 启动本地配置页（http://127.0.0.1:28670）
tt install path       # 把 tt 加入用户 PATH
tt install skills     # 把内置技能安装到指定目录
tt update check       # 检查是否有新版本（不下载）
tt version            # 查看当前版本
```

## 目录结构

```
TG/
├─ main.go              程序入口
├─ internal/            Go 后端全部实现（cli / config / web / host / output / update …）
├─ web/                 前端（React + vite，构建产物被 Go 嵌入）
├─ installer/           MSI 安装包定义（WiX v3）
├─ skills/              对外技能手册
└─ testdata/            测试语料（4GL 大纲夹具）
```

## 关于许可证

上游 TT 仓库**未声明开源许可证**（无 `LICENSE` 文件），因此本仓库不沿用任何开源许可条款，
目前仅作个人开发使用。若后续获得上游授权，或 GP 部分已完全重写，再决定采用哪种许可证。

## 致谢

- [TT](https://github.com/luyaogod/TT) — 本项目的架构参考与代码基线
