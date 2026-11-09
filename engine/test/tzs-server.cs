using System;
using System.IO;
using System.Reflection;
using System.Text;
using System.Threading;
using TzsCli.Designer;

// SPEC §11.24 (f): the long-lived half. One process, one Boot, then serve.
//
// The AssemblyVersion is not decoration: Designer.Boot -> SettingManager.LoadCommonData compares
// mta/ver against Assembly.GetEntryAssembly().GetName().Version and throws
// VersionIncompatibleException on a mismatch (SPEC §7 版本门禁). It belongs in the exe and
// nowhere else -- two copies in one assembly is CS0579 -- which is why it is here and not in the
// library (SPEC 纪律 5).
[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// tzs-server -- the daemon.
///
/// Two modes, one protocol:
///
///   --stdio    reads stdin, writes frames to stdout. The acceptance harness drives this
///              (`echo '{"id":1,"fn":"list_open"}' | tzs-server --stdio`) and so can any test
///              that would rather not own a pipe. Bare `tzs-server` falls into it automatically
///              when stdin is redirected, so the obvious shell one-liner works.
///   --daemon   listens on \\.\pipe\tzs-cli. The shipping mode, and the one tzs-cli uses.
///
/// Why both: SPEC §11.24 (f) makes the pipe the wire precisely so that the ported designer code
/// -- which writes to Console freely -- cannot corrupt a frame. Stdio mode keeps stdin/stdout as
/// the wire, so it is a test/diagnostic mode, and Rpc.Stdio redirects Console.Out to stderr to
/// keep the common case honest.
///
/// Boot happens once, before any request, because SettingManager.LoadCommonData binds the whole
/// process to one workspace: a package outside it is refused by TzpManager
/// (NotInCurrentWorkspaceException), and re-pointing Connection.Workspace is not something any
/// tool in this repo has survived (Bootstrap.Boot is deliberately idempotent and will not redo
/// it). A caller that needs another module's workspace points TZSCLI_WS (or --workspace) at it and
/// restarts the daemon -- which `tzs-cli stop` plus any call does on its own.
/// </summary>
class TzsServerMain
{
    // TZSCLI_INSTALL overrides the bundled path -- see Designer.Install. These probes build
    // into engine/out/, which is not a package and carries no designer, so they fall back to
    // an installed designer. Not `const` because it comes from the environment.
    static readonly string INSTALL = InstallFromEnv();

    static string InstallFromEnv() {
        string v = Environment.GetEnvironmentVariable("TZSCLI_INSTALL");
        if (!string.IsNullOrEmpty(v)) return v;
        // 随仓库自带的那份：build.sh 把 engine\designer\ 采到 out\designer\，
        // 与发行包 <引擎目录>\designer\ 是同一个布局。没有硬编码缺省 —— 见 engine/BUILD.md。
        return Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "designer");
    }

    /// <summary>The same default the other programs in this repo use (test/Edit.cs:49,
    /// test/Probe.cs:43, test/RoundTrip.cs:34), so a daemon started by hand behaves like a
    /// one-shot tool started by hand.</summary>
    // Aliased, not re-typed: the pipe name is derived from the workspace, so a client and a
    // server that disagree about the default would compute different pipe names and never meet.
    const string WS_DEFAULT = Rpc.DefaultWorkspace;

    [STAThread]
    static int Main(string[] args) {
        // One handler for everything the designer loads by name: its own siblings, Prism and
        // Newtonsoft.Json all live in INSTALL and are not next to our exe.
        AppDomain.CurrentDomain.AssemblyResolve += Resolve;

        bool stdio = false, daemon = false, pipeName = false;
        string ws = Environment.GetEnvironmentVariable("TZSCLI_WS");

        for (int i = 0; i < args.Length; i++) {
            switch (args[i]) {
                case "--stdio":  stdio = true; break;
                case "--daemon": daemon = true; break;
                case "--pipe-name": pipeName = true; break;
                case "--manifest":
                    RawOut(Rpc.ManifestJson() + "\n");
                    return 0;
                case "--help":
                case "-h":
                    RawOut(Usage());
                    return 0;
                case "--workspace":
                    if (i + 1 >= args.Length) { Rpc.Err("tzs-server: --workspace 需要参数"); return 4; }
                    ws = args[++i];
                    break;
                default:
                    Rpc.Err("tzs-server: 未知参数 " + args[i]);
                    Rpc.Err(Usage());
                    return 4;
            }
        }

        if (string.IsNullOrEmpty(ws)) ws = WS_DEFAULT;

        if (pipeName) {
            // Print the name THIS build would listen on, so a host process can ASK instead of
            // re-deriving it. Both ingredients are hostile to reimplementation: the hash is
            // int32-wrapping over UTF-16 code units, and the MVID lives in the assembly metadata.
            // Getting either wrong fails silently -- the client connects to a pipe nobody listens
            // on, concludes "no daemon", spawns one, and still cannot reach it.
            //
            // No Boot: this needs only the workspace and the MVID, and Boot costs ~900 ms.
            // Setting the hint covers `--workspace`, which the environment alone would not.
            Rpc.WorkspaceHint = ws;
            RawOut(Rpc.PipeName + "\n");
            return 0;
        }
        if (!stdio && !daemon) {
            // Redirected stdin means someone is piping requests in, i.e. they want the one-shot
            // form. tzs-cli always passes --daemon explicitly, so this can never hijack it.
            bool redirected = false;
            try { redirected = Console.IsInputRedirected; } catch { }
            stdio = redirected;
        }

        Rpc.Err("tzs-server: 引导 " + ws + "  (模式: " + (stdio ? "stdio" : "daemon") + ")");
        try {
            Designer.Boot(ws);
        } catch (Exception ex) {
            Rpc.Err("tzs-server: 引导失败: " + ex.GetType().Name + ": " + ex.Message);
            return 1;
        }
        Rpc.Err("tzs-server: 引导完成");

        return stdio ? Rpc.Stdio() : Rpc.Daemon();
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

    /// <summary>Writes through the raw stdout handle as UTF-8 bytes: Console.Out goes through the
    /// OEM code page (GBK here), which mangles every non-ASCII byte no JSON reader will accept
    /// (SPEC §11.20).</summary>
    static void RawOut(string s) {
        byte[] b = new UTF8Encoding(false).GetBytes(s);
        Stream o = Console.OpenStandardOutput();
        o.Write(b, 0, b.Length);
        o.Flush();
    }

    static string Usage() {
        return
            "tzs-server -- T100 .tzs 设计器的长驻 JSON-RPC 服务\n\n" +
            "用法:\n" +
            "  tzs-server [--daemon | --stdio] [--workspace <dir>]\n" +
            "  tzs-server --manifest       打函数表 JSON 后退出\n" +
            "  tzs-server --pipe-name      打本构建会监听的管道名后退出\n" +
            "  tzs-server --help\n\n" +
            "  --daemon   监听 \\\\.\\pipe\\" + Rpc.PipeName + "（默认；tzs-cli 用这个）\n" +
            "  --stdio    从 stdin 读请求、往 stdout 写应答\n" +
            "             未指定模式且 stdin 被重定向时自动选它\n" +
            "  --pipe-name 管道名随工作区与本程序集的 MVID 变化，所以由本程序回答；\n" +
            "             宿主进程（tt）应当问它，不要自己重算。不 Boot。\n\n" +
            "工作区取 TZSCLI_WS 环境变量，默认 " + WS_DEFAULT + "\n" +
            "  Boot 只做一次，整个进程绑定一个工作区；换工作区请 stop 后另起。\n" +
            "设计器目录取 TZSCLI_INSTALL 环境变量，默认 " + INSTALL + "\n\n" +
            "协议: 一行一个请求 {\"id\":n,\"fn\":\"...\",\"args\":{...}}\n" +
            "      一行一个应答 {\"id\":n,\"ok\":true,\"result\":{...},\"ms\":0.5}\n" +
            "                   {\"id\":n,\"ok\":false,\"error\":{\"code\":\"E_*\",\"kind\":\"*\",\"message\":\"...\"}}\n" +
            "      函数表见 --manifest 或 tzs-cli --help\n";
    }
}
