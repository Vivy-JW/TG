using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.IO.Pipes;
using System.Reflection;
using System.Text;
using Newtonsoft.Json;
using Newtonsoft.Json.Linq;

namespace TzsCli.Designer
{
    /// <summary>
    /// A TzsError carrying the error.detail the wire needs, without touching Errors.cs.
    ///
    /// Exists because SPEC §11.24 (a) makes the detail load-bearing -- "detail 里给可用值",
    /// "detail 里给近似候选" -- and an error that does not name the parameter it rejected cannot
    /// be self-corrected by the caller. Subclassing is enough: TzsError is not sealed.
    /// </summary>
    public sealed class DetailedError : TzsError {
        public readonly JObject Detail;
        public DetailedError(string code, string message, JObject detail) : base(code, message) {
            Detail = detail;
        }
    }

    /// <summary>
    /// One response line, classified into primitives.
    ///
    /// The CLI reads this instead of a JObject so that test/tzs-cli.cs needs no compile-time
    /// Newtonsoft reference: build.sh's `ref` mode hands a program -r: on the two project DLLs
    /// and nothing else, and all JSON handling therefore stays inside this library.
    /// </summary>
    public sealed class Reply {
        /// <summary>A line came back at all. False means EOF with no frame -- E_SERVER_DIED.</summary>
        public bool Received;
        /// <summary>The line was not a JSON object. The CLI must report E_SERVER_DIED rather than
        /// a parse error: for a dead or half-dead daemon a parser message reads like a fact about
        /// the form, which is the one outcome §11.24 (h) forbids.</summary>
        public bool ParseFailed;
        public bool Ok;
        public string Code;      // E_* when ok is false
        public string Kind;      // §11.24 (a) validation|not_found|designer|internal
        public string Message;
        /// <summary>result.handle when the frame carries one, so `tzs-cli open` can print the
        /// handle to the user without the CLI parsing JSON.</summary>
        public string ResultHandle;
        public string Raw;
    }

    /// <summary>
    /// The transport: line-delimited JSON over a stream, one request in flight, one response per
    /// request. SPEC §11.24 (a) (envelope), (e) (registration), (f) (daemon), (h) (fatal exit).
    ///
    /// Two front ends share one connection handler:
    ///
    ///   Stdio()   reads stdin, writes stdout. This is the mode the acceptance harness drives
    ///             (`echo '{"id":1,"fn":"list_open"}' | tzs-server`) and the one a test can use
    ///             without a pipe. Its wire IS the console, so it is a test/diagnostic mode.
    ///   Daemon()  listens on \\.\pipe\tzs-cli, one client at a time. The shipping mode: the
    ///             wire is the pipe, so nothing the ported designer code prints to Console can
    ///             reach it (SPEC §11.24 (f)).
    ///
    /// ---------------------------------------------------------------------------------------
    /// Why strictly serial, and why that is a correctness property rather than a simplification:
    ///
    /// The designer's live state is process-global and not thread-safe. SettingManager holds one
    /// tzpMap, one undoRedoManagerMap and one PreferenceManager.Current; TzpManager.Current is
    /// rewritten as a side effect of GetTzpManger (SPEC §11.24 (b)); the ported UndoRedo commands
    /// mutate a shared undo stack. Two requests in flight would interleave those with no lock the
    /// designer knows about. One request at a time, no worker threads -- the load watchdog inside
    /// Session.Open is the single exception, and it exists because it has to be able to fire while
    /// the main thread is parked inside Activator.CreateInstance (SPEC §11.24 (h)).
    /// ---------------------------------------------------------------------------------------
    /// </summary>
    public static class Rpc
    {
        /// <summary>The workspace the daemon defaults to when nothing names one. Kept here with
        /// the pipe name so the client and the server agree without either Boot-ing first.</summary>
        public const string DefaultWorkspace = @"D:\t100_wrok_dir\hengshuo\prd";

        /// <summary>Set by the CLI before it connects. The client has NOT called Designer.Boot --
        /// it cannot, it is a one-shot process -- so Designer.Workspace is empty there, and
        /// without this the client would hash "" and look for a pipe the server never created.
        /// That was a real bug: `tzs-cli --help` printed a pipe name with a 00000000 workspace
        /// tag while the daemon it spawned was listening on a different one.</summary>
        public static string WorkspaceHint;

        /// <summary>
        /// Derived, not constant. A fixed `tzs-cli` is machine-global, and that bit twice in one
        /// afternoon: W2-C's CLI attached silently to W2-A's daemon, which was serving a
        /// different build of this same assembly (stale behaviour, undetectable from the
        /// client); and W2-B avoided the CLI entirely because a stray daemon bound to one
        /// workspace would strand its `open` on another.
        ///
        /// Two things go in. The workspace, because a process is permanently bound to one --
        /// Designer.Boot cannot be re-pointed, so a daemon only ever serves one module. And this
        /// assembly's MVID, which changes on every build, so a client can never reach a daemon
        /// running stale bytes: it starts the matching one instead. Orphaned daemons from older
        /// builds are the cost, which is why `stop` takes no arguments and just works.
        ///
        /// Resolution order matters and is the whole point: a booted process uses what it booted
        /// with; a client uses its hint, then the environment, then the default. Both sides must
        /// reach the same string or the client spawns a daemon it cannot then talk to.
        /// </summary>
        public static string PipeName {
            get {
                string ws = Designer.Workspace;
                if (string.IsNullOrEmpty(ws)) ws = WorkspaceHint;
                if (string.IsNullOrEmpty(ws)) ws = Environment.GetEnvironmentVariable("TZSCLI_WS");
                if (string.IsNullOrEmpty(ws)) ws = DefaultWorkspace;
                ws = ws.ToLowerInvariant();
                int h = 0;
                foreach (char c in ws) h = unchecked(h * 31 + c);
                string mvid = typeof(Rpc).Assembly.ManifestModule.ModuleVersionId.ToString("N");
                return "tzs-cli-" + (h & 0x7fffffff).ToString("x8") + "-" + mvid.Substring(0, 8);
            }
        }

