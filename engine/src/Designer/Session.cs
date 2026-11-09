using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.Reflection;
using System.Threading;

namespace TzsCli.Designer
{
    /// <summary>
    /// One open package, addressed by a handle that is neither its path nor its ProgramKey.
    ///
    /// This is Probe.cs's Pkg (test/Probe.cs:104) promoted to the real session model. The one
    /// structural difference is the load/operate split, which is a hard constraint rather than
    /// a design taste:
    ///
    ///   TzpManager.GetChildNode calls SetInitGridX/SetInitGridY, and those deliberately throw
    ///   "illeagal call" when an UndoRedoManager is registered for the key (SPEC §11.24 (b),
    ///   §11.5 缺口 11). So the manager has to be absent during the load and present for every
    ///   mutation -- and since an already-loaded key cannot be reloaded, RegisterUndoRedo is a
    ///   one-time step of the session, not something each operation does.
    ///
    /// Edit.cs inlines that step four times (:340/:442/:507/:637) because each of its processes
    /// performs exactly one operation. Here it happens once per handle.
    ///
    /// Handle = "h" + a monotonically increasing integer, per SPEC §11.24 (g): never a path,
    /// never a ProgramKey (four files in the corpus share one ProgramKey), never reused.
    /// </summary>
    public sealed class Session
    {
        public string Handle, Path, Program;
        public object Tzp, Si, Key;
        public double LoadMs;

        /// <summary>True once an UndoRedoManager has been registered for this key. Irreversible
        /// for the life of the handle: the load and operate phases must not interleave.</summary>
        public bool Mutable;

        /// <summary>True after Close(). The server must reject every operation on a closed
        /// handle rather than let it resolve against a tzpMap entry that no longer exists.</summary>
        public bool Closed;

        static int _nextHandle;

        /// <summary>
        /// The load watchdog's timeout: TZSCLI_RELOAD_TIMEOUT in seconds, or 90.
        /// Kept here rather than at each call site so the environment variable is honoured the
        /// same way it has been in Edit.cs:282 and probe/AddField since it was introduced.
        /// </summary>
        public static int TimeoutFromEnv() {
            int t = 90;
            int.TryParse(Environment.GetEnvironmentVariable("TZSCLI_RELOAD_TIMEOUT") ?? "90", out t);
            return t;
        }

        /// <summary>
        /// Loads a package into the designer. Registers tzpMap[key] and EventAggregatorManager,
        /// runs SpecificationInfo.Create and writes the result back onto the TzpManager (SPEC
        /// §8.5 缺口 1 -- without that write-back AbstractSpecNode.Status NREs when it looks the
        /// manager up again). Does NOT register an UndoRedoManager; call RegisterUndoRedo
        /// before the first mutation.
        ///
        /// The watchdog is Probe.cs:114 verbatim, and it is not optional: SpecificationInfo's
        /// constructor can raise a modal DesignerMessageBox (most notoriously
        /// DatabaseSourceViewModel reporting tables missing from mta/tables.xml), and headless
        /// there is no message pump, so the load blocks forever with nothing on stdout.
        ///
        /// The watchdog runs on its own thread because the load has to stay on the calling STA
        /// thread; all it can do is Exit, since a thread parked inside a dispatcher frame will
        /// never observe a cancellation flag.
        /// </summary>
        public static Session Open(string path, int timeoutSeconds) {
            var s = new Session { Path = path, Handle = "h" + (++_nextHandle) };
            var done = new ManualResetEvent(false);
            var watch = new Thread(delegate() {
                if (done.WaitOne(timeoutSeconds * 1000)) return;
                // SPEC §11.24 (h): this exit is terminal and cannot be avoided -- the main
                // thread is parked inside Activator.CreateInstance below and a thread in a
                // dispatcher frame never observes a cancellation flag. OnFatal lets the
                // transport emit its fatal frame and flush before we take the process down.
                try { if (OnFatal != null) OnFatal(path); } catch { }
                // Json.Err writes UTF-8 bytes; Console.Error goes through the OEM code page
                // (GBK here) and mangles every Chinese diagnostic in a pipe -- SPEC §11.20.
                Json.Err("!! 加载超时 " + timeoutSeconds + "s: " + path);
                Environment.Exit(3);
            });
            watch.IsBackground = true;
            watch.Start();
            var sw = Stopwatch.StartNew();
            bool added = false;
            object map = null, k = null;
            try {
                object t = Activator.CreateInstance(Designer.A.GetType("SpecDesignerCommon.TzpManager"), new object[] { path });
                k = Reflect.Prop(t, "ProgramKey");
                map = Reflect.Prop(Designer.SettingManager, "tzpMap");

                // SPEC §11.24 (g): the original did map.Remove(k) then map.Add(k,t), which
                // silently orphans the previous handle's TzpManager while its UndoRedoManager
                // stays behind in undoRedoManagerMap. ProbeReopen proved the two consequences:
                // the survivor is whichever loaded last (so an earlier handle silently starts
                // answering with the wrong package's data), and if the occupant is Mutable the
                // replacement load throws "illeagal call SetInitGridY" half-way, leaving a
                // TzpManager in the map whose SpecificationInfo was never assigned -- a poisoned
                // key. Refusing outright is the only safe answer; the caller can close first.
                if (_open.ContainsKey(SlotKey(k)))
                    throw new TzsError("key_in_use",
                        "这个包已经在打开状态（ProgramKey 不含路径，同一个程序名的不同文件会撞）："
                        + _open[SlotKey(k)].Path + "；先 close 它，或换一个文件");
                if (((IDictionary)map).Contains(k))
                    throw new TzsError("key_in_use",
                        "这个 ProgramKey 已被占用，但不是本会话打开的（可能来自上一次失败的加载）");

                ((IDictionary)map).Add(k, t);
                added = true;
                var eam = Reflect.Find(Designer.A, "EventAggregatorManager");
                if (!(bool)Reflect.Call(eam, "ContainsKey", k)) Reflect.Call(eam, "CreateInstance", k);
                object si = Reflect.Call(Designer.A.GetType("SpecDesignerCommon.SpecificationInfo"), "Create", t);
                Reflect.SetProp(t, "SpecificationInfo", si);
                s.Tzp = t; s.Si = si; s.Key = k;
                s.Program = Reflect.S(Reflect.Prop(t, "ProgramName"));
                _open[SlotKey(k)] = s;
            } catch {
                // Roll the key back. Without this a failed load leaves the half-built TzpManager
                // in tzpMap and every later Open of that key reports "occupied" forever.
                if (added && map != null && k != null) {
                    try { ((IDictionary)map).Remove(k); } catch { }
                    try { Reflect.Call(Reflect.Find(Designer.A, "EventAggregatorManager"), "Remove", k); } catch { }
                }
                throw;
            } finally { done.Set(); sw.Stop(); s.LoadMs = sw.Elapsed.TotalMilliseconds; }
            return s;
        }

