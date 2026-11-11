package gp

// 营运据点（azp_file）查询：设置页「营运据点」页签的「从服务器刷新」。
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
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/dict/live"
)

// siteCodeRE 据点代号的合法字符。据点代号会成为 Oracle schema 名，拼进 SQL 前
// 必须限死字符集 —— erpdb 的 Query 只收 SQL 文本、没有绑定参数。
var siteCodeRE = regexp.MustCompile(`^[A-Za-z0-9_$#]+$`)

// siteNameFor 在已经连上的库上取据点名称（azp_file.azp02）；取不到返回空串。
// 连接由调用方持有，这里不再另开一次 —— 测试据点时只连一次库。
func siteNameFor(ctx context.Context, l *live.Live, company string) string {
	if !siteCodeRE.MatchString(company) {
		return ""
	}
	_, rows, err := l.Connector().Query(ctx,
		`SELECT azp02 FROM azp_file WHERE lower(azp01) = '`+strings.ToLower(company)+`'`)
	if err != nil || len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	return strings.TrimSpace(rows[0][0])
}

// Site 一个营运据点：代号 + 名称。
type Site struct {
	Code string `json:"code"`
	Name string `json:"name"`
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

	cols, rows, err := l.Connector().Query(ctx, `SELECT azp01, azp02 FROM azp_file ORDER BY azp01`)
	if err != nil {
		return nil, fmt.Errorf("查 azp_file 失败：%w", err)
	}
	if len(cols) < 2 {
		return nil, fmt.Errorf("azp_file 的列不对：%v", cols)
	}
	out := make([]Site, 0, len(rows))
	for _, r := range rows {
		code := strings.ToUpper(strings.TrimSpace(r[0]))
		if code == "" {
			continue
		}
		out = append(out, Site{Code: code, Name: strings.TrimSpace(r[1])})
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

// hSite 处理 POST /api/gp/site：
//
//	{"env":"22主机"}                → 据点全表（页签打开/刷新时用）
//	{"env":"22主机","code":"DIGIWIN"} → 按代号取一个
//
// 只读：不写配置（回写由前端拿着 code/name 走 /api/gp/env 的补丁保存）。
func (s *Service) hSite(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env  string `json:"env"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	env, err := s.findEnv(body.Env)
	if err != nil {
		writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if env.DB == nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "这个环境没有配置数据库连接"})
		return
	}
	if strings.TrimSpace(body.Code) == "" {
		sites, err := ListSites(r.Context(), *env.DB)
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "count": len(sites), "sites": sites})
		return
	}
	site, found, err := LookupSite(r.Context(), *env.DB, body.Code)
	if err != nil {
		writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "found": found, "code": site.Code, "name": site.Name})
}

// findEnv 按名字取一个 GP 环境（大小写不敏感）。找不到返回 404 级错误。
func (s *Service) findEnv(name string) (*config.NamedSsh, error) {
	envs, err := s.gpEnvs()
	if err != nil {
		return nil, err
	}
	want := strings.TrimSpace(name)
	for i := range envs {
		if strings.EqualFold(envs[i].Name, want) {
			return &envs[i], nil
		}
	}
	return nil, fmt.Errorf("环境不存在：%s", want)
}
