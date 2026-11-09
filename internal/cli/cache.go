package cli

// tt cache:配置目录下那些**可再生**中间数据的查看与清理。
//
// 这些目录与 config.json 同居一处,但性质相反 —— 一个是你的数据,删了就没了;
// 下面这些只是跑出来的副本与快照,删了会自愈。哪几个算缓存由
// internal/config/cache.go 定义一次,CLI / Web / 启动清理共用那一份。

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/output"
)

// cacheOlderThan --older-than:只清比这旧的(0 = 全清)。
var cacheOlderThan time.Duration

func newCacheCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "查看/清理缓存(企业快照、源码镜像、查询落盘、断点存档)",
		Long: `查看与清理配置目录下的缓存。

缓存 = 跑出来的、可再生的中间数据(它们与 config.json 同居一个目录,但删了会自愈):

  ents        企业目录快照(ENT→账号),10 分钟新鲜期,过期自动重查
  srccache    调试时的源码镜像,只活一轮调试
  execlog     调试执行的大输出落盘副本
  debug-bps   断点存档
  spill       查询被截断时落盘的完整结果

**config.json 不是缓存**,清除时一律不动;正在运行的服务状态文件也不动。

tt serve 启动时会自动清掉超过 7 天的缓存 —— 刻意不"启动即全清":spill 里那些文件
的用途正是"刚才那条查询的完整结果,想看全量去读它",启动就删会把上一条命令刚
告诉你的东西删掉。要立刻清空用 tt cache clear。`,
		Example: `  tt cache                      # 看各缓存目录占多少
  tt cache clear                # 全清
  tt cache clear --older-than 24h
  tt cache clear --dry-run      # 只算不删
  tt cache --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return err
			}
			st := config.CacheStatusOf(dir)
			if common.OutputFormat() != output.FormatTable {
				return output.Emit(cmd.OutOrStdout(), output.Options{
					Format: common.OutputFormat(), Meta: output.Meta{},
					Data: st,
				})
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "缓存目录: %s\n", st.Dir)
			for _, d := range st.Dirs {
				size := "—"
				if d.Exists {
					size = humanBytes(d.Bytes)
				}
				fmt.Fprintf(out, "  %-11s %10s  %d 个文件\n", d.Name, size, d.Files)
			}
			fmt.Fprintf(out, "\n共 %s / %d 个文件;tt serve 启动时清理超过 %d 小时的\n",
				humanBytes(st.Bytes), st.Files, st.MaxAgeHours)
			return nil
		},
	}
	cmd.AddCommand(newCacheClearCmd())
	return cmd
}

func newCacheClearCmd() *cobra.Command {
	var dryRun bool
	clear := &cobra.Command{
		Use:   "clear",
		Short: "清理缓存(默认全清;--older-than 只清旧的)",
		Long: `清理缓存目录。只删缓存,config.json 与运行中的服务状态文件一律不动。

--older-than 给一个时长(如 24h、7d 不接受,用 168h)只清比它旧的;
--dry-run 只统计会释放多少,不删。`,
		Example: `  tt cache clear
  tt cache clear --older-than 24h
  tt cache clear --dry-run`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cacheDir()
			if err != nil {
				return err
			}
			if dryRun {
				st := config.CacheStatusOf(dir)
				fmt.Fprintf(cmd.OutOrStdout(), "将释放 %s / %d 个文件(未删除)\n",
					humanBytes(st.Bytes), st.Files)
				return nil
			}
			removed, freed, err := config.CleanCache(dir, cacheOlderThan)
			if err != nil {
				return err
			}
			scope := "全部"
			if cacheOlderThan > 0 {
				scope = "超过 " + cacheOlderThan.String() + " 的"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "已清理%s缓存: %d 个文件,释放 %s\n",
				scope, removed, humanBytes(freed))
			return nil
		},
	}
	clear.Flags().DurationVar(&cacheOlderThan, "older-than", 0, "只清比这旧的 (如 24h;缺省 0 = 全清)")
	clear.Flags().BoolVar(&dryRun, "dry-run", false, "只统计会释放多少,不删")
	return clear
}

// cacheDir 数据目录 —— 缓存就是它下面的那几个子目录(落点规则在 config.Locations)。
func cacheDir() (string, error) {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return "", err
	}
	return config.LocationsAt(path).DataDir(), nil
}

// humanBytes 给人看的体积(与设置页同一套口径)。
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}
