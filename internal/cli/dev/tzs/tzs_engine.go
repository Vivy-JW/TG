// tzs_engine.go —— `tt dev tzs` 的**读写**动词：把 tt 接到 engine/ 那个 C# 引擎上。
//
// 与 tzs.go 里 `export` 的分工：export 是纯解压、不碰引擎，只读；这里全部要引擎。
//
// 为什么这里可以有写路径，而 tzs.go 的头注释写着「永远不写回」——那条红线是**有前提**的：
//
//	表单（SPEC）由设计器的表单设计器维护，tdev 没有对应的模型与验收样本；硬写就是拿真实包赌。
//
// 现在那个前提不成立了：engine/ 就是**设计器自己的代码**（它 LoadFrom 设计器的程序集，
// 布局属性走设计器自己的 XmlElement 索引器，验收用设计器自己的校验器与不动点）。所以在
// tdev 里第一次有了「对应模型」；红线随之改成「导出仍只读、要写走引擎」，而不是「不许写」。
//
// 三条纪律（与 A1/A2/A3 同源）：
//
//  1. **不自己重算引擎的任何东西。** 函数表、参数类型、管道名全部问引擎（--manifest /
//     --pipe-name）。管道名含引擎程序集的 MVID，重算一百行且算错了不响。
//  2. **请求一旦上线，绝不重试。** 协议无幂等键，而这些函数都在改设计器内存里的模型；
//     守护进程中途死掉时重试是在赌「上一次写进去了没有」。
//  3. **工作区没有缺省。** 它按环境配（hosts.sshs[].workspace），引擎内置的默认工作区是一个
//     真实客户目录，落到它上面会去 Boot 别人的包。解析顺序末端是**拒绝**，不是回落。
package tzs

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tt/internal/cli/dev/common"
	"tt/internal/config"
	"tt/internal/dev/tzs"
)

// tzsSettings 读配置里的 tzs 节（读不到就是零值：配置缺失绝不让命令失败）。
func tzsSettings() config.TzsSettings {
	if path, err := config.ResolvePath("", true); err == nil && path != "" {
		if root, err := config.Load(path); err == nil {
			return root.Tzs
		}
	}
	return config.TzsSettings{}
}

// tzsEngineExe 解析引擎 exe 的路径。**不碰工作区。**
//
// 与 tzsEngineOptions 分开是必要的：问函数表（`--manifest`）只依赖 exe。
// 把工作区也捆进来，会让两件事在"还没配工作区"的新机器上变成退 5 ——
// "动词名打错"（本该退 2）和"`<动词> --help`"（本该退 0，它就是说明书本身）。
func tzsEngineExe() (string, error) {
	// 候选与缺省只有一份实现 —— config.EngineExe(统一路径管理器):
	// tzs.serverExe 覆盖优先,否则 <tt.exe 目录>	zs	zs-server.exe。
	if p := config.EngineExe(tzsSettings().ServerExe); p != "" {
		return p, nil
	}
	return "", fmt.Errorf("定位不到 tt.exe 自身")
}

// tzsExecOptions 解析出一次引擎调用需要的四样东西；**缺工作区不报错**（留给调用方决定）。
//
// 分成"宽松 / 严格"两版是有意的：
//   - 问函数表（fns / manifest / `<动词> --help`）只需要 exe —— 它们是**动词索引与说明书**，
//     在还没配工作区的新机器上必须能用，否则说明书锁在门里面；
//   - doctor 的职责就是回答"还差什么"，它自己先因为缺工作区退 5 的话，就永远走不到
//     它那条「工作区：未配置」的自检项（README 承诺的正是那一句）；
//   - 运行类动词走 tzsEngineOptions（严格版）：没有工作区就不该开工（见下）。
//
// envName 取自**命令行**(--env,由根命令的 persistent flag 落到 common.Env)。
// 工作区按环境配,所以没有环境就没有"该用哪个工作区"可言 —— 这里不接受空值兜底,
// 空值会在 tzsEngineOptions 里被拒(doctor 允许空,它只报告)。
func tzsExecOptions(wsFlag, envName string) (tzs.Options, error) {
	exe, err := tzsEngineExe()
	if err != nil {
		return tzs.Options{}, err
	}
	ws := wsFlag
	wsSrc := ""
	if ws != "" {
		wsSrc = "--workspace"
	}
	if ws == "" {
		ws = os.Getenv("TZSCLI_WS")
		if ws != "" {
			wsSrc = "环境变量 TZSCLI_WS"
		}
	}
	if ws == "" {
		ws, wsSrc = envWorkspace(envName)
	}
	workDir := ""
	if path, err := config.ResolvePath("", true); err == nil && path != "" {
		workDir = filepath.Dir(path) // 状态文件与守护进程日志落在 config.json 旁边
	}
	return tzs.Options{
		Exe:          exe,
		InstallDir:   os.Getenv("TZSCLI_INSTALL"),
		Workspace:    ws,
		WorkspaceSrc: wsSrc,
		WorkDir:      workDir,
	}, nil
}

// envWorkspace 指定环境的工作区与**它的来源**（"环境「主机109」" 这种说法，直接进
// doctor 的报告）。工作区按环境配（hosts.sshs[].workspace），所以这个函数**必须**拿到
// 环境名 —— 没有"默认环境"可回落（那正是这次要根除的猜测）。
// envName 为空时返回空串，由上层决定是报错还是只报告（doctor）。
func envWorkspace(envName string) (ws, src string) {
	if strings.TrimSpace(envName) == "" {
		return "", ""
	}
	path, err := config.ResolvePath("", true)
	if err != nil || path == "" {
		return "", ""
	}
	root, err := config.Load(path)
	if err != nil {
		return "", ""
	}
	ref, err := root.Hosts.Resolve(envName)
	if err != nil || ref == nil || ref.Env == nil {
		return "", ""
	}
	name := ref.Name
	if name == "" {
		name = "（未命名）"
	}
	return ref.Env.Workspace, "环境「" + name + "」"
}

