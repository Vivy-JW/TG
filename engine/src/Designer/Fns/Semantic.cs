using System;
using System.Collections;
using System.Collections.Generic;
using System.Xml.Linq;
using Newtonsoft.Json.Linq;
using TzsCli;

namespace TzsCli.Designer
{
    // Deliberately NOT TzsCli.Designer.Fns like the other modules in this directory: Fns/Session.cs
    // declares a static module class also called Session, and a type in the same namespace outranks
    // an enclosing one, so inside .Fns every `Session s` parameter here would be CS0721. Read.cs and
    // Attr.cs sit in this namespace for the same reason. Rpc.BuildMap dispatches by class name, so
    // the split costs nothing.

    /// <summary>
    /// The semantic writes: the things a form's .tsd carries that are neither a spec attribute
    /// (Attr.cs) nor a structural edit -- the local strings, the ComboBox / RadioGroup option list,
    /// the 串查 program list, the &lt;table&gt; association section, the SD 规格描述 CDATA, the
    /// 引用标准 flag, and the cross-form copy.
    ///
    /// WHAT MAKES THIS FILE DIFFERENT FROM Attr.cs
    ///
    /// Attr.cs writes ONE attribute of ONE node through the designer's own command and reports the
    /// delta. These functions each drive a *designer workflow* -- an editor window, a grid's
    /// add/delete buttons, a combo box's SelectionChanged -- as one call. So each reproduces the
    /// workflow's GUARDS as well as its writes, and the guards are the load-bearing half:
    ///
    ///   * <c>XmlElement.ReplaceItems</c> returns silently unless the type is RadioGroup or ComboBox
    ///     (XmlElement.cs:2190-2196). Called on an Edit it is a no-op that looks exactly like
    ///     success.
    ///   * <c>ProgRelProgramsDataGrid.xaml:20-34</c> disables the whole 串查 panel when the bound
    ///     CiteStd is "Y". That binding is to a property neither SpecProgRelNode nor
    ///     AbstractSpecNode has, so the XAML gate is dead code -- but its intent is unambiguous and
    ///     refusing is the safe direction, so it is enforced on the attribute the binding means
    ///     (cite_std on the pfield node). See SetProgrelPrograms.
    ///   * <c>SpecPropertyEditor.SRAttributesCB_Click</c> (the INSERT/DELETE/APPEND ROW boxes)
    ///     returns early when the &lt;sr&gt; row is absent (:836-839) -- it edits the row, it does
    ///     not create it.
    ///   * <c>AbstractSpecNode.CDATA</c>'s setter is a silent no-op when IsCited (:226).
    ///
    /// Every one of those becomes an error carrying a machine-readable <c>detail.reason</c>, because
    /// the whole point of this layer is that "the designer ignored you" and "it worked" must not
    /// look alike. E_DESIGNER (wire kind `designer`: do not retry) for designer rules, E_NOT_FOUND /
    /// E_NO_SPEC_NODE where that is the honest classification -- see Rpc.Map for how each reaches the
    /// wire.
    ///
    /// The one case that is NOT an error is the designer's *other* silent path: old == new. That is
    /// the form already holding the requested state, which SPEC §11.24 (a) classifies as a success
    /// and which lands in `result` as `noop:true` + `code:"E_NO_OP"` (see NoOp below) -- never in
    /// `error`, where kind `internal` would tell the caller to report a bug instead of stopping.
    ///
    /// The .tsd side of every change is rendered by the designer itself: `save` calls SaveToTSD,
    /// which rebuilds &lt;strings&gt;, &lt;table&gt;, &lt;prog_rel&gt; and the per-node CDATA from the
    /// live model. So none of these functions writes .tsd text; only set_items, which also moves
    /// bytes in the .4fd (its &lt;Item&gt; children), touches the per-handle FormWriter -- because
    /// FormWriter is the only thing `save` renders for the .4fd
    /// (Fns/Session.Layout), and a model edit it never sees is a model edit that never lands.
    /// </summary>
    public static class Semantic
    {
        public static void Register(IDictionary<string, Fn> into) {
            into["set_local_string"]      = SetLocalString;
            into["set_items"]             = SetItems;
            into["set_progrel_programs"]  = SetProgrelPrograms;
            into["set_table_association"] = SetTableAssociation;
            into["set_spec_description"]  = SetSpecDescription;
            into["set_cited"]             = SetCited;
        }

        // ================================================================ errors

        /// <summary>
        /// A designer rule refused the call, carrying a machine-readable reason. Each of these
        /// functions has several distinct designer rules, and "which one fired" is not recoverable
        /// from prose, hence `detail.reason`.
        ///
        /// The TzsError *code* is "designer", not "E_DESIGNER": Rpc.Map keys on the code and turns the
        /// four classification keys (validation / not_found / designer / internal) into the wire code
        /// AND the recoverability kind of SPEC §11.24 (a). Passing the wire code through instead lands
        /// in Map's default arm, which reports kind `internal` -- "report a bug, do not retry" -- for
        /// what is a fact about the form. Errors.cs and TzsError.Designer both mean the classification
        /// key, and the wire code the caller sees is E_DESIGNER either way. Same for NotFound below.
        /// </summary>
        static DetailedError Refused(string reason, string message, JObject detail) {
            if (detail == null) detail = new JObject();
            detail["reason"] = reason;
            return new DetailedError("designer", message, detail);
        }

        static DetailedError NotFound(string what, string value, JObject detail) {
            if (detail == null) detail = new JObject();
            detail["reason"] = "not_found";
            detail["what"] = what;
            detail["value"] = value;
            return new DetailedError("not_found", what + " 不存在: " + value, detail);
        }

        /// <summary>
        /// The "already this value" answer, as a SUCCESS frame carrying `noop:true` and
        /// `code:"E_NO_OP"`. SPEC §11.24 (a) makes a no-op an OUTCOME, not a failure, and the three
        /// outcomes carry exactly one positive marker each (applied / clamped / noop).
        ///
        /// Both callers below used to THROW this. Rpc.Map classifies an E_* code it does not know as
        /// `kind:internal`, whose contract meaning is "unexpected exception -- report it, do not
        /// retry": the one answer that makes a caller abandon a request the form had already
        /// satisfied. Attr.cs and Action.cs have returned the success shape since the mid-wave
        /// amendment (commit 0c607df); these two were the last holdouts.
        ///
        /// The payload is the same shape a real write would have produced (path / name / value /
        /// after / target / cdata …), so the caller reads one vocabulary either way -- `after` is
        /// read back from the model, which is what makes "already this" a fact and not an assumption.
        /// </summary>
        static JObject NoOp(JObject res, string note) {
            if (res == null) res = new JObject();
            res["changed"] = false;
            res["noop"] = true;
            res["code"] = "E_NO_OP";
            if (note != null) res["note"] = note;
            return res;
        }

