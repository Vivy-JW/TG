package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"tt/internal/dbconfig"
)

// SchemaVersion 当前配置结构版本。随新配置文件写入（见 NewSkeleton / ensureSchemaVersion）。
// 旧结构的配置不做自动迁移 —— 读侧按当前结构解析，缺的节取缺省值。
//
// 4：新增 skills 节（第三方技能登记的路径清单）。
const SchemaVersion = 4

// 缺省值。合并前三处各自硬编码，现在只有这一份。
const (
	DefaultListen          = "127.0.0.1:28670"
	DefaultLaunchArgs      = "BBDL512840855a 2 12345 'N' {prog}"
	DefaultWatchdogSeconds = 1800
	DefaultTermWidth       = 200
	DefaultTermHeight      = 50
	DefaultPrintElements   = 1000
	DefaultSSHPort         = 22
	DefaultWorkspaceSuffix = "-ws"
)

// ---------- 环境模型 ----------

// SSHConfig 远程服务器连接配置
type SSHConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Addr 返回 SSH 地址 host:port
func (c *SSHConfig) Addr() string { return c.Host + ":" + strconv.Itoa(c.Port) }

// EntValue 企业编号(TOPENT)：数字或文本均可，兼容 JSON 数字。
// 需真实编号的场景用 Int()（非数字返回 false）。
type EntValue string

// UnmarshalJSON 同时接受 JSON 数字与字符串
func (e *EntValue) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*e = ""
		return nil
	}
	*e = EntValue(strings.Trim(s, `"`))
	return nil
}

// MarshalJSON 统一序列化为字符串
func (e EntValue) MarshalJSON() ([]byte, error) { return json.Marshal(string(e)) }

// Int 解析为数字（供数据库探测按企业编号匹配；非数字内容返回 false）
func (e EntValue) Int() (int, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(string(e)))
	return n, err == nil
}

