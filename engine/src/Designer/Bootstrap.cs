using System;
using System.Collections;
using System.IO;
using System.Reflection;
using System.Resources;
using System.Windows;
using System.Windows.Markup;

namespace TzsCli.Designer
{
    /// <summary>
    /// The headless designer context: the five-step boot every tool in this repo repeats.
    ///
    /// It appeared three times verbatim -- test/Edit.cs:265-284, test/Probe.cs:595-616,
    /// test/AddField.cs:195-212 -- and the function layer needs it once per process, so it
    /// lives here now.
    ///
    /// MUST be called on the process's STA main thread. Application.Current is thread-affine
    /// and SettingModel/SpecificationInfo resolve resources through it, so booting on a worker
    /// thread fails with a cross-thread exception rather than loading.
    /// </summary>
    public static class Designer
    {
        /// <summary>Where the designer's assemblies live. Everything is Assembly.LoadFrom()'d
        /// from here at runtime -- nothing in this library is compiled against it.
        ///
        /// Two answers, in this order:
        ///
        ///   1. TZSCLI_INSTALL, if set. This is the dev/build escape hatch: engine/out/ is not a
        ///      package, so the probe programs and a locally built engine point this at whatever
        ///      designer the machine has.
        ///   2. `&lt;this exe's directory&gt;\designer` -- the copy that SHIPS WITH US. The package
        ///      carries the designer's assemblies so that a run depends on the package's version
        ///      and nothing else; before this, the answer came from the user's config and two
        ///      machines with the same tt could run against different designers.
        ///
        /// There is deliberately no fallback to an installed designer: "the one next to us, or
        /// the one you named" is the whole rule, and a missing &lt;exe&gt;\designer is a broken package
        /// rather than a reason to guess. Not `const` because it is read from the environment once
        /// per process.</summary>
        public static readonly string Install = Resolve();

        static string Resolve() {
            string env = Env("TZSCLI_INSTALL");
            if (env != null) return env;
            return Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "designer");
        }

        static string Env(string name) {
            string v = Environment.GetEnvironmentVariable(name);
            return string.IsNullOrEmpty(v) ? null : v;
        }

        /// <summary>The languages the designer's own UI strings are looked up in. The designer
        /// ships several, and merging the wrong one leaves FindResource("Message_...") returning
        /// null -- which surfaces far away, as an ArgumentNullException inside string.Format.</summary>
        const string Lang = "zh-cn";

        /// <summary>Merges every `langs/&lt;Lang&gt;.xaml` the designer has EMBEDDED in its own
        /// assemblies.
        ///
        /// This used to read the one file out of the designer's decompiled source tree, on the
        /// stated grounds that "the installed copy is compiled into a baml resource and is not
        /// reachable by path". That reasoning was right about the path and stopped one step short:
        /// it is not on a path, it is inside the assembly. `&lt;Assembly&gt;.g.resources` is a normal
        /// .NET resource bundle, and ResourceReader hands back the entry as the ORIGINAL XAML
        /// (not baml) -- measured on the installed designer: SpecDesignerCommon carries
        /// langs/zh-cn.xaml at 88,737 bytes and SpecDesigner.Controls carries another at 450.
        /// Both are needed; that is why this walks the directory rather than loading one file.
        ///
        /// So the source tree is no longer a runtime dependency at all, and the strings come from
        /// the designer version this package carries -- which is now the point: one package, one
        /// set of UI strings. See Install for where that copy comes from.</summary>
        static void MergeLanguages(Application app) {
            string[] files;
            try { files = Directory.GetFiles(Install, "SpecDesigner*.dll"); }
            catch (Exception ex) {
                throw new Exception("读不到设计器目录 " + Install + ": " + ex.Message);
            }
            Array.Sort(files, StringComparer.Ordinal);   // deterministic merge order

            int merged = 0;
            foreach (string f in files) {
                Assembly asm;
                try { asm = Assembly.LoadFrom(f); } catch { continue; }   // not all are loadable
                string bundle = asm.GetName().Name + ".g.resources";
                using (Stream s = asm.GetManifestResourceStream(bundle)) {
                    if (s == null) continue;
                    using (var rr = new ResourceReader(s)) {
                        foreach (DictionaryEntry e in rr) {
                            if (!string.Equals(e.Key as string, "langs/" + Lang + ".xaml",
                                               StringComparison.OrdinalIgnoreCase)) continue;
                            var stream = e.Value as Stream;
                            if (stream == null) continue;
                            app.Resources.MergedDictionaries.Add(
                                (ResourceDictionary)XamlReader.Load(stream));
                            merged++;
                        }
                    }
                }
            }
            if (merged == 0)
                throw new Exception("设计器程序集里找不到 langs/" + Lang + ".xaml（Install=" + Install
                    + "）；没有它 FindResource(\"Message_...\") 会返回 null 并在很远的地方炸");
        }

