// TOP GP 的两块设置：站点管理（环境）与数据字典（同步 + 源码镜像）。
//
// 数据来自统一服务的 /api/gp（与 T100 同一份 config.json、同一套端口），
// 不再连 28671 的 ttgp。表单排版与 T100 的站点管理一致：两列平铺、
// SSH / 数据库 两个页签、没有 ENT；保存也跟 T100 一样走 500ms 自动保存。

import { useCallback, useEffect, useRef, useState } from 'react'
import { Check, Copy, Eye, EyeOff, RefreshCw, Trash2 } from 'lucide-react'
import {
  Button, Input, Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from '../../shared/ui'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../../shared/ui-radix'
import { Card, Field } from '../../shared/settings'
import { cn } from '../../shared/utils'
import { useToast } from './toast'

interface DbAcc { account: string; password?: string; hasPassword?: boolean }

/** 一个营运据点（azp_file 的一行）：代号 + 名称。 */
interface SiteRow { code: string; name: string }

/** /api/gp/test 单项结果。 */
interface TestRes { ok: boolean; msg: string }

/** 一项测试的结果行：✔/✘ + 文案。 */
function TestLine({ label, res }: { label: string; res?: TestRes }) {
  if (!res) return null
  return (
    <p className="text-[11px] leading-relaxed">
      <span className={cn('mr-1', res.ok ? 'text-emerald-500' : 'text-red-500')}>{res.ok ? '✔' : '✘'}</span>
      <span className="text-muted-foreground">{label}：{res.msg}</span>
    </p>
  )
}

export interface GpEnv {
  name: string
  host?: string
  port?: number
  user?: string
  password?: string
  hasPassword?: boolean
  instance?: string
  workspace?: string
  custRoot?: string
  /** 营运据点代号（azp_file.azp01 小写 = zta_file.zta02 = Oracle schema）；空 = "ds"。 */
  company?: string
  /** 营运据点名称（azp_file.azp02），「从服务器刷新」时抄下来，只是显示用。 */
  companyName?: string
  /** 上次「从服务器刷新」的时间（RFC3339）；只用来显示"什么时候抄的"。 */
  companySyncedAt?: string
  mirrorDir?: string
  dictDir?: string
  db?: { type?: string; host?: string; port?: number; sid?: string; service?: string; accounts?: DbAcc[] }
  dict?: { exists?: boolean; sizeMB?: number }
  mirror?: { exists?: boolean; files?: number; sizeMB?: number; path?: string }
  job?: { state?: string; kind?: string; progress?: string[] }
}

/** 只提交这次真正变了的字段；后端按补丁合并，不再整条替换。 */
interface GpEnvPatch {
  name?: string
  host?: string
  port?: number
  user?: string
  password?: string
  instance?: string
  workspace?: string
  custRoot?: string
  company?: string
  companyName?: string
  companySyncedAt?: string
  mirrorDir?: string
  dictDir?: string
  db?: {
    type?: string
    host?: string
    port?: number
    service?: string
    database?: string
    accounts?: DbAcc[]
  }
}

const input = 'h-7 w-full min-w-0 text-xs'

export function emptyGpEnv(): GpEnv {
  return {
    name: '', host: '', port: 22, user: '', hasPassword: false,
    instance: '', workspace: '', custRoot: '', company: '', companyName: '', companySyncedAt: '',
    db: { type: 'oracle', host: '', port: 1521, service: '', accounts: [{ account: '', password: '' }] },
  }
}
const emptyEnv = emptyGpEnv

/**
 * 表单基线：只包含本表单可编辑的字段，不含密码（密码单独用 pwd/dbPwd 的脏标记）。
 * 不把服务端返回的 dict/mirror/job 放进来 —— 它们会在任务轮询时变化，
 * 放进来会让"没改过"的表单反复变脏。
 */
function envSnapshot(c: GpEnv): string {
  return JSON.stringify({
    name: (c.name || '').trim(),
    host: (c.host || '').trim(),
    port: c.port ?? 22,
    user: (c.user || '').trim(),
    instance: (c.instance || '').trim(),
    company: (c.company || '').trim(),
    companyName: (c.companyName || '').trim(),
    companySyncedAt: (c.companySyncedAt || '').trim(),
    db: {
      host: (c.db?.host || '').trim(),
      port: c.db?.port ?? 1521,
      service: (c.db?.service || '').trim(),
      account: (c.db?.accounts?.[0]?.account || '').trim(),
    },
  })
}

/** 生成只包含差异的补丁；密码为空时不提交，交由后端保留旧口令。 */
function buildEnvPatch(
  c: GpEnv,
  base: string,
  pwd: string,
  pwdBase: string,
  dbPwd: string,
  dbPwdBase: string,
): GpEnvPatch {
  const b = base ? JSON.parse(base) : {}
  const patch: GpEnvPatch = {}
  const name = (c.name || '').trim()
  if (name !== (b.name || '')) patch.name = name
  if ((c.host || '').trim() !== (b.host || '')) patch.host = (c.host || '').trim()
  if ((c.port ?? 22) !== (b.port ?? 22)) patch.port = c.port ?? 22
  if ((c.user || '').trim() !== (b.user || '')) patch.user = (c.user || '').trim()
  if ((c.instance || '').trim() !== (b.instance || '')) patch.instance = (c.instance || '').trim()
  if ((c.company || '').trim() !== (b.company || '')) patch.company = (c.company || '').trim()
  if ((c.companyName || '').trim() !== (b.companyName || '')) patch.companyName = (c.companyName || '').trim()
  if ((c.companySyncedAt || '').trim() !== (b.companySyncedAt || '')) patch.companySyncedAt = (c.companySyncedAt || '').trim()

  const bdb = b.db || {}
  const db: NonNullable<GpEnvPatch['db']> = {}
  if ((c.db?.host || '').trim() !== (bdb.host || '')) db.host = (c.db?.host || '').trim()
  if ((c.db?.port ?? 1521) !== (bdb.port ?? 1521)) db.port = c.db?.port ?? 1521
  if ((c.db?.service || '').trim() !== (bdb.service || '')) db.service = (c.db?.service || '').trim()
  const account = (c.db?.accounts?.[0]?.account || '').trim()
  if (account !== (bdb.account || '')) db.accounts = [{ account }]
  // 只有“这次真的改过密码”才提交；空口令不提交，后端保留旧值。
  if (dbPwd && dbPwd !== dbPwdBase) db.accounts = [{ account, password: dbPwd }]
  if (Object.keys(db).length) patch.db = db
  if (pwd && pwd !== pwdBase) patch.password = pwd
  return patch
}

/** 两个分区共用的一小段状态：服务地址 + 环境清单。 */
function useGp() {
  const [base] = useState('') // 固定同源：GP 端点已挂在统一服务 /api/gp 下，不再读旧的 tt.gp.base
  const [envs, setEnvs] = useState<GpEnv[]>([])
  const [err, setErr] = useState('')
  const [querySource, setQuerySource] = useState('auto')

  const call = useCallback(async (path: string, init?: RequestInit) => {
    const r = await fetch(base + path, init)
    return await r.json()
  }, [base])

  const load = useCallback(async () => {
    try {
      const m = await call('/api/gp/meta')
      setEnvs(m.envs || [])
      setQuerySource(m.querySource || 'auto')
      setErr('')
    } catch (e) {
      setErr('连不上 TOP GP 服务（' + base + '）：' + String(e))
    }
  }, [call, base])

  useEffect(() => { void load() }, [load])

  return { base, envs, setEnvs, err, setErr, call, load, querySource, setQuerySource }
}

/** 站点管理（TOP GP）：与 T100 同样的两列表单、同样的 500ms 自动保存。 */
export function GpSites({ only, create, initial, onSaved, onDiscard, onDraftChange, onCopyDraft }: {
  only?: string
  create?: boolean
  /** 新建草稿的初始内容（复制时用）。 */
  initial?: GpEnv
  /** 保存成功后把左树/设置页的选中项跟到新名字（新建、改名、复制都需要）。 */
  onSaved?: (name: string) => void
  /** 删除未保存草稿：只清本地草稿，不调后端（草稿没有正式环境名）。 */
  onDiscard?: () => void
  /** 草稿内容变化时同步给父级，便于切换/复制草稿。 */
  onDraftChange?: (env: GpEnv) => void
  /** 复制草稿：交给父级追加一条新草稿，而不是覆盖当前草稿。 */
  onCopyDraft?: (env: GpEnv) => void
}) {
  const g = useGp()
  const toast = useToast()
  const [tab, setTab] = useState<'ssh' | 'db' | 'site'>('ssh')
  const [cur, setCur] = useState<GpEnv | null>(create ? (initial ?? emptyEnv()) : null)
  const [pwd, setPwd] = useState('')
  const [dbPwd, setDbPwd] = useState('')
  const [showPwd, setShowPwd] = useState(false)
  const [showDbPwd, setShowDbPwd] = useState(false)
  const [msg, setMsg] = useState('')
  const [busy, setBusy] = useState(false)
  const [saveState, setSaveState] = useState<'idle' | 'saving' | 'saved'>('idle')
  // 营运据点：从服务器抄 azp_file 的结果与忙状态（一次取全表，摆成一张可点的表）。
  const [siteBusy, setSiteBusy] = useState(false)
  const [siteMsg, setSiteMsg] = useState('')
  const [sites, setSites] = useState<SiteRow[]>([])
  // 每个页签各存各的结果：互不覆盖，切回去还能看到上次测的。
  const [test, setTest] = useState<{ ssh?: TestRes; db?: TestRes; site?: TestRes } | null>(null)
  const [testBusy, setTestBusy] = useState<'' | 'ssh' | 'db' | 'site'>('')

  // 基线 = 上一次读入/保存成功时的“表单值”（不含密码）；脏 = 碰过且与基线不同。
  const baselineRef = useRef('')
  // 密码单独记基线：保存成功后**不把输入框清空**。清空会让用户继续输入时
  // 只把后半截保存进去（实测：SSH 密码被存成 3 个字符），下一次自动保存又把
  // 整段覆盖成这半截。
  const pwdBaselineRef = useRef('')
  const dbPwdBaselineRef = useRef('')
  const selectedNameRef = useRef('')
  const saveChainRef = useRef<Promise<void>>(Promise.resolve())
  const curRef = useRef(cur)
  const pwdRef = useRef(pwd)
  const dbPwdRef = useRef(dbPwd)
  curRef.current = cur
  pwdRef.current = pwd
  dbPwdRef.current = dbPwd

  // 选中环境/新建环境时重置基线；保存成功后 g.envs 变化不重置，避免把用户
  // 正在输入的内容回灌成服务端旧值。
  useEffect(() => {
    if (create) {
      if (!curRef.current) {
        const e = initial ?? emptyEnv()
        setCur(e)
        baselineRef.current = envSnapshot(e)
        selectedNameRef.current = ''
      }
      return
    }
    const target = only ? g.envs.find((e) => e.name === only) : g.envs[0]
    if (!target) return
    if (selectedNameRef.current === target.name) return
    setCur(target)
    baselineRef.current = envSnapshot(target)
    selectedNameRef.current = target.name
    // 和 T100 一样：密码回填到输入框（显示为圆点），后续输入是在完整口令上改，
    // 不是从空白重新拼。保存成功后也不清空。
    const savedPwd = target.password || ''
    const savedDbPwd = target.db?.accounts?.[0]?.password || ''
    setPwd(savedPwd)
    setDbPwd(savedDbPwd)
    pwdBaselineRef.current = savedPwd
    dbPwdBaselineRef.current = savedDbPwd
    setTest(null)
    setSaveState('idle')
  }, [g.envs, only, create])

  const patch = (p: Partial<GpEnv>) => setCur((c) => ({ ...(c || emptyEnv()), ...p }))

  // 草稿内容同步给父级；父级用它维护多条草稿的切换和复制。
  useEffect(() => {
    if (create && cur && onDraftChange) onDraftChange(cur)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cur, create])

  const payload = (c: GpEnv) => ({
    ...c,
    password: pwd || undefined,
    db: {
      ...(c.db || {}),
      accounts: [{ account: c.db?.accounts?.[0]?.account || '', password: dbPwd || undefined }],
    },
  })

  // 和 T100 完全同款：脏 = 表单值变了，密码也是表单值的一部分。
  // 500ms 防抖会重置，连续输入期间不会保存；保存后输入框保留完整口令。
  const isDirty = () => !!cur && (
    envSnapshot(cur) !== baselineRef.current ||
    pwd !== pwdBaselineRef.current ||
    dbPwd !== dbPwdBaselineRef.current
  )
  const canSave = (c: GpEnv | null = curRef.current) =>
    !!c && !!c.name.trim() && !!c.host?.trim() && !!c.user?.trim() && !!c.instance?.trim()

  // 保存只发差异补丁；后端按字段合并，不会把未提交的 service / 密码 / 目录清掉。
  const doSave = async () => {
    const c = curRef.current
    if (!c || !canSave(c)) return
    const name = c.name.trim()
    const sentSnapshot = envSnapshot(c)
    const sentPwd = pwdRef.current
    const sentDbPwd = dbPwdRef.current
    const envPatch = buildEnvPatch(
      c, baselineRef.current,
      sentPwd, pwdBaselineRef.current,
      sentDbPwd, dbPwdBaselineRef.current,
    )
    if (Object.keys(envPatch).length === 0) return
    setSaveState('saving')
    try {
      const oldName = selectedNameRef.current
      const r = await g.call('/api/gp/env', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'save',
          name,
          oldName: oldName && oldName !== name ? oldName : undefined,
          patch: envPatch,
        }),
      })
      if (!r.ok) throw new Error(r.error || '保存失败')
      baselineRef.current = sentSnapshot
      selectedNameRef.current = name
      // 保留输入框里的密码（不清空），只把基线对齐到刚提交的值。
      // 这样用户继续输入时是从完整口令往后加，不会把半截口令当成新口令。
      pwdBaselineRef.current = sentPwd
      dbPwdBaselineRef.current = sentDbPwd
      setSaveState('saved')
      // 只更新本地清单，不回读整张表单；否则保存期间用户的下一步输入会被旧值盖掉。
      g.setEnvs((prev) => {
        const next: GpEnv = {
          ...c,
          name,
          password: sentPwd,
          hasPassword: !!sentPwd,
          db: c.db
            ? {
                ...c.db,
                accounts: (c.db.accounts || []).map((a, i) => ({
                  ...a,
                  password: i === 0 ? sentDbPwd : a.password,
                  hasPassword: i === 0 ? !!sentDbPwd : !!a.hasPassword,
                })),
              }
            : c.db,
        }
        const idx = prev.findIndex((e) => e.name === oldName || e.name === name)
        if (idx >= 0) {
          const out = [...prev]
          out[idx] = next
          return out
        }
        return [...prev, next]
      })
      // 只刷新“有没有密码”的显示，不回灌用户正在编辑的其它字段。
      setCur((prev) => prev ? {
        ...prev,
        hasPassword: !!(sentPwd || prev.hasPassword),
        db: prev.db ? {
          ...prev.db,
          accounts: (prev.db.accounts || []).map((a) => ({
            ...a,
            hasPassword: !!(a.hasPassword || sentDbPwd),
          })),
        } : prev.db,
      } : prev)
      onSaved?.(name)
      window.dispatchEvent(new Event('tt-gp-envs'))
    } catch (e: any) {
      setSaveState('idle')
      toast.error('保存失败', { description: e?.message || String(e), duration: 0 })
      throw e
    }
  }

  // 所有自动保存串行：同一次输入风暴只留最后一次，且不会两个请求互相覆盖。
  const enqueueSave = () => {
    const run = saveChainRef.current
      .then(() => doSave())
      .catch(() => { /* doSave 已 toast；这里只让链条继续 */ })
    saveChainRef.current = run
    return run
  }

  // T100 同款：任何表单变化后 500ms 合并提交。新建/未填完主机账号时不自动保存，
  // 避免每 500ms 弹一次“缺少 SSH 主机/账号”。
  useEffect(() => {
    if (!canSave() || !isDirty()) return
    const t = window.setTimeout(() => { void enqueueSave() }, 500)
    return () => window.clearTimeout(t)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cur, pwd, dbPwd])

  const del = async () => {
    if (!cur) return
    // 草稿还没落盘，没有正式环境名；删除只移除本地草稿。
    if (create) {
      onDiscard?.()
      setCur(null)
      return
    }
    const targetName = selectedNameRef.current || cur.name.trim()
    if (!targetName) return
    if (!confirm('删除环境 ' + targetName + '？（不删本地字典库）')) return
    setBusy(true)
    try {
      const r = await g.call('/api/gp/env', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'delete', name: targetName }),
      })
      if (!r.ok) throw new Error(r.error || '删除失败')
      setCur(null)
      baselineRef.current = ''
      selectedNameRef.current = ''
      pwdBaselineRef.current = ''
      dbPwdBaselineRef.current = ''
      setPwd('')
      setDbPwd('')
      await g.load()
      window.dispatchEvent(new Event('tt-gp-envs'))
    } catch (e: any) {
      toast.error('删除失败', { description: e?.message || String(e), duration: 0 })
    } finally { setBusy(false) }
  }

  const dup = () => {
    if (!cur) return
    const next: GpEnv = {
      ...cur,
      name: (cur.name || '') + '-copy',
      password: '',
      db: cur.db ? {
        ...cur.db,
        accounts: (cur.db.accounts || []).map((a, i) => i === 0
          ? { ...a, password: '', hasPassword: false }
          : a),
      } : cur.db,
    }
    // 草稿复制：追加为另一条草稿（父级负责选中新草稿），不覆盖当前草稿。
    if (create && onCopyDraft) {
      onCopyDraft(next)
      return
    }
    setCur(next)
    // 复制出来的这一条先不自动保存：等用户改任意字段后才落盘，避免悄悄多一条。
    baselineRef.current = envSnapshot(next)
    selectedNameRef.current = ''
    pwdBaselineRef.current = ''
    dbPwdBaselineRef.current = ''
    setPwd(''); setDbPwd(''); setTest(null); setSaveState('idle')
    setMsg('已复制一份，密码需重新输入；修改任意字段后会自动保存')
  }

  // 三个页签各测各自的：ssh 只测 SSH、db 只测库、site 只测据点。
  // 后端按 what 分流（internal/gp/gp.go 的 hTest），结果也只回这一项。
  const runTest = async (what: 'ssh' | 'db' | 'site') => {
    if (!cur) return
    setTestBusy(what); setMsg('')
    try {
      const r = await g.call('/api/gp/test', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ env: payload(cur), what }),
      })
      if (!r.ok) { setMsg(r.error || '测试失败'); return }
      const res = (r.result || {})[what]
      setTest((prev) => ({ ...(prev || {}), [what]: res }))
    } catch (e) {
      setMsg('测试失败：' + String(e))
    } finally { setTestBusy('') }
  }

  // 「从服务器刷新」：连库读 azp_file（固定 ds schema）把据点**全表**取回来，
  // 摆成一张可点的表 —— 用户不知道代号对应哪家公司，让他挑比让他算要顺手。
  // 只读不写：选中某行后才走同一个补丁保存，写回 config.json 的 company/companyName。
  const refreshSites = async () => {
    if (!cur) return
    setSiteBusy(true); setSiteMsg('')
    try {
      const r = await g.call('/api/gp/site', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ env: cur.name }),
      })
      if (!r.ok) { setSiteMsg(r.error || '查询失败'); return }
      const rows: SiteRow[] = Array.isArray(r.sites) ? r.sites : []
      setSites(rows)
      // 刷新时间跟着配置走（下次打开页面不用再点一次就知道是什么时候抄的）。
      patch({ companySyncedAt: new Date().toISOString() })
      setSiteMsg(rows.length ? `已取到 ${rows.length} 个据点` : 'azp_file 里一条据点都没有')
    } catch (e) {
      setSiteMsg('查询失败：' + String(e))
    } finally { setSiteBusy(false) }
  }

  // 第一次切到「营运据点」页签时自动抄一次清单：空表让人以为"没东西可选"，
  // 而这一抄只是一次只读查询。用 ref 挡住后续切换，不反复连库。
  const sitesLoadedRef = useRef(false)
  useEffect(() => {
    if (tab !== 'site' || sitesLoadedRef.current) return
    if (!(curRef.current?.name || '').trim()) return
    sitesLoadedRef.current = true
    void refreshSites()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  if (!cur) return null
  const acc = cur.db?.accounts?.[0]
  // 当前选中的据点：空 = ds（后端同样的默认）。azp_file 里没有 ds 这种系统库，
  // 所以"选中值不在清单里"时补一行显示出来，别让人以为没选中。
  const curCode = (cur.company || '').trim().toLowerCase() || 'ds'
  const shownRows: SiteRow[] = sites.some((s) => s.code.toLowerCase() === curCode)
    ? sites
    : [{ code: curCode, name: cur.companyName || '' }, ...sites]

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mb-3 flex items-center gap-2">
        <span className="rounded-[3px] bg-primary/15 px-1.5 py-0.5 text-[10px] font-semibold text-primary">TOP GP</span>
        <span className="text-[11px] text-muted-foreground">TOP GP 没有 ENT；填实例名即可，$TOP / $CUST 由后台推导；营运据点在单独页签里选。</span>
        {saveState !== 'idle' && (
          <span className="ml-auto text-[11px] text-muted-foreground">{saveState === 'saving' ? '保存中…' : '已保存'}</span>
        )}
      </div>

      <Tabs value={tab} onValueChange={(v) => setTab(v as 'ssh' | 'db' | 'site')} className="flex min-h-0 flex-1 flex-col">
        <div className="flex items-center gap-2">
          <TabsList>
            <TabsTrigger value="ssh">SSH 服务器</TabsTrigger>
            <TabsTrigger value="db">数据库</TabsTrigger>
            <TabsTrigger value="site">营运据点</TabsTrigger>
          </TabsList>
          <span className="ml-auto flex items-center gap-2">
            <Button variant="ghost" size="sm" title={create ? '复制当前草稿（只改本地表单）' : '复制该环境（整份抄一份，名称自动加 -copy）'}
              disabled={busy} onClick={dup}>
              <Copy className="h-3.5 w-3.5" />复制
            </Button>
            <Button variant="ghost" size="sm" title={create ? '放弃当前草稿（不写配置）' : '删除该环境'} disabled={busy} onClick={() => void del()}>
              <Trash2 className="h-3.5 w-3.5" />删除
            </Button>
          </span>
        </div>

        <TabsContent value="ssh" forceMount className="mt-3 min-h-0 flex-1 overflow-auto data-[state=inactive]:hidden">
          <div className="grid grid-cols-2 gap-x-4 gap-y-3">
            <Field label="环境名称">
              <Input className={input} value={cur.name} placeholder="如 TOPTEST" autoFocus={create}
                onChange={(e) => patch({ name: e.target.value })} />
            </Field>
            <Field label="IP 主机">
              <Input className={input} value={cur.host || ''} placeholder="172.16.1.22"
                onChange={(e) => patch({ host: e.target.value })} />
            </Field>
            <Field label="SSH 端口">
              <Input className={input} type="number" value={cur.port ?? 22}
                onChange={(e) => patch({ port: Number(e.target.value) || 22 })} />
            </Field>
            <Field label="账号">
              <Input className={input} value={cur.user || ''} placeholder="tiptop"
                onChange={(e) => patch({ user: e.target.value })} />
            </Field>
            <Field label="密码">
              <div className="relative">
                <Input className={input} type={showPwd ? 'text' : 'password'} autoComplete="new-password" value={pwd}
                  placeholder="未设置"
                  onChange={(e) => setPwd(e.target.value)} />
                <button type="button" title={showPwd ? '隐藏密码' : '显示密码'}
                  onClick={() => setShowPwd((v) => !v)}
                  className="absolute top-1/2 right-1 -translate-y-1/2 p-0.5 text-muted-foreground hover:text-foreground">
                  {showPwd ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                </button>
              </div>
            </Field>
            <div />
            <Field label="实例名">
              <Input className={input} value={cur.instance || ''}
                placeholder="topprod / toptest / topstd"
                onChange={(e) => patch({ instance: e.target.value })} />
              <span className="mt-1 block text-[10px] leading-relaxed text-muted-foreground/80">
                {cur.instance?.trim()
                  ? `源码根 /u1/${cur.instance.trim()}/tiptop；客制根 /u1/${cur.instance.trim()}/topcust`
                  : '填实例名后，后台自动推导 $TOP / $CUST'}
              </span>
            </Field>
          </div>
          <div className="mt-4 space-y-1">
            <Button size="sm" variant="outline" disabled={testBusy !== ''} onClick={() => void runTest('ssh')}>
              {testBusy === 'ssh' ? '测试中…' : '测试 SSH 连接'}
            </Button>
            <TestLine label="SSH" res={test?.ssh} />
          </div>
        </TabsContent>

        <TabsContent value="db" forceMount className="mt-3 min-h-0 flex-1 overflow-auto data-[state=inactive]:hidden">
          <div className="grid grid-cols-2 gap-x-4 gap-y-3">
            <Field label="数据库主机">
              <Input className={input} value={cur.db?.host || ''} placeholder="172.16.1.22"
                onChange={(e) => patch({ db: { ...(cur.db || {}), host: e.target.value } })} />
            </Field>
            <Field label="端口">
              <Input className={input} type="number" value={cur.db?.port ?? 1521}
                onChange={(e) => patch({ db: { ...(cur.db || {}), port: Number(e.target.value) || 1521 } })} />
            </Field>
            <Field label="服务名(SERVICE_NAME)">
              <Input className={input} value={cur.db?.service || ''} placeholder="如 toptest"
                onChange={(e) => patch({ db: { ...(cur.db || {}), service: e.target.value } })} />
            </Field>
            <Field label="数据库账号">
              <Input className={input} value={acc?.account || ''} placeholder="ds"
                onChange={(e) => patch({ db: { ...(cur.db || {}), accounts: [{ account: e.target.value, password: dbPwd || undefined }] } })} />
            </Field>
            <Field label="数据库密码">
              <div className="relative">
                <Input className={input} type={showDbPwd ? 'text' : 'password'} autoComplete="new-password" value={dbPwd}
                  placeholder="未设置"
                  onChange={(e) => setDbPwd(e.target.value)} />
                <button type="button" title={showDbPwd ? '隐藏密码' : '显示密码'}
                  onClick={() => setShowDbPwd((v) => !v)}
                  className="absolute top-1/2 right-1 -translate-y-1/2 p-0.5 text-muted-foreground hover:text-foreground">
                  {showDbPwd ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
                </button>
              </div>
            </Field>
          </div>
          <div className="mt-4 space-y-1">
            <Button size="sm" variant="outline" disabled={testBusy !== ''} onClick={() => void runTest('db')}>
              {testBusy === 'db' ? '测试中…' : '测试数据库连接'}
            </Button>
            <TestLine label="数据库" res={test?.db} />
          </div>
        </TabsContent>

        <TabsContent value="site" forceMount className="mt-3 flex min-h-0 flex-1 flex-col gap-2 overflow-auto data-[state=inactive]:hidden">
          <div className="flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
            <span>
              上次同步：{cur.companySyncedAt ? new Date(cur.companySyncedAt).toLocaleString('zh-CN') : '从未同步'}
            </span>
            <Button size="sm" variant="outline" className="ml-auto h-7" disabled={siteBusy}
              title="连库读 azp_file（固定读 ds 这个系统库）把全部营运据点列出来；只读，不改服务器上的任何东西。"
              onClick={() => void refreshSites()}>
              <RefreshCw className={cn('h-3.5 w-3.5', siteBusy && 'animate-spin')} />
              {siteBusy ? '同步中…' : '从服务器刷新'}
            </Button>
          </div>
          {shownRows.length === 0 ? (
            <div className="py-3 text-xs leading-relaxed text-muted-foreground">
              还没有据点清单。点上面的「从服务器刷新」读一次 azp_file（需要能连上该环境的库），之后在这里点一行就选中。
            </div>
          ) : (
            <div className="min-h-0 flex-1 overflow-auto rounded-ui border border-border">
              <Table container={false}>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="w-44">营运据点</TableHead>
                    <TableHead>营运据点名称</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {shownRows.map((s) => {
                    const sel = s.code.toLowerCase() === curCode
                    const missing = !sites.some((x) => x.code.toLowerCase() === s.code.toLowerCase())
                    return (
                      <TableRow key={s.code}
                        onClick={() => patch({ company: s.code.toLowerCase(), companyName: s.name })}
                        className={cn('cursor-pointer', sel ? 'bg-accent/70 hover:bg-accent/70' : 'hover:bg-accent/40')}>
                        <TableCell className="font-mono text-xs">
                          <span className="flex items-center gap-1">
                            {sel && <Check className="h-3.5 w-3.5 text-primary" />}
                            {s.code.toLowerCase()}
                          </span>
                        </TableCell>
                        <TableCell className="text-xs">
                          {s.name || (
                            <span className="text-muted-foreground">
                              {missing ? '（不在 azp_file 里，ds 这类系统库就是这样）' : '—'}
                            </span>
                          )}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          )}
          {siteMsg && <p className="text-[11px] text-muted-foreground">{siteMsg}</p>}
          <div className="mt-auto space-y-1 pt-2">
            <Button size="sm" variant="outline" disabled={testBusy !== ''} onClick={() => void runTest('site')}>
              {testBusy === 'site' ? '测试中…' : '测试据点'}
            </Button>
            <TestLine label="据点" res={test?.site} />
          </div>
        </TabsContent>

        {msg && <div className="mt-2 text-xs text-muted-foreground">{msg}</div>}
      </Tabs>
    </div>
  )
}

/**
 * 这里原本还有一个 `GpDict()`（「查询数据源」「数据字典同步」「源码镜像」三张卡），
 * 但**从来没有被任何地方挂载**——SettingsView 只 import 了 `GpSites`，同步/镜像按钮是
 * SettingsView 自己实现的（`SettingsView.tsx:1939` 起），所以那三张卡在界面上不存在。
 * 2026-11-09 删除死代码；其中「查询数据源」按 docs/gp-scope.md §10 Q6 处理：等 §4.D
 * 字典层改造成 GP 自己的之后（本地镜像 / 在线直查才有语义），在 SettingsView 里新建并接线
 * （`GET /api/gp/meta` 回读 querySource + 保存端点 + `tt gp` 本地库查询）。
 */
