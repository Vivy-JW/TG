// `tt install` —— 把 tt 装进使用者的环境。
//
//	tt install skills [--to <dir>|auto] [--agent <名字>] [--force] [--dry-run] [--strict]
//	                                           把官方技能 + 登记的第三方技能复制到目标目录
//	tt install path   [--dry-run]              把 exe 所在目录追加到**用户** PATH(HKCU，不需要管理员)
//
// 合并前 TDebug / TDev / TDictCli 各有一份同形实现，注释里写着"命令形态与另外两个一致
// （改动请三边同步）"。合并后只有这一份。
//
// 设计取舍：
//   - skills **不进二进制**（不用 go:embed）：技能内容是文档，随包发布、可被人直接改；
//     每个技能是一个带 SKILL.md 的目录。
//   - **两套来源彻底分开**：官方技能只来自 `<exe 目录>\skills`（发版包的一部分，走专用
//     路径）；第三方只来自用户在设置页登记的路径（config.json 的 skills.externals）。
//     装着陆后的合并、校验、冲突判定都在 internal/skill —— 设置页用的是同一份实现。
//   - PATH 只动 HKCU\Environment，绝不碰 HKLM/系统 PATH；实现在 internal/pathinstall，
//     与 Web 设置页的「加入 PATH」共用同一份。
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/pathinstall"
	"tt/internal/skill"
	"tt/internal/update"
)

const skillsDirName = "skills"

// skillsSource 解析「exe 旁边的 skills/」；测试可替换。
var skillsSource = defaultSkillsSource

func defaultSkillsSource() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("取不到当前可执行文件路径: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), skillsDirName), nil
}

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "把 tt 装进你的环境（skills / path）",
		Long: `把 tt 装进你的环境：

  tt install skills [--to <dir>|auto] [--agent <名字>] [--force] [--dry-run] [--strict]
                                             把官方技能 + 登记的第三方技能复制到目标目录
  tt install path   [--dry-run]              把 exe 所在目录追加到用户 PATH(HKCU)`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("请指定子命令: skills 或 path\n\n%s", cmd.UsageString())
		},
	}
	cmd.AddCommand(newInstallSkillsCmd(), newInstallPathCmd())
	return cmd
}

//---------------------------------------------------------------------------
// tt install skills
//---------------------------------------------------------------------------

