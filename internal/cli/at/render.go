package at

// --at-list 的渲染。输出是**固定分段**的契约（文法 / 来源 / 可用值 / 示例 / 备注），
// 人和 agent 读同一份；--json 是同一份数据，不是另一套说法。
//
// 关于“取不到企业清单”：ENT 只能靠连库（远端 gzou_t）或它的落盘快照得到。
// 没开 VPN、换了机器/库、快照坏了，都会让某个环境**一条都拿不到**。
// 这种情况**不许静默**：逐个环境给出 `!!` 行 + 原因 + 可执行的一句话，
// 并在末尾汇总“这个列表不完整”。但也不因一个环境把整张表判死 —— 其余环境照常列。

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
)

type contractValue struct {
	Handle     string `json:"handle"`
	Env        string `json:"env"`
	Ent        int    `json:"ent,omitempty"`
	Account    string `json:"account,omitempty"`
	Zone       string `json:"zone,omitempty"`
	DBTarget   string `json:"dbTarget,omitempty"`
	Source     string `json:"source,omitempty"` // live | snapshot | snapshot-stale
	Stale      bool   `json:"stale,omitempty"`
	AgeSeconds int    `json:"ageSeconds,omitempty"`
}

// contractFailure 一个“这个环境拿不到企业清单”的说明。
//
// Reason 是给机器看的稳定码；Error/Action 是给人和 agent 读的两句话。
type contractFailure struct {
	Env    string `json:"env"`
	Reason string `json:"reason"` // no-snapshot | snapshot-unreadable | config
	Error  string `json:"error,omitempty"`
	Action string `json:"action"`
}

type contract struct {
	OK          bool              `json:"ok"`
	Complete    bool              `json:"complete"`
	Command     string            `json:"command"`
	Scope       string            `json:"scope"`
	EntRequired bool              `json:"entRequired,omitempty"`
	Grammar     string            `json:"grammar"`
	Sources     []string          `json:"sources,omitempty"`
	Values      []contractValue   `json:"values"`
	Failed      []contractFailure `json:"failed,omitempty"`
	Example     string            `json:"example,omitempty"`
	Notes       []string          `json:"notes,omitempty"`
	Message     string            `json:"message,omitempty"`
}

// PrintContract 打一条命令的 --at 契约，返回建议的退出码。
//
// 退出码只有两种：
//
//	0 —— 能给出可用值（个别环境取不到只算“不完整”，见 Complete/Failed）；
//	3 —— **一条可用值都拿不到**（比如所有环境都没有快照、且配置也读不到）：
//	     这时空列表不代表“没有企业”，而是“问不到”，脚本必须能分开这两件事。
func PrintContract(w io.Writer, cmd *cobra.Command) (int, error) {
	scope := ScopeOf(cmd)
	sc := scope.String()
	c := contract{OK: true, Complete: true, Command: cmd.CommandPath(), Scope: sc, EntRequired: false, Values: []contractValue{}}
	exit := 0

	switch scope {
	case None:
		c.Message = noneMessage(cmd)
	case Env, EnvFilter, EnvEnt:
		c.Grammar = "<环境>"
		c.Sources = []string{"<环境> ← config.json 的 hosts.sshs[].name（本机配置，无网络）"}
		c.Example = exampleOf(cmd, "<环境>")
		c.Notes = append(c.Notes, "环境一律点名：没有默认环境可回落")
		if scope == EnvFilter {
			c.Notes = append(c.Notes,
				"本命令是会话寻址：--at 只用来收窄“在哪个环境里找会话”",
				"指名会话用 --session（tg session ls --json 是它的下拉）")
		}
		if scope == EnvEnt {
			c.Grammar = "<环境>（也可以写 <环境>@<据点编号>，据点可选）"
			c.Sources = append(c.Sources, "据点   = <环境> + \"@\" + 据点编号（azp_file.azp01，字母开头）")
			c.Example = exampleOf(cmd, "<环境>@<据点编号>")
			c.Notes = append(c.Notes,
				"据点候选在服务器库里，--at-list 是纯本机的，不拨 SSH",
				"据点清单：tg site list --env <环境>",
				"GP 里没有企业编号：@ 右边写据点编号，写数字会直接报错")
		}
		values, err := envValues()
		if err != nil {
			c.Complete = false
			c.Failed = append(c.Failed, contractFailure{Reason: "config", Error: err.Error(), Action: "检查配置文件：tg config path"})
			exit = 3
			break
		}
		if len(values) == 0 {
			c.Complete = false
			c.Failed = append(c.Failed, contractFailure{Reason: "config", Error: "配置里还没有任何环境", Action: "tg serve 到设置页添加环境，或 tg env list 确认"})
			exit = 3
			break
		}
		c.Values = values
	}

	if common.JSON {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		return exit, enc.Encode(c)
	}
	writeHuman(w, c)
	return exit, nil
}

