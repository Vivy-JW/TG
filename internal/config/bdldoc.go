package config

// Genero BDL 官方手册（Markdown 版）的落点。
//
// 内容不归 tg 管（16 MB / 5470 个文件的第三方文档，由 `tg install bdl` 从随包的
// BDL-Markdown.zip 解出来）；tg 管两件事就够了：装到哪儿、装没装。落点规则与
// mirror/dict 同一条 —— **配置住在哪，数据就住在哪**：便携形态跟着包走，普通形态在
// %APPDATA%\tg，用户改过就按改的来。

import (
	"os"
	"path/filepath"
	"strings"
)

// BDLDocDirName 是缺省落点在数据目录下的子目录名。
const BDLDocDirName = "BDL-Markdown"

// BDLDocDirFor 文档根目录（读配置）：顶层 bdldoc.dir 优先，否则 <数据目录>/BDL-Markdown。
//
// 配置读不到 / 写坏时安静回退缺省落点 —— 与 DictDBDirFor 同一条理由：文档根不该因为
// 配置有问题就变成空串，那会让 `tg install bdl` 相对当前目录落盘，装到哪儿全看用户
// 当时在哪个文件夹里。
func BDLDocDirFor(cfgPath string) string {
	if strings.TrimSpace(cfgPath) != "" {
		if r, err := Load(cfgPath); err == nil {
			if d := strings.TrimSpace(r.BDLDoc.Dir); d != "" {
				return AbsPath(d)
			}
		}
	}
	return filepath.Join(LocationsAt(cfgPath).DataDir(), BDLDocDirName)
}

// BDLDocStatus 手册落点的状态：生效目录 + 装没装（设置页那张家卡片要的全部信息）。
type BDLDocStatus struct {
	// Dir 生效目录（绝对路径；配置的或缺省的）。
	Dir string `json:"dir"`
	// Configured 配置里显式写下的值；空 = 用缺省落点。
	Configured string `json:"configured,omitempty"`
	// Exists 该目录是否存在于本机。
	Exists bool `json:"exists"`
	// Installed 目录里是否已经有东西。只回答"装过没"，不校验完整性 —— 完整性
	// 要遍历 5470 个文件，而这是一个每次进设置页都会读的状态。
	Installed bool `json:"installed"`
}

// BDLDocStatusFor 汇总手册落点状态（设置页与 CLI 共用的那一份判断）。
func BDLDocStatusFor(cfgPath string) BDLDocStatus {
	st := BDLDocStatus{Dir: BDLDocDirFor(cfgPath)}
	if strings.TrimSpace(cfgPath) != "" {
		if r, err := Load(cfgPath); err == nil {
			st.Configured = AbsPath(r.BDLDoc.Dir)
		}
	}
	if fi, err := os.Stat(st.Dir); err == nil && fi.IsDir() {
		st.Exists = true
		if ents, rerr := os.ReadDir(st.Dir); rerr == nil && len(ents) > 0 {
			st.Installed = true
		}
	}
	return st
}
