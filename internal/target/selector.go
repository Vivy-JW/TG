package target

import (
	"sort"
	"strings"
)

// SelectorKind 是选择器的六种形态（契约 2.1.4 的文法 + C2.4 的别名分流）。
//
// 它对应 2.2 那张"合法写法一览"里的 ③–⑧（① 是"省略"，用 `Request.Selector == nil` 表示）。
type SelectorKind int

const (
	// SelectorAlias 是写法 ③：`--session 汇率单`（按**别名**找）。别名里不许有 `/` 与 `#`（C2.4）。
	SelectorAlias SelectorKind = iota
	// SelectorRef 是写法 ④：`恒烁正式区#1`（**规范 id**：环境 + 序号，最精确且恒定）。
	SelectorRef
	// SelectorEnvJobSeq 是写法 ⑤：`恒烁正式区/apmt520#1`（（环境, 作业, 序号）；作业段只用来核对）。
	SelectorEnvJobSeq
	// SelectorEnvJob 是写法 ⑥：`恒烁正式区/apmt520`（（环境, 作业）；多条即 F3）。
	SelectorEnvJob
	// SelectorEnvEnt 是写法 ⑦：`恒烁正式区/99`（（环境, 企业编号）；第二段全数字，C2.3）。
	SelectorEnvEnt
	// SelectorEnv 是写法 ⑧：`恒烁正式区`（按环境找；该环境下唯一一条才落，多条即 F4）。
	SelectorEnv
)

// String 给出形态名（失败信息与文档用）。
func (k SelectorKind) String() string {
	switch k {
	case SelectorAlias:
		return "alias"
	case SelectorRef:
		return "env#n"
	case SelectorEnvJobSeq:
		return "env/job#n"
	case SelectorEnvJob:
		return "env/job"
	case SelectorEnvEnt:
		return "env/ent"
	case SelectorEnv:
		return "env"
	}
	return "?"
}

// Selector 是解析后的选择器（纯数据）。字段按"用户**写了什么**"填，没写的留零值。
type Selector struct {
	// Raw 是**原样**回显（C5.6：requested.session 用它）。任何规范化都发生在这里之后，
	// 不许回写到 Raw —— 否则"我要的"与"我拿到的"就没法一眼比出来。
	Raw string
	// Kind 见 SelectorKind。
	Kind SelectorKind
	// Env 是环境段（别名形态为空）。
	Env string
	// Prog 是作业段（仅 SelectorEnvJob / SelectorEnvJobSeq）。
	Prog string
	// Ent 是企业编号段（仅 SelectorEnvEnt，字符串形态，C2.3）。
	Ent string
	// N 是序号（仅 SelectorRef / SelectorEnvJobSeq；0 = 没写）。
	//
	// 序号**始终被服务端分配、始终存在**（C2.2），省掉的只是写法。
	N int
	// Alias 是别名的原样取值（仅 SelectorAlias）。
	Alias string
}

// ParseSelector 解析 `--session` 的取值（契约 2.1.4 的文法 + C2.4 的分流）。
//
// 文法：
//
//	选择器 := 环境名 | 环境名 "#" 序号 | 环境名 "/" 作业名 [ "#" 序号 ] | 环境名 "/" 企业编号
//
// 三条口径：
//
//   - 含 `/` 或 `#` 的一律按选择器解析，**不含这两个字符**的按别名（C2.4 的落地判据）；
//   - 空串（或全空白）按**文法非法**处理（F11），不当成"省略" —— 那个由 `Request.Selector == nil` 表示；
//   - 全程**不发请求、不查会话池**（C2.8：F9–F11 在发请求之前判定，这是纯函数）。
//
// **一处必须说清的边界**：C2.4 的分流只看 `/` 与 `#`，而写法 ⑧（`--session 恒烁正式区`）
// 字面上与别名 ③ 一模一样。本函数拿不到配置表，所以**不带分隔符的取值先按别名记下**；
// 知道环境清单的调用方请用 `ParseSelectorWith`（`Resolve` 也会用同一条规则再判一次，
// 所以这里就算记成了别名，落位时也不会错）。
//
// 返回的 `*Failure` 非 nil 即失败（F9 / F10 / F11，都是 `USAGE` 退 2）。
func ParseSelector(raw string) (Selector, *Failure) { return ParseSelectorWith(raw, nil) }

// ParseSelectorWith 与 `ParseSelector` 相同，但**知道配置里的环境名清单**，
// 于是能判"光有环境名"的写法 ⑧（精确相等才算 —— C2.1：不做前缀、不做大小写折叠）。
func ParseSelectorWith(raw string, environments []string) (Selector, *Failure) {
	if strings.TrimSpace(raw) == "" {
		return Selector{}, failureSelectorGrammar(raw) // F11
	}
	// C2.4：不含 `/` 与 `#` 的按别名解析（别名允许中文、空格与任意其它字符，Q17）。
	if !strings.ContainsAny(raw, SelectorSeps) {
		return promoteEnvName(Selector{Raw: raw, Kind: SelectorAlias, Alias: raw}, environments), nil
	}
	// 文法里最多一个序号段。
	if strings.Count(raw, SeqSep) > 1 {
		return Selector{}, failureSelectorGrammar(raw) // F10
	}

	sel := Selector{Raw: raw}
	head := raw
	if i := strings.Index(raw, SeqSep); i >= 0 {
		head = raw[:i]
		seq := raw[i+len(SeqSep):]
		n, ok := parseSeq(seq)
		if !ok {
			return Selector{}, failureSeqInvalid(seq) // F9
		}
		sel.N = n
	}

	segs := strings.Split(head, SelectorSep)
	switch len(segs) {
	case 1:
		if segs[0] == "" {
			return Selector{}, failureSelectorGrammar(raw) // F10（环境名为空）
		}
		sel.Env = segs[0]
		if sel.N > 0 {
			sel.Kind = SelectorRef
		} else {
			sel.Kind = SelectorEnv
		}
	case 2:
		if segs[0] == "" || segs[1] == "" {
			return Selector{}, failureSelectorGrammar(raw) // F10（空段）
		}
		sel.Env = segs[0]
		if SecondSegmentKind(segs[1]) == SecondSegEnt {
			if sel.N > 0 {
				// 文法里 `环境/企业编号` 不带序号段（2.1.4）—— 带了就是形态不对。
				return Selector{}, failureSelectorGrammar(raw) // F10
			}
			sel.Ent = segs[1]
			sel.Kind = SelectorEnvEnt
		} else {
			sel.Prog = segs[1]
			if sel.N > 0 {
				sel.Kind = SelectorEnvJobSeq
			} else {
				sel.Kind = SelectorEnvJob
			}
		}
	default:
		return Selector{}, failureSelectorGrammar(raw) // F10（段数不对）
	}
	return sel, nil
}

