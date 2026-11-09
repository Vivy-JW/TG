package target

// 结果信封与人话渲染（契约 5.1 / 5.2 / C5.1 / C5.6 / C5.7 / C5.9 / C5.10 / C5.11 / C5.14）。
//
// 断言一律落在**序列化出来的 JSON** 上，而不是只看常量表：
// 契约说的"恒定出现"指的是"打出来的信封里有这个键"（WP2 的契约测试用的是同一手法）。

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// TestSuccessEnvelopeShape 钉住 5.1 + C5.1 + C5.6 + C5.7：
// `ok` + `session`（恒定 11 键）+ `requested`（原样回显）+ `ui`（只有挂着界面会话时出现）。
func TestSuccessEnvelopeShape(t *testing.T) {
	// 取不到值的会话：Name / Ent / Epoch 都是空的 —— C5.1 明写"给空串，不省键"。
	env := NewSuccessEnvelope(Session{ID: "s1", Ref: "恒烁正式区#1", Env: "恒烁正式区", N: 1,
		Prog: "apmt520", State: StateIdle}, "恒烁正式区#1", map[string]any{"x": 1})
	raw := marshalMap(t, env)

	if raw["ok"] != true {
		t.Errorf("成功信封的 ok 该是 true（B7），实得 %v", raw["ok"])
	}
	for _, k := range SessionTopKeys {
		if _, ok := raw[k]; !ok {
			t.Errorf("5.1：成功信封恒定有 %q 键，实得 %v", k, mapKeys(raw))
		}
	}
	sec, ok := raw["session"].(map[string]any)
	if !ok {
		t.Fatalf("success 的 `session` 该是个对象，实得 %T", raw["session"])
	}
	for _, k := range SessionEnvelopeKeys {
		if _, present := sec[k]; !present {
			t.Errorf("C5.1：`session` 块恒定有 %q 键（取不到就给空串/零值，**不省键**），实得 %v",
				k, mapKeys(sec))
		}
	}
	// C5.6：requested.session 把调用方**原本写的**选择器原样回显。
	req, _ := raw["requested"].(map[string]any)
	if req == nil || req["session"] != "恒烁正式区#1" {
		t.Errorf("C5.6：requested.session 该原样回显 \"恒烁正式区#1\"，实得 %v", raw["requested"])
	}

	if _, present := raw["data"]; !present {
		t.Errorf("5.1：data 块该在（B7 的现行成功信封就有它）")
	}

}

// TestFailureEnvelopeKeysAreStable 钉住 C5.10 + C5.11：
// `ok` / `code` / `error` / `exitCode` 四键**恒定**；`hint` 有建议才出现；
// `candidates` 只在那五种码下出现 —— 而且**必须出现**（可能是空数组）。
func TestFailureEnvelopeKeysAreStable(t *testing.T) {
	f := newFailure(CodeUsage, "示例错")
	raw := marshalMap(t, f.Envelope())
	for _, k := range FailureRequiredKeys {
		if _, ok := raw[k]; !ok {
			t.Errorf("C5.10：失败信封恒定有 %q 键，实得 %v", k, mapKeys(raw))
		}
	}
	if raw["code"] != CodeUsage {
		t.Errorf("失败信封的 code = %v，期望 %v", raw["code"], CodeUsage)
	}
	if raw["error"] != "示例错" {
		t.Errorf("失败信封的 error = %v，期望 示例错", raw["error"])
	}
	if raw["exitCode"] != float64(ExitUsage) {
		t.Errorf("失败信封的 exitCode = %v，4.1.1 对 %s 写的是 %d", raw["exitCode"], CodeUsage, ExitUsage)
	}
	// hint 没给就**不出现**（C5.10：有修复建议才出现）。
	if _, ok := raw["hint"]; ok {
		t.Errorf("没给修复建议时不该有 hint 键（C5.10），实得 %v", mapKeys(raw))
	}
	// candidates 只在 C5.11 那五种码下出现；USAGE 不在其中。
	if _, ok := raw["candidates"]; ok {
		t.Errorf("code=%s（C5.11 不在必须带候选的五种码里）时不该有 candidates 键，实得 %v",
			CodeUsage, mapKeys(raw))
	}

	// 给了建议就有 hint。
	withHint := newFailure(CodeUsage, "示例错")
	withHint.Hint = "试试 --session"
	if _, ok := marshalMap(t, withHint.Envelope())["hint"]; !ok {
		t.Errorf("有修复建议时 hint 该出现（C5.10）")
	}

	// 五种码下 candidates **必须出现**：哪怕是空清单，键也要在（`[]`）。
	for _, code := range CandidatesRequiredCodes {
		e := newFailure(code, "x").Envelope()
		if e.Candidates == nil {
			t.Errorf("C5.11：%s 必须带候选清单（空也要出现，写成 []），Envelope().Candidates 是 nil", code)
		}
		raw := marshalMap(t, e)
		v, ok := raw["candidates"]
		if !ok {
			t.Errorf("C5.11：%s 的失败信封里**必须**有 candidates 键（omitempty 会把空切片一起丢掉，\n"+
				"    所以那个字段用指针 —— 见 envelope.go 的注释），实得 %v", code, mapKeys(raw))
			continue
		}
		if _, isList := v.([]any); !isList {
			t.Errorf("C5.11：%s 的 candidates 该是个数组，实得 %T", code, v)
		}
	}
}

