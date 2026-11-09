package target

// 选择器文法（契约 2.1.2 / 2.1.4 / 2.2 / 2.4 的 F9–F11 / C2.1 / C2.3 / C2.4 / C2.5 / Q17）。
//
// TDD：本文件先写，`selector.go` 随后 —— 纯函数（解析）是最好测的一块，
// 所以每一种写法、每一种非法形态各一条断言，且每条都在注释里写明它钉住的条款编号。

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 合法写法（契约 2.2 的输入集 A 里"纯文法"那一半）
// ---------------------------------------------------------------------------

// TestParseSelectorLegalForms 逐条钉住 2.1.4 的四条文法与 C2.4 的别名分流。
func TestParseSelectorLegalForms(t *testing.T) {
	cases := []struct {
		raw  string
		kind SelectorKind
		env  string
		prog string
		ent  string
		n    int
		why  string
	}{
		// 写法 ④（2.2）：规范 id = 环境#序号（C2.0 的"生命周期内恒定"那个形态）。
		{"恒烁正式区#1", SelectorRef, "恒烁正式区", "", "", 1, "C2.1.2：规范 id"},
		{"测试1012#12", SelectorRef, "测试1012", "", "", 12, "C2.1.2：序号是十进制正整数"},
		{"带 空格的环境#3", SelectorRef, "带 空格的环境", "", "", 3, "C2.1：环境名精确取值，允许空格"},
		// 写法 ⑤（2.2）：环境/作业#序号 —— 作业段只是"选择方式"（3.0），不改变序号作用域。
		{"恒烁正式区/apmt520#1", SelectorEnvJobSeq, "恒烁正式区", "apmt520", "", 1, "2.1.4 + 3.0"},
		// 写法 ⑥（2.2）：环境/作业。
		{"恒烁正式区/apmt520", SelectorEnvJob, "恒烁正式区", "apmt520", "", 0, "2.1.4"},
		{"恒烁正式区/A_1-x", SelectorEnvJob, "恒烁正式区", "A_1-x", "", 0, "B29 的作业名字符集"},
		// 写法 ⑦（2.2）：环境/企业编号 —— C2.3 的判据是"第二段全数字"。
		{"恒烁正式区/99", SelectorEnvEnt, "恒烁正式区", "", "99", 0, "C2.3：全数字 → 企业编号"},
		{"恒烁正式区/0", SelectorEnvEnt, "恒烁正式区", "", "0", 0, "C2.3：判据只看全数字，不校验它合不合理"},
		// 写法 ⑧（2.2）：环境 —— 但**光看字面**它与别名同形（C2.4 的分流只看 / 与 #），
		// 所以解析层先按别名记下；知道环境清单之后才分成环境（见 TestParseSelectorWithEnvNames）。
		{"恒烁正式区", SelectorAlias, "", "", "", 0, "C2.4 的字面分流（写法 ⑧ 由 promoteEnvName 判）"},
		// 写法 ③（2.2）：别名 —— 不含 / 与 # 的一律按别名（C2.4 的落地判据）。
		{"汇率单", SelectorAlias, "", "", "", 0, "C2.4：不含 / 与 #"},
		{"复现 1", SelectorAlias, "", "", "", 0, "Q17：别名允许空格"},
		{"HuiLvDan", SelectorAlias, "", "", "", 0, "Q17：别名允许 ASCII"},
		// C2.4 的分界：`1a` 不含 / 与 # → 别名；它**不会**被当成 `环境/1a` 或企业编号。
		{"1a", SelectorAlias, "", "", "", 0, "C2.4：分流只看有没有 / 与 #"},
	}
	for _, c := range cases {
		got, f := ParseSelector(c.raw)
		if f != nil {
			t.Errorf("ParseSelector(%q) 该合法（%s），却报 %s：%s", c.raw, c.why, f.Code, f.Message)
			continue
		}
		if got.Kind != c.kind || got.Env != c.env || got.Prog != c.prog || got.Ent != c.ent || got.N != c.n {
			t.Errorf("ParseSelector(%q) = {kind=%v env=%q prog=%q ent=%q n=%d}，期望 {kind=%v env=%q prog=%q ent=%q n=%d}（%s）",
				c.raw, got.Kind, got.Env, got.Prog, got.Ent, got.N, c.kind, c.env, c.prog, c.ent, c.n, c.why)
		}
		// C5.6：选择器必须**原样**回显（requested.session 用它）。
		if got.Raw != c.raw {
			t.Errorf("ParseSelector(%q).Raw = %q —— C5.6 要求原样回显，不许规范化", c.raw, got.Raw)
		}
		if c.kind == SelectorAlias && got.Alias != c.raw {
			t.Errorf("ParseSelector(%q).Alias = %q —— C2.5 末句：别名**按原样**参与寻址与比对",
				c.raw, got.Alias)
		}
	}
}

