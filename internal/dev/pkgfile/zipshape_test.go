package pkgfile

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"io"
	"os"
	"testing"

	"tt/internal/testkit"
)

//---------------------------------------------------------------------------
// zip 写回保真：设计器形态（无数据描述符、extra 原样保留）
//
// 背景：真机 S7 验收发现，用 archive/zip 的 Writer 重写出来的包在设计器里打不开，
// 报「Data descriptor signature not found」——因为 Go 的 Writer 会无条件置
// general purpose flag bit 3 并补数据描述符，而设计器（.NET）写的包是
// 「CRC/大小直接写在局部头里、没有描述符」的形态。下面这几个用例把这个形态钉住。
//---------------------------------------------------------------------------

// rawEntryShape 是断言用的最小结构快照。
type rawEntryShape struct {
	name     string
	flags    uint16
	method   uint16
	crc      uint32
	csize    uint32
	usize    uint32
	offset   uint32
	localHdr []byte
	central  []byte
	data     []byte
}

func shapeOf(t *testing.T, b []byte) []rawEntryShape {
	t.Helper()
	rw, err := parseRawZip(b)
	if err != nil {
		t.Fatalf("parseRawZip: %v", err)
	}
	out := make([]rawEntryShape, 0, len(rw.entries))
	for i := range rw.entries {
		e := &rw.entries[i]
		out = append(out, rawEntryShape{
			name: e.name, flags: e.flags, method: e.method, crc: e.crc32,
			csize: e.csize32, usize: e.usize32, offset: e.offset32,
			localHdr: e.localHdr, central: e.central, data: e.compressed,
		})
	}
	return out
}

// readAllViaStdlib 用 archive/zip 独立读一遍：CRC 由标准库校验（读时校验），
// 顺带证明产出物是标准库也认的合法 zip。
func readAllViaStdlib(t *testing.T, b []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatalf("archive/zip 打不开产出物: %v", err)
	}
	out := map[string][]byte{}
	for _, zf := range zr.File {
		rc, err := zf.Open()
		if err != nil {
			t.Fatalf("打开条目 %s: %v", zf.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("读条目 %s（CRC 校验失败？）: %v", zf.Name, err)
		}
		out[zf.Name] = data
	}
	return out
}

// TestRawParseMatchesStdlib 交叉验证：自己的原始解析与 archive/zip 逐字段一致。
// （写回的保真度建立在解析正确之上，所以先钉住解析。）
func TestRawParseMatchesStdlib(t *testing.T) {
	pkgs := testkit.CorpusFiles(t, ".tzc")
	n := 0
	for _, p := range pkgs {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		rw, err := parseRawZip(b)
		if err != nil {
			t.Fatalf("%s parseRawZip: %v", p, err)
		}
		zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
		if err != nil {
			t.Fatalf("%s archive/zip: %v", p, err)
		}
		if len(rw.entries) != len(zr.File) {
			t.Fatalf("%s 条目数 %d != %d", p, len(rw.entries), len(zr.File))
		}
		for i, zf := range zr.File {
			re := &rw.entries[i]
			if re.name != zf.Name {
				t.Errorf("%s 第 %d 条名字 %q != %q", p, i, re.name, zf.Name)
				continue
			}
			if re.flags != zf.Flags {
				t.Errorf("%s %s flags 0x%04x != 0x%04x", p, zf.Name, re.flags, zf.Flags)
			}
			if re.method != zf.Method {
				t.Errorf("%s %s method %d != %d", p, zf.Name, re.method, zf.Method)
			}
			if re.crc32 != zf.CRC32 {
				t.Errorf("%s %s crc %08x != %08x", p, zf.Name, re.crc32, zf.CRC32)
			}
			// 比真实值（Zip64 时来自 extra；32 位字段本身是 0xFFFFFFFF 标记）
			if re.csizeEff != zf.CompressedSize64 {
				t.Errorf("%s %s csize %d != %d", p, zf.Name, re.csizeEff, zf.CompressedSize64)
			}
			if re.usizeEff != zf.UncompressedSize64 {
				t.Errorf("%s %s usize %d != %d", p, zf.Name, re.usizeEff, zf.UncompressedSize64)
			}
			// 32 位字段要么是真值、要么是 Zip64 标记（不允许第三种形态）
			if re.csize32 != zip64Marker && uint64(re.csize32) != re.csizeEff {
				t.Errorf("%s %s 32 位 csize 与真实值不一致（%d vs %d）", p, zf.Name, re.csize32, re.csizeEff)
			}
			if re.usize32 != zip64Marker && uint64(re.usize32) != re.usizeEff {
				t.Errorf("%s %s 32 位 usize 与真实值不一致（%d vs %d）", p, zf.Name, re.usize32, re.usizeEff)
			}
			// 原始压缩字节必须能解出与标准库一致的内容。
			if re.method == zip.Deflate {
				fr := flate.NewReader(bytes.NewReader(re.compressed))
				raw, err := io.ReadAll(fr)
				fr.Close()
				if err != nil {
					t.Errorf("%s %s 原始压缩数据解不开: %v", p, zf.Name, err)
					continue
				}
				if !bytes.Equal(raw, mustRead(t, zf)) {
					t.Errorf("%s %s 原始压缩数据解出的内容与标准库不一致", p, zf.Name)
				}
			}
			n++
		}
	}
	t.Logf("原始解析 vs 标准库：%d 个包、%d 个条目逐字段一致", len(pkgs), n)
}

