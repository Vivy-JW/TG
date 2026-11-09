package verify

import (
	"bytes"
	"strings"
	"testing"

	"tt/internal/dev/fence"
	"tt/internal/dev/model"
	"tt/internal/dev/pkgfile"
	"tt/internal/dev/synth"
	"tt/internal/dev/testutil"
)

// 本包是三道闸门（gate1 字节恒等 + 授权 / gate2 不变量 I1–I15 / gate3 装载模拟）。
//
// **各条拒绝路径**已经由端到端用例覆盖（tt dev tzc selftest 的 31 项：改围栏行→3、
// 改 READONLY 区段→4、改结构行→3、塞 ]]> →3……），批 1 之后它们默认档就跑。
// 所以这里不重测那些，补的是端到端钉不到的两件：
//
//  1. **干净路径零发现** —— 端到端只会在"该拒绝的没拒绝"时红，而"不该拒绝的拒绝了"
//     在它那边表现为一串与预期不符的失败，定位不到是哪个闸门误报。
//  2. **gate1 那条设计决定**：受保护字节的**行尾归一**（CRLF↔LF）算 info 不算 error。
//     这是 README 里专门写了一段的一条，端到端没有用例单独钉它。
//
// 夹具走**合成包**（不依赖真实语料）：testutil 造包 → pkgfile 打开 → synth 合成 →
// fence 渲染/解析。这条链是仓库里的标准组装方式（fence_test.go 的 synthDoc 同款；
// 各包自己抄一份是因为 `_test.go` 不能被跨包 import）。

