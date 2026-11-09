using System;
using System.Collections;
using System.Collections.Generic;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;

namespace TzsCli.Designer
{
    // Deliberately NOT TzsCli.Designer.Fns, for the same reason Read.cs and Attr.cs are not:
    // Fns/Session.cs declares a static module class named Session, and a type in the same
    // namespace outranks the enclosing one, so inside .Fns every `Session s` parameter would be
    // CS0721. Rpc.BuildMap finds a module by simple type name (Rpc.cs FindModule), so the
    // namespace costs nothing.
    //
    // `Action` versus System.Action: this class shadows the delegate inside this namespace, and
    // the two places that name the delegate (Session.cs:151 `Action<string>`, Fns/Validate.cs
    // `Action<T>`) still bind to System.Action -- C# name lookup matches on arity, and this type
    // has none. Verified by building: TzsCli.Designer.dll compiles with this file in place.
    // If that ever stops being true, the fix is a child namespace, not an edit to Session.cs.

    /// <summary>
    /// The semantic / Action group: the three 用途 the designer's right-click menu offers, plus
    /// the two things the 项目 (Action) panel does -- add and delete an &lt;act&gt;, and tick its
    /// 型态 boxes.
    ///
    /// WHY THE ELEMENT IS NEVER BUILT HERE
    ///
    /// insert_semantic calls ComponentFactory.CreateEmptyComponentForBody(key, specType) and lets
    /// it decide the attributes, the same way ManagedForm's three ExecutedInsert* handlers do.
    /// That factory is the only place the combination lives:
    ///
    ///     REFERENCE  Edit       style="reference" sizePolicy="fixed" gridWidth="10" noEntry="true"
    ///     MULTILANG  ButtonEdit image="16/langmodify.png" action="update_item"
    ///     PROGREL    FFLabel    tag="sync" style="textFormat_html" sizePolicy="fixed" gridWidth="10"
    ///
    /// Re-typing that table here would make it a second, silently drifting copy -- and the whole
    /// reason the REPLACE-able path is a factory method rather than four SetAttribute calls in the
    /// handler is that this table is expected to change with the designer version. The result
    /// reports the element's attribute map and the factory's own output side by side so a caller
    /// can see the two are identical.
    ///
    /// WHY THE PARENT CONTAINER IS CHECKED BEFORE INSERTING
    ///
    /// FormSpecModel.GetNodeType is what decides which spec node a new element gets, and for two of
    /// the three 用途 it is conditional on the parent:
    ///
    ///   REFERENCE  needs Edit + style="reference" AND Parent.Type in {Table, Tree}
    ///   PROGREL    needs FFLabel + tag="sync"   AND Parent.Type in {Table, Tree}
    ///   MULTILANG  needs ButtonEdit + image="16/langmodify.png" -- unconditional
    ///
    /// so inserting a 参考栏位 into a Grid produces an ordinary FIELD (or a FORMONLY) element
    /// instead, with no &lt;rfield&gt; anywhere. That would be a silent wrong answer, so the parent
    /// is checked up front.
    ///
    /// WHY THE MINTED SPEC NODE IS PROMOTED OUT OF CREATE
    ///
    /// AbstractSpecNode.ToXml() returns null for a CREATE-status node ("c" means "in memory only,
    /// not yet in the file"), and SaveToTSD drops those. The node a fresh element gets from
    /// SpecificationInfo.Add is exactly such a node, so without promotion the &lt;rfield&gt; /
    /// &lt;mlfield&gt; / &lt;pfield&gt; would never reach the .tsd and `save` would appear to have
    /// inserted nothing. test/Edit.cs's `add` op promotes for the same reason and is the recipe
    /// the corpus baselines were measured against.
    ///
    /// ONE THING MANIFEST.CS GETS WRONG, AND WHY THIS FILE CANNOT FIX IT
    ///
    /// Both `types` (set_action_types) and `type` (add_action) are declared there as
    /// `P.Enum(..., new[] { "all", "none" })`, and Manifest.Check refuses any other value before a
    /// body runs. The real 型态 vocabulary is all / mi / di&lt;n&gt; / db&lt;n&gt;, which is not a
    /// static enum -- &lt;n&gt; comes from the form's own s_detail records (TypeVocabulary below) --
    /// so the two declarations block the very tokens the rules are about. `mi`/`di1`/`db1` are
    /// still parsed and accepted by the bodies here (a direct caller gets them), but through the
    /// JSON surface only all|none is reachable until those two entries are widened to a string
    /// list. The code does not paper over it: the parameter set is exactly what Manifest declares.
    /// </summary>
    public static class Action
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["insert_semantic"]  = InsertSemantic;
            into["add_action"]       = AddAction;
            into["delete_action"]    = DeleteAction;
            into["set_action_types"] = SetActionTypes;
        }

        // ================================================================ errors

        /// <summary>
        /// A refusal by the designer's own rules. `detail` carries the machine-readable half,
        /// which SPEC §11.24 (a) makes load-bearing: the 参考栏位 parent gate is recoverable if
        /// the caller is told which container was asked to hold what.
        /// </summary>
        static DetailedError Refuse(string message, JObject detail) {
            return new DetailedError("E_DESIGNER", message, detail == null ? new JObject() : detail);
        }

        /// <summary>
        /// The resource string FormatException? No -- the designer's own message for an action that
        /// has no 型态 to add to or remove from (SpecActionNode.AddActionType / RemoveActionType,
        /// SpecActionNode.cs:281/:318). Bootstrap merged langs/zh-cn.xaml into the application
        /// dictionary, so FindResource works headless; if it ever does not, the code name is kept
        /// in the message so the failure stays greppable.
        /// </summary>
        static string ActionHasTypeMessage() {
            string s = null;
            try {
                object app = Application.Current;
                if (app != null) s = Application.Current.FindResource("Message_ActionHasType") as string;
            } catch { }
            if (string.IsNullOrEmpty(s))
                s = "这个 Action 没有型态字段（type），设计器不允许给它设置型态";
            return "Message_ActionHasType: " + s;
        }

        // ================================================================ act plumbing

        /// <summary>
        /// SpecificationInfo._acts. The whole-form action list, tombstones included, which is why
        /// it is read directly instead of through the Actions / ActionsForView views: those filter
        /// DELETE-status nodes and toolbar actions out, and a caller asking to delete an id that is
        /// already a tombstone should be told so rather than told it does not exist.
        /// </summary>
        static IList Acts(Session s) { return Reflect.Prop(s.Si, "_acts") as IList; }

        /// <summary>Acts by id, in list order, tombstones included.</summary>
        static List<object> ActsById(Session s, string id) {
            var hits = new List<object>();
            IList all = Acts(s);
            if (all == null) return hits;
            foreach (object a in all) {
                if (a == null) continue;
                if (Read.Str(Reflect.Prop(a, "Name")) == id) hits.Add(a);
            }
            return hits;
        }

        /// <summary>
        /// The act named `id`, or the caller's version of the error. Several matches is possible --
        /// a tombstoned act and a freshly minted one can share a name -- and the live one wins,
        /// which is the one the caller can still change.
        /// </summary>
        static object FindAct(Session s, string id) {
            List<object> hits = ActsById(s, id);
            if (hits.Count == 0) {
                var d = new JObject();
                d["id"] = id;
                var cand = new List<string>();
                IList all = Acts(s);
                if (all != null) foreach (object a in all) {
                    string n = Read.Str(Reflect.Prop(a, "Name"));
                    if (n == null) continue;
                    if (n.IndexOf(id, StringComparison.OrdinalIgnoreCase) >= 0) cand.Add(n);
                }
                if (cand.Count > 0) d["candidates"] = Read.StrArray(cand);
                throw new DetailedError("not_found", "action 不存在: " + id, d);
            }
            foreach (object a in hits)
                if (StatusLetter(a) != "d") return a;
            return hits[0];
        }

        static string StatusLetter(object node) { return Json.StatusLetter(node); }

        /// <summary>SpecStatus as the enum's own name ("CREATE"), which is what a caller compares
        /// against when it wants to know whether a node is only in memory.</summary>
        static string StatusOf(object node) { return Read.Str(Reflect.Prop(node, "Status")); }

        static bool IsCreate(object node) {
            string st = StatusOf(node);
            return st != null && st.IndexOf("CREATE", StringComparison.Ordinal) >= 0;
        }

        static bool IsDelete(object node) {
            string st = StatusOf(node);
            return st != null && st.IndexOf("DELETE", StringComparison.Ordinal) >= 0;
        }

        static void SetModify(object node) {
            Reflect.SetProp(node, "Status", Enum.Parse(Reflect.Find(Designer.A, "SpecStatus"), "MODIFY"));
        }

        /// <summary>
        /// The action's 型态 attribute ("all", "all,mi", "di1", ...). Read through the node's own
        /// ActionTypes property -- the same accessor AddActionType/RemoveActionType write -- so the
        /// two cannot disagree about which attribute this is.
        /// </summary>
        static string ActionTypesOf(object node) {
            return Read.Str(Reflect.Prop(node, "ActionTypes"));
        }

        static List<string> SplitTypes(string s) {
            var list = new List<string>();
            if (string.IsNullOrEmpty(s)) return list;
            foreach (string t in s.Split(',')) {
                string x = t.Trim();
                if (x.Length > 0 && !list.Contains(x)) list.Add(x);
            }
            return list;
        }

        // ================================================================ the 型态 vocabulary

        /// <summary>
        /// The legal 型态 tokens, read from the form rather than from a constant list -- SPEC's
        /// "n 来自表单自己的 s_detail&lt;n&gt; 记录".
        ///
        /// Two designer call sites define this set and they do NOT read the same thing, so both are
        /// unioned rather than one being picked:
        ///
        ///   ActionTypeDataGrid.RenderTypesToDataGrid (:96) builds all / mi plus di&lt;n&gt; / db&lt;n&gt;
        ///     from the &lt;sr name="s_detail&lt;n&gt;"&gt; entries of the table association
        ///     (AssociateTable.Source), skipping status="d" ones.
        ///   ActionDefaults.UpdateGroups (:52) builds d[ib]&lt;n&gt; from the FormSpeDictionary keys
        ///     that start with "s_detail".
        ///
        /// They disagree on a real corpus file: aapp320(c).tzs has an s_detail1 component in its
        /// .4fd but no &lt;table&gt; section at all, so the grid offers only all/mi while the
        /// ActionDefaults panel offers di1/db1. The union is the set of tokens either designer
        /// surface can express; it never invents one.
        /// </summary>
        static List<string> TypeVocabulary(Session s) {
            var v = new List<string>();
            var seen = new HashSet<string>(StringComparer.Ordinal);
            v.Add("all"); seen.Add("all");
            v.Add("mi");  seen.Add("mi");

            XElement table = AssociateSource(s);
            if (table != null) {
                foreach (XElement sr in table.Descendants("sr")) {
                    XAttribute nm = sr.Attribute("name");
                    if (nm == null) continue;
                    XAttribute st = sr.Attribute("status");
                    if (st != null && st.Value == "d") continue;          // SpecStatus.DELETE
                    Regex re = new Regex("^s_detail(?<seq>\\d+)$");
                    Match m = re.Match(nm.Value);
                    if (!m.Success) continue;
                    string seq = m.Groups["seq"].Value;
                    if (seen.Add("di" + seq)) v.Add("di" + seq);
                    if (seen.Add("db" + seq)) v.Add("db" + seq);
                }
            }

            IDictionary dic = Read.SpecDic(s);
            if (dic != null) {
                Regex re2 = new Regex("^s_detail(?<seq>\\d+)$");
                foreach (object key in dic.Keys) {
                    string k = key == null ? null : key.ToString();
                    if (k == null) continue;
                    Match m = re2.Match(k);
                    if (!m.Success) continue;
                    string seq = m.Groups["seq"].Value;
                    if (seen.Add("di" + seq)) v.Add("di" + seq);
                    if (seen.Add("db" + seq)) v.Add("db" + seq);
                }
            }
            return v;
        }

        static XElement AssociateSource(Session s) {
            object assoc = Reflect.Prop(s.Si, "AssociateTable");
            if (assoc == null) return null;
            return Reflect.Prop(assoc, "Source") as XElement;
        }

        /// <summary>Case-insensitive membership, returning the vocabulary's own spelling.</summary>
        static string Canon(List<string> vocab, string token) {
            foreach (string v in vocab)
                if (string.Equals(v, token, StringComparison.OrdinalIgnoreCase)) return v;
            return null;
        }

        /// <summary>
        /// Parses a 型态 request into a target set. `types` may be one token or a comma-separated
        /// list of them; the token "none" means the empty set, which is how a caller asks for "no
        /// 型态" (and, per the designer's own third rule, lands on "all" again).
        ///
        /// Returns the empty list for "none", and never both at once -- "none,all" is refused
        /// rather than resolved, because the two readings of it ("clear" / "all") give different
        /// answers.
        /// </summary>
        static List<string> ParseTypes(Session s, string raw, out List<string> vocab) {
            vocab = TypeVocabulary(s);
            var want = new List<string>();
            bool sawNone = false;
            if (!string.IsNullOrEmpty(raw)) {
                foreach (string tok in raw.Split(',')) {
                    string t = tok.Trim();
                    if (t.Length == 0) continue;
                    if (string.Equals(t, "none", StringComparison.OrdinalIgnoreCase)) { sawNone = true; continue; }
                    string c = Canon(vocab, t);
                    if (c == null) {
                        var d = new JObject();
                        d["types"] = t;
                        d["legal"] = Read.StrArray(vocab);
                        throw new DetailedError("E_BAD_PARAM",
                            "未知的 Action 型态 \"" + t + "\"；这个表单可用的型态: " + string.Join(", ", vocab.ToArray()),
                            d);
                    }
                    if (!want.Contains(c)) want.Add(c);
                }
            }
            if (sawNone && want.Count > 0) {
                var d = new JObject();
                d["types"] = raw;
                d["legal"] = Read.StrArray(vocab);
                throw new DetailedError("E_BAD_PARAM",
                    "\"none\"（清空型态）不能和具体型态一起给；要么只给 none，要么给型态列表", d);
            }
            return want;
        }

        // ================================================================ the two 型态 rules
        //
        // Reproduced from SpecActionNode.AddActionType (:277) and RemoveActionType (:314) by
        // ALWAYS calling those two methods rather than re-implementing the string surgery here.
        // Their four rules are real and are exactly the ones an approximation gets wrong:
        //
        //   AddActionType("all")   -> also drops every "db*" from the set
        //   AddActionType("db<n>") -> also drops "all"
        //   RemoveActionType(x)    -> if the set would become empty, "all" is put back
        //   either, on a node whose 型态 is empty -> throws Message_ActionHasType
        //
        // Membership is `Regex.IsMatch(token, type)` in the designer, so the tokens are regexes;
        // calling through is also what keeps that subtlety out of here.

        static void AddType(object node, string type) { Reflect.Call(node, "AddActionType", type); }
        static void RemoveType(object node, string type) { Reflect.Call(node, "RemoveActionType", type); }

        /// <summary>
        /// Makes the node's 型态 set equal to `target`, and returns the resulting attribute value.
        ///
        /// Removals first, then additions: that is the order the ActionTypeDataGrid checkboxes
        /// produce (uncheck, then check), and it is the order in which the designer's own rules
        /// land where a caller expects. Walking the pre-change set is safe even though a removal
        /// can re-add "all" -- the re-added token is either wanted (then the add loop is a no-op)
        /// or dropped again by the add loop's own rule.
        /// </summary>
        static string ApplyTypes(object node, List<string> target) {
            List<string> before = SplitTypes(ActionTypesOf(node));
            foreach (string t in before) if (!target.Contains(t)) RemoveType(node, t);
            foreach (string t in target) if (!SplitTypes(ActionTypesOf(node)).Contains(t)) AddType(node, t);
            return ActionTypesOf(node);
        }

        /// <summary>
        /// The designer's refusal that guards both 型态 mutators, checked BEFORE calling them so
        /// the caller gets it as a wire error rather than as a NullReferenceException from inside
        /// FindResource (the designer's own throw site formats a resource string).
        ///
        /// Two conditions reach it. An act with no `type` attribute at all is the literal case --
        /// AddActionType/RemoveActionType throw Message_ActionHasType for it. A toolbar action is
        /// the other: SpecificationInfo.ActionsForView refuses to show one (SpecificationInfo.cs
        /// :767 filters on !IsToolBarAction), so the 型态 grid is unreachable for it by design, and
        /// "cannot be given a 型态" is the answer either way.
        /// </summary>
        static void GuardTypable(Session s, object node, string id) {
            if ((bool)Reflect.Prop(node, "IsToolBarAction"))
                throw Refuse(ActionHasTypeMessage() + "：\"" + id + "\" 是这张表单工具栏上的 action"
                    + "（<toolbar items=...> 里有它），设计器的 Action 型态界面根本不显示它", TypableDetail(s, id, node));
            if (string.IsNullOrEmpty(ActionTypesOf(node)))
                throw Refuse(ActionHasTypeMessage() + "：\"" + id + "\" 的 <act> 上没有 type 属性",
                    TypableDetail(s, id, node));
        }

        static JObject TypableDetail(Session s, string id, object node) {
            var d = new JObject();
            d["id"] = id;
            d["actionTypes"] = ActionTypesOf(node);
            d["toolbar"] = (bool)Reflect.Prop(node, "IsToolBarAction");
            d["legal"] = Read.StrArray(TypeVocabulary(s));
            return d;
        }

        // ================================================================ spec-node plumbing

        /// <summary>The spec slot a FormSpecModel actually holds, as (kind, node) -- the first
        /// non-null one in SpecSlots order. Used to report what the designer minted rather than
        /// what was asked for.</summary>
        static string SlotKind(object fsm, out object node) {
            node = null;
            if (fsm == null) return null;
            foreach (string kind in SpecSlots.Kinds) {
                node = Reflect.Prop(fsm, SpecSlots.ByKind[kind]);
                if (node != null) return kind;
            }
            return null;
        }

        /// <summary>
        /// Promotes every CREATE-status spec node on a FormSpecModel to MODIFY, and returns how
        /// many moved. Same sweep as test/Edit.cs's `add` op over the same seven slots, and the
        /// same guard: a slot that is not CREATE is left alone, because MODIFY on an unchanged node
        /// would put a `status="u"` on an entry the caller never touched.
        /// </summary>
        static int Promote(object fsm) {
            if (fsm == null) return 0;
            int n = 0;
            foreach (string kind in SpecSlots.Kinds) {
                object node = Reflect.Prop(fsm, SpecSlots.ByKind[kind]);
                if (node == null || !IsCreate(node)) continue;
                SetModify(node);
                n++;
            }
            return n;
        }

        static JObject Delta(JObject attrs, string path) {
            var d = new JObject();
            d["path"] = path;
            if (attrs != null && attrs.Count > 0) d["attrs"] = attrs;
            return d;
        }

        // ================================================================ insert_semantic

        /// <summary>
        /// The three 用途 of the designer's right-click menu: 新增参考栏位 (REFERENCE),
        /// 新增多语言栏位 (MULTILANG), 新增串查栏位 (PROGREL).
        ///
        /// `path` is the container the new element goes into, and the element is appended to it --
        /// the designer's handlers insert after the *selected* element (xmlElement.Index + 1), and
        /// with only a container to address, "after the last child" is that same rule.
        ///
        /// The two gates before the insert are both the designer's own:
        ///   * progrel -- ExecutedInsertQuery's CanExecute is `"Q" == GetCodeTemplate()`
        ///     (ManagedForm.xaml.cs:716), so a non-Q form cannot have a 串查栏位 at all.
        ///   * reference/progrel -- FormSpecModel.GetNodeType only resolves those two 用途 out of a
        ///     Table or a Tree parent; anywhere else the element would come back as a plain FIELD
        ///     with no &lt;rfield&gt;/&lt;pfield&gt; minted for it.
        /// </summary>
        public static object InsertSemantic(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string use  = Read.Need(a, "use").ToLowerInvariant();

            string specType;
            if (use == "reference") specType = "REFERENCE";
            else if (use == "multilang") specType = "MULTILANG";
            else if (use == "progrel") specType = "PROGREL";
            else throw TzsError.Validation("未知 use=" + use + "；合法值: reference, multilang, progrel");

            Attr.Urm(s);
            object parent = Attr.El(s, path);
            string parentType = Read.Str(Reflect.Prop(parent, "Type"));

            if (use == "progrel") {
                string tpl = CodeTemplate(s);
                if (tpl != "Q") {
                    var d = new JObject();
                    d["use"] = use;
                    d["codeTemplate"] = tpl;
                    d["required"] = "Q";
                    throw Refuse("串查栏位只能加在 code_template=\"Q\" 的表单上（当前是 \""
                        + (tpl == null ? "" : tpl) + "\"）；这是设计器的 CanExecutedInsertQuery 门禁", d);
                }
            }
            if (use != "multilang" && parentType != "Table" && parentType != "Tree") {
                var d = new JObject();
                d["path"] = path;
                d["use"] = use;
                d["parentType"] = parentType;
                d["required"] = new JArray("Table", "Tree");
                throw Refuse(use + " 栏位只能加在 Table / Tree 里（\"" + path + "\" 是 " + parentType
                    + "）：FormSpecModel.GetNodeType 只在 Table/Tree 父节点下把 "
                    + (use == "reference" ? "Edit+style=reference" : "FFLabel+tag=sync")
                    + " 认成 " + specType + "，放在别处会变成一个普通 FIELD", d);
            }

            // The factory, not a hand-built attribute combination. ManagedForm's three handlers
            // (ExecutedInsertReference :698 / ExecutedInsertMultiLanguage :707 /
            // ExecutedInsertQuery :723) all call exactly this, and nothing else.
            Type cft = Reflect.Find(Designer.A, "ComponentFactory");
            if (cft == null) throw new TzsError("internal", "找不到 ComponentFactory");
            Type snt = Reflect.Find(Designer.A, "SpecNodeType");
            if (snt == null) throw new TzsError("internal", "找不到 SpecNodeType");
            object st = Enum.Parse(snt, specType);
            object el = Reflect.Call(cft, "CreateEmptyComponentForBody", s.Key, st);
            if (el == null) throw new TzsError("internal",
                "CreateEmptyComponentForBody 返回了 null (use=" + use + ")");

            string name = Read.Str(Session.Raw(el, "name"));
            XElement factoryXml = (XElement)Reflect.Call(el, "ToXML");
            JObject factoryAttrs = Read.XAttrs(factoryXml);

            // AddComponetsUndoRedoCommand(list, container, index). The 3-arg ctor is the one the
            // designer's 新增栏位 handlers use (the 4-arg (posX, posY) one belongs to drag-drop);
            // for a Table/Tree container Execute() ends in AddNodeAt(el, index), and index =
            // child count is AddNodeAt's own clamp -- i.e. append.
            int index = Session.ChildCount(parent);
            Type listType = typeof(List<>).MakeGenericType(el.GetType());
            IList one = (IList)Activator.CreateInstance(listType);
            one.Add(el);
            object cmd = Attr.NewCommand("AddComponetsUndoRedoCommand", one, parent, index);
            Attr.Run(s, cmd);

            string newPath = path + "/" + name;
            object fsmNew = Reflect.Call(s.Si, "FindNodeByName", name);
            object node;
            string kind = SlotKind(fsmNew, out node);
            string gotType = fsmNew == null ? null : Read.Str(Reflect.Prop(fsmNew, "SpecNodeType"));
            int promoted = Promote(fsmNew);

            // The layout side. FormWriter holds its edits inside itself and `save` renders that
            // instance, so the insert has to be applied to it or it never reaches the .4fd.
            XElement elXml = (XElement)Reflect.Call(el, "ToXML");
            Attr.Layout(s).AddNode(path, elXml);

            JObject nowAttrs = Read.XAttrs(elXml);
            JObject changed = DiffAttrs(factoryAttrs, nowAttrs);

            var res = new JObject();
            res["tag"] = elXml == null ? null : elXml.Name.LocalName;
            res["name"] = name;
            res["path"] = newPath;
            res["use"] = use;
            res["specNodeType"] = gotType;
            res["specKind"] = kind;
            res["specStatus"] = node == null ? null : StatusLetter(node);
            res["promoted"] = promoted;
            res["attrs"] = nowAttrs;
            res["layout"] = nowAttrs;
            // The acceptance for this function is that the element carries what the factory
            // produced. Both maps come from live objects: `factory` is the factory element's own
            // ToXML snapshotted before the command, `attrs` the same element after. They are NOT
            // equal, and the difference is not a bug in this file -- it is what
            // AddComponetsUndoRedoCommand does to the element on its way in:
            //
            //   posX/posY        XmlElement.AddNodeAt ends in MeasurePos() (the Grid position is
            //                    recomputed, so the factory's 0,0 becomes a laid-out value)
            //   repeat           AddNodeAt strips `repeat` on any non-ScrollGrid parent
            //   fieldId          assigned by the Record rebuild that the same command triggers
            //   aggregate*/lstr* the attribute completer materialising the type's full set
            //
            // So `factory` is the honest statement of "what the factory produced" and
            // `insertDelta` is what the container added. What must hold, and does, is that every
            // attribute the factory is responsible for (style / sizePolicy / gridWidth / noEntry /
            // image / action / tag) survives untouched -- `factoryKept` says which ones did not.
            res["factory"] = factoryAttrs;
            res["insertDelta"] = changed;
            res["factoryKept"] = FactoryKept(factoryAttrs, nowAttrs);
            res["layoutDelta"] = Delta(nowAttrs, newPath);
            if (gotType != specType) {
                var d = new JObject();
                d["path"] = newPath;
                d["use"] = use;
                d["expected"] = specType;
                d["got"] = gotType;
                throw Refuse("插入的元素被设计器判成了 " + gotType + " 而不是 " + specType
                    + "（规格节点是 " + (kind == null ? "无" : kind) + "），没有生成 "
                    + specType + " 规格节点", d);
            }
            return res;
        }

        /// <summary>{attr: newValue} for every attribute that differs between two maps, plus
        /// `attr: null` for one the insert removed. Read off the live element, never re-typed.</summary>
        static JObject DiffAttrs(JObject before, JObject after) {
            var d = new JObject();
            var keys = new List<string>();
            foreach (JProperty p in before.Properties()) keys.Add(p.Name);
            foreach (JProperty p in after.Properties()) if (!keys.Contains(p.Name)) keys.Add(p.Name);
            keys.Sort(StringComparer.Ordinal);
            foreach (string k in keys) {
                JToken b = before[k], a = after[k];
                if (JToken.DeepEquals(b, a)) continue;
                d[k] = a == null ? JValue.CreateNull() : a;
            }
            return d;
        }

        /// <summary>
        /// The attributes the factory itself set, checked after the insert. This is the part of the
        /// comparison that carries information: everything else on the element came from the type's
        /// attribute completer, so comparing whole maps would only ever report the container's own
        /// layout work.
        ///
        /// Membership is "the factory's value for this name", so it needs no table of names: the
        /// factory snapshot is the list. An attribute that survived unchanged is not reported.
        /// </summary>
        static JObject FactoryKept(JObject factoryAttrs, JObject nowAttrs) {
            var lost = new JObject();
            foreach (JProperty p in factoryAttrs.Properties()) {
                JToken now = nowAttrs[p.Name];
                if (now == null) { lost[p.Name] = JValue.CreateNull(); continue; }
                if (!JToken.DeepEquals(p.Value, now)) lost[p.Name] = now;
            }
            return lost;
        }

        /// <summary>The form's code_template, read from the .tsd text the way Read.FormTree does
        /// rather than through SpecificationInfo.GetCodeTemplate -- that one dereferences
        /// &lt;other&gt;&lt;code_template&gt; without a null check and would NRE on a form that has
        /// neither, which is a legitimate shape (cs_excel_in_xmdl_s01 has no &lt;other&gt;).</summary>
        static string CodeTemplate(Session s) {
            try {
                byte[] zip = Read.Zip(s);
                return Read.CodeTemplate(Reflect.EntryText(zip, Read.Entry(zip, ".tsd")));
            } catch { return ""; }
        }

        // ================================================================ add_action

        /// <summary>
        /// Gives an element its &lt;act&gt;, or mints a standalone one.
        ///
        /// A Button's SpecNodeType is ACTION, so SpecificationInfo.Add already minted the &lt;act&gt;
        /// while the element was being created (FindActSpecById) -- that transient is what this
        /// promotes, and promoting is what makes it persist: ToXml() drops a CREATE-status act, so
        /// "is it non-null" is the wrong persistence test. test/Edit.cs's `act` op is the same
        /// sequence and says so at :589.
        ///
        /// `path` is the element to bind. The empty string is the standalone case -- 新增项目 in
        /// the 项目 panel, ActionDefaults.ExecutedAddAction -- which creates a bare
        /// SpecActionNode.Create(info, GetNewActionID()) with nothing bound to it. It has to be
        /// expressible somehow, and "" is the only value Manifest's required `path` admits that can
        /// never be a name-path.
        ///
        /// `type` is the new action's 型态 set, run through the same parser and the same
        /// AddActionType/RemoveActionType delegation as set_action_types. "all" is what
        /// SpecActionNode.Create writes and what every act in the corpus carries; "none" asks for
        /// the empty set, which the designer's own third rule turns straight back into "all" --
        /// reported in the result's `note` rather than passed over.
        /// </summary>
        public static object AddAction(Session s, JObject a) {
            string path = Read.Arg(a, "path");
            if (path == null) throw TzsError.Validation("缺少参数 path（控件代号路径；独立动作传空串）");
            string typeSpec = Read.Arg(a, "type");
            if (typeSpec == null) throw TzsError.Validation("缺少参数 type");
            string wantName = Read.Arg(a, "name");

            List<string> vocab;
            List<string> target = ParseTypes(s, typeSpec, out vocab);

            Attr.Urm(s);

            object node = null;
            string actId;
            bool standalone = path.Length == 0;
            bool reused = false, created = false, isDefault = false;

            if (standalone) {
                actId = string.IsNullOrEmpty(wantName) ? Read.Str(Reflect.Call(s.Si, "GetNewActionID")) : wantName;
                node = FindLiveAct(s, actId);
                if (node != null) {
                    reused = true;
                } else {
                    isDefault = (bool)Reflect.Call(s.Si, "IsActionDefault", actId);
                    node = Reflect.Call(Reflect.Find(Designer.A, "SpecActionNode"), "Create", s.Si, actId);
                    if (node == null) throw new TzsError("internal", "SpecActionNode.Create 返回了 null");
                    AddAct(s, node);
                    created = true;
                }
            } else {
                object el = Attr.El(s, path);
                string elName = Read.Str(Session.Raw(el, "name"));
                object fsm = Attr.Fsm(s, el);
                string nt = Read.Str(Reflect.Prop(fsm, "SpecNodeType"));
                if (nt != "ACTION") {
                    var d = new JObject();
                    d["path"] = path;
                    d["specNodeType"] = nt;
                    throw Refuse("\"" + elName + "\" 的 SpecNodeType 是 " + nt + "，不是 ACTION"
                        + "（只有 Button、且 style 不是 button_qrystr 才是）", d);
                }
                actId = string.IsNullOrEmpty(wantName) ? elName : wantName;

                object existing = Reflect.Prop(fsm, "SpecAction");
                if (existing != null && !IsCreate(existing)) {
                    // The file already has it. Not an error, and emphatically not a second node:
                    // report the no-op the way the designer's own panel would -- as a success frame
                    // carrying E_NO_OP (SPEC §11.24's error-code table), so a caller can tell it
                    // apart from "created" and from a real refusal.
                    var f = new JObject();
                    f["tag"] = "act";
                    f["actionTypes"] = ActionTypesOf(existing);
                    f["specStatus"] = StatusLetter(existing);
                    f["reused"] = true;
                    f["created"] = false;
                    f["path"] = path;
                    return NoOp(actId, f, "这个控件在文件里已经有 <act>（status=" + StatusLetter(existing)
                        + "），没有新建、也没有改动");
                }
                if (existing != null) {
                    node = existing;                  // the loading-minted transient
                    reused = true;
                } else {
                    isDefault = (bool)Reflect.Call(s.Si, "IsActionDefault", actId);
                    node = Reflect.Call(Reflect.Find(Designer.A, "SpecActionNode"), "Create", s.Si, actId);
                    if (node == null) throw new TzsError("internal", "SpecActionNode.Create 返回了 null");
                    AddAct(s, node);
                    Reflect.Call(fsm, "SetSpecNode", node);
                    created = true;
                }
            }

            string before = ActionTypesOf(node);
            string after = ApplyTypes(node, target);
            SetModify(node);

            var res = ActResult(node, actId, true, reused, created);
            res["before"] = before;
            res["actionTypes"] = after;
            res["typeRequested"] = typeSpec;
            res["vocabulary"] = Read.StrArray(vocab);
            if (standalone) res["standalone"] = true;
            if (isDefault) res["isActionDefault"] = true;
            if (after == null || after.Length == 0)
                res["note"] = "这个 act 没有型态属性；设计器不允许再给它设置型态（Message_ActionHasType）";
            // `target.Count == 0` rather than "the value changed": the point of the note is that
            // the request asked for NO 型态 and the designer's own third rule turned that into
            // "all" -- which is just as true on the path where nothing appeared to change (a fresh
            // act is already "all"), and that is exactly the case a caller would misread.
            else if (target.Count == 0)
                res["note"] = "type=\"none\" 请求的是空型态集合；设计器的规则 3（摘掉最后一个型态会把 \"all\" "
                    + "加回来）让它落到 \"" + after + "\"";
            else if (isDefault)
                res["note"] = actId + " 是 ActionDefault 名字：设计器的常规路径（FindActSpecById）"
                    + "不会为它建 <act>，这里走的是 Action 面板那条（FindActionDefaultSpecOrCreate）；"
                    + "要不要保留由调用方决定";
            return res;
        }

        /// <summary>SpecificationInfo.Add(SpecActionNode) -- the collection add plus the
        /// SpecPropertiesChangedEvent publish, so the designer's own views stay in step.</summary>
        static void AddAct(Session s, object node) { Reflect.Call(s.Si, "Add", node); }

        /// <summary>The live (non-tombstoned) act with this id, or null. FindActSpecById's own
        /// scan, which is why it is not a dictionary lookup.</summary>
        static object FindLiveAct(Session s, string id) {
            foreach (object a in ActsById(s, id))
                if (!IsDelete(a)) return a;
            return null;
        }

        /// <summary>The shape both add_action returns share. `specStatus` is the four-state letter
        /// (c / u / d), so a caller can see that a freshly minted act was promoted out of CREATE
        /// -- "c" would mean it is in memory only and ToXml() will drop it from the .tsd.
        ///
        /// `applied` is the positive marker SPEC §11.24's error-code table asks the applied
        /// outcome to carry (the other two are `clamped` and `noop`).</summary>
        static JObject ActResult(object node, string actId, bool changed, bool reused, bool created) {
            var res = new JObject();
            res["tag"] = "act";
            res["id"] = actId;
            res["actionTypes"] = ActionTypesOf(node);
            res["changed"] = changed;
            if (changed) res["applied"] = true;
            res["reused"] = reused;
            res["created"] = created;
            res["specStatus"] = node == null ? null : StatusLetter(node);
            return res;
        }

        // ================================================================ delete_action

        /// <summary>
        /// DeleteActUndoRedoCommand(act) -- which is a tombstone, not a removal: Execute() sets
        /// Status |= DELETE on the act and on its &lt;act_string&gt;, and SaveToTSD then writes the
        /// act out with status="d" (that is the SPEC's 墓碑 convention, and it is what makes the
        /// delete undoable).
        ///
        /// The designer's CanExecute gate for this command (ActionDefaultCommands
        /// .CanExecuteDeleteAction) also refuses ActionDefault-named actions and actions that are
        /// bound to a component -- it is a UI affordance for the 项目 panel, whose list only ever
        /// holds unbound ones, and it is deliberately NOT enforced here: `delete_action` takes an
        /// id, and an id that is bound is a legitimate thing for a caller to ask about (measured:
        /// deleting a bound act still leaves validate and RoundTrip clean). Repeated deletion is
        /// answered as a no-op in a SUCCESS frame (`noop:true` + `code:"E_NO_OP"`, SPEC §11.24's
        /// error-code table) rather than by tombstoning twice.
        /// </summary>
        public static object DeleteAction(Session s, JObject a) {
            string id = Read.Need(a, "id");

            object node = FindAct(s, id);
            if (IsDelete(node)) {
                var f = new JObject();
                f["tag"] = "act";
                f["deleted"] = false;
                f["specStatus"] = "d";
                return NoOp(id, f, "action \"" + id + "\" 已经是删除状态（status=d），没有再次打墓碑");
            }

            Attr.Urm(s);
            object cmd = Attr.NewCommand("DeleteActUndoRedoCommand", node);
            Attr.Run(s, cmd);

            var res = new JObject();
            res["id"] = id;
            res["tag"] = "act";
            res["deleted"] = true;
            res["changed"] = true;
            res["applied"] = true;
            res["specStatus"] = StatusLetter(node);
            res["note"] = "墓碑：<act id=\"" + id + "\"> 以 status=\"d\" 留在 .tsd 里（可撤销），"
                + "不是从文件里消失";
            return res;
        }

        // ================================================================ set_action_types

        /// <summary>
        /// Sets an action's 型态 combination, reproducing SpecActionNode's four rules exactly by
        /// calling its own AddActionType / RemoveActionType (see the block comment above
        /// ApplyTypes). `types` is a comma-separated list or a single token, and "none" asks for
        /// the empty set.
        ///
        /// The vocabulary comes from the form (TypeVocabulary), so a caller is told the legal
        /// tokens for THIS form in error.detail and does not have to guess at "db1".
        ///
        /// NOTE for the caller: Manifest.cs declares `types` as an Enum with the two values
        /// all|none, and Rpc's dispatcher refuses anything else before this body runs. The real
        /// vocabulary is all / mi / di&lt;n&gt; / db&lt;n&gt;, which a static enum cannot express,
        /// so the tokens this function actually understands are wider than the declaration --
        /// through the JSON surface only all and none are reachable today.
        /// </summary>
        public static object SetActionTypes(Session s, JObject a) {
            string id = Read.Need(a, "id");
            string raw = Read.Arg(a, "types");
            if (raw == null) throw TzsError.Validation("缺少参数 types");

            List<string> vocab;
            List<string> target = ParseTypes(s, raw, out vocab);

            Attr.Urm(s);
            object node = FindAct(s, id);
            if (IsDelete(node)) {
                var d = new JObject();
                d["id"] = id;
                throw Refuse("action \"" + id + "\" 已经是删除状态（status=d），改它的型态没有意义", d);
            }
            GuardTypable(s, node, id);

            string before = ActionTypesOf(node);
            string after = ApplyTypes(node, target);

            if (after == before) {
                var f = new JObject();
                f["tag"] = "act";
                f["actionTypes"] = before;
                f["requested"] = raw;
                f["vocabulary"] = Read.StrArray(vocab);
                return NoOp(id, f, "action \"" + id + "\" 的型态已经是 \"" + before + "\"，未改动任何值"
                    + ((string.IsNullOrEmpty(raw) || raw.IndexOf("none", StringComparison.OrdinalIgnoreCase) >= 0)
                       ? "：none 要的是空集合，摘掉最后一个型态后设计器的规则 3 把 \"all\" 加了回来，所以结果和原来一样"
                       : ""));
            }

            var res = new JObject();
            res["id"] = id;
            res["tag"] = "act";
            res["before"] = before;
            res["after"] = after;
            res["types"] = Read.StrArray(SplitTypes(after));
            res["requested"] = raw;
            res["vocabulary"] = Read.StrArray(vocab);
            res["changed"] = true;
            res["applied"] = true;
            res["specStatus"] = StatusLetter(node);
            res["note"] = "型态是 .tsd 属性，不在 .4fd 里，所以没有 layoutDelta";
            return res;
        }

        /// <summary>
        /// The "already this value" answer, as a SUCCESS frame carrying `noop:true` and
        /// `code:"E_NO_OP"` -- SPEC §11.24's error-code table (amended mid-wave, commit 0c607df)
        /// is explicit that E_NO_OP and E_ATTR_CLAMPED are 成功码 that live in `result`, not in
        /// `error`, and that the three outcomes carry exactly one positive marker each
        /// (applied / clamped / noop).
        ///
        /// It is not mere bookkeeping: Rpc.Map classifies any E_* code it does not know as
        /// `kind:internal`, whose contract meaning is "unexpected exception -- report it, do not
        /// retry". Sending a benign no-op down that channel would tell the caller the opposite of
        /// the truth. Attr.cs's set_layout_attr still throws its no-op; that predates the amendment
        /// and is not this file's to change.
        /// </summary>
        static JObject NoOp(string id, JObject fields, string note) {
            var res = new JObject();
            res["id"] = id;
            if (fields != null) foreach (JProperty p in fields.Properties()) res[p.Name] = p.Value;
            res["changed"] = false;
            res["noop"] = true;
            res["code"] = "E_NO_OP";
            if (note != null) res["note"] = note;
            return res;
        }
    }
}