// TestFailureEnvelopeCarriesCandidates 钉住 C5.11 + C5.12（有候选时逐键打出来）。
func TestFailureEnvelopeCarriesCandidates(t *testing.T) {
	f := newFailure(CodeAmbiguous, "有 2 条 SSH 会话，无法确定你要哪一条")
	f.Candidates = []Candidate{contractPool().Sessions[0].Candidate()}
	raw := marshalMap(t, f.Envelope())
	list, _ := raw["candidates"].([]any)
	if len(list) != 1 {
		t.Fatalf("candidates 该有 1 条，实得 %v", raw["candidates"])
	}
	one, _ := list[0].(map[string]any)
	for _, k := range CandidateKeys {
		if _, ok := one[k]; !ok {
			t.Errorf("C5.12：每个候选固定九键，缺 %q，实得 %v", k, mapKeys(one))
		}
	}
	if one["ref"] != "恒烁正式区#1" || one["n"] != float64(1) {
		t.Errorf("候选的身份字段 = %v", mapKeys(one))
	}
}

// TestCandidateLineFormat 钉住 C5.14 的人话候选行：
//
//	<ref>  <name>  <state>  ENT <ent>
//
// 两格缩进、字段间两空格；未命名写 `(未命名)`；第一个字段是 **ref**（恒定），不是 display。
func TestCandidateLineFormat(t *testing.T) {
	got := CandidateLine(Candidate{Ref: "恒烁正式区#1", Name: "汇率单", State: StateStopped, Ent: "99"})
	want := "  恒烁正式区#1  汇率单  已停站  ENT 99"
	if got != want {
		t.Errorf("候选行（C5.14）= %q\n    期望 %q", got, want)
	}
	if strings.Contains(got, "会话 ") {
		t.Errorf("候选行不该带 `会话 ` 前缀（C5.14 末段：那是 C5.9 身份行的前缀，两行不是同一行）")
	}
	// 未命名 / 取不到企业编号时的两个字面量。
	got2 := CandidateLine(Candidate{Ref: "恒烁正式区#2", State: StateIdle})
	if !strings.Contains(got2, "(未命名)") || !strings.Contains(got2, "ENT ?") {
		t.Errorf("未命名 / 取不到企业编号时要写字面量 (未命名) 与 ?（C5.14），实得 %q", got2)
	}
	// 第一个字段必须是 ref 的形态（环境#序号），不能是 display 的形态（含 `/`）。
	first := strings.Fields(strings.TrimSpace(want))[0]
	if strings.Contains(first, SelectorSep) {
		t.Errorf("候选行第一个字段 %q 含 %q —— 那是 display，会随会话当前作业漂（C5.14）",
			first, SelectorSep)
	}
	// 多条候选的渲染顺序稳定（按序号）。
	f := newFailure(CodeAmbiguous, "x")
	f.Candidates = []Candidate{
		{Ref: "恒烁正式区#2", N: 2, State: StateIdle},
		{Ref: "恒烁正式区#1", N: 1, State: StateIdle},
	}
	lines := f.CandidateLines()
	if len(lines) != 2 || !strings.Contains(lines[0], "#1") || !strings.Contains(lines[1], "#2") {
		t.Errorf("候选行该按序号稳定排序（同一个错误两次跑出的文案必须一样），实得 %v", lines)
	}
}