        /// <summary>Open handles by ProgramKey, so a collision is refused instead of silently
        /// evicting. Process-global because the designer's tzpMap is.
        ///
        /// Keyed by Fns.Session.KeyString -- NOT by the PackageKey object and NOT by its
        /// ToString(). Both of those are wrong, and each was tried:
        ///
        ///   * the object itself: PackageKey overrides Equals (PackageKey.cs:43) but NOT
        ///     GetHashCode, so under Dictionary's default comparer two equal keys hash
        ///     differently and ContainsKey misses. W2-C found it.
        ///   * ToString(): PackageKey does not override that either, so every package rendered
        ///     as the literal type name and the guard refused a second open of ANY other file
        ///     with E_KEY_IN_USE. That made multi-file support -- and therefore copy_component --
        ///     unreachable. W3-D found it; it was my regression.
        ///
        /// KeyString projects Program|PackType, while PackageKey.Equals also compares Memo. They
        /// agree for us because nothing in this tool ever varies Memo, and the projection is
        /// shared with list_open, so a caller sees two colliding files as the same string.
        /// If Memo ever becomes reachable, KeyString has to grow a third field.
        /// </summary>
        static readonly Dictionary<string, Session> _open = new Dictionary<string, Session>();

        static string SlotKey(object key) { return Fns.Session.KeyString(key) ?? ""; }

        /// <summary>Set by the transport so a fatal load timeout can emit its frame and flush
        /// before the watchdog calls Environment.Exit (SPEC §11.24 (h)).</summary>
        public static Action<string> OnFatal;

        /// <summary>
        /// The four lines Edit.cs inlines four times, so that every caller shares one copy:
        /// load UndoRedoFramework, new UndoRedoManager(100), Init(), and file it under this
        /// key in SettingManager.undoRedoManagerMap.
        ///
        /// Static and key-taking so that Step 2 (UICreator's layout setters) and Session both
        /// use it; the instance overload below is the same thing guarded by Mutable.
        /// </summary>
        public static void RegisterUndoRedo(object settingManager, object key) {
            var urf = Assembly.LoadFrom(System.IO.Path.Combine(Designer.Install, "UndoRedoFramework.dll"));
            object urm = Activator.CreateInstance(Reflect.Find(urf, "UndoRedoManager"), new object[] { 100 });
            Reflect.Call(urm, "Init");
            ((IDictionary)Reflect.Prop(settingManager, "undoRedoManagerMap"))[key] = urm;
        }

        /// <summary>Idempotent: the second call is a no-op. Registering a second manager would
        /// orphan the first, and every undo command the designer runs holds a reference to it.</summary>
        public void RegisterUndoRedo() {
            if (Mutable) return;
            RegisterUndoRedo(Designer.SettingManager, Key);
            Mutable = true;
        }

