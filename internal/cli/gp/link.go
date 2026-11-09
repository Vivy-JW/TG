package gp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/erpdb"
	"tt/internal/output"
)

type linkRow struct {
	Module     string `json:"module"`
	LinkedProg string `json:"linkedProg"`
	LinkFlag   string `json:"linkFlag,omitempty"`
	Entry      bool   `json:"entry"`
}

type linkResult struct {
	Program      string    `json:"program"`
	Count        int       `json:"count"`
	EntryCount   int       `json:"entryCount"`
	EntryModules []string  `json:"entryModules"`
	Links        []linkRow `json:"links"`
}

func newLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "link <作业/程序>",
		Aliases: []string{"p_link"},
		Short:   "p_link：查作业的程序链接（远程 Oracle）",
		Long: `p_link 的完整程序链接查询（远程 Oracle 的 gal_file）。

  tt gp link abmi600 --env 22主机
  tt gp p_link abmi600 --env 22主机

p_link 是一对多：同一个作业会链到多个程序/模块。入口行是
gal03 == gal01（程序名等于作业代号），输出里 entry=true。
不能只取 MAX(gal02) 压成一行，否则会选中子程序所在的模块。

数据源：远程 gal_file（不依赖本地 t_prog_link，后者目前不是全量）。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			setHumanDefault(cmd)
			return runLink(args[0])
		},
	}
}

func runLink(prog string) error {
	prog = strings.TrimSpace(prog)
	if prog == "" {
		return output.Errorf("USAGE", 2, "作业/程序代号不能为空")
	}
	t, err := resolveTarget(false)
	if err != nil {
		return err
	}

	ctx, cancel := withTimeout()
	defer cancel()
	l, err := openLive(ctx, t)
	if err != nil {
		return err
	}
	defer l.Close()

	sql := fmt.Sprintf(`
SELECT COALESCE(gal01,''), COALESCE(gal02,''), COALESCE(gal03,''), COALESCE(gal04,'')
  FROM gal_file
 WHERE UPPER(gal01) = UPPER(%s)
 ORDER BY CASE WHEN UPPER(gal03) = UPPER(gal01) THEN 0 ELSE 1 END,
          gal02, gal03`, erpdb.QuoteLit(prog))
	_, rows, err := queryLive(ctx, l, t, sql)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return envErr(t, "NOT_FOUND", 4, "p_link 没有作业 %q 的链接记录", prog).
			WithHint("确认作业代号（不要带模块前缀）；p_link 查作业，表结构用 tt gp table <表>")
	}

	res := linkResult{Program: prog}
	seenEntry := map[string]bool{}
	for _, r := range rows {
		gal01 := strings.TrimSpace(cell(r, 0))
		module := strings.ToLower(strings.TrimSpace(cell(r, 1)))
		linked := strings.TrimSpace(cell(r, 2))
		flag := strings.TrimSpace(cell(r, 3))
		entry := strings.EqualFold(linked, gal01)
		res.Links = append(res.Links, linkRow{
			Module:     module,
			LinkedProg: linked,
			LinkFlag:   flag,
			Entry:      entry,
		})
		if entry {
			res.EntryCount++
			if module != "" && !seenEntry[module] {
				seenEntry[module] = true
				res.EntryModules = append(res.EntryModules, module)
			}
		}
	}
	res.Count = len(res.Links)

	notes := []string{}
	if res.EntryCount == 0 {
		notes = append(notes, "没有 gal03==gal01 的入口行；链接数据可能不完整")
	}
	if res.EntryCount > 1 {
		notes = append(notes, fmt.Sprintf("有 %d 个入口行；入口模块：%s", res.EntryCount, strings.Join(res.EntryModules, ",")))
	}

	meta := metaNotes(t.meta, notes...)
	meta.TotalRows = res.Count
	meta.Returned = res.Count

	return output.Emit(stdout, output.Options{
		Format: common.OutputFormat(),
		Meta:   meta,
		Data:   res,
		Columns: []string{
			"模块", "链接程序", "链接标记", "入口",
		},
		Rows: linkRows(res.Links),
	})
}

func linkRows(links []linkRow) [][]string {
	out := make([][]string, 0, len(links))
	for _, l := range links {
		entry := ""
		if l.Entry {
			entry = "Y"
		}
		out = append(out, []string{l.Module, l.LinkedProg, l.LinkFlag, entry})
	}
	return out
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return row[i]
}