// TestIdentityLineFormat 钉住 C5.9：
//
//	会话 <display> · <name> · <state> · ENT <ent>
//
// ` · ` 是 空格 + U+00B7 + 空格；未命名写 `(未命名)`；ent 取不到写 `?`（**不省整段**）。
func TestIdentityLineFormat(t *testing.T) {
	got := IdentityLine(Session{
		Ref: "恒烁正式区#1", Display: "恒烁正式区/apmt520#1",
		Name: "汇率单", State: StateStopped, Ent: "99",
	})
	want := "会话 恒烁正式区/apmt520#1 · 汇率单 · 已停站 · ENT 99"
	if got != want {
		t.Errorf("身份行（C5.9）= %q\n    期望 %q", got, want)
	}
	// 身份行写的是 display（人读形态），不是 ref —— 对账一律用 ref（C5.9 的确切规则）。
	if !strings.Contains(got, "恒烁正式区/apmt520#1") {
		t.Errorf("C5.9：身份行写 `会话 ` + <display>，实得 %q", got)
	}
	got2 := IdentityLine(Session{Ref: "恒烁正式区#2", Display: "恒烁正式区/bsft001#2", State: StateIdle})
	if got2 != "会话 恒烁正式区/bsft001#2 · (未命名) · 空闲 · ENT ?" {
		t.Errorf("未命名 / 取不到企业编号的身份行 = %q\n    期望 %q（两个字段都不许省）",
			got2, "会话 恒烁正式区/bsft001#2 · (未命名) · 空闲 · ENT ?")
	}
	// 分隔符逐字：空格 + U+00B7 + 空格。
	if !strings.Contains(want, " \u00b7 ") {
		t.Fatalf("测试夹具里的分隔符写错了")
	}
	if strings.Count(got, " \u00b7 ") != 3 {
		t.Errorf("身份行的四个字段该用三个 ` · ` 分隔（C5.9），实得 %q", got)
	}
}

// TestStateLabel 钉住 C5.9 的 <state>：用**现行中文标签**（B30 的五个取值）。
//
// `internal/cli/debug/debugenv.go` 的 `dbgStateLabel` 是同一份值的生产实现 ——
// 今天两处各有一份（WP2 的契约测试也如实记过这处重复）。
// **WP4 接线时应当让 `dbgStateLabel` 引用本函数**，把那份重复收掉。
func TestStateLabel(t *testing.T) {
	for state, want := range map[string]string{
		StateIdle: "空闲", StateLoading: "连接中…", StateStopped: "已停站",
		StateRunning: "运行中", StateExit: "已断开",
	} {
		if got := StateLabel(state); got != want {
			t.Errorf("StateLabel(%q) = %q，期望 %q（B30 / C5.9）", state, got, want)
		}
	}
	// 认不出的取值按原样打（人话行不能因为一个没见过的状态就少一段）。
	if got := StateLabel("weird"); got != "weird" {
		t.Errorf("认不出的状态该原样打，实得 %q", got)
	}
	if got := StateLabel(""); got != "?" {
		t.Errorf("空状态该写字面量 ?（否则身份行会缺一段），实得 %q", got)
	}
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// marshalMap 把 v 序列化再解回 map（键的存在性只能在**打出来的 JSON** 上看）。
func marshalMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化失败：%v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("打出来的不是合法 JSON：%v\n%s", err, b)
	}
	return raw
}

// mapKeys 返回 map 的键（排序，失败信息用）。
func mapKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
