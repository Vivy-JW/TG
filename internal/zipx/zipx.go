// Package zipx 是解包的唯一实现：zip-slip 防护、反斜杠条目、载荷根识别、树复制。
//
// 为什么单独一个包：解包有两处真实调用方 —— 自助升级（internal/update 解开便携包）
// 与 `tg install bdl`（解开随包的 BDL 文档）。第两个调用方当初只是"顺手 import 一下
// update"，等于让"装文档"这件事依赖自助升级那条竖线；这套校验又最不该被复制第二份
// （复制出来的第二份迟早少一条 ..\ 检查，而它解出来的东西要落到用户的程序目录/数据目录）。
package zipx

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip 把 zip 解到 dest（dest 必须已经存在且为空目录由调用方保证）。
//
// 两处必须自己把关，不能靠 zip 内容自觉：
//   - **路径逃逸**（zip-slip）：条目名里带 ..\ 或绝对路径就拒绝。解出来的是要覆盖
//     用户程序目录的东西，一个越界条目就能往别处写文件。
//   - 条目名里的反斜杠：Windows 打包工具会写 `tzs\designer\x.dll`，直接当路径分隔符用。
func ExtractZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("打开 %s 失败: %w", zipPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("建目录 %s 失败: %w", target, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("建目录 %s 失败: %w", filepath.Dir(target), err)
		}
		if err := writeZipEntry(f, target); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, target string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("读 %s 失败: %w", f.Name, err)
	}
	defer rc.Close()
	// 权限位：zip 里的可执行位在 Windows 上无意义，统一 0644 即可（exe 靠扩展名）。
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("建 %s 失败: %w", target, err)
	}
	defer out.Close()
	if _, err := io.Copy(out, rc); err != nil {
		return fmt.Errorf("写 %s 失败: %w", target, err)
	}
	// 不 fsync：这两条线解的都是**可再生**的内容（升级包重下一次、手册重解一次），
	// 而 5470 个小文件各一次 FlushFileBuffers 要几十秒 —— 用 35 秒换"断电时可能
	// 少半页手册"不划算。Close 已经把数据交给系统缓存。
	return nil
}

// safeJoin 把 zip 里的条目名拼到 base 下，并保证结果不跑到 base 外面。
func safeJoin(base, name string) (string, error) {
	clean := strings.ReplaceAll(name, `\`, "/")
	if strings.HasPrefix(clean, "/") || filepath.IsAbs(clean) {
		return "", fmt.Errorf("压缩包里的条目 %q 是绝对路径，拒绝解包", name)
	}
	p := filepath.Join(base, filepath.FromSlash(clean))
	rel, err := filepath.Rel(base, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("压缩包里的条目 %q 逃出了目标目录，拒绝解包", name)
	}
	return p, nil
}

// PayloadRoot 找解包后的载荷根：便携包的 zip 里顶层是一个目录（tg-portable/），
// 但别人重打的包也可能是平的，所以两种都认。
func PayloadRoot(staging string) (string, error) {
	ents, err := os.ReadDir(staging)
	if err != nil {
		return "", fmt.Errorf("读 %s 失败: %w", staging, err)
	}
	if len(ents) == 1 && ents[0].IsDir() {
		return filepath.Join(staging, ents[0].Name()), nil
	}
	if len(ents) == 0 {
		return "", fmt.Errorf("解出来的 %s 是空的", staging)
	}
	return staging, nil
}

// CopyTree 把 src 树里的文件覆盖到 dst（就地覆盖）：已存在的文件被替换，src 里没有的
// 文件保持原样。
//
// skip 拿到的是**相对 src 的斜杠路径**，返回 true 表示这一条不许动（调用方用它保护
// 用户自己的文件，例如便携包里那份 config.json）；可以为 nil。
func CopyTree(src, dst string, skip func(rel string) bool) (files int, err error) {
	return files, filepath.WalkDir(src, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, rerr := filepath.Rel(src, path)
		if rerr != nil {
			return rerr
		}
		if skip != nil && skip(filepath.ToSlash(rel)) {
			return nil // 调用方点名要保护的东西
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := CopyFile(path, target); err != nil {
			return fmt.Errorf("覆盖 %s 失败: %w", target, err)
		}
		files++
		return nil
	})
}

// CopyFile 复制单个文件，必要时建父目录。
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	// 同 writeZipEntry：不 fsync（理由写在那一处）。
	return nil
}

// CountFiles 数一棵树里的文件个数（目录不算）。给"已经解好了"这类不需要再复制的
// 落盘路径报数用 —— 例如整个目录 rename 到位之后。
func CountFiles(root string) (int, error) {
	n := 0
	err := filepath.WalkDir(root, func(_ string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n, err
}
