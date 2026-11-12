// REST API 客户端
//
// 调试工作台**私有**接口(会话/断点/接口日志/WebSocket)一律经 API_BASE 拼前缀:
// 统一服务把本子系统用 http.StripPrefix 挂在 /debug 下,于是包内注册的 /api/sessions
// 对外是 /debug/api/sessions。这套前缀把它与共享层 /api/* 的同名路径
// (/api/status、/api/ws)区分开 —— 少了它,请求会落到共享层去。
//
// 共享端点(hosts 配置、dbprobe、dbaccverify、conntest)由统一服务在**顶层**各提供
// 一份,所以不带前缀。注意 /debug/api/hosts 也能通,但那是调试子系统留给独立运行
// (tt debug serve)用的只读副本 —— 统一服务下环境页读写的必须是顶层 /api/hosts。
export const API_BASE = '/debug/api'

export interface Frame { idx: number; func: string; file: string; line: number }
export interface SourceLine { num: number; text: string; isCur: boolean }
export interface StopInfo {
  reason: string; bpNum?: number; func?: string; file?: string; line?: number
  frames?: Frame[]; source?: SourceLine[]
}
export interface Breakpoint { num: number; file: string; line: number; func?: string; enabled: boolean; note?: string }
export interface VarItem { expr: string; value?: string }
export interface VarDecl { name: string; type: string }
// 正在执行的那条命令 —— 界面上的「AI 正在执行 continue(已 12s)」
export interface InflightInfo { cmd: string; since: string; elapsed: number }
export interface SessionBrief {
  id: string; module: string; prog: string; runProg?: string; state: string
  env?: string
  // 身份三键（C5.1/C5.4）：`ref` 是规范 id「环境#序号」，`name` 是别名（空串=未命名），
  // `ent` 是这条会话当前生效的企业编号。服务端**恒定给键**，取不到是空串。
  ref?: string; display?: string; name?: string; ent?: string; n?: number; epoch?: string
  // 用途（W1）：debug | record | replay。服务端**恒定给这个键**（取不到是空串 = 按 debug 处理）。
  purpose?: string
  file?: string; line?: number; func?: string; reason?: string
  holdingSeconds: number; breakpoints: number; watchdogSeconds: number
  // 这一轮真的不会自动放行（运行时口径：watchdogSeconds<=0）。停站时若为真，
  // 状态栏必须把它说出来 —— “看门狗已关闭”与“还有 30 分钟”是两件完全不同的事。
  watchdogDisabled?: boolean
  // 谁在驾驶:solo(纯人工)| collab(协作,AI 主导)
  mode?: string
  inflight?: InflightInfo
  // 停站行本身就是交互语句(程序把控制权交给了界面)
  waitingForUser?: boolean; waitingKind?: string
  // 运行态下距最近一次协议输出的秒数
  silentSeconds?: number
}
export interface WSLogItem {
  rowid: string; service: string; pid: string
  start: string; end: string; duration: string
  code: string; job: string
  reqPath: string; rspPath: string; reqSize: string; rspSize: string; errMsg: string
  // 对齐 T100 原生页 awsq990 补上的三项(中文名以 tdict rt wsfa_t 字典为准)
  origin: string  // wsfa013 发起端
  server: string  // wsfa018 服务端
  sso: string     // wsfa015 sso秒数
}
export interface WSLogContent {
  request: string; response: string
  // 请求报文内容不完整(超读取上限被截断 / 拿不到原文):
  // 按原报文重放不受影响,但不能基于它编辑后重放(会送出残缺报文)
  requestPartial?: boolean
  // 响应报文同理:只影响展示,不影响重放
  responsePartial?: boolean
}

// 企业目录(ENT → 数据库账号)。
//
// 一个企业对应的是**库里的一个账号/schema**(gzou_t.gzou003),不是另一台库 ——
// 库是环境级的,换环境才换库。account 为空或 "-" 表示这个企业在 gzou_t 里没配可用账号。
export interface EntItem { ent: number; account: string; placeholder?: boolean }

