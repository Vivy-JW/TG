package dict

// 营运据点（azp_file）查询：设置页「营运据点」页签的「从服务器刷新」，
// 以及 --at <环境>@<据点> 的解析。
//
// 为什么要有这一项：GP 一个库里装了几十家公司，每家公司一套字典
// （zta_file 按 zta02 分行）。用户不知道某个据点代号对应哪家公司，
// 而代号写错不报错、只是查出来空 —— 所以给一个"连库问一下"的按钮，
// 从 azp_file（营运中心主档）把代号和名字抄下来，连同 config.json 一起存。
//
// 口径：固定读 **ds** schema 的 azp_file。azp_file 是系统主档，各 schema
// 下内容一致（实测 ds / digiwin 都是 30 行），用 ds 读避免"据点还没配好
// 就连不上"的鸡生蛋问题。

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"tg/internal/dbconfig"
	"tg/internal/dict/live"
)

// siteCodeRE 据点代号的合法字符。据点代号会成为 Oracle schema 名，拼进 SQL 前
// 必须限死字符集 —— erpdb 的 Query 只收 SQL 文本、没有绑定参数。
var siteCodeRE = regexp.MustCompile(`^[A-Za-z0-9_$#]+$`)

// ValidSiteCode 据点/资料库编号是否只含允许的字符。
func ValidSiteCode(code string) bool { return siteCodeRE.MatchString(code) }

// SiteNameFor 在已经连上的库上按资料库编号取据点名称（azp_file.azp02）；取不到返回空串。
// 连接由调用方持有，这里不再另开一次 —— 测试据点时只连一次库。
func SiteNameFor(ctx context.Context, l *live.Live, schema string) string {
	if !siteCodeRE.MatchString(schema) {
		return ""
	}
	_, rows, err := l.Connector().Query(ctx,
		`SELECT azp02 FROM azp_file WHERE lower(azp03) = '`+strings.ToLower(schema)+`'`)
	if err != nil || len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	return strings.TrimSpace(rows[0][0])
}

// Site 一个营运据点：据点编号 + 名称 + 资料库编号。
//
// azp03 在 aooi901 的字段说明里叫「资料库编号」（默认 ds），也就是 zta_file.zta02
// 与 Oracle schema。它跟据点编号 azp01 **不是一回事**：实测 30 个据点里有 15 个
// 两者不同（COF → cof_demo、DEMO1 → demo1、DIGIWIN → digiwin），所以查字典要用
// azp03，不能拿 azp01 小写去猜。
//
// azp053 = 'Y' 表示该据点有实际的 ERP 数据库；p_cron / p_flownoty / p_zta 取
// 资料库清单时都是 WHERE azp053='Y'，所以 N 的行不该进候选。
type Site struct {
	Code   string `json:"code"`
	Name   string `json:"name"`
	Schema string `json:"schema"`
	HasDB  bool   `json:"hasDB"`
}

// ListSites 连库把 azp_file 的据点全表取回来（按代号排序）。
//
// 一次取全表而不是逐个查：设置页要的是"挑一个"，把所有据点摆成一张表
// 比"先记住代号再来查名字"顺手得多；azp_file 实测只有 30 行，代价可以忽略。
func ListSites(ctx context.Context, c dbconfig.Connection) ([]Site, error) {
	l, err := live.OpenCompany(ctx, c, SyncCompany)
	if err != nil {
		return nil, err
	}
	defer l.Close()

	cols, rows, err := l.Connector().Query(ctx,
		`SELECT azp01, azp02, azp03, azp053 FROM azp_file ORDER BY azp01`)
	if err != nil {
		return nil, fmt.Errorf("查 azp_file 失败：%w", err)
	}
	if len(cols) < 4 {
		return nil, fmt.Errorf("azp_file 的列不对：%v", cols)
	}
	out := make([]Site, 0, len(rows))
	for _, r := range rows {
		code := strings.ToUpper(strings.TrimSpace(r[0]))
		if code == "" {
			continue
		}
		out = append(out, Site{
			Code:   code,
			Name:   strings.TrimSpace(r[1]),
			Schema: strings.ToLower(strings.TrimSpace(r[2])),
			HasDB:  strings.EqualFold(strings.TrimSpace(r[3]), "Y"),
		})
	}
	return out, nil
}

// LookupSite 按据点代号取名字（大小写不敏感）。
//
// 返回 found=false 不是错误：像 "ds" 这种系统库在 azp_file 里没有对应据点
// （实测 30 个据点里没有 ds），用户仍然可以只填代号、名称留空。
func LookupSite(ctx context.Context, c dbconfig.Connection, code string) (site Site, found bool, err error) {
	want := strings.ToUpper(strings.TrimSpace(code))
	if want == "" {
		return Site{}, false, nil
	}
	sites, err := ListSites(ctx, c)
	if err != nil {
		return Site{}, false, err
	}
	for _, s := range sites {
		if strings.EqualFold(s.Code, want) {
			return s, true, nil
		}
	}
	return Site{}, false, nil
}

// ResolveSchema 把「据点编号」或「资料库编号」解析成要连的 schema。
//
// 先按 azp01（据点编号）找，找不到再按 azp03（资料库编号）找：实测 30 个据点里
// 有 15 个两者不同（COF → cof_demo、SYSTEM → ds），而人记得住的多半是据点编号。
//
// 三种情况都报错、不静默查空：查不到这一行、azp053≠Y（该据点没建库）、连不上库。
// 查空是最难查的失败 —— 用户会以为据点没数据，其实只是代号写错了。
func ResolveSchema(ctx context.Context, c dbconfig.Connection, code string) (Site, error) {
	want := strings.TrimSpace(code)
	if want == "" {
		return Site{}, fmt.Errorf("没有给据点编号")
	}
	if !siteCodeRE.MatchString(want) {
		return Site{}, fmt.Errorf("据点/资料库编号 %q 里有不允许的字符（只收字母、数字、_ $ #）", code)
	}
	sites, err := ListSites(ctx, c)
	if err != nil {
		return Site{}, err
	}
	pick := func(s Site) (Site, error) {
		if !s.HasDB {
			return Site{}, fmt.Errorf("据点 %s（%s）的 azp053 不是 Y：这个据点没有实际的 ERP 资料库，查不了字典",
				s.Code, s.Name)
		}
		if s.Schema == "" {
			return Site{}, fmt.Errorf("据点 %s（%s）没有填资料库编号（azp03），不知道要连哪个 schema",
				s.Code, s.Name)
		}
		return s, nil
	}
	for _, s := range sites {
		if strings.EqualFold(s.Code, want) {
			return pick(s)
		}
	}
	for _, s := range sites {
		if s.Schema != "" && strings.EqualFold(s.Schema, want) {
			return pick(s)
		}
	}
	return Site{}, fmt.Errorf("azp_file 里没有据点编号或资料库编号等于 %q 的行；清单：tg site list --env <环境>", code)
}
