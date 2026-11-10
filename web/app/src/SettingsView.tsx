// 统一设置页:左侧四个分区的导航,右侧按分区渲染卡片。
//
// 「站点管理」环境清单 + SSH/数据库(原调试页「环境」节与原字典页「环境配置」的并集,只留一份);
// 「数据字典」查询数据源 / 源码镜像 / 数据同步;
// 「DEBUG」调试参数 / 默认环境；
// 「技能」官方技能（只读）+ 第三方技能的登记与安装；
// 「应用设置」外观 / 服务 / 运行信息。
//
// 环境清单走**共享**端点 /api/hosts(顶层,不带 API_BASE 前缀):一个进程里它是唯一的数据源
// —— config.json 的 hosts 节。合并前这份清单藏在 debug 节里
// (所以老代码收发的是整个 debug 节点),现在收发的是 hosts 节。
//
// 保存**按节提交**:设置页的各张卡片各改各的节,不拿陈旧快照覆盖对方刚改好的节
// (后端约定:PUT /api/hosts 里省略的节保持原样)。
//
// 字段级也一样:**表单里没有的键不必回传**,后端会从旧配置原样补回(`putSection`)——
// 这条是 2026-10-15 补上的,它之前是"整节替换 + 每个字段 omitempty = 省略一个键就删掉它",
// 于是 `debug.aiTakeover`、`query.limit` 这种"后端有、界面没有"的字段每次保存都被静默抹掉。
// 现在两端各有一道:后端补回缺席的键,前端也**只提交自己管得着的**字段(不再把读到的
// 一整套默认值写回去 —— 那会在配置里凭空写出 `termWidth: 200` 这种用户从没设过的值)。
import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react'
import {
  AlertCircle, CheckCircle2, Copy, Download, Eye, EyeOff,
  FolderOpen, Monitor, Plus, RefreshCw, Search, Sun, Moon,
  Terminal, Trash2, Zap,
} from 'lucide-react'
import {
  api, type ConfigMeta, type ConfigStatus, type DBSyncEnv, type DBSyncJob, type DBSyncResp,
  type DiscoveredEnv,
  type HostsDb, type HostsEnt, type HostsPatch, type HostsSsh, type HostsView, type MirrorEnv, type MirrorJob, type MirrorResp,
  type UpdateState,
} from './api'
import {
  Button, Input, Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '../../shared/ui'
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
  Checkbox,
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
  Progress,
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
  Tabs, TabsContent, TabsList, TabsTrigger,
} from '../../shared/ui-radix'
import { Card, Field, InfoRow, SectionTitle, SettingRow } from '../../shared/settings'
import { DirPicker } from './DirPicker'
import { SkillsPanel } from './SkillsPanel'
import { GpSites, emptyGpEnv, type GpEnv } from './GpSettings'
import { gpMirror, gpPatchEnv, gpPickDir, gpSync, refreshGpEnvs, useGpEnvs } from './gpEnvs'
import { useToast } from './toast'
import { useStore, type GpEnvBrief, type ThemeMode } from './store'
import { cn } from '../../shared/utils'
import { SETTINGS_SECTIONS, parseRoute, settingsHash } from './routing'

// ---------- 表单模型 ----------

// T100 与 GP 共用同一张同步/镜像表：行上带产品别，按钮据此分流到各自的接口。
type SyncRow =
  | { product: 't100'; t100: DBSyncEnv }
  | { product: 'gp'; gp: GpEnvBrief }

type MirrorRow =
  | { product: 't100'; t100: MirrorEnv }
  | { product: 'gp'; gp: GpEnvBrief }

const rowName = (r: SyncRow | MirrorRow) => r.product === 't100' ? r.t100.name : r.gp.name

// 环境选中态：一个状态同时表达“选了哪个产品、哪一条”，从类型上不可能同时选中 T100 和 GP。
// T100 用下标（新建环境还没有稳定名字）；GP 用环境名。
type EnvSelection =
  | { product: 't100'; index: number }
  | { product: 'gp'; name: string }
  | null

interface SshDb {
  type: string // oracle | kingbase
  host: string
  port: number
  service: string // oracle: SERVICE_NAME
  database: string // kingbase: 库名
  accounts: { account: string; password: string }[]
  // 是否允许 AI 执行只读 SQL(排查业务数据)。undefined = 未配置 = 开启;
  // 显式 false 才关。**载入/保存都必须带上它** —— 否则在设置页点一次保存,
  // 这个开关就被静默抹掉、悄悄变回开启(我实现时实测过)。
  readonlySql?: boolean
  // SSH 隧道(手工写在 config.json、表单不编辑;保存由后端 preserveUngovernedFields 保留)。
  // 载入时要带出来 ——「测试连接」必须带上它,否则测的是本机直连那条走不通的链路。
  viaSsh?: {
    host: string; port?: number; user: string; password?: string
    bindPort?: number; remoteHost?: string; remotePort?: number
  }
}

interface SshItem {
  name: string
  host: string
  port: number
  user: string
  password: string
  zone: string
  // ENT 清单（只读）：由「ENT」页签的“从服务器刷新”写入配置，不提供手工维护。
  ents?: HostsEnt[]
  entsSyncedAt?: string
  // workspace 该环境的 .tzs 工作区（含 mta/ 的目录，绝对路径）。
  // 与 launchArgs / watchdogSeconds 不同：它是**必填**的（引擎的默认工作区是真实客户目录），
  // 而且只有环境级这一份 —— 切环境就该跟着换工作区。
  workspace: string
  // 该环境自己的源码镜像目录；空 = 使用全局 mirror.dir/<环境名>。
  mirrorDir: string
  // 该环境自己的本地字典库目录；空 = 使用全局 dict.dir。
  dictDir: string
  db: SshDb | null
}

// debug 节的表单。只放**这一页真的有控件**的字段（目前只有停站看门狗）。
//
// 刻意不做成"与 DebugSettings 一一对应的全集"：那种平行名单要么漏字段（保存一次就丢一个）、
// 要么把读到的默认值原样写回（凭空在配置里写出用户没设过的键）。缺席的键由服务端
// `putSection` 从旧配置补回 —— 保住配置这件事只有一处实现。
interface DebugForm {
  watchdogSeconds: number
}

interface DictForm {
  /** '' = 在线 | 'local:<环境>' = 该环境的本地 SQLite | 环境名 = 固定走该环境的在线库。 */
  querySource: string
  mirrorDir: string
  // dictDir 本地字典库根目录；空 = 缺省 <数据目录>/dict
  dictDir: string
  // 刻意没有 syncTarget:字典库按环境分文件,位置由服务端算;卡片里只读显示库目录。
  // **不要**为了"完整性"把它加回来再原样回传 —— 那等于把一个不展示的键交给这一页写。
}

/** 可自动保存的配置节。脏标记与提交范围都按它划分。 */
type ConfigKey = 'hosts' | 'debug' | 'listen' | 'query' | 'mirror' | 'dict'

// 主题切换卡片(shadcn 主题切换卡样式):三张卡各带一张迷你界面预览,选中卡描边+底色高亮。
const THEME_OPTIONS: { value: ThemeMode; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: '亮色', icon: Sun },
  { value: 'dark', label: '暗色', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Monitor },
]

// 外观选择:三张卡片,每张只有图标 + 名称。
//
// 原先每张卡里还画了一个"迷你界面"缩略图(左栏条 + 内容块,用被预览主题的配色)。
// 删掉的理由:那点 40px 高的示意既画不清主题差别(靠的是三块色块),又占掉了卡片
// 一半的高度;而"亮色/暗色/跟随系统"这三个词本身没有任何歧义,图标也一眼能认。
// 简化之后一行内三张卡更紧凑,选中的那档也只用一个边框 + 底色表达。
function ThemeCards({ value, onChange }: { value: ThemeMode; onChange: (t: ThemeMode) => void }) {
  return (
    <div className="grid max-w-md grid-cols-3 gap-2">
      {THEME_OPTIONS.map((o) => {
        const Icon = o.icon
        const active = value === o.value
        return (
          <button
            key={o.value}
            type="button"
            aria-pressed={active}
            title={`切换到「${o.label}」`}
            onClick={() => onChange(o.value)}
            className={cn('flex items-center justify-center gap-1.5 rounded-ui border px-3 py-2 text-xs transition-colors',
              active ? 'border-foreground/50 bg-accent/60 text-foreground' : 'border-border text-muted-foreground hover:bg-accent/40 hover:text-foreground')}
          >
            <Icon className="h-3.5 w-3.5 shrink-0" />
            {o.label}
          </button>
        )
      })}
    </div>
  )
}

/**
 * Go 的 time.Duration.String() → 秒数(后端把耗时按这个格式直接下发:
 * "2.1s" / "1m20.5s" / "1h2m3s")。解析不出来就返回 0,调用方据此不显示速度。
 */
function durSeconds(s: string): number {
  if (!s) return 0
  const re = /(\d+(?:\.\d+)?)(h|m|s)/g
  let total = 0, found = false, m: RegExpExecArray | null
  while ((m = re.exec(s))) {
    found = true
    const n = Number(m[1])
    total += m[2] === 'h' ? n * 3600 : m[2] === 'm' ? n * 60 : n
  }
  return found ? total : 0
}

// 镜像任务的数值行:**裸值**,从左到右按"先看什么"排 —— 大小(当前/总量)、速度、耗时、文件数。
// 速度是算出来的(bytes / elapsed):后端只下发耗时字符串,不为一个展示值再加 JSON 字段。
// 算不出来时(耗时为 0、刚开始)省掉那一格,不显示 "0 KB/s" 这种噪音。
function mirrorCellTexts(j: MirrorJob): ReactNode[] {
  const secs = durSeconds(j.elapsed)
  const rate = secs > 0 ? j.bytes / secs : 0
  return [
    fmtBytes(j.bytes) + (j.total > 0 ? ` / ${fmtBytes(j.total)}` : ''),
    rate > 0 ? `${(rate / 1024).toFixed(0)} KB/s` : '',
    j.elapsed ? (secs > 0 ? `${secs.toFixed(1)}s` : j.elapsed) : '',
    j.files > 0 ? `${j.files} 个文件` : '',
  ].filter(Boolean)
}

/**
 * 长跑任务的进度块(镜像拉取 / 字典同步共用)。
 *
 * 两类任务的"进度量"不是同一种东西:镜像按**字节**走,字典同步按**表**走。
 * 所以进度条只认一个 pct(由调用方按各自的量算好),下面那行数值由 pct 规则无关的
 * cells 描述符给出 —— 想显示什么就写什么,不必让这个组件认识两种任务的字段。
 *
 * 数值行刻意**不带描述性标签**("已传输:" / "文件数:" / "网络速度:" 这类):
 * 单位本身已经把含义说清(B/s 是速度、`a / b` 是总量、后面带 s 是秒数),
 * 再给每个数配个词只会把一行塞满,真正要看的数字被埋进文字里。
 */
function JobProgress({ env, phaseLabel, tag, pct, cells, error, message, stopping, onStop }: {
  env: string; phaseLabel: string
  /** 右上角的小标记(全量/增量、进行中/已结束) */
  tag?: string
  /** 0-100。算不出来(如刚开始、总量未知)时传 undefined,进度条显示为 0 */
  pct?: number
  /** 数值行:**裸值,不写前缀标签**。按"先看什么"从左到右排 */
  cells: ReactNode[]
  error?: string; message?: string
  /** 已经在停但还没停完(后端取消是异步的:SFTP 读到下一个分片才返回) */
  stopping?: boolean
  /** 有它才显示「停止」。取消后不该只是把按钮变灰 —— 要点得动,并由 stopping 反馈 */
  onStop?: () => void
}) {
  const failed = !!error
  const v = Math.max(0, Math.min(100, pct ?? 0))
  return (
    <div className="border border-border bg-muted/20 p-2">
      <div className="mb-1.5 flex items-center gap-2 text-xs">
        <span className="font-medium">{env}</span>
        <span className="text-muted-foreground">{phaseLabel}</span>
        {onStop && (
          // 停止放在这一行的最右:它作用的对象就是这条进度(误点「全量重建」后想收手)
          <Button variant="outline" size="sm" className="ml-auto h-6 px-2 text-[11px]"
            disabled={stopping} onClick={onStop}>
            {stopping ? '正在停止…' : '停止'}
          </Button>
        )}
        {tag && <span className={cn('text-[11px] text-muted-foreground', !onStop && 'ml-auto')}>{tag}</span>}
      </div>
      {/* 失败时把填充改成 destructive:进度条是这一块里唯一的视觉主体,
          出错还留着主色填充会让人以为还在正常跑 */}
      <Progress value={v} barClassName={failed ? 'bg-destructive' : undefined} />
      <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-[11px] text-muted-foreground">
        <span className="font-mono text-foreground">{v}%</span>
        {cells.map((c, i) => <span key={i} className="font-mono">{c}</span>)}
      </div>
      {message && (
        <p className={cn('mt-1.5 flex items-center gap-1 text-xs',
          failed ? 'text-red-600 dark:text-red-400' : 'text-emerald-600 dark:text-emerald-400')}>
          {failed ? <AlertCircle className="h-3.5 w-3.5" /> : <CheckCircle2 className="h-3.5 w-3.5" />}
          {error || message}
        </p>
      )}
    </div>
  )
}

/**
 * GP 的镜像/同步进度块。T100 的 JobProgress 已经有“环境 + 阶段 + 进度条 + 数值 + 结果”，
 * GP 的 Job 只是没有 phase/total，用 progress 的最后一条和 result 汇总出同样的信息，
 * 不另画一套 UI。
 */
