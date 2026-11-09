using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Reflection;
using System.Text;
using System.Windows;
using System.Windows.Markup;
using System.Xml.Linq;
using TzsCli;

[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// End-to-end: FormWriter edits the .4fd, the designer's own code derives .tsd and .bdx
/// from it. Compared against what the designer produced by hand for the same operation.
/// </summary>
class E2E
{
    static Assembly A; static object SM; static object si; static object pk;
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
    const string WS      = @"D:\t100_wrok_dir\hengshuo\prd";
    const string BEFORE  = WS + @"\apmt500_wf(c) - 测试备份 - 修改前.tzs";
    const string AFTER   = WS + @"\apmt500_wf(c).tzs";
    const string TABLE = "pmdl_t", COL = "pmdlud001", NEWNAME = "pmdl_t.pmdlud001";

    static object Call(object target, string name, params object[] args) {
        Type t = target as Type ?? target.GetType();
        foreach (var m in t.GetMethods(BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static)) {
            if (m.Name != name || m.GetParameters().Length != args.Length) continue;
            try { return m.Invoke(target is Type ? null : target, args); }
            catch (TargetInvocationException ex) {
                System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw(); return null; }
        }
        throw new Exception("no method " + name + "/" + args.Length + " on " + t);
    }
    static object Prop(object o, string n) {
        if (o == null) return null;
        var t = o.GetType();
        var pi = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (pi != null) return pi.GetValue(o, null);
        var fi = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        return fi == null ? null : fi.GetValue(o);
    }
    static object Prop2(Type t, string n) {
        var pi = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
        if (pi != null) return pi.GetValue(null, null);
        var fi = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
        return fi == null ? null : fi.GetValue(null);
    }
    static void SetProp(object o, string n, object v) {
        var t = o.GetType();
        var pi = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (pi != null) { pi.SetValue(o, v, null); return; }
        t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance).SetValue(o, v);
    }
    static string S(object o) { return o == null ? "(null)" : o.ToString(); }
    static void Dump(Exception ex) {
        for (Exception e = ex; e != null; e = e.InnerException) {
            Console.WriteLine("      " + e.GetType().FullName + ": " + e.Message);
            if (e.StackTrace != null) foreach (var l in e.StackTrace.Split('\n').Take(5)) Console.WriteLine("        " + l.Trim());
        }
    }
    static string ReadEntry(string tzs, string suffix) {
        using (var z = ZipFile.OpenRead(tzs))
            foreach (var e in z.Entries)
                if (e.FullName.EndsWith(suffix)) { using (var s = e.Open()) using (var r = new StreamReader(s, Encoding.UTF8)) return r.ReadToEnd(); }
        return null;
    }
    static XElement FieldElem(string tzs, string entry, string name) {
        var d = XElement.Parse(ReadEntry(tzs, entry));
        return d.Descendants("field").FirstOrDefault(e => (string)e.Attribute("name") == name);
    }
    /// <summary>Collects every element carrying the given name, keyed by tag, so field /
    /// hfield / pfield / sfield can all be compared in one pass.</summary>
    static Dictionary<string,string> Collect(XElement tsd, string name) {
        var d = new Dictionary<string,string>();
        foreach (var e in tsd.Descendants()) {
            if ((string)e.Attribute("name") != name) continue;
            d[e.Name.LocalName] = Attrs(e);
        }
        return d;
    }

    static string Attrs(XElement e) {
        if (e == null) return "(missing)";
        var sb = new StringBuilder();
        foreach (var a in e.Attributes()) sb.Append(a.Name).Append('=').Append(a.Value).Append(" | ");
        return sb.ToString();
    }

    [STAThread]
    static void Main() {
        // ---------- step 1: FormWriter edits the .4fd ----------
        string fd4 = ReadEntry(BEFORE, ".4fd");
        Console.WriteLine("[1] .4fd in : " + fd4.Length + " chars");

        var w = FormWriter.Load(fd4);
        // insert into the same grid the designer used
        var parent = w.Index.All.FirstOrDefault(e => e.Tag == "Grid" && e.Name != null && e.Parent != null);
        if (parent == null) { Console.WriteLine("    no Grid parent"); return; }
        Console.WriteLine("    parent   : " + parent.Path);

        var el = w.CloneTemplate("ButtonEdit", new Dictionary<string,string> {
            {"name", NEWNAME}, {"fieldId", null},
            {"sqlTabName", TABLE}, {"colName", COL}, {"fieldType", "TABLE_COLUMN"},
            {"title", "lbl_" + COL}, {"comment", "cmt_" + COL},
            {"image", "16/openwindow.png"}, {"action", "controlp"}, {"widget", "ButtonEdit"}
        });
        w.AddNode(parent.Path, el);
        string fd4New = w.Render();
        Console.WriteLine("    .4fd out: " + fd4New.Length + " chars  (delta " + (fd4New.Length - fd4.Length) + ")");
        // sanity: output is well-formed and contains the new element
        var check = XElement.Parse(fd4New);
        Console.WriteLine("    parses  : yes   new element present: "
            + check.Descendants().Any(e => (string)e.Attribute("name") == NEWNAME));

        // ---------- step 2: hand it to the designer ----------
        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };

        Application app = new Application();
        DesignerLang.Merge(app, INSTALL);
        A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        Type smT = A.GetType("SpecDesignerCommon.SettingManager");
        Type modelT = A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel");
        if (smT == null) { Console.WriteLine("    diag: SettingManager type NOT FOUND"); return; }
        if (modelT == null) { Console.WriteLine("    diag: SettingModel type NOT FOUND"); return; }
        SM = Call(smT, "Get");
        object model = Call(modelT, "Create");
        Console.WriteLine("    diag: SM=" + (SM == null ? "NULL" : "ok") + "  model=" + (model == null ? "NULL" : "ok"));
        SetProp(SM, "CurrentSetting", model);
        object conn = Prop(model, "Connection");
        Console.WriteLine("    diag: model.Connection=" + (conn == null ? "NULL" : "ok"));
        if (conn == null) return;
        SetProp(conn, "Workspace", WS);
        Call(SM, "LoadCommonData");

        Type prefT = null, pmT = null, eamT = null, tch = null;
        foreach (Type t in A.GetTypes()) {
            if (t.Name == "PreferenceModel") prefT = t;
            if (t.Name == "PreferenceManager") pmT = t;
            if (t.Name == "EventAggregatorManager") eamT = t;
            if (t.Name == "TableColumnHelper") tch = t;
        }
        object prefs = Activator.CreateInstance(prefT);
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(pmT, "Current"), "_preferenceModel", prefs);

        Type tzpT = A.GetType("SpecDesignerCommon.TzpManager");
        object tzp = Activator.CreateInstance(tzpT, new object[] { BEFORE });
        pk = Prop(tzp, "ProgramKey");
        ((IDictionary)Prop(SM, "tzpMap")).Add(pk, tzp);
        Call(eamT, "CreateInstance", pk);

        Assembly urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
        Type urmT = null; foreach (Type t in urf.GetTypes()) if (t.Name == "UndoRedoManager") urmT = t;
        object urm = Activator.CreateInstance(urmT, new object[] { 100 });
        Call(urm, "Init");
        // NOTE: registering the UndoRedoManager must happen AFTER SpecificationInfo.Create.
        // GetChildNode calls XmlElement.SetInitGridX(), which deliberately throws
        // "illeagal call SetInitGridX" when an UndoRedoManager is present. Load wants none,
        // enrichment wants one — hence the ordering.

        // inject the edited .4fd before the model is built
        SetProp(tzp, "GeneroFormString", fd4New);
        Console.WriteLine("[2] injected edited .4fd into TzpManager");

        Type siT = A.GetType("SpecDesignerCommon.SpecificationInfo");
        si = Call(siT, "Create", tzp);
        SetProp(tzp, "SpecificationInfo", si);
        Console.WriteLine("    SpecificationInfo created; Env=" + S(Prop(si, "Env")));
        if (si == null) return;

        ((IDictionary)Prop(SM, "undoRedoManagerMap")).Add(pk, urm);
        Console.WriteLine("    UndoRedoManager registered (after load)");

        // ---------- step 3: did loading register our new field? ----------
        object dict = Prop(si, "FormSpeDictionary");
        object fsm = null;
        foreach (DictionaryEntry de in (IDictionary)dict) if (S(de.Key) == NEWNAME) fsm = de.Value;
        Console.WriteLine("[3] FormSpeDictionary[\"" + NEWNAME + "\"] = " + (fsm == null ? "MISSING" : "present"));
        if (fsm == null) {
            Console.WriteLine("    -> loading did NOT register the element; nothing more to do");
            return;
        }
        object sf = Prop(fsm, "SpecField");
        Console.WriteLine("    SpecField = " + (sf == null ? "NULL" : sf.GetType().Name));
        if (sf == null) return;

        // ---------- step 4: enrichment (SPEC 8.6) ----------
        XElement ci  = (XElement)Call(tch, "GetColumnInfo", TABLE, COL);
        XElement cai = (XElement)Call(tch, "GetColumnAttrInfo", TABLE, COL);
        Call(sf, "SetAttribute", "widget", "ButtonEdit");
        if (ci != null) {
            Call(sf, "SetAttribute", "attribute", (string)ci.Attribute("attribute"));
            Call(sf, "SetAttribute", "type",      (string)ci.Attribute("type"));
            Call(sf, "SetAttribute", "req",       (string)ci.Attribute("req"));
        }
        if (cai != null)
            foreach (string k in new[]{"i_zoom","c_zoom","default","max","min","chk_ref","items"})
                Call(sf, "SetAttribute", k, (string)cai.Attribute(k));
        Call(sf, "SetAttribute", "Name", NEWNAME);                 // capital-N mirror

        // AbstractSpecNode.ToXml() returns null while Status == CREATE: nodes born during
        // load are transients and are not serialised. The CLI is the one making the change,
        // so it must promote the node out of CREATE itself.
        Type ssT = null; foreach (Type t in A.GetTypes()) if (t.Name == "SpecStatus") ssT = t;
        object modify = Enum.Parse(ssT, "MODIFY");
        SetProp(sf, "Status", modify);
        Console.WriteLine("    SpecField.Status -> " + S(Prop(sf, "Status")));

        // The same CREATE-is-transient rule applies to every node the load created for this
        // element, not just the field: help code, prog rel and the localisation strings.
        int promoted = 1;
        object hc = Prop(fsm, "SpecHelpCode");
        if (hc != null) { SetProp(hc, "Status", modify); promoted++; }
        object pr = Prop(fsm, "SpecProgRel");
        if (pr != null) { SetProp(pr, "Status", modify); promoted++; }
        object fsCol = Prop(si, "fieldStrings");
        if (fsCol is IEnumerable) {
            foreach (var item in (IEnumerable)fsCol) {
                string nm = S(Prop(item, "Name"));
                if (nm == "lbl_" + COL || nm == "cmt_" + COL) { SetProp(item, "Status", modify); promoted++; }
            }
        }
        Console.WriteLine("    promoted nodes out of CREATE: " + promoted
            + "  (helpCode=" + (hc != null) + " progRel=" + (pr != null) + ")");
        Call(si, "SetFieldLocalStringText", "lbl_" + COL, Call(tch, "GetColumnTextByFullName", TABLE + "." + COL));
        Call(si, "SetFieldLocalStringText", "cmt_" + COL, Call(tch, "GetColumnTextByFullName", TABLE + "." + COL));
        Console.WriteLine("[4] enrichment applied");

        // ---------- step 5: save ----------
        Call(si, "SaveToTSD");
        Call(si, "SaveBinding");
        Console.WriteLine("[5] SaveToTSD + SaveBinding done");

        // ---------- step 6: compare ----------
        // ---------- step 6: compare EVERY .tsd node the designer produced ----------
        var tsdOut = (XElement)Prop(si, "TSDElement");
        var theirsTsd = XElement.Parse(ReadEntry(AFTER, ".tsd"));

        string[] names = { NEWNAME, "lbl_" + COL, "cmt_" + COL };
        Console.WriteLine();
        Console.WriteLine("[6] .tsd node comparison (all node kinds)");
        int same = 0, diff = 0;
        foreach (var nm in names) {
            var mineList = Collect(tsdOut, nm);
            var theirList = Collect(theirsTsd, nm);
            var keys = mineList.Keys.Union(theirList.Keys).OrderBy(k => k).ToList();
            Console.WriteLine("  --- name=" + nm + "   mine=" + mineList.Count + " designer=" + theirList.Count);
            foreach (var k in keys) {
                string m, t;
                mineList.TryGetValue(k, out m); theirList.TryGetValue(k, out t);
                bool ok = m == t && m != null;
                if (ok) same++; else diff++;
                Console.WriteLine("      " + (ok ? "OK  " : "DIFF") + " <" + k + ">");
                if (!ok) {
                    Console.WriteLine("           mine    : " + (m ?? "(missing)"));
                    Console.WriteLine("           designer: " + (t ?? "(missing)"));
                }
            }
        }
        Console.WriteLine("  => identical nodes: " + same + "   differing: " + diff);

        XElement theirs = FieldElem(AFTER, ".tsd", NEWNAME);

        // ---------- step 7: .bdx ----------
        Console.WriteLine();
        Console.WriteLine("[7] .bdx");
        try {
            object binding = Prop(si, "SpecBinding");
            int n = 0;
            if (binding is IEnumerable) foreach (var _ in (IEnumerable)binding) n++;
            Console.WriteLine("    SpecBinding entries: " + n);
        } catch (Exception ex) { Console.WriteLine("    " + ex.Message); }
    }
}
