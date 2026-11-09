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
/// modify / delete, on top of the designer's own commands.
///
/// Unlike add_field this needs only ONE load, because both operations act on elements the
/// model already knows about -- add_field needed a second pass only so the designer could
/// mint spec nodes for elements it had never seen.
///
///   set <elementPath> <nodeKind>:<attr> <value>
///   del <elementPath>
///
/// Where add_field let UICreator decide everything, here the designer decides too -- by
/// running its own command:
///
///   modify -> SpecAttributeUndoRedoCommand   sets the spec attribute AND pushes it onto the
///                                            paired form element (TransformRequired,
///                                            TransformCanEdit, TransformTableColumn, ...),
///                                            and marks the node MODIFY via OnPropertyChanged
///   delete -> DeleteComponentsUndoRedoCommand is not used directly (it drives the live view
///              tree); the same three effects are applied: ClearBinding, then
///              SpecificationInfo.Remove(name), which clones a name-only tombstone with
///              status="d" and drops the RecordField.
///
/// The layout side stays a text splice: we diff the form element's attributes across the
/// command and write only what moved, so a one-attribute change stays a few bytes.
/// </summary>
class Edit
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
    static string WS = Environment.GetEnvironmentVariable("TZSCLI_WS") ?? @"D:\t100_wrok_dir\hengshuo\prd";

    // FormSpecModel slots, in the order a <field>-carrying element is worth trying.
    static readonly Dictionary<string,string> NODE_SLOTS = new Dictionary<string,string> {
        { "field",  "SpecField" },   { "hfield", "SpecHelpCode" }, { "pfield", "SpecProgRel" },
        { "rfield", "SpecReference" }, { "mlfield", "SpecMultiLang" }, { "tree", "SpecTree" },
        { "act",    "SpecAction" },
    };

    /// <summary>
    /// Reflection invoke that disambiguates overloads by argument type, not just arity.
    /// SpecificationInfo.Remove has both Remove(string) and a one-arg Remove(SpecActionNode),
    /// and picking by arity alone lands on whichever GetMethods() happened to return first.
    /// </summary>
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
        if (loose != null) return Invoke(loose, t, a);   // let the binder produce its own error
        throw new Exception("no " + n + "/" + a.Length + " on " + tt);
    }
    static object Invoke(MethodInfo m, object t, object[] a) {
        try { return m.Invoke(t is Type ? null : t, a); }
        catch (TargetInvocationException ex) { System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw(); return null; }
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

    /// <summary>Watchdog for the designer's modal dialogs -- see AddField.LoadWithWatchdog.</summary>
    static object LoadWithWatchdog(string tzsPath, int seconds) {
        var done = new ManualResetEvent(false);
        var watch = new Thread(delegate() {
            if (done.WaitOne(seconds * 1000)) return;
            Console.WriteLine();
            Console.WriteLine("!! 加载超时 " + seconds + "s —— 设计器多半弹了模态框（无界面下必挂）。");
            Console.WriteLine("   最常见原因：<工作区>/mta/tables.xml 缺少表单用到的表。");
            Console.Out.Flush();
            Environment.Exit(3);
        });
        watch.IsBackground = true;
        watch.Start();
        try {
            var t = Activator.CreateInstance(A.GetType("SpecDesignerCommon.TzpManager"), new object[] { tzsPath });
            object k = Prop(t, "ProgramKey");
            var map = (IDictionary)Prop(SM, "tzpMap");
            if (map.Contains(k)) map.Remove(k);
            map.Add(k, t);
            var eam = Find(A, "EventAggregatorManager");
            if (!(bool)Call(eam, "ContainsKey", k)) Call(eam, "CreateInstance", k);
            object s = Call(A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", t);
            SetProp(t, "SpecificationInfo", s);
            key = k; tzp = t; si = s;
            return s;
        } finally { done.Set(); }
    }

    /// <summary>
    /// The model's element for a .4fd name-path. FormNode is the &lt;Form&gt;, which is the
    /// second path segment (the first is the ManagedForm root), so we match from there and
    /// check the segment rather than trusting it.
    /// </summary>
    static object FindByPath(string path) {
        string[] parts = path.Split('/');
        object formNode = Prop(si, "FormNode");
        if (formNode == null || parts.Length < 2) return null;
        if (S(Prop(formNode, "Name")) != parts[1]) return null;
        string prefix = parts[0] + "/" + parts[1];
        return Walk(formNode, prefix, path);
    }
    static object Walk(object el, string here, string target) {
        if (here == target) return el;
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return null;
        foreach (var c in nodes) {
            object nv = Prop(c, "Name");                 // not S(): S(null) == "(null)"
            string seg = nv == null ? null : nv.ToString();
            if (string.IsNullOrEmpty(seg)) seg = S(Prop(c, "NodeName"));
            object hit = Walk(c, here + "/" + seg, target);
            if (hit != null) return hit;
        }
        return null;
    }

    /// <summary>
    /// name-path -> tabIndex for every element under the form, addressed exactly the way
    /// FormWriter addresses them: the document root's name, then the Form's name, then down.
    /// </summary>
    static Dictionary<string,string> TabMap(FormWriter w4) {
        var map = new Dictionary<string,string>();
        object formNode = Prop(si, "FormNode");
        if (formNode == null || w4.Index.All.Count == 0) return map;
        WalkTab(formNode, w4.Index.All[0].Name + "/" + S(Prop(formNode, "Name")), map);
        return map;
    }
    static void WalkTab(object el, string here, Dictionary<string,string> map) {
        string t = (string)Call(el, "GetAttribute", "tabIndex");
        if (t != null) map[here] = t;
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return;
        foreach (var c in nodes) {
            object nv = Prop(c, "Name");
            string seg = nv == null ? null : nv.ToString();
            if (string.IsNullOrEmpty(seg)) seg = S(Prop(c, "NodeName"));
            WalkTab(c, here + "/" + seg, map);
        }
    }

    static void WalkTab2(object el, string here, List<string> acc) {
        acc.Add(here);
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return;
        foreach (var c in nodes) {
            object nv = Prop(c, "Name");
            string seg = nv == null ? null : nv.ToString();
            if (string.IsNullOrEmpty(seg)) seg = S(Prop(c, "NodeName"));
            WalkTab2(c, here + "/" + seg, acc);
        }
    }

    static Dictionary<string,string> Attrs(object el) {
        var d = new Dictionary<string,string>();
        var names = Prop(el, "Attributes") as IEnumerable;
        if (names == null) return d;
        foreach (var n in names) {
            string k = n.ToString();
            d[k] = S(Call(el, "GetAttribute", k));
        }
        return d;
    }

    [STAThread]
    static void Main(string[] args) {
        if (args.Length > 0 && (args[0] == "-h" || args[0] == "--help")) { Usage(); return; }

        // info is read-only: `Edit.exe info <in.tzs> [<elementPath>]` prints JSON and writes
        // nothing, so it has no out argument and must keep stdout clean of prose.
        bool infoMode = args.Length > 0 && args[0] == "info";
        // 3 is enough for `tab` with no paths (document-order renumber); info needs only 2; the
        // branches validate their own arity beyond that.
        if (!infoMode && args.Length < 3) { Usage(); return; }
        if (infoMode && args.Length < 2) { Usage(); return; }

        string inTzs  = infoMode ? args[1] : args[0];
        string outTzs = infoMode ? null   : args[1];
        string op     = infoMode ? "info" : args[2];
        // del/wrap take every remaining argument as a path: a field is a widget plus its
        // label companion, and deleting one without the other leaves an orphan.
        // wrap also names the box type first, so its paths start one argument later.
        string boxType = op == "wrap" && args.Length > 3 ? args[3] : null;
        string[] paths = args.Skip(op == "wrap" ? 4 : 3).ToArray();
        string path   = paths.Length > 0 ? paths[0] : null;
        string spec   = op == "set" && args.Length > 4 ? args[4] : null;
        string value  = op == "set" && args.Length > 5 ? args[5] : null;
        if (op != "set") { spec = null; value = null; }

        byte[] original = File.ReadAllBytes(inTzs);
        string fdEntry = EntryName(original, ".4fd");
        string tsdEntry = EntryName(original, ".tsd");
        string bdxEntry = EntryName(original, ".bdx");
        if (!infoMode) Console.WriteLine("in  : " + inTzs + "  (" + original.Length + " bytes)");
        if (!infoMode) Console.WriteLine("out : " + outTzs);
        if (!infoMode) Console.WriteLine("op  : " + op + (boxType == null ? "" : " " + boxType)
            + (spec == null ? "" : "  " + spec + "=" + value));
        if (!infoMode) foreach (string pp in paths) Console.WriteLine("      " + pp);

        AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
            string p = Path.Combine(INSTALL, new AssemblyName(e.Name).Name + ".dll");
            return File.Exists(p) ? Assembly.LoadFrom(p) : null; };
        var app = new Application();
        DesignerLang.Merge(app, INSTALL);
        A = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        FE = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesigner.FormEditor.dll"));
        SM = Call(A.GetType("SpecDesignerCommon.SettingManager"), "Get");
        object mdl = Call(A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel"), "Create");
        SetProp(SM, "CurrentSetting", mdl);
        SetProp(Prop(mdl, "Connection"), "Workspace", WS);
        Call(SM, "LoadCommonData");
        object prefs = Activator.CreateInstance(Find(A, "PreferenceModel"));
        SetProp(prefs, "ValidateForm", false);
        SetProp(Prop2(Find(A, "PreferenceManager"), "Current"), "_preferenceModel", prefs);

        int tmo = 90;
        int.TryParse(Environment.GetEnvironmentVariable("TZSCLI_RELOAD_TIMEOUT") ?? "90", out tmo);
        LoadWithWatchdog(inTzs, tmo);
        if (infoMode) {
            // `info <in.tzs> [<elementPath>]` dumps a subtree; `--find <代号>` resolves a
            // component code to the path the other six operations take.
            string findTerm = null, pathFilter = null;
            for (int i = 2; i < args.Length; i++) {
                if (args[i] == "--find") {
                    if (i + 1 >= args.Length) {
                        Err("info --find: 需要一个控件代号");
                        Environment.Exit(2);
                    }
                    findTerm = args[++i];
                } else pathFilter = args[i];
            }
            if (findTerm != null && pathFilter != null) {
                Err("info: <elementPath> 与 --find 只能给一个");
                Environment.Exit(2);
            }
            if (findTerm != null) EmitFind(original, fdEntry, findTerm);
            else EmitInfo(original, fdEntry, tsdEntry, pathFilter);
            return;
        }
        Console.WriteLine("[1] loaded; FormSpeDictionary=" + Count(si) + " entries");

        string fd4 = EntryText(original, fdEntry);
        var w4 = FormWriter.Load(fd4);
        int changed = 0;
        if (op == "set") {
            if (w4.Index.ByPath(path) == null) { Console.WriteLine("!! 路径不存在: " + path); return; }
            object formEl = FindByPath(path);
            if (formEl == null) { Console.WriteLine("!! 模型里找不到该元素（路径对吗？）"); return; }
            object fsm = Call(si, "FindNodeByName", S(Prop(formEl, "Name")));
            if (fsm == null) { Console.WriteLine("!! 该元素没有规格模型: " + S(Prop(formEl, "Name"))); return; }
            Console.WriteLine("[2] <" + S(Prop(formEl, "NodeName")) + "> " + S(Prop(formEl, "Name")));
            if (spec == null || value == null) { Usage(); return; }
            int colon = spec.IndexOf(':');
            string kind = colon < 0 ? "field" : spec.Substring(0, colon);
            string attr = colon < 0 ? spec : spec.Substring(colon + 1);
            if (!NODE_SLOTS.ContainsKey(kind)) { Console.WriteLine("!! 未知节点类型: " + kind); return; }

            object node = Prop(fsm, NODE_SLOTS[kind]);
            if (node == null) { Console.WriteLine("!! 该元素没有 " + kind + " 节点"); return; }

            string old = S(Prop(node, "Source") is XElement ? Attr((XElement)Prop(node, "Source"), attr) : null);
            if (old == "(null)") old = null;
            if (old == value) {
                // AddAttributeChanged drops old==new, so the command would be a no-op that
                // still flips Status to MODIFY. Say so rather than report a silent success.
                Console.WriteLine("!! " + kind + "." + attr + " 已经是 \"" + value + "\"，无需修改");
                return;
            }

            // The command both writes the spec attribute and pushes it onto the form element
            // (TransformRequired / TransformCanEdit / TransformTableColumn / ...), and its
            // OnPropertyChanged is what sets Status |= MODIFY. Re-deriving any of that here
            // would be guessing at mapping tables the designer already owns.
            var urf = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
            object urm = Activator.CreateInstance(Find(urf, "UndoRedoManager"), new object[] { 100 });
            Call(urm, "Init");
            ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urm;

            var before = Attrs(formEl);
            Type cmdT = Find(A, "SpecAttributeUndoRedoCommand");
            object cmd = Activator.CreateInstance(cmdT, new object[] { node });
            Call(cmd, "AddAttributeChanged", attr, old, value);
            Call(urm, "AddThenExecute", cmd);

            var after = Attrs(formEl);
            foreach (var kv in after) {
                string was;
                if (before.TryGetValue(kv.Key, out was) && was == kv.Value) continue;
                w4.SetAttribute(path, kv.Key, kv.Value);
                changed++;
                Console.WriteLine("    .4fd  " + kv.Key + ": " + (was ?? "(absent)") + " → " + kv.Value);
            }
            Console.WriteLine("[3] " + kind + "." + attr + " = " + value
                + "（规格已置为 u）；表单元素动了 " + changed + " 个属性");
            if (Environment.GetEnvironmentVariable("TZSCLI_DEBUG") == "1") {
                foreach (string k in new[] { "style", "required", "notNull", "noEntry", "sqlTabName", "colName", "title" })
                    Console.WriteLine("    [dbg] " + k + " = <" + S(Call(formEl, "GetAttribute", k)) + ">");
                object sf2 = Prop(fsm, NODE_SLOTS[kind]);
                Console.WriteLine("    [dbg] spec." + NODE_SLOTS[kind] + " status=" + S(Prop(sf2, "Status"))
                    + " req=<" + S(Call(sf2, "GetAttribute", attr)) + ">");
                var names2 = Prop(formEl, "Attributes") as IEnumerable;
                int cn2 = 0; if (names2 != null) foreach (var _ in names2) cn2++;
                Console.WriteLine("    [dbg] element attribute count = " + cn2);
                object gc = Prop(fsm, "GeneroComponent");
                Console.WriteLine("    [dbg] fsm.GeneroComponent is my formEl? " + object.ReferenceEquals(gc, formEl)
                    + "   its style=<" + S(Call(gc, "GetAttribute", "style"))
                    + "> required=<" + S(Call(gc, "GetAttribute", "required")) + ">");
            }
        }
        else if (op == "del") {
            foreach (string p in paths) {
                if (w4.Index.ByPath(p) == null) { Console.WriteLine("!! 路径不存在: " + p); return; }
                object el = FindByPath(p);
                if (el == null) { Console.WriteLine("!! 模型里找不到: " + p); return; }
                string nm = S(Prop(el, "Name"));
                // Binding first: SaveBinding just serializes SpecBinding, so a binding left
                // pointing at a removed control would survive into the .bdx.
                object bind = Prop(el, "BindElement");
                if (bind != null) {
                    Console.WriteLine("    .bdx  解绑 " + nm + " ↔ " + S(Prop(bind, "Name")));
                    Call(el, "ClearBinding");
                }
                // Remove(name) clones a name-only tombstone with status="d" into the spec lists,
                // drops the RecordField, and for Table/Tree/ScrollGrid drops the table association.
                Call(si, "Remove", nm);
                w4.RemoveNode(p);
                changed++;
                Console.WriteLine("    del  <" + S(Prop(el, "NodeName")) + "> " + nm);
            }
            Console.WriteLine("[3] 删除 " + changed + " 个元素（规格留下 status=\"d\" 墓碑）");
        }
        else if (op == "wrap") {
            // HBox/VBox are not "new container to add fields into" -- they do not exist in
            // the widget box at all. The designer makes them by wrapping the current
            // selection (ManagedForm.CanLayoutCommand / ExecuteHBoxLayout), so this is a
            // layout refactor over EXISTING elements, not a field-adding operation.
            if (boxType != "HBox" && boxType != "VBox") {
                Console.WriteLine("!! wrap 的容器类型只能是 HBox 或 VBox"); return;
            }
            if (paths.Length < 1) { Usage(); return; }

            var els = new List<object>();
            string parentPath = null;
            foreach (string p in paths) {
                if (w4.Index.ByPath(p) == null) { Console.WriteLine("!! 路径不存在: " + p); return; }
                object el = FindByPath(p);
                if (el == null) { Console.WriteLine("!! 模型里找不到: " + p); return; }
                string pp = p.Substring(0, p.LastIndexOf('/'));
                if (parentPath == null) parentPath = pp;
                else if (parentPath != pp) {
                    // The command's Undo puts every element back into one container, so the
                    // designer assumes a single parent too; a cross-parent wrap is not a
                    // thing it can express.
                    Console.WriteLine("!! 待包裹的元素必须同属一个父节点：\n     " + parentPath + "\n     " + pp);
                    return;
                }
                els.Add(el);
            }
            object parentEl = Prop(els[0], "Parent");
            if (parentEl == null || S(Prop(parentEl, "NodeName")) == "Form") {
                Console.WriteLine("!! 不能把根布局包起来（设计器的 CanLayoutCommand 同样禁止）"); return;
            }

            // Same mime gates the designer applies before enabling the command; checking them
            // here turns a refusal into a sentence instead of an exception from inside it.
            var cft = A.GetType("SpecDesignerCommon.Helpers.ComponentFactory");
            foreach (var el in els) {
                if (!(bool)Call(cft, "AcceptMimes", boxType, S(Prop(el, "NodeName")))) {
                    Console.WriteLine("    !! " + S(Prop(el, "NodeName")) + " 不能被 " + boxType + " 接受"); return;
                }
            }
            if (!(bool)Call(cft, "AcceptMimes", S(Prop(parentEl, "NodeName")), boxType)) {
                Console.WriteLine("    !! 父容器 " + S(Prop(parentEl, "NodeName")) + " 不能接受 " + boxType); return;
            }

            var urf2 = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
            object urm2 = Activator.CreateInstance(Find(urf2, "UndoRedoManager"), new object[] { 100 });
            Call(urm2, "Init");
            ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urm2;

            var listT = typeof(List<>).MakeGenericType(Find(A, "XmlElement"));
            var sel = (IList)Activator.CreateInstance(listT);
            foreach (var el in els) sel.Add(el);

            object cmd = Activator.CreateInstance(Find(A, "AddToContainerUndoRedoCommand"),
                new object[] { sel, parentEl, Enum.Parse(Find(A, "ComponentType"), boxType) });
            Call(urm2, "AddThenExecute", cmd);

            object box = Prop(cmd, "_newContainer");
            if (box == null) { Console.WriteLine("!! 命令没有产出新容器"); return; }
            var boxXml = (XElement)Call(box, "ToXML");
            Console.WriteLine("    新建 <" + boxXml.Name.LocalName + " name=\"" + Attr(boxXml, "name")
                + "\" posX=" + Attr(boxXml, "posX") + " posY=" + Attr(boxXml, "posY")
                + " w=" + Attr(boxXml, "gridWidth") + " h=" + Attr(boxXml, "gridHeight")
                + ">  children=" + boxXml.Elements().Count());

            // The command already called SpecificationInfo.Add(_newContainer), so the spec
            // side is done; only the layout text needs the move.
            foreach (string p in paths) w4.RemoveNode(p);
            w4.AddNode(parentPath, boxXml);
            changed = els.Count;
            Console.WriteLine("[3] 把 " + changed + " 个元素包进新建的 " + boxType);
        }
        else if (op == "add") {
            // Adding a widget is one load, not two. add_field needed a second pass only
            // because UICreator builds its elements outside the model; here the designer's
            // own AddComponetsUndoRedoCommand does the whole job in place:
            //     container.AddNode(el)  +  Add(el, true) recursively  +  column enrichment
            // and for a Button, SpecificationInfo.Add -> FindAllFieldsSpecByName ->
            // FindActSpecById mints the <act> node with id = the button's name.
            if (paths.Length < 2) { Console.WriteLine("!! add 需要 <父路径> <控件类型>[:名字]"); return; }
            string parentPath = paths[0];
            string tspec = paths[1];
            int colon = tspec.IndexOf(':');
            string typeName = colon < 0 ? tspec : tspec.Substring(0, colon);
            string wantName = colon < 0 ? "" : tspec.Substring(colon + 1);

            object parentEl = FindByPath(parentPath);
            if (parentEl == null) { Console.WriteLine("!! 模型里找不到父节点: " + parentPath); return; }
            if (w4.Index.ByPath(parentPath) == null) { Console.WriteLine("!! 路径不存在: " + parentPath); return; }

            object ctype;
            try { ctype = Enum.Parse(Find(A, "ComponentType"), typeName, true); }
            catch { Console.WriteLine("!! 未知控件类型: " + typeName); return; }

            var cft = A.GetType("SpecDesignerCommon.Helpers.ComponentFactory");
            string parentTag = S(Prop(parentEl, "NodeName"));
            string ctypeName = S(ctype);                 // canonical casing, e.g. "Button"
            if (!(bool)Call(cft, "AcceptMimes", parentTag, ctypeName)) {
                Console.WriteLine("    !! " + parentTag + " 不接受 " + ctypeName); return;
            }

            string newName = wantName.Length > 0 ? wantName
                : S(Call(cft, "GetNewName", ctypeName.ToLower(), key));
            if ((bool)Call(si, "IsExists", newName)) {
                Console.WriteLine("!! 名字已存在: " + newName); return;
            }
            object el = Call(cft, "CreateEmptyComponent", key, ctype, newName);
            int row = w4.NextFreeRow(parentPath);

            var urf3 = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
            object urm3 = Activator.CreateInstance(Find(urf3, "UndoRedoManager"), new object[] { 100 });
            Call(urm3, "Init");
            ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urm3;

            var listT3 = typeof(List<>).MakeGenericType(Find(A, "XmlElement"));
            var one = (IList)Activator.CreateInstance(listT3);
            one.Add(el);
            // (list, container, posX, posY): the command adds posX/posY to the element's own
            // defaults, and for HBox/VBox/Table/Tree/Folder parents it ignores them entirely
            // and lets MeasurePos lay the child out instead.
            object addCmd = Activator.CreateInstance(Find(A, "AddComponetsUndoRedoCommand"),
                new object[] { one, parentEl, 1, row });
            Call(urm3, "AddThenExecute", addCmd);

            object fsmNew = Call(si, "FindNodeByName", newName);
            if (fsmNew == null) { Console.WriteLine("!! 设计器没有为该控件建立规格模型"); return; }
            int promoted = 0;
            foreach (string slot in NODE_SLOTS.Values) {
                object n2 = Prop(fsmNew, slot);
                if (n2 == null) continue;
                if (S(Prop(n2, "Status")).IndexOf("CREATE") < 0) continue;
                SetProp(n2, "Status", Enum.Parse(Find(A, "SpecStatus"), "MODIFY"));
                promoted++;
            }
            // A Button is exactly the case where the container is not a field: SpecField is
            // null and the node that matters is SpecAction, which NODE_SLOTS already covers.

            var elXml = (XElement)Call(el, "ToXML");
            Console.WriteLine("    新建 <" + elXml.Name.LocalName + " name=\"" + Attr(elXml, "name")
                + "\" posX=" + Attr(elXml, "posX") + " posY=" + Attr(elXml, "posY")
                + " w=" + Attr(elXml, "gridWidth") + " h=" + Attr(elXml, "gridHeight")
                + ">  规格节点 " + promoted + " 个脱离 CREATE");
            w4.AddNode(parentPath, elXml);
            changed = 1;
            Console.WriteLine("[3] 在 " + parentTag + " 下新增 " + typeName
                + "（SpecNodeType=" + S(Prop(fsmNew, "SpecNodeType")) + "）");
        }
        else if (op == "act") {
            // 新增项目: give an ALREADY EXISTING element its <act>. This is the panel's button,
            // as opposed to `add`, which creates a new element and gets an act as a side effect.
            //
            // `set <path> act:<attr> <value>` covers the ordinary case incidentally -- loading
            // mints a CREATE-status act and SpecAttributeUndoRedoCommand's OnPropertyChanged
            // promotes it -- but it cannot express "just create it", and if the value you pass
            // happens to equal the current one AddAttributeChanged drops the whole change and
            // nothing is promoted. Hence a dedicated op.
            if (paths.Length < 1) { Console.WriteLine("!! act 需要 [<元素路径>][:<act代号>]"); return; }
            string tspec = paths[0];
            int colon = tspec.IndexOf(':');
            string elPath = colon < 0 ? tspec : tspec.Substring(0, colon);
            string actId  = colon < 0 ? null : tspec.Substring(colon + 1);

            if (elPath.Length == 0) {
                // 新增项目 with no control: exactly what ActionDefaults' AddActionCommand does --
                // SpecActionNode.Create(info) with info.GetNewActionID(), i.e. id "action_N" and
                // nothing bound. This is the standalone kind; `output` in capp113 is one.
                string id = (actId != null && actId.Length > 0) ? actId : S(Call(si, "GetNewActionID"));
                object free = Call(Find(A, "SpecActionNode"), "Create", si, id);
                ((IList)Prop(si, "_acts")).Add(free);
                SetProp(free, "Status", Enum.Parse(Find(A, "SpecStatus"), "MODIFY"));
                Console.WriteLine("    <act id=\"" + id + "\">  status=u   （独立动作，不绑定任何控件）");
                changed = 1;
                Console.WriteLine("[3] 新增项目（无按钮）");
                goto afterAct;
            }

            object el = FindByPath(elPath);
            if (el == null) { Console.WriteLine("!! 模型里找不到该元素: " + elPath); return; }
            object fsmEl = Call(si, "FindNodeByName", S(Prop(el, "Name")));
            if (fsmEl == null) { Console.WriteLine("!! 该元素没有规格模型"); return; }
            if (S(Prop(fsmEl, "SpecNodeType")) != "ACTION") {
                Console.WriteLine("!! 该元素的 SpecNodeType 是 " + S(Prop(fsmEl, "SpecNodeType"))
                    + "，不是 ACTION（只有 Button 且 style 不是 button_qrystr 才是）");
                return;
            }
            if (actId == null || actId.Length == 0) actId = S(Prop(el, "Name"));

            // A CREATE-status act does not mean the file has one: loading mints one in memory
            // for every Button that lacks it (FindActSpecById), and ToXml() drops it on save.
            // That transient IS the thing to promote, so test persistence, not nullness.
            object existing = Prop(fsmEl, "SpecAction");
            if (existing != null && S(Prop(existing, "Status")).IndexOf("CREATE") < 0) {
                Console.WriteLine("!! " + S(Prop(el, "Name")) + " 在文件里已经有 act 了"
                    + "（status=" + S(Prop(existing, "Status")) + "）");
                return;
            }

            object node = existing;
            if (node != null) {
                Console.WriteLine("    复用加载期为它建的临时 act（status 原为 CREATE）");
            } else {
                // FindActSpecById would REFUSE when the name is an ActionDefault (tiptop.4ad has
                // 108 of them) -- that is why a button like detail_qrystr never gets one. The
                // panel path FindActionDefaultSpecOrCreate creates regardless, which is how
                // `output` ended up with an act despite being a default. Mirror that, but say so.
                bool isDefault = (bool)Call(si, "IsActionDefault", actId);
                node = Call(Find(A, "SpecActionNode"), "Create", si, actId);
                ((IList)Prop(si, "_acts")).Add(node);
                Call(fsmEl, "SetSpecNode", node);
                if (isDefault)
                    Console.WriteLine("    注意：" + actId + " 是 ActionDefault 名字，"
                        + "设计器的常规路径不会为它建 act");
            }
            SetProp(node, "Status", Enum.Parse(Find(A, "SpecStatus"), "MODIFY"));
            Console.WriteLine("    <act id=\"" + S(Prop(node, "Name")) + "\">  status=u");
            changed = 1;
            Console.WriteLine("[3] 为 <" + S(Prop(el, "NodeName")) + "> " + S(Prop(el, "Name")) + " 新增 act");
            afterAct: ;
        }
        else if (op == "tab") {
            // Reorder tab stops. tabIndex is ONE flat sequence over the whole form (not per
            // container), it only covers the 15 types whose mod-fd.spec properties list
            // includes tabIndex, and its default order is the layout tree's document order.
            //
            // ComponentTabIndexService keeps an "ordered prefix" plus the rest in document
            // order, and ArrangeTabIndex renumbers the prefix 1..K and the rest K+1..N,
            // skipping anything whose tabIndex is "". SetAsFirst moves one element to the
            // head of the prefix, so calling it in REVERSE of the wanted order leaves the
            // wanted order: each call flushes the current prefix behind the new head.
            if (paths.Length == 0) {
                // Same thing the toolbar's "Tab order specification" button does: its handler
                // (WidgetBox.ExecutedSetTabIndex) ends in ComponentTabIndexService.Show(true),
                // which rebuilds the list in document order and renumbers everything 1..N --
                // including elements whose tabIndex was "" (SortSourceList numbers all of them,
                // unlike ArrangeTabIndex which skips the empty ones).
                var svc0 = Call(Find(FE, "ComponentTabIndexService"), "Get", key);
                Call(svc0, "Register", Prop(si, "FormNode"));
                // The tabIndex setter goes through FormAttributesUndoRedoCommand, which needs a
                // registered manager -- without one the renumber silently does nothing.
                var urfT = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
                object urmT = Activator.CreateInstance(Find(urfT, "UndoRedoManager"), new object[] { 100 });
                Call(urmT, "Init");
                ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urmT;
                if (Environment.GetEnvironmentVariable("TZSCLI_DEBUG") == "1") {
                    Console.WriteLine("    [dbg] _sourceList=" + Count(Prop(svc0, "_sourceList"))
                        + " _tabIndexedList=" + Count(Prop(svc0, "_tabIndexedList"))
                        + " CurrentIndex=" + S(Prop(svc0, "CurrentIndex")));
                }
                var before0 = TabMap(w4);
                Call(svc0, "Show", true);
                var after0 = TabMap(w4);
                int moved0 = 0;
                foreach (var kv in before0) {
                    string now;
                    if (!after0.TryGetValue(kv.Key, out now) || now == kv.Value) continue;
                    w4.SetAttribute(kv.Key, "tabIndex", now);
                    moved0++;
                }
                Console.WriteLine("[3] 按文档顺序重排（等价于工具栏那个按钮）；tabIndex 改动 " + moved0 + " 处");
                changed = moved0;
                goto afterTab;
            }

            var els = new List<object>();
            foreach (string p in paths) {
                object el = FindByPath(p);
                if (el == null) {
                    Console.WriteLine("!! 模型里找不到: " + p);
                    // Dump what the model actually has for that leaf, so a path mismatch is
                    // diagnosable instead of a guess.
                    string leaf = p.Substring(p.LastIndexOf('/') + 1);
                    Console.WriteLine("   模型里以 \"" + leaf + "\" 结尾的路径：");
                    var found = new List<string>();
                    WalkTab2(Prop(si, "FormNode"), w4.Index.All[0].Name + "/" + S(Prop(Prop(si, "FormNode"), "Name")), found);
                    int shown = 0;
                    foreach (string f in found) {
                        if (!f.EndsWith("/" + leaf)) continue;
                        Console.WriteLine("     " + f);
                        if (++shown >= 4) break;
                    }
                    if (shown == 0) Console.WriteLine("     （一条都没有）");
                    return;
                }
                if (!((string)Call(el, "GetAttribute", "tabIndex") != null)) {
                    Console.WriteLine("!! " + p + " 没有 tabIndex 属性——它不是参与 Tab 的控件类型");
                    return;
                }
                els.Add(el);
            }

            // Every participating element's tabIndex, by name-path, so we can write back only
            // what actually moved. Keyed by path from the layout tree, which is what
            // FormWriter addresses.
            var before = TabMap(w4);

            var svc = Call(Find(FE, "ComponentTabIndexService"), "Get", key);
            Call(svc, "Register", Prop(si, "FormNode"));

            var urf4 = Assembly.LoadFrom(Path.Combine(INSTALL, "UndoRedoFramework.dll"));
            object urm4 = Activator.CreateInstance(Find(urf4, "UndoRedoManager"), new object[] { 100 });
            Call(urm4, "Init");
            ((IDictionary)Prop(SM, "undoRedoManagerMap"))[key] = urm4;

            for (int i = els.Count - 1; i >= 0; i--) Call(svc, "SetAsFirst", els[i]);

            var after = TabMap(w4);
            int moved = 0;
            foreach (var kv in before) {
                string now;
                if (!after.TryGetValue(kv.Key, out now) || now == kv.Value) continue;
                w4.SetAttribute(kv.Key, "tabIndex", now);
                moved++;
            }
            Console.WriteLine("[3] " + els.Count + " 个元素排到最前；tabIndex 改动 " + moved + " 处");
            changed = moved;
            afterTab: ;
        }
        else { Usage(); return; }

        // A spec-only change is legitimate: can_edit maps to noEntry, and if that attribute
        // already holds the target value the layout text is untouched while the .tsd still
        // has to be rewritten. So this must not be gated on FormWriter.Dirty -- Render()
        // returns the input byte-for-byte when nothing was applied, so the .4fd entry is
        // simply rewritten with identical bytes.
        string fd4New = w4.Render();
        Console.WriteLine("[4] .4fd " + fd4.Length + " → " + fd4New.Length + " chars ("
            + (fd4New.Length - fd4.Length >= 0 ? "+" : "") + (fd4New.Length - fd4.Length) + ")"
            + (w4.Dirty ? "" : "   [布局未变]"));
        Call(si, "SaveToTSD");
        Call(si, "SaveBinding");
        string tsdNew = ((XElement)Prop(si, "TSDElement")).ToString(SaveOptions.DisableFormatting);
        string bdxNew = Prop(tzp, "ElementBindings") as string;
        Console.WriteLine("[5] .tsd " + tsdNew.Length + " chars;  .bdx "
            + (bdxNew == null ? "(unchanged)" : bdxNew.Length + " chars"));

        var repl = new Dictionary<string, byte[]> {
            { fdEntry, Encoding.UTF8.GetBytes(fd4New) },
            { tsdEntry, Encoding.UTF8.GetBytes(tsdNew) }
        };
        if (bdxEntry != null && !string.IsNullOrEmpty(bdxNew)) repl[bdxEntry] = Encoding.UTF8.GetBytes(bdxNew);
        byte[] result = TzsRepacker.Repack(original, repl);
        File.WriteAllBytes(outTzs, result);
        Console.WriteLine("[6] repacked: " + original.Length + " → " + result.Length + " bytes");
    }

    static int Count(object o) { var d = Prop(o, "FormSpeDictionary") as ICollection; return d == null ? -1 : d.Count; }


    // ---------------- info: the read side ----------------

    static IDictionary _specDic;
    static int _infoCount;

    static string J(string s) {
        if (s == null) return "null";
        // Written by code point on purpose: this file has to survive being passed through
        // shells and patch scripts that eat backslashes, and a JSON escaper is nothing but
        // backslashes. Patching this method by hand kept collapsing them.
        char BS = (char)92, QT = (char)34;
        var sb = new StringBuilder();
        sb.Append(QT);
        foreach (char c in s) {
            if (c == QT) { sb.Append(BS).Append(QT); }
            else if (c == BS) { sb.Append(BS).Append(BS); }
            else if (c == (char)10) { sb.Append(BS).Append('n'); }
            else if (c == (char)13) { sb.Append(BS).Append('r'); }
            else if (c == (char)9) { sb.Append(BS).Append('t'); }
            else if (c < ' ') { sb.Append(BS).Append('u').Append(((int)c).ToString("x4")); }
            else { sb.Append(c); }
        }
        sb.Append(QT);
        return sb.ToString();
    }
    static object Raw(object el, string attr) { return Call(el, "GetAttribute", attr); }
    static string Seg(object el) {
        object nv = Prop(el, "Name");                    // not S(): S(null) == "(null)"
        string seg = nv == null ? null : nv.ToString();
        return string.IsNullOrEmpty(seg) ? S(Prop(el, "NodeName")) : seg;
    }

    /// <summary>
    /// stderr as UTF-8 bytes, for the same reason stdout is: .NET hands Console.Error the OEM
    /// code page (GBK here), so a redirected stderr turns every Chinese diagnostic into bytes
    /// the caller cannot decode. Not disposed -- it wraps a standard handle that outlives us.
    /// </summary>
    static void Err(string msg) {
        byte[] b = new UTF8Encoding(false).GetBytes(msg + "\n");
        Stream s = Console.OpenStandardError();
        s.Write(b, 0, b.Length);
        s.Flush();
    }

    /// <summary>
    /// The form as the designer's own 画面结构 panel shows it -- FormNode plus recursive
    /// Nodes, labelled by Name -- but as JSON, and with every node carrying the name-path
    /// that the other operations take as an argument. That path is the point: without it an
    /// AI would have to parse the .4fd itself before it could call set/del/add at all.
    /// </summary>
    // Written without a single backslash on purpose: this file keeps being patched through
    // shells and scripts that eat them, and JSON is nothing but quotes and backslashes.
    const char QT = (char)34;

    static void Key(StringBuilder sb, string k) { sb.Append(QT).Append(k).Append(QT).Append(':'); }

    /// <summary>
    /// The form as the designer's own 画面结构 panel shows it -- FormNode plus recursive
    /// Nodes, labelled by Name -- but as compact JSON, and with every node carrying the
    /// name-path the other operations take as an argument. That path is the point: without
    /// it an AI would have to parse the .4fd itself before it could call set/del/add at all.
    ///
    /// Compact, not indented: pretty-printing costs more than the data on a 670-element form
    /// (355 KB vs the payload), and no JSON reader cares.
    /// </summary>
    /// <summary>
    /// The designer's Status as the one letter the .tsd uses, or null for NULL -- the default
    /// state of a node loaded from disk and unchanged, and the only one worth leaving unsaid.
    ///
    /// CREATE is reported rather than hidden. It means the node is in memory only and ToXml()
    /// will drop it on save, so hiding it makes "on disk" and "phantom" indistinguishable --
    /// which matters most for actions: a Button with no &lt;act&gt; in the .tsd gets a synthetic
    /// action node at load time (SpecificationInfo.FindActSpecById), and without this it would
    /// read exactly like a real, persisted one.
    /// </summary>
    static string StatusLetter(object specNode) {
        string st = S(Prop(specNode, "Status"));
        if (st == "DELETE") return "d";
        if (st == "MODIFY") return "u";
        if (st == "CREATE") return "c";
        return null;
    }

    /// <summary>
    /// The fields every info node carries, shared by InfoNode (full tree) and InfoRef (flat
    /// match) so the two cannot drift. specNodeType in particular is what decides which
    /// &lt;nodeKind&gt; a `set` should use, so a second copy would be a real hazard.
    /// </summary>
    static void InfoFields(object el, string path, StringBuilder sb) {
        string name = (string)Raw(el, "name");

        Key(sb, "tag");   sb.Append(J(S(Prop(el, "NodeName"))));
        sb.Append(','); Key(sb, "name"); sb.Append(J(name));
        sb.Append(','); Key(sb, "path"); sb.Append(J(path));

        string ti = (string)Raw(el, "tabIndex");
        if (!string.IsNullOrEmpty(ti)) { sb.Append(','); Key(sb, "tabIndex"); sb.Append(ti); }

        string tab = (string)Raw(el, "sqlTabName"), col = (string)Raw(el, "colName");
        if (!string.IsNullOrEmpty(tab)) { sb.Append(','); Key(sb, "table");  sb.Append(J(tab)); }
        if (!string.IsNullOrEmpty(col)) { sb.Append(','); Key(sb, "column"); sb.Append(J(col)); }

        string ft = (string)Raw(el, "fieldType");
        if (!string.IsNullOrEmpty(ft)) { sb.Append(','); Key(sb, "fieldType"); sb.Append(J(ft)); }

        if (name != null && _specDic != null && _specDic.Contains(name)) {
            object fsm = _specDic[name];
            sb.Append(','); Key(sb, "specNodeType"); sb.Append(J(S(Prop(fsm, "SpecNodeType"))));
            object node = Prop(fsm, "SpecNode");
            if (node != null) {
                string desc = StatusLetter(node);
                if (desc != null) { sb.Append(','); Key(sb, "specStatus"); sb.Append(J(desc)); }
            }
        }
    }

    static int ChildCount(object el) {
        var kids = Prop(el, "Nodes") as IEnumerable;
        int n = 0;
        if (kids != null) foreach (var _ in kids) n++;
        return n;
    }

    /// <summary>
    /// One match without its subtree: a find can return many, and `info &lt;path&gt;` already
    /// expands any single one. childCount is kept because having children is what makes an
    /// element a legal target for add/wrap.
    /// </summary>
    static void InfoRef(object el, string path, StringBuilder sb) {
        sb.Append('{');
        InfoFields(el, path, sb);
        int n = ChildCount(el);
        if (n > 0) { sb.Append(','); Key(sb, "childCount"); sb.Append(n); }
        sb.Append('}');
    }

    static void InfoNode(object el, string path, StringBuilder sb, int depth) {
        _infoCount++;
        sb.Append('{');
        InfoFields(el, path, sb);
        if (ChildCount(el) > 0) {
            sb.Append(','); Key(sb, "children"); sb.Append('[');
            bool firstChild = true;
            foreach (var c in (IEnumerable)Prop(el, "Nodes")) {
                if (!firstChild) sb.Append(',');
                firstChild = false;
                InfoNode(c, path + "/" + Seg(c), sb, depth + 1);
            }
            sb.Append(']');
        }
        sb.Append('}');
    }

    static void EmitInfo(byte[] zip, string fdEntry, string tsdEntry, string pathFilter) {
        // JSON is UTF-8 by definition, and .NET's console defaults to the OEM code page (GBK
        // here), which mangles any non-ASCII name into bytes no JSON reader will accept.

        string fdText = EntryText(zip, fdEntry);
        var idx = ElementIndex.Build(fdText);
        string rootName = idx.All.Count > 0 ? idx.All[0].Name : "";
        object formNode = Prop(si, "FormNode");
        _specDic = (IDictionary)Prop(si, "FormSpeDictionary");
        _infoCount = 0;

        object start = formNode;
        string startPath = rootName + "/" + S(Prop(formNode, "Name"));
        if (pathFilter != null) {
            start = FindByPath(pathFilter);
            if (start == null) {
                Err("info: 模型里找不到 " + pathFilter);
                Environment.Exit(2);
            }
            startPath = pathFilter;
        }

        string tsdText = EntryText(zip, tsdEntry);
        string tpl = "";
        int ti = tsdText == null ? -1 : tsdText.IndexOf("<code_template");
        if (ti >= 0) {
            int v = tsdText.IndexOf("value=\"", ti);
            if (v >= 0) { v += 7; int e2 = tsdText.IndexOf('"', v); if (e2 > v) tpl = tsdText.Substring(v, e2 - v); }
        }

        var sb = new StringBuilder();
        sb.Append("{\n");
        sb.Append("  \"program\": ").Append(J(S(Prop(tzp, "ProgramName")))).Append(",\n");
        sb.Append("  \"env\": ").Append(J(S(Prop(si, "Env")))).Append(",\n");
        sb.Append("  \"codeTemplate\": ").Append(J(tpl)).Append(",\n");
        sb.Append("  \"specDictionarySize\": ").Append(_specDic == null ? 0 : _specDic.Count).Append(",\n");
        sb.Append("  \"root\": ");
        InfoNode(start, startPath, sb, 1);
        sb.Append(",\n  \"elementCount\": ").Append(_infoCount).Append("\n}\n");
        // Console.Out runs through the OEM code page (GBK here) even when OutputEncoding is
        // set, so write UTF-8 bytes to the raw stream -- JSON has to be UTF-8.
        using (var stdout = new StreamWriter(Console.OpenStandardOutput(), new UTF8Encoding(false))) {
            stdout.Write(sb.ToString());
            stdout.Flush();
        }
    }

    /// <summary>
    /// Every element whose 控件代号 matches, as parallel (element, path) lists in document
    /// order. `substring` is the fallback sweep for a partial or mistyped code.
    /// </summary>
    static void CollectByName(object el, string here, string term, bool substring, List<object> els, List<string> paths) {
        string s = Seg(el);
        if (s != null && (substring ? s.IndexOf(term, StringComparison.OrdinalIgnoreCase) >= 0 : s == term)) {
            els.Add(el);
            paths.Add(here);
        }
        var nodes = Prop(el, "Nodes") as IEnumerable;
        if (nodes == null) return;
        foreach (var c in nodes) CollectByName(c, here + "/" + Seg(c), term, substring, els, paths);
    }

    /// <summary>
    /// Actions by id. They live in their own collection, not in the element tree and not in
    /// FormSpeDictionary, so neither walk above nor the specNodeType lookup can see them.
    /// </summary>
    static void CollectActs(string term, bool substring, List<object> hits) {
        var acts = Prop(si, "Actions") as IEnumerable;
        if (acts == null) return;
        foreach (var a in acts) {
            string n = S(Prop(a, "Name"));
            if (n == null) continue;
            if (substring ? n.IndexOf(term, StringComparison.OrdinalIgnoreCase) >= 0 : n == term) hits.Add(a);
        }
    }

    /// <summary>
    /// `info &lt;in.tzs&gt; --find &lt;代号&gt;`: resolve a 控件代号 to the name-path the write
    /// operations take as their argument -- otherwise obtainable only by dumping the whole tree.
    ///
    /// The designer refuses to create or rename a component onto a name already in
    /// FormSpeDictionary (so that 字段属性 panel's own rename box cannot make a duplicate),
    /// which means a code normally identifies exactly one component. The count is still
    /// reported rather than a winner picked: a hand-edited .4fd can break that invariant, and
    /// silently acting on the wrong element is the one outcome worth spending a field on.
    ///
    /// An exact hit suppresses the substring sweep, so a code that also prefixes another
    /// element still resolves to itself. No match is a valid answer, not an error: matchCount
    /// comes back 0 and the exit code stays 0, because the argument was a query, not a
    /// location -- unlike `info &lt;path&gt;`, where being wrong means the caller is lost.
    ///
    /// Two lists come back: `matches` are elements and carry the path, `actionMatches` are
    /// actions and carry an id. A bound button is in both, under the same name.
    /// </summary>
    static void EmitFind(byte[] zip, string fdEntry, string term) {
        if (string.IsNullOrEmpty(term)) {
            Err("info --find: 需要一个控件代号");
            Environment.Exit(2);
        }

        string fdText = EntryText(zip, fdEntry);
        var idx = ElementIndex.Build(fdText);
        string rootName = idx.All.Count > 0 ? idx.All[0].Name : "";
        object formNode = Prop(si, "FormNode");
        _specDic = (IDictionary)Prop(si, "FormSpeDictionary");

        // A code can name an action rather than a component. 新增项目 writes an <act id="...">
        // with no <Button> behind it, and such an id appears in no layout element -- so without
        // this it comes back as a bare empty list that reads like a typo. Actions are NOT in
        // FormSpeDictionary (Rename guards them with a separate CheckActionNameIsExists), which
        // is exactly why the element walk above cannot see them.
        var els = new List<object>();
        var paths = new List<string>();
        var actHits = new List<object>();
        // Same start as EmitInfo: the ManagedForm root's name, then the <Form>'s.
        string startPath = rootName + "/" + Seg(formNode);
        CollectByName(formNode, startPath, term, false, els, paths);
        CollectActs(term, false, actHits);
        bool exact = els.Count > 0 || actHits.Count > 0;
        if (!exact) {
            CollectByName(formNode, startPath, term, true, els, paths);
            CollectActs(term, true, actHits);
        }

        var sb = new StringBuilder();
        sb.Append("{\n");
        sb.Append("  \"program\": ").Append(J(S(Prop(tzp, "ProgramName")))).Append(",\n");
        sb.Append("  \"env\": ").Append(J(S(Prop(si, "Env")))).Append(",\n");
        sb.Append("  \"query\": ").Append(J(term)).Append(",\n");
        sb.Append("  \"exact\": ").Append(exact ? "true" : "false").Append(",\n");
        sb.Append("  \"matchCount\": ").Append(els.Count).Append(",\n");
        sb.Append("  \"matches\": [");
        for (int i = 0; i < els.Count; i++) {
            if (i > 0) sb.Append(',');
            InfoRef(els[i], paths[i], sb);
        }
        sb.Append(']');

        // A bound button appears in both lists under the same name -- the element by name, the
        // action by id. On the action, specStatus "c" means the button has no <act> in the .tsd
        // (the node was synthesised at load time); absent means it really is on disk. That
        // distinction is the whole point of reporting a match here, so it is never left out.
        if (actHits.Count > 0) {
            sb.Append(",\n  \"actionMatches\": [");
            for (int i = 0; i < actHits.Count; i++) {
                if (i > 0) sb.Append(',');
                object a = actHits[i];
                sb.Append('{');
                Key(sb, "id"); sb.Append(J(S(Prop(a, "Name"))));
                string desc = StatusLetter(a);
                if (desc != null) { sb.Append(','); Key(sb, "specStatus"); sb.Append(J(desc)); }
                sb.Append('}');
            }
            sb.Append(']');
        }
        sb.Append("\n}\n");

        using (var stdout = new StreamWriter(Console.OpenStandardOutput(), new UTF8Encoding(false))) {
            stdout.Write(sb.ToString());
            stdout.Flush();
        }
    }

    static void Usage() {
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> set  <elementPath> <nodeKind>:<attr> <value>");
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> del  <elementPath>...");
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> wrap <HBox|VBox> <elementPath>...");
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> add  <parentPath> <ComponentType>[:name]");
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> act  <elementPath>[:<actId>]");
        Console.WriteLine("Edit.exe <in.tzs> <out.tzs> tab  [<elementPath>...]");
        Console.WriteLine("Edit.exe info <in.tzs> [<elementPath>]      # 只读，输出 JSON 到 stdout");
        Console.WriteLine("Edit.exe info <in.tzs> --find <控件代号>     # 只读，按代号定位，输出 JSON");
        Console.WriteLine();
        Console.WriteLine("  elementPath  名字路径，如 managedform/aapp320/mainlayout/.../apca_t.apcaent");
        Console.WriteLine("  nodeKind     " + string.Join(" | ", NODE_SLOTS.Keys.OrderBy(x => x).ToArray())
            + "   (默认 field)");
        Console.WriteLine();
        Console.WriteLine("del 接受多个路径：一个字段是控件 + 它的标签伴随节点，只删一个会留下孤儿。");
        Console.WriteLine("wrap 把【已有】元素包进一个新容器——HBox/VBox 不在控件箱里，设计器就是这么造它们的。");
        Console.WriteLine("     所有待包裹元素必须同属一个父节点，且父节点不能是 Form。");
        Console.WriteLine("add  在父容器下新增一个控件。Button 的 SpecNodeType 就是 ACTION，");
        Console.WriteLine("     设计器会自动为它造出 id = 控件名的 <act> 规格节点。");
        Console.WriteLine("tab  不带路径 = 按文档顺序重排 1..N（等同工具栏那个 Tab order 按钮）；");
        Console.WriteLine("     带路径   = 把列出的控件按给定顺序排到最前，其余保持文档顺序接在后面。");
        Console.WriteLine("act  为【已有】元素补一个 <act>（面板上的「新增项目」）。actId 默认取元素名；");
        Console.WriteLine("     绑定关系就是 act.id == 元素 name，没有别的地方记录。");
        Console.WriteLine();
        Console.WriteLine("examples:");
        Console.WriteLine("  Edit.exe a.tzs b.tzs set  <path> field:req Y");
        Console.WriteLine("  Edit.exe a.tzs b.tzs del  <path-to-Edit> <path-to-its-Label>");
        Console.WriteLine("  Edit.exe a.tzs b.tzs wrap HBox <path-to-A> <path-to-B>");
        Console.WriteLine("  Edit.exe a.tzs b.tzs add  <parent-path> Button:my_action");
        Console.WriteLine("  Edit.exe a.tzs b.tzs act  <path-to-Button>");
    }
}
