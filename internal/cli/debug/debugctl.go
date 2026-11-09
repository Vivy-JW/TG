package debug

// debug 命令行控制端:薄封装,直连 tt debug serve 的 REST。
// 标准 fgldb 命令用 exec 透传(原样返回输出),这里只封装 serve 没有的生命周期动作:
// SSH 连接 + 指定程序启动调试(start)、接口日志获取(wslogs)、按日志重放调试(wsdebug)。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tt/internal/cli/common"
	cdb "tt/internal/cli/db"
	"tt/internal/debug"
	"tt/internal/debugclient"
	"tt/internal/target"

	"github.com/spf13/cobra"
)

var (
	dbgAPIURL  string
	dbgTimeout int
	dbgZone    string
	dbgSSHName string
	wsService  string
	wsOrigin   string
	wsServer   string
	wsResult   string
	wsPID      string
	wsJob      string
	wsFrom     string
	wsTo       string
	wsOnlyFail bool
	wsPage     int

	dbgSoftWait int    // exec/step 的软等待秒数(0=不启用,走原来的硬超时)
	dbgExecFile string // exec --file:从文件读多条命令(一行一条)
	dbgWaitFor  string // wait 命令:等哪些事件
	dbgNoResume bool   // why 命令:探测完不自动放回运行

	dbgExecMax int    // exec --max:单条命令**显示**的行数上限(0=不限;不影响本地完整副本)
	dbgSQLFile string // sql --file:从本地文件读语句(不是远端 @)
	dbgSQLEnt  int    // sql --ent:显式指定企业编号(不给 = 用这条会话当前生效的企业编号)
	dbgEnvFlag string // start/wsdebug --env:这次启动要连哪个环境(权威;空=回落全局 --env/当前环境)
	// dbgEntRun 是 start/wsdebug --ent：**这一轮跑在哪个企业**。
	//
	// 必填（服务端会拒掉空的）：从前这里没有这个标志，企业编号只能靠"起会话后 tt debug topent"
	// 或者干脆沿用登录 shell 的 $TOPENT —— 后者是静默的，而 ENT 决定用哪个账号连库、以及
	// 据点校验过不过，"没说"与"说错了"在老实现里长得一模一样。
	dbgEntRun string
	// dbgFGLServer 是 start/wsdebug --fglserver:这一轮作业画面画到哪儿(FGLSERVER)。
	// 空串有语义("本轮交回 T100 自动设置"),所以判定一律用 cmd.Flags().Changed,
	// 绝不能写成 `if dbgFGLServer != ""`。
	dbgFGLServer string

	wsShow         string   // wslogs --show <rowid>:只看这一条的报文
	wsSaveReq      string   // wslogs --show --save-request:把请求原文另存一份(改完再 --request-file)
	wsDebugSet     []string // wsdebug --set k=v:改一个入参再重放(可重复)
	wsDebugReqFile string   // wsdebug --request-file:整份替换入参报文
)

// dbgAPIBase 返回调试 API 的根地址(含挂载前缀)。
//
// 两种服务把调试面挂在不同地方:独立调试服务在根(/api/…),统一服务在
// /debug/api/…。前缀记在状态文件里(见 servebg.go 的 ServeInfo.APIBase),
// 所以控制端命令不必知道对面是哪个服务。
func dbgAPIBase() string {
	if dbgAPIURL != "" {
		// 显式给了 --url：按原样用（自己带了 /api 就不补前缀，加前缀就补）。
		return joinAPIBase(dbgAPIURL, apiBaseFromState())
	}
	return debugclient.Discover(configDir(), configListen())
}

// joinAPIBase 给服务地址补上挂载前缀。URL 自己已经带了 API 路径的(如 …/debug/api)
// 原样返回 —— 否则 --url 显式指到 API 上时会被再拼一次前缀。
func joinAPIBase(url, prefix string) string {
	u := strings.TrimRight(url, "/")
	if strings.HasSuffix(u, "/api") {
		return u
	}
	return u + prefix
}

// actorSelf 这次调用算谁的动作：TT_ACTOR 环境变量优先（serve 起"人点的按钮"那条链路时
// 会传 human），否则按 ai 算 —— 命令行本来就是 AI 的接口。取值与 internal/debug 的
// actorHeader 同一套词汇（ai / human）。常量取自 internal/target，不在命令层另立一份。
func actorSelf() string {
	if v := strings.TrimSpace(os.Getenv(target.ActorEnv)); v != "" {
		return v
	}
	return "ai"
}

// dbgAPI 调 serve REST;非 2xx 时解析 {"error": ...} 返回错误
// dbgAPI 调 serve REST;非 2xx 时由客户端把失败信封翻成 `*debugclient.Failure`
// （码 / 候选 / hint / 退出码都在那一层，见 internal/debugclient）。
//
// 这里**不再自己实现一遍 HTTP**：其他命令也走同一个客户端，一套规矩。
// 从前各写一份的后果是"同一个失败在两条线上有不同的码与提示"。
func dbgAPI(method, path string, body any) ([]byte, error) {
	return dbgClient().Do(method, path, body)
}

// dbgClient 按当前配置目录算出调试面客户端（地址与前缀来自服务状态文件）。
func dbgClient() *debugclient.Client {
	return debugclient.New(dbgAPIBase())
}

func dbgJSONOut(v any) error {
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(v)
}

// 会话 id 的来源已经换成"向目标层要目标"（见 target.go 的 dbgSessionID）：
// 从前这里是"列一次会话、取回包数组第一项"的那个函数（契约 B10 点名的就是它）。
// 取第一项的写法多会话下会静默跑到另一条会话上，所以它整个被取代，不留副本，

