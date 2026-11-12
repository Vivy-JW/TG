// Package at 实现 tt 的 "目标" 公共设施：一个统一的取值文法、一份按命令归类的作用域表、
// 以及一个“现场讲清这条命令该怎么填”的 --at-list。
//
// 三条不变式（动它之前先读）：
//
//  1. **文法全局唯一**：只接受 `<环境>` 或 `<环境>@<企业>`，永远不接受会话引用 ——
//     指名会话是 `--session` 的活。各命令只能在作用域表里声明“我要哪一档”，
//     不允许自定义语法。
//  2. **解析 / 校验 / 报错 / 渲染只有这一份实现**。散出去就会变成“同一条命令两种说法”。
//  3. **--at-list 纯本地、永不拨 SSH**：值来自 config.json（环境）与 entdir 快照（企业），
//     快照过期如实标 stale。TG 只支持 TOP GP，T100 的 ENT 清单不再维护（GP 的据点清单用 `tg site list`）。
package at

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"tt/internal/cli/common"
)

// Scope 是“这条命令把 --at 用在哪”的唯一分类。
//
// 新增命令时到 scopes.go 的表里认领一档；不认领就是 None（接受但不生效是不允许的）。
type Scope int

const (
	// None 本命令没有目标参数：给了 --at 直接报错（不静默忽略）。
	None Scope = iota
	// Env 只接受 <环境>：连服务器/库，但不按企业解析账号（wslogs/mirror/form/tzs…）。
	Env
	// EnvEnt 接受 <环境> 或 <环境>@<企业>：要解析账号或按企业收窄，企业可选（db/ents/dict…）。
	EnvEnt
	// EnvEntReq 与 EnvEnt 同形，但企业**必需**：建立会话/起作业/查 SQL 都靠企业定账号。
	EnvEntReq
	// EnvFilter 会话寻址类：<环境> 只用来收窄“在哪条会话上动手”；@企业 拒绝，
	// 因为“断言那条会话属于某企业”已经有写法：--session <环境>/<企业>。
	EnvFilter
)

func (s Scope) String() string {
	switch s {
	case Env:
		return "env"
	case EnvEnt:
		return "env+ent"
	case EnvEntReq:
		return "env+ent(required)"
	case EnvFilter:
		return "env-filter"
	default:
		return "none"
	}
}

// 全局开关（由根命令注册，各命令只读）。
var (
	// Raw --at 的原始取值。
	Raw string
	// List --at-list 是否存在（真正处理在 Execute 的预扫描里，见 ListTarget）。
	List bool
	// Ent --at 解析出的企业编号（0 = 未带企业）。
	Ent int
	// Site --at 解析出的据点/资料库编号（GP 环境；空 = 没带）。
	// 存小写：azp01 是大写（COF），而解析成 schema 后是 cof_demo。
	Site string
)

// Value 是 --at 解析后的结果。
type Value struct {
	Raw     string
	Env     string
	Ent     int
	HasEnt  bool
	Site    string
	SiteRaw string // 原样（保留大小写，只用于报错文案）
	HasSite bool
}

// siteRE 据点/资料库编号的合法字符（T100 的企业编号是纯数字，GP 的是字母开头）。
//
// 限死字符集是因为它会拼进 SQL 当 schema 名（erpdb 的 Query 只收 SQL 文本、
// 没有绑定参数），而且它最终会成为 --at 的一部分。
var siteRE = regexp.MustCompile(`^[A-Za-z0-9_$#-]{1,20}$`)

