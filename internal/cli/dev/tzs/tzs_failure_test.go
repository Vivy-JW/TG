package tzs

// tzs_failure_test.go —— 失败出口的契约：`--json` 下 stdout **恰好一帧**。
//
// 这一层不碰引擎：失败是手工造的（`*tzs.TransportError` / `*tzs.UsageError` 都是普通结构体），
// 走的是真代码路径 `emitFailure`，只是没有守护进程在另一头。
//
// 规则与三处例外写在 `emitFailure` 的注释里；这里钉的是规则本身能机械验的那一半：
//
//	stdout 上恰好一个 JSON 对象、含 ok、ok:false 时含 error.code；退出码与错误自己的退出码一致。
//
// 从前这条不成立的地方正是那个缺陷：传输失败时 `json.Marshal(nil)` 往 stdout 打了个
// 字面量 `null` —— 看着像 JSON、`jq .ok` 会报错、`$(...)` 的消费方会静默丢掉失败。

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"tt/internal/dev/tzs"
)

func TestJsonFailureIsExactlyOneFrame(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
		exit int
	}{
		{"传输失败（请求可能已经上线）",
			&tzs.TransportError{Code: tzs.CodeServerDied, Msg: "守护进程没有应答就退出了"},
			tzs.CodeServerDied, tzs.ExitTransport},
		{"本地参数错",
			&tzs.UsageError{Msg: "未知函数 foo", Detail: []string{"是不是想写 bar"}},
			tzs.CodeBadRequest, tzs.ExitUsage},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got int
			stdout, stderr := captureBoth(t, func() { got = emitFailure(c.err, true) })

			if strings.TrimSpace(stdout) == "null" {
				t.Fatalf("stdout 上是字面量 null —— 看着像 JSON、其实什么都没说（这正是要修的那个缺陷）")
			}
			dec := json.NewDecoder(strings.NewReader(stdout))
			var f tzs.Reply
			if err := dec.Decode(&f); err != nil {
				t.Fatalf("stdout 不是一帧 JSON（%q）：%v", stdout, err)
			}
			if f.OK || f.Error == nil {
				t.Errorf("该是一帧 ok:false 的帧：%s", stdout)
			}
			if f.Error.Code != c.code {
				t.Errorf("code 想要 %s，得 %s", c.code, f.Error.Code)
			}
			if f.Error.Message == "" {
				t.Error("帧里要有 message —— 人读的就是它")
			}
			// "恰好一帧"的机械形态：再解一次必须是 EOF。
			var extra any
			if err := dec.Decode(&extra); err != io.EOF {
				t.Errorf("stdout 上不止一帧（第二次 Decode 得 %v）：%q", err, stdout)
			}

			if got != c.exit {
				t.Errorf("退出码想要 %d，得 %d", c.exit, got)
			}
			// --json 时诊断进帧，不该再往 stderr 抄一份
			if strings.TrimSpace(stderr) != "" {
				t.Errorf("--json 模式下 stderr 该是干净的，得：%q", stderr)
			}
		})
	}
}

// TestHumanFailureGoesToStderrOnly 非 --json 时反过来：stdout 一个字都不该有。
//
// 两条一起才说明"分股"这件事是**判据**，不是巧合。
func TestHumanFailureGoesToStderrOnly(t *testing.T) {
	var got int
	stdout, stderr := captureBoth(t, func() {
		got = emitFailure(&tzs.UsageError{Msg: "未知函数 foo"}, false)
	})
	if stdout != "" {
		t.Errorf("非 --json 时 stdout 该是空的，得：%q", stdout)
	}
	if !strings.Contains(stderr, "未知函数 foo") {
		t.Errorf("人话该进 stderr，得：%q", stderr)
	}
	if !strings.Contains(stderr, "退出码 2") {
		t.Errorf("人话里该点出退出码，得：%q", stderr)
	}
	if got != tzs.ExitUsage {
		t.Errorf("退出码想要 %d，得 %d", tzs.ExitUsage, got)
	}
}
