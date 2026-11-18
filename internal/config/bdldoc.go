package config

// Genero BDL 官方手册的落点 —— 规则本体在 docs.go（DocKind），这里只留对外名字。
//
// 早期只有这一种文档，函数名带 BDL 前缀且被 web/cli/测试引用；现在文档有两类
// （bdldoc / gpdoc），通用规则搬到 DocKind 上，这些名字作为稳定别名保留。

// BDLDocDirName 是缺省落点在数据目录下的子目录名。
const BDLDocDirName = "BDL-Markdown"

// BDLDocDirFor 文档根目录（读配置）：顶层 bdldoc.dir 优先，否则 <数据目录>/BDL-Markdown。
func BDLDocDirFor(cfgPath string) string { return DocBDL.DirFor(cfgPath) }

// BDLDocStatus 手册落点的状态（DocStatus 的别名，保留旧类型名）。
type BDLDocStatus = DocStatus

// BDLDocStatusFor 汇总手册落点状态。
func BDLDocStatusFor(cfgPath string) BDLDocStatus { return DocBDL.StatusFor(cfgPath) }