// tzsEngineOptions 是**运行类**动词用的严格版。
//
// 工作区的解析顺序：--workspace flag > TZSCLI_WS 环境变量 > **当前环境的 workspace**
// （hosts.sshs[].workspace；“当前环境” = hosts.activeEnv，也就是设置页里星标那个）。
// 机器级没有工作区这一层：一台机器对着几个客户就有几个工作区。
// **末端拒绝**：三层都没有时报错（调用方退 5）并说清该配哪个键，绝不 spawn —— 引擎的缺省
// 是一个真实客户目录，落上去等于拿别人的表单当草稿纸。
func tzsEngineOptions(wsFlag, envName string) (tzs.Options, error) {
	o, err := tzsExecOptions(wsFlag, envName)
	if err != nil {
		return tzs.Options{}, err
	}
	if o.Workspace == "" {
		return tzs.Options{}, fmt.Errorf(
			"没有可用的工作区：--workspace、TZSCLI_WS、以及环境 %s 的 workspace 都是空的。\n"+
				"  引擎的缺省工作区是一个真实客户目录，所以这里拒绝启动而不是回落。\n"+
				"  要么用 --env <环境名> 指定一个配了工作区的环境（设置页「站点管理 → 环境 → 工作区目录」），\n"+
				"  要么直接给 --workspace <目录> / 设 TZSCLI_WS。",
			envLabel(envName))
	}
	return o, nil
}

// envLabel 报错文案里的环境名：没给就明确说"未指定"，别让用户以为程序知道。
func envLabel(envName string) string {
	if strings.TrimSpace(envName) == "" {
		return "(未指定 --env)"
	}
	return "「" + envName + "」"
}

// ---------- 动词索引：唯一的枚举入口 ----------
//
// fns / manifest 两个命令已经删掉：对外只留"动词"这一层，而**函数表**（名字 + 参数 + 类型 +
// 必填 + 取值集）是引擎自己的内部契约，不该整个摊给调用方。于是枚举只剩两条路：
//
//	tt dev tzs --help            静态用法 + （引擎可达时）按组列出动词名
//	tt dev tzs <敲错的名字>      同上一份索引，走 stderr
//
// 参数的契约按需给：`tt dev tzs <动词> --help`。

// printFn 渲染单个动词的参数表（`tt dev tzs <动词> --help`）。
//
// 顺序照 clig.dev 的建议：**示例优先**，然后参数，最后错误码 —— 人们读示例比读文档多。
// 参数本身用 JSON 打 —— 它的字段（n/t/req/values/from）就是 manifest 的原样，
// 手写一遍等于抄第二份，抄了就会漂移。
func printFn(w io.Writer, fn *tzs.SpecFn) {
	fmt.Fprintf(w, "%s — %s\n", fn.Name, fn.Desc)
	marks := []string{}
	if fn.Writes {
		marks = append(marks, "写")
	}
	if fn.Slow {
		marks = append(marks, "慢")
	}
	if fn.NeedsHandle {
		marks = append(marks, "需要句柄")
	}
	fmt.Fprintf(w, "  组 %s   返回 %s   %s\n", fn.Group, fn.Returns, strings.Join(marks, " "))

	fmt.Fprintln(w, "  例：")
	fmt.Fprintf(w, "    %s\n", verbExample(fn))
	if h := filterHint(fn); h != "" {
		fmt.Fprintf(w, "  提示：%s\n", h)
	}

	if len(fn.Args) > 0 {
		fmt.Fprintln(w, "  参数：")
		for _, p := range fn.Args {
			b, _ := json.Marshal(p)
			fmt.Fprintf(w, "    %s\n", b)
		}
	}
	if len(fn.Errors) > 0 {
		fmt.Fprintf(w, "  错误码：%s\n", strings.Join(fn.Errors, " "))
	}
}

// verbExample 造一条**能直接粘**的例子：从 manifest 的参数表生成，不手写（手写必漂移）。
//
// 只列**必填**参数，加三个例外：① `handle` 不列（需要句柄的动词改用 `--form <程序名>` ——
// 那才是调用方想说的东西）；② `file` 列出来，即使它是选填 —— 任务级动词（工作流）正是靠
// `file` 或 `handle` 二选一寻址的，示例里缺了它这条命令就粘不了；③ `path` 同理，它是
// "path 或 paths 二选一"的一半，两个都是选填，只按必填生成会让示例缺了定位，
// 粘上去只会得到一句"需要 path 或 paths"。
func verbExample(spec *tzs.SpecFn) string {
	var parts []string
	for _, p := range spec.Args {
		if p.Name == "handle" {
			continue
		}
		if !p.Required && p.Name != "file" && p.Name != "path" {
			continue
		}
		parts = append(parts, tzs.JSONString(p.Name)+":"+placeholder(p))
	}
	cmd := "tt dev tzs " + spec.Name
	if spec.NeedsHandle {
		cmd += " --form aapp320"
	}
	if len(parts) > 0 {
		cmd += " --args '{" + strings.Join(parts, ",") + "}'"
	}
	return cmd + " --json"
}

