package update

import "tg/internal/zipx"

// keepLocal 是**载荷里可能有、但绝不能被覆盖的条目**：它们是用户的数据，恰好与随包
// 分发的那一份同名。
//
// 今天只有 config.json，而它恰好是最贵的一个：便携形态的配置就住在包内（那是便携契约），
// 而 zip 里带的是 config.empty.json 拷来的**空模板** —— 覆盖它等于删掉用户的口令、
// 环境与库连接。这条不是理论风险：真要“整个目录换掉”，第一个受害的就是它。
var keepLocal = map[string]bool{
	"config.json": true,
}

// copyTree 把 src 树里的文件覆盖到 dst（就地覆盖）：已存在的文件被替换，**src 里没有的
// 文件保持原样**，keepLocal 里的绝不碰。
//
// 为什么不是"整个目录换掉"：便携形态的 config.json 就在包内（那是便携契约），而 zip 里
// 带的是空的模板 —— 整目录替换会删掉用户的口令与环境。代价是上一版删掉/改名的文件会
// 留在原地（与"解压覆盖"的语义一致，README 里写明的就是这条路）。
//
// 复制本身走 zipx.CopyTree（与 `tg install bdl` 同一份实现）；这里只挂"保护本地文件"
// 这条策略 —— 策略留在 update 里，是因为"哪些算用户的数据"是升级语义，不是解包语义。
func copyTree(src, dst string) (int, error) {
	return zipx.CopyTree(src, dst, func(rel string) bool { return keepLocal[rel] })
}
