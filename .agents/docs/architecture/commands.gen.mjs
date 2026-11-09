import fs from "node:fs"

// 生成 commands.html（命令参考手册）：node commands.gen.mjs（在本目录下跑，直接写 commands.html）
// 用法：在本目录执行 node commands.gen.mjs —— 直接覆盖 commands.html

const R = 'real' // 实跑抓的
const S = 'shape' // 需真实环境，形状示意
const H = 'human' // 只有 human 是实跑/代码可证的

const envHead = [
  '# 环境 主机109 · SSH tiptop@10.0.0.1:22 · 区域 31',
  '# 企业(ENT) 109 → 账号 ds · 来源 ent(gzou_t) · 库 oracle 10.0.0.2:1521/t100dev client-direct · 只读事务',
]
const envJsonHead = '"source":"live","env":"主机109","sshHost":"tiptop@10.0.0.1:22","zone":"31","ent":109,"account":"ds","dialect":"oracle","target":"10.0.0.2:1521/t100dev","route":"client-direct","readonly":true'

const groups = [
  {
    id: 'env',
    title: '环境与配置',
    note: '坐标与配置。口令永不进输出；没有默认环境，每次都点名。',
    cmds: [
      {
        cmd: 'tt env list', why: '列出全部环境', mode: R, tip: '表格最后一行会写配置文件路径 —— 多份配置时先看它。',
        human: `名称      SSH                   区域  数据库
开发环境  tiptop@10.0.0.1:22    31    oracle 10.0.0.2:1521/t100dev
正式环境  tiptop@10.0.1.1:22    36    kingbase 10.0.1.2:54321/topprd

配置文件：C:\\Users\\me\\AppData\\Roaming\\tt\\config.json`,
        json: `{"config":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt\\\\config.json","sshs":[{"name":"开发环境","host":"10.0.0.1","port":22,"user":"tiptop","zone":"31","dbType":"oracle","dbAddress":"10.0.0.2:1521/t100dev"},{"name":"正式环境","host":"10.0.1.1","port":22,"user":"tiptop","zone":"36","dbType":"kingbase","dbAddress":"10.0.1.2:54321/topprd"}]}`,
      },
      {
        cmd: 'tt env show <环境名>', why: '一个环境的连接详情', mode: R, tip: '口令打码成 ******，只显示有没有配。',
        human: `名称      开发环境
SSH       tiptop@10.0.0.1:22
口令      ******
区域      31
数据库    oracle 10.0.0.2:1521/t100dev
只读 SQL  开启
隧道      tiptop@10.0.0.1:22
账号      ds, dsdemo`,
        json: `{"name":"开发环境","host":"10.0.0.1","port":22,"user":"tiptop","zone":"31","dbType":"oracle","dbAddress":"10.0.0.2:1521/t100dev"}`,
      },
      {
        cmd: 'tt env ents [<环境名>]', why: '环境支持的企业编号（ENT）与账号', mode: R, tip: '加 --refresh 才连库同步；不加只读本机配置（离线可答）。human 里那对 ** 是原文如此。',
        human: `环境 开发环境   区域 31 · oracle · 10.0.0.2:1521/t100dev   **未同步**（跑 tt env ents --refresh，或设置页「ENT」页签）

环境 正式环境   区域 36 · kingbase · 10.0.1.2:54321/topprd   **未同步**（跑 tt env ents --refresh，或设置页「ENT」页签）`,
        json: `{"count":2,"envs":[{"env":"开发环境","zone":"31","dialect":"oracle","target":"10.0.0.2:1521/t100dev","account":"ds","synced":false,"count":0}],"failed":0,"ok":true,"source":"config"}`,
      },
      {
        cmd: 'tt config path', why: '打印配置文件的解析结果', mode: R, tip: '回答“它到底在读哪个文件”，多份配置时最有用。',
        human: `C:\\Users\\me\\AppData\\Roaming\\tt\\config.json`,
        json: `{"default":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt\\\\config.json","exists":true,"path":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt\\\\config.json","portable":false}`,
      },
      {
        cmd: 'tt config get <键路径>', why: '按点分路径读一个值', mode: R, tip: 'JSON 模式下整份输出就是一个裸值（字符串/数字/对象），没有信封。',
        human: `开发环境`,
        json: `"开发环境"`,
      },
      {
        cmd: 'tt config set <键路径> <值>', why: '按点分路径写一个值（唯一写入口）', mode: R, tip: '值按 JSON 解析：&quot;-ws2&quot;（带引号）才是字符串，解析不了就当字符串；写盘是原子的。',
        human: `tdev.workspaceSuffix = "-ws2"`,
        json: `{"key":"tdev.workspaceSuffix","ok":true,"value":"-ws"}`,
      },
      {
        cmd: 'tt config show', why: '打印配置内容（口令打码）', mode: R, tip: '--human 下也是 JSON（配置本身就是 JSON），只多一行打码说明；--raw 看原样。',
        human: `{"bdldoc":{"dir":""},"debug":{"launchArgs":"BBDL512840855a 2 12345 'N' {prog}","termWidth":200,…},"hosts":{"sshs":[{"db":{"accounts":[{"account":"ds","password":"***"}…],"type":"oracle"},"name":"开发环境","password":"***","zone":"31"}]},"schemaVersion":3,…}

（口令已打码；加 --raw 查看原样内容）`,
        json: `{"bdldoc":{"dir":""},"debug":{…},"hosts":{"sshs":[{"name":"开发环境","password":"***","db":{"accounts":[{"account":"ds","password":"***"}]}}]},"schemaVersion":3}`,
      },
      {
        cmd: 'tt config validate', why: '校验配置完整性', mode: R, tip: 'problems 为 null 即没问题；每条问题都带路径。',
        human: `C:\\Users\\me\\AppData\\Roaming\\tt\\config.json：未发现问题。`,
        json: `{"config":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt\\\\config.json","ok":true,"problems":null}`,
      },
    ],
  },
  {
    id: 'session',
    title: '会话 session',
    note: '会话是一级对象：它持有身份（ref / name / n / epoch）与作业实例。目标一律 --session，多条不带目标会报歧义并列候选。',
    cmds: [
      {
        cmd: 'tt session new --env <环境> --ent <企业> --name <名字> [--prog <作业>]', why: '建会话（可同时起作业）', mode: S, tip: '要先有 serve。会话地址是 环境#序号，不是“默认那一条”。',
        human: `会话已建立：主机109#1   名字 A   企业 109   作业 bsft001_wf   状态 idle`,
        json: `{"ok":true,"session":{"id":"s7f3","ref":"主机109#1","name":"A","env":"主机109","ent":109,"n":1,"epoch":1,"prog":"bsft001_wf","state":"idle"}}`,
      },
      {
        cmd: 'tt session ls', why: '列会话', mode: S, tip: '默认过滤掉 state=exit 的；--all 全出。',
        human: `REF          名字  环境       ENT  作业          用途  状态   世代
主机109#1    A     主机109    109  bsft001_wf   -     idle   1`,
        json: `{"ok":true,"sessions":[{"id":"s7f3","ref":"主机109#1","display":"主机109#1","name":"A","env":"主机109","ent":109,"n":1,"epoch":1,"prog":"bsft001_wf","state":"idle"}]}`,
      },
      {
        cmd: 'tt session show --session <ref>', why: '身份 / 作业 / 停站位置', mode: S, tip: '不确定该传什么时先 show，ref 与别名都会回显。',
        human: `ref     主机109#1
名字    A        世代 1
环境    主机109   企业 109
作业    bsft001_wf
状态    stopped @ bsft001_wf:812`,
        json: `{"ok":true,"session":{"ref":"主机109#1","name":"A","epoch":1,"env":"主机109","ent":109,"prog":"bsft001_wf","state":"stopped","line":812}}`,
      },
      {
        cmd: 'tt session ent <企业编号> --session <ref>', why: '改会话的企业编号（仅空闲时可设）', mode: S, tip: '跑起来之后改不了 —— 那会把账号与 schema 换掉。',
        human: `主机109#1 企业 109 → 99   账号 ds → dsdemo`,
        json: `{"ok":true,"ref":"主机109#1","ent":99,"account":"dsdemo"}`,
      },
      {
        cmd: 'tt session job --prog <作业> --session <ref>', why: '在这条会话里起一个作业', mode: S, tip: '会话本来就空闲才能起；一轮一个作业。',
        human: `主机109#1 起作业 aapt320 → 等到入口停站 @ aapt320:34`,
        json: `{"ok":true,"ref":"主机109#1","prog":"aapt320","state":"stopped","line":34}`,
      },
      {
        cmd: 'tt session exec --session <ref> -- "<fgldb 命令>"', why: '透传一条 fgldb 命令', mode: S, tip: '等价 tt debug exec；命令原样给 fgldb，tt 不改写。',
        human: `> break 812
Breakpoint 1 at bsft001_wf:812`,
        json: `{"ok":true,"ref":"主机109#1","commands":["break 812"],"output":"Breakpoint 1 at bsft001_wf:812"}`,
      },
      {
        cmd: 'tt session quit --session <ref>', why: '结束本轮（会话保留，回空闲）', mode: S, tip: '会话还在，只是这一轮结束 —— 再 job 就能复用宿主、免重新登录。',
        human: `主机109#1 本轮已结束，会话保留（idle）`,
        json: `{"ok":true,"ref":"主机109#1","state":"idle"}`,
      },
      {
        cmd: 'tt session rm --session <ref>', why: '收掉会话（断连接、清资源）', mode: S, tip: '与 quit 的区别：rm 之后这条会话不存在了。',
        human: `主机109#1 已收掉（连接断开、资源清理）`,
        json: `{"ok":true,"ref":"主机109#1","removed":true}`,
      },
    ],
  },
  {
    id: 'db',
    title: '数据 db',
    note: '只读 SQL 与连通性。查元数据用 tt dict；这里只回答“库通不通 / 这条 SELECT 出什么”。',
    cmds: [
      {
        cmd: 'tt db ping --at <环境>[@<企业>]', why: '本机直连一次库（可走 SSH 隧道）', mode: S, tip: '验证网络 / VPN / 隧道与账号口令；对端视角用 probe。',
        human: `连接正常: 主机109@109 (ds, oracle) Oracle Database 19c Enterprise Edition`,
        json: `{"ok":true,"env":"主机109","ent":109,"account":"ds","dialect":"oracle","target":"10.0.0.2:1521/t100dev","version":"Oracle Database 19c Enterprise Edition"}`,
      },
      {
        cmd: 'tt db probe --at <环境>', why: '服务器侧连库，验证该企业账号', mode: H, tip: '走 SSH + sqlplus/ksql，验的是“服务器上有没有这个账号”。--json 原样打印同一份报告对象。',
        human: `区域: 31   Oracle: 10.0.0.2:1521 (service t100dev)
sqlplus: /u1/topprod/oracle/.../sqlplus
ORACLE_HOME: /u1/topprod/oracle/...

企业(ENT) → 账号(schema)，共 2 个（来自配置的 ENT 清单）：
✓ 109    → ds
· 99     → dsdemo

企业 109 连接验证: ds@T100DEV
  主机: 10.0.0.2:1521  服务名: t100dev
  结果: 连接成功 ✓`,
        json: ``,
      },
      {
        cmd: 'tt db sql "<查询语句>" --at <环境>@<企业>', why: '执行一条只读 SQL（最多 200 行）', mode: S, tip: '单条 SELECT/WITH、只读事务；totalRows 是下界（库侧多取一行判“还有更多”），要更多请收窄 WHERE。',
        human: `${envHead.join('\n')}
# 行数 3（共 3）
sfaa001  sfaa002  sfaa003
SFAA001  工单       2026-02-27`,
        json: `{"ok":true,${envJsonHead},"totalRows":3,"returned":3,"data":{"columns":["sfaa001","sfaa002","sfaa003"],"rows":[["SFAA001","工单","2026-02-27"]]}}`,
      },
    ],
  },
  {
    id: 'ops',
    title: '运维与交付',
    note: '装、起、更新、清。这一组大多与具体环境无关。',
    cmds: [
      {
        cmd: 'tt serve', why: '本地 Web 服务（工作站 + 设置页）', mode: S, tip: '调试与字典的 REST/WS 都在这里；CLI 与浏览器共用同一份会话。',
        human: `tt serve 已启动：http://127.0.0.1:28670
  工作站 / 设置页 / 字典 / 调试 REST + WebSocket
  Ctrl+C 停止（守护进程 tt debug serve 不受影响）`,
        json: ``,
      },
      {
        cmd: 'tt install skills [--to <目录>|--agent <名字>|--to auto]', why: '技能手册 → .agents/skills', mode: R, tip: '目标是“追加”语义：同名跳过，--force 才覆盖；装前逐个校验 frontmatter。',
        human: `技能安装完成: 4 个文件
  来源: D:\\我的项目\\TT\\skills
  目标: D:\\我的项目\\TT\\.agents\\skills
    tt-dev-tzc/SKILL.md
    tt-erp-architecture/SKILL.md
    tt-erp-debug/SKILL.md
    tt-erp-read/SKILL.md`,
        json: `{"files":["tt-dev-tzc/SKILL.md","tt-erp-architecture/SKILL.md","tt-erp-debug/SKILL.md","tt-erp-read/SKILL.md"],"ok":true,"skipped":null,"source":"D:\\\\TT\\\\skills","target":"D:\\\\TT\\\\.agents\\\\skills","why":""}`,
      },
      {
        cmd: 'tt install path [--dry-run]', why: '把 tt 目录加进用户 PATH（HKCU）', mode: H, tip: '只动 HKCU，不碰系统 PATH；--dry-run 先看会改成什么。',
        human: `{"changed":true,"dryRun":true,"exeDir":"D:\\\\我的项目\\\\TT","new":"…;D:\\\\我的项目\\\\TT","ok":true,"old":"…"}`,
        json: `{"changed":true,"dryRun":true,"exeDir":"D:\\\\我的项目\\\\TT","ok":true,"old":"…","new":"…;D:\\\\我的项目\\\\TT"}`,
      },
      {
        cmd: 'tt install bdl [--dir <目录>] [--force]', why: '解压随包的 BDL 语言文档', mode: S, tip: '首次运行也会自动解压；目录记在 config.json 的 bdldoc.dir。',
        human: `BDL 文档已解压到 C:\\Users\\me\\AppData\\Roaming\\tt\\bdldoc（1234 个文件）`,
        json: `{"ok":true,"dir":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt\\\\bdldoc","files":1234}`,
      },
      {
        cmd: 'tt update check', why: '查有没有新版（不下载）', mode: S, tip: '要能访问 GitHub；代理用 TT_PROXY 或 config net.proxy。',
        human: `当前 devel (commit ee6941d) · 最新 v0.5.0 → 有新版本
  升级：tt update`,
        json: `{"ok":true,"current":"devel","latest":"v0.5.0","updateAvailable":true}`,
      },
      {
        cmd: 'tt update log', why: '回读最近一次升级的阶段与结果', mode: S, tip: '升级是“交棒给脱离进程”，所以结果要事后回读。',
        human: `2026-02-27 18:02:11 下载 v0.5.0 → 校验 sha256 通过 → 交棒 → 成功（新版本已就位）`,
        json: `{"ok":true,"stage":"done","from":"devel","to":"v0.5.0","at":"2026-02-27T18:02:11+08:00"}`,
      },
      {
        cmd: 'tt cache [clear]', why: '查看 / 清理缓存', mode: R, tip: '缓存目录：企业快照、源码镜像、执行日志、断点存档、查询落盘、更新、runtime。',
        human: `缓存目录: C:\\Users\\me\\AppData\\Roaming\\tt
  ents            2.5 KB  1 个文件
  srccache             —  0 个文件
  execlog              —  0 个文件
  debug-bps            —  0 个文件
  spill                —  0 个文件
  update               —  0 个文件
  runtime              —  0 个文件

共 2.5 KB / 1 个文件;tt serve 启动时清理超过 168 小时的`,
        json: `{"ok":true,"totalRows":0,"returned":0,"data":{"dir":"C:\\\\Users\\\\me\\\\AppData\\\\Roaming\\\\tt","dirs":[{"name":"ents","exists":true,"bytes":2568,"files":1}],"bytes":2568,"files":1,"maxAgeHours":168}}`,
      },
      {
        cmd: 'tt cache clear [--older-than <时长>]', why: '清理缓存', mode: R, tip: '注意：这条不吃 --json，两种模式都打人读文案。',
        human: `已清理超过 9999h0m0s 的缓存: 0 个文件,释放 0 B`,
        json: `（同 human；这条命令目前只输出人读文案）`,
      },
      {
        cmd: 'tt version', why: '显示版本信息', mode: R, tip: '版本号唯一出处是仓库根的 VERSION；devel 后面跟 commit。',
        human: `tt devel (commit ee6941d, 2026-02-27)`,
        json: `{"hints":[],"ok":true,"version":"devel (commit ee6941d, 2026-02-27)"}`,
      },
    ],
  },
  {
    id: 'debug',
    title: '调试 debug',
    note: '通过 SSH 驱动服务器上的 fglrun -d（fgldb 文本协议）。先 tt serve，再 start / exec / stop 这一套；目标一律点名。',
    cmds: [
      {
        cmd: 'tt debug start <作业编号> --at <环境>@<企业>', why: '连 SSH 起作业，等到入口停站', mode: S, tip: '等到“入口停站”才返回 —— 不要拿 sleep 当同步。',
        human: `环境 主机109 · 企业 109 · 作业 bsft001_wf
会话 主机109#1  状态 stopped @ bsft001_wf:34（入口停站）`,
        json: `{"ok":true,"ref":"主机109#1","prog":"bsft001_wf","state":"stopped","line":34,"reason":"entry"}`,
      },
      {
        cmd: 'tt debug status', why: '调试服务状态与活动会话', mode: S, tip: '先看这条：serve 没起、会话不在，后面全都白搭。',
        human: `调试服务  运行中  http://127.0.0.1:28670
活动会话  1 条
  主机109#1  bsft001_wf  stopped@34  collab`,
        json: `{"ok":true,"sessions":[{"ref":"主机109#1","prog":"bsft001_wf","state":"stopped","line":34,"mode":"collab"}]}`,
      },
      {
        cmd: 'tt debug exec "<fgldb 命令>" [更多...] --session <ref>', why: '透传 fgldb 命令', mode: S, tip: 'print / next / where / info / list … 原样给 fgldb；可一次多条。',
        human: `> where
#0 bsft001_wf:812 in bsft001_b_fill()
#1 bsft001_wf:120 in MAIN`,
        json: `{"ok":true,"ref":"主机109#1","output":"#0 bsft001_wf:812 in bsft001_b_fill()\\n#1 bsft001_wf:120 in MAIN"}`,
      },
      {
        cmd: 'tt debug bp list|add|del', why: '断点面（带命中计数）', mode: S, tip: 'add 的位置：行号 / 函数名 / file:line；--kind watch 时位置是表达式。',
        human: `#  位置                 种类   命中  条件
1  bsft001_wf:812        line   3     -
2  bsft001_b_fill       func   0     sfba003 = "A001"`,
        json: `{"ok":true,"sessionId":"主机109#1","breakpoints":[{"id":1,"where":"bsft001_wf:812","kind":"line","hits":3},{"id":2,"where":"bsft001_b_fill","kind":"func","hits":0,"cond":"sfba003 = \\"A001\\""}],"fresh":true}`,
      },
      {
        cmd: 'tt debug stop', why: '当前停站现场', mode: S, tip: '可复取：state / 位置 / 断点 / TOPENT 一次给全，断开重连也能再问。',
        human: `会话 主机109#1  状态 stopped
位置 bsft001_wf:812（函数 bsft001_b_fill）
企业 109         TOPENT 109
断点 2 条（命中 1 条）`,
        json: `{"ok":true,"ref":"主机109#1","state":"stopped","line":812,"func":"bsft001_b_fill","ent":109,"topent":"109","breakpoints":2}`,
      },
      {
        cmd: 'tt debug source [DVM 文件] [--from <行> --to <行>]', why: '读服务器源码（白名单只读）', mode: S, tip: '登录区源码目录白名单内只读；不会碰工作区。',
        human: `  810  LET l_cnt = 0
  811  SELECT COUNT(*) INTO l_cnt FROM sfaa_t
  812    WHERE sfaa001 = g_master.sfaadocno`,
        json: `{"ok":true,"file":"bsft001.4gl","from":810,"to":812,"lines":["  LET l_cnt = 0","  SELECT COUNT(*) INTO l_cnt FROM sfaa_t","    WHERE sfaa001 = g_master.sfaadocno"]}`,
      },
      {
        cmd: 'tt debug locate <函数名|模块.函数>', why: '函数定位到 文件:行', mode: S, tip: '需停站（走 info line）；不停站时先 start 或 wait。',
        human: `bsft001_b_fill → bsft001_wf:812`,
        json: `{"ok":true,"func":"bsft001_b_fill","file":"bsft001_wf","line":812}`,
      },
      {
        cmd: 'tt debug logs [--tail <N>]', why: '会话最近事件', mode: S, tip: '停站 / 状态 / 日志 / 断开都在这条流里；tail 查询不下新断点。',
        human: `18:02:11  stopped   bsft001_wf:812  breakpoint#1
18:02:10  output    SELECT 1 row
18:02:09  state     running → stopped`,
        json: `{"ok":true,"ref":"主机109#1","events":[{"at":"18:02:11","type":"stopped","where":"bsft001_wf:812"}]}`,
      },
      {
        cmd: 'tt debug wait [--timeout <秒>]', why: '阻塞等事件', mode: S, tip: '等的是停站 / 退出 / 掉线 / 看门狗；给超时，别无限挂。',
        human: `等到停站：bsft001_wf:812（用时 4.2s）`,
        json: `{"ok":true,"ref":"主机109#1","event":"stopped","where":"bsft001_wf:812","elapsedSeconds":4.2}`,
      },
      {
        cmd: 'tt debug why', why: '探测在等用户还是在空转', mode: S, tip: '“慢”分两种：等人操作（正常）与慢查询 / 慢循环（要查）。',
        human: `在等用户操作（GDC 侧 INPUT，已等 12s；不是慢 SQL）`,
        json: `{"ok":true,"verdict":"waiting-input","elapsedSeconds":12}`,
      },
      {
        cmd: 'tt debug wslogs [--show <rowid>]', why: '接口日志列表（wssp/awsp 报文）', mode: S, tip: '--show 看某条报文的请求/响应；重放用 wsdebug。',
        human: `ROWID    时间       程序        接口            报文
r0a3f1   18:01:55   bsft001_wf  wssp.aooi210    2.1 KB
r0a3f0   18:01:44   bsft001_wf  awsp.sfaa_t     1.4 KB`,
        json: `{"ok":true,"totalRows":2,"returned":2,"data":[{"rowid":"r0a3f1","at":"18:01:55","prog":"bsft001_wf","api":"wssp.aooi210","bytes":2150}]}`,
      },
      {
        cmd: 'tt debug wsdebug <rowid> --at <环境>@<企业>', why: '按接口日志重放，等到停站', mode: S, tip: '池空照样能用：重放自己建一条会话（冷启动档）。',
        human: `重放 r0a3f1 → 新会话 主机109#2  状态 stopped @ bsft001_wf:34`,
        json: `{"sessionId":"s91c","module":"asf","prog":"bsft001_wf","runProg":"bsft001_wf"}`,
      },
      {
        cmd: 'tt debug mode [solo|collab]', why: '会话模式：纯人工 / 协作', mode: S, tip: 'collab = AI 主导；solo = 人操作，AI 只读不抢。',
        human: `主机109#1 模式 collab（AI 主导）`,
        json: `{"ok":true,"ref":"主机109#1","mode":"collab"}`,
      },
      {
        cmd: 'tt debug quit --session <ref>', why: '结束指定调试会话', mode: S, tip: '作业窗口随之关闭；要保留会话用 tt session quit。',
        human: `主机109#1 已结束（作业窗口关闭）`,
        json: `{"ok":true,"ref":"主机109#1","state":"exit"}`,
      },
    ],
  },
  {
    id: 'tzc',
    title: '代码包 .tzc',
    note: '唯一写路径是 apply：围栏 + 三道闸门 + 原子写。export 只读源包。',
    cmds: [
      {
        cmd: 'tt dev tzc export <pkg.tzc> [-o <dir>] [--only <点名>...]', why: '包 → 带围栏的 4GL 工作区', mode: S, tip: '-o 省略时导到 <包所在目录>/<程序名>-ws（身份后缀 (c)/(s) 会去掉）。',
        human: `已导出工作区：D:\\pkg\\capt110-ws
  程序：capt110（Code，ver 1.0）
  Region：10 个（可编辑 3，只读 7）`,
        json: `{"ok":true,"dir":"D:\\\\pkg\\\\capt110-ws","prog":"capt110","points":10,"editable":3,"readonly":7,"sha256":"3ddac577…"}`,
      },
      {
        cmd: 'tt dev tzc status [<dir>]', why: '报告改动（带行号、不写盘）', mode: S, tip: '先看清单再动手；一次改一大片，被拒时定位成本高。',
        human: `改动的点  function.capt110_calc（prog.full.4gl:1234）
新增的点  function.capt110_new（prog.full.4gl:1401）`,
        json: `{"ok":true,"changed_regions":[{"point":"function.capt110_calc","line":1234}],"added_regions":[{"point":"function.capt110_new","line":1401}],"deleted_regions":[]}`,
      },
      {
        cmd: 'tt dev tzc verify [<dir>] [--strict]', why: '不写盘跑 gate1 + gate2', mode: S, tip: '--strict 把 warn 也当失败，CI 用这个。',
        human: `gate1 ok（error=0 warn=0 info=1）   gate2 ok（V1–V7 通过）`,
        json: `{"ok":true,"gate1":{"error":0,"warn":0,"info":1},"gate2":{"violations":[]}}`,
      },
      {
        cmd: 'tt dev tzc apply [<dir>] [-o <pkg>] [--dry-run] [--yes]', why: '三道闸门 + 原子写回包（唯一写路径）', mode: S, tip: '任何一步失败，原包字节不变；--dry-run 只输出逐条目 diff。',
        human: `apply 成功 → D:\\pkg\\capt110(c).tzc
  已备份上一版到 .tdev\\prev.tzc；git 已提交`,
        json: `{"ok":true,"out":"D:\\\\pkg\\\\capt110(c).tzc","sha256":"9a1c…","actions":[{"entry":"capt110.4gl","action":"patch"}],"commit":"a91f2c3"}`,
      },
      {
        cmd: 'tt dev tzc unlock [<dir>] --yes', why: '框架解锁（单向，不可逆）', mode: S, tip: '解开后规格调整不再自动生成程序代码；--yes 必须由人确认，AI 不得自主加。',
        human: `已解锁整个框架（单向、不可逆）。区域/定义点变为可编辑。`,
        json: `{"ok":true,"unlocked":true,"irreversible":true}`,
      },
      {
        cmd: 'tt dev tzc rename [<dir>] <旧名> <新名> [--scope …] [--desc …]', why: '结构事务：改函数名', mode: S, tip: '围栏 fn + 签名行 + 描述块三处原子同步，并在 .tap 里复刻墓碑事务。',
        human: `已改名 adzi999_calc → adzi999_count（围栏 / 签名行 / 描述块 3 处同步）`,
        json: `{"ok":true,"old":"adzi999_calc","new":"adzi999_count","sites":["fence","signature","desc"]}`,
      },
      {
        cmd: 'tt dev tzc newfn [<dir>] --type FUNCTION|DIALOG|REPORT [--name <名>]', why: '新增自定定义点（带模板）', mode: S, tip: '没有 --desc：函数头用设计器模板（含顶部空行），描述自己填。',
        human: `已在 APPEND 锚点插入 FUNCTION adzi999_newfunc（含设计器模板头）`,
        json: `{"ok":true,"name":"adzi999_newfunc","type":"FUNCTION","anchor":"APPEND"}`,
      },
      {
        cmd: 'tt dev tzc selftest', why: '31 项内置对抗用例', mode: H, tip: '全部合成包，不需要真实语料；也是“这套闸门有没有被改坏”的回归入口。',
        human: `ok    export：围栏行数、可编辑区、只读区
ok    gate1：改围栏行被拒（退出码 3）
ok    gate1：改只读区被拒（退出码 4）
…
  ok    tzs export 纯解压：逐字节一致、不产生工作区产物、代码包走 tzs 被指回 tzc

自检：通过 31，失败 0`,
        json: ``,
      },
    ],
  },
  {
    id: 'shape',
    title: '结构与画面',
    note: '把编译产物变成结构：纯函数解析、零 IO、离线可跑。',
    cmds: [
      {
        cmd: 'tt outline <file.4gl>', why: '.4gl → 函数 / UI 块树与行范围', mode: R, tip: '本地文件即可，不需要环境；AI 找代码入口的第一站。',
        human: `FUNCTION c_brace  [2-8]`,
        json: `{"ok":true,"schema":"tt.outline/v1","data":{"kind":"outline.tree","file":"D:\\\\proj\\\\testdata\\\\fgl-fixtures\\\\block-comment-braces.4gl","source":{"mode":"file"},"lineBase":1,"total":1,"returned":1,"nodes":[{"label":"c_brace","kind":"FUNCTION","line":2,"endLine":8,"selStart":9,"selEnd":16,"children":[]}]}}`,
      },
      {
        cmd: 'tt interface show <form> --at <环境>', why: '页面结构（默认不展开字段）', mode: S, tip: '需要该环境的镜像目录（tt file mirror dir / tt file mirror pull）。',
        human: `aapp320  页面 2 个
  page1  「基本资料」  字段 12
  page2  「明细」      字段 9`,
        json: `{"ok":true,"form":"aapp320","pages":[{"name":"page1","title":"基本资料","fields":12},{"name":"page2","title":"明细","fields":9}]}`,
      },
      {
        cmd: 'tt interface pages <form>', why: '所有页面的扁平索引', mode: S, tip: '跨页找东西先看这个，比逐页 show 快。',
        human: `page  title       fields
page1 基本资料    12
page2 明细        9`,
        json: `{"ok":true,"pages":[{"page":"page1","title":"基本资料","fields":12},{"page":"page2","title":"明细","fields":9}]}`,
      },
      {
        cmd: 'tt interface fields <form> [--page <页>] [--col <列名>] [--kw <关键词>]', why: '字段扁平表（可按页/列名/关键词过滤）', mode: S, tip: '中文标签来自 .42s，字段定义来自 .42f —— 两边合起来才是人看到的画面。',
        human: `page   col      field       label
page1  aapp320_01 aapp320001  单别
page1  aapp320_02 aapp320002  单号`,
        json: `{"ok":true,"fields":[{"page":"page1","col":"aapp320_01","field":"aapp320001","label":"单别"}]}`,
      },
      {
        cmd: 'tt drawio compose <spec.json> [-o <文件>]', why: '排版 spec → .drawio 原型图', mode: R, tip: 'spec 里只用形状库的 id；产物人可打开、机器可再生成。',
        human: `已生成 out/ainq120.drawio —— 67 个控件 / 881 个单元格，来自「T100 表单控件库」
（本图全部使用 x/y 绝对定位，未使用 col/row 栅格，无列宽报告）`,
        json: `{"cells":881,"ok":true,"out":"D:\\\\proj\\\\out\\\\ainq120.drawio","report":"（本图全部使用 x/y 绝对定位，未使用 col/row 栅格，无列宽报告）","shapes":67}`,
      },
      {
        cmd: 'tt drawio lib [-o <目录>] [--catalog]', why: '形状库 → drawio 可加载的 mxlibrary', mode: R, tip: '--catalog 不落盘，把清单打到 stdout（给 AI 看的紧凑表）。',
        human: `# T100 组件库 · 形状清单

> 由 tt drawio lib --catalog 现打，不要手抄、不要手改。
> 这是给 AI 用的：拼图时只用下表的 id，不要去读 mxlibrary 文件，也不要自己编 XML。

## business — T100 业务组件库`,
        json: `（--catalog 输出 markdown 表格；落盘模式写出 dist/ 下的两个 .xml）`,
      },
    ],
  },
  {
    id: 'dict',
    title: '字典 dict',
    note: '表 / 字段 / 校验 / 开窗 / 消息 / 参数 / ACC。本地 SQLite 可离线查，也可直查远程库；两条源同一套信封。',
    cmds: [
      {
        cmd: 'tt dict prog <作业|程序编号> --at <环境>', why: '作业 / 程序编号 → 实体程序', mode: S, tip: '作业不等于程序：一个作业可能调多个程序，先 prog 再读源码。',
        human: `${envHead.join('\n')}
# 行数 2（共 2）
作业      程序        类型
bsft001   bsft001_wf  Code
bsft001   bsft001_b   Code`,
        json: `{"ok":true,${envJsonHead},"totalRows":2,"returned":2,"data":[{"job":"bsft001","prog":"bsft001_wf","type":"Code"}]}`,
      },
      {
        cmd: 'tt dict desc <table_name> [field_name]', why: '表 / 字段的结构说明', mode: S, tip: '不带字段名给全表；带字段名只解一个。',
        human: `${envHead.join('\n')}
# 行数 3（共 3）
字段        类型          长度  中文
sfaa001     CHAR          4     单别
sfaa002     CHAR          11    单号`,
        json: `{"ok":true,${envJsonHead},"totalRows":3,"returned":3,"data":[{"field":"sfaa001","type":"CHAR","len":4,"label":"单别"}]}`,
      },
      {
        cmd: 'tt dict r.t [table_name]', why: '列出表', mode: S, tip: '本地库没有该环境数据时先 tt dict db sync。',
        human: `${envHead.join('\n')}
# 行数 2（共 2）
表名     说明
sfaa_t   工单基本资料
sfba_t   工单单身`,
        json: `{"ok":true,${envJsonHead},"totalRows":2,"returned":2,"data":[{"table":"sfaa_t","title":"工单基本资料"}]}`,
      },
      {
        cmd: 'tt dict r.q [开窗码]', why: '开窗（查询）定义', mode: S, tip: '调开窗前确认过滤条件 —— 很多“查不到”是开窗条件带进来的。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
开窗码        程序          说明
q_ooag001_2   ooag001      人员编号（含停用）`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"code":"q_ooag001_2","prog":"ooag001"}]}`,
      },
      {
        cmd: 'tt dict r.v [识别码]', why: '校验带值定义', mode: S, tip: '查“这个字段为什么被拒”，先看它的校验带。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
识别码        字段        校验           值
v_sfaa001    sfaa001    cl_chk_docno   -`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"code":"v_sfaa001","field":"sfaa001","rule":"cl_chk_docno"}]}`,
      },
      {
        cmd: 'tt dict scc [分类码...]', why: '系统分类码', mode: S, tip: 'SCC 是系统级枚举，ACC 是应用分类码，别混。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
分类码   说明          值
1        有效          -
2        无效          -`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"scc":"1","label":"有效"}]}`,
      },
      {
        cmd: 'tt dict acc show <作业|分类>', why: 'ACC 元模型：分类、维护作业、参考字段', mode: S, tip: '先 show 再 query：query 的 spec 字段来自这里。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
作业      分类        维护作业      参考字段
aapt320   aapt320_01  aapt320        imaal003, imaal004`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"job":"aapt320","acc":"aapt320_01","maint":"aapt320","refs":["imaal003","imaal004"]}]}`,
      },
      {
        cmd: 'tt dict acc query <作业|分类> (--template | --spec <文件|->)', why: 'ACC 动态查询（受约束的 spec）', mode: S, tip: '--template 先拿到骨架，改完用 --spec - 从 stdin 回传；不允许自由 SQL。',
        human: `${envHead.join('\n')}
# 行数 2（共 2）
分类码   说明
A001     一般
A002     特殊`,
        json: `{"ok":true,${envJsonHead},"totalRows":2,"returned":2,"data":[{"code":"A001","label":"一般"}]}`,
      },
      {
        cmd: 'tt dict msg [编号[,编号...]]', why: '系统消息（编号 / 语句 / 类型 / 状态 / 建议作业）', mode: S, tip: '客户端弹的 msg 编号，用它查原文与建议动作。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
编号     类型   消息
sub-00022   E    请先查询再执行`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"no":"sub-00022","type":"E","text":"请先查询再执行"}]}`,
      },
      {
        cmd: 'tt dict sysp <编号>[,<编号>...]', why: '系统参数说明', mode: S, tip: 'S-MFG-0055 这种参数的含义与取值，别猜。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
参数          说明                    值
S-MFG-0055    发料套数控管否          Y`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"param":"S-MFG-0055","label":"发料套数控管否","value":"Y"}]}`,
      },
      {
        cmd: 'tt dict docp <编号>[,<编号>...]', why: '单据参数说明', mode: S, tip: '单据级参数（按单别走），与 sysp 是两层。',
        human: `${envHead.join('\n')}
# 行数 1（共 1）
参数             说明               值
D-MFG-0053      发料需求默认方式   1`,
        json: `{"ok":true,${envJsonHead},"totalRows":1,"returned":1,"data":[{"param":"D-MFG-0053","label":"发料需求默认方式","value":"1"}]}`,
      },
      {
        cmd: 'tt dict spill list|show <文件|latest>', why: '被截断查询的落盘全量', mode: R, tip: '截断时信封给 localPath，用 spill show 翻页读全量。',
        human: `（没有落盘过——只有查询被截断时才会落盘）`,
        json: `{"ok":true,"source":"spill","totalRows":0,"returned":0,"notes":["还没有落盘过——只有查询被截断时才会落盘（…\\\\spill 里没有文件）"],"data":[]}`,
      },
    ],
  },
  {
    id: 'tzs',
    title: '表单包 .tzs',
    note: '.tzs 的写路径是设计器自己的引擎（常驻子进程），不是我们拼 XML。动词表由引擎 manifest 给，所以 --help 才是完整说明书。',
    cmds: [
      {
        cmd: 'tt dev tzs export <pkg.tzs> [-o <dir>] [--force]', why: '纯解压（产物只读参考）', mode: R, tip: '注意：这条线只吃 -json（单横线），不认 --human；产物没有 manifest/围栏，不要改完塞回包。',
        human: `（该命令只输出 JSON；加 -json 与不加都一样）`,
        json: `{"bytes":124194,"count":5,"dir":"D:\\\\proj\\\\.tmp\\\\aapp320","entries":[{"name":"aapp320.4fd","size":92258,"sha256":"85c921b3…"},{"name":"aapp320.tsd","size":31420,"sha256":"c8a0101a…"}],"ok":true,"pkg":"testdata/tzs-mini/ws/aapp320(c).tzs","pkg_sha256":"3ddac577…","readonly":true}`,
      },
      {
        cmd: 'tt dev tzs formcheck --form42f <表单.42f> --tzs <包.tzs|表单.4fd>', why: '.42f ↔ 生成物 静态比对', mode: S, tip: '不开引擎、不连服务器、不碰工作区；差异分四组（存在性/编号/可写性/控件族），拿不准的进“未判定”不算错。退出码 0 一致 / 6 有确定差异。',
        human: `（只输出 JSON；退出码 6 表示有确定差异）`,
        json: `{"ok":true,"diffs":{"existence":[],"numbering":[{"col":"aapp320_07","want":"aapp320007","got":"aapp320008"}],"writable":[],"controls":[]},"undecided":[]}`,
      },
      {
        cmd: 'tt dev tzs <动词> --form <程序名> --args \'<JSON>\'', why: '引擎动词：open / save / validate / set_spec_attr …', mode: S, tip: '参数只用 JSON（--args 或 --args-file）；不搬句柄（每次 open 换号）；动词全名见 tt dev tzs --help。',
        human: `（只输出 JSON）`,
        json: `{"ok":true,"form":"aapp320","handle":"h9","verbs":"open/validate/save 均按函数表生成"}`,
      },
      {
        cmd: 'tt dev tzs doctor', why: '环境自检（引擎 / 设计器 / 工作区 / 管道）', mode: R, tip: '引擎、设计器目录、工作区三样少一样，动词全都用不了 —— 先跑这条。',
        human: `（只输出 JSON；--human 会报 flag provided but not defined）`,
        json: `{"checks":[{"name":"引擎 exe","level":"fail","message":"…tzs-server.exe 不存在或读不到"},{"name":"设计器目录","level":"fail","message":"… 不是目录"},{"name":"工作区","level":"fail","message":"未配置（--workspace / TZSCLI_WS / --env 的工作区 三选一）"},{"name":"状态文件","level":"ok","message":"…\\\\.tt-tzs.json（还没有记录）"},{"name":"语料根","level":"ok","message":"D:\\\\t100_wrok_dir（来自 内置缺省）"}]}`,
      },
      {
        cmd: 'tt dev tzs stop', why: '停掉本工作区的常驻引擎', mode: S, tip: '不启动，只停；每个工作区一个引擎。',
        human: `（只输出 JSON）`,
        json: `{"ok":true,"stopped":true,"workspace":"D:\\\\ws\\\\109"}`,
      },
      {
        cmd: 'tt dev tzs reap [--yes]', why: '清理重编后停不掉的孤儿守护进程', mode: S, tip: '引擎重编过、老进程还占着管道时用；--yes 才真杀。',
        human: `（只输出 JSON）`,
        json: `{"ok":true,"reaped":[{"pid":4312,"workspace":"D:\\\\ws\\\\109"}],"skipped":[]}`,
      },
    ],
  },
  {
    id: 'file',
    title: '文件 file',
    note: '看服务器文件（只读，服务端零进程）+ 离线镜像。镜像是一切“本地解析”的前提。',
    cmds: [
      {
        cmd: 'tt file ls [-la] [<路径>] --at <环境>', why: '列目录（默认 $TOP）', mode: S, tip: '默认就是 $TOP，不用记路径。',
        human: `类型  大小      时间               名字
d     -         2026-02-20 11:02  4gl
f     1.2 MB    2026-02-27 09:15  top_global.inc`,
        json: `{"ok":true,"totalRows":2,"returned":2,"data":[{"type":"d","name":"4gl","size":0},{"type":"f","name":"top_global.inc","size":1258291}]}`,
      },
      {
        cmd: 'tt file find <路径...> [-name G] [-type f|d|l] [-maxdepth N]', why: '按名字 / 类型找文件', mode: S, tip: '只读元数据，服务端零进程 —— 比在服务器上跑 find 便宜。',
        human: `$TOP/4gl/asf/asfp310.4gl
$TOP/4gl/asf/asfp311.4gl`,
        json: `{"ok":true,"totalRows":2,"returned":2,"data":[{"path":"/u1/top/4gl/asf/asfp310.4gl"}]}`,
      },
      {
        cmd: 'tt file stat <路径...>', why: '类型 / 大小 / 时间 / 权限', mode: S, tip: '多路径一次给完。',
        human: `f  395 KB  2026-02-27 09:15  -rw-r--r--  /u1/top/4gl/asf/asfp310.4gl`,
        json: `{"ok":true,"data":[{"type":"f","size":404864,"mode":"-rw-r--r--","path":"/u1/top/4gl/asf/asfp310.4gl"}]}`,
      },
      {
        cmd: 'tt file cat <文件...>', why: '输出内容（多文件直接拼接）', mode: S, tip: '读整份文件用它；找内容用 grep，大文件别 cat。',
        human: `# 程序 asfp310  整批發料作業
MAIN
   CALL cl_ap_init("asf","")`,
        json: `{"ok":true,"data":{"path":"/u1/top/4gl/asf/asfp310.4gl","content":"# 程序 asfp310…"}}`,
      },
      {
        cmd: 'tt file head|tail [-n N] <文件...>', why: '前 / 后 N 行（默认 10）', mode: S, tip: '先 head 看结构，再决定要不要整份取。',
        human: `# 程序 asfp310  整批發料作業
{<section id="asfp310.description" type="s" >}
#應用 a00 樣板自動產生(Version:3)`,
        json: `{"ok":true,"data":{"path":"/u1/top/4gl/asf/asfp310.4gl","lines":["# 程序 asfp310  整批發料作業","…"]}}`,
      },
      {
        cmd: 'tt file grep [-rinl] [-m N] [--include G] <模式> <路径...>', why: '找内容（RE2，服务端零进程）', mode: S, tip: '服务端把文件流回来在我们这侧匹配 —— 不装工具、不落进程。',
        human: `/u1/top/4gl/asf/asfp310.4gl:33:#171205-00054#1 效能優化
/u1/top/4gl/asf/asfp310.4gl:812:      CALL bsft001_b_fill()`,
        json: `{"ok":true,"totalRows":2,"returned":2,"data":[{"path":"/u1/top/4gl/asf/asfp310.4gl","line":33,"text":"#171205-00054#1 效能優化"}]}`,
      },
      {
        cmd: 'tt file wc|du <文件...>', why: '行 / 词 / 字节、占用大小', mode: S, tip: 'du -s 只给合计。',
        human: `  8556   40211  395603  /u1/top/4gl/asf/asfp310.4gl`,
        json: `{"ok":true,"data":[{"path":"/u1/top/4gl/asf/asfp310.4gl","lines":8556,"words":40211,"bytes":395603}]}`,
      },
      {
        cmd: 'tt file get <路径> [-o <本地|->]', why: '下载一个文件到本地', mode: S, tip: '-o - 直接吐到 stdout（管道用）。',
        human: `已下载 /u1/top/4gl/asf/asfp310.4gl → ./asfp310.4gl（395 KB）`,
        json: `{"ok":true,"from":"/u1/top/4gl/asf/asfp310.4gl","to":"./asfp310.4gl","bytes":404864}`,
      },
      {
        cmd: 'tt file mirror pull [<环境>] [--full]', why: '同步本地镜像（默认增量）', mode: S, tip: 'default(source) 收四类：4gl/**/*.4gl + 42f + 42s/zh_CN + *.inc；interface 只 42f/42s；--full 全量重建（只对本次 profile 的白名单负责）。',
        human: `镜像同步完成：D:\\t100_mirror\\主机109
  source     新增 12 / 更新 3 / 删除 1（共 4210 个文件：4gl/42f/42s/inc）
  interface  新增 2 / 更新 0`,
        json: `{"ok":true,"dir":"D:\\\\t100_mirror\\\\主机109","source":{"added":12,"updated":3,"deleted":1,"total":4210},"interface":{"added":2}}`,
      },
      {
        cmd: 'tt file mirror path [<环境>]', why: '打印镜像目录（AI 可直接去读）', mode: S, tip: '没设镜像目录会报错并给出该跑哪条命令 —— 这是刻意的。',
        human: `D:\\t100_mirror\\主机109`,
        json: `{"ok":true,"env":"主机109","dir":"D:\\\\t100_mirror\\\\主机109"}`,
      },
      {
        cmd: 'tt file mirror dir [<目录>] [--at <环境>]', why: '查看 / 设置镜像目录', mode: S, tip: '可以只给某个环境单独设；不给 --at 是全局根目录。',
        human: `镜像目录：D:\\t100_mirror（全局根目录；各环境落在 <根>\\<环境>）`,
        json: `{"ok":true,"dir":"D:\\\\t100_mirror","scope":"global"}`,
      },
    ],
  },
]

