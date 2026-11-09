// TOP GP 环境清单的读入口。
//
// 数据本体在 store 里（`gpEnvs` / `gpEnvsLoaded` / `gpEnvsErr`），和 T100 的 `envs` 并列；
// 这里只放两件东西：启动时拉一次，以及"同步/镜像"两个动作 —— 动作完成后刷新那份切片。

import { useStore, type GpEnvBrief } from './store'

export type { GpEnvBrief as GpEnvView }

function base(): string {
  return '' // 固定同源
}

/** 页面一打开就拉，别等用户点进设置页；重复调用由 store 自己合并。 */
export function refreshGpEnvs() {
  return useStore.getState().loadGpEnvs()
}

export function gpSync(name: string) {
  return fetch(base() + '/api/gp/sync', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ env: name }),
  }).then(() => refreshGpEnvs())
}

/** 改一个环境的某项配置（镜像目录之类）。只发这一份，密码由服务端沿用旧的。 */
export async function gpPatchEnv(name: string, patch: Partial<GpEnvBrief>) {
  // 只发补丁：旧实现会把整条 env 连同空的 sid/账号一起发过去，后端整条替换，
  // 于是改一次镜像目录就把 service 和 db 密码抹了。
  await fetch(base() + '/api/gp/env', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ action: 'save', name, patch }),
  })
  await refreshGpEnvs()
}

/** 弹本机选目录对话框，返回选中的绝对路径；取消返回空串。
 *
 * 端点用**共享的** `/api/pickdir`（`internal/web/pickdir.go`）——合并成一个进程之前
 * 这里写的是 GP 服务的 `/api/gp/pickdir`，那个前缀在统一服务下不存在，于是 404、
 * 函数吞掉异常返回空串，表现是"选目录按钮没反应"。T100 侧 `api.ts` 的 `pickDir` 调的就是同一个端点。
 */
export async function gpPickDir(initial: string): Promise<string> {
  try {
    const r = await fetch(base() + '/api/pickdir', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ initial, title: '选择 GP 源码镜像目录' }),
    })
    const j = await r.json()
    return j?.ok ? String(j.path || '') : ''
  } catch {
    return ''
  }
}

export function gpMirror(name: string, mode: 'incr' | 'full') {
  return fetch(base() + '/api/gp/mirror', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ env: name, mode }),
  }).then(() => refreshGpEnvs())
}

export function useGpEnvs(): { envs: GpEnvBrief[]; ready: boolean; refresh: () => void } {
  const envs = useStore((s) => s.gpEnvs)
  const ready = useStore((s) => s.gpEnvsLoaded)
  return { envs, ready, refresh: () => { void refreshGpEnvs() } }
}

// 模块加载即拉一次（与 T100 的 loadEnvs 在 App 挂载时拉是同一种"尽早"）。
void refreshGpEnvs()

// GP 表单保存后广播一声，让左树/SettingsView 那份 store 清单跟着刷新。
// 以前只 dispatch 没人接，于是新环境/改名要刷新整页才看得到。
if (typeof window !== 'undefined') {
  window.addEventListener('tt-gp-envs', () => { void refreshGpEnvs() })
}
