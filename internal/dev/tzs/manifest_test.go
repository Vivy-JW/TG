package tzs

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// fixtureManifest 是**手写的小夹具**，不是真 manifest 的拷贝。
//
// 为什么不拷真的：真 manifest 有 36 KB、50 个函数，它一改（引擎加个参数）测试就得跟着改，
// 而测试想钉住的是**解析规则**（req/values/from/分组/慢函数），不是引擎此刻有几个函数。
// 拷贝还会给人一个假印象：「这份测试证明了我们的工具和引擎对得上」—— 它证明不了，
// 那是 TTZS_E2E=1 的活。
//
// 每一项都是按 ENGINE 的真实形状写的（字段名、类型名、from 的写法），所以它同时充当
// 「我们读的字段名没写错」这份文档。
const fixtureManifest = `[
  {"fn":"list_open","group":"会话","desc":"列出打开的句柄","writes":false,"slow":false,
   "returns":"list<el>","needsHandle":false,"args":[],"errors":["E_NOT_FOUND","E_BAD_PARAM"]},
  {"fn":"save","group":"会话","desc":"把句柄的模型写回新包","writes":false,"slow":false,
   "returns":"void","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},{"n":"out","t":"path","req":true,"desc":"输出 .tzs 路径"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"nudge","group":"结构","desc":"按方向平移","writes":true,"slow":false,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"paths","t":"path[]","req":true},
           {"n":"direction","t":"enum","req":true,"values":["up","down","left","right"]},
           {"n":"offset","t":"int","req":false,"desc":"格数，默认 1"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"set_spec_attr","group":"属性","desc":"改字段规格属性","writes":true,"slow":false,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":true,"desc":"name-path"},
           {"n":"kind","t":"kind","req":true},
           {"n":"attr","t":"attr","req":true,"from":"spec:<kind>"},
           {"n":"value","t":"string","req":true,"desc":"新值"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"set_spec_attrs","group":"属性","desc":"一次改一个节点的多个规格属性","writes":true,"slow":false,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":true,"desc":"name-path"},
           {"n":"kind","t":"kind","req":true},
           {"n":"attrs","t":"attrs","req":true,"desc":"{\"属性名\":\"值\", …}"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"add_action","group":"语义/Action","desc":"新增 Action","writes":true,"slow":false,
   "returns":"el","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":true,"desc":"父容器的 name-path"},
           {"n":"type","t":"string","req":false,"desc":"型态，逗号分隔；合法集是该表单自己的"},
           {"n":"name","t":"string","req":false,"desc":"action id"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"set_items","group":"多语言/选项/串查","desc":"ComboBox 的选项值","writes":true,"slow":false,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":true},
           {"n":"items","t":"string[]","req":true,"desc":"name|text|description 字符串数组"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"set_excluded","group":"校验/工具","desc":"排除/取消排除控件","writes":true,"slow":false,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":true,"desc":"name-path"},
           {"n":"excluded","t":"bool","req":true,"desc":"true = 排除"}],
   "errors":["E_BAD_PARAM"]},
  {"fn":"validate","group":"校验/工具","desc":"跑设计器自己的校验器","writes":false,"slow":true,
   "returns":"delta","needsHandle":true,
   "args":[{"n":"handle","t":"handle","req":true},
           {"n":"path","t":"path","req":false,"desc":"只校验这条子树"}],
   "errors":["E_BAD_PARAM"]}
]`

func mustManifest(t *testing.T) *Manifest {
	t.Helper()
	m, err := ParseManifest([]byte(fixtureManifest))
	if err != nil {
		t.Fatalf("夹具 manifest 该能解析: %v", err)
	}
	return m
}

