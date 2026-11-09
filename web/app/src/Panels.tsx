// 右侧面板:运行/调试(VS Code 风格)+ 调用栈 / 变量监视 / 断点(一体化容器,分割线分区)
import * as React from 'react'
import { useEffect, useRef, useState } from 'react'
import { Eye, EyeOff, Maximize, Minimize2, Play, Ruler } from 'lucide-react'
import { useStore } from './store'
import type { Breakpoint } from './api'
import { Input } from '../../shared/ui'
import { cn } from '../../shared/utils'
import { Accordion, Checkbox } from '../../shared/ui-radix'
import { OutlineItem } from './OutlinePanel'
import { PanelContent, PanelHeader, PanelItem } from './PanelKit'
import { parseFglTree, type TNode } from './fglparse'
import { VarTreeNodes } from './VarTreeUi'
import { progKey } from './fglPath'

// 运行/调试区(VS Code Run and Debug 同款):绿色运行按钮 + 目标输入框。
// 输入作业编号或程序名(模块自动解析);也支持「模块/作业」显式指定模块。
// 会话进行中:显示当前作业编号 + 行号校准按钮(协议行号与源码偏移时手动触发)
function LaunchSection() {
  const launch = useStore((s) => s.launch)
  const launching = useStore((s) => s.launching)
  const sessionId = useStore((s) => s.sessionId)
  const prog = useStore((s) => s.prog)
  const sessions = useStore((s) => s.sessions)
  const state = useStore((s) => s.state)
  const sessionEnv = useStore((s) => s.sessionEnv)
  const calibrate = useStore((s) => s.calibrate)
  // 协作模式下启动/校准归 AI(你选的边界)
  const collab = useStore((s) => s.mode) === 'collab'
  // 没有会话时它就是个"上次输过什么"的记忆，方便反复起同一个作业。
  const [v, setV] = useState(() => localStorage.getItem('tt.launchTarget') || '')
  // 这个框**就是"当前会话的作业"**，不是全局记忆。
  // 从前这里拿全局 prog 兜底 —— 而切换会话的瞬间全局 prog 还是上一条会话的值，
  // 于是鼎捷2（aglq510）的框里写着鼎捷1 的 asf/bsft001_wf（实测反馈）。
  // 现在只认 sessions 里**这条会话自己**的 prog/module：它没起过作业就留空。
  const prevSidRef = useRef<string | null>(null)
  const editedRef = useRef(false)
  useEffect(() => {
    const switched = prevSidRef.current !== sessionId
    if (switched) { prevSidRef.current = sessionId; editedRef.current = false }
    if (!sessionId) {
      if (switched) setV(localStorage.getItem('tt.launchTarget') || '')
      return
    }
    const item = sessions.find((x) => x.id === sessionId)
    const p = item?.prog || ''
    const mod = item?.module || ''
    if (switched) {
      // 切到新会话：只认它自己的作业；没有就留空，绝不带上一条的名字。
      setV(p ? (mod ? `${mod}/${p}` : p) : '')
      return
    }
    // 同一条会话内：列表刷新（启动后 prog 落定）时跟上；用户正在改就不打扰。
    if (!editedRef.current && p) setV(mod ? `${mod}/${p}` : p)
  }, [sessionId, sessions])
  const doLaunch = () => {
    const t = v.trim()
    if (!t || launching) return
    editedRef.current = false // 启动后以服务端落定的作业为准
    localStorage.setItem('tt.launchTarget', t)
    const i = t.indexOf('/')
    if (i > 0) void launch(t.slice(0, i).trim(), t.slice(i + 1).trim())
    else void launch('', t)
  }
  // 单一常驻会话:会话空闲(idle)时仍显示启动输入,可复用宿主启动/换作业
  const inRun = !!sessionId && state !== 'idle' && state !== 'exit'
  return (
    <div className="shrink-0 border-b border-border">
      <div className="flex h-8 items-center px-2.5 text-xs font-medium text-muted-foreground">运行</div>
      {inRun ? (
        <div className="flex items-center gap-1 px-1.5 pb-1.5">
          <span
            className="min-w-0 flex-1 truncate bg-accent/40 px-1.5 py-0.5 font-mono text-xs text-foreground"
            title={prog || sessionId || ''}
          >
            {prog || (state === 'loading' ? '连接中…' : sessionEnv || '会话')}
          </span>
          <button
            title={collab ? '协作模式:行号校准由 AI 负责' : (state === 'stopped' ? '行号校准:协议行号与源码错位时点击对齐' : '行号校准(需停站后点击)')}
            disabled={collab || state !== 'stopped'}
            onClick={() => void calibrate()}
            className="p-1 transition-colors hover:bg-accent/60 disabled:pointer-events-none disabled:opacity-30"
          >
            <Ruler className="h-4 w-4 text-sky-600 dark:text-sky-400" />
          </button>
        </div>
      ) : (
        <div className="px-1.5 pb-1.5">
          {state === 'idle' && (
            <div className="pb-1 text-[11px] text-muted-foreground">
              {sessionEnv ? `会话空闲(${sessionEnv}),可直接启动调试或换作业` : '会话空闲,可直接启动调试'}
            </div>
          )}
          <div className="flex items-center gap-1">
            <button
              title="启动调试会话(Enter 同效)"
              disabled={launching || !v.trim()}
              onClick={doLaunch}
              className="p-1 transition-colors hover:bg-accent/60 disabled:pointer-events-none disabled:opacity-30"
            >
              <Play className="h-4 w-4 text-green-600 dark:text-green-500" fill="currentColor" />
            </button>
            <Input
              value={v}
              onChange={(e) => { editedRef.current = true; setV(e.target.value) }}
              onKeyDown={(e) => { if (e.key === 'Enter') doLaunch() }}
              placeholder="作业编号,如 bsft001_wf 或 asf/bsft001_wf"
              className="h-6 flex-1 px-1.5 text-xs"
            />
          </div>
        </div>
      )}
    </div>
  )
}

