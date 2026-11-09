package testenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoWithLocalConfig 造一个"仓库根"（有 go.mod）与一份 config.local.json，返回那个目录。
// 每个用例一份，互不影响；调用方用 t.Chdir 进去。
func repoWithLocalConfig(t *testing.T, c Config) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, LocalFileName), b, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// TestCorpusRootEnvWinsOverFile 优先级的第一条：环境变量压过文件。
func TestCorpusRootEnvWinsOverFile(t *testing.T) {
	fileCorpus, envCorpus := t.TempDir(), t.TempDir()
	t.Chdir(repoWithLocalConfig(t, Config{CorpusRoot: fileCorpus}))
	t.Setenv("TTZS_CORPUS", envCorpus)

	if got := CorpusRoot(); got != envCorpus {
		t.Fatalf("环境变量该压过 config.local.json：想 %q，得 %q", envCorpus, got)
	}
}

// TestCorpusRootAcceptsBothEnvVars 两个变量都认 ——
// 这正是从前四个包抄错的那件事：它们只认 TDEV_CORPUS，于是只设 TTZS_CORPUS 的机器
// 在那四个包上会静默跑零个包（"0 个包全部通过"）。
func TestCorpusRootAcceptsBothEnvVars(t *testing.T) {
	want := t.TempDir()
	t.Chdir(t.TempDir()) // 一个没有 go.mod 的目录：确保不读文件

	t.Setenv("TTZS_CORPUS", want)
	if got := CorpusRoot(); got != want {
		t.Errorf("只设 TTZS_CORPUS 也该认：想 %q，得 %q", want, got)
	}
	t.Setenv("TTZS_CORPUS", "")
	t.Setenv("TDEV_CORPUS", want)
	if got := CorpusRoot(); got != want {
		t.Errorf("只设 TDEV_CORPUS 也该认：想 %q，得 %q", want, got)
	}
}

// TestCorpusRootOverrideWinsWhenBothSet 两个都设时按声明顺序取第一个（TDEV_CORPUS 优先）。
func TestCorpusRootOverrideWinsWhenBothSet(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	t.Chdir(t.TempDir())
	t.Setenv("TDEV_CORPUS", first)
	t.Setenv("TTZS_CORPUS", second)

	if got := CorpusRoot(); got != first {
		t.Fatalf("两个都设时该取 TDEV_CORPUS（顺序在前）：想 %q，得 %q", first, got)
	}
}

// TestCorpusRootDoesNotFallBackWhenOverrideIsBroken 覆盖项设了却指不到 → 空，**不回落**。
//
// 这条是刻意反直觉的：静默换一份别的语料去跑，会留下"我以为跑的是这份"这种错误结论。
// 所以宁可返回空（调用方会跳过并打印原因），也不悄悄用缺省。
func TestCorpusRootDoesNotFallBackWhenOverrideIsBroken(t *testing.T) {
	fileCorpus := t.TempDir()
	t.Chdir(repoWithLocalConfig(t, Config{CorpusRoot: fileCorpus}))
	t.Setenv("TTZS_CORPUS", filepath.Join(t.TempDir(), "并不存在"))

	if got := CorpusRoot(); got != "" {
		t.Fatalf("指错了就该给空（不回落文件、也不回落缺省）：得 %q", got)
	}
}

// TestCorpusRootFileWhenNoEnv 没有环境变量时用文件里的。
func TestCorpusRootFileWhenNoEnv(t *testing.T) {
	fileCorpus := t.TempDir()
	t.Chdir(repoWithLocalConfig(t, Config{CorpusRoot: fileCorpus}))
	t.Setenv("TDEV_CORPUS", "")
	t.Setenv("TTZS_CORPUS", "")

	if got := CorpusRoot(); got != fileCorpus {
		t.Fatalf("没有环境变量时该用文件里的：想 %q，得 %q", fileCorpus, got)
	}
}

// TestOtherThreeFollowTheSameOrder 另外三项与语料根同一条规矩：环境变量 > 文件。
func TestOtherThreeFollowTheSameOrder(t *testing.T) {
	dir := repoWithLocalConfig(t, Config{
		EngineExe:   "exe-from-file",
		Workspace:   "ws-from-file",
		DesignerDir: "designer-from-file",
	})
	t.Chdir(dir)

	// 先只看文件
	t.Setenv("TTZS_EXE", "")
	t.Setenv("TTZS_WS", "")
	t.Setenv("TTZS_INSTALL", "")
	if got := EngineExe(); got != "exe-from-file" {
		t.Errorf("EngineExe 该取文件：得 %q", got)
	}
	if got := Workspace(); got != "ws-from-file" {
		t.Errorf("Workspace 该取文件：得 %q", got)
	}
	if got := DesignerDir(); got != "designer-from-file" {
		t.Errorf("DesignerDir 该取文件：得 %q", got)
	}

	// 再看环境变量压过文件
	t.Setenv("TTZS_EXE", "exe-from-env")
	t.Setenv("TTZS_WS", "ws-from-env")
	t.Setenv("TTZS_INSTALL", "designer-from-env")
	if got := EngineExe(); got != "exe-from-env" {
		t.Errorf("EngineExe 环境变量该压过文件：得 %q", got)
	}
	if got := Workspace(); got != "ws-from-env" {
		t.Errorf("Workspace 环境变量该压过文件：得 %q", got)
	}
	if got := DesignerDir(); got != "designer-from-env" {
		t.Errorf("DesignerDir 环境变量该压过文件：得 %q", got)
	}
}

