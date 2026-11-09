package update

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
	return out.Sync()
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

// PayloadRoot 找解包后的载荷根：便携包的 zip 里顶层是一个目录（tt-portable/），
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
func copyTree(src, dst string) (files int, err error) {
	return files, filepath.WalkDir(src, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		rel, rerr := filepath.Rel(src, path)
		if rerr != nil {
			return rerr
		}
		target := filepath.Join(dst, rel)
		if keepLocal[filepath.ToSlash(rel)] {
			return nil // 用户的数据：跟包内那份同名，但不是一回事
		}
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := copyFile(path, target); err != nil {
			return fmt.Errorf("覆盖 %s 失败: %w", target, err)
		}
		files++
		return nil
	})
}

func copyFile(src, dst string) error {
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
	return out.Sync()
}
