package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// pickDirMu 防止连点弹出多个原生对话框；同一时间只允许一个。
var pickDirMu sync.Mutex

// hPickDir 让服务端在本机弹出系统“选择文件夹”对话框，返回用户选中的绝对路径。
//
// 为什么不让浏览器自己选：<input type="file" webkitdirectory> 只给 File 对象，
// 拿不到绝对路径；File System Access API 的 showDirectoryPicker 也只给 handle。
// TT 的镜像目录要让 CLI 也能用，必须是真实路径，所以只能由服务端起这个对话框。
//
// 安全边界：这个对话框会阻塞在服务端进程里，且能读到用户文件系统的路径，
// 因此和 /api/shutdown 一样只接受本机回环请求。服务默认就绑 127.0.0.1。
func (s *Server) hPickDir(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("/api/pickdir 只接受本机请求"))
		return
	}
	var req struct {
		Initial string `json:"initial"`
		Title   string `json:"title"`
	}
	if r.Body != nil {
		_ = decodeBody(r, &req)
	}
	if !pickDirMu.TryLock() {
		writeErr(w, http.StatusConflict, fmt.Errorf("文件夹选择器已经打开，请在系统窗口里选完或取消"))
		return
	}
	defer pickDirMu.Unlock()

	path, canceled, err := pickDirectory(r.Context(), req.Initial, req.Title)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "canceled": true})
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if canceled || path == "" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "canceled": true})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": path})
}
