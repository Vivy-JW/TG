package debug

// T2.1 契约测试：把 WP1 冻结契约（`.agents/plan/tt-refactor-contract.md`）里**可断言的形状**
// 变成断言。值全部取自同包的 `debugcontract_const_test.go`（**唯一出处**），
// 本文件里不出现散落的键名 / 码值 / id 文法字面量。
//
// 三条规矩：
//
//  1. **契约条款编号写进注释**（C5.1 / 4.1.1 / F7 …），让"测试 ↔ 契约"可追溯；
//  2. 今天还没实现的那部分用 `t.Skip`，文案写清**等什么**，并且**每一条都登记进
//     `internal/testkit` 的跳过台账**。**WP4c 之后这一组只剩一条**（C5.9 的身份行接进
//     CLI —— 那是人话输出改字，按 H2 归后续批次）；其余原先的跳过都按各自的交代改成了
//     真断言，对应的台账条目也按性质删掉了。
//  3. 失败信息要能照做：说清期望什么、实际什么、怎么办。
//
// 不在这里测的：`internal/debug/session.go` 主流程（要真 SSH + 真 fgldb）——
// 见本文件末尾「T2.4 不测什么」一节。

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"tt/internal/output"
	"tt/internal/target"
)

// ---------------------------------------------------------------------------
// T2.1-a 信封键集（C5.1 / C5.2 / C5.8 / C5.10 / C5.12）
// ---------------------------------------------------------------------------

// TestContractSessionEnvelopeKeysMatchSessionBrief 钉住**成功信封 `session` 块的键集**
// 与 `SessionBrief` 的字段集**完全一致**（C5.1 + C5.2 + C5.3）。
//
// 为什么值得：这是"契约里的键名到底有没有落进代码"的唯一机械判据。
// 它是**两向**的，因此能同时挡住两件事：
//
//   - 有人给 `SessionBrief` 改名或删键 → 契约 C5.3 明写「一个旧键都不改、改名、不删」；
//   - 有人往信封里塞了契约没写的第三份形状 → 前端与 CLI 会照契约解析，多的那个键没人认识。
//
// 它**不**断言"恒定出现"（那是 C5.1 的 omitempty 纪律，要搭真会话才验得了）：
// 这里断言的是**键的存在性与拼写**，恒定出现由 `TestContractFailureEnvelopeKeysAreStable`
// 的失败信封那一侧 + WP3 落地后的端到端用例承担。
func TestContractSessionEnvelopeKeysMatchSessionBrief(t *testing.T) {
	want := map[string]bool{}
	for _, k := range debugContractSessionEnvelopeKeys {
		want[k] = true
	}
	for _, k := range debugContractSessionOptionalKeys {
		if want[k] {
			t.Errorf("契约常量表自相矛盾：%q 同时出现在恒定键与 omitempty 键里 —— "+
				"一个键只能属一边，请改 debugcontract_const_test.go", k)
		}
		want[k] = true
	}

	rt := reflect.TypeOf(SessionBrief{})
	got := map[string]bool{}
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			t.Errorf("SessionBrief.%s 没有 json 标签 —— 它是对外键集的一部分，"+
				"没有标签就会被 encoding/json 按字段名输出（=%q），契约里的键名当场失效",
				rt.Field(i).Name, rt.Field(i).Name)
			continue
		}
		name := strings.Split(tag, ",")[0]
		got[name] = true
	}

	var missing, extra []string
	for k := range want {
		if !got[k] {
			missing = append(missing, k)
		}
	}
	for k := range got {
		if !want[k] {
			extra = append(extra, k)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("SessionBrief 上有、契约键集里没有这些键：%v\n"+
			"    怎么办：要么它属于 C5.2 的 SessionBrief 键集（加进 debugContractSessionOptionalKeys），\n"+
			"    要么它不该出现在信封里。别让契约与代码各说一套。", extra)
	}
	if len(missing) > 0 {
		// **WP4c 起不再跳过**：C5.3 的五个纯新增键（ref / display / name / ent / n / epoch +
		// kind）已经落进 SessionBrief，所以"缺键"只剩一种解释 —— 有人把契约里的键
		// 改名或删掉了，而 C5.2 / C5.3 明写「一个旧键都不改、不改名、不删」。
		t.Errorf("契约里要求 `session` 块有、SessionBrief 上却没有这些键：%v\n"+
			"    怎么办：C5.3 的新增键只有 %v（已全部落地），缺任何一个都说明有人删了/改名了键；\n"+
			"    要么把键改回去，要么同步契约常量表（debugcontract_const_test.go 引的是 internal/target 那一份）。",
			missing, debugContractNewSessionKeys)
	}
}