// Parse 解析 --at 的取值：`<环境>`、`<环境>@<企业>`，或 GP 的 `<环境>@<据点>`。
//
// 从**最后一个 @** 右侧取值：纯数字=企业编号（T100），字母开头=据点/资料库编号
// （GP）。环境名里含 @ 也不会被切错。右侧两边都不像时按用法错报 ——
// 不猜“那大概是个没写企业的环境名”。
//
// 这里不做产品判断（Parse 是纯函数、不读配置）：同一个写法在两类环境含义不同，
// 由 Apply 按环境产品分派。
func Parse(raw string) (Value, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Value{}, fmt.Errorf("--at 不能为空：写 <环境> 或 <环境>@<企业>（不知道填什么就加 --at-list）")
	}
	i := strings.LastIndex(s, "@")
	if i < 0 {
		return Value{Raw: s, Env: s}, nil
	}
	env := strings.TrimSpace(s[:i])
	num := strings.TrimSpace(s[i+1:])
	if env == "" {
		return Value{}, fmt.Errorf("--at %q 少了环境名：写 <环境>@<企业>", s)
	}
	if n, err := strconv.Atoi(num); err == nil {
		if n <= 0 {
			return Value{}, fmt.Errorf("--at %q 的企业部分不是正整数（%q）；写法是 <环境>@<企业>，如 主机109@99（不知道填什么就加 --at-list）", s, num)
		}
		return Value{Raw: s, Env: env, Ent: n, HasEnt: true}, nil
	}
	if siteRE.MatchString(num) {
		return Value{Raw: s, Env: env, Site: strings.ToLower(num), SiteRaw: num, HasSite: true}, nil
	}
	return Value{}, fmt.Errorf("--at %q 的 %q 既不是企业编号（纯数字），也不是据点/资料库编号（字母、数字、_ $ # -，20 字内）；"+
		"写法是 <环境>@<企业>（T100）或 <环境>@<据点编号>（TOP GP），不知道填什么就加 --at-list", s, num)
}

// Apply 是根命令的 PersistentPreRunE：把 --at 落到各命令本来就读取的地方。
//
// 落点只有两个，都是既有开关的“同一个值”：
//   - 环境 → common.Env（与 --env 同一个变量，所有命令原样生效）；
//   - 企业 → 命令自己的 --ent 变量（命令若声明了该开关；没声明就是作用域配错了）。
//
// 这样各命令一行都不用改，也不会出现“--at 与 --env 各读一份”。
func Apply(cmd *cobra.Command, _ []string) error {
	raw := strings.TrimSpace(Raw)
	if raw == "" {
		return nil
	}
	scope := ScopeOf(cmd)
	v, err := Parse(raw)
	if err != nil {
		return err
	}

	// 同一个 @ 写法在两类环境含义不同，所以先看环境是哪个产品再分派。
	// product 为空按 T100 处理（老配置没有这个字段，而 GP 环境一定会写 gp）。
	prod := productOf(v.Env)
	switch {
	case v.HasSite && prod != productGP:
		return fmt.Errorf("--at %q：环境 %s 是 T100，@ 右边只能是企业编号（纯数字）；%q 更像据点/资料库编号 —— "+
			"是不是把据点码填到企业号的位置了？", raw, v.Env, v.Site)
	case v.HasEnt && prod == productGP:
		return fmt.Errorf("--at %q：环境 %s 是 TOP GP，@ 右边写据点编号（如 22主机@COF），不是数字企业号；"+
			"据点清单：tg site list --env %s", raw, v.Env, v.Env)
	}

	switch scope {
	case None:
		return rejectNone(cmd)
	case Env:
		if v.HasEnt || v.HasSite {
			return fmt.Errorf("%s 只接受 <环境>：去掉 @%s（这条命令不按企业/据点收窄）", cmd.CommandPath(), rightOf(v))
		}
	case EnvFilter:
		if v.HasEnt || v.HasSite {
			return fmt.Errorf("%s 的 --at 只用来收窄“在哪个环境里找会话”，不带企业/据点；"+
				"要断言那条会话属于某个企业，请用 --session %s/%s", cmd.CommandPath(), v.Env, rightOf(v))
		}
	case EnvEnt:
		// 环境必给，企业/据点可选
	case EnvEntReq:
		if !v.HasEnt && !v.HasSite {
			if prod == productGP {
				return fmt.Errorf("%s 需要 <环境>@<据点编号>：据点决定查哪个资料库；现成的取值：%s --at-list",
					cmd.CommandPath(), cmd.CommandPath())
			}
			return fmt.Errorf("%s 需要 <环境>@<企业>：企业编号决定用哪个账号连库；现成的取值：%s --at-list",
				cmd.CommandPath(), cmd.CommandPath())
		}
	}

	// 与 --env / --conn 冲突不合并（一致放行，方便新旧写法混用）。
	if cur := strings.TrimSpace(common.Env); cur != "" && cur != v.Env {
		return fmt.Errorf("--at 与 --env 给了两个不同的环境（%s / %s）：二选一", v.Env, cur)
	}

	// local 有两种写法：裸 `local`（T100 的本地字典副本）与 `local:<环境>`
	// （GP 的本地字典副本，按环境分目录）。从前只认前者，后者会被当成环境名
	// 去查配置，报“不存在的环境 local:22主机” —— 而 --env local:<环境> 是通的。
	switch {
	case v.Env == "local" || strings.HasPrefix(v.Env, "local:"):
		if !allowsLocal(cmd) {
			return fmt.Errorf("--at %s 只对 dict 的查询命令有效（表示本地字典副本）；%s 不接受，改用 --at <环境>",
				v.Env, cmd.CommandPath())
		}
		if rest := strings.TrimPrefix(v.Env, "local:"); v.Env != "local" && strings.TrimSpace(rest) == "" {
			return fmt.Errorf("--at %q 少了环境名：写 local:<环境>（如 local:22主机）", v.Env)
		}
	default:
		if err := checkEnv(cmd, v.Env); err != nil {
			return err
		}
	}

	common.Env = v.Env
	if v.HasSite {
		Site = v.Site
	}
	if v.HasEnt {
		Ent = v.Ent
		if f := cmd.Flag("ent"); f != nil {
			want := strconv.Itoa(v.Ent)
			if f.Changed {
				if got := strings.TrimSpace(f.Value.String()); got != want {
					return fmt.Errorf("--at 的企业（%s）与 --ent（%s）不一致：二选一，或保持一致", want, got)
				}
			} else if err := f.Value.Set(want); err != nil {
				return fmt.Errorf("把 --at 的企业写进 --ent 失败: %w", err)
			}
		}
		if err := checkEnt(cmd, v.Env, v.Ent); err != nil {
			return err
		}
	}
	return nil
}

