# web/app/scripts — 不经打包器的那几个脚本

这里放两类**在 Node 侧跑**的脚本：构建后处理，与前端检查项。

| 脚本 | 何时跑 | 做什么 |
|---|---|---|
| `gitkeep.mjs` | `postbuild` | 把产物目录里的**占位文件**写回去（打包会清空产出目录） |
| `fgltokens.test.mjs` | `check:tokens` | 4GL 词法的检查项 |
| `fgloutline.fixtures.mjs` | `check:outline` | 大纲/折叠对拍（跑 [fgl-fixtures](./fgl-fixtures/README.md) 里的夹具） |
| `store.test.mjs` | `check:store` | 状态容器的检查项（桩 window/document，在 Node 里跑） |

三个检查项都是同一个套路：**用打包器把目标模块打成一个临时脚本，再交给 Node 执行**。
所以被检查的模块必须在没有 DOM 的环境下能被**加载**（也就是模块顶层不能碰 DOM）。

**中间产物 `.tmp-*.mjs` 不入库**，也**不能落在产出目录下** —— 打包脚本会先清空产出目录。

## 判据

```bash
cd web && npm run check:app     # 三项一起跑（工作区根的脚本）
```

## 细节去哪

- 夹具的来源与改动约定 → [fgl-fixtures/README.md](./fgl-fixtures/README.md)
- 脚本怎么被调用 → [../README.md](../README.md)
