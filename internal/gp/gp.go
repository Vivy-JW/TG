// Package gp —— TOP GP 的执行面（Go 实现）。
//
// 它是「TT 只有一个服务」这件事的第一步：GP 的环境、源码镜像、字典同步与调试会话
// 都挂在统一服务的 /api/gp/ 下，不再有第二个进程、第二个端口。
//
// 环境本身住在 config.json 的 hosts.sshs 里（NamedSsh.Product == "gp"），
// 与 T100 的环境同一份文件、同一个清单 —— 设置页读的是一处，左树读的也是同一处。
package gp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"

	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/envmove"
	"tt/internal/host"
)

// Job 一个后台动作（镜像/同步）的进度。同一环境同时只跑一个。
type Job struct {
	State     string         `json:"state"` // idle | running | done | error
	Kind      string         `json:"kind"`  // mirror | sync
	Done      int64          `json:"done,omitempty"`
	Total     int64          `json:"total,omitempty"`
	Progress  []string       `json:"progress,omitempty"`
	Error     string         `json:"error,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Elapsed   string         `json:"elapsed,omitempty"`
	StartedAt string         `json:"startedAt,omitempty"`
	Started   time.Time      `json:"-"`
	Ended     time.Time      `json:"-"`
}

// Service 是 /api/gp/ 的处理器。
type Service struct {
	cfgPath string

	mu   sync.Mutex
	jobs map[string]*Job

	// cfgMu 串行化 GP 侧的配置写入。设置页改成 500ms 自动保存后，一次输入风暴
	// 可能同时排进多个保存请求；config.EditHosts 是裸读-改-写，不串行就会互相覆盖。
	cfgMu sync.Mutex

	// reload 在 config.json 写成功后调用，通知持有内存配置的子系统重读。
	// 与 T100 的 /api/hosts 保存路径同一机制；否则改完 GP 环境，调试服务仍拿旧清单（实测 404）。
	reload func() error

	sessMu sync.Mutex
	sess   map[string]*Session
}

// SetReloader 注入配置写入后的重载回调。应在开始服务前调用。
func (s *Service) SetReloader(f func() error) { s.reload = f }

// New 建一个 GP 服务。cfgPath 是 config.json 的路径（与统一服务同一份）。
func New(cfgPath string) *Service {
	return &Service{cfgPath: cfgPath, jobs: map[string]*Job{}, sess: map[string]*Session{}}
}

func (s *Service) job(name string) *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[name]
	if !ok {
		return &Job{State: "idle"}
	}
	cp := *j
	cp.Progress = append([]string(nil), j.Progress...)
	if !j.Started.IsZero() {
		end := j.Ended
		if end.IsZero() {
			end = time.Now()
		}
		cp.Elapsed = end.Sub(j.Started).Round(time.Second).String()
		cp.StartedAt = j.Started.Format(time.RFC3339)
	}
	return &cp
}

func (s *Service) setJob(name string, f func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[name]
	if !ok {
		j = &Job{State: "idle"}
		s.jobs[name] = j
	}
	f(j)
}

func (s *Service) progress(name, msg string) {
	s.setJob(name, func(j *Job) {
		j.Progress = append(j.Progress, msg)
		if len(j.Progress) > 40 {
			j.Progress = j.Progress[len(j.Progress)-40:]
		}
	})
}

// progressCount 同时更新“已完成/总数”和一条进度文字；镜像与字典同步共用。
func (s *Service) progressCount(name string, done, total int, msg string) {
	s.setJob(name, func(j *Job) {
		j.Done = int64(done)
		j.Total = int64(total)
		if msg == "" {
			return
		}
		j.Progress = append(j.Progress, msg)
		if len(j.Progress) > 40 {
			j.Progress = j.Progress[len(j.Progress)-40:]
		}
	})
}

// Handler 返回挂在 /api/gp/ 下的处理器（路径自带 /api/gp 前缀）。
func (s *Service) Handler() http.Handler { return http.HandlerFunc(s.serve) }

// ServeHTTP 让 *Service 自己就是个 http.Handler（web.Options.GP 直接收它）。
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.serve(w, r) }

func (s *Service) serve(w http.ResponseWriter, r *http.Request) {
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

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// gpEnvs 取全部 GP 环境。
func (s *Service) gpEnvs() ([]config.NamedSsh, error) {
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

// gpInstanceRe 实例名只允许普通路径段，不能带 / 或空格。
func validGPInstance(inst string) bool {
	if inst == "" {
		return false
	}
	for _, r := range inst {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}

// gpRootsForInstance 实例名 → GP 的 $TOP / $CUST。
// 现场约定是 /u1/<实例>/tiptop 与 /u1/<实例>/topcust（TOPTEST 即 /u1/toptest/...）。
func gpRootsForInstance(inst string) (workspace, custRoot string) {
	inst = strings.TrimSpace(inst)
	if inst == "" {
		return "", ""
	}
	base := "/u1/" + inst
	return base + "/tiptop", base + "/topcust"
}

// gpInstanceFromWorkspace 旧配置里只有 workspace 时，从 /u1/<实例>/tiptop 反推实例名。
func gpInstanceFromWorkspace(workspace string) string {
	parts := strings.Split(strings.Trim(strings.TrimSpace(workspace), "/"), "/")
	for i, p := range parts {
		if strings.EqualFold(p, "tiptop") && i > 0 {
			return parts[i-1]
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

// mirrorDir 该环境的镜像目录：环境自己指定优先，否则 <默认根>/<环境名>。
func mirrorDir(e config.NamedSsh) string {
	if e.MirrorDir != "" {
		return e.MirrorDir
	}
	base := os.Getenv("TT_GP_MIRROR_DIR")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".tt", "gp-mirror")
	}
	return filepath.Join(base, e.Name)
}

type manifest struct {
	Files   int     `json:"files"`
	SizeMB  float64 `json:"sizeMB"`
	Source  string  `json:"source"`
	Updated string  `json:"at"`
}

func readManifest(dir string) (manifest, bool) {
	b, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return manifest{}, false
	}
	var m manifest
	if json.Unmarshal(b, &m) != nil {
		return manifest{}, false
	}
	return m, true
}

func writeManifest(dir string, files int, sizeMB float64, source string) {
	_ = os.MkdirAll(dir, 0o755)
	m := manifest{Files: files, SizeMB: sizeMB, Source: source, Updated: time.Now().Format("2006-01-02 15:04:05")}
	b, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), b, 0o644)
}

// hMeta 环境清单 + 本地状态。只读 manifest/文件状态，不遍历目录、不打开 SQLite。
func (s *Service) hMeta(w http.ResponseWriter, _ *http.Request) {
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
		// CompanyName 营运据点名字（azp_file.azp02），「从服务器刷新」时抄下来。
		CompanyName string         `json:"companyName"`
		MirrorDir   string         `json:"mirrorDir"`
		DictDir     string         `json:"dictDir"`
		DB          map[string]any `json:"db,omitempty"`
		Mirror      map[string]any `json:"mirror"`
		Dict        map[string]any `json:"dict"`
		Job         *Job           `json:"job"`
	}
	out := make([]envView, 0, len(envs))
	for _, e := range envs {
		md := mirrorDir(e)
		mv := map[string]any{"path": md}
		if m, ok := readManifest(md); ok {
			mv["exists"] = true
			mv["files"] = m.Files
			mv["sizeMB"] = m.SizeMB
		} else if st, err := os.Stat(md); err == nil && st.IsDir() {
			mv["exists"] = true
		} else {
			mv["exists"] = false
		}
		dv := map[string]any{"path": dictPath(e)}
		if ex, mb := dictStat(e); ex {
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
			inst = gpInstanceFromWorkspace(e.Workspace)
		}
		out = append(out, envView{
			Name: e.Name, Host: e.Host, Port: e.Port, User: e.User,
			Password: e.Password, HasPwd: e.Password != "",
			Instance: inst, Workspace: e.Workspace, CustRoot: e.CustRoot,
			Company: e.Company, CompanyName: e.CompanyName, MirrorDir: e.MirrorDir,
			DictDir: e.DictDir,
			DB:      db, Mirror: mv, Dict: dv, Job: s.job(e.Name),
		})
	}
	writeJSON(w, 200, map[string]any{"ok": true, "envs": out})
}

// gpDBPatch 是数据库那一节的“只改提交的字段”补丁。
// 指针字段为 nil = 本次没提交，保留旧值；非 nil 的空串/0 = 明确改成空。
type gpDBPatch struct {
	Type        *string            `json:"type,omitempty"`
	Host        *string            `json:"host,omitempty"`
	Port        *int               `json:"port,omitempty"`
	Service     *string            `json:"service,omitempty"`
	Database    *string            `json:"database,omitempty"`
	Accounts    *[]dbconfig.DBAcct `json:"accounts,omitempty"`
	ReadonlySQL *bool              `json:"readonlySql,omitempty"`
}

// gpEnvPatch 是 GP 环境的补丁。只有提交了的键才会被写回；
// 密码/账号密码为空时按 GP 表单的约定“留空=不改”，不会把已存口令抹掉。
type gpEnvPatch struct {
	Name        *string    `json:"name,omitempty"`
	Host        *string    `json:"host,omitempty"`
	Port        *int       `json:"port,omitempty"`
	User        *string    `json:"user,omitempty"`
	Password    *string    `json:"password,omitempty"`
	Instance    *string    `json:"instance,omitempty"`
	Workspace   *string    `json:"workspace,omitempty"`
	CustRoot    *string    `json:"custRoot,omitempty"`
	Company     *string    `json:"company,omitempty"`
	CompanyName *string    `json:"companyName,omitempty"`
	MirrorDir   *string    `json:"mirrorDir,omitempty"`
	DictDir     *string    `json:"dictDir,omitempty"`
	Zone        *string    `json:"zone,omitempty"`
	DB          *gpDBPatch `json:"db,omitempty"`
}

// patchFromEnv 是旧版“整条 env”请求的兼容转换：所有字段都算作提交，
// 但空密码仍按“不改”处理，避免老客户端一次保存把口令抹掉。
func patchFromEnv(e config.NamedSsh) gpEnvPatch {
	p := gpEnvPatch{
		Name: &e.Name, Host: &e.Host, Port: &e.Port, User: &e.User,
		Workspace: &e.Workspace, CustRoot: &e.CustRoot, Company: &e.Company,
		CompanyName: &e.CompanyName,
		MirrorDir:   &e.MirrorDir, DictDir: &e.DictDir, Zone: &e.Zone,
	}
	// 旧客户端的“整条 env”没有 instance；空值不能当“清空实例”处理，
	// 否则会把已有的 workspace/custRoot 一起清掉。非空才交给推导。
	if e.Instance != "" {
		p.Instance = &e.Instance
	}
	if e.Password != "" {
		p.Password = &e.Password
	}
	if e.DB != nil {
		p.DB = &gpDBPatch{
			Type: &e.DB.Type, Host: &e.DB.Host, Port: &e.DB.Port,
			Service: &e.DB.Service, Database: &e.DB.Database,
			Accounts: &e.DB.Accounts, ReadonlySQL: e.DB.ReadonlySQL,
		}
	}
	return p
}

// applyEnvPatch 把补丁合并进 dst。它只碰补丁里出现的字段，
// 因此“只改主机”不会顺带清掉 service / 密码 / dictDir / mirrorDir。
func applyEnvPatch(dst *config.NamedSsh, p gpEnvPatch) {
	if p.Name != nil {
		dst.Name = strings.TrimSpace(*p.Name)
	}
	if p.Host != nil {
		dst.Host = strings.TrimSpace(*p.Host)
	}
	if p.Port != nil {
		dst.Port = *p.Port
	}
	if p.User != nil {
		dst.User = strings.TrimSpace(*p.User)
	}
	// GP 表单：密码留空 = 不改（不是“清空”）。
	if p.Password != nil && *p.Password != "" {
		dst.Password = *p.Password
	}
	if p.Instance != nil {
		inst := strings.TrimSpace(*p.Instance)
		dst.Instance = inst
		if inst == "" {
			// 清空实例名 = 不再保留旧实例推导出来的路径，避免旧路径继续生效。
			dst.Workspace, dst.CustRoot = "", ""
		} else {
			dst.Workspace, dst.CustRoot = gpRootsForInstance(inst)
		}
	}
	// 旧客户端（或高级配置）仍可直接给路径；给了 instance 时以 instance 推导为准。
	if p.Workspace != nil && p.Instance == nil {
		dst.Workspace = strings.TrimSpace(*p.Workspace)
	}
	if p.CustRoot != nil && p.Instance == nil {
		dst.CustRoot = strings.TrimSpace(*p.CustRoot)
	}
	// 营运据点：空 = 回到 "ds"（只影响直查；本地字典库同步固定按 ds）。
	if p.Company != nil {
		dst.Company = strings.ToLower(strings.TrimSpace(*p.Company))
	}
	// 据点名字只是显示用的抄本；随据点一起清空，避免代号换了名字还留着旧的。
	if p.CompanyName != nil {
		dst.CompanyName = strings.TrimSpace(*p.CompanyName)
	}
	if p.MirrorDir != nil {
		dst.MirrorDir = strings.TrimSpace(*p.MirrorDir)
	}
	if p.DictDir != nil {
		dst.DictDir = strings.TrimSpace(*p.DictDir)
	}
	if p.Zone != nil {
		dst.Zone = strings.TrimSpace(*p.Zone)
	}
	if p.DB != nil {
		applyGPEnvDB(dst, p.DB)
	}
}

// applyGPEnvDB 合并数据库补丁。DB 不存在时新建一节，字段逐个覆盖。
func applyGPEnvDB(dst *config.NamedSsh, p *gpDBPatch) {
	if p == nil {
		return
	}
	if dst.DB == nil {
		dst.DB = &dbconfig.Connection{Type: "oracle", Port: 1521}
	}
	if p.Type != nil {
		dst.DB.Type = strings.TrimSpace(*p.Type)
	}
	if p.Host != nil {
		dst.DB.Host = strings.TrimSpace(*p.Host)
	}
	if p.Port != nil {
		dst.DB.Port = *p.Port
	}
	if p.Service != nil {
		dst.DB.Service = strings.TrimSpace(*p.Service)
	}
	if p.Database != nil {
		dst.DB.Database = strings.TrimSpace(*p.Database)
	}
	if p.ReadonlySQL != nil {
		v := *p.ReadonlySQL
		dst.DB.ReadonlySQL = &v
	}
	if p.Accounts != nil {
		mergeGPAccounts(dst.DB, *p.Accounts)
	}
	if dst.DB.Type == "" {
		dst.DB.Type = "oracle"
	}
}

// mergeGPAccounts 按账号名合并口令：
//   - 单账号 ↔ 单账号：视为同一条（允许改账号名），空密码沿用旧密码；
//   - 多账号：同账号更新、空密码保留旧密码，未提交的账号原样保留。
//
// 这样 GP 表单只编辑首个账号时，不会把配置里其余账号一起抹掉。
func mergeGPAccounts(db *dbconfig.Connection, incoming []dbconfig.DBAcct) {
	if db == nil || len(incoming) == 0 {
		return
	}
	if len(db.Accounts) == 1 && len(incoming) == 1 {
		a := incoming[0]
		a.Account = strings.TrimSpace(a.Account)
		if a.Account == "" {
			return
		}
		if a.Password == "" {
			a.Password = db.Accounts[0].Password
		}
		db.Accounts = []dbconfig.DBAcct{a}
		return
	}
	out := append([]dbconfig.DBAcct(nil), db.Accounts...)
	for _, in := range incoming {
		account := strings.TrimSpace(in.Account)
		if account == "" {
			continue
		}
		found := -1
		for i := range out {
			if out[i].Account == account {
				found = i
				break
			}
		}
		if found >= 0 {
			if in.Password == "" {
				in.Password = out[found].Password
			}
			out[found] = dbconfig.DBAcct{Account: account, Password: in.Password}
			continue
		}
		out = append(out, dbconfig.DBAcct{Account: account, Password: in.Password})
	}
	db.Accounts = out
}

// hEnv 保存/删除一个 GP 环境（写回 config.json 的 hosts.sshs）。
//
// 保存走补丁语义：请求可带 patch（只改提交字段），也兼容旧版的整条 env。
// 这样设置页的自动保存只发“这次真正变了的字段”，不会把没碰过的 service、
// 密码、dictDir、mirrorDir 清掉。
func (s *Service) hEnv(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Action  string           `json:"action"`
		Name    string           `json:"name,omitempty"`
		OldName string           `json:"oldName,omitempty"`
		Env     *config.NamedSsh `json:"env,omitempty"`
		Patch   *gpEnvPatch      `json:"patch,omitempty"`
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

		var patch gpEnvPatch
		switch {
		case body.Patch != nil:
			patch = *body.Patch
		case body.Env != nil:
			patch = patchFromEnv(*body.Env)
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
		applyEnvPatch(&env, patch)
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
			env.Instance = gpInstanceFromWorkspace(env.Workspace)
		}
		if env.Instance != "" {
			if !validGPInstance(env.Instance) {
				return fmt.Errorf("实例名 %q 不合法：只能包含字母、数字、下划线、点或连字符（如 toptest）", env.Instance)
			}
			env.Workspace, env.CustRoot = gpRootsForInstance(env.Instance)
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

// mergeTestEnv 把“表单里没填的项”用已保存值补齐，供测试连接使用。
//
// 测试连接不是保存：用户可能只改了 DB 主机、口令框留空（表示不改），
// 这时必须拿已保存的口令去测；否则每次测试都要重新输入密码，且自动保存
// 清空密码框后测试必然失败（实测）。
func mergeTestEnv(req, saved config.NamedSsh) config.NamedSsh {
	env := req
	if env.Name == "" {
		env.Name = saved.Name
	}
	if env.Password == "" {
		env.Password = saved.Password
	}
	if env.Port == 0 {
		env.Port = saved.Port
	}
	if env.Host == "" {
		env.Host = saved.Host
	}
	if env.User == "" {
		env.User = saved.User
	}
	if env.Instance == "" {
		env.Instance = saved.Instance
	}
	if env.Instance == "" {
		env.Instance = gpInstanceFromWorkspace(saved.Workspace)
	}
	if env.Workspace == "" {
		env.Workspace = saved.Workspace
	}
	if env.CustRoot == "" {
		env.CustRoot = saved.CustRoot
	}
	if env.Instance != "" {
		env.Workspace, env.CustRoot = gpRootsForInstance(env.Instance)
	}

	if env.DB == nil {
		if saved.DB == nil {
			return env
		}
		db := *saved.DB
		db.Accounts = append([]dbconfig.DBAcct(nil), saved.DB.Accounts...)
		env.DB = &db
		return env
	}

	db := *env.DB
	db.Accounts = append([]dbconfig.DBAcct(nil), env.DB.Accounts...)
	if saved.DB != nil {
		if db.Type == "" {
			db.Type = saved.DB.Type
		}
		if db.Host == "" {
			db.Host = saved.DB.Host
		}
		if db.Port == 0 {
			db.Port = saved.DB.Port
		}
		if db.Service == "" {
			db.Service = saved.DB.Service
		}
		if db.Database == "" {
			db.Database = saved.DB.Database
		}
	}
	// 账号：请求没给账号时整组沿用；给了账号但口令为空时按账号名沿用旧口令。
	if len(db.Accounts) == 0 || strings.TrimSpace(db.Accounts[0].Account) == "" {
		if saved.DB != nil {
			db.Accounts = append([]dbconfig.DBAcct(nil), saved.DB.Accounts...)
		}
	} else if db.Accounts[0].Password == "" && saved.DB != nil {
		for _, a := range saved.DB.Accounts {
			if a.Account == db.Accounts[0].Account {
				db.Accounts[0].Password = a.Password
				break
			}
		}
	}
	env.DB = &db
	return env
}

// hTest 测 SSH 通不通（数据库连通性走统一层的 /api/conntest）。
func (s *Service) hTest(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env config.NamedSsh `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
		return
	}
	envs, _ := s.gpEnvs()
	env := body.Env
	for _, e := range envs {
		if e.Name == body.Env.Name {
			env = mergeTestEnv(body.Env, e)
			break
		}
	}
	res := map[string]any{}
	conn, err := host.Dial(config.SSHConfig{Host: env.Host, Port: env.Port, User: env.User, Password: env.Password})
	if err != nil {
		res["ssh"] = map[string]any{"ok": false, "msg": err.Error()}
		writeJSON(w, 200, map[string]any{"ok": true, "result": res})
		return
	}
	defer conn.Close()
	sc, err := conn.SFTP()
	if err != nil {
		res["ssh"] = map[string]any{"ok": false, "msg": "SSH 通了，但打不开 SFTP: " + err.Error()}
		writeJSON(w, 200, map[string]any{"ok": true, "result": res})
		return
	}
	defer sc.Close()
	top := env.Workspace
	msg := "已连接"
	if top != "" {
		if st, err := sc.Stat(top); err != nil || !st.IsDir() {
			res["ssh"] = map[string]any{"ok": false, "msg": "连上了，但 $TOP 不存在：" + top}
			writeJSON(w, 200, map[string]any{"ok": true, "result": res})
			return
		}
		msg = "已连接，$TOP 可读"
	}
	res["ssh"] = map[string]any{"ok": true, "msg": msg}

	// 顺带测一下库（与 T100 的「测试连接」同一个思路：SSH 通 + 库通才算真好）。
	if env.DB != nil && env.DB.Host != "" {
		var acct, pwd string
		for _, a := range env.DB.Accounts {
			if a.Account != "" {
				acct, pwd = a.Account, a.Password
				break
			}
		}
		switch {
		case acct == "":
			res["db"] = map[string]any{"ok": false, "msg": "没有数据库账号"}
		default:
			db, derr := sql.Open("oracle", gpDSN(env.DB, acct, pwd))
			if derr != nil {
				res["db"] = map[string]any{"ok": false, "msg": derr.Error()}
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				if perr := db.PingContext(ctx); perr != nil {
					res["db"] = map[string]any{"ok": false, "msg": perr.Error()}
				} else {
					res["db"] = map[string]any{"ok": true, "msg": "已连接 " + acct + "@" + env.DB.Host + "/" + env.DB.Service}
				}
				cancel()
				db.Close()
			}
		}
	}
	writeJSON(w, 200, map[string]any{"ok": true, "result": res})
}

