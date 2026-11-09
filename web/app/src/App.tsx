// 工作站外壳:左边是「环境 → 会话」两级目录,右边是所选目标的面板(带 Tab),底部状态栏。
//
// 布局是第一阶段的产出(用户 2026-10-17);**会话那一半已经是真的了**(2026-10-18):
// 树上每一条都是服务端会话句柄,在环境页建出来(建空闲会话 → 等登录 → 挂画面通道/proxy)。
// 旧外壳的那两条图标栏(左侧设置/会话、右侧四个视图)已经撤掉 —— 视图入口改成
// "环境页/会话页里的 Tab",设置挪到左下角。
//
// 路由是本文件唯一管 URL 的地方:状态在 `route`,片段由 `routeHash` 生成;浏览器后退/粘链接
// 走 `parseRoute` 回到同一个目标。两边都要做,否则"界面在这一页、地址栏还是上一页"。
import { useEffect, useMemo, useRef, useState } from 'react'
import { PanelLeft, RotateCcw, WifiOff, X } from 'lucide-react'
import { connectWS, useStore, type SessionDraft, type SessionItem } from './store'
import { refreshGpEnvs } from './gpEnvs'
import { Sidebar } from './Sidebar'
import { EnvView } from './EnvView'
import { SessionView } from './SessionView'
import { SettingsView } from './SettingsView'
import { IconBtn, Pane } from './Shell'
import { Dialog, DialogContent, DialogTitle } from '../../shared/ui-radix'
import { api } from './api'
import { parseRoute, routeHash, routeView, type EnvTab, type Route, type SessionTab } from './routing'
import { ErrorBoundary } from './ErrorBoundary'

// 浏览器标签页标题反馈:停站时标题闪烁(仅后台),运行/启动改前缀。
// 停站标题在「⚠ 停站 file:line」与原名之间每秒交替,后台标签会被浏览器高亮提醒。
function useDocTitleBlink() {
  const sessionId = useStore((s) => s.sessionId)
  const state = useStore((s) => s.state)
  const file = useStore((s) => s.stop?.file)
  const line = useStore((s) => s.stop?.line)
  useEffect(() => {
    const base = 'TT'
    const setT = (t: string) => { document.title = t }
    if (!sessionId) { setT(base); return }
    let timer: number | undefined
    if (state === 'stopped') {
      const loc = file && line ? `${file}:${line}` : '已停站'
      const flashA = `⚠ 停站 ${loc}`
      setT(flashA)
      let toggle = true
      const onVis = () => {
        if (document.hidden) {
          toggle = true
          timer = window.setInterval(() => {
            document.title = toggle ? base : flashA
            toggle = !toggle
          }, 1000)
        } else {
          if (timer) { clearInterval(timer); timer = undefined }
          setT(flashA)
        }
      }
      document.addEventListener('visibilitychange', onVis)
      if (document.hidden) onVis()
      return () => {
        document.removeEventListener('visibilitychange', onVis)
        if (timer) clearInterval(timer)
        setT(base)
      }
    }
    if (state === 'running') setT(`▶ 运行中 · ${base}`)
    else if (state === 'loading') setT(`⏳ 启动中 · ${base}`)
    else setT(base)
  }, [sessionId, state, file, line])
}

const SIDEBAR_KEY = 'tt.sidebarCollapsed'

