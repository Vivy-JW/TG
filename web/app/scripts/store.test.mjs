// store 状态机校验(纯状态,不连后端):
//   node 无浏览器环境 → 先补 window/localStorage/document,再动态载入 store
//   (store 初始化时会读 localStorage,静态 import 会先于补全局执行)
// 覆盖易回退的行为:
//   1. 点重放/启动的那一刻(后端响应之前)编辑器就必须在 loading,且旧调试现场已清空;
//   2. 旧会话(同目标复用宿主 → ID 与新会话相同)的收尾事件 idle/exit/dead
//      不得把刚置上的加载态清掉、也不得弹「后端断开」误报横幅,但宽限期过后必须放行;
//   3. 断点列表点击是「纯浏览跳转」:只发视口滚动信号,不搬动黄色停站高亮(currentLine)。
import assert from 'node:assert/strict'

globalThis.window = { setInterval: () => 1, clearInterval: () => {}, setTimeout: () => 1 }
globalThis.localStorage = { getItem: () => null, setItem: () => {}, removeItem: () => {} }
globalThis.document = { documentElement: { classList: { toggle: () => {} } } }

// 复用宿主:后端返回的 sessionId 与点击前是同一个(最容易被收尾事件误伤的路径)
let release = () => {}
globalThis.fetch = () => new Promise((res) => {
  release = () => res({
    ok: true, status: 200,
    json: async () => ({ ok: true, sessionId: 'sess-1', module: 'awsq990', prog: 'awsq990', runProg: 'awsq990' }),
  })
})

const { useStore } = await import('../src/store')
const S = () => useStore.getState()

useStore.setState({
  sessionId: 'sess-1', view: 'wslogs', loadingSource: false, state: 'stopped',
  sourceContent: 'MAIN\nEND MAIN', timeline: [{ t: '1', origin: 'system', kind: 'info', text: '旧会话日志' }],
  stop: { file: 'old.4gl', line: 7 }, frames: [{ n: 0, file: 'old.4gl', line: 7, func: 'main' }],
})

// 1) 点下去(await 之前)编辑器就应该已经在加载,且旧现场已清空
const p = S().replayStart('rowid-1')
assert.equal(S().view, 'debug', '应立即切到 debug 页')
assert.equal(S().loadingSource, true, '应立即进入加载态(不等后端响应)')
assert.equal(S().state, 'loading')
assert.equal(S().sessionId, null, '旧会话引用应立即解除')
assert.equal(S().sourceContent, '', '旧源码应清空')
assert.equal(S().timeline.length, 0, '旧时间线应清空')
assert.equal(S().stop, null)
assert.equal(S().launching, true)
console.log('OK 1) 点击瞬间即 loading,旧调试现场已清空')

// 2) 请求在途:旧会话的收尾事件(idle/exit/dead)不得清掉加载态
for (const ev of [
  { type: 'state', sessionId: 'sess-1', state: 'idle' },
  { type: 'state', sessionId: 'sess-1', state: 'exit' },
  { type: 'dead', sessionId: 'sess-1', text: 'SSH 连接断开' },
  { type: 'log', sessionId: 'sess-1', text: '重放调试启动,结束当前调试(会话保留)' },
]) S().onEvent(ev)
assert.equal(S().loadingSource, true, '在途收尾事件不得取消加载态')
assert.equal(S().state, 'loading')
assert.equal(S().backendDead, '', '不得弹出「后端断开」误报横幅')
assert.equal(S().timeline.length, 1, '普通日志(非收尾)仍应进入时间线')
console.log('OK 2) 在途旧会话收尾事件被丢弃,加载态与横幅不受影响')

// 3) 响应回来:绑定新会话,仍保持加载(等入口停站)
release()
await p
assert.equal(S().sessionId, 'sess-1')
assert.equal(S().loadingSource, true, '响应后仍应保持加载,等入口停站')
assert.equal(S().state, 'loading')
console.log('OK 3) 响应后绑定会话并保持加载')