// 面板「自动」开关:开(实心眼)= 停站后自动抓取该面板数据(调用栈/自动变量;
// 都要经调试会话逐条发命令,全开时步进卡)。默认关(空心眼):停站不再自动调度,
// 需要时点开,立即生效并自动展开面板。
function AutoSwitch({ on, onToggle, onTip, offTip }: {
  on: boolean; onToggle: () => void; onTip: string; offTip: string
}) {
  const Icon = on ? Eye : EyeOff
  return (
    <button
      title={on ? onTip : offTip}
      aria-pressed={on}
      onClick={onToggle}
      className={`flex w-6 shrink-0 items-center justify-center transition-colors ${
        on ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground group-hover:text-foreground'
      }`}
    >
      <Icon className="h-3.5 w-3.5" />
    </button>
  )
}

export function RightPanels() {
  const stackAuto = useStore((s) => s.stackAuto)
  const autovarsAuto = useStore((s) => s.autovarsAuto)
  const toggleStackAuto = useStore((s) => s.toggleStackAuto)
  const toggleAutovarsAuto = useStore((s) => s.toggleAutovarsAuto)
  // 协作模式下自动变量开关归 AI(它会在后台逐条发 print,占命令槽)
  const collab = useStore((s) => s.mode) === 'collab'
  // 这一列里放哪些面（header 上两个开关各自独立控制）：调试面板（运行 + 四块）与大纲
  const rightDebugOpen = useStore((s) => s.rightDebugOpen)
  const rightTreeOpen = useStore((s) => s.rightTreeOpen)
  const rightTop = useStore((s) => s.rightTop)
  // 手风琴开合(受控):默认只展开「变量监视/断点」;调用栈/自动变量随自动开关
  // 联动——开则展开(数据开始自动刷新),关(默认)则收起,需要时手动点开看存量
  const [open, setOpen] = useState<string[]>(() => {
    const v = ['watches', 'bps']
    if (stackAuto) v.push('stack')
    if (autovarsAuto) v.push('autovars')
    return v
  })
  useEffect(() => {
    setOpen((o) => (stackAuto ? (o.includes('stack') ? o : [...o, 'stack']) : o.filter((x) => x !== 'stack')))
  }, [stackAuto])
  useEffect(() => {
    setOpen((o) => (autovarsAuto ? (o.includes('autovars') ? o : [...o, 'autovars']) : o.filter((x) => x !== 'autovars')))
  }, [autovarsAuto])
  // 大纲这一块：从 header 的 list-tree 开关打开时它就**展开**（用户点它就是要看树），
  // 关掉时把它的开合态一并清掉，免得下次打开还留着上一次的折叠状态、却没有视觉来历。
  useEffect(() => {
    setOpen((o) => (rightTreeOpen ? (o.includes('outline') ? o : [...o, 'outline']) : o.filter((x) => x !== 'outline')))
  }, [rightTreeOpen])

  // "后开的在下面"：两块都在时，先开的在上、后开的排在后面（用 CSS order，不换 DOM 顺序 ——
  // 换顺序会让手风琴条目**重新挂载**，大纲的折叠集合与监视输入框当场丢）。
  const treeFirst = rightTop === 'tree'
  const debugOrder = treeFirst ? 1 : 0
  const treeOrder = treeFirst ? 0 : 9

  return (
    // 一体化侧栏面板(VS Code 经典):与侧栏同底色、无外框无圆角,区块间用分割线区分。
    // 这里**不再有上下硬分屏**：两块面是同一条手风琴的条目，谁占多少高度由内容高度与
    // 展开状态决定（空间不够时一起收缩，各自内部滚动）。
    <div className="flex h-full min-h-0 w-full flex-col overflow-hidden bg-background">
      {rightDebugOpen && <LaunchSection />}
      <Accordion type="multiple" value={open} onValueChange={setOpen} className="flex min-h-0 w-full flex-col">
        {rightDebugOpen && (
        <PanelItem value="stack" style={{ order: debugOrder }}>
          <PanelHeader leading={<AutoSwitch on={stackAuto} onToggle={() => toggleStackAuto()}
            onTip="自动刷新已开启:每次停站抓取调用栈(点击关闭,减少自动调度卡顿)"
            offTip="自动刷新已关闭(默认):停站后不再抓调用栈;点击开启" />}>
            <StackTitle />
          </PanelHeader>
            <PanelContent>
              <StackBody />
            </PanelContent>
        </PanelItem>
        )}

        {rightDebugOpen && (
        <PanelItem value="autovars" style={{ order: debugOrder }}>
          <PanelHeader leading={<AutoSwitch on={collab || autovarsAuto} onToggle={() => { if (!collab) toggleAutovarsAuto() }}
            onTip="自动求值已开启:每次停站求值源码窗变量并刷新本面板(点击关闭,减少自动调度卡顿)"
            offTip={collab ? '协作模式:自动求值开关由 AI 负责' : '自动求值已关闭(默认):停站后不再求值自动变量;点击开启'} />}>
            <AutovarsTitle />
          </PanelHeader>
          <PanelContent>
            <AutovarsBody />
          </PanelContent>
        </PanelItem>
        )}

        {rightDebugOpen && (
        <PanelItem value="watches" style={{ order: debugOrder }}>
          <PanelHeader leading={<span className="block w-6 shrink-0" aria-hidden />}>
            <WatchesTitle />
          </PanelHeader>
          <PanelContent>
            <WatchesBody />
          </PanelContent>
        </PanelItem>
        )}

        {rightDebugOpen && (
        <PanelItem value="bps" style={{ order: debugOrder }}>
          <PanelHeader leading={<span className="block w-6 shrink-0" aria-hidden />}>
            <BpsTitle />
          </PanelHeader>
          <PanelContent>
            <BpsBody />
          </PanelContent>
        </PanelItem>
        )}

        {/* 大纲：与上面四块同处一条手风琴（用户 2026-10-16 要求"合并成一个手风琴组件"），
            它自带折叠 chevron；占多少高度与其它条目同一套规则（内容高度 + 空间不够时收缩）。
            外层用 display:contents 的壳承载 order —— 顺序规则与条目自身要不要重挂载解耦。 */}
        {rightTreeOpen && (
          <div style={{ order: treeOrder, display: 'contents' }}>
            <OutlineItem />
          </div>
        )}
      </Accordion>
    </div>
  )
}

