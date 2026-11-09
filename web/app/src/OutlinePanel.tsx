// 大纲面板:解析当前显示的 4GL 源码为块树(顶层 FUNCTION/MAIN/REPORT,其下 DIALOG/MENU/
// INPUT/DISPLAY ARRAY/CONSTRUCT,再下 BEFORE/AFTER/ON… 子块;解析逻辑移植自 BDL,见 fgloutline.ts),
// 树形展示并点击跳行(落到名称列)。跳转直接操作编辑器视口(setPosition + revealLineInCenterIfOutsideViewport),
// 不走 revealReq——不受"仅停站可定位"的调试门限制,运行中也可浏览跳转,且不产生任何调试信号。
// 调试页行号需补偿 lineOffset(Monaco 顶部前插空行对齐 DVM 行号)。
// 高亮跟随编辑器光标:定位光标所在的最深层节点;若其上级被折叠,则高亮可见的那个祖先节点,
// 展开后再落到具体层级(不自动展开)。
import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react'
import { ChevronsDownUp, ChevronsUpDown } from 'lucide-react'
import { useStore } from './store'
import { editorRef } from './SourceView'
import { parseOutline, type OutlineNode } from './fgloutline'
import { PanelContent, PanelHeader, PanelItem } from './PanelKit'

// 节点 key 由 path+行+label 决定,行渲染、折叠集合、全量收集共用同一算法
// (只用到行号与标签,故不绑定完整 OutlineNode —— 索引层是它的子集)
const nodeKey = (node: { label: string; line: number }, path: string) => `${path}/${node.line}:${node.label}`

// 带文档序索引的节点:end = 解析器给出的真实块结束行(含),不再靠「先序下一个节点行号-1」反推
// —— 后者在末尾节点与块之间穿插的普通语句处都不准。
interface IndexedNode {
  label: string
  line: number
  end: number
  selStart: number // 名称列(0-based),点击落点
  children?: IndexedNode[]
}

function indexTree(roots: OutlineNode[], totalLines: number): IndexedNode[] {
  return roots.map((n) => ({
    label: n.label,
    line: n.line,
    end: Math.min(n.endLine, totalLines),
    selStart: n.selStart,
    children: n.children ? indexTree(n.children, totalLines) : undefined,
  }))
}

// 光标行 → 从根到最深包含节点的链(区间互斥且按行有序,逐层线性扫描即可)
const findChain = (nodes: IndexedNode[], line: number, path: string, acc: { node: IndexedNode; key: string }[]) => {
  for (const n of nodes) {
    if (line < n.line) break
    if (line <= n.end) {
      const key = nodeKey(n, path)
      acc.push({ node: n, key })
      if (n.children) findChain(n.children, line, key, acc)
      return
    }
  }
}

const collectKeys = (nodes: IndexedNode[], path: string, acc: string[]) => {
  for (const n of nodes) {
    const key = nodeKey(n, path)
    if (n.children && n.children.length) {
      acc.push(key)
      collectKeys(n.children, key, acc)
    }
  }
}

