package web

import (
	"net/http"

	"tt/internal/config"
	"tt/internal/pathinstall"
)

// configStatus 是"配置里那些路径型取值"的派生状态:值本身在 /api/hosts 里,
// 这里补上**只在服务端才能算出来的部分** —— 目录/文件是否真的存在、生效值是哪个。
//
// 为什么单独一个端点而不是塞进 /api/hosts:这些判断要 os.Stat,而 /api/hosts 是
// 设置页每次进入设置都会读的热路径;把它们掺进去等于每次读配置都摸一遍磁盘。
//
// 为什么放在共享层而不是字典子系统的私有 API:统一设置页的「数据字典」分区要显示这些
// 状态,而设置页是**共用**的 —— 让它去调 /dict/api/* 会把它绑死在字典子系统上
// (路由本身支持只有调试子系统的部署,那时数据字典卡片就整个打不开了)。
type configStatus struct {
	OK bool `json:"ok"`
	// Config 配置文件绝对路径。
	Config string `json:"config"`
	// Mirror 源码镜像根目录。
	Mirror config.DirStatus `json:"mirror"`
	// Dict 本地字典库根目录（空 = 缺省 <数据目录>/dict）。
	Dict config.DirStatus `json:"dict"`
	// Sync 字典同步的本地 SQLite。
	Sync config.FileStatus `json:"sync"`
	// Install 命令行安装状态(用户 PATH),不是 config.json 里的东西。
	Install pathinstall.Status `json:"install"`
	// Tzs .tzs 表单引擎的运行时依赖:引擎 exe(随包分发)、设计器目录与工作区(用户配的)。
	Tzs config.TzsStatus `json:"tzs"`
	// Cache 配置目录下那些**可再生**中间数据的占用(企业快照/源码镜像/查询落盘/断点存档)。
	// 与 config.json 同目录但性质相反,所以单独一个块,并明确"它不是你的数据"。
	Cache config.CacheStatus `json:"cache"`
}

// hConfigStatus 返回上述派生状态。配置读不到时不报错,返回缺省值 —— 设置页在没有
// 配置文件的首次运行下也要能渲染。
func (s *Server) hConfigStatus(w http.ResponseWriter, r *http.Request) {
	path, _ := s.configPath(true)
	st := configStatus{OK: true, Config: path, Sync: config.FileStatusOf("", s.syncDefaultTarget())}

	var tzsCfg config.TzsSettings
	root, err := config.Load(path)
	if err == nil {
		st.Mirror = config.DirStatusOf(root.Mirror.Dir)
		st.Dict = config.DirStatusOf(root.Dict.Dir)
		st.Sync = config.FileStatusOf(root.Sync.Target, s.syncDefaultTarget())
		tzsCfg = root.Tzs
		// 工作区**按环境配**,而"默认环境"已移除(环境一律点名)——
		// 所以这里没有一个"当前工作区"可报。工作区状态改为按环境逐个呈现,
		// 这个汇总端点不再替用户挑一个环境(挑错等于报别人的目录状态)。
	}
	// 引擎 exe 的状态与配置读没读到无关(它随包分发),所以放在分支外。
	// 第三个参数是"要检查哪个工作区":没有默认环境 ⇒ 传空 ⇒ 该项状态为"未指定"
	// (见 TzsStatusOf 对空值的处理)。
	st.Tzs = config.TzsStatusOf(tzsCfg, "", config.EngineExe(""))
	st.Install = pathinstall.Get()
	// 缓存就在数据目录下,配置读不到也算得出来(缓存与配置文件无关)。
	st.Cache = config.CacheStatusOf(config.LocationsAt(path).DataDir())
	writeJSON(w, http.StatusOK, st)
}

// hCacheClear 清理缓存(设置页那个「清除缓存」按钮走的端点)。
//
// **只删缓存**:config.json 与运行中的服务状态文件一律不动 —— 哪些算缓存由
// internal/config/cache.go 定义一次,这里不自己列目录,免得两处各列一份、某天漂了。
func (s *Server) hCacheClear(w http.ResponseWriter, r *http.Request) {
	path, err := s.configPath(true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	dir := config.LocationsAt(path).DataDir()
	removed, freed, err := config.CleanCache(dir, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	// 连同清完之后的状态一起回:前端不必再打一次 status 才能刷新数字
	writeJSON(w, http.StatusOK, map[string]any{
		"removed": removed, "freed": freed, "cache": config.CacheStatusOf(dir),
	})
}

// syncDefaultTarget 同步目标的缺省位置。由 Options 注入以便与字典子系统取同一个值
// (两边各算一次的话,设置页显示的"默认位置"可能和字典子系统实际写入的不是同一个);
// 未注入时统一走 config.Locations(数据目录下的 erp_data.db)。
func (s *Server) syncDefaultTarget() string {
	if s.opt.SyncDefaultTarget != "" {
		return s.opt.SyncDefaultTarget
	}
	path, _ := s.configPath(true)
	return config.LocationsAt(path).SyncTarget("", "")
}