// ---- 各面板标题(折叠时也始终可见,计数实时) ----

function StackTitle() {
  const n = useStore((s) => s.frames.length)
  return <span className="pl-0.5">调用栈 ({n})</span>
}

function AutovarsTitle() {
  const n = useStore((s) => s.autovars.length)
  return <span className="pl-0.5">自动变量 ({n})</span>
}

function WatchesTitle() {
  const n = useStore((s) => s.watches.length)
  return <span className="pl-0.5">变量监视 ({n})</span>
}

function BpsTitle() {
  const n = useStore((s) => s.breakpoints.length)
  return <span className="pl-0.5">断点 ({n})</span>
}

// ---- 各面板内容 ----

// 面板空态。原来这四个面板没数据时直接 `return null` —— 正文一片空白,与"这个面板坏了"
// 长得一样(时间线那边倒是有一句「暂无记录」,两处口径不一致,实测被测试者点出来过)。
// 而且要分清"还没有会话"与"有会话但此刻没有数据":前者是用户下一步该做的事,后者是等待。
function EmptyHint({ noSession, what, need }: { noSession: boolean; what: string; need: string }) {
  return (
    <div className="p-2 text-xs text-muted-foreground">
      {noSession ? `还没有调试会话：${need}` : `暂无${what}`}
    </div>
  )
}

