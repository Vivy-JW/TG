using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;

namespace TzsCli
{
    /// <summary>
    /// Minimal-diff writer for the .4fd entry of a .tzs.
    ///
    /// Strategy: the layout section is edited as raw text splices (using ElementIndex
    /// offsets) so untouched bytes survive verbatim; only the Record section is
    /// regenerated, because the designer renumbers fieldId/fieldIdRef on every save.
    /// </summary>
    public sealed class FormWriter
    {
        string _text;
        ElementIndex _idx;
        XElement _root;
        bool _dirty;

        FormWriter(string text) {
            _text = text;
            _idx = ElementIndex.Build(text);
            _root = XElement.Parse(text);
        }

        public static FormWriter Load(string fd4Text) { return new FormWriter(fd4Text); }

        public bool Dirty { get { return _dirty; } }
        public ElementIndex Index { get { return _idx; } }

        /// <summary>Final text. Guaranteed byte-identical to the input when nothing was applied.</summary>
        public string Render() {
            if (!_dirty) return _text;
            // 1) rebuild the Record section against the (already edited) tree. This also
            //    assigns fieldId to every element it walks, so the layout text has to be
            //    brought back in line before the section is rendered.
            var rb = new RecordRebuilder();
            if (HasColName != null) rb.HasColName = HasColName;   // null would clobber the default
            rb.Rebuild(_root);
            string rendered = rb.Render();

            // 2) push the model's fieldId back into the layout text. The Record section is
            //    regenerated wholesale, but the layout is spliced — so any id the rebuild
            //    changed (an insert shifts the traversal order) must be patched here or the
            //    two halves disagree and the RecordField's fieldIdRef dangles.
            SyncFieldIds();

            int rs, re;
            FindRecordSpan(_text, out rs, out re);
            if (rs < 0) throw new InvalidOperationException("Record section not found");
            string result = _text.Substring(0, rs) + rendered + _text.Substring(re);

            // 3) DiagramLayout is dropped by the designer's SaveToForm on every save
            if (DropDiagramLayout) {
                var dl = _idx.ByPath("DiagramLayout");
                if (dl != null) {
                    // recompute against `result` (the record splice shifts offsets)
                    var idx2 = ElementIndex.Build(result);
                    var d2 = idx2.All.FirstOrDefault(e => e.Tag == "DiagramLayout");
                    if (d2 != null) {
                        int s = LineStart(result, d2.OpenStart);
                        int e = LineEnd(result, d2.CloseEnd);
                        result = result.Remove(s, e - s);
                    }
                }
            }
            return result;
        }

        /// <summary>Patches layout elements whose fieldId differs between the model and the text.
        /// The element we inserted has no fieldId in the text at all, so this both updates and adds.</summary>
        void SyncFieldIds() {
            var idx = ElementIndex.Build(_text);
            var offsets = new List<int>();
            var oldLens = new List<int>();
            var pieces  = new List<string>();
            var form = _root.Element("Form");
            if (form == null) return;

            foreach (var e in form.DescendantsAndSelf()) {
                var fa = e.Attribute("fieldId");
                if (fa == null) continue;
                var span = idx.ByPath(PathOf(e, _root));
                if (span == null) continue;

                int vs, ve;
                if (FindAttrValueSpan(_text, span, "fieldId", out vs, out ve) >= 0) {
                    if (_text.Substring(vs, ve - vs) == fa.Value) continue;      // already in sync
                    offsets.Add(vs); oldLens.Add(ve - vs); pieces.Add(Escape(fa.Value));
                } else {
                    // attribute absent: insert it just before the tag's closing '>' (or '/>')
                    int at = span.OpenEnd - 1;
                    if (at > span.OpenStart && _text[at - 1] == '/') at--;
                    while (at > span.OpenStart && _text[at - 1] == ' ') at--;
                    offsets.Add(at); oldLens.Add(0);
                    pieces.Add(" fieldId=\"" + Escape(fa.Value) + "\"");
                }
            }
            // apply back-to-front so earlier offsets stay valid
            var order = Enumerable.Range(0, offsets.Count).OrderByDescending(i => offsets[i]).ToList();
            foreach (int i in order)
                _text = _text.Remove(offsets[i], oldLens[i]).Insert(offsets[i], pieces[i]);
            if (offsets.Count > 0) _idx = ElementIndex.Build(_text);
        }

