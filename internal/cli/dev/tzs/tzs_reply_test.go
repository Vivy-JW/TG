package tzs

// tzs_reply_test.go —— 两帧渲染的测试：`printHumanReply` / `printWireDetail`。
//
// 这一层**不碰引擎**：帧是手工造的（`*tzs.Reply` 就是个普通结构体），
// 不读配置、不连管道。tzs_verb_test.go 开头立的规矩在这里同样成立：
// "一条依赖本机配置的测试会时好时坏，那比不测更糟"。
//
// 造帧用的形状**照引擎实际产出的抄**（出处逐条写在用例里）：这些渲染器的全部价值就是
// "把引擎已经算出来的东西送出去"，用编出来的形状测等于没测。

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"tt/internal/dev/tzs"
)

// capture 把 stdout/stderr 换成同一根管道，返回 fn 期间的输出（两股合在一起）。
//
// 与 corpus_test.go 的 `silent`（丢到 DevNull）是两件事：这里要**看见**输出。
// 读端必须放在 goroutine 里 —— 管道缓冲区满了之后写端会阻塞，同 goroutine 读就是死锁。
func capture(t *testing.T, fn func()) string {
	t.Helper()
	out, errOut := captureBoth(t, fn)
	return out + errOut
}

// captureBoth 分别捕获 stdout 与 stderr。
//
// 要断言"这句话进的是哪一股流"时必须分开：合在一起就分不出"stdout 干净"与"两股都在说话"，
// 而 `--json` 的契约恰恰是"stdout 恰好一帧"——那是以 stdout 单独为对象的断言。
func captureBoth(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()
	ro, wo, err := os.Pipe()
	if err != nil {
		t.Fatalf("建 stdout 管道失败：%v", err)
	}
	re, we, err := os.Pipe()
	if err != nil {
		t.Fatalf("建 stderr 管道失败：%v", err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = wo, we
	gotOut, gotErr := make(chan string, 1), make(chan string, 1)
	go func() { b, _ := io.ReadAll(ro); gotOut <- string(b) }()
	go func() { b, _ := io.ReadAll(re); gotErr <- string(b) }()

	fn()

	os.Stdout, os.Stderr = oldOut, oldErr
	wo.Close()
	we.Close()
	stdout, stderr = <-gotOut, <-gotErr
	ro.Close()
	re.Close()
	return stdout, stderr
}

func frameError(code, kind, msg string, detail string) *tzs.Reply {
	r := &tzs.Reply{OK: false, Error: &tzs.WireError{Code: code, Kind: kind, Message: msg}}
	if detail != "" {
		r.Error.Detail = json.RawMessage(detail)
	}
	return r
}

// TestPrintWireDetail 是渲染器的主体测试：形状照引擎的产出抄。
func TestPrintWireDetail(t *testing.T) {
	cases := []struct {
		name   string
		detail string
		msg    string // error.message（用于"与它重复的 message 键要跳过"）
		want   []string
		absent []string
	}{
		{
			// Fns/Attr.cs:81-84,107-112 —— 属性值不在设计器声明的集合里。
			// 不带 `value` ⟹ `legal` 是**属性名集**，标签用"可选项"
			name:   "可选项 + 近似值",
			detail: `{"path":"managedform/a/HBoxT1/x","legal":["none","lower","upper"],"hint":"upper"}`,
			want:   []string{"位置：managedform/a/HBoxT1/x", "可选项：", "- none", "- upper", "提示：upper"},
			absent: []string{"合法值"},
		},
		{
			// 真机实测的载荷（aapt300(c).tzs 的 HBoxT1 上写 hidden=MAYBE）：
			// 带 `value` ⟹ `legal` 是**值集**，标签是"合法值"
			name:   "值集拒绝",
			detail: `{"path":"managedform/aapt300/HBoxT1","attr":"hidden","value":"MAYBE","legal":["false","true"],"type":"ENUM","source":"mta/mod-fd.spec","written":false}`,
			want: []string{
				"位置：managedform/aapt300/HBoxT1", "属性：hidden", "值：MAYBE",
				"合法值：", "- false", "- true",
				"类型：ENUM", "依据：mta/mod-fd.spec", "写入的值：false",
			},
		},
		{
			// 引擎侧 `out` 闸门的拒绝（2026-09-24 加）：detail 带 param/reason/value/conflict
			name:   "out 闸门的拒绝",
			detail: `{"param":"out","reason":"out-is-open-package","value":"C:/ws/a.tzs","conflict":"C:/ws/a.tzs"}`,
			want:   []string{"参数：out", "原因：out-is-open-package", "冲突的路径：C:/ws/a.tzs"},
		},
		{
			// 同样真机实测（同一个元素上写 case，它没有这个属性）：
			// 不带 value ⟹ 属性名集。写成"合法值"会被读成"case 的合法值是 tag/posX/…"
			name:   "属性名集拒绝",
			detail: `{"path":"managedform/aapt300/HBoxT1","attr":"case","legal":["tag","posX","posY"]}`,
			want:   []string{"属性：case", "可选项：", "- tag", "- posX"},
			absent: []string{"合法值"},
		},
		{
			// Rpc.cs:739-749 + OpenHandles Rpc.cs:468-484 —— 候选是**对象数组**，
			// 每个对象里 key 才是"该拿去重试的那个名字"。渲染器不认识 key 这个名字，
			// 它只是把对象原样紧凑打出来 —— 那是**故意的**（见 tzs_detail.go 顶部的说明）。
			name:   "候选是对象数组时原样打出来",
			detail: `{"param":"handle","reason":"ambiguous","candidates":[{"handle":"h3","program":"aapp320","key":"aapp320|Form","path":"D:\\pkg\\a.tzs","mutable":true}]}`,
			want:   []string{"参数：handle", "候选（key 可直接拿去重试）：", `"key":"aapp320|Form"`},
		},
		{
			// Fns/Semantic.cs:537,657 / Fns/Action.cs:146-154 —— 候选也可能是**字符串数组**
			name:   "候选是字符串数组时逐条列出",
			detail: `{"candidates":["adzi999","adzi1000"]}`,
			want:   []string{"候选（key 可直接拿去重试）：", "- adzi999", "- adzi1000"},
			absent: []string{`"adzi999"`}, // 字符串不该带引号
		},
		{
			// Fns/Attr.cs:609-616 —— E_ATTR_PARTIAL：哪几个落了、哪几个没落
			name:   "复数写入写了一半",
			detail: `{"path":"managedform/a/x","applied":["can_edit"],"failed":["can_query"],"retry":["can_query"]}`,
			want:   []string{"已写入：", "- can_edit", "失败：", "- can_query", "可重试：", "- can_query"},
		},
		{
			// 表里没有的键：按原名兜底打出来 —— 这就是"引擎加键不用改 Go"
			name:   "没见过的键按原名打出来",
			detail: `{"brandNewKey":"whatever","another":7}`,
			want:   []string{"another：7", "brandNewKey：whatever"},
		},
		{
			name:   "与 error.message 重复的 message 键要跳过",
			detail: `{"message":"同一句话","param":"path"}`,
			msg:    "同一句话",
			want:   []string{"参数：path"},
			absent: []string{"说明："},
		},
		{
			name:   "message 键与 error.message 不同时照打",
			detail: `{"message":"detail 里的另一句话"}`,
			msg:    "外面那句",
			want:   []string{"说明：detail 里的另一句话"},
		},
		{
			name:   "空串不打（它表示没设/继承）",
			detail: `{"reason":""}`,
			absent: []string{"原因："},
		},
		{
			name:   "空对象什么都没得打",
			detail: `{}`,
			absent: []string{"：", "- "},
		},
		{
			// 引擎没发过这种，但真发了也不该在这里死
			name:   "detail 不是对象时原样打出来",
			detail: `"boom"`,
			want:   []string{"detail（原样）：\"boom\""},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := &tzs.WireError{Code: "E_ATTR_VALUE_ILLEGAL", Kind: "validation", Message: c.msg}
			e.Detail = json.RawMessage(c.detail)
			var buf bytes.Buffer
			printWireDetail(&buf, e)
			out := buf.String()
			for _, w := range c.want {
				if !strings.Contains(out, w) {
					t.Errorf("该出现 %q，实际：\n%s", w, out)
				}
			}
			for _, a := range c.absent {
				if strings.Contains(out, a) {
					t.Errorf("不该出现 %q，实际：\n%s", a, out)
				}
			}
		})
	}
}

// TestPrintWireDetailNilAndEmpty —— 没有 detail 时一个字都不打（这是常态，不是异常）。
func TestPrintWireDetailNilAndEmpty(t *testing.T) {
	for _, e := range []*tzs.WireError{
		nil,
		{Code: "E_DESIGNER", Kind: "designer", Message: "设计器拒绝"},
		{Code: "E_DESIGNER", Kind: "designer", Message: "设计器拒绝", Detail: json.RawMessage(`null`)},
	} {
		var buf bytes.Buffer
		printWireDetail(&buf, e)
		if buf.Len() != 0 {
			t.Errorf("没有 detail 时不该有输出，实际：%q", buf.String())
		}
	}
}

// TestPrintWireDetailCapsLists —— 列表封顶，而且**封顶这件事要说出来**。
//
// 与本项目"截断绝不静默"是同一条：少给的东西必须让读的人知道少了。
func TestPrintWireDetailCapsLists(t *testing.T) {
	var items []string
	for i := 0; i < maxDetailItems+5; i++ {
		items = append(items, "item"+strconv.Itoa(i))
	}
	detail, _ := json.Marshal(map[string]any{"legal": items})

	e := &tzs.WireError{Code: "E_ATTR_VALUE_ILLEGAL", Kind: "validation", Detail: detail}
	var buf bytes.Buffer
	printWireDetail(&buf, e)
	out := buf.String()

	if strings.Count(out, "\n      - ") != maxDetailItems {
		t.Errorf("该只打 %d 条，实际：\n%s", maxDetailItems, out)
	}
	if !strings.Contains(out, "还有 5 个没打") {
		t.Errorf("封顶必须说出来（还要指出 --json 能看全），实际：\n%s", out)
	}
}

// TestHumanReplyRendersDetailOnError 默认（非 --json）模式也要看得见 detail。
//
// 这是这次改动的全部理由：引擎做 legal/hint/candidates 就是为了让调用方自己纠正，
// 而原先只有加 --json 才看得见 —— 等于没做。
func TestHumanReplyRendersDetailOnError(t *testing.T) {
	r := frameError("E_ATTR_VALUE_ILLEGAL", "validation", "属性值不在合法集里",
		`{"path":"managedform/a/x","legal":["none","lower","upper"],"hint":"upper"}`)

	var got int
	out := capture(t, func() { got = printHumanReply("set_layout_attr", r, nil, tzs.ExitUsage) })

	for _, want := range []string{
		"E_ATTR_VALUE_ILLEGAL (validation): 属性值不在合法集里",
		"可选项：", "- upper", "提示：upper",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("该出现 %q，实际：\n%s", want, out)
		}
	}
	if got != tzs.ExitUsage {
		t.Errorf("退出码该原样返回 %d，得 %d", tzs.ExitUsage, got)
	}
}

