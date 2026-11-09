package tzc

import (
	"path/filepath"
	"strings"
	"testing"
	"tt/internal/testkit"

	"tt/internal/dev/testutil"
)

// exportWS 在 dir 下导出一个临时工作区并返回其路径（供 cd 相关的测试用）。
func exportWS(t *testing.T, dir, prog string) string {
	t.Helper()
	pkg, err := testutil.NormalPkgPath(dir, prog)
	if err != nil {
		t.Fatal(err)
	}
	if code := testkit.Silent(t, func() int { return cmdExport([]string{pkg}) }); code != 0 {
		t.Fatalf("export %s 退出码 %d", prog, code)
	}
	ws := filepath.Join(dir, prog+"-ws")
	if !IsWorkspaceDir(ws) {
		t.Fatalf("默认工作区没建出来: %s", ws)
	}
	return ws
}

// TestResolveWorkspaceDirDefaultsToCwd 钉住「cd 进工作区后所有动词都能省略 <dir>」。
//
// 三层语义：
//  1. 显式给了 <dir> → 用给的（转绝对路径）；
//  2. 没给 + 当前目录不是工作区 → 明确报错（退出码 5），绝不瞎猜别处的目录；
//  3. 没给 + 当前目录就是工作区 → 用当前目录，行为与显式给出完全一致。
func TestResolveWorkspaceDirDefaultsToCwd(t *testing.T) {
	dir := t.TempDir()
	ws := exportWS(t, dir, "adzi999")

	// ① 显式 <dir>
	got, err := resolveWorkspaceDir(ws)
	if err != nil {
		t.Fatalf("显式 <dir> 不该报错: %v", err)
	}
	if got != ws {
		t.Errorf("resolveWorkspaceDir(显式) = %q，想要 %q", got, ws)
	}

	// ② 非工作区目录 + 省略 <dir>
	t.Chdir(dir)
	if _, err := resolveWorkspaceDir(""); err == nil {
		t.Fatal("当前目录不是工作区时，省略 <dir> 必须报错")
	} else if !strings.Contains(err.Error(), "工作区") {
		t.Errorf("错误信息应指出「工作区」: %v", err)
	}
	if code := testkit.Silent(t, func() int { return cmdStatus(nil) }); code != 5 {
		t.Errorf("非工作区省略 <dir> 的 status 退出码 %d，期望 5（IO·环境失败）", code)
	}

	// ③ 工作区内 + 省略 <dir>
	t.Chdir(ws)
	if got, err = resolveWorkspaceDir(""); err != nil {
		t.Fatalf("工作区内省略 <dir> 不该报错: %v", err)
	} else if got != ws {
		t.Errorf("resolveWorkspaceDir(\"\") = %q，想要 %q", got, ws)
	}

	if code := testkit.Silent(t, func() int { return cmdStatus(nil) }); code != 0 {
		t.Fatalf("工作区内 status（省略 <dir>）退出码 %d", code)
	}
	if code := testkit.Silent(t, func() int { return cmdVerify(nil) }); code != 0 {
		t.Fatalf("工作区内 verify（省略 <dir>）退出码 %d", code)
	}
	// rename：省略 <dir> 时收 2 个位置参数（旧名、新名）
	if code := testkit.Silent(t, func() int { return cmdRename([]string{"adzi999_calc", "adzi999_count"}) }); code != 0 {
		t.Fatalf("工作区内 rename（省略 <dir>）退出码 %d", code)
	}
	// rename：显式 <dir> 的 3 参数形态不受影响
	if code := testkit.Silent(t, func() int { return cmdRename([]string{ws, "adzi999_count", "adzi999_total"}) }); code != 0 {
		t.Fatalf("rename（显式 <dir>）退出码 %d", code)
	}
	// rename：3 个位置参数但第一个不是工作区 → 明确报错，别把函数名当目录
	if code := testkit.Silent(t, func() int { return cmdRename([]string{"adzi999_total", "adzi999_x", "adzi999_y"}) }); code != 5 {
		t.Errorf("rename 第一个参数不是工作区目录时退出码 %d，期望 5", code)
	}
	// newfn
	if code := testkit.Silent(t, func() int {
		return cmdNewfn([]string{"--type", "FUNCTION", "--name", "adzi999_extra"})
	}); code != 0 {
		t.Fatalf("工作区内 newfn（省略 <dir>）退出码 %d", code)
	}
	// apply：--dry-run 不落盘，真落盘走一次
	if code := testkit.Silent(t, func() int { return cmdApply([]string{"--dry-run"}) }); code != 0 {
		t.Fatalf("工作区内 apply --dry-run（省略 <dir>）退出码 %d", code)
	}
	if code := testkit.Silent(t, func() int { return cmdApply([]string{"--yes"}) }); code != 0 {
		t.Fatalf("工作区内 apply（省略 <dir>）退出码 %d", code)
	}
	// 落盘之后仍然只靠当前目录就能继续工作
	if code := testkit.Silent(t, func() int { return cmdStatus(nil) }); code != 0 {
		t.Fatalf("apply 之后 status（省略 <dir>）退出码 %d", code)
	}
	if code := testkit.Silent(t, func() int { return cmdVerify(nil) }); code != 0 {
		t.Fatalf("apply 之后 verify（省略 <dir>）退出码 %d", code)
	}
}

// TestUnlockResolvesDirFromCwd unlock 省略 <dir> 与显式给出必须得到同一个判定
// （这里用两个各自独立的工作区各跑一次，比较退出码）。
func TestUnlockResolvesDirFromCwd(t *testing.T) {
	a := t.TempDir()
	wsA := exportWS(t, a, "adzi997")
	t.Chdir(wsA)
	cwdCode := testkit.Silent(t, func() int { return cmdUnlock([]string{"--yes"}) })

	b := t.TempDir()
	wsB := exportWS(t, b, "adzi996")
	expCode := testkit.Silent(t, func() int { return cmdUnlock([]string{wsB, "--yes"}) })

	if cwdCode != expCode {
		t.Errorf("unlock 省略 <dir> 退出码 %d ≠ 显式 <dir> 退出码 %d", cwdCode, expCode)
	}
	if cwdCode == 2 {
		t.Errorf("unlock 省略 <dir> 不该落到用法错误（退出码 2）")
	}
}
