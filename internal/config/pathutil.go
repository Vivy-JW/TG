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

// TzsStatus 是 .tzs 引擎那几样东西的派生状态(见 schema.go 的 TzsSettings)。
//
// Designer 与 Engine 都是随包分发的,**它们不在说明分包或部署坏了**;Workspace 在用户自己的
// 机器上、由用户配置,它不在是配置问题。混成一个字段会让设置页把"你自己没配"说成"程序坏了"。
type TzsStatus struct {
	// Engine 引擎 exe:生效值可以是配置里覆盖的 serverExe,否则是 <tt.exe 目录>\tzs\ 下那个。
	Engine FileStatus `json:"engine"`
	// Designer 随包分发的设计器程序集目录(<引擎 exe 目录>\designer),引擎默认从这儿加载。
	Designer DirStatus `json:"designer"`
	// Workspace 引擎 Boot 的工作区(含 mta/ 的目录)。**按环境配**（见 NamedSsh.Workspace），
	// 这里拿到的是**当前环境**那一份。
	Workspace DirStatus `json:"workspace"`
}

// TzsStatusOf 汇总引擎那几样东西的状态。activeWorkspace 是当前环境的工作区
// （config.Root.ActiveWorkspace —— 只此一份取法），defaultEngineExe 是缺省引擎 exe
// （用 config.EngineExe("") 取 —— 候选只有那一份实现），参数注入是为了可测。
//
// Designer 的位置从**生效的**引擎 exe 推出来,而不是另配一个:引擎自己就是这么算的
// (D.Bootstrap 的 Install = <自己的目录>\designer),两边各算一次的话,设置页会报"有"
// 而引擎报"没有"。
func TzsStatusOf(s TzsSettings, activeWorkspace, defaultEngineExe string) TzsStatus {
	engine := FileStatusOf(s.ServerExe, defaultEngineExe)
	designerDir := ""
	if engine.Target != "" {
		designerDir = filepath.Join(filepath.Dir(engine.Target), "designer")
	}
	return TzsStatus{
		Engine:    engine,
		Designer:  DirStatusOf(designerDir),
		Workspace: DirStatusOf(activeWorkspace),
	}
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
