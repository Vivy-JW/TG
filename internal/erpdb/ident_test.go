package erpdb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"tt/internal/dbconfig"
)

// conn 造一个只填了本文件关心的那几个字段的连接（用户名词典表要的是 User/Type/Host）。
func conn(user, typ, host string) dbconfig.Connection {
	return dbconfig.Connection{Type: typ, Host: host, User: user}
}

// 本包管的是**怎么连 ERP 库**：连接（要真库，不测）、标识符与字面量转义（纯函数，这里测）、
// 只读 SQL 的文本闸门（纯函数，这里测）。
//
// 转义这一层是**安全关键**：远端库的账号密码是明文配置、查询里拼的是用户给的表名与关键字，
// 一处漏了就成注入。所以下面的判据写得比"能跑"严：**拒绝**的那些样例才是重点。
//
// **口令一律是假的**：这里用 "pw-dev" 只为分辨取到了哪个值。

func TestValidIdent(t *testing.T) {
	valid := []string{"dzea_t", "DS", "a", "A1", "_x", "t35prd", "gzou003", "a_b_c", "x9"}
	for _, s := range valid {
		if !ValidIdent(s) {
			t.Errorf("%q 该算合法标识符", s)
		}
	}
	invalid := []struct{ in, why string }{
		{"", "空"},
		{"9a", "数字开头"},
		{"0", "数字开头"},
		{"a-b", "连字符"},
		{"a b", "空格"},
		{"a.b", "点（限定名要拆开各自校验，不能整个放行）"},
		{"a;b", "分号"},
		{"a'b", "单引号"},
		{`a"b`, "双引号"},
		{"a(b)", "括号"},
		{"a*", "星号"},
		{"a%", "百分号"},
		{"a/../b", "路径式"},
		{"表", "非 ASCII"},
		{"a\nb", "换行"},
		{"a\tb", "制表符"},
		{"a--", "SQL 注释符"},
		{"a/*x*/", "SQL 块注释"},
	}
	for _, c := range invalid {
		if ValidIdent(c.in) {
			t.Errorf("%q 该被拒（%s）", c.in, c.why)
		}
	}

	// 保留字是**合法标识符** —— 本函数只管字符集，不做 SQL 关键字过滤。
	// 这条特意写出来，免得后来人以为"防注入"到这里就做完了。
	for _, kw := range []string{"drop", "select", "table", "insert"} {
		if !ValidIdent(kw) {
			t.Errorf("%q 在字符集上合法（本函数不过滤保留字），实得 false", kw)
		}
	}
}

// TestQuoteIdentReturnsEmptyOnInvalid 不过校验**返回空串**而不是原样返回 ——
// 拼进 SQL 会得到 "SELECT * FROM .t"，报的错与真实原因（表名非法）完全对不上。
func TestQuoteIdentReturnsEmptyOnInvalid(t *testing.T) {
	if got := QuoteIdent("dzea_t"); got != "dzea_t" {
		t.Errorf("合法标识符该原样返回，得 %q", got)
	}
	for _, bad := range []string{"", "a-b", "a;b", "表", "1a"} {
		if got := QuoteIdent(bad); got != "" {
			t.Errorf("非法标识符 %q 该返回空串，得 %q", bad, got)
		}
	}
}

// TestQuoteLitDoublesSingleQuotes 字面量转义只有一条规则：单引号翻倍。
//
// 这是**唯一**挡住值注入的东西（erpdb 走 simple protocol，没有绑定参数）。
func TestQuoteLitDoublesSingleQuotes(t *testing.T) {
	cases := map[string]string{
		"":             "''",
		"abc":          "'abc'",
		"O'Brien":      "'O''Brien'",
		"'; DROP--":    "'''; DROP--'",
		"''":           "''''''",
		"a'b'c":        "'a''b''c'",
		"%":            "'%'", // 通配符不加处理：它由调用方按 LIKE 语义自己决定
		`a"b`:          `'a"b'`,
		"line1\nline2": "'line1\nline2'",
	}
	for in, want := range cases {
		if got := QuoteLit(in); got != want {
			t.Errorf("QuoteLit(%q) = %q，想 %q", in, got, want)
		}
	}
	// 转义后的结果再"翻倍回去"应当还原：值里不可能出现落单的单引号。
	for in := range cases {
		got := QuoteLit(in)
		inner := got[1 : len(got)-1]
		if strings.Contains(strings.ReplaceAll(inner, "''", ""), "'") {
			t.Errorf("QuoteLit(%q) 的结果里有落单的单引号：%q", in, got)
		}
	}
}

