package at

// 取值来源：**只读本机**。环境来自 config.json 的 hosts.sshs；企业来自该环境的
// entdir 快照（与 tt debug ents / tt dict 同一份）。--at-list 与校验都走这里，
// 因此“下拉里看到的”和“用起来校验的”永远是同一套。

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/debug"
)

// EnvInfo 渲染与环境校验共用的最小环境信息。
type EnvInfo struct {
	Name     string
	Zone     string
	Dialect  string // oracle | kingbase
	DBTarget string // host:port/service（未配库为空）
}

// LoadEnvs 读环境清单（纯本地）。
func LoadEnvs() ([]EnvInfo, string, error) {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return nil, "", err
	}
	root, err := config.Load(path)
	if err != nil {
		return nil, path, err
	}
	out := make([]EnvInfo, 0, len(root.Hosts.SSHs))
	for _, e := range root.Hosts.SSHs {
		info := EnvInfo{Name: e.Name, Zone: e.Zone}
		if e.DB != nil {
			info.Dialect = e.DB.Type
			info.DBTarget = e.DB.Address()
		}
		out = append(out, info)
	}
	return out, path, nil
}

// SnapshotEnts 取某环境的“企业 → 账号”目录，**只读快照、不拨 SSH**。
// 没有快照返回 (nil, nil)：那是“这次没答案”，不是错误。
func SnapshotEnts(env string) (*debug.EntListing, error) {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return nil, err
	}
	cfg, err := debug.LoadConfig(path)
	if err != nil {
		return nil, err
	}
	cfg.DataDir = filepath.Dir(path)
	sub := cfg.CloneEnv(env)
	if sub == nil {
		return nil, fmt.Errorf("环境 %q 不存在", env)
	}
	sub.DataDir = cfg.DataDir
	return debug.LookupEnts(sub, debug.EntListOpt{Offline: true})
}

// checkEnv 校验 --at 的环境在配置里；失败时列出候选（可直接换一个填）。
//
// 读不到配置（首次使用、配置坏了）时**跳过**：那种情况该由命令自己报，
// 这里再报一次只会把真正的原因盖住。
func checkEnv(cmd *cobra.Command, env string) error {
	envs, path, err := LoadEnvs()
	if err != nil {
		return nil // 交给命令自己报“配置读不到”
	}
	if len(envs) == 0 {
		return fmt.Errorf("配置里还没有任何环境（%s）；先在设置里加一个，或 tt env list 确认", path)
	}
	for _, e := range envs {
		if e.Name == env {
			return nil
		}
	}
	return fmt.Errorf("%s 的 --at 指向了不存在的环境 %q。\n可用的是：%s\n（看每个环境的候选：tt env ents）",
		cmd.CommandPath(), env, strings.Join(envNames(envs), "、"))
}

// checkEnt 用快照校验企业是否在 gzou_t 的启用清单里；没有快照就跳过（不拦人）。
func checkEnt(cmd *cobra.Command, env string, ent int) error {
	l, err := SnapshotEnts(env)
	if err != nil || l == nil {
		return nil
	}
	for _, m := range l.Ents {
		if m.Ent == ent {
			return nil
		}
	}
	known := make([]string, 0, len(l.Ents))
	for i, m := range l.Ents {
		if i >= 12 {
			known = append(known, "…")
			break
		}
		known = append(known, fmt.Sprintf("%s@%d", env, m.Ent))
	}
	return fmt.Errorf("%s 的 --at 指向了企业 %d，但 %s 的 ENT 清单里没有它。\n可用的是：%s\n（完整清单：tt env ents %s；没同步过就跑 tt env ents --refresh）",
		cmd.CommandPath(), ent, env, strings.Join(known, "、"), env)
}

func envNames(envs []EnvInfo) []string {
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		out = append(out, e.Name)
	}
	return out
}
