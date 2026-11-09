package debug

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// 每一条**造会话**的路径都必须按发起方设模式（launchModeFor）。
//
// 事故背景（用户 2026-10-16 报"点了没反应"）：/api/sessions/switch 这条路漏了
// SetMode(launchModeFor(r))，新建出来的会话保留 NewSession 的初始值 —— 业主授权
// （config.aiTakeover=true）时那是 **collab**，于是流水线下一步（`debug topent`）以"人"
// 的身份被"协作模式:本次调试由 AI 主导"拒掉，操作静默失败。hSessionRestart 是同一天
// 发现的第二处。（launchModeFor 的注释早写着"两条启动路径必须都过这里"，实际有四条。）
//
// 这条测试把"必须都过"变成断言：以后再加一条造会话的路径、忘了设模式，就是红的。
// 与 testkit/single_source_test.go 同一路数：把注释里的纪律写成断言。
func TestEverySessionCreatePathSetsMode(t *testing.T) {
	src, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatalf("读不到 api.go: %v", err)
	}
	lines := strings.Split(string(src), "\n")

	// 造会话的入口（manager.go 里能 new 出 Session 的那几个）
	creates := []string{"LaunchWith(", "CreateSessionOn(", "LaunchReplay(", "mgr.Launch("}
	const lookahead = 20 // 设模式那句得跟在造会话后面不远处

	var missing []string
	for i, ln := range lines {
		if strings.HasPrefix(strings.TrimSpace(ln), "//") {
			continue
		}
		hit := ""
		for _, c := range creates {
			if strings.Contains(ln, c) {
				hit = c
				break
			}
		}
		if hit == "" {
			continue
		}
		ok := false
		for j := i; j < len(lines) && j < i+lookahead; j++ {
			if strings.Contains(lines[j], "SetMode(launchModeFor(") {
				ok = true
				break
			}
		}
		if !ok {
			missing = append(missing, fmt.Sprintf("api.go:%d  %s（用了 %s）", i+1, strings.TrimSpace(ln), hit))
		}
	}

	if len(missing) > 0 {
		t.Fatalf("这些造会话的地方没有按发起方设模式（漏了的后果：业主授权时新会话默认 collab，"+
			"人在界面上的动作会被自己的闸门拒掉，而且**静默**）:\n  - %s\n"+
			"修法：紧跟着写 %s", strings.Join(missing, "\n  - "), "sess.SetMode(launchModeFor(r))")
	}
}
