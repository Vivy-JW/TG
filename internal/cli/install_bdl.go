package cli

// `tg install bdl` —— 把随包的 Genero BDL 官方手册（Markdown 版）解到本机。
//
// 为什么有这条命令：Agent 读 Genero 代码时要能查到官方手册原文（4GL 语法、fgldb、
// Genero 与数据库的边界…），而那份文档有 5470 个文件、16 MB，**不该塞进技能正文**：
// 技能是给人读的行为说明，手册是查得到的资料。两者分开放，Agent 才知道该查哪个。
//
// 三条取舍（与 skills 那条线刻意不同）：
//   - **整包解，不登记、不合并**：文档目录只有 tg 自己写，没有"用户整理过的同名条目"
//     要保护；--force 只覆盖同名文件、**不删**目标里多出来的文件（用户可能往里加了
//     自己的笔记，删了就是数据丢失）。
//   - **装到哪儿听配置的**：--to > config 的 bdldoc.dir > <数据目录>/BDL-Markdown。
//     与 dict/mirror 同一条规矩（配置住在哪，数据就住在哪），Agent 那侧靠
//     `tg install bdl --dry-run` 打印的 dir 就是手册原文的位置。
//   - **不回写配置**：--to 只对这一次生效。长期换位置是 `tg config set bdldoc.dir`
//     的事 —— 一条命令干两件事，事后谁也说不清目录为什么在那儿。

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"tg/internal/cli/common"
	"tg/internal/config"
	"tg/internal/output"
	"tg/internal/zipx"
)

// bdlZipName 随包分发的文档压缩包名（与 tg.exe 同目录）。
const bdlZipName = "BDL-Markdown.zip"

// bdlZipSource 解析随包的 zip 路径；测试可替换。
var bdlZipSource = defaultBDLZipSource

func defaultBDLZipSource() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("取不到当前可执行文件路径: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), bdlZipName), nil
}

func newInstallBDLCmd() *cobra.Command {
	var (
		to     string
		from   string
		force  bool
		dryRun bool
	)
	cmd := &cobra.Command{
		Use:   "bdl",
		Short: "解出随包的 Genero BDL 官方手册（Markdown 版）",
		Long: `把随包分发的 Genero BDL 官方手册（Markdown 版）解到本机。

文档来自与 tg.exe 同目录的 ` + bdlZipName + `（发版包的一部分，也可以用 --from 另指一份）。

装到哪儿（第一个非空者胜出）：

  --to <目录>              只对这一次生效，不改配置
  config 的 bdldoc.dir     长期换位置改这里：tg config set bdldoc.dir <目录>
  缺省                     <数据目录>/BDL-Markdown（便携形态在包内，普通形态在 %APPDATA%\tg）

Agent 那侧靠 ` + "`tg install bdl --dry-run`" + ` 打印的 dir 找到手册原文（不写盘、可重复执行）；
配过 bdldoc.dir 的话 ` + "`tg config get bdldoc.dir`" + ` 也能读到。

目标目录已经存在且非空时**默认跳过**（退出 0，什么也不写）：文档目录只有 tg 自己写，
重复执行不该反复解压 16 MB。要刷新（例如升级后换成新一版手册）加 --force。--force
只覆盖同名文件，**不删**目标里多出来的文件 —— 你可能往里放了自己的笔记。

--dry-run 只打印计划，一个字节都不写。`,
		Example: `  tg install bdl
  tg install bdl --to D:\Docs\BDL-Markdown
  tg install bdl --force
  tg install bdl --dry-run
  tg install bdl --from D:\下载\BDL-Markdown.zip`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			zipPath, err := resolveBDLZip(from)
			if err != nil {
				return err
			}
			if !fileExists(zipPath) {
				return (&output.Error{
					Code: output.CodeNotFound, Exit: 1,
					Message: fmt.Sprintf("找不到文档包: %s", zipPath),
					Hint: "发版包应当把它与 tg.exe 放一起；从源码运行时先执行 make package。" +
						"也可以直接指定一份：tg install bdl --from <zip>",
				})
			}
			dir, why, err := resolveBDLDir(to)
			if err != nil {
				return err
			}

			// 重复执行不该反复解 16 MB：目标非空就跳过（--force 才刷新）。
			if !force && !dryRun && hasEntries(dir) {
				printBDLReport(bdlReport{Zip: zipPath, Dir: dir, Why: why, Skipped: true})
				return nil
			}

			rep := bdlReport{Zip: zipPath, Dir: dir, Why: why, DryRun: dryRun}
			if dryRun {
				rep.Existed = hasEntries(dir)
				printBDLReport(rep)
				return nil
			}

			files, err := extractBDL(zipPath, dir)
			if err != nil {
				return err
			}
			rep.Files = files
			printBDLReport(rep)
			return nil
		},
	}
	cmd.Flags().StringVar(&to, "to", "", "目标目录；只对这一次生效（长期改位置用 tg config set bdldoc.dir）")
	cmd.Flags().StringVar(&from, "from", "", "文档包路径（默认 <exe 目录>/"+bdlZipName+"）")
	cmd.Flags().BoolVar(&force, "force", false, "目标已存在时也解（只覆盖同名文件，不删多余文件）")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "只打印计划，不写盘")
	return cmd
}

