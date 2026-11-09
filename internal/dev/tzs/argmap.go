package tzs

// argmap.go —— 一份 JSON args → 引擎可用的 args。
//
// 命令面只接受**一种**参数写法：`--args '<JSON 对象>'` / `--args-file <文件>`。
// 没有 `--<参数> <值>` 那种写法，这是刻意的：
//
//   - 动词参数的名字与类型是**运行时**从引擎 manifest 来的（151 个参数、11 种类型）。
//     照它为每个动词手写一遍 flag，就是 50 × N 处会跟引擎漂移的代码；动态解析则把
//     `--paths a b c`、裸词、重复标量、负号、引号这些**语法问题**变成调用方要学的东西。
//   - 一份 JSON 把这一类问题一次消掉：数组就是数组、布尔就是布尔、字符串不用转义，
//     没有"按逗号切"与"能不能重复"这种只存在于命令行的规则。
//
// 本地只做**语法**层校验（未知函数 / 未知参数 / 缺必填 / int / bool / enum 静态取值 /
// 数组元素类型），语义层（值域）留给引擎：
//
//   - `kind` 的合法集（field/hfield/pfield/rfield/mlfield/tree/act）**不在 manifest 里**；
//   - `attr`（from: "spec:<kind>" / "layout"）的合法集要从**活着的模型**里取；
//   - `add_action` 的 `type` 合法集是**每张表单自己的**（该表单的 s_detail<n> 记录）。
//
// 这三样在本地拦下来，就会让一个引擎本会接受的调用**永远到不了引擎** —— 而越界拦截是
// 无声发生的：写错了不会有任何测试失败，只会让某些调用在真机上莫名其妙地过不去。
// 引擎拒绝它们的退出码是 kind=validation → 2，和本地拦下来一模一样，所以放过不会让
// 报错变晚或变差（它还会把 detail.legal 里的合法集带回来）。
//
// 相对引擎侧的 tzs-cli，这里有一处**有意差异**：位置参数与未知开关一律**当场报错**，
// 而 tzs-cli 会「忽略非选项参数」并继续。无声忽略正是引擎 Manifest.Check 双向卡 arity
// 要治的病 —— `--attrr` 打错了和一次合法调用在结果上无法区分，直到有人去读文件。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// BuildArgsFromJSON 按 manifest 把**一个 JSON 对象**定型成 args。
//
// 这是唯一的参数入口（见文件头）。除了语法层校验，它还把结果按键序重排成
// **manifest 的声明顺序** —— 同一份输入永远产出同一串字节，于是请求帧、日志与
// 测试断言都能逐字节比对，JSON 里键的书写顺序也不影响帧。
func BuildArgsFromJSON(m *Manifest, fn string, raw json.RawMessage) (json.RawMessage, error) {
	return BuildArgsForForm(m, fn, raw, "")
}

