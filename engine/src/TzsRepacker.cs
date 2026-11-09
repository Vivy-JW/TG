using System;
using System.Collections.Generic;
using System.IO;
using System.IO.Compression;
using System.Text;

namespace TzsCli
{
    /// <summary>One entry of a .tzs container, as located in the raw bytes.</summary>
    public sealed class ZipEntryRec
    {
        public string Name;
        public int Flags, Method, DosTime, DosDate;
        public long Crc32, CompSize, UncompSize;
        public int ExternalAttrs, VersionMadeBy, VersionNeeded;
        public int LocalOffset;          // offset of the local file header
        public int LocalHeaderLen;       // 30 + nameLen + extraLen
        public int ExtraLen;
        public byte[] LocalExtra;        // preserved verbatim so mtime survives a rewrite
        public byte[] CdExtra;           // central-directory extra, preserved for untouched entries
        public byte[] CdHeaderRaw;       // the whole 46-byte central header, copied then patched

        public bool IsChanged;

        /// <summary>Byte span of this entry in the source archive: local header + data.</summary>
        public int RawStart { get { return LocalOffset; } }
        public int RawLength { get { return LocalHeaderLen + (int)CompSize; } }
    }

    /// <summary>
    /// Repacks a .tzs from raw bytes, replacing only the named entries.
    ///
    /// Unchanged entries are copied verbatim (local header included), so their compressed
    /// bytes and timestamps survive exactly and a VCS diff shows only the entries that
    /// really changed. The designer's own PackageManager.Packing re-deflates everything at
    /// level 3 and stamps DateTime.Now on every entry, which is precisely what this avoids.
    /// </summary>
    public static class TzsRepacker
    {
        const uint LFH_SIG = 0x04034b50;
        const uint CDH_SIG = 0x02014b50;
        const uint EOCD_SIG = 0x06054b50;

        static uint U16(byte[] b, int i) { return (uint)(b[i] | (b[i + 1] << 8)); }
        static uint U32(byte[] b, int i) { return (uint)(b[i] | (b[i + 1] << 8) | (b[i + 2] << 16) | (b[i + 3] << 24)); }

        public static List<ZipEntryRec> ReadEntries(byte[] zip) {
            int eocd = -1;
            for (int i = zip.Length - 22; i >= 0 && i > zip.Length - 66000; i--)
                if (U32(zip, i) == EOCD_SIG) { eocd = i; break; }
            if (eocd < 0) throw new InvalidDataException("EOCD not found");

            uint count = U16(zip, eocd + 10);
            uint cdSize = U32(zip, eocd + 12);
            uint cdOff = U32(zip, eocd + 16);

            var list = new List<ZipEntryRec>();
            int p = (int)cdOff;
            for (int n = 0; n < count; n++) {
                if (U32(zip, p) != CDH_SIG) throw new InvalidDataException("bad central directory at " + p);
                var e = new ZipEntryRec {
                    VersionMadeBy = (int)U16(zip, p + 4),
                    VersionNeeded = (int)U16(zip, p + 6),
                    Flags = (int)U16(zip, p + 8),
                    Method = (int)U16(zip, p + 10),
                    DosTime = (int)U16(zip, p + 12),
                    DosDate = (int)U16(zip, p + 14),
                    Crc32 = U32(zip, p + 16),
                    CompSize = U32(zip, p + 20),
                    UncompSize = U32(zip, p + 24),
                    ExternalAttrs = (int)U32(zip, p + 38),
                    LocalOffset = (int)U32(zip, p + 42)
                };
                int nameLen = (int)U16(zip, p + 28);
                int extraLen = (int)U16(zip, p + 30);
                int cmtLen = (int)U16(zip, p + 32);
                e.Name = Encoding.UTF8.GetString(zip, p + 46, nameLen);
                e.CdExtra = new byte[extraLen];
                Array.Copy(zip, p + 46 + nameLen, e.CdExtra, 0, extraLen);
                e.CdHeaderRaw = new byte[46];
                Array.Copy(zip, p, e.CdHeaderRaw, 0, 46);

                // SharpZipLib emits ZIP64 always, even for tiny archives: the 32-bit size
                // and offset fields hold 0xFFFFFFFF and the real values live in a 0x0001
                // extra field, in the order uncompressed / compressed / localOffset / disk.
                if (e.CompSize == 0xFFFFFFFFL || e.UncompSize == 0xFFFFFFFFL || e.LocalOffset == 0xFFFFFFFFL) {
                    int ex = p + 46 + nameLen;
                    int end = ex + extraLen;
                    while (ex + 4 <= end) {
                        int id = (int)U16(zip, ex);
                        int sz = (int)U16(zip, ex + 2);
                        int dp = ex + 4;
                        if (id == 0x0001) {
                            if (e.UncompSize == 0xFFFFFFFFL && dp + 8 <= end) { e.UncompSize = ReadU64(zip, dp); dp += 8; }
                            if (e.CompSize == 0xFFFFFFFFL && dp + 8 <= end) { e.CompSize = ReadU64(zip, dp); dp += 8; }
                            if (e.LocalOffset == 0xFFFFFFFFL && dp + 8 <= end) { e.LocalOffset = (int)ReadU64(zip, dp); dp += 8; }
                        }
                        ex += 4 + sz;
                    }
                }

                int lp = e.LocalOffset;
                if (U32(zip, lp) != LFH_SIG) throw new InvalidDataException("bad local header for " + e.Name);
                int lNameLen = (int)U16(zip, lp + 26);
                e.ExtraLen = (int)U16(zip, lp + 28);
                e.LocalHeaderLen = 30 + lNameLen + e.ExtraLen;
                e.LocalExtra = new byte[e.ExtraLen];
                Array.Copy(zip, lp + 30 + lNameLen, e.LocalExtra, 0, e.ExtraLen);

                list.Add(e);
                p += 46 + nameLen + extraLen + cmtLen;
            }
            return list;
        }

