// 配置里那些"路径型"取值的公共解析。
//
// 合并前这套逻辑散在 internal/dict/server 的 config.go / dbsync.go 里
// (absPath + 各处自己 os.Stat),而设置页搬进调试工作台之后也需要同一份判断 ——
// 两边各算一遍的话,同一个 sync.target 会解析成不同结果,那是最难查的一类 bug。
// 所以收敛到这里,谁要用谁来取。
package config

import (
	"os"
	"path/filepath"
	"strings"
)

// AbsPath 把配置里的路径值转成绝对路径;空串原样返回空串(表示"未配置")。
func AbsPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// DirStatus 一个"目录型"配置值的状态:配置里的值 + 它在本机是否真的存在。
type DirStatus struct {
	// Dir 绝对路径;空 = 未配置。
	Dir string `json:"dir"`
	// Exists 该目录是否存在于本机。
	Exists bool `json:"exists"`
}

// DirStatusOf 解析一个目录型配置值并检查它是否存在。
func DirStatusOf(raw string) DirStatus {
	st := DirStatus{Dir: AbsPath(raw)}
	if st.Dir == "" {
		return st
	}
	if fi, err := os.Stat(st.Dir); err == nil && fi.IsDir() {
		st.Exists = true
	}
	return st
}

// FileStatus 一个"文件型"配置值的状态。
type FileStatus struct {
	// Target 当前生效值(绝对路径)。
	Target string `json:"target"`
	// Configured 配置里显式写下的值;空 = 用缺省位置。
	Configured string `json:"configured"`
	// DefaultTarget 缺省位置(未配置时生效的那个)。
	DefaultTarget string `json:"defaultTarget"`
	// Exists 生效值指向的文件当前是否存在。
	Exists bool `json:"exists"`
}

// FileStatusOf 汇总一个文件型配置值:生效值、显式配置值、缺省值、是否存在。
func FileStatusOf(configured, defaultTarget string) FileStatus {
	st := FileStatus{
		Configured:    AbsPath(configured),
		DefaultTarget: defaultTarget,
	}
	st.Target = st.Configured
	if st.Target == "" {
		st.Target = defaultTarget
	}
	if st.Target != "" {
		if _, err := os.Stat(st.Target); err == nil {
			st.Exists = true
		}
	}
	return st
}

// SyncTargetFor 数据同步(写本地 SQLite)的配置值 = 配置里 sync.target 的绝对路径;
// 未配置返回空串(落点交给 Locations.SyncTarget 的缺省规则)。
//
// 存在的理由:同一条配置,CLI 与 web 曾经各解析各的 —— 设置页改了 sync.target 之后
// `tt dict db sync` 仍然写旧位置,两个本地库悄悄分叉,而"tt dict db status"看到的
// 又是另一个。目标路径只该有一个答案。
func SyncTargetFor(root *Root) string {
	if root == nil {
		return ""
	}
	return AbsPath(root.Sync.Target)
}
