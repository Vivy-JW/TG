import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App'
import { ErrorBoundary } from './ErrorBoundary'
import { editorRef } from './SourceView'
import { useStore } from './store'
import { ToastProvider } from './toast'
import { applyDark, readStoredTheme, resolveDark } from '../../shared/theme'
import './index.css'

// 启动即同步主题类(store 默认暗色,但只有手动切换时才挂 .dark,首帧不挂会闪一下亮色)——
// 与 store 的取值规则保持一致;index.html 里的内联脚本负责更早的一帧(避免首屏闪色)。
applyDark(resolveDark(readStoredTheme()))

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    {/* 根上那道错误边界:任何漏出去的渲染期异常都只吃掉"这一屏"，
        不能再变成整页白屏(实测一个畸形变量值就能做到，见 ErrorBoundary.tsx)。
        它在 ToastProvider 外层:连消息出口自己崩了也还有人接住。 */}
    <ErrorBoundary what="工作台">
      {/* 统一消息出口挂在最外层:全站任何组件都能 useToast(),而 toast 只渲染一次。 */}
      <ToastProvider>
        <App />
      </ToastProvider>
    </ErrorBoundary>
  </React.StrictMode>,
)

// 浏览器控制台调试入口
;(window as any).__ed = editorRef
;(window as any).__store = useStore