        static JObject Detail(string key, object value) {
            var d = new JObject();
            if (key != null) d[key] = value == null ? (JToken)JValue.CreateNull() : JToken.FromObject(value);
            return d;
        }

        // ================================================================ plumbing

        static bool BoolArg(JObject a, string n, bool dflt) {
            JToken t = a == null ? null : a[n];
            if (t == null || t.Type == JTokenType.Null) return dflt;
            if (t.Type == JTokenType.Boolean) return (bool)t;
            bool b;
            return bool.TryParse(t.ToString(), out b) ? b : dflt;
        }

        /// <summary>ComponentType of a form element as the string the .tsd / &lt;sr kind=…&gt; use.</summary>
        static string TypeName(object el) {
            object t = el == null ? null : Reflect.Prop(el, "Type");
            return t == null ? null : t.ToString();
        }

        static object Si(Session s) {
            if (s == null || s.Si == null) throw TzsError.Validation("缺少 handle（需要已打开的会话）");
            return s.Si;
        }

        static string Env(Session s) { return Read.Str(Reflect.Prop(s.Si, "Env")); }

        /// <summary>One attribute of a spec node's Source, or null when absent. Absent and empty are
        /// different facts here: the designer compares against "d"/"u"/"N" and a missing attribute is
        /// not one of them.</summary>
        static string NodeAttr(object node, string attr) {
            return Reflect.Attr(Reflect.Prop(node, "Source") as XElement, attr);
        }

        static IEnumerable EnumOf(object o) { return o as IEnumerable ?? new object[0]; }

        /// <summary>A typed List&lt;designer XmlElement&gt;. Activator cannot bind a List&lt;object&gt;
        /// to the parameter types the designer declares, so the list is minted at the element type
        /// the live model actually uses (same trick as Attr.Batch).</summary>
        static IList TypedList(Type elemType) {
            return (IList)Activator.CreateInstance(typeof(List<>).MakeGenericType(elemType));
        }

        static bool AsBool(object o) { return o is bool && (bool)o; }

        /// <summary>XmlElement.ToXML() -- the element as the .4fd serialises it. It is what
        /// set_items hands to FormWriter, so that the written text and any later
        /// regeneration are the same bytes rather than two independently-derived attribute lists.</summary>
        static XElement ToXmlOf(object el) {
            XElement xe = Reflect.Call(el, "ToXML") as XElement;
            if (xe == null) throw new Exception("XmlElement.ToXML() 返回 null（设计器版本变了？）");
            return xe;
        }

        // ================================================================ set_local_string

        /// <summary>
        /// One &lt;sfield&gt; / &lt;sact&gt; local string: the `lbl_*` / `cmt_*` texts, which live in
        /// neither the layout tree nor FormSpeDictionary and so are unreachable through
        /// set_spec_attr.
        ///
        /// THE THREE DESIGNER APIS behind one entry point:
        ///
        ///   SetFieldLocalStringText(name, text)  creates the node if missing, then sets `text`
        ///                                        (SpecificationInfo.cs:2341)
        ///   SetActLocalStringText(name, text)    the same for &lt;sact&gt; (:2274)
        ///   DeleteFieldLocalStringText(name)     does NOT remove the node; it sets
        ///                                        Status |= DELETE, which SetStatusToSource renders
        ///                                        as lstr="d" (:2361)
        ///
        /// The tombstone is the point: SPEC §6.3 says deletion is soft throughout this format, and
        /// the picker (SpecFieldTextControl.DeleteButtonClicked, :86-89) implements it exactly that
        /// way. Removing the node would make "deleted" indistinguishable from "never had it", and
        /// .tsd regeneration would drop it -- so it stays and carries lstr="d".
        ///
        /// THE MANIFEST HAS NO DELETE FLAG, so the two directions are split by `text`:
        ///   text != ""  -> write (Set*LocalStringText)
        ///   text == ""  -> delete (Delete*LocalStringText, i.e. tombstone)
        /// That is the only reading available without inventing a 5th parameter (Manifest.Check
        /// rejects an argument the table does not declare). Consequence: a string cannot be set to
        /// the empty text; use text:"" to tombstone it.
        ///
        /// WHICH SIDE (sfield vs sact) is decided the way the designer decides it -- by the attribute
        /// that references the string, not by the caller:
        ///   * the element's `comment` on a Button -> act string (XmlElement.Comment, :364-368)
        ///   * anything else (`text`/`title`/`)     -> field string
        ///   * unreferenced: an existing &lt;sact&gt; or an &lt;act&gt; with that id decides, else field.
        /// `isAct` and `referencedBy` come back in the result, so a misclassification is visible
        /// rather than silent.
        ///
        /// No .4fd writes: the lstr flags and the strings are .tsd, and binding a string to an element
        /// is a layout-attribute change (set_layout_attr) that the picker performs separately -- this
        /// function does not do it for the caller.
        /// </summary>
        public static object SetLocalString(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string name = Read.Need(a, "name");
            string text = Read.NeedKey(a, "text");

            Attr.Urm(s);                                  // mutating: Loaded -> Mutable
            object si = Si(s);
            object el = Attr.El(s, path);

            bool isAct = IsActString(s, el, name);
            string getter  = isAct ? "GetActLocalStringText"    : "GetFieldLocalStringText";
            string setter  = isAct ? "SetActLocalStringText"    : "SetFieldLocalStringText";
            string deleter = isAct ? "DeleteActLocalStringText" : "DeleteFieldLocalStringText";

            string old = Read.Str(Reflect.Call(si, getter, name));

            var res = new JObject();
            res["path"] = path;
            res["name"] = name;
            res["element"] = Read.Str(Session.Raw(el, "name"));
            res["isAct"] = isAct;
            res["before"] = old == null ? (JToken)JValue.CreateNull() : new JValue(old);
            res["referencedBy"] = Read.StrArray(ReferencedBy(el, name));

            bool unchanged = false;
            if (text.Length == 0) {
                Reflect.Call(si, deleter, name);
                res["deleted"] = true;
                res["note"] = "软删除：节点留在 <strings> 里、lstr 置 d（SPEC §6.3 的墓碑机制），"
                    + "不是从 .tsd 里摘掉";
            } else {
                res["value"] = text;
                if (old == text) {
                    // Nothing to write: AbstractStringNode.AttributeChanged returns on old==value.
                    // A no-op SUCCESS (SPEC §11.24 (a)) -- see NoOp below.
                    unchanged = true;
                } else {
                    Reflect.Call(si, setter, name, text);
                }
            }

            object node = FindStringNode(s, isAct, name);
            if (node == null)
                throw Refused("local_string_lost",
                    "写完之后找不到本地化串节点 \"" + name + "\"（设计器的 Set*LocalStringText 应当创建它）",
                    Detail("name", name));
            res["after"] = Read.Str(Reflect.Prop(node, "Text"));
            string st = Json.StatusLetter(node);
            if (st != null) res["status"] = st;
            res["tombstoned"] = st == "d";
            return unchanged
                ? NoOp(res, "本地化串 \"" + name + "\" 已经是这个内容（AbstractStringNode.AttributeChanged"
                    + " 对 old==value 直接 return），未修改任何值")
                : res;
        }

