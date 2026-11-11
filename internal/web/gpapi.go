package web

// TOP GP 的 HTTP 面：/api/gp/*。
//
// 这里只做三件事：读配置里的 GP 环境、把请求翻译成执行面的调用、把结果写成 JSON。
// 每一项的实际实现都在别处：
//
//	环境补丁     internal/config（ApplyEnvPatch / MergeTestEnv / RootsForInstance）
//	源码镜像     internal/mirror（Pull + manifest）
//	字典同步     internal/dict（SyncEnv → dbsync.Run → SyncGP）
//	营运据点     internal/dict（ListSites / LookupSite）
//	调试会话     internal/gpsession（Manager）
//	后台任务     internal/web/jobs（Registry）
//
// 原来这些混在 internal/gp 一个包里（943 行），第③④步拆分后只剩这一层薄面。

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"tt/internal/config"
	dictlib "tt/internal/dict"
	"tt/internal/dict/dbsync"
	"tt/internal/dict/live"
	"tt/internal/envmove"
	_ "tt/internal/erpdb" // 注册 sql 驱动 "oracle"（go-ora）
	"tt/internal/gpsession"
	"tt/internal/host"
	"tt/internal/mirror"
	"tt/internal/web/jobs"
)

// GPAPI 是 /api/gp/ 的处理器。
type GPAPI struct {
	cfgPath string

	jobs *jobs.Registry

	// cfgMu 串行化 GP 侧的配置写入。设置页改成 500ms 自动保存后，一次输入风暴
	// 可能同时排进多个保存请求；config.EditHosts 是裸读-改-写，不串行就会互相覆盖。
	cfgMu sync.Mutex

	// reload 在 config.json 写成功后调用，通知持有内存配置的子系统重读。
	// 与 T100 的 /api/hosts 保存路径同一机制；否则改完 GP 环境，调试服务仍拿旧清单（实测 404）。
	reload func() error

	sess *gpsession.Manager
}

// NewGPAPI 建一个 GP 处理器。cfgPath 是 config.json 的路径（与统一服务同一份）。
func NewGPAPI(cfgPath string) *GPAPI {
	return &GPAPI{cfgPath: cfgPath, jobs: jobs.New(), sess: gpsession.NewManager()}
}

// SetReloader 注入配置写入后的重载回调。应在开始服务前调用。
func (s *GPAPI) SetReloader(f func() error) { s.reload = f }

func (s *GPAPI) job(name string) *jobs.Job { return s.jobs.Get(name) }

func (s *GPAPI) setJob(name string, f func(*jobs.Job)) { s.jobs.Update(name, f) }

func (s *GPAPI) progress(name, msg string) { s.jobs.Note(name, msg) }

// progressCount 同时更新“已完成/总数”和一条进度文字；镜像与字典同步共用。
func (s *GPAPI) progressCount(name string, done, total int, msg string) {
	s.jobs.Count(name, done, total, msg)
}

// ServeHTTP 让 *GPAPI 自己就是个 http.Handler（web.Options.GP 直接收它）。
func (s *GPAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.serve(w, r) }

