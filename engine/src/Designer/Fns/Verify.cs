using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;
using DS = TzsCli.Designer.Session;

namespace TzsCli.Designer.Fns
{
    /// <summary>
    /// `verify`: test/RoundTrip.cs asked as a question, read-only, answered as JSON.
    ///
    /// RoundTrip.exe reloads a produced .tzs, lets the *designer* regenerate .4fd/.tsd/.bdx from
    /// its own model, and diffs the result against what is actually in the package. It is the
    /// strongest self-contained question available -- "if the designer opened this and saved it,
    /// would it write the same thing?" -- and its answer is a fixed-point property, not a guess
    /// about intent. That program is the judge and is not this agent's file to touch, so this is
    /// the same comparison re-implemented against a live session.
    ///
    /// The three differences from RoundTrip.exe, all of them consequences of being a function
    /// rather than a batch judge:
    ///
    /// * It compares against the *session's* in-memory model instead of re-loading the file, so
    ///   it can be asked about an edited handle as well as a pristine one.
    /// * It writes nothing and marks nothing dirty -- SaveToForm / SaveToTSD / SaveBinding assign
    ///   in-memory snapshots, and no entry is repacked.
    /// * It reports the names behind every count, so a caller can diagnose from the response
    ///   alone instead of re-running the judge by hand.
    ///
    /// Everything else follows RoundTrip's decisions exactly, including the two that look like
    /// details and are not:
    ///
    /// * <b>fieldId is excluded from the attribute comparison.</b> It is renumbered wholesale on
    ///   every save and is not a persistent identifier -- that is why addressing is by name. Not
    ///   excluding it flags 20 of 89 designer-authored packages for something that is not a
    ///   defect, and buries the class of problem this comparison exists to catch (a stale value
    ///   left behind by a text splice).
    /// * <b>The path comparison is over sets, not lists.</b> Both sides collapse duplicates, so a
    ///   duplicated element shows up as "not dropped" rather than as a count change.
    /// </summary>
    public static class Verify
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["verify"] = Run;
        }

        static object Run(DS s0, JObject args) {
            DS s = TzsCli.Designer.Fns.Session.Resolve(s0, args);
            TzsCli.Designer.Fns.Validate.RejectPath(args);

            byte[] zip = File.ReadAllBytes(s.Path);
            string fdEntry  = Reflect.EntryName(zip, ".4fd");
            string tsdEntry = Reflect.EntryName(zip, ".tsd");
            string bdxEntry = Reflect.EntryName(zip, ".bdx");
            if (fdEntry == null || tsdEntry == null)
                throw new TzsError("internal", "包里缺 .4fd 或 .tsd: " + s.Path);
            string fdIn  = Reflect.EntryText(zip, fdEntry);
            string tsdIn = Reflect.EntryText(zip, tsdEntry);
            string bdxIn = bdxEntry == null ? null : Reflect.EntryText(zip, bdxEntry);

            // The designer's own regeneration, in the order its save path uses. In memory only:
            // no TzsRepacker, no File.WriteAllBytes, and the model is not marked dirty.
            var fdOut  = (XElement)Reflect.Call(s.Si, "SaveToForm");
            var tsdOut = (XElement)Reflect.Call(s.Si, "SaveToTSD");
            Reflect.Call(s.Si, "SaveBinding");

            XElement fdFormIn  = XElement.Parse(fdIn).Element("Form");
            XElement fdFormOut = fdOut == null ? null : fdOut.Element("Form");
            if (fdFormIn == null || fdFormOut == null)
                throw new TzsError("internal", "读入或重生成的 .4fd 没有 <Form>: " + s.Path);

            var kin = new HashSet<string>(TsdKeys(XElement.Parse(tsdIn)));
            var kout = new HashSet<string>(TsdKeys(tsdOut));
            var tsdAdd  = kout.Except(kin).OrderBy(x => x).ToList();
            var tsdDrop = kin.Except(kout).OrderBy(x => x).ToList();

            var fin  = new HashSet<string>(FormKeys(fdFormIn));
            var fout = new HashSet<string>(FormKeys(fdFormOut));
            var fdAdd  = fout.Except(fin).OrderBy(x => x).ToList();
            var fdDrop = fin.Except(fout).OrderBy(x => x).ToList();

            // Attribute-value diff: the path-level comparison only says whether an element is
            // there, this says whether its values are right. It is what catches a stale attribute
            // left behind by a text splice, which the path diff and a canonical-XML compare both
            // sail past (the designer reshuffles attribute order on every save, so canonical
            // equality reports DIFFERS on files that are perfectly fine).
            var ain  = AttrMap(fdFormIn);
            var aout = AttrMap(fdFormOut);
            var staleNames = new List<string>();
            int stale = 0;
            foreach (var kv in ain) {
                Dictionary<string, string> other;
                if (!aout.TryGetValue(kv.Key, out other)) continue;
                foreach (var a in kv.Value) {
                    string ov;
                    if (!other.TryGetValue(a.Key, out ov) || ov == a.Value) continue;
                    stale++;
                    if (staleNames.Count < 20) staleNames.Add(kv.Key + "." + a.Key + ": " + a.Value + " → " + ov);
                }
            }

            int layoutElems = fdFormOut.DescendantsAndSelf().Count();
            string bdxOut = Reflect.Prop(s.Tzp, "ElementBindings") as string;
            string tsdOutText = tsdOut == null ? null : tsdOut.ToString(SaveOptions.DisableFormatting);

            return new Dictionary<string, object> {
                // The eight fields RoundTrip's summary columns carry, in its order.
                { "tsdAdds",    tsdAdd.Count },
                { "tsdDrops",   tsdDrop.Count },
                { "fdPathAdds", fdAdd.Count },
                { "fdPathDrops", fdDrop.Count },
                { "stale",      stale },
                { "tsdNodesIn", kin.Count },
                { "tsdNodesOut", kout.Count },
                { "layoutElems", layoutElems },
                // ... and the names behind them, so the response is diagnosable on its own.
                { "tsdAddNames",    tsdAdd },
                { "tsdDropNames",   tsdDrop },
                { "fdPathAddNames", fdAdd },
                { "fdPathDropNames", fdDrop },
                { "staleAttrs",     staleNames },
                // The canonical-equality half of RoundTrip's verdict, computed with its own Canon.
                { "fdIdentical",  Canon(fdIn) == Canon(fdOut.ToString(SaveOptions.DisableFormatting)) },
                { "tsdIdentical", Canon(tsdIn) == Canon(tsdOutText) },
                { "bdxInPackage", bdxIn != null },
                { "bdxIdentical", bdxIn == null ? (object)null : Canon(bdxIn) == Canon(bdxOut) },
                { "codeTemplate", CodeTemplate(tsdIn) },
                { "env",          Reflect.S(Reflect.Prop(s.Si, "Env")) },
            };
        }

        // ------------------------------------------------------------------ RoundTrip's algorithms
        //
        // Copied rather than shared: RoundTrip.cs is the judge and is explicitly not to be
        // modified or linked against (SPEC §11.24 (j)). Diverging from it here would make
        // `verify` a second, quietly different judge.

        /// <summary>Canonical form: whitespace-insensitive, attribute order preserved.</summary>
        static string Canon(string xml) {
            if (xml == null) return null;
            try { return XElement.Parse(xml).ToString(SaveOptions.DisableFormatting); }
            catch { return null; }
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
        /// its own tag and name are unchanged.
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

        /// <summary>name-path -> attributes, with fieldId excluded (see the class comment).</summary>
        static Dictionary<string, Dictionary<string, string>> AttrMap(XElement element) {
            var map = new Dictionary<string, Dictionary<string, string>>();
            foreach (var e in element.Descendants()) {
                var stack = new List<string>();
                for (var c = e; c != null && c != element.Parent; c = c.Parent)
                    stack.Insert(0, (string)c.Attribute("name") ?? c.Name.LocalName);
                var d = new Dictionary<string, string>();
                foreach (var a in e.Attributes()) {
                    if (a.Name.LocalName == "fieldId") continue;
                    d[a.Name.LocalName] = a.Value;
                }
                map[string.Join("/", stack.ToArray())] = d;
            }
            return map;
        }

        /// <summary>The form's code_template, which decides the can_edit branch and is the axis
        /// that correlates with the known posX drift (TASKS.md 样本集).</summary>
        static string CodeTemplate(string tsd) {
            if (tsd == null) return null;
            int i = tsd.IndexOf("<code_template");
            if (i < 0) return null;
            int j = tsd.IndexOf("value=\"", i);
            if (j < 0) return null;
            j += 7;
            int k = tsd.IndexOf('"', j);
            return k < 0 ? null : tsd.Substring(j, k - j);
        }
    }
}
