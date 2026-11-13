package debug

// debug 环境/TOPENT 切换命令:
//   tt debug env           列出全部已配置环境与当前生效项(基于 /api/hosts + /api/sessions)
//   tt debug env <名称>     切换当前环境到该环境(重连会话,等会话回 idle)
//   tt debug topent        显示当前会话的 TOPENT override 与配置级默认
//   tt debug topent <值>    设置会话级 TOPENT override(仅 idle;立即下发到会话)
//   tt debug topent --clear 清除会话级 TOPENT override(回退配置默认)
//
// 与 debugctl 其它命令一致:均经 tt debug serve 的 REST 接口执行。

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	dbgEnvTimeout int
	// dbgEnvName 是 `debug env --name`：**创建会话时一并命名**。
	//
	// 需要它的理由是多会话：那几步（env / exec）都要能点名"我这条"会话，
	// 而点名要有名字。从前名字只能在 `debug start` 那一步给，于是前面几步只能裸奔。
	dbgEnvName string
)

// dbgSnapshot 会话快照中本命令关心的字段。
type dbgSnapshot struct {
	ID     string `json:"id"`
	Env    string `json:"env"`
	State  string `json:"state"`
	Topent string `json:"topent"` // 会话内手动设置的 TOPENT override(空=未设置)
	// TopentSession 是**启动时显式给**的 TOPENT(`tg debug start --ent` / 界面面板填的)。
	// 它是"用户给了就 export 进 shell"的那一个；空 = 用户没给，企业由登录 shell 的
	// $TOPENT 决定(见 TopentShell)。T100 的只读 SQL 以前按这三个值推 `ent`；GP 已经没有
	// 企业编号那层（账号取 db.accounts[0]），这几个字段只剩展示与 T100 会话排查用。
	TopentSession string `json:"topentSession"`
	TopentCfg     string `json:"topentCfg"`   // 配置级企业 TOPENT 默认(该环境 db.ent)
	TopentShell   string `json:"topentShell"` // 当前会话 shell 里的实际 TOPENT(连接时已按配置下发)
}

// dbgEnvItem 环境清单中的一项(取自 /api/hosts 返回的环境清单)。
type dbgEnvItem struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	Zone   string `json:"zone"`
	Topent string `json:"topent"`
	DBConn string `json:"dbConn"`
}

// dbgSettingsView /api/hosts 返回的环境清单(仅本命令需要)。
//
// 合并前这里读 /api/settings —— 那时它吐整个 debug 配置节,sshs 在节内。
// 拆节后环境清单归顶层 hosts,统一服务(tg serve)把它作为顶层 sshs 下发。
type dbgSettingsView struct {
	SSHs []dbgEnvItem `json:"sshs"`
}

type dbgEnvSSH struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	User string `json:"user"`
}

// debugEnvCmd 查看/切换当前调试环境(SSH 配置)。
var debugEnvCmd = &cobra.Command{
	Use:   "env [环境名]",
	Short: "查看/切换当前调试环境(SSH 配置,需 serve 在运行)",
	Long: `查看或切换当前调试环境(SSH 连接配置)。

无参数:列出 config.json debug.sshs 中的全部环境与当前会话所在环境/状态。
带环境名:把当前环境切到该环境并重连会话——先结束当前调试(若有),
再按新环境的 SSH/区域重连到空闲(idle)。

当前环境只存在于运行时(不写 config.json):选环境 = 连会话,服务重启后回到列表首条。

需要 tt debug serve 在运行;控制端命令会自动发现其地址。`,
	Example: `  tt debug env                 # 列出环境与当前会话
  tt debug env 示例测试区       # 切换当前环境到该环境(等会话回空闲)
  tt debug env --json          # 环境清单 JSON 输出`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return dbgEnvSwitch(args[0])
		}
		return dbgEnvList()
	},
}

// dbgSettings 拉取 /api/hosts 并解出环境清单。
func dbgSettings() (*dbgSettingsView, error) {
	data, err := dbgAPI("GET", "/api/hosts", nil)
	if err != nil {
		return nil, err
	}
	var s dbgSettingsView
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("解析设置失败: %w", err)
	}
	return &s, nil
}

