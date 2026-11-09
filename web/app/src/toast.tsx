// 全站统一的消息出口。
//
// 为什么要有这一层:在这之前,"操作失败了""已复制"这类结果各自内联在**触发它的那个
// 组件里** —— 于是它们会挤动所在卡片/工具条的布局(多一行、少一行),而同一件事在不同
// 页面又长得不一样。这个模块把"消息"从"内容"里分出来:内容该留在原地(时间线、
// 变量监视的错误值、日志表格里的错误列 —— 那些是用户要看的数据),而消息弹在右下角,
// 不占布局、不挤动任何东西。
//
// 判据(照着写就不会走偏):
//   · 它是**一次动作的结果**(保存失败/已复制/连接超时) → toast
//   · 它是**内容本身**(表格某列的错、调试器里的变量值) → 留在原地
//   · 它是**某个输入框自己的校验**(这个字段必填) → 留在那个控件旁
//     (搬进 toast 会让人不知道是哪一项不对)
//
// 用法:在应用根挂一次 <ToastProvider>,然后用 useToast():
//   const toast = useToast()
//   toast.error('保存失败: ' + msg)
//   toast.success('已复制')
//   toast.info('正在拉取…')

import * as React from 'react'
import * as ToastPrimitive from '@radix-ui/react-toast'
import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'
import { cn } from '../../shared/utils'
import { registerToast, type ToastApi, type ToastOptions, type ToastTone } from './toastBus'

interface ToastItem {
  id: string
  tone: ToastTone
  title: string
  description?: string
  /** 毫秒;<=0 表示不自动消失(需要用户自己关 —— 留给"必须看到"的失败)。 */
  duration: number
}

const ToastCtx = React.createContext<ToastApi | null>(null)

// 各种语气的默认停留时间。
//
// 失败停留更久(用户多半要去读,甚至去修):8 秒;成功/信息 3.5 秒足够扫一眼。
// 有些失败是"必须看见"的(例如保存被拒),调用方可用 duration: 0 钉住不自动关。
const DEFAULT_DURATION: Record<ToastTone, number> = { success: 3500, info: 3500, error: 8000 }

/**
 * ToastProvider 一次挂全站:它同时提供 useToast() 的出口与 toast 的渲染位置。
 *
 * 为什么合成一个:Radix 的 Toast.Provider 必须**包住** Viewport 与每个 Toast.Root
 * (它给子节点传 swipe 方向等上下文)。分成两个组件的话调用方得记得按正确顺序挂两层,
 * 漏一层就是"toast 永远不出现" —— 而这恰好是最难查的一类问题。
 */
export function ToastProvider({ children }: { children: React.ReactNode }) {
  const [items, setItems] = React.useState<ToastItem[]>([])

  const push = React.useCallback((tone: ToastTone, title: string, opts?: ToastOptions) => {
    // 同一个标题**不重复堆叠**:把它当成"同一件事又发生了一次",就地更新计时。
    // 否则连点两次保存会叠出两个一模一样的框,而用户只做了一件事。
    const id = `${tone}:${title}`
    setItems((prev) => [
      ...prev.filter((x) => x.id !== id),
      {
        id, tone, title,
        description: opts?.description,
        duration: opts?.duration ?? DEFAULT_DURATION[tone],
      },
    ])
    return id
  }, [])

  const api = React.useMemo<ToastApi>(() => ({
    success: (t, o) => push('success', t, o),
    error: (t, o) => push('error', t, o),
    info: (t, o) => push('info', t, o),
    dismiss: (id) => setItems((prev) => prev.filter((x) => x.id !== id)),
  }), [push])

  // 注册给非组件代码(store 的 action)。走 toastBus 而不是本文件导出的单例:
  // store.ts 必须能在裸 Node 里被加载,不能因它而牵进 radix(见 toastBus.ts)。
  React.useEffect(() => {
    registerToast(api)
    return () => registerToast(null)
  }, [api])

  const close = React.useCallback((id: string) => {
    setItems((prev) => prev.filter((x) => x.id !== id))
  }, [])

  return (
    <ToastCtx.Provider value={api}>
      <ToastPrimitive.Provider swipeDirection="right">
        {children}
        {items.map((it) => (
          <ToastRoot key={it.id} item={it} onClose={() => close(it.id)} />
        ))}
        <ToastPrimitive.Viewport
          // 右下角:那里通常没有可交互的主内容,不容易挡住用户正在看的表格/编辑器。
          //
          // bottom 要**让开状态栏**(h-7 = 28px):状态栏右端有一排可点的 chip(接管 / 会话空闲 /
          // 作业名…),toast 压在它们上面时 elementFromPoint 命中的是 toast —— 那些按钮点不动
          // (实测重叠 320×16px)。这里用 28+12 的固定值,不引 StatusBar 的常量:
          // 引进来会让组件层多一条反向依赖,而"状态栏多高"是那一行自己写死的事。
          className="fixed right-3 bottom-10 z-[100] flex max-h-[calc(100vh-3rem)] w-80 flex-col gap-2 outline-none"
        />
      </ToastPrimitive.Provider>
    </ToastCtx.Provider>
  )
}

/** useToast 取消息出口。必须在 <ToastProvider> 内使用。 */
export function useToast(): ToastApi {
  const ctx = React.useContext(ToastCtx)
  if (!ctx) {
    // 不静默降级成 no-op:那样"提示没出现"会变成最难查的一类 bug。
    throw new Error('useToast 必须在 <ToastProvider> 内使用(见 App.tsx 的挂载点)')
  }
  return ctx
}

const TONE_STYLE: Record<ToastTone, { icon: React.ReactNode; border: string }> = {
  success: { icon: <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" />, border: 'border-border' },
  error: { icon: <AlertCircle className="h-4 w-4 shrink-0 text-red-600 dark:text-red-400" />, border: 'border-red-500/40' },
  info: { icon: <Info className="h-4 w-4 shrink-0 text-muted-foreground" />, border: 'border-border' },
}

function ToastRoot({ item, onClose }: { item: ToastItem; onClose: () => void }) {
  const tone = TONE_STYLE[item.tone]
  return (
    <ToastPrimitive.Root
      // 非受控(defaultOpen)而不写 open={true}:写死 open 会让 Radix 认为
      // "打开状态由外面管",于是它**不再自己跑 duration 定时关闭** ——
      // 表现是 toast 永远不消失,得手点。状态的真实来源在 Provider 的数组里:
      // 关掉即从数组移除,所以不需要"重新打开"这条路。
      defaultOpen
      onOpenChange={(o) => { if (!o) onClose() }}
      duration={item.duration > 0 ? item.duration : Infinity}
      // 与全站一致:1px 边框 + 卡片底 + 圆角 token;只留一层投影让它浮起来。
      className={cn(
        'flex w-80 max-w-[calc(100vw-2rem)] items-start gap-2 rounded-ui border bg-card p-3 text-xs text-card-foreground shadow-lg',
        tone.border,
      )}
    >
      <span className="mt-0.5">{tone.icon}</span>
      <div className="min-w-0 flex-1">
        <ToastPrimitive.Title className="break-words font-medium">{item.title}</ToastPrimitive.Title>
        {item.description && (
          <ToastPrimitive.Description className="mt-1 break-words text-muted-foreground">
            {item.description}
          </ToastPrimitive.Description>
        )}
      </div>
      <ToastPrimitive.Close
        title="关闭"
        className="shrink-0 rounded-ui p-0.5 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
      >
        <X className="h-3.5 w-3.5" />
      </ToastPrimitive.Close>
    </ToastPrimitive.Root>
  )
}
