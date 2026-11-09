using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Linq;
using System.Reflection;
using System.Text;
using TzsCli;

/// <summary>
/// Acceptance for the repacker: the designer's own zip library must read the output,
/// entry contents must survive, and SpecDesignerCommon must accept the container.
/// </summary>
class VerifyRepack
{
    // TZSCLI_INSTALL overrides the bundled path -- see Designer.Install. These probes build
    // into engine/out/, which is not a package and carries no designer, so they fall back to
    // an installed designer. Not `const` because it comes from the environment.
    static readonly string INSTALL = InstallFromEnv();

    static string InstallFromEnv() {
        string v = Environment.GetEnvironmentVariable("TZSCLI_INSTALL");
        if (!string.IsNullOrEmpty(v)) return v;
        // 随仓库自带的那份：build.sh 把 engine\designer\ 采到 out\designer\，
        // 与发行包 <引擎目录>\designer\ 是同一个布局。没有硬编码缺省 —— 见 engine/BUILD.md。
        return Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "designer");
    }
    const string WS = @"D:\t100_wrok_dir\hengshuo\prd";

    static void Main() {
        string src = Path.Combine(WS, "apmt500_wf(c).tzs");
        byte[] orig = File.ReadAllBytes(src);

        // Replace the .tsd with a real edit: append one field, which is what a CLI would do.
        string tsd = ReadText(orig, "apmt500_wf.tsd");
        string tsdNew = tsd.Replace("</spec>",
            "  <field src=\"c\" ver=\"4\" column=\"\" name=\"__probe__\" table=\"\" attribute=\"\" type=\"\" req=\"\" i_zoom=\"\" c_zoom=\"\" chk_ref=\"\" items=\"\" default=\"\" max=\"\" min=\"\" can_edit=\"Y\" can_query=\"Y\" widget=\"\" cite_std=\"N\" status=\"u\" />\r\n</spec>");
        byte[] outp = TzsRepacker.Repack(orig, new Dictionary<string, byte[]> {
            { "apmt500_wf.tsd", Encoding.UTF8.GetBytes(tsdNew) }
        });
        string outPath = Path.Combine(Path.GetTempPath(), "repack_verify.tzs");
        File.WriteAllBytes(outPath, outp);
        Console.WriteLine("repacked " + orig.Length + " -> " + outp.Length + " bytes  vis " + outPath);
        Console.WriteLine();

        // ---- 1. the designer's own zip library must read it, via the same entry point
        //         PackageManager.Unpacking uses (ZipInputStream over the raw file) ----
        Console.WriteLine("=== SharpZipLib ZipInputStream (PackageManager.Unpacking's own reader) ===");
        var szAsm = Assembly.LoadFrom(Path.Combine(INSTALL, "ICSharpCode.SharpZipLib.dll"));
        Type zisT = szAsm.GetType("ICSharpCode.SharpZipLib.Zip.ZipInputStream");
        Type zeT = szAsm.GetType("ICSharpCode.SharpZipLib.Zip.ZipEntry");
        object zis = Activator.CreateInstance(zisT, new object[] { File.OpenRead(outPath) });
        var mine = new Dictionary<string, string>();
        while (true) {
            object e = zisT.GetMethod("GetNextEntry").Invoke(zis, null);
            if (e == null) break;
            if (!(bool)zeT.GetProperty("IsFile").GetValue(e, null)) continue;
            string nm = (string)zeT.GetProperty("Name").GetValue(e, null);
            // leaveOpen: the reader must not close the ZipInputStream between entries.
            // (The designer's own Unpacking simply never disposes its reader.)
            var sr = new StreamReader((Stream)zis, Encoding.UTF8, true, 4096, true);
            mine[nm] = sr.ReadToEnd();
            Console.WriteLine("    read " + nm.PadRight(24) + " " + mine[nm].Length + " chars");
        }
        ((IDisposable)zis).Dispose();

        // ---- 2. contents must survive ----
        Console.WriteLine();
        Console.WriteLine("=== content check ===");
        var origNames = new List<string>();
        using (var ms = new MemoryStream(orig))
        using (var za = new ZipArchive(ms, ZipArchiveMode.Read))
            foreach (var e in za.Entries) origNames.Add(e.FullName);
        Console.WriteLine("  entry order preserved: " + origNames.SequenceEqual(mine.Keys));
        int same = 0, changed = 0;
        foreach (var n in origNames) {
            string before = ReadText(orig, n);
            if (mine[n] == before) same++; else { changed++; Console.WriteLine("    CHANGED: " + n); }
        }
        Console.WriteLine("  entries identical: " + same + "   changed: " + changed + "   (expect exactly 1)");
        Console.WriteLine("  edited .tsd round-trips exactly: " + (mine["apmt500_wf.tsd"] == tsdNew));
        Console.WriteLine("  growth in .tsd content: " + (tsdNew.Length - tsd.Length) + " chars");

        // ---- 3. SpecDesignerCommon must accept the container ----
        Console.WriteLine();
        Console.WriteLine("=== SpecDesignerCommon.PackageManager.SeekReleaseVersion ===");
        var zz = Assembly.LoadFrom(Path.Combine(INSTALL, "SpecDesignerCommon.dll"));
        Type pmT = zz.GetType("SpecDesignerCommon.PackageManager");
        object ver = pmT.GetMethod("SeekReleaseVersion", BindingFlags.Public | BindingFlags.Static)
            .Invoke(null, new object[] { outPath });
        Console.WriteLine("  ver entry read back as: " + ver);
    }

    static string ReadText(byte[] zip, string name) {
        using (var ms = new MemoryStream(zip))
        using (var z = new ZipArchive(ms, ZipArchiveMode.Read))
        using (var r = new StreamReader(z.GetEntry(name).Open(), Encoding.UTF8))
            return r.ReadToEnd();
    }
}
