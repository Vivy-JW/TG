package db

// sqlSession —— 一条**进程内可复用**的只读 SQL 会话：建连一次，跑 N 条。
//
// 单条（tt db sql "<sql>"）与批量（--batch）都走它，所以"固定成本只付一次"这件事
// 只有一处实现。两条路线共用同一个类型，调用方不关心走的是哪条：
//
//   - direct：一条 erpdb 直连，TCP + Oracle 登录只付一次（实测 ~55ms/次 → 一次）；
//   - server：一条 SSH + 每句一次 sqlplus（拿不到直连时的兜底，sqlplus 的 ~1s/句仍在）。
//
// 只读语义不因为复用而削弱：每条语句照样过 safesql.Check，库会话照样是只读事务。

import (
	"context"
	"errors"
	"fmt"

	"tt/internal/dbconfig"
	"tt/internal/debug"
	"tt/internal/host"
	"tt/internal/output"
)

// sqlSession 一次执行要用的连接。direct 与 ssh 恰有一个非空。
type sqlSession struct {
	route   string // 直连时是 client-direct；服务器路线要等结果里的方言才能定
	ent     int
	account string
	target  string
	viaTun  bool

	direct  *debug.DirectQuery
	ssh     *host.SSHConn
	mgr     *debug.Manager
	dialect string
	acctSrc string
}

// openSession 建一次连。fallbackNote 非空表示直连没建起来、已回退服务器路线。
//
// 错误一律带稳定码：身份错 TARGET/2、直连连不上 DB_UNREACHABLE/4、
// SSH 连不上 SSH_FAILED/4、其余 DB_FAILED/4 —— 与单条命令原有的退出码一一对应。
func openSession(cfg *debug.Config, ent int, route string) (*sqlSession, string, error) {
	fallbackNote := ""
	if route != SQLRouteServer {
		cc, err := dbConnFor(cfg, ent)
		if err != nil {
			return nil, "", output.Errorf("TARGET", 2, "%v", err)
		}
		d, err := debug.OpenDirectQuery(context.Background(), *cc)
		if err == nil {
			return &sqlSession{
				route: dbconfig.RouteClientDirect, ent: ent, account: cc.Account,
				target: cc.Address(), viaTun: cc.ViaSSH != nil,
				direct: d, dialect: d.Dialect(),
			}, "", nil
		}
		// 只有“直连建不起来”才允许回退：查询本身的失败回退没有意义
		// （同一条语句在另一条路上是同一个错误，而重查询还要再付一次超时）。
		var unreachable *debug.DirectUnreachableError
		if route == SQLRouteDirect || !errors.As(err, &unreachable) {
			code := "DB_FAILED"
			if errors.As(err, &unreachable) {
				code = "DB_UNREACHABLE"
			}
			return nil, "", output.Errorf(code, 4, "%v", err)
		}
		fallbackNote = fmt.Sprintf("%v；已回退服务器 sqlplus 路线", err)
	}

	conn, err := host.Dial(cfg.SSH)
	if err != nil {
		return nil, "", output.Errorf("SSH_FAILED", 4, "SSH 连接失败: %v", err)
	}
	return &sqlSession{ent: ent, ssh: conn, mgr: debug.NewManager(cfg)}, fallbackNote, nil
}

// Query 在这条会话上跑一条只读 SQL。
func (s *sqlSession) Query(ctx context.Context, sqlText string, timeout int) (*debug.DBQueryResult, error) {
	if s.direct != nil {
		return s.direct.Query(ctx, sqlText)
	}
	res, err := s.mgr.RunReadonlyQuery(s.ssh, debug.ReadonlyQueryReq{SQL: sqlText, Ent: s.ent, Timeout: timeout})
	if err != nil {
		return nil, err
	}
	s.dialect = res.Dialect
	if s.account == "" {
		s.account, s.acctSrc = res.Account, res.AccountSource
	}
	if s.route == "" {
		s.route = serverRoute(res.Dialect)
	}
	return res, nil
}

// Close 释放连接（直连与 SSH 都是幂等关闭）。
func (s *sqlSession) Close() {
	if s == nil {
		return
	}
	if s.direct != nil {
		s.direct.Close()
	}
	if s.ssh != nil {
		_ = s.ssh.Close()
	}
}

// acctSource 信封里的账号来源：直连是按 ENT 清单定的，服务器路线由 debug 侧解析。
func (s *sqlSession) acctSource() string {
	if s.direct != nil {
		return dbconfig.AcctFromEntConfig
	}
	return s.acctSrc
}

// routeName 直连取会话路线；服务器路线在第一次查询后才由方言定下来。
func (s *sqlSession) routeName() string {
	if s.route != "" {
		return s.route
	}
	if s.ssh != nil {
		return serverRoute(s.dialect)
	}
	return ""
}

// applySessionMeta 把"这次落在哪"填进结果信封。
//
// 直连与服务器路线的账号来源不同（前者按 ENT 清单直连，后者由 debug 侧解析），
// 所以只在直连时覆盖 Account/AccountSource，避免把服务器路线已解析的值盖成空。
func applySessionMeta(res *debug.DBQueryResult, cfg *debug.Config, s *sqlSession, note string) {
	res.Env = cfg.EnvName()
	res.Zone = cfg.Zone
	res.Ent = s.ent
	if s.direct != nil {
		res.Account = s.account
		res.AccountSource = dbconfig.AcctFromEntConfig
		res.DBTarget = s.target
		res.Route = dbconfig.RouteClientDirect
		res.ViaTunnel = s.viaTun
	} else {
		res.Route = serverRoute(res.Dialect)
	}
	if note != "" {
		res.Notes = append(res.Notes, note)
	}
}
