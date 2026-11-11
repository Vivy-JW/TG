package dict

import (
	"database/sql"
	"path/filepath"
	"testing"

	"tt/internal/config"
)

func TestLookupModule(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "TOPTEST.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE t_prog(prog TEXT PRIMARY KEY, name TEXT, module TEXT, memo TEXT, name2 TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE t_prog_link(prog TEXT, module TEXT, linked_prog TEXT, link_flag TEXT, PRIMARY KEY(prog, module, linked_prog))`); err != nil {
		t.Fatal(err)
	}
	// t_prog 里故意放一个错的 CSUB 风格值：精确入口行 cbm/abmi600 必须赢。
	if _, err := db.Exec(`INSERT INTO t_prog(prog,module) VALUES('abmi600','zzz')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO t_prog_link(prog,module,linked_prog,link_flag) VALUES('abmi600','cbm','abmi600','Y'),('abmi600','zzz','other','Y')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	env := config.NamedSsh{Name: "TOPTEST", DictDir: dir}
	if mod, ok := LookupModule(env, "abmi600"); !ok || mod != "cbm" {
		t.Fatalf("LookupModule = %q,%v; want cbm,true", mod, ok)
	}
	if mod, ok := LookupModule(env, "cbm_abmi600"); !ok || mod != "cbm" {
		t.Fatalf("带前缀程序名应回退到短名: %q,%v", mod, ok)
	}
	if _, ok := LookupModule(env, "unknown"); ok {
		t.Fatal("未知程序不应命中")
	}
}
