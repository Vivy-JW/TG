package tzs

// tzs_verb_e2e_test.go —— 拿**真 manifest** 验动词面。
//
// 为什么放在 cli 包而不是 internal/dev/tzs 的 E2E 里：要证的几件事都是**命令层**的性质，
// 而且它们只需要 `tzs-server --manifest`（不 Boot、不连守护进程、不需要工作区），
// 所以这条 E2E 很便宜。
//
// 开启方式：
//	$env:TTZS_E2E="1"
//	$env:TTZS_EXE="D:\...\engine\out\tzs-server.exe"   # 可省：默认取仓库里的 engine/out
//	go test ./internal/cli/dev/ -run TestE2EVerbSurface -v

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tt/internal/dev/tzs"
	"tt/internal/testenv"
)

func requireRealManifest(t *testing.T) *tzs.Manifest {
	t.Helper()
	if os.Getenv("TTZS_E2E") == "" {
		t.Skip("需要真引擎：设 TTZS_E2E=1（可选 TTZS_EXE），见本文件顶部注释")
	}
	exe := testenv.EngineExe()
	if exe == "" {
		exe = filepath.Join("..", "..", "..", "engine", "out", "tzs-server.exe")
	}
	if _, err := os.Stat(exe); err != nil {
		t.Skipf("找不到引擎 exe（TTZS_EXE=%s）：%v", exe, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	m, err := tzs.FetchManifest(ctx, exe)
	if err != nil {
		t.Fatalf("拉不到真 manifest：%v", err)
	}
	return m
}

// TestE2EVerbSurface 拿真 manifest 证命令层的几条性质。
//
// 它们的共同后果都是"某个动词/参数悄悄不可达"，而现象看起来像引擎的问题：
//
//	① 内建动词与引擎函数撞名 → 内建赢，那个函数再也敲不出来；
//	② 有函数名却敲不出来（连字符别名也要能敲）；
//	③ 有参数却定型不了（类型没被认识 → 合法调用到不了引擎）；
//	④ 参数还能用 flag 形式给 —— 那就说明"只用 JSON"没有被真正落实。
func TestE2EVerbSurface(t *testing.T) {
	m := requireRealManifest(t)
	t.Logf("真 manifest：%d 个动词，%d 个参数", len(m.Fns), countParams(m))

	// ① 内建动词不能与引擎函数同名（同名时 warnBuiltinCollisions 会警告，这里直接判失败：
	//    真发生了就该改设计，而不是靠一句警告糊过去）。
	for _, b := range builtinVerbs {
		if m.ByName(b) != nil {
			t.Errorf("引擎有函数叫 %q，与内建动词撞名：内建优先，该函数无法用动词调用", b)
		}
	}

	switches := transportSwitchNames()
	for _, f := range m.Fns {
		// ② 本名与连字符形式都要找得到。
		if lookupVerb(m, f.Name) == nil {
			t.Errorf("动词 %q 用本名找不到", f.Name)
		}
		if kebab := strings.ReplaceAll(f.Name, "_", "-"); kebab != f.Name {
			if lookupVerb(m, kebab) == nil {
				t.Errorf("动词 %q 的连字符形式 %q 找不到", f.Name, kebab)
			}
		}

		// ③ 每个声明的参数都要能被 JSON 入口定型（给齐必填就是一次合法调用）。
		body := dummyArgsJSON(t, f)
		args, err := tzs.BuildArgsFromJSON(m, f.Name, body)
		if err != nil {
			t.Errorf("%s：给齐参数后 JSON 定型仍失败（%s）：%v", f.Name, body, err)
		} else if !json.Valid(args) {
			t.Errorf("%s：定型结果不是合法 JSON：%s", f.Name, args)
		}

		// ④ 同一个参数用 flag 形式给，必须当场被拒（参数只用 JSON）。
		for _, p := range f.Args {
			if switches[p.Name] {
				continue // 名字恰好是传输开关：那不是参数形式的错
			}
			if _, err := takeVerbFlags([]string{f.Name, "--" + p.Name, "v"}); err == nil {
				t.Errorf("%s --%s：参数只用 JSON 给，flag 形式该被拒", f.Name, p.Name)
			}
		}

		// ⑤ `--help` 里那条示例：`--args` 那一段必须真的是合法 JSON ——
		//    用坏例子教人是最坏的一种错，而所有示例都是生成的，一次全检很便宜。
		ex := verbExample(f)
		if f.NeedsHandle && !strings.Contains(ex, "--form") {
			t.Errorf("%s 的示例该用 --form 寻址：%s", f.Name, ex)
		}
		if !f.NeedsHandle && strings.Contains(ex, "--form") {
			t.Errorf("%s 不需要句柄，示例里不该有 --form：%s", f.Name, ex)
		}
		if i := strings.Index(ex, "--args '"); i >= 0 {
			rest := ex[i+len("--args '"):]
			if j := strings.Index(rest, "'"); j < 0 {
				t.Errorf("%s 的示例单引号没配对：%s", f.Name, ex)
			} else if !json.Valid([]byte(rest[:j])) {
				t.Errorf("%s 的示例里 args 不是合法 JSON：%s", f.Name, rest[:j])
			}
		}

		// ⑥ 示例里的占位符必须与参数的**角色**一致。包路径教成 `<name-path>` 是这条测试
		//    存在的理由：`open.path` 与 `field_add.out` 从前都被教错，而它们都是包路径
		//    （2026-09-24 的设计评审发现，修法是引擎声明里的 `role`）。
		if args := exampleArgsOf(t, ex); args != nil {
			for _, p := range f.Args {
				if p.Role != tzs.RolePackagePath {
					continue
				}
				if v, ok := args[p.Name].(string); ok && v == "<name-path>" {
					t.Errorf("%s：参数 %s 的角色是包路径，示例却教成 <name-path>：%s", f.Name, p.Name, ex)
				}
			}
		}

		// 动词的参数表要能渲染（`<动词> --help` 走的就是 printFn）。
		if f.Name == "list_open" {
			printFn(io.Discard, f)
		}
	}
}

// transportSwitchNames 把传输级开关的名字（不带 --）取成集合。
func transportSwitchNames() map[string]bool {
	out := map[string]bool{}
	for _, s := range verbTransportFlags {
		out[strings.TrimLeft(s, "-")] = true
	}
	return out
}

// dummyArgsJSON 给一个动词的**全部**参数各造一个类型正确的值。
//
// 给全部（不只必填）：这样"某个可选参数的类型没被认识"也会被抓到 ——
// 而那种错恰恰只在有人真的用它时才暴露。
func dummyArgsJSON(t *testing.T, f *tzs.SpecFn) json.RawMessage {
	t.Helper()
	obj := map[string]any{}
	for _, p := range f.Args {
		obj[p.Name] = dummyValue(p)
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("%s：造参数失败：%v", f.Name, err)
	}
	return b
}

func dummyValue(p *tzs.Param) any {
	switch p.Type {
	case tzs.TypeInt:
		return 1
	case tzs.TypeBool:
		return true
	case tzs.TypeEnum:
		if len(p.Values) > 0 {
			return p.Values[0]
		}
		return "x"
	case tzs.TypePathList, tzs.TypeStrList:
		return []string{"x"}
	case tzs.TypeAttrs:
		// `attrs` 是**对象**（属性名 → 字符串值），不是字符串。少了这一支，set_spec_attrs /
		// set_layout_attrs 会给本地定型判死 —— 这条测试从此一直红，而它归 TTZS_E2E，默认不跑，
		// 所以直到 2026-09-25 才被看见（见到它红的那一刻，先怀疑的对象是当天的改动，不是它）。
		return map[string]string{"can_edit": "N"}
	}
	return "x"
}

func countParams(m *tzs.Manifest) int {
	n := 0
	for _, f := range m.Fns {
		n += len(f.Args)
	}
	return n
}

// TestE2EManifestAdvertisesImpliedCodes —— 声明的 `errors[]` 必须含盖**声明本身就蕴含**的那些码。
//
// 为什么值得一条：这类偏离的形态是"能力真的存在、词汇表里却没有"，后果是照 `--help` 写分支的
// 调用方**漏掉那个码** —— 而漏掉 `E_NO_OP` 恰恰会把"什么也没改"当成失败（本轮正在修的那个）。
// 2026-09-25 实测到两处：八个能答 no-op 的动词（语义 / Action / 工具那几族）从没广告过
// `E_NO_OP`；`set_spec_description` 能答 `E_NO_SPEC_NODE` 而它不在表里。
//
// 前三条不变量**从声明算出来**（收 handle / 收组件路径 / 是写动词且收 kind），所以它们不需要
// 再维护一份名单 —— 引擎的 `AdvertisedErrors` 就是按同样三条算的，这条测试是它的**外部复核**：
// 引擎自己算错了（比如把 `Mutating` 判反）这里会红。第四条算不出来，见 noopVerbs。
func TestE2EManifestAdvertisesImpliedCodes(t *testing.T) {
	m := requireRealManifest(t)
	problems := impliedCodeProblems(m)
	for _, p := range problems {
		t.Error(p)
	}
	t.Logf("查了 %d 个动词的蕴含码、%d 个 no-op 动词", len(m.Fns), len(noopVerbs))
}

// impliedCodeProblems 把"声明蕴含了某个码、`errors[]` 却没广告"（以及反向的"广告了不该有的"）
// 逐个列出来。抽成**纯函数**是为了能用**手造的 manifest** 证明它真的会红：
// 只跑真引擎的那一版在修好的当天永远是绿的，而没见过它红的断言不算防线 —— 这个仓库的既有做法。
func impliedCodeProblems(m *tzs.Manifest) []string {
	var out []string
	advertises := func(f *tzs.SpecFn, code string) bool {
		for _, c := range f.Errors {
			if c == code {
				return true
			}
		}
		return false
	}
	inList := map[string]bool{}
	for _, n := range noopVerbs {
		inList[n] = true
	}

	for _, f := range m.Fns {
		var handle, componentPath, kindParam bool
		for _, p := range f.Args {
			if p.Type == tzs.TypeHandle {
				handle = true
			}
			if p.Role == tzs.RoleComponentPath {
				componentPath = true
			}
			if p.Type == tzs.TypeKind {
				kindParam = true
			}
		}
		implied := map[string]bool{
			tzs.CodeNoHandle:     handle,
			tzs.CodePathNotFound: componentPath,
			tzs.CodeNoSpecNode:   kindParam && f.Writes,
		}
		for code, want := range implied {
			if want && !advertises(f, code) {
				out = append(out, fmt.Sprintf("%s：声明蕴含 %s（handle=%v 组件路径=%v kind=%v writes=%v），"+
					"errors[] 里却没有：%v", f.Name, code, handle, componentPath, kindParam, f.Writes, f.Errors))
			}
		}
		// 反向：读动词不该广告 no-op 类的成功码（它们是"没改任何值"的结局，只有写动词才有）。
		if !f.Writes && advertises(f, tzs.CodeNoOp) {
			out = append(out, fmt.Sprintf("%s 不写模型，却广告了 %s", f.Name, tzs.CodeNoOp))
		}

		// 第四条：能在 noopVerbs 里的必须广告 E_NO_OP，**不在的必须不广告**。
		// 后半条同样重要：空广告会让照 `--help` 写分支的调用方写出一个永远走不到的分支。
		got := advertises(f, tzs.CodeNoOp)
		if got && !inList[f.Name] {
			out = append(out, fmt.Sprintf("%s 广告了 %s，但不在 noopVerbs 里：要么它答不了 no-op（那条广告是空的，"+
				"写下这个分支的调用方永远走不到），要么它答得了 —— 那就把它加进 noopVerbs（并写清在哪个文件读到的）",
				f.Name, tzs.CodeNoOp))
		}
		if !got && inList[f.Name] {
			out = append(out, fmt.Sprintf("%s 在 noopVerbs 里，errors[] 却没有 %s：%v", f.Name, tzs.CodeNoOp, f.Errors))
		}
	}
	return out
}

// TestImpliedCodeProblemsFiresOnDoctoredManifests —— 上一条的"见过它红"。
//
// 不用引擎：手造 manifest，五种毛病各来一次，断言**每一条都被点名**（不是"有问题"，
// 而是"点到那个动词、那个码"）。少了这条，上一条测试会在某天悄悄退化成恒绿。
func TestImpliedCodeProblemsFiresOnDoctoredManifests(t *testing.T) {
	handle := func() *tzs.Param { return &tzs.Param{Name: "handle", Type: tzs.TypeHandle, Required: true} }
	split := func() []*tzs.Param {
		return []*tzs.Param{handle(), {Name: "path", Type: tzs.TypePath, Role: tzs.RoleComponentPath}}
	}

	cases := []struct {
		name string
		fn   *tzs.SpecFn
		want string // 期望被点到的码；空 = 期望无问题
	}{
		{"正常：handle 与组件路径都广告了", &tzs.SpecFn{
			Name: "ok", Writes: true, Args: split(),
			Errors: []string{tzs.CodeNoHandle, tzs.CodePathNotFound},
		}, ""},
		{"收 handle 却没广告 E_NO_HANDLE", &tzs.SpecFn{
			Name: "no-handle", Writes: false, Args: []*tzs.Param{handle()}, Errors: nil,
		}, tzs.CodeNoHandle},
		{"收组件路径却没广告 E_PATH_NOT_FOUND", &tzs.SpecFn{
			Name: "no-path", Writes: false,
			Args:   []*tzs.Param{handle(), {Name: "path", Type: tzs.TypePath, Role: tzs.RoleComponentPath}},
			Errors: []string{tzs.CodeNoHandle},
		}, tzs.CodePathNotFound},
		{"写动词收 kind 却没广告 E_NO_SPEC_NODE", &tzs.SpecFn{
			Name: "set_spec_attr", Writes: true,
			Args:   []*tzs.Param{handle(), {Name: "kind", Type: tzs.TypeKind}},
			Errors: []string{tzs.CodeNoHandle},
		}, tzs.CodeNoSpecNode},
		{"读动词收 kind：**不该**广告 E_NO_SPEC_NODE", &tzs.SpecFn{
			Name: "list_spec_nodes", Writes: false,
			Args:   []*tzs.Param{handle(), {Name: "kind", Type: tzs.TypeKind}},
			Errors: []string{tzs.CodeNoHandle},
		}, ""},
		{"读动词广告了 E_NO_OP", &tzs.SpecFn{
			Name: "list_open", Writes: false, Args: []*tzs.Param{handle()},
			Errors: []string{tzs.CodeNoHandle, tzs.CodeNoOp},
		}, tzs.CodeNoOp},
		{"能答 no-op 却不在名单里", &tzs.SpecFn{
			Name: "some_new_writer", Writes: true, Args: []*tzs.Param{handle()},
			Errors: []string{tzs.CodeNoHandle, tzs.CodeNoOp},
		}, tzs.CodeNoOp},
		{"在名单里却没广告", &tzs.SpecFn{
			Name: "set_code_template", Writes: true, Args: []*tzs.Param{handle()},
			Errors: []string{tzs.CodeNoHandle},
		}, tzs.CodeNoOp},
	}

	for _, c := range cases {
		problems := impliedCodeProblems(&tzs.Manifest{Fns: []*tzs.SpecFn{c.fn}})
		if c.want == "" {
			if len(problems) != 0 {
				t.Errorf("%s：不该有问题，得到 %v", c.name, problems)
			}
			continue
		}
		if len(problems) == 0 {
			t.Errorf("%s：该报 %s，一个都没报", c.name, c.want)
			continue
		}
		hit := false
		for _, p := range problems {
			if strings.Contains(p, c.want) && strings.Contains(p, c.fn.Name) {
				hit = true
			}
		}
		if !hit {
			t.Errorf("%s：报的话里没同时点到 %s 与 %s：%v", c.name, c.fn.Name, c.want, problems)
		}
	}
}

// noopVerbs 是**读过 Fns/ 之后确认会回 `noop:true` + `code:"E_NO_OP"` 的动词**。
//
// 它算不出来：参数声明里没有"这个动词能不能答 no-op"这件事（有的话它就该像 `role` 一样进声明，
// 而不是写在这里）。所以加一个能答 no-op 的写动词时**来加一行**，并写下在哪个文件读到的。
//
// 十二处的出处（2026-09-25 逐个读过）：
//
//	Attr.cs       set_spec_attr / set_spec_attrs / set_layout_attr / set_layout_attrs（Noop()）
//	              set_tree_source（Noop(...,"tree/"+element,...)）/ rename_component（就地构造）
//	Action.cs     add_action / delete_action / set_action_types（Noop(id, fields, note)）
//	Semantic.cs   set_local_string / set_spec_description（Noop(res, note)）
//	PageTab.cs    set_code_template（就地构造；2026-09-25 之前缺 noop/code 两个标记）
//
// 反例：`set_cited` 不在这里 —— 它无条件跑命令并回报 wasCited/cited，没有"已经是这个值"的答案。
var noopVerbs = []string{
	"add_action", "delete_action", "rename_component", "set_action_types", "set_code_template",
	"set_layout_attr", "set_layout_attrs", "set_local_string", "set_spec_attr", "set_spec_attrs",
	"set_spec_description", "set_tree_source",
}

// TestE2ENarrowingHintFollowsTheManifest —— "先收窄/先过滤"提示只能挂在**可选 + 返回清单**的参数上。
//
// 为什么值得一条：这类提示是 `--help` 里唯一一句"教你怎么少花 context"的话，所以它一旦
// 挂错地方，代价是**照它做反而更糟**。2026-09-25 的评测实测到三处挂错：`open --help` 印
// "先收窄：path 能减少返回"（path 必填，不给开不了表）、`find_component`（query 必填）、
// `get_component`（返回单个 el，没有"全量"可收窄）。三条判据都是 manifest 里现成的事实。
func TestE2ENarrowingHintFollowsTheManifest(t *testing.T) {
	m := requireRealManifest(t)
	hinted := 0
	for _, f := range m.Fns {
		h := filterHint(f)
		if h == "" {
			continue
		}
		hinted++
		n := firstNarrowingParam(f)
		p := f.Param(n)
		if p == nil {
			t.Errorf("%s：提示点名 %q，manifest 里没有这个参数", f.Name, n)
			continue
		}
		if p.Required {
			t.Errorf("%s：提示点名了必填参数 %q（必填的是输入，不是过滤器）：%s", f.Name, n, h)
		}
		if !strings.HasPrefix(f.Returns, "list") {
			t.Errorf("%s：返回 %q 不是清单，没有全量可收窄：%s", f.Name, f.Returns, h)
		}
	}
	// 三个被实测点过名的，必须一直没有提示 —— 这条比上面的通判更硬：它们换不回旋余地。
	for _, name := range []string{"open", "find_component", "get_component"} {
		f := m.ByName(name)
		if f == nil {
			t.Errorf("manifest 里没有 %s？", name)
			continue
		}
		if h := filterHint(f); h != "" {
			t.Errorf("%s 曾被实测挂过一句不成立的收窄提示，现在又有了：%s", name, h)
		}
	}
	// 一处都没提示不判失败（把提示整段删掉也是允许的改法），但要让人在输出里看得见它没退化成空转。
	t.Logf("真 manifest：%d 个动词里 %d 个有收窄提示", len(m.Fns), hinted)
	if hinted == 0 {
		t.Errorf("一个收窄提示都没有 —— 这条断言成了空转（list_columns / list_tables 该有的）")
	}
}