// ---------- 渲染 ----------
const esc = (s) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
const slug = (s) => s.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

let total = 0
const nav = []
const body = []

groups.forEach((g) => {
  const cards = []
  g.cmds.forEach((c) => {
    total++
    const id = slug(c.cmd.replace(/^tt /, 'tt-'))
    cards.push(renderCard(c, id))
  })
  nav.push({ id: g.id, title: g.title, n: g.cmds.length })
  body.push(`<section class="grp" id="${g.id}">
  <h2>${esc(g.title)} <span class="cnt">${g.cmds.length}</span><button class="gtog" data-g="${g.id}">展开本组</button></h2>
  <p class="note">${esc(g.note)}</p>
  ${cards.join('\n  ')}
</section>`)
})

function renderCard(c, id) {
  const badge = c.mode === R ? '<span class="badge real">实跑</span>'
    : c.mode === H ? '<span class="badge half">部分实跑</span>'
      : '<span class="badge shape">示意</span>'
  const blocks = []
  if (c.human) blocks.push(block('human 输出', c.human, 'human'))
  if (c.json) blocks.push(block('json 输出（--json）', c.json, 'json'))
  const tip = c.tip ? `<p class="tip">${esc(c.tip)}</p>` : ''
  return `<details class="c" id="${id}" data-k="${esc((c.cmd + ' ' + c.why + ' ' + (c.tip || '')).toLowerCase())}">
  <summary>${badge}<code>${esc(c.cmd)}</code><span class="why">${esc(c.why)}</span></summary>
  <div class="in">
    ${tip}
    ${block('输入', c.cmd, 'sh')}
    ${blocks.join('\n    ')}
  </div>
</details>`
}

