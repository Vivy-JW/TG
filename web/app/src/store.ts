import { create } from 'zustand'
import { api, API_BASE, type Event, type StopInfo, type Breakpoint, type Frame, type VarItem, type WSLogItem, type WSLogContent, type WSLogQuery, type EntItem, type InflightInfo, type SessionBrief } from './api'
import { THEME_KEY, applyDark, readStoredTheme, resolveDark, watchSystemTheme, type ThemeMode } from '../../shared/theme'
// 统一消息出口(零依赖的那个文件,见 toastBus.ts 的说明:store 要能在裸 Node 里加载)
import { toast } from './toastBus'
// 只取类型:routing.ts 里有 DOM 相关逻辑,虽然都在函数体内,这里也只导入类型
// —— 打包后可确保 store 与路由模块之间没有运行时依赖(check:store 在 Node 里跑)。
// 视图键也取自这里:键只有一份定义(见 routing.ts 与本目录 README),别在这儿再列一遍。
import type { SettingsSectionKey, ViewKey } from './routing'
export type { ThemeMode }

// ---------- 工作站外壳的两级目录 ----------
//
// 环境 = 设置 → 站点管理里配的那一条(`config.json` 的 hosts.sshs),外壳只取显示用的字段;
// 会话 = 环境下面的二级条目，**每一条都是服务端真实的会话句柄**（`POST /api/sessions/new`
// 建出来的那条），不是本地假数据。第一阶段那批"只存在浏览器里"的会话已经随
// `tt.sessions.v1` 一起作废（见 SESSIONS_KEY 的注释）。
export interface EnvBrief { name: string; host?: string; zone?: string }

/** TOP GP 的环境。与 EnvBrief 分开：GP 有 $TOP/$CUST/镜像/字典状态，T100 有 zone/ENT。 */
export interface GpEnvBrief {
  name: string
  host?: string
  port?: number
  user?: string
  instance?: string
  workspace?: string
  custRoot?: string
  /** 营运据点代号（zta_file.zta02 / Oracle schema）；空 = ds。直查按它取，同步固定按 ds。 */
  company?: string
  /** 营运据点名字（azp_file.azp02），从服务器刷新时抄下来。 */
  companyName?: string
  mirrorDir?: string
  db?: { type?: string; host?: string; port?: number; sid?: string; accounts?: { account: string }[] }
  dict?: { exists?: boolean; sizeMB?: number; path?: string }
  mirror?: { exists?: boolean; files?: number; sizeMB?: number; path?: string }
  job?: {
    state?: string
    kind?: string
    done?: number
    total?: number
    progress?: string[]
    error?: string
    result?: Record<string, any>
    elapsed?: string
    startedAt?: string
  }
}
export type SessionType = 'debug'
export interface SessionItem {
  // 服务端会话句柄（C5.4：路径与句柄只认 id）。调试面板绑定、画面通道、结束会话都用它。
  id: string
  env: string
  name: string
  // type 是**派生的**（目前只有 debug 一种）。
  type: SessionType
  createdAt: number
  // 建会话时点名的企业编号与作业编号。ENT 是**会话级事实**（环境配置里已经没有它），
  // 作业编号是"这条会话要调哪个作业"——后面起作业都从这两个值往下走。
  ent: string
  prog: string
  // module 是该会话当前作业所属模块（服务端 brief 直接给）。
  // 不能从作业编号前缀推：bsft001_wf 属于 asf 模块，前缀却是 bsf。
  module: string
  // 服务端给的规范 id「环境#序号」（建会话时就分配了）。空串 = 服务端没给（不该发生）。
  ref: string
}

/**
 * 默认会话名：`<环境> <序号>`。
 *
 * 为什么不能沿用「新会话 N」：别名在服务端是**全池唯一（不区分大小写）**，而 N 是按
 * "本环境已有条数"算的 —— 正式区建一条叫「新会话 1」，到测试区再建又是一条「新会话 1」，
 * 第二条会被"别名已被占用"直接拒掉（用户的原话：另一个站点也是新会话 1，还不能直接创建）；
 * 就算建成了，树里两条同名也分不出谁是谁。
 *
 * 序号仍从本环境的条数起算（同一个站点里连续建是 1、2、3），再跳过已被占用的名字 ——
 * 历史遗留的「新会话 1」、或者别处手工起的同名，都不会撞上。
 */
function defaultSessionName(sessions: SessionItem[], env: string): string {
  const used = new Set(sessions.map((s) => (s.name || '').trim().toLowerCase()))
  let n = sessions.filter((s) => s.env === env).length + 1
  for (let i = 0; i < 999; i += 1) {
    const cand = `${env} ${n}`
    if (!used.has(cand.toLowerCase())) return cand
    n += 1
  }
  return `${env} ${Date.now()}`
}

/**
 * SessionRuntime 是**一条会话的运行时**（W6 的第一步）：服务端把它那一份状态广播过来，
 * 这里按 id 各存一份。它与「面板绑着的那条」解耦 —— 收到 A 的事件时，B 的运行时照旧更新。
 *
 * 这一批只收"索引级"的那几个字段（树上要看、页面切换要用的）；编辑器那一套
 * （源码、断点面板、调用栈、时间线）仍在 `useStore` 的单例上，等 W-P2 的下一阶段搬。
 * 字段全部来自服务端事实或本地派生，**没有第二份真相**：
 *   - `state` / `stop` / `prog` / `module` / `mode` 来自事件或 `GET /api/sessions`。
 */
export interface SessionRuntime {
  id: string
  state: string
  stop: StopInfo | null
  prog: string
  module: string
  mode: 'solo' | 'collab'
  // updatedAt 只用于"这份运行时是什么时候变的"（排查事件断档时有用），不参与渲染判定。
  updatedAt: number
}

/** 创建会话的入参（Debug 会话：企业编号 + 作业编号，名字可选）。 */
export interface CreateSessionInput {
  env: string
  ent: string
  /** TOP GP 专用：模块（csf）。给了就不按作业名前缀推导。 */
  module?: string
  prog: string
  name?: string
}

/** 环境页那张卡片填的东西：环境由它自己在哪一页决定，所以不带 `env`。 */
export type SessionDraft = Omit<CreateSessionInput, 'env'>

// timeline 条目(人/AI/系统 的操作与事件,可审计)
export interface TimelineItem {
  time: string
  origin: 'system' | 'human' | 'ai'
  text: string
  kind: 'info' | 'stop' | 'command' | 'warn' | 'error'
}

interface Watch { expr: string; value?: string; error?: string }

// 浏览页签:静态打开的源码文件(如 Ctrl+点击跳函数),与调试页互不干扰
export interface SrcTab {
  key: string
  file: string // DVM 文件名(如 s_apmt520_conf_chk.4gl)
  content: string
  path: string
  line?: number // 打开时定位行
  loading: boolean
  missing?: boolean // 无源码(只有 42m)
}

interface Store {
  // 连接
  wsConnected: boolean
  // 会话
  sessionId: string | null
  // 服务端的**全部**活动会话（`GET /api/sessions`，Q4 的"列全部"）。
  // 它只用于显示与"我选的那条还在不在"的校验 —— **不参与挑会话**：
  // 选中态是 `sessionId`，只能由人点（`selectSession`）或本页面自己发起的动作写。
  sessionList: SessionBrief[]
  // ---- 工作站外壳:环境树 + 会话树 ----
  // `envs` 来自共享端点 `/api/hosts` 的 `sshs`；`sessions` 是**外壳那棵树**，
  // 它由 `sessionList`（服务端池）**派生**（W2/W6/D9）—— 本地不存"有哪些会话"。
  // 它与下面那组调试态（`sessionId` / `state` / `stop` …）是一件事的两面：
  // 树回答"有哪些"，`sessionId` 回答"此刻面板绑的是哪一条"。
  envs: EnvBrief[]
  envsLoaded: boolean
  envsErr: string
  // TOP GP 的环境：来自本机 ttgp 服务，与上面那份**并列**（不是同一棵树里的成员）。
  gpEnvs: GpEnvBrief[]
  gpEnvsLoaded: boolean
  gpEnvsErr: string
  /** 设置页里当前要编辑的 GP 环境（左树点进来时写它） */
  settingsGpEnv: string
  setSettingsGpEnv: (n: string) => void
  // 当前路由所在的环境。
  activeEnv: string
  sessions: SessionItem[]
  // sessionsLoaded = 池**列过至少一次**（不是"有会话"）。路由要用它区分
  // "还没有会话"与"还没问过" —— 否则深链接会在首帧被当成失效目标弹回环境页。
  sessionsLoaded: boolean
  // editor 是**每条会话的编辑器现场**（W6/W-P2）：按 id 各一份，切换绑定时换进换出单例。
  editor: Record<string, EditorState>
  // runtimes 是**每条会话的运行时**（W6）：按 id 各一份，事件到了就更新。
  // 它与 `sessions`/`sessionList`（池的投影）分工：后者回答"有哪些"，前者回答"它现在怎么样"。
  runtimes: Record<string, SessionRuntime>

  loadEnvs: () => Promise<void>
  loadGpEnvs: () => Promise<void>
  setActiveEnv: (env: string) => void
  // createSession 建一条**真的** Debug 会话（企业编号 + 作业编号），默认把画面通道（proxy）
  // 一起挂上。onStep 用来把"正在登录 / 正在挂画面"这些阶段报给界面（长跑动作要有进度）。
  createSession: (input: CreateSessionInput, onStep?: (step: string) => void) => Promise<SessionItem>
  // ensureSource 在没有源码可看时把这条会话的**母版源码**打开。
  // 会话页有三种"有会话但没停站"的时刻（刚建好 / 刚起跑 / 从树上点回来），
  // 那时人也想看代码 —— 旧实现只有停站才拉源码，于是编辑器一片空白。
  ensureSource: () => Promise<void>
  // closeSession 结束一条会话（断开连接 + 从服务端池里与本地树里一起摘掉）。
  closeSession: (id: string) => Promise<void>

  module: string
  prog: string
  state: string // loading|stopped|running|idle|exit|''
  sessionEnv: string // 会话所属环境名(设置页 envs;idle/状态栏/会话列表显示)
  started: boolean // 程序是否已 run 过(入口停站时步进不可用)
  stop: StopInfo | null
  holdingSeconds: number
  // 这一轮不会自动放行（运行时口径）。停站时状态栏靠它区分“还有 N 秒”与“永远不会”。
  watchdogSeconds: number
  watchdogDisabled: boolean
  launching: boolean
  launchError: string // 启动失败显眼报错(编辑器外红色横幅,空 = 正常)
  // UI 面板显隐
  //
  // 右侧那一列**不再是一个"开关"**（原来的 showRight），而是两块各自独立的开关：
  // 调试工具面板（bug-play）与大纲/树（list-tree）。三个开关（含下方面板的 terminal）
  // 相互独立、可同时激活 —— 所以这里没有"右侧栏显不显示"这个字段，
  // 它是 `rightDebugOpen || rightTreeOpen` 算出来的（见 App.tsx 的 showRight）。
  showBottom: boolean
  // 下方面板是否「最大化」(占满代码区那一列,编辑器整块让位)。
  // 与下面的开关同一档:纯界面状态,只活在这一次会话里,不落盘 ——
  // 「最大化」是一次临时的"我要专看这段协议流",刷新后回到常规分栏才是预期。
  panelW: number
  setPanelW: (w: number) => void
  bottomMax: boolean
  rightDebugOpen: boolean // 右侧上半/整列：调试面板（变量/断点/调用栈）
  rightTreeOpen: boolean  // 右侧下半/整列：大纲（树）
  // 两块同时打开时谁在上半。规则是**后开的在下面**（用户定的）：单独开一个时它占满整列，
  // 再开另一个就拆一半给新来的（用户原话："做一个向下 split 的逻辑，拆出一半区域留给 tree 面板"）。
  rightTop: 'debug' | 'tree'
  toggleRightDebug: () => void
  toggleRightTree: () => void
  // 调用栈/自动变量面板「自动」开关(手风琴头部图标,默认关):停站后是否自动抓取该数据
  // (两者都要经调试会话逐条发命令,自动调度越多步进越卡;关掉后仅手动/开关开启时取)
  stackAuto: boolean
  autovarsAuto: boolean
  // 最近一次接口日志重放(wslogs「调试此调用」)的日志 rowid:重放会话点「重新开始」
  // 时按同一日志重放(报文参数在后端),而不是普通启动丢参数;null = 普通作业调试
  lastReplayRowid: string | null
  // 上次重放用的「改过的入参」(空 = 用原报文);重新开始时沿用它,避免编辑被悄悄丢弃
  lastReplayRequest: string
  // 数据
  breakpoints: Breakpoint[]
  adjustedBps: Record<number, number> // 点击行号 → 实际注册断点编号(fgldb 会把非可执行行的断点自动下移)
  frames: Frame[]
  watches: Watch[]
  autovars: VarItem[] // 停站自动变量(当前源码窗内变量的自动求值)
  selectedFrame: number // 当前选中栈帧(-1 = 未选)
  backendDead: string // 后端死亡/程序退出原因(空 = 正常)
  // 谁在驾驶 + 正在执行哪条命令(来自快照;协作模式下界面靠它收敛成观察台)
  mode: 'solo' | 'collab'
  inflight: InflightInfo | null
  timeline: TimelineItem[]
  rawLog: string[]
  runProg: string // gzzz_t 解析出的实体程序(源码命名/预取用);空 = 与 prog 相同
  // 视图与接口日志(VS Code 活动栏切换)
  view: ViewKey
  // 设置页当前分区(站点管理/数据字典/DEBUG/应用设置)。放在 store 而不是 SettingsView 局部
  // 是因为 App 的首屏深链接解析要写它。写在这里的是"纯状态",URL 片段由 SettingsView 同步。
  settingsSection: SettingsSectionKey

  theme: ThemeMode // 外观选择:暗色/亮色/跟随系统(html.dark 挂点,localStorage tt.theme 持久化)
  dark: boolean // 解析结果(system 时随系统变化);html.dark 与 Monaco 主题都按它渲染
  // 服务测试(复刻 awsq990 集成服务测试)

  wsLogs: WSLogItem[]
  wsLogsLoading: boolean
  wsLogsPage: number
  wsLogsHasMore: boolean
  wsLogSel: WSLogItem | null
  wsLogContent: WSLogContent | null
  wsLogTab: 'info' | 'request' | 'response'
  wsLogErr: string
  // 接口日志**按环境分开记**(与多会话同一条原则):上面这组扁平字段只代表
  // "当前所在环境的那一页",切环境时整组存进/取自本表。
  //
  // 为什么不能共用一个槽:环境页是 keep-alive 的,切回来时列表还是上一个环境的内容
  // —— 2026-10-23 实测:测试区/正式区的接口日志都显示成鼎捷的(反之亦然)。
  // 当时先做的是"切环境就清空",但那是把用户刚查好的结果扔掉;正确做法是各存一份。
  wsLogsByEnv: Record<string, WsLogSnap>
  // 企业目录(ENT → 数据库账号)按环境缓存。接口日志页与「创建会话」的 ENT 下拉
  // 共用这一份 —— 两处各拉一次会让同一张 gzou_t 被问两遍(每次都可能拨 SSH)。
  entsByEnv: Record<string, EntsView>
  // loadEnts 拉一个环境的企业目录。force = 跳过客户端缓存(服务端那边还有一层
  // 10 分钟缓存 + 快照,所以"刷新"不会每次都真的拨 SSH)。失败不抛错,写进 err。
  loadEnts: (env: string, force?: boolean) => Promise<EntItem[]>
  // 源码
  sourceContent: string
  sourcePath: string
  sourceDVM: string // 当前已加载源码对应的 DVM 模块文件名(无源码模块也记录,避免错文件高亮)
  // 该 DVM 的源码**没取到**(com 公共库只有 42m / 服务器暂时读不到)。
  // 必须与 sourceDVM 分开记:失败时 sourceDVM 也要更新(否则停站行会画在旧文件上),
  // 但若只凭 sourceDVM 判"同文件已加载",失败一次就把这个文件永久钉成了空白 ——
  // 之后每次 refreshSource 都在同文件分支直接 return,再也不重试。
  sourceMissing: boolean
  currentLine: number
  loadingSource: boolean
  lineOffset: number // 行号校准偏移:DVM 行号 - 磁盘行号(>0 时 Monaco 顶部前插 offset 行对齐协议流)
  // 源码多页签:调试页固定第一个(跟随停站),浏览页为静态打开的其它源码文件
  tabs: SrcTab[]
  activeTab: string // 'debug' | tabs[].key
  revealReq: { key: string; line: number; seq: number; nav?: boolean } | null // 定位信号:哪个页签滚到哪行(唯一驱动视口滚动)

