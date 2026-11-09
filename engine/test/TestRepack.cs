using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Text;
using TzsCli;

class TestRepack
{
    static string Ws = @"D:\t100_wrok_dir\hengshuo\prd";

    static void Main() {
        var files = Directory.GetFiles(Ws, "*.tzs").OrderBy(f => f).ToArray();

        // ---- 1. no replacements must be byte-identical ----
        int ok = 0, bad = 0; var badOnes = new List<string>(); string firstBad = null;
        foreach (var f in files) {
            byte[] src; try { src = File.ReadAllBytes(f); } catch { continue; }
            byte[] outp;
            try { outp = TzsRepacker.Repack(src, null); }
            catch (Exception ex) { bad++; badOnes.Add(Path.GetFileName(f) + " EX " + ex.Message); continue; }
            if (outp.Length == src.Length && Same(src, outp)) ok++;
            else { bad++; if (badOnes.Count == 0) firstBad = f;
                   badOnes.Add(Path.GetFileName(f) + " len " + src.Length + "->" + outp.Length); }
        }
        Console.WriteLine("=== no-op repack ===");
        Console.WriteLine("  byte-identical : " + ok);
        Console.WriteLine("  changed        : " + bad);
        foreach (var b in badOnes.Take(6)) Console.WriteLine("      " + b);
        if (bad > 0) {
            byte[] s0 = File.ReadAllBytes(firstBad ?? files[0]);
            byte[] o0 = TzsRepacker.Repack(s0, null);
            int at = FirstDiff(s0, o0);
            Console.WriteLine("  first diff in " + Path.GetFileName(firstBad ?? files[0]) + " at " + at
                + "   len orig=" + s0.Length + " mine=" + o0.Length + "  (delta " + (o0.Length - s0.Length) + ")");
            int a = Math.Max(0, at - 32), z = Math.Min(s0.Length, at + 48);
            Console.WriteLine("      orig: " + BitConverter.ToString(s0, a, z - a));
            Console.WriteLine("      mine: " + BitConverter.ToString(o0, a, Math.Min(o0.Length - a, z - a)));
            // raw 46-byte central header of entry 0, side by side
            int cd0 = FindCd(s0), cd1 = FindCd(o0);
            Console.WriteLine("      CD hdr of entry 0");
            Console.WriteLine("        orig: " + BitConverter.ToString(s0, cd0, 46));
            Console.WriteLine("        mine: " + BitConverter.ToString(o0, cd1, 46));
            var e0 = TzsRepacker.ReadEntries(s0);
            var e1 = TzsRepacker.ReadEntries(o0);
            Console.WriteLine("      exit code check: eocd orig=" + (s0.Length - 22 - EocdOff(s0)) + " tailBytes"
                + "   cdStart orig=" + CdOff(s0) + " mine=" + CdOff(o0));
            Console.WriteLine("      entry count orig=" + e0.Count + " mine=" + e1.Count);
            for (int i = 0; i < e0.Count; i++)
                Console.WriteLine("      entry " + e0[i].Name + "  flags " + e0[i].Flags + "/" + e1[i].Flags
                    + "  made " + e0[i].VersionMadeBy + "/" + e1[i].VersionMadeBy
                    + "  need " + e0[i].VersionNeeded + "/" + e1[i].VersionNeeded
                    + "  extAttrs " + e0[i].ExternalAttrs + "/" + e1[i].ExternalAttrs
                    + "  cdExtraLen " + (e0[i].CdExtra == null ? -1 : e0[i].CdExtra.Length)
                    + "/" + (e1[i].CdExtra == null ? -1 : e1[i].CdExtra.Length));
        }
        Console.WriteLine();

        // ---- 2. single-entry replacement locality ----
        string target = Path.Combine(Ws, "apmt500_wf(c).tzs");
        byte[] orig = File.ReadAllBytes(target);
        var before = TzsRepacker.ReadEntries(orig);
        Console.WriteLine("=== single-entry replacement (" + Path.GetFileName(target) + ") ===");
        Console.WriteLine("  entries: " + string.Join(", ", before.Select(e => e.Name + "(" + e.CompSize + ")")));

        // replace .tsd with a modified copy (append one dummy field, then strip it back)
        string tsdText;
        using (var z = ZipFile.OpenRead(target))
        using (var r = new StreamReader(z.GetEntry("apmt500_wf.tsd").Open(), Encoding.UTF8))
            tsdText = r.ReadToEnd();
        string tsdMod = tsdText.Replace("</spec>", "  <field name=\"__probe__\" status=\"u\" />\r\n</spec>");
        Console.WriteLine("  .tsd " + tsdText.Length + " -> " + tsdMod.Length + " chars");

        byte[] outp2 = TzsRepacker.Repack(orig, new Dictionary<string, byte[]> {
            { "apmt500_wf.tsd", Encoding.UTF8.GetBytes(tsdMod) }
        });
        var after = TzsRepacker.ReadEntries(outp2);
        Console.WriteLine("  size " + orig.Length + " -> " + outp2.Length + "  (delta " + (outp2.Length - orig.Length) + ")");
        Console.WriteLine("  entry order preserved: " + before.Select(e => e.Name).SequenceEqual(after.Select(e => e.Name)));

        // untouched entries must be byte-identical spans
        var replNames = new HashSet<string> { "apmt500_wf.tsd" };
        int sameSpans = 0, diffSpans = 0;
        for (int i = 0; i < before.Count; i++) {
            var b = before[i]; var a = after[i];
            if (replNames.Contains(b.Name)) continue;
            // locate the entry in the output the same way and compare raw bytes
            bool identical = SpanEqual(outp2, a, orig, b);
            if (identical) sameSpans++; else { diffSpans++; Console.WriteLine("      UNTOUCHED ENTRY CHANGED: " + b.Name); }
        }
        Console.WriteLine("  untouched entries byte-identical: " + sameSpans + "   changed: " + diffSpans);

        // changed entry must round-trip to the new content
        string back;
        using (var ms = new MemoryStream(outp2))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry("apmt500_wf.tsd").Open(), Encoding.UTF8))
            back = r.ReadToEnd();
        Console.WriteLine("  replaced entry round-trips: " + (back == tsdMod));

        // timestamps of untouched entries must survive
        int tsSame = 0, tsDiff = 0;
        for (int i = 0; i < before.Count; i++) {
            if (replNames.Contains(before[i].Name)) continue;
            if (before[i].DosTime == after[i].DosTime && before[i].DosDate == after[i].DosDate) tsSame++; else tsDiff++;
        }
        Console.WriteLine("  untouched entry timestamps preserved: " + tsSame + "   changed: " + tsDiff);

        // ---- 3. hand a repacked file to the original DLL ----
        string outPath = Path.Combine(Path.GetTempPath(), "repacked_probe.tzs");
        byte[] clean = TzsRepacker.Repack(orig, new Dictionary<string, byte[]> {
            { "apmt500_wf.4fd", Encoding.UTF8.GetBytes(ReadEntryText(orig, "apmt500_wf.4fd")) }   // same content, re-deflated
        });
        File.WriteAllBytes(outPath, clean);
        Console.WriteLine();
        Console.WriteLine("=== wrote " + outPath + " (" + clean.Length + " bytes) for DLL verification ===");
    }

    static string ReadEntryText(byte[] zip, string name) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
            return r.ReadToEnd();
    }
    static int EocdOff(byte[] d) {
        int i = d.Length - 22;
        while (i > 0 && !(d[i]==0x50&&d[i+1]==0x4b&&d[i+2]==0x05&&d[i+3]==0x06)) i--;
        return i;
    }
    static int CdOff(byte[] d) {
        int i = EocdOff(d);
        return (int)(d[i+16] | (d[i+17]<<8) | (d[i+18]<<16) | (d[i+19]<<24));
    }
    static int FindCd(byte[] d) {
        int i = d.Length - 22;
        while (i > 0 && !(d[i]==0x50&&d[i+1]==0x4b&&d[i+2]==0x05&&d[i+3]==0x06)) i--;
        return (int)(d[i+16] | (d[i+17]<<8) | (d[i+18]<<16) | (d[i+19]<<24));
    }
    static int FirstDiff(byte[] a, byte[] b) {
        int n = Math.Min(a.Length, b.Length);
        for (int i = 0; i < n; i++) if (a[i] != b[i]) return i;
        return n;
    }
    static bool Same(byte[] a, byte[] b) {
        if (a.Length != b.Length) return false;
        for (int i = 0; i < a.Length; i++) if (a[i] != b[i]) return false;
        return true;
    }
    static bool SpanEqual(byte[] zipA, ZipEntryRec a, byte[] zipB, ZipEntryRec b) {
        if (a.RawLength != b.RawLength) return false;
        for (int i = 0; i < a.RawLength; i++) if (zipA[a.RawStart + i] != zipB[b.RawStart + i]) return false;
        return true;
    }
}