// placeholder 给一个参数造一个类型正确的占位值（只用于帮助里的示例）。
//
// 用 `<...>` 而不是编造一个具体值：编造的值会被人当成"就该这么写"，而它可能只是某一
// 张表单的巧合。kind / attr 尤其如此 —— 它们的合法集是运行时从活模型里算的。
func placeholder(p *tzs.Param) string {
	// **角色优先于类型**：占位符由"这个参数是什么意思"决定，而不是由它声明成 `path` 还是
	// `string` 决定 —— `field_add.file` 是 path、`save.out` 是 string，两者都是包路径；
	// 而表单内的 name-path 不论哪种类型都该渲染成 `<name-path>`。判据从前是"参数名是不是
	// `file`"，于是 `open.path`（包路径）被教成 `<name-path>`、`save.out` 干脆教成 `<值>`
	// （2026-09-24 的设计评审发现；现在是引擎声明里的 `role`）。
	switch p.Role {
	case tzs.RolePackagePath:
		return tzs.JSONString("<包路径>")
	case tzs.RoleComponentPath:
		return tzs.JSONString("<name-path>")
	}
	switch p.Type {
	case tzs.TypeInt:
		return "0"
	case tzs.TypeBool:
		return "true"
	case tzs.TypeEnum:
		if len(p.Values) > 0 {
			return tzs.JSONString(p.Values[0])
		}
		return tzs.JSONString("<值>")
	case tzs.TypePathList, tzs.TypeStrList:
		return "[]"
	case tzs.TypeAttrs:
		// 一个具体的属性名，不是 `<...>`：这一半是"一次改多个"的语法示范，空对象只会教人
		// 写出必被拒的调用（引擎对空对象明确报错）。
		return `{"<属性名>":"<值>"}`
	case tzs.TypePath:
		// 剩下的 path 都是表单内的 name-path（包路径那几种在上面按 role 摘走了）。
		return tzs.JSONString("<name-path>")
	case tzs.TypeKind, tzs.TypeKindOrLayout:
		return tzs.JSONString("<kind>")
	case tzs.TypeAttr:
		return tzs.JSONString("<属性名>")
	}
	return tzs.JSONString("<值>")
}

// ---------- 具名动词：把命令行翻成一次 JSON-RPC ----------
//
// 55 个动词（open/save/close/field_add/…）**不是 55 段代码**，而是同一段代码跑 55 次：动词表来自
// 引擎的 manifest，参数定型来自同一份 manifest。所以引擎加/改一个函数，这里一行都不用动
// —— 这正是 internal/dev/tzs/manifest.go 那句「本地绝不抄第二份参数表」的兑现方式。

// builtinVerbs 是 tzs 自己的动词（不来自 manifest）。分派时它们**优先**。
//
// 注意里面**没有** fns / manifest：那两个命令已删除（见上面的「动词索引」）。
// 这份清单的另一处用途是 warnBuiltinCollisions —— 引擎若有同名函数，那个函数就不可达，
// 必须看得见。
//
// `formcheck` 是 M3 加进来的**静态**动词（不开引擎：比 `.42f` 与 tzs 生成物两份文件）。
var builtinVerbs = []string{"export", "formcheck", "doctor", "stop", "reap"}

// verbTransportFlags 是一条动词命令的**传输级**开关（不是动词参数）。
//
// 动词参数只有一种写法：`--args '<JSON 对象>'` / `--args-file <文件>`（见 tzs_engine.go
// 的说明与 internal/dev/tzs/argmap.go 的文件头）。所以这份清单很短，而且是封闭的：
// 任何别的 `--xxx` 都会被当场拒掉，而不是被当成某个动词参数悄悄收下。
var verbTransportFlags = []string{
	"--form", "--args", "--args-file", "--workspace", "--rpc-timeout", "--json", "-h", "--help",
}

// verbFlags 是一条动词命令解析出来的东西。
type verbFlags struct {
	verb        string   // 动词名
	extra       []string // 动词名之后多出来的位置参数（一律报错）
	workspace   string
	timeout     time.Duration
	asJSON      bool
	form        string // --form <程序名|ProgramKey>：按逻辑键寻址（见 BuildArgsForForm）
	args        string // --args 的原文
	argsSet     bool
	argsFile    string
	argsFileSet bool
	help        bool
}