// TestParseSelectorAliasWithHashOrSlashIsASelector 钉住 C2.4 的落地判据：
// `--session` 的取值里**含 `/` 或 `#`** 的一律按选择器解析，不含这两个字符的才按别名。
func TestParseSelectorAliasWithHashOrSlashIsASelector(t *testing.T) {
	for _, raw := range []string{"a/b", "a#1", "环境/作业#1"} {
		got, f := ParseSelector(raw)
		if f != nil {
			t.Errorf("ParseSelector(%q) 该按**选择器**解析并成功，却报 %s：%s", raw, f.Code, f.Message)
			continue
		}
		if got.Kind == SelectorAlias {
			t.Errorf("ParseSelector(%q) 被判成别名 —— C2.4：含 / 或 # 的一律按选择器解析，\n"+
				"    否则 `环境/作业#序号` 与别名之间的优先级会变成一条没人记得住的规则", raw)
		}
	}
}

// TestParseSelectorWithEnvNames 钉住写法 ⑧（2.2）与 C2.4 那条缝的补法：
// 不带分隔符的取值，**精确等于一个已配置的环境名**时按"按环境找"（写法 ⑧），否则按别名（写法 ③）。
//
// 为什么需要这条规则：C2.4 的分流只看 `/` 与 `#`，而写法 ⑧ 与别名 ③ 字面同形。
// 不补这条，2.2 的 ⑧ 与 F4 就永远不可达（解析层与落位层用的是同一个函数，见 promoteEnvName）。
func TestParseSelectorWithEnvNames(t *testing.T) {
	envs := []string{"恒烁正式区", "测试1012"}

	got, f := ParseSelectorWith("恒烁正式区", envs)
	if f != nil {
		t.Fatalf("写法 ⑧ 该合法，却报 %s：%s", f.Code, f.Message)
	}
	if got.Kind != SelectorEnv || got.Env != "恒烁正式区" || got.Raw != "恒烁正式区" {
		t.Errorf("带环境清单时 `恒烁正式区` 该判成写法 ⑧（按环境找），实得 %+v", got)
	}

	// 不在清单里的一律还是别名（写法 ③）—— 别名允许与作业名/企业编号同形（Q17）。
	if got, f := ParseSelectorWith("汇率单", envs); f != nil || got.Kind != SelectorAlias {
		t.Errorf("`汇率单` 不在环境清单里，该按别名（写法 ③），实得 %+v / %v", got, f)
	}
	// C2.1：精确匹配，不做前缀、不做大小写折叠。
	for _, raw := range []string{"恒烁正式", "恒烁正式区X", "hengshuo"} {
		if got, _ := ParseSelectorWith(raw, envs); got.Kind != SelectorAlias {
			t.Errorf("C2.1：%q 与配置里的环境名只是像，该按别名处理，实得 kind=%v", raw, got.Kind)
		}
	}
	// 没有清单（纯解析层）时按字面记成别名 —— 落位时再判一次。
	if got, _ := ParseSelectorWith("恒烁正式区", nil); got.Kind != SelectorAlias {
		t.Errorf("没有环境清单时该按字面记成别名（落位时由 Resolve 再判），实得 kind=%v", got.Kind)
	}
}

// ---------------------------------------------------------------------------
// 非法写法（契约 2.4 的 F9 / F10 / F11）
// ---------------------------------------------------------------------------

