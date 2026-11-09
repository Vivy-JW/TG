// Package pkgfile 实现设计指南 §5.1「Package 层 —— 包的只读视图」。
//
// 依据：
//   - 设计指南 §5.1 / §5.6 / §7 R2 R6 R7
//   - 反编译源码（容器层、条目→加载器分派表、打包）：SpecDesignerCommon/TzpManager.cs:150-182（类型由扩展名决定）
//     SpecDesignerCommon/PackageManager.cs:70-100（必需条目）
//     SpecDesignerCommon/PackageManager.cs:589-604（ver 按名字取、取第一行）
//     SpecDesignerCommon/TzpManager.cs:395-404（只比较 Major/Minor）
//
// 契约：Package 不可变，只读。写回一律走 Build()，不存在 SetPoint/SetTgl 之类接口
// （设计指南 §5.6 末段、红线 R5）。
package pkgfile

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

// ExitCodeError 让 CLI 层把错误映射为设计指南 §2 的退出码。
type ExitCodeError interface {
	error
	ExitCode() int
}

// FormatError = 退出码 2：zip 损坏、缺必须条目、ver 不存在/不匹配。
type FormatError struct {
	Msg    string
	Detail []string
}

func (e *FormatError) Error() string {
	if len(e.Detail) == 0 {
		return e.Msg
	}
	return e.Msg + "：" + strings.Join(e.Detail, "; ")
}

func (e *FormatError) ExitCode() int { return 2 }

// IOError = 退出码 5：磁盘 / 环境失败。
type IOError struct {
	Msg string
	Err error
}

func (e *IOError) Error() string {
	if e.Err == nil {
		return e.Msg
	}
	return e.Msg + "：" + e.Err.Error()
}

func (e *IOError) Unwrap() error { return e.Err }

func (e *IOError) ExitCode() int { return 5 }

// Kind 是包类型，由**文件扩展名**决定（不看内容）。
type Kind int

const (
	KindNone Kind = iota
	KindCode
	KindForm
	KindCodeSpec
	KindReportSpec
	KindReportCode
	KindReport
)

func (k Kind) String() string {
	switch k {
	case KindCode:
		return "Code"
	case KindForm:
		return "Form"
	case KindCodeSpec:
		return "CodeSpec"
	case KindReportSpec:
		return "ReportSpec"
	case KindReportCode:
		return "ReportCode"
	case KindReport:
		return "Report"
	}
	return "None"
}

// UploadCode 是上传代号，仅用于人类可读输出。
func (k Kind) UploadCode() string {
	switch k {
	case KindCode:
		return "CODE"
	case KindForm:
		return "SPEC"
	case KindCodeSpec:
		return "CSPEC"
	case KindReportSpec:
		return "RSPEC"
	case KindReportCode:
		return "GCODE"
	case KindReport:
		return "4RP"
	}
	return ""
}

// Version 是 ver 条目的主次版本。
type Version struct {
	Major, Minor int
	Raw          string
}

func (v Version) String() string { return v.Raw }

// Entry 是一个 zip 条目的字节视图。
type Entry struct {
	Name          string
	Method        uint16
	Modified      time.Time
	ExternalAttrs uint32
	Comment       string
	NonUTF8       bool
	Data          []byte
	Sha256        string
	CRC32         uint32
	// 原始压缩后大小，仅用于报告。
	CompressedSize uint64

	// raw 是条目在原包字节里的原始记录（写回时逐字节保真用，见 zipraw.go）。
	raw *rawZipEntry
}

func (e *Entry) Ext() string { return path.Ext(e.Name) }

func (e *Entry) Size() int { return len(e.Data) }

// Package 是包的只读视图。
type Package struct {
	Path         string
	Kind         Kind
	IsDiff       bool // .tzx
	IsIndFun     bool // .tzf
	IsSimpleForm bool // .tzv
	Entries      []*Entry
	Ver          Version

	index map[string]int
	raw   *rawZip // 整包原始 zip 结构（写回用）
}

// OpenOptions 控制打开时的校验强度。
type OpenOptions struct {
	// AcceptVer 是客户端可接受的 "Major.Minor"，默认 "1.0"
	// （设计器 1.0.0.251 → SettingManager.Version，只比主次版本）。
	AcceptVer string
}

// sha256hex 计算小写十六进制摘要。
func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// Sha256Hex 导出给其它层复用。
func Sha256Hex(b []byte) string { return sha256hex(b) }