// takeVerbFlags 解析一条动词命令。
//
// 不能用标准库 flag 解析整个 argv：**动词名本身**才是第一个参数，而它得先被取出来
// 才知道该问哪张参数表（那是运行时从引擎 manifest 拿的）。
//
// 这里的严格是有意的：位置参数与未知开关一律报错。动词参数既然只有 JSON 一种写法，
// 那么 `--handle h9` 这种写法就一定是调用方记错了 —— 当场说清楚，比把它当未知参数
// 发给引擎、或者更糟地悄悄忽略掉，都要好。
func takeVerbFlags(argv []string) (verbFlags, error) {
	var f verbFlags
	takeVal := func(i *int) (string, bool) {
		a := argv[*i]
		if eq := strings.IndexByte(a, '='); eq >= 0 {
			return a[eq+1:], true
		}
		if *i+1 < len(argv) {
			*i++
			return argv[*i], true
		}
		return "", false
	}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		switch {
		case a == "-h" || a == "--help":
			f.help = true
		case a == "--json":
			f.asJSON = true
		case a == "--workspace" || strings.HasPrefix(a, "--workspace="):
			v, ok := takeVal(&i)
			if !ok {
				return f, fmt.Errorf("--workspace 需要参数")
			}
			f.workspace = v
		case a == "--rpc-timeout" || strings.HasPrefix(a, "--rpc-timeout="):
			v, ok := takeVal(&i)
			if !ok {
				return f, fmt.Errorf("--rpc-timeout 需要参数")
			}
			n, e := strconv.Atoi(v)
			if e != nil || n <= 0 {
				return f, fmt.Errorf("--rpc-timeout 需要一个正整数秒数，收到 %q", v)
			}
			f.timeout = time.Duration(n) * time.Second
		case a == "--form" || strings.HasPrefix(a, "--form="):
			v, ok := takeVal(&i)
			if !ok {
				return f, fmt.Errorf("--form 需要参数（程序名，如 aapp320；或 ProgramKey，如 aapp320|Form）")
			}
			f.form = v
		case a == "--args" || strings.HasPrefix(a, "--args="):
			v, ok := takeVal(&i)
			if !ok {
				return f, fmt.Errorf("--args 需要参数（一个 JSON 对象）")
			}
			f.args, f.argsSet = v, true
		case a == "--args-file" || strings.HasPrefix(a, "--args-file="):
			v, ok := takeVal(&i)
			if !ok {
				return f, fmt.Errorf("--args-file 需要参数（路径，`-` 表示标准输入）")
			}
			f.argsFile, f.argsFileSet = v, true
		case strings.HasPrefix(a, "--"):
			return f, fmt.Errorf("未知开关 %s：动词参数只用 JSON 给\n  可用开关：%s\n  %s",
				a, strings.Join(verbTransportFlags, " "), verbUsageLine())
		default:
			if f.verb == "" {
				f.verb = a
				continue
			}
			// 动词名之后还有裸词：先留着，分派时可能拿它拼**两级动词名**（`field add`）。
			f.extra = append(f.extra, a)
		}
	}
	return f, nil
}

// verbUsageLine 是动词命令的用法一行（错误文案与帮助共用，避免两处措辞漂移）。
func verbUsageLine() string {
	return "tt dev tzs <动词> --args '{\"<参数>\": <值>}' [--args-file <文件>] [--json]"
}

// splitTwoWordVerb 尝试把「名词 动词」拼成引擎的动词名：`field add` → `field_add`。
//
// 为什么值得支持两级：clig.dev 推荐 `noun verb` 的两级子命令（`docker container create`），
// 因为动词一多，"名词+动词"比平铺一堆动宾混排更好猜。而这里是**纯机械拼接**：
// 不做映射表 —— 有表就有第二份动词名清单，引擎一改就漂移。命中不了就当没这回事。
//
// 返回命中的动词与"用掉了 rest 里的几个词"（0 = 没命中）。
func splitTwoWordVerb(m *tzs.Manifest, first string, rest []string) (*tzs.SpecFn, int) {
	if len(rest) == 0 {
		return nil, 0
	}
	if f := lookupVerb(m, first+"_"+rest[0]); f != nil {
		return f, 1
	}
	return nil, 0
}

// lookupVerb 按名字找动词：精确匹配优先，其次把 `-` 换成 `_` 再找一次。
//
// 允许连字符形式，是因为 shell 里手写 `form-tree` 太自然了，为一次拼写差异浪费一轮往返
// 没有意义。报错文案里给的永远是 manifest 的原名（见调用处）。
func lookupVerb(m *tzs.Manifest, name string) *tzs.SpecFn {
	if f := m.ByName(name); f != nil {
		return f
	}
	if strings.Contains(name, "-") {
		return m.ByName(strings.ReplaceAll(name, "-", "_"))
	}
	return nil
}

// warnBuiltinCollisions 在引擎的函数名与内建动词撞名时提醒一句。
//
// 撞名的后果是内建赢、那个函数从此不可达 —— 这件事必须**看得见**，不能静默。
func warnBuiltinCollisions(m *tzs.Manifest) {
	for _, b := range builtinVerbs {
		if m.ByName(b) != nil {
			fmt.Fprintf(os.Stderr,
				"警告：引擎的函数名 %q 与内建动词同名，内建优先，该函数无法用动词调用\n", b)
		}
	}
}

// readArgsBody 取出 --args / --args-file 里的 JSON 原文（都没给 = 空对象，`)` = 标准输入）。
// 读不动是 IO 失败（退 5），不是用法错：写法没错，是那份文件/标准输入拿不到。
func readArgsBody(f verbFlags) ([]byte, int) {
	var b []byte
	var err error
	switch {
	case f.argsSet:
		b = []byte(f.args)
	case f.argsFile == "-":
		b, err = io.ReadAll(os.Stdin)
	case f.argsFileSet:
		b, err = os.ReadFile(f.argsFile)
	default:
		// 一个参数都没有的动词（list_open 之类）不必写 `--args '{}'`。
		b = []byte("{}")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "读不到 --args-file %s：%v\n", f.argsFile, err)
		return nil, 5
	}
	return b, 0
}

// oldCallExample 给旧写法一条能直接粘的例子（把 `call <函数> --…` 里的函数名提到前面）。
func oldCallExample(rest []string) string {
	if len(rest) == 0 {
		return `open --args '{"path":"D:\\pkg\\a.tzs"}' --json`
	}
	return strings.Join(rest, " ")
}

