package tzs

// manifest.go —— `tzs-server --manifest` 的类型化视图。
//
// **这是参数定型的唯一权威来源，绝不要在 Go 侧抄第二份。** 引擎的 Manifest.cs 是
// 唯一产地：`--help`、服务端参数校验、以后的 MCP 工具表全部从它生成。我们只要在 Go 侧
// 再写一份 `map[string][]Param`，它就会在引擎加/改参数时静默过期 —— 表现是
// 「明明 manifest 里有这个参数，工具却说你打错了」，而两边的源码各自看都对。
//
// 于是本文件只做三件事：解 JSON、按 fn 查表、渲染 help。**没有任何业务知识**
// （比如「七种 kind 是哪七种」不在这里，也不在 Go 侧任何地方：`kind` 参数的合法集
// 引擎没写进 manifest，hardcode 一份就等于抄了第二份；引擎会用 E_BAD_PARAM 拒绝，
// 而 kind=validation 的退出码也是 2，和本地拦下来一模一样）。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// FnStop 是传输级命令的名字。它**不在 manifest 里**（引擎的 Rpc.Handle 在查表之前
// 就把它截走了）：`stop` 问的是守护进程本身，不是设计器。
//
// 所以 BuildArgsFromJSON("stop") 必然报「未知函数」—— 那是正确行为，命令层必须自己特判它。
const FnStop = "stop"

// Param 类型名。取值集合来自引擎的 Manifest.TypeName（唯一产地）。
//
// 契约给的清单漏了 `handle` —— 而它在真 manifest 里出现 44 次（每个需要句柄的函数都
// 有一个必填的 handle 参数）。它按字符串处理即可（形如 "h1"），所以类型表必须收它。
const (
	TypeString = "string"
	TypePath   = "path"
	TypeInt    = "int"
	TypeBool   = "bool"
	TypeKind   = "kind"
	// describe_kind 自己的词汇：七种规格节点，或 layout（那张表单上布局属性名的并集）。
	// 与 TypeKind 分开是有意的 —— 三个写动词的 kind 收不下 layout（没有布局规格节点可写）。
	TypeKindOrLayout = "kind-or-layout"
	TypeAttr         = "attr"
	TypeEnum         = "enum"
	TypeHandle       = "handle"
	TypePathList     = "path[]"
	TypeStrList      = "string[]"
	// 「一次改多个属性」的参数：`{"属性名":"值", …}`。和 path[]/string[] 一样是**容器**，
	// 所以本地必须认形状（对象、值是字符串）—— 把它当字符串处理会让一个合法调用被本地判死。
	TypeAttrs = "attrs"
)

// KnownType 报告 t 是不是引擎会产出的类型名。
//
// 用途只有一个：argmap 里给未知类型留一条「按字符串处理」的退路，并由测试钉住
// 「已知集合 == 引擎 TypeName 的输出集合」。**不要**用它去拒绝请求：引擎加一个新类型时
// 我们宁可按字符串发过去（引擎会用 E_BAD_PARAM 明确拒绝），也不要让旧客户端把一个
// 合法的新参数判成「未知类型」。
func KnownType(t string) bool {
	switch t {
	case TypeString, TypePath, TypeInt, TypeBool, TypeKind, TypeKindOrLayout, TypeAttr, TypeEnum, TypeHandle, TypePathList, TypeStrList, TypeAttrs:
		return true
	}
	return false
}

// 参数的角色（= 引擎 `Param.Role`，见 engine/src/Designer/SpecFn.cs 的 Role）。
//
// 为什么需要它：参数名来自 C# 函数签名，同一个概念在不同动词里叫不同名字
// （`open.path` vs `field_add.file`），同一个名字在不同动词里意思也不同（`path` 在 `open` 上
// 是**包路径**，在别处是表单内的 name-path）。角色的用途是**机械**的，不是文档：
//
//	① 生成的示例按角色挑占位符 —— 从前只看参数名是不是 `file`，于是 `open.path` 与
//	   `field_add.out` 这两个**包路径**都被教成 `<name-path>`；
//	② 一个动词会不会回 `E_PATH_NOT_FOUND` 取决于它收的是不是 name-path（引擎侧
//	   `AdvertisedErrors` 按 role 推导）。
const (
	RolePackagePath   = "package-path"
	RoleComponentPath = "component-path"
	RoleDirection     = "direction"
	RoleActionID      = "action-id"
	RoleNewName       = "new-name"
)