// hMirror 起一轮源码镜像（只拉 *.4gl）。mode=incr 增量，其它当全量。
func (s *Service) hMirror(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env  string `json:"env"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
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
	if cur := s.job(env.Name); cur.State == "running" {
		writeJSON(w, 200, map[string]any{"ok": true, "job": cur})
		return
	}
	s.setJob(env.Name, func(j *Job) {
		*j = Job{State: "running", Kind: "mirror", Started: time.Now()}
	})
	go s.runMirror(*env, body.Mode)
	writeJSON(w, 200, map[string]any{"ok": true, "job": s.job(env.Name)})
}

// hSync 起一轮字典同步（Oracle → 本地 SQLite）。
func (s *Service) hSync(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Env string `json:"env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]any{"ok": false, "error": "请求体不是 JSON"})
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
	if cur := s.job(env.Name); cur.State == "running" {
		writeJSON(w, 200, map[string]any{"ok": true, "job": cur})
		return
	}
	s.setJob(env.Name, func(j *Job) {
		*j = Job{State: "running", Kind: "sync", Started: time.Now()}
	})
	go s.runSync(*env)
	writeJSON(w, 200, map[string]any{"ok": true, "job": s.job(env.Name)})
}

func (s *Service) runMirror(env config.NamedSsh, mode string) {
	fail := func(err error) {
		s.setJob(env.Name, func(j *Job) {
			j.State = "error"
			j.Error = err.Error()
			j.Ended = time.Now()
		})
	}
	if env.Workspace == "" {
		fail(fmt.Errorf("这个环境没配 $TOP 源码根"))
		return
	}
	conn, err := host.Dial(config.SSHConfig{Host: env.Host, Port: env.Port, User: env.User, Password: env.Password})
	if err != nil {
		fail(err)
		return
	}
	defer conn.Close()
	sc, err := conn.SFTP()
	if err != nil {
		fail(err)
		return
	}
	defer sc.Close()

	base := mirrorDir(env)
	out := map[string]any{}
	total, bytes := 0, int64(0)
	doneBase, totalBase := 0, 0
	for _, src := range []struct{ label, path string }{
		{"tiptop", env.Workspace},
		{"topcust", env.CustRoot},
	} {
		if src.path == "" {
			continue
		}
		dst := filepath.Join(base, src.label)
		st, err := mirrorTree(conn, sc, src.path, dst, mode, func(done, total int, msg string) {
			s.progressCount(env.Name, doneBase+done, totalBase+total, msg)
		})
		if err != nil {
			fail(fmt.Errorf("%s: %w", src.label, err))
			return
		}
		st["src"] = env.User + "@" + env.Host + ":" + src.path
		st["dst"] = dst
		out[src.label] = st
		doneBase += st["processed"].(int)
		totalBase += st["total"].(int)
		total += st["files"].(int)
		bytes += int64(st["bytes"].(int64))
	}
	writeManifest(base, total, float64(bytes)/1048576.0, "mirror")
	s.setJob(env.Name, func(j *Job) {
		j.State = "done"
		j.Done = int64(doneBase)
		j.Total = int64(totalBase)
		j.Result = out
		j.Ended = time.Now()
	})
}

