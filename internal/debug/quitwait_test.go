package debug

import (
	"strings"
	"testing"
	"time"
)

// 收口的"执行者 / 等待者"协议：后来者必须**等这一次收口做完**，而不是立刻回 OK。
//
// 事故背景（用户 2026-10-16 报"点了没反应"）：前一条命令（另一个进程的
// `tt debug quit`）与另一条命令的 quit 撞在一起，后来者拿到"ok"就往下走，
// 紧跟着的 `debug topent` 撞上"仅会话空闲(idle)时可设置 TOPENT(当前 运行中)"，
// 整条操作静默失败 —— 界面上只看到"点了没反应"。
func TestBeginQuitSharesOneTeardown(t *testing.T) {
	s := &Session{state: StateLoading}

	done, mine := s.beginQuit()
	if !mine {
		t.Fatal("第一个调用者应当是执行者")
	}

	done2, mine2 := s.beginQuit()
	if mine2 {
		t.Fatal("第二个调用者不该成为执行者（否则两次收口会打架）")
	}
	if done2 != done {
		t.Fatal("等待者必须等**同一个**信号，否则等的是一个永远不会关闭的 chan")
	}

	// 执行者做完（这里直接摆成 idle 态）并关闭信号 → 等待者才返回，且返回后状态可干活。
	// 直接改 state 而不走 forceExit/enterIdle：那两个要真 PTY 与 emit 回调，这里只验协议本身。
	go func() {
		time.Sleep(150 * time.Millisecond)
		s.mu.Lock()
		s.state = StateIdle
		s.mu.Unlock()
		close(done)
	}()
	start := time.Now()
	if err := s.waitQuit(done2); err != nil {
		t.Fatalf("等待者该成功返回: %v", err)
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("等待者没有真的等那次收口（立刻返回正是这个 bug 的形态）")
	}
	if st := s.State(); st == StateRunning || st == StateLoading {
		t.Fatalf("等待返回时状态还不可用: %s", st)
	}
}

// 收口做完之后，**下一轮必须能重新拿到执行权**。
//
// 事故背景（2026-10-23，用户报"点了重新开始，步入/步过全灰"）：`quitting` 从前只置
// true、**从不复位**，于是每条会话的「结束调试 / 重新开始」只有**第一次**真的生效。
// 之后再点，`beginQuit` 一律返回 `!mine` + 一个**已经关闭**的 done，`waitQuit` 立刻
// 返回 nil（报成功），状态却留在 stopped —— 紧接着的启动撞"已存在调试会话"，
// 界面停在 idle，人看到的就是"步进整排变灰"。
//
// 这条用例走**真实入口** `withQuit`（不是手工调 begin/end），所以它能抓住"调用方
// 忘了复位"这一类错误：把 withQuit 里的 `defer endQuit` 去掉，它立刻红（calls 停在 1）。
// 只测一次 begin/end 是发现不了这个 bug 的 —— 这也是上一轮"全绿却仍有 bug"的原因之一。
func TestWithQuitRunsEveryRound(t *testing.T) {
	s := &Session{state: StateStopped}
	calls := 0
	for i := 0; i < 3; i++ {
		if err := s.withQuit(func() error { calls++; return nil }); err != nil {
			t.Fatalf("第 %d 轮: %v", i+1, err)
		}
	}
	if calls != 3 {
		t.Fatalf("收口只执行了 %d 次（应当每轮一次）—— quitting 没有复位就会这样", calls)
	}
}

// 收口做完但状态仍是 running/loading（收口没生效）时要说出来，不能当成功。
func TestWaitQuitReportsIneffectiveTeardown(t *testing.T) {
	s := &Session{state: StateLoading}
	done := make(chan struct{})
	close(done)
	err := s.waitQuit(done)
	if err == nil {
		t.Fatal("状态还是 loading 时不该报成功")
	}
	if !strings.Contains(err.Error(), "结束调试") {
		t.Fatalf("错误文案要说清是哪一步没生效：%v", err)
	}
}
