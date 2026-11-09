package dict

// 查询数据源(r.t/r.v/desc/scc/r.q 共用):本地 SQLite 镜像(*db.DB)与远程 ERP 库
// (*live.Live,金仓/Oracle)实现同一 db.Source 接口,数据源由配置/CLI 决定,
// 命令层不感知差异 —— 不再有 --online 开关。
//
// 选择优先级:
//  1. --env/--conn <环境名|local:<环境>>（覆盖本次调用；**显式指定即强制**）；
//  2. config.json 顶层 query.source（环境名 或 local:<环境>；可在 tt serve 的「设置」页切换）；
//  3. 都没有 → 报错（没有默认环境，也不猜本地库是哪个环境的）。
//
// **明确不做自动降级**:在线就是在线、本地就是本地,连不上直接报错并提示怎么切
// (--env local:<环境>,或前端「设置」页把「查询数据源」切成本地)。
//
// "local:<环境>" = <数据目录>/dict/<环境>.db（-d/--db 可显式覆盖）;环境名 = hosts.sshs 中该环境的
// db(客户端直连,凭据取账号列表首项)。

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dbconfig"
	"tt/internal/dict/db"
	"tt/internal/dict/live"
	"tt/internal/output"
)

var (
	dataSrc  db.Source
	srcLocal bool // 当前源为本地 SQLite(错误提示文案分流)

	// srcTarget 当前数据源的"落脚点"(环境/账号/库/路径)—— openQuerySource 时填好,
	// 由查询命令打进输出信封。它是"我这次到底查的是哪儿"的唯一答案。
	srcTarget *dbconfig.Target
	// srcNotes 打开数据源时产生的降级提示(如企业账号解析失败改用了列表首项)。
	srcNotes []string
	// srcDetail 本次查询的 detail 契约（summary | full）；空 = 不区分。
	// 由命令在 RunE 里设置，srcMeta 会带进信封。
	srcDetail string
	// srcStarted 数据源打开的时刻,用来算信封里的"用时" —— 它含连接与环境解析,
	// 不只是查询本身,那正是"这条命令为什么慢"要看的数。
	srcStarted time.Time
)

// openQuerySource 在 Group PersistentPreRunE 打开数据源。
//
// **不再有"默认环境"这一层回落**(hosts.activeEnv 已移除):要么命令行点名(--env),
// 要么配置里显式写了 query.source,要么就查本地副本。原来那条"未配置时用默认环境的
// 远程库直查"会在一台机器配了多个环境时替用户挑一个 —— 挑错的表现与"数据不存在"一样。
func openQuerySource() error {
	srcStarted = time.Now()
	if env := strings.TrimSpace(common.Env); env != "" { // ① 命令行指定(--env/--conn):强制
		if name, ok := localEnvName(env); ok {
			return openLocalSource(name)
		}
		if strings.EqualFold(env, "local") {
			if explicitLocalDB() != "" {
				return openLocalSource("")
			}
			return localNeedsEnvErr()
		}
		return openRemoteSource(env, dictEnt)
	}
	switch cfg := strings.TrimSpace(queryCfgSource()); {
	case strings.HasPrefix(strings.ToLower(cfg), "local:"): // ② 配置指定 local:<环境>
		return openLocalSource(strings.TrimSpace(cfg[len("local:"):]))
	case strings.EqualFold(cfg, "local"): // 旧写法：没有环境，不能猜
		return localNeedsEnvErr()
	case cfg == "" || strings.EqualFold(cfg, "auto"): // ③ 未配置：没有默认环境，不能猜
		return noSourceErr()
	default: // ② 配置指定了环境名:远程
		return openRemoteSource(cfg, dictEnt)
	}
}