        static readonly string[] LocalStringAttrs = { "text", "title", "comment", "placeholder" };

        /// <summary>
        /// sfield or sact, by the rule the designer itself uses to read the value back: a Button's
        /// `comment` is an action string (XmlElement.Comment :364), everything else is a field string.
        /// When the element references the name at all, the referencing attribute decides; when it
        /// does not (the caller is creating a string), an existing node of either kind decides -- so a
        /// re-write lands on the node already there instead of minting a second one of the other kind
        /// and leaving the first behind.
        /// </summary>
        static bool IsActString(Session s, object el, string name) {
            foreach (string attr in LocalStringAttrs) {
                string v = (string)Session.Raw(el, attr);
                if (v == null || v != name) continue;
                return attr == "comment" && TypeName(el) == "Button";
            }
            object act = FindStringNode(s, true, name);
            if (act != null && Json.StatusLetter(act) != "d") return true;
            foreach (object n in EnumOf(Reflect.Prop(s.Si, "Actions"))) {
                if (Read.Str(Reflect.Prop(n, "Name")) == name) return true;
            }
            return false;
        }

        static string[] ReferencedBy(object el, string name) {
            var hits = new List<string>();
            foreach (string attr in LocalStringAttrs) {
                string v = (string)Session.Raw(el, attr);
                if (v != null && v == name) hits.Add(attr);
            }
            return hits.ToArray();
        }

        static object FindStringNode(Session s, bool isAct, string name) {
            string col = isAct ? "ActionStrings" : "FieldStrings";
            foreach (object n in EnumOf(Reflect.Prop(s.Si, col))) {
                if (Read.Str(Reflect.Prop(n, "Name")) == name) return n;
            }
            return null;
        }

        // ================================================================ set_items

