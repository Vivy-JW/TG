package target

// 审计行（计划 WP6 的验收项：每次会话操作有一条 —— 谁 / 哪条 / 干了什么；
// 契约 C5.4 明写审计行用 **ref**；C3.7c / Q20 要求跨轮追踪用别名 / 规范 id 而不是序号）。

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAuditEntryJSONLineCarriesWhoWhichWhat 钉住审计行的三要素与形状。
//
// 形状取的是本仓库既有审计的**同一族**（`internal/form42f` 的 `AppendAudit` 写一条 JSON 行，
// `internal/form42f/do.go:571-586`）：一行一条、字段名固定、时间 RFC3339。
func TestAuditEntryJSONLineCarriesWhoWhichWhat(t *testing.T) {
	at := time.Date(2026, 10, 16, 14, 30, 22, 0, time.UTC)
	e := NewAuditEntry("agent", "恒烁正式区#1", "quit", at)
	line, err := e.JSONLine()
	if err != nil {
		t.Fatalf("审计行序列化失败：%v", err)
	}
	want := `{"time":"2026-10-16T14:30:22Z","actor":"agent","ref":"恒烁正式区#1","action":"quit"}`
	if line != want {
		t.Errorf("审计行 = %s\n    期望 %s", line, want)
	}
	if strings.Contains(line, "\n") {
		t.Errorf("一条审计就是**一行**（append 到日志时不该多出换行）：%q", line)
	}
	// 三个字段的名字是判据的一部分：谁 / 哪条 / 干了什么。
	for _, k := range []string{`"actor"`, `"ref"`, `"action"`, `"time"`} {
		if !strings.Contains(line, k) {
			t.Errorf("审计行少了 %s 这个键（谁 / 哪条 / 干了什么 三要素）", k)
		}
	}
}

// TestAuditEntryFillsInTime 钉住"没给时间就补当前时间"（与 `internal/form42f.AppendAudit` 同一口径）：
// 审计行不许出现零值时间 —— 那会让"什么时候干的"这一问没有答案。
func TestAuditEntryFillsInTime(t *testing.T) {
	e := NewAuditEntry("agent", "恒烁正式区#1", "quit", time.Time{})
	line, err := e.JSONLine()
	if err != nil {
		t.Fatalf("审计行序列化失败：%v", err)
	}
	var got struct {
		Time string `json:"time"`
	}
	if err := json.Unmarshal([]byte(line), &got); err != nil {
		t.Fatalf("审计行不是合法 JSON：%v\n%s", err, line)
	}
	if got.Time == "" {
		t.Fatalf("审计行的时间是空的：%s", line)
	}
	if _, err := time.Parse(time.RFC3339, got.Time); err != nil {
		t.Errorf("审计行的时间 %q 不是 RFC3339：%v", got.Time, err)
	}
}

// TestAuditRefIsTheCanonicalID 钉住 C5.4 那一句"机器对账、错误文案、**审计行**用它"：
// 审计行里的"哪条"是规范 id（`环境#序号`，生命周期内恒定，C2.0），不是会漂的 display。
func TestAuditRefIsTheCanonicalID(t *testing.T) {
	s := contractPool().Sessions[0]
	e := NewAuditEntry("agent", s.CanonicalRef(), "stop", time.Time{})
	if e.Ref != "恒烁正式区#1" {
		t.Errorf("审计行的 ref = %q，期望规范 id 恒烁正式区#1（C2.0 / C5.4）", e.Ref)
	}
	if strings.Contains(e.Ref, SelectorSep) {
		t.Errorf("审计行的 ref = %q 含 %q —— 那是 display（作业段会随 SetRun 漂，C5.4）",
			e.Ref, SelectorSep)
	}
	// 人话形态只是给人看的，不参与对账。
	if got := e.HumanLine(); !strings.Contains(got, "谁 agent") ||
		!strings.Contains(got, "哪条 恒烁正式区#1") || !strings.Contains(got, "干了什么 stop") {
		t.Errorf("审计行的人话形态 = %q，该有『谁 / 哪条 / 干了什么』三段", got)
	}
}

// TestDefaultActor 钉住"谁"的缺省来源：环境变量 `TT_ACTOR`
// （与 `internal/debug/api.go:279` 的 `TT_ACTOR` 同一个口径，不另立一个开关）。
func TestDefaultActor(t *testing.T) {
	getenv := func(k string) string {
		if k == ActorEnv {
			return "agent"
		}
		return ""
	}
	if got := DefaultActor(getenv); got != "agent" {
		t.Errorf("DefaultActor = %q，期望 agent（读 %s）", got, ActorEnv)
	}
	if got := DefaultActor(func(string) string { return "  " }); got == "" || strings.TrimSpace(got) == "" {
		t.Errorf("取不到（或只有空白）时该给一个字面量占位，而不是空串 —— 否则审计行会缺一段，实得 %q", got)
	}
	if ActorEnv != "TT_ACTOR" {
		t.Errorf("环境变量名 = %q，internal/debug 用的那个是 TT_ACTOR", ActorEnv)
	}
}
