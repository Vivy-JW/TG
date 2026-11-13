package live

import (
	"strings"
	"testing"

	"tg/internal/dict/db"
)

// 本文件测的是**拼 SQL 那一层**（纯字符串），不是查询本身 —— 那些要真远程库，
// 这里一律不连（本包默认档里连不上任何库）。
//
// 为什么值得单独测：这一层是**注入的落点**。erpdb 走 simple protocol（没有绑定参数），
// 值只能内联，所以"用户给的字符串能不能破出字面量"全靠这几个函数。
// 真正的查询语义（列序、方言差异）由真库验证；这里钉的是**它的围墙**。

// assertQuotesEven 检查一段 SQL 里单引号的总数是偶数 —— 每个字面量都开了又关。
//
// **这是必要条件，不是充分条件。** 总数为偶数也可能被狡猾的载荷蒙混。这里能立住
// 是因为值一律经 lit() 转义后**整段包起来**，所以"值没破出字面量"由
// TestLitIsQuoteDoubling 那条**可逆性**断言负责；这条只兜"拼起来之后有没有漏配引号"。
func assertQuotesEven(t *testing.T, sql string) {
	t.Helper()
	if n := strings.Count(sql, "'"); n%2 != 0 {
		t.Fatalf("单引号总数是奇数（%d）—— 有字面量没关上：%q", n, sql)
	}
}

// 一组"想破墙"的载荷。它们不该改变 SQL 的结构 —— 只会变成字面量里的内容。
var injectionPayloads = []string{
	"'",
	"''",
	"'; DROP TABLE zta_file; --",
	"' OR '1'='1",
	"%' OR 1=1 --",
	"a'b'c",
	"\\'",
	"`",
	"\"",
	"\x00",
	"'; EXEC xp_cmdshell('dir'); --",
}

func TestLitIsQuoteDoubling(t *testing.T) {
	if got := lit("abc"); got != "'abc'" {
		t.Errorf("得 %q，想 'abc'", got)
	}
	for _, p := range injectionPayloads {
		got := lit(p)
		assertQuotesEven(t, got)
		if len(got) < 2 || got[0] != '\'' || got[len(got)-1] != '\'' {
			t.Fatalf("lit(%q) = %q 该是单引号包起来的字面量", p, got)
		}
		inner := strings.ReplaceAll(got[1:len(got)-1], "''", "'")
		if inner != p {
			t.Errorf("还原不回去：lit(%q) 的内层是 %q", p, inner)
		}
	}
}

// TestTableLitRejectsInsteadOfQuoting 表名走的是**白名单**，不是转义 ——
// 非法名直接退化成空字面量（查不到、不报错），而不是"转义后照样拼进去"。
func TestTableLitRejectsInsteadOfQuoting(t *testing.T) {
	for _, good := range []string{"ima_file", "zta_file", "IMA01", "_x", "a1"} {
		if got, want := tableLit(good), "'"+good+"'"; got != want {
			t.Errorf("tableLit(%q) = %q，想 %q", good, got, want)
		}
	}
	for _, bad := range []string{
		"", "ima_file; DROP TABLE x", "a b", "a-b", "other.ima_file",
		"ima_file'", "表", "a*", "1a",
	} {
		if got := tableLit(bad); got != "''" {
			t.Errorf("tableLit(%q) = %q，该退化成空字面量（%q）", bad, got, "''")
		}
	}
}

