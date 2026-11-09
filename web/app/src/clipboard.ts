// 复制到系统剪贴板：异步 API 优先，失败退回 execCommand。
//
// 为什么需要兜底：`navigator.clipboard.writeText` 在**文档失焦**时直接抛
// NotAllowedError（无头/后台页、先把窗口切走再回来点，都会中）。原来两处调用
// （接口日志的报文复制、日志页的「复制 ID」）都只有 try/catch → 按钮红 1.5 秒说
// "复制失败"，没有任何别的出口，而这两段文本本来都是可以手选复制的东西。
//
// 返回是否成功，由调用方决定怎么提示（两处的界面语言不一样）。
export async function copyText(text: string): Promise<boolean> {
  if (!text) return false
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return true
    }
  } catch { /* 落到下面的兜底 */ }
  // 兜底：临时 textarea + execCommand('copy')。同步、不要求文档有焦点。
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.setAttribute('readonly', '')
    ta.style.position = 'fixed'
    ta.style.top = '-1000px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}