// ParseVersion 解析 ver 内容的第一行（PackageManager.SeekReleaseVersion 用 ReadLine）。
func ParseVersion(content []byte) (Version, error) {
	line := string(content)
	if i := strings.IndexAny(line, "\r\n"); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return Version{}, fmt.Errorf("ver 条目内容为空")
	}
	parts := strings.Split(line, ".")
	if len(parts) < 2 {
		return Version{}, fmt.Errorf("ver 内容 %q 不是 Major.Minor[.Build[.Rev]] 形式", line)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("ver 主版本 %q 不是数字", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("ver 次版本 %q 不是数字", parts[1])
	}
	return Version{Major: major, Minor: minor, Raw: line}, nil
}

// kindFromExt 复刻 TzpManager.Type 的扩展名分支（TzpManager.cs:150-182）。
func kindFromExt(ext string) (Kind, bool, bool, bool) {
	switch strings.ToLower(ext) {
	case ".tzs":
		return KindForm, false, false, false
	case ".tzv":
		return KindForm, false, false, true
	case ".tzc":
		return KindCode, false, false, false
	case ".tzf":
		return KindCode, false, true, false
	case ".tzx":
		return KindCode, true, false, false
	case ".tzd":
		return KindCodeSpec, false, false, false
	case ".tzr":
		return KindReportSpec, false, false, false
	case ".tzg":
		return KindReportCode, false, false, false
	case ".tzt":
		return KindReport, false, false, false
	}
	return KindNone, false, false, false
}

// mandatoryExts 复刻 PackageManager.Unpacking 的必需条目清单（PackageManager.cs:70-100）。
func mandatoryExts(k Kind, isDiff, isSimpleForm bool) []string {
	switch k {
	case KindReport:
		return []string{".tap", ".tgl"}
	case KindCode:
		if isDiff {
			return []string{".tap", ".tgl", ".src", ".apt"}
		}
		return []string{".tap", ".tgl"}
	case KindCodeSpec:
		return []string{".csd"}
	case KindReportSpec:
		return []string{".rsd"}
	case KindForm:
		if isSimpleForm {
			return []string{".tsd", ".4fd", ".4fdref"}
		}
		return []string{".tsd", ".4fd"}
	}
	// .tzg（ReportCode）在设计器里没有 case 分支，因此不检查必需条目。
	return nil
}

