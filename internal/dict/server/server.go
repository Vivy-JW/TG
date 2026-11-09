// Package server 提供字典侧的 REST 接口:源码镜像拉取与字典同步。
// 只覆盖这两个长跑动作,不含任何调试能力。
//
// 它**不再对应一个独立页面** —— 合并前字典页有自己的 SPA(挂在 /dict/),本包的端点
// 也挂在 /dict/api/ 下;那个页面已并入调试工作台里的统一设置页,于是本包的端点直接挂到
// 共享层 /api/ 下(见 internal/web 的 routes)。包结构保留是因为它持有两个长跑任务的
// 状态与处理器,与"页面挂在哪"无关。
//
// SSH 连接与服务器侧数据库探测复用 internal/host(与 CLI 的 db discover 同源);
// 客户端直连测试走 internal/erpdb(与 db ping / 远程直查同链路)。
//
// 对外只暴露两个东西:构造函数 New 与 Handler()。挂载方(tt serve / internal/web)
// 把它直接挂在共享层 /api/ 下 —— Handler 的路径是绝对 /api/…,不加 StripPrefix。
//
// 合并前它靠一个挂载前缀与调试工作台分开:同一个进程里两边各有自己的 /api/*,
// 不分开就会撞名。合并后共享层只有一套 /api/*,凡是两个工具都要的端点(主机/环境的
// 读写、dbprobe/dbaccverify/conntest 探测、健康检查)由 internal/web 在顶层各提供
// 一份,本包不再重复。
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"tt/internal/config"
)

// Server 字典侧配置服务:config.json 读写 + REST 接口 + 后台任务状态。
type Server struct {
	cfgPath string // config.json 路径(读写;文件不存在时首次保存创建)
	dbDir   string // 本地字典库目录（每个环境一个 .db；由挂载方注入，空=数据目录/dict）

	// 源码镜像拉取任务(单实例:同一时间只允许一个)
	mirrorMu    sync.Mutex
	mirror      mirrorJob
	mirrorStart time.Time
	// mirrorCancel 取消正在跑的那次拉取(POST /api/mirror/stop 用)。
	// 为 nil 表示当前没有可取消的任务 —— 停止端点据此区分"没在跑"与"正在跑"。
	// 与 mirror 同一个 mirrorMu 保护。
	mirrorCancel context.CancelFunc

	// 数据库同步任务(单实例:同一时间只允许一个)
	syncMu    sync.Mutex
	sync      dbSyncJob
	syncStart time.Time
	// syncCancel 同上,对应 POST /api/dbsync/stop。
	syncCancel context.CancelFunc
}

// New 创建服务实例。cfgPath 是配置文件的读写位置。
//
// 监听地址不在这里:HTTP 服务由挂载方(tt serve / internal/web)统一建立,
// 本包只提供 Handler()。
func New(cfgPath string) *Server {
	return &Server{cfgPath: cfgPath}
}

// SetDBDir 注入本地字典库目录（每个环境一个 .db）。空 = 按配置算（见 localDBDir）。
func (s *Server) SetDBDir(dir string) { s.dbDir = dir }

// localDBDir 返回本地字典库目录。
//
// 没有注入值时**每次读配置**（顶层 dict.dir，空则 <数据目录>/dict）：库根目录是设置页
// 可改的，若只在启动时算一次，用户改完会看到「设置里改了、同步还写到老地方」——
// 而且不报错，只是文件落在别处。读一次配置是微秒级，这条路径不在热循环里。
func (s *Server) localDBDir() string {
	if s.dbDir != "" {
		return s.dbDir
	}
	return config.DictDBDirFor(s.cfgPath)
}

// dbPathFor 某环境的本地库文件（含该环境的 dictDir 覆盖）。
func (s *Server) dbPathFor(env string) string {
	if s.dbDir != "" {
		return config.DictDBPathIn(s.dbDir, env)
	}
	return config.DictDBPathForEnv(s.cfgPath, env)
}

// Handler 返回字典侧的 HTTP 面(镜像/同步/BDL 文档的 REST 接口)。
//
// 路径是**绝对**的(/api/…),由 internal/web 直接挂在共享层 /api/ 下,不经 StripPrefix。
//
// 服务的路径(全部返回 JSON):
//
//	GET    /api/mirror         镜像根 + 各环境镜像现状 + 拉取任务状态
//	POST   /api/mirror/pull    启动一次源码镜像拉取
//	POST   /api/mirror/stop    取消正在跑的拉取(用户点「停止」)
//	GET    /api/dbsync         同步目标 + 可同步环境 + 同步任务状态
//	POST   /api/dbsync         启动一次字典同步
//	POST   /api/dbsync/stop    取消正在跑的同步
//	(未知 /api/ 路径)        404 JSON(见 hUnknownAPI)
//
// 主机/环境的读写走共享的顶层 /api/hosts;dbprobe / dbaccverify / conntest、
// 健康检查、配置派生状态、PATH 安装也由 internal/web 各提供一份,故本包不注册这些端点
// (它们两边都有,挂在同一进程里会撞名)。
//
// BDL 文档目录**不再有本包的端点**:它只是 config.json 里的一节(读写走 /api/hosts),
// 「目录是否存在」由 /api/config/status 统一回答。原先那个 GET /api/bdldoc 与它们重复。
//
// 用户 PATH 安装(/api/install)也**已移到** internal/web:把 tt 加进 PATH 是应用级
// 动作,与字典查询无关。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/mirror", s.hMirrorGet)
	mux.HandleFunc("POST /api/mirror/pull", s.hMirrorPull)
	mux.HandleFunc("POST /api/mirror/stop", s.hMirrorStop)
	mux.HandleFunc("GET /api/dbsync", s.hDBSyncGet)
	mux.HandleFunc("POST /api/dbsync", s.hDBSyncPost)
	mux.HandleFunc("POST /api/dbsync/stop", s.hDBSyncStop)
	// API 路径掉到这里说明接口不存在(写错了,或者是已经移到统一层的那些,如
	// /api/install)。必须回 404 JSON 而不是落到下面的 HTML 兜底页 —— 后者会让调用方
	// 拿到 200 + HTML,解析失败时报的错与真实原因(接口不存在)完全对不上。
	// 注意 Go 1.22 起具体模式优先于前缀模式,所以上面那些具名路由不受影响。
	mux.HandleFunc("/api/", s.hUnknownAPI)
	return mux
}

// hUnknownAPI 未注册的 /api/ 路径:明确 404。
//
// 这条兜底挂在共享层 /api/ 下,所以它也会接到与字典无关的路径 —— 文案因此写得通用些。
// 关键是**不能**落到 HTML 页:那样调用方拿到 200 + HTML,解析失败时报的错与真实原因
// (接口不存在)完全对不上。
func (s *Server) hUnknownAPI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 404, map[string]any{
		"ok":    false,
		"error": "未知接口: " + r.URL.Path,
	})
}

// ---------- 工具 ----------

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, err error) {
	writeJSON(w, code, map[string]any{"ok": false, "error": err.Error()})
}

func readBody[T any](w http.ResponseWriter, r *http.Request, out *T) bool {
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		fail(w, 400, fmt.Errorf("请求体解析失败: %v", err))
		return false
	}
	return true
}