// NamedSsh 服务器环境：SSH 连接 + 登录区域。
//
// 合并自 TDebug/host.NamedSsh 与 TDictCli/host.NamedSsh；TDebug 多出的
// LaunchArgs / WatchdogSeconds 是该环境的调试参数覆盖，omitempty 不影响 TDict 侧。
//
// 这是三个工具共用的唯一环境登记处：原来 tdebug 放在 debug.sshs、
// tdict 放在 hosts.sshs，schema 完全一致，合并后只留 hosts.sshs。
//
// **刻意没有 topent(企业编号)**:企业是"这一次要连哪个企业"运行时参数,
// 登记在环境上等于给它一个会被静默沿用的默认值 —— 而用错企业的典型症状是
// "查不到数据",与"数据不存在"完全一样。现在 ENT 只能每次显式给:
// CLI `--ent`、Web 会话面板的输入框。
type NamedSsh struct {
	Name            string `json:"name"`
	SSHConfig              // 匠名嵌入：host/port/user/password 提升到 ssh 层
	Zone            string `json:"zone,omitempty"`            // 登录后区域菜单代码：31开发 35测试 36正式 39PATCH t出货
	LaunchArgs      string `json:"launchArgs,omitempty"`      // 该环境的作业启动参数覆盖（空=用 debug.launchArgs）
	WatchdogSeconds int    `json:"watchdogSeconds,omitempty"` // 该环境的停站超时覆盖（0=用 debug.watchdogSeconds）
	// Workspace 这个环境的 .tzs 工作区（含 mta/ 的目录，绝对路径）。
	//
	// 与上面两项同类：按环境覆盖机器级的东西 —— 但工作区**没有**机器级那份
	// （见 TzsSettings）：切了环境就该跟着换工作区，而引擎内置的默认工作区是一个
	// 真实客户目录，配错就等于拿别人的表单当草稿纸。
	Workspace string `json:"workspace,omitempty"`
	// Product 这个环境属于哪个产品："" / "t100" = T100，"gp" = TOP GP。
	// 同一份 hosts.sshs 装两种环境，靠它区分；GP 的专用字段见下面几项。
	Product string `json:"product,omitempty"`
	// Instance TOP GP 的实例名（topprod / toptest / topstd …）。
	// 用户只填实例名，Workspace / CustRoot 由它推导：
	// /u1/<instance>/tiptop、/u1/<instance>/topcust。
	Instance string `json:"instance,omitempty"`
	// CustRoot TOP GP 的 $CUST 客制根（服务器上的路径）。T100 不用。
	CustRoot string `json:"custRoot,omitempty"`
	// Company 资料库编号（azp_file.azp03 的小写形式 = zta_file.zta02 =
	// Oracle schema 名：ds ↔ DS）。空 = "ds"。
	//
	// 为什么不叫"据点代号"：据点编号是 azp01，资料库编号是 azp03，两者不是
	// 一回事（实测 30 个据点里 15 个不同：COF → cof_demo、DEMO1 → demo1、
	// DIGIWIN → digiwin），aooi901（工厂基本资料维护作业）里也是分开的两个
	// 栏位。查字典要的是 schema，所以这里存 azp03，UI 上从据点表里挑。
	//
	// 为什么必须有这一项：GP 一个库里装了几十家公司，字典表按资料库编号分行
	// 存（同一个表名，ds / digiwin / puran 各一行）。连库账号**不等于**它 ——
	// 实测这套库账号叫 digiwin、而它的 schema 是 DIGIWIN，zta_file 里
	// zta02='ds' 有 2,962 行、'digiwin' 只有 2,902 行；两边混用会出现
	// "本地查得到、远程说表不存在"，且表格总数对不上。
	//
	// 用法：**只影响实际查库**（直查要按资料库编号选 schema 与字典行）。
	// 本地字典库的同步是固定按 "ds" 取的，见 internal/dict/dbsync 的 SyncGP。
	Company string `json:"company,omitempty"`
	// CompanyName 营运据点名字（azp_file.azp02），从服务器刷新时抄下来。
	//
	// 只是显示用：查询口径看 Company。存一份是为了不打开设置页就得连库，
	// 也让"这个代号到底是哪家公司"在 config.json 里自解释。
	CompanyName string `json:"companyName,omitempty"`
	// CompanySyncedAt 上次「从服务器刷新」的时间（RFC3339，本地时区）。
	//
	// 与 T100 的 entsSyncedAt 同义：只是让人知道这张据点表是什么时候抄的，
	// 不参与任何判断 —— 表里的 name 才是准的，过期了自己再点一次刷新。
	CompanySyncedAt string `json:"companySyncedAt,omitempty"`
	// MirrorDir 这个环境的源码镜像目录（绝对路径）；空 = 用顶层 mirror.dir/<环境名>。
	MirrorDir string `json:"mirrorDir,omitempty"`
	// DictDir 这个环境的本地字典库目录（绝对路径）；空 = 用顶层 dict.dir，再空 = <数据目录>/dict。
	//
	// 只到**目录**这一级：文件名始终是 <环境>.db（见 DictDBFileName）。不允许直接指到某个 .db ——
	// 那样同一个环境可能被 CLI 指向 A、被界面指向 B，而"读到另一个库"与"数据不存在"
	// 在输出上分不出来。
	DictDir string               `json:"dictDir,omitempty"`
	DB      *dbconfig.Connection `json:"db,omitempty"` // 该环境的数据库连接（与 SSH 一对一）
	// Ents 这个环境支持的企业编号与各自的账号（gzou_t 的 gzou001/gzou003）。
	//
	// 为什么存进配置（而不是每次现查 gzou_t）：企业清单是**慢变量**，而“能不能看
	// 到可选项”不该依赖网络（VPN 未开、不同环境在不同内网，都会把发现变成一堆超时）。
	// 因此这里存的是“某次从 gzou_t 抄下来的清单”，由设置页「ENT」页签或
	// `tt env ents --refresh` 显式同步（不允许手填）。entsSyncedAt 就是那份清单的
	// 抄写时间 —— 它不参与判定，但必须如实带出去：账号被改过而没同步，用户要能看见。
	Ents         []EntEntry `json:"ents,omitempty"`
	EntsSyncedAt string     `json:"entsSyncedAt,omitempty"` // RFC3339
}