// promoteEnvName 是 C2.4 与写法 ⑧ 之间那条缝的补法。
//
// C2.4 的分流判据只看"有没有 `/` 与 `#`"，于是 `--session 恒烁正式区`（写法 ⑧，按环境找）
// 与 `--session 汇率单`（写法 ③，按别名找）在字面上**完全同形**。能把两者分开的只有一件事：
// **它是否精确等于一个已配置的环境名**（C2.1：精确匹配，不做前缀、不做大小写折叠）。
//
// 同名时的优先级：**环境名赢**。理由：环境名是配置里的固定词表，优先级能由"查表"给出
// 唯一答案；反过来（别名赢）会让写法 ⑧ 永远不可达，而 2.2 的 ⑧ 与 F4 都指着它。
// 解析层没有配置表时按字面记成别名，`Resolve` 拿到池子后用同一个函数再判一次。
func promoteEnvName(sel Selector, environments []string) Selector {
	if sel.Kind != SelectorAlias || len(environments) == 0 {
		return sel
	}
	for _, env := range environments {
		if env != "" && env == sel.Alias {
			return Selector{Raw: sel.Raw, Kind: SelectorEnv, Env: env}
		}
	}
	return sel
}

// parseSeq 解析序号段：十进制、>= 1、**不写前导零**（C2.1.2 的规范 id 文法，F9）。//
// `#01` 被刻意拒掉：它与 `#1` 是同一个地址，允许两种写法就会对不上账。
func parseSeq(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	if len(s) > 1 && s[0] == '0' {
		return 0, false
	}
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
		n = n*10 + int(s[i]-'0')
		if n > 1<<30 {
			return 0, false // 序号是"第几条"，不可能是这个量级；防溢出而不是防呆
		}
	}
	if n < 1 {
		return 0, false
	}
	return n, true
}

// SecondSegmentKind 是 C2.3 的判据：选择器第二段**全数字 → 企业编号；否则 → 作业名**。
//
// 判据**只**看"全数字"这四个字，不先做 B29 的合法性校验：`1a` 这种串按 C2.3 落到作业那一支，
// 随后在"环境/作业"那一步按 F7 报 —— 它**不会**被静默当成企业编号 1
// （那才是"看起来落上了、其实落在别处"）。
func SecondSegmentKind(seg string) string {
	if seg == "" {
		return ""
	}
	for _, r := range seg {
		if r < '0' || r > '9' {
			return SecondSegJob
		}
	}
	return SecondSegEnt
}

// ValidateAlias 校验 `--name` 的取值（C2.4 / C2.5 / F19 / Q17）。
//
// 规矩只有三条：**非空**（去两侧空白后仍非空）、去两侧空白后 1–64 个**字符**、
// **只排除 `/` 与 `#`**。中文、空格、标点、任意其它字符一律允许。
// 返回 nil = 合法。
func ValidateAlias(raw string) *Failure {
	name := strings.TrimSpace(raw)
	if name == "" {
		return failureAliasInvalid() // F19（空串）
	}
	if strings.ContainsAny(name, AliasForbid) {
		return failureAliasInvalid() // F19（含 `/` 或 `#`）
	}
	if n := len([]rune(name)); n < AliasMinRunes || n > AliasMaxRunes {
		return failureAliasInvalid() // F19（长度）
	}
	return nil
}

// SameAlias 是别名的**查重**判据（C2.5）：不区分大小写地比 ——
// "两个只差大小写的同名别名算撞车"（F18 / F5）。
//
// 注意它与**查找**的分工：`--session <别名>` 的查找按原样比（C2.5 末句），
// 只有"这个别名还能不能给别人用"才用本函数。
func SameAlias(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

// AliasMatches 是**查找**判据：按 C2.5 口径把两侧空白去掉后原样比对
// （别名在创建时就被 `ValidateAlias` 去过两侧空白，存的是 trim 后的值；
//
//	这里去空白是让 `--session " 汇率单"` 这种手滑也能落到同一条）。
//
// 它**不**做大小写折叠：那一条只用于查重（SameAlias）。
func AliasMatches(stored, want string) bool {
	return strings.TrimSpace(stored) == strings.TrimSpace(want)
}

// sortCandidates 按"序号 → 原始句柄"给候选清单定序，让同一个错误两次跑出的文案一样。
func sortCandidates(list []Candidate) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].N != list[j].N {
			return list[i].N < list[j].N
		}
		return list[i].ID < list[j].ID
	})
}
