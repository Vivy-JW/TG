using System;
using System.Collections.Generic;
using System.IO;
using System.Security.Cryptography;
using System.Text;
using System.Xml.Linq;
using TzsCli;

namespace TzsCli.Designer
{
    /// <summary>
    /// The save tail: Edit.cs:716-741 extracted, with the entry names derived from the input
    /// bytes rather than read out of Main's locals.
    ///
    /// The order is the designer's own -- SaveToTSD() and SaveBinding() rebuild TSDElement and
    /// ElementBindings from the live model, and those two strings, not the ones in the original
    /// archive, are what has to be written back. Only the entries that actually changed are
    /// replaced, so every untouched entry keeps its original compressed bytes and timestamp
    /// (SPEC §11.7); the designer's own PackageManager.Packing cannot do that, which is the
    /// whole reason TzsRepacker exists.
    ///
    /// Returns the repacked bytes as well as writing them, so the caller can report the sizes
    /// ([6] repacked: N -> M bytes) without stat-ing the file it just wrote.
    /// </summary>
    public static class Save
    {
        /// <summary>`write` 是 `dry-run` 的刹车：为假时**照样**把包拼出来并回给调用方（"要写的是什么"
        /// 正是 dry-run 该回答的），只是不落盘。它必须是一个显式参数而不是 Save 自己去读 DryRun.Active
        /// —— 落盘是这个函数唯一的不可逆动作，这件事应当写在调用点上。</summary>
        public static byte[] Run(byte[] original, string outPath, FormWriter w4, Session s, bool write) {
            string fdEntry  = Reflect.EntryName(original, ".4fd");
            string tsdEntry = Reflect.EntryName(original, ".tsd");
            string bdxEntry = Reflect.EntryName(original, ".bdx");

            // A spec-only change is legitimate: can_edit maps to noEntry, and if that attribute
            // already holds the target value the layout text is untouched while the .tsd still
            // has to be rewritten. So this must not be gated on FormWriter.Dirty -- Render()
            // returns the input byte-for-byte when nothing was applied, so the .4fd entry is
            // simply rewritten with identical bytes.
            string fd4New = w4.Render();
            Reflect.Call(s.Si, "SaveToTSD");
            Reflect.Call(s.Si, "SaveBinding");
            string tsdNew = ((XElement)Reflect.Prop(s.Si, "TSDElement")).ToString(SaveOptions.DisableFormatting);
            string bdxNew = Reflect.Prop(s.Tzp, "ElementBindings") as string;

            var repl = new Dictionary<string, byte[]> {
                { fdEntry, Encoding.UTF8.GetBytes(fd4New) },
                { tsdEntry, Encoding.UTF8.GetBytes(tsdNew) }
            };
            // A null or empty ElementBindings means the package has no .bdx to write; the
            // original entry (if any) is then left untouched rather than blanked.
            if (bdxEntry != null && !string.IsNullOrEmpty(bdxNew)) repl[bdxEntry] = Encoding.UTF8.GetBytes(bdxNew);
            byte[] result = TzsRepacker.Repack(original, repl);
            if (write) File.WriteAllBytes(outPath, result);
            return result;
        }

        /// <summary>
        /// The digest of a package's bytes, so a caller can prove WHICH bytes landed on disk
        /// without a second engine call.
        ///
        /// WHY IT EXISTS (SPEC §11.9 item 14; measured by the 2026-09-25 eval, F10). "Prove the
        /// change reached the file" used to cost three engine calls -- save, close, open the new
        /// package, read -- because `get_component` and `verify` both read the in-memory model that
        /// save writes FROM, which makes them circular. With this digest the chain closes in one
        /// shell command: `get_component` shows the change in the model, `save` returns the digest
        /// of what it wrote, and `sha256sum <out>` on the file proves those are the bytes on disk.
        /// Two already-tested invariants hold that together -- save renders the current model
        /// (SPEC §11.24 (b)'s fixed point, which the corpus RoundTrip oracle runs on every package),
        /// and a digest of the written bytes is what it says it is.
        /// </summary>
        public static string Sha256Hex(byte[] bytes) {
            using (SHA256 h = SHA256.Create()) {
                byte[] d = h.ComputeHash(bytes);
                var sb = new StringBuilder(d.Length * 2);
                foreach (byte b in d) sb.Append(b.ToString("x2"));
                return sb.ToString();
            }
        }
    }
}
