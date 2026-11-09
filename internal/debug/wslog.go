package debug

// 接口日志(wsfa_t)查询与报文读取。
// 机制来自对 T100 的调研(awsp900_01.insertInto/writeResponseLog):
//   wsfa001=服务名 wsfa003/004=起止时间 wsfa005=耗时 wsfa006=返回码
//   wsfa007/008=请求/响应报文文件路径($TEMPDIR/日期/ws_req|res_时间_GUID.xml)
//   wsfa010/011=报文全文 CLOB(超过 A-SYS-0077 KB 上限时不入库,只记大小 wsfa016/017)
//   wsfa012=作业编号(gzja_t 服务名→程序解析结果) wsfa014=错误描述(≤100字)
// 列集对齐 T100 原生页 awsq990.4gl(整合服務端檢測工具):它 select 出
//   wsfa001,005,006,004,002,003,012,013,014,015,018,页面显示 9 列。
// 本工具此前只取其中一部分,现补齐原生页有而我们缺的四项:
//   wsfa002=服务程序序号(process_id) wsfa013=发起端 wsfa018=服务端 wsfa015=sso秒数
// (字段中文名以 tdict rt wsfa_t 的字典为准;awsa990 的列名与之一致)
// 重放调试 = 日志里内嵌的 `r.dg <作业> '<req>' '<rsp>'`:报文文件作 argv 重跑服务程序。

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"tt/internal/dbconfig"
	"tt/internal/host"

	"github.com/pkg/sftp"
)

// WSLogItem 接口日志列表行
type WSLogItem struct {
	RowID    string `json:"rowid"`
	Service  string `json:"service"`  // wsfa001 服务名称
	PID      string `json:"pid"`      // wsfa002 process_id(服务程序序号)
	Start    string `json:"start"`    // wsfa003 起始时间
	End      string `json:"end"`      // wsfa004 结束时间
	Duration string `json:"duration"` // wsfa005 处理时间
	Code     string `json:"code"`     // wsfa006 状态(srvcode,000=成功)
	Job      string `json:"job"`      // wsfa012 服务程序编号(重放时的作业)
	ReqPath  string `json:"reqPath"`  // wsfa007
	RspPath  string `json:"rspPath"`  // wsfa008
	ReqSize  string `json:"reqSize"`  // wsfa016(字节)
	RspSize  string `json:"rspSize"`  // wsfa017
	ErrMsg   string `json:"errMsg"`   // wsfa014 错误消息
	Origin   string `json:"origin"`   // wsfa013 发起端
	Server   string `json:"server"`   // wsfa018 服务端
	SSO      string `json:"sso"`      // wsfa015 sso秒数(小数 6 位)
}

// WSLogContent 选中日志的报文内容
type WSLogContent struct {
	Request  string `json:"request"`
	Response string `json:"response"`
	// RequestPartial 请求报文内容不完整:读到了文件/入库文本的上限,或压根拿不到。
	// **基于该内容编辑后重放会送出残缺报文**,故界面据此禁用"用修改后的入参重放"。
	// 注意:内容里不再追加"(仅显示前 N 字符)"之类的说明 —— 那是报文文本,
	// 源文件缺失时会被原样当成入参写回服务器。
	RequestPartial bool `json:"requestPartial,omitempty"`
	// ResponsePartial 响应报文同理(只影响展示,不影响重放)。
	ResponsePartial bool `json:"responsePartial,omitempty"`
}

// 报文读取上限:文件按字节截,CLOB 回退按字符截(两者同量级)。
// 超过上限的报文只读到这里,并把 *Partial 置真。
const (
	wslogFileMax    = 262144     // 报文文件单次读取上限(字节)
	wslogPayloadMax = 256 * 1024 // CLOB 回退单份上限(字符)
	// wslogClobChunk 一次从 CLOB 取多少字符。
	// Oracle 的 dbms_lob.substr 单次最多 32767 字符,而 sqlplus 会按 linesize 折行;
	// 取 2000 足够小,保证一个块自身不会被折断(块里的换行是报文自己的,原样保留)。
	wslogClobChunk = 2000
	// wslogChunkEnd 每块值的结束哨兵。sqlplus 会裁掉行尾空格,块边界正好落在缩进
	// 中间时那几个空格会被静默吃掉;把哨兵拼在值末尾,行尾就不是空白了。
	wslogChunkEnd = "\x02"
)

// reWSLogChunk 块分隔行:每块之前输出一行 @@TTCHUNK:<req|rsp>:<序号>@@。
// 解析时按它切块 —— 块内(报文自己的)换行原样保留,而 sqlplus 在块尾补的那一个
// 换行会被按行切分吃掉,不会混进报文。
var reWSLogChunk = regexp.MustCompile(`^@@TTCHUNK:(req|rsp):[0-9]+@@$`)