// EntEntry 一条“企业编号 → 数据库账号”的归档。
//
// 账号是连接该企业 schema 用的那个（gzou_t.gzou003）；Password 是同步时从
// db.accounts 抄过来的副本（账号清单里没收录时按 T100 惯例 账号=密码）。
// 运行时**优先用 db.accounts 里的口令**（那是唯一的编辑入口），这里这份只有
// 在账号清单里没有该账号时才用 —— 两处不一致时，以人在设置里改过的那份为准。
type EntEntry struct {
	Ent      int    `json:"ent"`
	Account  string `json:"account"`
	Password string `json:"password,omitempty"`
}

// Hosts 服务器环境清单：config.json 顶层 hosts 节。
//
// **刻意没有 ActiveEnv(默认环境)**:那是"猜一个环境"的入口 —— 同机多环境时
// 猜错的后果是连到另一台机/另一个企业上,而输出看起来完全正常(与"上错号"无法区分)。
// 所以环境一律**点名**:CLI 用 `--env <环境名>`(不传就报错并列出可用环境),
// Web 由会话面板选中。见 resolve.go 的 pickName —— 它现在只认显式名字。
type Hosts struct {
	SSHs []NamedSsh `json:"sshs"`
}

// EntsOf 返回某环境的 ENT 清单与同步时间（未配置/环境不存在返回 nil）。
func (h *Hosts) EntsOf(env string) ([]EntEntry, string) {
	if e := h.ByName(env); e != nil {
		return e.Ents, e.EntsSyncedAt
	}
	return nil, ""
}

// ByName 按名取环境；未命中(含 name 为空)返回 nil。
// 选择顺序只有一份实现(见 resolve.go 的 pickName);需要"为什么选中它"或
// "没找到时列出有哪些环境"时用 Hosts.Resolve。
func (h *Hosts) ByName(name string) *NamedSsh {
	target, _ := h.pickName(name)
	if target == "" {
		return nil
	}
	for i := range h.SSHs {
		if h.SSHs[i].Name == target {
			return &h.SSHs[i]
		}
	}
	return nil
}

// Names 返回全部环境名（顺序与配置一致）。
func (h *Hosts) Names() []string {
	out := make([]string, 0, len(h.SSHs))
	for i := range h.SSHs {
		out = append(out, h.SSHs[i].Name)
	}
	return out
}

// Normalize 补全缺省值：只补端口(22)。**不再补默认环境** —— 见 Hosts 的注释。
func (h *Hosts) Normalize() {
	for i := range h.SSHs {
		if h.SSHs[i].Port == 0 {
			h.SSHs[i].Port = DefaultSSHPort
		}
	}
}

// ---------- 工具设置节 ----------

// DebugSettings 调试工具自有设置（原 TDebug 的 debug 节，减去 sshs）。
//
// **刻意没有 ActiveEnv(调试默认环境)**:与 hosts 去掉默认环境同理 —— 调试连错环境
// 等于在别人的库上跑作业。环境一律由 `tt debug start --env <名>` 或界面点名。
type DebugSettings struct {
	LaunchArgs      string `json:"launchArgs,omitempty"`      // 作业启动参数默认模板，{prog} 替换为作业名（环境可覆盖）
	WatchdogSeconds int    `json:"watchdogSeconds,omitempty"` // 停站停留超时默认（秒）；环境可覆盖
	FGLServer       string `json:"fglserver,omitempty"`       // 留空使用 T100 按 SSH 来源 IP 自动设置
	TermWidth       int    `json:"termWidth,omitempty"`
	TermHeight      int    `json:"termHeight,omitempty"`
	PrintElements   int    `json:"printElements,omitempty"`      // fgldb 单次 print 的数组元素上限；0=默认 1000
	PersistBPs      *bool  `json:"persistBreakpoints,omitempty"` // 断点持久化开关（nil 视为 true）
	// AITakeover 默认起步模式：true 时新建调试会话直接以协作模式起（AI 可写：启动作业/下断点/步进/求值）。
	// 默认 false —— 新建会话以纯人工起步，人是默认驾驶者；但 AI 可对已有会话自行
	// `tt debug mode collab` 接管，这个开关只是省掉那一次切换。
	AITakeover bool `json:"aiTakeover,omitempty"`
}

