using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Reflection;
using System.Text;
using System.Threading;
using System.Windows;
using System.Windows.Markup;
using System.Xml.Linq;
using TzsCli;

[assembly: AssemblyVersion("1.0.0.251")]

/// <summary>
/// add_field, implemented on top of the designer's own layout engine.
///
/// UICreator decides everything: widget type (from the column's col_attr@widget), names,
/// label companions, placement, and the .bdx binding. We only splice the resulting
/// elements into the .4fd as text and let the designer regenerate the .tsd / .bdx.
///
/// Six drop targets, dispatched by ContainerType exactly as UICreator.Create does:
///
///   None        CreateNoneContainerWidget   flat siblings, two columns
///   Grid        CreateGridContainer         a new Grid holding the fields
///   Group       CreateGroupContainer        a new Group holding a new Grid
///   ScrollGrid  CreateScrollGridContainer   a new ScrollGrid, fields marked repeat
///   Table       CreateTableContainer        a new Table, fields carry LocalString
///   Tree        CreateTreeContainer         a new Tree (id/parentid/isnode/expanded phantoms)
///
/// The five container modes differ from None in one structural way: they return a *new
/// container element* rather than siblings, so the splice target is the new element and
/// its children ride along inside it. Only its own posX/posY need placing.
///
/// Note UICreator's creators are private static -- hence reflection, as elsewhere.
/// </summary>
class AddField
{
    static Assembly A, FE;
    static object SM, key, tzp, si;

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
    // The workspace is per-module: each has its own mta/ and <module>/tbl/*.tbl, and
    // TzpManager refuses a package from outside the configured one. batch-write.sh points
    // this at the nearest ancestor holding an mta/ directory.
    static string WS = Environment.GetEnvironmentVariable("TZSCLI_WS") ?? @"D:\t100_wrok_dir\hengshuo\prd";

    static readonly string[] CONTAINER_TYPES = { "None", "Grid", "Group", "ScrollGrid", "Table", "Tree" };

    const string DEFAULT_CONTAINER =
        "managedform/aapp320/mainlayout/condition/condition_page/vb_qbe"
        + "/hbox_8/group_aapp320main/vbox_1/hbox_3/group_ldcomp"
        + "/group_qbe_hbox/group_qbe_hbox_grid";

    static object Call(object t, string n, params object[] a) {
        Type tt = t as Type ?? t.GetType();
        foreach (var m in tt.GetMethods(BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance|BindingFlags.Static)) {
            if (m.Name != n || m.GetParameters().Length != a.Length) continue;
            try { return m.Invoke(t is Type ? null : t, a); }
            catch (TargetInvocationException ex) { System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw(); return null; }
        }
        throw new Exception("no " + n + "/" + a.Length + " on " + tt);
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
    static string Attr(XElement e, string n) {
        var a = e == null ? null : e.Attribute(n);
        return a == null ? null : a.Value;
    }

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

    /// <summary>Loads a .tzs into the designer, replacing any previous registration for the key.</summary>
    static object LoadDesigner(string tzsPath, string fd4Override) {
        var tzpT = A.GetType("SpecDesignerCommon.TzpManager");
        var t = Activator.CreateInstance(tzpT, new object[] { tzsPath });
        object k = Prop(t, "ProgramKey");
        var map = (IDictionary)Prop(SM, "tzpMap");
        if (map.Contains(k)) map.Remove(k);
        map.Add(k, t);
        var eam = Find(A, "EventAggregatorManager");
        if (!(bool)Call(eam, "ContainsKey", k)) Call(eam, "CreateInstance", k);
        if (fd4Override != null) SetProp(t, "GeneroFormString", fd4Override);
        // GetChildNode calls SetInitGridX/SetInitGridY, which deliberately throw when an
        // UndoRedoManager is registered. Loading wants none, enrichment wants one -- so
        // hide it for the duration of the load and put it back afterwards.
        var urmMap = (IDictionary)Prop(SM, "undoRedoManagerMap");
        object held = urmMap.Contains(k) ? urmMap[k] : null;
        if (held != null) urmMap.Remove(k);
        object s = Call(A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", t);
        if (held != null) urmMap[k] = held;
        SetProp(t, "SpecificationInfo", s);
        key = k; tzp = t; si = s;
        return s;
    }