        /// <summary>8 MB. A line longer than this is refused rather than grown: the only thing
        /// that produces one is a caller that is not speaking the protocol, and the response to
        /// that is E_BAD_REQUEST plus a resync, not an OutOfMemoryException in the daemon.</summary>
        public const int MaxLine = 8 * 1024 * 1024;

        /// <summary>The frozen module list (SPEC §11.24 (e)). Written here as names rather than as
        /// direct calls on purpose: three of the five files are being written in parallel, so a
        /// compile-time reference would make this file uncompilable until all five exist. The
        /// list is frozen -- the *presence* of each module is not. A missing one is reported on
        /// stderr and its functions answer E_NOT_IMPLEMENTED, which is an honest answer.</summary>
        static readonly string[] Modules = {
            "Read", "Attr", "Validate", "Verify", "Session",
            // Wave 3. Added here, before the files exist, so the four agents creating them do
            // not all have to edit this one line -- the same coordination the SpecFn.cs hoist
            // did in Wave 2. A name with no class behind it is reported on stderr and its
            // functions answer E_NOT_IMPLEMENTED, which is an honest answer.
            "Struct", "PageTab", "Action", "Semantic",
        };

        static IDictionary<string, Fn> _map;

        /// <summary>name -> body, built once by concatenating the Fns/*.Register(map) calls.
        /// Dictionary<string, Fn> rather than a fancier shape so the modules can fill it without
        /// knowing anything about the transport.</summary>
        public static IDictionary<string, Fn> Fns {
            get {
                if (_map == null) _map = BuildMap();
                return _map;
            }
        }

        static IDictionary<string, Fn> BuildMap() {
            var map = new Dictionary<string, Fn>(StringComparer.Ordinal);
            Assembly self = typeof(Rpc).Assembly;
            foreach (string m in Modules) {
                Type t = FindModule(self, m);
                if (t == null) {
                    Err("rpc: 找不到模块 " + m + " -- 它声明的函数会回 E_NOT_IMPLEMENTED");
                    continue;
                }
                try {
                    Reflect.Call(t, "Register", map);
                } catch (Exception ex) {
                    // A module that throws while registering is a bug, but it must not take the
                    // daemon down at startup -- the other four modules are still useful, and a
                    // daemon that answers "not implemented" is strictly better than no daemon.
                    Err("rpc: " + t.FullName + ".Register 抛异常: " + ex.GetType().Name + ": " + ex.Message);
                }
            }
            return map;
        }

        /// <summary>
        /// Finds one Fns module by class name.
        ///
        /// Not by full name, because the namespace is not part of the frozen contract: §11.24 (e)
        /// freezes the file layout (src/Designer/Fns/*.cs) and the Register signature, not the
        /// enclosing namespace -- and the two halves of W2 landed on different ones
        /// (TzsCli.Designer.Fns for Session/Validate/Verify, TzsCli.Designer for Read/Attr). A
        /// module is identified by what it is: a type whose simple name matches and which exposes
        /// Register. Keying on the namespace would have silently reported two working modules as
        /// missing, and E_NOT_IMPLEMENTED for every function they implement.
        /// </summary>
        static Type FindModule(Assembly a, string name) {
            Type[] all;
            try { all = a.GetTypes(); }
            catch (ReflectionTypeLoadException ex) { all = ex.Types; }
            foreach (Type t in all) {
                if (t == null || t.Name != name) continue;
                if (HasRegister(t)) return t;
            }
            return null;
        }

        static bool HasRegister(Type t) {
            foreach (MethodInfo m in t.GetMethods(BindingFlags.Public | BindingFlags.Static))
                if (m.Name == "Register" && m.GetParameters().Length == 1) return true;
            return false;
        }

        // ================================================================ fatal exit (h)

        static Conn _inflight;
        static bool _hook;

        /// <summary>
        /// SPEC §11.24 (h): a load timeout is terminal. The watchdog thread cannot abandon the
        /// load -- the main thread is parked inside Activator.CreateInstance and a thread in a
        /// dispatcher frame never observes a cancellation flag -- so all it can do is Exit(3).
        /// This hook is what lets the request that triggered it get an answer first: the frame is
        /// written and flushed *before* the watchdog calls Exit, so the caller sees
        /// E_FATAL_LOAD_TIMEOUT and then EOF, rather than EOF alone.
        ///
        /// The distinction matters downstream: "fatal frame then EOF" is a deterministic,
        /// explainable failure (mta/tables.xml is missing a table) and must NOT be retried, while
        /// "EOF with no frame" is a dead daemon and IS retried. Conflating them into a JSON parse
        /// error would tell the caller "this form has no such element", which is a lie.
        /// </summary>
        public static void InstallFatalHook() {
            if (_hook) return;
            _hook = true;
            Session.OnFatal = delegate(string path) {
                Conn c = _inflight;
                if (c == null) return;
                try { c.WriteFatal(path); }
                catch (Exception ex) { Err("rpc: 致命帧写出失败: " + ex.Message); }
            };
        }

        // ================================================================ front ends