// TestSelectAllSQL 表名与用户名都要过白名单 —— 用户名来自配置、表名可能来自用户。
func TestSelectAllSQL(t *testing.T) {
	got, err := SelectAllSQL(conn("ds", "oracle", "db1"), "dzea_t")
	if err != nil {
		t.Fatalf("该成功：%v", err)
	}
	if want := "SELECT * FROM ds.dzea_t"; got != want {
		t.Errorf("得 %q，想 %q", got, want)
	}

	bad := []struct{ name, user, table string }{
		{"表名带分号", "ds", "dzea_t; DROP TABLE x"},
		{"表名带空格", "ds", "dzea t"},
		{"表名带引号", "ds", "dzea_t'"},
		{"表名为空", "ds", ""},
		{"用户名带连字符", "d-s", "dzea_t"},
		{"用户名为空（连接没填凭据）", "", "dzea_t"},
		{"表名是限定名（点号要拆开各自校验，不能整个放行）", "ds", "other.dzea_t"},
	}
	for _, c := range bad {
		t.Run(c.name, func(t *testing.T) {
			if _, err := SelectAllSQL(conn(c.user, "oracle", "db1"), c.table); err == nil {
				t.Error("该被拒")
			}
		})
	}
}

// **本批已删掉的那条测试（留注释，不留代码）**：它从前叫
// `TestCheckReadOnlySQLIsAPrefixCheckNotAParser`，钉的是"前缀匹配有多弱" ——
// `TrimSpace`+`ToUpper` 看开头 + 看分号，其中一条断言是 `/* x */ DROP TABLE t` **会绕过**。
//
// 本次这个位置换成了 `checkReadOnlyStatement`（委托 `internal/safesql.Check`，词法级：
// 先剥注释，再按词边界/标识符前缀判），那一半断言当场反向 —— 整条测试按新性质重写成
// 下面三条：
//
//   - `TestCheckReadOnlySQLRejectsWritesAndMultipleStatements`（该拒的都拒，含前导注释绕过）
//   - `TestCheckReadOnlySQLAllowsReadOnlySelects`（该放的都放，宁可不误杀）
//   - `TestCheckReadOnlyStatementBlocksTheReadOnlyTransactionBypass`（**两层合起来才成立**）
//
// 为什么删掉而不是留一条 `t.Skip`：跳过的测试与通过的测试在报告里长得一样无害，
// 所以本仓有一条**跳过台账**（`internal/testkit/skipledger_test.go`）盯着每一处跳过。
// "旧断言作废"这件事不值得在台账里长期占一行 —— 判据已经在下面三条里，留着只是噪音。

// TestCheckReadOnlySQLRejectsWritesAndMultipleStatements 钉住文本层能挡住的那半边：
// 写操作（含**前导注释**那种绕过写法）与多语句，一条都不许放过去。
//
// 为什么必须换掉前缀匹配：`erpdb` 的库侧只读事务是本批新加的（金仓
// `default_transaction_read_only`、Oracle `SET TRANSACTION READ ONLY`），而它是
// **会话级/事务级**的状态 —— 一句 `SELECT set_config('default_transaction_read_only','off',false)`
// 就能把金仓那层拆掉。挡住那一句的**只有文本层**（见下面那条测试）。
// 留着前缀匹配 = 给纸门加了把锁。
func TestCheckReadOnlySQLRejectsWritesAndMultipleStatements(t *testing.T) {
	bad := []string{
		// ① 写操作直接贴进来（大小写不敏感、前面有空白）
		"INSERT INTO t VALUES (1)",
		"insert into t values (1)",
		"  \tUPDATE t SET a = 1",
		"DELETE FROM t",
		"DROP TABLE t",
		"ALTER TABLE t ADD c NUMBER",
		"CREATE TABLE t (a NUMBER)",
		"TRUNCATE TABLE t",
		"MERGE INTO t USING s ON (1=1) WHEN MATCHED THEN UPDATE SET a=1",
		"GRANT SELECT ON t TO u",
		"REVOKE SELECT ON t FROM u",
		"COMMIT",
		"CALL p()",
		"BEGIN NULL; END;",
		"SELECT 1 INTO v FROM dual",
		// ② **前导注释**的绕过写法：这一条是旧前缀匹配漏掉的现场
		"/* x */ DROP TABLE t",
		"-- x\nDELETE FROM t",
		// ③ 多语句（中间的分号）
		"SELECT 1; DROP TABLE t",
		"SELECT 1; SET SESSION CHARACTERISTICS AS TRANSACTION READ WRITE",
		// ④ 行锁与特权视图 / 带副作用的函数（只读查询不该碰）
		"SELECT * FROM t FOR UPDATE",
		"SELECT * FROM dba_users",
		"SELECT pg_read_file('/etc/passwd')",
		// ⑤ 不是查询（sqlplus 客户端命令）
		"HOST rm -rf /tmp/x",
	}
	for _, sql := range bad {
		if err := checkReadOnlyStatement(sql); err == nil {
			t.Errorf("%q 该被拒（文本层是真检查，不是前缀匹配）", sql)
		}
	}
}

