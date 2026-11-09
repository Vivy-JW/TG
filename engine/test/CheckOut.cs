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

[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>Fresh-process load of a produced .tzs, exactly as the designer would.</summary>
class CheckOut
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
    const string WS      = @"D:\t100_wrok_dir\hengshuo\prd";

    static object Call(object target, string name, params object[] args) {
        Type t = target as Type ?? target.GetType();
        foreach (var m in t.GetMethods(BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static)) {
            if (m.Name != name || m.GetParameters().Length != args.Length) continue;
            try { return m.Invoke(target is Type ? null : target, args); }
            catch (TargetInvocationException ex) {
                System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw(); return null; }
        }
        throw new Exception("no method " + name + "/" + args.Length);
    }
    static object Prop(object o, string n) {
        if (o == null) return null;
        var t = o.GetType();
        var pi = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (pi != null) return pi.GetValue(o, null);
        var fi = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        return fi == null ? null : fi.GetValue(o);
    }
    static void SetProp(object o, string n, object v) {
        var t = o.GetType();
        var pi = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
        if (pi != null) { pi.SetValue(o, v, null); return; }
        t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance).SetValue(o, v);
    }
    static string S(object o) { return o == null ? "(null)" : o.ToString(); }

    [STAThread]
    static void Main(string[] args) {
        string target = args.Length > 0 ? args[0] : WS + @"\aapp320(c)_AIADD.tzs";
        // The name the added element ended up with. Container modes that route through
        // GetNewNameFromColumnForDetail (Table/Tree/ScrollGrid) name a field from the bare
        // column instead of table.column, so this has to be passable.
        string added = args.Length > 1 ? args[1] : "apca_t.apcaent";
        Console.WriteLine("checking: " + target + "  (" + new FileInfo(target).Length + " bytes)");
        Console.WriteLine("expecting: " + added);
        Console.WriteLine();

        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };

        Application app = new Application();
        DesignerLang.Merge(app, INSTALL);
        Assembly A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        Type smT = A.GetType("SpecDesignerCommon.SettingManager");
        Type modelT = A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel");
        object SM = Call(smT, "Get");
        object model = Call(modelT, "Create");
        SetProp(SM, "CurrentSetting", model);
        SetProp(Prop(model, "Connection"), "Workspace", WS);
        Call(SM, "LoadCommonData");

        Type prefT = null, pmT = null;
        foreach (Type t in A.GetTypes()) {
            if (t.Name == "PreferenceModel") prefT = t;
            if (t.Name == "PreferenceManager") pmT = t;
        }
        object prefs = Activator.CreateInstance(prefT);
        SetProp(prefs, "ValidateForm", false);
        var pi = pmT.GetProperty("Current", BindingFlags.Public|BindingFlags.Static);
        SetProp(pi.GetValue(null, null), "_preferenceModel", prefs);

        // 1) container version gate
        Type pmgrT = A.GetType("SpecDesignerCommon.PackageManager");
        Console.WriteLine("  ver entry          : " + Call(pmgrT, "SeekReleaseVersion", target));

        // 2) full model load, exactly as OpenSpecFiles would do it
        Type tzpT = A.GetType("SpecDesignerCommon.TzpManager");
        object tzp = Activator.CreateInstance(tzpT, new object[] { target });
        // GetChildNode -> XmlElement.IsCantDel calls GetTzpManger(key).IsNormalStyle,
        // which NREs unless the manager is registered (gap #3 in SPEC 8.9).
        object key = Prop(tzp, "ProgramKey");
        ((IDictionary)Prop(SM, "tzpMap")).Add(key, tzp);
        Console.WriteLine("  TzpManager.Type    : " + Prop(tzp, "Type"));
        Console.WriteLine("  ProgramName        : " + Prop(tzp, "ProgramName"));
        Console.WriteLine("  Tsd length         : " + S(Prop(tzp, "Tsd")).Length);
        Console.WriteLine("  .4fd length        : " + S(Prop(tzp, "GeneroFormString")).Length);

        Type siT = A.GetType("SpecDesignerCommon.SpecificationInfo");
        object si = Call(siT, "Create", tzp);
        Console.WriteLine("  SpecificationInfo  : " + (si == null ? "NULL (FAIL)" : "OK"));
        if (si == null) return;
        Console.WriteLine("  Env                : " + S(Prop(si, "Env")));
        object d = Prop(si, "FormSpeDictionary");
        int n = 0; foreach (DictionaryEntry de in (IDictionary)d) n++;
        Console.WriteLine("  FormSpeDictionary  : " + n + " entries");

        // 3) the added element must be present, with its spec node
        string col = added.Contains(".") ? added.Substring(added.IndexOf('.') + 1) : added;
        string lbl = "lbl_" + col, cmt = "cmt_" + col;
        object fsm = null;
        foreach (DictionaryEntry de in (IDictionary)d) if (S(de.Key) == added) fsm = de.Value;
        Console.WriteLine("  added element      : " + (fsm == null ? "MISSING (FAIL)" : "present"));
        if (fsm != null) {
            object sf = Prop(fsm, "SpecField");
            object hc = Prop(fsm, "SpecHelpCode");
            Console.WriteLine("    SpecField        : " + (sf == null ? "null" : S(Prop(sf, "Source"))));
            Console.WriteLine("    SpecHelpCode     : " + (hc == null ? "null" : S(Prop(hc, "Source"))));
        }

        // 4) the produced .tsd must carry the new nodes
        var tsd = (XElement)Prop(si, "TSDElement");
        Console.WriteLine();
        Console.WriteLine("  .tsd nodes for " + added + " / " + lbl + " / " + cmt + ":");
        foreach (var e in tsd.Descendants()) {
            string nm = (string)e.Attribute("name");
            if (nm == added || nm == lbl || nm == cmt)
                Console.WriteLine("    <" + e.Name.LocalName + "> " + string.Join(" ", e.Attributes().Select(a => a.Name + "=" + a.Value).ToArray()));
        }

        // 5) the .4fd must carry the new layout element + RecordField
        var fe = (XElement)Prop(si, "FormElement");
        var le = fe.Element("Form").Descendants().FirstOrDefault(e => (string)e.Attribute("name") == added);
        var rf = fe.Descendants("RecordField").FirstOrDefault(e => (string)e.Attribute("name") == added);
        Console.WriteLine();
        Console.WriteLine("  .4fd layout element: " + (le == null ? "MISSING (FAIL)" : "<" + le.Name.LocalName + "> fieldId=" + (string)le.Attribute("fieldId")));
        Console.WriteLine("  .4fd RecordField   : " + (rf == null ? "MISSING (FAIL)" : "fieldIdRef=" + (string)rf.Attribute("fieldIdRef") + " colName=" + (string)rf.Attribute("colName")));
    }
}