// rejectNone 是本命令不认 --at 时的说法。会话类命令要给可执行的替代写法。
func rejectNone(cmd *cobra.Command) error {
	path := pathOf(cmd)
	switch {
	case strings.HasPrefix(path, "session"):
		return fmt.Errorf("%s 不接受 --at：会话身份来自 --session（tt session ls --json 可看全部）；\n"+
			"环境/企业跟着会话走，不用单独给", cmd.CommandPath())
	case strings.HasPrefix(path, "dev tzs"):
		return fmt.Errorf("%s 不接受 --at：它的参数由 tzs 自己的解析器接管 —— 环境用 --env，工作区用 --workspace", cmd.CommandPath())
	default:
		return fmt.Errorf("%s 不接受 --at：它不连接环境/企业（要操作会话或查库，换一条命令）", cmd.CommandPath())
	}
}

// rightOf 取 --at 里 @ 右边那一段（只用于报错文案）。
func rightOf(v Value) string {
	if v.HasEnt {
		return strconv.Itoa(v.Ent)
	}
	if v.SiteRaw != "" {
		return v.SiteRaw
	}
	return v.Site
}

func allowsLocal(cmd *cobra.Command) bool {
	return strings.HasPrefix(pathOf(cmd), "dict")
}

// pathOf 返回相对根命令的路径（"debug start"）；根命令返回 ""。
func pathOf(cmd *cobra.Command) string {
	root := cmd.Root()
	full := cmd.CommandPath()
	name := root.Name()
	if full == name {
		return ""
	}
	return strings.TrimPrefix(full, name+" ")
}
