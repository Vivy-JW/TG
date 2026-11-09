using System;
using System.Collections.Generic;
using System.IO;
using System.Xml.Linq;

namespace TzsCli.Designer {

    /// <summary>
    /// What values a **layout** attribute is allowed to take, read from the designer's own Form
    /// Designer specification -- &lt;workspace&gt;/mta/mod-fd.spec -- rather than guessed.
    ///
    /// WHY THIS EXISTS. set_layout_attr writes through the designer's XmlElement indexer, and the
    /// indexer does not police the value. Measured against the real engine:
    ///
    ///     set_layout_attr  case   = "GARBAGE"   ->  ok:true, applied:true, written:"GARBAGE"
    ///     set_layout_attr  scroll = "MAYBE"     ->  ok:true, applied:true, written:"MAYBE"
    ///
    /// `case` is declared none|lower|upper; `scroll` is a BOOLEAN. Both were written into the
    /// .4fd and reported as success. That is the worst class of error this project has: the caller
    /// is told it worked and has no way to discover otherwise. A rejection it can self-correct
    /// from is strictly better than a silent success, which is the whole argument for this class.
    ///
    /// WHERE THE RULE COMES FROM. mod-fd.spec declares each property once:
    ///
    ///     &lt;PropertyInfo name="case" type="ENUM" initialValue="none"
    ///                   editorInfo="alwaysUpdate:true;contains:none|lower|upper"/&gt;
    ///     &lt;PropertyInfo name="gridWidth" type="INTEGER"
    ///                   editorInfo="alwaysUpdate:true;range:0|4000;isDynamic:true"/&gt;
    ///
    /// 104 of its 260 properties carry a `contains:` (103 ENUM, 1 TEXT). The same file's
    /// &lt;NodeInfo mimeType="modFD/Edit" properties="..."/&gt; is what
    /// ComponentFactory.IsIncludeProperties consults for the *name* whitelist; this class is the
    /// value half of the same declaration.
    ///
    /// LAYOUT ONLY -- and that is a measured decision, not a convenience. mod-fd.spec is the
    /// *Form Designer* specification, so it describes .4fd element attributes. The .tsd spec-node
    /// attributes are a different surface with different value sets: `widget` is declared
    /// contains:ButtonEdit|CheckBox|...|WebComponent with **no Label**, yet the corpus's .tsd files
    /// carry `widget="Label"` 94 times (a form-only field needs no data widget). Running the layout
    /// rule over spec writes would reject 94 legal values in the corpus alone. set_spec_attr is
    /// therefore NOT guarded here -- see its call sites for the note that says so.
    ///
    /// EMPTY IS ALWAYS LEGAL. The corpus carries `hidden=""` 2649 times and `sizePolicy=""` once,
    /// neither of which is in its own declared set. A full scan of the 65 corpus packages found
    /// every **non-empty** value of every contains-attribute inside its declared set, and empty the
    /// only thing outside it: empty means "unset / inherit from the base data".
    ///
    /// NOT CHECKED, on purpose:
    ///
    ///   * `range:x|y` (gridWidth is range:0|4000). Two reasons, and the second is the real one.
    ///     Violations below the floor are already handled -- the indexer clamps against MinGrid* and
    ///     the caller gets E_ATTR_CLAMPED carrying `written` (measured: gridWidth=-3 came back
    ///     `clamped:true, written:"1"`). And nothing tells us the declared range is the MODEL's
    ///     limit rather than the UI spinner's: the widest gridWidth in the 65-package corpus is
    ///     2390, well inside 4000, so the corpus cannot settle it either. A hard reject built on a
    ///     guess is worse than a wide number, so this stays unchecked until something says which
    ///     one it is.
    ///   * Any type other than BOOLEAN and INTEGER. TEXT / FDSTYLE / FDITEM / MULTIPLELINES /
    ///     SOURCECODE and the rest are free-form as far as this file declares them, and `style`
    ///     (FDSTYLE) genuinely takes 22 distinct values in the corpus.
    ///   * Anything mod-fd.spec does not declare. set_tree_source and rename_component are not
    ///     covered by it at all.
    ///   * The spec-node attributes (req, can_edit, can_query, i_zoom, c_zoom, chk_ref, cite_std,
    ///     default, max, min, ...). They are not in mod-fd.spec; they come from the designer's C#
    ///     (SpecFieldNode.Create) and no file declares their value sets. Unchecked, and that gap
    ///     is written down in the docs rather than papered over.
    /// </summary>
    public static class SpecValues {

