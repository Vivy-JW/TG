package web

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// fsEntry 目录选择器里的一行。
type fsEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// fsListResp GET /api/fs/list 的响应。
//
// 只列**目录**，不列文件 —— 这个端点的唯一用途是给"选一个目录"配路径，
// 把文件也塞进来只会让用户多点几下、也让响应大一个数量级。
type fsListResp struct {
	OK     bool      `json:"ok"`
	Path   string    `json:"path"`   // 当前目录；空 = 还没进去（只给了盘符）
	Parent string    `json:"parent"` // 上级目录；空 = 已经是根
	Drives []fsEntry `json:"drives"` // 本机盘符（Windows 是 C:\ 这类；其他系统是 /）
	Dirs   []fsEntry `json:"dirs"`
	// Truncated 目录项太多被截断（超大目录不阻塞界面，用户可继续输入路径进入）。
	Truncated bool `json:"truncated,omitempty"`
}

// fsListLimit 一次最多列多少个子目录。Windows 的 node_modules / 系统目录能有几万项，
// 全量返回会让浏览器渲染卡住，而选择目录根本不需要看全。
const fsListLimit = 800

// hFsList 列出本机某个目录下的子目录（只读、只列目录、只接受本机请求）。
//
// 为什么要有它：浏览器**拿不到本地绝对路径** —— `<input webkitdirectory>` 只给相对路径，
// `showDirectoryPicker()` 返回的是 FileSystemDirectoryHandle，同样不暴露 "D:\..."。
// 所以"选目录"只能由服务端列出来、前端画成自己的选择器；
// 原来那种"服务端弹 Windows 对话框"虽然能拿到路径，但每次都要起 PowerShell（慢）。
// 这个端点让选择器变成纯 HTTP：几十毫秒，且外观跟设置页一致。
func (s *Server) hFsList(w http.ResponseWriter, r *http.Request) {
	if !isLoopback(r.RemoteAddr) || !hostIsLocal(r) {
		writeErr(w, http.StatusForbidden, fmt.Errorf("/api/fs/list 只接受本机请求"))
		return
	}
	raw := strings.TrimSpace(r.URL.Query().Get("path"))
	// Dirs/Drives 都显式给空切片：nil 会序列化成 null，前端 `data.dirs.length` 直接抛异常
	// （只列盘符那一屏必然走到这里）。
	resp := fsListResp{OK: true, Drives: drivesOf(), Dirs: []fsEntry{}}
	if raw == "" {
		writeJSON(w, http.StatusOK, resp)
		return
	}

	p := filepath.Clean(normalizeDrivePath(strings.TrimSpace(raw)))
	st, err := os.Stat(p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("目录不存在或无法访问: %s", p))
		return
	}
	if !st.IsDir() {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("不是目录: %s", p))
		return
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, fmt.Errorf("无法读取目录: %v", err))
		return
	}
	dirs := make([]fsEntry, 0, 32)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if len(dirs) >= fsListLimit {
			resp.Truncated = true
			break
		}
		name := e.Name()
		dirs = append(dirs, fsEntry{Name: name, Path: filepath.Join(p, name)})
	}
	sort.SliceStable(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})

	resp.Path = p
	if parent := filepath.Dir(p); parent != p {
		resp.Parent = parent
	}
	resp.Dirs = dirs
	writeJSON(w, http.StatusOK, resp)
}

// hostIsLocal 判断请求的 Host 头是否指向本机。
//
// 只查 RemoteAddr 挡不住 DNS rebinding：攻击者的域名解析到 127.0.0.1 之后，浏览器发出的
// 请求确实来自 127.0.0.1（RemoteAddr 过得了那道校验），但 Host 是 evil.com —— 同源策略
// 因此把响应归给 evil.com，攻击者能直接读到本机目录名。所以 Host 必须是回环地址或
// localhost。局域网用 IP 访问的请求在这步之前已经被 RemoteAddr 挡掉，不会误伤。
func hostIsLocal(r *http.Request) bool {
	h := strings.ToLower(r.Host)
	if h == "" {
		return false
	}
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	// IPv6 的方括号、DNS 名的结尾点都得先去掉再比对。
	h = strings.Trim(strings.TrimSuffix(h, "."), "[]")
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

// normalizeDrivePath 把 "D" / "D:" / "D:." / "D:\" 都补成盘根 "D:\"。
//
// 为什么这四种都要认：不补的话 `os.ReadDir("D:")` 读的是"该盘当前工作目录"
// （Windows 的经典怪癖），用户看到的目录会跟他以为的不一样。
// 而 **`filepath.Clean("D:")` 返回的是 `"D:."`** —— 所以调用方若先 Clean 再进这里，
// 只认 len 1/2 的实现会漏掉它（实测踩过）。这里四种形态一起收，且在 handler 里
// 先 normalize 再 Clean，两头都堵住。
func normalizeDrivePath(p string) string {
	if runtime.GOOS != "windows" || len(p) == 0 || len(p) > 3 {
		return p
	}
	c := p[0]
	if c >= 'a' && c <= 'z' {
		c -= 'a' - 'A'
	}
	if c < 'A' || c > 'Z' {
		return p
	}
	switch p[1:] {
	case "", ":", `:\`, ":.", `:/`:
		return string(c) + `:\`
	}
	return p
}

// drivesOf 枚举本机盘符。Windows 逐个 A–Z 试 Stat（比调 Win32 API 简单且够快）；
// 其他系统只有一个根。
func drivesOf() []fsEntry {
	if runtime.GOOS != "windows" {
		return []fsEntry{{Name: "/", Path: "/"}}
	}
	out := make([]fsEntry, 0, 8)
	for c := 'A'; c <= 'Z'; c++ {
		p := string(c) + `:\`
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			out = append(out, fsEntry{Name: string(c) + ":", Path: p})
		}
	}
	return out
}
