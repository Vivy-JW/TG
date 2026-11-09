package config

import (
	"strings"
	"testing"
)

// 看门狗是唯一会"静默关掉防行锁保护"的开关：0 会被 FillDefaults 补成默认 1800，
// 但负数会原样留着，而运行时的门是 <= 0 —— 于是 -1 等于关掉，配置里却看不出异常。
func TestWatchdogProblems(t *testing.T) {
	root := func(dbg int, envVals ...int) *Root {
		r := &Root{Debug: DebugSettings{WatchdogSeconds: dbg}}
		for _, v := range envVals {
			r.Hosts.SSHs = append(r.Hosts.SSHs, NamedSsh{Name: "甲", WatchdogSeconds: v})
		}
		return r
	}

	// 缺省与正常值都不该报
	for _, dbg := range []int{0, 300, 1800} {
		if p := WatchdogProblems(root(dbg, 0, 60)); len(p) != 0 {
			t.Errorf("debug=%d 不该报问题：%v", dbg, p)
		}
	}

	// debug 级负数 → 一条，且说清后果
	if p := WatchdogProblems(root(-1)); len(p) != 1 || !strings.Contains(p[0], "关闭看门狗") {
		t.Errorf("debug 级负数应报出一条“关闭看门狗”：%v", p)
	}

	// 环境级负数 → 指名是哪个环境（现场多环境时认不出是谁就白报）
	p := WatchdogProblems(root(0, 0, -5))
	if len(p) != 1 || !strings.Contains(p[0], "甲") {
		t.Errorf("环境级负数应指名环境：%v", p)
	}
}

// 配置层与运行时口径必须分开：配置里的 0 是"没写"（会被补成 1800），
// 只有负数才是"关"。把 0 也判成关闭会让"写 0 恢复默认"变成无法表达。
func TestWatchdogDisabledOnlyNegative(t *testing.T) {
	for _, v := range []int{0, 1, 1800} {
		if WatchdogDisabled(v) {
			t.Errorf("%d 不该算关闭（0 = 没写，会被补成默认）", v)
		}
	}
	if !WatchdogDisabled(-1) {
		t.Error("负数应当算关闭")
	}
}
