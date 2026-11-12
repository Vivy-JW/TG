package live

import (
	"fmt"
	"strings"

	"tt/internal/dict/db"
)

// QueryUITexts 直查 gae_file：画面元件译文。
//
// T100 把这份数据编译成 .42s 字符串表；GP 没有 .42s，运行期由
// lib/4gl/cl_ui_locale.4gl 从 gae_file 取（该 lib 三处 SELECT 都注释了
// #p_perlang），p_perlang「畫面元件顯示多語言設定作業」维护的就是它。
//
// 没有绑定参数（见 erpdb.Connector.Query），字面量统一走 lit。
func (l *Live) QueryUITexts(prog, lang string) ([]db.UIElementText, error) {
	prog = strings.TrimSpace(prog)
	lang = strings.TrimSpace(lang)
	if prog == "" || lang == "" {
		return nil, nil
	}
	rows, err := l.q(fmt.Sprintf(`
		SELECT TRIM(gae01), TRIM(gae02), TRIM(gae03), TRIM(gae04),
		       COALESCE(TRIM(gae11), ''), COALESCE(TRIM(gae12), '')
		  FROM gae_file
		 WHERE UPPER(TRIM(gae01)) = UPPER(%s)
		   AND TRIM(gae03) = %s
		 ORDER BY gae02, gae11 DESC, gae12`, lit(prog), lit(lang)))
	if err != nil {
		return nil, err
	}
	out := make([]db.UIElementText, 0, len(rows))
	for _, r := range rows {
		out = append(out, db.UIElementText{
			Prog:    strings.ToLower(strings.TrimSpace(get(r, 0))),
			Element: strings.TrimSpace(get(r, 1)),
			Lang:    strings.TrimSpace(get(r, 2)),
			Text:    strings.TrimSpace(get(r, 3)),
			Cust:    strings.TrimSpace(get(r, 4)),
			Profile: strings.ToLower(strings.TrimSpace(get(r, 5))),
		})
	}
	return out, nil
}
