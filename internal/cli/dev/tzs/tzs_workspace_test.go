package tzs

// 工作区的解析顺序（这条线唯一的必填项）：--workspace > TZSCLI_WS > **--env 指定环境**的 workspace，
// 末端**拒绝**而不是回落。
//
// 为什么值得一条测试：引擎内置的默认工作区是一个**真实客户目录**。三层都空却仍然启动，
// 就是拿别人的表单当草稿纸；而"切了环境还开着上一个客户的工作区"是同一件事的另一种形态。
// 这条测试同时钉住两件事：
//   - 工作区只按环境配（机器级那份已经没有了，见 config.TzsSettings）；
//   - **没有默认环境**：不点 --env 就是没指定，绝不回落到某个环境上（2026-10 移除 activeEnv）。

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// seedConfig 写一份带两个环境的配置，并把它指给 config.ResolvePath（TT_CONFIG 那条桥）。
func seedConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TT_CONFIG", path)
}

// 刻意**不写 activeEnv** —— 那个键已经不存在了；写了也不该被任何人读。
const twoEnvsConfig = `{
  "schemaVersion": 3,
  "hosts": {
    "sshs": [
      {"name": "甲", "host": "h1", "port": 22, "user": "u", "password": "p", "workspace": "D:\\ws\\jia"},
      {"name": "乙", "host": "h2", "port": 22, "user": "u", "password": "p", "workspace": "D:\\ws\\yi"}
    ]
  }
}`

func TestWorkspaceResolutionOrder(t *testing.T) {
	seedConfig(t, twoEnvsConfig)
	t.Setenv("TZSCLI_WS", "")

	// ① --workspace 最优先。
	o, err := tzsExecOptions(`D:\ws\flag`, "")
	if err != nil {
		t.Fatalf("tzsExecOptions: %v", err)
	}
	if o.Workspace != `D:\ws\flag` || o.WorkspaceSrc != "--workspace" {
		t.Errorf("① --workspace 该赢：ws=%q src=%q", o.Workspace, o.WorkspaceSrc)
	}

	// ② 其次是 TZSCLI_WS。
	t.Setenv("TZSCLI_WS", `D:\ws\envvar`)
	o, _ = tzsExecOptions("", "")
	if o.Workspace != `D:\ws\envvar` || o.WorkspaceSrc != "环境变量 TZSCLI_WS" {
		t.Errorf("② TZSCLI_WS 第二：ws=%q src=%q", o.Workspace, o.WorkspaceSrc)
	}

	// ③ 都没有 → **--env 点名那个环境**的那一份。
	t.Setenv("TZSCLI_WS", "")
	o, _ = tzsExecOptions("", "乙")
	if o.Workspace != `D:\ws\yi` {
		t.Errorf("③ 该取环境「乙」的工作区，得到 %q", o.Workspace)
	}
	if !strings.Contains(o.WorkspaceSrc, "乙") {
		t.Errorf("③ 来源该点名是哪个环境给的，得到 %q", o.WorkspaceSrc)
	}
}

// **不点环境就不再回落**：这条是 2026-10 移除 hosts.activeEnv 的核心承诺，
// 也是"连错机器却看起来一切正常"的根治。
func TestWorkspaceWithoutEnvIsRefused(t *testing.T) {
	seedConfig(t, twoEnvsConfig)
	t.Setenv("TZSCLI_WS", "")

	// 宽松版不报错（doctor 要能报告"没工作区"），但也**不许**替我们挑一个环境。
	o, err := tzsExecOptions("", "")
	if err != nil {
		t.Fatalf("宽松版不该报错: %v", err)
	}
	if o.Workspace != "" {
		t.Errorf("没点环境就不该取到任何环境的工作区，得到 %q（src=%q）", o.Workspace, o.WorkspaceSrc)
	}

	// 严格版拒绝，且提示必须告诉人用 --env 指定。
	if _, err := tzsEngineOptions("", ""); err == nil {
		t.Fatal("严格版该拒绝：没有环境就不该猜一个")
	} else if !strings.Contains(err.Error(), "--env") {
		t.Errorf("拒绝时的提示该指向 --env：%v", err)
	}
}

// 未配置工作时：宽松版（给 doctor / 动词索引用）不报错；严格版（运行类动词）拒绝并说清在哪填。
func TestWorkspaceMissingIsRefusedNotDefaulted(t *testing.T) {
	seedConfig(t, `{"schemaVersion": 3, "hosts": {"sshs": [
	  {"name": "甲", "host": "h1", "port": 22, "user": "u", "password": "p"}]}}`)
	t.Setenv("TZSCLI_WS", "")

	o, err := tzsExecOptions("", "甲")
	if err != nil {
		t.Fatalf("宽松版不该因为没有工作区而报错（doctor 要能报出这一项）: %v", err)
	}
	if o.Workspace != "" {
		t.Errorf("没配就是空串，得到 %q", o.Workspace)
	}

	if _, err := tzsEngineOptions("", "甲"); err == nil {
		t.Fatal("严格版该拒绝：引擎的默认工作区是真实客户目录，不能回落")
	} else {
		// 报错必须能自纠：说清在哪填（这一句是对外提示的一部分，用户照着它就能修好）。
		for _, want := range []string{"--workspace", "TZSCLI_WS", "工作区目录"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("拒绝时的提示里该有 %q：%v", want, err)
			}
		}
	}
}

// 一个环境都没配时也不 panic（首次运行：还没配环境就要问 doctor / 函数表）。
func TestWorkspaceResolutionWithoutAnyEnv(t *testing.T) {
	seedConfig(t, `{"schemaVersion": 3, "hosts": {"sshs": []}}`)
	t.Setenv("TZSCLI_WS", "")
	o, err := tzsExecOptions("", "")
	if err != nil {
		t.Fatalf("没有环境时不该报错: %v", err)
	}
	if o.Workspace != "" || o.WorkspaceSrc != "" {
		t.Errorf("没有环境就没有工作区，得到 ws=%q src=%q", o.Workspace, o.WorkspaceSrc)
	}
}
