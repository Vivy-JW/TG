package skill

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Origin 说明一个技能是从哪儿来的。两套来源彻底分开：
//
//   - 官方：<exe 目录>\skills。它是发版包的一部分，走专用路径，不参与第三方登记表的
//     校验与告警 —— 它坏了是包坏了，不是用户配错了。
//   - 第三方：用户在设置页登记的任意路径，一行一个技能目录。
type Origin string

const (
	OriginOfficial Origin = "official"
	OriginExternal Origin = "external"
)

// State 是一条登记（或一个技能目录）的判定结果。
//
// 除了 ok 之外一律**不是错误**：安装时逐条告警、跳过，其余照装。第三方路径指向一个
// 移动硬盘、一个已经删掉的目录都很正常，不该让整次安装失败。
type State string

const (
	StateOK       State = "ok"
	StateMissing  State = "missing"  // 路径不存在 / 不是目录
	StateInvalid  State = "invalid"  // 没有 SKILL.md，或 frontmatter 不合规
	StateConflict State = "conflict" // 与官方或先登记的第三方同名
)

// Info 是一个技能目录的判定结果。官方的 Info.State 恒为 ok（官方不合规是硬错误，
// 见 InspectOfficial）。第三方的为什么不是 ok，看 Why。
type Info struct {
	Origin      Origin `json:"origin"`
	Name        string `json:"name"`        // 磁盘上读到的 name；读不到为空
	Description string `json:"description"` // 读不到为空（设置页只读展示用）
	Dir         string `json:"dir"`         // 绝对路径
	Index       int    `json:"index"`       // 第三方登记的序号（1 起）；官方为 0
	State       State  `json:"state"`
	Why         string `json:"why,omitempty"`
}

// InspectOfficial 扫描官方技能树：<officialDir>/<name>/SKILL.md。
//
// 与第三方相反，这里**任何一处不合规都是硬错误** —— 官方技能是我们自己发的，
// 装一棵半成品树出去只会让 agent 那边静默少一个技能。错误信息一次列全。
//
// 返回值里的 extraFiles 是顶层**非目录**条目（README.md 之类），调用方照旧原样复制。
func InspectOfficial(officialDir string) (skills []Info, extraFiles []string, err error) {
	ents, err := os.ReadDir(officialDir)
	if err != nil {
		return nil, nil, fmt.Errorf("读不到官方技能目录 %s: %w", officialDir, err)
	}
	var probs []string
	for _, ent := range ents {
		name := ent.Name()
		if !ent.IsDir() {
			extraFiles = append(extraFiles, filepath.Join(officialDir, name))
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(officialDir, name, "SKILL.md"))
		if rerr != nil {
			probs = append(probs, fmt.Sprintf("%s: 技能目录缺少 SKILL.md —— 规范要求每个技能目录里有它", name))
			continue
		}
		meta, perr := ParseFrontmatter(b)
		if perr != nil {
			probs = append(probs, fmt.Sprintf("%s/SKILL.md: %v", name, perr))
			continue
		}
		for _, p := range ValidateMeta(name, meta) {
			probs = append(probs, fmt.Sprintf("%s/SKILL.md: %s", name, p))
		}
		skills = append(skills, Info{
			Origin: OriginOfficial, Name: name, Description: meta.Description,
			Dir: filepath.Join(officialDir, name), State: StateOK,
		})
	}
	if len(probs) > 0 {
		return nil, nil, fmt.Errorf("官方技能不合规，拒绝安装（%d 处）:\n  %s\n\n"+
			"Agent Skills 规范 → https://agentskills.io/specification"+
			"（name 必须等于目录名；description 非空、≤%d 字符）",
			len(probs), strings.Join(probs, "\n  "), DescriptionMax)
	}
	if len(skills) == 0 {
		return nil, extraFiles, fmt.Errorf("官方技能目录里没有任何技能: %s", officialDir)
	}
	return skills, extraFiles, nil
}

// InspectExternal 判定第三方登记表：paths 是配置里 skills.externals 的顺序，
// officialNames 是官方技能的名字（用于撞名判定）。
//
// 永不返回错误：每条登记自己带 State 与 Why。同名时**先登记者优先**，后者的状态是
// conflict —— 顺序写在配置里，用户能看见也能改。
func InspectExternal(paths []string, officialNames []string) []Info {
	takenOfficial := make(map[string]bool, len(officialNames))
	for _, n := range officialNames {
		takenOfficial[n] = true
	}
	taken := make(map[string]int, len(paths)) // name → 先登记且判定为 ok 的序号
	out := make([]Info, 0, len(paths))

	for i, raw := range paths {
		info := Info{Origin: OriginExternal, Index: i + 1, Dir: raw}
		dir := raw
		if abs, aerr := filepath.Abs(raw); aerr == nil {
			info.Dir, dir = abs, abs
		}
		st, serr := os.Stat(dir)
		switch {
		case serr != nil || !st.IsDir():
			info.State, info.Why = StateMissing, "目录不存在"
			out = append(out, info)
			continue
		}

		base := filepath.Base(dir)
		b, rerr := os.ReadFile(filepath.Join(dir, "SKILL.md"))
		if rerr != nil {
			info.State, info.Why = StateInvalid, "目录里没有 SKILL.md"
			out = append(out, info)
			continue
		}
		meta, perr := ParseFrontmatter(b)
		if perr != nil {
			info.State, info.Why = StateInvalid, "SKILL.md: "+perr.Error()
			out = append(out, info)
			continue
		}
		info.Name, info.Description = meta.Name, meta.Description
		if probs := ValidateMeta(base, meta); len(probs) > 0 {
			info.State, info.Why = StateInvalid, "SKILL.md: "+strings.Join(probs, "；")
			out = append(out, info)
			continue
		}
		if takenOfficial[meta.Name] {
			info.State, info.Why = StateConflict, "与官方技能同名，官方优先，已跳过"
			out = append(out, info)
			continue
		}
		if prev, ok := taken[meta.Name]; ok {
			info.State, info.Why = StateConflict, fmt.Sprintf("与第 %d 条同名，先登记者优先，已跳过", prev)
			out = append(out, info)
			continue
		}
		taken[meta.Name] = i + 1
		info.State = StateOK
		out = append(out, info)
	}
	return out
}

// HasAssets 判断一个技能目录下的根级 assets/ 是否有内容（至少一个文件）。
//
// 空目录不算：为一个空目录写一行"静态文件位置"没有意义，反而误导模型去找东西。
//
// 只认**根级** assets/。不做通用 ignore 语法 —— 那会让"为什么这个文件没装进去"
// 变成要读文档才能回答的问题。
func HasAssets(skillDir string) bool {
	dir := filepath.Join(skillDir, assetsDirName)
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		return false
	}
	found := false
	_ = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return nil // 读不动的子目录不影响"有没有内容"这个判断
		}
		if !d.IsDir() {
			found = true
			return fs.SkipAll
		}
		return nil
	})
	return found
}
