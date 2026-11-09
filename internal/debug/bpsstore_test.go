package debug

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 断点存档（bpsstore.go）：<DataDir>/debug-bps/<product>/<module>__<prog>.json 的落盘与读取。
//
// **本文件只管持久化**。文件头注释说"恢复时按行文本重新定位，源码变更不会把断点恢复到
// 错误位置" —— 那套重定位逻辑**不在这里**（在 session 那一侧），所以别以为这条测试
// 覆盖了"重定位对不对"。这里钉的是：路径怎么拼、写不写得下去、读不读得回来、
// 以及 dataDir 为空时**不落盘也不报错**。
//
// 从前的名字叫 persist_test.go，实际测的却是 session/manager —— 文件名与内容不符，
// 让人以为断点存档有测试。改名之后这块是**零覆盖**，这个文件补的就是它。

func sampleBPs() []StoredBP {
	return []StoredBP{
		{File: "prog.4gl", Line: 42, LineText: "  LET g_x = 1", Func: "main", Enabled: true},
		{File: "prog.4gl", Line: 88, LineText: "END MAIN", Func: "main", Enabled: false},
	}
}

func TestBPStorePath(t *testing.T) {
	cases := []struct {
		name         string
		module, prog string
		wantLeaf     string
	}{
		{"普通名字原样", "mta", "adzi999", "mta__adzi999.json"},
		{"点与连字符保留（正则把它们当安全字符）", "a.b-c", "p1", "a.b-c__p1.json"},
		{"下划线保留", "a_b", "p_2", "a_b__p_2.json"},
		{"斜杠被换成下划线", "a/b", "c\\d", "a_b__c_d.json"},
		// "区域:36" → "___36"，再拼 "__"，再 "程 序" → "___"：
		// 合起来 36 后面是 2+3=5 个下划线。
		{"冒号空格中文都换掉", "区域:36", "程 序", "___36_____.json"},
		{"空名字也给得出路径", "", "", "__.json"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := bpStorePath(filepath.Join("D:", "data"), "t100", c.module, c.prog)
			if filepath.Base(got) != c.wantLeaf {
				t.Errorf("文件名得 %q，想 %q", filepath.Base(got), c.wantLeaf)
			}
			if filepath.Dir(got) != filepath.Join("D:", "data", "debug-bps", "t100") {
				t.Errorf("该落在 <dataDir>/debug-bps/t100/ 下，得 %q", filepath.Dir(got))
			}
		})
	}
}

// TestBPStorePathNeverEscapesTheDir 路径段里的分隔符全被换掉，
// 所以 module/prog 里塞 `../` 也逃不出 debug-bps/。
//
// 这条值得单独钉：module 与 prog 来自配置与会话参数，不是常量。
func TestBPStorePathNeverEscapesTheDir(t *testing.T) {
	root := filepath.Join("D:", "data")
	base := filepath.Join(root, "debug-bps", "t100")
	for _, c := range []struct{ module, prog string }{
		{"..", ".."},
		{"../../etc", "passwd"},
		{`..\..\windows`, "system32"},
		{"a/../../b", "c"},
	} {
		got := bpStorePath(root, "t100", c.module, c.prog)
		if filepath.Dir(got) != base {
			t.Errorf("module=%q prog=%q 逃出了 debug-bps/：得 %q", c.module, c.prog, got)
		}
		if strings.Contains(filepath.Base(got), string(filepath.Separator)) {
			t.Errorf("文件名里不该还有分隔符：%q", filepath.Base(got))
		}
	}
}

// TestSaveBPsWithNoDataDirIsANoOp dataDir 为空 = 没开持久化：
// **不落盘、不报错**。报错的话，没配持久化的人每次下断点都会收到一条失败。
func TestSaveBPsWithNoDataDirIsANoOp(t *testing.T) {
	if err := saveBPs("", "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatalf("dataDir 为空该静默跳过，得 %v", err)
	}
	// 读也一样给一条明确的错（由调用方忽略），文案要能让人看懂为什么读不到。
	_, err := loadBPs("", "t100", "mta", "adzi999")
	if err == nil {
		t.Fatal("dataDir 为空时读该报错（由调用方忽略），不该静默返回空")
	}
	if !strings.Contains(err.Error(), "未启用断点持久化") {
		t.Errorf("文案该说清是没开持久化，得 %q", err.Error())
	}
}

func TestBPsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatalf("写：%v", err)
	}
	st, err := loadBPs(dir, "t100", "mta", "adzi999")
	if err != nil {
		t.Fatalf("读：%v", err)
	}
	if st.Module != "mta" || st.Prog != "adzi999" {
		t.Errorf("模块/程序名没带回来：%+v", st)
	}
	if len(st.Breakpoints) != 2 {
		t.Fatalf("断点数不对：%d", len(st.Breakpoints))
	}
	// LineText 是**重定位依据**，丢了的话源码一变断点就落到错位置 —— 必须原样带回来。
	if st.Breakpoints[0].LineText != "  LET g_x = 1" || st.Breakpoints[0].Line != 42 {
		t.Errorf("第一个断点没还原：%+v", st.Breakpoints[0])
	}
	// 启用态也要带回来（只存位置不存开关的话，恢复后全变回启用）。
	if !st.Breakpoints[0].Enabled || st.Breakpoints[1].Enabled {
		t.Errorf("启用态没还原：%+v", st.Breakpoints)
	}
	if st.SavedAt == "" {
		t.Error("该记下保存时间")
	}
}