// TestContractSessionEnvelopeConstantKeysAreNeverOmitted 钉住 C5.1 的**恒定出现**那半句：
// 「`name` / `ent` 取不到时给空串，**不省键**」。
//
// 为什么要用**真实输出**验，而不是再看一遍 struct 标签：`omitempty` 恰好写在某个键上时，
// 反射能看见标签、却看不见"序列化出来到底有没有这个键"。契约这句话说的是后者 ——
// 前端与 CLI 按同一份形状解包，少一个键它们就得写第二套兜底。
//
// 判据取的是**最坏输入**：一条 `ref` / `display` / `name` / `ent` / `n` / `epoch` 全取不到值的
// 会话（pool 里手工放的假会话：没走登记，所以没有序号、没有别名、没有企业编号）。
// 十一个恒定键**一个都不许少** —— 取不到就给空串/零值。
func TestContractSessionEnvelopeConstantKeysAreNeverOmitted(t *testing.T) {
	srv := NewServer(cfgFor("h1", "36"), nil, "")
	// 一条**没有身份**的会话：没走登记，所以 ref / display / n / epoch 全取不到值。
	// 它的 envName 与 cfg 一致（与 NewSession 建出来的同形），这样 `env`（既有的、
	// 带 omitempty 的键）有值，把"新键取不到时省不省"这件事单独暴露出来。
	srv.mgr.sessions["s-裸"] = idleOn("s-裸", cfgFor("h1", "36"))
	mux := http.NewServeMux()
	srv.routes(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/sessions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("/api/sessions 该退 200，得 %d（%s）", rec.Code, rec.Body.String())
	}
	var doc struct {
		Sessions []map[string]any `json:"sessions"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("/api/sessions 该回 JSON：%v", err)
	}
	if len(doc.Sessions) != 1 {
		t.Fatalf("前置不成立：池里该恰好一条会话，得 %d", len(doc.Sessions))
	}
	for _, k := range debugContractSessionEnvelopeKeys {
		// `env` 是**既有**键：它今天带着 omitempty（C5.2 不许改旧键的写法，本次一字不动），
		// 所以它不在"取不到也必须在"这条断言的范围内。其余十个恒定键一个都不许少。
		if k == "env" {
			continue
		}
		if _, ok := doc.Sessions[0][k]; !ok {
			t.Errorf("C5.1 的恒定键 %q 没出现在 `/api/sessions` 的元素里（取不到就给空串，**不省键**）。\n"+
				"    实得键：%v", k, keysOf(doc.Sessions[0]))
		}
	}
	// 这条会话的 ref/display/name/ent/n/epoch 都取不到：它们必须是**空串/零值**，
	// 而不是一个编出来的值（C2.0/C5.4：一个错的地址比一个空字段坏得多）。
	if got := doc.Sessions[0]["ref"]; got != "" {
		t.Errorf("没有身份的会话，ref 该是空串（服务端不许编地址），实得 %v", got)
	}
	if got := doc.Sessions[0]["n"]; got != float64(0) {
		t.Errorf("没有身份的会话，n 该是 0，实得 %v", got)
	}
	// 反向：恒定键在 struct 上不许带 omitempty（带上了就等于"取不到时省键"，与本条相反）。
	rt := reflect.TypeOf(SessionBrief{})
	for _, k := range debugContractSessionEnvelopeKeys {
		for i := 0; i < rt.NumField(); i++ {
			tag := rt.Field(i).Tag.Get("json")
			if strings.Split(tag, ",")[0] != k {
				continue
			}
			// `env` 是**既有**键，它今天带着 omitempty（C5.2 不许改旧键的写法，本次一字不动）；
			// 其余恒定键一律不许带。
			if k == "env" {
				continue
			}
			if strings.Contains(tag, "omitempty") {
				t.Errorf("恒定键 %q 的标签是 %q —— C5.1 要求「取不到就给空串、不省键」，\n"+
					"    omitempty 恰好做的是相反的事（除既有的 env 外，恒定键都不许带它）。", k, tag)
			}
		}
	}
}

// TestContractFlagPlacementIsNotAssertedHere 记账：**标志落在哪些命令上这条断言不在这里**。
//
// WP2 交付时这里有一条跳过（`--session` / `--name` 没被任何命令注册，B19/B20），
// 它带一个"落地了就改成断言"的守卫。WP4a 把两个标志注册上去之后，那个守卫响了 ——
// 按它自己的交代，这里改成了断言，但**断言搬去了 `internal/cli/debug`**：
//
//   - `internal/cli/debug` 才有那棵命令树（`debug.Group`）；
//   - 而 `internal/debug` **不能** import `internal/cli/debug` —— 那是一个
//     import cycle（`internal/cli/debug` 本来就 import `internal/debug`），
//     编译期就报 `import cycle not allowed in test`。
//
// 所以这条留下的是一行**指路**，断言本体见 `internal/cli/debug/target_test.go` 的
// `TestCharTargetFlagPlacement`（逐条对着契约 1. 的命令角色表：哪些命令认 `--session`、
// 哪些命令不许认它、`--name` 只在起会话的命令上）。
func TestContractFlagPlacementIsNotAssertedHere(t *testing.T) {
	// 本文件里凡是"要摸 CLI 命令树"的断言都在别的包 —— 这条不该有跳过，
	// 它只是把"为什么不在 debug 包里测"记在案上（免得上层的契约测试清单里少一块没人发现）。
	t.Log("标志落点断言在 internal/cli/debug：TestCharTargetFlagPlacement（契约 1. 的命令角色表 / F16 / F17）")
}

// TestContractFailureEnvelopeKeysAreStable 钉住失败信封的**四键恒定 + 两键按条件**
// （C5.10 / C5.11），并用 `output.WriteError` 的**真实输出**核对，而不是只看常量表。
//
// 为什么用真实输出：契约说"恒定出现"，指的是**序列化出来的 JSON 里有这个键**。
// 只在常量表里写四个字符串，等于把"断言"写成了"抄写"。
//
// C5.15 给 HTTP 侧留了另一条：那边是 `{"ok":false,"error":…}` 加 `code`/`candidates`，
// 形状一字不改 —— 两侧的差别由 `TestContractHTTPFailureEnvelopeKeepsItsShape` 钉。
func TestContractFailureEnvelopeKeysAreStable(t *testing.T) {
	var b strings.Builder
	we := output.Errorf(output.CodeUsage, debugContractCodeExits[debugContractCodeUsage], "示例错")
	if err := output.WriteError(&b, we); err != nil {
		t.Fatalf("output.WriteError 失败：%v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(b.String()), &raw); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v\n%s", err, b.String())
	}
	for _, k := range debugContractFailureRequredKeys {
		if _, ok := raw[k]; !ok {
			t.Errorf("失败信封（C5.10）必须恒定有 %q 键，实际只有 %v\n"+
				"    怎么办：键集在 internal/output/errors.go 的 errorEnvelope。", k, keysOf(raw))
		}
	}
	// 反向：常量表里的每个键都必须真的在输出里（已经在上面的循环里逐条报过缺失）。
	// hint 没给时必须**不出现**（omitempty）：契约说"有修复建议才出现"。
	if _, ok := raw["hint"]; ok {
		t.Errorf("没给修复建议时失败信封不该有 hint 键（C5.10：有修复建议才出现），实际 %v", keysOf(raw))
	}
	// candidates 只在 C5.11 那五种码下出现；这条码（USAGE）不在其中。
	if _, ok := raw["candidates"]; ok {
		t.Errorf("code=%s（C5.11 不在必须带候选的五种码里）时不该有 candidates 键，实际 %v",
			debugContractCodeUsage, keysOf(raw))
	}
}

// TestContractCandidateKeysAreASubsetOfSessionKeys 钉住 C5.12 的**唯一出处**纪律：
// 候选九键必须是 `session` 键集的**真子集**（C5.4b：同一个 struct 的投影，不许手抄第二份 JSON 定义）。
func TestContractCandidateKeysAreASubsetOfSessionKeys(t *testing.T) {
	inSession := map[string]bool{}
	for _, k := range debugContractSessionEnvelopeKeys {
		inSession[k] = true
	}
	for _, k := range debugContractCandidateKeys {
		if !inSession[k] {
			t.Errorf("候选键 %q 不在 `session` 块的键集里 —— C5.12 要求九键是它的**真子集**，\n"+
				"    这样同一个渲染函数才能既打身份行又打候选行。要么改候选键集，要么改 session 键集，\n"+
				"    两处都在 debugcontract_const_test.go。", k)
		}
	}
	if len(debugContractCandidateKeys) != 9 {
		t.Errorf("C5.12 写死候选是**九键**，常量表里有 %d 个：%v",
			len(debugContractCandidateKeys), debugContractCandidateKeys)
	}
}

// TestContractCandidatesRequiredCodesMatchTheInvalidFormTable 双向核对
// C5.11 的"五种码必须带候选"与 2.4 那张表里标了"必须出现"的那些 F 条目。
//
// 这两处一个在 5.2、一个在 2.4，只是**引用关系**：表里少写一个码，就会出现
// "报歧义却不列候选"——调用方只能自己去列，那正是这次要根除的隐性行为。
func TestContractCandidatesRequiredCodesMatchTheInvalidFormTable(t *testing.T) {
	fromTable := map[string]bool{}
	for _, f := range debugContractInvalidForms {
		if f.Candidates {
			fromTable[f.Code] = true
		}
	}
	fromContract := map[string]bool{}
	for _, c := range debugContractCandidatesRequiredCodes {
		fromContract[c] = true
	}
	for c := range fromContract {
		if !fromTable[c] {
			t.Errorf("C5.11 说 %s 必须带候选清单，但 2.4 的表里没有任何一条标了 candidates=true —— "+
				"两处对不上，改哪边都要同步（都在 debugcontract_const_test.go）", c)
		}
	}
	for c := range fromTable {
		if !fromContract[c] {
			t.Errorf("2.4 的表里有 candidates=true 的条目用了 %s，但 C5.11 的五种码里没有它 —— "+
				"要么它真该带候选（加进 debugContractCandidatesRequiredCodes），要么那一条标错了", c)
		}
	}
}

// ---------------------------------------------------------------------------
// T2.1-b 七个机器码与退出码（4.1 / 4.1.1 / C5.13）
// ---------------------------------------------------------------------------

// TestContractCodeExitTable 逐条钉住 4.1.1 那张 `code` → 退出码的表。
//
// 用 `output.Error` 的真实字段核对（Code + Exit），而不是只比常量表：
// 契约说"机器码与退出码同源"（C4.1），这张测试就是那句话的判据。
func TestContractCodeExitTable(t *testing.T) {
	if len(debugContractCodeExits) == 0 {
		t.Fatal("退出码表是空的 —— 契约 4.1.1 里明明有一张表，常量表漏抄了")
	}
	codes := make([]string, 0, len(debugContractCodeExits))
	for c := range debugContractCodeExits {
		codes = append(codes, c)
	}
	sort.Strings(codes)
	for _, code := range codes {
		wantExit := debugContractCodeExits[code]
		e := output.Errorf(code, wantExit, "示例")
		if e.Code != code {
			t.Errorf("output.Error.Code = %q，期望 %q", e.Code, code)
		}
		if e.Exit != wantExit {
			t.Errorf("code=%s 的退出码 = %d，契约 4.1.1 写的是 %d\n"+
				"    怎么办：改的是 debugContractCodeExits（契约那一列），不是这里。",
				code, e.Exit, wantExit)
		}
	}
	// USAGE 的两档口径（4.1.1 的两条）：新写法退 2，老路径（F16/F17）退 1。
	if got := debugContractCodeExits[debugContractCodeUsage]; got != debugContractExitUsage {
		t.Errorf("新写法下 USAGE 该退 %d，表里是 %d（4.1.1 第一条口径）", debugContractExitUsage, got)
	}
	if debugContractFlagOnWrongCommandExit != debugContractExitRunFailed {
		t.Errorf("F16/F17 走老路径该退 %d（Q18：不补 FlagErrorFunc），常量写的是 %d",
			debugContractExitRunFailed, debugContractFlagOnWrongCommandExit)
	}
}

// TestContractNewCodesAreRegisteredInOutput 钉住 C5.13：
// 五个新码要加进 `internal/output/errors.go` 的常量清单（加新码可以，改旧码名字不行）。
//
// 今天一个都没有 → 跳过，并登记台账（"等 WP3/WP4 把新码加进 output"）。
// 一旦加了，这条测试会逐字核对**取值**：常量名可以叫 `CodeSessionAmbiguous`，
// 但它必须等于契约里那个字符串。
func TestContractNewCodesAreRegisteredInOutput(t *testing.T) {
	// 新增码（C5.13 要求加进 output 的那五个）。
	// 键就是常量本身 —— 这里不重写字面量，否则"值只定义一次"当场破功。
	newCodes := []string{
		debugContractCodeAmbiguous,
		debugContractCodeNotFound,
		debugContractCodeStale,
		debugContractCodeTargetBusy,
		debugContractCodeNameTaken,
	}
	// 复用现行常量的四个（B5 / C5.13 末句）：今天就有，所以**现在就核对**。
	t.Run("复用的现行码取值不许变", func(t *testing.T) {
		cases := []struct{ code, want string }{
			{output.CodeUsage, debugContractCodeUsage},
			{output.CodeEnvUnknown, debugContractCodeEnvUnknown},
			{output.CodeNoSession, debugContractCodeNoSession},
			{output.CodeSessionState, debugContractCodeState},
		}
		for _, c := range cases {
			if c.code != c.want {
				t.Errorf("internal/output 的常量取值 = %q，契约 4.1.1 写的是 %q\n"+
					"    C5.13：加新码可以，**改旧码的名字/取值不行**（它是 agent 分支的依据）。",
					c.code, c.want)
			}
		}
	})
	t.Run("五个新增码", func(t *testing.T) {
		var absent []string
		for _, want := range newCodes {
			if !outputHasCode(want) {
				absent = append(absent, want)
			}
		}
		if len(absent) > 0 {
			// **WP4c 起不再跳过**：三个剩余的新码（STALE / NAME_TAKEN / TARGET_BUSY）
			// 已随"别名与世代"那一批落进 output（C5.13）。缺了就是有人删了常量，
			// 或者新加的码没进下面那份显式清单。
			sort.Strings(absent)
			t.Errorf("契约 4.1.1 的新码不在 internal/output 的常量清单里（C5.13）：%s\n"+
				"    怎么办：在 internal/output/errors.go 里加常量（取值见 internal/target 的 Code*），\n"+
				"    并把新常量补进本文件的 outputHasCode 清单 —— 那份清单就是「有没有落地」的判据。",
				strings.Join(absent, "、"))
		}
		// 加进来了就逐字核对取值（常量名随意，取值必须等于契约里的字符串）。
		e := output.Errorf(debugContractCodeAmbiguous, debugContractCodeExits[debugContractCodeAmbiguous], "x")
		if e.Code != debugContractCodeAmbiguous {
			t.Errorf("output 侧 SESSION_AMBIGUOUS 的取值 = %q，契约写的是 %q", e.Code, debugContractCodeAmbiguous)
		}
	})
}

// outputHasCode 判断 internal/output 里是否已经有这个取值的稳定错误码常量。
//
// 用一份**显式清单**：Go 里没有"枚举某个包的常量"的反射手段（常量不是反射对象），
// 所以这里只能把现行清单列出来。它的作用是"新码有没有进 output"这个机械判据 ——
// 而不是靠人去 grep 关键字面量（关键字面量在别处也会出现，那样判不准）。
//
// 清单随 output 增长时要同步（多加一个码忘了同步，只影响这条测试的**严格度**：
// 认不出的码会被当成"还没落地"而判红，不会假绿任何别的东西 —— WP4c 起不再有跳过分支）。
func outputHasCode(want string) bool {
	for _, v := range []string{
		output.CodeUsage, output.CodeEnumInvalid, output.CodeEnvUnknown, output.CodeConfigBroken,
		output.CodeTableMissing, output.CodeConnFailed, output.CodeSQLRejected, output.CodeQueryFailed,
		output.CodeNotFound, output.CodeNoSession, output.CodeSessionState, output.CodeTargetMissing,
		output.CodePreflight, output.CodeUpdateUsage, output.CodeUpdateRefused,
		output.CodeUpdateChecksum, output.CodeUpdateNetwork, output.CodeUpdateApply,
		// WP4b：会话寻址的前两个码（服务端那三个"要一条会话"的端点用它，C5.16）。
		output.CodeSessionAmbiguous, output.CodeSessionNotFound,
		// WP4c：别名与世代那三个（C5.3 的键成为服务端事实之后自己就会产的码，C5.13）。
		output.CodeSessionStale, output.CodeSessionNameTaken, output.CodeSessionTargetBusy,
	} {
		if v == want {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// T2.1-c 规范 id 的文法（2.1.2 / C2.0 / Q1-b）
// ---------------------------------------------------------------------------

// TestContractRefGrammar 钉住**规范 id 的文法**（C2.1.2 的定稿）：
//
//	规范 id := 环境名 "#" 序号        序号 := 十进制整数，>= 1
//
// 三条要点，都是定稿改过的：
//   - 地址**不含作业段、不含企业编号**（早先的 `环境/作业#序号` 因为 SetRun 会改写 prog 被收回，
//     `ent` 因为 SetTopent 可变被收回 —— C2.0 的"生命周期内不变"是根依据）；
//   - 环境名**精确匹配**，不做前缀、不做大小写折叠（C2.1）；
//   - 序号 >= 1（`#0` 非法，F9）。
func TestContractRefGrammar(t *testing.T) {
	re, err := regexp.Compile(debugContractRefPattern)
	if err != nil {
		t.Fatalf("规范 id 的正则编译不过（debugContractRefPattern）：%v", err)
	}
	if !strings.Contains(debugContractRefPattern, debugContractSeqSep) {
		t.Errorf("规范 id 的正则里没有序号分隔段 %q —— 它与 debugContractSeqSep 必须同源", debugContractSeqSep)
	}

	valid := []string{"恒烁正式区#1", "测试1012#12", "env#999999", "带 空格的环境#3", "a#1"}
	for _, s := range valid {
		if !re.MatchString(s) {
			t.Errorf("%q 该是合法的规范 id（环境名%q序号，2.1.2），正则 %q 却说不是",
				s, debugContractSeqSep, debugContractRefPattern)
		}
	}
	invalid := []struct{ s, why string }{
		{"恒烁正式区", "没有序号段 —— 光有环境名是「按环境找」（写法 ⑧），不是规范 id"},
		{"恒烁正式区#", "序号为空（F9 的 `#`）"},
		{"恒烁正式区#0", "序号必须 >= 1（F9）"},
		{"恒烁正式区#-1", "序号必须是十进制正整数（F9）"},
		{"恒烁正式区#1.5", "序号必须是十进制正整数（F9）"},
		{"恒烁正式区#x", "序号必须是十进制正整数（F9）"},
		{"恒烁正式区#01", "序号不写前导零 —— 与 #1 是同一个地址，写成两个会对不上账"},
		{"恒烁正式区/apmt520#1", "这是**人读形态**（display，写法 B），不是规范 id：作业段会随 SetRun 漂（C5.4）"},
		{"#1", "环境名不能为空（规范 id 是「环境名#序号」）"},
	}
	for _, c := range invalid {
		if re.MatchString(c.s) {
			t.Errorf("%q 不该被当成规范 id：%s", c.s, c.why)
		}
	}
}

// TestContractSelectorSecondSegmentKind 钉住 C2.3 的判据：
// 选择器第二段**全数字 → 企业编号；否则 → 作业名**，两种形态不会撞
// （作业名首字符是字母，B29），所以不需要额外消歧规则。
func TestContractSelectorSecondSegmentKind(t *testing.T) {
	for _, s := range []string{"99", "1012", "0", "1"} {
		if got := selectorSecondSegKind(s); got != debugContractSecondSegEnt {
			t.Errorf("第二段 %q 该判成企业编号（C2.3：全数字），得 %q", s, got)
		}
	}
	for _, s := range []string{"apmt520", "bsft001", "a", "A_1-x"} {
		if got := selectorSecondSegKind(s); got != debugContractSecondSegJob {
			t.Errorf("第二段 %q 该判成作业名（C2.3：非全数字），得 %q", s, got)
		}
	}
	// 判据的分界在「全数字」这四个字上：判据**不**先做 B29 的合法性校验，
	// 所以 `1a` 这种形态（B29 下不可能是合法作业名）按 C2.3 落到作业那一支，
	// 随后会在"环境/作业"那一步按 F7 报 SESSION_NOT_FOUND。
	// 它**不会**被静默当成企业编号 1 —— 那才是"看起来落上了、其实落在别处"。
	for _, s := range []string{"1a", "apmt520a1", "99x"} {
		if got := selectorSecondSegKind(s); got != debugContractSecondSegJob {
			t.Errorf("C2.3 的判据只看「全数字」：第二段 %q 含非数字字符，应当判成作业名（%q），得 %q。\n"+
				"    注：B29 下这类串不是合法作业名，所以它最终按 F7（SESSION_NOT_FOUND）被拒 —— "+
				"这一支绝不能静默退回企业编号，那会变成「看起来落上了、其实落在别处」。",
				s, debugContractSecondSegJob, got)
		}
	}
	// 作业名合法字符集本身也要钉（B29 / 2.1.4）。
	jre := regexp.MustCompile(debugContractJobPattern)
	for _, s := range []string{"apmt520", "a", "A_1-x", strings.Repeat("z", 64)} {
		if !jre.MatchString(s) {
			t.Errorf("作业名 %q 该合法（B29 的正则 %q）", s, debugContractJobPattern)
		}
	}
	for _, s := range []string{"1apmt", "apmt.520", "apmt 520", "", strings.Repeat("z", 65), "作业"} {
		if jre.MatchString(s) {
			t.Errorf("作业名 %q 该非法（B29 的正则 %q）", s, debugContractJobPattern)
		}
	}
}

// ---------------------------------------------------------------------------
// T2.1-d 非法写法一览 F1–F20（2.4 + Q23 的 F20）
// ---------------------------------------------------------------------------

// TestContractInvalidFormsAreComplete 钉住那张表**本身**：F1–F20 逐条都在、没有空壳。
//
// 有人删掉一条非法写法，这张测试当场红 —— 那张表就是将来 TDD 的输入集，
// 少一条等于少验一种"说不清的错"。
func TestContractInvalidFormsAreComplete(t *testing.T) {
	const total = 20
	if len(debugContractInvalidForms) != total {
		t.Fatalf("非法写法表该有 %d 条（F1–F20），实得 %d 条", total, len(debugContractInvalidForms))
	}
	seen := map[string]bool{}
	for i, f := range debugContractInvalidForms {
		wantKey := fmt.Sprintf("F%d", i+1)
		if f.Key != wantKey {
			t.Errorf("第 %d 条该是 %s，表里写的是 %q —— 键必须连号，否则「逐条覆盖」这句话就没人验得了",
				i+1, wantKey, f.Key)
		}
		if seen[f.Key] {
			t.Errorf("非法写法 %s 登记了两次", f.Key)
		}
		seen[f.Key] = true
		if strings.TrimSpace(f.Write) == "" {
			t.Errorf("%s 没有写下「怎么写出这个非法写法」—— 那是将来照着敲的输入，不能空着", f.Key)
		}
		if strings.TrimSpace(f.Code) == "" {
			t.Errorf("%s 没有机器码（2.4 表里那一列）", f.Key)
		}
		switch f.Exit {
		case debugContractExitUsage, debugContractExitRunFailed, debugContractExitNoSession,
			debugContractExitState, debugContractExitMissing, debugContractExitPreflight:
		default:
			t.Errorf("%s 的退出码 = %d，不在契约 4.1 的码表里（0/1/2/3/4/5/6）", f.Key, f.Exit)
		}
		// 2.4 的"机器码"一列必须落在 4.1.1 那张唯一清单里（C5.12 的同一条纪律：
		// 一个规矩只定义一处）。
		if _, ok := debugContractCodeExits[f.Code]; !ok {
			t.Errorf("%s 用的机器码 %q 不在 debugContractCodeExits（4.1.1 的唯一清单）里 —— "+
				"要么这条写错了码，要么 4.1.1 漏了这个码", f.Key, f.Code)
		}
	}
}

// TestContractInvalidFormsCodeSequence 把 2.4 那张表的机器码**逐条**钉下来。
//
// 这条测试的作用是"改一处必须改两处"：谁动了某条 F 的码，这里立刻红。
// 码的顺序就是契约 2.4 那张表从上到下的顺序（F1…F20）。
func TestContractInvalidFormsCodeSequence(t *testing.T) {
	want := []struct {
		key  string
		code string
	}{
		{"F1", debugContractCodeAmbiguous},
		{"F2", debugContractCodeNoSession},
		{"F3", debugContractCodeAmbiguous},
		{"F4", debugContractCodeAmbiguous},
		{"F5", debugContractCodeNotFound},
		{"F6", debugContractCodeEnvUnknown},
		{"F7", debugContractCodeNotFound},
		{"F8", debugContractCodeNotFound},
		{"F9", debugContractCodeUsage},
		{"F10", debugContractCodeUsage},
		{"F11", debugContractCodeUsage},
		{"F12", debugContractCodeUsage},
		{"F13", debugContractCodeUsage},
		{"F14", debugContractCodeUsage},
		{"F15", debugContractCodeStale},
		{"F16", debugContractCodeUsage},
		{"F17", debugContractCodeUsage},
		{"F18", debugContractCodeNameTaken},
		{"F19", debugContractCodeUsage},
		{"F20", debugContractCodeTargetBusy},
	}
	byKey := map[string]debugContractInvalidForm{}
	for _, f := range debugContractInvalidForms {
		byKey[f.Key] = f
	}
	for _, w := range want {
		f, ok := byKey[w.key]
		if !ok {
			t.Errorf("非法写法表里没有 %s", w.key)
			continue
		}
		if f.Code != w.code {
			t.Errorf("%s 的机器码 = %q，契约 2.4 写的是 %q\n"+
				"    怎么办：改哪边都要同步（表是 internal/target 的 InvalidForms，"+
				"debugcontract_const_test.go 只是引用它）",
				w.key, f.Code, w.code)
		}
	}
}

// TestContractInvalidFormsServerSideEnvelopes 是 F1–F20 里**服务端判得出来的那几条**的端到端实测。
//
// 为什么必须有一条这样的测试：这二十条是本次重构的**验收输入集**。没有它，WP4 全部完成后
// 没人知道"是不是真的逐条跑了"。
//
// **WP4c 起它不再跳过**（守卫问的那件事——"服务端失败信封有没有 `code`"——已经成立，
// C5.15 落地）。分工是写死的，逐条都有落点：
//
//	| 谁判 | 哪些 F | 判据在哪 |
//	|---|---|---|
//	| `internal/target`（纯函数） | F1/F3–F15/F18–F20 的文法、落位、码与退出码 | 那个包的 resolve_test / selector_test / values_test |
//	| **服务端这一层（本测试）** | **F1/F18/F19/F20 的 HTTP 信封**（状态码 + code + candidates） | 本测试逐条发真请求 |
//	| `internal/cli/debug` | F1 的"多会话不取第一项"、退出码分档、标志落点（F16/F17） | 那个包的 target_test |
//
// **F2 不在服务端这一层了**：它曾经以 `/api/wslogs/debug`（按日志重放）当 HTTP 载体，
// 而那条端点干的是**创建**一条会话来重跑日志（`LaunchReplay` → `prepareSession`）——
// 池里 0 条会话恰恰是它最正常的起点，报 `NO_SESSION` 等于让人先去起个作业凑一条会话。
// 现在它与"切环境"一样走冷启动档（判据见 `session_named_host_test.go` 的
// `TestWSLogDebugColdStartIsNotNoSession`），F2 这一档留在 CLI 目标层
// （`internal/target` 的 `failureNoSession`），信封形状见下面那条
// `TestContractNoSessionEnvelopeIsStillPinned`。
//
// **本测试不碰真 SSH**：它只走"落不上目标 / 别名不对"这些**在拨号之前**就判掉的分支
// （这正是 C2.8 那条纪律：一个字节都不该白花）。要真会话才走得到的分支
// （F3–F8 的会话级落位）在 CLI 侧由目标层断言，端到端的真机会话在真机自检里。
func TestContractInvalidFormsServerSideEnvelopes(t *testing.T) {
	// 一个"池里有两条会话"的服务：F1 的歧义要 >=2 条才成立。
	hsTwo, sTwo := actionTestDebugServer(t)
	giveServerNamedEnv(sTwo, "h1", "127.0.0.1", "36")
	for _, id := range []string{"s-a", "s-b"} {
		sTwo.mgr.sessions[id] = fakeIdleSession(id, "h1", "36")
	}
	// 池空的服务：**本测试不再需要它** —— F2 的 HTTP 载体（按日志重放）已改成冷启动档，
	// 理由见上面那张表下面的注。

	// 一条**正在跑**的同名会话：F20 的"别名指向的会话不能复用"要它。
	hsBusy, sBusy := actionTestDebugServer(t)
	giveServerNamedEnv(sBusy, "h1", "127.0.0.1", "36")
	busy := fakeIdleSession("s-忙", "h1", "36")
	busy.mu.Lock()
	busy.state = StateStopped // 已停站：不是 idle，不能复用（Q23）
	busy.mu.Unlock()
	sBusy.mgr.sessions[busy.ID] = busy
	// 别名是**服务端的事实**（C5.3 的 name 键）：直接登记在会话上（生产路径见 assignIdentityLocked）。
	sBusy.mgr.mu.Lock()
	_ = sBusy.mgr.assignNameLocked(busy, "汇率单")
	sBusy.mgr.mu.Unlock()

	cases := []struct {
		name    string
		hs      *httptest.Server
		path    string
		body    string
		status  int
		fKey    string // 契约 2.4 的那一行：码与"要不要候选"都从表里取，不在测试里重抄
		candLen int    // -1 = 这个码下不该出现 candidates（C5.11）
	}{
		{"F1 省略目标 + 池里两条", hsTwo, "/api/sessions/switch", `{"env":"h1","ent":"99"}`,
			http.StatusConflict, "F1", 2},
		{"F19 别名取值非法（含 /）", hsTwo, "/api/sessions", `{"prog":"apmt520","name":"不合/法"}`,
			http.StatusBadRequest, "F19", -1},
		{"F20 别名指向的会话不能复用", hsBusy, "/api/sessions", `{"prog":"apmt520","name":"汇率单","ssh":"h1"}`,
			http.StatusConflict, "F20", 1},
	}
	byKey := map[string]debugContractInvalidForm{}
	for _, f := range debugContractInvalidForms {
		byKey[f.Key] = f
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			want, ok := byKey[c.fKey]
			if !ok {
				t.Fatalf("契约 2.4 的表里没有 %s —— 这张表是验收输入集，不许缺行", c.fKey)
			}
			resp := doReq(t, "POST", c.hs.URL+c.path, "", c.body)
			if resp.StatusCode != c.status {
				t.Fatalf("该退 %d，得 %d（%s）", c.status, resp.StatusCode, respError(t, resp))
			}
			var raw map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
				t.Fatalf("失败信封不是合法 JSON：%v", err)
			}
			// C5.15：`ok` 与 `error` 的形状一个字都不改，只**增加** code 与 candidates。
			if raw["ok"] != false {
				t.Errorf("ok 必须是 false（C5.15），实得 %v", raw["ok"])
			}
			if msg, _ := raw["error"].(string); strings.TrimSpace(msg) == "" {
				t.Errorf("error 必须非空（C5.15），实得 %v", raw["error"])
			}
			if got := raw["code"]; got != want.Code {
				t.Errorf("code = %v；契约 2.4 的 %s 行写的是 %q（4.1.1 是唯一清单）",
					got, c.fKey, want.Code)
			}
			cands, hasCands := raw["candidates"]
			if c.candLen < 0 {
				if hasCands {
					t.Errorf("%s（code=%s）不该带候选清单（C5.11），实得 %v", c.fKey, want.Code, cands)
				}
				return
			}
			if !hasCands {
				t.Fatalf("%s（code=%s）**必须**带候选清单（C5.11），实得 %v", c.fKey, want.Code, raw)
			}
			list, okList := cands.([]any)
			if !okList || len(list) != c.candLen {
				t.Fatalf("candidates 该是 %d 条，实得 %v", c.candLen, cands)
			}
			// C5.12：候选是**九键**，唯一出处是 internal/target（C5.4b 不许手抄第二份）。
			for _, one := range list {
				row, _ := one.(map[string]any)
				for _, k := range debugContractCandidateKeys {
					if _, ok := row[k]; !ok {
						t.Errorf("候选行少了 %q 键（C5.12 的九键），实得 %v", k, row)
					}
				}
			}
		})
	}
}

