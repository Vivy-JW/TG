using System;
using System.Collections.Generic;
using Newtonsoft.Json.Linq;

namespace TzsCli.Designer
{
    /// <summary>
    /// The function table -- the single source of truth the CLI help, the argument validation and
    /// the (later) MCP tool list are all generated from. SPEC §11.24 (c) is the frozen shape of
    /// one entry; §11.22 is the list of the ~50 functions that will eventually exist.
    ///
    /// Every function in the eventual surface is declared here, including the ones whose bodies
    /// are still being written in parallel (src/Designer/Fns/*.cs). A declaration without a body
    /// is not a lie: dispatch answers E_NOT_IMPLEMENTED for it, which is a different answer from
    /// E_UNKNOWN_METHOD, and the difference is exactly "this will work later" versus "you typed
    /// it wrong". Declaring the whole surface up front is also what lets tzs-cli --help and the
    /// CLI's own argument typing work before any Fn exists.
    ///
    /// The *types* (SpecFn / Param / PType / Fn / P / SpecSlots) are frozen in SpecFn.cs. This
    /// file holds only the table, plus the validation that reads it.
    ///
    /// `slow:true` is on `validate` alone, and it is not decoration: SPEC §11.24 (c) measured
    /// 1.6 s on a 114-element form and 10.4 s on a 670-element one. It is surfaced in --help so a
    /// caller does not put it in a loop.
    /// </summary>
    public static class Manifest
    {
        // ---------------------------------------------------------------- groups
        const string G_WORKFLOW = "工作流";
        const string G_SESSION = "会话";
        const string G_READ    = "读";
        const string G_ATTR    = "属性";
        const string G_STRUCT  = "结构";
        const string G_PAGE    = "页签";
        const string G_SEM     = "语义/Action";
        const string G_MULTI   = "多语言/选项/串查";
        const string G_TAB     = "Tab 顺序";
        const string G_TOOL    = "校验/工具";

        // ---------------------------------------------------------------- error vocabulary
        /// <summary>The codes a declared function may put on the wire. Kept as arrays so the
        /// help and the (future) MCP tool schema cannot drift from what the Fns throw.</summary>
        static readonly string[] E_STD = { "E_NOT_FOUND", "E_BAD_PARAM", "E_DESIGNER", "E_INTERNAL" };

        /// <summary>`base_` + `extra`, minus duplicates. Exists so a family can be stated as "the
        /// common set plus what makes this one different" instead of as a fresh literal: the fresh
        /// literal is how E_STD's four codes got copy-pasted into five arrays, and how a code added
        /// to the common set would have been remembered in none of them.</summary>
        static string[] Plus(string[] base_, params string[] extra) {
            var codes = new List<string>(base_);
            foreach (string e in extra) if (!codes.Contains(e)) codes.Add(e);
            return codes.ToArray();
        }

        /// <summary>E_STD plus the no-op outcome code, for the writers that can answer "the value you
        /// asked for is already there".
        ///
        /// E_NO_OP belongs in the advertised set even though it arrives in `result` and not in
        /// `error` (SPEC §11.24 (a)): `noop:true` + `code:"E_NO_OP"` is one of the three outcomes a
        /// caller has to tell apart from `applied`, and a caller writing its branches from `--help`
        /// cannot branch on a code the help never mentions. Measured 2026-09-25: twelve verbs can
        /// answer it, and only the four attribute writers (E_SPEC / E_LAYOUT) advertised it -- the
        /// other eight were reachable through no declared vocabulary at all.</summary>
        static readonly string[] E_NOOP = Plus(E_STD, "E_NO_OP");
        // `open` is the only verb that loads a package, so it is the only one the engine's own
        // watchdog can kill mid-call (E_FATAL_LOAD_TIMEOUT, written by the watchdog thread just
        // before exit(3)). Kept as its own array rather than added to a shared one because no other
        // verb can produce it -- and the E_SESS array that used to hold E_KEY_IN_USE for `open` is
        // gone with it (open was its only user).
        static readonly string[] E_OPEN = {
            "E_NOT_FOUND", "E_BAD_PARAM", "E_DESIGNER", "E_KEY_IN_USE",
            "E_FATAL_LOAD_TIMEOUT", "E_INTERNAL" };

        // The two attribute writers answer with more than E_STD says, and the difference is the
        // point of both: a caller that cannot see E_ATTR_NOT_WHITELIST in the help will not know
        // that a refused name comes back with detail.legal, and one that cannot see
        // E_ATTR_VALUE_ILLEGAL will not know a *value* is checked at all -- which is the failure
        // this pair exists to end (case="GARBAGE" used to come back ok:true).
        //
        // E_ATTR_CLAMPED and E_NO_OP are listed even though they arrive as SUCCESSES in `result`
        // (SPEC §11.24 (a)), not as errors: they are the two outcomes a caller must be able to tell
        // apart from `applied`, so leaving them out of the advertised set would hide exactly the
        // vocabulary the distinction needs.
        static readonly string[] E_SPEC = Plus(E_NOOP,
            "E_ATTR_NOT_WHITELIST", "E_ATTR_VALUE_ILLEGAL", "E_ATTR_PARTIAL");
        // E_NO_SPEC_NODE is deliberately not listed here: AdvertisedErrors derives it for every
        // Mutating verb that takes a spec-node `kind`, which is exactly the set of verbs that
        // resolves one. It was hand-listed here once and the verb in the next array over was
        // forgotten (Fns/Semantic.cs's set_spec_description, 2026-09-25).
        static readonly string[] E_LAYOUT = Plus(E_NOOP,
            "E_ATTR_NOT_WHITELIST", "E_ATTR_VALUE_ILLEGAL", "E_ATTR_PARTIAL", "E_ATTR_CLAMPED");