// BuildArgsForForm 与 BuildArgsFromJSON 相同，但允许按**逻辑键**寻址（form 非空时）。
//
// form 是程序名（`aapp320`）或 ProgramKey（`aapp320|Form`）。它**不是** manifest 参数，
// 而是"哪张表单"的寻址方式，所以落在引擎的 `handle` 字段上 —— 引擎两种写法都认
// （句柄 `h1` / 程序名 / ProgramKey，见 Rpc.FindByKey）。线上契约因此一个字都没改。
//
// 为什么值得这么做：句柄是 `h1`、`h2`… 且**永不复用**（每次 open 都换号），让调用方
// 把它在每条命令之间搬运，等于让"我要改 aapp320 的表单"这句话被迫用一串易失的随机号
// 说出来。程序名与 ProgramKey 是 open 的返回里本来就有的东西（list_open 也报），
// 用它们寻址不需要任何新状态。
//
// 两种寻址**只能给一个**：同时给了 form 与 args.handle 是错误，不做优先级猜测 ——
// "我以为生效的是那个"是这类工具最难查的一类事故。反过来，给一个不需要句柄的动词
// （list_open/base_data 这类）传 form 也是错误：那说明调用方以为它针对某张表单。
func BuildArgsForForm(m *Manifest, fn string, raw json.RawMessage, form string) (json.RawMessage, error) {
	spec := m.ByName(fn)
	if spec == nil {
		if fn == FnStop {
			// stop 是传输级命令，引擎在查表之前就把它截走，所以它不在 manifest 里。
			return nil, &UsageError{Msg: FnStop + " 是传输级命令，不走参数定型"}
		}
		return nil, &UsageError{
			Msg:    "未知函数 " + fn,
			Detail: []string{"可用函数：" + ClipNames(m.Names())},
		}
	}

	// 行首 BOM 与两端空白剥掉：记事本/PowerShell 写 UTF-8 文件时常常带 BOM
	// （引擎那侧的帧读取同样剥它，见 wire.go 的入方向容错）。
	body := bytes.TrimSpace(raw)
	body = bytes.TrimSpace(bytes.TrimPrefix(body, utf8BOM))
	if len(body) == 0 || isJSONNull(body) {
		// 空 / null 等价于「一个参数都不给」，于是缺必填会照报。
		body = []byte("{}")
	}
	if body[0] != '{' {
		return nil, &UsageError{
			Msg:    "args 必须是 JSON 对象（不是数组/标量/字符串）",
			Detail: []string{"函数 " + fn, "收到：" + clip(body)},
		}
	}
	in := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &in); err != nil {
		return nil, &UsageError{
			Msg: "args 不是合法的 JSON 对象",
			Detail: []string{"函数 " + fn, err.Error(),
				"提示：Windows 的管道可能按控制台代码页重编码，中文/非 ASCII 内容请用 --args-file 写 UTF-8 文件"},
		}
	}

	// 未知参数：报**全部**且排序 —— map 迭代顺序是随机的，不排序就会同一条输入
	// 每次报出不同的参数名（测试与用户都会以为它是随机的）。
	var unknown []string
	for name := range in {
		if spec.Param(name) == nil {
			unknown = append(unknown, name)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, &UsageError{
			Msg:    "参数 " + strings.Join(unknown, ", ") + " 不在 manifest 里（函数 " + fn + "）",
			Detail: unknownParamDetail(spec, unknown),
		}
	}

	out := map[string]json.RawMessage{}
	var keys []string
	tookForm := false
	for _, p := range spec.Args {
		// 逻辑键寻址：form 落在 handle 字段上（引擎认句柄 / 程序名 / ProgramKey 三种写法）。
		if p.Name == "handle" && form != "" {
			if _, dup := in["handle"]; dup {
				return nil, &UsageError{
					Msg:    "handle 与 form 只能给一个（函数 " + fn + "）",
					Detail: []string{"要么 --form <程序名>，要么在 JSON 里给 handle；两个都给就没人知道哪个生效"},
				}
			}
			out[p.Name] = json.RawMessage(JSONString(form))
			keys = append(keys, p.Name)
			tookForm = true
			continue
		}
		v, ok := in[p.Name]
		// JSON 的 null 视同缺席 —— 与引擎 Manifest.Check 的 absent 判定同口径。
		if !ok || isJSONNull(v) {
			if p.Required {
				return nil, &UsageError{
					Msg:    "缺必填参数 " + p.Name + "（函数 " + fn + "）",
					Detail: []string{"必填：" + ClipNames(spec.RequiredParams())},
				}
			}
			continue
		}
		got, err := paramJSONValue(fn, p, v)
		if err != nil {
			return nil, err
		}
		out[p.Name] = json.RawMessage(got)
		keys = append(keys, p.Name)
	}
	// 给一个不需要句柄的动词传 form：调用方以为它针对某张表单，而它不针对。
	// 静默丢掉是最坏的选择（调用方会以为"指定了却没用"是自己的错觉）。
	if form != "" && !tookForm {
		return nil, &UsageError{
			Msg: "函数 " + fn + " 不接受 form（它不需要句柄）",
			Detail: []string{"form 是「哪张已打开的表单」的寻址方式，只有需要句柄的动词才有意义",
				"该函数的参数：" + ClipNames(paramNames(spec))},
		}
	}
	return emitArgs(keys, out), nil
}

// unknownParamDetail 拼"未知参数"的明细，并在调用方**把寻址方式写进 args** 时给出正确写法。
//
// 这条不是锦上添花：`form`/`program`/`key` 三个词是调用方最可能顺手写进 JSON 的
// （它们确实比 `handle` 更像人话），而只说"参数 form 不在 manifest 里"会让人
// 以为此路不通 —— 其实只是写法不对。
func unknownParamDetail(spec *SpecFn, unknown []string) []string {
	detail := []string{"该函数的参数：" + ClipNames(paramNames(spec))}
	for _, u := range unknown {
		if u == "form" || u == "program" || u == "key" {
			detail = append(detail,
				"按程序名寻址不是 JSON 参数，而是传输开关：tt dev tzs "+spec.Name+" --form <程序名> [--args '<JSON>']")
			break
		}
	}
	return detail
}

