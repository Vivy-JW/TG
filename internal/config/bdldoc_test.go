package config

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// 落点三级：配置的 bdldoc.dir > <数据目录>/BDL-Markdown。配置坏掉也要回退缺省。
func TestBDLDocDirFor(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.json")

	// 没配置文件：缺省落点。
	if got, want := BDLDocDirFor(cfgPath), filepath.Join(LocationsAt(cfgPath).DataDir(), BDLDocDirName); got != want {
		t.Fatalf("缺省 = %q, want %q", got, want)
	}

	// 配了 bdldoc.dir：用它。
	custom := filepath.Join(root, "docs", "bdl")
	if err := os.WriteFile(cfgPath, []byte(`{"bdldoc":{"dir":`+quoteJSON(custom)+`}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := BDLDocDirFor(cfgPath); got != custom {
		t.Fatalf("配置落点 = %q, want %q", got, custom)
	}

	// 配置写坏：安静回退缺省，不返回空串（空串会让 install bdl 相对当前目录落盘）。
	if err := os.WriteFile(cfgPath, []byte(`{"bdldoc":`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, want := BDLDocDirFor(cfgPath), filepath.Join(LocationsAt(cfgPath).DataDir(), BDLDocDirName); got != want {
		t.Fatalf("坏配置应回退缺省，got %q want %q", got, want)
	}
}

// 状态：exists 看目录、installed 看"里面有没有东西"，两者要能分开。
func TestBDLDocStatusFor(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "config.json")
	dir := filepath.Join(root, "docs")

	st := BDLDocStatusFor(cfgPath)
	if st.Exists || st.Installed {
		t.Fatalf("目录不存在时不该报已装: %+v", st)
	}
	if want := filepath.Join(LocationsAt(cfgPath).DataDir(), BDLDocDirName); st.Dir != want {
		t.Fatalf("生效目录 = %q, want %q", st.Dir, want)
	}
	if st.Configured != "" {
		t.Fatalf("没配过时 Configured 应当为空，got %q", st.Configured)
	}

	// 空目录：存在但没装。
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(`{"bdldoc":{"dir":`+quoteJSON(dir)+`}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st = BDLDocStatusFor(cfgPath)
	if !st.Exists || st.Installed {
		t.Fatalf("空目录 = exists 且未装: %+v", st)
	}
	if st.Configured != dir {
		t.Fatalf("Configured = %q, want %q", st.Configured, dir)
	}

	// 有东西：已装。
	if err := os.WriteFile(filepath.Join(dir, "01_x.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st = BDLDocStatusFor(cfgPath); !st.Exists || !st.Installed {
		t.Fatalf("有文件时应报已装: %+v", st)
	}
}

// quoteJSON 把路径写成 JSON 字符串（Windows 路径里有反斜杠，得转义）。
func quoteJSON(s string) string { return strconv.Quote(s) }
