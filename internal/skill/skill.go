// Package skill 是「技能」这件事的唯一实现：frontmatter 的解析与合规判定、
// 来源（官方 / 第三方登记）的扫描与冲突判定、装到 agent 目录的复制与改写。
//
// 为什么单独成包：这些逻辑原来长在 internal/cli/install.go 里，只有命令行一个调用方。
// 现在设置页也要「列出有哪些技能、哪个失联、装到哪儿」，而它拿不到 cli 包（clap 门面
// 里全是 cobra）。同一份判定必须两处一致 —— 尤其「什么算合规」和「名字撞了谁赢」，
// 两处各写一遍就会养出两套口径。
//
// 本包是纯管线：不读配置、不打印、不碰 cobra。目标目录由调用方给，登记清单由调用方读，
// 错误与告警都作为返回值带出去。
package skill

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// DescriptionMax 是 Agent Skills 规范给 description 的长度上限。
const DescriptionMax = 1024

// Meta 是规范要求必填的两个字段 —— 也是别的 agent 拿来当索引的两个字段：
// name 决定技能身份，description 决定它何时被加载。
type Meta struct {
	Name        string
	Description string
}

// ParseFrontmatter 从 SKILL.md 里抠出 name 与 description。
//
// 不引 YAML 依赖：这里要判的是两个标量字符串，不是完整解析。只认**顶层**的 `键: 值` ——
// 缩进行属于 metadata 之类的嵌套块，一律跳过（否则 `metadata:` 底下的 name 会被当成技能的 name）。
// 值是 `|` / `>` 块标量时收拢后面的缩进行；两边成对的引号去掉。
func ParseFrontmatter(b []byte) (Meta, error) {
	s := strings.TrimPrefix(string(b), "\ufeff") // BOM
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return Meta{}, fmt.Errorf("开头没有 YAML frontmatter（规范要求第一行是 ---，里面写 name 与 description）")
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return Meta{}, fmt.Errorf("frontmatter 没有闭合的 ---")
	}

	body := lines[1:end]
	var m Meta
	for i := 0; i < len(body); i++ {
		line := body[i]
		if line == "" || line[0] == ' ' || line[0] == '\t' || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		val = strings.TrimSpace(val)
		switch strings.TrimSpace(key) {
		case "name":
			m.Name = unquoteYAML(val)
		case "description":
			if val == "|" || val == ">" || val == "|-" || val == ">-" {
				var parts []string
				for j := i + 1; j < len(body); j++ {
					l := body[j]
					if strings.TrimSpace(l) == "" {
						continue
					}
					if l[0] != ' ' && l[0] != '\t' {
						break
					}
					parts = append(parts, strings.TrimSpace(l))
					i = j
				}
				m.Description = strings.Join(parts, " ")
			} else {
				m.Description = unquoteYAML(val)
			}
		}
	}
	return m, nil
}

func unquoteYAML(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		return s[1 : len(s)-1]
	}
	return s
}

// ValidateMeta 按 Agent Skills 规范逐条检查，返回全部问题（一次说完，别让人改一处跑一次）。
//
// 为什么是硬失败：不合规的技能装出去**不会报错**，失败都在我们看不到的地方 ——
// description 缺失时客户端直接跳过该技能；name 不等于目录名是规范的硬约束，
// 但各家客户端的宽容度不一（有的警告后照装）。两种都不该由我们发出去。
//
// dir 只用于比对 name（技能身份 = 目录名）。
func ValidateMeta(dir string, m Meta) []string {
	var probs []string
	switch {
	case m.Name == "":
		probs = append(probs, "frontmatter 里没有 name")
	case m.Name != dir:
		probs = append(probs, fmt.Sprintf("name 是 %q，必须等于目录名 %q", m.Name, dir))
	}
	if m.Name != "" && !ValidName(m.Name) {
		probs = append(probs, fmt.Sprintf("name %q 不合规：只允许小写字母、数字与连字符，"+
			"不能有连续或首尾连字符，最长 64 字符", m.Name))
	}
	switch n := utf8.RuneCountInString(m.Description); {
	case m.Description == "":
		probs = append(probs, "description 是空的：别的 agent 靠它决定何时加载这个技能")
	case n > DescriptionMax:
		probs = append(probs, fmt.Sprintf("description 有 %d 个字符，超过规范上限 %d", n, DescriptionMax))
	}
	return probs
}

// ValidName 是规范的 name 字符集：小写字母、数字、连字符，最长 64 字符，
// 不能以连字符开头或结尾，不能有连续连字符。
func ValidName(s string) bool {
	if utf8.RuneCountInString(s) > 64 {
		return false
	}
	prevHyphen := true // 首字符不许是连字符
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			prevHyphen = false
		case r == '-':
			if prevHyphen {
				return false
			}
			prevHyphen = true
		default:
			return false
		}
	}
	return !prevHyphen // 末字符不许是连字符
}