// TestSaveBPsIsAtomic 写盘走"临时文件 + Rename"：落盘后目录里不该留下 .tmp。
//
// 落盘失败的后果是断点丢失，而"半个文件"更坏 —— 下次读会解析失败，
// 那条错误看起来像存档坏了，其实只是上次写没写完。
func TestSaveBPsIsAtomic(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "debug-bps", "t100")
	entries, err := os.ReadDir(sub)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if len(names) != 1 || names[0] != "mta__adzi999.json" {
		t.Errorf("目录里该只有存档本身，实得 %v", names)
	}
}

// TestSaveBPsCreatesTheDir 存档目录不存在时要自己建 ——
// 首次下断点就撞上"目录不存在"会很费解。
func TestSaveBPsCreatesTheDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "还没建过", "再深一层")
	if err := saveBPs(dir, "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatalf("该自己建目录：%v", err)
	}
	if _, err := loadBPs(dir, "t100", "mta", "adzi999"); err != nil {
		t.Errorf("建完之后该读得回来：%v", err)
	}
}

// TestSaveBPsOverwrites 同一模块/程序再存一次要覆盖，不是追加。
func TestSaveBPsOverwrites(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatal(err)
	}
	one := []StoredBP{{File: "prog.4gl", Line: 7, LineText: "x", Enabled: true}}
	if err := saveBPs(dir, "t100", "mta", "adzi999", one); err != nil {
		t.Fatal(err)
	}
	st, err := loadBPs(dir, "t100", "mta", "adzi999")
	if err != nil {
		t.Fatal(err)
	}
	if len(st.Breakpoints) != 1 || st.Breakpoints[0].Line != 7 {
		t.Errorf("该被覆盖成最新那份，得 %+v", st.Breakpoints)
	}
}

// TestLoadBPsRejectsBadArchive 读的失败一律**报错**，由调用方决定怎么降级
// （与 entdir 的快照不同：那边的约定是"读坏了当没有"，这边是"报出来让调用方看"）。
func TestLoadBPsRejectsBadArchive(t *testing.T) {
	dir := t.TempDir()
	t.Run("文件不存在", func(t *testing.T) {
		if _, err := loadBPs(dir, "t100", "mta", "没存过"); err == nil {
			t.Error("没存过该报错")
		}
	})
	t.Run("不是 JSON", func(t *testing.T) {
		p := bpStorePath(dir, "t100", "mta", "坏的")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("{ 这不是 json"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := loadBPs(dir, "t100", "mta", "坏的"); err == nil {
			t.Error("坏存档该报错")
		}
	})
}

// TestBPsAreJSONWithStableKeys 存档是**对外可读的文件**（人也要能看、能改），
// 所以键名是契约。`lineText` 带 omitempty（旧存档可能没有），`enabled` 不带
// （缺了会被读成 false，把启用态悄悄翻掉）。
func TestBPsAreJSONWithStableKeys(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "adzi999", sampleBPs()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(bpStorePath(dir, "t100", "mta", "adzi999"))
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("落盘的不是合法 JSON：%v", err)
	}
	for _, k := range []string{"module", "prog", "savedAt", "breakpoints"} {
		if _, ok := raw[k]; !ok {
			t.Errorf("顶层该有 %q 键：%v", k, raw)
		}
	}
	bps, _ := raw["breakpoints"].([]any)
	if len(bps) == 0 {
		t.Fatal("breakpoints 该是个非空数组")
	}
	first, _ := bps[0].(map[string]any)
	for _, k := range []string{"file", "line", "enabled"} {
		if _, ok := first[k]; !ok {
			t.Errorf("断点该有 %q 键：%v", k, first)
		}
	}
	if _, ok := first["lineText"]; !ok {
		t.Errorf("断点该有 lineText 键（重定位依据）：%v", first)
	}
}

// 产品命名空间：T100 与 GP 的同名 module/prog 不能共用同一份断点。
// T100 内部仍按原行为跨环境共享（A 方案）。
func TestBPsProductNamespace(t *testing.T) {
	dir := t.TempDir()
	if err := saveBPs(dir, "t100", "mta", "same", []StoredBP{{File: "a.4gl", Line: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := saveBPs(dir, "gp", "mta", "same", []StoredBP{{File: "b.4gl", Line: 2}, {File: "b.4gl", Line: 3}}); err != nil {
		t.Fatal(err)
	}
	t100, err := loadBPs(dir, "t100", "mta", "same")
	if err != nil || len(t100.Breakpoints) != 1 || t100.Breakpoints[0].File != "a.4gl" {
		t.Fatalf("T100 断点被污染：%+v, err=%v", t100, err)
	}
	gp, err := loadBPs(dir, "gp", "mta", "same")
	if err != nil || len(gp.Breakpoints) != 2 || gp.Breakpoints[0].File != "b.4gl" {
		t.Fatalf("GP 断点被污染：%+v, err=%v", gp, err)
	}
}

// 旧扁平存档只给 T100 兼容读取；GP 不能读到 T100 时代的同名程序断点。
func TestBPsLegacyFallbackOnlyT100(t *testing.T) {
	dir := t.TempDir()
	legacy := bpLegacyStorePath(dir, "mta", "oldprog")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(bpStoreFile{Module: "mta", Prog: "oldprog", Breakpoints: []StoredBP{{File: "old.4gl", Line: 9}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if st, err := loadBPs(dir, "t100", "mta", "oldprog"); err != nil || len(st.Breakpoints) != 1 {
		t.Fatalf("T100 应兼容旧扁平存档：%+v, err=%v", st, err)
	}
	if _, err := loadBPs(dir, "gp", "mta", "oldprog"); err == nil {
		t.Fatal("GP 不该读取 T100 旧扁平存档")
	}
}
