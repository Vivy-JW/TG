package outline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/cli/common"
	"tt/internal/dev/fgl"
	"tt/internal/output"
	"tt/internal/testkit"
)

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("..", "..", "..", "testdata", "fgl-fixtures", name)
}

func parseFixture(t *testing.T, name string) []fgl.OutlineNode {
	t.Helper()
	b, err := os.ReadFile(fixturePath(t, name))
	if err != nil {
		t.Fatalf("读夹具失败: %v", err)
	}
	return fgl.ParseOutline(string(b))
}

func captureRun(t *testing.T, format string, fn func() error) (string, error) {
	t.Helper()
	oldHuman, oldJSON, oldCSV, oldFormat, oldEnv := common.Human, common.JSON, common.CSV, common.Format, common.Env
	defer func() {
		common.Human, common.JSON, common.CSV, common.Format, common.Env = oldHuman, oldJSON, oldCSV, oldFormat, oldEnv
	}()
	common.Human, common.JSON, common.CSV, common.Format, common.Env = false, false, false, "json", ""
	switch format {
	case "human":
		common.Human = true
	case "csv":
		common.CSV = true
	}
	var runErr error
	_, out := testkit.CaptureStdout(t, func() int {
		runErr = fn()
		return 0
	})
	return out, runErr
}

func TestParseDepth(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int
		bad  bool
	}{
		{"all", -1, false}, {"1", 1, false}, {"2", 2, false},
		{"0", 0, true}, {"-1", 0, true}, {"x", 0, true},
	} {
		got, err := parseDepth(tc.in)
		if tc.bad {
			if err == nil {
				t.Errorf("parseDepth(%q) 应报错", tc.in)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("parseDepth(%q) = %d, %v; want %d", tc.in, got, err, tc.want)
		}
	}
}

func TestParseKinds(t *testing.T) {
	got, err := parseKinds([]string{"function, dialog", "SUB"})
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []fgl.OutlineKind{fgl.KindFunction, fgl.KindDialog, fgl.KindSub} {
		if !got[k] {
			t.Errorf("缺少 kind %s", k)
		}
	}
	if _, err := parseKinds([]string{"NOPE"}); err == nil {
		t.Fatal("未知 kind 应报错")
	}
}

func TestFilterKindsKeepsAncestorPath(t *testing.T) {
	nodes := parseFixture(t, "dialog-command.4gl")
	got := filterKinds(nodes, map[fgl.OutlineKind]bool{fgl.KindDialog: true})
	if len(got) != 1 || got[0].Kind != fgl.KindFunction {
		t.Fatalf("应保留 FUNCTION 祖先，got %+v", got)
	}
	if len(got[0].Children) != 1 || got[0].Children[0].Kind != fgl.KindDialog {
		t.Fatalf("应只保留 DIALOG 命中节点，got %+v", got[0].Children)
	}
	if len(got[0].Children[0].Children) != 0 {
		t.Fatalf("未命中的 INPUT/SUB 不应出现在结果里: %+v", got[0].Children[0].Children)
	}
}

func TestBuildTreeDepthMarksTruncation(t *testing.T) {
	nodes := parseFixture(t, "input-on-key.4gl")
	tree, cut := buildTree(nodes, 1, 1)
	if !cut || len(tree) != 1 || !tree[0].ChildrenTruncated || len(tree[0].Children) != 0 {
		t.Fatalf("depth=1 应只留根并标 childrenTruncated: %+v cut=%v", tree, cut)
	}
}

func TestFlattenParentIndexesAreGlobal(t *testing.T) {
	nodes := parseFixture(t, "input-on-key.4gl")
	tree, _ := buildTree(nodes, 1, -1)
	flat := flatten(tree, 0, nil, false)
	if len(flat) != 3 {
		t.Fatalf("flat 节点数 = %d, want 3", len(flat))
	}
	if flat[0].Parent != nil {
		t.Errorf("根 parent 应为 nil: %+v", flat[0].Parent)
	}
	if flat[2].Parent == nil || *flat[2].Parent != 1 {
		t.Errorf("SUB 的 parent 应为 INPUT 的下标 1: %+v", flat[2].Parent)
	}
}

func TestBriefRemovesSelection(t *testing.T) {
	nodes := parseFixture(t, "input-on-key.4gl")
	tree, _ := buildTree(nodes, 1, -1)
	data := resultData(&sourceInputs{Path: "x.4gl", Mode: "file"}, false, true, 3, tree, false)
	got := data.Nodes.([]treeNode)
	if got[0].SelStart != nil || got[0].SelEnd != nil {
		t.Fatalf("--brief 应去掉 selStart/selEnd: %+v", got[0])
	}
}

func TestHumanTree(t *testing.T) {
	nodes := parseFixture(t, "input-on-key.4gl")
	tree, _ := buildTree(nodes, 1, -1)
	var b strings.Builder
	if err := writeTreeHuman(&b, tree, false, false); err != nil {
		t.Fatal(err)
	}
	want := "FUNCTION inp_key  [2-7]\n  INPUT BY NAME g_cust  [3-6]\n    ON KEY (F1, Control-z)  [4-5]\n"
	if b.String() != want {
		t.Fatalf("human tree:\n%q\nwant:\n%q", b.String(), want)
	}
}

func TestRunDefaultsToJSON(t *testing.T) {
	out, err := captureRun(t, "json", func() error {
		return (&options{depth: "all"}).run(fixturePath(t, "input-on-key.4gl"))
	})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		OK     bool   `json:"ok"`
		Schema string `json:"schema"`
		Data   struct {
			Kind     string `json:"kind"`
			File     string `json:"file"`
			LineBase int    `json:"lineBase"`
			Total    int    `json:"total"`
			Nodes    []any  `json:"nodes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("输出不是 JSON: %v\n%s", err, out)
	}
	if !got.OK || got.Schema != "tt.outline/v1" || got.Data.Kind != "outline.tree" ||
		got.Data.LineBase != 1 || got.Data.Total != 3 || len(got.Data.Nodes) != 1 {
		t.Fatalf("JSON 信封不符: %+v", got)
	}
	if !strings.HasSuffix(got.Data.File, "input-on-key.4gl") {
		t.Errorf("file = %q", got.Data.File)
	}
}

func TestRunHuman(t *testing.T) {
	out, err := captureRun(t, "human", func() error {
		return (&options{depth: "all"}).run(fixturePath(t, "input-on-key.4gl"))
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "FUNCTION inp_key  [2-7]") || !strings.Contains(out, "ON KEY") {
		t.Fatalf("human 输出不符:\n%s", out)
	}
}

func TestRunRejectsCSV(t *testing.T) {
	_, err := captureRun(t, "csv", func() error {
		return (&options{depth: "all"}).run(fixturePath(t, "input-on-key.4gl"))
	})
	var oe *output.Error
	if !errors.As(err, &oe) || oe.Code != output.CodeUsage {
		t.Fatalf("CSV 应报 USAGE，got %v", err)
	}
}