// TestParseSelectorF9SequenceMustBeDecimalPositive 钉住 F9：
// `#x` / `#1.5` / `#` / `#0` / `#-1`（**以及 `#01`**）都是**用法错**（USAGE，退 2），
// 不是"找不到会话"。
//
// `#01` 也在这一条里：`debugContractRefPattern`（规范的 id 文法）**刻意不允许前导零** ——
// `#01` 与 `#1` 是同一个地址，写成两个写法对不上账（WP2 的常量注释已把这件事写明）。
func TestParseSelectorF9SequenceMustBeDecimalPositive(t *testing.T) {
	for _, raw := range []string{
		"恒烁正式区/apmt520#x", "恒烁正式区/apmt520#1.5", "恒烁正式区/apmt520#",
		"恒烁正式区/apmt520#0", "恒烁正式区/apmt520#-1", "恒烁正式区/apmt520#01",
		"恒烁正式区#x", "恒烁正式区#0",
	} {
		_, f := ParseSelector(raw)
		if f == nil {
			t.Errorf("ParseSelector(%q) 该被拒（F9：序号非十进制正整数）", raw)
			continue
		}
		if f.Code != CodeUsage || f.Exit != ExitUsage {
			t.Errorf("ParseSelector(%q) 的码 = %s / 退 %d，F9 要的是 %s / 退 %d",
				raw, f.Code, f.Exit, CodeUsage, ExitUsage)
		}
		if !strings.Contains(f.Message, "序号必须是 >=1 的十进制整数") {
			t.Errorf("ParseSelector(%q) 的文案 = %q，F9 钉住的形状是「选择器里的序号必须是 >=1 的十进制整数：\"#…\"」",
				raw, f.Message)
		}
		// F9 的候选列是"不出现"（2.4 那张表）。
		if len(f.Candidates) != 0 {
			t.Errorf("ParseSelector(%q) 带出了 %d 条候选 —— F9 的候选列是不出现", raw, len(f.Candidates))
		}
	}
}

// TestParseSelectorF10Grammar 钉住 F10：段数不对 / 空段都是**用法错**。
func TestParseSelectorF10Grammar(t *testing.T) {
	for _, raw := range []string{
		"恒烁正式区/作业/多一段",   // 段数不对
		"恒烁正式区//apmt520", // 空段
		"恒烁正式区/",         // 空段
		"/apmt520",       // 环境名为空
		"#1",             // 环境名为空
		"恒烁正式区#1#2",      // 两个序号段
		"恒烁正式区/99#1",     // 文法里 `环境/企业编号` 不带序号段
		"a/b/c/d",        // 段数不对
	} {
		_, f := ParseSelector(raw)
		if f == nil {
			t.Errorf("ParseSelector(%q) 该被拒（F10：选择器文法非法）", raw)
			continue
		}
		if f.Code != CodeUsage || f.Exit != ExitUsage {
			t.Errorf("ParseSelector(%q) 的码 = %s / 退 %d，F10 要的是 %s / 退 %d",
				raw, f.Code, f.Exit, CodeUsage, ExitUsage)
		}
		if !strings.Contains(f.Message, "选择器文法非法") {
			t.Errorf("ParseSelector(%q) 的文案 = %q，F10 钉住的形状是「选择器文法非法：\"…\"（写法：环境 | 环境/作业 | 环境/作业#序号 | 环境/企业编号）」",
				raw, f.Message)
		}
		if !strings.Contains(f.Message, "环境/作业#序号") {
			t.Errorf("ParseSelector(%q) 的文案里没写出那四种写法（F10 的文案形状）：%q", raw, f.Message)
		}
		if len(f.Candidates) != 0 {
			t.Errorf("ParseSelector(%q) 带出了 %d 条候选 —— F10 的候选列是不出现", raw, len(f.Candidates))
		}
	}
}

// TestParseSelectorF11EmptyStringIsGrammarIllegal 钉住 F11：
// `--session ""` 按**文法非法**处理，**不是**"省略"。
//
// 这就是 `ParseSelector` 与 `Request.Selector == nil` 分工的地方：
// nil（没给这个标志）才是省略；给了空串走这里，退 2。
func TestParseSelectorF11EmptyStringIsGrammarIllegal(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t"} {
		_, f := ParseSelector(raw)
		if f == nil {
			t.Fatalf("ParseSelector(%q) 该被拒（F11：空串按文法非法处理，不当成省略）", raw)
		}
		if f.Code != CodeUsage || f.Exit != ExitUsage {
			t.Errorf("ParseSelector(%q) 的码 = %s / 退 %d，F11 要的是 %s / 退 %d",
				raw, f.Code, f.Exit, CodeUsage, ExitUsage)
		}
	}
}

// ---------------------------------------------------------------------------
// C2.3 的第二段判据（与 WP2 的 contract_debug_test.go 同源，落在生产侧）
// ---------------------------------------------------------------------------