func writeHuman(w io.Writer, c contract) {
	fmt.Fprintf(w, "%s —— --at 取值\n\n", c.Command)
	if c.Message != "" {
		fmt.Fprintf(w, "  %s\n", strings.ReplaceAll(c.Message, "\n", "\n  "))
		return
	}
	fmt.Fprintf(w, "  文法    %s\n", c.Grammar)
	if len(c.Sources) > 0 {
		fmt.Fprintf(w, "  来源\n")
		for _, s := range c.Sources {
			fmt.Fprintf(w, "          %s\n", s)
		}
	}

	fmt.Fprintf(w, "  可用值\n")
	if len(c.Values) == 0 {
		fmt.Fprintf(w, "          （无）\n")
	}
	for _, v := range c.Values {
		line := v.Handle
		extra := make([]string, 0, 3)
		if v.Account != "" {
			extra = append(extra, v.Account)
		}
		if v.Zone != "" {
			extra = append(extra, "区域 "+v.Zone)
		}
		if v.Source != "" {
			label := "快照"
			if v.Source == "config" {
				label = "配置"
			}
			switch {
			case v.Stale && v.AgeSeconds > 0:
				label = fmt.Sprintf("%s（%s前，已过期）", label, humanAge(v.AgeSeconds))
			case v.Stale:
				label = label + "（已过期）"
			case v.AgeSeconds > 0:
				label = fmt.Sprintf("%s（同步于 %s前）", label, humanAge(v.AgeSeconds))
			}
			extra = append(extra, label)
		}
		if len(extra) > 0 {
			line += "  " + strings.Join(extra, " · ")
		}
		fmt.Fprintf(w, "          %s\n", line)
	}

	// 取不到的环境：单独一段，原因 + 下一步都写出来。
	if len(c.Failed) > 0 {
		fmt.Fprintf(w, "  取不到（不是“没有企业”，是“问不到”）\n")
		for _, f := range c.Failed {
			who := f.Env
			if who == "" {
				who = "配置"
			}
			msg := f.Error
			if msg == "" {
				msg = f.Reason
			}
			if f.Action != "" {
				msg += " —— " + f.Action
			}
			fmt.Fprintf(w, "          !! %s：%s\n", who, msg)
		}
	}
	if !c.Complete {
		fmt.Fprintf(w, "  注意    上面的列表不完整：有环境取不到企业清单（见 !! 行）\n")
	}

	if c.Example != "" {
		fmt.Fprintf(w, "  示例    %s\n", c.Example)
	}
	if len(c.Notes) > 0 {
		fmt.Fprintf(w, "  备注\n")
		for _, n := range c.Notes {
			fmt.Fprintf(w, "          %s\n", n)
		}
	}
}

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

func envValues() ([]contractValue, error) {
	envs, _, err := LoadEnvs()
	if err != nil {
		return nil, err
	}
	out := make([]contractValue, 0, len(envs))
	for _, e := range envs {
		// 只给环境：不再附带区域/库地址 —— 那些字段会被误读成 @企业。
		out = append(out, contractValue{Handle: e.Name, Env: e.Name})
	}
	return out, nil
}

// exampleOf 用命令自己的路径与位置参数拼一条可复制的例子。
func exampleOf(cmd *cobra.Command, atValue string) string {
	usage := cmd.Use
	if i := strings.Index(usage, " "); i >= 0 {
		usage = strings.TrimSpace(usage[i+1:])
	} else {
		usage = ""
	}
	rest := ""
	if usage != "" {
		rest = " " + usage
	}
	return fmt.Sprintf("%s --at %s%s", cmd.CommandPath(), atValue, rest)
}

// noneMessage 解释“这条命令为什么没有 --at”，并给出可执行的替代写法。
func noneMessage(cmd *cobra.Command) string {
	path := pathOf(cmd)
	switch {
	case path == "":
		return "--at 是给具体命令用的：先选命令，再加 --at-list（例如 tg debug start --at-list）。"
	case strings.HasPrefix(path, "session"):
		return "本命令不接受 --at：会话身份来自 --session（tg session ls --json 可看全部）；\n" +
			"环境与企业跟着会话走，不用单独给。"
	default:
		return "本命令没有目标参数（不需要环境/企业）。"
	}
}