function block(label, text, kind) {
  return `<div class="blk ${kind}"><div class="lb">${esc(label)}</div><pre>${esc(text)}</pre></div>`
}

const html = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>TT 命令参考 · 示例手册</title>
<style>
:root{--ink:#111827;--mut:#6b7280;--line:#e5e7eb;--accent:#2563eb;--bg:#fff;--soft:#fafafa}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;background:var(--bg);color:var(--ink);
 font:14px/1.6 system-ui,"Segoe UI","Microsoft YaHei",sans-serif}
code,pre{font-family:ui-monospace,Consolas,"Courier New",monospace}
a{color:var(--accent);text-decoration:none}
.wrap{display:flex;align-items:flex-start;gap:0}
/* 侧栏 */
aside{position:sticky;top:0;height:100vh;width:250px;flex:0 0 250px;overflow:auto;
 border-right:1px solid var(--line);padding:22px 18px 40px}
aside h1{font-size:15px;margin:0 0 4px}
aside .sub{color:var(--mut);font-size:12px;margin:0 0 18px}
aside nav a{display:flex;justify-content:space-between;gap:8px;padding:5px 8px;border-radius:5px;
 color:var(--ink);font-size:13px}
aside nav a:hover{background:var(--soft)}
aside nav a .n{color:var(--mut);font-size:11px}
aside .lg{margin-top:20px;padding-top:14px;border-top:1px solid var(--line);font-size:11px;color:var(--mut)}
aside .lg div{margin:4px 0}
/* 正文 */
main{flex:1;min-width:0;padding:26px 34px 120px;max-width:1000px}
header.top{display:flex;flex-wrap:wrap;align-items:center;gap:10px;margin-bottom:6px}
header.top h1{font-size:22px;margin:0;font-weight:600}
.lead{color:var(--mut);font-size:13px;margin:0 0 18px}
.bar{position:sticky;top:0;z-index:5;background:var(--bg);padding:10px 0 12px;
 border-bottom:1px solid var(--line);display:flex;gap:8px;flex-wrap:wrap;align-items:center;margin-bottom:8px}