// TestParseManifestShapes 钉住每个字段的读法（字段名来自引擎的 Manifest.ToJson）。
func TestParseManifestShapes(t *testing.T) {
	m := mustManifest(t)

	if len(m.Fns) != 9 {
		t.Fatalf("该有 9 个函数，得 %d", len(m.Fns))
	}
	// 顺序即 manifest 顺序（--help 的分节顺序就是它，不是字母序）。
	if m.Fns[0].Name != "list_open" || m.Fns[len(m.Fns)-1].Name != "validate" {
		t.Errorf("函数顺序不对：%s … %s", m.Fns[0].Name, m.Fns[len(m.Fns)-1].Name)
	}

	lo := m.ByName("list_open")
	if lo == nil {
		t.Fatal("list_open 该在表里")
	}
	if lo.NeedsHandle {
		t.Error("list_open 的 needsHandle 该是 false（就绪握手靠它）")
	}
	if lo.Writes || lo.Slow {
		t.Error("list_open 既不改模型也不慢")
	}
	if lo.Returns != "list<el>" || len(lo.Args) != 0 {
		t.Errorf("list_open 的 returns/args 不对：%q %d", lo.Returns, len(lo.Args))
	}

	v := m.ByName("validate")
	if !v.Slow {
		t.Error("validate 该是 slow（670 元素实测 10.4 s）")
	}
	if !v.NeedsHandle {
		t.Error("validate 需要句柄")
	}

	nudge := m.ByName("nudge")
	if !nudge.Writes {
		t.Error("nudge 改模型")
	}
	p := nudge.Param("paths")
	if p == nil || p.Type != TypePathList || !p.Required {
		t.Errorf("nudge.paths 该是必填的 path[]：%+v", p)
	}
	d := nudge.Param("direction")
	if d == nil || d.Type != TypeEnum || len(d.Values) != 4 || d.Values[0] != "up" {
		t.Errorf("nudge.direction 的 enum 值不对：%+v", d)
	}
	off := nudge.Param("offset")
	if off == nil || off.Type != TypeInt || off.Required {
		t.Errorf("nudge.offset 该是选填的 int：%+v", off)
	}
	if nudge.Param("nope") != nil {
		t.Error("不存在的参数该返回 nil")
	}

	// from:* 只出现在 attr 上（真 manifest 里只有 set_spec_attr 与 set_layout_attr 两个）。
	ssa := m.ByName("set_spec_attr")
	attr := ssa.Param("attr")
	if attr == nil || attr.Type != TypeAttr || attr.From != "spec:<kind>" {
		t.Errorf("set_spec_attr.attr 该是 from=spec:<kind> 的 attr：%+v", attr)
	}
	if attr.Values != nil {
		t.Error("from:* 的参数**没有**静态 values —— 它的合法集要现场从模型里取")
	}

	// add_action 的 type 在真 manifest 里是一个普通 string（没有 from）：
	// 「型态」的合法集是**每张表单自己的**（该表单的 s_detail<n> 记录），静态列不出来。
	aa := m.ByName("add_action")
	if ty := aa.Param("type"); ty == nil || ty.Type != TypeString || ty.From != "" {
		t.Errorf("add_action.type 该是普通 string（无 from）：%+v", ty)
	}
	if nh := aa.Param("name"); nh == nil || !strings.Contains(nh.Desc, "action id") {
		t.Errorf("desc 该被读到：%+v", nh)
	}

	if got := len(m.ByName("set_excluded").RequiredParams()); got != 3 {
		t.Errorf("set_excluded 有 3 个必填，得 %d", got)
	}
	if strings.Join(m.Names()[:2], ",") != "list_open,save" {
		t.Errorf("Names 该按 manifest 顺序：%v", m.Names())
	}
}

// TestManifestGroups：分组顺序 = 首次出现的顺序（help 的分节顺序）。
func TestManifestGroups(t *testing.T) {
	m := mustManifest(t)
	want := []string{"会话", "结构", "属性", "语义/Action", "多语言/选项/串查", "校验/工具"}
	got := m.Groups()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("分组顺序不对：%v，想要 %v", got, want)
	}
}