// 4) 响应后 2s 宽限内迟到的收尾事件(WS 与 HTTP 竞态)同样不得清掉加载态
S().onEvent({ type: 'state', sessionId: 'sess-1', state: 'idle' })
S().onEvent({ type: 'dead', sessionId: 'sess-1', text: 'SSH 连接断开' })
assert.equal(S().loadingSource, true, '宽限期内迟到事件不得取消加载态')
assert.equal(S().backendDead, '')
console.log('OK 4) 响应后宽限期内迟到事件被丢弃')

// 5) 宽限期过后必须恢复放行(不能永久屏蔽真实退出,否则界面会卡在加载中)
const realNow = Date.now
Date.now = () => realNow() + 60_000
S().onEvent({ type: 'state', sessionId: 'sess-1', state: 'idle' })
Date.now = realNow
assert.equal(S().loadingSource, false, '宽限期过后真实 idle 必须生效')
assert.equal(S().state, 'idle')
console.log('OK 5) 宽限期结束后真实收尾事件正常生效')

// 6) 无旧会话时(首次从日志页重放)行为一致,且不做多余屏蔽
useStore.setState({ sessionId: null, view: 'wslogs', loadingSource: false, sourceContent: '' })
const p2 = S().replayStart('rowid-2')
assert.equal(S().loadingSource, true)
assert.equal(S().view, 'debug')
release()
await p2
assert.equal(S().sessionId, 'sess-1')
console.log('OK 6) 无旧会话时同样立即 loading')

// 7) 普通启动(launch)同样是「点下即 loading」,且清掉重放标记
useStore.setState({ sessionId: null, view: 'debug', loadingSource: false, lastReplayRowid: 'rowid-2' })
const p3 = S().launch('awsq990', 'awsq990')
assert.equal(S().loadingSource, true, 'launch 应立即进入加载态')
assert.equal(S().lastReplayRowid, null, '普通启动应清掉重放标记')
release()
await p3
console.log('OK 7) launch 同样点下即 loading')

// 8) 断点列表点击 = 纯浏览跳转:只发视口滚动信号,黄色停站高亮(currentLine)原地不动
useStore.setState({
  sessionId: 'sess-1', state: 'stopped', sourceDVM: 'awsq990.4gl',
  currentLine: 120, stop: { file: 'awsq990.4gl', line: 120 }, revealReq: null,
})
await S().jumpToBp({ num: 1, enabled: true, file: 'awsq990.4gl', line: 480, func: 'main' })
assert.equal(S().currentLine, 120, '黄色停站高亮必须留在真实停站行')
assert.equal(S().revealReq.line, 480, '应发出滚动到断点行的信号')
assert.equal(S().revealReq.nav, true, '断点跳转应标记为纯浏览跳转')
assert.equal(S().revealReq.key, 'debug')
// 运行中(idle/running)也允许跳转:只是滚视口,不动运行上下文
useStore.setState({ state: 'running' })
await S().jumpToBp({ num: 1, enabled: true, file: 'awsq990.4gl', line: 480, func: 'main' })
assert.equal(S().currentLine, 120, '运行中跳转同样不得改停站高亮')
assert.equal(S().revealReq.nav, true)
// 真停站定位仍必须搬动高亮(nav 默认 false)
S().reveal('debug', 500)
assert.equal(S().currentLine, 500, '停站/步进定位应照旧搬动黄色高亮')
assert.equal(S().revealReq.nav, false)
console.log('OK 8) 断点跳转只滚视口,黄色停站高亮不动')

// ---------- 9) 源码取不到一次,不得把该文件永久钉成空白 ----------
//
// refreshSource 的"同文件已加载"早退只看 sourceDVM,而**取不到源码时也会把 sourceDVM
// 记成该文件**(为的是别把停站行画在旧文件上)。两者一撞就是死结:失败一次之后,
// 每次同文件停站都在早退分支直接 return —— 代码视图永远空白,而服务器明明有源码。
let srcMode = 'ok'
globalThis.fetch = async (url) => {
  if (String(url).includes('/source')) {
    if (srcMode === 'fail') return { ok: false, status: 500, json: async () => ({ error: '暂时取不到' }) }
    return {
      ok: true, status: 200,
      json: async () => ({ ok: true, source: { path: '/u1/asf/4gl/asf_x.4gl', content: 'MAIN\nEND MAIN', dvmFile: 'asf_x.4gl' } }),
    }
  }
  return { ok: true, status: 200, json: async () => ({ ok: true }) }
}

