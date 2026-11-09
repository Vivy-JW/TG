package debug

// 这组测试盯的是 probe 的**收尾判据**（`probe_verdict.go`）。
//
// 为什么值得单独守：`tt debug probe` 本身要真 SSH + 真 fgldb，进不了单测，于是这条命令
// 曾经**没有人守**——它上一版把"程序自己跑完"当成错误，前六步全绿也照样 `exit 1`，
// 而人看到失败只会去查 SSH 和断点。判据抽成纯函数之后，这一段就能被钉住了。

import (
	"errors"
	"strings"
	"testing"

	"tt/internal/debug"
	"tt/internal/testkit"
)

// 六步全成、程序在步进之前自己跑完 ⇒ **通过**，但要把没覆盖的两步如实报出来。
//
// 这正是真机上的那一种（bsft001_wf + 没有 GDC 在听）：修前它 exit 1，修后它 exit 0。
func TestProbeVerdictProgramExitedAfterChainStillPasses(t *testing.T) {
	reached := []string{probeLogin, probeEntry, probeBreak, probeRun, probeSource, probeWhere}

	ok, missing := probeVerdict(reached)
	if !ok {
		t.Fatal("前六步全走通 + 程序自己跑完，必须判通过：这不是链路故障")
	}
	if len(missing) != 2 || missing[0] != probeStep || missing[1] != probePrint {
		t.Fatalf("没覆盖的应恰好是步进与求值（按固定顺序），实得 %v", missing)
	}
}

// 六步 + 步进 + 求值全成 ⇒ 通过且什么都不缺。
func TestProbeVerdictFullRunPasses(t *testing.T) {
	reached := []string{
		probeLogin, probeEntry, probeBreak, probeRun, probeSource, probeWhere,
		probeStep, probePrint,
	}

	ok, missing := probeVerdict(reached)
	if !ok || len(missing) != 0 {
		t.Fatalf("全走通应判通过且无缺失，实得 ok=%v missing=%v", ok, missing)
	}
}

// 前六步里**任意**一步没真走通 ⇒ 不通过。
//
// 这条防的是"把程序跑完当成功"顺手放宽成"什么都算成功"：漏一步链路必须红。
func TestProbeVerdictAnyMissingChainStepFails(t *testing.T) {
	all := []string{probeLogin, probeEntry, probeBreak, probeRun, probeSource, probeWhere}

	for i, drop := range all {
		reached := make([]string, 0, len(all)-1)
		for j, s := range all {
			if j != i {
				reached = append(reached, s)
			}
		}
		ok, missing := probeVerdict(reached)
		if ok {
			t.Errorf("漏掉 %q 时不该判通过", drop)
		}
		found := false
		for _, m := range missing {
			if m == drop {
				found = true
			}
		}
		if !found {
			t.Errorf("漏掉 %q 时缺失清单里应点名它，实得 %v", drop, missing)
		}
	}
}

// 一步都没走通 ⇒ 不通过，且六个链路步骤全在缺失清单里。
func TestProbeVerdictNothingReachedFails(t *testing.T) {
	ok, missing := probeVerdict(nil)
	if ok {
		t.Fatal("一步都没走通不可能判通过")
	}
	if len(missing) != len(probeRequired)+len(probeOptional) {
		t.Fatalf("缺失清单应是全部 %d 步，实得 %d 条：%v",
			len(probeRequired)+len(probeOptional), len(missing), missing)
	}
}

// "程序自己跑完"的识别：认文本、也认会话状态兜底；**真实故障不许被认成这个**。
func TestProbeProgramExited(t *testing.T) {
	cases := []struct {
		name string
		err  error
		st   debug.State
		want bool
	}{
		{"会话层那句 program exited", errors.New("program exited"), debug.StateIdle, true},
		{"文本被改写但状态已回落空闲", errors.New("命令未完成"), debug.StateIdle, true},
		{"该轮调试已结束", errors.New("命令未完成"), debug.StateExit, true},
		{"真正的命令失败", errors.New("No symbol table is loaded"), debug.StateStopped, false},
		{"没有错误就不算程序退出", nil, debug.StateIdle, false},
	}
	for _, c := range cases {
		if got := probeProgramExited(c.err, c.st); got != c.want {
			t.Errorf("%s：probeProgramExited = %v，要 %v", c.name, got, c.want)
		}
	}
}

// 收尾清单必须把两件事分开说：**链路故障**（✗）与**程序跑完没覆盖**（-）。
//
// 分不开就是这次要修的那个毛病：人看到 exit 1 会去查 SSH，而真因是作业跑完了。
func TestPrintProbeVerdictSeparatesFailureFromUncovered(t *testing.T) {
	reached := []string{probeLogin, probeEntry, probeBreak, probeRun, probeSource, probeWhere}
	ok, missing := probeVerdict(reached)

	_, out := testkit.CaptureStdout(t, func() int {
		printProbeVerdict(reached, missing, ok, probeStep)
		return 0
	})

	if !strings.Contains(out, "链路通过") {
		t.Errorf("六步全通时要说清链路通过：\n%s", out)
	}
	if !strings.Contains(out, "不是链路故障") {
		t.Errorf("步进/求值没覆盖时要说清「不是链路故障」：\n%s", out)
	}
	if strings.Contains(out, "✗") {
		t.Errorf("这种情形不该出现链路故障标记 ✗：\n%s", out)
	}
}
