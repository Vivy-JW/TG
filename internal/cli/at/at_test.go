package at

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

func TestParse(t *testing.T) {
	good := []struct {
		raw    string
		env    string
		ent    int
		hasEnt bool
	}{
		{"主机109", "主机109", 0, false},
		{"主机109@99", "主机109", 99, true},
		{" 主机109@99 ", "主机109", 99, true},
		{"with space@100", "with space", 100, true},
		{"a@b@7", "a@b", 7, true}, // 从最后一个 @ 切
	}
	for _, c := range good {
		v, err := Parse(c.raw)
		if err != nil {
			t.Fatalf("Parse(%q) 不该报错: %v", c.raw, err)
		}
		if v.Env != c.env || v.Ent != c.ent || v.HasEnt != c.hasEnt {
			t.Errorf("Parse(%q) = %+v，想要 env=%q ent=%d hasEnt=%v", c.raw, v, c.env, c.ent, c.hasEnt)
		}
	}

	bad := []string{"", "   ", "@99", "主机109@", "主机109@abc", "主机109@0", "主机109@-3"}
	for _, raw := range bad {
		if _, err := Parse(raw); err == nil {
			t.Errorf("Parse(%q) 该报错", raw)
		}
	}
}

// fakeRoot 造一棵最小命令树：作用域表按路径匹配，不需要真的注册命令。
func fakeRoot() (*cobra.Command, map[string]*cobra.Command) {
	root := &cobra.Command{Use: "tt"}
	mk := func(path ...string) *cobra.Command {
		cur := root
		for _, name := range path {
			next := &cobra.Command{Use: name}
			cur.AddCommand(next)
			cur = next
		}
		return cur
	}
	cmds := map[string]*cobra.Command{}
	for _, p := range [][]string{
		{"debug", "start"}, {"debug", "exec"}, {"debug", "wslogs"}, {"debug", "source"},
		{"dict", "r.t"}, {"dict", "db", "ping"}, {"dict", "spill", "show"},
		{"session", "new"}, {"session", "job"}, {"version"},
	} {
		cmds[strings.Join(p, " ")] = mk(p...)
	}
	return root, cmds
}

func TestScopeOf(t *testing.T) {
	_, cmds := fakeRoot()
	cases := map[string]Scope{
		"debug start":     EnvEntReq,
		"debug exec":      EnvFilter,
		"debug wslogs":    Env,
		"debug source":    None,   // 读本机 serve，不认 --at
		"dict r.t":        EnvEnt, // 组级 EnvEnt 继承
		"dict db ping":    Env,    // dict db 覆盖成 Env
		"dict spill show": None,   // 本地文件，覆盖成 None
		"session new":     EnvEntReq,
		"session job":     None,
		"version":         None,
	}
	for path, want := range cases {
		if got := ScopeOf(cmds[path]); got != want {
			t.Errorf("%s 的作用域 = %v，想要 %v", path, got, want)
		}
	}
}

func TestApplyRejects(t *testing.T) {
	_, cmds := fakeRoot()
	oldRaw, oldEnt := Raw, Ent
	t.Cleanup(func() { Raw, Ent = oldRaw, oldEnt })

	cases := []struct {
		path string
		raw  string
		want string // 错误里必须出现的关键词
	}{
		{"version", "主机109", "不接受 --at"},
		{"debug wslogs", "主机109@99", "只接受 <环境>"},
		{"debug exec", "主机109@99", "--session 主机109/99"},
		{"debug start", "主机109", "需要 <环境>@<企业>"},
	}
	for _, c := range cases {
		Raw = c.raw
		err := Apply(cmds[c.path], nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("Apply(%s, %q) = %v，想要含 %q", c.path, c.raw, err, c.want)
		}
	}
}

func TestPrintContractNoneAndEnv(t *testing.T) {
	_, cmds := fakeRoot()
	oldJSON := common.JSON
	t.Cleanup(func() { common.JSON = oldJSON })

	common.JSON = false
	var buf bytes.Buffer
	if _, err := PrintContract(&buf, cmds["version"]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "没有目标参数") {
		t.Errorf("version 的 --at-list 该说明“没有目标参数”，得到:\n%s", buf.String())
	}

	buf.Reset()
	if _, err := PrintContract(&buf, cmds["session job"]); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "--session") {
		t.Errorf("session job 的 --at-list 该指路 --session，得到:\n%s", buf.String())
	}
}

func TestListTarget(t *testing.T) {
	root, _ := fakeRoot()
	if _, ok := ListTarget(root, []string{"debug", "start"}); ok {
		t.Fatal("没给 --at-list 却走了 --at-list 路径")
	}
	cmd, ok := ListTarget(root, []string{"debug", "start", "--at-list"})
	if !ok || cmd == nil || cmd.Name() != "start" {
		t.Fatalf("--at-list 应定位到 debug start，得到 %v %v", cmd, ok)
	}
	if _, ok := ListTarget(root, []string{"debug", "start", "--at-list=false"}); ok {
		t.Fatal("--at-list=false 应视为没给")
	}
}