// TestManifestIndex 钉住动词索引的**边界**：列分组与名字，**不列参数**。
//
// 后半条是这次改动的要点：索引不该把函数表整个摊给调用方（参数是每个动词自己的契约，
// `tt dev tzs <动词> --help` 按需给一个）。所以这里既断言"该有的都在"，也断言"参数不在"。
func TestManifestIndex(t *testing.T) {
	var b strings.Builder
	mustManifest(t).Index(&b)
	out := b.String()

	for _, want := range []string{
		"[会话]", "[结构]", "[校验/工具]",
		"list_open", "nudge", "validate",
		"[slow]", "[写]",
		FnStop, // 提示 stop 不在表里
	} {
		if !strings.Contains(out, want) {
			t.Errorf("索引里缺 %q：\n%s", want, out)
		}
	}
	// 参数契约一概不出现：类型、必填、枚举值、from:* —— 那是 <动词> --help 的事。
	for _, forbidden := range []string{"path[]", "必填", "--paths", "enum(", "attr:spec:"} {
		if strings.Contains(out, forbidden) {
			t.Errorf("索引里不该出现参数契约 %q：\n%s", forbidden, out)
		}
	}
}

// TestParseManifestRejects：五种坏输入各报一条用法错（退出码 2）。
//
// manifest 拉不到/解不动归在**用法错**那一栏：到这一步工具连函数表都没有，
// 无法判断调用方写的东西是否合法，和「参数名打错」是同一类处境。
func TestParseManifestRejects(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string // 错误文案里必须出现的子串
	}{
		{"不是 JSON", "tzs-cli -- T100 设计器", "不是函数表"},
		{"是对象不是数组", `{"fn":"x"}`, "不是函数表"},
		{"空数组", `[]`, "空的"},
		{"缺 fn", `[{"group":"x"}]`, "没有 fn"},
		{"重名", `[{"fn":"a"},{"fn":"a"}]`, "重名"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, err := ParseManifest([]byte(c.in))
			if err == nil {
				t.Fatalf("该被拒，却解出了 %d 个函数", len(m.Fns))
			}
			var ue *UsageError
			if !errors.As(err, &ue) {
				t.Fatalf("该是 *UsageError，得 %T", err)
			}
			if ue.ExitCode() != ExitUsage {
				t.Errorf("该退 2，得 %d", ue.ExitCode())
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("文案里缺 %q：%v", c.want, err)
			}
		})
	}
}

// TestKnownTypeMatchesEngine：参数类型名的集合就是引擎 Manifest.TypeName 的十二个分支。
//
// 契约给的清单里**漏了 handle**，而它在真 manifest 里出现 44 次 ——
// 这一条不是多余的：漏了它，每个需要句柄的函数都会因为「未知类型」被本地判死。
func TestKnownTypeMatchesEngine(t *testing.T) {
	// 这十二个就是 engine/src/Designer/Manifest.cs 的 TypeName 全部返回值。
	// ⚠️ 引擎那边**加一个 PType 成员就必须在两个 switch 里各加一行**：TypeName 的 default 是
	// "string"，漏了只会静默丢掉类型信息（2026-09-25 的 KindOrLayout 就这样过了一版构建）。
	// 这条测试拦不住那种漏 —— 它拦的是相反的方向（这边认了引擎产不出的名字）。
	all := []string{"int", "bool", "handle", "kind", "kind-or-layout", "attr", "enum", "path", "path[]", "string[]", "attrs", "string"}
	for _, ty := range all {
		if !KnownType(ty) {
			t.Errorf("引擎会产出 t=%q，KnownType 却说不知道", ty)
		}
	}
	for _, ty := range []string{"", "object", "float", "number[]"} {
		if KnownType(ty) {
			t.Errorf("%q 不是引擎的类型名", ty)
		}
	}
	// 未知类型不报错、按字符串发（见 KnownType 的注释）：引擎加类型时，
	// 我们宁可让引擎回一句明确的 E_BAD_PARAM，也不要让合法调用永远到不了引擎。
	m := mustManifest(t)
	f := m.ByName("nudge")
	f.Args[3].Type = "futuretype"
	if _, err := BuildArgsFromJSON(m, "nudge", json.RawMessage(
		`{"handle":"h1","paths":["a"],"direction":"up","offset":3}`)); err != nil {
		t.Errorf("未知类型该按字符串放过，得 %v", err)
	}
}
