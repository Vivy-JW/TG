package debug

// WP4a 的两个新标志：`--session`（会话级目标）与 `--name`（起会话时的别名）。
//
// 为什么落点在这一个包：契约 1. 的**命令角色表**说的是"哪条命令认哪个标志"，
// 而命令树只在这里（`debug.Group`）。`internal/debug` 的契约测试想测它也不行 ——
// 那会是一个 import cycle（`internal/cli/debug` 本来就 import `internal/debug`），
// 所以 `internal/debug/contract_debug_test.go` 里留的是一行指路，断言本体在这儿。
//
// 三条断言逐条对着契约：
//
//  1. **会话级操作命令认 `--session`**（契约 1. 的命令角色表 / C3.1 / Q9）；
//  2. **会话外命令不认它**（C1.2 / C1.3 / C1.5）：status / sql / wslogs / wsdebug /
//     source / resolve / logs / ents / db / probe / serve —— 挂上去会让 F16 那条
//     "标志挂错命令"的自纠提示（`该命令不接受 --session`）落空；
//  3. **`--name` 只在起会话的命令上**（C3.5 / F17）：今天 = `start` 与 `wsdebug`
//     （Bug B：wsdebug 同样是起会话的命令，从前漏注册，于是它起的会话只能用 ref 寻址、
//     也不能当幂等键；`exec` / `quit` / `stop` 这些仍然不认它）。
//
// 另加两条"注册得上"的地基断言：标志名能 Lookup 到、且 help 文案非空 ——
// 后者是 `--help` 里能看见这条规矩的判据（C5.8/C2.5 那些取值纪律要写在文案里）。

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"tt/internal/target"
)

// TestCharTargetFlagPlacement 见文件头。
func TestCharTargetFlagPlacement(t *testing.T) {
	acceptSession := []string{
		"start", "exec", "quit", "why", "wait", "mode", "stop", "locate", "interrupt", "env", "topent",
	}
	rejectSession := []string{
		"status", "sql", "wslogs", "wsdebug", "source", "resolve", "logs", "ents", "db", "probe", "serve",
	}
	// `env` 也是**建会话**的命令（池空时它就是「创建会话」那条路），所以它认 --name：
	// 多会话要靠它把"我这条"建出来（`debug env --name <别名>` → 后续 topent/exec 才有点
	// 可点）。契约 C3.5 的"只在起会话的命令上"这条不变，变的是"起会话"的名单。
	acceptName := []string{"start", "wsdebug", "env"}
	rejectName := []string{
		"exec", "quit", "why", "wait", "mode", "stop", "locate", "interrupt", "topent",
		"status", "sql", "wslogs", "source", "resolve", "logs", "ents", "db", "probe", "serve",
	}

	// byName 把命令树摊成 名字 → 命令；顺带确认断言表里的名字都还在（改了命令名要一起改这里）。
	byName := map[string]bool{}
	for _, c := range Group.Commands() {
		byName[c.Name()] = true
	}
	hasFlag := func(cmdName, flag string) bool {
		for _, c := range Group.Commands() {
			if c.Name() == cmdName {
				return c.Flags().Lookup(flag) != nil
			}
		}
		return false
	}

	for _, c := range acceptSession {
		if !byName[c] {
			t.Errorf("断言表里的 `tt debug %s` 不在命令树上 —— 命令改名了就把这张表一起改", c)
			continue
		}
		if !hasFlag(c, "session") {
			t.Errorf("`tt debug %s` 该认 --session（契约 1. 的命令角色表：它是会话级操作）", c)
		}
	}
	for _, c := range rejectSession {
		if !byName[c] {
			t.Errorf("断言表里的 `tt debug %s` 不在命令树上 —— 命令改名了就把这张表一起改", c)
			continue
		}
		if hasFlag(c, "session") {
			t.Errorf("`tt debug %s` **不该**认 --session（C1.2/C1.3/C1.5：它的目标不是某一条会话）——\n"+
				"    挂上去会让 F16 那条「标志挂错命令」的自纠提示落空。", c)
		}
	}
	for _, c := range acceptName {
		if !hasFlag(c, "name") {
			t.Errorf("`tt debug %s` 该认 --name（C3.5：只在起会话的命令上有意义）", c)
		}
	}
	for _, c := range rejectName {
		if hasFlag(c, "name") {
			t.Errorf("`tt debug %s` 不该认 --name（F17：挂到非建会话命令上必须是「unknown flag」老路径）", c)
		}
	}

	// 地基：两个标志都真的注册上了，而且 help 文案非空 ——
	// 取值纪律（C2.4/C2.5 的"非空、1–64、只排除 / 与 #"）要能在 --help 里读到。
	for _, c := range Group.Commands() {
		for _, flag := range []string{"session", "name"} {
			f := c.Flags().Lookup(flag)
			if f == nil {
				continue
			}
			if f.Usage == "" {
				t.Errorf("`tt debug %s --%s` 的 help 文案是空的 —— 取值纪律要在 --help 里看得见", c.Name(), flag)
			}
		}
	}
}