// FillDefaults 补全调试设置缺省值（不写盘，仅在内存里生效）。
func (d *DebugSettings) FillDefaults() {
	if d.LaunchArgs == "" {
		d.LaunchArgs = DefaultLaunchArgs
	}
	if d.WatchdogSeconds == 0 {
		d.WatchdogSeconds = DefaultWatchdogSeconds
	}
	if d.TermWidth == 0 {
		d.TermWidth = DefaultTermWidth
	}
	if d.TermHeight == 0 {
		d.TermHeight = DefaultTermHeight
	}
	if d.PrintElements == 0 {
		d.PrintElements = DefaultPrintElements
	}
}

// WatchdogProblems 校验看门狗设置。
//
// 为什么值得单独一件事：`watchdogSeconds` 是**唯一**会“静默关掉防行锁保护”的开关。
// 0 会被 FillDefaults 归一成默认 1800（不会变成“关闭”），但**负数会原样留着**，
// 而 `Session.startWatchdog` 的门是 `<= 0` —— 于是 -1 = 关掉看门狗，而配置表面上
// 一直存在这个键。现场一旦写错，症状是“停站了再也没人管”，却没有任何地方提示。
//
// 统一在这里出问题列表，`tt config validate` 与设置页保存都调它，不各写一套。
func WatchdogProblems(r *Root) []string {
	if r == nil {
		return nil
	}
	var out []string
	if v := r.Debug.WatchdogSeconds; v < 0 {
		out = append(out, fmt.Sprintf(
			"debug.watchdogSeconds = %d：负数等于关闭看门狗，停站将不再自动放行（要恢复默认保护写 0；要缩短超时写正整数）", v))
	}
	for i := range r.Hosts.SSHs {
		e := &r.Hosts.SSHs[i]
		if e.WatchdogSeconds < 0 {
			name := e.Name
			if name == "" {
				name = fmt.Sprintf("hosts.sshs[%d]", i)
			}
			out = append(out, fmt.Sprintf(
				"环境 %q 的 watchdogSeconds = %d：负数等于关闭该环境的看门狗，停站将不再自动放行", name, e.WatchdogSeconds))
		}
	}
	return out
}

// WatchdogDisabled 判断一个看门狗秒数是不是“关闭”（负数）。0 不算关闭 —— 它会被
// FillDefaults 补成默认值，这一点与运行时 `<= 0` 的门故意不同：配置层面的 0 是
// “没写”，运行时的 0 是“显式关”。区分开才能让“写 0 的人以为关了”变成不可能。
func WatchdogDisabled(seconds int) bool { return seconds < 0 }

// BPsPersisted 断点持久化是否启用
func (d *DebugSettings) BPsPersisted(dataDir string) bool {
	return dataDir != "" && (d.PersistBPs == nil || *d.PersistBPs)
}

// QuerySettings 字典查询的数据源选择。
//
// Source: "" 或 "auto" = 不固定（命令行必须点名环境）；"local:<环境>" = 该环境的本地库；
// <环境名> = 固定直查该环境的远程库。裸 "local" 已无意义（本地库按环境分文件），会报错。
//
// 注意合并前 TDictCli 的 README 写着"默认是本地 SQLite 镜像"，与它自己的代码相反
// （cli/source.go 的注释写的是"缺省 = **在线**"）。这里以代码为准。
//
// 也**不做自动降级**：选了在线就是在线，连不上直接报错并提示怎么切，不静默回落本地 ——
// 静默回落会让用户以为查到的是实时数据，实际是几天前的镜像。
type QuerySettings struct {
	Source string `json:"source,omitempty"`

	// Limit 查询结果默认**返回**多少条(0 = 不限;未配置 = DefaultQueryLimit)。
	//
	// 它不是"显示上限"而是**返回上限**,文本/CSV/JSON 一视同仁 —— 否则换个输出格式
	// 就换了契约,而一次宽查询足以把 agent 的上下文打爆(实测 `tt dict msg --type 1
	// --status Y` 是 9.3 MB / 28005 条)。命令行的 --limit 覆盖本次调用。
	//
	// 截断时结果不会丢:完整的那份落盘,信封里给 totalRows 与 localPath。
	Limit int `json:"limit,omitempty"`
}

