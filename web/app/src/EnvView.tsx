// 环境页:一个环境对应的工作面板。
//
// 三个 Tab:创建会话 / 接口日志 / 接口调试。
// 其中两个是**已有的画面**,原样搬进来;「创建会话」是外壳自己的那一屏 ——
// 它已经从"只写本机"变成**真的向后端建会话**(见 store.createSession)。
import { useState, type ReactNode } from 'react'
import { Bug, Plus } from 'lucide-react'
import { Button, Input } from '../../shared/ui'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../shared/ui-radix'
import { useStore } from './store'
import type { SessionDraft, EnvBrief, SessionItem } from './store'
import type { EnvTab } from './routing'
import { Pane, TabStrip, ViewHeader, Meta, type TabItem } from './Shell'
import { WsLogView } from './WsLogView'

const ENV_TABS: TabItem<EnvTab>[] = [
  { key: 'create', label: '创建会话', hint: '在这个环境下面新建一条会话' },
  { key: 'wslogs', label: '接口日志', hint: '看这个环境的接口调用报文' },
]

export function EnvView({ env, tab, sessions, onTab, onCreate, leading }: {
  env: EnvBrief
  tab: EnvTab
  sessions: SessionItem[]
  onTab: (t: EnvTab) => void
  onCreate: (form: SessionDraft, onStep: (step: string) => void) => Promise<void>
  leading?: ReactNode
}) {
  // 名字里的"环境"这一层信息:宿主与登录区域是设置里配的,壳子只照实显示,不改。
  const meta = (
    <>
      <Meta title="环境(设置 → 站点管理)">{env.host ? `${env.host}` : '未配置服务器地址'}</Meta>
      {env.zone && <><span className="opacity-30">·</span><Meta title="登录区域">区域 {env.zone}</Meta></>}
      <span className="opacity-30">·</span>
      <Meta title="该环境下的会话数">{sessions.length} 个会话</Meta>
    </>
  )

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col bg-background">
      <ViewHeader title={env.name} meta={meta} leading={leading} />
      <TabStrip items={ENV_TABS} active={tab} onChange={onTab} />

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <Pane show={tab === 'create'} label="创建会话">
          <CreateSessionPane env={env} count={sessions.length} onCreate={onCreate} />
        </Pane>
        <Pane show={tab === 'wslogs'} label="接口日志"><WsLogView /></Pane>
      </div>
    </div>
  )
}

/**
 * 创建会话：模块（可留空，按 p_link 解析）+ 作业编号（可留空）。
 *
 * GP 没有企业编号（ENT）：一个库一个登录账号（db.accounts[0]），查哪个 schema 由
 * 「营运据点」决定，跟建会话无关 —— 所以这一屏只有两个栏位。
 * 作业编号是"这条会话要调哪个作业"：**填了就建完直接起作业**（停在入口），
 * **留空只建空闲会话**（之后在会话页右侧「运行」里再起）。
 */