useStore.setState({
  sessionId: 'sx', module: 'asf', prog: 'bsft001_wf', activeTab: 'debug',
  sourceDVM: '', sourceMissing: false, sourceContent: '', loadingSource: false,
})

srcMode = 'fail'
await S().refreshSource('asf_x.4gl')
assert.equal(S().sourceDVM, 'asf_x.4gl', '取不到时仍要记 DVM(否则停站行会画在旧文件上)')
assert.equal(S().sourceContent, '', '取不到就是空')
assert.equal(S().loadingSource, false, '失败也必须退出加载态,不能一直转圈')
assert.equal(S().sourceMissing, true, '要标记 missing,供后续重试判断')

srcMode = 'ok'
await S().refreshSource('asf_x.4gl')
assert.equal(S().sourceContent, 'MAIN\nEND MAIN', '服务器恢复后必须重新加载(旧实现这里永久空白)')
assert.equal(S().sourceMissing, false)
console.log('OK 9) 源码取不到后恢复能重新加载(missing 不再被 sourceDVM 早退吞掉)')

// ---------- 10) 去抖落位走"不加载"分支时,必须收掉加载态 ----------
//
// case 'stopped' 在跨文件停站时置 loadingSource=true,并排一次 150ms 去抖落位。
// 若这 150ms 内源码已被别的路径(快照轮询)加载好,去抖就会走"同文件只落光标"分支 ——
// 那条分支以前不清 loadingSource,于是转圈框永远挂在编辑器上。
const pending = []
globalThis.window.setTimeout = (fn) => { pending.push(fn); return pending.length }
globalThis.window.clearTimeout = () => {}

useStore.setState({
  sessionId: 'sx', activeTab: 'debug', state: 'running',
  sourceDVM: 'asf_x.4gl', sourceMissing: false, sourceContent: 'MAIN\nEND MAIN', loadingSource: false,
})
S().onEvent({ type: 'stopped', sessionId: 'sx', stop: { reason: 'breakpoint', file: 'bsft001_wf.4gl', line: 12 } })
assert.equal(S().loadingSource, true, '跨文件停站的瞬间应进入加载态')

// 去抖窗口内,快照轮询那条路径已经把新文件加载好了
useStore.setState({ sourceDVM: 'bsft001_wf.4gl', sourceContent: 'MAIN\nEND MAIN' })
for (const fn of pending.splice(0)) await fn()
assert.equal(S().loadingSource, false, '去抖走"同文件只落光标"分支时也必须收掉加载态')
console.log('OK 10) 停站落位不再把加载态永久留在界面上')

