package config

import (
	"fmt"
	"strings"
)

// 环境解析的唯一入口。
//
// 合并前「name → hosts.activeEnv → 首条」这个选择在四处各写了一遍
// (cli/dict/db.go、cli/dict/dbops.go、cli/dict/source.go、internal/debug/config.go),
// 每处的错误文案与边界行为还都不一样。这里收成一份:Hosts.pickName 决定"选哪个",
// Resolve 负责"找到它并说清是怎么选中的"。
//
// 选中的方式会随结果一起回给调用方(EnvRef.From),最终打进查询输出的环境头 ——
// "这个环境是命令行点名的,还是兜底取的首条"对排查"查错了地方"是必需信息。

// EnvRef 一次环境解析的结果。
type EnvRef struct {
	Env  *NamedSsh // 指向 Hosts.SSHs 里的元素(不是拷贝)
	Name string    // 生效环境名
	From string    // 它怎么被选中的:现在只有 flag(点名);tool-override / activeEnv / first 都已随"默认环境"移除
}

func (r *EnvRef) String() string {
	if r == nil {
		return ""
	}
	return r.Name
}

// pickName 环境名的唯一选择顺序:**只有显式 name**。
//
// 原先还有两级回落(hosts.activeEnv → 首条),已按"环境一律点名"的要求删除:
// 同机多环境时,回落到另一个环境去连,输出看起来完全正常 —— 与"上错号"无法区分。
// 所以这里不再猜:没给名字就是没给,由调用方报错并列出可用环境(见 Resolve)。
func (h *Hosts) pickName(name string) (string, string) {
	if name != "" {
		return name, "flag"
	}
	return "", ""
}

// Resolve 按名解析环境。**name 必须有值** —— 没给就报错并列出可用环境。
//
// 为什么不回落:显式点名的环境拼错了应当报错,绝不静默换一个环境去连
// (那正是"查错了地方却当成没有数据"的根因)。这条规则现在对"完全没给名字"
// 也同样适用 —— 没有名字就没有"合理的默认"可言。
func (h *Hosts) Resolve(name string) (*EnvRef, error) {
	if strings.TrimSpace(name) == "" {
		if names := h.Names(); len(names) > 0 {
			return nil, fmt.Errorf("必须指定环境：--at <环境> 或 --env <环境名>；可用环境: %s", strings.Join(names, "、"))
		}
		return nil, fmt.Errorf("尚未配置 SSH 环境(运行 tt serve 添加,或编辑 config.json hosts.sshs)")
	}
	target, from := h.pickName(name)
	for i := range h.SSHs {
		if h.SSHs[i].Name == target {
			return &EnvRef{Env: &h.SSHs[i], Name: target, From: from}, nil
		}
	}
	if names := h.Names(); len(names) > 0 {
		return nil, fmt.Errorf("未找到环境 %q；可用环境: %s（用 --at <环境> 或 --env <环境名>）", target, strings.Join(names, "、"))
	}
	return nil, fmt.Errorf("未找到环境 %q(配置里没有任何环境,运行 tt serve 添加)", target)
}

// ResolveForTool 在 Resolve 之上叠加工具级环境覆盖。
//
// 工具级覆盖(debug.activeEnv)已随"移除默认环境"一并删除:能落盘的默认值就是
// 会被静默沿用的默认值。所以这里现在等价于 Resolve —— 保留这个函数名是为了
// 调用点读起来仍然表达"这一步在解析该工具要用的环境"。
func (r *Root) ResolveForTool(tool, name string) (*EnvRef, error) {
	_ = tool // 目前没有任何工具级覆盖;参数留着以便将来真有"工具专属环境"时不必改调用点
	return r.Hosts.Resolve(name)
}
