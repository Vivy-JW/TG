package debug

// 断点面：`tt debug bp list / add / del`。
//
// # 为什么单开一个命令（A34）
//
// 命中计数是内核**已经给了**的事实（`info breakpoints` 的 `breakpoint already hit N time(s)`，
// 见契约 A33）。从前这条线上只有 `exec "info breakpoints"`，人得自己读 fgldb 原文才看得见
// "到底命中过没有" —— 而 `Address` 恒 `0x00000000`（命中与否都是 0），拿它当判据就会得出
// "断点没装上"的错误结论。所以这里把它变成一条**结构化的、带计数**的命令。
//
// # running 时的纪律（A34，冻结）
//
// 程序 `running` 时下断/查断，服务端直接 `400 {"code":"USAGE","error":"程序运行中,
// 禁止发送调试命令(仅允许中断)"}`。收到它**不重试、不静默**，而是给能照做的指引：
// 先 `tt debug interrupt`（这是 running 时唯一允许的命令），停下后再下断。
//
// 允许的自动化**只有 interrupt**。本工具选择**不自动中断**（`withRunningHint` 只给话）：
// 程序常常正停在 GDC 界面等用户输入，自动 SIGINT 会把用户正在填的对话框直接取消；
// 而这个决定的另一半 —— **绝不自动 continue / run** —— 是硬约束，与选哪种都无关。
// 两条都由 `bp_test.go` 钉住（桩只收到一次 POST，且没有任何 raw/continue/run 请求）。

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/debug"
)