// TestCheckReadOnlySQLAllowsReadOnlySelects 是上面那条的**反面判据**：
// 普通的只读查询不许被误杀 —— 误杀的代价是"字典查不出东西"，而它与"数据不存在"
// 在输出上一模一样（`internal/safesql` 的包注释专门警告过这种静默退化）。
//
// 样例覆盖本包**真实**的两个调用面：`SelectAllSQL` 给 `tt dict db sync` 的
// `SELECT * FROM <user>.<table>`，以及 `tt dict … --ent` 那条
// `SELECT gzou001, gzou003 FROM gzou_t …`（`internal/cli/dict/entacct.go`）。
func TestCheckReadOnlySQLAllowsReadOnlySelects(t *testing.T) {
	good := []string{
		"SELECT * FROM ds.dzea_t",
		"select 1 from dual",
		"WITH x AS (SELECT 1) SELECT * FROM x",
		"SELECT * FROM t WHERE a = 'O''Brien'",
		`SELECT gzou001, gzou003 FROM gzou_t WHERE gzoustus='Y' ORDER BY gzou001`,
		// 字面量里出现禁用词不算命中（`safesql` 会先把字面量掏空再扫）——
		// 值过 `QuoteLit` 之后长这样，误杀它会让"查一个叫 delete 的关键字"变成查不到。
		"SELECT * FROM dzea_t WHERE dzea002 = 'delete from t'",
		"SELECT * FROM dzea_t WHERE dzea002 = 'set_config'",
	}
	for _, sql := range good {
		if err := checkReadOnlyStatement(sql); err != nil {
			t.Errorf("%q 该放行（只读查询），实得：%v", sql, err)
		}
	}
}

// TestCheckReadOnlyStatementBlocksTheReadOnlyTransactionBypass 钉住**两层合起来才成立**
// 的那一条，也是本批最要紧的一条判据：
//
//	SELECT set_config('default_transaction_read_only','off',false)
//
// 金仓那层只读是**会话级** GUC，这一句能把它改回 off —— 改完之后库侧兜底当场不存在。
// 所以它必须被**文本层**挡住（`internal/safesql` 的禁用前缀里就有 `set_config`，
// 注释原话："能在一条 SELECT 里把 default_transaction_read_only 改回 off，直接拆掉只读事务"）。
//
// 反向也一并钉住：`current_setting` 是同一个信息面，同样不许。
//
// 为什么这条不能只写在 README 里：它是"两层"这个说法成立与否的**唯一**机械判据 ——
// 少一半，README 里那句"两层防线"就又变回一句空话（本次要修的正是这个）。
func TestCheckReadOnlyStatementBlocksTheReadOnlyTransactionBypass(t *testing.T) {
	bypasses := []struct{ sql, why string }{
		{"SELECT set_config('default_transaction_read_only','off',false)",
			"金仓那层只读是会话级 GUC，这一句把它关掉 = 拆掉库侧兜底"},
		{"SELECT current_setting('default_transaction_read_only')",
			"会话信息面，只读查询不需要它"},
	}
	for _, c := range bypasses {
		err := checkReadOnlyStatement(c.sql)
		if err == nil {
			t.Errorf("该被拒：%s\n    %s\n    拒它的必须是文本层 —— 库侧只读事务拦不住「把自己关掉」的语句", c.sql, c.why)
			continue
		}
		// 报错要点名是哪条规则（`safesql` 刻意不静默退化成空集），便于人照着改。
		if !strings.Contains(err.Error(), "set_config") && !strings.Contains(err.Error(), "current_setting") {
			t.Errorf("拒绝理由该点名那条受限函数，实得 %q", err.Error())
		}
	}
}

