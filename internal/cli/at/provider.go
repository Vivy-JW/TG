package at

// 取值来源：**只读本机**。环境来自 config.json 的 hosts.sshs；--at 的 @ 右边
// 在 TG（只有 TOP GP）里是据点编号，候选要连库才拿得到，见 tg site list。

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/config"
)

// productGP 是 GP 环境的产品标识（config.json hosts.sshs[].product）。
const productGP = "gp"

// EnvInfo 渲染与环境校验共用的最小环境信息。
type EnvInfo struct {
	Name     string
	Zone     string
	Dialect  string // oracle | kingbase
	DBTarget string // host:port/service（未配库为空）
	Product  string // gp | t100（空 = 老配置里没写，按 T100 处理）
	Company  string // GP 的资料库编号（azp_file.azp03），空 = ds
}

// productOf 读某环境的产品标识；读不到配置就返回空串（不拦人）。
//
// 为什么 --at 要看产品：`<环境>@<x>` 在 T100 里 x 是企业编号（纯数字），
// 在 TOP GP 里 x 是据点编号（字母）。同一个写法两种含义，必须按环境分派。
func productOf(env string) string {
	envs, _, err := LoadEnvs()
	if err != nil {
		return ""
	}
	for _, e := range envs {
		if e.Name == env {
			return strings.ToLower(strings.TrimSpace(e.Product))
		}
	}
	return ""
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
		info := EnvInfo{Name: e.Name, Zone: e.Zone, Product: e.Product, Company: e.Company}
		if e.DB != nil {
			info.Dialect = e.DB.Type
			info.DBTarget = e.DB.Address()
		}
		out = append(out, info)
	}
	return out, path, nil
}

// checkEnv 校验 --at 的环境在配置里；失败时列出候选（可直接换一个填）。
//
// 读不到配置（首次使用、配置坏了）时**跳过**：那种情况该由命令自己报，
// 这里再报一次只会把真正的原因盖住。
func checkEnv(cmd *cobra.Command, env string) error {
	envs, path, err := LoadEnvs()
	if err != nil {
		return nil
	}
	if len(envs) == 0 {
		return fmt.Errorf("配置里还没有任何环境（%s）；先在设置里加一个，或 tg env list 确认", path)
	}
	for _, e := range envs {
		if e.Name == env {
			return nil
		}
	}
	return fmt.Errorf("%s 的 --at 指向了不存在的环境 %q。\n可用的是：%s\n（看每个环境的候选：tg site list --env <环境>）",
		cmd.CommandPath(), env, strings.Join(envNames(envs), "、"))
}

func envNames(envs []EnvInfo) []string {
	out := make([]string, 0, len(envs))
	for _, e := range envs {
		out = append(out, e.Name)
	}
	return out
}
