package at

// --at-list 在 **cobra 解析之前** 处理。
//
// 为什么不能放在 PersistentPreRunE 里：cobra 的顺序是 ValidateArgs → PersistentPreRunE，
// 而 `tt debug start --at-list` 本来就不该带作业名 —— 放到 pre-run 会被"缺参数"拦住。
// 所以在 Execute 开头扫一遍 argv：只要出现 --at-list，就定位命令、打印契约、退出 0。

import (
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

// ListTarget 判断这次要不要走 --at-list；要的话返回目标命令。
//
// 顺手把输出形态（--json/--format）也认了 —— 这一步比 cobra 早，否则 --at-list 的
// 结构化输出永远拿不到。
func ListTarget(root *cobra.Command, args []string) (*cobra.Command, bool) {
	found := false
	rest := make([]string, 0, len(args))
	for _, a := range args {
		switch {
		case a == "--at-list" || a == "--at-list=true":
			found = true
		case a == "--at-list=false":
			// 显式关掉：当没给
		default:
			rest = append(rest, a)
		}
	}
	if !found {
		return nil, false
	}
	for i := 0; i < len(rest); i++ {
		switch a := rest[i]; {
		case a == "--json":
			common.JSON = true
		case a == "--csv":
			common.CSV = true
		case strings.HasPrefix(a, "--format="):
			common.Format = strings.TrimPrefix(a, "--format=")
		case a == "--format" && i+1 < len(rest):
			common.Format = rest[i+1]
			i++
		}
	}
	cmd, _, err := root.Find(rest)
	if err != nil || cmd == nil {
		return root, true
	}
	return cmd, true
}