// Param 是 manifest 里的一项参数（引擎的 Param / SPEC §11.24(c) 的 args[] 元素）。
type Param struct {
	Name     string   `json:"n"`
	Type     string   `json:"t"`
	Required bool     `json:"req"`
	Desc     string   `json:"desc,omitempty"`
	Values   []string `json:"values,omitempty"` // 仅 enum：静态合法集
	// Role 是"这个参数是什么意思"，只在名字看不出来时声明（见上面那组常量）。
	Role string `json:"role,omitempty"`
	// From 仅 attr：合法集要现场从模型里取（engine 的 DescribeFrom）。
	// 它的取值是 "spec:<kind>" / "layout" 这种模板串，**不是**一份可列举的清单 ——
	// 所以本地无从校验，只能让它上线（见 argmap 的本地校验清单）。
	From string `json:"from,omitempty"`
}

// SpecFn 是一个函数的声明。
type SpecFn struct {
	Name        string   `json:"fn"`
	Group       string   `json:"group"`
	Desc        string   `json:"desc"`
	Writes      bool     `json:"writes"` // 改名前的 Mutating：首次调用会把句柄从 Loaded 推到 Mutable
	Slow        bool     `json:"slow"`   // 只有 validate 为真（670 元素表单实测 10.4 s）
	Returns     string   `json:"returns"`
	NeedsHandle bool     `json:"needsHandle"`
	Args        []*Param `json:"args"`
	Errors      []string `json:"errors"`
}

// Param 按名查本函数的参数声明；不在声明里时返回 nil。
func (f *SpecFn) Param(name string) *Param {
	for _, p := range f.Args {
		if p.Name == name {
			return p
		}
	}
	return nil
}

// RequiredParams 返回必填参数名（用于报错文案「缺 X」时把话说全）。
func (f *SpecFn) RequiredParams() []string {
	var out []string
	for _, p := range f.Args {
		if p.Required {
			out = append(out, p.Name)
		}
	}
	return out
}

// Manifest 是 `--manifest` 的类型化视图（顶层是**数组**，带缩进）。
type Manifest struct {
	Fns []*SpecFn

	byName map[string]*SpecFn
}

// ParseManifest 解析 `--manifest` 的输出。
//
// 校验只覆盖「这份 manifest 自己自洽吗」，不覆盖「引擎实现了哪些函数」：
// 引擎会先声明全部 ~50 个函数，**再**逐个补实现，没实现的回 E_NOT_IMPLEMENTED
// （kind=internal，退 1）。那是诚实答案，我们不该在这里把它过滤掉 ——
// 过滤会让「这个函数以后会有」变成「没有这个函数」，两者对调用方的含义完全不同。
func ParseManifest(b []byte) (*Manifest, error) {
	var fns []*SpecFn
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&fns); err != nil {
		return nil, &UsageError{Msg: "拉到的 --manifest 不是函数表 JSON", Detail: []string{err.Error()}}
	}
	if len(fns) == 0 {
		// 空数组意味着引擎那侧的表是空的 —— 那它一定不是我们以为的那个 exe。
		return nil, &UsageError{Msg: "函数表是空的（拿到的像是另一个 exe 的输出）"}
	}
	m := &Manifest{Fns: fns, byName: make(map[string]*SpecFn, len(fns))}
	for i, f := range fns {
		if f == nil || strings.TrimSpace(f.Name) == "" {
			return nil, &UsageError{Msg: fmt.Sprintf("函数表第 %d 项没有 fn 字段", i)}
		}
		if _, dup := m.byName[f.Name]; dup {
			return nil, &UsageError{Msg: "函数表里有重名：" + f.Name}
		}
		m.byName[f.Name] = f
	}
	return m, nil
}

