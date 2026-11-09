package dict

// tt dict mirror 已迁移到 tt file mirror。这里只留一棵**隐藏**的转发树：
// 命令实现还是 file 包那一份（不复制逻辑），只在执行时打一行迁移提示。
//
// 为什么还留着：技能与旧脚本里仍有 `tt dict mirror path/pull` 的写法；
// 留一个版本的缓冲，下一版删除。

import (
	"tt/internal/cli/file"
)

func init() {
	alias := file.NewMirrorCmd()
	alias.Hidden = true
	// dict 组的 PersistentPreRunE 会打开查询数据源；镜像不依赖本地字典库，
	// 与从前一样覆盖掉它（同时补跑根命令的 --at 钩子）。
	alias.PersistentPreRunE = skipQuerySource
	for _, c := range alias.Commands() {
		c.Deprecated = "已迁移：请用 tt file mirror（例：tt file mirror path --at <环境>）"
	}
	Group.AddCommand(alias)
}
