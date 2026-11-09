package tzc

// `tt dev tzc uuid` 的单测：格式（16 位小写十六进制）、不重复、命令输出与 --json。
// 这个工具只在"用户拒绝提供单号"时用，格式错了就等于标记不可追溯，所以钉死。

import (
	"regexp"
	"strings"
	"testing"

	"tt/internal/testkit"
)

var reHex16 = regexp.MustCompile(`^[0-9a-f]{16}$`)

func TestNewModUUIDFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 20; i++ {
		id, err := newModUUID()
		if err != nil {
			t.Fatal(err)
		}
		if !reHex16.MatchString(id) {
			t.Fatalf("id = %q，要 16 位小写十六进制", id)
		}
		if seen[id] {
			t.Fatalf("两次生成了同一个 id：%q", id)
		}
		seen[id] = true
	}
}

func TestUUIDCommand(t *testing.T) {
	code, out := testkit.CaptureStdout(t, func() int { return cmdUUID(nil) })
	if code != 0 {
		t.Fatalf("退出码 %d", code)
	}
	id := strings.TrimSpace(out)
	if !reHex16.MatchString(id) {
		t.Fatalf("裸输出应为 16 位十六进制，实际 %q", id)
	}

	code, out = testkit.CaptureStdout(t, func() int { return cmdUUID([]string{"--json"}) })
	if code != 0 {
		t.Fatalf("--json 退出码 %d", code)
	}
	if !strings.Contains(out, `"id":"`) || !strings.Contains(out, `"ok":true`) {
		t.Errorf("--json 输出不对：%s", out)
	}

	// 多余位置参数 = 用法错（退出码 2），不能悄悄忽略
	if code, _ := testkit.CaptureStdout(t, func() int { return cmdUUID([]string{"extra"}) }); code != 2 {
		t.Errorf("多余参数应退 2，实际 %d", code)
	}
}
