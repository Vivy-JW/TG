// 渲染期异常的**最后一道网**。
//
// 为什么需要它：前端有一批"把后端/远端文本解析成界面结构"的地方（变量树、大纲、报文渲染…），
// 解析器出一次未捕获异常，React 会卸载整棵根 —— 用户看到的是**整页白屏**，而且没有
// 任何可读的原因（实测：一个两万层嵌套的变量值就能做到，2026-10-15）。
//
// 它不替代把 bug 修掉，只保证"一处坏掉"不连带把整个工作台带走：
//   - 根上包一层（main.tsx）：兜住任何漏出去的异常，给一屏能读的说明 + 「重试」；
//   - 每个视图再各包一层（App.tsx 的 ViewPane）：崩的是某一个视图时，会话栏、工具条、
//     右侧面板都还能用，切到别的视图也不受影响。
//
// 「重试」= 清掉错误状态重新挂载子树。它救不了必崩的输入（同一个值再渲染一次还是崩），
// 所以界面上同时把**报错原文**摆出来 —— 那是拿去修的东西，不是让人干瞪眼。
import { Component, type ErrorInfo, type ReactNode } from 'react'

interface Props {
  children: ReactNode
  /** 出错时显示的标题（按位置不同写清"哪一块"坏了）。 */
  what?: string
}

interface State {
  err: Error | null
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { err: null }

  static getDerivedStateFromError(err: Error): State {
    return { err }
  }

  componentDidCatch(err: Error, info: ErrorInfo) {
    // 控制台留一份完整的：界面那块只说一句，细节（含组件栈）给能看日志的人。
    console.error('[tt] 界面渲染出错：', err, info.componentStack)
  }

  render() {
    const { err } = this.state
    if (!err) return this.props.children
    return (
      <div className="flex h-full min-h-0 flex-col items-center justify-center gap-2 overflow-auto p-6 text-xs">
        <div className="text-sm font-medium text-red-600 dark:text-red-400">
          {this.props.what || '这块界面'}渲染出错了
        </div>
        <div className="max-w-2xl text-muted-foreground">
          一处坏掉不该连带把整个工作台带走 —— 别的视图与面板仍然可用。这一处的报错原文：
        </div>
        <pre className="max-h-48 w-full max-w-2xl overflow-auto rounded-ui border border-border bg-card p-2 font-mono text-[11px] text-foreground">
          {String(err?.stack || err?.message || err)}
        </pre>
        <button
          onClick={() => this.setState({ err: null })}
          title="清掉错误状态并重新挂载这块界面（必崩的输入会再崩一次，报错原文就是拿去修的证据）"
          className="rounded-ui border border-border px-2 py-1 text-xs text-muted-foreground hover:bg-accent hover:text-foreground"
        >
          重试
        </button>
      </div>
    )
  }
}
