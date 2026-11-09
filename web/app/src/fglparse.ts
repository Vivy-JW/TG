// fgldb print 输出的 RECORD/ARRAY 文本解析成树节点(PTY 模式用):
//   {{a = "x", b = (null), sub = {{c = 1}}}}   → record 字段树
//   { {1, 2}, {3, 4} }                          → 数组元素树([1] [2] …)
// fgldb 的 record 外层是双大括号,靠"匿名包装层塌缩"统一处理;
// 引号内的逗号/大括号不参与结构切分。解析只做展示层折叠,不改原文本语义。

// 树节点(纯前端结构:children 已在内存,无引用/无后端下钻)
export interface TNode {
  name: string
  value?: string
  type?: string
  children?: TNode[]
  open: boolean
  loading?: boolean
}

// 按顶层逗号切分(引号内、嵌套 {} () 内的逗号不算)
function splitTop(content: string): string[] {
  const parts: string[] = []
  let depth = 0
  let inQuote = false
  let cur = ''
  for (let i = 0; i < content.length; i++) {
    const ch = content[i]
    if (inQuote) {
      cur += ch
      if (ch === '\\') { cur += content[i + 1] ?? ''; i++ }
      else if (ch === '"') inQuote = false
      continue
    }
    if (ch === '"') { inQuote = true; cur += ch; continue }
    if (ch === '{' || ch === '(') depth++
    else if (ch === '}' || ch === ')') depth--
    if (ch === ',' && depth === 0) { parts.push(cur); cur = '' }
    else cur += ch
  }
  if (cur.trim()) parts.push(cur)
  return parts
}

// 解析深度上限。
//
// 为什么必须有：`parseGroup ⇄ parseValue` 是**递归**的，而值文本来自 fgldb 的输出 ——
// 一个畸形的（或恶意的）值 `'{'.repeat(20000)` 就能把调用栈打爆成 RangeError。
// 实测这个异常发生在渲染期，于是没有错误边界时**整个工作台白屏**（2026-10-15）。
// 真实 RECORD/ARRAY 的嵌套很少超过几层，32 已经宽得离谱；到顶之后按**文本**展示剩下的部分 ——
// 宁可少折叠一层，也不要让一个值把界面打死。
// （姊妹模块 fgloutline.ts 的做法更彻底：那边一律用显式栈，压根不递归。）
const MAX_PARSE_DEPTH = 32

// 解析一段值文本:大括号结构 → 子树,否则原样作为叶子
function parseValue(text: string, depth: number): { value?: string; children?: TNode[] } {
  const s = text.trim()
  if (depth < MAX_PARSE_DEPTH && s.startsWith('{') && s.endsWith('}')) {
    const kids = parseGroup(s.slice(1, -1), depth + 1)
    if (kids.length > 0) return { children: kids }
    return { value: '{}' }
  }
  return { value: s }
}

const fieldRe = /^([A-Za-z_][\w.]*)\s*=\s*([\s\S]*)$/

// 解析 {} 内的内容:全是 name=value → record 字段;否则按数组元素编号
function parseGroup(content: string, depth: number): TNode[] {
  const parts = splitTop(content)
  if (parts.length === 0) return []
  const trimmed = parts.map((p) => p.trim())
  const allFields = trimmed.every((p) => fieldRe.test(p))
  if (!allFields) {
    // 单一匿名大括号层 = record/数组的包装,塌缩后重析
    if (trimmed.length === 1 && depth < MAX_PARSE_DEPTH) {
      const t = trimmed[0]
      if (t.startsWith('{') && t.endsWith('}')) return parseGroup(t.slice(1, -1), depth + 1)
    }
    return trimmed.map((t, i) => {
      const v = parseValue(t, depth)
      const kids = v.children
      return {
        name: `[${i + 1}]`,
        value: kids ? undefined : v.value,
        open: false,
        children: kids,
        type: kids ? (kids.some((k) => !k.name.startsWith('[')) ? 'RECORD' : `ARRAY[${kids.length}]`) : undefined,
      }
    })
  }
  return trimmed.map((t) => {
    const m = t.match(fieldRe)!
    const v = parseValue(m[2], depth)
    const kids = v.children
    return {
      name: m[1],
      value: kids ? undefined : v.value,
      open: false,
      children: kids,
      type: kids ? (kids.some((k) => !k.name.startsWith('[')) ? 'RECORD' : `ARRAY[${kids.length}]`) : undefined,
    }
  })
}

// 入口:文本是 RECORD/ARRAY 结构时返回子节点树,否则 null(纯标量原样展示)
export function parseFglTree(text: string): TNode[] | null {
  const s = (text ?? '').trim()
  if (!s.startsWith('{') || !s.endsWith('}')) return null
  const kids = parseGroup(s.slice(1, -1), 1)
  if (kids.length === 0) return null
  return kids
}
