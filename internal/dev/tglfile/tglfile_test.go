package tglfile

import (
	"bytes"
	"testing"
)

// 本包处理 `.tgl` 骨架里的标记：区段边界、插入点占位符、集合锚点。
// 全是纯字节函数（输入一段 TGL，输出结构或改写后的字节），所以全是表驱动单测。
//
// 判据的来源是**设计器的真实行为**（包注释里逐条指了反编译源码的位置），
// 所以下面凡是"看起来奇怪"的地方都特意写清"这是照抄设计器，不是我们的选择"。

// 一份最小的 TGL：一个普通区段 + 三个锚点区段（锚点带 readonly="Y"）+ 一个占位符。
func sampleTGL() []byte {
	return []byte("#頭\r\n" +
		`{<section id="prog.main" type="s" >}` + "\r\n" +
		"MAIN\r\n" +
		`{<point name="main.define" edit="c"/>}` + "\r\n" +
		"END MAIN\r\n" +
		"{</section>}\r\n" +
		`{<section id="prog.other_function" readonly="Y" type="s" >}` + "\r\n" +
		`{<point name="other.function"/>}` + "\r\n" +
		"{</section>}\r\n" +
		`{<section id="prog.other_dialog" readonly="Y" type="s" >}` + "\r\n" +
		`{<point name="other.dialog"/>}` + "\r\n" +
		"{</section>}\r\n" +
		`{<section id="prog.other_report" readonly="Y" type="s" >}` + "\r\n" +
		`{<point name="other.report"/>}` + "\r\n" +
		"{</section>}\r\n")
}

func TestTrimEnd(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"不动的原样返回", "abc", "abc"},
		{"尾部空格与制表都去掉", "abc \t ", "abc"},
		{"尾部 CRLF / LF 都去掉", "abc\r\n", "abc"},
		{"CR 单独出现也去掉", "abc\r", "abc"},
		{"纵向制表与换页也算空白（设计器用的是 char.IsWhiteSpace）", "abc\x0b\x0c", "abc"},
		{"全是空白就剩空", " \r\n\t ", ""},
		{"空进空出", "", ""},
		{"只去尾部 —— 首部与中间的空白原样", "  a b  ", "  a b"},
		{"正文里的换行不能动（只 TrimEnd）", "a\nb\n", "a\nb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := string(TrimEnd([]byte(c.in))); got != c.want {
				t.Errorf("TrimEnd(%q) = %q，想 %q", c.in, got, c.want)
			}
		})
	}
}

func TestFindSections(t *testing.T) {
	secs, err := FindSections(sampleTGL())
	if err != nil {
		t.Fatalf("该能配对：%v", err)
	}
	if len(secs) != 4 {
		t.Fatalf("该找到 4 个区段，得 %d", len(secs))
	}
	if secs[0].ID != "prog.main" || secs[1].ID != "prog.other_function" {
		t.Errorf("区段 id 或顺序不对：%q %q", secs[0].ID, secs[1].ID)
	}
	// readonly="Y" 只出现在锚点区段上。
	if secs[0].ReadonlyMarker {
		t.Error("prog.main 没写 readonly，不该标成只读")
	}
	for i := 1; i < 4; i++ {
		if !secs[i].ReadonlyMarker {
			t.Errorf("第 %d 个区段写了 readonly=\"Y\"，该标成只读", i)
		}
	}
	// 区间要自洽：正文夹在两个标记之间，整段含两个标记。
	s := secs[0]
	if s.BodyStart != s.StartMarker.End || s.BodyEnd != s.EndMarker.Start {
		t.Errorf("正文字节区间与标记区间对不上：body=[%d,%d) mark=[%d,%d)+[%d,%d)",
			s.BodyStart, s.BodyEnd, s.StartMarker.Start, s.StartMarker.End, s.EndMarker.Start, s.EndMarker.End)
	}
	if s.Start != s.StartMarker.Start || s.End != s.EndMarker.End {
		t.Error("整段区间该正好包住两个标记")
	}
	if !bytes.Contains(SectionBody(sampleTGL(), s), []byte("MAIN")) {
		t.Errorf("正文该含 MAIN，得 %q", SectionBody(sampleTGL(), s))
	}
}