        /// <summary>
        /// Releases the handle. Verified by test/ProbeReopen.cs.
        ///
        /// The map removals alone are NOT enough. Every Open subscribes three handlers to the
        /// Global aggregator, and only one thing unsubscribes them: publishing TzpFileClose,
        /// which is what SettingManager.CloseFile (SettingManager.cs:517) subscribes to.
        /// Measured subscriber counts for one open/close cycle:
        ///
        ///     after one open                     : TzpFileClose=3  SaveSettingEvent=1
        ///     after the three map removals only  : TzpFileClose=3  SaveSettingEvent=1   <- leaked
        ///     after publishing TzpFileClose      : TzpFileClose=1  SaveSettingEvent=0   <- reclaimed
        ///
        /// The leaked SaveSettingEvent handler is the dangerous one: its body resolves the
        /// package with GetTzpManger(this.Key) (SpecificationInfo.cs:260), and by then that key
        /// belongs to a *different* package -- so the next save would write a closed package's
        /// model into the new one.
        ///
        /// Publishing TzpFileClose also does two of the three removals itself (tzpMap,
        /// undoRedoManagerMap) plus the ActionDefaults null-out, so the explicit removals below
        /// are belt-and-braces for the case where the key was never subscribed.
        ///
        /// The EventAggregatorManager instance is separate: CloseFile does not touch it, so it
        /// is removed explicitly.
        ///
        /// Removal is per map key, not per handle: in a key-collision cycle close(A) will find
        /// the entry occupied by B and remove B. Callers must not assume the named handle is the
        /// one removed. Absence is tolerated throughout.
        /// </summary>
        public void Close() {
            if (Closed) return;

            // The one that matters: reclaims the three Global subscribers this Open added.
            try {
                Type eamType = Reflect.Find(Designer.A, "EventAggregatorManager");
                object global = Reflect.Prop2(eamType, "Global");
                object evt = GlobalEvent("TzpFileClose");
                if (evt != null && global != null) Publish(evt, Key);
            } catch (Exception ex) {
                // A leaked subscriber is bad but not worth failing the close over; the server
                // logs it. Swallowing here keeps Close total, which the session model needs.
                Json.Err("Session.Close: TzpFileClose publish failed: " + ex.Message);
            }

            var map = (IDictionary)Reflect.Prop(Designer.SettingManager, "tzpMap");
            if (map.Contains(Key)) {
                Reflect.SetProp(map[Key], "ActionDefaults", null);
                map.Remove(Key);
            }
            ((IDictionary)Reflect.Prop(Designer.SettingManager, "undoRedoManagerMap")).Remove(Key);
            Reflect.Call(Reflect.Find(Designer.A, "EventAggregatorManager"), "Remove", Key);

            _open.Remove(SlotKey(Key));
            Closed = true;
        }

        /// <summary>
        /// EventAggregatorManager.Global.GetEvent&lt;T&gt;() without a compile-time reference to the
        /// designer. Generic method, so it needs MakeGenericMethod.
        /// </summary>
        private static object GlobalEvent(string eventTypeName) {
            object eam = Reflect.Prop2(Reflect.Find(Designer.A, "EventAggregatorManager"), "Global");
            if (eam == null) return null;
            Type evtType = Designer.A.GetType("SpecDesignerCommon.Events." + eventTypeName);
            if (evtType == null) return null;
            foreach (var m in eam.GetType().GetMethods()) {
                if (m.Name != "GetEvent" || !m.IsGenericMethodDefinition || m.GetParameters().Length != 0) continue;
                return m.MakeGenericMethod(evtType).Invoke(eam, null);
            }
            return null;
        }

        /// <summary>Prism CompositePresentationEvent&lt;T&gt;.Publish(T). The payload type is
        /// PackageKey, which we hold only as object -- fine, Invoke takes it as object.</summary>
        private static void Publish(object evt, object payload) {
            foreach (var m in evt.GetType().GetMethods()) {
                if (m.Name != "Publish" || m.GetParameters().Length != 1) continue;
                m.Invoke(evt, new object[] { payload });
                return;
            }
            throw new Exception("no Publish/1 on " + evt.GetType());
        }

        /// <summary>
        /// The model's element for a .4fd name-path (Edit.cs:161). FormNode is the &lt;Form&gt;,
        /// which is the second path segment (the first is the ManagedForm root), so we match
        /// from there and check the segment rather than trusting it.
        /// </summary>
        public object FindByPath(string path) {
            string[] parts = path.Split('/');
            object formNode = Reflect.Prop(Si, "FormNode");
            if (formNode == null || parts.Length < 2) return null;
            if (Reflect.S(Reflect.Prop(formNode, "Name")) != parts[1]) return null;
            string prefix = parts[0] + "/" + parts[1];
            return Walk(formNode, prefix, path);
        }

