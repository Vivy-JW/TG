using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using TzsCli;

class TestWriter
{
    static string Ws = @"D:\t100_wrok_dir\hengshuo\prd";

    static string ReadEntry(string tzs, string suffix) {
        using (var z = ZipFile.OpenRead(tzs))
            foreach (var e in z.Entries)
                if (e.FullName.EndsWith(suffix)) {
                    using (var s = e.Open()) using (var r = new StreamReader(s, Encoding.UTF8)) return r.ReadToEnd();
                }
        return null;
    }

    static void Main() {
        var files = Directory.GetFiles(Ws, "*.tzs").OrderBy(f => f).ToArray();

        // ---- 1. no-op must be byte-identical ----
        int noopOk = 0, noopBad = 0; var badOnes = new List<string>();
        foreach (var f in files) {
            string fd; try { fd = ReadEntry(f, ".4fd"); } catch { continue; }
            if (string.IsNullOrEmpty(fd)) continue;
            string outp;
            try { outp = FormWriter.Load(fd).Render(); }
            catch (Exception ex) { noopBad++; badOnes.Add(Path.GetFileName(f) + " EX " + ex.Message); continue; }
            if (outp == fd) noopOk++; else { noopBad++; badOnes.Add(Path.GetFileName(f)); }
        }
        Console.WriteLine("=== no-op round trip ===");
        Console.WriteLine("  byte-identical : " + noopOk);
        Console.WriteLine("  changed        : " + noopBad);
        foreach (var b in badOnes.Take(5)) Console.WriteLine("      " + b);
        Console.WriteLine();

        // ---- 2. set_attr diff locality ----
        string target = Path.Combine(Ws, "aapp320(c).tzs");
        string src = ReadEntry(target, ".4fd");
        var w = FormWriter.Load(src);
        Console.WriteLine("=== set_attr locality ===");
        var cand = w.Index.All.FirstOrDefault(e =>
            !e.SelfClosing && e.Parent != null && e.Tag == "VBox" && e.Name != null);
        if (cand == null) Console.WriteLine("  no suitable VBox found");
        else {
            string path = cand.Path;
            try {
                w.SetAttribute(path, "hidden", "true");
                string outp = w.Render();
                int n = FirstDiff(src, outp);
                int last = LastDiff(src, outp);
                XElement.Parse(outp);
                Console.WriteLine("  path            : " + path);
                Console.WriteLine("  first diff at   : " + n);
                Console.WriteLine("  last  diff at   : " + last + "   (span " + (last - n) + " bytes)");
                Console.WriteLine("  window          : " + Window(src, outp, n));
                Console.WriteLine("  output parses   : yes");
            } catch (Exception ex) {
                Console.WriteLine("  FAILED " + ex.GetType().Name + ": " + ex.Message);
                Console.WriteLine("  " + (ex.StackTrace ?? "").Split('\n')[0].Trim());
            }
        }
        Console.WriteLine();

        // ---- 3. add_node ----
        Console.WriteLine("=== add_node ===");
        try {
            var w2 = FormWriter.Load(src);
            var parent = w2.Index.All.FirstOrDefault(e => e.Tag == "VBox" && e.Name != null && e.Parent != null);
            if (parent == null) { Console.WriteLine("  no VBox container found"); }
            else {
                string parentPath = parent.Path;
                Console.WriteLine("  parent          : " + parentPath + "  kids=" + parent.Children.Count + " selfClosing=" + parent.SelfClosing);
                var el = w2.CloneTemplate("Edit", new Dictionary<string,string> {
                    {"name","probe_new_edit"}, {"fieldId",null},
                    {"sqlTabName","pmdl_t"}, {"colName","pmdlud001"}, {"fieldType","TABLE_COLUMN"},
                    {"title","lbl_pmdlud001"}, {"comment","cmt_pmdlud001"}
                });
                int before = CountTag(src, "Edit");
                w2.AddNode(parentPath, el);
                string outp = w2.Render();
                int after = CountTag(outp, "Edit");
                var root2 = XElement.Parse(outp);
                bool present = root2.Descendants("Edit").Any(e => (string)e.Attribute("name") == "probe_new_edit");
                var rf = root2.Descendants("RecordField")
                    .FirstOrDefault(r => (string)r.Attribute("name") == "probe_new_edit");
                Console.WriteLine("  Edit count      : " + before + " -> " + after);
                Console.WriteLine("  element present : " + present);
                Console.WriteLine("  RecordField     : " + (rf == null ? "MISSING" : "created, fieldIdRef=" + (string)rf.Attribute("fieldIdRef")
                    + " sqlTabName=" + (string)rf.Attribute("sqlTabName") + " colName=" + (string)rf.Attribute("colName")));
                Console.WriteLine("  output parses   : yes");
                Console.WriteLine("  size delta      : " + (outp.Length - src.Length) + " bytes");
                var idx2 = ElementIndex.Build(outp);
                var inst = idx2.All.FirstOrDefault(e => e.Name == "probe_new_edit");
                if (inst != null) {
                    int ls = outp.LastIndexOf('\n', Math.Max(0, inst.OpenStart - 1)) + 1;
                    int le = outp.IndexOf('\n', inst.OpenEnd);
                    string line = outp.Substring(ls, Math.Max(0, le - ls));
                    Console.WriteLine("  inserted line   : " + line.Substring(0, Math.Min(140, line.Length)) + (line.Length > 140 ? " ..." : ""));
                }
            }
        } catch (Exception ex) {
            Console.WriteLine("  FAILED " + ex.GetType().Name + ": " + ex.Message);
            Console.WriteLine("  " + (ex.StackTrace ?? "").Split('\n')[0].Trim());
        }
    }

    static int CountTag(string s, string tag) {
        int n = 0, i = 0;
        string open = "<" + tag + " ", self = "<" + tag + "/";
        while ((i = s.IndexOf(open, i, StringComparison.Ordinal)) >= 0) { n++; i += open.Length; }
        return n;
    }
    static int FirstDiff(string a, string b) {
        int n = Math.Min(a.Length, b.Length);
        for (int i = 0; i < n; i++) if (a[i] != b[i]) return i;
        return n;
    }
    static int LastDiff(string a, string b) {
        int i = a.Length - 1, j = b.Length - 1;
        while (i >= 0 && j >= 0 && a[i] == b[j]) { i--; j--; }
        return i + 1;
    }
    static string Window(string a, string b, int at) {
        int s = Math.Max(0, at - 70), e = Math.Min(a.Length, at + 70);
        return "\n      A: " + a.Substring(s, e - s).Replace("\r", "\\r").Replace("\n", "\\n")
             + "\n      B: " + b.Substring(s, Math.Min(b.Length - s, e - s)).Replace("\r", "\\r").Replace("\n", "\\n");
    }
}
