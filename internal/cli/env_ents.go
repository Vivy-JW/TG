package cli

// tt env ents —— 一个环境“支持哪些企业（ENT）、各用哪个数据库账号”。
//
// 数据来自 **config.json**（`hosts.sshs[].ents`，由设置页「ENT」页签或本命令的
// `--refresh` 从 gzou_t 同步而来）。默认**不联网**：环境属于配置，看清单不该依赖
// VPN/内网是否通；同步是一个显式动作。
//
// 环境一律点名：不给环境名 = 逐个环境都列（不是挑一个默认）。某个环境同步失败只影响
// 它自己，其余照常输出，最后用退出码 3 表示“至少有一个没同步成”。

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/debug"
	"tt/internal/output"
)

type envEntsView struct {
	Env        string            `json:"env"`
	Zone       string            `json:"zone,omitempty"`
	Dialect    string            `json:"dialect,omitempty"`
	Target     string            `json:"target,omitempty"`
	Account    string            `json:"account,omitempty"` // 查 gzou_t 用的系统账号（恒 ds，说明用）
	Synced     bool              `json:"synced"`
	SyncedAt   string            `json:"syncedAt,omitempty"`
	AgeSeconds int               `json:"ageSeconds,omitempty"`
	Count      int               `json:"count"`
	Ents       []config.EntEntry `json:"ents,omitempty"`
	Notes      []string          `json:"notes,omitempty"`
	Error      string            `json:"error,omitempty"`
}

