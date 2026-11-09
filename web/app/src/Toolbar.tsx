// 浮动调试工具条(脱离文档流,手柄拖拽,位置记忆)。
//
// 顶层那条横栏已经撤掉:它除了两个面板收展按钮之外空空如也,而那两个按钮只管
// 这一屏的下方面板与右侧栏,挂在全站共用的横栏上并不成立 —— 它们现在钉在
// 调试页签栏的右端(SourceView.tsx)。横栏一去,调试区也多了 32px 高度。
import { useState, type ComponentType, type ReactNode } from 'react'
import {
  RedoDot, StepForward, ArrowDownToDot, ArrowUpFromDot,
  RotateCcw, Square, GripVertical,
} from 'lucide-react'
import { useStore } from './store'

// 工具条内图标按钮(无独立边框,悬停浮起,禁用半透明)。页签栏右端的收展按钮也用它,
// 所以导出。
export function ToolIcon({ icon: Icon, label, onClick, disabled, color = 'text-sky-600 dark:text-sky-400' }: {
  icon: ComponentType<{ className?: string }>
  label: string
  onClick: () => void
  disabled?: boolean
  color?: string
}) {
  return (
    <button
      title={label}
      disabled={disabled}
      onClick={onClick}
      className={`rounded-ui p-1 transition-colors hover:bg-accent/60 disabled:pointer-events-none disabled:opacity-30 ${color}`}
    >
      <Icon className="h-4 w-4" />
    </button>
  )
}