// synthBase 造一份**基线**与它的围栏文本。
//
// **这里有一个坑，踩过一次，写下来免得重踩**：`base` 必须是"把基线自己的围栏文本
// 解析一遍"得到的文档，**不能**直接拿 `synth.Synthesize` 的产物。
//
// 区别在 Text 与坐标：合成产物 `doc0` 的 Text 是**未加围栏**的正文，Regions 也在那套
// 坐标里；而闸门要的 base 是 `loadBase` 从工作区读回来的那份 —— Text 是**加了围栏**的
// `base.full.4gl`，Regions 是围栏坐标。两者混用的话，gate1 会把每一条围栏行都判成
// "锚定部分被改动"，一份什么都没改的文档报出十几条 error（第一版就是 14 条）。
func synthBase(t *testing.T, prog string) (*model.Document, []byte) {
	t.Helper()
	p, err := testutil.NormalPkgPath(t.TempDir(), prog)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := pkgfile.Open(p, pkgfile.OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	doc0, err := synth.Synthesize(pkg, synth.Options{})
	if err != nil {
		t.Fatal(err)
	}
	fenced, _, _, err := fence.Render(doc0)
	if err != nil {
		t.Fatal(err)
	}
	// 基线 = 对它自己的围栏文本做一次解析。
	baseline, err := fence.Parse(doc0, fenced)
	if err != nil {
		t.Fatalf("基线自解析失败：%v", err)
	}
	return baseline.Doc, fenced
}

// codesOf 把报告里的码收集成集合，便于"有没有某一条"。
func codesOf(r *Report) map[string]int {
	out := map[string]int{}
	for _, f := range r.Findings {
		out[f.Code]++
	}
	return out
}

// TestCleanDocumentHasNoErrors 原样渲染、原样解析 —— 三道闸门都不该有意见。
//
// 这条看着平淡，但它钉的是**误报**：闸门写严了最容易出现的就是"什么都没改也报错"，
// 而那会让整条写回路径不可用（人会开始习惯性忽略报错）。
func TestCleanDocumentHasNoErrors(t *testing.T) {
	base, fenced := synthBase(t, "adzi999")
	parsed, err := fence.Parse(base, fenced)
	if err != nil {
		t.Fatalf("原样解析就不该失败：%v", err)
	}
	rep := Gate1(base, parsed)
	if rep.HasError() {
		t.Errorf("原样文档不该有 error，得 %s：%+v", rep.Summary(), rep.Findings)
	}
	if rep.HasDenied() {
		t.Errorf("原样文档不该被拒，得 %s", rep.Summary())
	}
}

// TestGate1EOLNormalizationIsInfoNotError 钉住 gate1 的**行尾等价**：
// 编辑器（VS Code 等）把 CRLF 归一成 LF 不算改动 —— 那些字节本来就不写回包。
//
// 要是不这么做，一次"另存为"就会让整份文档全红，而真正的问题（有人改了只读区）
// 会淹在这几百条里。所以：**等价处理 + 一条 info 说明发生过什么**。
func TestGate1EOLNormalizationIsInfoNotError(t *testing.T) {
	base, fenced := synthBase(t, "adzi999")

	// 先把 CRLF 全部归一成 LF（模拟编辑器另存）。围栏文本里本来就有两种行尾
	// （合成夹具刻意混用，见 testutil/fixture.go），所以这一步**必然**产生差异。
	normalized := bytes.ReplaceAll(fenced, []byte("\r\n"), []byte("\n"))
	if bytes.Equal(normalized, fenced) {
		// 判失败而不是 Skip：夹具是我们可以控制的，没有 CRLF 说明夹具变了 ——
		// 那时这条测试会**静默变成空转**（跳过的测试和通过的测试在报告里一样无害）。
		t.Fatal("夹具里没有 CRLF，构造不出行尾差异 —— 这条测试失去对象了，别让它静默跳过")
	}

	parsed, err := fence.Parse(base, normalized)
	if err != nil {
		t.Fatalf("行尾归一后该仍能解析：%v", err)
	}
	rep := Gate1(base, parsed)
	if rep.HasError() {
		t.Errorf("行尾归一不该算 error（那些字节不写回包），得 %s：%+v", rep.Summary(), rep.Findings)
	}
	if n := codesOf(rep)["gate1.eol-normalized"]; n == 0 {
		t.Errorf("该给一条 gate1.eol-normalized 的 info（让人知道发生过什么），得 %v", codesOf(rep))
	} else if n > 1 {
		t.Errorf("该只报一条汇总，得 %d 条", n)
	}
	// 这条必须是 info 级 —— 变成 warn 会让"全绿"带上噪声，变成 error 就直接不可用了。
	for _, f := range rep.Findings {
		if f.Code == "gate1.eol-normalized" && f.Severity != SevInfo {
			t.Errorf("该是 info 级，得 %q", f.Severity)
		}
	}
}

// TestGate1CatchesRealChange 行尾等价**不是**放行一切：
// 受保护字节里真的动了内容（这里改**只读区段的正文**）照样拦下。
//
// 与上一条合起来才完整：等价处理有边界，边界内是 info、边界外是 error。
//
// 这里挑只读区段而不是围栏行，是因为围栏行归 fence.Parse 管（改坏了它先报错），
// 而"正文被改"才是 gate1 字节恒等那一条的靶子。
func TestGate1CatchesRealChange(t *testing.T) {
	base, fenced := synthBase(t, "adzi999")

	// 找一个只读区段，改它正文里的一个字符（只读区是"受保护字节"）。
	var target *model.Region
	for _, r := range base.AllRegions() {
		if r.Kind == model.RegionSection && !r.Editable {
			target = r
			break
		}
	}
	if target == nil {
		t.Fatal("这份夹具里该有只读区段（包是未解锁的，SectionRegion 全是只读）")
	}

	// 在只读区正文中间插一个字符 —— 动内容，也顺便动了长度。
	tampered := make([]byte, 0, len(fenced)+1)
	tampered = append(tampered, fenced[:target.ContentSpan.Start+2]...)
	tampered = append(tampered, 'Z')
	tampered = append(tampered, fenced[target.ContentSpan.Start+2:]...)

	parsed, err := fence.Parse(base, tampered)
	if err != nil {
		// 解析层先拦住了，那更好 —— 那这条测试的对象就不存在了（如实说明，不假装通过）。
		t.Skipf("fence.Parse 自己就拦住了（%v）—— 这条测的是 gate1，不是解析层", err)
	}
	rep := Gate1(base, parsed)
	if !rep.HasError() {
		t.Errorf("改了只读区段正文该报 error，得 %s：%+v", rep.Summary(), rep.Findings)
	}
	if !rep.HasDenied() {
		t.Errorf("只读区的改动属于「写入被拒」（退出码 4）而不是普通校验失败（3），"+
			"该有 Denied 标记，得 %s", rep.Summary())
	}
}

// ---- Report：计数与摘要 ----

// TestReportCounts 三个等级各自计数；`Denied` 是**额外**一栏，不是第四个等级。
//
// 它决定退出码是 3（验证失败）还是 4（写入被拒）。**一条「写入被拒」的发现会同时计入
// Errors 与 Denied**（见 Add：先按等级计数，再在 error 级上补 Denied）——
// 所以反过来不成立：Errors 里有一部分可能属于被拒，而 Denied 必然是 Errors 的子集。
func TestReportCounts(t *testing.T) {
	rep := &Report{}
	if rep.HasError() || rep.HasDenied() {
		t.Error("空报告不该有 error / denied")
	}
	if got := rep.Summary(); got != "error=0 warn=0 info=0" {
		t.Errorf("空报告摘要得 %q", got)
	}

	rep.Add(Finding{Code: "a", Severity: SevInfo})
	rep.Add(Finding{Code: "b", Severity: SevWarn})
	rep.Add(Finding{Code: "c", Severity: SevError})
	rep.Add(Finding{Code: "d", Severity: SevError, Denied: true})

	// 两条 error（c 与 d）：Denied 不把 d 从 Errors 里挪走。
	if rep.Errors != 2 || rep.Warns != 1 || rep.Infos != 1 || rep.Denied != 1 {
		t.Errorf("计数不对：%+v", rep)
	}
	if rep.Denied > rep.Errors {
		t.Error("Denied 是 Errors 的子集，不该多过它")
	}
	if !rep.HasError() || !rep.HasDenied() {
		t.Error("该报告 error 与 denied 都有")
	}
	// Denied 只认 error 级：warn/info 上挂它不算数（那会让退出码 4 出现在没有 error 的报告里）。
	rep2 := &Report{}
	rep2.Add(Finding{Code: "w", Severity: SevWarn, Denied: true})
	rep2.Add(Finding{Code: "i", Severity: SevInfo, Denied: true})
	if rep2.Denied != 0 || rep2.HasDenied() {
		t.Errorf("warn/info 上挂 Denied 不该算写入被拒：%+v", rep2)
	}

	// Findings 顺序要保住 —— 报错的顺序就是判定的顺序，乱了会让人以为是另一处先出的问题。
	var gotCodes []string
	for _, f := range rep.Findings {
		gotCodes = append(gotCodes, f.Code)
	}
	if strings.Join(gotCodes, ",") != "a,b,c,d" {
		t.Errorf("发现该按追加顺序排列，得 %v", gotCodes)
	}
	if got := rep.Summary(); got != "error=2 warn=1 info=1" {
		t.Errorf("摘要得 %q", got)
	}
}

// TestSeverityValues 三个等级的字符串值是**对外契约**（写进 --json 的信封）。
func TestSeverityValues(t *testing.T) {
	if SevInfo != "info" || SevWarn != "warn" || SevError != "error" {
		t.Errorf("等级取值变了（--json 的信封形状跟着变）：%q %q %q", SevInfo, SevWarn, SevError)
	}
}

// TestAddAtFillsPosition 带位置的发现要能落到具体文件与行号 ——
// 报"某个 Region 被改了"没用，得指出是第几行。
func TestAddAtFillsPosition(t *testing.T) {
	base, _ := synthBase(t, "adzi999")
	rep := &Report{}
	rep.AddAt(Finding{Code: "X", Severity: SevError}, base, FileEdited, 0)
	if len(rep.Findings) != 1 {
		t.Fatalf("该记下一条，得 %d", len(rep.Findings))
	}
	f := rep.Findings[0]
	if f.File != FileEdited {
		t.Errorf("该带上文件名，得 %q", f.File)
	}
	if f.Line == 0 {
		t.Errorf("偏移 0 该换算成第 1 行，得 %d", f.Line)
	}
}
