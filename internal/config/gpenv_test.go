package config

import (
	"testing"

	"tg/internal/dbconfig"
)

func testGpEnv() NamedSsh {
	return NamedSsh{
		Name: "TOPTEST",
		SSHConfig: SSHConfig{
			Host: "172.16.1.22", Port: 22, User: "tiptop", Password: "ssh-pwd",
		},
		Product:   "gp",
		Workspace: "/u1/toptest/tiptop",
		CustRoot:  "/u1/toptest/topcust",
		MirrorDir: `D:\mirror\TOPTEST`,
		DictDir:   `D:\dict`,
		DB: &dbconfig.Connection{
			Type: "oracle", Host: "172.16.1.22", Port: 1521, Service: "toptest",
			Accounts: []dbconfig.DBAcct{{Account: "ds", Password: "db-pwd"}},
		},
	}
}

func strptr(s string) *string { return &s }

// 只改主机时，service / 密码 / 目录都必须原样保留。
func TestApplyEnvPatchHostOnlyPreservesEverythingElse(t *testing.T) {
	env := testGpEnv()
	ApplyEnvPatch(&env, EnvPatch{Host: strptr("10.0.0.9")})

	if env.Host != "10.0.0.9" {
		t.Fatalf("host = %q", env.Host)
	}
	if env.Password != "ssh-pwd" {
		t.Fatalf("ssh password lost: %q", env.Password)
	}
	if env.DB == nil || env.DB.Service != "toptest" {
		t.Fatalf("db service lost: %+v", env.DB)
	}
	if len(env.DB.Accounts) != 1 || env.DB.Accounts[0].Password != "db-pwd" {
		t.Fatalf("db password lost: %+v", env.DB.Accounts)
	}
	if env.MirrorDir != `D:\mirror\TOPTEST` || env.DictDir != `D:\dict` {
		t.Fatalf("dirs lost: mirror=%q dict=%q", env.MirrorDir, env.DictDir)
	}
}

// 表单只提交账号名（密码留空）时，按账号名沿用旧口令；service 不动。
func TestApplyEnvPatchAccountWithoutPasswordKeepsOld(t *testing.T) {
	env := testGpEnv()
	ApplyEnvPatch(&env, EnvPatch{DB: &DBPatch{
		Host:     strptr("172.16.1.23"),
		Accounts: &[]dbconfig.DBAcct{{Account: "ds"}},
	}})

	if env.DB.Host != "172.16.1.23" {
		t.Fatalf("db host = %q", env.DB.Host)
	}
	if env.DB.Service != "toptest" {
		t.Fatalf("db service lost: %+v", env.DB)
	}
	if len(env.DB.Accounts) != 1 || env.DB.Accounts[0].Password != "db-pwd" {
		t.Fatalf("db password lost: %+v", env.DB.Accounts)
	}
}

// 显式提交新密码才覆盖；只改 service 时密码不动。
func TestApplyEnvPatchServiceAndPassword(t *testing.T) {
	env := testGpEnv()
	ApplyEnvPatch(&env, EnvPatch{DB: &DBPatch{Service: strptr("toptest2")}})
	if env.DB.Service != "toptest2" || env.DB.Accounts[0].Password != "db-pwd" {
		t.Fatalf("service-only patch: %+v", env.DB)
	}

	ApplyEnvPatch(&env, EnvPatch{DB: &DBPatch{
		Accounts: &[]dbconfig.DBAcct{{Account: "ds", Password: "new-pwd"}},
	}})
	if env.DB.Accounts[0].Password != "new-pwd" {
		t.Fatalf("new password not applied: %+v", env.DB.Accounts)
	}
}

// 旧版整条 env 请求：空密码不应把已存口令清掉。
func TestPatchFromEnvBlankPasswordKeepsOld(t *testing.T) {
	env := testGpEnv()
	legacy := testGpEnv()
	legacy.Password = ""
	legacy.DB.Accounts = []dbconfig.DBAcct{{Account: "ds"}}

	ApplyEnvPatch(&env, PatchFromEnv(legacy))
	if env.Password != "ssh-pwd" {
		t.Fatalf("ssh password lost: %q", env.Password)
	}
	if env.DB.Accounts[0].Password != "db-pwd" {
		t.Fatalf("db password lost: %+v", env.DB.Accounts)
	}
}

// 测试连接：表单只改 DB 主机、口令留空时，必须拿已保存的 DB 口令去测。
func TestMergeTestEnvUsesSavedDBPassword(t *testing.T) {
	saved := testGpEnv()
	req := saved
	req.Password = ""
	req.DB = &dbconfig.Connection{
		Type: "oracle", Host: "172.16.1.23", Port: 1521, Service: "",
		Accounts: []dbconfig.DBAcct{{Account: "ds"}},
	}
	got := MergeTestEnv(req, saved)
	if got.Password != "ssh-pwd" {
		t.Fatalf("ssh password = %q", got.Password)
	}
	if got.DB.Service != "toptest" {
		t.Fatalf("db service = %q", got.DB.Service)
	}
	if got.DB.Accounts[0].Password != "db-pwd" {
		t.Fatalf("db password = %+v", got.DB.Accounts)
	}
}

// 多账号时只提交首个账号，其余账号与口令不能被抹掉。
func TestMergeAccountsPreservesOtherAccounts(t *testing.T) {
	db := &dbconfig.Connection{Accounts: []dbconfig.DBAcct{
		{Account: "ds", Password: "ds"},
		{Account: "dsdata", Password: "dsdata"},
	}}
	MergeAccounts(db, []dbconfig.DBAcct{{Account: "ds", Password: "new"}})
	if len(db.Accounts) != 2 {
		t.Fatalf("accounts = %+v", db.Accounts)
	}
	if db.Accounts[0].Password != "new" || db.Accounts[1].Password != "dsdata" {
		t.Fatalf("accounts = %+v", db.Accounts)
	}
}

// 实例名与路径推导：/u1/<实例>/tiptop 与 /u1/<实例>/topcust。
func TestRootsRoundTrip(t *testing.T) {
	top, cust := RootsForInstance("toptest")
	if top != "/u1/toptest/tiptop" || cust != "/u1/toptest/topcust" {
		t.Fatalf("roots = %q %q", top, cust)
	}
	if got := InstanceFromWorkspace(top); got != "toptest" {
		t.Fatalf("instance = %q", got)
	}
	if ValidInstance("top test") || ValidInstance("") {
		t.Fatalf("ValidInstance 放过了非法实例名")
	}
}