// TestSecondSegmentKindIsAllDigits 钉住 C2.3 + B29：
// 第二段**全数字 → 企业编号；否则 → 作业名**，两种形态不会撞（作业名首字符是字母）。
func TestSecondSegmentKindIsAllDigits(t *testing.T) {
	for _, s := range []string{"99", "1012", "0", "1"} {
		if got := SecondSegmentKind(s); got != SecondSegEnt {
			t.Errorf("第二段 %q 该判成企业编号（C2.3：全数字），得 %q", s, got)
		}
	}
	// 判据**不**先做 B29 的合法性校验：`1a` 这类串按 C2.3 落到作业那一支，
	// 随后在"环境/作业"那一步按 F7 报 SESSION_NOT_FOUND。
	// 它**不会**被静默当成企业编号 —— 那才是"看起来落上了、其实落在别处"。
	for _, s := range []string{"apmt520", "bsft001", "a", "A_1-x", "1a", "99x", "作业"} {
		if got := SecondSegmentKind(s); got != SecondSegJob {
			t.Errorf("C2.3 的判据只看「全数字」：第二段 %q 含非数字字符，该判成作业名（%q），得 %q",
				s, SecondSegJob, got)
		}
	}
	if got := SecondSegmentKind(""); got != "" {
		t.Errorf("空段的形态 = %q，该是空串（它走 F10 的文法非法，不走 C2.3 的分流）", got)
	}
}

// ---------------------------------------------------------------------------
// --name 的取值纪律（C2.4 / C2.5 / F19 / Q17）
// ---------------------------------------------------------------------------

// TestValidateAliasRules 钉住 Q17 定稿：别名**非空**（去两侧空白后仍非空）、
// 去两侧空白后 1–64 个字符、**只排除 `/` 与 `#`**；中文 / 空格 / 标点一律允许。
func TestValidateAliasRules(t *testing.T) {
	for _, ok := range []string{
		"汇率单", "复现 1", "HuiLvDan", "a", strings.Repeat("z", 64),
		"带-标点,。的 名字", "1", "  两侧有空白  ",
	} {
		if f := ValidateAlias(ok); f != nil {
			t.Errorf("别名 %q 该合法（C2.5 / Q17），却报 %s：%s", ok, f.Code, f.Message)
		}
	}
	for _, bad := range []string{
		"", "   ", "\t", "a/b", "a#b", "/", "#", "复现/1", "复现#1",
		strings.Repeat("z", 65),
	} {
		f := ValidateAlias(bad)
		if f == nil {
			t.Errorf("别名 %q 该被拒（C2.4 / C2.5 / F19）", bad)
			continue
		}
		if f.Code != CodeUsage || f.Exit != ExitUsage {
			t.Errorf("别名 %q 的码 = %s / 退 %d，F19 要的是 %s / 退 %d",
				bad, f.Code, f.Exit, CodeUsage, ExitUsage)
		}
		if !strings.Contains(f.Message, `"/"`) || !strings.Contains(f.Message, `"#"`) {
			t.Errorf("别名 %q 的文案 = %q，F19 钉住的形状是「别名不能为空、也不能含 \"/\" 或 \"#\"」",
				bad, f.Message)
		}
	}
}

// TestAliasLengthCountsRunesNotBytes 钉住 C2.5 的"1–64 个字符"是**字符**不是字节：
// 64 个汉字合法，65 个不合法。用字节数算的话，22 个汉字就会被误拒。
func TestAliasLengthCountsRunesNotBytes(t *testing.T) {
	if f := ValidateAlias(strings.Repeat("汇", 64)); f != nil {
		t.Errorf("64 个汉字该合法（C2.5：1–64 个**字符**），却报 %s：%s", f.Code, f.Message)
	}
	if f := ValidateAlias(strings.Repeat("汇", 65)); f == nil {
		t.Error("65 个汉字该被拒（C2.5：去空白后长度 1–64 个字符）")
	}
}

// TestSameAliasIsCaseInsensitive 钉住 C2.5 的查重口径：
// "两个只差大小写的同名别名算撞车"（F5）。
func TestSameAliasIsCaseInsensitive(t *testing.T) {
	if !SameAlias("HuiLvDan", "huilvdan") {
		t.Error("C2.5：两个只差大小写的同名别名该算撞车（查重不区分大小写）")
	}
	if !SameAlias("汇率单", "汇率单") {
		t.Error("同名该算撞车")
	}
	if SameAlias("汇率单", "HuiLvDan") {
		t.Error("C2.5：`汇率单` 与 `HuiLvDan` 视为**两个**名字，按原样比对 —— 不该算撞车")
	}
}
