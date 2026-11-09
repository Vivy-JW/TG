package tzs

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

//---------------------------------------------------------------------------
// BuildArgsFromJSON：唯一的参数入口
//
// 命令面不接受 `--<参数> <值>`（见 argmap.go 的文件头），所以这里就是**全部**的
// 参数判据。表里既有"必须本地拦住"的，也有"**绝不能**本地拦住"的反向断言 ——
// 后者更重要：越界拦截是**无声**发生的，写错了不会有任何测试失败，
// 只会让某些合法调用永远到不了引擎。
//---------------------------------------------------------------------------

func TestBuildArgsFromJSONTable(t *testing.T) {
	cases := []struct {
		name    string
		fn      string
		in      string
		want    string
		wantErr string
	}{
		// ---- 定型 ----
		{
			"数组 + 枚举 + 数字", "nudge",
			`{"handle":"h1","paths":["a","b"],"direction":"up","offset":2}`,
			`{"handle":"h1","paths":["a","b"],"direction":"up","offset":2}`, "",
		},
		{
			"负数", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","offset":-3}`,
			`{"handle":"h1","paths":["a"],"direction":"up","offset":-3}`, "",
		},
		{
			"选填缺席", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up"}`,
			`{"handle":"h1","paths":["a"],"direction":"up"}`, "",
		},
		{
			"选填给 null = 缺席", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","offset":null}`,
			`{"handle":"h1","paths":["a"],"direction":"up"}`, "",
		},
		{
			"空对象", "list_open", `{}`, `{}`, "",
		},
		// ---- attrs：一次改多个属性的那个对象 ----
		{
			"attrs：值都是字符串，键排一次序（重发的字节才稳定）", "set_spec_attrs",
			`{"handle":"h1","path":"p","kind":"field","attrs":{"can_query":"N","can_edit":"true"}}`,
			`{"handle":"h1","path":"p","kind":"field","attrs":{"can_edit":"true","can_query":"N"}}`, "",
		},
		{
			"attrs：值不是字符串 → 点名那个键", "set_spec_attrs",
			`{"handle":"h1","path":"p","kind":"field","attrs":{"gridWidth":20}}`,
			"", "gridWidth",
		},
		{
			"attrs：空对象 → 指路用单数形式", "set_spec_attrs",
			`{"handle":"h1","path":"p","kind":"field","attrs":{}}`,
			"", "空对象",
		},
		{
			"attrs：写成数组 → 说清要对象", "set_spec_attrs",
			`{"handle":"h1","path":"p","kind":"field","attrs":["a"]}`,
			"", "需要 JSON 对象",
		},
		{
			"没有参数时也 build 出空对象", "list_open", ``, `{}`, "",
		},
		{
			"null 等于空对象", "list_open", `null`, `{}`, "",
		},
		{
			"键序 = manifest 声明序（JSON 里反着给也一样）", "nudge",
			`{"offset":1,"direction":"up","paths":["a"],"handle":"h1"}`,
			`{"handle":"h1","paths":["a"],"direction":"up","offset":1}`, "",
		},
		{
			"空数组", "nudge",
			`{"handle":"h1","paths":[],"direction":"up"}`,
			`{"handle":"h1","paths":[],"direction":"up"}`, "",
		},
		{
			"字符串里的转义被规范化", "save",
			`{"handle":"h1","out":"D:\\x\\y.tzs"}`,
			`{"handle":"h1","out":"D:\\x\\y.tzs"}`, "",
		},
		{
			"string 里的逗号不切", "add_action",
			`{"handle":"h1","path":"p","type":"di3,db4"}`,
			`{"handle":"h1","path":"p","type":"di3,db4"}`, "",
		},
		{
			"string[] 保 |", "set_items",
			`{"handle":"h1","path":"p","items":["lbl_a|A|d1","lbl_b|B|d2"]}`,
			`{"handle":"h1","path":"p","items":["lbl_a|A|d1","lbl_b|B|d2"]}`, "",
		},
		{
			"布尔 true", "set_excluded",
			`{"handle":"h1","path":"p","excluded":true}`,
			`{"handle":"h1","path":"p","excluded":true}`, "",
		},
		{
			"布尔 false", "set_excluded",
			`{"handle":"h1","path":"p","excluded":false}`,
			`{"handle":"h1","path":"p","excluded":false}`, "",
		},
		{
			"值是空串", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":""}`,
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":""}`, "",
		},
		{
			"值是负数字符串", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":"-1"}`,
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":"-1"}`, "",
		},

		// ---- 反向断言：本地**绝不**拦 ----
		{
			// from:"spec:<kind>" 的合法集要现场从模型里取，静态列不出来。
			"attr(from:*) 放行", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"bogus_attr","value":"v"}`,
			`{"handle":"h1","path":"p","kind":"field","attr":"bogus_attr","value":"v"}`, "",
		},
		{
			// kind 的合法集（七种）**不在 manifest 里** —— Go 侧抄一份就是第二份真源。
			"kind 放行", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"alien","attr":"a","value":"v"}`,
			`{"handle":"h1","path":"p","kind":"alien","attr":"a","value":"v"}`, "",
		},
		{
			// add_action 的「型态」是一张**每表单自己的**词表（该表单的 s_detail<n>）。
			"per-form 词表放行", "add_action",
			`{"handle":"h1","path":"p","type":"Bogus"}`,
			`{"handle":"h1","path":"p","type":"Bogus"}`, "",
		},
		{
			// 引擎的 Manifest.Check 对 string 参数的数字/布尔是放行的，
			// 本地凭空加一条引擎没有的规则，就会让合法调用到不了引擎。
			"string 参数收到数字原样透传", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":-1}`,
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":-1}`, "",
		},

		// ---- 本地必须拦 ----
		{"未知函数", "nope", `{}`, "", "未知函数 nope"},
		{"未知函数要把可用清单带出来", "nope", `{}`, "", "nudge"},
		{"未知参数（报全部且排序）", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","zBogus":1,"aBogus":2}`,
			"", "aBogus, zBogus"},
		{"缺必填", "nudge", `{"handle":"h1","paths":["a"]}`, "", "缺必填参数 direction"},
		{"缺必填 handle", "nudge", `{"paths":["a"],"direction":"up"}`, "", "缺必填参数 handle"},
		{"int 收到字符串", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","offset":"2"}`,
			"", "需要整数"},
		{"int 收到小数", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","offset":1.5}`,
			"", "需要整数"},
		{"bool 收到字符串", "set_excluded",
			`{"handle":"h1","path":"p","excluded":"yes"}`,
			"", "需要 true/false"},
		{"枚举越界（列合法值）", "nudge",
			`{"handle":"h1","paths":["a"],"direction":"sideways"}`,
			"", "up|down|left|right"},
		{"数组元素不是字符串", "nudge",
			`{"handle":"h1","paths":["a",2],"direction":"up"}`,
			"", "数组元素必须是字符串"},
		{"数组位置给了标量", "nudge",
			`{"handle":"h1","paths":"a","direction":"up"}`,
			"", "需要字符串数组"},
		{"字符串参数收到对象", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":{"x":1}}`,
			"", "需要一个字符串"},
		{"字符串参数收到数组", "set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"a","value":["x"]}`,
			"", "需要一个字符串"},
		{"args 是数组不是对象", "nudge", `["a"]`, "", "必须是 JSON 对象"},
		{"args 是标量", "nudge", `2`, "", "必须是 JSON 对象"},
		{"args 不是 JSON", "nudge", `{`, "", "不是合法的 JSON 对象"},
		{"stop 不在 manifest 里", "stop", `{}`, "", "传输级命令"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := BuildArgsFromJSON(mustManifest(t), c.fn, json.RawMessage(c.in))
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("不该报错：%v", err)
				}
				if string(got) != c.want {
					t.Fatalf("定型结果不对\n  得 %s\n想 %s", got, c.want)
				}
				if !json.Valid(got) {
					t.Fatalf("产出的不是合法 JSON：%s", got)
				}
				return
			}
			if err == nil {
				t.Fatalf("该报错（含 %q），却得到 %s", c.wantErr, got)
			}
			var ue *UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("该是用法错（退 2），得 %T：%v", err, err)
			}
			if ue.ExitCode() != ExitUsage {
				t.Errorf("该退 2，得 %d", ue.ExitCode())
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("文案里缺 %q：%v", c.wantErr, err)
			}
		})
	}
}

