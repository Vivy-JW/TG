package dict

// tt dict 命令组的根:数据源钩子(打开/关闭查询数据源)与本地 SQLite 路径解析。
//
// 全局开关(--config/--json/--csv/-v/--env)由 tt 根命令的 persistent flags 提供
// (见 internal/cli/common),这里只保留字典独有的部分:--db 与数据源钩子。
// 合并前本文件还带着一份与 TDebug 逐行相同的配置路径解析(toolsHome/userConfigDir/
// isPortable/legacyConfigPaths/resolveConfigPath…),现在统一到 internal/config,
// 那一整块已删除。

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
	"tt/internal/config"
	"tt/internal/dict/db"
	"tt/internal/output"
)

// dbPath -d/--db:本地 SQLite 镜像路径(查询命令读它,db sync 写它)。

// dictFull 本次查询是否要完整视图。默认 summary；只有 --full 才回到旧的全量形状。
var dictFull bool
var dbPath string

// srcConfigPath 本次用的配置文件路径(PreRunE 里解析一次):它所在的目录既是
// 企业目录快照的落点,也是截断时结果落盘的落点。
var srcConfigPath string

// dictEnt 本次查询显式指定的企业编号(--ent);空 = 不按企业解析账号。
// 环境级 topent 已移除,所以这是企业编号的唯一来源之一(另一个是 tt debug --ent)。
var dictEnt string

func init() {
	// 字典独有的 persistent flag:--db 只对 tt dict 有意义
	Group.PersistentFlags().StringVarP(&dbPath, "db", "d", "",
		"本地 SQLite 路径(显式覆盖;缺省 = 数据目录/dict/<环境>.db;旧 TDICT_DB 仅对裸 local 生效)")
	// --limit 挂在命令组上而不是各命令上:返回条数上限是**全局策略**,不是某个命令的
	// 显示选项 —— 从前只有 msg/prog/r.t 各自定义了一个,其余命令等于没有防线。
	Group.PersistentFlags().IntVar(&queryLimit, "limit", -1,
		"结果最多返回多少条 (0 = 不限;-1 = 跟随配置;缺省取 config.json query.limit,再缺省 20)")
	// --ent 挂在命令组上:账号由企业编号决定(ENT → gzou_t → 账号),而企业编号
	// **不再存在环境配置里**(hosts.sshs[].topent 已删除)—— 每次显式给,
	// 不给就用账号列表首项(并在信封 notes 里说明)。
	Group.PersistentFlags().StringVar(&dictEnt, "ent", "",
		"企业编号(ENT):据此解析该环境用哪个账号(schema);不给 = 用账号列表首项")

	// 打开/关闭查询数据源:本地 SQLite(--db/TDICT_DB)或远程库(--env/query.source)。
	// 不依赖本地字典库的子命令(db/mirror/spill)各自覆盖本钩子。
	Group.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := runRootPreRun(cmd, args); err != nil {
			return err
		}
		// 落在**命令组自己**身上（没给子命令，或给了一个不认识的）时不碰数据源：
		// 这两种情况没人要查库，而 openQuerySource 在没配库的机器上会当场失败 ——
		// 于是 `tt dict` 想看一份命令列表，却拿到一句"本地数据库文件未找到"，
		// 而 `tt dict nope` 连"你打错了命令"都说不出来，被那句挡在前面（实测 2026-09-25）。
		// 判据用 `cmd == Group`：cobra 解析不到子命令时返回的就是组命令本身。
		if cmd == Group {
			return nil
		}
		loadQueryPolicy()
		return openQuerySource()
	}
	Group.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		closeQuerySource()
	}

	attachDataHint() // --help 末尾附一行版本 + 本地数据覆盖状态(见 helpdata.go)

	// 错误信封也要带"这次落在哪个环境/哪个账号" —— 出错时那比错误本身更难猜。
	common.MetaProvider = srcMeta
}

// runRootPreRun 显式补跑祖先命令(tt 根)的 PersistentPreRunE。
//
// cobra 只运行「从叶命令往上找到的第一个」PersistentPreRunE —— 子命令一旦定义,
// 父命令的就被整个跳过。从 Group.Parent() 起找,即刻意跳过 Group 自己的钩子
// (调用方本就是在替代它)。
//
// **必须把叶命令透传下去**: cobra 的契约是钩子收到"真正要执行的那条命令",
// 钩子也据此判自己该做什么（例如 tt 根钩子按命令路径解析 --at 的作用域）。
// 传祖先(p 自己)的话，根钩子会把一切当成 "tt" —— `tt dict db list --at X` 会被
// 判成"这条命令不接受 --at"。这不是理论，实测就是这么错的。
func runRootPreRun(cmd *cobra.Command, args []string) error {
	for p := Group.Parent(); p != nil; p = p.Parent() {
		if p.PersistentPreRunE != nil {
			return p.PersistentPreRunE(cmd, args)
		}
		if p.PersistentPreRun != nil {
			p.PersistentPreRun(cmd, args)
			return nil
		}
	}
	return nil
}

