package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tt/internal/testkit"
	"tt/internal/update"
)

// mkSkills 在 dir 下建一个技能源树：每个技能一个目录 + SKILL.md。
// 写的是**合规**的 frontmatter（name 等于目录名、description 非空）。
func mkSkills(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		writeSkill(t, dir, n, "---\nname: "+n+"\ndescription: 测试技能。\n---\n# "+n+"\n")
	}
}

// writeSkill 往 <dir>/<name>/SKILL.md 写任意内容（建目录）。
func writeSkill(t *testing.T, dir, name, body string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeConfig 在临时数据目录里落一份配置，并让本进程的 ResolveConfig 指过去。
func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dataDir := t.TempDir()
	cfg := filepath.Join(dataDir, "config.json")
	if err := os.WriteFile(cfg, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TT_CONFIG", cfg)
	return dataDir
}

// --to auto：当前目录已经为某个 agent 初始化过就别另起一个。
func TestResolveAutoTarget(t *testing.T) {
	cases := []struct {
		name string
		dirs []string // 预先创建的 agent 项目目录
		want string   // 期望落点（相对 cwd）
	}{
		{"都没有 → 跨客户端公约数", nil, filepath.Join(".agents", "skills")},
		{"只有 .claude → 用它", []string{".claude"}, filepath.Join(".claude", "skills")},
		{"只有 .cursor → 用它", []string{".cursor"}, filepath.Join(".cursor", "skills")},
		{"两个都有 → .agents 优先（读它的 agent 最多）", []string{".claude", ".agents"}, filepath.Join(".agents", "skills")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cwd := t.TempDir()
			for _, d := range c.dirs {
				if err := os.MkdirAll(filepath.Join(cwd, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			got, why, err := resolveAutoTarget(cwd)
			if err != nil {
				t.Fatalf("resolveAutoTarget: %v", err)
			}
			if want := filepath.Join(cwd, c.want); got != want {
				t.Errorf("落点 = %s, want %s", got, want)
			}
			if why == "" {
				t.Error("自动挑的落点必须给出理由 —— 用户得知道为什么装这儿")
			}
		})
	}
}

// --agent 的名字表与 --to/--agent 的互斥。
func TestResolveSkillsTargetByAgent(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, name := range []string{"agents", "universal", "claude-code", "claude", "copilot", "vscode", "cursor"} {
		dst, why, err := resolveSkillsTarget("", name)
		if err != nil {
			t.Fatalf("--agent %s: %v", name, err)
		}
		if !filepath.IsAbs(dst) {
			t.Errorf("--agent %s 的落点不是绝对路径: %s", name, dst)
		}
		if why == "" {
			t.Errorf("--agent %s 没给出「谁会读它」", name)
		}
	}
	if _, _, err := resolveSkillsTarget("", "nope"); err == nil {
		t.Error("不认识的 agent 名应报错（而不是静默装到某个默认位置）")
	}
	if _, _, err := resolveSkillsTarget(".x", "cursor"); err == nil {
		t.Error("--to 与 --agent 同时给应报错")
	}
	// 不给就是默认：<当前目录>/.agents/skills
	dst, why, err := resolveSkillsTarget("", "")
	if err != nil {
		t.Fatalf("默认落点: %v", err)
	}
	cwd, _ := os.Getwd()
	if want := filepath.Join(cwd, ".agents", skillsDirName); dst != want || why != "" {
		t.Errorf("默认落点 = %q（why=%q）, want %q（无解释）", dst, why, want)
	}
}

// 命令面的接线：--to auto 要真的落到探测出来的位置，而不只是 resolveAutoTarget 单测过。
func TestInstallSkillsCmdToAuto(t *testing.T) {
	src := t.TempDir()
	mkSkills(t, src, "alpha")
	cwd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cwd, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	writeConfig(t, `{"schemaVersion":4}`)

	old := skillsSource
	skillsSource = func() (string, error) { return src, nil }
	defer func() { skillsSource = old }()

	code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", "auto"})
		if err := cmd.Execute(); err != nil {
			t.Errorf("--to auto: %v", err)
			return 1
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".claude", "skills", "alpha", "SKILL.md")); err != nil {
		t.Fatalf("--to auto 没有落到已存在的 .claude/skills: %v", err)
	}
}

// 装完要留下"哪一版装的、装到了哪"这份戳 —— 升级后靠它把 agent 目录里那份刷成同版。
//
// 测试里 Version 是空的（go test 不注入版本号），而空版本刻意不写戳（见 install.go 的
// 说明）—— 所以这里显式给一个版本号，并把配置路径指到临时目录：戳落的是
// ResolveConfig 解析出的那个目录，指不过去就会写到开发者的真实数据目录里。
func TestInstallSkillsRecordsStamp(t *testing.T) {
	src := t.TempDir()
	mkSkills(t, src, "alpha")
	dataDir := writeConfig(t, `{"schemaVersion":4}`)
	target := filepath.Join(dataDir, "agent-skills")

	oldSrc, oldVer := skillsSource, Version
	skillsSource = func() (string, error) { return src, nil }
	Version = "9.9.9"
	defer func() { skillsSource, Version = oldSrc, oldVer }()

	if code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", target})
		if err := cmd.Execute(); err != nil {
			t.Errorf("install skills: %v", err)
			return 1
		}
		return 0
	}); code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}

	stamp := update.LoadSkillsStamp(dataDir)
	if stamp == nil {
		t.Fatalf("没留下技能戳：%s 不存在", update.SkillsStampPath(dataDir))
	}
	if stamp.Version != "9.9.9" {
		t.Errorf("戳里的版本 = %q，想要 9.9.9", stamp.Version)
	}
	if len(stamp.Targets) != 1 || !filepath.IsAbs(stamp.Targets[0]) {
		t.Errorf("戳里的目标该是那一个绝对路径: %v", stamp.Targets)
	}
	// 同版本不算漂移；换了版本才算。
	if got := update.StaleSkillsTargets(dataDir, "9.9.9"); len(got) != 0 {
		t.Errorf("同版本不该算漂移: %v", got)
	}
	if got := update.StaleSkillsTargets(dataDir, "10.0.0"); len(got) != 1 {
		t.Errorf("换版本该算漂移: %v", got)
	}
}