  // actions
  setWsConnected: (b: boolean) => void
  toggleBottom: () => void
  toggleBottomMax: () => void
  toggleStackAuto: () => void
  toggleAutovarsAuto: () => void
  // 接口日志重放启动(wslogs「调试此调用」与重放会话「重新开始」共用同一上下文)
  // ent = 重放这一轮的企业编号（接口日志页下拉选的那个）；空 = 不覆盖。
  // env = 这条日志所属的环境名（`activeEnv`）；空 = 不点名（落在服务端当前环境上）。
  replayStart: (rowid: string, request?: string, ent?: number, env?: string) => Promise<void>
  pushRaw: (line: string) => void
  sendRaw: (cmd: string) => Promise<void>
  pushTimeline: (item: Omit<TimelineItem, 'time'>) => void
  onEvent: (ev: Event) => void
  // WS 开场补发:只把历史填进时间线,不触发任何副作用
  onReplay: (ev: Event) => void
  setView: (v: ViewKey) => void
  setSettingsSection: (s: SettingsSectionKey) => void

  // 明细页的页签:分层记录 / 全文流。**记在本地**(localStorage),用户回来还在那一页 ——
  setLaunchError: (msg: string) => void
  setTheme: (t: ThemeMode) => void
  setActiveTab: (key: string) => void
  reveal: (key: string, line: number, nav?: boolean) => void
  closeTab: (key: string) => void
  openSourceTab: (file: string, line?: number) => Promise<void>
  locate: (word: string) => Promise<{ file: string; line: number }>
  calibrate: () => Promise<void>

  loadWsLogs: (q: WSLogQuery) => Promise<void>
  resetWsLogs: () => void
  // 写"当前环境这一页"的日志状态,并同步镜像到 wsLogsByEnv(供切回来时还原)。
  patchWsLog: (patch: Partial<WsLogSnap>) => void
  selectWsLog: (item: WSLogItem) => Promise<void>
  prefetchWsLog: (item: WSLogItem) => void
  setWsLogTab: (t: 'info' | 'request' | 'response') => void
  closeWsLogDetail: () => void
  replayDebug: (item: WSLogItem, request?: string, ent?: number) => Promise<void>
  launch: (module: string, prog: string) => Promise<void>
  refreshSnapshot: () => Promise<void>
  refreshSource: (file?: string, line?: number) => Promise<void>
  refreshFrames: () => Promise<Frame[]>
  refreshWatches: () => Promise<void>
  addWatch: (expr: string) => Promise<void>
  removeWatch: (expr: string) => void
  selectFrame: (idx: number) => Promise<void>
  toggleBPEnabled: (num: number, enabled: boolean) => Promise<void>
  jumpToBp: (b: Breakpoint) => Promise<void>
  runToCursor: (line: number) => Promise<void>
  control: (action: string, arg?: string) => Promise<void>
  // 切换模式(纯人工/协作)
  setMode: (mode: 'solo' | 'collab') => Promise<void>
  toggleBreakpoint: (line: number) => Promise<void>
  removeBreakpoint: (num: number) => Promise<void>
  quit: () => Promise<void>
  restart: () => Promise<void>
  doPrint: (expr: string) => Promise<void>
  adoptExisting: () => Promise<void>
  // 会话管理(右侧「会话」sheet):切换/重启/结束常驻会话(会先结束当前 debug)
  sessionOp: (op: 'close' | 'restart' | 'switch', env?: string, ent?: string) => Promise<void>
  // 人点一条会话（Q4 的硬规矩：选中态只能由人点出来，见上面「会话绑定的取数纪律」）。
  // 传 null = 明确回到"未选择"。
  selectSession: (id: string | null) => Promise<void>
  syncFromSessions: () => Promise<void>
  // 停站态到了但本地不知道停在哪时的兜底(入口停站只发 state,没有 stopped 事件)
  syncStopFromSnapshot: () => Promise<void>
}

let snapTimer: number | undefined
let holdTimer: number | undefined

// 主题的"用户选择"读一次即可:theme 与解析结果 dark 必须同源,别读两遍
const initialTheme: ThemeMode = readStoredTheme()

// 旧会话收口窗口:重放/重新开始都要先把上一个会话收口(同目标只结束本轮 = idle 复用宿主,
// 不同目标直接断开),收口过程会上报 idle/exit/dead。宿主复用时新旧会话 ID 相同,这些收尾
// 事件会跟着新会话一起通过上面的 sessionId 过滤,把刚置上的加载态清掉、或弹出"后端断开"
// 的误报横幅。收口期间(以及响应后 2s,容忍 WS 迟到)丢弃它们;万一真的启动失败,后端还有
// 「启动失败」日志兜底退出加载态,轮询也会兜底。
let retireID: string | null = null
let retireUntil = 0
const RETIRE_GRACE_MS = 2000

function beginRetire(id: string | null) { retireID = id; retireUntil = 0 }
function endRetire() { if (retireID) retireUntil = Date.now() + RETIRE_GRACE_MS }
function cancelRetire() { retireID = null; retireUntil = 0 }

// 该事件是否属于"正在收口的旧会话"的收尾事件(需丢弃)
function isRetiringTeardown(ev: Event): boolean {
  if (!retireID || ev.sessionId !== retireID) return false
  if (retireUntil && Date.now() >= retireUntil) { retireID = null; return false }
  return ev.type === 'dead' || (ev.type === 'state' && (ev.state === 'idle' || ev.state === 'exit'))
}

// 会话模式归一:solo(纯人工)| collab(协作,AI 主导)。
// 列表、快照两条路都要落同一个值:少落一处,界面就按旧模式收敛写操作 ——
// 典型表现是 AI 起的协作会话在界面上显示成"交给 AI"(当前纯人工),命令框还可写。
function modeOf(m?: string): 'solo' | 'collab' {
  return m === 'collab' ? 'collab' : 'solo'
}

// ---------- 会话绑定的取数纪律（Q4 的硬规矩 / C6.3 / C8.3）----------
//
// **前端不得向服务端询问"当前是哪条会话"**：它应当**列全部 + 自己记住人点的那条**。
// 从前那两个函数（`adoptFromServer` / `scheduleRebindCheck`）正是"靠服务端默认"的实现 ——
// 本地没绑定时按 `GET /api/sessions` 的**第一项**认领，本地绑的那条没了也按第一项重绑。
// 硬切之后（服务端五处 `Current()` 兜底全删）那套写法会把隐式默认从"服务端兜底"
// 搬到"HTTP 语义"上，问题原样存在 —— 所以本批把它整个换掉：
//
//   - `useStore.sessionId` = **人点的那条**（`selectSession` 写入，`tt.sessionId` 持久化）；
//   - `useStore.sessionList` = **列全部**（`refreshSessionList`），只用来显示与校验；
//   - 本地没选 → 就是**未选择**，界面提示"选一条"，绝不替人挑一条；
//   - 本地选的那条从服务端消失了 → **回到未选择**（清掉），也不挑下一条。
//
// 唯一的自动绑定是"本页面自己发起的那次会话动作"（`launch` / `replayStart` /
// `selectSession` 的返回值），那是**人点的**、有明确目标的，不算询问服务端"当前是哪条"。
const SESSION_KEY = 'tt.sessionId'

function readStoredSessionId(): string | null {
  const v = localStorage.getItem(SESSION_KEY)
  return v && v.trim() ? v.trim() : null
}

// setBoundSession 是"绑定/解绑本地选中会话"的**唯一**写入口：句柄与持久化必须同源，
// 两处各写一份必然出现"界面绑着 A、刷新后又回到 B"。
function setBoundSession(id: string | null) {
  if (id) localStorage.setItem(SESSION_KEY, id)
  else localStorage.removeItem(SESSION_KEY)
  useStore.setState({ sessionId: id })
}

// ---------- 外壳会话树：**服务端派生**（W2 / W6 / D9） ----------
//
// 树是 `GET /api/sessions` 与池事件（`session.added` / `session.removed`）的**投影**。
// 本地不再记"有哪些会话"：记了就必然与事实漂——CLI 建的会话在树里不存在、
// 服务端重启后树里留一条点进去什么都没有。`localStorage` 只留 UI 偏好
// （展开集合、实例顺序、面板宽度、主题）。
//
// 旧键只用来**清一次**：v2 存的是"浏览器自己记的会话树"，第二阶段起改为服务端派生，
// 留着它只会让旧条目在树上多活一会儿（它们点开也找不到服务端那条）。
const LEGACY_SESSIONS_KEY = 'tt.sessions.v2'

function clearLegacySessions() {
  try { localStorage.removeItem(LEGACY_SESSIONS_KEY) } catch { /* 隐私模式：清不掉也不影响事实 */ }
}

// 模块加载时清一次：旧树的数据没有任何用途，留着只会让人以为"浏览器还记着会话"。
clearLegacySessions()

// （不再有"用途"这一层：所有会话都是调试会话。）
// 下面这段注释保留在原位，避免与上面 Store 形状的注释断档。
// （恒定键的兜底：取不到给缺省，不让页面猜）。


// itemFromBrief 把池里的那一条投影成树里的一项。
//
// 这里**只做投影，不补业务字段**：brief 里没有的（比如"是谁在本机建的"）不该出现在树上。
function itemFromBrief(b: SessionBrief): SessionItem {

  return {
    id: b.id,
    env: b.env || '',
    name: (b.name || '').trim() || b.ref || b.id,
    type: 'debug',
    createdAt: 0,
    ent: b.ent || '',
    prog: b.prog || '',
    module: b.module || '',
    ref: b.ref || '',
  }
}

// indexFromList 把整份池投影成树（顺序固定：同一环境内按序号，环境间按名字），
// 让树的形状**只由服务端决定** —— 两次相同的池一定得到相同的树。
function indexFromList(list: SessionBrief[]): SessionItem[] {
  const items = list.filter((b) => b && b.id).map(itemFromBrief)
  items.sort((a, b) => (a.env === b.env ? a.ref.localeCompare(b.ref) : a.env.localeCompare(b.env)))
  return items
}

// ---------- 运行时（W6）：按 id 各一份 ----------
//
// 为什么现在就加：树与页面要回答"**别的**会话现在怎么样"（状态点、切页时的现场）。
// 单例那套字段只装得下"绑着的那条"，收到别人的事件只能丢掉 —— 而多会话下那是常态。
// 这一批只收索引级的字段；编辑器那一套仍在单例上（下一阶段搬）。

function emptyRuntime(id: string): SessionRuntime {
  return { id, state: '', stop: null, prog: '', module: '', mode: 'solo', updatedAt: 0 }
}

// runtimeFromBrief 用池里快照初始化/修正一份运行时。
//
// **不拿 brief 覆盖已有值**：brief 是"列表那一刻"的快照，而事件比它新
// （先起跑 → 事件 running，列表还是 idle）。所以只补空着的字段。
function runtimeFromBrief(b: SessionBrief, prev?: SessionRuntime): SessionRuntime {
  const r = prev ? { ...prev } : emptyRuntime(b.id)
  r.id = b.id
  if (!r.state && b.state) r.state = b.state
  if (!r.prog && b.prog) r.prog = b.prog
  if (!r.module && b.module) r.module = b.module
  if (b.mode === 'solo' || b.mode === 'collab') r.mode = b.mode
  r.updatedAt = Date.now()
  return r
}

function patchRuntime(id: string, patch: Partial<SessionRuntime>) {
  if (!id) return
  useStore.setState((s) => {
    // 第一次见到这条会话时，先从池里那一条取个底（作业/模块/模式）—— 事件可能比列表刷新
    // 先到（新建会话的 state 事件就在 added 之后几百毫秒内来），没有底就会把 prog 弄成空。
    const seed = s.runtimes[id] || (() => {
      const b = s.sessionList.find((x) => x.id === id)
      return b ? runtimeFromBrief(b) : emptyRuntime(id)
    })()
    return { runtimes: { ...s.runtimes, [id]: { ...seed, ...patch, id, updatedAt: Date.now() } } }
  })
}

function seedRuntimes(list: SessionBrief[]) {
  useStore.setState((s) => {
    const runtimes: Record<string, SessionRuntime> = { ...s.runtimes }
    const seen = new Set<string>()
    for (const b of list) {
      if (!b?.id) continue
      seen.add(b.id)
      runtimes[b.id] = runtimeFromBrief(b, runtimes[b.id])
    }
    // 池里没有的运行时不留：它属于已经消失的会话（与树同一条口径）。
    for (const id of Object.keys(runtimes)) if (!seen.has(id)) delete runtimes[id]
    return { runtimes }
  })
}

function dropRuntime(id: string) {
  if (!id) return
  useStore.setState((s) => {
    const hasRuntime = !!s.runtimes[id]
    const hasEditor = !!s.editor[id]
    if (!hasRuntime && !hasEditor) return {}
    const runtimes = { ...s.runtimes }
    delete runtimes[id]
    const editor = { ...s.editor }
    delete editor[id]
    return { runtimes, editor }
  })
}

/**
 * useSessionRuntime 读一条会话的运行时（W6）。页面与树用它回答"它现在怎么样"，
 * 不必先把面板绑过去。
 */
export function useSessionRuntime(id: string | null | undefined): SessionRuntime | undefined {
  return useStore((s) => (id ? s.runtimes[id] : undefined))
}

// ---------- 每条会话的**编辑器现场**（W6 / W-P2） ----------
//
// 单例那一组字段（源码、页签、断点面板、调用栈、时间线、停站现场…）是**页面的读入口**，
// 组件照着它渲染。多会话要的不是"再抄一份给组件读"，而是"切换时把这一条的那份换进来"：
//
//   A → B：先把 A 的现状存进 `editor[A]`，再把 `editor[B]`（没有就是空现场）换进单例；
//   B → A：A 的源码/页签/时间线/停站现场原样回来，不必重新拉一遍。
//
// 这样组件一行都不用改（它们只认识单例），而"每条会话各有各的页面现场"这件事成立 ——
// 也是 W-P3 多实例页签的地基（那时把"换进换出"换成"每实例一份 store"即可）。
//
// 哪些键属于现场：**只有这条会话自己的**。开关类（下方面板、右栏、主题）、
// 池与路由（sessions/sessionList/view…）不进来 —— 它们是应用级的，切换会话不该变。
const EDITOR_KEYS = [
  'module', 'prog', 'runProg', 'state', 'sessionEnv', 'started', 'stop', 'holdingSeconds',
  'watchdogSeconds', 'watchdogDisabled',
  'mode', 'inflight', 'backendDead',
  'breakpoints', 'adjustedBps', 'frames', 'watches', 'autovars', 'selectedFrame',
  'timeline', 'rawLog',
  'sourceContent', 'sourcePath', 'sourceDVM', 'sourceMissing', 'currentLine', 'loadingSource', 'lineOffset',
  'tabs', 'activeTab', 'revealReq',
  'lastReplayRowid', 'lastReplayRequest', 'launchError',
] as const

export type EditorKey = (typeof EDITOR_KEYS)[number]
export type EditorState = Pick<Store, EditorKey>

/** emptyEditor 给"这条会话还没打开过"一个干净现场（数组一律新建，别共享引用）。 */
function emptyEditor(): EditorState {
  return {
    module: '', prog: '', runProg: '', state: '', sessionEnv: '', started: false, stop: null,
    holdingSeconds: 0, watchdogSeconds: 0, watchdogDisabled: false, mode: 'solo', inflight: null, backendDead: '',
    breakpoints: [], adjustedBps: {}, frames: [], watches: [], autovars: [], selectedFrame: -1,
    timeline: [], rawLog: [],
    sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false, currentLine: 0,
    loadingSource: false, lineOffset: 0,
    tabs: [], activeTab: 'debug', revealReq: null,
    lastReplayRowid: null, lastReplayRequest: '', launchError: '',
  }
}

// captureEditor 把单例里属于"这条会话现场"的那些键抄出来。
function captureEditor(st: Store): EditorState {
  const out = emptyEditor()
  for (const k of EDITOR_KEYS) (out as any)[k] = st[k]
  return out
}

// applyEditor 把一份现场换进单例（组件下一帧就按它渲染）。
function applyEditor(set: (p: Partial<Store>) => void, ed: EditorState) {
  set(ed as Partial<Store>)
}

