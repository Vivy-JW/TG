package store

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// 本文件只用 git **命令行**（os/exec），不引任何 git 库：git 不是依赖，
// 只是工作区的版本留痕（export 时 init + 首次 commit，apply 成功后再 commit）。

// errGitMissing 表示环境里没有 git 可执行文件。
//
// Create 对它是宽容的（没有 git 也能产出完整工作区，只是不建版本库）；
// 显式调用 GitInit / GitCommit / GitHead 时会作为 *IOError 冒泡。
var errGitMissing = errors.New("git 可执行文件不在 PATH 中")

// gitError 保留 git 的退出码与 stderr，便于把 stderr 原样带进返回的错误。
type gitError struct {
	args   []string
	stderr string
	code   int
	err    error
}

func (e *gitError) Error() string {
	cmd := "git " + strings.Join(e.args, " ")
	detail := strings.TrimSpace(e.stderr)
	if detail == "" && e.err != nil {
		detail = e.err.Error()
	}
	if detail == "" {
		return fmt.Sprintf("%s 失败（退出码 %d）", cmd, e.code)
	}
	return fmt.Sprintf("%s 失败（退出码 %d）：%s", cmd, e.code, detail)
}

func (e *gitError) Unwrap() error { return e.err }

// gitRun 执行 `git -C <dir> -c core.autocrlf=false <args...>`。
//
// 显式关掉 autocrlf：让 git 里的字节 == 工作区里的字节（快照保真），
// 也避免 Windows 上出现「内容没改但 status 说改了」的假脏。
func gitRun(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir, "-c", "core.autocrlf=false"}, args...)
	cmd := exec.Command("git", full...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err == nil {
		return out.String(), nil
	}
	if errors.Is(err, exec.ErrNotFound) {
		return out.String(), fmt.Errorf("%w（%v）", errGitMissing, err)
	}
	ge := &gitError{args: full, stderr: errb.String(), code: -1, err: err}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		ge.code = ee.ExitCode()
	}
	return out.String(), ge
}

// GitInit 在工作区建版本库（`git -C <dir> init`）。
func (w *Workspace) GitInit() error {
	if _, err := gitRun(w.Dir, "init"); err != nil {
		return ioErr("git init 失败（工作区 "+w.Dir+"）", err)
	}
	return nil
}

// GitCommit 暂存全部改动并提交，返回 commit hash。
//
// 提交时用 `-c user.name=tdev -c user.email=tdev@local` 提供身份，**不依赖全局 git 配置**；
// 额外带上 `-c commit.gpgsign=false`，避免用户全局开了签名导致提交失败。
// 干净树（无任何改动）不是错误：直接返回当前 HEAD。
func (w *Workspace) GitCommit(message string) (string, error) {
	msg := strings.TrimSpace(message)
	if msg == "" {
		msg = "tdev"
	}
	if _, err := gitRun(w.Dir, "add", "-A"); err != nil {
		return "", ioErr("git add -A 失败（工作区 "+w.Dir+"）", err)
	}
	dirty, err := w.gitDirty()
	if err != nil {
		return "", err
	}
	if !dirty {
		// 干净树：返回当前 HEAD；尚无首个 commit（unborn HEAD）时返回空 hash + nil。
		head, herr := w.GitHead()
		if herr != nil {
			return "", nil
		}
		return head, nil
	}
	_, cerr := gitRun(w.Dir,
		"-c", "user.name=tdev",
		"-c", "user.email=tdev@local",
		"-c", "commit.gpgsign=false",
		"commit", "-m", msg)
	if cerr != nil {
		// 兜底：并发场景下别人刚提交完，树又干净了 —— 这不算错误。
		if d2, derr := w.gitDirty(); derr == nil && !d2 {
			if head, herr := w.GitHead(); herr == nil {
				return head, nil
			}
			return "", nil
		}
		return "", ioErr("git commit 失败（工作区 "+w.Dir+"）", cerr)
	}
	return w.GitHead()
}

// GitHead 返回当前 HEAD 的 commit hash（尚未 commit 时是 *IOError）。
func (w *Workspace) GitHead() (string, error) {
	out, err := gitRun(w.Dir, "rev-parse", "HEAD")
	if err != nil {
		return "", ioErr("git rev-parse HEAD 失败（工作区可能尚未 init 或尚无 commit）", err)
	}
	return strings.TrimSpace(out), nil
}

// gitDirty 判断工作区是否相对 HEAD 有改动（含未跟踪文件）。
func (w *Workspace) gitDirty() (bool, error) {
	out, err := gitRun(w.Dir, "status", "--porcelain")
	if err != nil {
		return false, ioErr("git status --porcelain 失败（工作区 "+w.Dir+"）", err)
	}
	return strings.TrimSpace(out) != "", nil
}