        static bool _booted;

        /// <summary>SpecDesignerCommon -- SettingManager, SpecificationInfo, TzpManager, ...</summary>
        public static Assembly A { get; private set; }

        /// <summary>SpecDesigner.FormEditor -- UICreator, ComponentTabIndexService, ...</summary>
        public static Assembly FE { get; private set; }

        /// <summary>The SettingManager singleton (SettingManager.Get()).</summary>
        public static object SettingManager { get; private set; }

        /// <summary>
        /// Idempotent one-time boot. The second and later calls do nothing -- in particular
        /// they do NOT re-point Connection.Workspace, because Application, the merged resource
        /// dictionary and the two loaded assemblies are process-wide state, and re-running
        /// LoadCommonData() for a different workspace is not something any tool has been shown
        /// to survive. A process that needs another module's workspace starts again.
        /// </summary>
        /// <summary>The workspace this process was booted against. Exposed because a process is
        /// permanently bound to one: Boot cannot be re-pointed (see the note above), and the
        /// transport names its pipe after it so that two daemons serving different modules
        /// cannot collide -- which they did, silently, when the pipe name was a constant.</summary>
        public static string Workspace { get; private set; }

        public static void Boot(string workspace) {
            if (_booted) return;
            Workspace = workspace;

            // Say it here, in one line, rather than letting LoadFrom fail on the first assembly
            // and reporting a path nobody chose. A package missing tzs\designer\ is a packaging
            // bug, and this is the message that names it as one.
            if (!Directory.Exists(Install))
                throw new Exception("设计器程序集目录不存在：" + Install
                    + "（发行包应在 tzs\\designer\\ 下带着它；开发时用 TZSCLI_INSTALL 指向已安装的设计器）");

            // The designer's assemblies reference each other and Prism by name only. Without
            // this handler, the first call into SpecificationInfo fails to resolve
            // UndoRedoFramework / Microsoft.Practices.Prism mid-call, which surfaces as a
            // confusing missing-dependency exception from somewhere deep inside the load.
            AppDomain.CurrentDomain.AssemblyResolve += delegate(object s, ResolveEventArgs e) {
                string p = Path.Combine(Install, new AssemblyName(e.Name).Name + ".dll");
                return File.Exists(p) ? Assembly.LoadFrom(p) : null; };

            var app = new Application();

            A  = Assembly.LoadFrom(Path.Combine(Install, "SpecDesignerCommon.dll"));
            FE = Assembly.LoadFrom(Path.Combine(Install, "SpecDesigner.FormEditor.dll"));

            // After A and FE, so that walking the install directory can resolve what those two
            // pull in. Nothing between here and the first FindResource needs the dictionary.
            MergeLanguages(app);

            SettingManager = Reflect.Call(A.GetType("SpecDesignerCommon.SettingManager"), "Get");
            object mdl = Reflect.Call(A.GetType("SpecDesignerCommon.Site.ViewModels.SettingModel"), "Create");
            Reflect.SetProp(SettingManager, "CurrentSetting", mdl);
            Reflect.SetProp(Reflect.Prop(mdl, "Connection"), "Workspace", workspace);
            // LoadCommonData() checks mta/ver against the entry assembly's version and throws
            // VersionIncompatibleException when they differ -- see SPEC §7 "版本门禁". The
            // CLI exe therefore has to carry the designer's AssemblyVersion (1.0.0.251).
            Reflect.Call(SettingManager, "LoadCommonData");

            // CRITICAL, and the reason this block cannot be skipped or moved later:
            // PreferenceManager.Current.Settings must be non-null BEFORE the first call that
            // writes an attribute on a loaded element. XmlElement.CheckOverlapping runs on
            // every attribute write during load and its second statement is
            //     if (!PreferenceManager.Current.Settings.ValidateForm) return;
            // (SpecDesignerCommon/ViewModel/XmlElement.cs:2522), so a null Settings is a
            // NullReferenceException inside GetChildNode on the very first package load --
            // before any tool-level code has had a chance to run.
            //
            // ValidateForm=false is not "avoid a modal dialog" here, whatever the older
            // comments said: it is the switch that turns overlap detection OFF for every
            // attribute write. Somebody who turns it on must restore it in a finally, because
            // it is process-wide (SPEC §11.24 (i)).
            object prefs = Activator.CreateInstance(Reflect.Find(A, "PreferenceModel"));
            Reflect.SetProp(prefs, "ValidateForm", false);
            Reflect.SetProp(Reflect.Prop2(Reflect.Find(A, "PreferenceManager"), "Current"), "_preferenceModel", prefs);

            _booted = true;
        }
    }
}
