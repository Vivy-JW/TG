// shadcn/ui 规范基础组件(轻依赖):语义 token + cn/cva,颜色一律不写死(亮暗由 tokens.css 决定)。
//
// 这里只放**不依赖 Radix 的那部分** —— 只用到按钮/输入框/表格这些的视图,
// 就不必把 Radix 全套也拉进来。带 Radix 的组件在同目录的 ui-radix.tsx。
import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from './utils'

const buttonVariants = cva(
  "inline-flex shrink-0 items-center justify-center gap-1.5 rounded-ui whitespace-nowrap text-sm font-medium transition-colors outline-none focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px] disabled:pointer-events-none disabled:opacity-40 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4",
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        destructive: 'bg-destructive text-white hover:bg-destructive/90',
        outline: 'border border-input bg-transparent hover:bg-accent hover:text-accent-foreground',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        link: 'text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-8 px-3',
        sm: 'h-7 gap-1 px-2 text-xs',
        lg: 'h-10 px-6',
        icon: 'h-8 w-8',
      },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  },
)

// 转发 ref:作为 Radix 的 asChild 子元素时(如 PopoverTrigger)需要真实 DOM 引用做定位锚点
export const Button = React.forwardRef<HTMLButtonElement,
  React.ButtonHTMLAttributes<HTMLButtonElement> &
  VariantProps<typeof buttonVariants> & { asChild?: boolean }
>(function Button({ className, variant, size, asChild = false, ...props }, ref) {
  const Comp = asChild ? Slot : 'button'
  return <Comp ref={ref} className={cn(buttonVariants({ variant, size }), className)} {...props} />
})

const badgeVariants = cva(
  'inline-flex items-center border px-1.5 py-0.5 text-[11px] font-medium',
  {
    variants: {
      tone: {
        default: 'border-transparent bg-secondary text-secondary-foreground',
        green: 'border-emerald-500/20 bg-emerald-500/10 text-emerald-600 dark:text-emerald-400',
        yellow: 'border-yellow-500/20 bg-yellow-500/10 text-yellow-600 dark:text-yellow-400',
        red: 'border-red-500/20 bg-red-500/10 text-red-600 dark:text-red-400',
        blue: 'border-sky-500/20 bg-sky-500/10 text-sky-600 dark:text-sky-400',
        gray: 'border-transparent bg-muted text-muted-foreground',
      },
    },
    defaultVariants: { tone: 'default' },
  },
)

export function Badge({ className, tone, ...props }:
  React.HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeVariants>) {
  return <span className={cn(badgeVariants({ tone }), className)} {...props} />
}

export function Panel({ title, right, children, className }: {
  title: React.ReactNode; right?: React.ReactNode; children: React.ReactNode; className?: string
}) {
  return (
    <div className={cn('flex min-h-0 flex-col border border-border bg-card', className)}>
      <div className="flex h-8 shrink-0 items-center justify-between border-b border-border px-2.5">
        <span className="text-xs font-medium text-muted-foreground">{title}</span>
        {right}
      </div>
      <div className="min-h-0 flex-1 overflow-auto">{children}</div>
    </div>
  )
}

// 有边框的输入框:圆角走全站唯一那一档(--radius-ui),见 tokens.css 的圆角口径。
// 聚焦高亮用 `ring-inset`(**画在边框内侧**),不用默认的外扩环:外扩环会溢出到
// 控件盒子之外,任何一层带 overflow 的祖先(设置页的内容列、可滚表格、未来的新容器)
// 都可能把它裁掉一条边或一个角 —— 表现是"聚焦了但高亮缺一角"。内嵌环画在盒内,
// 与祖先的 overflow 无关,是根治;它也正好与可编辑表格单元格的 CELL_EDITING 同一手法。
// 刻意不加 focus 时的圆角变化 —— 聚焦只改边框色与 ring,形状不跳。
export const Input = React.forwardRef<HTMLInputElement, React.InputHTMLAttributes<HTMLInputElement>>(
  function Input({ className, type, ...props }, ref) {
    return (
      <input
        ref={ref}
        type={type}
        className={cn(
          'flex h-8 w-full min-w-0 rounded-ui border border-input bg-transparent px-2 py-1 text-sm transition-[color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-inset focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30',
          className,
        )}
        {...props}
      />
    )
  },
)

export function Separator({ className, orientation = 'horizontal' }: {
  className?: string; orientation?: 'horizontal' | 'vertical'
}) {
  return (
    <div
      role="separator"
      className={cn(
        'shrink-0 bg-border',
        orientation === 'horizontal' ? 'h-px w-full' : 'h-full w-px',
        className,
      )}
    />
  )
}

// ---- Table(shadcn 规范,对齐 TbmLite 的可编辑表格):单元格自带网格线,控件嵌在格内 ----
// 表格容器横向可滚;TableHead/TableCell 带 border 形成 Excel 式网格(零圆角由 tokens.css 统一)。
// container=false 供「需要表头 sticky」的列表使用:外层 overflow-x-auto 的 div 按 CSS 规则
// 会让 overflow-y 也算成非 visible,于是它成了新的滚动祖先,里层 th 的 sticky 就失效了
// (滚动容器改由调用方提供)。默认仍套容器,不影响既有网格表格。
export function Table({ className, container = true, ...props }:
  React.ComponentProps<'table'> & { container?: boolean }) {
  const table = <table className={cn('w-full caption-bottom border-collapse text-sm', className)} {...props} />
  if (!container) return table
  return <div className="relative w-full overflow-x-auto">{table}</div>
}

export function TableHeader({ className, ...props }: React.ComponentProps<'thead'>) {
  return <thead className={cn('', className)} {...props} />
}

export function TableBody({ className, ...props }: React.ComponentProps<'tbody'>) {
  return <tbody className={cn('', className)} {...props} />
}

export function TableRow({ className, ...props }: React.ComponentProps<'tr'>) {
  return <tr className={cn('transition-colors hover:bg-muted/50', className)} {...props} />
}

export function TableHead({ className, ...props }: React.ComponentProps<'th'>) {
  return (
    <th
      className={cn(
        'h-7 border border-border bg-muted/50 px-2 text-left align-middle text-[11px] font-medium whitespace-nowrap text-muted-foreground',
        className,
      )}
      {...props}
    />
  )
}

export function TableCell({ className, ...props }: React.ComponentProps<'td'>) {
  return (
    <td className={cn('border border-border p-0 align-middle whitespace-nowrap', className)} {...props} />
  )
}

// 可编辑表格里「正在编辑这一格」的提示环。挂在做 p-0 的 TableCell 上,
// 用 ring 语义 token 而不是写死的颜色 —— 合并前两套 SPA 各写了一份 sky 色,
// 其中一份在暗色下对比度不够。
export const CELL_EDITING = 'focus-within:ring-1 focus-within:ring-inset focus-within:ring-ring/60'