// TestRepoDictQueryLiteralsPassTheRealCheck 用**仓库里真实写着的**语句再验一次：
// 换闸门之后，`internal/dict/live` 与 `internal/cli/dict` 里那些 SELECT 一条都不许被误杀。
//
// 为什么要扫源码而不是手抄样例：手抄的样例有一天会与生产脱节，而"闸门把真查询拒了"
// 的现场表现是**静默**的（字典查不出东西，与"数据不存在"长得一模一样 ——
// `internal/safesql` 的包注释专门警告过这种退化）。这条扫描是机械的：
// 那两个目录下每一段以 `SELECT` / `WITH` 开头的**反引号字符串字面量**都要过闸门。
//
// 覆盖面与它的边界（如实写）：
//   - 抓的是**整段**字面量（那正是 live 那一路的写法：`sql := \`SELECT …\``，拼值在末尾
//     用 `+ lit(x)`），所以扫到的就是真正送进 `erpdb.Query` 的那条语句；
//   - `internal/dict/db`（本地 SQLite 镜像）不在范围内：它不走 `erpdb`，闸门也管不着它；
//   - 反面判据在下面：扫到的条数必须够多，否则这条测试是空转。
func TestRepoDictQueryLiteralsPassTheRealCheck(t *testing.T) {
	root := repoRootOf(t)
	dirs := []string{
		filepath.Join(root, "internal", "dict", "live"),
		filepath.Join(root, "internal", "cli", "dict"),
	}
	re := regexp.MustCompile("(?s)`([^`]*)`")
	checked := 0
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join(dir, "*.go"))
		if err != nil {
			t.Fatalf("扫 %s 失败：%v", dir, err)
		}
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("读 %s 失败：%v", f, err)
			}
			rel, _ := filepath.Rel(root, f)
			for _, m := range re.FindAllStringSubmatch(string(b), -1) {
				body := strings.TrimSpace(m[1])
				head := strings.ToUpper(firstToken(body))
				if head != "SELECT" && head != "WITH" {
					continue
				}
				checked++
				if err := checkReadOnlyStatement(body); err != nil {
					t.Errorf("%s 里的这条只读查询被文本层拒了：\n%q\n→ %v\n"+
						"（闸门把真查询拒掉的现场表现是字典静默查不出东西，必须改闸门，不是改语句）",
						filepath.ToSlash(rel), body, err)
				}
			}
		}
	}
	if checked < 10 {
		t.Fatalf("只扫到 %d 条 SELECT/WITH 字面量 —— 扫描空转了（定位或正则错了），"+
			"这条「不误杀」的断言等于没跑", checked)
	}
}

// firstToken 取文本的第一个词（大写由调用方做）。
func firstToken(s string) string {
	if i := strings.IndexAny(s, " \t\r\n("); i >= 0 {
		return s[:i]
	}
	return s
}

// TestDebugPathDoesNotImportErpdb 钉住"本批改动**一个字都动不到 `tt debug sql`**"。
//
// `tt debug sql`（→ `POST /api/dbsql` → `internal/debug/dbsql.go` 的 `RunReadonlyQuery`）
// 走的是**经 SSH 用服务器侧 sqlplus / ksql 的 stdin 脚本**那条路，库侧只读**已经在脚本里**
// （`oraReadonlyScript` 的 `SET TRANSACTION READ ONLY`、`kbReadonlyScript` 的
// `SET SESSION CHARACTERISTICS AS TRANSACTION READ ONLY`）。它**不 import 本包**，
// 所以本批给 `erpdb` 加的那两层（金仓连接参数 / Oracle 包事务）与它没有交集。
//
// 这条测试把上面那句散文变成断言：**`internal/debug` 里出现 `tt/internal/erpdb` 就是红的**。
// 有人哪天把 `tt debug sql` 改成走客户端直连，这条会当场提醒他：那条通道的只读防线
// 与超时/限行/错误文案**全都不一样**，不能顺手换。
//
// 反面判据在下面：扫描必须真的扫到文件，否则这条等于空转。
func TestDebugPathDoesNotImportErpdb(t *testing.T) {
	root := repoRootOf(t)
	dir := filepath.Join(root, "internal", "debug")
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("扫 %s 失败：%v", dir, err)
	}
	if len(files) < 10 {
		t.Fatalf("只扫到 %d 个 .go —— 定位错了，这条断言等于空转", len(files))
	}
	const mod = "tt/internal/erpdb"
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读 %s 失败：%v", f, err)
		}
		if !strings.Contains(string(b), `"`+mod+`"`) {
			continue
		}
		rel, _ := filepath.Rel(root, f)
		t.Errorf("%s 引用了 %s —— `tt debug sql` 走的是 SSH + sqlplus/ksql 那条路，"+
			"它的库侧只读、超时、限行与错误文案都住在 internal/debug 自己的 dbsql.go 里。\n"+
			"    换成客户端直连等于把那条通道的防线整套换掉，不能顺手做（要做得单独一批）。",
			filepath.ToSlash(rel), mod)
	}
}

