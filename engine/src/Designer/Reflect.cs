using System;
using System.IO;
using System.IO.Compression;
using System.Reflection;
using System.Text;
using System.Xml.Linq;

namespace TzsCli.Designer
{
    /// <summary>
    /// The reflection plumbing every tool in this repo used to carry its own copy of.
    ///
    /// All of it is called *through* reflection rather than against a compile-time reference
    /// on purpose: the designer's assemblies are loaded out of INSTALL at runtime
    /// (Designer.Boot), so nothing here can be named in a using directive.
    ///
    /// Ported verbatim from test/Edit.cs (helpers at :50-126) so that the existing programs
    /// and the coming function layer share one implementation instead of N drifting ones.
    /// </summary>
    public static class Reflect
    {
        /// <summary>
        /// Reflection invoke that disambiguates overloads by argument type, not just arity.
        /// SpecificationInfo.Remove has both Remove(string) and a one-arg Remove(SpecActionNode),
        /// and picking by arity alone lands on whichever GetMethods() happened to return first --
        /// which throws ArgumentException or, worse, silently removes the wrong thing.
        ///
        /// Overloads are filtered by parameter-type assignability; if none fits, the first
        /// same-arity candidate is used so that the binder raises its own (informative) error
        /// rather than this method raising a vague one.
        ///
        /// This is deliberately NOT the weaker arity-only Call from test/AddField.cs:58.
        /// SPEC §11.24 0.3 froze the type-aware version: the weak one selects by parameter
        /// count alone and only catches TargetInvocationException, so an ArgumentException from
        /// a wrongly-chosen overload propagates straight out of it.
        /// </summary>
        public static object Call(object t, string n, params object[] a) {
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

        /// <summary>Invoke one reflected method, unwrapping TargetInvocationException so callers
        /// see the designer's own exception type and stack rather than a reflection wrapper.</summary>
        public static object Invoke(MethodInfo m, object t, object[] a) {
            try { return m.Invoke(t is Type ? null : t, a); }
            catch (TargetInvocationException ex) { System.Runtime.ExceptionServices.ExceptionDispatchInfo.Capture(ex.InnerException ?? ex).Throw(); return null; }
        }

        /// <summary>Property first, then field -- the designer mixes both, and the fields are
        /// often the private backing stores (e.g. SpecificationInfo._TSDValidater).</summary>
        public static object Prop(object o, string n) {
            if (o == null) return null;
            var t = o.GetType();
            var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
            if (p != null) return p.GetValue(o, null);
            var f = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
            return f == null ? null : f.GetValue(o);
        }

        /// <summary>Static members only, and the receiver is a Type rather than an instance --
        /// SettingManager / PreferenceManager / TzpManager.Current are all statics.</summary>
        public static object Prop2(Type t, string n) {
            var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
            if (p != null) return p.GetValue(null, null);
            var f = t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Static);
            return f == null ? null : f.GetValue(null);
        }

        public static void SetProp(object o, string n, object v) {
            var t = o.GetType();
            var p = t.GetProperty(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance);
            if (p != null) { p.SetValue(o, v, null); return; }
            t.GetField(n, BindingFlags.Public|BindingFlags.NonPublic|BindingFlags.Instance).SetValue(o, v);
        }

        /// <summary>Type by simple name. The designer's assemblies expose dozens of nested or
        /// namespaced types whose full name is not worth tracking here.</summary>
        public static Type Find(Assembly a, string name) {
            foreach (var t in a.GetTypes()) if (t.Name == name) return t;
            return null;
        }

        /// <summary>null renders as "(null)" -- the repo-wide convention for a diagnostic. Note
        /// the trap it creates: S(null) is never null, so code that must tell "absent" from
        /// "present but empty" uses Session.Seg / Reflect.Prop directly instead.</summary>
        public static string S(object o) { return o == null ? "(null)" : o.ToString(); }

        public static string Attr(XElement e, string n) {
            var a = e == null ? null : e.Attribute(n);
            return a == null ? null : a.Value;
        }

        /// <summary>Name of a zip entry by suffix, e.g. ".4fd" -- the workspace renames the stem
        /// per program, so only the extension is stable.</summary>
        public static string EntryName(byte[] zip, string suffix) {
            using (var ms = new MemoryStream(zip))
            using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
                foreach (var e in z.Entries) if (e.FullName.EndsWith(suffix)) return e.FullName;
            return null;
        }

        public static string EntryText(byte[] zip, string name) {
            using (var ms = new MemoryStream(zip))
            using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
            using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
                return r.ReadToEnd();
        }
    }
}
