// 设置页的布局件(Card / SettingRow / Field / SectionTitle)。
//
// 统一设置页用它搭各张卡片(含只读的路径行 + 区块标题)。合并前调试工作台与字典页
// 两边各有一套自己的"像卡片的 <section> + 标题"写法,样式靠各写一遍的 Tailwind 类维持一致。
import * as React from 'react'
import { cn } from './utils'

/**
 * 设置卡片:标题(可带说明与右侧操作位) + 一串字段,整张卡是一个竖直序列。
 *
 * 版式对齐 VS Code 的设置页:标题在上、字段在下,**卡片内部不打任何分割线**。
 * 卡片自己就是分组边界,里面再横线切割只会把一屏切成许多小格子,越看越碎。
 * 所以整张卡只有一层内边距,标题与字段之间靠间距分,不靠线。
 *
 * 刻意不用 Panel —— Panel 是 flex 列 + 自带 overflow-auto 的"面板",要撑满父容器;
 * 设置卡片是内容流里的静态盒子,滚动归外层的设置页内容列管。两者混用会套出嵌套滚动容器。
 *
 * 圆角取全站唯一那一档(--radius-ui → rounded-ui),与卡片里的输入框、下拉触发器同值:
 * 卡片与它内部控件的左右边缘落在同一条竖线上,弧度一致才不会看着像两套东西拼起来的。
 * 卡片**里面的那些块状元素**(进度块、虚线框、代码块)保持直角 —— 卡片是分组边界,
 * 里面再套圆角会把一屏碎成许多小圆片。
 *
 * title 可省:不是每张卡片都需要一个标题(「服务器与数据库」那张就不需要 ——
 * 右边一屏本来就只有它一张卡,顶上再写一遍卡名是把左侧选中的环境名说两遍)。
 * 标题/说明/右操作位三样都没有时,整个头部不渲染 —— 否则会留下一条空行把内容顶下去。
 */
export function Card({ id, title, description, right, children, className, panel }: {
  /** 锚点 id:锚点跳转落到这张卡时用它定位,scroll-mt-3 保证落点不贴着面板上沿 */
  id?: string
  /** 卡片标题。省掉 = 这张卡不要头部标题(与 description/right 一起都不给时头部整块不渲染) */
  title?: React.ReactNode
  description?: React.ReactNode
  /** 右侧操作位（可选）：卡片标题旁的按钮/状态 */
  right?: React.ReactNode
  children: React.ReactNode
  className?: string
  /**
   * 当"定高面板"用:自带高度的卡片(站点管理那两栏)必须打开它,否则内容一多就顶出边框。
   * 打开后卡片是个可收缩的 flex 列,内容区 `flex-1 min-h-0 overflow-hidden` —— 滚动由卡片里的列表/页签自己管（顶部操作行不会跟着滚走）。
   * 左栏那种"列表滚、底部按钮钉住"的还要在内容里再分一层(flex-1 min-h-0 overflow-auto 的列表 +
   * 固定高度的按钮行);里面分好之后内容区自己就不会溢出了,滚动条只出现在列表上。
   * 不传就是普通的内容流盒子,跟着内容长高(默认)。
   */
  panel?: boolean
}) {
  const hasHeader = title != null || description != null || right != null
  return (
    <section id={id} className={cn('scroll-mt-3 rounded-ui border border-border bg-card p-3', panel && 'flex min-h-0 flex-col', className)}>
      {hasHeader && (
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            {title != null && <h3 className="text-xs font-medium text-foreground">{title}</h3>}
            {description && (
              <p className="mt-1 text-[11px] leading-4 text-muted-foreground">{description}</p>
            )}
          </div>
          {right && <div className="flex shrink-0 items-center gap-1.5">{right}</div>}
        </div>
      )}
      <div className={cn('space-y-3', hasHeader && 'mt-3', panel && 'flex min-h-0 flex-1 flex-col overflow-hidden')}>{children}</div>
    </section>
  )
}