// repoRootOf 从本文件所在目录上溯到含 go.mod 的那一级。
//
// 不 import `internal/testkit`：那会把"测试助手"依赖进这个安全关键包
// （而这个包的生产代码只依赖标准库 + 两个驱动）。上溯十行就够了。
func repoRootOf(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("取工作目录失败：%v", err)
	}
	for i := 0; i < 12; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("从 %s 上溯没找到 go.mod —— 扫描会扫空目录", dir)
	return ""
}

// TestServerVersionBypassesOnlyTheTextLayer 钉住"检查只做一遍、且做在唯一那个入口上"
// 这个刻意的例外，免得有人以为 `ServerVersion` 忘了接闸门。
//
// `SELECT version()` 与 `SELECT banner FROM v$version …` 都是**代码字面量**，
// 不是调用方给的语句；而后者里的 `v$` 会被文本层当特权视图拒掉（那是对的规则，
// 只是不适用于"服务端版本"这条例外）。所以两个 `ServerVersion` 都走不带文本检查的
// 执行体；**库侧只读照旧生效**（Oracle 那条走的就是包事务的那个函数）。
func TestServerVersionBypassesOnlyTheTextLayer(t *testing.T) {
	if err := checkReadOnlyStatement("SELECT banner FROM v$version WHERE ROWNUM = 1"); err == nil {
		t.Errorf("特权视图 `v$*` 该被文本层拒（它是跨会话的元数据面）—— 上面那个例外才有存在意义")
	}
	if err := checkReadOnlyStatement("SELECT version()"); err != nil {
		t.Errorf("`SELECT version()` 是普通只读查询，不该被拒：%v", err)
	}
}

func TestGoOraDSNEscapesEveryComponent(t *testing.T) {
	got := goOraDSN("db1", 1521, "t35prd", "ds", "pw-dev")
	if want := "oracle://ds:pw-dev@db1:1521/t35prd"; got != want {
		t.Errorf("得 %q，想 %q", got, want)
	}
	// 两端都要能塞进去：DSN 是拼出来的字符串，口令里的 @ 与 : 会改变它解析出什么。
	got = goOraDSN("db1", 1521, "svc", "u@h", "p:w")
	if want := "oracle://u%40h:p%3Aw@db1:1521/svc"; got != want {
		t.Errorf("特殊字符该被转义，得 %q，想 %q", got, want)
	}
}

func TestURLEscape(t *testing.T) {
	cases := map[string]string{
		"plain": "plain",
		"p@ss":  "p%40ss",
		"p:ss":  "p%3Ass",
		"a/b":   "a%2Fb",
		"a?b":   "a%3Fb",
		"100%":  "100%25",
		"":      "",
		"a b":   "a b",   // 空格不转义：本函数只管 DSN 里会改变解析的那几个字符
		"%40":   "%2540", // 先转 % 再转别的，所以已是转义序列的输入不会被二次解释
	}
	for in, want := range cases {
		if got := urlEscape(in); got != want {
			t.Errorf("urlEscape(%q) = %q，想 %q", in, got, want)
		}
	}
}

// TestKingbaseDSN 金仓那条用的是 key=value 形式，与 oracle 的 URL 形式不同。
func TestKingbaseDSN(t *testing.T) {
	c := conn("ds", "kingbase", "db1")
	c.Port = 54321
	c.Database = "topprd"
	c.User = "ds"
	c.Password = "pw-dev"
	got := kingbaseDSN(c)
	for _, want := range []string{"host=db1", "port=54321", "dbname=topprd", "user=ds", "password=pw-dev"} {
		if !strings.Contains(got, want) {
			t.Errorf("DSN 里该有 %q，实得 %q", want, got)
		}
	}
}

func TestScanString(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{nil, ""},
		{"abc", "abc"},
		{[]byte("abc"), "abc"},
		{"", ""},
		{int64(42), "42"},     // 驱动可能给非字符串
		{float64(1.5), "1.5"}, // 不带多余的尾零
		{true, "true"},
		{time.Date(2026, 9, 26, 19, 30, 0, 0, time.UTC), "2026-09-26 19:30:00"},
	}
	for _, c := range cases {
		if got := scanString(c.in); got != c.want {
			t.Errorf("scanString(%v) = %q，想 %q", c.in, got, c.want)
		}
	}
}