// switchEditor 在绑定切换时搬运现场：先存旧的，再换新的（没有就给空现场）。
//
// **没有 saved 时必须换成空现场**：沿用上一条的源码，就是"绑的是 B、编辑器里是 A 的代码"，
// 那正是多会话最容易误导人的一种错。
function switchEditor(set: (p: Partial<Store>) => void, get: () => Store, next: string | null) {
  const st = get()
  const prev = st.sessionId
  if (prev === next) return
  // 只给**还活着**的会话存现场：它已经没了（`session.removed` / 结束会话）就不该再写回去，
  // 否则"清现场"之后又被这次搬运重新创建出来（同 id 复用时还会被挖出来）。
  const prevAlive = !st.sessionsLoaded || st.sessionList.some((b) => b.id === prev)
  if (prev && prevAlive) set({ editor: { ...st.editor, [prev]: captureEditor(st) } })
  const saved = next ? (get().editor[next] || null) : null
  applyEditor(set, saved || emptyEditor())
}

// sleep 是"等后端状态自己走到下一步"的最小工具(登录、登记画面都要等)。
const sleep = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))

// waitSessionIdle 等这条会话**登录完**(loading → idle)。
//
// 为什么要等:建会话是立刻返回的，而"它能不能用"要等登录完（loading → idle）。
// 而 create 是立刻返回的 —— 登录是另一条异步路（服务端 go bootToIdle）。
// 不轮询就会立刻撞 409，用户看到的解释却是"会话不是空闲(loading)"，而不是"再等等"。
//
// 失败要分得清三种:登录失败(会话被服务端终结)、超时、其它。三种的处置完全不同。
async function waitSessionIdle(id: string, timeoutMs = 90000): Promise<void> {
  const deadline = Date.now() + timeoutMs
  for (;;) {
    let snap: any
    try {
      snap = await api.snapshot(id)
    } catch (e: any) {
      // 登录失败时服务端会 ForceExit + Remove，于是这条 id 当场查不到。
      throw new Error(`会话登录失败（服务端已结束这条会话）：${e?.message || String(e)}`)
    }
    const state = String(snap?.state || '')
    if (state === 'idle') return
    if (state === 'exit') {
      throw new Error('会话登录失败：后端把这条会话结束了，检查环境配置（服务器地址/区域）与服务器状态')
    }
    if (Date.now() > deadline) {
      throw new Error(`会话 ${Math.round(timeoutMs / 1000)} 秒内没登录完（现在状态：${state || '未知'}）——连不上就挂不了画面通道，先查服务器`)
    }
    await sleep(400)
  }
}


// refreshSessionList 列全部会话（`GET /api/sessions`）填进 `sessionList`，并**同步派生树**。
//
// 它**只取数、不动绑定** —— 这正是与旧 `syncFromSessions` 的分工差别：
// 从前那个函数兼管"替人重绑一条"，而绑定现在只能由人点。
// 树（`sessions`）在这里跟着池一起重建：树不是另一个事实，就是池的投影（W2/W6）。
async function refreshSessionList(): Promise<SessionBrief[] | null> {
  try {
    const { sessions } = await api.list()
    const list = sessions || []
    useStore.setState({ sessionList: list, sessions: indexFromList(list), sessionsLoaded: true })
    seedRuntimes(list)
    return list
  } catch {
    return null // 问不到就保持现状:重连与下一次事件还会再来
  }
}

// upsertBrief / dropBrief 是**事件驱动**的那半边：池变了不等人刷新。
//
// 收到就改 `sessionList` 并重建树 —— 与 refreshSessionList 同一条投影路径
// （`indexFromList`），所以"列表刷新"与"事件到达"不会长出两种树的形状。
function upsertBrief(b: SessionBrief) {
  if (!b || !b.id) return
  const st = useStore.getState()
  const next = st.sessionList.filter((x) => x.id !== b.id)
  next.push(b)
  useStore.setState({ sessionList: next, sessions: indexFromList(next) })
  patchRuntime(b.id, runtimeFromBrief(b, st.runtimes[b.id]))
}

function dropBrief(id: string) {
  if (!id) return
  const st = useStore.getState()
  // 列表里有它 → 摘掉（树跟着池重建）。
  if (st.sessionList.some((x) => x.id === id)) {
    const next = st.sessionList.filter((x) => x.id !== id)
    useStore.setState({ sessionList: next, sessions: indexFromList(next) })
  }
  // **无论列表里有没有**，它的运行时与编辑器现场都要清：这条事件本身就是"它没了"的
  // 权威通知；列表可能还没刷新到（或本来就断档），留着现场只会在 id 复用时被挖出来。
  dropRuntime(id)
}

// checkSelectedStillThere 只问一件事：**本地选的那条还在不在**。
//
// 不在就**回到未选择**（而不是挑一条）。别名刻意叫 check 而不是 adopt：
// 这个函数的全部职责就是"敢让绑定为空"，一旦它开始挑一条，Q4 那条硬规矩就废了。
async function checkSelectedStillThere() {
  const bound = useStore.getState().sessionId
  const list = await refreshSessionList()
  if (!list) return
  if (!bound) return
  if (!list.some((s) => s.id === bound)) setBoundSession(null)
}

// 本地绑的会话被换掉时,别每个事件都去问一次服务端(一次 list 是一次网络往返):
// 攒到一个窗口后再问,由它决定要不要解绑。窗口内的后续事件照常进时间线,不丢信息。
let rebindTimer: number | undefined
const REBIND_CHECK_MS = 1500

function scheduleRebindCheck() {
  if (rebindTimer) return
  rebindTimer = window.setTimeout(() => {
    rebindTimer = undefined
    void checkSelectedStillThere()
  }, REBIND_CHECK_MS)
}

// alignSelected 按**人点的那条会话**（或本地恢复的那条）对齐界面：模式、状态、快照、源码。
//
// 它是 `selectSession` / `syncFromSessions` / `adoptExisting` 三处的**同一个落点** ——
// 三处各写一遍必然漂（从前 `syncFromSessions` 与 `adoptExisting` 就是两份几乎一样的代码）。
//
// 两种取数，按顺序：
//
//	① `sessionList` 里有它 → 列表就够（模块/作业/状态/环境/模式），并据此判断要不要取快照；
//	② 列表里没有（还没列过 / 刚起的会话还没刷到）→ **仍然取一次快照**：
//	   绑定的语义是"人点了这条"，不是"列表里有这条"，所以不能因为列表没刷到就只绑一个 id
//	   把状态留在空上（实测：那样"点一条停站的会话"会得到空白的调试页）。
//
// `setBoundSession` 不在这里做：绑不绑是调用方的事，这里只负责"绑上之后长什么样"。
//
// 它是 **async**（等那次快照落定）—— 调用方 `await` 它，于是"选中一条会话"这个动作
// 在 Promise 落定时就已经对齐完，测试与后续逻辑都能当它是同步的。
// waitSessionStateSettled 等一条**还在登录**的会话落定（idle / stopped / running / exit）。
//
// 为什么需要它：`selectSession` 的对齐在一次页面生命周期里只做一次；若那一次拿到的
// 状态是 `loading` 就直接返回，登录完成后界面没有任何东西会再来问 —— 用户看到的就是
// 永远停在「启动中…」（实测 2026-10-19）。这里用快照轮询把它推到落定为止。
async function waitSessionStateSettled(get: () => Store, id: string, timeoutMs = 90000): Promise<boolean> {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 750))
    if (get().sessionId !== id) return false
    try {
      const snap: any = await api.snapshot(id)
      if (get().sessionId !== id) return false
      const state = String(snap?.state || '')
      if (state && state !== 'loading') {
        void refreshSessionList()
        return true
      }
    } catch {
      // 会话可能刚被收掉：交给下一轮判断（列表/事件会让绑定失效）
      const list = await refreshSessionList()
      if (list && !list.some((x) => x.id === id)) return false
    }
    // 列表里的人也可能先变（WS 事件已经把它更新成 idle），这里顺手刷新一次
    void refreshSessionList()
  }
  return false
}

async function alignSelected(set: (p: Partial<Store>) => void, get: () => Store, id: string) {
  let s = useStore.getState().sessionList.find((x) => x.id === id)
  // 树里那条（人点的那条）还记着建会话时点名的作业。服务端对**没起过作业的空闲会话**
  // 是不知道作业编号的（`POST /sessions/new` 只建宿主），所以这里以树为准补上 ——
  // 否则会话页写"作业 (空)"，而人明明填过（用户 2026-10-18 的截图就是这一格）。
  const item = useStore.getState().sessions.find((x) => x.id === id)
  if (s) {
    set({
      module: s.module, prog: s.prog || item?.prog || '', runProg: s.runProg || s.prog || item?.prog || '',
      state: s.state, sessionEnv: s.env || '', mode: modeOf(s.mode),
    })
    if (s.state === 'idle') {
      // 空闲宿主(idle)：先清现场（不清的话界面上是 A 的代码、绑的是 B，最误导），
      // 但**不要在这里 return** —— `sessionList` 是列表那一刻的快照，可能已经过期：
      // 会话早已停在断点上，列表还写着 idle。这时若直接定案成"空闲"，工具栏的
      // 继续/步过/步入会整排变灰，而右侧还写着"会话空闲"（实测：鼎捷2 明明停在
      // aglq510 上，界面却点不动）。所以继续往下走一次 snapshot，由它定案。
      set({
        started: false, stop: null, currentLine: 0, breakpoints: [], frames: [],
        autovars: [], selectedFrame: -1, holdingSeconds: 0, loadingSource: false,
        sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false,
      })
    }
    if (s.state === 'loading') {
      // 还在登录：**自己轮询到它落定**，再走下面的常规对齐。
      // 从前这里直接 return —— 而对齐在一次页面生命周期里只做一次，登录完成后
      // 没有任何东西会再来刷新状态，界面就永远停在「启动中… / 连接中…」（实测 2026-10-19）。
      set({ loadingSource: false })
      const settled = await waitSessionStateSettled(get, id)
      if (!settled) return
      s = useStore.getState().sessionList.find((x) => x.id === id) || s
      if (s.state === 'idle') {
        set({
          state: 'idle', module: s.module, prog: s.prog || item?.prog || '',
          runProg: s.runProg || s.prog || item?.prog || '', sessionEnv: s.env || '',
          started: false, stop: null, currentLine: 0, breakpoints: [], frames: [],
          autovars: [], selectedFrame: -1, holdingSeconds: 0, loadingSource: false,
          sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false,
        })
        return
      }
    }
  }
  try {
    const snap: any = await api.snapshot(id)
    if (get().sessionId !== id) return // 期间人家又点了一条:这一轮结果作废
    // **不回退到 `get().x`**：快照没给就是没有，拿上一条会话的值兜底正是"切到 B 却显示 A
    // 的作业/环境"的来源（多会话下 A 的值一直有效，于是错得很安静）。
    set({
      state: snap.state || '',
      module: snap.module || '',
      prog: snap.prog || '',
      runProg: snap.runProg || snap.prog || '',
      sessionEnv: snap.env || '',
      stop: snap.stop, breakpoints: snap.breakpoints || [], started: !!snap.started,
      currentLine: snap.stop?.line ?? 0, holdingSeconds: snap.holdingSeconds || 0,
      watchdogSeconds: snap.watchdogSeconds || 0, watchdogDisabled: !!snap.watchdogDisabled,
      // 快照里的模式比列表更新（选中期间可能刚被切换过）
      mode: modeOf(snap.mode),
    })
    // 顺带纠正这条会话的运行时索引：列表快照可能过期（会话早已停站，列表还写着 idle），
    // 而 runtimeFromBrief 按"事件比列表新"的假设**不用 brief 覆盖已有值** —— 于是树上
    // 那个状态点会一直停在建会话时的样子。快照是刚取的，拿它把这条会话的索引对齐。
    patchRuntime(id, {
      state: snap.state || '',
      prog: snap.prog || '',
      module: snap.module || '',
      stop: snap.stop || null,
      mode: modeOf(snap.mode),
    })
    // 空闲会话**必须清掉上一条的源码**：它没有停站，下面不会走 refreshSource，
    // 不清就一直是 A 的代码配 B 的标题（实测：鼎捷2 里显示 aglq510）。
    // 与上面 idle 分支同一套字段，两处都得清 —— 只清一处会漏掉"列表还没刷到这条"的路径。
    if (snap.state === 'idle') {
      set({
        started: false, stop: null, currentLine: 0, breakpoints: [], frames: [],
        autovars: [], selectedFrame: -1, holdingSeconds: 0, loadingSource: false,
        sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false,
      })
    }
    if (snap.state === 'stopped') {
      // 有停站就去取源码。入口停站可能不带文件名(WS 服务程序在 gzzz_t 里解析不到实体程序),
      // refreshSource 会按「模块_程序.4gl」兜底合成 —— 这是它已有的能力。
      await get().refreshSource(snap.stop?.file, snap.stop?.line)
      if (!snapTimer) startHoldTimer(set, get)
    }
    if (get().stackAuto) void get().refreshFrames()
    void get().refreshWatches()
    alignAutoPrefs(set, get) // 换会话:自动变量开关对齐 + 补拉最近一次求值结果
  } catch { /* 会话瞬态:下一次对齐还会来 */ }
}

function now() { return new Date().toLocaleTimeString('zh-CN', { hour12: false }) }

// ---------- 事件去重与停站落位节流 ----------
//
// seenSeq/seenEpoch 是模块级而非 store 字段:每次事件都 set() 会引发无谓重渲染。
// WS 开场补发的那批历史会与已有时间线重叠,靠序号去重;服务端重启后序号归零,
// 靠 epoch 变化来重置游标。
let seenSeq = 0
let seenEpoch = ''

// 停站落位的尾随去抖:AI 快速连续单步时把多次位置变化合并成最后一次落位,
// 免得代码画面逐行闪。
// **只节流渲染,不节流事件** —— 每一步都照常进时间线,历史一条不少;
// 单步慢的时候(continue 命中断点、单次 next)窗口内只有一个目标,仍然立即落位。
let revealTimer: number | undefined
let pendingReveal: { file?: string; line: number } | null = null
const REVEAL_DEBOUNCE_MS = 150

function scheduleReveal(set: (p: Partial<Store>) => void, get: () => Store, file: string | undefined, line: number) {
  pendingReveal = { file, line }
  if (revealTimer) clearTimeout(revealTimer)
  revealTimer = window.setTimeout(async () => {
    revealTimer = undefined
    const target = pendingReveal
    pendingReveal = null
    if (!target) return
    // 只在调试页时跟随:用户正在浏览别的文件时不抢他的视线(时间线仍记录,可点回)
    if (get().activeTab !== 'debug') {
      set({ loadingSource: false })
      return
    }
    // 需要重新取源码的两种情形:跨文件,或这个文件上次没取到(见 sourceMissing)。
    // 条件必须与 refreshSource 的同文件早退保持一致 —— 否则两条路径对"要不要加载"
    // 的判断相反:这边认定不用加载(于是只落光标),那边却因为 loadingSource 还是 true
    // 一直转圈,代码永远出不来。
    if (target.file && (target.file !== get().sourceDVM || get().sourceMissing)) {
      await get().refreshSource(target.file, target.line)
    } else if (target.line) {
      set({ loadingSource: false })
      get().reveal('debug', target.line)
    } else {
      set({ loadingSource: false })
    }
  }, REVEAL_DEBOUNCE_MS)
}

// 跳到 MAIN 语句行(无停站位置时给用户一个可点断点的起点)
function jumpToMain(set: (p: Partial<Store>) => void, get: () => Store) {
  if (get().currentLine > 0) return
  const lines = get().sourceContent.split('\n')
  for (let i = 0; i < lines.length; i++) {
    if (/^\s*MAIN\b/i.test(lines[i])) { set({ currentLine: i + 1 }); break }
  }
}

// 接口日志报文缓存(模块级,不进 store state:预取成功不该触发重渲染)。
// 一次 rowid 的报文内容(= 请求 + 响应)不会再变(同一次调用的流水),故只增不减即可,
// 仅保留最近 N 条防内存无界。
const WSLOG_CACHE_MAX = 30
const wsLogContentCache = new Map<string, WSLogContent>()
// 在途请求合并:同一 rowid 并发点击/预取只发一次(否则悬停+点击会打两次 SSH+sqlplus)
const wsLogContentInflight = new Map<string, Promise<WSLogContent>>()


// 启动看门狗：点了「运行」之后一直停在 loading 的兜底（POST 超时/作业起不来/GDC 没起）。
// 120 秒只在时间线提醒一次；240 秒仍没落定就退出 loading 状态并给出可读的失败原因 ——
// 界面不会永远停在「启动中…」。
let launchStartedAt = 0
let launchWarned = false
// 接口日志查询的"请求代次"。同上:翻页/改条件会叠出多个在途查询。
let wsLogsGen = 0

