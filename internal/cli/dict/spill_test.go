package dict

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/testkit"
)

// 这一组测的是 tt dict spill：那条"读已经落盘的那份完整结果"的入口。
//
// 它值得有自己的测试，因为它**不连数据源**（这是它的全部意义）——所以测试也不必连：
// 造一个配置目录 + 一份落盘文件就够，判据全是"读回来的那一页对不对"。

// TestSpillPage 翻页判据：offset/limit 的组合，以及"单实体没有页"。
//
// 与截断那一侧共用 payloadLen 的判据（slice 按元素数），所以这里顺带把"两边同一种数法"
// 钉住：截断按 5 条截、翻页按 5 条读，得到的页必须一样。
func TestSpillPage(t *testing.T) {
	payload := []any{"a", "b", "c", "d", "e"}
	cases := []struct {
		offset, limit int
		wantFrom      int
		wantTo        int
		want          []any
	}{
		{0, 0, 0, 5, []any{"a", "b", "c", "d", "e"}}, // limit 0 = 不限
		{0, 2, 0, 2, []any{"a", "b"}},
		{2, 2, 2, 4, []any{"c", "d"}},
		{4, 2, 4, 5, []any{"e"}}, // 尾巴上不足一页
		{5, 2, 5, 5, []any{}},    // 正好在末尾：空页而不是越界
		{0, 99, 0, 5, []any{"a", "b", "c", "d", "e"}},
	}
	for _, c := range cases {
		page, from, to := spillPage(payload, c.offset, c.limit)
		got, _ := page.([]any)
		if from != c.wantFrom || to != c.wantTo {
			t.Errorf("offset=%d limit=%d：区间 [%d,%d)，want [%d,%d)", c.offset, c.limit, from, to, c.wantFrom, c.wantTo)
		}
		if len(got) != len(c.want) {
			t.Errorf("offset=%d limit=%d：%d 条，want %d 条", c.offset, c.limit, len(got), len(c.want))
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("offset=%d limit=%d：第 %d 条是 %v，want %v", c.offset, c.limit, i, got[i], c.want[i])
			}
		}
	}

	// 单实体：没有页可言，原样返回、计 1 条。
	one := map[string]any{"name": "aapt300"}
	page, from, to := spillPage(one, 0, 10)
	if from != 0 || to != 1 {
		t.Errorf("单实体该是 [0,1)，得 [%d,%d)", from, to)
	}
	if m, ok := page.(map[string]any); !ok || m["name"] != "aapt300" {
		t.Errorf("单实体该原样返回，得 %#v", page)
	}
	if spillLen(one) != 1 || spillLen(payload) != 5 {
		t.Errorf("计数与 payloadLen 不一致：%d / %d", spillLen(one), spillLen(payload))
	}
}

// TestSpillRows 把载荷推成一张表：列名是各元素键的**并集**，**排序后**输出。
//
// 为什么必须排序：Go 里 map 的迭代顺序是随机的 —— 不排序，同一份文件两次读出来的列序
// 会不同，而"同样的输入给同样的输出"是这类工具的基本要求。
func TestSpillRows(t *testing.T) {
	payload := []any{
		map[string]any{"b": "2", "a": "1"},
		map[string]any{"c": "3", "a": "1"}, // 并集里要出现 c
	}
	cols, rows := spillRows(payload)
	if strings.Join(cols, ",") != "a,b,c" {
		t.Errorf("列名该是排序后的并集 a,b,c，得 %v", cols)
	}
	if len(rows) != 2 || strings.Join(rows[0], ",") != "1,2," || strings.Join(rows[1], ",") != "1,,3" {
		t.Errorf("行不对：%v", rows)
	}

	// 不是对象数组（字符串数组）：一列，值原样。
	cols, rows = spillRows([]any{"x", "y"})
	if len(cols) != 1 || len(rows) != 2 || rows[0][0] != "x" {
		t.Errorf("字符串数组该给一列，得 cols=%v rows=%v", cols, rows)
	}

	// 单元格：nil 给空串（不是 "<nil>"），嵌套结构走紧凑 JSON，整数不带 .0。
	cols, rows = spillRows([]any{map[string]any{"n": nil, "m": map[string]any{"k": "v"}, "i": float64(7)}})
	if len(cols) != 3 {
		t.Fatalf("该有 3 列，得 %v", cols)
	}
	line := strings.Join(rows[0], ",")
	if !strings.Contains(line, `{"k":"v"}`) {
		t.Errorf("嵌套该走紧凑 JSON：%v", rows[0])
	}
	if !strings.Contains(line, "7") || strings.Contains(line, "7.0") {
		t.Errorf("数字不该带 .0：%v", rows[0])
	}
	if !strings.HasSuffix(line, ",") {
		t.Errorf("nil 该是空串（列序排序后是 i,m,n，空的在最后一列 → 行尾一个逗号）：%v", rows[0])
	}
}