func newInstallSkillsCmd() *cobra.Command {
	var (
		to     string
		agent  string
		force  bool
		dryRun bool
		strict bool
	)
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "把官方技能与登记的第三方技能合并到目标目录（默认 .agents/skills）",
		Long: `把技能合并到目标目录，默认是 <当前目录>/.agents/skills。

技能有两个来源：

  官方    与 tt 可执行文件同目录的 skills/。随包发布、走专用路径，不受本机配置影响。
  第三方  设置页「技能」里登记的路径（config.json 的 skills.externals），一行一个技能
          目录，位置任选。装它们之前会逐个校验 SKILL.md 的 frontmatter。

第三方里出问题的（路径找不到、没有 SKILL.md、name 与目录名不符、与别的技能同名）
只**告警并跳过**，官方技能与其它第三方照装 —— 用户勾在配置里的路径可能指向一块今天
没插的移动硬盘，那不该让整次安装失败。官方技能不合规则是硬错误：那是发版包坏了。

同一批技能里名字撞车时，官方优先；两条第三方同名，先登记的优先（顺序写在配置里）。

第三方技能根级的 assets/ **不复制**：目标里的 SKILL.md 正文首行会被写一行指针，指回
源目录的 assets（重复安装会替换那一行，不会堆）。这样知识库型技能不会把目标目录撑大。

要装到 agent 会去读的位置，用 --to auto 或点名一个 agent：

  tt install skills --to auto             # 已有 agent 目录就用它，否则建 .agents/skills
  tt install skills --agent claude-code   # 装到 Claude Code 读的位置
  tt install skills --agent agents        # 跨客户端公约数 .agents/skills

--agent 接受的名字：` + agentSkillNames() + `

--to auto 的探测顺序：.agents/skills 优先（读它的 agent 最多），其次 .claude/skills、
.github/skills、.cursor/skills；当前目录一个都没有就新建 .agents/skills。

目标目录不存在就新建；已经存在就**追加**：同名的条目默认跳过（不覆盖目标里已有的技能，
方便多个来源共用 .agents/skills），不冲突的技能照常加进去。要刷新同名条目（例如升级后
把技能刷成与 tt 同版）再加 --force（只删目标里名字来自来源树 or 登记表的那些条目，
不碰其它内容）。

--dry-run 只打印计划与告警，一个字节都不写。--strict 时有告警就返回非零退出码（给 CI 用）。`,
		Example: `  tt install skills --to auto
  tt install skills --agent claude-code
  tt install skills --to .agents/skills
  tt install skills --force
  tt install skills --dry-run
  tt install skills --to auto --strict`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := skillsSource()
			if err != nil {
				return err
			}
			if !isDir(src) {
				return fmt.Errorf("找不到 skills 目录: %s\n\nskills/ 与 tt.exe 同目录（免安装包里自带）；"+
					"从源码运行时先执行 make build（产物 tt.exe 与 skills/ 同目录）", src)
			}
			dst, why, terr := resolveSkillsTarget(to, agent)
			if terr != nil {
				return terr
			}
			externals, err := externalSkillPaths()
			if err != nil {
				return err
			}
			plan, err := skill.BuildPlan(src, externals)
			if err != nil {
				return err
			}
			rep, err := skill.Apply(plan, dst, skill.Options{Force: force, DryRun: dryRun})
			if err != nil {
				return err
			}

			// 记下"技能装到了哪、装它们的是哪个版本"。升级后靠它把 agent 目录里那份刷成
			// 与二进制同版 —— 不记的话它会永远停在装它的那个版本（见 update.SkillsStamp）。
			//
			// 没有版本号的构建（`make build` / `go test`）**不写**：戳的全部意义是"哪个
			// 版本装的"，记一个空版本只会让漂移提示说不清话，还会让测试在开发者的真实
			// 数据目录里留垃圾（它落的是 ResolveConfig 解析出的那个目录）。
			// dry-run 也不写：它承诺了一个字节都不动。
			if !dryRun && Version != "" {
				if cfgPath, cerr := common.ResolveConfig(true); cerr == nil {
					_ = update.RecordSkillsTarget(filepath.Dir(cfgPath), rep.Target, Version)
				}
			}

			printInstallReport(rep, src, why)
			if strict && len(rep.Warnings) > 0 {
				// 告警已经打出去了，这里只需要一个非零码收场（AlreadyReported 让根命令
				// 不再补一份错误信封，否则 stdout 上会出现两个 JSON 对象）。
				return exitStatus(1)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "目标目录；auto = 自动挑一个 agent 会读的位置（默认 <当前目录>/.agents/skills）")
	cmd.Flags().StringVar(&agent, "agent", "", "按 agent 名落到它读的位置："+agentSkillNames())
	cmd.Flags().BoolVar(&force, "force", false, "刷新已存在的同名条目（默认跳过）")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只打印计划与告警，不写盘")
	cmd.Flags().BoolVar(&strict, "strict", false, "有告警时返回非零退出码（CI 用）")
	return cmd
}

// externalSkillPaths 读配置里登记的第三方技能路径（按登记顺序）。
//
// 读不到配置不是错误：便携包第一次运行、或者压根不用第三方技能时，官方技能照样要能装。
// 但配置**存在却解析不了**要如实报错 —— 那种状态下"第三方装了什么"无从回答，静默跳过
// 会让人以为登记表是空的。
func externalSkillPaths() ([]string, error) {
	cfgPath, err := common.ResolveConfig(true)
	if err != nil {
		return nil, nil // 连落点都解析不出 = 没有配置 = 没有第三方技能
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("读不到第三方技能登记（配置文件解析失败）: %w", err)
	}
	return root.Skills.Paths(), nil
}