        /// <summary>Reads requests from stdin, writes frames to stdout. Returns the process exit
        /// code. See the class comment for why this is a test mode rather than the shipping one.</summary>
        public static int Stdio() {
            InstallFatalHook();
            Stream outS;
            try { outS = Console.OpenStandardOutput(); } catch { outS = new MemoryStream(); }
            // Diagnostics only. Console.Write* from the ported designer code must not land on the
            // wire -- in this mode the wire *is* stdout. (Json.cs writes to
            // Console.OpenStandardOutput() directly, which this cannot intercept; that is one more
            // reason the daemon, where the wire is the pipe, is the shipping mode.)
            try { Console.SetOut(Console.Error); } catch { }
            Err("tzs-server: stdio 模式（一次性），工作区见上方引导日志");
            bool stop;
            new Conn(Console.OpenStandardInput(), outS).Serve(out stop);
            return 0;
        }

        /// <summary>Listens on \\.\pipe\tzs-cli until a client sends `stop`. One client at a time:
        /// a second client is served after the first disconnects, which is what keeps the
        /// strictly-serial guarantee.
        ///
        /// One server instance is created and reused across clients (Disconnect then
        /// WaitForConnection again) rather than disposed and recreated. Recreating would leave a
        /// window with no pipe object at all, into which a client's Connect falls -- and the
        /// client would then spawn a second daemon, which would find the name busy and exit. The
        /// loop below has no such window.</summary>
        public static int Daemon() {
            InstallFatalHook();
            Err("tzs-server: 监听 \\\\.\\pipe\\" + PipeName);
            NamedPipeServerStream p;
            try {
                p = new NamedPipeServerStream(PipeName, PipeDirection.InOut, 1,
                                              PipeTransmissionMode.Byte, PipeOptions.None);
            } catch (IOException ex) {
                // maxNumberOfServerInstances is 1, so a second daemon on this machine cannot
                // create the instance at all. Exiting quietly is right: the caller's connect
                // will find the pipe the other process is serving.
                Err("tzs-server: 管道已被占用，退出（另一台守护进程在服务）: " + ex.Message);
                return 0;
            }
            try {
                for (;;) {
                    try { p.WaitForConnection(); }
                    catch (Exception ex) {
                        Err("tzs-server: WaitForConnection: " + ex.Message);
                        return 1;
                    }
                    bool stop = false;
                    try { new Conn(p, p).Serve(out stop); }
                    catch (Exception ex) {
                        Err("tzs-server: 连接异常: " + ex.GetType().Name + ": " + ex.Message);
                    }
                    try { p.Disconnect(); } catch { }
                    if (stop) { Err("tzs-server: stop -- 退出"); return 0; }
                }
            } finally {
                try { p.Dispose(); } catch { }
            }
        }

        // ================================================================ client side

        /// <summary>Connects to the daemon, or returns null (never throws) so the caller can
        /// decide whether to spawn one. The caller owns the returned stream.</summary>
        public static NamedPipeClientStream Connect(int timeoutMs) {
            NamedPipeClientStream c = null;
            try {
                c = new NamedPipeClientStream(".", PipeName, PipeDirection.InOut);
                c.Connect(timeoutMs);
                return c;
            } catch {
                try { if (c != null) c.Dispose(); } catch { }
                return null;
            }
        }

        /// <summary>Writes one request line and reads one response line. eof=true with a null
        /// return is "the daemon went away without answering" -- E_SERVER_DIED on the CLI side.
        /// overflow=true means the response exceeded MaxLine and was discarded to the next \n.
        ///
        /// readTimeoutMs is advisory only, and callers must enforce it themselves: PipeStream
        /// rejects ReadTimeout ("Timeouts are not supported on this stream"), so there is no
        /// timeout to install here. tzs-cli wraps the call in a bounded wait.</summary>
        public static string Exchange(Stream pipe, string requestLine, int readTimeoutMs,
                                      out bool eof, out bool overflow) {
            eof = false; overflow = false;
            byte[] b = new UTF8Encoding(false).GetBytes(requestLine + "\n");
            pipe.Write(b, 0, b.Length);
            pipe.Flush();
            var r = new LineReader(pipe);
            byte[] line = r.ReadLine(out overflow, out eof);
            if (line == null) return null;
            return Decode(line);
        }

        /// <summary>Classifies a response line. Never throws: anything that is not a JSON object
        /// with an `ok` field comes back ParseFailed, which the CLI turns into E_SERVER_DIED.</summary>
        public static Reply Classify(string line) {
            var r = new Reply();
            r.Raw = line;
            if (line == null) return r;
            r.Received = true;
            string t = line.Trim();
            if (t.Length == 0 || t[0] != '{') { r.ParseFailed = true; return r; }
            JObject o;
            try { o = JObject.Parse(t); } catch { r.ParseFailed = true; return r; }
            JToken ok = o["ok"];
            r.Ok = ok != null && ok.Type == JTokenType.Boolean && (bool)ok;
            JObject err = o["error"] as JObject;
            if (err != null) {
                r.Code = Scalar(err["code"]);
                r.Kind = Scalar(err["kind"]);
                r.Message = Scalar(err["message"]);
            }
            JObject res = o["result"] as JObject;
            if (res != null) r.ResultHandle = Scalar(res["handle"]);
            return r;
        }

        static string Scalar(JToken t) {
            if (t == null || t.Type == JTokenType.Null) return null;
            return t.Type == JTokenType.String ? (string)t : t.ToString(Formatting.None);
        }

        /// <summary>Builds a request line: {"id":n,"fn":"...","args":{...}}. argsJson is the args
        /// object already rendered by Json.J, spliced verbatim rather than re-parsed.
        ///
        /// Not validating it with JObject.Parse is deliberate: this runs in a process that lives
        /// for a few hundred milliseconds, and the extra parse buys nothing -- the string was
        /// produced by this library's own escaper key by key, so it is well formed by construction.
        /// A caller that hand-writes argsJson gets E_BAD_REQUEST from the server, which is the
        /// same answer the parse would have produced locally, minus a process worth of JIT.</summary>
        public static string Request(int id, string fn, string argsJson) {
            if (string.IsNullOrEmpty(argsJson)) argsJson = "{}";
            return "{\"id\":" + id.ToString(CultureInfo.InvariantCulture) +
                   ",\"fn\":" + Json.J(fn) + ",\"args\":" + argsJson + "}";
        }

