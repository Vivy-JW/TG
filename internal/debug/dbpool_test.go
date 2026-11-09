package debug

// 常驻直连池的契约：命中复用、空闲回收、出错摘除、配置热替换整池作废。
// 全部离线：池里放的是不连库的令牌（&DirectQuery{}），只验证"复用/回收"的簿记。

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"
)

func token() *DirectQuery { return &DirectQuery{} }

// open4Test 是测试用的 open：拿到一条令牌，并可选地记下"这条被关过"。
func open4Test(opened *int) func() (*DirectQuery, error) {
	return func() (*DirectQuery, error) {
		*opened++
		return token(), nil
	}
}

func TestDBDirectPoolReusesSameConn(t *testing.T) {
	p := newDBDirectPool()
	opened := 0
	open := open4Test(&opened)

	a, relA, err := p.acquire("k", open)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	defer relA()
	b, relB, err := p.acquire("k", open)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	defer relB()
	if opened != 1 || a != b {
		t.Fatalf("同一 key 应复用同一条连接（open 次数 %d）", opened)
	}
	// 不同 key 各自一条
	c, relC, _ := p.acquire("k2", open)
	defer relC()
	if c == a || opened != 2 {
		t.Fatalf("不同 key 不能共用连接")
	}
}

func TestDBDirectPoolReopensAfterIdle(t *testing.T) {
	p := newDBDirectPool()
	p.ttl = time.Millisecond
	opened := 0
	open := open4Test(&opened)

	_, rel, err := p.acquire("k", open)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	rel()
	time.Sleep(5 * time.Millisecond)
	_, rel2, err := p.acquire("k", open)
	if err != nil {
		t.Fatalf("不该报错: %v", err)
	}
	rel2()
	if opened != 2 {
		t.Errorf("空闲超时应重连（open 次数 %d，期望 2）", opened)
	}
}

func TestDBDirectPoolDropClosesAndForgets(t *testing.T) {
	p := newDBDirectPool()
	q, rel, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })
	rel()
	p.drop("k", q)
	if e := p.m["k"]; e != nil {
		t.Fatal("drop 之后池里不该还留着它")
	}
	if !q.closed.Load() {
		t.Fatal("drop 后应当关掉那条连接")
	}
	// 再取一次必须是新连接
	q2, rel2, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })
	rel2()
	if q2 == q {
		t.Fatal("被摘掉的连接不能又被发出来")
	}
}

func TestDBDirectPoolDropOnlyRemovesMatchingConn(t *testing.T) {
	// 并发下别的请求可能刚把 key 换成新连接：旧连接出错的 drop 不能连新的也删掉。
	p := newDBDirectPool()
	old, relOld, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })
	relOld()
	fresh, relFresh, _ := p.acquire("k2", func() (*DirectQuery, error) { return token(), nil })
	relFresh()
	p.mu.Lock()
	p.m["k"] = &dbPoolEntry{q: fresh, lastUsed: time.Now()} // 假装别处换过
	p.mu.Unlock()
	p.drop("k", old)
	if e := p.m["k"]; e == nil || e.q != fresh {
		t.Fatal("drop 只该摘掉出错的这一条，不该误删新连接")
	}
	if e := p.m["k"]; e.q.closed.Load() {
		t.Fatal("误删新连接：它不该被关掉")
	}
}

// 这一条是本次修的核心：一句写错的 SQL 不许把别人正用的连接抽走。
func TestDBDirectPoolSQLRejectDoesNotEvict(t *testing.T) {
	p := newDBDirectPool()
	opened := 0
	open := open4Test(&opened)

	q, rel, _ := p.acquire("k", open)
	rel()
	// ORA-00942：表或视图不存在 —— 典型的"这句 SQL 写错了"
	p.report("k", q, errors.New("query: ORA-00942: table or view does not exist"))

	e := p.m["k"]
	if e == nil || e.q != q {
		t.Fatal("SQL 写错不该换连接（换一条也查不出同一个错）")
	}
	if q.closed.Load() {
		t.Fatal("SQL 写错却把连接关了：正在用它的请求会收到 database is closed")
	}
	if opened != 1 {
		t.Fatalf("不该重连，open 次数 %d", opened)
	}

	// 连续失败到上限才换：给"真掉线但驱动没归类"留兜底
	for i := 0; i < dbDirectMaxStrikes-1; i++ {
		p.report("k", q, errors.New("query: ORA-00942: table or view does not exist"))
	}
	if e := p.m["k"]; e != nil {
		t.Fatalf("连续 %d 次失败后应退役，实际还在", dbDirectMaxStrikes)
	}
	if !q.closed.Load() {
		t.Fatal("退役的连接应当被关掉")
	}
}

// 成功一次就清零连续失败计数：偶尔打错一句不该把连接越推越靠近退役。
func TestDBDirectPoolSuccessResetsStrikes(t *testing.T) {
	p := newDBDirectPool()
	q, rel, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })
	rel()
	bad := errors.New("query: ORA-00942: table or view does not exist")
	p.report("k", q, bad)
	p.report("k", q, bad)
	p.report("k", q, nil)
	p.report("k", q, bad)
	if e := p.m["k"]; e == nil {
		t.Fatal("成功一次之后计数应清零，连接不该退役")
	}
}