        /// <summary>
        /// The option list of a ComboBox / RadioGroup -- the &lt;Item&gt; children and the
        /// comma-joined `items` attribute -- reproduced from ItemsPropertyEditor.saveChanged()
        /// (SpecEditor/Views/ItemsPropertyEditor.xaml.cs:171-209), the only writer of that pair in the
        /// designer.
        ///
        /// THE REFUSAL COMES FIRST. XmlElement.ReplaceItems (:2190) is
        ///
        ///     if (this.Type != ComponentType.RadioGroup &amp;&amp; this.Type != ComponentType.ComboBox)
        ///         return;
        ///
        /// -- a bare return, no exception, no message. That is the silent-corruption shape this layer
        /// exists to remove, so here it is E_DESIGNER with reason `items_unsupported`. The negative
        /// control in the acceptance run is this call on an Edit.
        ///
        /// Then saveChanged()'s own order, preserved because it is load-bearing:
        ///   1. rows whose Name AND Text are both blank (`Trim()==""`) are dropped;
        ///   2. a blank Name with a non-blank Text takes Name = Text (the grid's "type only the value"
        ///      path);
        ///   3. every OLD item whose Text is not among the survivors gets
        ///      DeleteFieldLocalStringText -- one item's local string is NAMED by its own text
        ///      (`GetFieldLocalStringText(xmlElement.Text)` at :95), so removing the item tombstones
        ///      that string;
        ///   4. each survivor gets SetFieldLocalStringText(text, description) -- so an item's
        ///      DESCRIPTION is the value of the local string named by the item's TEXT -- plus a fresh
        ///      component from ComponentFactory.CreateEmptyComponent(key, ComponentType.Item, name)
        ///      with lstrtext="true";
        ///   5. `element.ReplaceItems(list)`, which also rewrites `items` to "a, b, c" (:2204).
        ///
        /// THE ENCODING is forced by the manifest: `items` is declared PType.PathList, so every
        /// element must be a plain string (Manifest.Check rejects an object, and adding a parameter is
        /// not allowed -- the table is frozen). Each element is therefore
        /// "name|text|description" with the trailing parts optional, split on the FIRST two separators
        /// so a description may itself contain "|": "A" means name=text=A, description=""; "A|B" adds
        /// the text; "A|B|C" adds the description.
        ///
        /// The .4fd half belongs to FormWriter (that is what `save` renders): the existing &lt;Item&gt;
        /// children are removed text-wise, `items` is patched, and each new item is appended using the
        /// very XElement the model serialises (XmlElement.ToXML) -- so the written text and any later
        /// regeneration cannot disagree.
        /// </summary>
        public static object SetItems(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string[] specs = Read.ListArg(a, "items");
            if (specs == null) throw TzsError.Validation("set_items 需要 items（字符串数组）");

            Attr.Urm(s);
            object si = Si(s);
            object el = Attr.El(s, path);

            string type = TypeName(el);
            if (type != "RadioGroup" && type != "ComboBox")
                throw Refused("items_unsupported",
                    "元素 \"" + Read.Str(Session.Raw(el, "name")) + "\" 是 " + type
                    + "，不是 ComboBox/RadioGroup：XmlElement.ReplaceItems 对这种类型直接 return"
                    + "（不抛异常、不写任何东西），所以这里拒绝而不是假装成功",
                    Detail("type", type));

            var rows = new List<string[]>();
            var keptTexts = new List<string>();
            int droppedBlank = 0;
            foreach (string raw in specs) {
                string[] p = SplitItem(raw);
                string nm = p[0] ?? "";
                string tx = p[1] ?? "";
                if (nm.Trim().Length == 0 && tx.Trim().Length == 0) { droppedBlank++; continue; }  // :177
                if (nm.Trim().Length == 0) nm = tx;                                                // :181
                rows.Add(new string[] { nm, tx, p[2] ?? "" });
                keptTexts.Add(tx);
            }

            var oldNames = new List<string>();
            var oldTexts = new List<string>();
            foreach (object oi in EnumOf(Reflect.Call(el, "GetItems"))) {
                oldNames.Add(Read.Str(Session.Raw(oi, "name")));
                oldTexts.Add(Read.Str(Reflect.Prop(oi, "Text")));
            }

            // 3) removed items -> tombstone their local string (the string is named by the TEXT, :95)
            var tombstoned = new List<string>();
            foreach (string ot in oldTexts) {
                if (ot == null || keptTexts.Contains(ot)) continue;
                Reflect.Call(si, "DeleteFieldLocalStringText", ot);
                tombstoned.Add(ot);
            }

            // 4) survivors -> local string + a fresh Item component
            object itemType = ItemValue();
            IList newItems = TypedList(el.GetType());
            var made = new List<object>();
            foreach (string[] r in rows) {
                Reflect.Call(si, "SetFieldLocalStringText", r[1], r[2]);
                object item = Reflect.Call(Reflect.Find(Designer.A, "ComponentFactory"), "CreateEmptyComponent",
                    s.Key, itemType, r[0]);
                Reflect.SetProp(item, "Text", r[1]);
                Reflect.Call(item, "SetAttribute", "lstrtext", "true");
                newItems.Add(item);
                made.Add(item);
            }

            // 5) ReplaceItems -- the designer's own writer for both the Items list and `items`
            Reflect.Call(el, "ReplaceItems", newItems);

            // ---- the .4fd half -----------------------------------------------------------------
            FormWriter w = Attr.Layout(s);
            foreach (string n in oldNames) RemoveItemChild(w, path, n);
            var texts = new List<string>();
            foreach (string[] r in rows) texts.Add(r[1]);
            string joined = string.Join(", ", texts.ToArray());
            w.SetAttribute(path, "items", joined);
            foreach (object item in made) w.AddNode(path, ToXmlOf(item));

            var arr = new JArray();
            foreach (object item in made) {
                var o = new JObject();
                o["name"] = Read.Str(Session.Raw(item, "name"));
                o["text"] = Read.Str(Reflect.Prop(item, "Text"));
                arr.Add(o);
            }
            var res = new JObject();
            res["path"] = path;
            res["element"] = Read.Str(Session.Raw(el, "name"));
            res["type"] = type;
            res["count"] = rows.Count;
            res["items"] = arr;
            if (droppedBlank > 0) res["droppedBlank"] = droppedBlank;
            if (tombstoned.Count > 0) res["tombstonedStrings"] = Read.StrArray(tombstoned);
            res["itemsAttr"] = joined;
            res["note"] = "被删掉的 option 的本地化串只置 lstr=d（墓碑），不从 <strings> 摘除";

            // A column-backed element's `items` is DERIVED, and the designer re-derives it: the
            // column-change path pushes the column's col_attr value over the element's own
            // (SpecFieldNode.cs:389-397), and a reload recomputes it too -- measured, RoundTrip
            // reports `stale` on exactly this attribute afterwards while everything else holds.
            //
            // The designer's own items editor has the same fate, so this is us being faithful
            // rather than us being wrong: ComponentPropertyVisibilityMap exposes `items` for the
            // ComboBox WIDGET without asking whether the field is column-backed (SpecPropertyEditor
            // .xaml.cs:272, the only one of its nine dictionaries that lists `items`).
            //
            // Faithful is not the same as safe to leave silent. The call returns ok and the value
            // is gone after the next load, and `verify` cannot see it -- verify compares the live
            // model against the file on disk, never against a reload. That is exactly the shape of
            // the silent failures this project keeps finding, so say it out loud instead.
            string ft = Read.Str(Session.Raw(el, "fieldType"));
            if (ft == "COLUMN_LIKE" || ft == "TABLE_COLUMN") {
                res["derivedFromColumn"] = true;
                res["note"] = Read.Str(res["note"]) + "；**这一条是列派生值**：元素 " + ft
                    + " 的 items 来自绑定列的 col_attr，设计器在换列时（SpecFieldNode.cs:389-397）"
                    + "和加载时都会重算它，所以这次写入在下次加载后会被覆盖——设计器自己的 items "
                    + "编辑器同样如此（属性可见性按 ComboBox 控件分，不问是否列绑定），我们复现了它。"
                    + "要让它持久，改列的基础资料；另外 verify 看不见这件事（它比的是内存模型与"
                    + "磁盘文件，不是与一次重载）";
            }
            return res;
        }

        /// <summary>"name|text|description" split on the first two separators; a missing tail is the
        /// empty string. Nothing is trimmed here -- saveChanged trims only to decide "both blank" and
        /// writes the untrimmed values back, so the caller must not help.</summary>
        static string[] SplitItem(string raw) {
            if (raw == null) return new string[] { "", "", "" };
            int i = raw.IndexOf('|');
            if (i < 0) return new string[] { raw, raw, "" };
            string nm = raw.Substring(0, i);
            string rest = raw.Substring(i + 1);
            int j = rest.IndexOf('|');
            if (j < 0) return new string[] { nm, rest, "" };
            return new string[] { nm, rest.Substring(0, j), rest.Substring(j + 1) };
        }

        /// <summary>ComponentType.Item as the designer's own enum value (the value its factory's
        /// parameter takes -- passing the Type object would not bind and the loose binder would then
        /// pick a same-arity overload at random).</summary>
        static object ItemValue() {
            Type ct = Reflect.Find(Designer.A, "ComponentType");
            if (ct == null) throw new Exception("找不到 ComponentType 枚举（设计器版本变了？）");
            return Enum.Parse(ct, "Item");
        }

