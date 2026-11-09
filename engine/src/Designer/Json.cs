using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Text;
using TzsCli;

namespace TzsCli.Designer
{
    /// <summary>
    /// The read side: the form as the designer's own 画面结构 panel shows it, dumped as compact
    /// JSON with every node carrying the name-path the write operations take as an argument.
    ///
    /// Moved from test/Edit.cs:750-1090. The only change is the one SPEC §11.24 (j) requires:
    /// _specDic and _infoCount were module statics, and two handles open in one process would
    /// cross-talk through them, so they are parameters now. The session's Si/Tzp replace the
    /// module statics si/tzp the same way.
    /// </summary>
    public static class Json
    {
        public static string J(string s) {
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

        /// <summary>
        /// stderr as UTF-8 bytes, for the same reason stdout is: .NET hands Console.Error the OEM
        /// code page (GBK here), so a redirected stderr turns every Chinese diagnostic into bytes
        /// the caller cannot decode. Not disposed -- it wraps a standard handle that outlives us.
        ///
        /// Carried over from Edit.cs:783: it is not part of the JSON layer, but EmitInfo and
        /// EmitFind are its only callers, and they cannot be ported without it.
        /// </summary>
        public static void Err(string msg) {
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
        public const char QT = (char)34;

        public static void Key(StringBuilder sb, string k) { sb.Append(QT).Append(k).Append(QT).Append(':'); }

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
        public static string StatusLetter(object specNode) {
            string st = Reflect.S(Reflect.Prop(specNode, "Status"));
            if (st == "DELETE") return "d";
            if (st == "MODIFY") return "u";
            if (st == "CREATE") return "c";
            return null;
        }

        /// <summary>
        /// The fields every info node carries, shared by InfoNode (full tree) and InfoRef (flat
        /// match) so the two cannot drift. specNodeType in particular is what decides which
        /// &lt;nodeKind&gt; a `set` should use, so a second copy would be a real hazard.
        ///
        /// specDic is the session's FormSpeDictionary, passed in rather than held in a static
        /// (SPEC §11.24 (j)).
        /// </summary>
        public static void InfoFields(object el, string path, StringBuilder sb, IDictionary specDic) {
            string name = (string)Session.Raw(el, "name");

            Key(sb, "tag");   sb.Append(J(Reflect.S(Reflect.Prop(el, "NodeName"))));
            sb.Append(','); Key(sb, "name"); sb.Append(J(name));
            sb.Append(','); Key(sb, "path"); sb.Append(J(path));

            string ti = (string)Session.Raw(el, "tabIndex");
            if (!string.IsNullOrEmpty(ti)) { sb.Append(','); Key(sb, "tabIndex"); sb.Append(ti); }

            string tab = (string)Session.Raw(el, "sqlTabName"), col = (string)Session.Raw(el, "colName");
            if (!string.IsNullOrEmpty(tab)) { sb.Append(','); Key(sb, "table");  sb.Append(J(tab)); }
            if (!string.IsNullOrEmpty(col)) { sb.Append(','); Key(sb, "column"); sb.Append(J(col)); }

            string ft = (string)Session.Raw(el, "fieldType");
            if (!string.IsNullOrEmpty(ft)) { sb.Append(','); Key(sb, "fieldType"); sb.Append(J(ft)); }

            if (name != null && specDic != null && specDic.Contains(name)) {
                object fsm = specDic[name];
                sb.Append(','); Key(sb, "specNodeType"); sb.Append(J(Reflect.S(Reflect.Prop(fsm, "SpecNodeType"))));
                object node = Reflect.Prop(fsm, "SpecNode");
                if (node != null) {
                    string desc = StatusLetter(node);
                    if (desc != null) { sb.Append(','); Key(sb, "specStatus"); sb.Append(J(desc)); }
                }
            }
        }

        /// <summary>
        /// One match without its subtree: a find can return many, and `info &lt;path&gt;` already
        /// expands any single one. childCount is kept because having children is what makes an
        /// element a legal target for add/wrap.
        /// </summary>
        public static void InfoRef(object el, string path, StringBuilder sb, IDictionary specDic) {
            sb.Append('{');
            InfoFields(el, path, sb, specDic);
            int n = Session.ChildCount(el);
            if (n > 0) { sb.Append(','); Key(sb, "childCount"); sb.Append(n); }
            sb.Append('}');
        }

        /// <summary>Recursive tree node. infoCount is a counter rather than a static so that two
        /// concurrent dumps cannot corrupt each other's elementCount (SPEC §11.24 (j)).</summary>
        public static void InfoNode(object el, string path, StringBuilder sb, int depth, ref int infoCount, IDictionary specDic) {
            infoCount++;
            sb.Append('{');
            InfoFields(el, path, sb, specDic);
            if (Session.ChildCount(el) > 0) {
                sb.Append(','); Key(sb, "children"); sb.Append('[');
                bool firstChild = true;
                foreach (var c in (IEnumerable)Reflect.Prop(el, "Nodes")) {
                    if (!firstChild) sb.Append(',');
                    firstChild = false;
                    InfoNode(c, path + "/" + Session.Seg(c), sb, depth + 1, ref infoCount, specDic);
                }
                sb.Append(']');
            }
            sb.Append('}');
        }

        /// <summary>
        /// The whole form as one JSON document, written to stdout as UTF-8 bytes.
        ///
        /// Unlike the original, this does NOT call Environment.Exit when pathFilter does not
        /// resolve -- it throws TzsError("not_found"). The original's inline exit was right for
        /// Edit.exe, where a bad path is the end of the process, but in the long-lived server
        /// the same call would kill the daemon on the first bad argument. The transport maps
        /// the exception to error.kind; see SPEC §11.24 (a) and (h).
        /// </summary>
        public static void EmitInfo(Session s, byte[] zip, string fdEntry, string tsdEntry, string pathFilter) {
            // JSON is UTF-8 by definition, and .NET's console defaults to the OEM code page (GBK
            // here), which mangles any non-ASCII name into bytes no JSON reader will accept.

            string fdText = Reflect.EntryText(zip, fdEntry);
            var idx = ElementIndex.Build(fdText);
            string rootName = idx.All.Count > 0 ? idx.All[0].Name : "";
            object formNode = Reflect.Prop(s.Si, "FormNode");
            IDictionary specDic = (IDictionary)Reflect.Prop(s.Si, "FormSpeDictionary");
            int infoCount = 0;

            object start = formNode;
            string startPath = rootName + "/" + Reflect.S(Reflect.Prop(formNode, "Name"));
            if (pathFilter != null) {
                start = s.FindByPath(pathFilter);
                if (start == null) {
                    throw TzsError.NotFound("路径", pathFilter);
                }
                startPath = pathFilter;
            }

            string tsdText = Reflect.EntryText(zip, tsdEntry);
            string tpl = "";
            int ti = tsdText == null ? -1 : tsdText.IndexOf("<code_template");
            if (ti >= 0) {
                int v = tsdText.IndexOf("value=\"", ti);
                if (v >= 0) { v += 7; int e2 = tsdText.IndexOf('"', v); if (e2 > v) tpl = tsdText.Substring(v, e2 - v); }
            }

            var sb = new StringBuilder();
            sb.Append("{\n");
            sb.Append("  \"program\": ").Append(J(Reflect.S(Reflect.Prop(s.Tzp, "ProgramName")))).Append(",\n");
            sb.Append("  \"env\": ").Append(J(Reflect.S(Reflect.Prop(s.Si, "Env")))).Append(",\n");
            sb.Append("  \"codeTemplate\": ").Append(J(tpl)).Append(",\n");
            sb.Append("  \"specDictionarySize\": ").Append(specDic == null ? 0 : specDic.Count).Append(",\n");
            sb.Append("  \"root\": ");
            InfoNode(start, startPath, sb, 1, ref infoCount, specDic);
            sb.Append(",\n  \"elementCount\": ").Append(infoCount).Append("\n}\n");
            // Console.Out runs through the OEM code page (GBK here) even when OutputEncoding is
            // set, so write UTF-8 bytes to the raw stream -- JSON has to be UTF-8.
            using (var stdout = new StreamWriter(Console.OpenStandardOutput(), new UTF8Encoding(false))) {
                stdout.Write(sb.ToString());
                stdout.Flush();
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
        public static void EmitFind(Session s, byte[] zip, string fdEntry, string term) {
            if (string.IsNullOrEmpty(term)) {
                throw TzsError.Validation("--find 需要一个控件代号");
            }

            string fdText = Reflect.EntryText(zip, fdEntry);
            var idx = ElementIndex.Build(fdText);
            string rootName = idx.All.Count > 0 ? idx.All[0].Name : "";
            object formNode = Reflect.Prop(s.Si, "FormNode");
            IDictionary specDic = (IDictionary)Reflect.Prop(s.Si, "FormSpeDictionary");

            // A code can name an action rather than a component. 新增项目 writes an <act id="...">
            // with no <Button> behind it, and such an id appears in no layout element -- so without
            // this it comes back as a bare empty list that reads like a typo. Actions are NOT in
            // FormSpeDictionary (Rename guards them with a separate CheckActionNameIsExists), which
            // is exactly why the element walk above cannot see them.
            var els = new List<object>();
            var paths = new List<string>();
            var actHits = new List<object>();
            // Same start as EmitInfo: the ManagedForm root's name, then the <Form>'s.
            string startPath = rootName + "/" + Session.Seg(formNode);
            Session.CollectByName(formNode, startPath, term, false, els, paths);
            s.CollectActs(term, false, actHits);
            bool exact = els.Count > 0 || actHits.Count > 0;
            if (!exact) {
                Session.CollectByName(formNode, startPath, term, true, els, paths);
                s.CollectActs(term, true, actHits);
            }

            var sb = new StringBuilder();
            sb.Append("{\n");
            sb.Append("  \"program\": ").Append(J(Reflect.S(Reflect.Prop(s.Tzp, "ProgramName")))).Append(",\n");
            sb.Append("  \"env\": ").Append(J(Reflect.S(Reflect.Prop(s.Si, "Env")))).Append(",\n");
            sb.Append("  \"query\": ").Append(J(term)).Append(",\n");
            sb.Append("  \"exact\": ").Append(exact ? "true" : "false").Append(",\n");
            sb.Append("  \"matchCount\": ").Append(els.Count).Append(",\n");
            sb.Append("  \"matches\": [");
            for (int i = 0; i < els.Count; i++) {
                if (i > 0) sb.Append(',');
                InfoRef(els[i], paths[i], sb, specDic);
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
                    Key(sb, "id"); sb.Append(J(Reflect.S(Reflect.Prop(a, "Name"))));
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
    }
}