        /// <summary>The codes one function can put on the wire: its declared array, plus the ones its
        /// own parameters imply.
        ///
        /// WHY DERIVED. The arrays above say what each *family* of verbs was expected to throw, and
        /// they had forgotten the code every handle-taking verb really can throw and the one every
        /// name-path-taking verb can. Measured 2026-09-25 (§11.9 item 6): eight codes appeared in no
        /// verb's list at all -- E_NO_HANDLE, E_HANDLE_BUSY, E_PATH_NOT_FOUND, E_NO_SPEC_NODE,
        /// E_BAD_REQUEST, E_UNKNOWN_METHOD, E_SERVER_DIED, E_FATAL_LOAD_TIMEOUT -- so a caller
        /// writing its branches from `--help` was missing them. Whether a verb takes a handle or a
        /// path is already in its declaration, so the advertisement is computed rather than
        /// remembered. Of those eight: E_HANDLE_BUSY turned out to be unreachable and is gone from
        /// Rpc.Map; E_FATAL_LOAD_TIMEOUT is declared on `open`, the only verb that loads;
        /// E_NO_SPEC_NODE is derived for every Mutating verb that takes a spec-node `kind`, and
        /// hand-declared on the one verb that resolves a node without taking a kind (set_cited).
        ///
        /// NOT advertised, deliberately: E_BAD_REQUEST and E_UNKNOWN_METHOD (refused before a verb
        /// is dispatched) and E_SERVER_DIED (synthesised by the CLI). Those are facts about the
        /// call, not about the verb's body.</summary>
        static JArray AdvertisedErrors(SpecFn f) {
            var codes = new List<string>(f.Errors);
            bool handle = false, path = false, kind = false;
            foreach (Param p in f.Params) {
                if (p.Type == PType.Handle) handle = true;
                // A name-path, not a package path: only the former can come back E_PATH_NOT_FOUND
                // (`open.path` is a .tzs file, and a missing one is a load failure, not this).
                if (p.Role == Role.ComponentPath) path = true;
                if (p.Type == PType.Kind) kind = true;
            }
            if (handle && !codes.Contains("E_NO_HANDLE")) codes.Add("E_NO_HANDLE");
            if (path && !codes.Contains("E_PATH_NOT_FOUND")) codes.Add("E_PATH_NOT_FOUND");
            // A Mutating verb that takes a spec-node `kind` resolves one, so "this element has no
            // such node -- ask for another kind" is one of its answers. `Mutating` is the load-bearing
            // half: the read verbs take the same `kind` to filter a list, and an empty list is not
            // this code.
            if (kind && f.Mutating && !codes.Contains("E_NO_SPEC_NODE")) codes.Add("E_NO_SPEC_NODE");
            return new JArray(codes);
        }

        /// <summary>The 控件箱 (WidgetBox.xaml, 31 items) minus the container/semantic families,
        /// which get their own enums. It is an enum rather than a free string because
        /// ComponentFactory.ModFdInfo.IsIncludeAttribute NREs natively on a type outside the
        /// catalogue (SPEC §11.15) -- refusing it in validation turns a stack trace into a
        /// message that names the parameter.</summary>
        static readonly string[] WIDGETS = {
            "Button", "ButtonEdit", "Canvas", "CheckBox", "ComboBox", "DateEdit", "DateTimeEdit",
            "Edit", "FFImage", "FFLabel", "HLine", "Image", "Label", "ProgressBar", "RadioGroup",
            "Slider", "SpinEdit", "TextEdit", "TimeEdit", "WebComponent"
        };

        // The container vocabulary is NOT declared here any more. It used to be (a ten-entry array
        // that every `container`/`type` parameter published), and it was wrong for every verb that
        // used it: the creation verbs accept only Struct's six, `convert_container` accepts two.
        // Each parameter now publishes the array its own body validates against, so the declared set
        // and the enforced set are the same object -- see Fns/Struct.cs for the vocabulary itself
        // (including which container names exist only as commands). The type is `Struct` and it lives
        // in this namespace, not in `.Fns`: its own header explains why.

        // ---------------------------------------------------------------- table builders
        /// <summary>A handle-taking function. NeedsHandle=true is the default in SpecFn, and it
        /// is also what makes the transport resolve args.handle before the body runs.
        ///
        /// `mutating` does a second job here as well as marking the verb [写] in the help: it is
        /// what puts `dry_run` and `op` on the wire (see <see cref="Traced"/>). That is deliberate
        /// -- "this verb can change the model" is exactly the set of verbs where "try it without
        /// doing it" and "name this attempt" mean anything, and deriving the two from the flag is
        /// what keeps 36 declarations from drifting out of it. The one verb that leaves a trace
        /// WITHOUT being mutating (`save`, it writes a file and dirties nothing) declares them
        /// itself, right where its other exception is explained.</summary>
        static SpecFn F(string name, string group, string summary, bool mutating, string returns,
                        string[] errors, params Param[] ps) {
            var f = new SpecFn {
                Name = name, Group = group, Summary = summary, Params = ps,
                NeedsHandle = true, Mutating = mutating, Slow = false, Returns = returns, Errors = errors
            };
            if (mutating) { f.Traced = true; f.Params = TracedParams(f.Params); }
            return f;
        }

        /// <summary>Marks a verb that is NOT mutating as traced anyway, and declares its two
        /// switches at the call site. One verb needs this -- `save` -- and it is the one whose
        /// written bytes are the whole point, so it is the verb `op` is least optional for.</summary>
        static SpecFn Traced(SpecFn f) { f.Traced = true; return f; }

        /// <summary>The two switches every write verb answers: `dry_run` and `op`.
        ///
        /// Appended AFTER the verb's own parameters, which is where a cross-cutting switch belongs
        /// on the wire: the manifest's parameter order IS the frame's key order (internal/dev/tzs's
        /// argmap), so the verb's own arguments stay in the position every reader already expects
        /// and the two flags sit at the end.
        ///
        /// Both are optional and neither changes the verb's own semantics, which is why they can be
        /// derived instead of declared: nothing about a specific verb decides whether they apply.
        /// They are, however, *published* like any other parameter -- --help, --manifest and the Go
        /// client all read them off this array, and a capability that works but is advertised
        /// nowhere is the defect §11.9 item 10 was about (list_local_strings' unreachable filter).</summary>
        static Param[] TracedParams(Param[] own) {
            var ps = new List<Param>(own);
            ps.Add(new Param {
                Name = "dry_run", Type = PType.Bool, Required = false,
                Desc = "只算不做：真跑一遍这个动词，答案原样放进 preview，然后把模型回滚（盘上不写）",
            });
            ps.Add(new Param {
                Name = "op", Type = PType.Str, Required = false,
                Desc = "这次写的名字（你自己起）；超时后用 list_ops 问它，就知道到底写进去没有",
            });
            return ps.ToArray();
        }

        /// <summary>Same, but with no session: these answer from the workspace or from the
        /// catalogue, so an open package is neither required nor touched.</summary>
        static SpecFn N(string name, string group, string summary, string returns,
                        string[] errors, params Param[] ps) {
            SpecFn f = F(name, group, summary, false, returns, errors, ps);
            f.NeedsHandle = false;
            return f;
        }

        static Param Opt(PType t, string n, string desc) {
            return new Param { Name = n, Type = t, Required = false, Desc = desc };
        }

        /// <summary>The `path` + `paths` pair, as the six verbs that go through `Struct.PathsArg`
        /// declare it: either form accepted, **one of them required**, and that "one of them" is
        /// checked in the body because a manifest `Required` flag is per-parameter and cannot express
        /// 二选一 (the same reasoning as add_field's column/columns, §11.9 item 8).
        ///
        /// Both are declared for all six. Before 2026-09-25 five of them declared only `paths`, and
        /// required -- so `PathsArg`'s own message ("需要 path 或 paths") named a form the CLI refused
        /// before the request was sent, and `delete` (the one that did declare `path`) forced a dummy
        /// `path` on anyone who only wanted the batch form. Same class as item 10's
        /// `list_local_strings` filter: a capability the body has and the wire cannot deliver.</summary>
        static Param AnyPath() {
            return new Param { Name = "path", Type = PType.Path, Required = false,
                Desc = "name-path（与 paths 至少给一个）", Role = Role.ComponentPath };
        }