func TestKwWhere(t *testing.T) {
	t.Run("空关键字不加过滤子句", func(t *testing.T) {
		if got := kwWhere("id", "desc", ""); got != "" {
			t.Errorf("空关键字该返回空串，得 %q", got)
		}
	})
	t.Run("两端都包 UPPER，值的两端都补 %", func(t *testing.T) {
		got := kwWhere("z.zta01", "g.gat03", "abc")
		want := " AND (UPPER(z.zta01) LIKE UPPER('%abc%')" +
			" OR UPPER(COALESCE(g.gat03, '')) LIKE UPPER('%abc%'))"
		if got != want {
			t.Errorf("得 %q\n想 %q", got, want)
		}
	})
	t.Run("关键字里的通配符原样进字面量", func(t *testing.T) {
		got := kwWhere("id", "desc", "a*b")
		if !strings.Contains(got, "'%a*b%'") {
			t.Errorf("通配符该原样保留（LIKE 语义交给库），得 %q", got)
		}
	})
	t.Run("想注入的关键字破不出字面量", func(t *testing.T) {
		for _, p := range injectionPayloads {
			got := kwWhere("id", "desc", p)
			assertQuotesEven(t, got)
			if !strings.HasPrefix(got, " AND (UPPER(id) LIKE UPPER(") || !strings.HasSuffix(got, "))") {
				t.Errorf("载荷 %q 改变了 SQL 的形状：%q", p, got)
			}
		}
	})
}

// TestProgNameCol 语言 → gaz_file.gaz02 的取值：zh_TW 取繁体（0），其余取简体（2）。
func TestProgNameCol(t *testing.T) {
	cases := map[string]string{
		"zh_TW": "0", "tw": "0", "0": "0",
		"zh_CN": "2", "": "2", "2": "2", "whatever": "2",
	}
	for in, want := range cases {
		if got := progNameCol(in); got != want {
			t.Errorf("progNameCol(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestMsgWhereGPIsStructured 逐条钉 ze_file 每个条件子句的形状。
func TestMsgWhereGPIsStructured(t *testing.T) {
	t.Run("什么都不给：没有条件", func(t *testing.T) {
		if got := (&Live{}).msgWhereGP(db.MsgQuery{}); got != "" {
			t.Errorf("没有条件该给空串，得 %q", got)
		}
	})
	t.Run("精确码用等号，通配码用 LIKE", func(t *testing.T) {
		got := (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{"-263"}})
		if !strings.Contains(got, "UPPER(ze01) = UPPER('-263')") {
			t.Errorf("不带通配符的码该用 =，得 %q", got)
		}
		got = (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{"-263*"}})
		if !strings.Contains(got, "UPPER(ze01) LIKE UPPER('-263%')") {
			t.Errorf("带 * 的码该转成 %% 并走 LIKE，得 %q", got)
		}
	})
	t.Run("多个码之间是 OR，且整体被括号包住", func(t *testing.T) {
		got := (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{"a", "b"}})
		if !strings.HasPrefix(got, " WHERE (") || !strings.HasSuffix(got, ")") {
			t.Errorf("该是 WHERE + 括号包起来的 OR 组：%q", got)
		}
		if !strings.Contains(got, " OR ") {
			t.Errorf("多个码之间该是 OR：%q", got)
		}
	})
	t.Run("文字条件走 LIKE 且两端补 %", func(t *testing.T) {
		got := (&Live{}).msgWhereGP(db.MsgQuery{Text: "错误"})
		if !strings.Contains(got, "UPPER(ze03) LIKE UPPER('%错误%')") {
			t.Errorf("得 %q", got)
		}
	})
	t.Run("语言条件是等号", func(t *testing.T) {
		got := (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{"-263"}, Lang: "2"})
		if !strings.Contains(got, "ze02 = '2'") {
			t.Errorf("得 %q", got)
		}
	})
	t.Run("多个条件之间是 AND", func(t *testing.T) {
		got := (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{"a"}, Text: "x"})
		if strings.Count(got, " AND ") != 1 {
			t.Errorf("两个条件之间该是一个 AND：%q", got)
		}
	})
	t.Run("载荷破不出字面量", func(t *testing.T) {
		for _, p := range injectionPayloads {
			got := (&Live{}).msgWhereGP(db.MsgQuery{Codes: []string{p}, Text: p, Lang: p})
			assertQuotesEven(t, got)
		}
	})
}

// TestParseCount COUNT(*) 的字符串结果转整数。
func TestParseCount(t *testing.T) {
	for in, want := range map[string]int{"0": 0, "42": 42, " 7 ": 7, "": 0} {
		if got, err := parseCount(in); err != nil || got != want {
			t.Errorf("parseCount(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := parseCount("12x"); err == nil {
		t.Error("非数字应报错")
	}
}