// skipQuerySource 供「不依赖本地字典库」的子命令覆盖 Group 的数据源钩子,
// 同时补跑祖先命令的钩子。
func skipQuerySource(cmd *cobra.Command, args []string) error { return runRootPreRun(cmd, args) }

// resolveDBPath 读侧定位本地 SQLite。候选与优先级只有一份实现 —— config.Locations
// (统一路径管理器):TDICT_DB → -d(相对路径按当前目录绝对化)→ 数据目录下的缺省名,
// 取第一个**存在**的;全部不存在时报错并列出尝试过的路径。
func resolveDBPath(flagPath string) (string, error) {
	p, tried, err := config.LocationsAt(srcConfigPath).ResolveDictDB(flagPath)
	if err != nil {
		return "", fmt.Errorf(
			"本地数据库文件未找到。\n\n尝试了以下路径:\n%s\n\n设置 TDICT_DB 环境变量或用 -d 指定正确路径:\n  setx TDICT_DB \"D:\\path\\to\\erp_data.db\"\n  tt dict -d \"D:\\path\\to\\erp_data.db\" r.t dzea_t",
			config.FormatTriedPaths(tried),
		)
	}
	return p, nil
}

// GetDB 返回当前查询数据源(本地 SQLite 或某环境的远程库),
// 由 Group 的 PersistentPreRunE 打开(见 source.go)。
func GetDB() db.Source { return dataSrc }

// IsJSON 本次是否输出 JSON(默认形态)。
func IsJSON() bool { return common.OutputFormat() == output.FormatJSON }

// IsCSV 本次是否输出 CSV。
func IsCSV() bool { return common.OutputFormat() == output.FormatCSV }

// Format 本次的输出形态(命令自己分流时用)。
func Format() output.Format { return common.OutputFormat() }

// notFoundErr 精确 ID 查不到：成功与否不能混成“查到空”。
// 退出码 4 与用法错(1)/数据源错(2)/缺表(3)分开，agent 可以直接分支。
func notFoundErr(format string, a ...any) *output.Error {
	return output.Errorf("NOT_FOUND", 4, format, a...)
}

// emit 渲染一次**结果集**查询:JSON 给带环境信息的信封,CSV 给 `# ` 头 + 主体,
// 人读表格由调用方在 text 分支自己打(各命令的文本输出比一张表丰富)。
//
// 返回条数受**全局上限**约束(命令行 --limit > config.json query.limit > 内置缺省),
// 三种形态一视同仁 —— 换个输出格式就换契约的话,一次宽查询足以打爆 agent 的上下文
// (实测 `tt dict msg --type 1 --status Y` 是 9.3 MB / 28005 条)。
// 截断时完整的那份落盘,信封里给 totalRows 与 localPath:截断是为了保护上下文,
// 不是为了丢数据。
func emit(data any, columns []string, rows [][]string) error {
	return emitCapped(true, data, columns, rows)
}

// emitDetail 单实体 + 它的明细行(如"一张表的全部字段规格")。
//
// **不按行数截断**:那是一张表的字段清单,不是结果集的一页 —— 按 20 行切下去
// 会把字段切掉一半,而调用方要的就是"这张表完整长什么样"。
// 条数照报,`--limit` 对它无效。
func emitDetail(data any, columns []string, rows [][]string) error {
	return emitDetailNotes(data, columns, rows)
}

// emitDetailNotes 同 emitDetail，并把这些可直接照做的提示放进 notes。
// 用途：单表查询时顺手告诉 agent 多表/批量的写法，不要求它先读 --help。
func emitDetailNotes(data any, columns []string, rows [][]string, notes ...string) error {
	total := len(rows)
	if rows == nil {
		total = payloadLen(data)
	}
	m := srcMeta()
	m.TotalRows, m.Returned = total, total
	m.Notes = append(m.Notes, notes...)
	return emitWith(m, data, columns, rows)
}

// emitOne 单实体(如一条校验定义)。它没有"结果集行数"可言,记 1 条 ——
// 记 0 会让人以为什么都没查到。
func emitOne(data any) error {
	return emitOneNotes(data)
}

// emitOneNotes 同 emitOne，并把可直接照做的批量提示放进 notes。
func emitOneNotes(data any, notes ...string) error {
	m := srcMeta()
	m.TotalRows, m.Returned = 1, 1
	m.Notes = append(m.Notes, notes...)
	return emitWith(m, data, nil, nil)
}