// reRowid 行标识白名单(拼 SQL 防注入):Oracle rowid / 金仓 ctid「(页,元组)」
var reRowid = regexp.MustCompile(`^(\([0-9]+,[0-9]+\)|[A-Za-z0-9./]{1,20})$`)

// reWSLogRow 列表行解析:字段间以 | 分隔;ErrMsg(第 10 列)允许空格——
// 失败记录的错误描述如「[T100_message] 处理笔数 1, 成功 0, 失败 1」含空格,
// 用 \S+ 会整行匹配失败导致记录被静默丢弃。
// 末尾五列(rspSize/发起端/服务端/sso/结束时间)用 [^|]* 而不是 .* —— 它们不含 |,
// 收紧后贪婪的 (.*) 不会把尾巴吃空。
var reWSLogRow = regexp.MustCompile(`^(\S+)\|(\S*)\|(\S*)\|(.*)\|(.*)\|(\S*)\|(\S*)\|(\S*)\|(.*)\|(.*)\|(\S*)\|([^|]*)\|([^|]*)\|([^|]*)\|([^|]*)\|([^|]*)$`)

// sqlWSLogCols 列表/详情共用的选列片段(顺序即 reWSLogRow 的分组顺序)。
// 末四项是对齐 T100 原生页 awsq990 补上的:wsfa013 发起端 / wsfa018 服务端 /
// wsfa015 sso秒数(sso 固定 6 位小数,与原生页显示一致;Oracle 需显式 format,
// 金仓 numeric::text 本身就保留标度)/ wsfa004 结束时间(截到秒,与 start 一致)。
// 所有列都套 nvl/coalesce:|| 遇 NULL 会把整行拼成 NULL(此前只有数值列这么处理)。
func sqlWSLogCols(kingbase bool) string {
	if kingbase {
		return `coalesce(wsfa013,'')||'|'||coalesce(wsfa018,'')||'|'||coalesce(wsfa015::text,'')||'|'||coalesce(substr(wsfa004,1,19),'')`
	}
	return `nvl(wsfa013,'')||'|'||nvl(wsfa018,'')||'|'||nvl(to_char(wsfa015,'FM99999999999990.000000'),'')||'|'||substr(nvl(wsfa004,''),1,19)`
}

// WSLogFilter 列表过滤条件。口径对齐 awsq990 主查询:
//
//	基线永远排除 wsfa001='docno.storage'(SSO 记录:wsfa003 存的不是时间,会刷满整页)
//	wsfa001 服务名(= / in,本工具放宽为 LIKE 通配)、wsfa006 处理结果、wsfa013 发起端、wsfa018 服务端
//	时间窗:wsfa003 >= 起 且 wsfa004 <= 止(整个调用落在窗口内)
//	onlyFail 为本工具扩展
type WSLogFilter struct {
	Service   string // 服务名(wsfa001),支持 * ? 通配
	Job       string // 作业编号(wsfa012),支持 * ? 通配 —— 按"哪个作业"找日志的入口
	Result    string // 处理结果(wsfa006)等值
	Origin    string // 发起端(wsfa013)等值
	Server    string // 服务端(wsfa018)等值
	PID       string // 服务程序序号(wsfa002)等值 —— 界面「服务程序」列就是它
	OnlyFail  bool   // 仅失败(wsfa006<>'000')
	StartFrom string // 起始时间起(wsfa003 >=,字符串字典序即时间序)
	EndTo     string // 结束时间止(wsfa004 <=;纯日期补到当天 23:59:59.99999)
	Page      int    // 从 1 起
	PageSize  int    // 每页条数(50..500,默认 200)
}

// reQBEValue 时间类输入白名单(仅日期时间字符)
var reQBEValue = regexp.MustCompile(`^[0-9: -]{0,19}$`)

// reWSLogEq 等值条件输入白名单:只禁单引号。
// 单引号是唯一的字符串逃逸手段(Oracle 与金仓都不把 \ 当转义),禁掉即可安全内联;
// 其余字符(空格/中文等)一律放行,避免过严导致"填了却静默查不到"。
var reWSLogEq = regexp.MustCompile(`^[^']{1,40}$`)

