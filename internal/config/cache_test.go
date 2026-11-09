package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// buildCacheDir 造一个"配置目录":一份 config.json + 几个缓存子目录。
// 文件名与内容都不重要,要的是"哪些被删、哪些被留下"。
func buildCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"schemaVersion":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	// .tt-serve.json 是**运行中**服务的状态文件,删了 --stop 就找不着实例 —— 它不是缓存
	if err := os.WriteFile(filepath.Join(dir, ".tt-serve.json"), []byte(`{"pid":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, sub := range CacheSubdirs {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "x.bin"), []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestCacheStatusOf 统计只数缓存子目录,不把 config.json 算进去。
func TestCacheStatusOf(t *testing.T) {
	dir := buildCacheDir(t)
	st := CacheStatusOf(dir)

	if st.Dir != dir {
		t.Errorf("Dir = %q, 期望 %q", st.Dir, dir)
	}
	if len(st.Dirs) != len(CacheSubdirs) {
		t.Fatalf("应列出全部 %d 个缓存目录, got %d", len(CacheSubdirs), len(st.Dirs))
	}
	// 每个子目录一个 10 字节文件
	if st.Files != len(CacheSubdirs) || st.Bytes != int64(10*len(CacheSubdirs)) {
		t.Errorf("占用统计不符: %d 个文件 / %d 字节", st.Files, st.Bytes)
	}
	if st.MaxAgeHours != int(CacheMaxAge.Hours()) {
		t.Errorf("MaxAgeHours = %d", st.MaxAgeHours)
	}
}

// TestCleanCacheKeepsConfig 全清只删缓存 —— config.json 与运行中的服务状态文件必须原样留下。
// 这是这个功能最要紧的一条:它们与缓存同住一个目录,手滑一次就把用户的口令环境删了。
func TestCleanCacheKeepsConfig(t *testing.T) {
	dir := buildCacheDir(t)
	removed, freed, err := CleanCache(dir, 0)
	if err != nil {
		t.Fatalf("CleanCache: %v", err)
	}
	if removed != len(CacheSubdirs) || freed != int64(10*len(CacheSubdirs)) {
		t.Errorf("应删 %d 个文件 / %d 字节, got %d / %d",
			len(CacheSubdirs), 10*len(CacheSubdirs), removed, freed)
	}
	for _, keep := range []string{"config.json", ".tt-serve.json"} {
		if _, err := os.Stat(filepath.Join(dir, keep)); err != nil {
			t.Errorf("%s 不该被删: %v", keep, err)
		}
	}
	for _, sub := range CacheSubdirs {
		if _, err := os.Stat(filepath.Join(dir, sub)); !os.IsNotExist(err) {
			t.Errorf("缓存目录 %s 应已删除, got err=%v", sub, err)
		}
	}
	// 清完再统计应是全零,而不是把 config.json 算进去
	if st := CacheStatusOf(dir); st.Files != 0 || st.Bytes != 0 {
		t.Errorf("清完应为零, got %d 个文件 / %d 字节", st.Files, st.Bytes)
	}
}

// TestCleanCacheByAge 按年龄清:只删旧文件,新文件留着,空目录顺手回收。
//
// 启动清理走的就是这条 —— 刻意不"启动即全清":查询落盘的那些文件正是"刚才那条
// 查询的完整结果",启动就删会把上一条命令刚告诉用户的东西删掉。
func TestCleanCacheByAge(t *testing.T) {
	dir := buildCacheDir(t)
	old := filepath.Join(dir, "spill", "old.bin")
	fresh := filepath.Join(dir, "spill", "fresh.bin")
	for _, p := range []string{old, fresh} {
		if err := os.WriteFile(p, []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}

	removed, _, err := CleanCache(dir, time.Hour)
	if err != nil {
		t.Fatalf("CleanCache: %v", err)
	}
	// 每个缓存子目录里那份 10 字节的 x.bin 都是刚写的,只有 spill/old.bin 该走
	if removed != 1 {
		t.Errorf("只该删 1 个旧文件, got %d", removed)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("新文件不该被删: %v", err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf("旧文件应已删除, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); err != nil {
		t.Errorf("按年龄清也不该动 config.json: %v", err)
	}
}

// TestCleanCacheEmptyDir 定位不到配置目录时不该删任何东西(也不该 panic)。
func TestCleanCacheEmptyDir(t *testing.T) {
	if _, _, err := CleanCache("", 0); err == nil {
		t.Error("空目录应报错而不是静默成功")
	}
}