// 连接真的断了（驱动归类过的）→ 立刻退役，下一次请求重连。
func TestDBDirectPoolConnBrokenEvictsImmediately(t *testing.T) {
	p := newDBDirectPool()
	opened := 0
	open := open4Test(&opened)
	q, rel, _ := p.acquire("k", open)
	rel()

	p.report("k", q, sql.ErrConnDone)
	if e := p.m["k"]; e != nil {
		t.Fatal("连接断了应当立刻退役")
	}
	if !q.closed.Load() {
		t.Fatal("退役的连接应当被关掉")
	}
	if _, rel2, _ := p.acquire("k", open); opened != 2 {
		t.Fatalf("退役后下一次该重连，open 次数 %d", opened)
	} else {
		rel2()
	}
}

func TestConnBrokenClassification(t *testing.T) {
	broken := []error{
		sql.ErrConnDone,
		errors.New("query: ORA-03113: end-of-file on communication channel"),
		errors.New("query: ORA-12537: TNS:connection closed"),
		errors.New("read tcp 1.2.3.4:5678: connection reset by peer"),
	}
	for _, err := range broken {
		if !connBroken(err) {
			t.Errorf("应判为连接坏: %v", err)
		}
	}
	notBroken := []error{
		nil,
		errors.New("query: ORA-00942: table or view does not exist"),
		errors.New("query: ORA-00936: missing expression"),
		errors.New("只允许单条 SELECT"),
	}
	for _, err := range notBroken {
		if connBroken(err) {
			t.Errorf("不该判为连接坏: %v", err)
		}
	}
}

func TestDBDirectPoolCloseAll(t *testing.T) {
	p := newDBDirectPool()
	for _, k := range []string{"a", "b"} {
		_, rel, err := p.acquire(k, func() (*DirectQuery, error) { return token(), nil })
		if err != nil {
			t.Fatalf("不该报错: %v", err)
		}
		rel()
	}
	p.closeAll()
	if len(p.m) != 0 {
		t.Errorf("closeAll 之后池应为空，剩 %d 条", len(p.m))
	}
}

// 配置热替换时**在飞的请求不遭殃**：连接等它 release 再关。
func TestDBDirectPoolCloseAllKeepsInFlight(t *testing.T) {
	p := newDBDirectPool()
	q, rel, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })

	p.closeAll() // 相当于用户此刻在设置页点了一次保存
	if q.closed.Load() {
		t.Fatal("还有请求在用这条连接，closeAll 不能现在就关掉它")
	}
	if len(p.m) != 0 {
		t.Fatal("closeAll 之后不该再发放旧连接")
	}
	rel()
	if !q.closed.Load() {
		t.Fatal("release 之后应把退役的连接关掉")
	}

	// 下一次请求拿新连接
	q2, rel2, _ := p.acquire("k", func() (*DirectQuery, error) { return token(), nil })
	rel2()
	if q2 == q {
		t.Fatal("作废的连接不能又被发出来")
	}
}

func TestDBDirectPoolConcurrentOpenKeepsOne(t *testing.T) {
	p := newDBDirectPool()
	var opened int
	var mu sync.Mutex
	open := func() (*DirectQuery, error) {
		mu.Lock()
		opened++
		mu.Unlock()
		time.Sleep(5 * time.Millisecond) // 让并发请求都走到"自己开一条"
		return token(), nil
	}
	var wg sync.WaitGroup
	got := make([]*DirectQuery, 8)
	for i := range got {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q, rel, err := p.acquire("k", open)
			if err == nil {
				got[i] = q
				rel()
			}
		}(i)
	}
	wg.Wait()
	if len(p.m) != 1 {
		t.Fatalf("并发后池里应只有一条，实际 %d", len(p.m))
	}
	for i, q := range got {
		if q != got[0] {
			t.Errorf("第 %d 个调用拿到的不是同一条连接", i)
		}
	}
}

func TestDirectSQLFallsBackWhenNoDBConfigured(t *testing.T) {
	// 没挂库（测试里构造的 Server 就是这种）：直连这条路必须说"我不回答"，
	// 让调用方走原来的 SSH 路 —— 否则 /api/dbsql 的拨号失败契约会变。
	s := &Server{cfg: &Config{}, dbPool: newDBDirectPool()}
	if _, _, handled := s.directSQL(context.Background(), 99, 30, "select 1 from dual"); handled {
		t.Fatal("没挂库时不该由直连回答")
	}
}

func TestDirectSQLFallsBackWhenPoolMissing(t *testing.T) {
	// 直接字面量构造的 Server 可能没有池（老测试）：不能空指针崩，应当回退。
	s := &Server{cfg: &Config{}}
	if _, _, handled := s.directSQL(context.Background(), 99, 30, "select 1 from dual"); handled {
		t.Fatal("没有池时不该由直连回答")
	}
}