function StackBody() {
  const frames = useStore((s) => s.frames)
  const selected = useStore((s) => s.selectedFrame)
  const selectFrame = useStore((s) => s.selectFrame)
  const stopped = useStore((s) => s.state === 'stopped')
  // 协作模式下选帧归 AI(你选的边界:只保留悬浮取值/变量/调用栈/协议流)。
  // 后果是只能看栈顶帧的变量 —— 要深入看外层帧,请在对话里让 AI 切帧。
  const collab = useStore((s) => s.mode) === 'collab'
  const canPick = stopped && !collab
  const hasSession = useStore((s) => s.sessionId)
  if (frames.length === 0) return <EmptyHint noSession={!hasSession} what="调用栈" need="停站后这里会列出调用栈" />
  return (
    <div>
      {frames.map((f) => (
        <div
          key={f.idx}
          className={`border-b border-border/60 px-2 py-1 text-xs ${canPick ? 'cursor-pointer hover:bg-accent/40' : ''} ${
            selected === f.idx ? 'bg-sky-500/10' : ''
          } ${!canPick ? 'opacity-60' : ''}`}
          title={collab ? '协作模式:选帧由 AI 负责,请在对话里委托' : (stopped ? `点击切到该帧上下文(print/locals 随之切换)` : '停站后可切换栈帧')}
          onClick={() => canPick && void selectFrame(f.idx)}
        >
          <span className="mr-1.5 text-muted-foreground">#{f.idx}</span>
          <span className="text-sky-600 dark:text-sky-400">{f.func}</span>
          <span className="ml-1.5 text-muted-foreground">{f.file}:{f.line}</span>
        </div>
      ))}
    </div>
  )
}

// 条目视图构建:RECORD/ARRAY 文本解析成树,标量保持原文本(监视/自动变量共用)
interface WatchView { expr: string; error?: string; text?: string; root?: TNode }

