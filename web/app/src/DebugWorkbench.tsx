// 调试工作台:原来整块「调试」视图的内容,搬进会话页的 Debug Tab。
//
// 与旧外壳的差别只有一处:**不再有左侧会话栏**(选环境/开会话那件事已经归左侧树),
// 所以这里只剩「页签行 + 编辑器 | 右侧栏 + 下方面板」这三块。
// 其余的分割线拖拽、面板显隐、编辑器补一次 layout 的行为原样保留 —— 它们都是踩出来的。
import { useEffect, useState } from 'react'
import { useStore } from './store'
import { Toolbar } from './Toolbar'
import { SourceView, editorRef } from './SourceView'
import { RightPanels, TimelinePanel } from './Panels'
import { DebugHeader } from './DebugHeader'
import { ErrorBoundary } from './ErrorBoundary'

export function DebugWorkbench({ active }: { active: boolean }) {
  const rightDebugOpen = useStore((s) => s.rightDebugOpen)
  const rightTreeOpen = useStore((s) => s.rightTreeOpen)
  const showRight = rightDebugOpen || rightTreeOpen
  const showBottom = useStore((s) => s.showBottom)
  const bottomMax = useStore((s) => s.bottomMax)
  const panelW = useStore((s) => s.panelW)
  const setPanelW = useStore((s) => s.setPanelW)

  // 下方面板(操作时间线/协议流)高度拖拽(记忆到 localStorage)
  const [bottomH, setBottomH] = useState(() => {
    const v = Number(localStorage.getItem('tt.bottomH'))
    return v >= 120 && v <= 600 ? v : 192
  })

  // keep-alive 下 Monaco 容器在 display:none 期间 ResizeObserver 会拿到 0 尺寸;
  // 切回 Debug Tab 后补一次 layout(),确保渲染与视口不残留错位。
  useEffect(() => {
    if (!active) return
    const t = setTimeout(() => editorRef.current?.layout(), 60)
    return () => clearTimeout(t)
  }, [active])

  const onResizeDown = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault()
    const el = e.currentTarget
    el.classList.add('dragging')
    const startX = e.clientX
    const startW = panelW
    let latest = startW
    const move = (ev: MouseEvent) => {
      latest = Math.min(640, Math.max(240, startW - (ev.clientX - startX)))
      setPanelW(latest)
    }
    const up = () => {
      el.classList.remove('dragging')
      localStorage.setItem('tt.panelW', String(latest))
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
      // 兜住「收不到 mouseup」:鼠标在窗口外松开、或拖拽中窗口失焦 —— 否则这条 mousemove
      // 会一直挂着,面板跟着光标跑,直到用户再点一下(读码结论;难复现,但后果难看)。
      window.removeEventListener('blur', up)
      document.removeEventListener('mouseleave', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
    window.addEventListener('blur', up)
    document.addEventListener('mouseleave', up)
  }

  const onBottomResizeDown = (e: React.MouseEvent<HTMLDivElement>) => {
    e.preventDefault()
    const el = e.currentTarget
    el.classList.add('dragging')
    const startY = e.clientY
    const startH = bottomH
    let latest = startH
    const move = (ev: MouseEvent) => {
      // 分隔线跟手:向上拖 = 面板变高(与 VS Code 底部面板一致)
      latest = Math.min(600, Math.max(120, startH - (ev.clientY - startY)))
      setBottomH(latest)
    }
    const up = () => {
      el.classList.remove('dragging')
      localStorage.setItem('tt.bottomH', String(latest))
      window.removeEventListener('mousemove', move)
      window.removeEventListener('mouseup', up)
      window.removeEventListener('blur', up)
      document.removeEventListener('mouseleave', up)
    }
    window.addEventListener('mousemove', move)
    window.addEventListener('mouseup', up)
    window.addEventListener('blur', up)
    document.addEventListener('mouseleave', up)
  }

  return (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
      <Toolbar />
      {/* 整条 header(页签 + terminal/bug-play/list-tree 三个独立开关)横跨**整块调试区**,
          包括右侧那一列的上方 —— 这就是"header 连成一体"(用户 2026-10-16 的定案)。
          它必须在 editor|right 这层分栏**之外**:放进编辑器那一列的话,
          右侧栏又要另起一行图标,"哪一行管哪一块"就得靠猜。 */}
      <DebugHeader />
      {/* 面板紧贴(VS Code 经典密度):无外边距,仅靠 sash 分割线分区。
          min-w-0 + overflow-hidden:Monaco 会给编辑器写内联像素宽度,
          否则 flex 最小宽度被钉死,收起再展开时编辑区不回缩、右面板被挤出屏幕 */}
      <div className="flex min-h-0 min-w-0 flex-1 overflow-x-auto overflow-y-hidden">
        <div className="flex min-h-0 min-w-[320px] flex-1 flex-col overflow-hidden">
          {/* 编辑器:下方面板最大化时整块让位(那个"最大化"就是"我要专看这段协议流") */}
          <div className={`min-h-0 flex-1 overflow-hidden bg-card ${bottomMax ? 'hidden' : ''}`}>
            <SourceView />
          </div>
          {showBottom && (
            /* 最大化时这一列改吃整列高度(flex-1),**分割条不渲染** —— 那时高度由
               "最大化"接管,拖它什么也不会发生,留着只会让人以为还能拖。 */
            <div className={bottomMax ? 'flex min-h-0 flex-1 flex-col' : 'flex shrink-0 flex-col'}>
              {!bottomMax && (
                <div className="row-resizer" onMouseDown={onBottomResizeDown} title="拖拽调整下方面板高度" />
              )}
              <div style={bottomMax ? undefined : { height: bottomH }}
                className={bottomMax ? 'min-h-0 flex-1' : 'shrink-0'}>
                {/* 下方面板单独一层边界:时间线崩了不该把代码区一起带走 */}
                <ErrorBoundary what="下方面板（操作时间线）"><TimelinePanel /></ErrorBoundary>
              </div>
            </div>
          )}
        </div>
        {showRight && (
          <>
            <div className="col-resizer" onMouseDown={onResizeDown} title="拖拽调整代码区与侧边栏宽度" />
            {/* 右侧那一列里放两块面:调试面板(bug-play)与大纲/树(list-tree)。
                两者各自独立开合,但**同处一条手风琴**(见 RightPanels):不再上下硬切一半,
                谁占多少高度由内容与展开状态决定 —— 用户 2026-10-16 的定案。 */}
            <div style={{ width: panelW }} className="flex min-h-0 min-w-0 shrink-0 flex-col overflow-hidden">
              {/* 右侧栏单独一层边界:变量树/大纲崩了,代码区与工具条都还在
                  (实测过一次:一个畸形变量值把整页打成白屏)。 */}
              <ErrorBoundary what="右侧面板（调试 / 大纲）"><RightPanels /></ErrorBoundary>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