// wslogWhere 拼 WHERE 子句(纯函数,便于单测)。
// 条件全部为 AND,顺序固定;时间格式非法才返回 error(与既有接口行为一致)。
func wslogWhere(f WSLogFilter) (string, error) {
	// 基线:原生页固定排除的噪声服务(SSO 记录占满整页,见类型注释)
	wc := "1=1 AND wsfa001 != 'docno.storage'"
	if f.Service != "" {
		// 服务名(如 icd.erp.wo.out.query.get)支持 * ? 通配;其余字符白名单校验。
		// 这是原生 `wsfa001 = '值'` / `in (…)` 的超集:填精确值(不带通配符)时结果一致。
		patOK := regexp.MustCompile(`^[A-Za-z0-9._*?]{1,60}$`)
		if !patOK.MatchString(f.Service) {
			wc += " AND 1=0"
		} else {
			pat := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(f.Service), "*", "%"), "?", "_")
			wc += fmt.Sprintf(" AND UPPER(wsfa001) LIKE '%s'", strings.ReplaceAll(pat, "'", "''"))
		}
	}
	if f.Job != "" {
		// 作业编号(wsfa012,如 wssp01131):与 service 同款通配。
		// 这是**按"哪个作业"找日志**的唯一入口 —— 服务名是 oa.schema.data.get 这类反域名,
		// 光看服务名根本认不出是哪个作业在跑。
		patOK := regexp.MustCompile(`^[A-Za-z0-9._*?]{1,60}$`)
		if !patOK.MatchString(f.Job) {
			wc += " AND 1=0"
		} else {
			pat := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(f.Job), "*", "%"), "?", "_")
			wc += fmt.Sprintf(" AND UPPER(wsfa012) LIKE '%s'", strings.ReplaceAll(pat, "'", "''"))
		}
	}
	// 四个等值条件:处理结果 / 发起端 / 服务端 / 服务程序序号(原生 `and wsfa006 = '…'` 等)
	// wsfa002 是数字列,内联成 '861637' 由数据库隐式转换,与其它等值条件同一个白名单。
	for _, eq := range []struct{ val, col string }{
		{f.Result, "wsfa006"}, {f.Origin, "wsfa013"}, {f.Server, "wsfa018"}, {f.PID, "wsfa002"},
	} {
		v := strings.TrimSpace(eq.val)
		if v == "" {
			continue
		}
		if !reWSLogEq.MatchString(v) {
			wc += " AND 1=0"
			continue
		}
		wc += fmt.Sprintf(" AND %s = '%s'", eq.col, v)
	}
	if f.OnlyFail {
		wc += " AND wsfa006 <> '000'"
	}
	// 时间窗(原生口径):下界看起始时间 wsfa003,上界看结束时间 wsfa004
	if v := strings.TrimSpace(f.StartFrom); v != "" {
		if !reQBEValue.MatchString(v) {
			return "", fmt.Errorf("时间条件格式非法: %q", v)
		}
		wc += fmt.Sprintf(" AND wsfa003 >= '%s'", v)
	}
	if v := strings.TrimSpace(f.EndTo); v != "" {
		if !reQBEValue.MatchString(v) {
			return "", fmt.Errorf("时间条件格式非法: %q", v)
		}
		// 纯日期(yyyy-mm-dd)作上界时补到当天末尾(含毫秒位,与原生一致),
		// 否则字典序会把当天最后不到 1 秒的记录挡掉
		if len(v) == 10 {
			v += " 23:59:59.99999"
		}
		wc += fmt.Sprintf(" AND wsfa004 <= '%s'", v)
	}
	return wc, nil
}

// wsSysAccount 查接口日志(wsfa_t)固定用系统账号 ds。
//
// 为什么不按企业解析:wsfa_t 是 **ds 下的共享表** —— 接口日志不按企业分 schema,
// 报文里的企业只是行上的一个属性(与 gzou_t/gzzz_t 同一类)。所以这里没有"选哪个账号"
// 这回事,也就没有"回退"可言:压根不存在第二个候选。
//
// 对照:只读 SQL(tt debug sql)查的是**按企业分 schema 的业务表**,那边必须由 TOPENT
// 解析账号(见 dbsql.go 的 entAccount)。两者不一样,别把规矩搬错方向。
const wsSysAccount = dbconfig.SystemAccount

// WSLogAcct 说明这次查 wsfa_t 用了哪个账号、为什么,以及当前 TOPENT 是什么情况。
//
// 为什么要带它出去:"接口日志为空"这个结果,调用方必须能分辨是"这段时间确实没调用",
// 还是"企业/账号用错了"。只读 SQL 那边靠结果头的"企业 N → 账号 X"回答这个问题,
// 接口日志这边则要回答"它压根不按企业走"。
type WSLogAcct struct {
	Account string `json:"account"`
	Reason  string `json:"reason"`
	// TopentWarning 非空 = 当前 TOPENT 不是有效企业编号。
	//
	// **它不影响接口日志查询**(共享表),但会影响重放(wsdebug)时的框架据点校验,
	// 所以照样报出来 —— 免得把"日志查得到"和"重放跑得通"当成同一件事。
	TopentWarning string `json:"topentWarning,omitempty"`
}