        public static string ManifestJson() { return Manifest.ToJson(); }

        public static string Help() { return Manifest.Help(); }

        // ================================================================ session lookup

        /// <summary>
        /// handle -> Session. Session keeps its registry keyed by ProgramKey (which is correct:
        /// that is the key a handle must be unique against), so the lookup by handle happens here,
        /// by scanning that registry, rather than by adding a second one that could disagree with
        /// it. Read directly from the field because Session exposes no accessor and its own file
        /// is not ours to change.
        /// </summary>
        public static Session FindByHandle(string handle) {
            if (string.IsNullOrEmpty(handle)) return null;
            IDictionary open = Reflect.Prop2(typeof(Session), "_open") as IDictionary;
            if (open == null) return null;
            foreach (DictionaryEntry de in open) {
                Session s = de.Value as Session;
                if (s != null && string.Equals(s.Handle, handle, StringComparison.Ordinal)) return s;
            }
            return null;
        }

        /// <summary>
        /// Looks up an open session by a MEANINGFUL name instead of by the opaque handle:
        /// either the program name ("aapp320") or the ProgramKey string ("aapp320|Form").
        ///
        /// Why this exists: handles are minted as "h1", "h2", ... and are never reused
        /// (SPEC §11.24 (g)), so a caller that wants to "edit aapp320's form" has to carry an
        /// opaque, perishable id through every single call. Both names a caller would rather use
        /// are already here -- `open` returns them, `list_open` reports them -- so accepting them
        /// costs nothing and removes a whole class of "which handle was it again?".
        ///
        /// This is an ADDITIVE relaxation of §11.24 (b): the wire shape does not change, `handle`
        /// is still a string, and an exact handle match still wins (handles are always "h"+N, so
        /// a program name can never be mistaken for one).
        ///
        /// Returns EVERY match, never the first: one program can be open once per type (a Form
        /// and a Report share the program name), and silently picking one would be a wrong answer
        /// that looks like a right one. The caller turns 0/1/N into the right message.
        /// </summary>
        public static List<Session> FindByKey(string programOrKey) {
            var hits = new List<Session>();
            if (string.IsNullOrEmpty(programOrKey)) return hits;
            IDictionary open = Reflect.Prop2(typeof(Session), "_open") as IDictionary;
            if (open == null) return hits;
            foreach (DictionaryEntry de in open) {
                Session s = de.Value as Session;
                if (s == null || s.Closed) continue;
                // The registry's key IS the ProgramKey string (Session.SlotKey), so matching it
                // needs no second derivation of the format -- which is the whole point of reading
                // the live dictionary instead of recomputing a key.
                string slot = de.Key as string;
                if (string.Equals(s.Program, programOrKey, StringComparison.Ordinal)
                    || (slot != null && string.Equals(slot, programOrKey, StringComparison.Ordinal)))
                    hits.Add(s);
            }
            return hits;
        }

        /// <summary>
        /// Looks up an open session by the .tzs path it was opened from.
        ///
        /// Exists so a composed verb can mean "open it if it is not already open" without walking
        /// into E_KEY_IN_USE: if this file is already serving a handle, reuse that handle rather
        /// than ask for a second load of the same key. That trap is real and easy to hit by hand
        /// (a previous command left the form open, so the next `open` fails) -- measured while
        /// writing the task-verb baseline.
        ///
        /// Comparison is full-path-normalised and case-insensitive: the caller's spelling of the
        /// path is not necessarily the one the engine stored.
        /// </summary>
        public static Session FindByPath(string path) {
            if (string.IsNullOrEmpty(path)) return null;
            string want = NormPath(path);
            IDictionary open = Reflect.Prop2(typeof(Session), "_open") as IDictionary;
            if (open == null) return null;
            foreach (DictionaryEntry de in open) {
                Session s = de.Value as Session;
                if (s == null || s.Closed) continue;
                if (NormPath(s.Path) == want) return s;
            }
            return null;
        }

        static string NormPath(string p) {
            if (string.IsNullOrEmpty(p)) return "";
            string q;
            try { q = Path.GetFullPath(p); } catch { q = p; }
            return q.TrimEnd('\\', '/').ToLowerInvariant();
        }

        /// <summary>Every open handle, for the "you gave me a handle that does not exist" detail.
        /// SPEC §11.24 (a) asks for an approximate candidate list; the live handle list is the
        /// only useful one, and §11.24 (g) makes it worth printing in full because two files can
        /// share a ProgramKey and only list_open would show it.
        ///
        /// `key` is included so the detail itself teaches the alternative addressing: a caller
        /// that gets this list can immediately retry with `handle: "aapp320|Form"` instead of
        /// having to work out that form from `program` + a package type.</summary>
        public static JArray OpenHandles() {
            var arr = new JArray();
            IDictionary open = Reflect.Prop2(typeof(Session), "_open") as IDictionary;
            if (open == null) return arr;
            foreach (DictionaryEntry de in open) {
                Session s = de.Value as Session;
                if (s == null || s.Closed) continue;
                var o = new JObject();
                o["handle"] = s.Handle;
                o["program"] = s.Program;
                if (de.Key != null) o["key"] = de.Key.ToString();
                o["path"] = s.Path;
                o["mutable"] = s.Mutable;
                arr.Add(o);
            }
            return arr;
        }

        // ================================================================ code mapping