function CreateSessionPane({ env, count, onCreate }: {
  env: EnvBrief
  count: number
  onCreate: (form: SessionDraft, onStep: (step: string) => void) => Promise<void>
}) {
  const [name, setName] = useState('')
  // 模块可留空：后端按 p_link 的多行链接自动解析（优先 linked_prog == 程序名）。
  const [module, setModule] = useState('')
  const [prog, setProg] = useState('')
  const [busy, setBusy] = useState(false)
  const [step, setStep] = useState('')
  const [err, setErr] = useState('')
  // 默认名跟着站点走：不同站点各建一条时，"新会话 1"会互相撞（别名在服务端是全局唯一的）。
  const placeholder = `${env.name} ${count + 1}`

  const submit = async () => {
    if (busy) return
    // 作业编号**不是必填**:填了就建完直接起作业,留空就只建空闲会话(见 store.createSession)。
    setErr('')
    setBusy(true)
    setStep('')
    try {
      await onCreate({
        ent: '',
        module: module.trim() || undefined,
        prog: prog.trim(),
        name: name.trim(),
      }, setStep)
      // 成功后这条会话已经跳走了,但本面板是 keep-alive 的 —— 不清空的话下次回来
      // 还留着上一轮的输入,再点一次就会建出一条一模一样的会话。
      setName(''); setProg(''); setModule('')
    } catch (e: any) {
      // 建会话的失败原因必须原样显示(服务端信封里的那句话):是环境不存在、
      // 还是连不上,处置完全不同。
      setErr(e?.message || String(e))
    } finally {
      setBusy(false)
      setStep('')
    }
  }

  return (
    <div className="min-h-0 flex-1 overflow-auto p-6">
      <div className="mx-auto w-full max-w-[580px]">
        <h2 className="text-[17px] font-semibold tracking-tight text-foreground">在「{env.name}」下创建会话</h2>
        <p className="mt-1 text-xs text-muted-foreground">
          会话会挂在左侧这棵树的「{env.name}」下面。建完自动跳到它的 Debug 页。
        </p>

        <div className="mt-5 space-y-4 rounded-[12px] border border-border bg-card p-4">
          <label className="block">
            <span className="mb-1.5 block text-xs text-muted-foreground">会话名称</span>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder={placeholder}
              disabled={busy}
              onKeyDown={(e) => { if (e.key === 'Enter') void submit() }}
              className="h-8 text-[13px]" />
            <span className="mt-1 block text-[11px] text-muted-foreground/80">
              留空就用「{placeholder}」。每条会话都会有个名字，便于在树里认。
            </span>
          </label>

          <div className="grid grid-cols-2 gap-3">
            <label className="block">
              <span className="mb-1.5 block text-xs text-muted-foreground">
                模块 <span className="text-muted-foreground/70">可留空，按 p_link 自动解析</span>
              </span>
              <Input value={module} onChange={(e) => setModule(e.target.value)}
                placeholder="留空 = p_link 自动解析"
                disabled={busy}
                onKeyDown={(e) => { if (e.key === 'Enter') void submit() }}
                className="h-8 text-[13px]" />
            </label>
            <label className="block">
              <span className="mb-1.5 block text-xs text-muted-foreground">
                作业编号 <span className="text-muted-foreground/70">可留空</span>
              </span>
              <Input value={prog} onChange={(e) => setProg(e.target.value)}
                placeholder="如 csfi300_icd（留空 = 只建会话）"
                disabled={busy}
                onKeyDown={(e) => { if (e.key === 'Enter') void submit() }}
                className="h-8 text-[13px]" />
            </label>
          </div>
          <p className="-mt-2 text-[11px] text-muted-foreground/80">
            模块可留空，系统会按 p_link 的入口程序自动解析；作业编号填了就在建好的会话里直接起作业，留空只建空闲会话（之后在右侧「运行」里再起）。
          </p>

          <div className="flex items-start gap-2 rounded-[8px] border border-border bg-muted/30 px-3 py-2 text-[11px] leading-relaxed text-muted-foreground">
            <Bug className="mt-0.5 h-3.5 w-3.5 shrink-0 opacity-70" />
            <span>建出来的是<b className="font-medium text-foreground">调试会话</b>：填了作业编号就直接起作业、停在入口，可以下断点、单步、看变量。</span>
          </div>

          {err && (
            <div className="rounded-[8px] border border-red-500/30 bg-red-500/10 px-3 py-2 text-[11px] leading-relaxed text-red-600 dark:text-red-400">
              {err}
            </div>
          )}

          <div className="flex flex-wrap items-center gap-2 pt-1">
            <Button size="sm" onClick={() => void submit()} disabled={busy}>
              <Plus className="h-3.5 w-3.5" />
              {busy ? '创建中…' : '创建会话'}
            </Button>
          </div>
        </div>

      </div>
    </div>
  )
}
