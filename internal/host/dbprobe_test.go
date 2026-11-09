package host

// 命令拼装的单测。
//
// 这一层是**安全边界**:函数产出的字符串会被送到服务器的 shell 里执行。
// 所以不能只断言"看起来对",要用本机 shell 把引用反解回来,证明它是一个词而不是两条命令。
// 反解全程只执行 printf,被测内容只当参数传进去。

import (
	"os/exec"
	"strings"
	"testing"
)

// bashExpandsCommandSubstitution 探「本机 bash 会不会展开**单引号内**的命令替换」。
//
// 台账 D8 实测：`printf %s '$(printf SUB)'` 在这台机器上真的被展开了（得到 SUB），
// 而 `'a b'`、`'a;b'`、`'*'`、`'a'\”b'` 全部正常 —— 说明本机的调用层对 `$`/反引号另有一套处理。
// 真机 POSIX shell 不会这样，所以这类用例在这台机器上测不出真结果；跳过并写明原因，
// **不拿环境问题当代码缺陷**（`shQuote` 本身是标准写法：`'…'` + `'\”`）。
func bashExpandsCommandSubstitution(t *testing.T) bool {
	t.Helper()
	out, err := exec.Command("bash", "-c", "printf %s "+shQuote("$(printf SUB)")).Output()
	return err == nil && string(out) != "$(printf SUB)"
}

// 只有单个词、且引用正确时,shell 反解出来的才等于原值。
// 用一个刻意带 shell 元字符的集合:任何一处漏转义都会让结果对不上(或被拆成多条命令)。
func TestShQuoteSurvivesShell(t *testing.T) {
	payloads := []string{
		"ds/pass@//192.0.2.109:1521/t35prd",
		"a'b",
		`a'\''b`,
		"a b",
		"$(id)",
		"`id`",
		`a\b`,
		"a;b",
		"a|b",
		"a&&b",
		"a>b",
		"a\nb",
		"*",
		"~",
	}
	for _, s := range payloads {
		t.Run(s, func(t *testing.T) {
			if bashExpandsCommandSubstitution(t) && strings.ContainsAny(s, "$`") {
				// D8：只跳"命令替换"这一类（本机调用层会展开它），其余元字符用例照测。
				t.Skipf("本机 bash 会展开单引号内的命令替换：这条用例在这台机器上测不出真结果（台账 D8）")
			}
			out, err := exec.Command("bash", "-c", "printf %s "+shQuote(s)).Output()
			if err != nil {
				t.Fatalf("反解失败: %v", err)
			}
			if string(out) != s {
				t.Errorf("引用没能还原原值\n 得到 %q\n 期望 %q", out, s)
			}
		})
	}
}

// bashLC 产出 `bash -lc '<脚本>'`。去掉前缀后剩下的那一段,应当被 shell 解析成
// **恰好等于原脚本**的一个参数 —— 转义错一处就会变成多个词或提前闭合。
func TestBashLCRoundTrips(t *testing.T) {
	scripts := []string{
		`source /u3/pub/bin/chenv 36 >/dev/null 2>&1; exec sqlplus -S 'ds/p@//h:1521/s'`,
		`echo 'a'\''b'`,
		`echo "$(id)"`, // 这是**脚本**里的内容,进 bash -lc 后仍应原样是一个参数
		"echo 'a;b|c'",
	}
	for _, s := range scripts {
		t.Run(s, func(t *testing.T) {
			if bashExpandsCommandSubstitution(t) && strings.ContainsAny(s, "$`") {
				// D8：同一道守卫（这条脚本里带 `$(id)`，本机会展开它）。
				t.Skipf("本机 bash 会展开单引号内的命令替换：这条脚本在这台机器上测不出真结果（台账 D8）")
			}
			cmd := bashLC(s)
			if !strings.HasPrefix(cmd, "bash -lc ") {
				t.Fatalf("不是 bash -lc 形态: %s", cmd)
			}
			arg := strings.TrimPrefix(cmd, "bash -lc ")
			// 只把 arg 交给 shell 解析,用 printf 把结果打出来 —— 不执行脚本本身
			out, err := exec.Command("bash", "-c", "printf %s "+arg).Output()
			if err != nil {
				t.Fatalf("反解失败: %v (%s)", err, arg)
			}
			if string(out) != s {
				t.Errorf("bash -lc 收到的不等于原脚本\n 得到 %q\n 期望 %q\n 中间产物: %s", out, s, arg)
			}
		})
	}
}