        public bool DropDiagramLayout = true;

        // ---------- operations ----------

        /// <summary>
        /// Row just below every child of the container at <paramref name="parentPath"/>,
        /// i.e. a row guaranteed free. The designer computes placement from the drop
        /// position; a generator has no such input, so it appends below instead of
        /// cloning the template's coordinates (which would land on top of the template).
        /// </summary>
        public int NextFreeRow(string parentPath) {
            var parent = _idx.ByPath(parentPath);
            if (parent == null) throw new ArgumentException("path not found: " + parentPath);
            int maxY = 0;
            foreach (var c in parent.Children) {
                int y, h;
                int.TryParse(AttrOf(c, "posY"), out y);
                if (!int.TryParse(AttrOf(c, "gridHeight"), out h)) h = 1;
                if (y + h > maxY) maxY = y + h;
            }
            return maxY;
        }

        string AttrOf(ElementSpan el, string attr) {
            int vs, ve;
            if (FindAttrValueSpan(_text, el, attr, out vs, out ve) < 0) return null;
            return _text.Substring(vs, ve - vs);
        }

        /// <summary>
        /// Grows the container's gridHeight so a child placed at <paramref name="bottom"/>
        /// is not clipped.
        /// </summary>
        public void EnsureHeight(string parentPath, int bottom) {
            var parent = _idx.ByPath(parentPath);
            if (parent == null) return;
            var xe = FindElement(_root, parentPath);
            if (xe == null) return;
            int cur;
            if (!int.TryParse((string)xe.Attribute("gridHeight"), out cur)) return;
            if (bottom > cur) SetAttribute(parentPath, "gridHeight", bottom.ToString());
        }

        /// <summary>Adds a component element as a child of the container at <paramref name="parentPath"/>.</summary>
        public void AddNode(string parentPath, XElement element, int index = -1) {
            var parent = _idx.ByPath(parentPath);
            if (parent == null) throw new ArgumentException("parent path not found: " + parentPath);

            int indent = parent.Indent + 2;
            // Indent()'s contract is that its block's FIRST line carries no leading spaces -- the
            // caller supplies them, and the recursion already does so for each child. AddNode did
            // not, so the inserted element's opening tag landed in column 0 while its own subtree
            // was correctly indented, and the splice above/below it (which adds "\r\n" on both
            // sides) made it look like the whole block was out of place. Cosmetic only: the
            // designer reparses the XML and RoundTrip compares the normalised form, so no
            // criterion ever saw it.
            string block = new string(' ', indent) + Indent(element, indent);

            int at;
            string insert;
            if (parent.SelfClosing) {
                // expand <X ... /> into <X ...>\r\n<child>\r\n<indent></X>
                string open = OpenTagOf(element, _text, parent);   // reuse parent's own tag text
                string openText = _text.Substring(parent.OpenStart, parent.OpenEnd - parent.OpenStart);
                string expanded = openText.Substring(0, openText.Length - 2) + ">";   // strip " />" -> ">"
                insert = expanded + "\r\n" + block + "\r\n" + new string(' ', parent.Indent) + "</" + parent.Tag + ">";
                _text = _text.Remove(parent.OpenStart, parent.OpenEnd - parent.OpenStart).Insert(parent.OpenStart, insert);
            } else if (parent.Children.Count == 0) {
                at = parent.OpenEnd;
                insert = "\r\n" + block + "\r\n" + new string(' ', parent.Indent);
                _text = _text.Insert(at, insert);
            } else {
                int i = (index < 0 || index >= parent.Children.Count) ? parent.Children.Count - 1 : index;
                var anchor = parent.Children[i];
                at = LineBreakBefore(_text, anchor.CloseEnd);
                insert = "\r\n" + block;
                _text = _text.Insert(at, insert);
            }
            _dirty = true;
            Reindex();
        }