    /// <summary>
    /// Loads with a watchdog, and gives up if the designer does not come back.
    ///
    /// SpecificationInfo's constructor can raise a modal DesignerMessageBox -- most notably
    /// DatabaseSourceViewModel reporting tables missing from the workspace's mta/tables.xml.
    /// There is no message pump here, so Dispatcher.Invoke blocks forever and the process
    /// simply hangs with nothing on stdout to say why.
    ///
    /// The load has to stay on the calling STA thread: Application.Current is thread-affine
    /// and SpecificationInfo.Create resolves resources through it, so running the load on a
    /// worker thread throws cross-thread instead of loading. Only the watchdog is off-thread,
    /// and it can only call Environment.Exit -- the loading thread is stuck inside a
    /// dispatcher frame and would never observe a cancellation flag.
    /// </summary>
    static object LoadWithWatchdog(string tzsPath, string fd4Override, int seconds) {
        var done = new ManualResetEvent(false);
        var watch = new Thread(delegate() {
            if (done.WaitOne(seconds * 1000)) return;
            Console.WriteLine();
            Console.WriteLine("!! 重载超时 " + seconds + "s —— 设计器多半弹了模态框（无界面下必挂）。");
            Console.WriteLine("   最常见原因：本工作区的 mta/tables.xml 缺少表单用到的表，");
            Console.WriteLine("   DatabaseSourceViewModel..ctor 会为此弹「找不到表格基础资料」。");
            Console.WriteLine("   检查：<工作区>/mta/tables.xml 是否登记了该表单引用到的每张表。");
            Console.Out.Flush();
            Environment.Exit(3);
        });
        watch.IsBackground = true;
        watch.Start();
        try { return LoadDesigner(tzsPath, fd4Override); }
        finally { done.Set(); }
    }

    [STAThread]
    static void Main(string[] args) {
        if (args.Length > 0 && (args[0] == "-h" || args[0] == "--help")) { Usage(); return; }

        string inTzs  = args.Length > 0 ? args[0] : WS + @"\aapp320(c).tzs";
        string outTzs = args.Length > 1 ? args[1] : WS + @"\aapp320(c)_AIADD.tzs";
        string ctype  = args.Length > 2 && args[2].Length > 0 ? args[2] : "None";
        string container = args.Length > 3 && args[3].Length > 0 ? args[3] : DEFAULT_CONTAINER;

        if (Array.IndexOf(CONTAINER_TYPES, ctype) < 0) {
            Console.WriteLine("unknown container type: " + ctype + "  (expected one of "
                + string.Join(", ", CONTAINER_TYPES) + ")");
            return;
        }

        var want = new List<string[]>();
        for (int i = 4; i + 1 < args.Length; i += 2) want.Add(new[] { args[i], args[i + 1] });
        if (want.Count == 0) want.Add(new[] { "apca_t", "apcaent" });

        byte[] original = File.ReadAllBytes(inTzs);
        string fdEntry = EntryName(original, ".4fd");
        string tsdEntry = EntryName(original, ".tsd");
        string bdxEntry = EntryName(original, ".bdx");
        Console.WriteLine("in  : " + inTzs + "  (" + original.Length + " bytes)");
        Console.WriteLine("out : " + outTzs);
        Console.WriteLine("容器 : " + ctype + "  →  " + container);
        Console.WriteLine("取字段: " + string.Join(", ", want.Select(w => w[0] + "." + w[1]).ToArray()));

        // ---------- bootstrap ----------
        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };
        var app = new Application();
        DesignerLang.Merge(app, INSTALL);
        A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        FE = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesigner.FormEditor.dll"));
        var smT = A.GetType("SpecDesignerCommon.SettingManager");
        var mdlT = A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel");
        SM = Call(smT, "Get");
        object mdl = Call(mdlT, "Create");
        SetProp(SM, "CurrentSetting", mdl);
        SetProp(Prop(mdl, "Connection"), "Workspace", WS);
        Call(SM, "LoadCommonData");
        object prefs = Activator.CreateInstance(Find(A, "PreferenceModel"));
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(Find(A, "PreferenceManager"), "Current"), "_preferenceModel", prefs);

        // ---------- pass 1: load, so UICreator has a context to work in ----------
        // Both loads go through the watchdog: the designer raises modal dialogs for a few
        // conditions, and a dialog with no message pump is an unexplained hang.
        int reloadTimeout = 90;
        int.TryParse(Environment.GetEnvironmentVariable("TZSCLI_RELOAD_TIMEOUT") ?? "90", out reloadTimeout);
        LoadWithWatchdog(inTzs, null, reloadTimeout);
        var urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
        object urm = Activator.CreateInstance(Find(urf, "UndoRedoManager"), new object[] { 100 });
        Call(urm, "Init");
        ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urm;   // UICreator's layout setters need it
        Console.WriteLine("[1] designer context loaded (UndoRedoManager registered after Create)");