// ---------- 11) 别人建的会话：**不认领**（Q4 的硬规矩 / C6.3），要人点 ----------
//
// 场景:页面一直开着,这时 AI 从 CLI 起了调试。事件带着 sessionId 一路到来,而本地没选中
// 任何会话。**从前的行为是"认领它"**,理由是"不认领的话编辑器永远转圈" —— 但那恰恰是
// **向服务端问"当前是哪条"**:池里有多条时绑到哪一条取决于事件先到谁,于是会出现
// "我盯着 A 的现场、界面却显示 B 的停站"。WP-F 按 Q4 把这条便利去掉:
//
//   · 事件**不得**把本地绑定改掉(未选择就保持未选择);
//   · 事件**不得**因此去打一次 `GET /api/sessions`(那正是"问服务端当前是哪条"的形状);
//   · 人要看得见会话、并且能自己点一条 —— 点击那一下才绑定,而且绑定要连状态/源码一起对齐。
let listCalls = 0
globalThis.fetch = async (url) => {
  const u = String(url)
  if (u.endsWith('/sessions')) {
    listCalls++
    return {
      ok: true, status: 200,
      json: async () => ({
        ok: true,
        sessions: [{
          id: 'sess-ai', module: 'ain', prog: 'aint302_wf', runProg: 'aint302_wf',
          state: 'stopped', env: '尚阳通', mode: 'collab', holdingSeconds: 7, breakpoints: 2, watchdogSeconds: 1800,
        }],
      }),
    }
  }
  if (u.includes('/sessions/sess-ai/source')) {
    return {
      ok: true, status: 200,
      json: async () => ({ ok: true, source: { path: '/u1/topprd/erp/cin/cin_aint302_wf.4gl', content: 'MAIN\nEND MAIN' } }),
    }
  }
  if (u.includes('/sessions/sess-ai')) {
    return {
      ok: true, status: 200,
      json: async () => ({
        ok: true, id: 'sess-ai', module: 'ain', prog: 'aint302_wf', runProg: 'aint302_wf',
        state: 'stopped', env: '尚阳通', mode: 'collab', holdingSeconds: 7, breakpoints: [],
        stop: { reason: 'breakpoint', file: 'cin_aint302_wf.4gl', line: 24014, func: 'aint302_send' },
      }),
    }
  }
  return { ok: true, status: 200, json: async () => ({ ok: true }) }
}

const settle = async (n = 20) => { for (let i = 0; i < n; i++) await Promise.resolve() }

useStore.setState({
  sessionId: null, sessionList: [], state: '', mode: 'solo', stop: null, currentLine: 0, launching: false,
  sourceDVM: '', sourceMissing: false, sourceContent: '', loadingSource: false,
})

// 同一批到来的三个事件(启动 → 停站):本地没选中任何会话 —— **一个绑定都不该发生**
listCalls = 0
S().onEvent({ type: 'ai_action', sessionId: 'sess-ai', action: 'session.launch', text: '启动调试 ain/aint302_wf' })
S().onEvent({ type: 'state', sessionId: 'sess-ai', state: 'stopped' })
S().onEvent({ type: 'stopped', sessionId: 'sess-ai', stop: { reason: 'breakpoint', file: 'cin_aint302_wf.4gl', line: 24014 } })
await settle()

assert.equal(S().sessionId, null,
  '事件不得替人选中会话(Q4/C6.3:前端不得向服务端询问"当前是哪条")—— 未选择就是未选择')
assert.equal(listCalls, 0,
  '更不能为了"认领"去打 GET /api/sessions —— 那正是"问服务端当前是哪条"的形状')
console.log('OK 11a) 事件不认领会话:未选择保持未选择,且不多打一次列表请求')

// **人点一条**才绑定(会话面板里点的那一下),而且绑定要连状态/模式/源码一起对齐
await S().selectSession('sess-ai')
await settle(30)
assert.equal(S().sessionId, 'sess-ai', '人点了就必须绑上(否则编辑器永远转圈、命令框永远不可用)')
assert.equal(S().state, 'stopped')
assert.equal(S().mode, 'collab', '协作模式要跟着会话进来(否则界面按纯人工放开写操作)')
assert.equal(S().stop?.line, 24014)
assert.equal(S().sourceContent, 'MAIN\nEND MAIN', '选中要把源码一起对齐,不是只填一个 id')
console.log('OK 11b) 人点一条会话即选中:id/状态/模式/源码一起对齐')