// 把上一份的展开态搬到新一份上(按节点名对齐)。
//
// 为什么必须搬:展开态存在节点对象上(TNode.open),而每次数据刷新都会用 buildVarView
// **重建整批对象** —— 不搬的话"停站抓一次 / print 一次,变量树就整片折回去",用户得
// 一格一格重新点开(实测)。按名字对齐是稳的:record 字段名与数组元素名([1]/[2])
// 在同一次会话里稳定;名字对不上就保持默认(收起),不猜。
function carryOpen(next: TNode, prev?: TNode): void {
  if (!prev) return
  next.open = prev.open
  if (!next.children?.length || !prev.children?.length) return
  const byName = new Map(prev.children.map((c) => [c.name, c]))
  for (const c of next.children) carryOpen(c, byName.get(c.name))
}

// 用上一批的展开态重建这一批(两个变量树面板共用:自动变量、变量监视)。
function withPrevOpen(prev: WatchView[], next: WatchView[]): WatchView[] {
  const byExpr = new Map(prev.map((v) => [v.expr, v.root]))
  for (const v of next) {
    if (v.root) carryOpen(v.root, byExpr.get(v.expr))
  }
  return next
}

function buildVarView(expr: string, value?: string, error?: string): WatchView {
  if (error) return { expr, error }
  const kids = parseFglTree(value || '')
  if (kids) {
    const record = kids.some((k) => !k.name.startsWith('['))
    return { expr, root: { name: expr, open: false, children: kids, type: record ? 'RECORD' : `ARRAY[${kids.length}]` } }
  }
  return { expr, text: value }
}

