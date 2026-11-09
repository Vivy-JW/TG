// 纯前端变量树渲染:children 已在内存(如 fgldb print 文本解析出的 RECORD/ARRAY),
// 展开收起只是 UI 折叠,无后端交互。样式与 dap 分支的变量树一致。
import { useState } from 'react'
import type { TNode } from './fglparse'

// 一层节点一次最多渲染多少行。
//
// 为什么要有上限:一个 ARRAY 有 3000 个子节点时,展开它就直接铺出 3001 行 DOM
// (实测 ~450ms 且滚动发涩)。超出的部分**不是把数据藏起来**,而是先不渲染 + 一个
// 「再显示 N 项」—— 点一下多铺一屏,用户想看多久都行,DOM 不会一次爆掉。
const PAGE = 200

function NodeRow({ node, depth, toggle }: { node: TNode; depth: number; toggle: (n: TNode) => void }) {
  const expandable = !!node.children?.length
  const [shown, setShown] = useState(PAGE)
  const total = node.children?.length ?? 0
  return (
    <>
      <div
        className={`flex items-baseline gap-1 px-1 hover:bg-foreground/5 ${expandable ? 'cursor-pointer' : 'cursor-default'}`}
        style={{ paddingLeft: depth * 14 + 4 }}
        onClick={() => expandable && toggle(node)}
      >
        <span className={`w-3 shrink-0 text-muted-foreground ${expandable ? '' : 'opacity-0'}`}>
          {node.loading ? '…' : node.open ? '▾' : '▸'}
        </span>
        <span className="text-sky-700 dark:text-sky-300 whitespace-pre">{node.name}</span>
        {node.value !== undefined && (
          <>
            <span className="text-muted-foreground whitespace-pre"> = </span>
            <span className="text-foreground break-all">{node.value || '(空)'}</span>
          </>
        )}
        {node.type && <span className="ml-1 shrink-0 text-muted-foreground">{node.type}</span>}
      </div>
      {node.open && node.children?.slice(0, shown).map((c, j) => (
        <NodeRow key={j} node={c} depth={depth + 1} toggle={toggle} />
      ))}
      {node.open && total > shown && (
        <button
          className="block text-muted-foreground hover:text-foreground"
          style={{ paddingLeft: (depth + 1) * 14 + 4 }}
          onClick={(e) => { e.stopPropagation(); setShown((n) => n + PAGE) }}
          title={`这一层共 ${total} 项，已铺 ${shown} 项 —— 一次全铺出来会让 DOM 卡住`}
        >
          再显示 {Math.min(PAGE, total - shown)} 项（共 {total} 项）
        </button>
      )}
    </>
  )
}

// 展示一组已构建好的节点(顶层各自带展开状态)
export function VarTreeNodes({ nodes }: { nodes: TNode[] }) {
  const [, bump] = useState(0)
  const toggle = (n: TNode) => {
    if (!n.children?.length) return
    n.open = !n.open
    bump((v) => v + 1)
  }
  return (
    <div className="text-xs overflow-auto">
      {nodes.map((n, i) => (
        <NodeRow key={i} node={n} depth={0} toggle={toggle} />
      ))}
    </div>
  )
}