// TestContractNoSessionEnvelopeIsStillPinned 钉住 F2 的**信封**（`NO_SESSION` + 那句"先起一条"）。
//
// 生产里**已经没有端点产出这一档**：要一条**既有**会话的两条端点（切环境 / 按日志重放）
// 都改成了冷启动（它们本来就是**创建**会话的命令，池空是最正常的起点 —— 判据见
// `session_named_host_test.go` 的两条 `ColdStartIsNotNoSession`）。F2 这一档仍活在 CLI
// 目标层（`internal/target` 的 `failureNoSession`：`tt debug exec --session …` 那类
// "必须有既有会话"的命令），判据在那个包的 `resolve_test`。
//
// 那为什么还在这里钉它：`failTarget` 里 `targetEmptyPool` 那一支仍在（`resolveTarget` 的
// 四种结局与四种信封一一对应 —— 将来新增一条真的"必须有既有会话"的端点，拿 `mustTarget`
// 得到的码当场就是对的）。既然留着，就不能让它烂在那儿没人跑；所以像 F18 那样
// **直接驱动写信封的那一层**，把 409 + `NO_SESSION` + "不带候选"（C5.11）钉死。
func TestContractNoSessionEnvelopeIsStillPinned(t *testing.T) {
	rec := httptest.NewRecorder()
	failTarget(rec, targetEmptyPool, nil, "按接口日志重放")

	if rec.Code != http.StatusConflict {
		t.Fatalf("F2 该退 409（C5.16 的「缺目标/状态不符」那一档），实得 %d（%s）",
			rec.Code, strings.TrimSpace(rec.Body.String()))
	}
	var raw map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&raw); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v", err)
	}
	if raw["ok"] != false {
		t.Errorf("ok 必须是 false（C5.15），实得 %v", raw["ok"])
	}
	want := debugContractCodeNoSession
	if got := raw["code"]; got != want {
		t.Errorf("code = %v；契约 2.4 的 F2 行写的是 %q（4.1.1 是唯一清单）", got, want)
	}
	// 文案要能照做：池空时人/Agent 能看懂下一步是什么。
	msg, _ := raw["error"].(string)
	for _, wantText := range []string{"按接口日志重放", "没有 SSH 会话", "tt debug start"} {
		if !strings.Contains(msg, wantText) {
			t.Errorf("文案里该有 %q（哪个端点 + 现状 + 下一步），实得 %q", wantText, msg)
		}
	}
	// C5.11：F2 是"候选不出现"的那一档（表里 Candidates=false）。
	if _, has := raw["candidates"]; has {
		t.Errorf("F2（code=%s）不该带候选清单（C5.11），实得 %v", want, raw["candidates"])
	}
	if got := target.ExitFor(want, true, true); got != byKeyExit(t, "F2") {
		t.Errorf("F2 的退出码 = %d，契约 2.4 写的是 %d", got, byKeyExit(t, "F2"))
	}
}