// ---------- 接口日志:按环境分槽 ----------
//
// 扁平字段(wsLogs/wsLogSel/wsLogErr/…)只代表"当前环境那一页";每次写都镜像进
// wsLogsByEnv,切环境时整组换出/换入(见 setActiveEnv)。
//
// 为什么要这样:环境页 keep-alive,而日志结果只有一个槽 —— 切到乙环境时页面还挂着
// 甲环境的行(2026-10-23 实测:测试区/正式区的接口日志都显示成鼎捷的)。
// 另一种"切环境就清空"的做法虽然不会看错,但会把用户刚查好的结果扔掉;
// 这里按"一个站点一份、互不覆盖"来。
export type WsLogSnap = {
  items: WSLogItem[]
  page: number
  hasMore: boolean
  sel: WSLogItem | null
  content: WSLogContent | null
  tab: 'info' | 'request' | 'response'
  err: string
}

const emptyWsLogSnap = (): WsLogSnap => ({
  items: [], page: 1, hasMore: false, sel: null, content: null, tab: 'info', err: '',
})

type WsLogFlat = Pick<Store, 'wsLogs' | 'wsLogsPage' | 'wsLogsHasMore' | 'wsLogSel' | 'wsLogContent' | 'wsLogTab' | 'wsLogErr'>

const wsLogSnapOf = (s: WsLogFlat): WsLogSnap => ({
  items: s.wsLogs, page: s.wsLogsPage, hasMore: s.wsLogsHasMore,
  sel: s.wsLogSel, content: s.wsLogContent, tab: s.wsLogTab, err: s.wsLogErr,
})

const wsLogFlatOf = (sn: WsLogSnap) => ({
  wsLogs: sn.items, wsLogsPage: sn.page, wsLogsHasMore: sn.hasMore,
  wsLogSel: sn.sel, wsLogContent: sn.content, wsLogTab: sn.tab, wsLogErr: sn.err,
})

// 企业目录(ENT → 账号)按环境的客户端缓存。
//
// 它只是“别同一页反复问”的便利层:真值在服务端(内存 + 快照 + gzou_t),
// 新鲜度以服务端回的 source/stale 为准,这里记一个本地时间只用于过期判定。
interface EntsView {
  items: EntItem[]
  loading: boolean
  err: string
  source: string // live | snapshot | snapshot-stale(来自服务端)
  stale: boolean
  fetchedAt: string
  at: number // 本地拿到这份答案的时间(毫秒);0 = 没拿到过
}
const EMPTY_ENTS: EntsView = {
  items: [], loading: false, err: '', source: '', stale: false, fetchedAt: '', at: 0,
}
const ENTS_TTL_MS = 5 * 60_000

// 会话绑定的"代次":每次**权威地绑上一个会话**（launch / replayStart / 接管）自增。
//
// 为什么需要它：`syncFromSessions` 与 `adoptExisting` 都是异步的（先 api.list() 再 api.snapshot()），
// 而在这两次 await 期间用户可能已经点了「启动调试」绑上了新会话 —— 那一轮的结果早已不新鲜，
// 写回去就是把刚绑上的会话抹成 null（实测：慢一拍的 list() 让新会话看起来"没起来"）。
// 判据只有一条：进函数时记下代次，每个 await 之后比一次，对不上就整轮丢掉。
let bindEpoch = 0

// isSoapMode 判这种接口方式是不是 SOAP(wsfc001 映射,与 awsq990 一致)。
//
// 它必须只有**一处**:服务测试页拿它决定"SOAP 勾选框显不显示",store 拿它决定
// "换了方式要不要把勾选收回去" —— 两处各写一份,迟早会出现"界面看不见、请求里却生效"
// 的开关(这正是实测踩到的那个 bug)。
export function isSoapMode(mode: string): boolean {
  return mode === '1' || mode === '2' || mode === '5'
}

function loadWsLogContent(item: WSLogItem): Promise<WSLogContent> {
  const hit = wsLogContentCache.get(item.rowid)
  if (hit) return Promise.resolve(hit)
  const flying = wsLogContentInflight.get(item.rowid)
  if (flying) return flying
  const p = api.wsLogContent(item.rowid, useStore.getState().activeEnv || undefined).then(({ content }) => {
    wsLogContentCache.set(item.rowid, content)
    if (wsLogContentCache.size > WSLOG_CACHE_MAX) {
      const oldest = wsLogContentCache.keys().next().value
      if (oldest) wsLogContentCache.delete(oldest)
    }
    return content
  }).finally(() => { wsLogContentInflight.delete(item.rowid) })
  wsLogContentInflight.set(item.rowid, p)
  return p
}