// dbgWaitStopped 等待会话停站(入口停站/断点命中)。
// 走服务端长轮询 /wait:事件一到立刻返回,不再每 2s 查一次快照。
func dbgWaitStopped(id string, timeout time.Duration) (map[string]any, error) {
	q := url.Values{}
	q.Set("for", "stopped,exit")
	q.Set("timeout", strconv.Itoa(int(timeout.Seconds())))
	data, err := dbgAPI("GET", dbgSessionURL(id, "/wait?"+q.Encode()), nil)
	if err != nil {
		return nil, err
	}
	var w struct {
		TimedOut bool `json:"timedOut"`
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	if w.TimedOut {
		return nil, fmt.Errorf("等待停站超时(%s)", timeout)
	}
	// 再取一次完整快照当返回值(调用方按 snapshot 结构解析、由 start 原样打印)
	snap, err := dbgAPI("GET", dbgSessionURL(id, ""), nil)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(snap, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// dbgTargetEnv 这次启动要连哪个环境。
//
// 两条来源,命令自己的 --env 优先:
//   - `tt debug start x --env 恒烁正式区`(写在子命令上,语义最明确);
//   - `tt --env 恒烁正式区 debug start x`(根命令的持久开关,别处能用,这里也认)。
//
// 从前这里**只看 --ssh**,而 --env 是根命令的开关、debug 组里没人读 —— 于是
// `tt debug start … --env X` 既不报错也不生效,方向永远取服务端当前环境。同机多环境
// (同库不同企业)时那是"上错号"而不是"报错",所有按企业编号取数的查询返回 0 行,
// 看起来和"配置缺失"一模一样(实测踩过,结论会完全反向)。
func dbgTargetEnv() string {
	if v := strings.TrimSpace(dbgEnvFlag); v != "" {
		return v
	}
	return strings.TrimSpace(common.Env)
}

// dbgSnapView 会话快照里 CLI 侧用到的字段。
//
// **字段形状从 `target.Session` 派生**（契约 C5.17 / C5.18 / C5.19：客户端不许再手抄
// 第二份事实）：`id` / `env` / `state` / `prog` / `module` / `runProg` / `ref` /
// `display` / `name` / `n` / `epoch` 这些键的**唯一声明处**是 `internal/target`，
// 这里用嵌入把它们的名字与 JSON 标签都继承过来 —— 从前这里手抄过一份
// （`dbgSnapView` 的 struct 标签），服务端加一个键就要有人记得改两处。
//
// 留下来的字段是**服务端有、目标层不认**的那些（目标层只认"哪条会话"，
// 不认"这条会话现在什么样"）：`reused` 是客户端自己补的复用痕迹，
// `zone` / `topent*` / `fglserver` / `screenTarget` / `stop` 是快照里给人看的部分。
type dbgSnapView struct {
	target.Session

	Reused   bool   `json:"reused"`
	Zone     string `json:"zone"`
	Topent   string `json:"topent"`
	TopentCf string `json:"topentCfg"`
	TopentSh string `json:"topentShell"`
	// TopentSession 是**本会话显式给的企业编号**（`tt debug start --ent` / 面板输入；
	// 服务端的 `topentSession` 键，见 `Session.TopentSession`）。
	//
	// 它**只是**快照里既有的一栏，有它自己的含义 —— **不是** `ent` 的备胎：
	// `<ent>` 那一栏的唯一来源是快照的 `ent`（服务端 `SessionBrief.Ent`，
	// 契约 C5.1 / C5.3），见 `dbgIdentity`。上一批这里曾经被用来兜底 `ent`
	// （`dbgIdentityEnt`），那处兼容层已按验收判定删净：客户端不许在 `ent` 与
	// `topentSession` 之间做取舍（C5.17–C5.19）。
	TopentSession string `json:"topentSession"`
	// FGLServer 本轮 FGLSERVER 覆盖值(空 = 由 T100 按 SSH 来源 IP 自动设置);
	// ScreenTarget 是服务端给的同一件事的人话版本(见 debug.ScreenTarget)。
	FGLServer    string `json:"fglserver"`
	ScreenTarget string `json:"screenTarget"`
	Stop         *struct {
		File   string `json:"file"`
		Line   int    `json:"line"`
		Func   string `json:"func"`
		Reason string `json:"reason"`
	} `json:"stop"`
}

// dbgIdentity 会话的"我在哪儿"（契约 C5.9：**会话级命令唯一的身份行**）。
//
// 两行，格式逐字对着 C5.9：
//
//	会话 <display> · <name> · <state> · ENT <ent>
//	  账号 <acct> · 库 <target> · 区域 <zone>
//
// 第一行的渲染**不在本文件**：它是 `internal/target.IdentityLine`（C5.9 的唯一实现，
// 与候选行 `CandidateLine` 同处一个包，两者的字段规则才不会各写一遍）。
// 本函数只负责第二行那三样 —— 它们来自配置与企业快照，不是会话的身份键。
//
// 为什么非要有这一行:同一台机器上可以有多个环境指向**同一个库**(只有企业编号不同),
// 那时 topent/topentCfg/topentShell 三个裸字段要么全空、要么同值,人根本看不出这次
// 连的是哪个账号。账号由企业编号经 gzou_t 解析而来,查不到数据时"上错号"与"配置真的
// 缺失"在输出上一模一样 —— 这一行就是用来分开这两件事的。
//
// **C8.4 登记的那处单会话输出变化就是这里**：从
// `生效身份: 环境 … / 区域 … / 企业 … / 账号 … / 库 …` 换成上面两行（H2 的那一处例外，
// 表征测试 `TestDBGIdentity*` 同步改了）。
//
// 账号走 LookupEnts(Offline):只读快照、**一次网络都不发**,没有快照就只回"查不到" ——
// 这是输出里的一句说明,不值得为它拨一次 SSH(慢时 40 秒)。查不到不算错误,
// 但会说明为什么没有,并给出自己核对的命令。
func dbgIdentity(snap *dbgSnapView) string {
	if snap == nil {
		return "" // 没有快照时返回空串(不能打出半行)
	}
	// 第一行：C5.9 的身份行。`display` 取不到时（服务端没给这个键）退回 ref/句柄，
	// 那是 `target.Session.DisplayForm` 的兜底，不是这里再拼一遍。
	//
	// `<ent>` **直接用快照里的 `ent`**（服务端 `SessionBrief.Ent` = `Session.CurrentEnt()`，
	// C5.1 / C5.3）。这里**没有第二份取值链**：客户端不在 `ent` 与 `topentSession` 之间
	// 做取舍（C5.17–C5.19）—— 上一批那处兼容兜底已删净，`ent` 只有这一个来源。
	line := target.IdentityLine(snap.Session)

	// 第二行：账号 / 库 / 区域（`?` 与 `(未配置)` 的取舍见 C5.9 与 orPlain）。
	env := snap.Env
	if env == "" {
		env = "(未知)"
	}
	entForAcct := firstNonEmpty(snap.TopentSh, snap.TopentCf, snap.Topent)
	tgt, acct, acctWhy := dbgDBIdentity(env, entForAcct)
	second := fmt.Sprintf("  账号 %s · 库 %s · 区域 %s", acct, tgt, orPlain(snap.Zone))
	if acctWhy != "" {
		second += "  " + acctWhy
	}
	if snap.Reused {
		// 复用痕迹挂第二行末尾：身份行是 C5.9 定死的四字段格式，不许往里塞第五段。
		second += "  ← 复用了已登录的宿主(未重新连接)"
	}
	return line + "\n" + second
}

func orPlain(v string) string {
	if strings.TrimSpace(v) == "" {
		return "(未配置)"
	}
	return v
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// dbgDBIdentity 该环境挂的库 + 企业编号对应的账号(尽力而为,不连网优先)。
func dbgDBIdentity(env, entRaw string) (target, acct, why string) {
	path, err := resolveConfigPath()
	if err != nil {
		return "-", "-", "（读配置失败:" + err.Error() + "）"
	}
	cfg, err := debug.LoadConfig(path)
	if err != nil {
		return "-", "-", "（读配置失败:" + err.Error() + "）"
	}
	if env != "" && env != "(未知)" {
		if c2 := cfg.CloneEnv(env); c2 != nil {
			cfg = c2
		}
	} else {
		cfg.ApplyDefaultEnv()
	}
	target = "-"
	if cfg.DB != nil {
		target = cfg.DB.Address()
	}
	cfg.DataDir = filepath.Dir(path)
	ent := 0
	if n, err := strconv.Atoi(strings.TrimSpace(entRaw)); err == nil {
		ent = n
	}
	// Offline:有快照就照常给,没快照就只回"查不到" —— 这是输出里的一句说明,
	// 不值得为它拨一次 SSH(慢时 40 秒),更不该因为它没查到就把启动判成失败。
	l, err := debug.LookupEnts(cfg, debug.EntListOpt{Ent: ent, Offline: true})
	if err != nil || l == nil {
		return target, "-", "（账号未知:没有该环境的企业快照 —— 自己核一下:tt debug ents --env " + env + "）"
	}
	switch {
	case l.Current != nil && l.Current.Resolved:
		acct = l.Current.Account
	case entRaw == "":
		acct, why = "-", "（该环境没配 topent,账号由登录默认决定;用 tt debug ents 核对）"
	default:
		acct, why = "-", "（企业 "+entRaw+" 没解析出账号;用 tt debug ents 核对）"
	}
	return target, acct, why
}

// dbgVerifyEnv 把 --env 与实际生效环境对齐,不一致时**明说**。
//
// 这是"报了名就必须认账"的落点:点名了环境却没生效,不能静默 —— 静默回落与上错号
// 在输出上完全无法区分。返回 false 表示点名的环境确实没生效(调用方按失败处理)。
func dbgVerifyEnv(want string, snap *dbgSnapView) bool {
	if want == "" || snap == nil {
		return true
	}
	if snap.Env == want {
		return true
	}
	fmt.Fprintf(os.Stderr, "⚠ --env %s 未生效:实际会话环境是 %s\n", want, orPlain(snap.Env))
	fmt.Fprintf(os.Stderr, "  自己核一下:tt debug env            (列出环境与当前会话)\n")
	fmt.Fprintf(os.Stderr, "              tt debug start %s --env <环境名>   (点名必须与环境清单里的名字一致)\n", snap.Prog)
	return false
}

// debugStartCmd 封装:SSH 连接 + 对指定程序启动调试,等到入口停站
var debugStartCmd = &cobra.Command{
	Use:   "start <作业编号>",
	Short: "连接 SSH 并对指定程序启动调试,等到入口停站",
	Long: `封装"连服务器 + fglrun -d 启动作业 + 等入口停站"全流程。
之后用 tt debug exec 透传 fgldb 标准命令调试(参考 Genero 调试命令文档)。

--env <环境名> 指定这次连哪个环境(等价 --ssh);不给则用服务端当前环境。
点名了环境而实际没生效时会**明确告警**——同一台机器上可以有多个环境指向同一个库
(只有企业编号不同),那种情况下"上错了号"与"配置真的缺失"表现完全一样:
按企业编号取的数全返回 0 行。输出末尾的「生效身份」一行就是用来分开这两件事的。

--fglserver <ip:port> 覆盖**这一轮**的 FGLSERVER(作业界面画到哪),不动配置文件。
默认由 T100 按 SSH 来源 IP 自动设置,也就是回你本机的 GDC;要做 无头前端
(无头前端,听 7700)之类的实验时用 --fglserver <本机IP>:1300 传这一轮,
别再去改全局配置 —— 改了之后每一次放行都不会有画面,而且启动输出不会提这件事。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// 目标先判一次：--session 给了就在这里落位（F3–F8 的错当场报，不发那个 POST），
		// 落上的那一条随后**随请求体送出去** —— 从前这里只校验、不送（Bug A），
		// 于是服务端仍按"环境+名字"收窄候选，同环境两条空闲会话时照旧报歧义，
		// --help 里写着的这个标志等于帮不上忙。--name 的取值纪律也在发请求之前判（F19）。
		host, err := dbgResolveSessionFlag()
		if err != nil {
			return err
		}
		if err := dbgValidateNameFlag(); err != nil {
			return err
		}
		body := map[string]any{"prog": args[0]}
		if dbgModule != "" {
			body["module"] = dbgModule
		}
		// `--ent` 是**这一轮的企业**，随请求体送过去：服务端把它落到本轮 cfg.Topent，
		// 再 export 进会话 shell。空值不送 —— 由服务端拒绝（不再静默沿用登录 shell 的 $TOPENT）。
		if e := strings.TrimSpace(dbgEntRun); e != "" {
			body["ent"] = e
		}
		// 显式指名的宿主：只送**服务端原始句柄**（C5.4：路径与请求体只认 id，不认 ref/display）。
		// 空值不送 —— 没打 --session 时请求体与今天逐字相同（H2）。
		if host != nil {
			body["session"] = host.ID
		}
		// `--name` 是**服务端的事实**（契约 C5.3 的 `name` 键，WP4c 落地）：别名要参与寻址
		// 与 Q23 的幂等键判定，就必须随这次起会话送过去。空串不送（服务端按未命名处理，
		// 与今天逐字相同）。注意别名**只在创建时给一次**（C2.5b）：这里送的是"这条会话叫什么"，
		// 不是"给已有会话改名"。
		if n := strings.TrimSpace(dbgNameFlag); n != "" {
			body["name"] = n
		}
		targetEnv := dbgWireOverrides(cmd, body)
		// 只点名了会话、没点名环境：**环境由那一条会话决定**（它是那条会话自己的目标）。
		// 不补这一句的话，服务端会用"它自己当前的环境"去组装本轮配置 —— 点名恒一#1 却把
		// 这一轮落到另一个环境上，还会把那条会话判成"换目标"（服务端那一档对指名的会话
		// 是报错，见 namedHostConflict）。空值不送：没打 --session 时这里一个字节都不加（H2）。
		if host != nil && targetEnv == "" {
			if env := strings.TrimSpace(host.Env); env != "" {
				body["ssh"] = env
				targetEnv = env // 末尾那句 dbgVerifyEnv 要拿它核对 --env 有没有真的生效
			}
		}
		fglChanged := cmd.Flags().Changed("fglserver")
		// 复用痕迹:启动前记下那个空闲会话,回包 sessionId 与它相同就是"复用了宿主"。
		// 不标出来的话,"我并没有真正新建会话"这件事在输出里毫无痕迹(反馈 S1-④)。
		//
		// 这一步是**尽力而为**的（dbgAmbientSession）：它是痕迹、不是目标 —— 多会话下
		// 落不上目标就什么都不标，绝不把一条本来能成功的 start 判失败。
		prevID := ""
		if ambient := dbgAmbientSession(); ambient != nil && ambient.State == target.StateIdle {
			prevID = ambient.ID
		}
		data, err := dbgAPI("POST", "/api/sessions", body)
		if err != nil {
			return err
		}
		var r struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(data, &r)
		reused := dbgReusedHost(prevID, r.SessionID)
		raw, err := dbgWaitStopped(r.SessionID, time.Duration(dbgTimeout)*time.Second)
		if err != nil {
			return err
		}
		snap := dbgSnapOf(raw)
		if snap != nil {
			snap.Reused = reused
		}
		// 点了 --fglserver 就要认账(旧服务端不认这个字段时是静默忽略的)
		screenErr := dbgScreenVerify(dbgFGLServer, fglChanged, snap)
		// A35：**启动那一刻**就把"没接线"说清（别等他下完断点才发现程序已经退了）。
		// 判据是快照里的 fglserver —— 那是这一轮真正导出的那一个（见 wiring.go）。
		if snap != nil {
			printWiringHint(dbgWiringHint(snap.FGLServer))
		}
		// JSON 模式:把身份字段并进同一个快照对象,不另打散的行,别破坏 jq 取字段的习惯。
		if IsJSON() {
			applyIdentityTo(raw, snap)
			if err := dbgJSONOut(raw); err != nil {
				return err
			}
			return screenErr
		}
		if err := dbgJSONOut(raw); err != nil {
			return err
		}
		fmt.Println(dbgIdentity(snap))
		if line := dbgScreenLine(snap); line != "" {
			fmt.Println(line)
		}
		if !dbgVerifyEnv(targetEnv, snap) {
			// 点名的环境没生效:不是"成功跑起来了" —— 调用方得知道这一轮不可信。
			return fmt.Errorf("--env %s 未生效(实际 %s),本轮调试的环境不对", targetEnv, orPlain(snap.Env))
		}
		return screenErr
	},
}

// dbgWireOverrides 把"这一轮怎么接线"的三件套(环境 / 区域 / FGLSERVER)填进请求体,
// 并回出这次点名的环境名(用于事后核对 --env 有没有真的生效)。
//
// 三个开关的语义一致:给了就是权威,服务端不猜、不回落。差别只在 FGLSERVER 的
// 空串有语义 —— "本轮交回 T100 按 SSH 来源 IP 自动设置",所以必须按 Changed 判,
// 不能按值判空(按值判空的话,这个最好用的动作恰恰做不到)。
func dbgWireOverrides(cmd *cobra.Command, body map[string]any) string {
	target := dbgTargetEnv()
	if dbgZone != "" {
		body["zone"] = dbgZone
	}
	// --ssh 是环境名的历史叫法:两条并存时以 --ssh 为准(它就是为这件事存在的)。
	switch {
	case dbgSSHName != "":
		body["ssh"] = dbgSSHName
		target = dbgSSHName
	case target != "":
		body["ssh"] = strings.TrimSpace(target)
	}
	if cmd.Flags().Changed("fglserver") {
		body["fglserver"] = dbgFGLServer
	}
	return target
}

// dbgScreenLine 回"这一轮画面去哪"那一行(服务端已经算好人话,这里只做兜底与回显)。
//
// 为什么要单独打这一行:画面去哪是静默的 —— 被填错时程序照常启动、照常停站,
// 只是放行之后什么都不弹,看起来像卡死。实测事故就是配置里留着无头前端实验的值
// (10.1.25.33:1300 → 前端 7700),放行后 GDC 一片空白而启动输出只字未提。
func dbgScreenLine(snap *dbgSnapView) string {
	if snap == nil {
		return ""
	}
	if snap.ScreenTarget != "" {
		return snap.ScreenTarget
	}
	if snap.FGLServer != "" {
		return "画面去向: FGLSERVER=" + snap.FGLServer + "(本轮覆盖)"
	}
	return ""
}

// dbgScreenVerify 点了 --fglserver 就必须认账,没认账就判失败。
//
// 为什么非要有这一条:服务端**不认识的请求字段会被静静忽略**。用新 CLI 对着没重启的
// 旧服务端传 --fglserver,画面还留在老地方,而输出里没有任何痕迹 —— 那正是这次事故的
// 形状:以为自己改了接线、实际没改,于是往"GDC 是不是坏了"的方向一路查下去。
func dbgScreenVerify(want string, changed bool, snap *dbgSnapView) error {
	if !changed || snap == nil {
		return nil
	}
	if strings.TrimSpace(snap.FGLServer) == strings.TrimSpace(want) {
		return nil
	}
	fmt.Fprintf(os.Stderr, "⚠ --fglserver %q 未生效:本轮会话用的是 %q\n", want, snap.FGLServer)
	fmt.Fprintf(os.Stderr, "  多半是对面的服务端还没重启(它不认这个字段时会静静忽略):\n")
	fmt.Fprintf(os.Stderr, "  tt serve --stop; tt serve   然后重发这一轮\n")
	return fmt.Errorf("--fglserver %q 未生效(实际 %q),本轮画面去向不对", want, snap.FGLServer)
}

// dbgReusedHost 这次 start 是"复用了启动前那个空闲会话"吗。
//
// 判据只有一条:服务端回来的 sessionId 与启动前那个空闲会话相同。会话 id 每次新建都会
// 变(s<毫秒>),所以相同就意味着后端复用了那个已登录的宿主 —— 这时并没有重新登录,
// 也就没有"按新环境重连"这回事。
func dbgReusedHost(prevID, newID string) bool {
	return prevID != "" && newID != "" && prevID == newID
}

// dbgSnapOf 把 /api/sessions/{id} 的原始快照解成 dbgSnapView。
func dbgSnapOf(raw map[string]any) *dbgSnapView {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var v dbgSnapView
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	return &v
}

// applyIdentityTo 把"生效身份"并进 JSON 快照(env 已由服务端给出,这里补 reused/identity)。
func applyIdentityTo(raw map[string]any, snap *dbgSnapView) {
	if raw == nil || snap == nil {
		return
	}
	if snap.Env != "" {
		raw["env"] = snap.Env
	}
	if snap.Reused {
		raw["reused"] = true
	}
	raw["identity"] = dbgIdentity(snap)
}

// maxBatchCmds 一次批量最多执行的命令条数。
// 每条命令发一次独立 HTTP,不存在单次请求体积的问题 —— 这道上限纯粹是防
// "命令文件里混进上千行",跑起来没完。
const maxBatchCmds = 500

// parseCmdLines 解析命令文件:一行一条,跳过空行与 # 注释行,容忍 CRLF 与 UTF-8 BOM。
// 不认行内注释、不认续行 —— fgldb 命令本身就可能带 #,只能把行首的 # 当注释。
func parseCmdLines(data []byte) []string {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var out []string
	for _, ln := range strings.Split(string(data), "\n") {
		ln = strings.TrimSpace(ln) // 顺带吃掉行尾 \r
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		out = append(out, ln)
	}
	return out
}

// buildCmdList 组装要执行的命令列表:命令行位置参数在前,--file 里的行在后。
// filePath 为空表示没给 --file。两种来源都拿不出命令时报错(空批没有意义,
// 也不该悄悄成功)—— 分开报是因为这两种"空"要修的地方不一样。
func buildCmdList(args []string, fileData []byte, filePath string) ([]string, error) {
	var cmds []string
	for _, a := range args {
		if s := strings.TrimSpace(a); s != "" {
			cmds = append(cmds, s)
		}
	}
	if filePath != "" {
		fromFile := parseCmdLines(fileData)
		if len(fromFile) == 0 && len(cmds) == 0 {
			return nil, fmt.Errorf("命令文件里没有可执行的命令(空行与 # 注释会被跳过): %s", filePath)
		}
		cmds = append(cmds, fromFile...)
	}
	if len(cmds) == 0 {
		return nil, fmt.Errorf("需要至少一条命令(见 tt debug exec --help),或用 --file 指定命令文件")
	}
	if len(cmds) > maxBatchCmds {
		return nil, fmt.Errorf("一次最多 %d 条命令(本次 %d 条)", maxBatchCmds, len(cmds))
	}
	return cmds, nil
}

// batchAbortReason 一条命令执行完之后,是否该中止整批。返回 (是否中止, 给人看的原因)。
//
// state 只在 cmd 是放行类命令时才有值 —— 那时调用方会去读一次会话快照,它是**实况**:
//   - "stopped":程序停在了某个位置(现场刚打过一行),后一条命令仍建立在一个**已知**的
//     现场上,可以接着跑。"走一步、立刻取一批值"就靠这条能一次调用做完;
//   - 其它(运行中/空闲/已退出),或空串(快照没读到、没能确认):中止 —— 后面每条都会
//     落在一个没确认过的现场上。
//
// 非放行类命令传 state = "" 即可:它们不动程序位置,不需要确认。
//
// 普通错误(print 一个不存在的变量之类)**不中止** —— 那正是"一条打错不影响其余"的场景:
// 它在会话层只是输出里的一行文本,HTTP 仍然是 200。真正的硬失败(4xx)由调用方判。
func batchAbortReason(cmd string, softTimeout bool, state string) (bool, string) {
	if softTimeout {
		return true, "软等待到点,程序仍在运行(命令未被取消,也未发 SIGINT)"
	}
	if !debug.IsResumeCmd(cmd) {
		return false, ""
	}
	switch state {
	case "stopped":
		return false, ""
	case "running":
		return true, "它是放行类命令,程序正在运行;用 tt debug wait --for stopped 等它停下来" +
			"(或 tt debug why 看它在等用户还是空转),停下后再取后面的值"
	case "":
		return true, "它是放行类命令,但没能确认程序停在哪(快照没读到);先 tt debug stop 看清现场再继续"
	}
	return true, "它是放行类命令,而程序已离开调试器(当前 " + state + ");先确认现场再继续"
}

// rawOut 一条裸命令的回路(字段与 /raw 的回包一一对应)
type rawOut struct {
	Lines       []string `json:"lines"`
	SoftTimeout bool     `json:"softTimeout"`
	State       string   `json:"state"`
	Silent      float64  `json:"silentSeconds"`
	// 会话层就截断过(行数/单值超上限);以及完整输出的本地副本路径
	Truncated   bool   `json:"truncated"`
	TruncReason string `json:"truncReason"`
	LocalPath   string `json:"localPath"`
	TotalLines  int    `json:"totalLines"`
}

// applyMax 按 --max 裁剪要打印的行(只影响**显示**,不影响落盘的那种完整副本)
func applyMax(lines []string, max int) ([]string, bool) {
	if max <= 0 || len(lines) <= max {
		return lines, false
	}
	return lines[:max], true
}

// printLines 打印一条命令的输出,并按上限裁剪 + 说明完整副本在哪。
//
// 这是"别把 AI 上下文打爆"的落点:正文只给头部,完整内容留在本地可随时 grep。
// 少了任何一句提示,收到的就只是"被截断的内容",而调用方会以为那就是全部。
func printLines(r *rawOut) {
	shown, cut := applyMax(r.Lines, dbgExecMax)
	for _, ln := range shown {
		fmt.Println(ln)
	}
	if r.Truncated {
		what := "行数"
		if r.TruncReason == "value" {
			what = "单值长度"
		}
		fmt.Printf("⚠ 会话层已按%s上限截断过(超出部分没传回来)\n", what)
	}
	switch {
	case cut:
		fmt.Printf("…(共 %d 行,只显示前 %d 行", len(r.Lines), len(shown))
	case r.LocalPath != "":
		fmt.Printf("(输出 %d 行", len(r.Lines))
	default:
		return
	}
	if r.LocalPath != "" {
		fmt.Printf(";完整副本: %s", r.LocalPath)
	} else if cut {
		fmt.Printf(";要看全部加 --max 0")
	}
	fmt.Println(")")
}

// dbgRawOnce 透传一条 fgldb 命令
//
// 错误统一过 `withRunningHint`（A34）：程序 running 时服务端回 400「程序运行中…」，
// 那句话本身只说"禁止"，不说"那怎么办" —— 这里补上可执行的那一步（先 interrupt）。
// **不重试**：第二次同样会被拒，而"静默重试"恰恰会让人以为命令发出去了。
func dbgRawOnce(id, cmd string) (*rawOut, error) {
	body := map[string]any{"command": cmd, "timeout": dbgTimeout}
	if dbgSoftWait > 0 {
		body["wait"] = dbgSoftWait
	}
	data, err := dbgAPI("POST", dbgSessionURL(id, "/raw"), body)
	if err != nil {
		return nil, withRunningHint(err)
	}
	var r rawOut
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func printStillRunning(r *rawOut) {
	fmt.Printf("— 仍在运行(%s,已静默 %.0fs):命令未取消,也未发 SIGINT\n", r.State, r.Silent)
	fmt.Println("  接着可以: tt debug wait --for stopped --timeout 300   (等它停下来)")
	fmt.Println("            tt debug why                              (看它在等用户还是空转)")
}

// dbgExecOne 单条命令:输出与批量出现之前**逐字一致**(不带头部行),向后兼容
func dbgExecOne(id, cmd string) error {
	r, err := dbgRawOnce(id, cmd)
	if err != nil {
		return err
	}
	printLines(r)
	if r.SoftTimeout {
		printStillRunning(r)
		return nil
	}
	// 执行完成后再取一次快照:若是 step/next/continue 等使程序继续的命令,
	// 停站后自动回报现场(文件:行/函数/原因),避免 AI 再发一次查询才知道停哪。
	return dbgReportStopIfStopped(id, cmd)
}

// dbgExecBatch 批量:顺序执行、逐条标注输出。碰到该中止的情况就停下并说明原因,
// 剩下的标为未执行 —— 不闷头把后续命令发到一个已经变了的现场上。
func dbgExecBatch(id string, cmds []string) error {
	n := len(cmds)
	for i, c := range cmds {
		fmt.Printf("── [%d/%d] %s\n", i+1, n, c)
		r, err := dbgRawOnce(id, c)
		if err != nil {
			// 硬失败(会话不在可发命令的状态等):后续必然同样失败,直接收工
			fmt.Printf("  —— 失败:%v\n", err)
			fmt.Printf("— 已中止于第 %d/%d 条", i+1, n)
			if n-i-1 > 0 {
				fmt.Printf(",后 %d 条未执行", n-i-1)
			}
			fmt.Println()
			return nil
		}
		if len(r.Lines) == 0 {
			fmt.Println("(无输出)")
		}
		printLines(r)
		// 放行类命令让程序离开当前停站点。它可能停在了**新的**停站点(那就能接着取值),
		// 也可能还在跑、也可能已经退出 —— 读一次快照看实况,别靠猜。
		// 停住了的话 dbgFetchStopSite 顺带把现场打一行出来,后一条命令就建立在已知现场上。
		state := ""
		if debug.IsResumeCmd(c) && !r.SoftTimeout {
			if st, serr := dbgFetchStopSite(id); serr == nil {
				state = st
			} // 读不到就留空,按"没能确认"处理
		}
		stop, why := batchAbortReason(c, r.SoftTimeout, state)
		if !stop {
			continue
		}
		if r.SoftTimeout {
			printStillRunning(r)
		}
		fmt.Printf("— 已中止于第 %d/%d 条:%s", i+1, n, why)
		if n-i-1 > 0 {
			fmt.Printf("(后 %d 条未执行)", n-i-1)
		}
		fmt.Println()
		return nil
	}
	return nil
}

// debugExecCmd 透传任意 fgldb 标准调试命令,原样返回输出。
// continue/run 等命令会阻塞到程序再次停站(与 fgldb 提示符语义一致)。
var debugExecCmd = &cobra.Command{
	SilenceUsage: true, // 报错多为运行期(状态/SSH/库),不是用法问题:别打一大段 usage 误导
	Use:          "exec \"<命令>\" [更多命令...]",
	Short:        "透传 fgldb 标准调试命令(print/break/next/where/info/...),可一次多条",
	Long: `透传 Genero 调试器标准命令,输出为 fgldb 原生文本:
  tt debug exec "break 123"        下断点
  tt debug exec "continue"         继续运行,阻塞到下次停站
  tt debug exec "print ls_sql"     求值变量/表达式
  tt debug exec "info breakpoints" 查看断点
  tt debug exec "where"            调用栈
命令清单见 Genero 文档 Debugger commands(break/continue/print/where/info/watch/...)。

一次给多条就是批量,省掉每条一次进程启动(实测单次调用约 50ms,取十几个值就是秒级的差别):
  tt debug exec "print g_req_param" "print g_status" "info locals"
  tt debug exec --file cmds.txt     # 一行一条;空行与 # 开头跳过

批量逐条标注 [i/N] 与命令原文。放行类命令(continue/next/step/until/finish/run)执行后
会读一次现场:程序**停住了**就接着往下跑 —— 所以"走一步、立刻取一批值"一次调用就能做完:
  tt debug exec "next" "print g_qryparam.cond" "print arr.getLength()"
没停住(还在跑/已退出/没读到)才中止,剩余标为未执行 —— 后面的命令不该落在一个没确认过的
现场上。普通错误(print 一个不存在的变量)不中止,后面照跑。

--timeout / --wait 对**每一条**生效(不是整批的总预算),批量时按需调小。

continue/next 这类会让程序跑起来的命令,建议加 --wait N:到点若程序仍在跑就直接返回
(不再等,也**不发 SIGINT**)。不加 --wait 时,超过 --timeout 会发 SIGINT 探测 ——
那会打断正停在 GDC 界面上等用户输入的程序,所以别再用"短 --timeout"当软等待。`,
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var fileData []byte
		if dbgExecFile != "" {
			b, err := os.ReadFile(dbgExecFile)
			if err != nil {
				return fmt.Errorf("读取命令文件失败: %s: %w", dbgExecFile, err)
			}
			fileData = b
		}
		cmds, err := buildCmdList(args, fileData, dbgExecFile)
		if err != nil {
			return err
		}
		// 会话 id 只取一次:批量时它本来是每条都要重取的。
		// 目标由目标层判（--session / 唯一就落 / 多条报错），见 target.go。
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		if len(cmds) == 1 {
			return dbgExecOne(id, cmds[0])
		}
		return dbgExecBatch(id, cmds)
	},
}

// debugWhyCmd 探测程序此刻在等用户还是在空转。
var debugWhyCmd = &cobra.Command{
	Use:   "why",
	Short: "探测程序此刻在干什么:在等用户操作,还是在空转(慢查询/慢循环)",
	Long: `把「interrupt → where → 看停在哪一行」三步收成一条命令。

- 已经停在停站态时只做分类,不发任何命令(零副作用);
- 运行中才发 SIGINT,拿到停站现场后判断那一行是不是交互语句
  (INPUT / MENU / DISPLAY ARRAY / CONSTRUCT / PROMPT …);
- 默认探测完自动放回运行 --no-resume 可保留现场。

注意 SIGINT 的副作用随前端模式而变:TUI 的对话框会被取消(int_flag 置位),
GDC 这类 GUI 前端不会取消当前对话框。没有 DEFER INTERRUPT 时 SIGINT 默认终止进程,
现在只挂起是因为 fglrun -d 的调试器拦下了它 —— 所以别把探测当无代价操作频繁用。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		data, err := dbgAPI("POST", dbgSessionURL(id, "/why"), map[string]any{
			"resume": !dbgNoResume,
		})
		if err != nil {
			return err
		}
		if IsJSON() {
			fmt.Println(string(data))
			return nil
		}
		var r struct {
			Why struct {
				WaitingForUser bool   `json:"waitingForUser"`
				Kind           string `json:"kind"`
				Evidence       string `json:"evidence"`
				Reason         string `json:"reason"`
				Func           string `json:"func"`
				File           string `json:"file"`
				Line           int    `json:"line"`
				SourceText     string `json:"sourceText"`
				Resumed        bool   `json:"resumed"`
				Risk           string `json:"risk"`
			} `json:"why"`
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		w := r.Why
		if w.WaitingForUser {
			fmt.Printf("在等用户操作:停在 %s 语句上\n", w.Kind)
		} else {
			fmt.Println("不是在等用户")
		}
		fmt.Printf("  %s\n", w.Evidence)
		if w.File != "" || w.Line > 0 {
			fmt.Printf("  位置: %s:%d %s\n", w.File, w.Line, w.Func)
		}
		if w.SourceText != "" {
			fmt.Printf("  该行: %s\n", w.SourceText)
		}
		if w.Resumed {
			fmt.Println("  已自动放回运行。")
		}
		if w.Risk != "" {
			fmt.Printf("  ⚠ %s\n", w.Risk)
		}
		return nil
	},
}

// debugWaitCmd 长轮询:等到会话出现指定事件或超时。
var debugWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "阻塞等待会话事件(停站/退出/掉线/看门狗),或等到超时",
	Long: `等到会话出现指定事件才返回,替代"隔几秒查一次"的轮询。

  tt debug wait --for stopped --timeout 300   # 等程序停到断点上(用户在 GDC 操作后)
  tt debug wait --for exit,dead               # 等程序结束或掉线

超时**不是错误**:到点返回 timedOut 并给出当前状态(退出码 0),便于脚本分支。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		q := url.Values{}
		if dbgWaitFor != "" {
			q.Set("for", dbgWaitFor)
		}
		q.Set("timeout", strconv.Itoa(dbgTimeout))
		data, err := dbgAPI("GET", dbgSessionURL(id, "/wait?"+q.Encode()), nil)
		if err != nil {
			return err
		}
		if IsJSON() {
			fmt.Println(string(data))
			return nil
		}
		var r struct {
			State    string  `json:"state"`
			TimedOut bool    `json:"timedOut"`
			Silent   float64 `json:"silentSeconds"`
			Event    *struct {
				Type string `json:"type"`
			} `json:"event"`
			Stop *struct {
				Reason string `json:"reason"`
				Func   string `json:"func"`
				File   string `json:"file"`
				Line   int    `json:"line"`
			} `json:"stop"`
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		if r.TimedOut {
			fmt.Printf("超时:状态仍为 %s", r.State)
			if r.State == "running" {
				fmt.Printf("(已静默 %.0fs —— 可能在界面上等用户;可 tt debug why 确认)", r.Silent)
			}
			fmt.Println()
			return nil
		}
		ev := ""
		if r.Event != nil {
			ev = r.Event.Type
		}
		fmt.Printf("事件 %s(状态 %s)\n", ev, r.State)
		if r.Stop != nil {
			fmt.Printf("停站 %s:%d %s reason=%s\n", r.Stop.File, r.Stop.Line, r.Stop.Func, r.Stop.Reason)
		}
		return nil
	},
}

// debugModeCmd 查看/切换会话模式。
var debugModeCmd = &cobra.Command{
	Use:   "mode [solo|collab]",
	Short: "查看或切换会话模式:纯人工(solo) / 协作(collab,AI 主导)",
	Long: `不带参数 = 查看当前模式;带参数 = 切换。

  tt debug mode            # 查看模式、状态与正在执行的命令
  tt debug mode collab     # 请求切到协作模式

模式决定"谁能写":纯人工只有人能操作(AI 只能读),协作只有 AI 能操作(人只读,
要下断点/执行命令请在对话里委托给 AI)。

**两个方向都放行,人与 AI 都能切**:AI 可以用「tt debug mode collab」自行接管一个
纯人工会话(不必等用户在界面上点「交给 AI」),人在界面上的「接管」也随时能收回 ——
人始终有最终控制权。`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		if len(args) == 0 {
			data, err := dbgAPI("GET", dbgSessionURL(id, ""), nil)
			if err != nil {
				return err
			}
			if IsJSON() {
				fmt.Println(string(data))
				return nil
			}
			var r struct {
				Mode     string `json:"mode"`
				State    string `json:"state"`
				Inflight *struct {
					Cmd     string  `json:"cmd"`
					Elapsed float64 `json:"elapsed"`
				} `json:"inflight"`
			}
			if err := json.Unmarshal(data, &r); err != nil {
				return err
			}
			label := "纯人工(只有人能操作)"
			if r.Mode == "collab" {
				label = "协作(AI 主导,人只读)"
			}
			fmt.Printf("模式: %s  状态: %s\n", label, r.State)
			if r.Inflight != nil {
				fmt.Printf("正在执行: %s(已 %.0fs)\n", r.Inflight.Cmd, r.Inflight.Elapsed)
			}
			return nil
		}
		m := strings.ToLower(strings.TrimSpace(args[0]))
		if m != "solo" && m != "collab" {
			return fmt.Errorf("模式只能是 solo 或 collab")
		}
		data, err := dbgAPI("POST", dbgSessionURL(id, "/mode"), map[string]any{"mode": m})
		if err != nil {
			return err
		}
		if IsJSON() {
			fmt.Println(string(data))
			return nil
		}
		fmt.Printf("已切换到 %s\n", m)
		return nil
	},
}

// debugStatusCmd 服务状态 + 活动会话
var debugStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看调试服务状态与活动会话",
	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := dbgAPI("GET", "/api/status", nil)
		if err != nil {
			return err
		}
		sessions, err := dbgAPI("GET", "/api/sessions", nil)
		if err != nil {
			return err
		}
		fmt.Println(string(st))
		fmt.Println(string(sessions))
		return nil
	},
}

// debugQuitCmd 结束当前会话的本轮运行。
//
// 旧文案写「结束指定调试会话(作业窗口随之关闭)」—— 那是**错的**，实测（2026-10-29）
// 跑完 quit 之后会话仍在列表里，`state=idle`：作业确实结束了，但 SSH 连接 / PTY /
// 服务器 shell 全部保留，供下一轮免重登复用。用户以为收干净了，实际占着一套连接，
// 现场“这台机器挂了十几条 ssh”多半就是这么来的。真正断连是 `tt session rm`。
var debugQuitCmd = &cobra.Command{
	Use:   "quit",
	Short: "结束本轮运行（作业关闭、会话保留可直接再启动）；要断开连接用 tt session rm",
	Long: `结束本条会话的本轮运行：给 fgldb 发 quit、作业窗口关闭、事务放开。

**会话本身保留（state=idle）** —— SSH 连接、登录态、服务器 shell 都还在，下一轮
调试免重新登录。这是有意的（登录一次几秒，复用它）。

要真正断开连接（开会话积累太多 ssh 时）：

  tt session rm --session <ref>

停站持锁时的两条出路：等看门狗自动放行（默认 1800 秒），或直接 quit 立即放行。`,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		if _, err := dbgAPI("DELETE", dbgSessionURL(id, ""), nil); err != nil {
			return err
		}
		fmt.Println(`{"ok":true}`)
		return nil
	},
}

// wsLogItem 一条接口日志(与 /api/wslogs 的 items 元素同形)
type wsLogItem struct {
	Rowid    string `json:"rowid"`
	Service  string `json:"service"`
	Job      string `json:"job"`
	PID      string `json:"pid"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Duration string `json:"duration"`
	Code     string `json:"code"`
	ErrMsg   string `json:"errMsg"`
	Origin   string `json:"origin"`
	Server   string `json:"server"`
	// 报文大小(wsfa016/017)。请求那条用来判断存下来的报文完不完整:
	// 记录大小与存文长度一致 = 完整(见 showWSLog 的标题行)。
	ReqSize string `json:"reqSize"`
	RspSize string `json:"rspSize"`
}

// printWSLogItem 打一行摘要 + 一行 rowid。
// 每个字段都带标签:服务名 / 作业 / 服务程序是三样东西,靠列位置对齐会让人认错
// (service 是 oa.schema.data.get 这类长反域名,定宽列一撑就歪)。
func printWSLogItem(it wsLogItem) {
	// wsfa005 处理时间的单位是**秒**(界面那列写的就是「耗时(s)」)。
	// 这里以前标成 ms:0.42 秒的调用显示成 "0.42434ms",量级差一千倍,会被当成异常。
	line := fmt.Sprintf("%s  %s  %ss  作业=%s", it.Start, it.Code, it.Duration, it.Job)
	if it.PID != "" {
		// 服务程序序号 = 界面上「服务程序」那一列:与 --pid 过滤、以及用户截图里的数字对得上
		line += "  程序=" + it.PID
	}
	if it.Service != "" {
		line += "  服务=" + it.Service
	}
	if it.End != "" {
		line += "  结束=" + it.End
	}
	if it.Origin != "" || it.Server != "" {
		line += fmt.Sprintf("  %s/%s", it.Origin, it.Server)
	}
	if it.ErrMsg != "" {
		line += "  ERR=" + it.ErrMsg
	}
	fmt.Println(line)
	fmt.Println("  rowid=" + it.Rowid)
}

// wslOut 单条日志的报文内容(/api/wslogs/content)
type wslOut struct {
	Item    wsLogItem  `json:"item"`
	Acct    *wsLogAcct `json:"acct"`
	Content struct {
		Request        string `json:"request"`
		Response       string `json:"response"`
		RequestPartial bool   `json:"requestPartial"`
	} `json:"content"`
}

// wsLogAcct 接口日志是用哪个账号查的、为什么(服务端随回包带出来)。
//
// 为什么值得单独打一行:空结果时这是唯一能区分"这段时间确实没调用"与
// "账号/企业用错了"的线索。只读 SQL 那边靠结果头的"企业 N → 账号 X",
// 接口日志这边则要说明"它压根不按企业走"。
type wsLogAcct struct {
	Account       string `json:"account"`
	Reason        string `json:"reason"`
	TopentWarning string `json:"topentWarning"`
}

func printWSLogAcct(a *wsLogAcct) {
	if a == nil || a.Account == "" {
		return
	}
	fmt.Printf("[账号] %s —— %s\n", a.Account, a.Reason)
	if a.TopentWarning != "" {
		fmt.Printf("[账号] 注意: %s\n", a.TopentWarning)
	}
}

func wslogContentURL(rowid string) string {
	q := url.Values{}
	q.Set("rowid", rowid)
	if e := strings.TrimSpace(dbgTargetEnv()); e != "" {
		q.Set("env", e)
	}
	return "/api/wslogs/content?" + q.Encode()
}

func fetchWSLog(rowid string) (*wslOut, error) {
	data, err := dbgAPI("GET", wslogContentURL(rowid), nil)
	if err != nil {
		return nil, err
	}
	var r wslOut
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

// setJSONPath 在解析好的 JSON 里按点路径赋值,如
// digi-body.std_data.parameter.production_item_no;路径段落在数组上时按下标取。
//
// 值先按 JSON 解析:能当数字 / true / false / null 就用它,否则当字符串 ——
// 于是 `x=` 给空串、`x=123` 给数字、`x=abc` 给字符串,命令行上不必再挂类型开关。
//
// 路径必须已经存在:不自动造层级 —— 拼错一个段就会悄悄改出一份形状不对的报文,
// 那比当场报错难查得多。
func setJSONPath(root map[string]any, path, rawVal string) error {
	segs := strings.Split(path, ".")
	var val any = rawVal
	if rv := strings.TrimSpace(rawVal); rv != "" {
		var parsed any
		if json.Unmarshal([]byte(rv), &parsed) == nil {
			val = parsed
		}
	}
	var cur any = root
	for i, seg := range segs {
		last := i == len(segs)-1
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[seg]
			if !ok {
				return fmt.Errorf("路径不存在: %s(在 %q 处)", path, strings.Join(segs[:i+1], "."))
			}
			if last {
				node[seg] = val
				return nil
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(seg)
			if err != nil || idx < 0 || idx >= len(node) {
				return fmt.Errorf("路径不存在: %s(段 %q 不是合法数组下标)", path, seg)
			}
			if last {
				node[idx] = val
				return nil
			}
			cur = node[idx]
		default:
			return fmt.Errorf("路径走不通: %s(段 %q 处既不是对象也不是数组)", path, seg)
		}
	}
	return nil
}

// applySets 把 --set 的改动应用到请求报文上
func applySets(req string, sets []string) (string, error) {
	var root map[string]any
	if err := json.Unmarshal([]byte(req), &root); err != nil {
		return "", fmt.Errorf("入参报文不是 JSON(--set 只改 JSON;这条报文多半是 XML)。"+
			"用 tt debug wslogs --show <rowid> --save-request req.txt 导出原文,改完再 --request-file req.txt: %w", err)
	}
	for _, s := range sets {
		k, v, ok := strings.Cut(s, "=")
		if !ok {
			return "", fmt.Errorf("--set 要写成 路径=值,实际 %q", s)
		}
		if err := setJSONPath(root, strings.TrimSpace(k), v); err != nil {
			return "", err
		}
	}
	out, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// buildReplayRequest 组装重放用的入参报文。返回空串 = 用日志里的原报文。
func buildReplayRequest(rowid string) (string, error) {
	if wsDebugReqFile == "" && len(wsDebugSet) == 0 {
		return "", nil
	}
	req := ""
	if wsDebugReqFile != "" {
		b, err := os.ReadFile(wsDebugReqFile)
		if err != nil {
			return "", fmt.Errorf("读取入参文件失败: %s: %w", wsDebugReqFile, err)
		}
		req = string(b)
	}
	if len(wsDebugSet) == 0 {
		return req, nil
	}
	if req == "" {
		// 以日志里的原报文为底稿:--set 是"改一处",不是在空报文上造一份
		lg, err := fetchWSLog(rowid)
		if err != nil {
			return "", err
		}
		if lg.Content.RequestPartial {
			return "", fmt.Errorf("该日志的请求报文不完整(超出读取上限或源文件已清理),不能基于它改入参;" +
				"原样重放不受影响 —— 去掉 --set 即可")
		}
		req = lg.Content.Request
	}
	return applySets(req, wsDebugSet)
}

// printTopentContext 把重放环境的 TOPENT 说明白。
//
// 它是"重放能不能跑进业务逻辑"的关键,可快照里只有 topent/topentCfg/topentShell 三个
// 裸字段 —— 真有人把据点码填进企业编号的位置(配置里就叫 topent),然后一路查不出
// 程序为什么跑不到业务逻辑。
func printTopentContext(snap map[string]any) {
	eff, _ := snap["topentShell"].(string)
	if eff == "" {
		eff, _ = snap["topentCfg"].(string)
	}
	if eff == "" {
		return
	}
	src := "配置默认"
	if o, _ := snap["topent"].(string); o != "" {
		src = "会话覆盖"
	}
	fmt.Printf("重放环境:TOPENT=%s(%s)\n", eff, src)
	if _, err := strconv.Atoi(eff); err != nil {
		fmt.Println("  ⚠ 它不是纯数字。T100 的企业编号是数字,这里更像是把据点码填错了位置 ——")
		fmt.Println("    那样框架的前置校验会失败,程序跑不到业务逻辑(表现:一路退出、断点不命中)。")
		fmt.Println("    改:tt debug topent <企业编号>(例 99)")
	}
}

// debugWslogsCmd 获取接口日志列表
var debugWslogsCmd = &cobra.Command{
	SilenceUsage: true, // 报错多为运行期(状态/SSH/库),不是用法问题:别打一大段 usage 误导
	Use:          "wslogs",
	Short:        "获取接口日志列表(wssp/awsp 报文流水)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if wsShow != "" {
			return showWSLog(wsShow, IsJSON(), wsSaveReq)
		}
		// 条件对齐 awsq990(与 Web 工具条一致):服务名称 wsfa001(支持 * ? 通配)、
		// 处理结果 wsfa006、发起端 wsfa013、服务端 wsfa018;时间窗 = wsfa003 >= from 且 wsfa004 <= to
		q := url.Values{}
		q.Set("service", wsService)
		q.Set("result", wsResult)
		q.Set("origin", wsOrigin)
		q.Set("server", wsServer)
		if wsPID != "" {
			q.Set("pid", wsPID)
		}
		if wsJob != "" {
			q.Set("job", wsJob)
		}
		q.Set("onlyFail", strconv.Itoa(boolInt(wsOnlyFail)))
		q.Set("page", strconv.Itoa(wsPage))
		q.Set("pageSize", "50")
		q.Set("startFrom", wsFrom)
		q.Set("endTo", wsTo)
		if e := strings.TrimSpace(dbgTargetEnv()); e != "" {
			q.Set("env", e) // --at/--env 必须带到接口；否则列表永远查的是服务端当前环境
		}
		data, err := dbgAPI("GET", "/api/wslogs?"+q.Encode(), nil)
		if err != nil {
			return err
		}
		if IsJSON() {
			fmt.Println(string(data))
			return nil
		}
		var r struct {
			Items []wsLogItem `json:"items"`
			Acct  *wsLogAcct  `json:"acct"`
		}
		if err := json.Unmarshal(data, &r); err != nil {
			return err
		}
		printWSLogAcct(r.Acct)
		for _, it := range r.Items {
			printWSLogItem(it)
		}
		if len(r.Items) == 0 {
			// 静默成功最容易被当成"服务没起/语法错/通配不支持" —— 明确说清是没匹配
			fmt.Println("无匹配日志(条件太窄,或该时间窗内确实没有记录)")
			if wsFrom == "" && wsTo == "" {
				fmt.Println("  提示:没给 --from/--to 时按原生口径只查最近一段;放宽时间窗试试")
			}
		}
		return nil
	},
}

// showWSLog 打印单条日志的详情与请求/响应报文;reqSave 非空时把请求原文另存一份。
func showWSLog(rowid string, asJSON bool, reqSave string) error {
	data, err := dbgAPI("GET", wslogContentURL(rowid), nil)
	if err != nil {
		return err
	}
	if asJSON {
		fmt.Println(string(data))
		return nil
	}
	var r wslOut
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	printWSLogItem(r.Item)
	printWSLogAcct(r.Acct)
	if reqSave != "" {
		// 报文可能是 XML(不是 JSON),那样 --set 用不了,只能导出原文改完再 --request-file ——
		// 这个开关就是那条路的起点。
		if err := os.WriteFile(reqSave, []byte(r.Content.Request), 0o644); err != nil {
			return fmt.Errorf("写入 %s 失败: %w", reqSave, err)
		}
		fmt.Printf("\n请求原文已存到 %s(改完用: tt debug wsdebug %s --request-file %s)\n", reqSave, rowid, reqSave)
	}
	// "完不完整"直接判给人看:记录大小(wsfa016)与存文长度一致 = 完整。
	// 以前只写一句"不完整",没说判据,也没说这条到底是哪种情况 —— 实测有人据此
	// 差点把一个完好的样本判死(重放明明很忠实)。这里是**逐条**的结论,不是全局状态。
	verdict := "完整"
	if r.Item.ReqSize != "" {
		verdict = fmt.Sprintf("完整,记录大小 %s 字节", r.Item.ReqSize)
	}
	note := ""
	if r.Content.RequestPartial {
		verdict = fmt.Sprintf("不完整,记录大小 %s 字节,只存下 %d 字符", r.Item.ReqSize, len(r.Content.Request))
		// 这里以前写的是"原样重放不受影响" —— 那是错的,而且害人不浅:
		// 源文件已清理时重放用的就是这段残缺文本,JSON 报文会直接崩在解析上。
		note = "\n     ← 服务器上的原文件已清理,重放只能用这段残缺文本。" +
			"\n       JSON 报文会解析失败(框架先按 JSON 解析,失败后落到 XML 路径)," +
			"\n       表现是程序一路退出、断点不命中;wsdebug 遇到这种情况会告警。" +
			"\n       这条也改不了入参(拿不到完整原文),只能原样重放或换个样本。"
	}
	fmt.Printf("\n── 请求报文(%d 字符 · %s)%s\n%s\n", len(r.Content.Request), verdict, note, r.Content.Request)
	fmt.Printf("\n── 响应报文(%d 字符)\n%s\n", len(r.Content.Response), r.Content.Response)
	return nil
}

// debugSQLCmd 已迁移到 tt db sql；这里只保留隐藏转发（下一个版本移除）。
//
// 转发到 db.QueryOnce 的原因：旧命令走 serve（POST /api/dbsql），而 db sql 是本机
// 直连（SSH + sqlplus/ksql）。两条路必须给出同一份信封，所以共用同一个渲染实现。
var debugSQLCmd = &cobra.Command{
	Use:        `sql "<查询语句>"`,
	Short:      "（已迁移）执行一条只读 SQL —— 用 tt db sql",
	Hidden:     true,
	Deprecated: "已迁移：请用 tt db sql（--at <环境>@<企业>）",
	Args:       cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sqlText := ""
		if len(args) == 1 {
			sqlText = strings.TrimSpace(args[0])
		}
		if dbgSQLFile != "" {
			if sqlText != "" {
				return fmt.Errorf("语句既给了位置参数又给了 --file，二选一")
			}
			b, err := os.ReadFile(dbgSQLFile)
			if err != nil {
				return fmt.Errorf("读取语句文件失败: %s: %w", dbgSQLFile, err)
			}
			sqlText = strings.TrimSpace(string(b))
		}
		if sqlText == "" {
			return fmt.Errorf(`需要一条查询语句，或用 --file 指定语句文件`)
		}
		ent := dbgSQLEnt
		if ent <= 0 {
			ent = dbgSQLTopentForRun()
		}
		if ent <= 0 {
			return fmt.Errorf("需要 --at <环境>@<企业>（或 --env + --ent）：企业编号决定用哪个账号")
		}
		return cdb.QueryOnce(sqlText, ent, dbgTimeout, cdb.SQLRouteServer)
	},
}

// dbgSQLTopentForRun 取"这条会话本轮生效的企业编号"，给只读 SQL 显式填 `ent` 用
// （契约 C7.1-5：服务端那条"ent<=0 回落当前会话 TOPENT"的兜底已按 Q4/C8.2 删掉）。
//
// 判据与服务端原来的 `Session.TopentIntForDB` 同源：**会话内的 TOPENT 覆盖优先**，
// 其次是启动时显式给的那个值（快照里的 `topentSession`），再次是登录 shell 里实际
// 回读到的 `$TOPENT`（`topentShell` —— 用户没显式给 TOPENT 时，会话不会 export，
// 所以"这台机器连的是哪个企业"以它为准）。
//
// 取不到、或不是纯数字（典型是把据点码填进了企业编号的位置）**返回 0、不猜**：
// 那正是 `TopentIntForDB` 的注释里点名的危险现场 —— 拿错编号去查库的结果是 0 行，
// 而 0 行会被读成"这条业务数据不存在"。返回 0 时由服务端按 USAGE 明确拒掉。
func dbgSQLTopentForRun() int {
	cur := dbgAmbientSession()
	if cur == nil {
		return 0 // 池空或多条（落不上目标）：不替调用方挑一条，让服务端报清楚
	}
	snap, err := dbgSnap(cur.ID)
	if err != nil {
		return 0
	}
	raw := strings.TrimSpace(firstNonEmpty(snap.Topent, snap.TopentSession, snap.TopentShell))
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

// debugWsdebugCmd 对指定日志发起重放调试
var debugWsdebugCmd = &cobra.Command{
	SilenceUsage: true, // 报错多为运行期(状态/SSH/库),不是用法问题:别打一大段 usage 误导
	Use:          "wsdebug <rowid>",
	Short:        "按接口日志的报文参数启动重放调试,等到入口停站",
	Long: `取出该日志的请求/响应报文,解析出作业与启动参数,自动重放该次调用并停在入口。
rowid 从 tt debug wslogs 输出中取。

**要改入参再重放**(界面上那套能力,命令行同样有):
  --set 路径=值         按点路径改一个字段,可重复。路径形如
                        digi-body.std_data.parameter.production_item_no
                        值先按 JSON 解析(能当数字/布尔就用),否则当字符串;等号后留空即清空
  --request-file 文件   整份替换入参报文(想大改就先把原文存下来改完再送)

例:
  tt debug wsdebug <rowid> --set digi-body.std_data.parameter.production_item_no=
  tt debug wsdebug <rowid> --request-file my_req.json

**重放能不能跑进业务逻辑,取决于会话的 TOPENT(企业编号)。** 成功启动后输出里会打印
实际生效值;若它不是纯数字,多半是把据点码填进了企业编号的位置 —— 那样框架的前置
校验会失败,程序跑不到业务逻辑(表现是:程序一路退出、断点不命中)。用
tt debug topent <企业编号> 改。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rowid := args[0]
		// `--name` 的取值纪律在**发请求之前**判（F19 / C2.8）：不合法时一个字节都不发给服务端。
		// wsdebug 与 `debug start` 同样是**起会话**的命令（C3.5），所以这个标志的语义与它在
		// start 上**完全一致**（同一份取值纪律、同一套幂等语义：Q17 / Q23）。
		if err := dbgValidateNameFlag(); err != nil {
			return err
		}
		reqOverride, err := buildReplayRequest(rowid)
		if err != nil {
			return err
		}
		body := map[string]any{"rowid": rowid}
		// `--ent`：重放同样要**显式**说明跑在哪个企业（与 start 同一条规矩）。
		if e := strings.TrimSpace(dbgEntRun); e != "" {
			body["ent"] = e
		}
		if reqOverride != "" {
			body["request"] = reqOverride
		}
		// `--name` 是**服务端的事实**（C5.3 的 `name` 键）：随这次起会话送过去，别名从那一刻
		// 起参与寻址（`--session <别名>`）、当幂等键（Q23）、随会话消失即释放（Q19）。
		// 服务端那一侧由 `hWSLogDebug` → `LaunchReplay` 一路带到 `prepareSession` 登记。
		// 空串不送 —— 没打这个标志时请求体与今天逐字相同（H2）。
		if n := strings.TrimSpace(dbgNameFlag); n != "" {
			body["name"] = n
		}
		// 服务端不再自己挑"当前会话"来收口（WP4b：那五处 `Manager.Current()` 兜底全删了，
		// Q4/C8.2 硬切）。要收掉的那一条由这边显式给（服务端原始句柄，C5.4）：
		// 唯一就落；落不上（池空/多条）时不带这个键，由服务端按同一条规矩报
		// 409 + `SESSION_AMBIGUOUS` + 候选清单（F1 的形状），而不是随机收掉一条别人的会话。
		if cur := dbgAmbientSession(); cur != nil {
			body["session"] = cur.ID
		}
		targetEnv := dbgWireOverrides(cmd, body)
		fglChanged := cmd.Flags().Changed("fglserver")
		data, err := dbgAPI("POST", "/api/wslogs/debug", body)
		if err != nil {
			return err
		}
		var r struct {
			SessionID string `json:"sessionId"`
			Warn      string `json:"warn"`
		}
		_ = json.Unmarshal(data, &r)
		if r.Warn != "" && !IsJSON() {
			// 用入库的截断文本重放:后果是"程序一路退出、断点不命中" —— 不说清的话,
			// 下一件事一定是去查断点为什么没生效
			fmt.Printf("⚠ %s\n", r.Warn)
		}
		raw, err := dbgWaitStopped(r.SessionID, time.Duration(dbgTimeout)*time.Second)
		if err != nil {
			return err
		}
		snap := dbgSnapOf(raw)
		// 点了 --fglserver 就要认账(旧服务端不认这个字段时是静默忽略的)
		screenErr := dbgScreenVerify(dbgFGLServer, fglChanged, snap)
		// A35：重放这条线也是"起会话 + 放行"，没接线同样在 main 之后立刻 -6300 退出。
		if snap != nil {
			printWiringHint(dbgWiringHint(snap.FGLServer))
		}
		if IsJSON() {
			applyIdentityTo(raw, snap)
			if err := dbgJSONOut(raw); err != nil {
				return err
			}
			return screenErr
		}
		if !IsJSON() {
			printTopentContext(raw)
			fmt.Println(dbgIdentity(snap))
			if line := dbgScreenLine(snap); line != "" {
				fmt.Println(line)
			}
		}
		if !dbgVerifyEnv(targetEnv, snap) {
			return fmt.Errorf("--env %s 未生效(实际 %s),本次重放的环境不对", targetEnv, orPlain(snap.Env))
		}
		return screenErr
	},
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// addClientURLFlag 给控制端命令挂 --url(原为 debug 父命令的持久 flag):
// 只挂在真正连接服务的命令上,serve/probe/db 不需要。
func addClientURLFlag(cmds ...*cobra.Command) {
	for _, c := range cmds {
		c.Flags().StringVar(&dbgAPIURL, "url", "", "调试服务地址(默认自动发现运行中的后台实例;已带 /api 路径的按原样用,只给主机端口的自动补挂载前缀)")
	}
}

func init() {
	debugExecCmd.Flags().IntVar(&dbgTimeout, "timeout", 90, "等待停站超时(秒,对批量里的**每一条**生效);到点发 SIGINT 探测,所以长命令建议改用 --wait")
	debugExecCmd.Flags().IntVar(&dbgSoftWait, "wait", 0, "软等待秒数(对每一条生效):到点若程序仍在跑就直接返回(不发 SIGINT、不取消命令);0=不启用")
	debugExecCmd.Flags().StringVar(&dbgExecFile, "file", "", "命令文件:一行一条,空行与 # 开头跳过(与位置参数并用时位置参数在前)")
	debugExecCmd.Flags().IntVar(&dbgExecMax, "max", 2000, "单条命令**显示**的行数上限(0=不限)。超出的部分不打印,但完整输出会落本地并给出路径")
	debugStartCmd.Flags().IntVar(&dbgTimeout, "timeout", 120, "等待入口停站超时(秒)")
	// 企业编号:**必填**（企业编号决定用哪个账号连库与据点校验，不许静默沿用登录 shell 的 $TOPENT）
	const entHelp = "企业编号(ENT,**必填**):这一轮跑在哪个企业。不给会被服务端拒绝 —— 不再沿用登录 shell 的 $TOPENT"
	debugStartCmd.Flags().StringVar(&dbgEntRun, "ent", "", entHelp)
	debugWsdebugCmd.Flags().StringVar(&dbgEntRun, "ent", "", entHelp)
	debugSQLCmd.Flags().StringVar(&dbgSQLFile, "file", "", "从本地文件读查询语句(长语句用)")
	debugSQLCmd.Flags().IntVar(&dbgSQLEnt, "ent", 0, "企业编号(不给 = 用这条会话当前生效的企业编号；服务端要求显式给，见 C7.1-5)")
	debugSQLCmd.Flags().IntVar(&dbgTimeout, "timeout", 0, "查询超时秒数(默认 30,上限 120)")
	debugWsdebugCmd.Flags().IntVar(&dbgTimeout, "timeout", 150, "等待入口停站超时(秒)")
	debugWsdebugCmd.Flags().StringArrayVar(&wsDebugSet, "set", nil, "改一个入参再重放:路径=值(可重复;路径形如 digi-body.std_data.parameter.x;等号后留空即清空该字段)")
	debugWsdebugCmd.Flags().StringVar(&wsDebugReqFile, "request-file", "", "整份替换入参报文(与 --set 并用时以它为底稿)")
	debugWaitCmd.Flags().IntVar(&dbgTimeout, "timeout", 300, "最长等待秒数(到点返回 timedOut,不是错误)")
	debugWaitCmd.Flags().StringVar(&dbgWaitFor, "for", "stopped,exit,dead,watchdog", "等哪些事件(逗号分隔:stopped,exit,dead,watchdog)")
	debugWhyCmd.Flags().BoolVar(&dbgNoResume, "no-resume", false, "探测完不自动放回运行(默认会自动 continue 放回)")
	// 默认**不给模块**（A27/缺陷 #3）：以前默认写死 "asf"，把服务端登录后按
	// `<作业>.42r` 在各模块目录里做的权威解析整条短路了 —— 实测 `debug start axmt500_wf`
	// 于是报出 `asf_axmt500_wf.4gl`，而真名是 `axm_axmt500_wf.4gl`，`break asf_…:行`
	// 必然 `No source file named`。留空 = 让会话自己解析（唯一权威来源）。
	debugStartCmd.Flags().StringVarP(&dbgModule, "module", "m", "", debug.ModuleFlagHelp())
	debugStartCmd.Flags().StringVar(&dbgZone, "zone", "", "区域代码覆盖(31开发/35测试/36正式;默认取配置)")
	debugStartCmd.Flags().StringVar(&dbgSSHName, "ssh", "", "SSH 配置名(设置页配置的多 SSH;默认取配置)")
	// --env 与全局 --env 同名同义:子命令上显式给的优先级更高(根命令那个是持久开关,
	// 别处能用这里也必须能用;写在子命令上语义最明确)。--ssh 是环境名的历史叫法,两者并存。
	debugStartCmd.Flags().StringVar(&dbgEnvFlag, "env", "", "环境名(等价 --ssh;不给则用服务端当前环境)。同机多环境时务必点名")
	debugWsdebugCmd.Flags().StringVar(&dbgEnvFlag, "env", "", "环境名(等价 --ssh;不给则用服务端当前环境)。重放必须落在日志所在的那个环境上")
	debugWsdebugCmd.Flags().StringVar(&dbgSSHName, "ssh", "", "环境名(--env 的别名)")
	// --fglserver 是"这一轮画面去哪"的单轮覆盖:从前只能改全局 config.debug.fglserver,
	// 于是做一次前端实验 无头前端实验就把调试的画面接线顶掉了,而且静默(见 dbgScreenLine)。
	fglHelp := "本轮作业画面画到哪(FGLSERVER ip:port)。端口 = 前端监听端口-6400:GDC 默认听 7400 → 用 :1000;" +
		"无头前端 无头前端听 7700 → :1300。留空串 = 本轮交回 T100 按 SSH 来源 IP 自动设置;不给 = 用配置值"
	debugStartCmd.Flags().StringVar(&dbgFGLServer, "fglserver", "", fglHelp)
	debugWsdebugCmd.Flags().StringVar(&dbgFGLServer, "fglserver", "", fglHelp)
	debugWslogsCmd.Flags().StringVar(&wsService, "service", "", "服务名称 wsfa001(支持 * ? 通配)。是 oa.schema.data.get 这类反域名,不是作业名")
	debugWslogsCmd.Flags().StringVar(&wsJob, "job", "", "作业编号 wsfa012(支持 * ? 通配,如 wssp01131 / wssp*)—— 按「哪个作业」找日志的入口")
	debugWslogsCmd.Flags().StringVar(&wsServer, "server", "", "服务端 wsfa018(等值,如 T100)")
	debugWslogsCmd.Flags().StringVar(&wsOrigin, "origin", "", "发起端 wsfa013(等值,如 OA)")
	debugWslogsCmd.Flags().StringVar(&wsResult, "result", "", "处理结果 wsfa006(等值,如 000 成功 / 100 失败)")
	debugWslogsCmd.Flags().StringVar(&wsPID, "pid", "", "服务程序序号 wsfa002(等值;界面「服务程序」列;与 --service 一起用可精确定位某一次调用)")
	debugWslogsCmd.Flags().StringVar(&wsFrom, "from", "", "起始时间下界 wsfa003(yyyy-mm-dd)")
	debugWslogsCmd.Flags().StringVar(&wsTo, "to", "", "结束时间上界 wsfa004(yyyy-mm-dd,含当天)")
	debugWslogsCmd.Flags().BoolVar(&wsOnlyFail, "fail", false, "只看失败日志(wsfa006<>000)")
	debugWslogsCmd.Flags().IntVar(&wsPage, "page", 1, "页码(每页 50 条)")

	debugWslogsCmd.Flags().StringVar(&wsShow, "show", "", "只看这一条:按 rowid 打印详情与请求/响应报文(不看列表)")
	debugWslogsCmd.Flags().StringVar(&wsSaveReq, "save-request", "", "配合 --show:把请求原文存成文件(报文是 XML 时改入参只能走这条路)")

	// 会话寻址的两个新标志（契约 1. 的命令角色表 / C3.1 / C3.5 / C2.5）。
	//
	// `--session` 只挂**会话级操作命令**：status / sql / wslogs / wsdebug **不挂**
	// （C1.5 / C1.2 / C1.3：它们不认会话级目标，挂上去就会让 F16 那条"标志挂错命令"
	// 的自纠提示落空）。`--name` 只挂**起会话**的命令（C3.5：别处给它就是 F17）——
	// 今天那两条正是 `start` 与 `wsdebug`（Bug B：wsdebug 同样是起会话的命令，
	// 从前漏了它，于是它起的会话只能用 ref 寻址、也不能当幂等键）。
	dbgRegisterSessionFlag(debugStartCmd, debugExecCmd, debugQuitCmd, debugWhyCmd,
		debugWaitCmd, debugModeCmd)
	dbgRegisterNameFlag(debugStartCmd, debugWsdebugCmd)
	addClientURLFlag(debugStartCmd, debugExecCmd, debugStatusCmd, debugQuitCmd, debugWslogsCmd, debugSQLCmd, debugWsdebugCmd, debugWhyCmd, debugWaitCmd, debugModeCmd)
	Group.AddCommand(debugStartCmd, debugExecCmd, debugStatusCmd, debugQuitCmd, debugWslogsCmd, debugSQLCmd, debugWsdebugCmd, debugWhyCmd, debugWaitCmd, debugModeCmd)
}