// TestContractNameTakenEnvelopeIsWritten F18 的**信封**（`SESSION_NAME_TAKEN` + 占用者那一条）。
//
// F18 在生产里只剩**竞态**那一支到达（Q23 的三种结局已经把"空闲复用 / 忙碌报错"判在前头，
// 见 C1.6 与 `FailureNameTaken` 的注释）：两个同名请求同时在飞时，晚到的那一个在登记处
// 撞上已登记的别名。测它不必真的并发：撞车那一刻的判据是 `assignIdentityLocked`
// （下面直接驱动它），信封由 `writeNameTaken` 写（同样直接驱动）。
//
// 4.1.1 明写它归"用法错"那一行（**退 2 不退 3**），所以这里把**退出码**也钉下来：
// 服务端只给码与 HTTP 状态，退出码由调用方按 `target.ExitFor(code, ...)` 分档 ——
// 那一位与 4.1.1 的表同源，本测试核的就是"这张表说 F18 退几，就该退几"。
func TestContractNameTakenEnvelopeIsWritten(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	holder := fakeIdleSession("s-先到", "h1", "36")
	m.sessions[holder.ID] = holder
	m.mu.Lock()
	err := m.assignIdentityLocked(holder, "汇率单")
	m.mu.Unlock()
	if err != nil {
		t.Fatalf("第一条登记别名不该失败：%v", err)
	}
	// 第二条同名（大小写不同也算撞车：C2.5 的查重不区分大小写）。
	challenger := fakeIdleSession("s-后到", "h1", "36")
	m.mu.Lock()
	err = m.assignIdentityLocked(challenger, "汇率单")
	m.mu.Unlock()
	var taken *ErrNameTaken
	if !errors.As(err, &taken) {
		t.Fatalf("第二条用同名别名登记该失败（F18），实得 %v", err)
	}
	if taken.Occupant != holder.ID {
		t.Errorf("占用者该是 %s，实得 %s", holder.ID, taken.Occupant)
	}
	// 信封：409 + SESSION_NAME_TAKEN + 占用者那一条。
	rec := httptest.NewRecorder()
	srv := &Server{mgr: m}
	srv.writeNameTaken(rec, taken)
	if rec.Code != http.StatusConflict {
		t.Fatalf("F18 该退 409（C5.16 同一档：请求体合法、冲突的是会话池的状态），得 %d", rec.Code)
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("失败信封不是合法 JSON：%v\n%s", err, rec.Body.String())
	}
	if raw["code"] != debugContractCodeNameTaken {
		t.Errorf("code = %v，F18 要求 %q", raw["code"], debugContractCodeNameTaken)
	}
	list, ok := raw["candidates"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("F18 必须带候选清单，且只有占用者那一条（C5.11），实得 %v", raw["candidates"])
	}
	if row, _ := list[0].(map[string]any); row["id"] != holder.ID {
		t.Errorf("候选该是占用者 %s，实得 %v", holder.ID, list[0])
	}
	// 退出码：4.1.1 的表说 F18 退 2（**不退 3**）。
	if got := target.ExitFor(debugContractCodeNameTaken, true, true); got != byKeyExit(t, "F18") {
		t.Errorf("F18 的退出码 = %d，契约 2.4 / 4.1.1 写的是 %d", got, byKeyExit(t, "F18"))
	}
}