export const useStore = create<Store>((set, get) => ({
  wsConnected: false,
  // 刷新后恢复**人上次点的那条**（`tt.sessionId`）。恢复的是"选择"，不是"问服务端要一条"：
  // 这条会话若已经不在服务端了，`checkSelectedStillThere` 会把它清成"未选择"。
  sessionId: readStoredSessionId(), sessionList: [],
  envs: [], envsLoaded: false, envsErr: '', gpEnvs: [], gpEnvsLoaded: false, gpEnvsErr: '', settingsGpEnv: '', activeEnv: '', sessions: [], sessionsLoaded: false, editor: {}, runtimes: {}, module: '', prog: '', state: '', sessionEnv: '', started: false, stop: null, holdingSeconds: 0, watchdogSeconds: 0, watchdogDisabled: false,
  launching: false, launchError: '', showBottom: true, bottomMax: false,
  // 面板开关默认关(localStorage tt.stackAuto / tt.autovarsAuto 持久化)
  stackAuto: localStorage.getItem('tt.stackAuto') === '1',
  autovarsAuto: localStorage.getItem('tt.autovarsAuto') === '1',
  lastReplayRowid: null, lastReplayRequest: '',
  // 调试视图右侧那两块：默认只开调试面板（它是主场）；两块都开时上下的占比记在本地。
  rightDebugOpen: true,
  rightTreeOpen: false,
  rightTop: 'debug',
  // 右侧栏宽度(像素,夹取 240~640)。初值取持久化的那份,越界或没记过就用 320。
  // 放在 store 而不是某个组件的本地 state:浮动工具条按它算默认落点(见 Toolbar.tsx)。
  panelW: (() => {
    const v = Number(localStorage.getItem('tt.panelW'))
    return v >= 240 && v <= 640 ? v : 320
  })(),
  breakpoints: [], adjustedBps: {}, frames: [], watches: [], autovars: [], selectedFrame: -1, backendDead: '',
  mode: 'solo' as 'solo' | 'collab', inflight: null,
  timeline: [], rawLog: [],
  runProg: '',
  // 首屏落在设置页(而不是调试页):这个工具"先配环境才能干活",
  // 没有环境时调试页什么也做不了(App.tsx 里那条空环境兜底原本就是在补这件事)。
  // 已经有环境的用户也一样从这里开始 —— 配好之后活动栏第一个图标就是设置,
  // 想干活点「调试」只需往下一格。深链接(#debug、#wslogs…)仍然优先,见 App.tsx 的 parseHash。
  view: 'settings',
  settingsSection: 'sites',

  theme: initialTheme,
  dark: resolveDark(initialTheme),
  wsLogs: [], wsLogsLoading: false, wsLogsPage: 1, wsLogsHasMore: false,
  wsLogSel: null, wsLogContent: null, wsLogTab: 'info', wsLogErr: '',
  wsLogsByEnv: {},
  entsByEnv: {},

  sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false, currentLine: 0, loadingSource: false, lineOffset: 0,
  tabs: [], activeTab: 'debug', revealReq: null,

  setWsConnected: (b) => set({ wsConnected: b }),
  toggleBottom: () => set((st) => ({ showBottom: !st.showBottom, bottomMax: false })),
  // 右侧那两块：各自独立开关，**后开的在下面**（单独开一个时它占满整列）。
  // 三条规则都在这两个 action 里，界面上不需要知道顺序。
  toggleRightDebug: () => set((st) => {
    const open = !st.rightDebugOpen
    return { rightDebugOpen: open, rightTop: open && st.rightTreeOpen ? 'tree' : st.rightTop }
  }),
  toggleRightTree: () => set((st) => {
    const open = !st.rightTreeOpen
    return { rightTreeOpen: open, rightTop: open && st.rightDebugOpen ? 'debug' : st.rightTop }
  }),
  // 最大化/还原下方面板。收起下方面板时顺带解除最大化(否则再展开会以一个"占满整列"的
  // 面板回来,而用户要的是常规分栏)。
  //
  // 反过来也要成立:最大化**必须**把下方面板打开 —— 否则 `toggleBottom`（收起）之后
  // 再 `toggleBottomMax` 会得到 `showBottom=false && bottomMax=true` 这种自相矛盾的状态。
  // 界面上的可达路径走不到那里（穷举 87 条序列 0 违规），但 store 的不变量不该靠"用户点不到"来维持。
  toggleBottomMax: () => set((st) => ({
    bottomMax: !st.bottomMax,
    showBottom: st.bottomMax ? st.showBottom : true,
  })),

  setPanelW: (w) => set({ panelW: Math.min(640, Math.max(240, Math.round(w))) }),

  // 调用栈面板「自动」开关(默认关):开 → 停站后自动抓调用栈,已停站时立即抓一次
  toggleStackAuto: () => {
    const st = get()
    const on = !st.stackAuto
    localStorage.setItem('tt.stackAuto', on ? '1' : '0')
    set({ stackAuto: on })
    if (on && get().state === 'stopped') void get().refreshFrames()
  },
  // 自动变量面板「自动」开关(默认关):服务端停站后自动求值窗口变量才发生(见
  // 会话 SetAutovarsOn);开 → 下发当前会话(已停站时服务端立即补一次求值并推送)
  toggleAutovarsAuto: () => {
    const st = get()
    const on = !st.autovarsAuto
    localStorage.setItem('tt.autovarsAuto', on ? '1' : '0')
    set({ autovarsAuto: on })
    const sid = get().sessionId
    if (!sid) return
    void api.autovarsAuto(sid, on).catch(() => { /* 会话切换窗口:下次停站前由对齐逻辑重发 */ })
  },
  pushRaw: (line) => set((st) => {
    const log = st.rawLog.length > 3000 ? st.rawLog.slice(-2000) : st.rawLog
    return { rawLog: [...log, line] }
  }),

  // 直接执行 fgldb 命令(复刻原版 fgldeb Ctrl+D「Input Debugger Command」的透传):
  // 命令与输出原样进协议流;状态类命令执行后刷新快照/栈,与原版行为一致
  sendRaw: async (cmd) => {
    const { sessionId } = get()
    if (!sessionId || !cmd.trim()) return
    get().pushRaw(`> ${cmd}`)
    try {
      const r = await api.raw(sessionId, cmd)
      for (const l of r.lines) get().pushRaw(l)
      const head = cmd.trim().split(/\s+/)[0].toLowerCase()
      if (['step', 'next', 'continue', 'run'].includes(head)) {
        // 程序会继续运行:走轮询等下一次停站
        pollUntilStopped(set, get)
      } else if (['break', 'tbreak', 'clear', 'delete', 'enable', 'disable', 'where', 'frame', 'finish', 'return'].includes(head)) {
        void get().refreshSnapshot()
        if (get().state === 'stopped') void get().refreshFrames()
      }
    } catch (e: any) {
      get().pushRaw(`[错误] ${e.message || String(e)}`)
    }
  },

  pushTimeline: (item) => set((st) => ({
    timeline: [...st.timeline.slice(-500), { ...item, time: now() }],
  })),

  onEvent: (ev) => {
    const st = get()

    // ---- 池与界面会话的事件（W2）：**先于**"是不是我绑的那条"处理 ----
    //
    // 这四类回答的是"有哪些会话 / 它挂着画面没有"，与绑定无关：树是池的投影（D9），
    // 所以 CLI、别的浏览器、AI 建的会话都从这里进树。放在最前面还有个理由：
    // 下面的逻辑会把"别的会话的事件"当成"我绑的那条可能被换掉了"（scheduleRebindCheck），
    // 而池事件本来就该改索引，不该触发重绑。
    switch (ev.type) {
      case 'session.added': {
        const brief = (ev as any).brief as SessionBrief | undefined
        if (brief?.id) upsertBrief(brief)
        // 顺手校一次：事件是"叫一声"，权威是 List（断档/丢事件时靠它收敛）。
        void refreshSessionList()
        return
      }
      case 'session.removed': {
        const id = ev.sessionId
        dropBrief(id)

        // 我绑着的那条没了 → 回到未选择（绝不替人挑下一条）。
        if (id && get().sessionId === id) void get().selectSession(null)
        return
      }

    }

    // ---- 运行时：**任何**会话的状态都记，不只绑着的那条（W6） ----
    //
    // 这是"多会话"最直接的一步：别人（CLI / 另一个页面）起停的那条会话，它的状态点
    // 现在也能变。放在"是不是我绑的那条"之前 —— 否则整个记录会被下面的 return 吃掉。
    if (ev.sessionId) {
      switch (ev.type) {
        case 'state':
          patchRuntime(ev.sessionId, { state: ev.state || '' })
          break
        case 'stopped':
          patchRuntime(ev.sessionId, { state: 'stopped', stop: ev.stop || null })
          break
        case 'dead':
          patchRuntime(ev.sessionId, { state: 'exit' })
          break
      }
    }

    // **本地没选中就不认领**（WP-F / Q4 的硬规矩，见上面「会话绑定的取数纪律」）。
    //
    // 从前的写法是"事件带着会话 id 而本地还没绑定 → 认领它"，理由是"不认领的话编辑器
    // 永远转圈"。但那条便利的代价正是本批要根除的东西：**它等于向服务端问"当前是哪条"** ——
    // 别人（典型是 AI 从 CLI）刚建了会话，页面就自作主张绑上去；池里有多条时绑到哪一条
    // 取决于事件先到谁，于是"我盯着 A 的现场、界面却显示 B 的停站"。
    // 现在的口径：**列全部 + 等人点**（会话面板里点一条即选中）。
    // 人在面板上点一条之前，协议流照常进时间线，编辑器就是空的 —— 这是"未选择"该有的样子。
    //
    // 唯一例外是**本页面自己发起**的那次动作（launch / replayStart）：它们由
    // HTTP 响应权威绑定（见那两处的 set({sessionId: r.sessionId})），不经过这里。
    if (st.sessionId && ev.sessionId && ev.sessionId !== st.sessionId) {
      // 收到的是**另一条会话**的事件。多会话下这是常态（池里有 N 条），不再是"我绑的那条
      // 可能被换掉了"的信号 —— 它的运行时上面已经记过了，这里直接不动面板。
      //
      // 只有一个例外仍然值得问一次：事件来自一条**我不认识的会话**（池里没有它）。
      // 那说明索引断档（丢事件/服务端重启），此时按老规矩只问"我选的那条还在不在"。
      if (!st.sessionList.some((b) => b.id === ev.sessionId)) scheduleRebindCheck()
      return
    }
    // 刚点重放/重新开始:旧会话的收尾事件不得清掉新会话的加载态(见 retireID 注释)
    if (isRetiringTeardown(ev)) return
    // 序号去重:开场补发与流式之间可能重叠(服务端注释里说明了这点),
    // 同一序号只处理一次。无序号的事件(本地合成)照常处理。
    if (ev.seq) {
      if (ev.seq <= seenSeq) return
      seenSeq = ev.seq
    }
    switch (ev.type) {
      case 'output':
        st.pushRaw(ev.text || '')
        return
      case 'state':
        set({ state: ev.state || '' })
        if (ev.state === 'stopped') {
          // 停站后按面板开关自动刷新(异步,不阻塞):调用栈/自动变量默认关,监视保持
          if (get().stackAuto) void get().refreshFrames()
          void get().refreshWatches()
          startHoldTimer(set, get)
          // 入口停站(程序还没 run)只发 state=stopped,**没有 stopped 事件** ——
          // 停站位置只在快照里。少了这一句,刚绑上会话的页面会停在"已停站但编辑器空白"。
          void get().syncStopFromSnapshot()
        }
        if (ev.state === 'running' || ev.state === 'idle' || ev.state === 'exit') {
          if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
          stopHoldTimer()
          if (ev.state === 'running') set({ selectedFrame: -1 })
        }
        if (ev.state === 'idle') {
          // 本轮调试结束,宿主会话保留(idle):清运行现场,源码保留便于浏览/再次启动
          set({ started: false, stop: null, currentLine: 0, loadingSource: false, selectedFrame: -1, autovars: [], breakpoints: [], frames: [], adjustedBps: {} })
        }
        if (ev.state === 'exit') {
          st.pushTimeline({ origin: 'system', kind: 'warn', text: '会话已断开(结束会话/切换环境或连接中断)' })
          set({ loadingSource: false, currentLine: 0, started: false, stop: null })
          // 后端已移除该会话:按服务端现状对齐绑定(有常驻/新会话则重绑,没了即清),
          // 避免 sessionId 残留导致界面一直停在"会话进行中"(如 wslogs 重放被锁)
          void get().syncFromSessions()
        }
        return
      case 'dead':
        // 后端死亡/SSH 断开:forceExit 随后会发 state=exit
        set({ backendDead: ev.text || '调试后端连接断开' })
        st.pushTimeline({ origin: 'system', kind: 'warn', text: ev.text || '调试后端连接断开' })
        return
      case 'autovars':
        set({ autovars: ev.vars || [] })
        return
      case 'stopped': {
        // 跨文件停站:先收光标(源码未就位时不画停站行,防它在旧文件上错位),
        // 拉到新源码后再一次性落位
        const nl = !!ev.stop?.file && (ev.stop.file !== get().sourceDVM || get().sourceMissing)
        // 只在调试页时跟随:用户正在浏览别的文件就**不抢他的视线**(仍记时间线,可点回)
        const follow = get().activeTab === 'debug'
        set({
          stop: ev.stop || null, state: 'stopped',
          currentLine: nl ? 0 : (ev.stop?.line || get().currentLine),
          selectedFrame: -1, loadingSource: follow && nl,
          ...(follow ? { activeTab: 'debug' as const } : {}),
        })
        // 时间线不节流:每一步都记,历史一条不少
        st.pushTimeline({
          origin: 'system', kind: 'stop',
          text: `停站[${ev.stop?.reason}] ${ev.stop?.file || ''}:${ev.stop?.line ?? ''} ${ev.stop?.func || ''}`,
        })
        // 停站文件缺失(如 SIGINT 中断块无文件头)时,用栈顶帧的真实文件跟随
        void (async () => {
          let file = ev.stop?.file
          let frameLine = 0
          if (!file && get().stackAuto) {
            const frames = await get().refreshFrames()
            file = frames[0]?.file
            frameLine = frames[0]?.line ?? 0
          }
          // 落位走尾随去抖(只节流渲染):AI 连续单步时合并到最终位置
          scheduleReveal(set, get, file, frameLine || ev.stop?.line || 0)
          await get().refreshWatches()
          await get().refreshSnapshot()
        })()
        startHoldTimer(set, get)
        return
      }
      case 'watchdog-approaching':
        st.pushTimeline({ origin: 'system', kind: 'warn', text: ev.text || '停站接近上限,稍后会自动放行' })
        return
      case 'watchdog-failed':
        // 看门狗自己没成功 = 锁还在、没人管。必须显眼, 不能只是时间线里一行灰字。
        st.pushTimeline({ origin: 'system', kind: 'error', text: ev.text || '看门狗自动继续失败, 程序可能仍在持锁' })
        void get().refreshSnapshot()
        return
      case 'watchdog':
        st.pushTimeline({ origin: 'system', kind: 'warn', text: ev.text || '看门狗触发' })
        return
      case 'ai_action':
        st.pushTimeline({ origin: 'ai', kind: 'command', text: ev.text || '' })
        // 断点变更与模式切换会改变会话状态,拉一次快照让它们立刻反映到界面上
        // (断点列表只从快照取 —— AI 下的断点要马上出现在 gutter 里)
        if (ev.action?.startsWith('bp.') || ev.action === 'session.mode') void get().refreshSnapshot()
        return
      case 'log': {
        const text = ev.text || ''
        st.pushTimeline({ origin: 'system', kind: 'info', text })
        // 后端异步启动失败(HTTP 已返回 200,失败发生在 goroutine):会话随即被移除,
        // 必须立刻退出加载态、清掉会话引用并给出显眼报错,否则编辑器永久转圈、
        // 运行区还停留在"会话进行中"而无法再次启动
        if (text.startsWith('启动失败')) {
          if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
          stopHoldTimer()
          setBoundSession(null)
          set({ launchError: text, state: '', loadingSource: false, stop: null, currentLine: 0 })
        }
        return
      }

    }
  },

  // WS 开场补发:把历史填进时间线,让刷新页面后仍能看到"刚才发生了什么"。
  // **只追加时间线,不触发任何副作用** —— 不刷源码/栈、不起计时器;页面状态
  // 另有 refreshSnapshot 负责,两者不打架。序号去重与 onEvent 共用同一游标,
  // 所以重连时重复补发是幂等的。
  onReplay: (ev) => {
    const st = get()
    if (st.sessionId && ev.sessionId && ev.sessionId !== st.sessionId) return
    if (ev.seq) {
      if (ev.seq <= seenSeq) return
      seenSeq = ev.seq
    }
    switch (ev.type) {
      case 'stopped':
        st.pushTimeline({
          origin: 'system', kind: 'stop',
          text: `停站[${ev.stop?.reason}] ${ev.stop?.file || ''}:${ev.stop?.line ?? ''} ${ev.stop?.func || ''}`,
        })
        break
      case 'ai_action':
        st.pushTimeline({ origin: 'ai', kind: 'command', text: ev.text || '' })
        break
      case 'log':
        st.pushTimeline({ origin: 'system', kind: 'info', text: ev.text || '' })
        break
      case 'watchdog-approaching':
        st.pushTimeline({ origin: 'system', kind: 'warn', text: ev.text || '停站接近上限,稍后会自动放行' })
        return
      case 'watchdog-failed':
        // 看门狗自己没成功 = 锁还在、没人管。必须显眼, 不能只是时间线里一行灰字。
        st.pushTimeline({ origin: 'system', kind: 'error', text: ev.text || '看门狗自动继续失败, 程序可能仍在持锁' })
        void get().refreshSnapshot()
        return
      case 'watchdog':
        st.pushTimeline({ origin: 'system', kind: 'warn', text: ev.text || '看门狗触发' })
        break
    }
  },

  setView: (v) => set({ view: v }),
  setSettingsSection: (s) => set({ settingsSection: s }),

  setLaunchError: (msg) => set({ launchError: msg }),
  // 外观:选择 → 持久化 + 立即应用(跟随系统时 dark 取当前系统偏好)
  setTheme: (t) => {
    const dark = resolveDark(t)
    localStorage.setItem(THEME_KEY, t)
    applyDark(dark)
    set({ theme: t, dark })
  },

  setActiveTab: (key) => set({ activeTab: key }),

  // 定位信号(唯一驱动视口滚动)。
  // nav=true = 纯浏览跳转(断点列表点击):只滚视口,不动 currentLine(黄色停站高亮仍留在
  // 程序真正停的那行),也不受"仅停站可定位"的调试门限制——与大纲点击同一语义。
  reveal: (key, line, nav = false) => {
    if (!(line > 0)) return
    set((st) => ({
      revealReq: { key, line, nav, seq: (st.revealReq?.seq ?? 0) + 1 },
      currentLine: !nav && key === 'debug' ? line : st.currentLine,
    }))
  },

  locate: async (word) => {
    const { sessionId } = get()
    if (!sessionId) throw new Error('无调试会话')
    return api.locate(sessionId, word)
  },

  // 行号校准:检测 fgldb(DVM)行号与磁盘源码的偏移,前插 offset 行让 Monaco 行号对齐协议流。
  // 用户在协议流与源码对不上时主动触发(不同文件/编译版本的偏移可能不同)
  calibrate: async () => {
    const { sessionId, state } = get()
    if (!sessionId) return
    if (state !== 'stopped') {
      get().pushTimeline({ origin: 'system', kind: 'warn', text: '行号校准:需停站后才能检测偏移' })
      return
    }
    try {
      const { offset } = await api.calibrate(sessionId)
      if (offset < 0) {
        get().pushTimeline({ origin: 'system', kind: 'warn', text: `行号校准:源码比编译版本多 ${-offset} 行,无法前插对齐,请在服务器重新编译或核对源码版本` })
        return
      }
      set({ lineOffset: offset })
      get().pushTimeline({ origin: 'human', kind: 'info', text: offset > 0 ? `行号校准完成:协议行号比源码多 ${offset} 行,已前插对齐` : '行号校准完成:行号无偏移' })
      const st = get()
      if (st.stop?.line) get().reveal('debug', st.stop.line)
    } catch (e: any) {
      get().pushTimeline({ origin: 'system', kind: 'warn', text: `行号校准失败: ${e.message || String(e)}` })
    }
  },

  closeTab: (key) => set((st) => {
    const idx = st.tabs.findIndex((t) => t.key === key)
    if (idx < 0) return {}
    const tabs = st.tabs.filter((t) => t.key !== key)
    if (st.activeTab !== key) return { tabs }
    return { tabs, activeTab: tabs[idx] ? tabs[idx].key : tabs[idx - 1] ? tabs[idx - 1].key : 'debug' }
  }),

  openSourceTab: async (file, line) => {
    const key = 'src:' + file
    const st = get()
    const existing = st.tabs.find((t) => t.key === key)
    if (existing) {
      // 已开:激活;带行号才重新定位(纯激活保持离开时视口)
      set({ activeTab: key, tabs: line ? st.tabs.map((t) => t.key === key ? { ...t, line } : t) : st.tabs })
      if (line) get().reveal(key, line)
      return
    }
    if (!st.sessionId) return
    const tab: SrcTab = { key, file, content: '', path: '', line, loading: true }
    set({ tabs: [...st.tabs, tab], activeTab: key })
    try {
      const { source } = await api.sourceByFile(st.sessionId, file, st.module)
      set((s2) => ({ tabs: s2.tabs.map((t) => t.key === key ? { ...t, content: source.content, path: source.path, loading: false } : t) }))
      if (line) get().reveal(key, line)
    } catch {
      // 无源码(只有 42m 等):页签保留并标记,内容区给提示
      set((s2) => ({ tabs: s2.tabs.map((t) => t.key === key ? { ...t, loading: false, missing: true } : t) }))
    }
  },

  // loadEnts 取一个环境的企业目录(ENT → 数据库账号)。接口日志页与建会话的 ENT 下拉共用。
  //
  // 不需要活会话:数据来自服务端的 gzou_t(固定用系统账号 ds 读),与当前有没有会话无关,
  // 也不用先建一条。失败**不抛**:下拉需要一个“失败态 + 重试”的展示位,
  // 而不是把整页炸掉 —— 错误写进 entsByEnv[env].err,调用方就地显示。
  loadEnts: async (env, force) => {
    const key = (env || get().activeEnv || '').trim()
    if (!key) return []
    const cur = get().entsByEnv[key]
    // 客户端缓存只挡“同一页反复问”;真正的缓存与新鲜度在服务端(内存 + 快照,TTL 10 分钟)。
    if (!force && cur && cur.items.length && Date.now() - cur.at < ENTS_TTL_MS) return cur.items
    if (cur?.loading) return cur.items
    set((s) => ({ entsByEnv: { ...s.entsByEnv, [key]: { ...(cur || EMPTY_ENTS), loading: true, err: '' } } }))
    try {
      const r = await api.ents(key, { refresh: !!force })
      const items = (r.ents || [])
        .map((e) => ({ ent: Number(e.ent) || 0, account: (e.account || '').trim(), placeholder: !!e.placeholder }))
        .filter((e) => e.ent > 0)
        .sort((a, b) => a.ent - b.ent)
      set((s) => ({
        entsByEnv: {
          ...s.entsByEnv,
          [key]: {
            items, loading: false, err: '', source: r.source || '',
            stale: !!r.stale, fetchedAt: r.fetchedAt || '', at: Date.now(),
          },
        },
      }))
      return items
    } catch (e: any) {
      const msg = e?.message || String(e)
      set((s) => ({
        entsByEnv: { ...s.entsByEnv, [key]: { ...(s.entsByEnv[key] || EMPTY_ENTS), items: [], loading: false, err: msg, at: 0 } },
      }))
      return []
    }
  },

  loadWsLogs: async (q: WSLogQuery) => {    // 并发闸:一次查询只允许在途一次。按钮上有 disabled，但**同一个事件循环里连点**时
    // React 还没来得及重渲染，disabled 拦不住（实测 5 连点发出 5 个请求）。
    // 丢掉重复请求的代价只是"这一下没反应"，比叠出多个在途查询好。
    if (get().wsLogsLoading) return
    // 请求代次:翻页/改条件时会有多个查询在途,先发的慢响应回来时**不许**覆盖后发的结果。
    // 实测过一次:慢查询（空结果）落地把已经拿到的 6 行冲成 0 条 +「暂无日志记录」，
    // 而输入框里写着的是新条件 —— 用户看到的是一次并不存在的查询。
    const gen = ++wsLogsGen
    // 这次查的是**哪个环境**：响应回来时若用户已经切走，结果只能写进那个环境的槽，
    // 不许落在当前页上（否则又变成"乙环境显示甲环境数据"）。
    const targetEnv = (q.env || get().activeEnv || '').trim()
    set({ wsLogsLoading: true, wsLogErr: '' })
    try {
      // 带上"我在哪个环境"：统一服务启动时可能还没有生效环境（环境一律点名），
      // 而后端的接口日志要连 SSH —— 不点名就会报 "没有生效环境"。
      const r = await api.wsLogs({ ...q, env: q.env || get().activeEnv || undefined })
      if (gen !== wsLogsGen) return
      const patch: Partial<WsLogSnap> = {
        items: r.items || [], page: q.page ?? 1, hasMore: !!r.hasMore, sel: null, content: null, err: '',
      }
      if (!targetEnv || get().activeEnv === targetEnv) {
        get().patchWsLog(patch)
      } else {
        // 已经切到别的环境了:只更新它自己的那份。
        const st = get()
        set({
          wsLogsByEnv: {
            ...st.wsLogsByEnv,
            [targetEnv]: { ...(st.wsLogsByEnv[targetEnv] || emptyWsLogSnap()), ...patch },
          },
        })
      }
    } catch (e: any) {
      if (gen !== wsLogsGen) return
      // 失败**不清上一次的结果**：那条路会让界面显示成"暂无日志记录"，与"确实没有数据"
      // 一字不差（实测真 500 现场）。也不动选中行/明细面板 —— 成功清、失败留，两副面孔最费解。
      // 错误必须**留在界面上**（wsLogErr），不能只发一条 8 秒后消失的 toast。
      const msg = e?.message || String(e)
      // 失败也要按**发起时的环境**写槽：`patchWsLog` 固定写当前环境，如果用户在这段时间
      // 里切走了，乙环境的报错就会甩到甲环境的页面上，而且从此写进甲那一槽。
      const patch: Partial<WsLogSnap> = { err: msg }
      if (!targetEnv || get().activeEnv === targetEnv) {
        get().patchWsLog(patch)
      } else {
        const st = get()
        set({
          wsLogsByEnv: {
            ...st.wsLogsByEnv,
            [targetEnv]: { ...(st.wsLogsByEnv[targetEnv] || emptyWsLogSnap()), ...patch },
          },
        })
      }
      toast.error('查询接口日志失败', { description: msg })
    } finally {
      // 只有最新那一轮才许把 loading 关掉:否则慢请求还在途时转圈就没了，
      // 用户以为查完了，接着点，叠出更多竞态。
      if (gen === wsLogsGen) set({ wsLogsLoading: false })
    }
  },

  // 把"当前环境那一页"重置成空白(不动别的环境的槽)。
  // 推进代次:在途的旧查询回来时不许再覆盖。
  resetWsLogs: () => {
    wsLogsGen++
    get().patchWsLog(emptyWsLogSnap())
    set({ wsLogsLoading: false })
  },

  // 写"当前环境这一页"并镜像进 wsLogsByEnv —— 所有对扁平字段的写入都必须走这里,
  // 否则切回来时看到的是旧快照(只写扁平不写槽 = 分槽失效)。
  patchWsLog: (patch) => {
    const env = get().activeEnv
    set((st) => {
      const flat = { ...wsLogSnapOf(st), ...patch }
      return {
        ...wsLogFlatOf(flat),
        wsLogsByEnv: env ? { ...st.wsLogsByEnv, [env]: flat } : st.wsLogsByEnv,
      }
    })
  },

  // 点击行即取报文(接口一次返回 request+response,不是切页签才取)。
  // 命中预取缓存时瞬时显示;同一行的并发请求会被合并(见 loadWsLogContent)。
  selectWsLog: async (item) => {
    const cached = wsLogContentCache.get(item.rowid)
    get().patchWsLog({ sel: item, content: cached ?? null, err: '' })
    if (cached) return
    try {
      const content = await loadWsLogContent(item)
      // 期间可能已切到别的行:只在仍是当前选中时写回
      if (get().wsLogSel?.rowid === item.rowid) get().patchWsLog({ content })
    } catch (e: any) {
      if (get().wsLogSel?.rowid === item.rowid) {
        toast.error('读取报文详情失败', { description: e?.message || String(e) })
      }
    }
  },

  // 悬停预取:鼠标在行上停留一小会儿就开始取报文,等真点下去时通常已经就绪。
  // 已缓存/在途则直接返回,不产生重复请求。
  prefetchWsLog: (item) => {
    if (wsLogContentCache.has(item.rowid) || wsLogContentInflight.has(item.rowid)) return
    void loadWsLogContent(item)
      .then((content) => {
        if (get().wsLogSel?.rowid === item.rowid) get().patchWsLog({ content })
      })
      .catch(() => { /* 预取失败不打扰用户:真点下去时会再报错 */ })
  },

  setWsLogTab: (t) => get().patchWsLog({ tab: t }),

  // 关闭日志详情面板:回到"只显示列表"的默认态
  closeWsLogDetail: () => get().patchWsLog({ sel: null, content: null }),

  replayDebug: async (item, request, ent) => {
    // 把「这一页所属的环境」一起带上：日志列表那条路早就送了（`loadWsLogs` 的 `env`），
    // 只有重放漏了 —— 而重放要拨号读 `wsfa_t` 并**起一条会话**，环境必须点名
    // （统一服务刚启动时没有生效环境，不点名就是一句"没有生效环境"）。
    // ent 同理：它是这一轮跑在哪个企业的唯一来源，漏掉就等于沿用登录 shell 的 TOPENT。
    await get().replayStart(item.rowid, request, ent, get().activeEnv)
  },

  // 接口日志重放启动:立即切到 debug 页进 loading,再请求后端重放该日志
  // (后端按 rowid 重读报文并落临时文件,报文参数在 ArgsOverride 里,随会话保留);
  // request 非空 = 用界面改过的入参(后端写临时文件后优先使用)
  replayStart: async (rowid, request, ent, env) => {
    // 已有会话在跑:先结束它(一次只能调一个作业;后端重放也会收口,双保险)。
    // ID 记下来:后面还要用它发 quit,也是收口窗口的判定依据
    const old = get().sessionId
    beginRetire(old)
    // 一进 debug 页就进加载态:后端要 SSH 登录 + 解析作业 + 重读报文才返回(秒级),
    // 若等响应回来才置 loadingSource,这段时间编辑器既没内容也不转圈,像卡住。
    // 同时清掉上一会话的调试现场,避免"加载中"的画面里残留旧的时间线/调用栈
    setBoundSession(null) // 旧会话马上要被服务端收掉:本地当场解绑（localStorage 一起清）
    set({
      view: 'debug', wsLogErr: '', launchError: '', launching: true, state: 'loading',
      timeline: [], rawLog: [], watches: [], autovars: [], backendDead: '',
      selectedFrame: -1, stop: null, frames: [], breakpoints: [],
      sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false, currentLine: 0, lineOffset: 0,
      loadingSource: true, lastReplayRowid: rowid, lastReplayRequest: request ?? '',
    })
    try {
      if (old) void api.quit(old).catch(() => {})
      const r = await api.wsLogDebug(rowid, request, old || undefined, ent, env)
      const mod = r.module || ''
      const rp = r.runProg || r.prog || ''
      // 响应里的 id 是**本页面自己发起**的那条新会话 —— 权威绑定（人点的动作，不是问服务端）
      setBoundSession(r.sessionId)
      set({
        module: mod, prog: r.prog || rp,
        runProg: rp, state: 'loading', sourceDVM: '', sourceMissing: false, currentLine: 0, loadingSource: true,
      })
      bindEpoch++ // 权威绑定:在途的 list/snapshot 结果从此作废(见 bindEpoch 的说明)
      // 入口停站前保持加载态,源码由会话路径加载并定位 MAIN
      pollUntilStopped(set, get)
    } catch (e: any) {
      cancelRetire()
      // 启动失败原先是一条**占据一整行的红色横幅**(App.tsx 里那个横条 + 一个关闭按钮):
      // 它一出现就把整个工作台往下压,而它说的只是一句失败原因。改走统一消息出口,
      // 并**不自动关**(duration:0)—— 启动失败必须看见,不能 3 秒后自己溜走。
      toast.error('启动失败', { description: e?.message || String(e), duration: 0 })
      setBoundSession(null)
      set({ wsLogErr: e.message || String(e), state: '', launchError: `启动失败: ${e.message}`, loadingSource: false, stop: null })
    } finally {
      endRetire()
      set({ launching: false })
    }
  },

  launch: async (module, prog) => {
    launchStartedAt = Date.now()
    launchWarned = false
    set({ launching: true, launchError: '', timeline: [], rawLog: [], watches: [], autovars: [], backendDead: '', selectedFrame: -1, lastReplayRowid: null, lastReplayRequest: '' })
    // 启动调试:编辑器进入加载态(转圈),入口停站定位 MAIN 后一次性显示源码,
    // 避免启动过程中内容跳来跳去
    set({ sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false, currentLine: 0, loadingSource: true })
    // 同 replayStart:上一会话(同目标会被复用,ID 与新一轮相同)的收尾事件不得清掉加载态
    beginRetire(get().sessionId)
    try {
      // 带上"当前这条会话"：面板上的「运行」是**在这条空闲会话里起作业**，
      // 不是再建一条新会话。不带的话服务端只能按"服务当前环境"去组装，而统一服务
      // 启动时可能还没有生效环境 —— 报的是"必须显式指定企业编号(ENT)"，人看到的就是
      // "点了没反应"。
      //
      // **ENT 也要显式带上**：服务端的 ENT 纪律是"本次显式给 或 会话自己的 cfg.Topent"，
      // 而会话那个值在"收口重建"之后可能已经丢了（实测：鼎捷1/鼎捷2 都是 idle 且
      // cfg.Topent 为空，于是任何启动都 409，状态停在 idle，步入/步过整排变灰）。
      // 会话列表里的 ent 是服务端回读的**当前生效值**，拿它兜住这条链。
      const sid = get().sessionId
      const cur = sid ? get().sessions.find((x) => x.id === sid) : undefined
      const ent = (cur?.ent || '').trim()
      const r = await api.launch(module, prog, sid ? { session: sid, ...(ent ? { ent } : {}) } : undefined)
      // 作业编号解析:后端连 gzzz_t 后回读模块与实体程序(aint301_wf → aint302_wf)
      const mod = r.module || module
      const rp = r.runProg || prog
      // 同上：这是**本页面自己发起**的会话，响应里的 id 就是权威绑定（人点的动作）
      setBoundSession(r.sessionId)
      set({ module: mod, prog, runProg: rp, state: 'loading' })
      bindEpoch++ // 权威绑定:在途的 list/snapshot 结果从此作废(见 bindEpoch 的说明)
      get().pushTimeline({ origin: 'human', kind: 'command', text: `启动调试会话 ${mod}/${prog}` })
      alignAutoPrefs(set, get) // 自动变量开关下发到新会话(求值在服务端做)
      // 轮询直到入口停站(源码由会话路径加载并定位 MAIN)
      pollUntilStopped(set, get)
    } catch (e: any) {
      // 同步失败(如 prog 缺失/配置非法):HTTP 直接报错,同样必须退出加载态
      cancelRetire()
      get().pushTimeline({ origin: 'system', kind: 'warn', text: `启动失败: ${e.message}` })
      // **不要在这里解绑**：会话本身还在（只是这一轮没起来）。解绑会换走整份现场，
      // 人看到的是"会话没了/界面空了"，比错误原文更难理解（实测反馈：点了重新开始，
      // 之后步入/步过全灰，以为多会话坏了）。保留绑定 + 落回 idle，可以直接重试。
      set({ launchError: `启动失败: ${e.message}`, state: get().sessionId ? 'idle' : '', loadingSource: false, stop: null, currentLine: 0 })
      throw e
    } finally {
      endRetire()
      set({ launching: false })
    }
  },

  refreshSnapshot: async () => {
    const { sessionId } = get()
    if (!sessionId) return
    try {
      const snap = await api.snapshot(sessionId)
      // 注意:这里不更新 currentLine——下断点等操作也会触发快照刷新,
      // 焦点行只能由真正的停站事件/步进响应驱动,否则浏览位置会被拽回运行行
      set({
        state: snap.state, stop: snap.stop, breakpoints: snap.breakpoints || [],
        started: !!snap.started,
        holdingSeconds: snap.holdingSeconds || 0,
        watchdogSeconds: snap.watchdogSeconds || 0, watchdogDisabled: !!snap.watchdogDisabled,
        sessionEnv: snap.env || get().sessionEnv,
        // 谁在驾驶 + 正在执行哪条命令(界面按它收敛写操作、显示 inflight)
        mode: modeOf(snap.mode),
        inflight: snap.inflight || null,
        // 免模块启动时后端会按作业名解析模块,回读给前端(源码兜底路径依赖它)
        module: snap.module || get().module,
        runProg: snap.runProg || get().runProg,
      })
      if (snap.state === 'stopped' && !snapTimer) startHoldTimer(set, get)
      if (snap.state !== 'stopped' && snap.state !== 'loading' && snapTimer) {
        clearInterval(snapTimer); snapTimer = undefined
      }
    } catch { /* 会话可能已结束 */ }
  },

  refreshSource: async (file, line) => {
    const { sessionId, module, prog, runProg, sourceDVM } = get()
    if (!sessionId) return
    let f = file || get().stop?.file
    let entryMode = false
    if (!f) {
      // 入口停站:DVM 未上报文件名,按 T100 命名规则加载母版源码,便于直接点行号下断点
      // 注意:作业编号可能解析出不同名的实体程序(gzzz_t),母版跟实体程序走
      const rp = runProg || prog
      if (!rp) return
      f = `${module ? module + '_' : ''}${rp}.4gl`
      entryMode = true
    } else if (get().stop?.reason === 'entry') {
      // 入口停站:后端已给出真实源文件(转客制作业是 cpm_xxx.4gl),仍需定位 MAIN
      entryMode = true
    }
    if (f === sourceDVM && !get().sourceMissing) {
      // 同文件且**确认加载过**:行号直接落位
      if (line) set({ currentLine: line })
      return
    }
    // 跨文件停站:先收光标(currentLine=0)防它在旧文件上错位,源码到位后一次性落位
    // 行号偏移逐文件不同,切文件后需重新校准
    set({ loadingSource: true, currentLine: 0, lineOffset: 0 })
    try {
      const { source } = await api.sourceByFile(sessionId, f, module)
      set({ sourceContent: source.content, sourcePath: source.path, sourceDVM: f, sourceMissing: false })
      const st = get()
      if (line) {
        set({ currentLine: line })
        // 跨文件停站(F11 步入/断点/运行到其它文件):源码到位后发定位信号,
        // 由 reveal effect 滚动视口到停站行;入口停站走下方 MAIN 定位
        if (!entryMode) get().reveal('debug', line)
      } else if (entryMode && st.currentLine === 0) {
        jumpToMain(set, get)
        st.pushTimeline({ origin: 'system', kind: 'info', text: '入口停站:已显示源码,点击行号下断点后点「继续 F5」开始' })
      }
      // 入口停站:跳转到 MAIN 首条语句,让用户聚焦起点(不打断后续停站定位)
      if (entryMode && get().currentLine > 0) get().reveal('debug', get().currentLine)
    } catch {
      // 无源码(如 com 公共库只有 42m):明确置空,避免在旧文件上标错停站行
      if (entryMode) {
        // 客制母版回退:cpm_apmt580_wf.4gl(首字母 a→c 的客制目录命名)
        try {
          const { source } = await api.sourceByFile(sessionId, `c${module}_${runProg || prog}.4gl`, module)
          set({ sourceContent: source.content, sourcePath: source.path, sourceDVM: f, sourceMissing: false })
          if (get().currentLine === 0) {
            jumpToMain(set, get)
            get().pushTimeline({ origin: 'system', kind: 'info', text: '入口停站:已显示客制源码,点击行号下断点后点「继续 F5」开始' })
          }
          if (get().currentLine > 0) get().reveal('debug', get().currentLine)
          return
        } catch { /* 客制也没有 */ }
      }
      // 取不到源码:DVM 仍要记为 f(否则停站行会画在旧文件上),但要标 missing ——
      // 下一次同文件停站必须重新尝试,不能因为"已经记过 DVM"就永久空白
      set({ sourceContent: '', sourcePath: '', sourceDVM: f, sourceMissing: true })
      if (line) set({ currentLine: line })
    } finally {
      set({ loadingSource: false })
    }
  },

  refreshFrames: async () => {
    const { sessionId, state } = get()
    if (!sessionId || state !== 'stopped') return []
    try {
      const { frames } = await api.where(sessionId)
      set({ frames })
      return frames
    } catch { set({ frames: [] }); return [] }
  },

  refreshWatches: async () => {
    const { sessionId, watches, state } = get()
    if (!sessionId || state !== 'stopped') return
    const out: Watch[] = []
    for (const w of watches) {
      try {
        const { value } = await api.print(sessionId, w.expr)
        out.push({ expr: w.expr, value })
      } catch (e: any) {
        out.push({ expr: w.expr, error: e.message || String(e) })
      }
    }
    set({ watches: out })
  },

  addWatch: async (expr) => {
    const st = get()
    if (!expr.trim() || st.watches.some((w) => w.expr === expr)) return
    set({ watches: [...st.watches, { expr }] })
    await st.refreshWatches()
    st.pushTimeline({ origin: 'human', kind: 'command', text: `监视 ${expr}` })
  },

  removeWatch: (expr) => set((st) => ({ watches: st.watches.filter((w) => w.expr !== expr) })),

  control: async (action, arg) => {
    const { sessionId } = get()
    if (!sessionId) return
    get().pushTimeline({ origin: 'human', kind: 'command', text: arg ? `${action} ${arg}` : action })
    try {
      const resp = await api.control(sessionId, action, arg)
      if (action !== 'interrupt') {
        await get().refreshSnapshot()
        const st = get()
        // 停站后按面板开关同步调用栈取值
        if (st.state === 'stopped' && get().stackAuto) void st.refreshFrames()
      }
      if (action === 'run' || action === 'continue') pollUntilStopped(set, get)
      // 光标落位**不在这里做** —— 服务端的 stopped 事件由 onEvent 统一处理,
      // 人与 AI 由此走同一条路径。以前只有这里能落位,所以 AI 的步进/继续
      // 在界面上完全不可见(服务端的同步路径以前根本不发 stopped 事件)。
      // 保留响应路径反而会引入乱序:AI 紧接着再走一步时,这里的旧 stop
      // 会把光标拽回去。代价是落位依赖 WS 存活,断线由 backendDead 兜住。
      void resp
    } catch (e: any) {
      get().pushTimeline({ origin: 'system', kind: 'warn', text: `${action} 失败: ${e.message}` })
    }
  },

  // 切模式(纯人工/协作)。人与 AI 都能切任意方向 —— 界面上的「接管」同时是
  // "协作模式下不被锁死"的逃生舱口:人始终能把控制权收回来。
  setMode: async (mode) => {
    const { sessionId } = get()
    if (!sessionId) return
    try {
      await api.setMode(sessionId, mode)
      await get().refreshSnapshot() // mode 从快照回读,与 AI 发起的切换走同一条路
      get().pushTimeline({
        origin: 'human', kind: 'command',
        text: mode === 'collab' ? '交给 AI:协作模式' : '接管:纯人工模式',
      })
    } catch (e: any) {
      get().pushTimeline({ origin: 'system', kind: 'warn', text: `切换模式失败: ${e.message}` })
    }
  },

  toggleBreakpoint: async (line) => {
    const st = get()
    if (!st.sessionId) return
    // 当前视图仅显示一个文件的源码:按行号匹配;另查"点击行→注册断点"映射
    // (fgldb 会把空行/注释行上的断点自动调整到下一条可执行语句,点原行也要能取消)
    // 浏览页:断点归属页签文件(location 用 文件:行);调试页沿用现有归属逻辑
    const tabFile = st.activeTab !== 'debug' ? st.tabs.find((t) => t.key === st.activeTab)?.file : ''
    const inTab = (f?: string) => !!f && !!tabFile && f.replace(/^.*[\/]/, '').toLowerCase() === tabFile.toLowerCase()
    const existing = tabFile
      ? st.breakpoints.find((b) => inTab(b.file) && b.line === line)
      : st.breakpoints.find((b) => b.line === line) ||
        (st.adjustedBps[line] !== undefined
          ? st.breakpoints.find((b) => b.num === st.adjustedBps[line])
          : undefined)
    try {
      let newBp: Breakpoint | undefined
      if (existing) {
        await api.bpDel(st.sessionId, existing.num)
        st.pushTimeline({ origin: 'human', kind: 'command', text: `删除断点 ${existing.file}:${existing.line}` })
      } else {
        const loc = tabFile ? `${tabFile}:${line}` : st.stop?.file ? `${st.stop.file}:${line}` : String(line)
        const { breakpoint } = await api.bpAdd(st.sessionId, loc)
        newBp = breakpoint
        st.pushTimeline({ origin: 'human', kind: 'command', text: `断点 ${breakpoint.file}:${breakpoint.line}` })
      }
      await st.refreshSnapshot()
      // 维护"点击行→注册断点"映射:实际注册行号与点击行不同时记录;失效映射清理
      const next: Record<number, number> = {}
      for (const [clickLine, num] of Object.entries(st.adjustedBps)) {
        if (get().breakpoints.some((b) => b.num === num)) next[Number(clickLine)] = num
      }
      if (newBp && newBp.line !== line) next[line] = newBp.num
      set({ adjustedBps: next })
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `断点操作失败: ${e.message}` })
    }
  },

  removeBreakpoint: async (num) => {
    const st = get()
    if (!st.sessionId) return
    try {
      await api.bpDel(st.sessionId, num)
      await st.refreshSnapshot()
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `删除断点失败: ${e.message}` })
    }
  },

  // 选择栈帧:切换 print/locals 求值上下文并跳转该帧源码位置(仅停站时可用)
  selectFrame: async (idx) => {
    const st = get()
    if (!st.sessionId || st.state !== 'stopped') return
    const frame = st.frames.find((f) => f.idx === idx)
    try {
      await api.frame(st.sessionId, idx)
      set({ selectedFrame: idx })
      if (frame) {
        st.pushTimeline({ origin: 'human', kind: 'command', text: `选帧 #${idx} ${frame.func} ${frame.file}:${frame.line}` })
        if (frame.file && frame.file !== st.sourceDVM) await st.refreshSource(frame.file, frame.line)
        else if (frame.line) get().reveal('debug', frame.line)
      }
      void st.refreshWatches()
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `选帧失败: ${e.message}` })
    }
  },

  toggleBPEnabled: async (num, enabled) => {
    const st = get()
    if (!st.sessionId) return
    try {
      await api.bpEnabled(st.sessionId, num, enabled)
      st.pushTimeline({ origin: 'human', kind: 'command', text: `${enabled ? '启用' : '禁用'}断点 #${num}` })
      await st.refreshSnapshot()
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `断点启停失败: ${e.message}` })
    }
  },

  // 断点列表点击跳转:必要时切换到断点所在文件,再定位到断点行。
  // 纯浏览跳转(nav):只滚视口,不把黄色停站高亮拽到断点行(不改变运行上下文)
  jumpToBp: async (b) => {
    const st = get()
    if (b.file && b.file !== st.sourceDVM) await st.refreshSource(b.file)
    get().reveal('debug', b.line, true)
  },

  // 运行到光标:当前文件即停站文件时用行号,否则带文件名(fgldb until [file:]line)
  runToCursor: async (line) => {
    const st = get()
    if (!st.sessionId || st.state !== 'stopped') return
    const cur = st.sourceDVM
    const top = st.stop?.file || st.frames[0]?.file || ''
    const arg = cur && top && cur !== top ? `${cur}:${line}` : String(line)
    await st.control('until', arg)
  },

  quit: async () => {
    const st = get()
    if (!st.sessionId) return
    let kept = true
    try {
      const r = await api.quit(st.sessionId)
      kept = r?.kept !== false
      st.pushTimeline({ origin: 'human', kind: 'command', text: '结束调试(会话保留,可直接再次启动)' })
    } catch { /* ignore */ }
    if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
    stopHoldTimer()
    // 结束本轮调试:运行现场清空,会话与源码保留(idle)。仅在后端整体断开时才清绑定
    const base = {
      started: false, stop: null, breakpoints: [], frames: [], adjustedBps: {},
      currentLine: 0, autovars: [], selectedFrame: -1,
    }
    if (kept) {
      set({ state: 'idle', ...base })
    } else {
      // 后端把会话移除了:本地绑定也解掉（走 setBoundSession，localStorage 一起清），
      // 并刷一次列表 —— 会话面板要能立刻看出"这条没了"。
      setBoundSession(null)
      set({
        module: '', prog: '', runProg: '', sessionEnv: '', state: '', ...base,
        backendDead: '', tabs: [], activeTab: 'debug', sourceContent: '', sourcePath: '', sourceDVM: '', sourceMissing: false, lineOffset: 0,
      })
      void refreshSessionList()
    }
  },

  restart: async () => {
    const st = get()
    if (!st.sessionId) return
    const { module, prog, lastReplayRowid, lastReplayRequest } = st
    st.pushTimeline({ origin: 'human', kind: 'command', text: `重新开始 ${module}/${prog}${lastReplayRowid ? (lastReplayRequest ? '(按接口日志重放,用改过的入参)' : '(按原接口日志重放)') : ''}` })
    try { await api.quit(st.sessionId) } catch { /* 忽略,直接重启 */ }
    if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
    stopHoldTimer()
    // 结束本轮(会话保留 idle),再启动新的一轮——宿主复用,免重新登录。
    // 重放会话必须按原日志重放(报文参数在服务端 ArgsOverride 里,普通启动会丢参数)
    set({ state: 'idle', started: false, stop: null, breakpoints: [], frames: [], adjustedBps: {}, autovars: [], selectedFrame: -1, currentLine: 0 })
    // env 取「这一页所属的环境」：重新开始时本地可能已经没有任何会话，服务端也就没有
    // 可沿用的生效环境（服务重启过就更没有）—— 环境一律点名，这一次也一样。
    if (lastReplayRowid) await get().replayStart(lastReplayRowid, lastReplayRequest || undefined, undefined, st.activeEnv)
    else await get().launch(module, prog)
  },

  doPrint: async (expr) => {
    const st = get()
    if (!st.sessionId || !expr.trim()) return
    try {
      const { value } = await api.print(st.sessionId, expr)
      st.pushTimeline({ origin: 'human', kind: 'info', text: `print ${expr} → ${value}` })
      await st.addWatch(expr)
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `print ${expr} → ${e.message}` })
    }
  },

  // adoptExisting 是**首屏对齐**：列全部会话 + 恢复"人上次点的那条"（若还在），
  // **绝不替人挑一条**（WP-F / Q4）。
  //
  // 与从前的差别（这一处是本次的核心改动之一）：旧实现是"`sessions[0]` 认领并接管"，
  // 也就是**把服务端列表的第一项当成"当前会话"** —— 多会话下那是随机的，
  // 而那正是服务端五处 `Current()` 兜底被删掉的理由，前端不该把它捡回来。
  // 现在：本地记着的那条还在 → 对齐一次（状态/模式/快照/源码）；不在 → 回到未选择。
  adoptExisting: async () => {
    const list = await refreshSessionList()
    if (!list) return
    // 树跟着池一起重建（refreshSessionList 里做的），所以这里不需要再"摘条目"：
    // 服务端没有的会话在树上本来就不会出现（D9：树是池的投影）。
    const id = get().sessionId
    if (!id) return // 未选择:界面提示"选一条"(见会话面板),这里什么都不做
    if (!list.some((s) => s.id === id)) {
      setBoundSession(null)
      return
    }
    await get().selectSession(id)
  },

  // selectSession 是**唯一**能"绑定一条会话"的入口（Q4：选中态只能由人点出来）。
  //
  // 它做三件事，而且是同一件事的三面：写进 localStorage（刷新后恢复）、写进
  // `useStore.sessionId`（本次会话用）、按那条会话对齐状态（模式/快照/源码）。
  // 传 null = 明确回到"未选择"（界面上仍然列着全部会话，等人再点）。
  //
  // 对齐走 `alignSelected` 那一处：它不要求"这条必须在 `sessionList` 里" ——
  // 列表可能是空的（还没列过）或还没刷到这条（刚起的会话）。绑定的语义是"人点了这条"，
  // 不是"列表里有这条"，所以取数失败也只退到"只绑 id"，绝不清掉人的选择。
  selectSession: async (id) => {
    // 先搬编辑器的现场（W-P2）：把上一条的存进 `editor[旧]`，把这一条的换进单例。
    // 必须在 `setBoundSession` 之前 —— 它靠"当前的 sessionId"判断存谁。
    // 放到最后做就晚了：alignSelected 会用服务端事实覆盖 state/stop/breakpoints，
    // 而源码/页签/时间线这些客户端现场只有这一刻能搬。
    switchEditor(set, get, id)
    if (!id) {
      setBoundSession(null)
      set({
        module: '', prog: '', runProg: '', sessionEnv: '', state: '',
        started: false, stop: null, currentLine: 0, breakpoints: [], frames: [],
        adjustedBps: {}, autovars: [], selectedFrame: -1, holdingSeconds: 0, loadingSource: false,
      })
      return
    }
    setBoundSession(id)
    await alignSelected(set, get, id)
  },

  // 会话管理:右侧「会话」sheet 的 结束/重启/连接(后端会先结束当前 debug)
  sessionOp: async (op, env, ent) => {
    const st = get()
    // 第三档是"连到该环境"(Q26:不是"把一个会话切到另一个环境";连上去会收掉当前那条,Q27)
    const label = op === 'close' ? '结束会话' : op === 'restart' ? '重启会话' : '连到该环境'
    try {
      // **绝不假装成功**。原来这三条是 `if (… && st.sessionId) await …` 串起来的：
      // 条件不成立时**一个请求都不发**，却照样往时间线里记一条"结束会话 → 主机109"，
      // 再 sync 一次把还活着的会话又绑回来 —— 用户看到的就是"点了结束会话，什么都没发生，
      // 也没有任何提示"（实测踩到）。所以：做不到就**说清楚为什么**，让上层的 toast 去讲。
      if (op === 'close') {
        if (!st.sessionId) throw new Error('界面上没有当前会话（状态可能已经过期）。点右上角刷新看真实状态；若左侧列表里还有会话，直接点它的「结束会话」')
        await api.sessionClose(st.sessionId)
        // 关完**核对一次**：POST 成功不等于真的关掉了（服务端异常时可能 200 而会话还在）。
        const { sessions } = await api.list()
        if ((sessions || []).some((s) => s.id === st.sessionId)) {
          throw new Error('服务端还报着这个会话 —— 关闭没有真正生效，去看服务端日志（.tt-serve.log）里这一轮会话的收尾')
        }
      } else if (op === 'restart') {
        if (!st.sessionId) throw new Error('界面上没有当前会话，无法重启。点右上角刷新后重试')
        await api.sessionRestart(st.sessionId, ent)
      } else {
        if (!env) throw new Error('没有点名要切到哪个环境')
        // **带上要收掉的那条会话**（WP4b / Q4 / C8.2）：服务端不再自己找"当前会话"。
        // 这条会话会被服务端收掉（换环境 = 换连接目标），所以本地也当场解绑：
        // 之后新会话由响应里的 sessionId 权威绑定（服务端回的是它新建的那条）。
        await api.sessionSwitch(env, ent, st.sessionId || undefined)
        setBoundSession(null)
      }
      st.pushTimeline({ origin: 'human', kind: 'command', text: label + (env ? ' → ' + env : '') + (ent ? ' (ENT ' + ent + ')' : '') })
    } catch (e: any) {
      st.pushTimeline({ origin: 'system', kind: 'warn', text: `${label}失败: ${e.message || String(e)}` })
      throw e
    }
    await get().syncFromSessions()
  },

  // syncFromSessions：**只对齐"我选的那条"**，不问服务端"当前是哪条"（Q4 的硬规矩）。
  //
  // 从前它做的是"取列表第一项并绑上去"—— 换环境、结束会话、AI 起了新会话之后都靠它
  // 让界面跟上，而"跟上"的方式是**替人挑一条**。现在：
  //
  //   ① 列全部（填 `sessionList`，会话面板显示的就是它）；
  //   ② 本地没选 → 什么都不做（未选择就是要显示成未选择）；
  //   ③ 本地选的那条还在 → 按它对齐状态/快照/源码；
  //   ④ 不在了（被换掉/关掉/断开）→ **回到未选择**，绝不挑下一条。
  syncFromSessions: async () => {
    // 代次守卫:await 期间若有人权威地绑了新会话(launch/replayStart/selectSession),
    // 这一轮结果作废 —— 否则慢一拍的 list() 会把刚绑上的会话抹成 null（见 bindEpoch 的说明）。
    const epoch = bindEpoch
    if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
    stopHoldTimer()
    const clearRun = {
      started: false, stop: null, currentLine: 0, breakpoints: [], adjustedBps: {},
      frames: [], autovars: [], selectedFrame: -1, holdingSeconds: 0,
    }
    const list = await refreshSessionList()
    if (epoch !== bindEpoch) return
    if (!list) return // list 失败:保持现状
    const id = get().sessionId
    if (!id) return // 未选择:什么都不做（面板列着全部，等人点）
    if (!list.some((x) => x.id === id)) {
      // 选的那条在服务端没了:回到未选择（不是替人挑一条）
      setBoundSession(null)
      set({ module: '', prog: '', runProg: '', sessionEnv: '', state: '', ...clearRun, loadingSource: false })
      return
    }
    await alignSelected(set, get, id)
  },

  // 停站态到了,但本地没有停站位置:去快照问一次,并把它落到编辑器上。
  //
  // 为什么需要:停站有两条上报路 —— `stopped` 事件(带位置)与 `state=stopped`(只报状态)。
  // **入口停站只有后者**(程序还没 run,没有源码行可报),位置在快照里。于是"AI 起好会话、
  // 页面随后绑上"这条路上,页面拿到的是"已停站 + 空白编辑器":命令框又说"需停站后才能发送命令"。
  // 判据是 stop 是否已知 —— 正常断点停站走 stopped 事件,这里会在第一行早退,不多打一次往返。
  syncStopFromSnapshot: async () => {
    const st = get()
    if (!st.sessionId || st.state !== 'stopped' || st.stop) return
    await get().refreshSnapshot()
    const cur = get()
    if (cur.stop && (cur.stop.file !== cur.sourceDVM || cur.sourceMissing)) {
      await cur.refreshSource(cur.stop.file, cur.stop.line)
    }
  },

  // ---- 工作站外壳:环境树 + 会话树 ----
  //
  // 环境清单来自共享端点 `/api/hosts`(与设置页同一份数据源,含口令,所以只在需要时拉)。
  // 失败与"一个都没配"是两件事:`envsErr` 非空时界面要说"读不到",不能显示成"还没有环境"
  // (旧会话面板在这条上踩过一次,见 SessionPanel.tsx 的注释)。
  loadEnvs: async () => {
    try {
      const c = await api.settings()
      const sshs = Array.isArray(c?.sshs) ? c.sshs : []
      const envs: EnvBrief[] = sshs.flatMap((e): EnvBrief[] => {
        const name = (e?.name || '').trim()
        if (!name) return []
        // /api/hosts 是 T100 + GP 的合并清单。T100 的树不能把 GP 环境也画进来，
        // 否则同一个 TOPTEST 会同时出现在 T100 树和 GP 分区（实测截图就是这样）。
        if (String((e as any)?.product || '').toLowerCase() === 'gp') return []
        return [{ name, host: e.host, zone: e.zone }]
      })
      set({ envs, envsLoaded: true, envsErr: '' })
    } catch (e: any) {
      set({ envsLoaded: true, envsErr: e?.message || String(e) })
    }
  },
  // TOP GP 的环境清单：固定同源，和 T100 的 /api/hosts 一样。
  // 不再读旧的 localStorage `tt.gp.base`：那是 Python ttgp(28671) 时代的地址，
  // 留着会让 28670/28690 的前端去连一个已经停掉的服务，gpEnvs 变空。
  loadGpEnvs: async () => {
    try {
      const r = await fetch('/api/gp/meta')
      const m = await r.json()
      set({ gpEnvs: Array.isArray(m?.envs) ? m.envs : [], gpEnvsLoaded: true, gpEnvsErr: '' })
    } catch (e: any) {
      set({ gpEnvsLoaded: true, gpEnvsErr: e?.message || String(e) })
    }
  },
  setSettingsGpEnv: (n) => set({ settingsGpEnv: n }),
  // 切环境(路由驱动):把"当前这一页"的接口日志整组存回自己的槽,再把目标环境的槽
  // 换出来 —— 一个站点一份,切走再切回原样还在。
  setActiveEnv: (env) => {
    const cur = get()
    if (cur.activeEnv === env) { set({ activeEnv: env }); return }
    const byEnv = { ...cur.wsLogsByEnv }
    if (cur.activeEnv) byEnv[cur.activeEnv] = wsLogSnapOf(cur)
    const next = byEnv[env] || emptyWsLogSnap()
    set({ activeEnv: env, wsLogsByEnv: byEnv, ...wsLogFlatOf(next), wsLogsLoading: false })
  },

  // 建一条**真的** Debug 会话（企业编号 + 作业编号），作业编号填了就把作业直接起起来。
  //
  // 两步顺序不能换：
  //   ① `POST /sessions/new` 建**空闲**会话（登录是服务端的另一条异步路）；
  //   ② 等它登录完（loading → idle）—— 起作业的前置就是“空闲”。
  //
  // **作业编号填了就直接起作业**（用户 2026-10-19：“我既然填写了作业编号，就应该直接起作业，
  // 除非我没填”）。从前这里建完只留一条空闲会话，人还得在会话页再点一次 ▶ —— 而那个 ▶ 在
  // 状态停在 loading 时根本不显示，于是“填了作业编号却什么也没起”。
  // 只有没填（空着，或者只写了模块）才是“只建会话”那一档。
  //
  // 失败的分寸：①建不出来 / 登录不上 → 抛错（界面不跳转，错误原样显示）；
  // ②会话建好了但作业没起来 → **只报警不抛错** —— 会话能用这个事实不该被一句“启动失败”
  // 盖掉，跳到会话页看 `launchError`，改作业编号按 ▶ 重试就好。
  createSession: async (input, onStep) => {
    const env = (input.env || '').trim()
    const ent = (input.ent || '').trim()
    const prog = (input.prog || '').trim()
    // 建会话只有一个入口（不再有"用途"这一层）：
    // 名字留空就在**发请求之前**落成「<环境> <序号>」，而不是只在界面上显示一个
    // 服务端根本不知道的名字。
    const name = (input.name || '').trim() || defaultSessionName(get().sessions, env)
    // TOP GP 走「模块 + 作业名」：给模块就不要求 ENT（GP 没有 ENT 概念）
    const explicitMod = (input.module || '').trim()
    // 前置校验放在请求之前:与服务端同一套判据，但错误当场说、不白花一次 SSH 握手。
    //
    // **作业编号可以不填**：填了就在建好的会话里直接起作业（见下面那段 `launch`），
    // 空着就只建空闲宿主会话。这一格从前是必填 —— 于是“先挂一条会话、作业待会儿再起”这条路根本不存在。
    if (!env) throw new Error('没有点名环境')
    const isGp = get().gpEnvs.some((e) => e.name === env)
    // GP 的模块可以留空，由后端按 p_link 解析；T100 仍必须有 ENT 或显式模块。
    if (!isGp && !ent && !explicitMod) throw new Error('必须填企业编号（ENT）：它决定用哪个账号连库，环境配置里已经没有它了')
    // 作业编号也接受「模块/作业」（与右侧「运行」那个输入框同一种写法）。
    // 只写了模块（结尾是 `/`）按“没填作业”算，不拿一个空程序名去起作业。
    const slash = prog.indexOf('/')
    const mod = explicitMod || (slash > 0 ? prog.slice(0, slash).trim() : '')
    const progName = (slash > 0 ? prog.slice(slash + 1) : prog).trim()

    onStep?.('正在建会话（连 ' + env + '）…')
    const res = await api.sessionCreate(env, ent, name)
    const id = (res?.sessionId || '').trim()
    if (!id) throw new Error('服务端没回会话 ID（看 .tt-serve.log）')

    // 先刷一次池：树是池的投影（W2/D9），这一步让新会话与它的 `purpose` 立刻出现在树上，
    // 而不是等我们自己往本地数组里塞一条（塞出来的那条没有服务端的 state/purpose）。
    await refreshSessionList()
    const item: SessionItem = get().sessions.find((s) => s.id === id) || {
      // 极端情况：列表还没刷到它（服务端刚建、聚合有延迟）。用应答里的事实拼一条，
      // 只用于**导航**；下一轮 List / 事件会把树换成权威那份。
      id, env, type: 'debug', createdAt: Date.now(), ent, prog, module: mod,
      name: (res?.name || '').trim() || name,
      ref: (res?.ref || '').trim(),
    }

    onStep?.('会话已建好，正在登录 ' + env + '…')
    try {
      await waitSessionIdle(id)
    } catch (e: any) {
      get().pushTimeline({ origin: 'system', kind: 'warn', text: String(e?.message || e) })
      throw e
    }

    // 先绑上这条新建的会话：`launch` 是“在**当前**会话里起作业”，它读 `get().sessionId`；
    // 不先绑就会起在别处（或又新建一条）。
    await get().selectSession(id)
    if (progName) {
      onStep?.(`会话已就绪，正在启动作业 ${progName}（等它停在入口）…`)
      try {
        await get().launch(mod, progName)
      } catch (e: any) {
        // 会话已经建好并绑上了：作业没起来 ≠ 会话没建成 —— 不 throw。
        // 落在会话页上看 `launchError`，改个作业编号按 ▶ 就能重试。
        get().pushTimeline({ origin: 'system', kind: 'warn', text: `作业 ${progName} 没起来：${e?.message || e}` })
      }
    }

    get().pushTimeline({
      origin: 'system', kind: 'info',
      text: progName
        ? `新建会话 ${item.ref || id}（作业 ${mod ? mod + '/' : ''}${progName}，ENT ${ent}）`
        : `新建会话 ${item.ref || id}（空闲，未指定作业；ENT ${ent}）`,
    })
    return item
  },


  // （`pruneSessions` 已随"树由池派生"一起下线：树在 refreshSessionList 里随池重建，
  //  不再需要单独摘条目。）


  // ensureSource 在没有源码可看时，把这条会话的**母版源码**打开。
  //
  // 会话页有三种"有会话、但还没停站"的时刻：刚建好（idle）、刚起跑（running）、
  // 以及从树上点回来。这几种情况下人第一眼想看的就是这个作业的代码，而旧实现只在
  // `stopped` 时才拉源码 —— 于是编辑器一片空白（用户 2026-10-18 报的就是这个）。
  //
  // 只在**模块与程序都已知**时拉：模块没解析出来时拼不出正确文件名
  // （`<模块>_<程序>.4gl`），去服务器上按错名字瞎撞一串，最后只会得到"无源码"这个假结论。
  ensureSource: async () => {
    const st = get()
    if (!st.sessionId || st.loadingSource) return
    // 有停站位置的那条由 refreshSource(file,line) 负责（它要顺带落位，不能抢）
    if (st.state === 'stopped' && st.stop?.file) return
    if (st.sourceDVM) return

    let mod = st.module
    let prog = st.runProg || st.prog
    if (!mod || !prog) {
      // 模块/作业可能是**刚被服务端解析出来**的（起跑那一刻才知道），本地还空着 ——
      // 问一次快照再决定。不这么问的话，"从树上点开→别处启动→状态变成 running"
      // 这条路永远拉不到源码（状态变了但 module 还空，判据一直不成立）。
      try {
        const snap: any = await api.snapshot(st.sessionId)
        if (get().sessionId !== st.sessionId) return // 期间换了会话:这一轮作废
        mod = mod || snap?.module || ''
        prog = prog || snap?.prog || ''
        if (snap?.module || snap?.prog) {
          set({ module: mod, prog: prog, runProg: snap?.runProg || prog })
        }
      } catch {
        return // 会话瞬态/已不在:下一次状态变化还会来
      }
    }
    if (!mod || !prog) return
    if (get().sourceDVM) return
    await get().refreshSource()
  },

  // closeSession 结束一条会话：服务端断开连接并把它从池里摘掉，树里也摘掉。
  //
  // 「结束会话」与「结束本轮调试」是两件事（后者 `quit` 只结束运行、宿主保留）；
  // 树上那个按钮要的是前者。摘之前**核对一次** —— POST 成功不等于真的关掉了，
  // 服务端异常时可能回 200 而会话还在（旧面板的注释里记着这一次）。
  closeSession: async (id) => {
    const label = get().sessions.find((s) => s.id === id)?.name || id
    try {
      await api.sessionClose(id)
      // 核对一次：POST 成功不等于真的关掉了（服务端异常时可能回 200 而会话还在）。
      const { sessions } = await api.list()
      if ((sessions || []).some((s) => s.id === id)) {
        throw new Error(`服务端还报着会话「${label}」—— 关闭没有真正生效（看 .tt-serve.log）`)
      }
      // 树跟着池一起重建（close 的事件也会到，但这里不等它：人点了就该立刻少一条）。
      set({ sessionList: sessions || [], sessions: indexFromList(sessions || []), sessionsLoaded: true })
      dropRuntime(id)
      if (get().sessionId === id) await get().selectSession(null)
      get().pushTimeline({ origin: 'human', kind: 'command', text: `结束会话 ${label}` })
      toast.success(`已结束会话 ${label}`)
    } catch (e: any) {
      // 绝不假装成功：失败照实抛，界面上由 toast 讲
      toast.error('结束会话失败', { description: e?.message || String(e), duration: 0 })
      throw e
    }
  },
}))