// TestFindSectionsReadonlyIsCaseInsensitive 正则带 (?i)、值用 EqualFold：
// readonly="y" 与 readonly="Y" 同义（设计器如此）。
func TestFindSectionsReadonlyIsCaseInsensitive(t *testing.T) {
	for _, v := range []string{"Y", "y"} {
		tgl := []byte(`{<section id="a" readonly="` + v + `" >}` + "\n" + "{</section>}")
		secs, err := FindSections(tgl)
		if err != nil {
			t.Fatal(err)
		}
		if len(secs) != 1 || !secs[0].ReadonlyMarker {
			t.Errorf("readonly=%q 该算只读", v)
		}
	}
	tgl := []byte(`{<section id="a" readonly="N" >}` + "\n" + "{</section>}")
	secs, _ := FindSections(tgl)
	if secs[0].ReadonlyMarker {
		t.Error(`readonly="N" 不该算只读`)
	}
}

// TestFindSectionsUnpaired 数量不等就报 FormatError（退出码 2）——
// 设计器遇到这种 TGL 会抛「区段错误」，我们照抄成一条可读的错。
func TestFindSectionsUnpaired(t *testing.T) {
	tgl := []byte(`{<section id="a" >}` + "\n" + `{<section id="b" >}` + "\n" + "{</section>}")
	_, err := FindSections(tgl)
	if err == nil {
		t.Fatal("起止数量不等该报错")
	}
	var fe *FormatError
	if !asFormatError(err, &fe) {
		t.Fatalf("该是 *FormatError，得 %T", err)
	}
	if fe.ExitCode() != 2 {
		t.Errorf("格式错误的退出码该是 2，得 %d", fe.ExitCode())
	}
	// 详情要把两个计数都报出来 —— 只说"不配对"的话，人得自己去数。
	msg := err.Error()
	if !bytes.Contains([]byte(msg), []byte("2 次")) || !bytes.Contains([]byte(msg), []byte("1 次")) {
		t.Errorf("错误里该带上两个计数，得 %q", msg)
	}
}

// asFormatError 是 errors.As 的小包装（本包只关心这一种，省得每处都 import errors）。
func asFormatError(err error, target **FormatError) bool {
	if fe, ok := err.(*FormatError); ok {
		*target = fe
		return true
	}
	return false
}

// TestFindPlaceholdersIsCaseSensitive 这一条与区段标记**不同**：占位符的正则
// 没有 (?i)（设计器里就是大小写敏感的），所以 `{<POINT .../>}` 不算占位符。
func TestFindPlaceholdersIsCaseSensitive(t *testing.T) {
	tgl := sampleTGL()
	got := FindPlaceholders(tgl)
	if len(got) != 4 {
		t.Fatalf("样例里有 4 个占位符（1 个普通 + 3 个锚点里的），得 %d", len(got))
	}
	if got[0].Name != "main.define" {
		t.Errorf("第一个占位符名不对：%q", got[0].Name)
	}
	// Raw 必须与原文逐字节相同 —— 改回 TGL 时靠它原样写回。
	if len(got[0].Raw) == 0 || !bytes.Contains(tgl, got[0].Raw) {
		t.Error("Raw 该是原文里的那段字节")
	}

	// 反面：大小写变了就不认（与区段标记相反，这是设计器的真实差别）。
	upper := []byte(`{<POINT name="x"/>}`)
	if n := len(FindPlaceholders(upper)); n != 0 {
		t.Errorf("占位符判定是大小写敏感的，实得 %d 个", n)
	}
	if n := len(FindPlaceholders([]byte(`{<point name="x" />}`))); n != 1 {
		t.Errorf("自闭合前的空格该容许，实得 %d 个", n)
	}
}