        /// <summary>
        /// Removes one &lt;Item&gt; child from the .4fd text. ByPath returns the FIRST element with a
        /// path, so a repeated name is handled by asking again after each removal (Reindex makes the
        /// next one findable at the same path) rather than silently leaving siblings behind.
        /// </summary>
        static void RemoveItemChild(FormWriter w, string parentPath, string name) {
            if (string.IsNullOrEmpty(name)) return;
            string p = parentPath + "/" + name;
            for (int guard = 0; guard < 64; guard++) {
                if (w.Index.ByPath(p) == null) return;
                w.RemoveNode(p);
            }
        }

        // ================================================================ set_progrel_programs

        /// <summary>
        /// Add or delete one 串查程序 of a &lt;pfield&gt; node, through
        /// ChangeProgRelProgramUndoRedoCommand(node, program, isDelete) -- the same command
        /// ProgRelProgramsDataGrid's Add/Delete buttons run (Views/ProgRelProgramsDataGrid.xaml.cs:49
        /// /:66). Creation goes through ProgRelProgram.Create(node), which is where
        /// name="" type="" order=getMaxOrder()+1 come from; the program name is then written through
        /// the property (ProgRelProgram.Program -> ProgRelProgramAttributeUndoRedoCommand), which is
        /// what the grid's TextBox cell does.
        ///
        /// THERE IS NO REORDER in the designer's UI -- only Add and Delete -- so there is none here
        /// either: `order` is assigned at creation and never reshuffled.
        ///
        /// THE GRID GATE, reproduced: ProgRelProgramsDataGrid.xaml disables the whole panel when the
        /// bound CiteStd is "Y" (DataTrigger on ProgRelNode.CiteStd, :20-34). That property exists on
        /// neither SpecProgRelNode nor AbstractSpecNode, so the binding never resolves and in the
        /// designer the gate is dead code -- the buttons stay enabled. The intent ("a cited node's
        /// 串查 belongs to the standard, not to you") is unmistakable, and this layer's rule is that a
        /// designer gate is reproduced rather than lost to a stale binding, so it is enforced on the
        /// attribute the binding obviously means: cite_std on the pfield node. Refusing is the safe
        /// direction -- a write the designer would have silently ignored is exactly the failure mode
        /// being designed out. A caller who genuinely needs the permissive behaviour is asking for a
        /// contract change, not a workaround.
        /// </summary>
        public static object SetProgrelPrograms(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string program = Read.Need(a, "program");
            bool isDelete = BoolArg(a, "isDelete", false);

            Attr.Urm(s);
            object el = Attr.El(s, path);
            object fsm = Attr.Fsm(s, el);
            object node = Reflect.Prop(fsm, SpecSlots.ByKind["pfield"]);
            if (node == null)
                throw Refused("no_pfield",
                    "元素 \"" + Read.Str(Session.Raw(el, "name")) + "\" 没有 pfield（串查）节点；"
                    + "串查只存在于有 SpecProgRel 的字段上", Detail("path", path));

            string citeStd = NodeAttr(node, "cite_std");
            if (citeStd != "N")
                throw Refused("grid_disabled",
                    "串查程序的增删被拒：pfield 节点的 cite_std=\"" + citeStd + "\"（非 N）。设计器里这块"
                    + "面板在 CiteStd 为 Y 时整体 IsEnabled=false（ProgRelProgramsDataGrid.xaml:20-34），"
                    + "所以这里也拒绝", Detail("cite_std", citeStd));

            object existing = FindProgram(node, program);
            var res = new JObject();
            res["path"] = path;
            res["element"] = Read.Str(Session.Raw(el, "name"));
            res["program"] = program;

            if (isDelete) {
                if (existing == null) {
                    JObject d = Detail("program", program);
                    d["candidates"] = ProgramNames(node);
                    throw NotFound("这条串查里的程序", program, d);
                }
                object cmd = Attr.NewCommand("ChangeProgRelProgramUndoRedoCommand",
                    new object[] { node, existing, true });
                Attr.Run(s, cmd);
                res["deleted"] = true;
            } else {
                if (existing != null)
                    throw Refused("program_exists",
                        "串查程序 \"" + program + "\" 已经在这条 pfield 上了（设计器的 Add 会再加一条重复"
                        + "行；这里直接拒绝，避免造出重名行）", Detail("program", program));
                object prog = Reflect.Call(Reflect.Find(Designer.A, "ProgRelProgram"), "Create", node);
                if (prog == null) throw new Exception("ProgRelProgram.Create 返回 null");
                Reflect.SetProp(prog, "Program", program);
                object cmd = Attr.NewCommand("ChangeProgRelProgramUndoRedoCommand",
                    new object[] { node, prog, false });
                Attr.Run(s, cmd);
                res["added"] = true;
                res["order"] = Read.Str(Reflect.Prop(prog, "Order"));
            }

            res["programs"] = ProgramList(node);
            string st = Json.StatusLetter(node);
            if (st != null) res["specStatus"] = st;
            return res;
        }

        static object FindProgram(object node, string program) {
            foreach (object p in EnumOf(Reflect.Prop(node, "Programs"))) {
                if (Read.Str(Reflect.Prop(p, "Program")) == program) return p;
            }
            return null;
        }

        static JArray ProgramList(object node) {
            var arr = new JArray();
            foreach (object p in EnumOf(Reflect.Prop(node, "Programs"))) {
                var o = new JObject();
                o["program"] = Read.Str(Reflect.Prop(p, "Program"));
                o["type"] = Read.Str(Reflect.Prop(p, "Type"));
                o["order"] = Read.Str(Reflect.Prop(p, "Order"));
                arr.Add(o);
            }
            return arr;
        }

        static JArray ProgramNames(object node) {
            var arr = new JArray();
            foreach (object p in EnumOf(Reflect.Prop(node, "Programs")))
                arr.Add(Read.Str(Reflect.Prop(p, "Program")));
            return arr;
        }

        // ================================================================ set_table_association