// Open 打开包并做格式校验（设计指南 §2 退出码 2 的来源）。
func Open(p string, opts OpenOptions) (*Package, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, &IOError{Msg: "无法访问包文件 " + p, Err: err}
	}
	if st.IsDir() {
		return nil, &FormatError{Msg: p + " 是目录，不是 .tz 包"}
	}
	accept := opts.AcceptVer
	if accept == "" {
		accept = "1.0"
	}

	f, err := os.Open(p)
	if err != nil {
		return nil, &IOError{Msg: "无法打开包文件 " + p, Err: err}
	}
	defer f.Close()

	// 写回要逐字节保真（原 extra 字段、局部头形态），所以整包读进内存，
	// 既给 archive/zip 读内容，也自己解析一份原始结构（zipraw.go）。
	rawBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, &IOError{Msg: "读取包文件失败 " + p, Err: err}
	}
	zr, err := zip.NewReader(bytes.NewReader(rawBytes), int64(len(rawBytes)))
	if err != nil {
		return nil, &FormatError{Msg: "zip 结构损坏：" + p, Detail: []string{err.Error()}}
	}
	rw, err := parseRawZip(rawBytes)
	if err != nil {
		return nil, err
	}
	if len(rw.entries) != len(zr.File) {
		return nil, &FormatError{
			Msg: "zip 中央目录条目数不一致（文件可能在读取期间被改动）",
			Detail: []string{fmt.Sprintf("archive/zip=%d，原始解析=%d", len(zr.File), len(rw.entries)),
				p},
		}
	}

	kind, isDiff, isIndFun, isSimple := kindFromExt(path.Ext(p))
	if kind == KindNone {
		return nil, &FormatError{
			Msg:    "不是设计器认识的包类型（按扩展名判定）",
			Detail: []string{"文件：" + p, "支持的扩展名：.tzc .tzf .tzx .tzs .tzv .tzd .tzr .tzg .tzt"},
		}
	}

	pkg := &Package{
		Path:         p,
		Kind:         kind,
		IsDiff:       isDiff,
		IsIndFun:     isIndFun,
		IsSimpleForm: isSimple,
		index:        map[string]int{},
		raw:          rw,
	}

	for i, zf := range zr.File {
		re := &pkg.raw.entries[i]
		if zf.FileInfo().IsDir() {
			// 设计器只处理 IsFile 条目（PackageManager.cs:220），目录条目直接跳过；
			// 但写回时它会原样保留（R2：条目集合不增不减）。
			re.fileIdx = -1
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			return nil, &FormatError{Msg: "无法读取条目 " + zf.Name, Detail: []string{err.Error()}}
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, &FormatError{Msg: "解压条目失败 " + zf.Name, Detail: []string{err.Error()}}
		}
		e := &Entry{
			Name:           zf.Name,
			Method:         zf.Method,
			Modified:       zf.Modified,
			ExternalAttrs:  zf.ExternalAttrs,
			Comment:        zf.Comment,
			NonUTF8:        zf.NonUTF8,
			Data:           data,
			Sha256:         sha256hex(data),
			CRC32:          crc32.ChecksumIEEE(data),
			CompressedSize: zf.CompressedSize64,
			raw:            re,
		}
		if prev, dup := pkg.index[e.Name]; dup {
			return nil, &FormatError{
				Msg:    "zip 里有重名条目",
				Detail: []string{e.Name, fmt.Sprintf("第 %d 与第 %d 个条目同名", prev+1, len(pkg.Entries)+1)},
			}
		}
		re.fileIdx = len(pkg.Entries)
		pkg.index[e.Name] = len(pkg.Entries)
		pkg.Entries = append(pkg.Entries, e)
	}

	if len(pkg.Entries) == 0 {
		return nil, &FormatError{Msg: "zip 里没有任何文件条目", Detail: []string{p}}
	}

	// 必需条目：只按扩展名判存在，不看名字（PackageManager.cs:115-119 / 194-203）。
	var missing []string
	for _, ext := range mandatoryExts(kind, isDiff, isSimple) {
		if pkg.firstByExt(ext) == nil {
			missing = append(missing, ext)
		}
	}
	if len(missing) > 0 {
		return nil, &FormatError{
			Msg:    "包缺少必须条目（设计器会拒绝打开）",
			Detail: append([]string{"缺少：" + strings.Join(missing, " ")}, pkg.entryNames()...),
		}
	}

	// ver：必须存在，按精确名字 "ver" 取（PackageManager.cs:595），只比较主次版本。
	vEntry := pkg.Entry("ver")
	if vEntry == nil {
		return nil, &FormatError{
			Msg:    "包缺少 ver 条目（版本闸门，必须存在且名字精确为 ver、无目录前缀）",
			Detail: pkg.entryNames(),
		}
	}
	ver, err := ParseVersion(vEntry.Data)
	if err != nil {
		return nil, &FormatError{Msg: "ver 条目无法解析", Detail: []string{err.Error()}}
	}
	pkg.Ver = ver
	acc := strings.SplitN(strings.TrimSpace(accept), ".", 3)
	if len(acc) < 2 {
		return nil, &FormatError{Msg: "AcceptVer 参数不合法", Detail: []string{accept}}
	}
	am, _ := strconv.Atoi(acc[0])
	an, _ := strconv.Atoi(acc[1])
	if ver.Major != am || ver.Minor != an {
		return nil, &FormatError{
			Msg: "ver 与客户端版本不兼容",
			Detail: []string{
				fmt.Sprintf("包内 ver = %s（主 %d 次 %d）", ver.Raw, ver.Major, ver.Minor),
				fmt.Sprintf("客户端接受 = %s", accept),
			},
		}
	}

	return pkg, nil
}

// entryNames 返回条目名清单，用于错误详情。
func (p *Package) entryNames() []string {
	out := make([]string, 0, len(p.Entries))
	for _, e := range p.Entries {
		out = append(out, fmt.Sprintf("%s (%d B)", e.Name, len(e.Data)))
	}
	return out
}

// Entry 按精确名字取条目。
func (p *Package) Entry(name string) *Entry {
	if i, ok := p.index[name]; ok {
		return p.Entries[i]
	}
	return nil
}

// Has 判断条目是否存在。
func (p *Package) Has(name string) bool { return p.Entry(name) != nil }

func (p *Package) firstByExt(ext string) *Entry {
	for _, e := range p.Entries {
		if e.Ext() == ext {
			return e
		}
	}
	return nil
}

// ByExt 按扩展名取第一个条目（大小写敏感，对齐设计器行为）。
func (p *Package) ByExt(ext string) *Entry { return p.firstByExt(ext) }

// Tap 返回 .tap 条目（设计文件，唯一被服务端消费的条目）。
func (p *Package) Tap() *Entry { return p.firstByExt(".tap") }

// Tgl 返回 .tgl 条目（框架骨架）。
func (p *Package) Tgl() *Entry { return p.firstByExt(".tgl") }

// Full4gl 返回 .4gl 条目（服务器产出，默认只读，永不改写 —— 红线 R1）。
func (p *Package) Full4gl() *Entry { return p.firstByExt(".4gl") }

// Bdx 返回 .bdx 条目（表单控件绑定；本来就有才更新）。
func (p *Package) Bdx() *Entry { return p.firstByExt(".bdx") }