// wslogAcctFor 决定查接口日志用的账号并说明理由。纯函数(不碰 SSH),便于单测。
func wslogAcctFor(rawTopent string) WSLogAcct {
	a := WSLogAcct{
		Account: wsSysAccount,
		Reason:  "wsfa_t 是系统账号 ds 下的共享表,接口日志不按企业分 schema",
	}
	raw := strings.TrimSpace(rawTopent)
	if raw == "" {
		a.TopentWarning = "当前没有生效的 TOPENT(企业编号);不影响接口日志查询,但重放时框架按空企业走"
		return a
	}
	if n, ok := host.EntValue(raw).Int(); !ok || n <= 0 {
		a.TopentWarning = fmt.Sprintf("TOPENT 不是有效企业编号(%s);不影响接口日志查询,但框架的据点校验会用到它", raw)
	}
	return a
}

// isDBErrorLine 判断这是不是一行数据库报错。
//
// **不能只认 ORA-**:金仓报的是 `ERROR: relation "wsfa_t" does not exist` 或 `FATAL: …`。
// 漏掉它们,"表不存在/连不上"会被当成"这段时间没有日志" —— 一个静默的错误结论。
//
// 故意用"行首匹配"而不是 Contains:报文里的报文字段本身可能带 ORA- 字样,
// 那不该让整次查询失败。
func isDBErrorLine(ln string) bool {
	t := strings.TrimSpace(ln)
	for _, p := range []string{"ORA-", "SP2-", "ERROR", "FATAL"} {
		if strings.HasPrefix(t, p) {
			return true
		}
	}
	return false
}