// TestHumanReplyNoOpSaysSo —— W3：E_NO_OP 是"什么也没改"，文案要说人话。
//
// **退出码不动**：契约表把它的 kind 归成 internal → 1（见 client.go 的 CodeNoOp 注释）。
// 这条现在守的是**兜底**：2026-09-25 起引擎没有路径再把成功码发成错误帧了 —— 最后一处
// （set_local_string / set_spec_description 的 NoOp 分支）已按 SPEC §11.24(a) 改成成功帧，
// 走下面 TestHumanReplySuccessPathUnchanged 那条路。留着是因为旧引擎仍可能在跑
// （守护进程按 MVID 命名）。
func TestHumanReplyNoOpSaysSo(t *testing.T) {
	// ① 契约层面：帧 → 退出码仍然是 1（这条判据没被文案改动带偏）
	noop := frameError("E_NO_OP", "internal", "set_local_string: 内容已经是目标值", "")
	if got := tzs.ExitCode(noop, nil); got != tzs.ExitFrameErr {
		t.Fatalf("E_NO_OP 的退出码该仍是 %d（契约表），得 %d", tzs.ExitFrameErr, got)
	}

	// ② 文案层面：说清"不是错误"
	var got int
	out := capture(t, func() { got = printHumanReply("set_local_string", noop, nil, tzs.ExitFrameErr) })
	if !strings.Contains(out, "什么都没改") {
		t.Errorf("E_NO_OP 该被说成「什么也没改」，实际：\n%s", out)
	}
	if got != tzs.ExitFrameErr {
		t.Errorf("文案改了，退出码不该跟着变：该 %d，得 %d", tzs.ExitFrameErr, got)
	}

	// ③ 对照：真的内部错不该拿到那句人话
	internal := frameError("E_INTERNAL", "internal", "未预期异常", "")
	out = capture(t, func() { printHumanReply("open", internal, nil, tzs.ExitFrameErr) })
	if strings.Contains(out, "什么都没改") {
		t.Errorf("E_INTERNAL 不该被说成「什么也没改」，实际：\n%s", out)
	}
}

