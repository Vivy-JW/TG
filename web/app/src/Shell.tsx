// 工作站外壳的几块共用小件:主面板标题行、Tab 条、keep-alive 容器、图标按钮。
//
// 为什么单独一个文件:环境页与会话页是两种目标,但"标题 + Tab + 内容"的形状是同一套;
// 各写一遍的话,underline 的位置、标题字号、边距会在两个页面里慢慢长歪。
import { useEffect, useState, type ReactNode } from 'react'
import type { LucideIcon } from 'lucide-react'
import { cn } from '../../shared/utils'

/** 主面板顶部:左边标题与几粒灰字(宿主/区域/会话数),右边动作按钮。 */
export function ViewHeader({ title, meta, actions, leading }: {
  title: string
  meta?: ReactNode
  actions?: ReactNode
  leading?: ReactNode
}) {
  return (
    <div className="flex h-12 shrink-0 items-center gap-3 px-4">
      {leading}
      <h1 className="min-w-0 max-w-[46%] truncate text-[15px] font-semibold tracking-tight text-foreground" title={title}>
        {title}
      </h1>
      {meta && <div className="flex min-w-0 items-center gap-2 truncate text-[11px] text-muted-foreground">{meta}</div>}
      <div className="ml-auto flex shrink-0 items-center gap-1">{actions}</div>
    </div>
  )
}

/** 标题行里的一粒说明(环境地址 / 区域 / 会话数)。 */
export function Meta({ children, title }: { children: ReactNode; title?: string }) {
  return <span className="shrink-0 whitespace-nowrap" title={title}>{children}</span>
}

/** 标题行右侧的图标按钮。悬停浮起,选中用 accent 底。 */
export function IconBtn({ icon: Icon, label, onClick, active, disabled }: {
  icon: LucideIcon; label: string; onClick: () => void; active?: boolean; disabled?: boolean
}) {
  return (
    <button title={label} onClick={onClick} disabled={disabled}
      className={cn('flex h-7 w-7 items-center justify-center rounded-[7px] transition-colors',
        active ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground',
        'disabled:pointer-events-none disabled:opacity-35')}>
      <Icon className="h-4 w-4" />
    </button>
  )
}

export interface TabItem<K extends string> {
  key: K
  label: string
  icon?: LucideIcon
  hint?: string
}

/**
 * Tab 条(照 DSH:文字 + 细的浅蓝下划线,没有整块底色)。
 * 选中态只由下划线与字色表达 —— 这是这一屏唯一的强调色,别再加第二处。
 */
export function TabStrip<K extends string>({ items, active, onChange }: {
  items: TabItem<K>[]
  active: K
  onChange: (k: K) => void
}) {
  return (
    <div className="flex h-10 shrink-0 items-stretch gap-1 border-b border-border/70 px-2">
      {items.map((it) => {
        const on = it.key === active
        const Icon = it.icon
        return (
          <button key={it.key} onClick={() => onChange(it.key)} title={it.hint}
            className={cn('relative flex items-center gap-1.5 px-3 text-[13px] transition-colors',
              on ? 'text-foreground' : 'text-muted-foreground hover:text-foreground')}>
            {Icon && <Icon className="h-3.5 w-3.5" />}
            {it.label}
            <span className={cn('absolute inset-x-3 bottom-0 h-[2px] rounded-full bg-[var(--dsh-accent)] transition-opacity',
              on ? 'opacity-100' : 'opacity-0')} />
          </button>
        )
      })}
    </div>
  )
}

/**
 * keep-alive:第一次显示之后常驻挂载,切换只改 display。
 *
 * 现有几块画面(Monaco 编辑器、报文日志、日志列表、设置页)都有自己的滚动位置与本地输入,
 * 卸载重挂会把它们丢光 —— 这条在旧的五视图外壳里已经踩实(见 App.tsx 的历史注释),
 * 新外壳换成 Tab 之后约束不变。
 */
export function Pane({ show, label, children }: { show: boolean; label: string; children: ReactNode }) {
  const [mounted, setMounted] = useState(show)
  useEffect(() => { if (show) setMounted(true) }, [show])
  if (!mounted) return null
  return (
    <div className={show ? 'flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden' : 'hidden'}>
      {children}
    </div>
  )
}