// TestFindAnchorAndTheDotIsNotADot 锚点正则里捕获组的 `.` 是**任意字符**，
// 所以 other_function / otherXfunction 都会命中 —— 这是照抄设计器（CodeEditorManager
// 的 ProcessFunctionTypes 就是这么写的），不是我们的选择。特意钉住，免得有人"顺手修正"。
func TestFindAnchorAndTheDotIsNotADot(t *testing.T) {
	for _, name := range []string{"other.function", "other_function", "otherXfunction", "other-funcTION"} {
		tgl := []byte(`{<point name="` + name + `"/>}`)
		if m := FindAnchor(tgl, AnchorFunction); m == nil {
			t.Errorf("name=%q 该命中 function 锚点（那个 . 是任意字符）", name)
		}
	}
	// 类型必须对上：function 的锚点不该被 dialog 认领。
	tgl := []byte(`{<point name="other.function"/>}`)
	if m := FindAnchor(tgl, AnchorDialog); m != nil {
		t.Errorf("other.function 不该命中 dialog 锚点，得 %q", m.Name)
	}
	if m := FindAnchor(tgl, AnchorFunction); m == nil || m.Name != "other.function" {
		t.Error("该命中 function 锚点并带回捕获到的名字")
	}
	// 没有锚点就是 nil（调用方据此跳过），不是空 Marker。
	if m := FindAnchor([]byte("# 什么都没有\n"), AnchorFunction); m != nil {
		t.Errorf("没有锚点该返回 nil，得 %+v", m)
	}
}

func TestReplaceAnchor(t *testing.T) {
	t.Run("命中就换掉，返回 true", func(t *testing.T) {
		tgl := []byte("前\n" + `{<point name="other.function"/>}` + "\n后")
		got, ok := ReplaceAnchor(tgl, AnchorFunction, []byte("【新】"))
		if !ok {
			t.Fatal("该报告命中")
		}
		if bytes.Contains(got, []byte("other.function")) || !bytes.Contains(got, []byte("【新】")) {
			t.Errorf("没换掉：%q", got)
		}
		if !bytes.Contains(got, []byte("前")) || !bytes.Contains(got, []byte("后")) {
			t.Errorf("不该动锚点以外的字节：%q", got)
		}
	})
	t.Run("没命中就原样返回，返回 false", func(t *testing.T) {
		tgl := []byte("# 什么都没有\n")
		got, ok := ReplaceAnchor(tgl, AnchorReport, []byte("x"))
		if ok {
			t.Error("没命中不该报告命中")
		}
		if !bytes.Equal(got, tgl) {
			t.Errorf("没命中该原样返回，得 %q", got)
		}
	})
	t.Run("**所有**出现都被替换（设计器用的是 regex.Replace）", func(t *testing.T) {
		tgl := []byte(`{<point name="other.function"/>}` + "\n" + `{<point name="other.function"/>}`)
		got, ok := ReplaceAnchor(tgl, AnchorFunction, []byte("X"))
		if !ok || bytes.Contains(got, []byte("other.function")) {
			t.Errorf("该把两处都换掉：%q", got)
		}
	})
}

// TestReplaceAnchorExpandsDollarInRepl 钉住一件**注释与代码不符**的事。
//
// ReplaceAnchor 的注释写着"用 ReplaceAllLiteral 避免 repl 里的 $ 被当作替换模板"，
// 而代码调的是 `re.ReplaceAll(tgl, repl)` —— 那个**会**把 repl 里的 `$1`/`$name` 当模板展开。
// 所以 `$` 确实有影响，注释说的那层保护并不存在。
//
// 实践上不会出事：repl 是插回去的锚点标记（点名里不允许 `$`）。这里把**实际行为**钉下来，
// 免得后来人照注释去信任一层不存在的保护。（要不要改成 ReplaceAllLiteral 是另一件事，
// 改之前得先确认设计器的行为 —— 注释的原意可能正是想跟设计器一致。）
func TestReplaceAnchorExpandsDollarInRepl(t *testing.T) {
	tgl := []byte(`{<point name="other.function"/>}`)
	got, ok := ReplaceAnchor(tgl, AnchorFunction, []byte("$1"))
	if !ok {
		t.Fatal("该命中")
	}
	// $1 是捕获组 1（整个 `{<point ...>}`）—— 所以换出来的是原文自己，不是字面 "$1"。
	if bytes.Equal(got, []byte("$1")) {
		t.Error("repl 里的 $1 被当字面量了 —— 那说明用的是 ReplaceAllLiteral，注释是对的，请更新本测试")
	}
	if !bytes.Contains(got, []byte("{<point")) {
		t.Errorf("$1 该展开成捕获组（整段标记），得 %q", got)
	}

	// 字面量美元符要写 $$ 才能原样出来 —— 这正是"会展开"的证据。
	got2, _ := ReplaceAnchor(tgl, AnchorFunction, []byte("a$$b"))
	if string(got2) != "a$b" {
		t.Errorf("$$ 该展开成一个 $（说明确实在做模板展开），得 %q", got2)
	}
}