// DefaultQueryLimit 未配置 query.limit 时的返回条数上限。
// 取 20 与既有几处 --limit 的默认值一致(消息列表 / 程序作业数 / 用表反查)。
const DefaultQueryLimit = 20

// EffectiveLimit 生效的返回条数上限;0 = 不限。
func (q QuerySettings) EffectiveLimit() int {
	if q.Limit > 0 {
		return q.Limit
	}
	if q.Limit < 0 {
		return 0 // 显式配成负数 = 不限(与 --limit 0 同义)
	}
	return DefaultQueryLimit
}

// MirrorSettings 源码镜像落点（绝对路径）。
type MirrorSettings struct {
	// 不带 omitempty：空串是**有意义的取值**（"没配，走缺省"）。带 omitempty 时
	// 清空这个输入框会让 sec 里没有 dir 键，而 putSection 会把旧值补回来 ——
	// 用户看到的就是"清了保存，还是原样"。
	Dir string `json:"dir"`
}

// DictSettings 本地字典库（SQLite）的根目录（绝对路径）。
//
// 空 = 缺省落点 `<数据目录>/dict`。各环境仍是 `<根目录>/<环境>.db` —— 只挪根、
// 不按环境各指一个文件：CLI、Web、查询三边读同一份配置才不会各读各的库。
type DictSettings struct {
	Dir string `json:"dir"` // 同 MirrorSettings：空串有意义，不能 omitempty
}

// SyncSettings 旧版单文件字典库落点（绝对路径）。
//
// 现在本地库按环境分文件（<数据目录>/dict/<环境>.db），Target 不再参与新落点；
// 保留字段只为读旧配置不报错，设置页也不再写它。
type SyncSettings struct {
	Target string `json:"target,omitempty"`
}

// TdevSettings 设计器包工具设置。原 TDev 完全没有配置文件，这是新增节；
// 只放跨调用稳定的默认值，命令行 flag 仍然优先。
type TdevSettings struct {
	WorkspaceSuffix string `json:"workspaceSuffix,omitempty"` // 导出工作区默认后缀（缺省 -ws）
	DefaultOut      string `json:"defaultOut,omitempty"`      // tzs 解压等命令的默认输出目录
}

// WorkspaceSuffixOrDefault 返回生效的工作区后缀。
func (t *TdevSettings) WorkspaceSuffixOrDefault() string {
	if t.WorkspaceSuffix != "" {
		return t.WorkspaceSuffix
	}
	return DefaultWorkspaceSuffix
}

// SkillsSettings 第三方技能登记（官方技能不走这里）。
//
// 官方技能是发版包的一部分（<exe 目录>\skills），走专用路径，不在这张表里 —— 它们
// 不需要登记、也不能被禁用。这里只放用户自己攒的那些：一行一个**技能目录**（目录里有
// SKILL.md），路径可以是硬盘上的任意位置。
//
// **顺序即优先级**：两条登记解析出同一个技能名时，先登记者赢，后者告警丢弃。
// 不要把这里当成“父目录清单”—— 装什么必须能在这张表里看全，否则往文件夹里丢一个技能
// 就会静默改变安装结果。
//
// 这里刻意不存“装到哪儿了”：那是 update.SkillsStamp 的事（一次安装的事实，不属于
// 某条登记），两处都记会养出两个互相矛盾的真相源。
type SkillsSettings struct {
	Externals []SkillExternal `json:"externals,omitempty"`
}

// Entries 按登记顺序返回有效条目（对齐同一个过滤：路径为空的丢掉）。
//
// Web 的「技能」页靠它把 label 与磁盘判定结果一一对上 —— Paths() 已经丢过空条目，
// 直接拿原下标对齐会错位。
func (s *SkillsSettings) Entries() []SkillExternal {
	out := make([]SkillExternal, 0, len(s.Externals))
	for _, e := range s.Externals {
		if strings.TrimSpace(e.Path) != "" {
			out = append(out, e)
		}
	}
	return out
}