// byKeyExit 取 2.4 那张表里某一行写死的退出码（测试里不重抄这些数字）。
func byKeyExit(t *testing.T, key string) int {
	t.Helper()
	for _, f := range debugContractInvalidForms {
		if f.Key == key {
			return f.Exit
		}
	}
	t.Fatalf("契约 2.4 的表里没有 %s", key)
	return 0
}

// TestContractFlagConflictsIsNotYetEnforced 钉住四个标志的冲突矩阵（C3.3 的四种组合）
// 里**今天可测的两个半**：
//
//   - F12 / F13：`--session` 带的环境/企业与 `--env` / `--ent` 矛盾 → 退 2、`code: USAGE`；
//   - F14：`--session` 里的会话实际 ENT 与 `--ent` 不同 → 退 2、`code: USAGE`。
//
// **WP4a 之后这三条已经可断言了**（目标层是纯函数，`internal/target` 的
// `TestResolveF12EnvConflict` / `F13` / `F14` 逐条断言了码与退出码；CLI 那侧的包装
// 由 `internal/cli/debug/target_test.go` 钉住 `ExitFor(code, NewScenario, newSyntax)` 的分档 ——
// 分档判据是"**这个场景是不是新场景**"，不是"有没有打标志"，见 `target.Failure.NewScenario`）。
//
// 所以这里**不再跳过**：本包只做一件它能做的事 —— 核对 debugcontract 常量表里
// 这三条 F 的码与 `internal/target` 的常量一致（跨包字面量的唯一防线在那边，
// 这里只保证"本包引用的那份没漂"）。
func TestContractFlagConflictsIsNotYetEnforced(t *testing.T) {
	want := map[string]string{
		"F12": debugContractCodeUsage,
		"F13": debugContractCodeUsage,
		"F14": debugContractCodeUsage,
	}
	byKey := map[string]debugContractInvalidForm{}
	for _, f := range debugContractInvalidForms {
		byKey[f.Key] = f
	}
	for k, code := range want {
		f, ok := byKey[k]
		if !ok {
			t.Fatalf("非法写法表里没有 %s —— C3.3 的四种组合里这三条必须登记在案", k)
		}
		if f.Code != code {
			t.Errorf("%s 的机器码 = %q，C3.3 要求 %q", k, f.Code, code)
		}
		if f.Candidates {
			t.Errorf("%s 不该带候选清单（C5.11：标志冲突不是「该列候选」的场合）", k)
		}
	}
}

// TestContractAliasRulesAreNotYetEnforced 曾经钉住 `--name` 的取值纪律
// （C2.4 / C2.5 / F19 / Q17）：**非空、去空白后 1–64 字符、只排除 `/` 与 `#`**，
// 其余（中文、空格、标点）按原样允许。
//
// **WP4a 之后这条不再跳过**：`--name` 已在 `tt debug start` 上注册（B20 反面），
// 而取值纪律的实现与逐条判据在 `internal/target`（`ValidateAlias` 与它的
// `TestValidateAliasRules` / `TestAliasLengthCountsRunesNotBytes`）。
// 本包能做的、也只有意义的一件事是：核对本包引用的那份常量与契约逐字一致 ——
// 它们与 target 是一套值（跨包字面量由 `internal/target` 的 values 测试守着）。
func TestContractAliasRulesAreNotYetEnforced(t *testing.T) {
	if debugContractAliasForbid != "/#" {
		t.Fatalf("别名排除字符集常量 = %q，契约 C2.4/C2.5 写的是 %q", debugContractAliasForbid, "/#")
	}
	if debugContractAliasMinRunes != 1 || debugContractAliasMaxRunes != 64 {
		t.Fatalf("别名长度常量 = %d–%d，契约 C2.5 写的是 %d–%d",
			debugContractAliasMinRunes, debugContractAliasMaxRunes, 1, 64)
	}
	if target.AliasForbid != debugContractAliasForbid ||
		target.AliasMinRunes != debugContractAliasMinRunes ||
		target.AliasMaxRunes != debugContractAliasMaxRunes {
		t.Errorf("本包引用的一份（forbid=%q %d–%d）与 internal/target 的实现（forbid=%q %d–%d）不一致 ——\n"+
			"    两处必须同值：实现与判据同源，改一处就要改另一处。",
			debugContractAliasForbid, debugContractAliasMinRunes, debugContractAliasMaxRunes,
			target.AliasForbid, target.AliasMinRunes, target.AliasMaxRunes)
	}
}

// ---------------------------------------------------------------------------
// T2.1-e 规范 id 进出信封（C5.3 / C5.4 / F15 / Q22）
// ---------------------------------------------------------------------------

// TestContractIdentityKeysAreInSessionBrief 是 C5.3 / Q22 那五个新键（连同 `kind`）的**落地断言**。
//
// **WP4c 起它不再跳过**：`ref` / `display` / `name` / `ent` / `n` / `epoch` 已经在
// `SessionBrief` 上，`kind` 也补上了（C5.5）。这一条只做"键在不在、键名对不对"这一件事 ——
// "取不到时给空串而不省键"由 `TestContractSessionEnvelopeConstantKeysAreNeverOmitted`
// 用**真实输出**钉（那是 C5.1 的另一半）。
func TestContractIdentityKeysAreInSessionBrief(t *testing.T) {
	wantTags := map[string]string{
		"ref":     "ref",     // 规范 id（C2.0 恒定，C5.4）
		"display": "display", // 人读形态（作业段是快照，C5.4）
		"name":    "name",    // 别名（C2.5b）
		"ent":     "ent",     // 当前企业编号（快照，C2.5c）
		"n":       "n",       // 序号（C2.2 / Q24）
		"epoch":   "epoch",   // 世代号（F15）
		"kind":    "kind",    // 会话类型（C5.5：恒为 ssh）
	}
	rt := reflect.TypeOf(SessionBrief{})
	got := map[string]string{}
	for i := 0; i < rt.NumField(); i++ {
		tag := rt.Field(i).Tag.Get("json")
		if tag == "" {
			continue
		}
		got[strings.Split(tag, ",")[0]] = tag
	}
	for k, want := range wantTags {
		tag, ok := got[k]
		if !ok {
			t.Errorf("C5.3 / Q22 要求 `SessionBrief` 上有 %q 键，实际没有。\n    实得键：%v", k, keysOfTags(got))
			continue
		}
		if tag != want {
			t.Errorf("键 %q 的标签是 %q，契约 C5.2 要求键名一字不改（%q）", k, tag, want)
		}
	}
	// Q22 的"纯新增"：C5.3 明写**一个旧键都不改、不改名、不删**。旧键就在这两个常量表里，
	// 一个都不能少（少了就是有人删了旧键 —— 那会静默打断前端与老 CLI）。
	old := append([]string{"id", "env", "state", "prog"}, debugContractSessionOptionalKeys...)
	for _, k := range old {
		if _, ok := got[k]; !ok {
			t.Errorf("旧键 %q 不见了 —— C5.3 / Q22 是**纯新增**，旧键一个都不许改、不许删", k)
		}
	}
}