export function App() {
  useDocTitleBlink()

  const envs = useStore((s) => s.envs)
  const gpEnvs = useStore((s) => s.gpEnvs)
  const envsLoaded = useStore((s) => s.envsLoaded)
  const envsErr = useStore((s) => s.envsErr)
  const loadEnvs = useStore((s) => s.loadEnvs)
  const sessions = useStore((s) => s.sessions)
  const createSession = useStore((s) => s.createSession)
  const closeSession = useStore((s) => s.closeSession)
  const sessionList = useStore((s) => s.sessionList)

  // 树上一眼要看到的那件事:这条会话现在能不能用。状态只从服务端列表来(唯一的权威),
  // 不在树里另存一份 —— 两份必然漂(树的条目是本地缓存,会话状态是活的)。
  // 绑着的那一条用 store 里的实时状态盖一遍:WS 事件比列表刷新快,否则"运行中→停站"
  // 在树上要等到下次列表刷新才变色。
  const boundId = useStore((s) => s.sessionId)
  const boundState = useStore((s) => s.state)
  const runtimes = useStore((s) => s.runtimes)
  const sessionsLoaded = useStore((s) => s.sessionsLoaded)
  const sessionStates = useMemo(() => {
    const m: Record<string, string> = {}
    for (const b of sessionList) m[b.id] = b.state
    // 运行时先于列表：事件比列表新（列表是"刚才那一刻"，事件是"刚刚发生"）。
    for (const [id, r] of Object.entries(runtimes)) if (r.state) m[id] = r.state
    if (boundId && boundState) m[boundId] = boundState
    return m
  }, [sessionList, runtimes, boundId, boundState])

  const settingsSection = useStore((s) => s.settingsSection)


  const backendDead = useStore((s) => s.backendDead)
  const wsConnected = useStore((s) => s.wsConnected)
  const launching = useStore((s) => s.launching)
  const sessionId = useStore((s) => s.sessionId)
  const restart = useStore((s) => s.restart)

  // 侧栏收展:与旧外壳的 tt.sessionOpen 同一档持久化(纯界面状态)。
  const [collapsed, setCollapsed] = useState(() => localStorage.getItem(SIDEBAR_KEY) === '1')
  const toggleCollapsed = () => setCollapsed((v) => {
    localStorage.setItem(SIDEBAR_KEY, v ? '0' : '1')
    return !v
  })

  const [route, setRoute] = useState<Route>(() => parseRoute(location.hash))

  // 设置是**弹窗**(2026-10-19 用户:单独提取成一个 pop 页面),不是主面板的一屏:
  // 打开时下面的环境页/会话页留在遮罩后面,关掉就回到原来那一屏。
  // 片段仍然是 `#settings/<分区>`(深链接、前进后退都不变),所以这里另外记一份
  // "最后一个真正的主路由" —— 它只活在内存里:冷启动直接开 `#settings` 时没有它,
  // 关掉就落到第一个环境的「创建会话」(一个环境都没有时只能留在设置里)。
  const lastMainRef = useRef<Route | null>(null)
  useEffect(() => {
    if (route.kind === 'env' || route.kind === 'session') lastMainRef.current = route
  }, [route])
  // 主面板该显示哪一屏:设置打开时**不换屏**,继续显示上一次的主路由,只是上面多一层弹窗。
  // 冷启动直接开 `#settings` 时没有上一屏 —— 那就用默认落点(第一个环境的创建页)当背景,
  // 不然遮罩后面是一片空黑,关掉弹窗还会先闪一下空白。
  const mainRoute: Route = (route.kind === 'settings'
    ? lastMainRef.current ?? (envs[0]?.name ? { kind: 'env', env: envs[0].name, tab: 'create' } : null)
    : null) ?? route

  // 首次挂载:认深链接(设置分区)、拉环境清单、连 WS、接管已有会话。
  // 先动 URL 那条路:深链接里的设置分区要落进 store,后面的默认路由才不会覆盖它。
  useEffect(() => {
    const r = parseRoute(location.hash)
    const st = useStore.getState()
    if (r.kind === 'settings' && r.section) st.setSettingsSection(r.section)
    void loadEnvs()
      // GP 的环境和服务清单一起拉，别等用户点进设置页
      void refreshGpEnvs()
    const close = connectWS()
    void api.status().catch(() => {})
    void st.adoptExisting()
    // 环境清单只在设置页改得动;窗口回到前台时补一次,省得让用户刷新整页。
    const onFocus = () => void useStore.getState().loadEnvs()
    window.addEventListener('focus', onFocus)

    // 调试快捷键:F5 继续 / F10 步过 / F11 步入 / Shift+F11 步出
    const onKey = (e: KeyboardEvent) => {
      const s = useStore.getState()
      if (!s.sessionId) return
      const stopped = s.state === 'stopped'
      if (e.key === 'F5') { e.preventDefault(); if (stopped) void s.control('continue') }
      else if (e.key === 'F10') { e.preventDefault(); if (stopped) void s.control('next') }
      else if (e.key === 'F11') {
        e.preventDefault()
        if (stopped) void s.control(e.shiftKey ? 'finish' : 'step')
      }
    }
    window.addEventListener('keydown', onKey)
    return () => {
      window.removeEventListener('focus', onFocus)
      window.removeEventListener('keydown', onKey)
      close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // 浏览器前进/后退、或从别处再点一次同样的深链接:跟着片段走。
  useEffect(() => {
    const onHash = () => {
      const r = parseRoute(location.hash)
      const st = useStore.getState()
      if (r.kind === 'settings' && r.section) st.setSettingsSection(r.section)
      setRoute(r)
    }
    window.addEventListener('hashchange', onHash)
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  // 默认落点与失效目标:
  //   · 片段为空(首次打开)→ 第一个环境的「创建会话」;一个环境都没有 → 设置页(先配环境)。
  //   · 片段指着一个已经不在的环境/会话(改过配置、清过 localStorage)→ 回第一个环境,
  //     而不是把人留在"界面认得这个 ID、数据里没有它"的空壳上。
  useEffect(() => {
    if (!envsLoaded) return
    const first = envs[0]?.name
    const fallback = (): Route => first ? { kind: 'env', env: first, tab: 'create' } : { kind: 'settings' }
    if (route.kind === 'home') { setRoute(fallback()); return }
    if (route.kind === 'env' && !envs.some((e) => e.name === route.env) && !gpEnvs.some((e) => e.name === route.env)) { setRoute(fallback()); return }
    // 会话页要等**池列过至少一次**再判失效：树是池的投影（W-P1），首帧 `sessions` 还是空的，
    // 不 gate 的话"深链接到某条会话"会在数据回来之前就被当成失效、弹回环境页。
    if (route.kind === 'session' && sessionsLoaded && !sessions.some((s) => s.id === route.id)) setRoute(fallback())
  }, [envsLoaded, envs, sessions, sessionsLoaded, route])

  // 路由 → store.view:接口日志、设置页那些"进来就重拉一次"的 effect 靠它判断
  // 自己是不是此刻的前台。键的映射只有一份(见 routing.ts 的 routeView)。
  useEffect(() => { useStore.getState().setView(routeView(route)) }, [route])

  // 路由 → 面板绑定的那条会话。**这一步是"不会用错会话"的前端那一半**：
  // 地址栏说的是哪条，面板就必须绑哪条。不绑的话，从树上点进 B（或刚建成 B 跳过来），
  // 编辑器/变量/断点还停在上一条 A 上 —— 而且界面上每一处都自洽，看不出错。
  // `selectSession` 仍是唯一的绑定入口（Q4：选中态只能由人点或本页自己发起的动作写）。
  //
  // **同一个 id 在这一次页面生命周期里只对齐一次**，但"一次"必须真的发生：
  // 从前这里写的是 `if (st.sessionId === route.id) return` —— 刷新页面时 store 从
  // localStorage 里已经绑着同一条，于是**对齐被整个跳过**，快照（state/stop/源码）
  // 一条都没拉，界面就永远停在「启动中… / 连接中…」。实测 2026-10-19。
  const alignedRef = useRef<string | null>(null)
  useEffect(() => {
    if (route.kind !== 'session') { alignedRef.current = null; return }
    if (alignedRef.current === route.id) return
    alignedRef.current = route.id
    void useStore.getState().selectSession(route.id)
  }, [route])

  // 从设置页离开时重读一次环境清单:环境名/增删都只在那里改得动,不重读的话左侧树会
  // 停在旧名字上,而点它又会被判成"环境不存在"回退到第一个环境。
  const prevKindRef = useRef(route.kind)
  useEffect(() => {
    if (prevKindRef.current === 'settings' && route.kind !== 'settings') void loadEnvs()
    prevKindRef.current = route.kind
  }, [route.kind, loadEnvs])

  // 当前环境:环境页与左树按它过滤。
  useEffect(() => {
    const st = useStore.getState()
    if (route.kind === 'env') st.setActiveEnv(route.env)
    else if (route.kind === 'session') {
      const s = st.sessions.find((x) => x.id === route.id)
      if (s) st.setActiveEnv(s.env)
    }
  }, [route, sessions])


  // 路由 → 片段。首跑**让一次**:片段已经指着某个目标时,URL 是权威(深链接),
  // 这一刻的 route 就是它解析出来的,照写等于把它规范化掉,不写反而一个字节都不改。
  // 往后每换一次才写,且用 replace:push 会让浏览器后退要一格一格走遍所有 Tab。
  const hashWroteRef = useRef(false)
  useEffect(() => {
    if (!hashWroteRef.current) {
      hashWroteRef.current = true
      if (parseRoute(window.location.hash).kind !== 'home') return
    }
    // 还没选到目标(首次打开、环境清单在读):先把地址栏留着,别写一个 `#` 上去。
    if (route.kind === 'home') return
    const want = routeHash(route, settingsSection)
    if (window.location.hash !== want) window.history.replaceState(null, '', want)
  }, [route, settingsSection])

  // 面板显隐:环境页/会话页两块都 keep-alive(第一次显示后只改 display)。
  // 切走再切回时 Monaco 编辑器、日志详情的滚动与输入都还在。
  // 注意读的是 `mainRoute` 而不是 `route`:设置弹窗打开时这两屏要**继续显示**在遮罩下面,
  // 否则关掉弹窗的瞬间会看到一片空白、再闪回内容。
  const lastEnvRef = useRef<{ name: string; tab: EnvTab } | null>(null)
  if (mainRoute.kind === 'env') lastEnvRef.current = { name: mainRoute.env, tab: mainRoute.tab }
  const lastSessionRef = useRef<{ id: string; tab: SessionTab } | null>(null)
  if (mainRoute.kind === 'session') lastSessionRef.current = { id: mainRoute.id, tab: mainRoute.tab }

  const envShown = lastEnvRef.current
  const envBrief = envShown ? (envs.find((e) => e.name === envShown.name) || { name: envShown.name }) : null
  const sessionShown = lastSessionRef.current
  const sessionItem = sessionShown ? sessions.find((s) => s.id === sessionShown.id) || null : null

  // ---- 导航 ----
  const openEnv = (name: string) => setRoute({ kind: 'env', env: name, tab: 'create' })
  // GP 环境与 T100 一样进环境页（创建会话 / 接口日志）。
  // 真正的 GP 会话要等调试那一块也搬进 Go；在那之前这一页是同一个壳。
  const openGpEnv = (name: string) => setRoute({ kind: 'env', env: name, tab: 'create' })
  // 会话页只有一屏：Debug。
  const openSession = (s: SessionItem) => setRoute({ kind: 'session', id: s.id, tab: 'debug' })
  const goSettings = () => setRoute({ kind: 'settings' })
  // 关闭设置弹窗:回到打开它之前那一屏(内存里那份主路由);冷启动直接开 `#settings`
  // 时没有上一屏,就落到第一个环境的创建页 —— 一个环境都没有时无处可去(默认落点会
  // 立刻再把设置打开),那就只能留着它。
  const closeSettings = () => {
    if (lastMainRef.current) { setRoute(lastMainRef.current); return }
    const first = envs[0]?.name
    if (first) setRoute({ kind: 'env', env: first, tab: 'create' })
  }
  const goAddEnv = () => {
    useStore.getState().setSettingsSection('sites')
    setRoute({ kind: 'settings' })
  }
  // 「新会话」= 回到所属环境的「创建会话」Tab(那儿才是真正填写名字/选类型的地方)。
  const newSession = () => {
    // 左侧清单的顺序是 T100 在前、GP 在后；没有选中时回落到这个清单的第一台。
    // 只有 GP 时第一台就是 GP 环境（这是 GP 并入后的显式产品规则，T100 无此场景）。
    const target = mainRoute.kind === 'env' ? mainRoute.env
      : mainRoute.kind === 'session' ? sessions.find((s) => s.id === mainRoute.id)?.env
      : (envs[0]?.name || gpEnvs[0]?.name)
    if (!target) { goAddEnv(); return }
    setRoute({ kind: 'env', env: target, tab: 'create' })
  }
  const createUnder = async (envName: string, form: SessionDraft, onStep: (s: string) => void) => {
    // 建会话是**真**动作（建空闲会话 → 等登录 → 挂画面通道），失败原因由 store 抛上来，
    // 由环境页那张卡片原样显示；失败时**不跳走**（跳到一条没建成的会话上最误导）。
    const s = await createSession({ ...form, env: envName }, onStep)
    setRoute({ kind: 'session', id: s.id, tab: 'debug' })
  }

  const banner = backendDead || (sessionId && !wsConnected ? '服务连接断开,自动重连中…' : '')
  // 侧栏收起之后,展开入口跟着标题行走(否则用户没有地方把它叫回来)。
  const leading = collapsed
    ? <IconBtn icon={PanelLeft} label="展开左侧目录" onClick={toggleCollapsed} />
    : undefined

  return (
    <div className="dsh-shell flex h-screen flex-col overflow-hidden bg-background text-foreground">
      {banner && (
        <div className="flex shrink-0 items-center gap-2 border-b border-red-500/20 bg-red-500/10 px-3 py-1.5 text-xs text-red-600 dark:text-red-400">
          <WifiOff className="h-3.5 w-3.5" />
          <span>{banner}</span>
          {backendDead && (
            <button
              className="ml-auto inline-flex items-center gap-1 border border-red-500/30 px-2 py-0.5 hover:bg-red-500/10 disabled:opacity-40"
              disabled={launching}
              onClick={() => void restart()}
            >
              <RotateCcw className="h-3 w-3" />
              {launching ? '重启中…' : '重新启动'}
            </button>
          )}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        {!collapsed && (
          <ErrorBoundary what="左侧目录">
            <Sidebar
              envs={envs} gpEnvs={gpEnvs} envsLoaded={envsLoaded} envsErr={envsErr} sessions={sessions}
              states={sessionStates} route={route}
              onSelectEnv={openEnv} onSelectGpEnv={openGpEnv} onSelectSession={openSession}
              // 失败已经在 store 里 toast 过,这里只吞掉 Promise 的拒绝(不吞掉状态变化)
              onCloseSession={(s) => { void closeSession(s.id).catch(() => {}) }}
              onNewSession={newSession} onSettings={goSettings}
              onRefreshEnvs={() => void loadEnvs()} onAddEnv={goAddEnv}
              onCollapse={toggleCollapsed}
            />
          </ErrorBoundary>
        )}

        <main className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          {/* 环境页 */}
          {envBrief && envShown && (
            <Pane show={mainRoute.kind === 'env'} label="环境">
              <ErrorBoundary what={`环境「${envBrief.name}」`}>
                <EnvView
                  env={envBrief}
                  tab={envShown.tab}
                  sessions={sessions.filter((s) => s.env === envBrief.name)}
                  onTab={(t) => setRoute({ kind: 'env', env: envBrief.name, tab: t })}
                  onCreate={(form, onStep) => createUnder(envBrief.name, form, onStep)}
                  leading={leading}
                />
              </ErrorBoundary>
            </Pane>
          )}

          {/* 会话页 */}
          {sessionItem && sessionShown && (
            <Pane show={mainRoute.kind === 'session'} label="会话">
              <ErrorBoundary what={`会话「${sessionItem.name}」`}>
                <SessionView
                  session={sessionItem}
                  tab={sessionShown.tab}
                  onTab={(t) => setRoute({ kind: 'session', id: sessionItem.id, tab: t })}
                  onOpenEnv={() => openEnv(sessionItem.env)}
                  leading={leading}
                />
              </ErrorBoundary>
            </Pane>
          )}

          {/* 首屏兜底:环境清单还在读,或一个环境都没有(随后会跳设置弹窗) */}
          {mainRoute.kind === 'home' && (
            <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 p-8 text-center text-xs text-muted-foreground">
              {envsErr ? (
                <>
                  <div className="text-red-600 dark:text-red-400">读取环境清单失败</div>
                  <div className="max-w-xl break-all">{envsErr}</div>
                  <button onClick={() => void loadEnvs()}
                    className="rounded-[7px] border border-border px-2 py-1 text-foreground hover:bg-accent">
                    重试
                  </button>
                </>
              ) : (
                <div>{envsLoaded ? '还没有环境,先去设置里添加一个。' : '正在读取环境…'}</div>
              )}
            </div>
          )}
        </main>
      </div>

      {/* 设置:弹窗(不是一屏)。遮罩后面留着刚才那一页 —— 关掉就回去,不用再猜"我是从哪进来的"。
          尺寸只给一个区间:宽 min(1120px, 94vw)、高 min(860px, 88vh);内容自己滚,
          弹窗本身上不出现第二条滚动条(内层 SettingsView 左右两栏各自滚)。 */}
      <Dialog open={route.kind === 'settings'} onOpenChange={(o) => { if (!o) closeSettings() }}>
        <DialogContent
          aria-describedby={undefined}
          // 弹窗自己要能接住焦点:没有 tabIndex 的 div 上 focus() 是空操作,
          // 浏览器就把焦点还给了第一个可聚焦元素(X)——聚焦框就画在关闭按钮上了。
          tabIndex={-1}
          // 打开时把焦点落在弹窗本身上,而不是第一个可聚焦元素(右上角那个 X):
          // Radix 默认会聚焦它,于是每开一次设置都有一圈聚焦框挂在关闭按钮上。
          // 弹窗带 outline-none,焦点留在里面(键盘可达)但不多画一笔。
          onOpenAutoFocus={(e) => { e.preventDefault(); (e.currentTarget as HTMLElement | null)?.focus?.() }}
          className="flex h-[min(860px,88vh)] max-h-[88vh] w-[min(1120px,94vw)] max-w-none flex-col gap-0 overflow-hidden rounded-[12px] p-0 shadow-2xl"
        >
          <div className="flex h-12 shrink-0 items-center gap-3 border-b border-border px-4">
            <DialogTitle className="text-[15px] font-semibold tracking-tight">设置</DialogTitle>
            <span className="min-w-0 truncate text-[11px] text-muted-foreground">
              环境 / 数据字典 / 调试参数 / 应用
            </span>
            <button
              onClick={closeSettings}
              aria-label="关闭设置"
              title="关闭设置（Esc）"
              className="ml-auto rounded-ui p-1 text-muted-foreground transition-colors hover:bg-accent/60 hover:text-foreground focus:outline-none focus-visible:ring-1 focus-visible:ring-foreground/40"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="min-h-0 min-w-0 flex-1 overflow-hidden">
            <SettingsView />
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