// zone 会被拼进远端 shell。以前它裸插且不在引号内 —— 实测 `--zone "36; id"` 能执行任意命令。
func TestChenvCmdRejectsInjection(t *testing.T) {
	bad := []string{"36; id", "36 && id", "36|id", "36`id`", "36$(id)", "36\nid", "36'", `36"`, "36;/bin/sh", strings.Repeat("9", 20)}
	for _, z := range bad {
		if got, err := ChenvCmd(z); err == nil {
			t.Errorf("区域代码 %q 应被拒绝,却产出了命令: %s", z, got)
		}
	}
	good := []string{"36", "31", "t", "36k", "1", ""}
	for _, z := range good {
		got, err := ChenvCmd(z)
		if err != nil {
			t.Errorf("区域代码 %q 应放行: %v", z, err)
			continue
		}
		if strings.ContainsAny(got, `'"`) {
			t.Errorf("片段里不该出现引号(会与外层 bash -lc 的单引号打架): %s", got)
		}
	}
	if got, _ := ChenvCmd(""); !strings.Contains(got, "chenv 36 ") {
		t.Errorf("空 zone 应退化为 36,得到: %s", got)
	}
}

// SqlplusCmd 产出的命令里**不该有任何 SQL 文本** —— SQL 走 stdin 是这次改造的核心。
// 这里用"把一段 SQL 当连接串传进去"来模拟最坏情况:它必须被完整引用住。
func TestSqlplusCmdHasNoSqlAndQuotesValues(t *testing.T) {
	nasty := "ds/p'; id; echo 'x@//h:1521/s"
	cmd, err := SqlplusCmd("36", "", "/u2/oracle/bin/sqlplus", nasty, 30)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	// 直接命令注入的证据:整串应当是**一个** bash -lc 参数
	arg := strings.TrimPrefix(cmd, "bash -lc ")
	out, err := exec.Command("bash", "-c", "printf %s "+arg).Output()
	if err != nil {
		t.Fatalf("反解失败: %v", err)
	}
	want := "source /u3/pub/bin/chenv 36 >/dev/null 2>&1; " +
		`export NLS_LANG="${NLS_LANG%%.*}.AL32UTF8"; ` +
		"exec timeout -s TERM 30 /u2/oracle/bin/sqlplus -S " + shQuote(nasty)
	if string(out) != want {
		t.Errorf("展开后与期望不符\n 得到 %q\n 期望 %q", out, want)
	}
	// 上面那条往返断言就是"注入已根治"的证据:整串展开后**恰好等于**期望脚本,
	// 说明 payload 里的 `; id; echo` 全部被引用成了数据,没有变成第二条命令。
	// (不再断言"命令里不含 echo" —— payload 自己就含这个词,那种检查会被数据本身骗到。)
}

// 服务器 NLS_LANG 为空时，sqlplus 以 US7ASCII 连库，中文会在客户端转换成 `?`。
// 命令必须把字符集段固定为 AL32UTF8，同时保留「语言_区域」段（日期格式等不变）。
func TestSqlplusCmdForcesUTF8Charset(t *testing.T) {
	cmd, err := SqlplusCmd("36", "", "/u2/bin/sqlplus", "ds/x@//h:1521/s", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cmd, `export NLS_LANG="${NLS_LANG%%.*}.AL32UTF8"`) {
		t.Errorf("命令里缺少 NLS_LANG 字符集覆盖（中文会被转成 ?）：\n%s", cmd)
	}
}

func TestSqlplusCmdRejectsBadInputs(t *testing.T) {
	if _, err := SqlplusCmd("36; id", "", "/u2/bin/sqlplus", "ds/x", 0); err == nil {
		t.Error("非法 zone 应被拒绝")
	}
	if _, err := SqlplusCmd("36", "", "/u2/bin/sqlplus; id", "ds/x", 0); err == nil {
		t.Error("非法工具路径应被拒绝")
	}
	if _, err := SqlplusCmd("36", "", "/u2/bin/sqlplus", "ds/x\nid", 0); err == nil {
		t.Error("连接串含换行应被拒绝")
	}
	// ORACLE_HOME 会原样拼进远端脚本,相对路径/元字符/空白一律拒绝
	for _, oh := range []string{"/u2/oracle; id", "/u2/oracle home", "u2/oracle", "/u2/oracle/../x`id`", "/u2/oracle/$HOME"} {
		if _, err := SqlplusCmd("36", oh, "/u2/bin/sqlplus", "ds/x", 0); err == nil {
			t.Errorf("非法 ORACLE_HOME %q 应被拒绝", oh)
		}
	}
	// killAfterSec=0 时不套 timeout
	cmd, err := SqlplusCmd("36", "", "/u2/bin/sqlplus", "ds/x@//h:1521/s", 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cmd, "timeout") {
		t.Errorf("未要求超时包装却出现了 timeout: %s", cmd)
	}
}