// Tap2 返回 .tap2 条目（本次异动增量）。
func (p *Package) Tap2() *Entry { return p.firstByExt(".tap2") }

// IndexOf 返回条目在原 zip 中的序号。
func (p *Package) IndexOf(name string) int {
	if i, ok := p.index[name]; ok {
		return i
	}
	return -1
}

// Rebuild 描述一次重建：nil 表示该条目保持原字节。
// Replace 按精确条目名覆盖；Tap/Tgl 是便捷字段（按扩展名定位）。
type Rebuild struct {
	Tap     []byte
	Tgl     []byte
	Replace map[string][]byte
}

// EntryAction 是逐条目写回计划（--dry-run 与 apply 报告的数据源）。
type EntryAction struct {
	Name        string `json:"name"`
	Changed     bool   `json:"changed"`
	Reason      string `json:"reason,omitempty"`
	OldSize     int    `json:"old_size"`
	NewSize     int    `json:"new_size"`
	OldSha256   string `json:"old_sha256"`
	NewSha256   string `json:"new_sha256"`
	Method      uint16 `json:"method"`
	Passthrough bool   `json:"passthrough"`
}

// Plan 计算逐条目写回计划，不改动任何字节。
//
// 红线 R2：条目集合不增不减、名字不变、顺序不变（设计器 Packing 只遍历已有条目）。
// 红线 R1：.4gl 永不改写。
func (p *Package) Plan(r Rebuild) ([]EntryAction, error) {
	repl := map[string][]byte{}
	for k, v := range r.Replace {
		repl[k] = v
	}
	if r.Tap != nil {
		e := p.Tap()
		if e == nil {
			return nil, &FormatError{Msg: "包内没有 .tap 条目，无法写回"}
		}
		repl[e.Name] = r.Tap
	}
	if r.Tgl != nil {
		e := p.Tgl()
		if e == nil {
			return nil, &FormatError{Msg: "包内没有 .tgl 条目，无法写回"}
		}
		repl[e.Name] = r.Tgl
	}
	// 红线 R1：拒绝改写 .4gl。
	for name := range repl {
		if path.Ext(name) == ".4gl" {
			return nil, &FormatError{
				Msg:    "拒绝改写 .4gl 条目（红线 R1：服务器 build 产物，渲染结果 ⊉ 它）",
				Detail: []string{name, "如确需，另立 --sync-4gl 开关单独评审"},
			}
		}
	}
	// 未知替换目标必须真实存在（不新增条目）。
	for name := range repl {
		if p.Entry(name) == nil {
			return nil, &FormatError{
				Msg:    "拒绝新增 zip 条目（红线 R2）",
				Detail: []string{"目标条目不在原包内：" + name},
			}
		}
	}

	actions := make([]EntryAction, 0, len(p.Entries))
	for _, e := range p.Entries {
		a := EntryAction{
			Name:      e.Name,
			OldSize:   len(e.Data),
			OldSha256: e.Sha256,
			Method:    e.Method,
		}
		if nd, ok := repl[e.Name]; ok {
			a.Changed = true
			a.NewSize = len(nd)
			a.NewSha256 = sha256hex(nd)
		} else {
			a.NewSize = len(e.Data)
			a.NewSha256 = e.Sha256
			a.Passthrough = true
		}
		actions = append(actions, a)
	}
	return actions, nil
}

// Build 产出新包字节。逐条目遍历原包：被替换的用新内容，其余字节透传。
//
// 时间戳 / 压缩级别 / ACL 都不是功能（红线 R7）：这里沿用每条目原有的
// 压缩方法与修改时间，保证结果确定、可复现，也不引入时钟。
//
// 实现是**字节级重建**（zipraw.go），不是「重新写一个合法 zip」：
// 设计器的包没有数据描述符，局部头里直接带 CRC/大小和自己的 extra 字段；
// 用 archive/zip 的 Writer 重写会变成「bit3 + 数据描述符 + Go 自己的 extra」，
// 真机上设计器会拒绝（Data descriptor signature not found）。
func (p *Package) Build(r Rebuild) ([]byte, []EntryAction, error) {
	actions, err := p.Plan(r)
	if err != nil {
		return nil, nil, err
	}
	repl := map[string][]byte{}
	for k, v := range r.Replace {
		repl[k] = v
	}
	if r.Tap != nil {
		repl[p.Tap().Name] = r.Tap
	}
	if r.Tgl != nil {
		repl[p.Tgl().Name] = r.Tgl
	}

	out, err := p.rebuildZip(repl, actions)
	if err != nil {
		return nil, nil, err
	}
	return out, actions, nil
}