// printInstallReport 打安装结果。
//
// 告警走 stderr、放在最后：夹在一长串文件列表中间会被刷走，而"哪个技能没装上"正是
// 用户唯一需要动手的事。
func printInstallReport(rep skill.Report, src, why string) {
	if common.IsJSON() {
		warns := make([]map[string]any, 0, len(rep.Warnings))
		for _, w := range rep.Warnings {
			warns = append(warns, map[string]any{
				"kind": w.Kind, "index": w.Index, "path": w.Path,
				"name": w.Name, "why": w.Why, "text": w.Text(),
			})
		}
		_ = common.PrintJSON(map[string]any{
			"ok": true, "source": src, "target": rep.Target, "why": why,
			"dryRun": rep.DryRun, "files": rep.Copied, "skipped": rep.Skipped,
			"rewritten": rep.Rewritten, "warnings": warns,
		})
		return
	}

	if rep.DryRun {
		fmt.Println("(--dry-run，未写入任何文件)")
	}
	if len(rep.Copied) == 0 {
		fmt.Println("没有复制任何文件（目标里已经有全部同名条目）")
	} else {
		fmt.Printf("技能安装完成: %d 个文件\n", len(rep.Copied))
	}
	fmt.Printf("  来源: %s\n  目标: %s\n", src, rep.Target)
	if why != "" {
		fmt.Printf("  为什么是它: %s\n", why)
	}
	for _, f := range rep.Copied {
		fmt.Printf("    %s\n", f)
	}
	if len(rep.Rewritten) > 0 {
		fmt.Printf("  已改写 %d 份 SKILL.md 的静态文件指针（assets/ 没复制）: %s\n",
			len(rep.Rewritten), strings.Join(rep.Rewritten, ", "))
	}
	if len(rep.Skipped) > 0 {
		fmt.Printf("  跳过 %d 个已存在的条目（不覆盖；要刷新加 --force）: %s\n",
			len(rep.Skipped), strings.Join(rep.Skipped, ", "))
	}
	if len(rep.Warnings) > 0 {
		fmt.Fprintf(os.Stderr, "告警 %d 条（这些技能这次没装，其余照装）:\n", len(rep.Warnings))
		for _, w := range rep.Warnings {
			fmt.Fprintf(os.Stderr, "  %s\n", w.Text())
		}
	}
}

//---------------------------------------------------------------------------
// 目标目录：--to / --agent / auto
//---------------------------------------------------------------------------

// autoSkillsTarget 是 --to 的特殊值：不是目录，而是「替我挑一个 agent 会读的位置」。
const autoSkillsTarget = "auto"

// agentSkillsTarget 是一个「agent 读技能的位置」。
type agentSkillsTarget struct {
	Name    string   // --agent 接受的名字
	Root    string   // 该 agent 在项目里的目录（= 检测用的路标，也是 skills/ 的父目录）
	Aliases []string // 顺手接受的名字
	Who     string   // 装完谁会读它（写进输出，让用户知道这个选择意味着什么）
}

// Dir 是目标目录：<agent 项目目录>/skills。
func (a agentSkillsTarget) Dir() string { return filepath.Join(a.Root, skillsDirName) }

// agentSkillsDirs 是「agent 各自读技能的位置」—— --agent 与 --to auto 的唯一依据。
//
// **表里的顺序就是 --to auto 的探测顺序**：.agents/skills 排第一，因为它是跨客户端
// 公约数（Agent Skills 规范点名的互操作路径，pi、Codex、Cursor、Gemini CLI、OpenCode、
// Copilot 都扫它），装一次伺候的人最多。
//
// 只收路径真的不同的那几个 —— 路径相同的（Codex、Cline、Zed…）没有单列的必要。
var agentSkillsDirs = []agentSkillsTarget{
	{Name: "agents", Root: ".agents", Aliases: []string{"universal"},
		Who: "跨客户端公约数：pi、Codex、Cursor、Gemini CLI、OpenCode、Copilot 都读它"},
	{Name: "claude-code", Root: ".claude", Aliases: []string{"claude"},
		Who: "Claude Code"},
	{Name: "copilot", Root: ".github", Aliases: []string{"github-copilot", "vscode"},
		Who: "GitHub Copilot / VS Code 的工作区技能"},
	{Name: "cursor", Root: ".cursor",
		Who: "Cursor 的专属目录（它也读 .agents/skills）"},
}

// agentSkillNames 是 --agent 的全部合法名字（含别名），用于帮助与报错。
func agentSkillNames() string {
	var out []string
	for _, a := range agentSkillsDirs {
		out = append(out, a.Name)
		out = append(out, a.Aliases...)
	}
	return strings.Join(out, " / ")
}

func findAgentSkillsDir(name string) (agentSkillsTarget, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	for _, a := range agentSkillsDirs {
		if a.Name == n {
			return a, nil
		}
		for _, al := range a.Aliases {
			if al == n {
				return a, nil
			}
		}
	}
	return agentSkillsTarget{}, fmt.Errorf("不认识的 agent: %s\n可选: %s（或直接用 --to <目录> 给路径）",
		name, agentSkillNames())
}

