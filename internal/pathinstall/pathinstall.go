// 包 pathinstall:把可执行文件所在目录加入/移出**用户** PATH。
//
// 只动 HKCU\Environment,绝不碰 HKLM/系统 PATH,也不需要管理员。
// Web 设置页(/api/install)与 CLI(`tt install path`)共用这一份实现。
package pathinstall

import (
	"os"
	"path/filepath"
)

// Status 安装状态(Web 的 GET /api/install 与 CLI 的 `tt install path` 共用)。
type Status struct {
	Supported  bool   `json:"supported"`  // 本平台是否支持自动写入 PATH
	ExePath    string `json:"exePath"`    // 当前可执行文件
	ExeDir     string `json:"exeDir"`     // 将被加入 PATH 的目录
	InUserPath bool   `json:"inUserPath"` // 该目录是否已在用户 PATH 中
	UserPath   string `json:"userPath,omitempty"`
	Manual     string `json:"manual,omitempty"` // 不支持时的等价手动命令
	Note       string `json:"note,omitempty"`
}

// exePath 当前运行的可执行文件绝对路径(解析符号链接)。
func exePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(exe); err == nil {
		exe = r
	}
	return exe
}