// GET /api/ents 的应答:除了清单,把"这份答案有多新"也原样带出(缓存可以旧,但旧不能藏着)。
export interface EntsReply {
  ok: boolean
  env: string
  account: string // 查 gzou_t 用的系统账号(恒为 ds,说明用)
  ents: EntItem[]
  source: string // live | snapshot | snapshot-stale
  stale: boolean
  fetchedAt: string
  ageSeconds: number
  ttlSeconds: number
  notes?: string[]
}
// 接口日志查询条件(对齐 T100 原生页 awsq990 的 QBE):
// startFrom = wsfa003(起始时间)下界,endTo = wsfa004(结束时间)上界;
// result/origin/server 为等值过滤(wsfa006/wsfa013/wsfa018)
export interface WSLogQuery {
  service?: string
  job?: string // wsfa012 作业编号(支持 * ? 通配)—— 服务名是反域名,按作业找日志靠这个
  result?: string
  origin?: string
  server?: string
  pid?: string // wsfa002 服务程序序号(界面「服务程序」列;与 service 一起可精确定位某一次调用)
  onlyFail?: boolean
  // env：按哪个环境查（统一服务启动时可能没有"生效环境"，必须点名）。
  env?: string
  // ent：这次查的是哪个企业（接口日志页下拉选的那个）。
  // **wsfa_t 在系统账号 ds 下、不按企业分 schema** —— 它不改查询范围，
  // 只用于：结果里回显"选的是哪个企业"、重放时作为那一轮的 TOPENT。
  ent?: number
  page?: number
  startFrom?: string
  endTo?: string
}


// ---- 共享配置端点 /api/hosts(顶层,无前缀)的读写形状 ----
// 环境清单只有一份数据源(config.json 的 hosts 节),设置页的各张卡片共用它;
// 数据结构与 tt/internal/config 的 schema.go 一一对应。
export interface HostsDb {
  type: string // oracle | kingbase
  host: string
  port: number
  service?: string  // oracle: SERVICE_NAME
  database?: string // kingbase: 库名
  accounts?: { account: string; password: string }[]
  // 是否允许 AI 执行只读 SQL。undefined = 未配置 = 开启;显式 false 才关。
  readonlySql?: boolean
  // SSH 隧道：客户端不可达 DB 时经它转发（表单不编辑，手工写在 config.json；
  // 设置页保存时原样保留）。「测试连接」必须带上它，否则测的是另一条链路。
  viaSsh?: {
    host: string; port?: number; user: string; password?: string
    bindPort?: number; remoteHost?: string; remotePort?: number
  }
}
export interface HostsEnt { ent: number; account: string; password?: string }

export interface HostsSsh {
  name: string
  host: string
  port: number
  user: string
  password: string
  zone?: string
  // ENT 清单：由设置页「ENT」页签或 tt env ents --refresh 从 gzou_t 同步而来，
  // **不提供手工编辑**。entsSyncedAt 是那份清单的抄写时间（RFC3339）。
  ents?: HostsEnt[]
  entsSyncedAt?: string
  // 企业编号(topent)已从环境配置移除:ENT 每次显式给(CLI --ent / 会话面板)。
  // workspace 该环境的 .tzs 工作区（绝对路径）。机器级没有这一项（见 internal/config 的 TzsSettings）
  workspace?: string
  // mirrorDir 该环境自己的源码镜像目录；空 = 使用全局 mirror.dir/<环境名>。
  mirrorDir?: string
  // dictDir 该环境自己的本地字典库目录；空 = 使用全局 dict.dir。
  // 只到目录这一级，文件名始终是 <环境>.db。
  dictDir?: string
  db?: HostsDb | null
}
// 调试节(debug.*):本机参数,不随环境走
export interface HostsDebug {
  launchArgs?: string
  watchdogSeconds?: number
  fglserver?: string
  termWidth?: number
  termHeight?: number
  printElements?: number
  persistBreakpoints?: boolean
}
// GET /api/hosts 的响应:设置页用的那一份配置快照
export interface HostsView {
  config: string
  sshs: HostsSsh[]
  listen: string // 监听地址在顶层(合并前它藏在 debug.listen 里)
  debug: HostsDebug
  query: { source?: string }
  mirror: { dir?: string }
  // dict 本地字典库根目录；空 = 缺省 <数据目录>/dict
  dict: { dir?: string }
  sync: { target?: string }
  tdev: { workspaceSuffix?: string; defaultOut?: string }
  version: string
}
// 从 T100 设计器站点清单里发现的一条候选环境(字段见 internal/designer/sites.go)。
//
// 注意 zone 是**登录区域**(设计器的 Area),不是区域服务器——设计器那个 `Zone`
// 元素是服务名,单独在 serviceName 里带出来供参考。
export interface DiscoveredEnv {
  name: string
  uid?: string
  folder?: string
  protocol?: string
  host: string
  port: number
  user: string
  password: string
  zone?: string
  workspace?: string
  serviceName?: string
  // skipped 非空 = 这条不能导入,附原因(如协议不是 ssh)。仍然列出来,免得用户困惑于"少了一条"。
  skipped?: string
}

