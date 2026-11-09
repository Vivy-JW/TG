// Package config 是 tt 的统一配置层：位置解析、读写、校验、打码。
//
// 外部资源（缓存子目录、服务状态、本地字典库、引擎 exe……）的落点也在这里统一解析，
// 见 locations.go 的 Locations —— 所有"东西放哪/去哪读"的问题都从这一层取答案，
// 调用方不得自己再拼一份。
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ---------- 配置文件位置 ----------
//
// 存放规则（第一个存在的胜出）：
//  1. TT_CONFIG 环境变量          显式指定
//  2. --config <路径>             显式指定
//  3. <exe 目录>\.portable 存在    便携包：配置留在包内
//  4. %APPDATA%\tt\config.json    默认：固定用户的统一位置
//
// 没有旧位置兜底，也不做自动迁移：旧版的各种落点（%APPDATA%\T100\tt、tdebug/tdict、
// exe 或当前目录的 config.json）一律不读 —— 数据统一在 %APPDATA%\tt。
//
// 显式指定的路径就是答案。--config / TT_CONFIG 指向一个还不存在的文件时，**不**悄悄
// 改用默认落点 —— 否则便携版会把配置写进用户目录，测试脚本也会落在别处。
//
// 数据目录 = 配置所在目录，所以缓存、快照、状态文件都跟着落在同一个目录下。

const (
	// ToolDirName 统一用户目录（%APPDATA%）下本工具的目录。数据目录 = 配置所在目录，
	// 所以 ents / srccache / .tt-serve.json 都跟着落在同一个目录里。
	ToolDirName = "tt"
	// DefaultConfigName 缺省配置文件名。
	DefaultConfigName = "config.json"
	// PortableMark 便携标记：exe 同目录存在该文件即视为便携包。
	PortableMark = ".portable"
)

// UserConfigDir 统一用户目录：%APPDATA%\tt（os.UserConfigDir() 在 Windows 上即
// %AppData%）。定位不到时返回空串。
func UserConfigDir() string {
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, ToolDirName)
	}
	return ""
}

// UserConfigPath 统一用户目录下的配置路径；定位不到时返回空串。
func UserConfigPath() string {
	if dir := UserConfigDir(); dir != "" {
		return filepath.Join(dir, DefaultConfigName)
	}
	return ""
}

// IsPortable 是否便携包（决定配置留在包内还是进统一用户目录）。
func IsPortable() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(filepath.Dir(exe), PortableMark))
	return err == nil
}

// exeDir 当前可执行文件所在目录；取不到返回空串。
func exeDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

// DefaultConfigPath 未经显式指定时的默认落点：
// 便携包内，否则统一用户目录。
func DefaultConfigPath() string {
	if IsPortable() {
		if dir := exeDir(); dir != "" {
			return filepath.Join(dir, DefaultConfigName)
		}
	}
	if p := UserConfigPath(); p != "" {
		return p
	}
	if abs, err := filepath.Abs(DefaultConfigName); err == nil {
		return abs
	}
	return DefaultConfigName
}

// DefaultConfigPathHint 给 --help 用的一行提示（取不到用户目录时退回文件名）。
func DefaultConfigPathHint() string {
	if IsPortable() {
		return "<exe 目录>\\" + DefaultConfigName
	}
	if dir := UserConfigDir(); dir != "" {
		return filepath.Join(dir, DefaultConfigName)
	}
	return DefaultConfigName
}

// ResolvePath 解析配置文件路径，优先级见包注释。
// allowMissing 为 true 时，若所有候选都不存在则返回默认落点而非报错
// （写配置的命令需要它：首次保存要能创建文件）。
func ResolvePath(flagPath string, allowMissing bool) (string, error) {
	var candidates []string
	explicit := false // 是否由用户显式指定（环境变量或 --config）

	// 1. 环境变量（最高优先级）
	if v := os.Getenv("TT_CONFIG"); v != "" {
		candidates = append(candidates, v)
		explicit = true
	}

	if flagPath != "" {
		// 2. --config 显式指定：原样 / 相对 exe 同目录 / 相对当前目录
		candidates = append(candidates, flagPath)
		explicit = true
		if !filepath.IsAbs(flagPath) {
			if dir := exeDir(); dir != "" {
				candidates = append(candidates, filepath.Join(dir, flagPath))
			}
			if cwd, err := os.Getwd(); err == nil {
				candidates = append(candidates, filepath.Join(cwd, flagPath))
			}
		}
	} else if !explicit {
		if IsPortable() {
			// 3. 便携包：配置留在包内，不进统一用户目录
			if dir := exeDir(); dir != "" {
				candidates = append(candidates, filepath.Join(dir, DefaultConfigName))
			}
		} else if p := UserConfigPath(); p != "" {
			// 4. 统一用户目录 %APPDATA%\tt
			candidates = append(candidates, p)
		}
	}

	var tried []string
	for _, p := range candidates {
		abs, _ := filepath.Abs(p)
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		}
		tried = append(tried, abs)
	}

	if allowMissing {
		// 写路径。用户显式指定了路径时，那就是答案 —— 文件还不存在正是要新建它，
		// 不能悄悄改用默认落点：那会让便携版把配置写到用户目录里去，
		// 也会让 --config <临时目录> 的调用（测试与脚本）落在别处。
		if explicit && len(candidates) > 0 {
			if abs, err := filepath.Abs(candidates[0]); err == nil {
				return abs, nil
			}
			return candidates[0], nil
		}
		if p := DefaultConfigPath(); p != "" {
			return p, nil
		}
	}

	return "", fmt.Errorf(
		"配置文件未找到。\n\n尝试了以下路径:\n%s\n\n缺省位置: %s\n设置 TT_CONFIG 环境变量或使用 --config 指定正确路径:\n  setx TT_CONFIG \"D:\\path\\to\\config.json\"\n  tt --config \"D:\\path\\to\\config.json\" config path",
		FormatTriedPaths(tried), DefaultConfigPath(),
	)
}

// FormatTriedPaths 把尝试过的路径渲染成 --help/报错里的清单。
func FormatTriedPaths(paths []string) string {
	var s string
	for _, p := range paths {
		exists := ""
		if _, err := os.Stat(p); err == nil {
			exists = " (found)"
		}
		s += fmt.Sprintf("  - %s%s\n", p, exists)
	}
	return s
}

// SamePath 两个路径是否指向同一个文件（比较绝对路径，失败则退回字面比较）。
func SamePath(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return a == b
	}
	return aa == bb
}