        static Param AnyPaths() {
            return new Param { Name = "paths", Type = PType.PathList, Required = false,
                Desc = "批量的 name-path（与 path 至少给一个；给了它就忽略 path）",
                Role = Role.ComponentPath };
        }

        /// <summary>A **task-level** verb: it resolves its own session (handle / program name /
        /// ProgramKey / file), so the transport must not demand a handle before the body can even
        /// see `file`. Same shape as F otherwise -- `mutating` still has to be told the truth,
        /// because that is what the CLI shows as [写].</summary>
        static SpecFn W(string name, string group, string summary, bool mutating, string returns,
                        string[] errors, params Param[] ps) {
            SpecFn f = F(name, group, summary, mutating, returns, errors, ps);
            f.NeedsHandle = false;
            return f;
        }

        /// <summary>An optional enum still has to carry its Values: an Enum Param with a null
        /// Values array would accept any string, which is the silent-typo hole this table
        /// exists to close.</summary>
        static Param OptEnum(string n, string[] values, string desc) {
            return new Param { Name = n, Type = PType.Enum, Required = false, Values = values, Desc = desc };
        }

        /// <summary>Every declared function, in §11.22's order plus the 工作流 group in front.
        /// The order is the order --help prints them in, which is why it is the spec's order and
        /// not alphabetical.</summary>
        public static readonly SpecFn[] All = {

            // ------------------------------------------------------------ 工作流 (1)
            // Task-level verbs: one request does what used to be a chain of calls. They resolve
            // their OWN session (handle / program / ProgramKey / file), which is why NeedsHandle
            // is false -- the dispatcher must not demand a handle before the body can look at
            // `file`. Declared FIRST on purpose: help and the CLI index print in manifest order,
            // and the recommended entry point should be the first thing a caller sees
            // (clig.dev: "display the most common commands at the start of the help text").
            W("field_add", G_WORKFLOW,
              "按数据表的列一次性加字段（挑容器 → 建字段 → 报校验增量 → 可存新包，一次请求做完）",
              true, "report", E_STD,
                Opt(PType.Handle, "handle", "已在开的表单：句柄 h1 或程序名 aapp320 或 ProgramKey aapp320|Form"),
                P.As(Opt(PType.Path, "file", ".tzs 路径：没开就顺手开，已开着就复用（与 handle 二选一）"), Role.PackagePath),
                P.Str("table", true, "表名，如 pmdl_t"),
                P.StrList("columns", true, "列名数组；一次构造 N 列（列名可用 list_columns 取）"),
                Opt(PType.Path, "into", "父容器的 name-path；省略时自动挑（优先 worksheet）"),
                OptEnum("container", Struct.CONTAINER_TYPES, "容器模式，默认 None"),
                P.As(Opt(PType.Path, "out", "给了就把结果存成这个**新**包（绝不写源包）"), Role.PackagePath)),

            // ------------------------------------------------------------ 会话 (7)
            N("open", G_SESSION, "加载包 + 注册会话，返回句柄", "handle", E_OPEN,
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = ".tzs 路径", Role = Role.PackagePath },
                // `timeout` was read by Session.Open (Fns/Session.cs:220) but declared nowhere, and
                // Manifest.Check refuses an argument that is not in Params -- so the parameter was
                // unreachable from the wire. It lived only in Fns/Session.cs's `Descriptors`, a
                // second copy of this table that nothing read; that copy is now deleted and the
                // declaration moved here, where callers are actually validated against.
                new Param { Name = "timeout", Type = PType.Int, Required = false,
                            Desc = "加载看门狗秒数；默认 $TZSCLI_RELOAD_TIMEOUT 或 90" }),
                // `force` used to be declared here ("占用者只是 Loaded 时强制接管") and was never
                // implemented: Fns/Session.cs refuses it with E_NOT_IMPLEMENTED, because taking over
                // an occupant IS the silent eviction the contract forbids. An advertised parameter
                // that always throws is worse than an absent one -- the same reason `add_field.name`
                // was deleted -- so it is gone from the wire. The guard stays in Session.cs as a
                // backstop (if the declaration ever comes back, the refusal still holds), and
                // SPEC.md §11.24 (g) says what the caller must do instead: close, then open.
                // A clean executor in the 2026-09-24 baseline read `open --help`, saw `force` listed
                // and §6 saying it is unimplemented, and reported the contradiction.
            Traced(F("save", G_SESSION, "把句柄的模型写回新包（不改会话状态）；回 bytesIn/bytesOut 与写入字节的 sha256",
                false, "void", E_STD,
                P.Handle(),
                P.As(P.Str("out", true, "输出 .tzs 路径"), Role.PackagePath),
                // The two switches are declared here BY HAND because `save` is not Mutating and yet
                // is the verb they matter most for: it is the only one whose whole point is a file
                // on disk, so "did it land" (op) and "what would it write" (dry_run) are questions
                // about it, not about the model. Everything else gets them from F()'s Traced().
                new Param { Name = "dry_run", Type = PType.Bool, Required = false,
                    Desc = "只算不写：照样拼包并回 sha256，但不落盘（out 上不会出现文件）" },
                new Param { Name = "op", Type = PType.Str, Required = false,
                    Desc = "这次写的名字（你自己起）；超时后用 list_ops 问它，就知道到底写进去没有" })),
            // Mutating=false, and the argument for it was sitting in Fns/Session.cs's dead
            // `Descriptors` copy of this table, which said `true` -- the two disagreed and nothing
            // read either. Resolved in favour of the argument: `close` releases a handle, it does
            // not dirty a model, so it is not a write. Same reasoning as `save` below, which is
            // false for the mirror-image reason (it writes a file but leaves the model Loaded,
            // and that is exactly the fixed-point property the RoundTrip oracle tests).
            F("close", G_SESSION, "释放句柄（发布 TzpFileClose + 摘 EAM；句柄串永不复用）", false, "void", E_STD,
                P.Handle()),
            // `reload` is the "close + open" pair as one command, and it lives HERE rather than as
            // advice in the help because the pair has a cost a caller cannot see: close drops the
            // validate baseline (Fns/Session.cs), so the reload-then-measure-the-delta workflow
            // loses its reference point. reload keeps it when the file is byte-identical, and says
            // which of the three cases it hit.
            //
            // Not Mutating, and not merely by analogy: it takes a dirty model and makes it clean,
            // so it is the opposite of a write. The transport therefore does not offer dry_run on
            // it (a dry run of "throw away my changes" has no meaning).
            F("reload", G_SESSION, "丢弃内存改动、从盘重读同一个包（句柄不变；文件没变就保留 validate 基线）",
                false, "handle", E_STD,
                P.Handle(),
                new Param { Name = "timeout", Type = PType.Int, Required = false,
                            Desc = "重读的看门狗秒数；默认 $TZSCLI_RELOAD_TIMEOUT 或 90" }),
            // No `path` here. It was declared ("只校验这条子树") on validate and verify and implemented
            // as a refusal (Validate.RejectPath): the designer's validators take one whole XElement
            // and have no subtree entry point, so the parameter could never do anything but come back
            // E_NOT_IMPLEMENTED -- whose kind is `internal`, i.e. "report a bug, do not retry", for a
            // parameter the help was offering. Same call as `force` (item 4): advertised-but-inert is
            // worse than absent. The refusal stays in the body as the backstop for library callers.
            F("verify", G_SESSION, "对句柄的模型跑设计器自己的校验器，报 delta", false, "delta", E_STD,
                P.Handle()),
            N("list_open", G_SESSION, "列出打开的句柄；回吐 ProgramKey 让碰撞可见", "list<el>", E_STD),
            // The other end of the same conversation: list_open says what is open NOW, this says
            // what has just HAPPENED. It exists for one moment -- after a timeout, when the caller
            // must decide whether retrying is safe -- and the answer is three-valued on purpose:
            // an entry (ok/error) means retry is safe, a `pending` entry means do not, and NO entry
            // means the request never arrived, which is also safe to retry. That trichotomy is the
            // whole feature (幂等 op log，见 internal/dev/tzs/README.md); the log is per-process, and the
            // answer says so with daemonStartedAt.
            N("list_ops", G_SESSION, "写请求的操作日志：超时后问「那次写到底进去没有」", "list<el>", E_STD,
                Opt(PType.Str, "op", "只列这个名字的写请求（精确匹配；不给就是最近的全部）"),
                Opt(PType.Int, "limit", "最多回多少条，默认 20（新的在前）")),