// ---------- 12) 入口停站只有 state=stopped,位置要从快照补 ----------
//
// 程序还没 run 的入口停站**不发 stopped 事件**(没有源码行可报),位置只在快照里。
// 于是"AI 起好会话、页面随后绑上"这条路拿到的是"已停站 + 空白编辑器",而命令框还写着
// "需停站后才能发送命令"。判据是"停站态到了却不知道停在哪"→ 去快照问一次并落到编辑器。
useStore.setState({
  sessionId: 'sess-entry', state: '', mode: 'solo', stop: null, launching: false,
  sourceDVM: '', sourceMissing: false, sourceContent: '', loadingSource: false,
})
globalThis.fetch = async (url) => {
  const u = String(url)
  if (u.includes('/sessions/sess-entry/source')) {
    return {
      ok: true, status: 200,
      json: async () => ({ ok: true, source: { path: '/u1/t35prd/erp/asf/4gl/bsft001_wf.4gl', content: 'TYPE\n\nMAIN\n  DISPLAY 1\nEND MAIN' } }),
    }
  }
  if (u.includes('/sessions/sess-entry')) {
    return {
      ok: true, status: 200,
      json: async () => ({
        ok: true, id: 'sess-entry', module: 'asf', prog: 'bsft001_wf', runProg: 'bsft001_wf',
        state: 'stopped', env: '主机109', mode: 'solo', holdingSeconds: 2, breakpoints: [],
        // 入口停站:有文件、没有行号
        stop: { reason: 'entry', file: 'asf_bsft001_wf.4gl' },
      }),
    }
  }
  return { ok: true, status: 200, json: async () => ({ ok: true }) }
}
S().onEvent({ type: 'state', sessionId: 'sess-entry', state: 'stopped' })
await settle(30)
assert.equal(S().state, 'stopped')
assert.equal(S().sourceDVM, 'asf_bsft001_wf.4gl', '入口停站也必须把源码取回来(否则编辑器空白)')
assert.ok(S().sourceContent.length > 0, '源码内容要落到编辑器上')
assert.equal(S().currentLine, 3, '入口停站应跳到 MAIN 首条语句(第 3 行)')
console.log('OK 12) 入口停站(无 stopped 事件)从快照补齐位置与源码')

// ---------- 15) 会话操作不许"假装成功" ----------
// 原来 `sessionOp` 写成 `if (op === 'close' && st.sessionId) await api.sessionClose(...)`：
// 条件不成立时**一个请求都不发**，却照样记一条"结束会话"到时间线、再 sync 一次 ——
// 用户看到的是"点了结束会话，什么都没发生，也没有任何提示"（真机踩到过）。
// 现在的契约：做不到就抛，让面板的 toast 把原因说出来。
useStore.setState({ sessionId: null }) // 前面几条用例留了会话，这条要的是"没有会话"那一刻
const noSid = await S().sessionOp('close').then(() => null, (e) => e)
assert.ok(noSid && /没有当前会话/.test(noSid.message), '没有会话时 close 必须抛错（不许静默成功）')
const noEnv = await S().sessionOp('switch').then(() => null, (e) => e)
assert.ok(noEnv && /点名/.test(noEnv.message), '没点名环境时 switch 必须抛错')
const warned = S().timeline.filter((t) => t.kind === 'warn').map((t) => t.text)
assert.ok(warned.some((x) => x.includes('结束会话失败')), '失败要进时间线（kind=warn）')
assert.ok(!S().timeline.some((t) => t.kind === 'command' && t.text.startsWith('结束会话')),
  '什么都没发的时候，不许记一条"成功"的命令行')
console.log('OK 15) 会话操作:做不到就抛错并记 warn，绝不假装成功')

console.log('\n全部通过: 19/19')

// ---------- 16) 树由池派生:池事件直接改树,不等刷新 ----------
// W2/W6/D9:树不是本地账本,而是 GET /api/sessions 与 session.added/removed 的投影。
// 这一条钉的是"CLI / 别的浏览器建的会话会出现在树上"的前端半边。
useStore.setState({ sessionId: null, sessionList: [], sessions: [] })
S().onEvent({
  type: 'session.added', sessionId: 's-cli-1',
  brief: {
    id: 's-cli-1', ref: '主机109#2', display: '主机109#2', name: 'CLI 建的',
    env: '主机109', ent: '99', n: 2, epoch: 'e1', prog: 'axmt500_wf', state: 'idle',
    module: '', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0,
  },
})
let tree = S().sessions
assert.equal(tree.length, 1, 'session.added 应让树当场多一条')
assert.equal(tree[0].id, 's-cli-1')
assert.equal(tree[0].type, 'debug', '所有会话在界面上都是调试那一类')
assert.equal(tree[0].ent, '99')
assert.equal(tree[0].prog, 'axmt500_wf')
assert.equal(tree[0].ref, '主机109#2')