// listWSLogs 查询接口日志列表(Oracle: rowid + OFFSET/FETCH;金仓: ctid + OFFSET/LIMIT)
func listWSLogs(conn *host.SSHConn, dbc *dbRun, f WSLogFilter) (items []WSLogItem, hasMore bool, err error) {
	size := f.PageSize
	if size <= 0 || size > 500 {
		size = 200
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	wc, err := wslogWhere(f)
	if err != nil {
		return nil, false, err
	}
	connStr, err := dbc.dbConnStr(wsSysAccount)
	if err != nil {
		return nil, false, err
	}
	var out string
	if dbc.conn.Type == "kingbase" {
		// 金仓:ctid 作行标识;|| 遇 null 归 null,逐列 coalesce
		sql := fmt.Sprintf(`select wsfa.ctid||'|'||wsfa001||'|'||wsfa002||'|'||coalesce(substr(wsfa003,1,19),'')||'|'||coalesce(wsfa005::text,'')||'|'||coalesce(wsfa006,'')||'|'||coalesce(wsfa012,'')||'|'||coalesce(wsfa007,'')||'|'||coalesce(wsfa008,'')||'|'||coalesce(wsfa014,'')||'|'||coalesce(wsfa016::text,'')||'|'||coalesce(wsfa017::text,'')||'|'||%s from wsfa_t wsfa where %s order by wsfa003 desc, wsfa004 desc offset %d limit %d`, sqlWSLogCols(true), wc, (page-1)*size, size+1)
		out, err = dbc.exec(conn, connStr, "", sql, 40*time.Second)
	} else {
		sql := fmt.Sprintf(`set heading off
set feedback off
set trimspool on
set linesize 32767
select wsfa.rowid||'|'||wsfa001||'|'||wsfa002||'|'||substr(nvl(wsfa003,''),1,19)||'|'||nvl(wsfa005,'')||'|'||nvl(wsfa006,'')||'|'||nvl(wsfa012,'')||'|'||nvl(wsfa007,'')||'|'||nvl(wsfa008,'')||'|'||nvl(wsfa014,'')||'|'||nvl(wsfa016,'')||'|'||nvl(wsfa017,'')||'|'||%s
from wsfa_t wsfa where %s order by wsfa003 desc, wsfa004 desc offset %d rows fetch first %d rows only;`, sqlWSLogCols(false), wc, (page-1)*size, size+1)
		out, err = dbc.exec(conn, connStr, sql, "", 40*time.Second)
	}
	if err != nil {
		return nil, false, fmt.Errorf("查询 wsfa_t 失败: %w (%s)", err, host.FirstLines(out, 3))
	}
	var all []WSLogItem
	for _, ln := range strings.Split(out, "\n") {
		if isDBErrorLine(ln) {
			return nil, false, fmt.Errorf("wsfa_t 查询出错: %s", strings.TrimSpace(ln))
		}
		m := reWSLogRow.FindStringSubmatch(strings.TrimRight(ln, " \r"))
		if m == nil {
			continue
		}
		all = append(all, WSLogItem{
			RowID: m[1], Service: m[2], PID: m[3], Start: m[4], Duration: m[5],
			Code: m[6], Job: m[7], ReqPath: m[8], RspPath: m[9],
			ErrMsg: m[10], ReqSize: m[11], RspSize: m[12],
			Origin: m[13], Server: m[14], SSO: m[15], End: m[16],
		})
	}
	if len(all) > size {
		hasMore = true
		all = all[:size]
	}
	return all, hasMore, nil
}

// WSLogDetail 取单条日志:列表字段 + 报文内容(CLOB 优先,文件回退)
func WSLogDetail(conn *host.SSHConn, dbc *dbRun, rowid string) (*WSLogItem, *WSLogContent, error) {
	if !reRowid.MatchString(rowid) {
		return nil, nil, fmt.Errorf("rowid 格式非法")
	}
	var out string
	var err error
	connStr, err := dbc.dbConnStr(wsSysAccount)
	if err != nil {
		return nil, nil, err
	}
	if dbc.conn.Type == "kingbase" {
		sql := fmt.Sprintf(`select wsfa.ctid||'|'||wsfa001||'|'||wsfa002||'|'||coalesce(substr(wsfa003,1,19),'')||'|'||coalesce(wsfa005::text,'')||'|'||coalesce(wsfa006,'')||'|'||coalesce(wsfa012,'')||'|'||coalesce(wsfa007,'')||'|'||coalesce(wsfa008,'')||'|'||coalesce(wsfa014,'')||'|'||coalesce(wsfa016::text,'')||'|'||coalesce(wsfa017::text,'')||'|'||%s from wsfa_t wsfa where ctid='%s'`, sqlWSLogCols(true), rowid)
		out, err = dbc.exec(conn, connStr, "", sql, 30*time.Second)
	} else {
		sql := fmt.Sprintf(`set heading off
set feedback off
set trimspool on
set linesize 32767
set long 300000
set longchunksize 100000
select wsfa.rowid||'|'||wsfa001||'|'||wsfa002||'|'||substr(nvl(wsfa003,''),1,19)||'|'||nvl(wsfa005,'')||'|'||nvl(wsfa006,'')||'|'||nvl(wsfa012,'')||'|'||nvl(wsfa007,'')||'|'||nvl(wsfa008,'')||'|'||nvl(wsfa014,'')||'|'||nvl(wsfa016,'')||'|'||nvl(wsfa017,'')||'|'||%s
from wsfa_t wsfa where rowid='%s';`, sqlWSLogCols(false), rowid)
		out, err = dbc.exec(conn, connStr, sql, "", 30*time.Second)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("查询 wsfa_t 失败: %w (%s)", err, host.FirstLines(out, 3))
	}
	var item *WSLogItem
	for _, ln := range strings.Split(out, "\n") {
		if isDBErrorLine(ln) {
			return nil, nil, fmt.Errorf("wsfa_t 查询出错: %s", strings.TrimSpace(ln))
		}
		if m := reWSLogRow.FindStringSubmatch(strings.TrimRight(ln, " \r")); m != nil {
			item = &WSLogItem{
				RowID: m[1], Service: m[2], PID: m[3], Start: m[4], Duration: m[5],
				Code: m[6], Job: m[7], ReqPath: m[8], RspPath: m[9],
				ErrMsg: m[10], ReqSize: m[11], RspSize: m[12],
				Origin: m[13], Server: m[14], SSO: m[15], End: m[16],
			}
			break
		}
	}
	if item == nil {
		return nil, nil, fmt.Errorf("日志记录不存在(可能已被清理)")
	}

	content := &WSLogContent{}
	// 1) 优先读报文文件(磁盘,最完整;按日期目录轮转清理,旧文件可能不存在)
	//    两个文件共用一个 SFTP 通道并发读:此前是"每文件各开一次 SFTP 子通道 + 串行",
	//    每次都要一次子系统握手;pkg/sftp 的 Client 明确支持多 goroutine 并发(内部请求 id 多路复用),
	//    但不可与 Close 并发,故先 Wait 再 Close。
	//    返回值第二个布尔 = 是否被 wslogFileMax 上限截断(界面要据此禁用"改后重放")。
	readFile := func(s *sftp.Client, path string) (string, bool) {
		if path == "" {
			return "", false
		}
		f, e := s.Open(path)
		if e != nil {
			return "", false
		}
		defer f.Close()
		var size int64 = -1
		if st, e := f.Stat(); e == nil {
			size = st.Size()
		}
		buf := make([]byte, wslogFileMax)
		n, _ := f.Read(buf) // File.Read 内部会循环填满缓冲区(或到 EOF),无需自己重试
		return string(buf[:n]), size > int64(n)
	}
	var reqTrunc, rspTrunc bool
	if item.ReqPath != "" || item.RspPath != "" {
		if s, e := conn.SFTP(); e == nil {
			var wg sync.WaitGroup
			wg.Add(2)
			go func() { defer wg.Done(); content.Request, reqTrunc = readFile(s, item.ReqPath) }()
			go func() { defer wg.Done(); content.Response, rspTrunc = readFile(s, item.RspPath) }()
			wg.Wait()
			s.Close()
		}
	}

	// 2) 文件已被清理 → 回退 CLOB。整份读,不再只取前 2000 字符:
	//    wsfa010/011 存的就是报文全文(只有超过入库上限时才不存),此前固定截 2000,
	//    3~4KB 的正常报文也被判成"不完整",界面因此禁用"改后重放"。
	//    现在按 wslogClobChunk 分块取(块之间用 @@TTCHUNK:…@@ 行分隔),直到 wslogPayloadMax。
	if content.Request == "" || content.Response == "" {
		reqLen, rspLen := wslogClobLengths(conn, dbc, connStr, item.RowID)
		if reqLen > 0 || rspLen > 0 {
			req, rsp, reqCut, rspCut, e := readWSLogClobs(conn, dbc, connStr, item.RowID,
				content.Request == "", reqLen, content.Response == "", rspLen)
			if e != nil {
				// 读失败不等于"没有报文":内容留空、partial 置真,界面会明确标不完整
				if content.Request == "" {
					reqTrunc = true
				}
				if content.Response == "" {
					rspTrunc = true
				}
			} else {
				if content.Request == "" && reqLen > 0 {
					content.Request, reqTrunc = req, reqCut || req == ""
				}
				if content.Response == "" && rspLen > 0 {
					content.Response, rspTrunc = rsp, rspCut || rsp == ""
				}
			}
		}
	}

	// 拿不到内容时不再塞占位文案:那段文字会被当成入参写回服务器(见 WriteReplayFiles)。
	// 界面用空内容 + "不完整"横幅 + 空态提示三件事说清楚。
	// 请求为空一定要标不完整(它不能被当成入参);响应为空不标 —— 有些调用本来就没有响应体。
	if content.Request == "" {
		reqTrunc = true
	}
	content.RequestPartial = reqTrunc
	content.ResponsePartial = rspTrunc
	return item, content, nil
}

// wslogClobLengths 取两份报文的入库长度(字符数);没入库的返回 0。
// 查不到/查询失败一律当 0 —— 调用方会落到"没有报文"的路径,不会凭空造内容。
func wslogClobLengths(conn *host.SSHConn, dbc *dbRun, connStr, rowid string) (reqLen, rspLen int) {
	var out string
	var err error
	if dbc.conn.Type == "kingbase" {
		sql := fmt.Sprintf(`select coalesce(length(wsfa010),0)||'|'||coalesce(length(wsfa011),0) from wsfa_t where ctid='%s'`, rowid)
		out, err = dbc.exec(conn, connStr, "", sql, 30*time.Second)
	} else {
		sql := fmt.Sprintf(`set heading off
set feedback off
set pagesize 0
set trimspool on
select nvl(length(wsfa010),0)||'|'||nvl(length(wsfa011),0) from wsfa_t where rowid='%s';`, rowid)
		out, err = dbc.exec(conn, connStr, sql, "", 30*time.Second)
	}
	if err != nil {
		return 0, 0
	}
	return parseWSLogClobLengths(out)
}

// parseWSLogClobLengths 解析 "请求长度|响应长度" 那一行(纯函数,便于单测)。
func parseWSLogClobLengths(out string) (int, int) {
	for _, ln := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimSpace(ln), "|")
		if len(parts) != 2 {
			continue
		}
		a, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		b, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if e1 != nil || e2 != nil {
			continue
		}
		return a, b
	}
	return 0, 0
}