// resolveSkillsTarget 把 --to / --agent 解析成目标目录，并给出「为什么是它」。
//
// why 只在目标是自动挑出来的（--to auto / --agent）时非空：显式给了路径的人
// 不需要别人告诉他为什么要装这儿。
func resolveSkillsTarget(to, agent string) (dst, why string, err error) {
	if to != "" && agent != "" {
		return "", "", fmt.Errorf("--to 与 --agent 只能给一个：--to 直接指目录，--agent 按名字查目录")
	}
	if agent != "" {
		a, aerr := findAgentSkillsDir(agent)
		if aerr != nil {
			return "", "", aerr
		}
		cwd, cerr := os.Getwd()
		if cerr != nil {
			return "", "", fmt.Errorf("取不到当前目录: %w", cerr)
		}
		return filepath.Join(cwd, a.Dir()), a.Who + " 会读它", nil
	}
	if to != "" && to != autoSkillsTarget {
		return to, "", nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", fmt.Errorf("取不到当前目录: %w", err)
	}
	if to == autoSkillsTarget {
		return resolveAutoTarget(cwd)
	}
	return filepath.Join(cwd, ".agents", skillsDirName), "", nil
}

// resolveAutoTarget 挑一个落点：当前目录已经为某个 agent 初始化过就别另起一个，
// 否则建跨客户端公约数 .agents/skills。
//
// 探的是 agent 的项目目录（.claude/ 之类）而不是 skills/ 本身：那才是「这人用这个 agent」
// 的路标，而且第一次装的时候 skills/ 还不存在。
func resolveAutoTarget(cwd string) (dst, why string, err error) {
	for _, a := range agentSkillsDirs {
		if isDir(filepath.Join(cwd, a.Root)) {
			return filepath.Join(cwd, a.Dir()), "当前目录已有 " + a.Dir() + "（" + a.Who + "）", nil
		}
	}
	a, err := findAgentSkillsDir("agents")
	if err != nil {
		return "", "", err
	}
	return filepath.Join(cwd, a.Dir()), "当前目录还没有 agent 目录，新建 " + a.Dir() + "（" + a.Who + "）", nil
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

//---------------------------------------------------------------------------
// tt install path
//---------------------------------------------------------------------------

func newInstallPathCmd() *cobra.Command {
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "path",
		Short: "把 exe 所在目录加入用户 PATH（HKCU，不需要管理员）",
		Long: `把 tt.exe 所在目录追加到**用户** PATH（HKCU\Environment\Path）。

只动当前用户的注册表项，不需要管理员，绝不碰系统 PATH。已有该项时幂等
（不重复追加、不重排顺序，原有的 %USERPROFILE% 之类可展开变量原样保留）。
与 Web 设置页「设置 → 加入 PATH」是同一份实现。`,
		Example: `  tt install path
  tt install path --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, next, changed, err := pathinstall.Preview()
			if err != nil {
				return err
			}
			if !st.Supported {
				if common.IsJSON() {
					return common.PrintJSON(st)
				}
				fmt.Printf("%s\n手动命令: %s\n", st.Note, st.Manual)
				return nil
			}

			if dryRun {
				if common.IsJSON() {
					return common.PrintJSON(map[string]any{"ok": true, "dryRun": true, "exeDir": st.ExeDir,
						"changed": changed, "old": st.UserPath, "new": next})
				}
				fmt.Println("(--dry-run，未写注册表)")
				fmt.Printf("  exe 目录: %s\n", st.ExeDir)
				if changed {
					fmt.Printf("  现在(原值): %s\n", st.UserPath)
					fmt.Printf("  将要写入  : %s\n", next)
				} else {
					fmt.Println("  用户 PATH 里已经有该目录，无需改动")
				}
				return nil
			}

			if !changed {
				if common.IsJSON() {
					return common.PrintJSON(map[string]any{"ok": true, "changed": false, "exeDir": st.ExeDir, "userPath": st.UserPath})
				}
				fmt.Printf("用户 PATH 里已经有 %s，无需改动\n", st.ExeDir)
				return nil
			}
			if _, err := pathinstall.Add(); err != nil {
				return err
			}
			if common.IsJSON() {
				return common.PrintJSON(map[string]any{"ok": true, "changed": true, "exeDir": st.ExeDir, "userPath": next})
			}
			fmt.Printf("已把 %s 追加到用户 PATH（HKCU\\Environment\\Path，不需要管理员）\n", st.ExeDir)
			fmt.Printf("  新值: %s\n", next)
			fmt.Println("  新开的终端即可直接敲 `tt`（当前已开的终端需要重开）")
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只打印将要写入的内容，不碰注册表")
	return cmd
}
