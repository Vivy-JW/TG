// shadcn/ui 规范里**需要 Radix 的那部分**:下拉、弹层、复选、日历、确认弹窗、手风琴。
// 拆成单独文件是为了让不依赖 Radix 的视图只引 ui.tsx(不拉 Radix 全家桶)。
import * as React from 'react'
import * as AccordionPrimitive from '@radix-ui/react-accordion'
import * as AlertDialogPrimitive from '@radix-ui/react-alert-dialog'
import * as DialogPrimitive from '@radix-ui/react-dialog'
import * as CheckboxPrimitive from '@radix-ui/react-checkbox'
import * as PopoverPrimitive from '@radix-ui/react-popover'
import * as ProgressPrimitive from '@radix-ui/react-progress'
import * as SelectPrimitive from '@radix-ui/react-select'
import * as TabsPrimitive from '@radix-ui/react-tabs'
import { DayPicker } from 'react-day-picker'
import { CalendarDays, Check, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react'
import { Button } from './ui'
import { cn } from './utils'

// ---- Select(shadcn 规范,Radix 实现):全站下拉统一用这套,不再用原生 <select> ----
// 触发钮外观对齐 Input(同高同边框同圆角同聚焦环),尺寸/宽度由调用方 className 覆盖(cn 后写优先)。
export function Select(props: React.ComponentProps<typeof SelectPrimitive.Root>) {
  return <SelectPrimitive.Root {...props} />
}

export function SelectTrigger({ className, children, ...props }:
  React.ComponentProps<typeof SelectPrimitive.Trigger>) {
  return (
    <SelectPrimitive.Trigger
      className={cn(
        // rounded-ui:与 Input 同值 —— 上面那句"外观对齐 Input"要真的同值才算数
        // ring-inset:聚焦环画在内侧,理由与 Input 相同(tokens.css 的圆角口径那节有说明)
        'flex h-8 w-fit min-w-0 items-center justify-between gap-1.5 rounded-ui border border-input bg-transparent px-2 py-1 text-sm whitespace-nowrap transition-[color,box-shadow] outline-none',
        'data-[placeholder]:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-inset focus-visible:ring-ring/50',
        'disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30 [&>span]:truncate',
        className,
      )}
      {...props}
    >
      {children}
      <SelectPrimitive.Icon asChild>
        <ChevronDown className="size-4 shrink-0 opacity-50" />
      </SelectPrimitive.Icon>
    </SelectPrimitive.Trigger>
  )
}

export function SelectValue(props: React.ComponentProps<typeof SelectPrimitive.Value>) {
  return <SelectPrimitive.Value {...props} />
}

export function SelectContent({ className, children, position = 'popper', ...props }:
  React.ComponentProps<typeof SelectPrimitive.Content>) {
  return (
    <SelectPrimitive.Portal>
      <SelectPrimitive.Content
        position={position}
        sideOffset={4}
        className={cn(
          'relative z-50 max-h-[var(--radix-select-content-available-height)] min-w-28 overflow-y-auto overflow-x-hidden border border-border bg-popover text-popover-foreground shadow-md',
          position === 'popper' && 'w-full min-w-[var(--radix-select-trigger-width)]',
          className,
        )}
        {...props}
      >
        <SelectPrimitive.Viewport className="p-1">{children}</SelectPrimitive.Viewport>
      </SelectPrimitive.Content>
    </SelectPrimitive.Portal>
  )
}

export function SelectItem({ className, children, ...props }:
  React.ComponentProps<typeof SelectPrimitive.Item>) {
  return (
    <SelectPrimitive.Item
      className={cn(
        'relative flex w-full cursor-default items-center py-1 pr-6 pl-2 text-sm outline-none select-none',
        'focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50',
        className,
      )}
      {...props}
    >
      <span className="absolute right-1.5 flex size-3.5 items-center justify-center">
        <SelectPrimitive.ItemIndicator>
          <Check className="size-3.5 text-primary" />
        </SelectPrimitive.ItemIndicator>
      </span>
      <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
    </SelectPrimitive.Item>
  )
}

// ---- Popover(shadcn 规范,Radix 实现):弹层底色/边框走 popover 语义 token ----
export function Popover(props: React.ComponentProps<typeof PopoverPrimitive.Root>) {
  return <PopoverPrimitive.Root {...props} />
}

export function PopoverTrigger(props: React.ComponentProps<typeof PopoverPrimitive.Trigger>) {
  return <PopoverPrimitive.Trigger {...props} />
}

export function PopoverContent({ className, align = 'end', sideOffset = 4, ...props }:
  React.ComponentProps<typeof PopoverPrimitive.Content>) {
  return (
    <PopoverPrimitive.Portal>
      <PopoverPrimitive.Content
        align={align}
        sideOffset={sideOffset}
        className={cn('z-50 border border-border bg-popover p-0 text-popover-foreground shadow-md outline-none', className)}
        {...props}
      />
    </PopoverPrimitive.Portal>
  )
}

// ---- Tabs(shadcn 规范,Radix 实现) ----
// 设置页「SSH 服务器 / 数据库」用它。相对 shadcn 默认件按本仓库的口径改了三处:
//   1. 圆角一律走 --radius-ui(全站唯一的 6px),不用 rounded-lg/rounded-md —— 那些档位被清零了;
//   2. TabsList 给 muted 底 + p-0.5,活动项用卡片底 + 边框(分段控件观感,与全站朴素风格一致);
//   3. 不加 shadow —— 这个界面的基调是"面板 + 1px 分割线",弹窗之外不铺阴影。
//
// ⚠ 切页签**不卸载**内容:Radix 默认把非活动 TabsContent 从 DOM 里摘掉,非受控输入
// (defaultValue)会被重置。本仓库的表单都是受控的(值在 SettingsView 的 state 里),
// 卸不卸载不影响取值,但调用方仍应传 forceMount + data-[state=inactive]:hidden:
// 少一次卸载/重挂载,滚动位置与 DOM 顺序也更稳。
export function Tabs(props: React.ComponentProps<typeof TabsPrimitive.Root>) {
  return <TabsPrimitive.Root {...props} />
}

export function TabsList({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.List>) {
  return (
    <TabsPrimitive.List
      className={cn(
        'inline-flex w-fit items-center justify-center gap-0.5 rounded-ui border border-border bg-muted p-0.5 text-muted-foreground',
        className,
      )}
      {...props}
    />
  )
}

export function TabsTrigger({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Trigger>) {
  return (
    <TabsPrimitive.Trigger
      className={cn(
        'inline-flex items-center justify-center gap-1.5 rounded-ui px-2.5 py-1 text-xs whitespace-nowrap transition-colors outline-none',
        'text-muted-foreground hover:text-foreground',
        'focus-visible:ring-ring/50 focus-visible:ring-[3px]',
        // 活动态:卡片底 + 边框盖住 muted 底,视觉上"浮起一格"
        'data-[state=active]:border data-[state=active]:border-border data-[state=active]:bg-card data-[state=active]:font-medium data-[state=active]:text-foreground',
        'disabled:pointer-events-none disabled:opacity-50',
        className,
      )}
      {...props}
    />
  )
}

// TabsContent 刻意不预设布局类:设置页这几块内容是内容流里的普通盒子,
// 给它 flex-1 之类会与卡片内的 space-y 打架(官方件的 flex-1 是给整屏页签场景的)。
export function TabsContent({ className, ...props }: React.ComponentProps<typeof TabsPrimitive.Content>) {
  return <TabsPrimitive.Content className={cn('outline-none', className)} {...props} />
}

// ---- Progress(shadcn 规范,Radix 实现) ----
// 圆角走全站唯一那一档(--radius-ui):进度条是细长条,胶囊头在这个界面里会与
// 其它直角块状元素打架,所以轨道与填充同一档圆角。
// 只做「一条轨道 + 一段填充」,百分比文字由调用方放在旁边 —— 数字的字级与对齐
// 才能跟同一行的其它数值(大小、速度、耗时)一致,而不是塞进条子里被压成小字。
//
// barClassName 单独给**填充**那一段:失败要变红这类状态只能由调用方知道。
// 用它的理由是这个填充自带 bg-primary,靠 `[&>div]:bg-*` 之类的任意变体去覆盖
// 要赌 Tailwind 的生成顺序,不如显式把颜色交出去。
export function Progress({ value, className, barClassName, ...props }:
  React.ComponentProps<typeof ProgressPrimitive.Root> & { barClassName?: string }) {
  return (
    <ProgressPrimitive.Root
      className={cn('relative h-1.5 w-full overflow-hidden rounded-ui bg-muted', className)}
      value={value}
      {...props}
    >
      <ProgressPrimitive.Indicator
        className={cn('h-full w-full flex-1 bg-primary transition-[transform] duration-300', barClassName)}
        style={{ transform: `translateX(-${100 - (value || 0)}%)` }}
      />
    </ProgressPrimitive.Root>
  )
}

// ---- Checkbox(shadcn 规范,Radix 实现) ----
export function Checkbox({ className, ...props }: React.ComponentProps<typeof CheckboxPrimitive.Root>) {
  return (
    <CheckboxPrimitive.Root
      className={cn(
        'size-3.5 shrink-0 border border-input outline-none transition-colors',
        'data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground',
        'focus-visible:border-ring focus-visible:ring-ring/50 focus-visible:ring-[3px]',
        'disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    >
      <CheckboxPrimitive.Indicator className="flex items-center justify-center text-current">
        <Check className="size-3" />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  )
}

// ---- Accordion(shadcn 规范,Radix 实现) ----
// 刻意只做「中性的壳」:布局相关的类留给调用方 className。
// 面板那套需要撑满高度、条目之间画分割线,设置页那套是轻量树形 —— 两者需求相反,
// 把任一套的样式写进基础件都会让另一套到处覆盖。
export const Accordion = AccordionPrimitive.Root

export const AccordionItem = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Item>
>(({ className, ...props }, ref) => (
  <AccordionPrimitive.Item ref={ref} className={cn('min-h-0', className)} {...props} />
))
AccordionItem.displayName = 'AccordionItem'

export const AccordionTrigger = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Trigger>
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Trigger ref={ref} className={cn(className)} {...props}>
    {children}
  </AccordionPrimitive.Trigger>
))
AccordionTrigger.displayName = 'AccordionTrigger'

export const AccordionContent = React.forwardRef<
  React.ElementRef<typeof AccordionPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof AccordionPrimitive.Content>
>(({ className, children, ...props }, ref) => (
  <AccordionPrimitive.Content ref={ref} className={cn('min-h-0', className)} {...props}>
    {children}
  </AccordionPrimitive.Content>
))
AccordionContent.displayName = 'AccordionContent'

// 手风琴触发区里那个旋转 chevron(展开时翻 180°)。两处复用,故此收成组件。
export function AccordionChevron({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={cn('shrink-0 text-muted-foreground transition-transform duration-200', className)}
    >
      <path d="m6 9 6 6 6-6" />
    </svg>
  )
}

// ---- Calendar(shadcn 规范,react-day-picker + 语义 token 映射;零圆角) ----
export function Calendar({ className, classNames, ...props }: React.ComponentProps<typeof DayPicker>) {
  return (
    <DayPicker
      className={cn('p-2 text-xs', className)}
      weekStartsOn={1}
      formatters={{
        formatCaption: (d) => `${d.getFullYear()} 年 ${d.getMonth() + 1} 月`,
        formatWeekdayName: (d) => '日一二三四五六'[d.getDay()],
        formatDay: (d) => String(d.getDate()),
      }}
      labels={{ labelPrevious: () => '上个月', labelNext: () => '下个月' }}
      classNames={{
        months: 'relative flex flex-col',
        month: 'flex flex-col gap-2',
        month_caption: 'flex h-7 items-center justify-center',
        caption_label: 'text-xs font-medium text-foreground',
        nav: 'absolute inset-x-1 top-2 flex items-center justify-between',
        button_previous: 'flex h-6 w-6 items-center justify-center text-muted-foreground hover:bg-accent hover:text-foreground',
        button_next: 'flex h-6 w-6 items-center justify-center text-muted-foreground hover:bg-accent hover:text-foreground',
        month_grid: 'border-collapse',
        weekdays: 'flex',
        weekday: 'w-7 py-1 text-[11px] font-normal text-muted-foreground',
        weeks: 'flex flex-col',
        week: 'flex w-full',
        day: 'h-7 w-7 p-0 text-center',
        day_button: 'h-7 w-7 text-xs text-foreground hover:bg-accent hover:text-accent-foreground',
        selected: '[&>button]:bg-primary [&>button]:text-primary-foreground [&>button:hover]:bg-primary',
        today: '[&>button]:font-semibold [&>button]:text-primary',
        outside: '[&>button]:text-muted-foreground/40',
        disabled: 'opacity-30',
        hidden: 'invisible',
        ...classNames,
      }}
      components={{
        Chevron: ({ orientation }) => (
          orientation === 'left' ? <ChevronLeft className="size-3.5" /> : <ChevronRight className="size-3.5" />
        ),
      }}
      {...props}
    />
  )
}

const pad2 = (n: number) => String(n).padStart(2, '0')
/** Date → yyyy-MM-dd(与查询参数同格式) */
export const fmtYmd = (d: Date) => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
/** yyyy-MM-dd → Date(非法/空返回 undefined) */
export const parseYmd = (s: string): Date | undefined => {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec((s || '').trim())
  return m ? new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3])) : undefined
}