// runTzsVerb 跑一条具名动词命令：`tt dev tzs <动词> [--<参数> <值>…] [--args <JSON>] [--json]`。
//
// 它是**唯一**的表单读写入口，而且不认识任何具体动词 —— 名字、参数、必填、取值范围
// 全部来自 manifest。想确认"零 per-function 代码"：这个函数里没有一个动词的字面量。
func runTzsVerb(args []string, envName string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, tzsUsage)
		return 2
	}
	f, err := takeVerbFlags(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if f.verb == "" {
		fmt.Fprintln(os.Stderr, "用法："+verbUsageLine())
		fmt.Fprintln(os.Stderr, "  动词名见 `tt dev tzs --help`；某个动词的参数与 JSON 写法：tt dev tzs <动词> --help")
		return 2
	}
	verbName := f.verb

	// `call <函数>` 是上一版的写法。专设一条迁移指引：只说「未知子命令 call」，
	// 会让照旧文档敲命令的人去猜是不是自己拼错了。
	if verbName == "call" {
		fmt.Fprintln(os.Stderr, "`tt dev tzs call <函数>` 已删除：动词就是函数名，参数用 JSON 给")
		fmt.Fprintf(os.Stderr, "  例：tt dev tzs %s\n", oldCallExample(f.extra))
		fmt.Fprintln(os.Stderr, "  动词名见：tt dev tzs --help")
		return 2
	}

	// 先只解析 exe：问函数表不需要工作区，所以"动词名打错"（退 2）与
	// "`<动词> --help`"（退 0）在还没配工作区的新机器上也能给出正确答案。
	exe, err := tzsEngineExe()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 5
	}
	m, err := fetchManifest(exe, manifestTimeout)
	if err != nil {
		return emitFailure(err, f.asJSON)
	}
	warnBuiltinCollisions(m)

	spec := lookupVerb(m, verbName)
	// 两级写法：`field add` 等价于动词 `field_add`（纯拼接，见 splitTwoWordVerb）。
	if spec == nil {
		if two, used := splitTwoWordVerb(m, verbName, f.extra); two != nil {
			spec, verbName = two, two.Name
			f.extra = f.extra[used:]
		}
	}
	if spec == nil {
		// 打错名字是**发现动词**的主要路径（fns 已删），所以这里给全表而不是裁短的清单。
		fmt.Fprintf(os.Stderr, "未知动词 %q\n\n", strings.Join(append([]string{verbName}, f.extra...), " "))
		m.Index(os.Stderr)
		return 2
	}
	if len(f.extra) > 0 {
		// 动词名之后还剩下的裸词：那不是参数（参数一律走 --args）。
		fmt.Fprintf(os.Stderr, "多了位置参数 %s：动词后面只跟开关，参数全部写进 --args 的 JSON 里\n  %s\n",
			strings.Join(f.extra, " "), verbUsageLine())
		return 2
	}
	if f.help {
		printFn(os.Stdout, spec)
		return 0
	}

	// 参数只有一种来源：--args 或 --args-file（二选一；都没有 = 空对象）。
	if f.argsSet && f.argsFileSet {
		fmt.Fprintln(os.Stderr, "--args 与 --args-file 只能给一个")
		return 2
	}
	body, code := readArgsBody(f)
	if code != 0 {
		return code
	}
	// --form 落在引擎的 handle 字段上（两种寻址只能给一个；给不需要句柄的动词传 form 会报错）。
	raw, err := tzs.BuildArgsForForm(m, spec.Name, body, f.form)
	if err != nil {
		return emitFailure(err, f.asJSON)
	}

	// 参数都定型了，现在才需要工作区：**运行**要它，校验不要。
	o, err := tzsEngineOptions(f.workspace, envName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 5
	}
	pipe, err := tzs.Ensure(tzsCtx(), o)
	if err != nil {
		return emitFailure(err, f.asJSON)
	}

	d := tzs.DefaultDialer()
	// 慢调用要有动静：clig.dev 的规矩是「长操作先给一行输出，别让它看起来像卡死」。
	// 阈值 400ms —— 毫秒级的动词保持安静，只有真的慢（open 加载包、validate 跑校验器）
	// 才吱一声。走 stderr，不污染数据流。
	stopProgress := progressAfter(os.Stderr, 400*time.Millisecond, verbProgressText(spec.Name, f.form))
	reply, err := tzs.Call(tzsCtx(), d, pipe, 1, spec.Name, raw, tzs.NormalizeTimeout(f.timeout))
	stopProgress()
	code = tzs.ExitCode(reply, err)
	if code == 0 && reply != nil {
		maybeNudgeFilter(os.Stderr, spec, raw, reply.Result)
	}

	// open 成功后把「最近一次 open」记进状态文件（.tt-tzs.json 的 last）。
	//
	// 它**只是一份指引**，命令层并不读它来寻址：要指定哪张表单用 `--form <程序名>`
	// （引擎按程序名/ProgramKey 解析，见 tzs.BuildArgsForForm）。句柄只活在守护进程里、
	// 永不复用，进程一死全部失效，拿它去填等于把"可能已经没了"当成已知事实。
	if code == 0 && spec.Name == "open" && reply != nil {
		if h := replyString(reply, "handle"); h != "" {
			if p := replyString(reply, "path"); p != "" {
				_ = tzs.Remember(o, h, p)
			}
		}
	}

	if f.asJSON {
		if err != nil {
			// 传输失败：请求**可能已经上线**（所以绝不重试），只是应答没回来。
			// 这一帧是客户端合成的 —— 消费方不必分辨是谁发的，形状是一个。
			return emitFailure(err, true)
		}
		printRawReply(reply)
		return code
	}
	return printHumanReply(spec.Name, reply, err, code)
}

// narrowingParams 是"能把返回收窄"的参数名，按推荐顺序排。
//
// 这份清单**不是**动词表：它只是一组参数名，用来看某个动词有没有省 context 的手段。
// 加新名字的成本是零，而它换来的是"提示自动跟着 manifest 走"——引擎给哪个动词加了
// query，那个动词的提示就自动出现，不需要在 Go 侧登记。
var narrowingParams = []string{"query", "limit", "table", "kind", "path", "column"}

