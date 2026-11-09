// 会话页底部的状态栏:整条背景色表达调试状态(VS Code 风格:停站橙 / 运行蓝 / 启动灰),
// 单行紧凑布局。协作模式下额外承载"谁在驾驶 / AI 正在执行什么 / 接管按钮"。
//
// 位置:钉在**会话页**最底下(见 SessionView),不横跨整个壳子 —— 它报的是这条会话的
// 调试状态,左侧环境树与设置页没有状态可报。
import { useEffect, useState } from 'react'
import { useStore } from './store'

/** 状态栏高度（会话页底部用；高度只在这一处定义）。 */
export const STATUS_BAR_H = 'h-7'

export function StatusBar() {
  const sourcePath = useStore((s) => s.sourcePath)
  const prog = useStore((s) => s.prog)
  const state = useStore((s) => s.state)
  const stop = useStore((s) => s.stop)
  const hold = useStore((s) => s.holdingSeconds)
  const watchdogSeconds = useStore((s) => s.watchdogSeconds)
  const watchdogDisabled = useStore((s) => s.watchdogDisabled)
  const sessionEnv = useStore((s) => s.sessionEnv)
  const sessionId = useStore((s) => s.sessionId)
  const mode = useStore((s) => s.mode)
  const inflight = useStore((s) => s.inflight)
  const setMode = useStore((s) => s.setMode)
  const collab = mode === 'collab'

  // inflight 的 elapsed 是快照那一刻的读数,长命令(continue)会停在那儿不动 ——
  // 用 since 在本地每秒重算一次,界面才是"正在跑"而不是"卡住了"。
  const [, tick] = useState(0)
  useEffect(() => {
    if (!inflight) return
    const t = window.setInterval(() => tick((n) => n + 1), 1000)
    return () => clearInterval(t)
  }, [inflight])
  const inflightSecs = inflight ? Math.max(0, Math.round((Date.now() - Date.parse(inflight.since)) / 1000)) : 0

  let bar = 'bg-background text-muted-foreground'
  let msg: string | null = null
  let title = ''
  if (state === 'stopped') {
    // 入口停站只报文件、没有行号(程序还没 run):别拼出 "x.4gl:undefined"
    const loc = stop?.file ? (stop.line ? `${stop.file}:${stop.line}` : stop.file) : ''
    const atEntry = stop?.reason === 'entry'
    // 停站时把“还有多久自动放行”直接摆在状态栏上。
    //   - 看门狗关闭:红底，说清楚“不会自动放行”（入口停站不持锁，不算危险，照常橙）；
    //   - 快到了(<60s):红底，避免正看着代码被突然放行；
    //   - 其它：橙底，带剩余秒数。
    const secs = watchdogSeconds > 0 ? watchdogSeconds : 0
    const remain = secs > 0 ? Math.max(0, Math.round(secs - hold)) : 0
    const off = watchdogDisabled && !atEntry
    const closing = !off && secs > 0 && remain <= 60
    bar = off || closing ? 'bg-[#b91c1c] text-white' : 'bg-[#cc6633] text-white'
    const tail = atEntry
      ? '入口停站 · 未持锁'
      : off
        ? '看门狗已关闭 · 不会自动放行'
        : secs > 0
          ? `还有 ${remain}s 自动放行`
          : ''
    msg = `已停站${loc ? ` ${loc}` : ''} · 停留 ${Math.round(hold)}s${tail ? ` · ${tail}` : ''}`
    title = off
      ? '看门狗已关闭:程序会一直停在这里持锁,直到人工继续/结束。改动 debug.watchdogSeconds 或环境覆盖值后重启会话生效'
      : atEntry
        ? '程序已暂停在入口(未 run,不持事务锁),可下断点或启动'
        : '程序已暂停,可查看变量/下断点/继续;超时会自动放行以防行锁'
  } else if (state === 'running') {
    bar = 'bg-[#0078d4] text-white'
    msg = collab ? 'AI 已放行 · 在界面上观察即可' : '运行中 · 请到 GDC 操作'
    title = collab ? '程序运行中;协作模式下由 AI 主导,你在界面上只能观察与取值' : '程序运行中,请在 GDC 操作作业界面'
  } else if (state === 'loading') {
    bar = 'bg-accent text-foreground'
    msg = '启动中…'
  } else if (state === 'idle') {
    // 本轮调试结束,宿主会话保留(idle):再次启动/换作业免重新登录
    bar = 'bg-foreground/10 text-foreground'
    msg = sessionEnv ? `会话空闲(${sessionEnv}) · 可直接启动调试` : '会话空闲 · 可直接启动调试'
    title = '会话保留中(SSH/登录态未断开),可直接启动新一轮调试'
  }

  const chip = 'shrink-0 rounded-sm bg-black/20 px-1.5 py-px'

  return (
    // 不画上分割线:状态栏整条自带底色(停站橙/运行蓝/空闲灰),靠色块与内容区分隔就够了;
    // 顶上加的那条线在有色状态下是多余的,在空闲态(bg-background)反而把一条淡淡的横线
    // 悬在内容与状态栏之间,像没对齐的边框。
    <div className={`flex ${STATUS_BAR_H} shrink-0 items-center gap-3 px-3 text-[11px] transition-colors ${bar}`}>
      {/* 左:源码文件路径 */}
      <span className="min-w-0 flex-1 truncate font-mono" title={sourcePath || ''}>
        {sourcePath || ' '}
      </span>
      {/* 右:协作模式标识 + 在飞命令 + 会话状态 + 当前作业 */}
      <span className="flex shrink-0 items-center gap-2">
        {collab && (
          <span className={chip} title="本次调试由 AI 主导:你只能观察与取值;需要下断点或执行命令,请在对话里委托给 AI">
            协作模式 · AI 主导
          </span>
        )}
        {inflight && (
          <span className={`${chip} font-mono`} title="正在执行的调试命令">
            ⟳ {inflight.cmd}(已 {inflightSecs}s)
          </span>
        )}
        {sessionId && (
          <button
            className="shrink-0 rounded-sm border border-current/40 px-1.5 py-px opacity-80 hover:bg-black/10 hover:opacity-100"
            onClick={() => void setMode(collab ? 'solo' : 'collab')}
            title={collab ? '切回纯人工:界面恢复全部写操作' : '把本次调试交给 AI 主导:你转为只读观察'}
          >
            {collab ? '接管' : '交给 AI'}
          </button>
        )}
        {msg && <span title={title}>{msg}</span>}
        {state && prog && <span className="font-medium" title={`作业 ${prog}`}>{prog}</span>}
      </span>
    </div>
  )
}