func mustRead(t *testing.T, zf *zip.File) []byte {
	t.Helper()
	rc, err := zf.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestRebuildIdentityByteExact 零改动重建：
//   - 设计器形态的包 → 必须与原包**逐字节相同**（比"逐条目 sha256 一致"强得多）；
//   - 少数 tdev 早期产出的 bit3 包 → 归一化成设计器形态（不写数据描述符），
//     且归一化之后是稳定不动点。
func TestRebuildIdentityByteExact(t *testing.T) {
	pkgs := testkit.CorpusFiles(t, ".tzc")
	same, normalized := 0, 0
	for _, p := range pkgs {
		orig, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		pkg, err := Open(p, OpenOptions{})
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		out, _, err := pkg.Build(Rebuild{})
		if err != nil {
			t.Fatalf("%s Build: %v", p, err)
		}
		hadDesc := false
		for i := range pkg.raw.entries {
			if pkg.raw.entries[i].flags&zipFlagDataDescriptor != 0 {
				hadDesc = true
			}
		}
		if !hadDesc {
			if !bytes.Equal(orig, out) {
				t.Errorf("%s 零改动重建不是逐字节相同（原 %d B，产出 %d B）", p, len(orig), len(out))
			}
			same++
		} else {
			// bit3 输入：产出必须已经没有描述符位，且内容一致、可读。
			sh := shapeOf(t, out)
			for _, e := range sh {
				if e.flags&zipFlagDataDescriptor != 0 {
					t.Errorf("%s %s 归一化后仍带数据描述符位", p, e.name)
				}
			}
			got := readAllViaStdlib(t, out)
			for _, e := range pkg.Entries {
				if !bytes.Equal(got[e.Name], e.Data) {
					t.Errorf("%s %s 归一化后内容变了", p, e.Name)
				}
			}
			// 不动点：再重建一次必须逐字节相同。
			pkg2, err := Open(writeTemp(t, out), OpenOptions{})
			if err != nil {
				t.Fatalf("%s 归一化产物打不开: %v", p, err)
			}
			out2, _, err := pkg2.Build(Rebuild{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out, out2) {
				t.Errorf("%s 归一化产物不是不动点", p)
			}
			normalized++
		}
		// 无论哪条路径，产出都必须能被标准库读全（CRC 校验通过）。
		readAllViaStdlib(t, out)
	}
	t.Logf("零改动重建：%d 个包逐字节相同，%d 个包归一到设计器形态（原为 bit3 数据描述符）",
		same, normalized)
	if same == 0 {
		t.Errorf("语料里没有一个设计器形态的包？")
	}
}

// TestRebuildKeepsDesignerZipShape 改一个条目之后，容器形态必须还是设计器形态：
//   - 局部头里没有数据描述符位，CRC/大小就是真实值；
//   - 压缩数据后面紧跟下一个局部头或中央目录（没有描述符夹在中间）；
//   - 未改动条目的局部头 + 压缩数据逐字节照抄；中央目录记录只动偏移字段；
//   - extra 字段（UT-time / ux-infozip / Zip64 占位）一个字节都不变。
func TestRebuildKeepsDesignerZipShape(t *testing.T) {
	pkgs := testkit.CorpusFiles(t, ".tzc")
	// 挑一个设计器形态、且带 .tap 的包
	var target string
	for _, p := range pkgs {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		rw, err := parseRawZip(b)
		if err != nil {
			continue
		}
		hasTap, bit3 := false, false
		for i := range rw.entries {
			if rw.entries[i].name == "" {
				continue
			}
			if len(rw.entries[i].name) > 4 && rw.entries[i].name[len(rw.entries[i].name)-4:] == ".tap" {
				hasTap = true
			}
			if rw.entries[i].flags&zipFlagDataDescriptor != 0 {
				bit3 = true
			}
		}
		if hasTap && !bit3 {
			target = p
			break
		}
	}
	if target == "" {
		t.Skip("语料里没找到设计器形态的 .tap 包")
	}
	orig, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	pkg, err := Open(target, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tapName := pkg.Tap().Name
	newTap := append(append([]byte(nil), pkg.Tap().Data...), []byte("\r\n")...)
	out, actions, err := pkg.Build(Rebuild{Tap: newTap})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(actions) != len(pkg.Entries) {
		t.Fatalf("动作数 %d != 条目数 %d", len(actions), len(pkg.Entries))
	}

	before, err := parseRawZip(orig)
	if err != nil {
		t.Fatal(err)
	}
	after, err := parseRawZip(out)
	if err != nil {
		t.Fatalf("产出物 parseRawZip: %v", err)
	}
	if len(before.entries) != len(after.entries) {
		t.Fatalf("条目数变了 %d → %d", len(before.entries), len(after.entries))
	}

	// 逐个条目比对
	for i := range before.entries {
		b, a := &before.entries[i], &after.entries[i]
		if b.name != a.name {
			t.Fatalf("条目名/顺序变了：%q → %q", b.name, a.name)
		}
		if a.flags&zipFlagDataDescriptor != 0 {
			t.Errorf("%s 仍带数据描述符位（flags=0x%04x）", a.name, a.flags)
		}
		if b.name != tapName {
			// 未改动：局部头 + 压缩数据逐字节照抄；extra 原样
			if !bytes.Equal(b.localHdr, a.localHdr) {
				t.Errorf("%s 未改动却动了局部头", a.name)
			}
			if !bytes.Equal(b.compressed, a.compressed) {
				t.Errorf("%s 未改动的压缩数据变了", a.name)
			}
			if b.flags != a.flags || b.method != a.method || b.crc32 != a.crc32 ||
				b.csize32 != a.csize32 || b.usize32 != a.usize32 {
				t.Errorf("%s 未改动却动了字段", a.name)
			}
			// 中央目录记录：只允许偏移字段不同（前面的条目变长会挪动后面的偏移）
			cb, ca := append([]byte(nil), b.central...), append([]byte(nil), a.central...)
			o1, o2 := le32(cb, 42), le32(ca, 42)
			putLE32(cb, 42, 0)
			putLE32(ca, 42, 0)
			if !bytes.Equal(cb, ca) {
				t.Errorf("%s 未改动的中央目录记录被改了", a.name)
			}
			_ = o1
			_ = o2
		} else {
			// 目标条目：局部头除 flag/crc/大小外必须与原来一致
			h1, h2 := append([]byte(nil), b.localHdr...), append([]byte(nil), a.localHdr...)
			// 清掉会变动的字段：flags(6)、crc(14)、csize(18)、usize(22)
			for _, off := range []int{6, 14, 18, 22} {
				if off == 6 {
					putLE16(h1, off, 0)
					putLE16(h2, off, 0)
					continue
				}
				putLE32(h1, off, 0)
				putLE32(h2, off, 0)
			}
			if !bytes.Equal(h1, h2) {
				t.Errorf("%s 局部头除受控字段外被改动了", a.name)
			}
			if !bytes.Equal(b.localExtra(), a.localExtra()) {
				t.Errorf("%s 局部头 extra 被改动了", a.name)
			}
			if !bytes.Equal(b.centralExtra(), a.centralExtra()) {
				t.Errorf("%s 中央目录 extra 被改动了", a.name)
			}
			if a.usize32 != uint32(len(newTap)) {
				t.Errorf("%s usize=%d，期望 %d", a.name, a.usize32, len(newTap))
			}
		}
		// 数据后面紧跟下一个局部头或中央目录：证明没有数据描述符
		end := int(a.offset32) + len(a.localHdr) + len(a.compressed)
		if end+4 > len(out) {
			t.Fatalf("%s 数据越界", a.name)
		}
		next := string(out[end : end+4])
		if next != "PK\x03\x04" && next != "PK\x01\x02" {
			t.Errorf("%s 数据后面是 %q，说明夹了数据描述符", a.name, next)
		}
	}

	// 条目内容（用标准库独立读一遍，CRC 校验）
	got := readAllViaStdlib(t, out)
	if !bytes.Equal(got[tapName], newTap) {
		t.Errorf("改写的 .tap 内容没写进去")
	}
	for _, e := range pkg.Entries {
		if e.Name != tapName && !bytes.Equal(got[e.Name], e.Data) {
			t.Errorf("%s 内容变了", e.Name)
		}
	}
	// EOCD 的条目数与注释保持
	if le16(before.eocd, 10) != le16(after.eocd, 10) || !bytes.Equal(before.eocd[22:], after.eocd[22:]) {
		t.Errorf("EOCD 条目数或注释被改动")
	}
	// 中央目录尺寸/偏移必须自洽（否则任何严格读者都会挂）
	if int(le32(after.eocd, 16))+int(le32(after.eocd, 12)) != len(out)-len(after.eocd) {
		t.Errorf("EOCD 的中央目录偏移/尺寸不自洽")
	}
	t.Logf("形态保持：%s（改 %s，%d B → %d B）", target, tapName, len(orig), len(out))
}

// TestRebuildZip64EntryRewriteIsReadable 回归：**带 Zip64 字段**的包里改条目。
//
// 语料里有一批包（.NET 写的）把局部头与中央目录的 32 位大小字段都写成 0xFFFFFFFF，
// 真值放在 Zip64 extra 里（AppNote 4.5.3：原大小 → 压缩后大小 → 偏移）。
// 这类包改一个条目时，补丁必须打到**待写出的副本**上：曾经因为改的是原始解析结果，
// 出现「压缩数据变了、extra 里的尺寸没变」，产出物连 Go 自己都读不回来
// （zip: not a valid zip file），真机自然是打不开的。
func TestRebuildZip64EntryRewriteIsReadable(t *testing.T) {
	var target string
	for _, p := range testkit.CorpusFiles(t, ".tzc") {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		rw, err := parseRawZip(b)
		if err != nil {
			continue
		}
		for i := range rw.entries {
			e := &rw.entries[i]
			if e.usize32 == zip64Marker && isTapName(e.name) {
				target = p
			}
		}
		if target != "" {
			break
		}
	}
	if target == "" {
		t.Skip("语料里没有 Zip64 形态的 .tap 包")
	}
	pkg, err := Open(target, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	tapName := pkg.Tap().Name
	newTap := append(append([]byte(nil), pkg.Tap().Data...), []byte("\r\n")...)
	out, _, err := pkg.Build(Rebuild{Tap: newTap})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := readAllViaStdlib(t, out) // 任何一条读不回来都会在这里 fatal
	if !bytes.Equal(got[tapName], newTap) {
		t.Errorf("改写后的 .tap 内容不对（%d 字节，期望 %d）", len(got[tapName]), len(newTap))
	}
	for _, e := range pkg.Entries {
		if e.Name != tapName && !bytes.Equal(got[e.Name], e.Data) {
			t.Errorf("%s 内容变了", e.Name)
		}
	}
	// extra 里的真实值必须与产出物自洽
	rw, err := parseRawZip(out)
	if err != nil {
		t.Fatal(err)
	}
	for i := range rw.entries {
		e := &rw.entries[i]
		if e.csizeEff != uint64(len(e.compressed)) {
			t.Errorf("%s extra 里的压缩后大小 %d != 实际 %d", e.name, e.csizeEff, len(e.compressed))
		}
		if e.flags&zipFlagDataDescriptor != 0 {
			t.Errorf("%s 带数据描述符位", e.name)
		}
	}
	t.Logf("Zip64 包改写可读：%s（%s，新长度 %d）", target, tapName, len(newTap))
}

// isTapName 判断条目名是不是 .tap。
func isTapName(name string) bool {
	return len(name) >= 4 && name[len(name)-4:] == ".tap"
}

// TestRebuildNormalizesGoShapedZip 夹具是 Go 的 zip.Writer 写的（bit3 + 描述符）：
// 重建后必须变成设计器形态，且标准库仍能读全、形态稳定。
func TestRebuildNormalizesGoShapedZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"a.tap": "// tap 內容\r\n",
		"b.tgl": "// tgl 內容\r\n",
		"ver":   "1.0\r\n",
	}
	for _, n := range []string{"a.tap", "b.tgl", "ver"} {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: n, Method: zip.Deflate})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[n])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := writeTemp(t, buf.Bytes())
	pkg, err := Open(path, OpenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	bit3 := false
	for i := range pkg.raw.entries {
		if pkg.raw.entries[i].flags&zipFlagDataDescriptor != 0 {
			bit3 = true
		}
	}
	if !bit3 {
		t.Fatalf("夹具本来应该是 Go 写的 bit3 形态")
	}
	out, _, err := pkg.Build(Rebuild{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range shapeOf(t, out) {
		if e.flags&zipFlagDataDescriptor != 0 {
			t.Errorf("%s 归一化后仍带描述符位", e.name)
		}
	}
	if bytes.Contains(out, []byte("PK\x07\x08")) {
		t.Errorf("产出物里还有数据描述符签名")
	}
	got := readAllViaStdlib(t, out)
	for _, e := range pkg.Entries {
		if !bytes.Equal(got[e.Name], e.Data) {
			t.Errorf("%s 内容变了", e.Name)
		}
	}
}