function pollUntilStopped(set: (p: Partial<Store>) => void, get: () => Store) {
  if (snapTimer) clearInterval(snapTimer)
  snapTimer = window.setInterval(async () => {
    const st = get()
    if (!st.sessionId) { if (snapTimer) clearInterval(snapTimer); return }
    try {
      const snap = await api.snapshot(st.sessionId)
      // 跨文件停站:光标等 refreshSource 落位,不先画到旧文件上
      const nl = !!snap.stop?.file && snap.stop.file !== st.sourceDVM
      set({
        state: snap.state, stop: snap.stop, breakpoints: snap.breakpoints || [],
        started: !!snap.started,
        holdingSeconds: snap.holdingSeconds || 0,
        watchdogSeconds: snap.watchdogSeconds || 0, watchdogDisabled: !!snap.watchdogDisabled,
        sessionEnv: snap.env || get().sessionEnv,
        module: snap.module || get().module,
        runProg: snap.runProg || get().runProg,
        currentLine: nl ? 0 : (snap.stop?.line || get().currentLine),
        loadingSource: nl ? true : get().loadingSource,
      })
      if (nl && snap.stop?.file) void get().refreshSource(snap.stop.file, snap.stop.line)
      else if (snap.state === 'stopped' && snap.stop?.line && !nl) get().reveal('debug', snap.stop.line)
      if (snap.state === 'stopped') {
        clearInterval(snapTimer!)
        snapTimer = undefined
        set({ activeTab: 'debug' })
        startHoldTimer(set, get)
        if (get().stackAuto) void get().refreshFrames()
        void get().refreshWatches()
        void get().refreshSource(snap.stop?.file)
        // 兜底:停站宣告与后端断点恢复收尾之间仍有微小窗口(事件先于快照到达),
        // 稍后补一次快照,保证缓存的断点无需步进就能显示
        window.setTimeout(() => {
          const cur = get()
          if (cur.sessionId && cur.state === 'stopped') void cur.refreshSnapshot()
        }, 2500)
        if (snap.stop?.reason === 'breakpoint') {
          st.pushTimeline({ origin: 'system', kind: 'stop', text: `命中断点 ${snap.stop.file}:${snap.stop.line}` })
        }
      }
      // 启动看门狗：loading 拖太久要说出来（并且最终退出，不留一个永远转圈的界面）。
      if (snap.state === 'loading' && launchStartedAt > 0) {
        const waited = Date.now() - launchStartedAt
        if (waited > 120000 && !launchWarned) {
          launchWarned = true
          st.pushTimeline({ origin: 'system', kind: 'warn', text: '启动超过 120 秒仍未停站：检查 GDC 是否已起、作业是否在等界面输入（可看服务日志 .tt-serve.log）' })
          set({ launchError: '启动超过 120 秒仍未停站。常见原因：GDC 没有起来 / 作业在等服务端输入 / 断点下在不可执行的行。' })
        }
        if (waited > 240000) {
          if (snapTimer) clearInterval(snapTimer!)
          snapTimer = undefined
          launchStartedAt = 0
          set({ state: '', launching: false, loadingSource: false, launchError: '启动超时（240 秒仍未停站）：已退出等待。检查服务日志后重试。' })
          st.pushTimeline({ origin: 'system', kind: 'warn', text: '启动超时（240 秒）：已退出等待' })
          return
        }
      }
      if (snap.state === 'stopped' || snap.state === 'idle' || snap.state === 'exit') launchStartedAt = 0
      if (snap.state === 'exit') { clearInterval(snapTimer!); snapTimer = undefined }
      if (snap.state === 'idle') {
        // 本轮运行直接结束(程序立即退出等):回空闲,清运行现场(会话保留)
        clearInterval(snapTimer!); snapTimer = undefined
        stopHoldTimer()
        set({ started: false, stop: null, currentLine: 0, loadingSource: false, selectedFrame: -1, autovars: [], breakpoints: [], frames: [], adjustedBps: {} })
      }
    } catch {
      // 快照失败可能因为会话已被移除(启动失败/异常退出且 WS 事件未送达):
      // 确认会话确实消失后退出加载态并报错,避免永久转圈;瞬时网络错误则继续轮询
      const st = get()
      if (!st.sessionId) { if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined } return }
      try {
        const { sessions } = await api.list()
        if (!sessions.some((x) => x.id === st.sessionId)) {
          if (snapTimer) { clearInterval(snapTimer); snapTimer = undefined }
          stopHoldTimer()
          setBoundSession(null)
          set({ state: '', loadingSource: false, stop: null,
            launchError: st.launchError || '调试会话启动失败(会话已消失),请检查作业名或服务器状态' })
        }
      } catch { /* list 也失败:保持轮询,等服务恢复 */ }
    }
  }, 1000)
}