            // ------------------------------------------------------------ 读 (9)
            F("form_tree", G_READ, "表单结构树（画面结构页签），每节点带 name-path", false, "tree", E_STD,
                P.Handle(),
                Opt(PType.Path, "path", "从这条子树开始，默认整张表单"),
                Opt(PType.Int, "depth", "限制层数")),
            F("find_component", G_READ, "把控件代号解析成 name-path（精确优先，未中退化子串）", false, "list<el>", E_STD,
                P.Handle(),
                P.Str("query", true, "控件代号或其前缀")),
            F("get_component", G_READ, "一个元素的布局属性 + 它持有的规格节点属性", false, "el", E_STD,
                P.Handle(),
                // `path` is deliberately NOT required: the body takes `path` **or** `query` -- its own
                // message says "get_component 需要 path 或 query 之一", and `query` resolves a 控件代号
                // through the same resolver find_component uses. Declaring `path` required made the query
                // branch unreachable twice over (the CLI refuses a call with no `path` before the request
                // is ever sent), so the only way to use it was the two-call find_component -> get_component
                // dance. Same defect as set_spec_description's `kind` (SPEC §11.9 item 11), found by the
                // sweep that closed item 10.
                new Param { Name = "path", Type = PType.Path, Required = false, Desc = "name-path", Role = Role.ComponentPath },
                Opt(PType.Str, "query", "控件代号（与 path 二选一）"),
                Opt(PType.Kind, "kind", "限定只回哪一类规格节点")),
            F("list_spec_nodes", G_READ, "整张表单的规格节点清单（按 kind 过滤）", false, "list<el>", E_STD,
                P.Handle(),
                // The body read `name` and the manifest declared nothing, so "narrow the list before you
                // dump it" (SKILL §4.7) was unactionable here -- and this is the verb whose answer is
                // biggest: aapt300 has 269 field nodes alone. Same pair as list_tables/list_columns:
                // `query` on the wire, the old name kept as an alias for library callers.
                Opt(PType.Str, "query", "节点名子串"),
                Opt(PType.Kind, "kind", "只列这一类")),
            // Needs a handle, unlike its neighbours here. The attribute set is per-FILE, not
            // per-kind: an aapp320 field carries 23 attributes, a cs_excel one carries 20,
            // because the answer is read off the nodes a load actually produced. Answering
            // without a package would mean answering from a fabricated canonical node and
            // disagreeing with what set_spec_attr will accept. `kind` is optional so one call
            // can return all seven.
            F("describe_kind", G_READ, "某类规格节点运行时可写的属性白名单（§11.24(c) from:describe_kind）",
              false, "kindmap", E_STD,
                P.Handle(),
                P.KindOrLayout("kind", "七种之一，或 layout（布局属性：名字 + 类型 + 取值集）；省略则返回全部七种")),
            // 工作区里有哪些**表单包**。加它的理由（2026-09-25 的评测 F23，三个执行者都撞上）：
            // "哪个文件是 aapt300" 从前只有一个答案 —— 自己 `ls` 工作区，而真实工作区里那是
            // 29.9 KB 噪音（`.tzc`/`.bak` 混着）。它与 list_tables 同族：不需要句柄，答案来自文件系统。
            N("list_packages", G_READ, "工作区里的表单包（.tzs/.tzv）", "list<el>", E_STD,
                Opt(PType.Str, "query", "文件名子串（程序名就是文件名前缀，按程序名筛就行）"),
                Opt(PType.Int, "limit", "最多回多少条，默认 200")),
            N("list_tables", G_READ, "工作区数据字典里的表", "list<el>", E_STD,
                Opt(PType.Str, "query", "名称子串"),
                // `limit` was read by the implementation (IntArg(a,"limit",200)) but declared
                // nowhere -- and Manifest.Check refuses an argument that is not in Params, so the
                // cap could be neither raised (get table 201) nor lowered. Same class as `timeout`
                // on `open`, above.
                Opt(PType.Int, "limit", "最多返回多少张表；默认 200")),
            N("list_columns", G_READ, "一张表的列（含 widget/attribute/type/req 等列元数据）", "list<el>", E_STD,
                P.Str("table", true, "表名，如 pmdl_t"),
                Opt(PType.Str, "query", "列名子串")),
            F("list_records", G_READ, ".4fd 的 Record / RecordField 清单", false, "list<el>", E_STD,
                P.Handle()),
            F("list_local_strings", G_READ, "本地化串（sfield）清单", false, "list<el>", E_STD,
                P.Handle(),
                // `query` is what the body reads first (Read.cs), with `filter` kept as an alias for
                // direct library callers -- the same pair as list_tables / list_columns. It used to be
                // read as `filter` ONLY and declared nowhere, so "narrow the list before you dump it"
                // -- the rule SKILL §4.7 states for every list verb -- was unreachable from the wire on
                // the one verb whose answer is biggest on a large form (aapt300: ten strings for the
                // worksheet subtree alone, each with its text).
                Opt(PType.Str, "query", "串名子串"),
                Opt(PType.Path, "path", "限定某条子树")),

