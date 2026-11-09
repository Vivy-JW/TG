package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/update"
)

// Version 由打包脚本注入：build_portable.bat / build_msi.bat 读仓库根的 VERSION
// 文件，经 -ldflags "-X tt/internal/cli.Version=…" 传进来（VERSION 是版本号唯一的
// 一处，web 的 package.json 由 `make version-check` 与它对齐）。
// 本地 go build 时为空，此时用 Go 构建信息里的 VCS 修订回答"这份二进制对应哪次提交"。
var Version string

// versionString 组装版本描述：优先注入的版本号，再附 commit / 提交日期 / 是否带未提交改动。
// 本地 go build 也会带上这些 VCS 信息（见 runtime/debug.ReadBuildInfo），所以总能回答
// "手上这份二进制对应哪次提交、工作区干不干净"——这正是排查"文档与二进制是否同版"要的。
func versionString() string {
	rev, when, dirty := vcsInfo()
	tag := shortRev(rev)
	if tag != "" && dirty {
		tag += "+未提交改动"
	}
	switch {
	case Version != "" && tag != "":
		return fmt.Sprintf("%s (commit %s, %s)", Version, tag, when)
	case Version != "":
		return Version
	case tag != "":
		return fmt.Sprintf("devel (commit %s, %s)", tag, when)
	default:
		return "devel (无版本信息；用 build_portable.bat 打包会注入版本号)"
	}
}

// vcsInfo 从构建信息里取 VCS 修订号 / 提交日期 / 是否有未提交改动。
func vcsInfo() (rev, when string, dirty bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			if len(s.Value) >= 10 {
				when = s.Value[:10]
			} else {
				when = s.Value
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	return rev, when, dirty
}

// shortRev 取修订号前 7 位（与 git 的短 hash 一致）。
func shortRev(rev string) string {
	if len(rev) > 7 {
		return rev[:7]
	}
	return rev
}

// exitCodeOf 取出带退出码错误的退出码。
//
// 合并前只有 TDev 有明确的退出码契约（0 成功 / 2 包格式或用法错 / 3 校验失败 /
// 4 拒绝写入 / 5 IO 与环境失败 / 1 未分类），它在这条链路上保留下来：
// internal/cli/dev 把 tdev 的退出码包成 exitCoder，由这里解出来。
func exitCodeOf(err error) int {
	var ec exitCoder
	if errors.As(err, &ec) {
		return ec.ExitCode()
	}
	return 0
}

// exitCoder 带退出码的错误。
type exitCoder interface{ ExitCode() int }

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Long: `显示版本信息：注入的版本号 + 这次构建对应的提交。

完全离线：它只读本地的构建信息与 <数据目录>/update/check.json 里上一次检查的
结果（` + "`tt update check`" + ` 写的），不联网。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			hints := []string{}
			for _, msg := range []string{updateHint(), skillsHint()} {
				if msg != "" {
					hints = append(hints, msg)
				}
			}
			if common.IsJSON() {
				return common.PrintJSON(map[string]any{"ok": true, "version": versionString(), "hints": hints})
			}
			cmd.Println("tt " + versionString())
			for _, msg := range hints {
				cmd.Println(msg)
			}
			return nil
		},
	}
}

// skillsHint 技能树与二进制不同版时提一句。
//
// 升级成功后更新器自己会把技能刷到同版，所以这条通常只在"手动换了二进制"（拷了一份
// 新的 tt.exe、或从别的版本解压）时出现 —— 而那正是 agent 会默默读到旧手册的场景。
func skillsHint() string {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return ""
	}
	return update.SkillsDriftHint(filepath.Dir(cfgPath), Version)
}

// updateHint 读检查缓存给一句话提示。
//
// “读不到就不提示”是刻意的：tt version 是排查时最先敲的命令，它不能因为缓存坏了、
// 配置没了、或者检查结果过期（见 update.LoadCachedCheck 的两个作废条件）而失败。
func updateHint() string {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return ""
	}
	res := update.LoadCachedCheck(filepath.Dir(cfgPath), Version, time.Now())
	return res.Message()
}
