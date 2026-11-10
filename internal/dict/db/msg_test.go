package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// buildMsgDB 建一个只含消息档 t_msg 的临时库（tt gp sync 从 ze_file 落的那种）。
// 数据挑的是能把条件区分开的例子：
//
//	-263 / -26300  两个编号，钉住"编号默认精确"（-263 不能把 -26300 带出来）
//	azz-00041      只有 zh_TW（测语言过滤与 QueryMsgLangs）
//	std-00006      大小写不敏感
func buildMsgDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "msg.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	stmts := []string{
		`CREATE TABLE t_msg (code TEXT, lang TEXT, text TEXT, mdate TEXT, PRIMARY KEY (code, lang))`,
		`INSERT INTO t_msg VALUES ('-263','0','資料已經被鎖住,無法更改!','2007-07-24')`,
		`INSERT INTO t_msg VALUES ('-263','2','资料已经被锁住,无法更改!','2007-07-24')`,
		`INSERT INTO t_msg VALUES ('-26300','2','API错误：不支持DISTINCT.','')`,
		`INSERT INTO t_msg VALUES ('azz-00041','0','分類碼不存在','')`,
		`INSERT INTO t_msg VALUES ('std-00006','2','输入的资料已存在','')`,
		`INSERT INTO t_msg VALUES ('azz-00110','2','密码错误','')`,
		`INSERT INTO t_msg VALUES ('azz-00115','2','此账户必须重设密码','')`,
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

// TestQueryMsgsCodeExact 编号默认精确：-263 不能把 -26300 一起捞回来。
func TestQueryMsgsCodeExact(t *testing.T) {
	d := buildMsgDB(t)

	rows, err := d.QueryMsgs(MsgQuery{Codes: []string{"-263"}, Lang: "2"})
	if err != nil {
		t.Fatalf("QueryMsgs: %v", err)
	}
	if len(rows) != 1 || rows[0].Code != "-263" {
		t.Fatalf("-263 应精确命中 1 条, got %+v", rows)
	}
	if rows[0].Text != "资料已经被锁住,无法更改!" || rows[0].Lang != "2" {
		t.Errorf("字段映射不符: %+v", rows[0])
	}

	// 写了 * 才是通配
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"-263*"}, Lang: "2"})
	if err != nil {
		t.Fatalf("QueryMsgs(-263*): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("-263* 应命中 2 条, got %+v", rows)
	}

	// 大小写不敏感
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"STD-00006"}})
	if err != nil || len(rows) != 1 {
		t.Fatalf("大写编号应命中: %+v, %v", rows, err)
	}
}

// TestQueryMsgsTextAndLang 语句条件与语言条件：两者都是 WHERE 的一部分。
func TestQueryMsgsTextAndLang(t *testing.T) {
	d := buildMsgDB(t)

	rows, err := d.QueryMsgs(MsgQuery{Text: "密码"})
	if err != nil {
		t.Fatalf("QueryMsgs(密码): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("含'密码'的消息应 2 条, got %+v", rows)
	}

	// 编号 + 语句两个条件同时成立
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"azz-*"}, Text: "密码"})
	if err != nil {
		t.Fatalf("QueryMsgs(azz-* + 密码): %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("azz-* 且含'密码'应 2 条, got %+v", rows)
	}

	// 语言过滤
	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"-263"}, Lang: "0"})
	if err != nil || len(rows) != 1 || rows[0].Text != "資料已經被鎖住,無法更改!" {
		t.Fatalf("繁体的 -263 应命中: %+v, %v", rows, err)
	}
}

// TestQueryMsgLangs 同一编号有哪些语言行。
func TestQueryMsgLangs(t *testing.T) {
	d := buildMsgDB(t)

	langs, err := d.QueryMsgLangs(MsgQuery{Codes: []string{"-263"}})
	if err != nil {
		t.Fatalf("QueryMsgLangs: %v", err)
	}
	if len(langs) != 2 || langs[0] != "0" || langs[1] != "2" {
		t.Errorf("-263 应有 0/2 两种语言: %+v", langs)
	}

	only, err := d.QueryMsgLangs(MsgQuery{Codes: []string{"azz-00041"}})
	if err != nil || len(only) != 1 || only[0] != "0" {
		t.Errorf("azz-00041 只有繁体: %+v, %v", only, err)
	}
}

// TestQueryMsgsNoCond 一个条件都不给是用法错误（否则会扫全表）。
func TestQueryMsgsNoCond(t *testing.T) {
	d := buildMsgDB(t)
	if _, err := d.QueryMsgs(MsgQuery{}); err == nil {
		t.Fatal("无条件应报错")
	}
}

// TestQueryMsgsGPFilters T100 专有的筛选维度（类型/状态/建议作业）在 GP 上没有
// 对应列：给了就返回空集，而不是悄悄忽略条件。
func TestQueryMsgsGPFilters(t *testing.T) {
	d := buildMsgDB(t)

	rows, err := d.QueryMsgs(MsgQuery{Codes: []string{"azz-*"}, Types: []string{"1"}})
	if err != nil {
		t.Fatalf("QueryMsgs(type): %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("GP 没有消息类型列,应返回空集: %+v", rows)
	}

	rows, err = d.QueryMsgs(MsgQuery{Codes: []string{"azz-*"}, Status: []string{"N"}})
	if err != nil || len(rows) != 0 {
		t.Errorf("GP 没有状态列,应返回空集: %+v, %v", rows, err)
	}
}
