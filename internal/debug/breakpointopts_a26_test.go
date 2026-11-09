package debug

// A26（契约 §8 A26，**取代** A20 的 `condition` 映射）：
// 本机 fgldb 没有 `condition` 命令（实测 `Undefined command: "condition"`），
// 条件只能写进 break 指令本身：`break <loc> if <expr>`。
//
// 反证（TestConditionCommandWouldGoRed）在这里的意义：如果有人把生成器改回
// `condition <n> <expr>`，这个文件的断言会立刻变红 —— 不是靠注释守规矩，是靠断言。

import (
	"strings"
	"testing"
)

// TestInlineIfCommandVerbatim Cond 非空 ⇒ 逐字 `break <loc> if <expr>`。
func TestInlineIfCommandVerbatim(t *testing.T) {
	cases := []struct {
		name string
		opts BreakpointOpts
		num  int
		want []string
	}{
		{
			name: "条件断点：条件写进 break 自己（一条命令，没有尾巴）",
			opts: BreakpointOpts{Location: "axm_axmt500_wf.4gl:2141", Cond: "g_current_sw = 1"},
			num:  2,
			want: []string{"break axm_axmt500_wf.4gl:2141 if g_current_sw = 1"},
		},
		{
			name: "条件 + 忽略：先 break（带 if），再 ignore",
			opts: BreakpointOpts{Location: "axm_axmt500_wf.4gl:2141", Cond: "g_current_sw = 1", Ignore: 2},
			num:  2,
			want: []string{"break axm_axmt500_wf.4gl:2141 if g_current_sw = 1", "ignore 2 2"},
		},
		{
			name: "只有忽略：先 break，再 ignore（顺序固定）",
			opts: BreakpointOpts{Location: "axm_axmt500_wf.4gl:5043", Ignore: 1},
			num:  3,
			want: []string{"break axm_axmt500_wf.4gl:5043", "ignore 3 1"},
		},
		{
			name: "函数名断点同样内联条件",
			opts: BreakpointOpts{Location: "axmt500_ui_dialog", Cond: "g_flag = 1"},
			num:  1,
			want: []string{"break axmt500_ui_dialog if g_flag = 1"},
		},
		{
			name: "断点号未知（num<0）：仍然只有那一条带 if 的 break",
			opts: BreakpointOpts{Location: "axm_axmt500_wf.4gl:2141", Cond: "g_current_sw = 1", Ignore: 2},
			num:  -1,
			want: []string{"break axm_axmt500_wf.4gl:2141 if g_current_sw = 1"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.opts.Commands(c.num)
			if err != nil {
				t.Fatalf("不该报错：%v", err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("命令条数 %d，想 %d：%q", len(got), len(c.want), got)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("第 %d 条 = %q，想 %q", i, got[i], c.want[i])
				}
			}
			for _, cmd := range got {
				if strings.Contains(strings.ToLower(cmd), "condition") {
					t.Errorf("本机 fgldb 没有 condition 命令，生成器不许出现它：%q", cmd)
				}
			}
		})
	}
}

// TestWatchWithCondRejected 监视点 + 条件 ⇒ **明确报不支持**，一条命令都不产生。
// 为什么不猜：`watch` 在 fgldb 上没有"带条件"的实测依据，硬塞 `watch X if Y`
// 只是把错误从"下断点失败"推迟到"命中时机不对"。
func TestWatchWithCondRejected(t *testing.T) {
	opts := BreakpointOpts{Location: "g_current_row", Kind: BPKindWatch, Cond: "g_current_row > 0"}
	for _, num := range []int{1, -1} {
		got, err := opts.Commands(num)
		if err == nil {
			t.Fatalf("num=%d 该拒绝，却给出 %q", num, got)
		}
		if got != nil {
			t.Errorf("拒绝时不许给命令：%q", got)
		}
		if !strings.Contains(err.Error(), "不支持条件监视点") {
			t.Errorf("错误文案该直说「不支持条件监视点」：%v", err)
		}
	}
	if cmd, err := opts.BreakCmd(); err == nil {
		t.Errorf("单条入口也该拒，得 %q", cmd)
	}
	// 监视点 + 忽略次数仍然可以（ignore 实测可用）。
	ok := BreakpointOpts{Location: "g_current_row", Kind: BPKindWatch, Ignore: 1}
	got, err := ok.Commands(1)
	if err != nil {
		t.Fatalf("监视点 + ignore 该放行：%v", err)
	}
	want := []string{"watch g_current_row", "ignore 1 1"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("得 %q，想 %q", got, want)
	}
}

// TestConditionCommandWouldGoRed 反证：把生成口径改回 `condition <n> <expr>` 会红。
//
// 这不是"模拟"，而是把**旧口径的输出**与**契约口径的输出**摆在一起比 ——
// 两者任何时刻相等，就说明生成器又回到了 `condition`。
func TestConditionCommandWouldGoRed(t *testing.T) {
	opts := BreakpointOpts{Location: "axm_axmt500_wf.4gl:2141", Cond: "g_current_sw = 1", Ignore: 2}
	got, err := opts.Commands(2)
	if err != nil {
		t.Fatalf("不该报错：%v", err)
	}
	// A20 的旧口径（主机109 上实测第二条就挂）：
	old := []string{"break axm_axmt500_wf.4gl:2141", "condition 2 g_current_sw = 1", "ignore 2 2"}
	if len(got) == len(old) {
		same := true
		for i := range old {
			if got[i] != old[i] {
				same = false
				break
			}
		}
		if same {
			t.Fatalf("生成器回到了 condition 口径 —— A26 回归：%q", got)
		}
	}
	if len(got) != 2 || got[0] != "break axm_axmt500_wf.4gl:2141 if g_current_sw = 1" || got[1] != "ignore 2 2" {
		t.Fatalf("A26 口径该是 [break … if …, ignore 2 2]，得 %q", got)
	}
}

// TestCondRejectsStillSendNoBytes A20 的三条拒绝规则在 A26 里**一条没松**：
// 换行 / 以 end 结尾 / >512 字节 ⇒ 拒，且 nil 切片（= 一个字节都不发）。
func TestCondRejectsStillSendNoBytes(t *testing.T) {
	bad := []BreakpointOpts{
		{Location: "prog.4gl:42", Cond: "g_x > 1\ndelete"},
		{Location: "prog.4gl:42", Cond: "g_x > 1\rshell"},
		{Location: "prog.4gl:42", Cond: "g_x > 1 end"},
		{Location: "prog.4gl:42", Cond: strings.Repeat("a", 513)},
	}
	for _, o := range bad {
		got, err := o.Commands(7)
		if err == nil {
			t.Errorf("%q 该拒，得 %q", o.Cond, got)
		}
		if got != nil {
			t.Errorf("拒绝时必须返回 nil 切片（发字节是结构上做不到的），得 %q", got)
		}
	}
}
