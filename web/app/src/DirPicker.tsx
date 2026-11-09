// 目录选择器（前端自绘）。
//
// 为什么不直接用浏览器的 <input webkitdirectory> / showDirectoryPicker()：
// 这两种都**拿不到本地绝对路径**（只给文件名或 FileSystemDirectoryHandle），
// 而这里要的恰恰是 "D:\liuwpa\..." 这种能写进配置的路径。
// 所以路径数据由服务端 GET /api/fs/list 列出（只列目录），这里只负责画。
//
// 相对原来"服务端弹 Windows 文件夹对话框"的好处：没有再起一个 PowerShell 进程，
// 打开是几十毫秒；外观、字体、深浅色都跟设置页一致。
import { useEffect, useRef, useState } from 'react'
import { ChevronUp, Folder, RefreshCw } from 'lucide-react'
import { api, type FsList } from './api'
import { Button, Input } from '../../shared/ui'
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from '../../shared/ui-radix'
import { cn } from '../../shared/utils'

export function DirPicker({ open, initial, title, onPick, onClose }: {
  open: boolean
  /** 打开时定位到的目录（可为空 = 只列盘符） */
  initial: string
  title?: string
  onPick: (path: string) => void
  onClose: () => void
}) {
  const [data, setData] = useState<FsList | null>(null)
  const [path, setPath] = useState('')
  const [input, setInput] = useState('')
  const [err, setErr] = useState('')
  const [loading, setLoading] = useState(false)
  // 连续点目录时前一个响应可能后到，用自增 id 丢掉过期结果，免得列表跳回上一级。
  const reqId = useRef(0)

  const load = async (p: string) => {
    const id = ++reqId.current
    setLoading(true)
    setErr('')
    try {
      const r = await api.listDirs(p)
      if (id !== reqId.current) return
      setData(r)
      setPath(r.path || '')
      setInput(r.path || '')
    } catch (e: any) {
      if (id !== reqId.current) return
      setErr(e?.message || String(e))
    } finally {
      if (id === reqId.current) setLoading(false)
    }
  }

  useEffect(() => {
    if (!open) return
    setData(null)
    setErr('')
    setPath('')
    void load(initial || '')
    // initial 变化（换了一行/换了一个按钮）就重新定位
  }, [open, initial])

  const drive = (path || input).slice(0, 2).toUpperCase()

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose() }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title || '选择目录'}</DialogTitle>
          <DialogDescription>
            浏览本机目录；单击进入下一级，「选择此目录」确定。
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-1">
            {data?.drives?.map((d) => (
              <Button key={d.path} variant={drive.startsWith(d.name.toUpperCase()) ? 'default' : 'outline'}
                size="sm" className="h-7 px-2 text-[11px]"
                onClick={() => void load(d.path)}>
                {d.name}
              </Button>
            ))}
          </div>

          <div className="flex items-center gap-1">
            <Button variant="outline" size="sm" className="h-8 shrink-0 px-2"
              disabled={!data?.parent} title={data?.parent || '已经是根目录'}
              onClick={() => void load(data?.parent || '')}>
              <ChevronUp className="h-3.5 w-3.5" />
            </Button>
            <Input className="h-8 font-mono text-[11px]" value={input}
              placeholder="输入路径后回车，如 D:\t100"
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); void load(input) } }} />
            <Button variant="ghost" size="sm" className="h-8 shrink-0 px-2" title="重新读取"
              onClick={() => void load(input)}>
              <RefreshCw className={cn('h-3.5 w-3.5', loading && 'animate-spin')} />
            </Button>
          </div>

          <div className="h-72 overflow-auto rounded-ui border border-border">
            {err ? (
              <div className="px-3 py-2 text-[11px] text-amber-600 dark:text-amber-400">{err}</div>
            ) : !data ? (
              <div className="px-3 py-2 text-[11px] text-muted-foreground">读取中…</div>
            ) : (data.dirs ?? []).length === 0 ? (
              <div className="px-3 py-2 text-[11px] text-muted-foreground">
                {data.path ? '这个目录下没有子目录。' : '先选一个盘符。'}
              </div>
            ) : (
              (data.dirs ?? []).map((d) => (
                <button key={d.path} type="button"
                  className="flex w-full items-center gap-2 px-2 py-1 text-left text-[12px] hover:bg-accent"
                  title={d.path}
                  onClick={() => void load(d.path)}>
                  <Folder className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                  <span className="truncate">{d.name}</span>
                </button>
              ))
            )}
            {data?.truncated && (
              <div className="px-2 py-1 text-[11px] text-muted-foreground">
                目录太多，只列了前 800 个；可直接在上面输入完整路径。
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <div className="mr-auto min-w-0 flex-1 truncate font-mono text-[11px] text-muted-foreground"
            title={path || ''}>
            {path || '未选择'}
          </div>
          <Button variant="outline" onClick={onClose}>取消</Button>
          <Button disabled={!path} onClick={() => { onPick(path); onClose() }}>选择此目录</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
