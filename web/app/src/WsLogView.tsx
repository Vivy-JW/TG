// 接口日志视图:Chrome DevTools Network 风格。
// 默认只有日志列表(全宽);点击某一行才在右侧"嵌入"详情面板(基本信息/Request/Response + 重放调试),
// 此时列表收起为「状态 + 服务」两列给面板让位 —— 详情面板可拖拽分配宽度,也可主动关闭(✕ / Esc)。
// 查询条件对齐 awsq990 主查询 QBE:服务名(wsfa001)+ 开始时间范围(wsfa003);仅失败为本工具扩展
import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { Bug, Check, ChevronDown, ChevronLeft, ChevronRight, ChevronsUpDown, ChevronUp, Copy, Fingerprint, RefreshCw, Undo2, X } from 'lucide-react'
import { useStore } from './store'
import { Button, Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '../../shared/ui'
import { Checkbox, DatePicker, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../shared/ui-radix'
import type { WSLogItem } from './api'
import { PayloadEditor } from './PayloadEditor'
import { copyText } from './clipboard'
import { useToast } from './toast'

// 每个环境各记一份"这一页的输入"（查询条件/日期/企业/是否已查询）。
// 与 store 里的 wsLogsByEnv（结果）配套：结果 + 输入都按环境分开，切回来才是原样。
type WsLogViewState = {
  draft: { service: string; job: string; result: string; origin: string; server: string }
  applied: { service: string; job: string; result: string; origin: string; server: string }
  onlyFail: boolean
  from: string
  to: string
  ent: number
  queried: boolean
}
const wsLogViewStateByEnv = new Map<string, WsLogViewState>()

// 列表列定义:完整模式(未选中)与紧凑模式(详情展开,只留状态/服务)共用同一套渲染。
// 每个列都给出排序键,表头点击即排序。注意:排序只作用于**当前页**——分页在后端做,
// 未排序时的顺序就是服务端的 `order by wsfa003 desc`(最新在前)。
// 列集对齐 T100 原生页 awsq990.4gl(整合服務端檢測工具,见 debug/wslog.go 注释):
// 它显示 起始时间/发起端/服务端/服务名称/服务程序序号/状态/处理时间/错误消息/sso秒数 九列,
// 本工具此前只展示其中一部分,现补齐「服务程序序号 / 发起端 / 服务端 / sso时间」四项。
type SortKey = 'status' | 'service' | 'job' | 'pid' | 'start' | 'end' | 'dur' | 'sso' | 'origin' | 'server' | 'err'
type Col = {
  key: SortKey
  head: string
  cls: string // table-fixed 下的列宽;留空 = 吃掉剩余宽度(每套列里只应有一个)
  render: (it: WSLogItem) => ReactNode
  sortValue: (it: WSLogItem) => string | number
}

// 数值列排序:wsfa005/wsfa015 在库里是文本/数值混着取回的字符串,按数值比才对
const numSort = (s: string): number => {
  const n = Number.parseFloat(s)
  return Number.isFinite(n) ? n : Number.NaN
}

const COL_STATUS: Col = {
  key: 'status', head: '状态', cls: 'w-14',
  sortValue: (it) => it.code,
  render: (it) => (
    <span className={`font-mono text-[11px] ${it.code === '000' ? 'text-emerald-600 dark:text-emerald-500' : it.code ? 'text-red-500' : 'text-muted-foreground'}`}>
      {it.code || '-'}
    </span>
  ),
}
const COL_SERVICE: Col = {
  key: 'service', head: '服务', cls: 'w-40',
  sortValue: (it) => it.service,
  render: (it) => <span className="block truncate text-foreground" title={it.service}>{it.service}</span>,
}
const COL_JOB: Col = {
  key: 'job', head: '作业', cls: 'w-28',
  sortValue: (it) => it.job,
  render: (it) => <span className="block truncate text-sky-600 dark:text-sky-400" title={it.job}>{it.job}</span>,
}
const COL_PID: Col = {
  key: 'pid', head: '服务程序序号', cls: 'w-24',
  sortValue: (it) => numSort(it.pid),
  render: (it) => <span className="block truncate font-mono text-[11px] text-muted-foreground" title={it.pid}>{it.pid}</span>,
}
const COL_START: Col = {
  key: 'start', head: '开始时间', cls: 'w-36',
  // wsfa003 取出即 "yyyy-mm-dd hh:mm:ss",字典序 == 时间序(后端 SQL 也是这么比的)
  sortValue: (it) => it.start,
  render: (it) => <span className="font-mono text-[11px] text-muted-foreground">{it.start}</span>,
}
const COL_END: Col = {
  key: 'end', head: '结束时间', cls: 'w-36',
  // wsfa004(原生页取而不显的一列);列表里截到秒,毫秒位只参与后端时间窗过滤
  sortValue: (it) => it.end,
  render: (it) => <span className="font-mono text-[11px] text-muted-foreground">{it.end}</span>,
}
const COL_DUR: Col = {
  key: 'dur', head: '耗时(s)', cls: 'w-20',
  // wsfa005 是文本列,按数值排(空/非法值恒排最后)
  sortValue: (it) => numSort(it.duration),
  render: (it) => <span className="font-mono text-[11px] text-muted-foreground">{it.duration}</span>,
}
const COL_SSO: Col = {
  key: 'sso', head: 'sso时间(秒)', cls: 'w-24',
  sortValue: (it) => numSort(it.sso),
  render: (it) => <span className="block truncate font-mono text-[11px] text-muted-foreground" title={it.sso}>{it.sso}</span>,
}
const COL_ORIGIN: Col = {
  key: 'origin', head: '发起端', cls: 'w-24',
  sortValue: (it) => it.origin,
  render: (it) => <span className="block truncate text-muted-foreground" title={it.origin}>{it.origin}</span>,
}
const COL_SERVER: Col = {
  key: 'server', head: '服务端', cls: 'w-24',
  sortValue: (it) => it.server,
  render: (it) => <span className="block truncate text-muted-foreground" title={it.server}>{it.server}</span>,
}
const COL_ERR: Col = {
  key: 'err', head: '错误描述', cls: '',
  sortValue: (it) => it.errMsg,
  render: (it) => <span className="block truncate text-red-600 dark:text-red-400/90" title={it.errMsg}>{it.errMsg}</span>,
}
const COLS_FULL: Col[] = [
  COL_STATUS, COL_SERVICE, COL_JOB, COL_PID, COL_START, COL_END, COL_DUR, COL_SSO, COL_ORIGIN, COL_SERVER, COL_ERR,
]
// 详情展开时:只留「状态 + 服务 + 作业 + 开始时间」(其余列在右侧详情「基本信息」里都有)。
//
// 服务列要**给它一个下限宽度**,不能让四个列都吃"剩余空间":详情面板拖到最宽时列表只剩
// ~210px,而前三列各有固定宽度 —— 服务列分到的剩余宽度是负的,`table-fixed` 就把它压成 0,
// 整列(含表头)当场消失,只剩一个横向滚动条(实测)。给个 w-32 + 让容器横向滚动,
// 至少在窄列表里"能滚到看见它",而不是列没了。
const COLS_COMPACT: Col[] = [COL_STATUS, { ...COL_SERVICE, cls: 'w-32' }, COL_JOB, COL_START]

// 左右分栏按「比例」分配(默认五五分):拖拽只改比例,容器再窄也有可拖区间;
// 窗口缩放时两栏等比跟随,不会像固定像素那样一边被夹死、拖不动
const RATIO_MIN = 0.25 // 详情面板最小占比
const RATIO_MAX = 0.75 // 详情面板最大占比
const RESIZER_W = 1    // .col-resizer 的宽度(见 index.css:1px 线 + 透明溢出命中区)
// 两侧各要多少才"看得成"：
//   MIN_DETAIL_W —— 低于这个总宽就不再左右分栏（见 narrowSplit）
//   MIN_DETAIL_ABS —— 分栏时明细的**绝对**下限：占比下限 0.25 在 690px 的容器上只有 172px，
//     值列立刻变成一列汉字的宽度（用户实测截图里"服务名称 oa.sc he m.a dat a.get"就是它）
const MIN_LIST_W = 280
const MIN_DETAIL_W = 300
const MIN_DETAIL_ABS = 240

// 工具条查询输入框(四个条件共用外观;服务名称放通配,给宽一点)
// rounded-ui:全站唯一那一档圆角,见 tokens.css 的圆角口径
const QUERY_INPUT = 'h-7 w-52 rounded-ui border border-border bg-background px-2 text-xs text-foreground placeholder:text-muted-foreground focus:border-border focus:outline-none'
const QUERY_INPUT_SM = 'h-7 w-36 rounded-ui border border-border bg-background px-2 text-xs text-foreground placeholder:text-muted-foreground focus:border-border focus:outline-none'

// 一个查询条件 = **字段名(左) + 输入框(右)**,两件东西分开。
//
// 早先这里把字段名塞进 placeholder(「服务端(如 T100)」),于是"说明"与"能打字的地方"挤在
// 同一个框里 —— 分不清哪里是标签、哪里是输入(2026-10-15 用户指出)。
// 现在 placeholder 只说"能填什么"(例子/通配说明),完整语义(字段号、匹配方式)在 title 上。
//
// 定义在模块级、不是组件内部:内部定义会让 React 每次渲染都当成新组件而重挂,
// 输入一个字就丢焦点。
function QueryField({ label, title, placeholder, value, onChange, onKeyDown, wide }: {
  label: string; title: string; placeholder: string; value: string
  onChange: (v: string) => void; onKeyDown: (e: React.KeyboardEvent) => void; wide?: boolean
}) {
  return (
    <div className="flex items-center gap-1.5" title={title}>
      <span className="shrink-0 text-xs text-muted-foreground">{label}</span>
      <input
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        className={wide ? QUERY_INPUT : QUERY_INPUT_SM}
      />
    </div>
  )
}

export function WsLogView() {
  const toast = useToast()
  const wsLogs = useStore((s) => s.wsLogs)
  const loading = useStore((s) => s.wsLogsLoading)
  const page = useStore((s) => s.wsLogsPage)
  const hasMore = useStore((s) => s.wsLogsHasMore)
  const sel = useStore((s) => s.wsLogSel)
  const content = useStore((s) => s.wsLogContent)
  const tab = useStore((s) => s.wsLogTab)
  const err = useStore((s) => s.wsLogErr)
  const view = useStore((s) => s.view)
  const loadWsLogs = useStore((s) => s.loadWsLogs)
  const selectWsLog = useStore((s) => s.selectWsLog)
  const prefetchWsLog = useStore((s) => s.prefetchWsLog)
  const closeWsLogDetail = useStore((s) => s.closeWsLogDetail)
  const setWsLogTab = useStore((s) => s.setWsLogTab)
  const replayDebug = useStore((s) => s.replayDebug)
  const sessionId = useStore((s) => s.sessionId)
  const state = useStore((s) => s.state)
  // 宿主会话常驻:结束调试后会话仍空闲保留,不算"进行中";只有真在跑/停着/启动
  // 才算忙碌。重放按钮不设禁用——后端重放会先自动收口现有会话(见 wslog 重放接口)
  const sessionBusy = !!sessionId && state !== 'idle' && state !== 'exit' && state !== ''
  // 顶部查询条件:四个输入框存 draft(正在编辑),回车/刷新才写入 applied 并查询;
  // 日期/仅失败仍即时生效(沿用原有交互)。
  // 条件语义对齐 T100 原生页 awsq990:服务名称 wsfa001(支持 * ? 通配)、
  // 处理结果 wsfa006、发起端 wsfa013、服务端 wsfa018(后三者等值)。
  const [draft, setDraft] = useState({ service: '', job: '', result: '', origin: '', server: '' })
  const [applied, setApplied] = useState(draft)
  const [onlyFail, setOnlyFail] = useState(false)
  // 日期变化时要复用"最新已提交条件",用 ref 读避免把它写进 effect 依赖
  // (写进去会因输入框每次按键都触发查询);此前此处传空串,改日期会把已填条件清掉。
  const appliedRef = useRef(applied)
  appliedRef.current = applied
  const onlyFailRef = useRef(onlyFail)
  onlyFailRef.current = onlyFail
  // 默认过滤条件:当天(awsq990 查当天日志是最常用场景)
  const today = () => {
    const d = new Date()
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
  }
  const [from, setFrom] = useState(today)
  const [to, setTo] = useState(today)

  // 接口日志**不再依赖活会话**:查的是环境级库里的 wsfa_t(在系统账号 ds 下),
  // 企业编号由上面的下拉选(清单本身来自 gzou_t,见 store.loadEnts)。
  // 环境取"这一页所属的环境"(路由写进 store 的 activeEnv),不是"当前会话的环境"。
  const activeEnv = useStore((s) => s.activeEnv)
  const entsView = useStore((s) => s.entsByEnv[activeEnv])
  const loadEnts = useStore((s) => s.loadEnts)
  // 选中的企业(0 = 还没选)。它只决定"这次查的是哪个企业"(结果回显 + 重放用的 TOPENT),
  // 不改 wsfa_t 的查询范围 —— 接口日志不按企业分 schema。
  const [ent, setEnt] = useState(0)
  const entRef = useRef(ent)
  entRef.current = ent
  // 只有人点过「查询」之后,日期这类后续动作才会自己再发查询;
  // 首屏不发 —— "打开这一页就往人家的库上拨一次 SSH"不是看日志该有的副作用。
  const [queried, setQueried] = useState(false)

  // 切环境:**各存一份**,切走再切回原样还在(与多会话同一条原则)。
  // 结果那半份在 store(wsLogsByEnv,见 setActiveEnv),这半份是这一页自己的输入:
  // 查询条件/日期/企业/是否已查询 —— 都按环境记在模块级 map 里。
  //
  // 为什么不能共用:环境页 keep-alive,甲环境的企业编号在乙环境未必存在,
  // 甲环境的"已查询"状态套到乙环境上就会出现"空列表却没有查询按钮提示"这类怪相。
  const envRef = useRef(activeEnv)
  const viewStateRef = useRef<WsLogViewState>({ draft, applied, onlyFail, from, to, ent, queried })
  viewStateRef.current = { draft, applied, onlyFail, from, to, ent, queried }
  const suppressRequeryRef = useRef(false)
  useEffect(() => {
    if (envRef.current === activeEnv) return
    const prev = envRef.current
    envRef.current = activeEnv
    wsLogViewStateByEnv.set(prev, viewStateRef.current)
    const saved = wsLogViewStateByEnv.get(activeEnv)
    suppressRequeryRef.current = true
    if (saved) {
      setDraft(saved.draft); setApplied(saved.applied); setOnlyFail(saved.onlyFail)
      setFrom(saved.from); setTo(saved.to); setEnt(saved.ent); setQueried(saved.queried)
    } else {
      const d = today()
      const blank = { service: '', job: '', result: '', origin: '', server: '' }
      setDraft(blank); setApplied(blank); setOnlyFail(false)
      setFrom(d); setTo(d); setEnt(0); setQueried(false)
    }
    // 还原这一轮里 from/to 的变化不算"用户改日期",放掉抑制标记给下一次用。
    window.setTimeout(() => { suppressRequeryRef.current = false }, 0)
  }, [activeEnv])

  useEffect(() => {
    if (activeEnv) void loadEnts(activeEnv)
  }, [activeEnv, loadEnts])

  // 企业目录到了就默认选中第一个(常见情形下一个环境只有一两个启用企业)。
  useEffect(() => {
    const items = entsView?.items || []
    if (!ent && items.length) setEnt(items[0].ent)
  }, [entsView, ent])

  // 日期改了重查:只在已经查过一次之后(见 queried 的注释)。
  // 切环境还原状态时会同时改 from/to/ent/queried —— 那一下**不该**触发重查
  // (那是"恢复上次的画面",不是"用户改了日期";重查等于白拨一次 SSH)。
  useEffect(() => {
    if (suppressRequeryRef.current) return
    if (!queried || !entRef.current) return
    void loadWsLogs({ ...appliedRef.current, onlyFail: onlyFailRef.current, page: 1, startFrom: from, endTo: to, ent: entRef.current })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to])

  // 提交查询:Enter / 点「查询」/ 点刷新走这里（把草稿变成已提交条件）。
  const doLoad = (p = 1) => {
    if (!entRef.current) { toast.error('先选企业编号'); return }
    if (!queried) setQueried(true)
    setApplied(draft)
    void loadWsLogs({ ...draft, onlyFail, page: p, startFrom: from, endTo: to, ent: entRef.current })
  }
  // 翻页/勾「仅失败」**沿用已提交条件**，不读 draft。
  //
  // 原来它们也调 doLoad，于是"输入框里刚敲了半个条件、没回车"会被当成正式条件发出去：
  // 实测在服务名称里敲 `bom` 不回车直接点【下一页】，发出去的是 `service=bom&page=2` 的
  // **另一个查询**的第 2 页，列表变 0 条 —— 看着像"翻页把数据翻没了"。
  // 草稿只在回车/刷新那一下生效，这条边界必须守住。
  const loadPage = (p: number) => {
    void loadWsLogs({ ...appliedRef.current, onlyFail: onlyFailRef.current, page: p, startFrom: from, endTo: to, ent: entRef.current })
  }
  const onEnter = (e: React.KeyboardEvent) => { if (e.key === 'Enter') doLoad(1) }

  // 一条都没有时给个台阶:默认查的是"今天",而有的环境(如测试区)最近一条日志是一个多月前的
  // (实测 2026-10-23:测试区 wsfa_t 只到 2026-08-05),用户看到空列表只会以为"日志坏了"。
  // 放宽到最近一年 —— 日志表通常按月清理,一年足够覆盖;仍然是按 wsfa003 倒序取前 200 条,
  // 走索引,不会因为范围大而变慢。
  const widenToLastYear = () => {
    if (!entRef.current) { toast.error('先选企业编号'); return }
    const end = today()
    const d = new Date(); d.setDate(d.getDate() - 365)
    const start = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
    suppressRequeryRef.current = true
    window.setTimeout(() => { suppressRequeryRef.current = false }, 0)
    setFrom(start); setTo(end); setQueried(true)
    void loadWsLogs({ ...appliedRef.current, onlyFail: onlyFailRef.current, page: 1, startFrom: start, endTo: end, ent: entRef.current })
  }

  // ---- 宽度分配:左右两栏按比例分(默认 1:1),之和恒等于容器宽度 ----
  // 容器宽度实测。用 **callback ref** 而不是"挂载时读一次"：这一块**可能后出现**
  // （没有会话时整页是"未连接会话"的空态，split 容器根本没挂；会话一来它才挂上），
  // 那样 boxW 会永远是 0 —— 分栏退回 50%、窄屏判断也失效（实测踩到过）。
  const splitRef = useRef<HTMLDivElement | null>(null)
  const [splitEl, setSplitEl] = useState<HTMLDivElement | null>(null)
  const setSplitNode = useCallback((el: HTMLDivElement | null) => { splitRef.current = el; setSplitEl(el) }, [])
  const [boxW, setBoxW] = useState(0)
  useEffect(() => {
    if (!splitEl) return
    const ro = new ResizeObserver(() => setBoxW(splitEl.clientWidth))
    ro.observe(splitEl)
    setBoxW(splitEl.clientWidth)
    return () => ro.disconnect()
  }, [splitEl])
  const [ratio, setRatio] = useState(() => {
    const v = Number(localStorage.getItem('tt.wslogRatio'))
    return v >= RATIO_MIN && v <= RATIO_MAX ? v : 0.5
  })
  const splittable = Math.max(1, boxW - RESIZER_W)
  // 窄容器下**不再左右分栏**，改成"只显示明细、占满"（× 即返回列表）。
  //
  // 为什么：明细宽度原来只由占比决定（下限 0.25），容器只有 300px 时它就只剩 ~80px ——
  // 键值对与页签一个字符一行（用户实测截图里正是这样："本 信 息"竖着排、
  // 服务名 oa.sc he m.a dat a.get）。列表那一侧也已经被挤到看不出列含义。
  // 与其两边都不能看，不如让明细占满：它本来就是"点开一行要读的东西"。
  const narrowSplit = boxW > 0 && boxW < MIN_LIST_W + MIN_DETAIL_W + RESIZER_W
  // 分栏时明细宽度：按占比，但**夹一个绝对下限**（拖到最小也不许压到读不成）。
  const detailW = narrowSplit
    ? splittable
    : Math.max(MIN_DETAIL_ABS, Math.min(Math.round(splittable * ratio), splittable - MIN_LIST_W))
  // 拖拽时占比的有效下限：与上面的绝对下限一致，否则拖到最左会"线停住、占比还在变"
  const ratioMin = Math.min(RATIO_MAX, Math.max(RATIO_MIN, MIN_DETAIL_ABS / splittable))

  const onDetailResizeDown = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault()
    const el = e.currentTarget
    el.classList.add('dragging')
    const startX = e.clientX
    const startRatio = ratio
    let latest = ratio
    const move = (ev: MouseEvent) => {
      // 分隔线跟手:向左拖 = 详情面板变宽(与 VS Code 侧边栏一致);只改占比,总量不变
      const box = splitRef.current?.clientWidth || 0
      const sp = Math.max(1, box - RESIZER_W)
      latest = Math.min(RATIO_MAX, Math.max(ratioMin, startRatio - (ev.clientX - startX) / sp))
      setRatio(latest)
    }
    const up = () => {
      el.classList.remove('dragging')
      localStorage.setItem('tt.wslogRatio', String(latest))
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
      // 兜住「收不到 mouseup」：鼠标在窗口外松开、或拖拽中窗口失焦 —— 否则这条 mousemove
      // 会一直挂着，面板跟着光标跑，直到用户再点一下（读码结论；难复现，但后果难看）。
      window.removeEventListener('blur', up)
      document.removeEventListener('mouseleave', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
    window.addEventListener('blur', up)
    document.addEventListener('mouseleave', up)
  }

  // Esc 关闭详情(仅在本视图可见时生效,避免影响编辑器/调试页)
  useEffect(() => {
    if (!sel || view !== 'wslogs') return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { e.preventDefault(); closeWsLogDetail() }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [sel, view, closeWsLogDetail])

  const cols = sel ? COLS_COMPACT : COLS_FULL

  // ---- 排序:点表头循环 升序 → 降序 → 恢复默认(服务端的 wsfa003 desc,最新在前) ----
  const [sort, setSort] = useState<{ key: SortKey; dir: 'asc' | 'desc' } | null>(null)
  const toggleSort = (key: SortKey) => setSort((s) => {
    if (s?.key !== key) return { key, dir: 'asc' }
    return s.dir === 'asc' ? { key, dir: 'desc' } : null
  })
  const rows = useMemo(() => {
    if (!sort) return wsLogs
    // 排序键始终从完整列集里取:详情展开后列表切成紧凑列(无耗时/错误描述),
    // 此时排序仍然生效,只是没有那一列的表头指示箭头
    const col = COLS_FULL.find((c) => c.key === sort.key)
    if (!col) return wsLogs
    const sign = sort.dir === 'asc' ? 1 : -1
    // 空值/非法值恒排最后(不随升降序翻面),否则空串会把整列有效值挤到下面
    const emptyLast = (a: string, b: string) => (!a && !b ? 0 : !a ? 1 : !b ? -1 : null)
    return [...wsLogs].sort((x, y) => {
      const va = col.sortValue(x)
      const vb = col.sortValue(y)
      if (typeof va === 'number' && typeof vb === 'number') {
        const na = Number.isNaN(va)
        const nb = Number.isNaN(vb)
        if (na || nb) return na && nb ? 0 : na ? 1 : -1
        return (va - vb) * sign
      }
      const e = emptyLast(String(va), String(vb))
      if (e !== null) return e
      return String(va).localeCompare(String(vb)) * sign
    })
  }, [wsLogs, sort])

  // ---- 报文复制(Request / Response 两个页面的右上角按钮) ----
  const [copyState, setCopyState] = useState<'ok' | 'fail' | null>(null)
  // 换页签/换记录后不残留上一次的"已复制"反馈
  useEffect(() => { setCopyState(null) }, [tab, sel?.rowid])
  const copyPayload = async () => {
    if (!payloadText) return
    const ok = await copyText(payloadText)
    setCopyState(ok ? 'ok' : 'fail')
    if (!ok) {
      // 失败要留得住：原来只有按钮红 1.5 秒，用户不知道该干什么。报文就在编辑器里，
      // 说清"可以手选复制"比反复点按钮有用。
      toast.error('复制失败', {
        description: '系统剪贴板不可用（多半是窗口没有焦点）；这段报文可以直接在编辑器里选中复制。',
        duration: 0,
      })
    }
    window.setTimeout(() => setCopyState(null), ok ? 1500 : 4000)
  }

  // ---- 唯一标识复制 ----
  // 人看到可疑的一行 → 点一下把它的唯一标识(rowid / ctid)复制走 → 粘到对话里
  // → AI 用 tdebug wsdebug <标识> 直接重放这一行。复制内容**就是标识本身**,
  // 不带任何包装文案,免得 AI 还要从周围文字里把它抠出来。
  const [idState, setIdState] = useState<'ok' | 'fail' | null>(null)
  useEffect(() => { setIdState(null) }, [sel?.rowid])
  const copyRowID = async () => {
    if (!sel?.rowid) return
    const ok = await copyText(sel.rowid)
    setIdState(ok ? 'ok' : 'fail')
    if (!ok) {
      // 标识本身也显示在右侧「基本信息」里，手选同样能拿到 —— 把这句话说出来。
      toast.error('复制失败', {
        description: `系统剪贴板不可用（多半是窗口没有焦点）；这一行的标识是 ${sel.rowid}，可以在右侧「基本信息」里选中复制。`,
        duration: 0,
      })
    }
    window.setTimeout(() => setIdState(null), ok ? 1500 : 4000)
  }

  // 悬停预取:在行上停留 ~120ms 就开始取该行报文,点下去时通常已就绪(划过不停留则不触发)。
  // 只对当前页可见的列做,鼠标快速扫过不会发请求。
  const prefetchTimer = useRef<number | undefined>(undefined)
  const onRowEnter = (it: WSLogItem) => {
    window.clearTimeout(prefetchTimer.current)
    prefetchTimer.current = window.setTimeout(() => prefetchWsLog(it), 120)
  }
  const onRowLeave = () => window.clearTimeout(prefetchTimer.current)
  // 视图被隐藏/卸载时也要把预取定时器清掉：它只是个"悬停防抖"，留着会在用户已经离开
  // 这一页之后还去打一次报文请求（视图是 keep-alive，所以必须显式清）。
  useEffect(() => () => window.clearTimeout(prefetchTimer.current), [])
  useEffect(() => {
    if (view === 'wslogs') return
    window.clearTimeout(prefetchTimer.current)
  }, [view])

  // ---- 入参编辑(重放用) ----
  // reqDraft = 编辑器里的当前文本(草稿);与日志里的原始入参不一致即视为"改过",
  // 重放时把草稿发给后端(后端落临时文件后作为入参文件),未改则走原报文。
  // 换行/换报文时重置草稿,避免把上一行的修改带到下一行。
  const [reqDraft, setReqDraft] = useState('')
  useEffect(() => { setReqDraft(content?.request ?? '') }, [content?.request, sel?.rowid])
  const reqDirty = !!content && reqDraft !== content.request

  // 报文复制(Request / Response 两个页签的右上角按钮)。
  // Request 页签复制的是**编辑器里现在的内容**（改过就是草稿），不是原始报文：
  // 同屏写着"改了入参可以按它重放"，复制出去的却是旧的那份，用户会把错的报文贴给别人（实测）。
  // Response 页签没有草稿，照旧复制原始响应。
  const payloadText = tab === 'request'
    ? (reqDraft || content?.request || '')
    : tab === 'response' ? (content?.response || '') : ''

  // （原先这里有一道"没有会话就不渲染整页"的门。已拆除:接口日志查的是环境级的 wsfa_t，
  //   与有没有调试会话无关 —— 会话只在"重放"时用得上。见上面企业编号下拉的注释。）

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col">
      {/* 工具条分两行(条件多,挤一行看不清):
            第一行 = 企业编号(下拉) + 查询 + 五个查询条件(服务名称/作业编号/服务端/发起端/处理结果);
            第二行 = 时间范围 + 仅失败 + 条数/翻页(右对齐)。
          每个条件都是「字段名 + 输入框」两件东西(见 QueryField),不是把字段名写进输入框里。
          下缘分割线与列表连成整体;窄了各自内部换行(flex-wrap)。
          两行之间留出间距(pt/pb),否则输入框与日期行贴在一起显得挤。 */}
      <div className="shrink-0 border-b border-border">
        <div className="flex flex-wrap items-center gap-2 px-2 pt-2 pb-1.5">
          {/* 企业编号:清单来自 gzou_t(固定用系统账号 ds 查),下拉里带上 gzou003 的账号。
              选完点「查询」才发请求 —— 不依赖会话,也不在开页时自己拂一次 SSH。 */}
          <div className="flex items-center gap-1.5"
            title="企业编号来自 gzou_t(gzoustus='Y' 的 gzou001),用系统账号 ds 查;后面是该企业在 gzou003 里的账号/schema">
            <span className="shrink-0 text-xs text-muted-foreground">企业编号</span>
            <Select value={ent ? String(ent) : undefined} onValueChange={(v) => setEnt(Number(v) || 0)}>
              <SelectTrigger className="h-7 w-40 text-xs" aria-label="企业编号">
                <SelectValue placeholder={entsView?.loading ? '读取中…' : (entsView?.err ? '读不到企业清单' : '选择企业')} />
              </SelectTrigger>
              <SelectContent>
                {(entsView?.items || []).map((e) => (
                  <SelectItem key={e.ent} value={String(e.ent)}>
                    {e.ent}{e.account && e.account !== '-' ? ` · ${e.account}` : ' · 未配账号'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <button onClick={() => void loadEnts(activeEnv, true)} disabled={!activeEnv || entsView?.loading}
              title="重新读 gzou_t(固定用系统账号 ds)"
              className="p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:opacity-40">
              <RefreshCw className={`h-3.5 w-3.5 ${entsView?.loading ? 'animate-spin' : ''}`} />
            </button>
          </div>
          <Button size="sm" onClick={() => doLoad(1)} disabled={loading || !ent} title="按当前条件查询（回车同效）">
            {loading ? '查询中…' : '查询'}
          </Button>
          <QueryField
            label="服务名称" wide
            value={draft.service}
            onChange={(v) => setDraft((d) => ({ ...d, service: v }))}
            onKeyDown={onEnter}
            placeholder="支持 * ? 通配"
            title="服务名称 wsfa001:支持 * ? 通配,如 icd.erp.wo*;回车生效"
          />
          <QueryField
            label="作业编号"
            value={draft.job}
            onChange={(v) => setDraft((d) => ({ ...d, job: v }))}
            onKeyDown={onEnter}
            placeholder="如 wssp01131"
            title="作业编号 wsfa012:支持 * ? 通配,如 wssp01131 / wssp*。服务名是 oa.schema.data.get 这类反域名,认不出是哪个作业,按作业找要用这个;回车生效"
          />
          <QueryField
            label="服务端"
            value={draft.server}
            onChange={(v) => setDraft((d) => ({ ...d, server: v }))}
            onKeyDown={onEnter}
            placeholder="如 T100"
            title="服务端 wsfa018:等值匹配,如 T100;回车生效"
          />
          <QueryField
            label="发起端"
            value={draft.origin}
            onChange={(v) => setDraft((d) => ({ ...d, origin: v }))}
            onKeyDown={onEnter}
            placeholder="如 OA"
            title="发起端 wsfa013:等值匹配,如 OA / Athena;回车生效"
          />
          <QueryField
            label="处理结果"
            value={draft.result}
            onChange={(v) => setDraft((d) => ({ ...d, result: v }))}
            onKeyDown={onEnter}
            placeholder="如 000"
            title="处理结果 wsfa006:等值匹配,如 000 成功 / 100 失败;回车生效"
          />
          {/* 这里原先有一个单独的「刷新」图标（= 按当前条件重查）。
              现在左边已经有一个写明的「查询」按钮，同一动作留两个入口只会让人猜
              它们是不是一回事 —— 而且它还经常换行单占一行。 */}
        </div>
        <div className="flex flex-wrap items-center gap-2 px-2 pt-1.5 pb-2">
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            起始时间
            <DatePicker value={from} onChange={setFrom} title="起始时间 wsfa003 的下界(含当天)" />
            ~
            结束时间
            <DatePicker value={to} onChange={setTo} title="结束时间 wsfa004 的上界(含当天,补到 23:59:59.99999)" />
          </div>
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
            <Checkbox id="wslog-onlyfail" checked={onlyFail}
              onCheckedChange={(v) => {
                // 立即生效:显式传新值查询。原先靠 setTimeout(doLoad) 复用闭包里的旧 onlyFail,
                // 首次勾选实际发的是旧值(勾上了但结果没变),这里一并修掉。
                // 条件只取**已提交**的那份:输入框里的草稿要等回车才算数(见 loadPage 的说明)。
                const on = v === true
                setOnlyFail(on)
                void loadWsLogs({ ...appliedRef.current, onlyFail: on, page: 1, startFrom: from, endTo: to, ent: entRef.current })
              }} />
            <label htmlFor="wslog-onlyfail" className="cursor-pointer select-none">仅失败</label>
          </div>
          {sessionBusy && <span className="text-xs text-amber-600/80 dark:text-amber-500/80">调试会话忙碌中:重放将自动结束当前调试</span>}
          {/* 分页 */}
          <div className="ml-auto flex items-center gap-1 text-xs text-muted-foreground">
            <span>{wsLogs.length} 条</span>
            <button onClick={() => loadPage(page - 1)} disabled={loading || page <= 1} title="上一页"
              className="p-1 hover:bg-accent hover:text-foreground disabled:opacity-30">
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="whitespace-nowrap font-mono">第 {page} 页</span>
            <button onClick={() => loadPage(page + 1)} disabled={loading || !hasMore} title="下一页"
              className="p-1 hover:bg-accent hover:text-foreground disabled:opacity-30">
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>

      {/* 查询失败:错误留在界面上（有上一次结果时也不许只发一条会消失的 toast ——
          否则界面上只剩"上一次的数据"，看不出这次其实失败了）。列表整体下推的毛病由
          这条**常驻一行**解决：它不闪、不遮列表，出事就在那儿。 */}
      {err && rows.length > 0 && (
        <div className="flex shrink-0 items-center gap-2 border-b border-red-500/20 bg-red-500/10 px-2 py-1 text-xs text-red-600 dark:text-red-400">
          <span className="shrink-0 font-medium">查询失败</span>
          <span className="min-w-0 truncate" title={err}>{err}</span>
          <span className="ml-auto shrink-0 text-muted-foreground">下面是上一次的结果</span>
        </div>
      )}

      {/* 左右分配:左侧列表 flex-1 吃掉剩余宽度,右侧详情按占比取宽 + 5px 分隔条。
          min-w-0/overflow-hidden 锁住容器宽度:面板是 shrink-0,不加锁会把容器一路撑宽
          (宽度反馈环 → 整页横向溢出) */}
      <div ref={setSplitNode} className="flex min-h-0 min-w-0 flex-1 overflow-hidden">
        {/* 列表:表头与行同处一个滚动容器,横向滚动时表头跟着滚不错位。
            窄容器 + 选中某一行时整列让位给明细（见 narrowSplit）。 */}
        {!(sel && narrowSplit) && (
        <div className="flex min-w-0 flex-1 flex-col overflow-hidden bg-background">
          <div className="min-h-0 flex-1 overflow-auto">
            {/* table-fixed:列宽交给表头/单元格上的 w-* ,留空的那列吃剩余宽度。
                container=false 不套 shadcn Table 默认的 overflow-x-auto 容器(它会让 th 的
                sticky 失效),滚动交给外层。
                网格线用 shadcn Table 的单元格边框(设置页那张表的全框线观感),但必须:
                  · border-separate + border-spacing-0 —— border-collapse 下边框由表格整体绘制,
                    sticky 的 th 带背景压在上面时整行框线会看不见;
                  · 单元格只留 右+下(border-t-0/border-l-0),否则相邻格子边框不合并会叠成 2px;
                  · 表头背景必须不透明且与列表同色(bg-background),否则滚动时行会从表头透出来。 */}
            <Table container={false} className="table-fixed border-separate border-spacing-0 text-xs">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  {cols.map((c) => {
                    const active = sort?.key === c.key
                    const tip = active
                      ? (sort!.dir === 'asc' ? '升序,点击改降序' : '降序,点击恢复默认(最新在前)')
                      : '点击按此列升序排序'
                    return (
                      <TableHead key={c.key} className={`sticky top-0 z-10 border-t-0 border-l-0 bg-background ${c.cls}`}>
                        <button onClick={() => toggleSort(c.key)} title={tip}
                          className="group inline-flex h-7 w-full items-center gap-1 text-left text-[11px] font-medium text-muted-foreground transition-colors hover:text-foreground">
                          <span className="min-w-0 truncate">{c.head}</span>
                          {active
                            ? (sort!.dir === 'asc'
                              ? <ChevronUp className="h-3 w-3 shrink-0 text-foreground" />
                              : <ChevronDown className="h-3 w-3 shrink-0 text-foreground" />)
                            : <ChevronsUpDown className="h-3 w-3 shrink-0 opacity-0 transition-opacity group-hover:opacity-60" />}
                        </button>
                      </TableHead>
                    )
                  })}
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((it) => {
                  const selected = sel?.rowid === it.rowid
                  return (
                    <TableRow key={it.rowid} data-state={selected ? 'selected' : undefined}
                      onClick={() => void selectWsLog(it)}
                      onMouseEnter={() => onRowEnter(it)}
                      onMouseLeave={onRowLeave}
                      title={sel ? undefined : '点击在右侧查看请求/响应报文'}
                      className="h-7 cursor-pointer hover:bg-accent/40 data-[state=selected]:bg-sky-500/10">
                      {cols.map((c) => (
                        <TableCell key={c.key} className={`border-t-0 border-l-0 px-2 py-0 ${c.cls}`}>{c.render(it)}</TableCell>
                      ))}
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
            {rows.length === 0 && !loading && (
              // 失败与"确实没有数据"必须分得开：失败时**结果没被清掉**（store 那条），
              // 所以这里只在真的空列表上说话；有错就照实把错误说出来，并留一个重试出口
              // （原来错误只在右下角活 8 秒，之后界面上再也找不到这次失败的任何痕迹）。
              //
              // 第一档是"还没查过":接口日志不再跟着会话自动拉,要人选企业后点「查询」。
              !queried ? (
                <div className="flex flex-col items-center gap-1.5 p-6 text-center text-xs text-muted-foreground">
                  <span className="text-sm text-foreground">选企业编号，点「查询」</span>
                  <span>接口日志查的是环境库里的 wsfa_t（系统账号 ds），不需要先建会话。</span>
                  {entsView?.err && (
                    <span className="max-w-2xl whitespace-pre-wrap text-red-600 dark:text-red-400">
                      企业清单读取失败（gzou_t）：{entsView.err}
                    </span>
                  )}
                </div>
              ) : err ? (
                <div className="flex flex-col items-center gap-2 p-4 text-center text-xs">
                  <div className="text-red-600 dark:text-red-400">查询失败，没有拿到日志</div>
                  <div className="max-w-2xl whitespace-pre-wrap text-muted-foreground">{err}</div>
                  <button onClick={() => void loadWsLogs({ ...appliedRef.current, onlyFail: onlyFailRef.current, page: 1, startFrom: from, endTo: to, ent: entRef.current })}
                    className="rounded-ui border border-border px-2 py-1 text-muted-foreground hover:bg-accent hover:text-foreground">
                    重试
                  </button>
                </div>
              ) : (
                <div className="flex flex-col items-center gap-1.5 p-6 text-center text-xs text-muted-foreground">
                  <span className="text-sm text-foreground">这段日期没有接口调用记录</span>
                  <span>{from} ~ {to}{activeEnv ? ` · ${activeEnv}` : ''}</span>
                  <span>有些环境的日志是较早的（默认只查今天），放宽日期范围再看看。</span>
                  <button onClick={widenToLastYear}
                    className="mt-1 rounded-ui border border-border px-2 py-1 text-muted-foreground hover:bg-accent hover:text-foreground">
                    查最近一年
                  </button>
                </div>
              )
            )}
          </div>
        </div>
        )}

        {/* 详情(右列,仅选中行后出现):贴右缘整高嵌入,左侧 1px 分割线即拖拽把手。
            窄容器下不渲染列表、自己占满（那时也不渲染分割条 —— 没得拖）。 */}
        {sel && (
          <>
            {!narrowSplit && (
              <div className="col-resizer self-stretch" onMouseDown={onDetailResizeDown} title="拖拽分配宽度" />
            )}
            <div style={narrowSplit ? undefined : { width: boxW ? detailW : '50%' }}
              className={narrowSplit
                ? 'flex min-w-0 flex-1 flex-col overflow-hidden bg-card'
                : 'flex shrink-0 flex-col overflow-hidden bg-card'}>
              <div className="flex h-8 shrink-0 items-center gap-1 overflow-x-auto whitespace-nowrap border-b border-border px-2">
                <button onClick={closeWsLogDetail} title={narrowSplit ? '返回列表(Esc)' : '关闭详情(Esc)'}
                  className="p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground">
                  <X className="h-3.5 w-3.5" />
                </button>
                {([['info', '基本信息'], ['request', 'Request'], ['response', 'Response']] as const).map(([k, label]) => (
                  <button key={k} onClick={() => setWsLogTab(k)}
                    className={`inline-flex items-center gap-1 px-2 py-0.5 text-xs ${tab === k ? 'bg-accent text-foreground' : 'text-muted-foreground hover:text-foreground'}`}>
                    {label}
                    {k === 'request' && reqDirty && <span className="text-amber-600 dark:text-amber-400" title="入参已修改,重放会用它">•</span>}
                  </button>
                ))}
                {reqDirty && (
                  <button onClick={() => setReqDraft(content?.request ?? '')}
                    title="放弃修改,还原为日志里的原始入参"
                    className="inline-flex items-center gap-1 border border-border px-2 py-0.5 text-xs text-muted-foreground hover:bg-accent hover:text-foreground">
                    <Undo2 className="h-3.5 w-3.5" />
                    还原入参
                  </button>
                )}
                <button
                  onClick={() => void replayDebug(sel, reqDirty ? reqDraft : undefined, entRef.current || undefined)}
                  disabled={reqDirty && !!content?.requestPartial}
                  title={reqDirty && content?.requestPartial
                    ? '入参内容不完整(超过读取上限被截断,或未完整入库),不能基于它重放'
                    : reqDirty
                      ? '用上面编辑器里改过的入参重放此接口调用'
                      : '用该日志的原始报文重放此接口调用(T100 r.dg 同款;现有会话会自动收口)'}
                  className="ml-auto inline-flex items-center gap-1 border border-emerald-500/20 px-2 py-0.5 text-xs text-emerald-600 dark:text-emerald-400 hover:bg-emerald-500/10 disabled:pointer-events-none disabled:opacity-40"
                >
                  <Bug className="h-3.5 w-3.5" />
                  {reqDirty ? '用修改后的入参重放' : '调试此调用'}
                </button>
                {/* 把这行日志的唯一标识交给 AI:AI 用 tdebug wsdebug <标识> 直接重放这一行 */}
                <button
                  onClick={() => void copyRowID()}
                  title="复制这行日志的唯一标识(rowid / ctid)。把它发给 AI,AI 就能用 tdebug wsdebug <标识> 直接重放这一行"
                  className={`inline-flex shrink-0 items-center gap-1 border px-2 py-0.5 text-xs transition-colors ${
                    idState === 'fail'
                      ? 'border-red-500/20 text-red-600 dark:text-red-400'
                      : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'
                  }`}
                >
                  {idState === 'ok' ? <Check className="h-3.5 w-3.5" /> : <Fingerprint className="h-3.5 w-3.5" />}
                  {idState === 'ok' ? '已复制' : idState === 'fail' ? '复制失败' : '复制标识'}
                </button>
                {/* 报文页(Request / Response)的复制按钮:放在最右,即详情面板右上角 */}
                {tab !== 'info' && (
                  <button
                    onClick={() => void copyPayload()}
                    disabled={!payloadText}
                    title={payloadText ? '复制报文到剪贴板' : '报文未加载或无内容'}
                    className={`inline-flex shrink-0 items-center gap-1 border px-2 py-0.5 text-xs transition-colors disabled:pointer-events-none disabled:opacity-40 ${
                      copyState === 'fail'
                        ? 'border-red-500/20 text-red-600 dark:text-red-400'
                        : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'
                    }`}
                  >
                    {copyState === 'ok' ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                    {copyState === 'ok' ? '已复制' : copyState === 'fail' ? '复制失败' : '复制'}
                  </button>
                )}
              </div>
              {/* 内容不完整时点明:按原报文重放没问题,但不能在编辑器里改完再重放 */}
              {tab === 'request' && content?.requestPartial && (
                <div className="shrink-0 border-b border-border bg-amber-500/10 px-2.5 py-1 text-[11px] text-amber-600 dark:text-amber-400">
                  入参内容不完整(超过读取上限被截断,或源文件已清理且未完整入库):可以按原报文重放,但不能用修改后的入参重放。
                </div>
              )}
              {tab === 'response' && content?.responsePartial && (
                <div className="shrink-0 border-b border-border bg-amber-500/10 px-2.5 py-1 text-[11px] text-amber-600 dark:text-amber-400">
                  响应报文不完整(超过读取上限被截断,或未完整入库),仅供查看。
                </div>
              )}
              {/* 基本信息是普通内容流,需要外层滚动;报文页交给 Monaco 自己滚(嵌套滚动容器会出多余滚动条) */}
              <div className={`min-h-0 flex-1 ${tab === 'info' ? 'overflow-auto' : 'overflow-hidden'}`}>
                {tab === 'info' && <InfoBody item={sel} narrow={narrowSplit} />}
                {tab === 'request' && (
                  <PayloadBody text={reqDraft} loading={!content} editable onChange={setReqDraft}
                    path={`wslog:${sel.rowid}:req`} />
                )}
                {tab === 'response' && <PayloadBody text={content?.response} loading={!content} path={`wslog:${sel.rowid}:rsp`} />}
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  )
}

// 基本信息(DevTools Headers 风格的两列键值)
// 字段顺序与标签对齐 T100 原生页 awsq990 的列名(见 debug/wslog.go 顶部注释)
//
// narrow = 面板本身很窄时**改成上下两行**(字段名一行、值一行)：
// 左右两列在窄面板里会把值挤到几十像素，长值就被一个字符一行地掰断（用户实测截图）。
// 值的换行用 break-words 而不是 break-all：后者会连普通单词都从中间掰开。
function InfoBody({ item, narrow }: { item: WSLogItem; narrow?: boolean }) {
  const rows: [string, string][] = [
    ['服务名称', item.service],
    ['作业', item.job],
    ['服务程序序号', item.pid],
    ['处理结果', item.code],
    ['开始时间', item.start],
    ['结束时间', item.end],
    ['耗时(s)', item.duration],
    ['sso时间(秒)', item.sso],
    ['发起端', item.origin],
    ['服务端', item.server],
    ['错误消息', item.errMsg],
    ['请求报文文件', item.reqPath],
    ['响应报文文件', item.rspPath],
    ['请求大小(字节)', item.reqSize],
    ['响应大小(字节)', item.rspSize],
  ]
  return (
    <div className="p-2 text-xs">
      {rows.map(([k, v]) => (
        <div key={k}
          className={narrow
            ? 'border-b border-border/60 py-1'
            : 'flex gap-2 border-b border-border/60 py-1'}>
          <span className={narrow ? 'block text-muted-foreground' : 'w-28 shrink-0 text-muted-foreground'}>{k}</span>
          <span className={narrow
            ? 'mt-0.5 block break-words text-foreground'
            : 'min-w-0 flex-1 break-words text-foreground'}
            title={v || undefined}>{v || '-'}</span>
        </div>
      ))}
    </div>
  )
}

// 报文内容:与源码编辑器同一个 Monaco(同主题 = 同背景色)。
// Request 页可编辑(改过的入参可直接用于重放),Response 页只读。
function PayloadBody({ text, loading, path, editable, onChange }: {
  text?: string; loading: boolean; path?: string; editable?: boolean; onChange?: (v: string) => void
}) {
  return (
    <PayloadEditor
      text={text}
      loading={loading}
      path={path}
      editable={editable}
      onChange={onChange}
      emptyHint="无报文(超过入库大小上限且源文件已清理)"
    />
  )
}