        /// <summary>Sets an attribute on the element at <paramref name="path"/>.</summary>
        public void SetAttribute(string path, string attr, string value) {
            var el = _idx.ByPath(path);
            if (el == null) throw new ArgumentException("path not found: " + path);
            var xe = FindElement(_root, path);
            if (xe == null) throw new ArgumentException("path not found in tree: " + path);

            var existing = xe.Attribute(attr);
            if (existing != null) {
                // patch just the value text inside the tag
                int valStart, valEnd;
                int p = FindAttrValueSpan(_text, el, attr, out valStart, out valEnd);
                if (p < 0) throw new InvalidOperationException("attr span not found: " + attr);
                _text = _text.Remove(valStart, valEnd - valStart).Insert(valStart, Escape(value));
            } else {
                // insert a new attribute just before the tag's closing '>' (or '/>')
                int tagEnd = el.OpenEnd - 1;
                if (_text[tagEnd - 1] == '/') tagEnd--;
                while (tagEnd > el.OpenStart && _text[tagEnd - 1] == ' ') tagEnd--;
                string ins = " " + attr + "=\"" + Escape(value) + "\"";
                _text = _text.Insert(tagEnd, ins);
            }
            xe.SetAttributeValue(attr, value);
            _dirty = true;
            Reindex();
        }

        /// <summary>Removes the element at <paramref name="path"/> from the layout.</summary>
        public void RemoveNode(string path) {
            var el = _idx.ByPath(path);
            if (el == null) throw new ArgumentException("path not found: " + path);
            var xe = FindElement(_root, path);
            if (xe == null) throw new ArgumentException("path not found in tree: " + path);

            int s = LineStart(_text, el.OpenStart);
            int e = LineEnd(_text, el.CloseEnd);
            if (el.SelfClosing) {
                // self-closing occupies one line
                s = LineStart(_text, el.OpenStart);
                e = LineEnd(_text, el.OpenEnd);
            }
            _text = _text.Remove(s, e - s);
            xe.Remove();
            _dirty = true;
            Reindex();
        }

        // ---------- element construction ----------

        /// <summary>
        /// Builds a new element by cloning the attribute set of an existing element of the
        /// same tag in this very file. Cloning beats a hardcoded table because the file was
        /// written by a specific designer version and its attribute set/order is what that
        /// version round-trips.
        /// </summary>
        public XElement CloneTemplate(string tag, IDictionary<string,string> overrides) {
            var src = _root.Descendants().FirstOrDefault(e => e.Name.LocalName == tag && e.Elements().Any() == false);
            if (src == null) src = _root.Descendants().FirstOrDefault(e => e.Name.LocalName == tag);
            if (src == null) throw new InvalidOperationException("no template element of type " + tag + " in this file");

            var el = new XElement(tag);
            foreach (var a in src.Attributes()) el.SetAttributeValue(a.Name.LocalName, a.Value);
            if (overrides != null)
                foreach (var kv in overrides)
                    if (kv.Value == null) el.SetAttributeValue(kv.Key, null);
                    else el.SetAttributeValue(kv.Key, kv.Value);
            return el;
        }

        // ---------- rendering helpers ----------

