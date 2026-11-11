package gp

// 顶层命令 tg site：把 GP 的营运据点（azp_file）列出来。
//
// 为什么挂在顶层、不并进 tg gp：`--at` 到处都要写据点编号
// （tg dict r.t ima_file --at 22主机@COF），据点本身是"这个环境里能选哪些库"，
// 不该藏在某个产品组下面；而且它跟 T100 的 ENT 是同一个位置的概念
// （tg env ents ↔ tg site list），摆在顶层才对称。
//
// 只读：连库查 azp_file，不碰配置、不写库。

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	gpcore "tt/internal/gp"
	"tt/internal/output"
)

const siteLong = `营运据点取自 GP 的 azp_file（aooi901 工厂基本资料维护作业维护的那张表）。

每个据点有两个编号，别混：
  据点编号    azp01，如 COF / DEMO1 / DIGIWIN —— 人记得住的是这个
  资料库编号  azp03，如 cof_demo / demo1 / digiwin —— 查字典用的是这个（= Oracle schema）

两者的值经常不同（实测 30 个据点里 15 个不同），所以 --at 里写哪个都认：
先按据点编号找，找不到再按资料库编号找。

azp053='Y' 表示该据点有实际的 ERP 资料库；N 的照样列出来，但查字典会被拒。`

// NewSiteCmd 造顶层 `site` 命令。
func NewSiteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "site",
		Short: "营运据点（azp_file）：列全部据点 / 看一个",
		Long:  siteLong,
		Args:  cobra.NoArgs,
		RunE:  func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	c.AddCommand(newSiteListCmd(), newSiteShowCmd())
	return c
}

func newSiteListCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "列出全部营运据点（据点编号 / 名称 / 资料库编号）",
		Long: `列出该环境 azp_file 里的全部营运据点。

用 --env <环境> 点名；--all 连"没建库"（azp053≠Y）的也列出来 ——
默认只列能查字典的，免得选了一个查不动的。`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			t, err := resolveTarget(false)
			if err != nil {
				return err
			}
			if t.env.DB == nil {
				return envErr(t, "DB", 2, "环境 %q 没有配置数据库连接", t.env.Name).
					WithHint("在设置 → 站点管理 → 数据库里配置该环境的 Oracle 连接")
			}
			sites, err := gpcore.ListSites(context.Background(), *t.env.DB)
			if err != nil {
				return envErr(t, "GP_SITE", 5, "读营运据点失败: %v", err)
			}
			hidden := 0
			if !all {
				kept := make([]gpcore.Site, 0, len(sites))
				for _, s := range sites {
					if s.HasDB {
						kept = append(kept, s)
						continue
					}
					hidden++
				}
				sites = kept
			}

			if common.IsJSON() {
				meta := metaNotes(t.meta, "营运据点 azp_file")
				meta.TotalRows = len(sites)
				meta.Returned = len(sites)
				return output.Emit(stdout, output.Options{
					Format: common.OutputFormat(),
					Meta:   meta,
					Data: map[string]any{
						"环境":   t.env.Name,
						"据点":   sites,
						"条数":   len(sites),
						"隐藏":   hidden,
						"显示全部": all,
					},
				})
			}

			rows := make([][]string, 0, len(sites))
			for _, s := range sites {
				db := "有"
				if !s.HasDB {
					db = "未建库"
				}
				rows = append(rows, []string{s.Code, s.Schema, s.Name, db})
			}
			output.PrintTable([]string{"据点编号", "资料库编号", "名称", "可查字典"}, rows)
			if hidden > 0 {
				fmt.Printf("\n（另有 %d 个据点没建库，没列出来；要看加 --all）\n", hidden)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "连没建库（azp053≠Y）的据点也列出来")
	return cmd
}

func newSiteShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <据点编号|资料库编号>",
		Short: "看一个据点（据点编号或资料库编号都认）",
		Long: `按据点编号（azp01）或资料库编号（azp03）查一个据点。

这就是 --at <环境>@<编号> 背后用的那条解析：写错、没建库都会当场报错，
不会静默查出一片空。`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			t, err := resolveTarget(false)
			if err != nil {
				return err
			}
			if t.env.DB == nil {
				return envErr(t, "DB", 2, "环境 %q 没有配置数据库连接", t.env.Name).
					WithHint("在设置 → 站点管理 → 数据库里配置该环境的 Oracle 连接")
			}
			site, err := gpcore.ResolveSchema(context.Background(), *t.env.DB, strings.TrimSpace(args[0]))
			if err != nil {
				return envErr(t, "GP_SITE", 4, "%v", err)
			}
			if common.IsJSON() {
				meta := metaNotes(t.meta, "营运据点 azp_file")
				return output.Emit(stdout, output.Options{
					Format: common.OutputFormat(),
					Meta:   meta,
					Data: map[string]any{
						"环境":    t.env.Name,
						"据点编号":  site.Code,
						"资料库编号": site.Schema,
						"名称":    site.Name,
						"可查字典":  site.HasDB,
					},
				})
			}
			fmt.Printf("据点编号：%s\n资料库编号：%s\n名称：%s\n", site.Code, site.Schema, site.Name)
			return nil
		},
	}
}
