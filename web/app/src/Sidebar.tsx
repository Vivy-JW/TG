// 左侧工作区树:一级是环境(设置 → 站点管理里配的那些),二级是会话。
//
// 环境来自真实配置(`/api/hosts` 的 `sshs`),会话来自 `store.sessions` —— 那里面每一条都是
// **服务端真实的会话句柄**(`POST /api/sessions/new` 建出来的,见 store.createSession)。
// 形状按 DSH 的工作站:顶部品牌 + 「新会话」、中间一棵树、底部设置。
//
// 会话行的形状(用户 2026-10-18 定的):
//   · 左边一个**状态圆点**(灰=空闲 / 蓝=登录中 / 绿=运行 / 橙=停站 / 红=已退出),
//     不再用"聊天"图标 —— 图标说不了这条会话现在能不能用,颜色能;
//   · 圆点与文字的起点对齐父目录(环境)那一行的文件夹图标与名字,树看起来才是两列;
//   · 悬停时右侧换成「结束会话」按钮(点击**不**进会话页,只收掉它)。
//
// 交互:点会话名进会话页;小箭头只负责展开/收起(改名/删除留到后面,别在壳子里塞半套 CRUD)。
import { useEffect, useMemo, useState } from 'react'
import {
  ChevronRight, CircleX, Folder, FolderOpen, FolderPlus, PanelLeftClose,
  Plus, RefreshCw, Search, Settings, X,
} from 'lucide-react'
import { cn } from '../../shared/utils'
import type { Route } from './routing'
import type { EnvBrief, SessionItem } from './store'

export interface SidebarProps {
  envs: EnvBrief[]
  envsLoaded: boolean
  envsErr: string
  sessions: SessionItem[]
  /** 会话 id → 服务端状态(`idle` / `loading` / `running` / `stopped` / `exit`)。 */
  states: Record<string, string>
  route: Route
  onSelectEnv: (env: string) => void
  /** TOP GP 的环境：与 T100 并列出现在树上，但由另一套服务管。 */
  gpEnvs: { name: string; host?: string; port?: number }[]
  onSelectGpEnv: (env: string) => void
  onSelectSession: (s: SessionItem) => void
  onCloseSession: (s: SessionItem) => void
  onNewSession: () => void
  onSettings: () => void
  onRefreshEnvs: () => void
  onAddEnv: () => void
  onCollapse: () => void
}

/** 会话类型的短名(左树与会话页共用这一份口径)。 */
export const SESSION_TYPE_LABEL: Record<SessionItem['type'], string> = { debug: 'Debug' }

/**
 * 会话状态 → 圆点颜色 + 一句人话。
 *
 * 颜色与调试页的语义一致(停站橙 / 运行蓝 / 空闲灰),但**运行是绿**:树上要的是
 * "这条会话此刻是不是活的"这一眼,与状态栏那条"运行 = 蓝"表达的是不同的东西
 * (后者是 VS Code 的状态条配色)。两种口径都在各自的位置有说明,别再往一处合并。
 */
export const SESSION_STATE: Record<string, { dot: string; label: string }> = {
  idle: { dot: 'bg-muted-foreground/45', label: '空闲（已登录，可直接启动调试）' },
  loading: { dot: 'bg-sky-500 animate-pulse', label: '登录中…' },
  running: { dot: 'bg-emerald-500', label: '运行中' },
  stopped: { dot: 'bg-amber-500', label: '已停站' },
  exit: { dot: 'bg-red-500', label: '已退出' },
}

function stateOf(states: Record<string, string>, id: string) {
  return SESSION_STATE[states[id] || ''] || { dot: 'bg-muted-foreground/25', label: '状态未知' }
}

