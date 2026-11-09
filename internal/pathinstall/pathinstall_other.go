//go:build !windows

package pathinstall

import (
	"fmt"
	"path/filepath"
)

const pathSep = ":"

// Get 非 Windows 不自动改 PATH,给出等价的手动命令。
func Get() Status {
	exe := exePath()
	dir := filepath.Dir(exe)
	return Status{
		Supported: false,
		ExePath:   exe,
		ExeDir:    dir,
		Manual:    fmt.Sprintf("export PATH=\"$PATH:%s\"   # 追加到 ~/.bashrc 或 ~/.zshrc 后 source", dir),
		Note:      "当前系统未实现自动写入 PATH,请按上述命令手动加入(用户级,无需 root)。",
	}
}

func Preview() (Status, string, bool, error) {
	return Get(), "", false, fmt.Errorf("当前系统不支持自动写入 PATH,请按提示手动添加")
}

func Add() (Status, error) {
	return Get(), fmt.Errorf("当前系统不支持自动写入 PATH,请按提示手动添加")
}

func Remove() (Status, error) {
	return Get(), fmt.Errorf("当前系统不支持自动写入 PATH")
}