function GpJobProgress({ env, job }: { env: string; job: NonNullable<GpEnvBrief['job']> }) {
  const running = job.state === 'running'
  const done = job.state === 'done'
  const doneN = Number(job.done) || 0
  const totalN = Number(job.total) || 0
  const result = (job.result ?? {}) as Record<string, any>
  const cells: ReactNode[] = []

  if (job.kind === 'mirror') {
    let files = 0, bytes = 0, skipped = 0
    for (const v of Object.values(result)) {
      if (!v || typeof v !== 'object') continue
      const m = v as Record<string, any>
      files += Number(m.files) || 0
      bytes += Number(m.bytes) || 0
      skipped += Number(m.skipped) || 0
    }
    if (totalN) cells.push(`${doneN}/${totalN} 个`)
    if (files) cells.push(`下载 ${files}`)
    if (skipped) cells.push(`跳过 ${skipped}`)
    if (bytes) cells.push(`${(bytes / 1048576).toFixed(1)} MB`)
  } else {
    if (totalN) cells.push(`${doneN}/${totalN} 表`)
    if (result.columns != null) cells.push(`${result.columns} 字段`)
    if (result.sizeMB != null) cells.push(`${Number(result.sizeMB).toFixed(1)} MB`)
  }
  if (job.elapsed) cells.push(job.elapsed)

  const last = job.progress?.length ? job.progress[job.progress.length - 1] : ''
  if (running && last) cells.push(last)

  const pct = totalN > 0 ? Math.round((doneN * 100) / totalN) : done ? 100 : 0

  return (
    <JobProgress
      env={env}
      phaseLabel={job.state === 'error' ? '失败' : job.kind === 'mirror' ? '源码镜像' : '字典同步'}
      tag={running ? '进行中' : '已结束'}
      pct={pct}
      cells={cells}
      error={job.error}
      message={!running ? (last || (done ? '完成' : undefined)) : undefined}
    />
  )
}

// 阶段标签与字节格式化:原先在字典页的两个视图里,随功能一起搬进来。
const MIRROR_PHASE: Record<string, string> = {
  connect: '连接服务器…', probe: '探测 T100 目录…', pack: '服务器打包…',
  download: '下载并解压…', done: '完成', error: '失败',
}
const SYNC_PHASE: Record<string, string> = {
  open: '连接远程数据库…', table: '拉取数据表…', index: '创建主键索引…',
  replace: '替换本地数据库…', done: '完成', error: '失败',
}
/**
 * 表格里的「状态」格:同一时刻只可能有一个环境在跑(后端一次只跑一个动作),
 * 所以正在跑的那个显示阶段与进度,其余显示空闲词(已有镜像 / 未拉取…) ——
 * 于是"哪个环境在忙、忙到哪一步"一眼能落在具体那一行上,不用去别处找进度条。
 */
function EnvStatus({ busy, running, idle, error }: {
  busy: boolean; running: boolean; idle: string; error?: string
}) {
  if (!busy) return <span className="text-[11px] text-muted-foreground">{idle}</span>
  return (
    <span className={cn('flex items-center gap-1 text-[11px]',
      error ? 'text-red-600 dark:text-red-400' : 'text-foreground')}>
      {running
        ? <RefreshCw className="h-3 w-3 shrink-0 animate-spin" />
        : error
          ? <AlertCircle className="h-3 w-3 shrink-0" />
          : <CheckCircle2 className="h-3 w-3 shrink-0 text-emerald-600 dark:text-emerald-400" />}
      {error || idle}
    </span>
  )
}
function fmtBytes(n: number): string {
  if (!n) return '0 B'
  if (n >= 1 << 30) return (n / (1 << 30)).toFixed(2) + ' GB'
  if (n >= 1 << 20) return (n / (1 << 20)).toFixed(1) + ' MB'
  if (n >= 1 << 10) return (n / (1 << 10)).toFixed(1) + ' KB'
  return n + ' B'
}

// 设置控件的统一尺寸。w-full 是必须的:控件都住在「说明 + 控件」这一列里,
// 不写宽度时它们靠 flex 默认的 align-items:stretch 撑满,而 SelectTrigger 自带 w-full、
// Input 却只有 hidden 的块级宽度 —— 两种来源不一样,换一次外层布局就会一部分宽一部分窄。
// 显式写死 w-full,控件宽度就只由那一列决定。
const input = 'h-7 w-full min-w-0 text-xs'
// 单元格「正在编辑」提示环:用 ring 语义 token,不写死颜色(共享层里也只有这一份)
const CELL_EDITING = 'focus-within:ring-1 focus-within:ring-inset focus-within:ring-ring/60'

const blankDb = (): SshDb => ({ type: 'oracle', host: '', port: 1521, service: 'topprd', database: '', accounts: [] })
// 新增环境默认带三个常用账号(账号=密码),省去手工录入
const defaultAccounts = () => ['ds', 'dsdata', 'dsdemo'].map((a) => ({ account: a, password: a }))
// 新增环境用的空档:登录区域默认 1、服务名默认 topprd —— 都是现场最常见的取值,
// 先填上能省掉每次手敲(不对再改)。注意 service 是**真值**不是占位符:
// placeholder 在框里是灰字、不会被提交,而这里要的是"保存下去就是它"。
const blankSsh = (): SshItem => ({
  name: '', host: '', port: 22, user: '', password: '', zone: '1', workspace: '', mirrorDir: '', dictDir: '',
  db: { ...blankDb(), accounts: defaultAccounts() },
})

// ---------- 表单 ↔ 配置节的互转 ----------

/**
 * 环境清单 → hosts 节。**整节提交**,所以这里必须给出完整的 sshs 列表。
 *
 * 仍未覆盖的字段:`hosts.sshs[].launchArgs` / `watchdogSeconds`(每环境的调试覆盖项,
 * 本页不暴露)。它们由服务端的保留机制补回(见 internal/web 的 preserveUngovernedFields)。
 * 而 `workspace` 是这一页管得着的(表单里有输入框) —— 不要把它当成"未覆盖"加进那份保留名单,
 * 否则用户清空它时会被静默补回。
 *
 * 名字留空会被补成 `host-zone`(sshName 的口径,列表上显示的也是它):对**新环境**是省事,
 * 对**已有环境**等于改名 —— 服务端把"改名"判成搬家(录像跟着走、身份里的环境名一起改),
 * 不再当成"环境被删掉"去级联清库(见 internal/web 的 diffEnvs 与 Library.RenameEnv)。
 */
function hostsPatchOf(list: SshItem[]) {
  const sshsOut: HostsSsh[] = list.filter((x) => x.host).map((x) => {
    const o: HostsSsh = {
      name: x.name || `${x.host}-${x.zone}`.replace(/-$/, ''),
      host: x.host, port: x.port || 22, user: x.user, password: x.password,
    }
    if (x.zone.trim()) o.zone = x.zone.trim()
    // 企业编号(topent)不再写进环境配置 —— 该字段已整体移除,ENT 每次显式给。
    // 工作区：留空就是没配（不写这个键），与服务端“环境级没有兜底”一致。
    if (x.workspace.trim()) o.workspace = x.workspace.trim()
    // 环境自己的镜像目录：留空不写，后端回退到全局 mirror.dir/<环境名>。
    if (x.mirrorDir.trim()) o.mirrorDir = x.mirrorDir.trim()
    // 环境自己的库目录：留空不写，后端回退到全局 dict.dir（再回退数据目录/dict）。
    if (x.dictDir.trim()) o.dictDir = x.dictDir.trim()
    // ENT 清单原样带回：hosts 是整节提交的，这里漏了 → 改一次密码就把清单抹了。
    if (x.ents && x.ents.length) o.ents = x.ents.map((e) => ({ ent: e.ent, account: e.account, password: e.password }))
    if (x.entsSyncedAt) o.entsSyncedAt = x.entsSyncedAt
    if (x.db) {
      const db: HostsDb = { type: x.db.type || 'oracle', host: x.db.host.trim(), port: x.db.port || 0 }
      if (x.db.type === 'oracle') { if (x.db.service.trim()) db.service = x.db.service.trim() }
      else { if (x.db.database.trim()) db.database = x.db.database.trim() }
      const accounts = x.db.accounts.map((a) => ({ account: a.account.trim(), password: a.password })).filter((a) => a.account)
      if (accounts.length) db.accounts = accounts
      // 只有"显式关掉"才写:没配 = 默认开启,不必往 JSON 里塞一堆 true
      if (x.db.readonlySql === false) db.readonlySql = false
      o.db = db
    }
    return o
  })
  // 只有一个键了:activeEnv(默认环境)已整体移除,不再需要原样回传/顺延。
  return { sshs: sshsOut }
}

/**
 * debug 节 → 只带**这一页管得着**的键（目前只有停站看门狗）。
 *
 * 为什么现在敢只发一个键：后端 `putSection` 会把"本次没提交的键"从旧配置原样补回
 * （`debug.aiTakeover` / `query.limit` / `fglserver` / `persistBreakpoints` 都是这么保住的）。
 *
 * 老做法（把读到的整套值原样回传）有两个毛病，实测都踩到了：
 *   - **凭空写入**：配置里本来没有 `termWidth`，表单读到内置默认 200，保存一次就把它写死进
 *     config.json —— 用户从没设过这个值，却从此跟着他走（以后改内置默认值也带不动它）；
 *   - **漏一个键就丢一个字段**：`aiTakeover` 是后加的，名单没跟上，于是保存 DEBUG 卡片
 *     就把它删了（新建调试会话不再默认协作模式）。名单式回传天生跟不上 schema 的变化。
 * 现在把"保住没提交的键"这件事交给后端一处做，前端不再维护一份平行的字段名单。
 */
function debugPatchOf(d: DebugForm): NonNullable<HostsPatch['debug']> {
  const out: NonNullable<HostsPatch['debug']> = {}
  // 只有看门狗还在界面上（见「调试参数」卡片）
  if (d.watchdogSeconds) out.watchdogSeconds = d.watchdogSeconds
  return out
}

/**
 * 配置 → 环境表单。**只有这一处实现**：读设置时用它，保存成功后的回读也用它。
 *
 * 为什么保存后要回读：本地那份是"提交前的意图"，服务端返回的才是"存下来的事实"
 * （名字会被补成 host-zone、没填主机的条目根本不会存进去）。原来保存后只把名字 map 一遍，
 * 于是"界面列着、配置里没有"的幽灵行、以及"删掉最后一个环境后下标越界、整张表单与保存
 * 按钮一起消失"都能发生（实测，2026-10-15）。
 */
function sshsFormOf(c: HostsView): SshItem[] {
  // 设置页的 T100 表单只维护 T100；GP 环境由 GpSites 单独维护。
  // /api/hosts 是合并清单，这里不滤掉的话，T100 的“环境清单/同步/镜像”表格
  // 会混进 GP 行，选中态也会两套并存（实测截图就是这样）。
  return (c.sshs || []).flatMap((e) => {
    if (String((e as any).product || '').toLowerCase() === 'gp') return []
    const d = e.db
    return [{
      name: e.name || '', host: e.host || '', port: e.port || 22, user: e.user || '', password: e.password || '',
      zone: e.zone || '',
      workspace: e.workspace || '',
      mirrorDir: e.mirrorDir || '',
      dictDir: e.dictDir || '',
      // ENT 清单必须跟着表单走：hosts 是整节提交的，表单里没有它 = 保存一次就抹掉。
      ents: (e.ents || []).map((x) => ({ ent: x.ent, account: x.account, password: x.password })),
      entsSyncedAt: e.entsSyncedAt || '',
      db: d ? {
        type: d.type || 'oracle', host: d.host || '', port: d.port || 0,
        service: d.service || '', database: d.database || '',
        accounts: (d.accounts || []).map((a) => ({ account: a.account || '', password: a.password || '' })),
        readonlySql: d.readonlySql === false ? false : undefined,
        // 隧道配置必须跟着表单走:表单不编辑它,但「测试连接」要用它。
        // 漏了这一行,状态里就没有 viaSsh,测试照旧打本机直连(超时),
        // 而保存时后端 preserveUngovernedFields 又把文件里的那份留着 ——
        // 于是"文件里明明有隧道、界面上就是测不通",极难自查。
        viaSsh: d.viaSsh,
      } : null,
    }]
  })
}

