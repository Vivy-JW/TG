// 调试视图那条**整条 header**：左边是源码页签，右边是三个互相独立的开关。
//
// 为什么它是一条而不是两条（用户 2026-10-16 的定案）：
//   原来页签栏只占"编辑器那一列"，右侧栏自己另有一行 sheet 图标，两块面板各挂一个收展按钮；
//   于是同一条竖直位置上并排着两行控件，"哪一行管哪一块"要靠猜。现在：
//     · 页签在最左（这是"我在看哪个文件"）；
//     · 三个开关在最右（terminal / bug-play / list-tree），各自管一块面，**可同时激活**；
//     · 原来那两个「收起下方面板 / 收起右方面板」按钮没有了 —— terminal 就是前者，
//       bug-play 与 list-tree 就是后者的两块（用户原话："这样我们就不需要收起下方和右侧面板两个按钮了"）。
//
// 三个开关与它们管的东西：
//   terminal   → 下方面板（协议流 / 操作时间线）
//   bug-play   → 右侧那一列的「调试面板」（变量 / 监视 / 断点 / 调用栈）
//   list-tree  → 右侧那一列的「大纲」（树）
// 右侧两块**同时**打开时由外壳做上下分屏（见 App.tsx 的那段注释），本组件只管开关本身。
import { Bug, BugPlay, ListTree, Loader2, Terminal, X } from 'lucide-react'
import { useStore } from './store'
import { ToolIcon } from './Toolbar'
import { cn } from '../../shared/utils'

export function DebugHeader() {
  const tabs = useStore((s) => s.tabs)
  const activeTab = useStore((s) => s.activeTab)
  const setActiveTab = useStore((s) => s.setActiveTab)
  const closeTab = useStore((s) => s.closeTab)
  const prog = useStore((s) => s.prog)

  const showBottom = useStore((s) => s.showBottom)
  const toggleBottom = useStore((s) => s.toggleBottom)
  const rightDebugOpen = useStore((s) => s.rightDebugOpen)
  const rightTreeOpen = useStore((s) => s.rightTreeOpen)
  const toggleRightDebug = useStore((s) => s.toggleRightDebug)
  const toggleRightTree = useStore((s) => s.toggleRightTree)

  const isDebug = !tabs.find((t) => t.key === activeTab)
  // 激活态用主色、未激活用弱化色：三个开关都要能"一眼看出它现在是开还是关"
  // （它们控制的是面板的显隐，不是"当前选中哪一个"，所以不能做成互斥的高亮块）。
  const ACTIVE = 'text-sky-600 dark:text-sky-400'
  const IDLE = 'text-muted-foreground'

  return (
    <div className="flex h-8 shrink-0 items-stretch border-b border-border bg-background">
      {/* 页签区自己滚（overflow-x-auto 只包页签），右边那三个开关不跟着滚走 */}
      <div className="flex min-w-0 flex-1 items-stretch overflow-x-auto">
        <button onClick={() => setActiveTab('debug')}
          className={cn('flex shrink-0 items-center gap-1.5 border-r border-border px-3 text-xs transition-colors',
            isDebug ? 'bg-card font-medium text-foreground' : 'text-muted-foreground hover:bg-accent/60')}>
          <Bug className="h-3 w-3" />
          {prog || '调试'}
        </button>
        {tabs.map((t) => (
          <div key={t.key}
            className={cn('group flex shrink-0 items-center border-r border-border transition-colors',
              activeTab === t.key ? 'bg-card text-foreground' : 'text-muted-foreground hover:bg-accent/60')}>
            <button className="max-w-45 truncate px-3 text-xs" title={t.path || t.file}
              onClick={() => setActiveTab(t.key)}>
              {t.loading ? <Loader2 className="mr-1 inline h-3 w-3 animate-spin" /> : null}
              {t.file}
              {t.missing ? ' (无源码)' : ''}
            </button>
            <button className="mr-1 p-0.5 opacity-40 transition-opacity hover:bg-accent hover:opacity-100"
              onClick={() => closeTab(t.key)}>
              <X className="h-3 w-3" />
            </button>
          </div>
        ))}
      </div>
      <div className="flex shrink-0 items-center gap-0.5 pr-1">
        <ToolIcon icon={Terminal}
          label={showBottom ? '收起下方面板（协议流 / 操作时间线）' : '展开下方面板（协议流 / 操作时间线）'}
          color={showBottom ? ACTIVE : IDLE} onClick={() => void toggleBottom()} />
        <ToolIcon icon={BugPlay}
          label={rightDebugOpen ? '收起调试面板（变量 / 监视 / 断点 / 调用栈）' : '展开调试面板（变量 / 监视 / 断点 / 调用栈）'}
          color={rightDebugOpen ? ACTIVE : IDLE} onClick={() => void toggleRightDebug()} />
        <ToolIcon icon={ListTree}
          label={rightTreeOpen ? '收起大纲（树）' : '展开大纲（树）'}
          color={rightTreeOpen ? ACTIVE : IDLE} onClick={() => void toggleRightTree()} />
      </div>
    </div>
  )
}
