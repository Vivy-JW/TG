package cli

// `tg install gpdoc` 的入口：TOP GP 产品手册（Markdown 版）。
//
// 与 `tg install bdl` 共用 install_docs.go 的引擎，差别只有：落点 config 节是
// gpdoc.dir、缺省目录 <数据目录>/TOP-GP-Markdown、随包 zip 是 TOP-GP-Markdown.zip。

import "github.com/spf13/cobra"

// gpDocZipSource 解析随包 zip；测试可替换（与 bdlZipSource 同构）。
var gpDocZipSource = docGP.ZipSource

// gpSpec docGP 的副本，zip 来源换成可替换的 gpDocZipSource（测试用）。
func gpSpec() docSpec {
	s := docGP
	s.ZipSource = gpDocZipSource
	return s
}

// newInstallGPDocCmd 造 `tg install gpdoc`。
func newInstallGPDocCmd() *cobra.Command {
	return newInstallDocCmd(docGP)
}

// resolveGPDocZip 定文档包：--from 优先，否则 exe 旁边的随包 zip。
func resolveGPDocZip(from string) (string, error) {
	return resolveDocZip(from, gpSpec())
}

// resolveGPDocDir 定落点：--to > config 的 gpdoc.dir > <数据目录>/TOP-GP-Markdown。
func resolveGPDocDir(to string) (dir, why string, err error) {
	return resolveDocDir(to, gpSpec())
}
