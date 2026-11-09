package web

import (
	"net/http"

	"tt/internal/pathinstall"
)

// ---------- 命令行安装(把可执行文件目录加入用户 PATH) ----------
//
// 实现在 pathinstall 包(与 CLI 的 `tt install path` 共用同一份)。
//
// 这几个端点在**统一层**而不是字典子系统里:把 tt 加进 PATH 是应用级的动作,
// 与"字典查询"没有关系。合并前它挂在 /dict/api/install 下,于是一个共用设置页
// 要用到它就得去调另一个子系统的私有 API —— 那正是这次要拆掉的耦合。

// hInstallGet 返回安装状态:可执行文件/目录、是否已在用户 PATH。
func (s *Server) hInstallGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, pathinstall.Get())
}

// hInstallAdd 把当前可执行文件所在目录加入用户 PATH(幂等,用户级无需管理员)。
func (s *Server) hInstallAdd(w http.ResponseWriter, r *http.Request) {
	st, err := pathinstall.Add()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// hInstallRemove 从用户 PATH 移除该目录(幂等)。
func (s *Server) hInstallRemove(w http.ResponseWriter, r *http.Request) {
	st, err := pathinstall.Remove()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}