func newEnvEntsCmd() *cobra.Command {
	var (
		refresh bool
		cached  bool
		onlyEnt int
	)
	cmd := &cobra.Command{
		Use:   "ents [<环境名>]",
		Short: "列出环境支持的企业编号（ENT）与账号（读配置；--refresh 才连库同步）",
		Long: `列出环境在企业目录（gzou_t）里启用的企业编号（gzou001），以及各自的数据库账号
（gzou003）。这份清单存在 config.json 的 hosts.sshs[].ents 里，**默认直接读配置、
不联网**；要更新就显式同步（设置页「ENT」页签的「从服务器刷新」是同一个动作）。

  tt env ents                   # 逐个环境列（读配置，秒回、离线可用）
  tt env ents 主机109           # 只看这个环境
  tt env ents --ent 99          # 只留含企业 99 的环境
  tt env ents --refresh         # 连库读 gzou_t，并**写回配置**（同步）
  tt env ents --json            # 结构化

同步固定用系统账号 ds 读 gzou_t；清单里的账号会顺带补进该环境的账号清单（缺的才补，
密码优先用已有条目，否则按 T100 惯例 账号=密码，输出会点名提示）。
环境一律点名：不给环境名就是逐个环境都列 —— 某个环境同步失败只影响它自己（退出码 3）。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if refresh && cached {
				return output.Errorf("USAGE", 2, "--cached 是默认行为（读配置），与 --refresh 二选一")
			}
			name := ""
			if len(args) == 1 {
				name = strings.TrimSpace(args[0])
			} else if strings.TrimSpace(common.Env) != "" {
				name = strings.TrimSpace(common.Env)
			}
			return runEnvEnts(name, refresh, onlyEnt)
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "连库读 gzou_t 并写回 config.json（同步）")
	cmd.Flags().BoolVar(&cached, "cached", false, "只读配置、不连网（默认行为，保留兼容）")
	_ = cmd.Flags().MarkHidden("cached")
	cmd.Flags().IntVar(&onlyEnt, "ent", 0, "只留含这个企业编号的环境（0 = 全部）")
	return cmd
}

func runEnvEnts(name string, refresh bool, onlyEnt int) error {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return err
	}
	root, err := config.Load(path)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(root.Hosts.SSHs))
	if name != "" {
		if root.Hosts.ByName(name) == nil {
			return output.Errorf("ENV_UNKNOWN", 2, "没有名为 %q 的环境（用 tt env list 看全部）", name)
		}
		names = append(names, name)
	} else {
		for _, e := range root.Hosts.SSHs {
			names = append(names, e.Name)
		}
	}
	if len(names) == 0 {
		return output.Errorf("ENV_NONE", 2, "配置里还没有任何环境（%s）；先 tt serve 到设置页添加", path)
	}

	cfg, err := debug.LoadConfig(path)
	if err != nil {
		return err
	}
	cfg.DataDir = filepath.Dir(path)

	views := make([]envEntsView, 0, len(names))
	failed := 0
	for _, n := range names {
		v := envEntsView{Env: n, Account: dbconfig.SystemAccount}
		if e := root.Hosts.ByName(n); e != nil {
			v.Zone = e.Zone
			if e.DB != nil {
				v.Dialect, v.Target = e.DB.Type, e.DB.Address()
			}
		}
		if !refresh {
			ents, syncedAt := root.Hosts.EntsOf(n)
			v.Ents = filterEnts(ents, onlyEnt)
			v.Count = len(v.Ents)
			if onlyEnt > 0 && v.Count == 0 {
				continue
			}
			v.Synced = len(ents) > 0
			v.SyncedAt = syncedAt
			if t, err := time.Parse(time.RFC3339, syncedAt); err == nil {
				v.AgeSeconds = int(time.Since(t).Seconds())
			}
			views = append(views, v)
			continue
		}

		// --refresh：连库同步 + 写回配置。密码优先取账号清单（唯一的编辑入口），
		// 没有就留空 —— SaveEnvEnts 会按惯例补并记入“新增账号”。
		sub := cfg.CloneEnv(n)
		if sub == nil {
			v.Error = "环境不存在"
			failed++
			views = append(views, v)
			continue
		}
		sub.DataDir = cfg.DataDir
		l, err := debug.LookupEnts(sub, debug.EntListOpt{Refresh: true})
		if err != nil {
			v.Error = err.Error()
			failed++
			views = append(views, v)
			continue
		}
		synced := make([]config.SyncedEnt, 0, len(l.Ents))
		for _, m := range l.Ents {
			pwd := ""
			if sub.DB != nil {
				if p, ok := sub.DB.PasswordFor(m.Account); ok {
					pwd = p
				}
			}
			synced = append(synced, config.SyncedEnt{Ent: m.Ent, Account: m.Account, Password: pwd})
		}
		at := time.Now()
		added, err := config.SaveEnvEnts(path, n, synced, at)
		if err != nil {
			v.Error = err.Error()
			failed++
			views = append(views, v)
			continue
		}
		for _, s := range synced {
			v.Ents = append(v.Ents, config.EntEntry{Ent: s.Ent, Account: s.Account})
		}
		v.Ents = filterEnts(v.Ents, onlyEnt)
		v.Count = len(v.Ents)
		if onlyEnt > 0 && v.Count == 0 {
			continue
		}
		v.Synced = true
		v.SyncedAt = at.Format(time.RFC3339)
		if len(added) > 0 {
			v.Notes = append(v.Notes,
				"新增账号并写入账号清单："+strings.Join(added, "、")+"（密码按惯例=账号名，请在设置里改成真实口令）")
		}
		views = append(views, v)
	}

	if common.IsJSON() {
		env := map[string]any{
			"ok": failed == 0, "count": len(views), "failed": failed,
			"source": map[bool]string{true: "live", false: "config"}[refresh],
			"envs":   views,
		}
		if err := common.PrintJSON(env); err != nil {
			return err
		}
	} else {
		writeEnvEnts(os.Stdout, views, refresh)
	}

	if failed > 0 {
		return partialEnvErr{failed: failed}
	}
	return nil
}

func filterEnts(ents []config.EntEntry, onlyEnt int) []config.EntEntry {
	if onlyEnt <= 0 {
		return ents
	}
	out := make([]config.EntEntry, 0, 1)
	for _, e := range ents {
		if e.Ent == onlyEnt {
			out = append(out, e)
		}
	}
	return out
}

func writeEnvEnts(w *os.File, views []envEntsView, refreshed bool) {
	for i, v := range views {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if v.Error != "" {
			fmt.Fprintf(w, "!! 环境 %s 同步失败：%s\n", v.Env, v.Error)
			continue
		}
		meta := []string{"区域 " + orDash(v.Zone)}
		if v.Dialect != "" {
			meta = append(meta, v.Dialect)
		}
		if v.Target != "" {
			meta = append(meta, v.Target)
		}
		switch {
		case refreshed:
			fmt.Fprintf(w, "环境 %s   %s   刚同步\n", v.Env, strings.Join(meta, " · "))
		case !v.Synced:
			fmt.Fprintf(w, "环境 %s   %s   **未同步**（跑 tt env ents --refresh，或设置页「ENT」页签）\n", v.Env, strings.Join(meta, " · "))
		default:
			fmt.Fprintf(w, "环境 %s   %s   同步于 %s（%s前）\n",
				v.Env, strings.Join(meta, " · "), humanTime(v.SyncedAt), humanAge(v.AgeSeconds))
		}
		if len(v.Ents) > 0 {
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "  ENT\t账号")
			for _, m := range v.Ents {
				acct := m.Account
				if acct == "" || acct == "-" {
					acct = "-（未配账号）"
				}
				fmt.Fprintf(tw, "  %d\t%s\n", m.Ent, acct)
			}
			_ = tw.Flush()
			fmt.Fprintf(w, "  共 %d 个企业；--at 的值：%s@<上表中的企业>\n", v.Count, v.Env)
		}
		for _, n := range v.Notes {
			fmt.Fprintf(w, "  注：%s\n", n)
		}
	}
}

// partialEnvErr 表示“已经逐个环境报过了，只是有失败”。AlreadyReported 让根命令
// 不再重复打印一条错误（否则 stdout 上会出现两份说法）。
type partialEnvErr struct{ failed int }

func (e partialEnvErr) Error() string {
	return fmt.Sprintf("%d 个环境没同步成功", e.failed)
}
func (e partialEnvErr) ExitCode() int         { return 3 }
func (e partialEnvErr) AlreadyReported() bool { return true }

// humanTime 把 RFC3339 说成本地“年-月-日 时:分”。
func humanTime(raw string) string {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	return raw
}

// humanAge 把秒数说成人话（与 at 包同一套口径，两处输出要对得上）。
func humanAge(sec int) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%d 秒", sec)
	case sec < 3600:
		return fmt.Sprintf("%d 分钟", sec/60)
	default:
		return fmt.Sprintf("%d 小时", sec/3600)
	}
}