        // ---------------------------------------------------------------- tree walks
        //
        // All of these address elements the way FormWriter does: the document root's name,
        // then the Form's name, then down. The segment is Name, falling back to NodeName --
        // but only when Name is null or empty, because containers legitimately have no name
        // and a nameless node must still get a distinct path segment.

        public static object Walk(object el, string here, string target) {
            if (here == target) return el;
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return null;
            foreach (var c in nodes) {
                string seg = Seg(c);
                object hit = Walk(c, here + "/" + seg, target);
                if (hit != null) return hit;
            }
            return null;
        }

        /// <summary>The path segment for one element. Prop(), not S(): S(null) is "(null)".</summary>
        public static string Seg(object el) {
            object nv = Reflect.Prop(el, "Name");
            string seg = nv == null ? null : nv.ToString();
            return string.IsNullOrEmpty(seg) ? Reflect.S(Reflect.Prop(el, "NodeName")) : seg;
        }

        /// <summary>Every element, as parallel (path, element) lists in document order.</summary>
        public static void CollectAll(object el, string here, List<string> paths, List<object> els) {
            paths.Add(here); els.Add(el);
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return;
            foreach (var c in nodes) CollectAll(c, here + "/" + Seg(c), paths, els);
        }

        /// <summary>
        /// Every element whose 控件代号 matches, as parallel (element, path) lists in document
        /// order. `substring` is the fallback sweep for a partial or mistyped code.
        /// </summary>
        public static void CollectByName(object el, string here, string term, bool substring, List<object> els, List<string> paths) {
            string s = Seg(el);
            if (s != null && (substring ? s.IndexOf(term, StringComparison.OrdinalIgnoreCase) >= 0 : s == term)) {
                els.Add(el);
                paths.Add(here);
            }
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return;
            foreach (var c in nodes) CollectByName(c, here + "/" + Seg(c), term, substring, els, paths);
        }

        /// <summary>
        /// Actions by id. They live in their own collection (SpecificationInfo.Actions), not in
        /// the element tree and not in FormSpeDictionary, so neither CollectByName nor the
        /// specNodeType lookup can see them. Instance member because it dereferences Si.
        /// </summary>
        public void CollectActs(string term, bool substring, List<object> hits) {
            var acts = Reflect.Prop(Si, "Actions") as IEnumerable;
            if (acts == null) return;
            foreach (var a in acts) {
                string n = Reflect.S(Reflect.Prop(a, "Name"));
                if (n == null) continue;
                if (substring ? n.IndexOf(term, StringComparison.OrdinalIgnoreCase) >= 0 : n == term) hits.Add(a);
            }
        }

        public static int ChildCount(object el) {
            var kids = Reflect.Prop(el, "Nodes") as IEnumerable;
            int n = 0;
            if (kids != null) foreach (var _ in kids) n++;
            return n;
        }

        public static object Raw(object el, string attr) { return Reflect.Call(el, "GetAttribute", attr); }

        /// <summary>Every attribute of an element as a name -> value snapshot. Used to diff a
        /// form element across a designer command so only what moved reaches the .4fd text.</summary>
        public static Dictionary<string,string> Attrs(object el) {
            var d = new Dictionary<string,string>();
            var names = Reflect.Prop(el, "Attributes") as IEnumerable;
            if (names == null) return d;
            foreach (var n in names) {
                string k = n.ToString();
                d[k] = Reflect.S(Reflect.Call(el, "GetAttribute", k));
            }
            return d;
        }

        /// <summary>
        /// name-path -> tabIndex for every element under the form, addressed exactly the way
        /// FormWriter addresses them: the document root's name, then the Form's name, then down.
        /// The map is keyed by the .4fd path, which is what the text writer takes.
        /// </summary>
        public Dictionary<string,string> TabMap(TzsCli.FormWriter w4) {
            var map = new Dictionary<string,string>();
            object formNode = Reflect.Prop(Si, "FormNode");
            if (formNode == null || w4.Index.All.Count == 0) return map;
            WalkTab(formNode, w4.Index.All[0].Name + "/" + Reflect.S(Reflect.Prop(formNode, "Name")), map);
            return map;
        }

        public static void WalkTab(object el, string here, Dictionary<string,string> map) {
            string t = (string)Reflect.Call(el, "GetAttribute", "tabIndex");
            if (t != null) map[here] = t;
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return;
            foreach (var c in nodes) WalkTab(c, here + "/" + Seg(c), map);
        }

        public static void WalkTab2(object el, string here, List<string> acc) {
            acc.Add(here);
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return;
            foreach (var c in nodes) WalkTab2(c, here + "/" + Seg(c), acc);
        }
    }
}