        /// <summary>
        /// TzsError.Code -> (wire code, error.kind). TzsError carries a single string because
        /// Errors.cs predates §11.24 (a)'s split between the machine code and the recoverability
        /// category; keeping the table here means the split lives in one place and the throw sites
        /// stay untouched.
        ///
        /// The kind is what the caller acts on, and the assignments follow §11.24 (a)'s "调用方该
        /// 怎么办" column: validation/not_found are self-correctable, designer and internal are
        /// not. key_in_use is `designer` because the fix is a fact about the world (close the
        /// other handle), not about the request.
        /// </summary>
        internal static void Map(string code, out string wire, out string kind) {
            switch (code) {
                case "validation":   wire = "E_BAD_PARAM";       kind = "validation"; return;
                case "bad_param":    wire = "E_BAD_PARAM";       kind = "validation"; return;
                case "bad_request":  wire = "E_BAD_REQUEST";     kind = "validation"; return;
                case "unknown_method": wire = "E_UNKNOWN_METHOD"; kind = "validation"; return;
                case "not_found":    wire = "E_NOT_FOUND";       kind = "not_found";  return;
                case "designer":     wire = "E_DESIGNER";        kind = "designer";   return;
                case "key_in_use":   wire = "E_KEY_IN_USE";      kind = "designer";   return;
                case "not_implemented": wire = "E_NOT_IMPLEMENTED"; kind = "internal"; return;
                case "internal":     wire = "E_INTERNAL";        kind = "internal";   return;

                // Codes a Fn reports directly. They must be classified by their own semantics
                // rather than falling through to `internal`, because `kind` is the whole basis
                // of a caller's decision to self-correct (SPEC §11.24 (a)). W2-B found these
                // arriving as "don't retry" while carrying `detail.legal` -- the exact opposite
                // of what the contract says the field is for.
                // E_DESIGNER is the one that got missed, and the miss was systemic rather than
                // local: Attr.Refused reports "E_DESIGNER", which fell to the default below and
                // went out as kind:"internal" -- so EVERY designer refusal in the server (rename
                // collision, page delete rules, mime gates) told the caller "unexpected failure,
                // do not retry" instead of "the form says no". W3-B found it; the contract's
                // whole point is that those two are different.
                case "E_DESIGNER":      wire = code; kind = "designer";   return;
                case "E_ATTR_NOT_WHITELIST": wire = code; kind = "validation"; return;  // detail.legal
                case "E_ATTR_VALUE_ILLEGAL": wire = code; kind = "validation"; return;  // detail.legal + hint
                // The plural writers landed part of the request. `designer` rather than
                // `validation`: the names and values were checked before anything was written, so a
                // failure here is the model refusing mid-batch (see Attr.Partial).
                case "E_ATTR_PARTIAL":      wire = code; kind = "designer";   return;  // detail.applied + failed
                case "E_BAD_PARAM":     wire = code; kind = "validation"; return;
                // E_HANDLE_BUSY used to be listed here ("句柄状态不对"). Nothing ever threw it:
                // the only state test in the engine is Closed, and that is E_NO_HANDLE above. A
                // code that no Fn can produce is advertised-but-inert, so it is gone from here and
                // from SPEC §11.24 (a) rather than left for a caller to branch on forever.
                case "E_NO_HANDLE":     wire = code; kind = "not_found";  return;       // reopen it
                case "E_PATH_NOT_FOUND": wire = code; kind = "not_found"; return;
                case "E_NO_SPEC_NODE":  wire = code; kind = "not_found";  return;       // try another kind

                default:
                    // An E_* code is taken at face value (that is how a Fn reports something more
                    // specific than the four kinds); anything else is a bug in a Fn.
                    wire = code != null && code.StartsWith("E_") ? code : "E_INTERNAL";
                    kind = "internal";
                    return;
            }
        }

        // ================================================================ the connection

        /// <summary>
        /// One client's request/response loop over one stream. Nested so it can reach this class's
        /// privates without widening the public surface; the CLI never sees it.
        /// </summary>
        internal sealed class Conn
        {
            readonly Stream _in, _out;
            readonly object _wlock = new object();
            readonly IDictionary<string, Fn> _map;

            internal Conn(Stream inS, Stream outS) { _in = inS; _out = outS; _map = Rpc.Fns; }

            /// <summary>Serves until EOF or `stop`. `stop` is the daemon's cue to exit: a
            /// sentence in a protocol that has no other way to say "go away" (SPEC §11.24 (f)).</summary>
            internal void Serve(out bool stop) {
                stop = false;
                var r = new LineReader(_in);
                for (;;) {
                    bool overflow, eof;
                    byte[] raw = r.ReadLine(out overflow, out eof);
                    if (overflow) {
                        WriteError(null, "E_BAD_REQUEST", "validation",
                            "请求行超过 " + MaxLine + " 字节上限，已丢弃到下一个换行（服务器继续）", null, 0);
                        if (eof) return;
                        continue;
                    }
                    if (raw == null) return;   // EOF
                    string text = Decode(raw);
                    if (text.Length > 0 && text[0] == '\uFEFF') text = text.Substring(1);
                    Handle(text, out stop);
                    if (stop) return;
                }
            }

            // ------------------------------------------------------------ dispatch

