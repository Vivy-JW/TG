package dict

import (
	"fmt"
	"strings"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict/dbsync"

	"github.com/spf13/cobra"
)

// helpCmdFamilies 数据命令 → 它依赖的数据族,用于 --help 末尾的「本地数据」提示。
// 命令不在表里(db/mirror/spill 等)不显示提示。
var helpCmdFamilies = map[string][]string{
	"r.t":  {"table", "progtable"},
	"r.v":  {"check"},
	"scc":  {"scc"},
	"desc": {"spec"},
	"r.q":  {"win"},
	"msg":  {"msg"},
	"sysp": {"param"},
	"docp": {"param"},
	"prog": {"prog", "progtable", "subprog"},
}

// baseHelpFunc 是 cobra 的默认 help 实现。用一个无父命令的临时命令取出来,
// 免得取到自己或别的命令 SetHelpFunc 装过的那份(那会重复输出)。
var baseHelpFunc = (&cobra.Command{}).HelpFunc()

// versionLine 一行版本描述。版本号由打包脚本经 -ldflags 注入到 common.Version;
// 本地 go build 未注入时显示 devel(带 commit 的完整描述见 `tt version`)。
func versionLine() string {
	if common.Version != "" {
		return common.Version
	}
	return "devel"
}

// localDataHint 返回一行「本地数据」状态;keys 为空 = 全部数据族。
// 只读检查本地库,任何失败都静默(不该因为环境问题让 --help 报错)。
func localDataHint(keys ...string) string {
	p := helpLocalDBPath()
	if p == "" {
		return ""
	}
	st, err := dbsync.InspectLocal(p)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("本地数据(%s): %s", st.Path, st.Summary(keys...))
}

// helpLocalDBPath help 里能确定的那份本地库：显式 -d/TDICT_DB，或当前点名的 local:<环境>。
// 没点名环境时不猜（按环境分库，选错比不显示更坏）。
func helpLocalDBPath() string {
	if p := explicitLocalDB(); p != "" {
		return config.AbsPath(p)
	}
	if name, ok := localEnvName(strings.TrimSpace(common.Env)); ok && name != "" {
		return config.DictDBPathForEnv(dictCfgPath(), name)
	}
	if name, ok := localEnvName(strings.TrimSpace(queryCfgSource())); ok && name != "" {
		return config.DictDBPathForEnv(dictCfgPath(), name)
	}
	return ""
}

// attachDataHint 在命令组上装 help 钩子。子命令继承命令组的 help 函数,
// 所以这一处就能让所有命令的 --help 末尾都带上版本与本地数据状态。
func attachDataHint() {
	Group.SetHelpFunc(func(c *cobra.Command, args []string) {
		baseHelpFunc(c, args)
		out := c.OutOrStdout()
		fmt.Fprintln(out, "\n版本: tt dict "+versionLine())
		if c == Group {
			fmt.Fprintln(out, localDataHint())
			return
		}
		if keys, ok := helpCmdFamilies[c.Name()]; ok {
			fmt.Fprintln(out, localDataHint(keys...))
		}
	})
}