// firstNarrowingParam 返回该动词声明的第一个收窄参数名（没有则空）。
//
// 三个条件**都从 manifest 现成的事实判**，一条都不能少 —— 每一条都是被量出来的
// （2026-09-25 的评测，四个只拿 SKILL 的执行者）：
//
//  1. `!Writes`：会改模型的动词，参数是**输入**不是过滤器。少了这条，`field_add --help`
//     会印一句毫无意义的"先收窄：table…"。
//  2. `!Required`：**必填的东西不可能用来"少回一点"**。少了这条，`open --help` 挂着
//     "先收窄：path 能减少返回" —— 而 open 的 path 必填，不给根本开不了表（评测里被点名）。
//  3. `Returns` 以 `list` 开头：只有会回**一大块**的读动词才有"全量"可收窄。少了这条，
//     `find_component`（返回 list 但 `query` 必填 → 由 2 拦下）与 `get_component`
//     （返回单个 el）都会挂上一句不合身的提示。
func firstNarrowingParam(fn *tzs.SpecFn) string {
	if fn.Writes || !strings.HasPrefix(fn.Returns, "list") {
		return ""
	}
	for _, n := range narrowingParams {
		if p := fn.Param(n); p != nil && !p.Required {
			return n
		}
	}
	return ""
}

// filterHint 给"会回一大块"的动词一句**先过滤**的提示。
//
// 实测（真实语料）：`list_columns --table pmdl_t` 不给 query 回 109 列的完整元数据
// **44,655 字节**，给了 `query:"pmdl00"` 只剩 3,833 字节 —— 一次调用就差 40 KB 的 context。
// 这不是接口缺东西，是**没人告诉调用方要先过滤**。所以提示从 manifest 生成，见
// narrowingParams 的注释。
//
// **会改模型的动词不谈收窄**：它们的参数是输入（`field_add` 的 table 是"加哪张表的列"），
// 不是"少回一点"的过滤器。不加这条判据就会给 field_add 印一句毫无意义的"先收窄：table…"。
func filterHint(fn *tzs.SpecFn) string {
	if fn.Writes {
		return ""
	}
	p := firstNarrowingParam(fn)
	if p == "" {
		return ""
	}
	if p == "query" {
		return "先过滤：--args 里给 query（子串匹配）能显著减少返回；" +
			"实测 list_columns 不给过滤回 109 列 ≈ 44 KB，给了只剩几 KB"
	}
	return "先收窄：" + p + " 能减少返回；不给会回全量（清单类动词可能几十 KB）"
}

// maybeNudgeFilter 在这次"回了一大块、而且本来能收窄"时提醒一句（走 stderr）。
//
// 为什么在**事后**提醒：context 已经花掉了，但提示能把**下一次**调用引向省流量的写法
// （Anthropic 的 tool 指南里专门讲了这条：错误与截断提示可以用来引导调用方）。
// 走 stderr 而不是塞进结果，是为了不污染数据流。
func maybeNudgeFilter(w io.Writer, spec *tzs.SpecFn, args json.RawMessage, result json.RawMessage) {
	if spec.Writes {
		return // 改模型的动词：参数是输入，不是过滤器（同 filterHint 的判据）
	}
	if len(result) < 8192 && resultCount(result) < 20 {
		return // 小返回不提，免得变成噪音
	}
	p := firstNarrowingParam(spec)
	if p == "" {
		return
	}
	if argGiven(args, p) {
		return // 已经收窄过了，不念
	}
	n := resultCount(result)
	if n > 0 {
		fmt.Fprintf(w, "… 提示：本次 %s 回了 %d 条（≈%d KB）；给 --args 里的 %s 收窄能省很多 context\n",
			spec.Name, n, len(result)/1024, p)
		return
	}
	fmt.Fprintf(w, "… 提示：本次 %s 回了 ≈%d KB；给 --args 里的 %s 收窄能省很多 context\n",
		spec.Name, len(result)/1024, p)
}

// resultCount 从返回体里取列表条数（引擎的清单类返回用 count 或 returned）。
//
// 只认这两个键，不做通用遍历：这两个名字是引擎自己在多处用的约定
// （list_tables/list_columns/list_spec_nodes 用 count，查询输出信封用 returned）。
func resultCount(result json.RawMessage) int {
	var probe struct {
		Count    *int `json:"count"`
		Returned *int `json:"returned"`
	}
	if err := json.Unmarshal(result, &probe); err != nil {
		return 0
	}
	if probe.Count != nil {
		return *probe.Count
	}
	if probe.Returned != nil {
		return *probe.Returned
	}
	return 0
}

// argGiven 报告调用方是否给了某个参数（给了空串或 null 不算"收窄过"）。
func argGiven(args json.RawMessage, name string) bool {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(args, &m); err != nil {
		return false
	}
	v, ok := m[name]
	if !ok {
		return false
	}
	s := strings.TrimSpace(string(v))
	return s != "" && s != "null" && s != `""` && s != "[]"
}

// verbProgressText 是慢调用那一行的措辞。
func verbProgressText(verb, form string) string {
	if form != "" {
		return "正在执行 " + verb + "（" + form + "）"
	}
	return "正在执行 " + verb
}