        /// <summary>One declared property. Only the things we act on are kept.</summary>
        public sealed class Prop {
            public string Type;         // BOOLEAN / ENUM / INTEGER / TEXT / FDSTYLE / ...
            public string[] Contains;   // editorInfo "contains:a|b|c"; null when not declared
            public string Initial;      // initialValue; null when not declared
        }

        /// <summary>The pair a BOOLEAN attribute accepts. Declared once, because two callers who
        /// must agree use it: Check's rejection message (v.Legal) and the discovery surface
        /// (Describe) that tells a caller the set *before* it writes.</summary>
        public static readonly string[] BooleanValues = { "true", "false" };

        /// <summary>The declared shape of one layout attribute -- what `describe_kind` reports so a
        /// caller can write a value it has never written before without guessing.
        ///
        /// WHY IT EXISTS (SPEC §11.9 item 15; found by the 2026-09-25 eval, F13). `describe_kind
        /// --kind layout` used to return a bare list of names, so "隐藏该用 hidden 还是 invisible"
        /// and "这个属性收什么值" had no answer anywhere: the eval's executor guessed `hidden` and
        /// guessed that a BOOLEAN takes lowercase `true`, and a wrong guess here does not error --
        /// it does the other thing (that is the whole reason this class exists). The declared
        /// values were already parsed for the *rejection* path; this is the same table read
        /// forwards.
        ///
        /// `Values` is what Check will ACCEPT, not merely what the file lists: for a BOOLEAN that
        /// is the true/false pair even though mod-fd.spec declares no `contains:` for it. A null
        /// Values means the file declares no value set (TEXT / FDSTYLE / ...: free-form as far as
        /// anything says).</summary>
        public sealed class Declared {
            public string Type;
            public string[] Values;
            public string Initial;
        }

        public static Declared Describe(string workspace, string attr) {
            if (string.IsNullOrEmpty(attr)) return null;
            Dictionary<string, Prop> map = Load(workspace);
            if (map == null) return null;
            Prop p;
            if (!map.TryGetValue(attr, out p)) return null;   // not declared here -> say nothing
            return new Declared {
                Type = p.Type,
                Values = p.Type == "BOOLEAN" ? BooleanValues : p.Contains,
                Initial = p.Initial,
            };
        }

        /// <summary>The answer for one (attribute, value) pair.</summary>
        public sealed class Verdict {
            public bool Ok = true;
            public string[] Legal;      // the declared set, when it was consulted and violated
            public string Suggest;      // nearest declared value, when one is close enough
            public string Type;
            public string Message;
        }

        /// <summary>workspace -> its table. A workspace that has no mod-fd.spec caches null, so a
        /// miss costs one stat() per process rather than one per write.</summary>
        static readonly Dictionary<string, Dictionary<string, Prop>> _cache =
            new Dictionary<string, Dictionary<string, Prop>>(StringComparer.OrdinalIgnoreCase);

        /// <summary>The declared set for one attribute, or null when nothing is declared.
        /// Exposed so a caller can say *why* it is not checking (the docs quote it).</summary>
        public static string[] Legal(string workspace, string attr) {
            Dictionary<string, Prop> map = Load(workspace);
            if (map == null) return null;
            Prop p;
            if (!map.TryGetValue(attr, out p)) return null;
            return p.Contains;
        }

        /// <summary>Null workspace (a one-shot process that never bound one) fails OPEN: no
        /// declaration to read means no rule to apply, and inventing one would refuse writes this
        /// file knows nothing about.</summary>
        public static Verdict Check(string workspace, string attr, string value) {
            var v = new Verdict();
            Dictionary<string, Prop> map = Load(workspace);
            if (map == null || attr == null) return v;

            Prop p;
            if (!map.TryGetValue(attr, out p)) return v;   // not declared here -> not our business
            v.Type = p.Type;

            // Empty is "unset / inherit" and the corpus relies on it (hidden="", sizePolicy="").
            if (value == null || value.Length == 0) return v;

            // `type` is a declaration too, and honouring it is not the same as inventing a rule:
            // scroll="MAYBE" was accepted because `scroll` is BOOLEAN and we were only reading
            // `contains`. A scan of all 65 corpus packages finds no BOOLEAN attribute carrying
            // anything but true/false and no INTEGER attribute that does not parse, so enforcing
            // both costs nothing real -- measured before it was written, not after.
            if (p.Type == "BOOLEAN" && value != "true" && value != "false") {
                v.Ok = false;
                v.Legal = BooleanValues;
                v.Suggest = Nearest(value, v.Legal);
                v.Message = "属性 \"" + attr + "\" 是 BOOLEAN，只接受 true / false";
                return v;
            }
            if (p.Type == "INTEGER") {
                long n;
                if (long.TryParse(value, out n)) return v;
                v.Ok = false;                              // no `legal` list: it is not an enum
                v.Message = "属性 \"" + attr + "\" 是 INTEGER，值 \"" + value + "\" 不是整数";
                return v;
            }

            if (p.Contains == null) return v;              // declared, but with no value set
            for (int i = 0; i < p.Contains.Length; i++)
                if (p.Contains[i] == value) return v;

            v.Ok = false;
            v.Legal = p.Contains;
            v.Suggest = Nearest(value, p.Contains);
            v.Message = "属性 \"" + attr + "\" 的值 \"" + value + "\" 不在设计器声明的合法集里"
                      + "（" + (p.Type == null ? "?" : p.Type) + "）";
            return v;
        }