// readWSLogClobs 按块读回入库的请求/响应报文。
// needReq/needRsp 只有对应文件缺失、需要回退时才为真,避免白读一份。
// 返回的 *cut 表示"库里还有更多,但已到 wslogPayloadMax 上限"。
func readWSLogClobs(conn *host.SSHConn, dbc *dbRun, connStr, rowid string, needReq bool, reqLen int, needRsp bool, rspLen int) (req, rsp string, reqCut, rspCut bool, err error) {
	reqN, rspN := reqLen, rspLen
	if reqN > wslogPayloadMax {
		reqN, reqCut = wslogPayloadMax, true
	}
	if rspN > wslogPayloadMax {
		rspN, rspCut = wslogPayloadMax, true
	}
	if !needReq {
		reqN = 0
	}
	if !needRsp {
		rspN = 0
	}
	if reqN <= 0 && rspN <= 0 {
		return "", "", reqCut, rspCut, nil
	}
	kingbase := dbc.conn.Type == "kingbase"
	var b strings.Builder
	emit := func(kind, col string, n int) {
		for off := 0; off < n; off += wslogClobChunk {
			size := wslogClobChunk
			if off+size > n {
				size = n - off
			}
			if kingbase {
				fmt.Fprintf(&b, "select '@@TTCHUNK:%s:%d@@';\n", kind, off/wslogClobChunk)
				fmt.Fprintf(&b, "select substr(%s,%d,%d)||chr(2) from wsfa_t where ctid='%s';\n", col, off+1, size, rowid)
			} else {
				fmt.Fprintf(&b, "select '@@TTCHUNK:%s:%d@@' from dual;\n", kind, off/wslogClobChunk)
				// 末尾拼一个 STX 哨兵:块边界可能正好落在缩进中间,而 sqlplus 会裁掉
				// 行尾空格(TRIMOUT/TRIMSPOOL),没有哨兵时那几个空格会被静默吃掉。
				fmt.Fprintf(&b, "select dbms_lob.substr(%s,%d,%d)||chr(2) from wsfa_t where rowid='%s';\n", col, size, off+1, rowid)
			}
		}
	}
	if reqN > 0 {
		emit("req", "wsfa010", reqN)
	}
	if rspN > 0 {
		emit("rsp", "wsfa011", rspN)
	}
	var out string
	if kingbase {
		out, err = dbc.exec(conn, connStr, "", b.String(), 90*time.Second)
	} else {
		out, err = dbc.exec(conn, connStr, "set heading off\nset feedback off\nset pagesize 0\nset echo off\nset tab off\nset trimspool on\nset linesize 32767\n"+b.String(), "", 90*time.Second)
	}
	if err != nil {
		return "", "", reqCut, rspCut, err
	}
	req, rsp, gotReq, gotRsp := parseWSLogChunks(out)
	// 少了块 = 某条 select 没出力(报错/被裁),这份报文就是残缺的 —— 宁可标不完整,
	// 也不能让界面把半份报文当成原文去重放。
	if reqN > 0 && gotReq != wslogChunkCount(reqN) {
		reqCut = true
	}
	if rspN > 0 && gotRsp != wslogChunkCount(rspN) {
		rspCut = true
	}
	return req, rsp, reqCut, rspCut, nil
}