// TestHumanReplySuccessPathUnchanged —— 成功帧的打法一个字没动。
//
// 为什么值得一条：E_NO_OP / E_ATTR_CLAMPED **正常情况下**是以 ok:true + result.code
// 回来的（Fns/Attr.cs 的几处就是这样），走的就是这个分支。那里 code/noop/clamped
// 都在 JSON 里、看得见，所以这一支**刻意不加**任何"这不是错误"的措辞
// （见 printHumanReply 里的注释）—— 这条测试把"刻意"钉住。
func TestHumanReplySuccessPathUnchanged(t *testing.T) {
	r := &tzs.Reply{OK: true, Result: json.RawMessage(`{"code":"E_NO_OP","noop":true,"changed":false}`)}
	var got int
	out := capture(t, func() { got = printHumanReply("set_layout_attr", r, nil, tzs.ExitOK) })

	if !strings.HasPrefix(out, "set_layout_attr: ") {
		t.Errorf("成功帧该打 `动词: {result}`，实际：\n%s", out)
	}
	if !strings.Contains(out, `"noop":true`) {
		t.Errorf("result 该原样打出来（noop/code 都在里面），实际：\n%s", out)
	}
	if strings.Contains(out, "这不是错误") {
		t.Errorf("成功分支不该加 no-op 的措辞（code 已经在 JSON 里了），实际：\n%s", out)
	}
	if got != tzs.ExitOK {
		t.Errorf("成功帧该退 %d，得 %d", tzs.ExitOK, got)
	}
}