        // ------------------------------------------------------------------ the spec side's Y/N

        /// <summary>
        /// The three spec-node attributes the designer's own checkboxes write, and the only values
        /// they may take.
        ///
        /// This is NOT read from mod-fd.spec -- that file is the *Form Designer*'s and describes
        /// .4fd elements (see the class comment for why the layout rule must not be reused here).
        /// The domain is declared just as firmly, by three sources that agree:
        ///
        ///   * SpecPropertyEditor.xaml binds exactly four CheckBoxes through CheckedValueConverter
        ///     (Req :398, CanEdit :611, CanQuery :636, Gencode :1409), and that converter's
        ///     ConvertBack writes only "Y" or "N".
        ///   * The transforms compare against "Y" EXACTLY -- SpecNodeTransform.TransformCanEdit is
        ///         formElement.SetAttribute("noEntry", ("Y" == specFieldNode.CanEdit) ? "false" : "true");
        ///     So `can_edit="true"` reads back as CHECKED in the panel (the converter's Convert
        ///     accepts "TRUE" case-insensitively) while the transform computes the OPPOSITE
        ///     side-effect: the panel says editable and the runtime says not. That is a silent
        ///     corruption with no error anywhere, and it is what the SKILL's own example used to
        ///     teach -- which is the only reason this table exists.
        ///   * The corpus (65 packages, .tsd): can_edit Y4553/N1929/""74, can_query Y5263/N1219/""74,
        ///     req N5070/Y1284/""202. Nothing outside {Y, N, ""} has ever been written.
        ///
        /// Hand-written, therefore a drift risk, and accepted on purpose: the alternative is an
        /// error nobody reports. Each of the three checks above is re-runnable when the designer
        /// moves. `gen_code` (the fourth CheckBox, on the `act` node) is deliberately left out --
        /// the corpus scan covered &lt;field&gt; elements, so its domain is not measured here.
        /// </summary>
        static readonly Dictionary<string, string[]> SpecYN = new Dictionary<string, string[]>(StringComparer.Ordinal) {
            { "req",       new string[] { "Y", "N" } },
            { "can_edit",  new string[] { "Y", "N" } },
            { "can_query", new string[] { "Y", "N" } },
        };

        /// <summary>The spec-side half of Check(): a value the designer's own checkbox could not
        /// have produced is refused, because the transforms compare against "Y" literally. No
        /// workspace is needed -- the table is above.</summary>
        public static Verdict CheckSpec(string attr, string value) {
            var v = new Verdict();
            string[] legal;
            if (attr == null || !SpecYN.TryGetValue(attr, out legal)) return v;
            v.Type = "YESNO";
            if (value == null || value.Length == 0) return v;   // empty = unset, as everywhere else
            for (int i = 0; i < legal.Length; i++) if (legal[i] == value) return v;
            v.Ok = false;
            v.Legal = legal;
            v.Suggest = Nearest(value, legal);
            v.Message = "属性 \"" + attr + "\" 是设计器的勾选位，只接受 Y / N（空串＝没设）；"
                      + "写别的值面板与运行时会对不上";
            return v;
        }

        // ------------------------------------------------------------------ nearest-name / nearest-value

