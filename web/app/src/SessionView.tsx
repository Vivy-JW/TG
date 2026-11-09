// 会话页:一条会话对应的工作面板。
//
// 一屏:Debug（原来整块调试工作台,见 DebugWorkbench.tsx）。
import { useEffect, type ReactNode } from 'react'
import { FolderOpen } from 'lucide-react'
import { useStore, type SessionItem } from './store'
import type { SessionTab } from './routing'
import { Pane, TabStrip, ViewHeader, Meta, type TabItem } from './Shell'
import { DebugWorkbench } from './DebugWorkbench'
import { StatusBar } from './StatusBar'

const SESSION_TABS: TabItem<SessionTab>[] = [
  { key: 'debug', label: 'Debug', hint: '断点、单步、变量与调用栈' },
]

export function SessionView({ session, tab, onTab, onOpenEnv, leading }: {
  session: SessionItem
  tab: SessionTab
  onTab: (t: SessionTab) => void
  onOpenEnv: () => void
  leading?: ReactNode
}) {
  // 会话页一进来（以及状态每次变化）就问一句"要不要把母版源码打开"。
  //
  // 为什么放在这一层而不是绑定时一次性做完：会话刚建好是 idle、起跑后变成 running、
  // 命中第一个断点又变 stopped —— 模块名要等**起跑那一刻**服务端才解析出来，
  // 绑定时拉不到。所以由状态变化驱动重试，`ensureSource` 自己带幂等判断（有源码就早退）。
  const state = useStore((s) => s.state)
  useEffect(() => {
    void useStore.getState().ensureSource()
  }, [session.id, state])

  const meta = (
    <>
      <Meta title="会话所属环境">{session.env}</Meta>
      {session.ent && <><span className="opacity-30">·</span>
        <Meta title="企业编号（建会话时点名的那个）">ENT {session.ent}</Meta></>}
      {session.prog && <><span className="opacity-30">·</span>
        <Meta title="作业编号">作业 {session.prog}</Meta></>}
      {session.ref && <><span className="opacity-30">·</span>
        <Meta title="服务端规范 id（环境#序号）">{session.ref}</Meta></>}
    </>
  )

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col bg-background">
      <ViewHeader title={session.name} meta={meta} leading={leading}
        actions={
          <button onClick={onOpenEnv} title={`打开环境「${session.env}」`}
            className="flex h-7 items-center gap-1 rounded-[7px] border border-border px-2 text-[12px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground">
            <FolderOpen className="h-3.5 w-3.5" />
            {session.env}
          </button>
        } />
      <TabStrip items={SESSION_TABS} active={tab} onChange={onTab} />

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <Pane show={tab === 'debug'} label="Debug">
          <DebugWorkbench active={tab === 'debug'} />
        </Pane>
      </div>
      {/* 状态栏（停站橙/运行蓝/空闲灰 + 接管按钮）属于**这条会话**，所以钉在会话页底部，
          不再横跨整个壳子 —— 左侧树和环境页没有"调试状态"可报。 */}
      <StatusBar />
    </div>
  )
}
