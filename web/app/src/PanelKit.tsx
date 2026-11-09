// 右侧手风琴的**外壳件**：条目 / 条目内容 / 条目头行。
//
// 为什么要单独一个文件：这些东西原来住在 Panels.tsx 里，而「大纲」在 2026-10-16 之后
// 也是同一条手风琴里的一个 item（用户要求"两个面板合并成一个手风琴组件"）。大纲住在
// OutlinePanel.tsx，它要复用这三个壳件 —— 从 Panels.tsx 反向 import 会形成环
// （Panels → OutlinePanel → Panels），所以壳件抽到这里，两边都只依赖它。
//
// 共享的手风琴件（shared/ui-radix.tsx）刻意做成中性的壳：设置页那套是轻量树形
// （不撑高、无分割线），与这里的"撑满 + 分割线"相反；把任一套的样式写进共享件
// 都会让另一套到处覆盖。所以这层"面板专用样式"留在前端 app 里。
import * as React from 'react'
import { AccordionChevron, AccordionContent, AccordionItem, AccordionTrigger } from '../../shared/ui-radix'

// 面板条目:在共享的 AccordionItem 之上加「撑满高度 + 条目间 1px 分割线」。
// 展开的条目按内容自然高度分配、空间不够时一起收缩（min-h-0 + 默认 flex-shrink），
// 收起的条目只留标题那一行（data-[state=closed]:flex-none）。
export const PanelItem = React.forwardRef<
  React.ElementRef<typeof AccordionItem>,
  React.ComponentPropsWithoutRef<typeof AccordionItem>
>(({ className, ...props }, ref) => (
  <AccordionItem
    ref={ref}
    className={`flex min-h-0 flex-col overflow-hidden border-b border-border data-[state=closed]:flex-none last:border-b-0 ${className || ''}`}
    {...props}
  />
))
PanelItem.displayName = 'PanelItem'

// 面板条目内容区:撑满剩余高度并自行滚动(共享 AccordionContent 只给了 min-h-0)
export const PanelContent = React.forwardRef<
  React.ElementRef<typeof AccordionContent>,
  React.ComponentPropsWithoutRef<typeof AccordionContent>
>(({ className, ...props }, ref) => (
  <AccordionContent ref={ref} className={`min-h-0 flex-1 overflow-auto ${className || ''}`} {...props} />
))
PanelContent.displayName = 'PanelContent'

// 面板头行:左侧固定位(自动开关或等宽占位),右侧手风琴触发区(整块点击开合)。
// hover 底色挂在整行上(:hover 对父级同样生效),这样左侧图标位与右侧 chevron 一起变色;
// 不做成「左图标各自 hover」——同一行出现两种底色会显得是两个控件。
export function PanelHeader({ leading, children }: { leading?: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="group flex h-8 shrink-0 items-stretch transition-colors hover:bg-accent/50">
      {leading}
      <AccordionTrigger
        className="flex h-full min-w-0 flex-1 items-center justify-between gap-2 pr-2.5 text-xs font-medium text-muted-foreground transition-colors group-hover:text-foreground [&[data-state=open]>svg]:rotate-180"
      >
        <span className="min-w-0 flex-1 truncate pl-1">{children}</span>
        <AccordionChevron />
      </AccordionTrigger>
    </div>
  )
}
