// 外观主题:暗色 / 亮色 / 跟随系统。
// 只有"解析后的明暗"(dark 布尔)才是渲染依据:html.dark(整套 CSS 变量)与 Monaco 主题都跟它走,
// 而"用户选了什么"(theme)只用于设置页的选中态与持久化 —— 系统主题变化时,跟随模式的 dark 会变，
// 但 theme 仍然是 'system'(用户的选择没变)。
//
// 合并前只有调试工作台有主题机制,字典页的 dark: 类走的是 Tailwind v4 默认的
// prefers-color-scheme(跟随系统、不可切换)。合并后前端收敛成这一份,
// 整套 SPA 都按 html.dark 走,用户的选择全站共享。
export type ThemeMode = 'dark' | 'light' | 'system'

export const THEME_KEY = 'tt.theme'
export const THEME_MEDIA = '(prefers-color-scheme: dark)'

// 主题选择只能存 localStorage(它是纯客户端偏好,与 config.json 无关)。
// index.html 的首屏防闪色脚本里也写了同一个键名,改这里必须同步改那两个 html。
export const THEME_STORAGE_NOTE = 'index.html 的首屏脚本用同一个键,改名要一起改'

const mq = () => (typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia(THEME_MEDIA)
  : null)

// 系统当前是否偏好暗色(拿不到 matchMedia 时按暗色处理,与历史默认一致)
export function systemPrefersDark(): boolean {
  const m = mq()
  return m ? m.matches : true
}

// 用户选择 → 实际是否用暗色
export function resolveDark(mode: ThemeMode): boolean {
  if (mode === 'dark') return true
  if (mode === 'light') return false
  return systemPrefersDark()
}

// 读持久化选择:未设置默认暗色(保持与老版本一致,不让亮色系统用户"突然变亮")
export function readStoredTheme(): ThemeMode {
  try {
    const v = localStorage.getItem(THEME_KEY)
    return v === 'light' || v === 'system' ? v : 'dark'
  } catch {
    return 'dark'
  }
}

// 写持久化选择(设置页切换主题时调用;applyDark 由调用方负责)
export function storeTheme(mode: ThemeMode): void {
  try {
    localStorage.setItem(THEME_KEY, mode)
  } catch {
    /* 无痕模式等写不进去:本次会话仍然生效,只是不持久化 */
  }
}

// 把解析结果挂到 html.dark(整套主题变量的唯一开关)
export function applyDark(dark: boolean): void {
  document.documentElement.classList.toggle('dark', dark)
}

// 订阅系统主题变化(跟随模式下由调用方把新的 dark 应用出去)
export function watchSystemTheme(onChange: () => void): void {
  const m = mq()
  if (!m) return
  if (typeof m.addEventListener === 'function') m.addEventListener('change', onChange)
  else if (typeof (m as MediaQueryList & { addListener?: (f: () => void) => void }).addListener === 'function') {
    // 老内核兜底
    ;(m as MediaQueryList & { addListener: (f: () => void) => void }).addListener(onChange)
  }
}
