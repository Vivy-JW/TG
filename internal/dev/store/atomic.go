package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// AtomicWrite 以「同目录临时文件 → Write → Sync → Close → Rename」覆盖目标文件。
//
// 红线 R6：工作区里任何落盘都不得非原子写、不得先删后建 —— 中途失败（磁盘满、
// 进程被杀）时**目标文件字节不变**，绝不允许出现「半个文件」或「原文件已丢」。
//
//   - 临时文件与目标**同目录**，保证 os.Rename 不跨卷（跨卷 Rename 会失败或退化）。
//   - 目标已存在时保留其原有权限位；不存在则用 0644。
//   - 任何失败路径都清理临时文件；Rename 失败时目标保持原样。
func AtomicWrite(path string, b []byte) error {
	dir := filepath.Dir(path)

	mode := fs.FileMode(fileMode)
	if st, err := os.Stat(path); err == nil {
		if st.IsDir() {
			return &IOError{Msg: "AtomicWrite 目标是一个目录，拒绝写入：" + path}
		}
		mode = st.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return ioErr("无法检查目标文件 "+path, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return ioErr("创建同目录临时文件失败（目录 "+dir+"）", err)
	}
	tmpName := tmp.Name()
	abort := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}

	if _, err := tmp.Write(b); err != nil {
		abort()
		return ioErr("写入临时文件失败 "+tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		abort()
		return ioErr("同步临时文件失败 "+tmpName, err)
	}
	if err := tmp.Chmod(mode); err != nil {
		abort()
		return ioErr("设置临时文件权限失败 "+tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return ioErr("关闭临时文件失败 "+tmpName, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return ioErr("原子替换目标文件失败 "+path, err)
	}
	return nil
}
