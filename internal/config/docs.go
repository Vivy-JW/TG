package config

// 随包文档（Markdown 版第三方手册）的通用落点规则。
//
// 内容不归 tg 管（几十 MB、几千个文件的第三方文档，由 `tg install bdl` /
// `tg install gpdoc` 从随包的 zip 解出来）；tg 管两件事就够了：装到哪儿、装没装。
// 落点规则与 mirror/dict 同一条 —— **配置住在哪，数据就住在哪**：便携形态跟着包走，
// 普通形态在 %APPDATA%\tg，用户改过就按改的来。
//
// 每类文档一条 DocKind：配置节名、缺省子目录名、随包 zip 名、人读标题。新增一类文档
// 只加一条 DocKind + 一个配置节，CLI/设置页/打包脚本共用同一套规则。

import (
	"os"
	"path/filepath"
	"strings"
)

// DocKind 描述一类随包文档。
type DocKind struct {
	// Key 配置里的节名（也是设置页/JSON 的键）："bdldoc" / "gpdoc"。
	Key string
	// DirName 缺省落点在数据目录下的子目录名。
	DirName string
	// ZipName 随包分发的压缩包名（与 tg.exe 同目录）。
	ZipName string
	// Title 人读标题（CLI 输出与设置页卡片共用）。
	Title string
}

var (
	// DocBDL Genero BDL 官方手册（User Guide 6.00，19 章 / 约 5,450 个 md）。
	DocBDL = DocKind{
		Key:     "bdldoc",
		DirName: "BDL-Markdown",
		ZipName: "BDL-Markdown.zip",
		Title:   "Genero BDL 手册",
	}
	// DocGP TOP GP 产品手册（13 章 / 约 240 个 md，另带 llms.txt 与 README 索引）。
	DocGP = DocKind{
		Key:     "gpdoc",
		DirName: "TOP-GP-Markdown",
		ZipName: "TOP-GP-Markdown.zip",
		Title:   "TOP GP 手册",
	}
)

// DirFor 文档根目录（读配置）：顶层 <key>.dir 优先，否则 <数据目录>/<DirName>。
//
// 配置读不到 / 写坏时安静回退缺省落点 —— 与 DictDBDirFor 同一条理由：文档根不该因为
// 配置有问题就变成空串，那会让安装命令相对当前目录落盘，装到哪儿全看用户当时在哪个
// 文件夹里。
func (k DocKind) DirFor(cfgPath string) string {
	if strings.TrimSpace(cfgPath) != "" {
		if r, err := Load(cfgPath); err == nil {
			if d := strings.TrimSpace(r.DocDir(k.Key)); d != "" {
				return AbsPath(d)
			}
		}
	}
	return filepath.Join(LocationsAt(cfgPath).DataDir(), k.DirName)
}

// DocStatus 文档落点的状态：生效目录 + 装没装（设置页那张卡片要的全部信息）。
type DocStatus struct {
	// Dir 生效目录（绝对路径；配置的或缺省的）。
	Dir string `json:"dir"`
	// Configured 配置里显式写下的值；空 = 用缺省落点。
	Configured string `json:"configured,omitempty"`
	// Exists 该目录是否存在于本机。
	Exists bool `json:"exists"`
	// Installed 目录里是否已经有东西。只回答"装过没"，不校验完整性 —— 完整性
	// 要遍历几千个文件，而这是一个每次进设置页都会读的状态。
	Installed bool `json:"installed"`
}

// StatusFor 汇总落点状态（设置页与 CLI 共用的那一份判断）。
func (k DocKind) StatusFor(cfgPath string) DocStatus {
	st := DocStatus{Dir: k.DirFor(cfgPath)}
	if strings.TrimSpace(cfgPath) != "" {
		if r, err := Load(cfgPath); err == nil {
			st.Configured = AbsPath(r.DocDir(k.Key))
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