// keysOfTags 给失败信息用（键名排序后返回，输出必须稳定）。
func keysOfTags(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestContractEpochStalenessIsEnforced 是 F15（句柄过期）的**落地断言**。
//
// 三段，缺一不可 —— 它把"服务端产出的世代号"与"目标层的失效判定"接在一起：
//
//	① 服务端真的产 `epoch`（C5.3），且**第一次**拿到它之后手上就有一个世代号；
//	② 同一条会话**复用宿主跑新一轮**（`SetRun`，也就是 `tt debug start` 复用空闲宿主那条路）
//	   ⇒ 世代**换新**（本次定稿的语义见 nextSessionEpoch：建立 / 复用宿主 / 重启三处换代）；
//	③ 拿①那个旧世代号去落位 ⇒ 目标层报 `SESSION_STALE`、退 3、带当前代那一条候选（F15 / C5.11）。
//
// 为什么用真 Manager 而不是拼一个假视图：契约要的是"**服务端事实**让旧句柄失效"，
// 而①与②正是那件事的两半 —— 拼假数据只能证明目标层会算，证明不了服务端会换。
func TestContractEpochStalenessIsEnforced(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	live := idleOn("s1", cfgFor("h1", "36"))
	m.mu.Lock()
	if err := m.assignIdentityLocked(live, "汇率单"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"] = live
	m.mu.Unlock()

	before := m.Snapshot()
	if len(before) != 1 {
		t.Fatalf("前置不成立：池里该恰好一条会话，得 %d", len(before))
	}
	// ① 服务端产世代号。
	if strings.TrimSpace(before[0].Epoch) == "" {
		t.Fatalf("C5.3 要求 `session.epoch` 恒定出现且非空，实得 %q", before[0].Epoch)
	}
	oldEpoch := before[0].Epoch

	// ② 复用宿主跑新一轮 ⇒ 换代。
	if _, err := m.prepareSession(cfgFor("h1", "36"), "m1", "p1", "", "", "", "", ""); err != nil {
		t.Fatalf("复用空闲宿主该成功（它不拨号），实得：%v", err)
	}
	after := m.Snapshot()
	if len(after) != 1 || after[0].ID != "s1" {
		t.Fatalf("复用宿主不该换会话：实得 %+v", after)
	}
	if after[0].Epoch == oldEpoch {
		t.Errorf("复用宿主跑新一轮之后 epoch 没变（%q）—— F15 的「旧句柄当场失效」就判不出来了。\n"+
			"    换代的三个时机见 nextSessionEpoch 的注释（建立 / SetRun 复用 / 重启）。", oldEpoch)
	}

	// ③ 旧世代号 → SESSION_STALE（F15：环境与序号都对，但句柄是上一代）。
	// `Environments` 要给：目标层第一步就校验环境存在（C2.1 不猜、不回落），
	// 不给会先撞 ENV_UNKNOWN，测不到世代那一条。
	pool := target.Pool{Environments: []string{"h1-36"}, Sessions: sessionViews(after)}
	sel := selectorOf(t, "h1-36#1")
	if _, f := target.Resolve(pool, target.Request{Selector: sel, Epoch: oldEpoch}); f == nil {
		t.Fatalf("带上上一代的 epoch（%q）该被判失效（F15），却落上了", oldEpoch)
	} else {
		if f.Code != debugContractCodeStale {
			t.Errorf("F15 的码该是 %q，实得 %q（%s）", debugContractCodeStale, f.Code, f.Message)
		}
		if got := target.ExitFor(f.Code, f.NewScenario, true); got != byKeyExit(t, "F15") {
			t.Errorf("F15 的退出码 = %d，契约 2.4 写的是 %d", got, byKeyExit(t, "F15"))
		}
		if len(f.Candidates) != 1 {
			t.Errorf("F15 必须带候选（C5.11），且是**当前代**那一条，实得 %d 条", len(f.Candidates))
		}
	}
	// 反向判据：当前代的 epoch 带上去就该正常落位（不是"凡带 epoch 就拒"）。
	sel2 := selectorOf(t, "h1-36#1")
	if _, f := target.Resolve(pool, target.Request{Selector: sel2, Epoch: after[0].Epoch}); f != nil {
		t.Errorf("当前代的 epoch（%q）该正常落位，实得 %v", after[0].Epoch, f.Message)
	}
}

// selectorOf 解析一个选择器（测试里要按 `环境#序号` 落位；解析失败就当场判红）。
func selectorOf(t *testing.T, raw string) *target.Selector {
	t.Helper()
	sel, f := target.ParseSelector(raw)
	if f != nil {
		t.Fatalf("选择器 %q 该合法：%v", raw, f.Message)
	}
	return &sel
}

// TestContractNameIdempotencyIsEnforced 是 Q23 / C1.6 的**落地断言**：别名就是幂等键。
//
// 三条逐条（Q23 的原文）：
//
//	① **同名会话空闲 → 复用它**：再来一次拿到的是**同一条会话**（同一个 id / ref / n），
//	   不是第二条 —— 这就是"同一条命令重发不会悄悄多出一条会话"那句话的判据；
//	② **同名会话正在跑 / 已停站 → `SESSION_TARGET_BUSY` 退 3**：服务端 409 + 码 + 占用者候选，
//	   而且**不许静默再开一条**（判定在拨号之前，所以连 SSH 都不会去拨）；
//	③ **名字不存在 → 起新的**：池子里没有这个名字 ⇒ 放行，并且这次给定的别名会**登记**在
//	   结果那条会话上（否则调用方手上的别名下次就找不到了）。
//
// 判据全部用生产实现：①③ 在 Manager 上（复用宿主不拨号，所以测得了），②在 HTTP 信封上。
// 上一版这条是 `t.Skip`（那时服务端还不认别名）—— WP4c 把 `name` 变成服务端事实之后，
// 按它自己的交代改成了断言。
func TestContractNameIdempotencyIsEnforced(t *testing.T) {
	const alias = "复现-1"

	t.Run("① 空闲 → 复用同一条（不是第二条）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		// idleOn：envName 与 cfg 一致（与 NewSession 建出来的同形），否则这条会话的
		// ref 会因为"环境名还没定下来"而算成空串，测不到 C2.0 那句"地址一个字不变"。
		live := idleOn("s1", cfgFor("h1", "36"))
		live.Prog = "apmt520"
		m.mu.Lock()
		if err := m.assignIdentityLocked(live, alias); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记身份失败：%v", err)
		}
		m.sessions["s1"] = live
		m.mu.Unlock()
		first := m.Snapshot()[0]

		// 同一条命令原样重发（网络抖了、Agent 重试）。
		sess, err := m.LaunchWith(cfgFor("h1", "36"), "apmt520", "apmt520", alias, "")
		if err != nil {
			t.Fatalf("同名 + 空闲该复用宿主并成功，实得：%v", err)
		}
		if sess.ID != first.ID {
			t.Fatalf("该复用**同一条**会话（%s），实得 %s —— 别名当幂等键的判据就是这一条", first.ID, sess.ID)
		}
		got := m.Snapshot()
		if len(got) != 1 {
			t.Fatalf("池里该仍然只有一条会话（不许静默再开一条），实得 %d 条", len(got))
		}
		if got[0].Ref != first.Ref || got[0].N != first.N {
			t.Errorf("复用同一条会话时地址该一个字都不变：ref %q→%q，n %d→%d（C2.0）",
				first.Ref, got[0].Ref, first.N, got[0].N)
		}
		if got[0].Name != alias {
			t.Errorf("复用之后别名该还是 %q，实得 %q（C2.5b：生命周期内不变）", alias, got[0].Name)
		}
	})

	t.Run("② 正在跑 / 已停站 → SESSION_TARGET_BUSY（退 3）", func(t *testing.T) {
		hs, s := actionTestDebugServer(t)
		giveServerNamedEnv(s, "h1", "127.0.0.1", "36")
		busy := fakeIdleSession("s-忙", "h1", "36")
		busy.mu.Lock()
		busy.state = StateStopped
		busy.mu.Unlock()
		s.mgr.sessions[busy.ID] = busy
		s.mgr.mu.Lock()
		if err := s.mgr.assignNameLocked(busy, alias); err != nil {
			s.mgr.mu.Unlock()
			t.Fatalf("登记别名失败：%v", err)
		}
		s.mgr.mu.Unlock()

		resp := doReq(t, "POST", hs.URL+"/api/sessions", "",
			`{"prog":"apmt520","name":"`+alias+`","ssh":"h1"}`)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("Q23：忙碌的同名会话该当场拒绝（409），得 %d（%s）", resp.StatusCode, respError(t, resp))
		}
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			t.Fatalf("失败信封不是合法 JSON：%v", err)
		}
		if raw["code"] != debugContractCodeTargetBusy {
			t.Errorf("code = %v，F20 / Q23 要求 %q", raw["code"], debugContractCodeTargetBusy)
		}
		if got := target.ExitFor(debugContractCodeTargetBusy, true, true); got != byKeyExit(t, "F20") {
			t.Errorf("F20 的退出码 = %d，契约 2.4 写的是 %d", got, byKeyExit(t, "F20"))
		}
		list, ok := raw["candidates"].([]any)
		if !ok || len(list) != 1 {
			t.Fatalf("F20 必须带候选（占用者那一条，C5.11），实得 %v", raw["candidates"])
		}
		if row, _ := list[0].(map[string]any); row["id"] != busy.ID {
			t.Errorf("候选该是占用者 %s，实得 %v", busy.ID, list[0])
		}
		// 不许静默再开一条：池子里那条一动没动，也没有第二条被造出来。
		if got := len(s.mgr.Snapshot()); got != 1 {
			t.Errorf("忙碌的同名请求不许改动会话池（该报错就报错），实得 %d 条", got)
		}
	})

	t.Run("③ 名字不存在 → 起新的（并把名字登记上）", func(t *testing.T) {
		m := NewManager(cfgFor("h1", "36"))
		unnamed := idleOn("s1", cfgFor("h1", "36"))
		m.mu.Lock()
		if err := m.assignIdentityLocked(unnamed, ""); err != nil {
			m.mu.Unlock()
			t.Fatalf("登记身份失败：%v", err)
		}
		m.sessions["s1"] = unnamed
		m.mu.Unlock()
		if got := m.Snapshot()[0].Name; got != "" {
			t.Fatalf("前置不成立：这条会话该是无名的，实得 %q", got)
		}
		sess, err := m.LaunchWith(cfgFor("h1", "36"), "apmt520", "apmt520", alias, "")
		if err != nil {
			t.Fatalf("名字不存在该放行（起新的 / 落在这条宿主上），实得：%v", err)
		}
		if got := sess.Name(); got != alias {
			t.Errorf("这次给的别名没登记到会话上（%q）—— 下一次用这个别名就落不了位了", got)
		}
		// 反向：**别名不同**时不复用那条会话（C2.5b：别名只在创建时给一次，
		// 把 X 挂到一条已经叫 Y 的会话上就是静默改名）。
		if nameCompatible(alias, "另一个名字") {
			t.Errorf("一条已经叫 %q 的会话不该被复用成 %q（C2.5b）", alias, "另一个名字")
		}
		if !nameCompatible("", "随便什么名字") {
			t.Errorf("没名字的宿主可以被起名（Q23 的「名字不存在 → 起新的」落在复用宿主上的形态）")
		}
	})
}

