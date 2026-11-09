package skill

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Source 是一条**要装出去的技能**：官方目录里的一个，或用户登记且判定通过的一个。
type Source struct {
	Origin Origin
	Dir    string // 源目录（绝对路径）
	Name   string // 技能名 = SKILL.md 的 name = 源目录名
}

// Warning 是一条「这条技能没装成，但整次安装照常继续」的记录。
//
// 第三方的三条失败路径都落在这里：路径没了、内容不合规、跟别人撞名。用户明确要的是
// 「只警告，不影响其它技能安装」—— 因为第三方路径可能指向一个今天没插的移动硬盘。
type Warning struct {
	Kind  State  `json:"kind"`            // missing / invalid / conflict
	Index int    `json:"index,omitempty"` // 第三方登记的序号（1 起）
	Path  string `json:"path,omitempty"`
	Name  string `json:"name,omitempty"`
	Why   string `json:"why"`
}

// Text 是给人看的一行，含修复动作。
//
// 命令行与设置页共用这一句：同一个问题两处各写一段文案，改一处忘一处之后，
// 用户看到的两个入口对同一件事给出不同说法。
func (w Warning) Text() string {
	var b strings.Builder
	if w.Index > 0 {
		fmt.Fprintf(&b, "第 %d 条 ", w.Index)
	}
	if w.Path != "" {
		b.WriteString(w.Path)
		if w.Name != "" && w.Name != filepath.Base(w.Path) {
			fmt.Fprintf(&b, "（%s）", w.Name)
		}
	} else if w.Name != "" {
		b.WriteString(w.Name)
	}
	fmt.Fprintf(&b, "：%s", w.Why)
	b.WriteString("  —— 挪过位置就改 skills.externals 里的路径，删掉了就把这一条去掉")
	return b.String()
}

// Plan 是一次安装的完整计划：要装哪些技能、官方树的顶层文件、以及所有告警。
//
// 计划与执行分开是为了 --dry-run：它只跑 BuildPlan 并把计划打印出来，一个字节都不写。
type Plan struct {
	Skills     []Source
	ExtraFiles []string  // 官方树顶层的非目录条目（README.md 等），原样复制
	Warnings   []Warning // 第三方里没通过判定的那些，按登记顺序
}

// BuildPlan 组出安装计划。官方树不合规 → 返回错误（包坏了，不是用户配错了）；
// 第三方的任何问题都降级成 Warnings，不影响官方与其它第三方。
//
// externals 是配置里 skills.externals 的顺序，顺序即优先级（同名时先登记者赢）。
func BuildPlan(officialDir string, externals []string) (Plan, error) {
	officialDir = absOr(officialDir)
	official, extraFiles, err := InspectOfficial(officialDir)
	if err != nil {
		return Plan{}, err
	}

	names := make([]string, 0, len(official))
	for _, s := range official {
		names = append(names, s.Name)
	}

	p := Plan{ExtraFiles: extraFiles}
	p.Skills = make([]Source, 0, len(official)+len(externals))
	for _, s := range official {
		p.Skills = append(p.Skills, Source{Origin: s.Origin, Dir: s.Dir, Name: s.Name})
	}
	for _, info := range InspectExternal(externals, names) {
		if info.State == StateOK {
			p.Skills = append(p.Skills, Source{Origin: info.Origin, Dir: info.Dir, Name: info.Name})
			continue
		}
		p.Warnings = append(p.Warnings, Warning{
			Kind: info.State, Index: info.Index, Path: info.Dir, Name: info.Name, Why: info.Why,
		})
	}
	// 顶层文件按名字排一下：ReadDir 已经排过，但计划是给人看的，别指望上游。
	sort.Strings(p.ExtraFiles)
	return p, nil
}

func absOr(p string) string {
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}
