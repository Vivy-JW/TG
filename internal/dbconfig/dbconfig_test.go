package dbconfig

import "testing"

// 本包全是纯函数（连接类型的取值规则），零外部依赖，所以全是表驱动单测。
//
// **口令一律是假的**：真实的在 config.json 里，绝不进测试夹具。
// 这里用 "pw-dev" / "SECRET" 这类只为了分辨"取到了哪个值"。

func TestViaSSHEffectiveRemote(t *testing.T) {
	cases := []struct {
		name         string
		v            ViaSSH
		fallbackHost string
		fallbackPort int
		wantHost     string
		wantPort     int
	}{
		{"显式给了远端就用远端", ViaSSH{RemoteHost: "10.0.0.9", RemotePort: 9999}, "ssh1", 22, "10.0.0.9", 9999},
		{"都没给就回退连接的 host/port", ViaSSH{}, "ssh1", 1521, "ssh1", 1521},
		{"只给 host：port 各自回退", ViaSSH{RemoteHost: "10.0.0.9"}, "ssh1", 1521, "10.0.0.9", 1521},
		{"只给 port：host 各自回退", ViaSSH{RemotePort: 9999}, "ssh1", 1521, "ssh1", 9999},
		{"port<=0 视为没给（0 与负数都是）", ViaSSH{RemotePort: -1}, "ssh1", 1521, "ssh1", 1521},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h, p := c.v.EffectiveRemote(c.fallbackHost, c.fallbackPort)
			if h != c.wantHost || p != c.wantPort {
				t.Errorf("得 %s:%d，想 %s:%d", h, p, c.wantHost, c.wantPort)
			}
		})
	}
}

// TestReadonlySQLEnabledTriState 钉住那个**三态**：没配（nil）与显式关掉不是一回事。
//
// 指针在这里是有意义的 —— 用 bool 的话"没配"与"关掉"无法区分，而缺省是**开启**。
func TestReadonlySQLEnabledTriState(t *testing.T) {
	no := false
	yes := true
	cases := []struct {
		name string
		c    *Connection
		want bool
	}{
		{"nil 连接 = 开启（不给别人踩空）", nil, true},
		{"没配 = 开启", &Connection{}, true},
		{"显式关掉 = 关掉", &Connection{ReadonlySQL: &no}, false},
		{"显式开启 = 开启", &Connection{ReadonlySQL: &yes}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.ReadonlySQLEnabled(); got != c.want {
				t.Errorf("得 %v，想 %v", got, c.want)
			}
		})
	}
}

func TestDialCredTakesFirstAccount(t *testing.T) {
	cases := []struct {
		name   string
		c      *Connection
		wantU  string
		wantP  string
		wantOK bool
	}{
		{"取列表首项", &Connection{Accounts: []DBAcct{
			{Account: "ds", Password: "pw-dev"},
			{Account: "dsdata", Password: "SECRET"},
		}}, "ds", "pw-dev", true},
		{"nil 连接", nil, "", "", false},
		{"没有账号", &Connection{}, "", "", false},
		{"首项账号名为空 = 没有", &Connection{Accounts: []DBAcct{{Account: "", Password: "pw-dev"}}}, "", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, p, ok := c.c.DialCred()
			if u != c.wantU || p != c.wantP || ok != c.wantOK {
				t.Errorf("得 (%q, %q, %v)，想 (%q, %q, %v)", u, p, ok, c.wantU, c.wantP, c.wantOK)
			}
		})
	}
}

// TestPasswordForNeedsANonEmptyPassword "收录在册"必须真有密码才算 ——
// 空密码的条目等于没收，调用方要回退到"账号=密码"的惯例。
func TestPasswordForNeedsANonEmptyPassword(t *testing.T) {
	c := &Connection{Accounts: []DBAcct{
		{Account: "ds", Password: ""},
		{Account: "dsdata", Password: "SECRET"},
	}}
	if _, ok := c.PasswordFor("ds"); ok {
		t.Error("密码为空不该算收录")
	}
	if p, ok := c.PasswordFor("dsdata"); !ok || p != "SECRET" {
		t.Errorf("已收录该取到密码，得 (%q, %v)", p, ok)
	}
	if _, ok := c.PasswordFor("nobody"); ok {
		t.Error("未收录该返回 false")
	}
	if _, ok := (&Connection{}).PasswordFor("ds"); ok {
		t.Error("没有账号列表时该返回 false")
	}
	var nilConn *Connection
	if _, ok := nilConn.PasswordFor("ds"); ok {
		t.Error("nil 连接该返回 false")
	}
}

