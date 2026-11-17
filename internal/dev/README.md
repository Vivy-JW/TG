# internal/dev — 4GL 语法与测试夹具

> 2026-11-11 起，`.tzc` 代码包与 `.tzs` 表单包两条设计器管线（`fence` / `verify` /
> `store` / `split` / `pkgfile` / `synth` / `tapfile` / `tglfile` / `tzs` / `model`
> 与它们的命令面 `dev tzc`、`dev tzs`）整体删除：TG 只面向 TOP GP，不做设计器包。
> `engine/` 与 `designer-src/` 也在 2026-11-12 一并删除（删前打了归档 zip，在仓库外）。
> 现在这个目录只剩两样还在用的东西。

## `fgl/`

4GL 源码的语法、块与大纲解析：块信封（`FUNCTION` / `MAIN` / `DIALOG` / `REPORT`）、
函数与 UI 块树、`#include` 的处理。`tg outline` 靠它把文件变成大纲树；
`tg debug locate` 判"函数在第几行"也走这里。纯函数、零 IO。

## `testutil/`

管线测试夹具（造临时目录、写文件、比对稳定 JSON）。被 `fgl` 的测试与
`internal/cli/outline` 的测试共用。

## 想改这里之前

- 改**命令面**（动词、退出码、JSON 形状）→ 不在这个包，去 [internal/cli/outline](../cli/outline)。
- 改**解析**（新语法、新块类型）→ 在 `fgl/`，并同步 `testdata/fgl-fixtures/` 里的夹具。

## 判据

```bash
go test ./internal/dev/...
```
