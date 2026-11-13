package dict

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"tg/internal/config"
	"tg/internal/dict/live"
)

// 模块来源：debug 的解析链要把它写进信封，用户才知道"这次能跑"是不是碰巧碰上了线上库。
const (
	ModuleSourceLocal = "local" // 命中了本地字典（t_prog_link / t_prog）
	ModuleSourceLive  = "live"  // 本地没有，按环境连库查的 gal_file
	ModuleSourceNone  = "none"  // 两边都没有
)

// DefaultModuleLiveTimeout 兜底连库的超时。查模块失败不该把"起会话/读源码"卡死。
const DefaultModuleLiveTimeout = 8 * time.Second

// ResolveModule 解析"程序 → 实际模块"，本地优先后按环境连库兜底。
//
// 为什么要兜底：debug 的语义是"把这个作业跑起来/把它的源码找出来"，不是"查字典"；
// 本地库是缓存，昨天忘了同步不该导致今天起不了会话。所以这一条链允许落空时连一次库，
// 查到顺手回写 t_prog_link，让下次本地就能命中。tg dict 查询不走这里 —— 那条线保持
// 显式（--env 线上 / --env local: 本地），"查不到"就是查不到，不偷偷降级。
func ResolveModule(ctx context.Context, env config.NamedSsh, prog string, timeout time.Duration) (module, source string, err error) {
	prog = strings.TrimSpace(prog)
	if prog == "" {
		return "", ModuleSourceNone, nil
	}
	if m, ok := LookupModule(env, prog); ok && strings.TrimSpace(m) != "" {
		return strings.ToLower(strings.TrimSpace(m)), ModuleSourceLocal, nil
	}
	if env.DB == nil {
		return "", ModuleSourceNone, fmt.Errorf("本地字典里没有 %s，而这个环境没配数据库连接，连不上去查 gal_file", prog)
	}
	if timeout <= 0 {
		timeout = DefaultModuleLiveTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	l, err := live.OpenCompany(cctx, *env.DB, Company(env))
	if err != nil {
		return "", ModuleSourceNone, fmt.Errorf("本地字典里没有 %s，连库兜底也失败：%w", prog, err)
	}
	defer l.Close()

	// 与 progModule 同口径：只有 gal03 == gal01 那行才是入口模块；
	// 拿不到才退到 MAX(gal02)（与旧实现兼容）。模块名统一小写（线上是大写）。
	cands := []string{prog}
	if len(prog) > 4 && prog[3] == '_' {
		cands = append(cands, prog[4:])
	}
	for _, p := range cands {
		_, rows, qerr := l.Connector().Query(cctx, fmt.Sprintf(
			`SELECT COALESCE(MAX(gal02), '') FROM gal_file
			  WHERE lower(gal01) = lower('%s') AND lower(gal03) = lower(gal01)`,
			strings.ReplaceAll(p, "'", "''")))
		if qerr != nil {
			return "", ModuleSourceNone, fmt.Errorf("连上库了，但查 gal_file 失败：%w", qerr)
		}
		if mod := firstCell(rows); mod != "" {
			mod = strings.ToLower(mod)
			writeBackProgLink(env, p, mod)
			return mod, ModuleSourceLive, nil
		}
	}
	for _, p := range cands {
		_, rows, qerr := l.Connector().Query(cctx, fmt.Sprintf(
			`SELECT COALESCE(MAX(gal02), '') FROM gal_file WHERE lower(gal01) = lower('%s')`,
			strings.ReplaceAll(p, "'", "''")))
		if qerr != nil {
			return "", ModuleSourceNone, fmt.Errorf("连上库了，但查 gal_file 失败：%w", qerr)
		}
		if mod := firstCell(rows); mod != "" {
			mod = strings.ToLower(mod)
			writeBackProgLink(env, p, mod)
			return mod, ModuleSourceLive, nil
		}
	}
	return "", ModuleSourceNone, nil
}

func firstCell(rows [][]string) string {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(rows[0][0]))
}

// writeBackProgLink 把线上查到的入口模块写回本地字典（尽力而为，失败不影响本次结果）。
// 只动 t_prog_link 这张小表 —— debug 路径上不跑同步。
func writeBackProgLink(env config.NamedSsh, prog, module string) {
	path := dictPath(env)
	if _, err := os.Stat(path); err != nil {
		return
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return
	}
	defer db.Close()
	p := strings.ToLower(strings.TrimSpace(prog))
	m := strings.ToLower(strings.TrimSpace(module))
	if p == "" || m == "" {
		return
	}
	_, _ = db.Exec(
		`INSERT OR REPLACE INTO t_prog_link(prog, module, linked_prog, link_flag) VALUES (?, ?, ?, '')`,
		p, m, p)
}