// TestBuildArgsFromJSONOrderIsManifestOrder：输出顺序固定 = manifest 声明顺序。
//
// 固定顺序是为了可复现：同一份输入永远产出同一串字节，日志、--json、测试断言
// 才能逐字节比对（JSON 里键的书写顺序不该影响帧）。
func TestBuildArgsFromJSONOrderIsManifestOrder(t *testing.T) {
	m := mustManifest(t)
	a, err := BuildArgsFromJSON(m, "nudge",
		json.RawMessage(`{"offset":1,"direction":"up","paths":["a"],"handle":"h1"}`))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"handle":"h1","paths":["a"],"direction":"up","offset":1}`
	if string(a) != want {
		t.Errorf("参数顺序没按 manifest：\n  得 %s\n想 %s", a, want)
	}
}

// TestBuildArgsFromJSONReverseRequestCarried 是反向断言的**第二半**：
// 本地放行不算数，还得真的进了请求帧。
//
// 只说「本地没拦」是不够的 —— 一个把值悄悄丢掉的实现也能过那一半。
func TestBuildArgsFromJSONReverseRequestCarried(t *testing.T) {
	m := mustManifest(t)
	cases := []struct {
		fn    string
		in    string
		inReq string // 请求帧里必须原样出现的值
	}{
		{"set_spec_attr",
			`{"handle":"h1","path":"p","kind":"field","attr":"bogus_attr","value":"v"}`,
			`"attr":"bogus_attr"`},
		{"set_spec_attr",
			`{"handle":"h1","path":"p","kind":"alien","attr":"a","value":"v"}`,
			`"kind":"alien"`},
		{"add_action",
			`{"handle":"h1","path":"p","type":"Bogus"}`,
			`"type":"Bogus"`},
		{"nudge",
			`{"handle":"h1","paths":["a"],"direction":"up","offset":-7}`,
			`"offset":-7`},
	}
	for _, c := range cases {
		args, err := BuildArgsFromJSON(m, c.fn, json.RawMessage(c.in))
		if err != nil {
			t.Fatalf("%s：本地不该拦：%v", c.fn, err)
		}
		frame, err := MarshalRequest(1, c.fn, args)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(frame), c.inReq) {
			t.Errorf("%s：请求帧里缺 %s\n帧：%s", c.fn, c.inReq, frame)
		}
	}
}

// TestBuildArgsForForm 钉住逻辑键寻址：form 落在 handle 字段上，两种寻址只能给一个。
//
// 这是 "让调用方说'改 aapp320 的表单'" 而不是"搬一串易失的 h1" 的入口（见 Rpc.FindByKey）。
func TestBuildArgsForForm(t *testing.T) {
	cases := []struct {
		name    string
		fn      string
		in      string
		form    string
		want    string
		wantErr string
	}{
		{
			"form 顶替 handle（顺序仍按 manifest）", "save",
			`{"out":"D:\\x.tzs"}`, "aapp320",
			`{"handle":"aapp320","out":"D:\\x.tzs"}`, "",
		},
		{
			"ProgramKey 形式照发", "nudge",
			`{"paths":["a"],"direction":"up"}`, "aapp320|Form",
			`{"handle":"aapp320|Form","paths":["a"],"direction":"up"}`, "",
		},
		{
			"form 与 args.handle 同时给 = 错", "save",
			`{"handle":"h1","out":"x"}`, "aapp320",
			"", "只能给一个",
		},
		{
			"不需要句柄的动词给 form = 错", "list_open",
			`{}`, "aapp320",
			"", "不接受 form",
		},
		{
			"没给 form 时行为不变（handle 必填照报）", "save",
			`{"out":"x"}`, "",
			"", "缺必填参数 handle",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := BuildArgsForForm(mustManifest(t), c.fn, json.RawMessage(c.in), c.form)
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("不该报错：%v", err)
				}
				if string(got) != c.want {
					t.Fatalf("定型结果不对\n  得 %s\n想 %s", got, c.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("该报错（含 %q），却得到 %s", c.wantErr, got)
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("文案里缺 %q：%v", c.wantErr, err)
			}
		})
	}
}

// TestUnknownParamTeachesForm：把 form 写进 args 是最容易犯的一种写法错误
// （`form` 确实比 `handle` 更像人话），所以那条报错必须**指出正确的写法**，
// 而不只是说"不在 manifest 里" —— 后者会让人以为此路不通。
func TestUnknownParamTeachesForm(t *testing.T) {
	var ue *UsageError
	_, err := BuildArgsFromJSON(mustManifest(t), "nudge",
		json.RawMessage(`{"form":"aapp320","paths":["a"],"direction":"up"}`))
	if !errors.As(err, &ue) {
		t.Fatalf("该是用法错，得 %T：%v", err, err)
	}
	if !strings.Contains(err.Error(), "参数 form 不在 manifest 里") {
		t.Errorf("该先点出未知参数：%v", err)
	}
	if !strings.Contains(strings.Join(ue.Detail, " "), "--form") {
		t.Errorf("明细里该教正确的写法（--form <程序名>）：%v", ue.Detail)
	}
}

// TestJSONStringEscapesLikeTheWire：帮助里演示的写法与真正上线的写法必须同一个转义器。
func TestJSONStringEscapesLikeTheWire(t *testing.T) {
	cases := []string{`plain`, `<name-path>`, `a"b`, `a\b`, `中文`, "</section>"}
	for _, s := range cases {
		got := JSONString(s)
		if !strings.HasPrefix(got, `"`) || !strings.HasSuffix(got, `"`) {
			t.Fatalf("%q 没被编成 JSON 字符串：%s", s, got)
		}
		var back string
		if err := json.Unmarshal([]byte(got), &back); err != nil {
			t.Fatalf("%q 编出来的不是合法 JSON（%s）：%v", s, got, err)
		}
		if back != s {
			t.Errorf("往返丢了内容：%q → %s → %q", s, got, back)
		}
		// 关掉 HTML 转义（与 MarshalRequest 同一条）：`</section>` 在日志里要能照着读。
		if strings.Contains(got, `\u003c`) {
			t.Errorf("%q 被 HTML 转义了：%s", s, got)
		}
	}
}

// 这不是洁癖：记事本和 PowerShell 的 `Set-Content -Encoding utf8` 都会写 BOM，
// 而引擎那侧的帧读取同样剥它（见 wire.go 的入方向容错）。
func TestBuildArgsFromJSONToleratesBOM(t *testing.T) {
	in := append([]byte{0xEF, 0xBB, 0xBF}, []byte("  {\"handle\":\"h1\",\"out\":\"x\"}\n")...)
	got, err := BuildArgsFromJSON(mustManifest(t), "save", in)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"handle":"h1","out":"x"}` {
		t.Errorf("带 BOM 的输入没解析对：%s", got)
	}
}
