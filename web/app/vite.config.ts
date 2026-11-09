import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// 调试工作台 SPA。由 `tt serve` 挂在 /debug/ 下 —— 统一服务的顶层 mux 在同一进程里
// 还挂着共享层 /api/*，所以 base 必须是 /debug/，否则页面资源路径会和 API 路径搅在一起。
//
// API 分两类：
//   /debug/api/…  调试专属（会话、断点、WebSocket）—— 对外是 /debug/api/，后端挂载后是 /api/
//   /api/…        共享端点（hosts 配置、dbprobe、conntest…）
//
// 开发时 vite 在 /debug/ 上服务 SPA，并把上面两类都代理到后端。
// 后端地址默认 127.0.0.1:28670（tt serve 的默认监听）；端口被占用顺延时用 TT_PROXY 指定。
export default defineConfig(() => {
  const target = process.env.TT_PROXY || 'http://127.0.0.1:28670'
  return {
    base: '/debug/',
    plugins: [react(), tailwindcss()],
    build: {
      // 直接产出到 dist 根:合并后只剩这一套 SPA,再套一层 debug/ 是多余的嵌套。
      // 注意 base 仍然是 /debug/ —— 它决定资源 URL 与路由前缀,与产出目录无关。
      outDir: '../dist',
      emptyOutDir: true,
      chunkSizeWarningLimit: 9000,
    },
    server: {
      proxy: {
        // 调试专属接口：**原样转发**，不剥 /debug。剥前缀的是后端自己
        // （internal/web 用 http.StripPrefix("/debug", debugHandler)，剥完才是调试子系统的 /api/…）。
        // 这里再剥一次就变成共享层的 /api/sessions，命中「未知接口」。
        '/debug/api': { target, ws: true },
        // 共用接口：原样转发
        '/api': { target, ws: true },
      },
    },
  }
})
