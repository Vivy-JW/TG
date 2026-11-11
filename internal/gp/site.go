package gp

// POST /api/gp/site 的 HTTP 面：设置页「营运据点」页签。
//
// 纯查询逻辑在 internal/dict/site.go（ListSites / LookupSite / ResolveSchema）；
// 这里只负责从请求里取环境、回信封 —— 环境清单是 GP 服务（config 的 hosts.sshs）
// 的事，字典层不该认识 Service。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"tt/internal/config"
	dictlib "tt/internal/dict"
)

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
		sites, err := dictlib.ListSites(r.Context(), *env.DB)
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "count": len(sites), "sites": sites})
		return
	}
	site, found, err := dictlib.LookupSite(r.Context(), *env.DB, body.Code)
	if err != nil {
		writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "found": found, "code": site.Code, "name": site.Name, "schema": site.Schema, "hasDB": site.HasDB})
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