// progressAfter 在 d 之后往 w 打一行"还在跑"，返回一个**必须调用**的收尾函数。
//
// 为什么不是"开始/完成"两条：毫秒级的动词每次都打两行会变成噪音，而要求只是
// "别让它看起来像卡死"。所以只在真的慢时吱一声。
//
// 收尾之后绝不再打印：定时器与 stop 同时就绪时，select 的选择是随机的，
// 所以用一个互斥量把"已经收尾"钉住 —— 否则会给一条已经打完结果的命令补一行"还在跑"。
func progressAfter(w io.Writer, d time.Duration, what string) func() {
	var mu sync.Mutex
	stopped := false
	done := make(chan struct{})
	go func() {
		t := time.NewTimer(d)
		defer t.Stop()
		select {
		case <-t.C:
			mu.Lock()
			defer mu.Unlock()
			if stopped {
				return
			}
			fmt.Fprintf(w, "… %s；还在等引擎，慢是正常的\n", what)
		case <-done:
			return
		}
	}()
	return func() {
		mu.Lock()
		stopped = true
		mu.Unlock()
		close(done)
	}
}

// ---------- stop / reap / doctor ----------

func cmdTzsStop(args []string, envName string) int {
	fs := flag.NewFlagSet("tzs stop", flag.ContinueOnError)
	ws := fs.String("workspace", "", "工作区（默认取 TZSCLI_WS / 配置）")
	// 这里曾声明过一个 `all`（"停掉状态文件里记录的全部守护进程"），但它只是
	// `_ = fs.Bool(...)` —— 解析了、丢掉了，是个**许诺了却不做**的开关，而且没有任何
	// 文档提过它（所以删掉不带走谁的既有用法）。项目自己的标准：advertised-but-inert
	// 比不存在更坏（见 Manifest.cs 里 add_field.name 那条注释）。
	_ = fs.Bool("yes", false, "兼容保留")
	if err := common.ParseArgs(fs, args, "workspace"); err != nil {
		return 2
	}
	o, err := tzsEngineOptions(*ws, envName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 5
	}
	// stop 绝不 spawn：让 stop 去起一个服务器是件可笑的事。
	if err := tzs.Stop(tzsCtx(), o); err != nil {
		return common.Fail(err, false)
	}
	return 0
}

func cmdTzsReap(args []string, envName string) int {
	fs := flag.NewFlagSet("tzs reap", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "确认结束这些进程")
	ws := fs.String("workspace", "", "工作区（可选：reap 按**记录**收，与当前工作区无关）")
	if err := common.ParseArgs(fs, args, "workspace"); err != nil {
		return 2
	}
	// **宽松版**，与 doctor 同一个理由：`reap` 从不 spawn，所以"没配工作区就不许开工"那条
	// 保护对它不成立 —— 它遍历的是状态文件里**每一条记录自己的** workspace（`e.Workspace`），
	// 当前配置里有没有工作区与它要收哪些进程毫无关系。
	//
	// 从前这里走的是严格版（tzsEngineOptions），于是"引擎重编过、旧守护进程收不掉"这件事
	// 发生在**恰恰最需要它的场合**：换机器、换工作区、或者刚把配置改坏了的时候，reap 先
	// 因为"没配工作区"退 5，而它正是来收拾这种残局的。实测 2026-09-24 记在 §11.9。
	o, err := tzsExecOptions(*ws, envName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 5
	}
	pids, err := tzs.Reap(tzsCtx(), o, *yes)
	if err != nil {
		return common.Fail(err, false)
	}
	if !*yes && len(pids) > 0 {
		fmt.Fprintf(os.Stderr, "加 --yes 才会真的结束它们。\n")
	}
	return 0
}

func cmdTzsDoctor(args []string, envName string) int {
	fs := flag.NewFlagSet("tzs doctor", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "输出 JSON")
	if err := common.ParseArgs(fs, args); err != nil {
		return 2
	}
	// 宽松版：缺工作区时 doctor **自己**要在报告里说出来（那是它的职责），
	// 在这里先退 5 就永远走不到那条自检项。
	o, err := tzsExecOptions("", envName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 5
	}
	rep := tzs.Doctor(tzsCtx(), o)
	if *asJSON {
		b, _ := json.Marshal(rep)
		os.Stdout.Write(append(b, '\n'))
	} else {
		// 命令层渲染：Report 只带结构化条目，措辞留给这里。
		for _, c := range rep.Checks {
			fmt.Printf("  [%-4s] %-22s %s\n", c.Level, c.Name, c.Message)
		}
		if !rep.OK() {
			fmt.Fprintf(os.Stderr, "\n自检不通过：%s\n", strings.Join(rep.Failed(), ", "))
		}
	}
	if !rep.OK() {
		return 5
	}
	return 0
}

// ---------- 小工具 ----------

// tzsCtx 是一次命令的 context。引擎那侧的超时由 tzs.Call 自己管（命名管道不支持 deadline），
// 所以这里只用来传递取消。
func tzsCtx() context.Context { return context.Background() }

// 函数表拉取的时限。两处调用给的值不同，但**都必须有上限**。
//
// 为什么必须（2026-09-25 实测，一次真事故）：`tzsCtx()` 是 `context.Background()` —— 没有期限，
// 而 `FetchManifest` 会去 spawn `tzs-server.exe --manifest`。引擎 exe 存在但卡住时（半死、被
// 安全软件挂着），这条调用会**永远**挂着，而它是**每一条** `tt dev tzs <动词>` 的第一件事，
// 包括 `<动词> --help` —— 于是"还没有环境时唯一能用的一条命令"变成了唯一一条会挂死的命令。
// 那次是 `--help` 的测试挂满 10 分钟被 go test 判超时，goroutine dump 停在写动词索引上。
// （那是**第二个** bug，见 pos_test.go 的 captureStdout；两个叠在一起才让它显形。）
//
// 上限该给多少，看函数的体量：spawn 一个 8 KB 的 exe、打印 41 KB JSON，本机实测 ~50 ms。
const (
	// manifestTimeout 是"派发前要校验参数"的那条路：拿不到就退 5，所以给足冷启动、杀软扫描、
	// 机器正忙的时间 —— 但绝不是"永远"。
	manifestTimeout = 60 * time.Second
	// verbIndexTimeout 是 `--help` 末尾那段动词索引：它是**装饰**，拿不到就少一段、退出码不变，
	// 所以短得多 —— 谁都不该为了看帮助等一分钟。
	verbIndexTimeout = 10 * time.Second
)