// ---------------------------------------------------------------------------
// T2.1-f 人话输出（C5.9 身份行 / C5.14 候选行）
// ---------------------------------------------------------------------------

// TestContractCandidateLineFormat 钉住 C5.14 的候选行格式。
//
// 为什么值得钉：候选行的用途是"照着它重发一枪"。少一个空格、把 ref 换成 display，
// 抄下去的就是一条会漂的写法 —— 那正是 C5.4 说的"要恒定用 ref"。
//
// **WP4c 起改调真实现**（`target.CandidateLine`，C5.14 的唯一实现）：这一条上一版
// 自己带着一份手抄的渲染函数，并在注释里写明"落地后应当删掉辅助函数、改成调真实现"。
// 现在这么做了 —— 测试验的就是生产那一份。
func TestContractCandidateLineFormat(t *testing.T) {
	got := target.CandidateLine(target.Candidate{
		ID: "s1", Ref: "恒烁正式区#1", Display: "恒烁正式区/apmt520#1",
		Name: "汇率单", Env: "恒烁正式区", Ent: "99", N: 1, Prog: "apmt520", State: "stopped",
	})
	want := "  恒烁正式区#1  汇率单  已停站  ENT 99"
	if got != want {
		t.Errorf("候选行（C5.14）= %q\n    期望 %q\n"+
			"    格式：两格缩进 + <ref> + 两空格 + <name> + 两空格 + <state> + 两空格 + ENT <ent>",
			got, want)
	}
	if strings.Contains(got, "会话 ") {
		t.Errorf("候选行不该带 `会话 ` 前缀（C5.14 末段：那是 5.9 身份行的前缀，两行不是同一行）")
	}
	if got := target.CandidateLine(target.Candidate{Ref: "恒烁正式区#2", State: "idle"}); !strings.Contains(got, "(未命名)") || !strings.Contains(got, "ENT ?") {
		t.Errorf("未命名/取不到企业编号时要写字面量 (未命名) 与 ?（C5.14），实得 %q", got)
	}
	// 反向：第一个字段必须是 ref 的形态（环境#序号），不能是 display 的形态（含 /）。
	first := strings.Fields(strings.TrimSpace(want))[0]
	if strings.Contains(first, debugContractSelectorSep) {
		t.Errorf("候选行第一个字段 %q 含 %q —— 那是 display，会随会话当前作业漂（C5.14）",
			first, debugContractSelectorSep)
	}
}

// TestContractIdentityLineIsLanded 是 C5.9 身份行的**落地断言**（本批交付）。
//
// 上一版这条叫 `…IsNotYetLanded` 并且 `t.Skip`：服务端那时已经有 `display` 与 `name`、
// `internal/target` 也已经有 `IdentityLine`，差的是**CLI 那行人话输出换字** ——
// 而换字正是 H2 里"单会话输出逐字不变"的唯一例外（契约 C8.4 登记为第二处允许的变化）。
// 本批按 C8.4 换了，于是这条跳过**按它自己的交代改成了真断言**，
// 台账（`internal/testkit/skipledger_test.go`）里那条 `layPending` 也随之删掉
// —— 不再跳过的东西不该留在跳过台账里。
//
// 分工写清楚（免得两头都以为对方测了）：
//   - **这里**钉"服务端产出的数据 + C5.9 的渲染"这一个组合：
//     会话身份七键 → `sessionView` → `target.IdentityLine`，逐字对 C5.9；
//   - **CLI 那一头**（`internal/cli/debug/envtarget_test.go` 的 `TestDBGIdentity*`）
//     钉 `dbgIdentity` 真的把它打出来、且第二行（账号/库/区域）也在。
//     本包 import 不了 `internal/cli/debug`（那会成环），所以那条只能在那一侧。
//
// 身份行按 C5.9 是**两行**（第一行四字段，第二行账号/库/区域）。这里只钉第一行 ——
// 第二行那三样来自配置与企业快照，不是会话的身份键，由 CLI 那一侧的测试覆盖。
func TestContractIdentityLineIsLanded(t *testing.T) {
	m := NewManager(cfgFor("h1", "36"))
	live := idleOn("s1", cfgFor("h1", "36"))
	live.Prog = "apmt520"
	live.cfg.Topent = "99" // 起会话时给的 --ent（`ent` 那一栏的来源）
	m.mu.Lock()
	if err := m.assignIdentityLocked(live, "汇率单"); err != nil {
		m.mu.Unlock()
		t.Fatalf("登记身份失败：%v", err)
	}
	m.sessions["s1"] = live
	m.mu.Unlock()
	b := m.Snapshot()[0]
	if b.Display == "" || b.Name == "" {
		t.Fatalf("C5.3 要求服务端产出 display 与 name（WP4c 已落地），实得 display=%q name=%q", b.Display, b.Name)
	}

	// C5.9 的逐字格式：`会话 <display> · <name> · <state> · ENT <ent>`
	// （分隔符是 空格 + U+00B7 + 空格；`<state>` 用现行中文标签）。
	line := target.IdentityLine(sessionView(b))
	want := "会话 " + b.Display + " · " + b.Name + " · " + target.StateLabel(b.State) + " · ENT " + b.Ent
	if line != want {
		t.Errorf("C5.9 的身份行 = %q\n    期望 %q", line, want)
	}
	// 身份行写 display（人读形态），第 1 个字段必须含 `/`（`环境/作业#序号`）——
	// 写成 ref 就少了作业段，与 C5.9「身份行写字面量 会话 + <display>」不符。
	if first := strings.Fields(strings.TrimPrefix(line, "会话 "))[0]; !strings.Contains(first, debugContractSelectorSep) {
		t.Errorf("身份行第 1 个字段 %q 不含 %q —— 那不是 display（C5.9 要求写 display，对账才用 ref）",
			first, debugContractSelectorSep)
	}
	// 取不到的两个字段写字面量，**不省整段**（省了字段位置就随会话漂）。
	bare := target.IdentityLine(sessionView(SessionBrief{Env: "恒烁正式区", N: 2, Prog: "capp115", State: "idle"}))
	if bare != "会话 恒烁正式区/capp115#2 · "+target.UnnamedPlaceholder+" · 空闲 · ENT "+target.UnknownPlaceholder {
		t.Errorf("未命名/取不到企业编号时要写字面量 (未命名) 与 ?（C5.9），实得 %q", bare)
	}
}

// ---------------------------------------------------------------------------
// T2.1-g 端点与信封的两侧差别（C5.15 / C5.16 / 第 7 节的表）
// ---------------------------------------------------------------------------

// postForGenericFailure 发一条**必定走到"拨号失败"那一档**的请求，用来验通用失败信封的形状。
//
// 为什么是 `POST /api/wslogs/debug`：给它一条**池里的会话**（`actionTestServerWithSession`）
// 之后，落位那一关过得去，接着就是 `dialSSH` —— 而 `actionTestServer*` 起的服务**没有生效环境**
// （`&Config{Listen: …}`，SSH.Host 为空），于是当场 500 + 现行运行失败码。
// 那正好是"通用失败"那一档（不是 `SESSION_*` 那几族：它们的候选清单形状另有测试盯）。
//
// 从前这里打的是 `POST /api/wstest`。那条端点本批已**从产品里取消**
// （仓库主人定性：不是标准接口，是临时测试工具），所以换一个**仍然活着**的端点；
// 两条断言的意图一个字没改 —— 测的是信封形状，不是那条端点。
func postForGenericFailure(t *testing.T, hs *httptest.Server, sid string) *http.Response {
	t.Helper()
	return doReq(t, "POST", hs.URL+"/api/wslogs/debug", "", `{"rowid":"1","session":"`+sid+`"}`)
}

// TestContractHTTPFailureEnvelopeKeepsItsShape 钉住 C5.15 的两半：
// **不许动的部分**（`ok` / `error` 的形状，B8）与**必给的部分**（`code`）。
//
// WP4c 之前这条只钉前半句（那时 HTTP 失败信封是 `{"ok":false,"error":…}` 两键，
// 自己的注释里写着"加进 code 之后要一起改这里"）—— 现在按那句话改了：
// `code` 必须在（C5.15：必给），而 `candidates` 只在 C5.11 那五种码下出现，
// 这条请求的码（运行失败那一档）不在其中，所以这里反过来断言它**不该有**。
//
// 这与 CLI 侧（`output.WriteError` 的四键恒定）是**两套形状**，别混：
// HTTP 侧给前端与 CLI 的 `dbgAPI`，CLI 侧给人读的 `--json` 调用方。
func TestContractHTTPFailureEnvelopeKeepsItsShape(t *testing.T) {
	hs, sid := actionTestServerWithSession(t)
	resp := postForGenericFailure(t, hs, sid)
	if resp.StatusCode != 500 {
		t.Fatalf("前置不成立：期望拨号失败退 500，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("HTTP 失败信封不是合法 JSON：%v", err)
	}
	if raw["ok"] != false {
		t.Errorf("HTTP 失败信封的 ok 必须是 false（B8 / C5.15），实得 %v", raw["ok"])
	}
	if s, _ := raw["error"].(string); strings.TrimSpace(s) == "" {
		t.Errorf("HTTP 失败信封必须有非空的 error（B8），实得 %v", raw["error"])
	}
	// C5.15：`code` **必给**（调用方靠它分支，Q15 明写"Agent 读 code，不读数字"）。
	if code, _ := raw["code"].(string); strings.TrimSpace(code) == "" {
		t.Errorf("HTTP 失败信封必须有非空的 code（C5.15：必给），实得 %v", raw["code"])
	}
	// C5.11：这个码不在"必须带候选"的五种里，所以 candidates 不该出现。
	if _, ok := raw["candidates"]; ok {
		t.Errorf("code=%v 不在 C5.11 那五种码里，不该有 candidates 键，实得 %v", raw["code"], raw)
	}
}

// TestContractHTTPFailureCodeIsLanded 是 C5.15 / C5.16 的**落地断言**：
// 缺目标 / 歧义 / 别名撞车那几族用 409 或 404 并带 `code` 与 `candidates`。
//
// 上一版这条是 `t.Skip`（那时 HTTP 侧只有 `{ok,error}` 两键）。WP4c 补上之后
// 按它自己的交代改成断言 —— 逐条的实测在
// `TestContractInvalidFormsServerSideEnvelopes`（F1/F19/F20）与
// `TestActionEndpointsRequireExplicitTarget`（池空 / 歧义 / 指名不存在）里，
// 这里只钉住"这个键从此必给"这一条。
func TestContractHTTPFailureCodeIsLanded(t *testing.T) {
	hs, sid := actionTestServerWithSession(t)
	resp := postForGenericFailure(t, hs, sid)
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("HTTP 失败信封不是合法 JSON：%v", err)
	}
	code, _ := raw["code"].(string)
	if strings.TrimSpace(code) == "" {
		t.Fatalf("HTTP 失败信封必须有 code（C5.15），实得 %v", raw)
	}
	// 这个码必须落在 4.1.1 那张唯一清单里（或它是现行的运行失败码）——
	// 不许随手发明一个没人认识的字符串。
	known := map[string]bool{output.CodeQueryFailed: true, output.CodeConnFailed: true}
	for _, f := range debugContractInvalidForms {
		known[f.Code] = true
	}
	if !known[code] {
		t.Errorf("code = %q 既不在契约 2.4 的 F 表里，也不是现行的运行失败码 —— "+
			"4.1.1 是**唯一清单**，加新码要走那一处", code)
	}
}