            // ------------------------------------------------------------ 属性 (6)
            F("set_spec_attr", G_ATTR, "改字段规格属性（白名单来自 describe_kind）", true, "delta", E_SPEC,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Kind(),
                P.Attr("attr", "spec:<kind>"),
                P.Str("value", true, "新值")),
            F("set_spec_attrs", G_ATTR, "一次改一个节点的多个规格属性（先全量校验，再全量写）", true, "delta", E_SPEC,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Kind(),
                P.Attrs("attrs", "{\"属性名\":\"值\", …}；一次请求改多个，合成一步撤销")),
            F("set_layout_attr", G_ATTR, "改布局属性（走 XmlElement 索引器，铁律 §11.24(d)）", true, "delta", E_LAYOUT,
                P.Handle(),
                // NOT required, and that is the fix rather than the bug: this verb takes path OR
                // paths, and the descriptor cannot express an either/or. Declaring path required
                // made the batch form unreachable -- the caller's own client refused
                // `{"paths":[...],"attr":...,"value":...}` with "缺必填参数 path" before the request
                // ever reached the engine, so `paths` had never worked through tt at all. With
                // neither given the engine still refuses, in a sentence that names both.
                new Param { Name = "path", Type = PType.Path, Required = false, Desc = "name-path（与 paths 二选一）", Role = Role.ComponentPath },
                P.Attr("attr", "layout"),
                P.Str("value", true, "新值"),
                Opt(PType.PathList, "paths", "批量：对这些 path 一起改，忽略 path")),
            F("set_layout_attrs", G_ATTR, "一次改一个元素的多个布局属性（先全量校验，再全量写）", true, "delta", E_LAYOUT,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Attrs("attrs", "{\"属性名\":\"值\", …}；一次请求改多个，合成一步撤销")),
            F("set_tree_source", G_ATTR, "Tree 数据来源的某一格", true, "delta", E_NOOP,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "Tree 的 name-path", Role = Role.ComponentPath },
                P.Str("element", true, "子元素名"),
                P.Str("property", true, "属性名"),
                P.Str("value", true, "新值")),
            F("rename_component", G_ATTR, "改控件代号（设计器强制全表单唯一）", true, "delta", E_NOOP,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.As(P.Str("name", true, "新代号"), Role.NewName)),

