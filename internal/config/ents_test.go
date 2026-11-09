package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// SaveEnvEnts 是「ENT」页签 / tt env ents --refresh 的写入口：
// 它必须做到三件事 —— 清单落盘、同步时间落盘、把缺的账号补进账号清单
// （口令以账号清单为准；新账号按惯例 账号=密码）。
func TestSaveEnvEnts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"schemaVersion":2,"hosts":{"sshs":[{"name":"A","host":"h","port":22,"user":"u","password":"p",
	  "db":{"type":"oracle","host":"h","port":1521,"service":"s","accounts":[{"account":"ds","password":"SECRET"}]}}]}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 10, 19, 14, 0, 0, 0, time.Local)
	added, err := SaveEnvEnts(path, "A", []SyncedEnt{
		{Ent: 99, Account: "ds"},      // 已在账号清单：口令应抄成 SECRET
		{Ent: 907, Account: "dsdata"}, // 不在：按惯例补进去
	}, at)
	if err != nil {
		t.Fatalf("SaveEnvEnts: %v", err)
	}
	if len(added) != 1 || added[0] != "dsdata" {
		t.Fatalf("新增账号应为 [dsdata]，得到 %v", added)
	}

	root, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	e := root.Hosts.ByName("A")
	if e == nil {
		t.Fatal("环境 A 丢了")
	}
	if e.EntsSyncedAt != at.Format(time.RFC3339) {
		t.Errorf("entsSyncedAt = %q，想要 %q", e.EntsSyncedAt, at.Format(time.RFC3339))
	}
	if len(e.Ents) != 2 || e.Ents[0].Ent != 99 || e.Ents[1].Account != "dsdata" {
		t.Fatalf("清单不对: %+v", e.Ents)
	}
	if e.Ents[0].Password != "SECRET" {
		t.Errorf("已在账号清单的账号，口令应以清单为准（SECRET），得到 %q", e.Ents[0].Password)
	}
	if e.Ents[1].Password != "dsdata" {
		t.Errorf("新账号的口令应按惯例=账号名，得到 %q", e.Ents[1].Password)
	}

	// 账号清单里也必须有 dsdata（否则运行时连不上，而错误会伪装成“库里没这个号”）
	found := false
	for _, a := range e.DB.Accounts {
		if a.Account == "dsdata" {
			found = true
		}
	}
	if !found {
		t.Error("dsdata 没有被补进 db.accounts")
	}
}

// 再存一次要覆盖而不是追加（同步是幂等的：重复点刷新不该越长越多）。
func TestSaveEnvEntsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := `{"schemaVersion":2,"hosts":{"sshs":[{"name":"A","host":"h","port":22,"user":"u","password":"p"}]}}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := SaveEnvEnts(path, "A", []SyncedEnt{{Ent: 1, Account: "ds"}}, time.Now()); err != nil {
			t.Fatal(err)
		}
	}
	root, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(root.Hosts.ByName("A").Ents); n != 1 {
		t.Fatalf("重复同步后清单应仍为 1 条，得到 %d", n)
	}
}