// TestContractStatusStaysTwoDocuments 钉住 C1.5b / Q21 里**今天就不许动**的那一半：
//
// `tt debug status` 默认输出 JSON（今天就是），而且是**两个** JSON 文档
// （`GET /api/status` 与 `GET /api/sessions` 各一行），**不合并成一个信封**。
// 契约只在每个 session 元素上按 C5.8 换成 5.1 的 `session` 对象（纯新增四个键）。
//
// 这里调的是服务端真实端点，不是拼字符串 —— 它同时是"端点没被顺手改掉"的判据。
func TestContractStatusStaysTwoDocuments(t *testing.T) {
	srv := NewServer(cfgFor("h1", "36"), nil, "")
	mux := http.NewServeMux()
	srv.routes(mux)
	hs := newTestHTTPServer(t, mux)

	// /api/status：服务级文档，与 C5.8 无关，一个键都不许动。
	resp := doReq(t, "GET", hs.URL+"/api/status", "", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/api/status 该退 200，得 %d（%s）", resp.StatusCode, respError(t, resp))
	}
	var st map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("/api/status 该回一个 JSON 文档：%v", err)
	}
	// 今天这个文档是**服务级状态**（server/ssh/zone/listen/watchdog…），**没有** ok 键。
	// C1.5b 管的是 `tt debug status` 命令默认打 JSON 这件事，两个文档的形状它一个字都没改
	// （契约原话：行数、两个文档的顺序、以及 ok 键的有无都不动）。
	// 所以这里断言的是"它在、且是个有内容的 JSON 对象"，不是"它有 ok 键"。
	if len(st) == 0 {
		t.Errorf("/api/status 的文档不该是空的（C1.5b：这两个文档今天就打，形状不许动）")
	}
	if _, ok := st["server"]; !ok {
		t.Errorf("/api/status 该有 server 键（服务级状态文档的既有形状），实得 %v", keysOf(st))
	}

	// /api/sessions：会话列表文档。今天空池就是空数组 —— 这是"默认 JSON、两行文档"里的第二行。
	resp2 := doReq(t, "GET", hs.URL+"/api/sessions", "", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("/api/sessions 该退 200，得 %d（%s）", resp2.StatusCode, respError(t, resp2))
	}
	var ls map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&ls); err != nil {
		t.Fatalf("/api/sessions 该回一个 JSON 文档：%v", err)
	}
	if _, ok := ls["sessions"]; !ok {
		t.Errorf("/api/sessions 该有 sessions 键（C5.8 的宿主形状），实得 %v", keysOf(ls))
	}
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

// keysOf 把 map 的键排序后返回（失败信息里要用，顺序必须稳定）。
func keysOf(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// selectorSecondSegKind 是 C2.3 的判据：第二段全数字 → 企业编号；否则 → 作业名。
//
// WP3 起它**委托**给 `internal/target` 的生产实现（`SecondSegmentKind`）：
// 判据只有一处，测试验的就是那一处（不再是"测试自己抄一份、再验自己"）。
func selectorSecondSegKind(seg string) string { return target.SecondSegmentKind(seg) }

// invalidFormKeys 返回表里现有的 F 键（失败信息用）。
func invalidFormKeys() []string {
	out := make([]string, 0, len(debugContractInvalidForms))
	for _, f := range debugContractInvalidForms {
		out = append(out, f.Key)
	}
	return out
}

// 注：`contractRefKeyLanded` / `failureEnvelopeHasCode` 两个"落地了没有"的探针
// **WP4c 已删** —— 它们服务的跳过（"等 ref 键""等 HTTP 失败信封的 code"）都已改成
// 真断言，留着会让人以为还有东西在等。现在"落地了没有"的判据就是那些断言本身；
// `invalidFormKeys` 留着给失败信息列 F 键（仍在使用）。

// stateLabelOf 是 B30 / C5.9 的五个会话状态取值与它们的**中文标签**的表。
//
// 它是"值只定义一次"这条纪律在 T2.2 那一侧的落点：`internal/cli/debug` 的
// `dbgStateLabel`（debugenv.go:351）是这五个标签的**生产实现**，而 C5.9 的身份行
// `会话 <display> · <name> · <state> · ENT <ent>` 明确要求 `<state>` 用它。
//
// **一处已知的重复，如实写明**：`internal/cli/debug/currentsession_char_test.go` 的
// `TestCharDBGStateLabels` 里也有一份同样的五对。这不是偷懒 —— Go 的包**看不见**
// 另一个包 `_test.go` 里声明的符号（实测：`dbgcontract.StateLabelOf` 编译不过），
// 而 WP2 的纪律是只许新增 `_test.go`，不能把这五个标签挪进生产文件。
// 两边的成对断言（这条表 + 那条测试）就是防漂移的判据：任何一边改了标签，
// 另一边那条测试立刻红。
func stateLabelOf() map[string]string {
	return map[string]string{
		"idle":    "空闲",
		"loading": "连接中…",
		"stopped": "已停站",
		"running": "运行中",
		"exit":    "已断开",
	}
}

// newTestHTTPServer 起一个测试 HTTP 服务，挂上调试面自己的路由表。
// 复用 mode_test.go 的 doReq / respError。
func newTestHTTPServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	hs := httptest.NewServer(h)
	t.Cleanup(hs.Close)
	return hs
}

// ---------------------------------------------------------------------------
// T2.4 不测什么（WP2 的第三类交付；计划 Q13 / WP2 一节）
// ---------------------------------------------------------------------------
//
// 这一节**不是注释装饰**：半年后会有人以为"这些测过了"。逐条写清为什么测不了，
// 以及替代它的是什么。
//
// 1. `internal/debug/session.go` 的主流程 —— **不测**。要真 SSH PTY + 真 fgldb 交互：
//    那里 19 处 `time.Now()`、多处硬编码 `time.Sleep`，断言只能落在"最终态"，
//    而最终态要真引擎。理由与现状见 `internal/debug/README.md` 的「故意不覆盖」一节
//    （原文：要测得先造一个 fgldb 协议模拟器，那是**一次依赖反转**；而买家是一个
//    已被 87 个用例与真机自检覆盖的包的内部行覆盖）。计划 Q13 的定稿就是
//    「不硬测，老实写下不测什么」。替代：WP6 的"双 Agent 并发实测"（人工跑一遍）。
//
// 2. `api.go` 的**快乐路径** —— 不测。要 `*host.SSHConn`，它的 `cli` 是私有字段
//    （`host/ssh.go:18`），从外部伪造得实现 8 个方法、其中三个返回不可构造的类型。
//    测试只注入**必然失败**的 dial，买到的是错误半边（见 `api_action_test.go` 的文件头）。
//    所以本文件里凡是要走到"拨号成功之后"的断言，一律**不能**在这里写。
//
// 3. 服务端五处 `mgr.Current()`（B11 / C7.1）**今天只能钉住它自己的语义**，
//    钉不住"每一处各自干了什么"：
//    - `sessOfWriteGlobal`（api.go:367）：能测到的只有"池空 → 返回 nil → 放行"
//      与"池里有空宿主 → 放行"这两个分支（后者要真会话，见第 2 条）；
//    - `hSessionSwitch`（api.go:816）：能测到"池里有**同环境**会话 → 幂等返回那一条"，
//      因为它在 `Current() != nil && sameEnvAs` 时**直接 return**，不碰 pty/conn；
//      剩下两条分支（不同环境 → Close 旧会话 + 新建；池空 → 新建）都会打到
//      `Close()`/`CreateSessionOn()`，而手工造的 Session 没有 pty/conn，`forceExit()`
//      会在 `s.pty.Close()` 上打到 nil（`session_reuse_test.go:73-75` 已记过这个坑）；
//    - `hWSLogDebug`（api.go:1679）：必须先拨 SSH 查 `wsfa_t` 拿到 item 才走到那一步，
//      见第 2 条；
//    - `wslogTopent`（api.go:1724）：同上（它由 `hWSLogDebug` 调用）；
//    - `hDBSQL`（api.go:1754）：拨号失败发生在使用 ent 之前，拿不到可分辨的行为。
//    替代判据：`internal/testkit` 的静态检查（每一个 `mgr.Current()` 都记在案上，
//    WP4 删干净才转绿）+ `Manager.Current()` 自身的表征测试（T2.2）。
//
// 4. `prepareSession`（manager.go:242）的**"换环境 → 收掉旧会话再新建"那一条分支** —— 不测。
//    它要 `live.Close()`，同上会打到 nil 的 pty/conn。可测的两条（同目标空闲复用、
//    同目标忙碌报错）已在 `session_reuse_test.go` 与本次新增的表征测试里钉住。
//
// 5. 真机相关的一切（真 SSH / 真库 / 真 fgldb）**不在契约测试范围**（契约第 10 节末段）。
//    技能在 `.agents/skills/tt-test`，它规定跑真机测试必须先取用户授权、且环境配置
//    不得进仓库 —— 所以它不可能是一条 `go test` 用例。