// fetchManifest 拉函数表，**带一个期限**（理由见上）。
func fetchManifest(exe string, d time.Duration) (*tzs.Manifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	defer cancel()
	return tzs.FetchManifest(ctx, exe)
}

func replyString(r *tzs.Reply, key string) string {
	var m map[string]any
	if json.Unmarshal(r.Result, &m) != nil {
		return ""
	}
	s, _ := m[key].(string)
	return s
}

// printRawReply 把一帧原样写到 stdout。
//
// **它只收真帧**：从前这里还收一个 err，传输失败时 `json.Marshal(nil)` 会往 stdout
// 打一个字面量 `null` —— 看着像 JSON、其实什么都没说。那种失败现在走 emitFailure。
func printRawReply(r *tzs.Reply) {
	b, _ := json.Marshal(r)
	os.Stdout.Write(append(b, '\n'))
}

// emitFailure 是**没能成为帧的失败**的统一出口，并返回该给的退出码。
//
// 定下的规则：`tt dev tzs <动词> --json` 在"这次调用有了结论"时，stdout **恰好一帧** ——
// 引擎发的和客户端合成的**是同一个形状**（`tzs.Reply`，同一个 json.Marshal），
// 所以消费方不需要分辨这一帧是谁发的。
//
// 三处**刻意**的例外（不写出来这条规则就是假的）：
//
//  1. 用法屏 —— 没有动词、动词名打错（打的是动词索引）、`<动词> --help`。那些是**说明书**，
//     形态由说明书决定，不是调用结论。
//  2. 调用还没成立时的本地参数错（`--args` 与 `--args-file` 同时给、多了位置参数…）——
//     进 stderr。**这条改了口径**：从前 `localUsage` 对"本地参数错"一律不写 stdout，
//     理由是"stdout 上的每一行都该是引擎的帧"；那条理由在客户端开始合成帧之后不成立了，
//     现在它与别的失败同路。
//  3. 引擎 exe / 工作区这类**环境**失败（`readArgsBody`、`tzsEngineOptions` 那几处）——
//     同样进 stderr，它们连"要跑哪个动词"都还没算出来。
//
// 非 --json 时文案与从前一致（人话进 stderr）。
func emitFailure(err error, asJSON bool) int {
	code := common.ExitCodeOf(err)
	if asJSON {
		// **紧凑、一行**，与引擎帧一样。从前这里走 emitJSON（它会美化缩进），于是同一个
		// `--json` 出口有两种形状：引擎发的帧是一行，客户端合成的帧是多行 —— 按行读的消费方
		// （`| head -1`、逐行解析）在合成的帧上会踩空，而它分不出两者有什么不同。
		// （2026-09-25 实测发现：契约说"一行一个响应"，合成帧也该守。）
		b, merr := json.Marshal(tzs.SyntheticFailure(err))
		if merr != nil {
			fmt.Fprintf(os.Stderr, "合成帧失败：%v\n", merr)
			return code
		}
		os.Stdout.Write(append(b, '\n'))
		return code
	}
	fmt.Fprintf(os.Stderr, "错误（退出码 %d）：%v\n", code, err)
	return code
}

// printHumanReply 把一帧渲染成人看的样子，并返回该给的退出码。
func printHumanReply(fn string, r *tzs.Reply, err error, code int) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return code
	}
	if r.OK {
		b, _ := json.Marshal(r.Result)
		fmt.Printf("%s: %s\n", fn, b)
		return code
	}
	if r.Error != nil {
		fmt.Fprintf(os.Stderr, "%s (%s): %s\n", r.Error.Code, r.Error.Kind, r.Error.Message)
		if r.Error.IsSuccessCode() {
			// E_NO_OP / E_ATTR_CLAMPED 在语义上是"什么也没改"（见 client.go 的 CodeNoOp 注释）。
			//
			// **兜底，不是主路**（2026-09-25 起）：引擎里已经没有任何路径把成功码发成错误帧了
			// —— 最后一处（`set_local_string` / `set_spec_description` 的 NoOp 分支）已按
			// SPEC §11.24(a) 改成成功帧。主路是上面那个 `r.OK` 分支：`ok:true` +
			// `result.code`，原样打 JSON，`code`/`noop`/`clamped` 都在里面，看得见，
			// 不需要再加一层。
			//
			// 留着这段是因为它可以由**旧引擎**触发（守护进程按 MVID 命名，重编前后各有一批），
			// 而那正是最需要说人话的时刻：**退出码不动**（仍按契约表 kind=internal → 1），
			// 只把文案从"内部错误、上报"改成"什么也没改"。
			common.Line(os.Stderr, "    这不是错误：请求的值与当前值相同，设计器同值短路，什么都没改。")
			common.Line(os.Stderr, "    （退出码 1 来自契约表把它的 kind 归成 internal；见 SPEC §11.24(a)。）")
		}
		// 可操作的那一半（legal / hint / candidates / applied / failed）默认也要看得见 ——
		// 引擎做它们就是为了让调用方自己纠正，只让 --json 看得见等于没做。
		printWireDetail(os.Stderr, r.Error)
	}
	return code
}