// 没有 `onBack` / `backLabel` 了（用户 2026-10-16 要求去掉那个「返回「返回」按钮）：
// 设置页的出口只有左侧那一列视图图标。原来那个按钮的用意是"设置页不渲染右侧那四个依赖会话
// 的图标，得给一条回程路" —— 但设置是"先配环境"的地方，它不该猜你想回哪一件活儿。
export function SettingsView() {
  // TOP GP 的环境挂在本机的 ttgp 服务上；站点管理把它和 T100 的环境并成一张清单，
  // 每行标产品，点哪行就编辑哪边 —— 两边字段不同，表单各自一套。
  const { envs: gpEnvs } = useGpEnvs()
  const gpJobRunning = gpEnvs.some((g) => g.job?.state === 'running')
  // GP 的 job 进度在 /api/gp/meta 里，和 T100 一样按 800ms 轮询；否则表格里的
  // “进行中”和进度块会停在点击那一刻。
  useEffect(() => {
    if (!gpJobRunning) return
    const t = window.setInterval(() => { void refreshGpEnvs() }, 800)
    return () => window.clearInterval(t)
  }, [gpJobRunning])
  // 单选状态：T100 / GP 只可能有一个被选中。
  const [selection, setSelection] = useState<EnvSelection>({ product: 't100', index: 0 })
  const selectT100 = (index: number) => setSelection({ product: 't100', index })
  const selectGp = (name: string) => setSelection({ product: 'gp', name })
  const clearSelection = () => setSelection(null)
  // 左树点一条 GP 环境 → 进到这里：选中它（gpNew 让位）
  const settingsGpEnv = useStore((s) => s.settingsGpEnv)
  useEffect(() => {
    if (!settingsGpEnv) return
    setGpDraftId(null)
    selectGp(settingsGpEnv)
  }, [settingsGpEnv])
  // 未保存的 GP 草稿。复制时追加一条，而不是覆盖当前草稿。
  interface GpDraft { id: number; env: GpEnv }
  const [gpDrafts, setGpDrafts] = useState<GpDraft[]>([])
  const [gpDraftId, setGpDraftId] = useState<number | null>(null)
  const gpDraftSeq = useRef(1)
  const selectedDraft = gpDrafts.find((d) => d.id === gpDraftId)
  const addGpDraft = (env: GpEnv = emptyGpEnv()) => {
    const id = gpDraftSeq.current++
    setGpDrafts((ds) => [...ds, { id, env }])
    setGpDraftId(id)
    clearSelection()
  }
  const updateGpDraft = (id: number, env: GpEnv) => {
    setGpDrafts((ds) => ds.map((d) => d.id === id ? { ...d, env } : d))
  }
  const removeGpDraft = (id: number) => {
    const idx = gpDrafts.findIndex((d) => d.id === id)
    const next = gpDrafts.filter((d) => d.id !== id)
    setGpDrafts(next)
    if (gpDraftId === id) setGpDraftId(next.length ? next[Math.max(0, idx - 1)].id : null)
  }
  const copyGpDraft = (env: GpEnv) => {
    const taken = new Set(gpDrafts.map((d) => d.env.name))
    // GpSites 已经给复制体加了 "-copy"；这里只保证不重名。
    const base = (env.name || '新环境').trim() || '新环境'
    let name = base
    for (let n = 2; taken.has(name); n++) name = `${base}${n}`
    addGpDraft({ ...env, name })
  }
  const [gpMirrorDraft, setGpMirrorDraft] = useState<Record<string, string>>({})
  const section = useStore((s) => s.settingsSection)
  const setSection = useStore((s) => s.setSettingsSection)
  const theme = useStore((s) => s.theme)
  const setTheme = useStore((s) => s.setTheme)
  // 本页所有"动作结果"都从这里出去(见 toast.tsx 的判据)。原先它们是内联在各张卡片里的
  // 彩色段落 —— 出现/消失会顶动卡片高度,而这一页卡片多、表格多,串动尤其明显。
  const toast = useToast()

  const [envTab, setEnvTab] = useState<'ssh' | 'db' | 'ents'>('ssh')
  // 读配置失败的原因（非空 = 这一页要显示失败态而不是"加载配置中…"）。
  const [loadErr, setLoadErr] = useState('')
  const [cfg, setCfg] = useState<HostsView | null>(null)
  const [sshs, setSshs] = useState<SshItem[]>([])
  const [dbg, setDbg] = useState<DebugForm>({ watchdogSeconds: 0 })
  const [dict, setDict] = useState<DictForm>({ querySource: '', mirrorDir: '', dictDir: '' })
  const [listen, setListen] = useState('')
  // 派生状态(路径存不存在只有服务端算得出来)与配置元信息,以及 PATH 安装状态
  const [status, setStatus] = useState<ConfigStatus | null>(null)
  const [meta, setMeta] = useState<ConfigMeta | null>(null)
  const [installBusy, setInstallBusy] = useState(false)
  const [cacheBusy, setCacheBusy] = useState(false)
  // tt 自身的更新:状态靠 /api/update 读(不联网),查/装都是人去点才动作
  const [upd, setUpd] = useState<UpdateState | null>(null)
  const [updBusy, setUpdBusy] = useState('')
  // 源码镜像 / 字典同步的运行态:它们是**动作**(长跑任务 + 进度),不再是独立页面,
  // 就住在「数据字典」分区的对应卡片里。目标环境不再是这里的一份 state ——
  // 它由表格里被点的那一行给出(原先那份 state 只是"下拉里选中的那个",改成表格后没有了)。
  const [mirror, setMirror] = useState<MirrorResp | null>(null)
  const [sync, setSync] = useState<DBSyncResp | null>(null)
  const [opBusy, setOpBusy] = useState('')
  // 正在弹原生文件夹选择器的环境名；弹窗期间只禁用该行的文件夹按钮。
  // 目录选择器：前端自绘（后端 /api/fs/list 列目录）。原来这里只有一个 picking 字符串
  // 表示"正在弹系统对话框"，现在弹的是自己的弹窗，需要记 initial/title/回写动作。
  const [dirPick, setDirPick] = useState<{
    open: boolean; initial: string; title: string; onPick: (path: string) => void
  }>({ open: false, initial: '', title: '', onPick: () => {} })

  // 源码镜像 / 字典同步等长跑动作的结果,统一走 toast(见 toast.tsx 的判据):
  // 失败原来是一段内联在卡片里的红字 —— 它出现/消失会顶动卡片高度,
  // 而这段话往往很长(带原因与下一步),内联时还会把下面的表格挤下去。

  // 需要二次确认的动作(全量重建 / 开始同步 / 升级 tt)。
  // 这几个动作原先用 window.confirm:它是浏览器原生弹窗,样式与本界面无关、
  // 在 iframe/无头环境里还会被直接拦掉(拦掉时返回值按"取消"算,点了没反应还没提示)。
  // 统一改成与「删除环境」同一套 AlertDialog:文案里带上会被影响的环境与路径,
  // 确认按钮的文字按动作给(不是笼统的"确定")。
  const [ask, setAsk] = useState<{
    title: string; desc: string; confirmText: string; run: () => void
  } | null>(null)

  // 从唯一选中态派生旧代码使用的两个视角，避免大范围重写表单操作。
  const selSsh = selection?.product === 't100' ? selection.index : -1
  const gpSel = selection?.product === 'gp' ? selection.name : null
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved'>('idle')
  // 脏标记按**配置节**记。保存时只提交改过的节 —— 设置页的各张卡片共用 /api/hosts,
  // 把没改的节一起发过去就是用陈旧快照覆盖别张卡片刚改好的节。
  const dirtyRef = useRef<Record<ConfigKey, boolean>>({
    hosts: false, debug: false, listen: false, query: false, mirror: false, dict: false,
  })
  // 基线:上一次读到 / 保存成功时那一节的表单值(序列化后的样子)。
  //
  // 为什么要基线:只记"碰过没有"的话,改一下再改回原值仍然显示"有未保存的修改"、
  // 还常驻一个保存控件(实测)。脏 = **碰过 且 现在的值与基线不同**。
  // 判断放在渲染期算(而不是 onChange 里同步算):onChange 里此刻的 state 还是更新前那份,
  // 差一拍会让"改回原值"晚一步才认出来。
  const baselineRef = useRef<Partial<Record<ConfigKey, string>>>({})
  // 自动保存要在**最新一份表单**上算；setState 之后同一次事件里读到的还是旧值，
  // 所以每次渲染都把 ref 对齐。保存动作在 500ms 之后执行，读这里就是用户最后看到的那份。
  const formsRef = useRef({ sshs, dbg, dict, listen })
  formsRef.current = { sshs, dbg, dict, listen }
  const saveChainRef = useRef<Promise<void>>(Promise.resolve())
  const sectionSnapOf = (k: ConfigKey, f: { sshs: SshItem[]; dbg: DebugForm; dict: DictForm; listen: string }): string => JSON.stringify(
    k === 'hosts' ? hostsPatchOf(f.sshs)
      : k === 'debug' ? debugPatchOf(f.dbg)
        : k === 'listen' ? f.listen.trim()
          : k === 'query' ? { source: f.dict.querySource }
            : k === 'mirror' ? { dir: f.dict.mirrorDir.trim() }
                : { dir: f.dict.dictDir.trim() })
  const sectionSnap = (k: ConfigKey): string => sectionSnapOf(k, formsRef.current)
  const dirtyRefAny = () => (Object.keys(dirtyRef.current) as ConfigKey[])
    .some((k) => dirtyRef.current[k] && sectionSnap(k) !== baselineRef.current[k])
  const markDirty = (...keys: ConfigKey[]) => {
    keys.forEach((k) => { dirtyRef.current[k] = true })
  }
  // 基线对齐**要等 state 落地之后**：在 setX 之后立刻算，读到的还是更新前那一份（差一拍），
  // 结果是"刚读到配置就显示有未保存的修改"。所以这里只记下"要对齐哪几节"并递增一个计数，
  // 由下面那条 effect 在提交之后把基线对齐到当下。
  const resyncKeysRef = useRef<ConfigKey[] | null>(null)
  const [syncTick, setSyncTick] = useState(0)
  const cleanDirty = (keys?: ConfigKey[]) => {
    resyncKeysRef.current = keys ?? null
    setSyncTick((t) => t + 1)
  }
  useEffect(() => {
    const ks = resyncKeysRef.current ?? (Object.keys(dirtyRef.current) as ConfigKey[])
    ks.forEach((k) => { dirtyRef.current[k] = false; baselineRef.current[k] = sectionSnap(k) })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [syncTick])
  const isDirty = (...keys: ConfigKey[]) =>
    keys.some((k) => dirtyRef.current[k] && sectionSnap(k) !== baselineRef.current[k])

  const [showPwd, setShowPwd] = useState(false)
  // 「ENT」页签的同步状态（连库读 gzou_t 并写回配置；唯一会改动清单的入口）
  const [entsBusy, setEntsBusy] = useState(false)
  // 数据库页操作状态。accProbe 只记"哪一行、什么状态"用于行内图标;
  // 具体文案走 toast(见 verifyAcct)。
  const [busy, setBusy] = useState<string>('')
  const [accProbe, setAccProbe] = useState<{ i: number; state: 'testing' | 'ok' | 'err' } | null>(null)
  const [addAcct, setAddAcct] = useState({ account: '', password: '' })
  const [delTarget, setDelTarget] = useState<number | null>(null) // 待删除环境的下标(弹窗确认)

  const sshName = (e: SshItem) => e.name || `${e.host}-${e.zone}`.replace(/-$/, '')

  const loadSettings = useCallback(() => {
    api.settings().then((c) => {
      setCfg(c)
      setSshs(sshsFormOf(c))
      // 只读这一页真的有的那个字段。其余 debug 键（aiTakeover / fglserver / persistBreakpoints…）
      // 界面没有控件，也就**不必读进来再回传** —— 服务端 putSection 会从旧配置原样补回。
      setDbg({ watchdogSeconds: c.debug?.watchdogSeconds || 0 })
      setDict({
        querySource: c.query?.source === 'auto' ? '' : (c.query?.source || ''),
        mirrorDir: c.mirror?.dir || '',
        dictDir: c.dict?.dir || '',
      })
      setListen(c.listen || '')
      setLoadErr('')
      cleanDirty()
    }).catch((e) => {
      // 读不到配置要**留在界面上**（loadErr），不能只发一条会自己消失的 toast：
      // 那会让人永远停在"加载配置中…"，看着像还在读，其实早就失败了（实测）。
      setLoadErr(e?.message || String(e))
      toast.error('读取配置失败', { description: e?.message || String(e) })
    })
    // 派生状态与元信息:取不到不影响主流程(它们在卡片里只是提示)
    void api.configStatus().then(setStatus).catch(() => { /* 静默:提示性信息 */ })
    void api.configMeta().then(setMeta).catch(() => { /* 静默:提示性信息 */ })
    // 更新状态同样是不联网的读端点(检查结论来自缓存)
    void api.updateState().then(setUpd).catch(() => { /* 静默:提示性信息 */ })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => { loadSettings() }, [loadSettings])
  const anyDirty = dirtyRefAny()
  const view = useStore((s) => s.view)

  // 本页被 keep-alive 常驻挂载(切走只是 display:none),所以**不能轮询** ——
  // 改为"每次切回本页、且没有未保存修改时"回读一次,拿别处(命令行)改过的值。
  // 初值取"挂载那一刻可不可见":本页是 store 的初始视图,若从 false 起步,第一条 effect
  // 已经拉过一次 /api/hosts,这一条又会把同一次挂载当成"刚从别处切回来"再拉一次
  // （CDP 抓到两次同源调用，实测）。判据是"可见性**变化**"，不是"现在可见"。
  const visibleRef = useRef(view === 'settings')
  useEffect(() => {
    const visible = view === 'settings'
    if (visible && !visibleRef.current && !dirtyRefAny()) void loadSettings()
    visibleRef.current = visible
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [view])

  // 分区变化 → 同步到 URL 片段(用 replace 而不是 push:否则浏览器后退会一格一格走遍所有分区)
  //
  // 首跑要**让一次**:本页是 store 的初始视图,于是它比 App 那条深链接解析**先挂载**,
  // 而 React 的子 effect 先于父 effect 跑 —— 无条件写片段会把 `#wslogs`/
  // `#settings/data-dict` 覆盖成 `#settings/sites`,深链接等于全废(实测三个都落回站点管理)。
  // 所以片段已经指着某个视图(含设置页的分区)时,这次一个字都不写;往后换分区照旧同步。
  const hashSyncedRef = useRef(false)
  useEffect(() => {
    const want = settingsHash(section)
    if (!hashSyncedRef.current) {
      hashSyncedRef.current = true
      if (parseRoute(window.location.hash).kind !== 'home') return
    }
    if (window.location.hash !== want) window.history.replaceState(null, '', want)
  }, [section])

  // ---- 自动保存：按节提交；keys 指明这次要提交哪些节 ----
  //
  // 提交环境清单前先拦一道**只有前端才看得见**的事：`hostsPatchOf` 是整节提交，它会
  // `filter(x => x.host)` 掉没有服务器地址的条目 —— 于是界面照旧列着它、还显示"已保存 ✓"，
  // 而配置里根本没有它（用户以为存上了）。服务端见不到这类条目（上一步就被滤掉了），
  // 所以后端那些"缺少服务器地址"的校验永远不会响。返回 null = 可以保存。
  const hostsSaveBlocker = (list: SshItem[]): string | null => {
    for (let i = 0; i < list.length; i++) {
      if (!list[i].host.trim()) {
        const who = list[i].name.trim() || sshName(list[i]) || '还没填名字'
        return `第 ${i + 1} 个环境（${who}）没有服务器地址：没有地址的环境不会被保存，` +
          `界面会照旧列着它 —— 那是个假象。填上地址，或先把它删掉。`
      }
    }
    return null
  }

  const save = async (requested: ConfigKey[], list?: SshItem[]): Promise<void> => {
    if (!cfg) return
    const forms = formsRef.current
    // 排队期间可能已经被前一次保存清掉；只提交仍然有差异的节，避免重复 PUT。
    const keys = requested.filter((k) => dirtyRef.current[k] && sectionSnapOf(k, forms) !== baselineRef.current[k])
    if (keys.length === 0) return
    const hostsList = list || forms.sshs
    const sent: Partial<Record<ConfigKey, string>> = {}
    keys.forEach((k) => { sent[k] = sectionSnapOf(k, forms) })
    if (keys.includes('hosts')) sent.hosts = JSON.stringify(hostsPatchOf(hostsList))
    const body: HostsPatch = {}
    if (keys.includes('hosts')) {
      const blocker = hostsSaveBlocker(hostsList)
      if (blocker) {
        toast.error('保存失败', { description: blocker, duration: 0 })
        return
      }
      body.hosts = hostsPatchOf(hostsList)
    }
    if (keys.includes('debug')) body.debug = debugPatchOf(forms.dbg)
    if (keys.includes('listen')) body.listen = forms.listen.trim()
    if (keys.includes('query')) body.query = { source: forms.dict.querySource }
    if (keys.includes('mirror')) body.mirror = { dir: forms.dict.mirrorDir.trim() }
    if (keys.includes('dict')) body.dict = { dir: forms.dict.dictDir.trim() }

    try {
      const next = await api.saveSettings(body)
      setCfg(next)
      if (keys.includes('hosts')) {
        const serverList = sshsFormOf(next)
        const sentHosts = sent.hosts || ''
        const nowHosts = JSON.stringify(hostsPatchOf(formsRef.current.sshs))
        if (nowHosts === sentHosts) {
          const keepName = formsRef.current.sshs[selSsh] ? sshName(formsRef.current.sshs[selSsh]) : ''
          const same = serverList.findIndex((x) => sshName(x) === keepName)
          setSshs(serverList)
          formsRef.current = { ...formsRef.current, sshs: serverList }
          selectT100(same >= 0 ? same : Math.min(selSsh, Math.max(0, serverList.length - 1)))
          baselineRef.current.hosts = JSON.stringify(hostsPatchOf(serverList))
          dirtyRef.current.hosts = false
        } else {
          // 保存期间用户又改了：服务端那份不能回灌，保留本地并等下一次自动保存。
          baselineRef.current.hosts = sentHosts
          dirtyRef.current.hosts = true
        }
      }
      keys.filter((k) => k !== 'hosts').forEach((k) => {
        baselineRef.current[k] = sent[k]
        dirtyRef.current[k] = sectionSnapOf(k, formsRef.current) !== sent[k]
      })

      // 镜像目录保存后立刻回读：表格里的实际路径/基线随之更新，
      // 不要让用户点“增量更新”时还用着保存前的旧路径。
      if (keys.includes('hosts') || keys.includes('mirror')) {
        void api.mirror().then(setMirror).catch(() => { /* 提示性信息，失败不打断保存 */ })
      }
      void api.configStatus().then(setStatus).catch(() => {})
    } catch (ex: any) {

      toast.error('保存失败', { description: ex?.message || String(ex), duration: 0 })
    }
  }

  // 所有自动保存串行：同一次输入风暴只留最后一次，且不会两个 PUT 互相覆盖。
  const enqueueSave = (keys: ConfigKey[], list?: SshItem[]): Promise<void> => {
    const run = saveChainRef.current.then(() => save(keys, list))
    saveChainRef.current = run.catch(() => {})
    return run
  }

  const dirtyKeys = (): ConfigKey[] =>
    (Object.keys(dirtyRef.current) as ConfigKey[])
      .filter((k) => dirtyRef.current[k] && sectionSnap(k) !== baselineRef.current[k])

  // 自动保存：任何表单变化后 500ms 合并提交；保存期间的新改动排进同一条链。
  useEffect(() => {
    if (!cfg) return
    // 新加的空环境没有 host：先不自动保存（也没有必要每 500ms 弹一次“没有服务器地址”），
    // 等用户填上地址再走；显式的 ENT 同步/删除仍会经过 save 的校验并给出提示。
    const keys = dirtyKeys().filter((k) => !(k === 'hosts' && sshs.some((x) => !x.host.trim())))
    if (keys.length === 0) return
    const t = window.setTimeout(() => { void enqueueSave(keys) }, 500)
    return () => window.clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sshs, dbg, dict, listen, cfg])

  // ENT 同步会重读配置：先把当前（可能是新建、还没保存的）环境落盘。
  const flushHosts = async (): Promise<boolean> => {
    await saveChainRef.current
    if (dirtyRef.current.hosts && sectionSnap('hosts') !== baselineRef.current.hosts) {
      await enqueueSave(['hosts'])
      await saveChainRef.current
    }
    return !dirtyRef.current.hosts || sectionSnap('hosts') === baselineRef.current.hosts
  }

  // ---- 环境编辑 ----
  const patchSsh = (i: number, patch: Partial<SshItem>) => {
    const next = sshs.map((x, j) => (j === i ? { ...x, ...patch } : x))
    setSshs(next); markDirty('hosts')
  }
  const patchDb = (i: number, patch: Partial<SshDb>) => {
    const cur = sshs[i]
    if (!cur) return
    patchSsh(i, { db: cur.db ? { ...cur.db, ...patch } : { ...blankDb(), ...patch } })
  }
  const patchAcct = (i: number, j: number, patch: Partial<{ account: string; password: string }>) => {
    const cur = sshs[i]
    if (!cur?.db) return
    patchDb(i, { accounts: cur.db.accounts.map((a, k) => (k === j ? { ...a, ...patch } : a)) })
  }
  const delAcct = (i: number, j: number) => {
    const cur = sshs[i]
    if (!cur?.db) return
    patchDb(i, { accounts: cur.db.accounts.filter((_, k) => k !== j) })
  }
  const pushAcct = () => {
    const cur = sshs[selSsh]
    if (!cur?.db) return
    const account = addAcct.account.trim()
    // 账号与密码都必须填。**不再**"留空则口令同账号" —— 那会把"漏填密码"和
    // "口令恰好与账号相同"变成同一个动作,后者是极少数场景,而前者是漏填。
    // 按钮那边已经 disabled,这里是第二道(回车键也走这条路)。
    const password = addAcct.password.trim()
    if (!account || !password) return
    patchDb(selSsh, { accounts: [...cur.db.accounts, { account, password }] })
    setAddAcct({ account: '', password: '' })
  }
  // syncEnts 是「ENT」页签唯一的写入口：连库读 gzou_t → 写回 config.json。
  // 不允许手填：清单是“从库里抄下来的”，不是人凭记忆写的。
  const syncEnts = async () => {
    const s0 = sshs[selSsh]
    if (!s0?.host) { toast.error('先填 SSH 主机'); return }
    if (!(await flushHosts())) {
      toast.error('环境还没保存成功', { description: '同步会重读配置；先把上面的错误处理掉再试' })
      return
    }
    // flush 可能把服务器补全后的环境名写回 formsRef，用最新那份点名。
    const saved = formsRef.current.sshs[selSsh] || s0
    const name = sshName(saved)
    setEntsBusy(true)
    try {
      const r = await api.entsSync(name)
      await new Promise<void>((resolve) => { loadSettings(); setTimeout(resolve, 300) })
      const extra = r.addedAccounts?.length
        ? `；新增账号 ${r.addedAccounts.join('、')}（密码按惯例=账号名，请到「数据库」页签改成真实口令）`
        : ''
      toast.success(`已同步 ${r.count} 个企业${extra}`)
    } catch (e: any) {
      toast.error('同步 ENT 失败', { description: e?.message || String(e), duration: 0 })
    } finally {
      setEntsBusy(false)
    }
  }

  const addSsh = () => { selectT100(sshs.length); setSshs([...sshs, blankSsh()]); markDirty('hosts') }

  /**
   * 发现环境:扫描本机的 T100 设计器站点清单,让用户勾选导入。
   *
   * 为什么走这条路:TT 的环境配置在设计上就是**模仿设计器的站台管理**做的 ——
   * 主机/端口/账号/口令/登录区域/工作区这些字段两边同义,而初次装 TT 的机器上
   * 大概率已经装了设计器且存过站点。所以与其让用户对着设计器逐字段手抄,不如直接读。
   *
   * 只读:解的是设计器那份加密文件,我们**不写回**它(见 internal/designer 的包注释)。
   */
  const [disc, setDisc] = useState<{
    open: boolean
    loading: boolean
    reason: string
    sitesFile: string
    envs: DiscoveredEnv[]
    picked: Record<string, boolean>
  }>({ open: false, loading: false, reason: '', sitesFile: '', envs: [], picked: {} })

  // 候选行的稳定键:uid 优先(设计器保证唯一),没有就退回名字+下标 —— 同名站点也可能存在。
  const discKey = (e: DiscoveredEnv, i: number) => e.uid || `${e.name}#${i}`

  const openDiscover = () => {
    setDisc({ open: true, loading: true, reason: '', sitesFile: '', envs: [], picked: {} })
    api.discoverEnvs().then((r) => {
      // 默认勾选规则:**已存在的名字不勾**,其余可用的一条都勾上。
      // 同名仍然列出来并可手动勾(用户要的可能是"导入一份副本"),只是不替用户决定覆盖。
      const existing = new Set(sshs.map((x) => sshName(x)))
      const picked: Record<string, boolean> = {}
      ;(r.envs || []).forEach((e, i) => {
        picked[discKey(e, i)] = !e.skipped && !existing.has(e.name)
      })
      setDisc({ open: true, loading: false, reason: r.reason || '', sitesFile: r.sitesFile || '', envs: r.envs || [], picked })
    }).catch((e: any) => {
      setDisc((d) => ({ ...d, loading: false, reason: e?.message || String(e) }))
    })
  }

  const confirmDiscover = () => {
    const add: SshItem[] = []
    disc.envs.forEach((e, i) => {
      if (e.skipped || !disc.picked[discKey(e, i)]) return
      // 复用设置页的表单形状(SshItem),而不是直接塞 config.NamedSsh ——
      // 这样导入进来的环境和手填的走**同一条**保存路径(hostsPatchOf),不会有两套转换。
      add.push({
        ...blankSsh(),
        name: e.name,
        host: e.host,
        port: e.port || 22,
        user: e.user,
        password: e.password,
        zone: e.zone || '',
        workspace: e.workspace || '',
        // db 保持 blankSsh 的默认(空库):设计器的站点清单里没有数据库配置,
        // 用户导入后要在「数据库」页签自己补 —— 弹窗里也写明了这一点。
      })
    })
    setDisc((d) => ({ ...d, open: false }))
    if (add.length === 0) return
    selectT100(sshs.length) // 选中第一条新导入的,让用户直接接着填数据库
    setSshs([...sshs, ...add])
    markDirty('hosts')
  }

  /**
   * 复制当前环境。
   *
   * 用途是"同一套库、换一台服务器":SSH 四元组与库连接整份抄过来,改个地址就能用,
   * 比从空白重敲一遍省事。所以这里抄的是**当前正在编辑的那份 state**(含还没保存的改动),
   * 不是 config.json 里的旧值 —— 否则"改完→复制"会把刚改的丢掉。
   *
   * 名称必须当场定下来:后端按名字认环境、重名直接 400。而截断显示名是按
   * `主机-区域` 兜底算的,空名字的两条会撞成同一个键 —— 所以这里始终写一个显式名字
   * (原名/显示名 + -copy,撞了就再往后取 -copy2、-copy3)。
   *
   * 刻意**不**顺手做别的动作:复制多半是为了试一套新参数,顺手把线上某个环境改掉
   * 的代价比多点一下大得多。两个动作分开。
   */
  const dupSsh = (i: number) => {
    const src = sshs[i]
    if (!src) return
    const base = (src.name || sshName(src) || '环境') + '-copy'
    const taken = new Set(sshs.map((x) => sshName(x)))
    let name = base
    for (let n = 2; taken.has(name); n++) name = `${base}${n}`
    // 深拷贝:db 与 accounts 都是嵌套对象,浅拷贝会让两个环境共用同一个账号数组,
    // 改一处两处都变(而且 dirty 标记只认 hosts 节,这种串改极难排查)
    const item: SshItem = {
      ...src, name,
      db: src.db ? { ...src.db, accounts: src.db.accounts.map((a) => ({ ...a })) } : null,
    }
    setSshs([...sshs, item])
    selectT100(sshs.length)
    markDirty('hosts')
  }
  // setDefaultEnv 已删除:"设为默认环境"这个动作本身没有了
  // (默认环境已移除,环境一律由 --env / 会话面板点名)。
  // 删除环境:先弹窗二次确认(误删代价高且不可撤销),确定后立即落盘保存
  const askDelSsh = () => { if (sshs[selSsh]) setDelTarget(selSsh) }
  const confirmDelSsh = async () => {
    const i = delTarget
    setDelTarget(null)
    if (i === null || !sshs[i]) return
    const next = sshs.filter((_, j) => j !== i)
    selectT100(Math.max(0, Math.min(i, next.length - 1)))
    setSshs(next)
    formsRef.current = { ...formsRef.current, sshs: next }
    markDirty('hosts')
    await enqueueSave(['hosts'], next) // 确认删除 = 立即落盘（自动保存不会等 500ms）
  }

  // ---- 数据库页操作(针对当前环境的 db) ----
  // 从服务器获取:登录当前环境的 SSH 自动探测连接要素回填(辅助)
  const fetchFromServer = async () => {
    const s = sshs[selSsh]
    const d = s?.db
    if (!s?.host || !s?.user) { toast.error('请先填写 SSH 主机与账号'); return }
    if (!d) return
    setBusy('fetch')
    try {
      const r = await api.probeDB({ host: s.host, port: s.port || 22, user: s.user, password: s.password, zone: s.zone, type: d.type || 'oracle' })
      const patch: Partial<SshDb> = {}
      // 主机地址:服务器解析出的是 IP 就直接用;是服务器内部主机名(客户端多半不可达)则回填 SSH 主机
      // —— T100 的库通常与应用服务器同机,SSH 能连上就意味着该地址客户端可达。
      const probedHost = (r.host || '').trim()
      const isIP = /^\d{1,3}(\.\d{1,3}){3}$/.test(probedHost)
      const hostVal = (isIP ? probedHost : '') || (s.host || '').trim()
      if (hostVal) patch.host = hostVal
      if (r.type === 'kingbase') {
        if (r.database) patch.database = r.database
        if (r.port) patch.port = r.port
        // 结果的三种语气分开报:note 是"获取到了但有话要说"(信息),没有 note 才是干净成功。
        if (r.note) toast.info('获取完成,但有一处要确认', { description: r.note })
        else toast.success(`已获取金仓库信息:${r.database || '?'} @ ${hostVal || '?'}:${r.port || '?'}`, {
          description: '主机地址按 SSH 主机回填;如需从别处连库请手工调整。',
        })
      } else {
        if (r.service || r.tns) patch.service = r.service || r.tns || ''
        if (r.port) patch.port = r.port
        const inner = probedHost && !isIP ? `(服务器内部名 ${probedHost} 客户端多半不可达,已改用 SSH 主机)` : ''
        const detail = `服务器解析地址 ${probedHost || '未知'}:${r.port || '?'}(service ${r.service || r.tns || '?'});主机地址已回填 ${hostVal || '?'}${inner}`
        if (r.note) toast.error('获取到的信息需要确认', { description: r.note + ';' + detail })
        else toast.success('已从服务器获取库连接要素', { description: detail })
      }
      if (Object.keys(patch).length) patchDb(selSsh, patch)
    } catch (ex: any) {
      toast.error('从服务器获取失败', { description: ex?.message || String(ex) })
    } finally { setBusy('') }
  }
  // 附加账号行验证:服务器侧以 账号+密码 连当前环境的显式目标库
  const verifyAcct = async (j: number) => {
    const i = selSsh
    const s = sshs[i]
    const d = s?.db
    const row = d?.accounts?.[j]
    if (!s?.host || !s?.user) { toast.error('请先填写 SSH 主机与账号'); return }
    if (!d?.host.trim()) { toast.error('请先填写数据库主机地址'); return }
    if (!row?.account.trim()) { toast.error('账号不能为空'); return }
    setAccProbe({ i: j, state: 'testing' })
    try {
      await api.dbAccVerify({
        host: s.host, port: s.port || 22, user: s.user, password: s.password, zone: s.zone,
        type: d.type || 'oracle', account: row.account.trim(), acctPassword: row.password,
        dbHost: d.host.trim(), dbPort: d.port || 0,
        ...(d.type === 'kingbase' ? { dbDatabase: d.database.trim() } : { dbSvc: d.service.trim() }),
      })
      // 行内只留状态(图标/禁用),文案走 toast —— 那行字原来内联在账号表格下面,
      // 出现与消失都会顶动下面所有内容。
      setAccProbe({ i: j, state: 'ok' })
      toast.success(`账号 ${row.account.trim()} 连接正常`)
    } catch (ex: any) {
      setAccProbe({ i: j, state: 'err' })
      toast.error(`账号 ${row.account.trim()} 验证失败`, { description: ex?.message || String(ex) })
    }
  }
  // 客户端直连测试:验证「本机 → 库」这条链路真的通(与服务器侧验证互补)。
  // viaSsh 不在表单里(手工写在 config.json),但**必须带上**:真实查询走的就是这条隧道,
  // 不带就等于测另一条根本不存在的直连线路(表现为一直超时,而查询其实是好的)。
  const testConn = async () => {
    const d = sshs[selSsh]?.db
    if (!d?.host.trim()) { toast.error('请先填写数据库主机地址'); return }
    setBusy('conn')
    try {
      const r = await api.connTest({
        type: d.type || 'oracle', host: d.host.trim(), port: d.port || 0,
        service: d.service.trim(), database: d.database.trim(),
        accounts: d.accounts.map((a) => ({ account: a.account.trim(), password: a.password })).filter((a) => a.account),
        viaSsh: d.viaSsh,
      })
      const ver = (r.serverVersion || '').split('\n')[0]
      const how = r.via === 'tunnel' ? '(经 SSH 隧道)' : ''
      if (r.ok) toast.success(`连接正常${how}:${ver}`)
      else toast.error(r.stage === 'version' ? '取版本失败' : '连接失败', { description: r.error })
    } catch (ex: any) {
      toast.error('连接失败', { description: ex?.message || String(ex) })
    } finally { setBusy('') }
  }

  // ---- 命令行集成(用户 PATH) ----
  const doInstall = async (add: boolean) => {
    setInstallBusy(true)
    try {
      const st = add ? await api.installAdd() : await api.installRemove()
      setStatus((prev) => (prev ? { ...prev, install: st } : prev))
      // 成功要提醒的是"下一步动作"(新开终端才生效)——一句话,放 toast 里正合适。
      toast.success(add
        ? `已把 ${st.exeDir} 加入用户 PATH`
        : `已从用户 PATH 移除 ${st.exeDir}`, {
        description: add ? '新开的终端即可直接敲 tt。' : undefined,
      })
    } catch (ex: any) {
      toast.error('修改用户 PATH 失败', { description: ex?.message || String(ex) })
    } finally { setInstallBusy(false) }
  }

  // ---- tt 自身的更新 ----
  // 查:人去点才联网(服务不轮询——见 internal/web/update.go 的三条纪律)。
  const doUpdateCheck = async () => {
    setUpdBusy('check')
    try {
      const r = await api.updateCheck()
      if (!r.ok) {
        toast.error(r.error || '检查未成', { description: r.where })
        return
      }
      setUpd((prev) => (prev ? { ...prev, check: r.check, hint: r.hint } : prev))
      // 有新版是"信息"(用户可能不急),已是最新是"成功" —— 语气不同,但都不该顶动卡片。
      if (r.check?.newer) toast.info(r.hint || `有新版本 ${r.check?.latest}`)
      else toast.success(`已是最新(${r.check?.latest})`)
    } catch (ex: any) {
      toast.error('检查更新失败', { description: ex?.message || String(ex) })
    } finally { setUpdBusy('') }
  }

  // 装:服务端拉一个 `tt update --yes` 子进程去做 —— 本服务几分钟后会被它停掉替换,
  // 所以这里只能告诉用户“已接手”,不能等结果。
  // 这条确认尤其不能省:它会把当前服务停掉,页面随后断开。
  const doUpdateApply = () => {
    setAsk({
      title: '要现在升级 tt 吗?',
      desc: '下载并校验完成后会停掉本服务并替换程序文件,\n本页随后会断开,请稍后重新打开。',
      confirmText: '开始升级',
      run: () => void applyUpdate(),
    })
  }
  const applyUpdate = async () => {
    setUpdBusy('apply')
    try {
      const r = await api.updateApply()
      // 用 duration:0 —— 这条之后本服务会被更新器停掉、页面随即断开,
      // 信息必须留在屏幕上让用户看到(默认 8 秒可能还没读完页面就断了)。
      toast.success(r.note || `更新器已接手(pid ${r.pid})`, { duration: 0 })
    } catch (ex: any) {
      toast.error('交棒给更新器失败', { description: ex?.message || String(ex) })
    } finally { setUpdBusy('') }
  }

  // ---- 缓存清理 ----
  // 只删可再生的中间数据(后端只认 internal/config 那份清单);config.json 不碰。
  // 后端把清完的状态一起回,所以这里不必再打一次 configStatus。
  const doClearCache = async () => {
    setCacheBusy(true)
    try {
      const r = await api.cacheClear()
      setStatus((prev) => (prev ? { ...prev, cache: r.cache } : prev))
      toast.success(r.removed > 0
        ? `已清理 ${r.removed} 个文件,释放 ${fmtBytes(r.freed)}`
        : '没有可清理的文件')
    } catch (ex: any) {
      toast.error('清理缓存失败', { description: ex?.message || String(ex) })
    } finally { setCacheBusy(false) }
  }



  // ---- 源码镜像 / 字典同步(动作) ----
  // 只在「数据字典」分区且正在跑的时候轮询:本页是常驻挂载的,无脑轮询会一直打后端。
  // 环境不再需要在这里挑一个"当前项"了 —— 动作由表格里那一行给出(见 pullMirror/runSync 的参数)。
  const refreshMirror = useCallback(async () => {
    try { setMirror(await api.mirror()) } catch { /* 取不到就不显示操作区,不打扰 */ }
  }, [])
  const refreshSync = useCallback(async () => {
    try { setSync(await api.dbsync()) } catch { /* 同上 */ }
  }, [])
  useEffect(() => {
    if (section !== 'data-dict') return
    void refreshMirror(); void refreshSync()
  }, [section, refreshMirror, refreshSync])
  const mirrorRunning = !!mirror?.job?.running
  const syncRunning = !!sync?.job?.running
  useEffect(() => {
    if (!mirrorRunning) return
    const t = window.setInterval(() => { void refreshMirror() }, 800)
    return () => window.clearInterval(t)
  }, [mirrorRunning, refreshMirror])
  useEffect(() => {
    if (!syncRunning) return
    const t = window.setInterval(() => { void refreshSync() }, 800)
    return () => window.clearInterval(t)
  }, [syncRunning, refreshSync])

  // 任务结束时再取一次：轮询随 running=false 停下，不补这一下的话
  // 「本地库」列会一直停在“未同步”，直到切分区或刷新页面。
  useEffect(() => {
    if (sync?.job?.done && !sync.job.running) void refreshSync()
  }, [sync?.job?.done, sync?.job?.running, refreshSync])

  // 环境名由调用的那一行给出 —— 表格里每行一个按钮,不再有"先选环境"这一步。
  // 全量重建会整目录替换,先弹确认;确认后走 doMirror(动作与确认分开,
  // 弹窗那一侧只负责 setAsk 与描述文案)。
  const pullMirror = (full: boolean, env: string) => {
    if (!env) return
    if (!full) { void doMirror(false, env); return }
    setAsk({
      title: `全量重建环境「${env}」?`,
      desc: '将整目录替换本地镜像(删除服务器上已不存在的残留)。\n首次或按需执行,耗时较长。',
      confirmText: '全量重建',
      run: () => void doMirror(true, env),
    })
  }
  // 目录选择器（前端自绘，后端 /api/fs/list 列目录）；选中后写回该环境的 mirrorDir。
  const pickMirrorDir = (env: string, index: number, initial: string) => {
    if (index < 0) return
    setDirPick({
      open: true,
      initial,
      title: `选择环境「${env}」的镜像目录`,
      onPick: (path) => patchSsh(index, { mirrorDir: path }),
    })
  }

  const doMirror = async (full: boolean, env: string) => {
    setOpBusy('mirror')
    try {
      await api.mirrorPull(env, full)
      await refreshMirror()
      // 刻意不给"已开始…"这种成功提示:进度块马上就会出现,并写着环境、
      // 全量/增量与进行中状态 —— 同一件事在一屏里说两遍就是噪音。
      // (失败才留提示:失败原因只有它能说清,见下面的 catch。)
    } catch (ex: any) {
      toast.error('拉取源码镜像失败', { description: ex?.message || String(ex) })
    } finally { setOpBusy('') }
  }
  /**
   * 停止正在跑的拉取。不弹二次确认:这个动作本身就是"撤销误点",再让它过一道确认
   * 等于把最需要即时性的操作拖慢一步。后端的取消是**异步**的(SFTP 读到下一个分片才返回),
   * 所以这里不 await 到"真的停了"—— 立刻回读一次让轮询接管,按钮随即变成「正在停止…」。
   */
  const stopMirror = async () => {
    try {
      await api.mirrorStop()
      await refreshMirror()
    } catch (ex: any) {
      toast.error('停止失败', { description: ex?.message || String(ex) })
    }
  }
  const runSync = (env: string) => {
    if (!env) return
    setAsk({
      title: `同步环境「${env}」的字典数据?`,
      desc: `目标:${(sync?.envs.find((e) => e.name === env)?.dbPath || '')}\n将覆盖本地 SQLite(原库自动备份为 .bak),约 85 万行,需数分钟。`,
      confirmText: '开始同步',
      run: () => void doSync(env),
    })
  }
  const doSync = async (env: string) => {
    setOpBusy('sync')
    try {
      await api.dbsyncRun(env)
      await refreshSync()
      // 成功不给提示,理由同 doMirror(进度块随后就把这件事说清了)
    } catch (ex: any) {
      toast.error('同步字典数据失败', { description: ex?.message || String(ex) })
    } finally { setOpBusy('') }
  }
  /** 停止正在跑的同步(理由同 stopMirror:不二次确认,异步取消不等待)。 */
  const stopSync = async () => {
    try {
      await api.dbsyncStop()
      await refreshSync()
    } catch (ex: any) {
      toast.error('停止失败', { description: ex?.message || String(ex) })
    }
  }

  // 还没读到配置：失败就给失败态 + 重试，别让人永远停在"加载配置中…"
  // （实测：挡住 /api/hosts 后这一页永久转圈，恢复网络也不会自己回来 —— 只有切走再回来才重新拉）。
  if (!cfg) {
    return (
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 p-6 text-center text-xs">
        {loadErr ? (
          <>
            <div className="text-sm text-red-600 dark:text-red-400">读取配置失败</div>
            <div className="max-w-2xl whitespace-pre-wrap text-muted-foreground">{loadErr}</div>
            <button onClick={() => loadSettings()}
              className="rounded-ui border border-border px-2 py-1 text-muted-foreground hover:bg-accent hover:text-foreground">
              重试
            </button>
          </>
        ) : (
          <div className="text-muted-foreground">加载配置中…</div>
        )}
      </div>
    )
  }
  const cur = sshs[selSsh]
  const curDb = cur?.db
  // 设计器清单里有、TT 里也有的名字 —— 用于在「发现环境」弹窗里标注"已存在"。
  const existingNames = new Set(sshs.map((x) => sshName(x)))
  // isCurDefault 已删除:没有"默认环境"可判了(见上面 setDefaultEnv 的说明)。
  // 表格里那些行级动作的共用禁用条件:别的操作在跑、或这个动作自身在跑
  const mirrorBusy = mirrorRunning || opBusy !== ''
  const syncBusy = syncRunning || opBusy !== ''
  // 渲染期现算的 anyDirty 已在上面算过（它同时给"刷新确认"那条 effect 用），这里不再重复
  // 每个环境的按钮可用性：目录来自全局根或该环境自己的覆盖，且相关设置在自动保存后再点。
  const mirrorRowReady = (name: string) => {
    const ssh = sshs.find((x) => sshName(x) === name)
    return !!(ssh?.mirrorDir.trim() || dict.mirrorDir.trim()) && !isDirty('mirror') && !isDirty('hosts')
  }

  // 一个清单装两种环境：T100 行来自 /api/dbsync、/api/mirror，GP 行来自 gpEnvs。
  // T100 的响应里也带着 GP 环境（后端按 hosts 全量遍历），这里按名字排掉，
  // 否则 GP 会出现两次，其中一次的按钮还是 T100 的同步/镜像逻辑。
  const gpNameSet = new Set(gpEnvs.map((g) => g.name))
  const syncRows: SyncRow[] = [
    ...(sync?.envs ?? [])
      .filter((e) => !gpNameSet.has(e.name))
      .map((e) => ({ product: 't100' as const, t100: e })),
    ...gpEnvs.map((g) => ({ product: 'gp' as const, gp: g })),
  ].sort((a, b) => rowName(a).localeCompare(rowName(b), 'zh-CN'))

  const mirrorRows: MirrorRow[] = [
    ...(mirror?.envs ?? [])
      .filter((e) => !gpNameSet.has(e.name))
      .map((e) => ({ product: 't100' as const, t100: e })),
    ...gpEnvs.map((g) => ({ product: 'gp' as const, gp: g })),
  ].sort((a, b) => rowName(a).localeCompare(rowName(b), 'zh-CN'))

  // T100 的 /api/mirror、/api/dbsync 只有“当前/最近一次”一个任务；GP 的 job 按环境
  // 常驻在内存里，如果每个 done 的 job 都渲染，做完几个环境就会挂一串进度条。
  // 这里只取“正在跑的那一个；否则最近启动的那一个”，和 T100 的展示口径一致。
  const pickGpJob = (kind: 'sync' | 'mirror'): GpEnvBrief | undefined => {
    const rows = gpEnvs.filter((g) =>
      g.job?.kind === kind &&
      (g.job.state === 'running' || g.job.state === 'done' || g.job.state === 'error'))
    return rows.find((g) => g.job?.state === 'running') ??
      rows.sort((a, b) => String(b.job?.startedAt || '').localeCompare(String(a.job?.startedAt || '')))[0]
  }
  const gpSyncJob = pickGpJob('sync')
  const gpMirrorJob = pickGpJob('mirror')


  return (
    // flex-1 + min-w-0:填满活动栏右侧的整块区域。少了它这个根会按内容宽度撑开,
    // 于是外层的滚动条落在内容列的右缘(页面中间),右边留一大片空白 —— 滚动条看着像没贴着边。
    // 填满之后里面的 mx-auto max-w-3xl 才真正居中,标题与卡片始终对齐在同一条竖线上。
    <div className="flex h-full min-h-0 min-w-0 flex-1 text-xs">
      {/* 左侧导航:四个分区各一行。
          分区不带图标,也不给「设置」这类标题 —— 这一栏本身就在设置页里,
          重复一遍栏名是废话,而图标只是把每个分区名往右推了一格。
          原先选中项下面还会展开该分区的卡片清单,已经去掉:右边一屏就是那个分区的全部卡片,
          清单只是把它们再念一遍。
          p-1 是给选中项的 rounded-ui 留出边距:栏内不留白的话,圆角会被这一栏的
          overflow-auto 直接裁掉,选中项看着还是直角。 */}
      <div className="flex w-44 shrink-0 flex-col gap-0.5 overflow-auto border-r border-border p-1">
        {SETTINGS_SECTIONS.map((s) => {
          const active = section === s.key
          return (
            <button
              key={s.key}
              onClick={() => setSection(s.key)}
              className={cn('flex w-full items-center rounded-ui px-2 py-1.5 text-left transition-colors',
                active ? 'bg-accent font-medium text-accent-foreground' : 'text-muted-foreground hover:bg-accent/60 hover:text-foreground')}
            >
              <span className="min-w-0 flex-1 truncate">{s.label}</span>
            </button>
          )
        })}
      </div>

      {/* 右侧内容区:当前分区的全部卡片,整体滚动。
          站点管理是左右两栏(左边环境清单、右边连接表单),列宽放宽一档给表单留出原来的舒适宽度;
          其余分区都是单列设置列表,仍然收在 max-w-3xl。
          px-1 不是给排版留白,是给**输入框的聚焦环**留位置:内层 mx-auto 的宽度就等于卡片宽度,
          卡片里贴边的输入框 focus 时 ring 要往外扩约 2px,没有这点余量就会被本层的
          overflow-auto 裁掉 —— 表现是"选中时高亮缺一条边/一角"。 */}
      <div className={cn('min-h-0 flex-1 px-1', section === 'sites' ? 'overflow-hidden' : 'overflow-auto')}>
        <div className={cn('mx-auto p-4', section === 'sites' ? 'flex h-full min-h-0 max-w-5xl flex-col gap-3' : 'max-w-3xl space-y-3')}>
          <div className="flex items-center justify-between gap-2">
            <h2 className="text-sm font-medium text-foreground">
              {SETTINGS_SECTIONS.find((s) => s.key === section)?.label}
            </h2>
          </div>

          {/* ============ 站点管理(左右两栏:环境清单 | 连接表单) ============ */}
          {section === 'sites' && (
            <div className="grid min-h-0 flex-1 grid-cols-[16rem_1fr] gap-3">
              {/* 两栏一起填满弹窗余下的高度:环境再多也不顶高页面,列表/表单各自在卡片里滚。
                  不再定死 40rem —— 那会让小窗口出现外层滚动、把页签行和底部按钮推出视野。 */}
              <Card
                id="card-sites-list"
                panel
                title="环境清单"
              >
                {/* 列表撑满卡片余下的高度、自己滚:环境再多也不顶高卡片,
                    「新增环境」钉在卡片底部,不用翻到列表末尾去找 */}
                <div className="min-h-0 flex-1 space-y-0.5 overflow-auto">
                  {/* 只列 TOP GP 环境：TG 是 GP 专用项目，T100 环境不再出现在树里
                      （T100 的站点表单与「发现环境」入口也在本步一并删除）。 */}
                  {gpDrafts.length === 0 && gpEnvs.length === 0 && (
                    <div className="py-2 text-muted-foreground">(还没有环境)</div>
                  )}
                  {gpDrafts.map((d, i) => (
                    <div key={'gp-draft:' + d.id}
                      className={cn('flex items-center gap-1 rounded-ui px-1.5 py-1 transition-colors',
                        gpDraftId === d.id ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50')}>
                      <button className="min-w-0 flex-1 text-left"
                        onClick={() => { clearSelection(); setGpDraftId(d.id) }}>
                        <span className="flex items-center gap-1.5 truncate">
                          <span className="rounded-[3px] bg-primary/15 px-1 text-[9px] font-semibold text-primary">TOP GP</span>
                          <span className="truncate">新环境 {i + 1}</span>
                        </span>
                        <span className="block truncate text-[11px] text-muted-foreground">未保存</span>
                      </button>
                    </div>
                  ))}
                  {gpEnvs.map((e) => (
                    <div key={'gp:' + e.name}
                      className={cn('flex items-center gap-1 rounded-ui px-1.5 py-1 transition-colors',
                        gpSel === e.name ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50')}>
                      <button className="min-w-0 flex-1 text-left" onClick={() => { setGpDraftId(null); selectGp(e.name) }}>
                        <span className="flex items-center gap-1.5 truncate">
                          <span className="rounded-[3px] bg-primary/15 px-1 text-[9px] font-semibold text-primary">TOP GP</span>
                          <span className="truncate">{e.name}</span>
                        </span>
                        <span className="block truncate text-[11px] text-muted-foreground">{e.host}:{e.port}</span>
                      </button>
                    </div>
                  ))}
                </div>
                <div className="flex items-center gap-2">
                  {/* 只有 GP 一种环境可建:原来那个「T100 / TOP GP」二选一的下拉与
                      「发现环境」(扫 T100 设计器站点导入)都已删除。 */}
                  <Button size="sm" className="h-7" onClick={() => { setGpDraftId(null); addGpDraft() }}>
                    <Plus className="h-3.5 w-3.5" />新增环境
                  </Button>
                </div>
              </Card>

              <Card id="card-sites-conn" panel>
                {(gpSel || selectedDraft) ? (
                  <div className="min-h-0 flex-1 overflow-auto">
                    <GpSites key={selectedDraft ? `gp-draft:${selectedDraft.id}` : `gp:${gpSel || ''}`}
                      only={gpSel || undefined} create={!!selectedDraft}
                      initial={selectedDraft?.env}
                      onSaved={(name) => {
                        if (selectedDraft) removeGpDraft(selectedDraft.id)
                        selectGp(name)
                      }}
                      onDiscard={() => {
                        if (selectedDraft) removeGpDraft(selectedDraft.id)
                      }}
                      onDraftChange={(env) => {
                        if (selectedDraft) updateGpDraft(selectedDraft.id, env)
                      }}
                      onCopyDraft={(env) => copyGpDraft(env)} />
                  </div>
                ) : (
                  <div className="py-2 text-muted-foreground">还没有 TOP GP 环境。点左下角的「+ 新增环境」建一个。</div>
                )}
              </Card>
            </div>
          )}

          {/* ============ 数据字典 ============ */}
          {section === 'data-dict' && (
            <>
              {/* 「查询数据源」把数据源选择与「数据同步」合在一张卡里:两者讲的是同一件事 ——
                  本地那份 SQLite 从哪来、查的时候走哪条路。原先分成两张卡,第一张只放一个下拉,
                  第二张才是"这份数据怎么来的",读的人得自己把两处接起来。 */}
              <Card id="card-dict-query" title="查询数据源"
                description="r.t / r.v / desc / scc / r.q 等查询命令走哪条路取数。">
                <SettingRow
                  id="query.source"
                  label="数据源"
                  description="在线走该环境直查；本地只读该环境自己的 SQLite，各环境互不干扰。本地快照目前只支持 T100 字典；TOP GP 的本地库暂不参与 r.t / r.v / desc 查询。"
                  control={
                    <Select
                      value={dict.querySource === '' ? 'online' : dict.querySource}
                      onValueChange={(v) => { setDict((s) => ({ ...s, querySource: v === 'online' ? '' : v })); markDirty('query') }}>
                      <SelectTrigger className="h-7 w-full text-xs"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="online">在线（查询时用 --env 指定环境，实时读 Oracle）</SelectItem>
                        {(sync?.envs ?? []).map((e) => (
                          <SelectItem key={e.name} value={`local:${e.name}`} disabled={!e.exists}>
                            本地 · T100 · {e.name}{e.exists ? '' : ' · 未同步'}
                          </SelectItem>
                        ))}
                        {dict.querySource.startsWith('local:') &&
                          !(sync?.envs ?? []).some((e) => `local:${e.name}` === dict.querySource) && (
                          <SelectItem value={dict.querySource}>本地 · T100 · {dict.querySource.slice(6)}（不在当前同步清单）</SelectItem>
                        )}
                        {dict.querySource !== '' && dict.querySource !== 'online' &&
                          dict.querySource !== 'local' && !dict.querySource.startsWith('local:') && (
                          <SelectItem value={dict.querySource}>在线（{dict.querySource}）</SelectItem>
                        )}
                        {dict.querySource === 'local' && (
                          <SelectItem value="local" disabled>本地（旧值：未指定环境）</SelectItem>
                        )}
                      </SelectContent>
                    </Select>
                  }
                />
                {/* 本地库放哪：只挪**根目录**，各环境仍是 <根>/<环境>.db。
                    不做"每环境一个文件路径"—— 那样命令行与界面容易读到不同的库，
                    而"读到另一个环境的库"和"数据不存在"在输出上分不出来。 */}
                <SettingRow
                  id="dict.dir"
                  label="库根目录（默认）"
                  description={
                    !dict.dictDir.trim()
                      ? `留空 = 缺省目录${sync?.dir ? `：${sync.dir}` : ''}；各环境的库仍是 <根目录>/<环境>.db。`
                      : status?.dict.exists
                        ? '目录已存在；各环境的库仍是 <根目录>/<环境>.db。'
                        : '目录不存在，同步时会自动创建。'
                  }
                  control={
                    <div className="flex min-w-0 items-center gap-1">
                      <Input className={input} value={dict.dictDir} placeholder={sync?.dir || '留空 = 缺省目录'}
                        onChange={(e) => { setDict((s) => ({ ...s, dictDir: e.target.value })); markDirty('dict') }} />
                      <Button variant="outline" size="sm" className="h-7 shrink-0 px-2 text-[11px]"
                        onClick={() => setDirPick({
                          open: true,
                          initial: dict.dictDir.trim() || sync?.dir || '',
                          title: '选择本地库根目录',
                          onPick: (path) => { setDict((s) => ({ ...s, dictDir: path })); markDirty('dict') },
                        })}>
                        选择
                      </Button>
                    </div>
                  }
                />
                {dict.querySource === 'local' && (
                  <p className="text-[11px] leading-relaxed text-amber-600 dark:text-amber-400">
                    旧配置 query.source = local 没有指定环境；请选「本地（某环境）」或「在线」，选完自动保存。
                  </p>
                )}
                {(() => {
                  const m = dict.querySource.match(/^local:(.+)$/)
                  const e = m ? (sync?.envs ?? []).find((x) => x.name === m[1]) : undefined
                  return m && e && !e.exists
                    ? <p className="text-[11px] leading-relaxed text-amber-600 dark:text-amber-400">
                        该环境的本地库还没同步；先点下面「开始同步」，再查本地。
                      </p>
                    : null
                })()}

                {/* 数据同步(原「数据同步」卡片):一个环境一行,每行一个「开始同步」。
                    它必须留在这张卡里 —— 上面选了「本地」就等于声明"我读本地那份",
                    而本地那份正是这张表同步出来的。 */}
                <SectionTitle>数据同步(写入本地 SQLite)</SectionTitle>
                {/* 这里原先内联一段失败红字(拉取/同步/停止失败),已改走统一 toast:
                    它出现与消失都会顶动这张卡的高度,而错误详情往往好几行。
                    保留在卡片里的只有**说明性**的一行(下面那条"没有可同步的环境"),
                    那是"当前状态"而不是"刚才那次动作的结果"。 */}
                {syncRows.length === 0 && (
                  <p className="text-[11px] text-muted-foreground">
                    没有可同步的环境 —— 需要先在「站点管理」为环境挂上数据库连接。
                  </p>
                )}
                {syncRows.length > 0 && (
                  <Table>
                    <TableHeader>
                      <TableRow className="hover:bg-transparent">
                        <TableHead>环境</TableHead>
                        <TableHead className="w-28">类型</TableHead>
                        <TableHead>地址</TableHead>
                        <TableHead className="w-36">本地库文件</TableHead>
                        <TableHead className="w-28">同步结果</TableHead>
                        <TableHead className="w-24">状态</TableHead>
                        <TableHead className="w-24">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {syncRows.map((row) => {
                        if (row.product === 't100') {
                          const e = row.t100
                          const job = sync?.job
                          const busy = !!job && job.env === e.name && (job.running || job.done)
                          const failed = busy && job?.phase === 'error'
                          return (
                            <TableRow key={'t100:' + e.name} className="hover:bg-transparent">
                              <TableCell className="px-2">
                                {e.name}
                              </TableCell>
                              <TableCell className="px-2">{e.type}</TableCell>
                              <TableCell className="px-2">
                                <span className="block truncate font-mono text-[11px] text-muted-foreground" title={e.address}>{e.address}</span>
                              </TableCell>
                              <TableCell className="px-2">
                                {/* 文件名固定是 <环境>.db；这里能改的只有**目录**。
                                    右键/按钮打开目录选择器，选完写回该环境的 dictDir。 */}
                                <div className="flex items-center gap-1">
                                  <span className="min-w-0 flex-1 truncate font-mono text-[11px] text-muted-foreground" title={e.dbPath}>
                                    {e.dbPath.split(/[\\/]/).pop()}
                                  </span>
                                  <button
                                    type="button"
                                    className="inline-flex h-5 w-5 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
                                    title={`选择「${e.name}」的库目录；现在：${e.dbPath}`}
                                    onClick={() => {
                                      const idx = sshs.findIndex((x) => sshName(x) === e.name)
                                      if (idx < 0) return
                                      setDirPick({
                                        open: true,
                                        initial: sshs[idx].dictDir.trim() || dict.dictDir.trim() || sync?.dir || '',
                                        title: `选择环境「${e.name}」的库目录`,
                                        onPick: (path) => patchSsh(idx, { dictDir: path }),
                                      })
                                    }}>
                                    <FolderOpen className="h-3 w-3" />
                                  </button>
                                </div>
                              </TableCell>
                              <TableCell className="px-2" title={e.dbPath}>
                                {e.exists
                                  ? <span className="text-emerald-600 dark:text-emerald-400">已同步 · {fmtBytes(e.size)}</span>
                                  : <span className="text-muted-foreground">未同步</span>}
                              </TableCell>
                              <TableCell className="px-2">
                                <EnvStatus busy={busy} running={!!job?.running} idle="空闲"
                                  error={failed ? (job?.error || job?.message || '失败') : undefined} />
                              </TableCell>
                              <TableCell className="px-2">
                                {/* 表格里的行内动作一律用 ghost 文本按钮(与账号表、环境动作行一致):
                                    一整列实心主色块会把"表格"读成"一堆按钮",行与行的主次也没了。
                                    ghost 只是没有底色,仍是完整按钮 —— 点击区、禁用态、悬停反馈都在。 */}
                                <Button variant="ghost" size="sm" disabled={syncBusy}
                                  title={`从「${e.name}」拉取字典数据到本地 SQLite`}
                                  onClick={() => void runSync(e.name)}>
                                  开始同步
                                </Button>
                              </TableCell>
                            </TableRow>
                          )
                        }
                        const g = row.gp
                        const job = g.job?.kind === 'sync' ? g.job : undefined
                        const running = job?.state === 'running'
                        const seen = !!job && (running || job.state === 'done' || job.state === 'error')
                        return (
                          <TableRow key={'gp:' + g.name} className="hover:bg-transparent">
                            <TableCell className="px-2">
                              {g.name}
                            </TableCell>
                            <TableCell className="px-2">oracle</TableCell>
                            <TableCell className="px-2">
                              <span className="block truncate font-mono text-[11px] text-muted-foreground"
                                title={`${g.db?.host}:${g.db?.port}/${g.db?.sid}`}>
                                {g.db?.host}:{g.db?.port}/{g.db?.sid}
                              </span>
                            </TableCell>
                            <TableCell className="px-2">
                              <span className="block truncate font-mono text-[11px] text-muted-foreground">{g.name}.db</span>
                            </TableCell>
                            <TableCell className="px-2">
                              {g.dict?.exists
                                ? <span className="text-emerald-600 dark:text-emerald-400">已同步 · {Number(g.dict.sizeMB ?? 0).toFixed(1)} MB</span>
                                : <span className="text-muted-foreground">未同步</span>}
                            </TableCell>
                            <TableCell className="px-2">
                              <EnvStatus busy={seen} running={running} idle="空闲"
                                error={job?.state === 'error' ? (job.error || '失败') : undefined} />
                            </TableCell>
                            <TableCell className="px-2">
                              <Button variant="ghost" size="sm" disabled={running}
                                title={`从「${g.name}」拉取 GP 字典到本地 SQLite`}
                                onClick={() => { void gpSync(g.name) }}>
                                开始同步
                              </Button>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                )}
                {sync?.job && (sync.job.running || sync.job.done) && (
                  <JobProgress
                    env={sync.job.env}
                    phaseLabel={SYNC_PHASE[sync.job.phase] || sync.job.phase}
                    tag={sync.job.running ? '进行中' : '已结束'}
                    pct={sync.job.phase === 'done' ? 100
                      : sync.job.tableTotal > 0 ? Math.round((sync.job.tableIndex * 100) / sync.job.tableTotal) : 0}
                    error={sync.job.phase === 'error' ? (sync.job.error || sync.job.message) : undefined}
                    message={sync.job.running ? undefined
                      : (sync.job.phase === 'cancelled' ? sync.job.message
                        : (sync.job.phase === 'done' ? sync.job.message : undefined))}
                    cells={[
                      sync.job.tableTotal > 0 ? `${sync.job.tableIndex}/${sync.job.tableTotal} 表` : '—',
                      sync.job.table || '',
                      sync.job.tableRows > 0 ? `${sync.job.tableRows.toLocaleString()} 行` : '',
                      sync.job.totalRows > 0 ? `共 ${sync.job.totalRows.toLocaleString()} 行` : '',
                      sync.job.elapsed || '',
                    ].filter(Boolean)}
                    stopping={sync.job.phase !== 'error' && sync.job.message === '正在停止…'}
                    onStop={sync.job.running ? () => void stopSync() : undefined}
                  />
                )}
                {gpSyncJob?.job && (
                  <div className="mt-2"><GpJobProgress env={gpSyncJob.name} job={gpSyncJob.job} /></div>
                )}
                {sync?.job?.backup && <p className="text-[11px] text-muted-foreground">原库已备份:{sync.job.backup}</p>}
                {sync?.job?.warning && <pre className="whitespace-pre-wrap text-[11px] text-amber-600 dark:text-amber-400">{sync.job.warning}</pre>}

                {/* 本地库的落点**只读**:它是个本地缓存文件,位置由服务端算(缺省在数据目录下),
                    让用户在设置页里挑一个路径,除了能把 CLI/Web/查询三边指向不同的库之外没有好处。
                    本地库按环境分文件；这里只读显示库目录。
                    原先这里还跟了一句状态说明(文件在不在、同步会先备份 .bak、位置是否来自配置):
                    那些都是"用户不需要为它做决定"的信息 —— 在不在不影响他点「开始同步」,
                    备份与失败保护是服务端自动行为,位置来源看这个只读值本身也够。 */}
                <InfoRow label="库目录" value={sync?.dir || '—'} />
              </Card>

              <Card id="card-dict-mirror" title="源码镜像"
                description="把 T100 源码与画面文件(4gl/**/*.4gl + 42f + 42s/zh_CN + *.inc)拉到本地,供 AI 检索与 tt outline / tt interface 离线使用；每个环境可单独指定镜像目录。">
                <SettingRow
                  id="mirror.dir"
                  label="镜像根目录（默认）"
                  description={
                    !dict.mirrorDir.trim()
                      ? '留空 = 不设默认根目录；下方每个环境仍可单独指定。两者都没有的环境不能拉取。'
                      : status?.mirror.exists
                        ? '目录已存在；下方每个环境可单独覆盖，留空则用根目录/<环境名>。'
                        : '目录不存在，拉取时自动创建；下方每个环境可单独覆盖。'
                  }
                  control={<Input className={input} value={dict.mirrorDir} placeholder="如 D:\t100\mirror"
                    onChange={(e) => { setDict((s) => ({ ...s, mirrorDir: e.target.value })); markDirty('mirror') }} />}
                />
                {/* 一个环境一行:镜像本来就是**按环境**分开的(每个环境一个镜像目录、一份基线),
                    原先"先在下拉里挑一个环境、再点按钮"把这件事藏进了两步操作里 ——
                    看不出总共有几个环境、各自拉没拉过。改成表之后这些直接摊开。 */}
                {mirrorRows.length === 0 && (
                  <p className="text-[11px] text-muted-foreground">还没有 SSH 环境,先去「站点管理」添加。</p>
                )}
                {mirrorRows.length > 0 && (
                  <Table>
                    <TableHeader>
                      <TableRow className="hover:bg-transparent">
                        <TableHead>环境</TableHead>
                        <TableHead className="w-28">状态</TableHead>
                        <TableHead>镜像目录</TableHead>
                        <TableHead className="w-44">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {mirrorRows.map((row) => {
                        if (row.product === 't100') {
                          const e = row.t100
                          const job = mirror?.job
                          const busy = !!job && job.env === e.name && (job.running || job.done)
                          const failed = busy && job?.phase === 'error'
                          const sshIndex = sshs.findIndex((x) => sshName(x) === e.name)
                          const ssh = sshIndex >= 0 ? sshs[sshIndex] : undefined
                          const rowReady = mirrorRowReady(e.name)
                          const rowTitle = !(ssh?.mirrorDir.trim() || dict.mirrorDir.trim())
                            ? '先填全局根目录，或在该环境行单独填镜像目录'
                            : isDirty('hosts') || isDirty('mirror')
                              ? '镜像目录正在自动保存，稍等'
                              : ''
                          return (
                            <TableRow key={'t100:' + e.name} className="hover:bg-transparent">
                              <TableCell className="px-2">
                                {e.name}
                              </TableCell>
                              <TableCell className="px-2">
                                <EnvStatus busy={busy} running={!!job?.running}
                                  idle={e.ready ? '已有镜像' : '未拉取'}
                                  error={failed ? (job?.error || job?.message || '失败') : undefined} />
                              </TableCell>
                              <TableCell className="px-1 py-1">
                                {/* 输入框不画自己的边框，整个单元格就是输入区；hover/focus 时才浮出边界。
                                    右侧文件夹按钮是原生选择器的入口 —— 浏览器拿不到绝对路径，只能让本机服务弹框。 */}
                                <div className="group flex h-8 min-w-0 items-center rounded-ui border border-transparent transition-colors hover:border-border/70 focus-within:border-border focus-within:bg-background/30">
                                  <input
                                    className="min-w-0 flex-1 bg-transparent px-2 font-mono text-[11px] text-foreground outline-none placeholder:text-muted-foreground/70"
                                    value={ssh?.mirrorDir || ''}
                                    placeholder={e.path || '留空 = 用全局根目录/<环境名>'}
                                    title={e.path || '未设置镜像目录'}
                                    onChange={(ev) => {
                                      if (sshIndex < 0) return
                                      patchSsh(sshIndex, { mirrorDir: ev.target.value })
                                    }} />
                                  <button
                                    type="button"
                                    className="mr-0.5 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-40"
                                    title="选择文件夹"
                                    aria-label="选择文件夹"
                                    disabled={sshIndex < 0}
                                    onClick={() => void pickMirrorDir(
                                      e.name, sshIndex, ssh?.mirrorDir || e.path || dict.mirrorDir)}>
                                    <FolderOpen className="h-3.5 w-3.5" />
                                  </button>
                                </div>
                              </TableCell>
                              <TableCell className="px-2">
                                <div className="flex items-center gap-1 whitespace-nowrap">
                                  {/* 同上:两个都走 ghost 文本按钮。原来"增量更新"是实心主色,
                                      一行里最重的视觉落在按钮上而不是"哪个环境、拉没拉过"。
                                      「全量」也去掉 Download 图标:同一格两个按钮,一个带图标一个不带
                                      会让它们看着不是一类东西;动作名已经把事说清了。 */}
                                  <Button variant="ghost" size="sm" disabled={mirrorBusy || !rowReady}
                                    title={rowTitle || '只拉服务器上变过的文件'}
                                    onClick={() => void pullMirror(false, e.name)}>
                                    增量更新
                                  </Button>
                                  <Button variant="ghost" size="sm" disabled={mirrorBusy || !rowReady}
                                    title={rowTitle || '整目录替换(含服务器上已删的残留)'}
                                    onClick={() => void pullMirror(true, e.name)}>
                                    全量
                                  </Button>
                                </div>
                              </TableCell>
                            </TableRow>
                          )
                        }
                        const g = row.gp
                        const job = g.job?.kind === 'mirror' ? g.job : undefined
                        const running = job?.state === 'running'
                        const seen = !!job && (running || job.state === 'done' || job.state === 'error')
                        return (
                          <TableRow key={'gp:' + g.name} className="hover:bg-transparent">
                            <TableCell className="px-2">
                              {g.name}
                            </TableCell>
                            <TableCell className="px-2">
                              <EnvStatus busy={seen} running={running}
                                idle={g.mirror?.exists ? '已有镜像' : '未拉取'}
                                error={job?.state === 'error' ? (job.error || '失败') : undefined} />
                            </TableCell>
                            <TableCell className="px-1 py-1">
                              <div className="group flex h-8 min-w-0 items-center rounded-ui border border-transparent transition-colors hover:border-border/70 focus-within:border-border focus-within:bg-background/30">
                                <input
                                  className="min-w-0 flex-1 bg-transparent px-2 font-mono text-[11px] text-foreground outline-none placeholder:text-muted-foreground/70"
                                  value={gpMirrorDraft[g.name] ?? (g.mirrorDir || '')}
                                  placeholder={g.mirror?.path || '留空 = 默认 <镜像根>/<环境名>'}
                                  title={g.mirror?.path || '未设置镜像目录'}
                                  onChange={(ev) => setGpMirrorDraft((m) => ({ ...m, [g.name]: ev.target.value }))}
                                  onBlur={() => { void gpPatchEnv(g.name, { mirrorDir: gpMirrorDraft[g.name] ?? g.mirrorDir ?? '' }) }}
                                />
                                <button
                                  type="button"
                                  className="mr-0.5 inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-muted-foreground hover:bg-accent hover:text-foreground"
                                  title="选择文件夹"
                                  aria-label="选择文件夹"
                                  onClick={async () => {
                                    const p = await gpPickDir(g.mirrorDir || g.mirror?.path || '')
                                    if (!p) return
                                    setGpMirrorDraft((m) => ({ ...m, [g.name]: p }))
                                    await gpPatchEnv(g.name, { mirrorDir: p })
                                  }}>
                                  <FolderOpen className="h-3.5 w-3.5" />
                                </button>
                              </div>
                            </TableCell>
                            <TableCell className="px-2">
                              <div className="flex items-center gap-1 whitespace-nowrap">
                                <Button variant="ghost" size="sm" disabled={running}
                                  title="只拉服务器上变过的文件"
                                  onClick={() => { void gpMirror(g.name, 'incr') }}>
                                  增量更新
                                </Button>
                                <Button variant="ghost" size="sm" disabled={running}
                                  title="整目录替换（含服务器上已删的残留）"
                                  onClick={() => { void gpMirror(g.name, 'full') }}>
                                  全量
                                </Button>
                              </div>
                            </TableCell>
                          </TableRow>
                        )
                      })}
                    </TableBody>
                  </Table>
                )}
                {mirror?.job && (mirror.job.running || mirror.job.done) && (
                  <JobProgress
                    env={mirror.job.env}
                    phaseLabel={MIRROR_PHASE[mirror.job.phase] || mirror.job.phase}
                    tag={`${mirror.job.full ? '全量' : '增量'} · ${mirror.job.running ? '进行中' : '已结束'}`}
                    pct={mirror.job.total > 0
                      ? Math.round((mirror.job.bytes * 100) / mirror.job.total)
                      : mirror.job.phase === 'done' ? 100 : 0}
                    error={mirror.job.phase === 'error' ? (mirror.job.error || mirror.job.message) : undefined}
                    message={mirror.job.running ? undefined
                      : (mirror.job.phase === 'cancelled' ? mirror.job.message
                        : (mirror.job.phase === 'done' ? mirror.job.message : undefined))}
                    cells={[
                      ...mirrorCellTexts(mirror.job),
                      !mirror.job.running && mirror.job.elapsed ? `${mirror.job.elapsed}` : '',
                    ].filter(Boolean)}
                    stopping={mirror.job.phase !== 'error' && mirror.job.message === '正在停止…'}
                    onStop={mirror.job.running ? () => void stopMirror() : undefined}
                  />
                )}
                {gpMirrorJob?.job && (
                  <div className="mt-2"><GpJobProgress env={gpMirrorJob.name} job={gpMirrorJob.job} /></div>
                )}
              </Card>

            </>
          )}

          {/* ============ DEBUG ============ */}
          {/* ============ TOP GP（产品开关切到 GP 时，顶替站点管理与数据字典两块） ============ */}
          {section === 'debug' && (
            <>
              {/* 只剩看门狗一项。
                  其余五项(启动参数模板 / FGLServer / print 上限 / 终端尺寸 / 断点持久化)
                  已从界面移除:它们都有内置默认值(见 internal/config 的 FillDefaults:
                  BBDL…{prog} / 按 SSH 来源 IP / 1000 / 200×50 / 开启),现场几乎不需要改,
                  摆出来只会让人以为"必须调"。
                  ⚠ 但它们的值仍然**原样回传**(见 debugPatchOf 的注释):后端的 PUT 是
                  整节替换 + 字段 omitempty,不回传等于把这些键从 config.json 里删掉。
                  「默认环境」那张卡整张去掉 —— 调试会话不指定环境时继承站点默认
                  (站点管理里那个绿点),这一层覆盖没必要暴露。 */}
              <Card id="card-debug-params" title="调试参数"
                description="本机调试会话的默认参数。">
                <SettingRow id="debug.watchdogSeconds" label="停站看门狗(秒)"
                  description="停站后超过该时长无响应即判定失联。"
                  control={<Input className={input} type="number" value={dbg.watchdogSeconds || ''} placeholder="1800"
                    onChange={(e) => { setDbg((s) => ({ ...s, watchdogSeconds: Number(e.target.value) || 0 })); markDirty('debug') }} />} />
              </Card>
            </>
          )}

          {/* ============ 技能 ============ */}
          {section === 'skills' && <SkillsPanel />}

          {/* ============ 应用设置 ============ */}
          {section === 'app' && (
            <>
              <Card id="card-app-theme" title="外观"
                description="全站共用一份;切换后各视图都跟着变。">
                <ThemeCards value={theme} onChange={setTheme} />
              </Card>

              <Card id="card-app-service" title="服务"
                description="统一 Web 服务的监听地址。">
                <SettingRow id="listen" label="监听地址"
                  description="改完需重启 tt serve;端口占用时自动向后顺延。"
                  control={<Input className={input} value={listen} placeholder="127.0.0.1:28670"
                    onChange={(e) => { setListen(e.target.value); markDirty('listen') }} />} />
              </Card>

              <Card id="card-app-install" title="命令行集成"
                description="把 tt.exe 目录加入用户 PATH,之后任意终端可直接敲 tt。只改当前用户,不需管理员。">
                <SettingRow
                  id="app.install"
                  label="用户 PATH"
                  description={
                    !status?.install.supported
                      ? (status?.install.note || '本平台不支持自动写入')
                      : status.install.inUserPath
                        ? '已在用户 PATH 中'
                        : '尚未加入 —— 加完之后新开的终端才能直接敲 tt'
                  }
                  control={
                    <div className="flex items-center gap-1.5">
                      {status?.install.inUserPath ? (
                        <Button variant="outline" size="sm" disabled={installBusy}
                          onClick={() => void doInstall(false)}>
                          <Trash2 className="h-3.5 w-3.5" />{installBusy ? '处理中…' : '从 PATH 移除'}
                        </Button>
                      ) : (
                        <Button size="sm" disabled={installBusy || !status?.install.supported}
                          onClick={() => void doInstall(true)}>
                          <Terminal className="h-3.5 w-3.5" />{installBusy ? '处理中…' : '加入用户 PATH'}
                        </Button>
                      )}
                    </div>
                  }
                />
                {status?.install.exeDir && <InfoRow label="程序目录" value={status.install.exeDir} />}
                {!status?.install.supported && status?.install.manual && (
                  <pre className="overflow-x-auto border border-border bg-muted/30 p-2 text-[11px]">{status.install.manual}</pre>
                )}
              </Card>

              <Card id="card-app-info" title="运行信息">
                <InfoRow label="版本" value={cfg.version || '(未知)'} />
                <InfoRow label="配置文件" value={cfg.config} />
                <InfoRow label="缺省配置位置" value={meta?.defaultConfig || '—'} />
                <InfoRow label="便携模式" value={meta ? (meta.portable ? '是(配置留在程序目录)' : '否') : '—'} />
                <InfoRow label="统一工具目录" value={meta?.toolsHome || '—'} />
                <InfoRow label="配置结构版本" value={meta ? String(meta.schemaVersion) : '—'} />
                <InfoRow label="支持的库类型" value={meta?.supportedTypes?.join(' / ') || '—'} />
                {/* 「默认环境」一行已删:没有默认环境了(环境一律 --env 点名)。 */}
                <InfoRow label="环境数" value={String(cfg.sshs?.length ?? 0)} />
              </Card>

              <Card id="card-app-update" title="更新"
                description="检查/升级 tt 自身。查由你手动触发;升级会校验 sha256 后交由独立更新器完成。">
                <InfoRow label="当前版本" value={upd?.version || cfg.version || '(未知)'} />
                <InfoRow label="安装形态" value={upd ? `${upd.kind}${upd.canInstall ? '' : '（不让自装）'}` : '—'} />
                <InfoRow label="更新源" value={upd?.source || '—'} />
                {upd?.check && (
                  <InfoRow label="上次检查"
                    value={`${upd.check.latest}${upd.check.newer ? '（有新版本）' : '（已是最新）'} · ${new Date(upd.check.checkedAt).toLocaleString()}`} />
                )}
                {upd?.last && (
                  <InfoRow label="上次升级"
                    value={`${upd.last.phase}${upd.last.target ? ` → ${upd.last.target}` : ''}${upd.last.error ? ` · ${upd.last.error}` : ''}`} />
                )}
                {upd?.skillsDrift && <InfoRow label="技能" value={upd.skillsDrift} />}
                {upd && !upd.canInstall && upd.refusal && (
                  <p className="text-[11px] text-muted-foreground">{upd.refusal}</p>
                )}
                <div className="flex items-center gap-1.5">
                  <Button variant="outline" size="sm" disabled={updBusy !== ''}
                    onClick={() => void doUpdateCheck()}>
                    <RefreshCw className="h-3.5 w-3.5" />{updBusy === 'check' ? '检查中…' : '检查更新'}
                  </Button>
                  {upd?.check?.newer && upd.canInstall && (
                    <Button size="sm" disabled={updBusy !== ''} onClick={() => void doUpdateApply()}>
                      <Download className="h-3.5 w-3.5" />{updBusy === 'apply' ? '交棒中…' : `升级到 ${upd.check.latest}`}
                    </Button>
                  )}
                  {(updBusy !== '' || upd?.check?.newer) && (
                    <span className="text-[11px] text-muted-foreground">
                      {updBusy === 'apply' ? '升级期间本页会断开，之后重新打开即可' : '也可以直接敲 tt update'}
                    </span>
                  )}
                </div>
              </Card>

              <Card id="card-app-cache" title="缓存">
                <p className="text-[11px] text-muted-foreground">
                  跑出来的可再生数据(企业快照、源码镜像、查询落盘、断点存档)。它们与 config.json
                  同目录,但性质相反 —— 清除只动这些,配置文件一律不碰。
                </p>
                <InfoRow label="缓存目录" value={status?.cache?.dir || '—'} />
                {(status?.cache?.dirs ?? []).map((d) => (
                  <div key={d.name} className="flex items-center justify-between gap-2 text-[11px]">
                    <span className="font-mono">{d.name}</span>
                    <span className="text-muted-foreground">
                      {d.exists ? `${fmtBytes(d.bytes)} · ${d.files} 个文件` : '无'}
                    </span>
                  </div>
                ))}
                <InfoRow
                  label="共"
                  value={`${fmtBytes(status?.cache?.bytes ?? 0)} · ${status?.cache?.files ?? 0} 个文件`}
                />
                <div className="flex flex-wrap items-center gap-2 pt-1">
                  <Button size="sm" variant="outline" disabled={cacheBusy}
                    onClick={() => void doClearCache()}>
                    <Trash2 className="mr-1 h-3.5 w-3.5" />
                    {cacheBusy ? '清理中…' : '清除缓存'}
                  </Button>
                </div>
                <p className="text-[11px] text-muted-foreground">
                  tt serve 启动时会自动清理超过{' '}
                  {Math.max(1, Math.round((status?.cache?.maxAgeHours ?? 168) / 24))} 天的缓存
                  —— 刻意不"启动即全清":查询落盘的那些文件正是"刚才那条查询的完整结果",
                  启动就删会把上一条命令刚告诉你的东西删掉。
                </p>
              </Card>
            </>
          )}
        </div>
      </div>

      {/* 全量重建 / 开始同步 / 升级 tt 的二次确认:同一套 AlertDialog。
          文案里的换行(desc 里的 \n)由 AlertDialogDescription 的 whitespace-pre-line 呈现。 */}
      <AlertDialog open={ask !== null} onOpenChange={(o) => { if (!o) setAsk(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{ask?.title}</AlertDialogTitle>
            <AlertDialogDescription>{ask?.desc}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => { const a = ask; setAsk(null); a?.run() }}>
              {ask?.confirmText}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除环境二次确认 */}
      <AlertDialog open={delTarget !== null} onOpenChange={(o) => { if (!o) setDelTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>删除环境「{delTarget !== null ? sshName(sshs[delTarget]) : ''}」?</AlertDialogTitle>
            <AlertDialogDescription>
              该环境的 SSH 连接、登录区域、工作区与数据库账号配置将一并移除。
              它名下**长期库里的录像也会一起删掉**（不可再生，删了就再也造不出来）。
              点击「删除」后立即写入 config.json,不可撤销。
              {'\n'}只是改了名字的话不会删录像：改名会把它们搬到新名字下。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction onClick={() => void confirmDelSsh()}>删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 发现环境:扫描本机 T100 设计器的站点清单,勾选导入。
          用 Dialog 而不是 AlertDialog —— 这里装的是可交互内容(勾选列表),
          点遮罩/按 Esc 就该关掉,那是 Dialog 的语义(AlertDialog 会强制用户先做选择)。 */}
      <Dialog open={disc.open} onOpenChange={(o) => setDisc((d) => ({ ...d, open: o }))}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>发现环境 — 从 T100 设计器导入</DialogTitle>
            <DialogDescription>
              读取本机设计器的站点清单(只读,不会改动设计器的配置),勾选后加入环境清单。
              {"\n"}设计器里没有数据库配置,导入完成后请在「数据库」页签补上。
            </DialogDescription>
          </DialogHeader>

          {disc.loading ? (
            <div className="flex items-center gap-2 py-6 text-muted-foreground">
              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
              正在扫描设计器的站点清单…
            </div>
          ) : disc.envs.length === 0 ? (
            // 没扫到不是错误:把原因原样说给用户,并把查过的路径也列出来(便于自查)
            <div className="space-y-1.5 py-3">
              <p className="text-muted-foreground">{disc.reason || '没有发现可导入的环境。'}</p>
              {disc.sitesFile && (
                <p className="font-mono text-[11px] break-all text-muted-foreground">查找位置:{disc.sitesFile}</p>
              )}
            </div>
          ) : (
            <div className="min-h-0 flex-1 space-y-0.5 overflow-auto">
              {disc.envs.map((e, i) => {
                const key = discKey(e, i)
                const dup = existingNames.has(e.name)
                const disabled = !!e.skipped
                return (
                  <label key={key}
                    className={cn('flex items-start gap-2 rounded-ui px-1.5 py-1.5',
                      disabled ? 'opacity-50' : 'cursor-pointer hover:bg-accent/50')}>
                    <Checkbox className="mt-0.5" disabled={disabled}
                      checked={!!disc.picked[key] && !disabled}
                      onCheckedChange={(v) => setDisc((d) => ({ ...d, picked: { ...d.picked, [key]: v === true } }))} />
                    <span className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-center gap-1.5">
                        <span className="font-medium">{e.name}</span>
                        {dup && <span className="rounded-ui bg-muted px-1 text-[11px] text-muted-foreground">已存在</span>}
                        {e.folder && <span className="text-[11px] text-muted-foreground">来自文件夹「{e.folder}」</span>}
                      </span>
                      {e.skipped ? (
                        <span className="block text-[11px] text-red-600 dark:text-red-400">{e.skipped}</span>
                      ) : (
                        <span className="block font-mono text-[11px] text-muted-foreground">
                          {e.user}@{e.host}:{e.port}
                          {e.zone ? ` · 区域 ${e.zone}` : ''}
                          {e.workspace ? ` · ${e.workspace}` : ''}
                        </span>
                      )}
                    </span>
                  </label>
                )
              })}
            </div>
          )}

          <DialogFooter>
            <Button variant="ghost" size="sm" onClick={() => setDisc((d) => ({ ...d, open: false }))}>取消</Button>
            <Button variant="outline" size="sm" onClick={openDiscover} disabled={disc.loading}>
              <RefreshCw className="h-3.5 w-3.5" />重新扫描
            </Button>
            <Button size="sm" onClick={confirmDiscover}
              disabled={disc.loading || disc.envs.length === 0}>
              导入选中
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 目录选择器：前端自绘（后端列目录）。比服务端弹 Windows 对话框快一个数量级，
          而且外观跟设置页一致 —— 镜像目录的按钮就用它。 */}
      <DirPicker
        open={dirPick.open}
        initial={dirPick.initial}
        title={dirPick.title}
        onPick={dirPick.onPick}
        onClose={() => setDirPick((d) => ({ ...d, open: false }))}
      />
    </div>
  )
}