// TestCharServerCodeMapsToExitAndEnvelope 钉住**服务端失败信封**在 CLI 侧的两件事
// （契约 C5.15 / 4.1.1 / Q15 的分档 / C5.11 的候选）：
//
//  1. `code` → **退出码**：服务端只给码与 HTTP 状态，退出码由这里按 4.1.1 分档 ——
//     F18（`SESSION_NAME_TAKEN`）**退 2 不退 3**、F20（`SESSION_TARGET_BUSY`）退 3、
//     多会话歧义（`SESSION_AMBIGUOUS`）退 3 **与有没有打标志无关**（F1 是 Agent
//     最需要机器可读码的那一刻：它恰恰是**没打** `--session` 时才撞上歧义）；
//     而今天的既有失败（0 条会话 / 参数错 / 运行失败）维持 1，只有真用了新写法才升档。
//  2. `candidates` 透传：服务端给的那一条候选要原样落到 CLI 的失败信封里
//     （C5.11 要求那五种码下这个键**必须出现**，而候选只有服务端知道该列谁）。
//
// 为什么落点在这个包：命令树与 `dbgAPI` 都在这儿（`internal/debug` 看不到它们）。
func TestCharServerCodeMapsToExitAndEnvelope(t *testing.T) {
	defer func(s, n string) { dbgSessionFlag, dbgNameFlag = s, n }(dbgSessionFlag, dbgNameFlag)

	type exitCoder interface{ ExitCode() int }
	type envelopeWriter interface{ WriteEnvelope(io.Writer) error }

	cases := []struct {
		name       string
		code       string
		newSyntax  bool // 命令行上有没有出现 `--session` / `--name`
		wantExit   int
		wantCands  int // -1 = 信封里不该有 candidates（C5.11）
		wantInText string
	}{
		{"F20 忙碌（新场景，不带标志也退 3）", target.CodeTargetBusy, false, 3, 1, "不能复用"},
		{"F20 忙碌（带 --name）", target.CodeTargetBusy, true, 3, 1, ""},
		{"F18 撞车（退 2 不退 3）", target.CodeNameTaken, false, 2, 1, ""},
		{"F1 歧义（不带标志也退 3）", target.CodeAmbiguous, false, 3, 1, ""},
		{"F15 过期（退 3）", target.CodeStale, true, 3, 1, ""},
		{"F2 没有会话（既有场景，不带标志维持 1）", target.CodeNoSession, false, 1, -1, ""},
		{"F2 没有会话（带 --session 才升到 3）", target.CodeNoSession, true, 3, -1, ""},
		{"F6 环境不存在（既有场景，不带标志维持 1）", target.CodeEnvUnknown, false, 1, -1, ""},
		{"参数错（既有场景，不带标志维持 1）", target.CodeUsage, false, 1, -1, ""},
		{"运行失败（表外的现行码一律 1）", "QUERY_FAILED", true, 1, -1, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dbgSessionFlag, dbgNameFlag = "", ""
			if c.newSyntax {
				dbgNameFlag = "某某别名"
			}
			cands := []target.Candidate{{
				ID: "s1", Ref: "恒烁正式区#1", Display: "恒烁正式区/apmt520#1",
				Name: "汇率单", Env: "恒烁正式区", Ent: "99", N: 1, Prog: "apmt520", State: "stopped",
			}}
			err := dbgServerErrorOf(c.code, "别名不能复用", cands)
			ec, ok := err.(exitCoder)
			if !ok {
				t.Fatalf("服务端失败该带退出码（根命令靠 ExitCode() 退码），实得 %T", err)
			}
			if got := ec.ExitCode(); got != c.wantExit {
				t.Errorf("code=%s（newSyntax=%v）的退出码 = %d，4.1.1 / Q15 分档要求 %d",
					c.code, c.newSyntax, got, c.wantExit)
			}
			jw, ok := err.(envelopeWriter)
			if !ok {
				t.Fatalf("服务端失败该自己会打信封（否则 code/candidates 会被根命令的形状吃掉），实得 %T", err)
			}
			var b strings.Builder
			if werr := jw.WriteEnvelope(&b); werr != nil {
				t.Fatalf("WriteEnvelope 失败：%v", werr)
			}
			var raw map[string]any
			if uerr := json.Unmarshal([]byte(b.String()), &raw); uerr != nil {
				t.Fatalf("信封不是合法 JSON：%v\n%s", uerr, b.String())
			}
			if raw["ok"] != false || raw["code"] != c.code {
				t.Errorf("信封该是 ok=false + code=%s，实得 %v", c.code, raw)
			}
			if raw["exitCode"] != float64(c.wantExit) {
				t.Errorf("信封里的 exitCode = %v，期望 %d", raw["exitCode"], c.wantExit)
			}
			gotCands, hasCands := raw["candidates"]
			if c.wantCands < 0 {
				if hasCands {
					t.Errorf("code=%s 不在 C5.11 那五种里，不该有 candidates，实得 %v", c.code, gotCands)
				}
				return
			}
			list, okList := gotCands.([]any)
			if !okList || len(list) != c.wantCands {
				t.Fatalf("candidates 该是 %d 条（服务端给的候选要原样透传），实得 %v", c.wantCands, gotCands)
			}
			// C5.12 的九键：CLI 侧不许手抄第二份形状，解包用的就是 target.Candidate。
			row, _ := list[0].(map[string]any)
			for _, k := range target.CandidateKeys {
				if _, ok := row[k]; !ok {
					t.Errorf("候选行少了 %q 键（C5.12 的九键），实得 %v", k, row)
				}
			}
		})
	}

	// 老服务端（信封里没有 code）时的行为：一句散文 + 退 1 —— 与今天逐字相同（H2）。
	if err := dbgServerErrorOf("", "老服务端说的错", nil); err != nil {
		if _, ok := err.(exitCoder); ok {
			t.Errorf("没有 code 的失败不该带退出码（那会让老服务端下的既有场景改变退出码）")
		}
		if !strings.Contains(err.Error(), "老服务端说的错") {
			t.Errorf("文案该原样透出，实得 %q", err.Error())
		}
	}
}