// Paths 按登记顺序取出路径清单（给 skill.BuildPlan 用）。
//
// 只给路径：技能名、合规性、冲突一律由 skill 包从磁盘上判，配置里的 label 不参与。
func (s *SkillsSettings) Paths() []string {
	entries := s.Entries()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimSpace(e.Path))
	}
	return out
}

// SkillExternal 一条第三方技能登记。
//
// Path 是**唯一身份**（绝对路径）；Label 只是路径失效时的显示名，永远不是权威 ——
// 技能叫什么只认 SKILL.md 的 frontmatter，两处不一致时以磁盘为准。
type SkillExternal struct {
	Path  string `json:"path"`
	Label string `json:"label,omitempty"`
}

// NetSettings 出网设置。目前只有代理一件事。
//
// 为什么放配置里而不是只认环境变量：代理地址是**机器级事实**（内网部署的常态就是
// 出网必须经代理），而环境变量属于“这次会话”—— 换一个终端就没了，设置页也无从展示。
// 生效优先级见 internal/update 的 ProxyFor：参数 → 本字段 → 环境变量 → 直连。
type NetSettings struct {
	Proxy string `json:"proxy,omitempty"` // 如 http://127.0.0.1:10808；空 = 看环境变量，再不行直连
}

// TzsSettings 是 .tzs 表单引擎（engine/，一个 C# 外部 exe）的运行时依赖。
//
// 注意这里**没有设计器目录**：设计器的程序集由发行包自带（`tzs\designer\`），引擎默认
// 就从那儿加载，所以它不再是配置项。也**没有工作区** —— 工作区是**按环境**的（见
// NamedSsh.Workspace）：一台机器上对着几个客户就有几个工作区，放在机器级只会让“切了
// 环境、引擎还开在别的客户的目录上”这种事有机会发生。
//
// 与 tdev 节的口径差异：tdev 放的是“跨调用稳定的默认值”，flag 永远优先；这一节放的是
// 机器级依赖，命令行 flag 管不着。
type TzsSettings struct {
	ServerExe string `json:"serverExe,omitempty"` // 覆盖 tzs-server.exe；缺省 <exe 目录>\tzs\tzs-server.exe
}

// ---------- 配置根 ----------
// Root 是 config.json 的类型化视图。
//
// 持久化仍以 map（见 cfgfile.go）为准 —— Root 只是读侧的一层类型化收敛，
// 写侧一律走 EditSection，保证未知顶层键与未知节内键都不会被悄悄抹掉。
type Root struct {
	SchemaVersion int
	Hosts         Hosts
	Listen        string
	Debug         DebugSettings
	Query         QuerySettings
	Mirror        MirrorSettings
	Dict          DictSettings
	Sync          SyncSettings
	Tdev          TdevSettings
	Tzs           TzsSettings
	Net           NetSettings
	Skills        SkillsSettings
}

// Load 读取并解析配置，返回类型化视图。
// 文件不存在时返回一份填好缺省值的空配置（首次运行没有文件不是错误）。
func Load(path string) (*Root, error) {
	root, err := Open(path)
	if err != nil {
		return nil, err
	}
	r := &Root{}
	r.SchemaVersion = intOf(root["schemaVersion"])
	if err := decodeSection(root, "hosts", &r.Hosts); err != nil {
		return nil, err
	}
	r.Hosts.Normalize()
	r.Listen = stringOf(root["listen"])
	if r.Listen == "" {
		r.Listen = DefaultListen
	}
	if err := decodeSection(root, "debug", &r.Debug); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "query", &r.Query); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "mirror", &r.Mirror); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "dict", &r.Dict); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "sync", &r.Sync); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "tdev", &r.Tdev); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "tzs", &r.Tzs); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "net", &r.Net); err != nil {
		return nil, err
	}
	if err := decodeSection(root, "skills", &r.Skills); err != nil {
		return nil, err
	}
	return r, nil
}