// PUT /api/hosts 的请求体:只带要改的节,省略的节后端保持原样 ——
// 设置页的各张卡片各改各的节时不会用陈旧快照把对方刚改好的节覆盖掉。
export interface HostsPatch {
  hosts?: { sshs: HostsSsh[] }
  listen?: string
  debug?: HostsDebug
  query?: { source?: string }
  mirror?: { dir?: string }
  dict?: { dir?: string }
  sync?: { target?: string }
  tdev?: { workspaceSuffix?: string; defaultOut?: string }
}

// 目录选择器（前端自绘）用的目录项 / 列表
export interface FsEntry { name: string; path: string }
export interface FsList {
  ok: boolean
  path: string
  parent: string
  drives: FsEntry[]
  dirs: FsEntry[]
  truncated?: boolean
}

// 一个"目录型"配置值的状态:配置里的值 + 它在本机是否真的存在
export interface DirStatus {
  dir: string
  exists: boolean
}
// 一个"文件型"配置值的状态
export interface FileStatus {
  target: string // 当前生效值
  configured: string // 配置里显式写下的值(空 = 用缺省位置)
  defaultTarget: string // 缺省位置
  exists: boolean
}
// 命令行安装状态(用户 PATH);不是 config.json 里的东西
export interface InstallStatus {
  supported: boolean // 本平台是否支持自动写入
  exePath: string
  exeDir: string
  inUserPath: boolean
  userPath?: string
  manual?: string
  note?: string
}
// GET /api/update:tt 自身更新的状态。检查结论来自缓存,所以这个端点**不联网**
export interface UpdateCheck {
  current: string
  latest: string
  newer: boolean
  prerelease?: boolean
  publishedAt?: string
  checkedAt: string
}

// 上一次升级的记录(.tt-update.json)。阶段见 internal/update 的 Phase 常量
// (idle/checking/downloading/verifying/handed-off/applying/done/failed)。
export interface UpdateLast {
  phase: string
  from?: string
  target?: string
  kind?: string
  artifact?: string
  error?: string
  warnings?: string[]
  startedAt?: string
  finishedAt?: string
}

export interface UpdateState {
  ok: boolean
  version: string
  kind: string
  canInstall: boolean
  refusal?: string
  source: string
  hint?: string
  skillsDrift?: string
  check?: UpdateCheck
  last?: UpdateLast
}

// GET /api/config/status:配置里那些路径型取值的派生状态
export interface ConfigStatus {
  ok: boolean
  config: string
  mirror: DirStatus
  dict: DirStatus
  sync: FileStatus
  install: InstallStatus
  cache: CacheStatus
}
// 配置目录下的缓存:可再生的中间数据,与 config.json 同目录但性质相反
export interface CacheDirStatus {
  name: string
  path: string
  exists: boolean
  bytes: number
  files: number
}
export interface CacheStatus {
  dir: string
  dirs: CacheDirStatus[]
  bytes: number
  files: number
  maxAgeHours: number
}
// 字典同步的进度(dbsync.Run),经 GET /api/dbsync 轮询
export interface DBSyncEnv {
  name: string
  type: string
  address: string
  dbPath: string
  exists: boolean
  size: number
  modified?: string
}
export interface DBSyncJob {
  running: boolean
  env: string
  phase: string // open|table|index|replace|done|error
  message: string
  table: string
  tableIndex: number
  tableTotal: number
  tableRows: number
  totalRows: number
  tables: number
  elapsed: string
  target: string
  backup?: string
  warning?: string
  error?: string
  done: boolean
  startedAt?: string
}
export interface DBSyncResp {
  dir: string
  envs: DBSyncEnv[]
  job: DBSyncJob
}

