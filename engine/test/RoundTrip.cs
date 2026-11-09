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

/// <summary>
/// Fixed-point check: reload a produced .tzs and let the *designer* regenerate the .4fd,
/// .tsd and .bdx from its own model, then compare with what is actually in the package.
///
/// This is the strongest self-contained question we can ask about a produced file: "if the
/// designer opened this and saved it, would it write the same thing?" Anything the model
/// adds or drops shows up as a named difference rather than as a guess about intent.
///
/// Comparison is canonical-XML equality (whitespace-insensitive), because the package's
/// .4fd is indented and XElement.ToString() is not.
/// </summary>
class RoundTrip
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
    // The workspace is per-module: each one has its own mta/ and <module>/tbl/*.tbl, and
    // TzpManager refuses to open a package that lives outside the configured one
    // (NotInCurrentWorkspaceException). batch.sh points this at the nearest ancestor
    // holding an mta/ directory.
    static string WS = Environment.GetEnvironmentVariable("TZSCLI_WS") ?? @"D:\t100_wrok_dir\hengshuo\prd";

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

    /// <summary>Canonical form: whitespace-insensitive, attribute order preserved.</summary>
    static string Canon(string xml) {
        try { return XElement.Parse(xml).ToString(SaveOptions.DisableFormatting); }
        catch { return null; }
    }

    static string Entry(byte[] zip, string suffix, out string name) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
            foreach (var e in z.Entries)
                if (e.FullName.EndsWith(suffix)) {
                    name = e.FullName;
                    using (var r = new StreamReader(e.Open(), Encoding.UTF8)) return r.ReadToEnd();
                }
        name = null; return null;
    }

    /// <summary>Node names in a .tsd, as "kind:name" so field/hfield/pfield are distinguishable.</summary>
    static List<string> TsdKeys(XElement xs) {
        var list = new List<string>();
        foreach (var e in xs.Descendants()) {
            // act ids live in the id attribute, not name; a <act> has no name at all.
            string kind = e.Name.LocalName;
            if (kind == "act") { list.Add("act:" + (string)e.Attribute("id")); continue; }
            if (kind == "field" || kind == "hfield" || kind == "pfield"
                || kind == "rfield" || kind == "sfield" || kind == "mlfield")
                list.Add(kind + ":" + (string)e.Attribute("name"));
        }
        return list;
    }

    /// <summary>
    /// Layout elements as name-paths, so a moved or mis-parented element shows up even when
    /// its own tag and name are unchanged. Duplicates are counted, not collapsed.
    /// </summary>
    static List<string> FormKeys(XElement element) {
        var list = new List<string>();
        foreach (var e in element.Descendants()) {
            var stack = new List<string>();
            for (var c = e; c != null && c != element.Parent; c = c.Parent)
                stack.Insert(0, (string)c.Attribute("name") ?? c.Name.LocalName);
            list.Add(string.Join("/", stack.ToArray()));
        }
        return list;
    }

    /// <summary>
    /// name-path -> attributes. Path-level comparison only says whether an element is there;
    /// this says whether its values are right. Canonical XML equality cannot serve: it also
    /// reports DIFFERS for a pure attribute-order reshuffle, which the designer does on every
    /// save, so it cannot tell a stale value from a reshuffle.
    ///
    /// fieldId is deliberately excluded: it is renumbered wholesale on every save and is not
    /// a persistent identifier (that is why addressing is by name). Comparing it flags every
    /// designer-authored package -- 20 of 89 -- for something that is not a defect.
    /// </summary>
    static Dictionary<string, Dictionary<string,string>> AttrMap(XElement element) {
        var map = new Dictionary<string, Dictionary<string,string>>();
        foreach (var e in element.Descendants()) {
            var stack = new List<string>();
            for (var c = e; c != null && c != element.Parent; c = c.Parent)
                stack.Insert(0, (string)c.Attribute("name") ?? c.Name.LocalName);
            var d = new Dictionary<string,string>();
            foreach (var a in e.Attributes()) {
                if (a.Name.LocalName == "fieldId") continue;
                d[a.Name.LocalName] = a.Value;
            }
            map[string.Join("/", stack.ToArray())] = d;
        }
        return map;
    }

    static void Compare(string label, string aText, string bText) {
        string ca = Canon(aText), cb = Canon(bText);
        Console.WriteLine("  " + label.PadRight(10) + ": " + (ca == cb ? "IDENTICAL" : "DIFFERS")
            + "   (" + (aText ?? "").Length + " vs " + (bText ?? "").Length + " chars)");
    }

    static string CodeTemplate(string tsd) {
        if (tsd == null) return "?";
        int i = tsd.IndexOf("<code_template");
        if (i < 0) return "(none)";
        int j = tsd.IndexOf("value=\"", i);
        if (j < 0) return "(none)";
        j += 7;
        int k = tsd.IndexOf('"', j);
        string v = k < 0 ? "" : tsd.Substring(j, k - j);
        return v.Length == 0 ? "(none)" : v;      // empty would shift the driver's columns
    }

    /// <summary>Everything after the file is read. Returns the machine-readable summary fields.</summary>
    static string Run(string target, byte[] zip, bool quiet) {
        if (!quiet) Console.WriteLine("round-trip: " + target + "  (" + zip.Length + " bytes)");

        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };

        Application app = new Application();
        DesignerLang.Merge(app, INSTALL);
        Assembly A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        object SM = Call(A.GetType("SpecDesignerCommon.SettingManager"), "Get");
        object model = Call(A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel"), "Create");
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
        SetProp(pmT.GetProperty("Current", BindingFlags.Public|BindingFlags.Static).GetValue(null, null), "_preferenceModel", prefs);

        string fdName, tsdName, bdxName;
        string fdIn  = Entry(zip, ".4fd", out fdName);
        string tsdIn = Entry(zip, ".tsd", out tsdName);
        string bdxIn = Entry(zip, ".bdx", out bdxName);
        string tpl = CodeTemplate(tsdIn);
        if (fdIn == null || tsdIn == null) throw new Exception("package has no .4fd or no .tsd");

        object tzp = Activator.CreateInstance(A.GetType("SpecDesignerCommon.TzpManager"), new object[] { target });
        object key = Prop(tzp, "ProgramKey");
        ((IDictionary)Prop(SM, "tzpMap")).Add(key, tzp);
        object si = Call(A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", tzp);
        SetProp(tzp, "SpecificationInfo", si);
        if (si == null) throw new Exception("SpecificationInfo.Create returned null");

        // ---- regenerate through the designer's own save path ----
        var fdOut  = (XElement)Call(si, "SaveToForm");
        var tsdOut = (XElement)Call(si, "SaveToTSD");
        Call(si, "SaveBinding");

        var kin = new HashSet<string>(TsdKeys(XElement.Parse(tsdIn)));
        var kout = new HashSet<string>(TsdKeys(tsdOut));
        var added = kout.Except(kin).OrderBy(x => x).ToList();
        var dropped = kin.Except(kout).OrderBy(x => x).ToList();

        var fin  = new HashSet<string>(FormKeys(XElement.Parse(fdIn).Element("Form")));
        var fout = new HashSet<string>(FormKeys(fdOut.Element("Form")));
        var fadd = fout.Except(fin).OrderBy(x => x).ToList();
        var fdrop = fin.Except(fout).OrderBy(x => x).ToList();

        string bdxOut = Prop(tzp, "ElementBindings") as string;
        var fe = (XElement)Prop(si, "FormElement");
        int elems = fe.Element("Form").DescendantsAndSelf().Count();

        // ---- attribute-value diff: is every element's value set what the model would write? ----
        // The path-level diff above only says whether an element is there. This says whether
        // its values are right -- it is what catches a stale attribute left behind by a text
        // splice, which the path diff and the canonical-XML compare both sail past.
        var ain = AttrMap(XElement.Parse(fdIn).Element("Form"));
        var aout = AttrMap(fdOut.Element("Form"));
        var staleSamples = new List<string>();
        int stale = 0;
        foreach (var kv in ain) {
            Dictionary<string,string> other;
            if (!aout.TryGetValue(kv.Key, out other)) continue;
            foreach (var a in kv.Value) {
                string ov;
                if (!other.TryGetValue(a.Key, out ov) || ov == a.Value) continue;
                stale++;
                if (staleSamples.Count < 6) staleSamples.Add(kv.Key + "." + a.Key + ": " + a.Value + " → " + ov);
            }
        }

        if (!quiet) {
            Compare(".4fd", fdIn,  fdOut.ToString(SaveOptions.DisableFormatting));
            Compare(".tsd", tsdIn, tsdOut.ToString(SaveOptions.DisableFormatting));
            Console.WriteLine("  " + ".bdx".PadRight(10) + ": " + (bdxIn == null ? "(absent in package)" :
                (Canon(bdxIn) == Canon(bdxOut) ? "IDENTICAL" : "DIFFERS")));
            Console.WriteLine("  .tsd spec nodes: in package " + kin.Count + ", regenerated " + kout.Count);
            Console.WriteLine("    model ADDS " + added.Count + (added.Count == 0 ? "" : ": " + string.Join(", ", added.Take(20).ToArray())));
            Console.WriteLine("    model DROPS " + dropped.Count + (dropped.Count == 0 ? "" : ": " + string.Join(", ", dropped.Take(20).ToArray())));
            Console.WriteLine("    .4fd paths ADDS " + fadd.Count + (fadd.Count == 0 ? "" : ": " + string.Join(", ", fadd.Take(10).ToArray())));
            Console.WriteLine("    .4fd paths DROPS " + fdrop.Count + (fdrop.Count == 0 ? "" : ": " + string.Join(", ", fdrop.Take(10).ToArray())));
            Console.WriteLine("    .4fd 属性值不符 " + stale + (stale == 0 ? "" : ": " + string.Join("; ", staleSamples.ToArray())));
            Console.WriteLine("  form layout elements: " + elems);
        }

        // field order: env | code_template | tsdAdds | tsdDrops | fdPathAdds | fdPathDrops
        //              | staleAttrs | tsdNodesIn | tsdNodesOut | layoutElems | sample
        return string.Join("|", new[] {
            S(Prop(si, "Env")), tpl,
            added.Count.ToString(), dropped.Count.ToString(),
            fadd.Count.ToString(), fdrop.Count.ToString(),
            stale.ToString(),
            kin.Count.ToString(), kout.Count.ToString(), elems.ToString(),
            Sample(added, dropped, fadd, fdrop, staleSamples)
        });
    }

    /// <summary>First few differing names, so a failing file is diagnosable from the TSV alone.</summary>
    static string Sample(List<string> a, List<string> b, List<string> c, List<string> d, List<string> stale) {
        var all = new List<string>();
        foreach (var x in a) all.Add("+tsd:" + x);
        foreach (var x in b) all.Add("-tsd:" + x);
        foreach (var x in c) all.Add("+fd:" + x);
        foreach (var x in d) all.Add("-fd:" + x);
        foreach (var x in stale) all.Add("stale:" + x);
        if (all.Count == 0) return "";
        string s = string.Join("; ", all.Take(6).ToArray());
        return s.Replace("|", "/");               // a name must not split the driver's columns
    }

    [STAThread]
    static void Main(string[] args) {
        string target = args.Length > 0 ? args[0] : WS + @"\aapp320(c)_AIADD.tzs";
        bool quiet = Environment.GetEnvironmentVariable("TZSCLI_QUIET") == "1";
        string status = "ok", err = "";
        // 10 fields: env, code_template, tsdAdds, tsdDrops, fdPathAdds, fdPathDrops,
        // tsdNodesIn, tsdNodesOut, layoutElems, sample. Kept positionally aligned on failure
        // too, or the batch driver's cut/read would shift every column.
        string fields = "|||||||||";
        try {
            byte[] zip = File.ReadAllBytes(target);
            fields = Run(target, zip, quiet);
        } catch (Exception ex) {
            status = "FAIL";
            var inner = ex;
            while (inner.InnerException != null) inner = inner.InnerException;
            err = inner.GetType().Name + ": " + (inner.Message ?? "");
            err = err.Replace("|", "/").Replace("\r", " ").Replace("\n", " ");
            if (err.Length > 160) err = err.Substring(0, 160);
            if (!quiet) Console.WriteLine("  FAILED: " + err);
        }
        Console.WriteLine();
        Console.WriteLine("SUMMARY|" + status + "|" + fields + "|" + target + "|" + err);
    }
}