// emitArgs 按 keys 顺序把 vals 拼成一个 args 对象。
//
// 顺序 = manifest 里参数的**声明顺序**（不是 JSON 里的书写顺序）：见 BuildArgsFromJSON。
func emitArgs(keys []string, vals map[string]json.RawMessage) json.RawMessage {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(JSONString(k))
		buf.WriteByte(':')
		buf.Write(vals[k])
	}
	buf.WriteByte('}')
	return json.RawMessage(buf.Bytes())
}

// paramJSONValue 把一个 JSON 取值定型成 args 里那一段。
//
// 判据逐条对齐引擎的 Manifest.Check：
//
//	int  只认 JSON 数字（`"2"` 与 `1.5` 都拒 —— 引擎对这两种同样拒）
//	bool 只认字面量 true/false
//	enum 认字符串且在静态合法集内（Values 为空则不校验，与引擎一致）
//	数组 必须是字符串数组（元素是对象/数字都拒）
//	attrs 必须是对象且值是字符串（键排一次序，重发出去的字节稳定）
//	其余（string / path / handle / kind / attr / 未知类型）按字符串对待：
//	     对象与数组拒（引擎也拒），字符串重新转义成规范形式，其余标量原样透传
//
// 最后那一条的"原样透传"是有意的：引擎的 Check 对 string 参数收到数字/布尔是**放行**的，
// 本地凭空加一条引擎没有的规则，就会让一个引擎本会处理的调用永远到不了引擎。
func paramJSONValue(fn string, p *Param, v json.RawMessage) (string, error) {
	t := bytes.TrimSpace(v)
	switch p.Type {
	case TypePathList, TypeStrList:
		var arr []json.RawMessage
		if len(t) == 0 || t[0] != '[' || json.Unmarshal(t, &arr) != nil {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 需要字符串数组，收到 " + jsonKindName(t),
				Detail: []string{"函数 " + fn, "写法：\"" + p.Name + "\": [\"a\", \"b\"]"},
			}
		}
		var b bytes.Buffer
		b.WriteByte('[')
		for i, e := range arr {
			et := bytes.TrimSpace(e)
			s, ok := jsonString(et)
			if !ok {
				return "", &UsageError{
					Msg:    "参数 " + p.Name + " 的数组元素必须是字符串",
					Detail: []string{"函数 " + fn, "第 " + strconv.Itoa(i+1) + " 个收到 " + jsonKindName(et)},
				}
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(JSONString(s))
		}
		b.WriteByte(']')
		return b.String(), nil

	case TypeAttrs:
		// 一个 JSON 对象，值必须是字符串。形状逐条对齐引擎的 Manifest.Check；**不**判
		// 「哪些属性名合法、哪些值合法」—— 前者要看活模型、后者要看工作区的 mod-fd.spec，
		// 本地两样都没有（与 kind/attr 同一条分工，见 §11.24(c)）。
		var obj map[string]json.RawMessage
		if len(t) == 0 || t[0] != '{' || json.Unmarshal(t, &obj) != nil {
			return "", &UsageError{
				Msg: "参数 " + p.Name + " 需要 JSON 对象（属性名 → 字符串值），收到 " + jsonKindName(t),
				Detail: []string{"函数 " + fn,
					"写法：\"" + p.Name + "\": {\"can_query\": \"N\", \"req\": \"Y\"}"},
			}
		}
		if len(obj) == 0 {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 是空对象",
				Detail: []string{"函数 " + fn, "只改一个属性用单数形式的动词"},
			}
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys) // 稳定输出：同一次调用重发出去的字节一致，diff 与测试才有意义
		var b bytes.Buffer
		b.WriteByte('{')
		for i, k := range keys {
			raw := bytes.TrimSpace(obj[k])
			s, ok := jsonString(raw)
			if !ok {
				return "", &UsageError{
					Msg: "参数 " + p.Name + " 里 " + k + " 的值需要字符串，收到 " + jsonKindName(raw),
					Detail: []string{"函数 " + fn,
						"属性值在 .4fd/.tsd 里都是文本：写 \"20\" 而不是 20"},
				}
			}
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(JSONString(k))
			b.WriteByte(':')
			b.WriteString(JSONString(s))
		}
		b.WriteByte('}')
		return b.String(), nil

	case TypeInt:
		var n json.Number
		if len(t) == 0 || t[0] == '"' || json.Unmarshal(t, &n) != nil {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 需要整数，收到 " + jsonKindName(t),
				Detail: []string{"函数 " + fn, "JSON 里写不带引号的数字：" + p.Name + ": 2"},
			}
		}
		i, err := strconv.ParseInt(string(n), 10, 64)
		if err != nil {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 需要整数，收到 " + strconv.Quote(string(n)),
				Detail: []string{"函数 " + fn},
			}
		}
		return strconv.FormatInt(i, 10), nil

	case TypeBool:
		if string(t) == "true" || string(t) == "false" {
			return string(t), nil
		}
		return "", &UsageError{
			Msg:    "参数 " + p.Name + " 需要 true/false，收到 " + jsonKindName(t),
			Detail: []string{"函数 " + fn},
		}

	case TypeEnum:
		s, ok := jsonString(t)
		if !ok {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 需要一个字符串，收到 " + jsonKindName(t),
				Detail: []string{"函数 " + fn},
			}
		}
		// Values 为空时跳过校验（防御：一个没带 values 的 enum 在引擎那边会接受任意字符串，
		// 我们跟着放开，免得把引擎能处理的值在本地判死）。
		if len(p.Values) > 0 && !contains(p.Values, s) {
			return "", &UsageError{
				Msg:    "参数 " + p.Name + " 的取值不在允许集合内：" + strconv.Quote(s),
				Detail: []string{"函数 " + fn, "可用：" + strings.Join(p.Values, "|")},
			}
		}
		return JSONString(s), nil
	}

	// string / path / handle / kind / attr / 未知类型：一律按字符串对待。
	if len(t) > 0 && (t[0] == '{' || t[0] == '[') {
		return "", &UsageError{
			Msg:    "参数 " + p.Name + " 需要一个字符串，收到 " + jsonKindName(t),
			Detail: []string{"函数 " + fn},
		}
	}
	if s, ok := jsonString(t); ok {
		return JSONString(s), nil
	}
	// 数字 / 布尔：照引擎放行，原样透传（见本函数的注释）。
	return string(t), nil
}

