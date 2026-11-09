// toast 的**零依赖**出口,专给不在 React 组件里的代码用(store 的异步 action)。
//
// 为什么要单独一个文件、而不是直接从 toast.tsx 导出一个单例:
// `store.ts` 有一条纪律 —— 它必须能在**没有浏览器**的 Node 里被加载
// (见 scripts/store.test.mjs:它补几个全局后直接 import store)。而 toast.tsx 里
// 有 React 组件与 @radix-ui/react-toast,把 radix 拖进那条链路就可能在那里炸掉。
// 所以这里只放类型与一个注册点,不含任何运行时依赖。

export type ToastTone = 'success' | 'error' | 'info'

export interface ToastOptions {
  description?: string
  /** 毫秒;<=0 = 不自动消失(留给"必须看到"的失败)。 */
  duration?: number
}

export interface ToastApi {
  success: (title: string, opts?: ToastOptions) => string
  error: (title: string, opts?: ToastOptions) => string
  info: (title: string, opts?: ToastOptions) => string
  dismiss: (id: string) => void
}

/** ToastProvider 挂载时把自己注册进来;卸载时置空。 */
export function registerToast(api: ToastApi | null) {
  current = api
}

let current: ToastApi | null = null

/**
 * toast 全局出口。
 *
 * Provider 还没挂载时**静默丢弃**:这与组件里 useToast 抛错的策略不同,是刻意的 ——
 * 组件里写错是编码问题(越早暴露越好),而 store 的 action 可能在极早期被触发
 * (例如启动时重连),那时丢掉一条提示远好过让整个应用崩在一条消息上。
 */
export const toast: ToastApi = {
  success: (t, o) => current?.success(t, o) ?? '',
  error: (t, o) => current?.error(t, o) ?? '',
  info: (t, o) => current?.info(t, o) ?? '',
  dismiss: (id) => current?.dismiss(id),
}