// parseWSLogChunks 把 @@TTCHUNK:<kind>:<n>@@ 分隔的输出切回报文(纯函数,便于单测)。
// 每块的值以 STX(\x02) 结尾(见 readWSLogClobs 的 SQL):sqlplus 会在语句之后多打一个
// 空行,靠"行尾哨兵"而不是"下一个 marker"来断块,块尾的空格/换行才不会被误删或误加。
// 返回块数,调用方据此判断是否有块没读到。
func parseWSLogChunks(out string) (req, rsp string, reqChunks, rspChunks int) {
	var reqB, rspB strings.Builder
	kind := ""
	var lines []string
	flush := func() {
		if kind == "" {
			return
		}
		text := strings.Join(lines, "\n")
		if kind == "req" {
			reqB.WriteString(text)
			reqChunks++
		} else {
			rspB.WriteString(text)
			rspChunks++
		}
		lines = lines[:0]
	}
	for _, ln := range strings.Split(out, "\n") {
		ln = strings.TrimRight(ln, "\r")
		if m := reWSLogChunk.FindStringSubmatch(ln); m != nil {
			flush()
			kind = m[1]
			continue
		}
		if kind == "" {
			continue // 第一个块之前 / 块结束之后的 sqlplus 噪声一律不算内容
		}
		if strings.HasSuffix(ln, wslogChunkEnd) {
			lines = append(lines, strings.TrimSuffix(ln, wslogChunkEnd))
			flush()
			kind = "" // 本块到此为止;后面的空行是 sqlplus 的语句分隔行
			continue
		}
		lines = append(lines, ln)
	}
	flush()
	return reqB.String(), rspB.String(), reqChunks, rspChunks
}