// ---- DatePicker(shadcn 规范):按钮显示当前日期,弹层为日历 ----
export function DatePicker({ value, onChange, className, title, placeholder = '选择日期' }: {
  value: string; onChange: (v: string) => void; className?: string; title?: string; placeholder?: string
}) {
  const [open, setOpen] = React.useState(false)
  const sel = parseYmd(value)
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" title={title} aria-label={title || '选择日期'}
          className={cn('h-7 justify-start gap-1.5 px-2 font-normal', !sel && 'text-muted-foreground', className)}>
          <CalendarDays className="size-3.5" />
          {sel ? fmtYmd(sel) : placeholder}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto">
        <Calendar
          mode="single"
          selected={sel}
          defaultMonth={sel}
          onSelect={(d) => { if (d) { onChange(fmtYmd(d)); setOpen(false) } }}
        />
      </PopoverContent>
    </Popover>
  )
}

// ---- AlertDialog(shadcn 规范,Radix 实现):二次确认弹窗(不用浏览器原生 confirm) ----
export function AlertDialog(props: React.ComponentProps<typeof AlertDialogPrimitive.Root>) {
  return <AlertDialogPrimitive.Root {...props} />
}

export function AlertDialogContent({ className, children, onEscapeKeyDown, ...props }:
  React.ComponentProps<typeof AlertDialogPrimitive.Content>) {
  return (
    <AlertDialogPrimitive.Portal>
      <AlertDialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/50" />
      <AlertDialogPrimitive.Content
        className={cn(
          'fixed top-1/2 left-1/2 z-50 flex w-full max-w-sm -translate-x-1/2 -translate-y-1/2 flex-col gap-2 border border-border bg-card p-4 text-xs text-card-foreground shadow-lg outline-none',
          className,
        )}
        // Esc **不关**：AlertDialog 的语义是"用户必须做一次选择"（与下面 Dialog 那条注释
        // 同一件事）。实测 Radix 只默认拦了点遮罩与外部交互，Esc 这一路照旧关 ——
        // 于是"点错了删除"与"随手按一下 Esc"都会静默取消，与注释说的不是一回事。
        // 要关就点「取消」：那是一次明确的选择。
        onEscapeKeyDown={(e) => { e.preventDefault(); onEscapeKeyDown?.(e) }}
        {...props}
      >
        {children}
      </AlertDialogPrimitive.Content>
    </AlertDialogPrimitive.Portal>
  )
}