        // ---------- pass 2: let UICreator build the elements ----------
        Type cvmT = Find(FE, "ContainerViewModel"), pacT = Find(FE, "PrepareAddColumn"),
             ucT  = Find(FE, "UICreator"),         ctT  = Find(FE, "ContainerType");
        object ct = Activator.CreateInstance(ctT); SetProp(ct, "Type", ctype);
        object cvm = Activator.CreateInstance(cvmT);
        SetProp(cvm, "Container", ct);
        SetProp(cvm, "MaximumWidthStrValue", "20");
        SetProp(cvm, "NumberOfFieldsStrValue", "1");
        if (ctype == "ScrollGrid") {
            SetProp(cvm, "RepeatRowCountStrValue", "2");
            SetProp(cvm, "RepeatColumnCountStrValue", "1");
        }

        var tch = A.GetType("SpecDesignerCommon.Helpers.TableColumnHelper");
        var listT = typeof(List<>).MakeGenericType(pacT);
        var fields = (IList)Activator.CreateInstance(listT);
        foreach (var w in want) {
            object pac = Activator.CreateInstance(pacT);
            SetProp(pac, "Table", w[0]);
            SetProp(pac, "Column", w[1]);
            object colInfo  = Call(tch, "GetColumnInfo", w[0], w[1]);
            object colField = Call(tch, "GetColField", w[0], w[1]);
            string labelText = colInfo is XElement ? Attr((XElement)colInfo, "text") : w[1];
            if (string.IsNullOrEmpty(labelText)) labelText = w[1];

            // A column whose <col_attr><field> carries no widget has no form-widget mapping --
            // type_t (the "data type reference" table) is entirely made of those. UICreator
            // would NRE deep inside ComponentFactory.ModFdInfo.IsIncludeAttribute, so refuse
            // here where we can say why. The designer cannot drag such a column either.
            string widget = colField is XElement ? Attr((XElement)colField, "widget") : null;
            if (string.IsNullOrEmpty(widget)) {
                Console.WriteLine("!! " + w[0] + "." + w[1] + " 没有控件映射（col_attr@widget 为空），"
                    + "设计器无法把它变成控件 —— 跳过");
                continue;
            }

            SetProp(pac, "Description", labelText);
            SetProp(pac, "Label", labelText);
            SetProp(pac, "Widget", widget);
            if (colField is XElement) SetProp(pac, "Width", Attr((XElement)colField, "widget_width"));
            Console.WriteLine("    " + w[0] + "." + w[1] + "  → 表名「" + labelText + "」 widget="
                + Prop(pac, "Widget") + " width=" + Prop(pac, "Width"));
            fields.Add(pac);
        }
        if (fields.Count == 0) { Console.WriteLine("没有可用字段，终止"); return; }

        // None is CreateNoneContainerWidget; the rest are Create<Type>Container. All are
        // private static with the same (ContainerViewModel, IEnumerable<PrepareAddColumn>, PackageKey).
        string creator = ctype == "None" ? "CreateNoneContainerWidget" : "Create" + ctype + "Container";
        var produced = (IEnumerable)Call(ucT, creator, cvm, fields, key);

        // Walk the whole produced subtree, not just the returned roots: the container modes
        // return one wrapper whose fields hang off it, and those fields are what bind.
        var tree = new List<object>();
        foreach (var e in produced) WalkDesigner(e, tree);

        var news = new List<XElement>();          // top-level elements to splice
        foreach (var e in produced) {
            // ToXML() (not Source): it is what SaveToForm writes, and unlike Source it
            // carries the children -- which is the whole point of the container modes.
            news.Add((XElement)Call(e, "ToXML"));
        }

