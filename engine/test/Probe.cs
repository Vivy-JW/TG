using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Reflection;
using System.Text;
using System.Threading;
using System.Windows;
using System.Windows.Markup;

[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// P0 probes. Three things must be settled before any function layer is designed, and all
/// three are yes/no questions about the designer's own code rather than design choices:
///
///   1. Can one long-lived process hold several packages and interleave operations on them?
///      Every tool so far has done exactly one operation per process; a JSON-RPC server
///      cannot work that way. Related: how much of the 1.0s per call is load and how much
///      is per-operation, i.e. what the server actually saves.
///
///   2. Can the designer's own validators be triggered and their results collected? They are
///      private BackgroundWorkers that publish DocumentErrorsEvent -- and SaveSettingEvent
///      already runs them on every save, which means our existing RoundTrip has been running
///      them all along and discarding the output.
///
///   3. Which write path applies the layout-attribute rules? The XmlElement indexer enforces
///      a whitelist-by-presence plus repeat/step gating and MinGrid clamping, and only then
///      invokes FormAttributesUndoRedoCommand. Calling that command directly skips all of it
///      -- which is exactly the class of silent corruption our `set` already exhibits on the
///      spec side.
///
/// Everything here is in-memory; nothing is written back to the workspace.
/// </summary>
class Probe
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
    static readonly string WS =
        Environment.GetEnvironmentVariable("TZSCLI_WS") ?? @"D:\t100_wrok_dir\hengshuo\prd";

    static Assembly A, FE;
    static object SM;

    // ---------------------------------------------------------------- helpers

    static object Call(object t, string n, params object[] a) {
        Type tt = t as Type ?? t.GetType();
        MethodInfo loose = null;
        foreach (var m in tt.GetMethods(BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static)) {
            if (m.Name != n || m.GetParameters().Length != a.Length) continue;
            if (loose == null) loose = m;
            var ps = m.GetParameters();
            bool fits = true;
            for (int i = 0; i < ps.Length; i++) {
                if (a[i] == null) continue;
                if (!ps[i].ParameterType.IsAssignableFrom(a[i].GetType())) { fits = false; break; }
            }
            if (!fits) continue;
            return Invoke(m, t, a);
        }
        if (loose != null) return Invoke(loose, t, a);
        throw new Exception("no " + n + "/" + a.Length + " on " + tt);
    }
    static object Invoke(MethodInfo m, object t, object[] a) {
        try { return m.Invoke(t is Type ? null : t, a); }
        catch (TargetInvocationException ex) {
            System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw();
            return null;
        }
    }
    static object Prop(object o, string n) {
        if (o == null) return null;
        var t = o.GetType();
        var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (p != null) return p.GetValue(o, null);
        var f = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        return f == null ? null : f.GetValue(o);
    }
    static object Prop2(Type t, string n) {
        var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
        if (p != null) return p.GetValue(null, null);
        var f = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
        return f == null ? null : f.GetValue(null);
    }
    static void SetProp(object o, string n, object v) {
        var t = o.GetType();
        var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (p != null) { p.SetValue(o, v, null); return; }
        t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance).SetValue(o, v);
    }
    static Type Find(Assembly a, string name) {
        foreach (var t in a.GetTypes()) if (t.Name == name) return t;
        return null;
    }
    static string S(object o) { return o == null ? "(null)" : o.ToString(); }
    static string Q(object o) { return o == null ? "<absent>" : "<" + o + ">"; }

    // ---------------------------------------------------------------- package loading

    class Pkg {
        public string Path, Program;
        public object Tzp, Si, Key;
        public double LoadMs;
        public override string ToString() { return Program + " (" + Path + ")"; }
    }

    /// <summary>The designer raises a modal dialog from SpecificationInfo's constructor when a
    /// table the form references is missing from mta/tables.xml, and headless there is no
    /// message pump, so it hangs forever with nothing on stdout. Same watchdog as AddField.</summary>
    static Pkg LoadPackage(string tzs, int seconds) {
        var pkg = new Pkg { Path = tzs };
        var done = new ManualResetEvent(false);
        var watch = new Thread(delegate() {
            if (done.WaitOne(seconds * 1000)) return;
            Console.WriteLine("!! 加载超时 " + seconds + "s: " + tzs);
            Environment.Exit(3);
        });
        watch.IsBackground = true;
        watch.Start();
        var sw = Stopwatch.StartNew();
        try {
            object t = Activator.CreateInstance(A.GetType("SpecDesignerCommon.TzpManager"), new object[] { tzs });
            object k = Prop(t, "ProgramKey");
            var map = (IDictionary)Prop(SM, "tzpMap");
            if (map.Contains(k)) map.Remove(k);
            map.Add(k, t);
            var eam = Find(A, "EventAggregatorManager");
            if (!(bool)Call(eam, "ContainsKey", k)) Call(eam, "CreateInstance", k);
            object s = Call(A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", t);
            SetProp(t, "SpecificationInfo", s);
            pkg.Tzp = t; pkg.Si = s; pkg.Key = k;
            pkg.Program = S(Prop(t, "ProgramName"));
        } finally { done.Set(); sw.Stop(); pkg.LoadMs = sw.Elapsed.TotalMilliseconds; }
        return pkg;
    }

    /// <summary>Every mutation needs an UndoRedoManager registered for that key; the designer's
    /// SetInitGridX/Y throw if one is present during *load*, so it has to be added afterwards.</summary>
    static void RegisterUndoRedo(Pkg p) {
        var urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
        object urm = Activator.CreateInstance(Find(urf, "UndoRedoManager"), new object[] { 100 });
        Call(urm, "Init");
        ((IDictionary)Prop(SM, "undoRedoManagerMap"))[p.Key] = urm;
    }

    /// <summary>Walk the form model to a name-path, the same addressing the other tools use.</summary>
    static object FindByPath(Pkg p, string path) {
        string[] parts = path.Split('/');
        object form = Prop(p.Si, "FormNode");
        if (form == null || parts.Length < 2) return null;
        if (S(Prop(form, "Name")) != parts[1]) return null;
        return Walk(form, parts[0] + "/" + parts[1], path);
    }
    static object Walk(object el, string here, string target) {
        if (here == target) return el;
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return null;
        foreach (var c in nodes) {
            object nv = Prop(c, "Name");
            string seg = nv == null ? null : nv.ToString();
            if (string.IsNullOrEmpty(seg)) seg = S(Prop(c, "NodeName"));
            object hit = Walk(c, here + "/" + seg, target);
            if (hit != null) return hit;
        }
        return null;
    }
    static void CollectAll(object el, string here, List<string> paths, List<object> els) {
        paths.Add(here); els.Add(el);
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return;
        foreach (var c in nodes) {
            object nv = Prop(c, "Name");
            string seg = nv == null ? null : nv.ToString();
            if (string.IsNullOrEmpty(seg)) seg = S(Prop(c, "NodeName"));
            CollectAll(c, here + "/" + seg, paths, els);
        }
    }

    // ================================================================ probe 1

    static void Probe1MultiPackage(Pkg a, Pkg b) {
        Console.WriteLine("========== 探针 1：长驻进程 / 多包 ==========");
        Console.WriteLine("  [A] " + a.Program + "  加载 " + a.LoadMs.ToString("F0") + " ms");
        int aCount = CountAll(a);
        Console.WriteLine("      A 元素数 " + aCount);
        Console.WriteLine("  [B] " + b.Program + "  加载 " + b.LoadMs.ToString("F0") + " ms");

        // Both must be live in the same tzpMap at once -- that is the whole question.
        var map = (IDictionary)Prop(SM, "tzpMap");
        Console.WriteLine("  tzpMap 同时持有 " + map.Count + " 个包");

        // Re-read A after B was loaded and mutated; a global TzpManager.Current would bite here.
        int aCount2 = CountAll(a);
        Console.WriteLine("  B 加载后重读 A：元素数 " + aCount2 + (aCount2 == aCount ? "  (一致)" : "  !! 变了"));

        // Now mutate both, interleaved, and confirm each keeps its own state.
        RegisterUndoRedo(a); RegisterUndoRedo(b);
        string pathA = FirstWidgetPath(a), pathB = FirstWidgetPath(b);
        Console.WriteLine("  A 目标: " + pathA);
        Console.WriteLine("  B 目标: " + pathB);

        string attr = Environment.GetEnvironmentVariable("TZSCLI_PROBE_ATTR") ?? "case";
        string vA = SetViaIndexer(a, pathA, attr, "upper");
        string vB = SetViaIndexer(b, pathB, attr, "lower");
        Console.WriteLine("  A " + attr + " -> " + vA);
        Console.WriteLine("  B " + attr + " -> " + vB);

        // Cross-check: A must still report its own value, not B's.
        object elA = FindByPath(a, pathA), elB = FindByPath(b, pathB);
        string nowA = S(Call(elA, "GetAttribute", attr)), nowB = S(Call(elB, "GetAttribute", attr));
        Console.WriteLine("  交错写后：A=" + Q(nowA) + "  B=" + Q(nowB)
            + ((nowA == vA && nowB == vB) ? "   两包互不串扰 ✓" : "   !! 串扰"));

        // And A's own model must still be addressable by name after all that.
        Console.WriteLine("  A 仍可按路径寻址: " + (FindByPath(a, pathA) != null));

        // TzpManager.Current is a static that GetTzpManger sets as a side effect. If anything
        // resolves through it instead of through the key, two packages would cross-talk.
        object cur = Prop2(A.GetType("SpecDesignerCommon.TzpManager"), "Current");
        Console.WriteLine("  TzpManager.Current 指向: " + (cur == null ? "<null>" : S(Prop(cur, "ProgramName")))
            + "   (B 是 " + b.Program + "，A 是 " + a.Program + ")");

        // What does a server actually save? Load is paid once; per-op cost is the rest.
        var sw = Stopwatch.StartNew();
        for (int i = 0; i < 5; i++) {
            SetViaIndexer(a, pathA, attr, i % 2 == 0 ? "upper" : "lower");
        }
        sw.Stop();
        Console.WriteLine("  5 次内存操作共 " + sw.Elapsed.TotalMilliseconds.ToString("F0") + " ms"
            + "  → 单次约 " + (sw.Elapsed.TotalMilliseconds / 5).ToString("F1") + " ms");
        Console.WriteLine("  对照：单独启动一次进程约 1000 ms（其中约 890 ms 是与表单无关的固定初始化）");
        Console.WriteLine();
    }

    static int CountAll(Pkg p) {
        var paths = new List<string>(); var els = new List<object>();
        object form = Prop(p.Si, "FormNode");
        if (form == null) return 0;
        CollectAll(form, "x", paths, els);
        return els.Count;
    }

    /// <summary>Root element name of the .4fd, which is the first segment of every name-path.</summary>
    static string RootName(Pkg p) {
        byte[] zip = File.ReadAllBytes(p.Path);
        string fd = EntryText(zip, EntryName(zip, ".4fd"));
        if (fd == null) return "";
        int i = fd.IndexOf('<');
        if (i < 0) return "";
        int j = fd.IndexOf("name=\"", i);
        if (j < 0) return "";
        int k = fd.IndexOf('"', j + 6);
        return k < 0 ? "" : fd.Substring(j + 6, k - j - 6);
    }

    /// <summary>First element carrying a `case` attribute -- Buttons/Edits have one, containers
    /// do not, so this lands on a real widget rather than a Grid.</summary>
    static string FirstWidgetPath(Pkg p) {
        object form = Prop(p.Si, "FormNode");
        if (form == null) return null;
        var paths = new List<string>(); var els = new List<object>();
        CollectAll(form, RootName(p) + "/" + S(Prop(form, "Name")), paths, els);
        for (int i = 0; i < els.Count; i++)
            if (!string.IsNullOrEmpty((string)Call(els[i], "GetAttribute", "case"))) return paths[i];
        return paths.Count > 1 ? paths[1] : null;
    }

    // ================================================================ probe 3

    /// <summary>The XmlElement indexer -- the UI's own write path, so it carries the rules.</summary>
    static PropertyInfo Indexer(object el) {
        foreach (var p in el.GetType().GetProperties(BindingFlags.Public|BindingFlags.Instance)) {
            if (p.Name != "Item") continue;
            var ix = p.GetIndexParameters();
            if (ix.Length == 1 && ix[0].ParameterType == typeof(string) && p.CanWrite) return p;
        }
        return null;
    }
    static string SetViaIndexer(Pkg p, string path, string attr, string val) {
        object el = FindByPath(p, path);
        if (el == null) return "<路径不存在>";
        var pi = Indexer(el);
        if (pi == null) return "<没有索引器>";
        try {
            pi.SetValue(el, val, new object[] { attr });
        } catch (Exception ex) {
            return "<抛异常: " + (ex.InnerException ?? ex).Message + ">";
        }
        return S(Call(el, "GetAttribute", attr));
    }
    static string SetViaCommand(Pkg p, string path, string attr, string val) {
        object el = FindByPath(p, path);
        if (el == null) return "<路径不存在>";
        Type ct = Find(A, "FormAttributesUndoRedoCommand");
        object cmd = Activator.CreateInstance(ct, new object[] { el, attr, val });
        Call(cmd, "Execute");
        return S(Call(el, "GetAttribute", attr));
    }

    static void Probe3LayoutAttr(Pkg p) {
        Console.WriteLine("========== 探针 3：布局属性写路径 ==========");
        string path = FirstWidgetPath(p);
        Console.WriteLine("  目标 " + path);
        RegisterUndoRedo(p);

        object el = FindByPath(p, path);
        var attrs = new List<string>();
        var names = Prop(el, "Attributes") as IEnumerable;
        if (names != null) foreach (var n in names) attrs.Add(n.ToString());
        Console.WriteLine("  该元素的属性（" + attrs.Count + " 个）: " + string.Join(" ", attrs.Take(14)) + " ...");

        // (a) an attribute that exists -- the ordinary case. Flip the value rather than
        //     assigning a constant: the indexer short-circuits when old == new, and Probe 1
        //     already wrote `case` on this very element, so a fixed value proves nothing.
        string existing = attrs.Contains("case") ? "case" : (attrs.Contains("tag") ? "tag" : attrs[0]);
        string before = S(Call(el, "GetAttribute", existing));
        string wanted = before == "upper" ? "lower" : "upper";
        Console.WriteLine();
        Console.WriteLine("  (a) 索引器写【已存在】属性 " + existing + "=" + Q(before) + " → " + wanted);
        string r1 = SetViaIndexer(p, path, existing, wanted);
        Console.WriteLine("      结果 " + Q(r1) + (r1 == wanted ? "   ✓ 生效" : "   !! 没变"));

        // (b) an attribute this element does not have -- the UI cannot create attributes,
        //     only change ones that already exist. If the indexer enforces that, a name
        //     absent from the element must be refused.
        Console.WriteLine();
        Console.WriteLine("  (b) 索引器写【不存在】属性 zzz_not_an_attr → probe");
        string r2 = SetViaIndexer(p, path, "zzz_not_an_attr", "probe");
        Console.WriteLine("      结果 " + Q(r2) + (r2 == "(null)" || r2 == "" || r2 == "<absent>"
            ? "   ✓ 被拒绝（白名单生效）" : "   !! 居然写进去了"));

        // (c) the same write through the command class the indexer delegates to -- this is
        //     what a naive wrapper would call, and it must be shown to behave differently.
        Console.WriteLine();
        Console.WriteLine("  (c) 直接调 FormAttributesUndoRedoCommand 写同一个不存在的属性");
        string r3 = SetViaCommand(p, path, "zzz_not_an_attr2", "probe");
        Console.WriteLine("      结果 " + Q(r3) + (r3 == "probe"
            ? "   !! 绕过了白名单（证实：不能直接用这个类）" : "   也被拒绝?"));

        // (d) clamping: gridWidth is floored at MinGridWidth. Our first element has
        //     MinGridWidth 1, which cannot be floored, so scan for one that can.
        Console.WriteLine();
        string cPath = null; int cMin = 0; object cEl = null;
        {
            object form = Prop(p.Si, "FormNode");
            var ps = new List<string>(); var es = new List<object>();
            CollectAll(form, RootName(p) + "/" + S(Prop(form, "Name")), ps, es);
            for (int i = 0; i < es.Count; i++) {
                object mw = Prop(es[i], "MinGridWidth");
                int v = 0;
                try { v = mw == null ? 0 : (int)mw; } catch { v = 0; }
                if (v > 1) { cPath = ps[i]; cMin = v; cEl = es[i]; break; }
            }
        }
        if (cPath == null) {
            Console.WriteLine("  (d) 夹紧：全表单没有 MinGridWidth>1 的元素，此项未验证");
        } else {
            string g0 = S(Call(cEl, "GetAttribute", "gridWidth"));
            string r4 = SetViaIndexer(p, cPath, "gridWidth", "1");
            Console.WriteLine("  (d) 夹紧：" + cPath.Substring(cPath.Length > 40 ? cPath.Length - 40 : 0)
                + "  gridWidth " + Q(g0) + " → 1   MinGridWidth=" + cMin);
            Console.WriteLine("      结果 " + Q(r4)
                + (r4 != "1" ? "   ✓ 被夹到 " + r4 : "   !! 没夹"));
        }

        // (e) a rule that throws rather than clamps: repeat != true inside a ScrollGrid.
        Console.WriteLine();
        Console.WriteLine("  (e) 规则型拒绝（repeat 在 ScrollGrid 内必须为 true）—— 语料里 ScrollGrid 只有 1 处，");
        Console.WriteLine("      此处只验证索引器确实会走 gating 分支，不强行构造该场景。");
        Console.WriteLine();
    }

    // ================================================================ probe 2

    static readonly List<string> Errors = new List<string>();
    static object _evt;

    static void SubscribeErrors() {
        object eam = Prop2(Find(A, "EventAggregatorManager"), "Global");
        if (eam == null) { Console.WriteLine("  !! EventAggregatorManager.Global 取不到"); return; }
        Type evtType = A.GetType("SpecDesignerCommon.Events.DocumentErrorsEvent");
        MethodInfo ge = null;
        foreach (var m in eam.GetType().GetMethods())
            if (m.Name == "GetEvent" && m.IsGenericMethodDefinition && m.GetParameters().Length == 0) { ge = m; break; }
        if (ge == null) { Console.WriteLine("  !! 找不到 GetEvent<T>()"); return; }
        _evt = ge.MakeGenericMethod(evtType).Invoke(eam, null);
        Type argType = evtType.BaseType.GetGenericArguments()[0];
        typeof(Probe).GetMethod("SubscribeVia", BindingFlags.Static|BindingFlags.NonPublic)
            .MakeGenericMethod(argType).Invoke(null, new object[] { _evt });
        Console.WriteLine("  已订阅 " + evtType.Name + "（载荷 " + argType.Name + "）");
    }
    static void SubscribeVia<T>(object evt) {
        Action<T> h = delegate(T a) {
            lock (Errors) Errors.Add(S(Prop(a, "ErrorType")) + "  " + S(Prop(a, "Key")) + " — " + S(Prop(a, "Description")));
        };
        var m = evt.GetType().GetMethod("Subscribe", new Type[] { typeof(Action<T>) });
        if (m == null) throw new Exception("没有 Subscribe(Action<T>) on " + evt.GetType());
        m.Invoke(evt, new object[] { h });
    }

    /// <summary>Both validators are private BackgroundWorkers on SpecificationInfo, and they
    /// read this.FormElement / this.TSDElement -- ctor-time snapshots with private setters.
    /// SaveToForm/SaveToTSD are what re-assign those from the live model, so without them the
    /// validator inspects the file as loaded and every edit is invisible. (That is also why
    /// RoundTrip, which calls SaveToForm/SaveToTSD but never the workers, has never validated
    /// anything: the workers only run from the SaveSettingEvent subscriber.)</summary>
    static List<string> RunValidate(Pkg p, string label) {
        lock (Errors) Errors.Clear();
        Call(p.Si, "SaveToForm");
        Call(p.Si, "SaveToTSD");
        Call(p.Si, "InitTSDValidateWorker");
        Call(p.Si, "InitFormValidateWorker");
        object tsdv = Prop(p.Si, "_TSDValidater");
        object fmv  = Prop(p.Si, "_FormValidater");
        Console.WriteLine("  校验器字段：" + (tsdv != null) + " / " + (fmv != null));
        var sw = Stopwatch.StartNew();
        while (sw.ElapsedMilliseconds < 30000) {
            bool busy = (tsdv != null && (bool)Prop(tsdv, "IsBusy")) || (fmv != null && (bool)Prop(fmv, "IsBusy"));
            if (!busy) break;
            Thread.Sleep(100);
        }
        Thread.Sleep(600);                       // events are published on the worker thread
        List<string> got;
        lock (Errors) got = new List<string>(Errors);
        Console.WriteLine("  [" + label + "] 等待 " + sw.Elapsed.TotalMilliseconds.ToString("F0")
            + " ms，收到 " + got.Count + " 条");
        foreach (string e in got.Take(8)) Console.WriteLine("      " + e);
        if (got.Count > 8) Console.WriteLine("      ... 另有 " + (got.Count - 8) + " 条");
        return got;
    }

    static void Probe2Validate(Pkg p) {
        Console.WriteLine("========== 探针 2：设计器自己的校验器 ==========");
        SubscribeErrors();

        // A BackgroundWorker swallowing an exception looks exactly like "no errors found",
        // so check the two things the validators dereference before trusting a silent run.
        object xsd = Prop(SM, "Info_TsdXsd");
        Console.WriteLine("  Info_TsdXsd = " + (xsd == null
            ? "<null>  ← ValidateTsd 的 XSD 那一段会失效" : "已加载"));
        string res;
        try { res = S(Call(Application.Current, "FindResource", "Message_DuplicateFieldName")); }
        catch (Exception ex) { res = "<取不到: " + ex.Message + ">"; }
        Console.WriteLine("  Message_DuplicateFieldName = " + Q(res));

        List<string> clean = RunValidate(p, "原始");

        // Deliberately break the invariant validateForm names explicitly:
        // Message_DuplicateFieldName -- two elements under <Form> sharing a `name`.
        // (Must be two DESCENDANTS: the check walks Form.Descendants() and Form itself is
        // not among them, so renaming something to the form's own name creates no clash.)
        var paths = new List<string>(); var els = new List<object>();
        object form = Prop(p.Si, "FormNode");
        CollectAll(form, "x", paths, els);
        var named = new List<object>();
        foreach (var e in els) {
            string n = S(Prop(e, "Name"));
            if (string.IsNullOrEmpty(n)) continue;
            string tag = S(Prop(e, "NodeName"));
            if (tag == "Item") continue;      // the check skips <Item>
            if (tag == "Form") continue;      // and Form itself is not in Form.Descendants()
            named.Add(e);
        }
        if (named.Count >= 2) {
            string victim = S(Prop(named[1], "Name"));
            string thief  = S(Prop(named[0], "Name"));
            Call(named[1], "SetAttribute", "name", thief);
            Console.WriteLine();
            Console.WriteLine("  人为破坏：把 " + Q(victim) + " 的 name 改成 " + Q(thief) + "（重名）");
            List<string> broken = RunValidate(p, "破坏后");
            Call(named[1], "SetAttribute", "name", victim);
            Console.WriteLine("  → 校验器 " + (broken.Count > clean.Count
                ? "抓到了 ✓（" + broken.Count + " > " + clean.Count + "）"
                : "没抓到（" + broken.Count + " vs " + clean.Count + "）"));
        } else {
            Console.WriteLine("  找不到两个具名元素来构造重名，跳过破坏测试");
        }
        Console.WriteLine();
    }

    // ================================================================ probe 4

    /// <summary>ValidateForm is not a dead preference after all: it is the switch in front of
    /// XmlElement.CheckOverlapping, which runs on every attribute write during load. Our tools
    /// have always set it false, which silently disables that check. This asks whether turning
    /// it ON works headlessly -- if it does, `validate` gets overlap detection for free.</summary>
    static void Probe4ValidateFormFlag(Pkg p) {
        Console.WriteLine("========== 探针 4：ValidateForm 开关（重叠检测）==========");
        object pm = Prop2(Find(A, "PreferenceManager"), "Current");
        object prefs = Prop(pm, "_preferenceModel");
        if (prefs == null) prefs = Prop(pm, "Settings");   // the public accessor CheckOverlapping uses
        Console.WriteLine("  PreferenceManager.Current.Settings = "
            + (prefs == null ? "<null>  ← 加载必崩" : prefs.GetType().Name));
        if (prefs == null) { Console.WriteLine(); return; }

        string path = FirstWidgetPath(p);
        object el = FindByPath(p, path);
        Console.WriteLine("  目标 " + path + "  gridWidth=" + Q(S(Call(el, "GetAttribute", "gridWidth"))));

        SetProp(prefs, "ValidateForm", false);
        string off = null;
        try { off = SetViaIndexer(p, path, "gridWidth", "3"); }
        catch (Exception ex) { off = "<异常: " + (ex.InnerException ?? ex).Message + ">"; }
        Console.WriteLine("  ValidateForm=false  gridWidth→3  结果 " + Q(off));

        SetProp(prefs, "ValidateForm", true);
        string on = null, err = null;
        try { on = SetViaIndexer(p, path, "gridWidth", "4"); }
        catch (Exception ex) { err = (ex.InnerException ?? ex).GetType().Name + ": " + (ex.InnerException ?? ex).Message; }
        Console.WriteLine("  ValidateForm=true   gridWidth→4  结果 " + Q(on)
            + (err == null ? "" : "   异常 → " + err));
        Console.WriteLine("  → " + (on == "4"
            ? "重叠检测可无界面运行 ✓（`validate` 可以吃满设计器的规则）"
            : "开启后不可用，`validate` 只能拿到 XSD + 字段级那部分"));

        // IsPosFine is what drives the red "position wrong" border in the designer; if the
        // overlap pass runs, it is the observable output.
        Console.WriteLine("  IsPosFine=" + Q(S(Prop(el, "IsPosFine")))
            + "  MinGridWidth=" + Q(S(Prop(el, "MinGridWidth"))));
        SetProp(prefs, "ValidateForm", false);
        Console.WriteLine();
    }

    // ================================================================ probe 5

    /// <summary>Probe 3 showed the indexer mutates the element, but the thing that matters is
    /// whether the change reaches the document. SaveToForm() regenerates FormElement from the
    /// live model -- the same call the designer's own save path makes -- so anything missing
    /// from its output was never really written.</summary>
    static void Probe5WriteReachesOutput(Pkg p) {
        Console.WriteLine("========== 探针 5：索引器写入是否到达输出 ==========");
        string path = FirstWidgetPath(p);
        object el = FindByPath(p, path);
        string name = S(Prop(el, "Name"));
        string attr = "case";
        string orig = S(Call(el, "GetAttribute", attr));
        string wanted = orig == "upper" ? "lower" : "upper";
        Console.WriteLine("  元素 " + Q(name) + "  " + attr + "=" + Q(orig) + " → " + wanted);

        SetViaIndexer(p, path, attr, wanted);
        Console.WriteLine("  索引器后读回: " + Q(S(Call(el, "GetAttribute", attr))));

        object fd = Call(p.Si, "SaveToForm");
        var x = fd as System.Xml.Linq.XElement;
        if (x == null) { Console.WriteLine("  !! SaveToForm 没返回 XElement"); Console.WriteLine(); return; }

        // A name match alone is not enough: the first hit is the <RecordField> inside
        // <Record>, which has no `case` at all. Report every same-named element instead, so
        // the layout one can be told apart from the record one.
        int shown = 0; string found = null;
        foreach (var d in x.Descendants()) {
            if ((string)d.Attribute("name") != name) continue;
            string v = (string)d.Attribute(attr);
            if (shown < 4) Console.WriteLine("      重生成里的 <" + d.Name.LocalName + " name=\"" + name + "\"> "
                + attr + "=" + (v == null ? "<无此属性>" : "<" + v + ">"));
            shown++;
            if (v != null) found = v;
        }
        Console.WriteLine("  同名元素共 " + shown + " 处");
        Console.WriteLine("  → " + (found == wanted
            ? "✓ 写入到达输出（保存路径认这个改动）"
            : "!! 没到达"));
        Console.WriteLine("  重生成文档 " + x.ToString().Length + " 字符");
        Console.WriteLine();
    }

    // ---------------------------------------------------------------- plumbing

    static string EntryName(byte[] zip, string suffix) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
            foreach (var e in z.Entries) if (e.FullName.EndsWith(suffix)) return e.FullName;
        return null;
    }
    static string EntryText(byte[] zip, string name) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
            return r.ReadToEnd();
    }

    [STAThread]
    static void Main(string[] args) {
        if (args.Length < 1) {
            Console.WriteLine("Probe.exe <fileA.tzs> [fileB.tzs]");
            return;
        }
        string fileA = args[0];
        string fileB = args.Length > 1 ? args[1] : fileA;

        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };
        var app = new Application();
        DesignerLang.Merge(app, INSTALL);
        A  = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        FE = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesigner.FormEditor.dll"));
        SM = Call(A.GetType("SpecDesignerCommon.SettingManager"), "Get");
        object mdl = Call(A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel"), "Create");
        SetProp(SM, "CurrentSetting", mdl);
        SetProp(Prop(mdl, "Connection"), "Workspace", WS);
        Call(SM, "LoadCommonData");

        // Without this, PreferenceManager.Current.Settings is null and the very first
        // SetAttribute during load NREs inside XmlElement.CheckOverlapping. Note WHY that
        // matters: CheckOverlapping reads PreferenceManager.Current.Settings.ValidateForm
        // (XmlElement.cs:2522) -- so ValidateForm is not a dead preference, it is the switch
        // that turns overlap detection on and off for every attribute write during load.
        object prefs = Activator.CreateInstance(Find(A, "PreferenceModel"));
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(Find(A, "PreferenceManager"), "Current"), "_preferenceModel", prefs);

        int tmo = 90;
        int.TryParse(Environment.GetEnvironmentVariable("TZSCLI_RELOAD_TIMEOUT") ?? "90", out tmo);

        try {
            // Load both up front and reuse them: reloading a key whose UndoRedoManager is
            // registered throws "illeagal call SetInitGridX/Y", so the load/operate phases
            // must not interleave for the same key.
            var a = LoadPackage(fileA, tmo);
            var b = LoadPackage(fileB, tmo);
            Probe1MultiPackage(a, b);
            Probe3LayoutAttr(a);
            Probe4ValidateFormFlag(a);
            Probe5WriteReachesOutput(a);
            Probe2Validate(a);
        } catch (Exception ex) {
            var inner = ex;
            while (inner.InnerException != null) inner = inner.InnerException;
            Console.WriteLine();
            Console.WriteLine("!! 探针抛异常：" + inner.GetType().Name + ": " + inner.Message);
            Console.WriteLine(inner.StackTrace);
            Environment.Exit(4);
        }
        Console.WriteLine("========== 探针结束 ==========");
    }
}