export function AlertDialogHeader({ className, ...props }: React.ComponentProps<'div'>) {
  return <div className={cn('flex flex-col gap-1.5', className)} {...props} />
}

export function AlertDialogFooter({ className, ...props }: React.ComponentProps<'div'>) {
  return <div className={cn('mt-1 flex items-center justify-end gap-2', className)} {...props} />
}

export function AlertDialogTitle({ className, ...props }:
  React.ComponentProps<typeof AlertDialogPrimitive.Title>) {
  return <AlertDialogPrimitive.Title className={cn('text-sm font-medium text-foreground', className)} {...props} />
}

export function AlertDialogDescription({ className, ...props }:
  React.ComponentProps<typeof AlertDialogPrimitive.Description>) {
  return <AlertDialogPrimitive.Description className={cn('text-xs leading-5 text-muted-foreground whitespace-pre-line', className)} {...props} />
}

export function AlertDialogAction({ className, ...props }:
  React.ComponentProps<typeof AlertDialogPrimitive.Action>) {
  return (
    <AlertDialogPrimitive.Action asChild>
      <Button variant="destructive" className={className} {...props} />
    </AlertDialogPrimitive.Action>
  )
}

export function AlertDialogCancel({ className, ...props }:
  React.ComponentProps<typeof AlertDialogPrimitive.Cancel>) {
  return (
    <AlertDialogPrimitive.Cancel asChild>
      <Button variant="outline" className={className} {...props} />
    </AlertDialogPrimitive.Cancel>
  )
}

