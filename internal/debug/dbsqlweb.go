package debug

// `/api/dbsql` 的**直连优先**路径：在服务进程内复用一条直连会话。
//
// 这是"常驻"真正生效的地方 —— CLI 一条命令一个进程，把连接留在进程里没有意义
// （批量已经把固定成本摊成一次）；而 tt serve 活得很久，每个请求重新拨号+登录
// 是纯浪费（实测每条 1.3 s，其中 ~1.05 s 是 sqlplus 启动，~55 ms 是登录）。
//
// 三条出口刻意分开：
//   - handled=false：**这条路不该回答这次请求**（没挂库、拿不到企业账号、建连不可达）
//     → 调用方继续走原来的 SSH + sqlplus，兼容"客户端到不了库"的站点；
//   - handled=true + err：查询本身的失败（ORA-xxxx / 只读闸门）→ 原样报错，不回退
//     （同一条语句换条路是同一个错，重跑还要再付一次超时）；
//   - handled=true + err=nil：出结果。

import (
	"context"
	"strconv"
	"time"

	"tt/internal/dbconfig"
)

// directSQL 尝试走常驻直连；第二个返回值只在 handled=true 时有意义。
func (s *Server) directSQL(ctx context.Context, ent, timeout int, sqlText string) (*DBQueryResult, error, bool) {
	if s.dbPool == nil || s.cfg.DB == nil || !s.cfg.DB.ReadonlySQLEnabled() {
		return nil, nil, false
	}
	cc, err := s.cfg.DBConnForEnt(ent)
	if err != nil {
		return nil, nil, false
	}
	key := s.cfg.EnvName() + "|" + strconv.Itoa(ent) + "|" + cc.Account
	q, release, err := s.dbPool.acquire(key, func() (*DirectQuery, error) {
		octx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		return OpenDirectQuery(octx, *cc)
	})
	if err != nil {
		// 建不起来（网络 / 隧道 / 登录）：交回老路，不让"本机到不了库"的站点因此查不了数。
		return nil, nil, false
	}
	// release 必须在查询结束后调用：它才是"这条连接现在没人用了"的凭据
	// （退役/配置热替换要等它才关，见 dbpool.go）。
	defer release()

	qctx, cancel := context.WithTimeout(ctx, time.Duration(clampSQLTimeout(timeout))*time.Second)
	defer cancel()
	res, err := q.Query(qctx, sqlText)
	// 成败都交给池判断：SQL 写错不该把别人正用的连接抽走，只有像是连接断了才退役。
	s.dbPool.report(key, q, err)
	if err != nil {
		return nil, err, true
	}
	res.Env = s.cfg.EnvName()
	res.Zone = s.cfg.Zone
	res.Ent = ent
	res.Account = cc.Account
	res.AccountSource = dbconfig.AcctFromEntConfig
	res.DBTarget = cc.Address()
	res.Route = dbconfig.RouteClientDirect
	res.ViaTunnel = cc.ViaSSH != nil
	return res, nil, true
}
