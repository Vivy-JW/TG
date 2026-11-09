package gp

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/erpdb"
	"tt/internal/output"
)

type msgRow struct {
	Code     string `json:"code"`
	Lang     string `json:"lang"`
	LangName string `json:"langName,omitempty"`
	Text     string `json:"text"`
	Updated  string `json:"updated,omitempty"`
	Popup    string `json:"popup,omitempty"`
}

type msgResult struct {
	Code     string   `json:"code"`
	Lang     string   `json:"lang,omitempty"`
	Count    int      `json:"count"`
	Messages []msgRow `json:"messages"`
}

func newMsgCmd() *cobra.Command {
	var lang, code string
	cmd := &cobra.Command{
		Use:     "msg [消息编号]",
		Aliases: []string{"p_ze"},
		Short:   "p_ze：查错误消息（远程 Oracle）",
		Long: `p_ze 的错误消息查询（远程 Oracle 的 ze_file）。

  tt gp msg -0101 --env 22主机
  tt gp msg -0101 --env 22主机 --lang zh_CN
  tt gp msg --code=-0101 --env 22主机
  tt gp p_ze -0101 --env 22主机

ze02 是语言代码：0=繁体，2=简体（1 目前样本未出现，按原值返回）。
--lang 可用数字或别名：
  zh_TW / zh-HK / 0   → 0
  zh_CN / zh-Hans / 2 → 2

不给 --lang 时返回该编号的全部语言行，并按 ze02 排序。
消息编号允许带前导减号；也可用 --code=-0101 避免被当成 flag。`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			setHumanDefault(cmd)
			if len(args) > 0 {
				code = args[0]
			}
			return runMsg(code, lang)
		},
	}
	cmd.Flags().StringVar(&lang, "lang", "", "语言：zh_TW/zh_CN 或 0/1/2（不给=全部）")
	cmd.Flags().StringVar(&code, "code", "", "消息编号（用于 -0101 这种会被当成 flag 的编号）")
	return cmd
}

func runMsg(code, lang string) error {
	code = strings.TrimSpace(code)
	if code == "" {
		return output.Errorf("USAGE", 2, "消息编号不能为空")
	}
	lang, err := normalizeLang(lang)
	if err != nil {
		return err
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

	where := fmt.Sprintf("UPPER(ze01) = UPPER(%s)", erpdb.QuoteLit(code))
	if lang != "" {
		where += " AND ze02 = " + erpdb.QuoteLit(lang)
	}
	sql := fmt.Sprintf(`
SELECT NVL(ze01,''), NVL(ze02,''), NVL(ze03,''),
       NVL(TO_CHAR(ze04,'YYYY-MM-DD'),''), NVL(ze05,'')
  FROM ze_file
 WHERE %s
 ORDER BY ze02`, where)
	_, rows, err := queryLive(ctx, l, t, sql)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		hint := "确认消息编号（p_ze 的编号可能带前导减号，例如 -0101）"
		if lang != "" {
			hint = "确认该编号有 ze02=" + lang + " 的语言行；去掉 --lang 可看全部语言"
		}
		return envErr(t, "NOT_FOUND", 4, "p_ze 没有消息 %q 的记录", code).WithHint(hint)
	}

	res := msgResult{Code: code, Lang: lang}
	for _, r := range rows {
		m := msgRow{
			Code:    strings.TrimSpace(cell(r, 0)),
			Lang:    strings.TrimSpace(cell(r, 1)),
			Text:    cell(r, 2),
			Updated: strings.TrimSpace(cell(r, 3)),
			Popup:   strings.TrimSpace(cell(r, 4)),
		}
		m.LangName = langName(m.Lang)
		res.Messages = append(res.Messages, m)
	}
	res.Count = len(res.Messages)

	meta := metaNotes(t.meta)
	meta.TotalRows = res.Count
	meta.Returned = res.Count

	return output.Emit(stdout, output.Options{
		Format: common.OutputFormat(),
		Meta:   meta,
		Data:   res,
		Columns: []string{
			"编号", "语言", "语言名", "正文", "更新日期", "弹窗",
		},
		Rows: msgRows(res.Messages),
	})
}

func msgRows(msgs []msgRow) [][]string {
	out := make([][]string, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, []string{m.Code, m.Lang, m.LangName, m.Text, m.Updated, m.Popup})
	}
	return out
}

// normalizeLang 把 --lang 归一到 ze_file.ze02 的实际取值。
// 只接受确定的别名；未知值直接报错，避免把拼错的语言当成“查不到消息”。
func normalizeLang(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	switch strings.ToLower(s) {
	case "0", "zh_tw", "zh-tw", "zh_hant", "zh-hant", "tw", "zh_hk", "zh-hk":
		return "0", nil
	case "2", "zh_cn", "zh-cn", "zh_hans", "zh-hans", "cn", "zh_sg", "zh-sg":
		return "2", nil
	case "1":
		return "1", nil
	}
	return "", output.Errorf("USAGE", 2, "不认识的 --lang %q", s).
		WithHint("用 zh_TW/zh_CN 或 0/1/2；不给 --lang 可看全部语言")
}

func langName(lang string) string {
	switch strings.TrimSpace(lang) {
	case "0":
		return "繁体"
	case "1":
		return "语言1"
	case "2":
		return "简体"
	default:
		return ""
	}
}