// ---- Dialog(Radix):**装内容**的弹窗,与 AlertDialog 分工不同 ----
//
// 为什么不能拿 AlertDialog 顶替(它俩是两套原语,不是同一个东西的两种皮肤):
//   · AlertDialog 语义上要求"用户必须做选择",于是**点遮罩/按 Esc 都不关**
//     (想做也得显式改);它面向的是"确认/取消"两个动作。
//   · 这里要的是**可交互内容**(一列带勾选框的候选项、loading 态、可能还要滚动),
//     用户点外面就该关掉 —— 那是 Dialog 的语义。
// 样式与 AlertDialog 保持同一套 token(无阴影之外的装饰、1px 边框、卡片底),
// 免得两种弹窗看起来像两个产品。
export function Dialog(props: React.ComponentProps<typeof DialogPrimitive.Root>) {
  return <DialogPrimitive.Root {...props} />
}

export function DialogContent({ className, children, ...props }:
  React.ComponentProps<typeof DialogPrimitive.Content>) {
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/50" />
      <DialogPrimitive.Content
        className={cn(
          'fixed top-1/2 left-1/2 z-50 flex max-h-[80vh] w-full max-w-lg -translate-x-1/2 -translate-y-1/2 flex-col gap-2 border border-border bg-card p-4 text-xs text-card-foreground shadow-lg outline-none',
          className,
        )}
        {...props}
      >
        {children}
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  )
}

