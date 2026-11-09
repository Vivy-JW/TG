using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Text;
using System.Xml.Linq;
using TzsCli;

class TestRebuild
{
    static string Ws = @"D:\t100_wrok_dir\hengshuo\prd";

    static string ReadEntry(string tzs, string suffix) {
        using (var z = ZipFile.OpenRead(tzs))
            foreach (var e in z.Entries)
                if (e.FullName.EndsWith(suffix)) {
                    using (var s = e.Open()) using (var r = new StreamReader(s, Encoding.UTF8))
                        return r.ReadToEnd();
                }
        return null;
    }

    /// <summary>Span of the contiguous Record section in the raw text, including the
    /// leading indent of the first line and the trailing CRLF after the last close tag.
    /// The section is contiguous and sits immediately before &lt;Form&gt;, so anchoring on
    /// &lt;Form&gt; and scanning backwards is both simpler and more robust than depth counting.</summary>
    static void RecordSpan(string text, out int start, out int end) {
        int tagStart = text.IndexOf("<Record ", StringComparison.Ordinal);
        if (tagStart < 0) { start = end = -1; return; }
        int ls = text.LastIndexOf('\n', Math.Max(0, tagStart - 1));
        start = ls + 1;

        int formAt = text.IndexOf("<Form ", StringComparison.Ordinal);
        if (formAt < 0) { start = end = -1; return; }
        int lastClose = text.LastIndexOf("</Record>", formAt, StringComparison.Ordinal);
        if (lastClose < 0) { start = end = -1; return; }
        end = lastClose + "</Record>".Length;
        if (end + 1 < text.Length && text[end] == '\r' && text[end + 1] == '\n') end += 2;
        else if (end < text.Length && text[end] == '\n') end += 1;
    }

    /// <summary>Verifies the text scanner agrees with the XML parser on element count,
    /// and that every scanned span is a plausible slice of the source text.</summary>
    static void IndexSanity() {
        var files = Directory.GetFiles(Ws, "*.tzs").OrderBy(f => f).ToArray();
        int checkedFiles = 0, countMismatch = 0, spanBad = 0, pathDup = 0;
        foreach (var f in files) {
            string fd; try { fd = ReadEntry(f, ".4fd"); } catch { continue; }
            if (string.IsNullOrEmpty(fd)) continue;
            checkedFiles++;

            var idx = ElementIndex.Build(fd);

            int xmlCount = 0;
            try { var root = XElement.Parse(fd); xmlCount = 1 + root.Descendants().Count(); }
            catch { continue; }
            if (idx.All.Count != xmlCount) countMismatch++;

            foreach (var e in idx.All) {
                if (e.OpenStart < 0 || e.OpenEnd > fd.Length || e.CloseEnd > fd.Length) { spanBad++; continue; }
                if (fd[e.OpenStart] != '<') spanBad++;
                if (!e.SelfClosing && (e.CloseStart < 0 || e.CloseStart >= fd.Length || fd[e.CloseStart] != '<')) spanBad++;
                if (!e.SelfClosing && !fd.Substring(e.CloseStart, Math.Min(8, fd.Length - e.CloseStart)).StartsWith("</")) spanBad++;
            }
            var seen = new HashSet<string>();
            foreach (var e in idx.All) if (!seen.Add(e.Path)) pathDup++;
        }
        Console.WriteLine("ElementIndex sanity over " + checkedFiles + " files");
        Console.WriteLine("  element-count mismatches vs XML parser : " + countMismatch);
        Console.WriteLine("  malformed spans                        : " + spanBad);
        Console.WriteLine("  duplicate name-paths                   : " + pathDup);
        Console.WriteLine();
    }

    static void Main(string[] args) {
        IndexSanity();
        var files = Directory.GetFiles(Ws, "*.tzs").OrderBy(f => f).ToArray();
        Console.WriteLine("scanning " + files.Length + " .tzs files\n");

        int ok = 0, diff = 0, skip = 0;
        var failures = new List<string>();

        foreach (var f in files) {
            string fd;
            try { fd = ReadEntry(f, ".4fd"); } catch { skip++; continue; }
            if (string.IsNullOrEmpty(fd)) { skip++; continue; }

            int rs, re;
            RecordSpan(fd, out rs, out re);
            if (rs < 0) { skip++; continue; }
            string original = fd.Substring(rs, re - rs);

            string rendered;
            try {
                var root = XElement.Parse(fd);
                var rb = new RecordRebuilder();
                rb.Rebuild(root);
                rendered = rb.Render();
            } catch (Exception ex) {
                skip++; failures.Add(Path.GetFileName(f) + "  EXCEPTION " + ex.Message);
                continue;
            }

            if (original == rendered) ok++;
            else {
                diff++;
                if (failures.Count < 6) {
                    int k = 0; int lim = Math.Min(original.Length, rendered.Length);
                    while (k < lim && original[k] == rendered[k]) k++;
                    failures.Add(string.Format("{0}  first diff at {1}/{2}\n      orig: {3}\n      mine: {4}",
                        Path.GetFileName(f), k, original.Length,
                        Show(original, k), Show(rendered, k)));
                }
            }
        }

        Console.WriteLine("byte-identical Record section : " + ok);
        Console.WriteLine("differing                     : " + diff);
        Console.WriteLine("skipped (no .4fd / parse fail): " + skip);
        Console.WriteLine();

        // Classify: a file is "current format" when its first Record uses the canonical
        // attribute order emitted by today's ComponentFactory.CreateRecord. Files the
        // designer last saved with an older version use alphabetical order + a uid
        // attribute, and today's designer would normalise them too on any save.
        int oldFmt = 0, oldFmtMatch = 0;
        var remaining = new List<string>();
        foreach (var f in files) {
            string fd; try { fd = ReadEntry(f, ".4fd"); } catch { continue; }
            if (string.IsNullOrEmpty(fd)) continue;
            int rs, re; RecordSpan(fd, out rs, out re);
            if (rs < 0) continue;
            string orig = fd.Substring(rs, re - rs);
            string rendered;
            try { var root = XElement.Parse(fd); var rb = new RecordRebuilder(); rb.Rebuild(root); rendered = rb.Render(); }
            catch { continue; }
            bool oldFormat = orig.StartsWith("  <Record additionalTables=", StringComparison.Ordinal);
            if (oldFormat) { oldFmt++; if (orig == rendered) oldFmtMatch++; }
            else if (orig != rendered && remaining.Count < 8) remaining.Add(Path.GetFileName(f));
        }
        Console.WriteLine("pre-current-format files (alphabetical <Record> attrs): " + oldFmt
                          + "  (of which identical: " + oldFmtMatch + ")");
        Console.WriteLine("current-format files that STILL differ: " + remaining.Count);
        foreach (var r in remaining) Console.WriteLine("    " + r);
        Console.WriteLine();
        foreach (var s in failures) Console.WriteLine(s + "\n");
    }

    static string Show(string s, int at) {
        int a = Math.Max(0, at - 60), b = Math.Min(s.Length, at + 90);
        string seg = s.Substring(a, b - a);
        return seg.Replace("\r", "\\r").Replace("\n", "\\n");
    }
}