        static string Indent(XElement e, int indent) {
            var sb = new StringBuilder(SelfClosing(e));
            // nested children, if any, get one more level
            var kids = e.Elements().ToList();
            if (kids.Count > 0) {
                sb = new StringBuilder(OpenTag(e)).Append("\r\n");
                foreach (var k in kids) sb.Append(new string(' ', indent + 2)).Append(Indent(k, indent + 2)).Append("\r\n");
                sb.Append(new string(' ', indent)).Append("</").Append(e.Name.LocalName).Append(">");
            }
            return sb.ToString();
        }
        static string OpenTag(XElement e) {
            var sb = new StringBuilder("<").Append(e.Name.LocalName);
            foreach (var a in e.Attributes()) sb.Append(' ').Append(a.Name.LocalName).Append("=\"").Append(Escape(a.Value)).Append('"');
            return sb.Append('>').ToString();
        }
        static string SelfClosing(XElement e) {
            var sb = new StringBuilder("<").Append(e.Name.LocalName);
            foreach (var a in e.Attributes()) sb.Append(' ').Append(a.Name.LocalName).Append("=\"").Append(Escape(a.Value)).Append('"');
            return sb.Append(" />").ToString();
        }
        static string Escape(string s) {
            if (string.IsNullOrEmpty(s)) return s ?? "";
            return s.Replace("&","&amp;").Replace("<","&lt;").Replace(">","&gt;").Replace("\"","&quot;");
        }
        static string OpenTagOf(XElement e, string text, ElementSpan sp) {
            return text.Substring(sp.OpenStart, sp.OpenEnd - sp.OpenStart);
        }

        // ---------- text utilities ----------

        /// <summary>Supplies the colName capability test to RecordRebuilder (overridable from the DLL).</summary>
        public Func<string,bool> HasColName = null;

        void Reindex() {
            _idx = ElementIndex.Build(_text);
            _root = XElement.Parse(_text);
        }
        static int LineStart(string s, int pos) {
            int i = s.LastIndexOf('\n', Math.Max(0, pos - 1));
            return i < 0 ? 0 : i + 1;
        }
        static int LineEnd(string s, int pos) {
            int i = s.IndexOf('\n', pos);
            return i < 0 ? s.Length : i + 1;
        }
        static int LineBreakBefore(string s, int pos) {
            int i = pos - 1;
            if (i >= 0 && s[i] == '\n') i--;
            if (i >= 0 && s[i] == '\r') return i;
            return pos;
        }

        static XElement FindElement(XElement root, string path) {
            string[] parts = path.Split('/');
            XElement cur = null;
            foreach (var e in root.DescendantsAndSelf()) {
                if (PathOf(e, root) == path) { cur = e; break; }
            }
            return cur;
        }
        static string PathOf(XElement e, XElement root) {
            var stack = new List<string>();
            for (var c = e; c != null; c = c.Parent) {
                string n = (string)c.Attribute("name");
                stack.Insert(0, n ?? c.Name.LocalName);
                if (c == root) break;
            }
            return string.Join("/", stack.ToArray());
        }

        static int FindAttrValueSpan(string text, ElementSpan el, string attr, out int valStart, out int valEnd) {
            valStart = valEnd = -1;
            string needle = " " + attr + "=\"";
            int p = text.IndexOf(needle, el.OpenStart);
            if (p < 0 || p >= el.OpenEnd) {
                needle = " " + attr + "='";
                p = text.IndexOf(needle, el.OpenStart);
                if (p < 0 || p >= el.OpenEnd) return -1;
            }
            valStart = p + needle.Length;
            char q = text[valStart - 1];
            valEnd = text.IndexOf(q, valStart);
            return p;
        }

        static void FindRecordSpan(string text, out int start, out int end) {
            int tagStart = text.IndexOf("<Record ", StringComparison.Ordinal);
            if (tagStart < 0) { start = end = -1; return; }
            start = LineStart(text, tagStart);
            int formAt = text.IndexOf("<Form ", StringComparison.Ordinal);
            if (formAt < 0) { start = end = -1; return; }
            int lastClose = text.LastIndexOf("</Record>", formAt, StringComparison.Ordinal);
            if (lastClose < 0) { start = end = -1; return; }
            end = lastClose + "</Record>".Length;
            if (end + 1 < text.Length && text[end] == '\r' && text[end + 1] == '\n') end += 2;
            else if (end < text.Length && text[end] == '\n') end += 1;
        }
    }
}