// 页面「自动变量」开关与服务端会话对齐(求值在服务端做;启动/接管/换会话后重新下发),
// 已停站且开关开时拉一次最近求值结果,补上重连期间可能错过的 autovars 事件
function alignAutoPrefs(set: (p: Partial<Store>) => void, get: () => Store) {
  const sid = get().sessionId
  if (!sid || !get().autovarsAuto) return
  void api.autovarsAuto(sid, true).catch(() => { /* 会话瞬态:下次对齐逻辑重发 */ })
  if (get().state === 'stopped') {
    void api.autovars(sid).then(({ vars }) => set({ autovars: vars || [] })).catch(() => {})
  }
}

function startHoldTimer(set: (p: Partial<Store>) => void, get: () => Store) {
  stopHoldTimer()
  holdTimer = window.setInterval(() => {
    const st = get()
    if (st.state !== 'stopped') { stopHoldTimer(); return }
    set({ holdingSeconds: st.holdingSeconds + 1 })
  }, 1000)
}

function stopHoldTimer() {
  if (holdTimer) { clearInterval(holdTimer); holdTimer = undefined }
}

// 跟随系统:系统明暗变化时,只有"跟随系统"模式需要跟着动(dark 变 → html.dark 与 Monaco 一起变)
watchSystemTheme(() => {
  const st = useStore.getState()
  if (st.theme !== 'system') return
  const dark = resolveDark('system')
  applyDark(dark)
  useStore.setState({ dark })
})