        /// <summary>
        /// The &lt;table&gt;/&lt;tbl&gt;/&lt;sr&gt; association of a Table / ScrollGrid element: which
        /// table this grid is bound to, and whether the user may add/delete/append rows.
        ///
        /// TWO REFERENCE HANDLERS, both in SpecPropertyEditor.xaml.cs:
        ///
        ///   tableAssocCB_SelectionChanged (:950-1001) -- the table move. Mark the OLD table's
        ///     &lt;sr name=thisElement&gt; status="d", remove any same-named &lt;sr&gt; from the NEW
        ///     table, then add a fresh
        ///       &lt;sr name=… src=env status="u" cascade="Y" insert="Y" append="Y" delete="Y" kind=…&gt;
        ///     to the new table. `kind` is the element's own ComponentType, `status` is "u" (MODIFY,
        ///     via ReflectionHelpers.GetCustomDescription), `src` is the .tsd's env -- all three are
        ///     the designer's values, not re-derived here. Attribute ORDER is the reference's too:
        ///     XElement.Parse lays down name/src/status/cascade, the three Add() calls append
        ///     insert/append/delete, and SetAttributeValue("kind") appends last.
        ///
        ///   SRAttributesCB_Click (:823-864) -- the INSERT/DELETE/APPEND ROW checkboxes. It finds the
        ///     live &lt;sr&gt; for this element and, when there is none, RETURNS EARLY (:836-839): the
        ///     row must already exist, the boxes edit it and never create it. That early return is
        ///     reproduced as E_DESIGNER `sr_missing`, because a silent no-op here is indistinguishable
        ///     from a successful write.
        ///
        /// THE MANIFEST CANNOT CARRY A BOOLEAN, so the two behaviours are selected by `column`, the
        /// only optional parameter available (the move needs nothing but `table`, and `column` has no
        /// &lt;sr&gt; counterpart anywhere in the designer -- an &lt;sr&gt; has no column attribute).
        /// `column` naming one of the three row flags means "flip that box", which is what a click
        /// does and the only thing a checkbox without a value can mean; its new value is the negation
        /// of the current one. A `column` that is none of the three is E_BAD_PARAM naming the legal
        /// set. This is a documented consequence of the frozen manifest, not an invention: the
        /// alternative was to leave the checkbox behaviour unreachable from the CLI.
        ///
        /// No .4fd writes: the whole section is .tsd, rendered by SaveToTSD from AssociateTable.Source.
        /// </summary>
        public static object SetTableAssociation(Session s, JObject a) {
            string path = Read.Need(a, "path");
            string table = Read.Need(a, "table");
            string column = Read.Arg(a, "column");

            Attr.Urm(s);
            object si = Si(s);
            object el = Attr.El(s, path);
            string elementName = Read.Str(Session.Raw(el, "name"));
            if (string.IsNullOrEmpty(elementName))
                throw TzsError.Validation("元素 \"" + path + "\" 没有 name，无法与 <sr name=…> 对应");

            XElement assoc = Reflect.Prop(Reflect.Prop(si, "AssociateTable"), "Source") as XElement;
            if (assoc == null)
                throw Refused("no_table_section", "这张 .tsd 没有 <table> 段（AssociateTable.Source 为空）", null);

            if (!string.IsNullOrEmpty(column)) return Flag(assoc, elementName, column);

            // ---- table move (tableAssocCB_SelectionChanged) ---------------------------------
            XElement oldTbl;
            XElement live = LiveSr(assoc, elementName, out oldTbl);
            XElement newTbl = null;
            var tables = new List<string>();
            foreach (XElement t in assoc.Elements("tbl")) {
                if (Reflect.Attr(t, "status") == "d") continue;
                string tn = Reflect.Attr(t, "name");
                tables.Add(tn);
                if (tn == table && newTbl == null) newTbl = t;
            }
            if (newTbl == null) {
                JObject d = Detail("table", table);
                d["candidates"] = Read.StrArray(tables);
                throw NotFound("关联表", table + "（<table> 段里没有这张表）", d);
            }

            if (live != null) live.SetAttributeValue("status", "d");
            foreach (XElement sr in new List<XElement>(newTbl.Elements("sr"))) {
                if (Reflect.Attr(sr, "name") == elementName) sr.Remove();
            }
            XElement created = XElement.Parse("<sr name='' src='s' status='u' cascade='Y' />", LoadOptions.None);
            created.Add(new XAttribute("insert", "Y"));
            created.Add(new XAttribute("append", "Y"));
            created.Add(new XAttribute("delete", "Y"));
            created.SetAttributeValue("name", elementName);
            created.SetAttributeValue("src", Env(s));
            created.SetAttributeValue("status", "u");
            created.SetAttributeValue("kind", TypeName(el) ?? "");
            newTbl.Add(created);

            var res = new JObject();
            res["path"] = path;
            res["element"] = elementName;
            res["table"] = table;
            res["movedFrom"] = oldTbl == null ? null : Reflect.Attr(oldTbl, "name");
            res["sr"] = Read.XAttrs(created);
            res["note"] = "旧表里的同名 <sr> 置 status=d，新表里的同名 <sr> 先删后建（status=u），"
                + "新行的 insert/append/delete 都是 Y —— 与 tableAssocCB_SelectionChanged 一致";
            return res;
        }

        /// <summary>The INSERT/DELETE/APPEND ROW checkbox click, on an &lt;sr&gt; that already exists.
        /// `flag` names the box; the new value is the negation of the current one (see the note on
        /// SetTableAssociation for why a toggle is the only reading the manifest allows).</summary>
        static object Flag(XElement assoc, string elementName, string flag) {
            if (flag != "insert" && flag != "delete" && flag != "append")
                throw TzsError.Validation("column 只能是 insert/delete/append 之一（它在这里表示"
                    + " INSERT/DELETE/APPEND ROW 那个勾选框），收到: " + flag);

            XElement dummy;
            XElement sr = LiveSr(assoc, elementName, out dummy);
            if (sr == null)
                throw Refused("sr_missing",
                    "找不到 \"" + elementName + "\" 的活 <sr> 行：SRAttributesCB_Click 找不到就直接 return"
                    + "（SpecPropertyEditor.xaml.cs:836-839），它只改已存在的行、不负责创建。"
                    + "先用 table 参数把关联建好，再改这几个标记",
                    Detail("element", elementName));

            string cur = Reflect.Attr(sr, flag);
            string next = cur == "Y" ? "N" : "Y";
            sr.SetAttributeValue(flag, next);
            sr.SetAttributeValue("status", "u");     // the handler always forces MODIFY (:862)

            var res = new JObject();
            res["element"] = elementName;
            res["column"] = flag;
            res["old"] = cur;
            res["value"] = next;
            res["sr"] = Read.XAttrs(sr);
            return res;
        }