// GET /api/config/meta:配置元信息
export interface ConfigMeta {
  config: string
  defaultConfig: string
  portable: boolean
  toolsHome: string
  schemaVersion: number
  defaultListen: string
  supportedTypes: string[]
}

export interface Event {
  type: string; sessionId: string; time: string
  state?: string; stop?: StopInfo; vars?: VarItem[]; text?: string
  // 单调递增序号(服务端分配):WS 开场补发与去重用
  seq?: number
  // 谁发起的:ai | human | system
  actor?: string
  // 机器可读的动作名(bp.add / control.step / session.mode …),text 是给人看的
  action?: string
}


// 浏览器不声明 X-Actor —— 服务端不认这个头时一律按 human 处理(CLI 才声明 ai)。
// 注意 headers 必须在 opts 之后合并:早先写成 { headers: 默认, ...opts } 时,
// 调用方只要自带 headers 就会把默认头整块覆盖掉。
/**
 * 一次请求。**可选超时**（`timeoutMs`）。
 *
 * 为什么要有它：没有超时的话，一个永远不回来的请求会让调用方的 `finally` 永不执行 ——
 * 会话面板就是这么被卡死的（`runOp` 的 `busy` 一直置着 → 面板上**所有**按钮变灰、
 * 点了没反应、也没有任何提示，用户看到的就是"前端结束不了会话"）。
 * 超时不是"猜服务端错了"，而是把"没回来"变成一个**能说出口的事实**。
 */
async function req<T>(url: string, opts?: RequestInit & { timeoutMs?: number }): Promise<T> {
  const { timeoutMs, ...init } = opts || {}
  let timer: ReturnType<typeof setTimeout> | undefined
  let ac: AbortController | undefined
  if (timeoutMs && timeoutMs > 0) {
    ac = new AbortController()
    init.signal = ac.signal
    timer = setTimeout(() => ac?.abort(), timeoutMs)
  }
  try {
    const r = await fetch(url, {
      ...init,
      headers: { 'Content-Type': 'application/json', ...(init.headers || {}) },
    })
    const data = await r.json().catch(() => ({}))
    if (!r.ok || data.ok === false) throw new Error(data.error || `HTTP ${r.status}`)
    return data as T
  } catch (e: any) {
    if (ac?.signal.aborted) {
      throw new Error(`请求超时：${Math.round((timeoutMs || 0) / 1000)} 秒没有回应。服务端可能还在忙（或那次操作卡住了）——`
        + `点右上角刷新看现在的真实状态，别重复点`)
    }
    throw e
  } finally {
    if (timer) clearTimeout(timer)
  }
}

/** 一个技能目录的判定结果（官方与第三方共用）。state != ok 都不是错误，是“这个不装”。 */
export interface SkillInfo {
  origin: 'official' | 'external'
  name: string
  description?: string
  dir: string
  index?: number
  state: 'ok' | 'missing' | 'invalid' | 'conflict'
  why?: string
}

/** 第三方那一行：磁盘判定 + 登记表里的原值。 */
export interface SkillRow extends SkillInfo {
  /** 登记时给的显示名；路径失联时靠它认人（永远不是权威，技能只认 frontmatter） */
  label?: string
  /** 登记表里的原值（未绝对化），删除时按它匹配 */
  path?: string
}

/** GET /api/skills */
export interface SkillsResp {
  ok: boolean
  config: string
  official?: SkillInfo[]
  officialDir?: string
  /** 官方那一棵读不到/不合规时的原因（不会影响第三方那一栏） */
  officialError?: string
  externals: SkillRow[]
}