// TestMissingOrBrokenConfigIsNotAnError 没有这份文件、或文件不是 JSON —— 都当空配置，不报错。
//
// 没有 config.local.json 是**正常状态**（它在 .gitignore 里，绝大多数机器上就没有）。
// 读不动就报错的话，`go test ./...` 会在别人的机器上红 —— 那正是这个仓库最反对的假失败。
func TestMissingOrBrokenConfigIsNotAnError(t *testing.T) {
	t.Run("没有文件", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)
		if got := Local(); got != (Config{}) {
			t.Fatalf("没有文件该得到空配置，得 %+v", got)
		}
		if p := LocalPath(); p != "" {
			t.Fatalf("没有文件时 LocalPath 该是空串，得 %q", p)
		}
	})
	t.Run("文件不是 JSON", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module fixture\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, LocalFileName), []byte("{ 这不是 json"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Chdir(dir)
		if got := Local(); got != (Config{}) {
			t.Fatalf("坏文件该当空配置，得 %+v", got)
		}
	})
}

// TestCorpusRootDetailNamesTheRightCulprit 诊断必须点**真正**卡住的那一处。
//
// 这条是给 doctor 的文案兜底的：自己拿"配置文件在不在"去猜"是哪一处没指到"一定会猜错，
// 而猜错的结果是给出的指引正好指向错的那条路 —— 比如明明是环境变量指错了，
// 却说"你没设环境变量"。所以逐个处境钉住。
func TestCorpusRootDetailNamesTheRightCulprit(t *testing.T) {
	t.Run("环境变量指不到（哪怕配置文件存在且可用）", func(t *testing.T) {
		fileCorpus := t.TempDir()
		t.Chdir(repoWithLocalConfig(t, Config{CorpusRoot: fileCorpus}))
		t.Setenv("TTZS_CORPUS", filepath.Join(t.TempDir(), "并不存在"))

		got := CorpusRootDetail()
		if !strings.Contains(got, "TTZS_CORPUS") || !strings.Contains(got, "指不到目录") {
			t.Fatalf("该点名 TTZS_CORPUS 指不到目录，得 %q", got)
		}
		if strings.Contains(got, "config.local.json") {
			t.Fatalf("配置文件是好的，不该拿它当原因，得 %q", got)
		}
	})
	t.Run("配置文件里的指不到（且没设环境变量）", func(t *testing.T) {
		t.Chdir(repoWithLocalConfig(t, Config{CorpusRoot: filepath.Join(t.TempDir(), "并不存在")}))
		t.Setenv("TDEV_CORPUS", "")
		t.Setenv("TTZS_CORPUS", "")

		got := CorpusRootDetail()
		if !strings.Contains(got, LocalFileName) || !strings.Contains(got, "指不到目录") {
			t.Fatalf("该点名文件里的指不到目录，得 %q", got)
		}
	})
	t.Run("配置文件在但没写 corpusRoot", func(t *testing.T) {
		t.Chdir(repoWithLocalConfig(t, Config{}))
		t.Setenv("TDEV_CORPUS", "")
		t.Setenv("TTZS_CORPUS", "")

		got := CorpusRootDetail()
		if !strings.Contains(got, "没写 corpusRoot") {
			t.Fatalf("该说「在，但没写 corpusRoot」，得 %q", got)
		}
	})
	t.Run("环境变量可用时点名是哪个变量", func(t *testing.T) {
		t.Chdir(t.TempDir())
		want := t.TempDir()
		t.Setenv("TDEV_CORPUS", want)

		got := CorpusRootDetail()
		if !strings.Contains(got, "TDEV_CORPUS="+want) {
			t.Fatalf("该点名 TDEV_CORPUS 与它的值，得 %q", got)
		}
	})
}

// TestLocalFileIsFoundFromASubdirectory 上溯定位：从包目录（子目录）也能找到仓库根的那份。
// 测试的工作目录是**包目录**，这条正是那种处境。
func TestLocalFileIsFoundFromASubdirectory(t *testing.T) {
	root := repoWithLocalConfig(t, Config{CorpusRoot: "x"})
	sub := filepath.Join(root, "internal", "somepkg")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)

	want := filepath.Join(root, LocalFileName)
	if got := LocalPath(); got != want {
		t.Fatalf("从子目录该上溯找到仓库根那份：想 %q，得 %q", want, got)
	}
}
