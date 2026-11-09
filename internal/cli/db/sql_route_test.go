package db

// --route 的契约：取值校验（离线可测）、信封里 route 落到头部、以及
// “隐藏的 tt debug sql 转发不许悄悄改路线”。

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/dbconfig"
	"tt/internal/output"
)

func TestSQLRouteValid(t *testing.T) {
	for _, ok := range []string{SQLRouteAuto, SQLRouteDirect, SQLRouteServer} {
		if !sqlRouteValid(ok) {
			t.Errorf("%q 应合法", ok)
		}
	}
	for _, bad := range []string{"", "direct ", "AUTO", "sqlplus", "both"} {
		if sqlRouteValid(bad) {
			t.Errorf("%q 应非法", bad)
		}
	}
}

func TestQueryOnceRejectsUnknownRoute(t *testing.T) {
	err := QueryOnce("select 1 from dual", 99, 30, "sqlplus")
	var oe *output.Error
	if !errors.As(err, &oe) {
		t.Fatalf("应是带码的错误，得到 %T: %v", err, err)
	}
	if oe.Code != "USAGE" || oe.Exit != 2 {
		t.Errorf("未知路线应是 USAGE/2，得到 %s/%d", oe.Code, oe.Exit)
	}
	if !strings.Contains(oe.Error(), "--route") {
		t.Errorf("错误里要点名 --route，得到 %q", oe.Error())
	}
}

func TestQueryOnceRejectsMissingEntBeforeAnythingElse(t *testing.T) {
	// 没有企业编号就不该走任何路线（决定用哪个账号的是它）。
	err := QueryOnce("select 1 from dual", 0, 30, SQLRouteAuto)
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Exit != 2 {
		t.Fatalf("缺 ENT 应是 USAGE 退 2，得到 %v", err)
	}
}

func TestWriteSQLResultReportsDirectRoute(t *testing.T) {
	res := testResult()
	res.Route = "client-direct"
	res.SSHHost = ""
	res.ViaTunnel = true

	oldCSV, oldJSON := common.CSV, common.JSON
	common.CSV, common.JSON = true, false
	t.Cleanup(func() { common.CSV, common.JSON = oldCSV, oldJSON })

	var b bytes.Buffer
	if err := writeSQLResult(&b, res); err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	head := b.String()
	if !strings.Contains(head, "client-direct") {
		t.Errorf("头部应回显实际路线 client-direct:\n%s", head)
	}
	if !strings.Contains(head, "经 SSH 隧道") {
		t.Errorf("viaSsh 的直连应标出隧道:\n%s", head)
	}
}

func TestWriteSQLResultKeepsServerRouteForOldCallers(t *testing.T) {
	// tt debug sql 的转发不带 Route：仍按方言推成 server-sqlplus，输出一个字节不变。
	res := testResult()
	res.Route = ""
	var b bytes.Buffer
	oldCSV, oldJSON := common.CSV, common.JSON
	common.CSV, common.JSON = true, false
	t.Cleanup(func() { common.CSV, common.JSON = oldCSV, oldJSON })
	if err := writeSQLResult(&b, res); err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	if !strings.Contains(b.String(), "server-sqlplus") {
		t.Errorf("老调用方应仍显示 server-sqlplus:\n%s", b.String())
	}
}

func TestServerRouteFollowsDialect(t *testing.T) {
	if got := serverRoute("kingbase"); got != dbconfig.RouteServerKB {
		t.Errorf("kingbase 的服务器路线应是 %s，得到 %s", dbconfig.RouteServerKB, got)
	}
	if got := serverRoute("oracle"); got != dbconfig.RouteServerOracle {
		t.Errorf("oracle 的服务器路线应是 %s，得到 %s", dbconfig.RouteServerOracle, got)
	}
}

func TestDebugForwardStaysOnServerRoute(t *testing.T) {
	// 这条是给“隐藏转发”上的锁：空 route 必须落到 server，
	// 不许因为默认值改成 auto 就把旧命令的执行方式换掉。
	if got := normalizeSQLRoute(""); got != SQLRouteServer {
		t.Errorf("空 route 应视为 server，得到 %q", got)
	}
	if got := normalizeSQLRoute(SQLRouteAuto); got != SQLRouteAuto {
		t.Errorf("显式 auto 应保持不变，得到 %q", got)
	}
}