        /// <summary>Returns the repacked archive. With no replacements, the result equals the input.</summary>
        public static byte[] Repack(byte[] original, IDictionary<string, byte[]> replacements) {
            var entries = ReadEntries(original);
            var outMs = new MemoryStream();
            var offsets = new List<int>();

            foreach (var e in entries) {
                offsets.Add((int)outMs.Position);
                byte[] repl;
                if (replacements != null && replacements.TryGetValue(e.Name, out repl)) {
                    e.IsChanged = true;
                    WriteLocalHeader(outMs, e, repl);
                } else {
                    outMs.Write(original, e.RawStart, e.RawLength);   // verbatim
                }
            }

            long cdStart = outMs.Position;
            for (int i = 0; i < entries.Count; i++) WriteCentralHeader(outMs, entries[i], offsets[i]);

            long cdSize = outMs.Position - cdStart;

            // Preserve the original EOCD verbatim and patch only the three fields that move,
            // so anything else it carries (comment, disk numbering) survives.
            int eocd = FindEocd(original);
            var tail = new byte[original.Length - eocd];
            Array.Copy(original, eocd, tail, 0, tail.Length);
            PutU16(tail, 8, (uint)entries.Count);
            PutU16(tail, 10, (uint)entries.Count);
            PutU32(tail, 12, (uint)cdSize);
            PutU32(tail, 16, (uint)cdStart);
            outMs.Write(tail, 0, tail.Length);
            return outMs.ToArray();
        }

        static int FindEocd(byte[] zip) {
            for (int i = zip.Length - 22; i >= 0 && i > zip.Length - 66000; i--)
                if (U32(zip, i) == EOCD_SIG) return i;
            throw new InvalidDataException("EOCD not found");
        }

        static long ReadU64(byte[] b, int i) {
            return (long)(U32(b, i) | ((ulong)U32(b, i + 4) << 32));
        }

        /// <summary>Drops a ZIP64 (0x0001) record from an extra-field blob. A rewritten entry
        /// carries plain 32-bit sizes, so keeping the stale ZIP64 record would contradict them.</summary>
        static byte[] StripZip64(byte[] extra, out int newLen) {
            newLen = 0;
            if (extra == null || extra.Length == 0) return new byte[0];
            var keep = new List<byte>(extra.Length);
            int p = 0;
            while (p + 4 <= extra.Length) {
                int id = (int)U16(extra, p);
                int sz = (int)U16(extra, p + 2);
                if (p + 4 + sz > extra.Length) break;
                if (id != 0x0001) for (int k = 0; k < 4 + sz; k++) keep.Add(extra[p + k]);
                p += 4 + sz;
            }
            newLen = keep.Count;
            return keep.ToArray();
        }