// LoadHosts 读取配置文件并返回环境清单，要求至少配置了一个环境
// （CLI 命令都按"有环境可连"的前提工作）。**不再补默认环境** —— 环境一律点名。
// 读路径只认 hosts，不做任何旧结构兼容。
func LoadHosts(path string) (*Hosts, error) {
	r, err := Load(path)
	if err != nil {
		return nil, err
	}
	h := r.Hosts
	if len(h.SSHs) == 0 {
		return nil, fmt.Errorf("尚未配置 SSH 环境:请运行 tt serve 打开配置页添加，或编辑 config.json 的 hosts.sshs")
	}
	return &h, nil
}

// **ActiveEnv / ActiveWorkspace 已随"移除默认环境"删除**(不是改成返回 nil —— 那样
// 调用方会静默拿到空值,把"忘了指定环境"伪装成"这个环境没配工作区")。
// 需要环境时:Hosts.Resolve(name) / Hosts.ByName(name),名字必须由调用方给出;
// 需要工作区时:先定环境,再取该环境的 Workspace。

// ActiveHost 返回生效环境：工具的覆盖优先，其次 hosts.activeEnv，最后首条。
// 与 ResolveForTool 同一套规则,只是这里不要错误(取不到就是 nil)。
func (r *Root) ActiveHost(tool string) *NamedSsh {
	ref, err := r.ResolveForTool(tool, "")
	if err != nil {
		return nil
	}
	return ref.Env
}

// ---------- 读写辅助 ----------

// EditSection 只改一个顶层节，其余键（含本节的未知子键）原样保留。
// mutate 收到的 sec 一定是非 nil 的 map，可直接改。
func EditSection(path, key string, validate func(root map[string]any) error, mutate func(sec map[string]any) error) error {
	return Edit(path, validate, func(root map[string]any) error {
		if mutate == nil {
			return nil
		}
		return mutate(SectionOrEmpty(root, key))
	})
}

// SaveHosts 用类型化的环境清单整体替换 hosts 节（其余顶层节不动）。
func SaveHosts(path string, h Hosts) error {
	return Edit(path, nil, func(root map[string]any) error {
		sec, err := toMap(h)
		if err != nil {
			return err
		}
		root["hosts"] = sec
		ensureSchemaVersion(root)
		return nil
	})
}

// EditHosts 读-改-写环境清单：f 拿到当前清单（已补缺省），改完整体写回。
// 环境页的保存走它。
func EditHosts(path string, f func(h *Hosts) error) error {
	return Edit(path, nil, func(root map[string]any) error {
		var h Hosts
		if err := decodeSection(root, "hosts", &h); err != nil {
			return err
		}
		h.Normalize()
		if err := f(&h); err != nil {
			return err
		}
		sec, err := toMap(h)
		if err != nil {
			return err
		}
		root["hosts"] = sec
		ensureSchemaVersion(root)
		return nil
	})
}

// SetListen 写统一监听地址。
func SetListen(path, listen string) error {
	return Edit(path, nil, func(root map[string]any) error {
		root["listen"] = listen
		ensureSchemaVersion(root)
		return nil
	})
}

// ensureSchemaVersion 打上当前结构版本号。
func ensureSchemaVersion(root map[string]any) {
	root["schemaVersion"] = SchemaVersion
}

// ---------- map ↔ 类型转换 ----------

// decodeSection 把 root[key] 反序列化到 v；节缺失时把 v 留作零值（不是错误），
// 节存在但不是对象时报错。
func decodeSection(root map[string]any, key string, v any) error {
	raw, ok := root[key]
	if !ok || raw == nil {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return fmt.Errorf("配置节 %q 无法序列化: %w", key, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("配置节 %q 格式不正确: %v", key, err)
	}
	return nil
}

// toMap 把类型化结构转成可持久化的 map（经由 JSON 标签，保留 omitempty 语义）。
func toMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// intOf 从 JSON 解出来的值取整数（数字/字符串皆可），取不到返回 0。
func intOf(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		i, _ := strconv.Atoi(n)
		return i
	}
	return 0
}

// stringOf 从 JSON 解出来的值取字符串，取不到返回空串。
func stringOf(v any) string {
	s, _ := v.(string)
	return s
}