// TestFillDialCred 客户端直连前把内部凭据槽填上。两条规则要分清：
// 指定了 Account 就按它查（未收录回退"账号=密码"）；没指定就取列表首项。
func TestFillDialCred(t *testing.T) {
	t.Run("没指定 Account：取列表首项", func(t *testing.T) {
		c := &Connection{Accounts: []DBAcct{
			{Account: "ds", Password: "pw-dev"},
			{Account: "dsdata", Password: "SECRET"},
		}}
		if err := c.FillDialCred(); err != nil {
			t.Fatal(err)
		}
		if c.User != "ds" || c.Password != "pw-dev" {
			t.Errorf("得 (%q, %q)，想 (ds, pw-dev)", c.User, c.Password)
		}
	})
	t.Run("指定 Account 且已收录：用清单里的密码", func(t *testing.T) {
		c := &Connection{
			Accounts: []DBAcct{{Account: "ds", Password: "pw-dev"}, {Account: "dsdata", Password: "SECRET"}},
			Account:  "dsdata",
		}
		if err := c.FillDialCred(); err != nil {
			t.Fatal(err)
		}
		if c.User != "dsdata" || c.Password != "SECRET" {
			t.Errorf("得 (%q, %q)，想 (dsdata, SECRET)", c.User, c.Password)
		}
	})
	t.Run("指定 Account 但未收录：回退 T100 惯例 账号=密码", func(t *testing.T) {
		c := &Connection{
			Accounts: []DBAcct{{Account: "ds", Password: "pw-dev"}},
			Account:  "ghost",
		}
		if err := c.FillDialCred(); err != nil {
			t.Fatal(err)
		}
		if c.User != "ghost" || c.Password != "ghost" {
			t.Errorf("未收录该回退成 账号=密码，得 (%q, %q)", c.User, c.Password)
		}
	})
	t.Run("一个账号都没有：报错而不是填个空的", func(t *testing.T) {
		c := &Connection{}
		if err := c.FillDialCred(); err == nil {
			t.Fatal("没账号该报错（填空凭据会让失败推后到连接时，报的错与真实原因对不上）")
		}
	})
	t.Run("指定了 Account 时即便列表为空也不报错", func(t *testing.T) {
		// 服务器侧按 TOPENT 解析出的账号就是这样用的：清单里没有也照连（惯例 账号=密码）。
		c := &Connection{Account: "ghost"}
		if err := c.FillDialCred(); err != nil {
			t.Fatalf("该走惯例而不是报错：%v", err)
		}
		if c.User != "ghost" || c.Password != "ghost" {
			t.Errorf("得 (%q, %q)，想 (ghost, ghost)", c.User, c.Password)
		}
	})
}

func TestSvcFallsBackToDatabase(t *testing.T) {
	if got := (&Connection{Service: "t35prd", Database: "old"}).Svc(); got != "t35prd" {
		t.Errorf("有 Service 该用它，得 %q", got)
	}
	if got := (&Connection{Database: "topprd"}).Svc(); got != "topprd" {
		t.Errorf("没有 Service 该回退 Database（旧字段兜底），得 %q", got)
	}
	if got := (&Connection{}).Svc(); got != "" {
		t.Errorf("都没有该是空串，得 %q", got)
	}
}

func TestDefaultPort(t *testing.T) {
	if got := (&Connection{Type: "kingbase"}).DefaultPort(); got != 54321 {
		t.Errorf("金仓该是 54321，得 %d", got)
	}
	for _, ty := range []string{"oracle", "", "别的"} {
		if got := (&Connection{Type: ty}).DefaultPort(); got != 1521 {
			t.Errorf("类型 %q 该回退 1521，得 %d", ty, got)
		}
	}
}

