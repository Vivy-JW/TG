using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Globalization;
using System.IO;
using System.IO.Pipes;
using System.Reflection;
using System.Text;
using System.Threading;
using TzsCli.Designer;

// SPEC §11.24 (f) 的用户界面. Same AssemblyVersion rule as tzs-server.cs: this exe is a
// GetEntryAssembly() the designer may inspect, and the version gate compares it against mta/ver.
[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// tzs-cli -- the user-facing half. One request per process, over the daemon's pipe.
///
/// The whole point of the daemon is that this process stays small: P0 measured ~1000 ms per
/// one-shot process (about 890 ms of it fixed initialisation unrelated to the form) against
/// ~0 ms for an operation inside a warm process (SPEC §11.24 (f)). So this program never boots
/// the designer -- it connects, sends one line, reads one line, prints it, exits.
///
/// Connection policy: try the pipe; if nothing answers, spawn tzs-server --daemon and retry once
/// (SPEC §11.24 (f): 连不上就 spawn 一个再重试，首次约 1 s，之后 ~0 ms). `stop` deliberately does
/// not spawn -- "stop" that starts a server would be a joke at the caller's expense.
///
/// Failure classification, which is the part SPEC §11.24 (h) is emphatic about:
///
///   a frame arrives with error.code E_FATAL_LOAD_TIMEOUT, then EOF
///       -> the load timed out, the watchdog wrote the frame and exited 3. Deterministic and not
///          retryable, so this request is NOT retried. Exit 2 to make it scriptable.
///   EOF with no frame at all
///       -> E_SERVER_DIED. Synthesised locally, exit 3. The daemon is restarted by the *next*
///          command, not by this one.
///   a line arrives that is not a JSON object
///       -> also E_SERVER_DIED, never a parse error: a parser message ("unexpected token") reads
///          like a fact about the form, and an AI would conclude the element does not exist.
///
/// Exit codes: 0 = a frame came back (ok true or false), 1 = the frame said ok:false,
///             2 = E_FATAL_LOAD_TIMEOUT, 3 = transport failure, 4 = usage error.
/// </summary>
class TzsCliMain
{
    // TZSCLI_INSTALL overrides the bundled path -- see Designer.Install and tzs-server.cs.
    static readonly string INSTALL = InstallFromEnv();

    static string InstallFromEnv() {
        string v = Environment.GetEnvironmentVariable("TZSCLI_INSTALL");
        if (!string.IsNullOrEmpty(v)) return v;
        // 随仓库自带的那份：build.sh 把 engine\designer\ 采到 out\designer\，
        // 与发行包 <引擎目录>\designer\ 是同一个布局。没有硬编码缺省 —— 见 engine/BUILD.md。
        return Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "designer");
    }

    /// <summary>How long to wait for an already-running daemon before deciding there is none.
    /// Small on purpose: a warm daemon answers in about a millisecond, so anything longer only
    /// delays the spawn path.</summary>
    const int CONNECT_TRY_MS = 300;

    /// <summary>Cold start budget. Boot is ~900 ms plus the first load; 60 s leaves room for a
    /// slow disk without hiding a genuine hang -- and a genuine load hang is what the watchdog
    /// inside Session.Open is for, not this.</summary>
    const int SPAWN_WAIT_S = 60;

    /// <summary>Read timeout. validate is 10.4 s on a 670-element form (SPEC §11.24 (c)) and a
    /// load can legitimately take 90 s, so this is generous; it exists so a wedged daemon is a
    /// message rather than a hung shell.</summary>
    const int READ_TIMEOUT_MS = 300000;

    [STAThread]
    static int Main(string[] args) {
        AppDomain.CurrentDomain.AssemblyResolve += Resolve;

        if (args.Length == 0) { RawErr(Usage()); return 4; }

        string cmd = args[0];
        switch (cmd) {
            case "--help": case "-h": case "help":
                RawOut(Rpc.Help());
                return 0;

            case "--manifest":
                RawOut(Rpc.ManifestJson() + "\n");
                return 0;

            case "stop":
                return DoStop();

            case "open": {
                if (args.Length < 2) { RawErr("用法: tzs-cli open <file> [--workspace <dir>]\n"); return 4; }
                List<KeyValuePair<string, string>> kv = Flags(args, 2);
                string path;
                try { path = Path.GetFullPath(args[1]); }
                catch (Exception ex) { RawErr("tzs-cli: 路径不合法: " + ex.Message + "\n"); return 4; }
                kv.Insert(0, new KeyValuePair<string, string>("path", path));
                return Do("open", ArgsJson("open", kv), _ws);
            }

            case "save":
                return Do("save", ArgsJson("save", Flags(args, 1)), null);

            case "call": {
                if (args.Length < 2) { RawErr("用法: tzs-cli call <fn> [--<参数> <值> ...]\n"); return 4; }
                string fn = args[1];
                return Do(fn, ArgsJson(fn, Flags(args, 2)), null);
            }

            default:
                RawErr("tzs-cli: 未知子命令 " + cmd + "\n\n" + Usage());
                return 4;
        }
    }

    // ---------------------------------------------------------------- request side

    /// <summary>Set by Flags() when it sees --workspace, which is a daemon-side option rather than
    /// an argument to the function.</summary>
    static string _ws;

    /// <summary>Collects --name value / --name=value pairs. A bare --flag becomes true so that
    /// boolean parameters (force, excluded, cited) read naturally.</summary>
    static List<KeyValuePair<string, string>> Flags(string[] args, int start) {
        var kv = new List<KeyValuePair<string, string>>();
        for (int i = start; i < args.Length; i++) {
            string a = args[i];
            if (!a.StartsWith("--")) { Note("tzs-cli: 忽略非选项参数 " + a); continue; }
            string name = a.Substring(2), val = "true";
            int eq = name.IndexOf('=');
            if (eq >= 0) { val = name.Substring(eq + 1); name = name.Substring(0, eq); }
            else if (i + 1 < args.Length && !args[i + 1].StartsWith("--")) { val = args[++i]; }

            if (name == "workspace") { _ws = val; continue; }   // a daemon-side option
            kv.Add(new KeyValuePair<string, string>(name, val));
        }
        return kv;
    }

    /// <summary>Renders the args object. Value typing comes from the manifest, so `--offset 2`
    /// becomes the number 2 and `--paths a,b` becomes an array rather than a string that the
    /// server would then have to reinterpret. Everything JSON stays in Rpc/Json (the library):
    /// this program never names a Newtonsoft type, which is what lets build.sh's `ref` mode
    /// compile it without a -r: on the install dir.</summary>
    static string ArgsJson(string fn, List<KeyValuePair<string, string>> kv) {
        SpecFn decl = Manifest.Get(fn);
        var sb = new StringBuilder("{");
        for (int i = 0; i < kv.Count; i++) {
            if (i > 0) sb.Append(',');
            sb.Append(Json.J(kv[i].Key)).Append(':').Append(ValueJson(Param(decl, kv[i].Key), kv[i].Value));
        }
        sb.Append('}');
        return sb.ToString();
    }

    static Param Param(SpecFn f, string name) {
        if (f == null || f.Params == null) return null;
        foreach (Param p in f.Params) if (p.Name == name) return p;
        return null;
    }

    static string ValueJson(Param p, string v) {
        if (p != null) {
            switch (p.Type) {
                case PType.Int: {
                    long n;
                    if (long.TryParse(v, NumberStyles.Integer, CultureInfo.InvariantCulture, out n))
                        return n.ToString(CultureInfo.InvariantCulture);
                    break;
                }
                case PType.Bool:
                    if (v == "true" || v == "1" || v == "yes") return "true";
                    if (v == "false" || v == "0" || v == "no") return "false";
                    break;
                case PType.PathList:
                case PType.StrList: {
                    var sb = new StringBuilder("[");
                    string[] parts = v.Length == 0 ? new string[0] : v.Split(',');
                    for (int i = 0; i < parts.Length; i++) {
                        if (i > 0) sb.Append(',');
                        sb.Append(Json.J(parts[i].Trim()));
                    }
                    sb.Append(']');
                    return sb.ToString();
                }
            }
        }
        return Json.J(v);
    }

    // ---------------------------------------------------------------- transport

    static int Do(string fn, string argsJson, string workspace) {
        var t0 = Stopwatch.StartNew();
        // Before anything else: the pipe name is derived from the workspace, and this process
        // never Boots, so Designer.Workspace is empty here. Without the hint the client hashes
        // "" and looks for a pipe the daemon it spawns is not listening on.
        Rpc.WorkspaceHint = workspace;
        NamedPipeClientStream pipe = Rpc.Connect(CONNECT_TRY_MS);
        bool cold = pipe == null;
        if (cold) {
            Note("tzs-cli: 管道 \\\\.\\pipe\\" + Rpc.PipeName + " 无应答，启动 tzs-server ...");
            if (!Spawn(workspace)) {
                Died(1, "启动失败：找不到 tzs-server.exe（应与 tzs-cli.exe 同目录）");
                return 3;
            }
            var sw = Stopwatch.StartNew();
            while (sw.Elapsed.TotalSeconds < SPAWN_WAIT_S) {
                pipe = Rpc.Connect(500);
                if (pipe != null) break;
                Thread.Sleep(50);
            }
            if (pipe == null) {
                Died(1, "守护进程 " + SPAWN_WAIT_S + "s 内没有就绪");
                return 3;
            }
            Note("tzs-cli: 守护进程就绪，冷启动 " + sw.ElapsedMilliseconds + " ms");
        }
        long tConnect = t0.ElapsedMilliseconds;

        var total = Stopwatch.StartNew();
        try {
            string req = Rpc.Request(1, fn, argsJson);
            long tRequest = t0.ElapsedMilliseconds;

            // PipeStream refuses ReadTimeout ("Timeouts are not supported on this stream"), so the
            // timeout is enforced here: the exchange runs on a short-lived background thread and
            // the caller waits on an event. The thread is not a correctness problem on this side
            // -- it is the *server* whose strictly-serial guarantee forbids worker threads -- and
            // it leaks harmlessly because the process exits immediately after a timeout.
            Xfer x = Exchange(pipe, req, READ_TIMEOUT_MS);
            if (x.TimedOut) {
                Died(1, "守护进程无响应（读取超时 " + (READ_TIMEOUT_MS / 1000) + " s）");
                return 3;
            }
            if (x.Error != null) {
                Died(1, "连接中断: " + x.Error.GetType().Name + ": " + x.Error.Message);
                return 3;
            }
            if (x.Overflow) { Died(1, "应答超过 8 MB 上限"); return 3; }
            if (x.Line == null) {
                // EOF with no frame: the daemon died before answering. This is the case that must
                // never be reported as a missing element.
                Died(1, "守护进程在没有应答的情况下退出（EOF，无帧）");
                return 3;
            }

            Reply r = Rpc.Classify(x.Line);
            if (r.ParseFailed) {
                Died(1, "应答不是 JSON 对象（已按 E_SERVER_DIED 处理）: " + Clip(x.Line));
                return 3;
            }
            long tDone = t0.ElapsedMilliseconds;

            if (Environment.GetEnvironmentVariable("TZSCLI_DEBUG") == "1") {
                // Where this process's wall time actually went. The number that matters is the
                // server-side `ms` in the frame: 0.0x ms means the daemon answered out of a warm
                // designer, which is the entire point of the architecture. Everything above is
                // .NET Framework process startup, paid by every one-shot CLI process.
                double age = -1;
                try { age = (DateTime.Now - Process.GetCurrentProcess().StartTime).TotalMilliseconds; } catch { }
                Note("tzs-cli: [debug] 进程年龄=" + age.ToString("0", CultureInfo.InvariantCulture)
                     + "ms connect=" + (cold ? "冷启动" : (tConnect + "ms"))
                     + " request=" + (tRequest - tConnect) + "ms exchange=" + (tDone - tRequest) + "ms");
            }

            total.Stop();
            RawOut(x.Line + "\n");

            if (r.Ok) {
                if (!string.IsNullOrEmpty(r.ResultHandle))
                    Note("tzs-cli: handle = " + r.ResultHandle);
                Note("tzs-cli: ok，往返 " + total.ElapsedMilliseconds + " ms");
                return 0;
            }

            if (r.Code == "E_FATAL_LOAD_TIMEOUT") {
                // The frame arrived, so the answer is authoritative -- print it (done above) and
                // do NOT retry: the load is deterministic and will hang again (SPEC §11.24 (h) 3).
                Note("tzs-cli: 加载超时（守护进程已 exit 3）；本请求不重试，下次调用会重启守护进程");
                return 2;
            }

            Note("tzs-cli: " + r.Code + " (" + r.Kind + "): " + r.Message);
            return 1;
        } finally {
            try { pipe.Dispose(); } catch { }
        }
    }

    /// <summary>tzs-cli stop. Never spawns: asking a daemon to stop should not start one.</summary>
    static int DoStop() {
        NamedPipeClientStream pipe = Rpc.Connect(CONNECT_TRY_MS);
        if (pipe == null) {
            Note("tzs-cli: 没有运行中的守护进程（无需停止）");
            return 0;
        }
        try {
            Xfer x = Exchange(pipe, Rpc.Request(1, "stop", "{}"), 10000);
            if (x.TimedOut) { Note("tzs-cli: stop 超时"); return 1; }
            if (x.Line == null) { Note("tzs-cli: 守护进程已退出（未回 stop 应答）"); return 0; }
            RawOut(x.Line + "\n");
            Note("tzs-cli: 守护进程已停止");
            return 0;
        } catch (Exception ex) {
            Note("tzs-cli: stop 失败: " + ex.Message);
            return 1;
        } finally {
            try { pipe.Dispose(); } catch { }
        }
    }

    /// <summary>The result of one exchange, carried out of the worker thread by value rather than
    /// through a captured `out` parameter.</summary>
    sealed class Xfer {
        public string Line;
        public bool Eof, Overflow, TimedOut;
        public Exception Error;
        public ManualResetEvent Done = new ManualResetEvent(false);
    }

    /// <summary>Writes one request and reads one response, with a wall-clock bound.</summary>
    static Xfer Exchange(Stream pipe, string req, int timeoutMs) {
        Xfer x = new Xfer();
        var t = new Thread(delegate() {
            try { x.Line = Rpc.Exchange(pipe, req, timeoutMs, out x.Eof, out x.Overflow); }
            catch (Exception ex) { x.Error = ex; }
            finally { try { x.Done.Set(); } catch { } }
        });
        t.IsBackground = true;
        t.Start();
        if (!x.Done.WaitOne(timeoutMs)) x.TimedOut = true;
        return x;
    }

    /// <summary>Spawns the daemon with --daemon explicit. Explicit matters: if this process's own
    /// stdin is redirected (a script, a CI runner), the child would inherit that and auto-select
    /// stdio mode, and the pipe would never appear.
    ///
    /// The child's standard handles are the trap here, and it is not theoretical: Windows'
    /// CreateProcess duplicates every *inheritable* handle into the child, so a daemon started
    /// from inside `subprocess.run(capture_output=True)` keeps a copy of the caller's capture
    /// pipe. The caller then never sees EOF on that pipe, and it hangs for the whole lifetime of
    /// the daemon -- long past the point where the frames were printed. So all three of the
    /// child's standard handles are redirected (the daemon's wire is the pipe; anything it prints
    /// is diagnostic, and both of its output streams are drained into our stderr), and every
    /// inheritable handle this process holds is sealed around Process.Start.
    ///
    /// Sealing *every* handle rather than just GetStdHandle(STD_*) is the part that matters, and
    /// it is not belt-and-braces. A .NET Framework process holds a second, inheritable handle to
    /// its own stdout -- a duplicate the runtime makes for itself before Main runs; measured, not
    /// assumed (see SealInheritance). GetStdHandle(-11) does not name it, so clearing the flag on
    /// the standard handles leaves it inheritable, the daemon inherits it, and the caller's stdout
    /// capture still never returns EOF -- which is exactly the reported symptom: the previous
    /// "clear stdin/stdout/stderr" fix looked right and changed nothing.</summary>
    static bool Spawn(string workspace) {
        string dir = Path.GetDirectoryName(Assembly.GetEntryAssembly().Location);
        string exe = Path.Combine(dir, "tzs-server.exe");
        if (!File.Exists(exe)) { Note("tzs-cli: 期望的路径: " + exe); return false; }

        var psi = new ProcessStartInfo(exe, "--daemon");
        psi.UseShellExecute = false;
        psi.CreateNoWindow = true;
        psi.RedirectStandardInput = true;
        psi.RedirectStandardOutput = true;
        // stderr too. Leaving it to inherit is what hangs a caller that captures the stream: the
        // daemon outlives us, so it goes on holding the write end of the caller's stderr capture
        // pipe and the caller's read-until-EOF blocks forever. It also has to be redirected for a
        // second, quieter reason: with the flag cleared but the handle still passed in
        // STARTUPINFO, the child's STD_ERROR_HANDLE is a value nothing guaranteed is valid *in
        // the child*, and could land on one of the redirection pipes -- so the daemon's
        // diagnostics would corrupt its own wire. A redirect gives the child a stderr that is
        // unambiguously its own.
        //
        // The cost is that daemon diagnostics are only relayed while this process lives, rather
        // than staying on the caller's stderr afterwards. That is the same trade stdout already
        // makes, and it is the right one: the alternative is a daemon that can hang its caller.
        psi.RedirectStandardError = true;
        // The relay decodes the child's bytes, and Process's default for a redirected stream is
        // Console.OutputEncoding -- the OEM code page (GBK here), which is the exact trap
        // SPEC §11.20 calls out for our own output. The daemon writes UTF-8 (Rpc.Err), so without
        // this every Chinese diagnostic arrives as mojibake. It is set for stdout as well as
        // stderr: both are relayed the same way, and stdout was wrong in the same way already.
        var utf8 = new UTF8Encoding(false);
        psi.StandardOutputEncoding = utf8;
        psi.StandardErrorEncoding = utf8;
        if (!string.IsNullOrEmpty(workspace)) psi.EnvironmentVariables["TZSCLI_WS"] = workspace;

        Process p;
        List<int> sealedHandles = SealInheritance();
        try { p = Process.Start(psi); }
        catch (Exception ex) { Note("tzs-cli: Process.Start 失败: " + ex.Message); return false; }
        finally { UnsealInheritance(sealedHandles); }
        if (p == null) return false;

        Drain(p.StandardOutput, "tzs-server| ");
        Drain(p.StandardError, "tzs-server! ");
        return true;
    }

    /// <summary>Relays one of the child's output streams onto our stderr. The daemon's stdout and
    /// stderr both carry diagnostics -- its wire is the pipe -- so this is what keeps cold-start
    /// messages ("listening on \\.\pipe\...", "module X not found") in front of the user. After
    /// this process exits its end of the pipe closes and the daemon's writes fail with
    /// ERROR_BROKEN_PIPE rather than blocking, which is what makes it safe to leave the thread
    /// behind: a write end with no reader is an error, not a stall.</summary>
    static void Drain(StreamReader s, string prefix) {
        var t = new Thread(delegate() {
            try {
                string l;
                while ((l = s.ReadLine()) != null) Note(prefix + l);
            } catch { }
        });
        t.IsBackground = true;
        t.Start();
    }

    // ---------------------------------------------------------------- handle inheritance

    const uint HANDLE_FLAG_INHERIT = 0x1;

    /// <summary>One past the highest handle value worth looking at. Handle table indices are
    /// 4-aligned and the allocator hands out the lowest free one, so a process this short-lived
    /// never grows anywhere near it; that is 16384 GetHandleInformation calls, measured at 3.9 ms
    /// on this machine -- noise against a ~750 ms cold start. There is no public "enumerate my
    /// handles" API, and NtQuerySystemInformation would mean a machine-wide buffer and a privilege
    /// question for what a bounded scan answers outright.</summary>
    const int HANDLE_SCAN_LIMIT = 0x10000;

    [System.Runtime.InteropServices.DllImport("kernel32.dll", SetLastError = true)]
    static extern bool GetHandleInformation(IntPtr hObject, out uint lpdwFlags);

    [System.Runtime.InteropServices.DllImport("kernel32.dll", SetLastError = true)]
    static extern bool SetHandleInformation(IntPtr hObject, uint dwMask, uint dwFlags);

    /// <summary>
    /// Clears HANDLE_FLAG_INHERIT on every inheritable handle this process holds, so that
    /// CreateProcess (which duplicates *every* inheritable handle) hands the daemon nothing but
    /// the redirection pipes .NET creates inside Process.Start. Returns the values it changed so
    /// UnsealInheritance can put them back.
    ///
    /// Why not GetStdHandle: measured in this process under `capture_output=True`, the inheritable
    /// set at entry to Main is {stdin, stdout, stderr} plus a *second* pipe handle that is a
    /// duplicate of stdout -- writing a marker byte into it lands on the caller's stdout, which is
    /// how it was identified. It is created by the runtime before Main runs and GetStdHandle
    /// cannot name it, so only a sweep of the handle table reaches it. With stdout cleared but
    /// that duplicate left inheritable, the daemon still holds the caller's stdout pipe and the
    /// capture never EOFs: the fix that only cleared STD_* looked correct and reproduced the bug
    /// unchanged.
    ///
    /// Clearing the flag is the right lever rather than closing the handle: the duplicate is
    /// presumably what the runtime writes our own stdout through, so closing it would break our
    /// output, while clearing the flag costs nothing here -- nothing this process owns after the
    /// swap is something the child should have.
    /// </summary>
    static List<int> SealInheritance() {
        var sealedNow = new List<int>();
        for (int v = 4; v < HANDLE_SCAN_LIMIT; v += 4) {
            IntPtr h = new IntPtr(v);
            uint flags;
            if (!GetHandleInformation(h, out flags)) continue;
            if ((flags & HANDLE_FLAG_INHERIT) == 0) continue;
            if (SetHandleInformation(h, HANDLE_FLAG_INHERIT, 0)) sealedNow.Add(v);
        }
        return sealedNow;
    }

    static void UnsealInheritance(List<int> sealedNow) {
        if (sealedNow == null) return;
        foreach (int v in sealedNow)
            try { SetHandleInformation(new IntPtr(v), HANDLE_FLAG_INHERIT, HANDLE_FLAG_INHERIT); } catch { }
    }

    // ---------------------------------------------------------------- output

    /// <summary>A transport-level failure, rendered as a frame so the caller always receives
    /// something well formed to switch on (SPEC §11.24 (h)). Synthesised locally by design: the
    /// answer is about the transport, so it cannot have come from the server.
    ///
    /// kind is `internal`: unlike E_FATAL_LOAD_TIMEOUT this says nothing about the form, and the
    /// retry the wrapper performs is a property of the wrapper, not an instruction to the AI.</summary>
    static void Died(int id, string message) {
        RawOut("{\"id\":" + id.ToString(CultureInfo.InvariantCulture) +
               ",\"ok\":false,\"error\":{\"code\":\"E_SERVER_DIED\",\"kind\":\"internal\",\"message\":" +
               Json.J(message) + "}}\n");
        Note("tzs-cli: E_SERVER_DIED: " + message);
    }

    static string Clip(string s) { return s.Length <= 200 ? s : s.Substring(0, 200) + "..."; }

    static void RawOut(string s) { Raw(Console.OpenStandardOutput(), s); }

    static void RawErr(string s) { Raw(Console.OpenStandardError(), s); }

    /// <summary>A diagnostic line on stderr, as UTF-8 bytes (SPEC §11.20). Everything the user
    /// sees about progress, timings and failures goes here; stdout stays protocol-only.</summary>
    static void Note(string s) { RawErr(s + "\n"); }

    /// <summary>UTF-8 bytes straight to the handle. Console.Out/Error go through the OEM code page
    /// (GBK here), which turns every non-ASCII byte into something no JSON reader accepts
    /// (SPEC §11.20).
    ///
    /// A closed stdout is tolerated: `tzs-cli call ... | head -1` sends EPIPE as soon as head has
    /// what it wants, and dying with an unhandled IOException there would turn a well-formed
    /// frame into a stack trace on stderr.</summary>
    static void Raw(Stream s, string text) {
        try {
            byte[] b = new UTF8Encoding(false).GetBytes(text);
            s.Write(b, 0, b.Length);
            s.Flush();
        } catch (IOException) { }
        catch (ObjectDisposedException) { }
    }

    static Assembly Resolve(object sender, ResolveEventArgs e) {
        string name = new AssemblyName(e.Name).Name + ".dll";
        string here = Path.GetDirectoryName(Assembly.GetEntryAssembly().Location);
        foreach (string dir in new[] { here, INSTALL }) {
            string p = Path.Combine(dir, name);
            if (File.Exists(p)) { try { return Assembly.LoadFrom(p); } catch { } }
        }
        return null;
    }

    static string Usage() {
        return "tzs-cli -- T100 .tzs 设计器的长驻 JSON-RPC 客户端\n\n" +
               "用法:\n" +
               "  tzs-cli open <file> [--workspace <dir>]\n" +
               "  tzs-cli call <fn> [--<参数> <值> ...]\n" +
               "  tzs-cli save --handle <h> --out <file>\n" +
               "  tzs-cli stop\n" +
               "  tzs-cli --help | --manifest\n\n" +
               "连接 \\\\.\\pipe\\" + Rpc.PipeName + "；连不上就启动 tzs-server 再重试一次。\n" +
               "退出码: 0 有应答 / 1 应答为 ok:false / 2 E_FATAL_LOAD_TIMEOUT / 3 传输失败 / 4 用法错\n" +
               "函数清单: tzs-cli --help\n";
    }
}
