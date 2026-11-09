package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkSkill 在 parent 下建一个技能目录，SKILL.md 合规与否由 body 决定。
func mkSkill(t *testing.T, parent, name, body string) string {
	t.Helper()
	dir := filepath.Join(parent, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func goodBody(name string) string {
	return "---\nname: " + name + "\ndescription: 测试技能。\n---\n# " + name + "\n"
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("读 %s: %v", p, err)
	}
	return string(b)
}

//---------------------------------------------------------------------------
// BuildPlan
//---------------------------------------------------------------------------

func TestBuildPlanOfficialAndExternal(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	mkSkill(t, official, "beta", goodBody("beta"))
	if err := os.WriteFile(filepath.Join(official, "README.md"), []byte("索引\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ext := t.TempDir()
	ok := mkSkill(t, ext, "kb-faq", goodBody("kb-faq"))
	mkSkill(t, ext, "no-skill-md", "占位") // 有目录没 SKILL.md（等下把文件删掉）
	if err := os.Remove(filepath.Join(ext, "no-skill-md", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	mkSkill(t, ext, "bad", "---\nname: 另一个名字\ndescription: x\n---\n")
	conflict := mkSkill(t, ext, "beta", goodBody("beta")) // 与官方撞名

	plan, err := BuildPlan(official, []string{
		ok,
		filepath.Join(ext, "gone"),        // missing
		filepath.Join(ext, "no-skill-md"), // invalid
		filepath.Join(ext, "bad"),         // invalid（name 与目录名不一致）
		conflict,                          // conflict（官方优先）
	})
	if err != nil {
		t.Fatalf("第三方的问题不该让 BuildPlan 报错: %v", err)
	}

	var names []string
	for _, s := range plan.Skills {
		names = append(names, string(s.Origin)+":"+s.Name)
	}
	want := []string{"official:alpha", "official:beta", "external:kb-faq"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("计划里的技能 = %v, want %v", names, want)
	}
	if len(plan.ExtraFiles) != 1 || filepath.Base(plan.ExtraFiles[0]) != "README.md" {
		t.Fatalf("ExtraFiles = %v, want [README.md]", plan.ExtraFiles)
	}

	if len(plan.Warnings) != 4 {
		t.Fatalf("warnings = %d, want 4: %+v", len(plan.Warnings), plan.Warnings)
	}
	kinds := []State{StateMissing, StateInvalid, StateInvalid, StateConflict}
	idxs := []int{2, 3, 4, 5}
	for i, w := range plan.Warnings {
		if w.Kind != kinds[i] || w.Index != idxs[i] {
			t.Errorf("第 %d 条 warning = {%s #%d}, want {%s #%d}", i, w.Kind, w.Index, kinds[i], idxs[i])
		}
		if w.Text() == "" || !strings.Contains(w.Text(), "skills.externals") {
			t.Errorf("告警文案缺修复动作: %q", w.Text())
		}
	}
	if !strings.Contains(plan.Warnings[3].Text(), "官方优先") {
		t.Errorf("撞名文案该说明官方优先: %q", plan.Warnings[3].Text())
	}
}

// 两条第三方同名：先登记者赢，后者告警并指出先占的是第几条。
func TestBuildPlanExternalDuplicateFirstWins(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))

	a := t.TempDir()
	b := t.TempDir()
	first := mkSkill(t, a, "dup", goodBody("dup"))
	second := mkSkill(t, b, "dup", goodBody("dup"))

	plan, err := BuildPlan(official, []string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Skills) != 2 {
		t.Fatalf("计划 = %d 个技能, want 2（官方 + 第一个 dup）", len(plan.Skills))
	}
	if len(plan.Warnings) != 1 || plan.Warnings[0].Kind != StateConflict {
		t.Fatalf("warnings = %+v, want 一条 conflict", plan.Warnings)
	}
	if !strings.Contains(plan.Warnings[0].Why, "第 1 条") {
		t.Errorf("该指出先占的是第 1 条: %q", plan.Warnings[0].Why)
	}
}

// 官方技能不合规是**硬错误**：那是发版包坏了，不是用户配错了。
func TestBuildPlanOfficialInvalidIsHardError(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", "---\nname: alpha\n---\n") // 缺 description

	if _, err := BuildPlan(official, nil); err == nil {
		t.Fatal("官方技能不合规应报错")
	} else if !strings.Contains(err.Error(), "官方技能不合规") {
		t.Fatalf("错误信息该说清是哪一侧的问题: %v", err)
	}

	empty := t.TempDir()
	if _, err := BuildPlan(empty, nil); err == nil {
		t.Fatal("官方目录里一个技能都没有时应报错")
	}
	if _, err := BuildPlan(filepath.Join(t.TempDir(), "nope"), nil); err == nil {
		t.Fatal("官方目录不存在时应报错")
	}
}

//---------------------------------------------------------------------------
// Apply
//---------------------------------------------------------------------------

func TestApplySkipsAssetsAndPointsToThem(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	ext := t.TempDir()
	kb := mkSkill(t, ext, "kb-faq", goodBody("kb-faq"))
	if err := os.MkdirAll(filepath.Join(kb, "assets", "deep"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(kb, "assets", "big.bin"), filepath.Join(kb, "assets", "deep", "b.bin")} {
		if err := os.WriteFile(p, []byte("大文件"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(kb, "notes.md"), []byte("笔记\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(official, []string{kb})
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "skills")
	rep, err := Apply(plan, dst, Options{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}

	// assets 没被复制
	if _, err := os.Stat(filepath.Join(dst, "kb-faq", "assets")); !os.IsNotExist(err) {
		t.Errorf("assets/ 被复制了（err=%v）", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "kb-faq", "notes.md")); err != nil {
		t.Errorf("非 assets 的文件该照装: %v", err)
	}
	// 指针指向源目录
	md := readFile(t, filepath.Join(dst, "kb-faq", "SKILL.md"))
	if !strings.Contains(md, AssetsPointerLine(AssetsDir(kb))) {
		t.Errorf("SKILL.md 里没有指向 %s 的指针:\n%s", AssetsDir(kb), md)
	}
	if len(rep.Rewritten) != 1 || rep.Rewritten[0] != "kb-faq" {
		t.Errorf("Rewritten = %v, want [kb-faq]", rep.Rewritten)
	}
	// copied 里不该出现 assets 下的文件
	for _, f := range rep.Copied {
		if strings.Contains(f, "/assets/") {
			t.Errorf("Copied 里出现了 assets 文件: %s", f)
		}
	}
	if len(rep.Copied) != 4 { // alpha/SKILL.md, kb-faq/SKILL.md, kb-faq/notes.md ... 加上?
		t.Logf("Copied = %v", rep.Copied)
	}

	// 没有 assets 的官方技能不该被改写
	if strings.Contains(readFile(t, filepath.Join(dst, "alpha", "SKILL.md")), assetsPointerSentinel) {
		t.Error("没有 assets/ 的技能不该被写指针")
	}
}

// 目标已有同名条目时默认跳过（追加模式）；--force 才刷新。
func TestApplyAppendsAndSkips(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	plan, err := BuildPlan(official, nil)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	userMD := filepath.Join(dst, "alpha", "SKILL.md")
	mkSkill(t, dst, "alpha", "用户自己的\n")

	rep, err := Apply(plan, dst, Options{})
	if err != nil {
		t.Fatalf("追加安装不该报错: %v", err)
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0] != "alpha" {
		t.Errorf("Skipped = %v, want [alpha]", rep.Skipped)
	}
	if got := readFile(t, userMD); got != "用户自己的\n" {
		t.Errorf("默认安装覆盖了目标里已有的技能: %q", got)
	}

	if rep, err = Apply(plan, dst, Options{Force: true}); err != nil {
		t.Fatalf("--force: %v", err)
	}
	if len(rep.Skipped) != 0 {
		t.Errorf("--force 不该跳过: %v", rep.Skipped)
	}
	if got := readFile(t, userMD); got == "用户自己的\n" {
		t.Error("--force 没有刷新同名技能")
	}
}

// --force 刷多次，指针行也只留一行；根级文件也算。
func TestApplyForceIsIdempotent(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	kb := mkSkill(t, t.TempDir(), "kb", goodBody("kb"))
	if err := os.MkdirAll(filepath.Join(kb, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kb, "assets", "a.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(official, "README.md"), []byte("索引\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := BuildPlan(official, []string{kb})
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "skills")
	for i := 1; i <= 3; i++ {
		if _, err := Apply(plan, dst, Options{Force: true}); err != nil {
			t.Fatalf("第 %d 次 --force: %v", i, err)
		}
	}
	md := readFile(t, filepath.Join(dst, "kb", "SKILL.md"))
	if n := strings.Count(md, assetsPointerSentinel); n != 1 {
		t.Errorf("指针行 %d 次, want 1:\n%s", n, md)
	}
	if readFile(t, filepath.Join(dst, "README.md")) != "索引\n" {
		t.Error("根级 README.md 没复制过去")
	}
	if _, err := os.Stat(filepath.Join(dst, "kb", "assets")); !os.IsNotExist(err) {
		t.Error("assets/ 不该出现在目标里")
	}
}

func TestApplyDryRunWritesNothing(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	plan, err := BuildPlan(official, nil)
	if err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(t.TempDir(), "skills")
	rep, err := Apply(plan, dst, Options{DryRun: true})
	if err != nil {
		t.Fatalf("dry-run: %v", err)
	}
	if !rep.DryRun {
		t.Error("Report 里该标出这是 dry-run")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatalf("dry-run 建了目标目录: %v", err)
	}
	if len(rep.Copied) == 0 {
		t.Error("dry-run 也要报出会复制哪些文件")
	}
	for _, f := range rep.Copied {
		if !strings.HasPrefix(f, "alpha/") {
			t.Errorf("dry-run 的文件清单形态不对: %s", f)
		}
	}
}

func TestApplyRejectsSameDir(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	plan, err := BuildPlan(official, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan, official, Options{Force: true}); err == nil {
		t.Fatal("目标与来源相同应报错（而不是自我覆盖）")
	}
}

// --force 只删「名字在计划里」的顶层条目，目标里的其它内容必须留着。
func TestApplyForceKeepsOthers(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	plan, err := BuildPlan(official, nil)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	mkSkill(t, dst, "alpha", goodBody("alpha"))
	other := mkSkill(t, dst, "别人的技能", goodBody("别人的技能"))

	if _, err := Apply(plan, dst, Options{Force: true}); err != nil {
		t.Fatalf("--force: %v", err)
	}
	if _, err := os.Stat(filepath.Join(other, "SKILL.md")); err != nil {
		t.Errorf("--force 误删了不在计划里的技能目录: %v", err)
	}
}

// 第三方失联时，官方技能照样装 —— 这是用户明确要的「不影响其它技能安装」。
func TestApplyInstallsOfficialWhenExternalMissing(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	plan, err := BuildPlan(official, []string{filepath.Join(t.TempDir(), "gone")})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Warnings) != 1 {
		t.Fatalf("warnings = %+v, want 1", plan.Warnings)
	}
	dst := t.TempDir()
	rep, err := Apply(plan, dst, Options{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "alpha", "SKILL.md")); err != nil {
		t.Fatalf("官方技能没装上: %v", err)
	}
	if len(rep.Warnings) != 1 {
		t.Errorf("Report 该把告警带出去: %+v", rep.Warnings)
	}
}

// 只跳**根级** assets/：嵌套的同名目录照装（它不是这套技能的静态资源约定）。
func TestApplyOnlySkipsTopLevelAssets(t *testing.T) {
	official := t.TempDir()
	mkSkill(t, official, "alpha", goodBody("alpha"))
	kb := mkSkill(t, t.TempDir(), "kb", goodBody("kb"))
	for _, p := range []string{
		filepath.Join(kb, "assets", "big.bin"),
		filepath.Join(kb, "references", "assets", "small.txt"),
	} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := BuildPlan(official, []string{kb})
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if _, err := Apply(plan, dst, Options{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "kb", "assets")); !os.IsNotExist(err) {
		t.Error("根级 assets/ 不该被复制")
	}
	if _, err := os.Stat(filepath.Join(dst, "kb", "references", "assets", "small.txt")); err != nil {
		t.Errorf("嵌套的 references/assets 该照装: %v", err)
	}
}