// FetchManifest 跑一次 `tzs-server --manifest` 并把结果解析出来。
//
// 失败一律是用法错（退出码 2）：契约把「manifest 拉不到」归在用法错那一栏
// （见 UsageError 的注释）。**不 Boot**，所以它只值几十毫秒。
func FetchManifest(ctx context.Context, exe string) (*Manifest, error) {
	if strings.TrimSpace(exe) == "" {
		return nil, &UsageError{Msg: "没有指定 tzs-server 的路径（--exe / 配置）"}
	}
	out, err := runEngine(ctx, exe, "--manifest")
	if err != nil {
		return nil, &UsageError{Msg: "拉不到函数表（" + exe + " --manifest）", Detail: []string{err.Error()}}
	}
	return ParseManifest(out)
}

// ByName 按函数名查声明；未命中返回 nil（调用方据此报「未知函数」→ 退出 2）。
func (m *Manifest) ByName(name string) *SpecFn {
	if m == nil {
		return nil
	}
	return m.byName[name]
}

// Names 返回全部函数名（manifest 顺序）。
func (m *Manifest) Names() []string {
	if m == nil {
		return nil
	}
	out := make([]string, 0, len(m.Fns))
	for _, f := range m.Fns {
		out = append(out, f.Name)
	}
	return out
}

// Groups 按 manifest 顺序返回分组名（help 的分节顺序就是它）。
func (m *Manifest) Groups() []string {
	if m == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, f := range m.Fns {
		if seen[f.Group] {
			continue
		}
		seen[f.Group] = true
		out = append(out, f.Group)
	}
	return out
}

// Index 渲染**动词索引**：分组 + 名字，仅此而已。
//
// 为什么不把参数一起打出来：每个动词的参数是它自己的契约，`tt dev tzs <动词> --help`
// 按需给一个。把**全部**动词的参数一次摊开（真表约 36 KB）既淹没人，也等于把引擎的
// **函数表**整个交到调用方手里 —— 对外只该有"动词"这一层。
// （动词数不写在注释里：它是 `len(m.Fns)`，手写的那个数字一定会漂。）
//
// 从 manifest 生成而不是手写：手写的那份一定会漂移，而漂移的表现是
// 「索引里明明有这个名字，敲上去却是未知动词」。
func (m *Manifest) Index(w io.Writer) {
	if m == nil {
		fmt.Fprintln(w, "（没有动词表）")
		return
	}
	fmt.Fprintf(w, "%d 个动词（会改模型的标 [写]，慢的标 [slow]；参数：tt dev tzs <动词> --help）：\n", len(m.Fns))
	for _, g := range m.Groups() {
		fmt.Fprintf(w, "\n[%s]\n", g)
		for _, f := range m.Fns {
			if f.Group != g {
				continue
			}
			marks := ""
			if f.Slow {
				marks += " [slow]"
			}
			if f.Writes {
				marks += " [写]"
			}
			fmt.Fprintf(w, "  %s%s\n", f.Name, marks)
		}
	}
	fmt.Fprintf(w, "\n%s 是传输级命令，不在本表里。\n", FnStop)
}

//---------------------------------------------------------------------------

// runEngine 跑一次引擎的「不 Boot 开关」（--manifest / --pipe-name），返回 stdout。
//
// stdin 显式置空：不置空且父进程的 stdin 被重定向时，引擎会自己判定成 stdio 模式
// （tzs-server.cs 的 Console.IsInputRedirected）—— 对 --manifest / --pipe-name 无害
// （它们在模式判定之前就 return 了），但把 stdin 给它是个坏习惯：哪天多一个会 Boot 的
// 开关，就会变成「管道永远不出现」那种查半天的问题。
//
// 超时 30 s：这两个开关都是几十毫秒的活，给 30 s 是为了不把一次磁盘抖动判成失败，
// 同时保证一个卡住的 exe 不会把工具挂死。
func runEngine(ctx context.Context, exe string, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, exe, args...)
	cmd.Stdin = nil
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return out, nil
}
