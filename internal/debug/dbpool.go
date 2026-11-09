package debug

// 常驻进程里的直连会话池（只服务 tt serve；CLI 一条命令一个进程，不适用）。
//
// 要解决的问题：`/api/dbsql` 每个请求都重新拨 SSH、并且走服务器 sqlplus（实测
// 1.3 s/条）。而 tt serve 是长期活着的进程 —— 把"建连 + 登录"留在进程里，之后的
// 查询只付执行时间（同一台机实测 10–30 ms）。
//
// 边界（刻意）：
//   - **不做健康检查**：多一次往返就把省下的时间还回去了。连接死了由查询报错暴露，
//     出事那条从池里退役，下一次请求重连。
//   - **不存口令**：池里只有连接本身，凭据仍在配置里，随配置热替换整池作废。
//   - 空闲超时回收（默认 5 分钟）：比探测缓存长，因为"登录"比"探测路径"贵。
//
// 两条并发约束（都是"共享一条连接"带的，不是可选项）：
//
//  1. **一条 SQL 写错，不等于连接坏了。** 从前任何查询错误都直接 Close，于是 A 的
//     `select * from 不存在的表` 会把 B 正在用的连接从脚下抽走（B 收到
//     `sql: database is closed`）。现在只有"像是连接断了"的错误才立刻退役，其余
//     累计到 dbDirectMaxStrikes 次才换 —— 真掉线但驱动没归类的情况仍有兜底。
//  2. **退役不是立刻关。** 有请求正在这条连接上查询时（inUse > 0），退役只做标记，
//     等它 release 再关。配置热替换（closeAll）同理：整池退役，在飞的不受影响。

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// dbDirectIdleTTL 会话空闲多久回收。
const dbDirectIdleTTL = 5 * time.Minute

// dbDirectMaxStrikes 一条连接连续失败多少次后判定它不可用。
//
// 有它才敢"不因为一次 SQL 写错就扔连接"：驱动的连接错误分类不总是可靠（go-ora 把
// 一部分断连包成普通 error），三次连续失败足以说明问题，而重连只要 ~55ms。
const dbDirectMaxStrikes = 3

type dbPoolEntry struct {
	q        *DirectQuery
	lastUsed time.Time

	inUse   int  // 正在这条连接上查询的请求数
	retired bool // 退役：不再发放，等在飞的 release 后关闭
	strikes int  // 连续失败次数（非连接类错误累计）
}

// dbDirectPool 按 key（环境|企业|账号）复用直连会话。
type dbDirectPool struct {
	mu  sync.Mutex
	m   map[string]*dbPoolEntry
	ttl time.Duration
}

func newDBDirectPool() *dbDirectPool {
	return &dbDirectPool{m: make(map[string]*dbPoolEntry), ttl: dbDirectIdleTTL}
}

// acquire 取一条可复用的会话，并返回 release：查询结束后**必须**调用（defer）。
//
// release 是"这条连接还有人用"的唯一凭据 —— 退役与配置热替换都靠它决定现在关还是
// 等一会儿关。漏调不会泄漏（进程会关），但会让一条已经退役的连接迟迟不释放。
func (p *dbDirectPool) acquire(key string, open func() (*DirectQuery, error)) (*DirectQuery, func(), error) {
	p.mu.Lock()
	if e := p.m[key]; e != nil {
		if time.Since(e.lastUsed) < p.ttl {
			e.lastUsed = time.Now()
			e.inUse++
			q := e.q
			p.mu.Unlock()
			return q, p.releaseFunc(e), nil
		}
		// 空闲超时：先摘掉再关（此时 inUse 必为 0 —— 在使用中会不断刷新 lastUsed）
		delete(p.m, key)
		p.mu.Unlock()
		e.q.Close()
		return p.openAndStore(key, open)
	}
	p.mu.Unlock()
	return p.openAndStore(key, open)
}

// openAndStore 新建并登记；登记时若已有别处放进来的（并发），用先到的那条，
// 自己这条关掉 —— 否则同一 key 会攒下多条连接，越用越多。
func (p *dbDirectPool) openAndStore(key string, open func() (*DirectQuery, error)) (*DirectQuery, func(), error) {
	q, err := open()
	if err != nil {
		return nil, nil, err
	}
	p.mu.Lock()
	if e := p.m[key]; e != nil {
		e.lastUsed = time.Now()
		e.inUse++
		winner := e.q
		p.mu.Unlock()
		q.Close()
		return winner, p.releaseFunc(e), nil
	}
	e := &dbPoolEntry{q: q, lastUsed: time.Now(), inUse: 1}
	p.m[key] = e
	p.mu.Unlock()
	return q, p.releaseFunc(e), nil
}