func emitCapped(capping bool, data any, columns []string, rows [][]string) error {
	total := len(rows)
	if rows == nil {
		total = payloadLen(data)
	}
	m := srcMeta()
	m.TotalRows, m.Returned = total, total

	if limit := limitOf(); capping && limit > 0 && total > limit {
		// **先落盘,再截。** 顺序反了就把丢掉的那部分也一起丢了。
		//
		// 落盘失败就**不截断** —— 静默截断比给一坨大的更坏:调用方会拿着残缺的结果
		// 当成全部去下结论,而这正是"不要误导 agent"要防的事。宁可让它大,不可让它假。
		path, err := spillResult(data)
		if err != nil {
			m.Notes = append(m.Notes, "结果超过返回上限("+strconv.Itoa(limit)+" 条),"+
				"但完整结果落盘失败,因此**未截断**、原样返回: "+err.Error())
			return emitWith(m, data, columns, rows)
		}
		data = truncatePayload(data, limit)
		if len(rows) > limit {
			rows = rows[:limit]
		}
		m.Returned, m.Truncated, m.LocalPath = limit, true, path
		// 三处说同一件事(notes / truncated+计数 / localPath),因为它最容易被读漏:
		// 只看 notes 的、只看计数的、只看 data 的,都得撞上"这不是全部"。
		m.Notes = append(m.Notes, fmt.Sprintf(
			"结果已截断:只回了 %d / %d 条。完整结果在 %s —— 翻页读它:tt dict spill show %s --offset %d"+
				"(要重查一次拿全量也可以:加 --limit 0)", limit, total, path, filepath.Base(path), limit))
	}
	return emitWith(m, data, columns, rows)
}

func emitWith(m output.Meta, data any, columns []string, rows [][]string) error {
	return output.Emit(os.Stdout, output.Options{
		Format: Format(), Meta: m, Data: data, Columns: columns, Rows: rows,
	})
}

// ---- 返回条数上限 ----

// queryLimit --limit:本次查询最多返回多少条。
// 默认 -1 = 跟随 config.json 的 query.limit(它再缺省 DefaultQueryLimit);0 = 不限。
var queryLimit int

// srcQueryLimit 来自 config.json 的 query.limit(在 PreRunE 里读一次)。
var srcQueryLimit = config.DefaultQueryLimit

// limitOf 本次生效的返回条数上限;0 = 不限。
func limitOf() int {
	if queryLimit >= 0 {
		return queryLimit
	}
	return srcQueryLimit
}

// payloadLen 载荷的元素个数:slice/array 看长度,其它(单实体)算 1,nil 算 0。
func payloadLen(v any) int {
	if v == nil {
		return 0
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		return rv.Len()
	}
	return 1
}

// truncatePayload 载荷是 slice 时保留前 n 个元素;其它形态原样返回(单实体没什么可截的)。
func truncatePayload(v any, n int) any {
	if v == nil || n <= 0 {
		return v
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice || rv.Len() <= n {
		return v
	}
	return rv.Slice(0, n).Interface()
}

// spillResult 把完整结果落一份到 <配置目录>/spill/,返回路径。
//
// 它**必须在截断之前**成功 —— 落不了盘就不截断(见 emitCapped)。想看全量就去 grep
// 这个文件,而不必把 9 MB 灌回上下文,也不必重跑一次查询。
func spillResult(data any) (string, error) {
	dir := config.LocationsAt(srcConfigPath).DataDir()
	if dir == "" {
		return "", errors.New("定位不到配置目录")
	}
	if data == nil {
		return "", errors.New("没有可落盘的数据")
	}
	sub := config.CacheDir(dir, spillDirName)
	if sub == "" {
		return "", errors.New("定位不到配置目录")
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		return "", err
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	p := filepath.Join(sub, "query-"+time.Now().Format("20060102-150405")+".json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		return "", err
	}
	return p, nil
}

const spillDirName = "spill"

// loadQueryPolicy 读一次全局查询策略。读不到配置不算错 —— 用内置缺省,查询照跑。
func loadQueryPolicy() {
	path, err := common.ResolveConfig(true)
	if err != nil {
		return
	}
	srcConfigPath = path
	root, err := config.Load(path)
	if err != nil {
		return
	}
	srcQueryLimit = root.Query.EffectiveLimit()
}

// srcMeta 把当前数据源的落脚点转成输出信封的环境信息块。
func srcMeta() output.Meta {
	// 字典查询全是只读的:本地 SQLite 是只读副本,远程库走只读语句 —— 如实标出来。
	m := output.Meta{Detail: srcDetail, Notes: append([]string(nil), srcNotes...), Source: srcKind(), Readonly: true}
	if srcDetail == "summary" {
		m.Notes = append(m.Notes, "当前为摘要视图；完整字段/关系加 --full")
	}
	if srcTarget != nil {
		t := srcTarget
		m.Env, m.SSHHost, m.Zone, m.Topent = t.Env, t.SSHHost, t.Zone, t.Topent
		m.Ent, m.Account, m.AccountSource = t.Ent, t.Account, t.AccountSource
		m.Target, m.Dialect, m.Route = t.Address(), t.Dialect(), t.Route
		m.ViaTunnel, m.LocalDBPath = t.ViaTunnel, t.Path
	}
	if !srcStarted.IsZero() {
		m.Elapsed = time.Since(srcStarted).Seconds()
	}
	return m
}

// srcKind 数据源大类:local(本地 SQLite 镜像) / live(某环境的远程库)。
func srcKind() string {
	if srcLocal {
		return "local"
	}
	if srcTarget != nil {
		return "live"
	}
	return ""
}
