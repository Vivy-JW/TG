package file

// 三个根，都来自环境配置，不探测、不缓存。
//
// TG 只面向 TOP GP：GP 的源码是两棵并列的树 ——
//
//	$TOP  = workspace（/u1/<实例>/tiptop）
//	$CUST = custRoot（/u1/<实例>/topcust）
//
// 而"看服务器文件"这类命令天生只有一个根。所以这里取两棵树的公共父目录当根：
//
//	实例目录 = /u1/<实例>（有 instance 就按它推，见 config.InstanceRoot）
//
// 于是 tiptop / topcust 都成了根下的普通子目录，越界判定仍只有一个前缀；
// 传相对路径时照顾老用法：不带 tiptop/ 或 topcust/ 前缀的仍按 $TOP 解释。
//
// 另外：GP 没有 T100 的"登录区域（zone）"，也没有区域菜单可探，所以这里不再有
// ProbeTEnv / 30 分钟缓存 / --refresh 那套 —— 配置里写的是什么就用什么。

import (
	"strings"

	"tg/internal/config"
)

// resolveRoots 取本次的三个根：实例目录（越界判定的根）、$TOP、$CUST。
func (t *target) resolveRoots() error {
	top := cleanPath(t.ssh.Workspace)
	if top == "" {
		return usageErr("环境 %s 没有配置 $TOP（workspace）；在设置页「站点管理 → SSH 服务器」补上，例如 /u1/toptest/tiptop", t.env)
	}
	inst := cleanPath(config.InstanceRoot(*t.ssh))
	if inst == "" {
		// 既没有 instance 也没有 workspace 的上级目录：退回 $TOP，至少单棵树可用。
		inst = top
	}
	t.top = top
	t.cust = cleanPath(t.ssh.CustRoot)
	t.inst = inst
	return nil
}

// cleanPath 去掉首尾空白与尾斜杠（后面所有路径比较都按"无尾斜杠"做前缀判断）。
func cleanPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(p, "/")
}