// 大纲在右侧那条手风琴里是**一个 item**（不再是自成一块、靠自己那行标题行撑着的面板）。
// 用户 2026-10-16 的定案：两块面合并成一条手风琴，大纲也有自己的折叠 chevron，
// 谁占多少高度交给「内容自然高度 + 空间不够时一起收缩」，不再硬切一半。
// 因此这里同时给出 item 的标题行与内容体：折叠状态是这一份本地 state，必须两者同住。
export function OutlineItem() {
  const sourceContent = useStore((s) => s.sourceContent)
  const lineOffset = useStore((s) => s.lineOffset)
  const tabs = useStore((s) => s.tabs)
  const activeTab = useStore((s) => s.activeTab)
  const active = tabs.find((t) => t.key === activeTab)
  const isDebug = !active
  const content = isDebug ? (sourceContent || '') : (active!.content || '')
  const offset = isDebug ? lineOffset : 0
  // 行数与解析器、Monaco 用同一套行模型(含孤立 \r 也算断行)
  const totalLines = useMemo(() => content.split(/\r\n|\r|\n/).length, [content])
  const rooted = useMemo(() => indexTree(parseOutline(content), totalLines), [content, totalLines])
  // 折叠集合:默认全展开;key 与渲染行共用 nodeKey 算法
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set())
  // 换了内容(切到另一个页签 / 这份源码被重新读进来)就把折叠集合清空。
  //
  // 为什么必须清:nodeKey 里带**行号与标签**,内容一变旧 key 一个都对不上 —— 那时
  // collapsed 非空(所以按钮显示"全部展开")却没有一行真的折叠着,第一次点它把集合清空
  // 也就没有任何可见变化。实测表现是"换文件后按钮文案与实际相反、点一下没反应"。
  // 按内容本身比,而不是按引用:同一份内容重算树不该把用户的折叠态清掉。
  const lastContentRef = useRef('')
  useEffect(() => {
    if (lastContentRef.current === content) return
    lastContentRef.current = content
    setCollapsed(new Set())
  }, [content])
  const allKeys = useMemo(() => {
    const acc: string[] = []
    collectKeys(rooted, '', acc)
    return acc
  }, [rooted])
  const allExpanded = collapsed.size === 0
  const toggleAll = () => setCollapsed(allExpanded ? new Set(allKeys) : new Set())

  // 光标行跟踪:编辑器可能在面板挂载后才就绪,轮询挂接一次
  const [cursorLine, setCursorLine] = useState(0)
  useEffect(() => {
    const disp: { dispose(): void }[] = []
    let timer: number | undefined
    const attach = () => {
      const ed = editorRef.current
      if (!ed) return false
      const sync = () => {
        const p = ed.getPosition()
        setCursorLine(p ? p.lineNumber : 0)
      }
      disp.push(ed.onDidChangeCursorPosition(sync))
      disp.push(ed.onDidChangeModel(sync))
      sync()
      return true
    }
    if (!attach()) timer = window.setInterval(() => { if (attach()) window.clearInterval(timer) }, 200)
    return () => {
      if (timer) window.clearInterval(timer)
      disp.forEach((d) => d.dispose())
    }
  }, [])

  // 光标 → 源码行 → 节点链 → 可见高亮节点(沿链向下,遇到折叠的祖先即停)
  const cursorSrc = cursorLine - offset
  const visibleKey = useMemo(() => {
    if (cursorSrc <= 0) return ''
    const chain: { node: IndexedNode; key: string }[] = []
    findChain(rooted, cursorSrc, '', chain)
    if (!chain.length) return ''
    let key = chain[0].key
    for (let i = 1; i < chain.length; i++) {
      if (collapsed.has(chain[i - 1].key)) break
      key = chain[i].key
    }
    return key
  }, [rooted, cursorSrc, collapsed])

  // 高亮行滚动进可视区(不抢列表滚动位置)
  const hlRef = useRef<HTMLDivElement | null>(null)
  useEffect(() => {
    hlRef.current?.scrollIntoView({ block: 'nearest' })
  }, [visibleKey])

  const jump = (line: number, col: number) => {
    const ed = editorRef.current
    if (!ed) return
    const L = line + offset
    // 落在名称列上(selectionRange 语义),而不是一律行首
    ed.setPosition({ lineNumber: L, column: col + 1 })
    ed.revealLineInCenterIfOutsideViewport(L)
  }
  const toggleKey = (key: string) => {
    setCollapsed((prev) => {
      const n = new Set(prev)
      if (n.has(key)) n.delete(key)
      else n.add(key)
      return n
    })
  }

  const row = (node: IndexedNode, depth: number, path: string): ReactNode => {
    const key = nodeKey(node, path)
    const kids = node.children
    const hasKids = !!kids && kids.length > 0
    const isCollapsed = collapsed.has(key)
    const hl = key === visibleKey
    return (
      <div key={key}>
        <div ref={hl ? hlRef : undefined} onClick={() => jump(node.line, node.selStart)}
          className={`flex h-6 cursor-pointer items-center gap-1 pr-2 text-xs ${
            hl ? 'bg-accent/70 text-foreground' : 'hover:bg-accent/40'
          } ${hl ? '' : depth === 0 ? 'font-medium text-foreground' : depth === 1 ? 'text-sky-600 dark:text-sky-400' : 'text-muted-foreground'}`}
          style={{ paddingLeft: depth * 14 + 4 }}
          title={`${node.label} (第 ${node.line} 行)`}>
          {hasKids ? (
            <button className="w-3 shrink-0 text-muted-foreground"
              title={isCollapsed ? '展开' : '收起'}
              onClick={(e) => { e.stopPropagation(); toggleKey(key) }}>
              {isCollapsed ? '▸' : '▾'}
            </button>
          ) : (
            <span className="w-3 shrink-0" />
          )}
          <span className="min-w-0 truncate">{node.label}</span>
        </div>
        {hasKids && !isCollapsed && kids!.map((k) => row(k, depth + 1, key))}
      </div>
    )
  }

  return (
    <PanelItem value="outline">
      {/* 「全部折叠 / 全部展开」放在标题行的**左图标位**：这里原来是各面板的"自动"开关，
          同一个位置同一套观感；塞进右侧触发区会变成"按钮套按钮"（Radix 的 trigger 本身就是 button）。
          右边那条计数文字（"N 个顶层节点"）已按用户要求删掉 —— 树的形状自己就说明了它有多少。 */}
      <PanelHeader leading={
        <button title={allExpanded ? '全部折叠' : '全部展开'} onClick={toggleAll}
          disabled={allKeys.length === 0}
          className="flex w-6 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground disabled:pointer-events-none disabled:opacity-40">
          {allExpanded ? <ChevronsDownUp className="h-3.5 w-3.5" /> : <ChevronsUpDown className="h-3.5 w-3.5" />}
        </button>
      }>
        大纲
      </PanelHeader>
      <PanelContent className="py-1">
        {rooted.length === 0 && (
          <div className="p-2 text-xs text-muted-foreground">当前源码无可识别的大纲节点</div>
        )}
        {rooted.map((n) => row(n, 0, ''))}
      </PanelContent>
    </PanelItem>
  )
}