        static void WriteLocalHeader(Stream s, ZipEntryRec e, byte[] content) {
            // Re-deflate the replacement to raw deflate (ZIP method 8)
            byte[] comp;
            if (e.Method == 0) comp = content;
            else {
                using (var ms = new MemoryStream()) {
                    using (var ds = new DeflateStream(ms, CompressionMode.Compress, true)) ds.Write(content, 0, content.Length);
                    comp = ms.ToArray();
                }
            }
            e.Crc32 = Crc32(content);
            e.UncompSize = content.Length;
            e.CompSize = comp.Length;

            byte[] name = Encoding.UTF8.GetBytes(e.Name);
            int extraLen;
            byte[] extra = StripZip64(e.LocalExtra, out extraLen);
            var hdr = new byte[30 + name.Length + extraLen];
            PutU32(hdr, 0, LFH_SIG);
            PutU16(hdr, 4, (uint)e.VersionNeeded);
            PutU16(hdr, 6, (uint)(e.Flags & ~0x08));      // clear "sizes in data descriptor"
            PutU16(hdr, 8, (uint)e.Method);
            PutU16(hdr, 10, (uint)e.DosTime);
            PutU16(hdr, 12, (uint)e.DosDate);
            PutU32(hdr, 14, (uint)e.Crc32);
            PutU32(hdr, 18, (uint)e.CompSize);
            PutU32(hdr, 22, (uint)e.UncompSize);
            PutU16(hdr, 26, (uint)name.Length);
            PutU16(hdr, 28, (uint)extraLen);
            Array.Copy(name, 0, hdr, 30, name.Length);
            if (extraLen > 0) Array.Copy(extra, 0, hdr, 30 + name.Length, extraLen);
            s.Write(hdr, 0, hdr.Length);
            s.Write(comp, 0, comp.Length);
        }

        static void WriteCentralHeader(Stream s, ZipEntryRec e, int offset) {
            byte[] name = Encoding.UTF8.GetBytes(e.Name);
            byte[] extra;
            int extraLen;
            // untouched entries keep their central-directory extra verbatim (ZIP64 and all);
            // rewritten ones get plain 32-bit sizes, so a stale ZIP64 record must go
            if (e.IsChanged) { extra = StripZip64(e.CdExtra, out extraLen); }
            else { extra = e.CdExtra ?? new byte[0]; extraLen = extra.Length; }

            // Start from the original 46-byte header and patch only what actually moved.
            // SharpZipLib writes ZIP64 even for small archives: the 32-bit size fields hold
            // 0xFFFFFFFF and the real values sit in the extra field. Recomputing them here
            // would be valid ZIP but would not be byte-identical, so untouched entries keep
            // their placeholders and only the (possibly shifted) local offset is rewritten.
            var hdr = new byte[46 + name.Length + extraLen];
            Array.Copy(e.CdHeaderRaw, 0, hdr, 0, 46);
            if (e.IsChanged) {
                PutU32(hdr, 16, (uint)e.Crc32);
                PutU32(hdr, 20, (uint)e.CompSize);
                PutU32(hdr, 24, (uint)e.UncompSize);
                PutU16(hdr, 30, (uint)extraLen);
            }
            PutU32(hdr, 42, (uint)offset);
            Array.Copy(name, 0, hdr, 46, name.Length);
            if (extraLen > 0) Array.Copy(extra, 0, hdr, 46 + name.Length, extraLen);
            s.Write(hdr, 0, hdr.Length);
        }

        static void WriteEocd(Stream s, int count, int cdSize, int cdOffset) {
            var b = new byte[22];
            PutU32(b, 0, EOCD_SIG);
            PutU16(b, 8, (uint)count);
            PutU16(b, 10, (uint)count);
            PutU32(b, 12, (uint)cdSize);
            PutU32(b, 16, (uint)cdOffset);
            s.Write(b, 0, b.Length);
        }

        static void PutU16(byte[] b, int i, uint v) { b[i] = (byte)v; b[i + 1] = (byte)(v >> 8); }
        static void PutU32(byte[] b, int i, uint v) { b[i] = (byte)v; b[i+1] = (byte)(v>>8); b[i+2] = (byte)(v>>16); b[i+3] = (byte)(v>>24); }

        static uint[] _crcTable;
        static uint[] CrcTable() {
            if (_crcTable != null) return _crcTable;
            _crcTable = new uint[256];
            for (uint n = 0; n < 256; n++) {
                uint c = n;
                for (int k = 0; k < 8; k++) c = ((c & 1) != 0) ? (0xEDB88320u ^ (c >> 1)) : (c >> 1);
                _crcTable[n] = c;
            }
            return _crcTable;
        }
        public static uint Crc32(byte[] data) {
            var t = CrcTable();
            uint c = 0xFFFFFFFFu;
            for (int i = 0; i < data.Length; i++) c = t[(c ^ data[i]) & 0xFF] ^ (c >> 8);
            return c ^ 0xFFFFFFFFu;
        }
    }
}