// wslogChunkCount 读 n 个字符需要几个块。
func wslogChunkCount(n int) int {
	if n <= 0 {
		return 0
	}
	return (n + wslogClobChunk - 1) / wslogClobChunk
}

// maxReplayPayload 界面回传的入参上限(与接口测试同一量级;防止误把大文件塞进 argv 文件)
const maxReplayPayload = 256 * 1024

// checkReplayOverride 校验界面回传的"改过的入参":空表示用原报文;超限直接拒绝
func checkReplayOverride(s string) error {
	if s == "" {
		return nil
	}
	if len(s) > maxReplayPayload {
		return fmt.Errorf("入参 %d 字节,超过 %dKB 上限,拒绝重放", len(s), maxReplayPayload/1024)
	}
	return nil
}

// WriteReplayFiles 报文文件不存在时,把 CLOB 内容写到服务器临时文件供重放。
// 返回 (请求路径, 响应路径, 告警文案, error);告警非空表示这次重放的结果**可能不可信**。
// reqOverride 非空 = 界面里改过的入参:无条件落临时文件并优先使用
// (**不能走 ensure 的"原文件还在就用原文件"分支**,否则用户的修改会被静默忽略)。
//
// 请求与响应的处置**不对称**,理由见下面两处注释:
//   - 请求:能复用服务器上的原文件就复用(只读,最忠实);
//   - 响应:永远写新的临时文件,绝不复用 —— 那是日志的证据。
func WriteReplayFiles(conn *host.SSHConn, item *WSLogItem, content *WSLogContent, reqOverride string) (reqPath, rspPath, warn string, err error) {
	if err := checkReplayOverride(reqOverride); err != nil {
		return "", "", "", err
	}
	sftp, err := conn.SFTP()
	if err != nil {
		return "", "", "", err
	}
	defer sftp.Close()
	// 内容落 $TEMPDIR(aws 报文同目录规则)。目录只探一次。
	out, e := conn.Output("bash -lc 'echo $TEMPDIR'", 10*time.Second)
	tmpDir := strings.TrimSpace(out)
	if e != nil || tmpDir == "" {
		tmpDir = "/tmp"
	}
	writeTemp := func(text string) (string, error) {
		np := fmt.Sprintf("%s/tdebug_replay_%d.xml", tmpDir, time.Now().UnixNano()%1000000)
		nf, e := sftp.Create(np)
		if e != nil {
			return "", fmt.Errorf("写重放报文失败: %w", e)
		}
		defer nf.Close()
		if _, e := nf.Write([]byte(text)); e != nil {
			return "", fmt.Errorf("写重放报文失败: %w", e)
		}
		return np, nil
	}
	// 返回是否复用了原文件 —— 调用方据此判断"是不是在用入库文本"
	ensure := func(path, text string) (string, bool, error) {
		if path != "" {
			if f, e := sftp.Open(path); e == nil {
				f.Close()
				return path, true, nil // 文件还在,复用
			}
		}
		if text == "" {
			return "", false, nil
		}
		p, e := writeTemp(text)
		return p, false, e
	}
	reusedReq := false
	if reqOverride != "" {
		if reqPath, err = writeTemp(reqOverride); err != nil {
			return "", "", "", err
		}
	} else if reqPath, reusedReq, err = ensure(item.ReqPath, content.Request); err != nil {
		return "", "", "", err
	}
	// 响应**总是**写新的临时文件,绝不复用 item.RspPath:那个路径是这次调用在服务器上的
	// 原始响应报文,而重放程序会把结果写回它 —— 等于把日志证据抹掉。后来的人(包括 AI)
	// 再去看这条日志,读到的会是某次重放的产物,而不是当初真实返回的那份,排查方向会被
	// 整个带偏。请求可以复用(只读),响应不行。
	if rspPath, err = writeTemp(content.Response); err != nil {
		return "", "", "", err
	}
	if reqPath == "" {
		return "", "", "", fmt.Errorf("请求报文不可用(文件已清理且未入库)")
	}
	// 请求走了"用入库文本"这条路,而那份文本可能是不完整的(源文件已清理时尤甚)。
	// 不是错误,但必须让调用方知道:JSON 报文被截断后,框架先按 JSON 解析会失败、再落到
	// XML 路径,直接崩在 Start tag expected —— 表现为程序一路退出、断点怎么设都不命中。
	if !reusedReq && content.RequestPartial {
		warn = fmt.Sprintf("原始请求文件已不在服务器上,这次重放用的是入库的截断文本(%d 字符);"+
			"报文为 JSON 时框架会解析失败(表现:程序一路退出、断点不命中),结果不可信",
			len(content.Request))
	}
	return reqPath, rspPath, warn, nil
}
