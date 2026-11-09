package debug

// A20（契约 .agents/plan/tt-genero-formdef-contract.md §5 A20）：
// 条件断点 / 忽略次数 / 监视点 —— 按 GDB 原样语义生成命令。
//
// **A26（§8）已取代 A20 的 `condition` 映射**：主机109 实测 fgldb 没有 `condition`
// 命令，条件改成写进 `break <loc> if <expr>`。本文件保留 A20 的结构性断言
//（字段校验 / 拒绝规则 / ignore 顺序 / 存档往返），命令逐字的断言见
// breakpointopts_a26_test.go。
//
// 本文件**只测命令生成与存档往返**：不连线、不建会话、不发任何真实 GDB 命令。
// 这是硬边界（红线）——所有断言都落在纯函数 BreakpointOpts.Commands 上，
// 它不碰 *Session，也不做 IO。

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBreakpointOptsCommandsTable 命令**逐字**与 A26 口径一致（A20 的 `condition`
// 映射已被 A26 取代），且顺序固定：先 break/watch（条件内联在 break 里），再 ignore。
func TestBreakpointOptsCommandsTable(t *testing.T) {
	cases := []struct {
		name string
		opts BreakpointOpts
		num  int
		want []string
	}{
		{
			name: "只有位置：一条 break，没有尾巴命令",
			opts: BreakpointOpts{Location: "prog.4gl:42"},
			num:  7,
			want: []string{"break prog.4gl:42"},
		},
		{
			name: "条件：写进 break 自己（inline if）",
			opts: BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1"},
			num:  7,
			want: []string{"break prog.4gl:42 if g_x > 1"},
		},
		{
			name: "忽略次数：ignore <n> <count>",
			opts: BreakpointOpts{Location: "prog.4gl:42", Ignore: 5},
			num:  7,
			want: []string{"break prog.4gl:42", "ignore 7 5"},
		},
		{
			name: "条件 + 忽略：break（带 if）在前，ignore 在后",
			opts: BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1", Ignore: 5},
			num:  7,
			want: []string{"break prog.4gl:42 if g_x > 1", "ignore 7 5"},
		},
		{
			name: "Kind=break 与空 Kind 同义",
			opts: BreakpointOpts{Location: "main", Kind: "break", Cond: "g_flag = 1"},
			num:  3,
			want: []string{"break main if g_flag = 1"},
		},
		{
			name: "监视点：watch <location>",
			opts: BreakpointOpts{Location: "g_total", Kind: "watch"},
			num:  2,
			want: []string{"watch g_total"},
		},
		{
			// A26：watch + Cond 无实测依据 ⇒ 明确拒绝，见 breakpointopts_a26_test.go。
			name: "监视点带忽略次数（不带条件）",
			opts: BreakpointOpts{Location: "rec.field", Kind: "watch", Ignore: 2},
			num:  2,
			want: []string{"watch rec.field", "ignore 2 2"},
		},
		{
			name: "断点号未知（num<0）：只发下断点那一条",
			opts: BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1", Ignore: 5},
			num:  -1,
			want: []string{"break prog.4gl:42 if g_x > 1"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.opts.Commands(c.num)
			if err != nil {
				t.Fatalf("不该报错，得 %v", err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("命令条数 %d，想 %d：%q", len(got), len(c.want), got)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Errorf("第 %d 条命令得 %q，想 %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestBreakpointOptsIgnoreZeroSendsNoCommand Ignore=0 = 「不忽略」，**不发 ignore 命令**，
// 但条件照旧内联在 break 里（别顺手把条件也吞了）。
func TestBreakpointOptsIgnoreZeroSendsNoCommand(t *testing.T) {
	opts := BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1", Ignore: 0}
	got, err := opts.Commands(4)
	if err != nil {
		t.Fatalf("不该报错：%v", err)
	}
	want := []string{"break prog.4gl:42 if g_x > 1"}
	if len(got) != len(want) {
		t.Fatalf("得 %q，想 %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("第 %d 条得 %q，想 %q", i, got[i], want[i])
		}
	}
	for _, c := range got {
		if strings.HasPrefix(c, "ignore ") {
			t.Errorf("Ignore=0 不该出现 ignore 命令：%q", got)
		}
	}
}

// TestBreakpointOptsRejects 非法输入：**各自报错**（说清哪个字段、为什么），
// 且**不产生任何命令**（返回 nil —— 调用方据此一个字节都不发）。
func TestBreakpointOptsRejects(t *testing.T) {
	cases := []struct {
		name    string
		opts    BreakpointOpts
		field   string // 错误里必须点名的字段
		keyword string // 错误里必须说清的理由
	}{
		{
			name:    "Cond 含换行（注入）",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1\ndelete"},
			field:   "Cond",
			keyword: "换行",
		},
		{
			name:    "Cond 含回车",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1\rshell"},
			field:   "Cond",
			keyword: "换行",
		},
		{
			name:    "Cond 以 end 结尾（命令块结束符）",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Cond: "g_x > 1 end"},
			field:   "Cond",
			keyword: "end",
		},
		{
			name:    "Cond 整体就是 END（大小写不敏感）",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Cond: "END"},
			field:   "Cond",
			keyword: "end",
		},
		{
			name:    "Cond 超 512 字节",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Cond: strings.Repeat("a", 513)},
			field:   "Cond",
			keyword: "512",
		},
		{
			name:    "Ignore 为负",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Ignore: -1},
			field:   "Ignore",
			keyword: "负",
		},
		{
			name:    "Kind 不认识",
			opts:    BreakpointOpts{Location: "prog.4gl:42", Kind: "hbreak"},
			field:   "Kind",
			keyword: "break",
		},
		{
			name:    "watch 的 location 含分号",
			opts:    BreakpointOpts{Location: "g_x; shell id", Kind: "watch"},
			field:   "Location",
			keyword: "白名单",
		},
		{
			name:    "watch 的 location 含换行",
			opts:    BreakpointOpts{Location: "g_x\ncontinue", Kind: "watch"},
			field:   "Location",
			keyword: "换行",
		},
		{
			name:    "watch 的 location 含反引号",
			opts:    BreakpointOpts{Location: "g_x`id`", Kind: "watch"},
			field:   "Location",
			keyword: "白名单",
		},
		{
			name:    "watch 的 location 是空白",
			opts:    BreakpointOpts{Location: "   ", Kind: "watch"},
			field:   "Location",
			keyword: "空",
		},
		{
			name:    "break 的 location 含换行",
			opts:    BreakpointOpts{Location: "prog.4gl:42\ndelete"},
			field:   "Location",
			keyword: "换行",
		},
		{
			name:    "location 整个为空",
			opts:    BreakpointOpts{Location: ""},
			field:   "Location",
			keyword: "空",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.opts.Commands(7)
			if err == nil {
				t.Fatalf("该拒绝，却给出命令 %q", got)
			}
			if got != nil {
				t.Errorf("拒绝时**一条命令都不许给**，得 %q", got)
			}
			if !strings.Contains(err.Error(), c.field) {
				t.Errorf("错误没点名字段 %q：%v", c.field, err)
			}
			if !strings.Contains(err.Error(), c.keyword) {
				t.Errorf("错误没说清理由（想看到 %q）：%v", c.keyword, err)
			}
			// 单条入口也必须拒（别只在 Commands 里挡，BreakWith 走的是 BreakCmd）。
			if cmd, err2 := c.opts.BreakCmd(); err2 == nil {
				t.Errorf("BreakCmd 也该拒绝，却给出 %q", cmd)
			}
			// 断点号未知时同样拒绝：校验在前，发命令在后。
			if got2, err3 := c.opts.Commands(-1); err3 == nil {
				t.Errorf("num=-1 也该拒绝，却给出 %q", got2)
			}
		})
	}
}

// TestBreakpointCondLengthBoundary 512 字节是上限本身（>512 才拒）。
func TestBreakpointCondLengthBoundary(t *testing.T) {
	ok := BreakpointOpts{Location: "prog.4gl:42", Cond: strings.Repeat("b", 512)}
	if _, err := ok.Commands(1); err != nil {
		t.Errorf("512 字节该放行：%v", err)
	}
	bad := BreakpointOpts{Location: "prog.4gl:42", Cond: strings.Repeat("b", 513)}
	if _, err := bad.Commands(1); err == nil {
		t.Error("513 字节该拒")
	}
}

// TestWatchLocationWhitelistPositives 白名单是**表达式**白名单，不是"只要没分号就行"：
// 常见的监视表达式必须照常放行（否则新功能一上线就不可用）。
func TestWatchLocationWhitelistPositives(t *testing.T) {
	for _, loc := range []string{
		"g_x", "rec.field", "arr[3]", "*p_ptr", "g_x + 1", "rec.a[2] + g_b", "p->f1",
	} {
		opts := BreakpointOpts{Location: loc, Kind: "watch"}
		got, err := opts.Commands(9)
		if err != nil {
			t.Errorf("%q 该放行，得 %v", loc, err)
			continue
		}
		if len(got) != 1 || got[0] != "watch "+loc {
			t.Errorf("%q 生成的命令不对：%q", loc, got)
		}
	}
}

// ---------- 存档往返（A20「兼容」半边） ----------

// TestBPStoreRoundTripWithOpts 新字段写入/读回一致。
func TestBPStoreRoundTripWithOpts(t *testing.T) {
	dir := t.TempDir()
	in := []StoredBP{
		{File: "prog.4gl", Line: 42, LineText: "LET g_x = 1", Func: "main", Enabled: true,
			Cond: "g_x > 1", Ignore: 3, Kind: "break"},
		{File: "prog.4gl", Line: 88, Func: "watched", Enabled: true,
			Kind: "watch"},
	}
	if err := saveBPs(dir, "t100", "mta", "adzi999", in); err != nil {
		t.Fatalf("写：%v", err)
	}
	st, err := loadBPs(dir, "t100", "mta", "adzi999")
	if err != nil {
		t.Fatalf("读：%v", err)
	}
	if len(st.Breakpoints) != 2 {
		t.Fatalf("断点数不对：%d", len(st.Breakpoints))
	}
	got := st.Breakpoints[0]
	if got.Cond != "g_x > 1" || got.Ignore != 3 || got.Kind != "break" {
		t.Errorf("新字段没带回来：%+v", got)
	}
	if st.Breakpoints[1].Kind != "watch" || st.Breakpoints[1].Cond != "" || st.Breakpoints[1].Ignore != 0 {
		t.Errorf("第二条新字段不对：%+v", st.Breakpoints[1])
	}
}

// TestBPStoreReadsOldArchiveWithoutOpts **旧存档（没有新字段）必须仍能读**：
// 缺字段 = 零值（Cond="" / Ignore=0 / Kind=""），不报错。
func TestBPStoreReadsOldArchiveWithoutOpts(t *testing.T) {
	dir := t.TempDir()
	p := bpStorePath(dir, "t100", "mta", "oldprog")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	old := `{"module":"mta","prog":"oldprog","savedAt":"2025-01-01T00:00:00Z",
"breakpoints":[{"file":"prog.4gl","line":42,"lineText":"LET g_x = 1","enabled":true}]}`
	if err := os.WriteFile(p, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := loadBPs(dir, "t100", "mta", "oldprog")
	if err != nil {
		t.Fatalf("旧存档该读得回来：%v", err)
	}
	if len(st.Breakpoints) != 1 {
		t.Fatalf("断点数不对：%d", len(st.Breakpoints))
	}
	b := st.Breakpoints[0]
	if b.Cond != "" || b.Ignore != 0 || b.Kind != "" {
		t.Errorf("旧存档的新字段该是零值：%+v", b)
	}
	if b.File != "prog.4gl" || b.Line != 42 || !b.Enabled {
		t.Errorf("旧字段不该被新字段挤坏：%+v", b)
	}
}

// TestBPStoreNewFieldsAreOmitEmpty 没设的新字段不落盘 ——
// 旧版本程序读到一堆 `"cond":""` 只会让人怀疑存档格式变了。
func TestBPStoreNewFieldsAreOmitEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "plain", []StoredBP{
		{File: "prog.4gl", Line: 7, LineText: "x", Enabled: true},
	}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(bpStorePath(dir, "t100", "mta", "plain"))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Breakpoints []map[string]any `json:"breakpoints"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	first := raw.Breakpoints[0]
	for _, k := range []string{"cond", "ignore", "kind"} {
		if _, ok := first[k]; ok {
			t.Errorf("零值的 %q 不该落盘：%v", k, first)
		}
	}
}