// 池事件不认"绑定"这一套:我绑的是另一条,收到别人的 added 也照样进树
useStore.setState({ sessionId: 's-other' })
S().onEvent({
  type: 'session.added', sessionId: 's-cli-2',
  brief: { id: 's-cli-2', ref: '主机109#3', env: '主机109', name: 'B', state: 'idle', module: '', prog: '', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0 },
})
assert.ok(S().sessions.some((x) => x.id === 's-cli-2'), '别人的会话也要进树（树是池的投影）')
assert.equal(S().sessionId, 's-other', '池事件不得改绑定（绑定只能由人点）')

// 移除:树当场少一条
S().onEvent({ type: 'session.removed', sessionId: 's-cli-1', reason: 'close' })
assert.ok(!S().sessions.some((x) => x.id === 's-cli-1'), 'session.removed 应让树当场少一条')
console.log('OK 16) 树由池派生:added/removed 即时生效,且不动绑定')


// ---------- 17) 运行时按会话各存一份(W6):别的会话的状态也记 ----------
// 单例那套字段只装得下"绑着的那条";多会话下别人起停的事件必须落到它自己的运行时里,
// 否则树上的状态点只会在下一次 List 时才变(最长一秒多),而面板切回去还得重新问一遍。
useStore.setState({ sessionId: 's-bind', state: 'stopped', sessionList: [], sessions: [], runtimes: {} })
useStore.setState({ sessionList: [{ id: 's-bind', state: 'stopped', module: '', prog: 'p0', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0, mode: 'solo' },
  { id: 's-other', state: 'idle', module: 'asf', prog: 'axmt500_wf', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0, mode: 'collab' }] })

// 别的会话开始跑:运行时跟着变,单例与绑定一个都不动
S().onEvent({ type: 'state', sessionId: 's-other', state: 'running' })
assert.equal(S().runtimes['s-other']?.state, 'running', '别的会话的 state 事件要进它自己的运行时')
assert.equal(S().state, 'stopped', '不得改单例（那是绑定那条的状态）')
assert.equal(S().sessionId, 's-bind', '不得改绑定')
assert.equal(S().runtimes['s-other']?.prog, 'axmt500_wf', '运行时保留池里的作业编号')

// 它停站:停留现场也进运行时(切回这条会话时不必先问一次)
S().onEvent({ type: 'stopped', sessionId: 's-other', stop: { file: 'a.4gl', line: 42, reason: 'break' } })
assert.equal(S().runtimes['s-other']?.state, 'stopped')
assert.equal(S().runtimes['s-other']?.stop?.line, 42)

// 绑着的那条自己的事件:运行时与单例都更新（单例是页面在用的那份）
S().onEvent({ type: 'state', sessionId: 's-bind', state: 'running' })
assert.equal(S().state, 'running')
assert.equal(S().runtimes['s-bind']?.state, 'running')

