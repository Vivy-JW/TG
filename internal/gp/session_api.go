package gp

// 会话的 HTTP 面：建 / 列 / 执行命令 / 关闭。
//
// 与 T100 调试面的差别只在"建会话时跑什么"（见 internal/gpsession 的 Manager.New）；
// 对外的形状（id / env / prog / state / 停站位置）刻意保持一致，
// 前端将来可以从 /debug/api 平移到 /api/gp 而不用改数据结构。
//
// 会话本身不住在这个包里：它是一条 pty 的生命周期，归 internal/gpsession 管。

import (
	"encoding/json"
	"net/http"
	"time"

	"tt/internal/config"
)

// hSessionNew 建一条会话：POST {env, prog}。
func (s *Service) hSessionNew(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env  string `json:"env"`
		Prog string `json:"prog"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	if body.Prog == "" {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "程序编号必填"})
		return
	}
	envs, err := s.gpEnvs()
	if err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	var env *config.NamedSsh
	for i := range envs {
		if envs[i].Name == body.Env {
			env = &envs[i]
			break
		}
	}
	if env == nil {
		writeJSON(w, 404, map[string]any{"ok": false, "error": "环境不存在：" + body.Env})
		return
	}
	sess, err := s.sess.New(*env, body.Prog)
	if sess != nil {
		s.sess.Add(sess)
	}
	if err != nil {
		if sess != nil {
			writeJSON(w, 200, map[string]any{"ok": false, "error": err.Error(), "session": sess})
			return
		}
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true, "session": sess})
}

func (s *Service) hSessionList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "sessions": s.sess.List()})
}

// hSessionExec 执行一条 fgldb 命令：POST {id, cmd}。
func (s *Service) hSessionExec(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID  string `json:"id"`
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	sess := s.sess.ByID(body.ID)
	if sess == nil {
		writeJSON(w, 404, map[string]any{"ok": false, "error": "会话不存在或已关闭"})
		return
	}
	out, err := sess.Exec(body.Cmd, 30*time.Second)
	res := map[string]any{"ok": err == nil, "output": out, "file": sess.File, "line": sess.Line}
	if err != nil {
		res["error"] = err.Error()
	}
	writeJSON(w, 200, res)
}

func (s *Service) hSessionClose(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	if !s.sess.Close(body.ID) {
		writeJSON(w, 404, map[string]any{"ok": false, "error": "会话不存在"})
		return
	}
	writeJSON(w, 200, map[string]any{"ok": true})
}
