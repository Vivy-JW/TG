using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;

namespace TzsCli
{
    /// <summary>
    /// Faithful mirror of the designer's record-rebuild pass:
    ///   SpecificationInfo.RebuildScreenRecord / GetScreenRecord
    ///   ScreenRecordManager.AddRecordField / AddRecordFieldTo / GetNewFieldIdRef
    ///
    /// The designer deletes every &lt;Record&gt; and rebuilds it from the ViewModel tree on
    /// every save, renumbering fieldId/fieldIdRef in traversal order. We reproduce that
    /// so generated files are bit-compatible, but we render only the Record section and
    /// splice it into the original text, leaving the (much larger) layout untouched.
    /// </summary>
    public sealed class RecordRebuilder
    {
        /// <summary>Canonical RecordField attribute order, from core-br.spec's BR/RecordField NodeInfo.</summary>
        static readonly string[] RF_ORDER = {
            "name", "fieldType", "sqlTabName", "table_alias_name", "colName", "sqlType",
            "qual1", "precision", "precisionDecimal", "qual2", "scale", "scaleDecimal",
            "qualFraction", "length", "defaultValue", "fieldIdRef"
        };
        static readonly Dictionary<string,string> RF_DEFAULT = new Dictionary<string,string> {
            {"fieldType","NON_DATABASE"}, {"sqlTabName",""}, {"table_alias_name",""}, {"colName",""},
            {"sqlType","CHAR"}, {"qual1","YEAR"}, {"precision","4"}, {"precisionDecimal","16"},
            {"qual2","YEAR"}, {"scale","3"}, {"scaleDecimal","-1"}, {"qualFraction","YEAR"},
            {"length","1"}, {"defaultValue",""}, {"fieldIdRef",""}
        };
        static readonly string[] RECORD_ORDER = {
            "name","active","masterSqlTabName","masterTable","uniqueKey","additionalTables",
            "query","from","where","joinLeft","joinRight","joinOperator","order","recordOrder"
        };

        readonly List<XElement> _records = new List<XElement>();

        /// <summary>Decides whether a component owns a colName property (=> gets a RecordField).</summary>
        public Func<string,bool> HasColName = DefaultHasColName;

        // Authoritative list, taken from core-br.spec + mod-fd.spec: exactly the modFD
        // component types whose NodeInfo declares a "colName" property. Using a positive
        // list matters — a negative one silently admits Button/Text/Spacer and inflates
        // the id sequence.
        static readonly HashSet<string> HAS_COLNAME = new HashSet<string> {
            "ButtonEdit","CheckBox","ComboBox","DateEdit","DateTimeEdit","Edit","FFImage",
            "FFLabel","Field","Phantom","ProgressBar","RadioGroup","RadioGroupWithItem",
            "Slider","SpinEdit","TextEdit","TimeEdit","WebComponent"
        };
        static bool DefaultHasColName(string tag) { return HAS_COLNAME.Contains(tag); }

        // ---------- GetNewFieldIdRef (ScreenRecordManager.cs:167) ----------
        int NewFieldIdRef() {
            var list = new List<int>();
            foreach (var r in _records)
                foreach (var rf in r.Elements("RecordField")) {
                    var a = rf.Attribute("fieldIdRef");
                    if (a != null) { int v; if (int.TryParse(a.Value, out v)) list.Add(v); }
                }
            list.Sort();
            int num = 0;
            foreach (int n2 in list) {
                if (num == 0) num = n2;
                if (n2 - num > 1) return num + 1;
                num = n2;
            }
            return num + 1;
        }
        XElement GetRecordByName(string n) {
            return _records.FirstOrDefault(r => string.Equals((string)r.Attribute("name"), n, StringComparison.OrdinalIgnoreCase));
        }
        XElement FindRecordFieldByName(string name) {
            foreach (var r in _records)
                foreach (var c in r.Elements()) {
                    var a = c.Attribute("name");
                    if (a != null && a.Value == name) return c;
                }
            return null;
        }

        // ---------- ScreenRecordManager.AddRecordField (ScreenRecordManager.cs:29) ----------
        void AddRecordField(XElement el, string parentTag, string parentName) {
            bool parentIsGrid = parentTag == "Table" || parentTag == "Tree" || parentTag == "ScrollGrid";
            if (parentIsGrid) AddRecordFieldTo(el, parentName);
            else if (el.Name.LocalName == "Table" || el.Name.LocalName == "Tree" || el.Name.LocalName == "ScrollGrid") {
                if (GetRecordByName((string)el.Attribute("name")) == null) {
                    var rec = NewRecord();
                    rec.SetAttributeValue("name", ((string)el.Attribute("name") ?? "").ToLower());
                    _records.Add(rec);
                }
            }
            else AddRecordFieldTo(el, "Undefined");

            foreach (var child in el.Elements())
                AddRecordField(child, el.Name.LocalName, (string)el.Attribute("name"));
        }

