package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildMsgDB 建一个只含消息档两张表(gzze_t 消息档 / gzzal_t 作业名称)的临时库。
// 数据挑的是几个能把条件区分开的例子:
//
//	-263 / -26300  两个 SQLCODE,用来钉住"编号默认精确"(-263 不能把 -26300 带出来)
//	azz-00041      只有 zh_TW(用来测语言条件的过滤与 QueryMsgLangs)
//	azz-00110      带建议作业 azzi920,且 gzzal_t 有该作业的 zh_CN/zh_TW 名称
//	三种 gzze007(1/1/1/1/1/2/0)与两种 gzzestus(Y/N),让 --type/--status 有得筛
func buildMsgDB(t *testing.T, withGzzal bool) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "msg.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE gzze_t (gzzestus TEXT, gzze001 TEXT, gzze002 TEXT, gzze003 TEXT,
			gzze004 TEXT, gzze005 TEXT, gzze006 TEXT, gzze007 TEXT, gzze008 TEXT)`,
		`INSERT INTO gzze_t VALUES ('Y','-263','zh_CN','资料已经被锁住,无法更改!','请稍后再试',':EXEPROG','','1','N')`,
		`INSERT INTO gzze_t VALUES ('Y','-26300','zh_CN','API错误：不支持DISTINCT.','',':EXEPROG','','2','N')`,
		`INSERT INTO gzze_t VALUES ('Y','azz-00041','zh_TW','分類碼不存在','',':EXEPROG','','1','N')`,
		`INSERT INTO gzze_t VALUES ('Y','std-00006','zh_CN','输入的资料已存在','请重新输入','baa-00010','程序人员细节','1','N')`,
		`INSERT INTO gzze_t VALUES ('Y','azz-00110','zh_CN','密码错误','','azzi920','','1','N')`,
		`INSERT INTO gzze_t VALUES ('N','azz-00115','zh_CN','此账户必须重设密码','',':EXEPROG','','0','Y')`,
	}
	if withGzzal {
		stmts = append(stmts,
			`CREATE TABLE gzzal_t (gzzal001 TEXT, gzzal002 TEXT, gzzal003 TEXT)`,
			`INSERT INTO gzzal_t VALUES ('azzi920','zh_CN','信息维护作业')`,
			`INSERT INTO gzzal_t VALUES ('baa-00010','zh_CN','币别维护作业')`,
		)
	}
	for _, s := range stmts {
		if _, err := conn.Exec(s); err != nil {
			t.Fatalf("setup (%s): %v", s, err)
		}
	}
	if err := conn.Close(); err != nil {
		t.Fatalf("close setup conn: %v", err)
	}
	d, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(d.Close)
	return d
}

// TestMsgLike 模糊查询的入参转换(* 与 % 当通配,其余子串)。
func TestMsgLike(t *testing.T) {
	cases := []struct{ in, want string }{
		{"密码", "%密码%"},
		{"azz-*", "azz-%"},
		{"*-00006", "%-00006"},
		{"%密码%", "%密码%"},
	}
	for _, c := range cases {
		if got := MsgLike(c.in); got != c.want {
			t.Errorf("MsgLike(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestQueryMsgsCodeExact 编号默认精确:-263 不能把 -26300 一起捞回来
// (azzi920 的编号栏是子串,CLI 刻意反过来,见 db/msg.go 顶部说明)。
func TestQueryMsgsCodeExact(t *testing.T) {
	d := buildMsgDB(t, true)

	rows, err := d.QueryMsgs(MsgQuery{Codes: []string{"-263"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs: %v", err)
	}
	if len(rows) != 1 || rows[0].Code != "-263" {
		t.Fatalf("-263 应精确命中 1 条, got %+v", rows)
	}
	if rows[0].Text != "资料已经被锁住,无法更改!" || rows[0].TypeCode != "1" || rows[0].Status != "Y" {
		t.Errorf("字段映射不符: %+v", rows[0])
	}

	// 写了 * 才是通配
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"-263*"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(-263*): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("-263* 应命中 2 条, got %+v", rows)
	}

	// 大小写不敏感(远程 UPPER 两端,本地对齐)
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"STD-00006"}, Lang: "zh_CN"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("大写编号应命中: %+v, %v", rows, err)
	}
}

// TestQueryMsgsTextAndLang 语句条件与语言条件:两者都是 WHERE 的一部分。
func TestQueryMsgsTextAndLang(t *testing.T) {
	d := buildMsgDB(t, true)

	// 语句子串匹配:只有 azz-00110 / azz-00115 含"密码"
	rows, err := d.QueryMsgs(MsgQuery{Text: "密码", Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(密码): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("含'密码'的 zh_CN 消息应 2 条, got %+v", rows)
	}

	// 编号 + 语句两个条件同时成立
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"azz-*"}, Text: "密码", Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(azz-* + 密码): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("组合条件应 2 条, got %+v", rows)
	}
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"std-*"}, Text: "密码", Lang: "zh_CN"})
	if err != nil || len(rows) != 0 {
		t.Fatalf("组合条件应 0 条(std- 下没有含'密码'的), got %+v, %v", rows, err)
	}

	// 语言是硬条件:azz-00041 只有 zh_TW,查 zh_CN 取不到
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"azz-00041"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(azz-00041/zh_CN): %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("azz-00041 没有 zh_CN 行,应 0 条, got %+v", rows)
	}
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"azz-00041"}, Lang: "zh_TW"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("azz-00041 的 zh_TW 行应命中: %+v, %v", rows, err)
	}
}

// TestQueryMsgsProgName 建议作业的名称来自 gzzal_t;缺表时静默降级为空。
func TestQueryMsgsProgName(t *testing.T) {
	d := buildMsgDB(t, true)
	rows, err := d.QueryMsgs(MsgQuery{Codes: []string{"azz-00110"}, Lang: "zh_CN"})
	if err != nil || len(rows) != 1 {
		t.Fatalf("QueryMsgs: %+v, %v", rows, err)
	}
	if rows[0].ExecProg != "azzi920" || rows[0].ProgName != "信息维护作业" {
		t.Errorf("建议作业名称不符: %+v", rows[0])
	}

	// gzzal_t 不存在:主结果照给,名称留空
	noNames := buildMsgDB(t, false)
	rows, err = noNames.QueryMsgs(MsgQuery{Codes: []string{"azz-00110"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(无 gzzal_t): %v", err)
	}
	if len(rows) != 1 || rows[0].ProgName != "" {
		t.Errorf("gzzal_t 缺失时应静默降级: %+v", rows)
	}
}

// TestQueryMsgLangs 语言清单:同条件去掉语言,用于"该编号没有该语言行"的提示。
func TestQueryMsgLangs(t *testing.T) {
	d := buildMsgDB(t, true)

	langs, err := d.QueryMsgLangs(MsgQuery{Codes: []string{"azz-00041"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgLangs: %v", err)
	}
	if len(langs) != 1 || langs[0] != "zh_TW" {
		t.Fatalf("azz-00041 可用语言应只有 zh_TW, got %v", langs)
	}

	// 不存在的编号:空清单(命令层据此说"未找到"而不是"换语言")
	langs, err = d.QueryMsgLangs(MsgQuery{Codes: []string{"zzz-99999"}, Lang: "zh_CN"})
	if err != nil || len(langs) != 0 {
		t.Fatalf("不存在的编号应返回空清单: %v, %v", langs, err)
	}
}

// TestQueryMsgsByTypeStatusProg 三个专用条件:类型/状态/建议作业。
// 同一个 flag 内是 OR,不同 flag 之间是 AND。
func TestQueryMsgsByTypeStatusProg(t *testing.T) {
	d := buildMsgDB(t, true)

	// 类型:zh_CN 下 type=1 三条、type=2 一条、type=0 一条
	for _, c := range []struct {
		typ  string
		want int
	}{{"1", 3}, {"2", 1}, {"0", 1}} {
		rows, err := d.QueryMsgs(MsgQuery{Types: []string{c.typ}, Lang: "zh_CN"})
		if err != nil {
			t.Fatalf("QueryMsgs(type=%s): %v", c.typ, err)
		}
		if len(rows) != c.want {
			t.Errorf("type=%s 应 %d 条, got %d: %+v", c.typ, c.want, len(rows), rows)
		}
	}

	// 类型多选 = OR:type 1,2 共 4 条(3+1)
	rows, err := d.QueryMsgs(MsgQuery{Types: []string{"1", "2"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(type=1,2): %v", err)
	}
	if len(rows) != 4 {
		t.Errorf("type 1,2 应 4 条(OR), got %d", len(rows))
	}

	// 状态:Y 四条,N 一条(azz-00115)
	rows, err = d.QueryMsgs(MsgQuery{Status: []string{"N"}, Lang: "zh_CN"})
	if err != nil {
		t.Fatalf("QueryMsgs(status=N): %v", err)
	}
	if len(rows) != 1 || rows[0].Code != "azz-00115" {
		t.Errorf("status=N 应只命中 azz-00115, got %+v", rows)
	}
	// 大小写不敏感(与编号同一个规矩:UPPER 交给 SQL)
	if rows, err = d.QueryMsgs(MsgQuery{Status: []string{"y"}, Lang: "zh_CN"}); err != nil || len(rows) != 4 {
		t.Errorf("status=y 应与 Y 同效: %d 条, %v", len(rows), err)
	}

	// 建议作业:精确匹配。:EXEPROG 是"无建议作业"的哨兵值,照常可查
	rows, err = d.QueryMsgs(MsgQuery{Progs: []string{"azzi920"}, Lang: "zh_CN"})
	if err != nil || len(rows) != 1 || rows[0].Code != "azz-00110" {
		t.Errorf("prog=azzi920 应命中 azz-00110: %+v, %v", rows, err)
	}
	if rows, err = d.QueryMsgs(MsgQuery{Progs: []string{"baa-00010"}, Lang: "zh_CN"}); err != nil ||
		len(rows) != 1 || rows[0].Code != "std-00006" {
		t.Errorf("prog=baa-00010 应命中 std-00006: %+v, %v", rows, err)
	}
	if rows, err = d.QueryMsgs(MsgQuery{Progs: []string{"axmt500"}, Lang: "zh_CN"}); err != nil || len(rows) != 0 {
		t.Errorf("prog=axmt500 应 0 条: %+v, %v", rows, err)
	}

	// 跨 flag 是 AND:type=1 且语句含"密码"只剩 azz-00110;换成 type=2 就该为空
	// (空这一条是在钉住它是 AND 而不是退化成 OR)
	rows, err = d.QueryMsgs(MsgQuery{Types: []string{"1"}, Text: "密码", Lang: "zh_CN"})
	if err != nil || len(rows) != 1 || rows[0].Code != "azz-00110" {
		t.Errorf("type=1 + 语句含密码 应只剩 azz-00110: %+v, %v", rows, err)
	}
	if rows, err = d.QueryMsgs(MsgQuery{Types: []string{"2"}, Text: "密码", Lang: "zh_CN"}); err != nil || len(rows) != 0 {
		t.Errorf("type=2 + 语句含密码 应 0 条: %+v, %v", rows, err)
	}

	// 状态与类型一起给(AND):停用 且 警告 = azz-00115
	rows, err = d.QueryMsgs(MsgQuery{Types: []string{"0"}, Status: []string{"N"}, Lang: "zh_CN"})
	if err != nil || len(rows) != 1 || rows[0].Code != "azz-00115" {
		t.Errorf("type=0 + status=N 应命中 azz-00115: %+v, %v", rows, err)
	}
}

// TestQueryMsgsNoCond 空条件拒绝:消息档 3 万条,不设条件的全捞没有意义。
// 语言别不算筛子(只给 --lang 等于没筛),但类型/状态/建议作业算。
func TestQueryMsgsNoCond(t *testing.T) {
	d := buildMsgDB(t, true)
	if _, err := d.QueryMsgs(MsgQuery{Lang: "zh_CN"}); err == nil {
		t.Error("只有语言条件应被拒绝(语言不算筛选条件)")
	}
	if _, err := d.QueryMsgLangs(MsgQuery{Lang: "zh_CN"}); err == nil {
		t.Error("QueryMsgLangs 空条件应被拒绝")
	}
	if rows, err := d.QueryMsgs(MsgQuery{Types: []string{"1"}, Lang: "zh_CN"}); err != nil || len(rows) == 0 {
		t.Errorf("只给类型是合法的筛子,应当放行: %d 条, %v", len(rows), err)
	}
}