// mirrorTree 把 src 下所有 *.4gl 拉进 dst。
//
// 差异判定：大小 + mtime。mtime 在下载时按服务器写回，所以下一次能直接比 ——
// 只比大小会漏掉「长度一样但内容改了」的源码。
type mirrorFile struct {
	rpath, rrel string
	size        int64
	mt          time.Time
}

// collectByFind 用服务器侧一条 find 拿完整清单，避免逐目录 SFTP ReadDir。
// 输出用 NUL 分隔（文件名里可以有空格/换行/制表符，不能按行切）。
// find 不支持 -printf 或执行失败时返回错误，由调用方回退到 ReadDir。
func collectByFind(conn *host.SSHConn, src string) ([]mirrorFile, error) {
	q := "'" + strings.ReplaceAll(src, "'", `'\''`) + "'"
	cmd := "find " + q + ` -type f -iname '*.4gl' -printf '%P\0%s\0%T@\0' 2>/dev/null`
	out, err := conn.Output(cmd, 2*time.Minute)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts)%3 != 0 {
		return nil, fmt.Errorf("find 输出格式不完整（%d 段）", len(parts))
	}
	files := make([]mirrorFile, 0, len(parts)/3)
	for i := 0; i+2 < len(parts); i += 3 {
		rel := parts[i]
		size, err := strconv.ParseInt(parts[i+1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("解析 %q 大小失败：%w", rel, err)
		}
		sec, err := strconv.ParseFloat(parts[i+2], 64)
		if err != nil {
			return nil, fmt.Errorf("解析 %q 修改时间失败：%w", rel, err)
		}
		files = append(files, mirrorFile{
			rpath: strings.TrimSuffix(src, "/") + "/" + rel,
			rrel:  strings.TrimPrefix(rel, "/"),
			size:  size,
			mt:    time.Unix(int64(sec), 0),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rrel < files[j].rrel })
	return files, nil
}

func mirrorTree(conn *host.SSHConn, sc *sftp.Client, src, dst, mode string, progress func(done, total int, msg string)) (map[string]any, error) {
	var files []mirrorFile
	t0 := time.Now()

	files, err := collectByFind(conn, src)
	if err != nil {
		// find 不可用（BusyBox/权限/路径含不支持字符）时保留原逐目录实现。
		files = nil
		var collect func(rdir, rel string) error
		collect = func(rdir, rel string) error {
			entries, err := sc.ReadDir(rdir)
			if err != nil {
				return nil // 没权限的目录跳过，不拖垮整轮
			}
			sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
			for _, e := range entries {
				rpath := strings.TrimSuffix(rdir, "/") + "/" + e.Name()
				rrel := strings.TrimPrefix(rel+"/"+e.Name(), "/")
				if e.IsDir() {
					if err := collect(rpath, rrel); err != nil {
						return err
					}
					continue
				}
				if !strings.HasSuffix(strings.ToLower(e.Name()), ".4gl") {
					continue
				}
				files = append(files, mirrorFile{rpath: rpath, rrel: rrel, size: e.Size(), mt: e.ModTime()})
			}
			return nil
		}
		if err := collect(src, ""); err != nil {
			return nil, err
		}
	}

	total := len(files)
	if progress != nil {
		progress(0, total, "")
	}
	got, skipped, processed := 0, 0, 0
	var bytes int64
	var pruned int
	for _, f := range files {
		processed++
		lpath := filepath.Join(dst, filepath.FromSlash(f.rrel))
		if mode != "full" {
			if st, err := os.Stat(lpath); err == nil && st.Size() == f.size && st.ModTime().Unix() == f.mt.Unix() {
				skipped++
				if progress != nil && (processed%50 == 0 || processed == total) {
					progress(processed, total, fmt.Sprintf("  %d/%d 个（跳过 %d）", processed, total, skipped))
				}
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(lpath), 0o755); err != nil {
			continue
		}
		rf, err := sc.Open(f.rpath)
		if err != nil {
			continue
		}
		lf, err := os.Create(lpath)
		if err != nil {
			rf.Close()
			continue
		}
		n, cerr := io.Copy(lf, rf)
		lf.Close()
		rf.Close()
		if cerr != nil {
			continue
		}
		_ = os.Chtimes(lpath, f.mt, f.mt)
		got++
		bytes += n
		if progress != nil && (processed%50 == 0 || processed == total) {
			progress(processed, total, fmt.Sprintf("  %d/%d 个 … %.1f MB", processed, total, float64(bytes)/1048576))
		}
	}
	// 与 T100 对齐：只有全量模式清理本地残留；增量不动已有文件。
	if mode == "full" {
		n, err := pruneStale(dst, files)
		if err != nil {
			return nil, err
		}
		pruned = n
	}
	return map[string]any{
		"files": got, "skipped": skipped, "processed": processed, "total": total,
		"pruned": pruned, "bytes": bytes, "sizeMB": float64(bytes) / 1048576,
		"elapsed": time.Since(t0).Seconds(),
	}, nil
}

// pruneStale 全量模式下删除 dst 里服务器已不存在的 *.4gl，并清掉空目录。
// 只处理 .4gl，和 GP 镜像目录只收 .4gl 的范围一致；增量模式不调用。
func pruneStale(dst string, files []mirrorFile) (int, error) {
	if _, err := os.Stat(dst); err != nil {
		return 0, nil
	}
	keep := make(map[string]bool, len(files))
	for _, f := range files {
		keep[filepath.FromSlash(f.rrel)] = true
	}
	var stale []string
	var dirs []string
	err := filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 无权限的条目跳过，不拖垮整轮
		}
		if info.IsDir() {
			if path != dst {
				dirs = append(dirs, path)
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".4gl") {
			return nil
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil {
			return nil
		}
		if !keep[rel] {
			stale = append(stale, path)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	for _, p := range stale {
		_ = os.Remove(p)
	}
	// 从最深目录往回删，避免留下空壳。
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, d := range dirs {
		_ = os.Remove(d) // 非空会失败，正好保留
	}
	return len(stale), nil
}