// TestSpillShowPagesAFile 端到端：造一份落盘文件，跑 `tt dict spill show … --offset --limit`，
// 从信封里读回这一页。
//
// 这一条**不连数据源**（那正是 spill 的意义），所以它靠一份临时配置目录 + 一个 JSON 文件
// 就能跑 —— 不需要库、不需要环境。命令走**真实的命令树**（Group → spill → show），
// 于是 `--limit` 挂在组上这件事也一并被验到。
func TestSpillShowPagesAFile(t *testing.T) {
	// 一份"配置"（只需要它的目录）：spill 目录就是 config.json 旁边的 spill/。
	dir := tempConfigDir(t)
	spill := filepath.Join(dir, spillDirName)
	if err := os.MkdirAll(spill, 0o755); err != nil {
		t.Fatal(err)
	}
	items := []any{}
	for i := 0; i < 7; i++ {
		items = append(items, map[string]any{"i": float64(i)})
	}
	raw, _ := json.Marshal(items)
	// 名字里带时间戳是 spillResult 的命名法；"latest" 要认得它（按 mtime 新的在前）。
	if err := os.WriteFile(filepath.Join(spill, "query-20260925-120000.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	newer, _ := json.Marshal([]any{map[string]any{"i": float64(99)}})
	if err := os.WriteFile(filepath.Join(spill, "query-20260925-130000.json"), newer, 0o644); err != nil {
		t.Fatal(err)
	}

	// ① list：两份，新的在前
	out := runDict(t, "spill", "list")
	if !strings.Contains(out, "query-20260925-130000.json") ||
		strings.Index(out, "130000") > strings.Index(out, "120000") {
		t.Errorf("list 该把新的排前面：\n%s", out)
	}

	// ② show <文件名> --offset 2 --limit 3 → i=2,3,4；信封里 totalRows=7 / returned=3 / offset=2
	out = runDict(t, "spill", "show", "query-20260925-120000.json", "--offset", "2", "--limit", "3")
	env := decodeEnvelope(t, out)
	if env["totalRows"] != float64(7) || env["returned"] != float64(3) || env["offset"] != float64(2) {
		t.Errorf("计数不对：totalRows=%v returned=%v offset=%v", env["totalRows"], env["returned"], env["offset"])
	}
	if env["truncated"] != true || env["source"] != "spill" {
		t.Errorf("该标成截断的一页、出处 spill：%v / %v", env["truncated"], env["source"])
	}
	data, _ := env["data"].([]any)
	if len(data) != 3 {
		t.Fatalf("该回 3 条，得 %d", len(data))
	}
	for k, want := range []float64{2, 3, 4} {
		row, _ := data[k].(map[string]any)
		if row["i"] != want {
			t.Errorf("第 %d 条该是 i=%v，得 %v", k, want, row["i"])
		}
	}
	if !strings.Contains(out, "--offset 5") {
		t.Errorf("这一页该给出下一页的入口：\n%s", out)
	}

	// ③ latest + --limit 0：取最新那份、全量
	out = runDict(t, "spill", "show", "latest", "--limit", "0")
	env = decodeEnvelope(t, out)
	if env["totalRows"] != float64(1) || env["returned"] != float64(1) {
		t.Errorf("latest 该是那份 1 条的文件：totalRows=%v returned=%v", env["totalRows"], env["returned"])
	}

	// ④ 目录里什么都没有时 list 不报错（"从没截断过"不是故障）
	if err := os.RemoveAll(spill); err != nil {
		t.Fatal(err)
	}
	out = runDict(t, "spill", "list")
	if !strings.Contains(out, "还没有落盘过") {
		t.Errorf("空目录该说清「还没落盘过」而不是报错：\n%s", out)
	}
}

// runDict 从**真实的命令树**跑一条 tt dict 子命令，把 stdout 收回来。
//
// stdout 捕获走 testkit.CaptureStdout（边写边读的管道实现）：从前这里另有一份"临时文件当
// stdout"的写法，是本仓库的**第三份** stdout 捕获实现，签名与另外两份都不一样。
// flag 重置同理，走 testkit.ResetFlags。
func runDict(t *testing.T, args ...string) string {
	t.Helper()
	var runErr error
	_, out := testkit.CaptureStdout(t, func() int {
		root := &cobra.Command{Use: "tt", SilenceUsage: true, SilenceErrors: true}
		root.AddCommand(Group)
		root.SetArgs(append([]string{"dict"}, args...))
		runErr = root.Execute()
		testkit.ResetFlags(Group) // 下一次调用要像全新进程
		return 0
	})
	if runErr != nil {
		t.Fatalf("命令 %v 失败：%v", args, runErr)
	}
	return out
}

// tempConfigDir 造一份临时配置（spill 目录就是 config.json 旁边的 spill/），返回那个目录。
//
// `common.ConfigPath` 是**包级 var**（命令层的单例），所以用完必须用 t.Cleanup 还回去。
// 不还的话，下一条用例会拿到这条留下的临时目录 —— 而 t.TempDir 早就把它删了，
// 表现是"目录找得到、里面的东西没了"这种难查的失败。从前两处都直接赋值、都不还。
func tempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := common.ConfigPath
	t.Cleanup(func() { common.ConfigPath = old })
	common.ConfigPath = filepath.Join(dir, "config.json")
	if err := os.WriteFile(common.ConfigPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestTruncationPointsAtIndexSpill 截断时那句 note 必须指向 spill 入口（而不是只给路径）。
//
// 这条把"截断 → 落盘 → 给出下一步"整条链接上：这一侧的代码路径（emitCapped）在没有库的机器上
// 跑不到真实查询，但它只要一个 payload 就能跑 —— 而**它写出的那个文件正是 spill show 要读的**，
// 所以顺带验了命名与目录的一致性。
func TestTruncationPointsAtIndexSpill(t *testing.T) {
	tempConfigDir(t) // 只要那个目录存在，spill 落在它旁边
	oldLimit := queryLimit
	queryLimit = 2 // 本次最多回 2 条 → 3 条必然被截
	// srcConfigPath 是落盘目录的依据，真机上由查询组的 PersistentPreRunE（loadQueryPolicy）
	// 设置 —— 这里直接调 emitCapped 绕过了那条链，所以自己补上（第一次跑就漏了它，
	// 落盘报"定位不到配置目录"，于是这句 note 说的是"未截断"）。
	srcConfigPath = common.ConfigPath
	defer func() { queryLimit = oldLimit; srcConfigPath = "" }()

	_, out := testkit.CaptureStdout(t, func() int {
		if err := emitCapped(true, []any{"a", "b", "c"}, nil, nil); err != nil {
			t.Fatalf("emitCapped 失败：%v", err)
		}
		return 0
	})
	env := decodeEnvelope(t, out)
	if env["truncated"] != true || env["returned"] != float64(2) || env["totalRows"] != float64(3) {
		t.Errorf("该标成截断且计数对：truncated=%v returned=%v totalRows=%v",
			env["truncated"], env["returned"], env["totalRows"])
	}
	notes, _ := env["notes"].([]any)
	joined := ""
	for _, n := range notes {
		joined += fmt.Sprint(n) + "\n"
	}
	if !strings.Contains(joined, "tt dict spill show") || !strings.Contains(joined, "--offset 2") {
		t.Errorf("note 该给出 spill 的翻页入口（offset 从已回的条数起）：\n%s", joined)
	}

	// 那份落盘文件真的能被 spill show 读到（命名/目录一致）。
	name, _ := env["localPath"].(string)
	if name == "" {
		t.Fatal("信封里没有 localPath")
	}
	out = runDict(t, "spill", "show", filepath.Base(name), "--limit", "1")
	env = decodeEnvelope(t, out)
	if env["totalRows"] != float64(3) {
		t.Errorf("spill show 读到的该是同一份（3 条），得 %v", env["totalRows"])
	}
}

// decodeEnvelope 把一次 --json 输出解成信封。
func decodeEnvelope(t *testing.T, out string) map[string]any {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("输出不是合法 JSON：%v\n%s", err, out)
	}
	return env
}