func (s *GPAPI) serve(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/gp/meta":
		s.hMeta(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/env":
		s.hEnv(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/session/new":
		s.hSessionNew(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/api/gp/session/list":
		s.hSessionList(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/session/exec":
		s.hSessionExec(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/session/close":
		s.hSessionClose(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/sync":
		s.hSync(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/site":
		s.hSite(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/mirror":
		s.hMirror(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/gp/test":
		s.hTest(w, r)
	default:
		writeJSON(w, 404, map[string]any{"ok": false, "error": "没有这个端点: " + r.URL.Path})
	}
}

// gpEnvs 取全部 GP 环境。
func (s *GPAPI) gpEnvs() ([]config.NamedSsh, error) {
	h, err := config.LoadHosts(s.cfgPath)
	if err != nil {
		return nil, err
	}
	out := make([]config.NamedSsh, 0, len(h.SSHs))
	for _, e := range h.SSHs {
		if strings.EqualFold(e.Product, "gp") {
			out = append(out, e)
		}
	}
	return out, nil
}

// findEnv 按名字取一个 GP 环境（大小写不敏感）。
func (s *GPAPI) findEnv(name string) (*config.NamedSsh, error) {
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

// hMeta 环境清单 + 本地状态。只读 manifest/文件状态，不遍历目录、不打开 SQLite。
func (s *GPAPI) hMeta(w http.ResponseWriter, _ *http.Request) {
	envs, err := s.gpEnvs()
	if err != nil {
		writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	type envView struct {
		Name      string `json:"name"`
		Host      string `json:"host"`
		Port      int    `json:"port"`
		User      string `json:"user"`
		Password  string `json:"password"`
		HasPwd    bool   `json:"hasPassword"`
		Instance  string `json:"instance"`
		Workspace string `json:"workspace"`
		CustRoot  string `json:"custRoot"`
		// Company 营运据点代号（zta_file.zta02 / Oracle schema）；空 = "ds"。
		Company string `json:"company"`
		// CompanyName 营运据点名称（azp_file.azp02），「从服务器刷新」时抄下来。
		CompanyName string `json:"companyName"`
		// CompanySyncedAt 上次刷新的时间（RFC3339）；只是显示，不参与判断。
		CompanySyncedAt string         `json:"companySyncedAt"`
		MirrorDir       string         `json:"mirrorDir"`
		DictDir         string         `json:"dictDir"`
		DB              map[string]any `json:"db,omitempty"`
		Mirror          map[string]any `json:"mirror"`
		Dict            map[string]any `json:"dict"`
		Job             *jobs.Job      `json:"job"`
	}
	out := make([]envView, 0, len(envs))
	for _, e := range envs {
		md := mirror.Dir(e)
		mv := map[string]any{"path": md}
		if m, ok := mirror.ReadManifest(md); ok {
			mv["exists"] = true
			mv["files"] = m.Files
			mv["sizeMB"] = m.SizeMB
		} else if st, err := os.Stat(md); err == nil && st.IsDir() {
			mv["exists"] = true
		} else {
			mv["exists"] = false
		}
		dv := map[string]any{"path": dictlib.DictPath(e)}
		if ex, mb := dictlib.Stat(e); ex {
			dv["exists"] = true
			dv["sizeMB"] = mb
		} else {
			dv["exists"] = false
		}
		var db map[string]any
		if e.DB != nil {
			accs := make([]map[string]any, 0, len(e.DB.Accounts))
			for _, a := range e.DB.Accounts {
				accs = append(accs, map[string]any{
					"account": a.Account, "password": a.Password, "hasPassword": a.Password != "",
				})
			}
			db = map[string]any{"type": e.DB.Type, "host": e.DB.Host, "port": e.DB.Port,
				"sid": e.DB.Service, "service": e.DB.Service, "accounts": accs}
		}
		inst := e.Instance
		if inst == "" {
			inst = config.InstanceFromWorkspace(e.Workspace)
		}
		out = append(out, envView{
			Name: e.Name, Host: e.Host, Port: e.Port, User: e.User,
			Password: e.Password, HasPwd: e.Password != "",
			Instance: inst, Workspace: e.Workspace, CustRoot: e.CustRoot,
			Company: e.Company, CompanyName: e.CompanyName, CompanySyncedAt: e.CompanySyncedAt, MirrorDir: e.MirrorDir,
			DictDir: e.DictDir,
			DB:      db, Mirror: mv, Dict: dv, Job: s.job(e.Name),
		})
	}
	writeJSON(w, 200, map[string]any{"ok": true, "envs": out})
}

// hEnv 保存/删除一个 GP 环境（写回 config.json 的 hosts.sshs）。
//
// 保存走补丁语义：请求可带 patch（只改提交字段），也兼容旧版的整条 env。
// 这样设置页的自动保存只发“这次真正变了的字段”，不会把没碰过的 service、
// 密码、dictDir、mirrorDir 清掉。
func (s *GPAPI) hEnv(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action  string           `json:"action"`
		Name    string           `json:"name,omitempty"`
		OldName string           `json:"oldName,omitempty"`
		Env     *config.NamedSsh `json:"env,omitempty"`
		Patch   *config.EnvPatch `json:"patch,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON: " + err.Error()})
		return
	}
	if body.Action == "" {
		body.Action = "save"
	}
	name := strings.TrimSpace(body.Name)
	if name == "" && body.Env != nil {
		name = strings.TrimSpace(body.Env.Name)
	}
	if name == "" && body.Patch != nil && body.Patch.Name != nil {
		name = strings.TrimSpace(*body.Patch.Name)
	}
	if name == "" && body.Action != "delete" {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "环境名称必填"})
		return
	}

	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()

	// 改名和 T100 走同一条迁移管线：先搬按名字落盘的字典/快照/镜像，
	// 再写配置；写配置失败就搬回来。见 internal/envmove 的注释。
	var renameMoves []envmove.Move
	var renameOldName string

	err := config.EditHosts(s.cfgPath, func(h *config.Hosts) error {
		oldName := strings.TrimSpace(body.OldName)
		lookup := name
		if oldName != "" {
			lookup = oldName
		}
		idx := -1
		for i := range h.SSHs {
			if h.SSHs[i].Name == lookup {
				idx = i
				break
			}
		}
		if body.Action == "delete" {
			if idx < 0 {
				return fmt.Errorf("环境不存在：%s", name)
			}
			if !strings.EqualFold(h.SSHs[idx].Product, "gp") {
				return fmt.Errorf("「%s」不是 GP 环境，不能在 GP 页里删", name)
			}
			h.SSHs = append(h.SSHs[:idx], h.SSHs[idx+1:]...)
			return nil
		}

		var patch config.EnvPatch
		switch {
		case body.Patch != nil:
			patch = *body.Patch
		case body.Env != nil:
			patch = config.PatchFromEnv(*body.Env)
		default:
			return fmt.Errorf("保存请求缺少 env 或 patch")
		}

		if idx >= 0 && !strings.EqualFold(h.SSHs[idx].Product, "gp") {
			return fmt.Errorf("「%s」是 T100 环境，GP 页不能覆盖它；请换一个环境名", lookup)
		}
		if oldName != "" {
			if idx < 0 {
				return fmt.Errorf("原环境不存在：%s", oldName)
			}
			for i := range h.SSHs {
				if i != idx && h.SSHs[i].Name == name {
					return fmt.Errorf("环境名 %q 已被占用（T100 或 GP）", name)
				}
			}
			if oldName != name {
				moves, merr := envmove.Plan(s.cfgPath, oldName, name, h.SSHs[idx].Host, h.SSHs[idx].Zone)
				if merr != nil {
					return merr
				}
				if eerr := envmove.Execute(moves, name, oldName); eerr != nil {
					return eerr
				}
				renameMoves, renameOldName = moves, oldName
			}
		}

		var env config.NamedSsh
		if idx >= 0 {
			env = h.SSHs[idx]
		} else {
			env = config.NamedSsh{
				SSHConfig: config.SSHConfig{Port: 22},
				Name:      name,
				Product:   "gp",
			}
		}
		config.ApplyEnvPatch(&env, patch)
		// 改名走 oldName：目标名以请求的 name 为准，补丁里的 name 只做兜底。
		if name != "" {
			env.Name = name
		}
		env.Product = "gp"
		if strings.TrimSpace(env.Name) == "" {
			return fmt.Errorf("环境名称必填")
		}
		if strings.TrimSpace(env.Host) == "" {
			return fmt.Errorf("环境 %q 缺少 SSH 主机", env.Name)
		}
		if strings.TrimSpace(env.User) == "" {
			return fmt.Errorf("环境 %q 缺少 SSH 账号", env.Name)
		}
		// 实例名是 GP 的唯一入口：路径由它推导，用户不需要手填 $TOP/$CUST。
		// 旧配置没有 instance 时，从已有 workspace 反推一次，便于平滑迁移。
		if strings.TrimSpace(env.Instance) == "" {
			env.Instance = config.InstanceFromWorkspace(env.Workspace)
		}
		if env.Instance != "" {
			if !config.ValidInstance(env.Instance) {
				return fmt.Errorf("实例名 %q 不合法：只能包含字母、数字、下划线、点或连字符（如 toptest）", env.Instance)
			}
			env.Workspace, env.CustRoot = config.RootsForInstance(env.Instance)
		}
		if strings.TrimSpace(env.Workspace) == "" || strings.TrimSpace(env.CustRoot) == "" {
			return fmt.Errorf("环境 %q 缺少实例名：填 topprod / toptest / topstd 等，后台会推导 $TOP/$CUST", env.Name)
		}
		if idx >= 0 {
			h.SSHs[idx] = env
		} else {
			h.SSHs = append(h.SSHs, env)
		}
		return nil
	})
	if err != nil {
		if len(renameMoves) > 0 {
			if uerr := envmove.Undo(renameMoves, renameOldName); uerr != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error() + "；回滚落点失败：" + uerr.Error()})
				return
			}
		}
		writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	// 配置已落盘；和 T100 /api/hosts 保存后一样，通知调试服务重读环境清单。
	// 重载失败不回滚配置（已经写成功），只记日志，避免“保存成功但未生效”无人可见。
	if s.reload != nil {
		if rerr := s.reload(); rerr != nil {
			log.Printf("[gp] 环境已保存，但调试服务重载失败: %v", rerr)
		}
	}
	writeJSON(w, 200, map[string]any{"ok": true, "name": name})
}

// hTest 按页签分别测，三者互不牵连：
//
//	what=ssh  只测 SSH（连得上 + 能开 SFTP + $TOP 存在）
//	what=db   只测库（用表单里的账号直接 ping，不经过 SSH）
//	what=site 只测据点（按据点连库，看 zta_file 有没有这家公司的字典行）
//
// what 为空时按老行为把 ssh 和 db 都测一遍（旧前端还在这么调）。
func (s *GPAPI) hTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env  config.NamedSsh `json:"env"`
		What string          `json:"what"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	envs, _ := s.gpEnvs()
	env := body.Env
	for _, e := range envs {
		if e.Name == body.Env.Name {
			env = config.MergeTestEnv(body.Env, e)
			break
		}
	}
	what := strings.ToLower(strings.TrimSpace(body.What))
	res := map[string]any{}
	if what == "" || what == "ssh" {
		res["ssh"] = testSSH(env)
	}
	if what == "" || what == "db" {
		res["db"] = testDB(env)
	}
	if what == "site" {
		res["site"] = testSite(r.Context(), env)
	}
	writeJSON(w, 200, map[string]any{"ok": true, "result": res})
}

// testSSH 只测 SSH 这一项：端口能连、能开 SFTP、$TOP 可读。
func testSSH(env config.NamedSsh) map[string]any {
	conn, err := host.Dial(config.SSHConfig{Host: env.Host, Port: env.Port, User: env.User, Password: env.Password})
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	defer conn.Close()
	sc, err := conn.SFTP()
	if err != nil {
		return map[string]any{"ok": false, "msg": "SSH 通了，但打不开 SFTP: " + err.Error()}
	}
	defer sc.Close()
	if top := env.Workspace; top != "" {
		if st, err := sc.Stat(top); err != nil || !st.IsDir() {
			return map[string]any{"ok": false, "msg": "连上了，但 $TOP 不存在：" + top}
		}
		return map[string]any{"ok": true, "msg": "已连接，" + env.User + "@" + env.Host + "，$TOP 可读"}
	}
	return map[string]any{"ok": true, "msg": "已连接，" + env.User + "@" + env.Host}
}

// testDB 只测数据库这一项：拿表单里的账号直接连库 ping 一下，不依赖 SSH。
func testDB(env config.NamedSsh) map[string]any {
	if env.DB == nil || env.DB.Host == "" {
		return map[string]any{"ok": false, "msg": "还没有填数据库主机"}
	}
	var acct, pwd string
	for _, a := range env.DB.Accounts {
		if a.Account != "" {
			acct, pwd = a.Account, a.Password
			break
		}
	}
	if acct == "" {
		return map[string]any{"ok": false, "msg": "没有数据库账号"}
	}
	db, err := sql.Open("oracle", dbsync.DSN(*env.DB, acct, pwd))
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	return map[string]any{"ok": true, "msg": "已连接 " + acct + "@" + env.DB.Host + "/" + env.DB.Service}
}

// testSite 只测据点这一项：按据点连库，看 zta_file 里有没有这家公司的字典行，
// 顺带把 azp_file 里的名字捞出来。据点写错时不报错、只是查出来空 —— 这个按钮
// 就是用来提前发现这种"静默查空"的。
func testSite(ctx context.Context, env config.NamedSsh) map[string]any {
	if env.DB == nil || env.DB.Host == "" {
		return map[string]any{"ok": false, "msg": "还没有填数据库主机"}
	}
	schema := dictlib.Company(env)
	if !dictlib.ValidSiteCode(schema) {
		return map[string]any{"ok": false, "msg": "资料库编号只能由字母、数字、下划线组成：" + schema}
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	l, err := live.OpenCompany(ctx, *env.DB, schema)
	if err != nil {
		return map[string]any{"ok": false, "msg": err.Error()}
	}
	defer l.Close()

	_, rows, err := l.Connector().Query(ctx, `SELECT COUNT(*) FROM zta_file WHERE lower(zta02) = '`+schema+`'`)
	if err != nil {
		return map[string]any{"ok": false, "msg": "连上了，但读 zta_file 失败：" + err.Error()}
	}
	n := 0
	if len(rows) > 0 && len(rows[0]) > 0 {
		n, _ = strconv.Atoi(strings.TrimSpace(rows[0][0]))
	}
	if n == 0 {
		return map[string]any{"ok": false, "msg": "连上了，但 zta_file 里没有资料库「" + schema + "」的字典行（编号写错了？）"}
	}
	msg := "资料库 " + schema + " 可连：zta_file 有 " + strconv.Itoa(n) + " 张表"
	if name := dictlib.SiteNameFor(ctx, l, schema); name != "" {
		msg += "（" + name + "）"
	}
	return map[string]any{"ok": true, "msg": msg}
}

// hSite 处理 POST /api/gp/site：
//
//	{"env":"22主机"}                → 据点全表（页签打开/刷新时用）
//	{"env":"22主机","code":"DIGIWIN"} → 按代号取一个
//
// 只读：不写配置（回写由前端拿着 code/name 走 /api/gp/env 的补丁保存）。
func (s *GPAPI) hSite(w http.ResponseWriter, r *http.Request) {
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

// hMirror 起一轮源码镜像（只拉 *.4gl）。mode=incr 增量，其它当全量。
func (s *GPAPI) hMirror(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env  string `json:"env"`
		Mode string `json:"mode"`
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
	if cur := s.job(env.Name); cur.State == jobs.StateRunning {
		writeJSON(w, 200, map[string]any{"ok": true, "job": cur})
		return
	}
	s.jobs.Start(env.Name, "mirror")
	go s.runMirror(*env, body.Mode)
	writeJSON(w, 200, map[string]any{"ok": true, "job": s.job(env.Name)})
}

// hSync 起一轮字典同步（Oracle → 本地 SQLite）。
func (s *GPAPI) hSync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env string `json:"env"`
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
	if cur := s.job(env.Name); cur.State == jobs.StateRunning {
		writeJSON(w, 200, map[string]any{"ok": true, "job": cur})
		return
	}
	s.jobs.Start(env.Name, "sync")
	go s.runSync(*env)
	writeJSON(w, 200, map[string]any{"ok": true, "job": s.job(env.Name)})
}

func (s *GPAPI) runMirror(env config.NamedSsh, mode string) {
	out, err := mirror.Pull(env, mode, func(done, total int, msg string) {
		s.progressCount(env.Name, done, total, msg)
	})
	if err != nil {
		s.setJob(env.Name, func(j *jobs.Job) {
			j.State = jobs.StateError
			j.Error = err.Error()
			j.Ended = time.Now()
		})
		return
	}
	doneBase, totalBase := 0, 0
	for _, v := range out {
		if st, ok := v.(map[string]any); ok {
			if n, ok := st["processed"].(int); ok {
				doneBase += n
			}
			if n, ok := st["total"].(int); ok {
				totalBase += n
			}
		}
	}
	s.setJob(env.Name, func(j *jobs.Job) {
		j.State = jobs.StateDone
		j.Done = int64(doneBase)
		j.Total = int64(totalBase)
		j.Result = out
		j.Ended = time.Now()
	})
}

// runSync 跑一次字典同步，把结果写进 job。实际同步在 internal/dict 的 SyncEnv。
func (s *GPAPI) runSync(env config.NamedSsh) {
	ctx, cancel := context.WithTimeout(context.Background(), dictlib.SyncTimeout)
	defer cancel()
	st, err := dictlib.SyncEnv(ctx, env,
		func(msg string) { s.progress(env.Name, msg) },
		func(done, total int) {
			s.setJob(env.Name, func(j *jobs.Job) {
				j.Done = int64(done)
				j.Total = int64(total)
			})
		})
	s.setJob(env.Name, func(j *jobs.Job) {
		j.Ended = time.Now()
		if err != nil {
			j.State = jobs.StateError
			j.Error = err.Error()
			return
		}
		j.State = jobs.StateDone
		j.Done = int64(st.Tables)
		j.Total = int64(st.Tables)
		j.Result = map[string]any{
			"path": st.Path, "tables": st.Tables, "columns": st.Columns,
			"indexes": st.Indexes, "constraints": st.Constraints,
			"sizeMB": st.SizeMB, "messages": st.Messages,
		}
	})
}

// ---------------------------------------------------------------- 会话

// hSessionNew 建一条会话：POST {env, prog}。
func (s *GPAPI) hSessionNew(w http.ResponseWriter, r *http.Request) {
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
	env, err := s.findEnv(body.Env)
	if err != nil {
		writeJSON(w, 404, map[string]any{"ok": false, "error": err.Error()})
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

func (s *GPAPI) hSessionList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, 200, map[string]any{"ok": true, "sessions": s.sess.List()})
}

// hSessionExec 执行一条 fgldb 命令：POST {id, cmd}。
func (s *GPAPI) hSessionExec(w http.ResponseWriter, r *http.Request) {
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

func (s *GPAPI) hSessionClose(w http.ResponseWriter, r *http.Request) {
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