// 池事件:added 落运行时(状态来自 brief),removed 把它删掉
S().onEvent({ type: 'session.added', sessionId: 's-new', brief: { id: 's-new', ref: '主机109#5', env: '主机109', name: 'N', state: 'loading', module: '', prog: '', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0 } })
assert.equal(S().runtimes['s-new']?.state, 'loading', 'session.added 要给新会话落一份运行时')
S().onEvent({ type: 'session.removed', sessionId: 's-new' })
assert.equal(S().runtimes['s-new'], undefined, '会话没了,运行时也不留')
assert.equal(S().runtimes['s-other']?.state, 'stopped', '别人的运行时不受影响')
console.log('OK 17) 运行时按会话各存一份:别人的状态进别人的账,单例只跟绑定那条')
// ---------- 18) 每条会话各有各的编辑器现场(W6/W-P2):切走再切回来,现场还在 ----------
// 从前切会话是把单例整块清掉/覆盖:从 A 点到 B 再点回 A,A 的页签、时间线、协议流、
// 打开过的源码全没了(只剩服务端能重拉的那点东西)。现在切换时把现场**换进换出**。
const realFetch = globalThis.fetch
globalThis.fetch = async (url) => {
  const u = String(url)
  if (u.includes('/api/sessions') && !/\/api\/sessions\/[^/]+$/.test(u)) {
    return { ok: true, status: 200, json: async () => ({ ok: true, sessions: [] }) }
  }
  return {
    ok: true, status: 200,
    json: async () => ({
      ok: true, state: 'stopped', module: 'asf', prog: 'axmt500_wf', runProg: '',
      stop: { file: 'asf_axmt500_wf.4gl', line: 9, reason: 'break' },
      breakpoints: [], started: true, holdingSeconds: 0, mode: 'solo',
    }),
  }
}
try {
  useStore.setState({
    sessionId: 's-a', editor: {}, sessionList: [
      { id: 's-a', state: 'stopped', module: 'asf', prog: 'axmt500_wf', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0, mode: 'solo' },
      { id: 's-b', state: 'stopped', module: 'asf', prog: 'bsft001_wf', holdingSeconds: 0, breakpoints: 0, watchdogSeconds: 0, mode: 'solo' },
    ],
    sourceContent: 'A 的源码', sourcePath: '/x/a.4gl', sourceDVM: 'a.4gl',
    timeline: [{ time: 't1', origin: 'system', kind: 'info', text: 'A 的事件' }],
    rawLog: ['A 的协议行'],
    tabs: [{ key: 'k-a', file: 'a.4gl', content: 'A', path: '/x/a.4gl', loading: false }],
    activeTab: 'k-a',
  })

  await S().selectSession('s-b')
  // A 的现场被存起来了（下一页要用的都还在）
  assert.equal(S().editor['s-a']?.sourceContent, 'A 的源码', '切走时要把 A 的源码存下来')
  assert.equal(S().editor['s-a']?.activeTab, 'k-a', '页签选择也要存')
  assert.equal(S().editor['s-a']?.timeline.length, 1, '时间线要存')
  assert.equal(S().editor['s-a']?.rawLog[0], 'A 的协议行', '协议流要存')
  // 而 B 的页面是**干净的**：不许把 A 的源码/时间线/页签带过去
  assert.equal(S().timeline.length, 0, '切到 B 不许留着 A 的时间线')
  assert.equal(S().rawLog.length, 0, '切到 B 不许留着 A 的协议流')
  assert.equal(S().tabs.length, 0, '切到 B 不许留着 A 的页签')
  assert.notEqual(S().sourceContent, 'A 的源码', '切到 B 时 A 的源码不许留在编辑器里')

  // 切回 A：现场原样回来（页签、时间线、协议流不必重新拉）
  //
  // 注意**不**断言 sourceContent：A 是 stopped，alignSelected 会按停站行重拉一次源码
  // （那是权威，比缓存新）；这里钉的是缓存里那份"服务端不会告诉我们"的现场。
  await S().selectSession('s-a')
  assert.equal(S().activeTab, 'k-a', '切回 A 时页签要回来')
  assert.equal(S().tabs.length, 1, '切回 A 时打开过的源码页签要回来')
  assert.equal(S().timeline.length, 1, '切回 A 时时间线要回来')
  assert.equal(S().rawLog[0], 'A 的协议行', '切回 A 时协议流要回来')

  // 会话没了 → 它的现场也不留（否则同 id 复用时会把旧现场挖出来）
  S().onEvent({ type: 'session.removed', sessionId: 's-a' })
  assert.equal(S().editor['s-a'], undefined, '会话没了,编辑器现场也要清')
  console.log('OK 18) 每条会话各有各的编辑器现场:切走不串场,切回来还在,收掉即清')
} finally {
  globalThis.fetch = realFetch
}