// releaseFunc 生成一次性的 release（重复调用不会把 inUse 减成负数）。
func (p *dbDirectPool) releaseFunc(e *dbPoolEntry) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			if e.inUse > 0 {
				e.inUse--
			}
			closeNow := e.retired && e.inUse == 0
			p.mu.Unlock()
			if closeNow {
				e.q.Close()
			}
		})
	}
}

// report 上报一次查询的结果，由它决定这条连接要不要退役。
//
// 分成三类处理：
//   - 成功：清零连续失败计数（"刚才那条写错了"不该跟着连接一辈子）；
//   - 连接的错（驱动/网络/断连 ORA）：立刻退役，下一次请求重连；
//   - 其余（语法错、表不存在、只读闸门）：只记一笔，不动连接 —— 换条连接也查不出
//     同一句错 SQL，反而把别人手里的热连接抽走。
func (p *dbDirectPool) report(key string, q *DirectQuery, err error) {
	if err == nil {
		p.mu.Lock()
		if e := p.m[key]; e != nil && e.q == q {
			e.strikes = 0
		}
		p.mu.Unlock()
		return
	}
	p.mu.Lock()
	e := p.m[key]
	if e == nil || e.q != q {
		p.mu.Unlock()
		return
	}
	e.strikes++
	retire := connBroken(err) || e.strikes >= dbDirectMaxStrikes
	if !retire {
		p.mu.Unlock()
		return
	}
	delete(p.m, key)
	e.retired = true
	closeNow := e.inUse == 0
	p.mu.Unlock()
	if closeNow {
		q.Close()
	}
}

// drop 把**出错的这一条**从池里退役（别的请求可能刚换成新的了，比对指针再动）。
// 只在确知连接不可用时用；"这句 SQL 写错了"应当交给 report。
func (p *dbDirectPool) drop(key string, q *DirectQuery) {
	p.mu.Lock()
	e := p.m[key]
	if e == nil || e.q != q {
		// 池里已经是另一条（并发换过）或者已被摘掉：这条自己关，不动池里那条。
		p.mu.Unlock()
		if q != nil {
			q.Close()
		}
		return
	}
	delete(p.m, key)
	e.retired = true
	closeNow := e.inUse == 0
	p.mu.Unlock()
	if closeNow {
		q.Close()
	}
}

// closeAll 让整池退役：配置热替换（连接要素可能变了）用。
//
// **不立即 Close**：有请求正在查询时关掉连接，那个请求会以
// `sql: database is closed` 失败 —— 一次配置保存不该变成别人的一次查询失败。
// 在飞的请求 release 时自然关闭。
func (p *dbDirectPool) closeAll() {
	if p == nil {
		return
	}
	p.mu.Lock()
	old := p.m
	p.m = make(map[string]*dbPoolEntry)
	now := make([]*DirectQuery, 0, len(old))
	for _, e := range old {
		e.retired = true
		if e.inUse == 0 {
			now = append(now, e.q)
		}
	}
	p.mu.Unlock()
	for _, q := range now {
		q.Close()
	}
}

// connBroken 判断一个查询错误是不是"连接这条管子坏了"（vs "这句 SQL 有问题"）。
//
// 判不准时**倾向 false**：误判成断连会白白关掉别人手里的热连接，误判成 SQL 错
// 只是多试几次（有 dbDirectMaxStrikes 兜底）。
func connBroken(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	// 驱动没归类的情况：Oracle 断连类错误码 + 传输层常见文案。
	// 只列真·连接断开的码，别把 ORA-00942（表或视图不存在）这类算进来。
	msg := err.Error()
	for _, code := range []string{
		"ORA-03113", // end-of-file on communication channel
		"ORA-03114", // not connected to ORACLE
		"ORA-03135", // connection lost contact
		"ORA-12537", // TNS:connection closed
		"ORA-12541", // TNS:no listener
		"ORA-12547", // TNS:lost contact
		"ORA-12570", // TNS:packet reader failure
	} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	low := strings.ToLower(msg)
	return strings.Contains(low, "connection reset") ||
		strings.Contains(low, "broken pipe") ||
		strings.Contains(low, "connection refused") ||
		strings.Contains(low, "i/o timeout")
}
