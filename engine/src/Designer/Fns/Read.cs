using System;
using System.Collections;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;
using TzsCli;

namespace TzsCli.Designer
{
    // Deliberately NOT TzsCli.Designer.Fns like the other three modules in this
    // directory. Fns/Session.cs declares a static module class also called Session, and
    // a type in the same namespace outranks an enclosing one -- so inside .Fns the name
    // `Session` would bind to that module and every `Session s` parameter here would be
    // CS0721 (static types cannot be used as parameters). W2-B hit this and chose the
    // enclosing namespace; the alias route fails because a using-alias does not outrank
    // a same-namespace type. Rpc.BuildMap dispatches by class name, so the split costs
    // nothing. Do not "tidy" it without renaming the module class first.

    /// <summary>
    /// The read side of the function layer: the same form-as-a-tree the designer's own
    /// 画面结构 panel shows (SPEC §11.20), plus the model lookups the write functions need
    /// as input -- the spec-node attribute sets, the data dictionary, the records and the
    /// local strings.
    ///
    /// Each function is an <see cref="Fn"/>: it returns a JSON-serialisable object and
    /// signals failure by throwing. Nothing here writes to the model, so none of them needs
    /// an UndoRedoManager and none transitions the handle Loaded -> Mutable.
    ///
    /// form_tree / find_component are the bodies of Json.EmitInfo / Json.EmitFind (Edit.cs
    /// :899/:993) with the stdout write replaced by a return value. The JSON text is still
    /// produced by the same Json.InfoNode / Json.InfoRef builders and then parsed back into
    /// a JObject, so the two surfaces cannot drift: the acceptance test for these two is a
    /// structural comparison against `Edit.exe info` output, and re-typing the field list
    /// by hand is exactly how such a comparison starts failing.
    /// </summary>
    public static class Read
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["form_tree"]         = FormTree;
            into["find_component"]    = FindComponent;
            into["get_component"]     = GetComponent;
            into["describe_kind"]     = DescribeKind;
            into["list_spec_nodes"]   = ListSpecNodes;
            into["list_packages"]     = ListPackages;
            into["list_tables"]       = ListTables;
            into["list_columns"]      = ListColumns;
            into["list_records"]      = ListRecords;
            into["list_local_strings"] = ListLocalStrings;
        }

        // ================================================================ argument plumbing

        internal static string Arg(JObject a, string n) {
            if (a == null) return null;
            JToken t = a[n];
            if (t == null || t.Type == JTokenType.Null) return null;
            if (t.Type == JTokenType.String) return (string)t;
            return t.ToString();
        }

        /// <summary>Required and non-empty. For values where "" is a legal payload (clearing an
        /// attribute), use NeedKey instead.</summary>
        internal static string Need(JObject a, string n) {
            string v = Arg(a, n);
            if (string.IsNullOrEmpty(v)) throw TzsError.Validation("缺少参数 " + n);
            return v;
        }

        /// <summary>Required but may be the empty string: `value:""` is how an attribute is
        /// cleared, and rejecting it would make the empty value unreachable.</summary>
        internal static string NeedKey(JObject a, string n) {
            if (a == null || a[n] == null || a[n].Type == JTokenType.Null)
                throw TzsError.Validation("缺少参数 " + n);
            string v = Arg(a, n);
            return v == null ? "" : v;
        }

        internal static int IntArg(JObject a, string n, int dflt) {
            string s = Arg(a, n);
            int v;
            if (s == null || !int.TryParse(s, out v)) return dflt;
            return v;
        }

        internal static string[] ListArg(JObject a, string n) {
            if (a == null) return null;
            JToken t = a[n];
            if (t == null || t.Type == JTokenType.Null) return null;
            if (t.Type == JTokenType.String) return new string[] { (string)t };
            var list = new List<string>();
            foreach (JToken e in (JArray)t) list.Add(e.ToString());
            return list.ToArray();
        }

        // ================================================================ session plumbing

        internal static IDictionary SpecDic(Session s) {
            return (IDictionary)Reflect.Prop(s.Si, "FormSpeDictionary");
        }

        internal static object FormNode(Session s) { return Reflect.Prop(s.Si, "FormNode"); }

        internal static byte[] Zip(Session s) { return File.ReadAllBytes(s.Path); }

        internal static string Entry(byte[] zip, string suffix) {
            string n = Reflect.EntryName(zip, suffix);
            if (n == null) throw TzsError.NotFound("包内条目", suffix);
            return n;
        }

        /// <summary>First segment of every name-path: the .4fd document root's own name. It is
        /// read from the text rather than from the model because the model root is the
        /// &lt;Form&gt;, one level below it (same reason Edit.cs:905 does it).</summary>
        internal static string RootName(string fdText) {
            var idx = ElementIndex.Build(fdText);
            return idx.All.Count > 0 ? idx.All[0].Name : "";
        }

        internal static string CodeTemplate(string tsdText) {
            string tpl = "";
            if (tsdText == null) return tpl;
            int ti = tsdText.IndexOf("<code_template");
            if (ti < 0) return tpl;
            int v = tsdText.IndexOf("value=\"", ti);
            if (v < 0) return tpl;
            v += 7;
            int e = tsdText.IndexOf('"', v);
            return e > v ? tsdText.Substring(v, e - v) : tpl;
        }

        internal static string Str(object o) { return o == null ? null : o.ToString(); }

        /// <summary>Attribute names of an XElement in document order -- the materialised set the
        /// designer's own Create wrote plus whatever a load/enrich pass added. This is the
        /// whitelist §11.24 (d) rule 2 requires, and it is read from the live node rather than
        /// from a table here, so it cannot go stale.</summary>
        internal static List<string> AttrNames(XElement e) {
            var names = new List<string>();
            if (e == null) return names;
            foreach (var at in e.Attributes()) names.Add(at.Name.LocalName);
            return names;
        }

        internal static JObject XAttrs(XElement e) {
            var o = new JObject();
            if (e == null) return o;
            foreach (var at in e.Attributes()) o[at.Name.LocalName] = at.Value;
            return o;
        }

        internal static JObject AttrsToJ(IDictionary<string, string> d) {
            var o = new JObject();
            foreach (var kv in d) o[kv.Key] = kv.Value;
            return o;
        }

        internal static JArray StrArray(IEnumerable<string> xs) {
            var arr = new JArray();
            foreach (string x in xs) arr.Add(x);
            return arr;
        }

        // ================================================================ form_tree

        /// <summary>
        /// The whole form as the designer's 画面结构 panel shows it: FormNode plus recursive
        /// Nodes, labelled by Name, every node carrying the name-path the write operations
        /// take. Body of Json.EmitInfo (Edit.cs:899) with the stdout write replaced by a
        /// return value; `path` optionally re-roots the dump at a subtree.
        ///
        /// Throws not_found when the path does not resolve -- Edit.exe called Environment.Exit
        /// (2) there, which in a long-lived server would kill the daemon on the first typo.
        /// </summary>
        public static object FormTree(Session s, JObject a) {
            string pathFilter = Arg(a, "path");
            int maxDepth = IntArg(a, "depth", 0);

            byte[] zip = Zip(s);
            string fdText = Reflect.EntryText(zip, Entry(zip, ".4fd"));
            string rootName = RootName(fdText);
            object formNode = FormNode(s);
            IDictionary specDic = SpecDic(s);
            int infoCount = 0;

            object start = formNode;
            string startPath = rootName + "/" + Reflect.S(Reflect.Prop(formNode, "Name"));
            if (pathFilter != null) {
                start = s.FindByPath(pathFilter);
                if (start == null) throw TzsError.PathNotFound(pathFilter);
                startPath = pathFilter;
            }

            string tpl = CodeTemplate(Reflect.EntryText(zip, Entry(zip, ".tsd")));

            var sb = new StringBuilder();
            sb.Append("{\n");
            sb.Append("  \"program\": ").Append(Json.J(Reflect.S(Reflect.Prop(s.Tzp, "ProgramName")))).Append(",\n");
            sb.Append("  \"env\": ").Append(Json.J(Reflect.S(Reflect.Prop(s.Si, "Env")))).Append(",\n");
            sb.Append("  \"codeTemplate\": ").Append(Json.J(tpl)).Append(",\n");
            sb.Append("  \"specDictionarySize\": ").Append(specDic == null ? 0 : specDic.Count).Append(",\n");
            sb.Append("  \"root\": ");
            // depth 0 (the default) is the unlimited dump, produced by the very builder
            // Edit.exe info uses -- that is what makes the two structurally identical.
            if (maxDepth <= 0) Json.InfoNode(start, startPath, sb, 1, ref infoCount, specDic);
            else InfoNodeDepth(start, startPath, sb, 1, maxDepth, ref infoCount, specDic);
            sb.Append(",\n  \"elementCount\": ").Append(infoCount).Append("\n}\n");
            return JObject.Parse(sb.ToString());
        }

        /// <summary>Json.InfoNode with a level cap. Same field set; when the cap is reached the
        /// `children` key is simply absent, which is how the unlimited version already renders a
        /// leaf -- so a capped node is still a node the caller can feed to any other function.</summary>
        static void InfoNodeDepth(object el, string path, StringBuilder sb, int depth, int maxDepth, ref int infoCount, IDictionary specDic) {
            infoCount++;
            sb.Append('{');
            Json.InfoFields(el, path, sb, specDic);
            if (depth < maxDepth && Session.ChildCount(el) > 0) {
                sb.Append(','); Json.Key(sb, "children"); sb.Append('[');
                bool firstChild = true;
                foreach (var c in (IEnumerable)Reflect.Prop(el, "Nodes")) {
                    if (!firstChild) sb.Append(',');
                    firstChild = false;
                    InfoNodeDepth(c, path + "/" + Session.Seg(c), sb, depth + 1, maxDepth, ref infoCount, specDic);
                }
                sb.Append(']');
            }
            sb.Append('}');
        }

        // ================================================================ find_component

        /// <summary>
        /// `find` -- resolve a 控件代号 to the name-path the write operations take, plus the
        /// flat match list. Body of Json.EmitFind (Edit.cs:993).
        ///
        /// An exact hit suppresses the substring sweep; no match is a valid answer, not an
        /// error (matchCount 0, exit code 0 in the old CLI), because the argument is a query
        /// and not a location. Elements and actions come back in two lists: an &lt;act&gt; is
        /// not in FormSpeDictionary, so the element walk cannot see it.
        /// </summary>
        public static object FindComponent(Session s, JObject a) {
            string term = Need(a, "query");

            byte[] zip = Zip(s);
            string fdText = Reflect.EntryText(zip, Entry(zip, ".4fd"));
            string rootName = RootName(fdText);
            object formNode = FormNode(s);
            IDictionary specDic = SpecDic(s);

            var els = new List<object>();
            var paths = new List<string>();
            var actHits = new List<object>();
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
            sb.Append("  \"program\": ").Append(Json.J(Reflect.S(Reflect.Prop(s.Tzp, "ProgramName")))).Append(",\n");
            sb.Append("  \"env\": ").Append(Json.J(Reflect.S(Reflect.Prop(s.Si, "Env")))).Append(",\n");
            sb.Append("  \"query\": ").Append(Json.J(term)).Append(",\n");
            sb.Append("  \"exact\": ").Append(exact ? "true" : "false").Append(",\n");
            sb.Append("  \"matchCount\": ").Append(els.Count).Append(",\n");
            sb.Append("  \"matches\": [");
            for (int i = 0; i < els.Count; i++) {
                if (i > 0) sb.Append(',');
                Json.InfoRef(els[i], paths[i], sb, specDic);
            }
            sb.Append(']');
            if (actHits.Count > 0) {
                sb.Append(",\n  \"actionMatches\": [");
                for (int i = 0; i < actHits.Count; i++) {
                    if (i > 0) sb.Append(',');
                    object act = actHits[i];
                    sb.Append('{');
                    Json.Key(sb, "id"); sb.Append(Json.J(Reflect.S(Reflect.Prop(act, "Name"))));
                    string desc = Json.StatusLetter(act);
                    if (desc != null) { sb.Append(','); Json.Key(sb, "specStatus"); sb.Append(Json.J(desc)); }
                    sb.Append('}');
                }
                sb.Append(']');
            }
            sb.Append("\n}\n");
            return JObject.Parse(sb.ToString());
        }

        // ================================================================ get_component

        /// <summary>
        /// Everything find_component returns for one element, plus the parts an AI cannot see
        /// anywhere else:
        ///
        ///   spec   -- the spec node's own attributes, read from its Source XElement. This is
        ///             the whole point: req / can_edit / can_query / default / max / min /
        ///             i_zoom / c_zoom / chk_ref / items exist only in the .tsd, so without
        ///             this an AI can locate a field but cannot tell whether its last write
        ///             did anything. Every attribute is materialised (empty when unset), for
        ///             all seven kinds.
        ///   layout -- the element's own layout attributes. They are exactly the legal set for
        ///             set_layout_attr (the indexer's whitelist is presence-based), and the
        ///             caller otherwise has to guess at names like `noEntry`/`required`.
        ///
        /// The locator is `path` (exact name-path) or `query` (控件代号, the same lookup
        /// find_component does); one of them is required. A query that matches several
        /// elements is not resolved by picking a winner -- the candidates come back so the
        /// caller can choose (SPEC §11.20: the designer enforces uniqueness, but a
        /// hand-edited .4fd need not).
        /// </summary>
        public static object GetComponent(Session s, JObject a) {
            string path = Arg(a, "path");
            string query = Arg(a, "query");
            if (string.IsNullOrEmpty(path) && string.IsNullOrEmpty(query))
                throw TzsError.Validation("get_component 需要 path 或 query 之一");

            byte[] zip = Zip(s);
            string fdText = Reflect.EntryText(zip, Entry(zip, ".4fd"));
            string rootName = RootName(fdText);
            object formNode = FormNode(s);
            IDictionary specDic = SpecDic(s);

            object el = null;
            string elPath = null;
            if (!string.IsNullOrEmpty(path)) {
                el = s.FindByPath(path);
                if (el == null) throw TzsError.PathNotFound(path);
                elPath = path;
            } else {
                string[] hit = Resolve(s, formNode, rootName, query);
                el = s.FindByPath(hit[0]);
                elPath = hit[0];
                if (el == null) throw TzsError.PathNotFound(hit[0]);
            }

            string name = (string)Session.Raw(el, "name");

            // The base fields are produced by the same builder find_component uses, for the
            // same reason form_tree reuses InfoNode: two hand-maintained copies of the field
            // list is how the two surfaces start disagreeing.
            var sb = new StringBuilder();
            Json.InfoRef(el, elPath, sb, specDic);
            var res = JObject.Parse(sb.ToString());

            res["layout"] = AttrsToJ(Session.Attrs(el));

            object fsm = name != null && specDic != null && specDic.Contains(name) ? specDic[name] : null;
            string onlyKind = Arg(a, "kind");
            if (onlyKind != null && onlyKind.StartsWith("spec:", StringComparison.Ordinal)) onlyKind = onlyKind.Substring(5);
            if (onlyKind != null && Array.IndexOf(SpecSlots.Kinds, onlyKind) < 0)
                throw TzsError.Validation("未知 kind：" + onlyKind + "；合法值: " + string.Join(",", SpecSlots.Kinds));
            if (fsm != null) {
                var spec = new JObject();
                foreach (string kind in SpecSlots.Kinds) {
                    if (onlyKind != null && onlyKind != kind) continue;
                    object node = Reflect.Prop(fsm, SpecSlots.ByKind[kind]);
                    if (node == null) continue;
                    spec[kind] = SpecEntry(node);
                }
                if (spec.Count > 0) res["spec"] = spec;
                string nt = Str(Reflect.Prop(fsm, "SpecNodeType"));
                if (nt != null && res["specNodeType"] == null) res["specNodeType"] = nt;
            }

            // A pure action (an <act> with no <Button> behind it -- what 新增项目 writes) is in
            // no layout element and in no FormSpeDictionary entry, so it would otherwise come
            // back as "no such component". Its spec node is still addressable, which is what a
            // caller wanting to inspect or change it needs.
            if (res["spec"] == null && (onlyKind == null || onlyKind == "act")) {
                var acts = new List<object>();
                s.CollectActs(name ?? query, false, acts);
                if (acts.Count == 1) {
                    var spec = new JObject();
                    spec["act"] = SpecEntry(acts[0]);
                    res["spec"] = spec;
                    res["kind"] = "act";
                }
            }
            return res;
        }

        /// <summary>One spec node as {status?, attrs:{...}, children?}. Status carry-over is the
        /// same four-state letter as Json.StatusLetter ("c" means in memory only and dropped by
        /// ToXml on save -- SPEC §11.20 -- which is exactly what makes a synthetic node
        /// distinguishable from a persisted one).
        ///
        /// `children` appears only for a node whose Source has element children, i.e. the
        /// &lt;tree&gt; data source with its type/id/pid/desc/... cells. It is what makes
        /// set_tree_source callable: that function addresses one cell by element name and one
        /// attribute of it, and without this the caller would have to discover both by provoking
        /// the whitelist error.</summary>
        internal static JObject SpecEntry(object node) {
            var entry = new JObject();
            string st = Json.StatusLetter(node);
            if (st != null) entry["status"] = st;
            XElement src = Reflect.Prop(node, "Source") as XElement;
            entry["attrs"] = XAttrs(src);
            if (src != null && src.HasElements) {
                var kids = new JObject();
                foreach (XElement c in src.Elements()) kids[c.Name.LocalName] = XAttrs(c);
                entry["children"] = kids;
            }
            return entry;
        }

        /// <summary>
        /// 控件代号 -> exactly one name-path. Exact match first; only when that is empty does
        /// it fall back to the substring sweep, so a code that also prefixes another element
        /// still resolves to itself. Several hits is reported as the candidate list rather
        /// than resolved arbitrarily.
        /// </summary>
        internal static string[] Resolve(Session s, object formNode, string rootName, string term) {
            string startPath = rootName + "/" + Session.Seg(formNode);
            var els = new List<object>();
            var paths = new List<string>();
            Session.CollectByName(formNode, startPath, term, false, els, paths);
            if (paths.Count == 0) Session.CollectByName(formNode, startPath, term, true, els, paths);
            if (paths.Count == 0) throw TzsError.NotFound("控件代号", term);
            if (paths.Count > 1)
                throw TzsError.Validation("控件代号 \"" + term + "\" 命中 " + paths.Count + " 个元素，请改用 path："
                    + string.Join(" | ", paths.ToArray()));
            return new string[] { paths[0] };
        }

        // ================================================================ describe_kind

        /// <summary>
        /// The legal attribute names per spec-node kind -- `{kind: [attrName, ...]}` for all
        /// seven kinds. This is the authority the manifest's describe_from points at (SPEC
        /// §11.24 (c)) and the list §11.24 (d) rule 2 requires before any spec write.
        ///
        /// Materialised from the live model, never hardcoded: the names come from the nodes
        /// the designer itself built, so a designer change that adds or renames an attribute
        /// cannot silently disagree with a table here. A kind this form happens not to use is
        /// still answerable -- the node is fabricated through the same Spec*Node.Create the
        /// designer calls when it mints one, which is a pure factory (it returns a new node and
        /// registers nothing).
        ///
        /// `kind` optionally narrows the answer to one kind; "layout" is accepted too and
        /// returns the union of the layout attributes actually present on this form's elements
        /// (the legal set for set_layout_attr, which is presence-based per element and therefore
        /// only ever a per-element subset of this).
        /// </summary>
        public static object DescribeKind(Session s, JObject a) {
            string want = Arg(a, "kind");
            if (want != null && want.StartsWith("spec:", StringComparison.Ordinal)) want = want.Substring(5);

            var res = new JObject();
            if (want == null) {
                // All seven kinds, and only those: this is the shape the manifest's
                // describe_from names, and an extra key would be a second answer to the same
                // question. `kind:"layout"` is how the layout side is asked for explicitly.
                foreach (string kind in SpecSlots.Kinds) res[kind] = StrArray(KindAttrs(s, kind));
            } else if (want == "layout") {
                // One record per attribute rather than a bare name list: the facts the caller needs
                // to write a value it has never written before are exactly the ones this class of
                // answer was missing (2026-09-25 eval, F13). The two failures it measured were
                // `hidden` (ENUM false|true) chosen over `invisible` (BOOLEAN, whose 4.2 name is
                // isPassword) and the value's spelling -- and a wrong guess in either does not
                // error, it does the other thing.
                //
                // Both halves come from tables that already existed: the name union from the loaded
                // form, the shape from the same mta/mod-fd.spec table SpecValues uses to REJECT a
                // bad value. Reading it forwards is the difference between "the engine knows but
                // does not say" and what the eval had to guess at.
                string ws = Workspace(s);
                var attrs = new JArray();
                foreach (string name in LayoutAttrUnion(s)) {
                    var o = new JObject();
                    o["name"] = name;
                    SpecValues.Declared d = SpecValues.Describe(ws, name);
                    if (d != null) {
                        if (d.Type != null) o["type"] = d.Type;
                        if (d.Values != null) o["values"] = StrArray(d.Values);
                        if (d.Initial != null) o["initial"] = d.Initial;
                    }
                    attrs.Add(o);
                }
                res["count"] = attrs.Count;
                res["layout"] = attrs;
                // Two things a reader could otherwise get wrong: `values` is the engine's accepted
                // set (absent = nothing declares one), and this list is the union over the whole
                // form -- any ONE element allows a subset of it.
                res["note"] = "values = 引擎会接受的值集（没这个键 = 工作区的 mta/mod-fd.spec 没声明取值集）；"
                            + "这是一张表单的并集，某个元素具体能用哪些比它窄 —— 写错时 E_ATTR_NOT_WHITELIST / "
                            + "E_ATTR_VALUE_ILLEGAL 的 detail.legal 给的是该元素那一份";
                if (ws != null) res["source"] = Path.Combine(Path.Combine(ws, "mta"), "mod-fd.spec");
            } else {
                if (Array.IndexOf(SpecSlots.Kinds, want) < 0)
                    throw TzsError.Validation("未知 kind：" + want + "；合法值: "
                        + string.Join(",", SpecSlots.Kinds) + ",layout");
                res[want] = StrArray(KindAttrs(s, want));
            }
            return res;
        }

        /// <summary>The slot property each kind lives in, on SpecificationInfo (not on
        /// FormSpecModel): the Other* collections and Actions are the whole-form lists, and
        /// both live and tombstoned nodes are in them by design (SPEC §6.3).</summary>
        static readonly Dictionary<string, string> KindCollection = new Dictionary<string, string> {
            { "field",   "OtherFields"      },
            { "hfield",  "OtherHelpCode"    },
            { "pfield",  "OtherRrog"        },
            { "rfield",  "OtherRef"         },
            { "mlfield", "OtherMultiLangs"  },
            { "tree",    "OtherTrees"       },
            { "act",     "Actions"          },
        };

        /// <summary>The Spec*Node type the kind is built by, for the fabricate fallback.</summary>
        static readonly Dictionary<string, string> KindNodeType = new Dictionary<string, string> {
            { "field",   "SpecFieldNode"      },
            { "hfield",  "SpecHelpCodeNode"   },
            { "pfield",  "SpecProgRelNode"    },
            { "rfield",  "SpecReferenceNode"  },
            { "mlfield", "SpecMultiLangNode"  },
            { "tree",    "SpecTreeNode"       },
            { "act",     "SpecActionNode"     },
        };

        internal static IEnumerable NodesOfKind(Session s, string kind) {
            string prop;
            if (!KindCollection.TryGetValue(kind, out prop)) return new object[0];
            return Reflect.Prop(s.Si, prop) as IEnumerable;
        }

        /// <summary>
        /// Every node of one kind the session actually holds, from BOTH places they live.
        ///
        /// The Other* collections are the whole-form lists, but a load drains them: FindFieldSpecById
        /// moves each node it matches into its FormSpecModel (SpecificationInfo.cs:1314 does
        /// `this.fields.Remove(...)`), so after a load `OtherFields` can be empty while the fields
        /// are all in FormSpeDictionary. And it is the FormSpeDictionary node that set_spec_attr
        /// addresses (it resolves the element's name through FindNodeByName), so a describe_kind
        /// built only from Other* would answer from the *fabricated* canonical set -- 20 field
        /// attributes -- while the nodes a caller can actually write carry 23. Reading both is what
        /// keeps describe_kind and the write path from disagreeing.
        ///
        /// Duplicates are harmless here (the callers union attribute names / dedupe by name), and
        /// not deduping by node identity avoids depending on AbstractSpecNode.Equals.
        /// </summary>
        internal static List<object> LiveNodes(Session s, string kind) {
            // Guard rather than NRE. The manifest makes a handle mandatory for describe_kind
            // because the answer is per-file; if a future entry forgets, this says why instead
            // of surfacing a NullReferenceException from SpecDic.
            if (s == null)
                throw TzsError.Validation(
                    "describe_kind 需要 handle：属性集随文件而变（同一个 kind 在不同 .tsd 上属性数不同）");
            var list = new List<object>();
            string slot;
            bool hasSlot = SpecSlots.ByKind.TryGetValue(kind, out slot);
            IDictionary dic = SpecDic(s);
            if (hasSlot && dic != null) {
                foreach (object key in dic.Keys) {
                    object model = dic[key];
                    if (model == null) continue;
                    object node = Reflect.Prop(model, slot);
                    if (node != null) list.Add(node);
                }
            }
            IEnumerable col = NodesOfKind(s, kind);
            if (col != null) foreach (object node in col) if (node != null) list.Add(node);
            return list;
        }

        internal static List<string> KindAttrs(Session s, string kind) {
            var names = new List<string>();
            var seen = new HashSet<string>();
            foreach (object node in LiveNodes(s, kind)) {
                XElement src = Reflect.Prop(node, "Source") as XElement;
                foreach (string an in AttrNames(src)) if (seen.Add(an)) names.Add(an);
            }
            if (names.Count > 0) return names;

            object minted = Mint(s, kind);
            foreach (string an in AttrNames(Reflect.Prop(minted, "Source") as XElement))
                if (seen.Add(an)) names.Add(an);
            return names;
        }

        /// <summary>Spec*Node.Create(info, name) -- the designer's own factory, called without
        /// touching any of its collections. Returns null if the kind has no factory.</summary>
        internal static object Mint(Session s, string kind) {
            string tn;
            if (!KindNodeType.TryGetValue(kind, out tn)) return null;
            Type t = Reflect.Find(Designer.A, tn);
            if (t == null) return null;
            try { return Reflect.Call(t, "Create", s.Si, "cli_probe_kind"); } catch { return null; }
        }

        /// <summary>Union of every attribute name on every element of the form, deduplicated in
        /// document order. Advisory only: the per-element set is what the indexer enforces.</summary>
        internal static IEnumerable<string> LayoutAttrUnion(Session s) {
            var paths = new List<string>();
            var els = new List<object>();
            object formNode = FormNode(s);
            Session.CollectAll(formNode, "x", paths, els);
            var names = new List<string>();
            var seen = new HashSet<string>();
            foreach (object el in els) {
                var attrs = Reflect.Prop(el, "Attributes") as IEnumerable;
                if (attrs == null) continue;
                foreach (object n in attrs) {
                    string k = n.ToString();
                    if (seen.Add(k)) names.Add(k);
                }
            }
            names.Sort(StringComparer.Ordinal);
            return names;
        }

        // ================================================================ list_spec_nodes

        /// <summary>
        /// Every spec node of the open form, by kind, with its status letter. The counterpart of
        /// get_component for the .tsd side: it is how a caller finds what exists without
        /// walking the layout tree, and how the `d` tombstones SPEC §6.3 says accumulate
        /// become visible instead of merely absent.
        /// </summary>
        public static object ListSpecNodes(Session s, JObject a) {
            string only = Arg(a, "kind");
            // `query` is the name the manifest uses; `name` is the alias the body used first (and the
            // one a direct library caller may still pass). Declaring it took until 2026-09-25 --
            // before that the filter was unreachable from the wire, on the verb with the largest
            // answer in the set (SPEC §11.9 item 10).
            string filter = Arg(a, "query");
            if (filter == null) filter = Arg(a, "name");
            if (only != null && Array.IndexOf(SpecSlots.Kinds, only) < 0)
                throw TzsError.Validation("未知 kind：" + only + "；合法值: " + string.Join(",", SpecSlots.Kinds));

            var nodes = new JArray();
            var seen = new HashSet<string>();
            foreach (string kind in SpecSlots.Kinds) {
                if (only != null && only != kind) continue;
                foreach (object node in LiveNodes(s, kind)) {
                    string n = Str(Reflect.Prop(node, "Name"));
                    if (filter != null && (n == null || n.IndexOf(filter, StringComparison.OrdinalIgnoreCase) < 0)) continue;
                    string st = Json.StatusLetter(node);
                    // A node is reachable from both collections at once (the FormSpecModel slot and
                    // the whole-form list); one fact should not be reported twice.
                    if (!seen.Add(kind + "|" + n + "|" + (st ?? ""))) continue;
                    var o = new JObject();
                    o["kind"] = kind;
                    o["name"] = n;
                    if (st != null) o["status"] = st;
                    nodes.Add(o);
                }
            }
            var res = new JObject();
            res["program"] = Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            res["count"] = nodes.Count;
            res["nodes"] = nodes;
            return res;
        }

        // ================================================================ list_tables / list_columns

        static bool _tablesChecked;

        /// <summary>
        /// The data dictionary is a process-wide static pair of caches on TableColumnHelper, and
        /// SettingManager.LoadCommonData already called
        /// TableColumnHelper.Parse(workspace + "\mta\tables.xml", workspace) during Boot --
        /// so normally this is a no-op. It only re-runs Parse for the case where LoadCommonData
        /// skipped the file (it tolerates a missing entry), because GetTables() returning null
        /// is reported to the caller as "the workspace has no data dictionary", and that would
        /// be a lie if the file is right there.
        /// </summary>
        internal static void EnsureTables(Session s) {
            if (_tablesChecked) return;
            _tablesChecked = true;
            if (Tables(s) != null) return;
            string ws = Workspace(s);
            if (ws == null) return;
            string f = Path.Combine(Path.Combine(ws, "mta"), "tables.xml");
            if (!File.Exists(f)) return;
            Reflect.Call(Reflect.Find(Designer.A, "TableColumnHelper"), "Parse", f, ws);
        }

        internal static string Workspace(Session s) {
            object cur = Reflect.Prop(Designer.SettingManager, "CurrentSetting");
            object conn = Reflect.Prop(cur, "Connection");
            return Str(Reflect.Prop(conn, "Workspace"));
        }

        internal static IEnumerable Tables(Session s) {
            return Reflect.Call(Reflect.Find(Designer.A, "TableColumnHelper"), "GetTables") as IEnumerable;
        }

        // ================================================================ list_packages

        /// <summary>The two extensions the designer's form side owns: .tzs (表单) and .tzv
        /// (view). NOT .tzc/.tzf/.tzx -- those belong to the code-package line (tt-dev-tzc),
        /// and this verb's whole job is "which form package do I open".</summary>
        static readonly string[] PackageExts = { "*.tzs", "*.tzv" };

        /// <summary>
        /// The form packages under the workspace, by name.
        ///
        /// WHY IT EXISTS (SPEC §11.9's Go-side list; found by the 2026-09-25 eval, F23 --
        /// three of four executors hit it). "Which file is aapt300?" had no answer anywhere:
        /// the docs could only say "ls the workspace yourself", which in a real workspace is
        /// 29.9 KB of noise with .tzc/.bak variants mixed in, and every executor hand-filtered
        /// it. The designer's own rule (TzpManager.InCurrentWorkspace) is a *directory prefix*
        /// test, so anything under the workspace counts, nested module directories included --
        /// this walks the same way.
        ///
        /// IT DELIBERATELY DOES NOT READ THE PACKAGES. `program` is derived from the file NAME
        /// (the `<program>(c).tzs` convention), so it is a hint for picking a file; `open`
        /// remains the authority on what a package actually contains. Reading each package's
        /// .tsd to answer that would be 50-200 zip reads per call -- a different, much more
        /// expensive verb, and one whose answer goes stale the moment someone renames a file.
        ///
        /// No session: it answers from the filesystem, like list_tables answers from the data
        /// dictionary (the dispatch passes a null Session for a NeedsHandle=false verb).
        /// </summary>
        public static object ListPackages(Session s, JObject a) {
            string ws = Workspace(s);
            if (string.IsNullOrEmpty(ws))
                throw TzsError.Validation("list_packages 按工作区找包，而工作区是空的："
                    + "检查配置里的 tzs.workspace（它没有缺省，见 SKILL §1）");
            string filter = Arg(a, "query");
            if (filter == null) filter = Arg(a, "filter");   // 别名，与 list_tables / list_columns 同一条规矩
            int limit = IntArg(a, "limit", 200);

            var paths = new List<string>();
            try {
                foreach (string ext in PackageExts)
                    paths.AddRange(Directory.GetFiles(ws, ext, SearchOption.AllDirectories));
            } catch (Exception ex) {
                // 读不动就是读不动：说清是哪个目录、为什么，别让它变成 E_INTERNAL（"上报，别重试"）
                // —— 这对调用方是"环境不对"，退 2 才对得上。
                throw TzsError.Validation("扫工作区失败（" + ws + "）：" + ex.Message);
            }
            paths.Sort(StringComparer.OrdinalIgnoreCase);

            var arr = new JArray();
            int total = 0;
            foreach (string p in paths) {
                string name = Path.GetFileName(p);
                if (filter != null && name.IndexOf(filter, StringComparison.OrdinalIgnoreCase) < 0) continue;
                total++;
                if (limit > 0 && arr.Count >= limit) continue;
                var o = new JObject();
                o["name"] = name;
                o["path"] = p;
                string prog = ProgramFromName(name);
                if (prog != null) o["program"] = prog;
                try {
                    var fi = new FileInfo(p);
                    o["sizeBytes"] = fi.Length;
                    o["modified"] = fi.LastWriteTime.ToString("yyyy-MM-dd HH:mm:ss");
                } catch {
                    // 单条 stat 失败不影响整份清单：那几个字段缺席就是"没读到"。
                }
                arr.Add(o);
            }

            var res = new JObject();
            res["workspace"] = ws;
            res["count"] = total;
            res["returned"] = arr.Count;
            res["truncated"] = limit > 0 && arr.Count < total;
            res["packages"] = arr;
            res["note"] = "program 是从**文件名**推的（<程序名>(c).tzs 那条约定），不是打开包读的 —— "
                        + "权威答案用 open（它回的 program 是设计器自己算的）；这里只用来挑文件";
            return res;
        }

        /// <summary>`aapt300(c).tzs` → `aapt300`；`aapt300_wf(s).tzv` → `aapt300_wf`。
        /// 只认"最后一段括号"，没有就当文件名就是程序名。见 ListPackages 的说明：这是提示，
        /// 不是权威。</summary>
        static string ProgramFromName(string name) {
            if (string.IsNullOrEmpty(name)) return null;
            string stem = Path.GetFileNameWithoutExtension(name);
            int i = stem.LastIndexOf('(');
            if (i > 0 && stem.EndsWith(")")) stem = stem.Substring(0, i);
            return stem.Length == 0 ? null : stem;
        }

        /// <summary>
        /// The data dictionary's tables, optionally filtered by a name substring. This is the
        /// authority for "which tables may a field bind to" -- and the reason it is a function
        /// at all is that add_field's blocking preconditions (widget mapping, the <tbl> file
        /// existing) are both answerable from here (SPEC §11.15).
        /// </summary>
        public static object ListTables(Session s, JObject a) {
            // `query` is the name the manifest uses; `filter` is accepted as an alias so a
            // direct library caller can use either.
            string filter = Arg(a, "query");
            if (filter == null) filter = Arg(a, "filter");
            int limit = IntArg(a, "limit", 200);
            EnsureTables(s);

            var arr = new JArray();
            int total = 0;
            IEnumerable tables = Tables(s);
            if (tables != null) {
                foreach (object t in tables) {
                    XElement te = t as XElement;
                    if (te == null) continue;
                    string n = Reflect.Attr(te, "name");
                    if (filter != null && (n == null || n.IndexOf(filter, StringComparison.OrdinalIgnoreCase) < 0)) continue;
                    total++;
                    if (limit > 0 && arr.Count >= limit) continue;
                    arr.Add(XAttrs(te));
                }
            }
            var res = new JObject();
            res["program"] = Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            res["count"] = total;
            res["returned"] = arr.Count;
            res["truncated"] = limit > 0 && arr.Count < total;
            res["tables"] = arr;
            return res;
        }

        /// <summary>
        /// The columns of one table, from `<workspace>/<module>/tbl/<table>.tbl`. Each column
        /// carries its own attributes plus the `col_attr` entry when one exists -- that entry
        /// is what decides the widget a field is created with and its default width, so it is
        /// the missing piece add_field was blocked on.
        ///
        /// `column` narrows to one column (via TableColumnHelper.GetColumnInfo /
        /// GetColumnAttrInfo, the designer's own accessors).
        ///
        /// THE TWO CACHE TRAPS, deliberately handled: TableColumnHelper.GetColumnText and
        /// .IsPK read only the `columns` cache and return null/false -- silently, with no
        /// indication anything is missing -- unless FindTableColumns(table) has been called for
        /// that table first, because that is the method that parses the .tbl and fills the
        /// cache. So FindTableColumns is always called before either of them is used, and its
        /// result (not the cache) is what the enumeration walks.
        /// </summary>
        public static object ListColumns(Session s, JObject a) {
            string table = Need(a, "table");
            string oneCol = Arg(a, "column");          // exact name
            string filter = Arg(a, "query");           // substring
            if (filter == null) filter = Arg(a, "filter");
            EnsureTables(s);

            Type tch = Reflect.Find(Designer.A, "TableColumnHelper");
            XElement cols = Reflect.Call(tch, "FindTableColumns", table) as XElement;
            if (cols == null) throw TzsError.NotFound("表", table + Suggest(s, table));

            var arr = new JArray();
            var colNames = new List<string>();
            foreach (XElement c in cols.Elements("column")) {
                string cn = Reflect.Attr(c, "name");
                if (cn != null) colNames.Add(cn);
                if (oneCol != null && cn != oneCol) continue;
                if (filter != null && (cn == null || cn.IndexOf(filter, StringComparison.OrdinalIgnoreCase) < 0)) continue;
                arr.Add(ColumnJson(tch, table, c));
            }
            // An exact column name is a location, so not finding it is an error; a substring is
            // a query, so an empty result is a valid answer (same split as find_component).
            if (oneCol != null && arr.Count == 0) {
                List<string> near = Similar(colNames, oneCol);
                throw TzsError.NotFound("列", table + "." + oneCol + (near.Count == 0
                    ? "；" + table + " 里没有名字相近的列"
                    : "；近似候选: " + string.Join(", ", near.ToArray())));
            }

            var res = new JObject();
            res["program"] = Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            res["table"] = table;
            res["module"] = Reflect.Attr(cols, "module");
            res["count"] = arr.Count;
            res["columns"] = arr;
            return res;
        }

        static JObject ColumnJson(Type tch, string table, XElement c) {
            string cn = Reflect.Attr(c, "name");
            var o = new JObject();
            o["name"] = cn;
            o["info"] = XAttrs(c);
            // GetColumnInfo(table, column) -- the designer's own accessor, used here so the
            // output cannot diverge from what the designer would read.
            XElement info = Reflect.Call(tch, "GetColumnInfo", table, cn) as XElement;
            if (info != null && info != c) o["infoResolved"] = XAttrs(info);

            XElement ca = Reflect.Call(tch, "GetColumnAttrInfo", table, cn) as XElement;
            o["colAttr"] = ca == null ? (JToken)JValue.CreateNull() : XAttrs(ca);
            // Only safe because FindTableColumns(table) ran first -- see the trap note above.
            o["isPK"] = (bool)Reflect.Call(tch, "IsPK", table, cn);
            o["text"] = Reflect.Call(tch, "GetColumnText", table, cn) as string;
            return o;
        }

        /// <summary>
        /// Name candidates for a "did you mean" list: the longest matching suffix of `want`,
        /// widening from 6 characters down to 3 and stopping at the first width that finds
        /// anything. Suffix-first because these names are prefixed by table ({table}{seq}) and
        /// suffixed by meaning (…docno, …ent, …007), so the tail is the distinctive part.
        /// Eight at most: a not_found that dumps 3886 table names helps nobody.
        /// </summary>
        static List<string> Similar(IEnumerable<string> names, string want) {
            var hits = new List<string>();
            if (want == null) return hits;
            for (int len = Math.Min(6, want.Length); len >= 3 && hits.Count == 0; len--) {
                string frag = want.Substring(want.Length - len);
                foreach (string n in names) {
                    if (n == null) continue;
                    if (n.IndexOf(frag, StringComparison.OrdinalIgnoreCase) >= 0) hits.Add(n);
                    if (hits.Count >= 8) break;
                }
            }
            return hits;
        }

        /// <summary>A short "你是不是要找…" for a table name that does not resolve, or an explicit
        /// statement that nothing similar was found -- a not_found that silently omits the
        /// candidates is indistinguishable from one that never looked.</summary>
        static string Suggest(Session s, string table) {
            IEnumerable tables = Tables(s);
            if (tables == null)
                return "（工作区数据字典为空：LoadCommonData 没读到 mta/tables.xml）";
            var names = new List<string>();
            foreach (object t in tables) {
                XElement te = t as XElement;
                if (te == null) continue;
                names.Add(Reflect.Attr(te, "name"));
            }
            List<string> hits = Similar(names, table);
            return hits.Count == 0
                ? "；数据字典里没有名字相近的表"
                : "；近似候选: " + string.Join(", ", hits.ToArray());
        }

        // ================================================================ list_records

        /// <summary>
        /// The &lt;Record&gt;/&lt;RecordField&gt; section -- the .4fd side of the field table.
        /// Note GetRecords() rebuilds that section from the live model first (it calls
        /// SaveToForm internally), so this is a read that refreshes the model's own FormElement
        /// snapshot; nothing on disk is touched.
        ///
        /// fieldIdRef is reported but is NOT an identifier: the designer renumbers every
        /// fieldId/fieldIdRef on every save (SPEC §6.5), which is why addressing a field means
        /// addressing it by name and never by id.
        /// </summary>
        public static object ListRecords(Session s, JObject a) {
            var records = new JArray();
            IEnumerable recs = Reflect.Call(s.Si, "GetRecords") as IEnumerable;
            if (recs != null) {
                foreach (object r in recs) {
                    XElement re = r as XElement;
                    if (re == null) continue;
                    var o = new JObject();
                    o["attrs"] = XAttrs(re);
                    var fields = new JArray();
                    foreach (XElement f in re.Elements()) {
                        var fo = new JObject();
                        fo["name"] = Reflect.Attr(f, "name");
                        fo["attrs"] = XAttrs(f);
                        fields.Add(fo);
                    }
                    o["fields"] = fields;
                    o["fieldCount"] = fields.Count;
                    records.Add(o);
                }
            }
            var res = new JObject();
            res["program"] = Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            res["count"] = records.Count;
            res["records"] = records;
            return res;
        }

        // ================================================================ list_local_strings

        /// <summary>
        /// The &lt;sfield&gt; local strings (name/text/lstr) and the action local strings. These
        /// are the lbl_/cmt_ texts the designer's Transform* steps pair with a component, and
        /// the reason they need their own reader is that they are in neither the layout tree
        /// nor FormSpeDictionary.
        ///
        /// Tombstones are included and carry their status, for the same reason list_spec_nodes
        /// reports them: SPEC §6.3 says deletion never removes a node, so "absent" and
        /// "deleted" are different facts and only one of them is visible from the layout.
        /// </summary>
        public static object ListLocalStrings(Session s, JObject a) {
            // `query` is the name the manifest uses; `filter` is accepted as an alias so a
            // direct library caller can use either. Same pair as ListTables / ListColumns below --
            // this one read ONLY `filter` until 2026-09-25, and since `filter` was declared nowhere
            // the filter was unreachable from the wire (SPEC §11.9 item 10).
            string filter = Arg(a, "query");
            if (filter == null) filter = Arg(a, "filter");
            string path = Arg(a, "path");

            // `path` narrows the answer to the strings one subtree actually refers to. The
            // reference is by name, held in the element's own attributes: TransformReferenceNode
            // writes title="lbl_<rtn>" / comment="cmt_<rtn>" and TransformProgRelNode /
            // TransformFormToSpecField write text="lbl_...", so those attributes are the link
            // between an <sfield> and the component that displays it. There is no other link --
            // the strings are in neither the layout tree nor FormSpeDictionary.
            HashSet<string> wanted = null;
            if (path != null) {
                object el = s.FindByPath(path);
                if (el == null) throw TzsError.PathNotFound(path);
                wanted = new HashSet<string>();
                CollectLocalNames(el, wanted);
            }

            var arr = new JArray();
            IEnumerable fs = Reflect.Prop(s.Si, "FieldStrings") as IEnumerable;
            if (fs != null) {
                foreach (object n in fs) {
                    string name = Str(Reflect.Prop(n, "Name"));
                    if (!Wanted(wanted, name, filter)) continue;
                    var o = new JObject();
                    o["name"] = name;
                    o["text"] = Str(Reflect.Prop(n, "Text"));
                    string st = Json.StatusLetter(n);
                    if (st != null) o["status"] = st;
                    arr.Add(o);
                }
            }

            var acts = new JArray();
            IEnumerable asv = Reflect.Prop(s.Si, "ActionStrings") as IEnumerable;
            if (asv != null) {
                foreach (object n in asv) {
                    string name = Str(Reflect.Prop(n, "Name"));
                    if (!Wanted(wanted, name, filter)) continue;
                    var o = new JObject();
                    o["name"] = name;
                    o["text"] = Str(Reflect.Prop(n, "Text"));
                    string st = Json.StatusLetter(n);
                    if (st != null) o["status"] = st;
                    acts.Add(o);
                }
            }

            var res = new JObject();
            res["program"] = Str(Reflect.Prop(s == null ? null : s.Tzp, "ProgramName"));
            if (wanted != null) res["path"] = path;
            res["count"] = arr.Count;
            res["strings"] = arr;
            res["actionCount"] = acts.Count;
            res["actionStrings"] = acts;
            return res;
        }

        static bool Wanted(HashSet<string> wanted, string name, string filter) {
            if (name == null) return false;
            if (wanted != null && !wanted.Contains(name)) return false;
            if (filter != null && name.IndexOf(filter, StringComparison.OrdinalIgnoreCase) < 0) return false;
            return true;
        }

        static readonly string[] LocalStringAttrs = { "title", "text", "comment", "LocalString" };

        static void CollectLocalNames(object el, HashSet<string> into) {
            foreach (string k in LocalStringAttrs) {
                string v = (string)Session.Raw(el, k);
                if (!string.IsNullOrEmpty(v)) into.Add(v);
            }
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes == null) return;
            foreach (var c in nodes) CollectLocalNames(c, into);
        }
    }
}