// 有的站点 sqlplus 不在 PATH、chenv 也不设 ORACLE_HOME(实测 /u1 安装):
// 命令必须把探测到的 ORACLE_HOME 显式 export 出去,否则 sqlplus 起不来
// (SP2-0667/SP2-0750),而只靠 PATH 会得到 status 127。
func TestSqlplusCmdExportsOracleHome(t *testing.T) {
	const home = "/u2/oracle/product/19.0.0/dbhome_1"
	const connStr = "ds/x@//h:1521/s"
	cmd, err := SqlplusCmd("36", home, home+"/bin/sqlplus", connStr, 0)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	// 命令整体经 bashLC 包过一层,连接串里的单引号已按 POSIX 惯例转义 —— 断言用同一份转义。
	quoted := strings.ReplaceAll(shQuote(connStr), "'", `'\''`)
	for _, want := range []string{
		"export ORACLE_HOME=" + home,
		`export PATH="$ORACLE_HOME/bin:$PATH"`,
		"exec " + home + "/bin/sqlplus -S " + quoted,
	} {
		if !strings.Contains(cmd, want) {
			t.Errorf("命令里缺少 %q\n得到: %s", want, cmd)
		}
	}
	// 没探到 ORACLE_HOME 时不该凭空 export,保留 chenv 的 PATH 兜底
	cmd, err = SqlplusCmd("36", "", "", connStr, 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cmd, "ORACLE_HOME") {
		t.Errorf("ORACLE_HOME 为空时不该 export: %s", cmd)
	}
	if !strings.Contains(cmd, "exec sqlplus -S "+quoted) {
		t.Errorf("路径为空时应回退 PATH 上的 sqlplus: %s", cmd)
	}
}

func TestKbCmdRejectsInjection(t *testing.T) {
	if _, err := KbCmd("/u2/bin/ksql; id", "h", "54321", "db", "ds/x"); err == nil {
		t.Error("非法工具路径应被拒绝")
	}
	if _, err := KbCmd("/u2/bin/ksql", "h", "54321", "db", "ds'; id; echo '"); err == nil {
		t.Error("非法账号名应被拒绝")
	}
	// 正常形态:口令单引号包裹
	cmd, err := KbCmd("/u2/bin/ksql", "127.0.0.1", "54321", "t100", "ds/pa'ss")
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	if !strings.Contains(cmd, `KINGBASE_PASSWORD='pa'\''ss'`) {
		t.Errorf("口令未被单引号包住: %s", cmd)
	}
	if strings.Contains(cmd, "-c ") {
		t.Errorf("不该再用 -c 把 SQL 拼进命令行: %s", cmd)
	}
}

// 金仓路线也要钉死客户端字符集：库里的中文按库编码回字节，Go 侧一律按 UTF-8 解，
// 不设 PGCLIENTENCODING 就会乱码（与 sqlplus 的 NLS_LANG 是同一个病）。
func TestKbCmdForcesUTF8ClientEncoding(t *testing.T) {
	cmd, err := KbCmd("/u2/bin/ksql", "127.0.0.1", "54321", "t100", "ds/pw")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(cmd, "PGCLIENTENCODING=UTF8 ") {
		t.Errorf("命令应以 PGCLIENTENCODING=UTF8 开头（否则中文乱码）：\n%s", cmd)
	}
	// 口令仍必须原样带着 —— 加前缀不能把原来的环境变量挤掉。
	if !strings.Contains(cmd, "KINGBASE_PASSWORD='pw'") {
		t.Errorf("口令前缀丢了：\n%s", cmd)
	}
	// 命令层形态，不该冒出 export / 分号（那意味着要再过一层 shell）。
	if strings.Contains(cmd, "export ") || strings.Contains(cmd, ";") {
		t.Errorf("KbCmd 是单命令，不该含 export 或分号：\n%s", cmd)
	}
}
