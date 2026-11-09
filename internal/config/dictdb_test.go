package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDictDBPathIsPerEnv(t *testing.T) {
	dir := t.TempDir()
	a := DictDBPath(dir, "主机109")
	b := DictDBPath(dir, "尚阳通")
	if a == b {
		t.Fatalf("两个环境指向同一个库: %s", a)
	}
	if filepath.Dir(a) != filepath.Join(dir, DictDBDirName) {
		t.Fatalf("库文件应在 %s 下: %s", filepath.Join(dir, DictDBDirName), a)
	}
	if !strings.HasSuffix(a, ".db") || filepath.Base(a) == DefaultDictDBName {
		t.Fatalf("不该用旧的共享库名: %s", a)
	}
}

func TestDictDBFileNameSanitizes(t *testing.T) {
	name := DictDBFileName(`a/b:c*?"<>|`)
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		t.Errorf("文件名仍含非法字符: %q", name)
	}
	if !strings.HasSuffix(name, ".db") {
		t.Errorf("应以 .db 结尾: %q", name)
	}
	if got := DictDBFileName("   "); got != "env.db" {
		t.Errorf("空环境名 = %q, want env.db", got)
	}
}

func TestDictDBCandidatesForEnv(t *testing.T) {
	t.Setenv("TDICT_DB", "D:/tmp/legacy.db")
	l := LocationsAt(filepath.Join(t.TempDir(), "config.json"))
	got := l.DictDBCandidatesFor("主机109", "")
	want := l.DictDBPath("主机109")
	if len(got) != 1 || got[0] != want {
		t.Fatalf("候选 = %v, want [%s]（默认不该回落到 erp_data.db，也不该被 TDICT_DB 覆盖）", got, want)
	}

	flag := filepath.Join(t.TempDir(), "flag.db")
	t.Setenv("TDICT_DB", "")
	got = l.DictDBCandidatesFor("主机109", flag)
	if got[0] != flag {
		t.Fatalf("-d 应优先: %v", got)
	}
}

func TestResolveDictDBForMissing(t *testing.T) {
	t.Setenv("TDICT_DB", "")
	l := LocationsAt(filepath.Join(t.TempDir(), "config.json"))
	if _, tried, err := l.ResolveDictDBFor("主机109", ""); err == nil || len(tried) == 0 {
		t.Fatalf("缺库应报错并列出尝试过的路径: err=%v tried=%v", err, tried)
	}

	p := l.DictDBPath("主机109")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := l.ResolveDictDBFor("主机109", "")
	if err != nil || got != p {
		t.Fatalf("ResolveDictDBFor = %q, %v; want %q", got, err, p)
	}
}

// TestDictDBCandidatesForHonorsDictDir 钉住「写侧与读侧指向同一个文件」。
//
// 回归用例：dict.dir 可配之后，同步/状态/Web 都换成了 DictDBPathForEnv，只有查询侧
// 还走 Locations.DictDBPath —— 用户把库根目录改到 D 盘后，同步成功、状态页显示"已同步"，
// 去查询却报"本地字典库不存在"；旧位置若还留着旧库，则静默读到过期数据。
func TestDictDBCandidatesForHonorsDictDir(t *testing.T) {
	t.Setenv("TDICT_DB", "")
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.json")
	libDir := filepath.Join(root, "custom-dict")
	if err := os.WriteFile(cfgPath, []byte(fmt.Sprintf(`{"dict":{"dir":%q}}`, libDir)), 0o644); err != nil {
		t.Fatal(err)
	}
	l := LocationsAt(cfgPath)
	want := filepath.Join(libDir, DictDBFileName("主机109"))
	if got := DictDBPathForEnv(cfgPath, "主机109"); got != want {
		t.Fatalf("写侧 = %q, want %q", got, want)
	}
	if got := l.DictDBCandidatesFor("主机109", ""); len(got) != 1 || got[0] != want {
		t.Fatalf("读侧候选 = %v, want [%s]（读写不同一处 = 同步写 A、查询读 B）", got, want)
	}
	// 库落地后必须能被解析出来，否则连报错文案里的路径都会指错地方。
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(want, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, _, err := l.ResolveDictDBFor("主机109", ""); err != nil || p != want {
		t.Fatalf("ResolveDictDBFor = %q, %v; want %q", p, err, want)
	}
}

// TestDictDBCandidatesForEnvOverride 环境级 dictDir 优先于顶层 dict.dir。
func TestDictDBCandidatesForEnvOverride(t *testing.T) {
	t.Setenv("TDICT_DB", "")
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.json")
	topDir := filepath.Join(root, "top-dict")
	envDir := filepath.Join(root, "env-dict")
	body := fmt.Sprintf(`{"dict":{"dir":%q},"hosts":{"sshs":[{"name":"主机109","host":"10.0.0.1","dictDir":%q}]}}`, topDir, envDir)
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	l := LocationsAt(cfgPath)
	want := filepath.Join(envDir, DictDBFileName("主机109"))
	if got := l.DictDBCandidatesFor("主机109", ""); len(got) != 1 || got[0] != want {
		t.Fatalf("候选 = %v, want [%s]", got, want)
	}
}