        /// <summary>
        /// The closest member of `pool` to `want`, or null when nothing is close enough to be worth
        /// saying. Used for BOTH halves of "you got the name wrong" and "you got the value wrong":
        /// the caller's next move is the same either way, and so is the hint's shape.
        ///
        /// Case-insensitive, because `notNull` for `notnull` and `UPPER` for `upper` are the two
        /// mistakes this is for. The threshold is deliberately tight -- 1 edit for names of four
        /// characters or fewer, 2 beyond that -- because a wrong guess is worse than no guess:
        /// distance("case", "name") is 2, and suggesting `name` to somebody who typed `case` would
        /// cost them a round trip and their trust in the field.
        /// </summary>
        public static string Nearest(string want, IEnumerable<string> pool) {
            if (want == null || pool == null) return null;
            int limit = want.Length <= 4 ? 1 : 2;
            string best = null;
            int bestD = int.MaxValue;
            foreach (string c in pool) {
                if (c == null) continue;
                int d = Distance(want, c, limit);
                if (d < bestD) { bestD = d; best = c; }
            }
            return best != null && bestD <= limit ? best : null;
        }

        /// <summary>Levenshtein with an early exit: once every cell in a row exceeds `limit` the
        /// answer cannot come back under it, so the row is abandoned. Attribute names and enum
        /// values are short, but the pool is 66 names wide and this runs on every refusal.</summary>
        static int Distance(string a, string b, int limit) {
            int n = a.Length, m = b.Length;
            if (n - m > limit || m - n > limit) return limit + 1;
            var prev = new int[m + 1];
            var cur = new int[m + 1];
            for (int j = 0; j <= m; j++) prev[j] = j;
            for (int i = 1; i <= n; i++) {
                cur[0] = i;
                int rowMin = cur[0];
                for (int j = 1; j <= m; j++) {
                    int cost = Same(a[i - 1], b[j - 1]) ? 0 : 1;
                    int d = prev[j - 1] + cost;
                    if (prev[j] + 1 < d) d = prev[j] + 1;
                    if (cur[j - 1] + 1 < d) d = cur[j - 1] + 1;
                    cur[j] = d;
                    if (d < rowMin) rowMin = d;
                }
                if (rowMin > limit) return limit + 1;
                var t = prev; prev = cur; cur = t;
            }
            return prev[m];
        }

        /// <summary>Ordinal-ignore-case single characters. char.ToLowerInvariant would fold the
        /// Turkish dotted I into something else; the attribute names here are ASCII.</summary>
        static bool Same(char x, char y) {
            if (x == y) return true;
            return char.ToUpperInvariant(x) == char.ToUpperInvariant(y);
        }

        // ------------------------------------------------------------------ loading

        static Dictionary<string, Prop> Load(string workspace) {
            if (string.IsNullOrEmpty(workspace)) return null;
            lock (_cache) {
                if (_cache.ContainsKey(workspace)) return _cache[workspace];
                Dictionary<string, Prop> map = Read(workspace);
                _cache[workspace] = map;
                return map;
            }
        }

        static Dictionary<string, Prop> Read(string workspace) {
            string f = Path.Combine(Path.Combine(workspace, "mta"), "mod-fd.spec");
            if (!File.Exists(f)) return null;          // not a designer workspace; nothing to read
            XDocument doc;
            try { doc = XDocument.Load(f); }
            catch { return null; }                     // a malformed spec must not break writing

            var map = new Dictionary<string, Prop>(StringComparer.Ordinal);
            foreach (XElement e in doc.Descendants("PropertyInfo")) {
                XAttribute na = e.Attribute("name");
                if (na == null) continue;

                Prop p;
                if (!map.TryGetValue(na.Value, out p)) { p = new Prop(); map[na.Value] = p; }

                XAttribute ty = e.Attribute("type");
                if (ty != null && p.Type == null) p.Type = ty.Value;

                XAttribute iv = e.Attribute("initialValue");
                if (iv != null && p.Initial == null) p.Initial = iv.Value;

                XAttribute ei = e.Attribute("editorInfo");
                if (ei == null) continue;
                foreach (string part in ei.Value.Split(';')) {
                    int c = part.IndexOf(':');
                    if (c <= 0) continue;
                    if (string.CompareOrdinal(part.Substring(0, c), "contains") != 0) continue;
                    string[] vals = part.Substring(c + 1).Split('|');
                    p.Contains = p.Contains == null ? vals : Union(p.Contains, vals);
                }
            }
            return map;
        }

        /// <summary>Two PropertyInfo entries can share a name (mod-fd.spec includes core-br.spec,
        /// which declares some of the same ones). Union rather than last-wins: accepting more is
        /// the safe direction for a rule whose job is to refuse.</summary>
        static string[] Union(string[] a, string[] b) {
            var seen = new List<string>(a);
            for (int i = 0; i < b.Length; i++) {
                bool dup = false;
                for (int j = 0; j < seen.Count; j++) if (seen[j] == b[i]) { dup = true; break; }
                if (!dup) seen.Add(b[i]);
            }
            return seen.ToArray();
        }
    }
}