        // UICreator links label<->widget only where the designer actually binds them (None
        // mode does; ScrollGrid's bare label does not). Read the link off the element instead
        // of re-deriving it from names, or we would invent bindings the designer never makes.
        // The pair is also always normalised label-first: AddSpecBinding puts its first
        // argument in ObjectName1, and TBinding.Equals ignores order, so whichever direction
        // is added first is the one that survives.
        var bindPairs = new List<string[]>();
        var seenPair = new HashSet<string>();
        foreach (var e in tree) {
            object be = Prop(e, "BindElement");
            if (be == null) continue;
            object a = e, b = be;
            if (S(Prop(e, "NodeName")) != "Label" && S(Prop(be, "NodeName")) == "Label") { a = be; b = e; }
            string n1 = S(Prop(a, "Name")), n2 = S(Prop(b, "Name"));
            string k1 = n1 + "|" + n2, k2 = n2 + "|" + n1;
            if (seenPair.Contains(k1) || seenPair.Contains(k2)) continue;
            seenPair.Add(k1);
            bindPairs.Add(new[] { n1, n2 });
        }

        Console.WriteLine("[2] UICreator." + creator + " produced " + news.Count + " element(s)");
        if (Environment.GetEnvironmentVariable("TZSCLI_DEBUG") == "1") {
            foreach (var d in tree)
                Console.WriteLine("    [dbg] <" + S(Prop(d, "NodeName")) + "> designerName=<"
                    + S(Prop(d, "Name")) + "> xmlName=<" + Attr((XElement)Call(d, "ToXML"), "name") + ">");
        }

        // ---------- pass 3: splice into the .4fd as text ----------
        string fd4 = EntryText(original, fdEntry);
        if (container == "@auto") {
            container = FindAutoTarget(fd4);
            if (container == null) { Console.WriteLine("!! @auto found no container to drop into"); return; }
            Console.WriteLine("落点 : @auto → " + container);
        }
        var w4 = FormWriter.Load(fd4);
        int row = w4.NextFreeRow(container);
        bool wrap = ctype != "None";
        int bottom = row;

        foreach (var el in news) {
            if (el.Attribute("posY") != null) {
                // Container modes: the new container is a unit, so place it outright.
                // None mode: UICreator already laid the fields out relative to a fresh
                // container (rows from 1), so shift its row numbers below what is there.
                int y = 1;
                if (!wrap) int.TryParse(Attr(el, "posY") ?? "1", out y);
                el.SetAttributeValue("posY", (wrap ? row : y + row - 1).ToString());
            }
            if (wrap && el.Attribute("posX") != null) el.SetAttributeValue("posX", "1");

            w4.AddNode(container, el);
            int ey = row, eh = 1;
            int.TryParse(Attr(el, "posY") ?? "0", out ey);
            int.TryParse(Attr(el, "gridHeight") ?? "1", out eh);
            if (ey + eh > bottom) bottom = ey + eh;
        }
        w4.EnsureHeight(container, bottom);
        string fd4New = w4.Render();

        // Diagnostic escape hatch: the reload in pass 4 is where a designer modal dialog can
        // hang the process, and without the spliced text there is nothing to diff against.
        string dump = Environment.GetEnvironmentVariable("TZSCLI_DUMP_FD4");
        if (!string.IsNullOrEmpty(dump)) {
            File.WriteAllText(dump, fd4New, Encoding.UTF8);
            Console.WriteLine("    [dump] spliced .4fd written to " + dump);
        }

        Console.WriteLine("[3] spliced at row " + row + ";  .4fd " + fd4.Length + " → " + fd4New.Length
            + " chars (+" + (fd4New.Length - fd4.Length) + ")");
        foreach (var el in news)
            Console.WriteLine("    <" + el.Name.LocalName + " name=\"" + Attr(el, "name") + "\" posX="
                + Attr(el, "posX") + " posY=" + Attr(el, "posY") + " w=" + Attr(el, "gridWidth")
                + " h=" + Attr(el, "gridHeight") + ">  children=" + el.Elements().Count());

        // ---------- pass 4: reload the edited .4fd and finish the spec side ----------
        object si2 = LoadWithWatchdog(inTzs, fd4New, reloadTimeout);
        var dict = (IDictionary)Prop(si2, "FormSpeDictionary");
        Console.WriteLine("[4] reloaded; FormSpeDictionary=" + dict.Count + " entries");

        Type ssT = Find(A, "SpecStatus");
        object modify = Enum.Parse(ssT, "MODIFY");