        /// <summary>The live &lt;sr name=elementName&gt; -- not tombstoned -- anywhere in the section,
        /// plus the &lt;tbl&gt; that holds it. The reference searches Descendants("sr") for the
        /// checkbox path and walks tbl/sr for the move path; both land on this same node.</summary>
        static XElement LiveSr(XElement assoc, string elementName, out XElement owner) {
            owner = null;
            foreach (XElement tbl in assoc.Elements("tbl")) {
                foreach (XElement sr in tbl.Elements("sr")) {
                    if (Reflect.Attr(sr, "name") != elementName) continue;
                    if (Reflect.Attr(sr, "status") == "d") continue;
                    owner = tbl;
                    return sr;
                }
            }
            return null;
        }

        // ================================================================ set_spec_description

        /// <summary>
        /// The SD 规格描述 -- the CDATA body of a node, the free-text specification the designer shows
        /// in its spec editor. Per node it is SDSpecUndoRedoCommand(node, content)
        /// (UndoRedoCommands/SDSpecUndoRedoCommand.cs), reached the way the editor reaches it: through
        /// the CDATA property itself, so the designer's own guard runs.
        ///
        /// THE GUARD IS THE INTERESTING PART. AbstractSpecNode.CDATA's setter (:226) is
        ///
        ///     if (this.IsCited || value == this.CDATA) return;
        ///
        /// -- a cited node's description belongs to the standard (its Source is a copy of the cited
        /// XML, see set_cited), so the write is refused *silently*. Reporting "ok" there would be the
        /// exact lie this layer exists to prevent, so IsCited is checked here and surfaced as
        /// E_DESIGNER `cited`. The other half of the same guard, old==new, is the opposite case: the
        /// form already says what the caller asked for, so that one *is* reported as ok -- as the
        /// no-op success of §11.24 (a), carrying `noop:true` + `code:"E_NO_OP"`. The check is not
        /// redundant either way: the command class has no guard of its own, so anything that invoked
        /// it directly would bypass the rule.
        ///
        /// THE FOUR PROGRAM-LEVEL SPECS are the same CDATA on SpecificationInfo's four program nodes
        /// -- ProgramSpec (&lt;all&gt;), ProgramMISpec (&lt;mi_all&gt;), ProgramDBSpec (&lt;db_all&gt;),
        /// ProgramDISpec (&lt;di_all&gt;) -- which are SpecProgram : AbstractSpecNode, so the same
        /// command and the same guard apply. They have no name, so they are addressed by `path`: a bare
        /// `all` / `mi_all` / `db_all` / `di_all` selects one (a real name-path always contains "/", so
        /// a bare token cannot collide with an element path). `kind` is ignored for those.
        ///
        /// A content containing "]]>" is refused: XCData cannot represent it and the designer would
        /// throw from inside the setter rather than from anything the caller could act on.
        /// </summary>
        public static object SetSpecDescription(Session s, JObject a) {
            string kind = Read.Arg(a, "kind");
            string path = Read.Need(a, "path");
            string content = Read.NeedKey(a, "content");
            if (content.IndexOf("]]>", StringComparison.Ordinal) >= 0)
                throw TzsError.Validation("content 不能含 \"]]>\"（XCData 表示不了，设计器会在 setter 里抛）");

            Attr.Urm(s);
            object si = Si(s);

            object node;
            string target;
            int alias = ProgramAlias(path);
            if (alias >= 0) {
                node = Reflect.Prop(si, ProgramSpecProp[alias]);
                target = ProgramSpecAlias[alias];
                if (node == null) throw NotFound("程序级规格节点", target, null);
            } else {
                if (string.IsNullOrEmpty(kind))
                    throw TzsError.Validation("set_spec_description 需要 kind（七种之一），或 path=\"all\"/"
                        + "\"mi_all\"/\"db_all\"/\"di_all\" 指向程序级规格");
                if (Array.IndexOf(SpecSlots.Kinds, kind) < 0)
                    throw TzsError.Validation("未知 kind：" + kind + "；合法值: "
                        + string.Join(",", SpecSlots.Kinds));
                object el = Attr.El(s, path);
                object fsm = Attr.Fsm(s, el);
                node = Reflect.Prop(fsm, SpecSlots.ByKind[kind]);
                target = kind;
                if (node == null) {
                    // E_NO_SPEC_NODE, not the `designer` kind Refused() would give -- same reasoning as
                    // the identical refusal in set_cited below and in Attr.SpecNode: the message tells
                    // the caller to ask for another kind, so the classification must be not_found
                    // ("self-correctable") rather than designer ("do not retry, tell the user").
                    var dn = Detail("path", path);
                    dn["kind"] = kind;
                    dn["reason"] = "no_spec_node";
                    throw new DetailedError("E_NO_SPEC_NODE",
                        "元素 \"" + Read.Str(Session.Raw(el, "name")) + "\" 没有 " + kind
                        + " 节点（先用 get_component 看它有哪些 kind）", dn);
                }
            }

            if (IsCited(node))
                throw Refused("cited",
                    "节点被引用（cite_std != \"N\"）：AbstractSpecNode.CDATA 的 setter 对 IsCited 直接"
                    + " return —— 设计器会静默丢弃这次写入，所以这里报错而不是假成功。描述属于被引用的"
                    + "标准；要改先 set_cited cited:false 取消引用", Detail("target", target));

            string old = Read.Str(Reflect.Prop(node, "CDATA"));

            var res = new JObject();
            res["target"] = target;
            if (alias < 0) res["path"] = path;
            res["oldLength"] = old == null ? 0 : old.Length;
            res["newLength"] = content.Length;

            if (old == content) {
                // The other half of the guard below: a no-op SUCCESS, not an error (SPEC §11.24 (a)).
                res["cdata"] = old;
                string nst = Json.StatusLetter(node);
                if (nst != null) res["status"] = nst;
                return NoOp(res, "规格描述已经是这个内容（setter 对 value==CDATA 直接 return），未修改任何值");
            }

            // The designer's own path: CDATA property -> SDSpecUndoRedoCommand -> ReplaceNodes(XCData).
            Reflect.SetProp(node, "CDATA", content);

            res["cdata"] = Read.Str(Reflect.Prop(node, "CDATA"));
            string st = Json.StatusLetter(node);
            if (st != null) res["status"] = st;
            return res;
        }

