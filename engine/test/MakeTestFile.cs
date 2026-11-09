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
/// Produces one real .tzs by running the whole pipeline end to end:
///   FormWriter edits the .4fd  ->  the designer's own code derives .tsd and .bdx
///   ->  TzsRepacker rewrites only those entries.
/// </summary>
class MakeTestFile
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
    const string IN_TZS  = WS + @"\aapp320(c).tzs";
    const string OUT_TZS = WS + @"\aapp320(c)_AIADD.tzs";

    const string TABLE = "apca_t", COL = "apcaent";
    const string NEWNAME = TABLE + "." + COL;                 // <表>.<列>
    const string PARENT  = "managedform/aapp320/mainlayout/condition/condition_page/vb_qbe"
                         + "/hbox_8/group_aapp320main/vbox_1/hbox_3/group_ldcomp"
                         + "/group_qbe_hbox/group_qbe_hbox_grid";

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

    static byte[] ReadAll(string p) { return File.ReadAllBytes(p); }
    static string EntryText(byte[] zip, string name) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
            return r.ReadToEnd();
    }
    static string EntryName(byte[] zip, string suffix) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
            foreach (var e in z.Entries) if (e.FullName.EndsWith(suffix)) return e.FullName;
        return null;
    }

    [STAThread]
    static void Main() {
        byte[] original = ReadAll(IN_TZS);
        string fdEntry = EntryName(original, ".4fd");
        string tsdEntry = EntryName(original, ".tsd");
        string bdxEntry = EntryName(original, ".bdx");
        Console.WriteLine("input    : " + IN_TZS + "  (" + original.Length + " bytes)");
        Console.WriteLine("entries  : " + fdEntry + ", " + tsdEntry + ", " + (bdxEntry ?? "(no .bdx)"));

        // ---------- 1. edit the .4fd ----------
        string fd4 = EntryText(original, fdEntry);
        var w = FormWriter.Load(fd4);
        var parent = w.Index.ByPath(PARENT);
        if (parent == null) { Console.WriteLine("FATAL: parent not found"); return; }

        // The designer picks a position from the drop location; a generator has none, so
        // append on the first free row instead of inheriting the template's coordinates.
        int row = w.NextFreeRow(PARENT);
        Console.WriteLine("         free row in parent: " + row + "  (its own gridHeight will grow to fit)");

        // A dragged field is a PAIR: a Label carrying the caption, plus the control.
        var lbl = w.CloneTemplate("Label", new Dictionary<string,string> {
            {"name", "lbl_" + COL}, {"text", "lbl_" + COL}, {"lstrtext", "true"},
            {"posX", "1"}, {"posY", row.ToString()}, {"gridWidth", "10"}, {"gridHeight", "1"}
        });
        var el = w.CloneTemplate("ButtonEdit", new Dictionary<string,string> {
            {"name", NEWNAME}, {"fieldId", null},
            {"sqlTabName", TABLE}, {"colName", COL}, {"fieldType", "TABLE_COLUMN"},
            {"title", "lbl_" + COL}, {"comment", "cmt_" + COL},
            {"posX", "12"}, {"posY", row.ToString()},
            {"image", "16/openwindow.png"}, {"action", "controlp"}
        });
        w.AddNode(PARENT, lbl);
        w.AddNode(PARENT, el);
        w.EnsureHeight(PARENT, row + 1);
        string fd4New = w.Render();
        Console.WriteLine("step 1   : .4fd " + fd4.Length + " -> " + fd4New.Length + " chars (+" + (fd4New.Length - fd4.Length) + ")");

        // ---------- 2. let the designer derive .tsd / .bdx ----------
        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };

        Application app = new Application();
        DesignerLang.Merge(app, INSTALL);
        A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        Type smT = A.GetType("SpecDesignerCommon.SettingManager");
        Type modelT = A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel");
        SM = Call(smT, "Get");
        object model = Call(modelT, "Create");
        SetProp(SM, "CurrentSetting", model);
        SetProp(Prop(model, "Connection"), "Workspace", WS);
        Call(SM, "LoadCommonData");

        Type prefT = null, pmT = null, eamT = null, tch = null, ssT = null;
        foreach (Type t in A.GetTypes()) {
            if (t.Name == "PreferenceModel") prefT = t;
            if (t.Name == "PreferenceManager") pmT = t;
            if (t.Name == "EventAggregatorManager") eamT = t;
            if (t.Name == "TableColumnHelper") tch = t;
            if (t.Name == "SpecStatus") ssT = t;
        }
        object prefs = Activator.CreateInstance(prefT);
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(pmT, "Current"), "_preferenceModel", prefs);

        Type tzpT = A.GetType("SpecDesignerCommon.TzpManager");
        object tzp = Activator.CreateInstance(tzpT, new object[] { IN_TZS });
        pk = Prop(tzp, "ProgramKey");
        ((IDictionary)Prop(SM, "tzpMap")).Add(pk, tzp);
        Call(eamT, "CreateInstance", pk);

        SetProp(tzp, "GeneroFormString", fd4New);      // hand the edited .4fd to the model
        Type siT = A.GetType("SpecDesignerCommon.SpecificationInfo");
        si = Call(siT, "Create", tzp);
        SetProp(tzp, "SpecificationInfo", si);
        if (si == null) { Console.WriteLine("FATAL: SpecificationInfo is null"); return; }

        // UndoRedoManager must come AFTER load (SetInitGridX throws when one is present)
        Assembly urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
        Type urmT = null; foreach (Type t in urf.GetTypes()) if (t.Name == "UndoRedoManager") urmT = t;
        object urm = Activator.CreateInstance(urmT, new object[] { 100 });
        Call(urm, "Init");
        ((IDictionary)Prop(SM, "undoRedoManagerMap")).Add(pk, urm);

        object dict = Prop(si, "FormSpeDictionary");
        object fsm = null;
        foreach (DictionaryEntry de in (IDictionary)dict) if (S(de.Key) == NEWNAME) fsm = de.Value;
        if (fsm == null) { Console.WriteLine("FATAL: element not registered by the load"); return; }
        Console.WriteLine("step 2   : designer loaded the edited .4fd and registered " + NEWNAME);

        // ---------- 3. enrichment (SPEC 8.6) ----------
        object sf = Prop(fsm, "SpecField");
        if (sf == null) { Console.WriteLine("FATAL: no SpecField"); return; }
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
        Call(sf, "SetAttribute", "Name", NEWNAME);      // capital-N mirror
        // the Label companion carries no text of its own; it resolves through these
        string colText = Convert.ToString(Call(tch, "GetColumnTextByFullName", TABLE + "." + COL));
        Call(si, "SetFieldLocalStringText", "lbl_" + COL, colText);
        Call(si, "SetFieldLocalStringText", "cmt_" + COL, colText);
        Console.WriteLine("         label text: \"" + colText + "\"");

        // ---------- 4. promote every node the load created for this element out of CREATE ----------
        object modify = Enum.Parse(ssT, "MODIFY");
        int promoted = 0;
        foreach (string p in new[] { "SpecField", "SpecHelpCode", "SpecProgRel" }) {
            object n = Prop(fsm, p);
            if (n != null && S(Prop(n, "Status")).IndexOf("CREATE") >= 0) { SetProp(n, "Status", modify); promoted++; }
        }
        foreach (var item in (IEnumerable)Prop(si, "fieldStrings")) {
            string nm = S(Prop(item, "Name"));
            if ((nm == "lbl_" + COL || nm == "cmt_" + COL) && S(Prop(item, "Status")).IndexOf("CREATE") >= 0) {
                SetProp(item, "Status", modify); promoted++;
            }
        }
        Console.WriteLine("step 3/4 : enriched; promoted " + promoted + " node(s) out of CREATE");

        // ---------- 5. save ----------
        Call(si, "SaveToTSD");
        Call(si, "SaveBinding");
        string tsdNew = ((XElement)Prop(si, "TSDElement")).ToString(SaveOptions.DisableFormatting);
        // SaveBinding() stores the serialised payload on the TzpManager; read it back rather
        // than re-serialising here, so the bytes are the designer's own.
        string bdxNew = Prop(tzp, "ElementBindings") as string;
        if (bdxNew == string.Empty) bdxNew = null;
        Console.WriteLine("step 5   : .tsd " + tsdNew.Length + " chars; .bdx " + (bdxNew == null ? "unchanged" : bdxNew.Length + " chars"));

        // ---------- 6. repack, rewriting only what changed ----------
        var repl = new Dictionary<string, byte[]> { { tsdEntry, Encoding.UTF8.GetBytes(tsdNew) } };
        repl[fdEntry] = Encoding.UTF8.GetBytes(fd4New);
        if (bdxEntry != null && bdxNew != null) repl[bdxEntry] = Encoding.UTF8.GetBytes(bdxNew);

        byte[] result = TzsRepacker.Repack(original, repl);
        File.WriteAllBytes(OUT_TZS, result);
        Console.WriteLine("step 6   : repacked " + original.Length + " -> " + result.Length + " bytes");
        Console.WriteLine("output   : " + OUT_TZS);
        Console.WriteLine();

        // ---------- 7. self-check: the designer must accept what we just wrote ----------
        Console.WriteLine("=== self-check: reload the produced file through the designer ===");
        try {
            object tzp2 = Activator.CreateInstance(tzpT, new object[] { OUT_TZS });
            object key2 = Prop(tzp2, "ProgramKey");
            ((IDictionary)Prop(SM, "tzpMap")).Add(key2, tzp2);
            object si2 = Call(siT, "Create", tzp2);
            SetProp(tzp2, "SpecificationInfo", si2);
            Console.WriteLine("  TzpManager + SpecificationInfo : OK");
            Console.WriteLine("  ProgramName  : " + Prop(tzp2, "ProgramName"));
            object d2 = Prop(si2, "FormSpeDictionary");
            int n2 = 0; foreach (DictionaryEntry de in (IDictionary)d2) n2++;
            Console.WriteLine("  FormSpeDictionary entries : " + n2);
        } catch (Exception ex) {
            Console.WriteLine("  RELOAD FAILED: " + ex.GetType().Name + ": " + ex.Message);
        }
    }
    static string S(object o) { return o == null ? "(null)" : o.ToString(); }
}
