package debug

// A20（契约 §5 A20）：条件断点 / 忽略次数 / 监视点。
// **A26（契约 §8，取代 A20 的 `condition` 映射）：条件改写成 inline `if`。**
//
// 事实基础（主机109 真机实测，见 .agents/plan/L1-结果-2026-10-18.md §5.1，不重新逆向）：
//   - fgldb 的命令表里**没有 `condition`**；实跑 `condition 1 …` 回
//     `Undefined command: "condition"`。`condition` 在本机只作为 DAP 的 JSON 字段名存在；
//   - 所以条件必须写在 break 指令里：`break <loc> if <expr>`（GDB 的 "stop only if %s"）；
//   - `ignore` 可用（回显 `Will ignore next N crossings of breakpoint N`）；
//   - `watch` 上**没有**"带条件"的实测依据 ⇒ watch + Cond **明确报不支持**，不猜。
//
// 本文件只做两件事：**校验**（要拒就拒得明白，且一个字节都不发）与**生成命令**。
// 它不碰 *Session、不做 IO，所以可以在没有真机的情况下全覆盖。
//
// 注入防护的口径是「**先校验再拼**」，不是「先拼再 sanitize」：
// 含换行/以 end 结尾/超长的条件**整条拒掉**，绝不做"偷偷删掉危险字符"那种改写 ——
// 改写会静默改变用户的表达式语义，比拒绝坏得多。

import (
	"fmt"
	"strings"
)

const (
	// BPKindBreak 普通断点（Kind 为空时的默认值）
	BPKindBreak = "break"
	// BPKindWatch 监视点（`watch <location>`）
	BPKindWatch = "watch"

	// MaxBPCondBytes 条件表达式字节上限（A20：超长 ⇒ 拒）
	MaxBPCondBytes = 512
)

// BreakpointOpts 是 A20 冻结结构里的那一组字段。
//
// 为什么不叫 `Breakpoint`：本包已有**运行时断点账本**类型 `Breakpoint`
// （Num/File/Line/Func/Enabled/Note，来自 `info breakpoints`）。这里是**输入选项**，
// 两者职责不同（一个是"用户要下什么"，一个是"fgldb 说现在有什么"），
// 同名会让 `session.go` 里到处是别名。
type BreakpointOpts struct {
	Location string // 行号 / 函数名 / file:line；Kind=watch 时是被监视的表达式
	Cond     string // 条件表达式（空 = 无条件；非空 ⇒ 内联进 `break … if …`）
	Ignore   int    // 忽略前 N 次命中（0 = 不忽略）
	Kind     string // "break" | "watch"（空 = "break"）
}

// kind 归一化 Kind（空 ⇒ break）
func (o BreakpointOpts) kind() string {
	if o.Kind == "" {
		return BPKindBreak
	}
	return o.Kind
}

// watchLocChars 监视点 location 的**白名单**（标识符与表达式所需字符）。
//
// 用白名单而不是黑名单：黑名单永远少一个字符。这里刻意**不含**引号、分号、
// 反引号、反斜杠、制表符、括号外的 `|` —— 那些在表达式里没有合法用途。
func watchLocAllowed(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	}
	return strings.ContainsRune("_ .$:*&()[]+-/%<>=!~", r)
}

