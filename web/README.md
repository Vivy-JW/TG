# web — 前端

一套单页应用 + 一套共享设计件。用 npm 的工作区组织（根是 `web/`，workspace 唯一成员是 `app/`）。

```
web/
├─ package.json        工作区根：构建与检查的入口脚本
├─ app/                唯一 SPA（调试工作台 + 统一设置页）
├─ shared/             设计系统与共享件，被 app 以相对路径引用
```

## 与后端的关系

- **产物被后端内嵌**：构建输出到 `web/dist`，由后端编译时嵌进二进制。
- **挂载前缀是 `/debug/`**：它决定资源 URL 与浏览器路由前缀。
  **产出目录与它无关** —— 产出直接落在 `dist` 根，不再套一层子目录。
- 产物目录里有一个**占位文件**，让后端在没有前端产物时也能构建通过（那样得到的二进制
  没有界面，服务返回一张写着构建命令的引导页，而不是静默空白）。

## 开发模式

```bash
cd web && npm run dev:app      # 热更新；默认代理到后端 127.0.0.1:28670
tt serve                       # 后端
```

代理分两类：**调试专属接口**剥掉 `/debug` 前缀再转给后端（后端内部按 `/api/…` 注册），
**共享端点**原样转发。后端端口被占用会自动顺延，这时用环境变量 `TT_PROXY` 指过去。

## 两条纪律

1. **类型检查在构建里**，不在检查项里：`npm run build` 会先做一次全量类型检查再打包。
2. **检查项跑在 Node 里**（不需要浏览器）—— 所以 `src/` 里那些纯函数模块必须能在
   没有 DOM 的环境下被加载。这条约束决定了几个模块的写法（见 [app/src](./app/src/README.md)）。

## 判据

```bash
cd web && npm run check:app     # 三项检查：词法 / 大纲 / 状态容器
cd web && npm run build         # 含类型检查
```

## 细节去哪

- SPA 的构建配置与脚本 → [app/README.md](./app/README.md)
- 应用源码的结构 → [app/src/README.md](./app/src/README.md)
- 设计系统与共享件 → [shared/README.md](./shared/README.md)
- Node 侧的检查脚本与夹具 → [app/scripts/README.md](./app/scripts/README.md)
