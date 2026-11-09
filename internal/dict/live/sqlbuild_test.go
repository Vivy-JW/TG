package live

import (
	"strings"
	"testing"

	"tt/internal/dict/db"
)

// 本文件测的是**拼 SQL 那一层**（纯字符串），不是查询本身 —— 那些要真远程库，
// 这里一律不连（本包默认档里连不上任何库）。
//
// 为什么值得单独测：这一层是**注入的落点**。erpdb 走 simple protocol（没有绑定参数），
// 值只能内联，所以"用户给的字符串能不能破出字面量"全靠这几个函数。
// 真正的查询语义（列序、方言差异）由真库验证；这里钉的是**它的围墙**。

// assertQuotesEven 检查一段 SQL 里单引号的总数是偶数 —— 每个字面量都开了又关。
//
// **这是必要条件，不是充分条件。** 总数为偶数也可能被狡猾的载荷蒙混（`'a' OR '1'='1`
// 就是 6 个）。这里能立住是因为值一律经 lit() 转义后**整段包起来**，
// 所以"值没破出字面量"由 TestLitIsQuoteDoubling 那条**可逆性**断言负责；
// 这条只兜"拼起来之后有没有漏配引号"。
//
// （别把它写成"连续引号串必须是偶数" —— 那是错的：连着三个单引号是**正确**的，
// 一个开启引号加一个转义引号。第一版就是这么写错的。注意 gofmt 会把文档注释里成对的
// 单引号转成右引号，所以这里改用文字说。）
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
	"'; DROP TABLE gzze_t; --",
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
		// 内联之后，去掉两端引号、把翻倍的引号还原，应当逐字回到原值。
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
//
// 这条区别很重要：表名是标识符，没有"安全的转义"这一说。
func TestTableLitRejectsInsteadOfQuoting(t *testing.T) {
	for _, good := range []string{"dzea_t", "gzze_t", "GZOU003", "_x", "a1"} {
		if got, want := tableLit(good), "'"+good+"'"; got != want {
			t.Errorf("tableLit(%q) = %q，想 %q", good, got, want)
		}
	}
	for _, bad := range []string{
		"", "dzea_t; DROP TABLE x", "a b", "a-b", "other.dzea_t",
		"dzea_t'", "表", "a*", "1a",
	} {
		if got := tableLit(bad); got != "''" {
			t.Errorf("tableLit(%q) = %q，该退化成空字面量（%q）", bad, got, "''")
		}
	}
}

func TestKwWhere(t *testing.T) {
	t.Run("空关键字不加过滤子句", func(t *testing.T) {
		if got := kwWhere("id", "desc", ""); got != "" {
			t.Errorf("空关键字该返回空串（与本地 `?='' OR … LIKE '%%'` 等价），得 %q", got)
		}
	})
	t.Run("两端都包 UPPER，值的两端都补 %", func(t *testing.T) {
		got := kwWhere("dzea002", "dzea003", "abc")
		want := " WHERE (UPPER(dzea002) LIKE UPPER('%abc%')" +
			" OR UPPER(COALESCE(dzea003, '')) LIKE UPPER('%abc%'))"
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
			// 结构必须还是那句 WHERE，没有多出别的子句。
			if !strings.HasPrefix(got, " WHERE (UPPER(id) LIKE UPPER(") || !strings.HasSuffix(got, "))") {
				t.Errorf("载荷 %q 改变了 SQL 的形状：%q", p, got)
			}
		}
	})
}

func TestInUpperLits(t *testing.T) {
	if got := inUpperLits([]string{"0", "1"}); got != "UPPER('0'),UPPER('1')" {
		t.Errorf("得 %q", got)
	}
	if got := inUpperLits(nil); got != "" {
		t.Errorf("空列表该给空串，得 %q", got)
	}
	if got := inUpperLits([]string{}); got != "" {
		t.Errorf("空切片同上，得 %q", got)
	}
	for _, p := range injectionPayloads {
		assertQuotesEven(t, inUpperLits([]string{"ok", p}))
	}
}

// TestLiveMsgWhereIsStructured 逐条钉每个条件子句的**形状**，以及拼起来的方式。
//
// 与本地 db 包的 msgWhere 同构是刻意的（本地 SQLite 语义为蓝本，列序与条件一致），
// 所以这里的判据也用"列名 + 比较方式"来写。
func TestLiveMsgWhereIsStructured(t *testing.T) {
	t.Run("什么都不给：没有条件", func(t *testing.T) {
		if got := liveMsgWhere(db.MsgQuery{}); got != "" {
			t.Errorf("没有条件该给空串（调用方据此不加 WHERE），得 %q", got)
		}
	})
	t.Run("精确码用等号，通配码用 LIKE", func(t *testing.T) {
		got := liveMsgWhere(db.MsgQuery{Codes: []string{"azzi920"}})
		if !strings.Contains(got, "UPPER(COALESCE(gzze001, '')) = UPPER('azzi920')") {
			t.Errorf("不带通配符的码该用 =，得 %q", got)
		}
		got = liveMsgWhere(db.MsgQuery{Codes: []string{"azzi*"}})
		if !strings.Contains(got, "UPPER(COALESCE(gzze001, '')) LIKE UPPER('azzi%')") {
			t.Errorf("带 * 的码该转成 %% 并走 LIKE，得 %q", got)
		}
	})
	t.Run("多个码之间是 OR，且整体被括号包住", func(t *testing.T) {
		// 返回值是**整句**（带 " WHERE " 前缀），不是裸条件 —— 调用方直接拼在
		// `FROM gzze_t` 后面。括号是为了不与后面 AND 上来的其它条件粘连。
		got := liveMsgWhere(db.MsgQuery{Codes: []string{"a", "b"}})
		if !strings.HasPrefix(got, " WHERE (") || !strings.HasSuffix(got, ")") {
			t.Errorf("该是 WHERE + 括号包起来的 OR 组：%q", got)
		}
		if !strings.Contains(got, " OR ") {
			t.Errorf("多个码之间该是 OR：%q", got)
		}
	})
	t.Run("文字条件走 LIKE 且两端补 %%", func(t *testing.T) {
		got := liveMsgWhere(db.MsgQuery{Text: "错误"})
		if !strings.Contains(got, "UPPER(COALESCE(gzze003, '')) LIKE UPPER('%错误%')") {
			t.Errorf("得 %q", got)
		}
	})
	t.Run("多值字段都走 IN + UPPER", func(t *testing.T) {
		got := liveMsgWhere(db.MsgQuery{Types: []string{"1"}, Status: []string{"Y"}, Progs: []string{"p1", "p2"}})
		for _, want := range []string{
			"UPPER(gzze007) IN (UPPER('1'))",
			"UPPER(COALESCE(gzzestus, '')) IN (UPPER('Y'))",
			"UPPER(COALESCE(gzze005, '')) IN (UPPER('p1'),UPPER('p2'))",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("该有 %q，实得 %q", want, got)
			}
		}
	})
	t.Run("多个条件之间是 AND", func(t *testing.T) {
		got := liveMsgWhere(db.MsgQuery{Codes: []string{"a"}, Text: "x"})
		if strings.Count(got, " AND ") != 1 {
			t.Errorf("两个条件之间该是一个 AND：%q", got)
		}
	})
	t.Run("载荷破不出字面量", func(t *testing.T) {
		for _, p := range injectionPayloads {
			got := liveMsgWhere(db.MsgQuery{
				Codes: []string{p}, Text: p, Types: []string{p},
				Status: []string{p}, Progs: []string{p},
			})
			assertQuotesEven(t, got)
		}
	})
}