export function DialogHeader({ className, ...props }: React.ComponentProps<'div'>) {
  return <div className={cn('flex flex-col gap-1.5', className)} {...props} />
}

export function DialogFooter({ className, ...props }: React.ComponentProps<'div'>) {
  return <div className={cn('mt-1 flex items-center justify-end gap-2', className)} {...props} />
}

export function DialogTitle({ className, ...props }:
  React.ComponentProps<typeof DialogPrimitive.Title>) {
  return <DialogPrimitive.Title className={cn('text-sm font-medium text-foreground', className)} {...props} />
}

export function DialogDescription({ className, ...props }:
  React.ComponentProps<typeof DialogPrimitive.Description>) {
  return <DialogPrimitive.Description className={cn('text-xs leading-5 text-muted-foreground whitespace-pre-line', className)} {...props} />
}

export const stateTone = (s: string): 'green' | 'yellow' | 'red' | 'gray' | 'blue' => {
  if (s === 'stopped') return 'yellow'
  if (s === 'running') return 'green'
  if (s === 'exit') return 'red'
  if (s === 'loading') return 'blue'
  return 'gray'
}

export const stateLabel = (s: string) => {
  const m: Record<string, string> = { stopped: '已停站', running: '运行中', exit: '已退出', loading: '启动中' }
  return m[s] || '未连接'
}