/**
 * 一条设置:**左说明 + 右控件**(左右两端布局)。
 *
 * 左侧是标签与说明(占满剩余宽度),右侧是控件(定宽,贴右缘)。
 * 这是设置界面最常见的版式(编辑器/浏览器/各 SaaS 的设置页都是这样),
 * 视线沿一条竖线扫下来就能配对,不必在上下两块之间来回找。
 *
 * ⚠ 右侧控件的宽度是这个版式的关键,别写成 w-full 或 max-w-md:
 * 设置页的控件大多只是"一个下拉 / 一个不长的输入框",用不上 400px 以上;
 * 而横向空间是零和的 —— 右栏多占多少,左栏的说明就少多少。曾经的写法
 * (`w-full max-w-md shrink-0`)在窄窗口下会把右栏顶到父宽的最大值,
 * 左栏被压到只剩几个字符宽,说明文字于是**每行一个字**竖着排下来。
 * 现在的 w-80(320px)是实测下来够放下拉/路径输入框、又给说明留出两行余量的宽度。
 * 要更宽的表单请直接改这里,不要去改调用方。
 *
 * 要整行宽的控件(进度块、两张按环境的表格)不走这里,直接做卡片的子元素。
 *
 * 行与行之间有一条 1px 分割线(border-t):左右版式下,左右两块各自成列、
 * 纵向又没有对齐的参照,不加线时相邻两行会糊成一片 —— 上一行的控件与下一行的说明
 * 挨得太近时,读的人分不清哪个控件属于哪条说明。线取 border 语义 token,不用写死颜色。
 * 线画在**每行**的上沿(含第一行,于是卡片标题与第一条设置之间也有一条),
 * 这不是漏判"第一行该不该有线":卡片的 space-y-3 会给每条线上下各留 12px,
 * 所以线不会贴着标题,看着就是一组设置的分行线,而不是把标题切开的一条边。
 * 注意这条线与 Card 那句"卡片内部不打分割线"是有意冲突的:卡片是**分组**边界,
 * 一条条设置需要**行**边界,两种边界不矛盾。
 */
export function SettingRow({ id, label, description, control, className }: {
  /** 锚点 id,与 Card 的 id 一起构成深链接可定位的粒度 */
  id?: string
  label: React.ReactNode
  description?: React.ReactNode
  control: React.ReactNode
  className?: string
}) {
  return (
    <div id={id} className={cn('flex scroll-mt-3 items-start gap-4 border-t border-border pt-3', className)}>
      <div className="min-w-0 flex-1">
        <div className="text-xs text-foreground">{label}</div>
        {description && (
          <div className="mt-1 text-[11px] leading-4 text-muted-foreground">{description}</div>
        )}
      </div>
      {/* max-w-full:窗口很窄时允许它继续收,不至于把左栏挤没 */}
      <div className="w-80 max-w-full shrink-0">{control}</div>
    </div>
  )
}

/** 表单字段:标签在上、控件在下。与 SettingRow 同一套字级与间距,只是不带说明行。 */
export function Field({ label, children, className }: {
  label: React.ReactNode; children: React.ReactNode; className?: string
}) {
  return (
    <label className={cn('block', className)}>
      <span className="block text-xs text-foreground">{label}</span>
      <span className="mt-1.5 block">{children}</span>
    </label>
  )
}

/** 区块小标题:大写短标签 + 一条细线。统一设置页的卡片用它分区。 */
export function SectionTitle({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <div className={cn(
      'mb-1.5 flex items-center gap-2 text-[11px] font-medium tracking-wide text-muted-foreground uppercase',
      className,
    )}>
      <span className="shrink-0">{children}</span>
      <span className="h-px min-w-0 flex-1 bg-border" />
    </div>
  )
}

/**
 * 只读的信息行(运行信息、生效路径这类"看得见改不了"的值)。
 * 竖直间距交给 Card 的 space-y —— 原先自己带 py 会与卡片给的间距叠成两层。
 */
export function InfoRow({ label, value, title }: {
  label: React.ReactNode; value: React.ReactNode; title?: string
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="w-28 shrink-0 text-[11px] text-muted-foreground">{label}</span>
      <span className="min-w-0 flex-1 break-all font-mono text-[11px] text-foreground" title={title}>
        {value}
      </span>
    </div>
  )
}