export const api = {
  status: () => req<any>(`${API_BASE}/status`),
  // 环境清单:读共享端点。合并前这里调 /api/settings(旧 debug 节的 sshs),
  // 现在 hosts 节是唯一数据源,整个界面看到的是同一份环境。
  settings: () => req<HostsView>('/api/hosts'),
  // 保存:只发要改的节(见 HostsPatch),后端把 400 的中文 error 原样回给界面
  saveSettings: (cfg: HostsPatch) => req<HostsView>('/api/hosts', { method: 'PUT', body: JSON.stringify(cfg) }),
  // 发现环境:扫描本机的 T100 设计器站点清单(只读那份加密文件),返回候选环境让用户勾选。
  // found=false 且 reason 非空 = "没装设计器/没存过站点",那是正常结论而不是错误。
  discoverEnvs: () =>
    req<{ sitesFile: string; found: boolean; envs: DiscoveredEnv[]; reason?: string }>('/api/discover/envs'),
  // 自动获取数据库连接要素(SSH 上服务器探测「从服务器获取」;note 说明未获取到的原因)
  probeDB: (body: { host: string; port: number; user: string; password: string; zone: string; type: string }) =>
    req<{ type: string; tns?: string; port?: number; database?: string; sqlplus?: string; oracleHome?: string; twoTask?: string; host?: string; service?: string; note?: string }>('/api/dbprobe', { method: 'POST', body: JSON.stringify(body) }),
  // 客户端直连测试(按表单显式字段连库,凭据取账号列表首项)。
  // 共享端点要求 {connection: …} 包一层(沿用字典侧的请求形状),不再是平铺连接字段。
  connTest: (connection: HostsDb) =>
    req<{ ok: boolean; type?: string; serverVersion?: string; address?: string; stage?: string; error?: string; via?: string }>('/api/conntest', { method: 'POST', body: JSON.stringify({ connection }) }),
  // 账号清单「验证」:SSH 上服务器以该账号+密码连显式目标库 select 1(只读)
  dbAccVerify: (body: { host: string; port: number; user: string; password: string; zone: string; type: string; account: string; acctPassword: string; dbHost?: string; dbPort?: number; dbSvc?: string; dbDatabase?: string }) =>
    req<{ ok: boolean; error?: string }>('/api/dbaccverify', { method: 'POST', body: JSON.stringify(body) }),
  // 配置派生状态:路径型取值的"在本机是否真的存在"只有服务端算得出来(要 os.Stat)。
  // 单独一个端点而不并进 /api/hosts —— 后者是每次进设置都读的热路径。
  configStatus: () => req<ConfigStatus>('/api/config/status'),
  // 清理缓存:只删可再生的中间数据(config.json 与服务状态文件一律不动)。
  // 连同清完的状态一起回,前端不必再打一次 configStatus 才能刷新数字。
  cacheClear: () =>
    req<{ removed: number; freed: number; cache: CacheStatus }>('/api/cache/clear', { method: 'POST' }),
  // 配置元信息:配置文件缺省落点、便携标记、schema 版本、支持的库类型…
  configMeta: () => req<ConfigMeta>('/api/config/meta'),
  // 原生文件夹选择器：浏览器拿不到绝对路径，必须由本机服务端弹对话框。
  // 取消时返回 {ok:true,canceled:true}（不是错误，不要 toast）。
  pickDir: (initial: string, title = '选择文件夹') =>
    req<{ ok: boolean; path?: string; canceled?: boolean }>('/api/pickdir', {
      method: 'POST', body: JSON.stringify({ initial, title }),
    }),
  // 列本机目录（只列目录，只接受本机请求）。浏览器拿不到绝对路径，
  // 所以"选目录"要由服务端列出来、前端画成自己的选择器。
  listDirs: (path: string) =>
    req<FsList>('/api/fs/list?path=' + encodeURIComponent(path)),
  // 命令行安装(PATH):应用级动作,统一层提供(合并前在字典子系统的私有 API 下)
  installStatus: () => req<InstallStatus>('/api/install'),
  installAdd: () => req<InstallStatus>('/api/install', { method: 'POST' }),
  installRemove: () => req<InstallStatus>('/api/install', { method: 'DELETE' }),
  // tt 自身的更新。查是**人去点**才联网(服务不轮询);装由服务端拉一个
  // `tt update --yes` 子进程完成 —— 本服务几分钟后会被它停掉替换(见 internal/web/update.go)。
  updateState: () => req<UpdateState>('/api/update'),
  updateCheck: () => req<{ ok: boolean; check?: UpdateCheck; hint?: string; where?: string; error?: string }>(
    '/api/update/check', { method: 'POST' }),
  updateApply: () => req<{ ok: boolean; pid?: number; note?: string }>('/api/update/apply', { method: 'POST' }),
  // 字典类动作(字典同步)。合并前它在 /dict/api/* 下 —— 那是字典页
  // 私有 API;字典页并进设置页之后升到共享层,设置页因此不必跨子系统调用。
  dbsync: () => req<DBSyncResp>('/api/dbsync'),
  dbsyncRun: (env: string) =>
    req<{ ok: boolean }>('/api/dbsync', { method: 'POST', body: JSON.stringify({ env }) }),
  dbsyncStop: () => req<{ ok: boolean; error?: string }>('/api/dbsync/stop', { method: 'POST' }),
  list: () => req<{ sessions: SessionBrief[] }>(`${API_BASE}/sessions`),
  launch: (module: string, prog: string, opts?: { ssh?: string; zone?: string; session?: string; ent?: string }) =>
    req<{ sessionId: string; module?: string; prog?: string; runProg?: string }>(`${API_BASE}/sessions`, { method: 'POST', body: JSON.stringify({ module, prog, ...opts }), timeoutMs: 180000 }),
  snapshot: (id: string) => req<any>(`${API_BASE}/sessions/${id}`),
  // 结束调试:只结束本轮运行,宿主会话保留(idle),可直接再次启动
  quit: (id: string) => req<any>(`${API_BASE}/sessions/${id}`, { method: 'DELETE' }),
  // 会话管理(单一常驻会话)
  //
  // ent = 本次要用的企业编号(面板上填的那个)。环境配置里已不再存企业编号
  // (hosts.sshs[].topent 已移除),所以 ENT 必须随这次请求带上去;省略 = 不覆盖,
  // 沿用登录 shell 自己的 $TOPENT。
  // 三个会话操作都带超时：它们要连 SSH、起进程，慢是正常的（给 90 秒），
  // 但**不能永远不回来** —— 面板的 busy 就是靠它们的 promise 落定才解开的（见 req 的说明）。
  sessionRestart: (id: string, ent?: string) =>
    req<{ sessionId: string; env?: string; state?: string }>(
      `${API_BASE}/sessions/${id}/restart`, { method: 'POST', body: JSON.stringify({ ent: ent || '' }), timeoutMs: 90000 }),
  sessionClose: (id: string) =>
    req<any>(`${API_BASE}/sessions/${id}/close`, { method: 'POST', timeoutMs: 90000 }),
  // 连到该环境（Q26：与 `start` 同类"选环境并连上去"，**不是**"把一个会话切到另一个环境"）。
  // 连上去之前，**已在跑的那条会话会被服务端收掉** —— 这是有意的（Q27：我只想连到那个环境）。
  // 所以这里**必须带上"要收掉的那条会话"**（WP4b / Q4 / C8.2）：服务端那五处"没给目标就取当前
  // 会话"的兜底已经全删了，不带目标时池里有多条会话就会回 409 + SESSION_AMBIGUOUS（而不是替前端挑
  // 一条）；**不新增参数**（Q28：前端照旧自己记住人点的那条，并在请求里说明要收掉它）。
  // 传的是**服务端原始句柄**（C5.4：路径与句柄只认 id，不认 ref/display）。
  sessionSwitch: (env: string, ent?: string, sessionId?: string) =>
    req<{ sessionId: string; env?: string; state?: string }>(
      `${API_BASE}/sessions/switch`, { method: 'POST', body: JSON.stringify({ env, ent: ent || '', session: sessionId || '' }), timeoutMs: 90000 }),
  // 新建会话（`POST /sessions/new`）：**再给我一条**空闲会话，池里别的会话一条都不动。
  //
  // 与 sessionSwitch 的分工（后端 hSessionCreate 里写清了）：switch 是"连到这个环境" ——
  // 池里已有就复用，名字不同还会把空闲那条收掉重建；new 是"再建一条"，同名才算同一条
  // （幂等键是名字，`reused:true` 表示"本来就有"）。多会话下建第二条**只能用这条**，
  // 用 switch 会把上一条收掉。
  sessionCreate: (env: string, ent: string, name: string) =>
    req<{ sessionId: string; ref?: string; env?: string; name?: string; state?: string; reused?: boolean }>(
      `${API_BASE}/sessions/new`, { method: 'POST', body: JSON.stringify({ env, ent, name }), timeoutMs: 90000 }),
  // 空闲态重新设置会话 TOPENT(空值 = 清除覆盖并重新下发配置默认)
  topent: (id: string, value: string) => req<{ topent: string }>(`${API_BASE}/sessions/${id}/topent`, { method: 'POST', body: JSON.stringify({ value }) }),
  bpAdd: (id: string, location: string) =>
    req<{ breakpoint: Breakpoint }>(`${API_BASE}/sessions/${id}/breakpoints`, { method: 'POST', body: JSON.stringify({ location }) }),
  bpDel: (id: string, num: number) => req<any>(`${API_BASE}/sessions/${id}/breakpoints/${num}`, { method: 'DELETE' }),
  control: (id: string, action: string, arg?: string) =>
    req<any>(`${API_BASE}/sessions/${id}/control`, { method: 'POST', body: JSON.stringify({ action, arg }) }),
  // 切模式(纯人工/协作)。两个方向都放行:人与 AI 都能切(人在界面上的「接管」可随时收回)
  setMode: (id: string, mode: 'solo' | 'collab') =>
    req<{ mode: string }>(`${API_BASE}/sessions/${id}/mode`, { method: 'POST', body: JSON.stringify({ mode }) }),
  print: (id: string, expr: string) => req<{ value: string }>(`${API_BASE}/sessions/${id}/print`, { method: 'POST', body: JSON.stringify({ expr }) }),
  where: (id: string) => req<{ frames: Frame[] }>(`${API_BASE}/sessions/${id}/where`, { method: 'POST' }),
  raw: (id: string, command: string) => req<{ lines: string[] }>(`${API_BASE}/sessions/${id}/raw`, { method: 'POST', body: JSON.stringify({ command }) }),
  locals: (id: string) => req<{ vars: VarItem[] }>(`${API_BASE}/sessions/${id}/locals`),
  globals: (id: string, limit?: number) => req<{ vars: VarDecl[]; total: number }>(`${API_BASE}/sessions/${id}/globals${limit ? `?limit=${limit}` : ''}`),
  sources: (id: string) => req<{ sources: string[] }>(`${API_BASE}/sessions/${id}/sources`),
  functions: (id: string, limit?: number) => req<{ functions: string[]; total: number }>(`${API_BASE}/sessions/${id}/functions${limit ? `?limit=${limit}` : ''}`),
  autovars: (id: string) => req<{ vars: VarItem[] }>(`${API_BASE}/sessions/${id}/autovars`),
  // 自动变量面板开关:停站后是否由服务端自动求值当前源码窗变量(默认关)
  autovarsAuto: (id: string, auto: boolean) => req<any>(`${API_BASE}/sessions/${id}/autovars`, { method: 'POST', body: JSON.stringify({ auto }) }),
  frame: (id: string, num: number) => req<{ frame: number }>(`${API_BASE}/sessions/${id}/frame`, { method: 'POST', body: JSON.stringify({ num }) }),
  bpEnabled: (id: string, num: number, enabled: boolean) =>
    req<any>(`${API_BASE}/sessions/${id}/breakpoints/${num}/enabled`, { method: 'POST', body: JSON.stringify({ enabled }) }),
  sourceByFile: (id: string, file: string, module: string) =>
    req<{ source: { path: string; content: string; dvmFile?: string } }>(`${API_BASE}/sessions/${id}/source?file=${encodeURIComponent(file)}&module=${encodeURIComponent(module)}`),
  // 定位函数到源文件与行号(fgldb info line;仅停站可用)
  locate: (id: string, word: string) =>
    req<{ file: string; line: number }>(`${API_BASE}/sessions/${id}/locate`, { method: 'POST', body: JSON.stringify({ word }) }),
  // 行号校准:检测 fgldb(DVM)行号与磁盘源码的偏移(仅停站可用)
  calibrate: (id: string) => req<{ offset: number }>(`${API_BASE}/sessions/${id}/calibrate`, { method: 'POST' }),

  // 条件用对象传递(参数变多后位置参数太容易错位);空值不发送。
  // 判空要**先 trim**：输入框里看着是空的（几个空格），不 trim 就会发一个 `service=+++`
  // 的空白 LIKE 条件 —— 后端匹配不到，用户得到空列表且界面上没有任何提示（实测单测抓到）。
  wsLogs: (q: WSLogQuery) => {
    const p = new URLSearchParams()
    for (const [k, v] of Object.entries(q)) {
      if (v === undefined || v === null || v === false) continue
      if (typeof v === 'string') {
        const s = v.trim()
        if (!s) continue
        p.set(k, s)
        continue
      }
      if (v === '') continue
      p.set(k, typeof v === 'boolean' ? (v ? '1' : '0') : String(v))
    }
    p.set('onlyFail', q.onlyFail ? '1' : '0')
    p.set('page', String(q.page ?? 1))
    p.set('pageSize', '200')
    return req<{ items: WSLogItem[]; hasMore: boolean }>(`${API_BASE}/wslogs?${p.toString()}`)
  },
  wsLogContent: (rowid: string, env?: string) =>
    req<{ item: WSLogItem; content: WSLogContent }>(`${API_BASE}/wslogs/content?rowid=${encodeURIComponent(rowid)}${env ? `&env=${encodeURIComponent(env)}` : ''}`),
  // 企业目录(ENT → 数据库账号):与 tt debug 的「企业→账号」同一条规则、同一张 gzou_t
  // (固定用系统账号 ds 读),两边的缓存与快照也是同一份(后端的 entdir)。
  // 接口日志页与「创建会话」的 ENT 下拉都用它 —— **不需要一条活会话**。
  // 同步 ENT：连库读 gzou_t 并**写回 config.json**（设置页「ENT」页签的按钮）。
  entsSync: (env: string) =>
    req<{ ok: boolean; env: string; count: number; addedAccounts?: string[]; syncedAt: string }>(
      `${API_BASE}/ents/sync`, { method: 'POST', body: JSON.stringify({ env }) }),
  ents: (env: string, opt?: { refresh?: boolean }) =>
    req<EntsReply>(`${API_BASE}/ents?env=${encodeURIComponent(env)}${opt?.refresh ? '&refresh=1' : ''}`),
  // 重放调试:request 非空 = 用界面里改过的入参(后端落临时文件后作为入参文件)。
  // sessionId = 本地已绑定的那条会话（要收掉的那条，可空）：服务端不自己找当前会话
  // （WP4b / Q4 / C8.2），池里多条又不带目标时回 409 + SESSION_AMBIGUOUS，不会随机收掉别人的会话。
  // **没有会话时照样能调**：重放自己就是造会话的命令（池空 = 冷启动，服务端起一条新的），
  // 所以这里 `old || undefined` —— 让"点一下这条日志就开始重放"在任何时候都点得动。
  // ent = 重放这一轮的企业编号（接口日志页下拉选的那个），空则不覆盖。
  // env = **这条日志所属的环境名**（`activeEnv`），空则不点名。非空时必须送：
  //   重放要拨号读 `wsfa_t` 并起一条会话，而统一服务刚启动时**没有生效环境**
  //   （环境一律点名）—— 不送就会得到"服务当前没有生效环境"，
  //   而日志列表那条路早就送了（`wsLogs` 的 `env`），只有重放漏了。
  wsLogDebug: (rowid: string, request?: string, sessionId?: string, ent?: number, env?: string) =>
    req<{ sessionId: string; module?: string; prog?: string; runProg?: string }>(`${API_BASE}/wslogs/debug`, {
      method: 'POST',
      body: JSON.stringify({
        rowid, ...(request ? { request } : {}), ...(sessionId ? { session: sessionId } : {}),
        ...(ent && ent > 0 ? { ent: String(ent) } : {}),
        ...(env ? { ssh: env } : {}),
      }),
    }),
  sourcePreview: (module: string, prog: string) =>
    req<{ source: { path: string; content: string; dvmFile?: string } }>(`${API_BASE}/source-preview?module=${encodeURIComponent(module)}&prog=${encodeURIComponent(prog)}`),

  // ---- 技能(官方随包 + 第三方登记) ----
  // 只管「登记了哪些」:**能装到哪、要不要覆盖是命令行的事**(tt install skills),
  // 前端不再提供第二条安装入口。
  skills: () => req<SkillsResp>('/api/skills'),
  // 添加时**当场校验**:不合规/撞名会被后端直接拒(400),失联允许存下来(ok + warning)。
  skillsAdd: (path: string, label?: string) =>
    req<{ ok: boolean; unchanged?: boolean; entry: SkillRow; warning?: string }>(
      '/api/skills/add', { method: 'POST', body: JSON.stringify({ path, label }) }),
  // 只从登记表里去掉,不碰磁盘。removed=false 表示本来就没这条(幂等)。
  skillsRemove: (path: string) =>
    req<{ ok: boolean; removed: boolean }>('/api/skills/remove', { method: 'POST', body: JSON.stringify({ path }) }),

}