        static readonly string[] ProgramSpecAlias = { "all", "mi_all", "db_all", "di_all" };
        static readonly string[] ProgramSpecProp  = { "ProgramSpec", "ProgramMISpec", "ProgramDBSpec", "ProgramDISpec" };

        static int ProgramAlias(string path) {
            for (int i = 0; i < ProgramSpecAlias.Length; i++)
                if (ProgramSpecAlias[i] == path) return i;
            return -1;
        }

        static bool IsCited(object node) { return AsBool(Reflect.Prop(node, "IsCited")); }

        // ================================================================ set_cited

        /// <summary>
        /// 引用 / 取消引用标准: SpecCitedUndoRedoCommand(node, isCited), which swaps every spec node of
        /// the element between its own XML and the standard's (&lt;CiteSTD&gt;) and sets cite_std=Y/N on
        /// all of them (FormSpecModel.IsCited).
        ///
        /// THE GATING IS THE FUNCTION. AbstractSpecNode.CitedSpec
        /// (SpecProgRelNode.cs:92 is the pfield shape; the other six are the same) is
        ///
        ///     if (IsStandardProgram) return null;
        ///     var citeSTD = SpecificationInfo.CiteSTD;  if (citeSTD == null) return null;
        ///     …the cited element with this name…
        ///
        /// so the command is only meaningful for a NON-standard program (prog != std_prog) that really
        /// loaded a CiteSTD. Otherwise CitedSpec is null and Execute does one of two bad things: it
        /// NREs when the node is not in FormSpeDictionary, or -- worse -- it reaches
        /// `this._fsm.IsCited = isCited` with every Source swap skipped, stamping cite_std=Y on a node
        /// whose content was never cited. Both are refused here, as `standard_program` and
        /// `no_cited_spec`.
        ///
        /// NOTE for the acceptance run: all 91 corpus files have prog == std_prog and none carries
        /// cite_std="Y", so this function has no positive case in the corpus -- only the negative one.
        /// That is reported rather than worked around.
        /// </summary>
        public static object SetCited(Session s, JObject a) {
            string path = Read.Need(a, "path");
            bool cited = BoolArg(a, "cited", false);

            Attr.Urm(s);
            object tzp = s.Tzp;
            if (AsBool(Reflect.Prop(tzp, "IsStandardProgram")))
                throw Refused("standard_program",
                    "这张表单是标准程序（prog == std_prog）：AbstractSpecNode.CitedSpec 对标准程序直接返回"
                    + " null，没有可引用的来源，set_cited 对它没有意义",
                    Detail("program", Read.Str(Reflect.Prop(tzp, "ProgramName"))));

            object el = Attr.El(s, path);
            object fsm = Attr.Fsm(s, el);
            object node = FirstSlot(fsm);
            if (node == null) {
                // E_NO_SPEC_NODE, not the `designer` kind Refused() would give: §11.24 (a) classifies
                // this one as not_found ("该元素没有这种规格节点；换一个 kind"), because the caller's
                // fix is to ask for a different kind. `designer` means "do not retry, tell the user",
                // which is the opposite advice.
                var d = Detail("path", path);
                d["reason"] = "no_spec_node";
                throw new DetailedError("E_NO_SPEC_NODE",
                    "元素 \"" + Read.Str(Session.Raw(el, "name")) + "\" 没有任何规格节点", d);
            }

            if (Reflect.Prop(node, "CitedSpec") == null)
                throw Refused("no_cited_spec",
                    "元素 \"" + Read.Str(Session.Raw(el, "name")) + "\" 没有 CitedSpec：非标准程序必须先加载"
                    + "到标准程序的规格（CiteSTD），否则 SpecCitedUndoRedoCommand 要么 NRE，要么给一个"
                    + "没换过内容的节点盖上 cite_std=Y", Detail("path", path));

            bool wasCited = IsCited(fsm);
            object cmd = Attr.NewCommand("SpecCitedUndoRedoCommand", new object[] { node, cited });
            Attr.Run(s, cmd);

            var res = new JObject();
            res["path"] = path;
            res["element"] = Read.Str(Session.Raw(el, "name"));
            res["wasCited"] = wasCited;
            res["cited"] = IsCited(fsm);
            res["citeStd"] = NodeAttr(node, "cite_std");
            string txt = Read.Str(Reflect.Prop(node, "CDATA"));
            res["cdataLength"] = txt == null ? 0 : txt.Length;
            return res;
        }

        /// <summary>The first non-null slot of the FormSpecModel, in the frozen slot order. The command
        /// resolves the model itself (by the node's Name), so any node of the element serves; picking
        /// deterministically keeps the result reproducible.</summary>
        static object FirstSlot(object fsm) {
            foreach (string kind in SpecSlots.Kinds) {
                object n = Reflect.Prop(fsm, SpecSlots.ByKind[kind]);
                if (n != null) return n;
            }
            return null;
        }

        /// <summary>FormDesignSetting.containers (Helpers/FormDesignSetting.cs:114). Spelled here rather
        /// than reflected because it is a closed list in the designer and FormDesignSetting lives in
        /// SpecDesigner.FormEditor, which is not part of this library's load set.</summary>
        static readonly string[] Containers =
            { "Folder", "Form", "Grid", "Group", "HBox", "Page", "ScrollGrid", "Table", "Tree", "VBox" };

        static bool IsContainer(string nodeName) {
            return nodeName != null && Array.IndexOf(Containers, nodeName) >= 0;
        }

        /// <summary>The rows of list_open, taken from the function map rather than from a second
        /// registry -- the answer is then whatever the session model itself reports.</summary>
        static IEnumerable Rows() {
            Fn listOpen;
            if (!Rpc.Fns.TryGetValue("list_open", out listOpen)) return new object[0];
            object rows = listOpen(null, new JObject());
            return rows as IEnumerable ?? new object[0];
        }

        /// <summary>One field of a list_open row. The rows are Dictionary&lt;string,object&gt; (the shape
        /// Fns/Session.Info returns, which JToken.FromObject then serialises), so they are read as a
        /// dictionary rather than reflected on -- Reflect.Prop would silently return null for every key,
        /// which is exactly the kind of "no error, wrong answer" this file is supposed to refuse.</summary>
        static object RowField(object row, string key) {
            IDictionary d = row as IDictionary;
            if (d != null) return d.Contains(key) ? d[key] : null;
            return Reflect.Prop(row, key);
        }
    }
}
