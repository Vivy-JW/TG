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
using System.Xml.Linq;

[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// W0.4 probe. Wave 0 item §11.24(g) asks one question before Fn/Session.cs can be written:
/// TzpManager.CreateKey (TzpManager.cs:229) builds the key from program name + package type
/// ONLY -- not the path -- so four corpus files collide on one PackageKey. The existing
/// LoadPackage (Probe.cs / Edit.cs) resolves that with `map.Remove(k); map.Add(k, t);`, which
/// silently orphans the previous handle while its UndoRedoManager stays registered. A
/// long-lived server cannot do that.
///
/// So: can a key be RELEASED and re-opened? Three tests, one process, in-memory only.
///
///   1. open(A) -> mutate (which forces UndoRedoManager registration, the state the P0 probe
///      showed "cannot reload" while it is set) -> close(A) -> open(A) again. Does the reload
///      throw "illeagal call SetInitGridX/Y"? Does it read the file, or the mutated memory?
///
///   2. open(fileA) + open(fileB) where both share a ProgramKey. Which one survives, what does
///      tzpMap.Count say, and does close() resolve the collision?
///
///   3. after a close/reopen cycle, are the process-global singletons still usable?
///
/// Every result line that is a judgement starts with VERDICT so it can be grepped.
/// All output is ASCII: Console.WriteLine goes through the OEM code page and is mojibake in a pipe.
/// </summary>
class ProbeReopen
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

    // ---------------------------------------------------------------- helpers (from Probe.cs)

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
    static string ExText(Exception ex) {
        var inner = ex;
        while (inner.InnerException != null) inner = inner.InnerException;
        return inner.GetType().Name + ": " + inner.Message;
    }

    static IDictionary TzpMap() { return (IDictionary)Prop(SM, "tzpMap"); }
    static IDictionary UrmMap() { return (IDictionary)Prop(SM, "undoRedoManagerMap"); }
    static Type Eam() { return Find(A, "EventAggregatorManager"); }
    static bool EamHas(object key) { return (bool)Call(Eam(), "ContainsKey", key); }
    static object Cur() { return Prop2(A.GetType("SpecDesignerCommon.TzpManager"), "Current"); }

    // ---------------------------------------------------------------- package loading

    class Pkg {
        public string Path, Program;
        public object Tzp, Si, Key;
        public double LoadMs;
        /// <summary>True when this load found the key already occupied -- i.e. it collided.</summary>
        public bool Collided;
        /// <summary>The TzpManager that was silently evicted by this load (null when no collision).</summary>
        public object Replaced;
        public string ReplacedPath;
        public override string ToString() { return Program + " (" + Path + ")"; }
    }

    /// <summary>The designer raises a modal dialog from SpecificationInfo's constructor when a
    /// table the form references is missing from mta/tables.xml, and headless there is no
    /// message pump, so it hangs forever with nothing on stdout. Same watchdog as Probe.cs.</summary>
    static Pkg LoadPackage(string tzs, int seconds) {
        var pkg = new Pkg { Path = tzs };
        var done = new ManualResetEvent(false);
        var watch = new Thread(delegate() {
            if (done.WaitOne(seconds * 1000)) return;
            Console.WriteLine("!! LOAD TIMEOUT after " + seconds + "s: " + tzs);
            Environment.Exit(3);
        });
        watch.IsBackground = true;
        watch.Start();
        var sw = Stopwatch.StartNew();
        try {
            object t = Activator.CreateInstance(A.GetType("SpecDesignerCommon.TzpManager"), new object[] { tzs });
            object k = Prop(t, "ProgramKey");
            var map = TzpMap();
            // LoadPackage in Probe.cs / Edit.cs evicts silently. Record what it evicted so the
            // collision is observable -- the eviction itself is the behaviour under test.
            if (map.Contains(k)) {
                pkg.Collided = true;
                pkg.Replaced = map[k];
                pkg.ReplacedPath = S(Prop(pkg.Replaced, "ZipFile"));
                map.Remove(k);
            }
            map.Add(k, t);
            if (!EamHas(k)) Call(Eam(), "CreateInstance", k);
            object s = Call(A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", t);
            SetProp(t, "SpecificationInfo", s);
            pkg.Tzp = t; pkg.Si = s; pkg.Key = k;
            pkg.Program = S(Prop(t, "ProgramName"));
        } finally { done.Set(); sw.Stop(); pkg.LoadMs = sw.Elapsed.TotalMilliseconds; }
        return pkg;
    }

    /// <summary>Every mutation needs an UndoRedoManager registered for that key; the designer's
    /// SetInitGridX/Y throw if one is present during *load*, so it has to be added afterwards.
    /// This is the one-way door: once set, the P0 probe showed the key can no longer be loaded.</summary>
    static void RegisterUndoRedo(Pkg p) {
        var urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
        object urm = Activator.CreateInstance(Find(urf, "UndoRedoManager"), new object[] { 100 });
        Call(urm, "Init");
        UrmMap()[p.Key] = urm;
    }

    /// <summary>What the spec calls close: the three removals. Each one is reported so we can
    /// see which of them actually had something to remove. Note this is NOT the designer's own
    /// close -- that is SettingManager.CloseFile, reached by publishing TzpFileClose on the
    /// Global aggregator, and it does two extra things (see Test1b).</summary>
    static void ClosePkg(Pkg p, string label) {
        var map = TzpMap();
        var urm = UrmMap();
        bool hadTzp = map.Contains(p.Key);
        object occupant = hadTzp ? map[p.Key] : null;
        bool occupantIsUs = ReferenceEquals(occupant, p.Tzp);
        bool hadUrm = urm.Contains(p.Key);
        object urmVal = hadUrm ? urm[p.Key] : null;
        bool hadEam = EamHas(p.Key);
        object ad = hadTzp ? Prop(occupant, "ActionDefaults") : null;

        Console.WriteLine("  --- close(" + label + ")  key=" + p.Key + " ---");
        Console.WriteLine("      before : tzpMap[key]=" + hadTzp
            + (hadTzp ? "  occupantIsThisHandle=" + occupantIsUs : "")
            + "  urmMap[key]=" + hadUrm + "  EAM[key]=" + hadEam);
        if (hadUrm) Console.WriteLine("      urmMap[key] belongs to this handle: " + ReferenceEquals(urmVal, p.Tzp) + " (n/a, it is an UndoRedoManager)");

        bool r1 = map.Contains(p.Key);
        map.Remove(p.Key);
        bool r2 = urm.Contains(p.Key);
        urm.Remove(p.Key);
        bool r3 = EamHas(p.Key);
        Call(Eam(), "Remove", p.Key);

        Console.WriteLine("      remove tzpMap[key]  : had=" + r1 + " -> has=" + map.Contains(p.Key)
            + "   (tzpMap.Count now " + map.Count + ")");
        Console.WriteLine("      remove urmMap[key]  : had=" + r2 + " -> has=" + urm.Contains(p.Key)
            + "   (urmMap.Count now " + urm.Count + ")");
        Console.WriteLine("      EAM.Remove(key)     : had=" + r3 + " -> has=" + EamHas(p.Key));
        if (hadTzp) Console.WriteLine("      (this handle's ActionDefaults=" + Q(ad) + ". SettingManager.CloseFile"
            + " also sets ActionDefaults = null before removing; only load-bearing when non-null)");
        Console.WriteLine("      Global event subscribers after the three removals: " + SubCounts()
            + "   <- UNCHANGED: neither the TzpManager nor the SpecificationInfo unsubscribed");
        Console.WriteLine("      TzpManager.Current after close -> "
            + (Cur() == null ? "<null>" : S(Prop(Cur(), "ProgramName")) + " [" + (ReferenceEquals(Cur(), p.Tzp) ? "stale reference to the closed handle" : "other package") + "]"));
    }

    /// <summary>How many handlers are subscribed to TzpFileClose on the Global aggregator.
    /// SettingManager's ctor subscribes one for all time; each TzpManager ctor adds one, and
    /// only TzpManager.TzpFileClosed (fired by publishing TzpFileClose) removes it again.</summary>
    /// <summary>Subscriber count for one Global event. Prism's EventBase keeps its subscriber
    /// list in a private field whose name is a Prism-version detail, so find the ICollection-
    /// valued field instead of guessing the name. This is what makes the close-time leak visible:
    /// SettingManager's ctor subscribes one TzpFileClose handler for the life of the process,
    /// and every open adds TzpManager.TzpFileClosed (TzpManager.cs:225) and
    /// SpecificationInfo.OnTzpFileClose (SpecificationInfo.cs:149) -- neither of which the three
    /// removals touch.</summary>
    static string GlobalSubs(string eventName) {
        try {
            object eam = Prop2(Eam(), "Global");
            Type evtT = Find(A, eventName);
            if (evtT == null) return "<no type " + eventName + ">";
            MethodInfo ge = null;
            foreach (var m in eam.GetType().GetMethods())
                if (m.Name == "GetEvent" && m.IsGenericMethodDefinition && m.GetParameters().Length == 0) { ge = m; break; }
            if (ge == null) return "<no GetEvent<T>>";
            object evt = ge.MakeGenericMethod(evtT).Invoke(eam, null);
            var hits = new List<string>(); var names = new List<string>();
            for (Type t = evt.GetType(); t != null; t = t.BaseType) {
                foreach (var f in t.GetFields(BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.DeclaredOnly)) {
                    object v = null;
                    try { v = f.GetValue(evt); } catch { }
                    if (v is ICollection) hits.Add(((ICollection)v).Count.ToString());
                    else names.Add(t.Name + "." + f.Name + ":" + f.FieldType.Name);
                }
            }
            if (hits.Count > 0) return string.Join("/", hits.ToArray());
            return "<no ICollection field on " + evt.GetType().FullName + "; " + string.Join(" ", names.ToArray()) + ">";
        } catch (Exception ex) { return "<err: " + ex.Message + ">"; }
    }
    /// <summary>Publish one Global event -- the designer's own signalling path.</summary>
    static void PublishEvent(string eventName, object key) {
        object eam = Prop2(Eam(), "Global");
        Type evtT = Find(A, eventName);
        MethodInfo ge = null;
        foreach (var m in eam.GetType().GetMethods())
            if (m.Name == "GetEvent" && m.IsGenericMethodDefinition && m.GetParameters().Length == 0) { ge = m; break; }
        object evt = ge.MakeGenericMethod(evtT).Invoke(eam, null);
        Call(evt, "Publish", key);
    }
    /// <summary>Subscriber counts for the two Global events a handle registers on when it opens.</summary>
    static string SubCounts() {
        return "TzpFileClose=" + GlobalSubs("TzpFileClose")
             + "  SaveSettingEvent=" + GlobalSubs("SaveSettingEvent");
    }

    // ---------------------------------------------------------------- addressing

    /// <summary>Walk the form model to a name-path, the same addressing the other tools use.</summary>
    static object FindByPath(Pkg p, string path) {
        if (p.Si == null || path == null) return null;
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
    static int CountAll(Pkg p) {
        var paths = new List<string>(); var els = new List<object>();
        object form = Prop(p.Si, "FormNode");
        if (form == null) return 0;
        CollectAll(form, "x", paths, els);
        return els.Count;
    }
    static List<string> AllPaths(Pkg p) {
        var paths = new List<string>(); var els = new List<object>();
        object form = Prop(p.Si, "FormNode");
        if (form == null) return paths;
        CollectAll(form, RootName(p) + "/" + S(Prop(form, "Name")), paths, els);
        return paths;
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

    // ---------------------------------------------------------------- write path

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
        if (el == null) return "<path not found>";
        var pi = Indexer(el);
        if (pi == null) return "<no indexer>";
        try {
            pi.SetValue(el, val, new object[] { attr });
        } catch (Exception ex) {
            return "<threw: " + ExText(ex) + ">";
        }
        return S(Call(el, "GetAttribute", attr));
    }

    /// <summary>Read the attribute straight out of the .4fd on disk, so "the mutation is gone"
    /// can be checked against the file rather than against a second in-memory model. Several
    /// elements can share a name (the <RecordField> under <Record> does), so take the first
    /// one that actually carries the attribute.</summary>
    static string DiskAttr(string tzs, string elemName, string attr) {
        try {
            byte[] zip = File.ReadAllBytes(tzs);
            string fd = EntryText(zip, EntryName(zip, ".4fd"));
            if (fd == null) return "<no .4fd>";
            var x = XElement.Parse(fd);
            foreach (var d in x.Descendants()) {
                if ((string)d.Attribute("name") != elemName) continue;
                var v = d.Attribute(attr);
                if (v != null) return v.Value;
            }
            return "<no " + attr + " for that name in .4fd>";
        } catch (Exception ex) { return "<err: " + ex.Message + ">"; }
    }

    // ---------------------------------------------------------------- plumbing

    static string EntryName(byte[] zip, string suffix) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
            foreach (var e in z.Entries) if (e.FullName.EndsWith(suffix)) return e.FullName;
        return null;
    }
    static string EntryText(byte[] zip, string name) {
        if (name == null) return null;
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
            return r.ReadToEnd();
    }

    // ================================================================ test 1

    static void Test1(string fileA, int tmo) {
        Console.WriteLine("========== TEST 1: close then reopen the same file ==========");
        Console.WriteLine("  file A = " + fileA);
        Console.WriteLine("  Global event subscribers at start: " + SubCounts());

        // (1) open, do NOT register UndoRedo yet.
        var a = LoadPackage(fileA, tmo);
        int n0 = CountAll(a);
        Console.WriteLine();
        Console.WriteLine("  [1] open(A) ok, " + a.LoadMs.ToString("F0") + " ms   key=" + a.Key);
        Console.WriteLine("      elements=" + n0 + "   form elements in FormSpeDictionary="
            + CountDic(a));
        Console.WriteLine("      undoRedoManagerMap has key now: " + UrmMap().Contains(a.Key)
            + "   (state = Loaded)");
        Console.WriteLine("      Global event subscribers after open: " + SubCounts()
            + "   <- +2 TzpFileClose, +1 SaveSettingEvent per open");

        string path = FirstWidgetPath(a);
        string attr = "case";
        object el = FindByPath(a, path);
        string elemName = S(Prop(el, "Name"));
        string orig = S(Call(el, "GetAttribute", attr));
        string disk0 = DiskAttr(a.Path, elemName, attr);
        Console.WriteLine();
        Console.WriteLine("  [2] mutation target: " + path);
        Console.WriteLine("      name=" + Q(elemName) + "  " + attr + " in memory " + Q(orig)
            + "  in .4fd on disk " + Q(disk0));

        // Registering the UndoRedoManager is what makes the key un-reloadable -- that is the
        // point of the test, so it must happen before the mutation.
        RegisterUndoRedo(a);
        Console.WriteLine("      RegisteredUndoRedo -> undoRedoManagerMap has key: " + UrmMap().Contains(a.Key)
            + "   (state = Mutable)");
        string want = orig == "upper" ? "lower" : "upper";
        if (string.IsNullOrEmpty(orig)) want = "probe_w04";
        string r1 = SetViaIndexer(a, path, attr, want);
        Console.WriteLine("      indexer write " + attr + "=" + Q(want) + " -> " + Q(r1)
            + (r1 == want ? "   MUTATED" : "   !! WRITE DID NOT TAKE"));
        Console.WriteLine("      re-read through the model: "
            + Q(S(Call(FindByPath(a, path), "GetAttribute", attr))));

        // (3) close: the three removals.
        Console.WriteLine();
        Console.WriteLine("  [3] close(A)");
        ClosePkg(a, "A");

        // (4) reopen the same key in the same process -- THE question.
        Console.WriteLine();
        Console.WriteLine("  [4] open(A) again, same process, same key ...");
        Pkg a2 = null; string reopenErr = null;
        try {
            a2 = LoadPackage(fileA, tmo);
            Console.WriteLine("      loaded ok in " + a2.LoadMs.ToString("F0") + " ms");
        } catch (Exception ex) {
            reopenErr = ExText(ex);
            Console.WriteLine("      THREW: " + reopenErr);
            Console.WriteLine("      tzpMap has key after the failed load: " + TzpMap().Contains(a.Key)
                + "   (LoadPackage does map.Add BEFORE SpecificationInfo.Create)");
            Console.WriteLine("      EAM has key after the failed load: " + EamHas(a.Key));
        }
        Console.WriteLine("      VERDICT T1-REOPEN: " + (reopenErr == null
            ? "YES -- the key was released and re-opened without throwing"
            : "NO -- reopen threw " + reopenErr));

        if (a2 != null) {
            // (5) same content, and the in-memory mutation must be gone.
            int n1 = CountAll(a2);
            object el2 = FindByPath(a2, path);
            string v1 = el2 == null ? "<path not found>" : S(Call(el2, "GetAttribute", attr));
            string disk1 = DiskAttr(a2.Path, elemName, attr);
            Console.WriteLine();
            Console.WriteLine("  [5] element count: first load " + n0 + " -> reload " + n1
                + (n1 == n0 ? "   MATCH" : "   !! MISMATCH"));
            Console.WriteLine("      " + attr + " after reload: " + Q(v1)
                + "   (was " + Q(want) + " in the mutated memory, " + Q(disk0) + " on disk)");
            Console.WriteLine("      VERDICT T1-FRESH: " + (v1 == disk0
                ? "fresh load -- the mutation is GONE, the file was re-read"
                : v1 == want ? "!! the mutated value survived the reopen"
                : "unexpected value " + Q(v1)));

            // (6) mutate again -- is the second cycle fully functional?
            RegisterUndoRedo(a2);
            string want2 = v1 == "upper" ? "lower" : "upper";
            if (v1 == want2 || v1 == "<path not found>") want2 = "probe_w04_cycle2";
            string r2 = SetViaIndexer(a2, path, attr, want2);
            Console.WriteLine();
            Console.WriteLine("  [6] second cycle: register UndoRedoManager, write " + attr + "=" + Q(want2)
                + " -> " + Q(r2));
            Console.WriteLine("      VERDICT T1-FUNCTIONAL: " + (r2 == want2
                ? "YES -- the reopened handle accepts writes"
                : "NO -- second cycle write failed"));

            Console.WriteLine();
            Console.WriteLine("  [1b] now publish the designer's own close event TzpFileClose(key),");
            Console.WriteLine("       which SettingManager's ctor subscribed to -> CloseFile(key):");
            Console.WriteLine("       subscribers before publish: " + SubCounts());
            PublishEvent("TzpFileClose", a2.Key);
            Console.WriteLine("       subscribers after  publish: " + SubCounts()
                + "   <- both leaks reclaimed, back to SettingManager's own handler");
            Console.WriteLine("       tzpMap has key: " + TzpMap().Contains(a2.Key)
                + "  urmMap has key: " + UrmMap().Contains(a2.Key));
            Console.WriteLine("       old handle's SpecificationInfo after the event: "
                + (Prop(a2.Tzp, "SpecificationInfo") == null ? "nulled by TzpFileClosed" : "still set"));
            Console.WriteLine("       EAM has key after the event: " + EamHas(a2.Key)
                + "   <- CloseFile does not touch the EAM");
            Console.WriteLine("       NOTE the stale handle's SpecificationInfo also holds a"
                + " SaveSettingEvent subscription until TzpFileClose fires; its handler body calls");
            Console.WriteLine("       SettingManager.Get().GetTzpManger(this.Key).SaveSpecificationInfo(...)"
                + " -- i.e. it would write the CLOSED package's model");
            Console.WriteLine("       into whichever TzpManager now owns that key (SpecificationInfo.cs:260)."
                + " Not published here: it writes to disk.");
        }
        Console.WriteLine();
    }

    static int CountDic(Pkg p) {
        var d = Prop(p.Si, "FormSpeDictionary") as IDictionary;
        return d == null ? -1 : d.Count;
    }

    // ================================================================ test 2

    static void Test2(string fileA, string fileB, int tmo) {
        Console.WriteLine("========== TEST 2: two different files sharing one ProgramKey ==========");
        Console.WriteLine("  file A = " + fileA);
        Console.WriteLine("  file B = " + fileB);

        var a = LoadPackage(fileA, tmo);
        Console.WriteLine();
        Console.WriteLine("  [1] open(A): key=" + a.Key + "  elements=" + CountAll(a));
        var b = LoadPackage(fileB, tmo);
        Console.WriteLine("      open(B): key=" + b.Key + "  elements=" + CountAll(b));
        Console.WriteLine("      same key? " + Equals(a.Key, b.Key)
            + "   (B collided on load: " + b.Collided + ")");
        if (b.Collided)
            Console.WriteLine("      -> B evicted the handle that was under the key: " + b.ReplacedPath);
        Console.WriteLine("      VERDICT T2-COLLISION: " + (Equals(a.Key, b.Key)
            ? "the two files DO share one key" : "no collision observed"));

        // (2) does the second load disturb the first? A's handle is no longer in tzpMap, but we
        // still hold it, so we can ask A directly.
        Console.WriteLine();
        Console.WriteLine("  [2] A is still addressable through the handle we hold:");
        int na = CountAll(a), nb = CountAll(b);
        Console.WriteLine("      elements: A=" + na + "  B=" + nb + (na == nb ? "  (equal)" : "  (differ)"));
        var pa = AllPaths(a); var pb = new HashSet<string>(AllPaths(b));
        var commonAll = new List<string>();
        foreach (var s in pa) if (pb.Contains(s)) commonAll.Add(s);
        Console.WriteLine("      common element paths: " + commonAll.Count + " of " + pa.Count);
        int compared = 0, differs = 0, shown = 0; string probePath = null;
        foreach (var s in commonAll) {
            object ea = FindByPath(a, s), eb = FindByPath(b, s);
            if (ea == null || eb == null) continue;
            string va = S(Call(ea, "GetAttribute", "case"));
            string vb = S(Call(eb, "GetAttribute", "case"));
            if (va == "(null)" && vb == "(null)") continue;    // containers/form nodes have no `case`
            if (probePath == null) probePath = s;
            compared++;
            if (va != vb) differs++;
            if (shown < 3) {
                Console.WriteLine("        " + s);
                Console.WriteLine("          A " + Q(va) + "   B " + Q(vb));
                shown++;
            }
        }
        Console.WriteLine("      widgets with a value compared: " + compared + "   differing: " + differs);
        Console.WriteLine("      VERDICT T2-COMPARE: counts A=" + na + " B=" + nb + ", "
            + compared + " common widgets compared, " + differs + " differ");

        // (2b) The comparison above cannot settle cross-talk on its own: A and B are two
        // versions of the same program and agree on every common widget, so "values agree"
        // would also be what a cross-talk bug looks like. Mutate one and read the other.
        Console.WriteLine();
        Console.WriteLine("  [2b] mutate A in memory, then read the same path out of B:");
        if (probePath == null) {
            Console.WriteLine("      no widget carrying `case` is present in both -- skipped");
            Console.WriteLine("      VERDICT T2-XTALK: inconclusive");
        } else {
            object ea0 = FindByPath(a, probePath);
            string before = S(Call(ea0, "GetAttribute", "case"));
            string wantA = before == "upper" ? "lower" : "upper";
            RegisterUndoRedo(a);   // the manager is stored per KEY, so both handles share it
            Console.WriteLine("      registered an UndoRedoManager under the shared key: "
                + UrmMap().Contains(a.Key));
            string gotA = SetViaIndexer(a, probePath, "case", wantA);
            string bVal = S(Call(FindByPath(b, probePath), "GetAttribute", "case"));
            Console.WriteLine("      target " + probePath.Substring(probePath.Length > 60 ? probePath.Length - 60 : 0));
            Console.WriteLine("        A now " + Q(gotA) + " (was " + Q(before) + ")   B reads " + Q(bVal));
            Console.WriteLine("      VERDICT T2-XTALK: " + (gotA == wantA && bVal != wantA
                ? "the two handles are independent -- A's write did not reach B"
                : gotA != wantA ? "A's write did not take (" + Q(gotA) + ")"
                : "!! B saw A's value"));
        }

        // (3) what does tzpMap say?
        Console.WriteLine();
        var map = TzpMap();
        Console.WriteLine("  [3] tzpMap.Count = " + map.Count);
        bool hasKey = map.Contains(a.Key);
        object occupant = hasKey ? map[a.Key] : null;
        Console.WriteLine("      tzpMap[key] present: " + hasKey
            + "   is it A? " + ReferenceEquals(occupant, a.Tzp)
            + "   is it B? " + ReferenceEquals(occupant, b.Tzp));
        Console.WriteLine("      undoRedoManagerMap.Count = " + UrmMap().Count
            + "   <- registered by [2b]; but tzpMap[key] is B, so that manager is now attached"
            + " to an ORPHANED handle (A)");
        Console.WriteLine("      VERDICT T2-SURVIVOR: "
            + (ReferenceEquals(occupant, b.Tzp) ? "only B exists; the add replaced A's entry"
            : ReferenceEquals(occupant, a.Tzp) ? "only A exists" : "unexpected occupant"));

        // (4) does close resolve the collision?
        Console.WriteLine();
        Console.WriteLine("  [4] close(A) then open(A) -- does close resolve the collision?");
        ClosePkg(a, "A");
        Console.WriteLine("      NOTE: close is by key, so it removed whichever handle owned the key"
            + " -- here that is " + (b.ReplacedPath != null ? "B" : "?") + ", not A.");
        Console.WriteLine("      tzpMap.Count after close(A): " + TzpMap().Count
            + "   B still reachable through our handle: " + (CountAll(b) > 0));
        Pkg a2 = null; string err = null;
        try { a2 = LoadPackage(fileA, tmo); } catch (Exception ex) { err = ExText(ex); }
        if (err == null) {
            Console.WriteLine("      open(A) again ok: key=" + a2.Key + "  elements=" + CountAll(a2)
                + "  tzpMap.Count=" + TzpMap().Count);
            Console.WriteLine("      VERDICT T2-CLOSE: YES -- close frees the key, A loads again");
        } else {
            Console.WriteLine("      open(A) again THREW: " + err);
            Console.WriteLine("      VERDICT T2-CLOSE: NO -- " + err);
        }

        // (5) the other half of the hazard: silent eviction is only silent while the occupant is
        // Loaded. If it is already Mutable, the second load is what breaks -- which is the exact
        // case E_KEY_IN_USE exists to refuse.
        Console.WriteLine();
        Console.WriteLine("  [5] load B while the key's occupant is Mutable");
        if (a2 != null) {
            RegisterUndoRedo(a2);
            Console.WriteLine("      registered the occupant's UndoRedoManager -> urmMap has key: "
                + UrmMap().Contains(a2.Key) + "   (occupant of tzpMap[key] is A: "
                + ReferenceEquals(TzpMap()[a2.Key], a2.Tzp) + ")");
            Pkg b2 = null; string err2 = null;
            try { b2 = LoadPackage(fileB, tmo); } catch (Exception ex) { err2 = ExText(ex); }
            if (err2 == null) {
                Console.WriteLine("      open(B) succeeded: " + CountAll(b2) + " elements");
                Console.WriteLine("      tzpMap[key] is B: " + ReferenceEquals(TzpMap()[b2.Key], b2.Tzp)
                    + "   A's UndoRedoManager still registered: " + UrmMap().Contains(a2.Key)
                    + "   <- orphaned if A is dropped");
                Console.WriteLine("      VERDICT T2-MUTABLE: the load silently evicted a Mutable handle");
            } else {
                Console.WriteLine("      open(B) THREW: " + err2);
                Console.WriteLine("      state left behind -- tzpMap[key] present: " + TzpMap().Contains(a2.Key)
                    + "  occupantIsA: " + ReferenceEquals(TzpMap()[a2.Key], a2.Tzp)
                    + "  urmMap[key]: " + UrmMap().Contains(a2.Key)
                    + "  EAM[key]: " + EamHas(a2.Key));
                Console.WriteLine("      VERDICT T2-MUTABLE: a Mutable handle cannot be replaced -- the load throws");
            }
        }
        Console.WriteLine();
    }

    // ================================================================ test 3

    static void Test3() {
        Console.WriteLine("========== TEST 3: process-global side effects of close/reopen ==========");
        object pm = Prop2(Find(A, "PreferenceManager"), "Current");
        object prefs = pm == null ? null : Prop(pm, "Settings");
        bool prefsOk = prefs != null;
        bool vf = false;
        if (prefsOk) { object v = Prop(prefs, "ValidateForm"); vf = v is bool && (bool)v; }
        Console.WriteLine("  PreferenceManager.Current.Settings non-null : " + prefsOk
            + (prefsOk ? "  (" + prefs.GetType().Name + ")" : ""));
        Console.WriteLine("  ...ValidateForm                            : " + (prefsOk ? vf.ToString() : "n/a")
            + (prefsOk && !vf ? "  (false, as injected)" : prefsOk ? "  !! CHANGED" : ""));

        object xsd = Prop(SM, "Info_TsdXsd");
        Console.WriteLine("  SettingManager.Get().Info_TsdXsd non-null   : " + (xsd != null)
            + "  (len " + (xsd as string ?? "").Length + ")");

        bool appOk = Application.Current != null;
        string res = null;
        try { res = S(Call(Application.Current, "FindResource", "Message_DuplicateFieldName")); }
        catch (Exception ex) { res = "<" + ex.Message + ">"; }
        bool resOk = Application.Current != null && res != null && !res.StartsWith("<");
        Console.WriteLine("  Application.Current non-null               : " + appOk);
        // Do not print the resource VALUE: it is Chinese and the OEM code page mangles it in a
        // pipe. A returned string of the right shape is the whole assertion.
        Console.WriteLine("  Application.Current.FindResource works     : " + resOk
            + "  (returned " + (res == null ? "null" : res.Length + " chars") + ")");

        Console.WriteLine("  VERDICT T3-GLOBALS: " + ((prefsOk && !vf && xsd != null && appOk && resOk)
            ? "PASS -- all three globals survive the close/reopen cycle"
            : "FAIL -- PreferenceSettings=" + prefsOk + " ValidateForm=" + (prefsOk ? vf.ToString() : "?")
              + " Info_TsdXsd=" + (xsd != null) + " Application=" + appOk + " FindResource=" + resOk));
        Console.WriteLine();
    }

    // ================================================================ main

    [STAThread]
    static void Main(string[] args) {
        if (args.Length < 1) {
            Console.WriteLine("ProbeReopen.exe <fileA.tzs> [fileB.tzs]");
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
        // SetAttribute during load NREs inside XmlElement.CheckOverlapping.
        object prefs = Activator.CreateInstance(Find(A, "PreferenceModel"));
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(Find(A, "PreferenceManager"), "Current"), "_preferenceModel", prefs);

        int tmo = 90;
        int.TryParse(Environment.GetEnvironmentVariable("TZSCLI_RELOAD_TIMEOUT") ?? "90", out tmo);
        Console.WriteLine("workspace = " + WS + "   load timeout = " + tmo + "s");
        Console.WriteLine();

        int rc = 0;
        try {
            // Test 3 runs before Test 2 on purpose: Test 2's last step provokes a load the
            // designer is expected to reject, and if that path ever raises a modal dialog the
            // watchdog kills the process -- Test 3's global checks would be lost with it.
            Test1(fileA, tmo);
            Test3();
            Test2(fileA, fileB, tmo);
        } catch (Exception ex) {
            Console.WriteLine();
            Console.WriteLine("!! PROBE THREW: " + ex.GetType().Name + ": " + ex.Message);
            Console.WriteLine(ex.StackTrace);
            rc = 4;
        }
        Console.WriteLine("========== ProbeReopen end (rc=" + rc + ") ==========");
        Environment.Exit(rc);
    }
}