// bpView 是 `bp list` 的输出单元。
//
// **不复用内核的 `debug.Breakpoint`**：那个类型是"fgldb 说现在有什么"的账本，字段标签带
// `omitempty`（`hits` 为 0 时整键消失）。而这里要的是**给人/给 jq 的一份稳定形状**：
// `hits` 必须**永远存在**（没有命中就是 0），否则"没命中"与"这个版本不支持"在读侧分不清。
type bpView struct {
	Num     int    `json:"num"`
	Kind    string `json:"kind"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Func    string `json:"func,omitempty"`
	Module  string `json:"module,omitempty"`
	Enabled bool   `json:"enabled"`
	Cond    string `json:"cond,omitempty"`
	Ignore  int    `json:"ignore,omitempty"`
	// Hits 命中次数（A33）。**没有 omitempty**：0 也必须出现在 JSON 里。
	Hits int    `json:"hits"`
	Note string `json:"note,omitempty"`
}

// runningRejectMark 是服务端在"程序运行中"拒绝调试命令时信封里的那句原文（A34 实测）。
// 只认这一句：别的 400（参数错、位置不存在）不套 interrupt 指引。
const runningRejectMark = "程序运行中"

// withRunningHint 把 400「程序运行中…」变成**可执行**的指引（A34）。
//
// 判据是报文原文，不是 HTTP 码：同一条 `400` 还可能是"断点位置不存在"之类的参数错，
// 那些跟"程序在跑"毫无关系，套上 interrupt 指引只会把人带偏。
func withRunningHint(err error) error {
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), runningRejectMark) {
		return err
	}
	return fmt.Errorf("%w\n→ 先停站再下断：`tt debug interrupt`（程序运行中**只允许中断**），"+
		"等它停下（`tt debug wait --for stopped`）之后重新发这条命令。\n"+
		"  本工具不会自动重试这条命令，也**绝不**自动 continue / run —— 自动发 SIGINT 会把"+
		"用户正在填的 GDC 对话框直接取消，所以那一步留给人。", err)
}

// bpList 取会话的断点账本（含命中计数）。
func bpList(id string) ([]bpView, error) {
	views, _, _, err := bpListFresh(id)
	return views, err
}

// bpListFresh 是 A36 的真身：连同"这份账本**是不是刚从内核读回来**"一起给。
//
// 为什么单列一个函数：`bpList` 的既有调用点只关心清单；而**命令面**必须能说
// 「运行态读不到内核 ⇒ 下面的 hits 可能不是最新」（A36/A34：不许拿账本冒充当下）。
// 服务端在响应顶层给 `fresh` + `note`（`internal/debug/api.go` 的 hBPList）。
func bpListFresh(id string) ([]bpView, bool, string, error) {
	path := dbgSessionURL(id, "/breakpoints")
	if path == "" {
		return nil, false, "", fmt.Errorf("没有可用的会话句柄（先 tt debug start 或 tt debug status 看看有没有会话）")
	}
	data, err := dbgAPI("GET", path, nil)
	if err != nil {
		return nil, false, "", withRunningHint(err)
	}
	var r struct {
		Breakpoints []debug.Breakpoint `json:"breakpoints"`
		Fresh       *bool              `json:"fresh"`
		Note        string             `json:"note"`
	}
	if err := jsonUnmarshal(data, &r); err != nil {
		return nil, false, "", fmt.Errorf("解析断点账本失败: %w", err)
	}
	// 服务端没给 `fresh` 时按"新"处理：老服务端没有这个键，此时账本语义与以前一致。
	fresh := true
	if r.Fresh != nil {
		fresh = *r.Fresh
	}
	out := make([]bpView, 0, len(r.Breakpoints))
	for _, b := range r.Breakpoints {
		out = append(out, bpView{
			Num: b.Num, Kind: bpKindName(b.Kind), File: b.File, Line: b.Line, Func: b.Func,
			Module: b.Module, Enabled: b.Enabled, Cond: b.Cond, Ignore: b.Ignore,
			Hits: b.Hits, Note: b.Note,
		})
	}
	return out, fresh, r.Note, nil
}

// bpAdd 下一条断点（条件内联在 break 里、忽略次数走 ignore，口径见内核的 BreakpointOpts）。
func bpAdd(id string, o debug.BreakpointOpts) error {
	path := dbgSessionURL(id, "/breakpoints")
	if path == "" {
		return fmt.Errorf("没有可用的会话句柄（先 tt debug start 或 tt debug status 看看有没有会话）")
	}
	body := map[string]any{"location": o.Location}
	if o.Cond != "" {
		body["cond"] = o.Cond
	}
	if o.Ignore > 0 {
		body["ignore"] = o.Ignore
	}
	if o.Kind != "" {
		body["kind"] = o.Kind
	}
	data, err := dbgAPI("POST", path, body)
	if err != nil {
		return withRunningHint(err)
	}
	if IsJSON() {
		var raw map[string]any
		if err := jsonUnmarshal(data, &raw); err == nil {
			return dbgJSONOut(raw)
		}
		return nil
	}
	var r struct {
		Breakpoint debug.Breakpoint `json:"breakpoint"`
	}
	if err := jsonUnmarshal(data, &r); err != nil {
		return fmt.Errorf("解析下断回包失败: %w", err)
	}
	b := r.Breakpoint
	fmt.Println("断点 #" + strconv.Itoa(b.Num) + " 已下：" + bpLocationText(bpView{
		Kind: bpKindName(b.Kind), File: b.File, Line: b.Line, Func: b.Func,
	}))
	if b.Note != "" {
		fmt.Println("  " + b.Note)
	}
	return nil
}

// bpDel 删一条断点。
func bpDel(id string, num int) error {
	path := dbgSessionURL(id, "/breakpoints/"+strconv.Itoa(num))
	if path == "" {
		return fmt.Errorf("没有可用的会话句柄（先 tt debug start 或 tt debug status 看看有没有会话）")
	}
	if _, err := dbgAPI("DELETE", path, nil); err != nil {
		return withRunningHint(err)
	}
	fmt.Printf("断点 #%d 已删除\n", num)
	return nil
}

// bpKindName 把内核的 Kind 归一成给人看的词（空串 = break）。
func bpKindName(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return debug.BPKindBreak
	}
	return kind
}

// bpLocationText 一条断点的位置：监视点没有 file:line，位置就是被监视的表达式。
func bpLocationText(b bpView) string {
	if b.Kind == debug.BPKindWatch {
		if b.Func != "" {
			return b.Func
		}
		return "(未回显表达式)"
	}
	if b.File == "" && b.Line == 0 {
		if b.Func != "" {
			return b.Func
		}
		return "(位置未知)"
	}
	if b.Line == 0 {
		return b.File
	}
	return b.File + ":" + strconv.Itoa(b.Line)
}

// bpHitsText 命中计数的人话（A34：**必须**打出来，没有命中就是 0，不许留空）。
func bpHitsText(b bpView) string { return fmt.Sprintf("命中 %d 次", b.Hits) }

// bpListLine 一条断点的人读行。
func bpListLine(b bpView) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  #%-3d %-9s %s", b.Num, b.Kind, bpLocationText(b)))
	if b.Module != "" && b.Kind != debug.BPKindWatch {
		sb.WriteString(" (" + b.Module + ")")
	}
	sb.WriteString("  " + bpHitsText(b))
	if b.Enabled {
		sb.WriteString("  启用")
	} else {
		sb.WriteString("  已禁用")
	}
	if expr := strings.TrimSpace(b.Cond); expr != "" {
		sb.WriteString("  条件 " + expr)
	}
	if b.Ignore > 0 {
		sb.WriteString("  还忽略 " + strconv.Itoa(b.Ignore) + " 次")
	}
	if n := strings.TrimSpace(b.Note); n != "" {
		sb.WriteString("  " + n)
	}
	return sb.String()
}

// printBPList 打印账本（人读）。空表也要说话：说"没有断点"，并提一句下断的时机。
func printBPList(id string, bps []bpView) {
	if len(bps) == 0 {
		fmt.Printf("断点 0 个（会话 %s）\n", id)
		fmt.Println("  （没有断点）下断要在**停站态**：running 时服务端会拒（仅允许 interrupt）")
		return
	}
	fmt.Printf("断点 %d 个（会话 %s）\n", len(bps), id)
	for _, b := range bps {
		fmt.Println(bpListLine(b))
	}
}

// debugBPCmd 断点面。
var debugBPCmd = &cobra.Command{
	Use:   "bp",
	Short: "断点面：list（带命中计数）/ add / del",
	Long: `断点面。与 ` + "`tt debug exec \"info breakpoints\"`" + ` 的区别是**结构化**：
命中计数、条件、忽略次数都读成字段，不再让人对着 fgldb 原文猜。

  tt debug bp list                 列表（每条都带「命中 N 次」）
  tt debug bp list --json          同上，含 hits 字段（没命中的是 0，不是缺键）
  tt debug bp add axm_axmt500_wf.4gl:4130
  tt debug bp add 4130 --cond "1=1" --ignore 2
  tt debug bp add g_current_row --kind watch
  tt debug bp del 3

两条现场纪律（契约 A33 / A34）：
  · **命中计数才是判据**：` + "`Address`" + ` 恒 0x00000000（命中与否都是 0），别拿它判"装没装上"；
  · **下断要在停站态**：程序 running 时服务端直接 400「程序运行中,禁止发送调试命令(仅允许中断)」。
    这时先 ` + "`tt debug interrupt`" + `（running 时唯一允许的命令），停下后再下断。

已知边界（契约 A32）：` + "`ignore`" + ` 的**清零不受支持**（报文分不清"没设过"与"刚清零"）——
要改成"不忽略"请**删除后重建**（` + "`bp del` + `bp add`" + `），不要指望清零。`,
}