.bar input{flex:1;min-width:200px;padding:7px 10px;border:1px solid var(--line);border-radius:6px;
 font:inherit;font-size:13px;background:var(--bg);color:var(--ink)}
.bar input:focus{outline:2px solid #dbeafe;border-color:var(--accent)}
.bar button{padding:6px 11px;border:1px solid var(--line);background:var(--bg);border-radius:6px;
 font:inherit;font-size:12px;color:var(--ink);cursor:pointer}
.bar button:hover{border-color:#9ca3af}
.hits{color:var(--mut);font-size:12px}
/* 分组 */
.grp{margin:30px 0 0;scroll-margin-top:56px}
.grp h2{font-size:15px;font-weight:600;margin:0 0 4px;padding-bottom:6px;border-bottom:1px solid var(--ink)}
.gtog{float:right;margin-top:1px;padding:2px 8px;border:1px solid var(--line);background:var(--bg);
 border-radius:5px;font:inherit;font-size:11px;color:var(--mut);cursor:pointer}
.gtog:hover{color:var(--ink);border-color:#9ca3af}
.grp h2 .cnt{color:var(--mut);font-weight:400;font-size:12px;margin-left:6px}
.grp .note{color:var(--mut);font-size:12px;margin:8px 0 14px}
/* 命令卡 */
details.c{border-bottom:1px solid var(--line);padding:2px 0}
details.c[open]{background:var(--soft)}
details.c>summary{display:flex;align-items:baseline;gap:10px;padding:9px 6px;cursor:pointer;list-style:none}
details.c>summary::-webkit-details-marker{display:none}
details.c>summary::before{content:"▸";color:#9ca3af;font-size:11px;flex:0 0 12px}
details.c[open]>summary::before{content:"▾"}
details.c>summary code{font-size:12.5px;color:var(--ink);white-space:nowrap}
details.c>summary .why{color:var(--mut);font-size:12.5px;margin-left:auto;text-align:right}
.badge{font-size:10px;padding:1px 6px;border-radius:99px;border:1px solid var(--line);color:var(--mut);
 flex:0 0 auto;white-space:nowrap}
.badge.real{color:#0f766e;border-color:#99f6e4;background:#f0fdfa}
.badge.half{color:#92400e;border-color:#fde68a;background:#fffbeb}
.badge.shape{color:#6b7280;background:#fafafa}
.in{padding:4px 6px 18px 28px}
.tip{color:var(--ink);background:#fff;border-left:2px solid var(--accent);padding:6px 10px;margin:6px 0 12px;font-size:12.5px}
.blk{margin:10px 0 14px}
.lb{font-size:11px;color:var(--mut);margin-bottom:4px}
pre{margin:0;padding:11px 13px;background:#fff;border:1px solid var(--line);border-radius:6px;
 font-size:12px;line-height:1.55;white-space:pre-wrap;word-break:break-word;overflow-x:auto}
.blk.sh pre{border-color:#cbd5e1}
.blk.json pre{background:#fcfcfd}
.blk.human pre{background:#fffdf7;border-color:#f3e8d0}
.hide{display:none!important}
footer{max-width:1000px;padding:0 34px 60px;color:var(--mut);font-size:12px}
@media (max-width:820px){
 .wrap{flex-direction:column}
 aside{position:static;height:auto;width:auto;flex:none;border-right:0;border-bottom:1px solid var(--line)}
 main{padding:20px 16px 80px}
}
@media print{
 aside,.bar{display:none}
 details.c{border-bottom:1px solid #ddd}
 details.c>summary::before{content:""}
}
</style>
</head>
<body>
<div class="wrap">
<aside>
  <h1>TT 命令参考</h1>
  <p class="sub">${total} 条命令 · 默认折叠</p>
  <nav>
    ${nav.map((g) => `<a href="#${g.id}">${esc(g.title)}<span class="n">${g.n}</span></a>`).join('\n    ')}
  </nav>
  <div class="lg">
    <div><span class="badge real">实跑</span> 输出是本机真实抓取</div>
    <div><span class="badge half">部分实跑</span> human 可证，json 未抓</div>
    <div><span class="badge shape">示意</span> 需真实环境，形状示意</div>
    <div style="margin-top:12px;line-height:1.9">通用开关<br><code>--json</code> · <code>--human</code> · <code>--csv</code><br><code>--at 环境@企业</code> · <code>--session</code></div>
  </div>
</aside>
<main>
  <header class="top"><h1>TT 命令参考</h1></header>
  <p class="lead">与 <a href="commands.svg">命令地图</a> 同一套分组。点开一条看输入与两种输出；细节永远在 <code>tt &lt;组&gt; &lt;动词&gt; --help</code>。</p>

  <div class="bar">
    <input id="q" type="search" placeholder="过滤命令 / 用途 / 说明（按 / 聚焦）" autocomplete="off">
    <button id="expand">展开全部</button>
    <button id="collapse">收起全部</button>
    <span class="hits" id="hits"></span>
  </div>

  ${body.join('\n  ')}

  <footer>
    信封字段：<code>ok</code> · <code>source</code> · <code>env</code> · <code>zone</code> · <code>ent</code> · <code>account</code> · <code>dialect</code> · <code>target</code> · <code>route</code> · <code>readonly</code> · <code>totalRows</code> · <code>returned</code> · <code>truncated</code> · <code>localPath</code> · <code>notes</code> · <code>data</code>；
    失败信封：<code>ok=false</code> + <code>code</code> + <code>error</code> + <code>hint</code> + <code>exitCode</code>。
  </footer>
</main>
</div>
<script>
(function(){
  var cards = Array.prototype.slice.call(document.querySelectorAll('details.c'));
  var q = document.getElementById('q');
  var hits = document.getElementById('hits');

  function apply() {
    var s = q.value.trim().toLowerCase();
    var shown = 0;
    cards.forEach(function (d) {
      var hit = !s || d.getAttribute('data-k').indexOf(s) >= 0;
      d.classList.toggle('hide', !hit);
      if (hit) shown++;
    });
    document.querySelectorAll('section.grp').forEach(function (sec) {
      var any = sec.querySelectorAll('details.c:not(.hide)').length > 0;
      sec.classList.toggle('hide', !any);
    });
    hits.textContent = s ? (shown + ' / ' + cards.length) : '';
  }

  q.addEventListener('input', apply);
  document.addEventListener('keydown', function (e) {
    if (e.key === '/' && document.activeElement !== q) { e.preventDefault(); q.focus(); }
    if (e.key === 'Escape' && document.activeElement === q) { q.value = ''; apply(); q.blur(); }
  });

  document.querySelectorAll('.gtog').forEach(function(b){
    b.onclick = function(){
      var sec = document.getElementById(b.getAttribute('data-g'));
      var open = b.textContent.indexOf('展开') >= 0;
      sec.querySelectorAll('details.c').forEach(function(d){ d.open = open; });
      b.textContent = open ? '收起本组' : '展开本组';
    };
  });

  document.getElementById('expand').onclick = function(){ cards.forEach(function(d){d.open=true}); };
  document.getElementById('collapse').onclick = function(){ cards.forEach(function(d){d.open=false}); };

  function fromHash() {
    var id = location.hash.slice(1);
    if (!id) return;
    var el = document.getElementById(id);
    if (el && el.tagName === 'DETAILS') { el.open = true; el.scrollIntoView({block:'center'}); }
  }
  window.addEventListener('hashchange', fromHash);
  fromHash();
})();
</script>
</body>
</html>
`
fs.writeFileSync("commands.html", html)

