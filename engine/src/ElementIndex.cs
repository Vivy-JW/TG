using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Xml.Linq;

namespace TzsCli
{
    /// <summary>
    /// One element occurrence located in the raw .4fd text.
    /// Spans are character offsets into the original string; the writer splices
    /// text rather than re-serialising, so untouched bytes survive verbatim.
    /// </summary>
    public sealed class ElementSpan
    {
        public string Tag;
        public string Name;          // value of the name attribute, if any
        public string Path;          // name-path from the root, e.g. aapp320/mainlayout/condition
        public int Depth;
        public int OpenStart;        // offset of '<'
        public int OpenEnd;          // offset just past the opening tag's '>'
        public int CloseStart;       // offset of '<' of '</Tag>' (or OpenEnd when self-closing)
        public int CloseEnd;         // offset just past '</Tag>'
        public bool SelfClosing;
        public List<ElementSpan> Children = new List<ElementSpan>();
        public ElementSpan Parent;
        public int Indent;           // leading spaces on the element's line

        public override string ToString() { return Tag + " " + Path; }
    }

    /// <summary>
    /// Single-pass scanner over raw .4fd text producing an element table with
    /// byte offsets, so edits can be applied as minimal text splices.
    /// </summary>
    public sealed class ElementIndex
    {
        public string Text;
        public ElementSpan Root;
        public readonly List<ElementSpan> All = new List<ElementSpan>();
        readonly Dictionary<string, ElementSpan> _byPath = new Dictionary<string, ElementSpan>();

        public ElementSpan ByPath(string path) {
            ElementSpan s;
            return _byPath.TryGetValue(path, out s) ? s : null;
        }

        public static ElementIndex Build(string text) {
            var idx = new ElementIndex { Text = text };
            int i = 0, n = text.Length;
            var stack = new Stack<ElementSpan>();

            while (i < n) {
                int lt = text.IndexOf('<', i);
                if (lt < 0) break;
                if (lt + 1 >= n) break;
                char c1 = text[lt + 1];

                if (c1 == '!' || c1 == '?') { i = SkipTo(text, lt, '>') + 1; continue; }
                if (c1 == '/') {                                   // close tag
                    int gt = text.IndexOf('>', lt);
                    if (gt < 0) break;
                    if (stack.Count > 0) {
                        var top = stack.Pop();
                        top.CloseStart = lt; top.CloseEnd = gt + 1;
                        top.SelfClosing = false;
                    }
                    i = gt + 1; continue;
                }

                // opening tag
                int nameEnd = lt + 1;
                while (nameEnd < n && IsNameChar(text[nameEnd])) nameEnd++;
                string tag = text.Substring(lt + 1, nameEnd - lt - 1);
                if (tag.Length == 0) { i = lt + 1; continue; }

                int gt2 = SkipTo(text, lt, '>');
                if (gt2 < 0) break;
                bool selfClose = gt2 > lt && text[gt2 - 1] == '/';

                var el = new ElementSpan {
                    Tag = tag,
                    Depth = stack.Count,
                    OpenStart = lt,
                    OpenEnd = gt2 + 1,
                    CloseStart = gt2 + 1,
                    CloseEnd = gt2 + 1,
                    SelfClosing = selfClose
                };
                el.Name = GetAttr(text, lt, gt2, "name");
                el.Indent = LineIndent(text, lt);

                if (stack.Count > 0) {
                    el.Parent = stack.Peek();
                    el.Parent.Children.Add(el);
                    el.Path = el.Parent.Path + "/" + (el.Name ?? tag);
                } else {
                    el.Path = el.Name ?? tag;
                    idx.Root = el;
                }

                idx.All.Add(el);
                if (idx.ByPath(el.Path) == null) idx._byPath[el.Path] = el;

                if (selfClose) { i = gt2 + 1; }
                else { stack.Push(el); i = gt2 + 1; }
            }
            return idx;
        }

        static bool IsNameChar(char c) {
            return !(c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '>' || c == '/');
        }
        static int SkipTo(string s, int from, char target) {
            int q = -1;
            for (int i = from; i < s.Length; i++) {
                char c = s[i];
                if (q > 0) { if (c == q) q = -1; continue; }
                if (c == '"' || c == '\'') { q = c; continue; }
                if (c == target) return i;
            }
            return -1;
        }
        static string GetAttr(string s, int tagStart, int tagEnd, string attr) {
            string needle = attr + "=";
            int q = -1;
            for (int i = tagStart; i < tagEnd; i++) {
                char c = s[i];
                if (q > 0) { if (c == q) q = -1; continue; }
                if (c == '"' || c == '\'') { q = c; continue; }
                if (c == ' ' || c == '\t' || c == '\r' || c == '\n') {
                    if (string.CompareOrdinal(s, i + 1, needle, 0, needle.Length) == 0) {
                        int vs = i + 1 + needle.Length;
                        if (vs < s.Length && (s[vs] == '"' || s[vs] == '\'')) {
                            char vq = s[vs]; int ve = s.IndexOf(vq, vs + 1);
                            if (ve > 0) return s.Substring(vs + 1, ve - vs - 1);
                        }
                    }
                }
            }
            return null;
        }
        static int LineIndent(string s, int pos) {
            int ls = s.LastIndexOf('\n', Math.Max(0, pos - 1));
            int k = ls + 1, sp = 0;
            while (k < s.Length && s[k] == ' ') { sp++; k++; }
            return sp;
        }
    }
}