        // ---------- ScreenRecordManager.AddRecordFieldTo (ScreenRecordManager.cs:59) ----------
        void AddRecordFieldTo(XElement el, string recordName) {
            if (!HasColName(el.Name.LocalName)) return;

            var rec = GetRecordByName(recordName);
            if (rec == null) {
                rec = NewRecord();
                rec.SetAttributeValue("name", recordName == "Undefined" ? recordName : recordName.ToLower());
                _records.Add(rec);
            }

            var nameAttr = (string)el.Attribute("name");
            var rf = FindRecordFieldByName(nameAttr);
            if (rf == null) {
                rf = NewRecordField();
                rf.SetAttributeValue("name", nameAttr);
                el.SetAttributeValue("fieldId", NewFieldIdRef().ToString());
                rec.Add(rf);
            }
            var fid = el.Attribute("fieldId");
            if (fid != null) {
                rf.SetAttributeValue("fieldIdRef", fid.Value);
                rf.SetAttributeValue("fieldType",  (string)el.Attribute("fieldType"));
                rf.SetAttributeValue("sqlTabName", (string)el.Attribute("sqlTabName"));
                rf.SetAttributeValue("colName",    (string)el.Attribute("colName"));
            }
        }

        // ---------- SpecificationInfo.GetScreenRecord (SpecificationInfo.cs:2232) ----------
        void GetScreenRecord(XElement node) {
            foreach (var child in node.Elements()) {
                if (child.Attribute("fieldId") != null)
                    child.SetAttributeValue("fieldId", NewFieldIdRef().ToString());
                AddRecordField(child, node.Name.LocalName, (string)node.Attribute("name"));
                GetScreenRecord(child);
            }
        }

        /// <summary>Mirrors SpecificationInfo.RebuildScreenRecord + SaveToForm's record handling.</summary>
        public List<XElement> Rebuild(XElement formElement) {
            _records.Clear();
            var form = formElement.Element("Form");
            if (form != null) GetScreenRecord(form);
            return _records;
        }

        static XElement NewRecord() {
            var e = new XElement("Record");
            foreach (var p in RECORD_ORDER) e.SetAttributeValue(p, p == "recordOrder" ? "0" : (p == "active" ? "false" : ""));
            return e;
        }
        static XElement NewRecordField() {
            var e = new XElement("RecordField");
            foreach (var p in RF_ORDER) {
                string v; RF_DEFAULT.TryGetValue(p, out v);
                // never pass null: XLinq would drop the attribute, and a later
                // SetAttributeValue would then append it out of canonical order
                e.SetAttributeValue(p, v ?? "");
            }
            return e;
        }

        // ---------- rendering ----------
        /// <summary>Renders the Record section with the .4fd text conventions (CRLF, 2-space indent).</summary>
        public string Render() {
            var sb = new StringBuilder();
            foreach (var r in _records) {
                sb.Append("  ").Append(OpenTag(r)).Append("\r\n");
                foreach (var rf in r.Elements("RecordField"))
                    sb.Append("    ").Append(SelfClosing(rf)).Append("\r\n");
                sb.Append("  </Record>\r\n");
            }
            return sb.ToString();
        }

        static string OpenTag(XElement e) {
            var sb = new StringBuilder("<").Append(e.Name.LocalName);
            foreach (var a in e.Attributes()) sb.Append(' ').Append(a.Name.LocalName).Append("=\"").Append(Esc(a.Value)).Append('"');
            return sb.Append('>').ToString();
        }
        static string SelfClosing(XElement e) {
            var sb = new StringBuilder("<").Append(e.Name.LocalName);
            foreach (var a in e.Attributes()) sb.Append(' ').Append(a.Name.LocalName).Append("=\"").Append(Esc(a.Value)).Append('"');
            return sb.Append(" />").ToString();
        }
        static string Esc(string s) {
            if (string.IsNullOrEmpty(s)) return s ?? "";
            return s.Replace("&","&amp;").Replace("<","&lt;").Replace(">","&gt;").Replace("\"","&quot;");
        }

        public int RecordCount { get { return _records.Count; } }
        public IEnumerable<XElement> Records { get { return _records; } }
    }
}