// 自动变量:停站后从当前源码窗自动提取变量并求值(只读,可一键转为监视)
function AutovarsBody() {
  const autovars = useStore((s) => s.autovars)
  const addWatch = useStore((s) => s.addWatch)
  const [views, setViews] = useState<WatchView[]>([])
  // 展开态跨刷新保留（否则每次停站抓一次就整片折回去，见 carryOpen 的说明）
  const prevViewsRef = useRef<WatchView[]>([])
  useEffect(() => {
    const next = withPrevOpen(prevViewsRef.current, autovars.map((v) => buildVarView(v.expr, v.value)))
    prevViewsRef.current = next
    setViews(next)
  }, [autovars])
  const hasSession = useStore((s) => s.sessionId)
  if (autovars.length === 0) {
    return <EmptyHint noSession={!hasSession} what="自动变量" need="停站后按「自动」抓一次当前源码窗的变量" />
  }
  return (
    <div>
      {views.map((v) => (
        <div key={v.expr} className="group flex items-start gap-1 border-b border-border/60 px-2 py-1 text-xs">
          <button
            className="shrink-0 text-muted-foreground opacity-0 transition-opacity hover:text-emerald-600 dark:hover:text-emerald-400 group-hover:opacity-100"
            title="加入变量监视"
            onClick={() => void addWatch(v.expr)}
          >
            +
          </button>
          <div className="min-w-0 flex-1">
            {v.root ? (
              <VarTreeNodes nodes={[v.root]} />
            ) : (
              <>
                <div className="text-muted-foreground">{v.expr}</div>
                <div className="whitespace-pre-wrap break-all text-emerald-600 dark:text-emerald-400">{v.text}</div>
              </>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

function WatchesBody() {
  const watches = useStore((s) => s.watches)
  const hasSession = useStore((s) => s.sessionId)
  const removeWatch = useStore((s) => s.removeWatch)
  const doPrint = useStore((s) => s.doPrint)
  const [expr, setExpr] = useState('')
  const [views, setViews] = useState<WatchView[]>([])
  // 同上：展开态跨刷新保留（每次 print 都重建对象，不搬就等于每次都被折回去）
  const prevViewsRef = useRef<WatchView[]>([])
  useEffect(() => {
    const next = withPrevOpen(prevViewsRef.current, watches.map((w) => buildVarView(w.expr, w.value, w.error)))
    prevViewsRef.current = next
    setViews(next)
  }, [watches])
  return (
    <div>
      <div className="flex gap-1 p-1.5">
        <Input
          value={expr}
          onChange={(e) => setExpr(e.target.value)}
          onKeyDown={(e) => {
            // 回车即求值:doPrint 会先 print 显示一次,再自动加入监视列表
            if (e.key === 'Enter' && expr.trim()) { void doPrint(expr.trim()); setExpr('') }
          }}
          placeholder="表达式,如 lp_str / g_qryparam.*(回车求值)"
          className="h-7 flex-1 text-xs"
        />
      </div>
      {views.length === 0 && (
        <EmptyHint noSession={!hasSession} what="监视项" need="停站后在上面输入表达式回车，即可加入监视" />
      )}
      {views.map((v) => (
        <div key={v.expr} className="flex items-start gap-1 border-b border-border/60 px-2 py-1 text-xs">
          <button className="shrink-0 text-muted-foreground hover:text-red-600 dark:hover:text-red-400" onClick={() => removeWatch(v.expr)}>×</button>
          <div className="min-w-0 flex-1">
            {v.root ? (
              <VarTreeNodes nodes={[v.root]} />
            ) : (
              <>
                <div className="text-muted-foreground">{v.expr}</div>
                <div className={`whitespace-pre-wrap break-all ${v.error ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400'}`}>
                  {v.error || v.text}
                </div>
              </>
            )}
          </div>
        </div>
      ))}
    </div>
  )
}

function BpsBody() {
  const breakpoints = useStore((s) => s.breakpoints)
  const removeBreakpoint = useStore((s) => s.removeBreakpoint)
  const toggleBPEnabled = useStore((s) => s.toggleBPEnabled)
  const jumpToBp = useStore((s) => s.jumpToBp)
  // 当前停站位置:用来标出"正停在这个断点上"。用与编辑器停站行同一个黄色(黄=停站行),
  // 一眼就能把列表条目和源码里那条高亮对上;不在停站态/文件或行号不匹配就不标。
  const stop = useStore((s) => s.stop)
  const stopped = useStore((s) => s.state === 'stopped')
  const module = useStore((s) => s.module)
  const atStop = (b: Breakpoint) =>
    stopped && !!stop?.file && !!b.file && b.line === stop.line &&
    progKey(b.file, module) === progKey(stop.file, module)
  // 断点条目的位置文本。断点可能是**注入进来**的(测试夹具、或后端某条路径没带全字段),
  // 那时 file/line 缺一个,直接拼模板会在界面上写出字面量 `undefined:undefined`(实测)。
  const bpAt = (b: Breakpoint) => (b.file ? `${b.file}:${b.line ?? '?'}` : '(这条断点没带文件/行号)')
  const hasSession = useStore((s) => s.sessionId)
  const sessionId = hasSession
  if (breakpoints.length === 0) return <EmptyHint noSession={!hasSession} what="断点" need="起一个会话后在源码行号左侧点一下即可下断点" />
  return (
    <div>
      {breakpoints.map((b) => {
        const hit = atStop(b)
        return (
          <div key={b.num}
            title={hit ? `当前停在此断点:${bpAt(b)}` : undefined}
            className={`relative flex items-center gap-2 border-b border-border/60 px-2 py-1 text-xs ${
              b.enabled ? '' : 'opacity-50'} ${hit ? 'bg-yellow-500/15' : ''}`}>
            {/* 左侧 2px 黄条:绝定位,不给该行引入布局位移(列表行不会错位) */}
            {hit && <span className="absolute inset-y-0 left-0 w-0.5 bg-yellow-500" aria-hidden />}
            <Checkbox
              checked={b.enabled}
              onCheckedChange={(v) => void toggleBPEnabled(b.num, v === true)}
              disabled={!sessionId}
              className="cursor-pointer disabled:pointer-events-none disabled:opacity-30"
              title={!sessionId
                ? '还没有调试会话：断点是会话上的东西，先起一个会话再改'
                : (b.enabled ? '取消勾选禁用断点' : '勾选启用断点')}
            />
            <span
              className="min-w-0 flex-1 cursor-pointer truncate text-foreground hover:text-sky-700 dark:hover:text-sky-300 hover:underline"
              title={`跳转到 ${bpAt(b)}`}
              onClick={() => void jumpToBp(b)}
            >
              {b.func ? <span className="text-sky-600 dark:text-sky-400">{b.func} </span> : null}
              {bpAt(b)}
              {!b.enabled && <span className="ml-1 text-[10px] text-muted-foreground">(已禁用)</span>}
            </span>
            {/* 没有会话时这两个动作会静默失败（后端无会话直接拒），界面上却看不出为什么 ——
                禁用 + 说清原因，比"点了没反应"好。 */}
            <button
              className="text-muted-foreground hover:text-red-600 dark:hover:text-red-400 disabled:pointer-events-none disabled:opacity-30"
              disabled={!sessionId}
              title={sessionId ? '删掉这条断点' : '还没有调试会话：断点是会话上的东西，先起一个会话再改'}
              onClick={() => void removeBreakpoint(b.num)}>×</button>
          </div>
        )
      })}
    </div>
  )
}

export function TimelinePanel() {
  const [tab, setTab] = useState<'timeline' | 'raw'>('raw')
  const timeline = useStore((s) => s.timeline)
  const rawLog = useStore((s) => s.rawLog)
  const sessionId = useStore((s) => s.sessionId)
  const state = useStore((s) => s.state)
  const sendRaw = useStore((s) => s.sendRaw)
  const bodyRef = useRef<HTMLDivElement | null>(null)
  const [cmd, setCmd] = useState('')
  const [hist, setHist] = useState<string[]>([])
  const [histIdx, setHistIdx] = useState(-1)
  // 最大化/还原:状态在 store 里(App 那一层要靠它决定编辑器让不让位)
  const bottomMax = useStore((s) => s.bottomMax)
  const toggleBottomMax = useStore((s) => s.toggleBottomMax)
  // 新条目到达时自动滚动到底部(最新操作始终可见)
  useEffect(() => {
    const el = bodyRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [tab, timeline.length, rawLog.length])
  const kindTone: Record<string, string> = {
    stop: 'text-yellow-600 dark:text-yellow-400', warn: 'text-red-600 dark:text-red-400', command: 'text-sky-600 dark:text-sky-400', info: 'text-muted-foreground', error: 'text-red-600 dark:text-red-400 font-semibold',
  }
  // fgldb 命令直通输入(复刻原版 fgldeb Ctrl+D 子画面);仅停站时可发。
  // 协作模式下命令归 AI(服务端会 403)—— 界面不给入口,要看什么请在对话里委托。
  const collab = useStore((s) => s.mode) === 'collab'
  const canSend = !!sessionId && state === 'stopped' && !collab
  const submit = () => {
    const c = cmd.trim()
    if (!c || !canSend) return
    setHist((h) => [...h.filter((x) => x !== c), c].slice(-50))
    setHistIdx(-1)
    setCmd('')
    void sendRaw(c)
  }
  const onCmdKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') { e.preventDefault(); submit(); return }
    // ↑/↓ 翻命令历史(最新在末尾,↓ 可回到空)
    if (e.key === 'ArrowUp' || e.key === 'ArrowDown') {
      e.preventDefault()
      if (hist.length === 0) return
      let idx = histIdx
      if (e.key === 'ArrowUp') idx = histIdx < 0 ? hist.length - 1 : Math.max(0, histIdx - 1)
      else idx = histIdx < 0 ? -1 : (histIdx + 1 >= hist.length ? -1 : histIdx + 1)
      setHistIdx(idx)
      setCmd(idx >= 0 ? hist[idx] : '')
    }
  }
  return (
    <div className="flex h-full min-h-0 flex-col bg-background">
      {/* 面板页签(VS Code PROBLEMS/OUTPUT 式):纯文字 + 活动页签主色下划线压住分割线 */}
      <div className="flex h-8 shrink-0 items-center gap-1 border-b border-border px-1.5">
        {([['raw', `原始协议流 (${rawLog.length})`], ['timeline', '操作时间线']] as const).map(([k, label]) => (
          <button key={k}
            className={`relative flex h-full items-center px-2.5 text-xs transition-colors ${
              tab === k ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'
            }`}
            onClick={() => setTab(k)}
          >
            {label}
            {tab === k && <span className="absolute inset-x-0 bottom-0 h-px bg-primary" />}
          </button>
        ))}

        {/* 最大化 / 还原:钉在页签行右端(这一行本来右边就空着)。
            两个图标按状态二选一 —— 常规分栏时给「最大化」(Maximize),占满整列时给
            「还原」(Minimize2),不做成两个并排的按钮(那样总有一个是空点)。 */}
        <button
          onClick={() => void toggleBottomMax()}
          title={bottomMax ? '还原下方面板(编辑器回到常规分栏)' : '最大化下方面板(占满代码区那一列,编辑器让位)'}
          className={`${bottomMax ? '' : 'ml-auto '}mr-0.5 shrink-0 p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground`}
        >
          {bottomMax ? <Minimize2 className="h-3.5 w-3.5" /> : <Maximize className="h-3.5 w-3.5" />}
        </button>
      </div>
      <div ref={bodyRef} className="min-h-0 flex-1 overflow-auto p-1 font-mono text-[11px] leading-5">
        {tab === 'timeline' && (
          <>
            {timeline.length === 0 && <div className="p-2 text-muted-foreground">暂无记录</div>}
            {timeline.map((t, i) => (
              <div key={i} className="flex gap-2">
                <span className="shrink-0 font-mono text-muted-foreground">{t.time}</span>
                {/* 来源标签固定成一列(w-[4.5rem] 右对齐):三种来源的宽度不一致时
                    (「用户」两字 / 「系统」两字 / 「agent」五字母),正文的起始位置会随来源左右跳,
                    一列看下来参差不齐。定宽之后时间、来源、正文各自成列。
                    「agent」不再叫「AI」:这条时间线记的是"谁执行了这一步",
                    执行者既可能是 AI 也可能是用户手动,而真正"由 AI 代理执行"的那些用 agent 更准确。 */}
                <span className={cn('w-[4.5rem] shrink-0 text-right', t.origin === 'ai' ? 'text-sky-600 dark:text-sky-400' : 'text-muted-foreground')}>
                  {t.origin === 'ai' ? 'agent' : t.origin === 'human' ? '用户' : '系统'}
                </span>
                <span className={`min-w-0 ${kindTone[t.kind] || 'text-muted-foreground'}`}>{t.text}</span>
              </div>
            ))}
          </>
        )}
        {tab === 'raw' && (
          <>
            {rawLog.map((l, i) => (
              <div key={i} className="whitespace-pre-wrap break-all text-muted-foreground">{l || ' '}</div>
            ))}
          </>
        )}
      </div>
      {tab === 'raw' && (
        // 命令输入(VS Code Debug Console 式):> 提示符 + 无边框输入行,回车即发送
        <div className="flex h-8 shrink-0 items-center gap-1.5 border-t border-border px-2.5">
          <span className="shrink-0 font-mono text-xs leading-none text-sky-600 dark:text-sky-400" title="fgldb 命令直通">&gt;</span>
          <input
            value={cmd}
            onChange={(e) => setCmd(e.target.value)}
            onKeyDown={onCmdKey}
            disabled={!canSend}
            placeholder={collab ? '协作模式:命令由 AI 执行,请在对话里委托给 AI' : (canSend ? 'fgldb 命令,如 print lp_str / info breakpoints(↑↓ 历史),回车发送' : '需停站后才能发送命令')}
            className="h-6 min-w-0 flex-1 bg-transparent font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none disabled:opacity-50"
          />
        </div>
      )}
    </div>
  )
}
