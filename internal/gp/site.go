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
	"strings"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/dict/live"
)

// Site 一个营运据点：代号 + 名字。
type Site struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// LookupSite 连库查 azp_file，按据点代号取名字（大小写不敏感）。
//
// 返回 found=false 不是错误：像 "ds" 这种系统库在 azp_file 里没有对应据点
// （实测 30 个据点里没有 ds），用户仍然可以只填代号、名字留空。
func LookupSite(ctx context.Context, c dbconfig.Connection, code string) (site Site, found bool, err error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return Site{}, false, nil
	}
	l, err := live.OpenCompany(ctx, c, SyncCompany)
	if err != nil {
		return Site{}, false, err
	}
	defer l.Close()

	cols, rows, err := l.Connector().Query(ctx, `SELECT azp01, azp02 FROM azp_file ORDER BY azp01`)
	if err != nil {
		return Site{}, false, fmt.Errorf("查 azp_file 失败：%w", err)
	}
	if len(cols) < 2 {
		return Site{}, false, fmt.Errorf("azp_file 的列不对：%v", cols)
	}
	want := strings.ToUpper(code)
	for _, r := range rows {
		if strings.ToUpper(strings.TrimSpace(r[0])) == want {
			return Site{Code: strings.TrimSpace(r[0]), Name: strings.TrimSpace(r[1])}, true, nil
		}
	}
	return Site{}, false, nil
}

// hSite 处理 POST /api/gp/site：{"env":"22主机","code":"DIGIWIN"}。
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
