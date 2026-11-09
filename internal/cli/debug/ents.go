package debug

// tt debug ents —— 企业目录:当前环境有哪些企业编号(ENT),各用哪个数据库账号。
//
// 这是给 AI agent 的标准入口:它要回答"该查哪个 schema"必须先知道有哪些企业。
// 与 tt debug db 的分工:
//
//	db   —— 连接体检:真连一次库,验证某个账号连得上(慢、要联网、不落缓存)
//	ents —— 目录:有哪些企业、各是哪个账号(带落盘快照,不联网也能答,并如实标注新鲜度)
//
// 两者共用同一个 resolver(见 internal/debug/ents.go),所以清单永远一致。

import (
	"fmt"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/debug"
)

var (
	entsRefresh bool
	entsCached  bool
	entsOne     int
)

var debugEntsCmd = &cobra.Command{
	Use:   "ents",
	Short: "列出当前环境的企业编号(ENT)→ 数据库账号(schema);读配置,离线可答",
	Long: `列出当前环境的企业编号(ENT),以及每个企业在该库里该用哪个账号(schema)。

企业清单来自库里的 gzou_t 表(与 tt debug sql 解析账号用的是同一张表)。
注意语义:库是**环境级**的(一个环境挂一个库),"企业编号"只决定**库里的账号** ——
不存在"每个企业一个数据库"这回事。

结果会落一份快照到配置目录的 ents/ 下,10 分钟内再问就直接答;断网、没起 serve
也能答,此时用 stale 与 fetchedAt 如实标注"这是什么时候的数据"。

  tt debug ents                 列全部企业(命中快照则秒回)
  tt debug ents --ent 99        只看企业 99 的账号
  tt debug ents --refresh       跳过缓存与快照,强制现查(现查失败即失败)
  tt debug ents --cached        只用快照,完全不连网
  tt debug ents --env 示例测试区  问指定环境,而不是当前环境

agent 用法:先 --json 取 ents[].account,再用 tt db sql --at <环境>@<N> "..." 查数据。
查不到数据时**先核对 current 那一段**:企业编号不对会连到另一个 schema 拿到 0 行,
那看起来和"数据不存在"一模一样。`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := resolveConfigPath()
		if err != nil {
			return err
		}
		cfg, err := debug.LoadConfig(path)
		if err != nil {
			return err
		}
		cfg.ApplyDefaultEnv()
		// 第一次在 debug 组里接上全局 --env:agent 常常要问的不是当前环境,
		// 而是"正式区有多少企业"。
		if common.Env != "" {
			c2 := cfg.CloneEnv(common.Env)
			if c2 == nil {
				return fmt.Errorf("环境 %q 不存在(用 tt env list 查看)", common.Env)
			}
			cfg = c2
		}
		// 数据目录 = 配置所在目录:快照与 serve 侧落在同一份,谁先查到谁写
		cfg.DataDir = filepath.Dir(path)

		l, err := debug.LookupEnts(cfg, debug.EntListOpt{
			Ent: entsOne, Refresh: entsRefresh, Cached: entsCached,
		})
		if err != nil {
			return err
		}
		if IsJSON() {
			return printJSON(l)
		}
		printEntListing(cmd, l)
		return nil
	},
}

// printEntListing 人读输出:先交代"这份数据是什么、多新、这次问的是哪个企业",
// 再列清单 —— 顺序是刻意的,新鲜度和"当前企业"决定了清单该怎么读。
func printEntListing(cmd *cobra.Command, l *debug.EntListing) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "环境    %s\n", dashIfEmpty(l.Env))
	fmt.Fprintf(out, "数据库  %s %s\n", l.Dialect, dashIfEmpty(l.Target))
	fmt.Fprintf(out, "数据    %s\n", freshnessText(l))
	if l.Current != nil {
		fmt.Fprintf(out, "本次问的 %s\n", currentText(l.Current))
	}
	for _, n := range l.Notes {
		fmt.Fprintf(out, "注意    %s\n", n)
	}
	fmt.Fprintln(out)

	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "企业\t账号")
	for _, e := range l.Ents {
		acct := e.Account
		if e.Placeholder {
			acct = "-（该企业没配可用账号）"
		}
		fmt.Fprintf(w, "%d\t%s\n", e.Ent, acct)
	}
	if err := w.Flush(); err != nil {
		return
	}
	fmt.Fprintf(out, "\n共 %d 个企业\n", l.Count)
}

// freshnessText 把 Source/Stale/AgeSeconds 翻成一句话。agent 读 JSON,
// 人读这一行 —— 两边说的是同一件事,不能一边说"实时"一边偷偷用旧快照。
func freshnessText(l *debug.EntListing) string {
	what := map[string]string{
		"live":           "实时查询",
		"snapshot":       "快照",
		"snapshot-stale": "过期快照",
	}[l.Source]
	if what == "" {
		what = l.Source
	}
	s := fmt.Sprintf("%s · %s（%s前）", what, l.FetchedAt.Local().Format("15:04:05"), humanAge(l.AgeSeconds))
	if l.Stale {
		s += " ⚠ 已过期"
	}
	return s
}

func humanAge(sec int) string {
	switch {
	case sec < 60:
		return fmt.Sprintf("%d 秒", sec)
	case sec < 3600:
		return fmt.Sprintf("%d 分钟", sec/60)
	default:
		return fmt.Sprintf("%d 小时", sec/3600)
	}
}

func currentText(c *debug.EntCurrent) string {
	if c.Resolved {
		return fmt.Sprintf("企业 %d → 账号 %s（来自%s）", c.Ent, c.Account, entSourceLabel(c.Source))
	}
	raw := c.Raw
	if raw == "" {
		raw = "未定"
	}
	return fmt.Sprintf("%s ⚠ %s", raw, c.Reason)
}

func entSourceLabel(src string) string {
	switch src {
	case "flag":
		return " --ent"
	case "config":
		return "环境配置的 topent"
	default:
		return "配置"
	}
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func init() {
	debugEntsCmd.Flags().IntVar(&entsOne, "ent", 0, "只看这个企业编号(0=列出全部)")
	debugEntsCmd.Flags().BoolVar(&entsRefresh, "refresh", false, "跳过缓存与快照,强制现查 gzou_t")
	debugEntsCmd.Flags().BoolVar(&entsCached, "cached", false, "只用快照,完全不连网(没有快照则报错)")
	Group.AddCommand(debugEntsCmd)
}
