package interfacecmd

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict"
	"tt/internal/dict/db"
	"tt/internal/dict/live"
	"tt/internal/output"
	"tt/internal/screen"
)

// 画面文字（标签）的来源。
//
// T100 把标签编译成 .42s 字符串表，运行期直接读；GP 没有 .42s —— 它的画面文字
// 由 lib/4gl/cl_ui_locale.4gl 在运行期从 gae_file 取（该 lib 三处 SELECT 都注释了
// #p_perlang），p_perlang「畫面元件顯示多語言設定作業」维护的就是 gae01 程序 /
// gae02 元件 / gae03 语系 / gae04 译文。所以这里的标签来源只有两个：
//
//	gae_file  本地字典库 t_ui_text（同步来的），或远程直查 gae_file
//	42f 自带  .42f 里 <Page text>/<Label text> 的那份默认文案（多半是英文）
//
// 42f 自带文案由 form42f 解析后已经在 screen 的 textKey 里，找不到 gae 译文时
// screen 会把它标成 unresolved 并在 reason 里说清楚，不冒充中文。

const (
	labelsAuto  = "auto"
	labelsLocal = "local"
	labelsLive  = "live"
	labelsOff   = "off"
)

// langCode 把 --lang 归一成 gae03 的语系码：zh_TW → 0（繁），其余 → 2（简）。
// 也接受直接给码（"0"/"2"）或 gae_file 里的原始值。
func langCode(lang string) string {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh_tw", "zh-tw", "tw", "big5":
		return "0"
	case "", "zh_cn", "zh-cn", "cn", "gbk", "zh":
		return "2"
	}
	s := strings.TrimSpace(lang)
	if s != "" && len(s) <= 2 {
		return s
	}
	return "2"
}

// envNamed 取这次点名的环境（--at 与 --env 都写 common.Env）。
func envNamed() (*config.NamedSsh, error) {
	name := strings.TrimSpace(common.Env)
	if name == "" {
		return nil, usage("标签默认从 gae_file 取，需要点名环境：--at <环境>（或 --labels off 只看结构）").
			WithHint("不知道有哪些环境：tt interface --at-list")
	}
	cfgPath, err := common.ResolveConfig(false)
	if err != nil {
		return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	root, err := config.Load(cfgPath)
	if err != nil {
		return nil, output.Errorf(output.CodeConfigBroken, 2, "%v", err)
	}
	e := root.Hosts.ByName(name)
	if e == nil {
		return nil, output.Errorf(output.CodeEnvUnknown, 2, "未找到环境 %q（tg env list 看全部）", name)
	}
	return e, nil
}

// loadLabels 取 prog 的 token → 译文；第二个返回值是给人看的来源说明。
func loadLabels(prog string) (screen.MapLabels, string, error) {
	mode := strings.ToLower(strings.TrimSpace(f.labels))
	if mode == "" {
		mode = labelsAuto
	}
	if mode == labelsOff {
		return nil, "off", nil
	}
	if mode != labelsAuto && mode != labelsLocal && mode != labelsLive {
		return nil, "", usage("--labels 只能是 auto/local/live/off（当前 %q）", f.labels)
	}
	lang := langCode(f.lang)
	env, err := envNamed()
	if err != nil {
		if mode == labelsAuto {
			// 一点名信息都没有时，退成"只有结构"仍然有用，不拦人。
			return nil, "none", nil
		}
		return nil, "", err
	}
	if mode == labelsAuto || mode == labelsLocal {
		m, note, err := labelsFromLocal(*env, prog, lang)
		if err == nil && len(m) > 0 {
			return m, note, nil
		}
		if mode == labelsLocal {
			if err != nil {
				return nil, "", err
			}
			return nil, "none", nil
		}
	}
	m, note, err := labelsFromLive(*env, prog, lang)
	if err != nil {
		if mode == labelsLive {
			return nil, "", err
		}
		return nil, "none", nil
	}
	return m, note, nil
}

func labelsFromLocal(env config.NamedSsh, prog, lang string) (screen.MapLabels, string, error) {
	path := dict.DictPath(env)
	d, err := db.Open(path)
	if err != nil {
		return nil, "", output.Errorf("LABELS_INVALID", 1, "打开本地字典库失败 (%s): %v", path, err)
	}
	defer d.Close()
	rows, err := d.QueryUITexts(prog, lang)
	if err != nil {
		return nil, "", output.Errorf("LABELS_INVALID", 1, "%v", err)
	}
	if len(rows) == 0 {
		return nil, "", nil
	}
	m := pickTexts(rows)
	return m, fmt.Sprintf("gae_file(local:%s,lang=%s,%d)", env.Name, lang, len(m)), nil
}

func labelsFromLive(env config.NamedSsh, prog, lang string) (screen.MapLabels, string, error) {
	if env.DB == nil {
		return nil, "", usage("环境 %s 没配数据库连接，取不到 gae_file；先 tg dict db sync 或只给 --42f", env.Name)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	l, err := live.Open(ctx, *env.DB)
	if err != nil {
		return nil, "", output.Errorf("LABELS_INVALID", 1, "连库读 gae_file 失败: %v", err)
	}
	defer l.Close()
	rows, err := l.QueryUITexts(prog, lang)
	if err != nil {
		return nil, "", output.Errorf("LABELS_INVALID", 1, "%v", err)
	}
	if len(rows) == 0 {
		return nil, "", nil
	}
	m := pickTexts(rows)
	return m, fmt.Sprintf("gae_file(live:%s,lang=%s,%d)", env.Name, lang, len(m)), nil
}

// pickTexts 把可能多行（客制/行业别维度）的译文收成 token → 文本：
// 客制 Y 优先、其次非空文本；同一 token 没有唯一答案时不猜（跳过，留给 unresolved）。
func pickTexts(rows []db.UIElementText) screen.MapLabels {
	sort.SliceStable(rows, func(i, j int) bool {
		ci, cj := isYes(rows[i].Cust), isYes(rows[j].Cust)
		if ci != cj {
			return ci
		}
		return rows[i].Profile < rows[j].Profile
	})
	m := screen.MapLabels{}
	for _, r := range rows {
		token := strings.TrimSpace(r.Element)
		text := strings.TrimSpace(r.Text)
		if token == "" || text == "" {
			continue
		}
		if _, ok := m[token]; ok {
			continue
		}
		m[token] = text
	}
	return m
}

func isYes(s string) bool {
	s = strings.TrimSpace(s)
	return strings.EqualFold(s, "y") || s == "Y"
}
