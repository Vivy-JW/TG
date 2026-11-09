// 把便携版暂存目录打成 zip。
//
// 为什么单独成一个工具：原先是在 build_portable.bat 里塞一句 python -c，用的是
// os.listdir + z.write —— 那个组合**不会递归**，目录只会写进一个空条目：
// skills/ 下明明有 SKILL.md，打出来的包里却只有一个空的 skills/ 目录，
// 便携包于是缺了技能文档（而它正是给 AI 用的说明书）。
// 前身是 Python 脚本 tools/zip.py，2026-10 随「tools 去 Python」改成 Go。
//
// 两条纪律：
//   - 必须递归（理由如上）；
//   - 路径分隔符固定用 /（zip 规范），不依赖平台 —— Windows PowerShell 5.1 的
//     Compress-Archive 写反斜杠条目名，跨平台解压工具会读出错误目录，这是它
//     只配当回退、不配当正式实现的原因。
//
// 用法：go run ./tools/zip [暂存目录 [输出 zip]]
// 缺省 dist/tt-portable -> dist/tt-portable.zip（与 build_portable.bat 的调用一致）。
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	stage, out := "dist/tt-portable", "dist/tt-portable.zip"
	if len(os.Args) > 1 {
		stage = os.Args[1]
	}
	if len(os.Args) > 2 {
		out = os.Args[2]
	}

	if st, err := os.Stat(stage); err != nil || !st.IsDir() {
		return fmt.Errorf("暂存目录不存在: %s", stage)
	}

	// zip 里的顶层目录名 = 暂存目录名（解开就是 tt-portable/ 一个顶层目录，
	// 与旧回退 shutil.make_archive 的 includeBaseDirectory 语义一致）。
	root := filepath.Base(filepath.Clean(stage))

	zf, err := os.Create(out)
	if err != nil {
		return err
	}
	defer zf.Close()

	w := zip.NewWriter(zf)
	defer w.Close()

	n := 0
	err = filepath.WalkDir(stage, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(stage, p)
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		hdr.Name = root + "/" + filepath.ToSlash(rel)
		hdr.Method = zip.Deflate
		dst, err := w.CreateHeader(hdr)
		if err != nil {
			return err
		}
		src, err := os.Open(p)
		if err != nil {
			return err
		}
		_, err = io.Copy(dst, src)
		cerr := src.Close()
		if err != nil {
			return err
		}
		if cerr != nil {
			return cerr
		}
		n++
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("已打包 %d 个文件 -> %s\n", n, out)
	return nil
}