// 浮动工具条容器:脱离文档流,拖左侧手柄移动,位置记忆到 localStorage
function FloatingToolbar({ children }: { children: ReactNode }) {
  // 右侧那一列只要开着（调试面板或大纲任一块）就要让开它的宽度 —— 工具条默认落在它左边。
  //
  // ⚠ 两个 store 订阅必须**各自独立调用**：写成 `useStore(a) || useStore(b)` 时左边为真就短路，
  // 右边那次 hook 调用被跳过 —— 渲染次数一变 hook 数量就变，React 抛 #310
  // （"Rendered more hooks than during the previous render"），而且只在**状态切换时**才炸
  // （实测：开着调试面板再点开大纲，整块调试视图直接进错误边界）。值在下面合。
  // 同一个坑 App.tsx 的 sessionId 那条注释早写过一次。
  const rightDebugOpen = useStore((s) => s.rightDebugOpen)
  const rightTreeOpen = useStore((s) => s.rightTreeOpen)
  const showRight = rightDebugOpen || rightTreeOpen
  // 右侧栏宽度从 **store** 读(不再渲染期读 localStorage):拖宽右栏时写的是 store,
  // 而 localStorage 要到 mouseup 才更新 —— 渲染期读它，工具条就会一直停在旧位置，
  // 直到下一次重渲染才归位（实测拖到 640 后还压着右栏 184px，点一下别的才跳过去）。
  const panelW = useStore((s) => s.panelW)
  const [pos, setPos] = useState(() => {
    try {
      const v = JSON.parse(localStorage.getItem('tt.toolbarPos') || '')
      if (typeof v?.top === 'number' && typeof v?.left === 'number') {
        // 视口夹取:位置是在**当时那个窗口**里拖出来的。换个小屏/小窗口打开时，
        // 一个合法的旧位置可能整个落在视口外 —— 那套"继续/步过/结束调试"就此消失，
        // 而用户没有任何归位入口（只能清 localStorage）。这里按当前视口夹回可见范围。
        const top = Math.max(4, Math.min(Math.max(4, window.innerHeight - 60), v.top))
        const left = Math.max(4, Math.min(Math.max(4, window.innerWidth - 120), v.left))
        return { top, left }
      }
    } catch { /* 忽略 */ }
    return null // null = 默认定位(代码编辑器右上角,由 CSS 类给值)
  })

  const onDragStart = (e: React.MouseEvent) => {
    e.preventDefault()
    const el = (e.currentTarget.parentElement as HTMLElement)
    const rect = el.getBoundingClientRect()
    const dx = e.clientX - rect.left
    const dy = e.clientY - rect.top
    const move = (ev: MouseEvent) => {
      const left = Math.max(4, Math.min(window.innerWidth - rect.width - 4, ev.clientX - dx))
      const top = Math.max(4, Math.min(window.innerHeight - rect.height - 4, ev.clientY - dy))
      setPos({ top, left })
    }
    const up = () => {
      setPos((p: { top: number; left: number } | null) => {
        localStorage.setItem('tt.toolbarPos', JSON.stringify(p))
        return p
      })
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
      // 兜住「收不到 mouseup」：鼠标在窗口外松开、或拖拽中窗口失焦 —— 否则这条 mousemove
      // 会一直挂着，面板跟着光标跑，直到用户再点一下（读码结论；难复现，但后果难看）。
      window.removeEventListener('blur', up)
      document.removeEventListener('mouseleave', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
    window.addEventListener('blur', up)
    document.addEventListener('mouseleave', up)
  }

  // 默认:代码编辑器右上角(避开右侧栏自己的宽度;右侧栏收起时贴视口右缘)。
  // top 128 = 主面板标题行 48 + Tab 条 40 + 调试页签行约 32 + 一点余量:新外壳把
  // 「标题 / Tab」两层加在了调试区之上,这个数要跟着下移 ——
  // 仍旧落在编辑器自己的右上角,不压住那两层,也不压住页签栏右端的收展按钮。
  // right 只要让开右侧栏自己的宽度(新外壳没有那根竖排图标栏了,不能再按它算)。
  const style = pos
    ? { top: pos.top, left: pos.left }
    : { top: 128, right: showRight ? panelW + 8 : 8 }

  return (
    <div
      // rounded-ui:全站唯一那一档圆角(见 tokens.css 的文件头)。这块工具条是**浮在编辑器上**的
      // 一块面,与面板/按钮同属"面上的方块",直角色块浮在上面看着像没画完。
      className="fixed z-50 inline-flex items-center gap-0.5 rounded-ui border border-border/70 bg-card/95 px-1 py-0.5 shadow-xl"
      style={style}
    >
      <span
        title="拖拽移动工具条"
        onMouseDown={onDragStart}
        className="cursor-grab p-0.5 text-muted-foreground hover:bg-accent/60 hover:text-muted-foreground active:cursor-grabbing"
      >
        <GripVertical className="h-3.5 w-3.5" />
      </span>
      {children}
    </div>
  )
}

export function Toolbar() {
  const launching = useStore((s) => s.launching)
  const quit = useStore((s) => s.quit)
  const restart = useStore((s) => s.restart)
  const control = useStore((s) => s.control)
  const view = useStore((s) => s.view)

  const state = useStore((s) => s.state)
  const sessionId = useStore((s) => s.sessionId)
  const stopped = state === 'stopped'
  // 存在本轮调试(idle/exit/无会话之外)时才允许「结束调试」/步进类
  const inRun = !!sessionId && state !== '' && state !== 'exit' && state !== 'idle'
  // 协作模式下这些写操作归 AI —— 界面收敛成观察台。服务端同样会 403,
  // 这里禁用只是不让界面装作能做;原因写在底部状态栏的横幅上。
  const ro = useStore((s) => s.mode) === 'collab'

  // 浮动调试工具条(只在调试视图、且有会话时显示)
  if (view !== 'debug') return null
  return (
    <FloatingToolbar>
      <ToolIcon icon={StepForward} label={ro ? '协作模式下由 AI 主导' : '继续 (F5) — 运行到下一个断点'} disabled={ro || !stopped}
        onClick={() => void control('continue')} />
      <ToolIcon icon={RedoDot} label={ro ? '协作模式下由 AI 主导' : '步过 (F10)'} disabled={ro || !stopped}
        onClick={() => void control('next')} />
      <ToolIcon icon={ArrowDownToDot} label={ro ? '协作模式下由 AI 主导' : '步入 (F11)'} disabled={ro || !stopped}
        onClick={() => void control('step')} />
      <ToolIcon icon={ArrowUpFromDot} label={ro ? '协作模式下由 AI 主导' : '步出 (finish)'} disabled={ro || !stopped}
        onClick={() => void control('finish')} />
      <ToolIcon icon={RotateCcw} label={ro ? '协作模式下由 AI 主导' : '重新开始 — 复用会话重启同一作业'} color="text-green-600 dark:text-green-400" disabled={ro || launching || !sessionId}
        onClick={() => void restart()} />
      <ToolIcon icon={Square} label={ro ? '协作模式下由 AI 主导' : '结束调试 — 只结束本轮运行(会话保留)'} color="text-red-600 dark:text-red-400" disabled={ro || !inRun}
        onClick={() => void quit()} />
    </FloatingToolbar>
  )
}