            void Handle(string text, out bool stop) {
                stop = false;
                var sw = Stopwatch.StartNew();

                string t = text.Trim();
                if (t.Length == 0) {
                    WriteError(null, "E_BAD_REQUEST", "validation", "空行不是请求（一行一个请求）", null, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                JObject req;
                try { req = JObject.Parse(t); }
                catch (Exception ex) {
                    // Malformed JSON. The line was fully consumed by the reader, so the stream is
                    // already framed correctly again -- no resync needed here.
                    WriteError(null, "E_BAD_REQUEST", "validation", "不是合法的 JSON 对象: " + ex.Message, null, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                JToken id = req["id"];
                string fn = Scalar(req["fn"]);
                if (string.IsNullOrEmpty(fn)) {
                    WriteError(id, "E_BAD_REQUEST", "validation", "请求缺少 fn", null, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                JToken argsTok = req["args"];
                JObject args;
                if (argsTok == null || argsTok.Type == JTokenType.Null) args = new JObject();
                else {
                    args = argsTok as JObject;
                    if (args == null) {
                        WriteError(id, "E_BAD_REQUEST", "validation", "args 必须是对象，收到 " + argsTok.Type, null, sw.Elapsed.TotalMilliseconds);
                        return;
                    }
                }

                // `stop` is transport-level, not a designer function: it belongs to the daemon
                // (SPEC §11.24 (f) 的 `tzs-cli stop`), so it is not in the manifest and is not
                // validated against it.
                if (fn == "stop") {
                    var res = new JObject();
                    res["stopping"] = true;
                    var o = new JObject();
                    o["id"] = id == null ? (JToken)JValue.CreateNull() : id;
                    o["ok"] = true;
                    o["result"] = res;
                    o["ms"] = Ms(sw.Elapsed.TotalMilliseconds);
                    WriteLine(o.ToString(Formatting.None));
                    stop = true;
                    return;
                }

                SpecFn spec = Manifest.Get(fn);
                if (spec == null && !_map.ContainsKey(fn)) {
                    // Declared nowhere and implemented nowhere.
                    var det = new JObject();
                    det["fn"] = fn;
                    WriteError(id, "E_UNKNOWN_METHOD", "validation", "未知函数: " + fn, det, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                if (spec != null) {
                    JObject problem = Manifest.Check(spec, args);
                    if (problem != null) {
                        WriteError(id, "E_BAD_PARAM", "validation", (string)problem["message"], problem, sw.Elapsed.TotalMilliseconds);
                        return;
                    }
                }

                Fn f;
                if (!_map.TryGetValue(fn, out f)) {
                    // A different answer from E_UNKNOWN_METHOD by design: the name is right, the
                    // body is not written yet (three Fns modules are in flight). Retrying will not
                    // help, so the kind is internal, not validation.
                    var det = new JObject();
                    det["fn"] = fn;
                    det["declared"] = spec != null;
                    WriteError(id, "E_NOT_IMPLEMENTED", "internal",
                        "函数已声明但尚未实现: " + fn, det, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                bool needsHandle = spec == null ? args["handle"] != null : spec.NeedsHandle;
                Session s = null;
                if (needsHandle) {
                    try { s = Resolve(args); }
                    catch (DetailedError de) {
                        string w, k; Map(de.Code, out w, out k);
                        WriteError(id, w, k, de.Message, de.Detail, sw.Elapsed.TotalMilliseconds);
                        return;
                    }
                }

                // Publish the in-flight request to the watchdog hook before entering the body --
                // the load, and therefore the timeout, happens inside it (SPEC §11.24 (h)). The id
                // is published too, so the fatal frame echoes the same value the answer would have
                // carried and the caller can match it.
                _inflight = this;
                _inflightId = id;

                // Two switches that cut across every write verb, both closed here rather than in
                // each body (Manifest.Traced puts them on the wire; SpecFn.Traced is the flag that
                // says which verbs those are). Both need the same thing: the outcome of a body
                // that really ran.
                //
                //   dry_run -- run it, keep the answer, put the model back (DryRun).
                //   op      -- the caller's name for this attempt, logged BEFORE the body so that
                //              a request that takes the daemon down still leaves "it started" behind
                //              (OpLog).
                //
                // Keyed off spec.Traced and NOT off "args has an `op`": `list_ops` takes an `op`
                // too, as a filter, and reading it as a name made the query log itself as an
                // operation (measured 2026-09-26 -- the first cut of this code did exactly that).
                bool traced = spec != null && spec.Traced;
                bool dry = traced && DryRun.Requested(args);
                string opName = traced ? Scalar(args["op"]) : null;
                if (opName != null && opName.Length == 0) opName = null;
                OpLog.Entry opEntry = opName == null ? null : OpLog.Begin(opName, fn,
                    s == null ? null : s.Handle, s == null ? null : s.Program, dry);

                object result = null;
                string errWire = null, errKind = null, errMsg = null;
                JObject errDetail = null;
                DryRun.Scope dryScope = null;
                try {
                    // The arming is INSIDE the same catch as the body on purpose: taking the
                    // dry-run snapshot writes a file, so it can fail for environmental reasons
                    // (read-only directory, full disk), and an exception thrown here used to
                    // escape the dispatcher entirely and take the daemon down with it -- a preview
                    // flag that can kill the server. Measured 2026-09-26. With nothing armed,
                    // DryRun.Exit below returns null and there is nothing to roll back, which is
                    // correct: the body never ran.
                    if (dry) {
                        DryRun.Enter();
                        // Only a mutating verb has anything to roll back. `save --dry-run` is the
                        // other half: it never dirties the model, and its one side effect (the
                        // file) is suppressed inside Save.Run. A verb that resolves its own session
                        // arms itself from inside the body -- DryRun.Arm's contract.
                        if (spec.Mutating) DryRun.Arm(s);
                    }
                    result = f(s, args);
                } catch (DetailedError de) {
                    Map(de.Code, out errWire, out errKind);
                    errMsg = de.Message; errDetail = de.Detail;
                } catch (TzsError te) {
                    Map(te.Code, out errWire, out errKind);
                    errMsg = te.Message;
                } catch (Exception ex) {
                    // Unwrap first. Every designer call goes through reflection, and both
                    // Activator.CreateInstance and MethodInfo.Invoke wrap the real failure in
                    // TargetInvocationException -- so the outer type/message is always the same
                    // useless pair and the cause is one or more levels down. Reporting the outer
                    // one is how a diagnosis costs an hour.
                    Exception inner = ex;
                    string chain = inner.GetType().Name + ": " + inner.Message;
                    while (inner.InnerException != null && inner.InnerException != inner) {
                        inner = inner.InnerException;
                        chain += "  <<  " + inner.GetType().Name + ": " + inner.Message;
                    }
                    var detail = new JObject();
                    detail["exception"] = inner.GetType().FullName;
                    detail["at"] = inner.TargetSite == null ? null : inner.TargetSite.Name;
                    errWire = "E_INTERNAL"; errKind = "internal"; errMsg = chain; errDetail = detail;
                } finally {
                    _inflight = null;
                    // The rollback belongs in the finally, and that is not tidiness: a body that
                    // threw half-way (E_ATTR_PARTIAL is the one that says so out loud) has already
                    // written part of the model, and a dry run that leaves THAT behind is worse
                    // than no dry run at all -- it is the write the caller asked not to make.
                    if (dry) dryScope = DryRun.Exit();
                    if (dryScope != null) dryScope.Rollback();
                }

                if (errWire != null) {
                    // The verb's own refusal, reported as itself -- with one addition: under a dry
                    // run the caller has to be able to see that nothing stuck. `reverted` is exactly
                    // the proof, and for a partial write it is the difference between "the model is
                    // half-changed" and "the model is as you left it".
                    if (dry) {
                        if (errDetail == null) errDetail = new JObject();
                        errDetail["dryRun"] = true;
                        if (dryScope != null) errDetail["reverted"] = DryRun.Reverted(dryScope);
                    }
                    OpLog.End(opEntry, "error", errWire, errMsg, sw.Elapsed.TotalMilliseconds, null);
                    WriteError(id, errWire, errKind, errMsg, errDetail, sw.Elapsed.TotalMilliseconds);
                    return;
                }

                if (dry) result = DryRun.Envelope(result, fn, s == null ? null : s.Handle, dryScope);
                OpLog.End(opEntry, "ok", null, null, sw.Elapsed.TotalMilliseconds, result);
                WriteOk(id, result, sw.Elapsed.TotalMilliseconds);
            }

            Session Resolve(JObject args) {
                JToken h = args["handle"];
                if (h == null || h.Type == JTokenType.Null)
                    throw new DetailedError("bad_param", "缺少 handle", Det("handle", "required", "handle 是必填参数"));
                if (h.Type != JTokenType.String)
                    throw new DetailedError("bad_param", "handle 必须是字符串，收到 " + h.Type,
                        Det("handle", "invalid", "handle 必须是字符串"));
                string hs = (string)h;
                Session s = FindByHandle(hs);
                if (s != null) return s;

                // Not a handle -- so it may be a MEANINGFUL key: the program name, or the
                // ProgramKey string ("aapp320|Form"). Accepting both is what lets a caller say
                // "the form of aapp320" instead of carrying an opaque perishable id around.
                List<Session> byKey = FindByKey(hs);
                if (byKey.Count == 1) return byKey[0];
                if (byKey.Count > 1) {
                    // The message a caller reads FIRST must carry the fix, not just the fact.
                    string ambMsg = "程序名 " + hs + " 对应 " + byKey.Count
                        + " 个开着的会话：请用 ProgramKey 形式指定（例如 " + hs + "|Form，见 candidates 里的 key）";
                    var amb = Det("handle", "ambiguous", ambMsg);
                    amb["candidates"] = OpenHandles();
                    throw new DetailedError("bad_param", ambMsg, amb);
                }

                string msg = "句柄不存在或已关闭: " + hs
                    + "（handle 也可以写程序名或 ProgramKey，例如 aapp320 或 aapp320|Form）";
                var d = Det("handle", "not_found", msg);
                d["candidates"] = OpenHandles();
                // E_NO_HANDLE rather than the generic E_NOT_FOUND: §11.24 (a) lists it for exactly
                // this ("句柄不存在或已关闭"), kind is still not_found, and the caller's next move
                // ("open it again") is then a branch rather than a guess.
                throw new DetailedError("E_NO_HANDLE", msg, d);
            }

            static JObject Det(string param, string reason, string message) {
                var o = new JObject();
                o["param"] = param;
                o["reason"] = reason;
                o["message"] = message;
                return o;
            }

            // ------------------------------------------------------------ framing

            void WriteOk(JToken id, object result, double ms) {
                var o = new JObject();
                o["id"] = id == null ? (JToken)JValue.CreateNull() : id;
                o["ok"] = true;
                o["result"] = result == null ? (JToken)JValue.CreateNull()
                           : (result is JToken ? (JToken)result : JToken.FromObject(result));
                o["ms"] = Ms(ms);
                WriteLine(o.ToString(Formatting.None));
            }

            /// <summary>`id` is echoed, not regenerated, and `ok` is the only discriminator
            /// (SPEC §11.24 (a)) -- a caller that has to look at `error` to know whether the call
            /// succeeded will get it wrong on the success path.
            ///
            /// `ms` is rendered through JRaw so the number is exactly what
            /// InvariantCulture produces: on this machine the ambient culture is zh-CN, and a
            /// double formatted with it would emit a decimal comma and produce invalid JSON.</summary>
            void WriteError(JToken id, string wire, string kind, string message, JObject detail, double ms) {
                var err = new JObject();
                err["code"] = wire;
                err["kind"] = kind;
                err["message"] = message;
                if (detail != null) err["detail"] = detail;
                var o = new JObject();
                o["id"] = id == null ? (JToken)JValue.CreateNull() : id;
                o["ok"] = false;
                o["error"] = err;
                o["ms"] = Ms(ms);
                WriteLine(o.ToString(Formatting.None));
            }

            static JToken Ms(double ms) {
                return new JRaw(ms.ToString("0.###", CultureInfo.InvariantCulture));
            }

            /// <summary>The fatal frame of SPEC §11.24 (h). Written from the watchdog thread while
            /// the main thread is parked inside the load, which is safe here because the main
            /// thread is not writing -- strictly serial means at most one writer at a time, and at
            /// this instant it is this one.
            ///
            /// kind is `designer`: the caller must not retry (the load is deterministic and will
            /// hang again) and the fix is upstream -- a table missing from mta/tables.xml
            /// (SPEC §11.14). kind `internal` would say "report a bug", which is also true, but
            /// the actionable instruction is the one worth carrying.</summary>
            internal void WriteFatal(string path) {
                int to = Session.TimeoutFromEnv();
                var det = new JObject();
                det["path"] = path;
                det["timeoutSeconds"] = to;
                var err = new JObject();
                err["code"] = "E_FATAL_LOAD_TIMEOUT";
                err["kind"] = "designer";
                err["message"] = "加载超时 " + to + "s（无消息泵的模态框），守护进程随后 exit(3)，本请求不会被重试: " + path;
                err["detail"] = det;
                var o = new JObject();
                o["id"] = InflightId();
                o["ok"] = false;
                o["error"] = err;
                WriteLine(o.ToString(Formatting.None));
            }

            /// <summary>The id of the request being served, for the fatal frame. It has to be the
            /// same value the response would have carried, or the caller cannot match the answer
            /// to the question.</summary>
            static JToken InflightId() { return _inflightId == null ? (JToken)JValue.CreateNull() : _inflightId; }

            internal void WriteLine(string s) {
                byte[] b = new UTF8Encoding(false).GetBytes(s + "\n");
                lock (_wlock) {
                    _out.Write(b, 0, b.Length);
                    _out.Flush();
                }
            }
        }

        /// <summary>The id of the in-flight request, set before the body runs so the watchdog hook
        /// can echo it. Only ever one, because the transport is strictly serial.</summary>
        static JToken _inflightId;

        // ================================================================ line framing

        static string Decode(byte[] b) { return Encoding.UTF8.GetString(b); }

        /// <summary>
        /// stderr as UTF-8 bytes, for the same reason stdout is (SPEC §11.20): Console.Error runs
        /// through the OEM code page (GBK here) even when the stream is redirected and
        /// Console.OutputEncoding does not apply to a redirected stream, so every Chinese
        /// diagnostic would arrive as bytes the caller cannot decode. Not disposed -- it wraps a
        /// standard handle that outlives us.
        /// </summary>
        public static void Err(string msg) {
            try {
                byte[] b = new UTF8Encoding(false).GetBytes(msg + "\n");
                Stream s = Console.OpenStandardError();
                s.Write(b, 0, b.Length);
                s.Flush();
            } catch { }
        }

        /// <summary>
        /// Reads one \n-terminated line from a stream without assuming it can buffer the whole
        /// thing, and without assuming the stream is well behaved. Two properties the protocol
        /// needs and a ReadLine() cannot give:
        ///
        ///   * a hard cap. Past MaxLine the line is abandoned and bytes are discarded up to the
        ///     next \n, so the *next* request is framed correctly again -- the contract says the
        ///     server must resynchronise rather than die (SPEC §11.24 (f)).
        ///   * EOF is reported separately from "an empty line arrived", so a daemon that went away
        ///     is distinguishable from a caller that sent nothing.
        /// </summary>
        public sealed class LineReader {
            readonly Stream _s;
            readonly byte[] _buf = new byte[16384];
            int _len, _pos;
            readonly int _max;

            public LineReader(Stream s) : this(s, MaxLine) { }
            public LineReader(Stream s, int max) { _s = s; _max = max; }

            public byte[] ReadLine(out bool overflow, out bool eof) {
                overflow = false; eof = false;
                var acc = new MemoryStream();
                for (;;) {
                    if (_pos >= _len) {
                        int n;
                        try { n = _s.Read(_buf, 0, _buf.Length); }
                        catch (IOException) { eof = true; return acc.Length == 0 ? null : Finish(acc); }
                        if (n <= 0) {
                            eof = true;
                            return acc.Length == 0 ? null : Finish(acc);
                        }
                        _len = n; _pos = 0;
                    }
                    int start = _pos;
                    while (_pos < _len && _buf[_pos] != (byte)'\n') _pos++;
                    acc.Write(_buf, start, _pos - start);
                    if (_pos < _len) {
                        _pos++;                       // consume the terminator
                        if (acc.Length > _max) { overflow = true; return null; }
                        return Finish(acc);
                    }
                    if (acc.Length > _max) {
                        if (!SkipToNewline()) eof = true;
                        overflow = true;
                        return null;
                    }
                }
            }

            /// <summary>Discards input through the next \n. Returns false at EOF.</summary>
            bool SkipToNewline() {
                for (;;) {
                    if (_pos >= _len) {
                        int n;
                        try { n = _s.Read(_buf, 0, _buf.Length); }
                        catch (IOException) { return false; }
                        if (n <= 0) return false;
                        _len = n; _pos = 0;
                    }
                    while (_pos < _len && _buf[_pos] != (byte)'\n') _pos++;
                    if (_pos < _len) { _pos++; return true; }
                }
            }

            static byte[] Finish(MemoryStream acc) {
                byte[] b = acc.ToArray();
                // \r\n: strip the \r so the frame is the same in either convention.
                if (b.Length > 0 && b[b.Length - 1] == (byte)'\r') {
                    byte[] t = new byte[b.Length - 1];
                    Array.Copy(b, t, t.Length);
                    return t;
                }
                return b;
            }
        }
    }
}