// dbgSessions 拉取会话列表。
func dbgSessions() ([]struct {
	ID    string `json:"id"`
	Env   string `json:"env"`
	State string `json:"state"`
}, error) {
	data, err := dbgAPI("GET", "/api/sessions", nil)
	if err != nil {
		return nil, err
	}
	var r struct {
		Sessions []struct {
			ID    string `json:"id"`
			Env   string `json:"env"`
			State string `json:"state"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("解析会话列表失败: %w", err)
	}
	return r.Sessions, nil
}

// dbgEnvNamesOf 把 `/api/hosts` 的环境清单摊成名字列表（F6 的"可用环境：…"用它）。
//
// 目标层要"配置里全部环境名"这一个输入；它只经由 HTTP 拿（CLI 侧不另开一条读配置的路）。
func dbgEnvNamesOf(s *dbgSettingsView) []string {
	if s == nil {
		return nil
	}
	out := make([]string, 0, len(s.SSHs))
	for _, e := range s.SSHs {
		if e.Name != "" {
			out = append(out, e.Name)
		}
	}
	return out
}

// dbgEnvList 列出环境与当前生效项。
func dbgEnvList() error {
	s, err := dbgSettings()
	if err != nil {
		return err
	}
	// "当前会话"向目标层要（唯一就落；多会话又没给 --session 时落不上 → 不假报一个）。
	// 从前这里是取回包数组的第一项：多会话下列表顺序决定它说"当前"是哪条。
	curEnv, curState := "", ""
	if cur, err := dbgTargetSession(dbgEnvNamesOf(s), dbgRequest()); err == nil {
		curEnv, curState = cur.Env, cur.State
	}
	if IsJSON() {
		type out struct {
			Envs    []any `json:"envs"`
			Session any   `json:"session,omitempty"`
		}
		envs := make([]any, 0, len(s.SSHs))
		for _, e := range s.SSHs {
			envs = append(envs, e)
		}
		var sess any
		if curEnv != "" {
			sess = map[string]any{"env": curEnv, "state": curState}
		}
		return printJSON(out{Envs: envs, Session: sess})
	}

	if curEnv != "" {
		fmt.Printf("当前会话: %s (%s)\n", curEnv, dbgStateLabel(curState))
	} else {
		fmt.Println("当前会话: 未连接(用 tt debug env <环境名> 连接,或 tt debug start 启动调试)")
	}
	if len(s.SSHs) == 0 {
		fmt.Println("环境: 未配置 sshs")
		return nil
	}
	fmt.Printf("环境(%d):\n", len(s.SSHs))
	for _, e := range s.SSHs {
		name := e.Name
		if curEnv == e.Name {
			name += " ← 当前会话"
		}
		extra := e.DBConn
		if e.Topent != "" {
			extra += " (TOPENT默认=" + e.Topent + ")"
		}
		fmt.Printf("  %s %s:%d  zone=%-4s %s\n", name, e.Host, e.Port, e.Zone, extra)
	}
	return nil
}

// dbgEnvSwitch 「连到该环境」(Q26):连上该环境并把会话重连到 idle/exit;已在跑的那条会被收掉,这是有意的(Q27)。
func dbgEnvSwitch(name string) error {
	// 先校验环境存在(给出更友好的错误)
	s, err := dbgSettings()
	if err != nil {
		return err
	}
	found := false
	for _, e := range s.SSHs {
		if e.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("环境 %q 不存在(可先运行 tt debug env 查看已配置环境)", name)
	}
	// 目标不是当前会话环境、且会话不在空闲时,切换会结束当前调试——先提示(Web 端同样要求确认)
	cur := dbgAmbientSession()
	if cur != nil {
		if cur.Env != name && cur.State != "idle" && cur.State != "exit" {
			fmt.Printf("注意:切换会结束当前调试(会话状态 %s)并重连到 %s …\n", dbgStateLabel(cur.State), name)
		}
	}

	// 服务端不再自己挑"当前会话"（WP4b：那五处 `Manager.Current()` 兜底全删了，Q4/C8.2 硬切）。
	// "要收掉哪一条"必须由这边显式给 —— 否则多会话下服务端只能报 409 并列候选。
	// 这里给的是**服务端原始句柄**（C5.4：REST 只认 id），目标层的落位在 dbgAmbientSession 里做：
	// 唯一就落（含 `--session` 点名的那些写法），池空/多条时报 null 并让服务端按同一条规矩判。
	body := map[string]any{"env": name}
	// `--name`：创建即命名（多会话下前面几步才能点名到自己这条）。
	if n := strings.TrimSpace(dbgEnvName); n != "" {
		body["name"] = n
	}
	if cur != nil {
		body["session"] = cur.ID
	}
	data, err := dbgAPI("POST", "/api/sessions/switch", body)
	if err != nil {
		return err
	}
	var r struct {
		SessionID string `json:"sessionId"`
		Env       string `json:"env"`
		State     string `json:"state"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return fmt.Errorf("解析切换结果失败: %w", err)
	}
	fmt.Printf("切换会话到 %s …\n", r.Env)

	// 服务端异步登录到 idle;已在目标环境则幂等返回当前状态,不等 idle
	if r.State == "loading" || r.State == "" {
		deadline := time.Now().Add(time.Duration(dbgEnvTimeout) * time.Second)
		for {
			snap, err := dbgSnap(r.SessionID)
			if err == nil && snap != nil {
				if snap.State == "idle" {
					fmt.Printf("已切换:环境 %s 会话空闲(idle),可启动调试\n", r.Env)
					return nil
				}
				if snap.State == "exit" {
					return fmt.Errorf("切换失败:会话 %s 已断开", r.Env)
				}
			} else if err != nil {
				// 快照失败可能是会话被移除(登录失败):确认真不在列表里再提前报错
				gone := true
				if sessions, e2 := dbgSessions(); e2 == nil {
					for _, s2 := range sessions {
						if s2.ID == r.SessionID {
							gone = false
							break
						}
					}
				}
				if gone {
					return fmt.Errorf("切换失败:会话 %s 未建立成功(环境 %s 连接失败)", r.SessionID, r.Env)
				}
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("切换会话超时(%ds),请用 tt debug status 查看状态", dbgEnvTimeout)
			}
			time.Sleep(1 * time.Second)
		}
	}
	fmt.Printf("已在环境 %s(会话状态 %s),无需切换\n", r.Env, dbgStateLabel(r.State))
	return nil
}

// dbgSnap 取会话快照。
func dbgSnap(id string) (*dbgSnapshot, error) {
	if id == "" {
		return nil, fmt.Errorf("会话 id 为空")
	}
	data, err := dbgAPI("GET", dbgSessionURL(id, ""), nil)
	if err != nil {
		return nil, err
	}
	var s dbgSnapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("解析会话快照失败: %w", err)
	}
	return &s, nil
}

// dbgStateLabel 状态中文名。
func dbgStateLabel(s string) string {
	switch s {
	case "idle":
		return "空闲"
	case "loading":
		return "连接中…"
	case "stopped":
		return "已停站"
	case "running":
		return "运行中"
	case "exit":
		return "已断开"
	default:
		return s
	}
}

func orDefault(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func init() {
	debugEnvCmd.Flags().IntVar(&dbgEnvTimeout, "timeout", 120, "切换等待会话空闲超时(秒)")
	debugEnvCmd.Flags().StringVar(&dbgEnvName, "name", "", "创建会话时一并命名（多条会话下后续命令用 --session <别名> 点名；取值同 debug start --name）")
	// env 作用在"某一条会话"上（契约 1. 的命令角色表），所以认 --session：
	// 从前它们取列表第一个，多会话下"改的是哪条"由列表顺序决定。
	dbgRegisterSessionFlag(debugEnvCmd)
	addClientURLFlag(debugEnvCmd)
	Group.AddCommand(debugEnvCmd)
}