// Validate 校验全部字段。**校验通过之前不生成任何命令**（拒绝路径因此天然"不发字节"）。
// 错误文案要点名是哪个字段、为什么 —— 排障时不用猜。
func (o BreakpointOpts) Validate() error {
	loc := o.Location
	if strings.ContainsAny(loc, "\r\n") {
		return fmt.Errorf("断点选项 Location 含换行符：位置必须是单行（拒绝执行）")
	}
	if strings.TrimSpace(loc) == "" {
		return fmt.Errorf("断点选项 Location 为空：没有可下的位置（拒绝执行）")
	}
	switch o.Kind {
	case "", BPKindBreak, BPKindWatch:
	default:
		return fmt.Errorf("断点选项 Kind 非法（%q）：只能是 break / watch，空表示 break（拒绝执行）", o.Kind)
	}
	if o.kind() == BPKindWatch {
		for _, r := range loc {
			if !watchLocAllowed(r) {
				return fmt.Errorf("断点选项 Location 含白名单外字符（%q）：watch 只接受标识符/表达式（拒绝执行）", string(r))
			}
		}
		// A26：本机 fgldb 的 watch 没有"带条件"的实测依据（`watch X if Y` 到底认不认，
		// 谁都没验过）⇒ 明确报不支持，绝不用"大概能行"的猜法发出去。
		if strings.TrimSpace(o.Cond) != "" {
			return fmt.Errorf("不支持条件监视点（Kind=watch 且 Cond=%q）：fgldb 的 watch 没有带条件的实测依据，"+
				"条件只能内联在 break 上 —— 要条件请用 Kind=break，或去掉 Cond 只留监视（拒绝执行）", o.Cond)
		}
	}
	if o.Ignore < 0 {
		return fmt.Errorf("断点选项 Ignore 为负数（%d）：忽略次数不能小于 0，0 表示不忽略（拒绝执行）", o.Ignore)
	}
	// Cond：换行 ⇒ 拒（多行会把后续内容变成**另一条 fgldb 命令**）
	if strings.ContainsAny(o.Cond, "\r\n") {
		return fmt.Errorf("断点选项 Cond 含换行符：条件表达式必须是单行（拒绝执行）")
	}
	// Cond：以 end 结尾 ⇒ 拒（GDB 的 `end` 是命令块结束符）
	if f := strings.Fields(o.Cond); len(f) > 0 && strings.EqualFold(f[len(f)-1], "end") {
		return fmt.Errorf("断点选项 Cond 以 end 结尾：GDB 会把 end 当成命令块结束符（拒绝执行）")
	}
	// Cond：超长 ⇒ 拒
	if len(o.Cond) > MaxBPCondBytes {
		return fmt.Errorf("断点选项 Cond 超长（%d 字节 > 上限 %d）：请精简条件表达式（拒绝执行）",
			len(o.Cond), MaxBPCondBytes)
	}
	return nil
}

// BreakCmd 返回"下断点"那一条命令：
//
//	watch <loc>                      （Kind=watch，条件已在校验里被拒）
//	break <loc> if <expr>            （Cond 非空，A26：条件内联）
//	break <loc>                      （无条件）
//
// 校验不过 ⇒ 返回空串与错误（调用方据此**不发任何字节**）。
func (o BreakpointOpts) BreakCmd() (string, error) {
	if err := o.Validate(); err != nil {
		return "", err
	}
	loc := strings.TrimSpace(o.Location)
	if o.kind() == BPKindWatch {
		return "watch " + loc, nil
	}
	if expr := strings.TrimSpace(o.Cond); expr != "" {
		return "break " + loc + " if " + expr, nil
	}
	return "break " + loc, nil
}

// OptCmds 返回断点落定后要发的追加命令。
//
// **A26 起这里只剩 `ignore`**：条件已经在 BreakCmd 里内联进 break，
// 不再有 `condition <n> <expr>`（本机 fgldb 没有这条命令）。
// Ignore=0 ⇒ 不发 ignore。
func (o BreakpointOpts) OptCmds(num int) ([]string, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	if num < 0 {
		return nil, fmt.Errorf("断点编号无效（%d）：下断点之后才能追加条件/忽略次数", num)
	}
	var out []string
	if o.Ignore > 0 {
		out = append(out, fmt.Sprintf("ignore %d %d", num, o.Ignore))
	}
	return out, nil
}

// Commands 生成完整命令序列：BreakCmd（条件已在其中）+ 可选的 ignore（顺序固定）。
//
// num 是 fgldb 分配的断点号；num < 0 表示"还没下、编号未知"，此时只返回第一条
// （先下断点 → 从响应里读编号 → 再调一次 OptCmds）。校验不过一律返回 nil 切片：
// 调用方按 len(cmds) 判定"一条都不发"，不必再分情况。
func (o BreakpointOpts) Commands(num int) ([]string, error) {
	first, err := o.BreakCmd()
	if err != nil {
		return nil, err
	}
	if num < 0 {
		return []string{first}, nil
	}
	rest, err := o.OptCmds(num)
	if err != nil {
		return nil, err
	}
	return append([]string{first}, rest...), nil
}
