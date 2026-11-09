package config

import (
	"path/filepath"
	"strings"
)

// MirrorPath 返回某环境的源码镜像目录：
//   - 环境自己配了 mirrorDir → 用它（绝对化）；
//   - 否则用顶层 mirror.dir/<环境名>；
//   - 两处都没有 → 空串（调用方负责报“未设置”）。
//
// 这里只做路径解析，不碰文件系统；目录是否存在由 host.MirrorReadyDir 判断。
func (e *NamedSsh) MirrorPath(mirrorRoot string) string {
	if e == nil {
		return ""
	}
	if name := strings.TrimSpace(e.Name); name == "" {
		return ""
	}
	if d := strings.TrimSpace(e.MirrorDir); d != "" {
		return AbsPath(d)
	}
	if root := strings.TrimSpace(mirrorRoot); root != "" {
		return filepath.Join(AbsPath(root), e.Name)
	}
	return ""
}