var debugBPListCmd = &cobra.Command{
	Use:   "list",
	Short: "列出断点（每条带命中计数 Hits）",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		bps, fresh, note, err := bpListFresh(id)
		if err != nil {
			return err
		}
		if IsJSON() {
			if bps == nil {
				bps = []bpView{}
			}
			out := map[string]any{"ok": true, "sessionId": id, "breakpoints": bps, "fresh": fresh}
			if !fresh && strings.TrimSpace(note) != "" {
				out["note"] = note
			}
			return dbgJSONOut(out)
		}
		printBPList(id, bps)
		// A36：运行态读不到内核 ⇒ 必须把"这份不是最新"写在脸上，不许让人以为 hits 是当下的。
		if !fresh && strings.TrimSpace(note) != "" {
			fmt.Println("  ！" + note)
		}
		return nil
	},
}

var debugBPAddCmd = &cobra.Command{
	Use:   "add <位置>",
	Short: "下一条断点（位置：行号 / 函数名 / file:line；--kind watch 时是表达式）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		opts := debug.BreakpointOpts{
			Location: args[0],
			Cond:     bpCondFlag,
			Ignore:   bpIgnoreFlag,
			Kind:     bpKindFlag,
		}
		return bpAdd(id, opts)
	},
}

var debugBPDelCmd = &cobra.Command{
	Use:   "del <编号>",
	Short: "删除一条断点（编号见 bp list）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := dbgSessionID()
		if err != nil {
			return err
		}
		num, err := strconv.Atoi(strings.TrimSpace(args[0]))
		if err != nil || num <= 0 {
			return fmt.Errorf("断点编号得是正整数（实得 %q）；编号见 tt debug bp list", args[0])
		}
		return bpDel(id, num)
	},
}

// jsonUnmarshal 解包服务端回包（本文件只有一处 JSON 依赖，集中在这里便于改）。
func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

func init() {
	debugBPAddCmd.Flags().StringVar(&bpCondFlag, "cond", "", "条件表达式（内联成 `break <loc> if <expr>`，A26）")
	debugBPAddCmd.Flags().IntVar(&bpIgnoreFlag, "ignore", 0, "忽略前 N 次命中（0 = 不忽略；不支持清零，要改就删了重建）")
	debugBPAddCmd.Flags().StringVar(&bpKindFlag, "kind", "", "断点种类：break（默认）/ watch（监视点，位置当表达式）")
	dbgRegisterSessionFlag(debugBPListCmd, debugBPAddCmd, debugBPDelCmd)
	debugBPCmd.AddCommand(debugBPListCmd, debugBPAddCmd, debugBPDelCmd)
	Group.AddCommand(debugBPCmd)
}

// 命令行开关（`tt debug bp add`）。
var (
	bpCondFlag   string
	bpIgnoreFlag int
	bpKindFlag   string
)