            // ------------------------------------------------------------ 结构 (12)
            // with_label defaults TRUE, which is a behaviour change and the faithful one: the
            // widget box never creates these eight types bare (ComponentFactory .cs:98-127 via
            // CreateEmptyComponentWithLabel), so a form built through us looked different from one
            // built by hand. Pass with_label=false for the old single-element behaviour.
            F("add_widget", G_STRUCT, "往容器里加一个控件（走 AddComponetsUndoRedoCommand）", true, "el", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "父容器的 name-path", Role = Role.ComponentPath },
                P.Enum("type", WIDGETS),
                Opt(PType.Str, "name", "控件代号，默认走 ComponentFactory.GetNewName"),
                Opt(PType.Bool, "with_label", "给需要标签的 8 种控件配一个 Label 并双向绑定+AddSpecBinding"
                    + "（Edit/ComboBox/TextEdit/ButtonEdit/DateEdit/DateTimeEdit/SpinEdit/TimeEdit）。"
                    + "默认 true = 设计器控件箱的行为；false 得到一个裸控件")),
            // `columns` is the designer's OWN shape, not an invention: its WidgetBox button
            // "Create a structure from database columns" (WidgetBox.xaml:138) opens
            // DBStructureCreator, whose only non-UI line is one call to
            // UICreator.Create(LayoutEditor, container, SelectedFields, key)
            // (DBStructureCreator.xaml.cs:178) -- N columns in ONE construction. The legacy
            // AddField.exe already did it that way (AddField.cs:242-275). Passing one column at a
            // time makes the designer redo its whole per-call setup per field, and that setup
            // costs more the larger the form already is: adding 84 columns one at a time measured
            // 137 s, with per-field cost climbing 0.55 -> 3.25 s. It also gets the container
            // WRONG -- CreateNoneContainerWidget (the default) makes label+widget per field, so
            // filling a Table that way plants a <Label> beside every widget, which no Table in
            // that form has; CreateTableContainer returns the Table alone.
            F("add_field", G_STRUCT, "从数据字典加字段（控件类型由列元数据决定）", true, "el", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "父容器的 name-path", Role = Role.ComponentPath },
                P.Str("table", true, "表名"),
                P.Str("column", false, "单列名；与 columns 二选一（两个都不给会被拒）"),
                P.StrList("columns", false, "一次加多列；与 column 二选一（两个都不给会被拒）。N 列走同一次构造调用，"
                    + "所以 container=Table 得到的是「一个 Table 装 N 列」而不是每列一个容器"),
                OptEnum("container", Struct.CONTAINER_TYPES, "容器模式，默认 None（就地放一对标签+控件）"),
                // `name` used to be declared here and was never read by AddFieldFn -- an
                // advertised parameter that silently does nothing is worse than an absent one.
                // It is not simply implemented either: a column-backed field's name IS its
                // binding key, so a caller-chosen name would have to match the column.
                Opt(PType.Str, "widget", "覆盖列元数据给的控件类型")),
            F("insert_at", G_STRUCT, "在指定序号插入控件（往前/往后新增栏位）", true, "el", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "父容器的 name-path", Role = Role.ComponentPath },
                P.Enum("type", WIDGETS),
                Opt(PType.Int, "index", "插入位置，默认追加"),
                Opt(PType.Bool, "with_label", "同 add_widget：给需要标签的 8 种控件配一个 Label。"
                    + "默认 true = 设计器控件箱的行为")),
            F("delete", G_STRUCT, "删元素：布局真删、规格留墓碑（status=d）", true, "delta", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths()),
            F("move", G_STRUCT, "改 Z 序（最前/前一个/下一个/最后）", true, "delta", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths(),
                P.As(P.Enum("to", new[] { "first", "prev", "next", "last" }), Role.Direction)),
            // move's Z-order sibling: `move` stays inside one parent, this one changes the parent.
            // DragComponentsUndoRedoCommand is the designer's drag-drop command and the only
            // reparent primitive there is -- AddToContainer makes a NEW box, and cut+paste is
            // cross-form and empties the source. Same form only, deliberately.
            F("reparent", G_STRUCT, "把元素搬进另一个容器（同表单；DragComponentsUndoRedoCommand）",
                true, "delta", E_STD,
                P.Handle(),
                P.List("paths", true),
                new Param { Name = "into", Type = PType.Path, Required = true,
                            Desc = "目标容器的 name-path" }),
            F("nudge", G_STRUCT, "按方向平移（MoveComponentsUndoRedoCommand）", true, "delta", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths(),
                P.As(P.Enum("direction", new[] { "up", "down", "left", "right" }), Role.Direction),
                Opt(PType.Int, "offset", "格数，默认 1")),
            F("align", G_STRUCT, "对齐（STRETCH/LEFT/RIGHT/TOP/BOTTOM）", true, "delta", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths(),
                P.As(P.Enum("option", new[] { "stretch", "left", "right", "top", "bottom" }), Role.Direction)),
            F("fit_size", G_STRUCT, "尺寸自适应", true, "delta", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths()),
            F("wrap", G_STRUCT, "把选中元素整体包进一个新容器（不是加字段）", true, "el", E_STD,
                P.Handle(),
                AnyPath(),
                AnyPaths(),
                P.Enum("type", new[] { "hbox", "vbox", "grid", "group" })),
            F("break_layout", G_STRUCT, "拆箱（wrap 的对称操作）", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "盒子元素的 name-path", Role = Role.ComponentPath }),
            F("convert_widget", G_STRUCT, "转换控件类型（14 种）", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Enum("type", WIDGETS)),
            F("convert_container", G_STRUCT, "转换容器类型", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Enum("type", Struct.CONVERT_CONTAINER_TARGETS)),

            // ------------------------------------------------------------ 页签 (2)
            F("add_page", G_PAGE, "新增页签（Folder 下的 Page）", true, "el", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "Folder 的 name-path", Role = Role.ComponentPath },
                Opt(PType.Str, "name", "页签名")),
            F("delete_page", G_PAGE, "删除页签（含其子节点）", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "Page 的 name-path", Role = Role.ComponentPath }),

            // ------------------------------------------------------------ 语义 / Action (4)
            F("insert_semantic", G_SEM, "插入语义节点（REFERENCE/MULTILANG/PROGREL）", true, "el", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "父容器的 name-path", Role = Role.ComponentPath },
                P.Enum("use", new[] { "reference", "multilang", "progrel" })),
            F("add_action", G_SEM, "新增 Action（<act>，id=控件代号）", true, "el", E_NOOP,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true,
                            Desc = "父容器的 name-path；空串 = 独立动作（无按钮）", Role = Role.ComponentPath },
                // NOT a static enum. The vocabulary is the FORM's own: all / mi / di<n> / db<n>,
                // where n comes from that form's s_detail<n> records -- aapp320 has 12 tokens,
                // another form has 3. A fixed list cannot express that, and pinning one to
                // {"all","none"} made every other token E_BAD_PARAM before the body could look at
                // the form, so "adding db* evicts all" was unreachable over the wire. The body
                // validates against the form and returns the real set as `vocabulary`; detail.legal
                // carries it on rejection. W3-C found this.
                P.Str("type", false, "型态，逗号分隔；合法集是该表单自己的（见返回体的 vocabulary）"),
                P.As(Opt(PType.Str, "name", "action id"), Role.ActionId)),
            F("delete_action", G_SEM, "删除 Action", true, "delta", E_NOOP,
                P.Handle(),
                P.As(P.Str("id", true, "action id"), Role.ActionId)),
            F("set_action_types", G_SEM, "改 Action 的类型组合", true, "delta", E_NOOP,
                P.Handle(),
                P.As(P.Str("id", true, "action id"), Role.ActionId),
                // Same reason as add_action's `type`: the legal set is per-form, so it cannot be
                // an enum here. "none" requests the empty set, which rule 3 folds back to all.
                P.Str("types", true, "型态，逗号分隔；合法集见返回体的 vocabulary，\"none\" 表示空集")),

            // ------------------------------------------------------------ 多语言 / 选项 / 串查 (6)
            F("set_local_string", G_MULTI, "写一条本地化串（sfield）", true, "delta", E_NOOP,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Str("name", true, "串名，如 lbl_pmdlud001"),
                P.Str("text", true, "内容")),
            F("set_items", G_MULTI, "ComboBox/RadioGroup 的选项值（<Item>）", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.StrList("items", true, "\"name|text|description\" 字符串数组（不是路径）")),
            F("set_progrel_programs", G_MULTI, "串查程序的增删", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Str("program", true, "程序名"),
                Opt(PType.Bool, "isDelete", "true = 删掉这条串查")),
            F("set_table_association", G_MULTI, "改表关联（<table> 段）", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Str("table", true, "表名"),
                Opt(PType.Str, "column", "列名")),
            F("set_spec_description", G_MULTI, "写 SD 规格描述（CDATA）", true, "delta", E_NOOP,
                P.Handle(),
                // `path` before `kind`, like every other verb that takes both (set_spec_attr,
                // set_spec_attrs). Declaration order IS the wire key order (internal/dev/tzs's
                // argmap builds the frame in manifest order), and one verb emitting its keys in a
                // different order from its three siblings is the kind of difference a reader has to
                // stop and explain.
                new Param { Name = "path", Type = PType.Path, Required = true,
                    Desc = "name-path；或程序级规格的 all / mi_all / db_all / di_all",
                    Role = Role.ComponentPath },
                // Deliberately NOT P.Kind(): that helper declares `kind` required, and for the four
                // program-level aliases above it is ignored -- the function body's own check
                // ("set_spec_description 需要 kind（七种之一），或 path=\"all\"…", Semantic.cs) is the
                // one that knows the difference. Declaring it required here made the natural call
                // {"path":"all","content":"…"} impossible: the CLI refused it before the request was
                // ever sent, so the four program-level specs this verb implements on purpose were
                // unreachable from the wire. Same shape as Action.cs's branch, which validates
                // `type` in the body because the answer depends on the path.
                new Param { Name = "kind", Type = PType.Kind, Required = false,
                    Desc = "哪一类规格节点；path 是 all/mi_all/db_all/di_all 时忽略" },
                P.Str("content", true, "新内容")),
            F("set_cited", G_MULTI, "引用 / 取消引用标准（SpecCitedUndoRedoCommand）", true, "delta",
                // Not E_NOOP: set_cited has no "already this value" answer -- it runs the command
                // unconditionally and reports wasCited/cited. It does resolve a spec node though
                // (FirstSlot), so it is the one verb that needs E_NO_SPEC_NODE without taking a
                // `kind` (and therefore without the derivation in AdvertisedErrors).
                Plus(E_STD, "E_NO_SPEC_NODE"),
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Bool("cited", true, "true = 引用")),

            // ------------------------------------------------------------ Tab 顺序 (2)
            F("set_tab_order", G_TAB, "重排 tabIndex", true, "delta", E_STD,
                P.Handle(),
                P.List("paths", true)),
            F("tab_action", G_TAB, "Tab 顺序动作六合一", true, "delta", E_STD,
                P.Handle(),
                P.List("paths", true),
                P.As(P.Enum("action", new[] { "first", "prev", "next", "last", "clear", "auto" }), Role.Direction)),

            // ------------------------------------------------------------ 校验 / 工具 (4)
            Slow(F("validate", G_TOOL, "跑设计器自己的校验器，报 delta（慢，勿循环）", false, "delta", E_STD,
                P.Handle())),
            N("base_data", G_TOOL, "基础资料（MasterDetailView）", "list<el>", E_STD,
                Opt(PType.Str, "table", "表名"),
                Opt(PType.Str, "column", "列名")),
            F("set_excluded", G_TOOL, "排除 / 取消排除控件", true, "delta", E_STD,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path", Role = Role.ComponentPath },
                P.Bool("excluded", true, "true = 排除")),
            // The enum is the UNION across workspaces; the function body narrows it against the
            // workspace's own mta/code_template.xml and reports detail.legal from the file. R was
            // missing here, so a workspace that has it (hengshuo/prd does: F/P/Q/R/W) could never
            // set it -- the manifest rejected the value before the body ever saw the file.
            // W3-B found it. Keep this list a superset of what any workspace ships.
            F("set_code_template", G_TOOL, "改 code_template（F/P/Q/R/W）", true, "delta", E_NOOP,
                P.Handle(),
                new Param { Name = "path", Type = PType.Path, Required = true, Desc = "name-path（只校验，作用域是整张表单）", Role = Role.ComponentPath },
                P.Enum("template", new[] { "F", "P", "Q", "R", "W" })),
        };

        static SpecFn Slow(SpecFn f) { f.Slow = true; return f; }

        static readonly Dictionary<string, SpecFn> _byName = Build();

        static Dictionary<string, SpecFn> Build() {
            var d = new Dictionary<string, SpecFn>(StringComparer.Ordinal);
            foreach (SpecFn f in All) d[f.Name] = f;
            return d;
        }

        /// <summary>The descriptor for a name, or null when nothing declares it. Null is what the
        /// transport turns into E_UNKNOWN_METHOD, as opposed to E_NOT_IMPLEMENTED for a name that
        /// is declared but has no body yet.</summary>
        public static SpecFn Get(string name) {
            if (name == null) return null;
            SpecFn f;
            return _byName.TryGetValue(name, out f) ? f : null;
        }

        // ================================================================ validation

        /// <summary>
        /// Checks a request's args against the descriptor: arity, type, enum membership, required.
        /// Returns null when the call is well formed, else the wire-level reason -- callers put it
        /// straight into error.detail (SPEC §11.24 (a) requires detail to name the parameter, or
        /// the AI cannot self-correct).
        ///
        /// Arity is enforced in both directions: an undeclared key is refused, not ignored. That is
        /// the same disease §11.24 (d) describes on the write side -- `Edit set` writing case="upper"
        /// into the .tsd -- and a silently-ignored typo (`--attrr`) is indistinguishable from a
        /// legal call until someone reads the file. A parameter that is not in this table does not
        /// exist, so a Fn that needs one must declare it here.
        ///
        /// NOT checked here: the runtime attr whitelist behind describe_from. That set is
        /// materialised from the live model (SPEC §11.24 (c)), and this method has no session --
        /// it is the Fn's job, per §11.24 (d) 铁律 2.
        /// </summary>
        public static JObject Check(SpecFn fn, JObject args) {
            if (args == null) args = new JObject();

            foreach (JProperty p in args.Properties()) {
                bool known = false;
                foreach (Param d in fn.Params) if (d.Name == p.Name) { known = true; break; }
                if (!known) {
                    return Detail(p.Name, "unknown",
                        "参数 " + p.Name + " 不在 manifest 里（函数 " + fn.Name + "）；可用: " + Names(fn));
                }
            }

            foreach (Param d in fn.Params) {
                JToken v = args[d.Name];
                bool absent = v == null || v.Type == JTokenType.Null;
                if (absent) {
                    if (d.Required) return Detail(d.Name, "required", "参数 " + d.Name + " 必填");
                    continue;
                }
                string bad = TypeProblem(d, v);
                if (bad != null) return Detail(d.Name, "invalid", "参数 " + d.Name + ": " + bad);
            }
            return null;
        }

        /// <summary>Null when v is acceptable for d, else a one-line reason. Enum and Kind
        /// membership are checked here because both sets are static; anything richer (the
        /// describe_from whitelist) needs the model and is deferred.</summary>
        static string TypeProblem(Param d, JToken v) {
            switch (d.Type) {
                case PType.Int:
                    if (v.Type != JTokenType.Integer) return "需要整数，收到 " + v.Type;
                    return null;
                case PType.Bool:
                    if (v.Type != JTokenType.Boolean) return "需要 true/false，收到 " + v.Type;
                    return null;
                case PType.PathList:
                case PType.StrList:
                    if (v.Type != JTokenType.Array) return "需要字符串数组，收到 " + v.Type;
                    foreach (JToken e in (JArray)v)
                        if (e.Type == JTokenType.Array || e.Type == JTokenType.Object)
                            return "数组元素必须是字符串，收到 " + e.Type;
                    return null;
                case PType.Attrs:
                    // Shape only. WHICH attributes are legal, and which VALUES, are properties of the
                    // node and of the workspace specification -- this method has no session, so it
                    // cannot know either (the same division §11.24 (c) draws for describe_from).
                    if (v.Type != JTokenType.Object) return "需要 JSON 对象（属性名 → 字符串值），收到 " + v.Type;
                    var map = (JObject)v;
                    if (map.Count == 0) return "是空对象；只改一个属性用单数形式的动词";
                    foreach (JProperty p in map.Properties())
                        if (p.Value == null || p.Value.Type != JTokenType.String)
                            return "属性 " + p.Name + " 的值需要字符串，收到 "
                                 + (p.Value == null ? "null" : p.Value.Type.ToString())
                                 + "（属性值在 .4fd/.tsd 里都是文本）";
                    return null;
                case PType.Enum:
                    if (v.Type != JTokenType.String) return "需要一个字符串，收到 " + v.Type;
                    if (!In(d.Values, (string)v))
                        return "取值不在允许集合内: " + (string)v + "；可用: " + Join(d.Values);
                    return null;
                case PType.Kind:
                    if (v.Type != JTokenType.String) return "需要一个字符串，收到 " + v.Type;
                    if (!In(SpecSlots.Kinds, (string)v))
                        return "不是七种规格节点之一: " + (string)v + "；可用: " + Join(SpecSlots.Kinds);
                    return null;
                case PType.KindOrLayout:
                    // describe_kind only. Accepts the seven, "layout", and the "spec:<kind>" spelling
                    // the body also strips (Fns/Read.cs) -- that spelling is the shape `attr`'s
                    // describe_from template publishes ("spec:<kind>"), so a caller that copied it
                    // from there still works.
                    if (v.Type != JTokenType.String) return "需要一个字符串，收到 " + v.Type;
                    {
                        string k = (string)v;
                        if (In(SpecSlots.Kinds, k) || k == "layout") return null;
                        if (k.StartsWith("spec:", StringComparison.Ordinal) && In(SpecSlots.Kinds, k.Substring(5)))
                            return null;
                        return "不是七种规格节点之一、也不是 layout: " + k
                             + "；可用: " + Join(SpecSlots.Kinds) + ",layout";
                    }
                default:   // Str / Path / Handle / AttrName
                    if (v.Type == JTokenType.Object || v.Type == JTokenType.Array)
                        return "需要一个字符串，收到 " + v.Type;
                    return null;
            }
        }

        static bool In(string[] set, string v) {
            if (set == null) return true;
            foreach (string s in set) if (string.Equals(s, v, StringComparison.Ordinal)) return true;
            return false;
        }

        static string Join(string[] a) { return a == null ? "" : string.Join("|", a); }

        static string Names(SpecFn fn) {
            var b = new System.Text.StringBuilder();
            foreach (Param p in fn.Params) { if (b.Length > 0) b.Append(' '); b.Append(p.Name); }
            return b.Length == 0 ? "(无)" : b.ToString();
        }

        /// <summary>error.detail for a parameter problem. `param` is the key the caller got wrong,
        /// which is the whole point -- an error that does not name the parameter cannot be
        /// self-corrected (SPEC §11.24 (a)).</summary>
        static JObject Detail(string param, string reason, string message) {
            var o = new JObject();
            o["param"] = param;
            o["reason"] = reason;
            o["message"] = message;
            return o;
        }

        // ================================================================ serialisation

        /// <summary>The manifest as JSON, in the exact shape of SPEC §11.24 (c): fn / group /
        /// desc / writes / slow / args[{n,t,req,values,from}]. `writes` is Mutating and `slow` is
        /// Slow; the key names are the contract's, not nicer ones of my own, because this document
        /// is what the MCP adapter will read.</summary>
        public static string ToJson() {
            var arr = new JArray();
            foreach (SpecFn f in All) {
                var o = new JObject();
                o["fn"] = f.Name;
                o["group"] = f.Group;
                o["desc"] = f.Summary;
                o["writes"] = f.Mutating;
                o["slow"] = f.Slow;
                o["returns"] = f.Returns;
                o["needsHandle"] = f.NeedsHandle;
                var args = new JArray();
                foreach (Param p in f.Params) {
                    var a = new JObject();
                    a["n"] = p.Name;
                    a["t"] = TypeName(p.Type);
                    a["req"] = p.Required;
                    if (p.Desc != null) a["desc"] = p.Desc;
                    if (p.Values != null) a["values"] = new JArray(p.Values);
                    if (p.DescribeFrom != null) a["from"] = p.DescribeFrom;
                    if (p.Role != null) a["role"] = p.Role;
                    args.Add(a);
                }
                o["args"] = args;
                o["errors"] = AdvertisedErrors(f);
                arr.Add(o);
            }
            return arr.ToString(Newtonsoft.Json.Formatting.Indented);
        }

        public static string TypeName(PType t) {
            switch (t) {
                case PType.Int:      return "int";
                case PType.Bool:     return "bool";
                case PType.Handle:   return "handle";
                case PType.Kind:     return "kind";
                // Its own name, not "kind": the point of the type is that the accepted set is
                // wider, and a reader of --manifest should be able to see that. Publishing
                // "string" here (which is what the default below does) silently erases the
                // difference.
                case PType.KindOrLayout: return "kind-or-layout";
                case PType.AttrName: return "attr";
                case PType.Enum:     return "enum";
                case PType.Path:     return "path";
                case PType.PathList: return "path[]";
                case PType.StrList:  return "string[]";
                case PType.Attrs:    return "attrs";
                // ADD A LINE HERE WHEN YOU ADD A PType MEMBER. A forgotten member publishes as
                // "string", which every consumer already knows -- so nothing fails, the type
                // information just disappears. KindOrLayout was built that way once
                // (2026-09-25) before the live check caught it.
                default:             return "string";
            }
        }

        /// <summary>--help, generated from the table so it can never disagree with what the
        /// server validates.</summary>
        public static string Help() {
            var b = new System.Text.StringBuilder();
            b.Append("tzs-cli -- T100 .tzs 设计器的长驻 JSON-RPC 客户端\n\n");
            b.Append("用法:\n");
            b.Append("  tzs-cli open <file> [--workspace <dir>]\n");
            b.Append("  tzs-cli call <fn> [--<参数> <值> ...]\n");
            b.Append("  tzs-cli save --handle <h> --out <file>\n");
            b.Append("  tzs-cli stop\n");
            b.Append("  tzs-cli --help | --manifest\n\n");
            b.Append("传输: 命名管道 \\\\.\\pipe\\" + Rpc.PipeName + "（由 tzs-server 常驻）；\n");
            b.Append("      连不上时自动 spawn 一个 tzs-server 再重试一次。\n\n");
            b.Append("函数清单（本表由 manifest 生成，与服务器校验同源）:\n");

            string group = null;
            foreach (SpecFn f in All) {
                if (f.Group != group) { group = f.Group; b.Append("\n[").Append(group).Append("]\n"); }
                b.Append("  ").Append(Pad(f.Name, 22)).Append(f.Summary);
                if (f.Slow) b.Append("   [slow]");
                if (f.Mutating) b.Append("   [写]");
                b.Append('\n');
                foreach (Param p in f.Params) {
                    string t = p.Type == PType.Enum ? "enum(" + GetEnumLabel(p) + ")"
                             : p.Type == PType.AttrName ? "attr:" + p.DescribeFrom
                             : TypeName(p.Type);
                    b.Append("      --").Append(Pad(p.Name, 18)).Append(Pad(t, 26));
                    if (p.Required) b.Append("必填");
                    if (p.Desc != null) b.Append("  ").Append(p.Desc);
                    b.Append('\n');
                }
            }
            b.Append("\nE_* 错误码: E_BAD_REQUEST / E_UNKNOWN_METHOD / E_BAD_PARAM / E_NOT_IMPLEMENTED /\n");
            b.Append("            E_NOT_FOUND / E_KEY_IN_USE / E_DESIGNER / E_INTERNAL / E_FATAL_LOAD_TIMEOUT /\n");
            b.Append("            E_SERVER_DIED（后两个由客户端在传输层判定）\n");
            return b.ToString();
        }

        static string GetEnumLabel(Param p) {
            if (p.Values == null) return "?";
            return p.Values.Length > 6 ? p.Values.Length + " 个值" : Join(p.Values);
        }

        static string Pad(string s, int n) {
            if (s.Length >= n) return s + " ";
            return s + new string(' ', n - s.Length);
        }
    }
}
