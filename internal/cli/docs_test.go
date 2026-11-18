package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/testkit"
)

// runDocs 跑一次 `tg docs`，返回 stdout。格式走 common 的包级开关（--json/--format 是
// 根命令的持久标志，单测里没有根命令）。
func runDocs(t *testing.T, args ...string) string {
	t.Helper()
	return runDocsAs(t, true, args...)
}

// runDocsHuman 同上，但走人读表格分支。
func runDocsHuman(t *testing.T, args ...string) string {
	t.Helper()
	return runDocsAs(t, false, args...)
}

func runDocsAs(t *testing.T, json bool, args ...string) string {
	t.Helper()
	prevJSON, prevHuman := common.JSON, common.Human
	common.JSON, common.Human = json, !json
	t.Cleanup(func() { common.JSON, common.Human = prevJSON, prevHuman })

	_, out := testkit.CaptureStdout(t, func() int {
		cmd := newDocsCmd()
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Errorf("docs %v: %v", args, err)
			return 1
		}
		return 0
	})
	return out
}

// decodeDocs 把关掉表格分支后的 JSON 拆成报告。信封是 output.Emit 的 {"ok":..,"data":..}。
func decodeDocs(t *testing.T, out string) docsReport {
	t.Helper()
	var env struct {
		OK   bool       `json:"ok"`
		Data docsReport `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("不是 JSON: %v\n%s", err, out)
	}
	if !env.OK {
		t.Fatalf("ok=false: %s", out)
	}
	if env.Data.Schema != docsSchemaVersion {
		t.Fatalf("schema = %q", env.Data.Schema)
	}
	return env.Data
}

// 两份手册都没装：目录是缺省落点，installed=false。
func TestDocsBothMissing(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":6}`)
	rep := decodeDocs(t, runDocs(t))
	if len(rep.Docs) != 2 {
		t.Fatalf("docs = %d 条, want 2", len(rep.Docs))
	}
	want := map[string]string{
		"bdldoc": filepath.Join(dataDir, config.DocBDL.DirName),
		"gpdoc":  filepath.Join(dataDir, config.DocGP.DirName),
	}
	for _, d := range rep.Docs {
		if d.Dir != want[d.Key] {
			t.Errorf("%s dir = %q, want %q", d.Key, d.Dir, want[d.Key])
		}
		if d.Exists || d.Installed {
			t.Errorf("%s 目录还没建, exists/installed 应为 false", d.Key)
		}
		if d.ZipName == "" {
			t.Errorf("%s 缺 zipName, 没装时没得提示", d.Key)
		}
	}
}

// 装过一份：只有它 installed=true；--long 数文件数与体积。
func TestDocsInstalledAndLong(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":6}`)
	gpDir := filepath.Join(dataDir, config.DocGP.DirName)
	if err := os.MkdirAll(filepath.Join(gpDir, "01_环境架构"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := []byte("# 环境架构\n")
	if err := os.WriteFile(filepath.Join(gpDir, "01_环境架构", "0001-x.md"), body, 0o600); err != nil {
		t.Fatal(err)
	}

	rep := decodeDocs(t, runDocs(t))
	byKey := map[string]docEntry{}
	for _, d := range rep.Docs {
		byKey[d.Key] = d
	}
	if !byKey["gpdoc"].Installed || !byKey["gpdoc"].Exists {
		t.Fatalf("gpdoc 应 installed/exists=true: %+v", byKey["gpdoc"])
	}
	if byKey["bdldoc"].Installed {
		t.Fatalf("bdldoc 没装, 不该报 installed")
	}
	if byKey["gpdoc"].Files != 0 {
		t.Fatalf("没带 --long 不该数文件, got files=%d", byKey["gpdoc"].Files)
	}

	long := decodeDocs(t, runDocs(t, "--long"))
	for _, d := range long.Docs {
		if d.Key != "gpdoc" {
			continue
		}
		if d.Files != 1 || d.Bytes != int64(len(body)) {
			t.Fatalf("--long: files=%d bytes=%d, want 1/%d", d.Files, d.Bytes, len(body))
		}
	}
}

// 配了 gpdoc.dir：dir 与 configured 都跟着配置走。
func TestDocsHonoursConfiguredDir(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "docs", "gp")
	writeConfig(t, `{"schemaVersion":6,"gpdoc":{"dir":"`+filepath.ToSlash(custom)+`"}}`)
	rep := decodeDocs(t, runDocs(t))
	for _, d := range rep.Docs {
		if d.Key != "gpdoc" {
			continue
		}
		if d.Dir != custom {
			t.Fatalf("dir = %q, want %q", d.Dir, custom)
		}
		if d.Configured != custom {
			t.Fatalf("configured = %q, want %q", d.Configured, custom)
		}
	}
}

// 人读分支：表格里有目录与状态两栏（不解析列宽，只认关键字）。
func TestDocsTableMentionsBoth(t *testing.T) {
	dataDir := writeConfig(t, `{"schemaVersion":6}`)
	out := runDocsHuman(t)
	if out == "" {
		t.Fatal("表格分支没有输出")
	}
	for _, want := range []string{config.DocBDL.Title, config.DocGP.Title, filepath.Join(dataDir, config.DocBDL.DirName)} {
		if !strings.Contains(out, want) {
			t.Errorf("表格里没有 %q:\n%s", want, out)
		}
	}
}
