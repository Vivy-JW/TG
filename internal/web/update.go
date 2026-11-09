package web

// tt 自身的更新：设置页上的「检查更新」与「立即更新」。
//
// 与 handle 安装（internal/web/install.go）同一层：这三个端点挂在**统一层** /api/ 下，
// 不是某个子系统的私有 API —— 更新 tt 是应用级动作，与字典/调试都无关。
//
// 三条纪律（与命令行那一侧同一份，见 internal/cli/update.go）：
//
//   - **检查只由人点触发**。服务启动不查、也没有定时器；页面上的按钮就是那个"显式命令"。
//   - **装要人点头**。按钮点下去就是那个头，所以它等价于 `tt update --yes`。
//   - 检查结果落在缓存里（<数据目录>/update/check.json），页面与 `tt version` 离线读它。

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"tt/internal/config"
	"tt/internal/update"
	"tt/internal/winproc"
)

// hUpdateGet 返回更新相关的全部状态：当前版本、形态、上一次检查结论、上一次升级结果、
// 技能是否与二进制同版。**不联网** —— 页面上那行"有新版本"来自缓存。
func (s *Server) hUpdateGet(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	dataDir := filepath.Dir(cfgPath)
	exe, _ := os.Executable()
	kind := update.DetectKind(exe)

	out := map[string]any{
		"ok":          true,
		"version":     s.opt.Version, // 展示用（带 commit）；比较用 BareVersion
		"kind":        kind.String(),
		"canInstall":  update.InstallRefusal(kind, exe, s.opt.BareVersion) == "",
		"source":      update.RepoSlug,
		"skillsDrift": update.SkillsDriftHint(dataDir, s.opt.BareVersion),
	}
	if why := update.InstallRefusal(kind, exe, s.opt.BareVersion); why != "" {
		out["refusal"] = why
	}
	if res := update.LoadCachedCheck(dataDir, s.opt.BareVersion, time.Now()); res != nil {
		out["check"] = res
		out["hint"] = res.Message()
	}
	if st := update.LoadState(dataDir); st != nil {
		out["last"] = st
	}
	writeJSON(w, http.StatusOK, out)
}

// hUpdateCheck POST /api/update/check：人去点一次按钮 → 联网查一次 → 落缓存 → 回结果。
func (s *Server) hUpdateCheck(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	dataDir := filepath.Dir(cfgPath)
	client, err := s.updateClient(cfgPath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	res, err := update.Check(ctx, client, s.opt.BareVersion, false)
	if err != nil {
		// 出网失败不是服务端的错，所以用 200 + ok:false 让页面按"检查没成"展示，
		// 而不是当成 API 坏了。
		writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error(), "where": client.Where()})
		return
	}
	_ = update.SaveCheck(dataDir, res)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "check": res, "hint": res.Message(), "where": client.Where(),
	})
}

// hUpdateApply POST /api/update/apply：交棒给更新器。
//
// 为什么走子进程而不是直接调函数：这条流程的实现住在命令层（internal/cli），而命令层
// import 本包 —— 反过来 import 就成环。拉一个 `tt update --yes` 还顺带保证"页面点的"
// 与"人敲的"是同一条路径（包括确认、下载、校验、交棒）。
//
// 这个进程（服务自己）会在几分钟后被更新器停掉：它在安装目录里跑，锁着要被替换的文件。
// 所以这里立刻返回，页面上提示"更新器已接手"。
func (s *Server) hUpdateApply(w http.ResponseWriter, r *http.Request) {
	cfgPath, err := s.configPath(true)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if why := update.InstallRefusal(update.DetectKind(exe), exe, s.opt.BareVersion); why != "" {
		writeErr(w, http.StatusBadRequest, errFromString(why))
		return
	}
	// --yes：按钮点下去就是那个"头"。--config：交给同一个数据目录。
	// 日志落数据目录的 .tt-update.log（与命令行那条路一致，`tt update log` 读得到）。
	pid, err := winproc.SpawnDetached(exe,
		[]string{"update", "--yes", "--config", cfgPath},
		update.LogPath(filepath.Dir(cfgPath)))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "pid": pid, "handedOff": true,
		"note": "更新器已接手：它会在下载校验完成后停掉本服务并替换文件。进度看 tt update log。",
	})
}

// updateClient 按配置里的 net.proxy 造一个客户端（页面没有 --proxy 参数位）。
func (s *Server) updateClient(cfgPath string) (*update.Client, error) {
	proxy := ""
	if root, err := config.Load(cfgPath); err == nil && root != nil {
		proxy = root.Net.Proxy
	}
	agent := "tt-update/" + s.opt.BareVersion
	return update.NewClient(update.ExplicitProxy("", proxy), agent)
}

// errFromString 把一句已经写好的中文原因变成 error（writeErr 只认 error）。
func errFromString(msg string) error { return &simpleErr{msg} }

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }
