package pkgfile

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"fmt"
	"hash/crc32"
)

// ---------------------------------------------------------------------------
// zip 原始结构（写回保真的地基）
//
// 为什么不能用 archive/zip 的 Writer 直接写：
//
//	设计器（.NET）产出的包，局部头里**没有数据描述符**（general purpose flag bit 3 = 0），
//	CRC / 压缩后大小 / 原大小直接写在局部头里，并且每个条目都带自己的 extra 字段
//	（真机实测：UT-time + ux-infozip）。而 archive/zip 的 Writer 会**无条件**置 bit 3
//	并补一个数据描述符，还会用自己的 9 字节 UT extra 顶掉原 extra。
//	后果：这样产出的包在设计器里打开直接报
//	「Data descriptor signature not found」——S7 真机验收抓到的第一个不兼容。
//
// 所以写回改成**字节级重建**：
//
//	· 局部头与中央目录记录都从原包逐字节拷贝，只打补丁：清 bit 3、写真实 CRC/大小、
//	  更新局部头偏移、必要时更新 Zip64 extra 里的 8 字节字段；
//	· 未改动的条目连压缩数据都逐字节照抄（零改动重建 = 与原包逐字节相同）；
//	· 一切 extra 字段原样保留（不新增、不删除、不重排），版本号/时间/属性/注释不动。
//
// 这套规则是「照抄设计器的写法」，而不是「写一个合法 zip」：合法不够，得像它。
// ---------------------------------------------------------------------------

const (
	zipFlagDataDescriptor = 0x0008
	zip64Marker           = 0xFFFFFFFF
	zip64ExtraTag         = 0x0001
)

// rawZipEntry 是一个条目在**原包字节里**的原始视图。
type rawZipEntry struct {
	name       string
	localHdr   []byte // 30 + nlen + elen（局部头，不含数据）
	compressed []byte // 原压缩数据（逐字节）
	central    []byte // 46 + nlen + elen + clen（中央目录记录）

	nlen, elen int
	flags      uint16
	method     uint16
	crc32      uint32

	// 原 32 位字段：写回时「原来是标记就还是标记」，不做静默去 Zip64。
	csize32, usize32, offset32 uint32
	// 局部头自己的 32 位大小字段（局部 extra 的 Zip64 判定用它，不看中央目录）。
	lCsize32, lUsize32 uint32
	// 真实值：Zip64 时来自 extra（AppNote 4.5.3）。
	csizeEff, usizeEff, offEff uint64

	fileIdx int // 对应 Package.Entries 下标；目录条目为 -1
}

// rawZip 是整包的原始结构。
type rawZip struct {
	prefix  []byte // 第一个局部头之前的字节（正常为空）
	entries []rawZipEntry
	eocd    []byte // 原 EOCD（22 + 注释），逐字节保留
}

func le16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off:]) }
func le32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off:]) }

func putLE16(b []byte, off int, v uint16) { binary.LittleEndian.PutUint16(b[off:], v) }
func putLE32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }

// findEOCD 从尾部找 EOCD（注释最长 64 KiB，和 archive/zip 一样）。
func findEOCD(b []byte) int {
	const eocdLen = 22
	if len(b) < eocdLen {
		return -1
	}
	low := len(b) - eocdLen - 0xFFFF
	if low < 0 {
		low = 0
	}
	for i := len(b) - eocdLen; i >= low; i-- {
		if b[i] != 'P' || b[i+1] != 'K' || b[i+2] != 5 || b[i+3] != 6 {
			continue
		}
		if int(le16(b, i+20))+eocdLen == len(b)-i {
			return i
		}
	}
	return -1
}

// parseRawZip 把原包字节解析成逐条目原始记录。
//
// 定位条目数据用**中央目录**（权威），局部头只用来取原始字节与 extra；
// 同时校验局部头/中央目录的名字一致，不一致直接判为包格式错（写得出来也读不回来）。
func parseRawZip(b []byte) (*rawZip, error) {
	i := findEOCD(b)
	if i < 0 {
		return nil, &FormatError{Msg: "zip 里找不到 EOCD（中央目录结尾记录）"}
	}
	if le16(b, i+8) == 0xFFFF || le16(b, i+10) == 0xFFFF ||
		le32(b, i+12) == zip64Marker || le32(b, i+16) == zip64Marker {
		return nil, &FormatError{
			Msg:    "暂不支持写回 Zip64 包（条目数或中央目录偏移超过 32 位）",
			Detail: []string{"本工具只重写设计器 32 位形态的包；如需请在设计器里另存一次"},
		}
	}
	n := int(le16(b, i+10))
	cdSize := int(le32(b, i+12))
	cdStart := int(le32(b, i+16))
	if cdStart < 0 || cdSize < 0 || cdStart+cdSize > len(b) {
		return nil, &FormatError{Msg: "中央目录越界（包被截断？）",
			Detail: []string{fmt.Sprintf("cdStart=%d cdSize=%d len=%d", cdStart, cdSize, len(b))}}
	}
	rw := &rawZip{eocd: append([]byte(nil), b[i:]...)}

	p := cdStart
	cdEnd := cdStart + cdSize
	for k := 0; k < n; k++ {
		if p+46 > cdEnd || string(b[p:p+4]) != "PK\x01\x02" {
			return nil, &FormatError{Msg: fmt.Sprintf("中央目录第 %d 条记录损坏", k+1)}
		}
		nlen := int(le16(b, p+28))
		elen := int(le16(b, p+30))
		clen := int(le16(b, p+32))
		if p+46+nlen+elen+clen > cdEnd {
			return nil, &FormatError{Msg: fmt.Sprintf("中央目录第 %d 条记录越界", k+1)}
		}
		nameBytes := b[p+46 : p+46+nlen]
		csize32 := le32(b, p+20)
		usize32 := le32(b, p+24)
		offset32 := le32(b, p+42)
		csizeEff, usizeEff, offEff := uint64(csize32), uint64(usize32), uint64(offset32)
		cdExtra := b[p+46+nlen : p+46+nlen+elen]
		if csize32 == zip64Marker || usize32 == zip64Marker || offset32 == zip64Marker {
			u, c, o, err := zip64Read(cdExtra, usize32 == zip64Marker, csize32 == zip64Marker, offset32 == zip64Marker)
			if err != nil {
				return nil, &FormatError{Msg: fmt.Sprintf("条目 %q %v", string(nameBytes), err)}
			}
			if usize32 == zip64Marker {
				usizeEff = u
			}
			if csize32 == zip64Marker {
				csizeEff = c
			}
			if offset32 == zip64Marker {
				offEff = o
			}
		}
		off := int(offEff)
		if off+30 > len(b) || string(b[off:off+4]) != "PK\x03\x04" {
			return nil, &FormatError{Msg: fmt.Sprintf("条目 %q 的局部头偏移无效", string(nameBytes))}
		}
		lnlen := int(le16(b, off+26))
		lelen := int(le16(b, off+28))
		hdrEnd := off + 30 + lnlen + lelen
		if csizeEff > uint64(len(b)) || hdrEnd+int(csizeEff) > len(b) {
			return nil, &FormatError{Msg: fmt.Sprintf("条目 %q 的压缩数据越界", string(nameBytes))}
		}
		if !bytes.Equal(nameBytes, b[off+30:off+30+lnlen]) {
			return nil, &FormatError{
				Msg:    fmt.Sprintf("条目 %q 的局部头名字与中央目录不一致", string(nameBytes)),
				Detail: []string{"这种包写回去也读不回来，拒绝改写"},
			}
		}
		if k == 0 && off > 0 {
			rw.prefix = append([]byte(nil), b[:off]...)
		}
		rw.entries = append(rw.entries, rawZipEntry{
			name:       string(nameBytes),
			localHdr:   append([]byte(nil), b[off:hdrEnd]...),
			compressed: append([]byte(nil), b[hdrEnd:hdrEnd+int(csizeEff)]...),
			central:    append([]byte(nil), b[p:p+46+nlen+elen+clen]...),
			nlen:       nlen,
			elen:       elen,
			flags:      le16(b, p+8),
			method:     le16(b, p+10),
			crc32:      le32(b, p+16),
			csize32:    csize32,
			usize32:    usize32,
			offset32:   offset32,
			lCsize32:   le32(b, off+18),
			lUsize32:   le32(b, off+22),
			csizeEff:   csizeEff,
			usizeEff:   usizeEff,
			offEff:     offEff,
			fileIdx:    -1,
		})
		p += 46 + nlen + elen + clen
	}
	if p != cdEnd {
		return nil, &FormatError{Msg: "中央目录长度与记录条数不符"}
	}
	return rw, nil
}

// localExtra 返回局部头里的 extra 字段（副本偏移由调用方给）。
func (e *rawZipEntry) localExtra() []byte { return e.localHdr[30+e.nlen : 30+e.nlen+e.elen] }

// centralExtra 返回中央目录记录里的 extra 字段。
func (e *rawZipEntry) centralExtra() []byte {
	return e.central[46+e.nlen : 46+e.nlen+e.elen]
}

// zip64Read 按 AppNote 4.5.3 从 Zip64 extra 里取真实值。
//
// 顺序固定：原大小 → 压缩后大小 → 局部头偏移 → 磁盘号；只有对应的 32 位字段
// 被写成 0xFFFFFFFF 的字段才出现在 extra 里，所以「要读哪几个」由调用方
// 依 32 位字段的标记传入（wantU/wantC/wantO）。
func zip64Read(extra []byte, wantU, wantC, wantO bool) (usize, csize, off uint64, err error) {
	need := 0
	for _, w := range []bool{wantU, wantC, wantO} {
		if w {
			need += 8
		}
	}
	pos := 0
	for pos+4 <= len(extra) {
		tag := le16(extra, pos)
		sz := int(le16(extra, pos+2))
		if pos+4+sz > len(extra) {
			break
		}
		if tag == zip64ExtraTag {
			d := extra[pos+4 : pos+4+sz]
			if len(d) < need {
				return 0, 0, 0, &FormatError{
					Msg:    "Zip64 extra 长度不足，无法定位条目数据",
					Detail: []string{fmt.Sprintf("需要 %d 字节，只有 %d 字节", need, len(d))},
				}
			}
			q := 0
			if wantU {
				usize = binary.LittleEndian.Uint64(d[q:])
				q += 8
			}
			if wantC {
				csize = binary.LittleEndian.Uint64(d[q:])
				q += 8
			}
			if wantO {
				off = binary.LittleEndian.Uint64(d[q:])
			}
			return usize, csize, off, nil
		}
		pos += 4 + sz
	}
	if need == 0 {
		return 0, 0, 0, nil
	}
	return 0, 0, 0, &FormatError{Msg: "包自称 Zip64（32 位字段为 0xFFFFFFFF），但 extra 里没有 Zip64 字段"}
}

// patchZip64 把真实值写进 Zip64 extra 的 8 字节字段（字段顺序同 zip64Read）。
//
// 只有条目真的被改写、或它前面的条目变长导致偏移变化时才调用：
// 未改动的条目必须逐字节照抄，不做任何"顺手修正"。
func patchZip64(extra []byte, wantU, wantC, wantO bool, usize, csize, off uint64) error {
	if !wantU && !wantC && !wantO {
		return nil
	}
	_, _, _, err := zip64Read(extra, wantU, wantC, wantO) // 先确认长度够（顺带校验形态）
	if err != nil {
		return err
	}
	pos := 0
	for pos+4 <= len(extra) {
		tag := le16(extra, pos)
		sz := int(le16(extra, pos+2))
		if pos+4+sz > len(extra) {
			break
		}
		if tag == zip64ExtraTag {
			d := extra[pos+4 : pos+4+sz]
			q := 0
			if wantU {
				binary.LittleEndian.PutUint64(d[q:], usize)
				q += 8
			}
			if wantC {
				binary.LittleEndian.PutUint64(d[q:], csize)
				q += 8
			}
			if wantO {
				binary.LittleEndian.PutUint64(d[q:], off)
			}
			return nil
		}
		pos += 4 + sz
	}
	return &FormatError{Msg: "extra 里找不到 Zip64 字段"}
}

// compressEntry 用条目原有的压缩方法压回新内容。
// 原实现的行为保持不变：无法复现的压缩方法回退 deflate，并给出 reason。
func compressEntry(method uint16, data []byte) ([]byte, uint16, string, error) {
	if method == zip.Store {
		return data, zip.Store, "", nil
	}
	reason := ""
	if method != zip.Deflate {
		reason = fmt.Sprintf("原压缩方法 %d 无法复现，回退 deflate", method)
	}
	var b bytes.Buffer
	w, err := flate.NewWriter(&b, flate.DefaultCompression)
	if err != nil {
		return nil, 0, "", &IOError{Msg: "初始化 deflate 失败", Err: err}
	}
	if _, err := w.Write(data); err != nil {
		return nil, 0, "", &IOError{Msg: "压缩新内容失败", Err: err}
	}
	if err := w.Close(); err != nil {
		return nil, 0, "", &IOError{Msg: "收尾 deflate 流失败", Err: err}
	}
	return b.Bytes(), zip.Deflate, reason, nil
}

// rebuildZip 是写回的唯一实现：按原包顺序重建，逐字节保真 + 最小打补丁。
//
// repl 的键是条目名；actions 与 p.Entries 一一对应，用于回填 reason。
func (p *Package) rebuildZip(repl map[string][]byte, actions []EntryAction) ([]byte, error) {
	if p.raw == nil {
		return nil, &FormatError{Msg: "缺少原始 zip 结构信息，拒绝写回（内部错误）"}
	}
	var out bytes.Buffer
	out.Write(p.raw.prefix)
	cds := make([][]byte, 0, len(p.raw.entries))

	for ei := range p.raw.entries {
		re := &p.raw.entries[ei]
		off := out.Len()
		hdr := append([]byte(nil), re.localHdr...)
		cd := append([]byte(nil), re.central...)
		data := re.compressed
		crc, csize, usize := re.crc32, re.csizeEff, re.usizeEff
		method := re.method
		flags := re.flags &^ zipFlagDataDescriptor // 设计器形态：不写数据描述符
		changed := false
		if re.fileIdx >= 0 {
			if nd, ok := repl[re.name]; ok {
				changed = true
				enc, m2, reason, err := compressEntry(re.method, nd)
				if err != nil {
					return nil, err
				}
				if reason != "" {
					actions[re.fileIdx].Reason = reason
				}
				if uint64(len(nd)) > zip64Marker || uint64(len(enc)) > zip64Marker {
					return nil, &FormatError{Msg: "改写后的条目超过 4 GiB，暂不支持", Detail: []string{re.name}}
				}
				data = enc
				method = m2
				crc = crc32.ChecksumIEEE(nd)
				csize = uint64(len(enc))
				usize = uint64(len(nd))
			}
		}
		newOff := uint64(off)
		offChanged := newOff != re.offEff
		// 32 位装得下就写真实值；原来用 0xFFFFFFFF 标记的，保持标记（真值进 extra）。
		wantC, wantU, wantO := re.csize32 == zip64Marker, re.usize32 == zip64Marker, re.offset32 == zip64Marker
		lWantC, lWantU := re.lCsize32 == zip64Marker, re.lUsize32 == zip64Marker
		if !wantC && !wantU && !wantO && !lWantC && !lWantU &&
			(csize > zip64Marker || usize > zip64Marker || newOff > zip64Marker) {
			return nil, &FormatError{Msg: "包超过 4 GiB，暂不支持写回（原包没有 Zip64 字段）"}
		}

		// 局部头：清 bit 3、写真实 CRC/大小
		putLE16(hdr, 6, flags)
		if method != re.method {
			putLE16(hdr, 8, method)
		}
		putLE32(hdr, 14, crc)
		if lWantC {
			putLE32(hdr, 18, zip64Marker)
		} else {
			putLE32(hdr, 18, uint32(csize))
		}
		if lWantU {
			putLE32(hdr, 22, zip64Marker)
		} else {
			putLE32(hdr, 22, uint32(usize))
		}
		if changed && (lWantU || lWantC) {
			// 注意：必须补丁到**待写出的副本** hdr 上，不是 re.localHdr ——
			// 改原始解析结果会让"数据变了、尺寸字段没变"，产出物直接读不回来。
			if err := patchZip64(hdr[30+re.nlen:30+re.nlen+re.elen], lWantU, lWantC, false,
				usize, csize, 0); err != nil {
				return nil, err
			}
		}
		out.Write(hdr)
		out.Write(data)

		// 中央目录记录：同样的补丁 + 新的局部头偏移
		putLE16(cd, 8, flags)
		if method != re.method {
			putLE16(cd, 10, method)
		}
		putLE32(cd, 16, crc)
		if wantC {
			putLE32(cd, 20, zip64Marker)
		} else {
			putLE32(cd, 20, uint32(csize))
		}
		if wantU {
			putLE32(cd, 24, zip64Marker)
		} else {
			putLE32(cd, 24, uint32(usize))
		}
		if wantO {
			putLE32(cd, 42, zip64Marker)
		} else {
			putLE32(cd, 42, uint32(newOff))
		}
		if (changed || offChanged) && (wantU || wantC || wantO) {
			// 同上：补丁打到副本 cd 上。
			if err := patchZip64(cd[46+re.nlen:46+re.nlen+re.elen], wantU, wantC, wantO,
				usize, csize, newOff); err != nil {
				return nil, err
			}
		}
		cds = append(cds, cd)
	}

	cdStart := out.Len()
	for _, cd := range cds {
		out.Write(cd)
	}
	cdSize := out.Len() - cdStart
	if uint64(cdStart) > zip64Marker || uint64(cdSize) > zip64Marker {
		return nil, &FormatError{Msg: "中央目录超过 4 GiB，暂不支持写回"}
	}
	eocd := append([]byte(nil), p.raw.eocd...)
	putLE32(eocd, 12, uint32(cdSize))
	putLE32(eocd, 16, uint32(cdStart))
	out.Write(eocd)
	return out.Bytes(), nil
}
