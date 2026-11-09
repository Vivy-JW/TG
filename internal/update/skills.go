package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"tt/internal/config"
)

// 技能一致性：`tt install skills` 把 mkdir 之外的技能树复制到 agent 会读的目录里，
// 那一份**不随二进制升级自动变**。于是升级完就会出现"tt 是 0.2.2，agent 读的还是
// 0.2.1 的手册"——而 skills/ 是评测里执行者唯一能读的东西，这种错位比二进制旧更坏。
//
// 这份戳就是为了把两件事对上：装过哪些目录、装它们的是哪个 tt 版本。它住在数据目录
// （`.tt-skills.json`，与 `.tt-serve.json` 同类：不是缓存），因为它是**两个命令之间的
// 状态**：install 写、update 与 version 读。
//
// 为什么不给 SKILL.md 加 `version:` 字段：frontmatter 是跨客户端契约（我们的校验器正是
// 按它硬失败的那一套，见 internal/cli/install.go），多一个键要赌各家解析器的宽容度；
// 而且"装到哪些目录"这件事本来就不属于某个技能，属于这次安装。
type SkillsStamp struct {
	Version   string    `json:"version"`   // 写这份戳时手上那份 tt 的版本
	UpdatedAt time.Time `json:"updatedAt"` // 最后一次写入时间
	Targets   []string  `json:"targets"`   // 装到过哪些目录（**绝对路径**：升级时的 cwd 未必与安装时相同）
}

// SkillsStampPath 戳的落点。
func SkillsStampPath(dataDir string) string {
	if dataDir == "" {
		return ""
	}
	return filepath.Join(dataDir, ".tt-skills.json")
}

// LoadSkillsStamp 读戳；没有或坏掉时返回 nil（调用方按"没装过"处理）。
func LoadSkillsStamp(dataDir string) *SkillsStamp {
	p := SkillsStampPath(dataDir)
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var s SkillsStamp
	if err := json.Unmarshal(b, &s); err != nil {
		return nil
	}
	return &s
}

// RecordSkillsTarget 记下"技能装到了这个目录、装它们的是 version"。
//
// 幂等：同一目录再来一次只更新版本与时间，不产生第二条。目标路径先绝对化 ——
// `--to .agents/skills` 是相对当时的当前目录的，升级时那个目录早就不在了。
func RecordSkillsTarget(dataDir, dir, version string) error {
	p := SkillsStampPath(dataDir)
	if p == "" {
		return fmt.Errorf("数据目录不可用，技能安装记录无处可落")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("把技能目录 %s 绝对化失败: %w", dir, err)
	}
	st := LoadSkillsStamp(dataDir)
	if st == nil {
		st = &SkillsStamp{}
	}
	st.Version, st.UpdatedAt = version, time.Now()
	found := false
	for i, t := range st.Targets {
		if samePath(t, abs) {
			st.Targets[i], found = abs, true
			break
		}
	}
	if !found {
		st.Targets = append(st.Targets, abs)
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return config.AtomicWrite(p, append(b, '\n'))
}

// StaleSkillsTargets 返回需要刷新的技能目录：戳记的版本与 now 不同就全是。
//
// 版本相同就一个都不返回 —— 升级之外的一切（重新安装、手动替换二进制）都可能让戳与
// 实际不符，但只有"版本不同"是我们能可靠判断的那一种；猜错了会白刷一遍用户的技能目录。
func StaleSkillsTargets(dataDir, version string) []string {
	st := LoadSkillsStamp(dataDir)
	if st == nil || len(st.Targets) == 0 {
		return nil
	}
	if st.Version == version {
		return nil
	}
	out := make([]string, 0, len(st.Targets))
	for _, t := range st.Targets {
		if _, err := os.Stat(t); err == nil {
			out = append(out, t)
		}
	}
	return out
}

// SkillsDriftHint 一句话说明技能与二进制不同版（`tt version` 与设置页共用）。
// 对得上、或根本没装过，都返回空串。
func SkillsDriftHint(dataDir, version string) string {
	st := LoadSkillsStamp(dataDir)
	if st == nil || st.Version == "" || st.Version == version || len(st.Targets) == 0 {
		return ""
	}
	return fmt.Sprintf("agent 目录里的技能是 %s 版装的（装到 %d 处）——跑 tt install skills --force 刷新",
		st.Version, len(st.Targets))
}