        // Enrich every data-bound node the new subtree introduced, reading table/column off
        // the element itself -- detail containers name their fields from the bare column
        // (GetNewNameFromColumnForDetail), so reconstructing the name would guess wrong.
        // Empty, not just absent: AttachDefaultAttributes writes colName="" on every element,
        // and UICreator's Tree scaffolding (id/parentid/isnode/expanded) carries it that way.
        int enriched = 0;
        foreach (var el in news) {
            foreach (var d in el.DescendantsAndSelf()) {
                string tab = Attr(d, "sqlTabName"), col = Attr(d, "colName"), nm = Attr(d, "name");
                if (string.IsNullOrEmpty(tab) || string.IsNullOrEmpty(col) || nm == null) continue;
                object fsm = dict.Contains(nm) ? dict[nm] : null;
                if (fsm == null) { Console.WriteLine("    !! " + nm + " not registered"); continue; }
                Enrich(si2, fsm, tab, col);
                enriched++;
                Console.WriteLine("    enriched " + nm + "  (" + tab + "." + col + ")");
            }
        }

        // ToXml() silently drops anything still marked CREATE, so promote the whole new
        // subtree by name rather than guessing which node kinds it produced.
        var newNames = new HashSet<string>();
        foreach (var el in news) foreach (var d in el.DescendantsAndSelf()) {
            string n = Attr(d, "name"); if (n != null) newNames.Add(n);
        }
        if (Environment.GetEnvironmentVariable("TZSCLI_DEBUG") == "1") {
            foreach (var nm in newNames)
                if (dict.Contains(nm))
                    Console.WriteLine("    [dbg] " + nm + "  SpecNodeType=" + S(Prop(dict[nm], "SpecNodeType")));
        }
        int promoted = Promote(si2, dict, newNames, modify);
        Console.WriteLine("    " + enriched + " node(s) enriched, " + promoted + " promoted out of CREATE");

        foreach (var p in bindPairs) {
            object fa = dict.Contains(p[0]) ? dict[p[0]] : null;
            object fb = dict.Contains(p[1]) ? dict[p[1]] : null;
            if (fa == null || fb == null) {
                Console.WriteLine("    !! binding skipped: " + p[0] + " ↔ " + p[1] + " (not in FormSpeDictionary)");
                continue;
            }
            Call(si2, "AddSpecBinding", Prop(fa, "GeneroComponent"), Prop(fb, "GeneroComponent"));
            Console.WriteLine("    bound " + p[0] + " ↔ " + p[1]);
        }

        Call(si2, "SaveToTSD");
        Call(si2, "SaveBinding");
        string tsdNew = ((XElement)Prop(si2, "TSDElement")).ToString(SaveOptions.DisableFormatting);
        string bdxNew = Prop(tzp, "ElementBindings") as string;
        Console.WriteLine("[5] .tsd " + tsdNew.Length + " chars;  .bdx "
            + (bdxNew == null ? "(unchanged)" : bdxNew.Length + " chars"));

        // ---------- pass 5: repack, rewriting only what changed ----------
        var repl = new Dictionary<string, byte[]> {
            { fdEntry, Encoding.UTF8.GetBytes(fd4New) },
            { tsdEntry, Encoding.UTF8.GetBytes(tsdNew) }
        };
        if (bdxEntry != null && !string.IsNullOrEmpty(bdxNew)) repl[bdxEntry] = Encoding.UTF8.GetBytes(bdxNew);
        byte[] result = TzsRepacker.Repack(original, repl);
        File.WriteAllBytes(outTzs, result);
        Console.WriteLine("[6] repacked: " + original.Length + " → " + result.Length + " bytes; rewritten entries: "
            + string.Join(", ", repl.Keys.ToArray()));
    }

    static void Usage() {
        Console.WriteLine("AddField.exe <in.tzs> <out.tzs> [containerType] [containerPath] [table column]...");
        Console.WriteLine();
        Console.WriteLine("  containerType   " + string.Join(" | ", CONTAINER_TYPES) + "   (default None)");
        Console.WriteLine("  containerPath   name path of the drop target inside the form");
        Console.WriteLine("  table column    one or more column references (default: apca_t apcaent)");
        Console.WriteLine();
        Console.WriteLine("example:");
        Console.WriteLine("  AddField.exe aapp320(c).tzs out.tzs Table <path> apca_t apcaent apca_t apcacomp");
    }