// jsonString 判断 t 是不是一个 JSON 字符串，是则解出它的值。
func jsonString(t []byte) (string, bool) {
	if len(t) == 0 || t[0] != '"' {
		return "", false
	}
	var s string
	if err := json.Unmarshal(t, &s); err != nil {
		return "", false
	}
	return s, true
}

// isJSONNull 判断 t 是不是 JSON 的 null（「缺席」的另一种写法）。
func isJSONNull(t []byte) bool { return bytes.Equal(bytes.TrimSpace(t), []byte("null")) }

// jsonKindName 给一个 JSON 取值起个类型名，只用于报错文案。
//
// 存在的理由是错误必须能自纠：`收到 数字 1.5` 比 `收到 invalid` 有用得多。
func jsonKindName(t []byte) string {
	t = bytes.TrimSpace(t)
	if len(t) == 0 {
		return "空"
	}
	switch t[0] {
	case '{':
		return "JSON 对象"
	case '[':
		return "JSON 数组"
	case '"':
		return "字符串"
	case 't', 'f':
		return "布尔"
	case 'n':
		return "null"
	}
	return "数字 " + clip(t)
}

func contains(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// JSONString 把一个字符串编成 JSON 字符串（关掉 HTML 转义，见 MarshalRequest 的注释）。
//
// 导出是给命令层渲染 JSON 片段用的（`tt dev tzs <动词> --help` 里那条示例）：转义只该有
// 一份实现 —— 帮助里演示的写法与真正发上线的写法用同一个转义器，不一致才是事故。
func JSONString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		// encoding/json 对 string 不会失败；真失败也只能给出一个合法的 JSON 串。
		return `""`
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func paramNames(f *SpecFn) []string {
	out := make([]string, 0, len(f.Args))
	for _, p := range f.Args {
		out = append(out, p.Name)
	}
	return out
}

// ClipNames 把清单裁短（报错文案里不该出现一屏 50 个名字）。
//
// 导出是给命令层的：`tt dev tzs <未知动词>` 要列出可用动词清单，而这份裁短规则
// 只能有一处实现（抄一份就会在两处对"多长算长"给出不同答案）。
func ClipNames(names []string) string {
	if len(names) == 0 {
		return "(无)"
	}
	const max = 12
	if len(names) <= max {
		return strings.Join(names, " ")
	}
	return fmt.Sprintf("%s … 共 %d 个", strings.Join(names[:max], " "), len(names))
}
