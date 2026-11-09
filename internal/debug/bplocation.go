package debug

// 断点位置里的**文件名前缀**怎么定（缺陷 #3 / A26 的现场半）。
//
// 现场（.agents/plan/L1-结果-2026-10-18.md §5.1 / §7-3）：
//
//	debug start axmt500_wf --module asf   →  停站文件 asf_axmt500_wf.4gl
//	info breakpoints 里真名                →  axm_axmt500_wf.4gl
//	break asf_axmt500_wf.4gl:2141         →  No source file named asf_axmt500_wf.4gl.
//
// `asf_` 是哪来的（读代码，不是猜）：`--module` 的默认值被写死成 "asf"
// （internal/cli/debug/debugctl.go 的 flag 注册），而 `Session.Module` 本来是**登录后**
// 由 resolveModule（session.go:1254）按 `<作业>.42r` 在各模块 42r 目录里搜出来的 ——
// 写死的默认值把那条权威解析整条短路，于是入口停站与断点位置都用了一个编出来的模块名。
//
// 所以这里的规矩是：
//  1. 不给 --module ⇒ 走自动解析（唯一权威来源，`axmt500_wf → axm`、`bsft001_wf → asf`
//     这类映射只有 42r 目录能给答案，按作业名前缀推是**猜**，不许）；
//  2. 位置里的前缀与会话已知模块不一致 ⇒ 用**会话已知事实**改正，并说明改了什么叫；
//  3. 会话还没拿到模块 ⇒ **不改**，直接告诉人怎么给才不改错（A15：取不到就说取不到）。

import (
	"fmt"
	"strings"
)

// ModuleFlagHelp 是 `--module` 的帮助文本（CLI 注册 flag 时用同一份，别各写一句）。
func ModuleFlagHelp() string {
	return "T100 模块目录名。**留空 = 登录后自动解析**（按 <作业>.42r 在各模块 42r 目录里搜，" +
		"实测 axmt500_wf→axm、bsft001_wf→asf，这是唯一权威来源）；显式给出则按它拼 <模块>_<作业>.4gl"
}

// BPLocContext 是修正断点位置所需的那几条"会话已知事实"。
//
// 都由会话自己的解析结果填（登录后的 Module、客制目录判定、本会话作业名），
// 所以这里的修正不是猜测：拿不到的字段留空，ResolveLocation 就会拒绝改并给出办法。
type BPLocContext struct {
	Module     string   // 登录后权威解析到的模块目录名（resolveModule 的结果）
	CustModule string   // 客制模块名（apm→cpm）；非空时优先
	Progs      []string // 本会话可能启动的作业名（Prog / RunProg / runProgName()）
}

// knowsProg 这个位置的文件是不是"本会话自己的作业"。
//
// 只修本会话作业的文件名：`lib_cl_ap_code.4gl:4593` 这类公共库、别人模块的文件
// 一概不碰 —— 改错了比不改更坏（会变成"断点下到别的文件的同名行上"）。
func (c BPLocContext) knowsProg(prog string) bool {
	for _, p := range c.Progs {
		p = strings.TrimSuffix(strings.TrimSpace(p), ".4gl")
		if p == "" {
			continue
		}
		if strings.EqualFold(p, prog) {
			return true
		}
		// 入口文件可能是 <模块>_<RunProg>.4gl（gzzz_t 解析出的实体程序），
		// 所以 DVM 名的整段也要能匹配。
		if strings.EqualFold(strings.TrimPrefix(p, c.wantPrefix()+"_"), prog) {
			return true
		}
	}
	return false
}

// wantPrefix 源文件名该用的模块前缀：客制目录优先（转客制作业从 c** 启动）。
func (c BPLocContext) wantPrefix() string {
	if c.CustModule != "" {
		return c.CustModule
	}
	return c.Module
}

// ResolveLocation 修正 `break` 位置里的模块前缀；返回（修正后位置, 说明）。
// 说明为空 = 一个字符都没改。
//
// 只在"位置看起来就是 <模块>_<作业>.4gl[:行]，且作业是本会话的"时动手；
// 其余（行号、函数名、公共库文件、别的作业）一律原样返回。
func (c BPLocContext) ResolveLocation(loc string) (string, string) {
	raw := strings.TrimSpace(loc)
	if !strings.Contains(raw, ".4gl") {
		return loc, ""
	}
	file, line := raw, ""
	if i := strings.LastIndex(raw, ":"); i > 0 && isAllDigits(raw[i+1:]) {
		file, line = raw[:i], raw[i+1:]
	}
	if !strings.HasSuffix(file, ".4gl") {
		return loc, ""
	}
	base := strings.TrimSuffix(file, ".4gl")
	j := strings.Index(base, "_")
	if j <= 0 {
		return loc, ""
	}
	prefix, prog := base[:j], base[j+1:]
	if !c.knowsProg(prog) {
		return loc, "" // 不是本会话的作业：不碰
	}
	want := c.wantPrefix()
	if want == "" {
		// A15：事实不足就说取不到，并给出**可操作**的拿法。
		return loc, fmt.Sprintf("位置 %q 的模块前缀 %q 没法核对：本会话还没拿到登录后的模块"+
			"（它是按 <作业>.42r 在各模块 42r 目录里自动解析出来的，取不到就不猜）。"+
			"三步任选：① tt debug exec \"info breakpoints\" 看真实文件名；② tt debug stop 看停站位置的 file 字段；"+
			"③ 直接 break <行号>（当前文件），或 break <函数名> 让 fgldb 自己定位", raw, prefix)
	}
	if strings.EqualFold(prefix, want) {
		return loc, "" // 前缀本来就对（如 bsft001_wf 真在 asf 模块里）
	}
	fixed := want + "_" + prog + ".4gl"
	if line != "" {
		fixed += ":" + line
	}
	note := fmt.Sprintf("位置 %q 的模块前缀 %q 与本会话登录后解析到的模块 %q 不一致"+
		"（源文件名是 <模块>_<作业>.4gl）：已改用 %s。要换模块请用 --module（留空=自动解析）",
		raw, prefix, want, fixed)
	return fixed, note
}

// isAllDigits 纯数字判定（行号后缀）。
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
