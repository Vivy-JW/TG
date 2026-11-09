using System;
using System.Collections;
using System.Collections.Generic;
using System.Text;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;

namespace TzsCli.Designer
{
    // Deliberately NOT TzsCli.Designer.Fns like three of the modules in this directory.
    // Fns/Session.cs declares a static module class also called Session, and a type in the
    // same namespace outranks an enclosing one -- so inside .Fns the name `Session` would
    // bind to that module and every `Session s` parameter here would be CS0721. W2-B hit
    // this and chose the enclosing namespace; Rpc.BuildMap dispatches by class name, so the
    // split costs nothing. Do not "tidy" it without renaming the module class first.

    /// <summary>
    /// The structure functions: the twelve operations that change the shape of the form
    /// (add / insert / delete / reorder / move / align / fit / wrap / break / convert).
    ///
    /// Every one of them is the designer's own UndoRedo command, constructed with the
    /// arguments the designer's own call site passes and executed through the handle's
    /// registered UndoRedoManager. Nothing here re-implements placement, mime gating, name
    /// minting or spec-node minting -- SPEC §11.24 (d) 1's rule for attributes ("never call
    /// the command directly") has an exact structural twin here: never re-derive what the
    /// command already owns.
    ///
    /// TWO SIDES, AND THEY MUST AGREE.
    ///
    /// A designer command edits the MODEL (XmlElement trees). `save` does not render the
    /// model -- it renders the per-handle FormWriter from Fns/Session.cs. So every function
    /// here does the layout edit through `Fns.Session.Layout(s)` immediately after the
    /// command, and the .4fd half is derived from the model AFTER the command ran (the
    /// element's own ToXML(), placed at the index the model actually gave it). Deriving it
    /// from the model rather than from the request is what keeps the two halves agreeing
    /// where the designer's command surprises you: AddComponetsUndoRedoCommand inserts at
    /// position 0 whenever the container is not one of HBox/VBox/Table/Tree/Folder, and
    /// AddToContainerUndoRedoCommand puts its new box at the head. Reading the resulting
    /// index back is how this file stays right about that instead of encoding a guess.
    /// (test/Edit.cs's `add` splices at the END and so disagrees with its own model; that is
    /// a latent bug there, not a behaviour worth copying.)
    ///
    /// The FormWriter cannot express "insert before child 0" -- AddNode anchors on an
    /// existing child's closing tag, so its index means "after child[i]" (FormWriter.cs
    /// :174-179) -- hence PlaceChild and ResyncChildren below, which rebuild the parent's
    /// child list when the head is involved.
    /// </summary>
    public static class Struct
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["add_widget"]        = AddWidget;
            into["add_field"]         = AddFieldFn;
            into["insert_at"]         = InsertAt;
            into["delete"]            = DeleteFn;
            into["move"]              = MoveFn;
            into["nudge"]             = Nudge;
            into["align"]             = AlignFn;
            into["fit_size"]          = FitSize;
            into["wrap"]              = Wrap;
            into["break_layout"]      = BreakLayoutFn;
            into["convert_widget"]    = ConvertWidget;
            into["convert_container"] = ConvertContainer;
            into["reparent"]          = ReparentFn;
        }

        // ================================================================ designer types

        static Type Cft() { return Reflect.Find(Designer.A, "ComponentFactory"); }
        static Type ElT() { return Reflect.Find(Designer.A, "XmlElement"); }
        static Type CtT() { return Reflect.Find(Designer.A, "ComponentType"); }
        static Type FeT(string n) { return Designer.FE == null ? null : Reflect.Find(Designer.FE, n); }

        /// <summary>Type name -> the designer's enum member. The manifest's enums already
        /// constrain the wire value to a legal spelling, so a miss here means the manifest and
        /// the designer have drifted apart -- reported as validation because the caller can
        /// switch to another value, and the message names the legal set.</summary>
        static object CType(string name) {
            Type t = CtT();
            if (t == null) throw new TzsError("internal", "找不到 ComponentType 枚举（设计器版本变了？）");
            try { return Enum.Parse(t, name, true); }
            catch {
                throw TzsError.Validation("未知控件类型 " + name + "；合法值: "
                    + string.Join(", ", Enum.GetNames(t)));
            }
        }

        static object EnumT(string typeName, string member, string argName, string[] legal) {
            Type t = Reflect.Find(Designer.A, typeName);
            if (t == null) throw new TzsError("internal", "找不到 " + typeName + " 枚举（设计器版本变了？）");
            try { return Enum.Parse(t, member); }
            catch { throw TzsError.Validation("未知 " + argName + "；合法值: " + string.Join(", ", legal)); }
        }

        static object CTypeOf(object el) { return Reflect.Prop(el, "Type"); }
        static string TagOf(object el) { return Reflect.S(Reflect.Prop(el, "NodeName")); }

        static string NameOf(object el) {
            object n = Reflect.Prop(el, "Name");
            return n == null ? null : n.ToString();
        }

        static int IntOf(object v) {
            if (v == null) return 0;
            int i;
            return int.TryParse(v.ToString(), out i) ? i : 0;
        }

        /// <summary>The designer's own mime table (core-br.spec + mod-fd.spec, loaded by
        /// SpecificationInfo's ctor through ComponentFactory.SetSpecification). Asked in the
        /// same direction the designer asks it: `AcceptMimes(containerType, childElement)` for
        /// "may this go in there", and the string form for "may this container type itself go
        /// there".
        ///
        /// This is the load-bearing gate for wrap. The table says modFD/HBox accepts
        /// Folder/Grid/Group/HBox/ScrollGrid/Table/Tree/VBox and NOTHING else, so wrapping a
        /// plain widget into an HBox is refused here rather than failing obscurely inside the
        /// command -- and that refusal is the negative control for this module.</summary>
        static bool Accepts(object parentEl, object childEl) {
            Type t = Cft();
            if (t == null) throw new TzsError("internal", "找不到 ComponentFactory（设计器版本变了？）");
            object r = Reflect.Call(t, "AcceptMimes", parentEl, childEl);
            return r is bool && (bool)r;
        }

        static bool Accepts(string parentTag, string childTag) {
            Type t = Cft();
            if (t == null) throw new TzsError("internal", "找不到 ComponentFactory（设计器版本变了？）");
            object r = Reflect.Call(t, "AcceptMimes", parentTag, childTag);
            return r is bool && (bool)r;
        }

        static bool IsContainer(string nodeName) {
            Type t = Reflect.Find(Designer.A, "FormDesignSetting");
            if (t == null) throw new TzsError("internal", "找不到 FormDesignSetting");
            object r = Reflect.Call(t, "IsContainer", nodeName);
            return r is bool && (bool)r;
        }

        static bool IsFormField(string nodeName) {
            Type t = Reflect.Find(Designer.A, "FormDesignSetting");
            if (t == null) throw new TzsError("internal", "找不到 FormDesignSetting");
            object r = Reflect.Call(t, "IsFormField", nodeName);
            return r is bool && (bool)r;
        }

        /// <summary>A designer refusal: a fact about the form, not about the request. The caller
        /// must not retry (SPEC §11.24 (a)). Same code Attr's refusals carry, so the transport
        /// classifies both as `designer` through one table rather than two.</summary>
        static DetailedError Refuse(string message) { return new DetailedError("E_DESIGNER", message, null); }

        // ================================================================ session plumbing

        static FormWriter W(Session s) { return Attr.Layout(s); }

        /// <summary>The element's index among its parent's children -- the model's own z-order.
        /// XmlElement.Index is literally `Parent.Nodes.IndexOf(this)` (XmlElement.cs:1461).</summary>
        static int IndexOf(object el) { return IntOf(Reflect.Prop(el, "Index")); }

        static object ParentOf(object el) { return Reflect.Prop(el, "Parent"); }

        static List<object> ModelChildren(object el) {
            var res = new List<object>();
            var nodes = Reflect.Prop(el, "Nodes") as IEnumerable;
            if (nodes != null) foreach (object c in nodes) res.Add(c);
            return res;
        }

        static IList XmlList(params object[] els) {
            Type xt = ElT();
            if (xt == null) throw new TzsError("internal", "找不到 XmlElement 类型（设计器版本变了？）");
            IList list = (IList)Activator.CreateInstance(typeof(List<>).MakeGenericType(xt));
            foreach (object e in els) list.Add(e);
            return list;
        }

        static object NewCmd(string name, params object[] args) { return Attr.NewCommand(name, args); }
        static void Run(Session s, object cmd) { Attr.Run(s, cmd); }

        /// <summary>Registers the handle's UndoRedoManager (the one-time Loaded -> Mutable step,
        /// SPEC §11.24 (b)) and returns it. Idempotent: a second call hands back the same
        /// manager rather than orphaning the first.</summary>
        static object Urm(Session s) { return Attr.Urm(s); }

        /// <summary>The handle's FormWriter must know the path too. A path that resolves in the
        /// model but not in the text means the two have already diverged, and half-applying a
        /// structural edit on top of that is worse than refusing.</summary>
        static void NeedW(FormWriter w, string path) {
            if (w.Index.ByPath(path) == null)
                throw TzsError.NotFound("布局里的路径", path);
        }

        static string ParentPath(string path) {
            int i = path.LastIndexOf('/');
            return i > 0 ? path.Substring(0, i) : null;
        }

        // ================================================================ .4fd text side

        /// <summary>The exact text of the element at <paramref name="path"/>, parsed. Not the
        /// model's ToXML(): for an element that is already in the file this is byte-for-byte
        /// what is there, which is what keeps a re-placement from reformatting the document.</summary>
        static XElement SpanXml(FormWriter w, string path) {
            ElementSpan sp = w.Index.ByPath(path);
            if (sp == null) return null;
            return XElement.Parse(w.Index.Text.Substring(sp.OpenStart, sp.CloseEnd - sp.OpenStart));
        }

        /// <summary>The one thing FormWriter.Indent() cannot reproduce is character data; every
        /// .4fd in the corpus is attribute-only, but a form that carried text would lose it
        /// silently through the rebuild paths, so say so instead.</summary>
        static void NoTextNodes(XElement e) {
            foreach (XNode n in e.Nodes()) {
                if (n is XText && ((XText)n).Value.Trim().Length > 0)
                    throw new TzsError("internal",
                        "<" + e.Name.LocalName + "> 的元素内容里有文本，FormWriter 的缩进重建会丢掉它；"
                        + "这个结构操作在文本层做不了");
            }
        }

        /// <summary>Places <paramref name="el"/> at child position <paramref name="index"/> of
        /// the container at <paramref name="parentPath"/>, mirroring `Nodes.Insert(index, el)`.
        /// index == 0 needs the rebuild path; everything else maps onto AddNode's "after
        /// child[i]" as index-1.
        ///
        /// KNOWN COSMETIC ARTIFACT. FormWriter.Indent() builds an element's own opening line
        /// without a leading indent -- only its CHILDREN get `indent + 2` -- so every element
        /// AddNode writes lands at column 0. That is pre-existing (the batch-validated
        /// test/AddField.exe emits one such line per produced element, and Edit.cs's `add`/`wrap`
        /// the same), RoundTrip cannot see it (it compares canonical XML), and validate does not
        /// care. The rebuild path below makes it visible on MORE lines -- it re-adds every
        /// sibling, each losing its own indent -- which is the price of the one insertion
        /// position AddNode cannot express. Measured on aapp320(114): nudge 0 lines, add_widget 1
        /// (the new element, same as the reference tool), move to=first 25 (the Table's children,
        /// which is exactly that one position). Fixing it belongs in FormWriter.Indent, not here.</summary>
        static void PlaceChild(FormWriter w, string parentPath, XElement el, int index) {
            ElementSpan parent = w.Index.ByPath(parentPath);
            if (parent == null) throw TzsError.NotFound("布局里的路径", parentPath);
            if (parent.Children.Count == 0) { w.AddNode(parentPath, el); return; }   // AddNode's empty-container branch

            if (index < 0) index = 0;
            if (index == 0) {
                var xml = new List<XElement>();
                var paths = new List<string>();
                foreach (ElementSpan c in parent.Children) { paths.Add(c.Path); xml.Add(SpanXml(w, c.Path)); }
                foreach (string p in paths) w.RemoveNode(p);
                w.AddNode(parentPath, el);
                foreach (XElement x in xml) { NoTextNodes(x); w.AddNode(parentPath, x); }
                return;
            }
            if (index >= parent.Children.Count) index = parent.Children.Count - 1;
            w.AddNode(parentPath, el, index - 1);
        }

        /// <summary>Rewrites the container's child list so the text order equals the model
        /// order. Used where a command moved more than one child (add_field splices a whole
        /// UICreator result; wrap and break_layout hand children between containers), because
        /// those land at the head and PlaceChild can only rebuild one at a time.
        ///
        /// `resolved` supplies the XElement for a model element that is not in the text yet;
        /// anything else is read back out of the text before it is removed.</summary>
        static void ResyncChildren(FormWriter w, Session s, object parentEl, string parentPath,
                                   List<KeyValuePair<object, XElement>> resolved) {
            ElementSpan parent = w.Index.ByPath(parentPath);
            if (parent == null) throw TzsError.NotFound("布局里的路径", parentPath);

            var byRef = new List<KeyValuePair<object, XElement>>();
            var paths = new List<string>();
            foreach (ElementSpan c in parent.Children) {
                paths.Add(c.Path);
                object el = s.FindByPath(c.Path);
                XElement xe = SpanXml(w, c.Path);
                if (el != null && xe != null) { NoTextNodes(xe); byRef.Add(new KeyValuePair<object, XElement>(el, xe)); }
            }

            var want = new List<XElement>();
            foreach (object child in ModelChildren(parentEl)) {
                XElement xe = Lookup(resolved, child);
                if (xe == null) xe = Lookup(byRef, child);
                if (xe == null) throw new TzsError("internal",
                    "容器 " + parentPath + " 的模型子节点 \"" + (NameOf(child) ?? TagOf(child))
                    + "\" 在布局文本里没有对应元素——模型和文本已经分叉，不能继续写");
                want.Add(xe);
            }

            foreach (string p in paths) w.RemoveNode(p);
            foreach (XElement x in want) w.AddNode(parentPath, x);
        }

        /// <summary>Reference lookup, not Equals: XmlElement is a view-model class and the
        /// identity of "this child after the command" is the object, not a value.</summary>
        static XElement Lookup(List<KeyValuePair<object, XElement>> map, object key) {
            for (int i = 0; i < map.Count; i++)
                if (object.ReferenceEquals(map[i].Key, key)) return map[i].Value;
            return null;
        }

        /// <summary>The element's path, by walking the model. Needed for elements whose path the
        /// caller never gave us -- a wrap's children move into a new box, a delete pulls in the
        /// bound label companion, a convert replaces the object.</summary>
        static string PathOf(Session s, FormWriter w, object target) {
            if (target == null) return null;
            object formNode = Read.FormNode(s);
            if (formNode == null || w.Index.All.Count == 0) return null;
            string start = w.Index.All[0].Name + "/" + Session.Seg(formNode);
            var els = new List<object>();
            var paths = new List<string>();
            Session.CollectAll(formNode, start, paths, els);
            for (int i = 0; i < els.Count; i++)
                if (object.ReferenceEquals(els[i], target)) return paths[i];
            return null;
        }

        // ================================================================ result shapes

        /// <summary>An `el` result: the same fields form_tree/find_component emit for one node,
        /// built by the same builder so the two surfaces cannot drift.</summary>
        static JObject ElResult(Session s, object el, string path) {
            var sb = new StringBuilder();
            Json.InfoRef(el, path, sb, Read.SpecDic(s));
            return JObject.Parse(sb.ToString());
        }

        /// <summary>A `delta` result: one entry per element, each carrying only the layout
        /// attributes that actually moved. The change is already applied; this reports it, in
        /// the same spirit as Attr's layoutDelta.</summary>
        static JObject DeltaOf(List<string> paths, List<Dictionary<string, string>> before, List<object> els) {
            var arr = new JArray();
            for (int i = 0; i < els.Count; i++) {
                var one = new JObject();
                one["path"] = paths[i];
                one["name"] = NameOf(els[i]);
                one["tag"] = TagOf(els[i]);
                one["attrs"] = Attr.Diff(before[i], Session.Attrs(els[i]));
                arr.Add(one);
            }
            var res = new JObject();
            res["count"] = els.Count;
            res["delta"] = arr;
            return res;
        }

        /// <summary>Applies one element's layout delta to the .4fd and returns the same delta as
        /// JSON, so the two cannot be reported inconsistently.</summary>
        static JObject PatchOne(FormWriter w4, string path, object el, Dictionary<string, string> before) {
            JObject moved = Attr.Diff(before, Session.Attrs(el));
            Attr.Patch(w4, path, moved);
            var one = new JObject();
            one["path"] = path;
            one["name"] = NameOf(el);
            one["tag"] = TagOf(el);
            one["attrs"] = moved;
            return one;
        }

        /// <summary>Snapshots the layout attributes of every path, so a command's effect on the
        /// text can be limited to what really moved (SPEC §11.24 (d) 2's discipline, applied to
        /// structure: rewriting an unchanged attribute still rewrites bytes for no reason).</summary>
        static List<object> ResolveAll(Session s, string[] paths, out List<Dictionary<string, string>> before) {
            var els = new List<object>();
            before = new List<Dictionary<string, string>>();
            foreach (string p in paths) {
                object el = Attr.El(s, p);
                els.Add(el);
                before.Add(Session.Attrs(el));
            }
            return els;
        }

        /// <summary>`paths`, or a `path` that is a single value or an array.
        ///
        /// delete declares BOTH (`path` required, `paths` the batch form that takes over), so
        /// Manifest.Check refuses a batch call that omits `path` before this body ever runs. A
        /// caller who sends only the required `path` therefore gets array support for free, the
        /// same way set_layout_attr accepts its `path` as an array.</summary>
        static string[] PathsArg(JObject a, string fn) {
            string[] paths = Read.ListArg(a, "paths");
            if (paths == null || paths.Length == 0) {
                JToken pj = a == null ? null : a["path"];
                if (pj != null && pj.Type == JTokenType.Array) paths = Read.ListArg(a, "path");
            }
            if (paths == null || paths.Length == 0) {
                string one = Read.Arg(a, "path");
                if (!string.IsNullOrEmpty(one)) paths = new string[] { one };
            }
            if (paths == null || paths.Length == 0) throw TzsError.Validation(fn + " 需要 path 或 paths");
            return paths;
        }

        // ================================================================ add_widget

        /// <summary>
        /// A new empty widget in a container. Body of test/Edit.cs's `add`, minus its two
        /// defects: the .4fd splice follows the index the model gave the element, and the parent
        /// is checked in the text as well as in the model before anything is constructed.
        ///
        /// `ComponentType` comes from the manifest's WIDGETS enum, whose doc says why it is an
        /// enum and not a free string: ModFdInfo.IsIncludeAttribute NREs natively on a type
        /// outside the catalogue (SPEC §11.15).
        /// </summary>
        public static object AddWidget(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string type = Read.Need(a, "type");
            string wantName = Read.Arg(a, "name");

            object ctype = CType(type);
            string ctypeName = Reflect.S(ctype);
            object parentEl = Attr.El(s, path);
            FormWriter w4 = W(s);
            NeedW(w4, path);
            string parentTag = TagOf(parentEl);

            if (!Accepts(parentTag, ctypeName))
                throw Refuse(parentTag + " 不接受 " + ctypeName
                    + "（设计器的 mime 表不允许，界面上也拖不进去）");

            if (string.IsNullOrEmpty(wantName))
                wantName = Reflect.S(Reflect.Call(Cft(), "GetNewName", ctypeName.ToLower(), s.Key));
            if (IsExistingName(s, wantName) || IsExistingName(s, wantName.ToLowerInvariant()))
                throw Refuse("控件代号已存在: " + wantName);

            object el = Reflect.Call(Cft(), "CreateEmptyComponent", s.Key, ctype, wantName);
            if (el == null) throw new TzsError("internal", "CreateEmptyComponent 返回了 null");

            // The designer NORMALISES the name it is handed: asking for "zzProbe" produced an
            // element called "zzprobe". Looking the spec model up under the caller's spelling then
            // found nothing and the whole call died with E_INTERNAL -- so an explicit `name` was
            // advertised in the manifest and unusable. Read the real name back off the element
            // instead of guessing what the normalisation does.
            string elName = NameOf(el);
            if (string.IsNullOrEmpty(elName))
                throw new TzsError("internal", "CreateEmptyComponent 造出来的元素没有名字");

            // THE WIDGET BOX MAKES A LABEL TOO, and calling the bare CreateEmptyComponent -- which
            // is what this used to do -- is the one thing it never does. WidgetBox.xaml.cs:82 goes
            // through AddWidgetAdornerHelper to ComponentFactory.CreateEmptyComponentWithLabel,
            // which for the eight "needs a label" widget types builds a Label named "{widget}_1",
            // points BindElement at each other and registers the pair through
            // SpecificationInfo.AddSpecBinding (ComponentFactory.cs:49-134). The eight are Edit /
            // ComboBox / TextEdit / ButtonEdit / DateEdit / DateTimeEdit / SpinEdit / TimeEdit;
            // Button, CheckBox, Label and the rest get no label there either, so with_label is a
            // no-op for them rather than a lie.
            //
            // Those steps are reproduced rather than calling the helper, because the helper
            // generates its OWN widget name (it hands string.Empty to CreateEmptyComponent) and
            // this function's whole point is that the caller chooses one.
            object label = null;
            string labelName = null;
            if (TzsCli.Designer.Fns.Session.Bool(a, "with_label", true)
                && Array.IndexOf(LABELED_TYPES, ctypeName) >= 0) {
                labelName = Reflect.S(Reflect.Call(Cft(), "GetNewName", elName + "_1", s.Key));
                label = Reflect.Call(Cft(), "CreateEmptyComponent", s.Key, CType("Label"), labelName);
                if (label == null) throw new TzsError("internal", "CreateEmptyComponent(Label) 返回了 null");
                labelName = NameOf(label) ?? labelName;
                Reflect.SetProp(label, "BindElement", el);
                Reflect.SetProp(el, "BindElement", label);
            }

            int row = w4.NextFreeRow(path);
            // Reversed, so the prepend inside the command leaves [label, widget] -- see the note on
            // AddComponetsUndoRedoCommand's insertion order in this file's header.
            object cmd = NewCmd("AddComponetsUndoRedoCommand",
                label == null ? XmlList(el) : XmlList(el, label), parentEl, 1, row);
            Run(s, cmd);

            object fsm = Reflect.Call(s.Si, "FindNodeByName", elName);
            if (fsm == null) throw new TzsError("internal", "设计器没有为 " + elName + " 建立规格模型");
            int promoted = PromoteSlots(fsm);

            if (label != null) {
                object lfsm = Reflect.Call(s.Si, "FindNodeByName", labelName);
                if (lfsm != null) {
                    Reflect.Call(s.Si, "AddSpecBinding", Reflect.Prop(lfsm, "GeneroComponent"),
                                  Reflect.Prop(fsm, "GeneroComponent"));
                    // The label has a spec node of its own, and it is CREATE like the widget's was:
                    // unpromoted, ToXml() drops it and the label never reaches the .tsd.
                    promoted += PromoteSlots(lfsm);
                }
            }

            string elPath = PathOf(s, w4, el);
            if (elPath == null) throw new TzsError("internal", "新增元素没能定位回文本路径");
            if (label != null)
                PlaceChild(w4, path, (XElement)Reflect.Call(label, "ToXML"), IndexOf(label));
            PlaceChild(w4, path, (XElement)Reflect.Call(el, "ToXML"), IndexOf(el));

            var res = ElResult(s, el, elPath);
            res["promoted"] = promoted;
            res["parent"] = parentTag;
            // The index the model actually gave it -- not the one we asked for. Reported because
            // AddComponetsUndoRedoCommand does not honour a requested index on every container
            // type, and a caller that cannot see where the widget landed cannot tell.
            res["index"] = IndexOf(el);
            res["withLabel"] = label != null;
            if (label != null) {
                res["label"] = PathOf(s, w4, label);
                res["labelName"] = labelName;
            }
            return res;
        }

        /// <summary>The widget types CreateEmptyComponentWithLabel gives a Label to
        /// (ComponentFactory.cs:98-127). Everything else -- Button, CheckBox, Label, the containers
        /// -- is created bare there too.</summary>
        static readonly string[] LABELED_TYPES = {
            "Edit", "ComboBox", "TextEdit", "ButtonEdit",
            "DateEdit", "DateTimeEdit", "SpinEdit", "TimeEdit"
        };

        static bool IsExistingName(Session s, string name) {
            object r = Reflect.Call(s.Si, "IsExists", name);
            return r is bool && (bool)r;
        }

        /// <summary>Edit.cs's `add` promotion: every slot the element ended up holding.
        /// SPEC §11.24 (j) keeps this separate from the colName-gated one on purpose -- merging
        /// them either stops buttons promoting (a button has no colName) or wrongly promotes a
        /// Tree's five NON_DATABASE scaffolding children.</summary>
        static int PromoteSlots(object fsm) {
            object modify = Status("MODIFY");
            int n = 0;
            foreach (string slot in SpecSlots.ByKind.Values) {
                object node = Reflect.Prop(fsm, slot);
                if (node == null) continue;
                if (Reflect.S(Reflect.Prop(node, "Status")).IndexOf("CREATE", StringComparison.Ordinal) < 0) continue;
                Reflect.SetProp(node, "Status", modify);
                n++;
            }
            return n;
        }

        static object Status(string name) {
            Type t = Reflect.Find(Designer.A, "SpecStatus");
            if (t == null) throw new TzsError("internal", "找不到 SpecStatus 枚举（设计器版本变了？）");
            return Enum.Parse(t, name);
        }

        /// <summary>AddField.cs's Promote: the same walk, gated on a non-empty colName, plus the
        /// sfield list (the lbl_/cmt_ field strings, which are flat and not in
        /// FormSpeDictionary). The gate is the predicate SpecNodeTransform.TransformFieldType
        /// uses: UICreator's Tree brings five scaffolding children whose colName is "" and which
        /// no Tree in the corpus carries .tsd nodes for (SPEC §11.14).</summary>
        static int PromoteByColName(Session s, HashSet<string> names) {
            object modify = Status("MODIFY");
            int n = 0;
            IDictionary dict = Read.SpecDic(s);
            foreach (DictionaryEntry de in dict) {
                string key = de.Key == null ? null : de.Key.ToString();
                if (key == null || !names.Contains(key)) continue;
                object fsm = de.Value;
                object comp = Reflect.Prop(fsm, "GeneroComponent");
                if (comp == null) continue;
                object colName = Reflect.Call(comp, "GetAttribute", "colName");
                if (colName == null || colName.ToString().Length == 0) continue;
                foreach (string slot in SpecSlots.ByKind.Values) {
                    object node = Reflect.Prop(fsm, slot);
                    if (node == null) continue;
                    if (Reflect.S(Reflect.Prop(node, "Status")).IndexOf("CREATE", StringComparison.Ordinal) < 0) continue;
                    Reflect.SetProp(node, "Status", modify);
                    n++;
                }
            }
            var fss = Reflect.Prop(s.Si, "fieldStrings") as IEnumerable;
            if (fss != null) {
                foreach (object item in fss) {
                    string nm = NameOf(item);
                    if (nm == null || !names.Contains(nm)) continue;
                    if (Reflect.S(Reflect.Prop(item, "Status")).IndexOf("CREATE", StringComparison.Ordinal) < 0) continue;
                    Reflect.SetProp(item, "Status", modify);
                    n++;
                }
            }
            return n;
        }

        // ================================================================ add_field

        /// <summary>The containers <c>UICreator.Create</c> can build INTO -- its six dispatch
        /// branches (SPEC §11.10). This is the whitelist this file refuses against, and it is also
        /// what the manifest publishes for `container`.
        ///
        /// ONE PRODUCER, and it did not use to be: the manifest had its own ten-entry list (six
        /// creation targets plus the three that exist only as commands: HBox/VBox are
        /// AddToContainerUndoRedoCommand targets §11.17, Page is a Folder's only legal child, and
        /// Folder itself), so `--help` advertised four values that the body then refused with
        /// "未知容器模式". A caller reading the parameter table had no way to know.
        /// (Measured 2026-09-25; §11.9 item 5.)</summary>
        internal static readonly string[] CONTAINER_TYPES =
            { "None", "Grid", "Group", "ScrollGrid", "Table", "Tree" };

        /// <summary>
        /// A field from the workspace's data dictionary, built by the designer's own UICreator so
        /// the widget type, the label companion, the placement and the .bdx binding are the
        /// designer's decisions rather than ours (test/AddField.cs is the reference for every
        /// mechanical step below).
        ///
        /// The one structural difference from test/AddField.cs: it needed a SECOND load, because
        /// UICreator builds its elements outside the model and only a reload would register them.
        /// A long-lived session cannot reload -- SPEC §11.24 (b): a registered UndoRedoManager
        /// makes a re-open throw -- so the produced elements are handed to
        /// AddComponetsUndoRedoCommand instead. That command does the registration
        /// (SpecificationInfo.Add, recursively) AND the column enrichment that AddField.cs had to
        /// perform by hand in its `Enrich` step, so this is strictly less re-derivation.
        /// </summary>
        public static object AddFieldFn(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string table = Read.Need(a, "table");
            // One column, or many. `columns` is the designer's own shape, not an invention: its
            // WidgetBox button "Create a structure from database columns" opens DBStructureCreator,
            // whose only non-UI line is ONE call to
            // UICreator.Create(LayoutEditor, container, SelectedFields, key)
            // (DBStructureCreator.xaml.cs:178) -- N columns in a single construction. The legacy
            // AddField.exe already did it that way (AddField.cs:242-275).
            //
            // Calling it once per column is what made adding 84 columns take 137 s: the designer
            // redoes its whole per-call setup every time, and that setup costs more the larger the
            // form already is (measured: per-field cost climbed 0.55 -> 3.25 s, so the total is
            // quadratic). It is also what put a <Label> beside every widget -- the default
            // container mode is CreateNoneContainerWidget, which exists to make label+widget
            // pairs, while CreateTableContainer returns the Table alone.
            string[] columns = Read.ListArg(a, "columns");
            string single = Read.Arg(a, "column");
            if (columns == null || columns.Length == 0)
                columns = string.IsNullOrEmpty(single) ? null : new string[] { single };
            if (columns == null || columns.Length == 0)
                throw TzsError.Validation(
                    "add_field 需要 column（单列）或 columns（多列）之一；列名可用 list_columns 取");
            string container = Read.Arg(a, "container");
            string widgetArg = Read.Arg(a, "widget");
            if (string.IsNullOrEmpty(container)) container = "None";
            if (Array.IndexOf(CONTAINER_TYPES, container) < 0)
                throw TzsError.Validation("未知容器模式 " + container + "；合法值: "
                    + string.Join(", ", CONTAINER_TYPES));

            object parentEl = Attr.El(s, path);
            FormWriter w4 = W(s);
            NeedW(w4, path);
            string parentTag = TagOf(parentEl);

            // The handle must be Mutable BEFORE UICreator runs, not merely before the command
            // does: the creators position their elements through the GridX/GridY setters, which
            // dispatch FormPosUndoRedoCommand and throw "No UndoRedoManager" when the key has
            // none. test/AddField.cs registers the manager in pass 1 for exactly this reason
            // (:220-223, "UICreator's layout setters need it"). Urm is idempotent (Attr.Urm), so
            // the AddComponetsUndoRedoCommand below reuses the same manager.
            Urm(s);

            // The per-column lookup moved into the loop below: one PrepareAddColumn per column,
            // all of them handed to a single creator call.

            Type cvmT = FeT("ContainerViewModel"), pacT = FeT("PrepareAddColumn"),
                 ucT = FeT("UICreator"), ctT = FeT("ContainerType");
            if (ucT == null || cvmT == null || pacT == null || ctT == null)
                throw new TzsError("internal", "SpecDesigner.FormEditor 里找不到 UICreator 家族（设计器版本变了？）");

            object ct = Activator.CreateInstance(ctT);
            Reflect.SetProp(ct, "Type", container);
            object cvm = Activator.CreateInstance(cvmT);
            Reflect.SetProp(cvm, "Container", ct);
            // Hardcoded inputs, not derived: they are dials on a layout engine whose output is
            // what gets spliced, so "improving" them would silently change every produced field.
            Reflect.SetProp(cvm, "MaximumWidthStrValue", "20");
            Reflect.SetProp(cvm, "NumberOfFieldsStrValue", "1");
            if (container == "ScrollGrid") {
                Reflect.SetProp(cvm, "RepeatRowCountStrValue", "2");
                Reflect.SetProp(cvm, "RepeatColumnCountStrValue", "1");
            }

            // One PrepareAddColumn per column, SKIPPING (not throwing on) a column with no widget
            // mapping. With a single column, refusing is the right answer -- but with 84 of them,
            // aborting the whole call because one column of type_t has no mapping would be worse
            // than the designer, which simply cannot drag that column either (AddField.cs:256-260).
            // The skips are reported, so nothing disappears quietly. Refusing only matters when
            // nothing is left to build.
            IList fields = (IList)Activator.CreateInstance(typeof(List<>).MakeGenericType(pacT));
            var built = new List<string>();
            var skipped = new List<string>();
            string firstWidget = null;
            foreach (string column in columns) {
                XElement colField = TableColumn("GetColField", table, column);
                string widget = widgetArg;
                if (string.IsNullOrEmpty(widget)) widget = Reflect.Attr(colField, "widget");
                if (string.IsNullOrEmpty(widget)) { skipped.Add(column); continue; }

                XElement colInfo = TableColumn("GetColumnInfo", table, column);
                string label = Reflect.Attr(colInfo, "text");
                if (string.IsNullOrEmpty(label)) label = column;

                object pac = Activator.CreateInstance(pacT);
                Reflect.SetProp(pac, "Table", table);
                Reflect.SetProp(pac, "Column", column);
                Reflect.SetProp(pac, "Description", label);
                Reflect.SetProp(pac, "Label", label);
                Reflect.SetProp(pac, "Widget", widget);
                Reflect.SetProp(pac, "Width", Reflect.Attr(colField, "widget_width"));
                fields.Add(pac);
                built.Add(column);
                if (firstWidget == null) firstWidget = widget;
            }
            if (fields.Count == 0)
                throw Refuse("没有一列可以变成控件：" + (skipped.Count == 0
                    ? "columns 是空的"
                    : string.Join(", ", skipped.ToArray())
                      + " 的 col_attr@widget 为空，设计器无法把它们变成控件"));

            string creator = container == "None" ? "CreateNoneContainerWidget" : "Create" + container + "Container";
            object producedRaw = Reflect.Call(ucT, creator, cvm, fields, s.Key);
            var produced = new List<object>();
            if (producedRaw is IEnumerable) foreach (object o in (IEnumerable)producedRaw) produced.Add(o);
            if (produced.Count == 0) throw new TzsError("internal", "UICreator." + creator + " 没有产出元素");

            // The label<->widget link is read off the element, not re-derived from names: None
            // mode binds, ScrollGrid's bare label does not, and inventing the binding the
            // designer never made is exactly the failure that is invisible until .bdx time.
            var tree = new List<object>();
            foreach (object e in produced) WalkDesigner(e, tree);
            var bindPairs = new List<string[]>();
            var seenPair = new HashSet<string>();
            foreach (object e in tree) {
                object be = Reflect.Prop(e, "BindElement");
                if (be == null) continue;
                object e1 = e, e2 = be;
                if (TagOf(e) != "Label" && TagOf(be) == "Label") { e1 = be; e2 = e; }
                string n1 = NameOf(e1), n2 = NameOf(e2);
                if (n1 == null || n2 == null) continue;
                string k1 = n1 + "|" + n2, k2 = n2 + "|" + n1;
                if (seenPair.Contains(k1) || seenPair.Contains(k2)) continue;
                seenPair.Add(k1);
                bindPairs.Add(new[] { n1, n2 });
            }

            // ---- placement (AddField.cs:316-345). Written onto the MODEL, because the model is
            // what the command will consult and what ToXML() will report.
            int row = w4.NextFreeRow(path);
            bool wrapMode = container != "None";
            foreach (object e in produced) {
                XElement xml = (XElement)Reflect.Call(e, "ToXML");
                XAttribute py = xml.Attribute("posY");
                if (py != null) {
                    int y = 1;
                    if (!wrapMode) int.TryParse(py.Value, out y);
                    Reflect.Call(e, "SetAttribute", "posY", (wrapMode ? row : y + row - 1).ToString());
                }
                if (wrapMode && xml.Attribute("posX") != null)
                    Reflect.Call(e, "SetAttribute", "posX", "1");
            }

            // AddComponetsUndoRedoCommand prepends (AddNodeAt(el, 0)) on every container type,
            // so handing it the list reversed is what leaves the produced order intact.
            var rev = new List<object>(produced);
            rev.Reverse();
            object cmd = NewCmd("AddComponetsUndoRedoCommand", XmlList(rev.ToArray()), parentEl, 0, 0);
            Run(s, cmd);

            var producedXml = new List<KeyValuePair<object, XElement>>();
            var names = new HashSet<string>();
            foreach (object e in produced) {
                producedXml.Add(new KeyValuePair<object, XElement>(e, (XElement)Reflect.Call(e, "ToXML")));
                foreach (object d in Subtree(e)) { string nm = NameOf(d); if (nm != null) names.Add(nm); }
            }

            ResyncChildren(w4, s, parentEl, path, producedXml);

            int promoted = PromoteByColName(s, names);
            int bound = 0;
            foreach (string[] p in bindPairs) {
                object fa = Reflect.Call(s.Si, "FindNodeByName", p[0]);
                object fb = Reflect.Call(s.Si, "FindNodeByName", p[1]);
                if (fa == null || fb == null) continue;
                Reflect.Call(s.Si, "AddSpecBinding", Reflect.Prop(fa, "GeneroComponent"),
                              Reflect.Prop(fb, "GeneroComponent"));
                bound++;
            }

            // Report the element the caller means by "the field": the container in wrap mode, the
            // last produced element (the widget, not its label) in None mode. `added` carries all
            // of them, because None mode legitimately makes two.
            object primary = produced[produced.Count - 1];
            string primaryPath = PathOf(s, w4, primary);
            if (primaryPath == null) throw new TzsError("internal", "新增元素没能定位回文本路径");
            var res = ElResult(s, primary, primaryPath);
            res["container"] = container;
            res["table"] = table;
            res["columns"] = Read.StrArray(built);
            // `column`/`widget` stay for the single-column caller that already reads them; with N
            // columns they have no single answer, and `columns` above is what to read.
            if (built.Count == 1) { res["column"] = built[0]; res["widget"] = firstWidget; }
            if (skipped.Count > 0) res["skippedColumns"] = Read.StrArray(skipped);
            res["promoted"] = promoted;
            res["bound"] = bound;
            var added = new JArray();
            foreach (object e in produced) {
                string ep = PathOf(s, w4, e);
                var one = new JObject();
                one["path"] = ep;
                one["name"] = NameOf(e);
                one["tag"] = TagOf(e);
                // `table`/`column` per node -- the caller's very next question is "which of these
                // bound to which column", and the answer is on the element itself: the same
                // sqlTabName / colName that find_component reports (Json.cs reads them at exactly
                // one place, and so does this line, so the two cannot drift). Absent on a node that
                // is not a data field -- a companion Label has neither, and that absence is
                // information too: it is not the thing you bind.
                //
                // 2026-09-25 的评测里，执行者在这一层看不到绑定信息，只能靠节点名猜或回读
                // （"引擎知道却没说出来" —— 与 §11.9 第 14、15 条同族）。
                string et = Session.Raw(e, "sqlTabName") as string;
                string ec = Session.Raw(e, "colName") as string;
                if (et != null) one["table"] = et;
                if (ec != null) one["column"] = ec;
                added.Add(one);
            }
            res["added"] = added;
            return res;
        }

        static XElement TableColumn(string method, string table, string column) {
            Type t = Reflect.Find(Designer.A, "TableColumnHelper");
            if (t == null) throw new TzsError("internal", "找不到 TableColumnHelper（设计器版本变了？）");
            return Reflect.Call(t, method, table, column) as XElement;
        }

        static void WalkDesigner(object e, List<object> acc) {
            acc.Add(e);
            foreach (object c in ModelChildren(e)) WalkDesigner(c, acc);
        }

        static IEnumerable<object> Subtree(object e) {
            var acc = new List<object>();
            WalkDesigner(e, acc);
            return acc;
        }

        // ================================================================ insert_at

        /// <summary>
        /// The designer's 往前/往后新增栏位. ManagedForm.ExecutedInsertBeforeWidgets /
        /// ExecutedInsertAfterWidgets are the same call -- the 3-arg
        /// AddComponetsUndoRedoCommand(list, parent, index) -- with the index taken off a
        /// selected element; here the caller names the index outright, which is what an AI can
        /// compute.
        ///
        /// The index the model ends up with is read back afterwards and the text placed to
        /// match: the 3-arg overload honours `index` only for HBox/VBox/Table/Tree/Folder
        /// containers (AddComponetsUndoRedoCommand.Execute's switch) and prepends for every other
        /// container type, so trusting the request would leave the model and the file in
        /// different orders.
        /// </summary>
        public static object InsertAt(Session s, JObject a) {
            string path = Read.Need(a, "path");
            object ctype = CType(Read.Need(a, "type"));
            string ctypeName = Reflect.S(ctype);

            object parentEl = Attr.El(s, path);
            FormWriter w4 = W(s);
            NeedW(w4, path);
            string parentTag = TagOf(parentEl);

            if (!Accepts(parentTag, ctypeName))
                throw Refuse(parentTag + " 不接受 " + ctypeName);

            int count = ModelChildren(parentEl).Count;
            int index = Read.IntArg(a, "index", count);
            if (index < 0 || index > count)
                throw new DetailedError("E_BAD_PARAM",
                    "index 越界: " + index + "；" + parentTag + " 现在有 " + count
                    + " 个直接子节点，合法范围是 0.." + count,
                    new JObject { { "param", "index" }, { "value", index },
                                  { "min", 0 }, { "max", count } });

            object el = Reflect.Call(Cft(), "CreateEmptyComponent", s.Key, ctype, "");
            if (el == null) throw new TzsError("internal", "CreateEmptyComponent 返回了 null");
            string newName = NameOf(el);
            if (IsExistingName(s, newName))
                throw Refuse("控件代号已存在: " + newName);

            // Same label rule as add_widget -- see the long note there.
            object label = null;
            string labelName = null;
            if (TzsCli.Designer.Fns.Session.Bool(a, "with_label", true)
                && Array.IndexOf(LABELED_TYPES, ctypeName) >= 0) {
                labelName = Reflect.S(Reflect.Call(Cft(), "GetNewName", newName + "_1", s.Key));
                label = Reflect.Call(Cft(), "CreateEmptyComponent", s.Key, CType("Label"), labelName);
                if (label == null) throw new TzsError("internal", "CreateEmptyComponent(Label) 返回了 null");
                labelName = NameOf(label) ?? labelName;
                Reflect.SetProp(label, "BindElement", el);
                Reflect.SetProp(el, "BindElement", label);
            }

            object cmd = NewCmd("AddComponetsUndoRedoCommand",
                label == null ? XmlList(el) : XmlList(el, label), parentEl, index);
            Run(s, cmd);

            object fsm = Reflect.Call(s.Si, "FindNodeByName", newName);
            int promoted = fsm == null ? 0 : PromoteSlots(fsm);
            if (label != null && fsm != null) {
                object lfsm = Reflect.Call(s.Si, "FindNodeByName", labelName);
                if (lfsm != null) {
                    Reflect.Call(s.Si, "AddSpecBinding", Reflect.Prop(lfsm, "GeneroComponent"),
                                  Reflect.Prop(fsm, "GeneroComponent"));
                    promoted += PromoteSlots(lfsm);
                }
            }

            string elPath = PathOf(s, w4, el);
            if (elPath == null) throw new TzsError("internal", "新增元素没能定位回文本路径");
            if (label != null)
                PlaceChild(w4, path, (XElement)Reflect.Call(label, "ToXML"), IndexOf(label));
            PlaceChild(w4, path, (XElement)Reflect.Call(el, "ToXML"), IndexOf(el));

            var res = ElResult(s, el, elPath);
            res["promoted"] = promoted;
            res["parent"] = parentTag;
            res["index"] = IndexOf(el);
            res["withLabel"] = label != null;
            if (label != null) {
                res["label"] = PathOf(s, w4, label);
                res["labelName"] = labelName;
            }
            return res;
        }

        // ================================================================ delete

        /// <summary>
        /// Delete layout elements; the spec side keeps a tombstone (status="d"), which is what
        /// makes a deleted field visible to the code generator instead of merely absent.
        ///
        /// Body of ManagedForm.ExecuteDelete -- the designer's own delete *is* the command (its
        /// toolbar item goes through ComponentHelper.DeleteSelection, and this handler is what
        /// that ends in). Two details are the designer's, not ours: the bound companion is pulled
        /// into the list AND ClearBinding runs on the original before the command is constructed
        /// (the .bdx is serialised from SpecBinding, so a binding left pointing at a removed
        /// control survives the delete); and when the whole child list of a Folder/Table/Tree is
        /// selected the command replaces it with the PARENT, which is why the text side has to
        /// check for that rather than assume the named paths went away.
        /// </summary>
        public static object DeleteFn(Session s, JObject a) {
            string[] paths = PathsArg(a, "delete");
            FormWriter w4 = W(s);
            var before = new List<Dictionary<string, string>>();
            List<object> els = ResolveAll(s, paths, out before);
            foreach (string p in paths) NeedW(w4, p);

            object parentEl = ParentOf(els[0]);
            if (parentEl == null) throw Refuse("根节点没有父节点，删不了");

            IList list = XmlList();
            var textPaths = new List<string>(paths);
            foreach (object el in els) {
                list.Add(el);
                object bind = Reflect.Prop(el, "BindElement");
                if (bind != null) {
                    list.Add(bind);
                    // The companion is deliberately not required to be among `paths`: the whole
                    // point is that deleting one half of a label/widget pair would orphan the
                    // other. Its own path is found by walking the model.
                    string bp = PathOf(s, w4, bind);
                    if (bp != null && !textPaths.Contains(bp)) textPaths.Add(bp);
                    Reflect.Call(el, "ClearBinding");
                }
            }

            // DeleteComponentsUndoRedoCommand's own Fold/Table/Tree rule, evaluated before the
            // command so the text side knows which element actually disappears.
            string parentTag = TagOf(parentEl);
            bool bundling = (parentTag == "Folder" || parentTag == "Table" || parentTag == "Tree")
                            && list.Count == ModelChildren(parentEl).Count;

            object cmd = NewCmd("DeleteComponentsUndoRedoCommand", parentEl, list);
            Run(s, cmd);

            var arr = new JArray();
            if (bundling) {
                // What disappears is the CONTAINER, not the children: the command put the
                // container into its own list and reparented to the grandparent, so its path is
                // the children's common parent path. PathOf cannot be used here -- the container
                // has already left the tree, so walking the living model cannot see it.
                string boxPath = null;
                foreach (string p in paths) {
                    string pp = ParentPath(p);
                    if (pp != null && w4.Index.ByPath(pp) != null) { boxPath = pp; break; }
                }
                if (boxPath == null) throw new TzsError("internal", "整块删除后找不到被删容器的路径");
                w4.RemoveNode(boxPath);
                var one = new JObject();
                one["path"] = boxPath;
                one["name"] = NameOf(parentEl);
                one["tag"] = parentTag;
                one["attrs"] = new JObject();
                one["removed"] = true;
                arr.Add(one);
            } else {
                // The named paths AND the companions the command pulled in. A `{attr: value}`
                // diff would be empty here -- deleting changes no attribute -- so the delta says
                // `removed` instead of pretending to be an attribute change.
                foreach (string p in textPaths) {
                    if (w4.Index.ByPath(p) != null) w4.RemoveNode(p);
                }
                for (int i = 0; i < els.Count; i++) {
                    var one = new JObject();
                    one["path"] = paths[i];
                    one["name"] = NameOf(els[i]);
                    one["tag"] = TagOf(els[i]);
                    one["removed"] = true;
                    one["attrs"] = new JObject();
                    arr.Add(one);
                }
            }

            var res = new JObject();
            res["count"] = bundling ? 1 : els.Count;
            // True means the designer's own rule fired: the child list of a Folder/Table/Tree was
            // fully selected, so the command deleted the CONTAINER instead of emptying it. A
            // caller that asked to delete every child gets a different form than it asked for and
            // must be told, not left to discover it.
            res["wholeContainerDeleted"] = bundling;
            res["delta"] = arr;
            return res;
        }

        // ================================================================ move

        static readonly string[] MOVE_TO = { "first", "prev", "next", "last" };

        /// <summary>
        /// Z order. One command covers all four verbs, which is why the manifest's `to` has no
        /// absolute-index member: ChangeChildIndexUndoRedoCommand(element, newIndex) is the whole
        /// mechanism, and first/prev/next/last are the arithmetic the designer's own four
        /// handlers do (ManagedForm.xaml.cs:851-898) -- 0, index-1, index+1, count-1.
        /// </summary>
        public static object MoveFn(Session s, JObject a) {
            string[] paths = PathsArg(a, "move");
            string to = Read.Need(a, "to");
            if (Array.IndexOf(MOVE_TO, to) < 0)
                throw TzsError.Validation("未知 to=" + to + "；合法值: " + string.Join(", ", MOVE_TO));

            FormWriter w4 = W(s);
            var before = new List<Dictionary<string, string>>();
            List<object> els = ResolveAll(s, paths, out before);
            foreach (string p in paths) NeedW(w4, p);

            object el = els[0];
            object parentEl = ParentOf(el);
            if (parentEl == null) throw Refuse("根节点没有父节点，改不了 Z 序");
            int count = ModelChildren(parentEl).Count;
            int cur = IndexOf(el);

            // The designer's own gate (canMoveInBox, ManagedForm.xaml.cs:906-930): reordering
            // exists only inside Folder/HBox/VBox/Table/Tree, and never in a box of one.
            string parentTag = TagOf(parentEl);
            bool boxy = parentTag == "Folder" || parentTag == "HBox" || parentTag == "VBox"
                        || parentTag == "Table" || parentTag == "Tree";
            if (!boxy)
                throw Refuse(parentTag + " 里的元素不能改 Z 序（设计器只在 Folder/HBox/VBox/Table/Tree 里提供这个操作）");
            if (count < 2)
                throw Refuse(parentTag + " 只有一个子节点，没有 Z 序可改");

            int index;
            switch (to) {
                case "first": index = 0; break;
                case "prev":  index = cur > 0 ? cur - 1 : 0; break;
                case "next":  index = cur + 1; break;
                default:      index = count - 1; break;   // last
            }
            if (index < 0 || index > count - 1)
                throw new DetailedError("E_BAD_PARAM",
                    "index 越界: " + index + "；合法范围 0.." + (count - 1),
                    new JObject { { "param", "to" }, { "computed", index },
                                  { "min", 0 }, { "max", count - 1 } });

            string path = paths[0];
            string parentPath = ParentPath(path);
            XElement xml = SpanXml(w4, path);
            if (xml == null) throw TzsError.NotFound("布局里的路径", path);
            w4.RemoveNode(path);

            object cmd = NewCmd("ChangeChildIndexUndoRedoCommand", el, index);
            Run(s, cmd);

            PlaceChild(w4, parentPath, xml, IndexOf(el));

            var res = new JObject();
            res["path"] = path;
            res["name"] = NameOf(el);
            res["tag"] = TagOf(el);
            res["from"] = cur;
            res["to"] = IndexOf(el);
            res["delta"] = new JArray();
            return res;
        }

        // ================================================================ nudge

        static readonly Dictionary<string, string> DIRECTIONS = new Dictionary<string, string> {
            { "up", "Up" }, { "down", "Down" }, { "left", "Left" }, { "right", "Right" },
        };

        /// <summary>
        /// Grid-cell translation. MoveComponentsUndoRedoCommand(list, direction, offset) adds or
        /// subtracts `offset` from posX/posY -- that is the whole body of its Execute -- so the
        /// layout side is an attribute diff, exactly like set_layout_attr.
        /// </summary>
        public static object Nudge(Session s, JObject a) {
            string[] paths = PathsArg(a, "nudge");
            string dir = Read.Need(a, "direction");
            string dirName;
            if (!DIRECTIONS.TryGetValue(dir, out dirName))
                throw TzsError.Validation("未知 direction=" + dir + "；合法值: up, down, left, right");
            int offset = Read.IntArg(a, "offset", 1);
            if (offset <= 0) throw TzsError.Validation("offset 必须为正数，收到 " + offset);

            FormWriter w4 = W(s);
            var before = new List<Dictionary<string, string>>();
            List<object> els = ResolveAll(s, paths, out before);
            foreach (string p in paths) NeedW(w4, p);

            object cmd = NewCmd("MoveComponentsUndoRedoCommand", XmlList(els.ToArray()),
                                EnumT("MoveDirection", dirName, "direction",
                                      new[] { "up", "down", "left", "right" }), offset);
            Run(s, cmd);

            var arr = new JArray();
            for (int i = 0; i < els.Count; i++) arr.Add(PatchOne(w4, paths[i], els[i], before[i]));
            var res = new JObject();
            res["count"] = els.Count;
            res["direction"] = dir;
            res["offset"] = offset;
            res["delta"] = arr;
            return res;
        }

        // ================================================================ align

        static readonly string[] ALIGN_OPTS = { "stretch", "left", "right", "top", "bottom" };
        static readonly Dictionary<string, string> ALIGN = new Dictionary<string, string> {
            { "stretch", "STRETCH" }, { "left", "LEFT" }, { "right", "RIGHT" },
            { "top", "TOP" }, { "bottom", "BOTTOM" },
        };

        /// <summary>
        /// AlignUndoRedoCommand(elements, AlignOptions). STRETCH rewrites posX/gridWidth for every
        /// element to the union of their extents; LEFT/RIGHT move posX, TOP/BOTTOM move posY.
        /// The designer's CanExecuteAlignWidgets requires more than one selected element, which is
        /// not a formality: with one element LEFT is a no-op that reads like success.
        /// </summary>
        public static object AlignFn(Session s, JObject a) {
            string[] paths = PathsArg(a, "align");
            if (paths.Length < 2) throw TzsError.Validation("align 至少需要 2 个元素（设计器的 CanExecuteAlignWidgets 同样要求）");
            string opt = Read.Need(a, "option");
            string optName;
            if (!ALIGN.TryGetValue(opt, out optName))
                throw TzsError.Validation("未知 option=" + opt + "；合法值: " + string.Join(", ", ALIGN_OPTS));

            FormWriter w4 = W(s);
            var before = new List<Dictionary<string, string>>();
            List<object> els = ResolveAll(s, paths, out before);
            foreach (string p in paths) NeedW(w4, p);

            object cmd = NewCmd("AlignUndoRedoCommand", XmlList(els.ToArray()),
                                EnumT("AlignOptions", optName, "option", ALIGN_OPTS));
            Run(s, cmd);

            var arr = new JArray();
            for (int i = 0; i < els.Count; i++) arr.Add(PatchOne(w4, paths[i], els[i], before[i]));
            var res = new JObject();
            res["count"] = els.Count;
            res["option"] = opt;
            res["delta"] = arr;
            return res;
        }

        // ================================================================ fit_size

        /// <summary>
        /// ChangeSizeUndoRedoCommand(elements).
        ///
        /// The designer has NO fit-size feature: the command's only call site is the resize
        /// adorner (ResizeControl.xaml.cs:58), which drives it with a dragged rectangle through
        /// SetFinalSize. Constructing it and executing it alone is not a smaller version of that
        /// -- finX/finY/finWidth/finHeight default to 0, so every selected element would move to
        /// the origin and collapse to zero. The final size has to come from the caller, and the
        /// only size that is self-consistent without a UI is the element's own floor:
        /// XmlElement.GridWidth/GridHeight clamp with Math.Max(MinGrid*, value), so
        /// MinGridWidth/MinGridHeight are the smallest legal size, and 自适应 as a headless
        /// operation can only mean shrinking to it.
        ///
        /// Reported as a delta, so the caller sees exactly what the clamp produced and can tell a
        /// real resize from a no-op.
        /// </summary>
        public static object FitSize(Session s, JObject a) {
            string[] paths = PathsArg(a, "fit_size");

            FormWriter w4 = W(s);
            var before = new List<Dictionary<string, string>>();
            List<object> els = ResolveAll(s, paths, out before);
            foreach (string p in paths) NeedW(w4, p);

            object cmd = NewCmd("ChangeSizeUndoRedoCommand", XmlList(els.ToArray()));

            // SetFinalSize must be called for EVERY element the command holds: the ones left
            // alone would be driven to (0,0,0,0) by Execute. Reading MinGrid* rather than
            // computing a size is the point -- it is the designer's own floor.
            foreach (object el in els) {
                Reflect.Call(cmd, "SetFinalSize", el,
                    IntOf(Reflect.Prop(el, "GridX")), IntOf(Reflect.Prop(el, "GridY")),
                    IntOf(Reflect.Prop(el, "MinGridWidth")), IntOf(Reflect.Prop(el, "MinGridHeight")));
            }
            Run(s, cmd);

            var arr = new JArray();
            for (int i = 0; i < els.Count; i++) arr.Add(PatchOne(w4, paths[i], els[i], before[i]));
            var res = new JObject();
            res["count"] = els.Count;
            res["delta"] = arr;
            return res;
        }

        // ================================================================ wrap

        /// <summary>
        /// Select elements, put a new container around them. This is how the designer makes an
        /// HBox or a VBox at all -- they are not in the widget box -- so it is a layout refactor
        /// over EXISTING elements, not a way to add fields.
        ///
        /// The signature is one line but the surrounding rules are the operation:
        ///
        ///   - every element must share one parent. The command's Undo puts them all back into
        ///     one container, so a cross-parent wrap is not something it can express.
        ///   - the parent must not be the Form (the designer's CanLayoutCommand refuses the same).
        ///   - the designer's mime gate, both directions: the new container type must accept each
        ///     element, and the parent must accept the container type.
        ///
        /// The HBox/VBox half of that gate is load-bearing, and is this module's negative
        /// control: the mime table (core-br.spec, modFD/HBox) lists Grid/Group/ScrollGrid/Table/
        /// Tree/Folder/HBox/VBox and nothing else, so wrapping a plain widget into an HBox is
        /// refused by the designer's own table, quoted here rather than re-derived.
        /// </summary>
        public static object Wrap(Session s, JObject a) {
            string[] paths = PathsArg(a, "wrap");
            object boxType = CType(Read.Need(a, "type"));
            string boxName = Reflect.S(boxType);

            FormWriter w4 = W(s);
            var els = new List<object>();
            foreach (string p in paths) { els.Add(Attr.El(s, p)); NeedW(w4, p); }

            string parentPath = null;
            foreach (string p in paths) {
                string pp = ParentPath(p);
                if (parentPath == null) parentPath = pp;
                else if (parentPath != pp)
                    throw Refuse("待包裹的元素必须同属一个父节点：\n     " + parentPath + "\n     " + pp);
            }
            if (parentPath == null) throw Refuse("根节点不能作为包裹的目标父节点");

            object parentEl = ParentOf(els[0]);
            if (parentEl == null) throw Refuse("根节点没有父节点");
            string parentTag = TagOf(parentEl);
            if (parentTag == "Form")
                throw Refuse("不能把根布局包起来（设计器的 CanLayoutCommand 同样禁止）");
            if (w4.Index.ByPath(parentPath) == null)
                throw TzsError.NotFound("布局里的路径", parentPath);

            foreach (object el in els) {
                if (!Accepts(boxType, el))
                    throw Refuse("<" + TagOf(el) + "> " + (NameOf(el) ?? "") + " 不能被 " + boxName + " 接受"
                        + "（" + boxName + " 的 mime 表只收容器，不收控件——这就是 wrap 的用途边界）");
            }
            if (boxName != "HBox" && boxName != "VBox" && !Accepts(parentEl, boxType))
                throw Refuse("父容器 " + parentTag + " 不能接受 " + boxName);

            // Snapshot before the command: the wrapped elements leave the text, and their own
            // bytes are what should go back under the new box.
            var moved = new List<KeyValuePair<object, XElement>>();
            foreach (string p in paths) {
                XElement xe = SpanXml(w4, p);
                if (xe == null) throw TzsError.NotFound("布局里的路径", p);
                NoTextNodes(xe);
                moved.Add(new KeyValuePair<object, XElement>(Attr.El(s, p), xe));
            }

            object cmd = NewCmd("AddToContainerUndoRedoCommand", XmlList(els.ToArray()), parentEl, boxType);
            Run(s, cmd);

            object box = Reflect.Prop(cmd, "_newContainer");
            if (box == null) throw new TzsError("internal", "AddToContainerUndoRedoCommand 没有产出新容器");

            // The command already called SpecificationInfo.Add(_newContainer), so the spec side is
            // complete; only the layout text needs the move. The new box is not in the text yet,
            // so it goes in through `resolved`.
            var resolved = new List<KeyValuePair<object, XElement>>(moved);
            resolved.Add(new KeyValuePair<object, XElement>(box, (XElement)Reflect.Call(box, "ToXML")));
            ResyncChildren(w4, s, parentEl, parentPath, resolved);

            string boxPath = PathOf(s, w4, box);
            if (boxPath == null) throw new TzsError("internal", "新容器没能定位回文本路径");
            var res = ElResult(s, box, boxPath);
            res["wrapped"] = els.Count;
            res["parent"] = parentTag;
            return res;
        }

        // ================================================================ reparent

        /// <summary>
        /// Moves elements into another container in the SAME form -- the designer's drag-drop
        /// command, DragComponentsUndoRedoCommand.
        ///
        /// That command is the reparent primitive the surface was missing, and the gap was real:
        /// without it "put this Table into that page" has no correct implementation. Rebuilding the
        /// element inside the destination is NOT a substitute -- measured on apmt500_wf, rebuilt
        /// fields came back suffixed (pmdoent -> pmdoent_1; 0 of 84 names reused) because
        /// GetCurrentNewName cannot reuse a live name, and those names ARE the spec dictionary's
        /// binding keys, so the form ended up with two fields per column instead of a move.
        /// RoundTrip reports a clean fixed point either way and could not have caught it.
        ///
        /// The command is built with the SOURCE container, AppendSelection() takes each element and
        /// SetDestination() names the target; XmlElement.AddNodeAt does the actual move, detaching
        /// from the old parent and reattaching to the new one (XmlElement.cs:1480-1494). So the
        /// MODEL needs no help from us -- only the layout text does, as everywhere in this file:
        /// each element's own bytes are captured before the command and re-spliced afterwards.
        ///
        /// The index overload of SetDestination is used, which leaves GridX/GridY to the command;
        /// the moved elements are then parked at the destination's top-left, the only placement
        /// that does not depend on where they used to live.
        ///
        /// SAME FORM ONLY. Cross-form would be cut+paste, and that empties the source form.
        /// </summary>
        public static object ReparentFn(Session s, JObject a) {
            string[] paths = Read.ListArg(a, "paths");
            string into = Read.Need(a, "into");
            if (paths == null || paths.Length == 0)
                throw TzsError.Validation("reparent 需要 paths（要搬走的元素的 name-path 数组）");

            FormWriter w4 = W(s);
            NeedW(w4, into);
            object dst = Attr.El(s, into);
            string dstTag = TagOf(dst);
            if (!IsContainer(dstTag))
                throw Refuse("into= 指向的 \"" + (NameOf(dst) ?? into) + "\" 是 " + dstTag
                    + "，不是容器，接不了子节点");

            // Our own request-shape constraint is checked BEFORE the designer's acceptance rule.
            // "paths must share one parent" is a limit of DragComponentsUndoRedoCommand (it takes
            // one source container), and reporting it late means the caller gets a message about
            // the TARGET when the real problem is on the source side -- which is exactly what
            // happened while testing this: a Grid rejected the second element's HBox type before
            // the same-parent check ever ran.
            var req = new List<KeyValuePair<string, object>>();
            foreach (string p in paths) {
                object el = Attr.El(s, p);
                string nm = NameOf(el) ?? p;
                if (ParentOf(el) == null)
                    throw Refuse("paths 里的 \"" + nm + "\" 是表单根，搬不动");
                // A container cannot go inside itself or its own descendant: the move would
                // detach the subtree the destination itself lives in.
                for (object c = dst; c != null; c = ParentOf(c))
                    if (object.ReferenceEquals(c, el))
                        throw Refuse("不能把 \"" + nm + "\" 搬进它自己或它的子孙里（into=" + into + "）");
                req.Add(new KeyValuePair<string, object>(p, el));
            }

            object src = ParentOf(req[0].Value);
            foreach (KeyValuePair<string, object> q in req)
                if (!object.ReferenceEquals(ParentOf(q.Value), src))
                    throw Refuse("paths 里的 \"" + (NameOf(q.Value) ?? q.Key)
                        + "\" 和第一个元素不在同一个容器下；"
                        + "DragComponentsUndoRedoCommand 只能搬一个源容器里的元素，"
                        + "要跨容器请分多次调用");

            var els = new List<object>();
            var spans = new List<KeyValuePair<string, XElement>>();
            foreach (KeyValuePair<string, object> q in req) {
                if (!Accepts(dst, q.Value))
                    throw Refuse("into= 的 " + dstTag + " 不接受 \"" + (NameOf(q.Value) ?? q.Key)
                        + "\"（" + TagOf(q.Value) + "）——ComponentFactory.AcceptMimes 拒绝");
                XElement xe = SpanXml(w4, q.Key);
                if (xe == null) throw TzsError.NotFound("布局里的路径", q.Key);
                NoTextNodes(xe);
                spans.Add(new KeyValuePair<string, XElement>(q.Key, xe));
                els.Add(q.Value);
            }

            object cmd = NewCmd("DragComponentsUndoRedoCommand", src);
            foreach (object el in els) Reflect.Call(cmd, "AppendSelection", el, -1);
            Reflect.Call(cmd, "SetDestination", dst, 0);
            Run(s, cmd);

            // The text half. The spans were captured BEFORE the command, because the model has
            // already moved by now; the names do not change, so RemoveNode still finds them.
            foreach (KeyValuePair<string, XElement> m in spans) w4.RemoveNode(m.Key);
            foreach (KeyValuePair<string, XElement> m in spans) w4.AddNode(into, m.Value);
            foreach (object el in els) {
                Reflect.Call(el, "SetAttribute", "posX", "1");
                Reflect.Call(el, "SetAttribute", "posY", "1");
            }

            var arr = new JArray();
            foreach (object el in els) {
                var o = new JObject();
                o["name"] = NameOf(el);
                o["tag"] = TagOf(el);
                o["path"] = PathOf(s, w4, el);
                arr.Add(o);
            }
            var res = new JObject();
            res["into"] = into;
            res["containerTag"] = dstTag;
            res["count"] = els.Count;
            res["moved"] = arr;
            res["note"] = "同表单内移动（DragComponentsUndoRedoCommand）：元素本体就是原来那个——"
                + "名字、规格节点、.tsd 都不变，所以绑定不会断。坐标被停到目标的左上角（posX=posY=1）";
            return res;
        }

        // ================================================================ break_layout

        /// <summary>
        /// The inverse of wrap: dissolve an HBox/VBox and hand its children to the container.
        ///
        /// BreakLayoutUndoRedoCommand.Execute recomputes each child's position along the box's
        /// axis and appends them to `_container`, and drops the box's spec node
        /// (SpecificationInfo.Remove). The designer's CanBreakLayout adds two gates the command
        /// does not: the element must be an HBox or a VBox, and its parent must not be the Form
        /// nor another HBox/VBox.
        /// </summary>
        public static object BreakLayoutFn(Session s, JObject a) {
            string path = Read.Need(a, "path");
            FormWriter w4 = W(s);
            NeedW(w4, path);
            object box = Attr.El(s, path);

            string boxTag = TagOf(box);
            if (boxTag != "HBox" && boxTag != "VBox")
                throw Refuse("break_layout 只能拆 HBox/VBox，收到 <" + boxTag + ">");

            object parentEl = ParentOf(box);
            if (parentEl == null) throw Refuse("根节点没有父节点，拆不了");
            string parentTag = TagOf(parentEl);
            if (parentTag == "Form" || parentTag == "HBox" || parentTag == "VBox")
                throw Refuse("父容器是 " + parentTag + "，设计器不允许在这里拆箱");

            string parentPath = ParentPath(path);

            // The box and its children all leave the text; the children come back under the
            // parent, so their own bytes are what has to be re-spliced.
            var moved = new List<KeyValuePair<object, XElement>>();
            var kids = new List<object>();
            ElementSpan boxSpan = w4.Index.ByPath(path);
            if (boxSpan != null) {
                foreach (ElementSpan c in boxSpan.Children) {
                    object k = s.FindByPath(c.Path);
                    XElement xe = SpanXml(w4, c.Path);
                    if (k == null || xe == null) continue;
                    NoTextNodes(xe);
                    kids.Add(k);
                    moved.Add(new KeyValuePair<object, XElement>(k, xe));
                }
            }

            var before = new List<Dictionary<string, string>>();
            foreach (object k in kids) before.Add(Session.Attrs(k));

            object cmd = NewCmd("BreakLayoutUndoRedoCommand", box);
            Run(s, cmd);

            ResyncChildren(w4, s, parentEl, parentPath, moved);

            var arr = new JArray();
            for (int i = 0; i < kids.Count; i++) {
                string kp = PathOf(s, w4, kids[i]);
                if (kp == null) continue;
                arr.Add(PatchOne(w4, kp, kids[i], before[i]));
            }
            var res = new JObject();
            res["path"] = path;
            res["name"] = NameOf(box);
            res["tag"] = boxTag;
            res["count"] = kids.Count;
            res["delta"] = arr;
            return res;
        }

        // ================================================================ convert_*

        /// <summary>The 14 target types the designer's context menu offers
        /// (SpecDesigner.FormEditor/Themes/Generic.xaml:393-558, FormCommands.ConvertWidgetCommand).
        /// Kept as a list rather than as a second enum because the designer does not have one -- it
        /// is XAML. Button and Label are in the manifest's WIDGETS but NOT here, which is the
        /// point: `type` is validated by the transport against the widget catalogue, and this list
        /// is the narrower set the conversion actually supports.</summary>
        static readonly string[] CONVERT_WIDGET_TARGETS = {
            "ButtonEdit", "CheckBox", "ComboBox", "DateEdit", "Edit", "FFImage", "FFLabel",
            "ProgressBar", "RadioGroup", "Slider", "SpinEdit", "TextEdit", "TimeEdit",
            "DateTimeEdit",
        };

        /// <summary>
        /// ConvertWidgetTypeUndoRedoCommand(element, type). ComponentFactory.ConvertTo builds a
        /// fresh element of the target type with the same name and copies over every attribute the
        /// new type has (except `widget`), and the command hands the spec side to
        /// SpecificationInfo.ConvertComponentType.
        ///
        /// The gates are the designer's CanExecuteConvertToWidget + CanExecuteConvertWidget: the
        /// element must be a form field and not a container, its spec type must not be one of
        /// PROGREL/NONE/REFERENCE/MULTILANG (those carry a fixed widget by construction), and the
        /// target must differ from the current type.
        /// </summary>
        public static object ConvertWidget(Session s, JObject a) {
            string path = Read.Need(a, "path");
            object ctype = CType(Read.Need(a, "type"));
            string target = Reflect.S(ctype);

            if (Array.IndexOf(CONVERT_WIDGET_TARGETS, target) < 0)
                throw new DetailedError("E_DESIGNER",
                    "convert_widget 不支持转到 " + target + "；设计器的转换菜单只有这 14 种: "
                    + string.Join(", ", CONVERT_WIDGET_TARGETS),
                    new JObject { { "param", "type" }, { "value", target },
                                  { "legal", Read.StrArray(CONVERT_WIDGET_TARGETS) } });

            FormWriter w4 = W(s);
            NeedW(w4, path);
            object el = Attr.El(s, path);
            string tag = TagOf(el);

            if (IsContainer(tag))
                throw Refuse("<" + tag + "> 是容器，只能用 convert_container"
                    + "（设计器的 CanExecuteConvertToWidget 同样拒绝）");
            if (!IsFormField(tag))
                throw Refuse("<" + tag + "> 不是控件箱里的控件类型，不能转换");
            if (Reflect.S(CTypeOf(el)) == target)
                throw Refuse("<" + tag + "> 已经是 " + target + "（设计器的菜单项对当前类型置灰）");

            string st = Reflect.S(Reflect.Call(s.Si, "GetSpecType", el));
            if (st == "PROGREL" || st == "NONE" || st == "REFERENCE" || st == "MULTILANG")
                throw Refuse("<" + tag + "> 的规格类型是 " + st + "，它的控件类型由规格决定，不能改");

            return ConvertCommon(s, w4, path, el, ctype, "ConvertWidgetTypeUndoRedoCommand");
        }

        /// <summary>ConvertContainerTypeUndoRedoCommand(element, type), whose gate the designer
        /// states twice over: the context menu offers only Grid and Group
        /// (FormCommands.ConvertToContainerCommand) AND CanExecuteConvertToContainer requires the
        /// element to already BE a Grid or a Group -- you convert one container type into the
        /// other, you do not turn a widget into a container this way.</summary>
        internal static readonly string[] CONVERT_CONTAINER_TARGETS = { "Grid", "Group" };

        public static object ConvertContainer(Session s, JObject a) {
            string path = Read.Need(a, "path");
            object ctype = CType(Read.Need(a, "type"));
            string target = Reflect.S(ctype);

            if (Array.IndexOf(CONVERT_CONTAINER_TARGETS, target) < 0)
                throw new DetailedError("E_DESIGNER",
                    "convert_container 只能转成 Grid 或 Group；设计器的转换菜单只有这两种",
                    new JObject { { "param", "type" }, { "value", target },
                                  { "legal", Read.StrArray(CONVERT_CONTAINER_TARGETS) } });

            FormWriter w4 = W(s);
            NeedW(w4, path);
            object el = Attr.El(s, path);
            string tag = TagOf(el);
            if (tag != "Grid" && tag != "Group")
                throw Refuse("<" + tag + "> 不是 Grid/Group，不能做容器类型转换"
                    + "（设计器的 CanExecuteConvertToContainer 要求源元素本身就是容器）");
            if (Reflect.S(CTypeOf(el)) == target)
                throw Refuse("<" + tag + "> 已经是 " + target);

            return ConvertCommon(s, w4, path, el, ctype, "ConvertContainerTypeUndoRedoCommand");
        }

        /// <summary>Both converts replace the element in place: ComponentFactory.ConvertTo keeps
        /// the name, and the old object's slot is what the new one takes. The text side therefore
        /// removes the old lines and re-splices the new element at the SAME index -- read off the
        /// text, where the slot is unambiguous -- rather than at whatever index the command
        /// happened to use (ConvertContainerTypeUndoRedoCommand only preserves the index for
        /// HBox/VBox parents and prepends otherwise, which for a Grid/Group element is almost
        /// never where it came from).</summary>
        static object ConvertCommon(Session s, FormWriter w4, string path, object el,
                                    object ctype, string cmdName) {
            string parentPath = ParentPath(path);
            if (parentPath == null) throw Refuse("根节点不能转换");
            ElementSpan sp = w4.Index.ByPath(path);
            int index = sp == null ? 0 : sp.Parent.Children.IndexOf(sp);
            string oldTag = TagOf(el);

            object cmd = NewCmd(cmdName, el, ctype);
            Run(s, cmd);

            // The command swapped the model object; ComponentFactory.ConvertTo preserves the name,
            // so the replacement is whatever FormSpeDictionary now holds under it.
            object twin = Reflect.Call(s.Si, "FindNodeByName", NameOf(el));
            object replaced = twin == null ? null : Reflect.Prop(twin, "GeneroComponent");
            if (replaced == null) throw new TzsError("internal", "转换后找不到新元素（" + cmdName + "）");

            w4.RemoveNode(path);
            PlaceChild(w4, parentPath, (XElement)Reflect.Call(replaced, "ToXML"), index);

            string newPath = PathOf(s, w4, replaced);
            if (newPath == null) throw new TzsError("internal", "转换后的元素没能定位回文本路径");

            var res = ElResult(s, replaced, newPath);
            res["from"] = oldTag;
            res["to"] = TagOf(replaced);
            return res;
        }
    }
}