    /// <summary>
    /// Moves the newly-added nodes out of CREATE.
    ///
    /// AbstractSpecNode.ToXml() returns null for CREATE nodes -- they are load-time
    /// transients -- so a new field would vanish from the .tsd without this.
    ///
    /// But not every element the load invents a node for deserves one: UICreator's Tree
    /// brings five scaffolding children (id/parentid/isnode/expanded, plus the Edit named
    /// "name") that are NON_DATABASE -- colName is the empty string -- and no Tree in the
    /// corpus carries .tsd nodes for them. So the gate is the same predicate
    /// SpecNodeTransform.TransformFieldType uses: a non-empty colName.
    /// </summary>
    static int Promote(object si2, IDictionary dict, HashSet<string> names, object modify) {
        int n = 0;
        foreach (DictionaryEntry de in dict) {
            if (!names.Contains(S(de.Key))) continue;
            object fsm = de.Value;
            // XmlElement exposes its attributes through GetAttribute(key), not as properties,
            // so Prop() finds nothing here. And not S(...) -- S renders null as "(null)",
            // which is non-empty and would let everything through.
            object comp = Prop(fsm, "GeneroComponent");
            if (comp == null) continue;
            object colName = Call(comp, "GetAttribute", "colName");
            if (colName == null || colName.ToString().Length == 0) continue;
            // every spec-node slot a FormSpecModel can hold
            foreach (string p in new[] { "SpecField", "SpecHelpCode", "SpecProgRel", "SpecAction",
                                         "SpecReference", "SpecMultiLang", "SpecTree", "SpecItem" }) {
                object node = Prop(fsm, p);
                if (node == null) continue;
                if (S(Prop(node, "Status")).IndexOf("CREATE") < 0) continue;
                SetProp(node, "Status", modify);
                n++;
            }
        }
        // sfields (the lbl_/cmt_ field strings) are a flat list, not in FormSpeDictionary
        object fss = Prop(si2, "fieldStrings");
        if (fss is IEnumerable) {
            foreach (var item in (IEnumerable)fss) {
                if (!names.Contains(S(Prop(item, "Name")))) continue;
                if (S(Prop(item, "Status")).IndexOf("CREATE") < 0) continue;
                SetProp(item, "Status", modify);
                n++;
            }
        }
        return n;
    }

    /// <summary>Depth-first over the designer's own XmlElement tree (Nodes), not the XML.</summary>
    static void WalkDesigner(object e, List<object> acc) {
        acc.Add(e);
        var nodes = Prop(e, "Nodes") as IEnumerable;
        if (nodes == null) return;
        foreach (var c in nodes) WalkDesigner(c, acc);
    }

    /// <summary>
    /// Picks the drop target for "@auto": the first populated Grid, else the first populated
    /// element of the designer's own drop-container whitelist (Grid/HBox/VBox/Group/Folder/Page).
    ///
    /// The path cannot be hardcoded per file: the root element's name attribute is
    /// "managedform" in some packages and "ManagedForm" in others, and every form's
    /// container tree differs. ElementIndex already resolves name-paths, so use it.
    /// </summary>
    static string FindAutoTarget(string fd4Text) {
        var idx = ElementIndex.Build(fd4Text);
        string[] whitelist = { "Grid", "Group", "Folder", "Page", "HBox", "VBox" };
        for (int pass = 0; pass < 2; pass++) {
            foreach (var e in idx.All) {
                if (e.Children.Count == 0) continue;
                if (pass == 0 ? e.Tag != "Grid" : Array.IndexOf(whitelist, e.Tag) < 0) continue;
                return e.Path;
            }
        }
        return null;
    }

    static void Enrich(object si2, object fsm, string table, string col) {
        var tch = A.GetType("SpecDesignerCommon.Helpers.TableColumnHelper");
        object sf = Prop(fsm, "SpecField");
        if (sf == null) return;
        object ci  = Call(tch, "GetColumnInfo", table, col);
        object cai = Call(tch, "GetColumnAttrInfo", table, col);
        string widget = S(Prop(Prop(fsm, "GeneroComponent"), "NodeName"));
        Call(sf, "SetAttribute", "widget", widget);
        if (ci is XElement) {
            Call(sf, "SetAttribute", "attribute", Attr((XElement)ci, "attribute"));
            Call(sf, "SetAttribute", "type",      Attr((XElement)ci, "type"));
            Call(sf, "SetAttribute", "req",       Attr((XElement)ci, "req"));
        }
        if (cai is XElement)
            foreach (string k in new[] { "i_zoom", "c_zoom", "default", "max", "min", "chk_ref", "items" })
                Call(sf, "SetAttribute", k, Attr((XElement)cai, k));
        Call(sf, "SetAttribute", "Name", table + "." + col);      // the capital-N mirror
    }
}
