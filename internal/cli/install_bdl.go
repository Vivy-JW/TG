package cli

// `tg install bdl` 的入口：Genero BDL 官方手册。
//
// 引擎（定 zip / 定目录 / 跳过与 --force / 解包 / 报告）都在 install_docs.go，
// 与 `tg install gpdoc` 共用一套；这里只留 bdl 自己的名字，免得测试与旧调用点全改。

import "github.com/spf13/cobra"

// bdlZipName 随包分发的 BDL 手册 zip 文件名（与 config.DocBDL.ZipName 同值）。
const bdlZipName = "BDL-Markdown.zip"

// bdlReport 一次 bdl 安装的结果（docReport 的旧名字）。
type bdlReport = docReport

// bdlSpec docBDL 的副本，zip 来源换成可替换的 bdlZipSource（测试用）。
func bdlSpec() docSpec {
	s := docBDL
	s.ZipSource = bdlZipSource
	return s
}

// newInstallBDLCmd 造 `tg install bdl`。
func newInstallBDLCmd() *cobra.Command {
	return newInstallDocCmd(docBDL)
}

// resolveBDLZip 定 BDL 文档包：--from 优先，否则 exe 旁边的随包 zip。
func resolveBDLZip(from string) (string, error) {
	return resolveDocZip(from, bdlSpec())
}

// resolveBDLDir 定落点：--to > config 的 bdldoc.dir > <数据目录>/BDL-Markdown。
func resolveBDLDir(to string) (dir, why string, err error) {
	return resolveDocDir(to, docBDL)
}

// extractBDL 解 BDL 文档包到 dir，返回写出的文件数。
func extractBDL(zipPath, dir string) (files int, err error) {
	return extractDoc(zipPath, dir, docBDL)
}

// printBDLReport 打印一次 bdl 安装的结果。
func printBDLReport(r bdlReport) {
	printDocReport(docBDL, r)
}