// 连接 WebSocket(自动重连)
export function connectWS() {
  const setWs = useStore.getState().setWsConnected
  const onEvent = useStore.getState().onEvent
  const pushRaw = useStore.getState().pushRaw
  let closed = false
  let ws: WebSocket | undefined

  function connect() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    // WebSocket 也走私有前缀(API_BASE = /debug/api):后端把调试子系统挂在 /debug 下,
    // 包内注册的 /api/ws 对外是 /debug/api/ws —— 与共享层 /api/* 的同名路径靠它区分。
    ws = new WebSocket(`${proto}://${location.host}${API_BASE}/ws`)
    ws.onopen = () => {
      setWs(true)
      // 连接/重连后按服务端实况对齐会话绑定:后端重启、会话被移走或页面长时间
      // 挂着时清掉残留引用,避免界面停在"会话进行中"锁死 wslogs 重放等操作
      void useStore.getState().syncFromSessions()
    }
    ws.onclose = () => {
      setWs(false)
      if (!closed) setTimeout(connect, 2000)
    }
    ws.onmessage = (m) => {
      try {
        const ev: any = JSON.parse(m.data)
        // 建连开场:先补发一帧历史(填时间线),再是 hello 哨兵,之后才是流式事件。
        // epoch 变了说明服务端重启过 —— 序号已归零,重置去重游标。
        if (ev.type === 'replay' || ev.type === 'hello') {
          const epoch = String(ev.epoch || '')
          if (epoch && epoch !== seenEpoch) {
            seenEpoch = epoch
            seenSeq = 0
          }
          const replay = useStore.getState().onReplay
          for (const e of (ev.events || []) as Event[]) replay(e)
          return
        }
        onEvent(ev)
      } catch { pushRaw(String(m.data)) }
    }
  }
  connect()
  return () => { closed = true; ws?.close() }
}

// 调试句柄:浏览器控制台可用 __store.getState() 检查状态
if (typeof window !== 'undefined') {
  ;(window as unknown as Record<string, unknown>).__store = useStore
}

// 浏览器控制台调试入口(生产无副作用)
;(window as any).__store = useStore