// resolveBDLZip 定文档包：--from 优先，否则 exe 旁边的随包 zip。
func resolveBDLZip(from string) (string, error) {
	if from != "" {
		return config.AbsPath(from), nil
	}
	return bdlZipSource()
}

// resolveBDLDir 定目标目录：--to > config 的 bdldoc.dir > <数据目录>/BDL-Markdown。
//
// why 只在目标是推导出来的（配置/缺省）时非空 —— 显式给了 --to 的人不需要解释。
func resolveBDLDir(to string) (dir, why string, err error) {
	if to != "" {
		return config.AbsPath(to), "", nil
	}
	cfgPath, cerr := common.ResolveConfig(true)
	if cerr != nil {
		// 连配置落点都解析不出：仍然给缺省目录（BDLDocDirFor 对空 cfgPath 也能算）。
		return config.BDLDocDirFor(""), "", nil
	}
	dir = config.BDLDocDirFor(cfgPath)
	if root, lerr := config.Load(cfgPath); lerr == nil && root.BDLDoc.Dir != "" {
		return dir, "config 的 bdldoc.dir", nil
	}
	return dir, "缺省落点（数据目录下的 " + config.BDLDocDirName + "）", nil
}

// extractBDL 解包落到 dir。
//
// 先解到**目标旁边的**临时目录再 rename 到位：解到一半失败不会在目标里留下半套文档，
// 也不能解到 %TEMP% —— 跨卷 rename 会退化成整棵树复制，白白多写 16 MB。
// 目标已存在（--force）时只能合并（覆盖同名、保留多余），那是慢路径。
func extractBDL(zipPath, dir string) (files int, err error) {
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return 0, fmt.Errorf("建目录 %s 失败: %w", parent, err)
	}
	tmp, err := os.MkdirTemp(parent, ".tg-bdl-")
	if err != nil {
		return 0, fmt.Errorf("建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmp)

	if err := zipx.ExtractZip(zipPath, tmp); err != nil {
		return 0, (&output.Error{
			Code: output.CodeLocalIO, Exit: 1,
			Message: fmt.Sprintf("解不开文档包: %v", err),
			Hint:    "这份 zip 可能下载/拷贝时坏了；重新拿一份发版包里的 " + bdlZipName,
		})
	}
	root, err := zipx.PayloadRoot(tmp)
	if err != nil {
		return 0, fmt.Errorf("文档包是空的（%s）: %w", zipPath, err)
	}

	// 常见路径：目标还不存在 → 整个目录 rename 到位（一次 fs 操作，不复制）。
	if _, serr := os.Stat(dir); os.IsNotExist(serr) {
		if rerr := os.Rename(root, dir); rerr == nil {
			n, cerr := zipx.CountFiles(dir)
			return n, cerr
		}
	}
	n, err := zipx.CopyTree(root, dir, nil)
	if err != nil {
		return n, (&output.Error{
			Code: output.CodeLocalIO, Exit: 1,
			Message: fmt.Sprintf("写到 %s 失败: %v", dir, err),
			Hint:    "换个有写权限的位置：tg install bdl --to <目录>",
		})
	}
	return n, nil
}

// hasEntries 目标目录是否已经装了东西（不存在或空目录都算"没装"）。
func hasEntries(dir string) bool {
	ents, err := os.ReadDir(dir)
	return err == nil && len(ents) > 0
}

// bdlReport 一次安装的结果（JSON 与人类可读输出共用）。
type bdlReport struct {
	Zip     string `json:"zip"`
	Dir     string `json:"dir"`
	Why     string `json:"why,omitempty"`
	Files   int    `json:"files"`
	Skipped bool   `json:"skipped"`
	DryRun  bool   `json:"dryRun"`
	Existed bool   `json:"existed"`
}

func printBDLReport(r bdlReport) {
	if common.IsJSON() {
		_ = common.PrintJSON(map[string]any{
			"ok": true, "zip": r.Zip, "dir": r.Dir, "why": r.Why,
			"files": r.Files, "skipped": r.Skipped, "dryRun": r.DryRun, "existed": r.Existed,
		})
		return
	}
	if r.DryRun {
		fmt.Println("(--dry-run，未写入任何文件)")
		fmt.Printf("  文档包: %s\n  目标  : %s\n", r.Zip, r.Dir)
		if r.Why != "" {
			fmt.Printf("  为什么是它: %s\n", r.Why)
		}
		if r.Existed {
			fmt.Println("  目标已存在且非空：真跑的话会跳过（要刷新加 --force）")
		}
		return
	}
	if r.Skipped {
		fmt.Printf("%s 已存在且非空，未覆盖（要刷新加 --force）\n", r.Dir)
		return
	}
	fmt.Printf("Genero BDL 手册安装完成: %d 个文件\n", r.Files)
	fmt.Printf("  文档包: %s\n  目标  : %s\n", r.Zip, r.Dir)
	if r.Why != "" {
		fmt.Printf("  为什么是它: %s\n", r.Why)
	}
	fmt.Println("  手册原文就在上面的目标目录里；下次想问在哪：tg install bdl --dry-run")
}
