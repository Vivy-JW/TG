// 表单包的**纯解压**：把 zip 原样摊到目录里，不做任何解释、渲染或校验。
//
// 为什么单独放一个函数、而不是复用 pkgfile.Open：
//   - Open 是给 .tzc 代码包用的：它按扩展名判 kind、检查必需条目（.tsd/.4fd…）、
//     解析 ver、还要求 TAP/TGL —— 那些语义对「就想看看表单包里有什么」是多余的，
//     甚至会把一个缺条目的包直接挡在门外；
//   - 用户要的语义是「只是解压缩，不做任何特殊处理」，所以这里只做三件事：
//     逐条目解压、按相对路径落盘、拒绝越界路径（zip-slip）。
//
// 源头包永远只读（本函数不写回、也不产生任何工作区/审计产物）。
package pkgfile

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tt/internal/dev/model"
)

// ExtractedFile 是解压出来的一个文件。
type ExtractedFile struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	Sha256 string `json:"sha256"`
}

// UnzipTo 把 zip 包 src 纯解压到 dst。
//
// force=false 时：dst 已存在且非空 → 报错（避免和上一次解压/别人的文件混在一起）；
// force=true 时：覆盖同名文件（用截断写，不做「先删后建」，对齐红线 R6）。
//
// 返回按名字排序的条目清单（不含目录条目），供报告使用。
func UnzipTo(src, dst string, force bool) ([]ExtractedFile, error) {
	b, err := os.ReadFile(src)
	if err != nil {
		return nil, &IOError{Msg: "读不到包文件 " + src, Err: err}
	}
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return nil, &FormatError{Msg: "不是有效的 zip 包：" + src, Detail: []string{err.Error()}}
	}

	absDst, err := filepath.Abs(dst)
	if err != nil {
		return nil, &IOError{Msg: "解析目标目录失败 " + dst, Err: err}
	}
	if st, serr := os.Stat(absDst); serr == nil {
		if !st.IsDir() {
			return nil, &IOError{Msg: "目标已存在且不是目录：" + absDst}
		}
		if ents, rerr := os.ReadDir(absDst); rerr == nil && len(ents) > 0 && !force {
			return nil, &IOError{
				Msg: fmt.Sprintf("目标目录非空（%d 项），拒绝直接解压进去：%s", len(ents), absDst),
				Err: fmt.Errorf("换个空目录（-o）或加 --force 覆盖同名文件"),
			}
		}
	} else if err := os.MkdirAll(absDst, 0o755); err != nil {
		return nil, &IOError{Msg: "建目标目录失败 " + absDst, Err: err}
	}

	var out []ExtractedFile
	for _, zf := range zr.File {
		name := zf.Name
		if zf.FileInfo().IsDir() || strings.HasSuffix(name, "/") || strings.HasSuffix(name, `\`) {
			continue // 目录条目：由文件的父目录按需创建
		}
		target, jerr := safeJoin(absDst, name)
		if jerr != nil {
			return nil, &FormatError{Msg: "包里的条目名不安全，拒绝解压", Detail: []string{name, jerr.Error()}}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return nil, &IOError{Msg: "建目录失败 " + filepath.Dir(target), Err: err}
		}
		rc, oerr := zf.Open()
		if oerr != nil {
			return nil, &FormatError{Msg: "读不到条目 " + name, Detail: []string{oerr.Error()}}
		}
		data, rerr := io.ReadAll(rc)
		rc.Close()
		if rerr != nil {
			return nil, &FormatError{Msg: "解压条目失败 " + name, Detail: []string{rerr.Error()}}
		}
		// 纯解压 = 逐字节落盘；不进 store.AtomicWrite（那会留下 .tmp 命名约定），
		// 但也不用「先删后建」：O_TRUNC 覆盖即可（红线 R6 的意图）。
		f, ferr := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if ferr != nil {
			return nil, &IOError{Msg: "写文件失败 " + target, Err: ferr}
		}
		if _, werr := f.Write(data); werr != nil {
			f.Close()
			return nil, &IOError{Msg: "写文件失败 " + target, Err: werr}
		}
		if cerr := f.Close(); cerr != nil {
			return nil, &IOError{Msg: "收尾写文件失败 " + target, Err: cerr}
		}
		out = append(out, ExtractedFile{
			Name:   filepath.ToSlash(name),
			Size:   len(data),
			Sha256: model.Sha256Bytes(data),
		})
	}
	if len(out) == 0 {
		return nil, &FormatError{Msg: "zip 里没有任何文件条目：" + src}
	}
	sortExtracted(out)
	return out, nil
}

// sortExtracted 按条目名排序（稳定输出，便于比对与 JSON diff）。
func sortExtracted(fs []ExtractedFile) {
	sort.Slice(fs, func(i, j int) bool { return fs[i].Name < fs[j].Name })
}

// safeJoin 把 zip 条目名安全地拼到目标目录下：拒绝绝对路径、盘符、上跳与 NUL。
// （zip-slip：恶意/损坏的包用 ../ 就能写到目标目录之外。）
func safeJoin(dst, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("空条目名")
	}
	if strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("条目名里有 NUL")
	}
	clean := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("绝对路径")
	}
	if filepath.VolumeName(clean) != "" || (len(clean) >= 2 && clean[1] == ':') {
		return "", fmt.Errorf("带盘符")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".." {
			return "", fmt.Errorf("含上跳路径段 ..")
		}
	}
	target := filepath.Join(dst, filepath.FromSlash(clean))
	rel, err := filepath.Rel(dst, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("拼出来的路径跑到目标目录之外")
	}
	return target, nil
}
