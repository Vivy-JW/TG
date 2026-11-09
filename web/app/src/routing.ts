// 工作站的哈希路由:左边是「环境 → 会话」两级目录,右边是所选目标的那一屏(带 Tab)。
//
// 片段形态:
//   #env/<环境名>/<Tab>                环境页:create 创建会话 / wslogs 接口日志
//   #session/<会话 ID>/<Tab>           会话页:debug 调试
//   #settings/<分区>                   设置页的某个分区
//
// 认不出来的一律给 `home`(由外壳决定落到哪个环境,绝不猜)。
//
// 约束(踩过):本文件**不得被 store.ts 运行时导入**,且模块顶层不得访问 window/document/history。
// 原因:check:store 用裸打包器把它打进 Node 里跑(桩 window/document),顶层碰 DOM 会在
// 测试启动前就抛错。所以 DOM 访问全部留在函数体内、由 React effect 调用。

/** 设置页的五个分区 */
export type SettingsSectionKey = 'sites' | 'data-dict' | 'debug' | 'skills' | 'app'

/** 分区清单(键 + 显示名)。左树与深链接校验共用这一份,免得两处各列一遍。 */
export const SETTINGS_SECTIONS: { key: SettingsSectionKey; label: string }[] = [
  { key: 'sites', label: '站点管理' },
  { key: 'data-dict', label: '数据字典' },
  { key: 'debug', label: 'DEBUG' },
  { key: 'skills', label: '技能' },
  { key: 'app', label: '应用设置' },
]

export const DEFAULT_SETTINGS_SECTION: SettingsSectionKey = 'sites'

/** 环境页的三个 Tab */
export type EnvTab = 'create' | 'wslogs'

/** 会话页的 Tab（只剩调试） */
export type SessionTab = 'debug'

/**
 * 状态容器里的 `view`:它回答"此刻主面板上生效的是哪一块功能",
 * 供接口日志/设置页那些"进来就重拉一次"的 effect 判断自己是不是当前那一屏。
 * 新增的 create 是**外壳的 Tab**,不是新功能页 —— 它只用于让旧的判断落空。
 */
export type ViewKey = 'debug' | 'wslogs' | 'settings' | 'create'

/** 外壳当前该显示什么。`home` = 片段为空或认不出来,由 App 落到默认目标。 */
export type Route =
  | { kind: 'home' }
  | { kind: 'settings'; section?: SettingsSectionKey }
  | { kind: 'env'; env: string; tab: EnvTab }
  | { kind: 'session'; id: string; tab: SessionTab }

export function isSettingsSection(v: string): v is SettingsSectionKey {
  return SETTINGS_SECTIONS.some((s) => s.key === v)
}

function isEnvTab(v: string): v is EnvTab {
  return v === 'create' || v === 'wslogs'
}

function isSessionTab(v: string): v is SessionTab {
  return v === 'debug'
}

/** 会话 ID 的合法形态:字母数字开头,后面允许字母数字与 `-` `_` `.`,最长 80。 */
export function isSessionID(v: string): boolean {
  return /^[A-Za-z0-9][A-Za-z0-9._-]{0,79}$/.test(v)
}

function decodePart(s: string): string {
  try {
    return decodeURIComponent(s).trim()
  } catch {
    return ''
  }
}

/**
 * 解析 location.hash。认不出来的一律返回 `home`(不猜,让调用方走默认)。
 *
 * 环境名允许中文/空格(设置里就叫这个),所以编码后放进片段、解析时解回来。
 */
export function parseRoute(hash: string): Route {
  const raw = (hash || '').replace(/^#/, '').trim()
  if (!raw) return { kind: 'home' }
  const qi = raw.indexOf('?')
  const path = qi >= 0 ? raw.slice(0, qi) : raw
  const parts = path.split('/')
  const head = parts[0]

  if (head === 'settings') {
    const tail = parts[1]
    return tail && isSettingsSection(tail) ? { kind: 'settings', section: tail } : { kind: 'settings' }
  }

  if (head === 'env') {
    const env = parts[1] ? decodePart(parts[1]) : ''
    if (!env || env.length > 200) return { kind: 'home' }
    const tab: EnvTab = parts[2] && isEnvTab(parts[2]) ? parts[2] : 'create'
    return { kind: 'env', env, tab }
  }

  if (head === 'session') {
    const id = parts[1] ? decodePart(parts[1]) : ''
    if (!id || !isSessionID(id)) return { kind: 'home' }
    const tab: SessionTab = parts[2] && isSessionTab(parts[2]) ? parts[2] : 'debug'
    return { kind: 'session', id, tab }
  }

  return { kind: 'home' }
}

/** 设置页某个分区的 hash(带前导 #,可直接交给 history.replaceState) */
export function settingsHash(section: SettingsSectionKey): string {
  return `#settings/${section}`
}

/** 这份路由对应状态容器里的哪个 view(日志/设置那些 effect 靠它判断"我在不在前台")。 */
export function routeView(route: Route): ViewKey {
  switch (route.kind) {
    case 'settings': return 'settings'
    case 'env': return route.tab
    case 'session': return route.tab
    // home 只出现在"片段为空 / 认不出来"的那一瞬,主面板这时候还没选目标 ——
    // 给 create 是为了不让设置页(它只在 view==='settings' 时回读配置)被误判成前台。
    default: return 'create'
  }
}

/**
 * 当前路由对应的 hash。写片段这件事必须**双向**(读进来 + 写回去):只读不写的话,
 * 点开另一个 Tab 时 URL 还停在旧位置,一刷新就把人弹回旧页(实测过同样的毛病)。
 */
export function routeHash(route: Route, section: SettingsSectionKey): string {
  switch (route.kind) {
    case 'settings':
      return settingsHash(section)
    case 'env':
      return `#env/${encodeURIComponent(route.env)}/${route.tab}`
    case 'session':
      return `#session/${encodeURIComponent(route.id)}/${route.tab}`
    default:
      return '#'
  }
}
