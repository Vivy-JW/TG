package file

// $TOP 的来源。
//
// TG 只面向 TOP GP：$TOP 就是环境配置里的 workspace（`/u1/<实例>/tiptop`），
// 客制根是 custRoot（`/u1/<实例>/topcust`）。GP 没有 T100 的“登录区域（zone）”，
// 也没有区域菜单可探，所以这里不再有 ProbeTEnv / 缓存 / --refresh 那套：
// 配置里写的是什么就用什么，写错了直接报错让人去设置页补。

import (
	"strings"
)

// resolveTop 取本次的 $TOP：直接读环境配置，不探测、不缓存。
func (t *target) resolveTop() error {
	top := cleanTop(t.ssh.Workspace)
	if top == "" {
		return usageErr("环境 %s 没有配置 $TOP（workspace）；在设置页「站点管理 → SSH 服务器」补上，例如 /u1/toptest/tiptop", t.env)
	}
	t.top = top
	return nil
}

// cleanTop 去掉首尾空白与尾斜杠（后面所有路径比较都按“无尾斜杠”做前缀判断）。
func cleanTop(top string) string {
	top = strings.TrimSpace(top)
	if top == "" {
		return ""
	}
	return strings.TrimSuffix(top, "/")
}