func TestAddress(t *testing.T) {
	cases := []struct {
		name string
		c    Connection
		want string
	}{
		{"没主机就是空（不打印 ::/ 这种空壳）", Connection{Port: 1521}, ""},
		{"显式端口", Connection{Host: "db1", Port: 1600, Service: "t35prd"}, "db1:1600/t35prd"},
		{"端口缺省按类型补", Connection{Host: "db1", Service: "t35prd"}, "db1:1521/t35prd"},
		{"金仓的缺省端口是 54321", Connection{Type: "kingbase", Host: "db1", Database: "topprd"}, "db1:54321/topprd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.c.Address(); got != c.want {
				t.Errorf("得 %q，想 %q", got, c.want)
			}
		})
	}
}

// TestNewTargetSeedsClientDirectConvention 落脚点先按客户端直连惯例填，
// 等"按企业编号解析出账号"之后再 WithAccount 覆盖。
func TestNewTargetSeedsClientDirectConvention(t *testing.T) {
	conn := &Connection{
		Type: "oracle", Host: "db1", Service: "t35prd",
		Accounts: []DBAcct{{Account: "ds", Password: "pw-dev"}},
	}
	tg := NewTarget(conn, "E1", "ssh1", "u1", "36", "7", RouteClientDirect)

	if tg.Account != "ds" || tg.AccountSource != AcctFromPrimary {
		t.Errorf("账号该预填列表首项：得 (%q, %q)", tg.Account, tg.AccountSource)
	}
	if tg.Env != "E1" || tg.SSHHost != "ssh1" || tg.SSHUser != "u1" || tg.Zone != "36" || tg.Topent != "7" {
		t.Errorf("环境字段没带全：%+v", tg)
	}
	if tg.ViaTunnel {
		t.Error("没配 ViaSSH 时 ViaTunnel 该是 false")
	}
	if got := tg.Address(); got != "db1:1521/t35prd" {
		t.Errorf("Address 该委托给连接，得 %q", got)
	}
	if got := tg.Dialect(); got != "oracle" {
		t.Errorf("Dialect 该取连接类型，得 %q", got)
	}
}

func TestNewTargetWithTunnelAndNoConn(t *testing.T) {
	conn := &Connection{Type: "kingbase", Host: "db1", Database: "topprd", ViaSSH: &ViaSSH{Host: "ssh1"}}
	tg := NewTarget(conn, "E1", "ssh1", "u1", "36", "", RouteClientDirect)
	if !tg.ViaTunnel {
		t.Error("配了 ViaSSH 该标成经隧道")
	}

	// 没有连接（本地 SQLite 那种落脚点）也不该炸。
	none := NewTarget(nil, "", "", "", "", "", RouteLocalSQLite)
	if none.Account != "" || none.ViaTunnel {
		t.Errorf("没有连接时账号该留空：%+v", none)
	}
	// 连接的账号列表为空时同样留空，而不是拿个空串当账号。
	empty := NewTarget(&Connection{}, "E1", "", "", "", "", RouteClientDirect)
	if empty.Account != "" {
		t.Errorf("没有账号时该留空，得 %q", empty.Account)
	}
}

func TestWithAccount(t *testing.T) {
	tg := NewTarget(&Connection{Accounts: []DBAcct{{Account: "ds", Password: "pw-dev"}}},
		"E1", "ssh1", "u1", "36", "", RouteClientDirect)

	tg.WithAccount("dsdata", AcctFromEnt, 7)
	if tg.Account != "dsdata" || tg.AccountSource != AcctFromEnt || tg.Ent != 7 {
		t.Errorf("该被覆盖：得 (%q, %q, %d)", tg.Account, tg.AccountSource, tg.Ent)
	}

	// 空账号不覆盖（解析不出来时保留原来的预填值），ent<=0 也不覆盖。
	tg.WithAccount("", AcctFromEnt, 0)
	if tg.Account != "dsdata" || tg.Ent != 7 {
		t.Errorf("空账号 / ent<=0 不该覆盖：得 (%q, %d)", tg.Account, tg.Ent)
	}
}