// localEnvName 解析 local:<环境>；第二个返回值表示它是不是本地源写法。
func localEnvName(s string) (string, bool) {
	const prefix = "local:"
	if len(s) < len(prefix) || !strings.EqualFold(s[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(s[len(prefix):]), true
}

// explicitLocalDB 显式给的本地库路径（TDICT_DB / -d/--db）；空 = 没给。
func explicitLocalDB() string {
	if v := strings.TrimSpace(os.Getenv("TDICT_DB")); v != "" {
		return v
	}
	return strings.TrimSpace(dbPath)
}

// localNeedsEnvErr 裸 local（没有环境）时的报错：列出可选的 local:<环境>，不猜。
func localNeedsEnvErr() error {
	names := localDBEnvNames()
	hint := `本地库按环境分开，必须点名环境：
  tt dict <命令> --env local:<环境>
（首次使用先同步： tt dict db sync --env <环境>）`
	if len(names) > 0 {
		hint = hint + `

已有本地库：` + strings.Join(names, "、")
	}
	return output.Errorf(output.CodeUsage, 2, "数据源 local 缺少环境：%s", hint)
}

// noSourceErr 没给 --env、配置也没有 query.source 时的报错。
func noSourceErr() error {
	return output.Errorf(output.CodeUsage, 2, `没有点名数据源：用 --at <环境> 或 --env <环境> 走在线；
本地用 --env local:<环境>。也可以在设置页把 query.source 固定为某个环境或 local:<环境>。`)
}

// localDBEnvNames 列出已经有本地库文件的环境（local:<环境> 形式），用于报错提示。
func localDBEnvNames() []string {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return nil
	}
	hosts, err := config.LoadHosts(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, name := range hosts.Names() {
		if _, err := os.Stat(config.DictDBPathForEnv(path, name)); err == nil {
			out = append(out, "local:"+name)
		}
	}
	return out
}

// resolveDBPathForEnv 按环境定位本地库（TDICT_DB / -d / <数据目录>/dict/<环境>.db）。
func resolveDBPathForEnv(env, flagPath string) (string, error) {
	p, tried, err := config.LocationsAt(srcConfigPath).ResolveDictDBFor(env, flagPath)
	if err != nil {
		var msg strings.Builder
		fmt.Fprintln(&msg, "本地字典库不存在（环境 "+env+"）。")
		fmt.Fprintln(&msg, "尝试过:")
		for _, one := range tried {
			fmt.Fprintln(&msg, "  "+one)
		}
		fmt.Fprintln(&msg)
		fmt.Fprint(&msg, "先同步该环境: tt dict db sync --env "+env)
		return "", fmt.Errorf("%s", msg.String())
	}
	return p, nil
}

// closeQuerySource 在 Group PersistentPostRun 关闭数据源。
func closeQuerySource() {
	if dataSrc != nil {
		dataSrc.Close()
		dataSrc = nil
	}
}

// queryCfgSource 读 config.json 顶层 query.source("local" / 环境名 / "auto");
// 缺失、"auto" 或读不到配置时返回空串/auto(即走缺省策略:在线优先)。
func queryCfgSource() string {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return ""
	}
	root, err := config.Load(path)
	if err != nil {
		return ""
	}
	return root.Query.Source
}

// openLocalSource 打开本地 SQLite 镜像。
//
// env 非空：<数据目录>/dict/<env>.db（-d/TDICT_DB 可覆盖）；env 为空：只允许显式
// -d/TDICT_DB（裸 local 不能猜环境）。
func openLocalSource(env string) error {
	var (
		resolvedPath string
		err          error
	)
	if env == "" {
		if explicitLocalDB() == "" {
			return localNeedsEnvErr()
		}
		resolvedPath, err = resolveDBPath(dbPath)
	} else {
		resolvedPath, err = resolveDBPathForEnv(env, dbPath)
	}
	if err != nil {
		return err
	}
	where := env
	if where == "" {
		where = "local"
	}
	srcTarget = &dbconfig.Target{Env: where, Route: dbconfig.RouteLocalSQLite, Path: resolvedPath}
	srcNotes = nil
	if common.Verbose {
		fmt.Fprintf(os.Stderr, "[dict] 数据源: 本地 SQLite (%s) [%s]", resolvedPath, where)
		fmt.Fprintln(os.Stderr)
	}
	database, err := db.Open(resolvedPath)
	if err != nil {
		return fmt.Errorf("打开本地数据库 %s 失败: %w", resolvedPath, err)
	}
	dataSrc = database
	srcLocal = true
	return nil
}

// openRemoteSource 打开指定 SSH 环境的远程库(客户端直连)。
//
// entRaw = 用户显式给的企业编号(--ent);空 = 不按企业解析账号,用账号列表首项。
func openRemoteSource(target, entRaw string) error {
	path, err := common.ResolveConfig(false)
	if err != nil {
		return err
	}
	cfg, err := config.LoadHosts(path)
	if err != nil {
		return err
	}
	ref, err := cfg.Resolve(target)
	if err != nil {
		return err
	}
	cc, err := dbConnOf(ref)
	if err != nil {
		return err
	}
	srcTarget = dbconfig.NewTarget(cc, ref.Name, ref.Env.Host, ref.Env.User, ref.Env.Zone,
		entRaw, dbconfig.RouteClientDirect)
	// 用哪个账号由**显式给的企业编号**决定:与 tt debug 同一条规则、同一份快照 —— 见 entacct.go
	srcNotes = hookEntAccount(cc, ref, config.LocationsAt(path).DataDir(), entRaw)
	if common.Verbose {
		fmt.Fprintf(os.Stderr, "[dict] 数据源: 远程 %s (账号 %s/%s, %s %s)\n",
			ref.Name, srcTarget.Account, srcTarget.AccountSource, cc.Type, cc.Address())
	}
	l, err := live.Open(context.Background(), *cc)
	if err != nil {
		return fmt.Errorf("打开远程数据源 %q: %w", ref.Name, err)
	}
	dataSrc = l
	srcLocal = false
	return nil
}

// dbConnOf 取某环境一对一挂载的库连接(深拷贝:调用方要改 User/Password 等运行期槽位,
// 不能写回配置里的那份)。
func dbConnOf(ref *config.EnvRef) (*dbconfig.Connection, error) {
	if ref.Env.DB == nil {
		return nil, fmt.Errorf("环境 %q 未配置数据库(设置-环境-数据库页添加,或编辑 config.json hosts.sshs[].db)", ref.Name)
	}
	cc := *ref.Env.DB
	cc.Accounts = append([]dbconfig.DBAcct(nil), ref.Env.DB.Accounts...)
	return &cc, nil
}

// defaultEnvName 已删除 —— 它读的是 hosts.activeEnv(默认环境),而那个概念已移除。
// 没有"默认环境"就不该有任何地方替用户挑一个环境名;要环境就点 --env。

// missingTableErr 主字典表缺失。**这是错误(退出码 3),不是一句提示。**
//
// 从前它在十来处都是"打一句提示、然后退出 0",脚本与 agent 会把"没查过"
// 读成"查过了,没有" —— 而这两件事的后续动作完全不同(前者去 sync,后者改条件)。
// 缺表意味着数据源不完整,那是失败。
//
// 注意只用于**命令的主数据**;程序族那种"缺了还能靠另一族兜底"的降级
// (printProgTables、程序详情的用表索引)仍然打提示、不报错。
func missingTableErr(subject string) error {
	return output.Errorf(output.CodeTableMissing, output.ExitMissing, "%s", missingHint(subject))
}

// missingHint 主字典表缺失(IsMissingTable)时的提示。本地源给**两条路**:全量同步,
// 以及"免写盘"的远程直查(有些环境不允许/不方便写本地库,只报 db sync 会把人引到死路);
// 远程源说明库的问题与回退方式。subject 如 "校验定义 (dzcd_t 等表)"。
func missingHint(subject string) string {
	if srcLocal {
		return fmt.Sprintf("本地库尚未包含%s数据。二选一:\n  tt dict db sync --env <环境>   # 从该环境 ERP 全量刷新(原库自动备份为 .bak)\n  tt dict <命令> --env <环境名>       # 免写盘,直接查远程\n看各数据族缺什么: tt dict db status", subject)
	}
	return fmt.Sprintf("远程库缺少%s相关字典表(该环境数据库未同步或账号权限不足);可用 --env local:<环境> 切回该环境的本地库", subject)
}

// emptyHint 查询零结果时的提示:本地源沿用"先 db sync"建议,远程源说明语义。
// subject 如 "校验定义"。
func emptyHint(subject string) string {
	if srcLocal {
		return fmt.Sprintf("暂无%s数据。请先执行: tt dict db sync --env <环境>", subject)
	}
	return fmt.Sprintf("未查询到%s数据(库中无记录或 --kw 无匹配;可换 --env <其他环境> 再试)", subject)
}

// dictDataDir 数据目录（config.json 同目录）。
func dictDataDir() string {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return ""
	}
	return config.LocationsAt(path).DataDir()
}

// dictCfgPath 当前生效的配置文件路径（解析不到返回空）。
func dictCfgPath() string {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return ""
	}
	return path
}

// dictDBSyncTarget 某环境本地库的写侧目标：<该环境的库目录>/<环境>.db；
// -d/--db 或 TDICT_DB 显式给了就写那里。
func dictDBSyncTarget(env string) string {
	if p := strings.TrimSpace(dbPath); p != "" {
		return config.AbsPath(p)
	}
	return config.DictDBPathForEnv(dictCfgPath(), env)
}