export function Sidebar({
  envs, gpEnvs = [], envsLoaded, envsErr, sessions, states, route,
  onSelectEnv, onSelectGpEnv, onSelectSession, onCloseSession, onNewSession, onSettings, onRefreshEnvs, onAddEnv, onCollapse,
}: SidebarProps) {
  // 展开状态:没记过就是展开(新环境默认把会话露出来),用户收起后按环境名记住。
  const [closed, setClosed] = useState<Record<string, boolean>>({})
  const [filterOpen, setFilterOpen] = useState(false)
  const [q, setQ] = useState('')

  // 从外部(深链接/刷新)落到某条会话时,它所在的环境必须展开 —— 否则树里看不到"当前在哪"。
  const routeSessionEnv = route.kind === 'session' ? sessions.find((s) => s.id === route.id)?.env : undefined
  useEffect(() => {
    if (!routeSessionEnv) return
    // GP 环境的展开键带 `gp:` 前缀，避免和同名 T100 环境共用一份折叠状态。
    const key = gpEnvs.some((g) => g.name === routeSessionEnv) ? 'gp:' + routeSessionEnv : routeSessionEnv
    setClosed((c) => (c[key] ? { ...c, [key]: false } : c))
  }, [routeSessionEnv, gpEnvs])

  const query = q.trim().toLowerCase()
  const byEnv = useMemo(() => {
    const m = new Map<string, SessionItem[]>()
    for (const s of sessions) {
      const list = m.get(s.env)
      if (list) list.push(s)
      else m.set(s.env, [s])
    }
    return m
  }, [sessions])

  // 过滤:环境名或它下面任一条会话名命中就留下;命中会话名时只列命中的那几条,
  // 免得搜索结果的上下文被无关会话淹掉。
  const shown = useMemo(() => {
    if (!query) return envs.map((e) => ({ env: e, hits: byEnv.get(e.name) || [] }))
    return envs.flatMap((e) => {
      const list = byEnv.get(e.name) || []
      const hits = list.filter((s) => s.name.toLowerCase().includes(query))
      const envHit = e.name.toLowerCase().includes(query)
      if (!envHit && hits.length === 0) return []
      return [{ env: e, hits: envHit ? list : hits }]
    })
  }, [envs, byEnv, query])

  // GP 与 T100 用同一份 sessions 池，所以 GP 环境下面也要按 env 名挂会话；
  // 搜索规则也保持一致：命中环境名列全部，命中会话名只列命中项。
  const shownGp = useMemo(() => {
    if (!query) return gpEnvs.map((e) => ({ env: e, hits: byEnv.get(e.name) || [] }))
    return gpEnvs.flatMap((e) => {
      const list = byEnv.get(e.name) || []
      const hits = list.filter((s) => s.name.toLowerCase().includes(query))
      const envHit = e.name.toLowerCase().includes(query)
      if (!envHit && hits.length === 0) return []
      return [{ env: e, hits: envHit ? list : hits }]
    })
  }, [gpEnvs, byEnv, query])

  const rowBase = 'group flex h-8 w-full items-center gap-2 rounded-[8px] pr-2 text-left text-[13px] transition-colors'
  const rowIdle = 'text-muted-foreground hover:bg-accent/60 hover:text-foreground'
  const rowOn = 'bg-accent text-foreground'

  const isEnvOn = (name: string) => route.kind === 'env' && route.env === name

  const total = sessions.length

  return (
    <aside className="flex w-[272px] shrink-0 flex-col overflow-hidden bg-card">
      {/* 品牌行:与 DSH 的「deepseek HARNESS」同构 —— 字标 + 一个白底小徽章 + 收起按钮 */}
      <div className="flex h-14 shrink-0 items-center gap-2 px-3.5">
        <span className="text-[16px] font-semibold tracking-tight text-foreground">TG</span>
        <div className="flex items-center gap-1" title="TOP GP 版">
          <span className="rounded-[3px] bg-primary/15 px-1 py-px text-[9px] font-semibold leading-[1.4] text-primary">TOP GP</span>
        </div>
        <button onClick={onCollapse} title="收起左侧目录"
          className="ml-auto flex h-7 w-7 items-center justify-center rounded-[7px] text-muted-foreground transition-colors hover:bg-accent/60 hover:text-foreground">
          <PanelLeftClose className="h-4 w-4" />
        </button>
      </div>

      {/* 新会话:与环境页的「创建会话」Tab 是同一个动作,这里只是给它一个常驻入口 */}
      <div className="px-3">
        <button onClick={onNewSession} disabled={envs.length + gpEnvs.length === 0}
          title={envs.length + gpEnvs.length === 0 ? '先在设置里配一个环境' : '在环境页的「创建会话」里新建一条会话'}
          className="flex h-9 w-full items-center justify-center gap-1.5 rounded-[10px] border border-border bg-secondary/70 text-[13px] text-foreground transition-colors hover:bg-accent disabled:pointer-events-none disabled:opacity-40">
          <Plus className="h-4 w-4" />
          新会话
        </button>
      </div>

      {/* 区块标题:环境 + 三个小动作(搜索 / 刷新 / 去设置里加环境) */}
      <div className="mt-3 flex h-8 shrink-0 items-center gap-0.5 px-3.5">
        <span className="flex-1 text-[12px] text-muted-foreground">
          环境{total > 0 && <span className="ml-1 text-[11px] opacity-70">{envs.length}</span>}
        </span>
        <SidebarIcon icon={Search} label="搜索环境 / 会话" active={filterOpen}
          onClick={() => { setFilterOpen((v) => !v); if (filterOpen) setQ('') }} />
        <SidebarIcon icon={RefreshCw} label="重新读取环境清单" onClick={onRefreshEnvs} />
        <SidebarIcon icon={FolderPlus} label="去设置里添加环境" onClick={onAddEnv} />
      </div>

      {filterOpen && (
        <div className="px-3 pb-1">
          <div className="flex h-7 items-center gap-1 rounded-[7px] border border-border bg-background px-2">
            <Search className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <input autoFocus value={q} onChange={(e) => setQ(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Escape') { setQ(''); setFilterOpen(false) } }}
              placeholder="搜索环境 / 会话"
              className="min-w-0 flex-1 bg-transparent text-[12px] text-foreground outline-none placeholder:text-muted-foreground" />
            {q && (
              <button onClick={() => setQ('')} title="清空" className="text-muted-foreground hover:text-foreground">
                <X className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-2">
        {envsErr ? (
          <div className="space-y-1.5 p-2 text-xs">
            <div className="text-red-600 dark:text-red-400">读取环境清单失败</div>
            <div className="break-all text-muted-foreground">{envsErr}</div>
            <button onClick={onRefreshEnvs}
              className="rounded-[7px] border border-border px-2 py-0.5 text-muted-foreground hover:bg-accent hover:text-foreground">
              重试
            </button>
          </div>
        ) : !envsLoaded ? (
          <div className="p-2 text-xs text-muted-foreground">正在读取环境…</div>
        ) : envs.length === 0 ? (
          <div className="space-y-2 p-2 text-xs text-muted-foreground">
            <div>还没有环境。</div>
            <button onClick={onAddEnv}
              className="rounded-[7px] border border-border px-2 py-1 text-foreground hover:bg-accent">
              去设置 → 站点管理里添加
            </button>
          </div>
        ) : shown.length === 0 ? (
          <div className="p-2 text-xs text-muted-foreground">没有匹配的环境或会话。</div>
        ) : (
          shown.map(({ env, hits }) => {
            const open = !closed[env.name] || !!query
            const list = query ? hits : (byEnv.get(env.name) || [])
            return (
              <div key={env.name} className="mb-0.5">
                <div className={cn(rowBase, 'pl-1', isEnvOn(env.name) && routeSessionEnv !== env.name ? rowOn : rowIdle)}>
                  <button
                    title={open ? '收起该环境的会话' : '展开该环境的会话'}
                    onClick={() => setClosed((c) => ({ ...c, [env.name]: !c[env.name] }))}
                    className="flex h-6 w-5 shrink-0 items-center justify-center rounded-[5px] text-muted-foreground hover:text-foreground">
                    <ChevronRight className={cn('h-3.5 w-3.5 transition-transform', open && 'rotate-90')} />
                  </button>
                  <button onClick={() => onSelectEnv(env.name)} title={env.host || env.name}
                    className="flex min-w-0 flex-1 items-center gap-2 text-left">
                    {open ? <FolderOpen className="h-4 w-4 shrink-0 text-[var(--dsh-accent)]" />
                      : <Folder className="h-4 w-4 shrink-0 text-[var(--dsh-accent)]" />}
                    <span className="min-w-0 flex-1 truncate text-foreground">{env.name}</span>
                    {!open && list.length > 0 && (
                      <span className="shrink-0 text-[10px] text-muted-foreground">{list.length}</span>
                    )}
                  </button>
                </div>

                {open && list.map((s) => {
                  const on = route.kind === 'session' && route.id === s.id
                  const st = stateOf(states, s.id)
                  return (
                    // paddingLeft 与父目录那一行的**文件夹图标**同一列:行内边距 4 + 箭头 20 +
                    // 行内 gap 8 = 32;圆点比文件夹图标窄,所以外面套一个 w-4 的定宽槽,
                    // 文字起点才对得齐(32+16+8 = 56 = 环境名那一格)。
                    <div key={s.id}
                      className={cn(rowBase, 'relative mt-0.5 pr-1.5', on ? rowOn : rowIdle)}
                      style={{ paddingLeft: 32 }}>
                      <button onClick={() => onSelectSession(s)}
                        title={`${s.name} · ${st.label}`}
                        className="flex min-w-0 flex-1 items-center gap-2 text-left">
                        <span className="flex h-4 w-4 shrink-0 items-center justify-center">
                          <span className={cn('h-2 w-2 rounded-full', st.dot)} />
                        </span>
                        <span className="min-w-0 flex-1 truncate">{s.name}</span>
                      </button>
                      {/* 悬停时显示"结束会话" —— 一行里放不下两个东西,而收会话是唯一
                          需要在这里做的破坏性动作,不该藏进右键菜单。 */}
                      <button
                        onClick={(e) => { e.stopPropagation(); onCloseSession(s) }}
                        title="结束会话（断开连接并收掉它）"
                        className="absolute right-1.5 top-1/2 hidden h-6 w-6 -translate-y-1/2 items-center justify-center rounded-[6px] text-muted-foreground transition-colors hover:bg-background/60 hover:text-foreground group-hover:flex">
                        <CircleX className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  )
                })}

                {open && list.length === 0 && (
                  <div className="py-1 pl-[32px] text-[11px] text-muted-foreground/70">还没有会话</div>
                )}
              </div>
            )
          })
        )}

        {/* TOP GP 的环境：会话池与 T100 共用（GP 调试也走 /debug/api/sessions），
            所以这里和 T100 一样按 env 挂二级会话，删除走同一个 onCloseSession。 */}
        {shownGp.map(({ env, hits }) => {
          const key = 'gp:' + env.name
          const open = !closed[key] || !!query
          const list = query ? hits : (byEnv.get(env.name) || [])
          return (
            <div key={key} className="mb-0.5">
              <div className={cn(rowBase, 'pl-1', isEnvOn(env.name) && routeSessionEnv !== env.name ? rowOn : rowIdle)}>
                <button
                  title={open ? '收起该环境的会话' : '展开该环境的会话'}
                  onClick={() => setClosed((c) => ({ ...c, [key]: !c[key] }))}
                  className="flex h-6 w-5 shrink-0 items-center justify-center rounded-[5px] text-muted-foreground hover:text-foreground">
                  <ChevronRight className={cn('h-3.5 w-3.5 transition-transform', open && 'rotate-90')} />
                </button>
                <button onClick={() => onSelectGpEnv(env.name)} title={env.host || env.name}
                  className="flex min-w-0 flex-1 items-center gap-2 text-left">
                  {open ? <FolderOpen className="h-4 w-4 shrink-0 text-[var(--dsh-accent)]" />
                    : <Folder className="h-4 w-4 shrink-0 text-[var(--dsh-accent)]" />}
                  <span className="min-w-0 flex-1 truncate text-foreground">{env.name}</span>
                  {!open && list.length > 0 && (
                    <span className="shrink-0 text-[10px] text-muted-foreground">{list.length}</span>
                  )}
                </button>
              </div>

              {open && list.map((s) => {
                const on = route.kind === 'session' && route.id === s.id
                const st = stateOf(states, s.id)
                return (
                  <div key={s.id}
                    className={cn(rowBase, 'relative mt-0.5 pr-1.5', on ? rowOn : rowIdle)}
                    style={{ paddingLeft: 32 }}>
                    <button onClick={() => onSelectSession(s)}
                      title={`${s.name} · ${st.label}`}
                      className="flex min-w-0 flex-1 items-center gap-2 text-left">
                      <span className="flex h-4 w-4 shrink-0 items-center justify-center">
                        <span className={cn('h-2 w-2 rounded-full', st.dot)} />
                      </span>
                      <span className="min-w-0 flex-1 truncate">{s.name}</span>
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); onCloseSession(s) }}
                      title="结束会话（断开连接并收掉它）"
                      className="absolute right-1.5 top-1/2 hidden h-6 w-6 -translate-y-1/2 items-center justify-center rounded-[6px] text-muted-foreground transition-colors hover:bg-background/60 hover:text-foreground group-hover:flex">
                      <CircleX className="h-3.5 w-3.5" />
                    </button>
                  </div>
                )
              })}

              {open && list.length === 0 && (
                <div className="py-1 pl-[32px] text-[11px] text-muted-foreground/70">还没有会话</div>
              )}
            </div>
          )
        })}
      </div>

      {/* 底部:设置(与 DSH 一样钉在左下角) */}
      <div className="shrink-0 border-t border-border/60 p-2">
        <button onClick={onSettings}
          className={cn(rowBase, 'px-2', route.kind === 'settings' ? rowOn : rowIdle)}>
          <Settings className="h-4 w-4 shrink-0" />
          <span className="min-w-0 flex-1 truncate">设置</span>
        </button>
      </div>
    </aside>
  )
}

function SidebarIcon({ icon: Icon, label, onClick, active }: {
  icon: typeof Search; label: string; onClick: () => void; active?: boolean
}) {
  return (
    <button title={label} onClick={onClick}
      className={cn('flex h-6 w-6 items-center justify-center rounded-[6px] transition-colors',
        active ? 'bg-accent text-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground')}>
      <Icon className="h-3.5 w-3.5" />
    </button>
  )
}