// 没有版本号的构建不写戳：`make build` 装技能不该在数据目录里种下一条"空版本"记录。
func TestInstallSkillsSkipsStampWithoutVersion(t *testing.T) {
	src := t.TempDir()
	mkSkills(t, src, "alpha")
	dataDir := writeConfig(t, `{"schemaVersion":4}`)

	oldSrc, oldVer := skillsSource, Version
	skillsSource = func() (string, error) { return src, nil }
	Version = ""
	defer func() { skillsSource, Version = oldSrc, oldVer }()

	if code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", filepath.Join(dataDir, "skills")})
		if err := cmd.Execute(); err != nil {
			t.Errorf("install skills: %v", err)
			return 1
		}
		return 0
	}); code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}
	if st := update.LoadSkillsStamp(dataDir); st != nil {
		t.Errorf("没有版本号时不该写戳，却写下了 %+v", st)
	}
}

// 登记的第三方技能要跟官方一起装；它的根级 assets/ 不复制，改成在 SKILL.md 首行写指针。
func TestInstallSkillsCmdInstallsRegisteredExternal(t *testing.T) {
	official := t.TempDir()
	mkSkills(t, official, "alpha")

	lib := t.TempDir()
	kb := filepath.Join(lib, "kb-faq")
	writeSkill(t, lib, "kb-faq", "---\nname: kb-faq\ndescription: 知识库\n---\n# kb-faq\n")
	if err := os.MkdirAll(filepath.Join(kb, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kb, "assets", "big.bin"), []byte("大文件"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, `{"schemaVersion":4,"skills":{"externals":[{"path":`+quoteJSON(t, kb)+`,"label":"kb-faq"}]}}`)

	old := skillsSource
	skillsSource = func() (string, error) { return official, nil }
	defer func() { skillsSource = old }()

	dst := filepath.Join(t.TempDir(), "skills")
	code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", dst})
		if err := cmd.Execute(); err != nil {
			t.Errorf("install skills: %v", err)
			return 1
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}

	for _, n := range []string{"alpha", "kb-faq"} {
		if _, err := os.Stat(filepath.Join(dst, n, "SKILL.md")); err != nil {
			t.Fatalf("%s 没装上: %v", n, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dst, "kb-faq", "assets")); !os.IsNotExist(err) {
		t.Errorf("assets/ 被复制过去了（err=%v）", err)
	}
	md, err := os.ReadFile(filepath.Join(dst, "kb-faq", "SKILL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), filepath.Join(kb, "assets")) {
		t.Errorf("SKILL.md 里没有指回源 assets 的那一行:\n%s", md)
	}
}

// 登记的路径失效：官方照装、退出码 0；--strict 才变成非零。这是用户明确要的语义
// ——「警告不影响其它技能安装」，同时给 CI 一条能发现"装漏了一个"的路。
func TestInstallSkillsCmdWarnsOnMissingExternal(t *testing.T) {
	official := t.TempDir()
	mkSkills(t, official, "alpha")
	gone := filepath.Join(t.TempDir(), "gone")
	writeConfig(t, `{"schemaVersion":4,"skills":{"externals":[{"path":`+quoteJSON(t, gone)+`}]}}`)

	old := skillsSource
	skillsSource = func() (string, error) { return official, nil }
	defer func() { skillsSource = old }()

	dst := filepath.Join(t.TempDir(), "skills")
	code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", dst})
		if err := cmd.Execute(); err != nil {
			t.Errorf("第三方失联不该让命令失败: %v", err)
			return 1
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}
	if _, err := os.Stat(filepath.Join(dst, "alpha", "SKILL.md")); err != nil {
		t.Fatalf("官方技能没装上: %v", err)
	}

	// --strict：同样的输入，退出码非零，但技能照装。
	dst2 := filepath.Join(t.TempDir(), "skills")
	var code2 int
	testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", dst2, "--strict"})
		err := cmd.Execute()
		code2 = exitCodeOf(err)
		return 0
	})
	if code2 == 0 {
		t.Error("--strict 下有告警该返回非零退出码")
	}
	if _, err := os.Stat(filepath.Join(dst2, "alpha", "SKILL.md")); err != nil {
		t.Errorf("--strict 不该阻止其它技能安装: %v", err)
	}
}

// --dry-run：一个字节都不写（连目标目录都不建）。
func TestInstallSkillsCmdDryRun(t *testing.T) {
	official := t.TempDir()
	mkSkills(t, official, "alpha")
	writeConfig(t, `{"schemaVersion":4}`)

	old := skillsSource
	skillsSource = func() (string, error) { return official, nil }
	defer func() { skillsSource = old }()

	dst := filepath.Join(t.TempDir(), "skills")
	code := testkit.Silent(t, func() int {
		cmd := newInstallSkillsCmd()
		cmd.SetArgs([]string{"--to", dst, "--dry-run"})
		if err := cmd.Execute(); err != nil {
			t.Errorf("--dry-run: %v", err)
			return 1
		}
		return 0
	})
	if code != 0 {
		t.Fatalf("退出码 = %d, want 0", code)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("--dry-run 建了目标目录: %v", err)
	}
}

// quoteJSON 把 Windows 路径塞进 JSON 字符串（反斜杠要转义）。
func quoteJSON(t *testing.T, s string) string {
	t.Helper()
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}