func TestAnchorSectionID(t *testing.T) {
	cases := []struct {
		prog, id string
		want     AnchorKind
		ok       bool
	}{
		{"adzi999", "adzi999.other_function", AnchorFunction, true},
		{"adzi999", "adzi999.other_dialog", AnchorDialog, true},
		{"adzi999", "adzi999.other_report", AnchorReport, true},
		{"adzi999", "adzi999.main", "", false},
		{"adzi999", "别的程序.other_function", "", false}, // 程序名必须对上
		{"adzi999", "adzi999.other_other", "", false},
		{"", ".other_function", AnchorFunction, true}, // 空程序名也算：拼接结果对得上就行
	}
	for _, c := range cases {
		got, ok := AnchorSectionID(c.prog, c.id)
		if got != c.want || ok != c.ok {
			t.Errorf("AnchorSectionID(%q, %q) = (%q, %v)，想 (%q, %v)", c.prog, c.id, got, ok, c.want, c.ok)
		}
	}
}

// TestPatchSection 复刻 GenerateTGL 的补丁格式：startMarker + eol + content + eol + endMarker。
//
// eol 由调用方给（设计器用 Environment.NewLine）—— 这里给什么都不该影响它被原样采用。
func TestPatchSection(t *testing.T) {
	tgl := sampleTGL()
	got, err := PatchSection(tgl, "prog.main", []byte("新的正文"), []byte("\r\n"))
	if err != nil {
		t.Fatalf("该能打上：%v", err)
	}
	if !bytes.Contains(got, []byte(`{<section id="prog.main" type="s" >}`)) {
		t.Error("起始标记该原样保留")
	}
	if !bytes.Contains(got, []byte("\r\n新的正文\r\n")) {
		t.Errorf("该是 标记 + eol + 正文 + eol + 标记：%q", got)
	}
	// 标记本身在补丁后必须只剩一处，别把旧正文留在原地。
	if n := bytes.Count(got, []byte("MAIN\r\n")); n != 0 {
		t.Errorf("旧正文该被换掉，实得 %d 处：%q", n, got)
	}
	// 别的区段一个字节都不该动。
	for _, id := range []string{"prog.other_function", "prog.other_dialog"} {
		if !bytes.Contains(got, []byte(`id="`+id+`"`)) {
			t.Errorf("不该动 %s", id)
		}
	}

	// eol 换成 LF，格式跟着变 —— 说明它确实是参数而不是写死的。
	gotLF, err := PatchSection(tgl, "prog.main", []byte("X"), []byte("\n"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(gotLF, []byte("\r\nX\r\n")) {
		t.Error("给了 LF 就不该出现 CRLF 包裹")
	}
	if !bytes.Contains(gotLF, []byte("\nX\n")) {
		t.Errorf("该用 LF 包裹，得 %q", gotLF)
	}
}

func TestPatchSectionMissingIsFormatError(t *testing.T) {
	_, err := PatchSection(sampleTGL(), "prog.不存在的区段", []byte("x"), []byte("\n"))
	if err == nil {
		t.Fatal("找不到区段该报错")
	}
	var fe *FormatError
	if !asFormatError(err, &fe) {
		t.Fatalf("该是 *FormatError，得 %T", err)
	}
	// 报错要点名是**哪个**区段找不到 —— 否则一长串 id 里没法定位。
	if !bytes.Contains([]byte(err.Error()), []byte("prog.不存在的区段")) {
		t.Errorf("该带上区段 id，得 %q", err.Error())
	}
	// 不配对时也要报错（透传 FindSections 的错）。
	if _, err := PatchSection([]byte(`{<section id="a" >}`), "a", nil, []byte("\n")); err == nil {
		t.Error("区段不配对也该报错")
	}
}

func TestFormatErrorRendering(t *testing.T) {
	if got := (&FormatError{Msg: "一句话"}).Error(); got != "一句话" {
		t.Errorf("没有详情时不该多出分隔符，得 %q", got)
	}
	got := (&FormatError{Msg: "一句话", Detail: []string{"甲", "乙"}}).Error()
	if got != "一句话：甲; 乙" {
		t.Errorf("得 %q，想 %q", got, "一句话：甲; 乙")
	}
}
